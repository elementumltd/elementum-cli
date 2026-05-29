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

	"github.com/elementumltd/elementum-cli/analysis"
	"github.com/elementumltd/elementum-cli/auth"
	"github.com/elementumltd/elementum-cli/internal/client"
	"github.com/elementumltd/elementum-cli/ui"
	"github.com/spf13/cobra"
)

var conversationCmd = &cobra.Command{
	Use:   "conversation <agent-name-or-id> <conversation-id>",
	Short: "Analyze agent conversation performance",
	Long: `Fetch an agent conversation and produce a timing/performance analysis.

Shows tool call durations, LLM thinking time, turn-by-turn latency,
and aggregate statistics. Useful for understanding agent performance
bottlenecks after a chat session or UAT test.

The agent can be specified by name (case-insensitive) or UUID.

Output modes:
  (default)     Text summary with tables
  --json        Full structured JSON analysis
  --timeline    CLI visual timeline with horizontal bars
  --html        Interactive HTML waterfall opened in browser

Examples:
  ei conversation "IT Support Agent" <conversation-id>
  ei conversation "IT Support Agent" <conversation-id> --json
  ei conversation <agent-id> <conversation-id> --timeline
  ei conversation <agent-id> <conversation-id> --html`,
	Args: cobra.ExactArgs(2),
	RunE: runConversation,
}

var (
	convTimeline bool
	convHTML     bool
)

func init() {
	conversationCmd.Flags().BoolVar(&convTimeline, "timeline", false, "Show CLI visual timeline with horizontal bars")
	conversationCmd.Flags().BoolVar(&convHTML, "html", false, "Open interactive HTML waterfall in browser")
}

// GetConversationCmd returns the conversation command for registration.
func GetConversationCmd() *cobra.Command {
	return conversationCmd
}

func runConversation(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	conversationID := args[1]

	if !looksLikeUUID(conversationID) {
		return fmt.Errorf("conversation ID must be a valid UUID")
	}

	apiClient, err := auth.GetClientFromCmd(cmd)
	if err != nil {
		return err
	}

	agentID, err := resolveAgentID(ctx, cmd, apiClient, args[0])
	if err != nil {
		return err
	}

	if !isJSONOutput(cmd) && !convTimeline && !convHTML {
		fmt.Println(ui.InfoStyle.Render("Loading conversation..."))
	}

	// Fetch all pages of conversation messages
	parsed, err := fetchConversation(ctx, apiClient, agentID, conversationID)
	if err != nil {
		return fmt.Errorf("failed to fetch conversation: %w", err)
	}

	// Run analysis
	result := analysis.Analyze(parsed)

	// Route to output format
	if isJSONOutput(cmd) {
		return outputJSON(result)
	}
	if convHTML {
		return analysis.RenderHTML(result)
	}
	if convTimeline {
		analysis.RenderTimeline(result)
		return nil
	}

	renderTextSummary(result)
	return nil
}

const pageSize = 250

func fetchConversation(ctx context.Context, apiClient *client.Client, agentID, conversationID string) (*analysis.ParsedConversation, error) {
	var allEvents []analysis.RawEvent
	var parsed *analysis.ParsedConversation
	var cursor *string

	for {
		resp, err := client.GetConversationAnalysis(ctx, apiClient.Genqlient(), agentID, conversationID, pageSize, cursor)
		if err != nil {
			return nil, err
		}

		page, pageInfo, err := analysis.ParseResponse(resp)
		if err != nil {
			return nil, err
		}

		if parsed == nil {
			parsed = page
		}
		allEvents = append(allEvents, page.Events...)

		if !pageInfo.HasNextPage {
			break
		}
		cursor = pageInfo.EndCursor
	}

	if parsed != nil {
		parsed.Events = allEvents
	}
	return parsed, nil
}

func printDetail(label, value string) {
	fmt.Printf("  %s  %s\n", ui.LabelStyle.Render(label), value)
}

