package logging

import (
	"encoding/csv"
	"io"
	"strconv"

	"workmuch-go/internal/backend"
)

type CSVWriter struct {
	writer *csv.Writer
}

func NewCSVWriter(w io.Writer) *CSVWriter {
	return &CSVWriter{writer: csv.NewWriter(w)}
}

func (w *CSVWriter) WriteSample(sample backend.UsageSample, timestampSeconds float64) error {
	record := []string{
		sample.WindowTitle,
		sample.ProgramName,
		strconv.FormatFloat(sample.IdleSeconds, 'f', 6, 64),
		strconv.FormatFloat(timestampSeconds, 'f', 6, 64),
	}
	return w.writer.Write(record)
}

func (w *CSVWriter) Flush() error {
	w.writer.Flush()
	return w.writer.Error()
}
