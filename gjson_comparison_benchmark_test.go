package main

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	jsonpkg "github.com/telnet2/json-schema-path/json"
	"github.com/telnet2/json-schema-path/parser"
	"github.com/telnet2/json-schema-path/spec"
	"github.com/tidwall/gjson"
)

// Test data for comprehensive benchmarking
func generateBenchmarkJSON() string {
	return `{
		"enterprise": {
			"departments": [
				{
					"name": "Engineering",
					"teams": [
						{
							"team_name": "Backend",
							"members": [
								{"id": 1, "name": "Senior Dev 1", "email": "dev1@enterprise.com", "salary": 120000, "role": "senior"},
								{"id": 2, "name": "Senior Dev 2", "email": "dev2@enterprise.com", "salary": 125000, "role": "senior"},
								{"id": 3, "name": "Junior Dev 1", "email": "junior1@enterprise.com", "salary": 80000, "role": "junior"}
							]
						},
						{
							"team_name": "Frontend",
							"members": [
								{"id": 4, "name": "Frontend Dev 1", "email": "fe1@enterprise.com", "salary": 110000, "role": "senior"},
								{"id": 5, "name": "Frontend Dev 2", "email": "fe2@enterprise.com", "salary": 105000, "role": "mid"}
							]
						}
					]
				},
				{
					"name": "Marketing",
					"teams": [
						{
							"team_name": "Digital",
							"members": [
								{"id": 6, "name": "Digital Marketer 1", "email": "dm1@enterprise.com", "salary": 85000, "role": "specialist"},
								{"id": 7, "name": "Digital Marketer 2", "email": "dm2@enterprise.com", "salary": 90000, "role": "specialist"}
							]
						}
					]
				}
			],
			"infrastructure": {
				"servers": [
					{"id": "srv-001", "name": "Web Server 1", "cpu": "Intel Xeon", "memory": "64GB", "status": "active"},
					{"id": "srv-002", "name": "Database Server", "cpu": "AMD EPYC", "memory": "128GB", "status": "active"},
					{"id": "srv-003", "name": "Cache Server", "cpu": "Intel Xeon", "memory": "32GB", "status": "maintenance"}
				]
			}
		}
	}`
}

// Benchmark configurations with correct syntax patterns
var benchmarkConfigs = map[string]map[string]json.RawMessage{
	"simple_patterns": {
		"$.enterprise.departments[0].name": json.RawMessage(`{"validation":"string","required":true}`),
		"$.enterprise.departments[0].teams[0].team_name": json.RawMessage(`{"validation":"string","required":true}`),
		"$.enterprise.infrastructure.servers[0].name": json.RawMessage(`{"validation":"string","required":true}`),
	},
	"wildcard_patterns": {
		"$.enterprise.departments[*].name": json.RawMessage(`{"validation":"string","required":true}`),
		"$.enterprise.departments[*].teams[*].team_name": json.RawMessage(`{"validation":"string","required":true}`),
		"$.enterprise.infrastructure.servers[*].name": json.RawMessage(`{"validation":"string","required":true}`),
	},
	"group_patterns": {
		"$.enterprise.departments[*].(name|teams)": json.RawMessage(`{"validation":"mixed","description":"Department name or teams"}`),
		"$.enterprise.departments[*].teams[*].(team_name|members)": json.RawMessage(`{"validation":"mixed","description":"Team properties"}`),
		"$.enterprise.infrastructure.servers[*].(name|cpu|memory)": json.RawMessage(`{"validation":"mixed","description":"Server properties"}`),
	},
	"complex_patterns": {
		"$.enterprise.departments[*].teams[*].members[*].(name|email|salary)": json.RawMessage(`{"validation":"mixed","description":"Member properties"}`),
		"$.enterprise.departments[*].teams[*].members[*].role": json.RawMessage(`{"validation":"string","enum":["senior","mid","junior","specialist"]}`),
		"$.enterprise.infrastructure.servers[*].(name|cpu|memory|status)": json.RawMessage(`{"validation":"mixed","description":"Server configuration"}`),
	},
}

// GJSON-based validator for comparison
type GJSONValidator struct {
	patterns map[string]json.RawMessage
}

func NewGJSONValidator(patterns map[string]json.RawMessage) *GJSONValidator {
	return &GJSONValidator{patterns: patterns}
}

func (v *GJSONValidator) Validate(jsonData string) error {
	result := gjson.Parse(jsonData)
	
	// Simple pattern matching for GJSON
	for pattern := range v.patterns {
		if pattern == "$" {
			continue
		}
		
		// Convert our pattern syntax to GJSON syntax
		gjsonPattern := convertToGJSONPattern(pattern)
		
		// Check if this path exists
		if result.Get(gjsonPattern).Exists() {
			// Pattern matched - would call handler here
			continue
		}
	}
	
	return nil
}

// Convert json-schema-path pattern to GJSON pattern
func convertToGJSONPattern(pattern string) string {
	// Simple conversion - this is limited compared to our full pattern support
	gjsonPattern := pattern
	
	// Convert array wildcards - GJSON uses # for wildcards
	gjsonPattern = replaceAll(gjsonPattern, "[*]", "#")
	
	// Remove our special syntax that GJSON doesn't support
	// This is a simplified conversion - full conversion would be complex
	
	return gjsonPattern
}

func replaceAll(s, old, new string) string {
	// Simple string replacement
	result := ""
	for {
		idx := 0
		for i := 0; i <= len(s)-len(old); i++ {
			if s[i:i+len(old)] == old {
				idx = i
				break
			}
		}
		if idx == 0 && !strings.HasPrefix(s, old) {
			result += s
			break
		}
		result += s[:idx] + new
		s = s[idx+len(old):]
	}
	return result
}

