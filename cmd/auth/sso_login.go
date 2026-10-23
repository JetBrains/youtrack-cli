package auth

import (
	"bufio"
	"cmp"
	"context"
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"maps"
	"net"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/JetBrains/youtrack-cli/api"
	"github.com/JetBrains/youtrack-cli/internal"
	"github.com/JetBrains/youtrack-cli/internal/browserflow"
	"github.com/JetBrains/youtrack-cli/internal/config"
	"github.com/JetBrains/youtrack-cli/internal/output"
)

const defaultSSOTokenName = "youtrack-cli"

// ssoOptions are the tunables of the browser sign-in flow, filled from the flags of "auth login".
type ssoOptions struct {
	clientID    string // empty → api.DefaultOAuthClientID for the resolved Hub
	extraScopes []string
	port        int
	timeout     time.Duration
	session     bool
	tokenName   string
	noBrowser   bool   // print the authorization URL and accept the pasted redirect URL instead of opening a browser
	hubURL      string // explicit Hub URL (standalone Hub); empty → discovered from the instance
	insecure    bool   // allow plain http for loopback (local development) instances
}

func defaultSSOOptions() ssoOptions {
	return ssoOptions{timeout: browserflow.DefaultTimeout, tokenName: defaultSSOTokenName}
}

// ssoLoginResult is the output of a browser sign-in; the token is masked unless --show-token is set, and a session's refresh token is never printed.
type ssoLoginResult struct {
	Login           string `json:"login"`
	Name            string `json:"name"`
	Token           string `json:"token"`
	ExpiresAt       int64  `json:"expiresAt,omitempty"`
	HasRefreshToken bool   `json:"hasRefreshToken,omitempty"`
	Scope           string `json:"scope,omitempty"`
	HubURL          string `json:"hubUrl,omitempty"`
	ClientID        string `json:"clientId,omitempty"`
	Alias           string `json:"alias,omitempty"`
	ConfigFile      string `json:"_configFile,omitempty"`
	TokenStorage    string `json:"_tokenStorage,omitempty"`
	Hint            string `json:"_hint,omitempty"`
}

// ssoOnlyFlags take effect only in browser sign-in.
var ssoOnlyFlags = []string{"client-id", "port", "timeout", "session", "hub-url"}

// addSSOFlags registers the browser sign-in flags of "auth login".
func addSSOFlags(cmd *cobra.Command) {
	cmd.Flags().Int("port", 0, "Fixed loopback port for the OAuth callback (default: pick a free port)")
	cmd.Flags().String("client-id", "", `OAuth 2.0 client ID registered in Hub (default: "localhost-auth" on hub.jetbrains.com, "youtrack-cli" elsewhere)`)
	cmd.Flags().Duration("timeout", browserflow.DefaultTimeout, "How long to wait for the browser callback")
	cmd.Flags().Bool("session", false, "Save an auto-refreshing OAuth session (access + refresh token) instead of creating a permanent token")
	cmd.Flags().String("hub-url", "", "Hub base URL for a standalone Hub (default: discovered from the instance, e.g. <url>/hub)")
}

// ssoOptionsFromFlags reads the flags registered by addSSOFlags.
func ssoOptionsFromFlags(cmd *cobra.Command) ssoOptions {
	opts := defaultSSOOptions()
	opts.clientID, _ = cmd.Flags().GetString("client-id")
	opts.port, _ = cmd.Flags().GetInt("port")
	opts.timeout, _ = cmd.Flags().GetDuration("timeout")
	opts.session, _ = cmd.Flags().GetBool("session")
	opts.hubURL, _ = cmd.Flags().GetString("hub-url")
	opts.noBrowser, _ = cmd.Flags().GetBool("no-browser")
	opts.insecure, _ = cmd.Flags().GetBool("insecure")
	if name, _ := cmd.Flags().GetString("token-name"); name != "" {
		opts.tokenName = name
	}
	if scope, _ := cmd.Flags().GetString("scope"); scope != "" {
		for _, s := range strings.Split(scope, ",") {
			if s = strings.TrimSpace(s); s != "" {
				opts.extraScopes = append(opts.extraScopes, s)
			}
		}
	}
	return opts
}

