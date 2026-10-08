package desktop

import (
	"strings"
	"testing"
)

func TestSpanishMessagesPreserveTechnicalCodecAndFormatIDs(t *testing.T) {
	c := New(nil)
	defer c.Close()
	state := c.State()
	if state.Message != "Selecciona un video para empezar." || state.Options[0].Label != "H.264 - más compatible" ||
		state.Options[1].Label != "H.265 - compresión eficiente" || state.Options[2].Label != "VP9 - para video WebM" {
		t.Fatalf("unexpected Spanish text: %+v", state)
	}
	if state.Codec != "h264" || state.Format != "mp4" {
		t.Fatalf("translation changed technical IDs: %+v", state)
	}
	if err := c.Select(videoFile(t, "Vacaciones.mp4")); err != nil {
		t.Fatal(err)
	}
	if state := c.State(); state.Message != "Listo para convertir." {
		t.Fatalf("unexpected selected state: %+v", state)
	}
	if err := c.SetFormat("webm"); err == nil || !strings.Contains(err.Error(), "códec y formato") {
		t.Fatalf("invalid-target error is not in Spanish: %v", err)
	}
}
