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

// SchemaPathOptions control how schema paths are generated.
type SchemaPathOptions struct {
	// TerminalsOnly restricts emitted paths to schemas that represent terminal
	// JSON values (non-object, non-array). Container schemas are still
	// traversed but not returned in the result set.
	TerminalsOnly bool
}

type terminalState uint8

const (
	terminalUnknown terminalState = iota
	terminalComputing
	terminalNo
	terminalYes
)

// ExtractSchemaPaths generates all possible schema paths from a JSON Schema document.
// The returned paths are JSONPath-like expressions rooted at $. Object properties use
// dot notation when possible and fall back to bracket notation with proper escaping.
// Array items are represented using either concrete indices (for tuple validation)
// or the wildcard form "[*]" when the schema applies to any position.
func (pe *PathExtractor) ExtractSchemaPaths(schemaData string) ([]string, error) {
	return pe.ExtractSchemaPathsWithOptions(schemaData, defaultSchemaURL, SchemaPathOptions{})
}

// ExtractSchemaPathsWithURL behaves like ExtractSchemaPaths but allows specifying a base URL
// that is used when compiling the schema. This is useful when the schema contains relative $ref
// references that should be resolved from a known location.
func (pe *PathExtractor) ExtractSchemaPathsWithURL(schemaData, resourceURL string) ([]string, error) {
	return pe.ExtractSchemaPathsWithOptions(schemaData, resourceURL, SchemaPathOptions{})
}

// ExtractSchemaPathsWithOptions behaves like ExtractSchemaPathsWithURL but accepts additional
// options to control which paths are emitted.
func (pe *PathExtractor) ExtractSchemaPathsWithOptions(schemaData, resourceURL string, opts SchemaPathOptions) ([]string, error) {
	if resourceURL == "" {
		resourceURL = defaultSchemaURL
	}

	compiled, err := jsonschema.CompileString(resourceURL, schemaData)
	if err != nil {
		return nil, fmt.Errorf("failed to compile JSON schema: %w", err)
	}

	paths := make(map[string]string)
	visited := make(map[schemaVisitKey]struct{})
	stack := make(map[*jsonschema.Schema]int)
	terminalMemo := make(map[*jsonschema.Schema]terminalState)

	pe.collectSchemaPaths(compiled, "$", paths, visited, stack, terminalMemo, opts)

	result := make([]string, 0, len(paths))
	for _, path := range paths {
		result = append(result, path)
	}

	sort.Strings(result)
	return result, nil
}

func (pe *PathExtractor) collectSchemaPaths(schema *jsonschema.Schema, currentPath string, paths map[string]string, visited map[schemaVisitKey]struct{}, stack map[*jsonschema.Schema]int, terminalMemo map[*jsonschema.Schema]terminalState, opts SchemaPathOptions) {
	if schema == nil {
		return
	}

	if currentPath == "" {
		currentPath = "$"
	}

	if !opts.TerminalsOnly || isTerminalSchema(schema, terminalMemo) {
		addPath(paths, currentPath)
	}

	key := schemaVisitKey{schema: schema, path: currentPath}
	if _, seen := visited[key]; seen {
		return
	}
	visited[key] = struct{}{}

	inCycle := stack[schema] > 0
	stack[schema]++
	defer func() {
		stack[schema]--
		if stack[schema] == 0 {
			delete(stack, schema)
		}
	}()

	pe.collectSchemaTarget(schema.Ref, currentPath, paths, visited, stack, terminalMemo, opts, inCycle)
	pe.collectSchemaTarget(schema.RecursiveRef, currentPath, paths, visited, stack, terminalMemo, opts, inCycle)
	pe.collectSchemaTarget(schema.DynamicRef, currentPath, paths, visited, stack, terminalMemo, opts, inCycle)

	if schema.Not != nil {
		pe.collectSchemaTarget(schema.Not, currentPath, paths, visited, stack, terminalMemo, opts, inCycle)
	}

	for _, sub := range schema.AllOf {
		pe.collectSchemaTarget(sub, currentPath, paths, visited, stack, terminalMemo, opts, inCycle)
	}
	for _, sub := range schema.AnyOf {
		pe.collectSchemaTarget(sub, currentPath, paths, visited, stack, terminalMemo, opts, inCycle)
	}
	for _, sub := range schema.OneOf {
		pe.collectSchemaTarget(sub, currentPath, paths, visited, stack, terminalMemo, opts, inCycle)
	}

	if schema.If != nil {
		pe.collectSchemaTarget(schema.If, currentPath, paths, visited, stack, terminalMemo, opts, inCycle)
	}
	if schema.Then != nil {
		pe.collectSchemaTarget(schema.Then, currentPath, paths, visited, stack, terminalMemo, opts, inCycle)
	}
	if schema.Else != nil {
		pe.collectSchemaTarget(schema.Else, currentPath, paths, visited, stack, terminalMemo, opts, inCycle)
	}

	for name, sub := range schema.Properties {
		childPath := currentPath + formatPropertySegment(name)
		pe.collectSchemaTarget(sub, childPath, paths, visited, stack, terminalMemo, opts, inCycle)
	}

	for pattern, sub := range schema.PatternProperties {
		childPath := currentPath + "[~" + pattern.String() + "]"
		pe.collectSchemaTarget(sub, childPath, paths, visited, stack, terminalMemo, opts, inCycle)
	}

	switch addProps := schema.AdditionalProperties.(type) {
	case bool:
		if addProps && !opts.TerminalsOnly {
			addPath(paths, currentPath+"[#*]")
		}
	case *jsonschema.Schema:
		pe.collectSchemaTarget(addProps, currentPath+"[#*]", paths, visited, stack, terminalMemo, opts, inCycle)
	}

	if schema.UnevaluatedProperties != nil {
		pe.collectSchemaTarget(schema.UnevaluatedProperties, currentPath+"[#*]", paths, visited, stack, terminalMemo, opts, inCycle)
	}

	for _, sub := range schema.DependentSchemas {
		pe.collectSchemaTarget(sub, currentPath, paths, visited, stack, terminalMemo, opts, inCycle)
	}

	for _, dep := range schema.Dependencies {
		if sub, ok := dep.(*jsonschema.Schema); ok {
			pe.collectSchemaTarget(sub, currentPath, paths, visited, stack, terminalMemo, opts, inCycle)
		}
	}

	pe.expandArraySchemas(schema, currentPath, paths, visited, stack, terminalMemo, opts, inCycle)
}

