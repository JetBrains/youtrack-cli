package issue

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/JetBrains/youtrack-cli/api"
	"github.com/JetBrains/youtrack-cli/internal"
	"github.com/JetBrains/youtrack-cli/internal/output"
)

func newIssueVisibilityCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "visibility <issueId>",
		Short: "Set who can see an issue",
		Long:  visibilityLong("issue"),
		Example: "  youtrack issue visibility FOO-1 --groups \"Core Team\" --users alice\n" +
			"  youtrack issue visibility FOO-1 --public",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSetVisibility(cmd, args[0], func(c *api.Client, v api.VisibilityChange) (*api.Visibility, string, error) {
				return c.SetIssueVisibility(args[0], v)
			})
		},
	}
	addSetVisibilityFlags(cmd)
	return cmd
}

func visibilityLong(entity string) string {
	return "Set who can see the " + entity + ". The groups and users given replace both lists: a list left out is cleared,\n" +
		"so --users alice alone makes the " + entity + " visible to alice only. Use --public to remove the restriction.\n" +
		"The visibility is read back after the write and a mismatch is reported as an error."
}

func addSetVisibilityFlags(cmd *cobra.Command) {
	cmd.Flags().StringSlice("groups", nil, "Groups that can see it, by name or ID (comma-separated)")
	cmd.Flags().StringSlice("users", nil, "Users that can see it, by login or ID (comma-separated)")
	cmd.Flags().Bool("public", false, "Remove the restriction: visible to everyone who can see the project")
	cmd.MarkFlagsMutuallyExclusive("public", "groups")
	cmd.MarkFlagsMutuallyExclusive("public", "users")
	cmd.MarkFlagsOneRequired("groups", "users", "public")
}

// runSetVisibility applies --groups/--users/--public; set returns the re-read visibility and a user the server may add on its own.
func runSetVisibility(cmd *cobra.Command, label string, set func(*api.Client, api.VisibilityChange) (*api.Visibility, string, error)) error {
	groups, _ := cmd.Flags().GetStringSlice("groups")
	users, _ := cmd.Flags().GetStringSlice("users")
	public, _ := cmd.Flags().GetBool("public")
	groups, users = nonEmpty(groups), nonEmpty(users)
	if !public && len(groups)+len(users) == 0 {
		return fmt.Errorf("pass at least one group or user, or --public; empty lists would make it public")
	}

	client, err := internal.GetClient(cmd)
	if err != nil {
		return err
	}
	want := api.VisibilityChange{Public: public}
	for _, g := range groups {
		id, err := resolveGroupID(client, g)
		if err != nil {
			return err
		}
		want.GroupIDs = append(want.GroupIDs, id)
	}
	for _, u := range users {
		id, err := resolveUserID(client, u)
		if err != nil {
			return err
		}
		want.UserIDs = append(want.UserIDs, id)
	}

	got, allowedUserID, err := set(client, want)
	if err != nil {
		return err
	}
	if !want.MatchesAllowing(got, allowedUserID) {
		return fmt.Errorf("visibility of %s was not applied as requested; it is now %s", label, api.DescribeVisibility(got))
	}
	output.PrintSuccess(cmd, fmt.Sprintf("updated visibility for %s: %s", label, api.DescribeVisibility(got)))
	return nil
}
