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
	"github.com/elementumltd/elementum-cli/internal/client"
	"github.com/spf13/cobra"
)

var cloudlinksCmd = &cobra.Command{
	Use:   "cloudlinks",
	Short: "Manage cloudlinks",
	Long:  "Commands for listing and managing cloudlinks (data connectors) in your organization.",
}

var cloudlinksListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all cloudlinks",
	Long:  "Display a table of all cloudlinks (data connectors) in your organization.",
	RunE:  runCloudlinksList,
}

var cloudlinksFunctionsCmd = &cobra.Command{
	Use:   "functions <cloudlink-id-or-name>",
	Short: "List configured and available Snowflake functions on a cloudlink",
	Long: `Show both configured (stored) and available Snowflake functions for a cloudlink.

Configured functions are already set up in Elementum. Available functions are discovered
from the Snowflake database and can be configured as elementum_function resources.

Use --database and --schema to filter available functions by location.`,
	Args: cobra.ExactArgs(1),
	RunE: runCloudlinksFunctions,
}

var cloudlinksExploreCmd = &cobra.Command{
	Use:   "explore <cloudlink> [database] [schema]",
	Short: "Explore databases, schemas, and tables in a Snowflake cloudlink",
	Long: `Hierarchically explore a Snowflake cloudlink's structure.

With no extra arguments, lists databases.
With a database name, lists schemas in that database.
With database and schema, lists tables in that schema.

Use --database and --schema flags to jump directly to a level:
  --database ANALYTICS           Jump to schemas in ANALYTICS
  --database ANALYTICS --schema PUBLIC  Jump to tables

Examples:
  ei cloudlinks explore my-snowflake              # List databases
  ei cloudlinks explore my-snowflake ANALYTICS    # List schemas in ANALYTICS
  ei cloudlinks explore my-snowflake ANALYTICS PUBLIC  # List tables
  ei cloudlinks explore my-snowflake --database ANALYTICS --schema PUBLIC  # Same as above`,
	Args: cobra.RangeArgs(1, 3),
	RunE: runCloudlinksExplore,
}

var cloudlinksExportCmd = &cobra.Command{
	Use:   "export [name-or-id]",
	Short: "Export a CloudLink to Terraform",
	Long: `Export a CloudLink as Terraform configuration.

You can specify the CloudLink by:
- Name: Production API
- ID: cl_12345678

The command will generate an import block and run terraform to create the configuration.`,
	Args: cobra.ExactArgs(1),
	RunE: runCloudlinksExport,
}

func init() {
	cloudlinksCmd.AddCommand(cloudlinksListCmd)
	cloudlinksCmd.AddCommand(cloudlinksFunctionsCmd)
	cloudlinksCmd.AddCommand(cloudlinksExploreCmd)
	cloudlinksCmd.AddCommand(cloudlinksExportCmd)

	cloudlinksFunctionsCmd.Flags().String("type", "", "Filter by function type: procedure, udf")
	cloudlinksFunctionsCmd.Flags().String("database", "", "Filter available functions by database name")
	cloudlinksFunctionsCmd.Flags().String("schema", "", "Filter available functions by schema name")
	cloudlinksExploreCmd.Flags().String("database", "", "Filter to specific database")
	cloudlinksExploreCmd.Flags().String("schema", "", "Filter to specific schema (requires --database)")
	cloudlinksExportCmd.Flags().StringP("output", "o", "generated.tf", "Output file for generated Terraform configuration")
}

// GetCloudlinksCmd returns the cloudlinks command for registration
func GetCloudlinksCmd() *cobra.Command {
	return cloudlinksCmd
}

