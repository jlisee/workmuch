package logging

import (
	"bytes"
	"encoding/csv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

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

	require.NoError(t, writer.WriteSample(sample, 1700000000.5))
	require.NoError(t, writer.Flush())

	reader := csv.NewReader(strings.NewReader(buf.String()))
	record, err := reader.Read()
	require.NoError(t, err)
	assert.Equal(t, []string{
		"host",
		"user",
		"window",
		"program",
		"1.250000",
		"1700000000.500000",
	}, record)
}
