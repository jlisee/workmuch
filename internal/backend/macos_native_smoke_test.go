//go:build darwin && cgo

package backend

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMacOSNativeSmoke(t *testing.T) {
	if os.Getenv("WORKMUCH_RUN_MACOS_SMOKE") != "1" {
		t.Skip("set WORKMUCH_RUN_MACOS_SMOKE=1 to run the macOS smoke test")
	}

	backend, err := NewMacOSNativeBackend()
	require.NoError(t, err)
	defer func() {
		require.NoError(t, backend.Close())
	}()

	sample, err := backend.Sample(context.Background())
	require.NoError(t, err)
	if sample.ProgramName == "" && sample.WindowTitle == "" {
		t.Skip("macOS did not expose an active frontmost GUI application in this session")
	}
	assert.NotEmpty(t, sample.ProgramName)
	assert.GreaterOrEqual(t, sample.IdleSeconds, 0.0)
}
