package player

import (
	"time"

	"github.com/gdamore/tcell/v2"
)

const (
	SeekSmall = 5 * time.Second
	SeekLarge = 30 * time.Second
)

type EventResult int

const (
	EventContinue EventResult = iota
	EventQuit
)

func (p *Player) HandleEvent(ev tcell.Event) EventResult {
	switch ev := ev.(type) {
	case *tcell.EventResize:
		return p.handleResize(ev)
	case *tcell.EventKey:
		return p.handleKey(ev)
	}
	return EventContinue
}

func (p *Player) handleResize(ev *tcell.EventResize) EventResult {
	w, h := ev.Size()

	p.render.Sync()
	p.render.Clear()
	p.render.InvalidateCache()

	p.mu.Lock()
	dimensionsChanged := p.state.UpdateDimensions(w, h, p.meta)
	state := p.state.State
	currentTime := p.state.CurrentTime
	p.mu.Unlock()

	if dimensionsChanged && (state == StatePlaying || state == StateLoading) {
		p.StartPlayback(currentTime)
	}

	return EventContinue
}

func (p *Player) handleKey(ev *tcell.EventKey) EventResult {
	if ev.Key() == tcell.KeyEscape || ev.Key() == tcell.KeyCtrlC {
		return EventQuit
	}
	if ev.Key() == tcell.KeyRune && (ev.Rune() == 'q' || ev.Rune() == 'Q') {
		return EventQuit
	}

	p.mu.Lock()
	if p.state.State == StateError {
		p.state.State = StateStopped
		p.state.ErrorMsg = ""
		p.render.RequestClear()
	}
	p.mu.Unlock()

	switch ev.Key() {
	case tcell.KeyRune:
		return p.handleRune(ev.Rune())
	case tcell.KeyLeft:
		p.Seek(-SeekSmall)
	case tcell.KeyRight:
		p.Seek(SeekSmall)
	case tcell.KeyDown:
		p.Seek(-SeekLarge)
	case tcell.KeyUp:
		p.Seek(SeekLarge)
	case tcell.KeyHome:
		p.mu.RLock()
		ct := p.state.CurrentTime
		p.mu.RUnlock()
		p.Seek(-ct)
	case tcell.KeyEnd:
		p.mu.RLock()
		ct := p.state.CurrentTime
		dur := p.meta.Duration
		p.mu.RUnlock()
		if dur > time.Second {
			p.Seek(dur - ct - time.Second)
		}
	}
	return EventContinue
}

func (p *Player) handleRune(r rune) EventResult {
	switch r {
	case ' ':
		p.TogglePause()
	case 'r', 'R':
		p.render.Clear()
		p.StartPlayback(0)
	}
	return EventContinue
}
