package instance

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/JetBrains/youtrack-cli/internal"
	"github.com/JetBrains/youtrack-cli/internal/config"
)

func newRemoveCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "remove <alias>",
		Short: "Remove a configured instance",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			alias := args[0]
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			inst, ok := cfg.Instances[alias]
			if !ok {
				return fmt.Errorf("instance %q not found — run: youtrack instance list", alias)
			}
			confirmed, err := internal.Confirm(cmd, fmt.Sprintf("Remove instance %q (%s)?", alias, inst.URL))
			if err != nil {
				return err
			}
			if !confirmed {
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Canceled.")
				return nil
			}
			wasActive := cfg.CurrentInstance == alias
			if err := cfg.RemoveInstance(alias); err != nil {
				return err
			}
			if wasActive {
				cfg.CurrentInstance = ""
			}
			if err := cfg.Save(); err != nil {
				return err
			}
			if wasActive {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(),
					"Instance %q removed.\nRun: youtrack instance use <alias> to set a new active instance.\n", alias)
			} else {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Instance %q removed.\n", alias)
			}
			return nil
		},
	}
	cmd.Flags().Bool("force", false, "Skip the confirmation prompt")
	return cmd
}
