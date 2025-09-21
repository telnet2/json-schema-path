package validators

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	jsonpkg "github.com/telnet2/json-schema-path/json"
)

// OptimizedValidator provides optimized path validation with wildcard support
type OptimizedValidator struct {
	config         *SimpleValidatorConfig
	exactPaths     map[string]bool
	wildcardCache  map[string][]string
}

// NewOptimizedValidator creates an optimized validator with wildcard support
func NewOptimizedValidator(config *SimpleValidatorConfig) (*OptimizedValidator, error) {
	if config == nil {
		return nil, fmt.Errorf("configuration cannot be nil")
	}
	
	exactPaths := make(map[string]bool)
	wildcardCache := make(map[string][]string)
	
	// Pre-process paths for optimization
	for path := range config.Paths {
		exactPaths[path] = true
		
		// Handle wildcard patterns
		if strings.Contains(path, "[*]") {
			// Store wildcard patterns for fast matching
			basePath := strings.ReplaceAll(path, "[*]", "")
			wildcardCache[basePath] = append(wildcardCache[basePath], path)
		}
	}
	
	return &OptimizedValidator{
		config:        config,
		exactPaths:    exactPaths,
		wildcardCache: wildcardCache,
	}, nil
}

// NewOptimizedValidatorFromJSON creates an optimized validator from JSON schema data
func NewOptimizedValidatorFromJSON(schemaJSON string) (*OptimizedValidator, error) {
	processor := jsonpkg.NewPathExtractor()
	paths, err := processor.ExtractPaths(schemaJSON)
	if err != nil {
		return nil, fmt.Errorf("failed to extract schema paths: %w", err)
	}
	
	config := NewSimpleValidatorConfig("optimized_validator")
	config.AddPaths(paths)
	
	return NewOptimizedValidator(config)
}

// ValidatePath checks if a path exists using optimized matching
func (v *OptimizedValidator) ValidatePath(path string) bool {
	// Fast path: exact match
	if v.exactPaths[path] {
		return true
	}
	
	// Check wildcard patterns
	if strings.Contains(path, "[") {
		// Try to match against pre-computed wildcard patterns
		wildcardPath := convertToWildcardPattern(path)
		if v.exactPaths[wildcardPath] {
			return true
		}
		
		// Check base path matches
		for basePath, patterns := range v.wildcardCache {
			if strings.HasPrefix(path, basePath) {
				for _, pattern := range patterns {
					if matchesWildcardPattern(path, pattern) {
						return true
					}
				}
			}
		}
	}
	
	return false
}

// GetSupportedPaths returns all available paths
func (v *OptimizedValidator) GetSupportedPaths() []string {
	paths := make([]string, 0, len(v.exactPaths))
	for path := range v.exactPaths {
		paths = append(paths, path)
	}
	return paths
}

// GetConfig returns the validator configuration (implements UnifiedValidator)
func (v *OptimizedValidator) GetConfig() *ValidatorConfig {
	// Convert SimpleValidatorConfig to ValidatorConfig for interface compatibility
	paths := make(map[string]json.RawMessage)
	for path := range v.config.Paths {
		paths[path] = json.RawMessage("{}")
	}
	return &ValidatorConfig{
		Name:        v.config.Name,
		Description: v.config.Description,
		Paths:       paths,
	}
}

// GetName returns the validator name
func (v *OptimizedValidator) GetName() string {
	return v.config.Name
}

// Validate performs full validation with optimization
func (v *OptimizedValidator) Validate(jsonData string) (*ValidationReport, error) {
	processor := jsonpkg.NewPathExtractor()
	paths, err := processor.ExtractPaths(jsonData)
	if err != nil {
		return nil, fmt.Errorf("failed to extract paths: %w", err)
	}
	
	results := make([]ValidationResult, 0, len(paths))
	for _, path := range paths {
		if v.ValidatePath(path) {
			value, _ := processor.ExtractValue(jsonData, path)
			results = append(results, ValidationResult{
				Path:      path,
				Value:     value,
				Valid:     true,
				Timestamp: time.Now(),
			})
		}
	}
	
	return &ValidationReport{
		Results:    results,
		TotalPaths: len(results),
		ValidPaths: len(results),
	}, nil
}


// ValidateWithHandler performs validation with custom handler
func (v *OptimizedValidator) ValidateWithHandler(jsonData string, handler ValidationHandler) error {
	report, err := v.Validate(jsonData)
	if err != nil {
		return err
	}
	
	for _, result := range report.Results {
		if err := handler(result); err != nil {
			return err
		}
	}
	
	return nil
}

// matchesWildcardPattern checks if a path matches a wildcard pattern
func matchesWildcardPattern(path, pattern string) bool {
	// Simple wildcard matching - convert pattern to regex-like matching
	patternParts := strings.Split(pattern, "[*]")
	pathParts := strings.Split(path, "[")
	
	if len(patternParts) != len(pathParts) {
		return false
	}
	
	for i, patternPart := range patternParts {
		if i == 0 {
			// First part should match exactly
			if !strings.HasPrefix(pathParts[i], patternPart) {
				return false
			}
		} else {
			// Subsequent parts should contain the pattern
			pathPart := strings.Split(pathParts[i], "]")[0] // Extract index part
			if !strings.Contains(pathPart, patternPart) {
				return false
			}
		}
	}
	
	return true
}