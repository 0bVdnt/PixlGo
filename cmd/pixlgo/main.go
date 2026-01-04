package main

import (
	"fmt"
	"os"

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
}
