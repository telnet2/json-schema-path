package main

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	jsonpkg "github.com/telnet2/json-schema-path/json"
	"github.com/telnet2/json-schema-path/parser"
	"github.com/telnet2/json-schema-path/spec"
	"github.com/telnet2/json-schema-path/tree"

	"github.com/spf13/cobra"
)

const version = "1.0.0"

var (
	verbose             bool
	quiet               bool
	outputJSON          bool
	prettyPrint         bool
	schemaTerminalsOnly bool
)

// Constants for magic values
const (
	filePrefix = "@"
)

// Helper functions for improved readability

// readJSONFile reads JSON data from a file, handling the @ prefix convention
func readJSONFile(filename string) (string, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return "", fmt.Errorf("reading file %s: %w", filename, err)
	}
	return string(data), nil
}

// formatOutput formats data as JSON with optional pretty printing
func formatOutput(data interface{}) (string, error) {
	var output []byte
	var err error
	
	if prettyPrint {
		output, err = json.MarshalIndent(data, "", "  ")
	} else {
		output, err = json.Marshal(data)
	}
	
	if err != nil {
		return "", fmt.Errorf("formatting JSON output: %w", err)
	}
	return string(output), nil
}

// handleFileInput checks if input starts with @ and reads from file if needed
func handleFileInput(input string) (string, error) {
	if !strings.HasPrefix(input, filePrefix) {
		return input, nil
	}
	
	filename := strings.TrimPrefix(input, filePrefix)
	return readJSONFile(filename)
}

// exitWithError prints error message and exits with code 1
func exitWithError(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}

// logInfo prints info messages unless in JSON mode or quiet mode
func logInfo(format string, args ...interface{}) {
	if !outputJSON && !quiet {
		fmt.Fprintf(os.Stderr, format, args...)
	}
}

var rootCmd = &cobra.Command{
	Use:     "schemapath",
	Version: version,
	Short:   "A  expression parser with recursive structure support",
	Long: `A command-line utility for parsing and testing  expressions
that support recursive JSON schema structures using group operators and repetition patterns.

Features:
  • Support for complex path expressions with groups (|) and repetition ({*})
  • Recursive JSON schema structure navigation
  • Bracket notation with proper escaping
  • Fast pattern matching using trie data structure
  • High-performance JSON processing with bytedance/sonic AST

Examples:
  schemapath parse "$.node.(child|meta.child){*}.value"
  schemapath test "$.user.name" '{"user": {"name": "John"}}'
  schemapath extract "$.users[*].name" data.json`,
}

var parseCmd = &cobra.Command{
	Use:   "parse [expression]",
	Short: "Parse a  expression and display its structure",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		expression := args[0]
		logInfo("Parsing expression: %s\n", expression)

		expr, err := parser.ParseExpression(expression)
		if err != nil {
			exitWithError("Error parsing expression: %v", err)
		}

		if outputJSON {
			result := createParseResult(expression, expr)
			output, err := formatOutput(result)
			if err != nil {
				exitWithError("Error formatting output: %v", err)
			}
			fmt.Println(output)
		} else {
			displayParseResult(expr)
		}
	},
}

// createParseResult creates a structured result for JSON output
func createParseResult(expression string, expr *spec.PathExpression) map[string]interface{} {
	result := map[string]interface{}{
		"expression":    expression,
		"parsed":        expr.String(),
		"root":          expr.Root.String(),
		"segment_count": len(expr.Segments),
		"segments":      make([]string, len(expr.Segments)),
	}
	for i, segment := range expr.Segments {
		result["segments"].([]string)[i] = segment.String()
	}
	return result
}

// displayParseResult displays parsing results in human-readable format
func displayParseResult(expr *spec.PathExpression) {
	fmt.Printf("Parsed structure: %s\n", expr.String())
	fmt.Printf("Root: %s\n", expr.Root.String())
	fmt.Printf("Segments (%d):\n", len(expr.Segments))
	for i, segment := range expr.Segments {
		fmt.Printf("  [%d]: %s\n", i, segment.String())
	}
	if verbose {
		fmt.Printf("\nExpression validation: ✓ Valid\n")
	}
}

