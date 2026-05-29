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

var agentsDeleteCmd = &cobra.Command{
	Use:   "delete <namespace> <agent-name>",
	Short: "Delete an agent",
	Long: `Delete an Elementum agent by name within an app.

WARNING: This permanently deletes the agent and all its tools.

Examples:
  ei agents delete support-tickets "Support Agent"
  ei agents delete support-tickets "Support Agent" --force
  ei agents delete clm ticket-classifier --force`,
	Args: cobra.ExactArgs(2),
	RunE: runAgentsDelete,
}

func init() {
	agentsDeleteCmd.Flags().BoolP("force", "f", false, "Skip confirmation prompt")
}

func runAgentsDelete(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	namespace := args[0]
	agentNameArg := args[1]

	apiClient, err := auth.GetClientFromCmd(cmd)
	if err != nil {
		return err
	}

	// Resolve namespace to aspect ID
	aspectID, _, err := resolveAspectByNamespace(ctx, apiClient, namespace)
	if err != nil {
		return fmt.Errorf("failed to resolve app %q: %w", namespace, err)
	}

	// Find agent by name in the app
	agentID, err := findAgentByNameInApp(ctx, apiClient, aspectID, agentNameArg)
	if err != nil {
		return fmt.Errorf("failed to find agent %q in app %q: %w", agentNameArg, namespace, err)
	}

	force, _ := cmd.Flags().GetBool("force")

	// Get agent name for display
	agentName := agentNameArg
	resp, err := client.GetAgentById(ctx, apiClient.Genqlient(), agentID)
	if err == nil && resp.Organization.Agent != nil {
		agent := *resp.Organization.Agent
		agentName = agent.GetName()
	}

	if !force {
		confirmed, err := ui.Confirm(
			fmt.Sprintf("Delete agent %q?", agentName),
			fmt.Sprintf("ID: %s\nThis will permanently delete the agent and all its tools.", agentID),
		)
		if err != nil {
			return err
		}
		if !confirmed {
			fmt.Println(ui.MutedStyle.Render("Cancelled."))
			return nil
		}
	}

	if !isJSONOutput(cmd) {
		fmt.Println(ui.InfoStyle.Render(fmt.Sprintf("Deleting agent %q...", agentName)))
	}

	_, err = client.DeleteAgent(ctx, apiClient.Genqlient(), agentID)
	if err != nil {
		return fmt.Errorf("failed to delete agent: %w", err)
	}

	if isJSONOutput(cmd) {
		return outputJSON(map[string]string{
			"id":      agentID,
			"name":    agentName,
			"deleted": "true",
		})
	}

	fmt.Println(ui.SuccessStyle.Render("Deleted agent:") + fmt.Sprintf(" %s (%s)", agentName, agentID))
	return nil
}
