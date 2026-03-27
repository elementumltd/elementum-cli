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
	"path/filepath"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/elementumltd/elementum-cli/auth"
	"github.com/elementumltd/elementum-cli/discovery"
	"github.com/elementumltd/elementum-cli/ui"
	"github.com/elementumltd/elementum-cli/internal/client"
	"github.com/spf13/cobra"
)

var recordsCreateCmd = &cobra.Command{
	Use:   "create <namespace-or-url>",
	Short: "Create a new record in an app, element, or task",
	Long: `Create a new record in an Elementum App, Element, or Task.

You can specify the target using:
  - A namespace (e.g., "support-tickets", "my-app")
  - A full URL (e.g., "https://org.elementum.io/app/support-tickets/list")

Field values can be provided in several ways:
  - Field flags: -f "Title=Bug Report" -f "Priority=High"
  - JSON file: --from-file data.json
  - Interactive mode: -i

Examples:
  # Simple field values by name
  ei records create support-tickets -f "Title=Bug Report" -f "Priority=High"

  # With file attachment
  ei records create support-tickets -f "Title=Bug Report" -a "Attachments=./screenshot.png"

  # From JSON file
  ei records create support-tickets --from-file data.json

  # Interactive mode
  ei records create support-tickets -i

  # Using field IDs
  ei records create support-tickets -f "fdb890d8-a296-4b8d-b0ec-2656e2fcfb70=test"

  # Dry run (show what would be created)
  ei records create support-tickets -f "Title=Test" --dry-run`,
	Args: cobra.ExactArgs(1),
	RunE: runRecordsCreate,
}

func init() {
	recordsCreateCmd.Flags().StringArrayP("field", "f", nil, "Field value: \"Name=Value\" or \"fieldId=Value\" (repeatable)")
	recordsCreateCmd.Flags().String("from-file", "", "JSON file with field values")
	recordsCreateCmd.Flags().StringArrayP("attach", "a", nil, "File attachment: \"FieldName=/path/to/file\" (repeatable)")
	recordsCreateCmd.Flags().BoolP("interactive", "i", false, "Prompt for field values interactively")
	recordsCreateCmd.Flags().Bool("dry-run", false, "Show what would be created without creating")
}

