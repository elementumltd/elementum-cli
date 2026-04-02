// Copyright 2026 Elementum Ltd. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package logger provides structured logging for the elementum CLI using slog.
// Logs go to stderr so they don't interfere with command output.
package logger

import (
	"io"
	"log/slog"
	"os"
	"strings"

	"github.com/elementumltd/elementum-cli/internal/logging"
)

// Level represents the logging level for the CLI.
// These match Terraform's log levels for familiarity.
type Level string

const (
	LevelTrace Level = "trace"
	LevelDebug Level = "debug"
	LevelInfo  Level = "info"
	LevelWarn  Level = "warn"
	LevelError Level = "error"
	LevelOff   Level = "off"
)

var logger *slog.Logger
var currentLevel Level = LevelInfo

// ParseLevel converts a string to a Level, defaulting to LevelInfo for invalid input.
func ParseLevel(s string) Level {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "trace":
		return LevelTrace
	case "debug":
		return LevelDebug
	case "info":
		return LevelInfo
	case "warn", "warning":
		return LevelWarn
	case "error":
		return LevelError
	case "off":
		return LevelOff
	default:
		return LevelInfo
	}
}

// GetTFLogLevel returns the appropriate TF_LOG value for the given Level.
// Returns empty string for levels that shouldn't set TF_LOG.
func GetTFLogLevel(level Level) string {
	switch level {
	case LevelTrace:
		return "TRACE"
	case LevelDebug:
		return "DEBUG"
	case LevelWarn:
		return "WARN"
	case LevelError:
		return "ERROR"
	default:
		// info and off don't set TF_LOG
		return ""
	}
}

// GetCurrentLevel returns the current logging level.
func GetCurrentLevel() Level {
	return currentLevel
}

// Init initializes the global logger with the specified level.
// Call this early in main() before any logging occurs.
// This also configures the shared internal/logging package.
func Init(level Level) {
	currentLevel = level

	var slogLevel slog.Level
	var handler slog.Handler

	switch level {
	case LevelTrace, LevelDebug:
		slogLevel = slog.LevelDebug
	case LevelInfo:
		slogLevel = slog.LevelInfo
	case LevelWarn:
		slogLevel = slog.LevelWarn
	case LevelError:
		slogLevel = slog.LevelError
	case LevelOff:
		// Use a no-op handler that discards all logs
		handler = slog.NewTextHandler(io.Discard, nil)
		logger = slog.New(handler)
		// Disable warnings in the shared logging package
		logging.SetWarnEnabled(false)
		return
	default:
		slogLevel = slog.LevelInfo
	}

	handler = slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slogLevel,
	})
	logger = slog.New(handler)

	// Configure the shared logging package to route through our logger.
	// Enable warnings only at warn level or more verbose (trace, debug, info, warn).
	warnEnabled := level == LevelTrace || level == LevelDebug || level == LevelInfo || level == LevelWarn
	logging.SetWarnEnabled(warnEnabled)
	logging.SetWarnFunc(func(msg string, args ...any) {
		ensureLogger().Warn(msg, args...)
	})
	// Route demoted GraphQL warnings to debug level so they only show with --debug
	logging.SetDebugFunc(func(msg string, args ...any) {
		ensureLogger().Debug(msg, args...)
	})
}

// Default returns a no-op logger if Init hasn't been called yet.
// This prevents nil pointer panics during early startup.
func ensureLogger() *slog.Logger {
	if logger == nil {
		// Default to info level if not initialized
		handler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		})
		logger = slog.New(handler)
	}
	return logger
}

// Debug logs at debug level - only visible when --debug flag is set.
// Use for detailed troubleshooting information.
func Debug(msg string, args ...any) {
	ensureLogger().Debug(msg, args...)
}

// Info logs at info level - normal operational messages.
func Info(msg string, args ...any) {
	ensureLogger().Info(msg, args...)
}

// Warn logs at warn level - non-fatal issues that should be investigated.
// Use this instead of swallowing errors with _ = err
func Warn(msg string, args ...any) {
	ensureLogger().Warn(msg, args...)
}

// Error logs at error level - serious issues that affect functionality.
func Error(msg string, args ...any) {
	ensureLogger().Error(msg, args...)
}

// With returns a logger with additional context fields.
// Useful for adding consistent fields across multiple log calls.
func With(args ...any) *slog.Logger {
	return ensureLogger().With(args...)
}
