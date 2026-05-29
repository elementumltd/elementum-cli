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

	"github.com/elementumltd/elementum-cli/auth"
	"github.com/elementumltd/elementum-cli/internal/client"
	"github.com/elementumltd/elementum-cli/logger"
	"github.com/elementumltd/elementum-cli/ui"
	"github.com/spf13/cobra"
)

var deploymentsConfigureCmd = &cobra.Command{
	Use:   "configure <deployment-id>",
	Short: "Configure missing datasets for a deployment",
	Long: `Configure missing dataset mappings for a deployment that is in CONFIGURATION state.

Reads a JSON config file that specifies cloud link, Snowflake mapping, and field
column mappings for each dataset that needs configuration.

Example config file:
  {
    "datasets": [
      {
        "id": "element-uuid",
        "cloud_link_id": "cloudlink-uuid",
        "snowflake_mapping": {
          "database_name": "MY_DB",
          "schema_name": "MY_SCHEMA",
          "table_name": "MY_TABLE"
        },
        "fields": [
          { "id": "field-uuid", "name": "Title", "type": "TEXT", "column_name": "TITLE", "semantic_tags": ["TITLE"] },
          { "id": "field-uuid", "name": "ID", "type": "TEXT", "column_name": "ID", "semantic_tags": ["HANDLE"] }
        ]
      }
    ]
  }

Examples:
  ei deployments configure abc-123 --config config.json
  ei deployments configure abc-123 --config config.json --dry-run
  ei deployments configure abc-123 --config config.json --json`,
	Args: cobra.ExactArgs(1),
	RunE: runDeploymentsConfigure,
}

// datasetConfig represents a single dataset configuration in the config file.
type datasetConfig struct {
	ID               string                  `json:"id"`
	CloudLinkID      string                  `json:"cloud_link_id"`
	SnowflakeMapping *snowflakeMappingConfig `json:"snowflake_mapping"`
	Fields           []fieldConfig           `json:"fields"`
}

type snowflakeMappingConfig struct {
	DatabaseName string `json:"database_name"`
	SchemaName   string `json:"schema_name"`
	TableName    string `json:"table_name"`
}

type fieldConfig struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	Type         string   `json:"type"`
	ColumnName   string   `json:"column_name"`
	SemanticTags []string `json:"semantic_tags"`
}

type configFile struct {
	Datasets []datasetConfig `json:"datasets"`
}

func init() {
	deploymentsConfigureCmd.Flags().String("config", "", "Path to JSON config file (required)")
	deploymentsConfigureCmd.Flags().Bool("dry-run", false, "Show what would be configured without applying")
	_ = deploymentsConfigureCmd.MarkFlagRequired("config")
}

func runDeploymentsConfigure(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	deploymentID := args[0]

	apiClient, err := auth.GetClientFromCmd(cmd)
	if err != nil {
		return err
	}

	configPath, _ := cmd.Flags().GetString("config")
	dryRun, _ := cmd.Flags().GetBool("dry-run")

	logger.Debug("configuring deployment", "deployment_id", deploymentID, "config", configPath, "dry_run", dryRun)

	// Read and parse config file
	data, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg configFile
	if err := json.Unmarshal(data, &cfg); err != nil {
		return fmt.Errorf("failed to parse config file: %w", err)
	}

	if len(cfg.Datasets) == 0 {
		return fmt.Errorf("config file contains no datasets; use 'ei deployments show <id> --json' to get a config template")
	}

	if dryRun {
		fmt.Println(ui.WarningStyle.Render("Dry run — no changes will be made"))
		fmt.Println()
		for i, ds := range cfg.Datasets {
			fmt.Printf("Dataset %d: %s\n", i+1, ds.ID)
			fmt.Printf("  Cloud Link: %s\n", ds.CloudLinkID)
			if ds.SnowflakeMapping != nil {
				fmt.Printf("  Snowflake:  %s.%s.%s\n",
					ds.SnowflakeMapping.DatabaseName,
					ds.SnowflakeMapping.SchemaName,
					ds.SnowflakeMapping.TableName)
			}
			fmt.Printf("  Fields:     %d\n", len(ds.Fields))
			for _, f := range ds.Fields {
				fmt.Printf("    %s (%s) → %s\n", f.Name, f.Type, f.ColumnName)
			}
			fmt.Println()
		}
		return nil
	}

	if !isJSONOutput(cmd) {
		fmt.Println(ui.InfoStyle.Render(fmt.Sprintf("Configuring %d dataset(s) for deployment %s...", len(cfg.Datasets), ui.Truncate(deploymentID, 12))))
	}

	type configResult struct {
		DatasetID  string `json:"dataset_id"`
		Success    bool   `json:"success"`
		Error      string `json:"error,omitempty"`
		FieldCount int    `json:"field_count,omitempty"`
	}

	var results []configResult

	for _, ds := range cfg.Datasets {
		// Build AspectInput for the dataset
		input := client.AspectInput{
			CloudLinkId: &ds.CloudLinkID,
		}

		// Set Snowflake mapping if provided
		if ds.SnowflakeMapping != nil {
			input.CloudMapping = &client.AspectCloudMappingInput{
				SnowflakeMapping: &client.SnowflakeMappingInput{
					DatabaseName: ds.SnowflakeMapping.DatabaseName,
					SchemaName:   ds.SnowflakeMapping.SchemaName,
					TableName:    ds.SnowflakeMapping.TableName,
				},
			}
		}

		// Build fields
		if len(ds.Fields) > 0 {
			var fields []client.AspectFieldInput
			for _, f := range ds.Fields {
				field := client.AspectFieldInput{
					Name:       f.Name,
					Type:       client.AspectFieldType(f.Type),
					ColumnName: &f.ColumnName,
				}
				if f.ID != "" {
					field.Id = &f.ID
				}
				for _, tag := range f.SemanticTags {
					field.SemanticTags = append(field.SemanticTags, client.AspectFieldSemanticTag(tag))
				}
				fields = append(fields, field)
			}
			input.Fields = fields
		}

		logger.Debug("configuring dataset", "dataset_id", ds.ID, "cloud_link_id", ds.CloudLinkID, "field_count", len(ds.Fields))
		_, err := client.ConfigureDataset(ctx, apiClient.Genqlient(), ds.ID, input)
		if err != nil {
			results = append(results, configResult{
				DatasetID: ds.ID,
				Success:   false,
				Error:     err.Error(),
			})
			if !isJSONOutput(cmd) {
				fmt.Printf("  %s %s: %s\n", ui.RenderCross(), ds.ID, err.Error())
			}
			continue
		}

		results = append(results, configResult{
			DatasetID:  ds.ID,
			Success:    true,
			FieldCount: len(ds.Fields),
		})
		if !isJSONOutput(cmd) {
			fmt.Printf("  %s %s: configured with %d fields\n",
				ui.RenderCheckmark(), ds.ID, len(ds.Fields))
		}
	}

	if isJSONOutput(cmd) {
		return outputJSON(results)
	}

	// Summary
	successCount := 0
	for _, r := range results {
		if r.Success {
			successCount++
		}
	}

	fmt.Println()
	if successCount == len(results) {
		fmt.Printf("%s All %d dataset(s) configured successfully.\n",
			ui.SuccessStyle.Render("✓"), successCount)
	} else {
		fmt.Printf("%s %d/%d dataset(s) configured. %d failed.\n",
			ui.WarningStyle.Render("!"), successCount, len(results), len(results)-successCount)
	}
	fmt.Println()

	return nil
}
