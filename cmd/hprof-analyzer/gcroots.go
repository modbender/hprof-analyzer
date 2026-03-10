package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/modbender/hprof-analyzer/internal/analysis"
	"github.com/modbender/hprof-analyzer/internal/index"
	"github.com/modbender/hprof-analyzer/pkg/hprof"

	"github.com/spf13/cobra"
)

var (
	gcrClass    string
	gcrID       string
	gcrMaxPaths int
)

func newGCRootsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "gcroots <file.hprof>",
		Short: "Show shortest paths from GC roots to objects",
		Long:  "Find and display the shortest reference chains from GC roots to target objects. Requires indexing.",
		Args:  cobra.ExactArgs(1),
		RunE:  runGCRoots,
	}
	cmd.Flags().StringVar(&gcrClass, "class", "", "Target class name (e.g., java.util.HashMap)")
	cmd.Flags().StringVar(&gcrID, "id", "", "Target object ID (hex, e.g., 0x7f001234)")
	cmd.Flags().IntVar(&gcrMaxPaths, "max-paths", 10, "Maximum number of paths to show")
	return cmd
}

func runGCRoots(cmd *cobra.Command, args []string) error {
	if gcrClass == "" && gcrID == "" {
		return fmt.Errorf("specify --class or --id")
	}

	idx, err := index.EnsureIndexed(args[0])
	if err != nil {
		return err
	}

	var targetID uint64
	if gcrID != "" {
		id := strings.TrimPrefix(gcrID, "0x")
		targetID, err = strconv.ParseUint(id, 16, 64)
		if err != nil {
			return fmt.Errorf("invalid object ID %q: %w", gcrID, err)
		}
	}

	paths := analysis.FindGCRootPaths(idx, gcrClass, targetID, gcrMaxPaths)

	w, err := getOutput()
	if err != nil {
		return err
	}
	if w != os.Stdout {
		defer w.Close()
	}

	if len(paths) == 0 {
		fmt.Fprintln(w, "No paths found.")
		return nil
	}

	for i, p := range paths {
		rootTypeName := hprof.SubtagName(p.RootType)
		fmt.Fprintf(w, "Path %d (root type: %s):\n", i+1, rootTypeName)
		for j, node := range p.Path {
			indent := strings.Repeat("  ", j)
			fmt.Fprintf(w, "%s-> [0x%x] %s\n", indent, node.ObjectID, node.ClassName)
		}
		fmt.Fprintln(w)
	}

	return nil
}
