package parser

import (
	"fmt"
	"regexp"
	"strconv"

	"jsonpath-sdk/spec"
)

// Parser implements a recursive descent parser for schema-path expressions.
type Parser struct {
	tokens   []spec.Token
	position int
	current  spec.Token
}

// NewParser creates a new parser with the provided tokens.
func NewParser(tokens []spec.Token) *Parser {
	p := &Parser{tokens: tokens}
	if len(tokens) > 0 {
		p.current = tokens[0]
	}
	return p
}

func (p *Parser) advance() {
	p.position++
	if p.position >= len(p.tokens) {
		p.current = spec.Token{Type: spec.TokenEOF}
		return
	}
	p.current = p.tokens[p.position]
}

func (p *Parser) expect(tokenType spec.TokenType) error {
	if p.current.Type != tokenType {
		return fmt.Errorf("expected %v at position %d", tokenType, p.current.Position)
	}
	p.advance()
	return nil
}

// Parse converts tokens into a PathExpression AST.
func (p *Parser) Parse() (*spec.PathExpression, error) {
	expr := &spec.PathExpression{}

	if err := p.expect(spec.TokenRoot); err != nil {
		return nil, fmt.Errorf("expression must start with '$': %w", err)
	}
	expr.Root = &spec.RootNode{}

	segments, err := p.parsePath()
	if err != nil {
		return nil, err
	}
	expr.Segments = segments

	if p.current.Type != spec.TokenEOF {
		return nil, fmt.Errorf("unexpected token %v at position %d", p.current.Type, p.current.Position)
	}
	return expr, nil
}

func (p *Parser) parsePath() ([]spec.ASTNode, error) {
	segments := make([]spec.ASTNode, 0)
	for {
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

func (p *Parser) parseSegment() (spec.ASTNode, error) {
	switch p.current.Type {
	case spec.TokenDot:
		p.advance()
		return p.parseSegmentItem()
	case spec.TokenBracket:
		bracket, err := p.consumeBracket()
		if err != nil {
			return nil, err
		}
		return bracket, nil
	default:
		return nil, nil
	}
}

func (p *Parser) parseSegmentItem() (spec.ASTNode, error) {
	switch p.current.Type {
	case spec.TokenIdentifier:
		node := &spec.PropertyNode{Name: p.current.Value}
		p.advance()
		if p.current.Type == spec.TokenStar {
			p.advance()
			return &spec.RepetitionNode{Sequence: []spec.ASTNode{node}}, nil
		}
		return node, nil
	case spec.TokenLParen:
		return p.parseGroupExpression()
	default:
		return nil, fmt.Errorf("expected property or group at position %d", p.current.Position)
	}
}

func (p *Parser) parseGroupExpression() (*spec.GroupNode, error) {
	if err := p.expect(spec.TokenLParen); err != nil {
		return nil, err
	}

	alternatives := make([][]spec.ASTNode, 0)
	seq, err := p.parseGroupSeq()
	if err != nil {
		return nil, err
	}
	if len(seq) == 0 {
		return nil, fmt.Errorf("group alternative cannot be empty at position %d", p.current.Position)
	}
	alternatives = append(alternatives, seq)

	for p.current.Type == spec.TokenPipe {
		p.advance()
		nextSeq, err := p.parseGroupSeq()
		if err != nil {
			return nil, err
		}
		if len(nextSeq) == 0 {
			return nil, fmt.Errorf("group alternative cannot be empty at position %d", p.current.Position)
		}
		alternatives = append(alternatives, nextSeq)
	}

	if err := p.expect(spec.TokenRParen); err != nil {
		return nil, err
	}

	group := &spec.GroupNode{Alternatives: alternatives}
	if p.current.Type == spec.TokenStar {
		group.Repetition = true
		p.advance()
	}
	return group, nil
}

func (p *Parser) parseGroupSeq() ([]spec.ASTNode, error) {
	sequence := make([]spec.ASTNode, 0)
	primary, err := p.parseGroupPrimary()
	if err != nil {
		return nil, err
	}
	sequence = append(sequence, primary...)

	for p.current.Type == spec.TokenDot {
		p.advance()
		next, err := p.parseGroupPrimary()
		if err != nil {
			return nil, err
		}
		sequence = append(sequence, next...)
	}
	return sequence, nil
}

func (p *Parser) parseGroupPrimary() ([]spec.ASTNode, error) {
	switch p.current.Type {
	case spec.TokenIdentifier:
		sequence := []spec.ASTNode{&spec.PropertyNode{Name: p.current.Value}}
		p.advance()
		suffix, err := p.parseBracketSuffix()
		if err != nil {
			return nil, err
		}
		sequence = append(sequence, suffix...)
		if p.current.Type == spec.TokenStar {
			p.advance()
			return []spec.ASTNode{&spec.RepetitionNode{Sequence: sequence}}, nil
		}
		return sequence, nil
	case spec.TokenLParen:
		group, err := p.parseGroupExpression()
		if err != nil {
			return nil, err
		}
		return []spec.ASTNode{group}, nil
	default:
		return nil, fmt.Errorf("expected identifier or group at position %d", p.current.Position)
	}
}

func (p *Parser) parseBracketSuffix() ([]spec.ASTNode, error) {
	nodes := make([]spec.ASTNode, 0)
	for p.current.Type == spec.TokenBracket {
		bracket, err := p.consumeBracket()
		if err != nil {
			return nil, err
		}
		nodes = append(nodes, bracket)
	}
	return nodes, nil
}

func (p *Parser) consumeBracket() (*spec.BracketNode, error) {
	token := p.current
	if token.Type != spec.TokenBracket {
		return nil, fmt.Errorf("internal parser error: expected bracket token, got %v", token.Type)
	}
	p.advance()
	bracket, err := buildBracketNode(token)
	if err != nil {
		return nil, err
	}
	return bracket, nil
}

func buildBracketNode(token spec.Token) (*spec.BracketNode, error) {
	if token.Quoted {
		return &spec.BracketNode{Kind: spec.BracketProperty, Value: token.Value, Quoted: true}, nil
	}

	switch {
	case token.Value == "*":
		return &spec.BracketNode{Kind: spec.BracketArrayWildcard}, nil
	case len(token.Value) > 0 && token.Value[0] == '#':
		if len(token.Value) == 1 {
			return nil, fmt.Errorf("wildcard pattern cannot be empty")
		}
		return &spec.BracketNode{Kind: spec.BracketPropertyWildcard, Value: token.Value[1:]}, nil
	case len(token.Value) > 0 && token.Value[0] == '~':
		if len(token.Value) == 1 {
			return nil, fmt.Errorf("regex pattern cannot be empty")
		}
		pattern := token.Value[1:]
		if _, err := regexp.Compile(pattern); err != nil {
			return nil, fmt.Errorf("invalid regex pattern %q: %w", pattern, err)
		}
		return &spec.BracketNode{Kind: spec.BracketRegex, Value: pattern}, nil
	case isDigits(token.Value):
		idx, err := strconv.Atoi(token.Value)
		if err != nil {
			return nil, fmt.Errorf("invalid array index %q: %w", token.Value, err)
		}
		return &spec.BracketNode{Kind: spec.BracketArrayIndex, Index: idx}, nil
	default:
		return &spec.BracketNode{Kind: spec.BracketProperty, Value: token.Value}, nil
	}
}

func isDigits(value string) bool {
	if value == "" {
		return false
	}
	for i := 0; i < len(value); i++ {
		if value[i] < '0' || value[i] > '9' {
			return false
		}
	}
	return true
}

// ParseExpression tokenizes and parses an input string.
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
