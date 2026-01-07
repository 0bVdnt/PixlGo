package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/0bVdnt/PixlGo/internal/renderer"
	"github.com/0bVdnt/PixlGo/internal/video"
	"github.com/gdamore/tcell/v2"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: pixlgo <video-flie>")
		os.Exit(1)
	}

	videoPath := os.Args[1]

	// Initialize decoder
	decoder, err := video.NewDecoder(videoPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	defer decoder.Close()

	meta := decoder.Metadata()

	// Initialize renderer
	render, err := renderer.New()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing renderer: %v\n", err)
		os.Exit(1)
	}
	defer render.Close()

	screenW, screenH := render.Size()

	// State
	var (
		frameChan    <-chan *video.Frame
		cancelStream context.CancelFunc
		ctx          context.Context
		currentFrame *video.Frame
		currentTime  = time.Duration(0)
	)

	// Function to restart the stream (used for init, resize and seek)
	restartStream := func(startPos time.Duration, w, h int) {
		if cancelStream != nil {
			cancelStream() // Stop previous ffmpeg process
		}
		ctx, cancelStream = context.WithCancel(context.Background())
		var err error
		frameChan, err = decoder.Stream(ctx, w, h, startPos)
		if err != nil {
			render.Close()
			fmt.Fprintf(os.Stderr, "Error streaming: %v\n", err)
			os.Exit(1)
		}
	}

	// Calculate initial dimensions
	frameW, frameH := calculateDimensions(screenW, screenH, meta)
	restartStream(0, frameW, frameH)

	// Input handling in separate goroutine
	eventChan := make(chan tcell.Event)
	go func() {
		for {
			eventChan <- render.Screen().PollEvent()
		}
	}()

	// Main loop
	ticker := time.NewTicker(time.Second / 60) // UI Limit
	defer ticker.Stop()

	running := true
	for running {
		select {
		// 1. Recieve new frame
		case frame, ok := <-frameChan:
			if !ok {
				running = false // End of the video
				break
			}
			currentFrame = frame
			currentTime = frame.Timestamp

			// If rendering faster than video FPS, this loop handles it. Channel blocks until FFmpeg decodes next frame
			// If FFmpeg is faster than Tcell the channel fills up FFmpeg pauses
		// 2. Handle input
		case ev := <-eventChan:
			switch ev := ev.(type) {
			case *tcell.EventResize:
				render.Screen().Sync()
				screenW, screenH = render.Size()
				frameW, frameH = calculateDimensions(screenW, screenH, meta)
				restartStream(currentTime, frameW, frameH)
			case *tcell.EventKey:
				switch ev.Key() {
				case tcell.KeyEscape, tcell.KeyCtrlC:
					running = false
				case tcell.KeyRune:
					if ev.Rune() == 'q' {
						running = false
					}
				case tcell.KeyLeft:
					// Seek backward 5 seconds - Needs restart
					newTime := currentTime - 5*time.Second
					if newTime < 0 {
						newTime = 0
					}
					restartStream(newTime, frameW, frameH)
				case tcell.KeyRight:
					// Seek forward 5 seconds - Needs restart
					newTime := currentTime + 5*time.Second
					if newTime > meta.Duration {
						newTime = meta.Duration
					}
					restartStream(newTime, frameW, frameH)
				}
			}
		// 3. Render tick (Updates UI)
		case <-ticker.C:
			// Only clear if necessary, overwriting is faster
			// render.Clear()

			if currentFrame != nil {
				// Center logic
				cellH := frameH / 2
				offsetX := (screenW - frameW) / 2
				offsetY := (screenH - cellH - 2) / 2
				render.RenderImage(currentFrame.Image, offsetX, offsetY)
			}

			// Draw status
			drawStatus(render, screenW, screenH, currentTime, meta, frameW, frameH)
			render.Show()
		}
	}

	// Cleanup
	if cancelStream != nil {
		cancelStream()
	}
}

func calculateDimensions(screenW, screenH int, meta video.Metadata) (int, int) {
	frameW := screenW
	frameH := (screenH - 2) * 2

	videoAspect := float64(meta.Width) / float64(meta.Height)
	frameAspect := float64(frameW) / float64(frameH)

	if frameAspect > videoAspect {
		frameW = int(float64(frameH) * videoAspect)
	} else {
		frameH = int(float64(frameW) / videoAspect)
	}

	return (frameW / 2) * 2, (frameH / 2) * 2
}

func drawStatus(r *renderer.Renderer, w, h int, curr time.Duration, meta video.Metadata, fw, fh int) {
	statusStyle := tcell.StyleDefault.
		Background(tcell.ColorDarkBlue).
		Foreground(tcell.ColorWhite)

	statusY := h - 1
	status := fmt.Sprintf(" %s / %s | %dx%d | Q: Quit, Arrows(<-/->): Seek",
		curr.Round(time.Second), meta.Duration.Round(time.Second), fw, fh)

	// fill background line
	for x := 0; x < w; x++ {
		r.Screen().SetContent(x, statusY, ' ', nil, statusStyle)
	}
	r.DrawText(0, statusY, status, statusStyle)
}
