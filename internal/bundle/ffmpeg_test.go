package bundle

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestMissingBundleIsAnErrorNotADevelopmentFallback(t *testing.T) {
	executable := filepath.Join(t.TempDir(), "Conversor de Video.exe")
	if err := os.WriteFile(executable, []byte("not a package"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := FFmpeg(executable); err == nil || !strings.Contains(err.Error(), "FFmpeg incluido") {
		t.Fatalf("missing bundle was not reported: %v", err)
	}
}

func TestMacBundleRequiresAnExecutableResource(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Mac resource-layout check")
	}
	root := t.TempDir()
	resources := filepath.Join(root, "Contents", "Resources")
	if err := os.MkdirAll(resources, 0700); err != nil {
		t.Fatal(err)
	}
	ffmpeg := filepath.Join(resources, "ffmpeg")
	if err := os.WriteFile(ffmpeg, []byte("ffmpeg"), 0600); err != nil {
		t.Fatal(err)
	}
	executable := filepath.Join(resources, "Conversor de Video")
	if _, _, err := FFmpeg(executable); err == nil {
		t.Fatal("non-executable FFmpeg was accepted")
	}
	if err := os.Chmod(ffmpeg, 0700); err != nil {
		t.Fatal(err)
	}
	path, cleanup, err := FFmpeg(executable)
	if err != nil || path != ffmpeg {
		t.Fatalf("wrong bundle resource: %s, %v", path, err)
	}
	if err := cleanup(); err != nil {
		t.Fatal(err)
	}
}
