package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListUsers(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.URL.RawQuery, "query=")
		json.NewEncoder(w).Encode([]User{{Login: "alice", FullName: "Alice Smith"}})
	}))
	defer srv.Close()

	users, err := NewClient(srv.URL, "tok").ListUsers("alice")

	require.NoError(t, err)
	assert.Len(t, users, 1)
	assert.Equal(t, "alice", users[0].Login)
}

func TestGetUser_found(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode([]User{{Login: "alice", FullName: "Alice Smith"}})
	}))
	defer srv.Close()

	user, err := NewClient(srv.URL, "tok").GetUser("alice")

	require.NoError(t, err)
	assert.Equal(t, "alice", user.Login)
}

func TestGetUser_notFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode([]User{})
	}))
	defer srv.Close()

	_, err := NewClient(srv.URL, "tok").GetUser("nobody")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "nobody")
}

func TestGetCurrentUser(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.URL.Path, "/api/users/me")
		json.NewEncoder(w).Encode(User{Login: "admin", FullName: "Admin User"})
	}))
	defer srv.Close()

	user, err := NewClient(srv.URL, "tok").GetCurrentUser()

	require.NoError(t, err)
	assert.Equal(t, "admin", user.Login)
}
