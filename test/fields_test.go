package api_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/JetBrains/youtrack-cli/api"
)

// ---------- BundlePathForFieldType unit tests (no API needed) ----------

func TestBundlePathForFieldType_SimplTypes(t *testing.T) {
	simpleTypes := []string{
		api.FieldTypeString, api.FieldTypeText, api.FieldTypeInteger,
		api.FieldTypeFloat, api.FieldTypeDate, api.FieldTypeDateTime, api.FieldTypePeriod,
	}
	for _, ft := range simpleTypes {
		path, ok := api.BundlePathForFieldType(ft)
		assert.False(t, ok, "type %q should not have a bundle path", ft)
		assert.Empty(t, path, "type %q should return empty path", ft)
	}
}

func TestBundlePathForFieldType_BundleTypes(t *testing.T) {
	cases := []struct {
		fieldType  string
		bundlePath string
	}{
		{api.FieldTypeEnumSingle, "enum"},
		{api.FieldTypeEnumMulti, "enum"},
		{api.FieldTypeStateSingle, "state"},
		{api.FieldTypeVersionSingle, "version"},
		{api.FieldTypeVersionMulti, "version"},
		{api.FieldTypeBuildSingle, "build"},
		{api.FieldTypeBuildMulti, "build"},
		{api.FieldTypeOwnedSingle, "owned"},
		{api.FieldTypeOwnedMulti, "owned"},
		{api.FieldTypeUserSingle, "user"},
		{api.FieldTypeUserMulti, "user"},
	}
	for _, tc := range cases {
		path, ok := api.BundlePathForFieldType(tc.fieldType)
		assert.True(t, ok, "type %q should have a bundle path", tc.fieldType)
		assert.Equal(t, tc.bundlePath, path, "type %q bundle path", tc.fieldType)
	}
}

// ---------- Simple field type creation (API layer) ----------

func TestField_CreateSimpleTypes(t *testing.T) {
	c := newTestClient(t)

	simpleTypes := []string{
		api.FieldTypeString, api.FieldTypeText, api.FieldTypeInteger,
		api.FieldTypeFloat, api.FieldTypeDate, api.FieldTypeDateTime, api.FieldTypePeriod,
	}

	for _, ft := range simpleTypes {
		ft := ft
		t.Run(ft, func(t *testing.T) {
			name := fmt.Sprintf("tf-%s-%d", ft[:3], time.Now().UnixMilli()%100000)
			field, err := c.AddCustomField(name, ft)
			require.NoError(t, err, "create field type %q", ft)
			require.NotEmpty(t, field.ID)
			t.Cleanup(func() { _ = c.DeleteCustomField(field.ID) })

			assert.Equal(t, name, field.Name)
			require.NotNil(t, field.FieldType)
			assert.Equal(t, ft, field.FieldType.ID)
		})
	}
}

// ---------- Bundle-based field type creation (API layer) ----------

func TestField_CreateBundleTypes(t *testing.T) {
	c := newTestClient(t)

	// Test one single-value and one multi-value variant for each bundle family.
	// state[1] uses a StateBundle and is always single-value; tested separately.
	bundleTypes := []string{
		api.FieldTypeEnumSingle, api.FieldTypeEnumMulti,
		api.FieldTypeVersionSingle, api.FieldTypeVersionMulti,
	}

	for _, ft := range bundleTypes {
		ft := ft
		t.Run(ft, func(t *testing.T) {
			bundlePath, _ := api.BundlePathForFieldType(ft)
			suffix := fmt.Sprintf("%d", time.Now().UnixMilli()%100000)
			fieldName := fmt.Sprintf("tf-%s-%s", bundlePath, suffix)
			bundleName := fmt.Sprintf("tb-%s-%s", bundlePath, suffix)

			// Step 1: create the field
			field, err := c.AddCustomField(fieldName, ft)
			require.NoError(t, err, "create field type %q", ft)
			require.NotEmpty(t, field.ID)

			// Step 2: create bundle (register cleanup before attach so it runs first)
			bundle, err := c.CreateBundle(bundlePath, bundleName)
			require.NoError(t, err, "create bundle %q for field type %q", bundlePath, ft)
			require.NotEmpty(t, bundle.ID)

			// Cleanup order matters: detach/delete bundle before deleting field.
			t.Cleanup(func() { _ = c.DeleteBundle(bundlePath, bundle.ID) })
			t.Cleanup(func() { _ = c.DeleteCustomField(field.ID) })

			// Step 3: attach bundle to field
			err = c.AttachBundleToField(field.ID, bundle.ID, bundlePath)
			require.NoError(t, err, "attach bundle to field type %q", ft)

			assert.Equal(t, ft, field.FieldType.ID)
		})
	}
}

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
