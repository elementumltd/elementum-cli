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

package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/elementumltd/elementum-cli/auth"
	"github.com/elementumltd/elementum-cli/internal/client"
	"github.com/elementumltd/elementum-cli/ui"
	"github.com/spf13/cobra"
)

var automationRunCmd = &cobra.Command{
	Use:   "run <automation-id>",
	Short: "Run an on-demand or webhook automation",
	Long: `Triggers an automation with webhook or on-demand trigger.

This command supports two execution modes:

1. Webhook mode (default): Triggers via the automation's webhook URL
2. Widget mode: Triggers via a run automation widget on a specific record

Examples:
  # Run via webhook and wait for completion (default)
  ei automation run abc-123-uuid

  # Run without waiting
  ei automation run abc-123-uuid --no-wait

  # Run via widget on a specific record
  ei automation run abc-123-uuid --widget def-456 --record "app-id:REC-001"

  # JSON output
  ei automation run abc-123-uuid --json`,
	Args: cobra.ExactArgs(1),
	RunE: runAutomation,
}

func init() {
	automationRunCmd.Flags().Bool("no-wait", false, "Don't wait for execution to complete")
	automationRunCmd.Flags().String("widget", "", "Widget ID for on-demand execution (requires --record)")
	automationRunCmd.Flags().String("record", "", "Record ID in format 'aspectID:handle' (requires --widget)")
	automationsCmd.AddCommand(automationRunCmd)
}

func runAutomation(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	automationID := args[0]

	c, err := auth.GetClientFromCmd(cmd)
	if err != nil {
		return err
	}

	widgetID, _ := cmd.Flags().GetString("widget")
	recordID, _ := cmd.Flags().GetString("record")

	// Validate widget and record flags are used together
	if (widgetID != "" && recordID == "") || (widgetID == "" && recordID != "") {
		return fmt.Errorf("--widget and --record must be used together")
	}

	// Use widget execution path if widget is provided
	if widgetID != "" {
		return runWidgetAutomation(cmd, ctx, c, automationID, widgetID, recordID)
	}

	noWait, _ := cmd.Flags().GetBool("no-wait")
	wait := !noWait

	if !isJSONOutput(cmd) {
		fmt.Println(ui.InfoStyle.Render("⠿ Triggering automation..."))
	}

	// Trigger the webhook
	result, err := triggerAutomationWebhook(ctx, c, automationID)
	if err != nil {
		return err
	}

	if !result.Started {
		return fmt.Errorf("automation failed to start: %s", result.FailureReason)
	}

	executionID := result.ExecutionID

	if isJSONOutput(cmd) && !wait {
		return outputJSON(map[string]any{
			"automation_id": automationID,
			"execution_id":  executionID,
			"started":       true,
		})
	}

	if !wait {
		fmt.Printf("%s Execution started: %s\n", ui.SuccessStyle.Render("✓"), executionID)
		fmt.Printf("  Track with: ei automation status %s %s\n", automationID, executionID)
		return nil
	}

	// Wait for completion
	if !isJSONOutput(cmd) {
		fmt.Printf("%s Started execution %s\n", ui.SuccessStyle.Render("✓"), ui.Truncate(executionID, 12))
		fmt.Println(ui.InfoStyle.Render("⠿ Waiting for completion..."))
	}

	finalStatus, duration, errors, err := waitForExecution(ctx, c, automationID, executionID)
	if err != nil {
		return fmt.Errorf("error waiting for execution: %w", err)
	}

	if isJSONOutput(cmd) {
		return outputJSON(map[string]any{
			"automation_id": automationID,
			"execution_id":  executionID,
			"status":        finalStatus,
			"duration_ms":   duration,
			"errors":        errors,
		})
	}

	// Human-readable output
	statusDisplay := styledStatus(finalStatus)
	durationDisplay := formatDuration(duration)

	fmt.Printf("\n%s %s in %s\n", statusDisplay, executionID[:12], durationDisplay)
	if len(errors) > 0 {
		fmt.Printf("  %s %s\n", ui.ErrorStyle.Render("Errors:"), errors[0])
	}

	// Show detail command
	fmt.Printf("\n  Details: ei automation status %s %s\n", automationID, executionID)

	return nil
}

type webhookResult struct {
	ExecutionID   string
	Started       bool
	FailureReason string
}

func triggerAutomationWebhook(ctx context.Context, c *client.Client, automationID string) (*webhookResult, error) {
	// Get auth token
	token, err := c.GetAccessToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get access token: %w", err)
	}

	// Construct webhook URL
	webhookURL := fmt.Sprintf("%s/api/v1/webhooks/%s", c.GetRESTBaseURL(), automationID)

	// Make POST request to webhook
	body := []byte("{}")
	req, err := http.NewRequestWithContext(ctx, "POST", webhookURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	httpClient := &http.Client{Timeout: 30 * time.Second}
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to trigger webhook: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("webhook returned status %d: %s", resp.StatusCode, string(respBody))
	}

	// Parse response
	var webhookResp struct {
		WorkflowExecutionID string  `json:"workflowExecutionId"`
		Started             bool    `json:"started"`
		FailureReason       *string `json:"failureReason"`
	}
	if err := json.Unmarshal(respBody, &webhookResp); err != nil {
		return nil, fmt.Errorf("failed to parse webhook response: %w", err)
	}

	result := &webhookResult{
		ExecutionID: webhookResp.WorkflowExecutionID,
		Started:     webhookResp.Started,
	}
	if webhookResp.FailureReason != nil {
		result.FailureReason = *webhookResp.FailureReason
	}

	return result, nil
}

