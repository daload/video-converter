package desktop

import (
	"errors"
	"fmt"
	"math"
	"runtime"
	"strings"
	"syscall"
	"unsafe"
)

func (w *nativeWindow) verifyChoices() error {
	state := w.controller.State()
	index := int(w.call("SendMessageW", w.codec, cbGetCurSel, 0, 0))
	if index < 0 || index >= len(w.codecs) || w.codecs[index] != state.Codec {
		return errors.New("Comprobación: el códec mostrado no coincide con el seleccionado.")
	}
	count := int(w.call("SendMessageW", w.format, cbGetCount, 0, 0))
	if count != len(state.Formats) {
		return errors.New("Comprobación: la cantidad de formatos mostrados es incorrecta.")
	}
	for index, format := range state.Formats {
		length := w.call("SendMessageW", w.format, cbGetTextLen, uintptr(index), 0)
		buffer := make([]uint16, length+1)
		w.call("SendMessageW", w.format, cbGetText, uintptr(index), uintptr(unsafe.Pointer(&buffer[0])))
		runtime.KeepAlive(buffer)
		if syscall.UTF16ToString(buffer) != format {
			return errors.New("Comprobación: el selector incluye un formato incompatible.")
		}
	}
	index = int(w.call("SendMessageW", w.format, cbGetCurSel, 0, 0))
	if index < 0 || index >= len(state.Formats) || state.Formats[index] != state.Format {
		return errors.New("Comprobación: el formato mostrado no coincide con el seleccionado.")
	}
	return nil
}

func (w *nativeWindow) smokeSelect(hwnd uintptr, id, index int) {
	w.call("SendMessageW", hwnd, cbSetCurSel, uintptr(index), 0)
	w.call("SendMessageW", w.hwnd, wmCommand, uintptr(id)|1<<16, hwnd)
}

func (w *nativeWindow) smokeFail(err error) {
	w.result = err
	w.controller.ReportError(err)
	w.render()
	w.call("PostMessageW", w.hwnd, wmForceClose, 0, 0)
}

func (w *nativeWindow) smoke(state State) {
	if !w.smokeStarted {
		if w.displayedText(w.hwnd) != "Conversor de Video" ||
			w.displayedText(w.selectBtn) != "&Seleccionar video..." ||
			w.displayedText(w.codecLabel) != "&Códec" ||
			w.displayedText(w.formatLabel) != "&Formato" ||
			w.displayedText(w.convert) != "Con&vertir" {
			w.smokeFail(errors.New("Los controles de la ventana no están en español."))
			return
		}
		if err := w.verifyChoices(); err != nil {
			w.smokeFail(err)
			return
		}
		if state.File == "" {
			if w.call("IsWindowEnabled", w.convert) != 0 {
				w.smokeFail(errors.New("Comprobación: Convertir debe estar desactivado si no hay un video."))
			} else {
				w.smokeFail(errors.New("Comprobación: se necesita un video de prueba seleccionado."))
			}
			return
		}
		if state.Busy {
			w.smokeFail(errors.New("Comprobación: no debe haber una conversión activa al empezar."))
			return
		}
		for index := range w.codecs {
			w.smokeSelect(w.codec, idCodec, index)
			if err := w.verifyChoices(); err != nil {
				w.smokeFail(err)
				return
			}
		}
		foundCodec, foundFormat := false, false
		for index, codec := range w.codecs {
			if codec == "h264" {
				w.smokeSelect(w.codec, idCodec, index)
				foundCodec = true
				break
			}
		}
		for index, format := range w.formats {
			if format == "mp4" {
				w.smokeSelect(w.format, idFormat, index)
				foundFormat = true
				break
			}
		}
		if !foundCodec || !foundFormat {
			w.smokeFail(errors.New("Comprobación: la combinación H.264/MP4 no está disponible."))
			return
		}
		w.smokeStarted = true
		w.call("SendMessageW", w.convert, 0x00f5, 0, 0) // BM_CLICK
		return
	}
	if state.Status == "failed" {
		w.smokeFail(fmt.Errorf("Comprobación: no se pudo convertir el video: %s", state.Message))
		return
	}
	if state.Busy {
		for _, hwnd := range []uintptr{w.selectBtn, w.codec, w.format, w.convert} {
			if w.call("IsWindowEnabled", hwnd) != 0 {
				w.smokeFail(errors.New("Comprobación: los controles siguen activos durante la conversión."))
				return
			}
		}
		if (state.Progress == nil) != w.marquee {
			w.smokeFail(errors.New("Comprobación: el progreso indeterminado se muestra de forma incorrecta."))
			return
		}
		if state.Progress != nil {
			expected := uintptr(int(math.Max(0, math.Min(100, *state.Progress))))
			if w.call("SendMessageW", w.progress, 0x408, 0, 0) != expected {
				w.smokeFail(errors.New("Comprobación: el progreso mostrado no coincide con el calculado."))
			}
		}
		return
	}
	if state.Status != "completed" {
		w.smokeFail(errors.New("Comprobación: el botón Convertir no inició la conversión."))
		return
	}
	expected := strings.ReplaceAll(strings.ReplaceAll(state.Message, "\r\n", "\n"), "\n", "\r\n")
	if state.Output == "" || !strings.Contains(state.Message, state.Output) ||
		w.displayedText(w.message) != expected || w.call("SendMessageW", w.progress, 0x408, 0, 0) != 100 || w.marquee {
		w.smokeFail(errors.New("Comprobación: el resultado o el progreso final no se muestran correctamente."))
		return
	}
	w.call("PostMessageW", w.hwnd, wmForceClose, 0, 0)
}
