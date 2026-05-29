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

package discovery

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/elementumltd/elementum-cli/internal/client"
	"github.com/elementumltd/elementum-cli/logger"
)

// GetSkillsForApp fetches all agentic skills and their tools for an aspect
// (app or element). Skills are a distinct subsystem from agents: an agent
// references skills via `skill_ids`, but the skill itself is a first-class
// resource with its own typed tools. The existing CLI commands
// (`ei skills list`, `ei skill-tools list`) only fetch a shallow view; here
// we fetch everything needed to roundtrip a `skill-*.tf` file.
func GetSkillsForApp(ctx context.Context, c *client.Client, aspectID string) ([]AgenticSkill, error) {
	query := `
		query GetAppSkillsWithTools($aspectId: ID!) {
			organization {
				aspect(id: $aspectId) {
					... on AspectApp {
						agenticSkills(first: 500) {
							edges {
								node {
									id
									name
									description
									instructions
									status
									type
									properties
									tools(first: 500) {
										edges { node { ...AgenticSkillToolAllFields } }
									}
								}
							}
						}
					}
				}
			}
		}

		fragment AgenticSkillToolAllFields on AgenticSkillTool {
			__typename
			id
			name
			description
			status
			type
			... on AgenticSkillAutomationTool {
				startMessage
				automation { id }
			}
			... on AgenticSkillCreateRecordTool {
				startMessage
				aspect { __typename id }
				fields {
					field { id name }
					name
					description
					required
				}
			}
			... on AgenticSkillUpdateRecordTool {
				startMessage
				handleDescription
				aspect { __typename id }
				fields {
					field { id name }
					name
					description
					required
				}
			}
			... on AgenticSkillSearchAspectTool {
				startMessage
				queryDescription
				limit
				aspect { __typename id }
				fields {
					field { id name }
					name
					description
				}
			}
			... on AgenticSkillSearchTableTool {
				startMessage
				queryDescription
				limit
				table { __typename id }
				fields {
					field { id name }
					name
					description
				}
			}
			... on AgenticSkillRunAgentTool {
				startMessage
				workerTaskPrompt
				targetAgent { id }
			}
		}
	`

	var result struct {
		Organization struct {
			Aspect map[string]interface{} `json:"aspect"`
		} `json:"organization"`
	}

	if err := c.ExecuteInto(ctx, query, map[string]interface{}{"aspectId": aspectID}, &result); err != nil {
		return nil, fmt.Errorf("failed to fetch agentic skills for aspect %s: %w", aspectID, err)
	}

	if result.Organization.Aspect == nil {
		return nil, nil
	}

	raw, ok := result.Organization.Aspect["agenticSkills"].(map[string]interface{})
	if !ok {
		return nil, nil
	}
	edges, ok := raw["edges"].([]interface{})
	if !ok {
		return nil, nil
	}

	skills := make([]AgenticSkill, 0, len(edges))
	for _, edge := range edges {
		edgeMap, ok := edge.(map[string]interface{})
		if !ok {
			continue
		}
		node, ok := edgeMap["node"].(map[string]interface{})
		if !ok {
			continue
		}
		skill := extractAgenticSkill(node)
		skill.OwnerID = aspectID
		skill.OwnerType = inferOwnerType(result.Organization.Aspect)
		skills = append(skills, skill)
	}

	logger.Debug("discovered agentic skills", "aspectID", aspectID, "skillCount", len(skills))
	return skills, nil
}

func extractAgenticSkill(node map[string]interface{}) AgenticSkill {
	skill := AgenticSkill{
		ID:           getString(node, "id"),
		Name:         getString(node, "name"),
		Description:  getString(node, "description"),
		Instructions: getString(node, "instructions"),
		Status:       getString(node, "status"),
		Type:         getString(node, "type"),
	}

	// properties is JSON — keep whatever the platform returns verbatim (may
	// be nil, a string, or a nested object). Only emit if non-empty object.
	if props, ok := node["properties"].(map[string]interface{}); ok && len(props) > 0 {
		if s := marshalJSON(props); s != "" {
			skill.Properties = s
		}
	}

	if toolsWrap, ok := node["tools"].(map[string]interface{}); ok {
		if edges, ok := toolsWrap["edges"].([]interface{}); ok {
			for _, edge := range edges {
				edgeMap, ok := edge.(map[string]interface{})
				if !ok {
					continue
				}
				toolNode, ok := edgeMap["node"].(map[string]interface{})
				if !ok {
					continue
				}
				tool := extractAgenticSkillTool(toolNode)
				tool.SkillID = skill.ID
				skill.Tools = append(skill.Tools, tool)
			}
		}
	}

	return skill
}

