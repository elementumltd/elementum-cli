// Copyright (c) ElementumAI, Inc.
// SPDX-License-Identifier: MPL-2.0

package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/elementumltd/elementum-cli/auth"
	"github.com/elementumltd/elementum-cli/ui"
	"github.com/elementumltd/elementum-cli/internal/client"
	"github.com/spf13/cobra"
)

var agentsShowCmd = &cobra.Command{
	Use:   "show <agent-name-or-id>",
	Short: "Show agent details",
	Long: `Display detailed information about an agent.

Examples:
  ei agents show "Support Agent"
  ei agents show 794e1e48-73af-4760-8a1f-...`,
	Args: cobra.ExactArgs(1),
	RunE: runAgentsShow,
}

func runAgentsShow(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	apiClient, err := auth.GetClientFromCmd(cmd)
	if err != nil {
		return err
	}

	agentID, err := resolveAgentID(ctx, cmd, apiClient, args[0])
	if err != nil {
		return err
	}

	agent, err := getAgentFull(ctx, apiClient, agentID)
	if err != nil {
		return err
	}

	if isJSONOutput(cmd) {
		return outputJSON(agent)
	}

	fmt.Println()
	fmt.Println(ui.TitleStyle.Render(agent.Name))
	fmt.Println()
	fmt.Printf("  %s  %s\n", ui.LabelStyle.Render("ID:"), agent.ID)
	fmt.Printf("  %s  %s\n", ui.LabelStyle.Render("Type:"), agent.Type)
	fmt.Printf("  %s  %s\n", ui.LabelStyle.Render("Description:"), agent.Description)
	if agent.Model != "" {
		fmt.Printf("  %s  %s (%s)\n", ui.LabelStyle.Render("Model:"), agent.Model, agent.Provider)
	}
	if agent.FirstMessage != "" {
		fmt.Printf("  %s  %s\n", ui.LabelStyle.Render("First Msg:"), agent.FirstMessage)
	}
	if agent.AppName != "" {
		fmt.Printf("  %s  %s (%s)\n", ui.LabelStyle.Render("App:"), agent.AppName, agent.AppID)
	}

	fmt.Println()
	fmt.Println(ui.SubtitleStyle.Render("Instructions:"))
	fmt.Println(agent.Instructions)

	if len(agent.Tools) > 0 {
		fmt.Println()
		fmt.Println(ui.SubtitleStyle.Render(fmt.Sprintf("Tools (%d):", len(agent.Tools))))
		for i, tool := range agent.Tools {
			fmt.Printf("  %d. %s (%s) - %s\n", i+1, tool.Name, tool.Type, tool.ID)
			if tool.Description != "" {
				desc := tool.Description
				if len(desc) > 70 {
					desc = desc[:67] + "..."
				}
				fmt.Printf("     %s\n", desc)
			}
		}
	}

	if len(agent.Skills) > 0 {
		fmt.Println()
		fmt.Println(ui.SubtitleStyle.Render(fmt.Sprintf("Skills (%d):", len(agent.Skills))))
		for i, skill := range agent.Skills {
			fmt.Printf("  %d. %s - %s\n", i+1, skill.Name, skill.ID)
		}
	}

	fmt.Println()
	return nil
}

type agentDetail struct {
	ID           string       `json:"id"`
	Name         string       `json:"name"`
	Description  string       `json:"description"`
	Type         string       `json:"type"`
	Instructions string       `json:"instructions"`
	FirstMessage string       `json:"first_message,omitempty"`
	Model        string       `json:"model,omitempty"`
	Provider     string       `json:"provider,omitempty"`
	ConnectorID  string       `json:"connector_id,omitempty"`
	AppID        string       `json:"app_id,omitempty"`
	AppName      string       `json:"app_name,omitempty"`
	Tools        []toolBrief  `json:"tools,omitempty"`
	Skills       []skillBrief `json:"skills,omitempty"`
}

type toolBrief struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Type        string `json:"type"`
	Description string `json:"description,omitempty"`
}

type skillBrief struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func getAgentFull(ctx context.Context, apiClient *client.Client, agentID string) (*agentDetail, error) {
	var result struct {
		Organization struct {
			Agent *json.RawMessage `json:"agent"`
		} `json:"organization"`
	}

	query := `
		query GetAgentFull($agentId: ID!) {
			organization {
				agent(id: $agentId) {
					__typename
					id
					name
					description
					... on AgentElementum {
						instructions
						firstMessage
						aiProviderConnector { id model { name } provider { name } }
						app { id name }
						tools { edges { node { __typename id name description } } }
						skills { id name }
					}
					... on AgentSnowflake {
						instructions
						firstMessage
						aiProviderConnector { id model { name } provider { name } }
						app { id name }
						tools { edges { node { __typename id name description } } }
						skills { id name }
					}
					... on AgentBedrock {
						instructions
						firstMessage
						aiProviderConnector { id model { name } provider { name } }
						app { id name }
						tools { edges { node { __typename id name description } } }
						skills { id name }
					}
					... on AgentBrowserUse {
						instructions
						firstMessage
						persistSession
						flashMode
						vision
						sessionTimeoutMinutes
						aiProviderConnector { id model { name } provider { name } }
						app { id name }
						tools { edges { node { __typename id name description } } }
						skills { id name }
					}
				}
			}
		}
	`

	err := apiClient.ExecuteInto(ctx, query, map[string]interface{}{"agentId": agentID}, &result)
	if err != nil {
		return nil, fmt.Errorf("failed to query agent: %w", err)
	}

	if result.Organization.Agent == nil {
		return nil, fmt.Errorf("agent not found: %s", agentID)
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(*result.Organization.Agent, &raw); err != nil {
		return nil, fmt.Errorf("failed to parse agent: %w", err)
	}

	detail := &agentDetail{
		ID:          getString(raw, "id"),
		Name:        getString(raw, "name"),
		Description: getString(raw, "description"),
		Type:        strings.TrimPrefix(getString(raw, "__typename"), "Agent"),
	}

	detail.Instructions = getString(raw, "instructions")
	detail.FirstMessage = getString(raw, "firstMessage")

	if connector, ok := raw["aiProviderConnector"].(map[string]interface{}); ok {
		detail.ConnectorID = getString(connector, "id")
		if m, ok := connector["model"].(map[string]interface{}); ok {
			detail.Model = getString(m, "name")
		}
		if p, ok := connector["provider"].(map[string]interface{}); ok {
			detail.Provider = getString(p, "name")
		}
	}

	if app, ok := raw["app"].(map[string]interface{}); ok {
		detail.AppID = getString(app, "id")
		detail.AppName = getString(app, "name")
	}

	if tools, ok := raw["tools"].(map[string]interface{}); ok {
		if edges, ok := tools["edges"].([]interface{}); ok {
			for _, edge := range edges {
				if e, ok := edge.(map[string]interface{}); ok {
					if node, ok := e["node"].(map[string]interface{}); ok {
						detail.Tools = append(detail.Tools, toolBrief{
							ID:          getString(node, "id"),
							Name:        getString(node, "name"),
							Type:        strings.TrimPrefix(getString(node, "__typename"), "Agent"),
							Description: getString(node, "description"),
						})
					}
				}
			}
		}
	}

	if skills, ok := raw["skills"].([]interface{}); ok {
		for _, s := range skills {
			if sk, ok := s.(map[string]interface{}); ok {
				detail.Skills = append(detail.Skills, skillBrief{
					ID:   getString(sk, "id"),
					Name: getString(sk, "name"),
				})
			}
		}
	}

	return detail, nil
}

func getString(m map[string]interface{}, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}
