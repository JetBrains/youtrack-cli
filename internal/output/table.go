package output

import (
	"fmt"
	"io"
	"strings"
	"text/tabwriter"
	"time"
	"unicode/utf8"

	"github.com/JetBrains/youtrack-cli/api"
	"github.com/JetBrains/youtrack-cli/internal"
)

func commentText(comment *api.Comment) string {
	if comment.Deleted {
		return "[deleted]"
	}
	return comment.Text
}

// PrintTable renders data as a human-readable table. Falls back to JSON for unknown types.
func PrintTable(out io.Writer, data any) error {
	tw := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
	defer func() { _ = tw.Flush() }()
	w := safeWriter{tw}

	switch v := data.(type) {
	case *api.Issue:
		_, _ = fmt.Fprintln(w, "ID\tSUMMARY")
		_, _ = fmt.Fprintf(w, "%s\t%s\n", v.IDReadable, truncate(v.Summary, 72))
	case []api.Issue:
		_, _ = fmt.Fprintln(w, "ID\tSUMMARY")
		for i := range v {
			_, _ = fmt.Fprintf(w, "%s\t%s\n", v[i].IDReadable, truncate(v[i].Summary, 72))
		}
	case *api.Project:
		_, _ = fmt.Fprintln(w, "KEY\tNAME")
		_, _ = fmt.Fprintf(w, "%s\t%s\n", v.ShortName, v.Name)
	case []api.Project:
		_, _ = fmt.Fprintln(w, "KEY\tNAME")
		for _, p := range v {
			_, _ = fmt.Fprintf(w, "%s\t%s\n", p.ShortName, p.Name)
		}
	case *api.User:
		_, _ = fmt.Fprintln(w, "LOGIN\tNAME\tEMAIL")
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\n", v.Login, v.FullName, v.Email)
	case []api.User:
		_, _ = fmt.Fprintln(w, "LOGIN\tNAME\tEMAIL")
		for i := range v {
			_, _ = fmt.Fprintf(w, "%s\t%s\t%s\n", v[i].Login, v[i].FullName, v[i].Email)
		}
	case []api.Comment:
		_, _ = fmt.Fprintln(w, "ID\tAUTHOR\tTEXT")
		for _, c := range v {
			author := ""
			if c.Author != nil {
				author = c.Author.Login
			}
			_, _ = fmt.Fprintf(w, "%s\t%s\t%s\n", c.ID, author, truncate(commentText(&c), 60))
		}
	case *api.Comment:
		_, _ = fmt.Fprintln(w, "ID\tAUTHOR\tTEXT")
		author := ""
		if v.Author != nil {
			author = v.Author.Login
		}
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\n", v.ID, author, truncate(commentText(v), 60))
	case *api.Article:
		_, _ = fmt.Fprintln(w, "ID\tSUMMARY\tPROJECT")
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\n", v.IDReadable, truncate(v.Summary, 72), projectKey(v.Project))
	case []api.Article:
		_, _ = fmt.Fprintln(w, "ID\tSUMMARY\tPROJECT")
		for i := range v {
			_, _ = fmt.Fprintf(w, "%s\t%s\t%s\n", v[i].IDReadable, truncate(v[i].Summary, 72), projectKey(v[i].Project))
		}
	case *api.Group:
		_, _ = fmt.Fprintln(w, "ID\tNAME")
		_, _ = fmt.Fprintf(w, "%s\t%s\n", v.ID, v.Name)
	case []api.Group:
		_, _ = fmt.Fprintln(w, "ID\tNAME")
		for _, g := range v {
			_, _ = fmt.Fprintf(w, "%s\t%s\n", g.ID, g.Name)
		}
	case *api.Role:
		_, _ = fmt.Fprintln(w, "ID\tNAME\tDESCRIPTION")
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\n", v.ID, v.Name, v.Description)
	case []api.Role:
		_, _ = fmt.Fprintln(w, "ID\tNAME\tDESCRIPTION")
		for _, r := range v {
			_, _ = fmt.Fprintf(w, "%s\t%s\t%s\n", r.ID, r.Name, truncate(r.Description, 60))
		}
	case []api.Permission:
		_, _ = fmt.Fprintln(w, "ID\tNAME")
		for _, p := range v {
			_, _ = fmt.Fprintf(w, "%s\t%s\n", p.ID, p.Name)
		}
	case []api.AssignedRole:
		_, _ = fmt.Fprintln(w, "ID\tHOLDER\tROLE\tSCOPE")
		for _, a := range v {
			_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", a.ID, assignedRoleHolder(a.Holder), roleName(a.Role), assignedRoleScope(a.Scope))
		}
	case *api.Workflow:
		_, _ = fmt.Fprintln(w, "ID\tNAME\tTITLE")
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\n", v.ID, v.Name, v.Title)
	case []api.Workflow:
		_, _ = fmt.Fprintln(w, "ID\tNAME\tTITLE")
		for _, wf := range v {
			_, _ = fmt.Fprintf(w, "%s\t%s\t%s\n", wf.ID, wf.Name, wf.Title)
		}
	case *api.WorkflowRule:
		_, _ = fmt.Fprintln(w, "ID\tNAME\tTYPE\tTITLE")
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", v.ID, v.Name, v.Type, v.Title)
		_ = tw.Flush()
		printRuleScript(safeWriter{out}, v.Script)
	case *api.CustomField:
		_, _ = fmt.Fprintln(w, "ID\tNAME\tTYPE")
		typeID := ""
		if v.FieldType != nil {
			typeID = v.FieldType.ID
		}
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\n", v.ID, v.Name, typeID)
	case []api.CustomField:
		_, _ = fmt.Fprintln(w, "ID\tNAME\tTYPE")
		for _, f := range v {
			typeID := ""
			if f.FieldType != nil {
				typeID = f.FieldType.ID
			}
			_, _ = fmt.Fprintf(w, "%s\t%s\t%s\n", f.ID, f.Name, typeID)
		}
	case []api.Bundle:
		_, _ = fmt.Fprintln(w, "ID\tNAME")
		for _, b := range v {
			_, _ = fmt.Fprintf(w, "%s\t%s\n", b.ID, b.Name)
		}
	case []api.WorkItemType:
		_, _ = fmt.Fprintln(w, "ID\tNAME")
		for _, t := range v {
			_, _ = fmt.Fprintf(w, "%s\t%s\n", t.ID, t.Name)
		}
	case *api.Board:
		_, _ = fmt.Fprintln(w, "ID\tNAME")
		_, _ = fmt.Fprintf(w, "%s\t%s\n", v.ID, v.Name)
	case []api.Board:
		_, _ = fmt.Fprintln(w, "ID\tNAME")
		for _, b := range v {
			_, _ = fmt.Fprintf(w, "%s\t%s\n", b.ID, b.Name)
		}
	case *api.Sprint:
		_, _ = fmt.Fprintln(w, "ID\tNAME")
		_, _ = fmt.Fprintf(w, "%s\t%s\n", v.ID, v.Name)
	case []api.Sprint:
		_, _ = fmt.Fprintln(w, "ID\tNAME")
		for _, s := range v {
			_, _ = fmt.Fprintf(w, "%s\t%s\n", s.ID, s.Name)
		}
	case *api.Dashboard:
		_, _ = fmt.Fprintln(w, "ID\tNAME")
		_, _ = fmt.Fprintf(w, "%s\t%s\n", v.ID, v.Name)
	case []api.Dashboard:
		_, _ = fmt.Fprintln(w, "ID\tNAME")
		for _, d := range v {
			_, _ = fmt.Fprintf(w, "%s\t%s\n", d.ID, d.Name)
		}
	case *api.Tag:
		_, _ = fmt.Fprintln(w, "ID\tNAME")
		_, _ = fmt.Fprintf(w, "%s\t%s\n", v.ID, v.Name)
	case []api.Tag:
		_, _ = fmt.Fprintln(w, "ID\tNAME")
		for _, t := range v {
			_, _ = fmt.Fprintf(w, "%s\t%s\n", t.ID, t.Name)
		}
	case *api.SavedSearch:
		_, _ = fmt.Fprintln(w, "ID\tNAME\tQUERY")
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\n", v.ID, v.Name, v.Query)
	case []api.SavedSearch:
		_, _ = fmt.Fprintln(w, "ID\tNAME\tQUERY")
		for _, s := range v {
			_, _ = fmt.Fprintf(w, "%s\t%s\t%s\n", s.ID, s.Name, truncate(s.Query, 60))
		}
	case *api.Organization:
		_, _ = fmt.Fprintln(w, "ID\tNAME")
		_, _ = fmt.Fprintf(w, "%s\t%s\n", v.ID, v.Name)
	case []api.Organization:
		_, _ = fmt.Fprintln(w, "ID\tNAME")
		for _, o := range v {
			_, _ = fmt.Fprintf(w, "%s\t%s\n", o.ID, o.Name)
		}
	case []api.App:
		_, _ = fmt.Fprintln(w, "ID\tNAME\tTITLE\tVERSION")
		for _, a := range v {
			_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", a.ID, a.Name, a.Title, a.Version)
		}
	case *api.CanvasNode:
		_, _ = fmt.Fprintln(w, "ID\tLABEL\tX\tY")
		_, _ = fmt.Fprintf(w, "%s\t%s\t%d\t%d\n", v.ID, v.Label, v.X, v.Y)
	case *api.Canvas:
		_, _ = fmt.Fprintln(w, "ID\tNAME")
		_, _ = fmt.Fprintf(w, "%s\t%s\n", v.ID, v.Name)
	case []api.Canvas:
		_, _ = fmt.Fprintln(w, "ID\tNAME")
		for _, c := range v {
			_, _ = fmt.Fprintf(w, "%s\t%s\n", c.ID, c.Name)
		}
	case []api.GanttChart:
		_, _ = fmt.Fprintln(w, "ID\tNAME")
		for _, g := range v {
			_, _ = fmt.Fprintf(w, "%s\t%s\n", g.ID, g.Name)
		}
	case *api.GanttChart:
		_, _ = fmt.Fprintln(w, "ID\tNAME")
		_, _ = fmt.Fprintf(w, "%s\t%s\n", v.ID, v.Name)
	case *api.GanttResource:
		_, _ = fmt.Fprintln(w, "ID\tISSUE")
		_, _ = fmt.Fprintf(w, "%s\t%s\n", v.ID, issueKey(v.Issue))
	case []api.GanttResource:
		_, _ = fmt.Fprintln(w, "ID\tISSUE")
		for _, resource := range v {
			_, _ = fmt.Fprintf(w, "%s\t%s\n", resource.ID, issueKey(resource.Issue))
		}
	case []api.ProjectTemplate:
		_, _ = fmt.Fprintln(w, "ID\tNAME\tKEY")
		for _, t := range v {
			_, _ = fmt.Fprintf(w, "%s\t%s\t%s\n", t.ID, t.Name, t.ShortName)
		}
	case *api.IssueDraft:
		_, _ = fmt.Fprintln(w, "ID\tSUMMARY")
		_, _ = fmt.Fprintf(w, "%s\t%s\n", v.ID, truncate(v.Summary, 72))
	case []api.IssueDraft:
		_, _ = fmt.Fprintln(w, "ID\tSUMMARY")
		for _, d := range v {
			_, _ = fmt.Fprintf(w, "%s\t%s\n", d.ID, truncate(d.Summary, 72))
		}
	case []api.IssueAttachment:
		_, _ = fmt.Fprintln(w, "ID\tNAME\tURL")
		for _, a := range v {
			_, _ = fmt.Fprintf(w, "%s\t%s\t%s\n", a.ID, a.Name, a.URL)
		}
	case *api.IssueAttachment:
		_, _ = fmt.Fprintln(w, "ID\tNAME\tURL")
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\n", v.ID, v.Name, v.URL)
	case []api.ArticleAttachment:
		_, _ = fmt.Fprintln(w, "ID\tNAME\tURL")
		for _, a := range v {
			_, _ = fmt.Fprintf(w, "%s\t%s\t%s\n", a.ID, a.Name, a.URL)
		}
	case *api.ArticleAttachment:
		_, _ = fmt.Fprintln(w, "ID\tNAME\tURL")
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\n", v.ID, v.Name, v.URL)
	case *api.WorkItem:
		_, _ = fmt.Fprintln(w, "ID\tTEXT")
		_, _ = fmt.Fprintf(w, "%s\t%s\n", v.ID, v.Text)
	case []api.WorkItem:
		_, _ = fmt.Fprintln(w, "ID\tTEXT")
		for _, item := range v {
			_, _ = fmt.Fprintf(w, "%s\t%s\n", item.ID, item.Text)
		}
	case []api.IssueLink:
		_, _ = fmt.Fprintln(w, "ID\tTYPE\tISSUES")
		for _, l := range v {
			typeName := ""
			if l.LinkType != nil {
				typeName = l.LinkType.Name
			}
			ids := make([]string, len(l.Issues))
			for i := range l.Issues {
				ids[i] = l.Issues[i].IDReadable
			}
			_, _ = fmt.Fprintf(w, "%s\t%s\t%s\n", l.ID, typeName, joinStrings(ids))
		}
	case []api.LinkType:
		_, _ = fmt.Fprintln(w, "ID\tNAME\tSOURCE-TO-TARGET\tTARGET-TO-SOURCE")
		for _, lt := range v {
			_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", lt.ID, lt.Name, lt.SourceToTarget, lt.TargetToSource)
		}
	case []api.ProjectCustomField:
		_, _ = fmt.Fprintln(w, "ID\tFIELD")
		for _, f := range v {
			name := ""
			if f.Field != nil {
				name = f.Field.Name
			}
			_, _ = fmt.Fprintf(w, "%s\t%s\n", f.ID, name)
		}
	case *api.VCSChange:
		_, _ = fmt.Fprintln(w, "ID\tVERSION\tDATE\tAUTHOR\tSTATE\tSUMMARY")
		writeVCSChangeRow(w, v)
	case []api.VCSChange:
		_, _ = fmt.Fprintln(w, "ID\tVERSION\tDATE\tAUTHOR\tSTATE\tSUMMARY")
		for i := range v {
			writeVCSChangeRow(w, &v[i])
		}
	case *internal.UpdateStatus:
		if v.UpdateAvailable {
			_, _ = fmt.Fprintf(w, "YouTrack CLI %s is available (you have %s)\n",
				internal.FormatVersion(v.LatestVersion), internal.FormatVersion(v.CurrentVersion))
			_, _ = fmt.Fprintf(w, "Upgrade:\t%s\n", v.UpgradeCommand)
		} else {
			// Written to tw directly so the colored icon's escape sequence survives sanitizing.
			_, _ = fmt.Fprintf(tw, "%s YouTrack CLI %s — up to date\n", successIcon, internal.FormatVersion(v.CurrentVersion))
		}
	default:
		return printJSON(out, data)
	}
	return nil
}

