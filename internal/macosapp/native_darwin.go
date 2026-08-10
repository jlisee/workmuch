//go:build darwin && cgo

package macosapp

/*
#cgo LDFLAGS: -framework AppKit -framework Foundation -framework ServiceManagement
#include "native_darwin.h"
*/
import "C"

import (
	"errors"
	"fmt"
)

type mainAppService struct{}

func NewMainAppService() LoginItem {
	return mainAppService{}
}

func (mainAppService) Status() (LoginItemState, error) {
	result := C.wmMainAppServiceStatus()
	if result.ok == 0 {
		return LoginItemUnsupported, nativeError("read main app login item status", result.err)
	}
	_ = consumeCString(result.err)
	return mapLoginItemStatus(int(result.value)), nil
}

func (mainAppService) Register() error {
	result := C.wmRegisterMainAppService()
	if result.ok == 0 {
		return nativeError("register main app login item", result.err)
	}
	_ = consumeCString(result.err)
	return nil
}

type moveDialog struct{}

func NewMoveDialog() MoveDialog {
	return moveDialog{}
}

func (moveDialog) Show() error {
	result := C.wmShowMoveToApplicationsDialog()
	if result.ok == 0 {
		return nativeError("show move dialog", result.err)
	}
	_ = consumeCString(result.err)
	return nil
}

func consumeCString(value *C.char) string {
	if value == nil {
		return ""
	}
	defer C.wmMacOSAppFreeString(value)
	return C.GoString(value)
}

func nativeError(prefix string, value *C.char) error {
	message := consumeCString(value)
	if message == "" {
		return errors.New(prefix)
	}
	return fmt.Errorf("%s: %s", prefix, message)
}
