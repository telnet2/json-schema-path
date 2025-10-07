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
	config         *GenericValidatorConfig
	patternTree    *tree.PatternTree
	precomputed    map[string]precomputedValidation
	processor      *jsonpkg.PathExtractor
	parsedPatterns map[string]*tree.PatternTree // Cached parsed patterns for fast metadata lookup
	patternToMeta  map[string]json.RawMessage   // Pattern to metadata mapping
}

// NewOptimizedGenericValidator creates an optimized validator with pre-computation and pattern support
func NewOptimizedGenericValidator(config *GenericValidatorConfig) (*OptimizedGenericValidator, error) {
	if config == nil {
		return nil, fmt.Errorf("configuration cannot be nil")
	}

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

	return &OptimizedGenericValidator{
		config:         config,
		patternTree:    patternTree,
		precomputed:    make(map[string]precomputedValidation),
		processor:      jsonpkg.NewPathExtractor(),
		parsedPatterns: parsedPatterns,
		patternToMeta:  patternToMeta,
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

	// Pre-allocate results slice with known capacity
	results := make([]ValidationResult, 0, len(v.precomputed))
	timestamp := time.Now() // Single timestamp for all results

	// Process pre-computed paths efficiently
	for _, precomp := range v.precomputed {
		value, err := v.processor.ExtractValue(jsonData, precomp.path)
		if err == nil && value != nil {
			validationResult := ValidationResult{
				Path:        precomp.path,
				Value:       value,
				Metadata:    precomp.metadata,
				Timestamp:   timestamp, // Reuse same timestamp
				Valid:       true,
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