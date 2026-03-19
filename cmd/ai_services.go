// Copyright (c) ElementumAI, Inc.
// SPDX-License-Identifier: MPL-2.0

package cmd

import (
	"context"
	"fmt"

	"github.com/elementumltd/elementum-cli/auth"
	"github.com/elementumltd/elementum-cli/discovery"
	"github.com/elementumltd/elementum-cli/ui"
	"github.com/spf13/cobra"
)

var aiServicesCmd = &cobra.Command{
	Use:   "ai-services",
	Short: "Manage AI services",
	Long:  "Commands for listing and managing AI services (model configurations) in your organization.",
}

var aiServicesListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all AI services",
	Long:  "Display a table of all configured AI services (model configurations) in your organization.",
	RunE:  runAiServicesList,
}

func init() {
	aiServicesCmd.AddCommand(aiServicesListCmd)
}

// GetAiServicesCmd returns the ai-services command for registration
func GetAiServicesCmd() *cobra.Command {
	return aiServicesCmd
}

func runAiServicesList(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Get authenticated client
	client, err := auth.GetClientFromCmd(cmd)
	if err != nil {
		return err
	}

	// Show spinner while loading (skip for JSON output)
	if !isJSONOutput(cmd) {
		fmt.Println(ui.InfoStyle.Render("⠿ Loading AI services..."))
	}

	// Discover AI services (provider connectors)
	services, err := discovery.ListAiProviderConnectors(ctx, client)
	if err != nil {
		return fmt.Errorf("failed to list AI services: %w", err)
	}

	// JSON output
	if isJSONOutput(cmd) {
		return outputJSON(services)
	}

	if len(services) == 0 {
		fmt.Println()
		fmt.Println(ui.WarningStyle.Render("No AI services found in your organization."))
		fmt.Println()
		return nil
	}

	// Create table
	table := ui.NewTable([]string{"ID", "MODEL", "PROVIDER", "API TYPE", "COST/M TOKENS", "MULTIMODAL"})

	for _, svc := range services {
		multiModal := "No"
		if svc.Model.MultiModal {
			multiModal = "Yes"
		}

		cost := fmt.Sprintf("$%.4f", svc.CostPerMillionTokens)
		if svc.CostPerMillionTokens == 0 {
			cost = "-"
		}

		table.AddRow(
			svc.ID,
			svc.Model.Name,
			svc.ProviderName,
			svc.ApiType,
			cost,
			multiModal,
		)
	}

	// Display table
	fmt.Println()
	fmt.Println(ui.TitleStyle.Render("🔌 AI Services"))
	fmt.Println()
	fmt.Println(table.Render())
	fmt.Println()
	fmt.Println(ui.MutedStyle.Render(fmt.Sprintf("Total: %d services", len(services))))
	fmt.Println()

	return nil
}
