package logging

import (
	"errors"
	"fmt"
	"os"
	"time"

	"workmuch-go/internal/backend"
	"workmuch-go/internal/platform"
)

type DailyCSVWriter struct {
	logDir      string
	location    *time.Location
	currentPath string
	file        *os.File
	writer      *CSVWriter
}

func NewDailyCSVWriter(logDir string, now time.Time) (*DailyCSVWriter, error) {
	writer := &DailyCSVWriter{
		logDir:   logDir,
		location: now.Location(),
	}
	if err := writer.openFor(now); err != nil {
		return nil, err
	}
	return writer, nil
}

func (w *DailyCSVWriter) WriteSample(sample backend.UsageSample, timestampSeconds float64) error {
	sampleTime := time.Unix(int64(timestampSeconds), 0).In(w.location)
	if err := w.openFor(sampleTime); err != nil {
		return err
	}
	return w.writer.WriteSample(sample, timestampSeconds)
}

func (w *DailyCSVWriter) Flush() error {
	if w.writer == nil {
		return nil
	}
	return w.writer.Flush()
}

func (w *DailyCSVWriter) Close() error {
	if w.file == nil {
		return nil
	}

	flushErr := w.Flush()
	closeErr := w.file.Close()
	w.file = nil
	w.writer = nil
	return errors.Join(flushErr, closeErr)
}

func (w *DailyCSVWriter) CurrentPath() string {
	return w.currentPath
}

func (w *DailyCSVWriter) openFor(now time.Time) error {
	nextPath := platform.WorkLogPath(now.In(w.location), w.logDir)
	if nextPath == w.currentPath {
		return nil
	}

	if w.writer != nil {
		if err := w.writer.Flush(); err != nil {
			return fmt.Errorf("flush work log %s: %w", w.currentPath, err)
		}
	}

	nextFile, err := platform.OpenPrivateAppendFile(nextPath)
	if err != nil {
		return fmt.Errorf("open work log %s: %w", nextPath, err)
	}
	if w.file != nil {
		if err := w.file.Close(); err != nil {
			_ = nextFile.Close()
			return fmt.Errorf("close work log %s: %w", w.currentPath, err)
		}
	}

	w.currentPath = nextPath
	w.file = nextFile
	w.writer = NewCSVWriter(nextFile)
	return nil
}
