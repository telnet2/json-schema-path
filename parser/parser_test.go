package parser

import (
	"testing"

	"github.com/telnet2/json-schema-path/spec"
)

func TestParseExpressionBasic(t *testing.T) {
	expr, err := ParseExpression("$.user.name")
	if err != nil {
		t.Fatalf("ParseExpression error: %v", err)
	}
	if len(expr.Segments) != 2 {
		t.Fatalf("expected 2 segments, got %d", len(expr.Segments))
	}
	first, ok := expr.Segments[0].(*spec.PropertyNode)
	if !ok || first.Name != "user" {
		t.Fatalf("expected first segment to be property 'user', got %#v", expr.Segments[0])
	}
	second, ok := expr.Segments[1].(*spec.PropertyNode)
	if !ok || second.Name != "name" {
		t.Fatalf("expected second segment to be property 'name', got %#v", expr.Segments[1])
	}
	if got := expr.String(); got != "$.user.name" {
		t.Fatalf("unexpected string form: %s", got)
	}
}

func TestParseExpressionBracketSelectors(t *testing.T) {
	expr, err := ParseExpression(`$.data["prop"][#user*][~^id\\d+$][0][*]`)
	if err != nil {
		t.Fatalf("ParseExpression error: %v", err)
	}
	
	// Debug: Print all segments
	t.Logf("Segments for bracket selectors:")
	for i, seg := range expr.Segments {
		t.Logf("  [%d]: %#v", i, seg)
	}
	
	if len(expr.Segments) != 6 {
		t.Fatalf("expected 6 segments, got %d", len(expr.Segments))
	}
	expectKinds := []spec.BracketKind{
		spec.BracketProperty,
		spec.BracketPropertyWildcard,
		spec.BracketRegex,
		spec.BracketArrayIndex,
		spec.BracketArrayWildcard,
	}
	for i, kind := range expectKinds {
		bracket, ok := expr.Segments[i+1].(*spec.BracketNode)
		if !ok {
			t.Fatalf("segment %d expected bracket node, got %#v", i+1, expr.Segments[i+1])
		}
		if bracket.Kind != kind {
			t.Fatalf("segment %d expected kind %v, got %v", i+1, kind, bracket.Kind)
		}
	}
	if value := expr.Segments[2].(*spec.BracketNode).Value; value != "user*" {
		t.Fatalf("unexpected wildcard value: %s", value)
	}
	if value := expr.Segments[3].(*spec.BracketNode).Value; value != "^id\\d+$" {
		t.Fatalf("unexpected regex value: %s", value)
	}
	if idx := expr.Segments[4].(*spec.BracketNode).Index; idx != 0 {
		t.Fatalf("expected array index 0, got %d", idx)
	}
}

func TestParseExpressionGroupAndRepetition(t *testing.T) {
	expr, err := ParseExpression("$.node.(child|meta.child){*}.value")
	if err != nil {
		t.Fatalf("ParseExpression error: %v", err)
	}
	if len(expr.Segments) != 3 {
		t.Fatalf("expected 3 segments, got %d", len(expr.Segments))
	}
	group, ok := expr.Segments[1].(*spec.GroupNode)
	if !ok {
		t.Fatalf("expected second segment to be group, got %#v", expr.Segments[1])
	}
	if !group.Repetition {
		t.Fatalf("expected group to have repetition")
	}
	if len(group.Alternatives) != 2 {
		t.Fatalf("expected 2 alternatives, got %d", len(group.Alternatives))
	}
	if len(group.Alternatives[0]) != 1 {
		t.Fatalf("expected first alternative length 1, got %d", len(group.Alternatives[0]))
	}
	if prop, ok := group.Alternatives[0][0].(*spec.PropertyNode); !ok || prop.Name != "child" {
		t.Fatalf("unexpected first alternative: %#v", group.Alternatives[0][0])
	}
	if len(group.Alternatives[1]) != 2 {
		t.Fatalf("expected second alternative length 2, got %d", len(group.Alternatives[1]))
	}
	if got := expr.String(); got != "$.node.(child|meta.child){*}.value" {
		t.Fatalf("unexpected string form: %s", got)
	}
}

func TestParseExpressionPropertyRepetition(t *testing.T) {
	expr, err := ParseExpression("$.meta{*}.child")
	if err != nil {
		t.Fatalf("ParseExpression error: %v", err)
	}
	if len(expr.Segments) != 2 {
		t.Fatalf("expected 2 segments, got %d", len(expr.Segments))
	}
	repeat, ok := expr.Segments[0].(*spec.RepetitionNode)
	if !ok {
		t.Fatalf("expected first segment to be repetition node, got %#v", expr.Segments[0])
	}
	if len(repeat.Sequence) != 1 {
		t.Fatalf("expected repetition sequence length 1, got %d", len(repeat.Sequence))
	}
	if prop, ok := repeat.Sequence[0].(*spec.PropertyNode); !ok || prop.Name != "meta" {
		t.Fatalf("unexpected repetition inner node: %#v", repeat.Sequence[0])
	}
	if got := expr.String(); got != "$.meta{*}.child" {
		t.Fatalf("unexpected string form: %s", got)
	}
}

