# PixlGo

A terminal video player written in Go. Decodes video files with FFmpeg and renders frames directly in the terminal using Unicode half-block characters and true color.

![Go](https://img.shields.io/badge/Go-1.24+-00ADD8?logo=go&logoColor=white)
![License](https://img.shields.io/badge/license-MIT-blue)

## How It Works

Each terminal cell displays two vertical pixels by combining a foreground color (upper pixel) and a background color (lower pixel) on the `▀` character. Frames are decoded from FFmpeg as raw RGB, converted to RGBA, and diffed against the previous frame so only changed cells are redrawn. The target FPS adapts automatically based on the rendered resolution to keep the terminal responsive.

## Prerequisites

- **Go** 1.24 or later
- **FFmpeg** and **FFprobe** installed and available on `PATH`

### Installing FFmpeg

**Debian / Ubuntu:**

```bash
sudo apt update && sudo apt install ffmpeg
```

**Fedora:**

```bash
sudo dnf install ffmpeg
```

**macOS (Homebrew):**

```bash
brew install ffmpeg
```

**Arch Linux:**

```bash
sudo pacman -S ffmpeg
```

Verify the installation:

```bash
ffmpeg -version
ffprobe -version
```

## Building

Clone the repository and build:

```bash
git clone https://github.com/0bVdnt/PixlGo.git
cd PixlGo
go build -o pixlgo ./cmd/pixlgo
```

Or install directly into your `$GOPATH/bin`:

```bash
go install github.com/0bVdnt/PixlGo/cmd/pixlgo@latest
```

## Usage

```
pixlgo [options] <video-file>
```

### Options

| Flag       | Description                               |
| ---------- | ----------------------------------------- |
| `-debug`   | Enable debug logging to `/tmp/pixlgo.log` |
| `-version` | Print version and exit                    |

### Examples

Play a video:

```bash
./pixlgo video.mp4
```

Play with debug logging enabled:

```bash
./pixlgo -debug video.mp4
```

Follow the debug log in another terminal:

```bash
tail -f /tmp/pixlgo.log
```

Check the version:

```bash
./pixlgo -version
```

## Controls

| Key            | Action                 |
| -------------- | ---------------------- |
| `Space`        | Pause / Resume         |
| `Q` / `Esc`    | Quit                   |
| `←` / `→`      | Seek ±5 seconds        |
| `↑` / `↓`      | Seek ±30 seconds       |
| `Home` / `End` | Jump to start / end    |
| `R`            | Restart from beginning |

## Project Structure

```
├── cmd/
│   └── pixlgo/
│       └── main.go            Entry point, flag parsing, signal handling
└── internal/
    ├── logger/
    │   └── logger.go          Thread-safe debug logger
    ├── player/
    │   ├── controls.go        Pause, seek, playback start
    │   ├── events.go          Keyboard and resize event handling
    │   ├── player.go          Main loop, lifecycle management
    │   ├── render.go          Frame rendering, UI drawing
    │   └── state.go           Player state, frame dimension calculation
    ├── renderer/
    │   ├── image.go           Half-block image rendering with diff cache
    │   ├── renderer.go        Terminal screen management (tcell)
    │   ├── terminal.go        ASCII/ANSI rendering helpers
    │   └── widgets.go         Text, progress bar, message widgets
    └── video/
        ├── decoder.go         FFmpeg process management, frame extraction
        ├── frame.go           Frame type and thread-safe frame buffer
        ├── probe.go           Video metadata extraction via ffprobe
        └── stream.go          Streaming decode with pacing and frame dropping
```

## Terminal Recommendations

For the best results:

- Use a terminal with **true color** (24-bit) support — kitty, Alacritty, iTerm2, WezTerm, Windows Terminal, or any modern terminal emulator.
- Use a **small font size** to increase the effective resolution (more cells = more pixels).
- **Maximize the terminal window** or run full-screen for the highest detail.
- Avoid terminal multiplexers like tmux or screen unless they are configured for true color passthrough.

## Dependencies

| Module                        | Purpose                                    |
| ----------------------------- | ------------------------------------------ |
| `github.com/gdamore/tcell/v2` | Terminal screen control and input handling |

## License

MIT
