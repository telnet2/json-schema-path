package spec

/*
Package spec contains the core types and grammar definition for the schema-path
expression language. The language extends standard JSONPath style navigation
with recursion aware grouping, repetition, wildcards, and regular-expression
selectors.

EBNF Grammar
------------

```
Expression      ::= Root Path?
Root            ::= "$"
Path            ::= Segment*
Segment         ::= "." SegmentItem | BracketNotation
SegmentItem     ::= Identifier BracketSuffix? Repetition? | GroupExpression
Identifier      ::= [a-zA-Z_][a-zA-Z0-9_]*
BracketNotation ::= "[" BracketContent "]"
BracketContent  ::= QuotedString | WildcardContent | RegexContent | Index | Property
QuotedString    ::= '"' (EscapedChar | [^"\\])* '"'
WildcardContent ::= "#" Property
RegexContent    ::= "~" Property
Index           ::= [0-9]+
Property        ::= (EscapedChar | [^]\\])*
EscapedChar     ::= "\\" .
GroupExpression ::= "(" GroupSeq ("|" GroupSeq)* ")" Repetition?
GroupSeq        ::= GroupPrimary ("." GroupPrimary)*
GroupPrimary    ::= Identifier BracketSuffix? Repetition? | GroupExpression
BracketSuffix   ::= BracketNotation+
Repetition      ::= "{*}"
```

Semantics
---------

* Every expression begins at the root symbol `$`.
* Dot segments select object properties using identifier syntax.
* Bracket notation supports:
  - property lookups with optional quoting and escaping,
  - file-system style wildcards prefixed with `#`,
  - regular-expression selectors prefixed with `~`,
  - numeric array indices, and
  - the array wildcard `[*]`.
* Group expressions provide alternatives separated by `|`. Each alternative is
  a sequence of path segments relative to the group entry position.
* `{*}` denotes zero or more repetitions. Repetition can follow either an
  explicit group or an identifier/identifier+bracket sequence (allowing
  expressions such as `meta{*}.child`).
* Escaping inside brackets follows JSON rules for quoted strings and allows `]`
  or `\` to be escaped for unquoted content.

Runtime Matching
----------------

Parsing produces an abstract syntax tree (AST) which is compiled into an
epsilon-NFA-backed trie. Literal transitions are shared allowing multiple
expressions to be matched efficiently against JSON documents. Wildcard and
regular-expression selectors carry the compiled pattern for fast evaluation.

The matcher operates on `PathSegment` values emitted by the JSON processor.
`PathSegment` distinguishes between property accesses and array indices so the
matcher can correctly apply literal, wildcard, or numeric transitions.
*/

import "strconv"

// TokenType represents the different types of tokens in the path expression.
type TokenType int

const (
	TokenEOF        TokenType = iota
	TokenRoot                 // $
	TokenDot                  // .
	TokenIdentifier           // property name
	TokenLParen               // (
	TokenRParen               // )
	TokenPipe                 // |
	TokenStar                 // {*}
	TokenBracket              // [content]
)

// Token represents a single token in the path expression.
type Token struct {
	Type     TokenType
	Value    string
	Position int
	Quoted   bool
}

// ASTNode represents a node in the Abstract Syntax Tree.
type ASTNode interface {
	String() string
}

// RootNode represents the $ root of the expression.
type RootNode struct{}

func (r *RootNode) String() string { return "$" }

// PropertyNode represents .property access.
type PropertyNode struct {
	Name string
}

func (p *PropertyNode) String() string { return "." + p.Name }

// BracketKind enumerates the supported bracket selector modes.
type BracketKind int

const (
	BracketProperty BracketKind = iota
	BracketPropertyWildcard
	BracketRegex
	BracketArrayIndex
	BracketArrayWildcard
)

// BracketNode represents [content] access.
type BracketNode struct {
	Kind   BracketKind
	Value  string
	Index  int
	Quoted bool
}

func (b *BracketNode) String() string {
	switch b.Kind {
	case BracketProperty:
		if b.Quoted || !isIdentifier(b.Value) {
			return "[\"" + escapeString(b.Value) + "\"]"
		}
		return "[" + b.Value + "]"
	case BracketPropertyWildcard:
		return "[#" + b.Value + "]"
	case BracketRegex:
		return "[~" + b.Value + "]"
	case BracketArrayIndex:
		return "[" + strconv.Itoa(b.Index) + "]"
	case BracketArrayWildcard:
		return "[*]"
	default:
		return "[]"
	}
}

// GroupNode represents (expr1|expr2|...) with optional repetition.
type GroupNode struct {
	Alternatives [][]ASTNode
	Repetition   bool
}

func (g *GroupNode) String() string {
	result := "("
	for i, alt := range g.Alternatives {
		if i > 0 {
			result += "|"
		}
		for j, node := range alt {
			part := node.String()
			if j == 0 && len(part) > 0 && part[0] == '.' {
				result += part[1:]
			} else {
				result += part
			}
		}
	}
	result += ")"
	if g.Repetition {
		result += "{*}"
	}
	return result
}

// SequenceNode represents a sequence of AST nodes like property[bracket].
type SequenceNode struct {
	Sequence []ASTNode
}

func (s *SequenceNode) String() string {
	result := ""
	for i, node := range s.Sequence {
		part := node.String()
		if i == 0 && len(part) > 0 && part[0] == '.' {
			result += part[1:]
		} else {
			result += part
		}
	}
	return result
}

// RepetitionNode wraps a sequence of AST nodes that repeat with {*} semantics.
type RepetitionNode struct {
	Sequence []ASTNode
}

func (r *RepetitionNode) String() string {
	result := ""
	for i, node := range r.Sequence {
		part := node.String()
		if i == 0 && len(part) > 0 && part[0] == '.' {
			result += part[1:]
		} else {
			result += part
		}
	}
	result += "{*}"
	return result
}

// PathExpression represents the complete parsed path expression.
type PathExpression struct {
	Root     *RootNode
	Segments []ASTNode
}

func (p *PathExpression) String() string {
	result := p.Root.String()
	for _, segment := range p.Segments {
		switch node := segment.(type) {
		case *GroupNode, *RepetitionNode, *SequenceNode:
			result += "." + node.String()
		default:
			result += node.String()
		}
	}
	return result
}

// SegmentType distinguishes between property and array segments in runtime paths.
type SegmentType int

const (
	SegmentProperty SegmentType = iota
	SegmentArrayIndex
)

// PathSegment represents a single JSON navigation step extracted from a document.
type PathSegment struct {
	Type  SegmentType
	Key   string
	Index int
}

// NewPropertySegment constructs a property path segment.
func NewPropertySegment(name string) PathSegment {
	return PathSegment{Type: SegmentProperty, Key: name}
}

// NewArrayIndexSegment constructs an array index path segment.
func NewArrayIndexSegment(index int) PathSegment {
	return PathSegment{Type: SegmentArrayIndex, Index: index, Key: strconv.Itoa(index)}
}

func isIdentifier(value string) bool {
	if value == "" {
		return false
	}
	for i, r := range value {
		if i == 0 {
			if !(r == '_' || r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z') {
				return false
			}
			continue
		}
		if !(r == '_' || r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9') {
			return false
		}
	}
	return true
}

func escapeString(value string) string {
	escaped := ""
	for _, r := range value {
		switch r {
		case '\\':
			escaped += "\\\\"
		case '"':
			escaped += "\\\""
		default:
			escaped += string(r)
		}
	}
	return escaped
}
