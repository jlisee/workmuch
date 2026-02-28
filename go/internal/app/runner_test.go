package app

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestComputeNextSleepOnSchedule(t *testing.T) {
	now := time.Date(2026, time.February, 27, 10, 0, 0, 0, time.UTC)
	wakeAt := now.Add(2 * time.Second)
	period := 2 * time.Second

	sleepDuration, nextWakeAt := ComputeNextSleep(now, wakeAt, period)
	assert.Equal(t, 2*time.Second, sleepDuration)
	assert.True(t, nextWakeAt.Equal(wakeAt.Add(period)))
}

func TestComputeNextSleepWhenBehind(t *testing.T) {
	now := time.Date(2026, time.February, 27, 10, 0, 7, 0, time.UTC)
	wakeAt := time.Date(2026, time.February, 27, 10, 0, 2, 0, time.UTC)
	period := 2 * time.Second

	sleepDuration, nextWakeAt := ComputeNextSleep(now, wakeAt, period)
	assert.GreaterOrEqual(t, sleepDuration, time.Duration(0))
	expectedCurrentWake := time.Date(2026, time.February, 27, 10, 0, 8, 0, time.UTC)
	assert.Equal(t, expectedCurrentWake.Sub(now), sleepDuration)
	assert.True(t, nextWakeAt.Equal(expectedCurrentWake.Add(period)))
}
