package cmd

import (
	"github.com/spf13/cobra"

	"github.com/JetBrains/youtrack-cli/cmd/output"
	"github.com/JetBrains/youtrack-cli/internal"
)

func newSettingsCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "settings", Short: "Manage instance settings"}
	cmd.AddCommand(
		newSettingsGetCmd(),
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

func newSettingsGetCmd() *cobra.Command {
	return &cobra.Command{
		Use: "get", Short: "Get global settings",
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
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			return client.EnableFeatureFlag(name, enabled)
		},
	}
	cmd.Flags().String("name", "", "Feature flag name (required)")
	cmd.Flags().Bool("enabled", true, "Enable or disable")
	cmd.MarkFlagRequired("name")
	return cmd
}

func newSettingsEmailCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "email", Short: "Manage email settings"}
	set := &cobra.Command{
		Use: "set", Short: "Configure email server",
		RunE: func(cmd *cobra.Command, args []string) error {
			server, _ := cmd.Flags().GetString("server")
			port, _ := cmd.Flags().GetInt("port")
			from, _ := cmd.Flags().GetString("from")
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
	set.MarkFlagRequired("server")
	set.MarkFlagRequired("from")
	disable := &cobra.Command{
		Use: "disable", Short: "Disable email notifications",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			return client.DisableEmailSettings()
		},
	}
	cmd.AddCommand(set, disable)
	return cmd
}

func newSettingsBannerCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "banner", Short: "Manage global banner"}
	set := &cobra.Command{
		Use: "set", Short: "Show a global banner message",
		RunE: func(cmd *cobra.Command, args []string) error {
			msg, _ := cmd.Flags().GetString("message")
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			return client.SetGlobalBanner(msg, true)
		},
	}
	set.Flags().String("message", "", "Banner message (required)")
	set.MarkFlagRequired("message")
	disable := &cobra.Command{
		Use: "disable", Short: "Disable the global banner",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			return client.SetGlobalBanner("", false)
		},
	}
	cmd.AddCommand(set, disable)
	return cmd
}

func newSettingsReactionsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use: "reactions", Short: "Enable or disable reactions",
		RunE: func(cmd *cobra.Command, args []string) error {
			enabled, _ := cmd.Flags().GetBool("enabled")
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			return client.SetReactionsEnabled(enabled)
		},
	}
	cmd.Flags().Bool("enabled", true, "Enable reactions")
	return cmd
}

func newSettingsLicenseCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use: "license", Short: "Set the YouTrack license",
		RunE: func(cmd *cobra.Command, args []string) error {
			user, _ := cmd.Flags().GetString("user")
			key, _ := cmd.Flags().GetString("key")
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			return client.SetLicense(user, key)
		},
	}
	cmd.Flags().String("user", "", "License username (required)")
	cmd.Flags().String("key", "", "License key (required)")
	cmd.MarkFlagRequired("user")
	cmd.MarkFlagRequired("key")
	return cmd
}

func newSettingsAICmd() *cobra.Command {
	cmd := &cobra.Command{Use: "ai", Short: "Manage AI/Grazie settings"}
	jb := &cobra.Command{
		Use: "jetbrains", Short: "Enable/disable JetBrains AI",
		RunE: func(cmd *cobra.Command, args []string) error {
			enabled, _ := cmd.Flags().GetBool("enabled")
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			return client.SwitchJetBrainsAI(enabled)
		},
	}
	jb.Flags().Bool("enabled", true, "Enable JetBrains AI")
	tc := &cobra.Command{
		Use: "text-completion", Short: "Enable/disable text completion",
		RunE: func(cmd *cobra.Command, args []string) error {
			enabled, _ := cmd.Flags().GetBool("enabled")
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			return client.SwitchTextCompletion(enabled)
		},
	}
	tc.Flags().Bool("enabled", true, "Enable text completion")
	gc := &cobra.Command{
		Use: "grammar", Short: "Enable/disable grammar correction",
		RunE: func(cmd *cobra.Command, args []string) error {
			enabled, _ := cmd.Flags().GetBool("enabled")
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			return client.SwitchGrammarCorrection(enabled)
		},
	}
	gc.Flags().Bool("enabled", true, "Enable grammar correction")
	cmd.AddCommand(jb, tc, gc)
	return cmd
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
