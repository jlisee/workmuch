package tray

import (
	"context"
	"errors"
	"log"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"workmuch-go/internal/app"
)

type fakeMenuItem struct {
	title    string
	tooltip  string
	clicked  chan struct{}
	disabled bool
}

func newFakeMenuItem(title string, tooltip string) *fakeMenuItem {
	return &fakeMenuItem{
		title:   title,
		tooltip: tooltip,
		clicked: make(chan struct{}, 1),
	}
}

func (i *fakeMenuItem) ClickedCh() <-chan struct{} {
	return i.clicked
}

func (i *fakeMenuItem) Disable() {
	i.disabled = true
}

type fakeDriver struct {
	icon               []byte
	title              string
	tooltip            string
	menuItems          []*fakeMenuItem
	quitCalls          int
	quitTriggersOnExit bool
	onReady            func()
	onExit             func()
}

func (d *fakeDriver) Run(onReady func(), onExit func()) {
	d.onReady = onReady
	d.onExit = onExit
	onReady()
}

func (d *fakeDriver) Quit() {
	d.quitCalls++
	if d.quitTriggersOnExit && d.onExit != nil {
		d.onExit()
	}
}

func (d *fakeDriver) SetIcon(icon []byte) {
	d.icon = append([]byte(nil), icon...)
}

func (d *fakeDriver) SetTitle(title string) {
	d.title = title
}

func (d *fakeDriver) SetTooltip(tooltip string) {
	d.tooltip = tooltip
}

func (d *fakeDriver) AddMenuItem(title string, tooltip string) menuItem {
	item := newFakeMenuItem(title, tooltip)
	d.menuItems = append(d.menuItems, item)
	return item
}

type fakeController struct {
	startCalls int
	waitErr    error
	errCh      chan error
	startCtx   context.Context
}

func newFakeController() *fakeController {
	return &fakeController{errCh: make(chan error, 1)}
}

func (c *fakeController) Start(ctx context.Context) <-chan error {
	c.startCalls++
	c.startCtx = ctx
	return c.errCh
}

func (c *fakeController) Wait() error {
	return c.waitErr
}

func TestRunSetsUpTrayAndStartsCollector(t *testing.T) {
	t.Parallel()

	driver := &fakeDriver{quitTriggersOnExit: true}
	ctrl := newFakeController()
	r := newRunner(driver, func(app.Options, *log.Logger) controller {
		return ctrl
	})

	done := make(chan error, 1)
	go func() {
		done <- r.Run(context.Background(), app.DefaultOptions(), log.Default())
	}()

	require.Eventually(t, func() bool {
		return len(driver.menuItems) == 2
	}, time.Second, 10*time.Millisecond)

	assert.NotEmpty(t, driver.icon)
	assert.Equal(t, "", driver.title)
	assert.Equal(t, "Logging active", driver.tooltip)
	assert.Equal(t, "About", driver.menuItems[0].title)
	assert.Equal(t, "About Workmuch", driver.menuItems[0].tooltip)
	assert.False(t, driver.menuItems[0].disabled)
	assert.Equal(t, "Quit", driver.menuItems[1].title)
	assert.Equal(t, "Quit", driver.menuItems[1].tooltip)
	assert.Equal(t, 1, ctrl.startCalls)
	require.NotNil(t, ctrl.startCtx)

	driver.Quit()
	require.NoError(t, <-done)
}

func TestRunQuitMenuClickStopsCollectorAndExitsTray(t *testing.T) {
	t.Parallel()

	driver := &fakeDriver{quitTriggersOnExit: false}
	ctrl := newFakeController()
	r := newRunner(driver, func(app.Options, *log.Logger) controller {
		return ctrl
	})

	done := make(chan error, 1)
	go func() {
		done <- r.Run(context.Background(), app.DefaultOptions(), log.Default())
	}()

	require.Eventually(t, func() bool {
		return len(driver.menuItems) == 2
	}, time.Second, 10*time.Millisecond)

	driver.menuItems[1].clicked <- struct{}{}

	select {
	case err := <-done:
		require.NoError(t, err)
	case <-time.After(time.Second):
		t.Fatal("run did not finish after quit click")
	}
	assert.Equal(t, 1, driver.quitCalls)
}

func TestRunAboutMenuClickInvokesAboutHandler(t *testing.T) {
	t.Parallel()

	driver := &fakeDriver{quitTriggersOnExit: true}
	ctrl := newFakeController()
	var aboutCalls int
	r := newRunner(driver, func(app.Options, *log.Logger) controller {
		return ctrl
	})
	r.showAbout = func() error {
		aboutCalls++
		return nil
	}

	done := make(chan error, 1)
	go func() {
		done <- r.Run(context.Background(), app.DefaultOptions(), log.Default())
	}()

	require.Eventually(t, func() bool {
		return len(driver.menuItems) == 2
	}, time.Second, 10*time.Millisecond)

	driver.menuItems[0].clicked <- struct{}{}

	require.Eventually(t, func() bool {
		return aboutCalls == 1
	}, time.Second, 10*time.Millisecond)
	assert.Equal(t, 0, driver.quitCalls)

	driver.Quit()
	require.NoError(t, <-done)
}

func TestRunContextCancelStopsCollectorAndExitsTray(t *testing.T) {
	t.Parallel()

	driver := &fakeDriver{quitTriggersOnExit: true}
	ctrl := newFakeController()
	r := newRunner(driver, func(app.Options, *log.Logger) controller {
		return ctrl
	})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan error, 1)
	go func() {
		done <- r.Run(ctx, app.DefaultOptions(), log.Default())
	}()

	require.Eventually(t, func() bool {
		return len(driver.menuItems) == 2
	}, time.Second, 10*time.Millisecond)

	cancel()

	require.NoError(t, <-done)
	assert.Equal(t, 1, driver.quitCalls)
	require.NotNil(t, ctrl.startCtx)
	assert.ErrorIs(t, ctrl.startCtx.Err(), context.Canceled)
}

func TestRunReturnsCollectorFailure(t *testing.T) {
	t.Parallel()

	driver := &fakeDriver{quitTriggersOnExit: true}
	ctrl := newFakeController()
	expectedErr := errors.New("collector failed")
	ctrl.errCh <- expectedErr
	ctrl.waitErr = expectedErr
	r := newRunner(driver, func(app.Options, *log.Logger) controller {
		return ctrl
	})

	require.ErrorIs(t, r.Run(context.Background(), app.DefaultOptions(), log.Default()), expectedErr)
	assert.Equal(t, 1, driver.quitCalls)
}
