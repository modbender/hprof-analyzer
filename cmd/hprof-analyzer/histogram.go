package main

import (
	"context"
	"fmt"
	"os"
	"regexp"

	"github.com/modbender/hprof-analyzer/internal/analysis"
	"github.com/modbender/hprof-analyzer/internal/output"
	"github.com/modbender/hprof-analyzer/internal/parser"
	"github.com/modbender/hprof-analyzer/pkg/hprof"

	"github.com/spf13/cobra"
)

var (
	histTop    int
	histSort   string
	histFilter string
)

func newHistogramCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "histogram <file.hprof>",
		Short: "Show class histogram (instance count and size)",
		Long:  "Stream the HPROF file and produce a class histogram showing instance count and shallow size per class, sorted by size.",
		Args:  cobra.ExactArgs(1),
		RunE:  runHistogram,
	}
	cmd.Flags().IntVarP(&histTop, "top", "n", 0, "Show only top N classes (0 = all)")
	cmd.Flags().StringVar(&histSort, "sort", "size", "Sort by: size, count")
	cmd.Flags().StringVar(&histFilter, "filter", "", "Filter classes by regex pattern")
	return cmd
}

func runHistogram(cmd *cobra.Command, args []string) error {
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

	// Build string table and class name map in a single pass
	strings := make(map[uint64]string)      // string ID -> string value
	classNames := make(map[uint64]uint64)    // class obj ID -> class name string ID
	classSizes := make(map[uint64]uint32)    // class obj ID -> instance size

	hist := analysis.NewHistogram()

	ctx := context.Background()
	for rec, err := range r.Records(ctx) {
		if err != nil {
			return fmt.Errorf("reading records: %w", err)
		}

		switch rec.Tag {
		case hprof.TagUTF8:
			id, s, err := parser.ParseUTF8(rec.Body, header.IDSize)
			if err != nil {
				return err
			}
			strings[id] = s

		case hprof.TagLoadClass:
			lc, err := parser.ParseLoadClass(rec.Body, header.IDSize)
			if err != nil {
				return err
			}
			classNames[lc.ClassObjID] = lc.ClassNameID

		case hprof.TagHeapDump, hprof.TagHeapDumpSeg:
			for sub, err := range parser.ParseHeapDump(rec.Body, header.IDSize) {
				if err != nil {
					return err
				}
				switch obj := sub.(type) {
				case hprof.ClassDump:
					classSizes[obj.ClassObjID] = obj.InstanceSize
				case hprof.InstanceDump:
					nameID := classNames[obj.ClassObjID]
					name := strings[nameID]
					if name == "" {
						name = fmt.Sprintf("<class@0x%x>", obj.ClassObjID)
					}
					hist.Add(obj.ClassObjID, javaClassName(name), uint64(obj.DataSize))
				case hprof.ObjectArrayDump:
					nameID := classNames[obj.ElementClassID]
					name := strings[nameID]
					if name == "" {
						name = fmt.Sprintf("<class@0x%x>", obj.ElementClassID)
					}
					hist.Add(obj.ElementClassID, javaClassName(name)+"[]", uint64(obj.Length)*uint64(header.IDSize))
				case hprof.PrimitiveArrayDump:
					name := obj.ElementType.Name() + "[]"
					// Use element type as a pseudo class ID
					classID := uint64(obj.ElementType) | (1 << 63)
					hist.Add(classID, name, uint64(len(obj.Data)))
				}
			}
		}
	}

	results := hist.Results(histSort)

	// Apply filter
	if histFilter != "" {
		re, err := regexp.Compile(histFilter)
		if err != nil {
			return fmt.Errorf("invalid filter regex: %w", err)
		}
		filtered := results[:0]
		for _, e := range results {
			if re.MatchString(e.ClassName) {
				filtered = append(filtered, e)
			}
		}
		results = filtered
	}

	// Apply top limit
	if histTop > 0 && len(results) > histTop {
		results = results[:histTop]
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

	fmtr.WriteHeader([]string{"#", "Instances", "Shallow Size", "Class Name"})
	for i, e := range results {
		fmtr.WriteRow([]string{
			fmt.Sprintf("%d", i+1),
			fmt.Sprintf("%d", e.InstanceCount),
			formatBytes(e.ShallowSize),
			e.ClassName,
		})
	}
	return fmtr.Flush()
}

// javaClassName converts JVM internal class names (e.g., "java/lang/String")
// to dotted form (e.g., "java.lang.String").
func javaClassName(name string) string {
	result := make([]byte, len(name))
	for i := range name {
		if name[i] == '/' {
			result[i] = '.'
		} else {
			result[i] = name[i]
		}
	}
	return string(result)
}
