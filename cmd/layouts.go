// Copyright (c) ElementumAI, Inc.
// SPDX-License-Identifier: MPL-2.0

package cmd

import (
	"github.com/spf13/cobra"
)

var layoutsCmd = &cobra.Command{
	Use:     "layouts",
	Aliases: []string{"layout"},
	Short:   "Layout/stage operations",
	Long: `Commands for managing layouts (stages) on Elementum apps.

In Elementum, a "layout" is a stage on an app. Each stage defines what sections
(field groups, activity log, attachments, etc.) appear on the record detail page.`,
}

func init() {
	layoutsCmd.AddCommand(layoutListCmd)
	layoutsCmd.AddCommand(layoutCreateCmd)
	layoutsCmd.AddCommand(layoutDeleteCmd)
	layoutsCmd.AddCommand(layoutUpdateCmd)
	layoutsCmd.AddCommand(layoutShowCmd)
}

// GetLayoutsCmd returns the layouts command for registration
func GetLayoutsCmd() *cobra.Command {
	return layoutsCmd
}
