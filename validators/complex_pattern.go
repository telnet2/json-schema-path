package validators

import (
	"encoding/json"
	"fmt"
	"time"

	jsonpkg "github.com/telnet2/json-schema-path/json"
	"github.com/telnet2/json-schema-path/parser"
	"github.com/telnet2/json-schema-path/tree"
)

// ComplexPatternValidator provides validation with full json-schema-path pattern support
type ComplexPatternValidator struct {
	config         *GenericValidatorConfig
	patternTree    *tree.PatternTree
	processor      *jsonpkg.PathExtractor
	parsedPatterns map[string]*tree.PatternTree // Cached parsed patterns for fast metadata lookup
	patternToMeta  map[string]json.RawMessage   // Pattern to metadata mapping
}

// NewComplexPatternValidator creates a validator with full pattern support using direct configuration
func NewComplexPatternValidator(config *GenericValidatorConfig) (*ComplexPatternValidator, error) {
	if config == nil {
		return nil, fmt.Errorf("configuration cannot be nil")
	}

	processor := jsonpkg.NewPathExtractor()
	patternTree := tree.NewPatternTree()
	parsedPatterns := make(map[string]*tree.PatternTree, len(config.Paths))
	patternToMeta := make(map[string]json.RawMessage, len(config.Paths))

	// Parse all configured patterns once and cache them
	for patternStr, metadata := range config.Paths {
		expr, err := parser.ParseExpression(patternStr)
		if err != nil {
			return nil, fmt.Errorf("failed to parse pattern %s: %w", patternStr, err)
		}
		patternTree.AddPattern(expr)

		// Cache individual pattern tree for fast metadata lookup
		individualTree := tree.NewPatternTree()
		individualTree.AddPattern(expr)
		parsedPatterns[patternStr] = individualTree
		patternToMeta[patternStr] = metadata
	}

	return &ComplexPatternValidator{
		config:         config,
		patternTree:    patternTree,
		processor:      processor,
		parsedPatterns: parsedPatterns,
		patternToMeta:  patternToMeta,
	}, nil
}

// NewComplexPatternValidatorFromPaths creates a validator from a list of paths with metadata
func NewComplexPatternValidatorFromPaths(paths []string, metadata map[string]interface{}) (*ComplexPatternValidator, error) {
	config := NewGenericValidatorConfig("complex_pattern_validator")
	
	// Add paths with metadata
	for _, path := range paths {
		if meta, exists := metadata[path]; exists {
			config.AddPath(path, meta)
		} else {
			config.AddPath(path, map[string]interface{}{"validation": "any"})
		}
	}
	
	return NewComplexPatternValidator(config)
}

// Validate performs validation using complex pattern matching
func (v *ComplexPatternValidator) Validate(jsonData string) (*ValidationReport, error) {
	start := time.Now()

	// Extract all paths from JSON data
	paths, err := v.processor.ExtractPaths(jsonData)
	if err != nil {
		return nil, fmt.Errorf("failed to extract paths: %w", err)
	}

	// Pre-allocate results slice with reasonable capacity
	results := make([]ValidationResult, 0, len(paths)/4)
	timestamp := time.Now() // Single timestamp for all results

	// Check each path against configured patterns
	for _, path := range paths {
		segments := v.processor.ConvertPathToSegments(path)

		// Check if this path matches any configured pattern
		if v.patternTree.MatchSegments(segments) {
			value, _ := v.processor.ExtractValue(jsonData, path)
			metadata := v.getMetadataForPath(path)

			result := ValidationResult{
				Path:        path,
				Value:       value,
				Metadata:    metadata,
				Timestamp:   timestamp, // Reuse same timestamp
				Valid:       true,
				Description: fmt.Sprintf("Path %s matches configured pattern", path),
			}
			results = append(results, result)
		}
	}

	duration := time.Since(start)
	return v.processResults(results, duration), nil
}

// ValidateWithHandler performs validation with custom handler
func (v *ComplexPatternValidator) ValidateWithHandler(jsonData string, handler ValidationHandler) error {
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

// GetSupportedPaths returns all configured pattern paths
func (v *ComplexPatternValidator) GetSupportedPaths() []string {
	paths := make([]string, 0, len(v.config.Paths))
	for path := range v.config.Paths {
		paths = append(paths, path)
	}
	return paths
}

// GetConfig returns the validator configuration (implements UnifiedValidator)
func (v *ComplexPatternValidator) GetConfig() *ValidatorConfig {
	// Convert GenericValidatorConfig to ValidatorConfig for interface compatibility
	return &ValidatorConfig{
		Name:        v.config.Name,
		Description: v.config.Description,
		Paths:       v.config.Paths,
	}
}

// GetName returns the validator name
func (v *ComplexPatternValidator) GetName() string {
	return v.config.Name
}

// ValidatePath checks if a path matches any configured pattern (simplified interface)
func (v *ComplexPatternValidator) ValidatePath(path string) bool {
	segments := v.processor.ConvertPathToSegments(path)
	return v.patternTree.MatchSegments(segments)
}

// getMetadataForPath retrieves metadata for a matching path
func (v *ComplexPatternValidator) getMetadataForPath(path string) json.RawMessage {
	// Try exact match first
	if metadata, exists := v.config.Paths[path]; exists {
		return metadata
	}

	// Use pre-parsed patterns for fast lookup - NO temp tree creation!
	segments := v.processor.ConvertPathToSegments(path)
	for patternStr, patternTree := range v.parsedPatterns {
		if patternTree.MatchSegments(segments) {
			return v.patternToMeta[patternStr]
		}
	}

	return nil
}

// processResults processes and aggregates validation results
func (v *ComplexPatternValidator) processResults(results []ValidationResult, duration time.Duration) *ValidationReport {
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