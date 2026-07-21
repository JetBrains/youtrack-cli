package cmd

import (
	"strings"

	"github.com/spf13/cobra"

	"github.com/JetBrains/youtrack-cli/api"
	"github.com/JetBrains/youtrack-cli/internal"
	"github.com/JetBrains/youtrack-cli/internal/output"
)

func newFieldCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "field", Short: "Manage custom fields"}
	internal.RequireSubcommand(cmd)
	cmd.AddCommand(
		newFieldListCmd(),
		newFieldCreateCmd(),
		newFieldDeleteCmd(),
		newFieldBundleCmd(),
		newFieldWorkItemCmd(),
		newFieldPrivacyCmd(),
		newFieldWorkTimeCmd(),
		newFieldPrototypesCmd(),
	)
	return cmd
}

func newFieldListCmd() *cobra.Command {
	return &cobra.Command{
		Use: "list", Short: "List all custom fields",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			fields, err := client.ListCustomFields()
			if err != nil {
				return err
			}
			return output.Print(cmd, fields)
		},
	}
}

func newFieldCreateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use: "create", Aliases: []string{"add"}, Short: "Create a custom field",
		RunE: func(cmd *cobra.Command, args []string) error {
			name, _ := cmd.Flags().GetString("name")
			fieldType, _ := cmd.Flags().GetString("type")
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			field, err := client.AddCustomField(name, fieldType)
			if err != nil {
				return err
			}
			return output.Print(cmd, field)
		},
	}
	cmd.Flags().String("name", "", "Field name (required)")
	cmd.Flags().String("type", "", "Field type ID (required). Valid values: "+strings.Join(api.AllFieldTypes, ", "))
	internal.MustMarkRequired(cmd, "name", "type")
	return cmd
}

func newFieldDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use: "delete <id>", Short: "Delete a custom field",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			if err := client.DeleteCustomField(args[0]); err != nil {
				return err
			}
			output.PrintSuccess(cmd, "deleted field "+args[0])
			return nil
		},
	}
}

func newFieldBundleCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "bundle", Short: "Manage field bundles"}
	internal.RequireSubcommand(cmd)
	list := &cobra.Command{
		Use: "list", Short: "List bundles of a given type",
		RunE: func(cmd *cobra.Command, args []string) error {
			bundleType, _ := cmd.Flags().GetString("type")
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			bundles, err := client.ListBundles(bundleType)
			if err != nil {
				return err
			}
			return output.Print(cmd, bundles)
		},
	}
	list.Flags().String("type", "EnumBundle", "Bundle type")
	create := &cobra.Command{
		Use: "create", Short: "Create a bundle",
		RunE: func(cmd *cobra.Command, args []string) error {
			bundleType, _ := cmd.Flags().GetString("type")
			name, _ := cmd.Flags().GetString("name")
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			bundle, err := client.CreateBundle(bundleType, name)
			if err != nil {
				return err
			}
			return output.Print(cmd, bundle)
		},
	}
	create.Flags().String("type", "EnumBundle", "Bundle type")
	create.Flags().String("name", "", "Bundle name (required)")
	internal.MustMarkRequired(create, "name")
	addValue := &cobra.Command{
		Use: "add-value", Short: "Add a value to a bundle",
		RunE: func(cmd *cobra.Command, args []string) error {
			bundleType, _ := cmd.Flags().GetString("type")
			id, _ := cmd.Flags().GetString("id")
			value, _ := cmd.Flags().GetString("value")
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			return client.AddBundleValue(bundleType, id, value)
		},
	}
	addValue.Flags().String("type", "EnumBundle", "Bundle type")
	addValue.Flags().String("id", "", "Bundle ID (required)")
	addValue.Flags().String("value", "", "Value name (required)")
	internal.MustMarkRequired(addValue, "id", "value")
	attach := &cobra.Command{
		Use: "attach", Short: "Attach a bundle to a field",
		RunE: func(cmd *cobra.Command, args []string) error {
			fieldID, _ := cmd.Flags().GetString("field")
			bundleID, _ := cmd.Flags().GetString("id")
			bundleType, _ := cmd.Flags().GetString("type")
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			return client.AttachBundleToField(fieldID, bundleID, bundleType)
		},
	}
	attach.Flags().String("field", "", "Field ID (required)")
	attach.Flags().String("id", "", "Bundle ID (required)")
	attach.Flags().String("type", "EnumBundle", "Bundle type")
	internal.MustMarkRequired(attach, "field", "id")
	cmd.AddCommand(list, create, addValue, attach)
	return cmd
}

func newFieldPrivacyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "privacy <fieldId>",
		Short: "Change privacy of a custom field",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			private, _ := cmd.Flags().GetBool("private")
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			return client.ChangeCustomFieldPrivacy(args[0], private)
		},
	}
	cmd.Flags().Bool("private", true, "Set field as private (default true)")
	return cmd
}

func newFieldWorkTimeCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "worktime", Short: "Manage work time settings"}
	internal.RequireSubcommand(cmd)
	view := &cobra.Command{
		Use: "view", Aliases: []string{"get"}, Short: "View work time settings",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			settings, err := client.GetWorkTimeSettings()
			if err != nil {
				return err
			}
			return output.Print(cmd, settings)
		},
	}
	set := &cobra.Command{
		Use: "set", Short: "Set work time settings",
		RunE: func(cmd *cobra.Command, args []string) error {
			presentation, _ := cmd.Flags().GetString("minutes-a-day")
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			return client.SetWorkTimeSettings(presentation)
		},
	}
	set.Flags().String("minutes-a-day", "", "Minutes-a-day presentation string (required)")
	internal.MustMarkRequired(set, "minutes-a-day")
	cmd.AddCommand(view, set)
	return cmd
}

func newFieldPrototypesCmd() *cobra.Command {
	return &cobra.Command{
		Use: "prototypes", Short: "List custom field prototypes",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			fields, err := client.GetCustomFieldPrototypes()
			if err != nil {
				return err
			}
			return output.Print(cmd, fields)
		},
	}
}

func newFieldWorkItemCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "workitem", Short: "Manage work item types"}
	internal.RequireSubcommand(cmd)
	list := &cobra.Command{
		Use: "list", Short: "List work item types",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			types, err := client.ListWorkItemTypes()
			if err != nil {
				return err
			}
			return output.Print(cmd, types)
		},
	}
	add := &cobra.Command{
		Use: "add-type", Short: "Add a work item type",
		RunE: func(cmd *cobra.Command, args []string) error {
			name, _ := cmd.Flags().GetString("name")
			autoAttach, _ := cmd.Flags().GetBool("auto-attach")
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			t, err := client.AddWorkItemType(name, autoAttach)
			if err != nil {
				return err
			}
			return output.Print(cmd, t)
		},
	}
	add.Flags().String("name", "", "Type name (required)")
	add.Flags().Bool("auto-attach", false, "Auto-attach to all projects")
	internal.MustMarkRequired(add, "name")
	cmd.AddCommand(list, add)
	return cmd
}
