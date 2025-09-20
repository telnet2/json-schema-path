package json

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/bytedance/sonic"
	"github.com/bytedance/sonic/ast"

	"github.com/telnet2/json-schema-path/spec"
)

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
			for key := range objMap {
				child := node.Get(key)
				newPath := currentPath + "." + key
				pe.extractPathsFromAST(child, newPath, paths)
			}
		}
	case ast.V_ARRAY:
		// Use direct index access for array traversal
		if arraySlice, err := node.Array(); err == nil {
			for i := range arraySlice {
				child := node.Index(i)
				newPath := currentPath + "[" + strconv.Itoa(i) + "]"
				pe.extractPathsFromAST(child, newPath, paths)
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

	result := []spec.PathSegment{}
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

	return result
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
