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

var skillToolsUpdateCmd = &cobra.Command{
	Use:   "update <tool-id>",
	Short: "Update an agentic skill tool",
	Long: `Update properties of an existing agentic skill tool.

The update uses the same type-specific input as create. Provide the original tool type
along with the fields you want to change.

Examples:
  ei skill-tools update <tool-id> --type automation --name "New Name" --description "New desc"
  ei skill-tools update <tool-id> --type search-records --query-description "Updated query"
  ei skill-tools update <tool-id> --type automation --start-message "Processing..."`,
	Args: cobra.ExactArgs(1),
	RunE: runSkillToolsUpdate,
}

func init() {
	skillToolsUpdateCmd.Flags().String("type", "", "Tool type (required to determine update shape)")
	skillToolsUpdateCmd.Flags().String("name", "", "New tool name")
	skillToolsUpdateCmd.Flags().String("description", "", "New tool description")
	skillToolsUpdateCmd.Flags().String("start-message", "", "New start message")
	skillToolsUpdateCmd.Flags().String("query-description", "", "New query description (for search types)")
	skillToolsUpdateCmd.Flags().String("handle-description", "", "New handle description (for update-record)")
	skillToolsUpdateCmd.Flags().String("worker-task-prompt", "", "New worker task prompt (for run-agent)")
	_ = skillToolsUpdateCmd.MarkFlagRequired("type")

	skillToolsCmd.AddCommand(skillToolsUpdateCmd)
}

func runSkillToolsUpdate(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	toolID := args[0]

	apiClient, err := auth.GetClientFromCmd(cmd)
	if err != nil {
		return err
	}

	toolType, _ := cmd.Flags().GetString("type")
	name, _ := cmd.Flags().GetString("name")
	description, _ := cmd.Flags().GetString("description")
	startMessage, _ := cmd.Flags().GetString("start-message")
	queryDescription, _ := cmd.Flags().GetString("query-description")
	handleDescription, _ := cmd.Flags().GetString("handle-description")
	workerTaskPrompt, _ := cmd.Flags().GetString("worker-task-prompt")

	if name == "" && description == "" && startMessage == "" && queryDescription == "" && handleDescription == "" && workerTaskPrompt == "" {
		return fmt.Errorf("at least one update flag must be provided")
	}

	input, err := buildSkillToolUpdateInput(toolType, name, description, startMessage, queryDescription, handleDescription, workerTaskPrompt)
	if err != nil {
		return err
	}

	if !isJSONOutput(cmd) {
		fmt.Println(ui.InfoStyle.Render(fmt.Sprintf("Updating skill tool %s...", toolID)))
	}

	var result struct {
		AgenticSkillToolUpdate struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"agenticSkillToolUpdate"`
	}

	err = apiClient.ExecuteInto(ctx, client.UpdateAgenticSkillToolMutation, map[string]interface{}{
		"id":    toolID,
		"input": input,
	}, &result)
	if err != nil {
		return fmt.Errorf("failed to update skill tool: %w", err)
	}

	updated := result.AgenticSkillToolUpdate

	if isJSONOutput(cmd) {
		return outputJSON(map[string]string{
			"id":   updated.ID,
			"name": updated.Name,
		})
	}

	fmt.Println(ui.SuccessStyle.Render("Updated skill tool:") + fmt.Sprintf(" %s (%s)", updated.Name, updated.ID))
	return nil
}

func buildSkillToolUpdateInput(toolType, name, description, startMessage, queryDescription, handleDescription, workerTaskPrompt string) (map[string]interface{}, error) {
	base := map[string]interface{}{}
	if name != "" {
		base["name"] = name
	}
	if description != "" {
		base["description"] = description
	}
	if startMessage != "" {
		base["startMessage"] = startMessage
	}

	switch toolType {
	case "automation":
		return map[string]interface{}{"automation": base}, nil
	case "create-record":
		return map[string]interface{}{"createRecord": base}, nil
	case "search-records":
		if queryDescription != "" {
			base["queryDescription"] = queryDescription
		}
		return map[string]interface{}{"searchAspect": base}, nil
	case "update-record":
		if handleDescription != "" {
			base["handleDescription"] = handleDescription
		}
		return map[string]interface{}{"updateRecord": base}, nil
	case "search-table":
		if queryDescription != "" {
			base["queryDescription"] = queryDescription
		}
		return map[string]interface{}{"searchTable": base}, nil
	case "run-agent":
		if workerTaskPrompt != "" {
			base["workerTaskPrompt"] = workerTaskPrompt
		}
		return map[string]interface{}{"runAgent": base}, nil
	default:
		return nil, fmt.Errorf("unsupported tool type %q. Supported: automation, create-record, search-records, update-record, search-table, run-agent", toolType)
	}
}
