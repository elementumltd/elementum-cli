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
	"github.com/elementumltd/elementum-cli/internal/client"
	"github.com/elementumltd/elementum-cli/ui"
	"github.com/spf13/cobra"
)

var fileReadersDeleteCmd = &cobra.Command{
	Use:   "delete <app-namespace> <name-or-id>",
	Short: "Delete a file reader",
	Long: `Delete a file reader by name or ID.

WARNING: This permanently deletes the file reader.

Examples:
  ei file-readers delete support-tickets "Invoice Parser"
  ei file-readers delete support-tickets "Invoice Parser" --force
  ei file-readers delete support-tickets 794e1e48-73af-4760-... --force`,
	Args: cobra.ExactArgs(2),
	RunE: runFileReadersDeleteCmd,
}

func init() {
	fileReadersDeleteCmd.Flags().BoolP("force", "f", false, "Skip confirmation prompt")
}

func runFileReadersDeleteCmd(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	apiClient, err := auth.GetClientFromCmd(cmd)
	if err != nil {
		return err
	}

	appNamespace := args[0]
	nameOrID := args[1]

	aspectID, _, err := resolveAspectByNamespace(ctx, apiClient, appNamespace)
	if err != nil {
		return fmt.Errorf("failed to find app %q: %w", appNamespace, err)
	}

	fileReaderID, typeName, err := resolveFileReaderID(ctx, apiClient, aspectID, nameOrID)
	if err != nil {
		return err
	}

	force, _ := cmd.Flags().GetBool("force")

	// Get file reader name for display
	displayName := nameOrID
	if !looksLikeUUID(nameOrID) {
		displayName = nameOrID
	} else {
		// Try to get the name from the API
		detail, err := getFileReaderDetail(ctx, apiClient, aspectID, fileReaderID, typeName)
		if err == nil && detail.Name != "" {
			displayName = detail.Name
		}
	}

	cliType := graphQLTypeToCliType(typeName)

	if !force {
		confirmed, err := ui.Confirm(
			fmt.Sprintf("Delete %s file reader %q?", cliType, displayName),
			fmt.Sprintf("ID: %s\nThis will permanently delete the file reader.", fileReaderID),
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
		fmt.Println(ui.InfoStyle.Render(fmt.Sprintf("Deleting file reader %q...", displayName)))
	}

	_, err = client.DeleteDocumentModel(ctx, apiClient.Genqlient(), fileReaderID)
	if err != nil {
		// Fetch and display dependencies as blockers
		if blockers := getFileReaderBlockers(ctx, apiClient, aspectID, fileReaderID); blockers != "" {
			fmt.Println()
			fmt.Println(ui.ErrorStyle.Render("Cannot delete file reader - blocked by dependencies:"))
			fmt.Println(blockers)
			fmt.Println()
			fmt.Println(ui.MutedStyle.Render("Delete or update the blocking resources first, then retry."))
			return fmt.Errorf("file reader has active dependencies")
		}
		return fmt.Errorf("failed to delete file reader: %w", err)
	}

	if isJSONOutput(cmd) {
		return outputJSON(map[string]string{
			"id":      fileReaderID,
			"name":    displayName,
			"type":    cliType,
			"deleted": "true",
		})
	}

	fmt.Println(ui.SuccessStyle.Render("Deleted file reader:") + fmt.Sprintf(" %s (%s)", displayName, fileReaderID))
	return nil
}

// getFileReaderBlockers fetches dependencies that block deleting a file reader
// and returns a formatted string of blockers, or empty string if no blockers found
func getFileReaderBlockers(ctx context.Context, c *client.Client, aspectID, fileReaderID string) string {
	resp, err := client.GetDocumentModelDependencies(ctx, c.Genqlient(), aspectID, fileReaderID)
	if err != nil {
		return "" // Can't fetch dependencies, return empty
	}

	aspect := resp.Organization.Aspect
	if aspect == nil {
		return ""
	}

	var blockers []string

	// Type switch to access the document model
	switch a := (*aspect).(type) {
	case *client.GetDocumentModelDependenciesOrganizationAspectAspectApp:
		docModel := a.DocumentModel
		if docModel == nil {
			return ""
		}

		// Access usage through the interface method
		usage := docModel.GetUsage()
		for _, edge := range usage.Automations.Edges {
			blockers = append(blockers, fmt.Sprintf("  • Automation: %s (%s)", edge.Node.Name, edge.Node.Id))
		}
	}

	if len(blockers) == 0 {
		return ""
	}

	return strings.Join(blockers, "\n")
}
