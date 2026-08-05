package main

import (
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuiltBinaryVersionOutput(t *testing.T) {
	testCases := []struct {
		name    string
		ldflags string
		want    string
	}{
		{name: "unstamped", want: "workmuch dev\n"},
		{
			name:    "stamped",
			ldflags: "-X workmuch-go/internal/buildinfo.Version=20260805.43.5+g69d002bbb362",
			want:    "workmuch 20260805.43.5+g69d002bbb362\n",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			binary := filepath.Join(t.TempDir(), "workmuch")
			args := []string{"build", "-o", binary}
			if testCase.ldflags != "" {
				args = append(args, "-ldflags", testCase.ldflags)
			}
			args = append(args, "./cmd/workmuch-go")

			build := exec.Command("go", args...)
			build.Dir = filepath.Join("..", "..")
			output, err := build.CombinedOutput()
			require.NoError(t, err, "go build failed:\n%s", output)

			command := exec.Command(binary, "--version")
			output, err = command.CombinedOutput()
			require.NoError(t, err)
			assert.Equal(t, testCase.want, string(output))
		})
	}
}