// PrintPlain writes data as tab-separated values with no headers or borders.
// Returns an explicit error for types without a defined plain formatter.
func PrintPlain(out io.Writer, data any) error {
	out = safeWriter{out}
	write := func(fields ...string) {
		_, _ = fmt.Fprintln(out, strings.Join(fields, "\t"))
	}
	switch v := data.(type) {
	case *api.Issue:
		write(v.IDReadable, v.Summary)
	case []api.Issue:
		for i := range v {
			write(v[i].IDReadable, v[i].Summary)
		}
	case *api.Project:
		write(v.ShortName, v.Name)
	case []api.Project:
		for _, p := range v {
			write(p.ShortName, p.Name)
		}
	case *api.User:
		write(v.Login, v.FullName, v.Email)
	case []api.User:
		for i := range v {
			write(v[i].Login, v[i].FullName, v[i].Email)
		}
	case []api.Comment:
		for _, c := range v {
			author := ""
			if c.Author != nil {
				author = c.Author.Login
			}
			write(c.ID, author, commentText(&c))
		}
	case *api.Comment:
		author := ""
		if v.Author != nil {
			author = v.Author.Login
		}
		write(v.ID, author, commentText(v))
	case *api.Article:
		write(v.IDReadable, v.Summary, projectKey(v.Project))
	case []api.Article:
		for i := range v {
			write(v[i].IDReadable, v[i].Summary, projectKey(v[i].Project))
		}
	case *api.Group:
		write(v.ID, v.Name)
	case []api.Group:
		for _, g := range v {
			write(g.ID, g.Name)
		}
	case *api.Role:
		write(v.ID, v.Name, v.Description)
	case []api.Role:
		for _, r := range v {
			write(r.ID, r.Name, r.Description)
		}
	case []api.Permission:
		for _, p := range v {
			write(p.ID, p.Name)
		}
	case []api.AssignedRole:
		for _, a := range v {
			write(a.ID, assignedRoleHolder(a.Holder), roleName(a.Role), assignedRoleScope(a.Scope))
		}
	case *api.Workflow:
		write(v.ID, v.Name, v.Title)
	case []api.Workflow:
		for _, wf := range v {
			write(wf.ID, wf.Name, wf.Title)
		}
	case *api.WorkflowRule:
		write(v.ID, v.Name, v.Type, v.Title)
		printRuleScript(out, v.Script)
	case *api.CustomField:
		typeID := ""
		if v.FieldType != nil {
			typeID = v.FieldType.ID
		}
		write(v.ID, v.Name, typeID)
	case []api.CustomField:
		for _, f := range v {
			typeID := ""
			if f.FieldType != nil {
				typeID = f.FieldType.ID
			}
			write(f.ID, f.Name, typeID)
		}
	case []api.Bundle:
		for _, b := range v {
			write(b.ID, b.Name)
		}
	case []api.WorkItemType:
		for _, itemType := range v {
			write(itemType.ID, itemType.Name)
		}
	case *api.Board:
		write(v.ID, v.Name)
	case []api.Board:
		for _, b := range v {
			write(b.ID, b.Name)
		}
	case *api.Sprint:
		write(v.ID, v.Name)
	case []api.Sprint:
		for _, sprint := range v {
			write(sprint.ID, sprint.Name)
		}
	case *api.Dashboard:
		write(v.ID, v.Name)
	case []api.Dashboard:
		for _, d := range v {
			write(d.ID, d.Name)
		}
	case *api.Tag:
		write(v.ID, v.Name)
	case []api.Tag:
		for _, t := range v {
			write(t.ID, t.Name)
		}
	case *api.SavedSearch:
		write(v.ID, v.Name, v.Query)
	case []api.SavedSearch:
		for _, s := range v {
			write(s.ID, s.Name, s.Query)
		}
	case *api.Organization:
		write(v.ID, v.Name)
	case []api.Organization:
		for _, org := range v {
			write(org.ID, org.Name)
		}
	case []api.App:
		for _, a := range v {
			write(a.ID, a.Name, a.Title, a.Version)
		}
	case *api.CanvasNode:
		write(v.ID, v.Label, fmt.Sprint(v.X), fmt.Sprint(v.Y))
	case *api.Canvas:
		write(v.ID, v.Name)
	case []api.Canvas:
		for _, canvas := range v {
			write(canvas.ID, canvas.Name)
		}
	case *api.GanttChart:
		write(v.ID, v.Name)
	case []api.GanttChart:
		for _, chart := range v {
			write(chart.ID, chart.Name)
		}
	case *api.GanttResource:
		write(v.ID, issueKey(v.Issue))
	case []api.GanttResource:
		for _, resource := range v {
			write(resource.ID, issueKey(resource.Issue))
		}
	case []api.ProjectTemplate:
		for _, tmpl := range v {
			write(tmpl.ID, tmpl.Name, tmpl.ShortName)
		}
	case []api.IssueLink:
		for _, l := range v {
			typeName := ""
			if l.LinkType != nil {
				typeName = l.LinkType.Name
			}
			ids := make([]string, len(l.Issues))
			for i := range l.Issues {
				ids[i] = l.Issues[i].IDReadable
			}
			write(l.ID, typeName, joinStrings(ids))
		}
	case []api.LinkType:
		for _, lt := range v {
			write(lt.ID, lt.Name, lt.SourceToTarget, lt.TargetToSource)
		}
	case *api.VCSChange:
		write(v.ID, v.Version, vcsChangeAuthor(v), vcsChangeSummary(v))
	case []api.VCSChange:
		for i := range v {
			write(v[i].ID, v[i].Version, vcsChangeAuthor(&v[i]), vcsChangeSummary(&v[i]))
		}
	case []api.ProjectCustomField:
		for _, f := range v {
			name := ""
			if f.Field != nil {
				name = f.Field.Name
			}
			write(f.ID, name)
		}
	case *api.IssueDraft:
		write(v.ID, v.Summary)
	case []api.IssueDraft:
		for _, d := range v {
			write(d.ID, d.Summary)
		}
	case []api.IssueAttachment:
		for _, attachment := range v {
			write(attachment.ID, attachment.Name, attachment.URL)
		}
	case *api.IssueAttachment:
		write(v.ID, v.Name, v.URL)
	case []api.ArticleAttachment:
		for _, attachment := range v {
			write(attachment.ID, attachment.Name, attachment.URL)
		}
	case *api.ArticleAttachment:
		write(v.ID, v.Name, v.URL)
	case *api.WorkItem:
		write(v.ID, v.Text)
	case []api.WorkItem:
		for _, item := range v {
			write(item.ID, item.Text)
		}
	case *internal.UpdateStatus:
		if v.UpdateAvailable {
			write(internal.FormatVersion(v.LatestVersion), v.UpgradeCommand)
		} else {
			write(internal.FormatVersion(v.CurrentVersion), "up-to-date")
		}
	default:
		return fmt.Errorf("plain output is not supported for %T; use --json or select scalar columns with --fields", data)
	}
	return nil
}

