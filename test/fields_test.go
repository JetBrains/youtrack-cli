package api_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

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
