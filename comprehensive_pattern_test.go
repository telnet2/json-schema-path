package main

import (
	"fmt"
	"testing"

	jsonpkg "github.com/telnet2/json-schema-path/json"
	"github.com/telnet2/json-schema-path/parser"
	"github.com/telnet2/json-schema-path/tree"
)

// TestRegexPatternMatching tests regex pattern matching capabilities
func TestRegexPatternMatching(t *testing.T) {
	// Test JSON with regex-friendly property names
	testJSON := `{
		"users": [
			{"admin_user": "Admin1", "admin_email": "admin@test.com", "admin_level": "super"},
			{"user_name": "User1", "user_email": "user@test.com", "user_level": "normal"},
			{"service_name": "Service", "service_key": "svc123"}
		],
		"products": [
			{"laptop_device": "MacBook", "device_type": "computer"},
			{"phone_device": "iPhone", "device_category": "mobile"},
			{"coffee_table": "IKEA", "table_type": "furniture"}
		],
		"metadata": {
			"api_version": "1.2.3",
			"api_key": "secret123",
			"api_endpoint": "https://api.example.com"
		}
	}`

	processor := jsonpkg.NewPathExtractor()
	paths, _ := processor.ExtractPaths(testJSON)

	regexTests := []struct {
		name        string
		pattern     string
		expectedMatches []string
	}{
		{
			name:    "Admin prefix regex",
			pattern: "$.users[~^admin_.*]",
			expectedMatches: []string{
				"$.users[0].admin_user",
				"$.users[0].admin_email",
				"$.users[0].admin_level",
			},
		},
		{
			name:    "Device suffix regex",
			pattern: "$.products[~.*_device$]",
			expectedMatches: []string{
				"$.products[0].laptop_device",
				"$.products[1].phone_device",
			},
		},
		{
			name:    "API prefix regex",
			pattern: "$.metadata[~^api_.*]",
			expectedMatches: []string{
				"$.metadata.api_version",
				"$.metadata.api_key",
				"$.metadata.api_endpoint",
			},
		},
	}

	fmt.Printf("=== Regex Pattern Tests ===\n")
	for _, test := range regexTests {
		t.Run(test.name, func(t *testing.T) {
			fmt.Printf("\nTesting: %s\n", test.name)
			fmt.Printf("  Pattern: %s\n", test.pattern)

			expr, err := parser.ParseExpression(test.pattern)
			if err != nil {
				t.Fatalf("Failed to parse regex pattern: %v", err)
			}

			patternTree := tree.NewPatternTree()
			if err := patternTree.AddPattern(expr); err != nil {
				t.Fatalf("Failed to build pattern tree: %v", err)
			}

			// Find actual matches
			actualMatches := []string{}
			for _, path := range paths {
				segments := processor.ConvertPathToSegments(path)
				if patternTree.MatchSegments(segments) {
					actualMatches = append(actualMatches, path)
				}
			}

			fmt.Printf("  Expected matches: %v\n", test.expectedMatches)
			fmt.Printf("  Actual matches: %v\n", actualMatches)

			// Verify all expected matches are found
			for _, expected := range test.expectedMatches {
				found := false
				for _, actual := range actualMatches {
					if actual == expected {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected match %s not found in actual matches", expected)
				}
			}
		})
	}
}

// TestPropertyWildcardMatching tests property wildcard matching
func TestPropertyWildcardMatching(t *testing.T) {
	// Test JSON with property patterns
	testJSON := `{
		"users": [
			{"name": "John", "first_name": "John", "last_name": "Doe", "email": "john@doe.com"},
			{"admin_name": "Admin", "admin_email": "admin@test.com", "admin_level": "super"},
			{"service_name": "Service", "service_key": "svc123", "service_endpoint": "https://svc.example.com"}
		]
	}`

	processor := jsonpkg.NewPathExtractor()
	paths, _ := processor.ExtractPaths(testJSON)

	wildcardTests := []struct {
		name        string
		pattern     string
		expectedMatches []string
	}{
		{
			name:    "Name suffix wildcard",
			pattern: "$.users[*].[#*name]",
			expectedMatches: []string{
				"$.users[0].name",
				"$.users[0].first_name",
				"$.users[0].last_name",
				"$.users[1].admin_name",
				"$.users[2].service_name",
			},
		},
		{
			name:    "Admin prefix wildcard",
			pattern: "$.users[*].[#admin*]",
			expectedMatches: []string{
				"$.users[1].admin_name",
				"$.users[1].admin_email",
				"$.users[1].admin_level",
			},
		},
		{
			name:    "Service prefix wildcard",
			pattern: "$.users[*].[#service*]",
			expectedMatches: []string{
				"$.users[2].service_name",
				"$.users[2].service_key",
				"$.users[2].service_endpoint",
			},
		},
	}

	fmt.Printf("\n=== Property Wildcard Tests ===\n")
	for _, test := range wildcardTests {
		t.Run(test.name, func(t *testing.T) {
			fmt.Printf("\nTesting: %s\n", test.name)
			fmt.Printf("  Pattern: %s\n", test.pattern)

			expr, err := parser.ParseExpression(test.pattern)
			if err != nil {
				t.Fatalf("Failed to parse wildcard pattern: %v", err)
			}

			patternTree := tree.NewPatternTree()
			if err := patternTree.AddPattern(expr); err != nil {
				t.Fatalf("Failed to build pattern tree: %v", err)
			}

			// Find actual matches
			actualMatches := []string{}
			for _, path := range paths {
				segments := processor.ConvertPathToSegments(path)
				if patternTree.MatchSegments(segments) {
					actualMatches = append(actualMatches, path)
				}
			}

			fmt.Printf("  Expected matches: %v\n", test.expectedMatches)
			fmt.Printf("  Actual matches: %v\n", actualMatches)

			// Verify all expected matches are found
			for _, expected := range test.expectedMatches {
				found := false
				for _, actual := range actualMatches {
					if actual == expected {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected match %s not found in actual matches", expected)
				}
			}
		})
	}
}

// TestGroupPatternMatching tests group pattern matching
func TestGroupPatternMatching(t *testing.T) {
	// Test JSON
	testJSON := `{
		"data": {
			"users": [
				{"name": "User1", "email": "user1@test.com", "phone": "123-456-7890"},
				{"name": "User2", "email": "user2@test.com", "address": "123 Main St"}
			],
			"products": [
				{"id": 1, "name": "Product1", "price": 99.99},
				{"id": 2, "name": "Product2", "description": "Great product"}
			]
		}
	}`

	processor := jsonpkg.NewPathExtractor()
	paths, _ := processor.ExtractPaths(testJSON)

	groupTests := []struct {
		name        string
		pattern     string
		expectedMatches []string
	}{
		{
			name:    "Group pattern for user properties",
			pattern: "$.data.users[*].(name|email)",
			expectedMatches: []string{
				"$.data.users[0].name",
				"$.data.users[0].email",
				"$.data.users[1].name",
				"$.data.users[1].email",
			},
		},
		{
			name:    "Group pattern for product properties",
			pattern: "$.data.products[*].(name|id)",
			expectedMatches: []string{
				"$.data.products[0].name",
				"$.data.products[0].id",
				"$.data.products[1].name",
				"$.data.products[1].id",
			},
		},
		{
			name:    "Group pattern for timestamp fields",
			pattern: "$.metadata.(created_at|updated_at)",
			expectedMatches: []string{}, // This won't match our test JSON
		},
	}

	fmt.Printf("\n=== Group Pattern Tests ===\n")
	for _, test := range groupTests {
		t.Run(test.name, func(t *testing.T) {
			fmt.Printf("\nTesting: %s\n", test.name)
			fmt.Printf("  Pattern: %s\n", test.pattern)

			expr, err := parser.ParseExpression(test.pattern)
			if err != nil {
				t.Fatalf("Failed to parse group pattern: %v", err)
			}

			patternTree := tree.NewPatternTree()
			if err := patternTree.AddPattern(expr); err != nil {
				t.Fatalf("Failed to build pattern tree: %v", err)
			}

			// Find actual matches
			actualMatches := []string{}
			for _, path := range paths {
				segments := processor.ConvertPathToSegments(path)
				if patternTree.MatchSegments(segments) {
					actualMatches = append(actualMatches, path)
				}
			}

			fmt.Printf("  Expected matches: %v\n", test.expectedMatches)
			fmt.Printf("  Actual matches: %v\n", actualMatches)

			// Verify all expected matches are found
			for _, expected := range test.expectedMatches {
				found := false
				for _, actual := range actualMatches {
					if actual == expected {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected match %s not found in actual matches", expected)
				}
			}
		})
	}
}

// TestComplexPatternMatching tests complex pattern combinations
func TestComplexPatternMatching(t *testing.T) {
	// Test JSON
	testJSON := `{
		"enterprise": {
			"departments": [
				{
					"name": "Engineering",
					"teams": [
						{
							"team_name": "Backend",
							"members": [
								{"id": 1, "name": "Senior Dev 1", "email": "dev1@enterprise.com"},
								{"id": 2, "name": "Senior Dev 2", "email": "dev2@enterprise.com"}
							]
						}
					]
				}
			],
			"infrastructure": {
				"servers": [
					{"id": "srv-001", "cpu": "Intel Xeon", "memory": "64GB"},
					{"id": "srv-002", "cpu": "AMD EPYC", "memory": "128GB"}
				]
			}
		}
	}`

	processor := jsonpkg.NewPathExtractor()
	paths, _ := processor.ExtractPaths(testJSON)

	complexTests := []struct {
		name        string
		pattern     string
		expectedMatches []string
	}{
		{
			name:    "Deep nested with wildcards",
			pattern: "$.enterprise.departments[*].teams[*].members[*].(name|email)",
			expectedMatches: []string{
				"$.enterprise.departments[0].teams[0].members[0].name",
				"$.enterprise.departments[0].teams[0].members[0].email",
				"$.enterprise.departments[0].teams[0].members[1].name",
				"$.enterprise.departments[0].teams[0].members[1].email",
			},
		},
		{
			name:    "Multiple wildcards with groups",
			pattern: "$.enterprise.infrastructure.servers[*].(cpu|memory)",
			expectedMatches: []string{
				"$.enterprise.infrastructure.servers[0].cpu",
				"$.enterprise.infrastructure.servers[0].memory",
				"$.enterprise.infrastructure.servers[1].cpu",
				"$.enterprise.infrastructure.servers[1].memory",
			},
		},
	}

	fmt.Printf("\n=== Complex Pattern Tests ===\n")
	for _, test := range complexTests {
		t.Run(test.name, func(t *testing.T) {
			fmt.Printf("\nTesting: %s\n", test.name)
			fmt.Printf("  Pattern: %s\n", test.pattern)

			expr, err := parser.ParseExpression(test.pattern)
			if err != nil {
				t.Fatalf("Failed to parse complex pattern: %v", err)
			}

			patternTree := tree.NewPatternTree()
			if err := patternTree.AddPattern(expr); err != nil {
				t.Fatalf("Failed to build pattern tree: %v", err)
			}

			// Find actual matches
			actualMatches := []string{}
			for _, path := range paths {
				segments := processor.ConvertPathToSegments(path)
				if patternTree.MatchSegments(segments) {
					actualMatches = append(actualMatches, path)
				}
			}

			fmt.Printf("  Expected matches: %v\n", test.expectedMatches)
			fmt.Printf("  Actual matches: %v\n", actualMatches)

			// Verify all expected matches are found
			for _, expected := range test.expectedMatches {
				found := false
				for _, actual := range actualMatches {
					if actual == expected {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected match %s not found in actual matches", expected)
				}
			}
		})
	}
}