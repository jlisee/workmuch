//go:build darwin && cgo

package backend

/*
#include "macos_native_bridge_darwin.h"
*/
import "C"

func IsMacOSAccessibilityTrusted() (bool, error) {
	api, err := newMacOSNativeAPI()
	if err != nil {
		return false, err
	}
	defer api.Close()
	return api.IsAccessibilityTrusted(), nil
}

func PromptForMacOSAccessibility() error {
	C.wmAXIsProcessTrustedWithPrompt()
	return nil
}
