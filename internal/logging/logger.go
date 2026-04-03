// Package logging provides a structured logger wrapping log/slog.
package logging

import (
	"log/slog"
	"os"
	"strings"
)

// Logger wraps slog with consistent field keys for ByteFlow Studio.
type Logger struct {
	l *slog.Logger
}

// Standard field key constants used across the application.
const (
	KeyBlockID    = "block_id"
	KeySessionID  = "session_id"
	KeyFlowState  = "flow_state"
	KeyError      = "error"
	KeyDurationMs = "duration_ms"
)

// New creates a new Logger reading BYTEFLOW_LOG_LEVEL and BYTEFLOW_LOG_FORMAT
// environment variables.
func New() *Logger {
	level := parseLevel(os.Getenv("BYTEFLOW_LOG_LEVEL"))
	opts := &slog.HandlerOptions{Level: level}

	var handler slog.Handler
	if strings.ToLower(os.Getenv("BYTEFLOW_LOG_FORMAT")) == "json" {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		handler = slog.NewTextHandler(os.Stdout, opts)
	}

	return &Logger{l: slog.New(handler)}
}

// parseLevel converts a string to a slog.Level, defaulting to INFO.
func parseLevel(s string) slog.Level {
	switch strings.ToUpper(s) {
	case "DEBUG":
		return slog.LevelDebug
	case "WARN", "WARNING":
		return slog.LevelWarn
	case "ERROR":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// Info logs at INFO level.
func (lg *Logger) Info(msg string, args ...any) {
	if lg == nil {
		return
	}
	lg.l.Info(msg, args...)
}

// Debug logs at DEBUG level.
func (lg *Logger) Debug(msg string, args ...any) {
	if lg == nil {
		return
	}
	lg.l.Debug(msg, args...)
}

// Warn logs at WARN level.
func (lg *Logger) Warn(msg string, args ...any) {
	if lg == nil {
		return
	}
	lg.l.Warn(msg, args...)
}

// Error logs at ERROR level.
func (lg *Logger) Error(msg string, args ...any) {
	if lg == nil {
		return
	}
	lg.l.Error(msg, args...)
}

// With returns a new Logger with additional fields.
func (lg *Logger) With(args ...any) *Logger {
	if lg == nil {
		return nil
	}
	return &Logger{l: lg.l.With(args...)}
}
