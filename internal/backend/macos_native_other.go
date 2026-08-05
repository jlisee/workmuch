//go:build !darwin

package backend

import "fmt"

func NewMacOSNativeBackend() (Backend, error) {
	return nil, fmt.Errorf("%w: %s requires darwin", ErrNotImplemented, BackendMacOSNative)
}
