package parser

import (
	"os"
	"testing"
)

// TestWriteFixture writes the test HPROF to testdata/ for CLI testing.
// Run with: go test -run TestWriteFixture -v ./internal/parser/
func TestWriteFixture(t *testing.T) {
	if os.Getenv("WRITE_FIXTURE") == "" {
		t.Skip("set WRITE_FIXTURE=1 to write fixture")
	}
	data := buildTestHprof()
	err := os.WriteFile("../../testdata/small.hprof", data, 0644)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("wrote %d bytes to testdata/small.hprof", len(data))
}
