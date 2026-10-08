package converter

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"convertidor/internal/platform"
)

type Engine struct{ FFmpeg string }

func (e Engine) Convert(ctx context.Context, input, codec, format string, progress func(*float64)) (output string, resultErr error) {
	input, err := InputPath(input)
	if err != nil {
		return "", err
	}
	if !ValidTarget(codec, format) {
		return "", errors.New("La combinación de códec y formato no es compatible.")
	}
	temporary, err := os.MkdirTemp(filepath.Dir(input), ".convertidor-")
	if err != nil {
		return "", fmt.Errorf("No se pudo crear la carpeta de salida junto al video original.\nDetalles técnicos: %w", err)
	}
	defer func() {
		if err := os.RemoveAll(temporary); err != nil {
			resultErr = errors.Join(resultErr, fmt.Errorf("No se pudieron eliminar los archivos temporales de %s.\nDetalles técnicos: %w", temporary, err))
		}
	}()
	target := filepath.Join(temporary, "converted."+format)
	args, err := Arguments(input, target, codec, format)
	if err != nil {
		return "", err
	}
	tracker := &progressTracker{report: progress}
	command := exec.CommandContext(ctx, e.FFmpeg, args...)
	platform.PrepareCommand(command)
	command.Stdout = progressOutput{tracker}
	command.Stderr = tracker
	if err := command.Run(); err != nil {
		return "", fmt.Errorf("No se pudo convertir el video.\nDetalles técnicos: %w\n%s", err, tracker.diagnostic())
	}
	return publish(ctx, target, input, format)
}
