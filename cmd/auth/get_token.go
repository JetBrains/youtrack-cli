package auth

import (
	"fmt"
	"os"
	"path/filepath"
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
		Short: "Get a permanent token via login/password (Hub basic auth)",
		RunE: func(cmd *cobra.Command, args []string) error {
			baseURL, _ := cmd.Root().PersistentFlags().GetString("base-url")
			if baseURL == "" {
				baseURL = os.Getenv("YOUTRACK_BASE_URL")
			}
			if baseURL == "" {
				return fmt.Errorf("YOUTRACK_BASE_URL is not set; use --base-url or env var")
			}
			baseURL = strings.TrimRight(baseURL, "/")

			login, _ := cmd.Flags().GetString("login")
			password, _ := cmd.Flags().GetString("password")
			tokenName, _ := cmd.Flags().GetString("token-name")
			scope, _ := cmd.Flags().GetString("scope")
			save, _ := cmd.Flags().GetBool("save")

			if password == "" {
				fmt.Fprint(cmd.ErrOrStderr(), "Password: ")
				raw, err := term.ReadPassword(int(os.Stdin.Fd()))
				fmt.Fprintln(cmd.ErrOrStderr())
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
				home, err := os.UserHomeDir()
				if err != nil {
					return fmt.Errorf("resolving home dir: %w", err)
				}
				envFile := filepath.Join(home, ".youtrack-cli-env")
				existing, _ := os.ReadFile(envFile) // ignore error — file may not exist yet
				updated := upsertEnvLine(string(existing), "YOUTRACK_BASE_URL", baseURL)
				updated = upsertEnvLine(updated, "YOUTRACK_TOKEN", tok.Token)
				if err := os.WriteFile(envFile, []byte(updated), 0600); err != nil {
					return fmt.Errorf("writing env file: %w", err)
				}
				res.EnvFile = envFile
				res.Hint = "Token saved. The CLI will load it automatically on next run."
			}

			return output.Print(cmd, res)
		},
	}

	cmd.Flags().String("login", "", "YouTrack Hub login (required)")
	cmd.Flags().String("password", "", "Password (prompted interactively if omitted; avoid in scripts — visible in shell history)")
	cmd.Flags().String("token-name", "cli-token", "Name for the permanent token")
	cmd.Flags().String("scope", "", "Comma-separated Hub service names (default: YouTrack + YouTrack Administration)")
	cmd.Flags().Bool("save", false, "Write YOUTRACK_BASE_URL and YOUTRACK_TOKEN to ~/.youtrack-cli-env")
	_ = cmd.MarkFlagRequired("login")
	return cmd
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
	for _, line := range strings.Split(content, "\n") {
		if strings.HasPrefix(line, prefix) {
			lines = append(lines, newLine)
			found = true
		} else {
			lines = append(lines, line)
		}
	}
	if !found {
		lines = append(lines, newLine)
	}
	result := strings.Join(lines, "\n")
	if !strings.HasSuffix(result, "\n") {
		result += "\n"
	}
	return result
}
