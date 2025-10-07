# Path-Based Schema Validation

## The Concept

Instead of traditional JSON Schema validation (complex nested structures), use **path patterns as schema keys**:

```json
{
  "$.users[*].email": {"type": "string", "format": "email"},
  "$.users[*].age": {"type": "number", "minimum": 0},
  "$.company{*}.name": {"type": "string", "required": true}
}
```

## Why This Is Better

### 1. Simpler Than Traditional JSON Schema

**Traditional JSON Schema** (verbose, nested):
```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "type": "object",
  "properties": {
    "users": {
      "type": "array",
      "items": {
        "type": "object",
        "properties": {
          "email": {
            "type": "string",
            "format": "email"
          },
          "age": {
            "type": "number",
            "minimum": 0
          }
        },
        "required": ["email"]
      }
    }
  }
}
```

**Path-Based Schema** (flat, clear):
```json
{
  "$.users[*].email": {"type": "string", "format": "email", "required": true},
  "$.users[*].age": {"type": "number", "minimum": 0}
}
```

### 2. Handles Recursion Elegantly

**Traditional** (complex $ref):
```json
{
  "definitions": {
    "node": {
      "type": "object",
      "properties": {
        "value": {"type": "string"},
        "children": {
          "type": "array",
          "items": {"$ref": "#/definitions/node"}
        }
      }
    }
  }
}
```

**Path-Based** (simple {*}):
```json
{
  "$.tree{*}.value": {"type": "string"},
  "$.tree{*}.children[*]": {"type": "object"}
}
```

### 3. More Flexible Patterns

```json
{
  "$.config[#*Service].url": {"type": "string", "format": "uri"},
  "$.users[~^admin_.*].permissions": {"type": "array"},
  "$.data.(items|products)[*].price": {"type": "number", "minimum": 0}
}
```

## The Validation Flow

```go
// 1. Define path-based validation rules
rules := map[string]SchemaObject{
    "$.users[*].email": {Type: "string", Format: "email"},
    "$.company{*}.departments[*].budget": {Type: "number", Minimum: 0},
}

// 2. Create validator from rules
validator := NewPathBasedValidator(rules)

// 3. Validate JSON data
jsonData := `{
    "users": [
        {"email": "alice@example.com"},
        {"email": "invalid-email"}
    ],
    "company": {
        "departments": [
            {"name": "Engineering", "budget": 100000},
            {"name": "Marketing", "budget": -50000}
        ]
    }
}`

report := validator.Validate(jsonData)

// 4. Get validation results
for _, result := range report.Errors {
    fmt.Printf("%s: %s\n", result.Path, result.Error)
}
// Output:
// $.users[1].email: invalid email format
// $.company.departments[1].budget: must be >= 0
```

## Architecture

### Current Implementation

```
┌─────────────┐
│ JSON Data   │
└──────┬──────┘
       │
       ▼
┌─────────────────────────┐
│ ExtractPaths()          │  ← Extract ALL paths
│ ["$.users[0].email",    │
│  "$.users[1].email", …] │
└──────┬──────────────────┘
       │
       ▼
┌─────────────────────────┐
│ ConvertPathToSegments() │  ← Convert each path
│ For each path…          │
└──────┬──────────────────┘
       │
       ▼
┌─────────────────────────┐
│ MatchSegments()         │  ← Match against patterns
│ PatternTree traversal   │
└──────┬──────────────────┘
       │
       ▼
┌─────────────────────────┐
│ ExtractValue()          │  ← Get value for path
└──────┬──────────────────┘
       │
       ▼
┌─────────────────────────┐
│ Validate(value, schema) │  ← Validate value
└─────────────────────────┘
```

**Problems:**
- Extracts ALL paths upfront (slow for large JSON)
- Converts ALL paths to segments (allocations)
- Matches ALL paths even if no patterns apply
- Re-extracts values after matching

### Optimized Architecture

```
┌─────────────┐
│ JSON Data   │
└──────┬──────┘
       │
       ▼
┌──────────────────────────┐
│ StreamingWalk()          │  ← Stream path + value together
│ Walk JSON once           │
└──────┬───────────────────┘
       │
       ▼
┌──────────────────────────┐
│ FastPatternMatch()       │  ← Compiled pattern matcher
│ - Bloom filter (reject)  │
│ - Prefix trie (fast)     │
│ - Full match (precise)   │
└──────┬───────────────────┘
       │ (only if matches)
       ▼
┌──────────────────────────┐
│ Validate(value, schema)  │  ← Value already available!
└──────────────────────────┘
```

**Benefits:**
- Single pass through JSON
- No intermediate path storage
- Early rejection of non-matching paths
- Zero re-extraction (value available during walk)

## Performance Strategy

### Phase 3: Streaming + Fast Pattern Matching

#### 1. Streaming JSON Walker

