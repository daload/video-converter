package desktop

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

type convertFunc func(context.Context, string, string, string, func(*float64)) (string, error)

func (f convertFunc) Convert(ctx context.Context, input, codec, format string, progress func(*float64)) (string, error) {
	return f(ctx, input, codec, format, progress)
}

func videoFile(t *testing.T, name string) string {
	t.Helper()
	file := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(file, []byte("original video"), 0600); err != nil {
		t.Fatal(err)
	}
	return file
}

func TestOriginalFormatDefaultsFollowTheSelectedCodec(t *testing.T) {
	c := New(nil)
	defer c.Close()
	if state := c.State(); state.Codec != "h264" || state.Format != "mp4" || state.File != "" {
		t.Fatalf("unexpected defaults: %+v", state)
	}
	for _, test := range []struct{ name, want string }{
		{"Holiday.MOV", "mov"}, {"Holiday.mkv", "mkv"}, {"Holiday.webm", "mp4"}, {"Holiday.avi", "mp4"},
	} {
		if err := c.Select(videoFile(t, test.name)); err != nil {
			t.Fatal(err)
		}
		if state := c.State(); state.Format != test.want {
			t.Fatalf("%s: got %s, want %s", test.name, state.Format, test.want)
		}
	}
	input := videoFile(t, "Holiday.webm")
	if err := c.Select(input); err != nil {
		t.Fatal(err)
	}
	if err := c.SetCodec("vp9"); err != nil {
		t.Fatal(err)
	}
	if c.State().Format != "webm" {
		t.Fatal("VP9 must retain an original WebM format")
	}
	if err := c.SetFormat("mkv"); err != nil {
		t.Fatal(err)
	}
	if c.State().Format != "mkv" {
		t.Fatal("a manually selected valid format must persist")
	}
	if err := c.SetCodec("h265"); err != nil {
		t.Fatal(err)
	}
	if c.State().Format != "mp4" {
		t.Fatal("H.265 must fall back to MP4 for a WebM source")
	}
}

func TestInvalidActionsPreserveTheSelectionAndChoices(t *testing.T) {
	c := New(nil)
	defer c.Close()
	if err := c.Convert(); err == nil || c.State().Busy {
		t.Fatal("conversion without a video must fail without starting")
	}
	input := videoFile(t, "Holiday.mov")
	if err := c.Select(input); err != nil {
		t.Fatal(err)
	}
	for _, action := range []func() error{
		func() error { return c.SetCodec("unknown") },
		func() error { return c.SetFormat("webm") },
		func() error { return c.Select(filepath.Dir(input)) },
		func() error { return c.Select(input + ".missing") },
	} {
		if err := action(); err == nil {
			t.Fatal("invalid action was accepted")
		}
		state := c.State()
		if state.File != input || state.Codec != "h264" || state.Format != "mov" {
			t.Fatalf("invalid action changed the selection: %+v", state)
		}
	}
	c.ReportError(errors.New("cannot read this video"))
	if state := c.State(); state.Status != "failed" || state.Message != "cannot read this video" || state.File != input {
		t.Fatalf("the error must remain visible without losing the selection: %+v", state)
	}
}

