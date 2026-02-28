package backend

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"
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
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if sample.Host != "test-host" {
		t.Fatalf("unexpected host: %q", sample.Host)
	}
	if sample.User != "test-user" {
		t.Fatalf("unexpected user: %q", sample.User)
	}
	if sample.ProgramName != "Safari" {
		t.Fatalf("unexpected program name: %q", sample.ProgramName)
	}
	if sample.WindowTitle != "WorkMuch" {
		t.Fatalf("unexpected window title: %q", sample.WindowTitle)
	}
	if sample.IdleSeconds != 2.5 {
		t.Fatalf("unexpected idle seconds: %f", sample.IdleSeconds)
	}
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
	if err == nil {
		t.Fatalf("expected aggregate error")
	}
	if sample.ProgramName != "" || sample.WindowTitle != "" || sample.IdleSeconds != 0 {
		t.Fatalf("expected safe default sample, got %#v", sample)
	}
}

func TestParseIdleSeconds(t *testing.T) {
	idleSeconds, err := ParseIdleSeconds("\"HIDIdleTime\" = 12345000000")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if idleSeconds != 12.345 {
		t.Fatalf("unexpected idle seconds: %f", idleSeconds)
	}
}

func TestParseIdleSecondsMissingField(t *testing.T) {
	_, err := ParseIdleSeconds("nothing useful")
	if err == nil {
		t.Fatalf("expected parse error")
	}
}
