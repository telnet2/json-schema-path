package tree

import (
        "testing"

        "jsonpath-sdk/pkg/schemapath/parser"
)

func TestPatternTreeBasicOperations(t *testing.T) {
        tree := NewPatternTree()
        
        // Test basic tree structure
        if tree.Root == nil {
                t.Error("Expected root node, got nil")
        }
        
        if tree.Root.Type != NodeRoot {
                t.Errorf("Expected root type, got %v", tree.Root.Type)
        }
}

func TestAddBasicPattern(t *testing.T) {
        tests := []struct {
                name       string
                expression string
        }{
                {"simple property", "$.node"},
                {"nested properties", "$.node.child.value"},
                {"bracket notation", `$["property"]`},
                {"mixed notation", `$.node["child"].value`},
        }

        for _, tt := range tests {
                t.Run(tt.name, func(t *testing.T) {
                        tree := NewPatternTree()
                        
                        // Parse the expression
                        expr, err := parser.ParseExpression(tt.expression)
                        if err != nil {
                                t.Fatalf("Failed to parse expression %s: %v", tt.expression, err)
                        }
                        
                        // Add to tree
                        err = tree.AddPattern(expr)
                        if err != nil {
                                t.Fatalf("Failed to add pattern %s: %v", tt.expression, err)
                        }
                        
                        // Verify tree is not empty
                        if len(tree.Root.Children) == 0 {
                                t.Error("Expected tree to have children after adding pattern")
                        }
                })
        }
}

func TestAddGroupPattern(t *testing.T) {
        tree := NewPatternTree()
        
        // Parse a group expression
        expr, err := parser.ParseExpression("$.node.(child|value)")
        if err != nil {
                t.Fatalf("Failed to parse group expression: %v", err)
        }
        
        // Add to tree
        err = tree.AddPattern(expr)
        if err != nil {
                t.Fatalf("Failed to add group pattern: %v", err)
        }
        
        // Verify tree structure
        if len(tree.Root.Children) == 0 {
                t.Error("Expected tree to have children")
        }
        
        // Should have a property node for "node"
        nodeChild := tree.Root.Children[0]
        if nodeChild.Type != NodeProperty || nodeChild.Value != "node" {
                t.Errorf("Expected property node 'node', got type %v value %s", nodeChild.Type, nodeChild.Value)
        }
        
        // Should have a group node under "node"
        if len(nodeChild.Children) == 0 {
                t.Error("Expected 'node' to have children")
        }
        
        groupChild := nodeChild.Children[0]
        if groupChild.Type != NodeGroup {
                t.Errorf("Expected group node, got type %v", groupChild.Type)
        }
        
        // Group should have alternatives
        if len(groupChild.Alternatives) != 2 {
                t.Errorf("Expected 2 alternatives, got %d", len(groupChild.Alternatives))
        }
}

func TestMatchBasicPaths(t *testing.T) {
        tree := NewPatternTree()
        
        // Add patterns
        patterns := []string{
                "$.user.name",
                "$.user.email", 
                `$.data["key"]`,
        }
        
        for _, pattern := range patterns {
                expr, err := parser.ParseExpression(pattern)
                if err != nil {
                        t.Fatalf("Failed to parse pattern %s: %v", pattern, err)
                }
                err = tree.AddPattern(expr)
                if err != nil {
                        t.Fatalf("Failed to add pattern %s: %v", pattern, err)
                }
        }
        
        // Test matching paths
        testCases := []struct {
                path     []string
                expected bool
                name     string
        }{
                {[]string{"user", "name"}, true, "user.name should match"},
                {[]string{"user", "email"}, true, "user.email should match"},
                {[]string{"data", "key"}, true, "data[key] should match"},
                {[]string{"user", "age"}, false, "user.age should not match"},
                {[]string{"other", "value"}, false, "other.value should not match"},
                {[]string{"user"}, false, "incomplete path should not match"},
        }
        
        for _, tc := range testCases {
                t.Run(tc.name, func(t *testing.T) {
                        result := tree.MatchPath(tc.path)
                        if result != tc.expected {
                                t.Errorf("MatchPath(%v) = %v, expected %v", tc.path, result, tc.expected)
                        }
                })
        }
}

