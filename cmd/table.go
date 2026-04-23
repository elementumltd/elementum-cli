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
	"os"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/elementumltd/elementum-cli/auth"
	"github.com/elementumltd/elementum-cli/discovery"
	"github.com/elementumltd/elementum-cli/internal/client"
	"github.com/elementumltd/elementum-cli/logger"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var tableCmd = &cobra.Command{
	Use:   "table",
	Short: "Table management commands",
	Long:  "Commands for managing Elementum tables.",
}

var tableCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a multi-join table",
	Long: `Generate Terraform HCL for a multi-join Elementum table.

This command resolves human-readable names (apps, tables, fields, categories)
to UUIDs and generates ready-to-use Terraform configuration.

For CloudLink sources, use dot notation: "CloudLink Name.DATABASE.SCHEMA.TABLE"
The command will automatically check if base tables exist or generate them.

Input Modes:
  - Interactive: Run without --config for guided prompts
  - Config file: Provide a YAML file with --config

Output Modes:
  - Terraform HCL (default): Generate .tf file for use with 'ei apply'
  - curl command (--curl): Output curl command for direct API creation

Examples:
  # Interactive mode
  ei table create

  # From YAML config file
  ei table create --config table.yaml

  # Output to specific file
  ei table create --config table.yaml --output my-table.tf

  # Generate curl command instead of Terraform
  ei table create --config table.yaml --curl

YAML Config Format:
  name: "My Join Table"
  handle: "my_join_table"
  category: "Tables"
  source: "Snowflake Prod.ANALYTICS.PUBLIC.ORDERS"
  
  fields:
    - source: "Snowflake Prod.ANALYTICS.PUBLIC.ORDERS"
      columns:
        - name: "ORDER_ID"
        - name: "CUSTOMER_ID"
          alias: "Customer ID"
    - source: "Snowflake Prod.ANALYTICS.PUBLIC.CUSTOMERS"
      columns:
        - name: "NAME"
          alias: "Customer Name"
  
  joins:
    - source: "Snowflake Prod.ANALYTICS.PUBLIC.CUSTOMERS"
      type: INNER
      on:
        - left_field: "CUSTOMER_ID"
          right_field: "ID"`,
	RunE: runTableCreate,
}

func init() {
	tableCmd.AddCommand(tableCreateCmd)

	tableCreateCmd.Flags().String("config", "", "YAML config file path")
	tableCreateCmd.Flags().StringP("output", "o", "", "Output file path (default: stdout)")
	tableCreateCmd.Flags().Bool("curl", false, "Output curl command instead of Terraform HCL")
	tableCreateCmd.Flags().Bool("json", false, "Output resolved config as JSON (for debugging)")
}

// GetTableCmd returns the table command for registration
func GetTableCmd() *cobra.Command {
	return tableCmd
}

func runTableCreate(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Get authenticated client
	c, err := auth.GetClientFromCmd(cmd)
	if err != nil {
		return err
	}

	// Get flags
	configFile, _ := cmd.Flags().GetString("config")
	outputFile, _ := cmd.Flags().GetString("output")
	curlMode, _ := cmd.Flags().GetBool("curl")
	jsonMode, _ := cmd.Flags().GetBool("json")

	var config *discovery.TableCreateConfig

	if configFile != "" {
		// Load from YAML file
		config, err = loadTableConfig(configFile)
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}
	} else {
		// Interactive mode
		config, err = collectTableConfigInteractively(ctx, c)
		if err != nil {
			return fmt.Errorf("interactive mode failed: %w", err)
		}
	}

	// Resolve all names to IDs
	resolver := discovery.NewTableResolver(ctx, c)
	resolved, err := resolver.ResolveConfig(config)
	if err != nil {
		return fmt.Errorf("failed to resolve config: %w", err)
	}

	// Generate output
	var output string
	if jsonMode {
		// Debug JSON output
		output = formatResolvedConfigJSON(resolved)
	} else if curlMode {
		// Curl command output
		baseURL := c.GetRESTBaseURL()
		token, err := c.GetAccessToken(ctx)
		if err != nil {
			return fmt.Errorf("failed to get access token: %w", err)
		}
		output = discovery.GenerateCurlCommand(resolved, baseURL, token)
	} else {
		// Terraform HCL output (default)
		output = discovery.GenerateTerraformHCL(resolved)
	}

	// Write output
	if outputFile != "" {
		if err := os.WriteFile(outputFile, []byte(output), 0644); err != nil {
			return fmt.Errorf("failed to write output file: %w", err)
		}
		fmt.Printf("Output written to: %s\n", outputFile)
	} else {
		fmt.Println(output)
	}

	return nil
}

