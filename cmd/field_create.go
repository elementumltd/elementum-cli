// Copyright (c) ElementumAI, Inc.
// SPDX-License-Identifier: MPL-2.0

package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/elementumltd/elementum-cli/auth"
	"github.com/elementumltd/elementum-cli/ui"
	"github.com/elementumltd/elementum-cli/internal/client"
	"github.com/spf13/cobra"
)

var fieldCreateCmd = &cobra.Command{
	Use:   "create <namespace>",
	Short: "Create a field on an app or element",
	Long: `Create a new field on an Elementum App or Element.

Supported field types: text, number, boolean, date, datetime, dropdown,
multiselect, user, group, longtext, attachment, json, decimal.

For dropdown and multiselect fields, provide options with --options.

Examples:
  # Create a text field
  ei fields create support-tickets --name "Customer Email" --type text

  # Create a required text field
  ei fields create support-tickets --name "Subject" --type text --required

  # Create a dropdown with options
  ei fields create support-tickets --name "Priority" --type dropdown --options "Low,Medium,High,Critical"

  # Create a multiselect
  ei fields create support-tickets --name "Tags" --type multiselect --options "Bug,Feature,Question"

  # Create a number field
  ei fields create support-tickets --name "Score" --type number

  # Create a boolean field
  ei fields create support-tickets --name "Escalated" --type boolean

  # Create a user field
  ei fields create support-tickets --name "Assigned To" --type user

  # Create a date field with description
  ei fields create support-tickets --name "Due Date" --type date --description "Expected resolution date"

  # Dry run
  ei fields create support-tickets --name "Test" --type text --dry-run`,
	Args: cobra.ExactArgs(1),
	RunE: runFieldCreate,
}

func init() {
	fieldCreateCmd.Flags().String("name", "", "Field name (required)")
	fieldCreateCmd.Flags().String("type", "", "Field type: text, number, boolean, date, datetime, dropdown, multiselect, user, group, longtext, attachment, json, decimal (required)")
	fieldCreateCmd.Flags().StringSlice("options", nil, "Dropdown/multiselect options (comma-separated)")
	fieldCreateCmd.Flags().String("description", "", "Field description")
	fieldCreateCmd.Flags().Bool("required", false, "Make field required")
	fieldCreateCmd.Flags().Bool("show-on-create", false, "Show field on record creation form")
	fieldCreateCmd.Flags().Bool("dry-run", false, "Show what would be created without creating")
	_ = fieldCreateCmd.MarkFlagRequired("name")
	_ = fieldCreateCmd.MarkFlagRequired("type")
}

func runFieldCreate(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	namespace := args[0]

	c, err := auth.GetClientFromCmd(cmd)
	if err != nil {
		return err
	}

	name, _ := cmd.Flags().GetString("name")
	fieldType, _ := cmd.Flags().GetString("type")
	options, _ := cmd.Flags().GetStringSlice("options")
	description, _ := cmd.Flags().GetString("description")
	required, _ := cmd.Flags().GetBool("required")
	showOnCreate, _ := cmd.Flags().GetBool("show-on-create")
	dryRun, _ := cmd.Flags().GetBool("dry-run")

	aspectID, _, err := resolveAspectByNamespace(ctx, c, namespace)
	if err != nil {
		return fmt.Errorf("failed to resolve namespace %q: %w", namespace, err)
	}

	input, err := buildFieldCreateInput(fieldType, name, description, required, showOnCreate, options)
	if err != nil {
		return err
	}

	if dryRun {
		fmt.Println(ui.WarningStyle.Render("Dry run:"))
		fmt.Printf("  App:         %s\n", namespace)
		fmt.Printf("  Name:        %s\n", name)
		fmt.Printf("  Type:        %s\n", fieldType)
		fmt.Printf("  Required:    %v\n", required)
		if description != "" {
			fmt.Printf("  Description: %s\n", description)
		}
		if len(options) > 0 {
			fmt.Printf("  Options:     %s\n", strings.Join(options, ", "))
		}
		return nil
	}

	if !isJSONOutput(cmd) {
		fmt.Println(ui.InfoStyle.Render(fmt.Sprintf("⠿ Creating %s field %q on %s...", fieldType, name, namespace)))
	}

	resp, err := client.CreateField(ctx, c.Genqlient(), aspectID, input)
	if err != nil {
		return fmt.Errorf("failed to create field: %w", err)
	}

	created := resp.GetAspectFieldCreate()
	result := map[string]string{
		"id":   created.GetId(),
		"name": created.GetName(),
		"type": fieldType,
	}

	if isJSONOutput(cmd) {
		return outputJSON(result)
	}

	fmt.Println(ui.SuccessStyle.Render("Created field:") + fmt.Sprintf(" %s (%s)", created.GetName(), created.GetId()))
	return nil
}