func TestMatchGroupPaths(t *testing.T) {
        tree := NewPatternTree()
        
        // Add a group pattern
        expr, err := parser.ParseExpression("$.node.(child|value)")
        if err != nil {
                t.Fatalf("Failed to parse group pattern: %v", err)
        }
        err = tree.AddPattern(expr)
        if err != nil {
                t.Fatalf("Failed to add group pattern: %v", err)
        }
        
        // Test matching
        testCases := []struct {
                path     []string
                expected bool
                name     string
        }{
                {[]string{"node", "child"}, true, "node.child should match"},
                {[]string{"node", "value"}, true, "node.value should match"},
                {[]string{"node", "other"}, false, "node.other should not match"},
                {[]string{"other", "child"}, false, "other.child should not match"},
        }
        
        for _, tc := range testCases {
                t.Run(tc.name, func(t *testing.T) {
                        result := tree.MatchPath(tc.path)
                        if result != tc.expected {
                                t.Errorf("MatchPath(%v) = %v, expected %v", tc.path, result, tc.expected)
                        }
                })
        }
}

func TestTreeStringRepresentation(t *testing.T) {
        tree := NewPatternTree()
        
        // Add a simple pattern
        expr, err := parser.ParseExpression("$.node.child")
        if err != nil {
                t.Fatalf("Failed to parse expression: %v", err)
        }
        
        err = tree.AddPattern(expr)
        if err != nil {
                t.Fatalf("Failed to add pattern: %v", err)
        }
        
        // Get string representation
        result := tree.String()
        if result == "" {
                t.Error("Expected non-empty string representation")
        }
        
        t.Logf("Tree string representation:\n%s", result)
}

func TestComplexGroupWithRepetition(t *testing.T) {
        tree := NewPatternTree()
        
        // Add the complex recursive pattern from the specification
        expr, err := parser.ParseExpression("$.node.(child|meta.child){*}.value")
        if err != nil {
                t.Fatalf("Failed to parse complex pattern: %v", err)
        }
        
        err = tree.AddPattern(expr)
        if err != nil {
                t.Fatalf("Failed to add complex pattern: %v", err)
        }
        
        // Verify the tree structure
        treeStr := tree.String()
        if treeStr == "" {
                t.Error("Expected non-empty tree representation")
        }
        
        t.Logf("Complex tree structure:\n%s", treeStr)
        
        // Basic structure validation
        if len(tree.Root.Children) == 0 {
                t.Error("Expected tree to have children")
        }
}

func TestBracketPatternMatching(t *testing.T) {
        tree := NewPatternTree()
        
        // Add bracket patterns
        patterns := []string{
                `$["quoted-key"]`,
                `$[unquoted]`,
                `$.node["child"]`,
        }
        
        for _, pattern := range patterns {
                expr, err := parser.ParseExpression(pattern)
                if err != nil {
                        t.Fatalf("Failed to parse pattern %s: %v", pattern, err)
                }
                err = tree.AddPattern(expr)
                if err != nil {
                        t.Fatalf("Failed to add pattern %s: %v", pattern, err)
                }
        }
        
        // Test basic bracket matching
        testCases := []struct {
                path     []string
                expected bool
                name     string
        }{
                {[]string{"quoted-key"}, true, "quoted key should match"},
                {[]string{"unquoted"}, true, "unquoted key should match"},
                {[]string{"node", "child"}, true, "node.child bracket should match"},
                {[]string{"node", "other"}, false, "node.other should not match"},
                {[]string{"wrong"}, false, "wrong key should not match"},
        }
        
        for _, tc := range testCases {
                t.Run(tc.name, func(t *testing.T) {
                        result := tree.MatchPath(tc.path)
                        if result != tc.expected {
                                t.Errorf("MatchPath(%v) = %v, expected %v", tc.path, result, tc.expected)
                        }
                })
        }
}

func TestGroupWithSuffixMatching(t *testing.T) {
        // Test critical case: group followed by suffix ($.node.(a|b).c)
        tree := NewPatternTree()
        
        // Add pattern with group followed by property
        expr, err := parser.ParseExpression("$.node.(child|value).suffix")
        if err != nil {
                t.Fatalf("Failed to parse group with suffix pattern: %v", err)
        }
        err = tree.AddPattern(expr)
        if err != nil {
                t.Fatalf("Failed to add group with suffix pattern: %v", err)
        }
        
        // Test matching
        testCases := []struct {
                path     []string
                expected bool
                name     string
        }{
                {[]string{"node", "child", "suffix"}, true, "node.child.suffix should match"},
                {[]string{"node", "value", "suffix"}, true, "node.value.suffix should match"},
                {[]string{"node", "child"}, false, "node.child without suffix should not match"},
                {[]string{"node", "value"}, false, "node.value without suffix should not match"},
                {[]string{"node", "other", "suffix"}, false, "node.other.suffix should not match"},
                {[]string{"node", "child", "wrong"}, false, "node.child.wrong should not match"},
        }
        
        for _, tc := range testCases {
                t.Run(tc.name, func(t *testing.T) {
                        result := tree.MatchPath(tc.path)
                        if result != tc.expected {
                                t.Errorf("MatchPath(%v) = %v, expected %v", tc.path, result, tc.expected)
                        }
                })
        }
}

