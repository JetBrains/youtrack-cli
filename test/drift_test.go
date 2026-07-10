// Drift detection catches when YouTrack's REST API contract changes underneath us.
//
// TestAPIDrift parses the Go structs in api/*.go, collects their declared json
// field names, fetches the live server's OpenAPI 3.0 spec, and asserts every
// field we declare exists in the corresponding schema (directly or via allOf
// composition). It intentionally validates only a curated allowlist of types
// (typeMapping) — many of our Go structs are simplified request bodies, or
// read from YouTrack's discriminator-based polymorphic schemas where the base
// definition in the spec doesn't carry every field a concrete subtype returns.
// Only types verified to map cleanly onto a single schema are included.
//
// Run with: go test ./test/ -run TestAPIDrift -v
package api_test

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"testing"
	"time"
)

// OpenAPISpec represents the relevant parts of YouTrack's OpenAPI 3.0 document.
type OpenAPISpec struct {
	Components struct {
		Schemas map[string]OpenAPISchema `json:"schemas"`
	} `json:"components"`
}

// OpenAPISchema represents a schema definition, including allOf composition
// used throughout YouTrack's spec for discriminator-based inheritance.
type OpenAPISchema struct {
	Type       string                   `json:"type"`
	Properties map[string]OpenAPISchema `json:"properties"`
	AllOf      []OpenAPISchema          `json:"allOf,omitempty"`
	Ref        string                   `json:"$ref,omitempty"`
}

// GoStruct is a parsed Go struct declaration.
type GoStruct struct {
	Name   string
	Fields []GoField
}

// GoField is a single field of a parsed Go struct.
type GoField struct {
	Name     string
	JSONName string
}

// hubBackedStructs lists Go structs that decode from YouTrack endpoints backed
// by JetBrains Hub: /api/roles, /api/assignedRoles, and /api/admin/organizations
// proxy Hub entities (roles, role assignments, and organizations are owned by
// the bundled Hub, not YouTrack proper). Some YouTrack builds omit these
// schemas from /api/openapi.json entirely (the 2026.1 Docker image used in CI
// ships 218 schemas without them) while newer cloud builds document them
// (youtrack.jetbrains.com ships 232 schemas including all three). When the
// schema is missing from the fetched spec we skip with the reason below; when
// the schema is present, these structs are validated like any other.
var hubBackedStructs = map[string]string{
	"Role":         "Hub REST entity proxied via /api/roles — not in this build's YouTrack OpenAPI spec",
	"AssignedRole": "Hub REST entity proxied via /api/assignedRoles — not in this build's YouTrack OpenAPI spec",
	"Organization": "Hub REST entity proxied via /api/admin/organizations — not in this build's YouTrack OpenAPI spec",
}

// allowedMissingFields permits specific fields to be absent from a schema
// without failing, keyed by schema name then json field name. Each entry
// documents why the omission is acceptable rather than drift.
var allowedMissingFields = map[string]map[string]string{
	"User": {
		// The helpdesk standard/agent/reporter user type. Actively exercised:
		// every user fetch requests userType(id,name) (userFields in
		// api/users.go), `user create` always sends it (cmd/user/create.go),
		// and helpdesk agent auto-add branches on UserType.ID == "AGENT"
		// (api/projects.go). Live instances serve it, but some builds'
		// OpenAPI spec omits it from the User schema (present in newer cloud
		// specs) — a spec omission, not drift.
		"userType": "served by live instances and used by the CLI, but omitted from some builds' User schema",
	},
}

// typeMapping maps our Go struct names to YouTrack OpenAPI schema names.
// Only entries verified field-for-field against a live YouTrack instance are
// listed; everything else is intentionally skipped (see the package doc above).
var typeMapping = map[string]string{
	"Article":            "Article",
	"ArticleAttachment":  "ArticleAttachment",
	"AssignedRole":       "AssignedRole",
	"Board":              "Agile",
	"Comment":            "IssueComment",
	"CustomField":        "CustomField",
	"ExternalIssue":      "ExternalIssue",
	"FieldType":          "FieldType",
	"Group":              "UserGroup",
	"Issue":              "Issue",
	"IssueAttachment":    "IssueAttachment",
	"IssueLink":          "IssueLink",
	"IssueLinkType":      "IssueLinkType",
	"Organization":       "Organization",
	"Project":            "Project",
	"ProjectCustomField": "ProjectCustomField",
	"Reaction":           "Reaction",
	"Role":               "Role",
	"SavedSearch":        "SavedQuery",
	"Sprint":             "Sprint",
	"Tag":                "Tag",
	"User":               "User",
	"WorkItem":           "IssueWorkItem",
}

