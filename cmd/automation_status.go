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
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/elementumltd/elementum-cli/analysis"
	"github.com/elementumltd/elementum-cli/auth"
	"github.com/elementumltd/elementum-cli/discovery"
	"github.com/elementumltd/elementum-cli/server"
	"github.com/elementumltd/elementum-cli/ui"
	"github.com/elementumltd/elementum-cli/internal/client"
	"github.com/spf13/cobra"
)

var automationStatusCmd = &cobra.Command{
	Use:   "status (<automation-id> | <app-namespace> <automation-name>) [execution-id]",
	Short: "Show automation execution status",
	Long: `Show recent execution history for an automation.

Usage patterns:
  ei automation status <automation-id> [execution-id]
  ei automation status <app-namespace> <automation-name> [execution-id]

With no execution ID, lists recent executions with a health summary.
With an execution ID, shows detailed action breakdown for that execution.

Shortcut flags (auto-select execution):
  --latest          Auto-select the most recent execution
  --latest-failure  Auto-select the most recent failed execution

Output modes for execution detail:
  (default)  Fast table view of actions (no I/O data)
  --io NAME  Fetch I/O for specific action by name or 1-based index
  --all-io   Fetch I/O for all actions (slower, sequential)
  --expand   Same as --all-io (legacy flag)
  --timeline CLI visual timeline with duration bars
  --html     Open interactive HTML waterfall in browser (lazy loads I/O)
  --json     Full structured JSON output

Examples:
  # By automation UUID
  ei automation status 9063aed1-bf8c-430d-882f-8c502355a3c7

  # By app namespace + automation name
  ei automation status lumanow ValidateAndSubmit
  ei automation status lumanow "Process Request" --status FAILURE

  # Filter by status
  ei automation status lumanow ValidateAndSubmit --status FAILURE

  # Show last 24 hours
  ei automation status lumanow ValidateAndSubmit --since 24h

  # Auto-select latest execution and show details
  ei automation status lumanow ValidateAndSubmit --latest

  # Auto-select latest failure and show timeline
  ei automation status lumanow ValidateAndSubmit --latest-failure --timeline

  # One-liner debugging: latest failure + specific action I/O
  ei automation status lumanow ValidateAndSubmit --latest-failure --io "Send Email"

  # Show detail for a specific execution (fast, no I/O)
  ei automation status lumanow ValidateAndSubmit exec-456-uuid

  # Fetch I/O for specific action by name
  ei automation status lumanow ValidateAndSubmit exec-456-uuid --io "Send Email"

  # Fetch I/O for specific action by index (1-based)
  ei automation status lumanow ValidateAndSubmit exec-456-uuid --io 3

  # Fetch I/O for all actions (slower)
  ei automation status lumanow ValidateAndSubmit exec-456-uuid --all-io

  # Execution detail with timeline
  ei automation status lumanow ValidateAndSubmit exec-456-uuid --timeline

  # Open HTML waterfall in browser (lazy loads I/O on click)
  ei automation status lumanow ValidateAndSubmit exec-456-uuid --html

  # Watch for new executions (live stream)
  ei automation status lumanow ValidateAndSubmit --watch

  # JSON output
  ei automation status lumanow ValidateAndSubmit --json`,
	Args: cobra.RangeArgs(1, 3),
	RunE: runAutomationStatus,
}

func init() {
	automationStatusCmd.Flags().String("status", "", "Filter by status: SUCCESS, FAILURE, RUNNING, QUEUED, CANCELLED")
	automationStatusCmd.Flags().Int("limit", 20, "Number of executions to show")
	automationStatusCmd.Flags().String("since", "7d", "Time range: 7d, 24h, 30m, or date like 2026-02-01")
	automationStatusCmd.Flags().BoolP("watch", "w", false, "Watch for new executions (append-only live stream)")
	automationStatusCmd.Flags().Bool("expand", false, "Show full inputs/outputs for each action")
	automationStatusCmd.Flags().Bool("timeline", false, "Show CLI visual timeline with duration bars")
	automationStatusCmd.Flags().Bool("html", false, "Open interactive HTML waterfall in browser (lazy loads I/O)")
	automationStatusCmd.Flags().String("io", "", "Fetch I/O for specific action (name or 1-based index)")
	automationStatusCmd.Flags().Bool("all-io", false, "Fetch I/O for all actions (slower, sequential)")
	automationStatusCmd.Flags().Bool("latest", false, "Auto-select the most recent execution")
	automationStatusCmd.Flags().Bool("latest-failure", false, "Auto-select the most recent failed execution")
}