func runRecordsCreate(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Get authenticated client
	c, err := auth.GetClientFromCmd(cmd)
	if err != nil {
		return err
	}

	// Parse input (could be namespace or URL)
	input := args[0]
	var namespace string

	// Check if it looks like a URL
	if strings.Contains(input, "/") {
		resourceInfo, err := discovery.ParseResourceFromURL(input)
		if err != nil {
			return fmt.Errorf("failed to parse URL: %w", err)
		}
		namespace = resourceInfo.Identifier
	} else {
		namespace = input
	}

	// Resolve namespace to aspect ID
	aspectID, _, err := resolveAspectByNamespace(ctx, c, namespace)
	if err != nil {
		return fmt.Errorf("failed to resolve namespace %q: %w", namespace, err)
	}

	// Get flags
	fieldFlags, _ := cmd.Flags().GetStringArray("field")
	fromFile, _ := cmd.Flags().GetString("from-file")
	attachFlags, _ := cmd.Flags().GetStringArray("attach")
	interactive, _ := cmd.Flags().GetBool("interactive")
	dryRun, _ := cmd.Flags().GetBool("dry-run")

	// Fetch aspect info including type and fields
	aspectInfo, err := discovery.GetAspectInfo(ctx, c, aspectID)
	if err != nil {
		return fmt.Errorf("failed to fetch aspect info: %w", err)
	}
	fields := aspectInfo.Fields

	// Build record input
	recordInput := discovery.RecordCreateInput{
		AspectID: aspectID,
		Fields:   make(map[string]interface{}),
	}

	// Process field values from different sources
	if fromFile != "" {
		// Load from JSON file
		if err := loadFieldsFromFile(fromFile, fields, &recordInput); err != nil {
			return err
		}
	}

	// Process command-line field flags (override file values)
	for _, fieldFlag := range fieldFlags {
		name, value, err := discovery.ParseFieldAssignment(fieldFlag)
		if err != nil {
			return err
		}

		fieldID, fieldInfo, err := discovery.ResolveFieldNameToID(fields, name)
		if err != nil {
			return fmt.Errorf("failed to resolve field %q: %w", name, err)
		}

		// Convert value to appropriate type
		resolvedValue, err := resolveFieldValueWithLookup(ctx, c, fieldInfo, value)
		if err != nil {
			return fmt.Errorf("failed to resolve value for field %q: %w", name, err)
		}

		recordInput.Fields[fieldID] = resolvedValue
	}

	// Process attachment flags
	for _, attachFlag := range attachFlags {
		fieldName, filePath, err := discovery.ParseAttachmentAssignment(attachFlag)
		if err != nil {
			return err
		}

		// Validate file exists
		if err := discovery.ValidateAttachmentPath(filePath); err != nil {
			return err
		}

		// Resolve field name to ID if provided
		var fieldID string
		if fieldName != "" {
			fieldID, _, err = discovery.ResolveFieldNameToID(fields, fieldName)
			if err != nil {
				return fmt.Errorf("failed to resolve attachment field %q: %w", fieldName, err)
			}
		}

		recordInput.Attachments = append(recordInput.Attachments, discovery.AttachmentInput{
			FieldName: fieldName,
			FieldID:   fieldID,
			FilePath:  filePath,
		})
	}

	// Interactive mode
	if interactive {
		if err := collectFieldsInteractively(fields, &recordInput); err != nil {
			return err
		}
	}

	// Validate we have at least some input
	if len(recordInput.Fields) == 0 && len(recordInput.Attachments) == 0 {
		return fmt.Errorf("no field values provided; use -f, --from-file, -a, or -i to provide values")
	}

	// Validate required fields are provided
	missingFields := discovery.GetMissingRequiredFields(fields, recordInput.Fields, aspectInfo.Type)
	if len(missingFields) > 0 {
		// Check if the missing field is a non-system HANDLE field (requires user-provided ID)
		idField := discovery.GetIDField(fields)
		for _, missing := range missingFields {
			if idField != nil && !idField.System && missing == idField.Name {
				return fmt.Errorf("missing required field %q on %q\n\nThis object has a user-managed ID field that must be provided when creating records.\nUse: ei records create %s -f \"%s=<unique-value>\" -f \"Title=<title>\"",
					idField.Name, aspectInfo.Name, namespace, idField.Name)
			}
		}
		return fmt.Errorf("missing required fields: %s", strings.Join(missingFields, ", "))
	}

	// Dry run mode
	if dryRun {
		return displayDryRunWithAspectInfo(fields, &recordInput, aspectInfo)
	}

	// Create the record
	if !isJSONOutput(cmd) {
		fmt.Println(ui.InfoStyle.Render("Creating record..."))
	}

	result, err := discovery.CreateRecord(ctx, c, recordInput)
	if err != nil {
		return err
	}

	// Display result
	if isJSONOutput(cmd) {
		return outputJSON(map[string]interface{}{
			"id":     result.ID,
			"handle": result.Handle,
			"title":  result.Title,
			"url":    result.URL,
		})
	}

	fmt.Println()
	fmt.Println(ui.SuccessStyle.Render(fmt.Sprintf("Created record in %s", namespace)))
	fmt.Println()
	fmt.Printf("  %s  %s\n", ui.MutedStyle.Render("Handle:"), result.Handle)
	fmt.Printf("  %s  %s\n", ui.MutedStyle.Render("ID:"), result.ID)
	if result.Title != "" {
		fmt.Printf("  %s  %s\n", ui.MutedStyle.Render("Title:"), result.Title)
	}
	fmt.Printf("  %s  %s\n", ui.MutedStyle.Render("URL:"), result.URL)
	fmt.Println()

	return nil
}

