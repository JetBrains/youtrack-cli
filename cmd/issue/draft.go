package issue

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/JetBrains/youtrack-cli/api"
	"github.com/JetBrains/youtrack-cli/internal"
	"github.com/JetBrains/youtrack-cli/internal/output"
)

func newIssueDraftCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "draft",
		Short: "Manage issue drafts (template-prefilled issue stubs)",
	}
	internal.RequireSubcommand(cmd)
	cmd.AddCommand(newIssueDraftCreateCmd())
	cmd.AddCommand(newIssueDraftUpdateCmd())
	cmd.AddCommand(newIssueDraftSubmitCmd())
	cmd.AddCommand(newIssueDraftDeleteCmd())
	return cmd
}

func newIssueDraftDeleteCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete <draftId>",
		Short: "Delete a draft",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if abort, err := internal.ConfirmOrAbort(cmd, fmt.Sprintf("Delete draft %s?", args[0])); abort {
				return err
			}
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			return client.DeleteDraft(args[0])
		},
	}
	cmd.Flags().Bool("force", false, "Skip the confirmation prompt")
	return cmd
}

func newIssueDraftCreateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a draft pre-filled from the project's issue template",
		RunE: func(cmd *cobra.Command, args []string) error {
			projectShortName, _ := cmd.Flags().GetString("project")
			summary, _ := cmd.Flags().GetString("summary")

			description, _, err := textFromFlags(cmd, "description", "description-file")
			if err != nil {
				return err
			}

			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}

			project, err := client.GetProject(projectShortName)
			if err != nil {
				return err
			}

			draft, err := client.CreateDraft(project.ID, summary, description)
			if err != nil {
				return err
			}
			return output.Print(cmd, draft)
		},
	}
	cmd.Flags().String("project", "", "Project short name (required)")
	cmd.Flags().String("summary", "", "Issue summary (optional; template may prefill)")
	cmd.Flags().String("description", "", "Issue description")
	addTextFileFlag(cmd, "description", "description-file", "the description")
	internal.MustMarkRequired(cmd, "project")
	return cmd
}

func newIssueDraftUpdateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update <draftId>",
		Short: "Update a draft's summary, description, or custom fields",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			draftID := args[0]
			summary, _ := cmd.Flags().GetString("summary")
			customFields, _ := cmd.Flags().GetString("custom-fields")
			customFieldsFile, _ := cmd.Flags().GetString("custom-fields-file")

			description, descriptionSet, err := textFromFlags(cmd, "description", "description-file")
			if err != nil {
				return err
			}
			if customFieldsFile != "" && customFields != "" {
				return fmt.Errorf("--custom-fields and --custom-fields-file are mutually exclusive")
			}
			if customFieldsFile != "" {
				b, err := os.ReadFile(customFieldsFile)
				if err != nil {
					return fmt.Errorf("read custom-fields file: %w", err)
				}
				customFields = string(b)
			}

			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}

			var draft *api.IssueDraft
			if summary != "" || descriptionSet {
				var descriptionPtr *string
				if descriptionSet {
					descriptionPtr = &description
				}
				draft, err = client.UpdateDraft(draftID, summary, descriptionPtr)
				if err != nil {
					return err
				}
			} else {
				draft, err = client.GetDraft(draftID)
				if err != nil {
					return err
				}
			}

			if customFields != "" {
				var fields map[string]string
				if err := json.Unmarshal([]byte(customFields), &fields); err != nil {
					return fmt.Errorf(`parse --custom-fields: %w (custom-fields must be a flat JSON object of string values, e.g. {"Priority":"High"})`, err)
				}
				for name, value := range fields {
					field := findDraftField(draft.Fields, name)
					if field == nil {
						available := draftFieldNames(draft.Fields)
						return fmt.Errorf("field %q not found in draft; available: %v", name, available)
					}
					if err := client.UpdateDraftField(draftID, field.ID, field.Type, value); err != nil {
						return fmt.Errorf("update field %q: %w", name, err)
					}
				}
				// Re-fetch to return updated state.
				draft, err = client.GetDraft(draftID)
				if err != nil {
					return err
				}
			}

			return output.Print(cmd, draft)
		},
	}
	cmd.Flags().String("summary", "", "New summary")
	cmd.Flags().String("description", "", "New description (an empty string clears it)")
	addTextFileFlag(cmd, "description", "description-file", "the description")
	cmd.Flags().String("custom-fields", "", `JSON object of field values, e.g. '{"Type":"Bug","Priority":"High"}'`)
	cmd.Flags().String("custom-fields-file", "", "Read custom fields JSON from file")
	return cmd
}

func newIssueDraftSubmitCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "submit <draftId>",
		Short: "Submit a draft as a real issue",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			issue, err := client.SubmitDraft(args[0])
			if err != nil {
				return err
			}
			return output.Print(cmd, issue)
		},
	}
}

func findDraftField(fields []api.IssueDraftField, name string) *api.IssueDraftField {
	for i := range fields {
		if fields[i].Name == name {
			return &fields[i]
		}
	}
	return nil
}

func draftFieldNames(fields []api.IssueDraftField) []string {
	names := make([]string, len(fields))
	for i, f := range fields {
		names[i] = f.Name
	}
	return names
}
