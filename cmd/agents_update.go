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

var agentsUpdateCmd = &cobra.Command{
	Use:   "update <agent-name-or-id>",
	Short: "Update an agent",
	Long: `Update properties of an existing Elementum agent.

Examples:
  ei agents update "Support Agent" --name "New Support Agent"
  ei agents update "Support Agent" --description "Updated description"
  ei agents update "Support Agent" --instructions "New instructions..."
  ei agents update "Support Agent" --model "gpt-4o-mini"
  ei agents update "Support Agent" --first-message "Hi! What can I do for you?"`,
	Args: cobra.ExactArgs(1),
	RunE: runAgentsUpdate,
}

func init() {
	agentsUpdateCmd.Flags().String("name", "", "New agent name")
	agentsUpdateCmd.Flags().String("description", "", "New agent description")
	agentsUpdateCmd.Flags().String("instructions", "", "New agent instructions")
	agentsUpdateCmd.Flags().String("model", "", "New AI model name")
	agentsUpdateCmd.Flags().String("ai-provider-connector-id", "", "New AI provider connector ID")
	agentsUpdateCmd.Flags().String("first-message", "", "New first message")
}

func runAgentsUpdate(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	apiClient, err := auth.GetClientFromCmd(cmd)
	if err != nil {
		return err
	}

	agentID, err := resolveAgentID(ctx, cmd, apiClient, args[0])
	if err != nil {
		return err
	}

	name, _ := cmd.Flags().GetString("name")
	description, _ := cmd.Flags().GetString("description")
	instructions, _ := cmd.Flags().GetString("instructions")
	model, _ := cmd.Flags().GetString("model")
	connectorID, _ := cmd.Flags().GetString("ai-provider-connector-id")
	firstMessage, _ := cmd.Flags().GetString("first-message")

	if name == "" && description == "" && instructions == "" && model == "" && connectorID == "" && firstMessage == "" {
		return fmt.Errorf("at least one update flag must be provided (--name, --description, --instructions, --model, --ai-provider-connector-id, --first-message)")
	}

	if model != "" && connectorID == "" {
		connectorID, err = resolveModelToConnectorID(ctx, apiClient, model)
		if err != nil {
			return err
		}
	}

	// We update using the elementum variant by default - the API handles type dispatch
	updateInput := client.AgentUpdateV2Input{
		Elementum: &client.AgentElementumUpdateInput{},
	}

	elem := updateInput.Elementum
	if name != "" {
		elem.Name = &name
	}
	if description != "" {
		elem.Description = &description
	}
	if instructions != "" {
		elem.Instructions = &instructions
	}
	if connectorID != "" {
		elem.AiProviderConnectorId = &connectorID
	}
	if firstMessage != "" {
		elem.FirstMessage = &firstMessage
	}

	if !isJSONOutput(cmd) {
		fmt.Println(ui.InfoStyle.Render(fmt.Sprintf("Updating agent %s...", args[0])))
	}

	resp, err := client.UpdateAgent(ctx, apiClient.Genqlient(), agentID, updateInput)
	if err != nil {
		return fmt.Errorf("failed to update agent: %w", err)
	}

	updated := resp.AgentUpdateV2
	result := map[string]string{
		"id":   updated.GetId(),
		"name": updated.GetName(),
	}

	if isJSONOutput(cmd) {
		return outputJSON(result)
	}

	fmt.Println(ui.SuccessStyle.Render("Updated agent:") + fmt.Sprintf(" %s (%s)", updated.GetName(), updated.GetId()))
	return nil
}
