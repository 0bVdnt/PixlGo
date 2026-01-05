package main

import (
	"fmt"
	"os"
	"time"

	"github.com/0bVdnt/PixlGo/internal/renderer"
	"github.com/0bVdnt/PixlGo/internal/video"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: pixlgo <video-flie>")
		os.Exit(1)
	}

	videoPath := os.Args[1]

	fmt.Printf("Opening: %s\n", videoPath)

	decoder, err := video.NewDecoder(videoPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	defer decoder.Close()

	meta := decoder.Metadata()

	render := renderer.New()

	// Extract a frame - use small size for terminal
	frame, err := decoder.ExtractFrame(5*time.Second, 80, 40)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error extracting frame: %v\n", err)
		os.Exit(1)
	}

	// Clear screen and render
	render.Clear()
	render.HideCursor()

	// Print video info
	fmt.Printf("Video: %s (%dx%d @ %.1f fps)\n", videoPath, meta.Width, meta.Height, meta.FPS)
	fmt.Printf("Frame at: %v\n\n", frame.Timestamp)

	// Render as Colored Pixel Art
	output := render.RenderColor(frame.Image)
	fmt.Print(output)

	render.ShowCursor()
}
