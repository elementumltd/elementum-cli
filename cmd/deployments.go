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
	"encoding/json"
	"fmt"

	"github.com/elementumltd/elementum-cli/auth"
	"github.com/elementumltd/elementum-cli/internal/client"
	"github.com/elementumltd/elementum-cli/logger"
	"github.com/elementumltd/elementum-cli/ui"
	"github.com/spf13/cobra"
)

var deploymentsCmd = &cobra.Command{
	Use:   "deployments",
	Short: "Manage deployments",
	Long:  "Commands for listing, creating, showing, and configuring deployments across environments.",
}

var deploymentsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List deployments",
	Long: `List deployments in the organization.

Examples:
  ei deployments list
  ei deployments list --app my-app
  ei deployments list --limit 50
  ei deployments list --json`,
	RunE: runDeploymentsList,
}

func init() {
	deploymentsListCmd.Flags().String("app", "", "Filter by app namespace")
	deploymentsListCmd.Flags().Int("limit", 20, "Maximum number of deployments to return")

	deploymentsCmd.AddCommand(deploymentsListCmd)
	deploymentsCmd.AddCommand(deploymentsShowCmd)
	deploymentsCmd.AddCommand(deploymentsCreateCmd)
	deploymentsCmd.AddCommand(deploymentsConfigureCmd)
}

// GetDeploymentsCmd returns the deployments command for registration
func GetDeploymentsCmd() *cobra.Command {
	return deploymentsCmd
}

func runDeploymentsList(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	apiClient, err := auth.GetClientFromCmd(cmd)
	if err != nil {
		return err
	}

	appNamespace, _ := cmd.Flags().GetString("app")
	limit, _ := cmd.Flags().GetInt("limit")

	if !isJSONOutput(cmd) {
		fmt.Println(ui.InfoStyle.Render("Loading deployments..."))
	}

	// Build filter if app namespace is provided
	var filter *json.RawMessage
	if appNamespace != "" {
		aspectID, _, err := resolveAspectByNamespace(ctx, apiClient, appNamespace)
		if err != nil {
			return fmt.Errorf("failed to find app %q: %w", appNamespace, err)
		}
		// Filter scalar requires raw JSON — build via json.Marshal for safety
		filterObj := map[string]interface{}{
			"type":  "EQUALS",
			"field": "appId",
			"value": map[string]interface{}{"type": "ID", "value": aspectID},
		}
		f, err := json.Marshal(filterObj)
		if err != nil {
			return fmt.Errorf("failed to build filter: %w", err)
		}
		raw := json.RawMessage(f)
		filter = &raw
	}

	logger.Debug("listing deployments", "app", appNamespace, "limit", limit)
	resp, err := client.ListDeployments(ctx, apiClient.Genqlient(), &limit, filter)
	if err != nil {
		return fmt.Errorf("failed to list deployments: %w", err)
	}

	edges := resp.Organization.Deployments.Edges
	logger.Debug("deployments loaded", "count", len(edges), "total", resp.Organization.Deployments.Total)

	if isJSONOutput(cmd) {
		return outputJSON(edges)
	}

	if len(edges) == 0 {
		fmt.Println()
		fmt.Println(ui.WarningStyle.Render("No deployments found."))
		fmt.Println()
		return nil
	}

	table := ui.NewTable([]string{"STATUS", "APP", "SOURCE → TARGET", "NAME", "CREATED", "ID"})

	for _, edge := range edges {
		d := edge.Node
		status := styledDeploymentStatus(string(d.Status))
		appName := d.App.Name
		envFlow := fmt.Sprintf("%s → %s", d.SourceEnvironment.Name, d.TargetEnvironment.Name)
		name := ""
		if d.Name != nil {
			name = *d.Name
		}
		created := formatTimestamp(d.CreatedAt)

		table.AddRow(status, appName, envFlow, name, created, d.Id)
	}

	fmt.Println()
	fmt.Println(ui.TitleStyle.Render("Deployments"))
	fmt.Println()
	fmt.Println(table.Render())
	fmt.Println()
	fmt.Println(ui.MutedStyle.Render(fmt.Sprintf("Total: %d deployments", resp.Organization.Deployments.Total)))
	fmt.Println()

	return nil
}

// styledDeploymentStatus returns a color-coded deployment status string.
func styledDeploymentStatus(status string) string {
	switch status {
	case "COMPLETED":
		return ui.SuccessStyle.Render("COMPLETED")
	case "FAILED":
		return ui.ErrorStyle.Render("FAILED")
	case "CONFIGURATION":
		return ui.InfoStyle.Render("CONFIGURATION")
	case "PENDING", "SCOPING", "EXTRACTING", "DIFFING":
		return ui.WarningStyle.Render(status)
	default:
		return status
	}
}
