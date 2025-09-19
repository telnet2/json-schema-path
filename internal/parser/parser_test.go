package parser

import (
	"testing"
	"jsonpath-sdk/internal/spec"
)

func TestParseBasicExpressions(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "root only",
			input: "$",
			want:  "$",
		},
		{
			name:  "simple property",
			input: "$.node",
			want:  "$.node",
		},
		{
			name:  "nested property",
			input: "$.node.value",
			want:  "$.node.value",
		},
		{
			name:  "bracket notation unquoted",
			input: `$[property]`,
			want:  "$[property]",
		},
		{
			name:  "bracket notation quoted",
			input: `$["quoted-name"]`,
			want:  `$["quoted-name"]`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expr, err := ParseExpression(tt.input)
			if err != nil {
				t.Fatalf("ParseExpression() error = %v", err)
			}
			
			got := expr.String()
			if got != tt.want {
				t.Errorf("ParseExpression() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseGroupExpressions(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "simple group",
			input: "$.node.(child|value)",
		},
		{
			name:  "group with repetition",
			input: "$.node.(child|meta.child){*}",
		},
		{
			name:  "complex recursive structure", 
			input: "$.node.(child|meta.child){*}.value",
		},
		{
			name:  "bracket in group",
			input: `$.node.(child|meta["child"]){*}.value`,
		},
		{
			name:  "multiple brackets in group",
			input: `$.data.(items["key"]["subkey"]|nested.values){*}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expr, err := ParseExpression(tt.input)
			if err != nil {
				t.Fatalf("ParseExpression() error = %v for input: %s", err, tt.input)
			}
			
			// Basic validation - ensure we got a valid expression
			if expr.Root == nil {
				t.Error("Expected root node, got nil")
			}
			
			// Check that we can stringify (basic formatting test)
			result := expr.String()
			if result == "" {
				t.Error("String() returned empty result")
			}
			
			t.Logf("Input: %s -> Output: %s", tt.input, result)
		})
	}
}

func TestLexerTokenization(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []spec.TokenType
	}{
		{
			name:  "root only",
			input: "$",
			expected: []spec.TokenType{spec.TokenRoot, spec.TokenEOF},
		},
		{
			name:  "simple property",
			input: "$.node",
			expected: []spec.TokenType{spec.TokenRoot, spec.TokenDot, spec.TokenIdentifier, spec.TokenEOF},
		},
		{
			name:  "group expression",
			input: "$.(child|value)",
			expected: []spec.TokenType{spec.TokenRoot, spec.TokenDot, spec.TokenLParen, spec.TokenIdentifier, 
				spec.TokenPipe, spec.TokenIdentifier, spec.TokenRParen, spec.TokenEOF},
		},
		{
			name:  "repetition",
			input: "$.(child){*}",
			expected: []spec.TokenType{spec.TokenRoot, spec.TokenDot, spec.TokenLParen, spec.TokenIdentifier,
				spec.TokenRParen, spec.TokenStar, spec.TokenEOF},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lexer := NewLexer(tt.input)
			tokens, err := lexer.Tokenize()
			if err != nil {
				t.Fatalf("Tokenize() error = %v", err)
			}
			
			if len(tokens) != len(tt.expected) {
				t.Fatalf("Expected %d tokens, got %d", len(tt.expected), len(tokens))
			}
			
			for i, expectedType := range tt.expected {
				if tokens[i].Type != expectedType {
					t.Errorf("Token %d: expected type %v, got %v", i, expectedType, tokens[i].Type)
				}
			}
		})
	}
}

func TestErrorHandling(t *testing.T) {
	errorCases := []struct {
		name  string
		input string
	}{
		{
			name:  "no root",
			input: "node.value",
		},
		{
			name:  "unterminated quote",
			input: `$["unterminated`,
		},
		{
			name:  "unterminated group",
			input: "$.node.(child|value",
		},
		{
			name:  "invalid repetition",
			input: "$.node{*}",
		},
		{
			name:  "empty group",
			input: "$.node.()",
		},
	}

	for _, tt := range errorCases {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseExpression(tt.input)
			if err == nil {
				t.Errorf("Expected error for input: %s", tt.input)
			}
			t.Logf("Got expected error: %v", err)
		})
	}
}