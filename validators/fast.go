package validators

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	jsonpkg "github.com/telnet2/json-schema-path/json"
)

// FastValidator provides ultra-fast validation with pre-expanded patterns
type FastValidator struct {
	config       *SimpleValidatorConfig
	expandedPaths map[string]bool
}

// NewFastValidator creates a fast validator with pre-expanded patterns
func NewFastValidator(config *SimpleValidatorConfig) (*FastValidator, error) {
	if config == nil {
		return nil, fmt.Errorf("configuration cannot be nil")
	}
	
	expandedPaths := make(map[string]bool)
	
	// Pre-expand all patterns for ultra-fast lookup
	for path := range config.Paths {
		expandedPaths[path] = true
		
		// Expand wildcard patterns
		if strings.Contains(path, "[*]") {
			variants := generatePathVariants(path)
			for _, variant := range variants {
				expandedPaths[variant] = true
			}
		}
	}
	
	return &FastValidator{
		config:        config,
		expandedPaths: expandedPaths,
	}, nil
}

// NewFastValidatorFromJSON creates a fast validator from JSON schema data
func NewFastValidatorFromJSON(schemaJSON string) (*FastValidator, error) {
	processor := jsonpkg.NewPathExtractor()
	paths, err := processor.ExtractPaths(schemaJSON)
	if err != nil {
		return nil, fmt.Errorf("failed to extract schema paths: %w", err)
	}
	
	config := NewSimpleValidatorConfig("fast_validator")
	config.AddPaths(paths)
	
	return NewFastValidator(config)
}

// ValidatePath checks if a path exists using pre-expanded lookup (fastest)
func (v *FastValidator) ValidatePath(path string) bool {
	return v.expandedPaths[path]
}

// GetSupportedPaths returns all available paths
func (v *FastValidator) GetSupportedPaths() []string {
	paths := make([]string, 0, len(v.expandedPaths))
	for path := range v.expandedPaths {
		paths = append(paths, path)
	}
	return paths
}

// GetConfig returns the validator configuration (implements UnifiedValidator)
func (v *FastValidator) GetConfig() *ValidatorConfig {
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
func (v *FastValidator) GetName() string {
	return v.config.Name
}

// Validate performs full validation with maximum speed
func (v *FastValidator) Validate(jsonData string) (*ValidationReport, error) {
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
func (v *FastValidator) ValidateWithHandler(jsonData string, handler ValidationHandler) error {
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

// generatePathVariants generates possible wildcard variants for a path
func generatePathVariants(path string) []string {
	variants := []string{path}
	
	// Replace specific indices with wildcards
	wildcardPath := convertToWildcardPattern(path)
	if wildcardPath != path {
		variants = append(variants, wildcardPath)
	}
	
	return variants
}

// convertToWildcardPattern converts specific array indices to [*] wildcards
func convertToWildcardPattern(path string) string {
	result := path
	for i := 0; i < 10; i++ {
		result = strings.ReplaceAll(result, fmt.Sprintf("[%d]", i), "[*]")
	}
	return result
}