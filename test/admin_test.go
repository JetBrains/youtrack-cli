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

func TestGetGlobalSettings(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Contains(t, r.URL.Path, "/api/admin/globalSettings")
		json.NewEncoder(w).Encode(api.GlobalSettings{
			ID:                "1",
			BaseURL:           "https://example.youtrack.cloud",
			MaxUploadFileSize: 104857600,
		})
	}))
	defer srv.Close()

	settings, err := api.NewClient(srv.URL, "tok").GetGlobalSettings()

	require.NoError(t, err)
	assert.Equal(t, "https://example.youtrack.cloud", settings.BaseURL)
	assert.Equal(t, int64(104857600), settings.MaxUploadFileSize)
}

func TestEnableFeatureFlag(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Contains(t, r.URL.Path, "/api/admin/featureFlags/")
		assert.Contains(t, r.URL.Path, "myFeature")
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		assert.Equal(t, "myFeature", body["id"])
		assert.Equal(t, true, body["enabled"])
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	err := api.NewClient(srv.URL, "tok").EnableFeatureFlag("myFeature", true)

	require.NoError(t, err)
}

func TestSetLicense(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Contains(t, r.URL.Path, "/api/admin/globalSettings")
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		license, ok := body["license"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "alice", license["username"])
		assert.Equal(t, "LICENSE-KEY-123", license["license"])
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	err := api.NewClient(srv.URL, "tok").SetLicense("alice", "LICENSE-KEY-123")

	require.NoError(t, err)
}

func TestSetEmailSettings(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Contains(t, r.URL.Path, "/api/admin/globalSettings/notificationSettings")
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		emailSettings, ok := body["emailSettings"].(map[string]any)
		require.True(t, ok)
		mailServer, ok := emailSettings["mailServerSettings"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "smtp.example.com", mailServer["host"])
		assert.Equal(t, float64(587), mailServer["port"])
		assert.Equal(t, "no-reply@example.com", emailSettings["from"])
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	err := api.NewClient(srv.URL, "tok").SetEmailSettings("smtp.example.com", 587, "no-reply@example.com")

	require.NoError(t, err)
}

func TestDisableEmailSettings(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Contains(t, r.URL.Path, "/api/admin/globalSettings/notificationSettings")
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		emailSettings, ok := body["emailSettings"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, false, emailSettings["isEnabled"])
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	err := api.NewClient(srv.URL, "tok").DisableEmailSettings()

	require.NoError(t, err)
}

func TestSetGlobalBanner(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Contains(t, r.URL.Path, "/api/admin/globalSettings")
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		appearance, ok := body["appearanceSettings"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "Maintenance tonight", appearance["globalBannerText"])
		assert.Equal(t, true, appearance["showGlobalBanner"])
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	err := api.NewClient(srv.URL, "tok").SetGlobalBanner("Maintenance tonight", true)

	require.NoError(t, err)
}

func TestSetReactionsEnabled(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Contains(t, r.URL.Path, "/api/admin/globalSettings")
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		appearance, ok := body["appearanceSettings"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, false, appearance["enableReactions"])
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	err := api.NewClient(srv.URL, "tok").SetReactionsEnabled(false)

	require.NoError(t, err)
}

func TestSwitchJetBrainsAI(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Contains(t, r.URL.Path, "/api/grazie/config")
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		assert.Equal(t, true, body["enabled"])
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	err := api.NewClient(srv.URL, "tok").SwitchJetBrainsAI(true)

	require.NoError(t, err)
}

func TestSwitchTextCompletion(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Contains(t, r.URL.Path, "/api/grazie/config")
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		assert.Equal(t, false, body["enableTextCompletion"])
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	err := api.NewClient(srv.URL, "tok").SwitchTextCompletion(false)

	require.NoError(t, err)
}

func TestSwitchGrammarCorrection(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Contains(t, r.URL.Path, "/api/grazie/config")
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		assert.Equal(t, true, body["enableSpellChecker"])
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	err := api.NewClient(srv.URL, "tok").SwitchGrammarCorrection(true)

	require.NoError(t, err)
}

func TestListVCSServers(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Contains(t, r.URL.Path, "/api/admin/integrations/vcsHostingServers")
		json.NewEncoder(w).Encode([]api.VCSServer{
			{ID: "vcs-1", Type: "GitHub", URL: "https://github.com"},
		})
	}))
	defer srv.Close()

	servers, err := api.NewClient(srv.URL, "tok").ListVCSServers()

	require.NoError(t, err)
	assert.Len(t, servers, 1)
	assert.Equal(t, "vcs-1", servers[0].ID)
	assert.Equal(t, "GitHub", servers[0].Type)
}

func TestDeleteVCSServer(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "DELETE", r.Method)
		assert.Contains(t, r.URL.Path, "/api/admin/integrations/vcsHostingServers/")
		assert.Contains(t, r.URL.Path, "vcs-42")
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	err := api.NewClient(srv.URL, "tok").DeleteVCSServer("vcs-42")

	require.NoError(t, err)
}

func TestGetVCSConfigurations(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Contains(t, r.URL.Path, "/api/admin/integrations/vcsHostingServers/")
		assert.Contains(t, r.URL.Path, "srv-7")
		assert.Contains(t, r.URL.Path, "changesProcessors")
		json.NewEncoder(w).Encode([]map[string]any{
			{"id": "proc-1", "path": "/repo"},
		})
	}))
	defer srv.Close()

	configs, err := api.NewClient(srv.URL, "tok").GetVCSConfigurations("srv-7")

	require.NoError(t, err)
	assert.Len(t, configs, 1)
	assert.Equal(t, "proc-1", configs[0]["id"])
}

