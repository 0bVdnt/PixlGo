package renderer

import (
	"fmt"
	"image"
	"image/color"
	"strings"

	"github.com/gdamore/tcell/v2"
)

type Renderer struct {
	screen tcell.Screen
}

func New() (*Renderer, error) {
	screen, err := tcell.NewScreen()
	if err != nil {
		return nil, err
	}

	if err := screen.Init(); err != nil {
		return nil, err
	}

	screen.SetStyle(tcell.StyleDefault.Background(tcell.ColorBlack))
	screen.Clear()

	return &Renderer{screen: screen}, nil
}

// returns the underlying tcell screen
func (r *Renderer) Screen() tcell.Screen {
	return r.screen
}

// returns terminal dimensions
func (r *Renderer) Size() (width, height int) {
	return r.screen.Size()
}

// clears the screen
func (r *Renderer) Clear() {
	r.screen.Clear()
}

// Updates the screen
func (r *Renderer) Show() {
	r.screen.Show()
}

// cleans up the renderer
func (r *Renderer) Close() {
	r.screen.Fini()
}

// Renders an RGBA image to the screen at the given offset
func (r *Renderer) RenderImage(img *image.RGBA, offsetX, offsetY int) {
	if img == nil {
		return
	}
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()
	screenW, screenH := r.screen.Size()

	// Half blocks for 2 vertical pixels per cell
	for y := 0; y < height; y += 2 {
		cellY := offsetY + y/2
		if cellY >= screenH || cellY < 0 {
			continue
		}
		for x := range width {
			cellX := offsetX + x
			if cellX >= screenW || cellX < 0 {
				continue
			}

			// Top pixel (foreground)
			topC := img.RGBAAt(x, y)

			// Bottom pixel (background)
			var bottomC color.RGBA
			if y+1 < height {
				bottomC = img.RGBAAt(x, y+1)
			} else {
				bottomC = topC
			}

			style := tcell.StyleDefault.
				Foreground(tcell.NewRGBColor(int32(topC.R), int32(topC.G), int32(topC.B))).
				Background(tcell.NewRGBColor(int32(bottomC.R), int32(bottomC.G), int32(bottomC.B)))

			r.screen.SetContent(cellX, cellY, '▀', nil, style)
		}
	}
}

// Draws text at the given position
func (r *Renderer) DrawText(x, y int, text string, style tcell.Style) {
	for i, ch := range text {
		r.screen.SetContent(x+i, y, ch, nil, style)
	}
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