func extractAgenticSkillTool(node map[string]interface{}) AgenticSkillTool {
	typename := getString(node, "__typename")
	tool := AgenticSkillTool{
		ID:           getString(node, "id"),
		Name:         getString(node, "name"),
		Description:  getString(node, "description"),
		Status:       getString(node, "status"),
		StartMessage: getString(node, "startMessage"),
	}

	switch typename {
	case "AgenticSkillAutomationTool":
		tool.Type = "automation"
		if auto, ok := node["automation"].(map[string]interface{}); ok {
			tool.AutomationID = getString(auto, "id")
		}
	case "AgenticSkillCreateRecordTool":
		tool.Type = "create_record"
		if asp, ok := node["aspect"].(map[string]interface{}); ok {
			tool.TargetAspectID = getString(asp, "id")
		}
		tool.Fields = extractAgenticSkillFields(node)
	case "AgenticSkillUpdateRecordTool":
		tool.Type = "update_record"
		if asp, ok := node["aspect"].(map[string]interface{}); ok {
			tool.TargetAspectID = getString(asp, "id")
		}
		tool.HandleDescription = getString(node, "handleDescription")
		tool.Fields = extractAgenticSkillFields(node)
	case "AgenticSkillSearchAspectTool":
		tool.Type = "search_aspect"
		if asp, ok := node["aspect"].(map[string]interface{}); ok {
			tool.TargetAspectID = getString(asp, "id")
		}
		tool.QueryDescription = getString(node, "queryDescription")
		tool.ResultLimit = int(getFloat(node, "limit"))
		tool.Fields = extractAgenticSkillFields(node)
	case "AgenticSkillSearchTableTool":
		tool.Type = "search_table"
		if t, ok := node["table"].(map[string]interface{}); ok {
			tool.SearchTableID = getString(t, "id")
		}
		tool.QueryDescription = getString(node, "queryDescription")
		tool.ResultLimit = int(getFloat(node, "limit"))
		tool.Fields = extractAgenticSkillFields(node)
	case "AgenticSkillRunAgentTool":
		tool.Type = "run_agent"
		if tgt, ok := node["targetAgent"].(map[string]interface{}); ok {
			tool.TargetAgentID = getString(tgt, "id")
		}
		tool.WorkerTaskPrompt = getString(node, "workerTaskPrompt")
	}
	return tool
}

func extractAgenticSkillFields(node map[string]interface{}) []AgenticSkillField {
	raw, ok := node["fields"].([]interface{})
	if !ok {
		return nil
	}
	out := make([]AgenticSkillField, 0, len(raw))
	for _, f := range raw {
		fm, ok := f.(map[string]interface{})
		if !ok {
			continue
		}
		field := AgenticSkillField{
			Name:        getString(fm, "name"),
			Description: getString(fm, "description"),
			Required:    getBool(fm, "required"),
		}
		if fRef, ok := fm["field"].(map[string]interface{}); ok {
			field.FieldID = getString(fRef, "id")
		}
		out = append(out, field)
	}
	return out
}

// inferOwnerType reads the aspect's __typename to build an owner_type that
// matches the terraform provider's accepted values.
func inferOwnerType(aspect map[string]interface{}) string {
	switch getString(aspect, "__typename") {
	case "AspectElement":
		return "ELEMENT_ASPECT"
	default:
		// Absent __typename (we don't select it on AspectApp above) defaults
		// to APP_ASPECT, which is correct for our main code path.
		return "APP_ASPECT"
	}
}

// getString is defined in app.go and shared across the package.

func getBool(m map[string]interface{}, key string) bool {
	if v, ok := m[key].(bool); ok {
		return v
	}
	return false
}

func getFloat(m map[string]interface{}, key string) float64 {
	if v, ok := m[key].(float64); ok {
		return v
	}
	return 0
}

func marshalJSON(v interface{}) string {
	b, err := json.Marshal(v)
	if err != nil {
		return ""
	}
	return string(b)
}

// GetA2ASkillsForAgent fetches the A2A (agent-to-agent) skills declared on
// an AgentElementum's AgentCard. A2A skills are the routing advertisement
// other agents use to decide which peer agent should handle a request.
// Returns nil for agent types that don't support AgentCard (Bedrock,
// Snowflake, BrowserUse).
func GetA2ASkillsForAgent(ctx context.Context, c *client.Client, agentID string) ([]AgentA2ASkill, error) {
	resp, err := client.GetAgentA2ASkills(ctx, c.Genqlient(), agentID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch a2a skills for agent %s: %w", agentID, err)
	}
	if resp.Organization.Agent == nil {
		return nil, nil
	}
	agent := *resp.Organization.Agent
	elementumAgent, ok := agent.(*client.GetAgentA2ASkillsOrganizationAgentAgentElementum)
	if !ok || elementumAgent.Card == nil {
		return nil, nil
	}

	out := make([]AgentA2ASkill, 0, len(elementumAgent.Card.Skills.Edges))
	for _, edge := range elementumAgent.Card.Skills.Edges {
		out = append(out, AgentA2ASkill{
			ID:          edge.Node.Id,
			AgentID:     agentID,
			Name:        edge.Node.Name,
			Description: edge.Node.Description,
			Tags:        edge.Node.Tags,
			Examples:    edge.Node.Examples,
			InputModes:  edge.Node.InputModes,
			OutputModes: edge.Node.OutputModes,
		})
	}
	return out, nil
}
