package spec

/*
JSON Path Expression Formal Specification
=========================================

This specification defines a JSON path expression language that extends JSONPath
with recursive structure support and advanced bracket notation.

EBNF Grammar:
------------

Expression      ::= Root Path?
Root            ::= "$"
Path            ::= Segment*
Segment         ::= "." SegmentItem | BracketNotation
SegmentItem     ::= Identifier | GroupExpression
Identifier      ::= [a-zA-Z_][a-zA-Z0-9_]*
BracketNotation ::= "[" BracketContent "]"
BracketContent  ::= QuotedString | UnquotedString
QuotedString    ::= '"' (EscapedChar | [^"\\])* '"'
UnquotedString  ::= (EscapedChar | [^\]\\])*
EscapedChar     ::= "\\" .
GroupExpression ::= "(" GroupSeq ("|" GroupSeq)* ")" Repetition?
GroupSeq        ::= GroupPrimary ("." GroupPrimary)*
GroupPrimary    ::= Identifier BracketSuffix? | GroupExpression
BracketSuffix   ::= (BracketNotation)+
Repetition      ::= "{*}"

Semantic Rules:
--------------

1. Root Expression:
   - Every expression must start with "$" representing the root of JSON document

2. Property Access:
   - ".property" accesses object property
   - Property names follow identifier rules: [a-zA-Z_][a-zA-Z0-9_]*

3. Bracket Notation:
   - For Objects:
     * X[property] - accesses property literally (property name as-is)
     * X["quoted"] - accesses quoted property, supports escape sequences
     * X["\"escaped"] - property name is literal `"escaped`
   - For Arrays:
     * Applied to every array element (homogeneous assumption)
     * Same syntax as objects, operates on each element

4. Escape Sequences:
   - "\\" followed by any character (.) escapes that character literally
   - In quoted strings: \" escapes quote, \\ escapes backslash
   - In unquoted bracket content: \] escapes closing bracket, \\ escapes backslash
   - Any character can be escaped; the escaped character is included literally

5. Group Expressions:
   - A leading "." introduces the group as a segment: .(alternative1|alternative2|...)
   - Inside groups, each alternative is a sequence without a leading dot for the first item
   - Identifiers can be followed by bracket notation without dots (e.g., "meta[\"child\"]") 
   - Subsequent items within a group alternative are separated by dots
   - Each alternative can be a sequence of multiple path segments (e.g., "meta.child")
   - Used for representing alternative paths in recursive structures
   - AST String() representation avoids leading dots on first item of group alternatives

6. Repetition:
   - {*} after a group means zero or more repetitions of the entire group
   - Applies to all alternatives within the group
   - Enables recursive structure representation
   - Only valid after group expressions

7. Bracket Notation Scope:
   - Numeric array indices are not supported (arrays use element-wise application)
   - Wildcard operators (*) are not supported in bracket notation
   - Only property name literals and quoted strings are supported
   - Group alternatives cannot start with bracket notation (must start with identifier)
   - Multiple consecutive brackets are allowed after identifiers (e.g., meta["a"]["b"])

Examples:
--------

Basic Property Access:
- $.node.value
- $.user.name

Bracket Notation:
- $.data[property]          # literal property access
- $.data["quoted-name"]     # quoted property access  
- $.data["\"special"]       # property name is `"special`
- $.array[element]          # applied to each array element

Recursive Structures:
- $.node.(child|meta.child){*}.value
  Represents: node -> (child OR meta.child) -> repeat -> value
  
- $.tree.(left|right){*}.data
  Represents: tree -> (left OR right) -> repeat -> data

Group Expression Parsing:
- $.node.(child|meta["child"]){*}.value
  Demonstrates: identifier+bracket adjacency within group alternatives
  
- $.data.(items["key"]["subkey"]|nested.values){*}
  Demonstrates: multiple brackets and mixed property access in group alternatives

- $.root.(a.b["c"]|x["y"].z){*}
  Demonstrates: complex sequences with mixed dot-separated and adjacent notation

Type System Implications:
------------------------

The path expression $.node.(child|meta.child){*}.value implies:

type Node struct {
    Value interface{}           `json:"value"`
    Child *Node                `json:"child,omitempty"`
    Meta  *struct {
        Child *Node            `json:"child,omitempty"`
    } `json:"meta,omitempty"`
}

Parsing Algorithm:
-----------------

1. Lexical Analysis:
   - Tokenize input into: ROOT, DOT, IDENTIFIER, LBRACKET, RBRACKET, 
     LPAREN, RPAREN, PIPE, LBRACE, STAR, RBRACE, STRING, EOF
   
2. Syntactic Analysis:
   - Recursive descent parser following EBNF grammar
   - Build Abstract Syntax Tree (AST)

3. Semantic Analysis:
   - Validate group expressions have proper repetition syntax
   - Check escape sequences are valid
   - Ensure brackets are properly matched

4. Tree Construction:
   - Build trie/radix tree from AST for efficient pattern matching
   - Handle repetition by creating cycle-aware nodes
   - Support path expansion for testing against actual JSON paths
*/

// TokenType represents the different types of tokens in the path expression
type TokenType int

const (
        TokenEOF TokenType = iota
        TokenRoot          // $
        TokenDot           // .
        TokenIdentifier    // property name
        TokenLBracket      // [
        TokenRBracket      // ]
        TokenLParen        // (
        TokenRParen        // )
        TokenPipe          // |
        TokenLBrace        // {
        TokenStar          // *
        TokenRBrace        // }
        TokenString        // quoted or unquoted content
        TokenInvalid
)

// Token represents a single token in the path expression
type Token struct {
        Type     TokenType
        Value    string
        Position int
}

// ASTNode represents a node in the Abstract Syntax Tree
type ASTNode interface {
        String() string
}

// RootNode represents the $ root of the expression
type RootNode struct{}

func (r *RootNode) String() string { return "$" }

// PropertyNode represents .property access
type PropertyNode struct {
        Name string
}

func (p *PropertyNode) String() string { return "." + p.Name }

// BracketNode represents [content] access
type BracketNode struct {
        Content string
        Quoted  bool
}

func (b *BracketNode) String() string {
        if b.Quoted {
                return "[\"" + b.Content + "\"]"
        }
        return "[" + b.Content + "]"
}

// GroupNode represents (expr1|expr2|...) with optional {*}
type GroupNode struct {
        Alternatives [][]ASTNode // Each alternative is a sequence of path segments
        Repetition   bool        // true if followed by {*}
}

func (g *GroupNode) String() string {
        result := "("
        for i, alt := range g.Alternatives {
                if i > 0 {
                        result += "|"
                }
                for _, node := range alt {
                        result += node.String()
                }
        }
        result += ")"
        if g.Repetition {
                result += "{*}"
        }
        return result
}

// PathExpression represents the complete parsed path expression
type PathExpression struct {
        Root     *RootNode
        Segments []ASTNode
}

func (p *PathExpression) String() string {
        result := p.Root.String()
        for _, segment := range p.Segments {
                result += segment.String()
        }
        return result
}