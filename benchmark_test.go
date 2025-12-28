package main

import (
	"encoding/json"
	"fmt"
	"testing"

	jsonpkg "jsonpath-sdk/json"
	"jsonpath-sdk/parser"
	"jsonpath-sdk/tree"
)

// =============================================================================
// BENCHMARK DATA STRUCTURES
// =============================================================================

// Sample JSON documents of varying complexity
var (
	// Small JSON - simple object
	smallJSON = `{
		"name": "John Doe",
		"age": 30,
		"email": "john@example.com"
	}`

	// Medium JSON - nested object with arrays
	mediumJSON = `{
		"user": {
			"id": "12345",
			"profile": {
				"name": "John Doe",
				"age": 30,
				"email": "john@example.com",
				"address": {
					"street": "123 Main St",
					"city": "New York",
					"zip": "10001"
				}
			},
			"roles": ["admin", "user", "editor"],
			"settings": {
				"theme": "dark",
				"notifications": true
			}
		},
		"metadata": {
			"created": "2024-01-01",
			"updated": "2024-06-15"
		}
	}`

	// Large JSON - complex nested structure simulating OpenAPI schema
	largeJSON = `{
		"openapi": "3.0.0",
		"info": {
			"title": "Sample API",
			"version": "1.0.0",
			"description": "A sample API for benchmarking"
		},
		"paths": {
			"/users": {
				"get": {
					"summary": "Get all users",
					"responses": {
						"200": {
							"description": "Success",
							"content": {
								"application/json": {
									"schema": {
										"type": "array",
										"items": {
											"$ref": "#/components/schemas/User"
										}
									}
								}
							}
						}
					}
				},
				"post": {
					"summary": "Create user",
					"requestBody": {
						"content": {
							"application/json": {
								"schema": {
									"$ref": "#/components/schemas/User"
								}
							}
						}
					},
					"responses": {
						"201": {
							"description": "Created"
						}
					}
				}
			},
			"/users/{id}": {
				"get": {
					"summary": "Get user by ID",
					"parameters": [
						{
							"name": "id",
							"in": "path",
							"required": true,
							"schema": {"type": "string"}
						}
					],
					"responses": {
						"200": {
							"description": "Success",
							"content": {
								"application/json": {
									"schema": {
										"$ref": "#/components/schemas/User"
									}
								}
							}
						}
					}
				}
			}
		},
		"components": {
			"schemas": {
				"User": {
					"type": "object",
					"properties": {
						"id": {"type": "string"},
						"name": {"type": "string"},
						"email": {"type": "string", "format": "email"},
						"profile": {
							"type": "object",
							"properties": {
								"bio": {"type": "string"},
								"avatar": {"type": "string", "format": "uri"},
								"settings": {
									"type": "object",
									"properties": {
										"theme": {"type": "string"},
										"language": {"type": "string"}
									}
								}
							}
						},
						"roles": {
							"type": "array",
							"items": {"type": "string"}
						}
					},
					"required": ["id", "name", "email"]
				},
				"Address": {
					"type": "object",
					"properties": {
						"street": {"type": "string"},
						"city": {"type": "string"},
						"country": {"type": "string"},
						"zip": {"type": "string"}
					}
				}
			}
		}
	}`

	// Deeply recursive JSON - simulating recursive schema structure
	recursiveJSON = `{
		"node": {
			"value": "root",
			"type": "container",
			"child": {
				"value": "level1",
				"type": "container",
				"child": {
					"value": "level2",
					"type": "container",
					"child": {
						"value": "level3",
						"type": "leaf",
						"child": {
							"value": "level4",
							"type": "leaf"
						}
					}
				},
				"meta": {
					"child": {
						"value": "meta-level2",
						"type": "metadata"
					}
				}
			},
			"meta": {
				"child": {
					"value": "meta-level1",
					"type": "metadata",
					"child": {
						"value": "meta-nested",
						"type": "metadata"
					}
				}
			}
		}
	}`
)

