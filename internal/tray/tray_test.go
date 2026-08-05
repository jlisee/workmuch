package tray

import (
	"context"
	"errors"
	"log"
	"sync"
	"sync/atomic"
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
	quitCalls          atomic.Int64
	quitTriggersOnExit bool
	onReady            func()
	onExit             func()
	ready              chan struct{}
	quit               chan struct{}
	quitOnce           sync.Once
}

func newFakeDriver(quitTriggersOnExit bool) *fakeDriver {
	return &fakeDriver{
		quitTriggersOnExit: quitTriggersOnExit,
		ready:              make(chan struct{}),
		quit:               make(chan struct{}),
	}
}

func (d *fakeDriver) Run(onReady func(), onExit func()) {
	d.onReady = onReady
	d.onExit = onExit
	onReady()
	close(d.ready)
	<-d.quit
	if d.quitTriggersOnExit {
		onExit()
	}
}

func (d *fakeDriver) Quit() {
	d.quitCalls.Add(1)
	d.quitOnce.Do(func() { close(d.quit) })
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

func waitForDriverReady(t *testing.T, driver *fakeDriver) {
	t.Helper()
	select {
	case <-driver.ready:
	case <-time.After(time.Second):
		t.Fatal("tray did not become ready")
	}
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

	driver := newFakeDriver(true)
	ctrl := newFakeController()
	r := newRunner(driver, func(app.Options, *log.Logger) controller {
		return ctrl
	})

	done := make(chan error, 1)
	go func() {
		done <- r.Run(context.Background(), app.DefaultOptions(), log.Default())
	}()

	waitForDriverReady(t, driver)

	assert.NotEmpty(t, driver.icon)
	assert.Equal(t, "", driver.title)
	assert.Equal(t, "Logging active", driver.tooltip)
	assert.Equal(t, "About", driver.menuItems[0].title)
	assert.Equal(t, "About Workmuch", driver.menuItems[0].tooltip)
	assert.False(t, driver.menuItems[0].disabled)
	assert.Equal(t, "Status", driver.menuItems[1].title)
	assert.Equal(t, "Show Workmuch Status", driver.menuItems[1].tooltip)
	assert.False(t, driver.menuItems[1].disabled)
	assert.Equal(t, "Quit", driver.menuItems[2].title)
	assert.Equal(t, "Quit", driver.menuItems[2].tooltip)
	assert.Equal(t, 1, ctrl.startCalls)
	require.NotNil(t, ctrl.startCtx)

	driver.Quit()
	require.NoError(t, <-done)
}

func TestRunQuitMenuClickStopsCollectorAndExitsTray(t *testing.T) {
	t.Parallel()

	driver := newFakeDriver(false)
	ctrl := newFakeController()
	r := newRunner(driver, func(app.Options, *log.Logger) controller {
		return ctrl
	})

	done := make(chan error, 1)
	go func() {
		done <- r.Run(context.Background(), app.DefaultOptions(), log.Default())
	}()

	waitForDriverReady(t, driver)

	driver.menuItems[2].clicked <- struct{}{}

	select {
	case err := <-done:
		require.NoError(t, err)
	case <-time.After(time.Second):
		t.Fatal("run did not finish after quit click")
	}
	assert.Equal(t, int64(1), driver.quitCalls.Load())
}

func TestRunAboutMenuClickInvokesAboutHandler(t *testing.T) {
	t.Parallel()

	driver := newFakeDriver(true)
	ctrl := newFakeController()
	aboutCalled := make(chan struct{}, 1)
	r := newRunner(driver, func(app.Options, *log.Logger) controller {
		return ctrl
	})
	r.showAbout = func() error {
		aboutCalled <- struct{}{}
		return nil
	}

	done := make(chan error, 1)
	go func() {
		done <- r.Run(context.Background(), app.DefaultOptions(), log.Default())
	}()

	waitForDriverReady(t, driver)

	driver.menuItems[0].clicked <- struct{}{}

	select {
	case <-aboutCalled:
	case <-time.After(time.Second):
		t.Fatal("about handler was not called")
	}
	assert.Equal(t, int64(0), driver.quitCalls.Load())

	driver.Quit()
	require.NoError(t, <-done)
}

func TestRunStatusMenuClickInvokesStatusHandler(t *testing.T) {
	t.Parallel()

	driver := newFakeDriver(true)
	ctrl := newFakeController()
	statusCalled := make(chan struct{}, 1)
	r := newRunner(driver, func(app.Options, *log.Logger) controller {
		return ctrl
	})
	r.showStatus = func(app.Options) error {
		statusCalled <- struct{}{}
		return nil
	}

	done := make(chan error, 1)
	go func() {
		done <- r.Run(context.Background(), app.DefaultOptions(), log.Default())
	}()

	waitForDriverReady(t, driver)

	driver.menuItems[1].clicked <- struct{}{}

	select {
	case <-statusCalled:
	case <-time.After(time.Second):
		t.Fatal("status handler was not called")
	}
	assert.Equal(t, int64(0), driver.quitCalls.Load())

	driver.Quit()
	require.NoError(t, <-done)
}

func TestRunContextCancelStopsCollectorAndExitsTray(t *testing.T) {
	t.Parallel()

	driver := newFakeDriver(true)
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

	waitForDriverReady(t, driver)

	cancel()

	require.NoError(t, <-done)
	assert.Equal(t, int64(1), driver.quitCalls.Load())
	require.NotNil(t, ctrl.startCtx)
	assert.ErrorIs(t, ctrl.startCtx.Err(), context.Canceled)
}

func TestRunReturnsCollectorFailure(t *testing.T) {
	t.Parallel()

	driver := newFakeDriver(true)
	ctrl := newFakeController()
	expectedErr := errors.New("collector failed")
	ctrl.errCh <- expectedErr
	ctrl.waitErr = expectedErr
	r := newRunner(driver, func(app.Options, *log.Logger) controller {
		return ctrl
	})

	require.ErrorIs(t, r.Run(context.Background(), app.DefaultOptions(), log.Default()), expectedErr)
	assert.Equal(t, int64(1), driver.quitCalls.Load())
}