var testCmd = &cobra.Command{
	Use:   "test [expression] [json]",
	Short: "Test if a  expression matches the given JSON data",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		expression := args[0]
		jsonData := args[1]

		if !quiet {
			fmt.Printf("Testing expression: %s\n", expression)
		}

		// Handle file input and validation
		processedData, err := handleFileInput(jsonData)
		if err != nil {
			exitWithError("Error handling file input: %v", err)
		}

		processor := jsonpkg.NewPathExtractor()
		if err := processor.ValidateJSON(processedData); err != nil {
			exitWithError("Error: Invalid JSON data: %v", err)
		}

		if verbose {
			fmt.Printf("JSON data is valid\n")
		}

		// Test expression matching
		matchResult, err := testExpressionMatching(expression, processedData, processor)
		if err != nil {
			exitWithError("Error testing expression: %v", err)
		}

		// Output results
		if outputJSON {
			outputResultsJSON(expression, matchResult)
		} else {
			outputResultsText(expression, matchResult)
		}

		// Set exit code based on match result for scripting
		if matchResult.matchCount == 0 {
			os.Exit(1)
		}
	},
}

// matchResult holds the results of expression matching
type matchResult struct {
	expression     string
	totalPaths     int
	matchCount     int
	matchingPaths  []string
	allPaths       []string
}

// testExpressionMatching tests if an expression matches JSON data
func testExpressionMatching(expression string, jsonData string, processor *jsonpkg.PathExtractor) (*matchResult, error) {
	expr, err := parser.ParseExpression(expression)
	if err != nil {
		return nil, fmt.Errorf("parsing expression: %w", err)
	}

	patternTree := tree.NewPatternTree()
	if err := patternTree.AddPattern(expr); err != nil {
		return nil, fmt.Errorf("building pattern tree: %w", err)
	}

	paths, err := processor.ExtractPaths(jsonData)
	if err != nil {
		return nil, fmt.Errorf("extracting paths: %w", err)
	}

	result := &matchResult{
		expression: expression,
		totalPaths: len(paths),
		allPaths:   paths,
	}

	for _, path := range paths {
		segments := processor.ConvertPathToSegments(path)
		matches := patternTree.MatchSegments(segments)

		if matches {
			result.matchCount++
			result.matchingPaths = append(result.matchingPaths, path)
			if verbose || !quiet {
				fmt.Printf("  ✓ %s (matches)\n", path)
			}
		} else if verbose {
			fmt.Printf("  ✗ %s\n", path)
		}
	}

	return result, nil
}

// outputResultsJSON outputs results in JSON format
func outputResultsJSON(expression string, result *matchResult) {
	output := map[string]interface{}{
		"expression":     expression,
		"total_paths":    result.totalPaths,
		"matching_paths": result.matchCount,
		"matches":        result.matchingPaths,
		"success":        result.matchCount > 0,
	}
	if verbose {
		output["all_paths"] = result.allPaths
	}

	formatted, err := formatOutput(output)
	if err != nil {
		exitWithError("Error formatting JSON output: %v", err)
	}
	fmt.Println(formatted)
}

// outputResultsText outputs results in text format
func outputResultsText(expression string, result *matchResult) {
	if !quiet {
		fmt.Printf("\nFound %d paths in JSON:\n", result.totalPaths)
		fmt.Printf("\nResult: %d out of %d paths match the expression\n", 
			result.matchCount, result.totalPaths)
		if result.matchCount > 0 {
			fmt.Printf("✓ Expression matches JSON data\n")
		} else {
			fmt.Printf("✗ Expression does not match JSON data\n")
		}
	}
}


var extractCmd = &cobra.Command{
	Use:   "extract [expression] [json_file]",
	Short: "Extract values from JSON data using  expression",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		expression := args[0]
		filename := args[1]

		jsonData, err := readJSONFile(filename)
		if err != nil {
			exitWithError("Error reading file: %v", err)
		}

		processor := jsonpkg.NewPathExtractor()
		if err := processor.ValidateJSON(jsonData); err != nil {
			exitWithError("Invalid JSON in file %s: %v", filename, err)
		}

		values, paths, err := extractValuesFromJSON(expression, jsonData, processor)
		if err != nil {
			exitWithError("Error extracting values: %v", err)
		}

		if outputJSON {
			outputExtractResultsJSON(expression, filename, values, paths)
		} else {
			outputExtractResultsText(values, paths)
		}

		if len(values) == 0 {
			os.Exit(1)
		}
	},
}

