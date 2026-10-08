package desktop

/*
#cgo CFLAGS: -fobjc-arc -mmacosx-version-min=12.0
#cgo LDFLAGS: -framework Cocoa
#include <stdlib.h>
#include "window_darwin.h"
*/
import "C"

import (
	"encoding/json"
	"errors"
	"fmt"
	"runtime"
	"unsafe"
)

var nativeController *Controller
var nativeError error

func init() {
	// AppKit must run on the process's original thread.
	runtime.LockOSThread()
}

func Run(controller *Controller, testUI bool) error {
	nativeController, nativeError = controller, nil
	defer func() { nativeController = nil }()
	test := C.int(0)
	if testUI {
		test = 1
	}
	result := C.cv_run(test)
	if result != nil {
		defer C.free(unsafe.Pointer(result))
		return errors.New(C.GoString(result))
	}
	return nativeError
}

func RequestClose() { C.cv_stop() }

//export cv_action
func cv_action(action C.int, value *C.char) {
	if nativeController == nil {
		return
	}
	var err error
	switch action {
	case 0:
		err = nativeController.Select(C.GoString(value))
	case 1:
		err = nativeController.SetCodec(C.GoString(value))
	case 2:
		err = nativeController.SetFormat(C.GoString(value))
	case 3:
		err = nativeController.Convert()
	case 4:
		err = errors.New("Selecciona un solo video a la vez.")
	default:
		err = fmt.Errorf("Acción de la ventana desconocida: %d", action)
	}
	nativeController.ReportError(err)
}

//export cv_tick
func cv_tick() {
	if nativeController == nil {
		return
	}
	data, err := json.Marshal(nativeController.State())
	if err != nil {
		nativeError = fmt.Errorf("No se pudo actualizar la ventana.\nDetalles técnicos: %w", err)
		nativeController.ReportError(nativeError)
		C.cv_stop()
		return
	}
	state := C.CString(string(data))
	defer C.free(unsafe.Pointer(state))
	C.cv_render(state)
}
