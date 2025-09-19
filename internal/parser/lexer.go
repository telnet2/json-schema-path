package parser

import (
        "fmt"
        "unicode"

        "jsonpath-sdk/internal/spec"
)

// Lexer tokenizes JSON path expressions
type Lexer struct {
        input    string
        position int
        current  byte
        tokens   []spec.Token
}

// NewLexer creates a new lexer for the given input
func NewLexer(input string) *Lexer {
        l := &Lexer{
                input:   input,
                tokens:  make([]spec.Token, 0),
        }
        if len(input) > 0 {
                l.current = input[0]
        }
        return l
}

// advance moves to the next character
func (l *Lexer) advance() {
        l.position++
        if l.position >= len(l.input) {
                l.current = 0 // EOF
        } else {
                l.current = l.input[l.position]
        }
}

// peek returns the next character without advancing
func (l *Lexer) peek() byte {
        nextPos := l.position + 1
        if nextPos >= len(l.input) {
                return 0
        }
        return l.input[nextPos]
}

// skipWhitespace skips whitespace characters
func (l *Lexer) skipWhitespace() {
        for l.current != 0 && unicode.IsSpace(rune(l.current)) {
                l.advance()
        }
}

// readIdentifier reads an identifier following [a-zA-Z_][a-zA-Z0-9_]*
func (l *Lexer) readIdentifier() string {
        start := l.position
        
        // First character must be letter or underscore
        if !isLetter(l.current) && l.current != '_' {
                return ""
        }
        
        l.advance()
        
        // Subsequent characters can be letters, digits, or underscore
        for l.current != 0 && (isLetterOrDigit(l.current) || l.current == '_') {
                l.advance()
        }
        
        return l.input[start:l.position]
}

// readBracketContent reads content inside brackets until closing ]
func (l *Lexer) readBracketContent() (string, bool, error) {
        start := l.position
        quoted := false
        
        // Check if content starts with quote
        if l.current == '"' {
                quoted = true
                l.advance() // skip opening quote
                start = l.position // start after the quote
                
                // Read until closing quote
                for l.current != 0 && l.current != '"' {
                        if l.current == '\\' {
                                l.advance() // skip escape char
                                if l.current != 0 {
                                        l.advance() // skip escaped char
                                }
                        } else {
                                l.advance()
                        }
                }
                
                if l.current != '"' {
                        return "", false, fmt.Errorf("unterminated quoted string in bracket notation")
                }
                
                content := l.input[start:l.position]
                l.advance() // skip closing quote
                return content, quoted, nil
        }
        
        // Read unquoted content until ]
        for l.current != 0 && l.current != ']' {
                if l.current == '\\' {
                        l.advance() // skip escape char
                        if l.current != 0 {
                                l.advance() // skip escaped char
                        }
                } else {
                        l.advance()
                }
        }
        
        content := l.input[start:l.position]
        return content, quoted, nil
}

// addToken adds a token to the tokens slice
func (l *Lexer) addToken(tokenType spec.TokenType, value string) {
        l.tokens = append(l.tokens, spec.Token{
                Type:     tokenType,
                Value:    value,
                Position: l.position,
        })
}

// Tokenize converts the input string into tokens
func (l *Lexer) Tokenize() ([]spec.Token, error) {
        for l.current != 0 {
                l.skipWhitespace()
                
                if l.current == 0 {
                        break
                }
                
                switch l.current {
                case '$':
                        l.addToken(spec.TokenRoot, "$")
                        l.advance()
                case '.':
                        l.addToken(spec.TokenDot, ".")
                        l.advance()
                case '[':
                        l.advance() // skip opening bracket
                        content, quoted, err := l.readBracketContent()
                        if err != nil {
                                return nil, err
                        }
                        if l.current != ']' {
                                return nil, fmt.Errorf("expected closing bracket ']' at position %d", l.position)
                        }
                        
                        // Store the content with metadata about quoting
                        if quoted {
                                l.addToken(spec.TokenString, `"`+content+`"`)
                        } else {
                                l.addToken(spec.TokenString, content)
                        }
                        l.advance() // skip closing bracket
                case ']':
                        l.addToken(spec.TokenRBracket, "]")
                        l.advance()
                case '(':
                        l.addToken(spec.TokenLParen, "(")
                        l.advance()
                case ')':
                        l.addToken(spec.TokenRParen, ")")
                        l.advance()
                case '|':
                        l.addToken(spec.TokenPipe, "|")
                        l.advance()
                case '{':
                        l.advance()
                        if l.current == '*' {
                                l.advance()
                                if l.current == '}' {
                                        l.addToken(spec.TokenStar, "{*}")
                                        l.advance()
                                } else {
                                        return nil, fmt.Errorf("expected '}' after '{*' at position %d", l.position)
                                }
                        } else {
                                l.addToken(spec.TokenLBrace, "{")
                        }
                case '}':
                        l.addToken(spec.TokenRBrace, "}")
                        l.advance()
                default:
                        if isLetter(l.current) || l.current == '_' {
                                identifier := l.readIdentifier()
                                if identifier == "" {
                                        return nil, fmt.Errorf("invalid identifier at position %d", l.position)
                                }
                                l.addToken(spec.TokenIdentifier, identifier)
                        } else {
                                return nil, fmt.Errorf("unexpected character '%c' at position %d", l.current, l.position)
                        }
                }
        }
        
        l.addToken(spec.TokenEOF, "")
        return l.tokens, nil
}

// Helper functions
func isLetter(ch byte) bool {
        return (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z')
}

func isLetterOrDigit(ch byte) bool {
        return isLetter(ch) || (ch >= '0' && ch <= '9')
}