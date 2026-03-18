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

var elementsCmd = &cobra.Command{
	Use:   "elements",
	Short: "Manage elements",
	Long:  "Commands for listing and exporting Elementum elements (reference data objects).",
}

var elementsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all elements",
	Long:  "Display a table of all elements (reference data objects) in your organization.",
	RunE:  runElementsList,
}

var elementsExportCmd = &cobra.Command{
	Use:   "export [namespace-or-id]",
	Short: "Export an element to Terraform",
	Long: `Export an element as Terraform configuration.

You can specify the element by:
- Namespace: products
- ID: uuid

The command will generate an import block and run terraform to create the configuration.`,
	Args: cobra.ExactArgs(1),
	RunE: runElementsExport,
}

func init() {
	elementsCmd.AddCommand(elementsListCmd)
	elementsCmd.AddCommand(elementsExportCmd)

	elementsExportCmd.Flags().StringP("output", "o", "generated.tf", "Output file for generated Terraform configuration")
	elementsExportCmd.Flags().BoolP("verbose", "v", false, "Enable verbose debug output")
}

// GetElementsCmd returns the elements command for registration
func GetElementsCmd() *cobra.Command {
	return elementsCmd
}

func runElementsList(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Get authenticated client
	client, err := auth.GetClientFromCmd(cmd)
	if err != nil {
		return err
	}

	// Show spinner while loading (skip for JSON output)
	if !isJSONOutput(cmd) {
		fmt.Println(ui.InfoStyle.Render("Loading elements..."))
	}

	// Discover objects and filter to elements only
	objects, err := discovery.ListObjects(ctx, client)
	if err != nil {
		return fmt.Errorf("failed to list elements: %w", err)
	}

	// Filter to elements only
	var elements []discovery.ObjectSummary
	for _, obj := range objects {
		if obj.Type == "Element" {
			elements = append(elements, obj)
		}
	}

	// JSON output
	if isJSONOutput(cmd) {
		return outputJSON(elements)
	}

	if len(elements) == 0 {
		fmt.Println()
		fmt.Println(ui.WarningStyle.Render("No elements found in your organization."))
		fmt.Println()
		return nil
	}

	// Create table
	table := ui.NewTable([]string{"NAME", "NAMESPACE", "ID"})

	for _, elem := range elements {
		table.AddRow(
			elem.Name,
			elem.Namespace,
			elem.ID,
		)
	}

	// Display table
	fmt.Println()
	fmt.Println(ui.TitleStyle.Render("Elements"))
	fmt.Println()
	fmt.Println(table.Render())
	fmt.Println()
	fmt.Println(ui.MutedStyle.Render(fmt.Sprintf("Total: %d elements", len(elements))))
	fmt.Println()

	return nil
}

func runElementsExport(cmd *cobra.Command, args []string) error {
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

	fmt.Println(ui.InfoStyle.Render("Looking up Element..."))

	// Resolve element by namespace first, then by ID
	element, err := discovery.GetElementByNamespace(ctx, client, input)
	if err != nil {
		// Try by ID
		element, err = discovery.GetElement(ctx, client, input)
		if err != nil {
			return fmt.Errorf("failed to find Element: %w", err)
		}
	}

	fmt.Println(ui.SuccessStyle.Render(fmt.Sprintf("Found Element: %s (%s)", element.Name, element.ID)))

	// Check if terraform is installed
	if err := export.CheckTerraformInstalled(); err != nil {
		return fmt.Errorf("terraform is required: %w", err)
	}

	// Generate import block
	blocks := []export.ImportBlock{
		{
			ID:           element.ID,
			ResourceType: "elementum_element",
			ResourceName: export.SanitizeName(element.Name),
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
