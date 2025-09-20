package main

import (
	"fmt"
	"github.com/telnet2/json-schema-path/parser"
	"github.com/telnet2/json-schema-path/tree"
	"github.com/telnet2/json-schema-path/spec"
)

func main() {
	// Test the pattern matching directly
	expr, _ := parser.ParseExpression("$.store.products[*].name")
	
	patternTree := tree.NewPatternTree()
	patternTree.AddPattern(expr)
	
	// Test segments that should match
	testSegments := []spec.PathSegment{
		{Type: spec.SegmentProperty, Key: "store"},
		{Type: spec.SegmentProperty, Key: "products"},
		{Type: spec.SegmentArrayIndex, Index: 0}, // This should match [*]
		{Type: spec.SegmentProperty, Key: "name"},
	}
	
	fmt.Printf("Testing segments: %v\n", testSegments)
	result := patternTree.MatchSegments(testSegments)
	fmt.Printf("Match result: %v\n", result)
	
	// Let's also test a simpler case
	simpleExpr, _ := parser.ParseExpression("$.store.products[*]")
	simpleTree := tree.NewPatternTree()
	simpleTree.AddPattern(simpleExpr)
	
	simpleSegments := []spec.PathSegment{
		{Type: spec.SegmentProperty, Key: "store"},
		{Type: spec.SegmentProperty, Key: "products"},
		{Type: spec.SegmentArrayIndex, Index: 0},
	}
	
	fmt.Printf("\nSimple test segments: %v\n", simpleSegments)
	simpleResult := simpleTree.MatchSegments(simpleSegments)
	fmt.Printf("Simple match result: %v\n", simpleResult)
}