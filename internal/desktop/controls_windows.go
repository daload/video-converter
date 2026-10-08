package desktop

import (
	"fmt"
	"math"
	"runtime"
	"strings"
	"syscall"
	"unsafe"
)

func (w *nativeWindow) scale(value int) int { return (value*w.dpi + 48) / 96 }

func (w *nativeWindow) control(class, text string, style uintptr, id int) (uintptr, error) {
	classString, _ := syscall.UTF16PtrFromString(class)
	textString, _ := syscall.UTF16PtrFromString(text)
	hwnd := w.call("CreateWindowExW", 0, uintptr(unsafe.Pointer(classString)),
		uintptr(unsafe.Pointer(textString)), 0x50000000|style, 0, 0, 0, 0,
		w.hwnd, uintptr(id), w.instance, 0)
	runtime.KeepAlive(classString)
	runtime.KeepAlive(textString)
	if hwnd == 0 {
		return 0, fmt.Errorf("No se pudo crear el control de la ventana: %s.", class)
	}
	return hwnd, nil
}

func (w *nativeWindow) createControls() error {
	controls := []struct {
		target      *uintptr
		class, text string
		style       uintptr
		id          int
	}{
		{&w.fileLabel, "STATIC", "Video seleccionado:", 0, 0},
		{&w.selectBtn, "BUTTON", "&Seleccionar video...", 0x10000, idSelect},
		{&w.file, "EDIT", "", 0x00b108c4, 0}, // read-only multiline, scrollable, tab stop
		{&w.codecLabel, "STATIC", "&Códec", 0, 0},
		{&w.codec, "COMBOBOX", "", 0x00210003, idCodec}, // dropdown list
		{&w.formatLabel, "STATIC", "&Formato", 0, 0},
		{&w.format, "COMBOBOX", "", 0x00210003, idFormat},
		{&w.convert, "BUTTON", "Con&vertir", 0x10001, idConvert}, // default push button
		{&w.statusLabel, "STATIC", "Listo", 0, 0},
		{&w.progress, "msctls_progress32", "", 0, 0},
		{&w.message, "EDIT", "", 0x00b108c4, 0},
	}
	for _, control := range controls {
		hwnd, err := w.control(control.class, control.text, control.style, control.id)
		if err != nil {
			return err
		}
		*control.target = hwnd
	}
	w.setFont()
	w.call("SendMessageW", w.progress, 0x406, 0, 100) // PBM_SETRANGE32
	return nil
}

func (w *nativeWindow) setFont() {
	face, _ := syscall.UTF16PtrFromString("Segoe UI")
	height := -w.scale(16)
	font := w.call("CreateFontW", uintptr(height), 0, 0, 0, 400, 0, 0, 0,
		1, 0, 0, 5, 0, uintptr(unsafe.Pointer(face)))
	runtime.KeepAlive(face)
	if font == 0 {
		return
	}
	old := w.font
	w.font = font
	for _, hwnd := range []uintptr{w.fileLabel, w.selectBtn, w.file, w.codecLabel, w.codec,
		w.formatLabel, w.format, w.convert, w.statusLabel, w.message} {
		w.call("SendMessageW", hwnd, 0x0030, font, 1) // WM_SETFONT
	}
	if old != 0 {
		w.call("DeleteObject", old)
	}
}

func (w *nativeWindow) layout() {
	var rect winRect
	w.call("GetClientRect", w.hwnd, uintptr(unsafe.Pointer(&rect)))
	width, height := int(rect.Right), int(rect.Bottom)
	margin := w.scale(20)
	content := width - 2*margin
	move := func(hwnd uintptr, x, y, width, height int) {
		w.call("MoveWindow", hwnd, uintptr(x), uintptr(y), uintptr(width), uintptr(height), 1)
	}
	move(w.fileLabel, margin, margin, content-w.scale(180), w.scale(24))
	move(w.selectBtn, width-margin-w.scale(170), margin, w.scale(170), w.scale(30))
	move(w.file, margin, w.scale(58), content, w.scale(62))
	column := (content - w.scale(20)) / 2
	move(w.codecLabel, margin, w.scale(138), column, w.scale(24))
	move(w.codec, margin, w.scale(165), column, w.scale(300))
	move(w.formatLabel, margin+column+w.scale(20), w.scale(138), column, w.scale(24))
	move(w.format, margin+column+w.scale(20), w.scale(165), column, w.scale(300))
	move(w.convert, margin, w.scale(212), w.scale(150), w.scale(34))
	move(w.statusLabel, margin+w.scale(165), w.scale(216), content-w.scale(165), w.scale(28))
	move(w.progress, margin, w.scale(264), content, w.scale(22))
	move(w.message, margin, w.scale(304), content, height-w.scale(304)-margin)
}

