# Why We Need `{*}` Repetition Operator

## The Core Problem

This library is designed for **JSON Schema path extraction and validation**. The `{*}` operator solves a fundamental problem in JSON Schema: **recursive definitions**.

## What is a Recursive JSON Schema?

JSON Schemas often define data structures that can nest **indefinitely**. Common examples:

### 1. **File System / Tree Structures**

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "definitions": {
    "node": {
      "type": "object",
      "properties": {
        "name": {"type": "string"},
        "children": {
          "type": "array",
          "items": {"$ref": "#/definitions/node"}  ← RECURSIVE!
        }
      }
    }
  },
  "type": "object",
  "properties": {
    "root": {"$ref": "#/definitions/node"}
  }
}
```

This schema says: "A node has a name and can have children, which are also nodes, which can have children, which are also nodes..." **infinitely**.

**Valid data**:
```json
{
  "root": {
    "name": "folder1",
    "children": [
      {
        "name": "folder2",
        "children": [
          {
            "name": "folder3",
            "children": [
              {"name": "file.txt", "children": []}
            ]
          }
        ]
      }
    ]
  }
}
```

**Without `{*}`**, you'd need to write:
```bash
$.root.name                    # depth 0
$.root.children[*].name        # depth 1
$.root.children[*].children[*].name   # depth 2
$.root.children[*].children[*].children[*].name  # depth 3
# ... WHERE DO WE STOP?
```

**With `{*}`**:
```bash
$.root.(children[*]){*}.name   # ALL depths!
```

or even simpler:
```bash
$.root{*}.name                 # Find "name" at ANY depth
```

### 2. **Organization Hierarchies**

```json
{
  "definitions": {
    "department": {
      "type": "object",
      "properties": {
        "name": {"type": "string"},
        "manager": {"type": "string"},
        "subdepartments": {
          "type": "array",
          "items": {"$ref": "#/definitions/department"}  ← RECURSIVE!
        }
      }
    }
  },
  "properties": {
    "company": {"$ref": "#/definitions/department"}
  }
}
```

A department can contain subdepartments, which can contain subdepartments, indefinitely.

**With `{*}`**:
```bash
$.company.(subdepartments[*]){*}.name    # All department names at any level
$.company.(subdepartments[*]){*}.manager # All managers at any level
```

### 3. **JSON-LD / Linked Data**

```json
{
  "definitions": {
    "entity": {
      "properties": {
        "id": {"type": "string"},
        "related": {
          "type": "array",
          "items": {"$ref": "#/definitions/entity"}  ← RECURSIVE!
        }
      }
    }
  }
}
```

Entities can link to other entities, which link to others, indefinitely.

### 4. **AST / Syntax Trees**

```json
{
  "definitions": {
    "expression": {
      "oneOf": [
        {
          "type": "object",
          "properties": {
            "type": {"const": "binary"},
            "operator": {"type": "string"},
            "left": {"$ref": "#/definitions/expression"},   ← RECURSIVE!
            "right": {"$ref": "#/definitions/expression"}   ← RECURSIVE!
          }
        },
        {
          "type": "object",
          "properties": {
            "type": {"const": "literal"},
            "value": {"type": "number"}
          }
        }
      ]
    }
  }
}
```

Expressions contain sub-expressions, indefinitely.

**Example data**:
```json
{
  "type": "binary",
  "operator": "+",
  "left": {
    "type": "binary",
    "operator": "*",
    "left": {"type": "literal", "value": 2},
    "right": {"type": "literal", "value": 3}
  },
  "right": {"type": "literal", "value": 4}
}
```

**With `{*}`**:
```bash
$.{*}.value                    # Find all literal values at any depth
$.{*}[type="literal"].value    # Even with filtering
```

## The Schema Extraction Problem

When you call:
```go
paths, _ := extractor.ExtractSchemaPaths(schemaJSON)
```

For a recursive schema, what paths should it return?

### Without `{*}` - Impossible!

```
$.root.name                          ← Depth 0
$.root.children[*].name              ← Depth 1
$.root.children[*].children[*].name  ← Depth 2
... infinitely more?
```

**You can't enumerate them all** - it's infinite!

### With `{*}` - Elegant Solution!

```
$.root{*}.name                       ← Represents ALL depths!
```

The `{*}` operator says: "This pattern segment can repeat **zero or more times**", making it possible to express infinite recursive structures with **finite pattern notation**.

## Real-World JSON Schema Example

Here's an actual OpenAPI/Swagger-like schema with recursion:

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "definitions": {
    "schema": {
      "type": "object",
      "properties": {
        "type": {"type": "string"},
        "properties": {
          "type": "object",
          "additionalProperties": {"$ref": "#/definitions/schema"}
        },
        "items": {"$ref": "#/definitions/schema"},
        "allOf": {
          "type": "array",
          "items": {"$ref": "#/definitions/schema"}
        },
        "anyOf": {
          "type": "array",
          "items": {"$ref": "#/definitions/schema"}
        }
      }
    }
  },
  "type": "object",
  "properties": {
    "schema": {"$ref": "#/definitions/schema"}
  }
}
```

