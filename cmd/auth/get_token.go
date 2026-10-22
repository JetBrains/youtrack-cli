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

const defaultBaseURL = "https://youtrack.jetbrains.com"

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
		Long: `Prompts for the YouTrack URL, pre-filled with the active instance's URL
(or ` + defaultBaseURL + `), and signs in through the browser: the CLI
creates a permanent token through Hub and saves it.

When the instance's Hub has no OAuth client for the CLI, or with --no-browser,
the CLI prints the token page link and asks you to paste a permanent token.
With --login and --password it creates the token through Hub basic auth instead
(deprecated: the password passes through the CLI and SSO/2FA accounts cannot use it).

Use --json for machine-readable output.`,
		Example: `  # Sign in through the browser
  youtrack auth login

  # Paste a token instead of signing in through the browser
  youtrack auth login --base-url https://example.youtrack.cloud --no-browser`,
		RunE: func(cmd *cobra.Command, args []string) error {
			reader := bufio.NewReader(os.Stdin)

			// YOUTRACK_BASE_URL is intentionally ignored: login sets up credentials, possibly for a new instance, so the URL is always confirmed.
			baseURL, _ := cmd.Root().PersistentFlags().GetString("base-url")
			if baseURL == "" {
				url, err := promptBaseURL(cmd, reader)
				if err != nil {
					return err
				}
				baseURL = url
			}
			baseURL = strings.TrimRight(baseURL, "/")
			if baseURL != "" && !strings.HasPrefix(baseURL, "http://") && !strings.HasPrefix(baseURL, "https://") {
				baseURL = "https://" + baseURL
			}
			if baseURL == "" {
				return fmt.Errorf("base URL is required")
			}

			noBrowser, _ := cmd.Flags().GetBool("no-browser")
			switch {
			case flagChanged(cmd, "login") || flagChanged(cmd, "password"):
				return runBasicAuthFlow(cmd, reader, baseURL)
			case noBrowser || !term.IsTerminal(int(os.Stdin.Fd())):
				return runPasteTokenFlow(cmd, reader, baseURL)
			default:
				opts := defaultSSOOptions()
				return runSSOFlow(cmd, reader, baseURL, &opts)
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
	cmd.Flags().Bool("no-browser", false, "Paste a permanent token instead of signing in through the browser")
	return cmd
}

// requireSecureBasicAuth refuses to send a password over plain http to a non-loopback host unless --insecure is set.
func requireSecureBasicAuth(cmd *cobra.Command, baseURL string) error {
	if insecure, _ := cmd.Flags().GetBool("insecure"); insecure || !api.IsInsecureRemoteHTTP(baseURL) {
		return nil
	}
	return fmt.Errorf("refusing to send your password over plain http to %s — use the https URL, choose another sign-in method, or pass --insecure", baseURL)
}

// runBasicAuthFlow handles the Hub login/password → permanent-token creation flow.
func runBasicAuthFlow(cmd *cobra.Command, reader *bufio.Reader, baseURL string) error {
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
			return runPasteTokenFlow(cmd, reader, baseURL)
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

// promptBaseURL asks for the instance URL, pre-filled with the active instance's URL or defaultBaseURL: editable in a terminal, accepted with Enter on piped stdin.
func promptBaseURL(cmd *cobra.Command, reader *bufio.Reader) (string, error) {
	suggested := defaultBaseURL
	if cfg, err := config.Load(); err == nil {
		if inst, ok := cfg.Instances[cfg.CurrentInstance]; ok && inst.URL != "" {
			suggested = inst.URL
		}
	}

	var raw string
	var err error
	if term.IsTerminal(int(os.Stdin.Fd())) {
		raw, err = internal.ReadLineWithDefault(cmd.Context(), "YouTrack URL: ", suggested)
	} else {
		if _, err := fmt.Fprintf(cmd.ErrOrStderr(), "YouTrack URL [%s]: ", suggested); err != nil {
			return "", err
		}
		raw, err = internal.ReadLineContext(cmd.Context(), reader)
	}
	if err != nil {
		return "", fmt.Errorf("reading URL: %w", err)
	}
	if raw = strings.TrimSpace(raw); raw == "" {
		return suggested, nil
	}
	return raw, nil
}

// runPasteTokenFlow points the user at the token page (offering to open it in a terminal), then reads and verifies a pasted permanent token.
func runPasteTokenFlow(cmd *cobra.Command, reader *bufio.Reader, baseURL string) error {
	stderr := cmd.ErrOrStderr()
	save, _ := cmd.Flags().GetBool("save")

	// Best-effort Hub discovery; an unusable Hub falls back to the instance's own profile page, never to the advertised URL.
	tokenPageURL := baseURL + "/users/me?tab=account-security"
	if hubURL, isInternal, err := api.DetectHubURLContext(cmd.Context(), baseURL); err == nil && !isInternal {
		tokenPageURL = hubURL + "/users/me?tab=authentication"
	}
	if err := cmd.Context().Err(); err != nil {
		return err
	}

	_, _ = fmt.Fprintf(stderr, "Create a permanent token at:\n  %s\n\n", hyperlink(tokenPageURL))
	if noBrowser, _ := cmd.Flags().GetBool("no-browser"); !noBrowser && term.IsTerminal(int(os.Stdin.Fd())) {
		if _, err := fmt.Fprint(stderr, "Open it in your browser? [Y/n]: "); err != nil {
			return err
		}
		answer, err := internal.ReadLineContext(cmd.Context(), reader)
		if err != nil {
			return fmt.Errorf("reading answer: %w", err)
		}
		if a := strings.ToLower(strings.TrimSpace(answer)); a == "" || a == "y" || a == "yes" {
			if err := openBrowser(tokenPageURL); err != nil {
				_, _ = fmt.Fprintf(stderr, "Could not open the browser: %v\n", err)
			}
		}
	}

	if _, err := fmt.Fprint(stderr, "Paste your token: "); err != nil {
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
	_, _ = fmt.Fprintln(stderr, "Verifying token...")
	user, err := api.NewClient(baseURL, token).WithContext(cmd.Context()).GetCurrentUser()
	if err != nil {
		return fmt.Errorf("token verification failed: %w\nRe-run and paste the token again", err)
	}
	_, _ = fmt.Fprintf(stderr, "Authenticated as %s (%s)\n", user.FullName, user.Login)

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

	cfg, err := config.Load()
	if err != nil {
		return "", err
	}

	finalAlias := cfg.AddInstance(alias, baseURL, token)
	if cfg.CurrentInstance == "" {
		cfg.CurrentInstance = finalAlias
	}
	if err := cfg.Save(); err != nil {
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
