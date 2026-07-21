package issue

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/JetBrains/youtrack-cli/internal"
	"github.com/JetBrains/youtrack-cli/internal/output"
)

func newIssueLinkCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "link", Short: "Manage issue links"}
	internal.RequireSubcommand(cmd)
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
		Use:   "list <issueId>",
		Short: "List links on an issue",
		Long: `List all links on an issue. The ID column contains the opaque link ID
needed by 'issue link remove'. To extract it programmatically:

  youtrack issue link list <issueId> --format json | jq '.[].id'`,
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
	cmd := &cobra.Command{
		Use:   "remove <issueId> <linkId>",
		Short: "Remove a link from an issue",
		Long: `Remove a link from an issue. <linkId> is the opaque ID shown in the ID
column of 'issue link list' (e.g. 100-1). To find it:

  youtrack issue link list <issueId>`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if abort, err := internal.ConfirmOrAbort(cmd, fmt.Sprintf(
				"Remove link %s from issue %s?", args[1], args[0])); abort {
				return err
			}
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			return client.RemoveIssueLink(args[0], args[1])
		},
	}
	cmd.Flags().Bool("force", false, "Skip the confirmation prompt")
	return cmd
}

func newIssueLinkTypeCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "type", Short: "Manage issue link types"}
	internal.RequireSubcommand(cmd)
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
	create := &cobra.Command{
		Use: "create", Aliases: []string{"add"}, Short: "Create an issue link type",
		RunE: func(cmd *cobra.Command, args []string) error {
			name, _ := cmd.Flags().GetString("name")
			s2t, _ := cmd.Flags().GetString("source-to-target")
			t2s, _ := cmd.Flags().GetString("target-to-source")
			directed, _ := cmd.Flags().GetBool("directed")
			aggregation, _ := cmd.Flags().GetBool("aggregation")
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			lt, err := client.AddIssueLinkType(name, s2t, t2s, directed, aggregation)
			if err != nil {
				return err
			}
			return output.Print(cmd, lt)
		},
	}
	create.Flags().String("name", "", "Link type name (required)")
	create.Flags().String("source-to-target", "", "Source-to-target label (required)")
	create.Flags().String("target-to-source", "", "Target-to-source label (required)")
	create.Flags().Bool("directed", true, "Create a directed link type")
	create.Flags().Bool("aggregation", false, "Create an aggregation link type")
	internal.MustMarkRequired(create, "name", "source-to-target", "target-to-source")
	cmd.AddCommand(list, create)
	return cmd
}
