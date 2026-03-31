package output

import (
	"fmt"
	"io"
	"text/tabwriter"
	"unicode/utf8"

	"github.com/JetBrains/youtrack-cli/api"
)

// PrintTable renders data as a human-readable table. Falls back to JSON for unknown types.
func PrintTable(out io.Writer, data any) error {
	w := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
	defer w.Flush()

	switch v := data.(type) {
	case *api.Issue:
		fmt.Fprintln(w, "ID\tSUMMARY\tPROJECT")
		fmt.Fprintf(w, "%s\t%s\t%s\n", v.IDReadable, v.Summary, projectKey(v.Project))
	case []api.Issue:
		fmt.Fprintln(w, "ID\tSUMMARY\tPROJECT")
		for _, i := range v {
			fmt.Fprintf(w, "%s\t%s\t%s\n", i.IDReadable, i.Summary, projectKey(i.Project))
		}
	case *api.Project:
		fmt.Fprintln(w, "KEY\tNAME")
		fmt.Fprintf(w, "%s\t%s\n", v.ShortName, v.Name)
	case []api.Project:
		fmt.Fprintln(w, "KEY\tNAME")
		for _, p := range v {
			fmt.Fprintf(w, "%s\t%s\n", p.ShortName, p.Name)
		}
	case *api.User:
		fmt.Fprintln(w, "LOGIN\tNAME\tEMAIL")
		fmt.Fprintf(w, "%s\t%s\t%s\n", v.Login, v.FullName, v.Email)
	case []api.User:
		fmt.Fprintln(w, "LOGIN\tNAME\tEMAIL")
		for _, u := range v {
			fmt.Fprintf(w, "%s\t%s\t%s\n", u.Login, u.FullName, u.Email)
		}
	case []api.Comment:
		fmt.Fprintln(w, "ID\tAUTHOR\tTEXT")
		for _, c := range v {
			author := ""
			if c.Author != nil {
				author = c.Author.Login
			}
			text := c.Text
			if utf8.RuneCountInString(text) > 60 {
				text = string([]rune(text)[:57]) + "..."
			}
			fmt.Fprintf(w, "%s\t%s\t%s\n", c.ID, author, text)
		}
	case *api.Article:
		fmt.Fprintln(w, "ID\tSUMMARY\tPROJECT")
		fmt.Fprintf(w, "%s\t%s\t%s\n", v.IDReadable, v.Summary, projectKey(v.Project))
	case []api.Article:
		fmt.Fprintln(w, "ID\tSUMMARY\tPROJECT")
		for _, a := range v {
			fmt.Fprintf(w, "%s\t%s\t%s\n", a.IDReadable, a.Summary, projectKey(a.Project))
		}
	case *api.Group:
		fmt.Fprintln(w, "ID\tNAME")
		fmt.Fprintf(w, "%s\t%s\n", v.ID, v.Name)
	case []api.Group:
		fmt.Fprintln(w, "ID\tNAME")
		for _, g := range v {
			fmt.Fprintf(w, "%s\t%s\n", g.ID, g.Name)
		}
	case *api.Role:
		fmt.Fprintln(w, "ID\tNAME\tDESCRIPTION")
		fmt.Fprintf(w, "%s\t%s\t%s\n", v.ID, v.Name, v.Description)
	case []api.Role:
		fmt.Fprintln(w, "ID\tNAME\tDESCRIPTION")
		for _, r := range v {
			fmt.Fprintf(w, "%s\t%s\t%s\n", r.ID, r.Name, r.Description)
		}
	case []api.Permission:
		fmt.Fprintln(w, "ID\tNAME")
		for _, p := range v {
			fmt.Fprintf(w, "%s\t%s\n", p.ID, p.Name)
		}
	case *api.Workflow:
		fmt.Fprintln(w, "ID\tNAME\tTITLE")
		fmt.Fprintf(w, "%s\t%s\t%s\n", v.ID, v.Name, v.Title)
	case []api.Workflow:
		fmt.Fprintln(w, "ID\tNAME\tTITLE")
		for _, wf := range v {
			fmt.Fprintf(w, "%s\t%s\t%s\n", wf.ID, wf.Name, wf.Title)
		}
	case *api.CustomField:
		fmt.Fprintln(w, "ID\tNAME\tTYPE")
		typeID := ""
		if v.FieldType != nil {
			typeID = v.FieldType.ID
		}
		fmt.Fprintf(w, "%s\t%s\t%s\n", v.ID, v.Name, typeID)
	case []api.CustomField:
		fmt.Fprintln(w, "ID\tNAME\tTYPE")
		for _, f := range v {
			typeID := ""
			if f.FieldType != nil {
				typeID = f.FieldType.ID
			}
			fmt.Fprintf(w, "%s\t%s\t%s\n", f.ID, f.Name, typeID)
		}
	case []api.Bundle:
		fmt.Fprintln(w, "ID\tNAME")
		for _, b := range v {
			fmt.Fprintf(w, "%s\t%s\n", b.ID, b.Name)
		}
	case []api.WorkItemType:
		fmt.Fprintln(w, "ID\tNAME")
		for _, t := range v {
			fmt.Fprintf(w, "%s\t%s\n", t.ID, t.Name)
		}
	case *api.Board:
		fmt.Fprintln(w, "ID\tNAME")
		fmt.Fprintf(w, "%s\t%s\n", v.ID, v.Name)
	case []api.Board:
		fmt.Fprintln(w, "ID\tNAME")
		for _, b := range v {
			fmt.Fprintf(w, "%s\t%s\n", b.ID, b.Name)
		}
	case *api.Sprint:
		fmt.Fprintln(w, "ID\tNAME")
		fmt.Fprintf(w, "%s\t%s\n", v.ID, v.Name)
	case *api.Dashboard:
		fmt.Fprintln(w, "ID\tNAME")
		fmt.Fprintf(w, "%s\t%s\n", v.ID, v.Name)
	case []api.Dashboard:
		fmt.Fprintln(w, "ID\tNAME")
		for _, d := range v {
			fmt.Fprintf(w, "%s\t%s\n", d.ID, d.Name)
		}
	case *api.Tag:
		fmt.Fprintln(w, "ID\tNAME")
		fmt.Fprintf(w, "%s\t%s\n", v.ID, v.Name)
	case []api.Tag:
		fmt.Fprintln(w, "ID\tNAME")
		for _, t := range v {
			fmt.Fprintf(w, "%s\t%s\n", t.ID, t.Name)
		}
	case *api.SavedSearch:
		fmt.Fprintln(w, "ID\tNAME\tQUERY")
		fmt.Fprintf(w, "%s\t%s\t%s\n", v.ID, v.Name, v.Query)
	case []api.SavedSearch:
		fmt.Fprintln(w, "ID\tNAME\tQUERY")
		for _, s := range v {
			fmt.Fprintf(w, "%s\t%s\t%s\n", s.ID, s.Name, s.Query)
		}
	case *api.Organization:
		fmt.Fprintln(w, "ID\tNAME")
		fmt.Fprintf(w, "%s\t%s\n", v.ID, v.Name)
	case []api.Organization:
		fmt.Fprintln(w, "ID\tNAME")
		for _, o := range v {
			fmt.Fprintf(w, "%s\t%s\n", o.ID, o.Name)
		}
	case []api.App:
		fmt.Fprintln(w, "ID\tNAME\tTITLE\tVERSION")
		for _, a := range v {
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", a.ID, a.Name, a.Title, a.Version)
		}
	case *api.Canvas:
		fmt.Fprintln(w, "ID\tNAME")
		fmt.Fprintf(w, "%s\t%s\n", v.ID, v.Name)
	case []api.Canvas:
		fmt.Fprintln(w, "ID\tNAME")
		for _, c := range v {
			fmt.Fprintf(w, "%s\t%s\n", c.ID, c.Name)
		}
	case []api.RemovedDocument:
		fmt.Fprintln(w, "ID\tSUMMARY")
		for _, d := range v {
			fmt.Fprintf(w, "%s\t%s\n", d.IDReadable, d.Summary)
		}
	case []api.GanttChart:
		fmt.Fprintln(w, "ID\tNAME")
		for _, g := range v {
			fmt.Fprintf(w, "%s\t%s\n", g.ID, g.Name)
		}
	case []api.ProjectTemplate:
		fmt.Fprintln(w, "ID\tNAME\tKEY")
		for _, t := range v {
			fmt.Fprintf(w, "%s\t%s\t%s\n", t.ID, t.Name, t.ShortName)
		}
	case []api.IssueDraft:
		fmt.Fprintln(w, "ID\tSUMMARY")
		for _, d := range v {
			fmt.Fprintf(w, "%s\t%s\n", d.ID, d.Summary)
		}
	case []api.IssueAttachment:
		fmt.Fprintln(w, "ID\tNAME\tURL")
		for _, a := range v {
			fmt.Fprintf(w, "%s\t%s\t%s\n", a.ID, a.Name, a.URL)
		}
	case []api.ArticleAttachment:
		fmt.Fprintln(w, "ID\tNAME\tURL")
		for _, a := range v {
			fmt.Fprintf(w, "%s\t%s\t%s\n", a.ID, a.Name, a.URL)
		}
	case *api.WorkItem:
		fmt.Fprintln(w, "ID\tTEXT")
		fmt.Fprintf(w, "%s\t%s\n", v.ID, v.Text)
	case []api.IssueLink:
		fmt.Fprintln(w, "ID\tTYPE\tISSUES")
		for _, l := range v {
			typeName := ""
			if l.LinkType != nil {
				typeName = l.LinkType.Name
			}
			ids := make([]string, len(l.Issues))
			for i, is := range l.Issues {
				ids[i] = is.IDReadable
			}
			fmt.Fprintf(w, "%s\t%s\t%s\n", l.ID, typeName, joinStrings(ids))
		}
	case []api.LinkType:
		fmt.Fprintln(w, "ID\tNAME\tSOURCE-TO-TARGET\tTARGET-TO-SOURCE")
		for _, lt := range v {
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", lt.ID, lt.Name, lt.SourceToTarget, lt.TargetToSource)
		}
	case []api.ProjectCustomField:
		fmt.Fprintln(w, "ID\tFIELD")
		for _, f := range v {
			name := ""
			if f.Field != nil {
				name = f.Field.Name
			}
			fmt.Fprintf(w, "%s\t%s\n", f.ID, name)
		}
	default:
		return printJSON(out, data)
	}
	return nil
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
