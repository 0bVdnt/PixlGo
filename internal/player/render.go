package player

import (
	"fmt"
	"time"

	"github.com/gdamore/tcell/v2"
)

func (p *Player) Render() {
	if p.render.IsClosed() {
		return
	}

	p.mu.RLock()
	state := p.state.State
	lastFrame := p.state.LastFrame
	errorMsg := p.state.ErrorMsg
	screenW, screenH := p.state.ScreenW, p.state.ScreenH
	frameW, frameH := p.state.FrameW, p.state.FrameH
	currentTime := p.state.CurrentTime
	p.mu.RUnlock()

	stateChanged := state != p.prevState
	if stateChanged {
		p.render.RequestClear()
		p.render.InvalidateCache()
		p.prevState = state
	}

	if p.render.NeedsClear() {
		p.render.ClearVideoArea()
	}

	switch state {
	case StateLoading:
		p.render.RenderMessage("Loading video...", tcell.ColorDarkBlue)

	case StateError:
		p.render.RenderMessage(errorMsg, tcell.ColorDarkRed)

	default:
		if lastFrame != nil {
			cellH := frameH / 2
			offsetX := (screenW - frameW) / 2
			offsetY := (screenH - cellH - 3) / 2
			if offsetX < 0 {
				offsetX = 0
			}
			if offsetY < 0 {
				offsetY = 0
			}

			p.render.RenderImage(lastFrame.Image, offsetX, offsetY)
		} else {
			p.render.RenderMessage("Waiting...", tcell.ColorDarkBlue)
		}
	}

	p.renderUI(screenW, screenH, frameW, frameH, currentTime, state)
	p.render.Show()
}

func (p *Player) renderUI(w, h, frameW, frameH int, currentTime time.Duration, state State) {
	if w < 10 || h < 5 {
		return
	}

	p.mu.RLock()
	duration := p.meta.Duration
	codec := p.meta.Codec
	dropped := p.buffer.DroppedFrames()
	p.mu.RUnlock()

	// Progress bar
	barY := h - 2
	bgStyle := tcell.StyleDefault.Background(tcell.ColorBlack)
	p.render.FillLine(barY, bgStyle)

	if duration > 0 {
		progress := float64(currentTime) / float64(duration)
		p.render.ProgressBar(barY, progress, tcell.ColorGreen, tcell.ColorDarkGray)
	}

	// Status bar
	statusY := h - 1
	statusStyle := tcell.StyleDefault.
		Background(tcell.ColorDarkBlue).
		Foreground(tcell.ColorWhite)

	p.render.FillLine(statusY, statusStyle)

	if codec == "" {
		codec = "?"
	}

	droppedStr := ""
	if dropped > 0 {
		droppedStr = fmt.Sprintf(" D:%d", dropped)
	}

	status := fmt.Sprintf(" %s %s/%s │ %s │ %dx%d%s | Q: quit SPC:pause <-/->: seek",
		state.Icon(),
		formatDuration(currentTime),
		formatDuration(duration),
		codec,
		frameW, frameH,
		droppedStr,
	)

	if len(status) > w {
		status = status[:w]
	}

	p.render.DrawText(0, statusY, status, statusStyle)
}

func formatDuration(d time.Duration) string {
	if d < 0 {
		d = 0
	}

	d = d.Round(time.Second)
	h := d / time.Hour
	m := (d % time.Hour) / time.Minute
	s := (d % time.Minute) / time.Second

	if h > 0 {
		return fmt.Sprintf("%d:%02d:%02d", h, m, s)
	}
	return fmt.Sprintf("%d:%02d", m, s)
}
