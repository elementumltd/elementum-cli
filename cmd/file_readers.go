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

// Type aliases to keep code readable
type (
	listDocModelNode     = client.ListDocumentModelsOrganizationAspectAspectAppDocumentModelsDocumentModelConnectionEdgesDocumentModelEdgeNodeDocumentModel
	listDocModelAi       = client.ListDocumentModelsOrganizationAspectAspectAppDocumentModelsDocumentModelConnectionEdgesDocumentModelEdgeNodeDocumentModelAi
	listDocModelOCR      = client.ListDocumentModelsOrganizationAspectAspectAppDocumentModelsDocumentModelConnectionEdgesDocumentModelEdgeNodeDocumentModelOCR
	listDocModelJson     = client.ListDocumentModelsOrganizationAspectAspectAppDocumentModelsDocumentModelConnectionEdgesDocumentModelEdgeNodeDocumentModelJson
	listDocModelXml      = client.ListDocumentModelsOrganizationAspectAspectAppDocumentModelsDocumentModelConnectionEdgesDocumentModelEdgeNodeDocumentModelXml
	listDocModelsAppResp = client.ListDocumentModelsOrganizationAspectAspectApp
)

var fileReadersParentCmd = &cobra.Command{
	Use:   "file-readers",
	Short: "Manage file readers",
	Long:  "Commands for listing, creating, showing, updating, and deleting Elementum file readers (document models).",
}

var fileReadersListCmd = &cobra.Command{
	Use:   "list <app-namespace>",
	Short: "List file readers in an app",
	Long: `List file readers (document models) in an Elementum app.

File readers extract structured data from files. Supported types:
  - ai:   AI-powered extraction with custom fields
  - text: OCR text extraction
  - json: JSON parsing with schema
  - xml:  XML parsing with schema

Examples:
  ei file-readers list support-tickets
  ei file-readers list support-tickets --type ai
  ei file-readers list support-tickets --json`,
	Args: cobra.ExactArgs(1),
	RunE: runFileReadersListCmd,
}

func init() {
	fileReadersListCmd.Flags().String("type", "", "Filter by type: ai, text, json, xml")
	fileReadersParentCmd.AddCommand(fileReadersListCmd)
	fileReadersParentCmd.AddCommand(fileReadersShowCmd)
	fileReadersParentCmd.AddCommand(fileReadersCreateCmd)
	fileReadersParentCmd.AddCommand(fileReadersUpdateCmd)
	fileReadersParentCmd.AddCommand(fileReadersDeleteCmd)
}

// GetFileReadersCmd returns the file-readers command for registration
func GetFileReadersCmd() *cobra.Command {
	return fileReadersParentCmd
}

func runFileReadersListCmd(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	apiClient, err := auth.GetClientFromCmd(cmd)
	if err != nil {
		return err
	}

	appNamespace := args[0]
	typeFilter, _ := cmd.Flags().GetString("type")

	if !isJSONOutput(cmd) {
		fmt.Println(ui.InfoStyle.Render(fmt.Sprintf("Loading file readers from app %q...", appNamespace)))
	}

	aspectID, _, err := resolveAspectByNamespace(ctx, apiClient, appNamespace)
	if err != nil {
		return fmt.Errorf("failed to find app %q: %w", appNamespace, err)
	}

	resp, err := client.ListDocumentModels(ctx, apiClient.Genqlient(), aspectID)
	if err != nil {
		return fmt.Errorf("failed to list file readers: %w", err)
	}

	if resp.Organization.Aspect == nil {
		return fmt.Errorf("app not found")
	}

	aspect := *resp.Organization.Aspect
	appAspect, ok := aspect.(*listDocModelsAppResp)
	if !ok {
		return fmt.Errorf("aspect is not an app")
	}

	var fileReaders []fileReaderListItem
	if appAspect.DocumentModels == nil {
		if isJSONOutput(cmd) {
			return outputJSON(fileReaders)
		}
		fmt.Println()
		fmt.Println(ui.WarningStyle.Render(fmt.Sprintf("No file readers found in app %q.", appNamespace)))
		fmt.Println()
		return nil
	}

	for _, edge := range appAspect.DocumentModels.Edges {
		node := edge.Node
		typeName := ""
		name := ""

		if tn := node.GetTypename(); tn != nil {
			typeName = *tn
		}

		// Get name based on type
		name = getListDocModelName(node)

		cliType := graphQLTypeToCliType(typeName)

		// Apply type filter if specified
		if typeFilter != "" && cliType != typeFilter {
			continue
		}

		fileReaders = append(fileReaders, fileReaderListItem{
			ID:   node.GetId(),
			Name: name,
			Type: cliType,
		})
	}

	if isJSONOutput(cmd) {
		return outputJSON(fileReaders)
	}

	if len(fileReaders) == 0 {
		fmt.Println()
		if typeFilter != "" {
			fmt.Println(ui.WarningStyle.Render(fmt.Sprintf("No %s file readers found in app %q.", typeFilter, appNamespace)))
		} else {
			fmt.Println(ui.WarningStyle.Render(fmt.Sprintf("No file readers found in app %q.", appNamespace)))
		}
		fmt.Println()
		return nil
	}

	table := ui.NewTable([]string{"NAME", "TYPE", "ID"})

	for _, fr := range fileReaders {
		table.AddRow(fr.Name, fr.Type, fr.ID)
	}

	fmt.Println()
	fmt.Println(ui.TitleStyle.Render(fmt.Sprintf("File Readers in %s", appNamespace)))
	fmt.Println()
	fmt.Println(table.Render())
	fmt.Println()
	fmt.Println(ui.MutedStyle.Render(fmt.Sprintf("Total: %d file readers", len(fileReaders))))
	fmt.Println()

	return nil
}