This describes schemas that can nest indefinitely (like JSON Schema itself!).

**Paths we want to extract**:
```bash
$.schema{*}.type                    # All type definitions at any depth
$.schema{*}.properties[*].type      # Types of all properties
$.schema.properties{*}.type         # Alternative form
```

## Why gjson Can't Do This

gjson query language doesn't have a concept of:
- Recursive/indefinite depth traversal
- Pattern repetition
- Schema-aware path generation

It's designed for **querying known data structures**, not **defining patterns over schemas with recursive definitions**.

## The Epsilon-NFA Solution

Our implementation uses **epsilon transitions** in an NFA (Non-deterministic Finite Automaton):

```
State 0: $.root
  ↓
State 1: .children[*]
  ↓ (epsilon transition loops back to State 1)
  ↓
State 2: .name (terminal)
```

The epsilon transition creates a **loop** that allows the pattern to match:
- `$.root.name` (0 repetitions)
- `$.root.children[*].name` (1 repetition)
- `$.root.children[*].children[*].name` (2 repetitions)
- ... and so on

**This is impossible without repetition operators like `{*}`!**

## Comparison with Other Path Languages

| Language | Recursive Support | Syntax |
|----------|------------------|--------|
| **JSONPath** | ❌ No | `$..name` (descendant but not pattern-based) |
| **JSONPointer** | ❌ No | `/root/children/0/children/1` (concrete paths only) |
| **XPath** | ✅ Yes | `//name` (descendant axis) |
| **jq** | ⚠️ Partial | `.. \| .name?` (recursive descent) |
| **json-schema-path** | ✅ Yes | `$.root{*}.name` (pattern-based) |

## Use Cases Summary

The `{*}` operator is essential for:

1. ✅ **JSON Schema path extraction** - Handle recursive schema definitions
2. ✅ **API response validation** - Unknown nesting depths
3. ✅ **Configuration validation** - Hierarchical configs with arbitrary depth
4. ✅ **Tree structure traversal** - File systems, DOM trees, ASTs
5. ✅ **Graph data validation** - Linked data with cycles
6. ✅ **Dynamic schema validation** - When structure depth is unknown

## Conclusion

**We needed `{*}` because:**

1. **JSON Schemas can be recursive** - They define infinitely deep structures
2. **Finite patterns for infinite data** - `{*}` expresses "repeat N times where N is unknown"
3. **Schema extraction requires it** - Can't enumerate infinite paths without repetition notation
4. **Real-world data is recursive** - Trees, graphs, hierarchies are everywhere

**Without `{*}`, this library couldn't fulfill its core purpose**: extracting and validating paths from JSON Schemas with recursive definitions.

It's not just a nice-to-have feature - **it's the reason this library exists!**

The name "json-schema-path" isn't just about JSON paths - it's specifically about **patterns that match JSON Schema structures**, and schemas are inherently recursive.

## Example: The README Use Case

From the README:
> Built for recursive data patterns with support for repetition, wildcards, and group operators.

The library was designed from the ground up to handle:
```bash
$.schema.(properties|definitions){*}.type
```

This single pattern can match:
- `$.schema.type`
- `$.schema.properties.user.type`
- `$.schema.properties.user.properties.address.type`
- `$.schema.definitions.node.properties.children.items.type`
- ... infinitely deep!

That's the power of `{*}` - and that's why we need it! 🚀
