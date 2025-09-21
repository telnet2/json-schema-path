package validators

import (
	"fmt"
	"testing"
)

// BenchmarkRecursiveNestedSchema benchmarks validators with recursive nested schemas
func BenchmarkRecursiveNestedSchema(b *testing.B) {
	// Create deeply nested recursive JSON data
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
													"lead": {"name": "Alice", "email": "alice@techcorp.com"},
													"members": [
														{"name": "Bob", "role": "Senior Engineer"},
														{"name": "Carol", "role": "DevOps Engineer"}
													]
												},
												{
													"name": "Frontend Team",
													"lead": {"name": "David", "email": "david@techcorp.com"},
													"members": [
														{"name": "Eve", "role": "React Developer"},
														{"name": "Frank", "role": "UI/UX Designer"}
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

	// Test different recursive pattern complexities
	patternConfigs := []struct {
		name    string
		patterns map[string]interface{}
	}{
		{
			name: "Simple Recursive",
			patterns: map[string]interface{}{
				"$.enterprise.regions[*].name": map[string]interface{}{
					"validation": "string",
				},
			},
		},
		{
			name: "Medium Recursive",
			patterns: map[string]interface{}{
				"$.enterprise.regions[*].countries[*].name": map[string]interface{}{
					"validation": "string",
				},
				"$.enterprise.regions[*].countries[*].offices[*].name": map[string]interface{}{
					"validation": "string",
				},
			},
		},
		{
			name: "Deep Recursive",
			patterns: map[string]interface{}{
				"$.enterprise.regions[*].countries[*].offices[*].departments[*].name": map[string]interface{}{
					"validation": "string",
				},
				"$.enterprise.regions[*].countries[*].offices[*].departments[*].teams[*].name": map[string]interface{}{
					"validation": "string",
				},
			},
		},
		{
			name: "Full Recursive",
			patterns: map[string]interface{}{
				"$.enterprise.regions[*].countries[*].offices[*].departments[*].teams[*].members[*].name": map[string]interface{}{
					"validation": "string",
				},
				"$.enterprise.regions[*].countries[*].offices[*].departments[*].teams[*].members[*].role": map[string]interface{}{
					"validation": "string",
				},
				"$.enterprise.regions[*].countries[*].offices[*].departments[*].teams[*].lead.email": map[string]interface{}{
					"validation": "email",
				},
			},
		},
		{
			name: "Repetition Patterns",
			patterns: map[string]interface{}{
				"$.enterprise{*}.name": map[string]interface{}{
					"validation": "string",
				},
				"$.enterprise{*}.teams[*].members[*].name": map[string]interface{}{
					"validation": "string",
				},
			},
		},
	}

	// Test each validator type with recursive patterns
	validatorTypes := []string{"complex_pattern", "optimized_generic", "gjson"}

	for _, patternConfig := range patternConfigs {
		for _, validatorType := range validatorTypes {
			b.Run(fmt.Sprintf("%s-%s", validatorType, patternConfig.name), func(b *testing.B) {
				// Create validator configuration
				config := NewGenericValidatorConfig(fmt.Sprintf("%s_%s", validatorType, patternConfig.name))
				for pattern, metadata := range patternConfig.patterns {
					config.AddPath(pattern, metadata)
				}

				// Create validator
				var validator UnifiedValidator
				var err error
				
				switch validatorType {
				case "complex_pattern":
					validator, err = NewComplexPatternValidator(config)
				case "optimized_generic":
					validator, err = NewOptimizedGenericValidator(config)
				case "gjson":
					validator, err = NewGJSONValidator(config)
				}

				if err != nil {
					b.Fatalf("Failed to create validator: %v", err)
				}

				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					report, err := validator.Validate(recursiveJSON)
					if err != nil {
						b.Fatalf("Validation failed: %v", err)
					}
					if report.TotalPaths == 0 && patternConfig.name != "Repetition Patterns" {
						b.Fatalf("No paths validated for %s", patternConfig.name)
					}
				}
			})
		}
	}
}

// BenchmarkRecursiveNestedSchemaWithHandlers tests handler-based validation with recursive schemas
func BenchmarkRecursiveNestedSchemaWithHandlers(b *testing.B) {
	recursiveJSON := `{
		"organization": {
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
									"lead": {"name": "Alice", "email": "alice@megacorp.com"},
									"members": [
										{"name": "Bob", "role": "Senior Engineer"},
										{"name": "Carol", "role": "DevOps Engineer"}
									]
								}
							]
						}
					]
				}
			]
		}
	}`

	config := NewGenericValidatorConfig("recursive_handler_test")
	config.AddPath("$.organization.divisions[*].departments[*].teams[*].members[*].name", map[string]interface{}{
		"validation": "string",
	})
	config.AddPath("$.organization.divisions[*].departments[*].teams[*].members[*].role", map[string]interface{}{
		"validation": "string",
	})
	config.AddPath("$.organization.divisions[*].departments[*].teams[*].lead.email", map[string]interface{}{
		"validation": "email",
	})

	validatorTypes := []string{"complex_pattern", "optimized_generic", "gjson"}
	
	for _, validatorType := range validatorTypes {
		b.Run(validatorType, func(b *testing.B) {
			var validator UnifiedValidator
			var err error
			
			switch validatorType {
			case "complex_pattern":
				validator, err = NewComplexPatternValidator(config)
			case "optimized_generic":
				validator, err = NewOptimizedGenericValidator(config)
			case "gjson":
				validator, err = NewGJSONValidator(config)
			}

			if err != nil {
				b.Fatalf("Failed to create validator: %v", err)
			}

			resultCount := 0
			handler := func(result ValidationResult) error {
				resultCount++
				return nil
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				resultCount = 0
				err := validator.ValidateWithHandler(recursiveJSON, handler)
				if err != nil {
					b.Fatalf("Validation with handler failed: %v", err)
				}
				if resultCount == 0 {
					b.Fatalf("No results from handler")
				}
			}
		})
	}
}

// BenchmarkRecursiveNestedSchemaScalability tests scalability with varying depths
func BenchmarkRecursiveNestedSchemaScalability(b *testing.B) {
	// Create JSON data with different nesting depths
	depths := []struct {
		name  string
		depth int
	}{
		{"Shallow", 3},
		{"Medium", 5},
		{"Deep", 7},
		{"VeryDeep", 10},
	}

	for _, depth := range depths {
		b.Run(depth.name, func(b *testing.B) {
			// Generate nested JSON with specified depth
			nestedJSON := generateNestedJSON(depth.depth)
			
			config := NewGenericValidatorConfig(fmt.Sprintf("scalability_%s", depth.name))
			config.AddPath("$.data{*}.value", map[string]interface{}{
				"validation": "numeric",
			})
			config.AddPath("$.data{*}.items[*].name", map[string]interface{}{
				"validation": "string",
			})

			validator, err := NewComplexPatternValidator(config)
			if err != nil {
				b.Fatalf("Failed to create validator: %v", err)
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				report, err := validator.Validate(nestedJSON)
				if err != nil {
					b.Fatalf("Validation failed: %v", err)
				}
				_ = report.TotalPaths // Use the result
			}
		})
	}
}

// Helper function to generate nested JSON with specified depth
func generateNestedJSON(depth int) string {
	if depth <= 0 {
		return `{"value": 42, "items": [{"name": "item1"}, {"name": "item2"}]}`
	}
	
	innerJSON := generateNestedJSON(depth - 1)
	return fmt.Sprintf(`{"data": %s, "value": %d, "items": [{"name": "level%d_item1"}, {"name": "level%d_item2"}]}`, 
		innerJSON, depth*10, depth, depth)
}