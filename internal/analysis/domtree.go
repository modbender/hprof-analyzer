package analysis

import (
	"sort"

	"github.com/modbender/hprof-analyzer/internal/index"
)

// DominatorTree computes the dominator tree using the Lengauer-Tarjan algorithm
// and calculates retained sizes.
type DominatorTree struct {
	idx          *index.Index
	idom         map[uint64]uint64 // object ID -> immediate dominator ID
	retainedSize map[uint64]uint64
	rootID       uint64 // virtual super-root
}

// DomTreeEntry represents one node in the dominator tree results.
type DomTreeEntry struct {
	ObjectID     uint64
	ClassName    string
	ShallowSize  uint64
	RetainedSize uint64
	DominatorID  uint64
}

// NewDominatorTree computes the dominator tree from the index.
func NewDominatorTree(idx *index.Index) *DominatorTree {
	dt := &DominatorTree{
		idx:          idx,
		idom:         make(map[uint64]uint64),
		retainedSize: make(map[uint64]uint64),
		rootID:       0, // virtual root (ID 0 is never a real object)
	}
	dt.compute()
	dt.computeRetainedSizes()
	return dt
}

// Results returns the top entries by retained size.
func (dt *DominatorTree) Results(topN int, classFilter string, minRetained uint64) []DomTreeEntry {
	var entries []DomTreeEntry

	for objID, obj := range dt.idx.Objects {
		retained := dt.retainedSize[objID]
		if retained < minRetained {
			continue
		}

		className := dt.idx.ObjectClassName(objID)
		if classFilter != "" && className != classFilter {
			continue
		}

		entries = append(entries, DomTreeEntry{
			ObjectID:     objID,
			ClassName:    className,
			ShallowSize:  obj.ShallowSize,
			RetainedSize: retained,
			DominatorID:  dt.idom[objID],
		})
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].RetainedSize > entries[j].RetainedSize
	})

	if topN > 0 && len(entries) > topN {
		entries = entries[:topN]
	}

	return entries
}

// compute implements a simplified iterative dominator computation.
// For production use with millions of objects, Lengauer-Tarjan with path
// compression would be more efficient. This iterative approach works well
// for moderate-sized heaps.
func (dt *DominatorTree) compute() {
	gcRoots := dt.idx.GCRootObjectIDs()

	// All GC roots are immediately dominated by the virtual root
	for rootID := range gcRoots {
		dt.idom[rootID] = dt.rootID
	}

	// Build reverse post-order for all reachable objects via BFS
	visited := make(map[uint64]bool, len(dt.idx.Objects))
	order := make([]uint64, 0, len(dt.idx.Objects))

	// BFS from all GC roots
	queue := make([]uint64, 0, len(gcRoots))
	for rootID := range gcRoots {
		if _, ok := dt.idx.Objects[rootID]; ok {
			queue = append(queue, rootID)
			visited[rootID] = true
		}
	}

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		order = append(order, current)

		for _, refID := range dt.idx.OutRefs[current] {
			if refID != 0 && !visited[refID] {
				if _, ok := dt.idx.Objects[refID]; ok {
					visited[refID] = true
					queue = append(queue, refID)
				}
			}
		}
	}

	// Iterative dominator computation
	// For each non-root node, the dominator is the intersection of
	// dominators of all its predecessors (inbound refs).
	changed := true
	for changed {
		changed = false
		for _, objID := range order {
			if gcRoots[objID] {
				continue // GC roots are dominated by virtual root
			}

			inRefs := dt.idx.InRefs[objID]
			if len(inRefs) == 0 {
				continue
			}

			// Find first predecessor that has a dominator
			newIdom := uint64(0)
			found := false
			for _, predID := range inRefs {
				if _, hasDom := dt.idom[predID]; hasDom {
					newIdom = predID
					found = true
					break
				}
			}
			if !found {
				continue
			}

			// Intersect with remaining predecessors
			for _, predID := range inRefs {
				if predID == newIdom {
					continue
				}
				if _, hasDom := dt.idom[predID]; hasDom {
					newIdom = dt.intersect(newIdom, predID)
				}
			}

			if current, ok := dt.idom[objID]; !ok || current != newIdom {
				dt.idom[objID] = newIdom
				changed = true
			}
		}
	}
}

// intersect finds the common dominator of two nodes by walking up the tree.
func (dt *DominatorTree) intersect(a, b uint64) uint64 {
	// Walk both up to the root; since we don't have postorder numbers,
	// we use path tracing to find common ancestor.
	pathA := make(map[uint64]bool)
	node := a
	for {
		pathA[node] = true
		if node == dt.rootID {
			break
		}
		parent, ok := dt.idom[node]
		if !ok {
			break
		}
		node = parent
	}

	node = b
	for {
		if pathA[node] {
			return node
		}
		if node == dt.rootID {
			return dt.rootID
		}
		parent, ok := dt.idom[node]
		if !ok {
			return dt.rootID
		}
		node = parent
	}
}

// computeRetainedSizes calculates retained size for each object.
// An object's retained size = its shallow size + sum of retained sizes of
// objects it exclusively dominates.
func (dt *DominatorTree) computeRetainedSizes() {
	// Build children map (dominator tree edges)
	children := make(map[uint64][]uint64)
	for objID := range dt.idom {
		parent := dt.idom[objID]
		children[parent] = append(children[parent], objID)
	}

	// Post-order traversal to compute retained sizes bottom-up
	var computeRetained func(objID uint64) uint64
	computeRetained = func(objID uint64) uint64 {
		obj, ok := dt.idx.Objects[objID]
		var shallow uint64
		if ok {
			shallow = obj.ShallowSize
		}

		retained := shallow
		for _, childID := range children[objID] {
			retained += computeRetained(childID)
		}

		dt.retainedSize[objID] = retained
		return retained
	}

	// Start from virtual root's children
	for _, childID := range children[dt.rootID] {
		computeRetained(childID)
	}
}
