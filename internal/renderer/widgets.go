package renderer

import "github.com/gdamore/tcell/v2"

// draw text at specified position
func (r *Renderer) DrawText(x, y int, text string, style tcell.Style) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.screen == nil || r.closed {
		return
	}

	w, h := r.screen.Size()
	if y < 0 || y >= h {
		return
	}

	for i, ch := range text {
		if x+i >= 0 && x+i < w {
			r.screen.SetContent(x+i, y, ch, nil, style)
		}
	}
}

// Fills a horizontal line with a style
func (r *Renderer) FillLine(y int, style tcell.Style) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.screen == nil || r.closed {
		return
	}

	w, h := r.screen.Size()
	if y < 0 || y >= h {
		return
	}

	for x := range w {
		r.screen.SetContent(x, y, ' ', nil, style)
	}
}

// Displays a centered message
func (r *Renderer) RenderMessage(msg string, bgColor tcell.Color) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.screen == nil || r.closed {
		return
	}

	w, h := r.screen.Size()
	if w <= 0 || h <= 0 {
		return
	}

	style := tcell.StyleDefault.Background(bgColor).Foreground(tcell.ColorWhite)

	y := h / 2
	for x := range w {
		r.screen.SetContent(x, y, ' ', nil, style)
	}

	x := (w - len(msg)) / 2
	if x < 0 {
		x = 0
	}
	for i, ch := range msg {
		if x+i < w {
			r.screen.SetContent(x+i, y, ch, nil, style)
		}
	}
}

// Draws a horizontal progress bar
func (r *Renderer) ProgressBar(y int, progress float64, filledColor, emptyColor tcell.Color) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.screen == nil || r.closed {
		return
	}

	w, h := r.screen.Size()
	if y < 0 || y >= h || w < 4 {
		return
	}

	if progress < 0 {
		progress = 0
	}
	if progress > 1 {
		progress = 1
	}

	barW := w - 2
	filled := int(float64(barW) * progress)

	filledStyle := tcell.StyleDefault.Background(filledColor)
	emptyStyle := tcell.StyleDefault.Background(emptyColor)

	for x := 1; x < 1+filled && x < w-1; x++ {
		r.screen.SetContent(x, y, '━', nil, filledStyle)
	}
	for x := 1 + filled; x < 1+barW && x < w-1; x++ {
		r.screen.SetContent(x, y, '─', nil, emptyStyle)
	}

	// Position marker
	mx := 1 + filled
	if mx >= w-1 {
		mx = w - 2
	}
	r.screen.SetContent(mx, y, '●', nil, tcell.StyleDefault.Foreground(tcell.ColorWhite))
}