func waitForExecution(ctx context.Context, c *client.Client, automationID, executionID string) (status string, durationMs int, errors []string, err error) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	timeout := time.After(5 * time.Minute)

	for {
		select {
		case <-ctx.Done():
			return "", 0, nil, ctx.Err()
		case <-timeout:
			return "TIMEOUT", 0, nil, fmt.Errorf("execution did not complete within 5 minutes")
		case <-ticker.C:
			resp, err := client.GetAutomationExecution(ctx, c.Genqlient(), automationID, executionID)
			if err != nil {
				continue // Transient error, keep polling
			}

			if resp.Organization.Automation == nil {
				return "", 0, nil, fmt.Errorf("automation not found")
			}

			exec := resp.Organization.Automation.Execution
			execStatus := string(exec.Status)

			// Check if terminal state
			switch execStatus {
			case "SUCCESS", "FAILURE", "CANCELLED":
				return execStatus, exec.Duration, exec.Errors, nil
			}
			// Still running, continue polling
		}
	}
}

// runWidgetAutomation executes an automation via a run automation widget
func runWidgetAutomation(cmd *cobra.Command, ctx context.Context, c *client.Client, automationID, widgetID, recordID string) error {
	noWait, _ := cmd.Flags().GetBool("no-wait")
	wait := !noWait

	if !isJSONOutput(cmd) {
		fmt.Println(ui.InfoStyle.Render("⠿ Triggering automation via widget..."))
	}

	// Execute the automation via the widget
	input := client.DisplayWidgetRunAutomationExecuteInput{
		DisplayWidgetId: widgetID,
		RecordId:        recordID,
		Parameters:      []client.DisplayWidgetActionExecuteParameterInput{}, // Empty for now
	}

	result, err := client.ExecuteWidgetAutomation(ctx, c.Genqlient(), input)
	if err != nil {
		return fmt.Errorf("failed to execute automation: %w", err)
	}

	executionID := result.DisplayWidgetRunAutomationExecute.Id
	initialStatus := string(result.DisplayWidgetRunAutomationExecute.Status)

	if isJSONOutput(cmd) && !wait {
		return outputJSON(map[string]any{
			"automation_id": automationID,
			"widget_id":     widgetID,
			"record_id":     recordID,
			"execution_id":  executionID,
			"status":        initialStatus,
		})
	}

	if !wait {
		fmt.Printf("%s Execution started: %s\n", ui.SuccessStyle.Render("✓"), executionID)
		fmt.Printf("  Track with: ei automation status %s %s\n", automationID, executionID)
		return nil
	}

	// Wait for completion
	if !isJSONOutput(cmd) {
		fmt.Printf("%s Started execution %s\n", ui.SuccessStyle.Render("✓"), ui.Truncate(executionID, 12))
		fmt.Println(ui.InfoStyle.Render("⠿ Waiting for completion..."))
	}

	// Extract aspect ID from record ID (format: aspectID:handle)
	aspectID := extractAspectFromRecordID(recordID)

	finalStatus, err := waitForWidgetExecution(ctx, c, aspectID, widgetID, recordID)
	if err != nil {
		return fmt.Errorf("error waiting for execution: %w", err)
	}

	if isJSONOutput(cmd) {
		return outputJSON(map[string]any{
			"automation_id": automationID,
			"widget_id":     widgetID,
			"record_id":     recordID,
			"execution_id":  executionID,
			"status":        finalStatus,
		})
	}

	// Human-readable output
	statusDisplay := styledStatus(finalStatus)
	fmt.Printf("\n%s %s\n", statusDisplay, executionID[:12])

	// Show detail command
	fmt.Printf("\n  Details: ei automation status %s %s\n", automationID, executionID)

	return nil
}

// extractAspectFromRecordID extracts the aspect ID from a record ID (format: aspectID:handle)
func extractAspectFromRecordID(recordID string) string {
	for i, c := range recordID {
		if c == ':' {
			return recordID[:i]
		}
	}
	return recordID // Return as-is if no colon found
}

// waitForWidgetExecution polls the widget automation tracking for completion
func waitForWidgetExecution(ctx context.Context, c *client.Client, aspectID, widgetID, recordID string) (status string, err error) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	timeout := time.After(5 * time.Minute)

	for {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-timeout:
			return "TIMEOUT", fmt.Errorf("execution did not complete within 5 minutes")
		case <-ticker.C:
			resp, err := client.GetWidgetAutomationTracking(ctx, c.Genqlient(), aspectID, widgetID, recordID)
			if err != nil {
				continue // Transient error, keep polling
			}

			aspectPtr := resp.Organization.GetAspect()
			if aspectPtr == nil {
				continue
			}
			aspect := *aspectPtr

			widget := aspect.GetDisplayWidget()
			if widget == nil {
				continue
			}

			// Type assert to get the run automation widget
			runWidget, ok := widget.(*client.GetWidgetAutomationTrackingOrganizationAspectDisplayWidgetDisplayWidgetRunAutomationAction)
			if !ok {
				continue
			}

			tracking := runWidget.GetAutomationTracking()
			if tracking == nil || tracking.WorkflowExecution == nil {
				continue // Not started yet
			}

			execStatus := string(tracking.WorkflowExecution.Status)

			// Check if terminal state
			switch execStatus {
			case "SUCCESS", "FAILURE", "CANCELLED":
				return execStatus, nil
			}
			// Still running, continue polling
		}
	}
}
