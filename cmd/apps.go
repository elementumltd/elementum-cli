// Copyright (c) ElementumAI, Inc.
// SPDX-License-Identifier: MPL-2.0

package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/elementumltd/elementum-cli/auth"
	"github.com/elementumltd/elementum-cli/discovery"
	"github.com/elementumltd/elementum-cli/export"
	"github.com/elementumltd/elementum-cli/logger"
	"github.com/elementumltd/elementum-cli/ui"
	"github.com/spf13/cobra"
)

var appsCmd = &cobra.Command{
	Use:     "apps",
	Aliases: []string{"app"},
	Short:   "Manage apps",
	Long:    "Commands for listing, creating, updating, deleting, and exporting Elementum apps.",
}

var appsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all apps",
	Long:  "Display a table of all apps in your organization.",
	RunE:  runAppsList,
}

var appsShowCmd = &cobra.Command{
	Use:   "show [namespace-or-url]",
	Short: "Show app details",
	Long: `Display a tree view of an app and all its resources.

You can specify the app by:
- Namespace: accountassignment
- Full URL: https://appdemo.elementum.io/app/accountassignment/records`,
	Args: cobra.ExactArgs(1),
	RunE: runAppsShow,
}

var appsExportCmd = &cobra.Command{
	Use:   "export [namespace-or-url]",
	Short: "Export an app and its resources to Terraform",
	Long: `Export an app and its resources as Terraform configuration.

You can specify the app by:
- Namespace: accountassignment
- Full URL: https://appdemo.elementum.io/app/accountassignment/records

The command will:
1. Discover all resources in the app
2. Generate import blocks
3. Run terraform init and terraform plan -generate-config-out
4. Output a ready-to-use Terraform configuration file`,
	Args: cobra.ExactArgs(1),
	RunE: runAppsExport,
}

func init() {
	// Add subcommands
	appsCmd.AddCommand(appsListCmd)
	appsCmd.AddCommand(appsShowCmd)
	appsCmd.AddCommand(appsExportCmd)
	appsCmd.AddCommand(appCreateCmd)
	appsCmd.AddCommand(appDeleteCmd)
	appsCmd.AddCommand(appUpdateCmd)

	// Export flags
	appsExportCmd.Flags().StringP("output", "o", "generated.tf", "Output file for generated Terraform configuration (single-file mode)")
	appsExportCmd.Flags().StringP("directory", "d", "", "Output directory for multi-file export (splits into app-*.tf, automation-*.tf, etc.)")
	appsExportCmd.Flags().Bool("all", true, "Export all resources without prompting")
	appsExportCmd.Flags().Bool("beautify", true, "Resolve UUIDs to references and strip null attributes")
	appsExportCmd.Flags().Bool("recursive", true, "Recursively discover and export related apps/elements through automations and relationships")
	appsExportCmd.Flags().Bool("lenient", false, "Continue export on non-critical errors (timeouts, permission issues) instead of aborting. Produces potentially incomplete output.")
	appsExportCmd.Flags().Int64("max-concurrency", int64(discovery.DefaultMaxConcurrency), "Maximum number of concurrent API calls (default 20, set to 1 for sequential mode)")
}

// GetAppsCmd returns the apps command for registration
func GetAppsCmd() *cobra.Command {
	return appsCmd
}

func runAppsList(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Get authenticated client
	client, err := auth.GetClientFromCmd(cmd)
	if err != nil {
		return err
	}

	// Show spinner while loading (skip for JSON output)
	if !isJSONOutput(cmd) {
		fmt.Println(ui.InfoStyle.Render("Loading apps..."))
	}

	// Discover objects and filter to apps only
	objects, err := discovery.ListObjects(ctx, client)
	if err != nil {
		return fmt.Errorf("failed to list apps: %w", err)
	}

	// Filter to apps only
	var apps []discovery.ObjectSummary
	for _, obj := range objects {
		if obj.Type == "App" {
			apps = append(apps, obj)
		}
	}

	// JSON output
	if isJSONOutput(cmd) {
		return outputJSON(apps)
	}

	if len(apps) == 0 {
		fmt.Println()
		fmt.Println(ui.WarningStyle.Render("No apps found in your organization."))
		fmt.Println()
		return nil
	}

	// Create table
	table := ui.NewTable([]string{"NAME", "NAMESPACE", "ID"})

	for _, app := range apps {
		table.AddRow(
			app.Name,
			app.Namespace,
			app.ID,
		)
	}

	// Display table
	fmt.Println()
	fmt.Println(ui.TitleStyle.Render("Apps"))
	fmt.Println()
	fmt.Println(table.Render())
	fmt.Println()
	fmt.Println(ui.MutedStyle.Render(fmt.Sprintf("Total: %d apps", len(apps))))
	fmt.Println()

	return nil
}

