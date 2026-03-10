package main

import (
	"context"
	"fmt"
	"os"

	"github.com/modbender/hprof-analyzer/internal/analysis"
	"github.com/modbender/hprof-analyzer/internal/output"
	"github.com/modbender/hprof-analyzer/internal/parser"
	"github.com/modbender/hprof-analyzer/pkg/hprof"

	"github.com/spf13/cobra"
)

var (
	strFilter    string
	strTop       int
	strMinLength int
)

func newStringsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "strings <file.hprof>",
		Short: "Search and list strings from the heap dump",
		Long:  "Stream UTF8 string records from the HPROF file. Optionally filter by pattern and minimum length.",
		Args:  cobra.ExactArgs(1),
		RunE:  runStrings,
	}
	cmd.Flags().StringVar(&strFilter, "filter", "", "Filter strings by regex or substring")
	cmd.Flags().IntVarP(&strTop, "top", "n", 50, "Show only top N strings (0 = all)")
	cmd.Flags().IntVar(&strMinLength, "min-length", 0, "Minimum string length to include")
	return cmd
}

func runStrings(cmd *cobra.Command, args []string) error {
	f, err := os.Open(args[0])
	if err != nil {
		return fmt.Errorf("opening file: %w", err)
	}
	defer f.Close()

	r := parser.NewReader(f)
	header, err := r.ReadHeader()
	if err != nil {
		return fmt.Errorf("reading header: %w", err)
	}

	collector := analysis.NewStringCollector()

	ctx := context.Background()
	for rec, err := range r.Records(ctx) {
		if err != nil {
			return fmt.Errorf("reading records: %w", err)
		}
		if rec.Tag == hprof.TagUTF8 {
			id, s, err := parser.ParseUTF8(rec.Body, header.IDSize)
			if err != nil {
				return err
			}
			collector.Add(id, s)
		}
	}

	results, err := collector.Results(strFilter, strMinLength, strTop)
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

	fmtr, err := output.NewFormatter(w, outputFmt)
	if err != nil {
		return err
	}

	fmtr.WriteHeader([]string{"#", "ID", "Length", "Value"})
	for i, e := range results {
		val := e.Value
		if len(val) > 120 {
			val = val[:117] + "..."
		}
		fmtr.WriteRow([]string{
			fmt.Sprintf("%d", i+1),
			fmt.Sprintf("0x%x", e.ID),
			fmt.Sprintf("%d", len(e.Value)),
			val,
		})
	}
	return fmtr.Flush()
}
