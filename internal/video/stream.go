package video

import (
	"bufio"
	"context"
	"fmt"
	"image"
	"io"
	"os/exec"
	"runtime"
	"sync"
	"time"
)

// Holds streaming parameters
type StreamConfig struct {
	Width     int
	Height    int
	StartPos  time.Duration
	TargetFPS float64
}

// Calculates an appropriate FPS based on frame size
func DefaultTargetFPS(width, height int, sourceFPS float64) float64 {
	targetFPS := 24.0
	pixels := width * height
	if pixels > 100000 {
		targetFPS = 12
	} else if pixels > 50000 {
		targetFPS = 15
	} else if pixels > 25000 {
		targetFPS = 20
	}

	if sourceFPS > 0 && targetFPS > sourceFPS {
		targetFPS = sourceFPS
	}

	return targetFPS
}

// Manages the ffmpeg decode process
type Stream struct {
	cmd    *exec.Cmd
	cancel context.CancelFunc
	stdout io.ReadCloser
	stderr io.ReadCloser

	width     int
	height    int
	frameSize int
	fps       float64
	epoch     uint64
	startPos  time.Duration

	mu      sync.Mutex
	stopped bool
	done    chan struct{}
}

// Creates and starts a new decode stream
func StartStream(ctx context.Context, path string, config StreamConfig,
	epoch uint64, logFn func(string, ...any)) (*Stream, error) {
	width := normalizeEven(config.Width, 4, 4096)
	height := normalizeEven(config.Height, 4, 4096)

	args := buildFFmpegArgs(path, width, height, config.StartPos, config.TargetFPS)
	if logFn != nil {
		logFn("[epoch=%d] FFmpeg args: %w", epoch, args)
	}

	cmdCtx, cancel := context.WithCancel(ctx)
	cmd := exec.CommandContext(cmdCtx, "ffmpeg", args...)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		cancel()
		return nil, fmt.Errorf("stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		cancel()
		stdout.Close()
		return nil, fmt.Errorf("stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		cancel()
		stdout.Close()
		stderr.Close()
		return nil, fmt.Errorf("start: %w", err)
	}

	if logFn != nil {
		logFn("[epoch=%d] FFmpeg started, PID=%d", epoch, cmd.Process.Pid)
	}

	return &Stream{
		cmd:       cmd,
		cancel:    cancel,
		stdout:    stdout,
		stderr:    stderr,
		width:     width,
		height:    height,
		frameSize: width * height * 3,
		fps:       config.TargetFPS,
		epoch:     epoch,
		startPos:  config.StartPos,
		done:      make(chan struct{}),
	}, nil
}

// Builds arguments for FFmpeg
func buildFFmpegArgs(path string, width, height int, startPos time.Duration, fps float64) []string {
	args := []string{
		"-threads", fmt.Sprintf("%d", runtime.NumCPU()),
	}

	if startPos > 0 {
		args = append(args, "-ss", fmt.Sprintf("%.3f", startPos.Seconds()))
	}

	args = append(args,
		"-i", path,
		"-vf", fmt.Sprintf("fps=%.2f,scale=%d:%d", fps, width, height),
		"-pix_fmt", "rgb24",
		"-f", "rawvideo",
		"-an",
		"-sn",
		"-loglevel", "error",
		"-",
	)
	return args
}

// Reads frames from the stream and sends to buffer
func (s *Stream) ReadFrames(buffer *FrameBuffer, logFn func(string, ...any)) {
	defer func() {
		close(s.done)
		s.stdout.Close()
		s.cmd.Wait()
		if logFn != nil {
			logFn("[epoch=%d] Stream read loop exited", s.epoch)
		}
	}()

	// Start stderr reader
	go s.drainStderr(logFn)

	frameDuration := time.Duration(float64(time.Second) / s.fps)

	reader := bufio.NewReaderSize(s.stdout, s.frameSize*4)

	// Double buffer for frames
	frames := [2]*Frame{
		{Image: image.NewRGBA(image.Rect(0, 0, s.width, s.height))},
		{Image: image.NewRGBA(image.Rect(0, 0, s.width, s.height))},
	}
	frameIdx := 0

	rgbBuf := make([]byte, s.frameSize)
	currentTime := s.startPos
	playbackStart := time.Now()
	frameNum := 0

	for {
		// Check if stopped
		s.mu.Lock()
		stopped := s.stopped
		s.mu.Unlock()
		if stopped {
			return
		}

		// Check epoch before reading
		if buffer.Epoch() != s.epoch {
			return
		}
		_, err := io.ReadFull(reader, rgbBuf)
		if err != nil {
			if frameNum == 0 {
				buffer.SetError(ErrDecodeFailed)
			}
			return
		}

		// Timing check for frame dropping
		expectedTime := playbackStart.Add(time.Duration(frameNum) * frameDuration)
		now := time.Now()
		lag := now.Sub(expectedTime)

		if lag > frameDuration*5 {
			buffer.AddDropped()
			frameNum++
			currentTime += frameDuration
			continue
		}

		// Convert RGB24 to RGBA
		frame := frames[frameIdx]
		frameIdx = 1 - frameIdx
		convertRGB24ToRGBA(rgbBuf, frame.Image.Pix)
		frame.Timestamp = currentTime

		// Store with epoch check
		if !buffer.Store(frame, s.epoch) {
			return
		}

		frameNum++
		currentTime -= frameDuration

		// Pace control
		if lag < -5*time.Millisecond {
			time.Sleep(-lag - 2*time.Millisecond)
		}
	}
}

func (s *Stream) drainStderr(logFn func(string, ...any)) {
	buf := make([]byte, 1024)
	for {
		n, err := s.stderr.Read(buf)
		if n > 0 && logFn != nil {
			logFn("[epoch=%d] FFmpeg stderr: %s", s.epoch, string(buf[:n]))
		}
		if err != nil {
			break
		}
	}
	s.stderr.Close()
}

// Terminates the stream and waits for it to finish
func (s *Stream) Stop(logFn func(string, ...any)) {
	s.mu.Lock()
	if s.stopped {
		s.mu.Unlock()
		return
	}
	s.stopped = true
	s.mu.Unlock()
	if s.cancel != nil {
		s.cancel()
	}
	if s.cmd != nil && s.cmd.Process != nil {
		s.cmd.Process.Kill()
	}

	// Wait for read loop to finish
	select {
	case <-s.done:
	case <-time.After(500 * time.Millisecond):
	}
}

// Returns a channel that's closed when the stream finishes
func (s *Stream) Done() <-chan struct{} {
	return s.done
}

// Epoch returns the stream's epoch
func (s *Stream) Epoch() uint64 {
	return s.epoch
}

func convertRGB24ToRGBA(src, dst []byte) {
	for i, j := 0, 0; i < len(src); i, j = i+3, j+4 {
		dst[j] = src[i]
		dst[j+1] = src[i+1]
		dst[j+2] = src[i+2]
		dst[j+3] = 255
	}
}

func normalizeEven(v, min, max int) int {
	v = (v / 2) * 2
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}
