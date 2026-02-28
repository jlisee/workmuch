package backend

import (
	"context"
	"errors"
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
	sample, err := completeSample(UsageSample{})
	return sample, errors.Join(err, fmt.Errorf("%w: %s", ErrNotImplemented, BackendMacOSNative))
}

func (b *MacOSNativeBackend) Reset() error {
	return fmt.Errorf("%w: %s", ErrNotImplemented, BackendMacOSNative)
}

func (b *MacOSNativeBackend) Close() error {
	return nil
}
