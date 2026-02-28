package backend

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeExecutor struct {
	outputs map[string]fakeCommandOutput
}

type fakeCommandOutput struct {
	stdout string
	err    error
}

func (f *fakeExecutor) Run(_ context.Context, _ time.Duration, name string, args ...string) (string, error) {
	key := commandKey(name, args...)
	result, ok := f.outputs[key]
	if !ok {
		return "", fmt.Errorf("unexpected command: %s", key)
	}
	return result.stdout, result.err
}

func commandKey(name string, args ...string) string {
	return name + "|" + strings.Join(args, "|")
}

func TestMacOSSubprocessSampleSuccess(t *testing.T) {
	originalLookupHostname := lookupHostname
	originalLookupUsername := lookupUsername
	lookupHostname = func() (string, error) { return "test-host", nil }
	lookupUsername = func() (string, error) { return "test-user", nil }
	defer func() {
		lookupHostname = originalLookupHostname
		lookupUsername = originalLookupUsername
	}()

	fake := &fakeExecutor{
		outputs: map[string]fakeCommandOutput{
			commandKey(appNameCommand[0], appNameCommand[1:]...):         {stdout: "Safari\n"},
			commandKey(windowTitleCommand[0], windowTitleCommand[1:]...): {stdout: "WorkMuch\n"},
			commandKey(idleTimeCommand[0], idleTimeCommand[1:]...):       {stdout: "\"HIDIdleTime\" = 2500000000\n"},
		},
	}

	backend := newMacOSSubprocessBackend(fake)
	sample, err := backend.Sample(context.Background())
	require.NoError(t, err)
	assert.Equal(t, UsageSample{
		Host:        "test-host",
		User:        "test-user",
		ProgramName: "Safari",
		WindowTitle: "WorkMuch",
		IdleSeconds: 2.5,
	}, sample)
}

func TestMacOSSubprocessSampleFailureReturnsSafeDefaults(t *testing.T) {
	fake := &fakeExecutor{
		outputs: map[string]fakeCommandOutput{
			commandKey(appNameCommand[0], appNameCommand[1:]...):         {err: errors.New("no osascript")},
			commandKey(windowTitleCommand[0], windowTitleCommand[1:]...): {err: errors.New("no osascript")},
			commandKey(idleTimeCommand[0], idleTimeCommand[1:]...):       {stdout: "bad output"},
		},
	}

	backend := newMacOSSubprocessBackend(fake)
	sample, err := backend.Sample(context.Background())
	require.Error(t, err)
	assert.Empty(t, sample.ProgramName)
	assert.Empty(t, sample.WindowTitle)
	assert.Zero(t, sample.IdleSeconds)
}

func TestParseIdleSeconds(t *testing.T) {
	idleSeconds, err := ParseIdleSeconds("\"HIDIdleTime\" = 12345000000")
	require.NoError(t, err)
	assert.Equal(t, 12.345, idleSeconds)
}

func TestParseIdleSecondsMissingField(t *testing.T) {
	_, err := ParseIdleSeconds("nothing useful")
	require.Error(t, err)
}