func buildFieldCreateInput(fieldType, name, description string, required, showOnCreate bool, options []string) (client.AspectFieldCreateInput, error) {
	var descPtr *string
	if description != "" {
		descPtr = &description
	}

	var requiredPtr *bool
	if required {
		requiredPtr = &required
	}

	var showOnCreatePtr *bool
	if showOnCreate {
		showOnCreatePtr = &showOnCreate
	}

	input := client.AspectFieldCreateInput{}

	switch strings.ToLower(fieldType) {
	case "text":
		input.Text = &client.AspectTextFieldCreateInput{
			Name:         name,
			Description:  descPtr,
			Required:     requiredPtr,
			ShowOnCreate: showOnCreatePtr,
		}
	case "number":
		input.Number = &client.AspectNumberFieldCreateInput{
			Name:         name,
			Description:  descPtr,
			Required:     requiredPtr,
			ShowOnCreate: showOnCreatePtr,
		}
	case "boolean":
		input.Bool = &client.AspectBooleanFieldCreateInput{
			Name:         name,
			Description:  descPtr,
			Required:     requiredPtr,
			ShowOnCreate: showOnCreatePtr,
		}
	case "date":
		input.Date = &client.AspectDateFieldCreateInput{
			Name:         name,
			Description:  descPtr,
			Required:     requiredPtr,
			ShowOnCreate: showOnCreatePtr,
		}
	case "datetime":
		input.DateTime = &client.AspectDateTimeFieldCreateInput{
			Name:         name,
			Description:  descPtr,
			Required:     requiredPtr,
			ShowOnCreate: showOnCreatePtr,
		}
	case "dropdown":
		picklistValues := buildPicklistValues(options)
		input.Picklist = &client.AspectPicklistFieldCreateInput{
			Name:           name,
			Description:    descPtr,
			Required:       requiredPtr,
			ShowOnCreate:   showOnCreatePtr,
			PicklistValues: picklistValues,
		}
	case "multiselect":
		picklistValues := buildPicklistValues(options)
		input.MultiPicklist = &client.AspectMultiPicklistFieldCreateInput{
			Name:           name,
			Description:    descPtr,
			Required:       requiredPtr,
			ShowOnCreate:   showOnCreatePtr,
			PicklistValues: picklistValues,
		}
	case "user":
		input.User = &client.AspectUserFieldCreateInput{
			Name:         name,
			Description:  descPtr,
			Required:     requiredPtr,
			ShowOnCreate: showOnCreatePtr,
		}
	case "group":
		input.Group = &client.AspectGroupFieldCreateInput{
			Name:         name,
			Description:  descPtr,
			Required:     requiredPtr,
			ShowOnCreate: showOnCreatePtr,
		}
	case "longtext":
		input.Html = &client.AspectHtmlFieldCreateInput{
			Name:         name,
			Description:  descPtr,
			Required:     requiredPtr,
			ShowOnCreate: showOnCreatePtr,
		}
	case "attachment":
		input.Attachment = &client.AspectAttachmentFieldCreateInput{
			Name:         name,
			Description:  descPtr,
			Required:     requiredPtr,
			ShowOnCreate: showOnCreatePtr,
		}
	case "json":
		input.Json = &client.AspectJsonFieldCreateInput{
			Name:         name,
			Description:  descPtr,
			Required:     requiredPtr,
			ShowOnCreate: showOnCreatePtr,
		}
	case "decimal":
		input.Decimal = &client.AspectDecimalFieldCreateInput{
			Name:         name,
			Description:  descPtr,
			Required:     requiredPtr,
			ShowOnCreate: showOnCreatePtr,
		}
	default:
		return input, fmt.Errorf("unsupported field type %q. Supported: text, number, boolean, date, datetime, dropdown, multiselect, user, group, longtext, attachment, json, decimal", fieldType)
	}

	return input, nil
}

func buildPicklistValues(options []string) []client.PicklistValueV2Input {
	values := make([]client.PicklistValueV2Input, 0, len(options))
	for _, opt := range options {
		opt = strings.TrimSpace(opt)
		if opt != "" {
			values = append(values, client.PicklistValueV2Input{Value: opt})
		}
	}
	return values
}
