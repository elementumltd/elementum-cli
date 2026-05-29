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

var fieldUpdateCmd = &cobra.Command{
	Use:   "update <namespace> <field-name>",
	Short: "Update a field on an app or element",
	Long: `Update properties of an existing field on an Elementum App or Element.

You can rename the field, update its description, or toggle required/show-on-create.

Examples:
  ei field update support-tickets "Customer Email" --name "Contact Email"
  ei field update support-tickets Priority --description "Ticket priority level"
  ei field update support-tickets Priority --required
  ei field update support-tickets Score --no-required`,
	Args: cobra.ExactArgs(2),
	RunE: runFieldUpdate,
}

func init() {
	fieldUpdateCmd.Flags().String("name", "", "New field name")
	fieldUpdateCmd.Flags().String("description", "", "New field description")
	fieldUpdateCmd.Flags().Bool("required", false, "Make field required")
	fieldUpdateCmd.Flags().Bool("no-required", false, "Make field not required")
	fieldUpdateCmd.Flags().Bool("show-on-create", false, "Show field on record creation form")
	fieldUpdateCmd.Flags().Bool("no-show-on-create", false, "Hide field from record creation form")
}

func runFieldUpdate(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	namespace := args[0]
	fieldName := args[1]

	c, err := auth.GetClientFromCmd(cmd)
	if err != nil {
		return err
	}

	newName, _ := cmd.Flags().GetString("name")
	description, _ := cmd.Flags().GetString("description")
	requiredFlag, _ := cmd.Flags().GetBool("required")
	noRequiredFlag, _ := cmd.Flags().GetBool("no-required")
	showOnCreateFlag, _ := cmd.Flags().GetBool("show-on-create")
	noShowOnCreateFlag, _ := cmd.Flags().GetBool("no-show-on-create")

	if newName == "" && description == "" && !requiredFlag && !noRequiredFlag && !showOnCreateFlag && !noShowOnCreateFlag {
		return fmt.Errorf("at least one update flag must be provided (--name, --description, --required, --no-required, --show-on-create, --no-show-on-create)")
	}

	aspectID, _, err := resolveAspectByNamespace(ctx, c, namespace)
	if err != nil {
		return fmt.Errorf("failed to resolve namespace %q: %w", namespace, err)
	}

	fieldID, fieldTypeName, err := resolveFieldByName(ctx, c, aspectID, fieldName)
	if err != nil {
		return err
	}

	updateInput, err := buildFieldUpdateInput(fieldTypeName, newName, description, requiredFlag, noRequiredFlag, showOnCreateFlag, noShowOnCreateFlag)
	if err != nil {
		return err
	}

	if !isJSONOutput(cmd) {
		fmt.Println(ui.InfoStyle.Render(fmt.Sprintf("⠿ Updating field %q on %s...", fieldName, namespace)))
	}

	resp, err := client.UpdateField(ctx, c.Genqlient(), fieldID, updateInput)
	if err != nil {
		return fmt.Errorf("failed to update field: %w", err)
	}

	updated := resp.GetAspectFieldUpdate()
	result := map[string]string{
		"id":   updated.GetId(),
		"name": updated.GetName(),
	}

	if isJSONOutput(cmd) {
		return outputJSON(result)
	}

	fmt.Println(ui.SuccessStyle.Render("Updated field:") + fmt.Sprintf(" %s (%s)", updated.GetName(), updated.GetId()))
	return nil
}

func buildFieldUpdateInput(typeName, newName, description string, required, noRequired, showOnCreate, noShowOnCreate bool) (client.AspectFieldUpdateInput, error) {
	var namePtr *string
	if newName != "" {
		namePtr = &newName
	}

	var descPtr *string
	if description != "" {
		descPtr = &description
	}

	var requiredPtr *bool
	if required {
		t := true
		requiredPtr = &t
	} else if noRequired {
		f := false
		requiredPtr = &f
	}

	var showOnCreatePtr *bool
	if showOnCreate {
		t := true
		showOnCreatePtr = &t
	} else if noShowOnCreate {
		f := false
		showOnCreatePtr = &f
	}

	input := client.AspectFieldUpdateInput{}

	// Map GraphQL typename back to the appropriate update input variant
	switch typeName {
	case "AspectTextField":
		input.Text = &client.AspectTextFieldUpdateInput{
			Name: namePtr, Description: descPtr, Required: requiredPtr, ShowOnCreate: showOnCreatePtr,
		}
	case "AspectNumberField":
		input.Number = &client.AspectNumberFieldUpdateInput{
			Name: namePtr, Description: descPtr, Required: requiredPtr, ShowOnCreate: showOnCreatePtr,
		}
	case "AspectBooleanField":
		input.Bool = &client.AspectBooleanFieldUpdateInput{
			Name: namePtr, Description: descPtr, Required: requiredPtr, ShowOnCreate: showOnCreatePtr,
		}
	case "AspectDateField":
		input.Date = &client.AspectDateFieldUpdateInput{
			Name: namePtr, Description: descPtr, Required: requiredPtr, ShowOnCreate: showOnCreatePtr,
		}
	case "AspectDateTimeField":
		input.DateTime = &client.AspectDateTimeFieldUpdateInput{
			Name: namePtr, Description: descPtr, Required: requiredPtr, ShowOnCreate: showOnCreatePtr,
		}
	case "AspectPicklistField":
		input.Picklist = &client.AspectPicklistFieldUpdateInput{
			Name: namePtr, Description: descPtr, Required: requiredPtr, ShowOnCreate: showOnCreatePtr,
		}
	case "AspectMultiPicklistField":
		input.MultiPicklist = &client.AspectMultiPicklistFieldUpdateInput{
			Name: namePtr, Description: descPtr, Required: requiredPtr, ShowOnCreate: showOnCreatePtr,
		}
	case "AspectUserField":
		input.User = &client.AspectUserFieldUpdateInput{
			Name: namePtr, Description: descPtr, Required: requiredPtr, ShowOnCreate: showOnCreatePtr,
		}
	case "AspectGroupField":
		input.Group = &client.AspectGroupFieldUpdateInput{
			Name: namePtr, Description: descPtr, Required: requiredPtr, ShowOnCreate: showOnCreatePtr,
		}
	case "AspectHtmlField":
		input.Html = &client.AspectHtmlFieldUpdateInput{
			Name: namePtr, Description: descPtr, Required: requiredPtr, ShowOnCreate: showOnCreatePtr,
		}
	case "AspectAttachmentField":
		input.Attachment = &client.AspectAttachmentFieldUpdateInput{
			Name: namePtr, Description: descPtr, Required: requiredPtr, ShowOnCreate: showOnCreatePtr,
		}
	case "AspectJsonField":
		input.Json = &client.AspectJsonFieldUpdateInput{
			Name: namePtr, Description: descPtr, Required: requiredPtr, ShowOnCreate: showOnCreatePtr,
		}
	case "AspectDecimalField":
		input.Decimal = &client.AspectDecimalFieldUpdateInput{
			Name: namePtr, Description: descPtr, Required: requiredPtr, ShowOnCreate: showOnCreatePtr,
		}
	case "AspectHandleField":
		input.Handle = &client.AspectHandleFieldUpdateInput{
			Name: namePtr, Description: descPtr, Required: requiredPtr,
		}
	default:
		return input, fmt.Errorf("unsupported field type for update: %s", strings.TrimPrefix(typeName, "Aspect"))
	}

	return input, nil
}