```go
type PathValueHandler func(path string, value interface{}) error

func (v *StreamingValidator) WalkJSON(jsonData string, handler PathValueHandler) error {
    // Walk JSON once, calling handler for each path+value
    return walkJSONWithSonic(jsonData, "$", handler)
}

func (v *StreamingValidator) Validate(jsonData string) (*ValidationReport, error) {
    results := []ValidationResult{}

    v.WalkJSON(jsonData, func(path string, value interface{}) error {
        // Check if path matches any pattern (fast!)
        if schema := v.matchPattern(path); schema != nil {
            // Validate immediately (value already available)
            result := v.validateValue(path, value, schema)
            results = append(results, result)
        }
        return nil
    })

    return &ValidationReport{Results: results}, nil
}
```

**Benefits:**
- 1 pass instead of 3-4 passes
- No path storage (streaming)
- Value available immediately
- Early rejection

#### 2. Compiled Pattern Matcher

```go
type CompiledMatcher struct {
    bloomFilter  *BloomFilter      // Fast reject (99% non-matches)
    prefixTrie   *PrefixTrie       // Fast prefix check
    patternTrie  *PatternTree      // Precise matching
    patternCache map[string]int    // LRU cache
}

func (m *CompiledMatcher) Match(path string) *SchemaObject {
    // Level 1: Bloom filter (O(1), very fast)
    if !m.bloomFilter.MightContain(path) {
        return nil  // Definitely doesn't match
    }

    // Level 2: Check cache (O(1))
    if schemaID, ok := m.patternCache[path]; ok {
        return m.schemas[schemaID]
    }

    // Level 3: Prefix trie (O(k) where k = path length)
    if !m.prefixTrie.HasMatchingPrefix(path) {
        return nil
    }

    // Level 4: Full pattern match (O(n*m) but only for candidates)
    segments := ConvertPathToSegments(path)
    if schema := m.patternTrie.Match(segments); schema != nil {
        m.patternCache[path] = schema.ID  // Cache result
        return schema
    }

    return nil
}
```

#### 3. Hybrid Approach

```go
type HybridValidator struct {
    simplePatterns  map[string]*SchemaObject  // Use gjson
    complexPatterns *CompiledMatcher          // Use our engine
}

func (v *HybridValidator) Validate(jsonData string) (*ValidationReport, error) {
    results := []ValidationResult{}

    // Fast path: Use gjson for simple patterns
    for pattern, schema := range v.simplePatterns {
        gjsonPattern := convertToGJSON(pattern)
        gjson.Get(jsonData, gjsonPattern).ForEach(func(key, value gjson.Result) bool {
            result := validate(value, schema)
            results = append(results, result)
            return true
        })
    }

    // Full power: Use our engine for complex patterns
    v.WalkJSON(jsonData, func(path string, value interface{}) error {
        if schema := v.complexPatterns.Match(path); schema != nil {
            result := validate(value, schema)
            results = append(results, result)
        }
        return nil
    })

    return &ValidationReport{Results: results}, nil
}
```

## Performance Targets

| Metric | Current | Target (Phase 3) | Strategy |
|--------|---------|------------------|----------|
| **Simple patterns** | 1,733 ns | **< 800 ns** | gjson hybrid |
| **Complex patterns** | 1,733 ns | **< 1,000 ns** | Streaming + bloom filter |
| **Allocations** | 11 allocs | **< 5 allocs** | Pooling + streaming |
| **Memory** | 4,578 B | **< 2,000 B** | No path storage |

**Goal: Match or beat gjson for simple patterns, stay fast for complex patterns!**

## Use Cases

### 1. API Request Validation

```go
rules := map[string]SchemaObject{
    "$.body.email": {Type: "string", Format: "email", Required: true},
    "$.body.age": {Type: "number", Minimum: 18},
    "$.headers.Authorization": {Type: "string", Pattern: "^Bearer .*"},
}

validator := NewPathBasedValidator(rules)
errors := validator.Validate(requestJSON)
```

### 2. Configuration Validation

```go
rules := map[string]SchemaObject{
    "$.database[#*Connection].host": {Type: "string", Required: true},
    "$.database[#*Connection].port": {Type: "number", Min: 1, Max: 65535},
    "$.services[*].enabled": {Type: "boolean"},
}
```

### 3. Dynamic Schema Validation

```go
rules := map[string]SchemaObject{
    "$.data{*}.id": {Type: "string", Format: "uuid"},
    "$.data{*}.timestamp": {Type: "string", Format: "date-time"},
}
```

## Advantages Over Traditional Approaches

| Approach | Complexity | Recursion | Flexibility | Performance |
|----------|-----------|-----------|-------------|-------------|
| **JSON Schema** | High | $ref complex | Limited | Slow |
| **gjson** | Low | No | Limited | Fast |
| **Path-Based** | Low | {*} elegant | High | Can be fast! |

## Conclusion

Path-based validation is:
- ✅ **Simpler** than traditional JSON Schema
- ✅ **More flexible** than gjson
- ✅ **More powerful** for recursive structures
- ⚠️ **Can be optimized** to match gjson's speed

**With Phase 3 optimizations (streaming + bloom filter + hybrid), we can have the best of both worlds!**