// requireNoSSOFlags rejects browser sign-in flags (plus extra) on a path that never opens the browser, so a flag like --client-id is not silently ignored.
func requireNoSSOFlags(cmd *cobra.Command, reason string, extra ...string) error {
	return rejectFlags(cmd, reason, append(slices.Clone(ssoOnlyFlags), extra...)...)
}

// rejectFlags fails on the first of names that was set, because on the chosen path it would be silently ignored.
func rejectFlags(cmd *cobra.Command, reason string, names ...string) error {
	for _, name := range names {
		if flagChanged(cmd, name) {
			return fmt.Errorf("--%s has no effect %s", name, reason)
		}
	}
	return nil
}

// resolveSSOBaseURL resolves the target instance URL: positional arg (URL or configured alias) → --base-url flag → interactive prompt.
func resolveSSOBaseURL(cmd *cobra.Command, reader *bufio.Reader, args []string) (string, error) {
	raw := ""
	if len(args) > 0 {
		raw = strings.TrimSpace(args[0])
	}

	// An existing alias renews that instance's token in place.
	if raw != "" {
		if cfg, err := config.Load(); err == nil {
			if inst, ok := cfg.Instances[raw]; ok {
				if err := internal.RequireTokenURL(raw, &inst); err != nil {
					return "", err
				}
				if !flagChanged(cmd, "alias") {
					if err := cmd.Flags().Set("alias", raw); err != nil {
						return "", err
					}
				}
				return strings.TrimRight(inst.URL, "/"), nil
			}
		}
	}

	if raw == "" {
		raw, _ = cmd.Root().PersistentFlags().GetString("base-url")
	}
	if raw == "" {
		url, err := promptBaseURL(cmd, reader)
		if err != nil {
			return "", err
		}
		raw = url
	}
	raw = strings.TrimRight(raw, "/")
	if raw != "" && !strings.HasPrefix(raw, "http://") && !strings.HasPrefix(raw, "https://") {
		raw = "https://" + raw
	}
	if raw == "" {
		return "", fmt.Errorf("base URL is required")
	}
	return raw, nil
}

