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
	"github.com/elementumltd/elementum-cli/ui"
	"github.com/spf13/cobra"
)

var elementsSearchTablesCmd = &cobra.Command{
	Use:   "search-tables",
	Short: "Manage search tables for an element",
	Long:  "Commands for listing and deleting AI search tables on elements.",
}

var elementsSearchTablesListCmd = &cobra.Command{
	Use:   "list <namespace>",
	Short: "List search tables for an element",
	Long: `List all AI search tables configured for an element.

Examples:
  ei elements search-tables list users
  ei elements search-tables list users --json`,
	Args: cobra.ExactArgs(1),
	RunE: runElementsSearchTablesList,
}

var elementsSearchTablesDeleteCmd = &cobra.Command{
	Use:   "delete <id>",
	Short: "Delete a search table",
	Long: `Delete an AI search table by ID.

Examples:
  ei elements search-tables delete 9063aed1-bf8c-430d-882f-8c502355a3c7
  ei elements search-tables delete 9063aed1-bf8c-430d-882f-8c502355a3c7 --force`,
	Args: cobra.ExactArgs(1),
	RunE: runSearchTableDelete,
}

func init() {
	elementsSearchTablesCmd.AddCommand(elementsSearchTablesListCmd)
	elementsSearchTablesCmd.AddCommand(elementsSearchTablesDeleteCmd)
	elementsSearchTablesDeleteCmd.Flags().BoolP("force", "f", false, "Skip confirmation prompt")
	elementsCmd.AddCommand(elementsSearchTablesCmd)
}

func runElementsSearchTablesList(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	c, err := auth.GetClientFromCmd(cmd)
	if err != nil {
		return err
	}

	namespace := args[0]

	if !isJSONOutput(cmd) {
		fmt.Println(ui.InfoStyle.Render(fmt.Sprintf("Loading search tables for %q...", namespace)))
	}

	// Query search tables by namespace
	resp, err := client.GetAspectSearchTablesByNamespace(ctx, c.Genqlient(), namespace)
	if err != nil {
		return fmt.Errorf("failed to get search tables: %w", err)
	}

	// Find the aspect in the response
	edges := resp.Organization.Aspects.Edges
	if len(edges) == 0 {
		return fmt.Errorf("element not found: %s", namespace)
	}

	aspect := edges[0].Node

	// Check that it's an element
	elemAspect, ok := aspect.(*client.GetAspectSearchTablesByNamespaceOrganizationAspectsAspectConnectionEdgesAspectEdgeNodeAspectElement)
	if !ok {
		return fmt.Errorf("%q is not an element", namespace)
	}

	searchTables := elemAspect.SearchTables.Edges

	// JSON output
	if isJSONOutput(cmd) {
		type searchTableInfo struct {
			ID              string   `json:"id"`
			Field           string   `json:"field"`
			FieldID         string   `json:"field_id"`
			Status          string   `json:"status"`
			ErrorMessage    string   `json:"error_message,omitempty"`
			AttributeFields []string `json:"attribute_fields,omitempty"`
		}
		var tables []searchTableInfo
		for _, edge := range searchTables {
			st := edge.Node
			var attrs []string
			for _, af := range st.GetAttributeFields() {
				attrs = append(attrs, af.GetName())
			}
			errMsg := ""
			if st.GetErrorMessage() != nil {
				errMsg = *st.GetErrorMessage()
			}
			var fieldName, fieldID string
			if fieldPtr := st.GetField(); fieldPtr != nil {
				field := *fieldPtr
				fieldName = field.GetName()
				fieldID = field.GetId()
			}
			tables = append(tables, searchTableInfo{
				ID:              st.GetId(),
				Field:           fieldName,
				FieldID:         fieldID,
				Status:          string(st.GetStatus()),
				ErrorMessage:    errMsg,
				AttributeFields: attrs,
			})
		}
		return outputJSON(map[string]interface{}{
			"element":       aspect.GetName(),
			"element_id":    aspect.GetId(),
			"namespace":     elemAspect.Namespace,
			"search_tables": tables,
		})
	}

	if len(searchTables) == 0 {
		fmt.Println()
		fmt.Println(ui.WarningStyle.Render(fmt.Sprintf("No search tables found for element %q", namespace)))
		fmt.Println()
		return nil
	}

	// Create table
	table := ui.NewTable([]string{"ID", "FIELD", "STATUS", "ATTRIBUTES"})

	for _, edge := range searchTables {
		st := edge.Node
		var attrs []string
		for _, af := range st.GetAttributeFields() {
			attrs = append(attrs, af.GetName())
		}
		attrsStr := strings.Join(attrs, ", ")
		if len(attrsStr) > 40 {
			attrsStr = attrsStr[:37] + "..."
		}

		status := string(st.GetStatus())
		errMsg := st.GetErrorMessage()
		if errMsg != nil && *errMsg != "" {
			status = fmt.Sprintf("%s: %s", status, *errMsg)
		}

		var fieldNameForTable string
		if fp := st.GetField(); fp != nil {
			f := *fp
			fieldNameForTable = f.GetName()
		}

		table.AddRow(
			st.GetId(),
			fieldNameForTable,
			status,
			attrsStr,
		)
	}

	// Display table
	fmt.Println()
	fmt.Println(ui.TitleStyle.Render(fmt.Sprintf("Search Tables for %s (%s)", aspect.GetName(), namespace)))
	fmt.Println()
	fmt.Println(table.Render())
	fmt.Println()
	fmt.Println(ui.MutedStyle.Render(fmt.Sprintf("Total: %d search tables", len(searchTables))))
	fmt.Println()

	return nil
}

