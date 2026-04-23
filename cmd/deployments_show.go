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
	"github.com/elementumltd/elementum-cli/logger"
	"github.com/elementumltd/elementum-cli/ui"
	"github.com/spf13/cobra"
)

var deploymentsShowCmd = &cobra.Command{
	Use:   "show <deployment-id>",
	Short: "Show deployment details",
	Long: `Show detailed information about a deployment including status,
deployed entities, and missing configurations.

When used with --json, includes a config_template that can be used as a
starting point for ei deployments configure.

Examples:
  ei deployments show abc-123-uuid
  ei deployments show abc-123-uuid --json`,
	Args: cobra.ExactArgs(1),
	RunE: runDeploymentsShow,
}

// deploymentDetail holds the full deployment response from ExecuteInto.
type deploymentDetail struct {
	ID        string   `json:"id"`
	Name      *string  `json:"name"`
	Status    string   `json:"status"`
	Message   *string  `json:"message"`
	Errors    []string `json:"errors"`
	CreatedAt string   `json:"createdAt"`
	UpdatedAt string   `json:"updatedAt"`
	CreatedBy *struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"createdBy"`
	SourceEnvironment struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"sourceEnvironment"`
	TargetEnvironment struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"targetEnvironment"`
	App struct {
		ID        string `json:"id"`
		Name      string `json:"name"`
		Namespace string `json:"namespace"`
	} `json:"app"`
	AsyncTask *struct {
		ID             string `json:"id"`
		Status         string `json:"status"`
		CompletedSteps int    `json:"completedSteps"`
		TotalSteps     int    `json:"totalSteps"`
	} `json:"asyncTask"`
	DeployedEntities []struct {
		ID       string `json:"id"`
		IsSource bool   `json:"isSource"`
		Entity   struct {
			Typename  string `json:"__typename"`
			ID        string `json:"id"`
			Name      string `json:"name"`
			Namespace string `json:"namespace"`
		} `json:"entity"`
	} `json:"deployedEntities"`
	MissingConfigurations []missingConfig `json:"missingConfigurations"`
}

type missingConfig struct {
	Typename   string `json:"__typename"`
	ID         string `json:"id"`
	Name       string `json:"name"`
	Configured *bool  `json:"configured"`
	CloudLink  *struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"cloudLink"`
	CloudConnection *struct {
		Typename     string  `json:"__typename"`
		DatabaseName *string `json:"databaseName"`
		SchemaName   *string `json:"schemaName"`
		TableName    *string `json:"tableName"`
	} `json:"cloudConnection"`
	Fields *struct {
		Edges []struct {
			Node struct {
				ID                   string   `json:"id"`
				Name                 string   `json:"name"`
				Typename             string   `json:"__typename"`
				SemanticTags         []string `json:"semanticTags"`
				System               bool     `json:"system"`
				CloudFieldConnection *struct {
					ColumnName *string `json:"columnName"`
				} `json:"cloudFieldConnection"`
			} `json:"node"`
		} `json:"edges"`
	} `json:"fields"`
}

