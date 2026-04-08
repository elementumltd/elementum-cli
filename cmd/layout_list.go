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

var layoutListCmd = &cobra.Command{
	Use:   "list <namespace>",
	Short: "List layouts (stages) for an app",
	Long: `List all layouts/stages configured on an Elementum App.

Each stage has a name and contains display block sections (field groups, activity log, etc.).

Examples:
  ei layouts list support-tickets
  ei layouts list support-tickets --json`,
	Args: cobra.ExactArgs(1),
	RunE: runLayoutList,
}

type layoutListEntry struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	IsInitiate    bool   `json:"is_initiate"`
	DisplayBlocks int    `json:"display_blocks"`
}

func runLayoutList(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	namespace := args[0]

	c, err := auth.GetClientFromCmd(cmd)
	if err != nil {
		return err
	}

	if !isJSONOutput(cmd) {
		fmt.Println(ui.InfoStyle.Render(fmt.Sprintf("⠿ Loading layouts for %s...", namespace)))
	}

	app, err := discovery.GetAppByNamespace(ctx, c, namespace)
	if err != nil {
		return fmt.Errorf("failed to find app: %w", err)
	}

	var entries []layoutListEntry
	for _, layout := range app.Layouts {
		entries = append(entries, layoutListEntry{
			ID:            layout.ID,
			Name:          layout.Name,
			IsInitiate:    layout.IsInitiate,
			DisplayBlocks: len(layout.DisplayBlocks),
		})
	}

	if isJSONOutput(cmd) {
		return outputJSON(entries)
	}

	if len(entries) == 0 {
		fmt.Println()
		fmt.Println(ui.WarningStyle.Render("No layouts found."))
		fmt.Println()
		return nil
	}

	fmt.Println()
	fmt.Println(ui.TitleStyle.Render(fmt.Sprintf("Layouts for %s", namespace)))
	fmt.Println()

	table := ui.NewTable([]string{"NAME", "BLOCKS", "INITIATE", "ID"})
	for _, e := range entries {
		initiate := ""
		if e.IsInitiate {
			initiate = "yes"
		}
		table.AddRow(e.Name, fmt.Sprintf("%d", e.DisplayBlocks), initiate, e.ID)
	}
	fmt.Println(table.Render())
	fmt.Println()

	return nil
}
