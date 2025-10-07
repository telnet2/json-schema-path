package json

import (
	"fmt"
	"strconv"
	"strings"
	"sync"

	"github.com/bytedance/sonic"
	"github.com/bytedance/sonic/ast"

	"github.com/telnet2/json-schema-path/spec"
)

// Pool for reusing strings.Builder instances
var stringBuilderPool = sync.Pool{
	New: func() interface{} {
		return &strings.Builder{}
	},
}

// Pool for reusing segment slices
var segmentSlicePool = sync.Pool{
	New: func() interface{} {
		s := make([]spec.PathSegment, 0, 16)
		return &s
	},
}

// ProcessingError represents an error during JSON processing
type ProcessingError struct {
	Operation string
	Path      string
	Message   string
}

func (e ProcessingError) Error() string {
	if e.Path != "" {
		return fmt.Sprintf("%s at path '%s': %s", e.Operation, e.Path, e.Message)
	}
	return fmt.Sprintf("%s: %s", e.Operation, e.Message)
}

// PathExtractor extracts JSON paths from JSON data using sonic
type PathExtractor struct{}

// NewPathExtractor creates a new path extractor
func NewPathExtractor() *PathExtractor {
	return &PathExtractor{}
}

// ExtractPaths extracts all possible JSON paths from the given JSON data using AST
func (pe *PathExtractor) ExtractPaths(jsonData string) ([]string, error) {
	root, err := sonic.Get([]byte(jsonData))
	if err != nil {
		return nil, ProcessingError{
			Operation: "parsing JSON",
			Message:   err.Error(),
		}
	}

	paths := []string{}
	pe.extractPathsFromAST(&root, "$", &paths)
	return paths, nil
}

// PathValueHandler is a callback function that receives each path and its value during streaming walk
type PathValueHandler func(path string, value interface{}) error

// StreamingWalk walks through JSON data once, calling the handler for each path+value combination
// This is more efficient than ExtractPaths + ExtractValue as it avoids re-parsing and double traversal
func (pe *PathExtractor) StreamingWalk(jsonData string, handler PathValueHandler) error {
	root, err := sonic.Get([]byte(jsonData))
	if err != nil {
		return ProcessingError{
			Operation: "parsing JSON",
			Message:   err.Error(),
		}
	}

	return pe.streamingWalkAST(&root, "$", handler)
}

