//go:build darwin && cgo

package backend

/*
#cgo LDFLAGS: -framework ApplicationServices -framework CoreGraphics
#include "macos_native_bridge_darwin.h"
*/
import "C"

import (
	"errors"
	"fmt"
)

type nativeMacOSAPI struct{}

func newMacOSNativeAPI() (macOSNativeAPI, error) {
	return nativeMacOSAPI{}, nil
}

func (nativeMacOSAPI) IsAccessibilityTrusted() bool {
	return C.wmAXIsProcessTrusted() != 0
}

func (nativeMacOSAPI) GetFrontmostWindowInfo() (int, string, string, error) {
	result := C.wmGetFrontmostWindowInfo()
	appName := consumeBridgeCString(result.app_name)
	windowTitle := consumeBridgeCString(result.window_title)
	if result.ok == 0 {
		return 0, "", "", nativeBridgeError("frontmost window query failed", result.err)
	}
	_ = consumeBridgeCString(result.err)
	return int(result.pid), appName, windowTitle, nil
}

func (nativeMacOSAPI) GetFrontmostApplication() (int, string, error) {
	result := C.wmGetFrontmostApplication()
	appName := consumeBridgeCString(result.app_name)
	if result.ok == 0 {
		return 0, "", nativeBridgeError("frontmost application query failed", result.err)
	}
	_ = consumeBridgeCString(result.err)
	return int(result.pid), appName, nil
}

func (nativeMacOSAPI) GetFocusedWindowTitle(pid int) (string, error) {
	result := C.wmGetFocusedWindowTitle(C.int(pid))
	title := consumeBridgeCString(result.value)
	if result.ok == 0 {
		return "", nativeBridgeError("focused window title query failed", result.err)
	}
	_ = consumeBridgeCString(result.err)
	return title, nil
}

func (nativeMacOSAPI) GetIdleSeconds() (float64, error) {
	result := C.wmGetIdleSeconds()
	if result.ok == 0 {
		return 0, nativeBridgeError("idle seconds query failed", result.err)
	}
	_ = consumeBridgeCString(result.err)
	return float64(result.value), nil
}

func (nativeMacOSAPI) Close() error {
	return nil
}

func consumeBridgeCString(value *C.char) string {
	if value == nil {
		return ""
	}
	defer C.wmFreeString(value)
	return C.GoString(value)
}

func nativeBridgeError(prefix string, errMessage *C.char) error {
	message := consumeBridgeCString(errMessage)
	if message == "" {
		return errors.New(prefix)
	}
	return fmt.Errorf("%s: %s", prefix, message)
}
