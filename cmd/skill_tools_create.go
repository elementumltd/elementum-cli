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

var skillToolsCreateCmd = &cobra.Command{
	Use:   "create <skill-id-or-name>",
	Short: "Create a tool on an agentic skill",
	Long: `Create a new tool on an agentic skill.

Tool types:
  automation       - Execute an automation (requires --automation-id)
  create-record    - Create a record in an app (requires --target-id)
  search-records   - Search records in an app (requires --target-id, --query-description)
  update-record    - Update a record in an app (requires --target-id, --handle-description)
  search-table     - Search a table (requires --search-table-id, --query-description)
  run-agent        - Run another agent (requires --target-agent-id, --worker-task-prompt)

Examples:
  # Create an automation tool
  ei skill-tools create "Ticket Triage" --type automation \
    --name "Classify Ticket" --description "Runs classification" \
    --automation-id <automation-id>

  # Create a search records tool
  ei skill-tools create <skill-id> --type search-records \
    --name "Find Tickets" --description "Search support tickets" \
    --target-id <app-id> --query-description "Search for tickets matching criteria"

  # Create a create-record tool
  ei skill-tools create <skill-id> --type create-record \
    --name "Create Ticket" --description "Creates a new ticket" \
    --target-id <app-id>

  # Create an update-record tool
  ei skill-tools create <skill-id> --type update-record \
    --name "Update Ticket" --description "Updates a ticket" \
    --target-id <app-id> --handle-description "Update the ticket record"

  # Create a run-agent tool
  ei skill-tools create <skill-id> --type run-agent \
    --name "Delegate to Specialist" --description "Runs specialist agent" \
    --target-agent-id <agent-id> --worker-task-prompt "Handle this request"`,
	Args: cobra.ExactArgs(1),
	RunE: runSkillToolsCreate,
}

func init() {
	skillToolsCreateCmd.Flags().String("type", "", "Tool type: automation, create-record, search-records, update-record, search-table, run-agent (required)")
	skillToolsCreateCmd.Flags().String("name", "", "Tool name (required)")
	skillToolsCreateCmd.Flags().String("description", "", "Tool description")
	skillToolsCreateCmd.Flags().String("start-message", "", "Message shown when tool starts executing")
	skillToolsCreateCmd.Flags().String("automation-id", "", "Automation ID (for automation type)")
	skillToolsCreateCmd.Flags().String("target-id", "", "Target app ID (for create-record, search-records, update-record, search-table)")
	skillToolsCreateCmd.Flags().String("search-table-id", "", "Search table ID (for search-table type)")
	skillToolsCreateCmd.Flags().String("target-agent-id", "", "Target agent ID (for run-agent type)")
	skillToolsCreateCmd.Flags().String("query-description", "", "Query description (for search-records, search-table)")
	skillToolsCreateCmd.Flags().String("handle-description", "", "Handle description (for update-record)")
	skillToolsCreateCmd.Flags().String("worker-task-prompt", "", "Worker task prompt (for run-agent)")
	skillToolsCreateCmd.Flags().Int("result-limit", 0, "Max results returned (for search types)")
	_ = skillToolsCreateCmd.MarkFlagRequired("type")
	_ = skillToolsCreateCmd.MarkFlagRequired("name")

	skillToolsCmd.AddCommand(skillToolsCreateCmd)
}

