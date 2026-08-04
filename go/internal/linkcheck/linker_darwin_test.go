//go:build darwin && cgo

package linkcheck

import (
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNativeBackendAndTrayLinkWithoutDuplicateLibraryWarning(t *testing.T) {
	outputPath := filepath.Join(t.TempDir(), "workmuch-linkcheck")
	cmd := exec.Command("go", "build", "-o", outputPath, "./testdata/dualobjc")
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, string(output))
	assert.NotContains(t, string(output), "ignoring duplicate libraries")
}
