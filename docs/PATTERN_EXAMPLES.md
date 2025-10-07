# Pattern Examples: Simple vs Complex

## Quick Reference

| Pattern Type | Syntax | gjson Support | Our Support | Performance |
|-------------|--------|---------------|-------------|-------------|
| **Simple property** | `$.user.name` | ✅ Yes | ✅ Yes | gjson 2.3x faster |
| **Single array wildcard** | `$.users[*].name` | ✅ Yes | ✅ Yes | gjson 2.3x faster |
| **Repetition {*}** | `$.data{*}.name` | ❌ No | ✅ Yes | We win by default |
| **Property wildcard** | `$.config[#*service]` | ❌ No | ✅ Yes | We win by default |
| **Regex matching** | `$.users[~^user_.*]` | ❌ No | ✅ Yes | We win by default |
| **Group alternatives** | `$.user.(name\|email)` | ❌ No | ✅ Yes | We win by default |
| **Nested arrays** | `$.a[*].b[*].c[*]` | ⚠️ Limited | ✅ Yes | We're better |

## Simple Patterns

These are patterns that **both** gjson and json-schema-path can handle.

### 1. Basic Property Navigation

```javascript
// Pattern
$.user.profile.email

// JSON
{
  "user": {
    "profile": {
      "email": "alice@example.com"
    }
  }
}

// Matches
["$.user.profile.email"] → "alice@example.com"
```

**gjson**: `user.profile.email` ✅
**json-schema-path**: `$.user.profile.email` ✅

### 2. Array Index Access

```javascript
// Pattern
$.employees[0].name

// JSON
{
  "employees": [
    {"name": "Alice"},
    {"name": "Bob"}
  ]
}

// Matches
["$.employees[0].name"] → "Alice"
```

**gjson**: `employees.0.name` ✅
**json-schema-path**: `$.employees[0].name` ✅

### 3. Single Array Wildcard

```javascript
// Pattern
$.products[*].price

// JSON
{
  "products": [
    {"name": "Laptop", "price": 999},
    {"name": "Mouse", "price": 29}
  ]
}

// Matches
["$.products[0].price"] → 999
["$.products[1].price"] → 29
```

**gjson**: `products.#.price` ✅
**json-schema-path**: `$.products[*].price` ✅

**Winner**: gjson (2.3x faster) but both work!

---

## Complex Patterns

These patterns **only** json-schema-path can handle.

### 1. Repetition Operator `{*}` (Most Powerful!)

**Use case**: You don't know how deep your data is nested.

```javascript
// Pattern
$.organization{*}.name

// JSON
{
  "organization": {
    "name": "TechCorp",
    "division": {
      "name": "Engineering",
      "team": {
        "name": "Backend",
        "subteam": {
          "name": "API Team"
        }
      }
    }
  }
}

// Matches
["$.organization.name"] → "TechCorp"
["$.organization.division.name"] → "Engineering"
["$.organization.division.team.name"] → "Backend"
["$.organization.division.team.subteam.name"] → "API Team"
```

**gjson**: ❌ Cannot do this
**json-schema-path**: ✅ `$.organization{*}.name`

**Real-world example**: Finding all error messages in a deeply nested API response where you don't know the structure depth.

### 2. Multiple Nested Array Wildcards

**Use case**: Multi-level hierarchies (regions → countries → cities).

```javascript
// Pattern
$.regions[*].countries[*].offices[*].name

// JSON
{
  "regions": [
    {
      "name": "North America",
      "countries": [
        {
          "name": "USA",
          "offices": [
            {"name": "San Francisco HQ"},
            {"name": "New York Office"}
          ]
        }
      ]
    },
    {
      "name": "Europe",
      "countries": [
        {
          "name": "UK",
          "offices": [
            {"name": "London Office"}
          ]
        }
      ]
    }
  ]
}

// Matches (3 office names)
["$.regions[0].countries[0].offices[0].name"] → "San Francisco HQ"
["$.regions[0].countries[0].offices[1].name"] → "New York Office"
["$.regions[1].countries[0].offices[0].name"] → "London Office"
```

**gjson**: ⚠️ Can do `regions.#.countries.#.offices.#.name` but it's awkward
**json-schema-path**: ✅ `$.regions[*].countries[*].offices[*].name` (cleaner)

### 3. Property Wildcards `[#*pattern]`

**Use case**: Match properties by pattern when you don't know exact names.

```javascript
// Pattern
$.services[#*Service]  // All properties ENDING with "Service"

// JSON
{
  "services": {
    "authService": {"url": "https://auth.api"},
    "paymentService": {"url": "https://pay.api"},
    "notificationHub": {"url": "https://notify.api"},
    "loggingService": {"url": "https://log.api"}
  }
}

// Matches (3 services, not the Hub)
["$.services.authService"] → {"url": "https://auth.api"}
["$.services.paymentService"] → {"url": "https://pay.api"}
["$.services.loggingService"] → {"url": "https://log.api"}
```

**gjson**: ❌ Cannot match by property pattern
**json-schema-path**: ✅ `$.services[#*Service]`

**Other wildcard examples**:
- `[#admin*]` - Properties starting with "admin"
- `[#*_id]` - Properties ending with "_id"
- `[#*user*]` - Properties containing "user"

### 4. Regex Patterns `[~pattern]`

**Use case**: Complex property matching with full regex power.

