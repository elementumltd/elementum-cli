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
	"github.com/elementumltd/elementum-cli/ui"
	"github.com/elementumltd/elementum-cli/internal/client"
	"github.com/spf13/cobra"
)

var interventionsCmd = &cobra.Command{
	Use:   "interventions",
	Short: "Manage automation interventions",
	Long: `View automation interventions that require human attention.

Interventions are created when automations fail and need manual review.`,
}

var interventionsListCmd = &cobra.Command{
	Use:   "list <aspect-namespace-or-id>",
	Short: "List interventions for an aspect",
	Long: `List automation interventions for an app, element, or task.

Examples:
  # List all interventions for an app
  ei interventions list my-app

  # Filter by status
  ei interventions list my-app --status OPEN

  # Filter by automation name (partial match)
  ei interventions list my-app --automation "Process Orders"

  # Limit results
  ei interventions list my-app --limit 50

  # JSON output
  ei interventions list my-app --json`,
	Args: cobra.ExactArgs(1),
	RunE: runInterventionsList,
}

var interventionsShowCmd = &cobra.Command{
	Use:   "show <aspect-namespace-or-id> <intervention-id>",
	Short: "Show intervention details",
	Long: `Show detailed information about a specific intervention.

Examples:
  # Show intervention details
  ei interventions show my-app abc-123-intervention-id

  # JSON output
  ei interventions show my-app abc-123-intervention-id --json`,
	Args: cobra.ExactArgs(2),
	RunE: runInterventionsShow,
}

func init() {
	// List command flags
	interventionsListCmd.Flags().String("status", "", "Filter by status: OPEN, IN_PROGRESS, RESOLVED, IGNORED")
	interventionsListCmd.Flags().String("automation", "", "Filter by automation name (partial match)")
	interventionsListCmd.Flags().Int("limit", 100, "Max interventions to fetch (filtering done client-side)")

	// Register subcommands
	interventionsCmd.AddCommand(interventionsListCmd)
	interventionsCmd.AddCommand(interventionsShowCmd)
}

// GetInterventionsCmd returns the interventions command for registration
func GetInterventionsCmd() *cobra.Command {
	return interventionsCmd
}

// Intervention represents a normalized intervention for display
type Intervention struct {
	ID                string  `json:"id"`
	Status            string  `json:"status"`
	FailureCode       string  `json:"failureCode"`
	FailureReason     *string `json:"failureReason"`
	Resolution        *string `json:"resolution"`
	ResolutionSummary *string `json:"resolutionSummary"`
	Notes             *string `json:"notes"`
	CreatedAt         string  `json:"createdAt"`
	UpdatedAt         string  `json:"updatedAt"`
	FailureAt         string  `json:"failureAt"`
	ClosedAt          *string `json:"closedAt"`
	AutomationID      string  `json:"automationId"`
	AutomationName    string  `json:"automationName"`
}

