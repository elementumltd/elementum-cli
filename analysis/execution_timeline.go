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

package analysis

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	execTitleStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12"))
	execSuccessStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Bold(true)
	execFailureStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Bold(true)
	execRunningStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("11")).Bold(true)
	execMutedStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	execBarFull      = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
	execBarEmpty     = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	execLabelStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("15"))
	execTypeStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Italic(true)
	execIOKeyStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("14"))
	execIOValStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("15"))
)

const (
	barWidth     = 14
	maxIOPreview = 100
)

// RenderExecutionTimeline prints a CLI visual timeline showing each action
// with horizontal duration bars and status indicators.
func RenderExecutionTimeline(exec *ExecutionAnalysis, showIO bool) {
	if exec == nil {
		fmt.Println(execMutedStyle.Render("No execution data to display."))
		return
	}

	fmt.Println()

	// Header
	statusStr := renderExecutionStatus(exec.Status)
	fmt.Printf("%s (%s)  v%d  %s\n",
		execTitleStyle.Render(fmt.Sprintf("Execution %s...", truncateID(exec.ExecutionID, 12))),
		statusStr,
		exec.Version,
		formatDurationMs(int64(exec.Duration)),
	)
	fmt.Println(execMutedStyle.Render(strings.Repeat("─", 60)))
	fmt.Println()

	// Trigger indicator
	fmt.Println(triggerStyle.Render("⚡ TRIGGER"))
	fmt.Println(connectorStyle.Render("│"))
	fmt.Println(connectorStyle.Render("▼"))

	// Find max duration for bar scaling
	var maxDur int64
	for _, a := range exec.Actions {
		if a.Duration != nil && int64(*a.Duration) > maxDur {
			maxDur = int64(*a.Duration)
		}
	}
	if maxDur == 0 {
		maxDur = 1000 // Default scale
	}

	// Render each action
	for i, action := range exec.Actions {
		renderActionRow(i+1, action, maxDur, showIO)
		if i < len(exec.Actions)-1 {
			fmt.Println(connectorStyle.Render("▼"))
		}
	}

	fmt.Println()
	fmt.Println(execMutedStyle.Render(strings.Repeat("─", 60)))

	// Summary
	s := exec.Summary
	parts := []string{
		fmt.Sprintf("%d actions", s.TotalActions),
	}
	if s.SuccessCount > 0 {
		parts = append(parts, execSuccessStyle.Render(fmt.Sprintf("%d success", s.SuccessCount)))
	}
	if s.FailureCount > 0 {
		parts = append(parts, execFailureStyle.Render(fmt.Sprintf("%d failure", s.FailureCount)))
	}
	if s.RunningCount > 0 {
		parts = append(parts, execRunningStyle.Render(fmt.Sprintf("%d running", s.RunningCount)))
	}
	parts = append(parts, fmt.Sprintf("%s total", formatDurationMs(s.TotalDurationMs)))

	fmt.Printf("Summary: %s\n", strings.Join(parts, ", "))
	fmt.Println()
}

func renderActionRow(idx int, action ActionAnalysis, maxDur int64, showIO bool) {
	// Status indicator
	statusIcon := "█"
	var statusStyle lipgloss.Style
	switch action.Status {
	case "SUCCESS":
		statusStyle = execSuccessStyle
	case "FAILURE":
		statusStyle = execFailureStyle
	case "RUNNING":
		statusStyle = execRunningStyle
		statusIcon = "▶"
	default:
		statusStyle = execMutedStyle
	}

	// Name
	name := action.Name
	if name == "" {
		name = action.Type
	}

	// Duration bar
	durMs := int64(0)
	if action.Duration != nil {
		durMs = int64(*action.Duration)
	}
	bar := renderDurationBar(durMs, maxDur)

	// Duration string
	durStr := formatDurationMs(durMs)
	if durMs == 0 {
		durStr = "-"
	}

	// Main line
	fmt.Printf("%s %d. %s [%s]  %s  %s  %s\n",
		statusStyle.Render(statusIcon),
		idx,
		execLabelStyle.Render(name),
		execTypeStyle.Render(action.Type),
		bar,
		durStr,
		statusStyle.Render(action.Status),
	)

	// Error detail
	if action.Error != nil && *action.Error != "" {
		errMsg := *action.Error
		if len(errMsg) > 70 {
			errMsg = errMsg[:67] + "..."
		}
		fmt.Printf("│   %s\n", execFailureStyle.Render(errMsg))
	}

	// Inputs/Outputs if requested
	if showIO {
		if len(action.Inputs) > 0 {
			fmt.Printf("│   %s %s\n",
				execIOKeyStyle.Render("Inputs:"),
				execIOValStyle.Render(formatJSONPreview(action.Inputs)),
			)
		}
		if len(action.Outputs) > 0 {
			fmt.Printf("│   %s %s\n",
				execIOKeyStyle.Render("Outputs:"),
				execIOValStyle.Render(formatJSONPreview(action.Outputs)),
			)
		}
	}
}

func renderDurationBar(durMs, maxDur int64) string {
	if maxDur <= 0 {
		maxDur = 1
	}
	filled := int(float64(durMs) / float64(maxDur) * float64(barWidth))
	if filled > barWidth {
		filled = barWidth
	}
	if durMs > 0 && filled == 0 {
		filled = 1 // Minimum visibility
	}

	bar := execBarFull.Render(strings.Repeat("█", filled)) +
		execBarEmpty.Render(strings.Repeat("░", barWidth-filled))
	return bar
}

func renderExecutionStatus(status string) string {
	switch status {
	case "SUCCESS":
		return execSuccessStyle.Render(status)
	case "FAILURE":
		return execFailureStyle.Render(status)
	case "RUNNING":
		return execRunningStyle.Render(status)
	case "QUEUED":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("14")).Render(status)
	case "CANCELLED":
		return execMutedStyle.Render(status)
	default:
		return status
	}
}

func formatDurationMs(ms int64) string {
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

func truncateID(id string, maxLen int) string {
	if len(id) <= maxLen {
		return id
	}
	return id[:maxLen]
}

func formatJSONPreview(data json.RawMessage) string {
	if len(data) == 0 || string(data) == "null" {
		return "(empty)"
	}

	// Try to format as compact JSON
	s := string(data)
	if len(s) > maxIOPreview {
		s = s[:maxIOPreview-3] + "..."
	}

	// Remove newlines for compact display
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "  ", " ")

	return s
}
