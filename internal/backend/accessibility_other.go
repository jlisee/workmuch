//go:build !darwin

package backend

import "fmt"

func IsMacOSAccessibilityTrusted() (bool, error) {
	return false, fmt.Errorf("%w: macOS accessibility check requires darwin", ErrNotImplemented)
}

func PromptForMacOSAccessibility() error {
	return fmt.Errorf("%w: macOS accessibility prompt requires darwin", ErrNotImplemented)
}
