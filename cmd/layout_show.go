// Copyright (c) ElementumAI, Inc.
// SPDX-License-Identifier: MPL-2.0

package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/elementumltd/elementum-cli/auth"
	"github.com/elementumltd/elementum-cli/discovery"
	"github.com/elementumltd/elementum-cli/ui"
	"github.com/spf13/cobra"
)

var layoutShowCmd = &cobra.Command{
	Use:   "show <namespace> <stage-name>",
	Short: "Show layout details for a stage",
	Long: `Show detailed layout information for a specific stage, including all display block sections.

Examples:
  ei layout show support-tickets "Open"
  ei layout show support-tickets Initiate --json`,
	Args: cobra.ExactArgs(2),
	RunE: runLayoutShow,
}

type layoutShowResult struct {
	ID            string             `json:"id"`
	Name          string             `json:"name"`
	IsInitiate    bool               `json:"is_initiate"`
	DisplayBlocks []displayBlockInfo `json:"display_blocks"`
}

type displayBlockInfo struct {
	ID   string `json:"id"`
	Type string `json:"type"`
	Name string `json:"name,omitempty"`
}

func runLayoutShow(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	namespace := args[0]
	stageName := args[1]

	c, err := auth.GetClientFromCmd(cmd)
	if err != nil {
		return err
	}

	if !isJSONOutput(cmd) {
		fmt.Println(ui.InfoStyle.Render(fmt.Sprintf("⠿ Loading layout for %s / %s...", namespace, stageName)))
	}

	app, err := discovery.GetAppByNamespace(ctx, c, namespace)
	if err != nil {
		return fmt.Errorf("failed to find app: %w", err)
	}

	var found *discovery.Layout
	for i, layout := range app.Layouts {
		if strings.EqualFold(layout.Name, stageName) {
			found = &app.Layouts[i]
			break
		}
	}

	if found == nil {
		available := make([]string, len(app.Layouts))
		for i, l := range app.Layouts {
			available[i] = l.Name
		}
		return fmt.Errorf("stage %q not found. Available: %s", stageName, strings.Join(available, ", "))
	}

	blocks := make([]displayBlockInfo, len(found.DisplayBlocks))
	for i, b := range found.DisplayBlocks {
		blocks[i] = displayBlockInfo{
			ID:   b.ID,
			Type: b.Type,
			Name: b.Name,
		}
	}

	result := layoutShowResult{
		ID:            found.ID,
		Name:          found.Name,
		IsInitiate:    found.IsInitiate,
		DisplayBlocks: blocks,
	}

	if isJSONOutput(cmd) {
		return outputJSON(result)
	}

	fmt.Println()
	fmt.Println(ui.TitleStyle.Render(fmt.Sprintf("Layout: %s", found.Name)))
	fmt.Println()
	fmt.Printf("  %s ID: %s\n", ui.RenderBullet(), found.ID)
	if found.IsInitiate {
		fmt.Printf("  %s Initiate stage (platform-managed)\n", ui.RenderBullet())
	}
	fmt.Println()

	if len(found.DisplayBlocks) == 0 {
		fmt.Println(ui.WarningStyle.Render("No display blocks configured."))
	} else {
		fmt.Println(ui.SubtitleStyle.Render("Display Blocks:"))
		fmt.Println()
		table := ui.NewTable([]string{"TYPE", "NAME", "ID"})
		for _, b := range found.DisplayBlocks {
			table.AddRow(b.Type, b.Name, b.ID)
		}
		fmt.Println(table.Render())
	}
	fmt.Println()

	return nil
}
