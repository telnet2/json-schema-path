package validators

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/tidwall/gjson"
)

// GJSONValidator provides validation using tidwall/gjson for comparison
type GJSONValidator struct {
	config    *GenericValidatorConfig
	gjsonPatterns map[string]json.RawMessage
}

// NewGJSONValidator creates a GJSON-based validator for comparison
func NewGJSONValidator(config *GenericValidatorConfig) (*GJSONValidator, error) {
	if config == nil {
		return nil, fmt.Errorf("configuration cannot be nil")
	}
	
	// Convert patterns to GJSON-compatible format
	gjsonPatterns := make(map[string]json.RawMessage)
	for pattern, metadata := range config.Paths {
		gjsonPattern := convertToGJSONPattern(pattern)
		gjsonPatterns[gjsonPattern] = metadata
	}
	
	return &GJSONValidator{
		config:        config,
		gjsonPatterns: gjsonPatterns,
	}, nil
}

// NewGJSONValidatorFromPaths creates a GJSON validator from paths with metadata
func NewGJSONValidatorFromPaths(paths []string, metadata map[string]interface{}) (*GJSONValidator, error) {
	config := NewGenericValidatorConfig("gjson_validator")
	
	// Add paths with metadata
	for _, path := range paths {
		if meta, exists := metadata[path]; exists {
			config.AddPath(path, meta)
		} else {
			config.AddPath(path, map[string]interface{}{"validation": "any"})
		}
	}
	
	return NewGJSONValidator(config)
}

// Validate performs validation using GJSON for maximum performance
func (v *GJSONValidator) Validate(jsonData string) (*ValidationReport, error) {
	result := gjson.Parse(jsonData)
	results := []ValidationResult{}
	
	// Check each configured pattern
	for pattern, metadata := range v.config.Paths {
		// Convert our pattern syntax to GJSON syntax
		gjsonPattern := convertToGJSONPattern(pattern)
		
		// Check if this path exists
		if result.Get(gjsonPattern).Exists() {
			// For array results, we need to handle each element separately
			gjsonResult := result.Get(gjsonPattern)
			if gjsonResult.IsArray() {
				// For patterns with [*], GJSON returns array, we create result for each element
				gjsonResult.ForEach(func(key, value gjson.Result) bool {
					// Create a specific path for this array element
					specificPath := strings.Replace(pattern, "[*]", fmt.Sprintf("[%s]", key.String()), 1)
					
					validationResult := ValidationResult{
						Path:        specificPath,
						Value:       value.Value(),
						Metadata:    metadata,
						Timestamp:   time.Now(),
						Valid:       true,
						Description: fmt.Sprintf("GJSON pattern %s matched", gjsonPattern),
					}
					results = append(results, validationResult)
					return true
				})
			} else {
				// Single value result
				validationResult := ValidationResult{
					Path:        pattern,
					Value:       gjsonResult.Value(),
					Metadata:    metadata,
					Timestamp:   time.Now(),
					Valid:       true,
					Description: fmt.Sprintf("GJSON pattern %s matched", gjsonPattern),
				}
				results = append(results, validationResult)
			}
		}
	}
	
	return &ValidationReport{
		Results:    results,
		TotalPaths: len(results),
		ValidPaths: len(results),
	}, nil
}

// ValidateWithHandler performs validation with custom handler
func (v *GJSONValidator) ValidateWithHandler(jsonData string, handler ValidationHandler) error {
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

// GetSupportedPaths returns all configured paths
func (v *GJSONValidator) GetSupportedPaths() []string {
	paths := make([]string, 0, len(v.config.Paths))
	for path := range v.config.Paths {
		paths = append(paths, path)
	}
	return paths
}

// GetConfig returns the validator configuration
func (v *GJSONValidator) GetConfig() *ValidatorConfig {
	return &ValidatorConfig{
		Name:        v.config.Name,
		Description: v.config.Description,
		Paths:       v.config.Paths,
	}
}

// GetName returns the validator name
func (v *GJSONValidator) GetName() string {
	return v.config.Name
}

// ValidatePath checks if a path exists using GJSON (simplified interface)
func (v *GJSONValidator) ValidatePath(path string) bool {
	// Simplified - would need full JSON data to validate
	return false
}


// convertToGJSONPattern converts json-schema-path pattern to GJSON format
func convertToGJSONPattern(pattern string) string {
	// Simple conversion - this is limited compared to our full pattern support
	gjsonPattern := pattern
	
	// Convert array wildcards - GJSON uses # for wildcards
	// Replace [*] with # for GJSON compatibility
	gjsonPattern = strings.ReplaceAll(gjsonPattern, "[*]", "#")
	
	return gjsonPattern
}