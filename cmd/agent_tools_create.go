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
	"github.com/spf13/cobra"
)

var agentToolsCreateCmd = &cobra.Command{
	Use:   "create <agent-name-or-id>",
	Short: "Create a tool on an agent",
	Long: `Create a new tool on an agent.

Tool types:
  run-automation   - Execute an automation (requires --automation-id)
  create-record    - Create a record in an app (requires --target-id)
  search-records   - Search records in an app (requires --target-id, --query-description)
  update-record    - Update a record (requires --target-id, --handle-description)
  relate-record    - Relate two records (requires --target-id, --related-target-id)
  search-table     - Search a table (requires --search-table-id, --query-description)
  run-agent        - Run another agent (requires --target-agent-id, --worker-task-prompt)
  mcp              - MCP tool (requires --mcp-tool-name, --server-url)

Examples:
  # Create an automation tool
  ei agent-tools create "Support Agent" --type run-automation \
    --name "Classify Ticket" --description "Runs classification" \
    --automation-id <automation-id>

  # Create a search records tool
  ei agent-tools create "Support Agent" --type search-records \
    --name "Find Tickets" --description "Search support tickets" \
    --target-id <app-id> --query-description "Search for tickets matching criteria"

  # Create a create-record tool
  ei agent-tools create "Support Agent" --type create-record \
    --name "Create Ticket" --description "Creates a new ticket" \
    --target-id <app-id>

  # Create an update-record tool
  ei agent-tools create "Support Agent" --type update-record \
    --name "Update Ticket" --description "Updates ticket fields" \
    --target-id <app-id> --handle-description "Update the ticket"

  # Create a relate-record tool
  ei agent-tools create "Support Agent" --type relate-record \
    --name "Link KB Article" --description "Links a KB article to ticket" \
    --target-id <app-id> --related-target-id <element-id>

  # Create an MCP tool
  ei agent-tools create "Support Agent" --type mcp \
    --name "External API" --description "Calls external service" \
    --mcp-tool-name "api_call" --server-url "https://example.com/mcp"

  # Create a run-agent tool
  ei agent-tools create "Support Agent" --type run-agent \
    --name "Delegate" --description "Delegates to specialist" \
    --target-agent-id <agent-id> --worker-task-prompt "Handle this request"`,
	Args: cobra.ExactArgs(1),
	RunE: runAgentToolsCreate,
}

func init() {
	agentToolsCreateCmd.Flags().String("type", "", "Tool type: run-automation, create-record, search-records, update-record, relate-record, search-table, run-agent, mcp (required)")
	agentToolsCreateCmd.Flags().String("name", "", "Tool name (required)")
	agentToolsCreateCmd.Flags().String("description", "", "Tool description")
	agentToolsCreateCmd.Flags().String("start-message", "", "Message shown when tool starts")
	agentToolsCreateCmd.Flags().String("automation-id", "", "Automation ID (for run-automation)")
	agentToolsCreateCmd.Flags().String("target-id", "", "Target app ID (for create-record, search-records, update-record, relate-record)")
	agentToolsCreateCmd.Flags().String("related-target-id", "", "Related app/element ID (for relate-record)")
	agentToolsCreateCmd.Flags().String("search-table-id", "", "Search table ID (for search-table)")
	agentToolsCreateCmd.Flags().String("target-agent-id", "", "Target agent ID (for run-agent)")
	agentToolsCreateCmd.Flags().String("query-description", "", "Query description (for search-records, search-table)")
	agentToolsCreateCmd.Flags().String("handle-description", "", "Handle description (for update-record)")
	agentToolsCreateCmd.Flags().String("worker-task-prompt", "", "Worker task prompt (for run-agent)")
	agentToolsCreateCmd.Flags().String("mcp-tool-name", "", "MCP tool name (for mcp)")
	agentToolsCreateCmd.Flags().String("server-url", "", "Server URL (for mcp)")
	agentToolsCreateCmd.Flags().Int("result-limit", 0, "Max results returned (for search types)")
	_ = agentToolsCreateCmd.MarkFlagRequired("type")
	_ = agentToolsCreateCmd.MarkFlagRequired("name")

	agentToolsCmd.AddCommand(agentToolsCreateCmd)
}

