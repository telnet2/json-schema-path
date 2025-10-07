# Test Alignment Analysis

## The Question: Do Our Tests Align With Library Purpose?

**Library Purpose**: Extract and validate paths from **JSON Schemas** with recursive definitions.

**Our Tests**: Mixed - some align, some don't!

## ✅ Tests That Align Perfectly

### 1. JSON Schema Extraction Tests (`json/processor_test.go`)

**Test: `TestExtractSchemaPathsRecursiveSchema`**

```go
schema := `{
    "type": "object",
    "properties": {
        "value": {"type": "string"},
        "child": {"$ref": "#"}    ← RECURSIVE SCHEMA!
    }
}`

paths, _ := pe.ExtractSchemaPaths(schema)

expected := []string{
    "$",
    "$.child{*}",              ← {*} generated for recursion!
    "$.child{*}.value",
    "$.value",
}
```

**✅ PERFECT ALIGNMENT** - This tests the core purpose:
- Input: JSON Schema with `$ref` recursion
- Output: Paths with `{*}` to represent infinite depth
- Use case: Exactly what the library was designed for!

### 2. Basic Schema Extraction

```go
schema := `{
    "type": "object",
    "properties": {
        "name": {"type": "string"},
        "address": {
            "type": "object",
            "properties": {
                "street": {"type": "string"}
            }
        }
    }
}`

paths, _ := pe.ExtractSchemaPaths(schema)
// Returns: ["$", "$.address", "$.address.street", "$.name"]
```

**✅ ALIGNED** - Extracting paths from JSON Schema structure.

## ⚠️ Tests That Don't Fully Align

### 3. Validator Benchmarks (`validators/recursive_benchmark_test.go`)

**What they test:**

```go
recursiveJSON := `{
    "enterprise": {
        "regions": [
            {
                "countries": [
                    {
                        "offices": [...]
                    }
                ]
            }
        ]
    }
}`

// Testing patterns like:
"$.enterprise.regions[*].countries[*].offices[*].name"
"$.enterprise{*}.name"
```

**⚠️ MISALIGNMENT** - This is testing:
- Input: **Actual JSON data** (not a schema)
- Pattern: **Matching paths in data** (not extracting from schema)
- Use case: **Data querying** (like gjson)

**This is NOT the library's stated purpose!**

### What Should We Be Testing Instead?

Given a recursive **schema** like:
```json
{
  "type": "object",
  "properties": {
    "name": {"type": "string"},
    "regions": {
      "type": "array",
      "items": {
        "type": "object",
        "properties": {
          "name": {"type": "string"},
          "countries": {
            "type": "array",
            "items": {"$ref": "#/properties/regions/items"}  ← RECURSIVE!
          }
        }
      }
    }
  }
}
```

**We should extract paths from the schema:**
```go
paths, _ := extractor.ExtractSchemaPaths(schemaJSON)

// Should return:
[
  "$.name",
  "$.regions[*].name",
  "$.regions[*].countries{*}.name"  ← {*} for recursion!
]
```

**Then validate that actual data conforms:**
```go
validator := NewValidator(paths)
report := validator.Validate(actualDataJSON)
```

## The Gap: Two Different Use Cases

### Use Case 1: Schema Path Extraction (Library's Purpose)
```
Input:  JSON Schema with recursive definitions
Output: Pattern paths with {*}
Tool:   ExtractSchemaPaths()
```

**Example:**
```go
schema := `{"properties": {"node": {"$ref": "#/properties/node"}}}`
paths := ExtractSchemaPaths(schema)
// Returns: ["$.node{*}"]
```

### Use Case 2: Data Querying (What We're Benchmarking)
```
Input:  Actual JSON data
Input:  Query pattern (user-provided)
Output: Matching values
Tool:   Validators
```

**Example:**
```go
data := `{"company": {"divisions": [{"teams": [...]}]}}`
pattern := "$.company.divisions[*].teams[*].name"
matches := validator.Validate(data)
// Returns: All matching team names
```

**This is more like gjson's use case!**

## The Confusion

The library has **two modes**:

### Mode 1: Schema Extraction (Core Purpose)
- `ExtractSchemaPaths(schemaJSON)` → generates paths with `{*}`
- Handles recursive `$ref` definitions
- **This is unique and valuable!**

### Mode 2: Data Validation (What We're Testing)
- `Validate(dataJSON)` → matches paths in actual data
- Competes with gjson
- **This is where we're 2.3x slower!**

## Why Our Benchmarks Don't Align

Our benchmarks compare **Mode 2** (data querying) against gjson, but:

