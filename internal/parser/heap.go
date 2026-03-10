package parser

import (
	"encoding/binary"
	"fmt"
	"iter"

	"github.com/modbender/hprof-analyzer/pkg/hprof"
)

// heapReader is a helper for reading from a heap dump record body.
type heapReader struct {
	data   []byte
	pos    int
	idSize uint32
}

func (h *heapReader) remaining() int {
	return len(h.data) - h.pos
}

func (h *heapReader) readU1() (uint8, error) {
	if h.remaining() < 1 {
		return 0, fmt.Errorf("unexpected end of data reading u1 at offset %d", h.pos)
	}
	v := h.data[h.pos]
	h.pos++
	return v, nil
}

func (h *heapReader) readU2() (uint16, error) {
	if h.remaining() < 2 {
		return 0, fmt.Errorf("unexpected end of data reading u2 at offset %d", h.pos)
	}
	v := binary.BigEndian.Uint16(h.data[h.pos:])
	h.pos += 2
	return v, nil
}

func (h *heapReader) readU4() (uint32, error) {
	if h.remaining() < 4 {
		return 0, fmt.Errorf("unexpected end of data reading u4 at offset %d", h.pos)
	}
	v := binary.BigEndian.Uint32(h.data[h.pos:])
	h.pos += 4
	return v, nil
}

func (h *heapReader) readI4() (int32, error) {
	v, err := h.readU4()
	return int32(v), err
}

func (h *heapReader) readID() (uint64, error) {
	if h.idSize == 4 {
		v, err := h.readU4()
		return uint64(v), err
	}
	if h.remaining() < 8 {
		return 0, fmt.Errorf("unexpected end of data reading ID at offset %d", h.pos)
	}
	v := binary.BigEndian.Uint64(h.data[h.pos:])
	h.pos += 8
	return v, nil
}

func (h *heapReader) readBytes(n int) ([]byte, error) {
	if h.remaining() < n {
		return nil, fmt.Errorf("unexpected end of data reading %d bytes at offset %d", n, h.pos)
	}
	b := make([]byte, n)
	copy(b, h.data[h.pos:h.pos+n])
	h.pos += n
	return b, nil
}

func (h *heapReader) skip(n int) error {
	if h.remaining() < n {
		return fmt.Errorf("unexpected end of data skipping %d bytes at offset %d", n, h.pos)
	}
	h.pos += n
	return nil
}

func (h *heapReader) readValue(jtype hprof.JavaType) ([]byte, error) {
	size := int(jtype.Size(h.idSize))
	return h.readBytes(size)
}

// ParseHeapDump parses the body of a HEAP_DUMP or HEAP_DUMP_SEGMENT record
// and yields sub-records (ClassDump, InstanceDump, ObjectArrayDump,
// PrimitiveArrayDump, GCRoot).
func ParseHeapDump(body []byte, idSize uint32) iter.Seq2[any, error] {
	return func(yield func(any, error) bool) {
		hr := &heapReader{data: body, idSize: idSize}

		for hr.remaining() > 0 {
			tag, err := hr.readU1()
			if err != nil {
				yield(nil, err)
				return
			}

			var rec any
			switch tag {
			case hprof.SubtagRootUnknown:
				rec, err = parseRootUnknown(hr)
			case hprof.SubtagRootJNIGlobal:
				rec, err = parseRootJNIGlobal(hr)
			case hprof.SubtagRootJNILocal:
				rec, err = parseRootJNILocal(hr)
			case hprof.SubtagRootJavaFrame:
				rec, err = parseRootJavaFrame(hr)
			case hprof.SubtagRootNativeStack:
				rec, err = parseRootNativeStack(hr)
			case hprof.SubtagRootStickyClass:
				rec, err = parseRootStickyClass(hr)
			case hprof.SubtagRootThreadBlock:
				rec, err = parseRootThreadBlock(hr)
			case hprof.SubtagRootMonitorUsed:
				rec, err = parseRootMonitorUsed(hr)
			case hprof.SubtagRootThreadObj:
				rec, err = parseRootThreadObj(hr)
			case hprof.SubtagClassDump:
				rec, err = parseClassDump(hr)
			case hprof.SubtagInstanceDump:
				rec, err = parseInstanceDump(hr)
			case hprof.SubtagObjArrayDump:
				rec, err = parseObjArrayDump(hr)
			case hprof.SubtagPrimArrayDump:
				rec, err = parsePrimArrayDump(hr)
			default:
				yield(nil, fmt.Errorf("unknown heap dump sub-record tag 0x%02x at offset %d", tag, hr.pos-1))
				return
			}

			if err != nil {
				yield(nil, fmt.Errorf("parsing %s: %w", hprof.SubtagName(tag), err))
				return
			}
			if !yield(rec, nil) {
				return
			}
		}
	}
}

