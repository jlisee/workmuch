package doctor

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"workmuch-go/internal/macosapp"
)

func TestDetectLinuxSessionReportsWaylandAsUnsupported(t *testing.T) {
	t.Parallel()

	environment := map[string]string{
		"XDG_SESSION_TYPE": "wayland",
		"DISPLAY":          ":1",
		"WAYLAND_DISPLAY":  "wayland-0",
	}

	report := DetectLinuxSession("linux", func(name string) string {
		return environment[name]
	})

	assert.True(t, report.Applicable)
	assert.Equal(t, "wayland", report.Type)
	assert.Equal(t, LinuxSessionUnsupported, report.Support)
	assert.Equal(t, ":1", report.X11Display)
	assert.Equal(t, "wayland-0", report.WaylandDisplay)
	assert.Contains(t, report.Detail, "only X11/Xorg")
}

func TestDetectLinuxSessionInfersX11FromDisplay(t *testing.T) {
	t.Parallel()

	report := DetectLinuxSession("linux", func(name string) string {
		if name == "DISPLAY" {
			return ":0"
		}
		return ""
	})

	assert.True(t, report.Applicable)
	assert.Equal(t, "x11", report.Type)
	assert.Equal(t, LinuxSessionSupported, report.Support)
	assert.Equal(t, ":0", report.X11Display)
}

func TestDetectLinuxSessionIsNotApplicableOutsideLinux(t *testing.T) {
	t.Parallel()

	report := DetectLinuxSession("darwin", func(string) string {
		return "unexpected"
	})

	assert.False(t, report.Applicable)
}

type fakeLoginItemService struct {
	state macosapp.LoginItemState
	err   error
}

func (f fakeLoginItemService) Status() (macosapp.LoginItemState, error) {
	return f.state, f.err
}

func (fakeLoginItemService) Register() error {
	return nil
}

func TestLoginItemCheckerReportsEveryServiceManagementState(t *testing.T) {
	t.Parallel()

	states := []macosapp.LoginItemState{
		LoginItemNotRegistered,
		LoginItemEnabled,
		LoginItemRequiresApproval,
		LoginItemNotFound,
		LoginItemUnsupported,
	}
	for _, state := range states {
		checker := NativeLoginItemChecker{
			Platform: "darwin",
			Service:  fakeLoginItemService{state: state},
		}

		report := checker.Check(context.Background())

		assert.Equal(t, state, report.State)
		assert.Empty(t, report.Error)
	}
}

func TestLoginItemCheckerReportsFrameworkError(t *testing.T) {
	checker := NativeLoginItemChecker{
		Platform: "darwin",
		Service: fakeLoginItemService{
			state: LoginItemUnsupported,
			err:   errors.New("ServiceManagement unavailable"),
		},
	}

	report := checker.Check(context.Background())

	assert.Equal(t, LoginItemUnsupported, report.State)
	assert.ErrorContains(t, errors.New(report.Error), "ServiceManagement unavailable")
}

func TestLoginItemCheckerIsNotApplicableOutsideDarwin(t *testing.T) {
	t.Parallel()

	report := (NativeLoginItemChecker{Platform: "linux"}).Check(context.Background())

	assert.Equal(t, LoginItemNotApplicable, report.State)
}