func runAgentToolsCreate(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	apiClient, err := auth.GetClientFromCmd(cmd)
	if err != nil {
		return err
	}

	agentID, err := resolveAgentID(ctx, cmd, apiClient, args[0])
	if err != nil {
		return err
	}

	toolType, _ := cmd.Flags().GetString("type")
	name, _ := cmd.Flags().GetString("name")
	description, _ := cmd.Flags().GetString("description")
	startMessage, _ := cmd.Flags().GetString("start-message")
	automationID, _ := cmd.Flags().GetString("automation-id")
	targetID, _ := cmd.Flags().GetString("target-id")
	relatedTargetID, _ := cmd.Flags().GetString("related-target-id")
	searchTableID, _ := cmd.Flags().GetString("search-table-id")
	targetAgentID, _ := cmd.Flags().GetString("target-agent-id")
	queryDescription, _ := cmd.Flags().GetString("query-description")
	handleDescription, _ := cmd.Flags().GetString("handle-description")
	workerTaskPrompt, _ := cmd.Flags().GetString("worker-task-prompt")
	mcpToolName, _ := cmd.Flags().GetString("mcp-tool-name")
	serverURL, _ := cmd.Flags().GetString("server-url")
	resultLimit, _ := cmd.Flags().GetInt("result-limit")

	input, err := buildAgentToolInput(toolType, name, description, startMessage, automationID, targetID, relatedTargetID, searchTableID, targetAgentID, queryDescription, handleDescription, workerTaskPrompt, mcpToolName, serverURL, resultLimit)
	if err != nil {
		return err
	}

	if !isJSONOutput(cmd) {
		fmt.Println(ui.InfoStyle.Render(fmt.Sprintf("Creating %s tool %q on agent...", toolType, name)))
	}

	var result struct {
		AgentToolCreate struct {
			ID          string `json:"id"`
			Name        string `json:"name"`
			Description string `json:"description"`
		} `json:"agentToolCreate"`
	}

	mutation := `
		mutation CreateAgentTool($agentId: ID!, $input: AgentToolCreateInput!) {
			agentToolCreate(agentId: $agentId, input: $input) {
				id
				name
				description
			}
		}
	`

	err = apiClient.ExecuteInto(ctx, mutation, map[string]interface{}{
		"agentId": agentID,
		"input":   input,
	}, &result)
	if err != nil {
		return fmt.Errorf("failed to create agent tool: %w", err)
	}

	created := result.AgentToolCreate

	if isJSONOutput(cmd) {
		return outputJSON(map[string]string{
			"id":   created.ID,
			"name": created.Name,
			"type": toolType,
		})
	}

	fmt.Println(ui.SuccessStyle.Render("Created agent tool:") + fmt.Sprintf(" %s (%s)", created.Name, created.ID))
	return nil
}

func buildAgentToolInput(toolType, name, description, startMessage, automationID, targetID, relatedTargetID, searchTableID, targetAgentID, queryDescription, handleDescription, workerTaskPrompt, mcpToolName, serverURL string, resultLimit int) (map[string]interface{}, error) {
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
	case "run-automation":
		if automationID == "" {
			return nil, fmt.Errorf("--automation-id is required for run-automation tool type")
		}
		base["automationId"] = automationID
		return map[string]interface{}{"executeWorkflow": base}, nil

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

	case "relate-record":
		if targetID == "" {
			return nil, fmt.Errorf("--target-id is required for relate-record tool type")
		}
		if relatedTargetID == "" {
			return nil, fmt.Errorf("--related-target-id is required for relate-record tool type")
		}
		base["aspectId"] = targetID
		base["relatedAspectId"] = relatedTargetID
		return map[string]interface{}{"relateRecord": base}, nil

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

	case "mcp":
		if mcpToolName == "" {
			return nil, fmt.Errorf("--mcp-tool-name is required for mcp tool type")
		}
		if serverURL == "" {
			return nil, fmt.Errorf("--server-url is required for mcp tool type")
		}
		base["mcpToolName"] = mcpToolName
		base["serverUrl"] = serverURL
		return map[string]interface{}{"mcp": base}, nil

	default:
		return nil, fmt.Errorf("unsupported tool type %q. Supported: run-automation, create-record, search-records, update-record, relate-record, search-table, run-agent, mcp", toolType)
	}
}
