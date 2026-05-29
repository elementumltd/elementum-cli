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
	"context"
	"fmt"
	"time"

	"github.com/elementumltd/elementum-cli/auth"
	"github.com/elementumltd/elementum-cli/internal/client"
	"github.com/elementumltd/elementum-cli/logger"
	"github.com/elementumltd/elementum-cli/ui"
	"github.com/spf13/cobra"
)

const (
	// deploymentPollInterval is the interval between status checks when polling.
	deploymentPollInterval = 3 * time.Second

	// maxConsecutivePollErrors is the number of consecutive API errors before giving up.
	maxConsecutivePollErrors = 5
)

var deploymentsCreateCmd = &cobra.Command{
	Use:   "create <app-namespace>",
	Short: "Create a deployment",
	Long: `Create a new deployment to promote an app from one environment to another.

By default, waits for the deployment to complete (or reach configuration state).
Use --no-wait to return immediately after creation.

Examples:
  ei deployments create my-app --source staging --target production
  ei deployments create my-app --source staging --target production --name "v2.1 release"
  ei deployments create my-app --source staging --target production --no-wait
  ei deployments create my-app --source staging --target production --json`,
	Args: cobra.ExactArgs(1),
	RunE: runDeploymentsCreate,
}

func init() {
	deploymentsCreateCmd.Flags().String("source", "", "Source environment name or ID (required)")
	deploymentsCreateCmd.Flags().String("target", "", "Target environment name or ID (required)")
	deploymentsCreateCmd.Flags().String("name", "", "Deployment name")
	deploymentsCreateCmd.Flags().String("message", "", "Deployment message")
	deploymentsCreateCmd.Flags().Bool("no-wait", false, "Don't wait for deployment to complete")
	deploymentsCreateCmd.Flags().Duration("timeout", 10*time.Minute, "Timeout for waiting (default 10m)")
	_ = deploymentsCreateCmd.MarkFlagRequired("source")
	_ = deploymentsCreateCmd.MarkFlagRequired("target")
}

func runDeploymentsCreate(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	appNamespace := args[0]

	apiClient, err := auth.GetClientFromCmd(cmd)
	if err != nil {
		return err
	}

	source, _ := cmd.Flags().GetString("source")
	target, _ := cmd.Flags().GetString("target")
	name, _ := cmd.Flags().GetString("name")
	message, _ := cmd.Flags().GetString("message")
	noWait, _ := cmd.Flags().GetBool("no-wait")
	timeout, _ := cmd.Flags().GetDuration("timeout")

	if !isJSONOutput(cmd) {
		fmt.Println(ui.InfoStyle.Render("Resolving app and environments..."))
	}

	// Resolve app namespace to ID
	appID, _, err := resolveAspectByNamespace(ctx, apiClient, appNamespace)
	if err != nil {
		return fmt.Errorf("failed to find app %q: %w", appNamespace, err)
	}

	// Resolve source and target environment names to IDs
	sourceID, sourceName, err := resolveEnvironmentByName(ctx, apiClient, source)
	if err != nil {
		return fmt.Errorf("failed to resolve source environment: %w", err)
	}

	targetID, targetName, err := resolveEnvironmentByName(ctx, apiClient, target)
	if err != nil {
		return fmt.Errorf("failed to resolve target environment: %w", err)
	}

	if !isJSONOutput(cmd) {
		fmt.Printf("  App:    %s\n", appNamespace)
		fmt.Printf("  Source: %s\n", sourceName)
		fmt.Printf("  Target: %s\n", targetName)
		fmt.Println()
		fmt.Println(ui.InfoStyle.Render("Creating deployment..."))
	}

	// Build input
	input := client.DeploymentCreateInput{
		AppId:               appID,
		SourceEnvironmentId: sourceID,
		TargetEnvironmentId: targetID,
	}
	if name != "" {
		input.Name = &name
	}
	if message != "" {
		input.Message = &message
	}

	logger.Debug("creating deployment", "app_id", appID, "source", sourceID, "target", targetID)
	resp, err := client.CreateDeployment(ctx, apiClient.Genqlient(), input)
	if err != nil {
		return fmt.Errorf("failed to create deployment: %w", err)
	}
	logger.Debug("deployment created", "id", resp.DeploymentCreate.Id, "status", resp.DeploymentCreate.Status)

	deployment := resp.DeploymentCreate

	if isJSONOutput(cmd) && noWait {
		return outputJSON(map[string]any{
			"id":     deployment.Id,
			"status": deployment.Status,
		})
	}

	if noWait {
		fmt.Printf("%s Deployment created: %s (status: %s)\n",
			ui.SuccessStyle.Render("✓"), deployment.Id, styledDeploymentStatus(string(deployment.Status)))
		fmt.Printf("  Track with: ei deployments show %s\n", deployment.Id)
		return nil
	}

	if !isJSONOutput(cmd) {
		fmt.Printf("%s Deployment created: %s\n", ui.SuccessStyle.Render("✓"), ui.Truncate(deployment.Id, 12))
		fmt.Println(ui.InfoStyle.Render("Waiting for deployment to complete..."))
	}

	// Poll for completion
	finalStatus, errors, err := pollDeploymentStatus(ctx, apiClient, deployment.Id, timeout, cmd)
	if err != nil {
		return err
	}

	if isJSONOutput(cmd) {
		return outputJSON(map[string]any{
			"id":     deployment.Id,
			"status": finalStatus,
			"errors": errors,
		})
	}

	// Human-readable final output
	fmt.Println()
	switch finalStatus {
	case "COMPLETED":
		fmt.Printf("%s Deployment completed successfully!\n", ui.SuccessStyle.Render("✓"))
		fmt.Printf("  View details: ei deployments show %s\n", deployment.Id)
	case "CONFIGURATION":
		fmt.Printf("%s Deployment requires configuration.\n", ui.WarningStyle.Render("!"))
		fmt.Printf("  View missing configs: ei deployments show %s\n", deployment.Id)
		fmt.Printf("  Apply config: ei deployments configure %s --config <file.json>\n", deployment.Id)
	case "FAILED":
		fmt.Printf("%s Deployment failed.\n", ui.ErrorStyle.Render("✗"))
		for _, e := range errors {
			fmt.Printf("  %s %s\n", ui.RenderCross(), e)
		}
	default:
		fmt.Printf("  Final status: %s\n", styledDeploymentStatus(finalStatus))
	}

	return nil
}

