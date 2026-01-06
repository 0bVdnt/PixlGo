package main

import (
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

	// Frame dimensions - Leave room for status bar
	frameW := screenW
	frameH := (screenH - 2) * 2 // * 2 for half blocks

	// Maintain aspect ratio
	videoAspect := float64(meta.Width) / float64(meta.Height)
	frameAspect := float64(frameW) / float64(frameH)

	if frameAspect > videoAspect {
		frameW = int(float64(frameH) * videoAspect)
	} else {
		frameH = int(float64(frameW) / videoAspect)
	}

	// Ensure even dimensions
	frameW = (frameW / 2) * 2
	frameH = (frameH / 2) * 2

	// Extract initial frame
	currentTime := time.Duration(0)
	frame, err := decoder.ExtractFrame(currentTime, frameW, frameH)
	if err != nil {
		render.Close()
		fmt.Fprintf(os.Stderr, "Error extracting frame: %v\n", err)
		os.Exit(1)
	}

	// Main loop
	running := true
	for running {
		render.Clear()

		// Center the frame
		cellH := frameH / 2
		offsetX := (screenW - frameW) / 2
		offsetY := (screenH - cellH - 2) / 2

		// Render frame
		if frame != nil {
			render.RenderImage(frame.Image, offsetX, offsetY)
		}

		// Draw Status bar
		statusStyle := tcell.StyleDefault.
			Background(tcell.ColorDarkBlue).
			Foreground(tcell.ColorWhite)

		statusY := screenH - 2
		for x := range screenW {
			render.Screen().SetContent(x, statusY, ' ', nil, statusStyle)
		}

		status := fmt.Sprintf(" Position: %v / %v | Frame: %dx%d | Press Q to Quit, <-/-> to seek",
			currentTime.Round(time.Second),
			meta.Duration.Round(time.Second),
			frameW, frameH)
		render.DrawText(0, statusY, status, statusStyle)

		render.Show()

		// Handle events
		ev := render.Screen().PollEvent()
		switch ev := ev.(type) {
		case *tcell.EventKey:
			switch ev.Key() {
			case tcell.KeyEscape, tcell.KeyCtrlC:
				running = false
			case tcell.KeyRune:
				if ev.Rune() == 'q' || ev.Rune() == 'Q' {
					running = false
				}
			case tcell.KeyLeft:
				// Seek backwards 5 seconds
				currentTime -= 5 * time.Second
				if currentTime < 0 {
					currentTime = 0
				}
				frame, _ = decoder.ExtractFrame(currentTime, frameW, frameH)
			case tcell.KeyRight:
				// Seek forward 5 seconds
				currentTime += 5 * time.Second
				if currentTime > meta.Duration {
					currentTime = meta.Duration
				}
				frame, _ = decoder.ExtractFrame(currentTime, frameW, frameH)
			}
		case *tcell.EventResize:
			render.Screen().Sync()
			screenW, screenH = render.Size()
		}
	}
}
