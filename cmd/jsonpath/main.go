package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "jsonpath",
	Short: "A JSON path expression parser with recursive structure support",
	Long: `A command-line utility for parsing and testing JSON path expressions
that support recursive structures using group operators and repetition patterns.

Examples:
  jsonpath parse "$.node.(child|meta.child){*}.value"
  jsonpath test "$.user.name" '{"user": {"name": "John"}}'`,
}

var parseCmd = &cobra.Command{
	Use:   "parse [expression]",
	Short: "Parse a JSON path expression and display its structure",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		expression := args[0]
		fmt.Printf("Parsing expression: %s\n", expression)
		fmt.Printf("Parser implementation coming soon...\n")
	},
}

var testCmd = &cobra.Command{
	Use:   "test [expression] [json]",
	Short: "Test if a JSON path expression matches the given JSON data",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		expression := args[0]
		jsonData := args[1]
		fmt.Printf("Testing expression: %s\n", expression)
		fmt.Printf("Against JSON: %s\n", jsonData)
		fmt.Printf("Test implementation coming soon...\n")
	},
}

func init() {
	rootCmd.AddCommand(parseCmd)
	rootCmd.AddCommand(testCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}