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

var fieldDeleteCmd = &cobra.Command{
	Use:   "delete <namespace> <field-name>",
	Short: "Delete a field from an app or element",
	Long: `Delete a field from an Elementum App or Element by name.

WARNING: This permanently removes the field and all its data from all records.

Examples:
  ei field delete support-tickets "Customer Email"
  ei field delete support-tickets Priority --force
  ei field delete support-tickets Score --dry-run`,
	Args: cobra.ExactArgs(2),
	RunE: runFieldDelete,
}

func init() {
	fieldDeleteCmd.Flags().BoolP("force", "f", false, "Skip confirmation prompt")
	fieldDeleteCmd.Flags().Bool("dry-run", false, "Show what would be deleted without deleting")
}

func runFieldDelete(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	namespace := args[0]
	fieldName := args[1]

	c, err := auth.GetClientFromCmd(cmd)
	if err != nil {
		return err
	}

	dryRun, _ := cmd.Flags().GetBool("dry-run")
	force, _ := cmd.Flags().GetBool("force")

	aspectID, _, err := resolveAspectByNamespace(ctx, c, namespace)
	if err != nil {
		return fmt.Errorf("failed to resolve namespace %q: %w", namespace, err)
	}

	fieldID, fieldType, err := resolveFieldByName(ctx, c, aspectID, fieldName)
	if err != nil {
		return err
	}

	if dryRun {
		fmt.Println(ui.WarningStyle.Render("Dry run:") + fmt.Sprintf(" Would delete field %q (%s) from %s", fieldName, fieldID, namespace))
		return nil
	}

	if !force {
		confirmed, err := ui.Confirm(
			fmt.Sprintf("Delete field %q from %s?", fieldName, namespace),
			fmt.Sprintf("ID: %s\nType: %s\nThis will permanently remove the field and all its data from records.\nThis action cannot be undone.", fieldID, fieldType),
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
		fmt.Println(ui.InfoStyle.Render(fmt.Sprintf("⠿ Deleting field %q from %s...", fieldName, namespace)))
	}

	_, err = client.DeleteField(ctx, c.Genqlient(), fieldID)
	if err != nil {
		return fmt.Errorf("failed to delete field: %w", err)
	}

	if isJSONOutput(cmd) {
		return outputJSON(map[string]string{
			"id":      fieldID,
			"name":    fieldName,
			"deleted": "true",
		})
	}

	fmt.Println(ui.SuccessStyle.Render("Deleted field:") + fmt.Sprintf(" %s (%s)", fieldName, fieldID))
	return nil
}

func resolveFieldByName(ctx context.Context, c *client.Client, aspectID, fieldName string) (string, string, error) {
	resp, err := client.GetAspectFieldsNoValues(ctx, c.Genqlient(), aspectID)
	if err != nil {
		return "", "", fmt.Errorf("failed to fetch fields: %w", err)
	}

	if resp.Organization.Aspect == nil {
		return "", "", fmt.Errorf("aspect not found: %s", aspectID)
	}

	aspect := *resp.Organization.Aspect
	var fields []client.FieldDetailsNoValues
	switch a := aspect.(type) {
	case *client.GetAspectFieldsNoValuesOrganizationAspectAspectApp:
		for _, edge := range a.Fields.Edges {
			fields = append(fields, edge.Node)
		}
	case *client.GetAspectFieldsNoValuesOrganizationAspectAspectElement:
		for _, edge := range a.Fields.Edges {
			fields = append(fields, edge.Node)
		}
	case *client.GetAspectFieldsNoValuesOrganizationAspectAspectTask:
		for _, edge := range a.Fields.Edges {
			fields = append(fields, edge.Node)
		}
	default:
		return "", "", fmt.Errorf("unsupported aspect type")
	}

	for _, f := range fields {
		if strings.EqualFold(f.GetName(), fieldName) {
			typeName := ""
			if f.GetTypename() != nil {
				typeName = *f.GetTypename()
			}
			return f.GetId(), typeName, nil
		}
	}

	return "", "", fmt.Errorf("field %q not found", fieldName)
}