func runAutomationStatus(cmd *cobra.Command, args []string) error {
	var automationID string
	var executionID string

	if looksLikeUUID(args[0]) {
		// Existing: ei automation status <uuid> [execution-id]
		automationID = args[0]
		if len(args) == 2 {
			executionID = args[1]
		}
	} else {
		// New: ei automation status <app-namespace> <automation-name> [execution-id]
		if len(args) < 2 {
			return fmt.Errorf("when using namespace, provide: <app-namespace> <automation-name> [execution-id]")
		}
		namespace := args[0]
		automationName := args[1]
		if len(args) == 3 {
			executionID = args[2]
		}

		var err error
		automationID, err = resolveAutomationByName(cmd, namespace, automationName)
		if err != nil {
			return err
		}
	}

	// Check for --latest or --latest-failure flags
	latest, _ := cmd.Flags().GetBool("latest")
	latestFailure, _ := cmd.Flags().GetBool("latest-failure")

	if (latest || latestFailure) && executionID == "" {
		var err error
		executionID, err = resolveLatestExecution(cmd, automationID, latestFailure)
		if err != nil {
			return err
		}
	}

	// Detail mode: execution ID provided
	if executionID != "" {
		return runExecutionDetail(cmd, automationID, executionID)
	}

	// Watch mode
	watch, _ := cmd.Flags().GetBool("watch")
	if watch {
		return runExecutionWatch(cmd, automationID)
	}

	// List mode (default)
	return runExecutionList(cmd, automationID)
}

// resolveAutomationByName looks up an automation by app namespace and automation name
func resolveAutomationByName(cmd *cobra.Command, namespace, automationName string) (string, error) {
	ctx := context.Background()
	c, err := auth.GetClientFromCmd(cmd)
	if err != nil {
		return "", err
	}

	// Step 1: namespace -> aspect ID
	aspectID, _, err := resolveAspectByNamespace(ctx, c, namespace)
	if err != nil {
		return "", fmt.Errorf("app not found: %s", namespace)
	}

	// Step 2: Get automations for this aspect
	automations, err := discovery.GetAspectAutomations(ctx, c, aspectID)
	if err != nil {
		return "", fmt.Errorf("failed to list automations: %w", err)
	}

	// Step 3: Find by name (case-insensitive)
	for _, a := range automations {
		if strings.EqualFold(a.Name, automationName) {
			return a.ID, nil
		}
	}

	return "", fmt.Errorf("automation %q not found in %s", automationName, namespace)
}

// resolveLatestExecution fetches the most recent execution ID for an automation.
// If failureOnly is true, it filters for FAILURE status executions.
func resolveLatestExecution(cmd *cobra.Command, automationID string, failureOnly bool) (string, error) {
	ctx := context.Background()
	c, err := auth.GetClientFromCmd(cmd)
	if err != nil {
		return "", err
	}

	// Get aspect ID for permission-enforced query
	aspectID, automationName, _, err := getAspectInfoForAutomation(ctx, c, automationID)
	if err != nil {
		return "", err
	}

	// Build filter - use the same time range as default list mode
	filter, err := buildExecutionFilter(cmd)
	if err != nil {
		return "", err
	}

	// Override status filter if --latest-failure
	if failureOnly {
		filter.Status = []client.AutomationExecutionStatus{client.AutomationExecutionStatusFailure}
	}

	limit := 1
	sort := &client.AutomationExecutionSort{
		Direction: client.SortDirectionDesc,
	}

	if !isJSONOutput(cmd) {
		if failureOnly {
			fmt.Println(ui.InfoStyle.Render("⠿ Finding latest failed execution..."))
		} else {
			fmt.Println(ui.InfoStyle.Render("⠿ Finding latest execution..."))
		}
	}

	resp, err := client.GetAutomationExecutionsViaAspect(ctx, c.Genqlient(), aspectID, automationID, *filter, &limit, sort)
	if err != nil {
		return "", fmt.Errorf("failed to get executions: %w", err)
	}

	aspectPtr := resp.Organization.Aspect
	if aspectPtr == nil {
		return "", fmt.Errorf("aspect not found: %s", aspectID)
	}
	aspect := *aspectPtr

	automation := aspect.GetAutomation()
	if automation == nil {
		return "", fmt.Errorf("automation not found: %s", automationID)
	}

	executions := automation.Executions
	if len(executions.Edges) == 0 {
		if failureOnly {
			return "", fmt.Errorf("no failed executions found for %q in the specified time range", automationName)
		}
		return "", fmt.Errorf("no executions found for %q in the specified time range", automationName)
	}

	return executions.Edges[0].Node.Id, nil
}

// --- List Mode ---

