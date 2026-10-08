package converter

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func TestFormatDefaultsAndInvalidCombinations(t *testing.T) {
	for _, test := range []struct{ codec, original, expected string }{
		{"h264", "MOV", "mov"}, {"h265", "mkv", "mkv"}, {"h264", "avi", "mp4"},
		{"h264", "webm", "mp4"}, {"vp9", "webm", "webm"}, {"vp9", "mov", "mp4"},
	} {
		if result := DefaultFormat(test.codec, test.original); result != test.expected {
			t.Fatalf("%+v: got %s", test, result)
		}
	}
	for _, codec := range []string{"h264", "h265", "vp9"} {
		if _, err := Arguments("input", "output", codec, "exe"); err == nil {
			t.Fatal("invalid format accepted")
		}
	}
	if _, err := Arguments("input", "output", "h264", "webm"); err == nil {
		t.Fatal("H.264 cannot be muxed into WebM")
	}
}

func TestPublishingConcurrentResultsNeverOverwritesExistingFiles(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "Holiday.mov")
	temporary := filepath.Join(dir, "result")
	if err := os.WriteFile(temporary, []byte("converted video"), 0600); err != nil {
		t.Fatal(err)
	}
	existing := outputName(input, "mp4", 0)
	if err := os.WriteFile(existing, []byte("keep this"), 0600); err != nil {
		t.Fatal(err)
	}
	var workers sync.WaitGroup
	outputs := make(chan string, 12)
	for index := 0; index < 12; index++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			path, err := publish(context.Background(), temporary, input, "mp4")
			if err != nil {
				t.Error(err)
				return
			}
			outputs <- path
		}()
	}
	workers.Wait()
	close(outputs)
	unique := map[string]bool{}
	for output := range outputs {
		unique[output] = true
	}
	if len(unique) != 12 {
		t.Fatalf("only %d unique results", len(unique))
	}
	for number := 1; number <= 12; number++ {
		if !unique[outputName(input, "mp4", number)] {
			t.Fatalf("missing numbered result %d", number)
		}
	}
	content, err := os.ReadFile(existing)
	if err != nil || string(content) != "keep this" {
		t.Fatal("existing file was changed")
	}
	if err := exclusiveCopy(context.Background(), temporary, existing); !os.IsExist(err) {
		t.Fatal("exclusive copy accepted an existing output")
	}
	copyPath := filepath.Join(dir, "copied.mp4")
	if err := exclusiveCopy(context.Background(), temporary, copyPath); err != nil {
		t.Fatal(err)
	}
	copied, err := os.ReadFile(copyPath)
	if err != nil || string(copied) != "converted video" {
		t.Fatal("exclusive copy differs")
	}
}

func TestProgressReportsOnlyMeasuredPercentagesAndBoundsLogs(t *testing.T) {
	var percentages []*float64
	tracker := &progressTracker{report: func(value *float64) { percentages = append(percentages, value) }}
	writer := progressOutput{tracker}
	if _, err := writer.Write([]byte("out_time_us=1\n")); err != nil {
		t.Fatal(err)
	}
	if percentages[0] != nil {
		t.Fatal("unknown duration produced a fake percentage")
	}
	if _, err := tracker.Write([]byte("Duration: 00:00:10.00\n")); err != nil {
		t.Fatal(err)
	}
	for _, part := range []string{"out_time_", "us=5000000\nout_time_us=20000000\n"} {
		if _, err := writer.Write([]byte(part)); err != nil {
			t.Fatal(err)
		}
	}
	if *percentages[1] != 50 || *percentages[2] != 99 {
		t.Fatal("incorrect measured progress")
	}
	if _, err := tracker.Write(bytes.Repeat([]byte("x"), 20000)); err != nil {
		t.Fatal(err)
	}
	if len(tracker.diagnostic()) != 8192 {
		t.Fatal("diagnostic log is not bounded")
	}
}

func nativeFFmpeg(t *testing.T) string {
	t.Helper()
	path := os.Getenv("FFMPEG_TEST_BINARY")
	if path == "" {
		t.Skip("set FFMPEG_TEST_BINARY to run real codec conversion checks")
	}
	return path
}

