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
	"github.com/elementumltd/elementum-cli/internal/client"
	"github.com/elementumltd/elementum-cli/logger"
	"github.com/elementumltd/elementum-cli/ui"
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
	Use:   "functions <cloudlink-id-or-name> --database <DB> --schema <SCHEMA>",
	Short: "List configured and available Snowflake functions on a cloudlink",
	Long: `Show both configured (stored) and available Snowflake functions for a cloudlink.

Configured functions are already set up in Elementum. Available functions are discovered
from the Snowflake database and can be configured as elementum_function resources.

The --database and --schema flags are required to scope the Snowflake query and avoid timeouts.`,
	Args: cobra.ExactArgs(1),
	RunE: runCloudlinksFunctions,
}

var cloudlinksExploreCmd = &cobra.Command{
	Use:   "explore <cloudlink> [database] [schema] [table]",
	Short: "Explore databases, schemas, tables, and columns in a Snowflake cloudlink",
	Long: `Hierarchically explore a Snowflake cloudlink's structure.

With no extra arguments, lists databases.
With a database name, lists schemas in that database.
With database and schema, lists tables in that schema.
With database, schema, and table, lists columns in that table.

Use flags to jump directly to a level:
  --database ANALYTICS                        Jump to schemas in ANALYTICS
  --database ANALYTICS --schema PUBLIC        Jump to tables
  --database DB --schema SCH --table TBL      Jump to columns

Examples:
  ei cloudlinks explore my-snowflake              # List databases
  ei cloudlinks explore my-snowflake ANALYTICS    # List schemas in ANALYTICS
  ei cloudlinks explore my-snowflake ANALYTICS PUBLIC  # List tables
  ei cloudlinks explore my-snowflake ANALYTICS PUBLIC MY_TABLE  # List columns
  ei cloudlinks explore my-snowflake --database DB --schema SCH --table TBL  # Same as above`,
	Args: cobra.RangeArgs(1, 4),
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

var cloudlinksSearchServicesCmd = &cobra.Command{
	Use:   "search-services <cloudlink-or-ai-provider> --database <DB> --schema <SCHEMA>",
	Short: "List Cortex Search Services on a Snowflake cloudlink or AI provider",
	Long: `List available Snowflake Cortex Search Services that can be linked via elementum_linked_ai_search_table.

This command discovers existing Cortex Search Services in your Snowflake account that can be
connected to Elementum. Use the service name, database, and schema in your Terraform configuration.

You can specify either:
- A CloudLink name or ID
- A Snowflake AI provider name

The --database and --schema flags are required to scope the query.

Examples:
  ei cloudlinks search-services my-snowflake --database ANALYTICS --schema PUBLIC
  ei cloudlinks search-services "Snowflake" --database ELEMENTUM --schema PUBLIC`,
	Args: cobra.ExactArgs(1),
	RunE: runCloudlinksSearchServices,
}

func init() {
	cloudlinksCmd.AddCommand(cloudlinksListCmd)
	cloudlinksCmd.AddCommand(cloudlinksFunctionsCmd)
	cloudlinksCmd.AddCommand(cloudlinksExploreCmd)
	cloudlinksCmd.AddCommand(cloudlinksExportCmd)
	cloudlinksCmd.AddCommand(cloudlinksSearchServicesCmd)

	cloudlinksFunctionsCmd.Flags().String("type", "", "Filter by function type: procedure, udf")
	cloudlinksFunctionsCmd.Flags().String("database", "", "Filter available functions by database name")
	cloudlinksFunctionsCmd.Flags().String("schema", "", "Filter available functions by schema name")
	cloudlinksExploreCmd.Flags().String("database", "", "Filter to specific database")
	cloudlinksExploreCmd.Flags().String("schema", "", "Filter to specific schema (requires --database)")
	cloudlinksExploreCmd.Flags().String("table", "", "Filter to specific table (requires --database and --schema)")
	cloudlinksExportCmd.Flags().StringP("output", "o", "generated.tf", "Output file for generated Terraform configuration")
	cloudlinksSearchServicesCmd.Flags().String("database", "", "Filter by database name (required)")
	cloudlinksSearchServicesCmd.Flags().String("schema", "", "Filter by schema name (required)")
	_ = cloudlinksSearchServicesCmd.MarkFlagRequired("database")
	_ = cloudlinksSearchServicesCmd.MarkFlagRequired("schema")
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

	cloudLinkRef := args[0]
	functionType, _ := cmd.Flags().GetString("type")
	database, _ := cmd.Flags().GetString("database")
	schema, _ := cmd.Flags().GetString("schema")

	// Require both database and schema to avoid slow unfiltered queries
	if database == "" || schema == "" {
		return fmt.Errorf("--database and --schema are required to avoid Snowflake query timeouts\n\nExample:\n  ei cloudlinks functions %s --database MYDB --schema MYSCHEMA", cloudLinkRef)
	}

	// Get authenticated client
	c, err := auth.GetClientFromCmd(cmd)
	if err != nil {
		return err
	}

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
			fmt.Println(ui.WarningStyle.Render(fmt.Sprintf("No functions found in %s.%s", database, schema)))
			fmt.Println()
		} else {
			fmt.Println(ui.MutedStyle.Render(fmt.Sprintf("No additional available functions found in %s.%s", database, schema)))
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
	tableFlag, _ := cmd.Flags().GetString("table")

	// Determine database, schema, and table from args or flags
	database := ""
	schema := ""
	table := ""

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

	if len(args) >= 4 {
		table = args[3]
	} else if tableFlag != "" {
		table = tableFlag
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
	case table == "":
		return showTables(ctx, c, cloudLinkID, cloudLinkRef, database, schema, cmd)
	default:
		return showColumns(ctx, c, cloudLinkID, cloudLinkRef, database, schema, table, cmd)
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
	fmt.Println(ui.MutedStyle.Render(fmt.Sprintf("Tip: ei cloudlinks explore %s %s %s <TABLE> to list columns", cloudLinkRef, database, schema)))
	fmt.Println()

	return nil
}

func showColumns(ctx context.Context, c *client.Client, cloudLinkID, cloudLinkRef, database, schema, tableName string, cmd *cobra.Command) error {
	// Show spinner while loading (skip for JSON output)
	if !isJSONOutput(cmd) {
		fmt.Println(ui.InfoStyle.Render(fmt.Sprintf("Loading columns in %s.%s.%s...", database, schema, tableName)))
	}

	tableSchema, err := discovery.GetSnowflakeTableSchema(ctx, c, cloudLinkID, database, schema, tableName)
	if err != nil {
		return fmt.Errorf("failed to get table schema: %w", err)
	}

	// JSON output
	if isJSONOutput(cmd) {
		return outputJSON(tableSchema.Columns)
	}

	if len(tableSchema.Columns) == 0 {
		fmt.Println()
		fmt.Println(ui.WarningStyle.Render(fmt.Sprintf("No columns found in %s.%s.%s.", database, schema, tableName)))
		fmt.Println()
		return nil
	}

	// Create table
	table := ui.NewTable([]string{"NAME", "TYPE", "NULLABLE", "PRIMARY KEY", "COMMENT"})

	for _, col := range tableSchema.Columns {
		nullable := "No"
		if col.Nullable {
			nullable = "Yes"
		}
		pk := ""
		if col.PrimaryKey {
			pk = "Yes"
		}
		comment := col.Comment
		if len(comment) > 40 {
			comment = comment[:37] + "..."
		}
		table.AddRow(col.Name, col.DatabaseType, nullable, pk, comment)
	}

	// Display table
	fmt.Println()
	fmt.Println(ui.TitleStyle.Render(fmt.Sprintf("Columns in %s.%s.%s on CloudLink: %s", database, schema, tableName, cloudLinkRef)))
	fmt.Println()
	fmt.Println(table.Render())
	fmt.Println()
	fmt.Println(ui.MutedStyle.Render(fmt.Sprintf("Total: %d columns", len(tableSchema.Columns))))
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
		os.Setenv("DEBUG_GRAPHQL", "1")
	}

	// Get authenticated client
	client, err := auth.GetClientFromCmd(cmd)
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

	// Generate HCL from discovered data
	fmt.Println(ui.InfoStyle.Render("Generating Terraform configuration..."))
	content := export.GenerateCloudLinkHCL(cloudlink)

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

func runCloudlinksSearchServices(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	providerRef := args[0]
	database, _ := cmd.Flags().GetString("database")
	schema, _ := cmd.Flags().GetString("schema")

	// Get authenticated client
	c, err := auth.GetClientFromCmd(cmd)
	if err != nil {
		return err
	}

	// Show spinner while loading (skip for JSON output)
	if !isJSONOutput(cmd) {
		fmt.Println(ui.InfoStyle.Render(fmt.Sprintf("Loading Cortex Search Services from %s.%s...", database, schema)))
	}

	// Discover search services (providerRef can be cloudlink ID/name or AI provider name)
	services, err := discovery.ListSnowflakeCortexSearchServices(ctx, c, providerRef, database, schema)
	if err != nil {
		return fmt.Errorf("failed to list search services: %w", err)
	}

	// JSON output
	if isJSONOutput(cmd) {
		return outputJSON(services)
	}

	if len(services) == 0 {
		fmt.Println()
		fmt.Println(ui.WarningStyle.Render(fmt.Sprintf("No Cortex Search Services found in %s.%s", database, schema)))
		fmt.Println()
		return nil
	}

	// Create table
	table := ui.NewTable([]string{"SERVICE NAME", "DATABASE", "SCHEMA"})

	for _, svc := range services {
		table.AddRow(svc.Name, svc.Database, svc.Schema)
	}

	// Display table
	fmt.Println()
	fmt.Println(ui.TitleStyle.Render(fmt.Sprintf("Cortex Search Services (%s.%s)", database, schema)))
	fmt.Println(ui.MutedStyle.Render(fmt.Sprintf("Provider: %s", providerRef)))
	fmt.Println()
	fmt.Println(table.Render())
	fmt.Println()
	fmt.Println(ui.MutedStyle.Render(fmt.Sprintf("Total: %d services", len(services))))
	fmt.Println()
	fmt.Println(ui.SubtitleStyle.Render("Usage:"))
	fmt.Println("  Link a service with elementum_linked_ai_search_table:")
	fmt.Printf("  resource \"elementum_linked_ai_search_table\" \"example\" {\n")
	fmt.Printf("    object_id     = data.elementum_app.my_app.id\n")
	fmt.Printf("    cloud_link_id = data.elementum_cloudlink.%s.id\n", export.SanitizeName(providerRef))
	fmt.Printf("    database      = \"%s\"\n", database)
	fmt.Printf("    schema_name   = \"%s\"\n", schema)
	fmt.Printf("    service_name  = \"<SERVICE_NAME>\"\n")
	fmt.Printf("  }\n")
	fmt.Println()

	return nil
}