func runExecutionList(cmd *cobra.Command, automationID string) error {
	ctx := context.Background()

	c, err := auth.GetClientFromCmd(cmd)
	if err != nil {
		return err
	}

	if !isJSONOutput(cmd) {
		fmt.Println(ui.InfoStyle.Render("⠿ Loading executions..."))
	}

	// Get aspect ID for permission-enforced query
	aspectID, automationName, automationStatus, err := getAspectInfoForAutomation(ctx, c, automationID)
	if err != nil {
		return err
	}

	filter, err := buildExecutionFilter(cmd)
	if err != nil {
		return err
	}

	limit, _ := cmd.Flags().GetInt("limit")
	sort := &client.AutomationExecutionSort{
		Direction: client.SortDirectionDesc,
	}

	// Use aspect-routed query for permission enforcement
	resp, err := client.GetAutomationExecutionsViaAspect(ctx, c.Genqlient(), aspectID, automationID, *filter, &limit, sort)
	if err != nil {
		return fmt.Errorf("failed to get executions: %w", err)
	}

	aspectPtr := resp.Organization.Aspect
	if aspectPtr == nil {
		return fmt.Errorf("aspect not found: %s", aspectID)
	}
	aspect := *aspectPtr

	automation := aspect.GetAutomation()
	if automation == nil {
		return fmt.Errorf("automation not found: %s", automationID)
	}

	executions := automation.Executions

	// JSON output
	if isJSONOutput(cmd) {
		return outputJSON(map[string]any{
			"automation": map[string]any{
				"id":     automation.Id,
				"name":   automation.Name,
				"status": automationStatus,
			},
			"total":      executions.Total,
			"executions": executions.Edges,
		})
	}

	// Health summary header
	summary := buildHealthSummaryViaAspect(executions.Edges)
	sinceVal, _ := cmd.Flags().GetString("since")
	fmt.Println()
	fmt.Println(ui.TitleStyle.Render(fmt.Sprintf("%s (%s)", automationName, automationStatus)) +
		ui.MutedStyle.Render(fmt.Sprintf(" — Last %s: ", sinceVal)) + summary)
	fmt.Println()

	if len(executions.Edges) == 0 {
		fmt.Println(ui.WarningStyle.Render("No executions found in the specified time range."))
		fmt.Println()
		return nil
	}

	// Table
	table := ui.NewTable([]string{"STATUS", "STARTED", "DURATION", "VERSION", "ERRORS"})
	for _, edge := range executions.Edges {
		exec := edge.Node
		table.AddRow(
			styledStatus(string(exec.Status)),
			formatTimestamp(exec.StartedAt),
			formatDuration(exec.Duration),
			fmt.Sprintf("v%d", exec.Version),
			formatErrors(exec.Errors),
		)
	}

	fmt.Println(table.RenderSimple())
	fmt.Println(ui.MutedStyle.Render(fmt.Sprintf("Total: %d executions", executions.Total)))
	fmt.Println()

	return nil
}

// --- Detail Mode ---

func runExecutionDetail(cmd *cobra.Command, automationID, executionID string) error {
	ctx := context.Background()

	c, err := auth.GetClientFromCmd(cmd)
	if err != nil {
		return err
	}

	// Check output mode flags
	expand, _ := cmd.Flags().GetBool("expand")
	timeline, _ := cmd.Flags().GetBool("timeline")
	htmlOutput, _ := cmd.Flags().GetBool("html")
	jsonOutput := isJSONOutput(cmd)
	ioTarget, _ := cmd.Flags().GetString("io")
	allIO, _ := cmd.Flags().GetBool("all-io")

	if !jsonOutput {
		fmt.Println(ui.InfoStyle.Render("⠿ Loading execution details..."))
	}

	// Get aspect ID for permission-enforced query
	aspectID, automationName, _, err := getAspectInfoForAutomation(ctx, c, automationID)
	if err != nil {
		return err
	}

	// Determine I/O fetching strategy:
	// 1. --all-io or --expand: Fetch all I/O (legacy behavior)
	// 2. --html: Use lazy-loading server mode
	// 3. --io <target>: Fetch specific action I/O
	// 4. Default: Fast metadata-only query

	if allIO || expand {
		// Fetch all I/O (legacy behavior)
		return runExecutionDetailWithIO(ctx, c, aspectID, automationID, automationName, executionID, expand, timeline, htmlOutput, jsonOutput)
	}

	if htmlOutput {
		// HTML mode: Use lazy-loading server
		return runExecutionDetailWithServer(ctx, c, aspectID, automationID, automationName, executionID)
	}

	// Fast path: metadata-only query
	return runExecutionDetailMetadata(ctx, c, aspectID, automationID, automationName, executionID, ioTarget, timeline, jsonOutput)
}

// runExecutionDetailWithIO uses the aspect-routed query to fetch execution with inputs/outputs
func runExecutionDetailWithIO(
	ctx context.Context,
	c *client.Client,
	aspectID, automationID, automationName, executionID string,
	expand, timeline, htmlOutput, jsonOutput bool,
) error {
	// Fetch execution with I/O via aspect path
	limit := 100
	resp, err := client.GetAutomationExecutionWithActionIO(ctx, c.Genqlient(), aspectID, automationID, executionID, &limit, nil)
	if err != nil {
		return fmt.Errorf("failed to get execution: %w", err)
	}

	aspectPtr := resp.Organization.Aspect
	if aspectPtr == nil {
		return fmt.Errorf("aspect not found: %s", aspectID)
	}

	// Dereference the pointer-to-interface to call methods
	aspect := *aspectPtr
	automation := aspect.GetAutomation()
	if automation == nil {
		return fmt.Errorf("automation not found: %s", automationID)
	}

	exec := automation.Execution
	if exec.Id == "" {
		return fmt.Errorf("execution not found: %s", executionID)
	}

	// Build analysis structure
	execAnalysis, err := analysis.BuildExecutionAnalysisWithIO(automationID, automationName, &exec)
	if err != nil {
		return fmt.Errorf("failed to build execution analysis: %w", err)
	}

	// Route to output format
	if jsonOutput {
		return outputJSON(map[string]any{
			"automation": map[string]any{
				"id":   automationID,
				"name": automationName,
			},
			"execution": execAnalysis,
		})
	}

	if htmlOutput {
		return analysis.RenderExecutionHTML(execAnalysis)
	}

	if timeline || expand {
		analysis.RenderExecutionTimeline(execAnalysis, true)
		return nil
	}

	// Default: table view with I/O
	return renderExecutionTableWithIO(automationName, execAnalysis)
}