1. **gjson is designed for Mode 2** (data querying)
2. **Our library is designed for Mode 1** (schema extraction)
3. **Mode 2 is just a bonus feature**, not the core value proposition

**We're benchmarking the wrong thing!**

## What Should We Benchmark?

### Benchmark 1: Schema Extraction Performance

```go
func BenchmarkSchemaExtraction(b *testing.B) {
    schema := `{
        "definitions": {
            "node": {
                "properties": {
                    "value": {"type": "string"},
                    "children": {
                        "items": {"$ref": "#/definitions/node"}
                    }
                }
            }
        }
    }`

    extractor := NewPathExtractor()
    b.ResetTimer()

    for i := 0; i < b.N; i++ {
        paths, _ := extractor.ExtractSchemaPaths(schema)
        _ = paths
    }
}
```

**This tests our unique value!**

### Benchmark 2: Schema-Based Validation

```go
func BenchmarkSchemaValidation(b *testing.B) {
    // 1. Extract paths from schema
    schema := `...recursive schema...`
    paths, _ := extractor.ExtractSchemaPaths(schema)

    // 2. Create validator from schema paths
    validator := NewValidatorFromSchemaPaths(paths)

    // 3. Validate data against schema
    data := `...actual data...`
    b.ResetTimer()

    for i := 0; i < b.N; i++ {
        report, _ := validator.Validate(data)
        _ = report
    }
}
```

**This tests the full schema → validation workflow!**

### Benchmark 3: Compare with JSON Schema Validators

Instead of comparing with gjson (wrong comparison), compare with:
- `github.com/xeipuuv/gojsonschema`
- `github.com/santhosh-tekuri/jsonschema`

For **recursive schema handling**!

## The Truth About gjson Comparison

**gjson doesn't do what we do!**

| Feature | gjson | json-schema-path | Winner |
|---------|-------|------------------|--------|
| **Query JSON data** | ✅ 757ns | ⚠️ 1,733ns | gjson |
| **Extract paths from schema** | ❌ Can't | ✅ Yes | Us! |
| **Handle recursive schemas** | ❌ No | ✅ Yes | Us! |
| **Generate {*} patterns** | ❌ No | ✅ Yes | Us! |
| **Schema validation** | ❌ No | ✅ Yes | Us! |

**We're comparing apples (schema tool) to oranges (query tool)!**

## Recommendation: Refocus Tests

### Keep These Tests (Aligned)
✅ `TestExtractSchemaPaths` - Core purpose
✅ `TestExtractSchemaPathsRecursiveSchema` - Handles `$ref`
✅ Schema extraction with `{*}` generation

### Add These Tests (Missing!)
❌ Benchmark schema extraction performance
❌ Compare with other schema validators
❌ Test complex recursive schema patterns
❌ Benchmark schema → validator → data workflow

### Reconsider These Tests
⚠️ Data querying benchmarks vs gjson - Wrong comparison
⚠️ Validators without schema context - Not the main use case
⚠️ Pattern matching on raw data - Not unique value

## The Real Value Proposition

**What makes this library unique:**

```go
// THIS is what other libraries CAN'T do:

// 1. Take a recursive schema
schema := `{
    "properties": {
        "node": {
            "properties": {
                "value": {"type": "string"},
                "children": {
                    "items": {"$ref": "#/properties/node"}
                }
            }
        }
    }
}`

// 2. Extract patterns that represent infinite recursion
paths := ExtractSchemaPaths(schema)
// Returns: ["$.node{*}.value", "$.node{*}.children[*]"]

// 3. Validate any depth of data against the schema
validator := NewValidator(paths)
data := `{"node": {"value": "a", "children": [
    {"value": "b", "children": [
        {"value": "c", "children": []}
    ]}
]}}`

report := validator.Validate(data)
// Validates all depths correctly!
```

**gjson can't do step 2 or 3!**
**Standard JSON Schema validators can't generate patterns with {*}!**

## Conclusion

**Do our tests align?**

✅ **Yes** - Schema extraction tests (`TestExtractSchemaPaths*`)
❌ **No** - Data querying benchmarks (competing with gjson)
⚠️ **Partially** - Validators could use schema context

**What should we focus on?**

1. **Benchmark schema extraction** - Our unique strength
2. **Compare with schema validators** - Apples to apples
3. **Show the schema → pattern → validation workflow** - Full value
4. **Stop competing with gjson** - Different tools for different jobs

**The library is amazing at what it's designed for (schema extraction with `{*}`), but we're benchmarking it like a data query tool (where gjson wins)!**

Let's test what makes us special! 🎯
