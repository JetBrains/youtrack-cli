package auth

import (
	"bufio"
	"fmt"
	"net"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/JetBrains/youtrack-cli/api"
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
}

func defaultSSOOptions() ssoOptions {
	return ssoOptions{clientID: defaultOAuthClientID, timeout: browserflow.DefaultTimeout}
}

type ssoLoginResult struct {
	Login      string `json:"login"`
	Name       string `json:"name"`
	Token      string `json:"token"`
	ExpiresAt  int64  `json:"expiresAt"`
	Scope      string `json:"scope,omitempty"`
	ConfigFile string `json:"_configFile,omitempty"`
	Hint       string `json:"_hint,omitempty"`
}

// NewSSOLoginCmd constructs the browser single sign-on command (registered as "instance login"; also option 4 in the get-token menu).
func NewSSOLoginCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "login [url-or-alias]",
		Short: "Sign in through the browser (OAuth 2.0 + PKCE, ~1-hour session token)",
		Long: `Signs in to a YouTrack instance through the browser using OAuth 2.0 with
PKCE against the instance's JetBrains Hub. No password or token paste needed —
the browser session (including SSO) does the authentication.

The resulting access token is short-lived (~1 hour) and is not refreshed;
re-run this command to renew it, or use "youtrack auth login" to create a
permanent token instead.

Pass a URL to sign in to a new instance, or an existing alias to renew its
token. The Hub must have a public loopback OAuth client registered
(hub.jetbrains.com has one); when it does not, the command falls back to the
paste-token flow.`,
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
				raw, err := reader.ReadString('\n')
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
			if scope, _ := cmd.Flags().GetString("scope"); scope != "" {
				for _, s := range strings.Split(scope, ",") {
					if s = strings.TrimSpace(s); s != "" {
						opts.extraScopes = append(opts.extraScopes, s)
					}
				}
			}
			return runSSOFlow(cmd, reader, baseURL, opts)
		},
	}

	cmd.Flags().Bool("save", true, "Save credentials to ~/.youtrack-cli-config.json")
	cmd.Flags().String("alias", "", "Instance alias (derived from URL hostname if omitted; ignored when --save=false)")
	cmd.Flags().String("scope", "", "Comma-separated extra Hub service IDs to request in addition to the YouTrack service")
	cmd.Flags().Int("port", 0, "Fixed loopback port for the OAuth callback (default: pick a free port)")
	cmd.Flags().String("client-id", defaultOAuthClientID, "OAuth 2.0 client ID registered in Hub")
	cmd.Flags().Duration("timeout", browserflow.DefaultTimeout, "How long to wait for the browser callback")
	cmd.Flags().Bool("no-browser", false, "Print the authorization URL instead of opening a browser")
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
		line, err := reader.ReadString('\n')
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

