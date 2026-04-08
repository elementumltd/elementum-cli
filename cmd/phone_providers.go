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

var phoneProvidersCmd = &cobra.Command{
	Use:   "phone-providers",
	Short: "Manage phone providers",
	Long:  "Commands for listing and managing phone providers (such as Twilio) in your organization.",
}

var phoneProvidersListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all phone providers",
	Long:  "Display a table of all phone providers configured in your organization.",
	RunE:  runPhoneProvidersList,
}

func init() {
	phoneProvidersCmd.AddCommand(phoneProvidersListCmd)
}

// GetPhoneProvidersCmd returns the phone-providers command for registration
func GetPhoneProvidersCmd() *cobra.Command {
	return phoneProvidersCmd
}

func runPhoneProvidersList(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Get authenticated client
	client, err := auth.GetClientFromCmd(cmd)
	if err != nil {
		return err
	}

	// Show spinner while loading (skip for JSON output)
	if !isJSONOutput(cmd) {
		fmt.Println(ui.InfoStyle.Render("Loading phone providers..."))
	}

	// Discover phone providers
	providers, err := discovery.ListPhoneProviders(ctx, client)
	if err != nil {
		return fmt.Errorf("failed to list phone providers: %w", err)
	}

	// JSON output
	if isJSONOutput(cmd) {
		return outputJSON(providers)
	}

	if len(providers) == 0 {
		fmt.Println()
		fmt.Println(ui.WarningStyle.Render("No phone providers found in your organization."))
		fmt.Println()
		return nil
	}

	// Create table
	table := ui.NewTable([]string{"NAME", "TYPE", "SYSTEM", "ID"})

	for _, provider := range providers {
		system := "No"
		if provider.System {
			system = "Yes"
		}

		table.AddRow(
			provider.Name,
			provider.ProviderType,
			system,
			provider.ID,
		)
	}

	// Display table
	fmt.Println()
	fmt.Println(ui.TitleStyle.Render("Phone Providers"))
	fmt.Println()
	fmt.Println(table.Render())
	fmt.Println()
	fmt.Println(ui.MutedStyle.Render(fmt.Sprintf("Total: %d providers", len(providers))))
	fmt.Println()

	return nil
}
