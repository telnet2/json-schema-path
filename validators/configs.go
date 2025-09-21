package validators

import (
	"encoding/json"
)

// Common validation configurations for different use cases

// EmailValidationConfig provides email validation configuration
func EmailValidationConfig() *ValidatorConfig {
	return &ValidatorConfig{
		Name:        "email_validation",
		Description: "Email address validation with regex patterns",
		Paths: map[string]json.RawMessage{
			"$.user.email": json.RawMessage(`{
				"validation": "email",
				"required": true,
				"pattern": "^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\\.[a-zA-Z]{2,}$",
				"max_length": 100
			}`),
			"$.users[*].email": json.RawMessage(`{
				"validation": "email",
				"required": false,
				"pattern": "^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\\.[a-zA-Z]{2,}$"
			}`),
			"$.contacts[~^.*_email$]": json.RawMessage(`{
				"validation": "email",
				"required": false,
				"pattern": "^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\\.[a-zA-Z]{2,}$"
			}`),
		},
	}
}

// NumericValidationConfig provides numeric validation configuration
func NumericValidationConfig() *ValidatorConfig {
	return &ValidatorConfig{
		Name:        "numeric_validation",
		Description: "Numeric field validation with ranges",
		Paths: map[string]json.RawMessage{
			"$.items[*].price": json.RawMessage(`{
				"validation": "numeric",
				"min": 0,
				"max": 1000000,
				"precision": 2
			}`),
			"$.order.total": json.RawMessage(`{
				"validation": "numeric",
				"min": 0,
				"required": true
			}`),
			"$.products[#*price]": json.RawMessage(`{
				"validation": "numeric",
				"min": 0,
				"max": 1000000
			}`),
		},
	}
}

// StringValidationConfig provides string validation configuration
func StringValidationConfig() *ValidatorConfig {
	return &ValidatorConfig{
		Name:        "string_validation",
		Description: "String field validation with patterns",
		Paths: map[string]json.RawMessage{
			"$.user.name": json.RawMessage(`{
				"validation": "string",
				"required": true,
				"min_length": 2,
				"max_length": 50,
				"pattern": "^[a-zA-Z\\s]+$"
			}`),
			"$.users[*].name": json.RawMessage(`{
				"validation": "string",
				"min_length": 1,
				"max_length": 100
			}`),
			"$.fields[~^user_.*]": json.RawMessage(`{
				"validation": "string",
				"min_length": 1,
				"max_length": 50
			}`),
		},
	}
}

// EnterpriseValidationConfig provides comprehensive enterprise validation
func EnterpriseValidationConfig() *ValidatorConfig {
	return &ValidatorConfig{
		Name:        "enterprise_validation",
		Description: "Comprehensive enterprise data validation with complex patterns",
		Paths: map[string]json.RawMessage{
			"$.company.employees[*].profile.name": json.RawMessage(`{
				"validation": "string",
				"required": true,
				"pattern": "^[a-zA-Z\\s]+$"
			}`),
			"$.company.departments[*].teams[*].members[*].email": json.RawMessage(`{
				"validation": "email",
				"required": true,
				"pattern": "^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\\.[a-zA-Z]{2,}$"
			}`),
			"$.enterprise.(departments|divisions){*}.(budget|revenue)": json.RawMessage(`{
				"validation": "numeric",
				"min": 0,
				"max": 1000000000
			}`),
			"$.organization.contacts[#*manager]": json.RawMessage(`{
				"validation": "string",
				"required": true,
				"min_length": 2
			}`),
		},
	}
}

// APIValidationConfig provides API response validation
func APIValidationConfig() *ValidatorConfig {
	return &ValidatorConfig{
		Name:        "api_validation",
		Description: "API response validation with status codes and data patterns",
		Paths: map[string]json.RawMessage{
			"$.status": json.RawMessage(`{
				"validation": "string",
				"required": true,
				"enum": ["success", "error", "pending"]
			}`),
			"$.data[*].id": json.RawMessage(`{
				"validation": "string",
				"required": true,
				"pattern": "^[a-zA-Z0-9_-]+$"
			}`),
			"$.data[~^api_.*]": json.RawMessage(`{
				"validation": "string",
				"required": false
			}`),
			"$.metadata.(created_at|updated_at)": json.RawMessage(`{
				"validation": "string",
				"required": true,
				"pattern": "^\\d{4}-\\d{2}-\\d{2}T\\d{2}:\\d{2}:\\d{2}Z$"
			}`),
		},
	}
}