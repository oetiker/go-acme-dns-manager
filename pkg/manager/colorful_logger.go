// ColorfulLogger provides a human-friendly log output with emojis and colors
package manager

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
)

// ANSI color codes
const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorBlue   = "\033[36m"
	colorBold   = "\033[1m"
)

// Emoji for different log levels
const (
	emojiDebug = "🔍"  // Magnifying glass
	emojiInfo  = "ℹ️" // Information
	emojiWarn  = "⚠️" // Warning
	emojiError = "❌"  // Cross mark
)

// SimpleHandler is a basic slog.Handler that doesn't print timestamps
// and can use colors and emojis
type SimpleHandler struct {
	w         io.Writer
	level     slog.Leveler
	useColors bool
	useEmoji  bool
}

// Enabled implements slog.Handler.
func (h *SimpleHandler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= h.level.Level()
}

// Handle implements slog.Handler.
func (h *SimpleHandler) Handle(_ context.Context, r slog.Record) error {
	var prefix string

	// Format level with emoji and/or colors
	switch r.Level {
	case slog.LevelDebug:
		if h.useEmoji {
			prefix = emojiDebug + " "
		}
		if h.useColors {
			prefix += colorBlue + "DEBUG" + colorReset
		} else if !h.useEmoji {
			prefix = "DEBUG"
		}
	case slog.LevelInfo:
		if h.useEmoji {
			prefix = emojiInfo + " "
		}
		if h.useColors {
			prefix += colorGreen + "INFO" + colorReset
		} else if !h.useEmoji {
			prefix = "INFO"
		}
	case slog.LevelWarn:
		if h.useEmoji {
			prefix = emojiWarn + " "
		}
		if h.useColors {
			prefix += colorYellow + "WARN" + colorReset
		} else if !h.useEmoji {
			prefix = "WARN"
		}
	case slog.LevelError:
		if h.useEmoji {
			prefix = emojiError + " "
		}
		if h.useColors {
			prefix += colorRed + colorBold + "ERROR" + colorReset
		} else if !h.useEmoji {
			prefix = "ERROR"
		}
	}

	msg := r.Message
	if h.useColors && r.Level == slog.LevelError {
		msg = colorBold + msg + colorReset
	}

	_, err := fmt.Fprintf(h.w, "%s %s\n", prefix, msg)
	if err != nil {
		// We can't do much with a logging error except note it
		fmt.Fprintf(os.Stderr, "Error writing log: %v\n", err)
	}
	return nil
}

// WithAttrs implements slog.Handler.
func (h *SimpleHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	// For simplicity, we ignore attributes
	return h
}

// WithGroup implements slog.Handler.
func (h *SimpleHandler) WithGroup(name string) slog.Handler {
	// For simplicity, we ignore groups
	return h
}

// NewColorfulLogger creates a new human-friendly logger without timestamps
func NewColorfulLogger(w io.Writer, level LogLevel, useColors, useEmoji bool) *Logger {
	// Map our log level to slog level
	var slogLevel slog.Level
	switch level {
	case LogLevelDebug:
		slogLevel = slog.LevelDebug
	case LogLevelInfo:
		slogLevel = slog.LevelInfo
	case LogLevelWarn:
		slogLevel = slog.LevelWarn
	case LogLevelError, LogLevelQuiet:
		slogLevel = slog.LevelError
	default:
		slogLevel = slog.LevelInfo
	}

	// Create our custom handler
	handler := &SimpleHandler{
		w:         w,
		level:     slogLevel,
		useColors: useColors,
		useEmoji:  useEmoji,
	}

	slogger := slog.New(handler)
	return &Logger{
		slogger: slogger,
		level:   level,
	}
}
