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
	"sort"
	"time"
)

const (
	parallelWindowMs    = 1000
	outlierMultiplier   = 2.0
	thinkingThresholdMs = 500
	contentPreviewLen   = 200
)

// Analyze takes a parsed conversation and produces the full performance analysis.
func Analyze(parsed *ParsedConversation) *ConversationAnalysis {
	events := parsed.Events
	if len(events) == 0 {
		return &ConversationAnalysis{
			ConversationID: parsed.ConversationID,
			Title:          parsed.Title,
			AgentID:        parsed.AgentID,
			AgentName:      parsed.AgentName,
			MessageCount:   parsed.TotalMessages,
			SkillNames:     map[string]string{},
		}
	}

	// Sort events chronologically
	sort.Slice(events, func(i, j int) bool {
		return events[i].CreatedAt.Before(events[j].CreatedAt)
	})

	// Build indexes
	toolCallIndex := buildToolCallIndex(events)
	startEvents := buildToolEventIndex(events, EventToolStart)
	completeEvents := buildToolEventIndex(events, EventToolComplete)
	errorEvents := buildToolEventIndex(events, EventToolError)
	responseIndex := buildToolResponseIndex(events)

	// Determine timing strategy
	strategy := TimingEstimated
	if len(startEvents) > 0 {
		strategy = TimingExplicit
	}

	// Extract tool call timings
	toolCalls := extractToolCalls(events, toolCallIndex, startEvents, completeEvents, errorEvents, responseIndex, strategy)

	// Post-processing
	detectParallelGroups(toolCalls)
	detectOutliers(toolCalls)

	// Build skill name map from tool calls
	skillNames := buildSkillNameMap(toolCalls)

	// Compute stats
	stats := computeStats(toolCalls)

	// Build message events timeline
	messageEvents := buildMessageEvents(events)

	// Compute turn analysis
	turns := computeTurns(events, toolCalls)

	// Compute thinking periods
	thinkingPeriods := computeThinkingPeriods(events, toolCalls)

	// Build timeline info
	timelineInfo := buildTimeline(events, turns)

	return &ConversationAnalysis{
		ConversationID:     parsed.ConversationID,
		Title:              parsed.Title,
		AgentID:            parsed.AgentID,
		AgentName:          parsed.AgentName,
		MessageCount:       parsed.TotalMessages,
		TimingStrategy:     strategy,
		TimelineStart:      events[0].CreatedAt,
		TimelineEnd:        events[len(events)-1].CreatedAt,
		TotalAssistantMsgs: countByType(events, EventAssistant),
		ToolCalls:          toolCalls,
		Stats:              stats,
		ThinkingPeriods:    thinkingPeriods,
		Timeline:           timelineInfo,
		MessageEvents:      messageEvents,
		SkillNames:         skillNames,
	}
}

// --- Tool call extraction ---

type toolCallMeta struct {
	toolCallID  string
	name        string
	displayName string
	args        []ToolCallArg
	invokedAt   time.Time
}

func buildToolCallIndex(events []RawEvent) map[string]toolCallMeta {
	index := map[string]toolCallMeta{}
	for _, ev := range events {
		if ev.Type != EventAssistant {
			continue
		}
		for _, tc := range ev.ToolCalls {
			index[tc.ToolCallID] = toolCallMeta{
				toolCallID:  tc.ToolCallID,
				name:        tc.Name,
				displayName: tc.DisplayName,
				args:        tc.Arguments,
				invokedAt:   ev.CreatedAt,
			}
		}
	}
	return index
}

func buildToolEventIndex(events []RawEvent, evType RawEventType) map[string]RawEvent {
	index := map[string]RawEvent{}
	for _, ev := range events {
		if ev.Type == evType && ev.ToolCallID != "" {
			index[ev.ToolCallID] = ev
		}
	}
	return index
}

func buildToolResponseIndex(events []RawEvent) map[string]RawEvent {
	index := map[string]RawEvent{}
	for _, ev := range events {
		if ev.Type != EventToolMessage {
			continue
		}
		for _, tr := range ev.ToolResponses {
			index[tr.ToolCallID] = ev
		}
	}
	return index
}

