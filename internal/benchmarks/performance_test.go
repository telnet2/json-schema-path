package benchmarks

import (
        "fmt"
        "strings"
        "testing"

        "jsonpath-sdk/internal/json"
        "jsonpath-sdk/internal/parser"
        "jsonpath-sdk/internal/spec"
        "jsonpath-sdk/internal/tree"
)

// BenchmarkExpressionParsing benchmarks the parsing performance
func BenchmarkExpressionParsing(b *testing.B) {
        expressions := []string{
                "$.user.name",
                "$.users[0].profile.email",
                "$.data.(items|results)[*].(id|name)",
                "$.api.responses.(data|error).items[*].(id|name|description)",
                "$.node.(child|meta.child){*}.value",
                "$.complex.(a|b.(c|d.(e|f))).deep.nested.value",
        }

        for _, expr := range expressions {
                b.Run(fmt.Sprintf("parse_%s", expr), func(b *testing.B) {
                        for i := 0; i < b.N; i++ {
                                _, err := parser.ParseExpression(expr)
                                if err != nil {
                                        b.Fatalf("Parse error: %v", err)
                                }
                        }
                })
        }
}

// BenchmarkPatternTreeBuilding benchmarks tree construction performance
func BenchmarkPatternTreeBuilding(b *testing.B) {
        expressions := []string{
                "$.user.name",
                "$.users[0].profile.email", 
                "$.data.(items|results)[*].(id|name)",
                "$.api.responses.(data|error).items[*].(id|name|description)",
                "$.node.(child|meta.child){*}.value",
        }

        // Pre-parse expressions
        parsedExprs := make([]*spec.PathExpression, len(expressions))
        for i, expr := range expressions {
                parsed, err := parser.ParseExpression(expr)
                if err != nil {
                        b.Fatalf("Parse error: %v", err)
                }
                parsedExprs[i] = parsed
        }

        for i, expr := range expressions {
                b.Run(fmt.Sprintf("build_%s", expr), func(b *testing.B) {
                        for j := 0; j < b.N; j++ {
                                patternTree := tree.NewPatternTree()
                                err := patternTree.AddPattern(parsedExprs[i])
                                if err != nil {
                                        b.Fatalf("Tree build error: %v", err)
                                }
                        }
                })
        }
}

// BenchmarkJSONProcessing benchmarks JSON parsing and path extraction
func BenchmarkJSONProcessing(b *testing.B) {
        // Create test JSON data of various sizes
        jsonSizes := []struct {
                name string
                data string
        }{
                {
                        name: "small_object",
                        data: `{"user": {"name": "John", "email": "john@test.com"}}`,
                },
                {
                        name: "medium_nested",
                        data: generateNestedJSON(5, 3), // 5 levels deep, 3 properties per level
                },
                {
                        name: "large_array",
                        data: generateArrayJSON(100), // Array with 100 objects
                },
        }

        for _, size := range jsonSizes {
                b.Run(fmt.Sprintf("validate_%s", size.name), func(b *testing.B) {
                        processor := json.NewPathExtractor()
                        for i := 0; i < b.N; i++ {
                                err := processor.ValidateJSON(size.data)
                                if err != nil {
                                        b.Fatalf("Validation error: %v", err)
                                }
                        }
                })

                b.Run(fmt.Sprintf("extract_%s", size.name), func(b *testing.B) {
                        processor := json.NewPathExtractor()
                        for i := 0; i < b.N; i++ {
                                _, err := processor.ExtractPaths(size.data)
                                if err != nil {
                                        b.Fatalf("Path extraction error: %v", err)
                                }
                        }
                })
        }
}

// BenchmarkPatternMatching benchmarks the core pattern matching algorithms
func BenchmarkPatternMatching(b *testing.B) {
        // Setup test data
        jsonData := generateComplexJSON(50) // Complex JSON with 50 nested elements
        processor := json.NewPathExtractor()
        paths, _ := processor.ExtractPaths(jsonData)

        expressions := []string{
                "$.data.users[*].name",
                "$.data.(users|admins)[*].(name|email)",
                "$.api.responses[*].(data|error).items[*].metadata.(id|type)",
                "$.complex.(a|b.c.(d|e.f)){*}.value",
        }

        for _, expr := range expressions {
                // Pre-build pattern tree
                parsed, _ := parser.ParseExpression(expr)
                patternTree := tree.NewPatternTree()
                patternTree.AddPattern(parsed)

                b.Run(fmt.Sprintf("match_%s", expr), func(b *testing.B) {
                        for i := 0; i < b.N; i++ {
                                matchCount := 0
                                for _, path := range paths {
                                        segments := processor.ConvertPathToSegments(path)
                                        if patternTree.MatchPath(segments) {
                                                matchCount++
                                        }
                                }
                                // Prevent optimization
                                _ = matchCount
                        }
                })
        }
}

