package app

import (
	"log"
	"time"

	"workmuch-go/internal/backend"
	"workmuch-go/internal/status"
)

type runtimeStatusStore interface {
	Write(status.RuntimeStatus) error
	Path() string
}

type runtimeStatusTracker struct {
	store  runtimeStatusStore
	logger *log.Logger
	now    func() time.Time
	value  status.RuntimeStatus
}

func newRuntimeStatusTracker(store runtimeStatusStore, logger *log.Logger, now func() time.Time, selectedBackend string, activeBackend string, workLogPath string) *runtimeStatusTracker {
	if now == nil {
		now = time.Now
	}
	return &runtimeStatusTracker{
		store:  store,
		logger: logger,
		now:    now,
		value: status.RuntimeStatus{
			SelectedBackend:    selectedBackend,
			ActiveBackend:      activeBackend,
			CurrentWorkLogPath: workLogPath,
		},
	}
}

func (t *runtimeStatusTracker) Start() {
	if t == nil {
		return
	}
	now := t.now()
	t.value.StartedAt = &now
	t.value.StoppedAt = nil
	t.write()
}

func (t *runtimeStatusTracker) Stop() {
	if t == nil {
		return
	}
	now := t.now()
	t.value.StoppedAt = &now
	t.write()
}

func (t *runtimeStatusTracker) RecordSample(sample backend.UsageSample, success bool) {
	if t == nil {
		return
	}
	now := t.now()
	t.value.LastSampleAt = &now
	t.value.SampleCount++
	if success {
		t.value.LastSuccessfulSampleAt = &now
		t.value.LastSuccessfulSample = &status.ActivitySample{
			ProgramName: sample.ProgramName,
			WindowTitle: sample.WindowTitle,
			IdleSeconds: sample.IdleSeconds,
		}
	}
	t.write()
}

func (t *runtimeStatusTracker) RecordWarning(message string) {
	if t == nil || message == "" {
		return
	}
	now := t.now()
	t.value.LatestWarning = &status.RuntimeEvent{At: now, Message: message}
	t.write()
}

func (t *runtimeStatusTracker) RecordError(message string) {
	if t == nil || message == "" {
		return
	}
	now := t.now()
	t.value.LatestError = &status.RuntimeEvent{At: now, Message: message}
	t.write()
}

func (t *runtimeStatusTracker) write() {
	if t == nil || t.store == nil {
		return
	}
	if err := t.store.Write(t.value); err != nil && t.logger != nil {
		t.logger.Print(formatLogMessage("warning", "write runtime status", err, logField{key: "path", value: t.store.Path()}))
	}
}
