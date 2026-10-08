package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"convertidor/internal/bundle"
	"convertidor/internal/converter"
	"convertidor/internal/desktop"
	"convertidor/internal/platform"
)

var testUI = flag.Bool("test-ui", false, "Comprobar los controles, convertir un video de prueba y salir")

func run() (resultErr error) {
	ffmpeg := flag.String("ffmpeg", "", "Ejecutable de FFmpeg para ejecutar el código fuente")
	flag.Parse()
	if flag.NArg() > 1 {
		return fmt.Errorf("Selecciona un solo video a la vez.")
	}
	if *ffmpeg == "" {
		executable, err := os.Executable()
		if err != nil {
			return fmt.Errorf("No se pudo localizar la aplicación.\nDetalles técnicos: %w", err)
		}
		path, cleanup, err := bundle.FFmpeg(executable)
		if err != nil {
			return err
		}
		*ffmpeg = path
		defer func() { resultErr = errors.Join(resultErr, cleanup()) }()
	} else if info, err := os.Stat(*ffmpeg); err != nil || !info.Mode().IsRegular() {
		return fmt.Errorf("La ruta de FFmpeg debe indicar un archivo ejecutable existente.")
	}
	application := desktop.New(converter.Engine{FFmpeg: *ffmpeg})
	defer application.Close()
	if flag.NArg() == 1 {
		if err := application.Select(flag.Arg(0)); err != nil {
			return err
		}
	}
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(signals)
	finished := make(chan struct{})
	defer close(finished)
	go func() {
		select {
		case <-signals:
			desktop.RequestClose()
		case <-finished:
		}
	}()
	if err := desktop.Run(application, *testUI); err != nil {
		return err
	}
	if *testUI {
		return json.NewEncoder(os.Stdout).Encode(application.State())
	}
	return nil
}

func main() {
	if err := run(); err != nil {
		log.Printf("ERROR: %v", err)
		if !*testUI {
			if notificationErr := platform.ShowError(err.Error()); notificationErr != nil {
				log.Printf("No se pudo mostrar el error: %v", notificationErr)
			}
		}
		os.Exit(1)
	}
}
