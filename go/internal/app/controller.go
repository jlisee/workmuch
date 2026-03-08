package app

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"sync"
)

type RunnerFunc func(ctx context.Context, opts Options, logger *log.Logger) error

// Controller owns one collector run and exposes start/wait lifecycle helpers.
// It is safe to call Start multiple times; only the first call launches work.
type Controller struct {
	opts   Options
	logger *log.Logger
	run    RunnerFunc

	// startOnce guarantees we launch at most one collector goroutine.
	startOnce sync.Once
	// done is closed when the collector exits.
	done chan struct{}
	// errCh publishes the collector result exactly once.
	errCh chan error

	// result is stored under mu so Wait can return a stable terminal value.
	mu     sync.Mutex
	result error
}

// RunCollector is the production collector runner used by default app flows.
func RunCollector(ctx context.Context, opts Options, logger *log.Logger) error {
	return runCollector(ctx, opts, logger)
}

// NewController builds a lifecycle controller around the provided runner.
func NewController(opts Options, logger *log.Logger, run RunnerFunc) *Controller {
	if logger == nil {
		logger = log.New(os.Stderr, "", log.LstdFlags)
	}
	if run == nil {
		panic(fmt.Errorf("runner must not be nil"))
	}

	return &Controller{
		opts:   opts,
		logger: logger,
		run:    run,
	}
}

// Start launches the collector once and returns a channel that receives the
// terminal run error (or nil on clean shutdown).
func (c *Controller) Start(ctx context.Context) <-chan error {
	c.startOnce.Do(func() {
		c.done = make(chan struct{})
		c.errCh = make(chan error, 1)

		go func() {
			err := c.run(ctx, c.opts, c.logger)
			// Cancellation is an expected shutdown path, not a fatal condition.
			if errors.Is(err, context.Canceled) {
				err = nil
			}

			c.mu.Lock()
			c.result = err
			c.mu.Unlock()

			c.errCh <- err
			close(c.errCh)
			close(c.done)
		}()
	})

	if c.errCh == nil {
		ch := make(chan error, 1)
		ch <- nil
		close(ch)
		return ch
	}

	return c.errCh
}

// Wait blocks until the collector finishes and returns its terminal result.
// If Start was never called, Wait returns nil immediately.
func (c *Controller) Wait() error {
	if c.done == nil {
		return nil
	}

	<-c.done

	c.mu.Lock()
	defer c.mu.Unlock()
	return c.result
}
