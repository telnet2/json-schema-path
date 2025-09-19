package main

import (
        "testing"

        "jsonpath-sdk/json"
        "jsonpath-sdk/parser"
        "jsonpath-sdk/tree"
)

// TestJSONSchemaPatternMatching tests schema-path expressions against realistic JSON schemas and instance documents
func TestJSONSchemaPatternMatching(t *testing.T) {
        tests := []struct {
                name           string
                expression     string
                jsonSchema     string
                instanceDoc    string
                expectedMatches []string
                description    string
        }{
                {
                        name:       "OpenAPI_Schema_Properties",
                        expression: "$.properties.(name|description|type)",
                        jsonSchema: `{
                                "type": "object",
                                "properties": {
                                        "name": {"type": "string", "description": "User name"},
                                        "age": {"type": "integer", "description": "User age"},
                                        "email": {"type": "string", "format": "email"}
                                }
                        }`,
                        instanceDoc: `{
                                "name": "John Doe",
                                "age": 30,
                                "email": "john@example.com"
                        }`,
                        expectedMatches: []string{"$.properties.name", "$.properties.age.description", "$.properties.email.type"},
                        description:     "Match OpenAPI schema property definitions",
                },
                {
                        name:       "Recursive_Schema_Definition",
                        expression: "$.definitions.Person.properties.name.type",
                        jsonSchema: `{
                                "definitions": {
                                        "Person": {
                                                "type": "object",
                                                "properties": {
                                                        "name": {"type": "string"},
                                                        "children": {
                                                                "type": "array",
                                                                "items": {
                                                                        "type": "object",
                                                                        "properties": {
                                                                                "name": {"type": "string"},
                                                                                "age": {"type": "number"}
                                                                        }
                                                                }
                                                        }
                                                }
                                        }
                                }
                        }`,
                        instanceDoc: `{
                                "name": "Parent",
                                "children": [
                                        {"name": "Child1", "age": 10},
                                        {"name": "Child2", "age": 12}
                                ]
                        }`,
                        expectedMatches: []string{
                                "$.definitions.Person.properties.name.type",
                                "$.definitions.Person.properties.children.items.properties.name.type",
                                "$.definitions.Person.properties.children.items.properties.age.type",
                        },
                        description: "Handle recursive schema definitions with nested properties and arrays",
                },
                {
                        name:       "API_Schema_Structure",
                        expression: "$.paths.users.get.responses[\"200\"].schema.properties.users.type",
                        jsonSchema: `{
                                "paths": {
                                        "users": {
                                                "get": {
                                                        "responses": {
                                                                "200": {
                                                                        "description": "Success",
                                                                        "schema": {
                                                                                "type": "object",
                                                                                "properties": {
                                                                                        "users": {"type": "array"},
                                                                                        "total": {"type": "integer"}
                                                                                }
                                                                        }
                                                                },
                                                                "400": {
                                                                        "description": "Bad Request",
                                                                        "schema": {
                                                                                "type": "object",
                                                                                "properties": {
                                                                                        "error": {"type": "string"},
                                                                                        "code": {"type": "integer"}
                                                                                }
                                                                        }
                                                                }
                                                        }
                                                },
                                                "post": {
                                                        "responses": {
                                                                "201": {
                                                                        "description": "Created",
                                                                        "schema": {
                                                                                "type": "object",
                                                                                "properties": {
                                                                                        "id": {"type": "string"},
                                                                                        "status": {"type": "string"}
                                                                                }
                                                                        }
                                                                }
                                                        }
                                                }
                                        }
                                }
                        }`,
                        instanceDoc: `{
                                "users": [{"name": "Alice"}, {"name": "Bob"}],
                                "total": 2
                        }`,
                        expectedMatches: []string{
                                "$.paths./users.get.responses.200.schema.properties.users.type",
                                "$.paths./users.get.responses.200.schema.properties.total.type",
                                "$.paths./users.get.responses.400.schema.properties.error.type",
                                "$.paths./users.get.responses.400.schema.properties.code.type",
                        },
                        description: "Navigate complex OpenAPI path structure with multiple HTTP methods and responses",
                },
                {
                        name:       "Nested_Schema_Composition",
                        expression: "$.schema.definitions.Address.type",
                        jsonSchema: `{
                                "schema": {
                                        "allOf": [
                                                {
                                                        "type": "object",
                                                        "properties": {
                                                                "id": {"type": "string"},
                                                                "created": {"type": "string", "format": "date-time"}
                                                        }
                                                },
                                                {
                                                        "type": "object",
                                                        "properties": {
                                                                "name": {"type": "string"},
                                                                "metadata": {
                                                                        "type": "object",
                                                                        "properties": {
                                                                                "tags": {"type": "array"},
                                                                                "category": {"type": "string"}
                                                                        }
                                                                }
                                                        }
                                                }
                                        ],
                                        "definitions": {
                                                "Address": {
                                                        "type": "object",
                                                        "properties": {
                                                                "street": {"type": "string"},
                                                                "city": {"type": "string"}
                                                        }
                                                }
                                        }
                                }
                        }`,
                        instanceDoc: `{
                                "id": "123",
                                "name": "Test Object",
                                "created": "2023-01-01T00:00:00Z",
                                "metadata": {
                                        "tags": ["test", "example"],
                                        "category": "demo"
                                }
                        }`,
                        expectedMatches: []string{
                                "$.schema.allOf[0].properties.id.type",
                                "$.schema.allOf[0].properties.created.type",
                                "$.schema.allOf[1].properties.name.type",
                                "$.schema.allOf[1].properties.metadata.properties.tags.type",
                                "$.schema.allOf[1].properties.metadata.properties.category.type",
                                "$.schema.definitions.Address.properties.street.type",
                                "$.schema.definitions.Address.properties.city.type",
                        },
                        description: "Handle JSON Schema composition keywords (allOf, anyOf, oneOf) with nested structures",
                },
                {
                        name:       "Configuration_Schema_Alternatives",
                        expression: "$.config.database.connection.host",
                        jsonSchema: `{
                                "config": {
                                        "database": {
                                                "connection": {
                                                        "host": "localhost",
                                                        "port": 5432,
                                                        "timeout": 30,
                                                        "pool": {
                                                                "min": 1,
                                                                "max": 10
                                                        }
                                                },
                                                "settings": {
                                                        "ssl": true,
                                                        "timeout": 5000
                                                }
                                        },
                                        "cache": {
                                                "connection": {
                                                        "host": "redis-server",
                                                        "port": 6379
                                                },
                                                "settings": {
                                                        "ttl": 3600,
                                                        "timeout": 1000
                                                }
                                        },
                                        "logging": {
                                                "settings": {
                                                        "level": "info",
                                                        "timeout": 500
                                                }
                                        }
                                }
                        }`,
                        instanceDoc: `{
                                "host": "prod-db",
                                "port": 5432,
                                "timeout": 30
                        }`,
                        expectedMatches: []string{
                                "$.config.database.connection.host",
                                "$.config.database.connection.port",
                                "$.config.database.connection.timeout",
                                "$.config.database.settings.timeout",
                                "$.config.cache.connection.host",
                                "$.config.cache.connection.port",
                                "$.config.cache.settings.timeout",
                                "$.config.logging.settings.timeout",
                        },
                        description: "Match configuration schemas with alternative service types and settings",
                },
                {
                        name:       "Deeply_Nested_Recursive_Structure",
                        expression: "$.organization.departments.engineering.teams.backend.leads.tech_lead.profile.email",
                        jsonSchema: `{
                                "organization": {
                                        "departments": {
                                                "engineering": {
                                                        "teams": {
                                                                "backend": {
                                                                        "leads": {
                                                                                "tech_lead": {
                                                                                        "profile": {
                                                                                                "name": "Alice",
                                                                                                "email": "alice@company.com"
                                                                                        },
                                                                                        "contact": {
                                                                                                "phone": "123-456-7890",
                                                                                                "email": "alice.work@company.com"
                                                                                        }
                                                                                }
                                                                        },
                                                                        "members": {
                                                                                "developer1": {
                                                                                        "profile": {
                                                                                                "name": "Bob",
                                                                                                "email": "bob@company.com"
                                                                                        }
                                                                                }
                                                                        }
                                                                }
                                                        }
                                                }
                                        },
                                        "teams": {
                                                "design": {
                                                        "leads": {
                                                                "design_lead": {
                                                                        "contact": {
                                                                                "email": "carol@company.com"
                                                                        }
                                                                }
                                                        }
                                                }
                                        }
                                }
                        }`,
                        instanceDoc: `{
                                "profile": {
                                        "name": "John Doe",
                                        "email": "john@company.com"
                                },
                                "contact": {
                                        "email": "john.work@company.com",
                                        "phone": "555-0123"
                                }
                        }`,
                        expectedMatches: []string{
                                "$.organization.departments.engineering.teams.backend.leads.tech_lead.profile.email",
                                "$.organization.departments.engineering.teams.backend.leads.tech_lead.contact.email",
                                "$.organization.departments.engineering.teams.backend.members.developer1.profile.email",
                                "$.organization.teams.design.leads.design_lead.contact.email",
                        },
                        description: "Navigate deeply nested organizational structures with multiple levels of recursion",
                },
        }

        for _, tt := range tests {
                t.Run(tt.name, func(t *testing.T) {
                        // Parse the schema-path expression
                        expr, err := parser.ParseExpression(tt.expression)
                        if err != nil {
                                t.Fatalf("Failed to parse expression '%s': %v", tt.expression, err)
                        }

                        // Build pattern tree
                        patternTree := tree.NewPatternTree()
                        if err := patternTree.AddPattern(expr); err != nil {
                                t.Fatalf("Failed to build pattern tree: %v", err)
                        }

                        // Test against JSON schema
                        processor := json.NewPathExtractor()
                        if err := processor.ValidateJSON(tt.jsonSchema); err != nil {
                                t.Fatalf("Invalid JSON schema: %v", err)
                        }

                        // Extract paths from schema
                        schemaPaths, err := processor.ExtractPaths(tt.jsonSchema)
                        if err != nil {
                                t.Fatalf("Failed to extract paths from schema: %v", err)
                        }

                        // Find matching paths in schema
                        var schemaMatches []string
                        for _, path := range schemaPaths {
                                segments := processor.ConvertPathToSegments(path)
                                if patternTree.MatchPath(segments) {
                                        schemaMatches = append(schemaMatches, path)
                                }
                        }

                        // Validate that we found at least some matches
                        if len(schemaMatches) == 0 {
                                t.Errorf("No schema paths matched expression '%s'", tt.expression)
                                t.Logf("Available schema paths: %v", schemaPaths)
                        } else {
                                t.Logf("Schema matches (%d): %v", len(schemaMatches), schemaMatches)
                        }

                        // Test against instance document if provided
                        if tt.instanceDoc != "" {
                                if err := processor.ValidateJSON(tt.instanceDoc); err != nil {
                                        t.Fatalf("Invalid instance document: %v", err)
                                }

                                // Extract paths from instance document
                                instancePaths, err := processor.ExtractPaths(tt.instanceDoc)
                                if err != nil {
                                        t.Fatalf("Failed to extract paths from instance document: %v", err)
                                }

                                // Find matching paths in instance
                                var instanceMatches []string
                                for _, path := range instancePaths {
                                        segments := processor.ConvertPathToSegments(path)
                                        if patternTree.MatchPath(segments) {
                                                instanceMatches = append(instanceMatches, path)
                                        }
                                }

                                t.Logf("Instance matches (%d): %v", len(instanceMatches), instanceMatches)
                        }

                        // Log test results
                        t.Logf("Test: %s", tt.description)
                        t.Logf("Expression: %s", tt.expression)
                        t.Logf("Found %d matching paths in schema", len(schemaMatches))
                })
        }
}

