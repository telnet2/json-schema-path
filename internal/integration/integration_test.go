package integration

import (
	"testing"

	"jsonpath-sdk/internal/json"
	"jsonpath-sdk/internal/parser"
	"jsonpath-sdk/internal/tree"
)

// TestEndToEndPathMatching tests the complete pipeline from parsing to matching
func TestEndToEndPathMatching(t *testing.T) {
	tests := []struct {
		name       string
		expression string
		jsonData   string
		expected   bool
		desc       string
	}{
		{
			name:       "simple_property_match",
			expression: "$.user.name",
			jsonData:   `{"user": {"name": "John", "age": 30}}`,
			expected:   true,
			desc:       "Simple property access should match",
		},
		{
			name:       "array_index_match",
			expression: "$.users[0].name",
			jsonData:   `{"users": [{"name": "Alice"}, {"name": "Bob"}]}`,
			expected:   true,
			desc:       "Array index access should match",
		},
		{
			name:       "group_alternative_match",
			expression: "$.user.(name|email)",
			jsonData:   `{"user": {"name": "John", "id": 123}}`,
			expected:   true,
			desc:       "Group with alternative should match existing property",
		},
		{
			name:       "group_no_match",
			expression: "$.user.(email|phone)",
			jsonData:   `{"user": {"name": "John", "id": 123}}`,
			expected:   false,
			desc:       "Group with alternatives should not match if neither exists",
		},
		{
			name:       "nested_object_match",
			expression: "$.data.profile.settings.theme",
			jsonData:   `{"data": {"profile": {"settings": {"theme": "dark", "lang": "en"}}}}`,
			expected:   true,
			desc:       "Deep nested object access should match",
		},
		{
			name:       "bracket_notation_match",
			expression: "$.config[\"api-key\"]",
			jsonData:   `{"config": {"api-key": "secret123", "timeout": 30}}`,
			expected:   true,
			desc:       "Bracket notation with quoted key should match",
		},
		{
			name:       "complex_nested_arrays",
			expression: "$.departments[0].employees[1].contact.email",
			jsonData: `{
				"departments": [
					{
						"name": "Engineering",
						"employees": [
							{"name": "Alice", "contact": {"email": "alice@test.com"}},
							{"name": "Bob", "contact": {"email": "bob@test.com", "phone": "123"}}
						]
					}
				]
			}`,
			expected: true,
			desc:     "Complex nested array and object access should match",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse the expression
			expr, err := parser.ParseExpression(tt.expression)
			if err != nil {
				t.Fatalf("Failed to parse expression '%s': %v", tt.expression, err)
			}

			// Build pattern tree
			patternTree := tree.NewPatternTree()
			if err := patternTree.AddPattern(expr); err != nil {
				t.Fatalf("Failed to build pattern tree: %v", err)
			}

			// Extract paths from JSON
			processor := json.NewPathExtractor()
			if err := processor.ValidateJSON(tt.jsonData); err != nil {
				t.Fatalf("Invalid test JSON data: %v", err)
			}

			paths, err := processor.ExtractPaths(tt.jsonData)
			if err != nil {
				t.Fatalf("Failed to extract paths: %v", err)
			}

			// Check if any path matches
			found := false
			for _, path := range paths {
				segments := processor.ConvertPathToSegments(path)
				if patternTree.MatchPath(segments) {
					found = true
					t.Logf("Matched path: %s", path)
					break
				}
			}

			if found != tt.expected {
				t.Errorf("Expected match=%v for expression '%s' against JSON, got %v", 
					tt.expected, tt.expression, found)
				t.Logf("Extracted paths: %v", paths)
			}
		})
	}
}

// TestPerformancePathMatching benchmarks the pattern matching performance
func BenchmarkPathMatching(b *testing.B) {
	// Setup test data
	expression := "$.data.users[*].(name|email|profile.bio)"
	jsonData := `{
		"data": {
			"users": [
				{"name": "Alice", "email": "alice@test.com", "profile": {"bio": "Engineer"}},
				{"name": "Bob", "email": "bob@test.com", "profile": {"bio": "Designer"}},
				{"name": "Carol", "email": "carol@test.com", "profile": {"bio": "Manager"}}
			]
		}
	}`

	// Parse expression once
	expr, err := parser.ParseExpression(expression)
	if err != nil {
		b.Fatalf("Failed to parse expression: %v", err)
	}

	// Build pattern tree once
	patternTree := tree.NewPatternTree()
	if err := patternTree.AddPattern(expr); err != nil {
		b.Fatalf("Failed to build pattern tree: %v", err)
	}

	// Extract paths once
	processor := json.NewPathExtractor()
	paths, err := processor.ExtractPaths(jsonData)
	if err != nil {
		b.Fatalf("Failed to extract paths: %v", err)
	}

	// Benchmark the matching process
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, path := range paths {
			segments := processor.ConvertPathToSegments(path)
			patternTree.MatchPath(segments)
		}
	}
}