func parseRootUnknown(hr *heapReader) (hprof.GCRoot, error) {
	id, err := hr.readID()
	if err != nil {
		return hprof.GCRoot{}, err
	}
	return hprof.GCRoot{Type: hprof.RootUnknown, ObjectID: id}, nil
}

func parseRootJNIGlobal(hr *heapReader) (hprof.GCRoot, error) {
	id, err := hr.readID()
	if err != nil {
		return hprof.GCRoot{}, err
	}
	refID, err := hr.readID()
	if err != nil {
		return hprof.GCRoot{}, err
	}
	return hprof.GCRoot{Type: hprof.RootJNIGlobal, ObjectID: id, JNIGlobalRefID: refID}, nil
}

func parseRootJNILocal(hr *heapReader) (hprof.GCRoot, error) {
	id, err := hr.readID()
	if err != nil {
		return hprof.GCRoot{}, err
	}
	ts, err := hr.readU4()
	if err != nil {
		return hprof.GCRoot{}, err
	}
	fn, err := hr.readU4()
	if err != nil {
		return hprof.GCRoot{}, err
	}
	return hprof.GCRoot{Type: hprof.RootJNILocal, ObjectID: id, ThreadSerial: ts, FrameNumber: fn}, nil
}

func parseRootJavaFrame(hr *heapReader) (hprof.GCRoot, error) {
	id, err := hr.readID()
	if err != nil {
		return hprof.GCRoot{}, err
	}
	ts, err := hr.readU4()
	if err != nil {
		return hprof.GCRoot{}, err
	}
	fn, err := hr.readU4()
	if err != nil {
		return hprof.GCRoot{}, err
	}
	return hprof.GCRoot{Type: hprof.RootJavaFrame, ObjectID: id, ThreadSerial: ts, FrameNumber: fn}, nil
}

func parseRootNativeStack(hr *heapReader) (hprof.GCRoot, error) {
	id, err := hr.readID()
	if err != nil {
		return hprof.GCRoot{}, err
	}
	ts, err := hr.readU4()
	if err != nil {
		return hprof.GCRoot{}, err
	}
	return hprof.GCRoot{Type: hprof.RootNativeStack, ObjectID: id, ThreadSerial: ts}, nil
}

func parseRootStickyClass(hr *heapReader) (hprof.GCRoot, error) {
	id, err := hr.readID()
	if err != nil {
		return hprof.GCRoot{}, err
	}
	return hprof.GCRoot{Type: hprof.RootStickyClass, ObjectID: id}, nil
}

func parseRootThreadBlock(hr *heapReader) (hprof.GCRoot, error) {
	id, err := hr.readID()
	if err != nil {
		return hprof.GCRoot{}, err
	}
	ts, err := hr.readU4()
	if err != nil {
		return hprof.GCRoot{}, err
	}
	return hprof.GCRoot{Type: hprof.RootThreadBlock, ObjectID: id, ThreadSerial: ts}, nil
}

func parseRootMonitorUsed(hr *heapReader) (hprof.GCRoot, error) {
	id, err := hr.readID()
	if err != nil {
		return hprof.GCRoot{}, err
	}
	return hprof.GCRoot{Type: hprof.RootMonitorUsed, ObjectID: id}, nil
}

func parseRootThreadObj(hr *heapReader) (hprof.GCRoot, error) {
	id, err := hr.readID()
	if err != nil {
		return hprof.GCRoot{}, err
	}
	ts, err := hr.readU4()
	if err != nil {
		return hprof.GCRoot{}, err
	}
	// stack trace serial
	_, err = hr.readU4()
	if err != nil {
		return hprof.GCRoot{}, err
	}
	return hprof.GCRoot{Type: hprof.RootThreadObj, ObjectID: id, ThreadSerial: ts}, nil
}

