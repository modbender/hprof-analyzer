package analysis

import (
	"regexp"
	"sort"
	"strings"
)

// StringEntry represents a string found in the heap dump.
type StringEntry struct {
	ID    uint64
	Value string
}

// StringCollector accumulates and filters strings from HPROF records.
type StringCollector struct {
	entries []StringEntry
}

// NewStringCollector creates a new string collector.
func NewStringCollector() *StringCollector {
	return &StringCollector{}
}

// Add records a string.
func (sc *StringCollector) Add(id uint64, value string) {
	sc.entries = append(sc.entries, StringEntry{ID: id, Value: value})
}

// Results returns filtered and sorted strings.
func (sc *StringCollector) Results(filter string, minLength int, top int) ([]StringEntry, error) {
	var results []StringEntry
	var re *regexp.Regexp

	if filter != "" {
		var err error
		re, err = regexp.Compile(filter)
		if err != nil {
			// Fall back to substring match
			re = nil
		}
	}

	for _, e := range sc.entries {
		if minLength > 0 && len(e.Value) < minLength {
			continue
		}
		if filter != "" {
			if re != nil {
				if !re.MatchString(e.Value) {
					continue
				}
			} else if !strings.Contains(e.Value, filter) {
				continue
			}
		}
		results = append(results, e)
	}

	// Sort by length descending
	sort.Slice(results, func(i, j int) bool {
		return len(results[i].Value) > len(results[j].Value)
	})

	if top > 0 && len(results) > top {
		results = results[:top]
	}

	return results, nil
}
