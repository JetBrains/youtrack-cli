package cmd

import (
	"github.com/JetBrains/youtrack-cli/api"
	"github.com/JetBrains/youtrack-cli/cmd/output"
	"github.com/spf13/cobra"
)

func newOrgCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "org",
		Short: "Manage organizations",
	}
	cmd.AddCommand(newOrgListCmd())
	cmd.AddCommand(newOrgAddCmd())
	cmd.AddCommand(newOrgDeleteCmd())
	return cmd
}

func newOrgListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all organizations",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getClient(cmd)
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

func newOrgAddCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add",
		Short: "Create a new organization",
		RunE: func(cmd *cobra.Command, args []string) error {
			name, _ := cmd.Flags().GetString("name")

			client, err := getClient(cmd)
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
	cmd.MarkFlagRequired("name")
	return cmd
}

func newOrgDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete an organization by ID",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getClient(cmd)
			if err != nil {
				return err
			}
			return client.DeleteOrganization(args[0])
		},
	}
}
