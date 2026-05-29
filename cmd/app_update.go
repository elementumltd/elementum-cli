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
	"github.com/elementumltd/elementum-cli/internal/client"
	"github.com/elementumltd/elementum-cli/ui"
	"github.com/spf13/cobra"
)

var appUpdateCmd = &cobra.Command{
	Use:   "update <namespace-or-id>",
	Short: "Update an app or element",
	Long: `Update properties of an existing Elementum App or Element.

Examples:
  ei apps update support-tickets --name "Customer Tickets"
  ei apps update support-tickets --description "Track customer issues"
  ei apps update support-tickets --name "New Name" --description "New desc"`,
	Args: cobra.ExactArgs(1),
	RunE: runAppUpdate,
}

func init() {
	appUpdateCmd.Flags().String("name", "", "New app name")
	appUpdateCmd.Flags().String("description", "", "New app description")
}

func runAppUpdate(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	c, err := auth.GetClientFromCmd(cmd)
	if err != nil {
		return err
	}

	input := args[0]

	name, _ := cmd.Flags().GetString("name")
	description, _ := cmd.Flags().GetString("description")

	if name == "" && description == "" {
		return fmt.Errorf("at least one of --name or --description must be provided")
	}

	var aspectID string
	if looksLikeUUID(input) {
		aspectID = input
	} else {
		aspectID, _, err = resolveAspectByNamespace(ctx, c, input)
		if err != nil {
			return fmt.Errorf("failed to resolve %q: %w", input, err)
		}
	}

	if !isJSONOutput(cmd) {
		fmt.Println(ui.InfoStyle.Render(fmt.Sprintf("⠿ Updating %s...", input)))
	}

	updateInput := client.AspectInput{}
	if name != "" {
		updateInput.Name = &name
	}
	if description != "" {
		updateInput.Description = &description
	}

	resp, err := client.UpdateAspect(ctx, c.Genqlient(), aspectID, updateInput)
	if err != nil {
		return fmt.Errorf("failed to update app: %w", err)
	}

	updated := resp.GetAspectUpdate()
	result := map[string]string{
		"id":   updated.GetId(),
		"name": updated.GetName(),
	}

	if isJSONOutput(cmd) {
		return outputJSON(result)
	}

	fmt.Println(ui.SuccessStyle.Render("Updated:") + fmt.Sprintf(" %s (%s)", updated.GetName(), updated.GetId()))
	return nil
}
