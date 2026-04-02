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
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	toolBarStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("12")) // Cyan
	thinkBarStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))  // Gray
	outlierStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("11")).Bold(true)
	errorBarStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("9")) // Red
	turnHeaderStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("14"))
	labelStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("15"))
	mutedStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	statLabelStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("14")).Bold(true)
	successStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
	errStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
)

const (
	maxBarWidth   = 50
	labelWidth    = 35
	durationWidth = 10
)

// RenderTimeline prints a CLI visual timeline to stdout.
func RenderTimeline(a *ConversationAnalysis) {
	if a == nil || len(a.ToolCalls) == 0 {
		fmt.Println(mutedStyle.Render("No tool calls to display."))
		return
	}

	fmt.Println()

	maxDur := findMaxDuration(a)

	if len(a.Timeline.Turns) > 0 {
		renderTurns(a, maxDur)
	} else {
		for i := range a.ToolCalls {
			renderToolBar(&a.ToolCalls[i], maxDur)
		}
		fmt.Println()
	}

	renderStatsSummary(a)
}

func findMaxDuration(a *ConversationAnalysis) int64 {
	maxDur := int64(0)
	for _, tc := range a.ToolCalls {
		if tc.DurationMs > maxDur {
			maxDur = tc.DurationMs
		}
	}
	for _, tp := range a.ThinkingPeriods {
		if tp.DurationMs > maxDur {
			maxDur = tp.DurationMs
		}
	}
	if maxDur == 0 {
		maxDur = 1
	}
	return maxDur
}

type timelineItem struct {
	isThinking bool
	toolCall   *ToolCallInfo
	thinking   *ThinkingPeriod
	startMs    int64
}

func renderTurns(a *ConversationAnalysis, maxDur int64) {
	for turnIdx, turn := range a.Timeline.Turns {
		turnDur := formatMs(turn.EndToEndMs)
		header := fmt.Sprintf("Turn %d", turn.TurnIndex+1)
		divider := strings.Repeat("─", 40)
		fmt.Printf("%s  %s %s\n",
			turnHeaderStyle.Render(header),
			mutedStyle.Render(divider),
			mutedStyle.Render(turnDur),
		)

		items := collectTurnItems(a, turnIdx)
		for _, item := range items {
			if item.isThinking {
				renderThinkingBar(item.thinking.DurationMs, maxDur)
			} else {
				renderToolBar(item.toolCall, maxDur)
			}
		}
		fmt.Println()
	}
}

func collectTurnItems(a *ConversationAnalysis, turnIdx int) []timelineItem {
	turn := a.Timeline.Turns[turnIdx]
	turnStart := turn.UserMessageAt
	turnEnd := a.TimelineEnd
	if turnIdx+1 < len(a.Timeline.Turns) {
		turnEnd = a.Timeline.Turns[turnIdx+1].UserMessageAt
	}

	var items []timelineItem
	for i := range a.ThinkingPeriods {
		tp := &a.ThinkingPeriods[i]
		if !tp.StartedAt.Before(turnStart) && tp.StartedAt.Before(turnEnd) {
			items = append(items, timelineItem{isThinking: true, thinking: tp, startMs: tp.StartedAt.Sub(a.TimelineStart).Milliseconds()})
		}
	}
	for i := range a.ToolCalls {
		tc := &a.ToolCalls[i]
		if tc.StartedAt != nil && !tc.StartedAt.Before(turnStart) && tc.StartedAt.Before(turnEnd) {
			items = append(items, timelineItem{isThinking: false, toolCall: tc, startMs: tc.StartedAt.Sub(a.TimelineStart).Milliseconds()})
		}
	}

	for i := 1; i < len(items); i++ {
		for j := i; j > 0 && items[j].startMs < items[j-1].startMs; j-- {
			items[j], items[j-1] = items[j-1], items[j]
		}
	}
	return items
}

func renderToolBar(tc *ToolCallInfo, maxDur int64) {
	barLen := int(float64(tc.DurationMs) / float64(maxDur) * float64(maxBarWidth))
	if barLen < 1 && tc.DurationMs > 0 {
		barLen = 1
	}

	bar := strings.Repeat("█", barLen)

	label := tc.ResolvedLabel
	if len(label) > labelWidth-4 {
		label = label[:labelWidth-7] + "..."
	}

	dur := formatMs(tc.DurationMs)

	style := toolBarStyle
	suffix := ""
	if tc.Status == StatusError {
		style = errorBarStyle
		suffix = " " + errStyle.Render("ERR")
	} else if tc.IsOutlier {
		suffix = " " + outlierStyle.Render("▲ outlier")
	}

	// Parallel indicator
	prefix := "  █ "
	if tc.ParallelSize > 1 {
		prefix = "  ║ "
	}

	fmt.Printf("%s%-*s %s %s%s\n",
		mutedStyle.Render(prefix),
		labelWidth,
		labelStyle.Render(label),
		style.Render(bar),
		mutedStyle.Render(padLeft(dur, durationWidth)),
		suffix,
	)
}

func renderThinkingBar(durationMs int64, maxDur int64) {
	barLen := int(float64(durationMs) / float64(maxDur) * float64(maxBarWidth))
	if barLen < 1 && durationMs > 0 {
		barLen = 1
	}

	bar := strings.Repeat("▒", barLen)
	dur := formatMs(durationMs)

	fmt.Printf("%s%-*s %s %s\n",
		mutedStyle.Render("  ▓ "),
		labelWidth,
		mutedStyle.Render("LLM thinking"),
		thinkBarStyle.Render(bar),
		mutedStyle.Render(padLeft(dur, durationWidth)),
	)
}

func renderStatsSummary(a *ConversationAnalysis) {
	s := a.Stats

	var totalThinking int64
	for _, tp := range a.ThinkingPeriods {
		totalThinking += tp.DurationMs
	}
	total := s.TotalDurationMs + totalThinking
	toolPct := 0
	thinkPct := 0
	if total > 0 {
		toolPct = int(float64(s.TotalDurationMs) / float64(total) * 100)
		thinkPct = 100 - toolPct
	}

	parts := []string{
		statLabelStyle.Render(fmt.Sprintf("%d tool calls", s.TotalToolCalls)),
	}
	if s.AvgDurationMs > 0 {
		parts = append(parts, fmt.Sprintf("avg %s", formatMs(s.AvgDurationMs)))
	}
	if s.P95DurationMs > 0 {
		parts = append(parts, fmt.Sprintf("p95 %s", formatMs(s.P95DurationMs)))
	}

	line := strings.Join(parts, ", ")

	if total > 0 {
		line += mutedStyle.Render(" | ") +
			fmt.Sprintf("LLM: %s (%d%%)", formatMs(totalThinking), thinkPct) +
			mutedStyle.Render(" | ") +
			fmt.Sprintf("Tools: %s (%d%%)", formatMs(s.TotalDurationMs), toolPct)
	}

	if s.ErrorCount > 0 {
		line += mutedStyle.Render(" | ") + errStyle.Render(fmt.Sprintf("%d errors", s.ErrorCount))
	}

	fmt.Println(line)
	fmt.Println()
}

func formatMs(ms int64) string {
	if ms <= 0 {
		return "-"
	}
	if ms < 1000 {
		return fmt.Sprintf("%dms", ms)
	}
	return fmt.Sprintf("%.1fs", float64(ms)/1000.0)
}

func padLeft(s string, width int) string {
	if len(s) >= width {
		return s
	}
	return strings.Repeat(" ", width-len(s)) + s
}
