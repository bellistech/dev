// Package logger provides a simple leveled logger.
package logger

import (
	"fmt"
	"log"
	"os"
	"strings"
)

// Level represents a log level.
type Level int

const (
	LevelDebug Level = iota
	LevelInfo
	LevelWarn
	LevelError
)

// String returns the string representation of a Level.
func (l Level) String() string {
	switch l {
	case LevelDebug:
		return "DEBUG"
	case LevelInfo:
		return "INFO"
	case LevelWarn:
		return "WARN"
	case LevelError:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

// ParseLevel converts a string to a Level.
func ParseLevel(s string) Level {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "debug":
		return LevelDebug
	case "info":
		return LevelInfo
	case "warn", "warning":
		return LevelWarn
	case "error":
		return LevelError
	default:
		return LevelInfo
	}
}

// Logger is a simple leveled logger.
type Logger struct {
	level  Level
	prefix string
	logger *log.Logger
}

// New creates a new Logger with the specified level.
func New(level Level, prefix string) *Logger {
	flags := log.LstdFlags | log.Lmicroseconds
	return &Logger{
		level:  level,
		prefix: prefix,
		logger: log.New(os.Stderr, "", flags),
	}
}

// NewFromString creates a new Logger parsing the level from a string.
func NewFromString(levelStr, prefix string) *Logger {
	return New(ParseLevel(levelStr), prefix)
}

// SetLevel changes the log level.
func (l *Logger) SetLevel(level Level) {
	l.level = level
}

// GetLevel returns the current log level.
func (l *Logger) GetLevel() Level {
	return l.level
}

// log writes a log message if the level is enabled.
func (l *Logger) log(level Level, format string, args ...interface{}) {
	if level < l.level {
		return
	}

	msg := fmt.Sprintf(format, args...)
	prefix := ""
	if l.prefix != "" {
		prefix = "[" + l.prefix + "] "
	}
	l.logger.Printf("%s%s %s", prefix, level.String(), msg)
}

// Debug logs a debug message.
func (l *Logger) Debug(format string, args ...interface{}) {
	l.log(LevelDebug, format, args...)
}

// Info logs an info message.
func (l *Logger) Info(format string, args ...interface{}) {
	l.log(LevelInfo, format, args...)
}

// Warn logs a warning message.
func (l *Logger) Warn(format string, args ...interface{}) {
	l.log(LevelWarn, format, args...)
}

// Error logs an error message.
func (l *Logger) Error(format string, args ...interface{}) {
	l.log(LevelError, format, args...)
}

// Fatal logs an error message and exits.
func (l *Logger) Fatal(format string, args ...interface{}) {
	l.log(LevelError, format, args...)
	os.Exit(1)
}

// Default logger instance
var std = New(LevelInfo, "")

// SetLevel sets the level of the default logger.
func SetLevel(level Level) {
	std.SetLevel(level)
}

// SetLevelFromString sets the level of the default logger from a string.
func SetLevelFromString(levelStr string) {
	std.SetLevel(ParseLevel(levelStr))
}

// GetLevel returns the level of the default logger.
func GetLevel() Level {
	return std.GetLevel()
}

// Package-level convenience functions using the default logger

// Debug logs a debug message.
func Debug(format string, args ...interface{}) {
	std.Debug(format, args...)
}

// Info logs an info message.
func Info(format string, args ...interface{}) {
	std.Info(format, args...)
}

// Warn logs a warning message.
func Warn(format string, args ...interface{}) {
	std.Warn(format, args...)
}

// Error logs an error message.
func Error(format string, args ...interface{}) {
	std.Error(format, args...)
}

// Fatal logs an error message and exits.
func Fatal(format string, args ...interface{}) {
	std.Fatal(format, args...)
}
