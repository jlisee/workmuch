package backend

import (
	"context"
	"fmt"
)

type MacOSNativeBackend struct{}

func NewMacOSNativeBackend() (Backend, error) {
	return nil, fmt.Errorf("%w: %s", ErrNotImplemented, BackendMacOSNative)
}

func (b *MacOSNativeBackend) Name() string {
	return BackendMacOSNative
}

func (b *MacOSNativeBackend) Sample(_ context.Context) (UsageSample, error) {
	return UsageSample{}, fmt.Errorf("%w: %s", ErrNotImplemented, BackendMacOSNative)
}

func (b *MacOSNativeBackend) Reset() error {
	return fmt.Errorf("%w: %s", ErrNotImplemented, BackendMacOSNative)
}

func (b *MacOSNativeBackend) Close() error {
	return nil
}
