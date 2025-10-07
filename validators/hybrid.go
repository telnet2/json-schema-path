package validators

import (
	"encoding/json"
	"regexp"
	"strings"
	"time"

	"github.com/tidwall/gjson"

	jsonpkg "github.com/telnet2/json-schema-path/json"
	"github.com/telnet2/json-schema-path/parser"
	"github.com/telnet2/json-schema-path/tree"
)

// PatternComplexity classifies how complex a pattern is
type PatternComplexity int

const (
	// SimplePattern can be handled by gjson: $.a.b.c, $.arr[*].field
	SimplePattern PatternComplexity = iota
	// ComplexPattern requires our engine: {*}, [#*pattern], [~regex], (a|b)
	ComplexPattern
)

// HybridValidator intelligently chooses between gjson (fast, simple) and our engine (powerful, complex)
type HybridValidator struct {
	simplePatterns  map[string]simplePatternInfo  // Patterns gjson can handle
	complexPatterns map[string]complexPatternInfo // Patterns requiring our engine
	matcher         *CompiledMatcher              // For complex patterns
	processor       *jsonpkg.PathExtractor
}

type simplePatternInfo struct {
	gjsonPath string          // Converted to gjson syntax
	metadata  json.RawMessage
}

type complexPatternInfo struct {
	patternTree *tree.PatternTree
	metadata    json.RawMessage
}

// NewHybridValidator creates a new hybrid validator that uses the best strategy for each pattern
func NewHybridValidator(patterns map[string]json.RawMessage) (*HybridValidator, error) {
	simplePatterns := make(map[string]simplePatternInfo)
	complexPatterns := make(map[string]complexPatternInfo)
	complexPatternMap := make(map[string]json.RawMessage) // For CompiledMatcher

	for patternStr, metadata := range patterns {
		complexity := classifyPattern(patternStr)

		if complexity == SimplePattern {
			// Convert to gjson syntax
			gjsonPath, err := convertToGJSON(patternStr)
			if err == nil {
				simplePatterns[patternStr] = simplePatternInfo{
					gjsonPath: gjsonPath,
					metadata:  metadata,
				}
				continue
			}
			// If conversion fails, treat as complex
		}

		// Complex pattern - use our engine
		expr, err := parser.ParseExpression(patternStr)
		if err != nil {
			return nil, err
		}

		patternTree := tree.NewPatternTree()
		patternTree.AddPattern(expr)

		complexPatterns[patternStr] = complexPatternInfo{
			patternTree: patternTree,
			metadata:    metadata,
		}
		complexPatternMap[patternStr] = metadata
	}

	// Create compiled matcher for complex patterns
	var matcher *CompiledMatcher
	var err error
	if len(complexPatternMap) > 0 {
		matcher, err = NewCompiledMatcher(complexPatternMap)
		if err != nil {
			return nil, err
		}
	}

	return &HybridValidator{
		simplePatterns:  simplePatterns,
		complexPatterns: complexPatterns,
		matcher:         matcher,
		processor:       jsonpkg.NewPathExtractor(),
	}, nil
}

// Validate validates JSON data using the optimal strategy for each pattern
func (hv *HybridValidator) Validate(jsonData string) (*ValidationReport, error) {
	results := make([]ValidationResult, 0, 32)
	timestamp := time.Now()

	// Fast path: Use gjson for simple patterns
	for patternStr, info := range hv.simplePatterns {
		gjsonResults := gjson.Get(jsonData, info.gjsonPath)

		if gjsonResults.Exists() {
			// Handle both single value and array results
			if gjsonResults.IsArray() {
				gjsonResults.ForEach(func(key, value gjson.Result) bool {
					// Construct the full path
					path := constructPath(patternStr, key.String())
					result := ValidationResult{
						Path:        path,
						Value:       value.Value(),
						Metadata:    info.metadata,
						Timestamp:   timestamp,
						Valid:       true,
						Description: "Matched simple pattern (gjson): " + patternStr,
					}
					results = append(results, result)
					return true
				})
			} else {
				result := ValidationResult{
					Path:        patternStr,
					Value:       gjsonResults.Value(),
					Metadata:    info.metadata,
					Timestamp:   timestamp,
					Valid:       true,
					Description: "Matched simple pattern (gjson): " + patternStr,
				}
				results = append(results, result)
			}
		}
	}

	// Full power: Use our engine for complex patterns
	if hv.matcher != nil {
		err := hv.processor.StreamingWalk(jsonData, func(path string, value interface{}) error {
			pattern, metadata := hv.matcher.Match(path)
			if pattern != "" {
				result := ValidationResult{
					Path:        path,
					Value:       value,
					Metadata:    metadata,
					Timestamp:   timestamp,
					Valid:       true,
					Description: "Matched complex pattern: " + pattern,
				}
				results = append(results, result)
			}
			return nil
		})

		if err != nil {
			return nil, err
		}
	}

	return &ValidationReport{
		Results:    results,
		ValidPaths: len(results),
	}, nil
}

// classifyPattern determines if a pattern is simple (gjson-compatible) or complex
func classifyPattern(pattern string) PatternComplexity {
	// Check for complex features
	if strings.Contains(pattern, "{*}") {
		return ComplexPattern // Repetition operator
	}
	if strings.Contains(pattern, "[#") {
		return ComplexPattern // Property wildcard
	}
	if strings.Contains(pattern, "[~") {
		return ComplexPattern // Regex pattern
	}
	if strings.Contains(pattern, "(") && strings.Contains(pattern, "|") {
		return ComplexPattern // Group alternatives
	}

	// Check if it's just basic property access and array wildcards
	// $.a.b.c[*].d is simple
	// Everything else we've seen is simple for gjson
	return SimplePattern
}

// convertToGJSON converts our pattern syntax to gjson syntax
// $.users[*].email -> users.#.email
// $.user.profile.name -> user.profile.name
func convertToGJSON(pattern string) (string, error) {
	// Remove leading $
	if strings.HasPrefix(pattern, "$") {
		pattern = pattern[1:]
	}
	if strings.HasPrefix(pattern, ".") {
		pattern = pattern[1:]
	}

	// Replace [*] with .#
	pattern = strings.ReplaceAll(pattern, "[*]", ".#")

	// Replace [number] with .number
	re := regexp.MustCompile(`\[(\d+)\]`)
	pattern = re.ReplaceAllString(pattern, ".$1")

	return pattern, nil
}

// constructPath builds a full path from a pattern and result key
func constructPath(pattern string, key string) string {
	// For simple patterns like $.users[*].email where gjson returns array results,
	// we need to construct the actual path like $.users[0].email
	// This is a simplified version - a full implementation would be more sophisticated
	return strings.ReplaceAll(pattern, "[*]", "["+key+"]")
}

// GetSimplePatternCount returns the number of patterns being handled by gjson
func (hv *HybridValidator) GetSimplePatternCount() int {
	return len(hv.simplePatterns)
}

// GetComplexPatternCount returns the number of patterns being handled by our engine
func (hv *HybridValidator) GetComplexPatternCount() int {
	return len(hv.complexPatterns)
}

// GetStrategyBreakdown returns a breakdown of which patterns use which strategy
func (hv *HybridValidator) GetStrategyBreakdown() map[string]string {
	breakdown := make(map[string]string)

	for pattern := range hv.simplePatterns {
		breakdown[pattern] = "gjson (simple)"
	}

	for pattern := range hv.complexPatterns {
		breakdown[pattern] = "our-engine (complex)"
	}

	return breakdown
}
