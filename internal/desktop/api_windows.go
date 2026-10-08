package desktop

import (
	"strings"
	"syscall"

	"convertidor/internal/platform"
)

const (
	wmDestroy    = 0x0002
	wmSize       = 0x0005
	wmClose      = 0x0010
	wmCommand    = 0x0111
	wmTimer      = 0x0113
	wmDPIChanged = 0x02e0
	wmForceClose = 0x8001
	wmGetMinMax  = 0x0024
	idSelect     = 101
	idCodec      = 102
	idFormat     = 103
	idConvert    = 104
	cbGetCount   = 0x0146
	cbGetCurSel  = 0x0147
	cbGetText    = 0x0148
	cbGetTextLen = 0x0149
	cbReset      = 0x014b
	cbAdd        = 0x0143
	cbSetCurSel  = 0x014e
)

type winPoint struct{ X, Y int32 }
type winRect struct{ Left, Top, Right, Bottom int32 }
type winMessage struct {
	Window  uintptr
	Message uint32
	WParam  uintptr
	LParam  uintptr
	Time    uint32
	Point   winPoint
	Private uint32
}
type winClass struct {
	Size, Style             uint32
	Callback                uintptr
	ClassExtra, WindowExtra int32
	Instance, Icon, Cursor  uintptr
	Background              uintptr
	MenuName, ClassName     *uint16
	SmallIcon               uintptr
}
type minMaxInfo struct {
	Reserved, MaxSize, MaxPosition, MinTrack, MaxTrack winPoint
}

func (w *nativeWindow) call(name string, args ...uintptr) uintptr {
	value, _, _ := w.procs[name].Call(args...)
	return value
}

func loadWindowAPI() (map[string]*syscall.LazyProc, error) {
	libraries := []struct {
		name  string
		procs string
	}{
		{"user32.dll", "RegisterClassExW UnregisterClassW CreateWindowExW DefWindowProcW DestroyWindow ShowWindow UpdateWindow GetMessageW TranslateMessage DispatchMessageW IsDialogMessageW PostMessageW PostQuitMessage SendMessageW SetWindowTextW GetWindowTextW GetWindowTextLengthW EnableWindow IsWindowEnabled SetTimer KillTimer GetClientRect MoveWindow LoadCursorW MessageBoxExW SetWindowPos GetWindowLongPtrW SetWindowLongPtrW"},
		{"kernel32.dll", "GetModuleHandleW CreateActCtxW ActivateActCtx DeactivateActCtx ReleaseActCtx"},
		{"gdi32.dll", "CreateFontW DeleteObject"},
		{"comctl32.dll", "InitCommonControlsEx"},
	}
	procs := make(map[string]*syscall.LazyProc)
	for _, library := range libraries {
		dll, err := platform.WindowsSystemDLL(library.name)
		if err != nil {
			return nil, err
		}
		for _, name := range strings.Fields(library.procs) {
			proc := dll.NewProc(name)
			if err := proc.Find(); err != nil {
				return nil, err
			}
			procs[name] = proc
		}
		if library.name == "user32.dll" {
			for _, name := range []string{"SetThreadDpiAwarenessContext", "SetProcessDPIAware", "GetDpiForWindow"} {
				proc := dll.NewProc(name)
				if proc.Find() == nil {
					procs[name] = proc
				}
			}
		}
	}
	return procs, nil
}
