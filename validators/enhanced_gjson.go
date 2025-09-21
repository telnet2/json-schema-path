package validators

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/tidwall/gjson"
)

// EnhancedGJSONValidator provides high-performance validation using gjson with unified interface
type EnhancedGJSONValidator struct {
	config      *GenericValidatorConfig
	pathCache   map[string]json.RawMessage
}

// NewEnhancedGJSONValidator creates an enhanced GJSON validator with direct configuration
func NewEnhancedGJSONValidator(config *GenericValidatorConfig) (*EnhancedGJSONValidator, error) {
	if config == nil {
		return nil, fmt.Errorf("configuration cannot be nil")
	}
	
	// Build path cache for fast lookup
	pathCache := make(map[string]json.RawMessage)
	for path, metadata := range config.Paths {
		pathCache[path] = metadata
	}
	
	return &EnhancedGJSONValidator{
		config:    config,
		pathCache: pathCache,
	}, nil
}

// NewEnhancedGJSONValidatorFromPaths creates an enhanced GJSON validator from paths with metadata
func NewEnhancedGJSONValidatorFromPaths(paths []string, metadata map[string]interface{}) (*EnhancedGJSONValidator, error) {
	config := NewGenericValidatorConfig("enhanced_gjson_validator")
	
	// Add paths with metadata
	for _, path := range paths {
		if meta, exists := metadata[path]; exists {
			config.AddPath(path, meta)
		} else {
			config.AddPath(path, map[string]interface{}{"validation": "any"})
		}
	}
	
	return NewEnhancedGJSONValidator(config)
}

// Validate performs validation using gjson for maximum performance
func (v *EnhancedGJSONValidator) Validate(jsonData string) (*ValidationReport, error) {
	start := time.Now()
	results := []ValidationResult{}

	// Use gjson for efficient JSON traversal
	result := gjson.Parse(jsonData)
	
	// Extract all paths and check against our cache
	v.traverseGJSON(result, "$", &results, jsonData)

	duration := time.Since(start)
	return v.processResults(results, duration), nil
}

// ValidateWithHandler performs validation with custom handler
func (v *EnhancedGJSONValidator) ValidateWithHandler(jsonData string, handler ValidationHandler) error {
	result := gjson.Parse(jsonData)
	return v.traverseGJSONWithHandler(result, "$", handler, jsonData)
}

// GetSupportedPaths returns all configured paths
func (v *EnhancedGJSONValidator) GetSupportedPaths() []string {
	paths := make([]string, 0, len(v.pathCache))
	for path := range v.pathCache {
		paths = append(paths, path)
	}
	return paths
}

// GetConfig returns the validator configuration (implements UnifiedValidator)
func (v *EnhancedGJSONValidator) GetConfig() *ValidatorConfig {
	// Convert GenericValidatorConfig to ValidatorConfig for interface compatibility
	return &ValidatorConfig{
		Name:        v.config.Name,
		Description: v.config.Description,
		Paths:       v.config.Paths,
	}
}

// GetName returns the validator name
func (v *EnhancedGJSONValidator) GetName() string {
	return v.config.Name
}

// ValidatePath checks if a path exists in the cache (simplified interface)
func (v *EnhancedGJSONValidator) ValidatePath(path string) bool {
	_, exists := v.pathCache[path]
	return exists
}

