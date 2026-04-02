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
	"strings"

	"github.com/elementumltd/elementum-cli/auth"
	"github.com/elementumltd/elementum-cli/ui"
	"github.com/elementumltd/elementum-cli/internal/client"
	"github.com/spf13/cobra"
)

var skillsUpdateCmd = &cobra.Command{
	Use:   "update <skill-id-or-name>",
	Short: "Update an agentic skill",
	Long: `Update properties of an existing agentic skill.

Examples:
  ei skills update "Ticket Triage" --name "New Triage Skill"
  ei skills update "Ticket Triage" --description "Updated description"
  ei skills update "Ticket Triage" --instructions "New instructions..."
  ei skills update "Ticket Triage" --status INACTIVE`,
	Args: cobra.ExactArgs(1),
	RunE: runSkillsUpdate,
}

func init() {
	skillsUpdateCmd.Flags().String("name", "", "New skill name")
	skillsUpdateCmd.Flags().String("description", "", "New skill description")
	skillsUpdateCmd.Flags().String("instructions", "", "New skill instructions")
	skillsUpdateCmd.Flags().String("status", "", "New status: ACTIVE, DRAFT, INACTIVE")

	skillsCmd.AddCommand(skillsUpdateCmd)
}

func runSkillsUpdate(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	apiClient, err := auth.GetClientFromCmd(cmd)
	if err != nil {
		return err
	}

	skillIDOrName := args[0]
	skillID := skillIDOrName
	if !isUUID(skillIDOrName) {
		skill, err := findSkillByName(ctx, apiClient, skillIDOrName)
		if err != nil {
			return fmt.Errorf("failed to find skill %q: %w", skillIDOrName, err)
		}
		skillID = skill.ID
	}

	name, _ := cmd.Flags().GetString("name")
	description, _ := cmd.Flags().GetString("description")
	instructions, _ := cmd.Flags().GetString("instructions")
	status, _ := cmd.Flags().GetString("status")

	if name == "" && description == "" && instructions == "" && status == "" {
		return fmt.Errorf("at least one update flag must be provided (--name, --description, --instructions, --status)")
	}

	updateInput := client.AgenticSkillUpdateInput{}
	if name != "" {
		updateInput.Name = &name
	}
	if description != "" {
		updateInput.Description = &description
	}
	if instructions != "" {
		updateInput.Instructions = &instructions
	}
	if status != "" {
		s := client.AgenticSkillStatus(strings.ToUpper(status))
		switch s {
		case client.AgenticSkillStatusActive, client.AgenticSkillStatusDraft, client.AgenticSkillStatusInactive:
			updateInput.Status = &s
		default:
			return fmt.Errorf("invalid status %q. Supported: ACTIVE, DRAFT, INACTIVE", status)
		}
	}

	if !isJSONOutput(cmd) {
		fmt.Println(ui.InfoStyle.Render(fmt.Sprintf("Updating skill %s...", args[0])))
	}

	resp, err := client.UpdateAgenticSkill(ctx, apiClient.Genqlient(), skillID, updateInput)
	if err != nil {
		return fmt.Errorf("failed to update skill: %w", err)
	}

	updated := resp.AgenticSkillUpdate
	result := map[string]string{
		"id":     updated.Id,
		"name":   updated.Name,
		"status": string(updated.Status),
	}

	if isJSONOutput(cmd) {
		return outputJSON(result)
	}

	fmt.Println(ui.SuccessStyle.Render("Updated skill:") + fmt.Sprintf(" %s (%s)", updated.Name, updated.Id))
	return nil
}