// streamingWalkAST recursively walks AST nodes, calling handler for each path+value
func (pe *PathExtractor) streamingWalkAST(node *ast.Node, currentPath string, handler PathValueHandler) error {
	// Call handler for this path+value
	value, err := pe.astNodeToInterface(node)
	if err != nil {
		return err
	}

	if err := handler(currentPath, value); err != nil {
		return err
	}

	switch node.Type() {
	case ast.V_OBJECT:
		if objMap, err := node.Map(); err == nil {
			builder := stringBuilderPool.Get().(*strings.Builder)
			defer stringBuilderPool.Put(builder)

			baseLen := len(currentPath)
			builder.Grow(baseLen + 50)

			for key := range objMap {
				builder.Reset()
				builder.WriteString(currentPath)
				builder.WriteByte('.')
				builder.WriteString(key)

				child := node.Get(key)
				if err := pe.streamingWalkAST(child, builder.String(), handler); err != nil {
					return err
				}
			}
		}
	case ast.V_ARRAY:
		if arraySlice, err := node.Array(); err == nil {
			builder := stringBuilderPool.Get().(*strings.Builder)
			defer stringBuilderPool.Put(builder)

			baseLen := len(currentPath)
			builder.Grow(baseLen + 20)

			for i := range arraySlice {
				builder.Reset()
				builder.WriteString(currentPath)
				builder.WriteByte('[')
				builder.WriteString(strconv.Itoa(i))
				builder.WriteByte(']')

				child := node.Index(i)
				if err := pe.streamingWalkAST(child, builder.String(), handler); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

// ExtractValue extracts a specific value from JSON data at the given path using AST
func (pe *PathExtractor) ExtractValue(jsonData string, path string) (interface{}, error) {
	root, err := sonic.Get([]byte(jsonData))
	if err != nil {
		return nil, ProcessingError{
			Operation: "parsing JSON",
			Message:   err.Error(),
		}
	}

	segments := pe.ConvertPathToSegments(path)
	current := &root

	for _, segment := range segments {
		switch current.Type() {
		case ast.V_OBJECT:
			child := current.Get(segment.Key)
			if !child.Exists() {
				return nil, ProcessingError{
					Operation: "extracting value",
					Path:      path,
					Message:   fmt.Sprintf("property '%s' not found", segment.Key),
				}
			}
			current = child
		case ast.V_ARRAY:
			if segment.Type != spec.SegmentArrayIndex {
				return nil, ProcessingError{
					Operation: "extracting value",
					Path:      path,
					Message:   fmt.Sprintf("expected array index, got property '%s'", segment.Key),
				}
			}
			child := current.Index(segment.Index)
			if !child.Exists() {
				return nil, ProcessingError{
					Operation: "extracting value",
					Path:      path,
					Message:   fmt.Sprintf("array index %d out of bounds", segment.Index),
				}
			}
			current = child
		default:
			return nil, ProcessingError{
				Operation: "extracting value",
				Path:      path,
				Message:   fmt.Sprintf("cannot navigate further: %s is not an object or array", segment.Key),
			}
		}
	}

	return pe.astNodeToInterface(current)
}

// ValidateJSON validates if a string is valid JSON using sonic AST
func (pe *PathExtractor) ValidateJSON(jsonData string) error {
	_, err := sonic.Get([]byte(jsonData))
	if err != nil {
		return ProcessingError{
			Operation: "validating JSON",
			Message:   err.Error(),
		}
	}
	return nil
}

// extractPathsFromAST recursively extracts paths from AST nodes
func (pe *PathExtractor) extractPathsFromAST(node *ast.Node, currentPath string, paths *[]string) {
	*paths = append(*paths, currentPath)

	switch node.Type() {
	case ast.V_OBJECT:
		// Use the Map() method for simpler object traversal
		if objMap, err := node.Map(); err == nil {
			// Get a builder from the pool for constructing paths
			builder := stringBuilderPool.Get().(*strings.Builder)
			defer stringBuilderPool.Put(builder)

			baseLen := len(currentPath)
			builder.Grow(baseLen + 50) // Pre-allocate reasonable capacity

			for key := range objMap {
				builder.Reset()
				builder.WriteString(currentPath)
				builder.WriteByte('.')
				builder.WriteString(key)

				child := node.Get(key)
				pe.extractPathsFromAST(child, builder.String(), paths)
			}
		}
	case ast.V_ARRAY:
		// Use direct index access for array traversal
		if arraySlice, err := node.Array(); err == nil {
			// Get a builder from the pool for constructing paths
			builder := stringBuilderPool.Get().(*strings.Builder)
			defer stringBuilderPool.Put(builder)

			baseLen := len(currentPath)
			builder.Grow(baseLen + 20) // Pre-allocate for array notation

			for i := range arraySlice {
				builder.Reset()
				builder.WriteString(currentPath)
				builder.WriteByte('[')
				builder.WriteString(strconv.Itoa(i))
				builder.WriteByte(']')

				child := node.Index(i)
				pe.extractPathsFromAST(child, builder.String(), paths)
			}
		}
	}
}

// FormatJSON formats JSON data using sonic AST for pretty printing
func (pe *PathExtractor) FormatJSON(jsonData string) (string, error) {
	root, err := sonic.Get([]byte(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to parse JSON to AST: %w", err)
	}

	// Convert AST back to formatted JSON
	formatted, err := root.MarshalJSON()
	if err != nil {
		return "", fmt.Errorf("failed to marshal AST to JSON: %w", err)
	}

	// Use sonic to format with indentation
	var data interface{}
	if err := sonic.UnmarshalString(string(formatted), &data); err != nil {
		return "", fmt.Errorf("failed to parse formatted JSON: %w", err)
	}

	indented, err := sonic.MarshalIndent(data, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to indent JSON: %w", err)
	}

	return string(indented), nil
}

// ConvertPathToSegments converts a JSON path string to segments for tree matching
// Properly handles bracket notation, array indices, and dot notation
func (pe *PathExtractor) ConvertPathToSegments(path string) []spec.PathSegment {
	// Remove leading $
	if strings.HasPrefix(path, "$") {
		path = path[1:]
	}

	if path == "" {
		return []spec.PathSegment{}
	}

	// Handle leading dot
	if strings.HasPrefix(path, ".") {
		path = path[1:]
	}

	if path == "" {
		return []spec.PathSegment{}
	}

	// Get a slice from the pool
	resultPtr := segmentSlicePool.Get().(*[]spec.PathSegment)
	result := (*resultPtr)[:0] // Reset length, keep capacity
	i := 0

	for i < len(path) {
		if path[i] == '[' {
			// Find closing bracket
			end := strings.Index(path[i:], "]")
			if end == -1 {
				// Malformed path, treat as regular character
				result = append(result, spec.NewPropertySegment(string(path[i])))
				i++
				continue
			}

			// Extract bracket content
			bracketContent := path[i+1 : i+end]

			// For array indices, convert to property-like segment
			// This aligns with how the parser handles bracket notation
			if bracketContent != "" {
				if strings.HasPrefix(bracketContent, `"`) && strings.HasSuffix(bracketContent, `"`) {
					bracketContent = bracketContent[1 : len(bracketContent)-1]
					result = append(result, spec.NewPropertySegment(unescapeJSONString(bracketContent)))
				} else if isDigits(bracketContent) {
					if index, err := strconv.Atoi(bracketContent); err == nil {
						result = append(result, spec.NewArrayIndexSegment(index))
					} else {
						result = append(result, spec.NewPropertySegment(bracketContent))
					}
				} else {
					result = append(result, spec.NewPropertySegment(unescapeBracketContent(bracketContent)))
				}
			}

			i += end + 1
		} else if path[i] == '.' {
			// Skip dots between segments
			i++
		} else {
			// Regular property name
			start := i
			for i < len(path) && path[i] != '.' && path[i] != '[' {
				i++
			}
			if start < i {
				result = append(result, spec.NewPropertySegment(path[start:i]))
			}
		}
	}

	// Make a copy since we're returning the slice and the pool will reuse it
	resultCopy := make([]spec.PathSegment, len(result))
	copy(resultCopy, result)

	// Return slice to pool
	segmentSlicePool.Put(resultPtr)

	return resultCopy
}

func isDigits(value string) bool {
	if value == "" {
		return false
	}
	for i := 0; i < len(value); i++ {
		if value[i] < '0' || value[i] > '9' {
			return false
		}
	}
	return true
}

func unescapeBracketContent(value string) string {
	var builder strings.Builder
	for i := 0; i < len(value); i++ {
		if value[i] == '\\' && i+1 < len(value) {
			i++
		}
		builder.WriteByte(value[i])
	}
	return builder.String()
}

func unescapeJSONString(value string) string {
	// The content is already without surrounding quotes but may contain escapes.
	// Use strconv.Unquote to decode by adding surrounding quotes back.
	decoded, err := strconv.Unquote("\"" + value + "\"")
	if err != nil {
		return value
	}
	return decoded
}
