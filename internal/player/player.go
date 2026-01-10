package player

import (
	"context"
	"sync"
	"time"

	"github.com/0bVdnt/PixlGo/internal/logger"
	"github.com/0bVdnt/PixlGo/internal/renderer"
	"github.com/0bVdnt/PixlGo/internal/video"
	"github.com/gdamore/tcell/v2"
)

type Player struct {
	decoder *video.Decoder
	render  *renderer.Renderer
	buffer  *video.FrameBuffer
	meta    video.Metadata
	logger  *logger.Logger

	mu    sync.RWMutex
	state *PlayerState

	ctx      context.Context
	cancel   context.CancelFunc
	doneChan chan struct{}

	prevState State
}

type Config struct {
	VideoPath string
	Logger    *logger.Logger
}

func New(cfg Config) (*Player, error) {
	log := cfg.Logger
	if log == nil {
		log = logger.Noop()
	}

	log.Log("Creating decoder for: %s", cfg.VideoPath)

	decoder, err := video.NewDecoderWithLogger(cfg.VideoPath, log.Log)
	if err != nil {
		return nil, err
	}

	render, err := renderer.New()
	if err != nil {
		decoder.Close()
		return nil, err
	}

	ctx, cancel := context.WithCancel(context.Background())
	meta := decoder.Metadata()
	screenW, screenH := render.Size()

	return &Player{
		decoder:  decoder,
		render:   render,
		buffer:   video.NewFrameBuffer(),
		meta:     meta,
		logger:   log,
		state:    NewPlayerState(screenW, screenH, meta),
		ctx:      ctx,
		cancel:   cancel,
		doneChan: make(chan struct{}),
	}, nil
}

func (p *Player) Run() {
	defer p.cleanup()

	eventChan := make(chan tcell.Event, 50)
	go p.pollEvents(eventChan)

	time.Sleep(50 * time.Millisecond)
	p.drainInitialEvents(eventChan)

	p.mu.Lock()
	w, h := p.render.Size()
	p.state.UpdateDimensions(w, h, p.meta)
	p.mu.Unlock()

	p.StartPlayback(0)
	p.mainLoop(eventChan)
}

func (p *Player) mainLoop(eventchan <-chan tcell.Event) {
	ticker := time.NewTicker(33 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-p.ctx.Done():
			return

		case ev := <-eventchan:
			if ev == nil {
				return
			}
			if p.HandleEvent(ev) == EventQuit {
				return
			}

		case <-ticker.C:
			p.Update()
			p.Render()
		}
	}
}

func (p *Player) Update() {
	if err := p.buffer.GetError(); err != nil {
		p.mu.RLock()
		state := p.state.State
		p.mu.RUnlock()

		if state == StateLoading {
			p.SetError(err.Error())
		}
		return
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	switch p.state.State {
	case StateLoading:
		frame := p.buffer.Load()
		if frame != nil {
			p.state.LastFrame = frame
			p.state.CurrentTime = frame.Timestamp
			p.state.State = StatePlaying
		} else if time.Since(p.state.LoadingStart) > 10*time.Second {
			p.state.State = StateError
			p.state.ErrorMsg = "Timeout loading video"
		}
	case StatePlaying:
		frame := p.buffer.Load()
		if frame != nil {
			p.state.LastFrame = frame
			p.state.CurrentTime = frame.Timestamp
		}

		if !p.decoder.IsRunning() && p.buffer.FrameCount() > 0 {
			p.state.State = StateEnded
		}
	}
}

func (p *Player) pollEvents(eventChan chan<- tcell.Event) {
	screen := p.render.Screen()
	if screen == nil {
		return
	}

	for {
		ev := screen.PollEvent()
		if ev == nil {
			return
		}
		select {
		case eventChan <- ev:
		case <-p.doneChan:
			return
		case <-p.ctx.Done():
			return
		}
	}
}

func (p *Player) drainInitialEvents(eventchan <-chan tcell.Event) {
	for {
		select {
		case ev := <-eventchan:
			if ev == nil {
				return
			}
			if resize, ok := ev.(*tcell.EventResize); ok {
				w, h := resize.Size()
				p.mu.Lock()
				p.state.ScreenW = w
				p.state.ScreenH = h
				p.mu.Unlock()
			}
		case <-time.After(20 * time.Millisecond):
			return
		}
	}
}

func (p *Player) cleanup() {
	close(p.doneChan)
	p.decoder.Close()
	p.render.Close()
}

func (p *Player) Stop() {
	p.cancel()
}
