package analysis

import (
	"github.com/modbender/hprof-analyzer/internal/index"
)

// GCRootPath represents a path from a GC root to a target object.
type GCRootPath struct {
	RootID   uint64
	RootType uint8
	Path     []PathNode
}

// PathNode is one step in a GC root path.
type PathNode struct {
	ObjectID  uint64
	ClassName string
}

// FindGCRootPaths finds shortest paths from GC roots to objects matching the filter.
// If targetClass is non-empty, finds paths to instances of that class.
// If targetID is non-zero, finds paths to that specific object.
func FindGCRootPaths(idx *index.Index, targetClass string, targetID uint64, maxPaths int) []GCRootPath {
	if maxPaths <= 0 {
		maxPaths = 10
	}

	// Find target objects
	targets := make(map[uint64]bool)
	if targetID != 0 {
		targets[targetID] = true
	} else if targetClass != "" {
		for objID := range idx.Objects {
			if idx.ObjectClassName(objID) == targetClass {
				targets[objID] = true
			}
		}
	}

	if len(targets) == 0 {
		return nil
	}

	// BFS from each target backwards through inbound refs to find GC roots
	var results []GCRootPath
	gcRoots := idx.GCRootObjectIDs()

	for targetObjID := range targets {
		if len(results) >= maxPaths {
			break
		}

		paths := bfsToRoots(idx, targetObjID, gcRoots, maxPaths-len(results))
		results = append(results, paths...)
	}

	return results
}

// bfsToRoots does a BFS backwards from a target object to find GC roots.
func bfsToRoots(idx *index.Index, targetID uint64, gcRoots map[uint64]bool, maxPaths int) []GCRootPath {
	type bfsNode struct {
		id     uint64
		parent uint64
		depth  int
	}

	visited := make(map[uint64]bool)
	parent := make(map[uint64]uint64)
	queue := []bfsNode{{id: targetID, depth: 0}}
	visited[targetID] = true

	var results []GCRootPath

	for len(queue) > 0 && len(results) < maxPaths {
		node := queue[0]
		queue = queue[1:]

		// Check if this node is a GC root
		if gcRoots[node.id] {
			// Build path from root to target
			path := buildPath(idx, parent, node.id, targetID)

			// Find root type
			var rootType uint8
			for _, r := range idx.Roots {
				if r.ObjectID == node.id {
					rootType = r.Type
					break
				}
			}

			results = append(results, GCRootPath{
				RootID:   node.id,
				RootType: rootType,
				Path:     path,
			})
			continue
		}

		// Limit search depth to prevent extremely long paths
		if node.depth >= 50 {
			continue
		}

		// Walk inbound refs (objects that reference this object)
		for _, predID := range idx.InRefs[node.id] {
			if predID != 0 && !visited[predID] {
				visited[predID] = true
				parent[predID] = node.id
				queue = append(queue, bfsNode{id: predID, parent: node.id, depth: node.depth + 1})
			}
		}
	}

	return results
}

func buildPath(idx *index.Index, parentMap map[uint64]uint64, rootID, targetID uint64) []PathNode {
	// Reconstruct path from root to target
	var reversePath []uint64
	current := rootID
	for current != targetID {
		reversePath = append(reversePath, current)
		next, ok := parentMap[current]
		if !ok {
			break
		}
		current = next
	}
	reversePath = append(reversePath, targetID)

	// Convert to PathNodes
	path := make([]PathNode, len(reversePath))
	for i, objID := range reversePath {
		path[i] = PathNode{
			ObjectID:  objID,
			ClassName: idx.ObjectClassName(objID),
		}
	}

	return path
}
