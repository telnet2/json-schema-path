package validators

import (
	"encoding/json"
	"time"
)

// ValidationResult represents the outcome of a validation operation
type ValidationResult struct {
	Path        string          `json:"path"`
	Value       interface{}     `json:"value"`
	Metadata    json.RawMessage `json:"metadata"`
	Timestamp   time.Time       `json:"timestamp"`
	Error       error           `json:"error,omitempty"`
	Valid       bool            `json:"valid"`
	Description string          `json:"description,omitempty"`
}

// ValidationReport contains comprehensive validation results
type ValidationReport struct {
	Results      []ValidationResult `json:"results"`
	TotalPaths   int                `json:"total_paths"`
	ValidPaths   int                `json:"valid_paths"`
	InvalidPaths int                `json:"invalid_paths"`
	Errors       []error            `json:"errors,omitempty"`
	Duration     time.Duration      `json:"duration"`
}

// ValidationHandler is the function signature for validation handlers
type ValidationHandler func(result ValidationResult) error

// precomputedValidation represents pre-computed validation data
type precomputedValidation struct {
	path     string
	metadata json.RawMessage
}

// ValidatorConfig contains configuration for validators (for backward compatibility)
type ValidatorConfig struct {
	Name              string                     `yaml:"name" json:"name"`
	Description       string                     `yaml:"description" json:"description"`
	Paths             map[string]json.RawMessage `yaml:"paths" json:"paths"`
	SchemaDefinitions map[string]json.RawMessage `yaml:"schema_definitions,omitempty" json:"schema_definitions,omitempty"`
}

// UnifiedValidator interface provides a consistent API for all validator types
type UnifiedValidator interface {
	// Path validation for simple validators
	ValidatePath(path string) bool
	
	// Full validation for generic validators  
	Validate(jsonData string) (*ValidationReport, error)
	ValidateWithHandler(jsonData string, handler ValidationHandler) error
	
	// Common methods
	GetSupportedPaths() []string
	GetConfig() *ValidatorConfig
	GetName() string
}

// ValidatorOptions provides configuration options for validators
type ValidatorOptions struct {
	// Performance options
	EnablePrecomputation bool   `json:"enable_precomputation"`
	CacheSize           int    `json:"cache_size"`
	
	// Pattern matching options
	EnableWildcards     bool   `json:"enable_wildcards"`
	EnableRegex         bool   `json:"enable_regex"`
	EnableGroups        bool   `json:"enable_groups"`
	EnableRepetition    bool   `json:"enable_repetition"`
	
	// Validation options
	FailFast           bool   `json:"fail_fast"`
	ContinueOnError    bool   `json:"continue_on_error"`
	MaxDepth          int    `json:"max_depth"`
	
	// Metadata
	Name              string `json:"name"`
	Description       string `json:"description"`
}

// NewValidatorOptions creates default validator options
func NewValidatorOptions() *ValidatorOptions {
	return &ValidatorOptions{
		EnablePrecomputation: true,
		CacheSize:           1000,
		EnableWildcards:     true,
		EnableRegex:          true,
		EnableGroups:         true,
		EnableRepetition:     true,
		FailFast:            false,
		ContinueOnError:      true,
		MaxDepth:            50,
	}
}

// SimpleValidatorConfig provides configuration for simple path validators
type SimpleValidatorConfig struct {
	Name        string              `json:"name"`
	Description string              `json:"description"`
	Paths       map[string]bool     `json:"paths"`  // Simple path -> exists mapping
	Options     *ValidatorOptions   `json:"options"`
}

// NewSimpleValidatorConfig creates a simple validator configuration
func NewSimpleValidatorConfig(name string) *SimpleValidatorConfig {
	return &SimpleValidatorConfig{
		Name:    name,
		Paths:   make(map[string]bool),
		Options: NewValidatorOptions(),
	}
}

// AddPath adds a path to the simple validator configuration
func (c *SimpleValidatorConfig) AddPath(path string) *SimpleValidatorConfig {
	c.Paths[path] = true
	return c
}

// AddPaths adds multiple paths to the simple validator configuration
func (c *SimpleValidatorConfig) AddPaths(paths []string) *SimpleValidatorConfig {
	for _, path := range paths {
		c.Paths[path] = true
	}
	return c
}

// GenericValidatorConfig provides configuration for generic validators with metadata
type GenericValidatorConfig struct {
	Name        string                     `json:"name"`
	Description string                     `json:"description"`
	Paths       map[string]json.RawMessage `json:"paths"`  // Path -> metadata mapping
	Options     *ValidatorOptions          `json:"options"`
}

// NewGenericValidatorConfig creates a generic validator configuration
func NewGenericValidatorConfig(name string) *GenericValidatorConfig {
	return &GenericValidatorConfig{
		Name:    name,
		Paths:   make(map[string]json.RawMessage),
		Options: NewValidatorOptions(),
	}
}

// AddPath adds a path with metadata to the generic validator configuration
func (c *GenericValidatorConfig) AddPath(path string, metadata interface{}) *GenericValidatorConfig {
	metaJSON, err := json.Marshal(metadata)
	if err != nil {
		// If marshaling fails, store empty JSON object
		metaJSON = []byte("{}")
	}
	c.Paths[path] = json.RawMessage(metaJSON)
	return c
}

// AddValidationRule adds a validation rule with common metadata
func (c *GenericValidatorConfig) AddValidationRule(path string, ruleType string, required bool, constraints map[string]interface{}) *GenericValidatorConfig {
	metadata := map[string]interface{}{
		"validation": ruleType,
		"required":   required,
	}
	
	// Add constraints
	for k, v := range constraints {
		metadata[k] = v
	}
	
	return c.AddPath(path, metadata)
}

// Build creates the appropriate validator based on configuration
func (c *GenericValidatorConfig) Build() (UnifiedValidator, error) {
	if c.Options == nil {
		c.Options = NewValidatorOptions()
	}
	
	// Choose validator type based on options and complexity
	if c.Options.EnablePrecomputation && len(c.Paths) > 10 {
		return NewOptimizedGenericValidator(c)
	} else if c.Options.EnableWildcards || c.Options.EnableRegex || c.Options.EnableGroups {
		return NewComplexPatternValidator(c)
	} else {
		return NewSimpleGenericValidator(c)
	}
}

// Build creates the appropriate simple validator based on configuration  
func (c *SimpleValidatorConfig) Build() (UnifiedValidator, error) {
	if c.Options == nil {
		c.Options = NewValidatorOptions()
	}
	
	// Choose validator type based on options
	if c.Options.EnablePrecomputation && len(c.Paths) > 100 {
		return NewFastValidator(c)
	} else if c.Options.EnableWildcards {
		return NewOptimizedValidator(c)
	} else {
		return NewRawValidator(c)
	}
}