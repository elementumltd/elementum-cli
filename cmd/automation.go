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
	"strings"

	"github.com/elementumltd/elementum-cli/auth"
	"github.com/elementumltd/elementum-cli/discovery"
	"github.com/elementumltd/elementum-cli/export"
	eclient "github.com/elementumltd/elementum-cli/internal/client"
	"github.com/elementumltd/elementum-cli/state"
	"github.com/elementumltd/elementum-cli/ui"
	"github.com/spf13/cobra"
)

var automationsCmd = &cobra.Command{
	Use:     "automations",
	Aliases: []string{"automation"},
	Short:   "Manage and monitor automations",
	Long:    "Commands for listing, monitoring and managing Elementum automations.",
}

var automationsListCmd = &cobra.Command{
	Use:   "list <app-namespace>",
	Short: "List automations for an app or element",
	Long:  "Display automations configured for the specified app or element.",
	Args:  cobra.ExactArgs(1),
	RunE:  runAutomationsList,
}

var automationsShowCmd = &cobra.Command{
	Use:   "show [name-or-id]",
	Short: "Show automation details",
	Long: `Display details about an automation workflow.

By default, reads from Terraform state in the current directory.
Use --from-remote to query the Elementum API directly.

Shows the automation structure including:
- Trigger type and configuration
- Tasks with their refs
- Workflow status

Examples:
  # From state (default)
  ei automations show my_automation
  ei automations show abc-123-uuid
  ei automations show --state-file ./terraform.tfstate my_automation

  # From remote API
  ei automations show --from-remote abc-123-uuid
  ei automations show --from-remote abc-123-uuid --json`,
	Args: cobra.ExactArgs(1),
	RunE: runAutomationsShow,
}

func init() {
	automationsListCmd.Flags().Bool("details", false, "Show full task configuration for each automation")
	automationsShowCmd.Flags().String("state-file", "", "Path to Terraform state file (default: ./terraform.tfstate)")
	automationsShowCmd.Flags().Bool("from-remote", false, "Query automation from Elementum API instead of state")
	automationsShowCmd.Flags().Bool("hcl", false, "Export automation as Terraform HCL")
	automationsCmd.AddCommand(automationsListCmd)
	automationsCmd.AddCommand(automationsShowCmd)
	automationsCmd.AddCommand(automationStatusCmd)
	automationsCmd.AddCommand(automationDeleteCmd)
}

// GetAutomationsCmd returns the automations command for registration
func GetAutomationsCmd() *cobra.Command {
	return automationsCmd
}

func runAutomationsList(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	c, err := auth.GetClientFromCmd(cmd)
	if err != nil {
		return err
	}

	namespace := args[0]
	showDetails, _ := cmd.Flags().GetBool("details")

	if !isJSONOutput(cmd) {
		fmt.Println(ui.InfoStyle.Render("Loading automations..."))
	}

	// Resolve namespace to aspect ID
	aspectID, aspectType, err := resolveAspectByNamespace(ctx, c, namespace)
	if err != nil {
		return fmt.Errorf("failed to resolve namespace %q: %w", namespace, err)
	}

	// Fetch automations
	automations, err := discovery.GetAspectAutomations(ctx, c, aspectID)
	if err != nil {
		return fmt.Errorf("failed to list automations: %w", err)
	}

	// For --details mode, fetch full task configs
	if showDetails {
		return runAutomationsListDetailed(cmd, c, automations, aspectID, aspectType, namespace)
	}

	if isJSONOutput(cmd) {
		return outputJSON(automations)
	}

	if len(automations) == 0 {
		fmt.Println()
		fmt.Println(ui.WarningStyle.Render("No automations found."))
		fmt.Println()
		return nil
	}

	// Display table
	table := ui.NewTable([]string{"NAME", "STATUS", "TRIGGERS", "TASKS", "ID"})

	for _, auto := range automations {
		triggerTypes := make([]string, len(auto.Triggers))
		for i, t := range auto.Triggers {
			triggerTypes[i] = t.Type
		}

		table.AddRow(
			auto.Name,
			auto.Status,
			strings.Join(triggerTypes, ", "),
			fmt.Sprintf("%d", len(auto.Tasks)),
			auto.ID,
		)
	}

	fmt.Println()
	fmt.Println(ui.TitleStyle.Render(fmt.Sprintf("Automations (%s: %s)", aspectType, namespace)))
	fmt.Println()
	fmt.Println(table.Render())
	fmt.Println()
	fmt.Println(ui.MutedStyle.Render(fmt.Sprintf("Total: %d automations", len(automations))))
	fmt.Println()

	return nil
}

