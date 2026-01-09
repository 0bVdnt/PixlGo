package video

import (
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// Contains video file information
type Metadata struct {
	Width    int
	Height   int
	FPS      float64
	Duration time.Duration
	codec    string
}

// Checks if metadata has all the required fields
func (m *Metadata) IsValid() bool {
	return m.Width > 0 && m.Height > 0
}

// Extracts metadata from the video file
func Probe(path string) (*Metadata, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	meta := &Metadata{}

	// Probe video stream
	if err := probeVideoStream(ctx, path, meta); err != nil {
		return nil, err
	}

	// Probe Duration
	probeDuration(ctx, path, meta)

	// Set defaults
	if meta.FPS <= 0 {
		meta.FPS = 25
	}

	if !meta.IsValid() {
		return nil, fmt.Errorf("no video stream found")
	}

	return meta, nil
}

func probeVideoStream(ctx context.Context, path string, meta *Metadata) error {
	// Video stream info
	cmd := exec.CommandContext(ctx, "ffprobe",
		"-v", "error",
		"-select_streams", "v:0",
		"-show_entries", "stream=width,height,r_frame_rate,codec_name",
		"-of", "default=noprint_wrappers=1",
		path,
	)

	out, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("ffprobe failed: %w", err)
	}

	parseProbeOutput(string(out), meta)
	return nil
}

func parseProbeOutput(output string, meta *Metadata) {
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		idx := strings.Index(line, "=")
		if idx <= 0 {
			continue
		}

		key := line[:idx]
		val := line[idx+1:]

		switch key {
		case "width":
			meta.Width, _ = strconv.Atoi(val)
		case "height":
			meta.Height, _ = strconv.Atoi(val)
		case "r_frame_rate":
			meta.FPS = parseFPS(val)
		case "codec_name":
			meta.codec = val
		}
	}
}

func probeDuration(ctx context.Context, path string, meta *Metadata) {
	cmd := exec.CommandContext(ctx, "ffprobe",
		"-v", "error",
		"-show_entries", "format=duration",
		"-of", "default=noprint_wrappers=1:nokey=1",
		path,
	)

	out, err := cmd.Output()
	if err != nil {
		return
	}

	if dur, err := strconv.ParseFloat(strings.TrimSpace(string(out)), 64); err == nil && dur > 0 {
		meta.Duration = time.Duration(dur * float64(time.Second))
	}
}

func parseFPS(s string) float64 {
	s = strings.TrimSpace(s)
	if idx := strings.Index(s, "/"); idx > 0 {
		num, _ := strconv.ParseFloat(s[:idx], 64)
		den, _ := strconv.ParseFloat(s[idx+1:], 64)
		if den > 0 {
			return num / den
		}
		return 0
	}
	fps, _ := strconv.ParseFloat(s, 64)
	return fps
}