func (pe *PathExtractor) expandArraySchemas(schema *jsonschema.Schema, currentPath string, paths map[string]string, visited map[schemaVisitKey]struct{}, stack map[*jsonschema.Schema]int, terminalMemo map[*jsonschema.Schema]terminalState, opts SchemaPathOptions, inCycle bool) {
	for idx, sub := range schema.PrefixItems {
		childPath := fmt.Sprintf("%s[%d]", currentPath, idx)
		pe.collectSchemaTarget(sub, childPath, paths, visited, stack, terminalMemo, opts, inCycle)
	}

	switch items := schema.Items.(type) {
	case *jsonschema.Schema:
		pe.collectSchemaTarget(items, currentPath+"[*]", paths, visited, stack, terminalMemo, opts, inCycle)
	case []*jsonschema.Schema:
		for idx, sub := range items {
			childPath := fmt.Sprintf("%s[%d]", currentPath, idx)
			pe.collectSchemaTarget(sub, childPath, paths, visited, stack, terminalMemo, opts, inCycle)
		}
	}

	switch addItems := schema.AdditionalItems.(type) {
	case bool:
		if addItems && !opts.TerminalsOnly {
			addPath(paths, currentPath+"[*]")
		}
	case *jsonschema.Schema:
		pe.collectSchemaTarget(addItems, currentPath+"[*]", paths, visited, stack, terminalMemo, opts, inCycle)
	}

	if schema.Items2020 != nil {
		pe.collectSchemaTarget(schema.Items2020, currentPath+"[*]", paths, visited, stack, terminalMemo, opts, inCycle)
	}

	if schema.Contains != nil {
		pe.collectSchemaTarget(schema.Contains, currentPath+"[*]", paths, visited, stack, terminalMemo, opts, inCycle)
	}

	if schema.UnevaluatedItems != nil {
		pe.collectSchemaTarget(schema.UnevaluatedItems, currentPath+"[*]", paths, visited, stack, terminalMemo, opts, inCycle)
	}

	if !opts.TerminalsOnly && hasSchemaType(schema, "array") && len(schema.PrefixItems) == 0 && schema.Items == nil && schema.Items2020 == nil && schema.Contains == nil {
		addPath(paths, currentPath+"[*]")
	}
}

