package auth

import (
	"bufio"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/JetBrains/youtrack-cli/internal/output"
)

// LogoutResult is the JSON output of the auth logout command.
type LogoutResult struct {
	URL    string `json:"url"`
	Status string `json:"status"`
}

func newLogoutCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "logout",
		Short: "Remove saved credentials from ~/.youtrack-cli-env",
		Long: `Removes YOUTRACK_BASE_URL and YOUTRACK_TOKEN from ~/.youtrack-cli-env.

The token remains valid on the YouTrack server until deleted manually
from your profile (Users → your account → Authentication tab).

Use --format json for machine-readable output.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			force, _ := cmd.Flags().GetBool("force")

			home, err := os.UserHomeDir()
			if err != nil {
				return fmt.Errorf("resolving home dir: %w", err)
			}
			envFile := filepath.Join(home, ".youtrack-cli-env")

			existing, readErr := os.ReadFile(envFile)
			if readErr != nil && !errors.Is(readErr, fs.ErrNotExist) {
				return fmt.Errorf("reading env file: %w", readErr)
			}

			content := string(existing)
			url := extractEnvValue(content, "YOUTRACK_BASE_URL")
			hasCredentials := url != "" || extractEnvValue(content, "YOUTRACK_TOKEN") != ""

			if !hasCredentials {
				res := LogoutResult{Status: "not configured"}
				if !flagChanged(cmd, "format") {
					fmt.Fprintln(cmd.OutOrStdout(), "Not logged in — nothing to remove.")
					return nil
				}
				return output.Print(cmd, res)
			}

			if !force {
				label := url
				if label == "" {
					label = "current instance"
				}
				if _, err := fmt.Fprintf(cmd.ErrOrStderr(), "Remove credentials for %s? [y/N]: ", label); err != nil {
					return err
				}
				reader := bufio.NewReader(cmd.InOrStdin())
				answer, err := reader.ReadString('\n')
				if err != nil {
					return fmt.Errorf("confirmation requires interactive input; use --force to skip")
				}
				answer = strings.ToLower(strings.TrimSpace(answer))
				if answer != "y" && answer != "yes" {
					res := LogoutResult{URL: url, Status: "cancelled"}
					if !flagChanged(cmd, "format") {
						fmt.Fprintln(cmd.OutOrStdout(), "Cancelled.")
						return nil
					}
					return output.Print(cmd, res)
				}
			}

			updated := removeEnvLine(content, "YOUTRACK_BASE_URL")
			updated = removeEnvLine(updated, "YOUTRACK_TOKEN")
			if err := os.WriteFile(envFile, []byte(updated), 0600); err != nil {
				return fmt.Errorf("writing env file: %w", err)
			}

			res := LogoutResult{URL: url, Status: "logged out"}
			if !flagChanged(cmd, "format") {
				if url != "" {
					fmt.Fprintf(cmd.OutOrStdout(), "Logged out from %s\n", url)
				} else {
					fmt.Fprintln(cmd.OutOrStdout(), "Logged out.")
				}
				return nil
			}
			return output.Print(cmd, res)
		},
	}
	cmd.Flags().Bool("force", false, "Skip the confirmation prompt")
	return cmd
}

// extractEnvValue returns the value for the given KEY from an env file content string.
// Returns empty string if the key is not present.
func extractEnvValue(content, key string) string {
	prefix := key + "="
	for _, line := range strings.Split(content, "\n") {
		if strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, prefix) {
			return strings.TrimSpace(strings.TrimPrefix(line, prefix))
		}
	}
	return ""
}

// removeEnvLine removes all lines with the given KEY= prefix from an env file content string.
// Empty lines are also stripped. Trailing newline is preserved when content remains.
func removeEnvLine(content, key string) string {
	prefix := key + "="
	var lines []string
	for _, line := range strings.Split(strings.TrimRight(content, "\n"), "\n") {
		if line != "" && !strings.HasPrefix(line, prefix) {
			lines = append(lines, line)
		}
	}
	if len(lines) == 0 {
		return ""
	}
	return strings.Join(lines, "\n") + "\n"
}
