package api_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/JetBrains/youtrack-cli/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListUsers(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.URL.RawQuery, "query=")
		assert.Contains(t, r.URL.RawQuery, "$top=-1")
		json.NewEncoder(w).Encode([]api.User{{Login: "alice", FullName: "Alice Smith"}})
	}))
	defer srv.Close()

	users, err := api.NewClient(srv.URL, "tok").ListUsers("alice")

	require.NoError(t, err)
	assert.Len(t, users, 1)
	assert.Equal(t, "alice", users[0].Login)
}

func TestGetUser_found(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode([]api.User{{Login: "alice", FullName: "Alice Smith"}})
	}))
	defer srv.Close()

	user, err := api.NewClient(srv.URL, "tok").GetUser("alice")

	require.NoError(t, err)
	assert.Equal(t, "alice", user.Login)
}

func TestGetUser_notFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode([]api.User{})
	}))
	defer srv.Close()

	_, err := api.NewClient(srv.URL, "tok").GetUser("nobody")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "nobody")
}

func TestAddUser(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		var body api.AddUserRequest
		json.NewDecoder(r.Body).Decode(&body)
		assert.Equal(t, "alice", body.Login)
		assert.Equal(t, "Alice Smith", body.FullName)
		assert.Equal(t, "alice@example.com", body.Email)
		json.NewEncoder(w).Encode(api.User{Login: "alice", FullName: "Alice Smith"})
	}))
	defer srv.Close()

	user, err := api.NewClient(srv.URL, "tok").AddUser(api.AddUserRequest{
		Login:    "alice",
		FullName: "Alice Smith",
		Email:    "alice@example.com",
	})

	require.NoError(t, err)
	assert.Equal(t, "alice", user.Login)
}

func TestDeleteUser(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "DELETE", r.Method)
		assert.Contains(t, r.URL.Path, "u-1")
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	err := api.NewClient(srv.URL, "tok").DeleteUser("u-1")

	require.NoError(t, err)
}

func TestBanUser(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Contains(t, r.URL.Path, "u-1")
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		assert.Equal(t, true, body["banned"])
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	err := api.NewClient(srv.URL, "tok").BanUser("u-1", true)

	require.NoError(t, err)
}

func TestUnbanUser(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Contains(t, r.URL.Path, "u-1")
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		assert.Equal(t, false, body["banned"])
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	err := api.NewClient(srv.URL, "tok").BanUser("u-1", false)

	require.NoError(t, err)
}

func TestSetUserTimezone(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Contains(t, r.URL.Path, "u-1")
		assert.Contains(t, r.URL.Path, "profiles/general")
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		tz, ok := body["timezone"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "America/New_York", tz["id"])
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	err := api.NewClient(srv.URL, "tok").SetUserTimezone("u-1", "America/New_York")

	require.NoError(t, err)
}

func TestSetUserLocale(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Contains(t, r.URL.Path, "u-1")
		assert.Contains(t, r.URL.Path, "profiles/general")
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		locale, ok := body["locale"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "en_US", locale["locale"])
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	err := api.NewClient(srv.URL, "tok").SetUserLocale("u-1", "en_US")

	require.NoError(t, err)
}

func TestGetUserIssueDrafts(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Contains(t, r.URL.Path, "me")
		assert.Contains(t, r.URL.Path, "drafts")
		json.NewEncoder(w).Encode([]api.IssueDraft{
			{ID: "d-1", Summary: "Draft issue"},
		})
	}))
	defer srv.Close()

	drafts, err := api.NewClient(srv.URL, "tok").GetUserIssueDrafts("me")

	require.NoError(t, err)
	assert.Len(t, drafts, 1)
	assert.Equal(t, "Draft issue", drafts[0].Summary)
}

func TestDeleteAllUserDrafts(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "PUT", r.Method)
		assert.Contains(t, r.URL.Path, "me")
		assert.Contains(t, r.URL.Path, "drafts")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	err := api.NewClient(srv.URL, "tok").DeleteAllUserDrafts("me")

	require.NoError(t, err)
}

