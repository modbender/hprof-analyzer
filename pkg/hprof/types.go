package hprof

// Header represents the HPROF file header.
type Header struct {
	Format    string // e.g. "JAVA PROFILE 1.0.2"
	IDSize    uint32 // identifier size in bytes (4 or 8)
	Timestamp uint64 // timestamp in milliseconds since epoch
}

// Record represents a top-level HPROF record.
type Record struct {
	Tag       uint8
	Timestamp uint32 // relative offset from header timestamp
	Body      []byte // raw record body
}

// ClassDump represents a HEAP_DUMP CLASS_DUMP sub-record.
type ClassDump struct {
	ClassObjID         uint64
	StackTraceSerial   uint32
	SuperClassObjID    uint64
	ClassLoaderObjID   uint64
	Signers            uint64
	ProtectionDomain   uint64
	InstanceSize       uint32
	ConstantPoolCount  uint16
	ConstantPool       []ConstantPoolEntry
	StaticFieldCount   uint16
	StaticFields       []StaticField
	InstanceFieldCount uint16
	InstanceFields     []InstanceField
}

// ConstantPoolEntry represents one entry in a class's constant pool.
type ConstantPoolEntry struct {
	Index uint16
	Type  JavaType
	Value []byte
}

// StaticField represents a static field in a CLASS_DUMP.
type StaticField struct {
	NameID uint64
	Type   JavaType
	Value  []byte
}

// InstanceField represents an instance field descriptor in a CLASS_DUMP.
type InstanceField struct {
	NameID uint64
	Type   JavaType
}

// InstanceDump represents a HEAP_DUMP INSTANCE_DUMP sub-record.
type InstanceDump struct {
	ObjectID         uint64
	StackTraceSerial uint32
	ClassObjID       uint64
	DataSize         uint32
	Data             []byte
}

// ObjectArrayDump represents a HEAP_DUMP OBJ_ARRAY_DUMP sub-record.
type ObjectArrayDump struct {
	ObjectID         uint64
	StackTraceSerial uint32
	Length           uint32
	ElementClassID   uint64
	Elements         []uint64
}

// PrimitiveArrayDump represents a HEAP_DUMP PRIM_ARRAY_DUMP sub-record.
type PrimitiveArrayDump struct {
	ObjectID         uint64
	StackTraceSerial uint32
	Length           uint32
	ElementType      JavaType
	Data             []byte
}

// GCRootType identifies the type of GC root.
type GCRootType uint8

const (
	RootUnknown     GCRootType = GCRootType(SubtagRootUnknown)
	RootJNIGlobal   GCRootType = GCRootType(SubtagRootJNIGlobal)
	RootJNILocal    GCRootType = GCRootType(SubtagRootJNILocal)
	RootJavaFrame   GCRootType = GCRootType(SubtagRootJavaFrame)
	RootNativeStack GCRootType = GCRootType(SubtagRootNativeStack)
	RootStickyClass GCRootType = GCRootType(SubtagRootStickyClass)
	RootThreadBlock GCRootType = GCRootType(SubtagRootThreadBlock)
	RootMonitorUsed GCRootType = GCRootType(SubtagRootMonitorUsed)
	RootThreadObj   GCRootType = GCRootType(SubtagRootThreadObj)
)

// GCRoot represents a garbage collection root reference.
type GCRoot struct {
	Type             GCRootType
	ObjectID         uint64
	ThreadSerial     uint32 // for JNI local, Java frame, native stack, thread block
	FrameNumber      uint32 // for Java frame
	JNIGlobalRefID   uint64 // for JNI global
}

// LoadClass represents a LOAD_CLASS record.
type LoadClass struct {
	ClassSerialNum uint32
	ClassObjID     uint64
	StackTraceSerial uint32
	ClassNameID    uint64
}

// StackFrame represents a STACK_FRAME record.
type StackFrame struct {
	FrameID      uint64
	MethodNameID uint64
	MethodSigID  uint64
	SourceFileID uint64
	ClassSerial  uint32
	LineNumber   int32
}

// StackTrace represents a STACK_TRACE record.
type StackTrace struct {
	Serial     uint32
	ThreadSerial uint32
	FrameIDs   []uint64
}
