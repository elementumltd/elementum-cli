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

var agentsParentCmd = &cobra.Command{
	Use:   "agents",
	Short: "Manage agents",
	Long:  "Commands for listing, creating, showing, updating, and deleting Elementum agents.",
}

var agentsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List agents",
	Long: `List agents in an app or across the entire organization.

Examples:
  ei agents list --app clm             # List agents in the CLM app
  ei agents list                       # List all agents in the organization`,
	RunE: runAgentsListCmd,
}

func init() {
	agentsListCmd.Flags().String("app", "", "App namespace to filter agents")
	agentsParentCmd.AddCommand(agentsListCmd)
	agentsParentCmd.AddCommand(agentsCreateCmd)
	agentsParentCmd.AddCommand(agentsShowCmd)
	agentsParentCmd.AddCommand(agentsUpdateCmd)
	agentsParentCmd.AddCommand(agentsDeleteCmd)
}

// GetAgentsCmd returns the agents command for registration
func GetAgentsCmd() *cobra.Command {
	return agentsParentCmd
}

func runAgentsListCmd(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	apiClient, err := auth.GetClientFromCmd(cmd)
	if err != nil {
		return err
	}

	appNamespace, _ := cmd.Flags().GetString("app")

	if appNamespace != "" {
		return listAgentsInApp(ctx, cmd, apiClient, appNamespace)
	}

	return listAllAgents(ctx, cmd, apiClient)
}

func listAgentsInApp(ctx context.Context, cmd *cobra.Command, apiClient *client.Client, appNamespace string) error {
	if !isJSONOutput(cmd) {
		fmt.Println(ui.InfoStyle.Render(fmt.Sprintf("Loading agents from app %q...", appNamespace)))
	}

	aspectID, _, err := resolveAspectByNamespace(ctx, apiClient, appNamespace)
	if err != nil {
		return fmt.Errorf("failed to find app %q: %w", appNamespace, err)
	}

	resp, err := client.ListAppAgents(ctx, apiClient.Genqlient(), aspectID)
	if err != nil {
		return fmt.Errorf("failed to list agents: %w", err)
	}

	if resp.Organization.Aspect == nil {
		return fmt.Errorf("app not found")
	}

	aspect := *resp.Organization.Aspect
	appAspect, ok := aspect.(*client.ListAppAgentsOrganizationAspectAspectApp)
	if !ok {
		return fmt.Errorf("aspect is not an app")
	}

	agents := appAspect.AgentsV2.Edges

	if isJSONOutput(cmd) {
		return outputJSON(agents)
	}

	if len(agents) == 0 {
		fmt.Println()
		fmt.Println(ui.WarningStyle.Render(fmt.Sprintf("No agents found in app %q.", appNamespace)))
		fmt.Println()
		return nil
	}

	table := ui.NewTable([]string{"NAME", "TYPE", "DESCRIPTION", "ID"})

	for _, edge := range agents {
		agent := edge.Node
		desc := agent.GetDescription()
		if len(desc) > 50 {
			desc = desc[:47] + "..."
		}
		agentType := ""
		if tn := agent.GetTypename(); tn != nil {
			agentType = strings.TrimPrefix(*tn, "Agent")
		}
		table.AddRow(
			agent.GetName(),
			agentType,
			desc,
			agent.GetId(),
		)
	}

	fmt.Println()
	fmt.Println(ui.TitleStyle.Render(fmt.Sprintf("Agents in %s", appNamespace)))
	fmt.Println()
	fmt.Println(table.Render())
	fmt.Println()
	fmt.Println(ui.MutedStyle.Render(fmt.Sprintf("Total: %d agents", len(agents))))
	fmt.Println()

	return nil
}

type agentListItem struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Type     string `json:"__typename"`
	AppName  string `json:"app_name,omitempty"`
	AppID    string `json:"app_id,omitempty"`
	Model    string `json:"model,omitempty"`
	Provider string `json:"provider,omitempty"`
}

