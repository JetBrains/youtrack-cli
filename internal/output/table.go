package output

import (
	"fmt"
	"io"
	"strings"
	"text/tabwriter"
	"time"
	"unicode/utf8"

	"github.com/JetBrains/youtrack-cli/api"
)

// PrintTable renders data as a human-readable table. Falls back to JSON for unknown types.
func PrintTable(out io.Writer, data any) error {
	w := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
	defer func() { _ = w.Flush() }()

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
		for _, u := range v {
			_, _ = fmt.Fprintf(w, "%s\t%s\t%s\n", u.Login, u.FullName, u.Email)
		}
	case []api.Comment:
		_, _ = fmt.Fprintln(w, "ID\tAUTHOR\tTEXT")
		for _, c := range v {
			author := ""
			if c.Author != nil {
				author = c.Author.Login
			}
			_, _ = fmt.Fprintf(w, "%s\t%s\t%s\n", c.ID, author, truncate(c.Text, 60))
		}
	case *api.Article:
		_, _ = fmt.Fprintln(w, "ID\tSUMMARY\tPROJECT")
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\n", v.IDReadable, truncate(v.Summary, 72), projectKey(v.Project))
	case []api.Article:
		_, _ = fmt.Fprintln(w, "ID\tSUMMARY\tPROJECT")
		for _, a := range v {
			_, _ = fmt.Fprintf(w, "%s\t%s\t%s\n", a.IDReadable, truncate(a.Summary, 72), projectKey(a.Project))
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
	case []api.RemovedDocument:
		_, _ = fmt.Fprintln(w, "ID\tSUMMARY")
		for _, d := range v {
			_, _ = fmt.Fprintf(w, "%s\t%s\n", d.IDReadable, truncate(d.Summary, 72))
		}
	case []api.GanttChart:
		_, _ = fmt.Fprintln(w, "ID\tNAME")
		for _, g := range v {
			_, _ = fmt.Fprintf(w, "%s\t%s\n", g.ID, g.Name)
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
	case []api.ArticleAttachment:
		_, _ = fmt.Fprintln(w, "ID\tNAME\tURL")
		for _, a := range v {
			_, _ = fmt.Fprintf(w, "%s\t%s\t%s\n", a.ID, a.Name, a.URL)
		}
	case *api.WorkItem:
		_, _ = fmt.Fprintln(w, "ID\tTEXT")
		_, _ = fmt.Fprintf(w, "%s\t%s\n", v.ID, v.Text)
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
	default:
		return printJSON(out, data)
	}
	return nil
}

// PrintPlain writes data as tab-separated values with no headers or borders.
// Falls back to JSON for types without a defined plain formatter.
func PrintPlain(out io.Writer, data any) error {
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
		for _, u := range v {
			write(u.Login, u.FullName, u.Email)
		}
	case []api.Comment:
		for _, c := range v {
			author := ""
			if c.Author != nil {
				author = c.Author.Login
			}
			write(c.ID, author, c.Text)
		}
	case *api.Article:
		write(v.IDReadable, v.Summary, projectKey(v.Project))
	case []api.Article:
		for _, a := range v {
			write(a.IDReadable, a.Summary, projectKey(a.Project))
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
	case []api.Workflow:
		for _, wf := range v {
			write(wf.ID, wf.Name, wf.Title)
		}
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
	case []api.Board:
		for _, b := range v {
			write(b.ID, b.Name)
		}
	case []api.Dashboard:
		for _, d := range v {
			write(d.ID, d.Name)
		}
	case []api.Tag:
		for _, t := range v {
			write(t.ID, t.Name)
		}
	case []api.SavedSearch:
		for _, s := range v {
			write(s.ID, s.Name, s.Query)
		}
	case []api.App:
		for _, a := range v {
			write(a.ID, a.Name, a.Title, a.Version)
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
	default:
		return printJSON(out, data)
	}
	return nil
}

// PrintIssueDetail renders a single issue in a rich multi-line plain-text format,
// including custom fields (State, Priority, Assignee, etc.) and the full description.
func PrintIssueDetail(w io.Writer, v *api.Issue) {
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