func runCloudlinksList(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Get authenticated client
	client, err := auth.GetClientFromCmd(cmd)
	if err != nil {
		return err
	}

	// Show spinner while loading (skip for JSON output)
	if !isJSONOutput(cmd) {
		fmt.Println(ui.InfoStyle.Render("Loading cloudlinks..."))
	}

	// Discover cloudlinks
	cloudlinks, err := discovery.ListCloudLinks(ctx, client)
	if err != nil {
		return fmt.Errorf("failed to list cloudlinks: %w", err)
	}

	// JSON output
	if isJSONOutput(cmd) {
		return outputJSON(cloudlinks)
	}

	if len(cloudlinks) == 0 {
		fmt.Println()
		fmt.Println(ui.WarningStyle.Render("No cloudlinks found in your organization."))
		fmt.Println()
		return nil
	}

	// Create table
	table := ui.NewTable([]string{"NAME", "TYPE", "SYSTEM", "VALID", "ID"})

	for _, cl := range cloudlinks {
		system := "No"
		if cl.System {
			system = "Yes"
		}
		valid := "No"
		if cl.Valid {
			valid = "Yes"
		}

		table.AddRow(
			cl.Name,
			cl.Type,
			system,
			valid,
			cl.ID,
		)
	}

	// Display table
	fmt.Println()
	fmt.Println(ui.TitleStyle.Render("CloudLinks"))
	fmt.Println()
	fmt.Println(table.Render())
	fmt.Println()
	fmt.Println(ui.MutedStyle.Render(fmt.Sprintf("Total: %d cloudlinks", len(cloudlinks))))
	fmt.Println()

	return nil
}

func runCloudlinksFunctions(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Get authenticated client
	c, err := auth.GetClientFromCmd(cmd)
	if err != nil {
		return err
	}

	cloudLinkRef := args[0]
	functionType, _ := cmd.Flags().GetString("type")
	database, _ := cmd.Flags().GetString("database")
	schema, _ := cmd.Flags().GetString("schema")

	// Resolve cloudlink name to ID
	cloudLinkID, err := discovery.ResolveCloudLinkID(ctx, c, cloudLinkRef)
	if err != nil {
		return err
	}

	// Show spinner while loading (skip for JSON output)
	if !isJSONOutput(cmd) {
		fmt.Println(ui.InfoStyle.Render("Loading functions..."))
	}

	// Discover both stored and available functions
	result, err := discovery.ListCloudLinkFunctions(ctx, c, cloudLinkID, database, schema, functionType)
	if err != nil {
		return fmt.Errorf("failed to list functions: %w", err)
	}

	// JSON output
	if isJSONOutput(cmd) {
		return outputJSON(result)
	}

	// Build set of configured function names for marking available functions
	configuredNames := make(map[string]bool)
	for _, sf := range result.StoredFunctions {
		configuredNames[sf.Name] = true
	}

	// Display configured functions section
	if len(result.StoredFunctions) > 0 {
		fmt.Println()
		fmt.Println(ui.TitleStyle.Render("Configured Functions"))
		fmt.Println()

		storedTable := ui.NewTable([]string{"DISPLAY NAME", "NAME", "ID"})
		for _, sf := range result.StoredFunctions {
			storedTable.AddRow(sf.DisplayName, sf.Name, sf.ID)
		}
		fmt.Println(storedTable.Render())
		fmt.Println()
		fmt.Println(ui.MutedStyle.Render(fmt.Sprintf("Total: %d configured functions", len(result.StoredFunctions))))
		fmt.Println()
	}

	// Display available functions section
	available := result.AvailableFunctions
	if len(available) == 0 {
		if len(result.StoredFunctions) == 0 {
			fmt.Println()
			fmt.Println(ui.WarningStyle.Render("No functions found on this cloudlink."))
			fmt.Println()
		} else {
			fmt.Println(ui.MutedStyle.Render("No additional available functions found."))
			fmt.Println()
		}
		return nil
	}

	fmt.Println()
	fmt.Println(ui.TitleStyle.Render(fmt.Sprintf("Available Functions on CloudLink: %s", cloudLinkRef)))
	fmt.Println()

	availTable := ui.NewTable([]string{"NAME", "TYPE", "DATABASE.SCHEMA", "ARGS", "RETURN", "CONFIGURED", "DESCRIPTION"})

	for _, fn := range available {
		location := ""
		if fn.DatabaseName != "" && fn.SchemaName != "" {
			location = fn.DatabaseName + "." + fn.SchemaName
		} else if fn.DatabaseName != "" {
			location = fn.DatabaseName
		} else if fn.SchemaName != "" {
			location = fn.SchemaName
		}

		desc := fn.Description
		if len(desc) > 40 {
			desc = desc[:37] + "..."
		}

		configured := ""
		if configuredNames[fn.Name] {
			configured = "Yes"
		}

		availTable.AddRow(
			fn.Name,
			fn.TypeDisplayName(),
			location,
			fn.ArgsDisplay(),
			fn.ReturnType,
			configured,
			desc,
		)
	}

	fmt.Println(availTable.Render())
	fmt.Println()
	fmt.Println(ui.MutedStyle.Render(fmt.Sprintf("Total: %d available functions", len(available))))
	fmt.Println()

	return nil
}

