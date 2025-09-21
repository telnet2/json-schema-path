package validators

import (
	"fmt"
	"testing"
)

// Benchmark comparing the new generic validators with complex pattern support
func BenchmarkGenericValidators(b *testing.B) {
	// Sample JSON data for benchmarking
	jsonData := `{
		"company": {
			"employees": [
				{"name": "Alice Johnson", "email": "alice@company.com", "salary": 120000, "department": "Engineering"},
				{"name": "Bob Smith", "email": "bob@company.com", "salary": 95000, "department": "Marketing"},
				{"name": "Carol Davis", "email": "carol@company.com", "salary": 110000, "department": "Sales"},
				{"name": "David Wilson", "email": "david@company.com", "salary": 130000, "department": "Engineering"},
				{"name": "Eve Brown", "email": "eve@company.com", "salary": 105000, "department": "HR"}
			],
			"departments": [
				{"name": "Engineering", "budget": 5000000, "manager": "Alice Johnson"},
				{"name": "Marketing", "budget": 2000000, "manager": "Bob Smith"},
				{"name": "Sales", "budget": 3000000, "manager": "Carol Davis"},
				{"name": "HR", "budget": 1500000, "manager": "Eve Brown"}
			]
		}
	}`

	// Configuration with various pattern types
	config := NewGenericValidatorConfig("benchmark_config")
	config.Description = "Benchmark configuration with mixed pattern types"
	config.AddPath("$.company.employees[0].name", map[string]interface{}{"validation": "string"})
	config.AddPath("$.company.employees[0].email", map[string]interface{}{"validation": "email"})
	config.AddPath("$.company.employees[*].name", map[string]interface{}{"validation": "string"})
	config.AddPath("$.company.employees[*].salary", map[string]interface{}{"validation": "numeric", "min": 0, "max": 200000})
	config.AddPath("$.company.departments[*].name", map[string]interface{}{"validation": "string"})
	config.AddPath("$.company.departments[*].budget", map[string]interface{}{"validation": "numeric", "min": 0})
	config.AddPath("$.company.departments[0].manager", map[string]interface{}{"validation": "string"})
	config.AddPath("$.company.departments[1].manager", map[string]interface{}{"validation": "string"})

	validatorTypes := []string{"enhanced_gjson", "optimized_generic", "complex_pattern"}

	for _, validatorType := range validatorTypes {
		var validator UnifiedValidator
		var err error
		
		switch validatorType {
		case "enhanced_gjson":
			validator, err = NewEnhancedGJSONValidator(config)
		case "optimized_generic":
			validator, err = NewOptimizedGenericValidator(config)
		case "complex_pattern":
			validator, err = NewComplexPatternValidator(config)
		default:
			b.Fatalf("Unknown validator type: %s", validatorType)
		}
		
		if err != nil {
			b.Fatalf("Failed to create %s validator: %v", validatorType, err)
		}

		b.Run(validatorType, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				_, err := validator.Validate(jsonData)
				if err != nil {
					b.Fatalf("Validation failed: %v", err)
				}
			}
		})
	}
}

// Benchmark with complex patterns including wildcards and groups
func BenchmarkComplexPatterns(b *testing.B) {
	jsonData := `{
		"enterprise": {
			"departments": [
				{
					"name": "Engineering",
					"teams": [
						{"name": "Backend", "lead": "Alice", "members": ["Bob", "Carol"]},
						{"name": "Frontend", "lead": "David", "members": ["Eve", "Frank"]}
					]
				},
				{
					"name": "Product",
					"teams": [
						{"name": "Design", "lead": "Grace", "members": ["Henry", "Ivy"]},
						{"name": "QA", "lead": "Jack", "members": ["Kelly", "Liam"]}
					]
				}
			]
		}
	}`

	// Complex pattern configuration
	config := NewGenericValidatorConfig("complex_patterns")
	config.Description = "Complex patterns with nested wildcards"
	config.AddPath("$.enterprise.departments[*].teams[*].name", map[string]interface{}{"validation": "string"})
	config.AddPath("$.enterprise.departments[*].teams[*].lead", map[string]interface{}{"validation": "string"})
	config.AddPath("$.enterprise.departments[0].teams[0].name", map[string]interface{}{"validation": "string"})
	config.AddPath("$.enterprise.departments[0].teams[1].name", map[string]interface{}{"validation": "string"})
	config.AddPath("$.enterprise.departments[1].teams[0].name", map[string]interface{}{"validation": "string"})
	config.AddPath("$.enterprise.departments[1].teams[1].name", map[string]interface{}{"validation": "string"})

	b.Run("ComplexPatterns", func(b *testing.B) {
		validator, err := NewOptimizedGenericValidator(config)
		if err != nil {
			b.Fatalf("Failed to create validator: %v", err)
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := validator.Validate(jsonData)
			if err != nil {
				b.Fatalf("Validation failed: %v", err)
			}
		}
	})
}

// Benchmark validation with handlers
func BenchmarkValidationWithHandler(b *testing.B) {
	jsonData := `{
		"users": [
			{"id": 1, "name": "Alice", "email": "alice@example.com", "active": true},
			{"id": 2, "name": "Bob", "email": "bob@example.com", "active": false},
			{"id": 3, "name": "Carol", "email": "carol@example.com", "active": true}
		]
	}`

	config := NewGenericValidatorConfig("handler_test")
	config.AddPath("$.users[*].id", map[string]interface{}{"validation": "integer", "min": 1})
	config.AddPath("$.users[*].name", map[string]interface{}{"validation": "string", "min_length": 1})
	config.AddPath("$.users[*].email", map[string]interface{}{"validation": "email"})
	config.AddPath("$.users[*].active", map[string]interface{}{"validation": "boolean"})

	validator, err := NewEnhancedGJSONValidator(config)
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
		err := validator.ValidateWithHandler(jsonData, handler)
		if err != nil {
			b.Fatalf("Validation with handler failed: %v", err)
		}
	}
}

// Example usage function
func ExampleUnifiedValidator() {
	fmt.Println("=== Generic Validator Family Example ===")

	// JSON data
	jsonData := `{
		"store": {
			"products": [
				{"id": 1, "name": "Laptop", "price": 999.99, "category": "Electronics"},
				{"id": 2, "name": "Book", "price": 19.99, "category": "Education"}
			]
		}
	}`

	// Create configuration
	config := NewGenericValidatorConfig("store_validation")
	config.Description = "Validate store product data"
	config.AddPath("$.store.products[*].id", map[string]interface{}{"validation": "integer", "min": 1})
	config.AddPath("$.store.products[*].name", map[string]interface{}{"validation": "string", "min_length": 1})
	config.AddPath("$.store.products[*].price", map[string]interface{}{"validation": "numeric", "min": 0})

	// Create validator
	validator, err := NewOptimizedGenericValidator(config)
	if err != nil {
		fmt.Printf("Error creating validator: %v\n", err)
		return
	}

	// Perform validation
	report, err := validator.Validate(jsonData)
	if err != nil {
		fmt.Printf("Validation error: %v\n", err)
		return
	}

	fmt.Printf("Validation completed in %v\n", report.Duration)
	fmt.Printf("Found %d valid paths:\n", report.ValidPaths)
	
	for _, result := range report.Results {
		fmt.Printf("  ✓ %s = %v\n", result.Path, result.Value)
	}
}