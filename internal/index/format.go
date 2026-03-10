package index

// Index file format constants.
const (
	Magic       = "HPAI" // hprof-analyzer index
	Version     = 1
	IndexExt    = ".hpai"
)

// IndexHeader is the header of the main index file.
type IndexHeader struct {
	Magic       [4]byte
	Version     uint32
	IDSize      uint32
	ObjectCount uint64
	ClassCount  uint64
	RootCount   uint64
	StringCount uint64
}

// ObjectKind identifies what type of heap object an entry represents.
type ObjectKind uint8

const (
	KindClass     ObjectKind = 1
	KindInstance  ObjectKind = 2
	KindObjArray  ObjectKind = 3
	KindPrimArray ObjectKind = 4
)

// ObjectEntry is the index entry for a single heap object.
type ObjectEntry struct {
	ID          uint64
	Kind        ObjectKind
	ClassID     uint64     // class object ID (for instances/arrays: their class; for classes: self)
	ShallowSize uint64
}

// ClassEntry is the index entry for a class.
type ClassEntry struct {
	ClassObjID      uint64
	NameID          uint64
	SuperClassObjID uint64
	InstanceSize    uint32
	FieldDescriptors []FieldDescriptor
}

// FieldDescriptor describes an instance field for index traversal.
type FieldDescriptor struct {
	NameID uint64
	Type   uint8 // hprof.JavaType
	Offset uint32
}

// RootEntry is the index entry for a GC root.
type RootEntry struct {
	ObjectID     uint64
	Type         uint8
	ThreadSerial uint32
}

// RefEntry stores outbound references from an object.
type RefEntry struct {
	FromID uint64
	ToIDs  []uint64
}