type fileReaderListItem struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"`
}

// getListDocModelName extracts the name from a document model node using type switch
func getListDocModelName(node listDocModelNode) string {
	switch t := node.(type) {
	case *listDocModelAi:
		return t.Name
	case *listDocModelOCR:
		return t.Name
	case *listDocModelJson:
		return t.Name
	case *listDocModelXml:
		return t.Name
	default:
		return ""
	}
}

// graphQLTypeToCliType converts GraphQL type names to CLI type names
func graphQLTypeToCliType(typeName string) string {
	switch typeName {
	case "DocumentModelAi":
		return "ai"
	case "DocumentModelOCR":
		return "text"
	case "DocumentModelJson":
		return "json"
	case "DocumentModelXml":
		return "xml"
	default:
		return strings.ToLower(strings.TrimPrefix(typeName, "DocumentModel"))
	}
}

// resolveFileReaderID resolves a file reader name or ID to a UUID.
func resolveFileReaderID(ctx context.Context, apiClient *client.Client, aspectID, nameOrID string) (string, string, error) {
	if looksLikeUUID(nameOrID) {
		return nameOrID, "", nil
	}
	return resolveFileReaderByName(ctx, apiClient, aspectID, nameOrID)
}

// resolveFileReaderByName looks up a file reader by name within an app (case-insensitive).
func resolveFileReaderByName(ctx context.Context, apiClient *client.Client, aspectID, fileReaderName string) (string, string, error) {
	resp, err := client.ListDocumentModels(ctx, apiClient.Genqlient(), aspectID)
	if err != nil {
		return "", "", fmt.Errorf("failed to list file readers: %w", err)
	}

	if resp.Organization.Aspect == nil {
		return "", "", fmt.Errorf("app not found")
	}

	aspect := *resp.Organization.Aspect
	appAspect, ok := aspect.(*listDocModelsAppResp)
	if !ok {
		return "", "", fmt.Errorf("aspect is not an app")
	}

	if appAspect.DocumentModels == nil {
		return "", "", fmt.Errorf("file reader %q not found. Use 'ei file-readers list <app>' to list available file readers", fileReaderName)
	}

	var matches []struct{ id, name, typeName string }
	for _, edge := range appAspect.DocumentModels.Edges {
		node := edge.Node
		typeName := ""
		if tn := node.GetTypename(); tn != nil {
			typeName = *tn
		}

		name := getListDocModelName(node)

		if strings.EqualFold(name, fileReaderName) {
			matches = append(matches, struct{ id, name, typeName string }{node.GetId(), name, typeName})
		}
	}

	switch len(matches) {
	case 0:
		return "", "", fmt.Errorf("file reader %q not found. Use 'ei file-readers list <app>' to list available file readers", fileReaderName)
	case 1:
		return matches[0].id, matches[0].typeName, nil
	default:
		lines := []string{fmt.Sprintf("multiple file readers named %q found:", fileReaderName)}
		for _, m := range matches {
			lines = append(lines, fmt.Sprintf("  %s  %s (%s)", m.id, m.name, graphQLTypeToCliType(m.typeName)))
		}
		lines = append(lines, "Specify the file reader ID directly to disambiguate.")
		return "", "", fmt.Errorf("%s", strings.Join(lines, "\n"))
	}
}