// traverseGJSON recursively traverses gjson result
func (v *EnhancedGJSONValidator) traverseGJSON(result gjson.Result, currentPath string, results *[]ValidationResult, jsonData string) {
	// Check if current path matches any configured path
	if metadata, exists := v.pathCache[currentPath]; exists {
		value := result.Value()
		validationResult := ValidationResult{
			Path:      currentPath,
			Value:     value,
			Metadata:  metadata,
			Timestamp: time.Now(),
			Valid:     true,
			Description: fmt.Sprintf("Path %s validated successfully", currentPath),
		}
		*results = append(*results, validationResult)
	}

	// Handle different JSON types
	switch {
	case result.IsObject():
		result.ForEach(func(key, value gjson.Result) bool {
			newPath := fmt.Sprintf("%s.%s", currentPath, key.String())
			v.traverseGJSON(value, newPath, results, jsonData)
			return true
		})
	case result.IsArray():
		result.ForEach(func(index, value gjson.Result) bool {
			newPath := fmt.Sprintf("%s[%d]", currentPath, int(index.Num))
			wildcardPath := fmt.Sprintf("%s[*]", currentPath)
			
			// Check both specific index and wildcard paths
			if _, exists := v.pathCache[newPath]; exists {
				val := value.Value()
				validationResult := ValidationResult{
					Path:      newPath,
					Value:     val,
					Metadata:  v.pathCache[newPath],
					Timestamp: time.Now(),
					Valid:     true,
					Description: fmt.Sprintf("Path %s validated successfully", newPath),
				}
				*results = append(*results, validationResult)
			}
			
			if _, exists := v.pathCache[wildcardPath]; exists {
				val := value.Value()
				validationResult := ValidationResult{
					Path:      wildcardPath,
					Value:     val,
					Metadata:  v.pathCache[wildcardPath],
					Timestamp: time.Now(),
					Valid:     true,
					Description: fmt.Sprintf("Path %s validated successfully", wildcardPath),
				}
				*results = append(*results, validationResult)
			}
			
			// Continue recursing into the array element
			v.traverseGJSON(value, newPath, results, jsonData)
			return true
		})
	}
}

// traverseGJSONWithHandler recursively traverses with custom handler
func (v *EnhancedGJSONValidator) traverseGJSONWithHandler(result gjson.Result, currentPath string, handler ValidationHandler, jsonData string) error {
	// Check if current path matches any configured path
	if metadata, exists := v.pathCache[currentPath]; exists {
		value := result.Value()
		validationResult := ValidationResult{
			Path:      currentPath,
			Value:     value,
			Metadata:  metadata,
			Timestamp: time.Now(),
			Valid:     true,
			Description: fmt.Sprintf("Path %s validated successfully", currentPath),
		}
		
		if err := handler(validationResult); err != nil {
			return err
		}
	}

	// Handle different JSON types
	switch {
	case result.IsObject():
		for key, value := range result.Map() {
			newPath := fmt.Sprintf("%s.%s", currentPath, key)
			if err := v.traverseGJSONWithHandler(value, newPath, handler, jsonData); err != nil {
				return err
			}
		}
	case result.IsArray():
		for i, value := range result.Array() {
			newPath := fmt.Sprintf("%s[%d]", currentPath, i)
			wildcardPath := fmt.Sprintf("%s[*]", currentPath)
			
			// Check both specific index and wildcard paths
			if _, exists := v.pathCache[newPath]; exists {
				val := value.Value()
				validationResult := ValidationResult{
					Path:      newPath,
					Value:     val,
					Metadata:  v.pathCache[newPath],
					Timestamp: time.Now(),
					Valid:     true,
					Description: fmt.Sprintf("Path %s validated successfully", newPath),
				}
				if err := handler(validationResult); err != nil {
					return err
				}
			}
			
			if _, exists := v.pathCache[wildcardPath]; exists {
				val := value.Value()
				validationResult := ValidationResult{
					Path:      wildcardPath,
					Value:     val,
					Metadata:  v.pathCache[wildcardPath],
					Timestamp: time.Now(),
					Valid:     true,
					Description: fmt.Sprintf("Path %s validated successfully", wildcardPath),
				}
				if err := handler(validationResult); err != nil {
					return err
				}
			}
			
			// Continue recursing into the array element
			if err := v.traverseGJSONWithHandler(value, newPath, handler, jsonData); err != nil {
				return err
			}
		}
	}

	return nil
}

// processResults processes and aggregates validation results
func (v *EnhancedGJSONValidator) processResults(results []ValidationResult, duration time.Duration) *ValidationReport {
	report := &ValidationReport{
		Results:    results,
		TotalPaths: len(results),
		Duration:   duration,
	}

	for _, result := range results {
		if result.Valid {
			report.ValidPaths++
		} else {
			report.InvalidPaths++
			if result.Error != nil {
				report.Errors = append(report.Errors, result.Error)
			}
		}
	}

	return report
}