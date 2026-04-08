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
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/elementumltd/elementum-cli/auth"
	"github.com/elementumltd/elementum-cli/ui"
	"github.com/elementumltd/elementum-cli/internal/client"
	"github.com/spf13/cobra"
)

var fileReadersCreateCmd = &cobra.Command{
	Use:   "create <app-namespace>",
	Short: "Create a new file reader",
	Long: `Create a new file reader (document model) in an Elementum app.

Supported types:
  - ai:   AI-powered extraction with custom fields
  - text: OCR text extraction
  - json: JSON parsing with schema
  - xml:  XML parsing with schema

Field format (for AI type): name:type:description:required
  - name: Field name
  - type: text, number, decimal, boolean, date, datetime
  - description: Field description
  - required: true or false

Examples:
  # AI file reader with fields
  ei file-readers create support-tickets --type ai --name "Invoice Parser" \
    --instructions "Extract invoice data from the document" \
    --field "vendor_name:text:Name of the vendor:true" \
    --field "amount:decimal:Total invoice amount:true" \
    --field "invoice_date:date:Date on the invoice:false"

  # Text/OCR file reader
  ei file-readers create support-tickets --type text --name "Document OCR"

  # JSON file reader with structure file
  ei file-readers create support-tickets --type json --name "API Response" \
    --structure-file ./schema.json

  # Dry run
  ei file-readers create support-tickets --type ai --name "Test" \
    --instructions "Test instructions" --dry-run`,
	Args: cobra.ExactArgs(1),
	RunE: runFileReadersCreateCmd,
}

func init() {
	fileReadersCreateCmd.Flags().String("type", "", "File reader type: ai, text, json, xml (required)")
	fileReadersCreateCmd.Flags().String("name", "", "File reader name (required)")
	fileReadersCreateCmd.Flags().String("instructions", "", "AI instructions for extraction (required for type=ai)")
	fileReadersCreateCmd.Flags().StringSlice("field", []string{}, "Field definition: name:type:description:required (for type=ai, repeatable)")
	fileReadersCreateCmd.Flags().String("structure-file", "", "JSON file with structure definition (for type=json/xml)")
	fileReadersCreateCmd.Flags().Bool("dry-run", false, "Preview what would be created without creating")
	_ = fileReadersCreateCmd.MarkFlagRequired("type")
	_ = fileReadersCreateCmd.MarkFlagRequired("name")
}