func runCloudlinksExplore(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Get authenticated client
	c, err := auth.GetClientFromCmd(cmd)
	if err != nil {
		return err
	}

	cloudLinkRef := args[0]
	dbFlag, _ := cmd.Flags().GetString("database")
	schemaFlag, _ := cmd.Flags().GetString("schema")

	// Determine database and schema from args or flags
	database := ""
	schema := ""

	if len(args) >= 2 {
		database = args[1]
	} else if dbFlag != "" {
		database = dbFlag
	}

	if len(args) >= 3 {
		schema = args[2]
	} else if schemaFlag != "" {
		schema = schemaFlag
	}

	// Resolve cloudlink name to ID
	cloudLinkID, err := discovery.ResolveCloudLinkID(ctx, c, cloudLinkRef)
	if err != nil {
		return err
	}

	// Decide what to show based on what's specified
	switch {
	case database == "":
		return showDatabases(ctx, c, cloudLinkID, cloudLinkRef, cmd)
	case schema == "":
		return showSchemas(ctx, c, cloudLinkID, cloudLinkRef, database, cmd)
	default:
		return showTables(ctx, c, cloudLinkID, cloudLinkRef, database, schema, cmd)
	}
}

func showDatabases(ctx context.Context, c *client.Client, cloudLinkID, cloudLinkRef string, cmd *cobra.Command) error {
	// Show spinner while loading (skip for JSON output)
	if !isJSONOutput(cmd) {
		fmt.Println(ui.InfoStyle.Render("Loading databases..."))
	}

	databases, err := discovery.ListSnowflakeDatabases(ctx, c, cloudLinkID)
	if err != nil {
		return fmt.Errorf("failed to list databases: %w", err)
	}

	// JSON output
	if isJSONOutput(cmd) {
		return outputJSON(databases)
	}

	if len(databases) == 0 {
		fmt.Println()
		fmt.Println(ui.WarningStyle.Render("No databases found on this cloudlink."))
		fmt.Println()
		return nil
	}

	// Create table
	table := ui.NewTable([]string{"NAME", "OWNER", "COMMENT"})

	for _, db := range databases {
		comment := db.Comment
		if len(comment) > 50 {
			comment = comment[:47] + "..."
		}
		table.AddRow(db.Name, db.Owner, comment)
	}

	// Display table
	fmt.Println()
	fmt.Println(ui.TitleStyle.Render(fmt.Sprintf("Databases on CloudLink: %s", cloudLinkRef)))
	fmt.Println()
	fmt.Println(table.Render())
	fmt.Println()
	fmt.Println(ui.MutedStyle.Render(fmt.Sprintf("Total: %d databases", len(databases))))
	fmt.Println()
	fmt.Println(ui.MutedStyle.Render(fmt.Sprintf("Tip: ei cloudlinks explore %s <DATABASE> to list schemas", cloudLinkRef)))
	fmt.Println()

	return nil
}

