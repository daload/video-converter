package desktop

import (
	"context"
	"errors"
	"log"
	"path/filepath"
	"strings"
	"sync"

	"convertidor/internal/converter"
)

type Converter interface {
	Convert(context.Context, string, string, string, func(*float64)) (string, error)
}

type State struct {
	Options  []converter.Option `json:"options"`
	File     string             `json:"file"`
	Name     string             `json:"name"`
	Codec    string             `json:"codec"`
	Format   string             `json:"format"`
	Formats  []string           `json:"formats"`
	Busy     bool               `json:"busy"`
	Status   string             `json:"status"`
	Message  string             `json:"message"`
	Progress *float64           `json:"progress"`
	Output   string             `json:"output"`
}

type Controller struct {
	mu      sync.Mutex
	engine  Converter
	state   State
	context context.Context
	cancel  context.CancelFunc
	workers sync.WaitGroup
	closed  bool
}

func New(engine Converter) *Controller {
	ctx, cancel := context.WithCancel(context.Background())
	return &Controller{
		engine: engine, context: ctx, cancel: cancel,
		state: State{Codec: "h264", Format: "mp4", Status: "idle", Message: "Selecciona un video para empezar."},
	}
}

func (c *Controller) State() State {
	c.mu.Lock()
	defer c.mu.Unlock()
	state := c.state
	state.Options = converter.Options()
	for _, option := range state.Options {
		if option.ID == state.Codec {
			state.Formats = append([]string(nil), option.Formats...)
		}
	}
	if state.Progress != nil {
		progress := *state.Progress
		state.Progress = &progress
	}
	return state
}

func (c *Controller) editable() error {
	if c.closed {
		return errors.New("La aplicación se está cerrando.")
	}
	if c.state.Busy {
		return errors.New("Ya hay un video en proceso de conversión.")
	}
	return nil
}

func (c *Controller) reset() {
	c.state.Status, c.state.Output, c.state.Progress = "idle", "", nil
	c.state.Message = "Selecciona un video para empezar."
	if c.state.File != "" {
		c.state.Message = "Listo para convertir."
	}
}

func (c *Controller) Select(input string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if err := c.editable(); err != nil {
		return err
	}
	file, err := converter.InputPath(input)
	if err != nil {
		return err
	}
	c.state.File, c.state.Name = file, filepath.Base(file)
	original := strings.TrimPrefix(filepath.Ext(file), ".")
	c.state.Format = converter.DefaultFormat(c.state.Codec, original)
	c.reset()
	return nil
}

func (c *Controller) SetCodec(codec string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if err := c.editable(); err != nil {
		return err
	}
	for _, option := range converter.Options() {
		if option.ID == codec {
			if c.state.Codec != codec {
				c.state.Codec = codec
				original := strings.TrimPrefix(filepath.Ext(c.state.File), ".")
				c.state.Format = converter.DefaultFormat(codec, original)
				c.reset()
			}
			return nil
		}
	}
	return errors.New("El códec de video seleccionado no es compatible.")
}

func (c *Controller) SetFormat(format string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if err := c.editable(); err != nil {
		return err
	}
	if !converter.ValidTarget(c.state.Codec, format) {
		return errors.New("La combinación de códec y formato no es compatible.")
	}
	if c.state.Format != format {
		c.state.Format = format
		c.reset()
	}
	return nil
}

func (c *Controller) ReportError(err error) {
	if err == nil {
		return
	}
	log.Printf("ERROR: %v", err)
	c.mu.Lock()
	defer c.mu.Unlock()
	c.state.Message = err.Error()
	if !c.state.Busy {
		c.state.Status, c.state.Progress = "failed", nil
	}
}

func (c *Controller) Convert() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if err := c.editable(); err != nil {
		return err
	}
	if c.state.File == "" {
		return errors.New("Selecciona un video primero.")
	}
	input, codec, format := c.state.File, c.state.Codec, c.state.Format
	output, err := converter.AvailableOutput(input, format)
	if err != nil {
		return err
	}
	c.state.Busy, c.state.Status, c.state.Message = true, "running", "Convirtiendo..."
	c.state.Progress, c.state.Output = nil, output
	c.workers.Add(1)
	go func() {
		defer c.workers.Done()
		output, err := c.engine.Convert(c.context, input, codec, format, func(progress *float64) {
			c.mu.Lock()
			defer c.mu.Unlock()
			c.state.Progress = nil
			if progress != nil {
				value := *progress
				c.state.Progress = &value
			}
		})
		c.mu.Lock()
		defer c.mu.Unlock()
		c.state.Busy = false
		if err != nil {
			log.Printf("No se pudo convertir el video: %v", err)
			c.state.Status, c.state.Message, c.state.Progress = "failed", err.Error(), nil
			if output != "" {
				c.state.Output = output
				c.state.Message += "\nArchivo creado: " + output
			}
		} else {
			complete := 100.0
			c.state.Status, c.state.Message = "completed", "Guardado en: "+output
			c.state.Output, c.state.Progress = output, &complete
		}
	}()
	return nil
}

func (c *Controller) Close() {
	c.mu.Lock()
	c.closed = true
	c.cancel()
	c.mu.Unlock()
	c.workers.Wait()
}
