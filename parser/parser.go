// Package parser provides lexical analysis and parsing for schema-path expressions.
//
// The parser package implements a two-stage parsing process:
//   1. Lexical analysis: Tokenizes input into ROOT, DOT, IDENTIFIER, brackets, etc.
//   2. Syntactic analysis: Builds an Abstract Syntax Tree (AST) using recursive descent
//
// The parser supports the full schema-path grammar including:
//   - Root anchor ($)
//   - Property access (.name)
//   - Bracket notation (["key"] and [key])
//   - Group alternatives (|)
//   - Zero-or-more repetition ({*})
//
// Example usage:
//
//	expr, err := parser.ParseExpression("$.node.(child|meta.child){*}.value")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Println(expr.String()) // Output: $.node(.child|.meta.child){*}.value
//
// The parser handles escape sequences in both quoted and unquoted bracket notation,
// allowing special characters in property names.
package parser

import (
        "fmt"

        "jsonpath-sdk/spec"
)

// Parser implements a recursive descent parser for JSON path expressions
type Parser struct {
        tokens   []spec.Token
        position int
        current  spec.Token
}

// NewParser creates a new parser with the given tokens
func NewParser(tokens []spec.Token) *Parser {
        p := &Parser{
                tokens:   tokens,
                position: 0,
        }
        if len(tokens) > 0 {
                p.current = tokens[0]
        }
        return p
}

// advance moves to the next token
func (p *Parser) advance() {
        p.position++
        if p.position >= len(p.tokens) {
                p.current = spec.Token{Type: spec.TokenEOF}
        } else {
                p.current = p.tokens[p.position]
        }
}

// peek returns the next token without advancing
func (p *Parser) peek() spec.Token {
        nextPos := p.position + 1
        if nextPos >= len(p.tokens) {
                return spec.Token{Type: spec.TokenEOF}
        }
        return p.tokens[nextPos]
}

// expect checks if current token matches expected type and advances
func (p *Parser) expect(tokenType spec.TokenType) error {
        if p.current.Type != tokenType {
                return fmt.Errorf("expected %v, got %v at position %d", tokenType, p.current.Type, p.current.Position)
        }
        p.advance()
        return nil
}

// Parse parses the tokens into an AST
// Expression ::= Root Path?
func (p *Parser) Parse() (*spec.PathExpression, error) {
        expr := &spec.PathExpression{}
        
        // Parse root
        root, err := p.parseRoot()
        if err != nil {
                return nil, err
        }
        expr.Root = root
        
        // Parse optional path
        if p.current.Type != spec.TokenEOF {
                segments, err := p.parsePath()
                if err != nil {
                        return nil, err
                }
                expr.Segments = segments
        }
        
        // Ensure we've consumed all tokens
        if p.current.Type != spec.TokenEOF {
                return nil, fmt.Errorf("unexpected token %v at position %d", p.current.Type, p.current.Position)
        }
        
        return expr, nil
}

// parseRoot parses the root token
// Root ::= "$"
func (p *Parser) parseRoot() (*spec.RootNode, error) {
        if err := p.expect(spec.TokenRoot); err != nil {
                return nil, err
        }
        return &spec.RootNode{}, nil
}

// parsePath parses a path sequence
// Path ::= Segment*
func (p *Parser) parsePath() ([]spec.ASTNode, error) {
        segments := make([]spec.ASTNode, 0)
        
        for p.current.Type != spec.TokenEOF {
                segment, err := p.parseSegment()
                if err != nil {
                        return nil, err
                }
                if segment == nil {
                        break
                }
                segments = append(segments, segment)
        }
        
        return segments, nil
}

// parseSegment parses a single segment
// Segment ::= "." SegmentItem | BracketNotation
func (p *Parser) parseSegment() (spec.ASTNode, error) {
        switch p.current.Type {
        case spec.TokenDot:
                p.advance() // consume "."
                return p.parseSegmentItem()
                
        case spec.TokenString:
                // This represents bracket notation content from lexer
                content := p.current.Value
                p.advance()
                
                // Check if quoted (starts and ends with quotes)
                quoted := false
                if len(content) >= 2 && content[0] == '"' && content[len(content)-1] == '"' {
                        quoted = true
                        content = content[1 : len(content)-1] // remove quotes
                }
                
                return &spec.BracketNode{Content: content, Quoted: quoted}, nil
                
        default:
                return nil, nil // no more segments
        }
}