// deploymentStatusQuery is the lightweight query used for polling deployment progress.
const deploymentStatusQuery = `
	query GetDeploymentStatus($id: ID!) {
		organization {
			deployment(id: $id) {
				id
				status
				errors
				asyncTask {
					id status completedSteps totalSteps
				}
			}
		}
	}
`

// deploymentStatusResult is the response shape for the polling query.
type deploymentStatusResult struct {
	Organization struct {
		Deployment *struct {
			ID        string   `json:"id"`
			Status    string   `json:"status"`
			Errors    []string `json:"errors"`
			AsyncTask *struct {
				ID             string `json:"id"`
				Status         string `json:"status"`
				CompletedSteps int    `json:"completedSteps"`
				TotalSteps     int    `json:"totalSteps"`
			} `json:"asyncTask"`
		} `json:"deployment"`
	} `json:"organization"`
}

// pollDeploymentStatus polls the deployment status until it reaches a terminal state.
func pollDeploymentStatus(ctx context.Context, apiClient *client.Client, deploymentID string, timeout time.Duration, cmd *cobra.Command) (status string, errors []string, err error) {
	return pollDeploymentStatusWithInterval(ctx, apiClient, deploymentID, timeout, deploymentPollInterval, cmd)
}

// pollDeploymentStatusWithInterval is the internal implementation with configurable interval for testing.
func pollDeploymentStatusWithInterval(ctx context.Context, apiClient *client.Client, deploymentID string, timeout time.Duration, interval time.Duration, cmd *cobra.Command) (status string, errors []string, err error) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	timeoutCh := time.After(timeout)
	vars := map[string]any{"id": deploymentID}
	consecutiveErrors := 0
	var lastErr error

	for {
		select {
		case <-ctx.Done():
			return "", nil, ctx.Err()
		case <-timeoutCh:
			return "TIMEOUT", nil, fmt.Errorf("deployment did not complete within %s", timeout)
		case <-ticker.C:
			var result deploymentStatusResult
			err := apiClient.ExecuteInto(ctx, deploymentStatusQuery, vars, &result)
			if err != nil {
				consecutiveErrors++
				lastErr = err
				logger.Debug("deployment status poll error", "error", err, "consecutive_errors", consecutiveErrors)
				if consecutiveErrors >= maxConsecutivePollErrors {
					return "", nil, fmt.Errorf("failed to poll deployment status after %d consecutive errors: %w", consecutiveErrors, lastErr)
				}
				continue
			}

			// Reset error counter on successful poll
			consecutiveErrors = 0

			d := result.Organization.Deployment
			if d == nil {
				return "", nil, fmt.Errorf("deployment not found")
			}

			// Print progress if available
			if cmd != nil && !isJSONOutput(cmd) && d.AsyncTask != nil && d.AsyncTask.TotalSteps > 0 {
				fmt.Printf("\r  Progress: %d/%d steps (%s)   ",
					d.AsyncTask.CompletedSteps, d.AsyncTask.TotalSteps, styledDeploymentStatus(d.Status))
			}

			// Check if terminal state
			switch d.Status {
			case "COMPLETED", "FAILED", "CONFIGURATION":
				if cmd != nil && !isJSONOutput(cmd) {
					fmt.Println() // Clear progress line
				}
				return d.Status, d.Errors, nil
			}
			// Still in progress, continue polling
		}
	}
}