func waitUntilIdle(t *testing.T, c *Controller) State {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if state := c.State(); !state.Busy {
			return state
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("conversion did not finish")
	return State{}
}

func TestRunningConversionLocksChoicesAndReportsItsFinalOutput(t *testing.T) {
	input := videoFile(t, "Holiday.mov")
	output := filepath.Join(filepath.Dir(input), "Compatible - Holiday 2.mp4")
	started, release := make(chan struct{}), make(chan struct{})
	c := New(convertFunc(func(ctx context.Context, gotInput, codec, format string, report func(*float64)) (string, error) {
		if gotInput != input || codec != "h264" || format != "mp4" {
			return "", errors.New("the engine received different choices")
		}
		progress := 25.0
		report(&progress)
		close(started)
		select {
		case <-release:
			return output, nil
		case <-ctx.Done():
			return "", ctx.Err()
		}
	}))
	defer c.Close()
	if err := c.Select(input); err != nil {
		t.Fatal(err)
	}
	if err := c.SetFormat("mp4"); err != nil {
		t.Fatal(err)
	}
	if err := c.Convert(); err != nil {
		t.Fatal(err)
	}
	select {
	case <-started:
	case <-time.After(3 * time.Second):
		t.Fatal("the worker did not start")
	}
	for _, action := range []func() error{
		c.Convert,
		func() error { return c.Select(input) },
		func() error { return c.SetCodec("vp9") },
		func() error { return c.SetFormat("mkv") },
	} {
		if err := action(); err == nil {
			t.Fatal("editing during conversion must fail")
		}
	}
	state := c.State()
	if state.Progress == nil || *state.Progress != 25 {
		t.Fatalf("measured progress was not reported: %+v", state)
	}
	*state.Progress = 90
	state.Formats[0] = "bad"
	state.Options[0].Formats[0] = "bad"
	if state := c.State(); *state.Progress != 25 || state.Formats[0] != "mp4" || state.Options[0].Formats[0] != "mp4" {
		t.Fatal("snapshots must not expose mutable controller state")
	}
	close(release)
	state = waitUntilIdle(t, c)
	if state.Status != "completed" || state.Output != output || state.Progress == nil || *state.Progress != 100 ||
		state.Message != "Guardado en: "+output {
		t.Fatalf("unexpected completed state: %+v", state)
	}
}

func TestClosingCancelsTheWorkerAndRejectsLaterActions(t *testing.T) {
	started := make(chan struct{})
	c := New(convertFunc(func(ctx context.Context, _, _, _ string, report func(*float64)) (string, error) {
		close(started)
		<-ctx.Done()
		return "", ctx.Err()
	}))
	if err := c.Select(videoFile(t, "Holiday.mp4")); err != nil {
		t.Fatal(err)
	}
	if err := c.Convert(); err != nil {
		t.Fatal(err)
	}
	select {
	case <-started:
	case <-time.After(3 * time.Second):
		t.Fatal("the worker did not start")
	}
	c.Close()
	c.Close()
	if state := c.State(); state.Busy || state.Status != "failed" || state.Progress != nil {
		t.Fatalf("cancelled conversion must stop: %+v", state)
	}
	for _, action := range []func() error{
		c.Convert,
		func() error { return c.SetCodec("vp9") },
		func() error { return c.SetFormat("mkv") },
		func() error { return c.Select("missing.mp4") },
	} {
		if err := action(); err == nil {
			t.Fatal("a closed controller must reject further actions")
		}
	}
}

func TestConversionErrorsRemainVisibleAndAllowARetry(t *testing.T) {
	c := New(convertFunc(func(context.Context, string, string, string, func(*float64)) (string, error) {
		return "", errors.New("this file has no video stream")
	}))
	defer c.Close()
	if err := c.Select(videoFile(t, "Holiday.mp4")); err != nil {
		t.Fatal(err)
	}

	for range 2 {
		if err := c.Convert(); err != nil {
			t.Fatal(err)
		}
		state := waitUntilIdle(t, c)
		if state.Status != "failed" || state.Message != "this file has no video stream" || state.Progress != nil {
			t.Fatalf("unexpected failed state: %+v", state)
		}
	}
}

func TestPublishedOutputIsReportedWhenTemporaryCleanupFails(t *testing.T) {
	input := videoFile(t, "Holiday.mp4")
	output := filepath.Join(filepath.Dir(input), "Compatible - Holiday 2.mp4")
	c := New(convertFunc(func(context.Context, string, string, string, func(*float64)) (string, error) {
		return output, errors.New("cannot remove temporary files")
	}))
	defer c.Close()
	if err := c.Select(input); err != nil {
		t.Fatal(err)
	}
	if err := c.Convert(); err != nil {
		t.Fatal(err)
	}
	state := waitUntilIdle(t, c)
	if state.Status != "failed" || state.Output != output ||
		state.Message != "cannot remove temporary files\nArchivo creado: "+output {
		t.Fatalf("the window must report the error and the actual published output: %+v", state)
	}
}