func TestAllSupportedCodecsAndContainersProduceCorrectStreams(t *testing.T) {
	ffmpeg := nativeFFmpeg(t)
	ffprobe, err := exec.LookPath("ffprobe")
	if err != nil {
		t.Fatal("ffprobe is required for conversion checks")
	}
	dir := t.TempDir()
	input := filepath.Join(dir, "Holiday's video !.mov")
	generate := exec.Command(ffmpeg, "-v", "error", "-f", "lavfi", "-i", "testsrc2=size=64x64:rate=10", "-f", "lavfi", "-i", "sine=frequency=440:sample_rate=44100", "-t", "0.3", "-c:v", "libx265", "-x265-params", "pools=1:frame-threads=1:log-level=error", "-c:a", "aac", input)
	if log, err := generate.CombinedOutput(); err != nil {
		t.Fatalf("%v\n%s", err, log)
	}
	original, err := os.ReadFile(input)
	if err != nil {
		t.Fatal(err)
	}
	for _, option := range Options() {
		for _, format := range option.Formats {
			t.Run(option.ID+"-"+format, func(t *testing.T) {
				output, err := (Engine{FFmpeg: ffmpeg}).Convert(context.Background(), input, option.ID, format, nil)
				if err != nil {
					t.Fatal(err)
				}
				result, err := exec.Command(ffprobe, "-v", "error", "-show_streams", "-show_format", "-of", "json", output).Output()
				if err != nil {
					t.Fatal(err)
				}
				var probe struct {
					Streams []struct {
						CodecName   string `json:"codec_name"`
						CodecType   string `json:"codec_type"`
						PixelFormat string `json:"pix_fmt"`
					} `json:"streams"`
					Format struct {
						Name string            `json:"format_name"`
						Tags map[string]string `json:"tags"`
					} `json:"format"`
				}
				if err := json.Unmarshal(result, &probe); err != nil {
					t.Fatal(err)
				}
				expectedVideo := map[string]string{"h264": "h264", "h265": "hevc", "vp9": "vp9"}[option.ID]
				expectedAudio := "aac"
				if format == "webm" || (option.ID == "vp9" && format == "mkv") {
					expectedAudio = "opus"
				}
				if len(probe.Streams) != 2 || probe.Streams[0].CodecName != expectedVideo || probe.Streams[0].PixelFormat != "yuv420p" || probe.Streams[1].CodecName != expectedAudio {
					t.Fatalf("wrong stream shape: %s", result)
				}
				if format == "mov" && strings.TrimSpace(probe.Format.Tags["major_brand"]) != "qt" {
					t.Fatal("MOV was not encoded as QuickTime")
				}
				if format == "mp4" && probe.Format.Tags["major_brand"] == "qt  " {
					t.Fatal("MP4 was encoded as MOV")
				}
				if (format == "mkv" || format == "webm") && !strings.Contains(probe.Format.Name, "matroska") {
					t.Fatal("wrong container")
				}
				current, err := os.ReadFile(input)
				if err != nil || sha256.Sum256(current) != sha256.Sum256(original) {
					t.Fatal("original file changed")
				}
			})
		}
	}
	temporary, err := filepath.Glob(filepath.Join(dir, ".convertidor-*"))
	if err != nil || len(temporary) != 0 {
		t.Fatal("temporary conversion files remain")
	}
}

func TestFailedAndCancelledConversionsLeaveNoOutput(t *testing.T) {
	ffmpeg := nativeFFmpeg(t)
	dir := t.TempDir()
	input := filepath.Join(dir, "broken.mp4")
	if err := os.WriteFile(input, []byte("not a video"), 0600); err != nil {
		t.Fatal(err)
	}

	for _, cancelled := range []bool{false, true} {
		ctx, cancel := context.WithCancel(context.Background())
		if cancelled {
			cancel()
		}
		if _, err := (Engine{ffmpeg}).Convert(ctx, input, "h264", "mp4", nil); err == nil {
			t.Fatal("conversion incorrectly succeeded")
		}
		cancel()
	}
	files, err := os.ReadDir(dir)
	if err != nil || len(files) != 1 {
		t.Fatal("failed conversion left output or temporary files")
	}
}