// Schema-path expressions of varying complexity
var expressions = []struct {
	name string
	expr string
}{
	{"simple_property", "$.name"},
	{"nested_property", "$.user.profile.name"},
	{"deep_nested", "$.paths.users.get.responses.success.content.schema.type"},
	{"simple_group", "$.(name|email)"},
	{"group_with_path", "$.user.profile.(name|email|age)"},
	{"recursive_simple", "$.(child){*}.value"},
	{"recursive_complex", "$.node.(child|meta.child){*}.value"},
	{"schema_traverse", "$.(properties|items){*}.type"},
	{"openapi_pattern", "$.components.schemas.(User|Address).properties.(name|email|street).type"},
}

// =============================================================================
// PARSING BENCHMARKS
// =============================================================================

// BenchmarkParseExpression measures expression parsing speed
func BenchmarkParseExpression(b *testing.B) {
	for _, expr := range expressions {
		b.Run(expr.name, func(b *testing.B) {
			// Validate expression first
			if _, err := parser.ParseExpression(expr.expr); err != nil {
				b.Skipf("Parse error: %v", err)
				return
			}
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, _ = parser.ParseExpression(expr.expr)
			}
		})
	}
}

// BenchmarkBuildPatternTree measures tree construction speed
func BenchmarkBuildPatternTree(b *testing.B) {
	for _, expr := range expressions {
		b.Run(expr.name, func(b *testing.B) {
			parsed, err := parser.ParseExpression(expr.expr)
			if err != nil {
				b.Skipf("Parse error: %v", err)
				return
			}
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				patternTree := tree.NewPatternTree()
				_ = patternTree.AddPattern(parsed)
			}
		})
	}
}

// =============================================================================
// JSON PROCESSING BENCHMARKS
// =============================================================================

// BenchmarkExtractPaths measures path extraction from JSON documents
func BenchmarkExtractPaths(b *testing.B) {
	testCases := []struct {
		name string
		json string
	}{
		{"small_json", smallJSON},
		{"medium_json", mediumJSON},
		{"large_json", largeJSON},
		{"recursive_json", recursiveJSON},
	}

	processor := jsonpkg.NewPathExtractor()

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				_, err := processor.ExtractPaths(tc.json)
				if err != nil {
					b.Fatalf("Extract error: %v", err)
				}
			}
		})
	}
}

// BenchmarkValidateJSON measures JSON validation speed
func BenchmarkValidateJSON(b *testing.B) {
	testCases := []struct {
		name string
		json string
	}{
		{"small_json", smallJSON},
		{"medium_json", mediumJSON},
		{"large_json", largeJSON},
	}

	processor := jsonpkg.NewPathExtractor()

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				_ = processor.ValidateJSON(tc.json)
			}
		})
	}
}

// =============================================================================
// PATTERN MATCHING BENCHMARKS
// =============================================================================

// BenchmarkMatchPath measures path matching against patterns
func BenchmarkMatchPath(b *testing.B) {
	testCases := []struct {
		name    string
		pattern string
		paths   [][]string
	}{
		{
			name:    "simple_match",
			pattern: "$.user.name",
			paths: [][]string{
				{"user", "name"},
				{"user", "email"},
				{"user", "profile", "name"},
			},
		},
		{
			name:    "group_match",
			pattern: "$.(name|email|age)",
			paths: [][]string{
				{"name"},
				{"email"},
				{"age"},
				{"other"},
			},
		},
		{
			name:    "recursive_match",
			pattern: "$.(child){*}.value",
			paths: [][]string{
				{"value"},
				{"child", "value"},
				{"child", "child", "value"},
				{"child", "child", "child", "value"},
				{"child", "child", "child", "child", "value"},
			},
		},
		{
			name:    "complex_recursive",
			pattern: "$.node.(child|meta.child){*}.value",
			paths: [][]string{
				{"node", "value"},
				{"node", "child", "value"},
				{"node", "meta", "child", "value"},
				{"node", "child", "meta", "child", "value"},
				{"node", "meta", "child", "child", "value"},
			},
		},
	}

	for _, tc := range testCases {
		expr, _ := parser.ParseExpression(tc.pattern)
		patternTree := tree.NewPatternTree()
		_ = patternTree.AddPattern(expr)

		b.Run(tc.name, func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				for _, path := range tc.paths {
					patternTree.MatchPath(path)
				}
			}
		})
	}
}

// =============================================================================
// END-TO-END BENCHMARKS
// =============================================================================

