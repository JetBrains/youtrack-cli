package api_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/JetBrains/youtrack-cli/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------- Tags ----------

func TestListTags(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Contains(t, r.URL.Path, "/api/tags")
		json.NewEncoder(w).Encode([]api.Tag{
			{ID: "tag-1", Name: "urgent"},
			{ID: "tag-2", Name: "blocker"},
		})
	}))
	defer srv.Close()

	tags, err := api.NewClient(srv.URL, "tok").ListTags()
	require.NoError(t, err)
	assert.Len(t, tags, 2)
	assert.Equal(t, "urgent", tags[0].Name)
}

func TestAddTag(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		var body api.AddTagRequest
		json.NewDecoder(r.Body).Decode(&body)
		assert.Equal(t, "my-tag", body.Name)
		json.NewEncoder(w).Encode(api.Tag{ID: "tag-3", Name: "my-tag"})
	}))
	defer srv.Close()

	tag, err := api.NewClient(srv.URL, "tok").AddTag(api.AddTagRequest{Name: "my-tag"})
	require.NoError(t, err)
	assert.Equal(t, "tag-3", tag.ID)
	assert.Equal(t, "my-tag", tag.Name)
}

func TestDeleteTag(t *testing.T) {
	called := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "DELETE", r.Method)
		assert.Contains(t, r.URL.Path, "tag-1")
		called = true
		w.WriteHeader(204)
	}))
	defer srv.Close()

	err := api.NewClient(srv.URL, "tok").DeleteTag("tag-1")
	require.NoError(t, err)
	assert.True(t, called)
}

// ---------- Saved Searches ----------

func TestAddSavedSearch(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		var body api.AddSavedSearchRequest
		json.NewDecoder(r.Body).Decode(&body)
		assert.Equal(t, "My Search", body.Name)
		assert.Equal(t, "project: FOO", body.Query)
		json.NewEncoder(w).Encode(api.SavedSearch{ID: "sq-1", Name: "My Search", Query: "project: FOO"})
	}))
	defer srv.Close()

	result, err := api.NewClient(srv.URL, "tok").AddSavedSearch(api.AddSavedSearchRequest{
		Name:  "My Search",
		Query: "project: FOO",
	})
	require.NoError(t, err)
	assert.Equal(t, "sq-1", result.ID)
	assert.Equal(t, "project: FOO", result.Query)
}

func TestShareSavedSearch(t *testing.T) {
	called := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Contains(t, r.URL.Path, "sq-1")
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		assert.Equal(t, true, body["isShareable"])
		called = true
		w.WriteHeader(200)
	}))
	defer srv.Close()

	err := api.NewClient(srv.URL, "tok").ShareSavedSearch("sq-1")
	require.NoError(t, err)
	assert.True(t, called)
}

// ---------- Organizations ----------

func TestListOrganizations(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Contains(t, r.URL.Path, "/api/admin/organizations")
		json.NewEncoder(w).Encode([]api.Organization{
			{ID: "org-1", Name: "Acme"},
		})
	}))
	defer srv.Close()

	orgs, err := api.NewClient(srv.URL, "tok").ListOrganizations()
	require.NoError(t, err)
	assert.Len(t, orgs, 1)
	assert.Equal(t, "Acme", orgs[0].Name)
}

func TestAddOrganization(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		var body api.AddOrganizationRequest
		json.NewDecoder(r.Body).Decode(&body)
		assert.Equal(t, "NewOrg", body.Name)
		json.NewEncoder(w).Encode(api.Organization{ID: "org-2", Name: "NewOrg"})
	}))
	defer srv.Close()

	org, err := api.NewClient(srv.URL, "tok").AddOrganization(api.AddOrganizationRequest{Name: "NewOrg"})
	require.NoError(t, err)
	assert.Equal(t, "org-2", org.ID)
}

