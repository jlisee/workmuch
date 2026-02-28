package internal

import (
	"errors"
	"os/user"
	"testing"
)

func TestUsernameUsesUSER(t *testing.T) {
	originalLookupCurrentUser := lookupCurrentUser
	lookupCurrentUser = func() (*user.User, error) {
		t.Fatalf("lookupCurrentUser should not be called when USER is set")
		return nil, nil
	}
	defer func() {
		lookupCurrentUser = originalLookupCurrentUser
	}()

	t.Setenv("USER", "env-user")
	t.Setenv("USERNAME", "")

	username, err := Username()
	if err != nil {
		t.Fatalf("Username returned error: %v", err)
	}
	if username != "env-user" {
		t.Fatalf("unexpected username: %q", username)
	}
}

func TestUsernameUsesUSERNAMEWhenUSERUnset(t *testing.T) {
	originalLookupCurrentUser := lookupCurrentUser
	lookupCurrentUser = func() (*user.User, error) {
		t.Fatalf("lookupCurrentUser should not be called when USERNAME is set")
		return nil, nil
	}
	defer func() {
		lookupCurrentUser = originalLookupCurrentUser
	}()

	t.Setenv("USER", "")
	t.Setenv("USERNAME", "windows-user")

	username, err := Username()
	if err != nil {
		t.Fatalf("Username returned error: %v", err)
	}
	if username != "windows-user" {
		t.Fatalf("unexpected username: %q", username)
	}
}

func TestUsernameFallsBackToCurrentUser(t *testing.T) {
	originalLookupCurrentUser := lookupCurrentUser
	lookupCurrentUser = func() (*user.User, error) {
		return &user.User{Username: "lookup-user"}, nil
	}
	defer func() {
		lookupCurrentUser = originalLookupCurrentUser
	}()

	t.Setenv("USER", "")
	t.Setenv("USERNAME", "")

	username, err := Username()
	if err != nil {
		t.Fatalf("Username returned error: %v", err)
	}
	if username != "lookup-user" {
		t.Fatalf("unexpected username: %q", username)
	}
}

func TestUsernameReturnsLookupError(t *testing.T) {
	expectedErr := errors.New("lookup failed")

	originalLookupCurrentUser := lookupCurrentUser
	lookupCurrentUser = func() (*user.User, error) {
		return nil, expectedErr
	}
	defer func() {
		lookupCurrentUser = originalLookupCurrentUser
	}()

	t.Setenv("USER", "")
	t.Setenv("USERNAME", "")

	_, err := Username()
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected error %v, got %v", expectedErr, err)
	}
}
