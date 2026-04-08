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

var tablesCmd = &cobra.Command{
	Use:   "tables",
	Short: "Manage tables",
	Long:  "Commands for listing and exporting Elementum tables (external data).",
}

var tablesListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all tables",
	Long:  "Display a table of all tables (external data objects) in your organization.",
	RunE:  runTablesList,
}

var tablesExportCmd = &cobra.Command{
	Use:   "export [name-or-id]",
	Short: "Export a table to Terraform",
	Long: `Export an Elementum table as Terraform configuration.

You can specify the table by:
- Name: "Sales Data"
- Handle: sales_data
- ID: uuid

The command will generate an import block and run terraform to create the configuration.`,
	Args: cobra.ExactArgs(1),
	RunE: runTablesExport,
}

func init() {
	tablesCmd.AddCommand(tablesListCmd)
	tablesCmd.AddCommand(tablesExportCmd)

	tablesExportCmd.Flags().StringP("output", "o", "generated.tf", "Output file for generated Terraform configuration")
	tablesExportCmd.Flags().BoolP("verbose", "v", false, "Enable verbose debug output")
}

// GetTablesCmd returns the tables command for registration
func GetTablesCmd() *cobra.Command {
	return tablesCmd
}

func runTablesList(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Get authenticated client
	client, err := auth.GetClientFromCmd(cmd)
	if err != nil {
		return err
	}

	// Show spinner while loading (skip for JSON output)
	if !isJSONOutput(cmd) {
		fmt.Println(ui.InfoStyle.Render("Loading tables..."))
	}

	// Discover tables
	tables, err := discovery.ListTables(ctx, client)
	if err != nil {
		return fmt.Errorf("failed to list tables: %w", err)
	}

	// JSON output
	if isJSONOutput(cmd) {
		return outputJSON(tables)
	}

	if len(tables) == 0 {
		fmt.Println()
		fmt.Println(ui.WarningStyle.Render("No tables found in your organization."))
		fmt.Println()
		return nil
	}

	// Create table
	table := ui.NewTable([]string{"NAME", "HANDLE", "CLOUDLINK", "ID"})

	for _, t := range tables {
		table.AddRow(
			t.Name,
			t.Handle,
			t.CloudLinkName,
			t.ID,
		)
	}

	// Display table
	fmt.Println()
	fmt.Println(ui.TitleStyle.Render("Tables"))
	fmt.Println()
	fmt.Println(table.Render())
	fmt.Println()
	fmt.Println(ui.MutedStyle.Render(fmt.Sprintf("Total: %d tables", len(tables))))
	fmt.Println()

	return nil
}

func runTablesExport(cmd *cobra.Command, args []string) error {
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

	// Get credentials for provider config
	config, _ := auth.LoadConfig()
	creds, err := auth.GetCredentials(cmd, config)
	if err != nil {
		return err
	}

	fmt.Println(ui.InfoStyle.Render("Looking up Table..."))

	// Resolve table - try by name, handle, then ID
	var tableInfo *discovery.Table
	tableInfo, err = discovery.GetTableByName(ctx, client, input)
	if err != nil {
		tableInfo, err = discovery.GetTableByHandle(ctx, client, input)
		if err != nil {
			tableInfo, err = discovery.GetTable(ctx, client, input)
			if err != nil {
				return fmt.Errorf("failed to find Table: %w", err)
			}
		}
	}

	fmt.Println(ui.SuccessStyle.Render(fmt.Sprintf("Found Table: %s (%s)", tableInfo.Name, tableInfo.ID)))

	// Check if terraform is installed
	if err := export.CheckTerraformInstalled(); err != nil {
		return fmt.Errorf("terraform is required: %w", err)
	}

	// Generate import block
	blocks := []export.ImportBlock{
		{
			ID:           tableInfo.ID,
			ResourceType: "elementum_table",
			ResourceName: export.SanitizeName(tableInfo.Name),
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
