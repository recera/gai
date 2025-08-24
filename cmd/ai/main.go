// Package main provides the AI CLI for the GAI framework.
package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var version = "0.8.0"

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "ai",
	Short: "GAI Framework CLI",
	Long: `The GAI Framework CLI provides tools for development, testing, and management
of AI applications built with the Go AI Framework.

Features:
  - Development server with SSE/NDJSON streaming
  - Prompt template verification and management
  - Example project generation
  - Model testing and evaluation`,
	Version: version,
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func init() {
	// Add global flags
	rootCmd.PersistentFlags().BoolP("verbose", "v", false, "Enable verbose output")
	rootCmd.PersistentFlags().String("config", "", "Config file (default: $HOME/.gai/config.yaml)")
}