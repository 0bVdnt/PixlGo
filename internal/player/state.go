package player

import (
	"time"

	"github.com/0bVdnt/PixlGo/internal/video"
)

type State int

const (
	StateStopped State = iota
	StateLoading
	StatePlaying
	StatePaused
	StateError
	StateEnded
)

func (s State) String() string {
	switch s {
	case StateStopped:
		return "stopped"
	case StateLoading:
		return "loading"
	case StatePlaying:
		return "playing"
	case StatePaused:
		return "paused"
	case StateError:
		return "error"
	case StateEnded:
		return "ended"
	default:
		return "unknown"
	}
}

func (s State) Icon() string {
	switch s {
	case StatePlaying:
		return "▶"
	case StatePaused:
		return "⏸"
	case StateLoading:
		return "⏳"
	case StateError:
		return "⚠"
	case StateEnded:
		return "⏹"
	default:
		return "○"
	}
}

type PlayerState struct {
	State        State
	CurrentTime  time.Duration
	ErrorMsg     string
	LastFrame    *video.Frame
	LoadingStart time.Time

	ScreenW int
	ScreenH int
	FrameW  int
	FrameH  int
}

func NewPlayerState(screenW, screenH int, meta video.Metadata) *PlayerState {
	frameW, frameH := CalculateFrameDimensions(screenW, screenH, meta)
	return &PlayerState{
		State:   StateStopped,
		ScreenW: screenW,
		ScreenH: screenH,
		FrameW:  frameW,
		FrameH:  frameH,
	}
}

func CalculateFrameDimensions(screenW, screenH int, meta video.Metadata) (int, int) {
	availH := screenH - 3
	if availH < 2 {
		availH = 2
	}
	frameW := screenW
	frameH := availH * 2

	if meta.Width > 0 && meta.Height > 0 {
		aspect := float64(meta.Width) / float64(meta.Height)
		frameAspect := float64(frameW) / float64(frameH)

		if frameAspect > aspect {
			frameW = int(float64(frameH) * aspect)
		} else {
			frameH = int(float64(frameW) / aspect)
		}
	}

	frameW = clamp((frameW/2)*2, 4, screenW)
	frameH = clamp((frameH/2)*2, 4, availH*2)

	return frameW, frameH
}

func (ps *PlayerState) UpdateDimensions(screenW, screenH int, meta video.Metadata) bool {
	oldFrameW, oldFrameH := ps.FrameW, ps.FrameH

	ps.ScreenW = screenW
	ps.ScreenH = screenH
	ps.FrameW, ps.FrameH = CalculateFrameDimensions(screenW, screenH, meta)

	return ps.FrameW != oldFrameW || ps.FrameH != oldFrameH
}

func clamp(v, min, max int) int {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}
