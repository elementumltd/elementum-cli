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

	"github.com/elementumltd/elementum-cli/internal/client"
)

// Type aliases to keep this file readable despite genqlient's naming.
type (
	gqlEdge         = client.GetConversationAnalysisOrganizationAgentAgentV2ConversationV2AgentConversationMessagesAgentConversationEventConnectionEdgesAgentConversationEventEdge
	gqlEventNode    = client.GetConversationAnalysisOrganizationAgentAgentV2ConversationV2AgentConversationMessagesAgentConversationEventConnectionEdgesAgentConversationEventEdgeNodeAgentConversationEvent
	gqlUserMsg      = client.GetConversationAnalysisOrganizationAgentAgentV2ConversationV2AgentConversationMessagesAgentConversationEventConnectionEdgesAgentConversationEventEdgeNodeAgentConversationUserMessageEvent
	gqlAssistantMsg = client.GetConversationAnalysisOrganizationAgentAgentV2ConversationV2AgentConversationMessagesAgentConversationEventConnectionEdgesAgentConversationEventEdgeNodeAgentConversationAssistantMessageEvent
	gqlSystemMsg    = client.GetConversationAnalysisOrganizationAgentAgentV2ConversationV2AgentConversationMessagesAgentConversationEventConnectionEdgesAgentConversationEventEdgeNodeAgentConversationSystemMessageEvent
	gqlToolStart    = client.GetConversationAnalysisOrganizationAgentAgentV2ConversationV2AgentConversationMessagesAgentConversationEventConnectionEdgesAgentConversationEventEdgeNodeAgentConversationToolStartMessageEvent
	gqlToolComplete = client.GetConversationAnalysisOrganizationAgentAgentV2ConversationV2AgentConversationMessagesAgentConversationEventConnectionEdgesAgentConversationEventEdgeNodeAgentConversationToolCompleteMessageEvent
	gqlToolError    = client.GetConversationAnalysisOrganizationAgentAgentV2ConversationV2AgentConversationMessagesAgentConversationEventConnectionEdgesAgentConversationEventEdgeNodeAgentConversationToolErrorMessageEvent
	gqlToolMessage  = client.GetConversationAnalysisOrganizationAgentAgentV2ConversationV2AgentConversationMessagesAgentConversationEventConnectionEdgesAgentConversationEventEdgeNodeAgentConversationToolMessageEvent
	gqlToolCall     = client.GetConversationAnalysisOrganizationAgentAgentV2ConversationV2AgentConversationMessagesAgentConversationEventConnectionEdgesAgentConversationEventEdgeNodeAgentConversationAssistantMessageEventToolCallsAgentToolCall
	gqlValidArgs    = client.GetConversationAnalysisOrganizationAgentAgentV2ConversationV2AgentConversationMessagesAgentConversationEventConnectionEdgesAgentConversationEventEdgeNodeAgentConversationAssistantMessageEventToolCallsAgentToolCallArgumentsValidToolCallArguments
	gqlToolResponse = client.GetConversationAnalysisOrganizationAgentAgentV2ConversationV2AgentConversationMessagesAgentConversationEventConnectionEdgesAgentConversationEventEdgeNodeAgentConversationToolMessageEventToolResponsesAgentToolResponse
	gqlConversation = client.GetConversationAnalysisOrganizationAgentAgentV2ConversationV2AgentConversation
	gqlPageInfo     = client.GetConversationAnalysisOrganizationAgentAgentV2ConversationV2AgentConversationMessagesAgentConversationEventConnectionPageInfo
)

// ParsedConversation holds the extracted metadata and events from the GraphQL response.
type ParsedConversation struct {
	ConversationID string
	Title          *string
	AgentID        string
	AgentName      string
	Events         []RawEvent
	TotalMessages  int
}

// ParseResponse extracts a flat list of normalized events from a single page of GraphQL results.
func ParseResponse(resp *client.GetConversationAnalysisResponse) (*ParsedConversation, *gqlPageInfo, error) {
	agent := resp.Organization.Agent
	if agent == nil {
		return nil, nil, errorf("agent not found")
	}

	conv := (*agent).GetConversationV2()
	if conv == nil {
		return nil, nil, errorf("conversation not found")
	}

	parsed := &ParsedConversation{
		ConversationID: conv.Id,
		Title:          conv.Title,
		AgentID:        (*agent).GetId(),
		AgentName:      (*agent).GetName(),
		TotalMessages:  conv.Messages.Total,
	}

	edges := conv.Messages.Edges
	events := make([]RawEvent, 0, len(edges))

	for _, edge := range edges {
		ev := parseEdge(edge)
		if ev != nil {
			events = append(events, *ev)
		}
	}

	parsed.Events = events

	pageInfo := conv.Messages.PageInfo
	return parsed, &pageInfo, nil
}

