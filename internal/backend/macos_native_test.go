package backend

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeMacOSNativeAPI struct {
	trustChecks []bool
	trustIndex  int

	windowPID   int
	windowApp   string
	windowTitle string
	windowErr   error

	appPID  int
	appName string
	appErr  error

	focusedTitle    string
	focusedTitleErr error

	idleSeconds float64
	idleErr     error

	titleQueries int
	closed       bool
}

func (f *fakeMacOSNativeAPI) IsAccessibilityTrusted() bool {
	if len(f.trustChecks) == 0 {
		return false
	}
	if f.trustIndex >= len(f.trustChecks) {
		return f.trustChecks[len(f.trustChecks)-1]
	}

	value := f.trustChecks[f.trustIndex]
	f.trustIndex++
	return value
}

func (f *fakeMacOSNativeAPI) GetFrontmostWindowInfo() (int, string, string, error) {
	return f.windowPID, f.windowApp, f.windowTitle, f.windowErr
}

func (f *fakeMacOSNativeAPI) GetFrontmostApplication() (int, string, error) {
	return f.appPID, f.appName, f.appErr
}

func (f *fakeMacOSNativeAPI) GetFocusedWindowTitle(_ int) (string, error) {
	f.titleQueries++
	return f.focusedTitle, f.focusedTitleErr
}

func (f *fakeMacOSNativeAPI) GetIdleSeconds() (float64, error) {
	return f.idleSeconds, f.idleErr
}

func (f *fakeMacOSNativeAPI) Close() error {
	f.closed = true
	return nil
}

func stubIdentityLookups(t *testing.T) {
	t.Helper()

	originalLookupHostname := lookupHostname
	originalLookupUsername := lookupUsername
	lookupHostname = func() (string, error) { return "test-host", nil }
	lookupUsername = func() (string, error) { return "test-user", nil }

	t.Cleanup(func() {
		lookupHostname = originalLookupHostname
		lookupUsername = originalLookupUsername
	})
}

func TestMacOSNativeSampleUsesFrontmostWindowInfo(t *testing.T) {
	stubIdentityLookups(t)

	api := &fakeMacOSNativeAPI{
		trustChecks: []bool{true},
		windowPID:   42,
		windowApp:   "Safari",
		windowTitle: "Current Tab",
		idleSeconds: 2.5,
	}

	backend := newMacOSNativeBackend(api)
	sample, err := backend.Sample(context.Background())
	require.NoError(t, err)

	assert.Equal(t, UsageSample{
		Host:        "test-host",
		User:        "test-user",
		WindowTitle: "Current Tab",
		ProgramName: "Safari",
		IdleSeconds: 2.5,
	}, sample)
	assert.Equal(t, 0, api.titleQueries)
}

func TestMacOSNativeSampleFallsBackToFrontmostApplication(t *testing.T) {
	stubIdentityLookups(t)

	api := &fakeMacOSNativeAPI{
		trustChecks: []bool{false, false},
		appPID:      99,
		appName:     "Terminal",
		idleSeconds: 1.25,
	}

	backend := newMacOSNativeBackend(api)
	sample, err := backend.Sample(context.Background())
	require.NoError(t, err)

	assert.Equal(t, "Terminal", sample.ProgramName)
	assert.Equal(t, "", sample.WindowTitle)
	assert.Equal(t, 1.25, sample.IdleSeconds)
	assert.Equal(t, 0, api.titleQueries)
}

func TestMacOSNativeSampleUsesFocusedWindowTitleWhenTrusted(t *testing.T) {
	stubIdentityLookups(t)

	api := &fakeMacOSNativeAPI{
		trustChecks:  []bool{true},
		windowPID:    7,
		windowApp:    "Safari",
		focusedTitle: "Docs",
		idleSeconds:  0.5,
	}

	backend := newMacOSNativeBackend(api)
	sample, err := backend.Sample(context.Background())
	require.NoError(t, err)

	assert.Equal(t, "Docs", sample.WindowTitle)
	assert.Equal(t, 1, api.titleQueries)
}

func TestMacOSNativeSampleSkipsFocusedTitleWithoutPermission(t *testing.T) {
	stubIdentityLookups(t)

	api := &fakeMacOSNativeAPI{
		trustChecks:  []bool{false, false},
		windowPID:    7,
		windowApp:    "Safari",
		focusedTitle: "Ignored",
		idleSeconds:  0.5,
	}

	backend := newMacOSNativeBackend(api)
	sample, err := backend.Sample(context.Background())
	require.NoError(t, err)

	assert.Equal(t, "", sample.WindowTitle)
	assert.Equal(t, 0, api.titleQueries)
}

func TestMacOSNativeSampleRefreshesAccessibilityTrustAcrossSamples(t *testing.T) {
	stubIdentityLookups(t)

	api := &fakeMacOSNativeAPI{
		trustChecks:  []bool{false, false, true},
		windowPID:    7,
		windowApp:    "Safari",
		focusedTitle: "Docs",
		idleSeconds:  1.0,
	}

	backend := newMacOSNativeBackend(api)

	firstSample, err := backend.Sample(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "", firstSample.WindowTitle)
	assert.Equal(t, 0, api.titleQueries)

	secondSample, err := backend.Sample(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "Docs", secondSample.WindowTitle)
	assert.Equal(t, 1, api.titleQueries)
}

func TestMacOSNativeSampleReturnsPartialDataOnFailures(t *testing.T) {
	stubIdentityLookups(t)

	windowErr := errors.New("window unavailable")
	idleErr := errors.New("idle unavailable")
	api := &fakeMacOSNativeAPI{
		trustChecks: []bool{false, false},
		windowErr:   windowErr,
		appPID:      99,
		appName:     "Terminal",
		idleErr:     idleErr,
	}

	backend := newMacOSNativeBackend(api)
	sample, err := backend.Sample(context.Background())
	require.Error(t, err)

	assert.ErrorIs(t, err, windowErr)
	assert.ErrorIs(t, err, idleErr)
	assert.Equal(t, UsageSample{
		Host:        "test-host",
		User:        "test-user",
		WindowTitle: "",
		ProgramName: "Terminal",
		IdleSeconds: 0,
	}, sample)
}

func TestMacOSNativeResetIsNoOp(t *testing.T) {
	backend := newMacOSNativeBackend(&fakeMacOSNativeAPI{})
	require.NoError(t, backend.Reset())
}

func TestMacOSNativeCloseDelegatesToAPI(t *testing.T) {
	api := &fakeMacOSNativeAPI{}
	backend := newMacOSNativeBackend(api)

	require.NoError(t, backend.Close())
	assert.True(t, api.closed)
}
