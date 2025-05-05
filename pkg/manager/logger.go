package manager

import (
	"fmt"
	"io"
	"log/slog"
	"os"
)

// LogLevel represents the logging level
type LogLevel int

const (
	// LogLevelDebug represents debug level logging (most verbose)
	LogLevelDebug LogLevel = iota
	// LogLevelInfo represents info level logging (normal operations)
	LogLevelInfo
	// LogLevelWarn represents warning level logging
	LogLevelWarn
	// LogLevelError represents error level logging
	LogLevelError
	// LogLevelQuiet represents minimal logging (only errors and important messages)
	LogLevelQuiet
)

// LogFormat represents the logging output format
type LogFormat int

const (
	// LogFormatDefault uses emoji format if output is to a TTY, otherwise Go format
	LogFormatDefault LogFormat = iota
	// LogFormatGo uses standard Go log format with timestamps
	LogFormatGo
	// LogFormatEmoji uses emoji with colors for log prefixes
	LogFormatEmoji
	// LogFormatColor uses colored text without emoji
	LogFormatColor
	// LogFormatASCII uses plain text without colors or emoji
	LogFormatASCII
)

// Logger is a wrapper around slog to provide consistent logging across the application
type Logger struct {
	slogger *slog.Logger
	level   LogLevel
}

// DefaultLogger is the package-level logger
var DefaultLogger = NewLogger(os.Stdout, LogLevelInfo)

// NewLogger creates a new Logger instance
func NewLogger(w io.Writer, level LogLevel) *Logger {
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

	opts := &slog.HandlerOptions{
		Level: slogLevel,
	}

	handler := slog.NewTextHandler(w, opts)
	slogger := slog.New(handler)

	return &Logger{
		slogger: slogger,
		level:   level,
	}
}

// SetLevel changes the logging level
func (l *Logger) SetLevel(level LogLevel) {
	l.level = level

	// Reset the slog handler with the new level
	var slogLevel slog.Level
	switch level {
	case LogLevelDebug:
		slogLevel = slog.LevelDebug
	case LogLevelInfo:
		slogLevel = slog.LevelWarn
	case LogLevelWarn:
		slogLevel = slog.LevelWarn
	case LogLevelError, LogLevelQuiet:
		slogLevel = slog.LevelError
	default:
		slogLevel = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{
		Level: slogLevel,
	}

	handler := slog.NewTextHandler(os.Stdout, opts)
	l.slogger = slog.New(handler)
}

// Debug logs a debug message
func (l *Logger) Debug(msg string, args ...interface{}) {
	if l.level <= LogLevelDebug {
		l.slogger.Debug(msg, convertArgsToAttrs(args)...)
	}
}

// Info logs an info message
func (l *Logger) Info(msg string, args ...interface{}) {
	if l.level <= LogLevelInfo {
		l.slogger.Info(msg, convertArgsToAttrs(args)...)
	}
}

// Warn logs a warning message
func (l *Logger) Warn(msg string, args ...interface{}) {
	if l.level <= LogLevelWarn {
		l.slogger.Warn(msg, convertArgsToAttrs(args)...)
	}
}

// Error logs an error message
func (l *Logger) Error(msg string, args ...interface{}) {
	if l.level <= LogLevelError {
		l.slogger.Error(msg, convertArgsToAttrs(args)...)
	}
}

// Debugf logs a formatted debug message
func (l *Logger) Debugf(format string, args ...interface{}) {
	if l.level <= LogLevelDebug {
		msg := fmt.Sprintf(format, args...)
		l.slogger.Debug(msg)
	}
}

// Infof logs a formatted info message
func (l *Logger) Infof(format string, args ...interface{}) {
	if l.level <= LogLevelInfo {
		msg := fmt.Sprintf(format, args...)
		l.slogger.Info(msg)
	}
}

// Warnf logs a formatted warning message
func (l *Logger) Warnf(format string, args ...interface{}) {
	if l.level <= LogLevelWarn {
		msg := fmt.Sprintf(format, args...)
		l.slogger.Warn(msg)
	}
}

// Errorf logs a formatted error message
func (l *Logger) Errorf(format string, args ...interface{}) {
	if l.level <= LogLevelError {
		msg := fmt.Sprintf(format, args...)
		l.slogger.Error(msg)
	}
}

// Importantf logs a formatted important message that is always shown regardless of log level
func (l *Logger) Importantf(format string, args ...interface{}) {
	// Important messages are logged at Error level to ensure they are displayed
	msg := fmt.Sprintf(format, args...)
	l.slogger.Error(msg)
}

// Helper function to convert args to slog attributes
func convertArgsToAttrs(args []interface{}) []any {
	if len(args) == 0 {
		return nil
	}

	// For simple usage, just return the args
	return args
}

// isTerminal reports whether the file descriptor is connected to a terminal
func isTerminal(fd uintptr) bool {
	// A simple check - real terminals usually have a non-zero size
	fileInfo, err := os.Stdout.Stat()
	if err != nil {
		return false
	}

	// If it's a character device, it's likely a terminal
	return (fileInfo.Mode() & os.ModeCharDevice) != 0
}

// SetupDefaultLogger initializes the default logger with the specified level and format
func SetupDefaultLogger(level LogLevel, format ...LogFormat) {
	// Determine which format to use
	logFormat := LogFormatDefault
	if len(format) > 0 {
		logFormat = format[0]
	}

	// If format is Default, determine based on terminal detection
	if logFormat == LogFormatDefault {
		if isTerminal(os.Stdout.Fd()) {
			// Connected to a terminal, use emoji format by default
			logFormat = LogFormatEmoji
		} else {
			// Not connected to a terminal, use standard Go format
			logFormat = LogFormatGo
		}
	}

	// Create the logger based on format
	switch logFormat {
	case LogFormatGo:
		// Standard Go format with timestamps
		DefaultLogger = NewLogger(os.Stdout, level)
	case LogFormatEmoji:
		// Emoji format with colors if not disabled
		DefaultLogger = NewColorfulLogger(os.Stdout, level, false, true)
	case LogFormatColor:
		// Colored format without emoji
		DefaultLogger = NewColorfulLogger(os.Stdout, level, true, false)
	case LogFormatASCII:
		// Plain text format without colors or emoji
		DefaultLogger = NewColorfulLogger(os.Stdout, level, false, false)
	default:
		// Fall back to debug logger if all else fails
		DefaultLogger = NewLogger(os.Stdout, level)
	}
}

// GetDefaultLogger returns the default logger
func GetDefaultLogger() *Logger {
	return DefaultLogger
}
