package renderer

import (
	"sync"

	"github.com/gdamore/tcell/v2"
)

type Renderer struct {
	mu         sync.Mutex
	screen     tcell.Screen
	prevCells  []uint64
	prevW      int
	prevH      int
	closed     bool
	needsClear bool
}

// Creates a new terminal renderer
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

	return &Renderer{
		screen:     screen,
		needsClear: true,
	}, nil
}

// Returns undelying tcell screen
func (r *Renderer) Screen() tcell.Screen {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.screen
}

// Returns terminal dimensions
func (r *Renderer) Size() (width, height int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.screen == nil || r.closed {
		return 80, 24
	}
	return r.screen.Size()
}

// Clears the screen
func (r *Renderer) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.screen != nil && !r.closed {
		r.screen.Clear()
	}
	r.prevCells = nil
	r.needsClear = true
}

// marks that full clear is needed
func (r *Renderer) RequestClear() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.needsClear = true
}

// Forces a full screen refresh
func (r *Renderer) Sync() {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.screen != nil && !r.closed {
		r.screen.Sync()
	}
}

// Clears the render cache
func (r *Renderer) InvalidateCache() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.prevCells = nil
}

// Returns whether the renderer is closed
func (r *Renderer) IsClosed() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.closed || r.screen == nil
}

// Shuts down the renderer
func (r *Renderer) Close() {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.closed {
		return
	}
	r.closed = true

	if r.screen != nil {
		r.screen.Fini()
		r.screen = nil
	}
}

// Clears video display area
func (r *Renderer) ClearVideoArea() {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.screen == nil || r.closed {
		return
	}

	w, h := r.screen.Size()
	style := tcell.StyleDefault.Background(tcell.ColorBlack)

	for y := 0; y < h-2; y++ {
		for x := 0; x < w; x++ {
			r.screen.SetContent(x, y, ' ', nil, style)
		}
	}

	r.needsClear = false
}

// returns and clears the needsClear flag
func (r *Renderer) NeedsClear() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	result := r.needsClear
	r.needsClear = false
	return result
}
