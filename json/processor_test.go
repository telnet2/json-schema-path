package json

import (
	"reflect"
	"testing"

	"jsonpath-sdk/spec"
)

func TestConvertPathToSegments(t *testing.T) {
	pe := NewPathExtractor()

	tests := []struct {
		name     string
		path     string
		expected []spec.PathSegment
	}{
		{
			name:     "simple property path",
			path:     "$.user.name",
			expected: []spec.PathSegment{spec.NewPropertySegment("user"), spec.NewPropertySegment("name")},
		},
		{
			name:     "array index path",
			path:     "$.users[0].name",
			expected: []spec.PathSegment{spec.NewPropertySegment("users"), spec.NewArrayIndexSegment(0), spec.NewPropertySegment("name")},
		},
		{
			name: "multiple array indices",
			path: "$.data.users[0].contacts[1].email",
			expected: []spec.PathSegment{
				spec.NewPropertySegment("data"),
				spec.NewPropertySegment("users"),
				spec.NewArrayIndexSegment(0),
				spec.NewPropertySegment("contacts"),
				spec.NewArrayIndexSegment(1),
				spec.NewPropertySegment("email"),
			},
		},
		{
			name:     "quoted bracket property",
			path:     `$.data["property-name"].value`,
			expected: []spec.PathSegment{spec.NewPropertySegment("data"), spec.NewPropertySegment("property-name"), spec.NewPropertySegment("value")},
		},
		{
			name:     "root only",
			path:     "$",
			expected: []spec.PathSegment{},
		},
		{
			name:     "single property",
			path:     "$.name",
			expected: []spec.PathSegment{spec.NewPropertySegment("name")},
		},
		{
			name: "complex nested",
			path: "$.api.responses[0].data.users[2].profile",
			expected: []spec.PathSegment{
				spec.NewPropertySegment("api"),
				spec.NewPropertySegment("responses"),
				spec.NewArrayIndexSegment(0),
				spec.NewPropertySegment("data"),
				spec.NewPropertySegment("users"),
				spec.NewArrayIndexSegment(2),
				spec.NewPropertySegment("profile"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := pe.ConvertPathToSegments(tt.path)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("ConvertPathToSegments(%s) = %v, expected %v", tt.path, result, tt.expected)
			}
		})
	}
}

func TestExtractPaths(t *testing.T) {
	pe := NewPathExtractor()

	jsonData := `{
		"user": {
			"name": "John",
			"age": 30,
			"contacts": [
				{"type": "email", "value": "john@example.com"},
				{"type": "phone", "value": "123-456-7890"}
			]
		}
	}`

	paths, err := pe.ExtractPaths(jsonData)
	if err != nil {
		t.Fatalf("ExtractPaths failed: %v", err)
	}

	expectedPaths := []string{
		"$",
		"$.user",
		"$.user.name",
		"$.user.age",
		"$.user.contacts",
		"$.user.contacts[0]",
		"$.user.contacts[0].type",
		"$.user.contacts[0].value",
		"$.user.contacts[1]",
		"$.user.contacts[1].type",
		"$.user.contacts[1].value",
	}

	if len(paths) != len(expectedPaths) {
		t.Errorf("Expected %d paths, got %d", len(expectedPaths), len(paths))
	}

	// Check that all expected paths are present
	pathMap := make(map[string]bool)
	for _, path := range paths {
		pathMap[path] = true
	}

	for _, expected := range expectedPaths {
		if !pathMap[expected] {
			t.Errorf("Expected path '%s' not found in extracted paths", expected)
		}
	}
}

func TestValidateJSON(t *testing.T) {
	pe := NewPathExtractor()

	tests := []struct {
		name      string
		jsonData  string
		expectErr bool
	}{
		{
			name:      "valid JSON object",
			jsonData:  `{"name": "John", "age": 30}`,
			expectErr: false,
		},
		{
			name:      "valid JSON array",
			jsonData:  `[1, 2, 3, "test"]`,
			expectErr: false,
		},
		{
			name:      "invalid JSON - missing quote",
			jsonData:  `{"name: "John"}`,
			expectErr: true,
		},
		{
			name:      "invalid JSON - trailing comma",
			jsonData:  `{"name": "John",}`,
			expectErr: true,
		},
		{
			name:      "empty string",
			jsonData:  ``,
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := pe.ValidateJSON(tt.jsonData)
			if (err != nil) != tt.expectErr {
				t.Errorf("ValidateJSON(%s) error = %v, expectErr = %v", tt.jsonData, err, tt.expectErr)
			}
		})
	}
}

