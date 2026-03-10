package index

import (
	"bytes"
	"context"
	"encoding/binary"
	"testing"

	"github.com/modbender/hprof-analyzer/internal/parser"
	"github.com/modbender/hprof-analyzer/pkg/hprof"
)

func buildTestHprof() []byte {
	var buf []byte
	idSize := uint32(4)

	buf = append(buf, []byte("JAVA PROFILE 1.0.2")...)
	buf = append(buf, 0)
	buf = binary.BigEndian.AppendUint32(buf, idSize)
	buf = binary.BigEndian.AppendUint64(buf, 1700000000000)

	// UTF8: ID=1 -> "java/lang/Object"
	buf = appendRecord(buf, hprof.TagUTF8, appendID4(nil, 1), []byte("java/lang/Object"))
	// UTF8: ID=2 -> "value"
	buf = appendRecord(buf, hprof.TagUTF8, appendID4(nil, 2), []byte("value"))

	// LOAD_CLASS
	lcBody := binary.BigEndian.AppendUint32(nil, 1)
	lcBody = appendID4(lcBody, 100)
	lcBody = binary.BigEndian.AppendUint32(lcBody, 0)
	lcBody = appendID4(lcBody, 1)
	buf = appendRecord(buf, hprof.TagLoadClass, lcBody, nil)

	// HEAP_DUMP
	var heapBody []byte

	// ROOT_STICKY_CLASS
	heapBody = append(heapBody, hprof.SubtagRootStickyClass)
	heapBody = appendID4(heapBody, 100)

	// CLASS_DUMP
	heapBody = append(heapBody, hprof.SubtagClassDump)
	heapBody = appendID4(heapBody, 100)
	heapBody = binary.BigEndian.AppendUint32(heapBody, 0)
	heapBody = appendID4(heapBody, 0)
	heapBody = appendID4(heapBody, 0)
	heapBody = appendID4(heapBody, 0)
	heapBody = appendID4(heapBody, 0)
	heapBody = appendID4(heapBody, 0)
	heapBody = appendID4(heapBody, 0)
	heapBody = binary.BigEndian.AppendUint32(heapBody, 4)
	heapBody = binary.BigEndian.AppendUint16(heapBody, 0)
	heapBody = binary.BigEndian.AppendUint16(heapBody, 0)
	heapBody = binary.BigEndian.AppendUint16(heapBody, 1)
	heapBody = appendID4(heapBody, 2)
	heapBody = append(heapBody, byte(hprof.TypeInt))

	// INSTANCE_DUMP: obj 200
	heapBody = append(heapBody, hprof.SubtagInstanceDump)
	heapBody = appendID4(heapBody, 200)
	heapBody = binary.BigEndian.AppendUint32(heapBody, 0)
	heapBody = appendID4(heapBody, 100)
	heapBody = binary.BigEndian.AppendUint32(heapBody, 4)
	heapBody = binary.BigEndian.AppendUint32(heapBody, 42)

	// INSTANCE_DUMP: obj 300
	heapBody = append(heapBody, hprof.SubtagInstanceDump)
	heapBody = appendID4(heapBody, 300)
	heapBody = binary.BigEndian.AppendUint32(heapBody, 0)
	heapBody = appendID4(heapBody, 100)
	heapBody = binary.BigEndian.AppendUint32(heapBody, 4)
	heapBody = binary.BigEndian.AppendUint32(heapBody, 99)

	buf = appendRecord(buf, hprof.TagHeapDump, heapBody, nil)
	return buf
}

func appendRecord(buf []byte, tag uint8, body1, body2 []byte) []byte {
	fullBody := append(body1, body2...)
	buf = append(buf, tag)
	buf = binary.BigEndian.AppendUint32(buf, 0)
	buf = binary.BigEndian.AppendUint32(buf, uint32(len(fullBody)))
	buf = append(buf, fullBody...)
	return buf
}

func appendID4(buf []byte, id uint32) []byte {
	return binary.BigEndian.AppendUint32(buf, id)
}

func TestBuildIndex(t *testing.T) {
	data := buildTestHprof()

	// We can't use Builder directly since it needs a file path.
	// Instead, test the parser + index building logic manually.
	r := parser.NewReader(bytes.NewReader(data))
	header, err := r.ReadHeader()
	if err != nil {
		t.Fatalf("ReadHeader: %v", err)
	}

	idx := &Index{
		IDSize:     header.IDSize,
		Strings:    make(map[uint64]string),
		ClassNames: make(map[uint64]uint64),
		Classes:    make(map[uint64]*ClassEntry),
		Objects:    make(map[uint64]*ObjectEntry),
		OutRefs:    make(map[uint64][]uint64),
		InRefs:     make(map[uint64][]uint64),
	}

	ctx := context.Background()
	for rec, err := range r.Records(ctx) {
		if err != nil {
			t.Fatalf("Records: %v", err)
		}
		switch rec.Tag {
		case hprof.TagUTF8:
			id, s, err := parser.ParseUTF8(rec.Body, header.IDSize)
			if err != nil {
				t.Fatal(err)
			}
			idx.Strings[id] = s
		case hprof.TagLoadClass:
			lc, err := parser.ParseLoadClass(rec.Body, header.IDSize)
			if err != nil {
				t.Fatal(err)
			}
			idx.ClassNames[lc.ClassObjID] = lc.ClassNameID
		case hprof.TagHeapDump:
			for sub, err := range parser.ParseHeapDump(rec.Body, header.IDSize) {
				if err != nil {
					t.Fatal(err)
				}
				switch obj := sub.(type) {
				case hprof.ClassDump:
					idx.Classes[obj.ClassObjID] = &ClassEntry{
						ClassObjID:      obj.ClassObjID,
						NameID:          idx.ClassNames[obj.ClassObjID],
						SuperClassObjID: obj.SuperClassObjID,
						InstanceSize:    obj.InstanceSize,
					}
					idx.Objects[obj.ClassObjID] = &ObjectEntry{
						ID: obj.ClassObjID, Kind: KindClass, ClassID: obj.ClassObjID,
					}
				case hprof.InstanceDump:
					idx.Objects[obj.ObjectID] = &ObjectEntry{
						ID: obj.ObjectID, Kind: KindInstance, ClassID: obj.ClassObjID,
						ShallowSize: uint64(obj.DataSize),
					}
				case hprof.GCRoot:
					idx.Roots = append(idx.Roots, RootEntry{
						ObjectID: obj.ObjectID, Type: uint8(obj.Type),
					})
				}
			}
		}
	}

	if len(idx.Objects) != 3 { // 1 class + 2 instances
		t.Errorf("objects = %d, want 3", len(idx.Objects))
	}
	if len(idx.Classes) != 1 {
		t.Errorf("classes = %d, want 1", len(idx.Classes))
	}
	if len(idx.Roots) != 1 {
		t.Errorf("roots = %d, want 1", len(idx.Roots))
	}

	name := idx.ClassName(100)
	if name != "java.lang.Object" {
		t.Errorf("class name = %q, want %q", name, "java.lang.Object")
	}
}
