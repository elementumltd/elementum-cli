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
	"github.com/elementumltd/elementum-cli/discovery"
	"github.com/elementumltd/elementum-cli/internal/client"
	"github.com/elementumltd/elementum-cli/ui"
	"github.com/spf13/cobra"
)

var appCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new app or element",
	Long: `Create a new Elementum App or Element.

An App is a stateful object with stages/workflow. An Element is stateless reference data.

Examples:
  # Create an app with required fields
  ei apps create --name "Support Tickets" --namespace support-tickets --category "IT"

  # Create an element
  ei apps create --name "Locations" --namespace locations --category "Reference Data" --type element

  # Create with description
  ei apps create --name "Tasks" --namespace tasks --category "PM" --description "Project tasks"

  # Dry run
  ei apps create --name "Test" --namespace test --category "IT" --dry-run`,
	RunE: runAppCreate,
}

func init() {
	appCreateCmd.Flags().String("name", "", "App name (required)")
	appCreateCmd.Flags().String("namespace", "", "App namespace/slug (required)")
	appCreateCmd.Flags().String("category", "", "Category name (required)")
	appCreateCmd.Flags().String("type", "app", "Object type: app or element")
	appCreateCmd.Flags().String("description", "", "App description")
	appCreateCmd.Flags().Bool("dry-run", false, "Show what would be created without creating")
	_ = appCreateCmd.MarkFlagRequired("name")
	_ = appCreateCmd.MarkFlagRequired("namespace")
	_ = appCreateCmd.MarkFlagRequired("category")
}

func runAppCreate(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	c, err := auth.GetClientFromCmd(cmd)
	if err != nil {
		return err
	}

	name, _ := cmd.Flags().GetString("name")
	namespace, _ := cmd.Flags().GetString("namespace")
	categoryName, _ := cmd.Flags().GetString("category")
	objType, _ := cmd.Flags().GetString("type")
	description, _ := cmd.Flags().GetString("description")
	dryRun, _ := cmd.Flags().GetBool("dry-run")

	var aspectType client.AspectType
	switch strings.ToLower(objType) {
	case "app":
		aspectType = client.AspectTypeApp
	case "element":
		aspectType = client.AspectTypeElement
	default:
		return fmt.Errorf("invalid type %q: must be 'app' or 'element'", objType)
	}

	categoryID, err := resolveCategoryByName(ctx, c, categoryName)
	if err != nil {
		return err
	}

	handle := strings.ToUpper(strings.ReplaceAll(namespace, "-", ""))
	if len(handle) > 5 {
		handle = handle[:5]
	}

	if dryRun {
		fmt.Println(ui.WarningStyle.Render("Dry run:"))
		fmt.Printf("  Name:      %s\n", name)
		fmt.Printf("  Namespace: %s\n", namespace)
		fmt.Printf("  Handle:    %s\n", handle)
		fmt.Printf("  Type:      %s\n", aspectType)
		fmt.Printf("  Category:  %s (%s)\n", categoryName, categoryID)
		if description != "" {
			fmt.Printf("  Desc:      %s\n", description)
		}
		return nil
	}

	if !isJSONOutput(cmd) {
		fmt.Println(ui.InfoStyle.Render(fmt.Sprintf("⠿ Creating %s %q...", objType, name)))
	}

	input := client.AspectInput{
		Name:       &name,
		Namespace:  &namespace,
		Handle:     &handle,
		CategoryId: &categoryID,
		Type:       &aspectType,
	}
	if description != "" {
		input.Description = &description
	}

	resp, err := client.CreateAspect(ctx, c.Genqlient(), input)
	if err != nil {
		return fmt.Errorf("failed to create %s: %w", objType, err)
	}

	created := resp.GetAspectCreate()
	result := map[string]string{
		"id":   created.GetId(),
		"name": created.GetName(),
		"type": objType,
	}

	if isJSONOutput(cmd) {
		return outputJSON(result)
	}

	fmt.Println(ui.SuccessStyle.Render(fmt.Sprintf("Created %s:", objType)) + fmt.Sprintf(" %s (%s)", created.GetName(), created.GetId()))
	return nil
}

func resolveCategoryByName(ctx context.Context, c *client.Client, name string) (string, error) {
	categories, err := discovery.ListCategories(ctx, c)
	if err != nil {
		return "", fmt.Errorf("failed to list categories: %w", err)
	}

	for _, cat := range categories {
		if strings.EqualFold(cat.Name, name) {
			return cat.ID, nil
		}
	}

	available := make([]string, len(categories))
	for i, cat := range categories {
		available[i] = cat.Name
	}
	return "", fmt.Errorf("category %q not found. Available: %s", name, strings.Join(available, ", "))
}
