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
	"github.com/elementumltd/elementum-cli/internal/client"
	"github.com/elementumltd/elementum-cli/ui"
	"github.com/spf13/cobra"
)

var layoutDeleteCmd = &cobra.Command{
	Use:   "delete <namespace> <stage-name>",
	Short: "Delete a layout (stage) from an app",
	Long: `Delete a stage from an Elementum App.

WARNING: This permanently removes the stage and its layout configuration.

Examples:
  ei layout delete support-tickets "In Progress"
  ei layout delete support-tickets Review --force
  ei layout delete support-tickets Done --dry-run`,
	Args: cobra.ExactArgs(2),
	RunE: runLayoutDelete,
}

func init() {
	layoutDeleteCmd.Flags().BoolP("force", "f", false, "Skip confirmation prompt")
	layoutDeleteCmd.Flags().Bool("dry-run", false, "Show what would be deleted without deleting")
}

func runLayoutDelete(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	namespace := args[0]
	stageName := args[1]

	c, err := auth.GetClientFromCmd(cmd)
	if err != nil {
		return err
	}

	dryRun, _ := cmd.Flags().GetBool("dry-run")
	force, _ := cmd.Flags().GetBool("force")

	stageID, err := resolveStageByName(ctx, c, namespace, stageName)
	if err != nil {
		return err
	}

	if dryRun {
		fmt.Println(ui.WarningStyle.Render("Dry run:") + fmt.Sprintf(" Would delete stage %q (%s) from %s", stageName, stageID, namespace))
		return nil
	}

	if !force {
		confirmed, err := ui.Confirm(
			fmt.Sprintf("Delete stage %q from %s?", stageName, namespace),
			fmt.Sprintf("ID: %s\nThis will permanently remove the stage and its layout.\nThis action cannot be undone.", stageID),
		)
		if err != nil {
			return err
		}
		if !confirmed {
			fmt.Println(ui.MutedStyle.Render("Cancelled."))
			return nil
		}
	}

	if !isJSONOutput(cmd) {
		fmt.Println(ui.InfoStyle.Render(fmt.Sprintf("⠿ Deleting stage %q from %s...", stageName, namespace)))
	}

	_, err = client.DeleteAspectStage(ctx, c.Genqlient(), stageID)
	if err != nil {
		return fmt.Errorf("failed to delete stage: %w", err)
	}

	if isJSONOutput(cmd) {
		return outputJSON(map[string]string{
			"id":      stageID,
			"name":    stageName,
			"deleted": "true",
		})
	}

	fmt.Println(ui.SuccessStyle.Render("Deleted stage:") + fmt.Sprintf(" %s (%s)", stageName, stageID))
	return nil
}

func resolveStageByName(ctx context.Context, c *client.Client, namespace, stageName string) (string, error) {
	app, err := discovery.GetAppByNamespace(ctx, c, namespace)
	if err != nil {
		return "", fmt.Errorf("failed to find app %q: %w", namespace, err)
	}

	for _, layout := range app.Layouts {
		if strings.EqualFold(layout.Name, stageName) {
			return layout.ID, nil
		}
	}

	available := make([]string, len(app.Layouts))
	for i, l := range app.Layouts {
		available[i] = l.Name
	}
	return "", fmt.Errorf("stage %q not found on %s. Available: %s", stageName, namespace, strings.Join(available, ", "))
}