func parseClassDump(hr *heapReader) (hprof.ClassDump, error) {
	cd := hprof.ClassDump{}
	var err error

	cd.ClassObjID, err = hr.readID()
	if err != nil {
		return cd, err
	}
	cd.StackTraceSerial, err = hr.readU4()
	if err != nil {
		return cd, err
	}
	cd.SuperClassObjID, err = hr.readID()
	if err != nil {
		return cd, err
	}
	cd.ClassLoaderObjID, err = hr.readID()
	if err != nil {
		return cd, err
	}
	cd.Signers, err = hr.readID()
	if err != nil {
		return cd, err
	}
	cd.ProtectionDomain, err = hr.readID()
	if err != nil {
		return cd, err
	}
	// reserved IDs
	_, err = hr.readID()
	if err != nil {
		return cd, err
	}
	_, err = hr.readID()
	if err != nil {
		return cd, err
	}
	cd.InstanceSize, err = hr.readU4()
	if err != nil {
		return cd, err
	}

	// Constant pool
	cd.ConstantPoolCount, err = hr.readU2()
	if err != nil {
		return cd, err
	}
	cd.ConstantPool = make([]hprof.ConstantPoolEntry, cd.ConstantPoolCount)
	for i := range cd.ConstantPool {
		cd.ConstantPool[i].Index, err = hr.readU2()
		if err != nil {
			return cd, err
		}
		ty, err := hr.readU1()
		if err != nil {
			return cd, err
		}
		cd.ConstantPool[i].Type = hprof.JavaType(ty)
		cd.ConstantPool[i].Value, err = hr.readValue(cd.ConstantPool[i].Type)
		if err != nil {
			return cd, err
		}
	}

	// Static fields
	cd.StaticFieldCount, err = hr.readU2()
	if err != nil {
		return cd, err
	}
	cd.StaticFields = make([]hprof.StaticField, cd.StaticFieldCount)
	for i := range cd.StaticFields {
		cd.StaticFields[i].NameID, err = hr.readID()
		if err != nil {
			return cd, err
		}
		ty, err := hr.readU1()
		if err != nil {
			return cd, err
		}
		cd.StaticFields[i].Type = hprof.JavaType(ty)
		cd.StaticFields[i].Value, err = hr.readValue(cd.StaticFields[i].Type)
		if err != nil {
			return cd, err
		}
	}

	// Instance fields
	cd.InstanceFieldCount, err = hr.readU2()
	if err != nil {
		return cd, err
	}
	cd.InstanceFields = make([]hprof.InstanceField, cd.InstanceFieldCount)
	for i := range cd.InstanceFields {
		cd.InstanceFields[i].NameID, err = hr.readID()
		if err != nil {
			return cd, err
		}
		ty, err := hr.readU1()
		if err != nil {
			return cd, err
		}
		cd.InstanceFields[i].Type = hprof.JavaType(ty)
	}

	return cd, nil
}

func parseInstanceDump(hr *heapReader) (hprof.InstanceDump, error) {
	id := hprof.InstanceDump{}
	var err error

	id.ObjectID, err = hr.readID()
	if err != nil {
		return id, err
	}
	id.StackTraceSerial, err = hr.readU4()
	if err != nil {
		return id, err
	}
	id.ClassObjID, err = hr.readID()
	if err != nil {
		return id, err
	}
	id.DataSize, err = hr.readU4()
	if err != nil {
		return id, err
	}
	id.Data, err = hr.readBytes(int(id.DataSize))
	if err != nil {
		return id, err
	}
	return id, nil
}

func parseObjArrayDump(hr *heapReader) (hprof.ObjectArrayDump, error) {
	oa := hprof.ObjectArrayDump{}
	var err error

	oa.ObjectID, err = hr.readID()
	if err != nil {
		return oa, err
	}
	oa.StackTraceSerial, err = hr.readU4()
	if err != nil {
		return oa, err
	}
	oa.Length, err = hr.readU4()
	if err != nil {
		return oa, err
	}
	oa.ElementClassID, err = hr.readID()
	if err != nil {
		return oa, err
	}
	oa.Elements = make([]uint64, oa.Length)
	for i := range oa.Elements {
		oa.Elements[i], err = hr.readID()
		if err != nil {
			return oa, err
		}
	}
	return oa, nil
}

func parsePrimArrayDump(hr *heapReader) (hprof.PrimitiveArrayDump, error) {
	pa := hprof.PrimitiveArrayDump{}
	var err error

	pa.ObjectID, err = hr.readID()
	if err != nil {
		return pa, err
	}
	pa.StackTraceSerial, err = hr.readU4()
	if err != nil {
		return pa, err
	}
	pa.Length, err = hr.readU4()
	if err != nil {
		return pa, err
	}
	ty, err := hr.readU1()
	if err != nil {
		return pa, err
	}
	pa.ElementType = hprof.JavaType(ty)
	dataLen := int(pa.Length) * int(pa.ElementType.Size(hr.idSize))
	pa.Data, err = hr.readBytes(dataLen)
	if err != nil {
		return pa, err
	}
	return pa, nil
}
