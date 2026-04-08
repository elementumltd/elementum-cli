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