func loadTableConfig(path string) (*discovery.TableCreateConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	var config discovery.TableCreateConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	// Validate required fields
	if config.Name == "" {
		return nil, fmt.Errorf("name is required")
	}
	if config.Handle == "" {
		return nil, fmt.Errorf("handle is required")
	}
	if config.Category == "" {
		return nil, fmt.Errorf("category is required")
	}
	if config.Source == "" {
		return nil, fmt.Errorf("source is required")
	}
	if len(config.Fields) == 0 {
		return nil, fmt.Errorf("at least one field group is required")
	}

	return &config, nil
}

func collectTableConfigInteractively(ctx context.Context, c *client.Client) (*discovery.TableCreateConfig, error) {
	logger.Info("Starting interactive table creation...")

	resolver := discovery.NewTableResolver(ctx, c)

	config := &discovery.TableCreateConfig{}

	// Collect basic info
	var name, handle, description string
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewNote().
				Title("Create Multi-Join Table").
				Description("This wizard will guide you through creating a multi-join table."),
			huh.NewInput().
				Title("Table Name").
				Description("Human-readable name for the table").
				Value(&name).
				Validate(func(s string) error {
					if s == "" {
						return fmt.Errorf("name is required")
					}
					return nil
				}),
			huh.NewInput().
				Title("Table Handle").
				Description("Short identifier (e.g., 'my_table')").
				Value(&handle).
				Validate(func(s string) error {
					if s == "" {
						return fmt.Errorf("handle is required")
					}
					return nil
				}),
			huh.NewInput().
				Title("Description (optional)").
				Value(&description),
		),
	)
	if err := form.Run(); err != nil {
		return nil, err
	}
	config.Name = name
	config.Handle = handle
	config.Description = description

	// Select category
	categories, err := resolver.GetCategories()
	if err != nil {
		return nil, fmt.Errorf("failed to load categories: %w", err)
	}

	categoryOptions := make([]huh.Option[string], len(categories))
	for i, cat := range categories {
		categoryOptions[i] = huh.NewOption(cat.Name, cat.Name)
	}

	var selectedCategory string
	categoryForm := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Select Category").
				Options(categoryOptions...).
				Value(&selectedCategory),
		),
	)
	if err := categoryForm.Run(); err != nil {
		return nil, err
	}
	config.Category = selectedCategory

	// Collect source
	var sourceInput string
	sourceForm := huh.NewForm(
		huh.NewGroup(
			huh.NewNote().
				Title("Primary Source").
				Description("Enter an app name, table name, or CloudLink path.\nFor CloudLink: 'CloudLink Name.DATABASE.SCHEMA.TABLE'"),
			huh.NewInput().
				Title("Source").
				Value(&sourceInput).
				Validate(func(s string) error {
					if s == "" {
						return fmt.Errorf("source is required")
					}
					return nil
				}),
		),
	)
	if err := sourceForm.Run(); err != nil {
		return nil, err
	}
	config.Source = sourceInput

	// Collect fields for primary source
	primaryFields, err := collectTableFieldsInteractively(sourceInput)
	if err != nil {
		return nil, err
	}
	config.Fields = append(config.Fields, discovery.FieldGroup{
		Source:  sourceInput,
		Columns: primaryFields,
	})

	// Collect joins
	for {
		var addJoin bool
		joinPrompt := huh.NewForm(
			huh.NewGroup(
				huh.NewConfirm().
					Title("Add a join?").
					Value(&addJoin),
			),
		)
		if err := joinPrompt.Run(); err != nil {
			return nil, err
		}
		if !addJoin {
			break
		}

		join, fields, err := collectJoinInteractively(config)
		if err != nil {
			return nil, err
		}
		config.Joins = append(config.Joins, *join)
		if len(fields) > 0 {
			config.Fields = append(config.Fields, discovery.FieldGroup{
				Source:  join.Source,
				Columns: fields,
			})
		}
	}

	return config, nil
}

