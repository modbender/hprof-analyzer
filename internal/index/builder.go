package index

import (
	"context"
	"encoding/binary"
	"fmt"
	"os"

	"github.com/modbender/hprof-analyzer/internal/parser"
	"github.com/modbender/hprof-analyzer/pkg/hprof"
)

// Builder creates index files from an HPROF file.
type Builder struct {
	hprofPath string
}

// NewBuilder creates a new index builder.
func NewBuilder(hprofPath string) *Builder {
	return &Builder{hprofPath: hprofPath}
}

// Build creates all index data by parsing the HPROF file.
// Returns the in-memory Index.
func (b *Builder) Build(ctx context.Context) (*Index, error) {
	f, err := os.Open(b.hprofPath)
	if err != nil {
		return nil, fmt.Errorf("opening hprof: %w", err)
	}
	defer f.Close()

	r := parser.NewReader(f)
	header, err := r.ReadHeader()
	if err != nil {
		return nil, fmt.Errorf("reading header: %w", err)
	}

	idx := &Index{
		IDSize:     header.IDSize,
		Strings:    make(map[uint64]string),
		ClassNames: make(map[uint64]uint64), // class obj ID -> name string ID
		Classes:    make(map[uint64]*ClassEntry),
		Objects:    make(map[uint64]*ObjectEntry),
		OutRefs:    make(map[uint64][]uint64),
		InRefs:     make(map[uint64][]uint64),
	}

	for rec, err := range r.Records(ctx) {
		if err != nil {
			return nil, fmt.Errorf("reading records: %w", err)
		}

		switch rec.Tag {
		case hprof.TagUTF8:
			id, s, err := parser.ParseUTF8(rec.Body, header.IDSize)
			if err != nil {
				return nil, err
			}
			idx.Strings[id] = s

		case hprof.TagLoadClass:
			lc, err := parser.ParseLoadClass(rec.Body, header.IDSize)
			if err != nil {
				return nil, err
			}
			idx.ClassNames[lc.ClassObjID] = lc.ClassNameID

		case hprof.TagHeapDump, hprof.TagHeapDumpSeg:
			if err := b.indexHeapDump(rec.Body, header.IDSize, idx); err != nil {
				return nil, err
			}
		}
	}

	// Build inbound refs by inverting outbound refs
	for fromID, toIDs := range idx.OutRefs {
		for _, toID := range toIDs {
			if toID != 0 {
				idx.InRefs[toID] = append(idx.InRefs[toID], fromID)
			}
		}
	}

	return idx, nil
}

