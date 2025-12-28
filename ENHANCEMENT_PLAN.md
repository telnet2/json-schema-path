# Schema-Path Enhancement Plan

## Executive Summary

This document outlines a comprehensive plan to enhance the schema-path expression language for improved JSON Schema validation capabilities. The enhancements focus on four key areas: wildcard support, additional quantifiers, AST normalization, and bounded repetition. Each enhancement is designed to maintain the current O(n) parsing complexity and O(n*m) matching performance where n is path length and m is pattern complexity.

---

## Table of Contents

1. [Current State Analysis](#1-current-state-analysis)
2. [Enhancement 1: Wildcard Operator Support](#2-enhancement-1-wildcard-operator-support)
3. [Enhancement 2: One-or-More Quantifier `{+}`](#3-enhancement-2-one-or-more-quantifier)
4. [Enhancement 3: AST Representation Normalization](#4-enhancement-3-ast-representation-normalization)
5. [Enhancement 4: Bounded Repetition `{n,m}`](#5-enhancement-4-bounded-repetition-nm)
6. [Implementation Priority and Dependencies](#6-implementation-priority-and-dependencies)
7. [Performance Considerations](#7-performance-considerations)
8. [Testing Strategy](#8-testing-strategy)

---

## 1. Current State Analysis

### 1.1 Existing Capabilities

The schema-path language currently supports:

| Feature | Syntax | Example |
|---------|--------|---------|
| Root anchor | `$` | `$.schema` |
| Property access | `.name` | `$.user.name` |
| Bracket notation | `["key"]` | `$.data["special-key"]` |
| Group alternatives | `(a\|b)` | `$.(properties\|definitions)` |
| Zero-or-more repetition | `{*}` | `$.(child){*}` |

### 1.2 Architecture Overview

```
┌─────────────┐     ┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│   Input     │────▶│   Lexer     │────▶│   Parser    │────▶│  Pattern    │
│  Expression │     │ (lexer.go)  │     │ (parser.go) │     │    Tree     │
└─────────────┘     └─────────────┘     └─────────────┘     │  (tree.go)  │
                           │                   │             └─────────────┘
                           ▼                   ▼                    │
                      Token Stream      AST (spec.go)               ▼
                                                              Path Matching
```

### 1.3 Identified Gaps

For comprehensive JSON Schema validation, the following capabilities are missing:

1. **Wildcard matching**: Cannot express "any property" patterns like `$.properties.*.type`
2. **Required repetition**: No way to require at least one match (`{+}` semantics)
3. **Inconsistent representation**: AST output differs from input format
4. **Unbounded only**: Cannot limit recursion depth for safety/performance

---

## 2. Enhancement 1: Wildcard Operator Support

### 2.1 Motivation

JSON Schema validators frequently need to match patterns across all properties of an object without knowing property names in advance. Consider validating that every property in a schema has a `type` field:

```json
{
  "properties": {
    "name": {"type": "string"},
    "age": {"type": "integer"},
    "email": {"type": "string"}
  }
}
```

**Current limitation**: Must enumerate all property names or use external iteration.

**Desired expression**: `$.properties.*.type` - matches `type` field under any property.

### 2.2 Design

#### 2.2.1 Syntax

```
Wildcard ::= "*"
```

The wildcard `*` matches any single path segment (property name or array index).

#### 2.2.2 Semantic Rules

| Pattern | Matches | Does Not Match |
|---------|---------|----------------|
| `$.*.name` | `$.user.name`, `$.admin.name` | `$.name`, `$.a.b.name` |
| `$.data.*` | `$.data.x`, `$.data.0` | `$.data`, `$.data.x.y` |
| `$.*.*` | `$.a.b`, `$.x.y` | `$.a`, `$.a.b.c` |

#### 2.2.3 AST Node Addition

```go
// spec/specification.go

// WildcardNode represents * matching any single segment
type WildcardNode struct{}

func (w *WildcardNode) String() string { return "*" }
```

#### 2.2.4 Lexer Modification

```go
// parser/lexer.go - Add to Tokenize() switch statement

case '*':
    // Check if this is a standalone wildcard or part of {*}
    if l.peek() != '}' {
        l.addToken(spec.TokenWildcard, "*")
    }
    l.advance()
```

#### 2.2.5 Tree Matching Logic

```go
// tree/tree.go - Add to matchFromNode()

case NodeWildcard:
    // Wildcard matches any single segment
    // Simply consume current segment and continue matching
    if t.matchFromNode(child, path, pathIndex+1) {
        return true
    }
```

### 2.3 Design Rationale

**Why `*` syntax?**
- Consistent with JSONPath standard (`$..*.name`)
- Familiar to users from glob patterns and regex
- Single character for conciseness

**Why single-segment matching only?**
- Maintains predictable O(n) matching per path
- Deep wildcards (`**`) would require exponential backtracking
- Users can combine with `{*}` for recursive patterns: `$.(*.child){*}`

**Performance impact**: None. Wildcard matching is O(1) per segment - simply skip segment comparison and proceed.

---

## 3. Enhancement 2: One-or-More Quantifier `{+}`

### 3.1 Motivation

The current `{*}` quantifier allows zero repetitions, which may not always be desired. Consider matching a linked list that must have at least one node:

```
Expression: $.list.(next){*}.value
Problem:    Matches $.list.value (zero repetitions)
Desired:    Only match $.list.next.value, $.list.next.next.value, etc.
```

In JSON Schema validation, this is critical for patterns like:
- Requiring at least one level of nesting
- Ensuring recursive structures have minimum depth
- Validating that certain paths must traverse specific intermediate nodes

### 3.2 Design

#### 3.2.1 Syntax

```
Repetition ::= "{*}" | "{+}"
```

| Quantifier | Meaning | Matches |
|------------|---------|---------|
| `{*}` | Zero or more | 0, 1, 2, 3, ... repetitions |
| `{+}` | One or more | 1, 2, 3, ... repetitions |

#### 3.2.2 AST Modification

```go
// spec/specification.go

type GroupNode struct {
    Alternatives [][]ASTNode
    Repetition   RepetitionType  // Changed from bool
}

type RepetitionType int

const (
    RepetitionNone     RepetitionType = iota  // No repetition
    RepetitionZeroMore                         // {*}
    RepetitionOneMore                          // {+}
)
```

#### 3.2.3 Lexer Modification

```go
// parser/lexer.go - Modify the '{' case

case '{':
    l.advance()
    switch l.current {
    case '*':
        l.advance()
        if l.current == '}' {
            l.addToken(spec.TokenZeroMore, "{*}")
            l.advance()
        } else {
            return nil, fmt.Errorf("expected '}' after '{*'")
        }
    case '+':
        l.advance()
        if l.current == '}' {
            l.addToken(spec.TokenOneMore, "{+}")
            l.advance()
        } else {
            return nil, fmt.Errorf("expected '}' after '{+'")
        }
    default:
        l.addToken(spec.TokenLBrace, "{")
    }
```

#### 3.2.4 Tree Matching Logic

```go
// tree/tree.go

func (t *PatternTree) matchGroupNode(groupNode *TreeNode, path []string, pathIndex int) bool {
    switch groupNode.RepetitionType {
    case RepetitionNone:
        return t.matchNonRepeatingGroup(groupNode, path, pathIndex)
    case RepetitionZeroMore:
        return t.matchZeroOrMoreGroup(groupNode, path, pathIndex)
    case RepetitionOneMore:
        return t.matchOneOrMoreGroup(groupNode, path, pathIndex)
    }
    return false
}

func (t *PatternTree) matchOneOrMoreGroup(groupNode *TreeNode, path []string, pathIndex int) bool {
    // Must match at least one iteration - do NOT try zero iterations first
    for _, alternative := range groupNode.Alternatives {
        currentIndex := pathIndex
        if t.matchAlternativeSegments(alternative, path, &currentIndex) {
            // After one iteration, can do zero or more additional
            if t.matchZeroOrMoreGroup(groupNode, path, currentIndex) {
                return true
            }
        }
    }
    return false
}
```

### 3.3 Design Rationale

**Why `{+}` syntax?**
- Consistent with regex convention where `+` means "one or more"
- Natural extension of existing `{*}` syntax
- Users familiar with regex will immediately understand

**Why not `{1,}` syntax?**
- More verbose than necessary for common case
- `{+}` is cleaner and more readable
- Bounded repetition `{n,m}` is a separate enhancement (see Section 5)

**Implementation approach - reuse `{*}` logic**:
The `{+}` matching is implemented by requiring one successful match first, then delegating to `{*}` logic. This:
- Minimizes code duplication
- Ensures consistent behavior for subsequent matches
- Makes the "at least one" semantics crystal clear in code

**Performance impact**: Identical to `{*}`. The only difference is skipping the "zero iterations" branch.

---

## 4. Enhancement 3: AST Representation Normalization

### 4.1 Motivation

Currently, the AST output differs from the input expression in subtle ways:

```
Input:   $.(properties|definitions){*}.type
Output:  $(.properties|.definitions){*}.type
```

The leading dot before the group moves inside each alternative. While semantically equivalent, this creates:

1. **Confusion**: Users expect output to match input
2. **Testing difficulty**: String comparisons fail unexpectedly
3. **Round-trip problems**: Parse → String → Parse may not be idempotent

### 4.2 Design

#### 4.2.1 Normalization Rules

Define a canonical form where AST `String()` output matches parsed input:

| Input | Current Output | Normalized Output |
|-------|----------------|-------------------|
| `$.(a\|b)` | `$(.a\|.b)` | `$.(a\|b)` |
| `$.x.(a\|b)` | `$.x(.a\|.b)` | `$.x.(a\|b)` |
| `$.(a.b\|c)` | `$(.a.b\|.c)` | `$.(a.b\|c)` |

#### 4.2.2 Implementation Approach

Modify `GroupNode.String()` to track context:

```go
// spec/specification.go

func (g *GroupNode) String() string {
    result := ".("  // Always include leading dot for group as segment
    for i, alt := range g.Alternatives {
        if i > 0 {
            result += "|"
        }
        for j, node := range alt {
            s := node.String()
            // For first node in alternative, strip leading dot if present
            if j == 0 && len(s) > 0 && s[0] == '.' {
                s = s[1:]
            }
            // For subsequent nodes, keep the dot (it's a separator)
            if j > 0 {
                result += "."
            }
            result += s
        }
    }
    result += ")"
    if g.Repetition {
        result += "{*}"
    }
    return result
}
```

#### 4.2.3 Alternative: Store Original Form

Instead of reconstructing, store the original text:

```go
type GroupNode struct {
    Alternatives [][]ASTNode
    Repetition   bool
    OriginalText string  // Preserve input exactly
}

func (g *GroupNode) String() string {
    if g.OriginalText != "" {
        return g.OriginalText
    }
    // Fall back to reconstruction
    ...
}
```

### 4.3 Design Rationale

**Why normalize output to match input?**
- Principle of least surprise: output should resemble input
- Enables reliable testing with string equality
- Supports round-trip parsing for tooling (formatters, linters)

**Why not change the internal representation?**
- Internal form with explicit dots is actually clearer for matching logic
- Normalization is purely a presentation concern
- Keeps AST semantically unambiguous

**Trade-off considered**: Storing original text is simpler but uses more memory. For most use cases, reconstruction is preferable as expressions are typically short.

**Performance impact**: None for parsing/matching. Minor overhead on `String()` calls, which are typically used only for debugging/display.

---

## 5. Enhancement 4: Bounded Repetition `{n,m}`

### 5.1 Motivation

Unbounded recursion with `{*}` can be problematic:

1. **Performance risk**: Deeply nested structures may cause excessive matching
2. **Validation requirements**: Some schemas define maximum nesting depth
3. **Safety**: Prevent denial-of-service via pathological inputs

Example use cases:
- Limit tree traversal to 10 levels: `$.(left|right){0,10}.value`
- Require exactly 3 levels: `$.(child){3}.data`
- Allow 2-5 repetitions: `$.(next){2,5}.value`

### 5.2 Design

#### 5.2.1 Syntax

```
Repetition ::= "{*}" | "{+}" | "{" Number "}" | "{" Number? "," Number? "}"
Number     ::= [0-9]+
```

| Syntax | Meaning | Equivalent |
|--------|---------|------------|
| `{*}` | Zero or more | `{0,}` |
| `{+}` | One or more | `{1,}` |
| `{3}` | Exactly 3 | `{3,3}` |
| `{2,5}` | 2 to 5 inclusive | - |
| `{,5}` | 0 to 5 | `{0,5}` |
| `{3,}` | 3 or more | - |

#### 5.2.2 AST Modification

```go
// spec/specification.go

type GroupNode struct {
    Alternatives [][]ASTNode
    MinRepeat    int   // Minimum repetitions (0 for {*}, 1 for {+})
    MaxRepeat    int   // Maximum repetitions (-1 for unlimited)
}

// Helper constants
const UnlimitedRepeat = -1
```

#### 5.2.3 Lexer Modification

```go
// parser/lexer.go

func (l *Lexer) readRepetition() (min, max int, err error) {
    // Already consumed '{'

    // Check for {*} or {+} shortcuts
    if l.current == '*' {
        l.advance()
        if l.current != '}' {
            return 0, 0, fmt.Errorf("expected '}' after '*'")
        }
        l.advance()
        return 0, UnlimitedRepeat, nil
    }
    if l.current == '+' {
        l.advance()
        if l.current != '}' {
            return 0, 0, fmt.Errorf("expected '}' after '+'")
        }
        l.advance()
        return 1, UnlimitedRepeat, nil
    }

    // Parse {n}, {n,}, {,m}, or {n,m}
    min = -1
    max = -1

    // Parse first number if present
    if isDigit(l.current) {
        min = l.readNumber()
    }

    if l.current == '}' {
        // {n} - exactly n
        if min < 0 {
            return 0, 0, fmt.Errorf("empty repetition {}")
        }
        l.advance()
        return min, min, nil
    }

    if l.current != ',' {
        return 0, 0, fmt.Errorf("expected ',' or '}' in repetition")
    }
    l.advance() // consume ','

    // Parse second number if present
    if isDigit(l.current) {
        max = l.readNumber()
    }

    if l.current != '}' {
        return 0, 0, fmt.Errorf("expected '}' in repetition")
    }
    l.advance()

    // Apply defaults
    if min < 0 {
        min = 0 // {,m} means {0,m}
    }
    if max < 0 {
        max = UnlimitedRepeat // {n,} means unlimited
    }

    // Validate
    if max != UnlimitedRepeat && min > max {
        return 0, 0, fmt.Errorf("invalid repetition: min %d > max %d", min, max)
    }

    return min, max, nil
}
```

#### 5.2.4 Tree Matching Logic

```go
// tree/tree.go

func (t *PatternTree) matchBoundedGroup(
    groupNode *TreeNode,
    path []string,
    pathIndex int,
    currentCount int,
) bool {
    minRepeat := groupNode.MinRepeat
    maxRepeat := groupNode.MaxRepeat

    // Check if we've reached minimum - try to continue after group
    if currentCount >= minRepeat {
        if t.continueAfterGroup(groupNode, path, pathIndex) {
            return true
        }
    }

    // Check if we've hit maximum
    if maxRepeat != UnlimitedRepeat && currentCount >= maxRepeat {
        return false
    }

    // Try one more iteration
    for _, alternative := range groupNode.Alternatives {
        tempIndex := pathIndex
        if t.matchAlternativeSegments(alternative, path, &tempIndex) {
            if t.matchBoundedGroup(groupNode, path, tempIndex, currentCount+1) {
                return true
            }
        }
    }

    return false
}
```

### 5.3 Design Rationale

**Why support full `{n,m}` syntax?**
- Complete solution covering all common repetition patterns
- Familiar syntax from regex
- Subsumes `{*}` and `{+}` as special cases (backwards compatible)

**Why include exact repetition `{n}`?**
- Common use case: "exactly 3 levels deep"
- More readable than `{3,3}`
- Natural extension of the syntax

**Why allow omitting bounds?**
- `{n,}` for "at least n" is cleaner than `{n,999999}`
- `{,m}` for "at most m" reads naturally
- Consistent with regex conventions

**Performance consideration**:
The `currentCount` parameter adds one integer to the call stack per recursion level. For bounded repetitions, this is capped at `maxRepeat` levels. For unbounded, it's capped by path length. No performance degradation compared to current implementation.

**Safety benefit**:
With bounded repetition, validators can set reasonable limits:
```go
// Limit recursion to prevent DoS
pattern := "$.(properties|items){0,100}.type"
```

---

## 6. Implementation Priority and Dependencies

### 6.1 Dependency Graph

```
                    ┌─────────────────┐
                    │  Enhancement 3  │
                    │ AST Normalization│
                    └────────┬────────┘
                             │ (independent)
                             ▼
┌─────────────────┐   ┌─────────────────┐   ┌─────────────────┐
│  Enhancement 1  │   │  Enhancement 2  │   │  Enhancement 4  │
│    Wildcard     │   │      {+}        │◀──│     {n,m}       │
└─────────────────┘   └─────────────────┘   └─────────────────┘
        │                     │                      │
        │                     │                      │
        └─────────┬───────────┴──────────────────────┘
                  ▼
         All require changes to:
         - spec/specification.go (AST nodes)
         - parser/lexer.go (tokenization)
         - parser/parser.go (parsing)
         - tree/tree.go (matching)
```

### 6.2 Recommended Implementation Order

| Phase | Enhancement | Effort | Value | Risk |
|-------|-------------|--------|-------|------|
| **1** | Wildcard `*` | Medium | High | Low |
| **2** | AST Normalization | Low | Medium | Low |
| **3** | One-or-more `{+}` | Low | Medium | Low |
| **4** | Bounded `{n,m}` | Medium | Medium | Medium |

**Rationale**:

1. **Wildcard first**: Highest value for JSON Schema validation. Enables `$.properties.*.type` pattern that is currently impossible.

2. **AST Normalization second**: Low effort, improves developer experience and testing. No external dependencies.

3. **`{+}` quantifier third**: Builds naturally on existing `{*}` implementation. Small change with clear value.

4. **Bounded repetition last**: Most complex change. Requires refactoring existing repetition handling. Can be deferred if not immediately needed.

### 6.3 Estimated Effort

| Enhancement | Files Modified | Lines of Code | Testing Effort |
|-------------|----------------|---------------|----------------|
| Wildcard | 4 | ~50 | Medium |
| AST Normalization | 1 | ~30 | Low |
| `{+}` Quantifier | 4 | ~40 | Low |
| Bounded `{n,m}` | 4 | ~120 | High |

---

## 7. Performance Considerations

### 7.1 Current Performance Characteristics

| Operation | Complexity | Notes |
|-----------|------------|-------|
| Lexing | O(n) | n = expression length |
| Parsing | O(n) | Single pass, no backtracking |
| Tree Building | O(n) | Linear in AST size |
| Path Matching | O(n × m) | n = path length, m = pattern size |

### 7.2 Enhancement Impact Analysis

| Enhancement | Parsing Impact | Matching Impact |
|-------------|----------------|-----------------|
| Wildcard | None | None (O(1) per segment) |
| AST Normalization | None | None (display only) |
| `{+}` Quantifier | None | None (same as `{*}`) |
| Bounded `{n,m}` | None | O(1) extra per recursion |

### 7.3 Worst-Case Scenarios

**Concern**: Could enhancements create pathological cases?

**Analysis**:

1. **Wildcard**: Matches exactly one segment. No backtracking. Safe.

2. **`{+}`**: Strictly fewer branches than `{*}` (skips zero-iteration case). Actually faster.

3. **Bounded `{n,m}`**: Adds early termination at `maxRepeat`. Potentially faster than unbounded.

**Conclusion**: All enhancements maintain or improve performance characteristics.

### 7.4 Benchmarking Strategy

Before and after benchmarks should cover:

```go
// Benchmark cases
var benchCases = []struct {
    name    string
    pattern string
    path    []string
}{
    {"shallow", "$.a.b.c", []string{"a", "b", "c"}},
    {"deep_recursive", "$.(a){*}.b", strings.Split("a.a.a.a.a.a.a.a.a.a.b", ".")},
    {"wide_alternatives", "$.(a|b|c|d|e){*}.x", []string{"a", "b", "c", "d", "e", "x"}},
    {"wildcard_chain", "$.*.*.*", []string{"x", "y", "z"}},
    {"bounded_deep", "$.(a){0,5}.b", []string{"a", "a", "a", "b"}},
}
```

---

## 8. Testing Strategy

### 8.1 Test Categories

#### 8.1.1 Unit Tests (per enhancement)

```go
// Enhancement 1: Wildcard
func TestWildcardParsing(t *testing.T) { ... }
func TestWildcardMatching(t *testing.T) { ... }
func TestWildcardWithGroups(t *testing.T) { ... }

// Enhancement 2: {+} Quantifier
func TestOnePlusQuantifierParsing(t *testing.T) { ... }
func TestOnePlusQuantifierMatching(t *testing.T) { ... }
func TestOnePlusVsZeroPlus(t *testing.T) { ... }

// Enhancement 3: AST Normalization
func TestASTRoundTrip(t *testing.T) { ... }
func TestASTOutputFormat(t *testing.T) { ... }

// Enhancement 4: Bounded Repetition
func TestBoundedRepetitionParsing(t *testing.T) { ... }
func TestBoundedRepetitionMatching(t *testing.T) { ... }
func TestBoundedEdgeCases(t *testing.T) { ... }
```

#### 8.1.2 Integration Tests

```go
func TestJSONSchemaValidationPatterns(t *testing.T) {
    patterns := []struct {
        name        string
        expression  string
        schema      string
        shouldMatch []string
    }{
        {
            name:       "AllPropertyTypes",
            expression: "$.properties.*.type",
            schema:     `{"properties":{"a":{"type":"string"},"b":{"type":"int"}}}`,
            shouldMatch: []string{"$.properties.a.type", "$.properties.b.type"},
        },
        {
            name:       "RequiredNesting",
            expression: "$.(properties){+}.type",
            schema:     `{"properties":{"x":{"type":"object","properties":{"y":{"type":"string"}}}}}`,
            shouldMatch: []string{"$.properties.x.properties.y.type"},
        },
        // ... more cases
    }
}
```

#### 8.1.3 Regression Tests

Ensure all existing tests continue to pass:

```bash
go test ./... -v
```

#### 8.1.4 Fuzz Testing

```go
func FuzzExpressionParsing(f *testing.F) {
    f.Add("$.a.b.c")
    f.Add("$.(a|b){*}.c")
    f.Add("$.*.x")
    f.Add("$.(a){+}.b")
    f.Add("$.(a){2,5}.b")

    f.Fuzz(func(t *testing.T, expr string) {
        // Should not panic
        parsed, err := parser.ParseExpression(expr)
        if err == nil {
            // Round-trip should work
            reparsed, err2 := parser.ParseExpression(parsed.String())
            if err2 != nil {
                t.Errorf("Round-trip failed: %s -> %s -> error", expr, parsed.String())
            }
        }
    })
}
```

### 8.2 Test Coverage Targets

| Component | Current Coverage | Target Coverage |
|-----------|------------------|-----------------|
| parser/lexer.go | ~80% | 95% |
| parser/parser.go | ~75% | 95% |
| tree/tree.go | ~70% | 90% |
| spec/specification.go | ~60% | 85% |

---

## Appendix A: Grammar Summary (Post-Enhancement)

```ebnf
Expression      ::= Root Path?
Root            ::= "$"
Path            ::= Segment*
Segment         ::= "." SegmentItem | BracketNotation
SegmentItem     ::= Identifier | Wildcard | GroupExpression
Identifier      ::= [a-zA-Z_][a-zA-Z0-9_]*
Wildcard        ::= "*"
BracketNotation ::= "[" BracketContent "]"
BracketContent  ::= QuotedString | UnquotedString
QuotedString    ::= '"' (EscapedChar | [^"\\])* '"'
UnquotedString  ::= (EscapedChar | [^\]\\])*
EscapedChar     ::= "\\" .
GroupExpression ::= "(" GroupSeq ("|" GroupSeq)* ")" Repetition?
GroupSeq        ::= GroupPrimary ("." GroupPrimary)*
GroupPrimary    ::= Identifier BracketSuffix? | Wildcard | GroupExpression
BracketSuffix   ::= (BracketNotation)+
Repetition      ::= "{*}" | "{+}" | "{" Number "}" | "{" Number? "," Number? "}"
Number          ::= [0-9]+
```

---

## Appendix B: Example Expressions (Post-Enhancement)

| Use Case | Expression | Description |
|----------|------------|-------------|
| All property types | `$.properties.*.type` | Type of every property |
| Required recursion | `$.(child){+}.value` | At least one child level |
| Limited depth | `$.(left\|right){0,10}.data` | Binary tree max 10 deep |
| Exact depth | `$.levels.*.items{3}.name` | Exactly 3 levels of items |
| Complex schema | `$.(properties\|items\|additionalProperties){*}.*.type` | All types in schema |

---

## Appendix C: Backwards Compatibility

All enhancements are **fully backwards compatible**:

1. **Existing expressions**: Parse and execute identically
2. **Existing API**: No breaking changes to public interfaces
3. **Existing tests**: Continue to pass without modification

New functionality is purely additive.

---

*Document Version: 1.0*
*Last Updated: 2024*
*Status: Proposed*