func TestAPIDrift(t *testing.T) {
	requireE2EEnv(t)
	baseURL := os.Getenv("YOUTRACK_BASE_URL")

	spec, err := fetchOpenAPISpec(baseURL)
	if err != nil {
		t.Fatalf("failed to fetch OpenAPI spec: %v", err)
	}
	t.Logf("fetched OpenAPI spec from %s (%d schemas)", baseURL, len(spec.Components.Schemas))

	goTypes, err := parseGoStructs("../api")
	if err != nil {
		t.Fatalf("failed to parse api/*.go: %v", err)
	}
	t.Logf("parsed %d Go structs from api/*.go", len(goTypes))

	for _, gs := range goTypes {
		schemaName, ok := typeMapping[gs.Name]
		if !ok {
			continue
		}

		t.Run(gs.Name, func(t *testing.T) {
			schema, ok := spec.Components.Schemas[schemaName]
			if !ok {
				if reason, hubBacked := hubBackedStructs[gs.Name]; hubBacked {
					t.Skipf("schema %q not in this instance's OpenAPI spec: %s", schemaName, reason)
				}
				t.Fatalf("schema %q not found in OpenAPI spec", schemaName)
			}

			props := allSchemaProperties(spec, schema, map[string]bool{})
			for _, field := range gs.Fields {
				if schemaHasProperty(props, field.JSONName) {
					continue
				}
				if reason, allowed := allowedMissingFields[schemaName][field.JSONName]; allowed {
					t.Logf("allowing missing field %s.%s: %s", schemaName, field.Name, reason)
					continue
				}
				t.Errorf("field %s (json: %s) not found in schema %s", field.Name, field.JSONName, schemaName)
			}
		})
	}
}

func fetchOpenAPISpec(baseURL string) (*OpenAPISpec, error) {
	url := strings.TrimRight(baseURL, "/") + "/api/openapi.json"

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d fetching %s", resp.StatusCode, url)
	}

	var spec OpenAPISpec
	if err := json.NewDecoder(resp.Body).Decode(&spec); err != nil {
		return nil, fmt.Errorf("decoding openapi.json: %w", err)
	}
	return &spec, nil
}

// parseGoStructs parses every *.go file (excluding tests) in dir and collects
// exported struct fields that carry a json tag.
func parseGoStructs(dir string) ([]GoStruct, error) {
	files, err := filepath.Glob(filepath.Join(dir, "*.go"))
	if err != nil {
		return nil, err
	}

	var structs []GoStruct
	fset := token.NewFileSet()
	for _, file := range files {
		if strings.HasSuffix(file, "_test.go") {
			continue
		}

		node, err := parser.ParseFile(fset, file, nil, parser.ParseComments)
		if err != nil {
			return nil, fmt.Errorf("parsing %s: %w", file, err)
		}

		ast.Inspect(node, func(n ast.Node) bool {
			typeSpec, ok := n.(*ast.TypeSpec)
			if !ok {
				return true
			}
			structType, ok := typeSpec.Type.(*ast.StructType)
			if !ok {
				return true
			}

			gs := GoStruct{Name: typeSpec.Name.Name}
			for _, field := range structType.Fields.List {
				if len(field.Names) == 0 || !field.Names[0].IsExported() {
					continue // skip embedded and unexported fields
				}

				gf := GoField{Name: field.Names[0].Name}
				if field.Tag != nil {
					tag := reflect.StructTag(strings.Trim(field.Tag.Value, "`"))
					if jsonTag := tag.Get("json"); jsonTag != "" {
						name := strings.Split(jsonTag, ",")[0]
						if name != "" && name != "-" {
							gf.JSONName = name
						}
					}
				}
				if gf.JSONName != "" {
					gs.Fields = append(gs.Fields, gf)
				}
			}

			if len(gs.Fields) > 0 {
				structs = append(structs, gs)
			}
			return true
		})
	}

	return structs, nil
}

// allSchemaProperties flattens a schema's own properties together with those
// contributed by its allOf composition (including $ref'd base schemas),
// which is how YouTrack's OpenAPI spec models inheritance.
func allSchemaProperties(spec *OpenAPISpec, schema OpenAPISchema, seen map[string]bool) map[string]OpenAPISchema {
	props := make(map[string]OpenAPISchema, len(schema.Properties))
	for k, v := range schema.Properties {
		props[k] = v
	}

	for _, sub := range schema.AllOf {
		if sub.Ref != "" {
			name := refSchemaName(sub.Ref)
			if name == "" || seen[name] {
				continue
			}
			seen[name] = true
			if refSchema, ok := spec.Components.Schemas[name]; ok {
				for k, v := range allSchemaProperties(spec, refSchema, seen) {
					props[k] = v
				}
			}
		}
		for k, v := range sub.Properties {
			props[k] = v
		}
	}

	return props
}

var schemaRefPattern = regexp.MustCompile(`^#/components/schemas/(.+)$`)

func refSchemaName(ref string) string {
	if m := schemaRefPattern.FindStringSubmatch(ref); len(m) > 1 {
		return m[1]
	}
	return ""
}

func schemaHasProperty(props map[string]OpenAPISchema, jsonName string) bool {
	if _, ok := props[jsonName]; ok {
		return true
	}
	lower := strings.ToLower(jsonName)
	for k := range props {
		if strings.ToLower(k) == lower {
			return true
		}
	}
	return false
}
