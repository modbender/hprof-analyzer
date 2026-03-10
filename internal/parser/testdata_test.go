package parser

import (
	"encoding/binary"

	"github.com/modbender/hprof-analyzer/pkg/hprof"
)

// buildTestHprof creates a minimal valid HPROF binary for testing.
// It uses 4-byte IDs and includes:
// - Header (JAVA PROFILE 1.0.2, idSize=4)
// - 2 UTF8 strings (IDs 1, 2)
// - 1 LOAD_CLASS
// - 1 HEAP_DUMP with: 1 CLASS_DUMP, 1 INSTANCE_DUMP, 1 ROOT_STICKY_CLASS
func buildTestHprof() []byte {
	var buf []byte
	idSize := uint32(4)

	// Header: format string + null + idSize(4) + timestamp(8)
	buf = append(buf, []byte("JAVA PROFILE 1.0.2")...)
	buf = append(buf, 0) // null terminator
	buf = binary.BigEndian.AppendUint32(buf, idSize)
	buf = binary.BigEndian.AppendUint64(buf, 1700000000000) // timestamp

	// UTF8 record: tag=0x01, ts=0, body = ID + bytes
	// String ID=1: "java/lang/Object"
	buf = appendRecord(buf, hprof.TagUTF8, appendID4(nil, 1), []byte("java/lang/Object"))
	// String ID=2: "value"
	buf = appendRecord(buf, hprof.TagUTF8, appendID4(nil, 2), []byte("value"))

	// LOAD_CLASS record: serial(4) + classObjID(id) + stackTrace(4) + classNameID(id)
	lcBody := binary.BigEndian.AppendUint32(nil, 1)  // serial
	lcBody = appendID4(lcBody, 100)                    // class obj ID
	lcBody = binary.BigEndian.AppendUint32(lcBody, 0) // stack trace
	lcBody = appendID4(lcBody, 1)                      // class name ID -> "java/lang/Object"
	buf = appendRecord(buf, hprof.TagLoadClass, lcBody, nil)

	// HEAP_DUMP record with sub-records
	var heapBody []byte

	// ROOT_STICKY_CLASS sub-record: tag(1) + objID(id)
	heapBody = append(heapBody, hprof.SubtagRootStickyClass)
	heapBody = appendID4(heapBody, 100)

	// CLASS_DUMP sub-record
	heapBody = append(heapBody, hprof.SubtagClassDump)
	heapBody = appendID4(heapBody, 100)                         // class obj ID
	heapBody = binary.BigEndian.AppendUint32(heapBody, 0)       // stack trace serial
	heapBody = appendID4(heapBody, 0)                           // super class
	heapBody = appendID4(heapBody, 0)                           // class loader
	heapBody = appendID4(heapBody, 0)                           // signers
	heapBody = appendID4(heapBody, 0)                           // protection domain
	heapBody = appendID4(heapBody, 0)                           // reserved1
	heapBody = appendID4(heapBody, 0)                           // reserved2
	heapBody = binary.BigEndian.AppendUint32(heapBody, 16)      // instance size
	heapBody = binary.BigEndian.AppendUint16(heapBody, 0)       // constant pool count
	heapBody = binary.BigEndian.AppendUint16(heapBody, 0)       // static field count
	heapBody = binary.BigEndian.AppendUint16(heapBody, 1)       // instance field count
	heapBody = appendID4(heapBody, 2)                           // field name ID -> "value"
	heapBody = append(heapBody, byte(hprof.TypeInt))            // field type

	// INSTANCE_DUMP sub-record
	heapBody = append(heapBody, hprof.SubtagInstanceDump)
	heapBody = appendID4(heapBody, 200)                         // object ID
	heapBody = binary.BigEndian.AppendUint32(heapBody, 0)       // stack trace serial
	heapBody = appendID4(heapBody, 100)                         // class obj ID
	heapBody = binary.BigEndian.AppendUint32(heapBody, 4)       // data size
	heapBody = binary.BigEndian.AppendUint32(heapBody, 42)      // int field value

	buf = appendRecord(buf, hprof.TagHeapDump, heapBody, nil)

	return buf
}

func appendRecord(buf []byte, tag uint8, body1 []byte, body2 []byte) []byte {
	fullBody := append(body1, body2...)
	buf = append(buf, tag)
	buf = binary.BigEndian.AppendUint32(buf, 0)                  // timestamp
	buf = binary.BigEndian.AppendUint32(buf, uint32(len(fullBody))) // length
	buf = append(buf, fullBody...)
	return buf
}

func appendID4(buf []byte, id uint32) []byte {
	return binary.BigEndian.AppendUint32(buf, id)
}