func runAutomationsListDetailed(cmd *cobra.Command, c *eclient.Client, automations []discovery.Automation, aspectID, aspectType, namespace string) error {
	ctx := context.Background()

	type automationWithWorkflow struct {
		Automation discovery.Automation
		Workflow   *eclient.WorkflowFullDetails
	}

	// Fetch full workflow details for each automation
	results := make([]automationWithWorkflow, len(automations))
	for i, auto := range automations {
		results[i].Automation = auto
		if auto.WorkflowID != "" {
			wf, err := c.GetWorkflowWithFullTasks(ctx, aspectID, auto.WorkflowID)
			if err == nil {
				results[i].Workflow = wf
			}
		}
	}

	if isJSONOutput(cmd) {
		jsonResults := make([]map[string]any, len(results))
		for i, r := range results {
			entry := map[string]any{
				"id":     r.Automation.ID,
				"name":   r.Automation.Name,
				"status": r.Automation.Status,
			}
			if r.Workflow != nil {
				entry["triggers"] = r.Workflow.Triggers
				entry["tasks"] = r.Workflow.Tasks
				entry["outputs"] = r.Workflow.Outputs
			}
			jsonResults[i] = entry
		}
		return outputJSON(jsonResults)
	}

	if len(automations) == 0 {
		fmt.Println()
		fmt.Println(ui.WarningStyle.Render("No automations found."))
		fmt.Println()
		return nil
	}

	fmt.Println()
	fmt.Println(ui.TitleStyle.Render(fmt.Sprintf("Automations (%s: %s) - Full Details", aspectType, namespace)))
	fmt.Println()

	for _, r := range results {
		tree := ui.NewTree(fmt.Sprintf("%s [%s]", r.Automation.Name, r.Automation.Status))
		tree.Root.AddChildWithLabel("ID", r.Automation.ID)

		if r.Workflow != nil {
			renderFullWorkflow(tree.Root, r.Workflow)
		} else {
			// Fallback: show basic task info
			if len(r.Automation.Tasks) > 0 {
				tasksNode := tree.Root.AddChildWithLabel("Tasks", fmt.Sprintf("(%d)", len(r.Automation.Tasks)))
				for _, t := range r.Automation.Tasks {
					tasksNode.AddChildWithLabel(fmt.Sprintf("%s [%s]", t.Name, t.Type), t.ID)
				}
			}
		}

		fmt.Println(tree.Render())
		fmt.Println()
	}

	fmt.Println(ui.MutedStyle.Render(fmt.Sprintf("Total: %d automations", len(automations))))
	fmt.Println()

	return nil
}

func runAutomationsShow(cmd *cobra.Command, args []string) error {
	nameOrID := args[0]

	// Check if querying from remote API
	fromRemote, _ := cmd.Flags().GetBool("from-remote")
	if fromRemote {
		return runAutomationsShowFromRemote(cmd, nameOrID)
	}

	// Default: read from state
	return runAutomationsShowFromState(cmd, nameOrID)
}

