package validators

import (
	"fmt"
)





// Builder pattern for creating validators fluently

// ValidatorBuilder provides a fluent interface for building validators
type ValidatorBuilder struct {
	validatorType string
	name          string
	description   string
	paths         []string
	pathMetadata  map[string]interface{}
	options       *ValidatorOptions
}

// NewValidatorBuilder creates a new validator builder
func NewValidatorBuilder(validatorType string) *ValidatorBuilder {
	return &ValidatorBuilder{
		validatorType: validatorType,
		name:          "custom_validator",
		description:   "Custom built validator",
		paths:         []string{},
		pathMetadata:  make(map[string]interface{}),
		options:       NewValidatorOptions(),
	}
}

// WithName sets the validator name
func (b *ValidatorBuilder) WithName(name string) *ValidatorBuilder {
	b.name = name
	return b
}

// WithDescription sets the validator description
func (b *ValidatorBuilder) WithDescription(description string) *ValidatorBuilder {
	b.description = description
	return b
}

// AddPath adds a simple path (for simple validators)
func (b *ValidatorBuilder) AddPath(path string) *ValidatorBuilder {
	b.paths = append(b.paths, path)
	return b
}

// AddValidationRule adds a path with validation metadata (for generic validators)
func (b *ValidatorBuilder) AddValidationRule(path string, ruleType string, required bool, constraints map[string]interface{}) *ValidatorBuilder {
	metadata := map[string]interface{}{
		"validation": ruleType,
		"required":   required,
	}
	for k, v := range constraints {
		metadata[k] = v
	}
	b.pathMetadata[path] = metadata
	return b
}

// WithOptions sets validator options
func (b *ValidatorBuilder) WithOptions(options *ValidatorOptions) *ValidatorBuilder {
	b.options = options
	return b
}

// Build creates the validator
func (b *ValidatorBuilder) Build() (UnifiedValidator, error) {
	switch b.validatorType {
	case "raw":
		config := NewSimpleValidatorConfig(b.name)
		config.Description = b.description
		config.Options = b.options
		config.AddPaths(b.paths)
		return NewRawValidator(config)
		
	case "optimized":
		config := NewSimpleValidatorConfig(b.name)
		config.Description = b.description
		config.Options = b.options
		config.AddPaths(b.paths)
		return NewOptimizedValidator(config)
		
	case "fast":
		config := NewSimpleValidatorConfig(b.name)
		config.Description = b.description
		config.Options = b.options
		config.AddPaths(b.paths)
		return NewFastValidator(config)
		
	case "complex_pattern":
		config := NewGenericValidatorConfig(b.name)
		config.Description = b.description
		config.Options = b.options
		for path, metadata := range b.pathMetadata {
			config.AddPath(path, metadata)
		}
		return NewComplexPatternValidator(config)
		
	case "enhanced_gjson":
		config := NewGenericValidatorConfig(b.name)
		config.Description = b.description
		config.Options = b.options
		for path, metadata := range b.pathMetadata {
			config.AddPath(path, metadata)
		}
		return NewEnhancedGJSONValidator(config)
		
	case "optimized_generic":
		config := NewGenericValidatorConfig(b.name)
		config.Description = b.description
		config.Options = b.options
		for path, metadata := range b.pathMetadata {
			config.AddPath(path, metadata)
		}
		return NewOptimizedGenericValidator(config)
		
	case "simple_generic":
		config := NewGenericValidatorConfig(b.name)
		config.Description = b.description
		config.Options = b.options
		for path, metadata := range b.pathMetadata {
			config.AddPath(path, metadata)
		}
		return NewSimpleGenericValidator(config)
		
	default:
		return nil, fmt.Errorf("unknown validator type: %s", b.validatorType)
	}
}

// Common validator configurations as convenience functions

// NewEmailValidator creates an email validation validator
func NewEmailValidator(validatorType string) (UnifiedValidator, error) {
	return NewValidatorBuilder(validatorType).
		WithName("email_validator").
		WithDescription("Email address validation").
		AddValidationRule("$.user.email", "email", true, map[string]interface{}{
			"pattern":   "^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\\.[a-zA-Z]{2,}$",
			"max_length": 100,
		}).
		AddValidationRule("$.users[*].email", "email", false, map[string]interface{}{
			"pattern": "^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\\.[a-zA-Z]{2,}$",
		}).
		Build()
}

// NewNumericValidator creates a numeric validation validator
func NewNumericValidator(validatorType string) (UnifiedValidator, error) {
	return NewValidatorBuilder(validatorType).
		WithName("numeric_validator").
		WithDescription("Numeric field validation").
		AddValidationRule("$.items[*].price", "numeric", true, map[string]interface{}{
			"min":       0,
			"max":       1000000,
			"precision": 2,
		}).
		AddValidationRule("$.order.total", "numeric", true, map[string]interface{}{
			"min": 0,
		}).
		Build()
}

// NewStringValidator creates a string validation validator
func NewStringValidator(validatorType string) (UnifiedValidator, error) {
	return NewValidatorBuilder(validatorType).
		WithName("string_validator").
		WithDescription("String field validation").
		AddValidationRule("$.user.name", "string", true, map[string]interface{}{
			"min_length": 2,
			"max_length": 50,
			"pattern":    "^[a-zA-Z\\s]+$",
		}).
		AddValidationRule("$.product.description", "string", false, map[string]interface{}{
			"max_length": 500,
		}).
		Build()
}