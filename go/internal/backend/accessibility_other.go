//go:build !darwin

package backend

import "fmt"

func IsMacOSAccessibilityTrusted() (bool, error) {
	return false, fmt.Errorf("%w: macOS accessibility check requires darwin", ErrNotImplemented)
}
