package auth

import (
	"bufio"
	"cmp"
	"context"
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"net"
	"os"
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

const defaultOAuthClientID = "localhost-auth"

// ssoOptions are the tunables of the browser SSO flow: plain defaults from the get-token menu, flag-filled by "instance login".
type ssoOptions struct {
	clientID    string
	extraScopes []string
	port        int
	timeout     time.Duration
	noBrowser   bool
	hubURL      string // explicit Hub URL (standalone Hub); empty → discovered from the instance
	insecure    bool   // allow plain http for loopback (local development) instances
}

func defaultSSOOptions() ssoOptions {
	return ssoOptions{clientID: defaultOAuthClientID, timeout: browserflow.DefaultTimeout}
}

// ssoLoginResult is the output of a browser sign-in; it never contains the tokens themselves.
type ssoLoginResult struct {
	Login           string `json:"login"`
	Name            string `json:"name"`
	TokenPreview    string `json:"tokenPreview"`
	ExpiresAt       int64  `json:"expiresAt"`
	HasRefreshToken bool   `json:"hasRefreshToken"`
	Scope           string `json:"scope,omitempty"`
	HubURL          string `json:"hubUrl"`
	ClientID        string `json:"clientId"`
	Alias           string `json:"alias,omitempty"`
	ConfigFile      string `json:"_configFile,omitempty"`
	TokenStorage    string `json:"_tokenStorage,omitempty"`
	Hint            string `json:"_hint,omitempty"`
}

// NewSSOLoginCmd constructs the browser single sign-on command (registered as "instance login"; also option 4 in the get-token menu).
func NewSSOLoginCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "login [url-or-alias]",
		Short: "Sign in through the browser (OAuth 2.0 + PKCE, auto-refreshing session)",
		Long: `Signs in to a YouTrack instance through the browser using the OAuth 2.0
authorization-code flow with PKCE (S256) against the instance's Hub. No password
or token paste needed — the browser session (including SSO) does the
authentication.

The CLI listens on 127.0.0.1 for the redirect, verifies the state, exchanges the
code, and saves a short-lived access token plus a refresh token (in the OS
keyring when available). Every command refreshes the access token
automatically shortly before it expires; if the session is revoked, sign in
again with this command.

Headless / SSH / WSL: pass --no-browser (or run where no browser is detected),
open the printed URL on any device, and paste the URL the browser ends up on
(http://127.0.0.1:<port>/?code=...&state=...) back into the terminal.

Pass a URL to sign in to a new instance, or an existing alias to renew its
session. The Hub needs a public OAuth client (no secret) with Authorization
Code + PKCE and a loopback redirect URI (http://127.0.0.1); hub.jetbrains.com
provides one as "localhost-auth". Use --client-id for your own client and
--hub-url for a standalone Hub. When the client is missing, the command falls
back to the paste-token flow.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			reader := bufio.NewReader(os.Stdin)

			baseURL, err := resolveSSOBaseURL(cmd, reader, args)
			if err != nil {
				return err
			}

			// Prompt for the alias like get-token does, only when saving and not preset.
			save, _ := cmd.Flags().GetBool("save")
			if save && !flagChanged(cmd, "alias") && term.IsTerminal(int(os.Stdin.Fd())) {
				derived := config.DeriveAlias(baseURL)
				if _, err := fmt.Fprintf(cmd.ErrOrStderr(), "Instance alias [%s]: ", derived); err != nil {
					return err
				}
				raw, err := internal.ReadLineContext(cmd.Context(), reader)
				if err != nil {
					return fmt.Errorf("reading alias: %w", err)
				}
				alias := strings.TrimSpace(raw)
				if alias == "" {
					alias = derived
				}
				if err := cmd.Flags().Set("alias", alias); err != nil {
					return err
				}
			}

			opts := defaultSSOOptions()
			opts.clientID, _ = cmd.Flags().GetString("client-id")
			opts.port, _ = cmd.Flags().GetInt("port")
			opts.timeout, _ = cmd.Flags().GetDuration("timeout")
			opts.noBrowser, _ = cmd.Flags().GetBool("no-browser")
			opts.hubURL, _ = cmd.Flags().GetString("hub-url")
			opts.insecure, _ = cmd.Flags().GetBool("insecure")
			if scope, _ := cmd.Flags().GetString("scope"); scope != "" {
				for _, s := range strings.Split(scope, ",") {
					if s = strings.TrimSpace(s); s != "" {
						opts.extraScopes = append(opts.extraScopes, s)
					}
				}
			}
			return runSSOFlow(cmd, reader, baseURL, &opts)
		},
	}

	cmd.Flags().Bool("save", true, "Save credentials to ~/.youtrack-cli-config.json")
	cmd.Flags().String("alias", "", "Instance alias (derived from URL hostname if omitted; ignored when --save=false)")
	cmd.Flags().String("scope", "", "Comma-separated extra Hub service IDs to request in addition to the YouTrack service")
	cmd.Flags().Int("port", 0, "Fixed loopback port for the OAuth callback (default: pick a free port)")
	cmd.Flags().String("client-id", defaultOAuthClientID, "OAuth 2.0 client ID registered in Hub")
	cmd.Flags().Duration("timeout", browserflow.DefaultTimeout, "How long to wait for the browser callback")
	cmd.Flags().Bool("no-browser", false, "Print the authorization URL and accept the pasted redirect URL instead of opening a browser (SSH, WSL, headless)")
	cmd.Flags().String("hub-url", "", "Hub base URL for a standalone Hub (default: discovered from the instance, e.g. <url>/hub)")
	cmd.Flags().Bool("insecure", false, "Allow plain http for a local (localhost/127.0.0.1) instance or Hub; remote http is always refused")
	return cmd
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
		if _, err := fmt.Fprint(cmd.ErrOrStderr(), "YouTrack base URL: "); err != nil {
			return "", err
		}
		line, err := internal.ReadLineContext(cmd.Context(), reader)
		if err != nil {
			return "", fmt.Errorf("reading URL: %w", err)
		}
		raw = strings.TrimSpace(line)
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

// runSSOFlow runs the full browser OAuth 2.0 + PKCE flow against the Hub serving baseURL and saves the resulting session.
func runSSOFlow(cmd *cobra.Command, reader *bufio.Reader, baseURL string, opts *ssoOptions) error {
	stderr := cmd.ErrOrStderr()
	interactive := term.IsTerminal(int(os.Stdin.Fd()))

	if !interactive && !opts.noBrowser {
		return fmt.Errorf("browser login needs an interactive terminal — pass --no-browser to paste the redirect URL instead, use `youtrack auth login` (paste token), or set YOUTRACK_TOKEN")
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
		return fmt.Errorf("could not discover the YouTrack service ID from %s — check the URL (it must be the YouTrack base URL); older instances may not support browser login, use `youtrack auth login` instead", baseURL)
	}
	if opts.hubURL != "" {
		hubURL = strings.TrimRight(opts.hubURL, "/")
		if err := api.ValidateOAuthURL(hubURL, opts.insecure); err != nil {
			return fmt.Errorf("--hub-url: %w", err)
		}
	} else if err := api.ValidateHubURL(baseURL, hubURL); err != nil {
		// Never hand an unvalidated server-advertised Hub URL to the browser; take the same paste-token fallback as a failed pre-flight.
		_, _ = fmt.Fprintf(stderr, "Hub URL is not usable for browser sign-in (%v); falling back to the paste-token flow.\n\n", err)
		return runDirectTokenFlow(cmd, reader, baseURL)
	}
	if err := api.ValidateOAuthURL(hubURL, opts.insecure); err != nil {
		return fmt.Errorf("hub: %w (pass --hub-url to point at the https Hub)", err)
	}

	endpoints, err := api.DiscoverOAuthEndpointsContext(cmd.Context(), hubURL)
	if err != nil {
		return err
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
	authorizeURL, err := api.BuildAuthorizationURL(endpoints.AuthorizationEndpoint, &api.AuthorizationRequest{
		ClientID:      opts.clientID,
		RedirectURI:   redirectURI,
		Scopes:        scopes,
		State:         state,
		CodeChallenge: api.PKCEChallengeS256(verifier),
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
			"Register a public client in YouTrack (Authorization Code + Require PKCE, no secret, redirect URI http://127.0.0.1) and pass --client-id; falling back to the paste-token flow.\n\n",
			opts.clientID, err)
		return runDirectTokenFlow(cmd, reader, baseURL)
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

	scope := tok.Scope
	if scope == "" {
		scope = strings.Join(scopes, " ")
	}
	sess := config.OAuthSession{
		AccessToken:        tok.AccessToken,
		RefreshToken:       tok.RefreshToken,
		ExpiresAt:          api.TokenExpiry(requestedAt, tok.ExpiresIn),
		TokenType:          cmp.Or(tok.TokenType, "Bearer"),
		Scope:              scope,
		HubURL:             hubURL,
		ClientID:           opts.clientID,
		TokenEndpoint:      endpoints.TokenEndpoint,
		RevocationEndpoint: endpoints.RevocationEndpoint,
	}
	result := ssoLoginResult{
		Login:           user.Login,
		Name:            user.FullName,
		TokenPreview:    maskToken(tok.AccessToken),
		ExpiresAt:       sess.ExpiresAt,
		HasRefreshToken: sess.RefreshToken != "",
		Scope:           scope,
		HubURL:          hubURL,
		ClientID:        opts.clientID,
	}
	renewal := "the session refreshes automatically"
	if !result.HasRefreshToken {
		renewal = "Hub issued no refresh token, so re-run `youtrack instance login` to renew"
	}

	if save, _ := cmd.Flags().GetBool("save"); save {
		alias, _ := cmd.Flags().GetString("alias")
		var finalAlias string
		var inKeyring bool
		err := config.Update(cmd.Context(), func(cfg *config.Config) error {
			finalAlias = cfg.AddInstanceOAuth(alias, baseURL, &sess)
			if cfg.CurrentInstance == "" {
				cfg.CurrentInstance = finalAlias
			}
			return nil
		})
		if err != nil {
			return fmt.Errorf("saving config: %w", err)
		}
		if cfg, err := config.Load(); err == nil {
			inKeyring = cfg.TokenInKeyring(finalAlias)
		}
		if !result.HasRefreshToken {
			config.PurgeKeyringRefreshToken(finalAlias) // drop a refresh token left over from an earlier sign-in
		}
		path, _ := config.ConfigPath()
		result.Alias = finalAlias
		result.ConfigFile = path
		result.TokenStorage = "file " + path + " (0600)"
		if inKeyring {
			result.TokenStorage = "OS keyring (service youtrack-cli)"
		} else {
			_, _ = fmt.Fprintf(stderr, "warning: OS keyring unavailable — tokens stored in %s (mode 0600)\n", path)
		}
		result.Hint = fmt.Sprintf("Signed in as %q — access token expires at %s; %s.",
			finalAlias, time.Unix(sess.ExpiresAt, 0).Format("15:04"), renewal)
	} else {
		result.Hint = fmt.Sprintf("Verified only (--save=false): nothing was stored. The access token would expire at %s.",
			time.Unix(sess.ExpiresAt, 0).Format("15:04"))
	}

	// Like the other auth commands, JSON (explicit or non-TTY stdout) goes through output.Print; table and plain get the human summary.
	if !jsonOutput(cmd) {
		w := cmd.OutOrStdout()
		_, _ = fmt.Fprintf(w, "Signed in as %s (%s)\n", output.Sanitize(result.Name), output.Sanitize(result.Login))
		_, _ = fmt.Fprintf(w, "  Token:   %s\n", result.TokenPreview)
		if result.TokenStorage != "" {
			_, _ = fmt.Fprintf(w, "  Stored:  %s\n", result.TokenStorage)
			_, _ = fmt.Fprintf(w, "  Config:  %s\n", result.ConfigFile)
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
