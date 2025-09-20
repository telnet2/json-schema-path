package json

import (
	"fmt"
	"sort"
	"strings"
	"unicode"

	"github.com/santhosh-tekuri/jsonschema/v5"
)

type schemaVisitKey struct {
	schema *jsonschema.Schema
	path   string
}

const defaultSchemaURL = "inline.json"

// ExtractSchemaPaths generates all possible schema paths from a JSON Schema document.
// The returned paths are JSONPath-like expressions rooted at $. Object properties use
// dot notation when possible and fall back to bracket notation with proper escaping.
// Array items are represented using either concrete indices (for tuple validation)
// or the wildcard form "[*]" when the schema applies to any position.
func (pe *PathExtractor) ExtractSchemaPaths(schemaData string) ([]string, error) {
	return pe.ExtractSchemaPathsWithURL(schemaData, defaultSchemaURL)
}

// ExtractSchemaPathsWithURL behaves like ExtractSchemaPaths but allows specifying a base URL
// that is used when compiling the schema. This is useful when the schema contains relative $ref
// references that should be resolved from a known location.
func (pe *PathExtractor) ExtractSchemaPathsWithURL(schemaData, resourceURL string) ([]string, error) {
	if resourceURL == "" {
		resourceURL = defaultSchemaURL
	}

	compiled, err := jsonschema.CompileString(resourceURL, schemaData)
	if err != nil {
		return nil, fmt.Errorf("failed to compile JSON schema: %w", err)
	}

	paths := make(map[string]struct{})
	visited := make(map[schemaVisitKey]struct{})
	stack := make(map[*jsonschema.Schema]bool)

	pe.collectSchemaPaths(compiled, "$", paths, visited, stack)

	result := make([]string, 0, len(paths))
	for path := range paths {
		result = append(result, path)
	}

	sort.Strings(result)
	return result, nil
}

func (pe *PathExtractor) collectSchemaPaths(schema *jsonschema.Schema, currentPath string, paths map[string]struct{}, visited map[schemaVisitKey]struct{}, stack map[*jsonschema.Schema]bool) {
	if schema == nil {
		return
	}

	if currentPath == "" {
		currentPath = "$"
	}

	addPath(paths, currentPath)

	key := schemaVisitKey{schema: schema, path: currentPath}
	if _, seen := visited[key]; seen {
		return
	}
	visited[key] = struct{}{}

	if stack[schema] {
		return
	}

	stack[schema] = true
	defer delete(stack, schema)

	pe.collectSchemaPaths(schema.Ref, currentPath, paths, visited, stack)
	pe.collectSchemaPaths(schema.RecursiveRef, currentPath, paths, visited, stack)
	pe.collectSchemaPaths(schema.DynamicRef, currentPath, paths, visited, stack)

	if schema.Not != nil {
		pe.collectSchemaPaths(schema.Not, currentPath, paths, visited, stack)
	}

	for _, sub := range schema.AllOf {
		pe.collectSchemaPaths(sub, currentPath, paths, visited, stack)
	}
	for _, sub := range schema.AnyOf {
		pe.collectSchemaPaths(sub, currentPath, paths, visited, stack)
	}
	for _, sub := range schema.OneOf {
		pe.collectSchemaPaths(sub, currentPath, paths, visited, stack)
	}

	if schema.If != nil {
		pe.collectSchemaPaths(schema.If, currentPath, paths, visited, stack)
	}
	if schema.Then != nil {
		pe.collectSchemaPaths(schema.Then, currentPath, paths, visited, stack)
	}
	if schema.Else != nil {
		pe.collectSchemaPaths(schema.Else, currentPath, paths, visited, stack)
	}

	for name, sub := range schema.Properties {
		childPath := currentPath + formatPropertySegment(name)
		pe.collectSchemaPaths(sub, childPath, paths, visited, stack)
	}

	for pattern, sub := range schema.PatternProperties {
		childPath := currentPath + "[~" + pattern.String() + "]"
		pe.collectSchemaPaths(sub, childPath, paths, visited, stack)
	}

	switch addProps := schema.AdditionalProperties.(type) {
	case bool:
		if addProps {
			addPath(paths, currentPath+"[#*]")
		}
	case *jsonschema.Schema:
		pe.collectSchemaPaths(addProps, currentPath+"[#*]", paths, visited, stack)
	}

	if schema.UnevaluatedProperties != nil {
		pe.collectSchemaPaths(schema.UnevaluatedProperties, currentPath+"[#*]", paths, visited, stack)
	}

	for _, sub := range schema.DependentSchemas {
		pe.collectSchemaPaths(sub, currentPath, paths, visited, stack)
	}

	for _, dep := range schema.Dependencies {
		if sub, ok := dep.(*jsonschema.Schema); ok {
			pe.collectSchemaPaths(sub, currentPath, paths, visited, stack)
		}
	}

	pe.expandArraySchemas(schema, currentPath, paths, visited, stack)
}

