package main

import (
        "encoding/json"
        "fmt"
        "os"
        "strings"

        jsonpkg "jsonpath-sdk/internal/json"
        "jsonpath-sdk/internal/parser"
        "jsonpath-sdk/internal/tree"
        "github.com/spf13/cobra"
)

const version = "1.0.0"

var (
        verbose     bool
        quiet       bool
        outputJSON  bool
        prettyPrint bool
)

// logInfo prints info messages unless in JSON mode or quiet mode
func logInfo(format string, args ...interface{}) {
        if !outputJSON && !quiet {
                fmt.Fprintf(os.Stderr, format, args...)
        }
}

// readInput reads input from file or direct string
func readInput(input string) (string, error) {
        if strings.HasPrefix(input, "@") {
                filename := input[1:]
                data, err := os.ReadFile(filename)
                return string(data), err
        }
        return input, nil
}

var rootCmd = &cobra.Command{
        Use:     "jsonpath",
        Version: version,
        Short:   "A JSON path expression parser with recursive structure support",
        Long: `A command-line utility for parsing and testing JSON path expressions
that support recursive structures using group operators and repetition patterns.

Features:
  • Support for complex path expressions with groups (|) and repetition ({*})
  • Bracket notation with proper escaping
  • Fast pattern matching using trie data structure
  • JSON processing with bytedance/sonic for performance

Examples:
  jsonpath parse "$.node.(child|meta.child){*}.value"
  jsonpath test "$.user.name" '{"user": {"name": "John"}}'
  jsonpath validate '{"users": [{"name": "Alice"}, {"name": "Bob"}]}'
  jsonpath extract "$.users[*].name" data.json`,
}