func runSkillToolsCreate(cmd *cobra.Command, args []string) error {
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
		if !isJSONOutput(cmd) {
			fmt.Println(ui.InfoStyle.Render(fmt.Sprintf("Found skill: %s (%s)", skill.Name, skill.ID)))
		}
	}

	toolType, _ := cmd.Flags().GetString("type")
	name, _ := cmd.Flags().GetString("name")
	description, _ := cmd.Flags().GetString("description")
	startMessage, _ := cmd.Flags().GetString("start-message")
	automationID, _ := cmd.Flags().GetString("automation-id")
	targetID, _ := cmd.Flags().GetString("target-id")
	searchTableID, _ := cmd.Flags().GetString("search-table-id")
	targetAgentID, _ := cmd.Flags().GetString("target-agent-id")
	queryDescription, _ := cmd.Flags().GetString("query-description")
	handleDescription, _ := cmd.Flags().GetString("handle-description")
	workerTaskPrompt, _ := cmd.Flags().GetString("worker-task-prompt")
	resultLimit, _ := cmd.Flags().GetInt("result-limit")

	input, err := buildSkillToolInput(toolType, name, description, startMessage, automationID, targetID, searchTableID, targetAgentID, queryDescription, handleDescription, workerTaskPrompt, resultLimit)
	if err != nil {
		return err
	}

	if !isJSONOutput(cmd) {
		fmt.Println(ui.InfoStyle.Render(fmt.Sprintf("Creating %s tool %q on skill...", toolType, name)))
	}

	var result struct {
		AgenticSkillToolCreate struct {
			ID   string `json:"id"`
			Name string `json:"name"`
			Type string `json:"type"`
		} `json:"agenticSkillToolCreate"`
	}

	err = apiClient.ExecuteInto(ctx, client.CreateAgenticSkillToolMutation, map[string]interface{}{
		"skillId": skillID,
		"input":   input,
	}, &result)
	if err != nil {
		return fmt.Errorf("failed to create skill tool: %w", err)
	}

	created := result.AgenticSkillToolCreate

	if isJSONOutput(cmd) {
		return outputJSON(map[string]string{
			"id":   created.ID,
			"name": created.Name,
			"type": created.Type,
		})
	}

	fmt.Println(ui.SuccessStyle.Render("Created skill tool:") + fmt.Sprintf(" %s (%s)", created.Name, created.ID))
	return nil
}

func buildSkillToolInput(toolType, name, description, startMessage, automationID, targetID, searchTableID, targetAgentID, queryDescription, handleDescription, workerTaskPrompt string, resultLimit int) (map[string]interface{}, error) {
	toolType = strings.ToLower(toolType)

	base := map[string]interface{}{
		"name": name,
	}
	if description != "" {
		base["description"] = description
	}
	if startMessage != "" {
		base["startMessage"] = startMessage
	}

	switch toolType {
	case "automation":
		if automationID == "" {
			return nil, fmt.Errorf("--automation-id is required for automation tool type")
		}
		base["automationId"] = automationID
		return map[string]interface{}{"automation": base}, nil

	case "create-record":
		if targetID == "" {
			return nil, fmt.Errorf("--target-id is required for create-record tool type")
		}
		base["aspectId"] = targetID
		return map[string]interface{}{"createRecord": base}, nil

	case "search-records":
		if targetID == "" {
			return nil, fmt.Errorf("--target-id is required for search-records tool type")
		}
		if queryDescription == "" {
			return nil, fmt.Errorf("--query-description is required for search-records tool type")
		}
		base["aspectId"] = targetID
		base["queryDescription"] = queryDescription
		if resultLimit > 0 {
			base["limit"] = resultLimit
		}
		return map[string]interface{}{"searchAspect": base}, nil

	case "update-record":
		if targetID == "" {
			return nil, fmt.Errorf("--target-id is required for update-record tool type")
		}
		if handleDescription == "" {
			return nil, fmt.Errorf("--handle-description is required for update-record tool type")
		}
		base["aspectId"] = targetID
		base["handleDescription"] = handleDescription
		return map[string]interface{}{"updateRecord": base}, nil

	case "search-table":
		if searchTableID == "" {
			return nil, fmt.Errorf("--search-table-id is required for search-table tool type")
		}
		if queryDescription == "" {
			return nil, fmt.Errorf("--query-description is required for search-table tool type")
		}
		base["tableId"] = searchTableID
		base["queryDescription"] = queryDescription
		if targetID != "" {
			base["aspectId"] = targetID
		}
		if resultLimit > 0 {
			base["limit"] = resultLimit
		}
		return map[string]interface{}{"searchTable": base}, nil

	case "run-agent":
		if targetAgentID == "" {
			return nil, fmt.Errorf("--target-agent-id is required for run-agent tool type")
		}
		if workerTaskPrompt == "" {
			return nil, fmt.Errorf("--worker-task-prompt is required for run-agent tool type")
		}
		base["targetAgentId"] = targetAgentID
		base["workerTaskPrompt"] = workerTaskPrompt
		return map[string]interface{}{"runAgent": base}, nil

	default:
		return nil, fmt.Errorf("unsupported tool type %q. Supported: automation, create-record, search-records, update-record, search-table, run-agent", toolType)
	}
}
