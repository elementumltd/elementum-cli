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

	"github.com/elementumltd/elementum-cli/auth"
	"github.com/elementumltd/elementum-cli/internal/client"
	"github.com/elementumltd/elementum-cli/state"
	"github.com/elementumltd/elementum-cli/ui"
	"github.com/spf13/cobra"
)

var refsCmd = &cobra.Command{
	Use:   "refs [automation-id]",
	Short: "Show available value references from Terraform state or remote API",
	Long: `Display value references (refs) from Elementum resources.

By default, reads from terraform.tfstate in the current directory.
Use --from-remote to query refs directly from the Elementum API.

This command helps you discover what refs are available for use in
automation tasks, filters, and other value reference fields.

Examples:
  # From state (default)
  ei refs
  ei refs --state-file ./terraform.tfstate
  ei refs --resource elementum_record_created_trigger.my_trigger

  # From remote API (requires authentication)
  ei refs --from-remote <automation-id>
  ei refs --from-remote <automation-id> --after <task-id>
  ei refs --from-remote <automation-id> --json`,
	RunE: runRefs,
}

func init() {
	refsCmd.Flags().String("state-file", "", "Path to terraform.tfstate file (default: ./terraform.tfstate)")
	refsCmd.Flags().String("resource", "", "Filter to specific resource by name or address pattern")
	refsCmd.Flags().Bool("from-remote", false, "Query refs from remote server (requires authentication)")
	refsCmd.Flags().String("after", "", "Show refs available after a specific task (for --from-remote)")
}

// GetRefsCmd returns the refs command for registration
func GetRefsCmd() *cobra.Command {
	return refsCmd
}

func runRefs(cmd *cobra.Command, args []string) error {
	fromRemote, _ := cmd.Flags().GetBool("from-remote")

	if fromRemote {
		if len(args) == 0 {
			return fmt.Errorf("automation ID is required when using --from-remote\nUsage: ei refs --from-remote <automation-id>")
		}
		return runRefsFromRemote(cmd, args[0])
	}

	return runRefsFromState(cmd)
}

func runRefsFromRemote(cmd *cobra.Command, automationID string) error {
	ctx := context.Background()
	afterTaskID, _ := cmd.Flags().GetString("after")

	// Get authenticated client
	c, err := auth.GetClientFromCmd(cmd)
	if err != nil {
		return err
	}

	// Show status message (skip for JSON output)
	if !isJSONOutput(cmd) {
		fmt.Println(ui.InfoStyle.Render(fmt.Sprintf("⠿ Fetching refs for automation: %s", automationID)))
	}

	// First, get automation details to find workflow ID and aspect ID
	automationResp, err := client.GetAutomationDetails(ctx, c.Genqlient(), automationID)
	if err != nil {
		return fmt.Errorf("failed to get automation details: %w", err)
	}

	automation := automationResp.Organization.Automation
	if automation == nil {
		return fmt.Errorf("automation not found: %s", automationID)
	}

	// Get aspect ID from entity
	if automation.Entity == nil {
		return fmt.Errorf("automation has no associated entity (app)")
	}

	// Extract aspect ID based on entity type
	var aspectID string
	switch entity := automation.Entity.(type) {
	case *client.GetAutomationDetailsOrganizationAutomationEntityAspectApp:
		aspectID = entity.Id
	case *client.GetAutomationDetailsOrganizationAutomationEntityAspectElement:
		aspectID = entity.Id
	case *client.GetAutomationDetailsOrganizationAutomationEntityAspectTask:
		aspectID = entity.Id
	default:
		return fmt.Errorf("unsupported entity type for automation")
	}

	// Get workflow ID from current workflow
	if automation.Current == nil {
		return fmt.Errorf("automation has no current workflow (draft only?)")
	}
	workflowID := automation.Current.Id

	// Prepare optional parentTaskId parameter
	var parentTaskID *string
	if afterTaskID != "" {
		parentTaskID = &afterTaskID
	}

	// Query available references
	refsResp, err := client.GetWorkflowAvailableReferences(ctx, c.Genqlient(), workflowID, parentTaskID, aspectID, nil, nil)
	if err != nil {
		return fmt.Errorf("failed to get workflow references: %w", err)
	}

	// Extract refs from response
	aspect := refsResp.Organization.Aspect
	if aspect == nil {
		return fmt.Errorf("aspect not found")
	}

	// Get workflow from aspect (need to type switch based on aspect type)
	var workflow *client.GetWorkflowAvailableReferencesOrganizationAspectWorkflow
	switch a := (*aspect).(type) {
	case *client.GetWorkflowAvailableReferencesOrganizationAspectAspectApp:
		workflow = a.Workflow
	case *client.GetWorkflowAvailableReferencesOrganizationAspectAspectElement:
		workflow = a.Workflow
	case *client.GetWorkflowAvailableReferencesOrganizationAspectAspectTask:
		workflow = a.Workflow
	default:
		return fmt.Errorf("unsupported aspect type")
	}

	if workflow == nil {
		return fmt.Errorf("workflow not found in aspect")
	}

	// Build result structure for JSON output
	type refProperty struct {
		Name      string `json:"name"`
		Type      string `json:"type"`
		Multiple  bool   `json:"multiple"`
		GroupName string `json:"group_name"`
	}

	type remoteRefsResult struct {
		AutomationID   string        `json:"automation_id"`
		AutomationName string        `json:"automation_name"`
		WorkflowID     string        `json:"workflow_id"`
		AfterTaskID    string        `json:"after_task_id,omitempty"`
		Refs           []refProperty `json:"refs"`
	}

	result := remoteRefsResult{
		AutomationID:   automationID,
		AutomationName: automation.Name,
		WorkflowID:     workflowID,
		AfterTaskID:    afterTaskID,
		Refs:           []refProperty{},
	}

	// Flatten refs from groups
	for _, group := range workflow.AvailableReferences {
		for _, prop := range group.Properties {
			result.Refs = append(result.Refs, refProperty{
				Name:      prop.Name,
				Type:      string(prop.FieldType),
				Multiple:  prop.Multiple,
				GroupName: group.Name,
			})
		}
	}

	// JSON output
	if isJSONOutput(cmd) {
		return outputJSON(result)
	}

	// No refs found
	if len(result.Refs) == 0 {
		fmt.Println()
		fmt.Println(ui.WarningStyle.Render("No refs available at this position in the workflow."))
		fmt.Println()
		return nil
	}

	// Display refs
	fmt.Println()
	positionLabel := "after trigger"
	if afterTaskID != "" {
		positionLabel = fmt.Sprintf("after task: %s", afterTaskID)
	}
	fmt.Println(ui.MutedStyle.Render(fmt.Sprintf("Refs available %s:", positionLabel)))
	fmt.Println()

	// Group refs by group name for better display
	currentGroup := ""
	for _, ref := range result.Refs {
		if ref.GroupName != currentGroup {
			if currentGroup != "" {
				fmt.Println()
			}
			fmt.Println(ui.LabelStyle.Render(ref.GroupName))
			currentGroup = ref.GroupName
		}

		typeStr := ref.Type
		if ref.Multiple {
			typeStr += "[]"
		}
		fmt.Printf("├─ %-24s %s\n", ref.Name, ui.MutedStyle.Render(typeStr))
	}
	fmt.Println()

	// Usage hint
	fmt.Println(ui.SubtitleStyle.Render("Usage in Terraform:"))
	fmt.Printf("  refs[\"%s\"]\n", result.Refs[0].Name)
	fmt.Println()

	return nil
}

