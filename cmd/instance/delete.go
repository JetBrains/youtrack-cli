package instance

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/JetBrains/youtrack-cli/internal"
	"github.com/JetBrains/youtrack-cli/internal/config"
	"github.com/JetBrains/youtrack-cli/internal/output"
)

func newDeleteCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "delete <alias>",
		Aliases: []string{"remove"},
		Short:   "Delete a configured instance",
		Args:    cobra.ExactArgs(1),
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
			if abort, err := internal.ConfirmOrAbort(cmd, fmt.Sprintf("Delete instance %q (%s)?", alias, inst.URL)); abort {
				return err
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
			config.PurgeKeyringToken(alias)
			message := fmt.Sprintf("instance %q deleted", alias)
			if wasActive {
				message += "; run 'youtrack instance use <alias>' to set a new active instance"
			}
			output.PrintSuccess(cmd, message)
			return nil
		},
	}
	cmd.Flags().Bool("force", false, "Skip the confirmation prompt")
	return cmd
}