// BenchmarkEndToEnd measures complete workflow: parse -> build tree -> extract paths -> match
func BenchmarkEndToEnd(b *testing.B) {
	testCases := []struct {
		name       string
		expression string
		jsonData   string
	}{
		{
			name:       "simple_small",
			expression: "$.name",
			jsonData:   smallJSON,
		},
		{
			name:       "nested_medium",
			expression: "$.user.profile.name",
			jsonData:   mediumJSON,
		},
		{
			name:       "recursive_complex",
			expression: "$.node.(child|meta.child){*}.value",
			jsonData:   recursiveJSON,
		},
		{
			name:       "schema_large",
			expression: "$.components.schemas.User.properties.(name|email).type",
			jsonData:   largeJSON,
		},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				// Parse expression
				expr, _ := parser.ParseExpression(tc.expression)

				// Build pattern tree
				patternTree := tree.NewPatternTree()
				_ = patternTree.AddPattern(expr)

				// Extract paths from JSON
				processor := jsonpkg.NewPathExtractor()
				paths, _ := processor.ExtractPaths(tc.jsonData)

				// Match each path
				for _, path := range paths {
					segments := processor.ConvertPathToSegments(path)
					patternTree.MatchPath(segments)
				}
			}
		})
	}
}

// BenchmarkEndToEndWithPrebuiltTree measures matching with pre-built tree (typical use case)
func BenchmarkEndToEndPrebuilt(b *testing.B) {
	testCases := []struct {
		name       string
		expression string
		jsonData   string
	}{
		{
			name:       "simple_small",
			expression: "$.name",
			jsonData:   smallJSON,
		},
		{
			name:       "nested_medium",
			expression: "$.user.profile.name",
			jsonData:   mediumJSON,
		},
		{
			name:       "recursive_complex",
			expression: "$.node.(child|meta.child){*}.value",
			jsonData:   recursiveJSON,
		},
		{
			name:       "schema_large",
			expression: "$.components.schemas.User.properties.(name|email).type",
			jsonData:   largeJSON,
		},
	}

	for _, tc := range testCases {
		// Pre-build expression and tree (one-time cost)
		expr, _ := parser.ParseExpression(tc.expression)
		patternTree := tree.NewPatternTree()
		_ = patternTree.AddPattern(expr)
		processor := jsonpkg.NewPathExtractor()

		b.Run(tc.name, func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				// Extract paths from JSON
				paths, _ := processor.ExtractPaths(tc.jsonData)

				// Match each path
				for _, path := range paths {
					segments := processor.ConvertPathToSegments(path)
					patternTree.MatchPath(segments)
				}
			}
		})
	}
}

// =============================================================================
// COMPARISON: SCHEMA-PATH vs STANDARD JSON UNMARSHALING
// =============================================================================

// BenchmarkCompare_Validation compares JSON validation approaches
// This is a FAIR comparison: both approaches validate the same JSON
func BenchmarkCompare_Validation(b *testing.B) {
	testCases := []struct {
		name string
		json string
	}{
		{"small", smallJSON},
		{"medium", mediumJSON},
		{"large", largeJSON},
	}

	for _, tc := range testCases {
		processor := jsonpkg.NewPathExtractor()
		jsonBytes := []byte(tc.json)

		b.Run(tc.name+"_schema_path_sonic", func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				_ = processor.ValidateJSON(tc.json)
			}
		})

		b.Run(tc.name+"_std_unmarshal", func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				var data interface{}
				_ = json.Unmarshal(jsonBytes, &data)
			}
		})
	}
}

// BenchmarkCompare_SingleValueExtraction compares extracting a SINGLE known value
// This is a FAIR comparison: both approaches extract the same value
func BenchmarkCompare_SingleValueExtraction(b *testing.B) {
	jsonData := mediumJSON
	targetPath := "$.user.profile.name"
	jsonBytes := []byte(jsonData)

	processor := jsonpkg.NewPathExtractor()

	b.Run("schema_path_direct", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			// Direct extraction without pattern matching
			_, _ = processor.ExtractValue(jsonData, targetPath)
		}
	})

	b.Run("std_unmarshal_navigate", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			var data map[string]interface{}
			_ = json.Unmarshal(jsonBytes, &data)
			if user, ok := data["user"].(map[string]interface{}); ok {
				if profile, ok := user["profile"].(map[string]interface{}); ok {
					_ = profile["name"]
				}
			}
		}
	})
}

