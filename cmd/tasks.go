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
	"os"

	"github.com/elementumltd/elementum-cli/auth"
	"github.com/elementumltd/elementum-cli/discovery"
	"github.com/elementumltd/elementum-cli/export"
	"github.com/elementumltd/elementum-cli/logger"
	"github.com/elementumltd/elementum-cli/ui"
	"github.com/spf13/cobra"
)

var tasksCmd = &cobra.Command{
	Use:   "tasks",
	Short: "Manage tasks",
	Long:  "Commands for listing and exporting Elementum tasks (objects of type Task).",
}

var tasksListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all tasks",
	Long:  "Display a table of all tasks (objects of type Task) in your organization.",
	RunE:  runTasksList,
}

var tasksExportCmd = &cobra.Command{
	Use:   "export [namespace-or-id]",
	Short: "Export a task to Terraform",
	Long: `Export an Elementum task as Terraform configuration.

You can specify the task by:
- Namespace: my-task
- ID: uuid

The command will generate an import block and run terraform to create the configuration.`,
	Args: cobra.ExactArgs(1),
	RunE: runTasksExport,
}

func init() {
	tasksCmd.AddCommand(tasksListCmd)
	tasksCmd.AddCommand(tasksExportCmd)

	tasksExportCmd.Flags().StringP("output", "o", "generated.tf", "Output file for generated Terraform configuration")
	tasksExportCmd.Flags().BoolP("verbose", "v", false, "Enable verbose debug output")
}

// GetTasksCmd returns the tasks command for registration
func GetTasksCmd() *cobra.Command {
	return tasksCmd
}

func runTasksList(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Get authenticated client
	client, err := auth.GetClientFromCmd(cmd)
	if err != nil {
		return err
	}

	// Show spinner while loading (skip for JSON output)
	if !isJSONOutput(cmd) {
		fmt.Println(ui.InfoStyle.Render("Loading tasks..."))
	}

	// Discover objects and filter to tasks only
	objects, err := discovery.ListObjects(ctx, client)
	if err != nil {
		return fmt.Errorf("failed to list tasks: %w", err)
	}

	// Filter to tasks only
	var tasks []discovery.ObjectSummary
	for _, obj := range objects {
		if obj.Type == "Task" {
			tasks = append(tasks, obj)
		}
	}

	// JSON output
	if isJSONOutput(cmd) {
		return outputJSON(tasks)
	}

	if len(tasks) == 0 {
		fmt.Println()
		fmt.Println(ui.WarningStyle.Render("No tasks found in your organization."))
		fmt.Println()
		return nil
	}

	// Create table
	table := ui.NewTable([]string{"NAME", "NAMESPACE", "HANDLE", "ID"})

	for _, task := range tasks {
		table.AddRow(
			task.Name,
			task.Namespace,
			task.Handle,
			task.ID,
		)
	}

	// Display table
	fmt.Println()
	fmt.Println(ui.TitleStyle.Render("Tasks"))
	fmt.Println()
	fmt.Println(table.Render())
	fmt.Println()
	fmt.Println(ui.MutedStyle.Render(fmt.Sprintf("Total: %d tasks", len(tasks))))
	fmt.Println()

	return nil
}

func runTasksExport(cmd *cobra.Command, args []string) error {
	input := args[0]
	ctx := context.Background()

	// Get flags
	outputFile, _ := cmd.Flags().GetString("output")
	verbose, _ := cmd.Flags().GetBool("verbose")

	if verbose {
		_ = os.Setenv("DEBUG_GRAPHQL", "1")
	}

	// Set GraphQL debug mode when log level is debug or trace
	level := logger.GetCurrentLevel()
	if level == logger.LevelDebug || level == logger.LevelTrace {
		_ = os.Setenv("DEBUG_GRAPHQL", "1")
	}

	// Get authenticated client
	client, err := auth.GetClientFromCmd(cmd)
	if err != nil {
		return err
	}

	fmt.Println(ui.InfoStyle.Render("Looking up Task..."))

	// Resolve task by namespace first, then by name, then by ID
	task, err := discovery.GetTaskByNamespace(ctx, client, input)
	if err != nil {
		task, err = discovery.GetTaskByName(ctx, client, input)
		if err != nil {
			task, err = discovery.GetTask(ctx, client, input)
			if err != nil {
				return fmt.Errorf("failed to find Task: %w", err)
			}
		}
	}

	fmt.Println(ui.SuccessStyle.Render(fmt.Sprintf("Found Task: %s (%s)", task.Name, task.ID)))

	// Generate HCL from discovered data
	fmt.Println(ui.InfoStyle.Render("Generating Terraform configuration..."))
	content := export.GenerateAspectTaskHCL(task)

	// Write to output file
	err = os.WriteFile(outputFile, []byte(content), 0644)
	if err != nil {
		return fmt.Errorf("failed to write output file: %w", err)
	}

	fmt.Println(ui.SuccessStyle.Render(fmt.Sprintf("Generated %s", outputFile)))
	fmt.Println()
	fmt.Println(ui.SubtitleStyle.Render("Next steps:"))
	fmt.Printf("  %s Review %s\n", ui.RenderBullet(), outputFile)
	fmt.Printf("  %s Run: tofu plan\n", ui.RenderBullet())
	fmt.Println()

	return nil
}
