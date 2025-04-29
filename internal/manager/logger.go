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

// Important logs an important message that is always shown regardless of log level
func (l *Logger) Important(msg string, args ...interface{}) {
	// Always show important messages
	l.slogger.Error(msg, convertArgsToAttrs(args)...)
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

// SetupDefaultLogger initializes the default logger with the specified level
func SetupDefaultLogger(level LogLevel) {
	DefaultLogger = NewLogger(os.Stdout, level)
}

// GetDefaultLogger returns the default logger
func GetDefaultLogger() *Logger {
	return DefaultLogger
}
