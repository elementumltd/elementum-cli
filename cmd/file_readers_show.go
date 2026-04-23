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

// Type aliases for GetAiDocumentModel types
type (
	getAiDocModelAppResp   = client.GetAiDocumentModelOrganizationAspectAspectApp
	getAiDocModelAi        = client.GetAiDocumentModelOrganizationAspectAspectAppDocumentModelDocumentModelAi
	getAiDocModelField     = client.GetAiDocumentModelOrganizationAspectAspectAppDocumentModelDocumentModelAiFieldsDocumentModelFieldConnectionEdgesDocumentModelFieldEdgeNodeDocumentModelAiField
	getTextDocModelAppResp = client.GetTextDocumentModelOrganizationAspectAspectApp
	getTextDocModelOCR     = client.GetTextDocumentModelOrganizationAspectAspectAppDocumentModelDocumentModelOCR
	getJsonDocModelAppResp = client.GetJsonDocumentModelOrganizationAspectAspectApp
	getJsonDocModelJson    = client.GetJsonDocumentModelOrganizationAspectAspectAppDocumentModelDocumentModelJson
	getXmlDocModelAppResp  = client.GetXmlDocumentModelOrganizationAspectAspectApp
	getXmlDocModelXml      = client.GetXmlDocumentModelOrganizationAspectAspectAppDocumentModelDocumentModelXml
)

var fileReadersShowCmd = &cobra.Command{
	Use:   "show <app-namespace> <name-or-id>",
	Short: "Show file reader details",
	Long: `Display detailed information about a file reader.

Examples:
  ei file-readers show support-tickets "Invoice Parser"
  ei file-readers show support-tickets 794e1e48-73af-4760-...`,
	Args: cobra.ExactArgs(2),
	RunE: runFileReadersShowCmd,
}

func runFileReadersShowCmd(cmd *cobra.Command, args []string) error {
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

	detail, err := getFileReaderDetail(ctx, apiClient, aspectID, fileReaderID, typeName)
	if err != nil {
		return err
	}

	if isJSONOutput(cmd) {
		return outputJSON(detail)
	}

	fmt.Println()
	fmt.Println(ui.TitleStyle.Render(detail.Name))
	fmt.Println()
	fmt.Printf("  %s  %s\n", ui.LabelStyle.Render("ID:"), detail.ID)
	fmt.Printf("  %s  %s\n", ui.LabelStyle.Render("Type:"), detail.Type)

	if detail.Instructions != "" {
		fmt.Println()
		fmt.Println(ui.SubtitleStyle.Render("Instructions:"))
		fmt.Println(detail.Instructions)
	}

	if len(detail.Fields) > 0 {
		fmt.Println()
		fmt.Println(ui.SubtitleStyle.Render(fmt.Sprintf("Fields (%d):", len(detail.Fields))))
		for i, field := range detail.Fields {
			required := ""
			if field.Required {
				required = " (required)"
			}
			fmt.Printf("  %d. %s (%s)%s\n", i+1, field.Name, field.Type, required)
			if field.Description != "" {
				desc := field.Description
				if len(desc) > 70 {
					desc = desc[:67] + "..."
				}
				fmt.Printf("     %s\n", desc)
			}
		}
	}

	fmt.Println()
	return nil
}

type fileReaderDetail struct {
	ID           string                `json:"id"`
	Name         string                `json:"name"`
	Type         string                `json:"type"`
	Instructions string                `json:"instructions,omitempty"`
	Fields       []fileReaderFieldInfo `json:"fields,omitempty"`
}

type fileReaderFieldInfo struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Type        string `json:"type"`
	Description string `json:"description,omitempty"`
	Required    bool   `json:"required"`
}

func getFileReaderDetail(ctx context.Context, apiClient *client.Client, aspectID, fileReaderID, typeName string) (*fileReaderDetail, error) {
	// If we don't know the type, try to determine it from the list
	if typeName == "" {
		resp, err := client.ListDocumentModels(ctx, apiClient.Genqlient(), aspectID)
		if err != nil {
			return nil, fmt.Errorf("failed to list file readers: %w", err)
		}

		if resp.Organization.Aspect == nil {
			return nil, fmt.Errorf("app not found")
		}

		aspect := *resp.Organization.Aspect
		appAspect, ok := aspect.(*listDocModelsAppResp)
		if !ok {
			return nil, fmt.Errorf("aspect is not an app")
		}

		if appAspect.DocumentModels != nil {
			for _, edge := range appAspect.DocumentModels.Edges {
				if edge.Node.GetId() == fileReaderID {
					if tn := edge.Node.GetTypename(); tn != nil {
						typeName = *tn
					}
					break
				}
			}
		}
	}

	// Fetch detailed information based on type
	switch typeName {
	case "DocumentModelAi":
		return getAiFileReaderDetail(ctx, apiClient, aspectID, fileReaderID)
	case "DocumentModelOCR":
		return getTextFileReaderDetail(ctx, apiClient, aspectID, fileReaderID)
	case "DocumentModelJson":
		return getJsonFileReaderDetail(ctx, apiClient, aspectID, fileReaderID)
	case "DocumentModelXml":
		return getXmlFileReaderDetail(ctx, apiClient, aspectID, fileReaderID)
	default:
		return &fileReaderDetail{
			ID:   fileReaderID,
			Type: graphQLTypeToCliType(typeName),
		}, nil
	}
}

