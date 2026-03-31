package api

import "strings"

// Field type IDs as accepted by the YouTrack REST API fieldType.id field.
// Simple types have a plain string; bundle-based types encode cardinality
// with a [1] (single-value) or [*] (multi-value) suffix.
const (
	// Simple (non-bundle) field types
	FieldTypeString   = "string"
	FieldTypeText     = "text"
	FieldTypeInteger  = "integer"
	FieldTypeFloat    = "float"
	FieldTypeDate     = "date"
	FieldTypeDateTime = "date and time"
	FieldTypePeriod   = "period"

	// Bundle-based field types — single-value
	FieldTypeEnumSingle    = "enum[1]"
	FieldTypeStateSingle   = "state[1]"
	FieldTypeVersionSingle = "version[1]"
	FieldTypeBuildSingle   = "build[1]"
	FieldTypeOwnedSingle   = "owned[1]"
	FieldTypeUserSingle    = "user[1]"

	// Bundle-based field types — multi-value
	FieldTypeEnumMulti    = "enum[*]"
	FieldTypeStateMulti   = "state[*]"
	FieldTypeVersionMulti = "version[*]"
	FieldTypeBuildMulti   = "build[*]"
	FieldTypeOwnedMulti   = "owned[*]"
	FieldTypeUserMulti    = "user[*]"
)

// AllFieldTypes lists every valid field type ID for use in help text and validation.
var AllFieldTypes = []string{
	FieldTypeString, FieldTypeText, FieldTypeInteger, FieldTypeFloat,
	FieldTypeDate, FieldTypeDateTime, FieldTypePeriod,
	FieldTypeEnumSingle, FieldTypeEnumMulti,
	FieldTypeStateSingle, FieldTypeStateMulti,
	FieldTypeVersionSingle, FieldTypeVersionMulti,
	FieldTypeBuildSingle, FieldTypeBuildMulti,
	FieldTypeOwnedSingle, FieldTypeOwnedMulti,
	FieldTypeUserSingle, FieldTypeUserMulti,
}

// BundlePathForFieldType returns the bundle URL path segment (e.g. "enum") for
// bundle-based field types, and false for simple types that have no bundle.
// FieldTypeEnumSingle ("enum[1]") → ("enum", true), FieldTypeString → ("", false).
func BundlePathForFieldType(fieldType string) (string, bool) {
	prefix, _, isBundled := strings.Cut(fieldType, "[")
	if !isBundled {
		return "", false
	}
	switch prefix {
	case "enum", "state", "version", "build", "owned", "user":
		return prefix, true
	default:
		return "", false
	}
}
