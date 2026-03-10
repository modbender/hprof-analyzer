package analysis

import (
	"sort"
)

// HistogramEntry represents a single class in the histogram.
type HistogramEntry struct {
	ClassName    string
	InstanceCount int
	ShallowSize  uint64
}

// Histogram accumulates instance counts and sizes per class.
type Histogram struct {
	entries map[uint64]*HistogramEntry // keyed by class object ID
}

// NewHistogram creates a new histogram accumulator.
func NewHistogram() *Histogram {
	return &Histogram{
		entries: make(map[uint64]*HistogramEntry),
	}
}

// Add records an instance of the given class.
func (h *Histogram) Add(classObjID uint64, className string, shallowSize uint64) {
	e, ok := h.entries[classObjID]
	if !ok {
		e = &HistogramEntry{ClassName: className}
		h.entries[classObjID] = e
	}
	e.InstanceCount++
	e.ShallowSize += shallowSize
}

// Results returns sorted histogram entries.
// sortBy can be "size" (default) or "count".
func (h *Histogram) Results(sortBy string) []HistogramEntry {
	results := make([]HistogramEntry, 0, len(h.entries))
	for _, e := range h.entries {
		results = append(results, *e)
	}

	switch sortBy {
	case "count":
		sort.Slice(results, func(i, j int) bool {
			return results[i].InstanceCount > results[j].InstanceCount
		})
	default: // "size"
		sort.Slice(results, func(i, j int) bool {
			return results[i].ShallowSize > results[j].ShallowSize
		})
	}

	return results
}
