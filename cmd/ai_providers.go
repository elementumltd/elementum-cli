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

var aiProvidersCmd = &cobra.Command{
	Use:   "ai-providers",
	Short: "Manage AI providers",
	Long:  "Commands for listing and managing AI providers available in your organization.",
}

var aiProvidersListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all AI providers",
	Long:  "Display a table of all AI providers available in your organization with their available models and features.",
	RunE:  runAiProvidersList,
}

func init() {
	aiProvidersCmd.AddCommand(aiProvidersListCmd)
}

// GetAiProvidersCmd returns the ai-providers command for registration
func GetAiProvidersCmd() *cobra.Command {
	return aiProvidersCmd
}

func runAiProvidersList(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Get authenticated client
	client, err := auth.GetClientFromCmd(cmd)
	if err != nil {
		return err
	}

	// Show spinner while loading (skip for JSON output)
	if !isJSONOutput(cmd) {
		fmt.Println(ui.InfoStyle.Render("⠿ Loading AI providers..."))
	}

	// Discover AI providers
	providers, err := discovery.ListAiProviders(ctx, client)
	if err != nil {
		return fmt.Errorf("failed to list AI providers: %w", err)
	}

	// JSON output
	if isJSONOutput(cmd) {
		return outputJSON(providers)
	}

	if len(providers) == 0 {
		fmt.Println()
		fmt.Println(ui.WarningStyle.Render("No AI providers found in your organization."))
		fmt.Println()
		return nil
	}

	// Create table
	table := ui.NewTable([]string{"NAME", "MODELS", "FEATURES"})

	for _, provider := range providers {
		// Count non-legacy models
		modelCount := 0
		for _, m := range provider.AvailableModels {
			if !m.Legacy {
				modelCount++
			}
		}

		table.AddRow(
			provider.Name,
			fmt.Sprintf("%d", modelCount),
			fmt.Sprintf("%d", len(provider.AvailableFeatures)),
		)
	}

	// Display table
	fmt.Println()
	fmt.Println(ui.TitleStyle.Render("🤖 AI Providers"))
	fmt.Println()
	fmt.Println(table.Render())
	fmt.Println()
	fmt.Println(ui.MutedStyle.Render(fmt.Sprintf("Total: %d providers", len(providers))))
	fmt.Println()

	// Show detailed model info
	fmt.Println(ui.SubtitleStyle.Render("Available Models by Provider:"))
	fmt.Println()
	for _, provider := range providers {
		fmt.Printf("%s:\n", ui.LabelStyle.Render(provider.Name))
		for _, model := range provider.AvailableModels {
			if model.Legacy {
				continue // Skip legacy models in the display
			}
			multiModal := ""
			if model.MultiModal {
				multiModal = " (multimodal)"
			}
			fmt.Printf("  - %s [%s]%s\n", model.Name, model.ModelType, multiModal)
		}
		fmt.Println()
	}

	return nil
}