func runDeploymentsShow(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	deploymentID := args[0]

	apiClient, err := auth.GetClientFromCmd(cmd)
	if err != nil {
		return err
	}

	if !isJSONOutput(cmd) {
		fmt.Println(ui.InfoStyle.Render("Loading deployment details..."))
	}

	// Use ExecuteInto due to EnvironmentDeploymentConfiguration union with 20+ members.
	// For AspectElement, fetch fields + cloud connection so --json can produce a config template.
	var result struct {
		Organization struct {
			Deployment *deploymentDetail `json:"deployment"`
		} `json:"organization"`
	}

	query := `
		query GetDeployment($id: ID!) {
			organization {
				deployment(id: $id) {
					id
					name
					status
					message
					errors
					createdAt
					updatedAt
					createdBy { id name }
					sourceEnvironment { id name }
					targetEnvironment { id name }
					app {
						id name
						... on AspectApp { namespace }
					}
					asyncTask {
						id status completedSteps totalSteps
					}
					deployedEntities {
						id
						isSource
						entity {
							__typename
							... on AspectApp { id name namespace }
							... on AspectElement { id name namespace }
							... on AspectTask { id name namespace }
							... on AspectTransaction { id name }
							... on Table { id name }
						}
					}
					missingConfigurations {
						__typename
						... on AspectElement {
							id name configured
							cloudLink { id name }
							cloudConnection {
								__typename
								... on SnowflakeConnection { databaseName schemaName tableName }
							}
							fields {
								edges {
									node {
										id name __typename semanticTags system
										cloudFieldConnection { columnName }
									}
								}
							}
						}
						... on Table { id name }
						... on StoredSnowflakeFunction { id name }
						... on Procedure { id name }
						... on AiProviderUnconfigured { id name }
						... on BotConfigurationUnconfigured { id name }
						... on PhoneProviderUnconfigured { id name }
						... on PhoneServiceUnconfigured { id name }
						... on AiOpenAiProvider { id name }
						... on AiElementumProvider { id name }
						... on AiElementumOpenAiProvider { id name }
						... on AiBedrockProvider { id name }
						... on AiGeminiProvider { id name }
						... on AiElementumGeminiProvider { id name }
						... on AiSnowflakeProvider { id name }
						... on AiElementumSnowflakeProvider { id name }
						... on AiDeepgramProvider { id name }
						... on AiElementumDeepgramProvider { id name }
						... on AiElevenLabProvider { id name }
						... on AiElementumElevenLabProvider { id name }
						... on PhoneProviderElementum { id name }
						... on PhoneProviderElementumTwilio { id name }
						... on PhoneProviderSipTrunk { id name }
						... on PhoneProviderTwilio { id name }
						... on PhoneServiceElementum { id name }
						... on PhoneServiceSipTrunk { id name }
						... on PhoneServiceTwilio { id name }
					}
				}
			}
		}
	`

	logger.Debug("fetching deployment", "id", deploymentID)
	err = apiClient.ExecuteInto(ctx, query, map[string]any{"id": deploymentID}, &result)
	if err != nil {
		return fmt.Errorf("failed to get deployment: %w", err)
	}

	d := result.Organization.Deployment
	if d == nil {
		return fmt.Errorf("deployment %q not found", deploymentID)
	}

	if isJSONOutput(cmd) {
		return outputDeploymentJSON(d)
	}

	// Header
	fmt.Println()
	name := d.ID
	if d.Name != nil && *d.Name != "" {
		name = *d.Name
	}
	fmt.Println(ui.TitleStyle.Render(fmt.Sprintf("Deployment: %s", name)))
	fmt.Println()

	// Status
	fmt.Printf("  Status:      %s\n", styledDeploymentStatus(d.Status))
	fmt.Printf("  App:         %s (%s)\n", d.App.Name, d.App.Namespace)
	fmt.Printf("  Source:      %s\n", d.SourceEnvironment.Name)
	fmt.Printf("  Target:      %s\n", d.TargetEnvironment.Name)
	if d.Message != nil && *d.Message != "" {
		fmt.Printf("  Message:     %s\n", *d.Message)
	}
	if d.CreatedBy != nil {
		fmt.Printf("  Created by:  %s\n", d.CreatedBy.Name)
	}
	fmt.Printf("  Created:     %s\n", formatTimestamp(d.CreatedAt))
	fmt.Printf("  Updated:     %s\n", formatTimestamp(d.UpdatedAt))

	if d.AsyncTask != nil {
		fmt.Printf("  Progress:    %d/%d steps\n", d.AsyncTask.CompletedSteps, d.AsyncTask.TotalSteps)
	}

	// Errors
	if len(d.Errors) > 0 {
		fmt.Println()
		fmt.Println(ui.ErrorStyle.Render("  Errors:"))
		for _, e := range d.Errors {
			fmt.Printf("    %s %s\n", ui.RenderCross(), e)
		}
	}

	// Deployed entities
	if len(d.DeployedEntities) > 0 {
		fmt.Println()
		fmt.Println(ui.SubtitleStyle.Render("Deployed Entities"))
		fmt.Println()

		table := ui.NewTable([]string{"TYPE", "NAME", "SOURCE", "ID"})
		for _, entity := range d.DeployedEntities {
			e := entity.Entity
			source := ""
			if entity.IsSource {
				source = "yes"
			}
			table.AddRow(e.Typename, e.Name, source, e.ID)
		}
		fmt.Println(table.Render())
	}

	// Missing configurations
	if len(d.MissingConfigurations) > 0 {
		fmt.Println()
		fmt.Println(ui.SubtitleStyle.Render("Missing Configurations"))
		fmt.Println()

		table := ui.NewTable([]string{"TYPE", "NAME", "CLOUD LINK", "CONFIGURED", "ID"})
		for _, mc := range d.MissingConfigurations {
			cloudLink := ""
			if mc.CloudLink != nil {
				cloudLink = mc.CloudLink.Name
			}
			configured := ""
			if mc.Configured != nil {
				if *mc.Configured {
					configured = ui.SuccessStyle.Render("yes")
				} else {
					configured = ui.ErrorStyle.Render("no")
				}
			}
			table.AddRow(mc.Typename, mc.Name, cloudLink, configured, mc.ID)
		}
		fmt.Println(table.Render())

		fmt.Println()
		fmt.Printf("  %s Use %s to get JSON with a ready-to-use config template.\n",
			ui.RenderBullet(), ui.InfoStyle.Render("ei deployments show "+deploymentID+" --json"))
		fmt.Printf("  %s Use %s to apply configurations.\n",
			ui.RenderBullet(), ui.InfoStyle.Render("ei deployments configure "+deploymentID+" --config <file.json>"))
	}

	fmt.Println()

	return nil
}

