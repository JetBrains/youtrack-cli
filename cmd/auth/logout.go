package auth

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/JetBrains/youtrack-cli/internal"
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
					_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Not logged in — nothing to remove.")
					return nil
				}
				return output.Print(cmd, res)
			}

			label := url
			if label == "" {
				label = "current instance"
			}
			confirmed, err := internal.Confirm(cmd, fmt.Sprintf("Remove credentials for %s?", label))
			if err != nil {
				return err
			}
			if !confirmed {
				res := LogoutResult{URL: url, Status: "canceled"}
				if !flagChanged(cmd, "format") {
					_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Canceled.")
					return nil
				}
				return output.Print(cmd, res)
			}

			updated := removeEnvLine(content, "YOUTRACK_BASE_URL")
			updated = removeEnvLine(updated, "YOUTRACK_TOKEN")
			if err := os.WriteFile(envFile, []byte(updated), 0600); err != nil {
				return fmt.Errorf("writing env file: %w", err)
			}

			res := LogoutResult{URL: url, Status: "logged out"}
			if !flagChanged(cmd, "format") {
				if url != "" {
					_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Logged out from %s\n", url)
				} else {
					_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Logged out.")
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
