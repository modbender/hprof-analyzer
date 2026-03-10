package main

import (
	"fmt"
	"os"

	"github.com/modbender/hprof-analyzer/internal/analysis"
	"github.com/modbender/hprof-analyzer/internal/index"
	"github.com/modbender/hprof-analyzer/internal/output"

	"github.com/spf13/cobra"
)

var (
	domTop         int
	domClass       string
	domMinRetained string
)

func newDomtreeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "domtree <file.hprof>",
		Short: "Show dominator tree (objects by retained size)",
		Long:  "Compute the dominator tree and show objects sorted by retained size. Requires indexing the heap dump.",
		Args:  cobra.ExactArgs(1),
		RunE:  runDomtree,
	}
	cmd.Flags().IntVarP(&domTop, "top", "n", 20, "Show top N objects")
	cmd.Flags().StringVar(&domClass, "class", "", "Filter by class name")
	cmd.Flags().StringVar(&domMinRetained, "min-retained", "", "Minimum retained size (e.g., 1MB, 100KB)")
	return cmd
}

func runDomtree(cmd *cobra.Command, args []string) error {
	idx, err := index.EnsureIndexed(args[0])
	if err != nil {
		return err
	}

	minRetained := parseSize(domMinRetained)
	dt := analysis.NewDominatorTree(idx)
	entries := dt.Results(domTop, domClass, minRetained)

	w, err := getOutput()
	if err != nil {
		return err
	}
	if w != os.Stdout {
		defer w.Close()
	}

	fmtr, err := output.NewFormatter(w, outputFmt)
	if err != nil {
		return err
	}

	fmtr.WriteHeader([]string{"#", "Retained Size", "Shallow Size", "Object ID", "Class Name"})
	for i, e := range entries {
		fmtr.WriteRow([]string{
			fmt.Sprintf("%d", i+1),
			formatBytes(e.RetainedSize),
			formatBytes(e.ShallowSize),
			fmt.Sprintf("0x%x", e.ObjectID),
			e.ClassName,
		})
	}
	return fmtr.Flush()
}

// parseSize parses a human-readable size string like "1MB", "100KB", "1024".
func parseSize(s string) uint64 {
	if s == "" {
		return 0
	}
	var n uint64
	var unit string
	fmt.Sscanf(s, "%d%s", &n, &unit)
	switch unit {
	case "KB", "kb", "K", "k":
		return n * 1024
	case "MB", "mb", "M", "m":
		return n * 1024 * 1024
	case "GB", "gb", "G", "g":
		return n * 1024 * 1024 * 1024
	default:
		return n
	}
}
