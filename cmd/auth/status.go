package auth

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/JetBrains/youtrack-cli/internal"
	"github.com/JetBrains/youtrack-cli/internal/output"
)

// AuthStatus is the JSON output of the auth status command.
type AuthStatus struct {
	URL      string `json:"url"`
	Login    string `json:"login"`
	FullName string `json:"fullName"`
	Status   string `json:"status"`
}

func newStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show current authentication status",
		Long: `Check whether the CLI is authenticated and display the current user's info.

This command always exits 0; it is an informational probe, not a gate.
Check the "status" field for the authentication result:
  "authenticated"       — token is valid
  "invalid token: ..."  — token rejected by the server
  "not configured: ..." — YOUTRACK_BASE_URL or YOUTRACK_TOKEN is not set

Use --format json for machine-readable output.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			baseURL, _ := cmd.Root().PersistentFlags().GetString("base-url")
			if baseURL == "" {
				baseURL = os.Getenv("YOUTRACK_BASE_URL")
			}

			client, err := internal.GetClient(cmd)
			if err != nil {
				s := AuthStatus{
					URL:    baseURL,
					Status: fmt.Sprintf("not configured: %s", err.Error()),
				}
				if !cmd.Root().PersistentFlags().Changed("format") {
					printAuthStatus(cmd, s)
					return nil
				}
				return output.Print(cmd, s)
			}

			user, err := client.GetCurrentUser()
			if err != nil {
				s := AuthStatus{
					URL:    client.BaseURL,
					Status: fmt.Sprintf("invalid token: %s", err.Error()),
				}
				if !cmd.Root().PersistentFlags().Changed("format") {
					printAuthStatus(cmd, s)
					return nil
				}
				return output.Print(cmd, s)
			}

			s := AuthStatus{
				URL:      client.BaseURL,
				Login:    user.Login,
				FullName: user.FullName,
				Status:   "authenticated",
			}
			if !cmd.Root().PersistentFlags().Changed("format") {
				printAuthStatus(cmd, s)
				return nil
			}
			return output.Print(cmd, s)
		},
	}
}

func printAuthStatus(cmd *cobra.Command, s AuthStatus) {
	w := cmd.OutOrStdout()
	switch {
	case s.Status == "authenticated":
		fmt.Fprintf(w, "Authenticated\n")
		fmt.Fprintf(w, "  URL:   %s\n", s.URL)
		fmt.Fprintf(w, "  User:  %s (%s)\n", s.FullName, s.Login)
	case strings.HasPrefix(s.Status, "invalid token"):
		fmt.Fprintf(w, "Token invalid\n")
		fmt.Fprintf(w, "  URL:   %s\n", s.URL)
		fmt.Fprintf(w, "  Error: %s\n", strings.TrimPrefix(s.Status, "invalid token: "))
	default:
		fmt.Fprintf(w, "Not configured\n")
		if s.URL != "" {
			fmt.Fprintf(w, "  URL:   %s\n", s.URL)
		}
		fmt.Fprintf(w, "  Error: %s\n", strings.TrimPrefix(s.Status, "not configured: "))
	}
}