func runInterventionsList(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	namespace := args[0]

	c, err := auth.GetClientFromCmd(cmd)
	if err != nil {
		return err
	}

	// Resolve aspect by namespace
	aspectID, _, err := resolveAspectByNamespace(ctx, c, namespace)
	if err != nil {
		return fmt.Errorf("failed to resolve aspect: %w", err)
	}

	// Get filter flags for client-side filtering
	statusFilter, _ := cmd.Flags().GetString("status")
	automationFilter, _ := cmd.Flags().GetString("automation")
	limit, _ := cmd.Flags().GetInt("limit")

	if !isJSONOutput(cmd) {
		fmt.Println(ui.InfoStyle.Render("⠿ Loading interventions..."))
	}

	// Call API (no server-side filter)
	resp, err := client.GetAspectInterventions(ctx, c.Genqlient(), aspectID, nil, nil, &limit, nil)
	if err != nil {
		return fmt.Errorf("failed to get interventions: %w", err)
	}

	// Extract interventions from polymorphic response
	interventions, total, aspectName, err := extractInterventionsFromResponse(resp)
	if err != nil {
		return err
	}

	// Apply client-side filtering
	interventions = filterInterventions(interventions, statusFilter, automationFilter)

	// JSON output
	if isJSONOutput(cmd) {
		return outputJSON(map[string]any{
			"aspect": map[string]any{
				"id":   aspectID,
				"name": aspectName,
			},
			"total":         total,
			"interventions": interventions,
		})
	}

	// Display header
	fmt.Println()
	fmt.Println(ui.TitleStyle.Render(fmt.Sprintf("Interventions for %s", aspectName)))
	fmt.Println()

	if len(interventions) == 0 {
		fmt.Println(ui.SuccessStyle.Render("No interventions found."))
		fmt.Println()
		return nil
	}

	// Display table
	table := ui.NewTable([]string{"STATUS", "AUTOMATION", "FAILURE CODE", "CREATED"})
	for _, intervention := range interventions {
		table.AddRow(
			styledInterventionStatus(intervention.Status),
			ui.Truncate(intervention.AutomationName, 30),
			formatFailureCode(intervention.FailureCode),
			formatTimestamp(intervention.CreatedAt),
		)
	}

	fmt.Println(table.RenderSimple())
	if len(interventions) < total {
		fmt.Println(ui.MutedStyle.Render(fmt.Sprintf("Showing: %d of %d interventions", len(interventions), total)))
	} else {
		fmt.Println(ui.MutedStyle.Render(fmt.Sprintf("Total: %d interventions", len(interventions))))
	}
	fmt.Println()

	return nil
}

func runInterventionsShow(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	namespace := args[0]
	interventionID := args[1]

	c, err := auth.GetClientFromCmd(cmd)
	if err != nil {
		return err
	}

	// Resolve aspect by namespace
	aspectID, _, err := resolveAspectByNamespace(ctx, c, namespace)
	if err != nil {
		return fmt.Errorf("failed to resolve aspect: %w", err)
	}

	if !isJSONOutput(cmd) {
		fmt.Println(ui.InfoStyle.Render("⠿ Loading intervention details..."))
	}

	// Call API
	resp, err := client.GetAspectIntervention(ctx, c.Genqlient(), aspectID, interventionID)
	if err != nil {
		return fmt.Errorf("failed to get intervention: %w", err)
	}

	// Extract intervention from polymorphic response
	intervention, aspectName, err := extractInterventionFromResponse(resp)
	if err != nil {
		return err
	}

	if intervention == nil {
		return fmt.Errorf("intervention not found: %s", interventionID)
	}

	// JSON output
	if isJSONOutput(cmd) {
		return outputJSON(map[string]any{
			"aspect": map[string]any{
				"id":   aspectID,
				"name": aspectName,
			},
			"intervention": intervention,
		})
	}

	// Display detail view
	displayInterventionDetail(intervention)

	return nil
}

// filterInterventions applies client-side filtering to interventions
func filterInterventions(interventions []Intervention, statusFilter, automationFilter string) []Intervention {
	if statusFilter == "" && automationFilter == "" {
		return interventions
	}

	var filtered []Intervention
	for _, i := range interventions {
		// Filter by status (exact match, case-insensitive)
		if statusFilter != "" && !strings.EqualFold(i.Status, statusFilter) {
			continue
		}

		// Filter by automation name (partial match, case-insensitive)
		if automationFilter != "" && !strings.Contains(strings.ToLower(i.AutomationName), strings.ToLower(automationFilter)) {
			continue
		}

		filtered = append(filtered, i)
	}

	return filtered
}

