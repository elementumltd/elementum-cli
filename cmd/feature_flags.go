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

var featureFlagsCmd = &cobra.Command{
	Use:   "feature-flags",
	Short: "Manage feature flags",
	Long:  "Commands for listing and managing feature flags in your organization.",
}

var featureFlagsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all feature flags",
	Long:  "Display a table of all feature flags in your organization with their current state.",
	RunE:  runFeatureFlagsList,
}

func init() {
	featureFlagsCmd.AddCommand(featureFlagsListCmd)
}

// GetFeatureFlagsCmd returns the feature-flags command for registration
func GetFeatureFlagsCmd() *cobra.Command {
	return featureFlagsCmd
}

func runFeatureFlagsList(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Get authenticated client
	client, err := auth.GetClientFromCmd(cmd)
	if err != nil {
		return err
	}

	// Show spinner while loading (skip for JSON output)
	if !isJSONOutput(cmd) {
		fmt.Println(ui.InfoStyle.Render("Loading feature flags..."))
	}

	// Discover feature flags
	flags, err := discovery.ListFeatureFlags(ctx, client)
	if err != nil {
		return fmt.Errorf("failed to list feature flags: %w", err)
	}

	// JSON output
	if isJSONOutput(cmd) {
		return outputJSON(flags)
	}

	if len(flags) == 0 {
		fmt.Println()
		fmt.Println(ui.WarningStyle.Render("No feature flags found in your organization."))
		fmt.Println()
		return nil
	}

	// Create table
	table := ui.NewTable([]string{"FEATURE", "ENABLED"})

	for _, flag := range flags {
		enabled := "No"
		if flag.Enabled {
			enabled = "Yes"
		}
		table.AddRow(
			flag.Feature,
			enabled,
		)
	}

	// Display table
	fmt.Println()
	fmt.Println(ui.TitleStyle.Render("Feature Flags"))
	fmt.Println()
	fmt.Println(table.Render())
	fmt.Println()
	fmt.Println(ui.MutedStyle.Render(fmt.Sprintf("Total: %d feature flags", len(flags))))
	fmt.Println()

	return nil
}
