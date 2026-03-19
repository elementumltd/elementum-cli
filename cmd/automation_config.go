// Copyright (c) ElementumAI, Inc.
// SPDX-License-Identifier: MPL-2.0

package cmd

import (
	"context"
	"fmt"

	"github.com/elementumltd/elementum-cli/analysis"
	"github.com/elementumltd/elementum-cli/auth"
	"github.com/elementumltd/elementum-cli/ui"
	"github.com/elementumltd/elementum-cli/internal/client"
	"github.com/spf13/cobra"
)

var automationConfigCmd = &cobra.Command{
	Use:   "config <automation-id>",
	Short: "Render automation configuration flow",
	Long: `Display the full configuration of an automation as a visual flow.

Shows the trigger and all tasks in execution order, with proper
rendering of switch branches and for_each loops.

Output modes:
  (default)     CLI visual flow with connectors and nesting
  --json        Full structured JSON of the automation config
  --timeline    Alias for default CLI visual flow
  --html        Interactive HTML visualization opened in browser

Examples:
  ei automation config abc-123-uuid
  ei automation config abc-123-uuid --json
  ei automation config abc-123-uuid --html
  ei automation config abc-123-uuid --timeline`,
	Args: cobra.ExactArgs(1),
	RunE: runAutomationConfig,
}

var (
	autoConfigTimeline bool
	autoConfigHTML     bool
)

func init() {
	automationConfigCmd.Flags().BoolVar(&autoConfigTimeline, "timeline", false, "Show CLI visual flow (same as default)")
	automationConfigCmd.Flags().BoolVar(&autoConfigHTML, "html", false, "Open interactive HTML visualization in browser")
	automationsCmd.AddCommand(automationConfigCmd)
}

func runAutomationConfig(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	automationID := args[0]

	c, err := auth.GetClientFromCmd(cmd)
	if err != nil {
		return err
	}

	if !isJSONOutput(cmd) && !autoConfigHTML {
		fmt.Println(ui.InfoStyle.Render(fmt.Sprintf("Loading automation: %s", automationID)))
	}

	// Fetch automation details
	resp, err := client.GetAutomationDetails(ctx, c.Genqlient(), automationID)
	if err != nil {
		return fmt.Errorf("failed to get automation: %w", err)
	}

	automation := resp.Organization.Automation
	if automation == nil {
		return fmt.Errorf("automation not found: %s", automationID)
	}

	// Determine workflow to render
	var workflowID string
	var aspectID string
	if automation.Current != nil {
		workflowID = automation.Current.Id
	} else if automation.Draft != nil {
		workflowID = automation.Draft.Id
	}

	if entity := automation.Entity; entity != nil {
		switch e := entity.(type) {
		case *client.GetAutomationDetailsOrganizationAutomationEntityAspectApp:
			aspectID = e.Id
		case *client.GetAutomationDetailsOrganizationAutomationEntityAspectElement:
			aspectID = e.Id
		case *client.GetAutomationDetailsOrganizationAutomationEntityAspectTask:
			aspectID = e.Id
		}
	}

	if workflowID == "" {
		return fmt.Errorf("automation has no published or draft workflow")
	}

	// Fetch workflow with nested children for visualization
	vizResp, err := client.GetWorkflowForVisualization(ctx, c.Genqlient(), workflowID)
	if err != nil {
		return fmt.Errorf("failed to fetch workflow details: %w", err)
	}
	if vizResp.Organization.Workflow == nil {
		return fmt.Errorf("workflow not found: %s", workflowID)
	}

	// aspectID is used for debug output only now
	_ = aspectID

	// Build structured config from genqlient response
	cfg := analysis.BuildAutomationConfigFromGenqlient(
		automation.Id,
		automation.Name,
		string(automation.Status),
		vizResp.Organization.Workflow,
	)

	// Route to output format
	if isJSONOutput(cmd) {
		return outputJSON(cfg)
	}
	if autoConfigHTML {
		return analysis.RenderAutomationHTML(cfg)
	}

	// Default + --timeline both render CLI flow
	analysis.RenderAutomationTimeline(cfg)

	fmt.Println(ui.MutedStyle.Render("Use --html for interactive visualization, --json for structured data"))
	fmt.Println()

	return nil
}

// RenderAutomationConfigText renders a text summary of the automation config.
// This is used by the default and timeline modes as a shared text renderer.
func RenderAutomationConfigText(cfg *analysis.AutomationConfig) {
	fmt.Println()
	fmt.Println(ui.TitleStyle.Render(fmt.Sprintf("Automation: %s", cfg.AutomationName)))

	printDetail("Status:", cfg.Status)
	printDetail("Workflow:", cfg.WorkflowID)
	if cfg.Version > 0 {
		printDetail("Version:", fmt.Sprintf("v%d", cfg.Version))
	}
	fmt.Println()

	if cfg.Trigger != nil {
		fmt.Println(ui.SubtitleStyle.Render("Trigger"))
		triggerTable := ui.NewTable([]string{"TYPE", "ID"})
		triggerTable.AddRow(cfg.Trigger.Type, cfg.Trigger.ID)
		fmt.Println(triggerTable.RenderSimple())
		fmt.Println()
	}

	if len(cfg.Tasks) > 0 {
		fmt.Println(ui.SubtitleStyle.Render(fmt.Sprintf("Tasks (%d top-level, %d total)", cfg.Summary.TopLevelTasks, cfg.Summary.TotalTasks)))
		taskTable := ui.NewTable([]string{"#", "NAME", "TYPE", "KIND"})
		flattenTasksToTable(cfg.Tasks, taskTable, "")
		fmt.Println(taskTable.RenderSimple())
		fmt.Println()
	}

	if len(cfg.Outputs) > 0 {
		fmt.Println(ui.SubtitleStyle.Render("Outputs"))
		for _, o := range cfg.Outputs {
			fmt.Printf("  %s %s\n", ui.RenderBullet(), o)
		}
		fmt.Println()
	}

	// Summary stats
	s := cfg.Summary
	fmt.Println(ui.SubtitleStyle.Render("Summary"))
	printDetail("Trigger:", s.TriggerType)
	printDetail("Tasks:", fmt.Sprintf("%d top-level, %d total", s.TopLevelTasks, s.TotalTasks))
	if s.SwitchCount > 0 {
		printDetail("Switches:", fmt.Sprintf("%d (%d cases)", s.SwitchCount, s.TotalCases))
	}
	if s.ForEachCount > 0 {
		printDetail("For Each:", fmt.Sprintf("%d loop(s)", s.ForEachCount))
	}
	if s.OutputCount > 0 {
		printDetail("Outputs:", fmt.Sprintf("%d", s.OutputCount))
	}
	fmt.Println()
}

func flattenTasksToTable(tasks []*analysis.AutomationNode, table *ui.Table, prefix string) {
	for _, t := range tasks {
		idx := prefix + t.Index
		name := t.Name
		if name == "" {
			name = t.Type
		}
		table.AddRow(idx, name, t.Type, string(t.NodeKind))

		if t.NodeKind == analysis.NodeSwitch {
			for _, c := range t.Cases {
				label := c.Label
				if label == "" {
					label = "Default"
				}
				table.AddRow("", fmt.Sprintf("  Case: %s", label), "", "case")
				flattenTasksToTable(c.Tasks, table, idx)
			}
		}

		if t.NodeKind == analysis.NodeForEach {
			if len(t.LoopBody) > 0 {
				table.AddRow("", "  [Loop Body]", "", "loop")
				flattenTasksToTable(t.LoopBody, table, idx)
			}
		}
	}
}