// extractInterventionsFromResponse extracts interventions from the polymorphic response
func extractInterventionsFromResponse(resp *client.GetAspectInterventionsResponse) ([]Intervention, int, string, error) {
	if resp == nil {
		return nil, 0, "", fmt.Errorf("no response received")
	}

	aspect := resp.Organization.Aspect
	if aspect == nil {
		return nil, 0, "", fmt.Errorf("aspect not found")
	}

	var interventions []Intervention
	var total int
	var aspectName string

	switch a := (*aspect).(type) {
	case *client.GetAspectInterventionsOrganizationAspectAspectApp:
		aspectName = a.Name
		if a.Health != nil && a.Health.Automations != nil && a.Health.Automations.Interventions != nil {
			total = a.Health.Automations.Interventions.Total
			for _, edge := range a.Health.Automations.Interventions.Edges {
				interventions = append(interventions, convertInterventionApp(edge.Node))
			}
		}
	case *client.GetAspectInterventionsOrganizationAspectAspectElement:
		aspectName = a.Name
		if a.Health != nil && a.Health.Automations != nil && a.Health.Automations.Interventions != nil {
			total = a.Health.Automations.Interventions.Total
			for _, edge := range a.Health.Automations.Interventions.Edges {
				interventions = append(interventions, convertInterventionElement(edge.Node))
			}
		}
	case *client.GetAspectInterventionsOrganizationAspectAspectTask:
		aspectName = a.Name
		if a.Health != nil && a.Health.Automations != nil && a.Health.Automations.Interventions != nil {
			total = a.Health.Automations.Interventions.Total
			for _, edge := range a.Health.Automations.Interventions.Edges {
				interventions = append(interventions, convertInterventionTask(edge.Node))
			}
		}
	default:
		return nil, 0, "", fmt.Errorf("unsupported aspect type")
	}

	return interventions, total, aspectName, nil
}

// extractInterventionFromResponse extracts a single intervention from the polymorphic response
func extractInterventionFromResponse(resp *client.GetAspectInterventionResponse) (*Intervention, string, error) {
	if resp == nil {
		return nil, "", fmt.Errorf("no response received")
	}

	aspect := resp.Organization.Aspect
	if aspect == nil {
		return nil, "", fmt.Errorf("aspect not found")
	}

	var intervention *Intervention
	var aspectName string

	switch a := (*aspect).(type) {
	case *client.GetAspectInterventionOrganizationAspectAspectApp:
		aspectName = a.Name
		if a.Health != nil && a.Health.Automations != nil && a.Health.Automations.Intervention != nil {
			i := convertInterventionShowApp(*a.Health.Automations.Intervention)
			intervention = &i
		}
	case *client.GetAspectInterventionOrganizationAspectAspectElement:
		aspectName = a.Name
		if a.Health != nil && a.Health.Automations != nil && a.Health.Automations.Intervention != nil {
			i := convertInterventionShowElement(*a.Health.Automations.Intervention)
			intervention = &i
		}
	case *client.GetAspectInterventionOrganizationAspectAspectTask:
		aspectName = a.Name
		if a.Health != nil && a.Health.Automations != nil && a.Health.Automations.Intervention != nil {
			i := convertInterventionShowTask(*a.Health.Automations.Intervention)
			intervention = &i
		}
	default:
		return nil, "", fmt.Errorf("unsupported aspect type")
	}

	return intervention, aspectName, nil
}

// Conversion functions for each aspect type (list mode)
func convertInterventionApp(node client.GetAspectInterventionsOrganizationAspectAspectAppHealthAspectHealthAutomationsAutomationHealthInterventionsAutomationInterventionConnectionEdgesAutomationInterventionEdgeNodeAutomationIntervention) Intervention {
	return Intervention{
		ID:                node.Id,
		Status:            string(node.Status),
		FailureCode:       string(node.FailureCode),
		FailureReason:     node.FailureReason,
		Resolution:        ptrString(node.Resolution),
		ResolutionSummary: node.ResolutionSummary,
		Notes:             node.Notes,
		CreatedAt:         node.CreatedAt,
		UpdatedAt:         node.UpdatedAt,
		FailureAt:         node.FailureAt,
		ClosedAt:          node.ClosedAt,
		AutomationID:      node.Automation.Id,
		AutomationName:    node.Automation.Name,
	}
}

func convertInterventionElement(node client.GetAspectInterventionsOrganizationAspectAspectElementHealthAspectHealthAutomationsAutomationHealthInterventionsAutomationInterventionConnectionEdgesAutomationInterventionEdgeNodeAutomationIntervention) Intervention {
	return Intervention{
		ID:                node.Id,
		Status:            string(node.Status),
		FailureCode:       string(node.FailureCode),
		FailureReason:     node.FailureReason,
		Resolution:        ptrString(node.Resolution),
		ResolutionSummary: node.ResolutionSummary,
		Notes:             node.Notes,
		CreatedAt:         node.CreatedAt,
		UpdatedAt:         node.UpdatedAt,
		FailureAt:         node.FailureAt,
		ClosedAt:          node.ClosedAt,
		AutomationID:      node.Automation.Id,
		AutomationName:    node.Automation.Name,
	}
}

