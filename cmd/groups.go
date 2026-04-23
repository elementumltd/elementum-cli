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
	"os"

	"github.com/elementumltd/elementum-cli/auth"
	"github.com/elementumltd/elementum-cli/discovery"
	"github.com/elementumltd/elementum-cli/export"
	"github.com/elementumltd/elementum-cli/logger"
	"github.com/elementumltd/elementum-cli/ui"
	"github.com/spf13/cobra"
)

var groupsCmd = &cobra.Command{
	Use:   "groups",
	Short: "Manage groups",
	Long:  "Commands for listing and exporting Elementum groups.",
}

var groupsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all groups",
	Long:  "Display a table of all groups in your organization.",
	RunE:  runGroupsList,
}

var groupsExportCmd = &cobra.Command{
	Use:   "export [name-or-id]",
	Short: "Export a group to Terraform",
	Long: `Export an Elementum group as Terraform configuration.

You can specify the group by:
- Name: "Engineering Team"
- ID: uuid

The command will generate an import block and run terraform to create the configuration.`,
	Args: cobra.ExactArgs(1),
	RunE: runGroupsExport,
}

func init() {
	groupsCmd.AddCommand(groupsListCmd)
	groupsCmd.AddCommand(groupsExportCmd)

	groupsExportCmd.Flags().StringP("output", "o", "generated.tf", "Output file for generated Terraform configuration")
	groupsExportCmd.Flags().BoolP("verbose", "v", false, "Enable verbose debug output")
}

// GetGroupsCmd returns the groups command for registration
func GetGroupsCmd() *cobra.Command {
	return groupsCmd
}

func runGroupsList(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Get authenticated client
	client, err := auth.GetClientFromCmd(cmd)
	if err != nil {
		return err
	}

	// Show spinner while loading (skip for JSON output)
	if !isJSONOutput(cmd) {
		fmt.Println(ui.InfoStyle.Render("Loading groups..."))
	}

	// Discover groups
	groups, err := discovery.ListGroups(ctx, client)
	if err != nil {
		return fmt.Errorf("failed to list groups: %w", err)
	}

	// JSON output
	if isJSONOutput(cmd) {
		return outputJSON(groups)
	}

	if len(groups) == 0 {
		fmt.Println()
		fmt.Println(ui.WarningStyle.Render("No groups found in your organization."))
		fmt.Println()
		return nil
	}

	// Create table
	table := ui.NewTable([]string{"NAME", "MEMBERS", "ID"})

	for _, group := range groups {
		table.AddRow(
			group.Name,
			fmt.Sprintf("%d", len(group.Members)),
			group.ID,
		)
	}

	// Display table
	fmt.Println()
	fmt.Println(ui.TitleStyle.Render("Groups"))
	fmt.Println()
	fmt.Println(table.Render())
	fmt.Println()
	fmt.Println(ui.MutedStyle.Render(fmt.Sprintf("Total: %d groups", len(groups))))
	fmt.Println()

	return nil
}

func runGroupsExport(cmd *cobra.Command, args []string) error {
	input := args[0]
	ctx := context.Background()

	// Get flags
	outputFile, _ := cmd.Flags().GetString("output")
	verbose, _ := cmd.Flags().GetBool("verbose")

	if verbose {
		_ = os.Setenv("DEBUG_GRAPHQL", "1")
	}

	// Set GraphQL debug mode when log level is debug or trace
	level := logger.GetCurrentLevel()
	if level == logger.LevelDebug || level == logger.LevelTrace {
		_ = os.Setenv("DEBUG_GRAPHQL", "1")
	}

	// Get authenticated client
	client, err := auth.GetClientFromCmd(cmd)
	if err != nil {
		return err
	}

	fmt.Println(ui.InfoStyle.Render("Looking up Group..."))

	// Try to get group by name first, then by ID
	group, err := discovery.GetGroupByName(ctx, client, input)
	if err != nil {
		// Try by ID
		group, err = discovery.GetGroup(ctx, client, input)
		if err != nil {
			return fmt.Errorf("failed to find Group: %w", err)
		}
	}

	fmt.Println(ui.SuccessStyle.Render(fmt.Sprintf("Found Group: %s (%s)", group.Name, group.ID)))

	// Generate HCL from discovered data
	fmt.Println(ui.InfoStyle.Render("Generating Terraform configuration..."))
	content := export.GenerateGroupHCL(group)

	// Write to output file
	err = os.WriteFile(outputFile, []byte(content), 0644)
	if err != nil {
		return fmt.Errorf("failed to write output file: %w", err)
	}

	fmt.Println(ui.SuccessStyle.Render(fmt.Sprintf("Generated %s", outputFile)))
	fmt.Println()
	fmt.Println(ui.SubtitleStyle.Render("Next steps:"))
	fmt.Printf("  %s Review %s\n", ui.RenderBullet(), outputFile)
	fmt.Printf("  %s Run: tofu plan\n", ui.RenderBullet())
	fmt.Println()

	return nil
}