// runExecutionDetailMetadata uses the fast metadata-only query with optional single-action I/O fetching
func runExecutionDetailMetadata(
	ctx context.Context,
	c *client.Client,
	aspectID, automationID, automationName, executionID string,
	ioTarget string,
	timeline, jsonOutput bool,
) error {
	// Fetch execution metadata (no I/O) - fast path
	limit := 100
	resp, err := client.GetAutomationExecutionMetadata(ctx, c.Genqlient(), aspectID, automationID, executionID, &limit, nil)
	if err != nil {
		return fmt.Errorf("failed to get execution: %w", err)
	}

	aspectPtr := resp.Organization.Aspect
	if aspectPtr == nil {
		return fmt.Errorf("aspect not found: %s", aspectID)
	}

	aspect := *aspectPtr
	automation := aspect.GetAutomation()
	if automation == nil {
		return fmt.Errorf("automation not found: %s", automationID)
	}

	exec := automation.Execution
	if exec.Id == "" {
		return fmt.Errorf("execution not found: %s", executionID)
	}

	// Build analysis structure (without I/O)
	execAnalysis, err := analysis.BuildExecutionAnalysisMetadata(automationID, automationName, &exec)
	if err != nil {
		return fmt.Errorf("failed to build execution analysis: %w", err)
	}

	// If --io flag provided, fetch I/O for specific action
	if ioTarget != "" {
		if err := fetchActionIO(ctx, c, aspectID, automationID, executionID, execAnalysis, ioTarget); err != nil {
			return err
		}
	}

	// Route to output format
	if jsonOutput {
		return outputJSON(map[string]any{
			"automation": map[string]any{
				"id":   automationID,
				"name": automationName,
			},
			"execution": execAnalysis,
		})
	}

	if timeline {
		analysis.RenderExecutionTimeline(execAnalysis, ioTarget != "")
		return nil
	}

	// Default: table view
	return renderExecutionTable(automationName, execAnalysis, ioTarget != "")
}

// fetchActionIO fetches I/O for a specific action by name or 1-based index
func fetchActionIO(
	ctx context.Context,
	c *client.Client,
	aspectID, automationID, executionID string,
	execAnalysis *analysis.ExecutionAnalysis,
	target string,
) error {
	if len(execAnalysis.Actions) == 0 {
		return fmt.Errorf("no actions to fetch I/O for")
	}

	// Try to parse as 1-based index first
	var actionIdx int = -1
	if idx, err := strconv.Atoi(target); err == nil {
		if idx < 1 || idx > len(execAnalysis.Actions) {
			return fmt.Errorf("action index %d out of range (1-%d)", idx, len(execAnalysis.Actions))
		}
		actionIdx = idx - 1
	} else {
		// Search by name (case-insensitive)
		for i, a := range execAnalysis.Actions {
			if strings.EqualFold(a.Name, target) || strings.EqualFold(a.Type, target) {
				actionIdx = i
				break
			}
		}
		if actionIdx < 0 {
			return fmt.Errorf("action %q not found", target)
		}
	}

	action := &execAnalysis.Actions[actionIdx]

	// Fetch I/O for this action
	ioResp, err := client.GetActionExecutionIO(ctx, c.Genqlient(), aspectID, automationID, executionID, action.ID)
	if err != nil {
		return fmt.Errorf("failed to get action I/O: %w", err)
	}

	ioAspectPtr := ioResp.Organization.Aspect
	if ioAspectPtr == nil {
		return fmt.Errorf("aspect not found for I/O fetch")
	}
	ioAspect := *ioAspectPtr
	ioAutomation := ioAspect.GetAutomation()
	if ioAutomation == nil {
		return fmt.Errorf("automation not found for I/O fetch")
	}

	ioAction := ioAutomation.Execution.Action
	if ioAction == nil {
		return fmt.Errorf("action not found in execution")
	}

	// Update the action with I/O data
	if ioAction.InputsOutputs != nil {
		if ioAction.InputsOutputs.Inputs != nil {
			action.Inputs = *ioAction.InputsOutputs.Inputs
		}
		if ioAction.InputsOutputs.Outputs != nil {
			action.Outputs = *ioAction.InputsOutputs.Outputs
		}
	}
	action.IOLoaded = true

	return nil
}

// runExecutionDetailWithServer starts a local HTTP server for lazy I/O loading in HTML mode
func runExecutionDetailWithServer(
	ctx context.Context,
	c *client.Client,
	aspectID, automationID, automationName, executionID string,
) error {
	// Fetch execution metadata (no I/O) - fast path
	limit := 100
	resp, err := client.GetAutomationExecutionMetadata(ctx, c.Genqlient(), aspectID, automationID, executionID, &limit, nil)
	if err != nil {
		return fmt.Errorf("failed to get execution: %w", err)
	}

	aspectPtr := resp.Organization.Aspect
	if aspectPtr == nil {
		return fmt.Errorf("aspect not found: %s", aspectID)
	}

	aspect := *aspectPtr
	automation := aspect.GetAutomation()
	if automation == nil {
		return fmt.Errorf("automation not found: %s", automationID)
	}

	exec := automation.Execution
	if exec.Id == "" {
		return fmt.Errorf("execution not found: %s", executionID)
	}

	// Build analysis structure (without I/O)
	execAnalysis, err := analysis.BuildExecutionAnalysisMetadata(automationID, automationName, &exec)
	if err != nil {
		return fmt.Errorf("failed to build execution analysis: %w", err)
	}

	// Start the server for lazy I/O loading
	return startExecutionServer(ctx, c, aspectID, automationID, executionID, execAnalysis)
}