// BenchmarkCompare_PatternMatching compares pattern-based path discovery
// This is WHERE SCHEMA-PATH EXCELS: finding all paths matching a pattern
func BenchmarkCompare_PatternMatching(b *testing.B) {
	jsonData := largeJSON
	jsonBytes := []byte(jsonData)

	// Pattern: find all "type" fields in schema definitions
	pattern := "$.(properties|items){*}.type"
	expr, _ := parser.ParseExpression(pattern)
	patternTree := tree.NewPatternTree()
	_ = patternTree.AddPattern(expr)
	processor := jsonpkg.NewPathExtractor()

	b.Run("schema_path_pattern", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			paths, _ := processor.ExtractPaths(jsonData)
			matches := 0
			for _, path := range paths {
				segments := processor.ConvertPathToSegments(path)
				if patternTree.MatchPath(segments) {
					matches++
				}
			}
			_ = matches
		}
	})

	b.Run("std_unmarshal_manual_traverse", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			var data map[string]interface{}
			_ = json.Unmarshal(jsonBytes, &data)
			matches := 0
			// Manual recursive traversal to find all "type" fields under properties/items
			var traverse func(v interface{}, depth int)
			traverse = func(v interface{}, depth int) {
				switch val := v.(type) {
				case map[string]interface{}:
					if _, hasType := val["type"]; hasType {
						matches++
					}
					if props, ok := val["properties"].(map[string]interface{}); ok {
						traverse(props, depth+1)
					}
					if items, ok := val["items"].(map[string]interface{}); ok {
						traverse(items, depth+1)
					}
					for _, child := range val {
						if m, ok := child.(map[string]interface{}); ok {
							traverse(m, depth+1)
						}
					}
				}
			}
			traverse(data, 0)
			_ = matches
		}
	})
}

// BenchmarkCompare_MultiDocumentMatching simulates real-world validation scenario:
// Pre-compile pattern ONCE, then match against MANY documents
func BenchmarkCompare_MultiDocumentMatching(b *testing.B) {
	// Simulate 100 JSON documents to validate
	documents := make([]string, 100)
	for i := 0; i < 100; i++ {
		documents[i] = mediumJSON // Use same structure for consistency
	}

	pattern := "$.user.profile.(name|email)"
	expr, _ := parser.ParseExpression(pattern)
	patternTree := tree.NewPatternTree()
	_ = patternTree.AddPattern(expr)
	processor := jsonpkg.NewPathExtractor()

	b.Run("schema_path_precompiled", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			totalMatches := 0
			for _, doc := range documents {
				paths, _ := processor.ExtractPaths(doc)
				for _, path := range paths {
					segments := processor.ConvertPathToSegments(path)
					if patternTree.MatchPath(segments) {
						totalMatches++
					}
				}
			}
			_ = totalMatches
		}
	})

	b.Run("std_unmarshal_each_doc", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			totalMatches := 0
			for _, doc := range documents {
				var data map[string]interface{}
				_ = json.Unmarshal([]byte(doc), &data)
				if user, ok := data["user"].(map[string]interface{}); ok {
					if profile, ok := user["profile"].(map[string]interface{}); ok {
						if _, ok := profile["name"]; ok {
							totalMatches++
						}
						if _, ok := profile["email"]; ok {
							totalMatches++
						}
					}
				}
			}
			_ = totalMatches
		}
	})
}

// BenchmarkCompare_MemoryUsage compares memory allocation patterns
func BenchmarkCompare_MemoryUsage(b *testing.B) {
	testCases := []struct {
		name string
		json string
	}{
		{"small", smallJSON},
		{"medium", mediumJSON},
		{"large", largeJSON},
	}

	for _, tc := range testCases {
		processor := jsonpkg.NewPathExtractor()
		jsonBytes := []byte(tc.json)

		b.Run(tc.name+"_schema_path_parse", func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				_, _ = processor.ExtractPaths(tc.json)
			}
		})

		b.Run(tc.name+"_std_unmarshal", func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				var data interface{}
				_ = json.Unmarshal(jsonBytes, &data)
			}
		})
	}
}

// =============================================================================
// SCALABILITY BENCHMARKS
// =============================================================================

