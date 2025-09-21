package validators

import (
	"encoding/json"
	"fmt"
	"time"

	jsonpkg "github.com/telnet2/json-schema-path/json"
	"github.com/telnet2/json-schema-path/parser"
	"github.com/telnet2/json-schema-path/tree"
)

// OptimizedGenericValidator provides optimized validation with pre-computation and complex patterns
type OptimizedGenericValidator struct {
	config      *GenericValidatorConfig
	patternTree *tree.PatternTree
	precomputed map[string]precomputedValidation
	processor   *jsonpkg.PathExtractor
}

// NewOptimizedGenericValidator creates an optimized validator with pre-computation and pattern support
func NewOptimizedGenericValidator(config *GenericValidatorConfig) (*OptimizedGenericValidator, error) {
	if config == nil {
		return nil, fmt.Errorf("configuration cannot be nil")
	}
	
	patternTree := tree.NewPatternTree()

	// Parse all configured patterns
	for patternStr := range config.Paths {
		expr, err := parser.ParseExpression(patternStr)
		if err != nil {
			return nil, fmt.Errorf("failed to parse pattern %s: %w", patternStr, err)
		}
		patternTree.AddPattern(expr)
	}

	return &OptimizedGenericValidator{
		config:      config,
		patternTree: patternTree,
		precomputed: make(map[string]precomputedValidation),
		processor:   jsonpkg.NewPathExtractor(),
	}, nil
}

// Validate performs validation with pre-computed path matching and complex patterns
func (v *OptimizedGenericValidator) Validate(jsonData string) (*ValidationReport, error) {
	start := time.Now()

	// Pre-compute paths if not already done
	if len(v.precomputed) == 0 {
		if err := v.precomputePaths(jsonData); err != nil {
			return nil, fmt.Errorf("failed to precompute paths: %w", err)
		}
	}

	results := []ValidationResult{}

	// Process pre-computed paths efficiently
	for _, precomp := range v.precomputed {
		value, err := v.processor.ExtractValue(jsonData, precomp.path)
		if err == nil && value != nil {
			validationResult := ValidationResult{
				Path:      precomp.path,
				Value:     value,
				Metadata:  precomp.metadata,
				Timestamp: time.Now(),
				Valid:     true,
				Description: fmt.Sprintf("Path %s matches configured pattern", precomp.path),
			}
			results = append(results, validationResult)
		}
	}

	duration := time.Since(start)
	return v.processResults(results, duration), nil
}

// ValidateWithHandler performs validation with custom handler
func (v *OptimizedGenericValidator) ValidateWithHandler(jsonData string, handler ValidationHandler) error {
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
func (v *OptimizedGenericValidator) GetSupportedPaths() []string {
	paths := make([]string, 0, len(v.config.Paths))
	for path := range v.config.Paths {
		paths = append(paths, path)
	}
	return paths
}

// GetConfig returns the validator configuration (implements UnifiedValidator)
func (v *OptimizedGenericValidator) GetConfig() *ValidatorConfig {
	// Convert GenericValidatorConfig to ValidatorConfig for interface compatibility
	return &ValidatorConfig{
		Name:        v.config.Name,
		Description: v.config.Description,
		Paths:       v.config.Paths,
	}
}

// GetName returns the validator name
func (v *OptimizedGenericValidator) GetName() string {
	return v.config.Name
}

// ValidatePath checks if a path matches any configured pattern (simplified interface)
func (v *OptimizedGenericValidator) ValidatePath(path string) bool {
	segments := v.processor.ConvertPathToSegments(path)
	return v.patternTree.MatchSegments(segments)
}

// precomputePaths extracts all paths and matches them against configured patterns
func (v *OptimizedGenericValidator) precomputePaths(jsonData string) error {
	allPaths, err := v.processor.ExtractPaths(jsonData)
	if err != nil {
		return err
	}

	// Find paths that match any configured pattern
	for _, path := range allPaths {
		segments := v.processor.ConvertPathToSegments(path)
		
		// Check if this path matches any configured pattern
		if v.patternTree.MatchSegments(segments) {
			metadata := v.getMetadataForPath(path)
			v.precomputed[path] = precomputedValidation{
				path:     path,
				metadata: metadata,
			}
		}
	}

	return nil
}

// getMetadataForPath retrieves metadata for a matching path
func (v *OptimizedGenericValidator) getMetadataForPath(path string) json.RawMessage {
	// Try exact match first
	if metadata, exists := v.config.Paths[path]; exists {
		return metadata
	}

	// Try to find matching pattern
	segments := v.processor.ConvertPathToSegments(path)
	for patternStr, metadata := range v.config.Paths {
		expr, err := parser.ParseExpression(patternStr)
		if err != nil {
			continue
		}
		
		// Create temporary pattern tree to test this specific pattern
		tempTree := tree.NewPatternTree()
		tempTree.AddPattern(expr)
		
		if tempTree.MatchSegments(segments) {
			return metadata
		}
	}

	return nil
}

// processResults processes and aggregates validation results
func (v *OptimizedGenericValidator) processResults(results []ValidationResult, duration time.Duration) *ValidationReport {
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