// Copyright (c) ElementumAI, Inc.
// SPDX-License-Identifier: MPL-2.0

package cmd

import (
	"context"
	"fmt"

	"github.com/elementumltd/elementum-cli/auth"
	"github.com/elementumltd/elementum-cli/ui"
	"github.com/elementumltd/elementum-cli/internal/client"
	"github.com/spf13/cobra"
)

var automationDeleteCmd = &cobra.Command{
	Use:   "delete (<automation-id> | <app-namespace> <automation-name>)",
	Short: "Delete an automation",
	Long: `Delete an automation by ID or by app namespace and automation name.

Usage patterns:
  ei automation delete <automation-id>
  ei automation delete <app-namespace> <automation-name>

Examples:
  # Delete by UUID
  ei automation delete 9063aed1-bf8c-430d-882f-8c502355a3c7

  # Delete by app namespace + automation name
  ei automation delete myapp "Process Request"

  # Skip confirmation prompt
  ei automation delete myapp "Process Request" --force

  # Preview what would be deleted without actually deleting
  ei automation delete myapp "Process Request" --dry-run`,
	Args: cobra.RangeArgs(1, 2),
	RunE: runAutomationDelete,
}

func init() {
	automationDeleteCmd.Flags().BoolP("force", "f", false, "Skip confirmation prompt")
	automationDeleteCmd.Flags().Bool("dry-run", false, "Show what would be deleted without actually deleting")
}

func runAutomationDelete(cmd *cobra.Command, args []string) error {
	var automationID string
	var automationName string

	if looksLikeUUID(args[0]) {
		// ei automation delete <uuid>
		automationID = args[0]
	} else {
		// ei automation delete <app-namespace> <automation-name>
		if len(args) < 2 {
			return fmt.Errorf("when using namespace, provide: <app-namespace> <automation-name>")
		}
		namespace := args[0]
		automationName = args[1]

		var err error
		automationID, err = resolveAutomationByName(cmd, namespace, automationName)
		if err != nil {
			return err
		}
	}

	ctx := context.Background()
	c, err := auth.GetClientFromCmd(cmd)
	if err != nil {
		return err
	}

	// Fetch automation details (name, status, and current workflow ID)
	resp, err := client.GetAutomationDetails(ctx, c.Genqlient(), automationID)
	if err != nil {
		return fmt.Errorf("failed to get automation: %w", err)
	}

	automation := resp.Organization.Automation
	if automation == nil {
		return fmt.Errorf("automation not found: %s", automationID)
	}

	automationName = automation.Name
	automationStatus := automation.Status

	// Get current workflow ID if automation is published
	var currentWorkflowID string
	if automation.Current != nil {
		currentWorkflowID = automation.Current.Id
	}

	dryRun, _ := cmd.Flags().GetBool("dry-run")
	force, _ := cmd.Flags().GetBool("force")

	// Build description for dry-run and confirmation
	needsDisable := automationStatus == client.AutomationStatusActive && currentWorkflowID != ""
	actionDesc := "delete"
	if needsDisable {
		actionDesc = "disable and delete"
	}

	if dryRun {
		msg := fmt.Sprintf(" Would %s automation %q (%s)", actionDesc, automationName, automationID)
		if needsDisable {
			msg += fmt.Sprintf("\n  - Status: %s (will be disabled first)", automationStatus)
		}
		fmt.Println(ui.WarningStyle.Render("Dry run:") + msg)
		return nil
	}

	// Confirmation prompt
	if !force {
		confirmMsg := fmt.Sprintf("ID: %s\nThis action cannot be undone.", automationID)
		if needsDisable {
			confirmMsg = fmt.Sprintf("ID: %s\nStatus: %s (will be disabled first)\nThis action cannot be undone.", automationID, automationStatus)
		}
		confirmed, err := ui.Confirm(
			fmt.Sprintf("Delete automation %q?", automationName),
			confirmMsg,
		)
		if err != nil {
			return err
		}
		if !confirmed {
			fmt.Println(ui.MutedStyle.Render("Cancelled."))
			return nil
		}
	}

	// Disable the automation first if it's active
	if needsDisable {
		fmt.Println(ui.InfoStyle.Render("Disabling automation..."))
		inactiveStatus := client.WorkflowStatusInactive
		_, err = client.DeactivateWorkflow(ctx, c.Genqlient(), currentWorkflowID, client.WorkflowInput{
			Status: &inactiveStatus,
		})
		if err != nil {
			// Fetch and display dependencies as blockers
			if blockers := getAutomationBlockers(ctx, c, automationID); blockers != "" {
				fmt.Println()
				fmt.Println(ui.ErrorStyle.Render("Cannot disable automation - blocked by dependencies:"))
				fmt.Println(blockers)
				fmt.Println()
				fmt.Println(ui.MutedStyle.Render("Disable or delete the blocking resources first, then retry."))
				return fmt.Errorf("automation has active dependencies")
			}
			return fmt.Errorf("failed to disable automation: %w", err)
		}
	}

	// Delete the automation
	_, err = client.DeleteAutomation(ctx, c.Genqlient(), automationID)
	if err != nil {
		return fmt.Errorf("failed to delete automation: %w", err)
	}

	fmt.Println(ui.SuccessStyle.Render("Deleted automation:") + fmt.Sprintf(" %s (%s)", automationName, automationID))
	return nil
}

// getAutomationBlockers fetches dependencies that block disabling/deleting an automation
// and returns a formatted string of blockers, or empty string if no blockers found
func getAutomationBlockers(ctx context.Context, c *client.Client, automationID string) string {
	resp, err := client.GetAutomationDependencies(ctx, c.Genqlient(), automationID)
	if err != nil {
		return "" // Can't fetch dependencies, return empty
	}

	automation := resp.Organization.Automation
	if automation == nil {
		return ""
	}

	var blockers []string

	// Check for dependent automations (e.g., RunAutomation tasks)
	for _, edge := range automation.Usage.Automations.Edges {
		if edge.Node != nil {
			blockers = append(blockers, fmt.Sprintf("  • Automation: %s (%s)", edge.Node.Name, edge.Node.Id))
		}
	}

	// Check for dependent agents
	for _, edge := range automation.Usage.Agents.Edges {
		if edge.Node != nil {
			blockers = append(blockers, fmt.Sprintf("  • Agent: %s", (*edge.Node).GetName()))
		}
	}

	// Check for dependent agent tools
	for _, edge := range automation.Usage.AgentTools.Edges {
		if edge.Node != nil {
			blockers = append(blockers, fmt.Sprintf("  • Agent Tool: %s", (*edge.Node).GetName()))
		}
	}

	if len(blockers) == 0 {
		return ""
	}

	result := ""
	for _, b := range blockers {
		result += b + "\n"
	}
	return result
}
