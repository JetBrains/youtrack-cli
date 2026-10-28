package issue

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/JetBrains/youtrack-cli/api"
	"github.com/JetBrains/youtrack-cli/internal"
	"github.com/JetBrains/youtrack-cli/internal/output"
)

func newIssueCreateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "create",
		Aliases: []string{"add"},
		Short:   "Create a new issue",
		Example: "  youtrack issue create --project FOO --summary \"Login button broken\"\n  youtrack issue create --project FOO --summary \"Fix\" --custom-fields '{\"State\":\"Fixed\"}'",
		RunE: func(cmd *cobra.Command, args []string) error {
			projectShortName, _ := cmd.Flags().GetString("project")
			summary, _ := cmd.Flags().GetString("summary")
			customFields, _ := cmd.Flags().GetString("custom-fields")
			customFieldsFile, _ := cmd.Flags().GetString("custom-fields-file")
			parent, _ := cmd.Flags().GetString("parent")
			asUser, _ := cmd.Flags().GetString("asUser")

			description, _, err := textFromFlags(cmd, "description", "description-file")
			if err != nil {
				return err
			}

			if cmd.Flags().Changed("custom-fields-file") && cmd.Flags().Changed("custom-fields") {
				return fmt.Errorf("--custom-fields and --custom-fields-file are mutually exclusive")
			}
			if cmd.Flags().Changed("custom-fields-file") {
				b, err := os.ReadFile(customFieldsFile)
				if err != nil {
					return fmt.Errorf("read custom-fields file: %w", err)
				}
				customFields = string(b)
			}

			// Parse before creating so bad input never leaves a stray, partially-fielded issue.
			var customFieldsCmd string
			if cmd.Flags().Changed("custom-fields") || cmd.Flags().Changed("custom-fields-file") {
				q, err := jsonToCommand(customFields)
				if err != nil {
					return fmt.Errorf("parse custom-fields: %w", err)
				}
				customFieldsCmd = q
			}

			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}

			project, err := client.GetProject(projectShortName)
			if err != nil {
				return err
			}

			req := api.AddIssueRequest{
				Summary:     summary,
				Description: description,
				Project:     &api.Project{ID: project.ID},
			}

			if asUser != "" {
				user, err := client.GetUser(asUser)
				if err != nil {
					return err
				}
				req.Reporter = &api.User{ID: user.ID}
			}

			issue, err := client.AddIssue(req)
			if err != nil {
				return err
			}

			// Custom fields and parent apply as follow-up commands; roll back the issue if either fails so create stays atomic.
			var applyErr error
			if customFieldsCmd != "" {
				if err := client.ApplyCommand(customFieldsCmd, []string{issue.IDReadable}); err != nil {
					applyErr = fmt.Errorf("apply custom fields: %w", err)
				}
			}
			if applyErr == nil && parent != "" {
				if err := client.ApplyCommand("subtask of "+parent, []string{issue.IDReadable}); err != nil {
					applyErr = fmt.Errorf("set parent: %w", err)
				}
			}
			if applyErr != nil {
				if delErr := client.DeleteIssue(issue.IDReadable); delErr != nil {
					return fmt.Errorf("%w; issue %s was created but could not be rolled back: %w", applyErr, issue.IDReadable, delErr)
				}
				return fmt.Errorf("%w; created issue %s was rolled back", applyErr, issue.IDReadable)
			}

			return output.Print(cmd, issue)
		},
	}
	cmd.Flags().String("project", "", "Project short name (required)")
	cmd.Flags().String("summary", "", "Issue summary (required)")
	cmd.Flags().String("description", "", "Issue description")
	addTextFileFlag(cmd, "description", "description-file", "the description")
	cmd.Flags().String("custom-fields", "", `JSON object of custom field values, e.g. '{"State":"Fixed"}'`)
	cmd.Flags().String("custom-fields-file", "", "Read custom fields JSON from file")
	cmd.Flags().String("parent", "", "Parent issue ID (e.g. FOO-1)")
	cmd.Flags().String("asUser", "", "Create as this user (login)")
	internal.MustMarkRequired(cmd, "project", "summary")
	return cmd
}

// jsonToCommand converts {"Type":"Bug","Subsystem":"UI"} into a YouTrack command,
// preserving JSON key order so conditional fields can be ordered before dependents
// (e.g. Type before Subsystem). User logins are braced downstream by api.WrapUserValues.
func jsonToCommand(jsonStr string) (string, error) {
	shape := fmt.Errorf(`custom-fields must be a flat JSON object of string values, e.g. {"Type":"Bug","Assignee":"alice.qa"}`)
	// Validate the whole document before decoding in order: More alone does not
	// distinguish a closing brace from EOF or reject content after the object.
	if !json.Valid([]byte(jsonStr)) {
		return "", shape
	}
	dec := json.NewDecoder(strings.NewReader(jsonStr))
	if tok, err := dec.Token(); err != nil || tok != json.Delim('{') {
		return "", shape
	}
	var parts []string
	for dec.More() {
		key, err := dec.Token()
		if err != nil {
			return "", shape
		}
		var value *string
		if err := dec.Decode(&value); err != nil {
			return "", fmt.Errorf("%w (%w)", err, shape)
		}
		if value == nil {
			return "", shape
		}
		parts = append(parts, key.(string), *value)
	}
	return strings.Join(parts, " "), nil
}
