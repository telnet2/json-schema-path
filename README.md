# json-schema-path

[![Go Version](https://img.shields.io/badge/go-1.22+-00ADD8?style=flat&logo=go)](https://go.dev/)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/telnet2/json-schema-path)](https://goreportcard.com/report/github.com/telnet2/json-schema-path)

A high-performance Go library and CLI tool for navigating and querying JSON schema structures using advanced path expressions. Built for recursive data patterns with support for repetition, wildcards, and group operators.

## 🌟 Features

- **🚀 Blazing Fast**: Powered by [bytedance/sonic](https://github.com/bytedance/sonic) for native JSON AST parsing
- **🔄 Recursive Navigation**: Zero-or-more repetition with `{*}` for deep traversal patterns
- **🎯 Group Operators**: Alternative path matching with `|` for flexible queries
- **🔧 Advanced Bracket Notation**: Support for wildcards `[*]`, regex `[~pattern]`, and property wildcards `[#*key]`
- **📊 Efficient Matching**: Trie-based pattern matching for optimal performance
- **💻 Dual Interface**: Both command-line tool and Go SDK
- **✅ Production Ready**: Comprehensive test suite with 100% path expression coverage

## 📦 Installation

### CLI Tool

```bash
go install github.com/telnet2/json-schema-path/cmd/schemapath@latest
```

### Go Library

```bash
go get github.com/telnet2/json-schema-path
```

### From Source

```bash
git clone https://github.com/telnet2/json-schema-path.git
cd json-schema-path
go build ./cmd/schemapath
```

## 🚀 Quick Start

### Command Line

```bash
# Parse and validate expressions
schemapath parse "$.children[*]{*}.name"

# Test against JSON data
schemapath test "$.users[*].profile.email" '{"users":[{"profile":{"email":"user@example.com"}}]}'

# Extract from files
schemapath extract "$.products[*].price" data.json
```

### Go SDK

```go
package main

import (
    "fmt"
    "log"
    "github.com/telnet2/json-schema-path/parser"
    "github.com/telnet2/json-schema-path/json"
    "github.com/telnet2/json-schema-path/tree"
)

func main() {
    // Parse expression
    expr, err := parser.ParseExpression("$.node.(child|meta.child){*}.value")
    if err != nil {
        log.Fatal(err)
    }
    
    // Create pattern matcher
    patternTree := tree.NewPatternTree()
    patternTree.AddPattern(expr)
    
    // Process JSON
    processor := json.NewPathExtractor()
    jsonData := `{"node": {"child": {"value": 42}}}`
    
    paths, _ := processor.ExtractPaths(jsonData)
    for _, path := range paths {
        segments := processor.ConvertPathToSegments(path)
        if patternTree.MatchSegments(segments) {
            value, _ := processor.ExtractValue(jsonData, path)
            fmt.Printf("Match: %s = %v\n", path, value)
        }
    }
}
```

## 📖 Expression Syntax

### Basic Navigation

| Pattern | Description | Example |
|---------|-------------|---------|
| `$` | Root of JSON document | `$` |
| `.property` | Object property access | `$.user.name` |
| `["key"]` | Bracket notation | `$.data["api-key"]` |
| `[*]` | Array wildcard | `$.items[*]` |

### Advanced Features

#### Repetition Patterns
```bash
$.meta{*}.child           # Zero or more .meta hops
$.tree.(left|right){*}    # Recursive tree traversal
$.node.children{*}.name   # Deep nested search
```

#### Group Operators
```bash
$.user.(name|email)                    # Match either property
$.data.(items|products|services){*}    # Multiple alternatives
$.config.(api["version"]|version)     # Mixed notation
```

#### Bracket Selectors
```bash
$.items[#*service]        # Properties ending with "service"
$.fields[~^user_.*]       # Regex pattern matching
$.data["quoted-key"]     # Quoted property names
$.array[0][*]             # Array index + wildcard
```

### Complex Examples

```bash
# Deep recursive search with alternatives
$.root.(items["subitems"]|nested.values){*}.id

# Mixed patterns with repetition
$.company.employees[*].(skills|certificates){*}.name

# Schema validation patterns
$.schema.(properties|definitions){*}.type
```

## 🎯 Real-World Examples

### E-Commerce Product Catalog

```json
{
  "store": {
    "products": [
      {
        "id": 1,
        "name": "Laptop",
        "variants": [
          {"color": "black", "price": 999},
          {"color": "silver", "price": 1099}
        ]
      }
    ]
  }
}
```

```bash
# Extract all product names
schemapath extract "$.store.products[*].name" catalog.json

# Get all variant prices
schemapath extract "$.store.products[*].variants[*].price" catalog.json

# Find specific color variants
schemapath test "$.store.products[*].variants[?color='black']" @catalog.json
```

### Organization Hierarchy

```json
{
  "company": {
    "ceo": {
      "name": "Alice",
      "reports": [
        {
          "name": "Bob",
          "department": "Engineering",
          "reports": [
            {"name": "Charlie", "role": "Senior Dev"}
          ]
        }
      ]
    }
  }
}
```

```bash
# Find all employee names recursively
schemapath extract "$.company.(ceo|reports){*}.name" org.json

# Get all departments in hierarchy
schemapath extract "$.company.(ceo.reports|reports){*}.department" org.json
```

### Configuration Management

```json
{
  "app": {
    "database": {"host": "localhost", "port": 5432},
    "cache": {"redis": {"host": "redis.local"}},
    "features": {"feature-flags": {"new-ui": true}}
  }
}
```

```bash
# Extract database configuration
schemapath extract "$.app.database.(host|port)" config.json

# Get feature flags with escaping
schemapath extract "$.app.features[\"feature-flags\"]" config.json
```

## ⚡ Performance

Built for high-performance JSON processing with multiple optimization strategies:

- **Native AST Parsing**: Direct JSON-to-AST conversion eliminates double parsing
- **Hybrid Strategy**: Uses gjson for simple patterns, our engine for complex ones
- **Streaming Walk**: Single-pass JSON traversal with immediate value extraction
- **Bloom Filters**: Fast rejection of non-matching paths (O(1) lookups)
- **LRU Caching**: 80-90% cache hit rate for repeated patterns
- **Memory Efficient**: 450x less memory than gjson on large JSON

**HybridValidator** (Phase 3) achieves:
- **Small JSON (<1KB)**: 1,130 ns/op (2.77x slower than gjson, but handles all patterns)
- **Large JSON (>1MB)**: 55.8 ms/op (**1.35x FASTER than gjson!** 🎉)
- **Complex patterns**: Only we can handle `{*}`, `[#*pattern]`, `[~regex]`, `(a|b)`
- **Multi-pattern**: 2.8 μs for 4 patterns on small JSON

See detailed benchmarks:
- [Phase 3 Results](docs/PHASE3_RESULTS.md) - Streaming + Bloom + Hybrid (beats gjson!)
- [Phase 1 Results](docs/PHASE1_RESULTS.md) - Cached patterns (1.41x speedup)
- [Phase 2 Results](docs/PHASE2_RESULTS.md) - Memory pooling experiments
- [gjson Comparison](docs/GJSON_COMPARISON.md) - Head-to-head comparison

## 🏗️ Architecture

```
json-schema-path/
├── cmd/schemapath/         # CLI application
├── docs/                   # Documentation and benchmarks
├── json/                   # JSON processing with sonic/AST
├── parser/                 # Expression parser & lexer
├── spec/                   # Grammar specification & AST nodes
├── tree/                   # Pattern matching trie implementation
├── validators/             # Validator implementations
└── schema_test.go          # Integration tests
```

## 📚 Documentation

### Core Concepts
- [Grammar Specification](spec/SPECIFICATION.md) - Formal EBNF grammar and language semantics
- [Why Repetition `{*}`?](docs/WHY_REPETITION.md) - Understanding recursive schemas and the need for `{*}`
- [Pattern Examples](docs/PATTERN_EXAMPLES.md) - Simple vs complex patterns explained
- [Path-Based Validation](docs/PATH_BASED_VALIDATION.md) - Validation architecture and use cases

### Performance
- **[Phase 3 Results](docs/PHASE3_RESULTS.md)** - Streaming + Bloom + Hybrid (**We beat gjson!**)
- [Phase 1 Results](docs/PHASE1_RESULTS.md) - Cached patterns (1.41x speedup)
- [Phase 2 Results](docs/PHASE2_RESULTS.md) - Memory pooling experiments
- [Performance Benchmarks](docs/BENCHMARKS.md) - Comprehensive analysis
- [Optimization Guide](docs/OPTIMIZATION_GUIDE.md) - Optimization strategies
- [gjson Comparison](docs/GJSON_COMPARISON.md) - Head-to-head comparison

### Development
- [Specification Analysis](docs/SPECIFICATION_ANALYSIS.md) - Parser compliance
- [Test Alignment](docs/TEST_ALIGNMENT.md) - Test coverage analysis
- [Contributing Guide](CONTRIBUTING.md) - How to contribute

## 🧪 Testing

Comprehensive test coverage across all components:

```bash
# Run all tests
go test ./...

# Run with coverage
go test -cover ./...

# Run benchmarks
go test -bench=. ./...

# Test specific components
go test ./parser -v
go test ./json -v
go test ./tree -v
```

## 🤝 Contributing

We welcome contributions! Please see our [Contributing Guide](CONTRIBUTING.md) for details on:
- Development setup and workflow
- Code style and testing guidelines
- Submitting issues and pull requests
- Project structure overview

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- [bytedance/sonic](https://github.com/bytedance/sonic) - High-performance JSON library
- JSONPath specification and related standards
- The Go community for excellent tooling and libraries

## 📞 Support

- **Issues**: [GitHub Issues](https://github.com/telnet2/json-schema-path/issues)
- **Discussions**: [GitHub Discussions](https://github.com/telnet2/json-schema-path/discussions)

---

**⭐ Star this repository if you find it useful!**