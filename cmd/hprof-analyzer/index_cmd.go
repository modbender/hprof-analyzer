package main

import (
	"fmt"
	"os"
	"time"

	"github.com/modbender/hprof-analyzer/internal/index"

	"github.com/spf13/cobra"
)

func newIndexCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "index <file.hprof>",
		Short: "Build index files for faster analysis",
		Long:  "Parse the HPROF file and create index files (.hpai) for use by analysis commands like domtree and gcroots.",
		Args:  cobra.ExactArgs(1),
		RunE:  runIndex,
	}
}

func runIndex(cmd *cobra.Command, args []string) error {
	start := time.Now()

	idx, err := index.EnsureIndexed(args[0])
	if err != nil {
		return err
	}

	w, err := getOutput()
	if err != nil {
		return err
	}
	if w != os.Stdout {
		defer w.Close()
	}

	fmt.Fprintf(w, "Index built in %v\n", time.Since(start).Round(time.Millisecond))
	fmt.Fprintf(w, "  Objects:  %d\n", len(idx.Objects))
	fmt.Fprintf(w, "  Classes:  %d\n", len(idx.Classes))
	fmt.Fprintf(w, "  GC Roots: %d\n", len(idx.Roots))
	fmt.Fprintf(w, "  Strings:  %d\n", len(idx.Strings))

	return nil
}
