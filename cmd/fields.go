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

var fieldsCmd = &cobra.Command{
	Use:     "fields",
	Aliases: []string{"field"},
	Short:   "Field operations",
	Long:    "Commands for working with Elementum fields.",
}

var fieldsValuesCmd = &cobra.Command{
	Use:   "values <namespace> <field-name>",
	Short: "List values for a dropdown/picklist field",
	Long: `Fetch and display available values for a dropdown or picklist field.

For static picklists, displays all available options with their IDs and labels.
For dynamic picklists, shows the related aspect and suggests searching for values.

Examples:
  ei fields values support-tickets Status
  ei fields values support-tickets Priority
  ei fields values luma Stage --json`,
	Args: cobra.ExactArgs(2),
	RunE: runFieldsValues,
}

func init() {
	fieldsCmd.AddCommand(fieldsValuesCmd)
	fieldsCmd.AddCommand(fieldListCmd)
	fieldsCmd.AddCommand(fieldCreateCmd)
	fieldsCmd.AddCommand(fieldDeleteCmd)
	fieldsCmd.AddCommand(fieldUpdateCmd)
}

// GetFieldsCmd returns the fields command for registration
func GetFieldsCmd() *cobra.Command {
	return fieldsCmd
}

func runFieldsValues(cmd *cobra.Command, args []string) error {
	namespace := args[0]
	fieldName := args[1]
	ctx := context.Background()

	// Get authenticated client
	c, err := auth.GetClientFromCmd(cmd)
	if err != nil {
		return err
	}

	// Show loading message (skip for JSON output)
	if !isJSONOutput(cmd) {
		fmt.Println(ui.InfoStyle.Render(fmt.Sprintf("Looking up field %q in %s...", fieldName, namespace)))
	}

	// Resolve namespace to aspect ID
	aspectID, _, err := resolveAspectByNamespace(ctx, c, namespace)
	if err != nil {
		return fmt.Errorf("failed to resolve namespace %q: %w", namespace, err)
	}

	// Fetch field info with values
	fieldInfo, err := discovery.GetFieldWithValues(ctx, c, aspectID, fieldName)
	if err != nil {
		return fmt.Errorf("failed to fetch field: %w", err)
	}

	// JSON output
	if isJSONOutput(cmd) {
		return outputJSON(fieldInfo)
	}

	// Display field info
	fmt.Println()
	fmt.Println(ui.TitleStyle.Render(fieldInfo.Name))
	fmt.Println()

	// Show field type info
	fmt.Printf("  %s Field ID: %s\n", ui.RenderBullet(), fieldInfo.ID)
	fmt.Printf("  %s Type: %s\n", ui.RenderBullet(), fieldInfo.Type)
	fmt.Printf("  %s Config: %s\n", ui.RenderBullet(), fieldInfo.ConfigType)

	// For dynamic/API picklists, show info
	if fieldInfo.ConfigType == "dynamic" || fieldInfo.ConfigType == "api" || fieldInfo.ConfigType == "dynamic_or_api" {
		fmt.Println()
		fmt.Println(ui.WarningStyle.Render("This field's values come from a dynamic source (another aspect or API)."))
		fmt.Println(ui.MutedStyle.Render("Values cannot be listed directly. Check the Elementum UI or the source aspect."))
		fmt.Println()
		return nil
	}

	// For static picklists, show values
	if len(fieldInfo.Values) == 0 {
		fmt.Println()
		fmt.Println(ui.WarningStyle.Render("No values found for this field."))
		fmt.Println()
		return nil
	}

	fmt.Println()
	fmt.Println(ui.SubtitleStyle.Render("Available Values:"))
	fmt.Println()

	// Build table
	table := ui.NewTable([]string{"LABEL", "ID", "ACTIVE"})
	for _, v := range fieldInfo.Values {
		active := "yes"
		if !v.Active {
			active = "no"
		}
		table.AddRow(v.Label, v.ID, active)
	}
	fmt.Println(table.Render())
	fmt.Println()

	// Usage tip
	fmt.Println(ui.MutedStyle.Render("Usage with ei records create:"))
	if len(fieldInfo.Values) > 0 {
		fmt.Printf("  ei records create %s -f \"%s=%s\"\n", namespace, fieldInfo.Name, fieldInfo.Values[0].Label)
	}
	fmt.Println()

	return nil
}
