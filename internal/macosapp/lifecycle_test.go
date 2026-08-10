package macosapp

import (
	"bytes"
	"errors"
	"log"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeLoginItem struct {
	state         LoginItemState
	statusErr     error
	registerErr   error
	statusCalls   int
	registerCalls int
}

func (f *fakeLoginItem) Status() (LoginItemState, error) {
	f.statusCalls++
	return f.state, f.statusErr
}

func (f *fakeLoginItem) Register() error {
	f.registerCalls++
	return f.registerErr
}

type fakeAccessibility struct {
	trusted      bool
	trustedErr   error
	promptErr    error
	trustedCalls int
	promptCalls  int
}

func (f *fakeAccessibility) IsTrusted() (bool, error) {
	f.trustedCalls++
	return f.trusted, f.trustedErr
}

func (f *fakeAccessibility) Prompt() error {
	f.promptCalls++
	return f.promptErr
}

type fakeMoveDialog struct {
	err   error
	calls int
}

func (f *fakeMoveDialog) Show() error {
	f.calls++
	return f.err
}

func TestLoginItemStatusMapping(t *testing.T) {
	t.Parallel()

	tests := []struct {
		raw  int
		want LoginItemState
	}{
		{raw: 0, want: LoginItemNotRegistered},
		{raw: 1, want: LoginItemEnabled},
		{raw: 2, want: LoginItemRequiresApproval},
		{raw: 3, want: LoginItemNotFound},
		{raw: -1, want: LoginItemUnsupported},
		{raw: 99, want: LoginItemUnsupported},
	}
	for _, test := range tests {
		assert.Equal(t, test.want, mapLoginItemStatus(test.raw))
	}
}

func TestInstalledBundleRegistersFirstRunAndPromptsForAccessibility(t *testing.T) {
	t.Parallel()

	loginItem := &fakeLoginItem{state: LoginItemNotRegistered}
	accessibility := &fakeAccessibility{}
	dialog := &fakeMoveDialog{}
	proceed, err := PrepareBundledTrayLaunch(LaunchEnvironment{
		Platform:       "darwin",
		ExecutablePath: InstalledExecutablePath,
	}, LaunchDependencies{
		LoginItem:     loginItem,
		Accessibility: accessibility,
		MoveDialog:    dialog,
	}, log.New(&bytes.Buffer{}, "", 0))

	require.NoError(t, err)
	assert.True(t, proceed)
	assert.Equal(t, 1, loginItem.statusCalls)
	assert.Equal(t, 1, loginItem.registerCalls)
	assert.Equal(t, 1, accessibility.trustedCalls)
	assert.Equal(t, 1, accessibility.promptCalls)
	assert.Zero(t, dialog.calls)
}

func TestEnabledAndApprovalRequiredLoginItemsAreNotRegisteredAgain(t *testing.T) {
	t.Parallel()

	for _, state := range []LoginItemState{LoginItemEnabled, LoginItemRequiresApproval} {
		loginItem := &fakeLoginItem{state: state}
		proceed, err := PrepareBundledTrayLaunch(LaunchEnvironment{
			Platform:       "darwin",
			ExecutablePath: InstalledExecutablePath,
		}, LaunchDependencies{
			LoginItem:     loginItem,
			Accessibility: &fakeAccessibility{trusted: true},
			MoveDialog:    &fakeMoveDialog{},
		}, log.New(&bytes.Buffer{}, "", 0))

		require.NoError(t, err)
		assert.True(t, proceed)
		assert.Zero(t, loginItem.registerCalls)
	}
}

func TestFrameworkFailuresAreLoggedAndDoNotStopInstalledBundle(t *testing.T) {
	t.Parallel()

	var output bytes.Buffer
	loginItem := &fakeLoginItem{
		state:       LoginItemNotRegistered,
		registerErr: errors.New("registration denied"),
	}
	accessibility := &fakeAccessibility{trustedErr: errors.New("trust query failed")}
	proceed, err := PrepareBundledTrayLaunch(LaunchEnvironment{
		Platform:       "darwin",
		ExecutablePath: InstalledExecutablePath,
	}, LaunchDependencies{
		LoginItem:     loginItem,
		Accessibility: accessibility,
		MoveDialog:    &fakeMoveDialog{},
	}, log.New(&output, "", 0))

	require.NoError(t, err)
	assert.True(t, proceed)
	assert.Contains(t, output.String(), "registration denied")
	assert.Contains(t, output.String(), "trust query failed")
	assert.Zero(t, accessibility.promptCalls)
}

func TestBundleOutsideApplicationsShowsMoveDialogAndStopsBeforeFrameworkCalls(t *testing.T) {
	t.Parallel()

	loginItem := &fakeLoginItem{}
	accessibility := &fakeAccessibility{}
	dialog := &fakeMoveDialog{}
	proceed, err := PrepareBundledTrayLaunch(LaunchEnvironment{
		Platform:       "darwin",
		ExecutablePath: "/Volumes/WorkMuch/WorkMuch.app/Contents/MacOS/workmuch",
	}, LaunchDependencies{
		LoginItem:     loginItem,
		Accessibility: accessibility,
		MoveDialog:    dialog,
	}, log.New(&bytes.Buffer{}, "", 0))

	require.NoError(t, err)
	assert.False(t, proceed)
	assert.Equal(t, 1, dialog.calls)
	assert.Zero(t, loginItem.statusCalls)
	assert.Zero(t, loginItem.registerCalls)
	assert.Zero(t, accessibility.trustedCalls)
	assert.Zero(t, accessibility.promptCalls)
}

func TestCheckoutTrayLaunchDoesNotUseInstalledAppFrameworkBehavior(t *testing.T) {
	t.Parallel()

	loginItem := &fakeLoginItem{}
	accessibility := &fakeAccessibility{}
	dialog := &fakeMoveDialog{}
	proceed, err := PrepareBundledTrayLaunch(LaunchEnvironment{
		Platform:       "darwin",
		ExecutablePath: "/Users/dev/workmuch/bin/workmuch",
	}, LaunchDependencies{
		LoginItem:     loginItem,
		Accessibility: accessibility,
		MoveDialog:    dialog,
	}, log.New(&bytes.Buffer{}, "", 0))

	require.NoError(t, err)
	assert.True(t, proceed)
	assert.Zero(t, dialog.calls)
	assert.Zero(t, loginItem.statusCalls)
	assert.Zero(t, accessibility.trustedCalls)
}

func TestTrustedInstalledBundleDoesNotPromptAgain(t *testing.T) {
	t.Parallel()

	accessibility := &fakeAccessibility{trusted: true}
	proceed, err := PrepareBundledTrayLaunch(LaunchEnvironment{
		Platform:       "darwin",
		ExecutablePath: InstalledExecutablePath,
	}, LaunchDependencies{
		LoginItem:     &fakeLoginItem{state: LoginItemEnabled},
		Accessibility: accessibility,
		MoveDialog:    &fakeMoveDialog{},
	}, log.New(&bytes.Buffer{}, "", 0))

	require.NoError(t, err)
	assert.True(t, proceed)
	assert.Equal(t, 1, accessibility.trustedCalls)
	assert.Zero(t, accessibility.promptCalls)
}