// Our json-schema-path validator
type SchemaPathValidator struct {
	patterns map[string]json.RawMessage
}

func NewSchemaPathValidator(patterns map[string]json.RawMessage) *SchemaPathValidator {
	return &SchemaPathValidator{patterns: patterns}
}

func (v *SchemaPathValidator) Validate(jsonData string) error {
	processor := jsonpkg.NewPathExtractor()
	allPaths, err := processor.ExtractPaths(jsonData)
	if err != nil {
		return err
	}
	
	// For each extracted path, check if it matches our patterns
	for _, path := range allPaths {
		if metadata, exists := v.patterns[path]; exists {
			// Get the value at this path
			value, err := processor.ExtractValue(jsonData, path)
			if err == nil && value != nil {
				// Would call handler here with path, value, metadata
				_ = metadata
			}
		}
	}
	
	return nil
}

// Comprehensive benchmark comparing GJSON vs SchemaPath
func BenchmarkGJSONvsSchemaPath(b *testing.B) {
	testJSON := generateBenchmarkJSON()
	
	for configName, patterns := range benchmarkConfigs {
		b.Run(configName, func(b *testing.B) {
			// Test GJSON validator
			gjsonValidator := NewGJSONValidator(patterns)
			
			// Test SchemaPath validator  
			schemaPathValidator := NewSchemaPathValidator(patterns)
			
			b.Run("GJSON", func(b *testing.B) {
				for i := 0; i < b.N; i++ {
					if err := gjsonValidator.Validate(testJSON); err != nil {
						b.Fatalf("GJSON validation failed: %v", err)
					}
				}
			})
			
			b.Run("SchemaPath", func(b *testing.B) {
				for i := 0; i < b.N; i++ {
					if err := schemaPathValidator.Validate(testJSON); err != nil {
						b.Fatalf("SchemaPath validation failed: %v", err)
					}
				}
			})
		})
	}
}

// Pattern matching capability comparison
func BenchmarkPatternCapabilities(b *testing.B) {
	testJSON := generateBenchmarkJSON()
	
	// Test specific pattern capabilities
	capabilities := []struct {
		name     string
		pattern  string
		gjsonCap bool
	}{
		{"Simple Path", "$.enterprise.departments[0].name", true},
		{"Array Index", "$.enterprise.departments[0].teams[0].team_name", true},
		{"Array Wildcard", "$.enterprise.departments[*].name", true},
		{"Group Alternative", "$.enterprise.departments[*].(name|teams)", false},
		{"Deep Wildcard", "$.enterprise.departments[*].teams[*].members[*].name", false},
		{"Complex Group", "$.enterprise.infrastructure.servers[*].(name|cpu|memory)", false},
	}
	
	for _, cap := range capabilities {
		b.Run(cap.name, func(b *testing.B) {
			// Test with SchemaPath
			patterns := map[string]json.RawMessage{
				cap.pattern: json.RawMessage(`{"validation":"test"}`),
			}
			
			schemaPathValidator := NewSchemaPathValidator(patterns)
			
			b.Run("SchemaPath", func(b *testing.B) {
				for i := 0; i < b.N; i++ {
					schemaPathValidator.Validate(testJSON)
				}
			})
			
			if cap.gjsonCap {
				gjsonValidator := NewGJSONValidator(patterns)
				b.Run("GJSON", func(b *testing.B) {
					for i := 0; i < b.N; i++ {
						gjsonValidator.Validate(testJSON)
					}
				})
			}
		})
	}
}

// Memory efficiency comparison
func BenchmarkMemoryEfficiency(b *testing.B) {
	testJSON := generateBenchmarkJSON()
	
	// Large pattern set
	largePatterns := make(map[string]json.RawMessage)
	for i := 0; i < 50; i++ {
		pattern := fmt.Sprintf("$.enterprise.departments[*].teams[*].members[%d].(name|email|salary)", i)
		largePatterns[pattern] = json.RawMessage(`{"validation":"mixed"}`)
	}
	
	b.Run("Large Pattern Set", func(b *testing.B) {
		b.Run("SchemaPath", func(b *testing.B) {
			validator := NewSchemaPathValidator(largePatterns)
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				validator.Validate(testJSON)
			}
		})
		
		b.Run("GJSON", func(b *testing.B) {
			validator := NewGJSONValidator(largePatterns)
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				validator.Validate(testJSON)
			}
		})
	})
}

// Pattern compilation vs runtime performance
func BenchmarkCompilationVsRuntime(b *testing.B) {
	testJSON := generateBenchmarkJSON()
	patterns := benchmarkConfigs["complex_patterns"]
	
	b.Run("Compilation", func(b *testing.B) {
		b.Run("SchemaPath", func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				// Simulate pattern compilation
				for pattern := range patterns {
					expr, err := parser.ParseExpression(pattern)
					if err != nil {
						b.Fatalf("Failed to parse pattern: %v", err)
					}
					_ = expr
				}
			}
		})
	})
	
	b.Run("Runtime", func(b *testing.B) {
		// Pre-compile patterns
		compiledPatterns := make([]*spec.PathExpression, 0, len(patterns))
		for pattern := range patterns {
			expr, err := parser.ParseExpression(pattern)
			if err != nil {
				b.Fatalf("Failed to parse pattern: %v", err)
			}
			compiledPatterns = append(compiledPatterns, expr)
		}
		
		b.Run("SchemaPath", func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				processor := jsonpkg.NewPathExtractor()
				allPaths, _ := processor.ExtractPaths(testJSON)
				
				for _, path := range allPaths {
					for _, expr := range compiledPatterns {
						// Would check if path matches pattern
						_ = expr
						_ = path
					}
				}
			}
		})
	})
}