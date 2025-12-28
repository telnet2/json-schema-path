package main

import (
	"fmt"
	"testing"

	"jsonpath-sdk/json"
	"jsonpath-sdk/parser"
	"jsonpath-sdk/tree"
)

// TestRecursiveJSONSchemaReferences proves that schema-path can represent
// JSON Schema with recursive $ref patterns
func TestRecursiveJSONSchemaReferences(t *testing.T) {
	tests := []struct {
		name        string
		description string
		expression  string
		jsonSchema  string
		expected    []string
	}{
		{
			name:        "LinkedList_Recursive_Reference",
			description: "JSON Schema with self-referencing LinkedList node (like $ref: '#/definitions/Node')",
			expression:  "$.definitions.Node.(properties.next){*}.properties.value.type",
			jsonSchema: `{
				"definitions": {
					"Node": {
						"type": "object",
						"properties": {
							"value": {"type": "string"},
							"next": {
								"$ref": "#/definitions/Node",
								"properties": {
									"value": {"type": "string"},
									"next": {
										"properties": {
											"value": {"type": "integer"},
											"next": {
												"properties": {
													"value": {"type": "boolean"}
												}
											}
										}
									}
								}
							}
						}
					}
				}
			}`,
			expected: []string{
				"$.definitions.Node.properties.value.type",
				"$.definitions.Node.properties.next.properties.value.type",
				"$.definitions.Node.properties.next.properties.next.properties.value.type",
				"$.definitions.Node.properties.next.properties.next.properties.next.properties.value.type",
			},
		},
		{
			name:        "BinaryTree_Recursive_Reference",
			description: "JSON Schema with binary tree structure (left/right children referencing same type)",
			expression:  "$.(left|right){*}.value",
			jsonSchema: `{
				"type": "object",
				"properties": {
					"value": {"type": "integer"}
				},
				"left": {
					"value": 10,
					"left": {
						"value": 5,
						"right": {"value": 7}
					},
					"right": {"value": 15}
				},
				"right": {
					"value": 20,
					"left": {"value": 18},
					"right": {
						"value": 25,
						"right": {"value": 30}
					}
				}
			}`,
			expected: []string{
				"$.left.value",
				"$.left.left.value",
				"$.left.left.right.value",
				"$.left.right.value",
				"$.right.value",
				"$.right.left.value",
				"$.right.right.value",
				"$.right.right.right.value",
			},
		},
		{
			name:        "JSONSchema_Recursive_Definitions",
			description: "Typical JSON Schema with properties/additionalProperties recursive pattern",
			expression:  "$.(properties|additionalProperties){*}.type",
			jsonSchema: `{
				"type": "object",
				"properties": {
					"name": {"type": "string"},
					"config": {
						"type": "object",
						"properties": {
							"enabled": {"type": "boolean"},
							"nested": {
								"type": "object",
								"additionalProperties": {
									"type": "string"
								}
							}
						},
						"additionalProperties": {
							"type": "integer",
							"properties": {
								"extra": {"type": "number"}
							}
						}
					}
				},
				"additionalProperties": {
					"type": "object",
					"properties": {
						"dynamic": {"type": "array"}
					}
				}
			}`,
			expected: []string{
				"$.properties.name.type",
				"$.properties.config.type",
				"$.properties.config.properties.enabled.type",
				"$.properties.config.properties.nested.type",
				"$.properties.config.properties.nested.additionalProperties.type",
				"$.properties.config.additionalProperties.type",
				"$.properties.config.additionalProperties.properties.extra.type",
				"$.additionalProperties.type",
				"$.additionalProperties.properties.dynamic.type",
			},
		},
		{
			name:        "JSONSchema_Items_Array_Recursion",
			description: "JSON Schema with array items containing nested schemas",
			expression:  "$.(items|properties){*}.type",
			jsonSchema: `{
				"type": "array",
				"items": {
					"type": "object",
					"properties": {
						"id": {"type": "string"},
						"children": {
							"type": "array",
							"items": {
								"type": "object",
								"properties": {
									"name": {"type": "string"},
									"data": {
										"type": "array",
										"items": {"type": "number"}
									}
								}
							}
						}
					}
				}
			}`,
			expected: []string{
				"$.items.type",
				"$.items.properties.id.type",
				"$.items.properties.children.type",
				"$.items.properties.children.items.type",
				"$.items.properties.children.items.properties.name.type",
				"$.items.properties.children.items.properties.data.type",
				"$.items.properties.children.items.properties.data.items.type",
			},
		},
		{
			name:        "OpenAPI_Components_Schemas_Recursive",
			description: "OpenAPI 3.0 components/schemas with recursive definitions",
			expression:  "$.components.schemas.(Employee|Department).(properties|allOf){*}.type",
			jsonSchema: `{
				"components": {
					"schemas": {
						"Employee": {
							"type": "object",
							"properties": {
								"name": {"type": "string"},
								"manager": {
									"type": "object",
									"properties": {
										"name": {"type": "string"},
										"level": {"type": "integer"}
									}
								},
								"department": {
									"allOf": [
										{"type": "object"},
										{
											"properties": {
												"name": {"type": "string"}
											}
										}
									]
								}
							}
						},
						"Department": {
							"type": "object",
							"properties": {
								"name": {"type": "string"},
								"head": {
									"type": "object",
									"allOf": [
										{
											"properties": {
												"title": {"type": "string"}
											}
										}
									]
								}
							}
						}
					}
				}
			}`,
			expected: []string{
				"$.components.schemas.Employee.type",
				"$.components.schemas.Employee.properties.name.type",
				"$.components.schemas.Employee.properties.manager.type",
				"$.components.schemas.Employee.properties.manager.properties.name.type",
				"$.components.schemas.Employee.properties.manager.properties.level.type",
				"$.components.schemas.Employee.properties.department.allOf.0.type",
				"$.components.schemas.Employee.properties.department.allOf.1.properties.name.type",
				"$.components.schemas.Department.type",
				"$.components.schemas.Department.properties.name.type",
				"$.components.schemas.Department.properties.head.type",
				"$.components.schemas.Department.properties.head.allOf.0.properties.title.type",
			},
		},
		{
			name:        "Filesystem_Tree_Structure",
			description: "Recursive directory/file tree structure",
			expression:  "$.(children|subdirs){*}.name",
			jsonSchema: `{
				"name": "root",
				"type": "directory",
				"children": [
					{
						"name": "src",
						"type": "directory",
						"children": [
							{"name": "main.go", "type": "file"},
							{
								"name": "pkg",
								"type": "directory",
								"subdirs": [
									{
										"name": "utils",
										"children": [
											{"name": "helper.go", "type": "file"}
										]
									}
								]
							}
						]
					},
					{
						"name": "docs",
						"subdirs": [
							{
								"name": "api",
								"children": [
									{"name": "readme.md"}
								]
							}
						]
					}
				]
			}`,
			expected: []string{
				"$.children.0.name",
				"$.children.0.children.0.name",
				"$.children.0.children.1.name",
				"$.children.0.children.1.subdirs.0.name",
				"$.children.0.children.1.subdirs.0.children.0.name",
				"$.children.1.name",
				"$.children.1.subdirs.0.name",
				"$.children.1.subdirs.0.children.0.name",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Logf("\n=== %s ===", tt.name)
			t.Logf("Description: %s", tt.description)
			t.Logf("Expression: %s", tt.expression)

			// Parse the expression
			expr, err := parser.ParseExpression(tt.expression)
			if err != nil {
				t.Fatalf("Failed to parse expression: %v", err)
			}
			t.Logf("Parsed AST: %s", expr.String())

			// Build pattern tree
			patternTree := tree.NewPatternTree()
			if err := patternTree.AddPattern(expr); err != nil {
				t.Fatalf("Failed to build pattern tree: %v", err)
			}

			// Extract paths from JSON
			processor := json.NewPathExtractor()
			paths, err := processor.ExtractPaths(tt.jsonSchema)
			if err != nil {
				t.Fatalf("Failed to extract paths: %v", err)
			}

			// Find matching paths
			var matches []string
			for _, path := range paths {
				segments := processor.ConvertPathToSegments(path)
				if patternTree.MatchPath(segments) {
					matches = append(matches, path)
				}
			}

			t.Logf("Total paths in JSON: %d", len(paths))
			t.Logf("Matching paths: %d", len(matches))
			for i, m := range matches {
				t.Logf("  [%d] %s", i+1, m)
			}

			// Verify we got matches
			if len(matches) == 0 {
				t.Errorf("Expected matches but got none!")
				t.Logf("Available paths:")
				for _, p := range paths {
					t.Logf("  - %s", p)
				}
			}
		})
	}
}

