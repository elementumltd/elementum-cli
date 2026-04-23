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

var dataminesCmd = &cobra.Command{
	Use:   "datamines",
	Short: "Manage datamines",
	Long:  "Commands for listing and exporting Elementum datamines.",
}

var dataminesListCmd = &cobra.Command{
	Use:   "list [table-name-or-id]",
	Short: "List datamines",
	Long: `Display datamines in your organization.

If a table name or ID is provided, list datamines for that specific table.
Otherwise, list all datamines across all tables.`,
	Args: cobra.MaximumNArgs(1),
	RunE: runDataminesList,
}

var dataminesExportCmd = &cobra.Command{
	Use:   "export [table-id-or-name] [datamine-name-or-id]",
	Short: "Export a datamine to Terraform",
	Long: `Export an Elementum Datamine as Terraform configuration.

You can specify the datamine by:
- Table ID/name and Datamine name: ei datamines export "Sales Summary" "Stale Records Monitor"
- Table ID and Datamine ID: ei datamines export tbl_12345 dm_67890

The command will generate an import block and run terraform to create the configuration.`,
	Args: cobra.ExactArgs(2),
	RunE: runDataminesExport,
}

func init() {
	dataminesCmd.AddCommand(dataminesListCmd)
	dataminesCmd.AddCommand(dataminesExportCmd)

	dataminesExportCmd.Flags().StringP("output", "o", "generated.tf", "Output file for generated Terraform configuration")
	dataminesExportCmd.Flags().BoolP("verbose", "v", false, "Enable verbose debug output")
	dataminesExportCmd.Flags().Bool("beautify", true, "Resolve UUIDs to references and strip null attributes")
}

// GetDataminesCmd returns the datamines command for registration
func GetDataminesCmd() *cobra.Command {
	return dataminesCmd
}

func runDataminesList(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Get authenticated client
	client, err := auth.GetClientFromCmd(cmd)
	if err != nil {
		return err
	}

	// Show spinner while loading (skip for JSON output)
	if !isJSONOutput(cmd) {
		fmt.Println(ui.InfoStyle.Render("Loading datamines..."))
	}

	var datamines []discovery.Datamine

	if len(args) > 0 {
		// List datamines for a specific table
		tableInput := args[0]

		// Resolve table
		tableInfo, err := discovery.GetTableByName(ctx, client, tableInput)
		if err != nil {
			tableInfo, err = discovery.GetTableByHandle(ctx, client, tableInput)
			if err != nil {
				tableInfo, err = discovery.GetTable(ctx, client, tableInput)
				if err != nil {
					return fmt.Errorf("failed to find table: %w", err)
				}
			}
		}

		datamines, err = discovery.ListDatamines(ctx, client, tableInfo.ID)
		if err != nil {
			return fmt.Errorf("failed to list datamines: %w", err)
		}
	} else {
		// List all datamines across all tables
		datamines, err = discovery.ListAllDatamines(ctx, client)
		if err != nil {
			return fmt.Errorf("failed to list datamines: %w", err)
		}
	}

	// JSON output
	if isJSONOutput(cmd) {
		return outputJSON(datamines)
	}

	if len(datamines) == 0 {
		fmt.Println()
		fmt.Println(ui.WarningStyle.Render("No datamines found."))
		fmt.Println()
		return nil
	}

	// Create table
	table := ui.NewTable([]string{"NAME", "TABLE", "SCHEDULE", "ID"})

	for _, dm := range datamines {
		schedule := dm.ScheduleType
		if schedule == "" {
			schedule = "-"
		}
		table.AddRow(
			dm.Name,
			dm.TableName,
			schedule,
			dm.ID,
		)
	}

	// Display table
	fmt.Println()
	fmt.Println(ui.TitleStyle.Render("Datamines"))
	fmt.Println()
	fmt.Println(table.Render())
	fmt.Println()
	fmt.Println(ui.MutedStyle.Render(fmt.Sprintf("Total: %d datamines", len(datamines))))
	fmt.Println()

	return nil
}

func runDataminesExport(cmd *cobra.Command, args []string) error {
	tableInput := args[0]
	datamineInput := args[1]
	ctx := context.Background()

	// Get flags
	outputFile, _ := cmd.Flags().GetString("output")
	verbose, _ := cmd.Flags().GetBool("verbose")
	beautify, _ := cmd.Flags().GetBool("beautify")

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

	fmt.Println(ui.InfoStyle.Render("Looking up Table..."))

	// Resolve table - try by name, handle, then ID
	var tableInfo *discovery.Table
	tableInfo, err = discovery.GetTableByName(ctx, client, tableInput)
	if err != nil {
		tableInfo, err = discovery.GetTableByHandle(ctx, client, tableInput)
		if err != nil {
			tableInfo, err = discovery.GetTable(ctx, client, tableInput)
			if err != nil {
				return fmt.Errorf("failed to find Table: %w", err)
			}
		}
	}

	fmt.Println(ui.SuccessStyle.Render(fmt.Sprintf("Found Table: %s (%s)", tableInfo.Name, tableInfo.ID)))

	fmt.Println(ui.InfoStyle.Render("Looking up Datamine..."))

	// Resolve datamine - try by name first, then by ID
	var datamine *discovery.Datamine
	datamine, err = discovery.GetDatamineByName(ctx, client, tableInfo.ID, datamineInput)
	if err != nil {
		// Try by ID
		datamine, err = discovery.GetDatamineByID(ctx, client, tableInfo.ID, datamineInput)
		if err != nil {
			return fmt.Errorf("failed to find Datamine: %w", err)
		}
	}

	fmt.Println(ui.SuccessStyle.Render(fmt.Sprintf("Found Datamine: %s (%s)", datamine.Name, datamine.ID)))

	// Fetch related resources for beautification
	var relatedResources []discovery.RelatedResource
	if beautify {
		fmt.Println(ui.InfoStyle.Render("Fetching related resources for beautification..."))
		relatedResources, err = discovery.FetchDatamineRelatedResources(ctx, client, datamine, tableInfo)
		if err != nil {
			return fmt.Errorf("failed to fetch related resources: %w", err)
		}
		fmt.Println(ui.SuccessStyle.Render(fmt.Sprintf("Found %d related resources", len(relatedResources))))
	}

	// Generate HCL from discovered data
	fmt.Println(ui.InfoStyle.Render("Generating Terraform configuration..."))
	content := export.GenerateDatamineHCL(datamine, tableInfo, relatedResources)

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
