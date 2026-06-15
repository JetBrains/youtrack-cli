package user

import (
	"github.com/spf13/cobra"

	"github.com/JetBrains/youtrack-cli/internal"
	"github.com/JetBrains/youtrack-cli/internal/output"
)

func newUserTagCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "tag", Short: "View tags associated with a user"}
	list := &cobra.Command{
		Use:   "list <userId>",
		Short: "List tags owned by a user",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			tags, err := client.ListUserTags(args[0])
			if err != nil {
				return err
			}
			return output.Print(cmd, tags)
		},
	}
	cmd.AddCommand(list)
	return cmd
}