func convertInterventionTask(node client.GetAspectInterventionsOrganizationAspectAspectTaskHealthAspectHealthAutomationsAutomationHealthInterventionsAutomationInterventionConnectionEdgesAutomationInterventionEdgeNodeAutomationIntervention) Intervention {
	return Intervention{
		ID:                node.Id,
		Status:            string(node.Status),
		FailureCode:       string(node.FailureCode),
		FailureReason:     node.FailureReason,
		Resolution:        ptrString(node.Resolution),
		ResolutionSummary: node.ResolutionSummary,
		Notes:             node.Notes,
		CreatedAt:         node.CreatedAt,
		UpdatedAt:         node.UpdatedAt,
		FailureAt:         node.FailureAt,
		ClosedAt:          node.ClosedAt,
		AutomationID:      node.Automation.Id,
		AutomationName:    node.Automation.Name,
	}
}

// Conversion functions for show mode (single intervention)
func convertInterventionShowApp(node client.GetAspectInterventionOrganizationAspectAspectAppHealthAspectHealthAutomationsAutomationHealthInterventionAutomationIntervention) Intervention {
	return Intervention{
		ID:                node.Id,
		Status:            string(node.Status),
		FailureCode:       string(node.FailureCode),
		FailureReason:     node.FailureReason,
		Resolution:        ptrString(node.Resolution),
		ResolutionSummary: node.ResolutionSummary,
		Notes:             node.Notes,
		CreatedAt:         node.CreatedAt,
		UpdatedAt:         node.UpdatedAt,
		FailureAt:         node.FailureAt,
		ClosedAt:          node.ClosedAt,
		AutomationID:      node.Automation.Id,
		AutomationName:    node.Automation.Name,
	}
}

func convertInterventionShowElement(node client.GetAspectInterventionOrganizationAspectAspectElementHealthAspectHealthAutomationsAutomationHealthInterventionAutomationIntervention) Intervention {
	return Intervention{
		ID:                node.Id,
		Status:            string(node.Status),
		FailureCode:       string(node.FailureCode),
		FailureReason:     node.FailureReason,
		Resolution:        ptrString(node.Resolution),
		ResolutionSummary: node.ResolutionSummary,
		Notes:             node.Notes,
		CreatedAt:         node.CreatedAt,
		UpdatedAt:         node.UpdatedAt,
		FailureAt:         node.FailureAt,
		ClosedAt:          node.ClosedAt,
		AutomationID:      node.Automation.Id,
		AutomationName:    node.Automation.Name,
	}
}

func convertInterventionShowTask(node client.GetAspectInterventionOrganizationAspectAspectTaskHealthAspectHealthAutomationsAutomationHealthInterventionAutomationIntervention) Intervention {
	return Intervention{
		ID:                node.Id,
		Status:            string(node.Status),
		FailureCode:       string(node.FailureCode),
		FailureReason:     node.FailureReason,
		Resolution:        ptrString(node.Resolution),
		ResolutionSummary: node.ResolutionSummary,
		Notes:             node.Notes,
		CreatedAt:         node.CreatedAt,
		UpdatedAt:         node.UpdatedAt,
		FailureAt:         node.FailureAt,
		ClosedAt:          node.ClosedAt,
		AutomationID:      node.Automation.Id,
		AutomationName:    node.Automation.Name,
	}
}

// Helper function to convert resolution to pointer
func ptrString(r *client.AutomationInterventionResolution) *string {
	if r == nil {
		return nil
	}
	s := string(*r)
	return &s
}

// styledInterventionStatus returns a color-coded status string
func styledInterventionStatus(status string) string {
	switch status {
	case "OPEN":
		return ui.ErrorStyle.Render("OPEN")
	case "IN_PROGRESS":
		return ui.WarningStyle.Render("IN_PROGRESS")
	case "RESOLVED":
		return ui.SuccessStyle.Render("RESOLVED")
	case "IGNORED":
		return ui.MutedStyle.Render("IGNORED")
	default:
		return status
	}
}

