# Schema Path Expression Specification

This document defines the schema-path expression language implemented by this
project.  The language borrows the familiar `$.property` style from JSONPath and
extends it to express recursive shapes, wildcard properties, regular-expression
matching, and repetition of object chains.

## Lexical Structure

* **Root** – every expression begins with the root token `$`.
* **Dot (`.`)** – separates object properties and group expressions.
* **Identifiers** – property names following `[a-zA-Z_][a-zA-Z0-9_]*`.
* **Bracket notation** – `[ ... ]` introduces special selectors:
  * `["name"]` or `[name]` – literal property access with escaping support.
  * `["escaped\"name"]` – uses `\` to escape quotes and brackets.
  * `[#pattern]` – file-system style wildcard using `*`, `?`, and character
    classes; implemented with Go's `path.Match` rules.
  * `[~regex]` – regular-expression selector using Go's RE syntax.
  * `[N]` – zero-based array index.
  * `[*]` – array wildcard matching any index.
* **Groups** – parenthesised sequences `( ... )` with alternatives separated by
  `|`.
* **Repetition** – `{*}` applies to the preceding group or identifier sequence,
  denoting zero-or-more occurrences.

Whitespace may appear between tokens and is ignored.

## Grammar (EBNF)

```
Expression      ::= Root Path?
Root            ::= "$"
Path            ::= Segment*
Segment         ::= "." SegmentItem | BracketNotation
SegmentItem     ::= Identifier Repetition? | GroupExpression
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

## Semantics

* **Properties** – `$.user.name` navigates object members using identifiers.
* **Bracket properties** – allow property names that would otherwise require
  quoting, plus wildcard and regex matching.
* **Array indices** – `[N]` advances into the Nth array element; `[*]` expands
  to every element.
* **Groups** – `(child|meta.child)` creates alternatives evaluated relative to
  the group's entry position.  Each alternative is a sequence of AST nodes.
* **Repetition** – `{*}` performs a Kleene star over the preceding sequence.
  Repetition can apply both to a group and to a simple identifier chain (e.g.
  `meta{*}` expands to zero or more `.meta` hops).
* **Escaping** – `\` escapes the following character inside brackets, allowing
  literal `]` or `\` to appear in property names.

## Runtime Model

The parser produces an abstract syntax tree composed of the following node
types:

* `PropertyNode` – `.name`
* `BracketNode` – `[ ... ]` selectors with a discriminated union describing the
  underlying kind (literal, wildcard, regex, array index, array wildcard)
* `GroupNode` – group alternatives with an optional repetition flag
* `RepetitionNode` – wrapper for identifier/bracket sequences that repeat

The matcher compiles the AST into an epsilon-NFA-backed trie. Literal
transitions are shared between patterns so multiple expressions can be matched
simultaneously.  Repetition introduces epsilon transitions that loop back to the
sequence entry, enabling recursive descent without materialising infinite paths.

Matching operates on `spec.PathSegment` values that distinguish between property
segments and array indices.  The JSON processor in this repository converts
extracted document paths into these segments, allowing the trie to evaluate
wildcards and regular expressions efficiently.

## Examples

* `$.node.(child|meta.child){*}.value` – matches values reachable via any number
  of `child` hops, alternating between direct and `meta.child` edges.
* `$.config[#*service].instances[*].id` – selects the `id` of every instance for
  properties ending with `service`.
* `$.meta{*}.child.value` – follows zero or more `.meta` properties before
  selecting the final `.child.value`.

The specification above is implemented by the parser and matcher provided in
this repository.  Validation errors are raised when expressions violate the
grammar (for example an empty group alternative, malformed regex, or missing
closing bracket).
