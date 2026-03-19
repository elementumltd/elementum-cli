// Copyright (c) ElementumAI, Inc.
// SPDX-License-Identifier: MPL-2.0

package client

import (
	"context"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// AsyncTaskPollerConfig configures the polling behavior for async tasks.
type AsyncTaskPollerConfig struct {
	InitialDelay time.Duration // Initial delay between polls
	MaxDelay     time.Duration // Maximum delay between polls
	BackoffRate  float64       // Multiplier for exponential backoff
	Timeout      time.Duration // Maximum time to wait for completion
	JitterFrac   float64       // Fraction of delay to add as jitter (0.0-0.5)
}

// DefaultAsyncTaskPollerConfig returns sensible defaults for async task polling.
func DefaultAsyncTaskPollerConfig() AsyncTaskPollerConfig {
	return AsyncTaskPollerConfig{
		InitialDelay: 1 * time.Second,
		MaxDelay:     5 * time.Second,
		BackoffRate:  1.5,
		Timeout:      1 * time.Minute,
		JitterFrac:   0.2, // +/- 20%
	}
}

// AsyncTaskResult contains the result of waiting for an async task.
type AsyncTaskResult struct {
	TaskID        string   // The async task ID
	Status        string   // Final status (COMPLETE, ERROR, etc.)
	SearchTableID string   // The created search table ID (if successful)
	FieldID       string   // The field ID of the search table
	FieldName     string   // The field name of the search table
	Errors        []string // Any errors from the task
}

// WaitForAsyncTask polls the async task endpoint until the task completes or times out.
// It uses exponential backoff with jitter to avoid thundering herd.
func WaitForAsyncTask(ctx context.Context, client *Client, taskID string, config AsyncTaskPollerConfig) (*AsyncTaskResult, error) {
	delay := config.InitialDelay
	deadline := time.Now().Add(config.Timeout)

	for {
		// Check context cancellation
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("context cancelled while waiting for async task %s: %w", taskID, ctx.Err())
		default:
		}

		// Check timeout
		if time.Now().After(deadline) {
			return nil, fmt.Errorf("timeout waiting for async task %s after %v", taskID, config.Timeout)
		}

		// Poll the async task
		result, err := pollAsyncTask(ctx, client, taskID)
		if err != nil {
			// Log transient errors but continue polling
			tflog.Warn(ctx, "Transient error polling async task", map[string]interface{}{
				"task_id": taskID,
				"error":   err.Error(),
			})
		} else if result != nil {
			// Check if task is complete
			switch result.Status {
			case "COMPLETE":
				tflog.Debug(ctx, "Async task completed successfully", map[string]interface{}{
					"task_id":         taskID,
					"search_table_id": result.SearchTableID,
				})
				return result, nil

			case "ERROR":
				errMsg := "async task failed"
				if len(result.Errors) > 0 {
					errMsg = fmt.Sprintf("async task failed: %s", strings.Join(result.Errors, "; "))
				}
				return result, fmt.Errorf("%s", errMsg)

			case "CREATED", "IN_PROGRESS":
				// Still processing, continue polling
				tflog.Debug(ctx, "Async task still in progress", map[string]interface{}{
					"task_id": taskID,
					"status":  result.Status,
				})

			default:
				tflog.Debug(ctx, "Async task has unexpected status", map[string]interface{}{
					"task_id": taskID,
					"status":  result.Status,
				})
			}
		}

		// Wait before next poll with jitter
		jitteredDelay := addJitter(delay, config.JitterFrac)
		time.Sleep(jitteredDelay)

		// Exponential backoff
		delay = time.Duration(float64(delay) * config.BackoffRate)
		if delay > config.MaxDelay {
			delay = config.MaxDelay
		}
	}
}

// pollAsyncTask queries the async task by ID and extracts the result.
func pollAsyncTask(ctx context.Context, client *Client, taskID string) (*AsyncTaskResult, error) {
	resp, err := GetAsyncTask(ctx, client.Genqlient(), taskID)
	if err != nil {
		return nil, err
	}

	taskPtr := resp.GetMe().AsyncTask
	if taskPtr == nil {
		return nil, fmt.Errorf("async task %s not found", taskID)
	}

	// Dereference the pointer to interface
	task := *taskPtr

	result := &AsyncTaskResult{
		TaskID: task.GetId(),
		Status: string(task.GetStatus()),
		Errors: task.GetErrors(),
	}

	// Extract search table ID based on task type
	switch t := task.(type) {
	case *GetAsyncTaskMeUserAsyncTaskAsyncTaskSearchTableCreate:
		if t.SearchTable != nil {
			st := *t.SearchTable
			result.SearchTableID = st.GetId()
			result.FieldID = st.GetField().GetId()
			result.FieldName = st.GetField().GetName()
		}
	case *GetAsyncTaskMeUserAsyncTaskAsyncTaskTableSearchTableCreate:
		if t.SearchTable != nil {
			st := *t.SearchTable
			result.SearchTableID = st.GetId()
			result.FieldID = st.GetField().GetId()
			result.FieldName = st.GetField().GetName()
		}
	}

	return result, nil
}

// addJitter adds random jitter to a duration.
// jitterFrac should be between 0.0 and 0.5 for reasonable behavior.
func addJitter(d time.Duration, jitterFrac float64) time.Duration {
	if jitterFrac <= 0 {
		return d
	}
	if jitterFrac > 0.5 {
		jitterFrac = 0.5
	}
	// Add jitter in range [-jitterFrac, +jitterFrac] of the duration
	jitter := (rand.Float64()*2 - 1) * jitterFrac * float64(d)
	return d + time.Duration(jitter)
}
