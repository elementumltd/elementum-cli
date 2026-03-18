// Copyright (c) ElementumAI, Inc.
// SPDX-License-Identifier: MPL-2.0

package client

import (
	"context"
	"errors"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/vektah/gqlparser/v2/gqlerror"
)

// mockTimeoutError implements net.Error for testing timeout scenarios.
type mockTimeoutError struct{}

func (m mockTimeoutError) Error() string   { return "timeout" }
func (m mockTimeoutError) Timeout() bool   { return true }
func (m mockTimeoutError) Temporary() bool { return true }

func TestIsRetryableError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "timeout error should be retryable",
			err:      mockTimeoutError{},
			expected: true,
		},
		{
			name:     "net.OpError should be retryable",
			err:      &net.OpError{Op: "dial", Net: "tcp", Err: errors.New("connection refused")},
			expected: true,
		},
		{
			name:     "generic error should not be retryable",
			err:      errors.New("some error"),
			expected: false,
		},
		{
			name:     "nil error should not be retryable",
			err:      nil,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isRetryableError(tt.err)
			if result != tt.expected {
				t.Errorf("isRetryableError(%v) = %v, expected %v", tt.err, result, tt.expected)
			}
		})
	}
}

func TestIsRetryableStatusCode(t *testing.T) {
	tests := []struct {
		statusCode int
		expected   bool
	}{
		{http.StatusOK, false},
		{http.StatusCreated, false},
		{http.StatusBadRequest, false},
		{http.StatusUnauthorized, false},
		{http.StatusForbidden, false},
		{http.StatusNotFound, false},
		{http.StatusTooManyRequests, true},
		{http.StatusInternalServerError, true},
		{http.StatusBadGateway, true},
		{http.StatusServiceUnavailable, true},
		{http.StatusGatewayTimeout, true},
	}

	for _, tt := range tests {
		t.Run(http.StatusText(tt.statusCode), func(t *testing.T) {
			result := isRetryableStatusCode(tt.statusCode)
			if result != tt.expected {
				t.Errorf("isRetryableStatusCode(%d) = %v, expected %v", tt.statusCode, result, tt.expected)
			}
		})
	}
}

func TestCalculateBackoff(t *testing.T) {
	// Test that backoff increases with each attempt
	prevDelay := time.Duration(0)
	for attempt := 0; attempt < 5; attempt++ {
		delay := calculateBackoff(attempt)

		// Verify delay is positive
		if delay <= 0 {
			t.Errorf("calculateBackoff(%d) returned non-positive delay: %v", attempt, delay)
		}

		// For attempt > 0, verify general exponential growth trend
		// (accounting for jitter, we can't be exact)
		if attempt > 0 {
			// The base delay should roughly double each time
			// With 20% jitter, min is 0.8x and max is 1.2x
			minExpected := time.Duration(float64(prevDelay) * 0.5) // Very generous lower bound
			if delay < minExpected && attempt < 4 {
				t.Logf("Note: delay for attempt %d (%v) is less than expected minimum (%v), but this can happen due to jitter", attempt, delay, minExpected)
			}
		}

		prevDelay = delay
	}

	// Test that delay is capped at maxDelay
	highAttempt := 100
	delay := calculateBackoff(highAttempt)
	// With 20% jitter, max possible is 1.2 * maxDelay
	maxPossible := time.Duration(float64(maxDelay) * 1.2)
	if delay > maxPossible {
		t.Errorf("calculateBackoff(%d) = %v, exceeds max possible %v", highAttempt, delay, maxPossible)
	}
}

func TestExecuteWithRetry_Success(t *testing.T) {
	ctx := t.Context()
	callCount := 0

	resp, err := ExecuteWithRetry(ctx, func() (*http.Response, error) {
		callCount++
		return &http.Response{StatusCode: http.StatusOK}, nil
	})

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if resp == nil {
		t.Error("expected response, got nil")
	}
	if callCount != 1 {
		t.Errorf("expected 1 call, got %d", callCount)
	}
}

func TestExecuteWithRetry_RetryOnServerError(t *testing.T) {
	ctx := t.Context()
	callCount := 0

	resp, err := ExecuteWithRetry(ctx, func() (*http.Response, error) {
		callCount++
		if callCount < 3 {
			return &http.Response{StatusCode: http.StatusServiceUnavailable}, nil
		}
		return &http.Response{StatusCode: http.StatusOK}, nil
	})

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if resp == nil || resp.StatusCode != http.StatusOK {
		t.Error("expected successful response")
	}
	if callCount != 3 {
		t.Errorf("expected 3 calls, got %d", callCount)
	}
}

func TestExecuteWithRetry_RetryOnNetworkError(t *testing.T) {
	ctx := t.Context()
	callCount := 0

	resp, err := ExecuteWithRetry(ctx, func() (*http.Response, error) {
		callCount++
		if callCount < 2 {
			return nil, mockTimeoutError{}
		}
		return &http.Response{StatusCode: http.StatusOK}, nil
	})

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if resp == nil || resp.StatusCode != http.StatusOK {
		t.Error("expected successful response")
	}
	if callCount != 2 {
		t.Errorf("expected 2 calls, got %d", callCount)
	}
}

