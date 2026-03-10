package hprof

// Heap dump sub-record tags.
const (
	SubtagRootUnknown    uint8 = 0xFF
	SubtagRootJNIGlobal  uint8 = 0x01
	SubtagRootJNILocal   uint8 = 0x02
	SubtagRootJavaFrame  uint8 = 0x03
	SubtagRootNativeStack uint8 = 0x04
	SubtagRootStickyClass uint8 = 0x05
	SubtagRootThreadBlock uint8 = 0x06
	SubtagRootMonitorUsed uint8 = 0x07
	SubtagRootThreadObj  uint8 = 0x08
	SubtagClassDump      uint8 = 0x20
	SubtagInstanceDump   uint8 = 0x21
	SubtagObjArrayDump   uint8 = 0x22
	SubtagPrimArrayDump  uint8 = 0x23
)

// SubtagName returns a human-readable name for a heap dump sub-record tag.
func SubtagName(tag uint8) string {
	switch tag {
	case SubtagRootUnknown:
		return "ROOT_UNKNOWN"
	case SubtagRootJNIGlobal:
		return "ROOT_JNI_GLOBAL"
	case SubtagRootJNILocal:
		return "ROOT_JNI_LOCAL"
	case SubtagRootJavaFrame:
		return "ROOT_JAVA_FRAME"
	case SubtagRootNativeStack:
		return "ROOT_NATIVE_STACK"
	case SubtagRootStickyClass:
		return "ROOT_STICKY_CLASS"
	case SubtagRootThreadBlock:
		return "ROOT_THREAD_BLOCK"
	case SubtagRootMonitorUsed:
		return "ROOT_MONITOR_USED"
	case SubtagRootThreadObj:
		return "ROOT_THREAD_OBJ"
	case SubtagClassDump:
		return "CLASS_DUMP"
	case SubtagInstanceDump:
		return "INSTANCE_DUMP"
	case SubtagObjArrayDump:
		return "OBJ_ARRAY_DUMP"
	case SubtagPrimArrayDump:
		return "PRIM_ARRAY_DUMP"
	default:
		return "UNKNOWN"
	}
}
