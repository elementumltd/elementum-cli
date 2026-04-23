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

	// Generate HCL from discovered data
	fmt.Println(ui.InfoStyle.Render("Generating Terraform configuration..."))
	content := export.GenerateElementHCL(element)

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
