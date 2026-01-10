package player

import "time"

func (p *Player) TogglePause() {
	p.mu.Lock()
	state := p.state.State
	currentTime := p.state.CurrentTime
	p.mu.Unlock()

	switch state {
	case StatePlaying:
		p.decoder.Stop()
		p.mu.Lock()
		p.state.State = StatePaused
		p.mu.Unlock()

	case StatePaused, StateEnded, StateStopped:
		p.StartPlayback(currentTime)
	}
}

func (p *Player) Seek(delta time.Duration) {
	p.mu.Lock()
	currentTime := p.state.CurrentTime
	duration := p.meta.Duration
	state := p.state.State
	frameW, frameH := p.state.FrameW, p.state.FrameH
	p.mu.Unlock()

	newTime := currentTime + delta

	if newTime < 0 {
		newTime = 0
	}
	if duration > 0 && newTime >= duration {
		newTime = duration - time.Second
		if newTime < 0 {
			newTime = 0
		}
	}

	p.mu.Lock()
	p.state.CurrentTime = newTime
	p.mu.Unlock()

	switch state {
	case StatePaused, StateEnded:
		go func() {
			if frame, err := p.decoder.ExtractFrame(newTime, frameW, frameH); err == nil {
				p.buffer.StoreForce(frame)
				p.mu.Lock()
				p.state.LastFrame = frame
				p.mu.Unlock()
			}
		}()

	case StatePlaying, StateLoading:
		p.StartPlayback(newTime)

	default:
		p.StartPlayback(newTime)
	}
}

func (p *Player) StartPlayback(pos time.Duration) {
	p.render.RequestClear()

	p.mu.Lock()
	p.state.CurrentTime = pos
	p.state.State = StateLoading
	p.state.LoadingStart = time.Now()
	frameW, frameH := p.state.FrameW, p.state.FrameH
	p.mu.Unlock()

	p.render.InvalidateCache()

	targetFPS := calculateTargetFPS(frameW, frameH)
	if err := p.decoder.StartStream(p.ctx, frameW, frameH, pos, p.buffer, targetFPS); err != nil {
		p.SetError("Start failed: " + err.Error())
	}
}

func (p *Player) SetError(msg string) {
	p.render.RequestClear()
	p.mu.Lock()
	p.state.State = StateError
	p.state.ErrorMsg = msg
	p.mu.Unlock()
}

func calculateTargetFPS(width, height int) float64 {
	targetFPS := 24.0
	pixels := width * height

	if pixels > 100000 {
		targetFPS = 12.0
	} else if pixels > 50000 {
		targetFPS = 16.0
	} else if pixels > 25000 {
		targetFPS = 20.0
	}

	return targetFPS
}
