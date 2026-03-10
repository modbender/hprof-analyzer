package parser

import (
	"bufio"
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"iter"

	"github.com/modbender/hprof-analyzer/pkg/hprof"
)

// Reader provides streaming access to an HPROF binary file.
type Reader struct {
	r      io.Reader
	br     *bufio.Reader
	idSize uint32
	header hprof.Header
}

// NewReader creates a new HPROF streaming reader.
func NewReader(r io.Reader) *Reader {
	return &Reader{
		r:  r,
		br: bufio.NewReaderSize(r, 256*1024), // 256KB buffer
	}
}

// ReadHeader parses the HPROF file header.
// Must be called before Records().
func (r *Reader) ReadHeader() (hprof.Header, error) {
	// Read null-terminated format string
	formatBytes, err := r.readUntilNull()
	if err != nil {
		return hprof.Header{}, fmt.Errorf("reading format string: %w", err)
	}

	format := string(formatBytes)
	if format != "JAVA PROFILE 1.0.1" && format != "JAVA PROFILE 1.0.2" {
		return hprof.Header{}, fmt.Errorf("unsupported format: %q", format)
	}

	// Read ID size (4 bytes, big-endian)
	var idSize uint32
	if err := binary.Read(r.br, binary.BigEndian, &idSize); err != nil {
		return hprof.Header{}, fmt.Errorf("reading ID size: %w", err)
	}
	if idSize != 4 && idSize != 8 {
		return hprof.Header{}, fmt.Errorf("unsupported ID size: %d", idSize)
	}

	// Read timestamp (8 bytes, big-endian — two uint32: high and low)
	var tsHigh, tsLow uint32
	if err := binary.Read(r.br, binary.BigEndian, &tsHigh); err != nil {
		return hprof.Header{}, fmt.Errorf("reading timestamp high: %w", err)
	}
	if err := binary.Read(r.br, binary.BigEndian, &tsLow); err != nil {
		return hprof.Header{}, fmt.Errorf("reading timestamp low: %w", err)
	}

	r.idSize = idSize
	r.header = hprof.Header{
		Format:    format,
		IDSize:    idSize,
		Timestamp: uint64(tsHigh)<<32 | uint64(tsLow),
	}
	return r.header, nil
}

// IDSize returns the identifier size from the header.
func (r *Reader) IDSize() uint32 {
	return r.idSize
}

// Records returns an iterator that yields top-level HPROF records one at a time.
// ReadHeader must be called before this method.
func (r *Reader) Records(ctx context.Context) iter.Seq2[hprof.Record, error] {
	return func(yield func(hprof.Record, error) bool) {
		for {
			if ctx.Err() != nil {
				yield(hprof.Record{}, ctx.Err())
				return
			}

			// Read tag (1 byte)
			tag, err := r.br.ReadByte()
			if err == io.EOF {
				return // normal end
			}
			if err != nil {
				yield(hprof.Record{}, fmt.Errorf("reading record tag: %w", err))
				return
			}

			// Read timestamp offset (4 bytes)
			var ts uint32
			if err := binary.Read(r.br, binary.BigEndian, &ts); err != nil {
				yield(hprof.Record{}, fmt.Errorf("reading record timestamp: %w", err))
				return
			}

			// Read body length (4 bytes)
			var length uint32
			if err := binary.Read(r.br, binary.BigEndian, &length); err != nil {
				yield(hprof.Record{}, fmt.Errorf("reading record length: %w", err))
				return
			}

			// Read body
			body := make([]byte, length)
			if _, err := io.ReadFull(r.br, body); err != nil {
				yield(hprof.Record{}, fmt.Errorf("reading record body (%d bytes): %w", length, err))
				return
			}

			if !yield(hprof.Record{Tag: tag, Timestamp: ts, Body: body}, nil) {
				return
			}
		}
	}
}

func (r *Reader) readUntilNull() ([]byte, error) {
	var buf []byte
	for {
		b, err := r.br.ReadByte()
		if err != nil {
			return nil, err
		}
		if b == 0 {
			return buf, nil
		}
		buf = append(buf, b)
	}
}
