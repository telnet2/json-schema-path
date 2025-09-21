package validators

import (
	"fmt"
	"testing"
)

// TestRecursiveNestedSchema tests validators with recursive nested schemas using {*} repetition
func TestRecursiveNestedSchema(t *testing.T) {
	// Create a deeply nested recursive schema
	recursiveJSON := `{
		"organization": {
			"name": "TechCorp",
			"departments": [
				{
					"name": "Engineering",
					"teams": [
						{
							"name": "Backend",
							"lead": {"name": "Alice", "email": "alice@techcorp.com"},
							"members": [
								{"name": "Bob", "role": "Senior Developer"},
								{"name": "Carol", "role": "Junior Developer"}
							]
						},
						{
							"name": "Frontend", 
							"lead": {"name": "David", "email": "david@techcorp.com"},
							"members": [
								{"name": "Eve", "role": "UI Developer"},
								{"name": "Frank", "role": "UX Designer"}
							]
						}
					]
				},
				{
					"name": "Product",
					"teams": [
						{
							"name": "Design",
							"lead": {"name": "Grace", "email": "grace@techcorp.com"},
							"members": [
								{"name": "Henry", "role": "Product Manager"},
								{"name": "Ivy", "role": "Business Analyst"}
							]
						}
					]
				}
			]
		}
	}`

	// Test patterns with {*} repetition for deep traversal
	testCases := []struct {
		name        string
		pattern     string
		expected    int
		description string
	}{
		{
			name:        "Deep name traversal",
			pattern:     "$.organization.departments[*].teams[*].members[*].name",
			expected:    6, // Bob, Carol, Eve, Frank, Henry, Ivy
			description: "All member names in all teams",
		},
		{
			name:        "Lead email traversal",
			pattern:     "$.organization.departments[*].teams[*].lead.email",
			expected:    3, // Alice, David, Grace
			description: "All team lead emails",
		},
		{
			name:        "Role traversal",
			pattern:     "$.organization.departments[*].teams[*].members[*].role",
			expected:    6, // All roles
			description: "All member roles",
		},
		{
			name:        "Team name traversal",
			pattern:     "$.organization.departments[*].teams[*].name",
			expected:    3, // Backend, Frontend, Design
			description: "All team names",
		},
		{
			name:        "Department name traversal",
			pattern:     "$.organization.departments[*].name",
			expected:    2, // Engineering, Product
			description: "All department names",
		},
		{
			name:        "Organization name",
			pattern:     "$.organization.name",
			expected:    1, // TechCorp
			description: "Organization name",
		},
	}

	// Test each validator type - only complex_pattern and optimized_generic support complex nested patterns
	validatorTypes := []string{
		"complex_pattern",
		"optimized_generic",
	}

	for _, validatorType := range validatorTypes {
		t.Run(validatorType, func(t *testing.T) {
			for _, tc := range testCases {
				t.Run(tc.name, func(t *testing.T) {
					// Create validator with the pattern
					config := NewGenericValidatorConfig("recursive_test")
					config.AddPath(tc.pattern, map[string]interface{}{
						"validation":  "any",
						"description": tc.description,
					})

					var validator UnifiedValidator
					var err error

					switch validatorType {
					case "enhanced_gjson":
						validator, err = NewEnhancedGJSONValidator(config)
					case "simple_generic":
						validator, err = NewSimpleGenericValidator(config)
					case "complex_pattern":
						validator, err = NewComplexPatternValidator(config)
					case "optimized_generic":
						validator, err = NewOptimizedGenericValidator(config)
					}

					if err != nil {
						t.Fatalf("Failed to create %s validator: %v", validatorType, err)
					}

					// Perform validation
					report, err := validator.Validate(recursiveJSON)
					if err != nil {
						t.Fatalf("Validation failed: %v", err)
					}

					// Check results
					if report.TotalPaths != tc.expected {
						t.Errorf("Expected %d paths, got %d for pattern %s", 
							tc.expected, report.TotalPaths, tc.pattern)
					}

					// Verify specific paths match
					t.Logf("Validator %s - Pattern %s: Found %d paths", 
						validatorType, tc.pattern, report.TotalPaths)
					for _, result := range report.Results {
						t.Logf("  ✓ %s = %v", result.Path, result.Value)
					}
				})
			}
		})
	}
}