func (pe *PathExtractor) expandArraySchemas(schema *jsonschema.Schema, currentPath string, paths map[string]struct{}, visited map[schemaVisitKey]struct{}, stack map[*jsonschema.Schema]bool) {
	for idx, sub := range schema.PrefixItems {
		childPath := fmt.Sprintf("%s[%d]", currentPath, idx)
		pe.collectSchemaPaths(sub, childPath, paths, visited, stack)
	}

	switch items := schema.Items.(type) {
	case *jsonschema.Schema:
		pe.collectSchemaPaths(items, currentPath+"[*]", paths, visited, stack)
	case []*jsonschema.Schema:
		for idx, sub := range items {
			childPath := fmt.Sprintf("%s[%d]", currentPath, idx)
			pe.collectSchemaPaths(sub, childPath, paths, visited, stack)
		}
	}

	switch addItems := schema.AdditionalItems.(type) {
	case bool:
		if addItems {
			addPath(paths, currentPath+"[*]")
		}
	case *jsonschema.Schema:
		pe.collectSchemaPaths(addItems, currentPath+"[*]", paths, visited, stack)
	}

	if schema.Items2020 != nil {
		pe.collectSchemaPaths(schema.Items2020, currentPath+"[*]", paths, visited, stack)
	}

	if schema.Contains != nil {
		pe.collectSchemaPaths(schema.Contains, currentPath+"[*]", paths, visited, stack)
	}

	if schema.UnevaluatedItems != nil {
		pe.collectSchemaPaths(schema.UnevaluatedItems, currentPath+"[*]", paths, visited, stack)
	}

	if hasSchemaType(schema, "array") && len(schema.PrefixItems) == 0 && schema.Items == nil && schema.Items2020 == nil && schema.Contains == nil {
		addPath(paths, currentPath+"[*]")
	}
}

func addPath(paths map[string]struct{}, path string) {
	if _, exists := paths[path]; !exists {
		paths[path] = struct{}{}
	}
}

func hasSchemaType(schema *jsonschema.Schema, target string) bool {
	if schema == nil {
		return false
	}
	for _, t := range schema.Types {
		if t == target {
			return true
		}
	}
	return false
}

func formatPropertySegment(name string) string {
	if isIdentifier(name) {
		return "." + name
	}
	return "[\"" + escapePropertyName(name) + "\"]"
}

func isIdentifier(value string) bool {
	if value == "" {
		return false
	}
	for i, r := range value {
		if i == 0 {
			if !(r == '_' || unicode.IsLetter(r)) {
				return false
			}
		} else {
			if !(r == '_' || unicode.IsLetter(r) || unicode.IsDigit(r)) {
				return false
			}
		}
	}
	return true
}

func escapePropertyName(value string) string {
	if !strings.ContainsAny(value, "\\\"") {
		return value
	}
	var builder strings.Builder
	for _, r := range value {
		if r == '\\' || r == '"' {
			builder.WriteByte('\\')
		}
		builder.WriteRune(r)
	}
	return builder.String()
}