func runAppsShow(cmd *cobra.Command, args []string) error {
	input := args[0]
	ctx := context.Background()

	// Get authenticated client
	client, err := auth.GetClientFromCmd(cmd)
	if err != nil {
		return err
	}

	// Parse namespace from input (URL or namespace)
	namespace, err := discovery.ParseNamespaceFromURL(input)
	if err != nil {
		return fmt.Errorf("failed to parse app identifier: %w", err)
	}

	// Show spinner while loading (skip for JSON output)
	if !isJSONOutput(cmd) {
		fmt.Println(ui.InfoStyle.Render(fmt.Sprintf("Looking up app: %s", namespace)))
	}

	// Lookup app by namespace
	app, err := discovery.GetAppByNamespace(ctx, client, namespace)
	if err != nil {
		return fmt.Errorf("failed to find app: %w", err)
	}

	// JSON output
	if isJSONOutput(cmd) {
		return outputJSON(app)
	}

	// Build tree
	tree := ui.NewTree(fmt.Sprintf("%s (%s)", app.Name, app.ID))

	// Add fields
	if len(app.Fields) > 0 {
		fieldsNode := tree.Root.AddChildWithLabel("Fields", fmt.Sprintf("(%d)", len(app.Fields)))
		displayCount := 5
		for i, field := range app.Fields {
			if i >= displayCount && len(app.Fields) > displayCount+2 {
				fieldsNode.AddChildWithLabel(fmt.Sprintf("... %d more", len(app.Fields)-displayCount))
				break
			}
			meta := fmt.Sprintf("[%s]", field.Type)
			if field.Required {
				meta += " required"
			}
			if len(field.Options) > 0 {
				meta += fmt.Sprintf(" (%d options)", len(field.Options))
			}
			fieldsNode.AddChildWithLabel(field.Name, meta)
		}
	}

	// Add layouts
	if len(app.Layouts) > 0 {
		layoutsNode := tree.Root.AddChildWithLabel("Layouts", fmt.Sprintf("(%d)", len(app.Layouts)))
		for _, layout := range app.Layouts {
			layoutNode := layoutsNode.AddChildWithLabel(layout.Name)

			// Add display blocks for this layout
			if len(layout.DisplayBlocks) > 0 {
				blocksNode := layoutNode.AddChildWithLabel("Display Blocks", fmt.Sprintf("(%d)", len(layout.DisplayBlocks)))
				for _, block := range layout.DisplayBlocks {
					blockLabel := fmt.Sprintf("[%s]", block.Type)
					if block.Name != "" {
						blockLabel = fmt.Sprintf("%s - %s", block.Name, block.Type)
					}
					blocksNode.AddChildWithLabel(blockLabel)
				}
			}
		}
	}

	// Add flows
	if len(app.Flows) > 0 {
		flowsNode := tree.Root.AddChildWithLabel("Flows", fmt.Sprintf("(%d)", len(app.Flows)))
		for _, flow := range app.Flows {
			flowsNode.AddChildWithLabel(flow.Name)
		}
	}

	// Add automations
	if len(app.Automations) > 0 {
		automationsNode := tree.Root.AddChildWithLabel("Automations", fmt.Sprintf("(%d)", len(app.Automations)))
		for _, automation := range app.Automations {
			autoNode := automationsNode.AddChildWithLabel(automation.Name)

			// Show triggers
			if len(automation.Triggers) > 0 {
				for _, trigger := range automation.Triggers {
					autoNode.AddChildWithLabel(fmt.Sprintf("trigger: %s", trigger.Type))
				}
			}

			// Show task count
			if len(automation.Tasks) > 0 {
				autoNode.AddChildWithLabel(fmt.Sprintf("tasks: %d", len(automation.Tasks)))
			}
		}
	}

	// Add agents
	if len(app.Agents) > 0 {
		agentsNode := tree.Root.AddChildWithLabel("Agents", fmt.Sprintf("(%d)", len(app.Agents)))
		for _, agent := range app.Agents {
			agentNode := agentsNode.AddChildWithLabel(agent.Name)
			if len(agent.Tools) > 0 {
				agentNode.AddChildWithLabel(fmt.Sprintf("tools: %d", len(agent.Tools)))
			}
		}
	}

	// Add widgets
	if len(app.Widgets) > 0 {
		widgetsNode := tree.Root.AddChildWithLabel("Widgets", fmt.Sprintf("(%d)", len(app.Widgets)))
		for _, widget := range app.Widgets {
			widgetsNode.AddChildWithLabel(widget.Name)
		}
	}

	// Add AI File Readers
	if len(app.AIFileReaders) > 0 {
		fileReadersNode := tree.Root.AddChildWithLabel("AI File Readers", fmt.Sprintf("(%d)", len(app.AIFileReaders)))
		for _, reader := range app.AIFileReaders {
			fileReadersNode.AddChildWithLabel(reader.Name)
		}
	}

	// Add Approval Processes
	if len(app.Approvals) > 0 {
		approvalsNode := tree.Root.AddChildWithLabel("Approval Processes", fmt.Sprintf("(%d)", len(app.Approvals)))
		for _, approval := range app.Approvals {
			approvalsNode.AddChildWithLabel(approval.Name)
		}
	}

	// Display tree
	fmt.Println()
	fmt.Println(tree.Render())
	fmt.Println()

	// Display summary
	fmt.Println(ui.SubtitleStyle.Render("Summary"))
	fmt.Printf("  %s %d fields\n", ui.RenderBullet(), len(app.Fields))
	fmt.Printf("  %s %d layouts\n", ui.RenderBullet(), len(app.Layouts))
	fmt.Printf("  %s %d automations\n", ui.RenderBullet(), len(app.Automations))
	fmt.Printf("  %s %d agents\n", ui.RenderBullet(), len(app.Agents))
	if len(app.Flows) > 0 {
		fmt.Printf("  %s %d flows\n", ui.RenderBullet(), len(app.Flows))
	}
	if len(app.Widgets) > 0 {
		fmt.Printf("  %s %d widgets\n", ui.RenderBullet(), len(app.Widgets))
	}
	if len(app.AIFileReaders) > 0 {
		fmt.Printf("  %s %d AI file readers\n", ui.RenderBullet(), len(app.AIFileReaders))
	}
	if len(app.Approvals) > 0 {
		fmt.Printf("  %s %d approval processes\n", ui.RenderBullet(), len(app.Approvals))
	}
	fmt.Println()

	return nil
}

