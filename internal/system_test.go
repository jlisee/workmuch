package internal

import (
	"errors"
	"os/user"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	require.NoError(t, err)
	assert.Equal(t, "env-user", username)
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
	require.NoError(t, err)
	assert.Equal(t, "windows-user", username)
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
	require.NoError(t, err)
	assert.Equal(t, "lookup-user", username)
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
	assert.ErrorIs(t, err, expectedErr)
}