// BenchmarkMemoryUsage benchmarks memory efficiency
func BenchmarkMemoryUsage(b *testing.B) {
        // Test memory usage with large pattern trees
        expressions := []string{
                "$.data.(a|b|c|d|e)[*].(name|id|type|status|value)",
                "$.api.(v1|v2|v3).(users|posts|comments)[*].(id|created|updated)",
                "$.nested.(level1.(level2.(level3|level4)|level5)|level6).value",
        }

        for _, expr := range expressions {
                b.Run(fmt.Sprintf("memory_%s", expr), func(b *testing.B) {
                        b.ReportAllocs()
                        for i := 0; i < b.N; i++ {
                                // Parse expression
                                parsed, _ := parser.ParseExpression(expr)
                                
                                // Build tree
                                patternTree := tree.NewPatternTree()
                                patternTree.AddPattern(parsed)
                                
                                // Test with sample paths
                                testPaths := [][]string{
                                        {"data", "a", "0", "name"},
                                        {"data", "b", "1", "id"},
                                        {"api", "v1", "users", "0", "created"},
                                }
                                
                                for _, pathSegments := range testPaths {
                                        patternTree.MatchPath(pathSegments)
                                }
                        }
                })
        }
}

// BenchmarkConcurrentAccess benchmarks thread safety and concurrent performance
func BenchmarkConcurrentAccess(b *testing.B) {
        // Setup shared pattern tree
        expr, _ := parser.ParseExpression("$.data.(users|posts)[*].(id|name|created)")
        patternTree := tree.NewPatternTree()
        patternTree.AddPattern(expr)
        
        processor := json.NewPathExtractor()
        testPaths := [][]string{
                {"data", "users", "0", "id"},
                {"data", "users", "1", "name"},
                {"data", "posts", "0", "created"},
                {"data", "posts", "1", "id"},
        }

        b.RunParallel(func(pb *testing.PB) {
                for pb.Next() {
                        for _, pathSegments := range testPaths {
                                _ = patternTree.MatchPath(pathSegments)
                        }
                }
        })
}

// Helper functions to generate test data
func generateNestedJSON(depth, width int) string {
        if depth == 0 {
                return `"leaf_value"`
        }

        var props []string
        for i := 0; i < width; i++ {
                key := fmt.Sprintf("prop_%d", i)
                value := generateNestedJSON(depth-1, width)
                props = append(props, fmt.Sprintf(`"%s": %s`, key, value))
        }

        return fmt.Sprintf("{%s}", strings.Join(props, ", "))
}

func generateArrayJSON(size int) string {
        var items []string
        for i := 0; i < size; i++ {
                item := fmt.Sprintf(`{
                        "id": %d,
                        "name": "User_%d",
                        "email": "user_%d@test.com",
                        "active": %v,
                        "profile": {
                                "bio": "Bio for user %d",
                                "preferences": {
                                        "theme": "dark",
                                        "notifications": true
                                }
                        }
                }`, i, i, i, i%2 == 0, i)
                items = append(items, item)
        }

        return fmt.Sprintf(`{"users": [%s]}`, strings.Join(items, ", "))
}

func generateComplexJSON(complexity int) string {
        // Generate a complex nested structure for benchmarking
        return fmt.Sprintf(`{
                "data": %s,
                "api": {
                        "responses": [
                                {"data": {"items": [{"id": 1, "metadata": {"type": "user"}}]}},
                                {"error": {"code": 404, "items": [{"id": 2, "metadata": {"type": "error"}}]}}
                        ]
                },
                "complex": %s
        }`, generateArrayJSON(complexity), generateNestedJSON(4, 3))
}