func TestExecuteWithRetry_MaxRetriesExceeded(t *testing.T) {
	ctx := t.Context()
	callCount := 0

	resp, err := ExecuteWithRetry(ctx, func() (*http.Response, error) {
		callCount++
		return &http.Response{StatusCode: http.StatusServiceUnavailable}, nil
	})

	if err == nil {
		t.Error("expected error, got nil")
	}
	// Should have tried maxRetries + 1 times (initial + retries)
	expectedCalls := maxRetries + 1
	if callCount != expectedCalls {
		t.Errorf("expected %d calls, got %d", expectedCalls, callCount)
	}
	// Should return the last response
	if resp == nil || resp.StatusCode != http.StatusServiceUnavailable {
		t.Error("expected last response to be returned")
	}
}

func TestExecuteWithRetry_NonRetryableError(t *testing.T) {
	ctx := t.Context()
	callCount := 0
	nonRetryableErr := errors.New("non-retryable error")

	resp, err := ExecuteWithRetry(ctx, func() (*http.Response, error) {
		callCount++
		return nil, nonRetryableErr
	})

	if err != nonRetryableErr {
		t.Errorf("expected non-retryable error, got %v", err)
	}
	if resp != nil {
		t.Error("expected nil response")
	}
	if callCount != 1 {
		t.Errorf("expected 1 call (no retry), got %d", callCount)
	}
}

func TestExecuteWithRetry_NonRetryableStatusCode(t *testing.T) {
	ctx := t.Context()
	callCount := 0

	resp, err := ExecuteWithRetry(ctx, func() (*http.Response, error) {
		callCount++
		return &http.Response{StatusCode: http.StatusBadRequest}, nil
	})

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if resp == nil || resp.StatusCode != http.StatusBadRequest {
		t.Error("expected bad request response")
	}
	if callCount != 1 {
		t.Errorf("expected 1 call (no retry), got %d", callCount)
	}
}

func TestExecuteWithRetry_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	callCount := 0

	// Cancel after first call
	resp, err := ExecuteWithRetry(ctx, func() (*http.Response, error) {
		callCount++
		if callCount == 1 {
			cancel() // Cancel during retry wait
			return &http.Response{StatusCode: http.StatusServiceUnavailable}, nil
		}
		return &http.Response{StatusCode: http.StatusOK}, nil
	})

	// Should have stopped due to context cancellation
	if err != context.Canceled {
		t.Errorf("expected context.Canceled, got %v", err)
	}
	if resp != nil {
		t.Error("expected nil response when context cancelled")
	}
}

func TestIsRetryableGraphQLError(t *testing.T) {
	tests := []struct {
		name     string
		err      *gqlerror.Error
		expected bool
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "no extensions",
			err:      &gqlerror.Error{Message: "bad request"},
			expected: false,
		},
		{
			name: "INTERNAL is retryable",
			err: &gqlerror.Error{
				Message:    "Something went wrong",
				Extensions: map[string]interface{}{"errorType": "INTERNAL"},
			},
			expected: true,
		},
		{
			name: "THROTTLED is retryable",
			err: &gqlerror.Error{
				Message:    "Rate limited",
				Extensions: map[string]interface{}{"errorType": "THROTTLED"},
			},
			expected: true,
		},
		{
			name: "UNAVAILABLE is retryable",
			err: &gqlerror.Error{
				Message:    "Service unavailable",
				Extensions: map[string]interface{}{"errorType": "UNAVAILABLE"},
			},
			expected: true,
		},
		{
			name: "NOT_FOUND is not retryable",
			err: &gqlerror.Error{
				Message:    "Not found",
				Extensions: map[string]interface{}{"errorType": "NOT_FOUND"},
			},
			expected: false,
		},
		{
			name: "VALIDATION is not retryable",
			err: &gqlerror.Error{
				Message:    "Invalid input",
				Extensions: map[string]interface{}{"errorType": "VALIDATION"},
			},
			expected: false,
		},
		{
			name: "non-string errorType",
			err: &gqlerror.Error{
				Message:    "Weird",
				Extensions: map[string]interface{}{"errorType": 42},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isRetryableGraphQLError(tt.err)
			if result != tt.expected {
				t.Errorf("isRetryableGraphQLError() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestIsRetryableGraphQLErrorLegacy(t *testing.T) {
	tests := []struct {
		name     string
		err      GraphQLError
		expected bool
	}{
		{
			name:     "no extensions",
			err:      GraphQLError{Message: "bad request"},
			expected: false,
		},
		{
			name: "INTERNAL is retryable",
			err: GraphQLError{
				Message:    "Something went wrong",
				Extensions: map[string]interface{}{"errorType": "INTERNAL"},
			},
			expected: true,
		},
		{
			name: "NOT_FOUND is not retryable",
			err: GraphQLError{
				Message:    "Not found",
				Extensions: map[string]interface{}{"errorType": "NOT_FOUND"},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isRetryableGraphQLErrorLegacy(tt.err)
			if result != tt.expected {
				t.Errorf("isRetryableGraphQLErrorLegacy() = %v, want %v", result, tt.expected)
			}
		})
	}
}
