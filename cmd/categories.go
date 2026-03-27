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

	"github.com/elementumltd/elementum-cli/auth"
	"github.com/elementumltd/elementum-cli/discovery"
	"github.com/elementumltd/elementum-cli/ui"
	"github.com/spf13/cobra"
)

var categoriesCmd = &cobra.Command{
	Use:   "categories",
	Short: "Manage categories",
	Long:  "Commands for listing and managing Elementum categories. Categories are used to organize Apps, Elements, and Tasks.",
}

var categoriesListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all categories",
	Long:  "Display a table of all categories in your organization. Categories are used to organize Apps, Elements, and Tasks.",
	RunE:  runCategoriesList,
}

func init() {
	categoriesCmd.AddCommand(categoriesListCmd)
}

// GetCategoriesCmd returns the categories command for registration
func GetCategoriesCmd() *cobra.Command {
	return categoriesCmd
}

func runCategoriesList(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Get authenticated client
	client, err := auth.GetClientFromCmd(cmd)
	if err != nil {
		return err
	}

	// Show spinner while loading (skip for JSON output)
	if !isJSONOutput(cmd) {
		fmt.Println(ui.InfoStyle.Render("⠿ Loading categories..."))
	}

	// Discover categories
	categories, err := discovery.ListCategories(ctx, client)
	if err != nil {
		return fmt.Errorf("failed to list categories: %w", err)
	}

	// JSON output
	if isJSONOutput(cmd) {
		return outputJSON(categories)
	}

	if len(categories) == 0 {
		fmt.Println()
		fmt.Println(ui.WarningStyle.Render("No categories found in your organization."))
		fmt.Println()
		return nil
	}

	// Create table
	table := ui.NewTable([]string{"NAME", "ID"})

	for _, cat := range categories {
		table.AddRow(
			cat.Name,
			cat.ID,
		)
	}

	// Display table
	fmt.Println()
	fmt.Println(ui.TitleStyle.Render("📁 Categories"))
	fmt.Println()
	fmt.Println(table.Render())
	fmt.Println()
	fmt.Println(ui.MutedStyle.Render(fmt.Sprintf("Total: %d categories", len(categories))))
	fmt.Println()

	return nil
}
