package auth

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/JetBrains/youtrack-cli/api"
	"github.com/JetBrains/youtrack-cli/internal"
	"github.com/JetBrains/youtrack-cli/internal/config"
	"github.com/JetBrains/youtrack-cli/internal/output"
)

const showTokenUsage = "Print the full token instead of the masked form, in every output format"

type getTokenResult struct {
	*api.HubToken
	ConfigFile string `json:"_configFile,omitempty"`
	Hint       string `json:"_hint,omitempty"`
}

// NewGetTokenCmd constructs the interactive authentication command. Registered
// as "auth login" (alias "get-token") and reused by cmd/instance as "instance add";
// both override Use and Short.
func NewGetTokenCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get-token",
		Short: "Authenticate interactively and save a permanent token",
		Long: `Prompts for the YouTrack URL (if not already configured) and lets you choose
an authentication method:

  1) Browser SSO  — OAuth 2.0 + PKCE sign-in (auto-refreshing session;
                    same as "youtrack instance login", which has more options)
  2) Web browser  — open the token page in a browser, then paste the token
  3) Paste token  — enter a token directly without opening a browser
  4) Basic auth   — login + password via Hub API (deprecated: no SSO/2FA)

If basic auth is unavailable (e.g. cloud or SSO-only instances), the CLI
falls back to option 3 automatically.

Use --json for machine-readable output.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			reader := bufio.NewReader(os.Stdin)

			// Resolve base URL — flag → interactive prompt.
			// YOUTRACK_BASE_URL is intentionally ignored here: get-token is used to
			// set up credentials (possibly for a new instance), so the URL should
			// always be confirmed. Use --base-url to skip the prompt in scripts.
			baseURL, _ := cmd.Root().PersistentFlags().GetString("base-url")
			if baseURL == "" {
				if _, err := fmt.Fprint(cmd.ErrOrStderr(), "YouTrack base URL: "); err != nil {
					return err
				}
				raw, err := internal.ReadLineContext(cmd.Context(), reader)
				if err != nil {
					return fmt.Errorf("reading URL: %w", err)
				}
				baseURL = strings.TrimSpace(raw)
			}
			baseURL = strings.TrimRight(baseURL, "/")
			// Add https:// if the user omitted the scheme.
			if baseURL != "" && !strings.HasPrefix(baseURL, "http://") && !strings.HasPrefix(baseURL, "https://") {
				baseURL = "https://" + baseURL
			}
			if baseURL == "" {
				return fmt.Errorf("base URL is required")
			}

			// Resolve alias — flag → interactive prompt with derived default.
			// Only prompt when saving: alias is meaningless without a config entry.
			// Skip prompt when --alias was explicitly provided or --save=false.
			save, _ := cmd.Flags().GetBool("save")
			if save && !flagChanged(cmd, "alias") {
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

			// Auth-type selection.
			_, _ = fmt.Fprintf(cmd.ErrOrStderr(),
				"\nSelect authentication method:\n  1) Browser single sign-on (OAuth, auto-refreshing session)\n  2) Web browser (open token page and paste)\n  3) Paste token directly\n  4) Basic auth (login + password; deprecated)\nSelect [1-4]: ")
			choiceRaw, err := internal.ReadLineContext(cmd.Context(), reader)
			if err != nil {
				return fmt.Errorf("reading auth type: %w", err)
			}
			switch strings.TrimSpace(choiceRaw) {
			case "1":
				opts := defaultSSOOptions()
				opts.insecure, _ = cmd.Flags().GetBool("insecure")
				return runSSOFlow(cmd, reader, baseURL, &opts)
			case "2":
				return runWebFlow(cmd, reader, baseURL)
			case "3":
				return runDirectTokenFlow(cmd, reader, baseURL)
			case "4":
				return runBasicAuthFlow(cmd, reader, baseURL)
			default:
				return fmt.Errorf("invalid choice %q: enter 1, 2, 3, or 4", strings.TrimSpace(choiceRaw))
			}
		},
	}

	cmd.Flags().String("login", "", "Hub login (basic auth; prompted interactively if omitted)")
	cmd.Flags().String("password", "", "Password (basic auth; prompted interactively if omitted — avoid in scripts)")
	cmd.Flags().String("token-name", "cli-token", "Name for the permanent token (basic auth only)")
	cmd.Flags().String("scope", "", "Comma-separated Hub service names (basic auth only; default: YouTrack + YouTrack Administration)")
	cmd.Flags().Bool("insecure", false, "Allow basic auth over plain http to a non-loopback host (sends the password unencrypted)")
	cmd.Flags().Bool("save", true, "Save credentials to ~/.youtrack-cli-config.json")
	cmd.Flags().String("alias", "", "Instance alias (derived from URL hostname if omitted; ignored when --save=false)")
	cmd.Flags().Bool("show-token", false, showTokenUsage)
	return cmd
}

// requireSecureBasicAuth refuses to send a password over plain http to a non-loopback host unless --insecure is set.
func requireSecureBasicAuth(cmd *cobra.Command, baseURL string) error {
	if insecure, _ := cmd.Flags().GetBool("insecure"); insecure || !api.IsInsecureRemoteHTTP(baseURL) {
		return nil
	}
	return fmt.Errorf("refusing to send your password over plain http to %s — use the https URL, choose another sign-in method, or pass --insecure", baseURL)
}

// warnBasicAuthDeprecated steers users off password sign-in, which bypasses SSO and two-factor authentication.
func warnBasicAuthDeprecated(cmd *cobra.Command) {
	output.PrintWarn(cmd, "basic auth is deprecated: it sends your password through the CLI and does not work with SSO or two-factor authentication — "+
		"prefer browser sign-in (youtrack instance login) or a permanent token of a dedicated, least-privilege account for automation")
}

// runBasicAuthFlow handles the Hub login/password → permanent-token creation flow.
func runBasicAuthFlow(cmd *cobra.Command, reader *bufio.Reader, baseURL string) error {
	warnBasicAuthDeprecated(cmd)
	if flagChanged(cmd, "verbose") {
		api.SetHubVerbose(cmd.ErrOrStderr())
		defer api.SetHubVerbose(nil)
	}

	login, _ := cmd.Flags().GetString("login")
	password, _ := cmd.Flags().GetString("password")
	tokenName, _ := cmd.Flags().GetString("token-name")
	scope, _ := cmd.Flags().GetString("scope")
	save, _ := cmd.Flags().GetBool("save")

	if err := requireSecureBasicAuth(cmd, baseURL); err != nil {
		return err
	}

	if login == "" {
		if _, err := fmt.Fprint(cmd.ErrOrStderr(), "Login: "); err != nil {
			return err
		}
		raw, err := internal.ReadLineContext(cmd.Context(), reader)
		if err != nil {
			return fmt.Errorf("reading login: %w", err)
		}
		login = strings.TrimSpace(raw)
	}

	if password == "" {
		if _, err := fmt.Fprint(cmd.ErrOrStderr(), "Password: "); err != nil {
			return err
		}
		raw, err := internal.ReadPasswordContext(cmd.Context(), int(os.Stdin.Fd()))
		if _, printErr := fmt.Fprintln(cmd.ErrOrStderr()); printErr != nil {
			return printErr
		}
		if err != nil {
			return fmt.Errorf("reading password: %w", err)
		}
		password = string(raw)
	}

	me, err := api.GetHubMeContext(cmd.Context(), baseURL, login, password)
	if err != nil {
		if strings.Contains(err.Error(), "basic auth is not available") {
			_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "\nBasic auth is not available for this instance.\nFalling back to direct token input.\n\n")
			return runDirectTokenFlow(cmd, reader, baseURL)
		}
		return err
	}

	services, err := api.ListHubServicesContext(cmd.Context(), baseURL, login, password)
	if err != nil {
		return fmt.Errorf("listing Hub services: %w", err)
	}

	scopeIDs, err := resolveScope(scope, services)
	if err != nil {
		return err
	}

	tok, err := api.CreateHubTokenContext(cmd.Context(), baseURL, login, password, me.ID, tokenName, scopeIDs)
	if err != nil {
		return fmt.Errorf("creating token: %w", err)
	}

	res := getTokenResult{HubToken: tok}
	if save {
		if _, err := saveCredentials(cmd, reader, baseURL, tok.Token, &res); err != nil {
			return err
		}
	}
	return writeTokenResult(cmd, res)
}

// runWebFlow handles the browser-based token creation flow.
// It detects whether Hub is internal or external, opens the appropriate token
// management page, then prompts the user to paste the token they created.
func runWebFlow(cmd *cobra.Command, reader *bufio.Reader, baseURL string) error {
	if flagChanged(cmd, "verbose") {
		api.SetHubVerbose(cmd.ErrOrStderr())
		defer api.SetHubVerbose(nil)
	}

	save, _ := cmd.Flags().GetBool("save")

	_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "Detecting Hub configuration for %s...\n", baseURL)
	hubURL, isInternal, err := api.DetectHubURLContext(cmd.Context(), baseURL)
	if err != nil {
		return fmt.Errorf("detecting Hub: %w", err)
	}

	var tokenPageURL string
	if isInternal {
		tokenPageURL = baseURL + "/users/me?tab=account-security"
	} else {
		tokenPageURL = hubURL + "/users/me?tab=authentication"
	}

	_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "Token page:\n  %s\n\n", hyperlink(tokenPageURL))
	_, _ = fmt.Fprintln(cmd.ErrOrStderr(), "Steps:")
	_, _ = fmt.Fprintln(cmd.ErrOrStderr(), "  1. Log in if prompted")
	_, _ = fmt.Fprintln(cmd.ErrOrStderr(), "  2. Click 'New token' (or 'Generate new token')")
	_, _ = fmt.Fprintln(cmd.ErrOrStderr(), "  3. Copy the token value and return here")

	if term.IsTerminal(int(os.Stdin.Fd())) {
		if err := openBrowser(tokenPageURL); err != nil {
			_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "  (Could not open browser: %v)\n", err)
		}
	}

	if _, err := fmt.Fprint(cmd.ErrOrStderr(), "\nPaste your token: "); err != nil {
		return err
	}
	tokenRaw, err := internal.ReadLineContext(cmd.Context(), reader)
	if err != nil {
		return fmt.Errorf("reading token: %w", err)
	}
	token := strings.TrimSpace(tokenRaw)
	if token == "" {
		return fmt.Errorf("token cannot be empty")
	}

	// Verify before saving — do not persist an invalid token.
	_, _ = fmt.Fprintln(cmd.ErrOrStderr(), "Verifying token...")
	verifyClient := api.NewClient(baseURL, token).WithContext(cmd.Context())
	user, err := verifyClient.GetCurrentUser()
	if err != nil {
		return fmt.Errorf("token verification failed: %w\nRe-run and paste the token again", err)
	}
	_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "Authenticated as %s (%s)\n", user.FullName, user.Login)

	res := getTokenResult{HubToken: &api.HubToken{Token: token}}
	if save {
		if _, err := saveCredentials(cmd, reader, baseURL, token, &res); err != nil {
			return err
		}
	}
	return writeTokenResult(cmd, res)
}

// writeTokenResult prints res in the resolved output format. Every format masks the token unless --show-token is set, because the output usually feeds terminals, scripts, and agents whose logs must not keep the secret.
func writeTokenResult(cmd *cobra.Command, res getTokenResult) error {
	if res.HubToken != nil {
		tok := *res.HubToken
		tok.Token = revealToken(cmd, tok.Token)
		res.HubToken = &tok
	}
	if !jsonOutput(cmd) {
		printTokenResult(cmd, res)
		return nil
	}
	return output.Print(cmd, res)
}

// revealToken returns the token in full only when --show-token is set; otherwise it returns the masked form.
func revealToken(cmd *cobra.Command, token string) string {
	if show, _ := cmd.Flags().GetBool("show-token"); show {
		return token
	}
	return maskToken(token)
}

// printTokenResult writes a human-readable summary of a successful get-token result; res.Token is already masked or revealed by writeTokenResult.
func printTokenResult(cmd *cobra.Command, res getTokenResult) {
	w := cmd.OutOrStdout()
	name := "token"
	if res.HubToken != nil && res.Name != "" {
		name = res.Name
	}
	_, _ = fmt.Fprintf(w, "Token ready: %s\n", name)
	if res.HubToken != nil && res.Token != "" {
		_, _ = fmt.Fprintf(w, "  Token:  %s\n", res.Token)
	}
	if res.ConfigFile != "" {
		_, _ = fmt.Fprintf(w, "  Saved:  %s\n", res.ConfigFile)
	}
	if res.Hint != "" {
		_, _ = fmt.Fprintf(w, "\n%s\n", res.Hint)
	}
}

// maskToken shows a short prefix + **** — a JWT gets an 8-char prefix (masking only the last segment would dump its whole header and payload), perm tokens keep everything up to the secret segment.
func maskToken(token string) string {
	if strings.HasPrefix(token, "eyJ") && strings.Count(token, ".") == 2 {
		prefix := token
		if len(prefix) > 8 {
			prefix = prefix[:8]
		}
		return prefix + "…****"
	}
	if i := strings.LastIndex(token, "."); i >= 0 {
		return token[:i+1] + "****"
	}
	if len(token) > 4 {
		return token[:4] + "****"
	}
	return "****"
}

// runDirectTokenFlow prompts the user to paste a token directly, without opening a browser.
// It detects the correct token-management page URL and prints it for the user to open manually.
func runDirectTokenFlow(cmd *cobra.Command, reader *bufio.Reader, baseURL string) error {
	save, _ := cmd.Flags().GetBool("save")

	// Determine the token page URL using Hub discovery (best-effort; fall back to YouTrack profile).
	tokenPageURL := baseURL + "/users/me?tab=account-security"
	if hubURL, isInternal, err := api.DetectHubURLContext(cmd.Context(), baseURL); err == nil {
		if !isInternal {
			tokenPageURL = hubURL + "/users/me?tab=authentication"
		}
	}
	if err := cmd.Context().Err(); err != nil {
		return err
	}

	_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "Create a permanent token at:\n  %s\n\n", hyperlink(tokenPageURL))

	if _, err := fmt.Fprint(cmd.ErrOrStderr(), "Paste your token: "); err != nil {
		return err
	}
	tokenRaw, err := internal.ReadLineContext(cmd.Context(), reader)
	if err != nil {
		return fmt.Errorf("reading token: %w", err)
	}
	token := strings.TrimSpace(tokenRaw)
	if token == "" {
		return fmt.Errorf("token cannot be empty")
	}

	// Verify before saving — do not persist an invalid token.
	_, _ = fmt.Fprintln(cmd.ErrOrStderr(), "Verifying token...")
	verifyClient := api.NewClient(baseURL, token).WithContext(cmd.Context())
	user, err := verifyClient.GetCurrentUser()
	if err != nil {
		return fmt.Errorf("token verification failed: %w\nRe-run and paste the token again", err)
	}
	_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "Authenticated as %s (%s)\n", user.FullName, user.Login)

	res := getTokenResult{HubToken: &api.HubToken{Token: token}}
	if save {
		if _, err := saveCredentials(cmd, reader, baseURL, token, &res); err != nil {
			return err
		}
	}
	return writeTokenResult(cmd, res)
}

// saveCredentials persists baseURL + token to ~/.youtrack-cli-config.json.
// Returns the alias under which the new instance was saved.
func saveCredentials(cmd *cobra.Command, _ *bufio.Reader, baseURL, token string, res *getTokenResult) (string, error) {
	alias, _ := cmd.Flags().GetString("alias")

	var cfg *config.Config
	var finalAlias string
	err := config.Update(cmd.Context(), func(c *config.Config) error {
		cfg = c
		finalAlias = c.AddInstance(alias, baseURL, token)
		if c.CurrentInstance == "" {
			c.CurrentInstance = finalAlias
		}
		return nil
	})
	if err != nil {
		return "", fmt.Errorf("saving config: %w", err)
	}

	path, _ := config.ConfigPath()
	if !cfg.TokenInKeyring(finalAlias) {
		_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "warning: OS keyring unavailable — token stored in plain text at %s\n", path)
	}
	res.ConfigFile = path
	if cfg.CurrentInstance == finalAlias {
		res.Hint = fmt.Sprintf("Token saved as %q (active). The CLI will load it automatically.", finalAlias)
	} else {
		res.Hint = fmt.Sprintf("Token saved as %q. Run: youtrack instance use %s", finalAlias, finalAlias)
	}
	return finalAlias, nil
}

// resolveScope maps the --scope flag to Hub service IDs.
// Defaults to YouTrack + YouTrack Administration (or first available service).
// When YouTrack Administration is available, it must be included — a token with
// only the YouTrack service cannot perform administrative operations.
func resolveScope(scope string, services []api.HubService) ([]string, error) {
	if scope == "" {
		var ids []string
		for _, svc := range services {
			if svc.Name == "YouTrack" || svc.Name == "YouTrack Administration" {
				ids = append(ids, svc.ID)
			}
		}
		if len(ids) == 0 && len(services) > 0 {
			ids = []string{services[0].ID}
		}
		return ids, nil
	}

	var ids []string
	for _, name := range strings.Split(scope, ",") {
		name = strings.TrimSpace(name)
		found := false
		for _, svc := range services {
			if svc.Name == name || svc.ID == name {
				ids = append(ids, svc.ID)
				found = true
				break
			}
		}
		if !found {
			names := make([]string, len(services))
			for i, s := range services {
				names[i] = s.Name
			}
			return nil, fmt.Errorf("service %q not found; available: %s", name, strings.Join(names, ", "))
		}
	}

	// When YouTrack Administration is available on this instance it must be
	// included, otherwise admin API calls will fail with 403.
	var adminID string
	for _, svc := range services {
		if svc.Name == "YouTrack Administration" {
			adminID = svc.ID
			break
		}
	}
	if adminID != "" {
		adminIncluded := false
		for _, id := range ids {
			if id == adminID {
				adminIncluded = true
				break
			}
		}
		if !adminIncluded {
			return nil, fmt.Errorf("scope must include 'YouTrack Administration' when it is available; add it to --scope or omit --scope to use the default (YouTrack + YouTrack Administration)")
		}
	}

	return ids, nil
}

// hyperlink returns an OSC 8 terminal hyperlink for the given URL.
// Terminals that support OSC 8 render it as a clickable link; others display the raw URL.
func hyperlink(url string) string {
	return fmt.Sprintf("\033]8;;%s\033\\%s\033]8;;\033\\", url, url)
}

// flagChanged reports whether a flag (local or inherited persistent) was explicitly
// set by the user. Using cmd.Flag() is more reliable than cmd.Root().PersistentFlags()
// because Cobra sets Changed on the child command's merged flag set, not the original.
func flagChanged(cmd *cobra.Command, name string) bool {
	f := cmd.Flag(name)
	return f != nil && f.Changed
}

// jsonOutput reports whether the resolved output format is JSON (honors --json, --plain, --format, and non-TTY auto-detection).
func jsonOutput(cmd *cobra.Command) bool {
	return output.Format(cmd) == "json"
}

// openBrowser opens the given URL in the system's default browser.
func openBrowser(targetURL string) error {
	var cmdName string
	var args []string
	switch runtime.GOOS {
	case "darwin":
		cmdName = "open"
		args = []string{targetURL}
	case "windows":
		cmdName = "rundll32"
		args = []string{"url.dll,FileProtocolHandler", targetURL}
	default:
		cmdName = "xdg-open"
		args = []string{targetURL}
	}
	return exec.Command(cmdName, args...).Start()
}
