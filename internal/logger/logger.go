package logger

import (
	"fmt"
	"os"
	"sync"
	"time"
)

// Thread safe debug logging
type Logger struct {
	mu      sync.Mutex
	file    *os.File
	enabled bool
}

// Creates a new logger
func New(path string) (*Logger, error) {
	if path == "" {
		return &Logger{enabled: false}, nil
	}

	file, err := os.Create(path)
	if err != nil {
		return nil, err
	}

	return &Logger{
		file:    file,
		enabled: true,
	}, nil
}

// returns a no-op logger
func Noop() *Logger {
	return &Logger{enabled: false}
}

// Writes formatted message with timestamp
func (l *Logger) Log(format string, args ...any) {
	if !l.enabled || l.file == nil {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()

	timestamp := time.Now().Format("15:04:05.000")
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintf(l.file, "[%s] %s\n", timestamp, msg)
	l.file.Sync()
}

// Closes the log file
func (l *Logger) Close() {
	if l.file != nil {
		l.file.Close()
	}
}

// Returns whether logging is enabled
func (l *Logger) IsEnabled() bool {
	return l.enabled
}
