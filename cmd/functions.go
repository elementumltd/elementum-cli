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
	"github.com/elementumltd/elementum-cli/discovery"
	"github.com/elementumltd/elementum-cli/ui"
	"github.com/elementumltd/elementum-cli/internal/client"
	"github.com/spf13/cobra"
)

var functionsCmd = &cobra.Command{
	Use:   "functions",
	Short: "Manage stored Snowflake functions",
	Long:  "Commands for listing and managing stored Snowflake functions (procedures and UDFs).",
}

var functionsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all stored functions",
	Long:  "Display a table of all stored Snowflake functions across all CloudLinks.",
	RunE:  runFunctionsListCmd,
}

var functionsDeleteCmd = &cobra.Command{
	Use:   "delete <id-or-name>",
	Short: "Delete a stored function",
	Long: `Delete a stored Snowflake function by ID or display name.

WARNING: This permanently deletes the stored function.

Examples:
  ei functions delete "Invoice Parser"
  ei functions delete "Invoice Parser" --force
  ei functions delete 794e1e48-73af-4760-... --force`,
	Args: cobra.ExactArgs(1),
	RunE: runFunctionsDeleteCmd,
}

func init() {
	functionsCmd.AddCommand(functionsListCmd)
	functionsCmd.AddCommand(functionsDeleteCmd)

	functionsDeleteCmd.Flags().BoolP("force", "f", false, "Skip confirmation prompt")
}

// GetFunctionsCmd returns the functions command for registration
func GetFunctionsCmd() *cobra.Command {
	return functionsCmd
}

func runFunctionsListCmd(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Get authenticated client
	apiClient, err := auth.GetClientFromCmd(cmd)
	if err != nil {
		return err
	}

	// Show spinner while loading (skip for JSON output)
	if !isJSONOutput(cmd) {
		fmt.Println(ui.InfoStyle.Render("Loading stored functions..."))
	}

	// Discover stored functions
	functions, err := discovery.ListAllStoredFunctions(ctx, apiClient)
	if err != nil {
		return fmt.Errorf("failed to list stored functions: %w", err)
	}

	// JSON output
	if isJSONOutput(cmd) {
		return outputJSON(functions)
	}

	if len(functions) == 0 {
		fmt.Println()
		fmt.Println(ui.WarningStyle.Render("No stored functions found in your organization."))
		fmt.Println()
		return nil
	}

	// Create table
	table := ui.NewTable([]string{"DISPLAY NAME", "NAME", "TYPE", "DATABASE.SCHEMA", "CLOUDLINK", "ID"})

	for _, fn := range functions {
		location := ""
		if fn.DatabaseName != "" && fn.SchemaName != "" {
			location = fn.DatabaseName + "." + fn.SchemaName
		}

		fnType := fn.Type
		if fnType == "PROCEDURE" {
			fnType = "Procedure"
		} else if fnType == "USER_DEFINED_FUNCTION" {
			fnType = "UDF"
		}

		table.AddRow(
			fn.DisplayName,
			fn.Name,
			fnType,
			location,
			fn.CloudLinkName,
			fn.ID,
		)
	}

	// Display table
	fmt.Println()
	fmt.Println(ui.TitleStyle.Render("Stored Functions"))
	fmt.Println()
	fmt.Println(table.Render())
	fmt.Println()
	fmt.Println(ui.MutedStyle.Render(fmt.Sprintf("Total: %d stored functions", len(functions))))
	fmt.Println()

	return nil
}

