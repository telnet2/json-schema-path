package main

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/telnet2/json-schema-path/validators"
)

func main() {
	fmt.Println("=== Recursive Nested Schema Validation Demo ===")
	fmt.Println()

	// Create a deeply nested recursive organizational structure
	recursiveJSON := `{
		"enterprise": {
			"name": "TechCorp Global",
			"regions": [
				{
					"name": "North America",
					"countries": [
						{
							"name": "United States",
							"offices": [
								{
									"name": "San Francisco HQ",
									"departments": [
										{
											"name": "Engineering",
											"teams": [
												{
													"name": "Platform Team",
													"lead": {"name": "Alice Johnson", "email": "alice@techcorp.com"},
													"members": [
														{"name": "Bob Smith", "role": "Senior Engineer", "skills": ["Go", "Kubernetes"]},
														{"name": "Carol Davis", "role": "DevOps Engineer", "skills": ["Docker", "AWS"]}
													]
												},
												{
													"name": "Frontend Team",
													"lead": {"name": "David Wilson", "email": "david@techcorp.com"},
													"members": [
														{"name": "Eve Brown", "role": "React Developer", "skills": ["React", "TypeScript"]},
														{"name": "Frank Miller", "role": "UI/UX Designer", "skills": ["Figma", "CSS"]}
													]
												}
											]
										}
									]
								}
							]
						}
					]
				}
			]
		}
	}`

	fmt.Println("1. Testing Complex Nested Patterns:")
	fmt.Println("-----------------------------------")

	// Create validator for complex nested patterns
	config := validators.NewGenericValidatorConfig("recursive_validator")
	
	// Test deep nested patterns with multiple wildcards
	config.AddPath("$.enterprise.regions[*].countries[*].offices[*].departments[*].teams[*].members[*].name", map[string]interface{}{
		"validation": "string",
		"description": "All team member names across all levels",
	})
	
	config.AddPath("$.enterprise.regions[*].countries[*].offices[*].departments[*].teams[*].lead.email", map[string]interface{}{
		"validation": "email",
		"description": "All team lead emails",
	})
	
	config.AddPath("$.enterprise.regions[*].countries[*].offices[*].departments[*].teams[*].members[*].role", map[string]interface{}{
		"validation": "string",
		"description": "All team member roles",
	})
	
	config.AddPath("$.enterprise.regions[*].countries[*].offices[*].departments[*].teams[*].members[*].skills[*]", map[string]interface{}{
		"validation": "string",
		"description": "All skills across all team members",
	})

	validator, err := validators.NewComplexPatternValidator(config)
	if err != nil {
		log.Fatalf("Failed to create validator: %v", err)
	}

	report, err := validator.Validate(recursiveJSON)
	if err != nil {
		log.Fatalf("Validation failed: %v", err)
	}

	fmt.Printf("Found %d valid paths in %v\n", report.TotalPaths, report.Duration)
	fmt.Printf("- Team member names: %d\n", countPathsWithPattern(report.Results, "name"))
	fmt.Printf("- Team lead emails: %d\n", countPathsWithPattern(report.Results, "email"))
	fmt.Printf("- Team member roles: %d\n", countPathsWithPattern(report.Results, "role"))
	fmt.Printf("- Skills: %d\n", countPathsWithPattern(report.Results, "skills"))

	fmt.Println("\n2. Testing {*} Repetition Patterns:")
	fmt.Println("-----------------------------------")

	// Test {*} repetition patterns for deep traversal
	repetitionConfig := validators.NewGenericValidatorConfig("repetition_validator")
	
	// Use {*} for zero-or-more repetition to find data at any depth
	repetitionConfig.AddPath("$.enterprise{*}.name", map[string]interface{}{
		"validation": "string",
		"description": "All names at any depth using {*} repetition",
	})
	
	repetitionConfig.AddPath("$.enterprise{*}.skills[*]", map[string]interface{}{
		"validation": "string",
		"description": "All skills found through {*} repetition",
	})

	repetitionValidator, err := validators.NewComplexPatternValidator(repetitionConfig)
	if err != nil {
		log.Fatalf("Failed to create repetition validator: %v", err)
	}

	repetitionReport, err := repetitionValidator.Validate(recursiveJSON)
	if err != nil {
		log.Fatalf("Repetition validation failed: %v", err)
	}

	fmt.Printf("Found %d paths with {*} repetition in %v\n", repetitionReport.TotalPaths, repetitionReport.Duration)
	
	fmt.Println("\n3. Sample Results:")
	fmt.Println("-----------------")
	
	// Show some sample results
	sampleCount := 0
	for _, result := range report.Results {
		if sampleCount < 5 {
			fmt.Printf("  ✓ %s = %v\n", result.Path, result.Value)
			sampleCount++
		}
	}
	
	if len(report.Results) > 5 {
		fmt.Printf("  ... and %d more results\n", len(report.Results)-5)
	}

	fmt.Println("\n4. Performance Analysis:")
	fmt.Println("------------------------")
	fmt.Printf("- Complex nested pattern matching: %v\n", report.Duration)
	fmt.Printf("- {*} repetition pattern matching: %v\n", repetitionReport.Duration)
	fmt.Printf("- Average time per path: %v\n", report.Duration/time.Duration(report.TotalPaths))
}

// Helper function to count paths containing a specific pattern
func countPathsWithPattern(results []validators.ValidationResult, pattern string) int {
	count := 0
	for _, result := range results {
		if contains(result.Path, pattern) {
			count++
		}
	}
	return count
}

// Helper function to check if string contains substring
func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}