// outputDeploymentJSON outputs the deployment with an additional config_template
// that can be used as a starting point for `ei deployments configure`.
func outputDeploymentJSON(d *deploymentDetail) error {
	datasets := buildConfigTemplate(d.MissingConfigurations)

	output := struct {
		*deploymentDetail
		ConfigTemplate *configFile `json:"config_template,omitempty"`
	}{
		deploymentDetail: d,
	}

	if len(datasets) > 0 {
		output.ConfigTemplate = &configFile{Datasets: datasets}
	}

	return outputJSON(output)
}

// typenameToFieldType maps GraphQL __typename (stripped of "Aspect" prefix and "Field" suffix) to AspectFieldType values.
var typenameToFieldType = map[string]string{
	"Text":          "TEXT",
	"Number":        "NUMBER",
	"Decimal":       "DECIMAL",
	"Boolean":       "BOOLEAN",
	"Date":          "DATE",
	"DateTime":      "DATETIME",
	"Picklist":      "PICKLIST",
	"MultiPicklist": "MULTI_PICKLIST",
	"User":          "USER_ID",
	"Group":         "GROUP_ID",
	"Html":          "HTML",
	"Json":          "JSON",
	"Attachment":    "ATTACHMENT_ID",
}

// fieldTypenameToType maps GraphQL __typename to AspectFieldType values.
func fieldTypenameToType(typename string) string {
	// Strip "Aspect" prefix and "Field" suffix
	t := strings.TrimPrefix(typename, "Aspect")
	t = strings.TrimSuffix(t, "Field")
	if fieldType, ok := typenameToFieldType[t]; ok {
		return fieldType
	}
	return "TEXT" // default fallback
}

// buildConfigTemplate creates a config template from missing configurations.
// Only AspectElement entries (datasets) produce template entries.
func buildConfigTemplate(configs []missingConfig) []datasetConfig {
	var datasets []datasetConfig

	for _, mc := range configs {
		if mc.Typename != "AspectElement" {
			continue
		}

		ds := datasetConfig{
			ID:          mc.ID,
			CloudLinkID: "<CLOUD_LINK_ID>",
		}

		// Pre-fill cloud link ID from source if available
		if mc.CloudLink != nil {
			ds.CloudLinkID = mc.CloudLink.ID
		}

		// Pre-fill Snowflake mapping from source cloud connection
		if mc.CloudConnection != nil && mc.CloudConnection.Typename == "SnowflakeConnection" {
			ds.SnowflakeMapping = &snowflakeMappingConfig{
				DatabaseName: derefOr(mc.CloudConnection.DatabaseName, "<DATABASE_NAME>"),
				SchemaName:   derefOr(mc.CloudConnection.SchemaName, "<SCHEMA_NAME>"),
				TableName:    derefOr(mc.CloudConnection.TableName, "<TABLE_NAME>"),
			}
		} else {
			ds.SnowflakeMapping = &snowflakeMappingConfig{
				DatabaseName: "<DATABASE_NAME>",
				SchemaName:   "<SCHEMA_NAME>",
				TableName:    "<TABLE_NAME>",
			}
		}

		// Build fields from the element's field list
		if mc.Fields != nil {
			for _, edge := range mc.Fields.Edges {
				f := edge.Node
				if f.System {
					continue // Skip system fields
				}

				fc := fieldConfig{
					ID:   f.ID,
					Name: f.Name,
					Type: fieldTypenameToType(f.Typename),
				}

				// Use existing column mapping if available
				if f.CloudFieldConnection != nil && f.CloudFieldConnection.ColumnName != nil {
					fc.ColumnName = *f.CloudFieldConnection.ColumnName
				}

				for _, tag := range f.SemanticTags {
					fc.SemanticTags = append(fc.SemanticTags, tag)
				}

				ds.Fields = append(ds.Fields, fc)
			}
		}

		datasets = append(datasets, ds)
	}

	return datasets
}

// derefOr dereferences a string pointer, returning fallback if nil.
func derefOr(s *string, fallback string) string {
	if s != nil {
		return *s
	}
	return fallback
}
