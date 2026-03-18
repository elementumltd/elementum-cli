// Copyright (c) ElementumAI, Inc.
// SPDX-License-Identifier: MPL-2.0

package client

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"net"
	"net/http"
	"time"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/vektah/gqlparser/v2/gqlerror"
)

const (
	maxRetries     = 3
	initialDelay   = 200 * time.Millisecond
	maxDelay       = 10 * time.Second
	backoffFactor  = 2.0
	jitterFraction = 0.2 // +/- 20% jitter
)

// retryableGraphQLErrorTypes are GraphQL error codes that indicate a transient
// server-side failure worth retrying. These arrive as HTTP 200 responses with
// errors in the body, so the HTTP-level retry logic never sees them.
var retryableGraphQLErrorTypes = map[string]bool{
	"INTERNAL":     true,
	"THROTTLED":    true,
	"UNAVAILABLE":  true,
	"SERVICE_DOWN": true,
}

// isRetryableGraphQLError checks whether a gqlerror indicates a transient
// server error that should be retried.
func isRetryableGraphQLError(err *gqlerror.Error) bool {
	if err == nil {
		return false
	}
	if err.Extensions == nil {
		return false
	}
	errorType, ok := err.Extensions["errorType"].(string)
	if !ok {
		return false
	}
	return retryableGraphQLErrorTypes[errorType]
}

// isRetryableGraphQLErrorLegacy checks the legacy GraphQLError struct.
func isRetryableGraphQLErrorLegacy(err GraphQLError) bool {
	if err.Extensions == nil {
		return false
	}
	errorType, ok := err.Extensions["errorType"].(string)
	if !ok {
		return false
	}
	return retryableGraphQLErrorTypes[errorType]
}

// graphQLRetryBackoff sleeps for the appropriate backoff duration,
// respecting context cancellation. Returns false if the context was cancelled.
func graphQLRetryBackoff(ctx context.Context, attempt int, opName string) bool {
	delay := calculateBackoff(attempt)
	tflog.Warn(ctx, "Retrying after transient GraphQL error", map[string]interface{}{
		"attempt":   attempt + 1,
		"max_retry": maxRetries,
		"delay_ms":  delay.Milliseconds(),
		"operation": opName,
	})
	select {
	case <-ctx.Done():
		return false
	case <-time.After(delay):
		return true
	}
}

// RetryableHTTPFunc is a function that performs an HTTP request and can be retried.
type RetryableHTTPFunc func() (*http.Response, error)

// isRetryableError determines if an error should trigger a retry.
func isRetryableError(err error) bool {
	// Network timeouts
	if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
		return true
	}
	// Connection errors
	if _, ok := err.(*net.OpError); ok {
		return true
	}
	return false
}

// isRetryableStatusCode determines if an HTTP status code should trigger a retry.
func isRetryableStatusCode(statusCode int) bool {
	switch statusCode {
	case http.StatusTooManyRequests, // 429
		http.StatusInternalServerError, // 500
		http.StatusBadGateway,          // 502
		http.StatusServiceUnavailable,  // 503
		http.StatusGatewayTimeout:      // 504
		return true
	}
	return false
}

// calculateBackoff returns the delay with jitter for a given attempt number.
func calculateBackoff(attempt int) time.Duration {
	delay := float64(initialDelay) * math.Pow(backoffFactor, float64(attempt))
	if delay > float64(maxDelay) {
		delay = float64(maxDelay)
	}
	// Add jitter: +/- jitterFraction
	jitter := delay * jitterFraction * (2*rand.Float64() - 1)
	return time.Duration(delay + jitter)
}

// ExecuteWithRetry executes an HTTP request function with exponential backoff retry.
// It retries on network errors and specific HTTP status codes (429, 5xx).
// The function returns the response from the last attempt or an error if all retries fail.
func ExecuteWithRetry(ctx context.Context, fn RetryableHTTPFunc) (*http.Response, error) {
	var lastErr error
	var lastResp *http.Response

	for attempt := 0; attempt <= maxRetries; attempt++ {
		// Wait before retry (not on first attempt)
		if attempt > 0 {
			delay := calculateBackoff(attempt - 1)
			tflog.Debug(ctx, "Retrying GraphQL request", map[string]interface{}{
				"attempt":   attempt,
				"max_retry": maxRetries,
				"delay_ms":  delay.Milliseconds(),
			})

			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(delay):
			}
		}

		resp, err := fn()

		// Handle network/connection errors
		if err != nil {
			if isRetryableError(err) {
				lastErr = err
				tflog.Warn(ctx, "Retryable network error occurred", map[string]interface{}{
					"attempt": attempt,
					"error":   err.Error(),
				})
				continue
			}
			// Non-retryable error, return immediately
			return nil, err
		}

		// Check for retryable HTTP status codes
		if isRetryableStatusCode(resp.StatusCode) {
			lastResp = resp
			lastErr = fmt.Errorf("HTTP %d", resp.StatusCode)
			tflog.Warn(ctx, "Retryable HTTP status code", map[string]interface{}{
				"attempt":     attempt,
				"status_code": resp.StatusCode,
			})
			continue
		}

		// Success - return the response
		return resp, nil
	}

	// All retries exhausted
	if lastResp != nil {
		return lastResp, lastErr
	}
	return nil, fmt.Errorf("max retries (%d) exceeded: %w", maxRetries, lastErr)
}