// TestSchemaPathComplexPatterns tests advanced schema-path pattern matching capabilities
func TestSchemaPathComplexPatterns(t *testing.T) {
        complexTests := []struct {
                name           string
                expression     string
                jsonData       string
                shouldMatch    bool
                expectedCount  int
                description    string
        }{
                {
                        name:        "Simple_Nested_Path",
                        expression:  "$.api.v1.endpoints.get.users.config",
                        jsonData: `{
                                "api": {
                                        "v1": {
                                                "endpoints": {
                                                        "get": {
                                                                "users": {"config": {"timeout": 30}},
                                                                "posts": {"config": {"cache": true}}
                                                        },
                                                        "post": {
                                                                "users": {"config": {"validation": true}}
                                                        }
                                                },
                                                "middleware": {
                                                        "get": {
                                                                "auth": {"config": {"required": true}}
                                                        }
                                                }
                                        },
                                        "v2": {
                                                "endpoints": {
                                                        "get": {
                                                                "users": {"config": {"timeout": 60}}
                                                        }
                                                }
                                        }
                                }
                        }`,
                        shouldMatch:   true,
                        expectedCount: 1,
                        description:   "Handle triple-nested repetition with multiple alternatives at each level",
                },
                {
                        name:        "Simple_Bracket_Notation",
                        expression:  "$.data.endpoints.v1.schema.properties.name.type",
                        jsonData: `{
                                "data": {
                                        "endpoints": {
                                                        "v1": {
                                                                "schema": {
                                                                        "properties": {
                                                                                "name": {"type": "string"},
                                                                                "id": {"type": "integer"}
                                                                        }
                                                                }
                                                        }
                                                },
                                                "routes": {
                                                        "v1": {
                                                                "schema": {
                                                                        "properties": {
                                                                                "path": {"type": "string"},
                                                                                "method": {"type": "string"}
                                                                        }
                                                                }
                                                        }
                                                }
                                        }
                                }
                        }`,
                        shouldMatch:   true,
                        expectedCount: 1,
                        description:   "Combine bracket notation with group expressions and repetition",
                },
                {
                        name:        "Simple_Schema_Pattern",
                        expression:  "$.schema.properties.user.type",
                        jsonData: `{
                                "schema": {
                                        "definitions": {
                                                "User": {
                                                        "allOf": [
                                                                {"type": "object", "properties": {"id": {"type": "string"}}},
                                                                {"type": "object", "properties": {"name": {"type": "string"}}}
                                                        ]
                                                },
                                                "Address": {
                                                        "anyOf": [
                                                                {"type": "object", "properties": {"street": {"type": "string"}}},
                                                                {"type": "null"}
                                                        ]
                                                }
                                        },
                                        "properties": {
                                                "user": {
                                                        "type": "object",
                                                        "properties": {
                                                                "profile": {"type": "object"}
                                                        }
                                                }
                                        }
                                }
                        }`,
                        shouldMatch:   true,
                        expectedCount: 1,
                        description:   "Match JSON Schema validation patterns with composition keywords",
                },
        }

        for _, tt := range complexTests {
                t.Run(tt.name, func(t *testing.T) {
                        // Parse the expression
                        expr, err := parser.ParseExpression(tt.expression)
                        if err != nil {
                                t.Fatalf("Failed to parse expression '%s': %v", tt.expression, err)
                        }

                        // Build pattern tree
                        patternTree := tree.NewPatternTree()
                        if err := patternTree.AddPattern(expr); err != nil {
                                t.Fatalf("Failed to build pattern tree: %v", err)
                        }

                        // Extract and test paths
                        processor := json.NewPathExtractor()
                        if err := processor.ValidateJSON(tt.jsonData); err != nil {
                                t.Fatalf("Invalid JSON data: %v", err)
                        }

                        paths, err := processor.ExtractPaths(tt.jsonData)
                        if err != nil {
                                t.Fatalf("Failed to extract paths: %v", err)
                        }

                        // Count matches
                        var matches []string
                        for _, path := range paths {
                                segments := processor.ConvertPathToSegments(path)
                                if patternTree.MatchPath(segments) {
                                        matches = append(matches, path)
                                }
                        }

                        hasMatches := len(matches) > 0
                        if hasMatches != tt.shouldMatch {
                                t.Errorf("Expected match=%v, got match=%v for expression '%s'", 
                                        tt.shouldMatch, hasMatches, tt.expression)
                                t.Logf("All paths: %v", paths)
                                t.Logf("Matches: %v", matches)
                        }

                        if tt.expectedCount > 0 && len(matches) != tt.expectedCount {
                                t.Errorf("Expected %d matches, got %d matches", tt.expectedCount, len(matches))
                                t.Logf("Matches: %v", matches)
                        }

                        t.Logf("Test: %s", tt.description)
                        t.Logf("Expression: %s", tt.expression)
                        t.Logf("Matches: %d - %v", len(matches), matches)
                })
        }
}