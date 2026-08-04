package app

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFormatLogMessageIncludesContext(t *testing.T) {
	t.Parallel()

	message := formatLogMessage(
		"warning",
		"backend sample",
		errors.New("window unavailable"),
		logField{key: "backend", value: "macos-native"},
		logField{key: "path", value: "/Users/test/.workmuch/status.json"},
	)

	assert.Equal(t, `warning: backend sample backend="macos-native" path="/Users/test/.workmuch/status.json": window unavailable`, message)
}
