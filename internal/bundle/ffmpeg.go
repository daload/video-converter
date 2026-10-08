package bundle

import (
	"archive/zip"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
)

func FFmpeg(executable string) (path string, cleanup func() error, resultErr error) {
	if runtime.GOOS != "windows" {
		path = filepath.Join(filepath.Dir(executable), "..", "Resources", "ffmpeg")
		info, err := os.Stat(path)
		if err != nil {
			return "", nil, fmt.Errorf("No se pudo abrir FFmpeg incluido.\nDetalles técnicos: %w", err)
		}
		if !info.Mode().IsRegular() || info.Mode().Perm()&0111 == 0 {
			return "", nil, errors.New("FFmpeg incluido no es un archivo ejecutable.")
		}
		return path, func() error { return nil }, nil
	}
	archive, err := zip.OpenReader(executable)
	if err != nil {
		return "", nil, fmt.Errorf("No se pudo abrir el paquete de FFmpeg incluido.\nDetalles técnicos: %w", err)
	}
	defer archive.Close()
	source, err := archive.Open("ffmpeg.exe")
	if err != nil {
		return "", nil, fmt.Errorf("No se pudo abrir FFmpeg incluido.\nDetalles técnicos: %w", err)
	}
	defer source.Close()
	temp, err := os.MkdirTemp("", "ConversorDeVideo-")
	if err != nil {
		return "", nil, fmt.Errorf("No se pudo crear la carpeta temporal de FFmpeg.\nDetalles técnicos: %w", err)
	}
	cleanup = func() error {
		if err := os.RemoveAll(temp); err != nil {
			return fmt.Errorf("No se pudo eliminar la carpeta temporal de FFmpeg.\nDetalles técnicos: %w", err)
		}
		return nil
	}
	path = filepath.Join(temp, "ffmpeg.exe")
	target, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0700)
	if err != nil {
		return "", nil, errors.Join(fmt.Errorf("No se pudo crear el archivo temporal de FFmpeg.\nDetalles técnicos: %w", err), cleanup())
	}
	_, copyErr := io.Copy(target, source)
	if err := errors.Join(copyErr, target.Close()); err != nil {
		return "", nil, errors.Join(fmt.Errorf("No se pudo extraer FFmpeg.\nDetalles técnicos: %w", err), cleanup())
	}
	return path, cleanup, nil
}
