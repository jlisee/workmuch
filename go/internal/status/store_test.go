package status

import (
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStoreWriteReadRoundTrip(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "status.json")
	store := NewStore(path)
	startedAt := time.Date(2026, time.July, 8, 9, 0, 0, 0, time.UTC)
	sampledAt := startedAt.Add(5 * time.Second)
	warningAt := sampledAt.Add(time.Second)
	status := RuntimeStatus{
		StartedAt:              &startedAt,
		LastSampleAt:           &sampledAt,
		LastSuccessfulSampleAt: &sampledAt,
		SampleCount:            3,
		SelectedBackend:        "auto",
		ActiveBackend:          "macos-native",
		CurrentWorkLogPath:     "/Users/test/.workmuch/2026-07-08.worklog",
		LatestWarning: &RuntimeEvent{
			At:      warningAt,
			Message: "backend sample warning backend=macos-native: no title",
		},
	}

	require.NoError(t, store.Write(status))

	loaded, err := store.Read()
	require.NoError(t, err)
	assert.Equal(t, status, loaded)
	assert.Equal(t, path, store.Path())
}

func TestStoreReadMissingReturnsNotFound(t *testing.T) {
	t.Parallel()

	store := NewStore(filepath.Join(t.TempDir(), "missing", "status.json"))

	loaded, err := store.Read()
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrNotFound))
	assert.Equal(t, RuntimeStatus{}, loaded)
}

func TestStoreWriteReplacesExistingFile(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "status.json")
	store := NewStore(path)
	firstStartedAt := time.Date(2026, time.July, 8, 9, 0, 0, 0, time.UTC)
	secondStartedAt := firstStartedAt.Add(time.Minute)

	require.NoError(t, store.Write(RuntimeStatus{
		StartedAt:   &firstStartedAt,
		SampleCount: 1,
	}))
	require.NoError(t, store.Write(RuntimeStatus{
		StartedAt:   &secondStartedAt,
		SampleCount: 2,
	}))

	loaded, err := store.Read()
	require.NoError(t, err)
	require.NotNil(t, loaded.StartedAt)
	assert.Equal(t, secondStartedAt, *loaded.StartedAt)
	assert.Equal(t, int64(2), loaded.SampleCount)
}
