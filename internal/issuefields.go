package internal

import (
	"regexp"
	"strings"
)

// issueAttributes lists the documented attributes of the YouTrack Issue entity
// (https://www.jetbrains.com/help/youtrack/devportal/api-entity-Issue.html), keyed by lowercase name.
// A --fields entry naming one of them is an issue attribute; anything else is a custom field name.
var issueAttributes = func() map[string]string {
	names := []string{
		"id", "idReadable", "summary", "description", "wikifiedDescription",
		"project", "reporter", "updater", "draftOwner", "isDraft",
		"created", "updated", "resolved", "numberInProject",
		"commentsCount", "comments", "pinnedComments", "attachments",
		"links", "parent", "subtasks", "tags", "customFields",
		"visibility", "voters", "votes", "watchers", "externalIssue",
	}
	m := make(map[string]string, len(names))
	for _, n := range names {
		m[strings.ToLower(n)] = n
	}
	return m
}()

// IssueColumn is one column requested with `issue list --fields`.
type IssueColumn struct {
	Spec   string // attribute spec in canonical casing ("summary", "project.shortName"), or the custom field name as typed
	Custom bool   // true when Spec names a custom field rather than an issue attribute
}

// IssueColumns splits a --fields value into issue-attribute and custom-field columns, keeping the requested order.
// An entry is an attribute when its leading name (before any "." or "(") matches a documented Issue attribute,
// case-insensitively, so "Summary" and "summary" both mean the attribute. Every other entry is taken verbatim
// as a custom field name ("Priority", "state", "Fix versions"), matched case-insensitively like the YouTrack server does.
func IssueColumns(fields string) []IssueColumn {
	if strings.TrimSpace(fields) == "" {
		return nil
	}
	var cols []IssueColumn
	for _, part := range SplitFields(fields) {
		if part == "" {
			continue
		}
		end := strings.IndexAny(part, ".(")
		if end < 0 {
			end = len(part)
		}
		if canonical, ok := issueAttributes[strings.ToLower(part[:end])]; ok {
			cols = append(cols, IssueColumn{Spec: canonical + part[end:]})
			continue
		}
		cols = append(cols, IssueColumn{Spec: part, Custom: true})
	}
	return cols
}

// attributeNameRe matches names that could be an issue attribute; custom field names with spaces or punctuation cannot.
var attributeNameRe = regexp.MustCompile(`^[A-Za-z_$][A-Za-z0-9_$]*$`)

// IssueRequestFields returns the expanded fields= projection for cols. Attribute columns are always included; custom
// field names are too when they could also be an attribute the list above omits (the server ignores unknown names),
// so a --fields entry that worked as an attribute before custom-field columns existed keeps working.
func IssueRequestFields(cols []IssueColumn) string {
	var specs []string
	for _, c := range cols {
		if !c.Custom || attributeNameRe.MatchString(c.Spec) {
			specs = append(specs, c.Spec)
		}
	}
	return expandFields(strings.Join(specs, ","))
}

// IssueCustomFieldNames returns the custom field names requested in cols, in order.
func IssueCustomFieldNames(cols []IssueColumn) []string {
	var names []string
	for _, c := range cols {
		if c.Custom {
			names = append(names, c.Spec)
		}
	}
	return names
}