func (b *Builder) indexHeapDump(body []byte, idSize uint32, idx *Index) error {
	for sub, err := range parser.ParseHeapDump(body, idSize) {
		if err != nil {
			return err
		}

		switch obj := sub.(type) {
		case hprof.ClassDump:
			ce := &ClassEntry{
				ClassObjID:      obj.ClassObjID,
				NameID:          idx.ClassNames[obj.ClassObjID],
				SuperClassObjID: obj.SuperClassObjID,
				InstanceSize:    obj.InstanceSize,
			}
			// Build field descriptors with offsets
			var offset uint32
			ce.FieldDescriptors = buildFieldDescriptors(obj, idSize, &offset)
			idx.Classes[obj.ClassObjID] = ce

			idx.Objects[obj.ClassObjID] = &ObjectEntry{
				ID:          obj.ClassObjID,
				Kind:        KindClass,
				ClassID:     obj.ClassObjID,
				ShallowSize: 0, // classes don't have a meaningful shallow size in the dump
			}

			// Outbound refs from class: superclass, class loader, static field refs
			var refs []uint64
			if obj.SuperClassObjID != 0 {
				refs = append(refs, obj.SuperClassObjID)
			}
			if obj.ClassLoaderObjID != 0 {
				refs = append(refs, obj.ClassLoaderObjID)
			}
			for _, sf := range obj.StaticFields {
				if sf.Type == hprof.TypeObject && len(sf.Value) >= int(idSize) {
					ref := readID(sf.Value, idSize)
					if ref != 0 {
						refs = append(refs, ref)
					}
				}
			}
			if len(refs) > 0 {
				idx.OutRefs[obj.ClassObjID] = refs
			}

		case hprof.InstanceDump:
			idx.Objects[obj.ObjectID] = &ObjectEntry{
				ID:          obj.ObjectID,
				Kind:        KindInstance,
				ClassID:     obj.ClassObjID,
				ShallowSize: uint64(obj.DataSize),
			}

			// Extract outbound refs from instance data
			refs := extractInstanceRefs(obj, idSize, idx)
			if len(refs) > 0 {
				idx.OutRefs[obj.ObjectID] = refs
			}

		case hprof.ObjectArrayDump:
			idx.Objects[obj.ObjectID] = &ObjectEntry{
				ID:          obj.ObjectID,
				Kind:        KindObjArray,
				ClassID:     obj.ElementClassID,
				ShallowSize: uint64(obj.Length) * uint64(idSize),
			}

			// All non-null elements are outbound refs
			var refs []uint64
			for _, elemID := range obj.Elements {
				if elemID != 0 {
					refs = append(refs, elemID)
				}
			}
			if len(refs) > 0 {
				idx.OutRefs[obj.ObjectID] = refs
			}

		case hprof.PrimitiveArrayDump:
			idx.Objects[obj.ObjectID] = &ObjectEntry{
				ID:          obj.ObjectID,
				Kind:        KindPrimArray,
				ClassID:     uint64(obj.ElementType) | (1 << 63), // pseudo class ID
				ShallowSize: uint64(len(obj.Data)),
			}
			// Primitive arrays have no outbound refs

		case hprof.GCRoot:
			idx.Roots = append(idx.Roots, RootEntry{
				ObjectID:     obj.ObjectID,
				Type:         uint8(obj.Type),
				ThreadSerial: obj.ThreadSerial,
			})
		}
	}
	return nil
}

// extractInstanceRefs extracts object references from instance data by walking
// the class hierarchy's field descriptors.
func extractInstanceRefs(inst hprof.InstanceDump, idSize uint32, idx *Index) []uint64 {
	var refs []uint64
	// Walk class hierarchy to find all object-type fields
	classID := inst.ClassObjID
	offset := uint32(0)

	for classID != 0 {
		ce, ok := idx.Classes[classID]
		if !ok {
			break
		}
		for _, fd := range ce.FieldDescriptors {
			if hprof.JavaType(fd.Type) == hprof.TypeObject {
				if int(offset+idSize) <= len(inst.Data) {
					ref := readID(inst.Data[offset:], idSize)
					if ref != 0 {
						refs = append(refs, ref)
					}
				}
				offset += idSize
			} else {
				offset += hprof.JavaType(fd.Type).Size(idSize)
			}
		}
		classID = ce.SuperClassObjID
	}

	return refs
}

func buildFieldDescriptors(cd hprof.ClassDump, idSize uint32, offset *uint32) []FieldDescriptor {
	fds := make([]FieldDescriptor, len(cd.InstanceFields))
	for i, f := range cd.InstanceFields {
		fds[i] = FieldDescriptor{
			NameID: f.NameID,
			Type:   uint8(f.Type),
			Offset: *offset,
		}
		*offset += f.Type.Size(idSize)
	}
	return fds
}

func readID(data []byte, idSize uint32) uint64 {
	if idSize == 4 {
		return uint64(binary.BigEndian.Uint32(data[:4]))
	}
	return binary.BigEndian.Uint64(data[:8])
}

// Save writes the index to disk alongside the HPROF file.
func (b *Builder) Save(idx *Index) error {
	path := b.hprofPath + IndexExt
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("creating index file: %w", err)
	}
	defer f.Close()

	// Write header
	hdr := IndexHeader{
		Version:     Version,
		IDSize:      idx.IDSize,
		ObjectCount: uint64(len(idx.Objects)),
		ClassCount:  uint64(len(idx.Classes)),
		RootCount:   uint64(len(idx.Roots)),
		StringCount: uint64(len(idx.Strings)),
	}
	copy(hdr.Magic[:], Magic)
	if err := binary.Write(f, binary.LittleEndian, &hdr); err != nil {
		return fmt.Errorf("writing header: %w", err)
	}

	return nil
}
