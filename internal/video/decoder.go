package video

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"image"
	"io"
	"os"
	"os/exec"
	"time"
)

var (
	ErrNoVideoStream = errors.New("no video stream found")
	ErrDecodeFailed  = errors.New("decode failed")
)

type Decoder struct {
	path     string
	metadata Metadata
}

// A decoded video frame
type Frame struct {
	Image     *image.RGBA
	Width     int
	Height    int
	Timestamp time.Duration
}

// Creates a new video decoder
func NewDecoder(videoPath string) (*Decoder, error) {
	// Check if file exists
	if _, err := os.Stat(videoPath); err != nil {
		return nil, fmt.Errorf("cannot access file: %w", err)
	}

	// Check if ffmpeg exists
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		return nil, fmt.Errorf("ffmpeg not found")
	}

	// Use the new Probe function
	meta, err := Probe(videoPath)
	if err != nil {
		return nil, err
	}

	return &Decoder{
		path:     videoPath,
		metadata: *meta,
	}, nil
}

// Returns video metadata
func (d *Decoder) Metadata() Metadata {
	return d.metadata
}

// Decode from given timestamp
func (d *Decoder) Stream(ctx context.Context, width, height int, startPos time.Duration) (<-chan *Frame, error) {
	width = (width / 2) * 2
	height = (height / 2) * 2

	if width < 2 {
		width = 2
	}
	if height < 2 {
		height = 2
	}

	args := []string{
		"-ss", fmt.Sprintf("%.3f", startPos.Seconds()),
		"-i", d.path,
		"-vf", fmt.Sprintf("scale=%d:%d", width, height),
		"-pix_fmt", "rgba",
		"-f", "rawvideo",
		"-loglevel", "quiet",
		"-", // Output to stdout
	}

	cmd := exec.CommandContext(ctx, "ffmpeg", args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to get stdout: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start ffmpeg: %w", err)
	}

	frameChan := make(chan *Frame, 2) // Small buffer

	go func() {
		defer close(frameChan)
		defer cmd.Wait()                // Ensure cleanup
		frameSize := width * height * 4 // RGBA
		buf := make([]byte, frameSize)

		// Use a buffered reader for better performance
		reader := bufio.NewReaderSize(stdout, frameSize*2)

		// Calculate duration per frame for timestamps
		frameDuration := time.Duration(float64(time.Second) / d.metadata.FPS)
		currentPos := startPos

		for {
			// Check if context cancelled
			select {
			case <-ctx.Done():
				return
			default:
			}

			// Read exactly one frame of bytes
			_, err := io.ReadFull(reader, buf)
			if err != nil {
				return // EOF or Error
			}

			// Copy buffer to image
			img := image.NewRGBA(image.Rect(0, 0, width, height))
			copy(img.Pix, buf)

			frame := &Frame{
				Image:     img,
				Width:     width,
				Height:    height,
				Timestamp: currentPos,
			}

			select {
			case frameChan <- frame:
				currentPos += frameDuration
			case <-ctx.Done():
				return
			}
		}
	}()

	return frameChan, nil
}

func (d *Decoder) ExtractFrame(timestamp time.Duration, width, height int) (*Frame, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Ensure even dimensions
	width = (width / 2) * 2
	height = (height / 2) * 2

	if width < 2 {
		width = 2
	}

	if height < 2 {
		height = 2
	}

	args := []string{
		"-ss", fmt.Sprintf("%.3f", timestamp.Seconds()),
		"-i", d.path,
		"-vframes", "1",
		"-vf", fmt.Sprintf("scale=%d:%d", width, height),
		"-pix_fmt", "rgba",
		"-f", "rawvideo",
		"-loglevel", "quiet",
		"-",
	}

	cmd := exec.CommandContext(ctx, "ffmpeg", args...)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to get stdout: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start ffmpeg: %w", err)
	}

	frameSize := width * height * 4 // RGBA
	buf := make([]byte, frameSize)

	reader := bufio.NewReader(stdout)
	_, err = io.ReadFull(reader, buf)
	if err != nil {
		cmd.Wait()
		return nil, fmt.Errorf("failed to read frame: %w", err)
	}

	cmd.Wait()

	img := image.NewRGBA(image.Rect(0, 0, width, height))
	copy(img.Pix, buf)

	return &Frame{
		Image:     img,
		Width:     width,
		Height:    height,
		Timestamp: timestamp,
	}, nil
}

func (d *Decoder) Close() {
	// Does nothing right now
}
