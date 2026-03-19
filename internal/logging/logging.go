// Copyright (c) ElementumAI, Inc.
// SPDX-License-Identifier: MPL-2.0

// Package logging provides a unified logging interface for both CLI and Terraform provider.
// By default, warnings are suppressed. Call SetWarnEnabled(true) to enable them.
// The CLI's logger.Init() automatically configures this package.
package logging

import (
	"fmt"
	"log"
	"strings"
	"sync"
)

// WarnFunc is the function signature for warning handlers.
// msg is the message, keyvals are slog-style key-value pairs.
type WarnFunc func(msg string, keyvals ...any)

var (
	mu          sync.RWMutex
	warnEnabled bool
	warnFunc    WarnFunc = defaultWarn
	debugFunc   WarnFunc // optional debug-level handler for demoted warnings
)

// defaultWarn logs to stderr using the standard library logger.
func defaultWarn(msg string, keyvals ...any) {
	// Format key-value pairs for log.Printf
	if len(keyvals) > 0 {
		log.Printf("[WARN] %s %v", msg, keyvals)
	} else {
		log.Printf("[WARN] %s", msg)
	}
}

// SetWarnEnabled enables or disables warning output.
// When disabled (default), Warn calls are no-ops.
func SetWarnEnabled(enabled bool) {
	mu.Lock()
	defer mu.Unlock()
	warnEnabled = enabled
}

// SetWarnFunc sets a custom warning function.
// This is used by the CLI logger to route warnings through slog.
func SetWarnFunc(fn WarnFunc) {
	mu.Lock()
	defer mu.Unlock()
	warnFunc = fn
}

// SetDebugFunc sets a debug-level handler for demoted warnings.
// GraphQL partial-data warnings that are non-actionable get routed here
// instead of to the warn handler, so they only show at debug log level.
func SetDebugFunc(fn WarnFunc) {
	mu.Lock()
	defer mu.Unlock()
	debugFunc = fn
}

// Warn logs a warning message if warnings are enabled.
// keyvals are slog-style key-value pairs (e.g., "key1", value1, "key2", value2).
func Warn(msg string, keyvals ...any) {
	mu.RLock()
	enabled := warnEnabled
	fn := warnFunc
	mu.RUnlock()

	if enabled && fn != nil {
		fn(msg, keyvals...)
	}
}

// isNonActionableGraphQLError returns true for GraphQL errors that are expected
// data-quality issues in the platform (orphaned references, deleted items).
// These are demoted to debug level since the partial data is still valid.
func isNonActionableGraphQLError(message string) bool {
	return strings.Contains(message, "The item does not exist") ||
		strings.Contains(message, "An unexpected error has occurred")
}

// GraphQLWarn logs a GraphQL-specific warning with operation context.
// Known non-actionable errors (orphaned references, server-side data issues)
// are demoted to debug level to avoid noisy output during exports.
func GraphQLWarn(message, operation string, path any, variables map[string]any) {
	msg := fmt.Sprintf("GraphQL error: %s", message)
	args := []any{
		"operation", operation,
		"path", path,
		"variables", variables,
	}

	if isNonActionableGraphQLError(message) {
		// Demote to debug level — these are expected data-quality issues
		mu.RLock()
		fn := debugFunc
		mu.RUnlock()
		if fn != nil {
			fn(msg, args...)
		}
		return
	}

	Warn(msg, args...)
}
