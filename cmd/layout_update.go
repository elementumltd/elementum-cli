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

var layoutUpdateCmd = &cobra.Command{
	Use:   "update <namespace> <stage-name>",
	Short: "Update a layout (stage) on an app",
	Long: `Update properties of an existing stage on an Elementum App.

Examples:
  ei layout update support-tickets "Open" --name "New"
  ei layout update support-tickets "In Progress" --color "#FF5733"
  ei layout update support-tickets "Done" --icon "check"`,
	Args: cobra.ExactArgs(2),
	RunE: runLayoutUpdate,
}

func init() {
	layoutUpdateCmd.Flags().String("name", "", "New stage name")
	layoutUpdateCmd.Flags().String("color", "", "New stage color (hex)")
	layoutUpdateCmd.Flags().String("icon", "", "New stage icon")
	layoutUpdateCmd.Flags().Int("order", -1, "New display order")
}

func runLayoutUpdate(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	namespace := args[0]
	stageName := args[1]

	c, err := auth.GetClientFromCmd(cmd)
	if err != nil {
		return err
	}

	newName, _ := cmd.Flags().GetString("name")
	color, _ := cmd.Flags().GetString("color")
	icon, _ := cmd.Flags().GetString("icon")
	order, _ := cmd.Flags().GetInt("order")

	if newName == "" && color == "" && icon == "" && order == -1 {
		return fmt.Errorf("at least one of --name, --color, --icon, or --order must be provided")
	}

	stageID, err := resolveStageByName(ctx, c, namespace, stageName)
	if err != nil {
		return err
	}

	if !isJSONOutput(cmd) {
		fmt.Println(ui.InfoStyle.Render(fmt.Sprintf("⠿ Updating stage %q on %s...", stageName, namespace)))
	}

	input := client.AspectStageUpdateInput{}
	if newName != "" {
		input.Name = &newName
	}
	if color != "" {
		input.Color = &color
	}
	if icon != "" {
		input.Icon = &icon
	}
	if order >= 0 {
		input.DisplayOrder = &order
	}

	resp, err := client.UpdateAspectStage(ctx, c.Genqlient(), stageID, input)
	if err != nil {
		return fmt.Errorf("failed to update stage: %w", err)
	}

	updated := resp.GetAspectStageUpdate()
	result := map[string]string{
		"id":   updated.Id,
		"name": updated.Name,
	}

	if isJSONOutput(cmd) {
		return outputJSON(result)
	}

	fmt.Println(ui.SuccessStyle.Render("Updated stage:") + fmt.Sprintf(" %s (%s)", updated.Name, updated.Id))
	return nil
}