func TestExtractValue(t *testing.T) {
	pe := NewPathExtractor()

	jsonData := `{
                "user": {
			"name": "John",
			"contacts": [
				{"type": "email", "value": "john@example.com"}
			]
		}
	}`

	tests := []struct {
		name      string
		path      string
		expected  interface{}
		expectErr bool
	}{
		{
			name:      "simple property",
			path:      "$.user.name",
			expected:  "John",
			expectErr: false,
		},
		{
			name:      "nested object",
			path:      "$.user",
			expected:  map[string]interface{}{"name": "John", "contacts": []interface{}{map[string]interface{}{"type": "email", "value": "john@example.com"}}},
			expectErr: false,
		},
		{
			name:      "array element property",
			path:      "$.user.contacts[0].type",
			expected:  "email",
			expectErr: false,
		},
		{
			name:      "non-existent property",
			path:      "$.user.nonexistent",
			expected:  nil,
			expectErr: true,
		},
		{
			name:      "invalid array index",
			path:      "$.user.contacts[5].type",
			expected:  nil,
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := pe.ExtractValue(jsonData, tt.path)
			if (err != nil) != tt.expectErr {
				t.Errorf("ExtractValue(%s) error = %v, expectErr = %v", tt.path, err, tt.expectErr)
				return
			}
			if !tt.expectErr && !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("ExtractValue(%s) = %v, expected %v", tt.path, result, tt.expected)
			}
		})
	}
}

func TestExtractSchemaPaths(t *testing.T) {
	pe := NewPathExtractor()

	schema := `{
                "type": "object",
                "properties": {
                        "name": {"type": "string"},
                        "address": {
                                "type": "object",
                                "properties": {
                                        "street": {"type": "string"},
                                        "phones": {
                                                "type": "array",
                                                "items": {
                                                        "type": "object",
                                                        "properties": {
                                                                "type": {"type": "string"}
                                                        }
                                                }
                                        }
                                }
                        }
                }
        }`

	paths, err := pe.ExtractSchemaPaths(schema)
	if err != nil {
		t.Fatalf("ExtractSchemaPaths failed: %v", err)
	}

	expected := []string{
		"$",
		"$.address",
		"$.address.phones",
		"$.address.phones[*]",
		"$.address.phones[*].type",
		"$.address.street",
		"$.name",
	}

	if len(paths) != len(expected) {
		t.Fatalf("expected %d paths, got %d: %v", len(expected), len(paths), paths)
	}

	pathSet := make(map[string]bool)
	for _, p := range paths {
		pathSet[p] = true
	}

	for _, exp := range expected {
		if !pathSet[exp] {
			t.Errorf("expected schema path %s not found", exp)
		}
	}
}

func TestExtractSchemaPathsRecursiveSchema(t *testing.T) {
	pe := NewPathExtractor()

	schema := `{
                "type": "object",
                "properties": {
                        "value": {"type": "string"},
                        "child": {"$ref": "#"}
                }
        }`

	paths, err := pe.ExtractSchemaPaths(schema)
	if err != nil {
		t.Fatalf("ExtractSchemaPaths failed: %v", err)
	}

	expected := []string{
		"$",
		"$.child{*}",
		"$.child{*}.value",
		"$.value",
	}

	if len(paths) != len(expected) {
		t.Fatalf("expected %d paths, got %d: %v", len(expected), len(paths), paths)
	}

	pathSet := make(map[string]bool, len(paths))
	for _, p := range paths {
		pathSet[p] = true
	}

	for _, exp := range expected {
		if !pathSet[exp] {
			t.Errorf("expected schema path %s not found", exp)
		}
	}

	if pathSet["$.child.child"] {
		t.Errorf("unexpected recursive path expansion included")
	}
	if pathSet["$.child"] {
		t.Errorf("expected $.child to be consolidated into $.child{*}")
	}
	if pathSet["$.child.value"] {
		t.Errorf("expected $.child.value to be consolidated into $.child{*}.value")
	}
}

func TestExtractSchemaPathsTerminalsOnly(t *testing.T) {
	pe := NewPathExtractor()

	schema := `{
                "type": "object",
                "properties": {
                        "name": {"type": "string"},
                        "address": {
                                "type": "object",
                                "properties": {
                                        "street": {"type": "string"},
                                        "phones": {
                                                "type": "array",
                                                "items": {
                                                        "type": "object",
                                                        "properties": {
                                                                "type": {"type": "string"}
                                                        }
                                                }
                                        }
                                }
                        }
                }
        }`

	opts := SchemaPathOptions{TerminalsOnly: true}
	paths, err := pe.ExtractSchemaPathsWithOptions(schema, defaultSchemaURL, opts)
	if err != nil {
		t.Fatalf("ExtractSchemaPathsWithOptions failed: %v", err)
	}

	expected := []string{
		"$.address.phones[*].type",
		"$.address.street",
		"$.name",
	}

	if len(paths) != len(expected) {
		t.Fatalf("expected %d paths, got %d: %v", len(expected), len(paths), paths)
	}

	pathSet := make(map[string]bool)
	for _, p := range paths {
		pathSet[p] = true
	}

	for _, exp := range expected {
		if !pathSet[exp] {
			t.Errorf("expected schema path %s not found", exp)
		}
	}

	disallowed := []string{"$", "$.address", "$.address.phones", "$.address.phones[*]"}
	for _, d := range disallowed {
		if pathSet[d] {
			t.Errorf("did not expect non-terminal path %s", d)
		}
	}
}
