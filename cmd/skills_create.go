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

var skillsCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create an agentic skill",
	Long: `Create a new agentic skill.

Skills define reusable capabilities for agents. They have instructions that guide
agent behavior and can have tools attached for executing actions.

The --owner-id and --owner-type specify what the skill belongs to:
  - owner-type APP_ASPECT: skill belongs to an app (pass app ID as owner-id)
  - owner-type ORGANIZATION: skill belongs to the org (pass org ID)

Examples:
  # Create a skill on an app
  ei skills create --name "Ticket Triage" \
    --description "Triages incoming support tickets" \
    --instructions "When a ticket comes in, classify it by urgency..." \
    --owner-id <app-id> --owner-type APP_ASPECT

  # Create with a specific status
  ei skills create --name "Draft Skill" \
    --description "Work in progress" \
    --instructions "..." \
    --owner-id <app-id> --owner-type APP_ASPECT \
    --status DRAFT

  # Create using app namespace instead of ID
  ei skills create --name "Ticket Triage" \
    --description "Triages incoming support tickets" \
    --instructions "Classify tickets by urgency..." \
    --app support-tickets`,
	RunE: runSkillsCreate,
}

func init() {
	skillsCreateCmd.Flags().String("name", "", "Skill name (required)")
	skillsCreateCmd.Flags().String("description", "", "Skill description (required)")
	skillsCreateCmd.Flags().String("instructions", "", "Skill instructions (required)")
	skillsCreateCmd.Flags().String("owner-id", "", "Owner ID (app ID or org ID)")
	skillsCreateCmd.Flags().String("owner-type", "APP_ASPECT", "Owner type: APP_ASPECT or ORGANIZATION")
	skillsCreateCmd.Flags().String("app", "", "App namespace (shorthand for --owner-id with --owner-type APP_ASPECT)")
	skillsCreateCmd.Flags().String("status", "ACTIVE", "Skill status: ACTIVE, DRAFT, INACTIVE")
	skillsCreateCmd.Flags().Bool("dry-run", false, "Show what would be created without creating")
	_ = skillsCreateCmd.MarkFlagRequired("name")
	_ = skillsCreateCmd.MarkFlagRequired("description")
	_ = skillsCreateCmd.MarkFlagRequired("instructions")

	skillsCmd.AddCommand(skillsCreateCmd)
}

func runSkillsCreate(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	apiClient, err := auth.GetClientFromCmd(cmd)
	if err != nil {
		return err
	}

	name, _ := cmd.Flags().GetString("name")
	description, _ := cmd.Flags().GetString("description")
	instructions, _ := cmd.Flags().GetString("instructions")
	ownerID, _ := cmd.Flags().GetString("owner-id")
	ownerType, _ := cmd.Flags().GetString("owner-type")
	appNamespace, _ := cmd.Flags().GetString("app")
	status, _ := cmd.Flags().GetString("status")
	dryRun, _ := cmd.Flags().GetBool("dry-run")

	if appNamespace != "" {
		if looksLikeUUID(appNamespace) {
			ownerID = appNamespace
		} else {
			ownerID, _, err = resolveAspectByNamespace(ctx, apiClient, appNamespace)
			if err != nil {
				return fmt.Errorf("failed to resolve app %q: %w", appNamespace, err)
			}
		}
		ownerType = "APP_ASPECT"
	}

	if ownerID == "" {
		return fmt.Errorf("either --owner-id or --app must be provided")
	}

	statusEnum := client.AgenticSkillStatus(strings.ToUpper(status))
	switch statusEnum {
	case client.AgenticSkillStatusActive, client.AgenticSkillStatusDraft, client.AgenticSkillStatusInactive:
	default:
		return fmt.Errorf("invalid status %q. Supported: ACTIVE, DRAFT, INACTIVE", status)
	}

	if dryRun {
		fmt.Println(ui.WarningStyle.Render("Dry run:"))
		fmt.Printf("  Name:         %s\n", name)
		fmt.Printf("  Description:  %s\n", description)
		fmt.Printf("  Owner:        %s (%s)\n", ownerID, ownerType)
		fmt.Printf("  Status:       %s\n", status)
		fmt.Printf("  Instructions: %s\n", truncate(instructions, 80))
		return nil
	}

	if !isJSONOutput(cmd) {
		fmt.Println(ui.InfoStyle.Render(fmt.Sprintf("Creating skill %q...", name)))
	}

	input := client.AgenticSkillCreateInput{
		Name:         name,
		Description:  description,
		Instructions: instructions,
		OwnerId:      ownerID,
		OwnerType:    ownerType,
		Status:       statusEnum,
		Type:         client.AgenticSkillTypeCustom,
	}

	resp, err := client.CreateAgenticSkill(ctx, apiClient.Genqlient(), input)
	if err != nil {
		return fmt.Errorf("failed to create skill: %w", err)
	}

	created := resp.AgenticSkillCreate
	result := map[string]string{
		"id":     created.Id,
		"name":   created.Name,
		"status": string(created.Status),
	}

	if isJSONOutput(cmd) {
		return outputJSON(result)
	}

	fmt.Println(ui.SuccessStyle.Render("Created skill:") + fmt.Sprintf(" %s (%s)", created.Name, created.Id))
	return nil
}