func runFunctionsDeleteCmd(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	apiClient, err := auth.GetClientFromCmd(cmd)
	if err != nil {
		return err
	}

	nameOrID := args[0]

	// Resolve function ID and CloudLink ID
	fn, err := resolveStoredFunction(ctx, apiClient, nameOrID)
	if err != nil {
		return err
	}

	force, _ := cmd.Flags().GetBool("force")

	// Get display name for confirmation
	displayName := fn.DisplayName
	if displayName == "" {
		displayName = fn.Name
	}

	if !force {
		confirmed, err := ui.Confirm(
			fmt.Sprintf("Delete stored function %q?", displayName),
			fmt.Sprintf("ID: %s\nCloudLink: %s\nThis will permanently delete the stored function.", fn.ID, fn.CloudLinkName),
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
		fmt.Println(ui.InfoStyle.Render(fmt.Sprintf("Deleting stored function %q...", displayName)))
	}

	_, err = client.DeleteStoredSnowflakeFunction(ctx, apiClient.Genqlient(), fn.ID)
	if err != nil {
		// Fetch and display dependencies as blockers
		if blockers := getStoredFunctionBlockers(ctx, apiClient, fn.CloudLinkID, fn.ID); blockers != "" {
			fmt.Println()
			fmt.Println(ui.ErrorStyle.Render("Cannot delete stored function - blocked by dependencies:"))
			fmt.Println(blockers)
			fmt.Println()
			fmt.Println(ui.MutedStyle.Render("Delete or update the blocking resources first, then retry."))
			return fmt.Errorf("stored function has active dependencies")
		}
		return fmt.Errorf("failed to delete stored function: %w", err)
	}

	if isJSONOutput(cmd) {
		return outputJSON(map[string]string{
			"id":          fn.ID,
			"name":        fn.Name,
			"displayName": displayName,
			"deleted":     "true",
		})
	}

	fmt.Println(ui.SuccessStyle.Render("Deleted stored function:") + fmt.Sprintf(" %s (%s)", displayName, fn.ID))
	return nil
}

// resolveStoredFunction finds a stored function by ID or display name
func resolveStoredFunction(ctx context.Context, apiClient *client.Client, nameOrID string) (*discovery.StoredFunction, error) {
	// If it looks like a UUID, try to find by ID
	if looksLikeUUID(nameOrID) {
		fn, err := discovery.GetStoredFunctionByID(ctx, apiClient, nameOrID)
		if err != nil {
			return nil, fmt.Errorf("stored function not found: %s", nameOrID)
		}
		return fn, nil
	}

	// Otherwise, search by display name
	functions, err := discovery.ListAllStoredFunctions(ctx, apiClient)
	if err != nil {
		return nil, fmt.Errorf("failed to list stored functions: %w", err)
	}

	var matches []discovery.StoredFunction
	for _, fn := range functions {
		if strings.EqualFold(fn.DisplayName, nameOrID) || strings.EqualFold(fn.Name, nameOrID) {
			matches = append(matches, fn)
		}
	}

	if len(matches) == 0 {
		return nil, fmt.Errorf("stored function not found: %q", nameOrID)
	}

	if len(matches) > 1 {
		var names []string
		for _, fn := range matches {
			names = append(names, fmt.Sprintf("%s (%s)", fn.DisplayName, fn.ID))
		}
		return nil, fmt.Errorf("multiple stored functions match %q: %s\nSpecify the ID to disambiguate", nameOrID, strings.Join(names, ", "))
	}

	return &matches[0], nil
}

// getStoredFunctionBlockers fetches dependencies that block deleting a stored function
// and returns a formatted string of blockers, or empty string if no blockers found
func getStoredFunctionBlockers(ctx context.Context, c *client.Client, cloudLinkID, functionID string) string {
	resp, err := client.GetStoredFunctionDependencies(ctx, c.Genqlient(), cloudLinkID, functionID)
	if err != nil {
		return "" // Can't fetch dependencies, return empty
	}

	cloudLink := resp.Organization.CloudLink
	if cloudLink == nil {
		return ""
	}

	var blockers []string

	// Type switch to access the stored function
	switch cl := (*cloudLink).(type) {
	case *client.GetStoredFunctionDependenciesOrganizationCloudLinkCloudLinkSnowflake:
		sf := cl.StoredSnowflakeFunction
		if sf.Usage == nil {
			return ""
		}

		for _, edge := range sf.Usage.Automations.Edges {
			blockers = append(blockers, fmt.Sprintf("  - Automation: %s (%s)", edge.Node.Name, edge.Node.Id))
		}
	}

	if len(blockers) == 0 {
		return ""
	}

	return strings.Join(blockers, "\n")
}
