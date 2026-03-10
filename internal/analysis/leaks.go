package analysis

import (
	"fmt"
	"sort"

	"github.com/modbender/hprof-analyzer/internal/index"
)

// LeakSuspect represents a potential memory leak.
type LeakSuspect struct {
	ObjectID        uint64
	ClassName       string
	RetainedSize    uint64
	RetainedPercent float64
	ShallowSize     uint64
	Description     string
	AccumulationPoint string
}

// FindLeakSuspects analyzes the dominator tree to find potential memory leaks.
// It flags objects retaining more than threshold percent of the total heap.
func FindLeakSuspects(idx *index.Index, threshold float64) []LeakSuspect {
	if threshold <= 0 {
		threshold = 10.0
	}

	dt := NewDominatorTree(idx)

	// Calculate total heap size
	var totalHeapSize uint64
	for _, obj := range idx.Objects {
		totalHeapSize += obj.ShallowSize
	}

	if totalHeapSize == 0 {
		return nil
	}

	// Get all objects sorted by retained size
	entries := dt.Results(0, "", 0)

	var suspects []LeakSuspect
	for _, entry := range entries {
		percent := float64(entry.RetainedSize) / float64(totalHeapSize) * 100
		if percent < threshold {
			continue
		}

		suspect := LeakSuspect{
			ObjectID:        entry.ObjectID,
			ClassName:       entry.ClassName,
			RetainedSize:    entry.RetainedSize,
			RetainedPercent: percent,
			ShallowSize:     entry.ShallowSize,
		}

		// Determine description
		suspect.Description = fmt.Sprintf(
			"%s retains %.1f%% of the heap (%s retained, %s shallow)",
			entry.ClassName,
			percent,
			formatSize(entry.RetainedSize),
			formatSize(entry.ShallowSize),
		)

		// Find accumulation point: the largest single child that dominates
		// a significant portion of the retained size
		suspect.AccumulationPoint = findAccumulationPoint(idx, dt, entry.ObjectID, entry.RetainedSize)

		suspects = append(suspects, suspect)
	}

	sort.Slice(suspects, func(i, j int) bool {
		return suspects[i].RetainedSize > suspects[j].RetainedSize
	})

	return suspects
}

// findAccumulationPoint identifies where objects accumulate under a suspect.
func findAccumulationPoint(idx *index.Index, dt *DominatorTree, objID uint64, retainedSize uint64) string {
	// Look at outbound refs to find the child with the largest retained size
	refs := idx.OutRefs[objID]
	if len(refs) == 0 {
		return ""
	}

	var maxChildRetained uint64
	var maxChildID uint64
	for _, refID := range refs {
		childRetained := dt.retainedSize[refID]
		if childRetained > maxChildRetained {
			maxChildRetained = childRetained
			maxChildID = refID
		}
	}

	if maxChildRetained == 0 {
		return ""
	}

	// If the largest child retains significantly less than the object itself,
	// this object is an accumulation point
	diff := retainedSize - maxChildRetained
	if diff > retainedSize/4 { // >25% is in direct children
		return fmt.Sprintf("Accumulates in %s (0x%x)", idx.ObjectClassName(objID), objID)
	}

	return fmt.Sprintf("Dominates through %s (0x%x, %s retained)",
		idx.ObjectClassName(maxChildID), maxChildID, formatSize(maxChildRetained))
}

func formatSize(b uint64) string {
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
