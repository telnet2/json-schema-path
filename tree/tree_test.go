package tree

import (
	"testing"

	jsonpkg "jsonpath-sdk/json"
	"jsonpath-sdk/parser"
	"jsonpath-sdk/spec"
)

func TestPatternTreeLiteralMatching(t *testing.T) {
	expr, err := parser.ParseExpression("$.user.name")
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	tree := NewPatternTree()
	if err := tree.AddPattern(expr); err != nil {
		t.Fatalf("add pattern error: %v", err)
	}
	if !tree.MatchSegments([]spec.PathSegment{spec.NewPropertySegment("user"), spec.NewPropertySegment("name")}) {
		t.Fatalf("expected literal path to match")
	}
	if tree.MatchSegments([]spec.PathSegment{spec.NewPropertySegment("user"), spec.NewPropertySegment("age")}) {
		t.Fatalf("unexpected match for user.age")
	}
}

func TestPatternTreeBracketMatching(t *testing.T) {
	expr, err := parser.ParseExpression(`$.data["prop"].value`)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	tree := NewPatternTree()
	if err := tree.AddPattern(expr); err != nil {
		t.Fatalf("add pattern error: %v", err)
	}
	path := []spec.PathSegment{spec.NewPropertySegment("data"), spec.NewPropertySegment("prop"), spec.NewPropertySegment("value")}
	if !tree.MatchSegments(path) {
		t.Fatalf("expected bracket property to match")
	}
}

func TestPatternTreeWildcardAndRegex(t *testing.T) {
	expr, err := parser.ParseExpression(`$.data[#user*][~^id\\d+$]`)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	tree := NewPatternTree()
	if err := tree.AddPattern(expr); err != nil {
		t.Fatalf("add pattern error: %v", err)
	}
	if !tree.MatchSegments([]spec.PathSegment{spec.NewPropertySegment("data"), spec.NewPropertySegment("userA"), spec.NewPropertySegment("id42")}) {
		t.Fatalf("expected wildcard/regex path to match")
	}
	if tree.MatchSegments([]spec.PathSegment{spec.NewPropertySegment("data"), spec.NewPropertySegment("admin"), spec.NewPropertySegment("id42")}) {
		t.Fatalf("unexpected match for admin path")
	}
}

func TestPatternTreeArrayMatching(t *testing.T) {
	expr, err := parser.ParseExpression("$.items[*].value")
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	tree := NewPatternTree()
	if err := tree.AddPattern(expr); err != nil {
		t.Fatalf("add pattern error: %v", err)
	}
	if !tree.MatchSegments([]spec.PathSegment{spec.NewPropertySegment("items"), spec.NewArrayIndexSegment(3), spec.NewPropertySegment("value")}) {
		t.Fatalf("expected wildcard array match")
	}
	if tree.MatchSegments([]spec.PathSegment{spec.NewPropertySegment("items"), spec.NewPropertySegment("value")}) {
		t.Fatalf("unexpected match for missing index")
	}
}

func TestPatternTreeGroupRepetition(t *testing.T) {
	expr, err := parser.ParseExpression("$.node.(child|meta.child){*}.value")
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	tree := NewPatternTree()
	if err := tree.AddPattern(expr); err != nil {
		t.Fatalf("add pattern error: %v", err)
	}
	// Zero iterations
	if !tree.MatchSegments([]spec.PathSegment{spec.NewPropertySegment("node"), spec.NewPropertySegment("value")}) {
		t.Fatalf("expected zero-iteration match")
	}
	// Single child
	if !tree.MatchSegments([]spec.PathSegment{spec.NewPropertySegment("node"), spec.NewPropertySegment("child"), spec.NewPropertySegment("value")}) {
		t.Fatalf("expected single child match")
	}
	// Mixed alternatives
	path := []spec.PathSegment{
		spec.NewPropertySegment("node"),
		spec.NewPropertySegment("meta"),
		spec.NewPropertySegment("child"),
		spec.NewPropertySegment("child"),
		spec.NewPropertySegment("value"),
	}
	if !tree.MatchSegments(path) {
		t.Fatalf("expected mixed alternative match")
	}
}

func TestPatternTreePropertyRepetition(t *testing.T) {
	expr, err := parser.ParseExpression("$.meta{*}.child")
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	tree := NewPatternTree()
	if err := tree.AddPattern(expr); err != nil {
		t.Fatalf("add pattern error: %v", err)
	}
	if !tree.MatchSegments([]spec.PathSegment{spec.NewPropertySegment("child")}) {
		t.Fatalf("expected zero meta match")
	}
	if !tree.MatchSegments([]spec.PathSegment{spec.NewPropertySegment("meta"), spec.NewPropertySegment("child")}) {
		t.Fatalf("expected single meta match")
	}
	if !tree.MatchSegments([]spec.PathSegment{spec.NewPropertySegment("meta"), spec.NewPropertySegment("meta"), spec.NewPropertySegment("child")}) {
		t.Fatalf("expected repeated meta match")
	}
}

func TestPatternTreeMultiplePatterns(t *testing.T) {
	tree := NewPatternTree()
	patterns := []string{"$.user.name", "$.user.email", "$.orders[*].id"}
	for _, p := range patterns {
		expr, err := parser.ParseExpression(p)
		if err != nil {
			t.Fatalf("parse error: %v", err)
		}
		if err := tree.AddPattern(expr); err != nil {
			t.Fatalf("add pattern error: %v", err)
		}
	}
	if !tree.MatchSegments([]spec.PathSegment{spec.NewPropertySegment("user"), spec.NewPropertySegment("email")}) {
		t.Fatalf("expected user.email to match")
	}
	if !tree.MatchSegments([]spec.PathSegment{spec.NewPropertySegment("orders"), spec.NewArrayIndexSegment(2), spec.NewPropertySegment("id")}) {
		t.Fatalf("expected orders[*].id to match")
	}
}

func TestPatternTreeJSONIntegration(t *testing.T) {
	expr, err := parser.ParseExpression("$.node.(child|meta.child){*}.value")
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	tree := NewPatternTree()
	if err := tree.AddPattern(expr); err != nil {
		t.Fatalf("add pattern error: %v", err)
	}
	jsonData := `{"node":{"child":{"value":1},"meta":{"child":{"value":2}}}}`
	processor := jsonpkg.NewPathExtractor()
	paths, err := processor.ExtractPaths(jsonData)
	if err != nil {
		t.Fatalf("extract paths error: %v", err)
	}
	found := false
	for _, p := range paths {
		segments := processor.ConvertPathToSegments(p)
		if tree.MatchSegments(segments) {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected at least one matching path in JSON")
	}
}