// runSSOFlow runs the full browser OAuth 2.0 + PKCE flow against the Hub serving baseURL and saves a permanent token, or with --session an auto-refreshing OAuth session.
func runSSOFlow(cmd *cobra.Command, reader *bufio.Reader, baseURL string, opts *ssoOptions) error {
	stderr := cmd.ErrOrStderr()
	interactive := term.IsTerminal(int(os.Stdin.Fd()))

	if !interactive && !opts.noBrowser {
		return fmt.Errorf("browser login needs an interactive terminal — pass --no-browser to paste a token (with --session, the redirect URL), or set YOUTRACK_TOKEN")
	}

	// TLS: the instance and Hub must be https; plain http is only for loopback development instances behind --insecure.
	if err := api.ValidateOAuthURL(baseURL, opts.insecure); err != nil {
		return err
	}
	if opts.insecure && strings.HasPrefix(strings.ToLower(baseURL), "http://") {
		_, _ = fmt.Fprintf(stderr, "warning: --insecure: signing in over plain http to %s — for local development only\n", baseURL)
	}

	_, _ = fmt.Fprintf(stderr, "Discovering Hub configuration for %s...\n", baseURL)
	hubURL, ytServiceID := discoverHub(cmd.Context(), baseURL)
	if err := cmd.Context().Err(); err != nil {
		return err
	}
	if ytServiceID == "" {
		_, _ = fmt.Fprintf(stderr, "Could not discover the YouTrack service ID from %s (this instance may be too old for browser sign-in); falling back to the paste-token flow.\n\n", baseURL)
		return runPasteTokenFlow(cmd, reader, baseURL)
	}
	if opts.hubURL != "" {
		hubURL = strings.TrimRight(opts.hubURL, "/")
		if err := api.ValidateOAuthURL(hubURL, opts.insecure); err != nil {
			return fmt.Errorf("--hub-url: %w", err)
		}
	} else if err := api.ValidateHubURL(baseURL, hubURL); err != nil {
		// Never hand an unvalidated server-advertised Hub URL to the browser; take the same paste-token fallback as a failed pre-flight.
		_, _ = fmt.Fprintf(stderr, "Hub URL is not usable for browser sign-in (%v); falling back to the paste-token flow.\n\n", err)
		return runPasteTokenFlow(cmd, reader, baseURL)
	}
	if err := api.ValidateOAuthURL(hubURL, opts.insecure); err != nil {
		return fmt.Errorf("hub: %w (pass --hub-url to point at the https Hub)", err)
	}

	endpoints, err := api.DiscoverOAuthEndpointsContext(cmd.Context(), hubURL)
	if err != nil {
		return err
	}
	if opts.clientID == "" {
		opts.clientID = api.DefaultOAuthClientID(hubURL)
	}

	// PKCE is mandatory: a fresh 256-bit verifier per attempt, S256 challenge only (plain is never offered).
	verifier, err := api.GeneratePKCEVerifier()
	if err != nil {
		return err
	}
	state, err := browserflow.GenerateState()
	if err != nil {
		return err
	}

	listener, err := browserflow.ListenLoopback(opts.port)
	var port int
	switch {
	case err == nil:
		port = listener.Addr().(*net.TCPAddr).Port
	case opts.port != 0:
		// A fixed port in use may belong to another program that would receive the redirect; stop instead of guessing.
		return err
	default:
		// No loopback networking at all: continue paste-only on a random high port that nothing listens on.
		_, _ = fmt.Fprintf(stderr, "Could not listen for the browser callback (%v); you will paste the redirect URL instead.\n", err)
		port, err = randomHighPort()
		if err != nil {
			return err
		}
		listener = nil
	}
	closeListener := func() {
		if listener != nil {
			_ = listener.Close()
		}
	}
	// Bare path-less loopback URI on the literal 127.0.0.1 we bound (RFC 8252 §8.3), so the browser can't miss us by resolving "localhost" to IPv6 [::1].
	redirectURI := fmt.Sprintf("http://127.0.0.1:%d", port)

	scopes := append([]string{ytServiceID}, opts.extraScopes...)
	requested := scopes
	if !opts.session && !slices.Contains(scopes, api.HubServiceID) {
		requested = append(slices.Clone(scopes), api.HubServiceID)
	}
	authorizeURL, err := api.BuildAuthorizationURL(endpoints.AuthorizationEndpoint, &api.AuthorizationRequest{
		ClientID:      opts.clientID,
		RedirectURI:   redirectURI,
		Scopes:        requested,
		State:         state,
		CodeChallenge: api.PKCEChallengeS256(verifier),
		Offline:       opts.session,
	})
	if err != nil {
		closeListener()
		return err
	}

	// Pre-flight: verify the Hub knows this client and accepts the loopback redirect; fall back to paste-token otherwise.
	if err := api.PreflightOAuthClientContext(cmd.Context(), authorizeURL); err != nil {
		closeListener()
		if contextErr := cmd.Context().Err(); contextErr != nil {
			return contextErr
		}
		_, _ = fmt.Fprintf(stderr, "Hub did not accept OAuth client %q with a loopback redirect (%v).\n"+
			"Ask a YouTrack administrator to register a client under Administration > Access Management > OAuth Clients "+
			"(Authentication tab: Client ID youtrack-cli, Authorization Code flow + Require PKCE, Redirect URI http://127.0.0.1, no Client secret), "+
			"or pass the Client ID of an existing client with --client-id; "+
			"falling back to the paste-token flow.\n\n",
			opts.clientID, err)
		return runPasteTokenFlow(cmd, reader, baseURL)
	}

	link := authorizeURL
	if term.IsTerminal(int(os.Stderr.Fd())) && os.Getenv("NO_COLOR") == "" {
		link = hyperlink(authorizeURL)
	}
	pastePrompt := fmt.Sprintf("If the browser is on another device, it will end on a page that fails to load at %s/?code=...\nPaste that full URL here and press Enter:", redirectURI)
	if opts.noBrowser || listener == nil {
		_, _ = fmt.Fprintf(stderr, "Open this URL in a browser on any device to sign in:\n  %s\n\n", link)
	} else {
		_, _ = fmt.Fprintf(stderr, "Opening your browser to complete sign-in...\nIf it does not open, use this URL:\n  %s\n", link)
	}

	res, err := browserflow.Run(cmd.Context(), &browserflow.Options{
		Listener:    listener,
		State:       state,
		OpenURL:     authorizeURL,
		NoOpen:      opts.noBrowser,
		Timeout:     opts.timeout,
		Paste:       reader,
		RedirectURI: redirectURI,
		PastePrompt: pastePrompt,
		Warn: func(format string, args ...any) {
			_, _ = fmt.Fprintf(stderr, format+"\n", args...)
		},
	})
	if err != nil {
		return err
	}

	_, _ = fmt.Fprintln(stderr, "Exchanging authorization code...")
	requestedAt := time.Now()
	tok, err := api.ExchangeOAuthCodeContext(cmd.Context(), endpoints.TokenEndpoint, opts.clientID, res.Code, redirectURI, verifier)
	if err != nil {
		return fmt.Errorf("exchanging authorization code: %w", err)
	}
	if tok.ExpiresIn <= 60 {
		return fmt.Errorf("token response contained invalid expires_in %d", tok.ExpiresIn)
	}

	_, _ = fmt.Fprintln(stderr, "Verifying token...")
	user, err := api.NewClient(baseURL, tok.AccessToken).WithContext(cmd.Context()).GetCurrentUser()
	if err != nil {
		return fmt.Errorf("token verification failed: %w", err)
	}

	// Check the config is loadable before creating a permanent token, so a broken config fails before anything exists in Hub.
	save, _ := cmd.Flags().GetBool("save")
	if save {
		if _, err := config.Load(); err != nil {
			return err
		}
	}

	result := ssoLoginResult{
		Login: user.Login,
		Name:  user.FullName,
		Token: tok.AccessToken,
		Scope: tok.Scope,
	}
	var sess config.OAuthSession
	newTokenID := ""
	if opts.session {
		scope := cmp.Or(tok.Scope, strings.Join(scopes, " "))
		sess = config.OAuthSession{
			AccessToken:   tok.AccessToken,
			RefreshToken:  tok.RefreshToken,
			ExpiresAt:     api.TokenExpiry(requestedAt, tok.ExpiresIn),
			TokenType:     cmp.Or(tok.TokenType, "Bearer"),
			Scope:         scope,
			HubURL:        hubURL,
			ClientID:      opts.clientID,
			TokenEndpoint: endpoints.TokenEndpoint,
		}
		result.ExpiresAt = sess.ExpiresAt
		result.HasRefreshToken = sess.RefreshToken != ""
		result.Scope = scope
		result.HubURL = hubURL
		result.ClientID = opts.clientID
	} else {
		_, _ = fmt.Fprintln(stderr, "Creating permanent token...")
		perm, err := api.CreateHubTokenWithAccessTokenContext(cmd.Context(), hubURL, tok.AccessToken, opts.tokenName, scopes)
		if err != nil {
			return fmt.Errorf("creating permanent token: %w\nre-run with --session to save an auto-refreshing session instead", err)
		}
		result.Token = perm.Token
		result.Scope = ""
		newTokenID = perm.ID
	}

	alias := ""
	if save {
		alias, _ = cmd.Flags().GetString("alias")
		var previous map[string]config.Instance
		err := config.Update(cmd.Context(), func(cfg *config.Config) error {
			previous = maps.Clone(cfg.Instances)
			if opts.session {
				alias = cfg.AddInstanceOAuth(alias, baseURL, &sess)
			} else {
				alias = cfg.AddInstance(alias, baseURL, result.Token)
			}
			if cfg.CurrentInstance == "" {
				cfg.CurrentInstance = alias
			}
			return nil
		})
		if err != nil {
			return revokeUnsavedToken(cmd, hubURL, tok.AccessToken, newTokenID, opts.tokenName, fmt.Errorf("saving config: %w", err))
		}
		result.Alias = alias
		result.ConfigFile, _ = config.ConfigPath()
		result.TokenStorage = "file " + result.ConfigFile + " (0600)"
		if cfg, err := config.Load(); err == nil {
			// The keyring writes can fail independently; claim keyring storage only when every credential made it.
			if cfg.TokenInKeyring(alias) && (!opts.session || sess.RefreshToken == "" || cfg.RefreshTokenInKeyring(alias)) {
				result.TokenStorage = "OS keyring (service youtrack-cli)"
			} else {
				_, _ = fmt.Fprintf(stderr, "warning: OS keyring unavailable — tokens stored in %s (mode 0600)\n", result.ConfigFile)
			}
		}
		if opts.session {
			warnReplacedPermanentToken(cmd, user.Login, alias, previous[alias].Token)
		} else {
			revokeReplacedToken(cmd, hubURL, tok.AccessToken, user.Login, opts.tokenName, previous[alias].Token, result.Token)
		}
	}
	renewal := "the session refreshes automatically"
	if opts.session && !result.HasRefreshToken {
		renewal = "Hub issued no refresh token, so re-run `youtrack auth login --session` to renew"
	}
	switch {
	case !opts.session && save:
		result.Hint = fmt.Sprintf("Permanent token %q saved as %q. Revoke it in your Hub profile under Authentication.", opts.tokenName, alias)
	case !opts.session:
		result.Hint = fmt.Sprintf("Permanent token %q created. Revoke it in your Hub profile under Authentication.", opts.tokenName)
	case save:
		result.Hint = fmt.Sprintf("Session saved as %q — access token expires at %s; %s.",
			alias, time.Unix(result.ExpiresAt, 0).Format("15:04"), renewal)
	default:
		result.Hint = fmt.Sprintf("Verified only (--save=false): nothing was stored. The access token would expire at %s.", time.Unix(result.ExpiresAt, 0).Format("15:04"))
	}

	// Every format masks the token unless --show-token is set, as writeTokenResult does for the permanent-token flows.
	result.Token = revealToken(cmd, result.Token)
	// Like writeTokenResult: JSON (explicit, or the default for non-TTY stdout) goes through output.Print for scripts; table and --plain print the summary, which plain output cannot render as a struct.
	if !jsonOutput(cmd) {
		w := cmd.OutOrStdout()
		_, _ = fmt.Fprintf(w, "Signed in as %s (%s)\n", output.Sanitize(result.Name), output.Sanitize(result.Login))
		_, _ = fmt.Fprintf(w, "  Token:  %s\n", result.Token)
		if result.ConfigFile != "" {
			_, _ = fmt.Fprintf(w, "  Saved:  %s\n", result.ConfigFile)
		}
		_, _ = fmt.Fprintf(w, "\n%s\n", result.Hint)
		return nil
	}
	return output.Print(cmd, result)
}