func listAllAgents(ctx context.Context, cmd *cobra.Command, apiClient *client.Client) error {
	if !isJSONOutput(cmd) {
		fmt.Println(ui.InfoStyle.Render("Loading all agents..."))
	}

	var result struct {
		Organization struct {
			Agents struct {
				Edges []struct {
					Node struct {
						ID          string  `json:"id"`
						Name        string  `json:"name"`
						Description string  `json:"description"`
						Typename    *string `json:"__typename"`
						App         *struct {
							ID   string `json:"id"`
							Name string `json:"name"`
						} `json:"app"`
						AiProviderConnector *struct {
							ID    string `json:"id"`
							Model struct {
								Name string `json:"name"`
							} `json:"model"`
							Provider struct {
								Name string `json:"name"`
							} `json:"provider"`
						} `json:"aiProviderConnector"`
					} `json:"node"`
				} `json:"edges"`
			} `json:"agents"`
		} `json:"organization"`
	}

	query := `
		query ListAllAgents {
			organization {
				agents {
					edges {
						node {
							id
							name
							description
							__typename
							... on AgentElementum {
								app { id name }
								aiProviderConnector { id model { name } provider { name } }
							}
							... on AgentSnowflake {
								app { id name }
								aiProviderConnector { id model { name } provider { name } }
							}
							... on AgentBedrock {
								app { id name }
								aiProviderConnector { id model { name } provider { name } }
							}
							... on AgentBrowserUse {
								app { id name }
								aiProviderConnector { id model { name } provider { name } }
							}
						}
					}
				}
			}
		}
	`

	err := apiClient.ExecuteInto(ctx, query, nil, &result)
	if err != nil {
		return fmt.Errorf("failed to list agents: %w", err)
	}

	agents := result.Organization.Agents.Edges

	if isJSONOutput(cmd) {
		return outputJSON(agents)
	}

	if len(agents) == 0 {
		fmt.Println()
		fmt.Println(ui.WarningStyle.Render("No agents found."))
		fmt.Println()
		return nil
	}

	table := ui.NewTable([]string{"NAME", "TYPE", "APP", "MODEL", "ID"})

	for _, edge := range agents {
		n := edge.Node
		agentType := ""
		if n.Typename != nil {
			agentType = strings.TrimPrefix(*n.Typename, "Agent")
		}
		appName := ""
		model := ""
		if n.App != nil {
			appName = n.App.Name
		}
		if n.AiProviderConnector != nil {
			model = n.AiProviderConnector.Model.Name
		}

		table.AddRow(n.Name, agentType, appName, model, n.ID)
	}

	fmt.Println()
	fmt.Println(ui.TitleStyle.Render("All Agents"))
	fmt.Println()
	fmt.Println(table.Render())
	fmt.Println()
	fmt.Println(ui.MutedStyle.Render(fmt.Sprintf("Total: %d agents", len(agents))))
	fmt.Println()

	return nil
}

// resolveAgentID resolves an agent argument to a UUID.
func resolveAgentID(ctx context.Context, cmd *cobra.Command, apiClient *client.Client, nameOrID string) (string, error) {
	if looksLikeUUID(nameOrID) {
		return nameOrID, nil
	}
	return resolveAgentByName(ctx, cmd, apiClient, nameOrID)
}

// resolveAgentByName looks up an agent by name across the organization (case-insensitive).
func resolveAgentByName(ctx context.Context, cmd *cobra.Command, apiClient *client.Client, agentName string) (string, error) {
	resp, err := client.GetAgents(ctx, apiClient.Genqlient(), nil)
	if err != nil {
		return "", fmt.Errorf("failed to list agents: %w", err)
	}

	var matches []struct{ id, name string }
	for _, edge := range resp.Organization.Agents.Edges {
		agent := edge.Node
		if strings.EqualFold(agent.GetName(), agentName) {
			matches = append(matches, struct{ id, name string }{agent.GetId(), agent.GetName()})
		}
	}

	switch len(matches) {
	case 0:
		return "", fmt.Errorf("agent %q not found. Use 'ei agents list' to list available agents", agentName)
	case 1:
		return matches[0].id, nil
	default:
		lines := []string{fmt.Sprintf("multiple agents named %q found:", agentName)}
		for _, m := range matches {
			lines = append(lines, fmt.Sprintf("  %s  %s", m.id, m.name))
		}
		lines = append(lines, "Specify the agent ID directly to disambiguate.")
		return "", fmt.Errorf("%s", strings.Join(lines, "\n"))
	}
}

// resolveModelToConnectorID resolves a model name to an AI provider connector ID.
func resolveModelToConnectorID(ctx context.Context, apiClient *client.Client, modelName string) (string, error) {
	resp, err := client.GetAIProviderConnectors(ctx, apiClient.Genqlient())
	if err != nil {
		return "", fmt.Errorf("failed to fetch AI provider connectors: %w", err)
	}

	for _, edge := range resp.Organization.AiProviderConnectors.Edges {
		node := edge.GetNode()
		model := node.GetModel()
		provider := node.GetProvider()
		if model.Name == modelName {
			return node.GetId(), nil
		}
		fullName := fmt.Sprintf("%s: %s", provider.GetName(), model.Name)
		if fullName == modelName {
			return node.GetId(), nil
		}
	}

	// Build available models list
	var available []string
	for _, edge := range resp.Organization.AiProviderConnectors.Edges {
		node := edge.GetNode()
		available = append(available, node.GetModel().Name)
	}

	return "", fmt.Errorf("model %q not found. Available: %s", modelName, strings.Join(available, ", "))
}