// runSearchTableDelete is shared between elements and apps
func runSearchTableDelete(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	c, err := auth.GetClientFromCmd(cmd)
	if err != nil {
		return err
	}

	searchTableID := args[0]
	force, _ := cmd.Flags().GetBool("force")

	if !force {
		confirmed, err := ui.Confirm(
			fmt.Sprintf("Delete search table %s?", searchTableID),
			"This will permanently delete the AI search table.\nThis action cannot be undone.",
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
		fmt.Println(ui.InfoStyle.Render("Deleting search table..."))
	}

	_, err = client.DeleteAspectSearchTable(ctx, c.Genqlient(), searchTableID)
	if err != nil {
		// Delete failed - fetch usages to explain why
		usages := fetchSearchTableUsages(ctx, c, searchTableID)
		if len(usages) > 0 {
			fmt.Println(ui.ErrorStyle.Render(fmt.Sprintf("Cannot delete search table %s - the following resources are using it:", searchTableID)))
			for _, u := range usages {
				fmt.Printf("  - %s\n", u)
			}
			fmt.Println()
			fmt.Println(ui.MutedStyle.Render("Remove or update these resources first."))
			return fmt.Errorf("delete blocked by usages")
		}
		return fmt.Errorf("failed to delete search table: %w", err)
	}

	if isJSONOutput(cmd) {
		return outputJSON(map[string]string{
			"id":      searchTableID,
			"deleted": "true",
		})
	}

	fmt.Println(ui.SuccessStyle.Render("Deleted search table:") + fmt.Sprintf(" %s", searchTableID))
	return nil
}

// fetchSearchTableUsages retrieves all resources that use the given search table.
func fetchSearchTableUsages(ctx context.Context, c *client.Client, searchTableID string) []string {
	result, err := client.GetAspectSearchTableDependencies(ctx, c.Genqlient(), searchTableID)
	if err != nil {
		return nil
	}

	var usages []string
	st := result.AspectSearchTableRefreshStatus
	if st == nil {
		return usages
	}

	usage := st.GetUsage()

	// Agent Tools (includes skill tools)
	for _, edge := range usage.AgentTools.Edges {
		node := edge.Node
		if node != nil {
			usages = append(usages, fmt.Sprintf("agent tool: %q (id: %s)", node.GetName(), node.GetId()))
		}
	}

	// Agents
	for _, edge := range usage.Agents.Edges {
		node := edge.Node
		if node != nil {
			usages = append(usages, fmt.Sprintf("agent: %q (id: %s)", node.GetName(), node.GetId()))
		}
	}

	// Automations
	for _, edge := range usage.Automations.Edges {
		usages = append(usages, fmt.Sprintf("automation: %q (id: %s)", edge.Node.Name, edge.Node.Id))
	}

	return usages
}
