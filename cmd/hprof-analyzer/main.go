package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	version    = "dev"
	commit     = "none"
	date       = "unknown"
	outputFmt  string
	outputFile string
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "hprof-analyzer",
		Short: "CLI tool for analyzing Java HPROF heap dumps",
		Long:  "A fast, MAT-level CLI tool for Java heap dump analysis. Supports streaming analysis, indexing, dominator trees, GC root paths, OQL queries, and automated leak detection.",
		Version: version,
	}

	rootCmd.PersistentFlags().StringVar(&outputFmt, "format", "table", "Output format: table, json, csv")
	rootCmd.PersistentFlags().StringVarP(&outputFile, "output", "o", "", "Output file (default: stdout)")

	rootCmd.AddCommand(
		newSummaryCmd(),
		newHistogramCmd(),
		newStringsCmd(),
		newIndexCmd(),
		newDomtreeCmd(),
		newGCRootsCmd(),
		newOQLCmd(),
		newLeaksCmd(),
		newVersionCmd(),
		newUpgradeCmd(),
	)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// getOutput returns the writer for command output based on --output flag.
func getOutput() (*os.File, error) {
	if outputFile == "" {
		return os.Stdout, nil
	}
	return os.OpenFile(outputFile, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
}
