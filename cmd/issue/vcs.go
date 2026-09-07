package issue

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/JetBrains/youtrack-cli/api"
	"github.com/JetBrains/youtrack-cli/internal"
	"github.com/JetBrains/youtrack-cli/internal/output"
)

// shortHashLength mirrors the server-side threshold: at 12 characters or more,
// YouTrack treats a hash as a prefix of, or prefixed by, a stored version.
const shortHashLength = 12

// findVCSChange locates a change already linked to the issue by its database ID
// (e.g. "127-27") or by its commit hash. It returns nil when nothing matches,
// and an error when an abbreviated hash matches more than one change.
//
// Stored hashes are not reliably lowercase, so hashes compare case-insensitively.
// An exact hit on an ID or a full hash wins over an abbreviation, which keeps the
// result stable when a short hash happens to prefix an unrelated commit.
func findVCSChange(changes []api.VCSChange, ref string) (*api.VCSChange, error) {
	for i := range changes {
		if changes[i].ID == ref || strings.EqualFold(changes[i].Version, ref) {
			return &changes[i], nil
		}
	}

	var matches []*api.VCSChange
	for i := range changes {
		if changes[i].Version == "" || !hashesOverlap(changes[i].Version, ref) {
			continue
		}
		matches = append(matches, &changes[i])
	}

	switch len(matches) {
	case 0:
		return nil, nil
	case 1:
		return matches[0], nil
	default:
		versions := make([]string, 0, len(matches))
		for _, m := range matches {
			versions = append(versions, m.Version)
		}
		return nil, fmt.Errorf("%q matches several VCS changes (%s); use a longer hash or the change ID",
			ref, strings.Join(versions, ", "))
	}
}

// hashesOverlap reports whether two commit hashes identify the same commit, one
// of them possibly abbreviated. Below shortHashLength an abbreviation is too
// weak to trust, so only equal hashes count — and those the caller matched already.
func hashesOverlap(stored, ref string) bool {
	if len(stored) < shortHashLength || len(ref) < shortHashLength {
		return false
	}
	stored, ref = strings.ToLower(stored), strings.ToLower(ref)
	return strings.HasPrefix(stored, ref) || strings.HasPrefix(ref, stored)
}

func newIssueVCSCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "vcs", Short: "Manage VCS changes linked to an issue"}
	internal.RequireSubcommand(cmd)
	cmd.AddCommand(
		newIssueVCSListCmd(),
		newIssueVCSViewCmd(),
		newIssueVCSAttachCmd(),
		newIssueVCSDetachCmd(),
	)
	return cmd
}

func newIssueVCSListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list <issueId>",
		Short: "List VCS changes and pull requests linked to an issue",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			changes, err := client.ListIssueVCSChanges(args[0])
			if err != nil {
				return err
			}
			return output.Print(cmd, changes)
		},
	}
}

func newIssueVCSViewCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "view <issueId> <changeId>",
		Aliases: []string{"get"},
		Short:   "View a VCS change linked to an issue",
		Args:    cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			change, err := client.GetIssueVCSChange(args[0], args[1])
			if err != nil {
				return err
			}
			return output.Print(cmd, change)
		},
	}
}

func newIssueVCSAttachCmd() *cobra.Command {
	var message string
	var mute bool

	cmd := &cobra.Command{
		Use:   "attach <issueId> <commitHash>",
		Short: "Attach a commit to an issue",
		Long: "Attach a commit to an issue.\n\n" +
			"YouTrack resolves the hash against the VCS integrations configured for the\n" +
			"issue's project, so the commit must be one an integration can see. A hash of\n" +
			"12 characters or more may be abbreviated. When the issue already holds a\n" +
			"detached change for the commit, the change is re-attached instead.",
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			issueID, ref := args[0], args[1]
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}

			changes, err := client.ListIssueVCSChanges(issueID)
			if err != nil {
				return err
			}

			// A change the issue already holds in the detached state is
			// re-attached; only an unknown commit needs a fresh link.
			existing, err := findVCSChange(changes, ref)
			if err != nil {
				return err
			}
			if existing != nil {
				if existing.State == nil || *existing.State != api.VCSChangeStateDetached {
					return fmt.Errorf("%s is already attached to %s", ref, issueID)
				}
				if err := client.SetIssueVCSChangeState(issueID, existing.ID, api.VCSChangeStateDefault, mute); err != nil {
					return err
				}
				output.PrintSuccess(cmd, "re-attached VCS change "+existing.ID+" to "+issueID)
				return nil
			}

			change, err := client.AttachIssueVCSChange(issueID, ref, message, mute)
			if err != nil {
				return err
			}
			return output.Print(cmd, change)
		},
	}

	cmd.Flags().StringVar(&message, "message", "", "commit message; used only by a Generic VCS integration")
	addMuteFlag(cmd, &mute)
	return cmd
}

func newIssueVCSDetachCmd() *cobra.Command {
	var mute bool

	cmd := &cobra.Command{
		Use:     "detach <issueId> <changeIdOrHash>",
		Aliases: []string{"delete", "remove"},
		Short:   "Detach a VCS change from an issue",
		Long: "Detach a VCS change from an issue.\n\n" +
			"YouTrack offers two mechanisms and accepts only one of them per change, so\n" +
			"the command picks it: a commit you attached by hand is deleted outright,\n" +
			"while one a VCS integration attached is detached instead — deleting those is\n" +
			"refused by the server. YouTrack keeps a detached record only where it needs\n" +
			"one to stop the integration re-attaching the commit, which is when the commit\n" +
			"message names the issue; `attach` restores a record that survived. Accepts\n" +
			"either the change ID or the commit hash.",
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			issueID, ref := args[0], args[1]
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}

			changes, err := client.ListIssueVCSChanges(issueID)
			if err != nil {
				return err
			}
			target, err := findVCSChange(changes, ref)
			if err != nil {
				return err
			}
			if target == nil {
				return fmt.Errorf("issue %s has no VCS change matching %q", issueID, ref)
			}

			// Only a manual attachment can be deleted; anything else must be detached.
			if target.State != nil && *target.State == api.VCSChangeStateManuallyAttached {
				if err := client.DeleteIssueVCSChange(issueID, target.ID); err != nil {
					return err
				}
				output.PrintSuccess(cmd, "removed VCS change "+target.ID+" from "+issueID)
				return nil
			}

			if err := client.SetIssueVCSChangeState(issueID, target.ID, api.VCSChangeStateDetached, mute); err != nil {
				return err
			}
			output.PrintSuccess(cmd, "detached VCS change "+target.ID+" from "+issueID)
			return nil
		},
	}

	addMuteFlag(cmd, &mute)
	return cmd
}

func addMuteFlag(cmd *cobra.Command, mute *bool) {
	cmd.Flags().BoolVar(mute, "mute-notifications", false,
		"suppress issue-change notifications; needs Apply Commands Silently permission")
}