func extractToolCalls(
	events []RawEvent,
	callIndex map[string]toolCallMeta,
	starts, completes, errors map[string]RawEvent,
	responses map[string]RawEvent,
	defaultStrategy TimingStrategy,
) []ToolCallInfo {

	seen := map[string]bool{}
	var result []ToolCallInfo

	for _, meta := range callIndex {
		if seen[meta.toolCallID] {
			continue
		}
		seen[meta.toolCallID] = true

		tc := ToolCallInfo{
			ToolCallID:    meta.toolCallID,
			ToolName:      meta.name,
			DisplayName:   meta.displayName,
			Arguments:     meta.args,
			ResolvedLabel: ResolveToolLabel(meta.name, meta.displayName, meta.args),
		}

		startEv, hasStart := starts[meta.toolCallID]
		completeEv, hasComplete := completes[meta.toolCallID]
		errorEv, hasError := errors[meta.toolCallID]

		if hasStart {
			// Strategy 1: Explicit events
			tc.Strategy = TimingExplicit
			t := startEv.CreatedAt
			tc.StartedAt = &t

			if hasError {
				tc.Status = StatusError
				t2 := errorEv.CreatedAt
				tc.CompletedAt = &t2
				tc.ErrorMessage = errorEv.ErrorMessage
				tc.DurationMs = t2.Sub(t).Milliseconds()
			} else if hasComplete {
				tc.Status = StatusSuccess
				t2 := completeEv.CreatedAt
				tc.CompletedAt = &t2
				tc.DurationMs = t2.Sub(t).Milliseconds()
			} else {
				tc.Status = StatusPending
			}
		} else {
			// Strategy 2: Estimated from assistant message timestamp
			tc.Strategy = TimingEstimated
			t := meta.invokedAt
			tc.StartedAt = &t

			if hasError {
				tc.Status = StatusError
				t2 := errorEv.CreatedAt
				tc.CompletedAt = &t2
				tc.ErrorMessage = errorEv.ErrorMessage
				tc.DurationMs = t2.Sub(t).Milliseconds()
			} else if hasComplete {
				tc.Status = StatusSuccess
				t2 := completeEv.CreatedAt
				tc.CompletedAt = &t2
				tc.DurationMs = t2.Sub(t).Milliseconds()
			} else if respEv, hasResp := responses[meta.toolCallID]; hasResp {
				tc.Status = StatusSuccess
				t2 := respEv.CreatedAt
				if !t2.IsZero() {
					tc.CompletedAt = &t2
					tc.DurationMs = t2.Sub(t).Milliseconds()
				}
			} else {
				tc.Status = StatusPending
			}
		}

		result = append(result, tc)
	}

	sort.Slice(result, func(i, j int) bool {
		if result[i].StartedAt == nil {
			return true
		}
		if result[j].StartedAt == nil {
			return false
		}
		return result[i].StartedAt.Before(*result[j].StartedAt)
	})

	return result
}

// --- Parallel detection ---

func detectParallelGroups(calls []ToolCallInfo) {
	if len(calls) == 0 {
		return
	}

	group := 0
	groupStart := calls[0].StartedAt
	members := []int{0}

	for i := 1; i < len(calls); i++ {
		if calls[i].StartedAt == nil || groupStart == nil {
			group++
			groupStart = calls[i].StartedAt
			members = []int{i}
			continue
		}

		if calls[i].StartedAt.Sub(*groupStart).Milliseconds() <= parallelWindowMs {
			members = append(members, i)
		} else {
			assignGroup(calls, members, group)
			group++
			groupStart = calls[i].StartedAt
			members = []int{i}
		}
	}
	assignGroup(calls, members, group)
}

func assignGroup(calls []ToolCallInfo, members []int, group int) {
	for _, idx := range members {
		calls[idx].ParallelGroup = group
		calls[idx].ParallelSize = len(members)
	}
}

// --- Outlier detection ---

func detectOutliers(calls []ToolCallInfo) {
	byName := map[string][]int64{}
	for _, tc := range calls {
		if tc.DurationMs > 0 {
			byName[tc.ResolvedLabel] = append(byName[tc.ResolvedLabel], tc.DurationMs)
		}
	}

	avgByName := map[string]float64{}
	for name, durations := range byName {
		var sum int64
		for _, d := range durations {
			sum += d
		}
		avgByName[name] = float64(sum) / float64(len(durations))
	}

	for i := range calls {
		avg, ok := avgByName[calls[i].ResolvedLabel]
		if ok && float64(calls[i].DurationMs) > avg*outlierMultiplier && len(byName[calls[i].ResolvedLabel]) > 1 {
			calls[i].IsOutlier = true
		}
	}
}

// --- Aggregate stats ---

