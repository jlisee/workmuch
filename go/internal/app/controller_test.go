package app

import (
	"context"
	"errors"
	"log"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestControllerStartRunsCollector(t *testing.T) {
	t.Parallel()

	started := make(chan struct{}, 1)
	controller := NewController(DefaultOptions(), log.Default(), func(ctx context.Context, _ Options, _ *log.Logger) error {
		started <- struct{}{}
		<-ctx.Done()
		return ctx.Err()
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errCh := controller.Start(ctx)

	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("collector did not start")
	}

	cancel()
	require.NoError(t, <-errCh)
	require.NoError(t, controller.Wait())
}

func TestControllerWaitReturnsCollectorError(t *testing.T) {
	t.Parallel()

	expectedErr := errors.New("boom")
	controller := NewController(DefaultOptions(), log.Default(), func(context.Context, Options, *log.Logger) error {
		return expectedErr
	})

	errCh := controller.Start(context.Background())

	require.ErrorIs(t, <-errCh, expectedErr)
	require.ErrorIs(t, controller.Wait(), expectedErr)
}

func TestRunTreatsCanceledContextAsCleanShutdown(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	require.NoError(t, Run(ctx, DefaultOptions(), log.Default()))
}

func TestControllerStartOnlyLaunchesCollectorOnce(t *testing.T) {
	t.Parallel()

	started := 0
	controller := NewController(DefaultOptions(), log.Default(), func(ctx context.Context, _ Options, _ *log.Logger) error {
		started++
		<-ctx.Done()
		return ctx.Err()
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	first := controller.Start(ctx)
	second := controller.Start(ctx)
	assert.True(t, first == second)

	cancel()
	require.NoError(t, controller.Wait())
	assert.Equal(t, 1, started)
}

func TestNewControllerPanicsWhenRunnerIsNil(t *testing.T) {
	t.Parallel()

	require.Panics(t, func() {
		_ = NewController(DefaultOptions(), log.Default(), nil)
	})
}