// startExecutionServer starts the HTTP server for lazy I/O loading
func startExecutionServer(
	ctx context.Context,
	c *client.Client,
	aspectID, automationID, executionID string,
	execAnalysis *analysis.ExecutionAnalysis,
) error {
	srv := server.NewExecutionServer(c, aspectID, automationID, executionID, execAnalysis)
	return srv.Start(ctx)
}

// renderExecutionTable renders execution without I/O (unless showIO is true for fetched actions)
func renderExecutionTable(_ string, exec *analysis.ExecutionAnalysis, showIO bool) error {
	// Header
	fmt.Println()
	fmt.Println(ui.TitleStyle.Render(fmt.Sprintf("Execution %s (%s)", ui.Truncate(exec.ExecutionID, 12), styledStatus(exec.Status))))
	fmt.Printf("  %s  %s\n", ui.LabelStyle.Render("Started:"), exec.StartedAt.Local().Format("2006-01-02 15:04:05"))
	if exec.CompletedAt != nil {
		fmt.Printf("  %s  %s\n", ui.LabelStyle.Render("Completed:"), exec.CompletedAt.Local().Format("2006-01-02 15:04:05"))
	}
	fmt.Printf("  %s  %s\n", ui.LabelStyle.Render("Duration:"), formatDuration(exec.Duration))
	fmt.Printf("  %s  v%d\n", ui.LabelStyle.Render("Version:"), exec.Version)

	if len(exec.Errors) > 0 {
		fmt.Printf("  %s  %s\n", ui.LabelStyle.Render("Errors:"), ui.ErrorStyle.Render(strings.Join(exec.Errors, "; ")))
	}
	fmt.Println()

	// Actions table
	if len(exec.Actions) > 0 {
		fmt.Println(ui.SubtitleStyle.Render(fmt.Sprintf("Actions (%d):", len(exec.Actions))))
		fmt.Println()

		for i, action := range exec.Actions {
			taskName := action.Name
			if taskName == "" {
				taskName = action.Type
			}

			durationStr := "-"
			if action.Duration != nil {
				durationStr = formatDuration(*action.Duration)
			}

			// Action header line
			fmt.Printf("  %d. %s  %s  [%s]  %s\n",
				i+1,
				styledStatus(action.Status),
				taskName,
				ui.MutedStyle.Render(action.Type),
				durationStr,
			)

			// Error detail
			if action.Error != nil && *action.Error != "" {
				errMsg := *action.Error
				if len(errMsg) > 80 {
					errMsg = errMsg[:77] + "..."
				}
				fmt.Printf("     %s %s\n", ui.ErrorStyle.Render("Error:"), errMsg)
			}

			// Show I/O only if loaded (no truncation since user explicitly requested --io)
			if showIO && action.IOLoaded {
				if len(action.Inputs) > 0 && string(action.Inputs) != "null" && string(action.Inputs) != "{}" {
					inputPreview := formatIOPreview(action.Inputs, 0)
					fmt.Printf("     %s %s\n", ui.LabelStyle.Render("Inputs:"), ui.MutedStyle.Render(inputPreview))
				}
				if len(action.Outputs) > 0 && string(action.Outputs) != "null" && string(action.Outputs) != "{}" {
					outputPreview := formatIOPreview(action.Outputs, 0)
					fmt.Printf("     %s %s\n", ui.LabelStyle.Render("Outputs:"), ui.MutedStyle.Render(outputPreview))
				}
			}

			fmt.Println()
		}

		// Show failed action error detail if not already shown inline
		if exec.FailedAction != nil && exec.FailedAction.Error != nil {
			fmt.Println(ui.ErrorStyle.Render(fmt.Sprintf("Failed Action: %s", *exec.FailedAction.Error)))
		}

		// Hint about --io flag
		if !showIO {
			fmt.Println(ui.MutedStyle.Render("Tip: Use --io <name|index> to fetch inputs/outputs for a specific action"))
			fmt.Println(ui.MutedStyle.Render("     Use --all-io to fetch inputs/outputs for all actions"))
		}
	} else {
		fmt.Println(ui.MutedStyle.Render("No actions recorded for this execution."))
		fmt.Println()
	}

	return nil
}

// getAspectInfoForAutomation fetches the aspect ID, name, and status for a given automation
func getAspectInfoForAutomation(ctx context.Context, c *client.Client, automationID string) (string, string, string, error) {
	// Use GetAutomationExecutions with a minimal filter to get the entity
	filter := client.AutomationExecutionFilter{
		Range: client.DateRangeInput{
			Start: time.Now().Add(-24 * time.Hour).UTC().Format(time.RFC3339),
			End:   time.Now().UTC().Format(time.RFC3339),
		},
	}
	limit := 1

	resp, err := client.GetAutomationExecutions(ctx, c.Genqlient(), automationID, filter, &limit, nil)
	if err != nil {
		return "", "", "", err
	}

	automation := resp.Organization.Automation
	if automation == nil {
		return "", "", "", fmt.Errorf("automation not found: %s", automationID)
	}

	entity := automation.GetEntity()
	if entity == nil {
		return "", "", "", fmt.Errorf("could not determine aspect for automation")
	}

	// Extract aspect ID from entity union type using type assertion
	// The Entity interface is implemented by AspectApp, AspectElement, AspectTask, AspectTransaction, User
	type entityWithID interface {
		GetId() string
	}
	if e, ok := entity.(entityWithID); ok {
		return e.GetId(), automation.Name, string(automation.Status), nil
	}

	return "", "", "", fmt.Errorf("unexpected entity type for automation")
}