// runSSOFlow runs the full browser OAuth 2.0 + PKCE flow against the Hub serving baseURL and saves the short-lived access token.
func runSSOFlow(cmd *cobra.Command, reader *bufio.Reader, baseURL string, opts ssoOptions) error {
	stderr := cmd.ErrOrStderr()
	interactive := term.IsTerminal(int(os.Stdin.Fd()))

	if !interactive && !opts.noBrowser {
		return fmt.Errorf("browser login needs an interactive terminal — use `youtrack auth login` (paste token) or set YOUTRACK_TOKEN\nnote: the OAuth callback goes to this machine's loopback, so the flow cannot work over plain SSH unless you forward the callback port")
	}

	_, _ = fmt.Fprintf(stderr, "Discovering Hub configuration for %s...\n", baseURL)
	hubURL, ytServiceID := discoverHub(baseURL)
	if ytServiceID == "" {
		return fmt.Errorf("could not discover the YouTrack service ID from %s — this instance may be too old for browser login; use `youtrack auth login` instead", baseURL)
	}

	endpoints, err := api.DiscoverOAuthEndpoints(hubURL)
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
	q := url.Values{}
	q.Set("response_type", "code")
	q.Set("client_id", opts.clientID)
	q.Set("redirect_uri", redirectURI)
	q.Set("scope", strings.Join(scopes, " "))
	q.Set("state", state)
	q.Set("code_challenge", api.PKCEChallengeS256(verifier))
	q.Set("code_challenge_method", "S256")
	authorizeURL := endpoints.AuthorizationEndpoint + "?" + q.Encode()

	// Pre-flight: verify the Hub knows this client and accepts the loopback redirect; fall back to paste-token otherwise.
	if err := api.PreflightOAuthClient(authorizeURL); err != nil {
		_ = listener.Close()
		_, _ = fmt.Fprintf(stderr, "This Hub has no public CLI OAuth client (%q); falling back to the paste-token flow.\n\n", opts.clientID)
		return runDirectTokenFlow(cmd, reader, baseURL)
	}

	if opts.noBrowser {
		_, _ = fmt.Fprintf(stderr, "Open this URL in your browser to sign in:\n  %s\n\nWaiting for the callback on %s ...\n", hyperlink(authorizeURL), redirectURI)
	} else {
		_, _ = fmt.Fprintf(stderr, "Opening your browser to complete sign-in...\nIf it does not open, use this URL:\n  %s\n", hyperlink(authorizeURL))
	}

	res, err := browserflow.Run(cmd.Context(), browserflow.Options{
		Listener: listener,
		State:    state,
		OpenURL:  authorizeURL,
		NoOpen:   opts.noBrowser,
		Timeout:  opts.timeout,
		Warn: func(format string, args ...any) {
			_, _ = fmt.Fprintf(stderr, format+"\n", args...)
		},
	})
	if err != nil {
		return err
	}

	_, _ = fmt.Fprintln(stderr, "Exchanging authorization code...")
	tok, err := api.ExchangeOAuthCode(endpoints.TokenEndpoint, opts.clientID, res.Code, redirectURI, verifier)
	if err != nil {
		return fmt.Errorf("exchanging authorization code: %w", err)
	}
	if tok.ExpiresIn <= 60 {
		return fmt.Errorf("token response contained invalid expires_in %d", tok.ExpiresIn)
	}

	_, _ = fmt.Fprintln(stderr, "Verifying token...")
	user, err := api.NewClient(baseURL, tok.AccessToken).GetCurrentUser()
	if err != nil {
		return fmt.Errorf("token verification failed: %w", err)
	}

	expiresAt := time.Now().Unix() + tok.ExpiresIn - 60 // 60s clock skew margin
	result := ssoLoginResult{
		Login:     user.Login,
		Name:      user.FullName,
		Token:     tok.AccessToken,
		ExpiresAt: expiresAt,
		Scope:     tok.Scope,
	}

	if save, _ := cmd.Flags().GetBool("save"); save {
		alias, _ := cmd.Flags().GetString("alias")
		cfg, err := config.Load()
		if err != nil {
			return err
		}
		finalAlias := cfg.AddInstanceOAuth(alias, baseURL, tok.AccessToken, expiresAt)
		if cfg.CurrentInstance == "" {
			cfg.CurrentInstance = finalAlias
		}
		if err := cfg.Save(); err != nil {
			return fmt.Errorf("saving config: %w", err)
		}
		path, _ := config.ConfigPath()
		result.ConfigFile = path
		result.Hint = fmt.Sprintf(
			"Session token saved as %q — expires at %s (~1 hour, no refresh; re-run `youtrack instance login %s` to renew).\nFor a permanent token, use `youtrack auth login` instead.",
			finalAlias, time.Unix(expiresAt, 0).Format("15:04"), finalAlias)
	} else {
		result.Hint = fmt.Sprintf("Session token expires at %s (~1 hour, no refresh). For a permanent token, use `youtrack auth login`.",
			time.Unix(expiresAt, 0).Format("15:04"))
	}

	// "table" is the human mode; --json/--plain and non-TTY stdout resolve to a structured format that must go through output.Print for scripts.
	if output.Format(cmd) == "table" {
		w := cmd.OutOrStdout()
		_, _ = fmt.Fprintf(w, "Signed in as %s (%s)\n", result.Name, result.Login)
		_, _ = fmt.Fprintf(w, "  Token:  %s\n", maskToken(result.Token))
		if result.ConfigFile != "" {
			_, _ = fmt.Fprintf(w, "  Saved:  %s\n", result.ConfigFile)
		}
		_, _ = fmt.Fprintf(w, "\n%s\n", result.Hint)
		return nil
	}
	return output.Print(cmd, result)
}

// discoverHub resolves the Hub base URL and YouTrack service ID (OAuth scope): /api/config ring(...) primary, MCP oauth-protected-resource fallback.
func discoverHub(baseURL string) (hubURL, ytServiceID string) {
	if ring, err := api.GetRingConfig(baseURL); err == nil {
		hubURL = strings.TrimRight(ring.URL, "/")
		ytServiceID = ring.ServiceID
	}
	if hubURL == "" || ytServiceID == "" {
		if res, err := api.GetOAuthProtectedResource(baseURL); err == nil {
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
