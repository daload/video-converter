package desktop

import (
	"errors"
	"os"
	"runtime"
	"syscall"
	"unsafe"
)

// The v6 Windows common controls provide the native marquee progress bar.
// An activation context avoids depending on an externally installed manifest
// or changing the parent's executable packaging. Its temporary source is
// deleted immediately after Windows has loaded it.
func (w *nativeWindow) enableVisualStyles() (func(), error) {
	file, err := os.CreateTemp("", "convertidor-native-*.manifest")
	if err != nil {
		return nil, err
	}
	defer os.Remove(file.Name())
	const manifest = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<assembly xmlns="urn:schemas-microsoft-com:asm.v1" manifestVersion="1.0">
 <assemblyIdentity version="1.0.0.0" processorArchitecture="amd64" name="Convertidor.Native" type="win32"/>
 <dependency><dependentAssembly>
  <assemblyIdentity type="win32" name="Microsoft.Windows.Common-Controls" version="6.0.0.0" processorArchitecture="amd64" publicKeyToken="6595b64144ccf1df" language="*"/>
 </dependentAssembly></dependency>
</assembly>`
	_, writeErr := file.WriteString(manifest)
	closeErr := file.Close()
	if writeErr != nil {
		return nil, writeErr
	}
	if closeErr != nil {
		return nil, closeErr
	}
	path, err := syscall.UTF16PtrFromString(file.Name())
	if err != nil {
		return nil, err
	}
	context := struct {
		Size, Flags             uint32
		Source                  *uint16
		Architecture, Language  uint16
		Directory, ResourceName *uint16
		ApplicationName         *uint16
		Module                  uintptr
	}{Source: path}
	context.Size = uint32(unsafe.Sizeof(context))
	handle := w.call("CreateActCtxW", uintptr(unsafe.Pointer(&context)))
	runtime.KeepAlive(context)
	runtime.KeepAlive(path)
	if handle == ^uintptr(0) {
		return nil, errors.New("No se pudieron cargar los estilos de los controles de Windows.")
	}
	var cookie uintptr
	if w.call("ActivateActCtx", handle, uintptr(unsafe.Pointer(&cookie))) == 0 {
		w.call("ReleaseActCtx", handle)
		return nil, errors.New("No se pudieron activar los estilos de los controles de Windows.")
	}
	return func() {
		w.call("DeactivateActCtx", 0, cookie)
		w.call("ReleaseActCtx", handle)
	}, nil
}
