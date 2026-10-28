package api_test

import (
	"context"
	"github.com/JetBrains/youtrack-cli/api"
	"github.com/stretchr/testify/require"
	"os"
	"testing"
)

type visibilityFixture struct {
	issue string
	group api.Group
	user  api.User
}

func newVisibilityFixture(t *testing.T, c *api.Client) visibilityFixture {
	t.Helper()
	proj := createTempProject(t, c)
	issue, err := c.AddIssue(api.AddIssueRequest{Summary: "visibility test", Project: &api.Project{ID: proj.ID}})
	require.NoError(t, err)
	t.Cleanup(func() { _ = c.DeleteIssue(issue.IDReadable) })

	suffix := uniqueProjectShortName("v")
	group, err := c.AddGroup("e2e-visibility-" + suffix)
	require.NoError(t, err)
	successor := groupDeleteSuccessor(t, c, group.ID)
	t.Cleanup(func() { _ = c.DeleteGroup(group.ID, successor.ID) })

	me, err := c.GetCurrentUser()
	require.NoError(t, err)
	login := "e2e.visibility." + suffix
	user, err := c.AddUser(&api.AddUserRequest{
		Login: login, FullName: "Visibility User", Email: suffix + "@example.com", Password: "Test1234!",
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = c.DeleteUser(user.ID, me.ID) })

	return visibilityFixture{issue: issue.IDReadable, group: *group, user: *user}
}

func readVisibility(t *testing.T, c *api.Client, path string) *api.Visibility {
	t.Helper()
	var result struct {
		Visibility *api.Visibility `json:"visibility"`
	}
	require.NoError(t, c.Get(path+"?fields=visibility($type,permittedGroups(id,name),permittedUsers(id,login))", &result))
	return result.Visibility
}

func permittedIDs(v *api.Visibility) (groups, users []string) {
	if v == nil || v.Type != "LimitedVisibility" {
		return nil, nil
	}
	for _, g := range v.PermittedGroups {
		groups = append(groups, g.ID)
	}
	for _, u := range v.PermittedUsers {
		users = append(users, u.ID)
	}
	return groups, users
}

func visibilityUserToken(t *testing.T, c *api.Client, user api.User) string {
	t.Helper()
	synced, err := c.WaitForHubSync(user.ID)
	require.NoError(t, err)
	require.True(t, synced)
	base := os.Getenv("YOUTRACK_BASE_URL")
	hubUser, err := api.GetHubMeContext(t.Context(), base, user.Login, "Test1234!")
	require.NoError(t, err)
	services, err := api.ListHubServicesContext(t.Context(), base, os.Getenv("YOUTRACK_LOGIN"), os.Getenv("YOUTRACK_PASSWORD"))
	require.NoError(t, err)
	var scopes []string
	for _, service := range services {
		if service.Name == "YouTrack" {
			scopes = append(scopes, service.ID)
		}
	}
	require.NotEmpty(t, scopes)
	token, err := api.CreateHubTokenContext(t.Context(), base, user.Login, "Test1234!", hubUser.ID, "visibility-reader", scopes)
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = api.DeleteHubTokenContext(context.Background(), base, user.Login, "Test1234!", hubUser.ID, token.ID)
	})
	return token.Token
}
