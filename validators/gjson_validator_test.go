package validators

import (
	"fmt"
	"strings"
	"testing"
	"github.com/tidwall/gjson"
)

func TestGJSONValidatorDebug(t *testing.T) {
	// Test with very simple JSON first
	simpleJSON := `{"company": {"employees": [{"name": "Alice"}]}}`
	
	fmt.Printf("Testing simple JSON: %s\n", simpleJSON)
	
	// Test GJSON directly
	result := gjson.Parse(simpleJSON)
	fmt.Printf("Direct GJSON test - company.employees.#.name exists: %t, value: %v\n", 
		result.Get("company.employees.#.name").Exists(), 
		result.Get("company.employees.#.name").Value())
	
	// Test the conversion function directly
	original := "$.company.employees[*].name"
	converted := original
	if strings.HasPrefix(converted, "$") {
		converted = converted[1:]
	}
	if strings.HasPrefix(converted, ".") {
		converted = converted[1:]
	}
	converted = strings.ReplaceAll(converted, "[*]", "#")
	
	fmt.Printf("Conversion test: '%s' -> '%s'\n", original, converted)
	fmt.Printf("Converted pattern exists: %t, value: %v\n", 
		result.Get(converted).Exists(), 
		result.Get(converted).Value())
	
	// Now test the validator
	config := NewGenericValidatorConfig("gjson_test")
	config.AddPath("$.company.employees[*].name", map[string]interface{}{"validation": "string"})

	validator, err := NewGJSONValidator(config)
	if err != nil {
		t.Fatalf("Error creating validator: %v", err)
	}

	report, err := validator.Validate(simpleJSON)
	if err != nil {
		t.Fatalf("Validation error: %v", err)
	}

	fmt.Printf("Validation report:\n")
	fmt.Printf("Total paths: %d\n", report.TotalPaths)
	fmt.Printf("Results: %d\n", len(report.Results))

	for i, result := range report.Results {
		fmt.Printf("  %d: Path=%s, Valid=%t, Value=%v\n", i, result.Path, result.Valid, result.Value)
	}
}