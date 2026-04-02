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
	"github.com/elementumltd/elementum-cli/discovery"
	"github.com/elementumltd/elementum-cli/ui"
	"github.com/elementumltd/elementum-cli/internal/client"
	"github.com/spf13/cobra"
)

var appDeleteCmd = &cobra.Command{
	Use:   "delete <namespace-or-id>",
	Short: "Delete an app or element",
	Long: `Delete an Elementum App or Element by namespace or ID.

WARNING: This permanently deletes the app and all its records, automations, and agents.

Examples:
  ei apps delete support-tickets
  ei apps delete support-tickets --force
  ei apps delete 9063aed1-bf8c-430d-882f-8c502355a3c7
  ei apps delete support-tickets --dry-run`,
	Args: cobra.ExactArgs(1),
	RunE: runAppDelete,
}

func init() {
	appDeleteCmd.Flags().BoolP("force", "f", false, "Skip confirmation prompt")
	appDeleteCmd.Flags().Bool("dry-run", false, "Show what would be deleted without deleting")
}

func runAppDelete(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	c, err := auth.GetClientFromCmd(cmd)
	if err != nil {
		return err
	}

	input := args[0]
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	force, _ := cmd.Flags().GetBool("force")

	var aspectID, aspectName, aspectType string

	if looksLikeUUID(input) {
		aspectID = input
		info, err := getAspectSummary(ctx, c, aspectID)
		if err != nil {
			return fmt.Errorf("app not found: %s", aspectID)
		}
		aspectName = info.Name
		aspectType = info.Type
	} else {
		aspectID, aspectType, err = resolveAspectByNamespace(ctx, c, input)
		if err != nil {
			return fmt.Errorf("failed to resolve %q: %w", input, err)
		}
		aspectName = input
	}

	if dryRun {
		fmt.Println(ui.WarningStyle.Render("Dry run:") + fmt.Sprintf(" Would delete %s %q (%s)", aspectType, aspectName, aspectID))
		return nil
	}

	if !force {
		confirmed, err := ui.Confirm(
			fmt.Sprintf("Delete %s %q?", aspectType, aspectName),
			fmt.Sprintf("ID: %s\nThis will permanently delete the %s and all its records, automations, agents, and fields.\nThis action cannot be undone.", aspectID, aspectType),
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
		fmt.Println(ui.InfoStyle.Render(fmt.Sprintf("⠿ Deleting %s %q...", aspectType, aspectName)))
	}

	_, err = client.DeleteAspect(ctx, c.Genqlient(), aspectID)
	if err != nil {
		// Delete failed - fetch usages to explain why
		usages := fetchAspectUsages(ctx, c, aspectID)
		if len(usages) > 0 {
			fmt.Println(ui.ErrorStyle.Render(fmt.Sprintf("Cannot delete %s %q - the following resources are using it:", aspectType, aspectName)))
			for _, u := range usages {
				fmt.Printf("  - %s\n", u)
			}
			fmt.Println()
			fmt.Println(ui.MutedStyle.Render("Remove or update these resources first."))
			return fmt.Errorf("delete blocked by usages")
		}
		return fmt.Errorf("failed to delete %s: %w", aspectType, err)
	}

	if isJSONOutput(cmd) {
		return outputJSON(map[string]string{
			"id":      aspectID,
			"name":    aspectName,
			"type":    aspectType,
			"deleted": "true",
		})
	}

	fmt.Println(ui.SuccessStyle.Render(fmt.Sprintf("Deleted %s:", aspectType)) + fmt.Sprintf(" %s (%s)", aspectName, aspectID))
	return nil
}

// fetchAspectUsages retrieves all resources that use the given aspect.
func fetchAspectUsages(ctx context.Context, c *client.Client, aspectID string) []string {
	result, err := client.GetAspectDependencies(ctx, c.Genqlient(), aspectID)
	if err != nil {
		return nil
	}

	var usages []string
	aspect := result.Organization.Aspect
	if aspect == nil {
		return usages
	}

	usage := (*aspect).GetUsage()

	// Automations
	for _, edge := range usage.Automations.Edges {
		if edge.Node != nil {
			usages = append(usages, fmt.Sprintf("automation: %q (id: %s)", edge.Node.Name, edge.Node.Id))
		}
	}

	// Agent Tools
	for _, edge := range usage.AgentTools.Edges {
		if edge.Node != nil {
			usages = append(usages, fmt.Sprintf("agent tool: %q (id: %s)", (*edge.Node).GetName(), (*edge.Node).GetId()))
		}
	}

	// Fields
	for _, edge := range usage.Fields.Edges {
		if edge.Node != nil {
			usages = append(usages, fmt.Sprintf("field: %q (id: %s)", (*edge.Node).GetName(), (*edge.Node).GetId()))
		}
	}

	// Relations
	for _, edge := range usage.Relations.Edges {
		if edge.Node != nil {
			// Aspect is an interface (not a pointer), call GetName() directly
			aspectName := edge.Node.Aspect.GetName()
			if aspectName != "" {
				usages = append(usages, fmt.Sprintf("relation on %q (id: %s)", aspectName, edge.Node.Id))
			} else {
				usages = append(usages, fmt.Sprintf("relation: (id: %s)", edge.Node.Id))
			}
		}
	}

	// Search Tables
	for _, edge := range usage.SearchTables.Edges {
		if edge.Node != nil {
			// edge.Node is a pointer to interface, dereference and use interface methods
			node := *edge.Node
			fieldName := node.GetField().GetName()
			nodeID := node.GetId()
			if fieldName != "" {
				usages = append(usages, fmt.Sprintf("search table on field %q (id: %s)", fieldName, nodeID))
			} else {
				usages = append(usages, fmt.Sprintf("search table: (id: %s)", nodeID))
			}
		}
	}

	// Agents
	for _, edge := range usage.Agents.Edges {
		if edge.Node != nil {
			usages = append(usages, fmt.Sprintf("agent: %q (id: %s)", (*edge.Node).GetName(), (*edge.Node).GetId()))
		}
	}

	// Tables
	for _, edge := range usage.Tables.Edges {
		if edge.Node != nil {
			usages = append(usages, fmt.Sprintf("table: %q (id: %s)", edge.Node.Name, edge.Node.Id))
		}
	}

	// Datamines
	for _, edge := range usage.Datamines.Edges {
		if edge.Node != nil {
			usages = append(usages, fmt.Sprintf("datamine: %q (id: %s)", edge.Node.Name, edge.Node.Id))
		}
	}

	// Display Widgets
	for _, edge := range usage.DisplayWidgets.Edges {
		if edge.Node != nil {
			usages = append(usages, fmt.Sprintf("display widget: %q (id: %s)", (*edge.Node).GetName(), (*edge.Node).GetId()))
		}
	}

	return usages
}

type aspectSummary struct {
	Name string
	Type string
}

func getAspectSummary(ctx context.Context, c *client.Client, aspectID string) (*aspectSummary, error) {
	info, err := discovery.GetAspectInfo(ctx, c, aspectID)
	if err != nil {
		return nil, err
	}
	return &aspectSummary{Name: info.Name, Type: string(info.Type)}, nil
}
