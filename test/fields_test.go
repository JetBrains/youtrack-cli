package api_test

import (
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/JetBrains/youtrack-cli/api"
)

func fieldExistsInList(c *api.Client, id string) (bool, error) {
	fields, err := c.ListCustomFields()
	if err != nil {
		return false, err
	}
	for _, f := range fields {
		if f.ID == id {
			return true, nil
		}
	}
	return false, nil
}

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
		{api.FieldTypeOwnedSingle, "ownedField"},
		{api.FieldTypeOwnedMulti, "ownedField"},
		{api.FieldTypeUserSingle, "user"},
		{api.FieldTypeUserMulti, "user"},
	}
	for _, tc := range cases {
		path, ok := api.BundlePathForFieldType(tc.fieldType)
		assert.True(t, ok, "type %q should have a bundle path", tc.fieldType)
		assert.Equal(t, tc.bundlePath, path, "type %q bundle path", tc.fieldType)
	}
}

func TestNormalizeBundlePathType(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"enum", "enum"},
		{"EnumBundle", "enum"},
		{"state", "state"},
		{"StateBundle", "state"},
		{"version", "version"},
		{"VersionBundle", "version"},
		{"build", "build"},
		{"BuildBundle", "build"},
		{"ownedField", "ownedField"},
		{"owned", "ownedField"},
		{"OwnedBundle", "ownedField"},
		{"user", "user"},
		{"UserBundle", "user"},
		{"unknown", "unknown"},
	}
	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			assert.Equal(t, tc.want, api.NormalizeBundlePathType(tc.input))
		})
	}
}

func TestCmd_FieldList(t *testing.T) {
	out := runCmd(t, "field", "list", "--format", "json")
	assert.NotEmpty(t, out)
}

func TestCmd_FieldBundleList(t *testing.T) {
	out := runCmd(t, "field", "bundle", "list", "--type", "enum", "--format", "json")
	assert.NotEmpty(t, out)
}

func TestCmd_FieldBundleList_PascalCaseType(t *testing.T) {
	// Should accept "EnumBundle" as alias for "enum"
	out := runCmd(t, "field", "bundle", "list", "--type", "EnumBundle", "--format", "json")
	assert.NotEmpty(t, out)
}

func TestCmd_FieldWorktimeGet(t *testing.T) {
	out := runCmd(t, "field", "worktime", "get", "--format", "json")
	assert.NotEmpty(t, out)
}

func TestCmd_FieldAdd(t *testing.T) {
	c := newTestClient(t)

	name := "TestField-" + strconv.FormatInt(time.Now().UnixMilli(), 36)
	out := runCmd(t, "field", "add", "--name", name, "--type", api.FieldTypeString, "--format", "json")
	assert.Contains(t, out, name)

	// cleanup: find and delete the field by name
	t.Cleanup(func() {
		fields, err := c.ListCustomFields()
		if err != nil {
			return
		}
		for _, f := range fields {
			if f.Name == name {
				_ = c.DeleteCustomField(f.ID)
				break
			}
		}
	})
}

func TestCmd_FieldDelete(t *testing.T) {
	c := newTestClient(t)

	name := "TestField-" + strconv.FormatInt(time.Now().UnixMilli(), 36)
	field, err := c.AddCustomField(name, api.FieldTypeString)
	require.NoError(t, err)
	require.NotEmpty(t, field.ID)

	runCmd(t, "field", "delete", field.ID)

	// YouTrack deletion may not be immediately reflected in listings; poll briefly.
	require.Eventually(t, func() bool {
		exists, err := fieldExistsInList(c, field.ID)
		return err == nil && !exists
	}, 5*time.Second, 300*time.Millisecond, "deleted field %s should not appear in list", field.ID)
}

func TestCmd_FieldPrivacy(t *testing.T) {
	c := newTestClient(t)

	name := "TestField-" + strconv.FormatInt(time.Now().UnixMilli(), 36)
	field, err := c.AddCustomField(name, api.FieldTypeString)
	require.NoError(t, err)
	require.NotEmpty(t, field.ID)
	t.Cleanup(func() { _ = c.DeleteCustomField(field.ID) })

	// setting --private should succeed with exit 0
	runCmd(t, "field", "privacy", field.ID, "--private")
}

func TestCmd_FieldPrototypes(t *testing.T) {
	requireE2EEnv(t)
	// YouTrack may transiently return HTTP 500 while a recently-deleted
	// field prototype is still referenced in its index; poll until clear.
	var out string
	require.Eventually(t, func() bool {
		var err error
		out, err = tryCmdNoFail(t, "field", "prototypes", "--format", "json")
		return err == nil
	}, 15*time.Second, 500*time.Millisecond, "field prototypes should return 200 once YouTrack index stabilizes")
	assert.Contains(t, out, "[")
}
