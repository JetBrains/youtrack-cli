package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGet_success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"id": "1"})
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "test-token")
	var result map[string]string
	err := c.Get("/test", &result)

	require.NoError(t, err)
	assert.Equal(t, "1", result["id"])
}

func TestGet_httpError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(404)
		json.NewEncoder(w).Encode(map[string]string{"error": "Not Found"})
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "test-token")
	err := c.Get("/missing", nil)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "404")
}

func TestPost_success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		var body map[string]string
		json.NewDecoder(r.Body).Decode(&body)
		assert.Equal(t, "hello", body["name"])
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"id": "42"})
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "test-token")
	var result map[string]string
	err := c.Post("/test", map[string]string{"name": "hello"}, &result)

	require.NoError(t, err)
	assert.Equal(t, "42", result["id"])
}

func TestDelete_success(t *testing.T) {
	called := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "DELETE", r.Method)
		called = true
		w.WriteHeader(204)
	}))
	defer srv.Close()

	err := NewClient(srv.URL, "tok").Delete("/test/1")

	require.NoError(t, err)
	assert.True(t, called)
}