func showSchemas(ctx context.Context, c *client.Client, cloudLinkID, cloudLinkRef, database string, cmd *cobra.Command) error {
	// Show spinner while loading (skip for JSON output)
	if !isJSONOutput(cmd) {
		fmt.Println(ui.InfoStyle.Render(fmt.Sprintf("Loading schemas in %s...", database)))
	}

	schemas, err := discovery.ListSnowflakeSchemas(ctx, c, cloudLinkID, database)
	if err != nil {
		return fmt.Errorf("failed to list schemas: %w", err)
	}

	// JSON output
	if isJSONOutput(cmd) {
		return outputJSON(schemas)
	}

	if len(schemas) == 0 {
		fmt.Println()
		fmt.Println(ui.WarningStyle.Render(fmt.Sprintf("No schemas found in database %s.", database)))
		fmt.Println()
		return nil
	}

	// Create table
	table := ui.NewTable([]string{"NAME", "DATABASE", "OWNER", "COMMENT"})

	for _, s := range schemas {
		comment := s.Comment
		if len(comment) > 40 {
			comment = comment[:37] + "..."
		}
		table.AddRow(s.Name, s.DatabaseName, s.Owner, comment)
	}

	// Display table
	fmt.Println()
	fmt.Println(ui.TitleStyle.Render(fmt.Sprintf("Schemas in %s on CloudLink: %s", database, cloudLinkRef)))
	fmt.Println()
	fmt.Println(table.Render())
	fmt.Println()
	fmt.Println(ui.MutedStyle.Render(fmt.Sprintf("Total: %d schemas", len(schemas))))
	fmt.Println()
	fmt.Println(ui.MutedStyle.Render(fmt.Sprintf("Tip: ei cloudlinks explore %s %s <SCHEMA> to list tables", cloudLinkRef, database)))
	fmt.Println()

	return nil
}

func showTables(ctx context.Context, c *client.Client, cloudLinkID, cloudLinkRef, database, schema string, cmd *cobra.Command) error {
	// Show spinner while loading (skip for JSON output)
	if !isJSONOutput(cmd) {
		fmt.Println(ui.InfoStyle.Render(fmt.Sprintf("Loading tables in %s.%s...", database, schema)))
	}

	tables, err := discovery.ListSnowflakeTables(ctx, c, cloudLinkID, database, schema)
	if err != nil {
		return fmt.Errorf("failed to list tables: %w", err)
	}

	// JSON output
	if isJSONOutput(cmd) {
		return outputJSON(tables)
	}

	if len(tables) == 0 {
		fmt.Println()
		fmt.Println(ui.WarningStyle.Render(fmt.Sprintf("No tables found in %s.%s.", database, schema)))
		fmt.Println()
		return nil
	}

	// Create table
	table := ui.NewTable([]string{"NAME", "DATABASE", "SCHEMA", "TYPE", "ROWS", "SIZE"})

	for _, t := range tables {
		// Format size nicely
		size := formatBytes(t.Bytes)
		table.AddRow(t.Name, t.DatabaseName, t.SchemaName, t.Type, fmt.Sprintf("%d", t.Rows), size)
	}

	// Display table
	fmt.Println()
	fmt.Println(ui.TitleStyle.Render(fmt.Sprintf("Tables in %s.%s on CloudLink: %s", database, schema, cloudLinkRef)))
	fmt.Println()
	fmt.Println(table.Render())
	fmt.Println()
	fmt.Println(ui.MutedStyle.Render(fmt.Sprintf("Total: %d tables", len(tables))))
	fmt.Println()

	return nil
}

// formatBytes formats bytes into a human-readable string
func formatBytes(bytes int64) string {
	if bytes == 0 {
		return "-"
	}
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

func runCloudlinksExport(cmd *cobra.Command, args []string) error {
	input := args[0]
	ctx := context.Background()

	// Get flags
	outputFile, _ := cmd.Flags().GetString("output")

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

	fmt.Println(ui.InfoStyle.Render("Looking up CloudLink..."))

	// Try to get CloudLink by name first, then by ID
	var cloudlink *discovery.CloudLink
	cloudlink, err = discovery.GetCloudLinkByName(ctx, client, input)
	if err != nil {
		// Try by ID
		cloudlink, err = discovery.GetCloudLink(ctx, client, input)
		if err != nil {
			return fmt.Errorf("failed to find CloudLink: %w", err)
		}
	}

	fmt.Println(ui.SuccessStyle.Render(fmt.Sprintf("Found CloudLink: %s (%s)", cloudlink.Name, cloudlink.ID)))

	// Check if terraform is installed
	if err := export.CheckTerraformInstalled(); err != nil {
		return fmt.Errorf("terraform is required: %w", err)
	}

	// Generate import block
	blocks := []export.ImportBlock{
		{
			ID:           cloudlink.ID,
			ResourceType: "elementum_cloudlink",
			ResourceName: export.SanitizeName(cloudlink.Name),
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
