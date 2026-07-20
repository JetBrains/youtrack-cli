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
		newSettingsSystemCmd(),
		newSettingsFeatureCmd(),
		newSettingsEmailCmd(),
		newSettingsBannerCmd(),
		newSettingsReactionsCmd(),
		newSettingsLicenseCmd(),
		newSettingsAICmd(),
		newSettingsFrontendCmd(),
	)
	return cmd
}

func newSettingsSystemCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "system", Short: "Manage system settings"}
	internal.RequireSubcommand(cmd)
	view := &cobra.Command{
		Use: "view", Aliases: []string{"get"}, Short: "View system settings",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			s, err := client.GetSystemSettings()
			if err != nil {
				return err
			}
			return output.Print(cmd, s)
		},
	}
	update := &cobra.Command{
		Use: "update", Short: "Update system settings",
		RunE: func(cmd *cobra.Command, args []string) error {
			if abort, err := internal.ConfirmOrAbort(cmd, "Update instance-wide system settings?"); abort {
				return err
			}
			baseURL, _ := cmd.Flags().GetString("base-url")
			maxSize, _ := cmd.Flags().GetInt64("max-upload-size")
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			s, err := client.UpdateSystemSettings(baseURL, maxSize)
			if err != nil {
				return err
			}
			return output.Print(cmd, s)
		},
	}
	update.Flags().String("base-url", "", "Instance base URL")
	update.Flags().Int64("max-upload-size", 0, "Max upload file size in bytes")
	update.Flags().Bool("force", false, "Skip the confirmation prompt")
	cmd.AddCommand(view, update)
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

func newSettingsEmailCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "email", Short: "Manage email settings"}
	internal.RequireSubcommand(cmd)
	set := &cobra.Command{
		Use: "set", Short: "Configure email server",
		RunE: func(cmd *cobra.Command, args []string) error {
			server, _ := cmd.Flags().GetString("server")
			port, _ := cmd.Flags().GetInt("port")
			from, _ := cmd.Flags().GetString("from")
			if abort, err := internal.ConfirmOrAbort(cmd, fmt.Sprintf(
				"Configure instance-wide email server to %s:%d?", server, port)); abort {
				return err
			}
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			return client.SetEmailSettings(server, port, from)
		},
	}
	set.Flags().String("server", "", "SMTP server (required)")
	set.Flags().Int("port", 25, "SMTP port")
	set.Flags().String("from", "", "From address (required)")
	set.Flags().Bool("force", false, "Skip the confirmation prompt")
	internal.MustMarkRequired(set, "server", "from")
	disable := &cobra.Command{
		Use: "disable", Short: "Disable email notifications",
		RunE: func(cmd *cobra.Command, args []string) error {
			if abort, err := internal.ConfirmOrAbort(cmd,
				"Disable email notifications for the whole instance?"); abort {
				return err
			}
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			return client.DisableEmailSettings()
		},
	}
	disable.Flags().Bool("force", false, "Skip the confirmation prompt")
	cmd.AddCommand(set, disable)
	return cmd
}

func newSettingsBannerCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "banner", Short: "Manage global banner"}
	internal.RequireSubcommand(cmd)
	set := &cobra.Command{
		Use: "set", Short: "Show a global banner message",
		RunE: func(cmd *cobra.Command, args []string) error {
			msg, _ := cmd.Flags().GetString("message")
			if abort, err := internal.ConfirmOrAbort(cmd, fmt.Sprintf(
				"Show this banner to every user on the instance: %q?", msg)); abort {
				return err
			}
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			return client.SetGlobalBanner(msg, true)
		},
	}
	set.Flags().String("message", "", "Banner message (required)")
	set.Flags().Bool("force", false, "Skip the confirmation prompt")
	internal.MustMarkRequired(set, "message")
	disable := &cobra.Command{
		Use: "disable", Short: "Disable the global banner",
		RunE: func(cmd *cobra.Command, args []string) error {
			if abort, err := internal.ConfirmOrAbort(cmd, "Disable the global banner?"); abort {
				return err
			}
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			return client.SetGlobalBanner("", false)
		},
	}
	disable.Flags().Bool("force", false, "Skip the confirmation prompt")
	cmd.AddCommand(set, disable)
	return cmd
}

func newSettingsReactionsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use: "reactions", Short: "Enable or disable reactions",
		RunE: func(cmd *cobra.Command, args []string) error {
			enabled, _ := cmd.Flags().GetBool("enabled")
			verb := "Enable"
			if !enabled {
				verb = "Disable"
			}
			if abort, err := internal.ConfirmOrAbort(cmd, fmt.Sprintf(
				"%s reactions for the whole instance?", verb)); abort {
				return err
			}
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			return client.SetReactionsEnabled(enabled)
		},
	}
	cmd.Flags().Bool("enabled", true, "Enable reactions")
	cmd.Flags().Bool("force", false, "Skip the confirmation prompt")
	return cmd
}

func newSettingsLicenseCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use: "license", Short: "Set the YouTrack license",
		RunE: func(cmd *cobra.Command, args []string) error {
			user, _ := cmd.Flags().GetString("user")
			key, _ := cmd.Flags().GetString("key")
			if abort, err := internal.ConfirmOrAbort(cmd,
				"Overwrite the active instance license?"); abort {
				return err
			}
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			return client.SetLicense(user, key)
		},
	}
	cmd.Flags().String("user", "", "License username (required)")
	cmd.Flags().String("key", "", "License key (required)")
	cmd.Flags().Bool("force", false, "Skip the confirmation prompt")
	internal.MustMarkRequired(cmd, "user", "key")
	return cmd
}

func newSettingsAICmd() *cobra.Command {
	cmd := &cobra.Command{Use: "ai", Short: "Manage AI/Grazie settings"}
	internal.RequireSubcommand(cmd)
	jb := &cobra.Command{
		Use: "jetbrains", Short: "Enable/disable JetBrains AI",
		RunE: func(cmd *cobra.Command, args []string) error {
			enabled, _ := cmd.Flags().GetBool("enabled")
			if abort, err := internal.ConfirmOrAbort(cmd, confirmAIToggleMsg("JetBrains AI", enabled)); abort {
				return err
			}
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			return client.SwitchJetBrainsAI(enabled)
		},
	}
	jb.Flags().Bool("enabled", true, "Enable JetBrains AI")
	jb.Flags().Bool("force", false, "Skip the confirmation prompt")
	tc := &cobra.Command{
		Use: "text-completion", Short: "Enable/disable text completion",
		RunE: func(cmd *cobra.Command, args []string) error {
			enabled, _ := cmd.Flags().GetBool("enabled")
			if abort, err := internal.ConfirmOrAbort(cmd, confirmAIToggleMsg("text completion", enabled)); abort {
				return err
			}
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			return client.SwitchTextCompletion(enabled)
		},
	}
	tc.Flags().Bool("enabled", true, "Enable text completion")
	tc.Flags().Bool("force", false, "Skip the confirmation prompt")
	gc := &cobra.Command{
		Use: "grammar", Short: "Enable/disable grammar correction",
		RunE: func(cmd *cobra.Command, args []string) error {
			enabled, _ := cmd.Flags().GetBool("enabled")
			if abort, err := internal.ConfirmOrAbort(cmd, confirmAIToggleMsg("grammar correction", enabled)); abort {
				return err
			}
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			return client.SwitchGrammarCorrection(enabled)
		},
	}
	gc.Flags().Bool("enabled", true, "Enable grammar correction")
	gc.Flags().Bool("force", false, "Skip the confirmation prompt")
	cmd.AddCommand(jb, tc, gc)
	return cmd
}

func confirmAIToggleMsg(feature string, enabled bool) string {
	verb := "Enable"
	if !enabled {
		verb = "Disable"
	}
	return fmt.Sprintf("%s %s for the whole instance?", verb, feature)
}

func newSettingsFrontendCmd() *cobra.Command {
	return &cobra.Command{
		Use: "frontend", Short: "Get frontend configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			cfg, err := client.GetFrontendConfig()
			if err != nil {
				return err
			}
			return output.Print(cmd, cfg)
		},
	}
}
