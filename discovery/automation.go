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

package discovery

import (
	"context"
	"fmt"

	"github.com/elementumltd/elementum-cli/logger"
	"github.com/elementumltd/elementum-cli/internal/client"
)

// AutomationDiscoveryContext tracks visited resources for cycle detection
type AutomationDiscoveryContext struct {
	VisitedApps           map[string]bool
	VisitedElements       map[string]bool
	VisitedDatamines      map[string]bool
	VisitedTables         map[string]bool
	VisitedDocumentModels map[string]bool
	ReferencedApps        []*App
	ReferencedElements    []*Element
	ReferencedDatamines   []*Datamine
	ReferencedTables      []*Table
	ReferencedFileReaders []*FileReader
}

// NewAutomationDiscoveryContext creates a new context for automation discovery
func NewAutomationDiscoveryContext() *AutomationDiscoveryContext {
	return &AutomationDiscoveryContext{
		VisitedApps:           make(map[string]bool),
		VisitedElements:       make(map[string]bool),
		VisitedDatamines:      make(map[string]bool),
		VisitedTables:         make(map[string]bool),
		VisitedDocumentModels: make(map[string]bool),
		ReferencedApps:        make([]*App, 0),
		ReferencedElements:    make([]*Element, 0),
		ReferencedDatamines:   make([]*Datamine, 0),
		ReferencedTables:      make([]*Table, 0),
		ReferencedFileReaders: make([]*FileReader, 0),
	}
}

// DiscoverAutomationDependencies finds cross-app/element references in automations
// and recursively discovers all dependent resources with cycle detection
func DiscoverAutomationDependencies(ctx context.Context, c *client.Client, app *App, visited *AutomationDiscoveryContext) error {
	// Mark this app as visited
	visited.VisitedApps[app.ID] = true

	// Check all automations for cross-app/element references and datamine triggers
	for _, automation := range app.Automations {
		// Check triggers for datamine references
		for _, trigger := range automation.Triggers {
			if trigger.Type == "datamine" && trigger.DatamineID != "" {
				// Skip if already visited
				if visited.VisitedDatamines[trigger.DatamineID] {
					continue
				}

				// Discover datamine and its dependencies
				err := DiscoverDatamineDependencies(ctx, c, trigger.DatamineID, visited)
				if err != nil {
					logger.Warn("failed to discover datamine (may be inaccessible)", "datamineID", trigger.DatamineID, "error", err)
					continue
				}
			}
		}

		// Check tasks for cross-app/element references
		for _, task := range automation.Tasks {
			// Check if task references a different object (app or element)
			if task.ObjectID != "" && task.ObjectID != app.ID {
				// Skip if already visited
				if visited.VisitedApps[task.ObjectID] || visited.VisitedElements[task.ObjectID] {
					continue
				}

				// Try to fetch as an app first
				referencedApp, err := GetApp(ctx, c, task.ObjectID)
				if err == nil && referencedApp != nil {
					visited.VisitedApps[task.ObjectID] = true
					visited.ReferencedApps = append(visited.ReferencedApps, referencedApp)

					// Recurse into referenced app's automations
					err = DiscoverAutomationDependencies(ctx, c, referencedApp, visited)
					if err != nil {
						return fmt.Errorf("failed to discover dependencies for app %s: %w", task.ObjectID, err)
					}
					continue
				}

				// Try as element if app fetch failed
				referencedElement, err := GetElement(ctx, c, task.ObjectID)
				if err == nil && referencedElement != nil {
					visited.VisitedElements[task.ObjectID] = true
					visited.ReferencedElements = append(visited.ReferencedElements, referencedElement)
					continue
				}

				// If both failed, log but don't fail the entire discovery
				// The object might not exist or might not be accessible
			}

			// Check if task references a document model (ai_file_read and bulk_excel tasks)
			if task.DocumentModelID != "" {
				// Skip if already visited
				if visited.VisitedDocumentModels[task.DocumentModelID] {
					continue
				}

				// Mark as visited
				visited.VisitedDocumentModels[task.DocumentModelID] = true

				// Discover document model details - this will pinpoint any access issues
				fileReader, err := DiscoverDocumentModel(ctx, c, app.ID, task.DocumentModelID)
				if err != nil {
					logger.Warn("failed to discover document model (may be inaccessible)",
						"documentModelID", task.DocumentModelID,
						"taskID", task.ID,
						"taskName", task.Name,
						"error", err)
					continue
				}
				if fileReader != nil {
					visited.ReferencedFileReaders = append(visited.ReferencedFileReaders, fileReader)
				}
			}
		}
	}

	return nil
}

// DiscoverDatamineDependencies discovers a datamine and its table dependencies
// following the chain: Datamine -> Table -> App/Element (recursive)
func DiscoverDatamineDependencies(ctx context.Context, c *client.Client, datamineID string, visited *AutomationDiscoveryContext) error {
	// Mark datamine as visited
	visited.VisitedDatamines[datamineID] = true

	// First, we need to find the datamine. Since we need the table ID to query it,
	// we'll search all tables for this datamine
	datamine, table, err := findDatamineByID(ctx, c, datamineID)
	if err != nil {
		return fmt.Errorf("failed to find datamine %s: %w", datamineID, err)
	}

	visited.ReferencedDatamines = append(visited.ReferencedDatamines, datamine)

	// Now discover the table
	if table != nil && !visited.VisitedTables[table.ID] {
		visited.VisitedTables[table.ID] = true
		visited.ReferencedTables = append(visited.ReferencedTables, table)

		// If the table has a source app, discover that app
		if table.SourceID != "" && !visited.VisitedApps[table.SourceID] {
			sourceApp, err := GetApp(ctx, c, table.SourceID)
			if err == nil && sourceApp != nil {
				visited.VisitedApps[table.SourceID] = true
				visited.ReferencedApps = append(visited.ReferencedApps, sourceApp)

				// Recurse into the source app's automations
				err = DiscoverAutomationDependencies(ctx, c, sourceApp, visited)
				if err != nil {
					logger.Warn("failed to discover automation dependencies for source app", "appID", table.SourceID, "error", err)
				}
			}
		}
	}

	return nil
}