// PrintIssueDetail renders a single issue in a rich multi-line plain-text format,
// including custom fields (State, Priority, Assignee, etc.) and the full description.
func PrintIssueDetail(w io.Writer, v *api.Issue) {
	w = safeWriter{w}
	_, _ = fmt.Fprintf(w, "%s  %s\n", v.IDReadable, v.Summary)

	label := func(k, val string) {
		if val != "" {
			_, _ = fmt.Fprintf(w, "%-12s%s\n", k+":", val)
		}
	}

	_, _ = fmt.Fprintln(w)
	reporter := ""
	if v.Reporter != nil {
		reporter = v.Reporter.Login
	}
	label("Reporter", reporter)
	label("Project", projectKey(v.Project))
	if v.Created != 0 {
		label("Created", time.UnixMilli(v.Created).UTC().Format("2006-01-02 15:04:05"))
	}
	if v.Resolved != nil && *v.Resolved != 0 {
		label("Resolved", time.UnixMilli(*v.Resolved).UTC().Format("2006-01-02 15:04:05"))
	}

	if len(v.CustomFields) > 0 {
		_, _ = fmt.Fprintln(w)
		for _, f := range v.CustomFields {
			label(f.Name, issueFieldValue(f.Value))
		}
	}

	if v.Description != "" {
		_, _ = fmt.Fprintln(w)
		_, _ = fmt.Fprintln(w, v.Description)
	}
}

