//go:build darwin && cgo

package backend

func IsMacOSAccessibilityTrusted() (bool, error) {
	api, err := newMacOSNativeAPI()
	if err != nil {
		return false, err
	}
	defer api.Close()
	return api.IsAccessibilityTrusted(), nil
}