func runAutomationsShowFromState(cmd *cobra.Command, nameOrID string) error {
	stateFilePath, _ := cmd.Flags().GetString("state-file")

	// Find state file
	statePath, err := state.FindStateFile(stateFilePath)
	if err != nil {
		return err
	}

	// Read state file
	tfState, err := state.ReadStateFile(statePath)
	if err != nil {
		return err
	}

	// Try to find automation by ID first, then by name
	automation, err := state.FindAutomationByID(tfState, nameOrID)
	if err != nil {
		// Try by name
		automation, err = state.FindAutomationByName(tfState, nameOrID)
		if err != nil {
			return fmt.Errorf("automation '%s' not found in state file: %s\nUse --from-remote to query the API directly", nameOrID, statePath)
		}
	}

	// JSON output
	if isJSONOutput(cmd) {
		return outputJSON(automation)
	}

	// Show info message
	fmt.Println(ui.InfoStyle.Render(fmt.Sprintf("Reading from: %s", statePath)))
	fmt.Println()

	// Build tree for display
	tree := ui.NewTree(fmt.Sprintf("%s (%s)", automation.Name, automation.ID))

	// Add resource address
	tree.Root.AddChildWithLabel("Resource", automation.Address)

	// Add IDs
	if automation.WorkflowID != "" {
		tree.Root.AddChildWithLabel("Workflow ID", automation.WorkflowID)
	}
	if automation.AppID != "" {
		tree.Root.AddChildWithLabel("App ID", automation.AppID)
	}

	// Add trigger info
	if automation.TriggerType != "" {
		triggerLabel := automation.TriggerType
		if automation.TriggerID != "" {
			triggerLabel += fmt.Sprintf(" (%s)", automation.TriggerID)
		}
		tree.Root.AddChildWithLabel("Trigger", triggerLabel)
	}

	// Add refs
	if len(automation.Refs) > 0 {
		refsNode := tree.Root.AddChildWithLabel("Refs", fmt.Sprintf("(%d)", len(automation.Refs)))
		for _, ref := range automation.Refs {
			refsNode.AddChildWithLabel(ref.Name, state.FormatRefType(ref))
		}
	}

	// Add tasks
	if len(automation.Tasks) > 0 {
		tasksNode := tree.Root.AddChildWithLabel("Tasks", fmt.Sprintf("(%d)", len(automation.Tasks)))
		for _, task := range automation.Tasks {
			taskLabel := fmt.Sprintf("%s [%s]", task.Name, task.TaskType)
			taskNode := tasksNode.AddChildWithLabel(taskLabel)
			taskNode.AddChildWithLabel("Resource", task.Address)

			// Add task refs
			if len(task.Refs) > 0 {
				taskRefsNode := taskNode.AddChildWithLabel("Refs", fmt.Sprintf("(%d)", len(task.Refs)))
				for _, ref := range task.Refs {
					taskRefsNode.AddChildWithLabel(ref.Name, state.FormatRefType(ref))
				}
			}
		}
	}

	// Display tree
	fmt.Println(tree.Render())
	fmt.Println()

	// Usage tips
	fmt.Println(ui.SubtitleStyle.Render("Usage in Terraform:"))
	fmt.Printf("  %s %s.refs[\"<ref-name>\"]\n", ui.RenderBullet(), automation.Address)
	for _, task := range automation.Tasks {
		if len(task.Refs) > 0 {
			fmt.Printf("  %s %s.refs[\"<ref-name>\"]\n", ui.RenderBullet(), task.Address)
		}
	}
	fmt.Println()

	return nil
}

