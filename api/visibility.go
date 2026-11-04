package api

import (
	"cmp"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strings"
)

const visibilityFields = "visibility($type,permittedGroups(id,name),permittedUsers(id,login))"

// VisibilityChange is a visibility to write: public, or limited to the listed groups and users.
// It always sends both axes, because YouTrack keeps an omitted axis as it was.
type VisibilityChange struct {
	Public   bool
	GroupIDs []string
	UserIDs  []string
}

func (v VisibilityChange) MarshalJSON() ([]byte, error) {
	if v.Public {
		return json.Marshal(map[string]any{"$type": "UnlimitedVisibility"})
	}
	return json.Marshal(map[string]any{
		"$type":           "LimitedVisibility",
		"permittedGroups": idRefs(v.GroupIDs),
		"permittedUsers":  idRefs(v.UserIDs),
	})
}

func idRefs(ids []string) []map[string]string {
	refs := make([]map[string]string, len(ids))
	for i, id := range ids {
		refs[i] = map[string]string{"id": id}
	}
	return refs
}

// Matches reports whether the visibility read back from the server is the one that was written.
// YouTrack stores a limited visibility with no groups and no users as public.
func (v VisibilityChange) Matches(got *Visibility) bool {
	return v.MatchesAllowing(got, "")
}

// MatchesAllowing is Matches that also accepts allowedUserID among the permitted users, because
// YouTrack may add an issue's reporter to its visibility.
func (v VisibilityChange) MatchesAllowing(got *Visibility, allowedUserID string) bool {
	if v.Public || len(v.GroupIDs)+len(v.UserIDs) == 0 {
		return isPublic(got)
	}
	if isPublic(got) {
		return false
	}
	groups := make([]string, len(got.PermittedGroups))
	for i, g := range got.PermittedGroups {
		groups[i] = g.ID
	}
	users := make([]string, 0, len(got.PermittedUsers))
	for i := range got.PermittedUsers {
		if id := got.PermittedUsers[i].ID; id != allowedUserID || slices.Contains(v.UserIDs, id) {
			users = append(users, id)
		}
	}
	return sameSet(v.GroupIDs, groups) && sameSet(v.UserIDs, users)
}

func isPublic(v *Visibility) bool {
	return v == nil || v.Type != "LimitedVisibility" || (len(v.PermittedGroups) == 0 && len(v.PermittedUsers) == 0)
}

func sameSet(a, b []string) bool {
	a, b = slices.Clone(a), slices.Clone(b)
	slices.Sort(a)
	slices.Sort(b)
	return slices.Equal(slices.Compact(a), slices.Compact(b))
}

// DescribeVisibility renders a visibility as "public" or "groups: A, B; users: none".
func DescribeVisibility(v *Visibility) string {
	if isPublic(v) {
		return "public"
	}
	groups := make([]string, len(v.PermittedGroups))
	for i, g := range v.PermittedGroups {
		groups[i] = cmp.Or(g.Name, g.ID)
	}
	users := make([]string, len(v.PermittedUsers))
	for i := range v.PermittedUsers {
		users[i] = cmp.Or(v.PermittedUsers[i].Login, v.PermittedUsers[i].ID)
	}
	return fmt.Sprintf("groups: %s; users: %s", listOrNone(groups), listOrNone(users))
}

func listOrNone(names []string) string {
	if len(names) == 0 {
		return "none"
	}
	return strings.Join(names, ", ")
}

// ResolveVisibilityUserID resolves an exact login or ID using only basic user data.
func (c *Client) ResolveVisibilityUserID(identifier string) (string, error) {
	return c.resolveVisibilityID(fmt.Sprintf(pathUser, url.PathEscape(identifier)), "user", identifier)
}

// ResolveVisibilityGroupID resolves an exact group name or ID without a list search.
func (c *Client) ResolveVisibilityGroupID(identifier string) (string, error) {
	return c.resolveVisibilityID(fmt.Sprintf(pathGroup, url.PathEscape(identifier)), "group", identifier)
}

func (c *Client) resolveVisibilityID(path, kind, identifier string) (string, error) {
	var result struct {
		ID string `json:"id"`
	}
	if err := c.Get(path+"?fields=id", &result); err != nil {
		var apiErr *APIError
		if errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusNotFound {
			return "", fmt.Errorf("%s %q: %w", kind, identifier, ErrNotFound)
		}
		return "", err
	}
	if result.ID == "" {
		return "", fmt.Errorf("%s %q returned no ID", kind, identifier)
	}
	return result.ID, nil
}