func runAppsExport(cmd *cobra.Command, args []string) error {
	input := args[0]
	ctx := context.Background()

	// Get flags
	outputFile, _ := cmd.Flags().GetString("output")
	outputDir, _ := cmd.Flags().GetString("directory")
	exportAll, _ := cmd.Flags().GetBool("all")
	beautify, _ := cmd.Flags().GetBool("beautify")
	recursive, _ := cmd.Flags().GetBool("recursive")
	lenient, _ := cmd.Flags().GetBool("lenient")
	maxConcurrency, _ := cmd.Flags().GetInt64("max-concurrency")

	// Multi-file mode if directory is specified
	multiFileMode := outputDir != ""

	// Create parallel config (nil means sequential mode, used when maxConcurrency <= 1)
	var parallelConfig *discovery.ParallelConfig
	useParallel := maxConcurrency > 1
	if useParallel {
		parallelConfig = &discovery.ParallelConfig{MaxConcurrency: maxConcurrency}
	}

	// Set GraphQL debug mode when log level is debug or trace
	level := logger.GetCurrentLevel()
	if level == logger.LevelDebug || level == logger.LevelTrace {
		os.Setenv("DEBUG_GRAPHQL", "1")
	}

	// Create error collector for structured error tracking
	ec := discovery.NewErrorCollector()

	// Get authenticated client
	client, err := auth.GetClientFromCmd(cmd)
	if err != nil {
		return err
	}

	// Parse namespace from input (URL or namespace)
	fmt.Println(ui.InfoStyle.Render("Parsing app identifier..."))
	namespace, err := discovery.ParseNamespaceFromURL(input)
	if err != nil {
		return fmt.Errorf("failed to parse app identifier: %w", err)
	}

	// Get credentials for provider config
	config, _ := auth.LoadConfig()
	creds, err := auth.GetCredentials(cmd, config)
	if err != nil {
		return err
	}

	// Show spinner while loading
	fmt.Println(ui.InfoStyle.Render(fmt.Sprintf("Looking up app by namespace: %s", namespace)))

	// Lookup app by namespace (use parallel version if enabled)
	var app *discovery.App
	if useParallel {
		app, err = discovery.GetAppByNamespaceParallel(ctx, client, namespace, parallelConfig, ec)
	} else {
		app, err = discovery.GetAppByNamespace(ctx, client, namespace, ec)
	}
	if err != nil {
		return fmt.Errorf("failed to find app: %w", err)
	}

	// Check for fatal errors after initial app discovery
	if ec.HasFatal() {
		if !lenient {
			fmt.Println(ui.ErrorStyle.Render("Export aborted due to errors during app discovery."))
			fmt.Println(ec.Summary())
			return fmt.Errorf("export aborted: fatal errors during discovery (use --lenient to continue anyway)")
		}
		fmt.Println(ui.WarningStyle.Render("Errors encountered during discovery, continuing in --lenient mode..."))
	}

	fmt.Println(ui.SuccessStyle.Render(fmt.Sprintf("Found app: %s (%s)", app.Name, app.ID)))
	logger.Debug("app discovery results", "fields", len(app.Fields), "layouts", len(app.Layouts), "flows", len(app.Flows), "agents", len(app.Agents), "widgets", len(app.Widgets))
	fmt.Println()

	// If recursive flag is set, discover all related apps/elements through automations and relationships
	if recursive {
		fmt.Println(ui.InfoStyle.Render("Discovering related apps and elements (recursive)..."))
		var uCtx *discovery.UnifiedDiscoveryContext
		if useParallel {
			uCtx, err = discovery.DiscoverAllDependenciesParallel(ctx, client, app.ID, "App", parallelConfig, ec)
		} else {
			uCtx, err = discovery.DiscoverAllDependencies(ctx, client, app.ID, "App", ec)
		}
		if err != nil {
			return fmt.Errorf("failed to discover dependencies: %w", err)
		}

		// Check for fatal errors after recursive discovery
		if ec.HasFatal() {
			if !lenient {
				fmt.Println(ui.ErrorStyle.Render("Export aborted due to errors during recursive discovery."))
				fmt.Println(ec.Summary())
				return fmt.Errorf("export aborted: fatal errors during recursive discovery (use --lenient to continue anyway)")
			}
			fmt.Println(ui.WarningStyle.Render("Errors encountered during recursive discovery, continuing in --lenient mode..."))
		}

		// Store discovered resources on the app
		for _, discoveredApp := range uCtx.Apps {
			app.DiscoveredApps = append(app.DiscoveredApps, discoveredApp)
		}
		for _, discoveredElement := range uCtx.Elements {
			logger.Debug("discovered element", "id", discoveredElement.ID, "name", discoveredElement.Name, "handle", discoveredElement.Handle)
			app.DiscoveredElements = append(app.DiscoveredElements, discoveredElement)
		}
		for _, discoveredTask := range uCtx.Tasks {
			logger.Debug("discovered task", "id", discoveredTask.ID, "name", discoveredTask.Name, "namespace", discoveredTask.Namespace)
			app.DiscoveredTasks = append(app.DiscoveredTasks, discoveredTask)
		}
		for tableID, discoveredTable := range uCtx.Tables {
			logger.Debug("discovered table", "mapKey", tableID, "tableID", discoveredTable.ID, "name", discoveredTable.Name, "type", discoveredTable.Type, "handle", discoveredTable.Handle)
			app.DiscoveredTables = append(app.DiscoveredTables, discoveredTable)
		}
		for _, discoveredDatamine := range uCtx.Datamines {
			logger.Debug("discovered datamine", "id", discoveredDatamine.ID, "name", discoveredDatamine.Name, "tableID", discoveredDatamine.TableID)
			app.DiscoveredDatamines = append(app.DiscoveredDatamines, discoveredDatamine)
		}
		for _, discoveredCloudLink := range uCtx.CloudLinks {
			logger.Debug("discovered cloudlink", "id", discoveredCloudLink.ID, "name", discoveredCloudLink.Name)
			app.DiscoveredCloudLinks = append(app.DiscoveredCloudLinks, discoveredCloudLink)
		}

		// Collect AI provider connectors from AI search tables (must be called AFTER discovered elements are merged)
		discovery.CollectSearchTableAiProviderConnectors(app)

		// Discover widget aspects from discovered apps/elements/tasks (must be called AFTER discovered resources are merged)
		// This picks up related objects referenced by widgets on discovered resources for UUID beautification
		if err := discovery.DiscoverWidgetAspects(ctx, client, app); err != nil {
			logger.Warn("failed to discover widget aspects from discovered resources", "error", err)
		}

		fmt.Println(ui.SuccessStyle.Render(fmt.Sprintf("Found %d related apps, %d elements, %d tasks, %d tables, %d datamines, %d cloudlinks",
			len(uCtx.Apps), len(uCtx.Elements), len(uCtx.Tasks), len(uCtx.Tables), len(uCtx.Datamines), len(uCtx.CloudLinks))))
		fmt.Println()
	}

	// Discover datamines and tables referenced by automations (skip if recursive already did this)
	if !recursive {
		fmt.Println(ui.InfoStyle.Render("Discovering datamine dependencies..."))
		automationCtx := discovery.NewAutomationDiscoveryContext()
		if err := discovery.DiscoverAutomationDependencies(ctx, client, app, automationCtx); err != nil {
			severity := discovery.ClassifyError(err)
			ec.Add(severity, "dependency_discovery", "datamine dependencies", err)
			if !lenient && severity == discovery.SeverityFatal {
				fmt.Println(ui.ErrorStyle.Render("Export aborted due to errors during dependency discovery."))
				fmt.Println(ec.Summary())
				return fmt.Errorf("export aborted: fatal errors during dependency discovery (use --lenient to continue anyway)")
			}
		}

		// Store discovered datamines and tables on the app for later use
		app.ReferencedDatamines = automationCtx.ReferencedDatamines
		app.ReferencedTables = automationCtx.ReferencedTables

		if len(automationCtx.ReferencedDatamines) > 0 || len(automationCtx.ReferencedTables) > 0 {
			fmt.Println(ui.SuccessStyle.Render(fmt.Sprintf("Found %d datamines and %d tables", len(automationCtx.ReferencedDatamines), len(automationCtx.ReferencedTables))))
		}
	}

	// Determine which resource types to export
	selectedTypes := map[string]bool{}

	if exportAll {
		// Export everything (except automations - not supported yet)
		if len(app.Fields) > 0 {
			selectedTypes["fields"] = true
		}
		if len(app.Layouts) > 0 {
			selectedTypes["layouts"] = true
		}
		if len(app.Flows) > 0 {
			selectedTypes["flows"] = true
		}
		// Only export published and active automations
		publishedAutomationCount := 0
		for _, auto := range app.Automations {
			if auto.HasPublished && auto.Status == "ACTIVE" {
				publishedAutomationCount++
			}
		}
		if publishedAutomationCount > 0 {
			selectedTypes["automations"] = true
		}
		// Check for agents on the app, discovered apps, or discovered tasks
		hasAgents := len(app.Agents) > 0
		if !hasAgents {
			for _, discoveredApp := range app.DiscoveredApps {
				if len(discoveredApp.Agents) > 0 {
					hasAgents = true
					break
				}
			}
		}
		if !hasAgents {
			for _, discoveredTask := range app.DiscoveredTasks {
				if len(discoveredTask.Agents) > 0 {
					hasAgents = true
					break
				}
			}
		}
		if hasAgents {
			selectedTypes["agents"] = true
		}
		// Check for widgets on the app, discovered apps, or discovered elements
		hasWidgets := len(app.Widgets) > 0
		if !hasWidgets {
			for _, discoveredApp := range app.DiscoveredApps {
				if len(discoveredApp.Widgets) > 0 {
					hasWidgets = true
					break
				}
			}
		}
		if !hasWidgets {
			for _, elem := range app.DiscoveredElements {
				if len(elem.Widgets) > 0 {
					hasWidgets = true
					break
				}
			}
		}
		if hasWidgets {
			selectedTypes["widgets"] = true
		}
		if len(app.AIFileReaders) > 0 {
			selectedTypes["ai_file_readers"] = true
		}
		if len(app.Approvals) > 0 {
			selectedTypes["approval_processes"] = true
		}
		if len(app.Relationships) > 0 {
			selectedTypes["relationships"] = true
		}
		if len(app.Views) > 0 {
			selectedTypes["views"] = true
		}
		if len(app.AccessPolicies) > 0 {
			selectedTypes["access_policies"] = true
		}
		if len(app.DiscoveredApps) > 0 {
			selectedTypes["discovered_apps"] = true
		}
		if len(app.DiscoveredElements) > 0 {
			selectedTypes["discovered_elements"] = true
		}
		if len(app.DiscoveredTasks) > 0 {
			selectedTypes["discovered_tasks"] = true
		}
		if len(app.DiscoveredTables) > 0 || len(app.ReferencedTables) > 0 {
			selectedTypes["tables"] = true
		}
		if len(app.DiscoveredDatamines) > 0 || len(app.ReferencedDatamines) > 0 {
			selectedTypes["datamines"] = true
		}
		if len(app.DiscoveredCloudLinks) > 0 {
			selectedTypes["cloudlinks"] = true
		}
	} else {
		// Interactive selection
		var options []huh.Option[string]
		if len(app.Fields) > 0 {
			options = append(options, huh.NewOption(fmt.Sprintf("Fields (%d)", len(app.Fields)), "fields"))
		}
		if len(app.Layouts) > 0 {
			options = append(options, huh.NewOption(fmt.Sprintf("Layouts (%d)", len(app.Layouts)), "layouts"))
		}
		if len(app.Flows) > 0 {
			options = append(options, huh.NewOption(fmt.Sprintf("Flows (%d)", len(app.Flows)), "flows"))
		}
		if len(app.Automations) > 0 {
			options = append(options, huh.NewOption(fmt.Sprintf("Automations (%d)", len(app.Automations)), "automations"))
		}
		if len(app.Agents) > 0 {
			options = append(options, huh.NewOption(fmt.Sprintf("Agents (%d)", len(app.Agents)), "agents"))
		}

		if len(options) == 0 {
			return fmt.Errorf("no exportable resources found in app")
		}

		var selected []string
		form := huh.NewForm(
			huh.NewGroup(
				huh.NewMultiSelect[string]().
					Title("Select resource types to export").
					Options(options...).
					Value(&selected),
			),
		)

		if err := form.Run(); err != nil {
			return fmt.Errorf("selection cancelled: %w", err)
		}

		for _, s := range selected {
			selectedTypes[s] = true
		}
	}

	if len(selectedTypes) == 0 {
		return fmt.Errorf("no resource types selected for export")
	}

	// Check if terraform is installed
	if err := export.CheckTerraformInstalled(); err != nil {
		return fmt.Errorf("terraform is required: %w", err)
	}

	// Generate import blocks using the export package's complete implementation
	blocks := export.GenerateImportBlocks(app, selectedTypes)

	fmt.Println(ui.SuccessStyle.Render(fmt.Sprintf("Generated %d import blocks", len(blocks))))

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

	fmt.Println(ui.SuccessStyle.Render("Generated import blocks"))

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

	// Multi-file mode: use IR pipeline to organize into separate files
	if multiFileMode {
		fmt.Println(ui.InfoStyle.Render("Organizing configuration into multiple files..."))

		// Create and run the IR pipeline
		pipeline := export.NewExportPipeline(app, blocks)
		if err := pipeline.AddTofuOutput(content); err != nil {
			return fmt.Errorf("failed to parse generated config: %w", err)
		}
		pipeline.AddCLIBlocks()

		// Execute multi-file export (IR transforms run during Execute)
		blockExport := pipeline.ExecuteMultiFile()
		multiExport := blockExport.ToMultiFileExport()

		// Add provider config
		multiExport.ProviderFile = export.RenderProviderConfig(creds.Organization, creds.Instance, creds.Environment, creds.ClientID, creds.ClientSecret)

		// Generate locals.tf (stage lookups, system field IDs, option lookups)
		appResourceName := export.AppResourceName(app)
		multiExport.LocalsFile = export.GenerateLocals(app, appResourceName, blocks)

		// Generate data.tf (category, cloudlink, and relationship data sources)
		// Append to existing DataSourcesFile (which may contain user/group data sources from access policies)
		var dataSources strings.Builder
		if multiExport.DataSourcesFile != "" {
			dataSources.WriteString(multiExport.DataSourcesFile)
			dataSources.WriteString("\n")
		}
		dataSources.WriteString(export.GenerateCategoryCloudLinkDataSources(app))
		dataSources.WriteString(export.GenerateRelationshipDataSources(app))
		if dataSources.Len() > 0 {
			multiExport.DataSourcesFile = dataSources.String()
		}

		// Apply post-processing to all files (flow injection, cloud mappings, etc.)
		if beautify {
			fmt.Println(ui.InfoStyle.Render("Beautifying configuration..."))
			uuidMap := pipeline.GetUUIDMap()
			multiExport.PostProcessAll(func(s string) string {
				return export.PostProcessHCLWithUUIDMap(s, blocks, app, uuidMap)
			})
			fmt.Println(ui.SuccessStyle.Render("Configuration beautified"))
		}

		// Write to directory
		if err := export.WriteMultipleFiles(multiExport, outputDir); err != nil {
			return fmt.Errorf("failed to write output files: %w", err)
		}

		fileCount := multiExport.GetFileCount()
		fmt.Println(ui.SuccessStyle.Render(fmt.Sprintf("Generated %d files in %s", fileCount, outputDir)))

		// Display any warnings/errors that occurred
		if len(ec.All()) > 0 {
			fmt.Println()
			fmt.Println(ec.Summary())
		}

		fmt.Println()
		fmt.Println(ui.SubtitleStyle.Render("Next steps:"))
		fmt.Printf("  %s Review files in %s\n", ui.RenderBullet(), outputDir)
		fmt.Printf("  %s cd %s && terraform plan\n", ui.RenderBullet(), outputDir)
		fmt.Println()

		return nil
	}

	// Single-file mode: use IR pipeline for beautification
	if beautify {
		fmt.Println(ui.InfoStyle.Render("Beautifying configuration (resolving UUIDs, stripping nulls)..."))

		// Create and run the IR pipeline for single-file output
		pipeline := export.NewExportPipeline(app, blocks)
		if err := pipeline.AddTofuOutput(content); err != nil {
			return fmt.Errorf("failed to parse generated config: %w", err)
		}
		pipeline.AddCLIBlocks()

		// Execute returns beautified HCL (runs IR transforms: null stripping, UUID resolution, etc.)
		content = pipeline.Execute()

		// Apply remaining string-based post-processing (flow injection, cloud mappings, etc.)
		content = export.PostProcessHCLWithUUIDMap(content, blocks, app, pipeline.GetUUIDMap())

		fmt.Println(ui.SuccessStyle.Render("Configuration beautified"))
	}

	// Write output
	err = os.WriteFile(outputFile, []byte(content), 0644)
	if err != nil {
		return fmt.Errorf("failed to write output file: %w", err)
	}
	fmt.Println(ui.SuccessStyle.Render(fmt.Sprintf("Generated %s", outputFile)))

	// Display any warnings/errors that occurred
	if len(ec.All()) > 0 {
		fmt.Println()
		fmt.Println(ec.Summary())
	}

	fmt.Println()
	fmt.Println(ui.SubtitleStyle.Render("Next steps:"))
	fmt.Printf("  %s Review %s\n", ui.RenderBullet(), outputFile)
	fmt.Printf("  %s Run: terraform plan\n", ui.RenderBullet())
	fmt.Println()

	return nil
}