func TestLexerTokenization(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []spec.TokenType
	}{
		{
			name:     "root only",
			input:    "$",
			expected: []spec.TokenType{spec.TokenRoot, spec.TokenEOF},
		},
		{
			name:     "bracket",
			input:    "$[value]",
			expected: []spec.TokenType{spec.TokenRoot, spec.TokenBracket, spec.TokenEOF},
		},
		{
			name:  "group",
			input: "$.node.(child|value)",
			expected: []spec.TokenType{
				spec.TokenRoot, spec.TokenDot, spec.TokenIdentifier, spec.TokenDot,
				spec.TokenLParen, spec.TokenIdentifier, spec.TokenPipe, spec.TokenIdentifier,
				spec.TokenRParen, spec.TokenEOF,
			},
		},
		{
			name:  "repetition",
			input: "$.(child){*}",
			expected: []spec.TokenType{
				spec.TokenRoot, spec.TokenDot, spec.TokenLParen, spec.TokenIdentifier,
				spec.TokenRParen, spec.TokenStar, spec.TokenEOF,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lexer := NewLexer(tt.input)
			tokens, err := lexer.Tokenize()
			if err != nil {
				t.Fatalf("Tokenize error: %v", err)
			}
			if len(tokens) != len(tt.expected) {
				t.Fatalf("expected %d tokens, got %d", len(tt.expected), len(tokens))
			}
			for i, tok := range tokens {
				if tok.Type != tt.expected[i] {
					t.Fatalf("token %d expected %v, got %v", i, tt.expected[i], tok.Type)
				}
			}
		})
	}
}

func TestParseExpressionProblematicPattern(t *testing.T) {
	// Test the expression that's causing parsing errors
	expr, err := ParseExpression("$.children[*]{*}.name")
	if err != nil {
		t.Fatalf("ParseExpression error for $.children[*]{*}.name: %v", err)
	}
	if len(expr.Segments) != 2 {
		t.Fatalf("expected 2 segments, got %d", len(expr.Segments))
	}
	
	// First segment should be children with bracket wildcard and repetition
	first, ok := expr.Segments[0].(*spec.RepetitionNode)
	if !ok {
		t.Fatalf("expected first segment to be repetition node, got %#v", expr.Segments[0])
	}
	if len(first.Sequence) != 2 {
		t.Fatalf("expected repetition sequence length 2, got %d", len(first.Sequence))
	}
	// Check the property node
	prop, ok := first.Sequence[0].(*spec.PropertyNode)
	if !ok || prop.Name != "children" {
		t.Fatalf("expected first sequence item to be property 'children', got %#v", first.Sequence[0])
	}
	// Check the bracket wildcard node
	bracket, ok := first.Sequence[1].(*spec.BracketNode)
	if !ok || bracket.Kind != spec.BracketArrayWildcard {
		t.Fatalf("expected second sequence item to be bracket wildcard, got %#v", first.Sequence[1])
	}
	// Second segment should be name property
	second, ok := expr.Segments[1].(*spec.PropertyNode)
	if !ok || second.Name != "name" {
		t.Fatalf("expected second segment to be property 'name', got %#v", expr.Segments[1])
	}
}

func TestParseExpressionPropertyWithBracket(t *testing.T) {
	// Test property with bracket but no repetition
	expr, err := ParseExpression("$.users[*]")
	if err != nil {
		t.Fatalf("ParseExpression error for $.users[*]: %v", err)
	}
	if len(expr.Segments) != 2 {
		t.Fatalf("expected 2 segments, got %d", len(expr.Segments))
	}
	
	// First segment should be property
	first, ok := expr.Segments[0].(*spec.PropertyNode)
	if !ok || first.Name != "users" {
		t.Fatalf("expected first segment to be property 'users', got %#v", expr.Segments[0])
	}
	
	// Second segment should be bracket wildcard
	second, ok := expr.Segments[1].(*spec.BracketNode)
	if !ok || second.Kind != spec.BracketArrayWildcard {
		t.Fatalf("expected second segment to be bracket wildcard, got %#v", expr.Segments[1])
	}
}

func TestParseExpressionInvalidPatterns(t *testing.T) {
	cases := []string{
		"$[~]",
		"$[#]",
		"$.(child|){*}",
		"$.node.(|child)",
	}
	for _, input := range cases {
		if _, err := ParseExpression(input); err == nil {
			t.Fatalf("expected error for %s", input)
		}
	}
}