func runAutomationsShowFromRemote(cmd *cobra.Command, automationID string) error {
	ctx := context.Background()

	// Get authenticated client
	c, err := auth.GetClientFromCmd(cmd)
	if err != nil {
		return err
	}

	// Show spinner while loading (skip for JSON output)
	if !isJSONOutput(cmd) {
		fmt.Println(ui.InfoStyle.Render(fmt.Sprintf("Looking up automation from API: %s", automationID)))
	}

	// Fetch automation details (basic info + workflow IDs)
	resp, err := eclient.GetAutomationDetails(ctx, c.Genqlient(), automationID)
	if err != nil {
		return fmt.Errorf("failed to get automation: %w", err)
	}

	automation := resp.Organization.Automation
	if automation == nil {
		return fmt.Errorf("automation not found: %s", automationID)
	}

	// Determine which workflow to show full details for
	var workflowID string
	var aspectID string
	if automation.Current != nil {
		workflowID = automation.Current.Id
	} else if automation.Draft != nil {
		workflowID = automation.Draft.Id
	}

	// Get entity/aspect ID
	if entity := automation.Entity; entity != nil {
		switch e := entity.(type) {
		case *eclient.GetAutomationDetailsOrganizationAutomationEntityAspectApp:
			aspectID = e.Id
		case *eclient.GetAutomationDetailsOrganizationAutomationEntityAspectElement:
			aspectID = e.Id
		case *eclient.GetAutomationDetailsOrganizationAutomationEntityAspectTask:
			aspectID = e.Id
		}
	}

	// Fetch full task details if we have a workflow
	var fullWorkflow *eclient.WorkflowFullDetails
	if workflowID != "" {
		if aspectID != "" {
			fullWorkflow, err = c.GetWorkflowWithFullTasks(ctx, aspectID, workflowID)
		} else {
			fullWorkflow, err = c.GetWorkflowWithFullTasksByWorkflowID(ctx, workflowID)
		}
		if err != nil {
			// Non-fatal: fall back to basic info
			fullWorkflow = nil
		}
	}

	// Check for --hcl flag
	hclOutput, _ := cmd.Flags().GetBool("hcl")
	if hclOutput {
		if fullWorkflow == nil {
			return fmt.Errorf("cannot generate HCL: workflow details not available")
		}

		// Convert to discovery automation
		auto := discovery.ConvertWorkflowToAutomation(
			automation.Id,
			automation.Name,
			workflowID,
			string(automation.Status),
			fullWorkflow,
		)

		// Get aspect name for resource naming
		aspectName := automation.Name // fallback
		if entity := automation.Entity; entity != nil {
			switch e := entity.(type) {
			case *eclient.GetAutomationDetailsOrganizationAutomationEntityAspectApp:
				aspectName = e.Name
			case *eclient.GetAutomationDetailsOrganizationAutomationEntityAspectElement:
				aspectName = e.Name
			case *eclient.GetAutomationDetailsOrganizationAutomationEntityAspectTask:
				aspectName = e.Name
			}
		}

		// Enrich with available references for proper refs["Name"] syntax
		discovery.EnrichAutomationRefs(ctx, c, aspectID, &auto)

		// Generate HCL
		hcl := export.ExportSingleAutomationHCL(aspectID, aspectName, auto)
		fmt.Print(hcl)
		return nil
	}

	// Build result for JSON output
	result := map[string]any{
		"id":     automation.Id,
		"name":   automation.Name,
		"status": automation.Status,
	}

	if automation.Current != nil {
		current := automation.Current
		workflow := map[string]any{
			"id":       current.Id,
			"version":  current.Version,
			"status":   current.Status,
			"terminal": current.Terminal,
		}

		if fullWorkflow != nil {
			// Include full RawData for triggers (RawData has json:"-" tag, so merge manually)
			triggers := make([]map[string]interface{}, len(fullWorkflow.Triggers))
			for i, t := range fullWorkflow.Triggers {
				merged := make(map[string]interface{})
				for k, v := range t.RawData {
					merged[k] = v
				}
				merged["id"] = t.ID
				merged["__typename"] = t.Typename
				triggers[i] = merged
			}
			workflow["triggers"] = triggers

			// Include full RawData for tasks
			tasks := make([]map[string]interface{}, len(fullWorkflow.Tasks))
			for i, t := range fullWorkflow.Tasks {
				merged := make(map[string]interface{})
				for k, v := range t.RawData {
					merged[k] = v
				}
				merged["id"] = t.ID
				merged["name"] = t.Name
				merged["__typename"] = t.Typename
				if t.Previous != nil {
					merged["previous"] = map[string]interface{}{"id": t.Previous.ID}
				}
				if t.Next != nil {
					merged["next"] = map[string]interface{}{"id": t.Next.ID}
				}
				tasks[i] = merged
			}
			workflow["tasks"] = tasks
			workflow["outputs"] = fullWorkflow.Outputs
		}

		result["current"] = workflow
	}

	if automation.Draft != nil {
		result["draft"] = map[string]any{
			"id":      automation.Draft.Id,
			"version": automation.Draft.Version,
			"status":  automation.Draft.Status,
		}
	}

	if isJSONOutput(cmd) {
		return outputJSON(result)
	}

	// Build tree for display
	tree := ui.NewTree(fmt.Sprintf("%s (%s)", automation.Name, automation.Id))
	tree.Root.AddChildWithLabel("Status", string(automation.Status))

	if automation.Current != nil {
		current := automation.Current
		workflowNode := tree.Root.AddChildWithLabel("Current Workflow", fmt.Sprintf("v%d (%s)", current.Version, current.Status))
		workflowNode.AddChildWithLabel("ID", current.Id)

		if fullWorkflow != nil {
			renderFullWorkflow(workflowNode, fullWorkflow)
		}
	}

	if automation.Draft != nil {
		draftNode := tree.Root.AddChildWithLabel("Draft", fmt.Sprintf("v%d (%s)", automation.Draft.Version, automation.Draft.Status))
		draftNode.AddChildWithLabel("ID", automation.Draft.Id)
	}

	fmt.Println()
	fmt.Println(tree.Render())
	fmt.Println()

	fmt.Println(ui.SubtitleStyle.Render("To see value references:"))
	fmt.Printf("  %s After 'tofu apply': run 'ei refs' to see refs from state\n", ui.RenderBullet())
	fmt.Printf("  %s During apply: run 'TF_LOG=DEBUG tofu apply' to see refs as created\n", ui.RenderBullet())
	fmt.Println()

	return nil
}