func computeStats(calls []ToolCallInfo) AggregateStats {
	stats := AggregateStats{
		TotalToolCalls: len(calls),
		ByToolName:     map[string]ToolStats{},
	}

	if len(calls) == 0 {
		return stats
	}

	var durations []int64
	for _, tc := range calls {
		if tc.DurationMs > 0 {
			durations = append(durations, tc.DurationMs)
			stats.TotalDurationMs += tc.DurationMs
		}
		if tc.Status == StatusError {
			stats.ErrorCount++
		}

		ts := stats.ByToolName[tc.ResolvedLabel]
		ts.Count++
		if tc.DurationMs > 0 {
			if ts.MinMs == 0 || tc.DurationMs < ts.MinMs {
				ts.MinMs = tc.DurationMs
			}
			if tc.DurationMs > ts.MaxMs {
				ts.MaxMs = tc.DurationMs
			}
		}
		stats.ByToolName[tc.ResolvedLabel] = ts
	}

	if len(durations) > 0 {
		sort.Slice(durations, func(i, j int) bool { return durations[i] < durations[j] })
		stats.AvgDurationMs = stats.TotalDurationMs / int64(len(durations))
		stats.MedianDurationMs = percentile(durations, 50)
		stats.P95DurationMs = percentile(durations, 95)
		stats.MaxDurationMs = durations[len(durations)-1]
	}

	// Compute per-tool averages
	for name, ts := range stats.ByToolName {
		var sum int64
		var count int
		for _, tc := range calls {
			if tc.ResolvedLabel == name && tc.DurationMs > 0 {
				sum += tc.DurationMs
				count++
			}
		}
		if count > 0 {
			ts.AvgMs = sum / int64(count)
		}
		stats.ByToolName[name] = ts
	}

	return stats
}

func percentile(sorted []int64, p int) int64 {
	if len(sorted) == 0 {
		return 0
	}
	idx := (p * len(sorted)) / 100
	if idx >= len(sorted) {
		idx = len(sorted) - 1
	}
	return sorted[idx]
}

// --- Message events ---

func buildMessageEvents(events []RawEvent) []MessageEvent {
	var result []MessageEvent
	for _, ev := range events {
		switch ev.Type {
		case EventUser:
			result = append(result, MessageEvent{
				Type:           "user",
				Timestamp:      ev.CreatedAt,
				ContentPreview: trimContent(ev.Content, contentPreviewLen),
			})
		case EventAssistant:
			result = append(result, MessageEvent{
				Type:           "assistant",
				Timestamp:      ev.CreatedAt,
				ContentPreview: trimContent(ev.Content, contentPreviewLen),
			})
		}
	}
	return result
}

func trimContent(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}

// --- Turn analysis ---

func computeTurns(events []RawEvent, toolCalls []ToolCallInfo) []TurnAnalysis {
	var turns []TurnAnalysis

	for i, ev := range events {
		if ev.Type != EventUser {
			continue
		}

		turn := TurnAnalysis{
			TurnIndex:     len(turns),
			UserMessageAt: ev.CreatedAt,
			UserContent:   trimContent(ev.Content, contentPreviewLen),
		}

		// Find next user message to determine turn boundary
		var turnEnd time.Time
		for j := i + 1; j < len(events); j++ {
			if events[j].Type == EventUser {
				break
			}
			turnEnd = events[j].CreatedAt

			if events[j].Type == EventAssistant && turn.FirstAssistantAt == nil {
				t := events[j].CreatedAt
				turn.FirstAssistantAt = &t
				turn.TimeToFirstResponseMs = t.Sub(ev.CreatedAt).Milliseconds()
			}
		}

		if !turnEnd.IsZero() {
			t := turnEnd
			turn.AgentDoneAt = &t
			turn.EndToEndMs = turnEnd.Sub(ev.CreatedAt).Milliseconds()
		}

		// Count tool calls in this turn
		for _, tc := range toolCalls {
			if tc.StartedAt != nil && !tc.StartedAt.Before(ev.CreatedAt) {
				if turnEnd.IsZero() || !tc.StartedAt.After(turnEnd) {
					turn.ToolCallCount++
				}
			}
		}

		turns = append(turns, turn)
	}

	return turns
}

// --- Thinking periods ---

func computeThinkingPeriods(events []RawEvent, toolCalls []ToolCallInfo) []ThinkingPeriod {
	if len(toolCalls) == 0 {
		return nil
	}

	// Build a sorted list of tool call time ranges
	type timeRange struct {
		start time.Time
		end   time.Time
		id    string
	}

	var ranges []timeRange
	for _, tc := range toolCalls {
		if tc.StartedAt == nil {
			continue
		}
		end := *tc.StartedAt
		if tc.CompletedAt != nil {
			end = *tc.CompletedAt
		}
		ranges = append(ranges, timeRange{start: *tc.StartedAt, end: end, id: tc.ToolCallID})
	}

	if len(ranges) == 0 {
		return nil
	}

	sort.Slice(ranges, func(i, j int) bool {
		return ranges[i].start.Before(ranges[j].start)
	})

	// Compute high-water-mark end time to find gaps
	var periods []ThinkingPeriod
	highWater := ranges[0].start

	// Initial thinking: conversation start to first tool
	convStart := events[0].CreatedAt
	if ranges[0].start.Sub(convStart).Milliseconds() >= thinkingThresholdMs {
		periods = append(periods, ThinkingPeriod{
			StartedAt:        convStart,
			EndedAt:          ranges[0].start,
			DurationMs:       ranges[0].start.Sub(convStart).Milliseconds(),
			BeforeToolCallID: ranges[0].id,
		})
	}

	// Inter-tool gaps (only when no tools are running)
	for i := range ranges {
		if ranges[i].end.After(highWater) {
			highWater = ranges[i].end
		}

		if i+1 < len(ranges) {
			gap := ranges[i+1].start.Sub(highWater)
			if gap.Milliseconds() >= thinkingThresholdMs {
				p := ThinkingPeriod{
					StartedAt:        highWater,
					EndedAt:          ranges[i+1].start,
					DurationMs:       gap.Milliseconds(),
					AfterToolCallID:  ranges[i].id,
					BeforeToolCallID: ranges[i+1].id,
				}
				p.Segments = splitThinkingPeriod(p, events)
				periods = append(periods, p)
			}
		}
	}

	// Trailing thinking: last tool to conversation end
	convEnd := events[len(events)-1].CreatedAt
	if convEnd.Sub(highWater).Milliseconds() >= thinkingThresholdMs {
		periods = append(periods, ThinkingPeriod{
			StartedAt:       highWater,
			EndedAt:         convEnd,
			DurationMs:      convEnd.Sub(highWater).Milliseconds(),
			AfterToolCallID: ranges[len(ranges)-1].id,
		})
	}

	return periods
}

