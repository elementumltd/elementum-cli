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
	"time"
)

// ConversationAnalysis is the top-level result of analyzing a conversation.
type ConversationAnalysis struct {
	ConversationID     string            `json:"conversationId"`
	Title              *string           `json:"title"`
	AgentID            string            `json:"agentId"`
	AgentName          string            `json:"agentName"`
	MessageCount       int               `json:"messageCount"`
	TimingStrategy     TimingStrategy    `json:"timingStrategy"`
	TimelineStart      time.Time         `json:"timelineStart"`
	TimelineEnd        time.Time         `json:"timelineEnd"`
	TotalAssistantMsgs int               `json:"totalAssistantMessages"`
	ToolCalls          []ToolCallInfo    `json:"toolCalls"`
	Stats              AggregateStats    `json:"stats"`
	ThinkingPeriods    []ThinkingPeriod  `json:"thinkingPeriods"`
	Timeline           TimelineInfo      `json:"timeline"`
	MessageEvents      []MessageEvent    `json:"messageEvents"`
	SkillNames         map[string]string `json:"skillNames"`
}

type TimingStrategy string

const (
	TimingExplicit  TimingStrategy = "explicit"
	TimingEstimated TimingStrategy = "estimated"
)

// ToolCallInfo represents a single tool invocation with timing data.
type ToolCallInfo struct {
	ToolCallID    string         `json:"toolCallId"`
	ToolName      string         `json:"toolName"`
	DisplayName   string         `json:"displayName"`
	ResolvedLabel string         `json:"resolvedLabel"`
	Status        ToolCallStatus `json:"status"`
	StartedAt     *time.Time     `json:"startedAt"`
	CompletedAt   *time.Time     `json:"completedAt"`
	DurationMs    int64          `json:"durationMs"`
	ErrorMessage  string         `json:"errorMessage,omitempty"`
	Arguments     []ToolCallArg  `json:"arguments,omitempty"`
	IsOutlier     bool           `json:"isOutlier"`
	ParallelGroup int            `json:"parallelGroup"`
	ParallelSize  int            `json:"parallelSize"`
	Strategy      TimingStrategy `json:"strategy"`
}

type ToolCallStatus string

const (
	StatusSuccess ToolCallStatus = "success"
	StatusError   ToolCallStatus = "error"
	StatusPending ToolCallStatus = "pending"
)

// ToolCallArg is a name/value pair from the tool call arguments.
type ToolCallArg struct {
	Name  string          `json:"name"`
	Value json.RawMessage `json:"value"`
}

// AggregateStats contains summary statistics across all tool calls.
type AggregateStats struct {
	TotalToolCalls   int                  `json:"totalToolCalls"`
	AvgDurationMs    int64                `json:"avgDurationMs"`
	MedianDurationMs int64                `json:"medianDurationMs"`
	P95DurationMs    int64                `json:"p95DurationMs"`
	MaxDurationMs    int64                `json:"maxDurationMs"`
	TotalDurationMs  int64                `json:"totalDurationMs"`
	ErrorCount       int                  `json:"errorCount"`
	ByToolName       map[string]ToolStats `json:"byToolName"`
}

// ToolStats contains per-tool-type aggregate statistics.
type ToolStats struct {
	Count int   `json:"count"`
	AvgMs int64 `json:"avgMs"`
	MaxMs int64 `json:"maxMs"`
	MinMs int64 `json:"minMs"`
}

// ThinkingPeriod represents a gap where the LLM was processing/reasoning.
type ThinkingPeriod struct {
	StartedAt        time.Time         `json:"startedAt"`
	EndedAt          time.Time         `json:"endedAt"`
	DurationMs       int64             `json:"durationMs"`
	AfterToolCallID  string            `json:"afterToolCallId,omitempty"`
	BeforeToolCallID string            `json:"beforeToolCallId,omitempty"`
	Segments         []ThinkingSegment `json:"segments,omitempty"`
}

// ThinkingSegment breaks a thinking period into sub-segments when a user
// message falls within the gap (cross-turn splitting).
type ThinkingSegment struct {
	Type       SegmentType `json:"type"`
	StartedAt  time.Time   `json:"startedAt"`
	EndedAt    time.Time   `json:"endedAt"`
	DurationMs int64       `json:"durationMs"`
}

type SegmentType string

const (
	SegmentLLM      SegmentType = "llm"
	SegmentUserIdle SegmentType = "user_idle"
)

// TimelineInfo contains turn-level timing data.
type TimelineInfo struct {
	TotalDurationMs       int64          `json:"totalDurationMs"`
	TimeToFirstResponseMs int64          `json:"timeToFirstResponseMs"`
	FirstTurnEndToEndMs   int64          `json:"firstTurnEndToEndMs"`
	Turns                 []TurnAnalysis `json:"turns"`
}

// TurnAnalysis groups the conversation into user-to-agent turns.
type TurnAnalysis struct {
	TurnIndex             int        `json:"turnIndex"`
	UserMessageAt         time.Time  `json:"userMessageAt"`
	UserContent           string     `json:"userContent"`
	FirstAssistantAt      *time.Time `json:"firstAssistantAt"`
	AgentDoneAt           *time.Time `json:"agentDoneAt"`
	TimeToFirstResponseMs int64      `json:"timeToFirstResponseMs"`
	EndToEndMs            int64      `json:"endToEndMs"`
	ToolCallCount         int        `json:"toolCallCount"`
}

// MessageEvent is a simplified chronological event for the message timeline.
type MessageEvent struct {
	Type           string    `json:"type"`
	Timestamp      time.Time `json:"timestamp"`
	ContentPreview string    `json:"contentPreview"`
}

// RawEvent is the normalized intermediate representation between
// the genqlient response types and the analysis engine.
type RawEvent struct {
	Type          RawEventType
	ID            string
	CreatedAt     time.Time
	Content       string
	ToolCallID    string
	ToolCalls     []RawToolCall
	ToolResponses []RawToolResponse
	ErrorMessage  string
	StartMessage  string
}

type RawEventType int

const (
	EventUser RawEventType = iota
	EventAssistant
	EventSystem
	EventToolStart
	EventToolComplete
	EventToolError
	EventToolMessage
)

// RawToolCall is a normalized tool call from an assistant message.
type RawToolCall struct {
	ToolCallID  string
	Name        string
	DisplayName string
	Arguments   []ToolCallArg
}

// RawToolResponse is a normalized tool response from a tool message event.
type RawToolResponse struct {
	ToolCallID  string
	Name        string
	DisplayName string
}
