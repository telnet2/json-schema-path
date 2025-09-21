package validators

import (
	"encoding/json"
	"fmt"
	"time"

	jsonpkg "github.com/telnet2/json-schema-path/json"
)

// SimpleGenericValidator provides basic generic validation without complex patterns
type SimpleGenericValidator struct {
	config    *GenericValidatorConfig
	pathCache map[string]json.RawMessage
	processor *jsonpkg.PathExtractor
}

// NewSimpleGenericValidator creates a simple generic validator with direct configuration
func NewSimpleGenericValidator(config *GenericValidatorConfig) (*SimpleGenericValidator, error) {
	if config == nil {
		return nil, fmt.Errorf("configuration cannot be nil")
	}
	
	// Build path cache for faster lookups
	pathCache := make(map[string]json.RawMessage)
	for path, metadata := range config.Paths {
		pathCache[path] = metadata
	}
	
	return &SimpleGenericValidator{
		config:    config,
		pathCache: pathCache,
		processor: jsonpkg.NewPathExtractor(),
	}, nil
}

// NewSimpleGenericValidatorFromPaths creates a simple generic validator from paths with metadata
func NewSimpleGenericValidatorFromPaths(paths []string, metadata map[string]interface{}) (*SimpleGenericValidator, error) {
	config := NewGenericValidatorConfig("simple_generic_validator")
	
	// Add paths with metadata
	for _, path := range paths {
		if meta, exists := metadata[path]; exists {
			config.AddPath(path, meta)
		} else {
			config.AddPath(path, map[string]interface{}{"validation": "any"})
		}
	}
	
	return NewSimpleGenericValidator(config)
}

// Validate performs basic validation with exact path matching
func (v *SimpleGenericValidator) Validate(jsonData string) (*ValidationReport, error) {
	start := time.Now()
	results := []ValidationResult{}

	// Extract all paths from JSON data
	paths, err := v.processor.ExtractPaths(jsonData)
	if err != nil {
		return nil, fmt.Errorf("failed to extract paths: %w", err)
	}

	// Check each path against configured paths (exact match only)
	for _, path := range paths {
		if metadata, exists := v.pathCache[path]; exists {
			value, _ := v.processor.ExtractValue(jsonData, path)
			
			result := ValidationResult{
				Path:      path,
				Value:     value,
				Metadata:  metadata,
				Timestamp: time.Now(),
				Valid:     true,
				Description: fmt.Sprintf("Path %s validated successfully", path),
			}
			results = append(results, result)
		}
	}

	duration := time.Since(start)
	return v.processResults(results, duration), nil
}

// ValidateWithHandler performs validation with custom handler
func (v *SimpleGenericValidator) ValidateWithHandler(jsonData string, handler ValidationHandler) error {
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
func (v *SimpleGenericValidator) GetSupportedPaths() []string {
	paths := make([]string, 0, len(v.pathCache))
	for path := range v.pathCache {
		paths = append(paths, path)
	}
	return paths
}

// GetConfig returns the validator configuration (implements UnifiedValidator)
func (v *SimpleGenericValidator) GetConfig() *ValidatorConfig {
	// Convert GenericValidatorConfig to ValidatorConfig for interface compatibility
	return &ValidatorConfig{
		Name:        v.config.Name,
		Description: v.config.Description,
		Paths:       v.config.Paths,
	}
}

// GetName returns the validator name
func (v *SimpleGenericValidator) GetName() string {
	return v.config.Name
}

// ValidatePath checks if a path exists in the cache (simplified interface)
func (v *SimpleGenericValidator) ValidatePath(path string) bool {
	_, exists := v.pathCache[path]
	return exists
}

// processResults processes and aggregates validation results
func (v *SimpleGenericValidator) processResults(results []ValidationResult, duration time.Duration) *ValidationReport {
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