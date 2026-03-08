package tray

import (
	"context"
	_ "embed"
	"log"
	"os"
	"sync"

	systray "github.com/tailscale/systray"

	"workmuch-go/internal/app"
)

//go:embed assets/icon32x32.png
var trayIcon []byte

type menuItem interface {
	ClickedCh() <-chan struct{}
	Disable()
}

type driver interface {
	Run(onReady func(), onExit func())
	Quit()
	SetIcon(icon []byte)
	SetTitle(title string)
	SetTooltip(tooltip string)
	AddMenuItem(title string, tooltip string) menuItem
}

type controller interface {
	Start(ctx context.Context) <-chan error
	Wait() error
}

type controllerFactory func(opts app.Options, logger *log.Logger) controller

type runner struct {
	driver        driver
	newController controllerFactory
}

func Run(opts app.Options, logger *log.Logger) error {
	return RunWithContext(context.Background(), opts, logger)
}

func RunWithContext(ctx context.Context, opts app.Options, logger *log.Logger) error {
	return newRunner(systrayDriver{}, func(opts app.Options, logger *log.Logger) controller {
		return app.NewController(opts, logger, app.RunCollector)
	}).Run(ctx, opts, logger)
}

func newRunner(driver driver, newController controllerFactory) runner {
	return runner{
		driver:        driver,
		newController: newController,
	}
}

func (r runner) Run(ctx context.Context, opts app.Options, logger *log.Logger) error {
	if logger == nil {
		logger = log.New(os.Stderr, "", log.LstdFlags)
	}

	collector := r.newController(opts, logger)
	collectorCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	resultCh := make(chan error, 1)
	var resultOnce sync.Once
	var quitOnce sync.Once
	finish := func(err error) {
		resultOnce.Do(func() {
			resultCh <- err
		})
	}
	quitTray := func() {
		quitOnce.Do(func() {
			r.driver.Quit()
		})
	}

	r.driver.Run(func() {
		r.driver.SetIcon(trayIcon)
		r.driver.SetTitle("")
		r.driver.SetTooltip("Logging active")

		about := r.driver.AddMenuItem("About", "Logging active")
		about.Disable()

		r.driver.AddMenuItem("Quit", "Quit")
		errCh := collector.Start(collectorCtx)

		go func() {
			if err := <-errCh; err != nil {
				logger.Printf("tray collector failed: %v", err)
				finish(err)
				quitTray()
			}
		}()

		go func() {
			<-ctx.Done()
			cancel()
			finish(collector.Wait())
			quitTray()
		}()
	}, func() {
		cancel()
		finish(collector.Wait())
	})

	return <-resultCh
}

type systrayDriver struct{}

func (systrayDriver) Run(onReady func(), onExit func()) {
	systray.Run(onReady, onExit)
}

func (systrayDriver) Quit() {
	systray.Quit()
}

func (systrayDriver) SetIcon(icon []byte) {
	systray.SetIcon(icon)
}

func (systrayDriver) SetTitle(title string) {
	systray.SetTitle(title)
}

func (systrayDriver) SetTooltip(tooltip string) {
	systray.SetTooltip(tooltip)
}

func (systrayDriver) AddMenuItem(title string, tooltip string) menuItem {
	return systrayMenuItem{item: systray.AddMenuItem(title, tooltip)}
}

type systrayMenuItem struct {
	item *systray.MenuItem
}

func (m systrayMenuItem) ClickedCh() <-chan struct{} {
	return m.item.ClickedCh
}

func (m systrayMenuItem) Disable() {
	m.item.Disable()
}