// splitThinkingPeriod breaks a thinking gap into segments if a user message
// falls within it (cross-turn splitting).
func splitThinkingPeriod(p ThinkingPeriod, events []RawEvent) []ThinkingSegment {
	// Find user/assistant messages within the thinking period
	type boundaryMsg struct {
		time   time.Time
		isUser bool
	}

	var boundaries []boundaryMsg
	for _, ev := range events {
		if ev.CreatedAt.After(p.StartedAt) && ev.CreatedAt.Before(p.EndedAt) {
			switch ev.Type {
			case EventUser:
				boundaries = append(boundaries, boundaryMsg{time: ev.CreatedAt, isUser: true})
			case EventAssistant:
				boundaries = append(boundaries, boundaryMsg{time: ev.CreatedAt, isUser: false})
			}
		}
	}

	if len(boundaries) == 0 {
		return nil
	}

	// Build segments
	var segments []ThinkingSegment
	cursor := p.StartedAt

	for _, b := range boundaries {
		if b.time.Sub(cursor).Milliseconds() > 0 {
			segType := SegmentLLM
			if len(segments) > 0 && !b.isUser {
				segType = SegmentUserIdle
			}
			segments = append(segments, ThinkingSegment{
				Type:       segType,
				StartedAt:  cursor,
				EndedAt:    b.time,
				DurationMs: b.time.Sub(cursor).Milliseconds(),
			})
		}
		cursor = b.time
	}

	// Final segment to period end
	if cursor.Before(p.EndedAt) {
		segments = append(segments, ThinkingSegment{
			Type:       SegmentLLM,
			StartedAt:  cursor,
			EndedAt:    p.EndedAt,
			DurationMs: p.EndedAt.Sub(cursor).Milliseconds(),
		})
	}

	return segments
}

// --- Timeline ---

func buildTimeline(events []RawEvent, turns []TurnAnalysis) TimelineInfo {
	info := TimelineInfo{
		Turns: turns,
	}

	if len(events) > 1 {
		info.TotalDurationMs = events[len(events)-1].CreatedAt.Sub(events[0].CreatedAt).Milliseconds()
	}

	if len(turns) > 0 {
		info.TimeToFirstResponseMs = turns[0].TimeToFirstResponseMs
		info.FirstTurnEndToEndMs = turns[0].EndToEndMs
	}

	return info
}

// --- Skill name map ---

func buildSkillNameMap(toolCalls []ToolCallInfo) map[string]string {
	names := map[string]string{}
	for _, tc := range toolCalls {
		if tc.ToolName == "runSkillTool" {
			for _, arg := range tc.Arguments {
				if arg.Name == "skillId" {
					var id string
					if err := unjsonString(arg.Value, &id); err == nil && id != "" {
						names[id] = tc.ResolvedLabel
					}
				}
			}
		}
	}
	return names
}

func unjsonString(raw []byte, s *string) error {
	return json_Unmarshal(raw, s)
}

// Avoid importing encoding/json just for Unmarshal (it's already in types.go).
var json_Unmarshal = func(data []byte, v any) error {
	// Simple inline JSON string extraction
	if len(data) >= 2 && data[0] == '"' && data[len(data)-1] == '"' {
		if s, ok := v.(*string); ok {
			*s = string(data[1 : len(data)-1])
			return nil
		}
	}
	return &parseError{msg: "not a JSON string"}
}

// --- Helpers ---

func countByType(events []RawEvent, t RawEventType) int {
	n := 0
	for _, ev := range events {
		if ev.Type == t {
			n++
		}
	}
	return n
}
