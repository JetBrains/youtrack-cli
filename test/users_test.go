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
