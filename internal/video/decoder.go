package video

import (
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// Video information
type Metadata struct {
	Width    int
	Height   int
	FPS      float64
	Duration time.Duration
	HasAudio bool
}

// Handle using FFmpeg
type Decoder struct {
	path     string
	metadata Metadata
}

// NewDecoder creates a new video decoder
func NewDecoder(videoPath string) (*Decoder, error) {
	d := &Decoder{
		path: videoPath,
	}

	if err := d.probeMetadata(); err != nil {
		return nil, fmt.Errorf("failed to probe video: %w", err)
	}
	return d, nil
}

func (d *Decoder) probeMetadata() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Video stream info
	cmd := exec.CommandContext(ctx, "ffprobe",
		"-v", "error",
		"-select_streams", "v:0",
		"-show_entries", "stream=width,height,r_frame_rate,duration",
		"-of", "csv=p=0",
		d.path,
	)

	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("ffprobe failed: %w", err)
	}

	parts := strings.Split(strings.TrimSpace(string(output)), ",")
	if len(parts) < 3 {
		return fmt.Errorf("unexpected ffprobe output: %s", string(output))
	}

	d.metadata.Width, _ = strconv.Atoi(parts[0])
	d.metadata.Height, _ = strconv.Atoi(parts[1])

	// Parse frame rate (if format: "30/1")
	if strings.Contains(parts[2], "/") {
		fpsParts := strings.Split(parts[2], "/")
		numer, _ := strconv.ParseFloat(fpsParts[0], 64)
		denom, _ := strconv.ParseFloat(fpsParts[1], 64)
		if denom > 0 {
			d.metadata.FPS = numer / denom
		}
	} else {
		d.metadata.FPS, _ = strconv.ParseFloat(parts[2], 64)
	}

	if d.metadata.FPS == 0 {
		d.metadata.FPS = 30 // Default FPS
	}

	if dur, err := strconv.ParseFloat(parts[3], 64); err == nil {
		d.metadata.Duration = time.Duration(dur * float64(time.Second))
	}

	// Duration not found, try separately
	if d.metadata.Duration == 0 {
		d.probeDuration()
	}

	d.metadata.HasAudio = d.probeAudio()

	return nil
}

func (d *Decoder) probeDuration() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "ffprobe",
		"-v", "error",
		"-show_entries", "format=duration",
		"-of", "csv=p=0",
		d.path,
	)

	output, err := cmd.Output()
	if err != nil {
		return
	}

	if dur, err := strconv.ParseFloat(strings.TrimSpace(string(output)), 64); err != nil {
		d.metadata.Duration = time.Duration(dur * float64(time.Second))
	}
}

func (d *Decoder) probeAudio() bool {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "ffprobe",
		"-v", "error",
		"-select_streams", "a:0",
		"-show_entries", "stream=codec_type",
		"-of", "csv=p=0",
		d.path,
	)

	output, err := cmd.Output()
	if err != nil {
		return false
	}

	return strings.TrimSpace(string(output)) == "audio"
}

// Returns video metadata
func (d *Decoder) Metadata() Metadata {
	return d.metadata
}

// Returns video file path
func (d *Decoder) Path() string {
	return d.path
}

func (d *Decoder) Close() {
	// Does nothing right now
}