func (w *nativeWindow) text(hwnd uintptr, text string) {
	buffer, _ := syscall.UTF16PtrFromString(strings.ReplaceAll(text, "\x00", "\uFFFD"))
	w.call("SetWindowTextW", hwnd, uintptr(unsafe.Pointer(buffer)))
	runtime.KeepAlive(buffer)
}

func (w *nativeWindow) comboItems(hwnd uintptr, items []string) {
	w.call("SendMessageW", hwnd, cbReset, 0, 0)
	for _, item := range items {
		buffer, _ := syscall.UTF16PtrFromString(item)
		w.call("SendMessageW", hwnd, cbAdd, 0, uintptr(unsafe.Pointer(buffer)))
		runtime.KeepAlive(buffer)
	}
}

func (w *nativeWindow) render() State {
	state := w.controller.State()
	var ids, labels []string
	for _, option := range state.Options {
		ids = append(ids, option.ID)
		labels = append(labels, option.Label)
	}
	optionKey := strings.Join(ids, "\x00") + "|" + strings.Join(labels, "\x00")
	if optionKey != w.optionKey {
		w.comboItems(w.codec, labels)
		w.codecs, w.optionKey = ids, optionKey
	}
	formatKey := strings.Join(state.Formats, "\x00")
	if formatKey != w.formatKey {
		w.comboItems(w.format, state.Formats)
		w.formats, w.formatKey = append([]string(nil), state.Formats...), formatKey
	}
	for index, codec := range w.codecs {
		if codec == state.Codec {
			w.call("SendMessageW", w.codec, cbSetCurSel, uintptr(index), 0)
		}
	}
	for index, format := range w.formats {
		if format == state.Format {
			w.call("SendMessageW", w.format, cbSetCurSel, uintptr(index), 0)
		}
	}
	fileText := "No se ha seleccionado un video."
	if state.File != "" {
		fileText = state.Name + "\r\n" + state.File
	}
	if fileText != w.fileText {
		w.text(w.file, fileText)
		w.fileText = fileText
	}
	message := strings.ReplaceAll(strings.ReplaceAll(state.Message, "\r\n", "\n"), "\n", "\r\n")
	if message != w.messageText {
		w.text(w.message, message)
		w.messageText = message
	}
	enabled := uintptr(1)
	if state.Busy {
		enabled = 0
	}
	for _, hwnd := range []uintptr{w.selectBtn, w.codec, w.format} {
		w.call("EnableWindow", hwnd, enabled)
	}
	if state.File == "" {
		enabled = 0
	}
	w.call("EnableWindow", w.convert, enabled)
	marquee := state.Busy && state.Progress == nil
	if marquee != w.marquee {
		style := w.call("GetWindowLongPtrW", w.progress, ^uintptr(15)) // GWL_STYLE = -16
		if marquee {
			style |= 8
		} else {
			w.call("SendMessageW", w.progress, 0x40a, 0, 0)
			style &^= 8
		}
		w.call("SetWindowLongPtrW", w.progress, ^uintptr(15), style)
		w.call("SetWindowPos", w.progress, 0, 0, 0, 0, 0, 0x0037)
		if marquee {
			w.call("SendMessageW", w.progress, 0x40a, 1, 30)
		}
		w.marquee = marquee
	}
	status := "Listo"
	switch state.Status {
	case "running":
		status = "Convirtiendo"
	case "completed":
		status = "Completado"
	case "failed":
		status = "Error"
	}
	percent := 0.0
	if state.Progress != nil && !math.IsNaN(*state.Progress) {
		percent = math.Max(0, math.Min(100, *state.Progress))
		status += fmt.Sprintf(" — %.0f%%", percent)
	}
	if !marquee {
		w.call("SendMessageW", w.progress, 0x402, uintptr(int(percent)), 0)
	}
	if status != w.statusText {
		w.text(w.statusLabel, status)
		w.statusText = status
	}
	return state
}

func (w *nativeWindow) displayedText(hwnd uintptr) string {
	length := w.call("GetWindowTextLengthW", hwnd)
	buffer := make([]uint16, length+1)
	w.call("GetWindowTextW", hwnd, uintptr(unsafe.Pointer(&buffer[0])), uintptr(len(buffer)))
	runtime.KeepAlive(buffer)
	return syscall.UTF16ToString(buffer)
}
