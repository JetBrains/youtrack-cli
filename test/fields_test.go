package api_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestField_ListCustomFields(t *testing.T) {
	c := newTestClient(t)

	fields, err := c.ListCustomFields()
	require.NoError(t, err)
	// Every YouTrack instance has at least some built-in custom fields.
	assert.NotEmpty(t, fields)
}

func TestField_ListBundles(t *testing.T) {
	c := newTestClient(t)

	// The YouTrack API path for enum bundles uses lowercase "enum".
	bundles, err := c.ListBundles("enum")
	require.NoError(t, err)
	// Every YouTrack instance has at least the built-in Priorities and Types bundles.
	assert.NotEmpty(t, bundles)
}

func TestField_AddBundleValue(t *testing.T) {
	c := newTestClient(t)

	// Create a fresh enum bundle so the added value can be cleaned up by deleting the whole bundle.
	bundleName := fmt.Sprintf("testbundle-%d", time.Now().UnixMilli()%100000)
	bundle, err := c.CreateBundle("enum", bundleName)
	require.NoError(t, err)
	require.NotEmpty(t, bundle.ID)
	t.Cleanup(func() { _ = c.DeleteBundle("enum", bundle.ID) })

	// Add a unique value to the new bundle.
	value := fmt.Sprintf("testval-%d", time.Now().UnixMilli()%100000)
	err = c.AddBundleValue("enum", bundle.ID, value)
	require.NoError(t, err)
}

func TestField_ListWorkItemTypes(t *testing.T) {
	c := newTestClient(t)

	types, err := c.ListWorkItemTypes()
	require.NoError(t, err)
	// Every YouTrack instance has built-in work item types.
	assert.NotEmpty(t, types)
}

func TestField_GetWorkTimeSettings(t *testing.T) {
	c := newTestClient(t)

	settings, err := c.GetWorkTimeSettings()
	require.NoError(t, err)
	assert.NotEmpty(t, settings.ID)
}

func TestCmd_FieldList(t *testing.T) {
	out := runCmd(t, "field", "list", "--format", "json")
	assert.NotEmpty(t, out)
}

func TestCmd_FieldBundleList(t *testing.T) {
	out := runCmd(t, "field", "bundle", "list", "--type", "enum", "--format", "json")
	assert.NotEmpty(t, out)
}

func TestCmd_FieldWorktimeGet(t *testing.T) {
	out := runCmd(t, "field", "worktime", "get", "--format", "json")
	assert.NotEmpty(t, out)
}