// randomHighPort picks a port in the dynamic range for a paste-only redirect URI when no loopback listener could be bound.
func randomHighPort() (int, error) {
	var b [2]byte
	if _, err := rand.Read(b[:]); err != nil {
		return 0, err
	}
	return 49152 + int(binary.BigEndian.Uint16(b[:]))%16384, nil
}

// revokeUnsavedToken deletes a permanent token that was created but could not be saved, so a failed save does not leave a working token nobody can see; it returns saveErr, noting when the token is still live.
func revokeUnsavedToken(cmd *cobra.Command, hubURL, accessToken, tokenID, tokenName string, saveErr error) error {
	if tokenID == "" {
		return saveErr
	}
	if err := api.DeleteHubTokenWithAccessTokenContext(cmd.Context(), hubURL, accessToken, tokenID); err != nil {
		return fmt.Errorf("%w\nthe permanent token %q created for this sign-in could not be revoked (%w); revoke it in your Hub profile under Authentication", saveErr, tokenName, err)
	}
	return saveErr
}

// warnReplacedPermanentToken reports a permanent token of the signed-in user that a --session sign-in just replaced; the session token lacks Hub's scope, so the CLI cannot revoke it.
func warnReplacedPermanentToken(cmd *cobra.Command, login, alias, oldToken string) {
	if owner, _, ok := api.ParsePermanentToken(oldToken); ok && owner == login {
		output.PrintWarn(cmd, fmt.Sprintf("the permanent token previously saved for %q is still valid in Hub; revoke it in your Hub profile under Authentication if nothing else uses it", alias))
	}
}

