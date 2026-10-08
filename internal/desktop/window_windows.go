package desktop

import (
	"errors"
	"fmt"
	"runtime"
	"sync"
	"syscall"
	"unsafe"

	"convertidor/internal/platform"
)

// Only the HWND publication is shared. All controls and state rendering belong
// to the locked Run thread; RequestClose only posts a native message.
var windowsGUI struct {
	sync.Mutex
	window  *nativeWindow
	pending bool
}

var windowCallback = syscall.NewCallback(nativeWindowProc)

type nativeWindow struct {
	controller                                                 *Controller
	procs                                                      map[string]*syscall.LazyProc
	hwnd                                                       uintptr
	instance                                                   uintptr
	font                                                       uintptr
	dpi                                                        int
	className                                                  *uint16
	selectBtn, file, codec, format, convert, progress, message uintptr
	fileLabel, codecLabel, formatLabel, statusLabel            uintptr
	codecs, formats                                            []string
	optionKey, formatKey, fileText, messageText, statusText    string
	marquee                                                    bool
	testUI                                                     bool
	smokeStarted                                               bool
	result                                                     error
}

// RequestClose can be called from a signal goroutine, including during startup.
// It bypasses the interactive close confirmation; the caller owns cancellation.
func RequestClose() {
	windowsGUI.Lock()
	defer windowsGUI.Unlock()
	if w := windowsGUI.window; w != nil && w.hwnd != 0 {
		w.call("PostMessageW", w.hwnd, wmForceClose, 0, 0)
	} else {
		windowsGUI.pending = true
	}
}

