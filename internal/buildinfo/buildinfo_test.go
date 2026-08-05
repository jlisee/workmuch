package buildinfo

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestVersionDefaultsToDev(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "dev", Version)
}
