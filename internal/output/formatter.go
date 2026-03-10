package output

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

// Formatter defines the interface for output formatting.
type Formatter interface {
	// WriteHeader writes column headers.
	WriteHeader(columns []string) error
	// WriteRow writes a single row of data.
	WriteRow(values []string) error
	// Flush flushes any buffered output.
	Flush() error
}

// NewFormatter creates a formatter for the given format name.
func NewFormatter(w io.Writer, format string) (Formatter, error) {
	switch strings.ToLower(format) {
	case "table", "":
		return NewTableFormatter(w), nil
	case "json":
		return NewJSONFormatter(w), nil
	case "csv":
		return NewCSVFormatter(w), nil
	default:
		return nil, fmt.Errorf("unknown format: %q (supported: table, json, csv)", format)
	}
}

// TableFormatter outputs data as an aligned text table.
type TableFormatter struct {
	w       io.Writer
	columns []string
	rows    [][]string
}

func NewTableFormatter(w io.Writer) *TableFormatter {
	return &TableFormatter{w: w}
}

func (f *TableFormatter) WriteHeader(columns []string) error {
	f.columns = columns
	return nil
}

func (f *TableFormatter) WriteRow(values []string) error {
	f.rows = append(f.rows, values)
	return nil
}

func (f *TableFormatter) Flush() error {
	if len(f.columns) == 0 && len(f.rows) == 0 {
		return nil
	}

	// Calculate column widths
	widths := make([]int, len(f.columns))
	for i, col := range f.columns {
		widths[i] = len(col)
	}
	for _, row := range f.rows {
		for i, val := range row {
			if i < len(widths) && len(val) > widths[i] {
				widths[i] = len(val)
			}
		}
	}

	// Print header
	if len(f.columns) > 0 {
		for i, col := range f.columns {
			if i > 0 {
				fmt.Fprint(f.w, "  ")
			}
			fmt.Fprintf(f.w, "%-*s", widths[i], col)
		}
		fmt.Fprintln(f.w)

		// Separator line
		for i, w := range widths {
			if i > 0 {
				fmt.Fprint(f.w, "  ")
			}
			fmt.Fprint(f.w, strings.Repeat("-", w))
		}
		fmt.Fprintln(f.w)
	}

	// Print rows
	for _, row := range f.rows {
		for i, val := range row {
			if i > 0 {
				fmt.Fprint(f.w, "  ")
			}
			if i < len(widths) {
				fmt.Fprintf(f.w, "%-*s", widths[i], val)
			} else {
				fmt.Fprint(f.w, val)
			}
		}
		fmt.Fprintln(f.w)
	}

	return nil
}

// JSONFormatter outputs data as a JSON array of objects.
type JSONFormatter struct {
	w       io.Writer
	columns []string
	rows    []map[string]string
}

func NewJSONFormatter(w io.Writer) *JSONFormatter {
	return &JSONFormatter{w: w}
}

func (f *JSONFormatter) WriteHeader(columns []string) error {
	f.columns = columns
	return nil
}

func (f *JSONFormatter) WriteRow(values []string) error {
	row := make(map[string]string, len(f.columns))
	for i, col := range f.columns {
		if i < len(values) {
			row[col] = values[i]
		}
	}
	f.rows = append(f.rows, row)
	return nil
}

func (f *JSONFormatter) Flush() error {
	enc := json.NewEncoder(f.w)
	enc.SetIndent("", "  ")
	if f.rows == nil {
		f.rows = []map[string]string{}
	}
	return enc.Encode(f.rows)
}

// CSVFormatter outputs data in CSV format.
type CSVFormatter struct {
	w *csv.Writer
}

func NewCSVFormatter(w io.Writer) *CSVFormatter {
	return &CSVFormatter{w: csv.NewWriter(w)}
}

func (f *CSVFormatter) WriteHeader(columns []string) error {
	return f.w.Write(columns)
}

func (f *CSVFormatter) WriteRow(values []string) error {
	return f.w.Write(values)
}

func (f *CSVFormatter) Flush() error {
	f.w.Flush()
	return f.w.Error()
}
