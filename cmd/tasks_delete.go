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
	"github.com/elementumltd/elementum-cli/ui"
	"github.com/elementumltd/elementum-cli/internal/client"
	"github.com/spf13/cobra"
)

var tasksDeleteCmd = &cobra.Command{
	Use:   "delete <namespace-or-id>",
	Short: "Delete a task",
	Long: `Delete an Elementum Task by namespace or ID.

WARNING: This permanently deletes the task and all its records.

Examples:
  ei tasks delete action-items
  ei tasks delete action-items --force
  ei tasks delete 9063aed1-bf8c-430d-882f-8c502355a3c7
  ei tasks delete action-items --dry-run`,
	Args: cobra.ExactArgs(1),
	RunE: runTasksDelete,
}

func init() {
	tasksDeleteCmd.Flags().BoolP("force", "f", false, "Skip confirmation prompt")
	tasksDeleteCmd.Flags().Bool("dry-run", false, "Show what would be deleted without deleting")
	tasksCmd.AddCommand(tasksDeleteCmd)
}

func runTasksDelete(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	c, err := auth.GetClientFromCmd(cmd)
	if err != nil {
		return err
	}

	input := args[0]
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	force, _ := cmd.Flags().GetBool("force")

	var aspectID, aspectName string

	if looksLikeUUID(input) {
		aspectID = input
		info, err := getAspectSummary(ctx, c, aspectID)
		if err != nil {
			return fmt.Errorf("task not found: %s", aspectID)
		}
		if info.Type != "AspectTask" {
			return fmt.Errorf("%q is a %s, not a task", input, info.Type)
		}
		aspectName = info.Name
	} else {
		var aspectType string
		aspectID, aspectType, err = resolveAspectByNamespace(ctx, c, input)
		if err != nil {
			return fmt.Errorf("failed to resolve %q: %w", input, err)
		}
		if aspectType != "Task" {
			return fmt.Errorf("%q is a %s, not a task", input, aspectType)
		}
		aspectName = input
	}

	if dryRun {
		fmt.Println(ui.WarningStyle.Render("Dry run:") + fmt.Sprintf(" Would delete task %q (%s)", aspectName, aspectID))
		return nil
	}

	if !force {
		confirmed, err := ui.Confirm(
			fmt.Sprintf("Delete task %q?", aspectName),
			fmt.Sprintf("ID: %s\nThis will permanently delete the task and all its records.\nThis action cannot be undone.", aspectID),
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
		fmt.Println(ui.InfoStyle.Render(fmt.Sprintf("Deleting task %q...", aspectName)))
	}

	_, err = client.DeleteAspect(ctx, c.Genqlient(), aspectID)
	if err != nil {
		// Delete failed - fetch usages to explain why
		usages := fetchAspectUsages(ctx, c, aspectID)
		if len(usages) > 0 {
			fmt.Println(ui.ErrorStyle.Render(fmt.Sprintf("Cannot delete task %q - the following resources are using it:", aspectName)))
			for _, u := range usages {
				fmt.Printf("  - %s\n", u)
			}
			fmt.Println()
			fmt.Println(ui.MutedStyle.Render("Remove or update these resources first."))
			return fmt.Errorf("delete blocked by usages")
		}
		return fmt.Errorf("failed to delete task: %w", err)
	}

	if isJSONOutput(cmd) {
		return outputJSON(map[string]string{
			"id":      aspectID,
			"name":    aspectName,
			"type":    "Task",
			"deleted": "true",
		})
	}

	fmt.Println(ui.SuccessStyle.Render("Deleted task:") + fmt.Sprintf(" %s (%s)", aspectName, aspectID))
	return nil
}