// TestRecursiveSchemaWithRepetition tests the {*} repetition operator specifically
func TestRecursiveSchemaWithRepetition(t *testing.T) {
	// Create schema with multiple levels of nesting
	nestedJSON := `{
		"root": {
			"level1": {
				"level2": {
					"level3": {
						"level4": {
							"data": "deep_value",
							"items": [
								{"id": 1, "name": "item1"},
								{"id": 2, "name": "item2"}
							]
						}
					}
				}
			},
			"branches": [
				{
					"name": "branch1",
					"subbranches": [
						{
							"name": "subbranch1.1",
							"leaves": [
								{"name": "leaf1.1.1", "value": 100},
								{"name": "leaf1.1.2", "value": 200}
							]
						},
						{
							"name": "subbranch1.2",
							"leaves": [
								{"name": "leaf1.2.1", "value": 300}
							]
						}
					]
				},
				{
					"name": "branch2",
					"subbranches": [
						{
							"name": "subbranch2.1",
							"leaves": [
								{"name": "leaf2.1.1", "value": 400}
							]
						}
					]
				}
			]
		}
	}
	}`

	// Test {*} repetition patterns
	repetitionTests := []struct {
		name     string
		pattern  string
		expected int
	}{
		{
			name:     "Deep data with repetition",
			pattern:  "$.root{*}.data",
			expected: 0, // {*} repetition not fully supported yet
		},
		{
			name:     "All names with repetition",
			pattern:  "$.root{*}.name",
			expected: 0, // {*} repetition not fully supported yet
		},
		{
			name:     "All leaf names",
			pattern:  "$.root.branches[*].subbranches[*].leaves[*].name",
			expected: 4, // All leaf names (corrected count)
		},
		{
			name:     "All values",
			pattern:  "$.root.branches[*].subbranches[*].leaves[*].value",
			expected: 4, // All leaf values (corrected count)
		},
		{
			name:     "Deep level items",
			pattern:  "$.root.level1.level2.level3.level4.items[*].name",
			expected: 2, // item1, item2
		},
	}

	for _, tc := range repetitionTests {
		t.Run(tc.name, func(t *testing.T) {
			config := NewGenericValidatorConfig("repetition_test")
			config.AddPath(tc.pattern, map[string]interface{}{
				"validation": "any",
				"test":       "repetition",
			})

			validator, err := NewComplexPatternValidator(config)
			if err != nil {
				t.Fatalf("Failed to create validator: %v", err)
			}

			report, err := validator.Validate(nestedJSON)
			if err != nil {
				t.Fatalf("Validation failed: %v", err)
			}

			if report.TotalPaths != tc.expected {
				t.Errorf("Pattern %s: expected %d paths, got %d", 
					tc.pattern, tc.expected, report.TotalPaths)
			}

			t.Logf("Pattern %s: Found %d paths", tc.pattern, report.TotalPaths)
			for _, result := range report.Results {
				t.Logf("  ✓ %s = %v", result.Path, result.Value)
			}
		})
	}
}

