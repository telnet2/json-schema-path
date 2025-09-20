package parser

import (
	"fmt"
	"unicode"

	"github.com/telnet2/json-schema-path/spec"
)

// Lexer tokenizes schema-path expressions.
type Lexer struct {
	input    string
	position int
	current  byte
	tokens   []spec.Token
}

// NewLexer creates a new lexer for the given input.
func NewLexer(input string) *Lexer {
	l := &Lexer{
		input:  input,
		tokens: make([]spec.Token, 0),
	}
	if len(input) > 0 {
		l.current = input[0]
	}
	return l
}

func (l *Lexer) advance() {
	l.position++
	if l.position >= len(l.input) {
		l.current = 0
		return
	}
	l.current = l.input[l.position]
}

func (l *Lexer) skipWhitespace() {
	for l.current != 0 && unicode.IsSpace(rune(l.current)) {
		l.advance()
	}
}

func (l *Lexer) addToken(tokenType spec.TokenType, value string, quoted bool) {
	l.tokens = append(l.tokens, spec.Token{
		Type:     tokenType,
		Value:    value,
		Position: l.position,
		Quoted:   quoted,
	})
}

func (l *Lexer) readIdentifier() string {
	start := l.position
	if !(isLetter(l.current) || l.current == '_') {
		return ""
	}
	l.advance()
	for l.current != 0 && (isLetter(l.current) || isDigit(l.current) || l.current == '_') {
		l.advance()
	}
	return l.input[start:l.position]
}

func (l *Lexer) readBracketContent() (string, bool, error) {
	if l.current == '"' {
		l.advance()
		buf := make([]byte, 0)
		for {
			if l.current == 0 {
				return "", false, fmt.Errorf("unterminated quoted string in bracket notation")
			}
			if l.current == '"' {
				l.advance()
				break
			}
			if l.current == '\\' {
				l.advance()
				if l.current == 0 {
					return "", false, fmt.Errorf("unterminated escape sequence in bracket notation")
				}
			}
			buf = append(buf, l.current)
			l.advance()
		}
		return string(buf), true, nil
	}

	buf := make([]byte, 0)
	for l.current != 0 && l.current != ']' {
		if l.current == '\\' {
			l.advance()
			if l.current == 0 {
				return "", false, fmt.Errorf("unterminated escape sequence in bracket notation")
			}
		}
		buf = append(buf, l.current)
		l.advance()
	}
	return string(buf), false, nil
}

// Tokenize converts the input string into tokens understood by the parser.
func (l *Lexer) Tokenize() ([]spec.Token, error) {
	for l.current != 0 {
		l.skipWhitespace()
		if l.current == 0 {
			break
		}

		switch l.current {
		case '$':
			l.addToken(spec.TokenRoot, "$", false)
			l.advance()
		case '.':
			l.addToken(spec.TokenDot, ".", false)
			l.advance()
		case '(':
			l.addToken(spec.TokenLParen, "(", false)
			l.advance()
		case ')':
			l.addToken(spec.TokenRParen, ")", false)
			l.advance()
		case '|':
			l.addToken(spec.TokenPipe, "|", false)
			l.advance()
		case '[':
			l.advance()
			content, quoted, err := l.readBracketContent()
			if err != nil {
				return nil, err
			}
			if l.current != ']' {
				return nil, fmt.Errorf("expected closing ']' at position %d", l.position)
			}
			l.addToken(spec.TokenBracket, content, quoted)
			l.advance()
		case '{':
			startPos := l.position
			l.advance()
			if l.current != '*' {
				return nil, fmt.Errorf("unexpected character '{' at position %d", startPos)
			}
			l.advance()
			if l.current != '}' {
				return nil, fmt.Errorf("expected '}' to close repetition at position %d", l.position)
			}
			l.advance()
			l.addToken(spec.TokenStar, "{*}", false)
		default:
			if ident := l.readIdentifier(); ident != "" {
				l.addToken(spec.TokenIdentifier, ident, false)
			} else {
				return nil, fmt.Errorf("unexpected character '%c' at position %d", l.current, l.position)
			}
		}
	}

	l.addToken(spec.TokenEOF, "", false)
	return l.tokens, nil
}

func isLetter(ch byte) bool {
	return (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z')
}

func isDigit(ch byte) bool {
	return ch >= '0' && ch <= '9'
}