```javascript
// Pattern
$.users[~^user_[0-9]+$]  // Match "user_" followed by digits

// JSON
{
  "users": {
    "user_123": {"name": "Alice"},
    "user_456": {"name": "Bob"},
    "admin_789": {"name": "Charlie"},
    "user_abc": {"name": "Invalid"}
  }
}

// Matches (only valid user_NNN patterns)
["$.users.user_123"] → {"name": "Alice"}
["$.users.user_456"] → {"name": "Bob"}
```

**gjson**: ❌ No regex support
**json-schema-path**: ✅ `$.users[~^user_[0-9]+$]`

### 5. Group Alternatives `(a|b|c)`

**Use case**: Match multiple possible property names.

```javascript
// Pattern
$.user.(firstName|name|fullName)  // Match any of these

// JSON
{
  "user": {
    "firstName": "Alice",
    "email": "alice@example.com",
    "age": 30
  }
}

// Matches
["$.user.firstName"] → "Alice"
```

**Different user object**:
```json
{
  "user": {
    "name": "Bob",
    "email": "bob@example.com"
  }
}

// Matches
["$.user.name"] → "Bob"
```

**gjson**: ❌ Would need 3 separate queries
**json-schema-path**: ✅ Single query `$.user.(firstName|name|fullName)`

### 6. Combined Complex Patterns

**The ultimate power**: Combine multiple features!

```javascript
// Pattern: Find all team member names across dynamic org structure
$.company.(divisions|departments){*}.teams[*].(members|contractors)[*].name

// This combines:
// - Group alternatives: (divisions|departments)
// - Repetition: {*}
// - Array wildcards: [*]
// - Another group: (members|contractors)

// JSON
{
  "company": {
    "divisions": [
      {
        "name": "Engineering",
        "teams": [
          {
            "name": "Backend",
            "members": [
              {"name": "Alice"},
              {"name": "Bob"}
            ],
            "contractors": [
              {"name": "Carol"}
            ]
          }
        ]
      }
    ],
    "departments": [
      {
        "teams": [
          {
            "members": [
              {"name": "Dave"}
            ]
          }
        ]
      }
    ]
  }
}

// Matches (4 people across the entire org)
["$.company.divisions[0].teams[0].members[0].name"] → "Alice"
["$.company.divisions[0].teams[0].members[1].name"] → "Bob"
["$.company.divisions[0].teams[0].contractors[0].name"] → "Carol"
["$.company.departments[0].teams[0].members[0].name"] → "Dave"
```

**gjson**: ❌ Would need 10+ separate queries and custom logic
**json-schema-path**: ✅ Single elegant query!

---

## Real-World Use Cases

### Use Case 1: Cloud Infrastructure Config

**Problem**: Extract all API endpoints from nested cloud configuration.

```javascript
// Pattern
$.services{*}.endpoints[*].url

// Handles any depth of service nesting
{
  "services": {
    "frontend": {
      "endpoints": [
        {"url": "https://app.example.com"}
      ]
    },
    "backend": {
      "api": {
        "v1": {
          "endpoints": [
            {"url": "https://api.example.com/v1"}
          ]
        }
      }
    }
  }
}
```

**gjson**: ❌ Can't handle unknown nesting depth
**json-schema-path**: ✅ Works perfectly

### Use Case 2: Monitoring All Database Connections

**Problem**: Find all database connection strings regardless of naming convention.

```javascript
// Pattern
$.config[#*Database].[#*Connection*]

// Matches various naming patterns
{
  "config": {
    "primaryDatabase": {
      "connectionString": "...",
      "connectionPool": "..."
    },
    "analyticsDatabase": {
      "dbConnection": "..."
    }
  }
}
```

**gjson**: ❌ Need to know all possible property names
**json-schema-path**: ✅ Flexible wildcard matching

### Use Case 3: Error Message Extraction

**Problem**: Find all error messages in API response, unknown structure.

```javascript
// Pattern
$.response{*}.(error|errorMessage|message)

// Works with any API structure
{
  "response": {
    "status": "error",
    "error": "Invalid request",
    "data": {
      "validation": {
        "fields": {
          "email": {
            "errorMessage": "Invalid format"
          }
        }
      }
    }
  }
}

// Finds both error messages at different depths
```

**gjson**: ❌ Would need recursive custom parsing
**json-schema-path**: ✅ One query finds all errors

---

## Performance Trade-offs

### When to Use gjson

✅ **Simple, known paths**:
- `user.profile.email`
- `products.#.price`
- When structure is well-defined
- When raw speed matters (757ns)

### When to Use json-schema-path

✅ **Complex, dynamic patterns**:
- Unknown nesting depth: `{*}`
- Property wildcards: `[#*pattern]`
- Regex matching: `[~pattern]`
- Multiple alternatives: `(a|b|c)`
- Schema validation with metadata
- When flexibility > raw speed (1,733ns is still very fast!)

---

## Summary

**Complex patterns** = Patterns that require advanced features:
- 🔄 **Repetition** `{*}` for unknown depth
- 🎯 **Property wildcards** `[#*pattern]` for pattern matching
- 🔍 **Regex** `[~pattern]` for complex matching
- 🔀 **Groups** `(a|b)` for alternatives
- 🔗 **Combinations** of all the above

**gjson is faster for simple queries, but can't handle complex patterns at all.**

**json-schema-path trades a bit of speed (2.3x slower) for immense flexibility and power.**

Choose the right tool for your use case! 🚀