// loadFieldsFromFile loads field values from a JSON file
func loadFieldsFromFile(filePath string, fields []discovery.AspectFieldInfo, input *discovery.RecordCreateInput) error {
	jsonInput, err := discovery.ParseJSONInputFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to parse JSON file: %w", err)
	}

	// Process fields
	for name, value := range jsonInput.Fields {
		fieldID, fieldInfo, err := discovery.ResolveFieldNameToID(fields, name)
		if err != nil {
			return fmt.Errorf("failed to resolve field %q: %w", name, err)
		}

		// For JSON input, values are already typed - convert to appropriate format
		var resolvedValue interface{}
		switch v := value.(type) {
		case string:
			resolvedValue, err = discovery.ResolveFieldValue(fieldInfo, v)
			if err != nil {
				return fmt.Errorf("failed to resolve value for field %q: %w", name, err)
			}
		case float64:
			// JSON numbers come as float64
			if fieldInfo != nil && fieldInfo.Type == "AspectNumberField" {
				resolvedValue = int64(v)
			} else {
				resolvedValue = v
			}
		case bool:
			resolvedValue = v
		case nil:
			// Skip null values
			continue
		case []interface{}:
			// Multi-select values
			resolvedValue = v
		default:
			resolvedValue = value
		}

		input.Fields[fieldID] = resolvedValue
	}

	// Process attachments
	baseDir := filepath.Dir(filePath)
	for _, att := range jsonInput.Attachments {
		attPath := att.Path
		// If path is relative, make it relative to the JSON file
		if !filepath.IsAbs(attPath) {
			attPath = filepath.Join(baseDir, attPath)
		}

		if err := discovery.ValidateAttachmentPath(attPath); err != nil {
			return err
		}

		var fieldID string
		if att.FieldName != "" {
			fieldID, _, err = discovery.ResolveFieldNameToID(fields, att.FieldName)
			if err != nil {
				return fmt.Errorf("failed to resolve attachment field %q: %w", att.FieldName, err)
			}
		}

		input.Attachments = append(input.Attachments, discovery.AttachmentInput{
			FieldName: att.FieldName,
			FieldID:   fieldID,
			FilePath:  attPath,
			FileName:  att.Name,
		})
	}

	return nil
}

// resolveFieldValueWithLookup resolves a field value, performing user/group lookups if needed
func resolveFieldValueWithLookup(ctx context.Context, c *client.Client, fieldInfo *discovery.AspectFieldInfo, value string) (interface{}, error) {
	if fieldInfo == nil {
		return value, nil
	}

	// Handle user/group fields with email/name lookup
	switch fieldInfo.Type {
	case "AspectUserField":
		// Check if it looks like an email
		if strings.Contains(value, "@") {
			userID, err := discovery.LookupUserByEmail(ctx, c, value)
			if err != nil {
				return nil, err
			}
			return userID, nil
		}
		// Otherwise, try standard resolution (UUID check)
		return discovery.ResolveFieldValue(fieldInfo, value)

	case "AspectGroupField":
		// Try to lookup by name first, then fall back to standard resolution
		groupID, err := discovery.LookupGroupByName(ctx, c, value)
		if err == nil {
			return groupID, nil
		}
		// Fall back to standard resolution (UUID check)
		return discovery.ResolveFieldValue(fieldInfo, value)

	default:
		return discovery.ResolveFieldValue(fieldInfo, value)
	}
}

