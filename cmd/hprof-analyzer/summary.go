package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/modbender/hprof-analyzer/internal/output"
	"github.com/modbender/hprof-analyzer/internal/parser"
	"github.com/modbender/hprof-analyzer/pkg/hprof"

	"github.com/spf13/cobra"
)

func newSummaryCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "summary <file.hprof>",
		Short: "Print HPROF file summary",
		Long:  "Stream the entire HPROF file and print header info, record counts, and heap statistics.",
		Args:  cobra.ExactArgs(1),
		RunE:  runSummary,
	}
}

func runSummary(cmd *cobra.Command, args []string) error {
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

	w, err := getOutput()
	if err != nil {
		return err
	}
	if w != os.Stdout {
		defer w.Close()
	}

	// Print header info
	ts := time.UnixMilli(int64(header.Timestamp))
	fmt.Fprintf(w, "Format:     %s\n", header.Format)
	fmt.Fprintf(w, "ID Size:    %d bytes\n", header.IDSize)
	fmt.Fprintf(w, "Timestamp:  %s\n", ts.Format(time.RFC3339))
	fmt.Fprintln(w)

	// Count records
	recordCounts := make(map[uint8]int)
	var totalRecords int
	var totalInstances, totalClasses, totalObjArrays, totalPrimArrays int
	var totalGCRoots int
	var totalHeapBytes uint64

	ctx := context.Background()
	for rec, err := range r.Records(ctx) {
		if err != nil {
			return fmt.Errorf("reading records: %w", err)
		}
		recordCounts[rec.Tag]++
		totalRecords++

		if rec.Tag == hprof.TagHeapDump || rec.Tag == hprof.TagHeapDumpSeg {
			for sub, err := range parser.ParseHeapDump(rec.Body, header.IDSize) {
				if err != nil {
					return fmt.Errorf("parsing heap dump: %w", err)
				}
				switch obj := sub.(type) {
				case hprof.ClassDump:
					totalClasses++
					_ = obj
				case hprof.InstanceDump:
					totalInstances++
					totalHeapBytes += uint64(obj.DataSize)
				case hprof.ObjectArrayDump:
					totalObjArrays++
					totalHeapBytes += uint64(obj.Length) * uint64(header.IDSize)
				case hprof.PrimitiveArrayDump:
					totalPrimArrays++
					totalHeapBytes += uint64(len(obj.Data))
				case hprof.GCRoot:
					totalGCRoots++
				}
			}
		}
	}

	// Print record type counts
	fmt.Fprintln(w, "Record Counts:")
	fmtr, err := output.NewFormatter(w, outputFmt)
	if err != nil {
		return err
	}
	fmtr.WriteHeader([]string{"Tag", "Name", "Count"})

	// Print in tag order
	for tag := uint8(0); tag < 0xFF; tag++ {
		if count, ok := recordCounts[tag]; ok {
			fmtr.WriteRow([]string{
				fmt.Sprintf("0x%02X", tag),
				hprof.TagName(tag),
				fmt.Sprintf("%d", count),
			})
		}
	}
	fmtr.Flush()

	// Print heap summary
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Heap Summary:")
	fmt.Fprintf(w, "  Classes:          %d\n", totalClasses)
	fmt.Fprintf(w, "  Instances:        %d\n", totalInstances)
	fmt.Fprintf(w, "  Object Arrays:    %d\n", totalObjArrays)
	fmt.Fprintf(w, "  Primitive Arrays: %d\n", totalPrimArrays)
	fmt.Fprintf(w, "  GC Roots:         %d\n", totalGCRoots)
	fmt.Fprintf(w, "  Total Objects:    %d\n", totalInstances+totalObjArrays+totalPrimArrays)
	fmt.Fprintf(w, "  Total Records:    %d\n", totalRecords)
	fmt.Fprintf(w, "  Heap Data Size:   %s\n", formatBytes(totalHeapBytes))

	return nil
}

func formatBytes(b uint64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)
	switch {
	case b >= GB:
		return fmt.Sprintf("%.2f GB", float64(b)/float64(GB))
	case b >= MB:
		return fmt.Sprintf("%.2f MB", float64(b)/float64(MB))
	case b >= KB:
		return fmt.Sprintf("%.2f KB", float64(b)/float64(KB))
	default:
		return fmt.Sprintf("%d B", b)
	}
}
