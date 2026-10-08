package converter

import (
	"bytes"
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"
	"sync"
)

var durationPattern = regexp.MustCompile(`Duration:\s+(\d{1,6}:\d{2}:\d{2}(?:\.\d{1,9})?)`)

type progressTracker struct {
	mu       sync.Mutex
	duration float64
	log      []byte
	pending  string
	report   func(*float64)
}

func (p *progressTracker) Write(data []byte) (int, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.log = append(p.log, data...)
	if p.duration == 0 {
		if match := durationPattern.FindSubmatch(p.log); match != nil {
			var hours, minutes int
			var seconds float64
			if _, err := fmt.Sscanf(string(match[1]), "%d:%d:%f", &hours, &minutes, &seconds); err != nil {
				log.Printf("No se pudo leer la duración del video para calcular el progreso: %v", err)
			} else {
				p.duration = float64(hours*3600+minutes*60) + seconds
			}
		}
	}
	if len(p.log) > 8192 {
		p.log = bytes.Clone(p.log[len(p.log)-8192:])
	}
	return len(data), nil
}

func (p *progressTracker) diagnostic() string {
	p.mu.Lock()
	defer p.mu.Unlock()
	return string(p.log)
}

type progressOutput struct{ tracker *progressTracker }

func (w progressOutput) Write(data []byte) (int, error) {
	p := w.tracker
	p.pending += string(data)
	if len(p.pending) > 65536 {
		return 0, fmt.Errorf("La información de progreso de FFmpeg es demasiado larga.")
	}
	for {
		line, rest, found := strings.Cut(p.pending, "\n")
		if !found {
			break
		}
		p.pending = rest
		value, ok := strings.CutPrefix(strings.TrimSpace(line), "out_time_us=")
		if !ok || value == "N/A" {
			continue
		}
		time, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return 0, fmt.Errorf("No se pudo leer el progreso de FFmpeg.\nDetalles técnicos: %w", err)
		}
		p.mu.Lock()
		duration := p.duration
		p.mu.Unlock()
		var percentage *float64
		if duration > 0 {
			completed := min(99.0, max(0.0, time/1e6/duration*100))
			percentage = &completed
		}
		if p.report != nil {
			p.report(percentage)
		}
	}
	return len(data), nil
}
