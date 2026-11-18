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
			// The prompt runs unlocked; Remove re-reads the config under the lock so it cannot drop a concurrent refresh.
			_, wasActive, purgeErr, err := config.Remove(cmd.Context(), alias, inst.URL)
			if err != nil {
				return err
			}
			if purgeErr != nil {
				output.PrintWarn(cmd, purgeErr.Error())
			}
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