// extractValuesFromJSON extracts values matching the expression from JSON data
func extractValuesFromJSON(expression string, jsonData string, processor *jsonpkg.PathExtractor) ([]interface{}, []string, error) {
	expr, err := parser.ParseExpression(expression)
	if err != nil {
		return nil, nil, fmt.Errorf("parsing expression: %w", err)
	}

	patternTree := tree.NewPatternTree()
	if err := patternTree.AddPattern(expr); err != nil {
		return nil, nil, fmt.Errorf("building pattern tree: %w", err)
	}

	paths, err := processor.ExtractPaths(jsonData)
	if err != nil {
		return nil, nil, fmt.Errorf("extracting paths: %w", err)
	}

	var values []interface{}
	var matchingPaths []string

	for _, path := range paths {
		segments := processor.ConvertPathToSegments(path)
		if patternTree.MatchSegments(segments) {
			value, err := processor.ExtractValue(jsonData, path)
			if err == nil {
				values = append(values, value)
				matchingPaths = append(matchingPaths, path)
			}
		}
	}

	return values, matchingPaths, nil
}

// outputExtractResultsJSON outputs extraction results in JSON format
func outputExtractResultsJSON(expression, filename string, values []interface{}, paths []string) {
	result := map[string]interface{}{
		"expression": expression,
		"file":       filename,
		"matches":    len(values),
		"values":     values,
	}
	if verbose {
		result["paths"] = paths
	}

	output, err := formatOutput(result)
	if err != nil {
		exitWithError("Error formatting output: %v", err)
	}
	fmt.Println(output)
}

// outputExtractResultsText outputs extraction results in text format
func outputExtractResultsText(values []interface{}, paths []string) {
	if !quiet && len(values) > 0 {
		fmt.Printf("Found %d matching values:\n", len(values))
		for i, value := range values {
			fmt.Printf("[%d] %s: %v\n", i+1, paths[i], value)
		}
	} else if !quiet {
		fmt.Println("No matching values found")
	}
}

var schemaCmd = &cobra.Command{
	Use:   "schema [schema_file]",
	Short: "Generate all schema paths from a JSON Schema document",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		filename := args[0]

		schemaData, err := readJSONFile(filename)
		if err != nil {
			exitWithError("Error reading schema file: %v", err)
		}

		paths, err := extractSchemaPaths(schemaData, filename)
		if err != nil {
			exitWithError("Error generating schema paths: %v", err)
		}

		if outputJSON {
			outputSchemaResultsJSON(filename, paths)
		} else {
			outputSchemaResultsText(paths)
		}
	},
}

// extractSchemaPaths extracts all schema paths from a schema file
func extractSchemaPaths(schemaData string, filename string) ([]string, error) {
	processor := jsonpkg.NewPathExtractor()

	absPath, err := filepath.Abs(filename)
	if err != nil {
		return nil, fmt.Errorf("resolving schema path: %w", err)
	}

	schemaPath := filepath.ToSlash(absPath)
	if !strings.HasPrefix(schemaPath, "/") {
		schemaPath = "/" + schemaPath
	}

	schemaURL := url.URL{Scheme: "file", Path: schemaPath}
	opts := jsonpkg.SchemaPathOptions{TerminalsOnly: schemaTerminalsOnly}

	paths, err := processor.ExtractSchemaPathsWithOptions(schemaData, schemaURL.String(), opts)
	if err != nil {
		return nil, fmt.Errorf("extracting schema paths: %w", err)
	}

	return paths, nil
}

// outputSchemaResultsJSON outputs schema results in JSON format
func outputSchemaResultsJSON(filename string, paths []string) {
	result := map[string]interface{}{
		"file":        filename,
		"total_paths": len(paths),
		"paths":       paths,
	}

	output, err := formatOutput(result)
	if err != nil {
		exitWithError("Error formatting output: %v", err)
	}
	fmt.Println(output)
}

// outputSchemaResultsText outputs schema results in text format
func outputSchemaResultsText(paths []string) {
	if !quiet {
		fmt.Printf("Found %d schema paths:\n", len(paths))
	}
	for _, path := range paths {
		fmt.Println(path)
	}
}

func init() {
	// Add global flags
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose output")
	rootCmd.PersistentFlags().BoolVarP(&quiet, "quiet", "q", false, "Enable quiet mode (minimal output)")
	rootCmd.PersistentFlags().BoolVarP(&outputJSON, "json", "j", false, "Output results in JSON format")
	rootCmd.PersistentFlags().BoolVarP(&prettyPrint, "pretty", "p", false, "Pretty print JSON output")

	// Add command-specific flags
	testCmd.Flags().StringP("file", "f", "", "Read JSON data from file")
	schemaCmd.Flags().BoolVar(&schemaTerminalsOnly, "terminals-only", false, "Only emit terminal value schemas (exclude objects and arrays)")

	// Add all commands
	rootCmd.AddCommand(parseCmd)
	rootCmd.AddCommand(testCmd)
	rootCmd.AddCommand(extractCmd)
	rootCmd.AddCommand(schemaCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
