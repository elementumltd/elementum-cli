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

var a2aSkillsCmd = &cobra.Command{
	Use:   "a2a-skills",
	Short: "Manage A2A (Agent-to-Agent) skills",
	Long: `Commands for listing and deleting A2A skills on agents.

A2A skills define what capabilities an agent exposes to other agents,
including supported input/output MIME types. This enables agents to
discover and invoke each other's capabilities.`,
}

var a2aSkillsListCmd = &cobra.Command{
	Use:   "list <agent-name-or-id>",
	Short: "List A2A skills on an agent",
	Long: `List all A2A skills attached to an agent.

Examples:
  ei a2a-skills list "Support Bot"
  ei a2a-skills list abc123-agent-uuid
  ei a2a-skills list "Sales Assistant" --json`,
	Args: cobra.ExactArgs(1),
	RunE: runListA2ASkills,
}

var a2aSkillsDeleteCmd = &cobra.Command{
	Use:   "delete <skill-id>",
	Short: "Delete an A2A skill",
	Long: `Delete an A2A skill by its ID.

Examples:
  ei a2a-skills delete abc123-skill-uuid`,
	Args: cobra.ExactArgs(1),
	RunE: runDeleteA2ASkill,
}

func init() {
	a2aSkillsCmd.AddCommand(a2aSkillsListCmd)
	a2aSkillsCmd.AddCommand(a2aSkillsDeleteCmd)
}

// GetA2ASkillsCmd returns the a2a-skills command for registration
func GetA2ASkillsCmd() *cobra.Command {
	return a2aSkillsCmd
}

func runListA2ASkills(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	apiClient, err := auth.GetClientFromCmd(cmd)
	if err != nil {
		return err
	}

	agentID, err := resolveAgentID(ctx, cmd, apiClient, args[0])
	if err != nil {
		return err
	}

	skills, agentName, err := getA2ASkills(ctx, apiClient, agentID)
	if err != nil {
		return err
	}

	if isJSONOutput(cmd) {
		return outputJSON(skills)
	}

	if len(skills) == 0 {
		fmt.Println(ui.WarningStyle.Render(fmt.Sprintf("No A2A skills found on agent %q.", agentName)))
		return nil
	}

	table := ui.NewTable([]string{"NAME", "ID", "DESCRIPTION", "INPUT MODES", "OUTPUT MODES"})
	for _, skill := range skills {
		inputModes := formatModes(skill.InputModes)
		outputModes := formatModes(skill.OutputModes)
		desc := skill.Description
		if len(desc) > 40 {
			desc = desc[:37] + "..."
		}
		table.AddRow(skill.Name, skill.ID, desc, inputModes, outputModes)
	}

	fmt.Println()
	fmt.Println(ui.TitleStyle.Render(fmt.Sprintf("A2A Skills on %s", agentName)))
	fmt.Println()
	fmt.Println(table.Render())
	fmt.Println()
	fmt.Println(ui.MutedStyle.Render(fmt.Sprintf("Total: %d skills", len(skills))))
	fmt.Println()
	fmt.Println(ui.MutedStyle.Render("Use 'ei a2a-skills delete <skill-id>' to remove a skill"))
	fmt.Println()

	return nil
}

func runDeleteA2ASkill(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	skillID := args[0]

	apiClient, err := auth.GetClientFromCmd(cmd)
	if err != nil {
		return err
	}

	err = deleteA2ASkill(ctx, apiClient, skillID)
	if err != nil {
		return fmt.Errorf("failed to delete A2A skill: %w", err)
	}

	fmt.Println(ui.SuccessStyle.Render(fmt.Sprintf("Deleted A2A skill %s", skillID)))
	return nil
}

type a2aSkillInfo struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Tags        []string `json:"tags,omitempty"`
	Examples    []string `json:"examples,omitempty"`
	InputModes  []string `json:"input_modes,omitempty"`
	OutputModes []string `json:"output_modes,omitempty"`
}

func getA2ASkills(ctx context.Context, apiClient *client.Client, agentID string) ([]a2aSkillInfo, string, error) {
	// Use genqlient-generated type-safe function
	resp, err := client.GetAgentA2ASkills(ctx, apiClient.Genqlient(), agentID)
	if err != nil {
		return nil, "", fmt.Errorf("failed to query A2A skills: %w", err)
	}

	agentPtr := resp.Organization.Agent
	if agentPtr == nil {
		return nil, "", fmt.Errorf("agent not found: %s", agentID)
	}

	// Dereference pointer to interface
	agent := *agentPtr
	agentName := agent.GetName()

	// A2A skills are only available on AgentElementum type (via AgentCard)
	// Type switch to check if this is an AgentElementum with a card
	elementumAgent, ok := agent.(*client.GetAgentA2ASkillsOrganizationAgentAgentElementum)
	if !ok || elementumAgent.Card == nil {
		// Not an AgentElementum or no card - return empty skills
		return nil, agentName, nil
	}

	var skills []a2aSkillInfo
	for _, edge := range elementumAgent.Card.Skills.Edges {
		skills = append(skills, a2aSkillInfo{
			ID:          edge.Node.Id,
			Name:        edge.Node.Name,
			Description: edge.Node.Description,
			Tags:        edge.Node.Tags,
			Examples:    edge.Node.Examples,
			InputModes:  edge.Node.InputModes,
			OutputModes: edge.Node.OutputModes,
		})
	}

	return skills, agentName, nil
}

func deleteA2ASkill(ctx context.Context, apiClient *client.Client, skillID string) error {
	// Use genqlient-generated type-safe function
	_, err := client.DeleteAgentA2ASkill(ctx, apiClient.Genqlient(), skillID)
	return err
}

// formatModes formats a slice of MIME types for display
func formatModes(modes []string) string {
	if len(modes) == 0 {
		return "-"
	}
	if len(modes) == 1 {
		return modes[0]
	}
	if len(modes) <= 2 {
		return fmt.Sprintf("%s, %s", modes[0], modes[1])
	}
	return fmt.Sprintf("%s +%d more", modes[0], len(modes)-1)
}
