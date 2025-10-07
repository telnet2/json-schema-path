package validators

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/tidwall/gjson"
)

// Generate small JSON data (~500 bytes)
func generateSmallJSON() string {
	return `{
		"users": [
			{
				"name": "Alice",
				"email": "alice@example.com",
				"age": 30,
				"profile": {
					"bio": "Software engineer",
					"location": "San Francisco"
				}
			},
			{
				"name": "Bob",
				"email": "bob@example.com",
				"age": 25,
				"profile": {
					"bio": "Product manager",
					"location": "New York"
				}
			}
		],
		"company": {
			"name": "TechCorp",
			"founded": 2010
		}
	}`
}

// Generate large JSON data (>1MB)
func generateLargeJSON() string {
	var builder strings.Builder
	builder.WriteString(`{"enterprise": {"regions": [`)

	// Create 10 regions
	for r := 0; r < 10; r++ {
		if r > 0 {
			builder.WriteString(",")
		}
		builder.WriteString(fmt.Sprintf(`{"name": "Region%d", "countries": [`, r))

		// 10 countries per region
		for c := 0; c < 10; c++ {
			if c > 0 {
				builder.WriteString(",")
			}
			builder.WriteString(fmt.Sprintf(`{"name": "Country%d", "offices": [`, c))

			// 20 offices per country
			for o := 0; o < 20; o++ {
				if o > 0 {
					builder.WriteString(",")
				}
				builder.WriteString(fmt.Sprintf(`{"name": "Office%d", "employees": [`, o))

				// 50 employees per office
				for e := 0; e < 50; e++ {
					if e > 0 {
						builder.WriteString(",")
					}
					builder.WriteString(fmt.Sprintf(`{
						"id": %d,
						"name": "Employee%d",
						"email": "employee%d@company.com",
						"age": %d,
						"department": "Department%d",
						"salary": %d,
						"skills": ["skill1", "skill2", "skill3"],
						"metadata": {
							"hired": "2020-01-01",
							"performance": "excellent"
						}
					}`, e, e, e, 25+(e%40), e%10, 50000+(e*1000)))
				}

				builder.WriteString(`]}`)
			}

			builder.WriteString(`]}`)
		}

		builder.WriteString(`]}`)
	}

	builder.WriteString(`]}}`)

	data := builder.String()
	// Verify it's > 1MB
	if len(data) < 1024*1024 {
		panic(fmt.Sprintf("Generated JSON is only %d bytes, need > 1MB", len(data)))
	}
	return data
}

// ========== SMALL JSON BENCHMARKS ==========

// Benchmark simple pattern: $.users[*].email
func BenchmarkSmallJSON_Simple_gjson(b *testing.B) {
	jsonData := generateSmallJSON()
	pattern := "users.#.email"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result := gjson.Get(jsonData, pattern)
		_ = result
	}
}

func BenchmarkSmallJSON_Simple_Streaming(b *testing.B) {
	jsonData := generateSmallJSON()
	patterns := map[string]json.RawMessage{
		"$.users[*].email": json.RawMessage(`{"type": "string"}`),
	}

	validator, err := NewStreamingValidator(patterns)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		report, err := validator.Validate(jsonData)
		if err != nil {
			b.Fatal(err)
		}
		_ = report
	}
}

func BenchmarkSmallJSON_Simple_Hybrid(b *testing.B) {
	jsonData := generateSmallJSON()
	patterns := map[string]json.RawMessage{
		"$.users[*].email": json.RawMessage(`{"type": "string"}`),
	}

	validator, err := NewHybridValidator(patterns)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		report, err := validator.Validate(jsonData)
		if err != nil {
			b.Fatal(err)
		}
		_ = report
	}
}

func BenchmarkSmallJSON_Simple_OptimizedGeneric(b *testing.B) {
	jsonData := generateSmallJSON()
	patterns := map[string]json.RawMessage{
		"$.users[*].email": json.RawMessage(`{"type": "string"}`),
	}

	config := &GenericValidatorConfig{
		Name:  "test",
		Paths: patterns,
	}

	validator, err := NewOptimizedGenericValidator(config)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		report, err := validator.Validate(jsonData)
		if err != nil {
			b.Fatal(err)
		}
		_ = report
	}
}

// Benchmark complex pattern: $.users{*}.email (requires {*} support)
func BenchmarkSmallJSON_Complex_Streaming(b *testing.B) {
	jsonData := generateSmallJSON()
	patterns := map[string]json.RawMessage{
		"$.users{*}.email": json.RawMessage(`{"type": "string"}`),
	}

	validator, err := NewStreamingValidator(patterns)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		report, err := validator.Validate(jsonData)
		if err != nil {
			b.Fatal(err)
		}
		_ = report
	}
}

func BenchmarkSmallJSON_Complex_Hybrid(b *testing.B) {
	jsonData := generateSmallJSON()
	patterns := map[string]json.RawMessage{
		"$.users{*}.email": json.RawMessage(`{"type": "string"}`),
	}

	validator, err := NewHybridValidator(patterns)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		report, err := validator.Validate(jsonData)
		if err != nil {
			b.Fatal(err)
		}
		_ = report
	}
}

// ========== LARGE JSON BENCHMARKS ==========

func BenchmarkLargeJSON_Simple_gjson(b *testing.B) {
	jsonData := generateLargeJSON()
	pattern := "enterprise.regions.#.countries.#.offices.#.employees.#.email"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result := gjson.Get(jsonData, pattern)
		_ = result
	}
}

