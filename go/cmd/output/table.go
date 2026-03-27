package output

import (
	"fmt"
	"os"
	"text/tabwriter"
	"unicode/utf8"

	"youtrack-cli/api"
)

// PrintTable renders data as a human-readable table. Falls back to JSON for unknown types.
func PrintTable(data any) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
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
	default:
		return printJSON(data)
	}
	return nil
}

func projectKey(p *api.Project) string {
	if p == nil {
		return ""
	}
	return p.ShortName
}
