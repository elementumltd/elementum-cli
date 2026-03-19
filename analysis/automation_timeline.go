// Copyright (c) ElementumAI, Inc.
// SPDX-License-Identifier: MPL-2.0

package analysis

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	triggerStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("11")).Bold(true) // Yellow
	taskBlockStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("12")).Bold(true) // Cyan
	switchStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("13")).Bold(true) // Magenta
	forEachStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("14")).Bold(true) // Light cyan
	caseStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("13"))            // Magenta
	connectorStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))             // Gray
	indexStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))             // Gray
	typeTagStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Italic(true)
	detailKeyStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("14"))
	detailValStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("15"))
	titleBarStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12"))
	statusActive   = lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Bold(true)
	statusInactive = lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Bold(true)
	statusDraft    = lipgloss.NewStyle().Foreground(lipgloss.Color("11")).Bold(true)
	summaryLabel   = lipgloss.NewStyle().Foreground(lipgloss.Color("14")).Bold(true)
)

// RenderAutomationTimeline prints a CLI visual flow to stdout.
func RenderAutomationTimeline(cfg *AutomationConfig) {
	if cfg == nil {
		fmt.Println(mutedStyle.Render("No automation config to display."))
		return
	}

	fmt.Println()

	// Header
	styledStatus := renderStatus(cfg.Status)
	fmt.Printf("%s  %s\n",
		titleBarStyle.Render(cfg.AutomationName),
		styledStatus,
	)
	fmt.Println(connectorStyle.Render(strings.Repeat("─", 60)))
	fmt.Println()

	indent := "  "

	// Trigger
	if cfg.Trigger != nil {
		renderTriggerNode(cfg.Trigger, indent)
		fmt.Println(connectorStyle.Render(indent + "│"))
		fmt.Println(connectorStyle.Render(indent + "▼"))
	}

	// Tasks
	for i, task := range cfg.Tasks {
		renderTaskNode(task, indent)
		if i < len(cfg.Tasks)-1 {
			fmt.Println(connectorStyle.Render(indent + "│"))
			fmt.Println(connectorStyle.Render(indent + "▼"))
		}
	}

	// Outputs
	if len(cfg.Outputs) > 0 {
		fmt.Println()
		fmt.Println(connectorStyle.Render(indent + "│"))
		fmt.Println(connectorStyle.Render(indent + "▼"))
		fmt.Printf("%s%s %s\n",
			indent,
			triggerStyle.Render("⬡ OUTPUTS"),
			mutedStyle.Render(fmt.Sprintf("(%d)", len(cfg.Outputs))),
		)
		for _, o := range cfg.Outputs {
			fmt.Printf("%s  %s %s\n", indent, connectorStyle.Render("•"), detailValStyle.Render(o))
		}
	}

	fmt.Println()

	// Summary
	renderAutomationSummary(cfg)
}

func renderStatus(status string) string {
	upper := strings.ToUpper(status)
	switch upper {
	case "ACTIVE":
		return statusActive.Render(upper)
	case "INACTIVE":
		return statusInactive.Render(upper)
	default:
		return statusDraft.Render(upper)
	}
}

func renderTriggerNode(t *AutomationNode, indent string) {
	fmt.Printf("%s%s  %s\n",
		indent,
		triggerStyle.Render("⚡ TRIGGER"),
		detailValStyle.Render(t.Type),
	)
	renderNodeDetails(t.Details, indent+"  ")
}

func renderTaskNode(task *AutomationNode, indent string) {
	switch task.NodeKind {
	case NodeSwitch:
		renderSwitchNode(task, indent)
	case NodeForEach:
		renderForEachNode(task, indent)
	default:
		renderSimpleTask(task, indent)
	}
}

func renderSimpleTask(task *AutomationNode, indent string) {
	name := task.Name
	if name == "" {
		name = task.Type
	}
	fmt.Printf("%s%s %s  %s\n",
		indent,
		taskBlockStyle.Render("█"),
		indexStyle.Render(task.Index+"."),
		labelStyle.Render(name)+" "+typeTagStyle.Render("["+task.Type+"]"),
	)
	renderNodeDetailsForType(task.Type, task.Details, indent+"  ")
}

func renderSwitchNode(task *AutomationNode, indent string) {
	name := task.Name
	if name == "" {
		name = "Switch"
	}
	fmt.Printf("%s%s %s  %s\n",
		indent,
		switchStyle.Render("◆"),
		indexStyle.Render(task.Index+"."),
		labelStyle.Render(name)+" "+typeTagStyle.Render("[switch]"),
	)

	for i, c := range task.Cases {
		isLast := i == len(task.Cases)-1
		renderCaseBranch(c, indent, isLast, task.Index)
	}
}