// collectFieldsInteractively prompts the user for field values
func collectFieldsInteractively(fields []discovery.AspectFieldInfo, input *discovery.RecordCreateInput) error {
	// Filter to editable fields (non-system, non-calculated)
	var editableFields []discovery.AspectFieldInfo
	for _, f := range fields {
		if f.System {
			continue
		}
		// Skip certain field types that can't be set directly
		switch f.Type {
		case "AspectHandleField", "AspectAttachmentField":
			continue
		}
		editableFields = append(editableFields, f)
	}

	if len(editableFields) == 0 {
		return fmt.Errorf("no editable fields found")
	}

	fmt.Println()
	fmt.Println(ui.TitleStyle.Render("Enter field values (press Enter to skip optional fields):"))
	fmt.Println()

	for _, field := range editableFields {
		// Build prompt
		label := field.Name
		if field.Required {
			label += " *"
		}

		var value string
		var err error

		switch field.Type {
		case "AspectPicklistField":
			// Show dropdown options
			var options []huh.Option[string]
			options = append(options, huh.NewOption("(skip)", ""))
			for _, opt := range field.Options {
				options = append(options, huh.NewOption(opt.Label, opt.ID))
			}
			selectField := huh.NewSelect[string]().
				Title(label).
				Options(options...).
				Value(&value)
			if err = selectField.Run(); err != nil {
				return err
			}

		case "AspectMultiPicklistField":
			// Show multi-select options
			var selected []string
			var options []huh.Option[string]
			for _, opt := range field.Options {
				options = append(options, huh.NewOption(opt.Label, opt.ID))
			}
			multiSelect := huh.NewMultiSelect[string]().
				Title(label).
				Options(options...).
				Value(&selected)
			if err = multiSelect.Run(); err != nil {
				return err
			}
			if len(selected) > 0 {
				input.Fields[field.ID] = selected
			}
			continue

		case "AspectBooleanField":
			// Show yes/no
			var confirmed bool
			confirmField := huh.NewConfirm().
				Title(label).
				Value(&confirmed)
			if err = confirmField.Run(); err != nil {
				return err
			}
			input.Fields[field.ID] = confirmed
			continue

		case "AspectHtmlField":
			// Use text area for long text
			textArea := huh.NewText().
				Title(label).
				Value(&value)
			if err = textArea.Run(); err != nil {
				return err
			}

		default:
			// Use input for most fields
			inputField := huh.NewInput().
				Title(label).
				Value(&value)
			if err = inputField.Run(); err != nil {
				return err
			}
		}

		// Skip empty values (unless required)
		if value == "" {
			if field.Required {
				fmt.Printf("  %s is required, please provide a value\n", field.Name)
				continue
			}
			continue
		}

		// For picklist fields, value is already the ID
		if field.Type == "AspectPicklistField" {
			input.Fields[field.ID] = value
		} else {
			// Resolve value for other field types
			resolved, err := discovery.ResolveFieldValue(&field, value)
			if err != nil {
				fmt.Printf("  Warning: %v (skipping)\n", err)
				continue
			}
			input.Fields[field.ID] = resolved
		}
	}

	return nil
}

// displayDryRun shows what would be created without actually creating
func displayDryRun(fields []discovery.AspectFieldInfo, input *discovery.RecordCreateInput) error {
	return displayDryRunWithAspectInfo(fields, input, nil)
}

// displayDryRunWithAspectInfo shows what would be created with aspect type info
func displayDryRunWithAspectInfo(fields []discovery.AspectFieldInfo, input *discovery.RecordCreateInput, aspectInfo *discovery.AspectInfo) error {
	// Build field name lookup
	fieldByID := make(map[string]discovery.AspectFieldInfo)
	for _, f := range fields {
		fieldByID[f.ID] = f
	}

	fmt.Println()
	if aspectInfo != nil {
		fmt.Println(ui.TitleStyle.Render(fmt.Sprintf("Dry Run - Would create record in %s (%s):", aspectInfo.Name, aspectInfo.Type)))
	} else {
		fmt.Println(ui.TitleStyle.Render("Dry Run - Would create record with:"))
	}
	fmt.Println()

	if len(input.Fields) > 0 {
		fmt.Println(ui.SubtitleStyle.Render("Fields:"))
		for fieldID, value := range input.Fields {
			fieldName := fieldID
			if f, ok := fieldByID[fieldID]; ok {
				fieldName = f.Name
			}

			// Format value for display
			valueStr := fmt.Sprintf("%v", value)
			if len(valueStr) > 60 {
				valueStr = valueStr[:57] + "..."
			}
			fmt.Printf("  %s = %s\n", fieldName, valueStr)
		}
		fmt.Println()
	}

	if len(input.Attachments) > 0 {
		fmt.Println(ui.SubtitleStyle.Render("Attachments:"))
		for _, att := range input.Attachments {
			fieldName := att.FieldName
			if fieldName == "" {
				fieldName = "(no field)"
			}
			fmt.Printf("  %s: %s\n", fieldName, att.FilePath)
		}
		fmt.Println()
	}

	fmt.Println(ui.MutedStyle.Render("Run without --dry-run to create the record."))
	fmt.Println()

	return nil
}

// outputJSONForCreate outputs the result as JSON
func outputJSONForCreate(data interface{}) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(data)
}
