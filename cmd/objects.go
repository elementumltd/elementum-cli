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
	"github.com/spf13/cobra"
)

var objectsCmd = &cobra.Command{
	Use:   "objects",
	Short: "Manage objects",
	Long:  "Commands for listing and managing Elementum objects (apps, elements, and tasks).",
}

var objectsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all objects (apps, elements, and tasks)",
	Long:  "Display a table of all objects (apps, elements, and tasks) in your organization.",
	RunE:  runObjectsList,
}

var objectsShowCmd = &cobra.Command{
	Use:   "show <namespace>",
	Short: "Show detailed information about an object",
	Long:  "Display data source and configuration details for an app, element, or task.",
	Args:  cobra.ExactArgs(1),
	RunE:  runObjectsShow,
}

var objectsSearchCmd = &cobra.Command{
	Use:   "search [query]",
	Short: "Search for objects (apps, elements, tasks)",
	Long:  "Search for apps, elements, and tasks by name in your organization.",
	Args:  cobra.ExactArgs(1),
	RunE:  runObjectsSearch,
}

func init() {
	objectsCmd.AddCommand(objectsListCmd)
	objectsCmd.AddCommand(objectsShowCmd)
	objectsCmd.AddCommand(objectsSearchCmd)
}

// GetObjectsCmd returns the objects command for registration
func GetObjectsCmd() *cobra.Command {
	return objectsCmd
}

func runObjectsList(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Get authenticated client
	client, err := auth.GetClientFromCmd(cmd)
	if err != nil {
		return err
	}

	// Show spinner while loading (skip for JSON output)
	if !isJSONOutput(cmd) {
		fmt.Println(ui.InfoStyle.Render("⠿ Loading objects..."))
	}

	// Discover objects (apps and elements)
	objects, err := discovery.ListObjects(ctx, client)
	if err != nil {
		return fmt.Errorf("failed to list objects: %w", err)
	}

	// JSON output
	if isJSONOutput(cmd) {
		return outputJSON(objects)
	}

	if len(objects) == 0 {
		fmt.Println()
		fmt.Println(ui.WarningStyle.Render("No objects found in your organization."))
		fmt.Println()
		return nil
	}

	// Create table
	table := ui.NewTable([]string{"TYPE", "NAME", "NAMESPACE"})

	for _, obj := range objects {
		table.AddRow(
			obj.Type,
			obj.Name,
			obj.Namespace,
		)
	}

	// Display table
	fmt.Println()
	fmt.Println(ui.TitleStyle.Render("📦 Objects"))
	fmt.Println()
	fmt.Println(table.Render())
	fmt.Println()
	fmt.Println(ui.MutedStyle.Render(fmt.Sprintf("Total: %d objects", len(objects))))
	fmt.Println()

	return nil
}

func runObjectsShow(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	namespace := args[0]

	// Get authenticated client
	client, err := auth.GetClientFromCmd(cmd)
	if err != nil {
		return err
	}

	// Get object details
	details, err := discovery.GetObjectDetails(ctx, client, namespace)
	if err != nil {
		return fmt.Errorf("failed to get object details: %w", err)
	}

	// JSON output
	if isJSONOutput(cmd) {
		return outputJSON(details)
	}

	// Display tree format
	fmt.Println()
	fmt.Println(ui.TitleStyle.Render(fmt.Sprintf("%s (%s)", details.Name, details.ID)))
	fmt.Println()

	// Build tree output
	var lines []string
	lines = append(lines, fmt.Sprintf("├── Type: %s", details.Type))
	lines = append(lines, fmt.Sprintf("├── Namespace: %s", details.Namespace))
	if details.Handle != "" {
		lines = append(lines, fmt.Sprintf("├── Handle: %s", details.Handle))
	}

	// Data Source section
	lines = append(lines, "├── Data Source")
	if details.SourceType != "" {
		lines = append(lines, fmt.Sprintf("│   ├── Source Type: %s", details.SourceType))
	}
	if details.StorageType != "" {
		lines = append(lines, fmt.Sprintf("│   ├── Storage Type: %s", details.StorageType))
	}
	if details.CloudLinkName != "" {
		cloudLinkType := details.CloudLinkType
		if cloudLinkType != "" {
			cloudLinkType = strings.TrimPrefix(cloudLinkType, "CloudLink")
			lines = append(lines, fmt.Sprintf("│   └── Cloud Link: %s (%s)", details.CloudLinkName, cloudLinkType))
		} else {
			lines = append(lines, fmt.Sprintf("│   └── Cloud Link: %s", details.CloudLinkName))
		}
	} else {
		lines = append(lines, "│   └── Cloud Link: (none)")
	}

	// Snowflake Connection section (only if external table)
	if details.DatabaseName != "" || details.SchemaName != "" || details.TableName != "" {
		lines = append(lines, "├── Snowflake Connection")
		if details.DatabaseName != "" {
			lines = append(lines, fmt.Sprintf("│   ├── Database: %s", details.DatabaseName))
		}
		if details.SchemaName != "" {
			lines = append(lines, fmt.Sprintf("│   ├── Schema: %s", details.SchemaName))
		}
		if details.TableName != "" {
			lines = append(lines, fmt.Sprintf("│   ├── Table: %s", details.TableName))
		}
		lines = append(lines, fmt.Sprintf("│   └── Read Only: %t", details.ReadOnly))
	}

	// Category
	if details.CategoryName != "" {
		lines = append(lines, fmt.Sprintf("├── Category: %s", details.CategoryName))
	} else {
		lines = append(lines, "├── Category: (none)")
	}

	// Description (last item)
	if details.Description != "" {
		lines = append(lines, fmt.Sprintf("└── Description: %s", details.Description))
	} else {
		lines = append(lines, "└── Description: (none)")
	}

	for _, line := range lines {
		fmt.Println(line)
	}
	fmt.Println()

	return nil
}

func runObjectsSearch(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	query := args[0]

	// Get authenticated client
	client, err := auth.GetClientFromCmd(cmd)
	if err != nil {
		return err
	}

	// Show spinner while searching (skip for JSON output)
	if !isJSONOutput(cmd) {
		fmt.Printf("%s %s...\n", ui.InfoStyle.Render("Searching for"), ui.InfoStyle.Render(fmt.Sprintf("%q", query)))
	}

	// Search for objects
	objects, err := discovery.SearchObjects(ctx, client, query)
	if err != nil {
		return fmt.Errorf("search failed: %w", err)
	}

	// JSON output
	if isJSONOutput(cmd) {
		return outputJSON(objects)
	}

	if len(objects) == 0 {
		fmt.Println()
		fmt.Println(ui.WarningStyle.Render(fmt.Sprintf("No objects found matching %q.", query)))
		fmt.Println()
		return nil
	}

	// Create table
	table := ui.NewTable([]string{"TYPE", "NAME", "NAMESPACE"})

	for _, obj := range objects {
		table.AddRow(
			obj.Type,
			obj.Name,
			obj.Namespace,
		)
	}

	// Display table
	fmt.Println()
	fmt.Println(ui.TitleStyle.Render(fmt.Sprintf("Search Results for %q", query)))
	fmt.Println()
	fmt.Println(table.Render())
	fmt.Println()
	fmt.Println(ui.MutedStyle.Render(fmt.Sprintf("Found %d matching objects", len(objects))))
	fmt.Println()

	return nil
}
