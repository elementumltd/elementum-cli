// Copyright (c) ElementumAI, Inc.
// SPDX-License-Identifier: MPL-2.0

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
		os.Setenv("DEBUG_GRAPHQL", "1")
	}

	// Set GraphQL debug mode when log level is debug or trace
	level := logger.GetCurrentLevel()
	if level == logger.LevelDebug || level == logger.LevelTrace {
		os.Setenv("DEBUG_GRAPHQL", "1")
	}

	// Get authenticated client
	client, err := auth.GetClientFromCmd(cmd)
	if err != nil {
		return err
	}

	// Get credentials for provider config
	config, _ := auth.LoadConfig()
	creds, err := auth.GetCredentials(cmd, config)
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

	// Check if terraform is installed
	if err := export.CheckTerraformInstalled(); err != nil {
		return fmt.Errorf("terraform is required: %w", err)
	}

	// Generate import block
	blocks := []export.ImportBlock{
		{
			ID:           group.ID,
			ResourceType: "elementum_group",
			ResourceName: export.SanitizeName(group.Name),
		},
	}

	// Create Terraform runner
	runner, err := export.NewTerraformRunner()
	if err != nil {
		return fmt.Errorf("failed to create terraform runner: %w", err)
	}
	fmt.Printf("Working directory: %s\n", runner.WorkDir)

	// Write imports and provider config
	providerConfig := export.RenderProviderConfig(creds.Organization, creds.Instance, creds.Environment, creds.ClientID, creds.ClientSecret)
	imports := export.RenderImportBlocks(blocks)

	err = runner.WriteImports(providerConfig, imports)
	if err != nil {
		return fmt.Errorf("failed to write import files: %w", err)
	}

	fmt.Println(ui.SuccessStyle.Render("Generated import block"))

	// Run terraform init
	fmt.Println(ui.InfoStyle.Render("Running terraform init..."))
	if err := runner.Init(); err != nil {
		return fmt.Errorf("terraform init failed: %w", err)
	}
	fmt.Println(ui.SuccessStyle.Render("Terraform initialized"))

	// Run terraform plan -generate-config-out
	fmt.Println(ui.InfoStyle.Render("Generating Terraform configuration..."))
	generatedFile := "generated.tf"
	_, err = runner.GenerateConfig(generatedFile)
	if err != nil {
		return fmt.Errorf("terraform plan failed: %w", err)
	}

	// Read the generated file
	content, err := runner.GetGeneratedFile(generatedFile)
	if err != nil {
		return fmt.Errorf("failed to read generated config: %w", err)
	}

	// Write to output file
	err = os.WriteFile(outputFile, []byte(content), 0644)
	if err != nil {
		return fmt.Errorf("failed to write output file: %w", err)
	}

	fmt.Println(ui.SuccessStyle.Render(fmt.Sprintf("Generated %s", outputFile)))
	fmt.Println()
	fmt.Println(ui.SubtitleStyle.Render("Next steps:"))
	fmt.Printf("  %s Review %s\n", ui.RenderBullet(), outputFile)
	fmt.Printf("  %s Run: terraform plan\n", ui.RenderBullet())
	fmt.Println()

	return nil
}
