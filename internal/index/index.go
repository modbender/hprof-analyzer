package index

import (
	"context"
	"fmt"
	"os"
	"strings"
)

// Index is the in-memory representation of the heap dump index.
// For large dumps, this would be backed by memory-mapped files;
// for now we use in-memory maps built during indexing.
type Index struct {
	IDSize     uint32
	Strings    map[uint64]string
	ClassNames map[uint64]uint64   // class obj ID -> name string ID
	Classes    map[uint64]*ClassEntry
	Objects    map[uint64]*ObjectEntry
	OutRefs    map[uint64][]uint64
	InRefs     map[uint64][]uint64
	Roots      []RootEntry
}

// ClassName returns the dotted Java class name for a class object ID.
func (idx *Index) ClassName(classObjID uint64) string {
	nameID, ok := idx.ClassNames[classObjID]
	if !ok {
		return fmt.Sprintf("<class@0x%x>", classObjID)
	}
	name, ok := idx.Strings[nameID]
	if !ok {
		return fmt.Sprintf("<class@0x%x>", classObjID)
	}
	return javaClassName(name)
}

// ObjectClassName returns the class name for an object's class.
func (idx *Index) ObjectClassName(objID uint64) string {
	obj, ok := idx.Objects[objID]
	if !ok {
		return "<unknown>"
	}
	switch obj.Kind {
	case KindClass:
		return idx.ClassName(obj.ID)
	case KindInstance:
		return idx.ClassName(obj.ClassID)
	case KindObjArray:
		return idx.ClassName(obj.ClassID) + "[]"
	case KindPrimArray:
		// Decode pseudo class ID
		elemType := uint8(obj.ClassID & 0xFF)
		return hprofTypeName(elemType) + "[]"
	}
	return "<unknown>"
}

func hprofTypeName(t uint8) string {
	switch t {
	case 4:
		return "boolean"
	case 5:
		return "char"
	case 6:
		return "float"
	case 7:
		return "double"
	case 8:
		return "byte"
	case 9:
		return "short"
	case 10:
		return "int"
	case 11:
		return "long"
	default:
		return "unknown"
	}
}

func javaClassName(name string) string {
	return strings.ReplaceAll(name, "/", ".")
}

// GCRootObjectIDs returns the set of unique GC root object IDs.
func (idx *Index) GCRootObjectIDs() map[uint64]bool {
	roots := make(map[uint64]bool, len(idx.Roots))
	for _, r := range idx.Roots {
		if r.ObjectID != 0 {
			roots[r.ObjectID] = true
		}
	}
	return roots
}

// EnsureIndexed checks for an existing index file and returns an Index.
// If no index exists or it's stale, it builds one.
func EnsureIndexed(hprofPath string) (*Index, error) {
	indexPath := hprofPath + IndexExt

	hprofInfo, err := os.Stat(hprofPath)
	if err != nil {
		return nil, fmt.Errorf("stat hprof: %w", err)
	}

	indexInfo, err := os.Stat(indexPath)
	if err == nil && indexInfo.ModTime().After(hprofInfo.ModTime()) {
		// Index exists and is newer — but for now we always rebuild since
		// we use in-memory index. In a future version, we'd mmap the index files.
		// For now, fall through to rebuild.
	}

	builder := NewBuilder(hprofPath)
	idx, err := builder.Build(context.Background())
	if err != nil {
		return nil, fmt.Errorf("building index: %w", err)
	}

	// Save marker file so we know it was indexed
	if err := builder.Save(idx); err != nil {
		// Non-fatal: we have the in-memory index
		fmt.Fprintf(os.Stderr, "warning: could not save index: %v\n", err)
	}

	return idx, nil
}
