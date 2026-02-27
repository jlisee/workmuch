package backend

import (
	"context"
	"fmt"
)

type LinuxBackend struct{}

func NewLinuxBackend() (Backend, error) {
	return nil, fmt.Errorf("%w: %s", ErrNotImplemented, BackendLinux)
}

func (b *LinuxBackend) Name() string {
	return BackendLinux
}

func (b *LinuxBackend) Sample(_ context.Context) (UsageSample, error) {
	return UsageSample{}, fmt.Errorf("%w: %s", ErrNotImplemented, BackendLinux)
}

func (b *LinuxBackend) Reset() error {
	return fmt.Errorf("%w: %s", ErrNotImplemented, BackendLinux)
}

func (b *LinuxBackend) Close() error {
	return nil
}
