package logging

import (
	"bytes"
	"encoding/csv"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

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

func TestDailyCSVWriterRotatesAtLocalMidnight(t *testing.T) {
	logDir := t.TempDir()
	location := time.FixedZone("test-local", -5*60*60)
	dayOne := time.Date(2026, time.August, 6, 23, 59, 58, 0, location)
	dayTwo := dayOne.Add(3 * time.Second)

	writer, err := NewDailyCSVWriter(logDir, dayOne)
	require.NoError(t, err)
	t.Cleanup(func() {
		assert.NoError(t, writer.Close())
	})

	require.NoError(t, writer.WriteSample(backend.UsageSample{
		Host:        "host",
		User:        "user",
		WindowTitle: "before midnight",
		ProgramName: "program",
	}, float64(dayOne.Unix())))
	require.NoError(t, writer.Flush())

	require.NoError(t, writer.WriteSample(backend.UsageSample{
		Host:        "host",
		User:        "user",
		WindowTitle: "after midnight",
		ProgramName: "program",
	}, float64(dayTwo.Unix())))
	require.NoError(t, writer.Flush())

	dayOnePath := filepath.Join(logDir, "2026-08-06.worklog")
	dayTwoPath := filepath.Join(logDir, "2026-08-07.worklog")
	assert.Equal(t, dayTwoPath, writer.CurrentPath())
	assert.Equal(t, []string{"before midnight"}, readWindowTitles(t, dayOnePath))
	assert.Equal(t, []string{"after midnight"}, readWindowTitles(t, dayTwoPath))
}

func readWindowTitles(t *testing.T, path string) []string {
	t.Helper()

	contents, err := os.ReadFile(path)
	require.NoError(t, err)
	records, err := csv.NewReader(strings.NewReader(string(contents))).ReadAll()
	require.NoError(t, err)

	titles := make([]string, 0, len(records))
	for _, record := range records {
		require.Len(t, record, 6)
		titles = append(titles, record[2])
	}
	return titles
}
