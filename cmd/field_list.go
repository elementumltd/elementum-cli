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
	"github.com/elementumltd/elementum-cli/ui"
	"github.com/elementumltd/elementum-cli/internal/client"
	"github.com/spf13/cobra"
)

var fieldListCmd = &cobra.Command{
	Use:   "list <namespace>",
	Short: "List fields on an app or element",
	Long: `List all fields on an Elementum App or Element.

Examples:
  ei fields list support-tickets
  ei fields list support-tickets --json
  ei fields list support-tickets --all   # Include system fields`,
	Args: cobra.ExactArgs(1),
	RunE: runFieldList,
}

func init() {
	fieldListCmd.Flags().Bool("all", false, "Include system fields")
}

type fieldListEntry struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Type        string `json:"type"`
	Required    bool   `json:"required"`
	System      bool   `json:"system"`
	Description string `json:"description,omitempty"`
}

func runFieldList(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	namespace := args[0]

	c, err := auth.GetClientFromCmd(cmd)
	if err != nil {
		return err
	}

	showAll, _ := cmd.Flags().GetBool("all")

	aspectID, _, err := resolveAspectByNamespace(ctx, c, namespace)
	if err != nil {
		return fmt.Errorf("failed to resolve namespace %q: %w", namespace, err)
	}

	if !isJSONOutput(cmd) {
		fmt.Println(ui.InfoStyle.Render(fmt.Sprintf("⠿ Loading fields for %s...", namespace)))
	}

	resp, err := client.GetAspectFieldsNoValues(ctx, c.Genqlient(), aspectID)
	if err != nil {
		return fmt.Errorf("failed to fetch fields: %w", err)
	}

	if resp.Organization.Aspect == nil {
		return fmt.Errorf("aspect not found: %s", aspectID)
	}

	aspect := *resp.Organization.Aspect
	var rawFields []client.FieldDetailsNoValues
	switch a := aspect.(type) {
	case *client.GetAspectFieldsNoValuesOrganizationAspectAspectApp:
		for _, edge := range a.Fields.Edges {
			rawFields = append(rawFields, edge.Node)
		}
	case *client.GetAspectFieldsNoValuesOrganizationAspectAspectElement:
		for _, edge := range a.Fields.Edges {
			rawFields = append(rawFields, edge.Node)
		}
	case *client.GetAspectFieldsNoValuesOrganizationAspectAspectTask:
		for _, edge := range a.Fields.Edges {
			rawFields = append(rawFields, edge.Node)
		}
	default:
		return fmt.Errorf("unsupported aspect type")
	}

	var entries []fieldListEntry
	for _, f := range rawFields {
		if !showAll && f.GetSystem() {
			continue
		}

		desc := ""
		if f.GetDescription() != nil {
			desc = *f.GetDescription()
		}

		typeName := ""
		if f.GetTypename() != nil {
			typeName = simplifyFieldType(*f.GetTypename())
		}

		entries = append(entries, fieldListEntry{
			ID:          f.GetId(),
			Name:        f.GetName(),
			Type:        typeName,
			Required:    f.GetRequired(),
			System:      f.GetSystem(),
			Description: desc,
		})
	}

	if isJSONOutput(cmd) {
		return outputJSON(entries)
	}

	if len(entries) == 0 {
		fmt.Println()
		fmt.Println(ui.WarningStyle.Render("No fields found."))
		fmt.Println()
		return nil
	}

	fmt.Println()
	suffix := ""
	if !showAll {
		suffix = " (custom only, use --all for system fields)"
	}
	fmt.Println(ui.TitleStyle.Render(fmt.Sprintf("Fields on %s%s", namespace, suffix)))
	fmt.Println()

	table := ui.NewTable([]string{"NAME", "TYPE", "REQUIRED", "SYSTEM", "ID"})
	for _, e := range entries {
		req := ""
		if e.Required {
			req = "yes"
		}
		sys := ""
		if e.System {
			sys = "yes"
		}
		table.AddRow(e.Name, e.Type, req, sys, e.ID)
	}
	fmt.Println(table.Render())
	fmt.Println()

	return nil
}

func simplifyFieldType(typename string) string {
	mappings := map[string]string{
		"AspectTextField":          "text",
		"AspectNumberField":        "number",
		"AspectBooleanField":       "boolean",
		"AspectDateField":          "date",
		"AspectDateTimeField":      "datetime",
		"AspectPicklistField":      "dropdown",
		"AspectMultiPicklistField": "multiselect",
		"AspectUserField":          "user",
		"AspectGroupField":         "group",
		"AspectHtmlField":          "longtext",
		"AspectAttachmentField":    "attachment",
		"AspectJsonField":          "json",
		"AspectDecimalField":       "decimal",
		"AspectHandleField":        "handle",
	}
	if simple, ok := mappings[typename]; ok {
		return simple
	}
	return strings.TrimPrefix(typename, "Aspect")
}
