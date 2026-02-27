package backend

import "testing"

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
