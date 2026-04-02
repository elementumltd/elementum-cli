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

var phoneServicesCmd = &cobra.Command{
	Use:   "phone-services",
	Short: "Manage phone services",
	Long:  "Commands for listing and managing phone services (phone numbers linked to AI agents) in your apps.",
}

var phoneServicesListCmd = &cobra.Command{
	Use:   "list <app-namespace>",
	Short: "List phone services for an app",
	Long:  "Display a table of all phone services configured for a specific app.",
	Args:  cobra.ExactArgs(1),
	RunE:  runPhoneServicesList,
}

func init() {
	phoneServicesCmd.AddCommand(phoneServicesListCmd)
}

// GetPhoneServicesCmd returns the phone-services command for registration
func GetPhoneServicesCmd() *cobra.Command {
	return phoneServicesCmd
}

func runPhoneServicesList(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	namespace := args[0]

	// Get authenticated client
	client, err := auth.GetClientFromCmd(cmd)
	if err != nil {
		return err
	}

	// Show spinner while loading (skip for JSON output)
	if !isJSONOutput(cmd) {
		fmt.Println(ui.InfoStyle.Render(fmt.Sprintf("Loading phone services for %s...", namespace)))
	}

	// Resolve namespace to aspect ID
	aspectID, _, err := resolveAspectByNamespace(ctx, client, namespace)
	if err != nil {
		return fmt.Errorf("failed to resolve namespace %q: %w", namespace, err)
	}

	// Discover phone services
	services, err := discovery.GetPhoneServicesForApp(ctx, client, aspectID)
	if err != nil {
		return fmt.Errorf("failed to list phone services: %w", err)
	}

	// JSON output
	if isJSONOutput(cmd) {
		return outputJSON(services)
	}

	if len(services) == 0 {
		fmt.Println()
		fmt.Println(ui.WarningStyle.Render(fmt.Sprintf("No phone services found for %s.", namespace)))
		fmt.Println()
		return nil
	}

	// Create table
	table := ui.NewTable([]string{"PHONE NUMBER", "PROVIDER", "AGENT", "ID"})

	for _, service := range services {
		agentDisplay := service.AgentName
		if agentDisplay == "" {
			agentDisplay = service.AgentID
		}

		table.AddRow(
			service.PhoneNumber,
			service.ProviderID,
			agentDisplay,
			service.ID,
		)
	}

	// Display table
	fmt.Println()
	fmt.Println(ui.TitleStyle.Render(fmt.Sprintf("Phone Services - %s", namespace)))
	fmt.Println()
	fmt.Println(table.Render())
	fmt.Println()
	fmt.Println(ui.MutedStyle.Render(fmt.Sprintf("Total: %d services", len(services))))
	fmt.Println()

	return nil
}
