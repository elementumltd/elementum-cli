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
	"github.com/elementumltd/elementum-cli/internal/client"
	"github.com/elementumltd/elementum-cli/ui"
	"github.com/spf13/cobra"
)

var skillsCmd = &cobra.Command{
	Use:   "skills",
	Short: "Manage agentic skills",
	Long:  "Commands for listing, creating, showing, updating, and deleting agentic skills.",
}

var skillsListCmd = &cobra.Command{
	Use:   "list <namespace>",
	Short: "List agentic skills in an app",
	Long: `List agentic skills belonging to an app.

Examples:
  ei skills list support-tickets
  ei skills list clm --json`,
	Args: cobra.ExactArgs(1),
	RunE: runListSkills,
}

var skillsDeleteCmd = &cobra.Command{
	Use:   "delete <namespace> <skill-name>",
	Short: "Delete an agentic skill",
	Long: `Delete an agentic skill by name within an app. Also deletes all tools on the skill.

Examples:
  ei skills delete support-tickets "Ticket Triage"
  ei skills delete clm escalation-handler`,
	Args: cobra.ExactArgs(2),
	RunE: runDeleteSkill,
}

func init() {
	skillsCmd.AddCommand(skillsListCmd)
	skillsCmd.AddCommand(skillsDeleteCmd)
}

// GetSkillsCmd returns the skills command for registration
func GetSkillsCmd() *cobra.Command {
	return skillsCmd
}

type skillInfo struct {
	ID          string
	Name        string
	Description string
	Status      string
	OwnerName   string
	OwnerID     string
	ToolCount   int
}

func runListSkills(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	namespace := args[0]

	apiClient, err := auth.GetClientFromCmd(cmd)
	if err != nil {
		return err
	}

	// Resolve namespace to aspect ID
	aspectID, _, err := resolveAspectByNamespace(ctx, apiClient, namespace)
	if err != nil {
		return fmt.Errorf("failed to resolve app %q: %w", namespace, err)
	}

	skills, err := listSkillsInApp(ctx, apiClient, aspectID)
	if err != nil {
		return err
	}

	if isJSONOutput(cmd) {
		return outputJSON(skills)
	}

	if len(skills) == 0 {
		fmt.Println(ui.WarningStyle.Render(fmt.Sprintf("No skills found in app %q.", namespace)))
		return nil
	}

	table := ui.NewTable([]string{"NAME", "ID", "STATUS", "TOOLS"})
	for _, skill := range skills {
		table.AddRow(skill.Name, skill.ID, skill.Status, fmt.Sprintf("%d", skill.ToolCount))
	}

	fmt.Println()
	fmt.Println(ui.TitleStyle.Render(fmt.Sprintf("Skills in %s", namespace)))
	fmt.Println()
	fmt.Println(table.Render())
	fmt.Println()
	fmt.Println(ui.MutedStyle.Render(fmt.Sprintf("Total: %d skills", len(skills))))
	fmt.Println()

	return nil
}

func runDeleteSkill(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	namespace := args[0]
	skillName := args[1]

	apiClient, err := auth.GetClientFromCmd(cmd)
	if err != nil {
		return err
	}

	// Resolve namespace to aspect ID
	aspectID, _, err := resolveAspectByNamespace(ctx, apiClient, namespace)
	if err != nil {
		return fmt.Errorf("failed to resolve app %q: %w", namespace, err)
	}

	// Find skill by name in the app
	skill, err := findSkillByNameInApp(ctx, apiClient, aspectID, skillName)
	if err != nil {
		return fmt.Errorf("failed to find skill %q in app %q: %w", skillName, namespace, err)
	}

	if !isJSONOutput(cmd) {
		fmt.Println(ui.InfoStyle.Render(fmt.Sprintf("Found skill: %s (%s)", skill.Name, skill.ID)))
	}

	// First delete all tools on the skill
	tools, err := getSkillTools(ctx, apiClient, skill.ID)
	if err != nil {
		fmt.Println(ui.WarningStyle.Render(fmt.Sprintf("Could not list tools: %v", err)))
	} else if len(tools) > 0 {
		fmt.Println(ui.InfoStyle.Render(fmt.Sprintf("Deleting %d tools first...", len(tools))))
		for _, tool := range tools {
			err = deleteSkillTool(ctx, apiClient, tool.ID)
			if err != nil {
				fmt.Println(ui.WarningStyle.Render(fmt.Sprintf("  Failed to delete tool %s: %v", tool.Name, err)))
			} else {
				fmt.Println(ui.MutedStyle.Render(fmt.Sprintf("  Deleted tool: %s", tool.Name)))
			}
		}
	}

	// Now delete the skill
	err = deleteSkill(ctx, apiClient, skill.ID)
	if err != nil {
		return fmt.Errorf("failed to delete skill: %w", err)
	}

	fmt.Println(ui.SuccessStyle.Render(fmt.Sprintf("Deleted skill %s", skill.ID)))
	return nil
}

func listSkillsInApp(ctx context.Context, apiClient *client.Client, aspectID string) ([]skillInfo, error) {
	var result struct {
		Organization struct {
			Aspect *struct {
				AgenticSkills struct {
					Edges []struct {
						Node struct {
							ID          string `json:"id"`
							Name        string `json:"name"`
							Description string `json:"description"`
							Status      string `json:"status"`
							Tools       struct {
								Edges []struct {
									Node struct {
										ID string `json:"id"`
									} `json:"node"`
								} `json:"edges"`
							} `json:"tools"`
						} `json:"node"`
					} `json:"edges"`
				} `json:"agenticSkills"`
			} `json:"aspect"`
		} `json:"organization"`
	}

	query := `
		query ListAppSkills($aspectId: ID!) {
			organization {
				aspect(id: $aspectId) {
					... on AspectApp {
						agenticSkills {
							edges {
								node {
									id
									name
									description
									status
									tools {
										edges {
											node {
												id
											}
										}
									}
								}
							}
						}
					}
				}
			}
		}
	`

	err := apiClient.ExecuteInto(ctx, query, map[string]interface{}{"aspectId": aspectID}, &result)
	if err != nil {
		return nil, fmt.Errorf("failed to query skills: %w", err)
	}

	if result.Organization.Aspect == nil {
		return nil, fmt.Errorf("app not found")
	}

	var skills []skillInfo
	for _, edge := range result.Organization.Aspect.AgenticSkills.Edges {
		skill := skillInfo{
			ID:          edge.Node.ID,
			Name:        edge.Node.Name,
			Description: edge.Node.Description,
			Status:      edge.Node.Status,
			ToolCount:   len(edge.Node.Tools.Edges),
		}
		skills = append(skills, skill)
	}

	return skills, nil
}

func findSkillByNameInApp(ctx context.Context, apiClient *client.Client, aspectID, name string) (*skillInfo, error) {
	skills, err := listSkillsInApp(ctx, apiClient, aspectID)
	if err != nil {
		return nil, err
	}

	for _, skill := range skills {
		if skill.Name == name {
			return &skill, nil
		}
	}

	return nil, fmt.Errorf("skill not found: %s", name)
}

func deleteSkill(ctx context.Context, apiClient *client.Client, skillID string) error {
	var result struct {
		AgenticSkillDelete *struct {
			ID string `json:"id"`
		} `json:"agenticSkillDelete"`
	}

	mutation := `
		mutation DeleteAgenticSkill($id: ID!) {
			agenticSkillDelete(id: $id) {
				id
			}
		}
	`

	return apiClient.ExecuteInto(ctx, mutation, map[string]interface{}{"id": skillID}, &result)
}

// isUUID is defined in export.go