func collectTableFieldsInteractively(sourceName string) ([]discovery.ColumnConfig, error) {
	// For now, collect field names manually
	var fieldNames string
	fieldForm := huh.NewForm(
		huh.NewGroup(
			huh.NewNote().
				Title(fmt.Sprintf("Fields from %q", sourceName)).
				Description("Enter column names, one per line.\nFormat: COLUMN_NAME or COLUMN_NAME -> Alias"),
			huh.NewText().
				Title("Columns").
				Value(&fieldNames).
				Validate(func(s string) error {
					if s == "" {
						return fmt.Errorf("at least one column is required")
					}
					return nil
				}),
		),
	)
	if err := fieldForm.Run(); err != nil {
		return nil, err
	}

	// Parse field names
	var columns []discovery.ColumnConfig
	lines := strings.Split(fieldNames, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Check for alias format: "COLUMN_NAME -> Alias"
		if idx := strings.Index(line, "->"); idx > 0 {
			name := strings.TrimSpace(line[:idx])
			alias := strings.TrimSpace(line[idx+2:])
			columns = append(columns, discovery.ColumnConfig{Name: name, Alias: alias})
		} else {
			columns = append(columns, discovery.ColumnConfig{Name: line})
		}
	}

	return columns, nil
}

func collectJoinInteractively(config *discovery.TableCreateConfig) (*discovery.JoinConfig, []discovery.ColumnConfig, error) {
	join := &discovery.JoinConfig{}

	// Collect join source
	var joinSource string
	sourceForm := huh.NewForm(
		huh.NewGroup(
			huh.NewNote().
				Title("Join Source").
				Description("Enter an app name, table name, or CloudLink path to join."),
			huh.NewInput().
				Title("Source to Join").
				Value(&joinSource).
				Validate(func(s string) error {
					if s == "" {
						return fmt.Errorf("source is required")
					}
					return nil
				}),
		),
	)
	if err := sourceForm.Run(); err != nil {
		return nil, nil, err
	}
	join.Source = joinSource

	// Select join type
	var joinType string
	typeForm := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Join Type").
				Options(
					huh.NewOption("INNER", "INNER"),
					huh.NewOption("LEFT", "LEFT"),
					huh.NewOption("RIGHT", "RIGHT"),
					huh.NewOption("FULL", "FULL"),
				).
				Value(&joinType),
		),
	)
	if err := typeForm.Run(); err != nil {
		return nil, nil, err
	}
	join.Type = joinType

	// Collect join condition
	var leftField, rightField string
	conditionForm := huh.NewForm(
		huh.NewGroup(
			huh.NewNote().
				Title("Join Condition").
				Description("Specify which fields to join on."),
			huh.NewInput().
				Title("Left Field (from existing sources)").
				Description("Field name from primary source or previously joined sources").
				Value(&leftField).
				Validate(func(s string) error {
					if s == "" {
						return fmt.Errorf("left field is required")
					}
					return nil
				}),
			huh.NewInput().
				Title(fmt.Sprintf("Right Field (from %q)", joinSource)).
				Value(&rightField).
				Validate(func(s string) error {
					if s == "" {
						return fmt.Errorf("right field is required")
					}
					return nil
				}),
		),
	)
	if err := conditionForm.Run(); err != nil {
		return nil, nil, err
	}

	join.On = []discovery.JoinOnConfig{{
		LeftField:  leftField,
		RightField: rightField,
	}}

	// Collect fields from join source
	fields, err := collectTableFieldsInteractively(joinSource)
	if err != nil {
		return nil, nil, err
	}

	return join, fields, nil
}

func formatResolvedConfigJSON(resolved *discovery.ResolvedConfig) string {
	// Create a simplified view for JSON output
	output := map[string]interface{}{
		"config":     resolved.Config,
		"categoryId": resolved.CategoryID,
		"sources":    make(map[string]interface{}),
		"fields":     resolved.FieldsToCreate,
		"joins":      resolved.JoinsToCreate,
	}

	for name, source := range resolved.Sources {
		output["sources"].(map[string]interface{})[name] = map[string]interface{}{
			"id":              source.ID,
			"type":            source.Type,
			"terraformName":   source.TerraformName,
			"needsGeneration": source.NeedsGeneration,
			"fieldCount":      len(source.Fields),
		}
	}

	data, _ := yaml.Marshal(output)
	return string(data)
}
