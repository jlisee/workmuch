package backend

import (
	"errors"
	"testing"
)

func TestNewBackendAutoSelectsMacOSSubprocess(t *testing.T) {
	b, err := NewBackend("darwin", BackendAuto)
	if err != nil {
		t.Fatalf("NewBackend returned error: %v", err)
	}
	if b.Name() != BackendMacOSSubprocess {
		t.Fatalf("expected %q backend, got %q", BackendMacOSSubprocess, b.Name())
	}
}

func TestNewBackendUnknownBackend(t *testing.T) {
	_, err := NewBackend("darwin", "does-not-exist")
	if err == nil {
		t.Fatalf("expected error for unknown backend")
	}
}

func TestNewBackendLinuxNotImplemented(t *testing.T) {
	_, err := NewBackend("linux", BackendLinux)
	if err == nil {
		t.Fatalf("expected not implemented error")
	}
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
	if err != nil {
		t.Fatalf("completeSample returned error: %v", err)
	}
	if sample.Host != "test-host" {
		t.Fatalf("unexpected host: %q", sample.Host)
	}
	if sample.User != "test-user" {
		t.Fatalf("unexpected user: %q", sample.User)
	}
	if sample.WindowTitle != "window" || sample.ProgramName != "program" || sample.IdleSeconds != 3.5 {
		t.Fatalf("unexpected sample contents: %#v", sample)
	}
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
	if !errors.Is(err, hostErr) {
		t.Fatalf("expected host error, got %v", err)
	}
	if !errors.Is(err, userErr) {
		t.Fatalf("expected user error, got %v", err)
	}
	if sample.Host != "" || sample.User != "" {
		t.Fatalf("expected empty identity fields, got %#v", sample)
	}
	if sample.ProgramName != "program" {
		t.Fatalf("unexpected sample contents: %#v", sample)
	}
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
	if err != nil {
		t.Fatalf("completeSample returned error: %v", err)
	}
	if sample.Host != "Josephs-MacBook-Pro" {
		t.Fatalf("unexpected host: %q", sample.Host)
	}
}
