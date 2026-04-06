package issue

import (
	"github.com/spf13/cobra"

	"github.com/JetBrains/youtrack-cli/internal"
	"github.com/JetBrains/youtrack-cli/internal/output"
)

func newIssueLinkCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "link", Short: "Manage issue links"}
	cmd.AddCommand(
		newIssueLinkListCmd(),
		newIssueLinkAddCmd(),
		newIssueLinkRemoveCmd(),
		newIssueLinkTypeCmd(),
	)
	return cmd
}

func newIssueLinkListCmd() *cobra.Command {
	return &cobra.Command{
		Use: "list <issueId>", Short: "List links on an issue",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			links, err := client.GetLinkedIssues(args[0])
			if err != nil {
				return err
			}
			return output.Print(cmd, links)
		},
	}
}

func newIssueLinkAddCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use: "add <issueId>", Short: "Link an issue to another",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			target, _ := cmd.Flags().GetString("target")
			linkTypeID, _ := cmd.Flags().GetString("type")
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			return client.LinkIssues(args[0], target, linkTypeID)
		},
	}
	cmd.Flags().String("target", "", "Target issue ID (required)")
	cmd.Flags().String("type", "", "Link type ID (required)")
	internal.MustMarkRequired(cmd, "target", "type")
	return cmd
}

func newIssueLinkRemoveCmd() *cobra.Command {
	return &cobra.Command{
		Use: "remove <issueId> <linkId>", Short: "Remove a link from an issue",
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			return client.RemoveIssueLink(args[0], args[1])
		},
	}
}

func newIssueLinkTypeCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "type", Short: "Manage issue link types"}
	list := &cobra.Command{
		Use: "list", Short: "List issue link types",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			types, err := client.GetIssueLinkTypes()
			if err != nil {
				return err
			}
			return output.Print(cmd, types)
		},
	}
	add := &cobra.Command{
		Use: "add", Short: "Create an issue link type",
		RunE: func(cmd *cobra.Command, args []string) error {
			name, _ := cmd.Flags().GetString("name")
			s2t, _ := cmd.Flags().GetString("source-to-target")
			t2s, _ := cmd.Flags().GetString("target-to-source")
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			lt, err := client.AddIssueLinkType(name, s2t, t2s)
			if err != nil {
				return err
			}
			return output.Print(cmd, lt)
		},
	}
	add.Flags().String("name", "", "Link type name (required)")
	add.Flags().String("source-to-target", "", "Source-to-target label (required)")
	add.Flags().String("target-to-source", "", "Target-to-source label (required)")
	internal.MustMarkRequired(add, "name", "source-to-target", "target-to-source")
	cmd.AddCommand(list, add)
	return cmd
}