// issueFieldValue converts a custom field value (any) to a display string.
func issueFieldValue(v any) string {
	switch val := v.(type) {
	case nil:
		return ""
	case map[string]any:
		return compactObject(val)
	case []any:
		parts := make([]string, 0, len(val))
		for _, item := range val {
			if m, ok := item.(map[string]any); ok {
				if s := compactObject(m); s != "" {
					parts = append(parts, s)
				}
			}
		}
		return strings.Join(parts, ", ")
	default:
		return fmt.Sprintf("%v", v)
	}
}

func truncate(s string, max int) string {
	if utf8.RuneCountInString(s) <= max {
		return s
	}
	return string([]rune(s)[:max-3]) + "..."
}

func joinStrings(ss []string) string {
	result := ""
	for i, s := range ss {
		if i > 0 {
			result += ","
		}
		result += s
	}
	return result
}

func projectKey(p *api.Project) string {
	if p == nil {
		return ""
	}
	return p.ShortName
}

func issueKey(issue *api.Issue) string {
	if issue == nil {
		return ""
	}
	return issue.IDReadable
}

func roleName(r *api.Role) string {
	if r == nil {
		return ""
	}
	return r.Name
}

func assignedRoleHolder(h *api.RoleHolder) string {
	if h == nil {
		return ""
	}
	if h.Login != "" {
		return h.Login
	}
	return h.Name
}

