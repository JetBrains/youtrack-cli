package user

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/JetBrains/youtrack-cli/api"
	"github.com/JetBrains/youtrack-cli/internal"
	"github.com/JetBrains/youtrack-cli/internal/output"
)

var userTypeAPIIDs = map[string]string{
	"standard": "STANDARD_USER",
	"agent":    "AGENT",
	"reporter": "REPORTER",
}

func newUserCreateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "create",
		Aliases: []string{"add"},
		Short:   "Create a new user",
		Example: "  youtrack user create --login jane --fullName \"Jane Doe\" --email jane@x.io --password secret",
		RunE: func(cmd *cobra.Command, args []string) error {
			login, _ := cmd.Flags().GetString("login")
			fullName, _ := cmd.Flags().GetString("fullName")
			email, _ := cmd.Flags().GetString("email")
			password, _ := cmd.Flags().GetString("password")
			typeFlag, _ := cmd.Flags().GetString("type")

			apiID, ok := userTypeAPIIDs[typeFlag]
			if !ok {
				return fmt.Errorf("invalid --type %q: must be one of standard, agent, reporter", typeFlag)
			}

			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			user, err := client.AddUser(&api.AddUserRequest{
				Login:    login,
				FullName: fullName,
				Email:    email,
				Password: password,
				UserType: &api.UserType{ID: apiID},
			})
			if err != nil {
				return err
			}
			return output.Print(cmd, user)
		},
	}
	cmd.Flags().String("login", "", "User login (required)")
	cmd.Flags().String("fullName", "", "User full name (required)")
	cmd.Flags().String("email", "", "User email (required)")
	cmd.Flags().String("password", "", "User password (required)")
	cmd.Flags().String("type", "standard", "User type: standard, agent, or reporter")
	internal.MustMarkRequired(cmd, "login", "fullName", "email", "password")
	return cmd
}