// findDatamineByID searches for a datamine by its ID using the direct organization.datamine query
func findDatamineByID(ctx context.Context, c *client.Client, datamineID string) (*Datamine, *Table, error) {
	// Use genqlient to query the datamine directly by ID
	resp, err := client.GetDatamineDirectly(ctx, c.Genqlient(), datamineID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to query datamine: %w", err)
	}

	if resp.Organization.Datamine == nil {
		return nil, nil, fmt.Errorf("datamine not found: %s", datamineID)
	}

	dm := resp.Organization.Datamine
	datamine := &Datamine{
		ID:     dm.Id,
		Name:   dm.Name,
		Active: dm.Active,
	}

	if dm.Table != nil {
		datamine.TableID = dm.Table.Id
		datamine.TableName = dm.Table.Name
	}

	for _, col := range dm.PrimaryColumns {
		if col != nil {
			datamine.PrimaryColumnIDs = append(datamine.PrimaryColumnIDs, (*col).GetId())
		}
	}

	// Build table struct
	var table *Table
	if dm.Table != nil {
		t := dm.Table
		table = &Table{
			ID:           t.Id,
			Name:         t.Name,
			Handle:       t.Handle,
			Type:         t.Type,
			Configured:   t.Configured,
			CloudManaged: t.CloudManaged,
			CategoryID:   t.Category.Id,
		}
		if t.Source != nil {
			table.SourceID = t.Source.Id
		}
		if cloudLink := t.GetCloudLink(); cloudLink != nil {
			table.CloudLinkID = (*cloudLink).GetId()
			table.CloudLinkName = (*cloudLink).GetName()
		}
		// Extract Snowflake connection fields
		if cloudConn := t.GetCloudConnection(); cloudConn != nil {
			if snowflake, ok := (*cloudConn).(*client.GetDatamineDirectlyOrganizationDatamineTableCloudConnectionSnowflakeConnection); ok {
				if snowflake.DatabaseName != nil {
					table.SnowflakeDatabaseName = *snowflake.DatabaseName
				}
				if snowflake.SchemaName != nil {
					table.SnowflakeSchemaName = *snowflake.SchemaName
				}
				if snowflake.TableName != nil {
					table.SnowflakeTableName = *snowflake.TableName
				}
			}
		}
		// Extract table fields (for datamine primary_column_ids beautification)
		for _, f := range t.GetFields() {
			table.Fields = append(table.Fields, TableField{
				ID:   f.GetId(),
				Name: f.GetName(),
			})
		}
	}

	return datamine, table, nil
}

// GetAllDependencies returns all apps and elements that need to be exported
// based on automation cross-references
func GetAllDependencies(ctx context.Context, c *client.Client, rootApp *App) (*AutomationDiscoveryContext, error) {
	visited := NewAutomationDiscoveryContext()

	err := DiscoverAutomationDependencies(ctx, c, rootApp, visited)
	if err != nil {
		return nil, err
	}

	return visited, nil
}

// DiscoverDocumentModel fetches document model details by ID
// This is called separately from the automation query to isolate access errors
// and pinpoint exactly which document model is causing issues
func DiscoverDocumentModel(ctx context.Context, c *client.Client, appID, documentModelID string) (*FileReader, error) {
	// First, query just the type to determine which detail function to use
	query := `
		query GetDocumentModelType($aspectId: ID!, $documentModelId: ID!) {
			organization {
				aspect(id: $aspectId) {
					... on AspectApp {
						documentModel(id: $documentModelId) {
							__typename
							id
						}
					}
				}
			}
		}
	`

	variables := map[string]interface{}{
		"aspectId":        appID,
		"documentModelId": documentModelID,
	}

	var result struct {
		Organization struct {
			Aspect struct {
				DocumentModel *struct {
					Typename string `json:"__typename"`
					ID       string `json:"id"`
				} `json:"documentModel"`
			} `json:"aspect"`
		} `json:"organization"`
	}

	err := c.ExecuteInto(ctx, query, variables, &result)
	if err != nil {
		return nil, fmt.Errorf("failed to query document model type: %w", err)
	}

	if result.Organization.Aspect.DocumentModel == nil {
		return nil, fmt.Errorf("document model not found: %s", documentModelID)
	}

	docModel := result.Organization.Aspect.DocumentModel

	logger.Debug("discovered document model",
		"id", docModel.ID,
		"type", docModel.Typename,
		"appID", appID,
	)

	// Return a minimal FileReader with just the type info
	// Full details will be fetched when exporting the app's AIFileReaders
	return &FileReader{
		ID:   docModel.ID,
		Type: docModel.Typename,
	}, nil
}
