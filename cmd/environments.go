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
	"github.com/elementumltd/elementum-cli/internal/client"
	"github.com/elementumltd/elementum-cli/logger"
	"github.com/elementumltd/elementum-cli/ui"
	"github.com/spf13/cobra"
)

var environmentsCmd = &cobra.Command{
	Use:     "environments",
	Aliases: []string{"envs"},
	Short:   "Manage environments",
	Long:    "Commands for listing Elementum organization environments.",
}

var environmentsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List environments",
	Long: `List all environments in the organization.

Examples:
  ei environments list
  ei envs list
  ei envs list --json`,
	RunE: runEnvironmentsList,
}

func init() {
	environmentsCmd.AddCommand(environmentsListCmd)
}

// GetEnvironmentsCmd returns the environments command for registration
func GetEnvironmentsCmd() *cobra.Command {
	return environmentsCmd
}

func runEnvironmentsList(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	apiClient, err := auth.GetClientFromCmd(cmd)
	if err != nil {
		return err
	}

	if !isJSONOutput(cmd) {
		fmt.Println(ui.InfoStyle.Render("Loading environments..."))
	}

	logger.Debug("listing environments")
	resp, err := client.ListEnvironments(ctx, apiClient.Genqlient())
	if err != nil {
		return fmt.Errorf("failed to list environments: %w", err)
	}

	edges := resp.Organization.Environments.Edges
	logger.Debug("environments loaded", "count", len(edges))

	if isJSONOutput(cmd) {
		return outputJSON(edges)
	}

	if len(edges) == 0 {
		fmt.Println()
		fmt.Println(ui.WarningStyle.Render("No environments found."))
		fmt.Println()
		return nil
	}

	table := ui.NewTable([]string{"NAME", "SUBDOMAIN", "DESCRIPTION", "ID"})

	for _, edge := range edges {
		env := edge.Node
		subdomain := ""
		if env.Subdomain != nil {
			subdomain = *env.Subdomain
		}
		description := ""
		if env.Description != nil {
			description = *env.Description
		}
		description = ui.TruncateUTF8(description, 50)
		table.AddRow(env.Name, subdomain, description, env.Id)
	}

	fmt.Println()
	fmt.Println(ui.TitleStyle.Render("Environments"))
	fmt.Println()
	fmt.Println(table.Render())
	fmt.Println()
	fmt.Println(ui.MutedStyle.Render(fmt.Sprintf("Total: %d environments", len(edges))))
	fmt.Println()

	return nil
}

// resolveEnvironmentByName resolves an environment name to its ID.
// Matches case-insensitively. If the input looks like a UUID, returns it directly.
func resolveEnvironmentByName(ctx context.Context, apiClient *client.Client, nameOrID string) (id string, name string, err error) {
	if looksLikeUUID(nameOrID) {
		return nameOrID, nameOrID, nil
	}

	logger.Debug("resolving environment by name", "name", nameOrID)
	resp, err := client.ListEnvironments(ctx, apiClient.Genqlient())
	if err != nil {
		return "", "", fmt.Errorf("failed to list environments: %w", err)
	}

	var matches []struct{ id, name string }
	for _, edge := range resp.Organization.Environments.Edges {
		env := edge.Node
		if strings.EqualFold(env.Name, nameOrID) {
			matches = append(matches, struct{ id, name string }{env.Id, env.Name})
		}
	}

	switch len(matches) {
	case 0:
		return "", "", fmt.Errorf("environment %q not found. Use 'ei environments list' to see available environments", nameOrID)
	case 1:
		return matches[0].id, matches[0].name, nil
	default:
		lines := []string{fmt.Sprintf("multiple environments named %q found:", nameOrID)}
		for _, m := range matches {
			lines = append(lines, fmt.Sprintf("  %s  %s", m.id, m.name))
		}
		lines = append(lines, "Specify the environment ID directly to disambiguate.")
		return "", "", fmt.Errorf("%s", strings.Join(lines, "\n"))
	}
}