func runFileReadersCreateCmd(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	apiClient, err := auth.GetClientFromCmd(cmd)
	if err != nil {
		return err
	}

	appNamespace := args[0]
	readerType, _ := cmd.Flags().GetString("type")
	name, _ := cmd.Flags().GetString("name")
	instructions, _ := cmd.Flags().GetString("instructions")
	fields, _ := cmd.Flags().GetStringSlice("field")
	structureFile, _ := cmd.Flags().GetString("structure-file")
	dryRun, _ := cmd.Flags().GetBool("dry-run")

	// Validate type
	if readerType != "ai" && readerType != "text" && readerType != "json" && readerType != "xml" {
		return fmt.Errorf("invalid type %q. Supported: ai, text, json, xml", readerType)
	}

	// Type-specific validation
	if readerType == "ai" && instructions == "" {
		return fmt.Errorf("--instructions is required for AI file readers")
	}

	if (readerType == "json" || readerType == "xml") && structureFile == "" {
		return fmt.Errorf("--structure-file is required for %s file readers", readerType)
	}

	aspectID, _, err := resolveAspectByNamespace(ctx, apiClient, appNamespace)
	if err != nil {
		return fmt.Errorf("failed to find app %q: %w", appNamespace, err)
	}

	// Parse fields for AI type
	var parsedFields []client.DocumentModelAiFieldInput
	if readerType == "ai" && len(fields) > 0 {
		parsedFields, err = parseFieldDefinitions(fields)
		if err != nil {
			return err
		}
	}

	// Parse structure for JSON/XML type
	var structure *client.DocumentModelJsonStructureInput
	if (readerType == "json" || readerType == "xml") && structureFile != "" {
		structure, err = parseStructureFile(structureFile)
		if err != nil {
			return err
		}
	}

	if dryRun {
		fmt.Println(ui.WarningStyle.Render("Dry run:"))
		fmt.Printf("  App:          %s\n", appNamespace)
		fmt.Printf("  Type:         %s\n", readerType)
		fmt.Printf("  Name:         %s\n", name)
		if instructions != "" {
			fmt.Printf("  Instructions: %s\n", truncate(instructions, 80))
		}
		if len(parsedFields) > 0 {
			fmt.Printf("  Fields:       %d fields\n", len(parsedFields))
			for _, f := range parsedFields {
				req := ""
				if f.Required {
					req = " (required)"
				}
				fmt.Printf("    - %s (%s)%s\n", f.Name, f.Type, req)
			}
		}
		if structureFile != "" {
			fmt.Printf("  Structure:    %s\n", structureFile)
		}
		return nil
	}

	if !isJSONOutput(cmd) {
		fmt.Println(ui.InfoStyle.Render(fmt.Sprintf("Creating %s file reader %q...", readerType, name)))
	}

	var createInput client.DocumentModelCreateInput

	switch readerType {
	case "ai":
		createInput.Ai = &client.DocumentModelAiCreateInput{
			Name:         name,
			Instructions: instructions,
			Fields:       parsedFields,
		}
	case "text":
		createInput.Ocr = &client.DocumentModelOCRCreateInput{
			Name: name,
		}
	case "json":
		createInput.Json = &client.DocumentModelJsonCreateInput{
			Name:          name,
			JsonStructure: *structure,
		}
	case "xml":
		createInput.Xml = &client.DocumentModelXmlCreateInput{
			Name:   name,
			Values: *structure,
		}
	}

	resp, err := client.CreateDocumentModel(ctx, apiClient.Genqlient(), aspectID, createInput)
	if err != nil {
		return fmt.Errorf("failed to create file reader: %w", err)
	}

	created := resp.DocumentModelCreate
	result := map[string]string{
		"id":   created.GetId(),
		"type": readerType,
	}

	// Get name from response
	if ai, ok := created.(*client.CreateDocumentModelDocumentModelCreateDocumentModelAi); ok {
		result["name"] = ai.Name
	} else {
		result["name"] = name
	}

	if isJSONOutput(cmd) {
		return outputJSON(result)
	}

	fmt.Println(ui.SuccessStyle.Render("Created file reader:") + fmt.Sprintf(" %s (%s)", result["name"], result["id"]))
	return nil
}

// parseFieldDefinitions parses field definitions from CLI flags
// Format: name:type:description:required
func parseFieldDefinitions(fields []string) ([]client.DocumentModelAiFieldInput, error) {
	var result []client.DocumentModelAiFieldInput

	for _, f := range fields {
		parts := strings.SplitN(f, ":", 4)
		if len(parts) < 2 {
			return nil, fmt.Errorf("invalid field format %q. Expected: name:type:description:required", f)
		}

		name := parts[0]
		fieldType := strings.ToUpper(parts[1])
		description := ""
		required := false

		if len(parts) >= 3 {
			description = parts[2]
		}
		if len(parts) >= 4 {
			required = strings.ToLower(parts[3]) == "true"
		}

		// Validate field type
		var aiFieldType client.DocumentModelAiFieldType
		switch fieldType {
		case "TEXT":
			aiFieldType = client.DocumentModelAiFieldTypeText
		case "NUMBER":
			aiFieldType = client.DocumentModelAiFieldTypeNumber
		case "DECIMAL":
			aiFieldType = client.DocumentModelAiFieldTypeDecimal
		case "BOOLEAN":
			aiFieldType = client.DocumentModelAiFieldTypeBoolean
		case "DATE":
			aiFieldType = client.DocumentModelAiFieldTypeDate
		case "DATETIME":
			aiFieldType = client.DocumentModelAiFieldTypeDatetime
		default:
			return nil, fmt.Errorf("invalid field type %q. Supported: text, number, decimal, boolean, date, datetime", fieldType)
		}

		result = append(result, client.DocumentModelAiFieldInput{
			Name:        name,
			Type:        aiFieldType,
			Description: description,
			Required:    required,
		})
	}

	return result, nil
}

