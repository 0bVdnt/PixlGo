package main

import (
	"fmt"
	"os"
	"time"

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

	fmt.Println("\n=== Video Metadata ===")
	fmt.Printf("Resolution: %dx%d\n", meta.Width, meta.Height)
	fmt.Printf("FPS: %.2f\n", meta.FPS)
	fmt.Printf("Duration: %v\n", meta.Duration)
	fmt.Printf("Has audio: %v\n", meta.HasAudio)

	fmt.Println("\n=== Extracting Frame ===")
	frame, err := decoder.ExtractFrame(1*time.Second, 80, 40)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error extracting frame: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Frame extracted: %dx%d at %v\n", frame.Width, frame.Height, frame.Timestamp)

	fmt.Println("\nSample pixles(top-left corner):")
	for y := 0; y < 3; y++ {
		for x := 0; x < 5; x++ {
			c := frame.Image.RGBAAt(x, y)
			fmt.Printf("(%3d,%3d,%3d) ", c.R, c.G, c.B)
		}
		fmt.Println()
	}
}