// BenchmarkScalability_PathDepth measures performance with increasing path depth
func BenchmarkScalability_PathDepth(b *testing.B) {
	depths := []int{1, 2, 5, 10, 20}

	for _, depth := range depths {
		// Generate JSON with given depth
		jsonData := generateDeepJSON(depth)
		pattern := generateDeepPattern(depth)

		expr, _ := parser.ParseExpression(pattern)
		patternTree := tree.NewPatternTree()
		_ = patternTree.AddPattern(expr)
		processor := jsonpkg.NewPathExtractor()

		b.Run(fmt.Sprintf("depth_%d", depth), func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				paths, _ := processor.ExtractPaths(jsonData)
				for _, path := range paths {
					segments := processor.ConvertPathToSegments(path)
					patternTree.MatchPath(segments)
				}
			}
		})
	}
}

// BenchmarkScalability_RecursionDepth measures performance with recursive patterns
func BenchmarkScalability_RecursionDepth(b *testing.B) {
	depths := []int{1, 2, 5, 10}

	pattern := "$.(child){*}.value"
	expr, _ := parser.ParseExpression(pattern)
	patternTree := tree.NewPatternTree()
	_ = patternTree.AddPattern(expr)

	for _, depth := range depths {
		path := generateRecursivePath(depth)

		b.Run(fmt.Sprintf("recursion_%d", depth), func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				patternTree.MatchPath(path)
			}
		})
	}
}

// BenchmarkScalability_AlternativeCount measures performance with multiple alternatives
func BenchmarkScalability_AlternativeCount(b *testing.B) {
	counts := []int{2, 5, 10, 20}

	for _, count := range counts {
		pattern := generateAlternativePattern(count)
		paths := generateAlternativePaths(count)

		expr, _ := parser.ParseExpression(pattern)
		patternTree := tree.NewPatternTree()
		_ = patternTree.AddPattern(expr)

		b.Run(fmt.Sprintf("alternatives_%d", count), func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				for _, path := range paths {
					patternTree.MatchPath(path)
				}
			}
		})
	}
}

// =============================================================================
// HELPER FUNCTIONS
// =============================================================================

func generateDeepJSON(depth int) string {
	if depth <= 0 {
		return `{"value": "leaf"}`
	}

	result := `{"level": {`
	for i := 0; i < depth-1; i++ {
		result += `"nested": {`
	}
	result += `"value": "deep"`
	for i := 0; i < depth-1; i++ {
		result += `}`
	}
	result += `}}`
	return result
}

func generateDeepPattern(depth int) string {
	pattern := "$"
	pattern += ".level"
	for i := 0; i < depth-1; i++ {
		pattern += ".nested"
	}
	pattern += ".value"
	return pattern
}

func generateRecursivePath(depth int) []string {
	path := make([]string, 0, depth+1)
	for i := 0; i < depth; i++ {
		path = append(path, "child")
	}
	path = append(path, "value")
	return path
}

func generateAlternativePattern(count int) string {
	pattern := "$.("
	for i := 0; i < count; i++ {
		if i > 0 {
			pattern += "|"
		}
		pattern += fmt.Sprintf("prop%d", i)
	}
	pattern += ")"
	return pattern
}

func generateAlternativePaths(count int) [][]string {
	paths := make([][]string, count)
	for i := 0; i < count; i++ {
		paths[i] = []string{fmt.Sprintf("prop%d", i)}
	}
	return paths
}

// =============================================================================
// MEMORY BENCHMARKS
// =============================================================================

// BenchmarkMemory_TreeSize measures memory usage of pattern trees
func BenchmarkMemory_TreeSize(b *testing.B) {
	patterns := []struct {
		name    string
		pattern string
	}{
		{"simple", "$.name"},
		{"nested", "$.user.profile.settings.theme"},
		{"group_small", "$.(a|b|c)"},
		{"group_large", "$.(a|b|c|d|e|f|g|h|i|j)"},
		{"recursive", "$.(child){*}.value"},
		{"complex", "$.node.(child|meta.child){*}.(value|type)"},
	}

	for _, p := range patterns {
		b.Run(p.name, func(b *testing.B) {
			expr, _ := parser.ParseExpression(p.pattern)
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				patternTree := tree.NewPatternTree()
				_ = patternTree.AddPattern(expr)
				_ = patternTree // prevent optimization
			}
		})
	}
}