// parseStructureFile reads and parses a JSON structure file
func parseStructureFile(path string) (*client.DocumentModelJsonStructureInput, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read structure file: %w", err)
	}

	// Parse as generic JSON first to determine structure
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("invalid JSON in structure file: %w", err)
	}

	// Convert to DocumentModelJsonStructureInput format
	properties, err := convertToStructureProperties(raw)
	if err != nil {
		return nil, err
	}

	return &client.DocumentModelJsonStructureInput{
		Properties: properties,
	}, nil
}

// convertToStructureProperties converts a JSON object to structure properties
func convertToStructureProperties(obj map[string]interface{}) ([]client.DocumentModelJsonStructureParameterInput, error) {
	var properties []client.DocumentModelJsonStructureParameterInput

	for name, value := range obj {
		prop, err := convertValueToProperty(name, value)
		if err != nil {
			return nil, err
		}
		properties = append(properties, prop)
	}

	return properties, nil
}

// convertValueToProperty converts a JSON value to a structure property
func convertValueToProperty(name string, value interface{}) (client.DocumentModelJsonStructureParameterInput, error) {
	var prop client.DocumentModelJsonStructureParameterInput

	switch v := value.(type) {
	case string:
		prop.Text = &client.DocumentModelJsonStructureTextValueInput{Name: name}
	case float64:
		// Check if it's an integer
		if v == float64(int64(v)) {
			prop.Number = &client.DocumentModelJsonStructureNumberValueInput{Name: name}
		} else {
			prop.Decimal = &client.DocumentModelJsonStructureDecimalValueInput{Name: name}
		}
	case bool:
		prop.Bool = &client.DocumentModelJsonStructureBooleanValueInput{Name: name}
	case map[string]interface{}:
		// Check if it has a special type indicator
		if typeStr, ok := v["_type"].(string); ok {
			switch strings.ToLower(typeStr) {
			case "date":
				prop.Date = &client.DocumentModelJsonStructureDateValueInput{Name: name}
			case "datetime":
				prop.Datetime = &client.DocumentModelJsonStructureDateTimeValueInput{Name: name}
			case "number":
				prop.Number = &client.DocumentModelJsonStructureNumberValueInput{Name: name}
			case "decimal":
				prop.Decimal = &client.DocumentModelJsonStructureDecimalValueInput{Name: name}
			case "boolean", "bool":
				prop.Bool = &client.DocumentModelJsonStructureBooleanValueInput{Name: name}
			case "text", "string":
				prop.Text = &client.DocumentModelJsonStructureTextValueInput{Name: name}
			default:
				return prop, fmt.Errorf("unknown type indicator %q for property %q", typeStr, name)
			}
		} else {
			// Nested object
			nestedProps, err := convertToStructureProperties(v)
			if err != nil {
				return prop, err
			}
			prop.Object = &client.DocumentModelJsonStructureObjectInput{
				Name:       name,
				Properties: nestedProps,
			}
		}
	case []interface{}:
		// Array - determine element type from first element
		if len(v) > 0 {
			elemProp, err := convertValueToProperty("item", v[0])
			if err != nil {
				return prop, err
			}
			prop.Array = &client.DocumentModelJsonStructureArrayInput{
				Name:      name,
				ArrayType: elemProp,
			}
		} else {
			// Empty array - default to text elements
			prop.Array = &client.DocumentModelJsonStructureArrayInput{
				Name: name,
				ArrayType: client.DocumentModelJsonStructureParameterInput{
					Text: &client.DocumentModelJsonStructureTextValueInput{Name: "item"},
				},
			}
		}
	default:
		prop.Text = &client.DocumentModelJsonStructureTextValueInput{Name: name}
	}

	return prop, nil
}
