package versioncalc

import (
	"os/exec"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestVersionsAreValidAndOrderedByDebian(t *testing.T) {
	dpkg, err := exec.LookPath("dpkg")
	if err != nil {
		t.Skip("dpkg is not installed")
	}

	versions := []string{
		"20260804.100.8+g111111111111",
		"20260805.42.9+g222222222222",
		"20260805.43.0+g333333333333",
		"20260805.43.5+g444444444444",
	}
	for _, version := range versions {
		cmd := exec.Command(dpkg, "--validate-version", version)
		require.NoError(t, cmd.Run(), "version should be Debian-compatible: %s", version)
	}
	for index := 1; index < len(versions); index++ {
		cmd := exec.Command(dpkg, "--compare-versions", versions[index-1], "lt", versions[index])
		require.NoError(t, cmd.Run(), "%s should sort before %s", versions[index-1], versions[index])
	}
}
