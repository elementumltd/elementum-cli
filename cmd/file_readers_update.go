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

var fileReadersUpdateCmd = &cobra.Command{
	Use:   "update <app-namespace> <name-or-id>",
	Short: "Update a file reader",
	Long: `Update properties of an existing file reader.

Examples:
  # Update name
  ei file-readers update support-tickets "Invoice Parser" --name "New Invoice Parser"

  # Update AI instructions
  ei file-readers update support-tickets "Invoice Parser" --instructions "New extraction instructions"

  # Update fields (replaces all fields)
  ei file-readers update support-tickets "Invoice Parser" \
    --field "vendor:text:Vendor name:true" \
    --field "total:decimal:Total amount:true"`,
	Args: cobra.ExactArgs(2),
	RunE: runFileReadersUpdateCmd,
}

func init() {
	fileReadersUpdateCmd.Flags().String("name", "", "New file reader name")
	fileReadersUpdateCmd.Flags().String("instructions", "", "New AI instructions (for AI type only)")
	fileReadersUpdateCmd.Flags().StringSlice("field", []string{}, "Replace fields with new definitions: name:type:description:required (for AI type)")
	fileReadersUpdateCmd.Flags().String("structure-file", "", "JSON file with new structure definition (for JSON/XML type)")
}

func runFileReadersUpdateCmd(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	apiClient, err := auth.GetClientFromCmd(cmd)
	if err != nil {
		return err
	}

	appNamespace := args[0]
	nameOrID := args[1]

	name, _ := cmd.Flags().GetString("name")
	instructions, _ := cmd.Flags().GetString("instructions")
	fields, _ := cmd.Flags().GetStringSlice("field")
	structureFile, _ := cmd.Flags().GetString("structure-file")

	if name == "" && instructions == "" && len(fields) == 0 && structureFile == "" {
		return fmt.Errorf("at least one update flag must be provided (--name, --instructions, --field, --structure-file)")
	}

	aspectID, _, err := resolveAspectByNamespace(ctx, apiClient, appNamespace)
	if err != nil {
		return fmt.Errorf("failed to find app %q: %w", appNamespace, err)
	}

	fileReaderID, typeName, err := resolveFileReaderID(ctx, apiClient, aspectID, nameOrID)
	if err != nil {
		return err
	}

	cliType := graphQLTypeToCliType(typeName)

	// Validate type-specific flags
	if (instructions != "" || len(fields) > 0) && cliType != "ai" {
		return fmt.Errorf("--instructions and --field are only valid for AI file readers")
	}
	if structureFile != "" && cliType != "json" && cliType != "xml" {
		return fmt.Errorf("--structure-file is only valid for JSON/XML file readers")
	}

	// Build update input
	var updateInput client.DocumentModelUpdateInput

	if name != "" {
		updateInput.Name = &name
	}

	// Handle AI-specific updates
	if cliType == "ai" && (instructions != "" || len(fields) > 0) {
		aiUpdate := &client.DocumentModelAiUpdateInput{}

		if instructions != "" {
			aiUpdate.Instructions = &instructions
		}

		if len(fields) > 0 {
			parsedFields, err := parseFieldDefinitions(fields)
			if err != nil {
				return err
			}
			aiUpdate.Fields = parsedFields
		}

		updateInput.Ai = aiUpdate
	}

	// Handle JSON structure update
	if cliType == "json" && structureFile != "" {
		structure, err := parseStructureFile(structureFile)
		if err != nil {
			return err
		}
		updateInput.JsonStructure = structure
	}

	// Handle XML structure update
	if cliType == "xml" && structureFile != "" {
		structure, err := parseStructureFile(structureFile)
		if err != nil {
			return err
		}
		updateInput.Xml = &client.DocumentModelXmlUpdateInput{
			Values: structure,
		}
	}

	if !isJSONOutput(cmd) {
		fmt.Println(ui.InfoStyle.Render(fmt.Sprintf("Updating file reader %s...", nameOrID)))
	}

	resp, err := client.UpdateDocumentModel(ctx, apiClient.Genqlient(), fileReaderID, updateInput)
	if err != nil {
		return fmt.Errorf("failed to update file reader: %w", err)
	}

	updated := resp.DocumentModelUpdate
	result := map[string]string{
		"id":   updated.GetId(),
		"type": cliType,
	}

	// Get typename from response if available
	if tn := updated.GetTypename(); tn != nil {
		result["type"] = graphQLTypeToCliType(*tn)
	}

	displayName := nameOrID
	if name != "" {
		displayName = name
	}

	if isJSONOutput(cmd) {
		result["name"] = displayName
		return outputJSON(result)
	}

	fmt.Println(ui.SuccessStyle.Render("Updated file reader:") + fmt.Sprintf(" %s (%s)", displayName, updated.GetId()))
	return nil
}

