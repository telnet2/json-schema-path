# JSONPath SDK

[![Go Version](https://img.shields.io/badge/go-1.22+-00ADD8?style=flat&logo=go)](https://go.dev/)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/yourusername/jsonpath-sdk)](https://goreportcard.com/report/github.com/yourusername/jsonpath-sdk)
[![Build Status](https://img.shields.io/github/workflow/status/yourusername/jsonpath-sdk/CI)](https://github.com/yourusername/jsonpath-sdk/actions)

A high-performance Golang SDK and command-line utility for JSON path expressions with advanced support for recursive structures, group operators, and repetition patterns. Built with [bytedance/sonic](https://github.com/bytedance/sonic) for blazing-fast JSON processing and AST parsing.

## ✨ Key Features

- 🚀 **High Performance**: Powered by bytedance/sonic with native AST parsing
- 🔄 **Recursive Structures**: Support for complex recursive JSON patterns with `{*}` repetition
- 🎯 **Group Operators**: Alternative path matching with `|` operators in group expressions
- 🔧 **Bracket Notation**: Advanced bracket notation with proper escape sequence handling  
- 📊 **Trie Pattern Matching**: Efficient pattern matching using trie/radix tree structures
- 💻 **CLI & SDK**: Both command-line utility and programmatic Go SDK
- ✅ **Comprehensive Testing**: Extensive test coverage with integration testing

## 📖 Table of Contents

- [Installation](#-installation)
- [Quick Start](#-quick-start)
- [CLI Usage](#-cli-usage)
- [SDK Usage](#-sdk-usage)  
- [Path Expression Syntax](#-path-expression-syntax)
- [Examples](#-examples)
- [Performance](#-performance)
- [Project Structure](#-project-structure)
- [Contributing](#-contributing)
- [License](#-license)

## 🚀 Installation

### Using Go Install

```bash
go install github.com/yourusername/jsonpath-sdk/cmd/jsonpath@latest
```

### From Source

```bash
git clone https://github.com/yourusername/jsonpath-sdk.git
cd jsonpath-sdk
go build ./cmd/jsonpath
```

### As Go Module

```bash
go get github.com/yourusername/jsonpath-sdk
```

## ⚡ Quick Start

### Command Line

```bash
# Parse and validate a path expression
jsonpath parse "$.user.(name|email)"

# Test path against JSON data
jsonpath test "$.users[*].name" '{"users":[{"name":"Alice"},{"name":"Bob"}]}'

# Extract values from JSON file
jsonpath extract "$.data.items[*].value" data.json

# Validate JSON format
jsonpath validate '{"valid": "json"}'
```

### Go SDK

```go
package main

import (
    "fmt"
    "log"
    "jsonpath-sdk/internal/json"
    "jsonpath-sdk/internal/parser"
    "jsonpath-sdk/internal/tree"
)

func main() {
    // JSON data
    jsonData := `{"user": {"name": "John", "profile": {"email": "john@test.com"}}}`
    
    // Parse path expression
    expr, err := parser.ParseExpression("$.user.(name|profile.email)")
    if err != nil {
        log.Fatal(err)
    }
    
    // Create pattern tree and extract paths
    tree := tree.NewPatternTree()
    tree.AddPattern(expr)
    
    processor := json.NewPathExtractor()
    paths, _ := processor.ExtractPaths(jsonData)
    
    // Test matches
    for _, path := range paths {
        segments := processor.ConvertPathToSegments(path)
        if tree.MatchPath(segments) {
            value, _ := processor.ExtractValue(jsonData, path)
            fmt.Printf("Match: %s = %v\\n", path, value)
        }
    }
}
```

## 💻 CLI Usage

The `jsonpath` CLI provides four main commands:

### Parse Command

Parse and analyze path expressions:

```bash
jsonpath parse "$.node.(child|meta.child){*}.value"
# Output: Parsed structure with segments and validation

# JSON output format
jsonpath parse --json --pretty "$.user.profile.settings"
```

### Test Command  

Test path expressions against JSON data:

```bash
# Test with inline JSON
jsonpath test "$.user.name" '{"user": {"name": "John"}}'

# Test with JSON file
jsonpath test "$.users[*].active" @users.json

# Verbose output showing all paths
jsonpath test --verbose "$.data.items[*]" '{"data":{"items":[1,2,3]}}'
```

### Extract Command

Extract matching values from JSON:

```bash
# Extract from file
jsonpath extract "$.products[*].price" catalog.json

# JSON output format
jsonpath extract --json "$.users[*].name" users.json
```

### Validate Command

Validate JSON format:

```bash
# Validate JSON string
jsonpath validate '{"valid": true}'

# Validate JSON file  
jsonpath validate @data.json

# Show formatted output
jsonpath validate --verbose '{"compact":true}'
```

### Global Flags

- `--verbose, -v`: Enable detailed output
- `--quiet, -q`: Minimal output mode
- `--json, -j`: Output results in JSON format
- `--pretty, -p`: Pretty print JSON output

## 🔧 SDK Usage

### Core Components

#### JSON Processor

```go
import "jsonpath-sdk/internal/json"

processor := json.NewPathExtractor()

// Validate JSON
err := processor.ValidateJSON(jsonData)

// Extract all paths from JSON
paths, err := processor.ExtractPaths(jsonData)

// Extract specific value
value, err := processor.ExtractValue(jsonData, "$.user.name")

// Format JSON with indentation
formatted, err := processor.FormatJSON(jsonData)
```

#### Path Expression Parser

```go
import "jsonpath-sdk/internal/parser"

// Parse expression into AST
expr, err := parser.ParseExpression("$.node.(child|meta){*}")

// Access parsed components
fmt.Println(expr.Root.String())    // "$"
fmt.Println(len(expr.Segments))    // Number of segments
```

#### Pattern Tree Matching

```go
import "jsonpath-sdk/internal/tree"

// Create pattern tree
tree := tree.NewPatternTree()

// Add patterns
err := tree.AddPattern(expr)

// Match paths
segments := processor.ConvertPathToSegments("$.node.child.value")
matches := tree.MatchPath(segments)
```

## 📝 Path Expression Syntax

### Basic Syntax

| Pattern | Description | Example |
|---------|-------------|---------|
| `$` | Root of JSON document | `$` |
| `.property` | Object property access | `$.user.name` |
| `[key]` | Bracket notation | `$.data[property]` |
| `["quoted"]` | Quoted bracket notation | `$.data["api-key"]` |

### Advanced Features

#### Group Expressions
```bash
$.user.(name|email)           # Match either name OR email
$.data.(items|products)       # Alternative object properties
$.node.(child|meta.child)     # Nested alternatives
```

#### Repetition Patterns
```bash
$.tree.(left|right){*}        # Recursive tree traversal
$.node.(child|meta.child){*}.value  # Deep recursive search
```

#### Bracket Notation with Escaping
```bash
$.data["quoted-key"]          # Quoted property names
$.data["\"escaped"]           # Escaped quotes in property names
$.config[api-key]             # Unquoted bracket notation
```

#### Complex Expressions
```bash
# Recursive structure with alternatives and bracket notation
$.root.(items["key"]["subkey"]|nested.values){*}

# Mixed property access patterns  
$.data.(user.profile["settings"]|config["user-prefs"]){*}.theme
```

### Formal Grammar (EBNF)

```ebnf
Expression      ::= Root Path?
Root            ::= "$"
Path            ::= Segment*
Segment         ::= "." SegmentItem | BracketNotation
SegmentItem     ::= Identifier | GroupExpression
GroupExpression ::= "(" GroupSeq ("|" GroupSeq)* ")" Repetition?
GroupSeq        ::= GroupPrimary ("." GroupPrimary)*
Repetition      ::= "{*}"
BracketNotation ::= "[" BracketContent "]"
BracketContent  ::= QuotedString | UnquotedString
```

## 📚 Examples

### E-Commerce Product Catalog

```json
{
  "store": {
    "products": [
      {
        "id": 1,
        "name": "Laptop",
        "specs": {"cpu": "Intel", "ram": "16GB"},
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
jsonpath extract "$.store.products[*].name" catalog.json

# Get all variant prices  
jsonpath extract "$.store.products[*].variants[*].price" catalog.json

# Find CPU specs using bracket notation
jsonpath extract "$.store.products[*].specs[\"cpu\"]" catalog.json
```

### Recursive Organization Structure  

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
jsonpath extract "$.company.(ceo|reports){*}.name" org.json

# Get all departments in hierarchy
jsonpath extract "$.company.(ceo.reports|reports){*}.department" org.json
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
jsonpath extract "$.app.database.(host|port)" config.json

# Get feature flag values with escaping
jsonpath extract "$.app.features[\"feature-flags\"]" config.json
```

## ⚡ Performance

### Benchmarks

The SDK leverages bytedance/sonic for high-performance JSON processing:

- **JSON Parsing**: Up to 2-3x faster than standard library
- **AST Navigation**: Direct node traversal without reflection overhead  
- **Pattern Matching**: Efficient trie-based matching for complex expressions
- **Memory Usage**: Reduced allocations through AST node reuse

### Performance Features

- **Native AST Parsing**: Direct JSON-to-AST conversion eliminates double parsing
- **Efficient Tree Structures**: Trie/radix trees for pattern matching  
- **Zero-Copy Operations**: Minimal string allocations during parsing
- **Streaming Support**: Process large JSON documents efficiently

Run benchmarks locally:

```bash
go test -bench=. ./internal/benchmarks/
```

## 📁 Project Structure

```
jsonpath-sdk/
├── cmd/
│   └── jsonpath/           # CLI application
│       └── main.go
├── internal/
│   ├── benchmarks/         # Performance benchmarks
│   ├── cli/                # CLI-specific logic  
│   ├── integration/        # Integration tests
│   ├── json/              # JSON processing with sonic/AST
│   │   ├── processor.go   # Main JSON processor
│   │   └── ast_helpers.go # AST helper functions
│   ├── parser/            # Path expression parser
│   │   ├── lexer.go       # Lexical analysis
│   │   └── parser.go      # Syntax analysis & AST building
│   ├── spec/              # Formal specification
│   │   └── specification.go
│   └── tree/              # Pattern tree implementation
│       └── tree.go        # Trie/radix tree matching
├── pkg/
│   └── jsonpath/          # Public SDK interfaces (planned)
├── go.mod                 # Go module definition
└── go.sum                 # Dependency checksums
```

## 🧪 Testing

### Run All Tests

```bash
# Run all tests with coverage
go test -v -cover ./...

# Run integration tests  
go test -v ./internal/integration/

# Run benchmarks
go test -bench=. ./internal/benchmarks/
```

### Test Categories

- **Unit Tests**: Individual component testing  
- **Integration Tests**: End-to-end pipeline testing
- **Performance Tests**: Benchmarking and performance validation
- **Parser Tests**: Expression parsing and validation
- **JSON Tests**: AST processing and path extraction

## 🛠 Development

### Prerequisites

- Go 1.22 or higher
- Git

### Setup Development Environment

```bash
git clone https://github.com/yourusername/jsonpath-sdk.git
cd jsonpath-sdk

# Install dependencies
go mod tidy

# Run tests
go test ./...

# Build CLI
go build -o jsonpath ./cmd/jsonpath
```

### Code Style

- Follow standard Go conventions (`go fmt`, `go vet`)
- Write comprehensive tests for new features
- Update documentation for API changes
- Add benchmarks for performance-critical code

## 🤝 Contributing

We welcome contributions! Here's how to get started:

### Reporting Issues

1. Check existing issues before creating new ones
2. Use issue templates when available  
3. Provide minimal reproduction examples
4. Include system information (Go version, OS)

### Pull Requests

1. Fork the repository
2. Create a feature branch: `git checkout -b feature/amazing-feature`
3. Make your changes with tests
4. Run all tests: `go test ./...`
5. Commit changes: `git commit -m 'Add amazing feature'`  
6. Push branch: `git push origin feature/amazing-feature`
7. Open a Pull Request

### Development Guidelines

- **Code Quality**: Maintain test coverage above 80%
- **Performance**: Add benchmarks for new features
- **Documentation**: Update README and godoc comments
- **Backward Compatibility**: Avoid breaking changes in public APIs

## 📜 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- [bytedance/sonic](https://github.com/bytedance/sonic) - High-performance JSON library
- [spf13/cobra](https://github.com/spf13/cobra) - CLI framework
- JSONPath specification and related standards

## 📞 Support

- **Issues**: [GitHub Issues](https://github.com/yourusername/jsonpath-sdk/issues)
- **Discussions**: [GitHub Discussions](https://github.com/yourusername/jsonpath-sdk/discussions)  
- **Documentation**: [Wiki](https://github.com/yourusername/jsonpath-sdk/wiki)

---

⭐ **Star this repository if you find it useful!**

Built with ❤️ using Go and high-performance JSON processing.