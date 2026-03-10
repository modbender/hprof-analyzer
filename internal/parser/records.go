package parser

import (
	"encoding/binary"
	"fmt"

	"github.com/modbender/hprof-analyzer/pkg/hprof"
)

// ParseUTF8 parses a UTF8 string record body and returns (id, string).
func ParseUTF8(body []byte, idSize uint32) (uint64, string, error) {
	if len(body) < int(idSize) {
		return 0, "", fmt.Errorf("UTF8 body too short: %d bytes", len(body))
	}
	var id uint64
	if idSize == 4 {
		id = uint64(binary.BigEndian.Uint32(body[:4]))
	} else {
		id = binary.BigEndian.Uint64(body[:8])
	}
	return id, string(body[idSize:]), nil
}

// ParseLoadClass parses a LOAD_CLASS record body.
func ParseLoadClass(body []byte, idSize uint32) (hprof.LoadClass, error) {
	hr := &heapReader{data: body, idSize: idSize}
	lc := hprof.LoadClass{}
	var err error

	lc.ClassSerialNum, err = hr.readU4()
	if err != nil {
		return lc, err
	}
	lc.ClassObjID, err = hr.readID()
	if err != nil {
		return lc, err
	}
	lc.StackTraceSerial, err = hr.readU4()
	if err != nil {
		return lc, err
	}
	lc.ClassNameID, err = hr.readID()
	if err != nil {
		return lc, err
	}
	return lc, nil
}

// ParseStackFrame parses a STACK_FRAME record body.
func ParseStackFrame(body []byte, idSize uint32) (hprof.StackFrame, error) {
	hr := &heapReader{data: body, idSize: idSize}
	sf := hprof.StackFrame{}
	var err error

	sf.FrameID, err = hr.readID()
	if err != nil {
		return sf, err
	}
	sf.MethodNameID, err = hr.readID()
	if err != nil {
		return sf, err
	}
	sf.MethodSigID, err = hr.readID()
	if err != nil {
		return sf, err
	}
	sf.SourceFileID, err = hr.readID()
	if err != nil {
		return sf, err
	}
	sf.ClassSerial, err = hr.readU4()
	if err != nil {
		return sf, err
	}
	sf.LineNumber, err = hr.readI4()
	if err != nil {
		return sf, err
	}
	return sf, nil
}

// ParseStackTrace parses a STACK_TRACE record body.
func ParseStackTrace(body []byte, idSize uint32) (hprof.StackTrace, error) {
	hr := &heapReader{data: body, idSize: idSize}
	st := hprof.StackTrace{}
	var err error

	st.Serial, err = hr.readU4()
	if err != nil {
		return st, err
	}
	st.ThreadSerial, err = hr.readU4()
	if err != nil {
		return st, err
	}
	numFrames, err := hr.readU4()
	if err != nil {
		return st, err
	}
	st.FrameIDs = make([]uint64, numFrames)
	for i := range st.FrameIDs {
		st.FrameIDs[i], err = hr.readID()
		if err != nil {
			return st, err
		}
	}
	return st, nil
}