// buildHealthSummaryViaAspect builds a summary string like "142 success, 3 failures, 1 running" for aspect-routed query
func buildHealthSummaryViaAspect(edges []client.GetAutomationExecutionsViaAspectOrganizationAspectAutomationExecutionsAutomationExecutionConnectionEdgesAutomationExecutionEdge) string {
	counts := map[client.AutomationExecutionStatus]int{}
	for _, edge := range edges {
		counts[edge.Node.Status]++
	}

	var parts []string
	if n := counts[client.AutomationExecutionStatusSuccess]; n > 0 {
		parts = append(parts, ui.SuccessStyle.Render(fmt.Sprintf("%d success", n)))
	}
	if n := counts[client.AutomationExecutionStatusFailure]; n > 0 {
		parts = append(parts, ui.ErrorStyle.Render(fmt.Sprintf("%d failures", n)))
	}
	if n := counts[client.AutomationExecutionStatusRunning]; n > 0 {
		parts = append(parts, ui.WarningStyle.Render(fmt.Sprintf("%d running", n)))
	}
	if n := counts[client.AutomationExecutionStatusQueued]; n > 0 {
		parts = append(parts, ui.InfoStyle.Render(fmt.Sprintf("%d queued", n)))
	}
	if n := counts[client.AutomationExecutionStatusCancelled]; n > 0 {
		parts = append(parts, ui.MutedStyle.Render(fmt.Sprintf("%d cancelled", n)))
	}

	if len(parts) == 0 {
		return ui.MutedStyle.Render("no executions")
	}
	return strings.Join(parts, ", ")
}

// renderExecutionTableWithIO renders the execution table with inputs/outputs for each action
func renderExecutionTableWithIO(_ string, exec *analysis.ExecutionAnalysis) error {
	// Header
	fmt.Println()
	fmt.Println(ui.TitleStyle.Render(fmt.Sprintf("Execution %s (%s)", ui.Truncate(exec.ExecutionID, 12), styledStatus(exec.Status))))
	fmt.Printf("  %s  %s\n", ui.LabelStyle.Render("Started:"), exec.StartedAt.Local().Format("2006-01-02 15:04:05"))
	if exec.CompletedAt != nil {
		fmt.Printf("  %s  %s\n", ui.LabelStyle.Render("Completed:"), exec.CompletedAt.Local().Format("2006-01-02 15:04:05"))
	}
	fmt.Printf("  %s  %s\n", ui.LabelStyle.Render("Duration:"), formatDuration(exec.Duration))
	fmt.Printf("  %s  v%d\n", ui.LabelStyle.Render("Version:"), exec.Version)

	if len(exec.Errors) > 0 {
		fmt.Printf("  %s  %s\n", ui.LabelStyle.Render("Errors:"), ui.ErrorStyle.Render(strings.Join(exec.Errors, "; ")))
	}
	fmt.Println()

	// Actions with I/O
	if len(exec.Actions) > 0 {
		fmt.Println(ui.SubtitleStyle.Render(fmt.Sprintf("Actions (%d):", len(exec.Actions))))
		fmt.Println()

		for i, action := range exec.Actions {
			taskName := action.Name
			if taskName == "" {
				taskName = action.Type
			}

			durationStr := "-"
			if action.Duration != nil {
				durationStr = formatDuration(*action.Duration)
			}

			// Action header line
			fmt.Printf("  %d. %s  %s  [%s]  %s\n",
				i+1,
				styledStatus(action.Status),
				taskName,
				ui.MutedStyle.Render(action.Type),
				durationStr,
			)

			// Error detail
			if action.Error != nil && *action.Error != "" {
				errMsg := *action.Error
				if len(errMsg) > 80 {
					errMsg = errMsg[:77] + "..."
				}
				fmt.Printf("     %s %s\n", ui.ErrorStyle.Render("Error:"), errMsg)
			}

			// Inputs (if present and not empty/null) - no truncation since user explicitly filtered by action
			if len(action.Inputs) > 0 && string(action.Inputs) != "null" && string(action.Inputs) != "{}" {
				inputPreview := formatIOPreview(action.Inputs, 0)
				fmt.Printf("     %s %s\n", ui.LabelStyle.Render("Inputs:"), ui.MutedStyle.Render(inputPreview))
			}

			// Outputs (if present and not empty/null) - no truncation since user explicitly filtered by action
			if len(action.Outputs) > 0 && string(action.Outputs) != "null" && string(action.Outputs) != "{}" {
				outputPreview := formatIOPreview(action.Outputs, 0)
				fmt.Printf("     %s %s\n", ui.LabelStyle.Render("Outputs:"), ui.MutedStyle.Render(outputPreview))
			}

			fmt.Println()
		}

		// Show failed action error detail if not already shown inline
		if exec.FailedAction != nil && exec.FailedAction.Error != nil {
			fmt.Println(ui.ErrorStyle.Render(fmt.Sprintf("Failed Action: %s", *exec.FailedAction.Error)))
		}
	} else {
		fmt.Println(ui.MutedStyle.Render("No actions recorded for this execution."))
		fmt.Println()
	}

	return nil
}

