package converter

import (
	"context"
	"encoding/json"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestSilentOddSizedVideoIsPaddedWithoutAddingAudio(t *testing.T) {
	ffmpeg := nativeFFmpeg(t)
	ffprobe, err := exec.LookPath("ffprobe")
	if err != nil {
		t.Fatal(err)
	}
	input := filepath.Join(t.TempDir(), "odd.mkv")
	command := exec.Command(ffmpeg, "-v", "error", "-f", "lavfi", "-i", "testsrc=size=65x63:rate=10",
		"-t", "0.2", "-c:v", "ffv1", "-pix_fmt", "bgr0", input)
	if log, err := command.CombinedOutput(); err != nil {
		t.Fatalf("%v\n%s", err, log)
	}
	output, err := (Engine{ffmpeg}).Convert(context.Background(), input, "h264", "mp4", nil)
	if err != nil {
		t.Fatal(err)
	}
	data, err := exec.Command(ffprobe, "-v", "error", "-show_entries", "stream=codec_type,width,height", "-of", "json", output).Output()
	if err != nil {
		t.Fatal(err)
	}
	var result struct {
		Streams []struct {
			Type   string `json:"codec_type"`
			Width  int    `json:"width"`
			Height int    `json:"height"`
		} `json:"streams"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatal(err)
	}
	if len(result.Streams) != 1 || result.Streams[0].Type != "video" || result.Streams[0].Width != 66 || result.Streams[0].Height != 64 {
		t.Fatalf("incorrect silent-video shape: %s", data)
	}
}