func assignedRoleScope(s *api.RoleScope) string {
	if s == nil {
		return ""
	}
	if s.Project != nil && s.Project.ShortName != "" {
		return s.Project.ShortName
	}
	if s.Organization != nil && s.Organization.Name != "" {
		return s.Organization.Name
	}
	switch s.Type {
	case "GlobalScope":
		return "global"
	case "ProjectScope":
		return "project"
	case "OrganizationScope":
		return "organization"
	}
	return s.Type
}

// vcsChangeStateNames labels the state codes YouTrack stores on a VCS change.
var vcsChangeStateNames = map[int]string{
	api.VCSChangeStateDefault:          "attached",
	api.VCSChangeStateManuallyAttached: "manual",
	api.VCSChangeStateDetached:         "detached",
	api.VCSChangeStateLegacy:           "legacy",
}

func writeVCSChangeRow(w io.Writer, ch *api.VCSChange) {
	version := ch.Version
	if len(version) > 12 {
		version = version[:12]
	}
	date := ""
	if ch.Date != 0 {
		date = time.UnixMilli(ch.Date).UTC().Format("2006-01-02 15:04")
	}
	state := ""
	if ch.State != nil {
		if name, ok := vcsChangeStateNames[*ch.State]; ok {
			state = name
		} else {
			state = fmt.Sprintf("%d", *ch.State)
		}
	}
	_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
		ch.ID, version, date, vcsChangeAuthor(ch), state, vcsChangeSummary(ch))
}

func vcsChangeAuthor(ch *api.VCSChange) string {
	if ch.Author == nil {
		return ""
	}
	if ch.Author.Login != "" {
		return ch.Author.Login
	}
	return ch.Author.FullName
}

// vcsChangeSummary is the pull request title, or a commit's first message line.
func vcsChangeSummary(ch *api.VCSChange) string {
	if ch.Title != "" {
		return truncate(ch.Title, 60)
	}
	line := strings.TrimSpace(ch.Text)
	if idx := strings.IndexByte(line, '\n'); idx >= 0 {
		line = strings.TrimSpace(line[:idx])
	}
	return truncate(line, 60)
}

func printRuleScript(w io.Writer, script string) {
	if script == "" {
		return
	}
	_, _ = fmt.Fprintf(w, "\n%s", script)
	if !strings.HasSuffix(script, "\n") {
		_, _ = fmt.Fprintln(w)
	}
}