// TestRecursiveRepetitionMechanism demonstrates how {*} handles zero-or-more repetitions
func TestRecursiveRepetitionMechanism(t *testing.T) {
	testCases := []struct {
		expression string
		testPaths  []struct {
			path    []string
			matches bool
		}
	}{
		{
			expression: "$.node.(child){*}.value",
			testPaths: []struct {
				path    []string
				matches bool
			}{
				{[]string{"node", "value"}, true},                             // Zero repetitions
				{[]string{"node", "child", "value"}, true},                    // One repetition
				{[]string{"node", "child", "child", "value"}, true},           // Two repetitions
				{[]string{"node", "child", "child", "child", "value"}, true},  // Three repetitions
				{[]string{"node", "other", "value"}, false},                   // Wrong path
			},
		},
		{
			expression: "$.(left|right){*}.data",
			testPaths: []struct {
				path    []string
				matches bool
			}{
				{[]string{"data"}, true},                                      // Zero repetitions
				{[]string{"left", "data"}, true},                              // One left
				{[]string{"right", "data"}, true},                             // One right
				{[]string{"left", "right", "data"}, true},                     // Mixed
				{[]string{"left", "left", "right", "data"}, true},             // Deep mixed
				{[]string{"right", "right", "right", "data"}, true},           // Deep right
				{[]string{"up", "data"}, false},                               // Wrong direction
			},
		},
		{
			expression: "$.(properties|definitions){*}.type",
			testPaths: []struct {
				path    []string
				matches bool
			}{
				{[]string{"type"}, true},                                                        // Zero
				{[]string{"properties", "type"}, true},                                          // One props
				{[]string{"definitions", "type"}, true},                                         // One defs
				{[]string{"properties", "properties", "type"}, true},                            // Nested props
				{[]string{"properties", "definitions", "properties", "type"}, true},             // Mixed deep
				{[]string{"definitions", "properties", "definitions", "type"}, true},            // Alternating
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.expression, func(t *testing.T) {
			// Parse and build tree
			expr, err := parser.ParseExpression(tc.expression)
			if err != nil {
				t.Fatalf("Parse error: %v", err)
			}

			patternTree := tree.NewPatternTree()
			if err := patternTree.AddPattern(expr); err != nil {
				t.Fatalf("Tree build error: %v", err)
			}

			// Test each path
			for _, tp := range tc.testPaths {
				result := patternTree.MatchPath(tp.path)
				pathStr := fmt.Sprintf("$.%s", stringJoin(tp.path, "."))

				if result != tp.matches {
					t.Errorf("Path %s: expected match=%v, got match=%v",
						pathStr, tp.matches, result)
				} else {
					status := "✓"
					if !result {
						status = "✗"
					}
					t.Logf("%s %s (match=%v)", status, pathStr, result)
				}
			}
		})
	}
}

func stringJoin(s []string, sep string) string {
	result := ""
	for i, v := range s {
		if i > 0 {
			result += sep
		}
		result += v
	}
	return result
}
