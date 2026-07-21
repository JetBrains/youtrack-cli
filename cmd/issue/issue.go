package issue

import (
	"github.com/spf13/cobra"

	"github.com/JetBrains/youtrack-cli/internal"
)

func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "issue",
		Short: "Manage issues",
	}
	internal.RequireSubcommand(cmd)

	cmd.AddGroup(
		&cobra.Group{ID: "work", Title: "Work with issues:"},
		&cobra.Group{ID: "apply", Title: "Apply changes:"},
		&cobra.Group{ID: "content", Title: "Discussion & content:"},
		&cobra.Group{ID: "track", Title: "Track & engage:"},
		&cobra.Group{ID: "ai", Title: "AI:"},
	)

	internal.AddToGroup(cmd, "work",
		newIssueViewCmd(), newIssueListCmd(), newIssueCreateCmd(),
		newIssueUpdateCmd(), newIssueDeleteCmd(),
		newIssueAssignCmd(), newIssueUnassignCmd(),
	)
	internal.AddToGroup(cmd, "apply",
		newIssueCommandCmd(), newIssueFieldCmd(), newIssueVisibilityCmd(),
		newIssueTagCmd(), newIssueLinkCmd(),
	)
	internal.AddToGroup(cmd, "content",
		newIssueCommentCmd(), newIssueAttachmentCmd(), newIssueWorkitemCmd(),
		newIssueDraftCmd(), newIssueReactionCmd(),
	)
	internal.AddToGroup(cmd, "track",
		newIssueVoteCmd(), newIssueUnvoteCmd(), newIssueWatchCmd(), newIssueUnwatchCmd(),
		newIssueActivityCmd(), newIssueCountCmd(), newIssueSprintCmd(), newIssueVCSCmd(),
	)
	internal.AddToGroup(cmd, "ai",
		newIssueSuggestCmd(), newIssueAssistCmd(),
	)

	return cmd
}
