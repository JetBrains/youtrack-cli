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
		Long: "Create an issue. Summary, description, custom fields, Type and tags all go in the one request\n" +
			"that creates it, so workflows that require a field accept it.\n" +
			"Only --parent is applied afterwards; if that fails, the new issue is deleted again.",
		Example: "  youtrack issue create --project FOO --summary \"Login button broken\"\n" +
			"  youtrack issue create --project FOO --summary \"Fix\" --type Task --custom-fields '{\"Priority\":\"Major\",\"Assignee\":\"me\"}'\n" +
			"  youtrack issue create --project FOO --summary \"Leak\" --tag security",
		RunE: func(cmd *cobra.Command, args []string) error {
			projectShortName, _ := cmd.Flags().GetString("project")
			summary, _ := cmd.Flags().GetString("summary")
			customFields, _ := cmd.Flags().GetString("custom-fields")
			customFieldsFile, _ := cmd.Flags().GetString("custom-fields-file")
			parent, _ := cmd.Flags().GetString("parent")
			asUser, _ := cmd.Flags().GetString("asUser")
			issueType, _ := cmd.Flags().GetString("type")
			tagNames, _ := cmd.Flags().GetStringArray("tag")

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
			var fieldValues [][2]string
			if cmd.Flags().Changed("custom-fields") || cmd.Flags().Changed("custom-fields-file") {
				pairs, err := parseCustomFields(customFields)
				if err != nil {
					return fmt.Errorf("parse custom-fields: %w", err)
				}
				fieldValues = pairs
			}
			if cmd.Flags().Changed("type") {
				for _, kv := range fieldValues {
					if strings.EqualFold(kv[0], "Type") {
						return fmt.Errorf("--type and a Type entry in --custom-fields are mutually exclusive")
					}
				}
				fieldValues = append(fieldValues, [2]string{"Type", issueType})
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
			if req.CustomFields, err = buildCustomFields(client, project, fieldValues); err != nil {
				return err
			}
			if req.Tags, err = resolveTags(client, tagNames); err != nil {
				return err
			}
			if parent != "" {
				if _, err := client.GetIssue(parent); err != nil {
					return fmt.Errorf("parent %s: %w", parent, err)
				}
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

			// YouTrack ignores a parent in the create request, so it is linked by a follow-up command;
			// roll back if that fails so a retry does not leave a duplicate.
			if parent != "" {
				if err := client.ApplyCommand("subtask of "+parent, []string{issue.IDReadable}); err != nil {
					if delErr := client.DeleteIssue(issue.IDReadable); delErr != nil {
						return fmt.Errorf("set parent: %w; issue %s was created but could not be rolled back: %w", err, issue.IDReadable, delErr)
					}
					return fmt.Errorf("set parent: %w; created issue %s was rolled back", err, issue.IDReadable)
				}
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
	cmd.Flags().String("type", "", "Issue Type, e.g. Task (the project default, usually Bug, otherwise)")
	cmd.Flags().StringArray("tag", nil, "Tag the issue, by tag name or ID (repeatable)")
	cmd.Flags().String("parent", "", "Parent issue ID (e.g. FOO-1)")
	cmd.Flags().String("asUser", "", "Create as this user (login)")
	internal.MustMarkRequired(cmd, "project", "summary")
	return cmd
}

// parseCustomFields reads {"Type":"Bug","Subsystem":"UI"} into name/value pairs in document order.
func parseCustomFields(jsonStr string) ([][2]string, error) {
	shape := fmt.Errorf(`custom-fields must be a flat JSON object of string values, e.g. {"Type":"Bug","Assignee":"alice.qa"}`)
	// Validate the whole document before decoding in order: More alone does not
	// distinguish a closing brace from EOF or reject content after the object.
	if !json.Valid([]byte(jsonStr)) {
		return nil, shape
	}
	dec := json.NewDecoder(strings.NewReader(jsonStr))
	if tok, err := dec.Token(); err != nil || tok != json.Delim('{') {
		return nil, shape
	}
	var pairs [][2]string
	for dec.More() {
		key, err := dec.Token()
		if err != nil {
			return nil, shape
		}
		var value *string
		if err := dec.Decode(&value); err != nil {
			return nil, fmt.Errorf("%w (%w)", err, shape)
		}
		if value == nil {
			return nil, shape
		}
		pairs = append(pairs, [2]string{key.(string), *value})
	}
	return pairs, nil
}

// buildCustomFields types each value by the project's field schema; "me" in a user field is the current user.
func buildCustomFields(client *api.Client, project *api.Project, values [][2]string) ([]api.IssueCustomFieldValue, error) {
	if len(values) == 0 {
		return nil, nil
	}
	schemas, err := client.GetProjectFieldSchemas(project.ID)
	if err != nil {
		return nil, fmt.Errorf("read fields of project %s: %w", project.ShortName, err)
	}
	var me string
	fields := make([]api.IssueCustomFieldValue, 0, len(values))
	for _, kv := range values {
		schema, ok := api.FindProjectField(schemas, kv[0])
		if !ok {
			return nil, fmt.Errorf("project %s has no field %q; its fields are: %s",
				project.ShortName, kv[0], strings.Join(api.ProjectFieldNames(schemas), ", "))
		}
		value := kv[1]
		if schema.IsUserField() && strings.EqualFold(value, "me") {
			if me == "" {
				user, err := client.GetCurrentUser()
				if err != nil {
					return nil, err
				}
				me = user.Login
			}
			value = me
		}
		field, err := api.BuildIssueCustomField(schema, value)
		if err != nil {
			return nil, err
		}
		fields = append(fields, field)
	}
	return fields, nil
}

// resolveTags finds tags by name or ID among the tags the current user can use.
func resolveTags(client *api.Client, names []string) ([]api.Tag, error) {
	var tags []api.Tag
	for _, name := range names {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		id, err := client.ResolveTagID(name)
		if err != nil {
			return nil, err
		}
		tags = append(tags, api.Tag{ID: id})
	}
	return tags, nil
}
