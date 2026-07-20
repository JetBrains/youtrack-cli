package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/JetBrains/youtrack-cli/internal"
	"github.com/JetBrains/youtrack-cli/internal/output"
)

func newSettingsCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "settings", Short: "Manage instance settings"}
	internal.RequireSubcommand(cmd)
	cmd.AddCommand(
		newSettingsViewCmd(),
		newSettingsFeatureCmd(),
	)
	return cmd
}

func newSettingsViewCmd() *cobra.Command {
	return &cobra.Command{
		Use: "view", Aliases: []string{"get"}, Short: "View global settings",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			s, err := client.GetGlobalSettings()
			if err != nil {
				return err
			}
			return output.Print(cmd, s)
		},
	}
}

func newSettingsFeatureCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use: "feature", Short: "Enable or disable a feature flag",
		Long: "Enable or disable an instance-wide YouTrack feature flag by its internal ID.\n" +
			"There is no discovery endpoint for flag IDs — get them from YouTrack " +
			"support, release notes, or the instance's admin UI (Administration > " +
			"System Settings) before running this.",
		RunE: func(cmd *cobra.Command, args []string) error {
			name, _ := cmd.Flags().GetString("name")
			enabled, _ := cmd.Flags().GetBool("enabled")
			verb := "Enable"
			if !enabled {
				verb = "Disable"
			}
			if abort, err := internal.ConfirmOrAbort(cmd, fmt.Sprintf(
				"%s instance-wide feature flag %s?", verb, name)); abort {
				return err
			}
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			return client.EnableFeatureFlag(name, enabled)
		},
	}
	cmd.Flags().String("name", "", "Feature flag name (required)")
	cmd.Flags().Bool("enabled", true, "Enable or disable")
	cmd.Flags().Bool("force", false, "Skip the confirmation prompt")
	internal.MustMarkRequired(cmd, "name")
	return cmd
}