func TestListVCSProcessors(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Contains(t, r.URL.Path, "/api/admin/integrations/vcsHostingProcessors")
		json.NewEncoder(w).Encode([]api.VCSProcessor{
			{ID: "proc-1", Path: "/repo/path", Enabled: true},
		})
	}))
	defer srv.Close()

	procs, err := api.NewClient(srv.URL, "tok").ListVCSProcessors()

	require.NoError(t, err)
	assert.Len(t, procs, 1)
	assert.Equal(t, "proc-1", procs[0].ID)
	assert.True(t, procs[0].Enabled)
}

func TestDeleteVCSProcessor(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "DELETE", r.Method)
		assert.Contains(t, r.URL.Path, "/api/admin/integrations/vcsHostingProcessors/")
		assert.Contains(t, r.URL.Path, "proc-99")
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	err := api.NewClient(srv.URL, "tok").DeleteVCSProcessor("proc-99")

	require.NoError(t, err)
}

func TestListMailboxServers(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Contains(t, r.URL.Path, "/api/admin/integrations/mailbox/mailboxes")
		json.NewEncoder(w).Encode([]api.MailboxServer{
			{ID: "mb-1", Login: "support@example.com", Host: "imap.example.com", Port: 993},
		})
	}))
	defer srv.Close()

	servers, err := api.NewClient(srv.URL, "tok").ListMailboxServers()

	require.NoError(t, err)
	assert.Len(t, servers, 1)
	assert.Equal(t, "mb-1", servers[0].ID)
	assert.Equal(t, "imap.example.com", servers[0].Host)
}

func TestAddMailboxServer(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Contains(t, r.URL.Path, "/api/admin/integrations/mailbox/mailboxes")
		var body api.AddMailboxServerRequest
		json.NewDecoder(r.Body).Decode(&body)
		assert.Equal(t, "imap.example.com", body.Host)
		assert.Equal(t, "support", body.Login)
		assert.Equal(t, "IMAP", body.Protocol)
		assert.Equal(t, 993, body.Port)
		json.NewEncoder(w).Encode(api.MailboxServer{
			ID: "mb-new", Login: body.Login, Host: body.Host, Port: body.Port,
		})
	}))
	defer srv.Close()

	server, err := api.NewClient(srv.URL, "tok").AddMailboxServer(api.AddMailboxServerRequest{
		Protocol: "IMAP",
		Host:     "imap.example.com",
		Port:     993,
		Login:    "support",
		Password: "secret",
	})

	require.NoError(t, err)
	assert.Equal(t, "mb-new", server.ID)
	assert.Equal(t, "imap.example.com", server.Host)
}

func TestDeleteMailboxServer(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "DELETE", r.Method)
		assert.Contains(t, r.URL.Path, "/api/admin/integrations/mailbox/mailboxes/")
		assert.Contains(t, r.URL.Path, "mb-5")
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	err := api.NewClient(srv.URL, "tok").DeleteMailboxServer("mb-5")

	require.NoError(t, err)
}

func TestAddMailboxRule(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Contains(t, r.URL.Path, "/api/admin/integrations/mailbox/rules")
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		mailbox, ok := body["mailbox"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "mb-1", mailbox["id"])
		project, ok := body["project"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "proj-1", project["id"])
		assert.Equal(t, "Support Inbox Rule", body["name"])
		assert.Equal(t, false, body["disabled"])
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	err := api.NewClient(srv.URL, "tok").AddMailboxRule("mb-1", "proj-1", "Support Inbox Rule")

	require.NoError(t, err)
}

func TestGetFrontendConfig(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Contains(t, r.URL.Path, "/api/config")
		json.NewEncoder(w).Encode(map[string]any{"version": "2024.1", "buildNumber": "241.1234"})
	}))
	defer srv.Close()

	cfg, err := api.NewClient(srv.URL, "tok").GetFrontendConfig()

	require.NoError(t, err)
	assert.Equal(t, "2024.1", cfg["version"])
}

func TestListPlugins(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Contains(t, r.URL.Path, "pluginUsages")
		json.NewEncoder(w).Encode([]api.Plugin{
			{ID: "pl-1", Enabled: true, State: "LOADED"},
		})
	}))
	defer srv.Close()

	plugins, err := api.NewClient(srv.URL, "tok").ListPlugins()

	require.NoError(t, err)
	assert.Len(t, plugins, 1)
	assert.True(t, plugins[0].Enabled)
}

func TestDeletePlugin(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "DELETE", r.Method)
		assert.Contains(t, r.URL.Path, "pl-1")
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	err := api.NewClient(srv.URL, "tok").DeletePlugin("pl-1")

	require.NoError(t, err)
}

func TestRawRequest(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Contains(t, r.URL.Path, "/api/issues")
		json.NewEncoder(w).Encode([]any{map[string]any{"id": "FOO-1"}})
	}))
	defer srv.Close()

	result, err := api.NewClient(srv.URL, "tok").RawRequest("GET", "/api/issues", nil)

	require.NoError(t, err)
	assert.NotNil(t, result)
}
