package main

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"workmuch-go/internal/app"
)

func TestSelectRunModeDefaultsToTray(t *testing.T) {
	t.Parallel()

	assert.Equal(t, runModeTray, selectRunMode(app.DefaultOptions()))
}

func TestSelectRunModeUsesConsoleForQA(t *testing.T) {
	t.Parallel()

	opts := app.DefaultOptions()
	opts.QAConsole = true

	assert.Equal(t, runModeConsole, selectRunMode(opts))
}
