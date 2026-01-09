package video

import (
	"image"
	"sync"
	"time"
)

// Represents a decoded video frame
type Frame struct {
	Image     *image.RGBA
	Timestamp time.Duration
}

// Provides thread-safe access to current frame
type FrameBuffer struct {
	mu         sync.RWMutex
	frame      *Frame
	epoch      uint64
	dropped    uint64
	frameCount uint64
	lastError  error
}

// Creates a new frame buffer
func NewFrameBuffer() *FrameBuffer {
	return &FrameBuffer{epoch: 1}
}

// Clears the buffer and increments the epoch
func (fb *FrameBuffer) Reset() uint64 {
	fb.mu.Lock()
	defer fb.mu.Unlock()
	fb.frame = nil
	fb.epoch++
	fb.dropped = 0
	fb.frameCount = 0
	fb.lastError = nil
	return fb.epoch
}

// Returns the current epoch
func (fb *FrameBuffer) Epoch() uint64 {
	fb.mu.RLock()
	defer fb.mu.Unlock()
	return fb.epoch
}

// Saves a new frame (only if epoch matches)
func (fb *FrameBuffer) Store(f *Frame, epoch uint64) bool {
	fb.mu.Lock()
	defer fb.mu.Unlock()

	if epoch != fb.epoch {
		return false
	}

	fb.frame = f
	fb.frameCount++
	return true
}

// Stores a frame without epoch check
func (fb *FrameBuffer) StoreForce(f *Frame) {
	fb.mu.Lock()
	defer fb.mu.Unlock()
	fb.frame = f
	fb.frameCount++
}

// Returns the current frame
func (fb *FrameBuffer) Load() *Frame {
	fb.mu.RLock()
	defer fb.mu.Unlock()
	return fb.frame
}

// Returns the count of dropped frames
func (fb *FrameBuffer) DroppedFrames() uint64 {
	fb.mu.RLock()
	defer fb.mu.RUnlock()
	return fb.dropped
}

// Returns total frames received
func (fb *FrameBuffer) FrameCount() uint64 {
	fb.mu.RLock()
	defer fb.mu.RUnlock()
	return fb.frameCount
}

// Increments the dropped frame counter
func (fb *FrameBuffer) AddDropped() {
	fb.mu.Lock()
	fb.dropped++
	fb.mu.Unlock()
}

// Sets an error state
func (fb *FrameBuffer) SetError(err error) {
	fb.mu.Lock()
	fb.lastError = err
	fb.mu.Unlock()
}

// Returns last error
func (fb *FrameBuffer) GetError() error {
	fb.mu.RLock()
	defer fb.mu.RUnlock()
	return fb.lastError
}

// Returns the current frame's timestamp
func (fb *FrameBuffer) Timestamp() time.Duration {
	fb.mu.RLock()
	defer fb.mu.RUnlock()
	if fb.frame != nil {
		return fb.frame.Timestamp
	}
	return 0
}