func runRefsFromState(cmd *cobra.Command) error {
	stateFilePath, _ := cmd.Flags().GetString("state-file")
	resourceFilter, _ := cmd.Flags().GetString("resource")

	// Find state file
	statePath, err := state.FindStateFile(stateFilePath)
	if err != nil {
		return err
	}

	// Read and parse state file
	tfState, err := state.ReadStateFile(statePath)
	if err != nil {
		return err
	}

	// Extract refs
	stateRefs, err := state.ExtractRefs(tfState, resourceFilter)
	if err != nil {
		return err
	}
	stateRefs.StateFile = statePath

	// JSON output
	if isJSONOutput(cmd) {
		return outputJSON(stateRefs)
	}

	// No refs found
	if len(stateRefs.Resources) == 0 {
		fmt.Println()
		if resourceFilter != "" {
			fmt.Println(ui.WarningStyle.Render(fmt.Sprintf("No refs found matching '%s' in state file.", resourceFilter)))
		} else {
			fmt.Println(ui.WarningStyle.Render("No Elementum trigger or task resources with refs found in state."))
			fmt.Println()
			fmt.Println(ui.MutedStyle.Render("Refs are populated after 'tofu apply' creates resources."))
			fmt.Println(ui.MutedStyle.Render("Make sure you have automation triggers or tasks in your configuration."))
		}
		fmt.Println()
		return nil
	}

	// Display refs
	fmt.Println()
	fmt.Println(ui.MutedStyle.Render(fmt.Sprintf("Reading refs from: %s", statePath)))
	fmt.Println()

	for _, resource := range stateRefs.Resources {
		renderResourceRefs(resource)
	}

	// Usage hint
	fmt.Println(ui.SubtitleStyle.Render("Usage in Terraform:"))
	if len(stateRefs.Resources) > 0 {
		firstResource := stateRefs.Resources[0]
		if len(firstResource.Refs) > 0 {
			fmt.Printf("  %s.refs[\"%s\"]\n", firstResource.Address, firstResource.Refs[0].Name)
		}
	}
	fmt.Println()

	return nil
}

func renderResourceRefs(resource state.ResourceRefs) {
	// Resource header
	fmt.Println(ui.LabelStyle.Render(resource.Address))

	// Tree-style refs list
	for i, ref := range resource.Refs {
		prefix := "├─"
		if i == len(resource.Refs)-1 {
			prefix = "└─"
		}

		typeStr := state.FormatRefType(ref)
		// Pad the name and type for alignment
		fmt.Printf("%s %-24s %s\n", prefix, ref.Name, ui.MutedStyle.Render(typeStr))
	}
	fmt.Println()
}