func getAiFileReaderDetail(ctx context.Context, apiClient *client.Client, aspectID, fileReaderID string) (*fileReaderDetail, error) {
	resp, err := client.GetAiDocumentModel(ctx, apiClient.Genqlient(), aspectID, fileReaderID)
	if err != nil {
		return nil, fmt.Errorf("failed to get AI file reader: %w", err)
	}

	if resp.Organization.Aspect == nil {
		return nil, fmt.Errorf("app not found")
	}

	aspect := *resp.Organization.Aspect
	appAspect, ok := aspect.(*getAiDocModelAppResp)
	if !ok {
		return nil, fmt.Errorf("aspect is not an app")
	}

	if appAspect.DocumentModel == nil {
		return nil, fmt.Errorf("file reader not found: %s", fileReaderID)
	}

	aiModel, ok := appAspect.DocumentModel.(*getAiDocModelAi)
	if !ok {
		return nil, fmt.Errorf("file reader is not an AI type")
	}

	detail := &fileReaderDetail{
		ID:           aiModel.Id,
		Name:         aiModel.Name,
		Type:         "ai",
		Instructions: aiModel.Instructions,
	}

	for _, edge := range aiModel.Fields.Edges {
		node := edge.Node
		aiField, ok := node.(*getAiDocModelField)
		if ok {
			detail.Fields = append(detail.Fields, fileReaderFieldInfo{
				ID:          aiField.Id,
				Name:        aiField.Name,
				Type:        string(aiField.FieldType),
				Description: aiField.Description,
				Required:    aiField.Required,
			})
		}
	}

	return detail, nil
}

func getTextFileReaderDetail(ctx context.Context, apiClient *client.Client, aspectID, fileReaderID string) (*fileReaderDetail, error) {
	resp, err := client.GetTextDocumentModel(ctx, apiClient.Genqlient(), aspectID, fileReaderID)
	if err != nil {
		return nil, fmt.Errorf("failed to get text file reader: %w", err)
	}

	if resp.Organization.Aspect == nil {
		return nil, fmt.Errorf("app not found")
	}

	aspect := *resp.Organization.Aspect
	appAspect, ok := aspect.(*getTextDocModelAppResp)
	if !ok {
		return nil, fmt.Errorf("aspect is not an app")
	}

	if appAspect.DocumentModel == nil {
		return nil, fmt.Errorf("file reader not found: %s", fileReaderID)
	}

	ocrModel, ok := appAspect.DocumentModel.(*getTextDocModelOCR)
	if !ok {
		return nil, fmt.Errorf("file reader is not a text/OCR type")
	}

	return &fileReaderDetail{
		ID:   ocrModel.Id,
		Name: ocrModel.Name,
		Type: "text",
	}, nil
}

func getJsonFileReaderDetail(ctx context.Context, apiClient *client.Client, aspectID, fileReaderID string) (*fileReaderDetail, error) {
	resp, err := client.GetJsonDocumentModel(ctx, apiClient.Genqlient(), aspectID, fileReaderID)
	if err != nil {
		return nil, fmt.Errorf("failed to get JSON file reader: %w", err)
	}

	if resp.Organization.Aspect == nil {
		return nil, fmt.Errorf("app not found")
	}

	aspect := *resp.Organization.Aspect
	appAspect, ok := aspect.(*getJsonDocModelAppResp)
	if !ok {
		return nil, fmt.Errorf("aspect is not an app")
	}

	if appAspect.DocumentModel == nil {
		return nil, fmt.Errorf("file reader not found: %s", fileReaderID)
	}

	jsonModel, ok := appAspect.DocumentModel.(*getJsonDocModelJson)
	if !ok {
		return nil, fmt.Errorf("file reader is not a JSON type")
	}

	return &fileReaderDetail{
		ID:   jsonModel.Id,
		Name: jsonModel.Name,
		Type: "json",
	}, nil
}

func getXmlFileReaderDetail(ctx context.Context, apiClient *client.Client, aspectID, fileReaderID string) (*fileReaderDetail, error) {
	resp, err := client.GetXmlDocumentModel(ctx, apiClient.Genqlient(), aspectID, fileReaderID)
	if err != nil {
		return nil, fmt.Errorf("failed to get XML file reader: %w", err)
	}

	if resp.Organization.Aspect == nil {
		return nil, fmt.Errorf("app not found")
	}

	aspect := *resp.Organization.Aspect
	appAspect, ok := aspect.(*getXmlDocModelAppResp)
	if !ok {
		return nil, fmt.Errorf("aspect is not an app")
	}

	if appAspect.DocumentModel == nil {
		return nil, fmt.Errorf("file reader not found: %s", fileReaderID)
	}

	xmlModel, ok := appAspect.DocumentModel.(*getXmlDocModelXml)
	if !ok {
		return nil, fmt.Errorf("file reader is not an XML type")
	}

	return &fileReaderDetail{
		ID:   xmlModel.Id,
		Name: xmlModel.Name,
		Type: "xml",
	}, nil
}
