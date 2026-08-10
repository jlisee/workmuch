//go:build darwin && !cgo

package backend

import "fmt"

func IsMacOSAccessibilityTrusted() (bool, error) {
	return false, fmt.Errorf("%w: macOS accessibility check requires cgo on darwin", ErrNotImplemented)
}

func PromptForMacOSAccessibility() error {
	return fmt.Errorf("%w: macOS accessibility prompt requires cgo on darwin", ErrNotImplemented)
}