func TestDeleteOrganization(t *testing.T) {
	called := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "DELETE", r.Method)
		assert.Contains(t, r.URL.Path, "org-1")
		called = true
		w.WriteHeader(204)
	}))
	defer srv.Close()

	err := api.NewClient(srv.URL, "tok").DeleteOrganization("org-1")
	require.NoError(t, err)
	assert.True(t, called)
}

// ---------- Apps ----------

func TestListApps(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Contains(t, r.URL.Path, "/api/admin/apps")
		json.NewEncoder(w).Encode([]api.App{
			{ID: "app-1", Name: "myplugin", Title: "My Plugin", Version: "1.0"},
		})
	}))
	defer srv.Close()

	apps, err := api.NewClient(srv.URL, "tok").ListApps()
	require.NoError(t, err)
	assert.Len(t, apps, 1)
	assert.Equal(t, "myplugin", apps[0].Name)
	assert.Equal(t, "1.0", apps[0].Version)
}

func TestDeleteApp(t *testing.T) {
	called := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "DELETE", r.Method)
		assert.Contains(t, r.URL.Path, "app-1")
		called = true
		w.WriteHeader(204)
	}))
	defer srv.Close()

	err := api.NewClient(srv.URL, "tok").DeleteApp("app-1")
	require.NoError(t, err)
	assert.True(t, called)
}

func TestDeactivateApp(t *testing.T) {
	called := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Contains(t, r.URL.Path, "app-1")
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		gc, ok := body["globalConfig"].(map[string]any)
		assert.True(t, ok)
		assert.Equal(t, false, gc["enabled"])
		called = true
		w.WriteHeader(200)
	}))
	defer srv.Close()

	err := api.NewClient(srv.URL, "tok").DeactivateApp("app-1")
	require.NoError(t, err)
	assert.True(t, called)
}

// ---------- Canvas ----------

func TestAddCanvas(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Contains(t, r.URL.Path, "/api/planningCanvases")
		var body api.AddCanvasRequest
		json.NewDecoder(r.Body).Decode(&body)
		assert.Equal(t, "Sprint Board", body.Name)
		json.NewEncoder(w).Encode(api.Canvas{ID: "cv-1", Name: "Sprint Board"})
	}))
	defer srv.Close()

	canvas, err := api.NewClient(srv.URL, "tok").AddCanvas(api.AddCanvasRequest{Name: "Sprint Board"})
	require.NoError(t, err)
	assert.Equal(t, "cv-1", canvas.ID)
}

func TestGetCanvas(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Contains(t, r.URL.Path, "cv-1")
		json.NewEncoder(w).Encode(api.Canvas{ID: "cv-1", Name: "Sprint Board"})
	}))
	defer srv.Close()

	canvas, err := api.NewClient(srv.URL, "tok").GetCanvas("cv-1")
	require.NoError(t, err)
	assert.Equal(t, "Sprint Board", canvas.Name)
}

func TestDeleteCanvas(t *testing.T) {
	called := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "DELETE", r.Method)
		assert.Contains(t, r.URL.Path, "cv-1")
		called = true
		w.WriteHeader(204)
	}))
	defer srv.Close()

	err := api.NewClient(srv.URL, "tok").DeleteCanvas("cv-1")
	require.NoError(t, err)
	assert.True(t, called)
}

func TestReloadCanvas(t *testing.T) {
	called := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Contains(t, r.URL.Path, "cv-1")
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		assert.Equal(t, true, body["reload"])
		called = true
		w.WriteHeader(200)
	}))
	defer srv.Close()

	err := api.NewClient(srv.URL, "tok").ReloadCanvas("cv-1")
	require.NoError(t, err)
	assert.True(t, called)
}