// formatFailureCode formats a failure code for display
func formatFailureCode(code string) string {
	// Make it more readable by replacing underscores with spaces
	return strings.ReplaceAll(code, "_", " ")
}

// displayInterventionDetail renders detailed intervention information
func displayInterventionDetail(intervention *Intervention) {
	fmt.Println()
	fmt.Println(ui.TitleStyle.Render(fmt.Sprintf("Intervention %s", ui.Truncate(intervention.ID, 12))))
	fmt.Println()

	// Status section
	fmt.Printf("  %s  %s\n", ui.LabelStyle.Render("Status:"), styledInterventionStatus(intervention.Status))
	fmt.Printf("  %s  %s\n", ui.LabelStyle.Render("Automation:"), intervention.AutomationName)
	fmt.Printf("  %s  %s\n", ui.LabelStyle.Render("Automation ID:"), intervention.AutomationID)
	fmt.Println()

	// Failure section
	fmt.Println(ui.SubtitleStyle.Render("Failure Details"))
	fmt.Printf("  %s  %s\n", ui.LabelStyle.Render("Code:"), formatFailureCode(intervention.FailureCode))
	if intervention.FailureReason != nil && *intervention.FailureReason != "" {
		fmt.Printf("  %s  %s\n", ui.LabelStyle.Render("Reason:"), wordWrap(*intervention.FailureReason, 60))
	}
	fmt.Println()

	// Timeline section
	fmt.Println(ui.SubtitleStyle.Render("Timeline"))
	fmt.Printf("  %s  %s\n", ui.LabelStyle.Render("Created:"), formatTimestamp(intervention.CreatedAt))
	if intervention.FailureAt != "" {
		fmt.Printf("  %s  %s\n", ui.LabelStyle.Render("Failed At:"), formatTimestamp(intervention.FailureAt))
	}
	fmt.Printf("  %s  %s\n", ui.LabelStyle.Render("Updated:"), formatTimestamp(intervention.UpdatedAt))
	if intervention.ClosedAt != nil {
		fmt.Printf("  %s  %s\n", ui.LabelStyle.Render("Closed At:"), formatTimestamp(*intervention.ClosedAt))
	}
	fmt.Println()

	// Resolution section (if present)
	if intervention.Resolution != nil || intervention.ResolutionSummary != nil {
		fmt.Println(ui.SubtitleStyle.Render("Resolution"))
		if intervention.Resolution != nil {
			fmt.Printf("  %s  %s\n", ui.LabelStyle.Render("Type:"), formatResolution(*intervention.Resolution))
		}
		if intervention.ResolutionSummary != nil {
			fmt.Printf("  %s  %s\n", ui.LabelStyle.Render("Summary:"), wordWrap(*intervention.ResolutionSummary, 60))
		}
		fmt.Println()
	}

	// Notes section (if present)
	if intervention.Notes != nil && *intervention.Notes != "" {
		fmt.Println(ui.SubtitleStyle.Render("Notes"))
		fmt.Printf("  %s\n", wordWrap(*intervention.Notes, 70))
		fmt.Println()
	}
}

// formatResolution formats a resolution type for display
func formatResolution(resolution string) string {
	// Make it more readable
	result := strings.ReplaceAll(resolution, "_", " ")
	return strings.ToLower(result)
}

// wordWrap wraps text at the specified width
func wordWrap(text string, width int) string {
	if len(text) <= width {
		return text
	}

	var lines []string
	words := strings.Fields(text)
	currentLine := ""

	for _, word := range words {
		if len(currentLine)+len(word)+1 > width {
			if currentLine != "" {
				lines = append(lines, currentLine)
			}
			currentLine = word
		} else {
			if currentLine != "" {
				currentLine += " "
			}
			currentLine += word
		}
	}

	if currentLine != "" {
		lines = append(lines, currentLine)
	}

	return strings.Join(lines, "\n           ") // Indent continuation lines
}
