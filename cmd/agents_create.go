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

var agentsCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new agent",
	Long: `Create a new Elementum AI agent on an app.

Examples:
  # Create an Elementum agent with a model name
  ei agents create --app support-tickets --name "Support Agent" \
    --description "Handles support tickets" \
    --instructions "You are a helpful support agent." \
    --model "gpt-4o"

  # Create with explicit AI provider connector ID
  ei agents create --app support-tickets --name "Support Agent" \
    --description "Handles support tickets" \
    --instructions "You are a helpful support agent." \
    --ai-provider-connector-id "abc-123"

  # Create with a first message
  ei agents create --app support-tickets --name "Support Agent" \
    --description "Handles support tickets" \
    --instructions "You are a helpful support agent." \
    --model "gpt-4o" \
    --first-message "Hello! How can I help you today?"

  # Dry run
  ei agents create --app support-tickets --name "Test Agent" \
    --description "Test" --instructions "Test" --model "gpt-4o" --dry-run`,
	RunE: runAgentsCreate,
}

func init() {
	agentsCreateCmd.Flags().String("app", "", "App namespace or ID (required)")
	agentsCreateCmd.Flags().String("name", "", "Agent name (required)")
	agentsCreateCmd.Flags().String("description", "", "Agent description (required)")
	agentsCreateCmd.Flags().String("instructions", "", "Agent instructions/system prompt (required)")
	agentsCreateCmd.Flags().String("model", "", "AI model name (e.g. gpt-4o, claude-3-5-sonnet)")
	agentsCreateCmd.Flags().String("ai-provider-connector-id", "", "AI provider connector ID (alternative to --model)")
	agentsCreateCmd.Flags().String("first-message", "", "Agent's greeting message")
	agentsCreateCmd.Flags().String("type", "elementum", "Agent type: elementum, snowflake, bedrock, browser_use")
	agentsCreateCmd.Flags().Bool("dry-run", false, "Show what would be created without creating")
	_ = agentsCreateCmd.MarkFlagRequired("app")
	_ = agentsCreateCmd.MarkFlagRequired("name")
	_ = agentsCreateCmd.MarkFlagRequired("description")
	_ = agentsCreateCmd.MarkFlagRequired("instructions")
}

func runAgentsCreate(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	apiClient, err := auth.GetClientFromCmd(cmd)
	if err != nil {
		return err
	}

	appInput, _ := cmd.Flags().GetString("app")
	name, _ := cmd.Flags().GetString("name")
	description, _ := cmd.Flags().GetString("description")
	instructions, _ := cmd.Flags().GetString("instructions")
	model, _ := cmd.Flags().GetString("model")
	connectorID, _ := cmd.Flags().GetString("ai-provider-connector-id")
	firstMessage, _ := cmd.Flags().GetString("first-message")
	agentType, _ := cmd.Flags().GetString("type")
	dryRun, _ := cmd.Flags().GetBool("dry-run")

	if model == "" && connectorID == "" {
		return fmt.Errorf("either --model or --ai-provider-connector-id must be provided")
	}

	var appID string
	if looksLikeUUID(appInput) {
		appID = appInput
	} else {
		appID, _, err = resolveAspectByNamespace(ctx, apiClient, appInput)
		if err != nil {
			return fmt.Errorf("failed to resolve app %q: %w", appInput, err)
		}
	}

	if connectorID == "" {
		connectorID, err = resolveModelToConnectorID(ctx, apiClient, model)
		if err != nil {
			return err
		}
	}

	if dryRun {
		fmt.Println(ui.WarningStyle.Render("Dry run:"))
		fmt.Printf("  App:          %s\n", appInput)
		fmt.Printf("  Name:         %s\n", name)
		fmt.Printf("  Description:  %s\n", description)
		fmt.Printf("  Type:         %s\n", agentType)
		fmt.Printf("  Model:        %s\n", model)
		fmt.Printf("  Connector:    %s\n", connectorID)
		if firstMessage != "" {
			fmt.Printf("  First Msg:    %s\n", firstMessage)
		}
		fmt.Printf("  Instructions: %s\n", truncate(instructions, 80))
		return nil
	}

	if !isJSONOutput(cmd) {
		fmt.Println(ui.InfoStyle.Render(fmt.Sprintf("Creating agent %q...", name)))
	}

	var firstMessagePtr *string
	if firstMessage != "" {
		firstMessagePtr = &firstMessage
	}

	var createInput client.AgentCreateV3Input

	switch agentType {
	case "elementum":
		createInput.Elementum = &client.AgentElementumCreateInput{
			OwnerId:               appID,
			OwnerType:             "APP_ASPECT",
			Name:                  name,
			Description:           description,
			Instructions:          instructions,
			AiProviderConnectorId: connectorID,
			MaxLength:             4096,
			Temperature:           0.7,
			FirstMessage:          firstMessagePtr,
		}
	case "snowflake":
		createInput.Snowflake = &client.AgentSnowflakeCreateInput{
			OwnerId:               appID,
			OwnerType:             "APP_ASPECT",
			Name:                  name,
			Description:           description,
			Instructions:          instructions,
			AiProviderConnectorId: connectorID,
			MaxLength:             4096,
			Temperature:           0.7,
			FirstMessage:          firstMessagePtr,
		}
	case "bedrock":
		createInput.Bedrock = &client.AgentBedrockCreateInput{
			OwnerId:               appID,
			OwnerType:             "APP_ASPECT",
			Name:                  name,
			Description:           description,
			Instructions:          instructions,
			AiProviderConnectorId: connectorID,
			MaxLength:             4096,
			Temperature:           0.7,
			FirstMessage:          firstMessagePtr,
		}
	case "browser_use":
		createInput.BrowserUse = &client.AgentBrowserUseCreateInput{
			OwnerId:               appID,
			OwnerType:             "APP_ASPECT",
			Name:                  name,
			Description:           description,
			Instructions:          instructions,
			AiProviderConnectorId: connectorID,
			MaxLength:             4096,
			Temperature:           0.7,
			FirstMessage:          firstMessagePtr,
		}
	default:
		return fmt.Errorf("unsupported agent type %q. Supported: elementum, snowflake, bedrock, browser_use", agentType)
	}

	resp, err := client.CreateAgent(ctx, apiClient.Genqlient(), createInput)
	if err != nil {
		return fmt.Errorf("failed to create agent: %w", err)
	}

	created := resp.AgentCreateV3
	result := map[string]string{
		"id":   created.GetId(),
		"name": created.GetName(),
		"type": agentType,
	}

	if isJSONOutput(cmd) {
		return outputJSON(result)
	}

	fmt.Println(ui.SuccessStyle.Render("Created agent:") + fmt.Sprintf(" %s (%s)", created.GetName(), created.GetId()))
	return nil
}
