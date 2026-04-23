package auth

import (
	"fmt"
	"os"

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
		Long:  "Check whether the CLI is authenticated and display the current user's info.",
		RunE: func(cmd *cobra.Command, args []string) error {
			baseURL, _ := cmd.Root().PersistentFlags().GetString("base-url")
			if baseURL == "" {
				baseURL = os.Getenv("YOUTRACK_BASE_URL")
			}

			client, err := internal.GetClient(cmd)
			if err != nil {
				// Not configured — report as status, not a fatal error.
				return output.Print(cmd, AuthStatus{
					URL:    baseURL,
					Status: fmt.Sprintf("not configured: %s", err.Error()),
				})
			}

			user, err := client.GetCurrentUser()
			if err != nil {
				return output.Print(cmd, AuthStatus{
					URL:    client.BaseURL,
					Status: fmt.Sprintf("invalid token: %s", err.Error()),
				})
			}

			return output.Print(cmd, AuthStatus{
				URL:      client.BaseURL,
				Login:    user.Login,
				FullName: user.FullName,
				Status:   "authenticated",
			})
		},
	}
}
