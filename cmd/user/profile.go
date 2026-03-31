package user

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/JetBrains/youtrack-cli/internal"
)

func newUserProfileCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "profile", Short: "Manage user profile settings"}
	cmd.AddCommand(newUserProfilePropertyCmd())
	cmd.AddCommand(newUserProfileAppearanceCmd())
	cmd.AddCommand(newUserProfileNotificationsCmd())
	cmd.AddCommand(newUserProfileGrazieCmd())
	cmd.AddCommand(newUserProfileArticlesCmd())
	cmd.AddCommand(newUserProfileIssuesListCmd())
	cmd.AddCommand(newUserProfileGeneralCmd())
	cmd.AddCommand(newUserProfileTipsCmd())
	return cmd
}

func newUserProfilePropertyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "property <userId> <key> <value>",
		Short: "Set a profile property for a user",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			if err := client.SetUserProfileProperty(args[0], args[1], args[2]); err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), "updated profile property "+args[1])
			return nil
		},
	}
	return cmd
}

func newUserProfileAppearanceCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "appearance <userId>",
		Short: "Set appearance profile settings (JSON body)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			body, _ := cmd.Flags().GetString("body")
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			var m map[string]any
			if err := json.Unmarshal([]byte(body), &m); err != nil {
				return fmt.Errorf("invalid JSON: %w", err)
			}
			return client.SetUserAppearanceSetting(args[0], m)
		},
	}
	cmd.Flags().String("body", "{}", "JSON body")
	cmd.MarkFlagRequired("body")
	return cmd
}

func newUserProfileNotificationsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "notifications <userId>",
		Short: "Set notifications profile settings (JSON body)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			body, _ := cmd.Flags().GetString("body")
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			var m map[string]any
			if err := json.Unmarshal([]byte(body), &m); err != nil {
				return fmt.Errorf("invalid JSON: %w", err)
			}
			return client.SetUserNotificationsProfile(args[0], m)
		},
	}
	cmd.Flags().String("body", "{}", "JSON body")
	cmd.MarkFlagRequired("body")
	return cmd
}

func newUserProfileGrazieCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "grazie <userId>",
		Short: "Set Grazie (AI) profile settings (JSON body)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			body, _ := cmd.Flags().GetString("body")
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			var m map[string]any
			if err := json.Unmarshal([]byte(body), &m); err != nil {
				return fmt.Errorf("invalid JSON: %w", err)
			}
			return client.SetUserGrazieProfile(args[0], m)
		},
	}
	cmd.Flags().String("body", "{}", "JSON body")
	cmd.MarkFlagRequired("body")
	return cmd
}

func newUserProfileArticlesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "articles <userId>",
		Short: "Set articles profile settings (JSON body)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			body, _ := cmd.Flags().GetString("body")
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			var m map[string]any
			if err := json.Unmarshal([]byte(body), &m); err != nil {
				return fmt.Errorf("invalid JSON: %w", err)
			}
			return client.SetUserArticlesProfile(args[0], m)
		},
	}
	cmd.Flags().String("body", "{}", "JSON body")
	cmd.MarkFlagRequired("body")
	return cmd
}

func newUserProfileIssuesListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "issues-list <userId>",
		Short: "Set issues list profile settings (JSON body)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			body, _ := cmd.Flags().GetString("body")
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			var m map[string]any
			if err := json.Unmarshal([]byte(body), &m); err != nil {
				return fmt.Errorf("invalid JSON: %w", err)
			}
			return client.SetUserIssuesListSetting(args[0], m)
		},
	}
	cmd.Flags().String("body", "{}", "JSON body")
	cmd.MarkFlagRequired("body")
	return cmd
}

func newUserProfileTipsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tips <userId> <key=true|false>...",
		Short: "Set profile tip flags (key=true|false pairs)",
		Args:  cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			tips := make(map[string]bool)
			for _, pair := range args[1:] {
				k, v := pair, "true"
				if idx := len(pair) - 1; pair[idx] == '=' {
					k = pair[:idx]
				} else {
					for i, c := range pair {
						if c == '=' {
							k = pair[:i]
							v = pair[i+1:]
							break
						}
					}
				}
				tips[k] = v != "false"
			}
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			return client.SetUserProfileTips(args[0], tips)
		},
	}
	return cmd
}

func newUserProfileGeneralCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "general <userId>",
		Short: "Set general profile settings (JSON body)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			body, _ := cmd.Flags().GetString("body")
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			var m map[string]any
			if err := json.Unmarshal([]byte(body), &m); err != nil {
				return fmt.Errorf("invalid JSON: %w", err)
			}
			return client.SetUserGeneralProfile(args[0], m)
		},
	}
	cmd.Flags().String("body", "{}", "JSON body")
	cmd.MarkFlagRequired("body")
	return cmd
}
