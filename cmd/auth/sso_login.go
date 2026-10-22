package auth

import (
	"bufio"
	"context"
	"fmt"
	"maps"
	"net"
	"net/url"
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

const (
	defaultOAuthClientID = "localhost-auth"
	defaultSSOTokenName  = "youtrack-cli"
)

// ssoOptions are the tunables of the browser sign-in flow, filled from the flags of "auth login".
type ssoOptions struct {
	clientID    string
	extraScopes []string
	port        int
	timeout     time.Duration
	session     bool
	tokenName   string
}

func defaultSSOOptions() ssoOptions {
	return ssoOptions{clientID: defaultOAuthClientID, timeout: browserflow.DefaultTimeout, tokenName: defaultSSOTokenName}
}

type ssoLoginResult struct {
	Login      string `json:"login"`
	Name       string `json:"name"`
	Token      string `json:"token"`
	ExpiresAt  int64  `json:"expiresAt,omitempty"`
	Scope      string `json:"scope,omitempty"`
	ConfigFile string `json:"_configFile,omitempty"`
	Hint       string `json:"_hint,omitempty"`
}

// ssoOnlyFlags take effect only in browser sign-in.
var ssoOnlyFlags = []string{"client-id", "port", "timeout", "session"}

// addSSOFlags registers the browser sign-in flags of "auth login".
func addSSOFlags(cmd *cobra.Command) {
	cmd.Flags().Int("port", 0, "Fixed loopback port for the OAuth callback (default: pick a free port)")
	cmd.Flags().String("client-id", defaultOAuthClientID, "OAuth 2.0 client ID registered in Hub")
	cmd.Flags().Duration("timeout", browserflow.DefaultTimeout, "How long to wait for the browser callback")
	cmd.Flags().Bool("session", false, "Save the ~1-hour OAuth access token instead of creating a permanent token")
}

// ssoOptionsFromFlags reads the flags registered by addSSOFlags.
func ssoOptionsFromFlags(cmd *cobra.Command) ssoOptions {
	opts := defaultSSOOptions()
	opts.clientID, _ = cmd.Flags().GetString("client-id")
	opts.port, _ = cmd.Flags().GetInt("port")
	opts.timeout, _ = cmd.Flags().GetDuration("timeout")
	opts.session, _ = cmd.Flags().GetBool("session")
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

// runSSOFlow runs the full browser OAuth 2.0 + PKCE flow against the Hub serving baseURL and saves a permanent token, or the short-lived access token with --session.
func runSSOFlow(cmd *cobra.Command, reader *bufio.Reader, baseURL string, opts *ssoOptions) error {
	stderr := cmd.ErrOrStderr()
	interactive := term.IsTerminal(int(os.Stdin.Fd()))

	if !interactive {
		return fmt.Errorf("browser login needs an interactive terminal — pass --no-browser to paste a token, or set YOUTRACK_TOKEN\nnote: the OAuth callback goes to this machine's loopback, so the flow cannot work over plain SSH unless you forward the callback port")
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
	// Never hand an unvalidated server-advertised Hub URL to the browser; take the same paste-token fallback as a failed pre-flight.
	if err := api.ValidateHubURL(baseURL, hubURL); err != nil {
		_, _ = fmt.Fprintf(stderr, "Hub URL is not usable for browser sign-in (%v); falling back to the paste-token flow.\n\n", err)
		return runPasteTokenFlow(cmd, reader, baseURL)
	}

	endpoints, err := api.DiscoverOAuthEndpointsContext(cmd.Context(), hubURL)
	if err != nil {
		return err
	}

	verifier, err := api.GeneratePKCEVerifier()
	if err != nil {
		return err
	}
	state, err := browserflow.GenerateState()
	if err != nil {
		return err
	}

	listener, err := browserflow.ListenLoopback(opts.port)
	if err != nil {
		return err
	}
	port := listener.Addr().(*net.TCPAddr).Port
	// Bare path-less loopback URI on the literal 127.0.0.1 we bound (RFC 8252 §8.3), so the browser can't miss us by resolving "localhost" to IPv6 [::1].
	redirectURI := fmt.Sprintf("http://127.0.0.1:%d", port)

	scopes := append([]string{ytServiceID}, opts.extraScopes...)
	requested := scopes
	if !opts.session {
		requested = append(slices.Clone(scopes), api.HubServiceID)
	}
	q := url.Values{}
	q.Set("response_type", "code")
	q.Set("client_id", opts.clientID)
	q.Set("redirect_uri", redirectURI)
	q.Set("scope", strings.Join(requested, " "))
	q.Set("state", state)
	q.Set("code_challenge", api.PKCEChallengeS256(verifier))
	q.Set("code_challenge_method", "S256")
	authorizeURL := endpoints.AuthorizationEndpoint + "?" + q.Encode()

	// Pre-flight: verify the Hub knows this client and accepts the loopback redirect; fall back to paste-token otherwise.
	if err := api.PreflightOAuthClientContext(cmd.Context(), authorizeURL); err != nil {
		_ = listener.Close()
		if contextErr := cmd.Context().Err(); contextErr != nil {
			return contextErr
		}
		_, _ = fmt.Fprintf(stderr, "This Hub has no public CLI OAuth client (%q); falling back to the paste-token flow.\n\n", opts.clientID)
		return runPasteTokenFlow(cmd, reader, baseURL)
	}

	_, _ = fmt.Fprintf(stderr, "Opening your browser to complete sign-in...\nIf it does not open, use this URL:\n  %s\n", hyperlink(authorizeURL))

	res, err := browserflow.Run(cmd.Context(), browserflow.Options{
		Listener: listener,
		State:    state,
		OpenURL:  authorizeURL,
		Timeout:  opts.timeout,
		Warn: func(format string, args ...any) {
			_, _ = fmt.Fprintf(stderr, format+"\n", args...)
		},
	})
	if err != nil {
		return err
	}

	_, _ = fmt.Fprintln(stderr, "Exchanging authorization code...")
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

	result := ssoLoginResult{
		Login: user.Login,
		Name:  user.FullName,
		Token: tok.AccessToken,
		Scope: tok.Scope,
	}
	if opts.session {
		result.ExpiresAt = time.Now().Unix() + tok.ExpiresIn - 60 // 60s clock skew margin
	} else {
		_, _ = fmt.Fprintln(stderr, "Creating permanent token...")
		perm, err := api.CreateHubTokenWithAccessTokenContext(cmd.Context(), hubURL, tok.AccessToken, opts.tokenName, scopes)
		if err != nil {
			return fmt.Errorf("creating permanent token: %w\nre-run with --session to save the ~1-hour access token instead", err)
		}
		result.Token = perm.Token
		result.Scope = ""
	}

	save, _ := cmd.Flags().GetBool("save")
	alias := ""
	if save {
		alias, _ = cmd.Flags().GetString("alias")
		cfg, err := config.Load()
		if err != nil {
			return err
		}
		previous := maps.Clone(cfg.Instances)
		if opts.session {
			alias = cfg.AddInstanceOAuth(alias, baseURL, result.Token, result.ExpiresAt)
		} else {
			alias = cfg.AddInstance(alias, baseURL, result.Token)
		}
		if cfg.CurrentInstance == "" {
			cfg.CurrentInstance = alias
		}
		if err := cfg.Save(); err != nil {
			return fmt.Errorf("saving config: %w", err)
		}
		result.ConfigFile, _ = config.ConfigPath()
		if !opts.session {
			revokeReplacedToken(cmd, hubURL, tok.AccessToken, user.Login, opts.tokenName, previous[alias].Token, result.Token)
		}
	}
	switch {
	case !opts.session && save:
		result.Hint = fmt.Sprintf("Permanent token %q saved as %q. Revoke it in your Hub profile under Authentication.", opts.tokenName, alias)
	case !opts.session:
		result.Hint = fmt.Sprintf("Permanent token %q created. Revoke it in your Hub profile under Authentication.", opts.tokenName)
	case save:
		result.Hint = fmt.Sprintf("Session token saved as %q — expires at %s (~1 hour, no refresh; re-run `youtrack instance login %s` to renew).",
			alias, time.Unix(result.ExpiresAt, 0).Format("15:04"), alias)
	default:
		result.Hint = fmt.Sprintf("Session token expires at %s (~1 hour, no refresh).", time.Unix(result.ExpiresAt, 0).Format("15:04"))
	}

	// Every format masks the token unless --show-token is set, as writeTokenResult does for the permanent-token flows.
	result.Token = revealToken(cmd, result.Token)
	// Like writeTokenResult: JSON (explicit, or the default for non-TTY stdout) goes through output.Print for scripts; table and --plain print the summary, which plain output cannot render as a struct.
	if !jsonOutput(cmd) {
		w := cmd.OutOrStdout()
		_, _ = fmt.Fprintf(w, "Signed in as %s (%s)\n", result.Name, result.Login)
		_, _ = fmt.Fprintf(w, "  Token:  %s\n", result.Token)
		if result.ConfigFile != "" {
			_, _ = fmt.Fprintf(w, "  Saved:  %s\n", result.ConfigFile)
		}
		_, _ = fmt.Fprintf(w, "\n%s\n", result.Hint)
		return nil
	}
	return output.Print(cmd, result)
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
