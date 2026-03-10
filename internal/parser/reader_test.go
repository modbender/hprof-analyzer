package parser

import (
	"bytes"
	"context"
	"testing"

	"github.com/modbender/hprof-analyzer/pkg/hprof"
)

func TestReadHeader(t *testing.T) {
	data := buildTestHprof()
	r := NewReader(bytes.NewReader(data))

	header, err := r.ReadHeader()
	if err != nil {
		t.Fatalf("ReadHeader: %v", err)
	}

	if header.Format != "JAVA PROFILE 1.0.2" {
		t.Errorf("Format = %q, want %q", header.Format, "JAVA PROFILE 1.0.2")
	}
	if header.IDSize != 4 {
		t.Errorf("IDSize = %d, want 4", header.IDSize)
	}
	if header.Timestamp != 1700000000000 {
		t.Errorf("Timestamp = %d, want 1700000000000", header.Timestamp)
	}
}

func TestRecords(t *testing.T) {
	data := buildTestHprof()
	r := NewReader(bytes.NewReader(data))

	_, err := r.ReadHeader()
	if err != nil {
		t.Fatalf("ReadHeader: %v", err)
	}

	tags := make(map[uint8]int)
	ctx := context.Background()
	for rec, err := range r.Records(ctx) {
		if err != nil {
			t.Fatalf("Records: %v", err)
		}
		tags[rec.Tag]++
	}

	if tags[hprof.TagUTF8] != 2 {
		t.Errorf("UTF8 count = %d, want 2", tags[hprof.TagUTF8])
	}
	if tags[hprof.TagLoadClass] != 1 {
		t.Errorf("LOAD_CLASS count = %d, want 1", tags[hprof.TagLoadClass])
	}
	if tags[hprof.TagHeapDump] != 1 {
		t.Errorf("HEAP_DUMP count = %d, want 1", tags[hprof.TagHeapDump])
	}
}

func TestParseHeapDump(t *testing.T) {
	data := buildTestHprof()
	r := NewReader(bytes.NewReader(data))

	header, err := r.ReadHeader()
	if err != nil {
		t.Fatalf("ReadHeader: %v", err)
	}

	ctx := context.Background()
	var heapBody []byte
	for rec, err := range r.Records(ctx) {
		if err != nil {
			t.Fatalf("Records: %v", err)
		}
		if rec.Tag == hprof.TagHeapDump {
			heapBody = rec.Body
		}
	}

	if heapBody == nil {
		t.Fatal("no HEAP_DUMP record found")
	}

	var roots, classes, instances int
	for sub, err := range ParseHeapDump(heapBody, header.IDSize) {
		if err != nil {
			t.Fatalf("ParseHeapDump: %v", err)
		}
		switch obj := sub.(type) {
		case hprof.GCRoot:
			roots++
			if obj.ObjectID != 100 {
				t.Errorf("GCRoot ObjectID = %d, want 100", obj.ObjectID)
			}
		case hprof.ClassDump:
			classes++
			if obj.ClassObjID != 100 {
				t.Errorf("ClassDump ClassObjID = %d, want 100", obj.ClassObjID)
			}
			if obj.InstanceSize != 16 {
				t.Errorf("ClassDump InstanceSize = %d, want 16", obj.InstanceSize)
			}
			if len(obj.InstanceFields) != 1 {
				t.Errorf("ClassDump InstanceFields = %d, want 1", len(obj.InstanceFields))
			}
		case hprof.InstanceDump:
			instances++
			if obj.ObjectID != 200 {
				t.Errorf("InstanceDump ObjectID = %d, want 200", obj.ObjectID)
			}
			if obj.ClassObjID != 100 {
				t.Errorf("InstanceDump ClassObjID = %d, want 100", obj.ClassObjID)
			}
		}
	}

	if roots != 1 {
		t.Errorf("roots = %d, want 1", roots)
	}
	if classes != 1 {
		t.Errorf("classes = %d, want 1", classes)
	}
	if instances != 1 {
		t.Errorf("instances = %d, want 1", instances)
	}
}

func TestParseUTF8(t *testing.T) {
	data := buildTestHprof()
	r := NewReader(bytes.NewReader(data))

	header, err := r.ReadHeader()
	if err != nil {
		t.Fatalf("ReadHeader: %v", err)
	}

	strings := make(map[uint64]string)
	ctx := context.Background()
	for rec, err := range r.Records(ctx) {
		if err != nil {
			t.Fatalf("Records: %v", err)
		}
		if rec.Tag == hprof.TagUTF8 {
			id, s, err := ParseUTF8(rec.Body, header.IDSize)
			if err != nil {
				t.Fatalf("ParseUTF8: %v", err)
			}
			strings[id] = s
		}
	}

	if strings[1] != "java/lang/Object" {
		t.Errorf("string[1] = %q, want %q", strings[1], "java/lang/Object")
	}
	if strings[2] != "value" {
		t.Errorf("string[2] = %q, want %q", strings[2], "value")
	}
}