func TestRepetitionWithSuffixMatching(t *testing.T) {
        // Test critical case: repetition with suffix ($.node.(a|b){*}.c)
        tree := NewPatternTree()
        
        // Add pattern with repetition followed by suffix
        expr, err := parser.ParseExpression("$.node.(child|value){*}.suffix")
        if err != nil {
                t.Fatalf("Failed to parse repetition with suffix pattern: %v", err)
        }
        err = tree.AddPattern(expr)
        if err != nil {
                t.Fatalf("Failed to add repetition with suffix pattern: %v", err)
        }
        
        // Test matching with different iteration counts
        testCases := []struct {
                path     []string
                expected bool
                name     string
        }{
                // Zero iterations (should match suffix directly)
                {[]string{"node", "suffix"}, true, "zero iterations should match"},
                
                // One iteration
                {[]string{"node", "child", "suffix"}, true, "one iteration with child should match"},
                {[]string{"node", "value", "suffix"}, true, "one iteration with value should match"},
                
                // Multiple iterations
                {[]string{"node", "child", "child", "suffix"}, true, "two child iterations should match"},
                {[]string{"node", "value", "child", "suffix"}, true, "value then child should match"},
                {[]string{"node", "child", "value", "child", "suffix"}, true, "multiple mixed iterations should match"},
                
                // Invalid cases
                {[]string{"node", "child"}, false, "missing suffix should not match"},
                {[]string{"node", "other", "suffix"}, false, "invalid alternative should not match"},
                {[]string{"node", "child", "other", "suffix"}, false, "invalid iteration should not match"},
                {[]string{"node", "child", "suffix", "extra"}, false, "extra segments should not match"},
        }
        
        for _, tc := range testCases {
                t.Run(tc.name, func(t *testing.T) {
                        result := tree.MatchPath(tc.path)
                        if result != tc.expected {
                                t.Errorf("MatchPath(%v) = %v, expected %v", tc.path, result, tc.expected)
                        }
                })
        }
}

func TestComplexNestedPatterns(t *testing.T) {
        // Test the specification's complex example
        tree := NewPatternTree()
        
        // Add the complex pattern from specification
        expr, err := parser.ParseExpression("$.node.(child|meta.child){*}.value")
        if err != nil {
                t.Fatalf("Failed to parse complex nested pattern: %v", err)
        }
        err = tree.AddPattern(expr)
        if err != nil {
                t.Fatalf("Failed to add complex nested pattern: %v", err)
        }
        
        testCases := []struct {
                path     []string
                expected bool
                name     string
        }{
                // Zero iterations
                {[]string{"node", "value"}, true, "direct to value should match"},
                
                // One iteration - simple child
                {[]string{"node", "child", "value"}, true, "one child iteration should match"},
                
                // One iteration - meta.child
                {[]string{"node", "meta", "child", "value"}, true, "one meta.child iteration should match"},
                
                // Multiple iterations
                {[]string{"node", "child", "child", "value"}, true, "two child iterations should match"},
                {[]string{"node", "meta", "child", "child", "value"}, true, "meta.child then child should match"},
                {[]string{"node", "child", "meta", "child", "value"}, true, "child then meta.child should match"},
                
                // Invalid cases
                {[]string{"node", "child"}, false, "incomplete path should not match"},
                {[]string{"node", "meta", "value"}, false, "incomplete meta path should not match"},
                {[]string{"node", "other", "value"}, false, "invalid alternative should not match"},
        }
        
        for _, tc := range testCases {
                t.Run(tc.name, func(t *testing.T) {
                        result := tree.MatchPath(tc.path)
                        if result != tc.expected {
                                t.Errorf("MatchPath(%v) = %v, expected %v", tc.path, result, tc.expected)
                        }
                })
        }
}