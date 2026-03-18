// Copyright (c) ElementumAI, Inc.
// SPDX-License-Identifier: MPL-2.0

package cmd

import (
	"context"
	"fmt"

	"github.com/elementumltd/elementum-cli/auth"
	"github.com/elementumltd/elementum-cli/ui"
	"github.com/elementumltd/elementum-cli/internal/client"
	"github.com/spf13/cobra"
)

var layoutCreateCmd = &cobra.Command{
	Use:   "create <namespace>",
	Short: "Create a new layout (stage) on an app",
	Long: `Create a new layout stage on an Elementum App.

Stages define the different phases a record goes through (e.g., Open, In Progress, Done).
Each stage can have its own layout with different display block sections.

Examples:
  ei layouts create support-tickets --name "In Progress"
  ei layouts create support-tickets --name "Review" --color "#FF5733" --icon "eye"
  ei layouts create support-tickets --name "Done" --dry-run`,
	Args: cobra.ExactArgs(1),
	RunE: runLayoutCreate,
}

func init() {
	layoutCreateCmd.Flags().String("name", "", "Stage name (required)")
	layoutCreateCmd.Flags().String("color", "", "Stage color (hex, e.g., #FF5733)")
	layoutCreateCmd.Flags().String("icon", "", "Stage icon name")
	layoutCreateCmd.Flags().Bool("dry-run", false, "Show what would be created without creating")
	_ = layoutCreateCmd.MarkFlagRequired("name")
}

func runLayoutCreate(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	namespace := args[0]

	c, err := auth.GetClientFromCmd(cmd)
	if err != nil {
		return err
	}

	name, _ := cmd.Flags().GetString("name")
	color, _ := cmd.Flags().GetString("color")
	icon, _ := cmd.Flags().GetString("icon")
	dryRun, _ := cmd.Flags().GetBool("dry-run")

	aspectID, _, err := resolveAspectByNamespace(ctx, c, namespace)
	if err != nil {
		return fmt.Errorf("failed to resolve namespace %q: %w", namespace, err)
	}

	if dryRun {
		fmt.Println(ui.WarningStyle.Render("Dry run:"))
		fmt.Printf("  App:   %s\n", namespace)
		fmt.Printf("  Name:  %s\n", name)
		if color != "" {
			fmt.Printf("  Color: %s\n", color)
		}
		if icon != "" {
			fmt.Printf("  Icon:  %s\n", icon)
		}
		return nil
	}

	if !isJSONOutput(cmd) {
		fmt.Println(ui.InfoStyle.Render(fmt.Sprintf("⠿ Creating stage %q on %s...", name, namespace)))
	}

	input := client.AspectStageCreateInput{
		Name: name,
	}
	if color != "" {
		input.Color = &color
	}
	if icon != "" {
		input.Icon = &icon
	}

	resp, err := client.CreateAspectStage(ctx, c.Genqlient(), aspectID, input)
	if err != nil {
		return fmt.Errorf("failed to create stage: %w", err)
	}

	created := resp.GetAspectStageCreate()
	result := map[string]string{
		"id":   created.Id,
		"name": created.Name,
	}

	if isJSONOutput(cmd) {
		return outputJSON(result)
	}

	fmt.Println(ui.SuccessStyle.Render("Created stage:") + fmt.Sprintf(" %s (%s)", created.Name, created.Id))
	return nil
}
