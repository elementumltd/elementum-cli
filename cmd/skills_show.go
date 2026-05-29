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

var skillsShowCmd = &cobra.Command{
	Use:   "show <namespace> <skill-name>",
	Short: "Show skill details",
	Long: `Display detailed information about an agentic skill within an app.

Examples:
  ei skills show support-tickets "Ticket Triage"
  ei skills show clm escalation-handler --json`,
	Args: cobra.ExactArgs(2),
	RunE: runSkillsShow,
}

func init() {
	skillsCmd.AddCommand(skillsShowCmd)
}

func runSkillsShow(cmd *cobra.Command, args []string) error {
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

	detail, err := getSkillFull(ctx, apiClient, skill.ID)
	if err != nil {
		return err
	}

	if isJSONOutput(cmd) {
		return outputJSON(detail)
	}

	fmt.Println()
	fmt.Println(ui.TitleStyle.Render(detail.Name))
	fmt.Println()
	fmt.Printf("  %s  %s\n", ui.LabelStyle.Render("ID:"), detail.ID)
	fmt.Printf("  %s  %s\n", ui.LabelStyle.Render("Status:"), detail.Status)
	fmt.Printf("  %s  %s\n", ui.LabelStyle.Render("Type:"), detail.Type)
	fmt.Printf("  %s  %s\n", ui.LabelStyle.Render("Description:"), detail.Description)

	fmt.Println()
	fmt.Println(ui.SubtitleStyle.Render("Instructions:"))
	fmt.Println(detail.Instructions)

	if len(detail.Tools) > 0 {
		fmt.Println()
		fmt.Println(ui.SubtitleStyle.Render(fmt.Sprintf("Tools (%d):", len(detail.Tools))))
		for i, tool := range detail.Tools {
			fmt.Printf("  %d. %s (%s) - %s\n", i+1, tool.Name, tool.ToolType, tool.ID)
			if tool.Description != "" {
				desc := tool.Description
				if len(desc) > 70 {
					desc = desc[:67] + "..."
				}
				fmt.Printf("     %s\n", desc)
			}
		}
	}

	fmt.Println()
	return nil
}

type skillDetail struct {
	ID           string           `json:"id"`
	Name         string           `json:"name"`
	Description  string           `json:"description"`
	Instructions string           `json:"instructions"`
	Status       string           `json:"status"`
	Type         string           `json:"type"`
	Tools        []skillToolBrief `json:"tools,omitempty"`
}

type skillToolBrief struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	ToolType    string `json:"type"`
	Description string `json:"description,omitempty"`
}

func getSkillFull(ctx context.Context, apiClient *client.Client, skillID string) (*skillDetail, error) {
	var result struct {
		Organization struct {
			AgenticSkill *struct {
				ID           string `json:"id"`
				Name         string `json:"name"`
				Description  string `json:"description"`
				Instructions string `json:"instructions"`
				Status       string `json:"status"`
				Type         string `json:"type"`
				Tools        struct {
					Edges []struct {
						Node struct {
							ID          string `json:"id"`
							Name        string `json:"name"`
							Type        string `json:"type"`
							Description string `json:"description"`
						} `json:"node"`
					} `json:"edges"`
				} `json:"tools"`
			} `json:"agenticSkill"`
		} `json:"organization"`
	}

	query := `
		query GetSkillFull($id: ID!) {
			organization {
				agenticSkill(id: $id) {
					id
					name
					description
					instructions
					status
					type
					tools {
						edges {
							node {
								id
								name
								type
								description
							}
						}
					}
				}
			}
		}
	`

	err := apiClient.ExecuteInto(ctx, query, map[string]interface{}{"id": skillID}, &result)
	if err != nil {
		return nil, fmt.Errorf("failed to query skill: %w", err)
	}

	if result.Organization.AgenticSkill == nil {
		return nil, fmt.Errorf("skill not found: %s", skillID)
	}

	s := result.Organization.AgenticSkill
	detail := &skillDetail{
		ID:           s.ID,
		Name:         s.Name,
		Description:  s.Description,
		Instructions: s.Instructions,
		Status:       s.Status,
		Type:         s.Type,
	}

	for _, edge := range s.Tools.Edges {
		detail.Tools = append(detail.Tools, skillToolBrief{
			ID:          edge.Node.ID,
			Name:        edge.Node.Name,
			ToolType:    edge.Node.Type,
			Description: edge.Node.Description,
		})
	}

	return detail, nil
}
