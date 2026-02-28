package logging

import (
	"bytes"
	"encoding/csv"
	"strings"
	"testing"

	"workmuch-go/internal/backend"
)

func TestCSVWriterWritesExpectedColumnOrder(t *testing.T) {
	var buf bytes.Buffer
	writer := NewCSVWriter(&buf)

	sample := backend.UsageSample{
		Host:        "host",
		User:        "user",
		WindowTitle: "window",
		ProgramName: "program",
		IdleSeconds: 1.25,
	}

	if err := writer.WriteSample(sample, 1700000000.5); err != nil {
		t.Fatalf("WriteSample failed: %v", err)
	}
	if err := writer.Flush(); err != nil {
		t.Fatalf("Flush failed: %v", err)
	}

	reader := csv.NewReader(strings.NewReader(buf.String()))
	record, err := reader.Read()
	if err != nil {
		t.Fatalf("failed to read csv record: %v", err)
	}

	if len(record) != 6 {
		t.Fatalf("expected 6 columns, got %d", len(record))
	}
	if record[0] != "host" {
		t.Fatalf("unexpected col0: %q", record[0])
	}
	if record[1] != "user" {
		t.Fatalf("unexpected col1: %q", record[1])
	}
	if record[2] != "window" {
		t.Fatalf("unexpected col2: %q", record[2])
	}
	if record[3] != "program" {
		t.Fatalf("unexpected col3: %q", record[3])
	}
	if record[4] != "1.250000" {
		t.Fatalf("unexpected col4: %q", record[4])
	}
	if record[5] != "1700000000.500000" {
		t.Fatalf("unexpected col5: %q", record[5])
	}
}