func renderCaseBranch(c *SwitchCase, indent string, isLast bool, parentIdx string) {
	branch := "├"
	continuation := "│"
	if isLast {
		branch = "└"
		continuation = " "
	}

	label := c.Label
	if label == "" {
		label = "Default"
	}

	fmt.Printf("%s%s%s %s\n",
		indent,
		connectorStyle.Render(branch+"─── "),
		caseStyle.Render("Case:"),
		detailValStyle.Render(label),
	)

	caseIndent := indent + connectorStyle.Render(continuation) + "    "

	for i, t := range c.Tasks {
		renderTaskNode(t, caseIndent)
		if i < len(c.Tasks)-1 {
			fmt.Println(connectorStyle.Render(caseIndent + "│"))
			fmt.Println(connectorStyle.Render(caseIndent + "▼"))
		}
	}

	if len(c.Tasks) == 0 {
		fmt.Printf("%s%s\n", caseIndent, mutedStyle.Render("(empty)"))
	}

	if !isLast {
		fmt.Println(connectorStyle.Render(indent + "│"))
	}
}

func renderForEachNode(task *AutomationNode, indent string) {
	name := task.Name
	if name == "" {
		name = "For Each"
	}
	listLabel := task.LoopList
	if listLabel == "" {
		listLabel = "(list)"
	}

	fmt.Printf("%s%s %s  %s\n",
		indent,
		forEachStyle.Render("⟳"),
		indexStyle.Render(task.Index+"."),
		labelStyle.Render(name)+" "+typeTagStyle.Render("[for_each]"),
	)
	fmt.Printf("%s  %s %s\n",
		indent,
		detailKeyStyle.Render("list:"),
		detailValStyle.Render(listLabel),
	)

	if len(task.LoopBody) > 0 {
		boxWidth := 40
		fmt.Println(connectorStyle.Render(indent + "  │"))
		fmt.Printf("%s  %s\n", indent, connectorStyle.Render("┌"+strings.Repeat("─", boxWidth)+"┐"))

		bodyIndent := indent + "  " + connectorStyle.Render("│") + " "
		for i, t := range task.LoopBody {
			renderTaskNode(t, bodyIndent)
			if i < len(task.LoopBody)-1 {
				fmt.Println(connectorStyle.Render(bodyIndent + "│"))
				fmt.Println(connectorStyle.Render(bodyIndent + "▼"))
			}
		}

		fmt.Printf("%s  %s\n", indent, connectorStyle.Render("└"+strings.Repeat("─", boxWidth)+"┘"))
	}
}

func renderNodeDetails(details map[string]any, indent string) {
	renderNodeDetailsForType("", details, indent)
}

func renderNodeDetailsForType(taskType string, details map[string]any, indent string) {
	if len(details) == 0 {
		return
	}

	// Get priority fields if we know the task type
	priorityFields := TaskPriorityFields[taskType]
	shown := 0
	const maxDetails = 8

	// Show priority fields first
	shownKeys := make(map[string]bool)
	for _, k := range priorityFields {
		if shown >= maxDetails {
			break
		}
		v, ok := details[k]
		if !ok {
			continue
		}

		valStr := FormatValue(v)
		if valStr == "" {
			continue
		}

		displayName := GetFieldDisplayName(k)
		valStr = truncateForCLI(valStr, 70)

		fmt.Printf("%s%s %s\n", indent, detailKeyStyle.Render(displayName+":"), detailValStyle.Render(valStr))
		shown++
		shownKeys[k] = true
	}

	// Show remaining fields
	for k, v := range details {
		if shown >= maxDetails {
			remaining := len(details) - shown
			if remaining > 0 {
				fmt.Printf("%s%s\n", indent, mutedStyle.Render(fmt.Sprintf("... %d more fields", remaining)))
			}
			break
		}

		if shownKeys[k] {
			continue
		}

		valStr := FormatValue(v)
		if valStr == "" {
			continue
		}

		displayName := GetFieldDisplayName(k)
		valStr = truncateForCLI(valStr, 70)

		fmt.Printf("%s%s %s\n", indent, detailKeyStyle.Render(displayName+":"), detailValStyle.Render(valStr))
		shown++
		shownKeys[k] = true
	}
}

func truncateForCLI(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}

func renderAutomationSummary(cfg *AutomationConfig) {
	s := cfg.Summary
	fmt.Println(connectorStyle.Render(strings.Repeat("─", 60)))
	fmt.Println(summaryLabel.Render("Summary"))

	parts := []string{
		fmt.Sprintf("Trigger: %s", s.TriggerType),
		fmt.Sprintf("%d top-level tasks (%d total)", s.TopLevelTasks, s.TotalTasks),
	}
	if s.SwitchCount > 0 {
		parts = append(parts, fmt.Sprintf("%d switch (%d cases)", s.SwitchCount, s.TotalCases))
	}
	if s.ForEachCount > 0 {
		parts = append(parts, fmt.Sprintf("%d for_each loop(s)", s.ForEachCount))
	}
	if s.OutputCount > 0 {
		parts = append(parts, fmt.Sprintf("%d outputs", s.OutputCount))
	}

	for _, p := range parts {
		fmt.Printf("  %s %s\n", connectorStyle.Render("•"), p)
	}
	fmt.Println()
}