// revokeReplacedToken deletes the permanent token a re-login just replaced, so signing in again does not leave a working token behind in Hub. Only tokens of the signed-in user that carry the CLI's token name are deleted; any other token is kept and reported.
func revokeReplacedToken(cmd *cobra.Command, hubURL, accessToken, login, tokenName, oldToken, newToken string) {
	owner, oldID, ok := api.ParsePermanentToken(oldToken)
	if !ok || owner != login || oldToken == newToken {
		return
	}
	old, err := api.GetHubTokenWithAccessTokenContext(cmd.Context(), hubURL, accessToken, oldID)
	switch {
	case err != nil:
		output.PrintWarn(cmd, fmt.Sprintf("could not look up the replaced permanent token (%v); it may still be valid, check your Hub profile under Authentication", err))
	case old.Name != tokenName:
		output.PrintWarn(cmd, fmt.Sprintf("the replaced permanent token %q is still valid; revoke it in your Hub profile under Authentication if nothing else uses it", old.Name))
	default:
		if err := api.DeleteHubTokenWithAccessTokenContext(cmd.Context(), hubURL, accessToken, oldID); err != nil {
			output.PrintWarn(cmd, fmt.Sprintf("could not revoke the replaced permanent token %q (%v); revoke it in your Hub profile under Authentication", old.Name, err))
		}
	}
}

// discoverHub resolves the Hub base URL and YouTrack service ID (OAuth scope): /api/config ring(...) primary, MCP oauth-protected-resource fallback.
func discoverHub(ctx context.Context, baseURL string) (hubURL, ytServiceID string) {
	if ring, err := api.GetRingConfigContext(ctx, baseURL); err == nil {
		hubURL = strings.TrimRight(api.ResolveHubURL(baseURL, ring.URL), "/")
		ytServiceID = ring.ServiceID
	}
	if hubURL == "" || ytServiceID == "" {
		if res, err := api.GetOAuthProtectedResourceContext(ctx, baseURL); err == nil {
			if hubURL == "" && len(res.AuthorizationServers) > 0 {
				hubURL = strings.TrimRight(res.AuthorizationServers[0], "/")
			}
			if ytServiceID == "" && len(res.ScopesSupported) > 0 {
				ytServiceID = res.ScopesSupported[0]
			}
		}
	}
	if hubURL == "" {
		hubURL = baseURL + "/hub" // embedded Hub
	}
	return hubURL, ytServiceID
}
