// Copyright (c) ElementumAI, Inc.
// SPDX-License-Identifier: MPL-2.0

package cmd

import (
	"context"
	"fmt"

	"github.com/elementumltd/elementum-cli/auth"
	"github.com/elementumltd/elementum-cli/ui"
	"github.com/spf13/cobra"
)

var agentToolsUpdateCmd = &cobra.Command{
	Use:   "update <tool-id>",
	Short: "Update an agent tool",
	Long: `Update properties of an existing agent tool.

Provide the original tool type along with the fields you want to change.

Examples:
  ei agent-tools update <tool-id> --type run-automation --name "New Name"
  ei agent-tools update <tool-id> --type search-records --query-description "Updated query"
  ei agent-tools update <tool-id> --type run-automation --description "Updated desc"`,
	Args: cobra.ExactArgs(1),
	RunE: runAgentToolsUpdate,
}

func init() {
	agentToolsUpdateCmd.Flags().String("type", "", "Tool type (required to determine update shape)")
	agentToolsUpdateCmd.Flags().String("name", "", "New tool name")
	agentToolsUpdateCmd.Flags().String("description", "", "New tool description")
	agentToolsUpdateCmd.Flags().String("start-message", "", "New start message")
	agentToolsUpdateCmd.Flags().String("query-description", "", "New query description (search types)")
	agentToolsUpdateCmd.Flags().String("handle-description", "", "New handle description (update-record)")
	agentToolsUpdateCmd.Flags().String("worker-task-prompt", "", "New worker task prompt (run-agent)")
	_ = agentToolsUpdateCmd.MarkFlagRequired("type")

	agentToolsCmd.AddCommand(agentToolsUpdateCmd)
}

func runAgentToolsUpdate(cmd *cobra.Command, args []string) error {
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

	input, err := buildAgentToolUpdateInput(toolType, name, description, startMessage, queryDescription, handleDescription, workerTaskPrompt)
	if err != nil {
		return err
	}

	if !isJSONOutput(cmd) {
		fmt.Println(ui.InfoStyle.Render(fmt.Sprintf("Updating agent tool %s...", toolID)))
	}

	var result struct {
		AgentToolUpdate struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"agentToolUpdate"`
	}

	mutation := `
		mutation UpdateAgentTool($id: ID!, $input: AgentToolUpdateInput!) {
			agentToolUpdate(id: $id, input: $input) {
				id
				name
				description
			}
		}
	`

	err = apiClient.ExecuteInto(ctx, mutation, map[string]interface{}{
		"id":    toolID,
		"input": input,
	}, &result)
	if err != nil {
		return fmt.Errorf("failed to update agent tool: %w", err)
	}

	updated := result.AgentToolUpdate

	if isJSONOutput(cmd) {
		return outputJSON(map[string]string{
			"id":   updated.ID,
			"name": updated.Name,
		})
	}

	fmt.Println(ui.SuccessStyle.Render("Updated agent tool:") + fmt.Sprintf(" %s (%s)", updated.Name, updated.ID))
	return nil
}

func buildAgentToolUpdateInput(toolType, name, description, startMessage, queryDescription, handleDescription, workerTaskPrompt string) (map[string]interface{}, error) {
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
	case "run-automation":
		return map[string]interface{}{"executeWorkflow": base}, nil
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
	case "relate-record":
		return map[string]interface{}{"relateRecord": base}, nil
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
	case "mcp":
		return map[string]interface{}{"mcp": base}, nil
	default:
		return nil, fmt.Errorf("unsupported tool type %q. Supported: run-automation, create-record, search-records, update-record, relate-record, search-table, run-agent, mcp", toolType)
	}
}
