// Copyright (c) ElementumAI, Inc.
// SPDX-License-Identifier: MPL-2.0

package cmd

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/elementumltd/elementum-cli/auth"
	"github.com/elementumltd/elementum-cli/ui"
	"github.com/spf13/cobra"
)

var agentToolsCmd = &cobra.Command{
	Use:   "agent-tools",
	Short: "Manage agent tools",
	Long:  "Commands for listing, creating, updating, and deleting agent tools.",
}

var agentToolsListCmd = &cobra.Command{
	Use:   "list <agent-name-or-id>",
	Short: "List tools on an agent",
	Args:  cobra.ExactArgs(1),
	RunE:  runListAgentTools,
}

var agentToolsDeleteCmd = &cobra.Command{
	Use:   "delete <tool-id>",
	Short: "Delete an agent tool",
	Args:  cobra.ExactArgs(1),
	RunE:  runDeleteAgentTool,
}

func init() {
	agentToolsCmd.AddCommand(agentToolsListCmd)
	agentToolsCmd.AddCommand(agentToolsDeleteCmd)
}

// GetAgentToolsCmd returns the agent-tools command for registration
func GetAgentToolsCmd() *cobra.Command {
	return agentToolsCmd
}

func runListAgentTools(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	apiClient, err := auth.GetClientFromCmd(cmd)
	if err != nil {
		return err
	}

	agentID, err := resolveAgentID(ctx, cmd, apiClient, args[0])
	if err != nil {
		return err
	}

	var result struct {
		Organization struct {
			Agent *struct {
				ID    string `json:"id"`
				Name  string `json:"name"`
				Tools struct {
					Edges []struct {
						Node json.RawMessage `json:"node"`
					} `json:"edges"`
				} `json:"tools"`
			} `json:"agent"`
		} `json:"organization"`
	}

	query := `
		query GetAgentTools($agentId: ID!) {
			organization {
				agent(id: $agentId) {
					id
					name
					tools {
						edges {
							node {
								__typename
								id
								name
								description
								... on AgentExecuteWorkflowTool {
									automation {
										id
										name
									}
								}
								... on AgentSearchTableTool {
									table {
										id
									}
								}
							}
						}
					}
				}
			}
		}
	`

	err = apiClient.ExecuteInto(ctx, query, map[string]interface{}{"agentId": agentID}, &result)
	if err != nil {
		return fmt.Errorf("failed to query agent tools: %w", err)
	}

	if result.Organization.Agent == nil {
		return fmt.Errorf("agent not found: %s", agentID)
	}

	agent := result.Organization.Agent
	tools := agent.Tools.Edges

	if len(tools) == 0 {
		fmt.Println(ui.WarningStyle.Render("No tools found on this agent."))
		return nil
	}

	fmt.Println()
	fmt.Println(ui.TitleStyle.Render(fmt.Sprintf("Tools on %s", agent.Name)))
	fmt.Println()

	for i, edge := range tools {
		var tool map[string]interface{}
		json.Unmarshal(edge.Node, &tool)

		fmt.Printf("%d. %s (%s)\n", i+1, tool["name"], tool["__typename"])
		fmt.Printf("   ID: %s\n", tool["id"])
		if desc, ok := tool["description"].(string); ok && desc != "" {
			if len(desc) > 80 {
				desc = desc[:77] + "..."
			}
			fmt.Printf("   Description: %s\n", desc)
		}
		if automation, ok := tool["automation"].(map[string]interface{}); ok {
			fmt.Printf("   Automation: %s (%s)\n", automation["name"], automation["id"])
		}
		if table, ok := tool["table"].(map[string]interface{}); ok {
			fmt.Printf("   Search Table: %s\n", table["id"])
		}
		fmt.Println()
	}

	fmt.Println(ui.MutedStyle.Render(fmt.Sprintf("Total: %d tools", len(tools))))
	fmt.Println()
	fmt.Println(ui.MutedStyle.Render("Use 'ei agent-tools delete <tool-id>' to remove a tool"))
	fmt.Println()

	return nil
}

func runDeleteAgentTool(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	toolID := args[0]

	apiClient, err := auth.GetClientFromCmd(cmd)
	if err != nil {
		return err
	}

	var result struct {
		AgentToolDelete *struct {
			ID string `json:"id"`
		} `json:"agentToolDelete"`
	}

	mutation := `
		mutation DeleteAgentTool($id: ID!) {
			agentToolDelete(id: $id) {
				id
			}
		}
	`

	err = apiClient.ExecuteInto(ctx, mutation, map[string]interface{}{"id": toolID}, &result)
	if err != nil {
		return fmt.Errorf("failed to delete agent tool: %w", err)
	}

	fmt.Println(ui.SuccessStyle.Render(fmt.Sprintf("Deleted agent tool %s", toolID)))
	return nil
}