func TestAddCanvasNode(t *testing.T) {
	called := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Contains(t, r.URL.Path, "cv-1/cards")
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		assert.Equal(t, float64(10), body["x"])
		assert.Equal(t, float64(20), body["y"])
		assert.Equal(t, "Node A", body["label"])
		called = true
		w.WriteHeader(200)
	}))
	defer srv.Close()

	err := api.NewClient(srv.URL, "tok").AddCanvasNode("cv-1", 10, 20, "Node A", "desc")
	require.NoError(t, err)
	assert.True(t, called)
}

func TestGetCanvasLinks(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Contains(t, r.URL.Path, "cv-1/links")
		json.NewEncoder(w).Encode([]api.CanvasLink{
			{ID: "lnk-1"},
		})
	}))
	defer srv.Close()

	links, err := api.NewClient(srv.URL, "tok").GetCanvasLinks("cv-1")
	require.NoError(t, err)
	assert.Len(t, links, 1)
	assert.Equal(t, "lnk-1", links[0].ID)
}

func TestLinkCanvasNodes(t *testing.T) {
	called := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Contains(t, r.URL.Path, "cv-1/links")
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		src, _ := body["sourceCard"].(map[string]any)
		tgt, _ := body["targetCard"].(map[string]any)
		assert.Equal(t, "card-a", src["id"])
		assert.Equal(t, "card-b", tgt["id"])
		called = true
		w.WriteHeader(200)
	}))
	defer srv.Close()

	err := api.NewClient(srv.URL, "tok").LinkCanvasNodes("cv-1", "card-a", "card-b")
	require.NoError(t, err)
	assert.True(t, called)
}

// ---------- Reports / Removed Documents / Gantt ----------

func TestGetRemovedDocuments(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Contains(t, r.URL.Path, "/api/removedDocuments")
		json.NewEncoder(w).Encode([]api.RemovedDocument{
			{ID: "rd-1", IDReadable: "FOO-5", Summary: "Deleted issue"},
		})
	}))
	defer srv.Close()

	docs, err := api.NewClient(srv.URL, "tok").GetRemovedDocuments()
	require.NoError(t, err)
	assert.Len(t, docs, 1)
	assert.Equal(t, "FOO-5", docs[0].IDReadable)
}

func TestRestoreDocument(t *testing.T) {
	called := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Contains(t, r.URL.Path, "/api/removedDocuments")
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		docs, _ := body["documents"].([]any)
		assert.Len(t, docs, 1)
		first, _ := docs[0].(map[string]any)
		assert.Equal(t, "rd-1", first["id"])
		assert.Equal(t, false, body["remove"])
		called = true
		w.WriteHeader(200)
	}))
	defer srv.Close()

	err := api.NewClient(srv.URL, "tok").RestoreDocument("rd-1")
	require.NoError(t, err)
	assert.True(t, called)
}

func TestDeleteReport(t *testing.T) {
	called := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "DELETE", r.Method)
		assert.True(t, strings.HasSuffix(r.URL.Path, "/api/reports/rpt-1"))
		called = true
		w.WriteHeader(204)
	}))
	defer srv.Close()

	err := api.NewClient(srv.URL, "tok").DeleteReport("rpt-1")
	require.NoError(t, err)
	assert.True(t, called)
}

func TestListGanttCharts(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Contains(t, r.URL.Path, "/api/ganttCharts")
		json.NewEncoder(w).Encode([]api.GanttChart{
			{ID: "gc-1", Name: "Q1 Gantt"},
		})
	}))
	defer srv.Close()

	charts, err := api.NewClient(srv.URL, "tok").ListGanttCharts()
	require.NoError(t, err)
	assert.Len(t, charts, 1)
	assert.Equal(t, "Q1 Gantt", charts[0].Name)
}

func TestDeleteGanttChart(t *testing.T) {
	called := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "DELETE", r.Method)
		assert.Contains(t, r.URL.Path, "gc-1")
		called = true
		w.WriteHeader(204)
	}))
	defer srv.Close()

	err := api.NewClient(srv.URL, "tok").DeleteGanttChart("gc-1")
	require.NoError(t, err)
	assert.True(t, called)
}
