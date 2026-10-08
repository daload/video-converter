package platform

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"runtime"
	"syscall"
	"unicode/utf16"
	"unsafe"
)

func systemDLL(name string) (*syscall.LazyDLL, error) {
	directory := make([]uint16, 32768)
	getDirectory := syscall.NewLazyDLL("kernel32.dll").NewProc("GetSystemDirectoryW")
	length, _, err := getDirectory.Call(uintptr(unsafe.Pointer(&directory[0])), uintptr(len(directory)))
	if length == 0 {
		return nil, fmt.Errorf("No se pudo localizar la carpeta del sistema de Windows.\nDetalles técnicos: %w", err)
	}
	if length >= uintptr(len(directory)) {
		return nil, fmt.Errorf("La ruta de la carpeta del sistema de Windows es demasiado larga.")
	}
	dll := syscall.NewLazyDLL(filepath.Join(syscall.UTF16ToString(directory), name))
	if err := dll.Load(); err != nil {
		return nil, fmt.Errorf("No se pudo cargar la biblioteca de Windows %s.\nDetalles técnicos: %w", name, err)
	}
	return dll, nil
}

// WindowsSystemDLL loads a named Windows library from the system directory.
// Native desktop code shares this loader rather than searching the working directory.
func WindowsSystemDLL(name string) (*syscall.LazyDLL, error) {
	return systemDLL(name)
}

type openFilename struct {
	structSize                   uint32
	owner, instance              uintptr
	filter, customFilter         *uint16
	maxCustomFilter, filterIndex uint32
	file                         *uint16
	maxFile                      uint32
	fileTitle                    *uint16
	maxFileTitle                 uint32
	initialDir, title            *uint16
	flags                        uint32
	fileOffset, fileExtension    uint16
	defaultExt                   *uint16
	customData, hook             uintptr
	templateName                 *uint16
	reserved                     unsafe.Pointer
	reservedWord, flagsEx        uint32
}

func ChooseVideo() (string, error) {
	return ChooseVideoForWindow(0)
}

// ChooseVideoForWindow keeps the native file dialog modal to its owner.
func ChooseVideoForWindow(owner uintptr) (string, error) {
	dll, err := systemDLL("comdlg32.dll")
	if err != nil {
		return "", err
	}
	buffer := make([]uint16, 32768)
	filter := utf16.Encode([]rune("Archivos de video (*.*)\x00*.*\x00\x00"))
	title, err := syscall.UTF16PtrFromString("Seleccionar un video")
	if err != nil {
		return "", err
	}
	dialog := openFilename{
		owner:  owner,
		filter: &filter[0], filterIndex: 1, file: &buffer[0],
		maxFile: uint32(len(buffer)), title: title,
		flags: 0x1000 | 0x800 | 0x8 | 0x80000,
	}
	dialog.structSize = uint32(unsafe.Sizeof(dialog))
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	result, _, _ := dll.NewProc("GetOpenFileNameW").Call(uintptr(unsafe.Pointer(&dialog)))
	runtime.KeepAlive(&dialog)
	runtime.KeepAlive(filter)
	runtime.KeepAlive(title)
	runtime.KeepAlive(buffer)
	if result == 0 {
		code, _, _ := dll.NewProc("CommDlgExtendedError").Call()
		if code == 0 {
			return "", nil
		}
		return "", fmt.Errorf("No se pudo abrir el selector de archivos (error de Windows 0x%X).", code)
	}
	return syscall.UTF16ToString(buffer), nil
}

func ShowError(message string) error {
	dll, err := systemDLL("user32.dll")
	if err != nil {
		return err
	}
	title, _ := syscall.UTF16PtrFromString("Conversor de Video")
	text, err := syscall.UTF16PtrFromString(message)
	if err != nil {
		return err
	}
	dll.NewProc("MessageBoxExW").Call(0, uintptr(unsafe.Pointer(text)), uintptr(unsafe.Pointer(title)), 0x10, 0x0c0a)
	runtime.KeepAlive(title)
	runtime.KeepAlive(text)
	return nil
}

func PrepareCommand(command *exec.Cmd) {
	command.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
}
