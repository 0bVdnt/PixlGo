package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/0bVdnt/PixlGo/internal/logger"
	"github.com/0bVdnt/PixlGo/internal/player"
)

var (
	debugMode bool
	version   = "0.1.0"
)

func main() {
	flag.BoolVar(&debugMode, "debug", false, "Enable debug logging to /tmp/pixlgo.log")
	showVersion := flag.Bool("version", false, "Show version")
	flag.Parse()

	if *showVersion {
		fmt.Printf("pixlgo v%s\n", version)
		os.Exit(0)
	}

	args := flag.Args()
	if len(args) < 1 {
		printUsage()
		os.Exit(1)
	}
	videoPath := args[0]

	// Setup logging
	var log *logger.Logger
	var err error

	if debugMode {
		log, err = logger.New("/tmp/pixlgo.log")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: could not create log file: %v\n", err)
			log = logger.Noop()
		} else {
			defer log.Close()
		}
	} else {
		log = logger.Noop()
	}

	log.Log("pixlgo: v%s starting", version)
	log.Log("Video: %s", videoPath)

	// Create player
	p, err := player.New(player.Config{
		VideoPath: videoPath,
		Logger:    log,
	})

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		log.Log("Signal received")
		p.Stop()
	}()

	// Run player
	p.Run()

	log.Log("Exiting")
}

func printUsage() {
	fmt.Println("pixlgo - Terminal video player")
	fmt.Println()
	fmt.Println("Usage: pixlgo [options] <video-file>")
	fmt.Println()
	fmt.Println("Options:")
	fmt.Println("  -debug    Enable debug logging to /tmp/pixlgo.log")
	fmt.Println("  -version  Show version")
	fmt.Println()
	fmt.Println("Controls:")
	fmt.Println("  Space	   Pause/Resume")
	fmt.Println("  Q/Esc	   Quit")
	fmt.Println("  Left/Right  Seek ±5s")
	fmt.Println("  Up/Down 	   Seek ±30s")
	fmt.Println("  R           Restart")
	fmt.Println("  Home/End    Go to start/end")
}
