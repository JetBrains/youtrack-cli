package instance

import (
	"github.com/spf13/cobra"

	"github.com/JetBrains/youtrack-cli/internal"
	"github.com/JetBrains/youtrack-cli/internal/output"
)

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use: "version", Short: "Show the active instance's YouTrack version and build number",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			v, err := client.GetInstanceVersion()
			if err != nil {
				return err
			}
			return output.Print(cmd, v)
		},
	}
}
