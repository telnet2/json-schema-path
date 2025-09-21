package validators

import (
	"fmt"
	"testing"
)

// BenchmarkAllValidators runs comprehensive benchmarks across all validator types
func BenchmarkAllValidators(b *testing.B) {
	// Test data with various complexity levels
	complexJSON := `{
		"company": {
			"employees": [
				{"name": "Alice", "email": "alice@company.com", "salary": 120000},
				{"name": "Bob", "email": "bob@company.com", "salary": 95000},
				{"name": "Carol", "email": "carol@company.com", "salary": 110000}
			],
			"departments": [
				{"name": "Engineering", "budget": 5000000},
				{"name": "Marketing", "budget": 2000000}
			]
		}
	}`

	// Simple validators - basic path validation
	simpleConfig := NewSimpleValidatorConfig("simple_test")
	simpleConfig.AddPath("$.company.employees[0].name")
	simpleConfig.AddPath("$.company.employees[*].name")
	simpleConfig.AddPath("$.company.departments[*].name")

	// Generic validators - with metadata
	genericConfig := NewGenericValidatorConfig("generic_test")
	genericConfig.AddPath("$.company.employees[*].name", map[string]interface{}{
		"validation": "string",
		"required":   true,
	})
	genericConfig.AddPath("$.company.employees[*].email", map[string]interface{}{
		"validation": "email",
		"pattern":    "^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\\.[a-zA-Z]{2,}$",
	})
	genericConfig.AddPath("$.company.employees[*].salary", map[string]interface{}{
		"validation": "numeric",
		"min":        0,
		"max":        200000,
	})

	// Test configurations for different validator types
	validatorConfigs := []struct {
		name      string
		validator func() (UnifiedValidator, error)
		jsonData  string
	}{
		// Simple validators
		{
			name: "Raw-Simple",
			validator: func() (UnifiedValidator, error) {
				return NewRawValidator(simpleConfig)
			},
			jsonData: complexJSON,
		},
		{
			name: "Optimized-Simple",
			validator: func() (UnifiedValidator, error) {
				return NewOptimizedValidator(simpleConfig)
			},
			jsonData: complexJSON,
		},
		{
			name: "Fast-Simple",
			validator: func() (UnifiedValidator, error) {
				return NewFastValidator(simpleConfig)
			},
			jsonData: complexJSON,
		},
		
		// Generic validators
		{
			name: "EnhancedGJSON-Generic",
			validator: func() (UnifiedValidator, error) {
				return NewEnhancedGJSONValidator(genericConfig)
			},
			jsonData: complexJSON,
		},
		{
			name: "SimpleGeneric-Generic",
			validator: func() (UnifiedValidator, error) {
				return NewSimpleGenericValidator(genericConfig)
			},
			jsonData: complexJSON,
		},
		{
			name: "ComplexPattern-Generic",
			validator: func() (UnifiedValidator, error) {
				return NewComplexPatternValidator(genericConfig)
			},
			jsonData: complexJSON,
		},
		{
			name: "OptimizedGeneric-Generic",
			validator: func() (UnifiedValidator, error) {
				return NewOptimizedGenericValidator(genericConfig)
			},
			jsonData: complexJSON,
		},
		{
			name: "GJSON-Generic",
			validator: func() (UnifiedValidator, error) {
				return NewGJSONValidator(genericConfig)
			},
			jsonData: complexJSON,
		},
	}

	// Run benchmarks for each validator
	for _, config := range validatorConfigs {
		b.Run(config.name, func(b *testing.B) {
			validator, err := config.validator()
			if err != nil {
				b.Fatalf("Failed to create validator: %v", err)
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				report, err := validator.Validate(config.jsonData)
				if err != nil {
					b.Fatalf("Validation failed: %v", err)
				}
				if report.TotalPaths == 0 {
					b.Fatalf("No paths validated")
				}
			}
		})
	}
}

// BenchmarkPathValidation compares path validation performance
func BenchmarkPathValidation(b *testing.B) {
	// Create a validator with many paths
	config := NewSimpleValidatorConfig("path_benchmark")
	
	// Add many paths for benchmarking
	for i := 0; i < 100; i++ {
		config.AddPath(fmt.Sprintf("$.data.items[%d].name", i))
		config.AddPath(fmt.Sprintf("$.data.items[%d].value", i))
		config.AddPath(fmt.Sprintf("$.data.items[%d].metadata.id", i))
	}
	
	jsonData := `{
		"data": {
			"items": [
				{"name": "item0", "value": 100, "metadata": {"id": "id0"}},
				{"name": "item1", "value": 200, "metadata": {"id": "id1"}},
				{"name": "item2", "value": 300, "metadata": {"id": "id2"}}
			]
		}
	}`

	validatorTypes := []string{"raw", "optimized", "fast", "gjson"}
	
	for _, vType := range validatorTypes {
		b.Run(vType, func(b *testing.B) {
			var validator UnifiedValidator
			var err error
			
			switch vType {
			case "raw":
				validator, err = NewRawValidator(config)
			case "optimized":
				validator, err = NewOptimizedValidator(config)
			case "fast":
				validator, err = NewFastValidator(config)
			case "gjson":
				// Convert SimpleValidatorConfig to GenericValidatorConfig for GJSON
				gjsonConfig := NewGenericValidatorConfig("gjson_validator")
				gjsonConfig.Description = config.Description
				for path := range config.Paths {
					gjsonConfig.AddPath(path, map[string]interface{}{"validation": "any"})
				}
				validator, err = NewGJSONValidator(gjsonConfig)
			}
			
			if err != nil {
				b.Fatalf("Failed to create validator: %v", err)
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				report, err := validator.Validate(jsonData)
				if err != nil {
					b.Fatalf("Validation failed: %v", err)
				}
				_ = report.TotalPaths // Use the result to prevent optimization
			}
		})
	}
}

// BenchmarkValidationWithHandlers tests handler-based validation
func BenchmarkValidationWithHandlers(b *testing.B) {
	config := NewGenericValidatorConfig("handler_benchmark")
	config.AddPath("$.users[*].id", map[string]interface{}{"validation": "integer"})
	config.AddPath("$.users[*].name", map[string]interface{}{"validation": "string"})
	config.AddPath("$.users[*].email", map[string]interface{}{"validation": "email"})

	jsonData := `{
		"users": [
			{"id": 1, "name": "Alice", "email": "alice@example.com"},
			{"id": 2, "name": "Bob", "email": "bob@example.com"},
			{"id": 3, "name": "Carol", "email": "carol@example.com"}
		]
	}`

	validatorTypes := []string{"enhanced_gjson", "simple_generic", "complex_pattern", "optimized_generic"}
	
	for _, vType := range validatorTypes {
		b.Run(vType, func(b *testing.B) {
			var validator UnifiedValidator
			var err error
			
			switch vType {
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
				if resultCount == 0 {
					b.Fatalf("No results from handler")
				}
			}
		})
	}
}