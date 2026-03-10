package hprof

// Top-level HPROF record tags.
const (
	TagUTF8         uint8 = 0x01
	TagLoadClass    uint8 = 0x02
	TagUnloadClass  uint8 = 0x03
	TagStackFrame   uint8 = 0x04
	TagStackTrace   uint8 = 0x05
	TagAllocSites   uint8 = 0x06
	TagHeapSummary  uint8 = 0x07
	TagStartThread  uint8 = 0x0A
	TagEndThread    uint8 = 0x0B
	TagHeapDump     uint8 = 0x0C
	TagHeapDumpSeg  uint8 = 0x1C
	TagHeapDumpEnd  uint8 = 0x2C
	TagCPUSamples   uint8 = 0x0D
	TagControlSet   uint8 = 0x0E
)

// TagName returns a human-readable name for a top-level record tag.
func TagName(tag uint8) string {
	switch tag {
	case TagUTF8:
		return "UTF8"
	case TagLoadClass:
		return "LOAD_CLASS"
	case TagUnloadClass:
		return "UNLOAD_CLASS"
	case TagStackFrame:
		return "STACK_FRAME"
	case TagStackTrace:
		return "STACK_TRACE"
	case TagAllocSites:
		return "ALLOC_SITES"
	case TagHeapSummary:
		return "HEAP_SUMMARY"
	case TagStartThread:
		return "START_THREAD"
	case TagEndThread:
		return "END_THREAD"
	case TagHeapDump:
		return "HEAP_DUMP"
	case TagHeapDumpSeg:
		return "HEAP_DUMP_SEGMENT"
	case TagHeapDumpEnd:
		return "HEAP_DUMP_END"
	case TagCPUSamples:
		return "CPU_SAMPLES"
	case TagControlSet:
		return "CONTROL_SETTINGS"
	default:
		return "UNKNOWN"
	}
}
