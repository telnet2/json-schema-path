package json

import (
        "fmt"
        "strconv"
        "strings"

        "github.com/bytedance/sonic"
)

// PathExtractor extracts JSON paths from JSON data using sonic
type PathExtractor struct{}

// NewPathExtractor creates a new path extractor
func NewPathExtractor() *PathExtractor {
        return &PathExtractor{}
}

// ExtractPaths extracts all possible JSON paths from the given JSON data
func (pe *PathExtractor) ExtractPaths(jsonData string) ([]string, error) {
        var data interface{}
        if err := sonic.UnmarshalString(jsonData, &data); err != nil {
                return nil, fmt.Errorf("failed to parse JSON: %w", err)
        }

        paths := []string{}
        pe.extractPathsRecursive(data, "$", &paths)
        return paths, nil
}

// ExtractValue extracts a specific value from JSON data at the given path
func (pe *PathExtractor) ExtractValue(jsonData string, path string) (interface{}, error) {
        var data interface{}
        if err := sonic.UnmarshalString(jsonData, &data); err != nil {
                return nil, fmt.Errorf("failed to parse JSON: %w", err)
        }

        // Convert path to segments and navigate
        segments := pe.ConvertPathToSegments(path)
        current := data
        
        for _, segment := range segments {
                switch v := current.(type) {
                case map[string]interface{}:
                        if val, exists := v[segment]; exists {
                                current = val
                        } else {
                                return nil, fmt.Errorf("property '%s' not found", segment)
                        }
                case []interface{}:
                        // Handle array index
                        if index, err := strconv.Atoi(segment); err == nil && index >= 0 && index < len(v) {
                                current = v[index]
                        } else {
                                return nil, fmt.Errorf("invalid array index '%s'", segment)
                        }
                default:
                        return nil, fmt.Errorf("cannot navigate further: %s is not an object or array", segment)
                }
        }
        
        return current, nil
}

// ValidateJSON validates if a string is valid JSON using sonic
func (pe *PathExtractor) ValidateJSON(jsonData string) error {
        var temp interface{}
        return sonic.UnmarshalString(jsonData, &temp)
}

// extractPathsRecursive recursively extracts paths from JSON data
func (pe *PathExtractor) extractPathsRecursive(data interface{}, currentPath string, paths *[]string) {
        *paths = append(*paths, currentPath)

        switch v := data.(type) {
        case map[string]interface{}:
                for key, value := range v {
                        newPath := currentPath + "." + key
                        pe.extractPathsRecursive(value, newPath, paths)
                }
        case []interface{}:
                for i, value := range v {
                        newPath := currentPath + "[" + strconv.Itoa(i) + "]"
                        pe.extractPathsRecursive(value, newPath, paths)
                }
        }
}

// FormatJSON formats JSON data using sonic for pretty printing
func (pe *PathExtractor) FormatJSON(jsonData string) (string, error) {
        var data interface{}
        if err := sonic.UnmarshalString(jsonData, &data); err != nil {
                return "", fmt.Errorf("failed to parse JSON: %w", err)
        }

        formatted, err := sonic.MarshalIndent(data, "", "  ")
        if err != nil {
                return "", fmt.Errorf("failed to format JSON: %w", err)
        }

        return string(formatted), nil
}

// ConvertPathToSegments converts a JSON path string to segments for tree matching
// Properly handles bracket notation, array indices, and dot notation
func (pe *PathExtractor) ConvertPathToSegments(path string) []string {
        // Remove leading $ 
        if strings.HasPrefix(path, "$") {
                path = path[1:]
        }
        
        if path == "" {
                return []string{}
        }
        
        // Handle leading dot
        if strings.HasPrefix(path, ".") {
                path = path[1:]
        }
        
        if path == "" {
                return []string{}
        }
        
        result := []string{}
        i := 0
        
        for i < len(path) {
                if path[i] == '[' {
                        // Find closing bracket
                        end := strings.Index(path[i:], "]")
                        if end == -1 {
                                // Malformed path, treat as regular character
                                result = append(result, string(path[i]))
                                i++
                                continue
                        }
                        
                        // Extract bracket content
                        bracketContent := path[i+1 : i+end]
                        
                        // For array indices, convert to property-like segment
                        // This aligns with how the parser handles bracket notation
                        if bracketContent != "" {
                                // Remove quotes if present and treat as property name
                                if strings.HasPrefix(bracketContent, `"`) && strings.HasSuffix(bracketContent, `"`) {
                                        bracketContent = bracketContent[1 : len(bracketContent)-1]
                                }
                                result = append(result, bracketContent)
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
                                result = append(result, path[start:i])
                        }
                }
        }
        
        return result
}