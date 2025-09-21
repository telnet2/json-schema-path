package validators

import (
	"encoding/json"
	"fmt"
	"time"

	jsonpkg "github.com/telnet2/json-schema-path/json"
)

// RawValidator provides direct path validation using JSON traversal
type RawValidator struct {
	config    *SimpleValidatorConfig
	pathCache map[string]bool
}

// NewRawValidator creates a new raw validator with direct configuration
func NewRawValidator(config *SimpleValidatorConfig) (*RawValidator, error) {
	if config == nil {
		return nil, fmt.Errorf("configuration cannot be nil")
	}
	
	// Build path cache for faster lookups
	pathCache := make(map[string]bool)
	for path := range config.Paths {
		pathCache[path] = true
	}
	
	return &RawValidator{
		config:    config,
		pathCache: pathCache,
	}, nil
}

// NewRawValidatorFromJSON creates a raw validator from JSON schema data
func NewRawValidatorFromJSON(schemaJSON string) (*RawValidator, error) {
	processor := jsonpkg.NewPathExtractor()
	paths, err := processor.ExtractPaths(schemaJSON)
	if err != nil {
		return nil, fmt.Errorf("failed to extract schema paths: %w", err)
	}
	
	config := NewSimpleValidatorConfig("raw_validator")
	config.AddPaths(paths)
	
	return NewRawValidator(config)
}

// ValidatePath checks if a path exists using direct lookup
func (v *RawValidator) ValidatePath(path string) bool {
	return v.pathCache[path]
}

// GetSupportedPaths returns all available paths
func (v *RawValidator) GetSupportedPaths() []string {
	paths := make([]string, 0, len(v.pathCache))
	for path := range v.pathCache {
		paths = append(paths, path)
	}
	return paths
}

// GetConfig returns the validator configuration (implements UnifiedValidator)
func (v *RawValidator) GetConfig() *ValidatorConfig {
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
func (v *RawValidator) GetName() string {
	return v.config.Name
}

// Validate performs full validation (simple validators return basic report)
func (v *RawValidator) Validate(jsonData string) (*ValidationReport, error) {
	// For simple validators, extract paths and check against our cache
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
func (v *RawValidator) ValidateWithHandler(jsonData string, handler ValidationHandler) error {
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