//go:build darwin && cgo

package backend

func NewMacOSNativeBackend() (Backend, error) {
	api, err := newMacOSNativeAPI()
	if err != nil {
		return nil, err
	}
	return newMacOSNativeBackend(api), nil
}
