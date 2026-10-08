package converter

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
)

func InputPath(input string) (string, error) {
	path, err := filepath.Abs(input)
	if err != nil {
		return "", fmt.Errorf("No se pudo leer la ruta del video.\nDetalles técnicos: %w", err)
	}
	info, err := os.Stat(path)
	if err != nil {
		return "", fmt.Errorf("No se pudo abrir el video seleccionado.\nDetalles técnicos: %w", err)
	}
	if !info.Mode().IsRegular() {
		return "", errors.New("Selecciona un archivo de video, no una carpeta ni un dispositivo.")
	}
	return path, nil
}

func outputName(input, format string, number int) string {
	name := strings.TrimSuffix(filepath.Base(input), filepath.Ext(input))
	suffix := ""
	if number > 0 {
		suffix = fmt.Sprintf(" %d", number)
	}
	return filepath.Join(filepath.Dir(input), "Compatible - "+name+suffix+"."+format)
}

func AvailableOutput(input, format string) (string, error) {
	for number := 0; ; number++ {
		path := outputName(input, format, number)
		if _, err := os.Lstat(path); errors.Is(err, os.ErrNotExist) {
			return path, nil
		} else if err != nil {
			return "", fmt.Errorf("No se pudo comprobar el archivo de salida.\nDetalles técnicos: %w", err)
		}
	}
}

type contextReader struct {
	context context.Context
	reader  io.Reader
}

func (r contextReader) Read(data []byte) (int, error) {
	if err := r.context.Err(); err != nil {
		return 0, err
	}
	return r.reader.Read(data)
}

func exclusiveCopy(ctx context.Context, source, destination string) (resultErr error) {
	input, err := os.Open(source)
	if err != nil {
		return err
	}
	defer input.Close()
	output, err := os.OpenFile(destination, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(output, contextReader{ctx, input})
	resultErr = errors.Join(copyErr, output.Close())
	if resultErr != nil {
		resultErr = errors.Join(resultErr, os.Remove(destination))
	}
	return resultErr
}

func publish(ctx context.Context, temporary, input, format string) (string, error) {
	for number := 0; ; number++ {
		if err := ctx.Err(); err != nil {
			return "", err
		}
		output := outputName(input, format, number)
		err := os.Link(temporary, output)
		if err == nil {
			return output, nil
		}
		if errors.Is(err, os.ErrExist) {
			continue
		}
		// FAT/exFAT cannot publish with a hard link; exclusive creation still prevents overwrites.
		log.Printf("No se pudo crear un enlace físico; se usará una copia sin sobrescribir archivos: %v", err)
		copyErr := exclusiveCopy(ctx, temporary, output)
		if errors.Is(copyErr, os.ErrExist) {
			continue
		}
		if copyErr != nil {
			return "", fmt.Errorf("No se pudo guardar el video convertido.\nDetalles técnicos: %w", errors.Join(err, copyErr))
		}
		return output, nil
	}
}
