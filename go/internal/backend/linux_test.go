package backend

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeLinuxX11Client struct {
	windowID      uint32
	windowErr     error
	windowTitle   string
	titleErr      error
	programName   string
	programErr    error
	idleMillis    uint32
	idleErr       error
	closed        bool
	closeCalls    int
	closeErr      error
	titleWindow   uint32
	programWindow uint32
}

func (f *fakeLinuxX11Client) ActiveWindow() (uint32, error) {
	return f.windowID, f.windowErr
}

func (f *fakeLinuxX11Client) WindowTitle(window uint32) (string, error) {
	f.titleWindow = window
	return f.windowTitle, f.titleErr
}

func (f *fakeLinuxX11Client) ProgramName(window uint32) (string, error) {
	f.programWindow = window
	return f.programName, f.programErr
}

func (f *fakeLinuxX11Client) IdleMilliseconds() (uint32, error) {
	return f.idleMillis, f.idleErr
}

func (f *fakeLinuxX11Client) Close() error {
	f.closed = true
	f.closeCalls++
	return f.closeErr
}

func TestLinuxSampleUsesActiveWindowAndIdleTime(t *testing.T) {
	stubIdentityLookups(t)

	client := &fakeLinuxX11Client{
		windowID:    42,
		windowTitle: "Porting WorkMuch",
		programName: "Code",
		idleMillis:  2500,
	}
	var connectedDisplay string
	backend, err := newLinuxBackend(":99", func(display string) (linuxX11Client, error) {
		connectedDisplay = display
		return client, nil
	})
	require.NoError(t, err)

	sample, err := backend.Sample(context.Background())
	require.NoError(t, err)
	assert.Equal(t, UsageSample{
		Host:        "test-host",
		User:        "test-user",
		WindowTitle: "Porting WorkMuch",
		ProgramName: "Code",
		IdleSeconds: 2.5,
	}, sample)
	assert.Equal(t, ":99", connectedDisplay)
	assert.Equal(t, uint32(42), client.titleWindow)
	assert.Equal(t, uint32(42), client.programWindow)
}

func TestLinuxSampleReturnsPartialDataOnFailures(t *testing.T) {
	stubIdentityLookups(t)

	titleErr := errors.New("title unavailable")
	idleErr := errors.New("idle unavailable")
	client := &fakeLinuxX11Client{
		windowID:    7,
		titleErr:    titleErr,
		programName: "Terminal",
		idleErr:     idleErr,
	}
	backend, err := newLinuxBackend(":1", func(string) (linuxX11Client, error) {
		return client, nil
	})
	require.NoError(t, err)

	sample, err := backend.Sample(context.Background())
	require.Error(t, err)
	assert.ErrorIs(t, err, titleErr)
	assert.ErrorIs(t, err, idleErr)
	assert.Equal(t, "Terminal", sample.ProgramName)
	assert.Empty(t, sample.WindowTitle)
	assert.Zero(t, sample.IdleSeconds)
}

func TestLinuxSampleStillQueriesIdleWhenWindowLookupFails(t *testing.T) {
	stubIdentityLookups(t)

	windowErr := errors.New("window unavailable")
	client := &fakeLinuxX11Client{
		windowErr:  windowErr,
		idleMillis: 1250,
	}
	backend, err := newLinuxBackend(":1", func(string) (linuxX11Client, error) {
		return client, nil
	})
	require.NoError(t, err)

	sample, err := backend.Sample(context.Background())
	require.ErrorIs(t, err, windowErr)
	assert.Equal(t, 1.25, sample.IdleSeconds)
	assert.Zero(t, client.titleWindow)
	assert.Zero(t, client.programWindow)
}

func TestNewLinuxBackendRequiresDisplay(t *testing.T) {
	_, err := newLinuxBackend("", func(string) (linuxX11Client, error) {
		t.Fatal("connector should not be called without DISPLAY")
		return nil, nil
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "DISPLAY")
}

func TestLinuxResetReconnectsToDisplay(t *testing.T) {
	first := &fakeLinuxX11Client{}
	second := &fakeLinuxX11Client{windowID: 8, programName: "Firefox"}
	clients := []linuxX11Client{first, second}
	connectCalls := 0

	backend, err := newLinuxBackend(":2", func(display string) (linuxX11Client, error) {
		assert.Equal(t, ":2", display)
		client := clients[connectCalls]
		connectCalls++
		return client, nil
	})
	require.NoError(t, err)
	require.NoError(t, backend.Reset())

	assert.True(t, first.closed)
	assert.Equal(t, 2, connectCalls)
	sample, err := backend.Sample(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "Firefox", sample.ProgramName)
}

func TestLinuxCloseIsIdempotent(t *testing.T) {
	client := &fakeLinuxX11Client{}
	backend, err := newLinuxBackend(":3", func(string) (linuxX11Client, error) {
		return client, nil
	})
	require.NoError(t, err)

	require.NoError(t, backend.Close())
	require.NoError(t, backend.Close())
	assert.True(t, client.closed)
	assert.Equal(t, 1, client.closeCalls)
}

func TestParseWMClassUsesClassThenFallsBackToInstance(t *testing.T) {
	assert.Equal(t, "Code", parseWMClass([]byte("code\x00Code\x00")))
	assert.Equal(t, "terminal", parseWMClass([]byte("terminal\x00")))
	assert.Empty(t, parseWMClass(nil))
}

func TestFindTopLevelWindowMatchesPythonTraversal(t *testing.T) {
	parents := map[uint32]uint32{10: 20, 20: 30, 30: 1}
	titles := map[uint32]string{20: "Editor"}

	window, err := findTopLevelWindow(
		10,
		1,
		func(window uint32) (uint32, error) { return parents[window], nil },
		func(window uint32) (string, error) { return titles[window], nil },
	)
	require.NoError(t, err)
	assert.Equal(t, uint32(20), window)
}

func TestFindTopLevelWindowStopsAtRootChild(t *testing.T) {
	parents := map[uint32]uint32{10: 20, 20: 1}

	window, err := findTopLevelWindow(
		10,
		1,
		func(window uint32) (uint32, error) { return parents[window], nil },
		func(uint32) (string, error) { return "", nil },
	)
	require.NoError(t, err)
	assert.Equal(t, uint32(20), window)
}
