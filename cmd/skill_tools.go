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

	"github.com/elementumltd/elementum-cli/auth"
	"github.com/elementumltd/elementum-cli/ui"
	"github.com/elementumltd/elementum-cli/internal/client"
	"github.com/spf13/cobra"
)

var skillToolsCmd = &cobra.Command{
	Use:   "skill-tools",
	Short: "Manage agentic skill tools",
	Long:  "Commands for listing, creating, updating, and deleting agentic skill tools.",
}

var skillToolsListCmd = &cobra.Command{
	Use:   "list <skill-id>",
	Short: "List tools on an agentic skill",
	Args:  cobra.ExactArgs(1),
	RunE:  runListSkillTools,
}

var skillToolsDeleteCmd = &cobra.Command{
	Use:   "delete <tool-id>",
	Short: "Delete an agentic skill tool",
	Args:  cobra.ExactArgs(1),
	RunE:  runDeleteSkillTool,
}

var skillToolsDeleteAllCmd = &cobra.Command{
	Use:   "delete-all <skill-id>",
	Short: "Delete all tools on an agentic skill",
	Args:  cobra.ExactArgs(1),
	RunE:  runDeleteAllSkillTools,
}

func init() {
	skillToolsCmd.AddCommand(skillToolsListCmd)
	skillToolsCmd.AddCommand(skillToolsDeleteCmd)
	skillToolsCmd.AddCommand(skillToolsDeleteAllCmd)
}

// GetSkillToolsCmd returns the skill-tools command for registration
func GetSkillToolsCmd() *cobra.Command {
	return skillToolsCmd
}

func runListSkillTools(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	skillID := args[0]

	apiClient, err := auth.GetClientFromCmd(cmd)
	if err != nil {
		return err
	}

	tools, err := getSkillTools(ctx, apiClient, skillID)
	if err != nil {
		return err
	}

	if len(tools) == 0 {
		fmt.Println(ui.WarningStyle.Render("No tools found on this skill."))
		return nil
	}

	table := ui.NewTable([]string{"NAME", "ID", "AUTOMATION ID"})
	for _, tool := range tools {
		automationID := "(none)"
		if tool.AutomationID != "" {
			automationID = tool.AutomationID
		}
		table.AddRow(tool.Name, tool.ID, automationID)
	}

	fmt.Println()
	fmt.Println(ui.TitleStyle.Render("Skill Tools"))
	fmt.Println()
	fmt.Println(table.Render())
	fmt.Println()
	fmt.Println(ui.MutedStyle.Render(fmt.Sprintf("Total: %d tools", len(tools))))
	fmt.Println()

	return nil
}

func runDeleteSkillTool(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	toolID := args[0]

	apiClient, err := auth.GetClientFromCmd(cmd)
	if err != nil {
		return err
	}

	err = deleteSkillTool(ctx, apiClient, toolID)
	if err != nil {
		return fmt.Errorf("failed to delete tool: %w", err)
	}

	fmt.Println(ui.SuccessStyle.Render(fmt.Sprintf("Deleted tool %s", toolID)))
	return nil
}

func runDeleteAllSkillTools(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	skillID := args[0]

	apiClient, err := auth.GetClientFromCmd(cmd)
	if err != nil {
		return err
	}

	tools, err := getSkillTools(ctx, apiClient, skillID)
	if err != nil {
		return err
	}

	if len(tools) == 0 {
		fmt.Println(ui.WarningStyle.Render("No tools found on this skill."))
		return nil
	}

	fmt.Println(ui.InfoStyle.Render(fmt.Sprintf("Deleting %d tools...", len(tools))))

	for _, tool := range tools {
		err = deleteSkillTool(ctx, apiClient, tool.ID)
		if err != nil {
			fmt.Println(ui.ErrorStyle.Render(fmt.Sprintf("Failed to delete %s: %v", tool.Name, err)))
		} else {
			fmt.Println(ui.SuccessStyle.Render(fmt.Sprintf("Deleted %s", tool.Name)))
		}
	}

	return nil
}

type skillToolInfo struct {
	ID           string
	Name         string
	AutomationID string
}

func getSkillTools(ctx context.Context, apiClient *client.Client, skillID string) ([]skillToolInfo, error) {
	var result struct {
		Organization struct {
			AgenticSkill *struct {
				ID    string `json:"id"`
				Tools struct {
					Edges []struct {
						Node struct {
							ID         string `json:"id"`
							Name       string `json:"name"`
							Automation *struct {
								ID string `json:"id"`
							} `json:"automation"`
						} `json:"node"`
					} `json:"edges"`
				} `json:"tools"`
			} `json:"agenticSkill"`
		} `json:"organization"`
	}

	query := `
		query GetSkillTools($skillId: ID!) {
			organization {
				agenticSkill(id: $skillId) {
					id
					tools {
						edges {
							node {
								id
								name
								... on AgenticSkillAutomationTool {
									automation {
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

	err := apiClient.ExecuteInto(ctx, query, map[string]interface{}{"skillId": skillID}, &result)
	if err != nil {
		return nil, fmt.Errorf("failed to query skill tools: %w", err)
	}

	if result.Organization.AgenticSkill == nil {
		return nil, fmt.Errorf("skill not found: %s", skillID)
	}

	var tools []skillToolInfo
	for _, edge := range result.Organization.AgenticSkill.Tools.Edges {
		tool := skillToolInfo{
			ID:   edge.Node.ID,
			Name: edge.Node.Name,
		}
		if edge.Node.Automation != nil {
			tool.AutomationID = edge.Node.Automation.ID
		}
		tools = append(tools, tool)
	}

	return tools, nil
}

func deleteSkillTool(ctx context.Context, apiClient *client.Client, toolID string) error {
	var result struct {
		AgenticSkillToolDelete *struct {
			ID string `json:"id"`
		} `json:"agenticSkillToolDelete"`
	}

	mutation := `
		mutation DeleteSkillTool($id: ID!) {
			agenticSkillToolDelete(id: $id) {
				id
			}
		}
	`

	return apiClient.ExecuteInto(ctx, mutation, map[string]interface{}{"id": toolID}, &result)
}
