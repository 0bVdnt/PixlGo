package renderer

import (
	"fmt"
	"image"
	"image/color"
	"strings"
)

// Updates the screen
func (r *Renderer) Show() {
	r.screen.Show()
}

// Render image as ascii art
func (r *Renderer) RenderASCII(img *image.RGBA) string {
	if img == nil {
		return ""
	}
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	// ASCII character set
	chars := []rune(" .:-=+*#%@")

	var sb strings.Builder
	for y := range height {
		for x := range width {
			c := img.RGBAAt(x, y)

			// Calculate brightness (0-255)
			brightness := (int(c.R) + int(c.G) + int(c.B)) / 3

			// Map to character index
			idx := brightness * (len(chars) - 1) / 255
			sb.WriteRune(chars[idx])
		}
		sb.WriteRune('\n')
	}
	return sb.String()
}

// RenderColor renders an image with ANSI colors using half blocks
func (r *Renderer) RenderColor(img *image.RGBA) string {
	if img == nil {
		return ""
	}

	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	var sb strings.Builder

	// Use half blocks - each character is represents 2 vertical pixels
	for y := range height {
		for x := range width {
			// Top pixel - Foreground
			top := img.RGBAAt(x, y)

			// Bottom pixel - Background
			var bottom color.RGBA
			if y+1 < height {
				bottom = img.RGBAAt(x, y+1)
			} else {
				bottom = top
			}

			// ANSI escape: set foreground and background, print upper half block
			fmt.Fprintf(&sb, "\x1b[38;2;%d;%d;%dm\x1b[48;2;%d;%d;%dm▀",
				top.R, top.G, top.B,
				bottom.R, bottom.G, bottom.B)
		}
		sb.WriteString("\x1b[0m\n")
	}
	return sb.String()
}

// moves cursor to a position
func (r *Renderer) MoveCursor(x, y int) {
	fmt.Printf("\x1b[%d;%dH", y+1, x+1)
}

// hides the cursor
func (r *Renderer) HideCursor() {
	fmt.Print("\x1b[?25l")
}

// ShowCursor shows the cursor
func (r *Renderer) ShowCursor() {
	fmt.Print("\x1b[?25h")
}
