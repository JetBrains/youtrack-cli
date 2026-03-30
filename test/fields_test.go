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

func TestListCustomFields(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Contains(t, r.URL.Path, "/api/admin/customFieldSettings/customFields")
		json.NewEncoder(w).Encode([]api.CustomField{
			{ID: "cf-1", Name: "Priority", FieldType: &api.FieldType{ID: "enum[1]"}},
		})
	}))
	defer srv.Close()

	fields, err := api.NewClient(srv.URL, "tok").ListCustomFields()

	require.NoError(t, err)
	assert.Len(t, fields, 1)
	assert.Equal(t, "Priority", fields[0].Name)
}

func TestAddCustomField(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Contains(t, r.URL.Path, "/api/admin/customFieldSettings/customFields")
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		assert.Equal(t, "Severity", body["name"])
		fieldType, ok := body["fieldType"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "enum[1]", fieldType["id"])
		json.NewEncoder(w).Encode(api.CustomField{ID: "cf-new", Name: "Severity"})
	}))
	defer srv.Close()

	field, err := api.NewClient(srv.URL, "tok").AddCustomField("Severity", "enum[1]")

	require.NoError(t, err)
	assert.Equal(t, "Severity", field.Name)
}

func TestDeleteCustomField(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "DELETE", r.Method)
		assert.Contains(t, r.URL.Path, "cf-1")
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	err := api.NewClient(srv.URL, "tok").DeleteCustomField("cf-1")

	require.NoError(t, err)
}

func TestListBundles(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Contains(t, r.URL.Path, "/api/admin/customFieldSettings/bundles/EnumBundle")
		json.NewEncoder(w).Encode([]api.Bundle{
			{ID: "b-1", Name: "Priority Bundle"},
		})
	}))
	defer srv.Close()

	bundles, err := api.NewClient(srv.URL, "tok").ListBundles("EnumBundle")

	require.NoError(t, err)
	assert.Len(t, bundles, 1)
	assert.Equal(t, "Priority Bundle", bundles[0].Name)
}

func TestListWorkItemTypes(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Contains(t, r.URL.Path, "/api/admin/timeTrackingSettings/workItemTypes")
		json.NewEncoder(w).Encode([]api.WorkItemType{
			{ID: "wt-1", Name: "Development"},
			{ID: "wt-2", Name: "Testing"},
		})
	}))
	defer srv.Close()

	types, err := api.NewClient(srv.URL, "tok").ListWorkItemTypes()

	require.NoError(t, err)
	assert.Len(t, types, 2)
	assert.Equal(t, "Development", types[0].Name)
}

func TestAddWorkItemType(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Contains(t, r.URL.Path, "/api/admin/timeTrackingSettings/workItemTypes")
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		assert.Equal(t, "Review", body["name"])
		json.NewEncoder(w).Encode(api.WorkItemType{ID: "wt-new", Name: "Review"})
	}))
	defer srv.Close()

	wt, err := api.NewClient(srv.URL, "tok").AddWorkItemType("Review", false)

	require.NoError(t, err)
	assert.Equal(t, "Review", wt.Name)
}

func TestCreateBundle(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Contains(t, r.URL.Path, "EnumBundle")
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		assert.Equal(t, "Priority Bundle", body["name"])
		json.NewEncoder(w).Encode(api.Bundle{ID: "b-new", Name: "Priority Bundle"})
	}))
	defer srv.Close()

	bundle, err := api.NewClient(srv.URL, "tok").CreateBundle("EnumBundle", "Priority Bundle")

	require.NoError(t, err)
	assert.Equal(t, "Priority Bundle", bundle.Name)
}

func TestAddBundleValue(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Contains(t, r.URL.Path, "EnumBundle")
		assert.Contains(t, r.URL.Path, "b-1")
		assert.Contains(t, r.URL.Path, "values")
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		assert.Equal(t, "Critical", body["name"])
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	err := api.NewClient(srv.URL, "tok").AddBundleValue("EnumBundle", "b-1", "Critical")

	require.NoError(t, err)
}

func TestAttachBundleToField(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "PUT", r.Method)
		assert.Contains(t, r.URL.Path, "cf-1")
		assert.Contains(t, r.URL.Path, "fieldDefaults/bundle")
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		assert.Equal(t, "b-1", body["id"])
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	err := api.NewClient(srv.URL, "tok").AttachBundleToField("cf-1", "b-1", "EnumBundle")

	require.NoError(t, err)
}

func TestChangeCustomFieldPrivacy(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Contains(t, r.URL.Path, "cf-1")
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		assert.Equal(t, true, body["isPrivate"])
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	err := api.NewClient(srv.URL, "tok").ChangeCustomFieldPrivacy("cf-1", true)

	require.NoError(t, err)
}

func TestGetWorkTimeSettings(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Contains(t, r.URL.Path, "workTimeSettings")
		json.NewEncoder(w).Encode(api.WorkTimeSettings{
			ID: "wts-1", MinutesADay: 480, DaysAWeek: 5,
		})
	}))
	defer srv.Close()

	settings, err := api.NewClient(srv.URL, "tok").GetWorkTimeSettings()

	require.NoError(t, err)
	assert.Equal(t, 480, settings.MinutesADay)
}

func TestSetWorkTimeSettings(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Contains(t, r.URL.Path, "workTimeSettings")
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		assert.Equal(t, "8h", body["minutesADayPresentation"])
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	err := api.NewClient(srv.URL, "tok").SetWorkTimeSettings("8h")

	require.NoError(t, err)
}

func TestGetCustomFieldPrototypes(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Contains(t, r.URL.Path, "customFields")
		json.NewEncoder(w).Encode([]api.CustomField{
			{ID: "cf-1", Name: "Priority"},
		})
	}))
	defer srv.Close()

	fields, err := api.NewClient(srv.URL, "tok").GetCustomFieldPrototypes()

	require.NoError(t, err)
	assert.Len(t, fields, 1)
	assert.Equal(t, "Priority", fields[0].Name)
}
