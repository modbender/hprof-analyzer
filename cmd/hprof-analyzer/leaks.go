package main

import (
	"fmt"
	"os"

	"github.com/modbender/hprof-analyzer/internal/analysis"
	"github.com/modbender/hprof-analyzer/internal/index"

	"github.com/spf13/cobra"
)

var leaksThreshold float64

func newLeaksCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "leaks <file.hprof>",
		Short: "Detect potential memory leak suspects",
		Long:  "Analyze the dominator tree to find objects retaining a large percentage of the heap, suggesting potential memory leaks.",
		Args:  cobra.ExactArgs(1),
		RunE:  runLeaks,
	}
	cmd.Flags().Float64Var(&leaksThreshold, "threshold", 10.0, "Minimum heap percentage to flag as suspect")
	return cmd
}

func runLeaks(cmd *cobra.Command, args []string) error {
	idx, err := index.EnsureIndexed(args[0])
	if err != nil {
		return err
	}

	suspects := analysis.FindLeakSuspects(idx, leaksThreshold)

	w, err := getOutput()
	if err != nil {
		return err
	}
	if w != os.Stdout {
		defer w.Close()
	}

	if len(suspects) == 0 {
		fmt.Fprintln(w, "No leak suspects found above threshold.")
		return nil
	}

	fmt.Fprintf(w, "Leak Suspects Report (%d found):\n", len(suspects))
	fmt.Fprintln(w, "================================")
	fmt.Fprintln(w)

	for i, s := range suspects {
		fmt.Fprintf(w, "Suspect %d:\n", i+1)
		fmt.Fprintf(w, "  Object:    [0x%x] %s\n", s.ObjectID, s.ClassName)
		fmt.Fprintf(w, "  Retained:  %s (%.1f%% of heap)\n", formatBytes(s.RetainedSize), s.RetainedPercent)
		fmt.Fprintf(w, "  Shallow:   %s\n", formatBytes(s.ShallowSize))
		fmt.Fprintf(w, "  %s\n", s.Description)
		if s.AccumulationPoint != "" {
			fmt.Fprintf(w, "  %s\n", s.AccumulationPoint)
		}
		fmt.Fprintln(w)
	}

	return nil
}
