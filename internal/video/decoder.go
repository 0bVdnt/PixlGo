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
	"sync"
	"time"
)

type LogFunc func(format string, args ...any)

var (
	ErrNoVideoStream = errors.New("no video stream found")
	ErrDecodeFailed  = errors.New("decode failed")
)

type Decoder struct {
	path     string
	metadata Metadata
	logFn    LogFunc

	mu      sync.Mutex
	stream  *Stream
	running bool
}

// Creates a new video decoder
func NewDecoder(path string) (*Decoder, error) {
	return NewDecoderWithLogger(path, nil)
}

func NewDecoderWithLogger(path string, logFn LogFunc) (*Decoder, error) {
	if logFn == nil {
		logFn = func(format string, args ...any) {}
	}
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("cannot access file: %w", err)
	}
	logFn("File: %s (%d bytes)", path, info.Size())

	if _, err := exec.LookPath("ffmpeg"); err != nil {
		return nil, fmt.Errorf("ffmpeg not found")
	}

	meta, err := Probe(path)
	if err != nil {
		return nil, err
	}

	logFn("Metadata: %dx%d @ %.2f fps, codec=%s, duration=%v",
		meta.Width, meta.Height, meta.FPS, meta.Codec, meta.Duration)

	return &Decoder{
		path:     path,
		metadata: *meta,
		logFn:    logFn,
	}, nil
}

// Returns video metadata
func (d *Decoder) Metadata() Metadata {
	return d.metadata
}

// Returns the path of the video
func (d *Decoder) Path() string {
	return d.path
}

func (d *Decoder) IsRunning() bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.running
}

// Stops the current stream
func (d *Decoder) Stop() {
	d.mu.Lock()
	stream := d.stream
	d.stream = nil
	d.running = false
	d.mu.Unlock()

	if stream != nil {
		stream.Stop(d.logFn)
	}
}

func (d *Decoder) Close() {
	d.Stop()
}

// Begin decoding video frames
func (d *Decoder) StartStream(ctx context.Context, width, height int,
	startPos time.Duration, buffer *FrameBuffer, targetFPS float64) error {
	d.Stop()
	epoch := buffer.Reset()

	if targetFPS <= 0 {
		targetFPS = DefaultTargetFPS(width, height, d.metadata.FPS)
	}

	d.logFn("[epoch=%d] StartStream: %dx%d @ %.1f fps, startPos=%v",
		epoch, width, height, targetFPS, startPos)

	config := StreamConfig{
		Width:     width,
		Height:    height,
		StartPos:  startPos,
		TargetFPS: targetFPS,
	}

	stream, err := StartStream(ctx, d.path, config, epoch, d.logFn)
	if err != nil {
		return err
	}

	d.mu.Lock()
	d.stream = stream
	d.running = true
	d.mu.Unlock()

	go func() {
		stream.ReadFrames(buffer, d.logFn)
		d.mu.Lock()
		if d.stream == stream {
			d.running = false
		}
		d.mu.Unlock()
	}()
	return nil
}

func (d *Decoder) ExtractFrame(timestamp time.Duration, width, height int) (*Frame, error) {
	return ExtractSingleFrame(d.path, timestamp, width, height)
}

func ExtractSingleFrame(path string, timestamp time.Duration, width, height int) (*Frame, error) {
	width = normalizeEven(width, 4, 4096)
	height = normalizeEven(height, 4, 4096)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "ffmpeg",
		"-ss", fmt.Sprintf("%.3f", timestamp.Seconds()),
		"-i", path,
		"-vframes", "1",
		"-vf", fmt.Sprintf("scale=%d:%d", width, height),
		"-pix_fmt", "rgb24",
		"-f", "rawvideo",
		"-loglevel", "error",
		"-",
	)

	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("extract frame: %w", err)
	}

	expectedSize := width * height * 3
	if len(out) < expectedSize {
		return nil, fmt.Errorf("incomplete: got %d, want %d", len(out), expectedSize)
	}

	frame := &Frame{
		Image:     createRGBAFromRGB24(out[:expectedSize], width, height),
		Timestamp: timestamp,
	}
	return frame, nil
}

func createRGBAFromRGB24(rgb []byte, width, height int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	convertRGB24ToRGBA(rgb, img.Pix)
	return img
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