// formatIOPreview formats JSON inputs/outputs for compact display.
// If maxLen is 0, no truncation is applied.
func formatIOPreview(data []byte, maxLen int) string {
	if len(data) == 0 {
		return "(empty)"
	}

	s := string(data)
	// Remove newlines for compact display
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "  ", " ")

	if maxLen > 0 && len(s) > maxLen {
		s = s[:maxLen-3] + "..."
	}

	return s
}

// --- Watch Mode ---

func runExecutionWatch(cmd *cobra.Command, automationID string) error {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	c, err := auth.GetClientFromCmd(cmd)
	if err != nil {
		return err
	}

	// Get aspect ID for permission-enforced query
	aspectID, automationName, _, err := getAspectInfoForAutomation(ctx, c, automationID)
	if err != nil {
		return err
	}

	filter, err := buildExecutionFilter(cmd)
	if err != nil {
		return err
	}

	limit := 5 // Small limit for watch mode -- just get recent
	sort := &client.AutomationExecutionSort{
		Direction: client.SortDirectionDesc,
	}

	// Initial fetch using aspect-routed query
	resp, err := client.GetAutomationExecutionsViaAspect(ctx, c.Genqlient(), aspectID, automationID, *filter, &limit, sort)
	if err != nil {
		return fmt.Errorf("failed to get executions: %w", err)
	}

	aspectPtr := resp.Organization.Aspect
	if aspectPtr == nil {
		return fmt.Errorf("aspect not found: %s", aspectID)
	}
	aspect := *aspectPtr

	automation := aspect.GetAutomation()
	if automation == nil {
		return fmt.Errorf("automation not found: %s", automationID)
	}

	fmt.Println()
	fmt.Println(ui.TitleStyle.Render(fmt.Sprintf("Watching %s...", automationName)) +
		ui.MutedStyle.Render(" (Ctrl+C to stop)"))
	fmt.Println()

	// Print initial executions
	lastSeenID := ""
	for i := len(automation.Executions.Edges) - 1; i >= 0; i-- {
		exec := automation.Executions.Edges[i].Node
		printWatchLineViaAspect(exec)
		lastSeenID = exec.Id
	}

	// Poll loop
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			fmt.Println()
			fmt.Println(ui.MutedStyle.Render("Stopped watching."))
			return nil
		case <-ticker.C:
			pollResp, err := client.GetAutomationExecutionsViaAspect(ctx, c.Genqlient(), aspectID, automationID, *filter, &limit, sort)
			if err != nil {
				// Don't fail on transient errors during watch
				continue
			}
			pollAspectPtr := pollResp.Organization.Aspect
			if pollAspectPtr == nil {
				continue
			}
			pollAspect := *pollAspectPtr
			pollAutomation := pollAspect.GetAutomation()
			if pollAutomation == nil {
				continue
			}

			edges := pollAutomation.Executions.Edges

			// Find new executions (ones we haven't seen yet)
			var newExecs []client.GetAutomationExecutionsViaAspectOrganizationAspectAutomationExecutionsAutomationExecutionConnectionEdgesAutomationExecutionEdgeNodeAutomationExecution
			for _, edge := range edges {
				if edge.Node.Id == lastSeenID {
					break
				}
				newExecs = append(newExecs, edge.Node)
			}

			// Print new executions in chronological order (oldest first)
			for i := len(newExecs) - 1; i >= 0; i-- {
				printWatchLineViaAspect(newExecs[i])
			}

			if len(newExecs) > 0 {
				lastSeenID = newExecs[0].Id
			}
		}
	}
}

func printWatchLineViaAspect(exec client.GetAutomationExecutionsViaAspectOrganizationAspectAutomationExecutionsAutomationExecutionConnectionEdgesAutomationExecutionEdgeNodeAutomationExecution) {
	ts := formatTimestamp(exec.StartedAt)
	status := styledStatus(string(exec.Status))
	duration := formatDuration(exec.Duration)
	version := fmt.Sprintf("v%d", exec.Version)

	line := fmt.Sprintf("%s  %s  %s  %s", ts, status, version, duration)
	if len(exec.Errors) > 0 {
		line += "  " + ui.ErrorStyle.Render(exec.Errors[0])
	}
	fmt.Println(line)
}

// --- Helpers ---

