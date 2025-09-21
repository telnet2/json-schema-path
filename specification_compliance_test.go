package main

import (
	"fmt"
	"testing"

	"github.com/telnet2/json-schema-path/parser"
)

// TestSpecificationCompliance verifies our parser supports the syntax described in SPECIFICATION.md
func TestSpecificationCompliance(t *testing.T) {
	// Test cases from the specification
	testCases := []struct {
		name        string
		pattern     string
		shouldParse bool
		description string
	}{
		// Basic patterns from specification
		{"Simple Property", "$.user.name", true, "Basic property access"},
		{"Array Index", "$.users[0]", true, "Array index access"},
		{"Array Wildcard", "$.users[*]", true, "Array wildcard"},
		
		// Bracket notation from specification
		{"Bracket Property", "$.user[\"name\"]", true, "Quoted property in brackets"},
		{"Bracket Property Simple", "$.user[name]", true, "Simple property in brackets"},
		
		// Wildcard patterns from specification
		{"Property Wildcard", "$.config[#*service]", true, "Properties ending with 'service'"},
		{"Property Wildcard Prefix", "$.config[#admin*]", true, "Properties starting with 'admin'"},
		{"Property Wildcard Contains", "$.config[#*user*]", true, "Properties containing 'user'"},
		
		// Regex patterns from specification  
		{"Regex Pattern", "$.fields[~^user_.*]", true, "Regex pattern matching"},
		{"Regex Simple", "$.user[~admin]", true, "Simple regex pattern"},
		
		// Group patterns from specification
		{"Group Alternatives", "$.user.(name|email)", true, "Group with alternatives"},
		{"Group Complex", "$.data.(items|products)[*].id", true, "Complex group with wildcards"},
		
		// Repetition patterns from specification
		{"Repetition Simple", "$.meta{*}", true, "Simple repetition"},
		{"Repetition Complex", "$.node.(child|meta.child){*}.value", true, "Complex repetition from spec"},
		
		// Examples from specification
		{"Spec Example 1", "$.node.(child|meta.child){*}.value", true, "Example from spec"},
		{"Spec Example 2", "$.config[#*service].instances[*].id", true, "Example from spec"},
	}

	fmt.Printf("=== Testing Specification Compliance ===\n")
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			fmt.Printf("\nTesting: %s\n", tc.name)
			fmt.Printf("  Pattern: %s\n", tc.pattern)
			fmt.Printf("  Description: %s\n", tc.description)
			
			expr, err := parser.ParseExpression(tc.pattern)
			if tc.shouldParse {
				if err != nil {
					t.Errorf("Expected pattern to parse, but got error: %v", err)
					fmt.Printf("  ❌ Parse Error: %v\n", err)
				} else {
					fmt.Printf("  ✅ Parsed successfully: %s\n", expr.String())
				}
			} else {
				if err == nil {
					t.Errorf("Expected pattern to fail parsing, but it succeeded")
					fmt.Printf("  ❌ Unexpectedly parsed: %s\n", expr.String())
				} else {
					fmt.Printf("  ✅ Correctly failed to parse: %v\n", err)
				}
			}
		})
	}
}

// Test what we actually support vs what the spec says
func TestActualSupportVsSpecification(t *testing.T) {
	// Test patterns that should work according to spec
	specSupported := []string{
		"$.user.name",
		"$.users[0]", 
		"$.users[*]",
		"$.user[\"name\"]",
		"$.user[name]",
		"$.config[#*service]",
		"$.fields[~^user_.*]",
		"$.user.(name|email)",
		"$.meta{*}",
		"$.node.(child|meta.child){*}.value",
	}

	// Test patterns that our tests showed don't work
	problematic := []string{
		"$.users[*].[#*name]",     // Property wildcard syntax issue
		"$.users[*].[#admin*]",     // Property wildcard syntax issue  
		"$.users[~^admin_.*]",     // Regex with anchors
		"$.users[~.*_user$]",       // Regex with suffix anchors
		"$.users[*].*",             // Property wildcard
	}

	fmt.Printf("\n=== Specification vs Reality ===\n")
	
	fmt.Printf("\n✅ Patterns that SHOULD work (according to spec):\n")
	for _, pattern := range specSupported {
		expr, err := parser.ParseExpression(pattern)
		if err != nil {
			fmt.Printf("  ❌ %s - Error: %v\n", pattern, err)
		} else {
			fmt.Printf("  ✅ %s - Parsed: %s\n", pattern, expr.String())
		}
	}

	fmt.Printf("\n🔍 Patterns with issues:\n")
	for _, pattern := range problematic {
		expr, err := parser.ParseExpression(pattern)
		if err != nil {
			fmt.Printf("  ❌ %s - Error: %v\n", pattern, err)
		} else {
			fmt.Printf("  ⚠️  %s - Parsed but may not match correctly: %s\n", pattern, expr.String())
		}
	}
}

// Test the specific syntax issues we found
func TestSyntaxIssues(t *testing.T) {
	fmt.Printf("\n=== Detailed Syntax Analysis ===\n")
	
	// Property wildcard syntax issues
	propertyWildcardTests := []string{
		"$.users[#*name]",      // Should work - property ending with 'name'
		"$.users[#admin*]",     // Should work - property starting with 'admin'
		"$.users[#*user*]",     // Should work - property containing 'user'
		"$.users[*].[#*name]",   // Current failing syntax
		"$.users[*].[#admin*]",  // Current failing syntax
	}

	fmt.Printf("\n🔧 Property Wildcard Tests:\n")
	for _, pattern := range propertyWildcardTests {
		expr, err := parser.ParseExpression(pattern)
		if err != nil {
			fmt.Printf("  ❌ %s - Error: %v\n", pattern, err)
		} else {
			fmt.Printf("  ✅ %s - Parsed: %s\n", pattern, expr.String())
		}
	}

	// Regex pattern tests
	regexTests := []string{
		"$.users[~admin]",        // Simple contains - works
		"$.users[~^admin]",       // Starts with - should work
		"$.users[~user$]",         // Ends with - should work  
		"$.users[~^admin_.*]",     // Complex regex - should work
		"$.users[~.*_user$]",     // Complex regex - should work
	}

	fmt.Printf("\n🔧 Regex Pattern Tests:\n")
	for _, pattern := range regexTests {
		expr, err := parser.ParseExpression(pattern)
		if err != nil {
			fmt.Printf("  ❌ %s - Error: %v\n", pattern, err)
		} else {
			fmt.Printf("  ✅ %s - Parsed: %s\n", pattern, expr.String())
		}
	}
}

// Test what the specification grammar actually supports
func TestGrammarAnalysis(t *testing.T) {
	fmt.Printf("\n=== Grammar Analysis ===\n")
	
	// According to the EBNF grammar:
	// WildcardContent ::= "#" Property
	// RegexContent    ::= "~" Property
	// Property        ::= (EscapedChar | [^]\\])*
	
	// This means:
	// - Property wildcards: [#pattern] where pattern is any valid property name
	// - Regex patterns: [~pattern] where pattern is any valid regex
	
	fmt.Printf("\n📋 Key findings from grammar analysis:\n")
	fmt.Printf("  1. Property wildcards use [#pattern] syntax\n")
	fmt.Printf("  2. Regex patterns use [~pattern] syntax\n") 
	fmt.Printf("  3. Property names can contain letters, numbers, underscores\n")
	fmt.Printf("  4. Special characters like * and $ are part of the property name\n")
	fmt.Printf("  5. The parser may not be interpreting * as wildcard in property context\n")
}