package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/JetBrains/youtrack-cli/api"
	"github.com/JetBrains/youtrack-cli/internal"
	"github.com/JetBrains/youtrack-cli/internal/output"
)

func newOrgCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "org",
		Short: "Manage organizations",
	}
	internal.RequireSubcommand(cmd)
	cmd.AddCommand(newOrgListCmd())
	cmd.AddCommand(newOrgCreateCmd())
	cmd.AddCommand(newOrgDeleteCmd())
	return cmd
}

func newOrgListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all organizations",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			orgs, err := client.ListOrganizations()
			if err != nil {
				return err
			}
			return output.Print(cmd, orgs)
		},
	}
}

func newOrgCreateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "create",
		Aliases: []string{"add"},
		Short:   "Create a new organization",
		RunE: func(cmd *cobra.Command, args []string) error {
			name, _ := cmd.Flags().GetString("name")

			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}

			org, err := client.AddOrganization(api.AddOrganizationRequest{Name: name})
			if err != nil {
				return err
			}
			return output.Print(cmd, org)
		},
	}
	cmd.Flags().String("name", "", "Organization name (required)")
	internal.MustMarkRequired(cmd, "name")
	return cmd
}

func newOrgDeleteCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete an organization by ID",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if abort, err := internal.ConfirmOrAbort(cmd, fmt.Sprintf("Delete organization %s?", args[0])); abort {
				return err
			}
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			return client.DeleteOrganization(args[0])
		},
	}
	cmd.Flags().Bool("force", false, "Skip the confirmation prompt")
	return cmd
}
