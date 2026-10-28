package issue

import (
	"errors"
	"fmt"
	"github.com/JetBrains/youtrack-cli/api"
	"github.com/spf13/cobra"
	"net/http"
	"regexp"
	"strings"
)

// entityIDRe matches opaque IDs such as 4-1; they are passed through, because looking up a group by ID
// needs more permissions than setting visibility, and the read-back still verifies the result.
var entityIDRe = regexp.MustCompile(`^[0-9]+-[0-9]+$`)

func resolveGroupID(client *api.Client, nameOrID string) (string, error) {
	if entityIDRe.MatchString(nameOrID) {
		return nameOrID, nil
	}
	return client.ResolveVisibilityGroupID(nameOrID)
}

func resolveUserID(client *api.Client, loginOrID string) (string, error) {
	if entityIDRe.MatchString(loginOrID) {
		return loginOrID, nil
	}
	return client.ResolveVisibilityUserID(loginOrID)
}

// addVisibilityFlags adds the flags that set visibility in the same request that creates an entity.
func addVisibilityFlags(cmd *cobra.Command, entity string) {
	cmd.Flags().StringArray("visibility", nil, "Limit the "+entity+" to this group name or user login (repeatable; prefix group: or user: if a name is both)")
	cmd.Flags().Bool("public", false, "Make the "+entity+" visible to everyone who can see the project")
	cmd.MarkFlagsMutuallyExclusive("visibility", "public")
}

// visibilityFromFlags resolves --visibility and --public, or returns nil to keep the server default.
func visibilityFromFlags(cmd *cobra.Command, client *api.Client) (*api.VisibilityChange, error) {
	if public, _ := cmd.Flags().GetBool("public"); public {
		return &api.VisibilityChange{Public: true}, nil
	}
	if !cmd.Flags().Changed("visibility") {
		return nil, nil
	}
	names, _ := cmd.Flags().GetStringArray("visibility")
	names = nonEmpty(names)
	if len(names) == 0 {
		return nil, fmt.Errorf("--visibility needs a group name or user login; use --public for no restriction")
	}
	v := &api.VisibilityChange{}
	for _, name := range names {
		groupID, userID, err := resolvePrincipal(client, name)
		if err != nil {
			return nil, err
		}
		if groupID != "" {
			v.GroupIDs = append(v.GroupIDs, groupID)
		} else {
			v.UserIDs = append(v.UserIDs, userID)
		}
	}
	return v, nil
}

// resolvePrincipal finds the group or user a --visibility value names; exactly one of the IDs is set.
func resolvePrincipal(client *api.Client, name string) (groupID, userID string, err error) {
	if g, ok := strings.CutPrefix(name, "group:"); ok {
		id, err := resolveGroupID(client, g)
		return id, "", err
	}
	if u, ok := strings.CutPrefix(name, "user:"); ok {
		id, err := resolveUserID(client, u)
		return "", id, err
	}
	group, groupErr := client.ResolveVisibilityGroupID(name)
	user, userErr := client.ResolveVisibilityUserID(name)
	// Group reads require permissions that comment authors need not have. A forbidden
	// group lookup must not block a confirmed user match; all other failures remain errors.
	var apiErr *api.APIError
	if userErr == nil && errors.As(groupErr, &apiErr) && apiErr.StatusCode == http.StatusForbidden {
		groupErr = api.ErrNotFound
	}
	// Other lookup failures could hide an ambiguous name and must not guess.
	for _, err := range []error{groupErr, userErr} {
		if err != nil && !errors.Is(err, api.ErrNotFound) {
			return "", "", fmt.Errorf("look up %q: %w", name, err)
		}
	}
	switch {
	case groupErr == nil && userErr == nil:
		return "", "", fmt.Errorf("%q is both a group and a user login; pass group:%s or user:%s", name, name, name)
	case groupErr == nil:
		return group, "", nil
	case userErr == nil:
		return "", user, nil
	default:
		return "", "", fmt.Errorf("no group or user %q (%w; %w)", name, groupErr, userErr)
	}
}

// checkCreatedVisibility reports an error when a new entity did not get the requested visibility.
func checkCreatedVisibility(want *api.VisibilityChange, got *api.Visibility, allowedUserID, label string) error {
	if want == nil || want.MatchesAllowing(got, allowedUserID) {
		return nil
	}
	return fmt.Errorf("%s was created, but its visibility is %s instead of the one requested; fix it with the visibility command",
		label, api.DescribeVisibility(got))
}

func nonEmpty(values []string) []string {
	var result []string
	for _, v := range values {
		if v = strings.TrimSpace(v); v != "" {
			result = append(result, v)
		}
	}
	return result
}
