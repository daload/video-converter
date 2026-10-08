package converter

import (
	"fmt"
	"strings"
)

type Option struct {
	ID      string   `json:"id"`
	Label   string   `json:"label"`
	Help    string   `json:"help"`
	Formats []string `json:"formats"`
}

func Options() []Option {
	return []Option{
		{"h264", "H.264 - más compatible", "Recomendado para la mayoría de los editores y reproductores de video.", []string{"mp4", "mov", "mkv"}},
		{"h265", "H.265 - compresión eficiente", "Algunos editores de video no pueden abrir archivos H.265.", []string{"mp4", "mov", "mkv"}},
		{"vp9", "VP9 - para video WebM", "Compatible con MP4, WebM y MKV.", []string{"mp4", "webm", "mkv"}},
	}
}

func ValidTarget(codec, format string) bool {
	for _, option := range Options() {
		if option.ID == codec {
			for _, supported := range option.Formats {
				if supported == format {
					return true
				}
			}
		}
	}
	return false
}

func Arguments(input, output, codec, format string) ([]string, error) {
	if !ValidTarget(codec, format) {
		return nil, fmt.Errorf("La combinación de códec y formato no es compatible.")
	}
	args := []string{"-nostdin", "-n", "-protocol_whitelist", "file,pipe", "-i", input, "-map", "0:v:0", "-map", "0:a:0?", "-sn", "-dn",
		"-vf", "pad=ceil(iw/2)*2:ceil(ih/2)*2", "-pix_fmt", "yuv420p"}
	switch codec {
	case "h264":
		args = append(args, "-c:v", "libx264", "-crf", "18", "-preset", "medium")
	case "h265":
		args = append(args, "-c:v", "libx265", "-crf", "22", "-preset", "medium")
		if format == "mp4" || format == "mov" {
			args = append(args, "-tag:v", "hvc1")
		}
	case "vp9":
		args = append(args, "-c:v", "libvpx-vp9", "-crf", "30", "-b:v", "0", "-deadline", "good", "-cpu-used", "2")
	}
	if format == "webm" || (codec == "vp9" && format == "mkv") {
		args = append(args, "-c:a", "libopus", "-b:a", "128k")
	} else {
		args = append(args, "-c:a", "aac", "-b:a", "192k")
	}
	return append(args, "-progress", "pipe:1", "-nostats", output), nil
}

func DefaultFormat(codec, original string) string {
	original = strings.ToLower(original)
	if ValidTarget(codec, original) {
		return original
	}
	return "mp4"
}