// parseSegmentItem parses a segment item
// SegmentItem ::= Identifier | GroupExpression
func (p *Parser) parseSegmentItem() (spec.ASTNode, error) {
        switch p.current.Type {
        case spec.TokenIdentifier:
                name := p.current.Value
                p.advance()
                return &spec.PropertyNode{Name: name}, nil
                
        case spec.TokenLParen:
                return p.parseGroupExpression()
                
        default:
                return nil, fmt.Errorf("expected identifier or group expression, got %v at position %d", p.current.Type, p.current.Position)
        }
}

// parseGroupExpression parses a group expression with optional repetition
// GroupExpression ::= "(" GroupSeq ("|" GroupSeq)* ")" Repetition?
func (p *Parser) parseGroupExpression() (*spec.GroupNode, error) {
        if err := p.expect(spec.TokenLParen); err != nil {
                return nil, err
        }
        
        group := &spec.GroupNode{
                Alternatives: make([][]spec.ASTNode, 0),
        }
        
        // Parse first sequence
        firstSeq, err := p.parseGroupSeq()
        if err != nil {
                return nil, err
        }
        group.Alternatives = append(group.Alternatives, firstSeq)
        
        // Parse additional alternatives
        for p.current.Type == spec.TokenPipe {
                p.advance() // consume "|"
                seq, err := p.parseGroupSeq()
                if err != nil {
                        return nil, err
                }
                group.Alternatives = append(group.Alternatives, seq)
        }
        
        if err := p.expect(spec.TokenRParen); err != nil {
                return nil, err
        }
        
        // Check for optional repetition
        if p.current.Type == spec.TokenStar {
                group.Repetition = true
                p.advance()
        }
        
        return group, nil
}

// parseGroupSeq parses a group sequence
// GroupSeq ::= GroupPrimary ("." GroupPrimary)*
func (p *Parser) parseGroupSeq() ([]spec.ASTNode, error) {
        sequence := make([]spec.ASTNode, 0)
        
        // Parse first primary
        primary, err := p.parseGroupPrimary()
        if err != nil {
                return nil, err
        }
        sequence = append(sequence, primary...)
        
        // Parse additional primaries separated by dots
        for p.current.Type == spec.TokenDot {
                p.advance() // consume "."
                primary, err := p.parseGroupPrimary()
                if err != nil {
                        return nil, err
                }
                sequence = append(sequence, primary...)
        }
        
        return sequence, nil
}

// parseGroupPrimary parses a group primary
// GroupPrimary ::= Identifier BracketSuffix? | GroupExpression
func (p *Parser) parseGroupPrimary() ([]spec.ASTNode, error) {
        switch p.current.Type {
        case spec.TokenIdentifier:
                nodes := make([]spec.ASTNode, 0)
                
                // Add identifier
                name := p.current.Value
                p.advance()
                nodes = append(nodes, &spec.PropertyNode{Name: name})
                
                // Parse optional bracket suffix
                brackets, err := p.parseBracketSuffix()
                if err != nil {
                        return nil, err
                }
                nodes = append(nodes, brackets...)
                
                return nodes, nil
                
        case spec.TokenLParen:
                group, err := p.parseGroupExpression()
                if err != nil {
                        return nil, err
                }
                return []spec.ASTNode{group}, nil
                
        default:
                return nil, fmt.Errorf("expected identifier or group expression, got %v at position %d", p.current.Type, p.current.Position)
        }
}

// parseBracketSuffix parses optional bracket notation after identifier
// BracketSuffix ::= (BracketNotation)+
func (p *Parser) parseBracketSuffix() ([]spec.ASTNode, error) {
        brackets := make([]spec.ASTNode, 0)
        
        for p.current.Type == spec.TokenString {
                content := p.current.Value
                p.advance()
                
                // Check if quoted
                quoted := false
                if len(content) >= 2 && content[0] == '"' && content[len(content)-1] == '"' {
                        quoted = true
                        content = content[1 : len(content)-1] // remove quotes
                }
                
                brackets = append(brackets, &spec.BracketNode{Content: content, Quoted: quoted})
        }
        
        return brackets, nil
}

// ParseExpression is a convenience function that combines lexing and parsing
func ParseExpression(input string) (*spec.PathExpression, error) {
        lexer := NewLexer(input)
        tokens, err := lexer.Tokenize()
        if err != nil {
                return nil, fmt.Errorf("lexer error: %w", err)
        }
        
        parser := NewParser(tokens)
        expr, err := parser.Parse()
        if err != nil {
                return nil, fmt.Errorf("parser error: %w", err)
        }
        
        return expr, nil
}