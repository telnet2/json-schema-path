package main

import (
	"sort"
	"testing"

	jsonpkg "github.com/telnet2/json-schema-path/json"
	"github.com/telnet2/json-schema-path/parser"
	"github.com/telnet2/json-schema-path/tree"
)

func collectMatches(t *testing.T, expression string, jsonData string) []string {
	t.Helper()
	expr, err := parser.ParseExpression(expression)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	patternTree := tree.NewPatternTree()
	if err := patternTree.AddPattern(expr); err != nil {
		t.Fatalf("add pattern error: %v", err)
	}
	processor := jsonpkg.NewPathExtractor()
	paths, err := processor.ExtractPaths(jsonData)
	if err != nil {
		t.Fatalf("extract paths error: %v", err)
	}
	matches := make([]string, 0)
	for _, path := range paths {
		segments := processor.ConvertPathToSegments(path)
		if patternTree.MatchSegments(segments) {
			matches = append(matches, path)
		}
	}
	sort.Strings(matches)
	return matches
}

func TestSchemaRecursiveMatching(t *testing.T) {
	jsonData := `{"node":{"value":"root","child":{"value":"direct"},"meta":{"child":{"value":"nested"}}}}`
	matches := collectMatches(t, "$.node.(child|meta.child){*}.value", jsonData)
	expected := []string{
		"$.node.value",
		"$.node.child.value",
		"$.node.meta.child.value",
	}
	sort.Strings(expected)
	if len(matches) != len(expected) {
		t.Fatalf("expected %d matches, got %d: %v", len(expected), len(matches), matches)
	}
	for i, exp := range expected {
		if matches[i] != exp {
			t.Fatalf("expected match %s, got %s", exp, matches[i])
		}
	}
}

func TestSchemaWildcardRegexMatching(t *testing.T) {
	jsonData := `{"config":{"cache-service":{"instances":[{"id":"cache-1"}]},"user-service":{"instances":[{"id":"id42"}]}}}`
	matches := collectMatches(t, "$.config[#*service].instances[*].id", jsonData)
	expected := []string{
		"$.config.cache-service.instances[0].id",
		"$.config.user-service.instances[0].id",
	}
	if len(matches) != len(expected) {
		t.Fatalf("expected %d matches, got %d: %v", len(expected), len(matches), matches)
	}
	for i, exp := range expected {
		if matches[i] != exp {
			t.Fatalf("expected match %s, got %s", exp, matches[i])
		}
	}
}

func TestSchemaPropertyRepetition(t *testing.T) {
	jsonData := `{"meta":{"meta":{"child":{"value":1}},"child":{"value":2}}}`
	matches := collectMatches(t, "$.meta{*}.child.value", jsonData)
	expected := []string{
		"$.meta.child.value",
		"$.meta.meta.child.value",
	}
	if len(matches) != len(expected) {
		t.Fatalf("expected %d matches, got %d: %v", len(expected), len(matches), matches)
	}
	for i, exp := range expected {
		if matches[i] != exp {
			t.Fatalf("expected match %s, got %s", exp, matches[i])
		}
	}
}
