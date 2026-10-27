package api

import (
	"fmt"
	"net/url"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const issueFieldsWithoutDescription = "id,idReadable,summary,project(id,shortName,name),reporter(id,login,fullName),resolved,created,updated,updater(id,login,fullName)"
const issueFields = issueFieldsWithoutDescription + ",description"
const issueDetailCustomFields = ",customFields(name,value(name,login,text,presentation))"
const issueDetailFields = issueFields + issueDetailCustomFields

// issueCustomFieldValues fetches custom fields with the sub-fields that identify each value type: enum/state/version/build
// (name), user (login, fullName), text (text), period (presentation); number and date values are scalars.
// projectCustomField.emptyFieldText is what the UI shows for an unset value ("Unassigned", "No Fix versions").
const issueCustomFieldValues = "customFields(name,value(name,login,fullName,text,presentation),projectCustomField(emptyFieldText))"
const commentFields = "id,text,author(id,login,fullName),created,updated,deleted,visibility"
const draftFields = "id,summary,description,created,fields(id,name,$type)"

// AddIssueRequest is the body for creating an issue.
type AddIssueRequest struct {
	Summary     string   `json:"summary"`
	Description string   `json:"description,omitempty"`
	Project     *Project `json:"project"`
	Reporter    *User    `json:"reporter,omitempty"`
}

// UpdateIssueRequest contains only explicitly selected issue fields.
type UpdateIssueRequest struct {
	Summary     *string `json:"summary,omitempty"`
	Description *string `json:"description,omitempty"`
}

// AddCommentRequest is the body for creating a comment.
type AddCommentRequest struct {
	Text   string `json:"text"`
	Author *User  `json:"author,omitempty"`
}

// ApplyCommandRequest is the body for the /api/commands endpoint.
type ApplyCommandRequest struct {
	Query  string  `json:"query"`
	Issues []Issue `json:"issues"`
}

// readableIssueIDRe matches readable issue IDs such as FOO-42; database IDs such as 2-15 start with a digit.
var readableIssueIDRe = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_]*-[0-9]+$`)

// commandIssues builds the issues list for /api/commands and /api/commands/assist.
// Both endpoints accept idReadable, so readable IDs are sent as-is instead of costing one lookup request each;
// any other form is resolved to its database ID first, so a bad ID still fails before the command runs.
func (c *Client) commandIssues(issueIDs []string) ([]Issue, error) {
	issues := make([]Issue, len(issueIDs))
	for i, id := range issueIDs {
		if readableIssueIDRe.MatchString(id) {
			issues[i] = Issue{IDReadable: id}
			continue
		}
		dbID, err := c.issueDBID(id)
		if err != nil {
			return nil, fmt.Errorf("resolve issue %s: %w", id, err)
		}
		issues[i] = Issue{ID: dbID}
	}
	return issues, nil
}

// issueDBID resolves an issue ID in any form to its database ID, fetching nothing else.
func (c *Client) issueDBID(id string) (string, error) {
	var result struct {
		ID string `json:"id"`
	}
	if err := c.Get(fmt.Sprintf(pathIssue+"?fields=id", url.PathEscape(id)), &result); err != nil {
		return "", err
	}
	return result.ID, nil
}

var userValueRe = regexp.MustCompile(`(?i)\b(assignee|reporter)\s+(\S*[.\-@]\S+)`)

// WrapUserValues braces logins containing dots/hyphens for YouTrack's command parser.
// Example: "assignee alice.qa" → "assignee {alice.qa}"
// Logins that are already braced are left unchanged.
func WrapUserValues(query string) string {
	return userValueRe.ReplaceAllStringFunc(query, func(match string) string {
		sub := userValueRe.FindStringSubmatch(match)
		if len(sub) < 3 {
			return match
		}
		keyword, login := sub[1], sub[2]
		if len(login) > 0 && login[0] == '{' {
			return match
		}
		return keyword + " {" + login + "}"
	})
}

func (c *Client) GetIssueDetail(id string) (*Issue, error) {
	var result Issue
	path := fmt.Sprintf(pathIssue+"?fields=%s", url.PathEscape(id), issueDetailFields)
	err := c.Get(path, &result)
	return &result, err
}

func (c *Client) GetIssue(id string) (*Issue, error) {
	return c.GetIssueProjection(id, issueFields)
}

// IssueViewFields is the default issue view projection; detail adds custom field values for plain output.
func IssueViewFields(detail, description bool) string {
	fields := issueFieldsWithoutDescription
	if description {
		fields = issueFields
	}
	if detail {
		fields += issueDetailCustomFields
	}
	return fields
}

// GetIssueProjection fetches an issue with exactly the given fields, plus any Client.Fields.
func (c *Client) GetIssueProjection(id, fields string) (*Issue, error) {
	var result Issue
	path := fmt.Sprintf(pathIssue+"?fields=%s", url.PathEscape(id), fields)
	err := c.Get(path, &result)
	return &result, err
}

func (c *Client) ListIssues(query string, limit int) ([]Issue, error) {
	return c.ListIssuesPage(query, limit, 0)
}

// ListIssuesPage returns one page of issues starting at offset.
func (c *Client) ListIssuesPage(query string, limit, offset int) ([]Issue, error) {
	return c.ListIssuesPageWithCustomFields(query, limit, offset, nil)
}

// ListIssuesPageWithCustomFields is ListIssuesPage that also returns the named custom fields of each issue.
// Names go in the repeatable customFields= parameter, which the server matches case-insensitively and uses to
// return only those fields; a name no matched project has is simply absent from customFields.
func (c *Client) ListIssuesPageWithCustomFields(query string, limit, offset int, customFields []string) ([]Issue, error) {
	var result []Issue
	fields := issueFields
	var filter strings.Builder
	if len(customFields) > 0 {
		fields += "," + issueCustomFieldValues
		for _, name := range customFields {
			filter.WriteString("&customFields=" + url.QueryEscape(name))
		}
	}
	path := pathIssues + fmt.Sprintf("?query=%s&$top=%d&$skip=%d&fields=%s%s",
		url.QueryEscape(query), limit, offset, fields, filter.String())
	err := c.Get(path, &result)
	return result, err
}

func (c *Client) AddIssue(req AddIssueRequest) (*Issue, error) {
	var result Issue
	err := c.Post(pathIssues+"?fields="+issueFields, req, &result)
	return &result, err
}

func (c *Client) UpdateIssue(id string, req UpdateIssueRequest) (*Issue, error) {
	var result Issue
	path := fmt.Sprintf(pathIssue+"?fields=%s", url.PathEscape(id), issueFields)
	err := c.Post(path, req, &result)
	return &result, err
}

func (c *Client) DeleteIssue(id string) error {
	return c.Delete(fmt.Sprintf(pathIssue, url.PathEscape(id)))
}

// ListComments returns every comment on the issue, paging with $skip; without
// $top the server returns at most 42 entries.
func (c *Client) ListComments(issueID string) ([]Comment, error) {
	const pageSize = 500
	var result []Comment
	for skip := 0; ; skip += pageSize {
		var page []Comment
		path := fmt.Sprintf(pathIssueComments+"?fields=%s&$top=%d&$skip=%d", url.PathEscape(issueID), commentFields, pageSize, skip)
		if err := c.Get(path, &page); err != nil {
			return nil, err
		}
		result = append(result, page...)
		if len(page) < pageSize {
			return result, nil
		}
	}
}

func (c *Client) AddComment(issueID string, req AddCommentRequest) (*Comment, error) {
	var result Comment
	path := fmt.Sprintf(pathIssueComments+"?fields=%s", url.PathEscape(issueID), commentFields)
	err := c.Post(path, req, &result)
	return &result, err
}

func (c *Client) DeleteComment(issueID, commentID string) error {
	return c.Delete(fmt.Sprintf(pathIssueComment,
		url.PathEscape(issueID), url.PathEscape(commentID)))
}

func (c *Client) GetIssueCount(query string) (int, error) {
	const pollTimeout = 5 * time.Second
	const pollInterval = 200 * time.Millisecond

	deadline := time.Now().Add(pollTimeout)
	for {
		var result struct {
			Count int `json:"count"`
		}
		if err := c.postQuery(pathIssuesGetterCount+"?fields=count",
			map[string]any{"query": query}, &result); err != nil {
			return 0, err
		}
		if result.Count >= 0 {
			return result.Count, nil
		}
		if time.Now().After(deadline) {
			return 0, fmt.Errorf("issue count timed out after %v (query: %q): YouTrack async computation did not complete", pollTimeout, query)
		}
		if err := c.wait(pollInterval); err != nil {
			return 0, err
		}
	}
}

func (c *Client) GetIssueLinkTypes() ([]IssueLinkType, error) {
	var result []IssueLinkType
	err := c.Get(pathIssueLinkTypes+"?fields=id,name,sourceToTarget,targetToSource,directed&$top=500", &result)
	return result, err
}

func (c *Client) GetIssueLinkType(id string) (*IssueLinkType, error) {
	types, err := c.GetIssueLinkTypes()
	if err != nil {
		return nil, err
	}
	for i := range types {
		if types[i].ID == id {
			return &types[i], nil
		}
	}
	return nil, fmt.Errorf("link type %q not found", id)
}

func (c *Client) AddIssueLinkType(name, sourceToTarget, targetToSource string, directed, aggregation bool) (*IssueLinkType, error) {
	var result IssueLinkType
	err := c.Post(pathIssueLinkTypes+"?fields=id,name,sourceToTarget,targetToSource,directed", map[string]any{
		"name":           name,
		"sourceToTarget": sourceToTarget,
		"targetToSource": targetToSource,
		"directed":       directed,
		"aggregation":    aggregation,
	}, &result)
	return &result, err
}

func (c *Client) GetLinkedIssues(issueID string) ([]IssueLink, error) {
	var result []IssueLink
	path := fmt.Sprintf(pathIssueLinks+"?fields=id,direction,linkType(id,name,sourceToTarget,targetToSource),issues(id,idReadable,summary)&$top=500", url.PathEscape(issueID))
	err := c.Get(path, &result)
	return result, err
}

func (c *Client) LinkIssues(issueID, targetID, linkTypeID string) error {
	if linkTypeID == "" {
		return fmt.Errorf("link type ID must not be empty")
	}
	// Body requires the opaque internal ID; URL path segment accepts the readable form.
	targetDBID, err := c.issueDBID(targetID)
	if err != nil {
		return fmt.Errorf("resolve target issue %s: %w", targetID, err)
	}
	// For directed link types, append 's' (OUTWARD) unless the caller already
	// included a direction marker ('s' or 't'). Non-directed link types use the ID as-is.
	linkID := linkTypeID
	if last := linkTypeID[len(linkTypeID)-1]; last != 's' && last != 't' {
		lt, err := c.GetIssueLinkType(linkTypeID)
		if err == nil && lt.Directed {
			linkID = linkTypeID + "s"
		}
	}
	path := fmt.Sprintf(pathIssueLinkTypeIssues, url.PathEscape(issueID), url.PathEscape(linkID))
	return c.Post(path, map[string]any{"id": targetDBID}, nil)
}

// RemoveIssueLink removes all linked issues from the given link on issueID.
// linkID is the direction-type ID returned by GetLinkedIssues (e.g. "167-0s").
// The YouTrack REST API does not support DELETE on the link itself; each linked
// issue must be removed individually via DELETE /issues/{id}/links/{linkId}/issues/{targetId}.
func (c *Client) RemoveIssueLink(issueID, linkID string) error {
	// Fetch the link to discover which issues are attached to it.
	var link IssueLink
	path := fmt.Sprintf(pathIssueLink+"?fields=id,issues(id)&$top=500",
		url.PathEscape(issueID), url.PathEscape(linkID))
	if err := c.Get(path, &link); err != nil {
		return err
	}
	for i := range link.Issues {
		target := &link.Issues[i]
		delPath := fmt.Sprintf(pathIssueLinkTypeIssue,
			url.PathEscape(issueID), url.PathEscape(linkID), url.PathEscape(target.ID))
		if err := c.Delete(delPath); err != nil {
			return err
		}
	}
	return nil
}

func (c *Client) TagIssue(issueID, tagID string) error {
	path := fmt.Sprintf(pathIssueTags, url.PathEscape(issueID))
	return c.Post(path, map[string]any{"id": tagID}, nil)
}

func (c *Client) UntagIssue(issueID, tagID string) error {
	return c.Delete(fmt.Sprintf(pathIssueTag, url.PathEscape(issueID), url.PathEscape(tagID)))
}

func (c *Client) ListIssueTags(issueID string) ([]Tag, error) {
	var result []Tag
	path := fmt.Sprintf(pathIssueTags+"?fields=id,name,owner(id,login,fullName),color,untagOnResolve&$top=500", url.PathEscape(issueID))
	err := c.Get(path, &result)
	return result, err
}

func (c *Client) GetIssueTag(issueID, tagID string) (*Tag, error) {
	var result Tag
	path := fmt.Sprintf(pathIssueTag+"?fields=id,name,owner(id,login,fullName),color,untagOnResolve", url.PathEscape(issueID), url.PathEscape(tagID))
	err := c.Get(path, &result)
	return &result, err
}

func (c *Client) GetIssueAttachments(issueID string) ([]IssueAttachment, error) {
	var result []IssueAttachment
	path := fmt.Sprintf(pathIssueAttachments+"?fields=id,name,url,mimeType,created&$top=500", url.PathEscape(issueID))
	err := c.Get(path, &result)
	return result, err
}

func (c *Client) AddIssueAttachment(issueID, filePath string) (*IssueAttachment, error) {
	var result []IssueAttachment
	path := fmt.Sprintf(pathIssueAttachments+"?fields=id,name,url,mimeType,created", url.PathEscape(issueID))
	if err := c.Multipart(path, filePath, filepath.Base(filePath), "application/octet-stream", &result); err != nil {
		return nil, err
	}
	if len(result) == 0 {
		return nil, fmt.Errorf("attachment upload returned empty response")
	}
	return &result[0], nil
}

func (c *Client) DeleteIssueAttachment(issueID, attachmentID string) error {
	return c.Delete(fmt.Sprintf(pathIssueAttachment,
		url.PathEscape(issueID), url.PathEscape(attachmentID)))
}

func (c *Client) AddWorkItem(issueID string, minutes int, text, typeID string) (*WorkItem, error) {
	var result WorkItem
	body := map[string]any{
		"duration": map[string]any{"minutes": minutes},
		"date":     time.Now().UnixMilli(),
	}
	if text != "" {
		body["text"] = text
	}
	if typeID != "" {
		body["type"] = map[string]any{"id": typeID}
	}
	path := fmt.Sprintf(pathIssueWorkItems+"?fields=id,text,date,duration(minutes),author(id,name)", url.PathEscape(issueID))
	err := c.Post(path, body, &result)
	return &result, err
}

func (c *Client) ListWorkItems(issueID string) ([]WorkItem, error) {
	var result []WorkItem
	path := fmt.Sprintf(pathIssueWorkItems+"?fields=id,text,date,duration(minutes),author(id,login,fullName)&$top=500", url.PathEscape(issueID))
	err := c.Get(path, &result)
	return result, err
}

func (c *Client) GetCustomFieldValue(issueID, fieldName string) (*IssueField, error) {
	var fields []IssueField
	path := fmt.Sprintf(pathIssueFields+"?fields=id,name,value(name,presentation,text,login,minutes)&$top=500", url.PathEscape(issueID))
	if err := c.Get(path, &fields); err != nil {
		return nil, err
	}
	for i := range fields {
		if fields[i].Name == fieldName {
			return &fields[i], nil
		}
	}
	return nil, fmt.Errorf("field '%s' not found on issue %s", fieldName, issueID)
}

func (c *Client) SetCustomField(issueID, fieldName, fieldValue, fieldType string) error {
	var fields []struct {
		ID   string `json:"id"`
		Name string `json:"name"`
		Type string `json:"$type"`
	}
	path := fmt.Sprintf(pathIssueFields+"?fields=id,name,$type&$top=500", url.PathEscape(issueID))
	if err := c.Get(path, &fields); err != nil {
		return err
	}
	var fieldID, issueFieldType string
	for _, f := range fields {
		if f.Name == fieldName {
			fieldID, issueFieldType = f.ID, f.Type
			break
		}
	}
	if fieldID == "" {
		return fmt.Errorf("field '%s' not found on issue %s", fieldName, issueID)
	}
	var body map[string]any
	switch fieldType {
	case "period":
		body = map[string]any{"value": map[string]any{"presentation": fieldValue}}
	case "text":
		body = map[string]any{"value": map[string]any{"text": fieldValue}}
	case "user":
		user, err := c.GetUser(fieldValue)
		if err != nil {
			return err
		}
		body = map[string]any{"value": map[string]any{"id": user.ID}}
	case "int":
		n, err := strconv.Atoi(fieldValue)
		if err != nil {
			return fmt.Errorf("invalid integer value '%s': %w", fieldValue, err)
		}
		body = map[string]any{"value": n}
	case "string":
		body = map[string]any{"value": fieldValue}
	default:
		body = map[string]any{"value": map[string]any{"name": fieldValue}}
	}
	if isMultiValueField(issueFieldType) {
		// Multi-value fields (e.g. MultiEnumIssueCustomField) take an array; a single object is rejected with HTTP 500.
		body["value"] = []any{body["value"]}
	}
	return c.Post(fmt.Sprintf(pathIssueField, url.PathEscape(issueID), url.PathEscape(fieldID)), body, nil)
}

func (c *Client) UpdateComment(issueID, commentID, text string) (*Comment, error) {
	var result Comment
	path := fmt.Sprintf(pathIssueComment+"?fields=%s", url.PathEscape(issueID), url.PathEscape(commentID), commentFields)
	err := c.Post(path, map[string]any{"text": text}, &result)
	return &result, err
}

func (c *Client) UpdateCommentVisibility(issueID, commentID string, groupIDs, userIDs []string) error {
	groups := make([]map[string]any, len(groupIDs))
	for i, id := range groupIDs {
		groups[i] = map[string]any{"id": id}
	}
	users := make([]map[string]any, len(userIDs))
	for i, id := range userIDs {
		users[i] = map[string]any{"id": id}
	}
	body := map[string]any{
		"visibility": map[string]any{
			"$type":           "LimitedVisibility",
			"permittedGroups": groups,
			"permittedUsers":  users,
		},
	}
	return c.Post(fmt.Sprintf(pathIssueComment, url.PathEscape(issueID), url.PathEscape(commentID)), body, nil)
}

func (c *Client) PinComment(issueID, commentID string, pinned bool) error {
	return c.Post(fmt.Sprintf(pathIssueComment, url.PathEscape(issueID), url.PathEscape(commentID)),
		map[string]any{"pinned": pinned}, nil)
}

func (c *Client) UpdateIssueVisibility(issueID string, groupIDs, userIDs []string) error {
	groups := make([]map[string]any, len(groupIDs))
	for i, id := range groupIDs {
		groups[i] = map[string]any{"id": id}
	}
	users := make([]map[string]any, len(userIDs))
	for i, id := range userIDs {
		users[i] = map[string]any{"id": id}
	}
	return c.Post(fmt.Sprintf(pathIssue, url.PathEscape(issueID)), map[string]any{
		"visibility": map[string]any{
			"$type":           "LimitedVisibility",
			"permittedGroups": groups,
			"permittedUsers":  users,
		},
	}, nil)
}

func (c *Client) GetCommandAssist(query string, issueIDs []string) (map[string]any, error) {
	issues, err := c.commandIssues(issueIDs)
	if err != nil {
		return nil, err
	}
	var result map[string]any
	err = c.postQuery(pathCommandsAssist+"?fields=commands(error,description),query,suggestions(id,option,description,completionStart,completionEnd,prefix,suffix,shortName,fullName,group)",
		map[string]any{"query": query, "issues": issues}, &result)
	return result, err
}

func (c *Client) ListIssueCommentReactions(issueID, commentID string) ([]Reaction, error) {
	var result []Reaction
	path := fmt.Sprintf(pathIssueCommentReactions+"?fields=id,reaction,author(id,login,fullName)&$top=500",
		url.PathEscape(issueID), url.PathEscape(commentID))
	err := c.Get(path, &result)
	return result, err
}

func (c *Client) AddReaction(issueID, commentID, reaction string) error {
	path := fmt.Sprintf(pathIssueCommentReactions, url.PathEscape(issueID), url.PathEscape(commentID))
	return c.Post(path, map[string]any{"reaction": reaction}, nil)
}

func (c *Client) DeleteReaction(issueID, commentID, reactionID string) error {
	return c.Delete(fmt.Sprintf(pathIssueCommentReaction,
		url.PathEscape(issueID), url.PathEscape(commentID), url.PathEscape(reactionID)))
}

func (c *Client) CreateDraft(projectID, summary, description string) (*IssueDraft, error) {
	body := map[string]any{
		"project": map[string]string{"id": projectID},
	}
	if summary != "" {
		body["summary"] = summary
	}
	if description != "" {
		body["description"] = description
	}
	var result IssueDraft
	path := fmt.Sprintf(pathUserDrafts, "me") + "?fields=" + draftFields
	err := c.Post(path, body, &result)
	return &result, err
}

func (c *Client) GetDraft(draftID string) (*IssueDraft, error) {
	var result IssueDraft
	path := fmt.Sprintf(pathUserDraft+"?fields=%s", "me", url.PathEscape(draftID), draftFields)
	err := c.Get(path, &result)
	return &result, err
}

// UpdateDraft sets a non-empty summary, and the description when it is not nil; an empty description clears it.
func (c *Client) UpdateDraft(draftID, summary string, description *string) (*IssueDraft, error) {
	body := map[string]any{}
	if summary != "" {
		body["summary"] = summary
	}
	if description != nil {
		body["description"] = *description
	}
	var result IssueDraft
	path := fmt.Sprintf(pathUserDraft+"?fields=%s", "me", url.PathEscape(draftID), draftFields)
	err := c.Post(path, body, &result)
	return &result, err
}

// UpdateDraftField sets a single custom field on a draft by field ID.
// fieldType is the YouTrack $type of the field (e.g. "StateIssueCustomField").
// valueName is the display name of the desired value (e.g. "Fixed").
func (c *Client) UpdateDraftField(draftID, fieldID, fieldType, valueName string) error {
	valueType := draftFieldValueType(fieldType)
	body := map[string]any{
		"$type": fieldType,
	}
	value := map[string]string{"name": valueName}
	if valueType != "" {
		value["$type"] = valueType
	}
	body["value"] = value
	if isMultiValueField(fieldType) {
		body["value"] = []map[string]string{value}
	}
	path := fmt.Sprintf(pathUserDraftField, "me", url.PathEscape(draftID), url.PathEscape(fieldID))
	return c.Post(path, body, nil)
}

// isMultiValueField reports whether an issue custom field $type holds an array of values.
func isMultiValueField(fieldType string) bool {
	return strings.HasPrefix(fieldType, "Multi")
}

// draftFieldValueType maps a field $type to its value $type.
func draftFieldValueType(fieldType string) string {
	switch fieldType {
	case "StateIssueCustomField":
		return "StateBundleElement"
	case "SingleEnumIssueCustomField", "MultiEnumIssueCustomField":
		return "EnumBundleElement"
	case "SingleVersionIssueCustomField", "MultiVersionIssueCustomField":
		return "VersionBundleElement"
	case "SingleBuildIssueCustomField", "MultiBuildIssueCustomField":
		return "BuildBundleElement"
	case "SingleOwnedIssueCustomField", "MultiOwnedIssueCustomField":
		return "OwnedBundleElement"
	default:
		return ""
	}
}

func (c *Client) DeleteDraft(draftID string) error {
	path := fmt.Sprintf(pathUserDraft, "me", url.PathEscape(draftID))
	return c.Delete(path)
}

func (c *Client) SubmitDraft(draftID string) (*Issue, error) {
	var result Issue
	path := pathIssues + "?draftId=" + url.QueryEscape(draftID) + "&fields=" + issueFields
	err := c.Post(path, map[string]any{}, &result)
	return &result, err
}

// ApplyCommand POSTs the command to /api/commands for the given issues.
func (c *Client) ApplyCommand(query string, issueIDs []string) error {
	issues, err := c.commandIssues(issueIDs)
	if err != nil {
		return err
	}
	return c.Post(pathCommands, ApplyCommandRequest{
		Query:  WrapUserValues(query),
		Issues: issues,
	}, nil)
}

func (c *Client) CreateIssueDraft(projectShortName, summary string) (*IssueDraft, error) {
	project, err := c.GetProject(projectShortName)
	if err != nil {
		return nil, err
	}
	var result IssueDraft
	path := fmt.Sprintf(pathIssueDrafts+"?fields=id,summary", "me")
	err = c.Post(path, map[string]any{
		"project": map[string]any{"id": project.ID},
		"summary": summary,
	}, &result)
	return &result, err
}

// WatchIssue and the other watcher/voter methods pass the ID straight into the path,
// which accepts the readable form, so no lookup request is needed first.
// IssueWatchers exposes the current user's watch state as hasStar; YouTrack ignores unknown body fields, so a wrong name is a silent no-op.
func (c *Client) WatchIssue(id string) error {
	return c.setWatching(id, true)
}

func (c *Client) UnwatchIssue(id string) error {
	return c.setWatching(id, false)
}

func (c *Client) setWatching(id string, watching bool) error {
	return c.Post(fmt.Sprintf(pathIssueWatchers, url.PathEscape(id)), map[string]any{"hasStar": watching}, nil)
}

func (c *Client) VoteIssue(id string) error {
	return c.setVote(id, true)
}

func (c *Client) UnvoteIssue(id string) error {
	return c.setVote(id, false)
}

func (c *Client) setVote(id string, vote bool) error {
	return c.Post(fmt.Sprintf(pathIssueVoters, url.PathEscape(id)), map[string]any{"hasVote": vote}, nil)
}

// ListRecentComments returns the last `top` comments; YouTrack serves them oldest-first, so fetch all and keep the tail.
func (c *Client) ListRecentComments(issueID string, top int) ([]Comment, error) {
	result, err := c.ListComments(issueID)
	if err != nil {
		return nil, err
	}
	if top >= 0 && len(result) > top {
		result = result[len(result)-top:]
	}
	return result, nil
}

func (c *Client) GetComment(issueID, commentID string) (*Comment, error) {
	var result Comment
	path := fmt.Sprintf(pathIssueComment+"?fields=%s", url.PathEscape(issueID), url.PathEscape(commentID), commentFields)
	err := c.Get(path, &result)
	return &result, err
}

func (c *Client) GetWorkItem(issueID, workItemID string) (*WorkItem, error) {
	var result WorkItem
	path := fmt.Sprintf(pathIssueWorkItem+"?fields=id,text,date,duration(minutes),author(id,login,fullName)", url.PathEscape(issueID), url.PathEscape(workItemID))
	err := c.Get(path, &result)
	return &result, err
}

func (c *Client) UpdateWorkItem(issueID, workItemID string, minutes int, text string) (*WorkItem, error) {
	var result WorkItem
	body := map[string]any{}
	if minutes > 0 {
		body["duration"] = map[string]any{"minutes": minutes}
	}
	if text != "" {
		body["text"] = text
	}
	path := fmt.Sprintf(pathIssueWorkItem+"?fields=id,text,date,duration(minutes),author(id,login,fullName)", url.PathEscape(issueID), url.PathEscape(workItemID))
	err := c.Post(path, body, &result)
	return &result, err
}

func (c *Client) DeleteWorkItem(issueID, workItemID string) error {
	return c.Delete(fmt.Sprintf(pathIssueWorkItem, url.PathEscape(issueID), url.PathEscape(workItemID)))
}

func (c *Client) ListIssueActivities(issueID string) ([]map[string]any, error) {
	var result []map[string]any
	path := fmt.Sprintf(pathIssueActivities+"?fields=id,timestamp,author(id,login,fullName),target,added,removed,field(id,name)&categories=CommentsCategory,IssueResolvedCategory,CustomFieldCategory&$top=500", url.PathEscape(issueID))
	err := c.Get(path, &result)
	return result, err
}

func (c *Client) ListIssueSprints(issueID string) ([]Sprint, error) {
	var result []Sprint
	path := fmt.Sprintf(pathIssueSprints+"?fields=id,name,archived,isDefault&$top=500", url.PathEscape(issueID))
	err := c.Get(path, &result)
	return result, err
}

// vcsChangeFields is the field selector shared by every VCS change request.
// idReadable and title are pull-request-only; $type tells the two kinds apart.
const vcsChangeFields = "id,$type,date,fetched,files,text,version,urls,state,idReadable,title,author(id,login,fullName)"

func (c *Client) ListIssueVCSChanges(issueID string) ([]VCSChange, error) {
	var result []VCSChange
	path := fmt.Sprintf(pathIssueVCSChanges+"?fields="+vcsChangeFields+"&$top=500", url.PathEscape(issueID))
	err := c.Get(path, &result)
	return result, err
}

func (c *Client) GetIssueVCSChange(issueID, changeID string) (*VCSChange, error) {
	var result VCSChange
	path := fmt.Sprintf(pathIssueVCSChange+"?fields="+vcsChangeFields, url.PathEscape(issueID), url.PathEscape(changeID))
	if err := c.Get(path, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// AttachIssueVCSChange links a commit to an issue by its hash.
//
// The commit is not created here: YouTrack looks it up among the VCS
// integrations configured for the issue's project and answers 404 when none of
// them knows the hash. A Generic VCS integration is the exception — it accepts
// any hash and takes the commit message from text.
func (c *Client) AttachIssueVCSChange(issueID, version, text string, mute bool) (*VCSChange, error) {
	state := VCSChangeStateManuallyAttached
	body := VCSChange{Version: version, State: &state, Text: text}

	var result VCSChange
	path := fmt.Sprintf(pathIssueVCSChanges+"?fields="+vcsChangeFields, url.PathEscape(issueID)) + muteSuffix(mute)
	if err := c.Post(path, body, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// SetIssueVCSChangeState updates the state of a VCS change already on the issue.
// Detaching uses VCSChangeStateDetached; VCSChangeStateDefault re-attaches a
// change that an integration had attached and a user then detached.
//
// The request asks for no fields, matching what YouTrack's own web client sends.
// A field projection makes the server render the change after the update, and
// detaching often deletes it — YouTrack keeps a detached record only as a marker
// that stops an integration re-attaching the commit, which it needs only when the
// commit message names the issue, and never for a Generic VCS integration. Asking
// for fields therefore turns a successful detach into "HTTP 500: VcsChange[..]
// was removed".
func (c *Client) SetIssueVCSChangeState(issueID, changeID string, state int, mute bool) error {
	body := VCSChange{State: &state}
	path := fmt.Sprintf(pathIssueVCSChange, url.PathEscape(issueID), url.PathEscape(changeID))
	if mute {
		path += "?muteUpdateNotifications=true"
	}
	return c.Post(path, body, nil)
}

// DeleteIssueVCSChange removes a manually attached VCS change from an issue.
// YouTrack rejects the call for changes an integration attached; detach those
// with SetIssueVCSChangeState instead.
func (c *Client) DeleteIssueVCSChange(issueID, changeID string) error {
	return c.Delete(fmt.Sprintf(pathIssueVCSChange, url.PathEscape(issueID), url.PathEscape(changeID)))
}

// muteUpdateNotifications suppresses issue-change notifications; it needs the
// Apply Commands Silently permission in every affected project.
func muteSuffix(mute bool) string {
	if mute {
		return "&muteUpdateNotifications=true"
	}
	return ""
}