func TestGetUserTimezone(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.URL.Path, "u-1")
		assert.Contains(t, r.URL.Path, "profiles/general")
		json.NewEncoder(w).Encode(map[string]any{
			"timezone": map[string]any{"id": "America/New_York", "presentation": "Eastern Time"},
		})
	}))
	defer srv.Close()

	tz, err := api.NewClient(srv.URL, "tok").GetUserTimezone("u-1")

	require.NoError(t, err)
	assert.Equal(t, "Eastern Time", tz)
}

func TestGetUserArticleDrafts(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Contains(t, r.URL.Path, "me")
		assert.Contains(t, r.URL.Path, "articleDrafts")
		json.NewEncoder(w).Encode([]api.ArticleDraft{{ID: "ad-1", Summary: "Draft article"}})
	}))
	defer srv.Close()

	drafts, err := api.NewClient(srv.URL, "tok").GetUserArticleDrafts("me")

	require.NoError(t, err)
	assert.Len(t, drafts, 1)
	assert.Equal(t, "Draft article", drafts[0].Summary)
}

func TestPinProjectForUser(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Contains(t, r.URL.Path, "u-1")
		assert.Contains(t, r.URL.Path, "pinnedProjects")
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		assert.Equal(t, "p-1", body["id"])
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	err := api.NewClient(srv.URL, "tok").PinProjectForUser("u-1", "p-1")

	require.NoError(t, err)
}

func TestSetUserGeneralProfile(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Contains(t, r.URL.Path, "u-1")
		assert.Contains(t, r.URL.Path, "profiles/general")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	err := api.NewClient(srv.URL, "tok").SetUserGeneralProfile("u-1", map[string]any{"key": "val"})

	require.NoError(t, err)
}

func TestSetUserProfileProperty(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Contains(t, r.URL.Path, "u-1")
		assert.Contains(t, r.URL.Path, "propertiesProfile/properties")
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		assert.Equal(t, "theme", body["key"])
		assert.Equal(t, "dark", body["value"])
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	err := api.NewClient(srv.URL, "tok").SetUserProfileProperty("u-1", "theme", "dark")

	require.NoError(t, err)
}

func TestSetUserAppearanceSetting(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Contains(t, r.URL.Path, "u-1")
		assert.Contains(t, r.URL.Path, "profiles/appearance")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	err := api.NewClient(srv.URL, "tok").SetUserAppearanceSetting("u-1", map[string]any{"theme": "dark"})

	require.NoError(t, err)
}

func TestSetUserNotificationsProfile(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Contains(t, r.URL.Path, "profiles/notifications")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	err := api.NewClient(srv.URL, "tok").SetUserNotificationsProfile("u-1", map[string]any{})

	require.NoError(t, err)
}

func TestSetUserGrazieProfile(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Contains(t, r.URL.Path, "profiles/grazie")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	err := api.NewClient(srv.URL, "tok").SetUserGrazieProfile("u-1", map[string]any{})

	require.NoError(t, err)
}

func TestSetUserArticlesProfile(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Contains(t, r.URL.Path, "profiles/articles")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	err := api.NewClient(srv.URL, "tok").SetUserArticlesProfile("u-1", map[string]any{})

	require.NoError(t, err)
}

func TestSetUserIssuesListSetting(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Contains(t, r.URL.Path, "profiles/issuesList")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	err := api.NewClient(srv.URL, "tok").SetUserIssuesListSetting("u-1", map[string]any{})

	require.NoError(t, err)
}

func TestSetUserProfileTips(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Contains(t, r.URL.Path, "profiles/tips")
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		assert.Equal(t, true, body["showTips"])
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	err := api.NewClient(srv.URL, "tok").SetUserProfileTips("u-1", map[string]bool{"showTips": true})

	require.NoError(t, err)
}

func TestGetCurrentUser(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.URL.Path, "/api/users/me")
		json.NewEncoder(w).Encode(api.User{Login: "admin", FullName: "Admin User"})
	}))
	defer srv.Close()

	user, err := api.NewClient(srv.URL, "tok").GetCurrentUser()

	require.NoError(t, err)
	assert.Equal(t, "admin", user.Login)
}