var parseCmd = &cobra.Command{
        Use:   "parse [expression]",
        Short: "Parse a JSON path expression and display its structure",
        Args:  cobra.ExactArgs(1),
        Run: func(cmd *cobra.Command, args []string) {
                expression := args[0]
                
                logInfo("Parsing expression: %s\n", expression)
                
                // Parse the expression using the parser
                expr, err := parser.ParseExpression(expression)
                if err != nil {
                        fmt.Fprintf(os.Stderr, "Error parsing expression: %v\n", err)
                        os.Exit(1)
                }
                
                if outputJSON {
                        // Output as JSON structure
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
                        
                        var output []byte
                        if prettyPrint {
                                output, _ = json.MarshalIndent(result, "", "  ")
                        } else {
                                output, _ = json.Marshal(result)
                        }
                        fmt.Println(string(output))
                } else {
                        // Display the parsed structure in human-readable format
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
        },
}

var testCmd = &cobra.Command{
        Use:   "test [expression] [json]",
        Short: "Test if a JSON path expression matches the given JSON data",
        Args:  cobra.ExactArgs(2),
        Run: func(cmd *cobra.Command, args []string) {
                expression := args[0]
                jsonData := args[1]
                
                if !quiet {
                        fmt.Printf("Testing expression: %s\n", expression)
                }
                
                // Handle file input
                if strings.HasPrefix(jsonData, "@") {
                        filename := jsonData[1:]
                        data, err := ioutil.ReadFile(filename)
                        if err != nil {
                                fmt.Fprintf(os.Stderr, "Error reading file %s: %v\n", filename, err)
                                os.Exit(1)
                        }
                        jsonData = string(data)
                }
                
                // Validate and format JSON
                processor := jsonpkg.NewPathExtractor()
                if err := processor.ValidateJSON(jsonData); err != nil {
                        fmt.Fprintf(os.Stderr, "Error: Invalid JSON data: %v\n", err)
                        os.Exit(1)
                }
                
                if verbose {
                        fmt.Printf("JSON data is valid\n")
                }
                
                // Parse the expression
                expr, err := parser.ParseExpression(expression)
                if err != nil {
                        fmt.Fprintf(os.Stderr, "Error parsing expression: %v\n", err)
                        os.Exit(1)
                }
                
                // Build pattern tree
                patternTree := tree.NewPatternTree()
                if err := patternTree.AddPattern(expr); err != nil {
                        fmt.Fprintf(os.Stderr, "Error building pattern tree: %v\n", err)
                        os.Exit(1)
                }
                
                // Extract all paths from JSON
                paths, err := processor.ExtractPaths(jsonData)
                if err != nil {
                        fmt.Fprintf(os.Stderr, "Error extracting paths: %v\n", err)
                        os.Exit(1)
                }
                
                matchCount := 0
                matchingPaths := []string{}
                
                for _, path := range paths {
                        segments := processor.ConvertPathToSegments(path)
                        matches := patternTree.MatchPath(segments)
                        
                        if matches {
                                matchCount++
                                matchingPaths = append(matchingPaths, path)
                                if verbose || !quiet {
                                        fmt.Printf("  ✓ %s (matches)\n", path)
                                }
                        } else if verbose {
                                fmt.Printf("  ✗ %s\n", path)
                        }
                }
                
                if outputJSON {
                        result := map[string]interface{}{
                                "expression":     expression,
                                "total_paths":    len(paths),
                                "matching_paths": matchCount,
                                "matches":        matchingPaths,
                                "success":        matchCount > 0,
                        }
                        if verbose {
                                result["all_paths"] = paths
                        }
                        
                        var output []byte
                        if prettyPrint {
                                output, _ = json.MarshalIndent(result, "", "  ")
                        } else {
                                output, _ = json.Marshal(result)
                        }
                        fmt.Println(string(output))
                } else {
                        if !quiet {
                                fmt.Printf("\nFound %d paths in JSON:\n", len(paths))
                        }
                        
                        if !quiet {
                                fmt.Printf("\nResult: %d out of %d paths match the expression\n", matchCount, len(paths))
                                if matchCount > 0 {
                                        fmt.Printf("✓ Expression matches JSON data\n")
                                } else {
                                        fmt.Printf("✗ Expression does not match JSON data\n")
                                }
                        }
                }
                
                // Set exit code based on match result for scripting
                if matchCount == 0 {
                        os.Exit(1)
                }
        },
}

// Additional utility commands
var validateCmd = &cobra.Command{
        Use:   "validate [json]",
        Short: "Validate JSON data format",
        Args:  cobra.ExactArgs(1),
        Run: func(cmd *cobra.Command, args []string) {
                jsonData := args[0]
                
                // Handle file input
                if strings.HasPrefix(jsonData, "@") {
                        filename := jsonData[1:]
                        data, err := ioutil.ReadFile(filename)
                        if err != nil {
                                fmt.Fprintf(os.Stderr, "Error reading file %s: %v\n", filename, err)
                                os.Exit(1)
                        }
                        jsonData = string(data)
                }
                
                processor := jsonpkg.NewPathExtractor()
                if err := processor.ValidateJSON(jsonData); err != nil {
                        if !quiet {
                                fmt.Fprintf(os.Stderr, "Invalid JSON: %v\n", err)
                        }
                        os.Exit(1)
                }
                
                if !quiet {
                        fmt.Println("✓ JSON is valid")
                }
                
                if verbose {
                        // Show formatted JSON
                        formatted, err := processor.FormatJSON(jsonData)
                        if err == nil {
                                fmt.Printf("\nFormatted JSON:\n%s\n", formatted)
                        }
                }
        },
}

var extractCmd = &cobra.Command{
        Use:   "extract [expression] [json_file]",
        Short: "Extract values from JSON data using path expression",
        Args:  cobra.ExactArgs(2),
        Run: func(cmd *cobra.Command, args []string) {
                expression := args[0]
                filename := args[1]
                
                // Read JSON file
                data, err := ioutil.ReadFile(filename)
                if err != nil {
                        fmt.Fprintf(os.Stderr, "Error reading file %s: %v\n", filename, err)
                        os.Exit(1)
                }
                jsonData := string(data)
                
                processor := jsonpkg.NewPathExtractor()
                if err := processor.ValidateJSON(jsonData); err != nil {
                        fmt.Fprintf(os.Stderr, "Invalid JSON in file %s: %v\n", filename, err)
                        os.Exit(1)
                }
                
                // Parse the expression
                expr, err := parser.ParseExpression(expression)
                if err != nil {
                        fmt.Fprintf(os.Stderr, "Error parsing expression: %v\n", err)
                        os.Exit(1)
                }
                
                // Build pattern tree and extract matching paths
                patternTree := tree.NewPatternTree()
                if err := patternTree.AddPattern(expr); err != nil {
                        fmt.Fprintf(os.Stderr, "Error building pattern tree: %v\n", err)
                        os.Exit(1)
                }
                
                paths, err := processor.ExtractPaths(jsonData)
                if err != nil {
                        fmt.Fprintf(os.Stderr, "Error extracting paths: %v\n", err)
                        os.Exit(1)
                }
                
                matchingValues := []interface{}{}
                matchingPaths := []string{}
                
                for _, path := range paths {
                        segments := processor.ConvertPathToSegments(path)
                        if patternTree.MatchPath(segments) {
                                matchingPaths = append(matchingPaths, path)
                                
                                // Extract the actual value
                                value, err := processor.ExtractValue(jsonData, path)
                                if err == nil {
                                        matchingValues = append(matchingValues, value)
                                }
                        }
                }
                
                if outputJSON {
                        result := map[string]interface{}{
                                "expression": expression,
                                "file":       filename,
                                "matches":    len(matchingValues),
                                "values":     matchingValues,
                        }
                        if verbose {
                                result["paths"] = matchingPaths
                        }
                        
                        var output []byte
                        if prettyPrint {
                                output, _ = json.MarshalIndent(result, "", "  ")
                        } else {
                                output, _ = json.Marshal(result)
                        }
                        fmt.Println(string(output))
                } else {
                        if !quiet && len(matchingValues) > 0 {
                                fmt.Printf("Found %d matching values:\n", len(matchingValues))
                                for i, value := range matchingValues {
                                        fmt.Printf("[%d] %s: %v\n", i+1, matchingPaths[i], value)
                                }
                        } else if !quiet {
                                fmt.Println("No matching values found")
                        }
                }
                
                if len(matchingValues) == 0 {
                        os.Exit(1)
                }
        },
}

func init() {
        // Add global flags
        rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose output")
        rootCmd.PersistentFlags().BoolVarP(&quiet, "quiet", "q", false, "Enable quiet mode (minimal output)")
        rootCmd.PersistentFlags().BoolVarP(&outputJSON, "json", "j", false, "Output results in JSON format")
        rootCmd.PersistentFlags().BoolVarP(&prettyPrint, "pretty", "p", false, "Pretty print JSON output")
        
        // Add command-specific flags
        testCmd.Flags().StringP("file", "f", "", "Read JSON data from file")
        
        // Add all commands
        rootCmd.AddCommand(parseCmd)
        rootCmd.AddCommand(testCmd)
        rootCmd.AddCommand(validateCmd)
        rootCmd.AddCommand(extractCmd)
}

func main() {
        if err := rootCmd.Execute(); err != nil {
                fmt.Fprintf(os.Stderr, "Error: %v\n", err)
                os.Exit(1)
        }
}