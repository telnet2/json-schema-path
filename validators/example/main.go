package main

import (
	"fmt"
	"log"

	"github.com/telnet2/json-schema-path/validators"
)

func main() {
	fmt.Println("=== Unified Validator Example ===")

	// Example JSON schema
	schemaJSON := `{
		"type": "object",
		"properties": {
			"users": {
				"type": "array",
				"items": {
					"type": "object",
					"properties": {
						"name": {"type": "string"},
						"email": {"type": "string"}
					}
				}
			}
		}
	}`

	// Create validators using direct constructors (no factory needed)
	
	// 1. Raw validator - simple path matching
	rawValidator, err := validators.NewRawValidatorFromJSON(schemaJSON)
	if err != nil {
		log.Fatalf("Failed to create raw validator: %v", err)
	}

	// 2. Optimized validator - with wildcard support
	optConfig := validators.NewSimpleValidatorConfig("optimized_validator")
	optConfig.AddPath("$.properties.users.items.properties.name")
	optConfig.AddPath("$.properties.users[*].properties.name")
	optValidator, err := validators.NewOptimizedValidator(optConfig)
	if err != nil {
		log.Fatalf("Failed to create optimized validator: %v", err)
	}

	// 3. Fast validator - pre-expanded patterns
	fastConfig := validators.NewSimpleValidatorConfig("fast_validator")
	fastConfig.AddPaths([]string{
		"$.properties.users.items.properties.name",
		"$.properties.users.items.properties.email",
		"$.properties.users[0].properties.name",
		"$.properties.users[*].properties.name",
	})
	fastValidator, err := validators.NewFastValidator(fastConfig)
	if err != nil {
		log.Fatalf("Failed to create fast validator: %v", err)
	}

	// Test paths
	testPaths := []string{
		"$.properties.users.items.properties.name",
		"$.properties.users[0].properties.name",
		"$.properties.users[*].properties.name",
	}

	// Validate paths
	validatorList := []struct {
		name      string
		validator validators.UnifiedValidator
	}{
		{"Raw", rawValidator},
		{"Optimized", optValidator},
		{"Fast", fastValidator},
	}

	for _, v := range validatorList {
		fmt.Printf("\n%s Validator Results:\n", v.name)
		for _, path := range testPaths {
			result := v.validator.ValidatePath(path)
			fmt.Printf("  %s: %v\n", path, result)
		}
		fmt.Printf("  Supported paths: %d\n", len(v.validator.GetSupportedPaths()))
	}

	// Demonstrate generic validator with metadata
	fmt.Println("\n=== Generic Validator with Metadata ===")
	
	genericConfig := validators.NewGenericValidatorConfig("email_validator")
	genericConfig.AddPath("$.users[*].email", map[string]interface{}{
		"validation": "email",
		"required":   true,
		"pattern":    "^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\\.[a-zA-Z]{2,}$",
	})
	genericConfig.AddPath("$.users[*].name", map[string]interface{}{
		"validation": "string",
		"min_length": 2,
		"max_length": 50,
	})

	genericValidator, err := validators.NewComplexPatternValidator(genericConfig)
	if err != nil {
		log.Fatalf("Failed to create generic validator: %v", err)
	}

	// Test data
	testData := `{
		"users": [
			{"name": "Alice Johnson", "email": "alice@example.com"},
			{"name": "Bob Smith", "email": "bob@example.com"}
		]
	}`

	report, err := genericValidator.Validate(testData)
	if err != nil {
		log.Fatalf("Validation failed: %v", err)
	}

	fmt.Printf("Generic validation completed in %v\n", report.Duration)
	fmt.Printf("Found %d valid paths:\n", report.ValidPaths)
	for _, result := range report.Results {
		fmt.Printf("  ✓ %s = %v\n", result.Path, result.Value)
	}
}