func Run(controller *Controller, testUI bool) error {
	if controller == nil {
		return errors.New("La ventana necesita un controlador de conversión.")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	procs, err := loadWindowAPI()
	if err != nil {
		return fmt.Errorf("No se pudo iniciar la ventana.\nDetalles técnicos: %w", err)
	}
	w := &nativeWindow{controller: controller, procs: procs, dpi: 96, testUI: testUI}
	releaseStyles, err := w.enableVisualStyles()
	if err != nil {
		return fmt.Errorf("No se pudieron configurar los controles de Windows.\nDetalles técnicos: %w", err)
	}
	defer releaseStyles()
	windowsGUI.Lock()
	if windowsGUI.window != nil {
		windowsGUI.Unlock()
		return errors.New("Ya hay una ventana de la aplicación abierta.")
	}
	windowsGUI.window = w
	windowsGUI.Unlock()
	defer func() {
		// Prevent signal handlers from posting to a destroyed/reused handle.
		windowsGUI.Lock()
		hwnd := w.hwnd
		w.hwnd = 0
		windowsGUI.Unlock()
		if hwnd != 0 {
			w.call("KillTimer", hwnd, 1)
			w.call("DestroyWindow", hwnd)
		}
		windowsGUI.Lock()
		windowsGUI.window = nil
		windowsGUI.pending = false
		windowsGUI.Unlock()
		if w.font != 0 {
			w.call("DeleteObject", w.font)
		}
		if w.className != nil {
			w.call("UnregisterClassW", uintptr(unsafe.Pointer(w.className)), w.instance)
			runtime.KeepAlive(w.className)
		}
	}()
	var previousDPI uintptr
	if procs["SetThreadDpiAwarenessContext"] != nil {
		previousDPI = w.call("SetThreadDpiAwarenessContext", ^uintptr(3)) // per-monitor v2
		if previousDPI != 0 {
			defer w.call("SetThreadDpiAwarenessContext", previousDPI)
		}
	} else if procs["SetProcessDPIAware"] != nil {
		w.call("SetProcessDPIAware")
	}
	init := struct{ Size, Classes uint32 }{8, 0x20} // progress controls
	if w.call("InitCommonControlsEx", uintptr(unsafe.Pointer(&init))) == 0 {
		return errors.New("No se pudieron iniciar los controles de progreso.")
	}
	w.instance = w.call("GetModuleHandleW", 0)
	w.className, _ = syscall.UTF16PtrFromString("ConvertidorNativeWindow")
	class := winClass{
		Size: uint32(unsafe.Sizeof(winClass{})), Callback: windowCallback,
		Instance: w.instance, Cursor: w.call("LoadCursorW", 0, 32512),
		Background: 16, ClassName: w.className,
	}
	if w.call("RegisterClassExW", uintptr(unsafe.Pointer(&class))) == 0 {
		return errors.New("No se pudo registrar la ventana de la aplicación.")
	}
	runtime.KeepAlive(class)
	title, _ := syscall.UTF16PtrFromString("Conversor de Video")
	hwnd := w.call("CreateWindowExW", 0, uintptr(unsafe.Pointer(w.className)),
		uintptr(unsafe.Pointer(title)), 0x02cf0000, 0x80000000, 0x80000000, 760, 560,
		0, 0, w.instance, 0)
	runtime.KeepAlive(title)
	runtime.KeepAlive(w.className)
	if hwnd == 0 {
		return errors.New("No se pudo crear la ventana de la aplicación.")
	}
	windowsGUI.Lock()
	w.hwnd = hwnd
	pending := windowsGUI.pending
	windowsGUI.pending = false
	windowsGUI.Unlock()
	if procs["GetDpiForWindow"] != nil {
		w.dpi = int(w.call("GetDpiForWindow", hwnd))
	}
	w.call("SetWindowPos", hwnd, 0, 0, 0, uintptr(w.scale(760)), uintptr(w.scale(560)), 0x0016)
	if err := w.createControls(); err != nil {
		return err
	}
	w.render()
	w.layout()
	w.call("ShowWindow", hwnd, 5)
	w.call("UpdateWindow", hwnd)
	if w.call("SetTimer", hwnd, 1, 200, 0) == 0 {
		return errors.New("No se pudo iniciar la actualización de la ventana.")
	}
	if pending {
		w.call("PostMessageW", hwnd, wmForceClose, 0, 0)
	}
	var message winMessage
	for {
		value := int32(w.call("GetMessageW", uintptr(unsafe.Pointer(&message)), 0, 0, 0))
		if value == -1 {
			return errors.New("La ventana dejó de responder.")
		}
		if value == 0 {
			break
		}
		if w.call("IsDialogMessageW", hwnd, uintptr(unsafe.Pointer(&message))) == 0 {
			w.call("TranslateMessage", uintptr(unsafe.Pointer(&message)))
			w.call("DispatchMessageW", uintptr(unsafe.Pointer(&message)))
		}
	}
	return w.result
}

func nativeWindowProc(hwnd uintptr, message uint32, wp uintptr, lp unsafe.Pointer) uintptr {
	windowsGUI.Lock()
	w := windowsGUI.window
	windowsGUI.Unlock()
	if w == nil {
		return 0
	}
	switch message {
	case wmClose:
		if w.controller.State().Busy {
			text, _ := syscall.UTF16PtrFromString("¿Detener la conversión y cerrar?\nSe eliminará el archivo incompleto. El video original no cambiará.")
			title, _ := syscall.UTF16PtrFromString("Conversor de Video")
			answer := w.call("MessageBoxExW", hwnd, uintptr(unsafe.Pointer(text)),
				uintptr(unsafe.Pointer(title)), 0x00000124, 0x0c0a) // Yes/No, No default; Spanish
			runtime.KeepAlive(text)
			runtime.KeepAlive(title)
			if answer != 6 {
				return 0
			}
		}
		w.call("DestroyWindow", hwnd)
		return 0
	case wmForceClose:
		w.call("DestroyWindow", hwnd)
		return 0
	case wmDestroy:
		w.call("KillTimer", hwnd, 1)
		windowsGUI.Lock()
		w.hwnd = 0
		windowsGUI.Unlock()
		w.call("PostQuitMessage", 0)
		return 0
	case wmCommand:
		if wp&0xffff == 1 && w.call("IsWindowEnabled", w.convert) != 0 {
			w.action(idConvert, 0) // Enter / dialog IDOK
			return 0
		}
		w.action(int(wp&0xffff), int((wp>>16)&0xffff))
		return 0
	case 0x0400: // DM_GETDEFID: native dialog keyboard navigation
		return 0x534b<<16 | idConvert
	case wmTimer:
		if wp == 1 {
			state := w.render()
			if w.testUI {
				w.smoke(state)
			}
		}
		return 0
	case wmSize:
		if w.file != 0 {
			w.layout()
		}
		return 0
	case wmGetMinMax:
		info := (*minMaxInfo)(lp)
		info.MinTrack = winPoint{int32(w.scale(540)), int32(w.scale(440))}
		return 0
	case wmDPIChanged:
		w.dpi = int(wp & 0xffff)
		rect := (*winRect)(lp)
		w.call("SetWindowPos", hwnd, 0, uintptr(rect.Left), uintptr(rect.Top),
			uintptr(rect.Right-rect.Left), uintptr(rect.Bottom-rect.Top), 0x0014)
		if w.file != 0 {
			w.setFont()
			w.layout()
		}
		return 0
	}
	return w.call("DefWindowProcW", hwnd, uintptr(message), wp, uintptr(lp))
}

func (w *nativeWindow) action(id, notification int) {
	var err error
	switch {
	case id == idSelect && notification == 0:
		var path string
		path, err = platform.ChooseVideoForWindow(w.hwnd)
		if err == nil && path != "" {
			err = w.controller.Select(path)
		}
	case id == idCodec && notification == 1:
		index := int(w.call("SendMessageW", w.codec, cbGetCurSel, 0, 0))
		if index >= 0 && index < len(w.codecs) {
			err = w.controller.SetCodec(w.codecs[index])
		}
	case id == idFormat && notification == 1:
		index := int(w.call("SendMessageW", w.format, cbGetCurSel, 0, 0))
		if index >= 0 && index < len(w.formats) {
			err = w.controller.SetFormat(w.formats[index])
		}
	case id == idConvert && notification == 0:
		err = w.controller.Convert()
	}
	if err != nil {
		w.controller.ReportError(err)
	}
	w.render()
}