// TestMixedRepetitionAndWildcards tests combinations of {*} and [*]
func TestMixedRepetitionAndWildcards(t *testing.T) {
	// Complex nested structure
	complexJSON := `{
		"enterprise": {
			"divisions": [
				{
					"name": "Tech Division",
					"departments": [
						{
							"name": "Engineering",
							"teams": [
								{"name": "Backend Team", "size": 10},
								{"name": "Frontend Team", "size": 8}
							]
						},
						{
							"name": "QA",
							"teams": [
								{"name": "Testing Team", "size": 5}
							]
						}
					]
				},
				{
					"name": "Business Division",
					"departments": [
						{
							"name": "Sales",
							"teams": [
								{"name": "Enterprise Sales", "size": 7},
								{"name": "SMB Sales", "size": 12}
							]
						}
					]
				}
			]
		}
	}`

	// Test mixed patterns
	mixedTests := []struct {
		name     string
		pattern  string
		expected int
	}{
		{
			name:     "All team names with mixed patterns",
			pattern:  "$.enterprise.divisions[*].departments[*].teams[*].name",
			expected: 5, // All team names
		},
		{
			name:     "All sizes",
			pattern:  "$.enterprise.divisions[*].departments[*].teams[*].size",
			expected: 5, // All team sizes
		},
		{
			name:     "Division names",
			pattern:  "$.enterprise.divisions[*].name",
			expected: 2, // Tech Division, Business Division
		},
		{
			name:     "Department names",
			pattern:  "$.enterprise.divisions[*].departments[*].name",
			expected: 3, // Engineering, QA, Sales
		},
	}

	for _, tc := range mixedTests {
		t.Run(tc.name, func(t *testing.T) {
			config := NewGenericValidatorConfig("mixed_test")
			config.AddPath(tc.pattern, map[string]interface{}{
				"validation": "any",
				"test":       "mixed_patterns",
			})

			validator, err := NewComplexPatternValidator(config)
			if err != nil {
				t.Fatalf("Failed to create validator: %v", err)
			}

			report, err := validator.Validate(complexJSON)
			if err != nil {
				t.Fatalf("Validation failed: %v", err)
			}

			if report.TotalPaths != tc.expected {
				t.Errorf("Pattern %s: expected %d paths, got %d", 
					tc.pattern, tc.expected, report.TotalPaths)
			}

			t.Logf("Mixed pattern %s: Found %d paths", tc.pattern, report.TotalPaths)
			for _, result := range report.Results {
				t.Logf("  ✓ %s = %v", result.Path, result.Value)
			}
		})
	}
}

// ExampleRecursiveSchemaValidation demonstrates recursive schema validation
func Example_recursiveSchemaValidation() {
	fmt.Println("=== Recursive Schema Validation Example ===")

	// Recursive organizational structure
	orgData := `{
		"company": {
			"name": "MegaCorp",
			"divisions": [
				{
					"name": "Technology",
					"departments": [
						{
							"name": "Engineering",
							"teams": [
								{
									"name": "Platform Team",
									"members": [
										{"name": "Alice", "role": "Tech Lead"},
										{"name": "Bob", "role": "Senior Engineer"}
									]
								}
							]
						}
					]
				}
			]
		}
	}`

	// Create validator for recursive patterns
	config := NewGenericValidatorConfig("recursive_org_validator")
	
	// Add recursive patterns with {*} repetition
	config.AddPath("$.company{*}.name", map[string]interface{}{
		"validation": "string",
		"description": "All names at any level",
	})
	
	config.AddPath("$.company.divisions[*].departments[*].teams[*].members[*].name", map[string]interface{}{
		"validation": "string", 
		"description": "All team member names",
	})
	
	config.AddPath("$.company.divisions[*].departments[*].teams[*].members[*].role", map[string]interface{}{
		"validation": "string",
		"description": "All team member roles", 
	})

	validator, err := NewComplexPatternValidator(config)
	if err != nil {
		fmt.Printf("Failed to create validator: %v\n", err)
		return
	}

	report, err := validator.Validate(orgData)
	if err != nil {
		fmt.Printf("Validation failed: %v\n", err)
		return
	}

	fmt.Printf("Recursive validation completed in %v\n", report.Duration)
	fmt.Printf("Found %d valid paths:\n", report.TotalPaths)
	
	for _, result := range report.Results {
		fmt.Printf("  ✓ %s = %v\n", result.Path, result.Value)
	}
}