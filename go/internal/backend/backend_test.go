package backend

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewBackendAutoSelectsMacOSSubprocess(t *testing.T) {
	b, err := NewBackend("darwin", BackendAuto)
	require.NoError(t, err)
	assert.Equal(t, BackendMacOSSubprocess, b.Name())
}

func TestNewBackendUnknownBackend(t *testing.T) {
	_, err := NewBackend("darwin", "does-not-exist")
	require.Error(t, err)
}

func TestNewBackendLinuxNotImplemented(t *testing.T) {
	_, err := NewBackend("linux", BackendLinux)
	require.Error(t, err)
}

func TestCompleteSampleAddsIdentity(t *testing.T) {
	originalLookupHostname := lookupHostname
	originalLookupUsername := lookupUsername
	lookupHostname = func() (string, error) { return "test-host", nil }
	lookupUsername = func() (string, error) { return "test-user", nil }
	defer func() {
		lookupHostname = originalLookupHostname
		lookupUsername = originalLookupUsername
	}()

	sample, err := completeSample(UsageSample{
		WindowTitle: "window",
		ProgramName: "program",
		IdleSeconds: 3.5,
	})
	require.NoError(t, err)
	assert.Equal(t, UsageSample{
		Host:        "test-host",
		User:        "test-user",
		WindowTitle: "window",
		ProgramName: "program",
		IdleSeconds: 3.5,
	}, sample)
}

func TestCompleteSampleReturnsLookupErrors(t *testing.T) {
	hostErr := errors.New("host failed")
	userErr := errors.New("user failed")

	originalLookupHostname := lookupHostname
	originalLookupUsername := lookupUsername
	lookupHostname = func() (string, error) { return "", hostErr }
	lookupUsername = func() (string, error) { return "", userErr }
	defer func() {
		lookupHostname = originalLookupHostname
		lookupUsername = originalLookupUsername
	}()

	sample, err := completeSample(UsageSample{ProgramName: "program"})
	require.Error(t, err)
	assert.ErrorIs(t, err, hostErr)
	assert.ErrorIs(t, err, userErr)
	assert.Equal(t, UsageSample{ProgramName: "program"}, sample)
}

func TestCompleteSampleNormalizesLocalHostnameSuffix(t *testing.T) {
	originalLookupHostname := lookupHostname
	originalLookupUsername := lookupUsername
	lookupHostname = func() (string, error) { return "Josephs-MacBook-Pro.LOCAL", nil }
	lookupUsername = func() (string, error) { return "test-user", nil }
	defer func() {
		lookupHostname = originalLookupHostname
		lookupUsername = originalLookupUsername
	}()

	sample, err := completeSample(UsageSample{})
	require.NoError(t, err)
	assert.Equal(t, "Josephs-MacBook-Pro", sample.Host)
}