// TestLargeDatasetPerformance tests performance with larger datasets
func BenchmarkLargeDatasetMatching(b *testing.B) {
	// Create a larger JSON dataset
	jsonData := `{
		"users": [`

	// Generate 100 user entries
	for i := 0; i < 100; i++ {
		if i > 0 {
			jsonData += ","
		}
		jsonData += `{
			"id": ` + string(rune('0'+i%10)) + `,
			"name": "User` + string(rune('0'+i%10)) + `",
			"email": "user` + string(rune('0'+i%10)) + `@test.com",
			"profile": {
				"bio": "Bio for user ` + string(rune('0'+i%10)) + `",
				"settings": {"theme": "dark", "notifications": true}
			}
		}`
	}
	jsonData += `]}`

	expression := "$.users[*].profile.settings.theme"
	
	// Setup
	expr, _ := parser.ParseExpression(expression)
	patternTree := tree.NewPatternTree()
	patternTree.AddPattern(expr)
	processor := json.NewPathExtractor()
	paths, _ := processor.ExtractPaths(jsonData)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		matchCount := 0
		for _, path := range paths {
			segments := processor.ConvertPathToSegments(path)
			if patternTree.MatchPath(segments) {
				matchCount++
			}
		}
		// Ensure the compiler doesn't optimize away the work
		_ = matchCount
	}
}

// TestErrorHandling tests error conditions across components
func TestErrorHandling(t *testing.T) {
	tests := []struct {
		name        string
		expression  string
		jsonData    string
		expectError bool
		desc        string
	}{
		{
			name:        "invalid_expression_syntax",
			expression:  "$.user.(name|",
			jsonData:    `{"user": {"name": "John"}}`,
			expectError: true,
			desc:        "Invalid expression syntax should cause parse error",
		},
		{
			name:        "invalid_json_data",
			expression:  "$.user.name",
			jsonData:    `{"user": {"name": "John"`,
			expectError: true,
			desc:        "Invalid JSON data should cause validation error",
		},
		{
			name:        "empty_json",
			expression:  "$.user.name",
			jsonData:    `{}`,
			expectError: false,
			desc:        "Empty JSON should not cause error, just no matches",
		},
		{
			name:        "null_json",
			expression:  "$.user.name",
			jsonData:    `null`,
			expectError: false,
			desc:        "Null JSON should not cause error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test parser
			expr, parseErr := parser.ParseExpression(tt.expression)
			if tt.expectError && parseErr == nil {
				t.Errorf("Expected parse error but got none")
				return
			}
			if !tt.expectError && parseErr != nil {
				t.Errorf("Unexpected parse error: %v", parseErr)
				return
			}

			if parseErr != nil {
				return // Skip rest if parse failed as expected
			}

			// Test JSON validation
			processor := json.NewPathExtractor()
			validateErr := processor.ValidateJSON(tt.jsonData)
			if tt.expectError && validateErr == nil {
				t.Errorf("Expected JSON validation error but got none")
				return
			}
			if !tt.expectError && validateErr != nil {
				t.Errorf("Unexpected JSON validation error: %v", validateErr)
				return
			}

			if validateErr != nil {
				return // Skip rest if validation failed as expected
			}

			// Test path extraction (should not error even with unusual JSON)
			paths, extractErr := processor.ExtractPaths(tt.jsonData)
			if extractErr != nil {
				t.Errorf("Unexpected path extraction error: %v", extractErr)
				return
			}

			// Test pattern tree building
			patternTree := tree.NewPatternTree()
			if err := patternTree.AddPattern(expr); err != nil {
				t.Errorf("Unexpected pattern tree error: %v", err)
				return
			}

			// Test matching (should not error)
			for _, path := range paths {
				segments := processor.ConvertPathToSegments(path)
				patternTree.MatchPath(segments) // Should not panic or error
			}
		})
	}
}

// TestSpecificationCompliance tests compliance with the formal specification
func TestSpecificationCompliance(t *testing.T) {
	specTests := []struct {
		name       string
		expression string
		desc       string
	}{
		{
			name:       "root_required",
			expression: "$.data",
			desc:       "Root ($) is required at start of expression",
		},
		{
			name:       "property_access",
			expression: "$.user.name",
			desc:       "Property access with dot notation",
		},
		{
			name:       "bracket_access",
			expression: "$.user[\"name\"]",
			desc:       "Bracket notation with quoted string",
		},
		{
			name:       "array_index",
			expression: "$.users[0]",
			desc:       "Array index access",
		},
		{
			name:       "group_alternatives",
			expression: "$.user.(name|email)",
			desc:       "Group with pipe-separated alternatives",
		},
		{
			name:       "group_repetition",
			expression: "$.data.(child|meta.child){*}",
			desc:       "Group with repetition operator",
		},
		{
			name:       "nested_groups",
			expression: "$.node.(child|meta.(name|id)).value",
			desc:       "Nested group expressions",
		},
		{
			name:       "complex_pattern",
			expression: "$.api.responses[*].(data|error).items[0].(id|name)",
			desc:       "Complex pattern with multiple constructs",
		},
	}

	for _, tt := range specTests {
		t.Run(tt.name, func(t *testing.T) {
			// Should be able to parse according to specification
			expr, err := parser.ParseExpression(tt.expression)
			if err != nil {
				t.Errorf("Failed to parse spec-compliant expression '%s': %v", tt.expression, err)
				return
			}

			// Should be able to build pattern tree
			patternTree := tree.NewPatternTree()
			if err := patternTree.AddPattern(expr); err != nil {
				t.Errorf("Failed to build pattern tree for spec-compliant expression '%s': %v", tt.expression, err)
			}

			// Expression should round-trip through String() method
			roundTrip := expr.String()
			if roundTrip == "" {
				t.Errorf("Empty string representation for expression '%s'", tt.expression)
			}

			t.Logf("Expression: %s -> AST: %s", tt.expression, roundTrip)
		})
	}
}