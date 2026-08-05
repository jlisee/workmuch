package app

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"workmuch-go/internal/backend"
	"workmuch-go/internal/status"
)

type fakeRuntimeStatusWriter struct {
	path   string
	writes []status.RuntimeStatus
}

func (w *fakeRuntimeStatusWriter) Write(value status.RuntimeStatus) error {
	w.writes = append(w.writes, value)
	return nil
}

func (w *fakeRuntimeStatusWriter) Path() string {
	return w.path
}

func TestRuntimeStatusTrackerRecordsLifecycleSamplesAndWarnings(t *testing.T) {
	t.Parallel()

	times := []time.Time{
		time.Date(2026, time.July, 8, 9, 0, 0, 0, time.UTC),
		time.Date(2026, time.July, 8, 9, 0, 1, 0, time.UTC),
		time.Date(2026, time.July, 8, 9, 0, 2, 0, time.UTC),
		time.Date(2026, time.July, 8, 9, 0, 3, 0, time.UTC),
	}
	timeIndex := 0
	writer := &fakeRuntimeStatusWriter{path: "/tmp/status.json"}
	tracker := newRuntimeStatusTracker(writer, nil, func() time.Time {
		value := times[timeIndex]
		timeIndex++
		return value
	}, "auto", "macos-native", "/tmp/worklog")

	tracker.Start()
	tracker.RecordWarning("warning: backend sample backend=\"macos-native\": no title")
	tracker.RecordSample(backend.UsageSample{
		ProgramName: "Terminal",
		WindowTitle: "Status work",
		IdleSeconds: 1.25,
	}, true)
	tracker.Stop()

	require.Len(t, writer.writes, 4)
	finalStatus := writer.writes[len(writer.writes)-1]
	require.NotNil(t, finalStatus.StartedAt)
	require.NotNil(t, finalStatus.StoppedAt)
	require.NotNil(t, finalStatus.LastSampleAt)
	require.NotNil(t, finalStatus.LastSuccessfulSampleAt)
	require.NotNil(t, finalStatus.LatestWarning)
	assert.Equal(t, times[0], *finalStatus.StartedAt)
	assert.Equal(t, times[3], *finalStatus.StoppedAt)
	assert.Equal(t, int64(1), finalStatus.SampleCount)
	assert.Equal(t, "auto", finalStatus.SelectedBackend)
	assert.Equal(t, "macos-native", finalStatus.ActiveBackend)
	assert.Equal(t, "/tmp/worklog", finalStatus.CurrentWorkLogPath)
	require.NotNil(t, finalStatus.LastSuccessfulSample)
	assert.Equal(t, "Terminal", finalStatus.LastSuccessfulSample.ProgramName)
	assert.Equal(t, "Status work", finalStatus.LastSuccessfulSample.WindowTitle)
	assert.Equal(t, 1.25, finalStatus.LastSuccessfulSample.IdleSeconds)
	assert.Contains(t, finalStatus.LatestWarning.Message, "backend sample")
}

func TestRuntimeStatusTrackerPreservesLastSuccessfulSampleAfterFailure(t *testing.T) {
	t.Parallel()

	times := []time.Time{
		time.Date(2026, time.July, 8, 9, 0, 0, 0, time.UTC),
		time.Date(2026, time.July, 8, 9, 0, 1, 0, time.UTC),
	}
	timeIndex := 0
	writer := &fakeRuntimeStatusWriter{}
	tracker := newRuntimeStatusTracker(writer, nil, func() time.Time {
		value := times[timeIndex]
		timeIndex++
		return value
	}, "auto", backend.BackendLinux, "/tmp/worklog")

	tracker.RecordSample(backend.UsageSample{
		ProgramName: "Firefox",
		WindowTitle: "Documentation",
		IdleSeconds: 2.5,
	}, true)
	tracker.RecordSample(backend.UsageSample{}, false)

	require.Len(t, writer.writes, 2)
	finalStatus := writer.writes[len(writer.writes)-1]
	assert.Equal(t, int64(2), finalStatus.SampleCount)
	require.NotNil(t, finalStatus.LastSampleAt)
	assert.Equal(t, times[1], *finalStatus.LastSampleAt)
	require.NotNil(t, finalStatus.LastSuccessfulSampleAt)
	assert.Equal(t, times[0], *finalStatus.LastSuccessfulSampleAt)
	require.NotNil(t, finalStatus.LastSuccessfulSample)
	assert.Equal(t, "Firefox", finalStatus.LastSuccessfulSample.ProgramName)
	assert.Equal(t, "Documentation", finalStatus.LastSuccessfulSample.WindowTitle)
}
