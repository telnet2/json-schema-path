// Package json provides JSON processing utilities for schema-path expressions.
// It uses bytedance/sonic for high-performance JSON parsing and AST traversal,
// providing 2-3x faster parsing compared to the standard library.
//
// Key features:
//   - Path extraction from JSON documents
//   - Value extraction at specific paths
//   - JSON validation using sonic AST
//   - Efficient AST traversal without reflection overhead
package json

import (
        "fmt"

        "github.com/bytedance/sonic/ast"
)

// astNodeToInterface converts an AST node to a standard Go interface{} value
func (pe *PathExtractor) astNodeToInterface(node *ast.Node) (interface{}, error) {
        switch node.Type() {
        case ast.V_NULL:
                return nil, nil
        case ast.V_TRUE:
                return true, nil
        case ast.V_FALSE:
                return false, nil
        case ast.V_NUMBER:
                // Try to get as int64 first, fall back to float64
                if intVal, err := node.Int64(); err == nil {
                        return intVal, nil
                }
                return node.Float64()
        case ast.V_STRING:
                str, err := node.String()
                return str, err
        case ast.V_ARRAY:
                arrayVal, err := node.Array()
                if err != nil {
                        return nil, err
                }
                return arrayVal, nil
        case ast.V_OBJECT:
                objVal, err := node.Map()
                if err != nil {
                        return nil, err
                }
                return objVal, nil
        default:
                return nil, fmt.Errorf("unsupported AST node type: %v", node.Type())
        }
}

// GetASTNodeAtPath traverses the AST to find the node at the specified path segments
func (pe *PathExtractor) GetASTNodeAtPath(root *ast.Node, segments []string) (*ast.Node, error) {
        current := root
        
        for _, segment := range segments {
                switch current.Type() {
                case ast.V_OBJECT:
                        child := current.Get(segment)
                        if !child.Exists() {
                                return nil, fmt.Errorf("property '%s' not found", segment)
                        }
                        current = child
                case ast.V_ARRAY:
                        // Handle numeric array index
                        if index, err := parseArrayIndex(segment); err == nil {
                                child := current.Index(index)
                                if !child.Exists() {
                                        return nil, fmt.Errorf("array index %d out of bounds", index)
                                }
                                current = child
                        } else {
                                return nil, fmt.Errorf("invalid array index '%s'", segment)
                        }
                default:
                        return nil, fmt.Errorf("cannot navigate further from node type %v with segment '%s'", current.Type(), segment)
                }
        }
        
        return current, nil
}

// parseArrayIndex safely parses a string to an array index.
// It handles bracket notation by stripping brackets if present,
// then converts the remaining content to an integer.
func parseArrayIndex(segment string) (int, error) {
        // Handle empty segment
        if len(segment) == 0 {
                return -1, fmt.Errorf("empty segment")
        }

        // Handle bracket notation by stripping brackets if present
        if len(segment) >= 2 && segment[0] == '[' && segment[len(segment)-1] == ']' {
                segment = segment[1 : len(segment)-1]
        }

        // Handle empty content after stripping brackets
        if len(segment) == 0 {
                return -1, fmt.Errorf("empty index")
        }

        // Convert to integer using manual parsing for efficiency
        index := 0
        for i, char := range segment {
                if char < '0' || char > '9' {
                        return -1, fmt.Errorf("non-numeric character '%c' at position %d", char, i)
                }
                index = index*10 + int(char-'0')
        }
        return index, nil
}

// TraverseASTWithCallback traverses the entire AST tree and calls the callback for each node.
// It uses efficient native AST navigation via sonic's Get() and Index() methods,
// avoiding costly interface{} to string conversions.
func (pe *PathExtractor) TraverseASTWithCallback(node *ast.Node, path string, callback func(string, *ast.Node)) {
        callback(path, node)

        switch node.Type() {
        case ast.V_OBJECT:
                // Use MapUseNode() for efficient object traversal with direct node access
                objMap, err := node.MapUseNode()
                if err != nil {
                        return // Skip object if map cannot be retrieved
                }
                for key, childNode := range objMap {
                        newPath := path + "." + key
                        pe.TraverseASTWithCallback(&childNode, newPath, callback)
                }
        case ast.V_ARRAY:
                // Use Len() and Index() for efficient array traversal
                length, err := node.Len()
                if err != nil {
                        return // Skip array if length cannot be determined
                }
                for i := 0; i < length; i++ {
                        child := node.Index(i)
                        if child.Exists() {
                                newPath := fmt.Sprintf("%s[%d]", path, i)
                                pe.TraverseASTWithCallback(child, newPath, callback)
                        }
                }
        }
}