package auth

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/JetBrains/youtrack-cli/api"
	"github.com/JetBrains/youtrack-cli/internal/output"
)

type getTokenResult struct {
	*api.HubToken
	EnvFile string `json:"_envFile,omitempty"`
	Hint    string `json:"_hint,omitempty"`
}

func newGetTokenCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get-token",
		Short: "Authenticate interactively and save a permanent token",
		Long: `Prompts for the YouTrack URL (if not already configured) and lets you choose
between basic auth (login/password via Hub) or a browser-based flow where you
create a token in the YouTrack UI and paste it here.

Note: --format table is not supported for this command and falls back to JSON.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			reader := bufio.NewReader(os.Stdin)

			// Resolve base URL — flag → env → interactive prompt.
			baseURL, _ := cmd.Root().PersistentFlags().GetString("base-url")
			if baseURL == "" {
				baseURL = os.Getenv("YOUTRACK_BASE_URL")
			}
			if baseURL == "" {
				if _, err := fmt.Fprint(cmd.ErrOrStderr(), "YouTrack base URL: "); err != nil {
					return err
				}
				raw, err := reader.ReadString('\n')
				if err != nil {
					return fmt.Errorf("reading URL: %w", err)
				}
				baseURL = strings.TrimSpace(raw)
			}
			baseURL = strings.TrimRight(baseURL, "/")
			if baseURL == "" {
				return fmt.Errorf("base URL is required")
			}

			// Auth-type selection.
			fmt.Fprintf(cmd.ErrOrStderr(),
				"\nAuth type:\n  1) Basic auth (login + password)\n  2) Web browser (open YouTrack and paste token)\nEnter 1 or 2: ")
			choiceRaw, err := reader.ReadString('\n')
			if err != nil {
				return fmt.Errorf("reading auth type: %w", err)
			}
			switch strings.TrimSpace(choiceRaw) {
			case "1":
				return runBasicAuthFlow(cmd, reader, baseURL)
			case "2":
				return runWebFlow(cmd, reader, baseURL)
			default:
				return fmt.Errorf("invalid choice %q: enter 1 or 2", strings.TrimSpace(choiceRaw))
			}
		},
	}

	cmd.Flags().String("login", "", "Hub login (basic auth; prompted interactively if omitted)")
	cmd.Flags().String("password", "", "Password (basic auth; prompted interactively if omitted — avoid in scripts)")
	cmd.Flags().String("token-name", "cli-token", "Name for the permanent token (basic auth only)")
	cmd.Flags().String("scope", "", "Comma-separated Hub service names (basic auth only; default: YouTrack + YouTrack Administration)")
	cmd.Flags().Bool("save", true, "Write YOUTRACK_BASE_URL and YOUTRACK_TOKEN to ~/.youtrack-cli-env")
	return cmd
}

// runBasicAuthFlow handles the Hub login/password → permanent-token creation flow.
func runBasicAuthFlow(cmd *cobra.Command, reader *bufio.Reader, baseURL string) error {
	login, _ := cmd.Flags().GetString("login")
	password, _ := cmd.Flags().GetString("password")
	tokenName, _ := cmd.Flags().GetString("token-name")
	scope, _ := cmd.Flags().GetString("scope")
	save, _ := cmd.Flags().GetBool("save")

	if login == "" {
		if _, err := fmt.Fprint(cmd.ErrOrStderr(), "Login: "); err != nil {
			return err
		}
		raw, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("reading login: %w", err)
		}
		login = strings.TrimSpace(raw)
	}

	if password == "" {
		if _, err := fmt.Fprint(cmd.ErrOrStderr(), "Password: "); err != nil {
			return err
		}
		raw, err := term.ReadPassword(int(os.Stdin.Fd()))
		if _, printErr := fmt.Fprintln(cmd.ErrOrStderr()); printErr != nil {
			return printErr
		}
		if err != nil {
			return fmt.Errorf("reading password: %w", err)
		}
		password = string(raw)
	}

	me, err := api.GetHubMe(baseURL, login, password)
	if err != nil {
		return fmt.Errorf("authentication failed: %w", err)
	}

	services, err := api.ListHubServices(baseURL, login, password)
	if err != nil {
		return fmt.Errorf("listing Hub services: %w", err)
	}

	scopeIDs, err := resolveScope(scope, services)
	if err != nil {
		return err
	}

	tok, err := api.CreateHubToken(baseURL, login, password, me.ID, tokenName, scopeIDs)
	if err != nil {
		return fmt.Errorf("creating token: %w", err)
	}

	res := getTokenResult{HubToken: tok}
	if save {
		if err := saveToEnvFile(baseURL, tok.Token, &res); err != nil {
			return err
		}
	}
	return output.Print(cmd, res)
}

// runWebFlow handles the browser-based token creation flow.
// It detects whether Hub is internal or external, opens the appropriate token
// management page, then prompts the user to paste the token they created.
func runWebFlow(cmd *cobra.Command, reader *bufio.Reader, baseURL string) error {
	save, _ := cmd.Flags().GetBool("save")

	fmt.Fprintf(cmd.ErrOrStderr(), "Detecting Hub configuration for %s...\n", baseURL)
	hubURL, isInternal, err := api.DetectHubURL(baseURL)
	if err != nil {
		return fmt.Errorf("detecting Hub: %w", err)
	}

	var tokenPageURL string
	if isInternal {
		tokenPageURL = baseURL + "/users/me?tab=account-security"
	} else {
		tokenPageURL = hubURL + "/users/me?tab=authentication"
	}

	fmt.Fprintf(cmd.ErrOrStderr(), "Opening token page in your browser:\n  %s\n\n", tokenPageURL)
	fmt.Fprintln(cmd.ErrOrStderr(), "Steps:")
	fmt.Fprintln(cmd.ErrOrStderr(), "  1. Log in if prompted")
	fmt.Fprintln(cmd.ErrOrStderr(), "  2. Click 'New token' (or 'Generate new token')")
	fmt.Fprintln(cmd.ErrOrStderr(), "  3. Copy the token value and return here")

	if err := openBrowser(tokenPageURL); err != nil {
		fmt.Fprintf(cmd.ErrOrStderr(), "  (Could not open browser automatically: %v)\n", err)
		fmt.Fprintf(cmd.ErrOrStderr(), "  Open manually: %s\n", tokenPageURL)
	}

	if _, err := fmt.Fprint(cmd.ErrOrStderr(), "\nPaste your token: "); err != nil {
		return err
	}
	tokenRaw, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("reading token: %w", err)
	}
	token := strings.TrimSpace(tokenRaw)
	if token == "" {
		return fmt.Errorf("token cannot be empty")
	}

	// Verify before saving — do not persist an invalid token.
	fmt.Fprintln(cmd.ErrOrStderr(), "Verifying token...")
	verifyClient := api.NewClient(baseURL, token)
	user, err := verifyClient.GetCurrentUser()
	if err != nil {
		return fmt.Errorf("token verification failed: %w\nRe-run and paste the token again", err)
	}
	fmt.Fprintf(cmd.ErrOrStderr(), "Authenticated as %s (%s)\n", user.FullName, user.Login)

	res := getTokenResult{HubToken: &api.HubToken{Token: token}}
	if save {
		if err := saveToEnvFile(baseURL, token, &res); err != nil {
			return err
		}
	}
	return output.Print(cmd, res)
}

// saveToEnvFile writes YOUTRACK_BASE_URL and YOUTRACK_TOKEN to ~/.youtrack-cli-env
// and updates res with the file path and a hint message.
func saveToEnvFile(baseURL, token string, res *getTokenResult) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("resolving home dir: %w", err)
	}
	envFile := filepath.Join(home, ".youtrack-cli-env")
	existing, _ := os.ReadFile(envFile) // ignore error — file may not exist yet
	updated := upsertEnvLine(string(existing), "YOUTRACK_BASE_URL", baseURL)
	updated = upsertEnvLine(updated, "YOUTRACK_TOKEN", token)
	if err := os.WriteFile(envFile, []byte(updated), 0600); err != nil {
		return fmt.Errorf("writing env file: %w", err)
	}
	res.EnvFile = envFile
	res.Hint = "Token saved. The CLI will load it automatically on next run."
	return nil
}

// resolveScope maps the --scope flag to Hub service IDs.
// Defaults to YouTrack + YouTrack Administration (or first available service).
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
	return ids, nil
}

// upsertEnvLine sets KEY=value in an env file's content string,
// replacing an existing line for that key or appending one if absent.
func upsertEnvLine(content, key, value string) string {
	prefix := key + "="
	newLine := key + "=" + value
	var lines []string
	found := false
	if content != "" {
		for _, line := range strings.Split(strings.TrimRight(content, "\n"), "\n") {
			if strings.HasPrefix(line, prefix) {
				lines = append(lines, newLine)
				found = true
			} else {
				lines = append(lines, line)
			}
		}
	}
	if !found {
		lines = append(lines, newLine)
	}
	return strings.Join(lines, "\n") + "\n"
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
