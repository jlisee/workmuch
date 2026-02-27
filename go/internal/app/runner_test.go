package app

import (
	"testing"
	"time"
)

func TestComputeNextSleepOnSchedule(t *testing.T) {
	now := time.Date(2026, time.February, 27, 10, 0, 0, 0, time.UTC)
	wakeAt := now.Add(2 * time.Second)
	period := 2 * time.Second

	sleepDuration, nextWakeAt := ComputeNextSleep(now, wakeAt, period)
	if sleepDuration != 2*time.Second {
		t.Fatalf("unexpected sleep duration: %v", sleepDuration)
	}
	if !nextWakeAt.Equal(wakeAt.Add(period)) {
		t.Fatalf("unexpected next wake: %v", nextWakeAt)
	}
}

func TestComputeNextSleepWhenBehind(t *testing.T) {
	now := time.Date(2026, time.February, 27, 10, 0, 7, 0, time.UTC)
	wakeAt := time.Date(2026, time.February, 27, 10, 0, 2, 0, time.UTC)
	period := 2 * time.Second

	sleepDuration, nextWakeAt := ComputeNextSleep(now, wakeAt, period)
	if sleepDuration < 0 {
		t.Fatalf("sleep duration must never be negative: %v", sleepDuration)
	}
	expectedCurrentWake := time.Date(2026, time.February, 27, 10, 0, 8, 0, time.UTC)
	if sleepDuration != expectedCurrentWake.Sub(now) {
		t.Fatalf("unexpected sleep duration: %v", sleepDuration)
	}
	if !nextWakeAt.Equal(expectedCurrentWake.Add(period)) {
		t.Fatalf("unexpected next wake: %v", nextWakeAt)
	}
}