func parseEdge(edge gqlEdge) *RawEvent {
	node := edge.Node
	switch msg := node.(type) {
	case *gqlUserMsg:
		return &RawEvent{
			Type:      EventUser,
			ID:        msg.Id,
			CreatedAt: mustParseTime(msg.CreatedAt),
			Content:   msg.Content,
		}
	case *gqlAssistantMsg:
		return &RawEvent{
			Type:      EventAssistant,
			ID:        msg.Id,
			CreatedAt: mustParseTime(msg.CreatedAt),
			Content:   msg.Content,
			ToolCalls: parseToolCalls(msg.ToolCalls),
		}
	case *gqlSystemMsg:
		return &RawEvent{
			Type:      EventSystem,
			ID:        msg.Id,
			CreatedAt: mustParseTime(msg.CreatedAt),
			Content:   msg.Content,
		}
	case *gqlToolStart:
		return &RawEvent{
			Type:         EventToolStart,
			ID:           msg.Id,
			CreatedAt:    mustParseTime(msg.CreatedAt),
			ToolCallID:   msg.ToolCallId,
			StartMessage: msg.Message,
		}
	case *gqlToolComplete:
		return &RawEvent{
			Type:       EventToolComplete,
			ID:         msg.Id,
			CreatedAt:  mustParseTime(msg.CreatedAt),
			ToolCallID: msg.ToolCallId,
		}
	case *gqlToolError:
		return &RawEvent{
			Type:         EventToolError,
			ID:           msg.Id,
			CreatedAt:    mustParseTime(msg.CreatedAt),
			ToolCallID:   msg.ToolCallId,
			ErrorMessage: msg.ErrorMessage,
		}
	case *gqlToolMessage:
		return &RawEvent{
			Type:          EventToolMessage,
			ID:            msg.Id,
			CreatedAt:     mustParseTime(msg.CreatedAt),
			ToolResponses: parseToolResponses(msg.ToolResponses),
		}
	default:
		return nil
	}
}

func parseToolCalls(calls []gqlToolCall) []RawToolCall {
	result := make([]RawToolCall, 0, len(calls))
	for _, tc := range calls {
		raw := RawToolCall{
			ToolCallID:  tc.ToolCallId,
			DisplayName: tc.DisplayName,
		}
		if tc.Name != nil {
			raw.Name = *tc.Name
		}

		if tc.Arguments != nil {
			if valid, ok := (*tc.Arguments).(*gqlValidArgs); ok {
				for _, jv := range valid.JsonValues {
					raw.Arguments = append(raw.Arguments, ToolCallArg{
						Name:  jv.Name,
						Value: jv.Value,
					})
				}
			}
		}

		result = append(result, raw)
	}
	return result
}

func parseToolResponses(responses []gqlToolResponse) []RawToolResponse {
	result := make([]RawToolResponse, 0, len(responses))
	for _, tr := range responses {
		raw := RawToolResponse{
			ToolCallID:  tr.ToolCallId,
			DisplayName: tr.DisplayName,
		}
		if tr.Name != nil {
			raw.Name = *tr.Name
		}
		if tr.ResponseData != nil {
			raw.ResponseData = *tr.ResponseData
		}
		result = append(result, raw)
	}
	return result
}

func mustParseTime(s string) time.Time {
	t, err := time.Parse(time.RFC3339Nano, s)
	if err != nil {
		t, err = time.Parse(time.RFC3339, s)
		if err != nil {
			return time.Time{}
		}
	}
	return t
}

type parseError struct{ msg string }

func (e *parseError) Error() string { return e.msg }

func errorf(format string, args ...any) error {
	return &parseError{msg: fmtSprintf(format, args...)}
}

// fmtSprintf avoids importing fmt for a single use.
func fmtSprintf(format string, args ...any) string {
	if len(args) == 0 {
		return format
	}
	// Fall back to simple replacement for this internal usage
	result := format
	for _, a := range args {
		if s, ok := a.(string); ok {
			result = replaceFirst(result, "%s", s)
		}
	}
	return result
}

func replaceFirst(s, old, new string) string {
	for i := 0; i <= len(s)-len(old); i++ {
		if s[i:i+len(old)] == old {
			return s[:i] + new + s[i+len(old):]
		}
	}
	return s
}

// ResolveToolLabel produces a human-readable label from a generic tool name
// by inspecting the tool call arguments.
func ResolveToolLabel(name, displayName string, args []ToolCallArg) string {
	switch name {
	case "runSkillTool":
		if n := findArg(args, "name"); n != "" {
			return n
		}
		return displayName
	case "loadSkill":
		if id := findArg(args, "skillId"); id != "" {
			return "loadSkill(" + truncate(id, 20) + ")"
		}
		return "loadSkill"
	case "discoverSkills":
		if q := findArg(args, "query"); q != "" {
			return "discoverSkills(\"" + truncate(q, 30) + "\")"
		}
		return "discoverSkills"
	default:
		if displayName != "" {
			return displayName
		}
		return name
	}
}

func findArg(args []ToolCallArg, name string) string {
	for _, a := range args {
		if a.Name == name {
			var s string
			if err := json.Unmarshal(a.Value, &s); err == nil {
				return s
			}
			return string(a.Value)
		}
	}
	return ""
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}