// renderTextSummary produces the default styled terminal output.
func renderTextSummary(a *analysis.ConversationAnalysis) {
	fmt.Println()

	title := "(untitled)"
	if a.Title != nil && *a.Title != "" {
		title = *a.Title
	}
	fmt.Println(ui.TitleStyle.Render(fmt.Sprintf("Conversation: %s", title)))
	printDetail("Agent:", a.AgentName)
	printDetail("Messages:", fmt.Sprintf("%d (%d assistant)", a.MessageCount, a.TotalAssistantMsgs))
	printDetail("Duration:", formatDuration(int(a.Timeline.TotalDurationMs)))
	printDetail("Strategy:", string(a.TimingStrategy))
	fmt.Println()

	renderTurnsTable(a)
	renderToolCallsTable(a)
	renderStatsSection(a)

	fmt.Println(ui.MutedStyle.Render("Use --timeline for visual timeline, --html for interactive waterfall, --json for structured data"))
	fmt.Println()
}

func renderTurnsTable(a *analysis.ConversationAnalysis) {
	if len(a.Timeline.Turns) == 0 {
		return
	}
	fmt.Println(ui.SubtitleStyle.Render("Turns"))
	turnTable := ui.NewTable([]string{"#", "USER MESSAGE", "TTFR", "E2E", "TOOLS"})
	for _, turn := range a.Timeline.Turns {
		msg := turn.UserContent
		if len(msg) > 40 {
			msg = msg[:37] + "..."
		}
		turnTable.AddRow(
			fmt.Sprintf("%d", turn.TurnIndex+1),
			msg,
			formatDuration(int(turn.TimeToFirstResponseMs)),
			formatDuration(int(turn.EndToEndMs)),
			fmt.Sprintf("%d", turn.ToolCallCount),
		)
	}
	fmt.Println(turnTable.RenderSimple())
	fmt.Println()
}

func renderToolCallsTable(a *analysis.ConversationAnalysis) {
	if len(a.ToolCalls) == 0 {
		return
	}
	fmt.Println(ui.SubtitleStyle.Render(fmt.Sprintf("Tool Calls (%d)", len(a.ToolCalls))))
	toolTable := ui.NewTable([]string{"STATUS", "TOOL", "DURATION", "STRATEGY"})
	for _, tc := range a.ToolCalls {
		label := tc.ResolvedLabel
		if len(label) > 35 {
			label = label[:32] + "..."
		}
		dur := formatDuration(int(tc.DurationMs))
		if tc.IsOutlier {
			dur += " " + ui.WarningStyle.Render("(outlier)")
		}
		toolTable.AddRow(styledToolStatus(tc.Status), label, dur, string(tc.Strategy))
	}
	fmt.Println(toolTable.RenderSimple())
	fmt.Println()
}

func renderStatsSection(a *analysis.ConversationAnalysis) {
	s := a.Stats
	if s.TotalToolCalls == 0 {
		return
	}

	fmt.Println(ui.SubtitleStyle.Render("Statistics"))
	printDetail("Total:", fmt.Sprintf("%d tool calls in %s", s.TotalToolCalls, formatDuration(int(s.TotalDurationMs))))
	printDetail("Avg:", formatDuration(int(s.AvgDurationMs)))
	printDetail("Median:", formatDuration(int(s.MedianDurationMs)))
	printDetail("P95:", formatDuration(int(s.P95DurationMs)))
	printDetail("Max:", formatDuration(int(s.MaxDurationMs)))
	if s.ErrorCount > 0 {
		printDetail("Errors:", ui.ErrorStyle.Render(fmt.Sprintf("%d", s.ErrorCount)))
	}
	fmt.Println()

	var totalThinking int64
	for _, tp := range a.ThinkingPeriods {
		totalThinking += tp.DurationMs
	}
	total := s.TotalDurationMs + totalThinking
	if total > 0 {
		toolPct := int(float64(s.TotalDurationMs) / float64(total) * 100)
		thinkPct := 100 - toolPct
		fmt.Println(ui.SubtitleStyle.Render("Time Breakdown"))
		printDetail("Tool execution:", fmt.Sprintf("%s (%d%%)", formatDuration(int(s.TotalDurationMs)), toolPct))
		printDetail("LLM thinking:", fmt.Sprintf("%s (%d%%)", formatDuration(int(totalThinking)), thinkPct))
		fmt.Println()
	}
}

func styledToolStatus(status analysis.ToolCallStatus) string {
	switch status {
	case analysis.StatusSuccess:
		return ui.SuccessStyle.Render("OK")
	case analysis.StatusError:
		return ui.ErrorStyle.Render("ERR")
	case analysis.StatusPending:
		return ui.WarningStyle.Render("...")
	default:
		return string(status)
	}
}