// parseSince parses a --since value into a time.Time.
// Supports: "7d", "24h", "30m", "60s", "2026-02-01", "2026-02-01T10:00:00Z"
func parseSince(value string) (time.Time, error) {
	if value == "" {
		value = "7d"
	}

	// Try RFC3339 first
	if t, err := time.Parse(time.RFC3339, value); err == nil {
		return t, nil
	}

	// Try date-only
	if t, err := time.Parse("2006-01-02", value); err == nil {
		return t, nil
	}

	// Try shorthand: Nd, Nh, Nm, Ns
	if len(value) >= 2 {
		suffix := value[len(value)-1:]
		numStr := value[:len(value)-1]
		num, err := strconv.Atoi(numStr)
		if err == nil && num > 0 {
			switch suffix {
			case "d":
				return time.Now().Add(-time.Duration(num) * 24 * time.Hour), nil
			case "h":
				return time.Now().Add(-time.Duration(num) * time.Hour), nil
			case "m":
				return time.Now().Add(-time.Duration(num) * time.Minute), nil
			case "s":
				return time.Now().Add(-time.Duration(num) * time.Second), nil
			}
		}
	}

	return time.Time{}, fmt.Errorf("invalid --since value: %q (use 7d, 24h, 30m, a date like 2026-02-01, or RFC3339)", value)
}

// buildExecutionFilter builds an AutomationExecutionFilter from command flags.
func buildExecutionFilter(cmd *cobra.Command) (*client.AutomationExecutionFilter, error) {
	sinceVal, _ := cmd.Flags().GetString("since")
	since, err := parseSince(sinceVal)
	if err != nil {
		return nil, err
	}

	filter := &client.AutomationExecutionFilter{
		Range: client.DateRangeInput{
			Start: since.UTC().Format(time.RFC3339),
			End:   time.Now().UTC().Format(time.RFC3339),
		},
	}

	statusFlag, _ := cmd.Flags().GetString("status")
	if statusFlag != "" {
		status := client.AutomationExecutionStatus(strings.ToUpper(statusFlag))
		filter.Status = []client.AutomationExecutionStatus{status}
	}

	return filter, nil
}

// formatDuration formats a duration in milliseconds to a human-readable string.
func formatDuration(ms int) string {
	if ms <= 0 {
		return "-"
	}
	if ms < 1000 {
		return fmt.Sprintf("%dms", ms)
	}
	if ms < 60000 {
		return fmt.Sprintf("%.1fs", float64(ms)/1000)
	}
	if ms < 3600000 {
		minutes := ms / 60000
		seconds := (ms % 60000) / 1000
		if seconds == 0 {
			return fmt.Sprintf("%dm", minutes)
		}
		return fmt.Sprintf("%dm %ds", minutes, seconds)
	}
	hours := ms / 3600000
	minutes := (ms % 3600000) / 60000
	if minutes == 0 {
		return fmt.Sprintf("%dh", hours)
	}
	return fmt.Sprintf("%dh %dm", hours, minutes)
}

// styledStatus returns a color-coded status string.
func styledStatus(status string) string {
	switch status {
	case "SUCCESS":
		return ui.SuccessStyle.Render("SUCCESS")
	case "FAILURE":
		return ui.ErrorStyle.Render("FAILURE")
	case "RUNNING":
		return ui.WarningStyle.Render("RUNNING")
	case "QUEUED":
		return ui.InfoStyle.Render("QUEUED")
	case "CANCELLED":
		return ui.MutedStyle.Render("CANCELLED")
	default:
		return status
	}
}

// formatTimestamp formats a datetime string for display.
func formatTimestamp(ts string) string {
	t, err := time.Parse(time.RFC3339, ts)
	if err != nil {
		// Try RFC3339Nano
		t, err = time.Parse(time.RFC3339Nano, ts)
		if err != nil {
			return ts // Return as-is if unparseable
		}
	}
	return t.Local().Format("2006-01-02 15:04:05")
}

// formatErrors formats an error list for table display.
func formatErrors(errors []string) string {
	if len(errors) == 0 {
		return "-"
	}
	first := errors[0]
	if len(first) > 50 {
		first = first[:47] + "..."
	}
	if len(errors) > 1 {
		return fmt.Sprintf("%s (+%d more)", first, len(errors)-1)
	}
	return first
}

// buildHealthSummary builds a summary string like "142 success, 3 failures, 1 running"
func buildHealthSummary(edges []client.GetAutomationExecutionsOrganizationAutomationExecutionsAutomationExecutionConnectionEdgesAutomationExecutionEdge) string {
	counts := map[client.AutomationExecutionStatus]int{}
	for _, edge := range edges {
		counts[edge.Node.Status]++
	}

	var parts []string
	if n := counts[client.AutomationExecutionStatusSuccess]; n > 0 {
		parts = append(parts, ui.SuccessStyle.Render(fmt.Sprintf("%d success", n)))
	}
	if n := counts[client.AutomationExecutionStatusFailure]; n > 0 {
		parts = append(parts, ui.ErrorStyle.Render(fmt.Sprintf("%d failures", n)))
	}
	if n := counts[client.AutomationExecutionStatusRunning]; n > 0 {
		parts = append(parts, ui.WarningStyle.Render(fmt.Sprintf("%d running", n)))
	}
	if n := counts[client.AutomationExecutionStatusQueued]; n > 0 {
		parts = append(parts, ui.InfoStyle.Render(fmt.Sprintf("%d queued", n)))
	}
	if n := counts[client.AutomationExecutionStatusCancelled]; n > 0 {
		parts = append(parts, ui.MutedStyle.Render(fmt.Sprintf("%d cancelled", n)))
	}

	if len(parts) == 0 {
		return ui.MutedStyle.Render("no executions")
	}
	return strings.Join(parts, ", ")
}