func BenchmarkLargeJSON_Simple_Streaming(b *testing.B) {
	jsonData := generateLargeJSON()
	patterns := map[string]json.RawMessage{
		"$.enterprise.regions[*].countries[*].offices[*].employees[*].email": json.RawMessage(`{"type": "string"}`),
	}

	validator, err := NewStreamingValidator(patterns)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		report, err := validator.Validate(jsonData)
		if err != nil {
			b.Fatal(err)
		}
		_ = report
	}
}

func BenchmarkLargeJSON_Simple_Hybrid(b *testing.B) {
	jsonData := generateLargeJSON()
	patterns := map[string]json.RawMessage{
		"$.enterprise.regions[*].countries[*].offices[*].offices[*].employees[*].email": json.RawMessage(`{"type": "string"}`),
	}

	validator, err := NewHybridValidator(patterns)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		report, err := validator.Validate(jsonData)
		if err != nil {
			b.Fatal(err)
		}
		_ = report
	}
}

func BenchmarkLargeJSON_Simple_OptimizedGeneric(b *testing.B) {
	jsonData := generateLargeJSON()
	patterns := map[string]json.RawMessage{
		"$.enterprise.regions[*].countries[*].offices[*].employees[*].email": json.RawMessage(`{"type": "string"}`),
	}

	config := &GenericValidatorConfig{
		Name:  "test",
		Paths: patterns,
	}

	validator, err := NewOptimizedGenericValidator(config)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		report, err := validator.Validate(jsonData)
		if err != nil {
			b.Fatal(err)
		}
		_ = report
	}
}

// Complex pattern with {*}
func BenchmarkLargeJSON_Complex_Streaming(b *testing.B) {
	jsonData := generateLargeJSON()
	patterns := map[string]json.RawMessage{
		"$.enterprise{*}.email": json.RawMessage(`{"type": "string"}`),
	}

	validator, err := NewStreamingValidator(patterns)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		report, err := validator.Validate(jsonData)
		if err != nil {
			b.Fatal(err)
		}
		_ = report
	}
}

func BenchmarkLargeJSON_Complex_Hybrid(b *testing.B) {
	jsonData := generateLargeJSON()
	patterns := map[string]json.RawMessage{
		"$.enterprise{*}.email": json.RawMessage(`{"type": "string"}`),
	}

	validator, err := NewHybridValidator(patterns)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		report, err := validator.Validate(jsonData)
		if err != nil {
			b.Fatal(err)
		}
		_ = report
	}
}

// ========== MULTI-PATTERN BENCHMARKS ==========

func BenchmarkSmallJSON_MultiPattern_Streaming(b *testing.B) {
	jsonData := generateSmallJSON()
	patterns := map[string]json.RawMessage{
		"$.users[*].email":             json.RawMessage(`{"type": "string"}`),
		"$.users[*].name":              json.RawMessage(`{"type": "string"}`),
		"$.users[*].profile.location":  json.RawMessage(`{"type": "string"}`),
		"$.company.name":               json.RawMessage(`{"type": "string"}`),
	}

	validator, err := NewStreamingValidator(patterns)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		report, err := validator.Validate(jsonData)
		if err != nil {
			b.Fatal(err)
		}
		_ = report
	}
}

func BenchmarkSmallJSON_MultiPattern_Hybrid(b *testing.B) {
	jsonData := generateSmallJSON()
	patterns := map[string]json.RawMessage{
		"$.users[*].email":             json.RawMessage(`{"type": "string"}`),
		"$.users[*].name":              json.RawMessage(`{"type": "string"}`),
		"$.users[*].profile.location":  json.RawMessage(`{"type": "string"}`),
		"$.company.name":               json.RawMessage(`{"type": "string"}`),
	}

	validator, err := NewHybridValidator(patterns)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		report, err := validator.Validate(jsonData)
		if err != nil {
			b.Fatal(err)
		}
		_ = report
	}
}

func BenchmarkLargeJSON_MultiPattern_Streaming(b *testing.B) {
	jsonData := generateLargeJSON()
	patterns := map[string]json.RawMessage{
		"$.enterprise.regions[*].countries[*].offices[*].employees[*].email":   json.RawMessage(`{"type": "string"}`),
		"$.enterprise.regions[*].countries[*].offices[*].employees[*].name":    json.RawMessage(`{"type": "string"}`),
		"$.enterprise.regions[*].countries[*].offices[*].employees[*].salary":  json.RawMessage(`{"type": "number"}`),
		"$.enterprise.regions[*].countries[*].offices[*].name":                 json.RawMessage(`{"type": "string"}`),
	}

	validator, err := NewStreamingValidator(patterns)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		report, err := validator.Validate(jsonData)
		if err != nil {
			b.Fatal(err)
		}
		_ = report
	}
}

func BenchmarkLargeJSON_MultiPattern_Hybrid(b *testing.B) {
	jsonData := generateLargeJSON()
	patterns := map[string]json.RawMessage{
		"$.enterprise.regions[*].countries[*].offices[*].employees[*].email":   json.RawMessage(`{"type": "string"}`),
		"$.enterprise.regions[*].countries[*].offices[*].employees[*].name":    json.RawMessage(`{"type": "string"}`),
		"$.enterprise.regions[*].countries[*].offices[*].employees[*].salary":  json.RawMessage(`{"type": "number"}`),
		"$.enterprise.regions[*].countries[*].offices[*].name":                 json.RawMessage(`{"type": "string"}`),
	}

	validator, err := NewHybridValidator(patterns)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		report, err := validator.Validate(jsonData)
		if err != nil {
			b.Fatal(err)
		}
		_ = report
	}
}
