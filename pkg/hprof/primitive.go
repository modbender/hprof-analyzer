package hprof

// JavaType represents a Java primitive type or object reference in HPROF.
type JavaType uint8

const (
	TypeObject  JavaType = 2
	TypeBoolean JavaType = 4
	TypeChar    JavaType = 5
	TypeFloat   JavaType = 6
	TypeDouble  JavaType = 7
	TypeByte    JavaType = 8
	TypeShort   JavaType = 9
	TypeInt     JavaType = 10
	TypeLong    JavaType = 11
)

// Size returns the byte size of a value of this type. For TypeObject, the size
// depends on the ID size from the HPROF header, so idSize must be passed.
func (t JavaType) Size(idSize uint32) uint32 {
	switch t {
	case TypeObject:
		return idSize
	case TypeBoolean, TypeByte:
		return 1
	case TypeChar, TypeShort:
		return 2
	case TypeFloat, TypeInt:
		return 4
	case TypeDouble, TypeLong:
		return 8
	default:
		return 0
	}
}

// Name returns a human-readable name for the Java type.
func (t JavaType) Name() string {
	switch t {
	case TypeObject:
		return "object"
	case TypeBoolean:
		return "boolean"
	case TypeChar:
		return "char"
	case TypeFloat:
		return "float"
	case TypeDouble:
		return "double"
	case TypeByte:
		return "byte"
	case TypeShort:
		return "short"
	case TypeInt:
		return "int"
	case TypeLong:
		return "long"
	default:
		return "unknown"
	}
}
