package issue

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/JetBrains/youtrack-cli/api"
	"github.com/JetBrains/youtrack-cli/internal"
	"github.com/JetBrains/youtrack-cli/internal/output"
)

func newIssueListCmd() *cobra.Command {
	var limit int
	var offset int

	cmd := &cobra.Command{
		Use:   "list",
		Short: "Search issues by query",
		Long: "Search issues by query.\n\n" +
			"--fields picks the table/plain columns, like the visible columns of the issue list in the YouTrack UI.\n" +
			"Each entry is either an issue attribute (idReadable, summary, reporter, created, project.shortName, ...)\n" +
			"or, for any other name, a custom field of the issue's project (Priority, State, Assignee, \"Fix versions\").\n" +
			"Attribute and custom field names are case-insensitive; attribute names win on a clash.\n" +
			"JSON output keeps the full issue objects, with only the requested custom fields under customFields.",
		Example: "  youtrack issue list --query \"assignee: me #Unresolved\"\n" +
			"  youtrack issue list --query \"project: FOO\" --limit 20 --offset 20 -f idReadable,summary,project.shortName\n" +
			"  youtrack issue list --query \"project: FOO #Unresolved\" --fields idReadable,summary,Priority,State,Assignee",
		RunE: func(cmd *cobra.Command, args []string) error {
			if limit < 0 {
				return fmt.Errorf("--limit must not be negative, got %d", limit)
			}
			if offset < 0 {
				return fmt.Errorf("--offset must not be negative, got %d", offset)
			}
			query, _ := cmd.Flags().GetString("query")
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			fields, _ := cmd.Root().PersistentFlags().GetString("fields")
			cols := internal.IssueColumns(fields)
			customFields := internal.IssueCustomFieldNames(cols)
			// Custom field names travel in customFields=; only identifier-like ones also go in fields=, which must stay URL-safe.
			client.Fields = internal.IssueRequestFields(cols)
			issues, err := client.ListIssuesPageWithCustomFields(query, limit, offset, customFields)
			if err != nil {
				return err
			}
			if len(issues) == 0 {
				output.PrintTip(cmd, "No issues found — try a different --query or check your permissions")
			} else if missing := missingCustomFields(issues, customFields); len(missing) > 0 {
				output.PrintWarn(cmd, fmt.Sprintf(
					"no custom field named %s in the projects of the listed issues — check the names with: youtrack project field list <project>",
					strings.Join(missing, ", ")))
			}
			if len(cols) == 0 {
				return output.Print(cmd, issues)
			}
			return output.PrintIssueColumns(cmd, issues, cols)
		},
	}
	cmd.Flags().String("query", "", "YouTrack query string (required)")
	cmd.Flags().IntVar(&limit, "limit", 100, "Maximum number of issues to return (non-negative; 0 returns none)")
	cmd.Flags().IntVar(&offset, "offset", 0, "Number of issues to skip (non-negative)")
	internal.MustMarkRequired(cmd, "query")
	return cmd
}

// missingCustomFields returns the quoted requested names that no listed issue has as a custom field or attribute.
// YouTrack returns every project field on an issue, even unset ones, so absence means the name matched nothing.
func missingCustomFields(issues []api.Issue, names []string) []string {
	var missing []string
	for _, name := range names {
		found := false
		for i := 0; i < len(issues) && !found; i++ {
			for _, f := range issues[i].CustomFields {
				if strings.EqualFold(f.Name, name) {
					found = true
					break
				}
			}
			if !found {
				found = hasAttribute(&issues[i], name)
			}
		}
		if !found {
			missing = append(missing, fmt.Sprintf("%q", name))
		}
	}
	return missing
}

// hasAttribute reports whether the server returned a top-level attribute named name on the issue.
func hasAttribute(issue *api.Issue, name string) bool {
	b, err := json.Marshal(issue)
	if err != nil {
		return false
	}
	var obj map[string]json.RawMessage
	if json.Unmarshal(b, &obj) != nil {
		return false
	}
	_, ok := obj[name]
	return ok
}