func (pe *PathExtractor) collectSchemaTarget(sub *jsonschema.Schema, targetPath string, paths map[string]string, visited map[schemaVisitKey]struct{}, stack map[*jsonschema.Schema]int, terminalMemo map[*jsonschema.Schema]terminalState, opts SchemaPathOptions, inCycle bool) {
	if sub == nil {
		return
	}

	if stack[sub] == 0 {
		pe.collectSchemaPaths(sub, targetPath, paths, visited, stack, terminalMemo, opts)
		return
	}

	if inCycle {
		return
	}

	pe.collectSchemaPaths(sub, targetPath, paths, visited, stack, terminalMemo, opts)

	if strings.HasSuffix(targetPath, "{*}") {
		return
	}

	pe.collectSchemaPaths(sub, targetPath+"{*}", paths, visited, stack, terminalMemo, opts)
}

func addPath(paths map[string]string, path string) {
	key := canonicalMeaningKey(path)
	if existing, exists := paths[key]; exists {
		if shouldReplacePath(existing, path) {
			paths[key] = path
		}
		return
	}
	paths[key] = path
}

func canonicalMeaningKey(path string) string {
	return strings.ReplaceAll(path, "{*}", "")
}

func shouldReplacePath(existing, candidate string) bool {
	existingHasStar := strings.Contains(existing, "{*}")
	candidateHasStar := strings.Contains(candidate, "{*}")

	if candidateHasStar && !existingHasStar {
		return true
	}
	if existingHasStar && !candidateHasStar {
		return false
	}

	if len(candidate) < len(existing) {
		return true
	}
	if len(candidate) > len(existing) {
		return false
	}

	return candidate < existing
}

func isTerminalSchema(schema *jsonschema.Schema, memo map[*jsonschema.Schema]terminalState) bool {
	if schema == nil {
		return false
	}

	if memo == nil {
		memo = make(map[*jsonschema.Schema]terminalState)
	}

	switch memo[schema] {
	case terminalYes:
		return true
	case terminalNo:
		return false
	case terminalComputing:
		return false
	}

	memo[schema] = terminalComputing

	if hasSchemaType(schema, "object") || isObjectLike(schema) {
		memo[schema] = terminalNo
		return false
	}
	if hasSchemaType(schema, "array") || isArrayLike(schema) {
		memo[schema] = terminalNo
		return false
	}

	refs := []*jsonschema.Schema{schema.Ref, schema.RecursiveRef, schema.DynamicRef}
	for _, ref := range refs {
		if ref != nil && !isTerminalSchema(ref, memo) {
			memo[schema] = terminalNo
			return false
		}
	}

	combinations := [][]*jsonschema.Schema{schema.AllOf, schema.AnyOf, schema.OneOf}
	for _, group := range combinations {
		if len(group) == 0 {
			continue
		}
		for _, sub := range group {
			if !isTerminalSchema(sub, memo) {
				memo[schema] = terminalNo
				return false
			}
		}
	}

	if schema.If != nil {
		if schema.Then != nil && !isTerminalSchema(schema.Then, memo) {
			memo[schema] = terminalNo
			return false
		}
		if schema.Else != nil && !isTerminalSchema(schema.Else, memo) {
			memo[schema] = terminalNo
			return false
		}
	}

	if schema.ContentSchema != nil && !isTerminalSchema(schema.ContentSchema, memo) {
		memo[schema] = terminalNo
		return false
	}

	memo[schema] = terminalYes
	return true
}

func isObjectLike(schema *jsonschema.Schema) bool {
	if schema == nil {
		return false
	}

	if len(schema.Properties) > 0 || len(schema.PatternProperties) > 0 || schema.PropertyNames != nil || schema.UnevaluatedProperties != nil || len(schema.Required) > 0 || len(schema.DependentSchemas) > 0 || len(schema.DependentRequired) > 0 {
		return true
	}

	if schema.AdditionalProperties != nil {
		return true
	}

	if len(schema.Dependencies) > 0 {
		return true
	}

	if schema.MinProperties >= 0 || schema.MaxProperties >= 0 {
		return true
	}

	if schema.RegexProperties {
		return true
	}

	return false
}

func isArrayLike(schema *jsonschema.Schema) bool {
	if schema == nil {
		return false
	}

	if len(schema.PrefixItems) > 0 {
		return true
	}

	if schema.Items != nil || schema.Items2020 != nil {
		return true
	}

	if schema.AdditionalItems != nil {
		return true
	}

	if schema.Contains != nil || schema.UnevaluatedItems != nil {
		return true
	}

	if schema.MinItems >= 0 || schema.MaxItems >= 0 {
		return true
	}

	if schema.UniqueItems {
		return true
	}

	if schema.MaxContains >= 0 {
		return true
	}

	return false
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
