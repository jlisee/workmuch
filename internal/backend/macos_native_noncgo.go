//go:build darwin && !cgo

package backend

import "fmt"

func NewMacOSNativeBackend() (Backend, error) {
	return nil, fmt.Errorf("%w: %s requires cgo on darwin", ErrNotImplemented, BackendMacOSNative)
}
