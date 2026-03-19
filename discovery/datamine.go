// Copyright (c) ElementumAI, Inc.
// SPDX-License-Identifier: MPL-2.0

package discovery

import (
	"context"
	"fmt"

	"github.com/elementumltd/elementum-cli/internal/client"
)

// GetTable retrieves a table by its ID using genqlient
func GetTable(ctx context.Context, c *client.Client, tableID string) (*Table, error) {
	result, err := client.GetTable(ctx, c.Genqlient(), tableID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch table: %w", err)
	}

	tableData := client.ExtractTableFromGetTable(result)
	if tableData == nil {
		return nil, fmt.Errorf("table not found: %s", tableID)
	}

	table := &Table{
		ID:           tableData.ID,
		Name:         tableData.Name,
		Handle:       tableData.Handle,
		Type:         tableData.Type,
		Configured:   tableData.Configured,
		CloudManaged: tableData.CloudManaged,
	}

	if tableData.CategoryID != nil {
		table.CategoryID = *tableData.CategoryID
	}
	if tableData.SourceID != nil {
		table.SourceID = *tableData.SourceID
	}
	if tableData.CloudLinkID != nil {
		table.CloudLinkID = *tableData.CloudLinkID
	}
	if tableData.CloudLinkName != nil {
		table.CloudLinkName = *tableData.CloudLinkName
	}
	// Extract Snowflake connection fields
	if tableData.SnowflakeDatabaseName != nil {
		table.SnowflakeDatabaseName = *tableData.SnowflakeDatabaseName
	}
	if tableData.SnowflakeSchemaName != nil {
		table.SnowflakeSchemaName = *tableData.SnowflakeSchemaName
	}
	if tableData.SnowflakeTableName != nil {
		table.SnowflakeTableName = *tableData.SnowflakeTableName
	}

	// Extract table fields
	for _, fieldData := range tableData.Fields {
		field := TableField{
			ID:   fieldData.ID,
			Name: fieldData.Name,
			Type: fieldData.Type,
		}
		table.Fields = append(table.Fields, field)
	}

	// Fetch search tables for this table
	searchTables, err := GetSearchTablesForTable(ctx, c, tableID)
	if err == nil {
		table.SearchTables = searchTables
	}
	// Ignore errors - search tables are optional and may not exist

	return table, nil
}

// GetTableByName retrieves a table by its name using genqlient
func GetTableByName(ctx context.Context, c *client.Client, name string) (*Table, error) {
	filter := client.BuildTableNameFilter(name)
	result, err := client.GetTables(ctx, c.Genqlient(), filter)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch tables: %w", err)
	}

	tables := client.ExtractTablesFromGetTables(result)
	for _, t := range tables {
		if t.Name == name {
			// Get full table details
			return GetTable(ctx, c, t.ID)
		}
	}

	return nil, fmt.Errorf("table not found by name: %s", name)
}

// ListTables returns all tables in the organization
func ListTables(ctx context.Context, c *client.Client) ([]Table, error) {
	result, err := client.GetTables(ctx, c.Genqlient(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch tables: %w", err)
	}

	tablesData := client.ExtractTablesFromGetTables(result)
	var tables []Table
	for _, t := range tablesData {
		table := Table{
			ID:     t.ID,
			Name:   t.Name,
			Handle: t.Handle,
			Type:   t.Type,
		}
		tables = append(tables, table)
	}

	return tables, nil
}

// GetTableByHandle retrieves a table by its handle/namespace using genqlient
func GetTableByHandle(ctx context.Context, c *client.Client, handle string) (*Table, error) {
	// List all tables (no filter for handle, we filter in Go)
	result, err := client.GetTables(ctx, c.Genqlient(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch tables: %w", err)
	}

	tables := client.ExtractTablesFromGetTables(result)
	for _, t := range tables {
		if t.Handle == handle {
			// Get full table details
			return GetTable(ctx, c, t.ID)
		}
	}

	return nil, fmt.Errorf("table not found by handle: %s", handle)
}

// Datamine represents an Elementum Datamine
type Datamine struct {
	ID               string
	Name             string
	Description      string
	Active           bool
	TableID          string
	TableName        string
	PrimaryColumnIDs []string
	ScheduleType     string // "fixed" or "cron"
	TimeUnit         string // For fixed schedules
	Value            int64  // For fixed schedules
	CronExpression   string // For cron schedules
}

// GetDatamineByID retrieves a datamine by its table and ID using genqlient
func GetDatamineByID(ctx context.Context, c *client.Client, tableID, datamineID string) (*Datamine, error) {
	result, err := client.GetDatamine(ctx, c.Genqlient(), tableID, datamineID)
	if err != nil {
		return nil, fmt.Errorf("failed to get datamine: %w", err)
	}

	dmData := client.ExtractDatamine(result)
	if dmData == nil {
		return nil, fmt.Errorf("datamine not found: %s", datamineID)
	}

	datamine := &Datamine{
		ID:               dmData.ID,
		Name:             dmData.Name,
		Active:           dmData.Active,
		PrimaryColumnIDs: dmData.PrimaryColumnIDs,
	}

	if dmData.Description != nil {
		datamine.Description = *dmData.Description
	}
	if dmData.TableID != nil {
		datamine.TableID = *dmData.TableID
	}

	// Parse schedule from client.DatamineData
	if dmData.ScheduleFixed != nil {
		datamine.ScheduleType = "fixed"
		datamine.TimeUnit = dmData.ScheduleFixed.TimeUnit
		datamine.Value = int64(dmData.ScheduleFixed.Value)
	} else if dmData.ScheduleCron != nil {
		datamine.ScheduleType = "cron"
		datamine.CronExpression = dmData.ScheduleCron.CronExpression
	}

	return datamine, nil
}

// GetDatamineByName retrieves a datamine by name from a table
func GetDatamineByName(ctx context.Context, c *client.Client, tableID, name string) (*Datamine, error) {
	datamines, err := ListDatamines(ctx, c, tableID)
	if err != nil {
		return nil, err
	}

	for _, dm := range datamines {
		if dm.Name == name {
			return &dm, nil
		}
	}

	return nil, fmt.Errorf("datamine not found with name: %s", name)
}

// ListDatamines returns all datamines for a table using genqlient
func ListDatamines(ctx context.Context, c *client.Client, tableID string) ([]Datamine, error) {
	result, err := client.GetDatamines(ctx, c.Genqlient(), tableID, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list datamines: %w", err)
	}

	if result.Organization.Table.Datamines == nil {
		return nil, fmt.Errorf("table not found: %s", tableID)
	}

	datamines := []Datamine{}
	for _, edge := range result.Organization.Table.Datamines.Edges {
		node := edge.Node
		dm := Datamine{
			ID:     node.Id,
			Name:   node.Name,
			Active: node.Active,
		}

		if node.Description != nil {
			dm.Description = *node.Description
		}

		if node.Table != nil {
			dm.TableID = node.Table.Id
		}

		for _, col := range node.PrimaryColumns {
			if col != nil {
				colID := client.ExtractDatamineColumnIDFromList(col)
				if colID != "" {
					dm.PrimaryColumnIDs = append(dm.PrimaryColumnIDs, colID)
				}
			}
		}

		// Parse schedule from genqlient type
		if node.Schedule != nil {
			fixed, cron := client.ExtractDatamineScheduleFromListNode(node.Schedule)
			if fixed != nil {
				dm.ScheduleType = "fixed"
				dm.TimeUnit = fixed.TimeUnit
				dm.Value = int64(fixed.Value)
			} else if cron != nil {
				dm.ScheduleType = "cron"
				dm.CronExpression = cron.CronExpression
			}
		}

		datamines = append(datamines, dm)
	}

	return datamines, nil
}

// ListAllDatamines returns all datamines across all tables in the organization.
func ListAllDatamines(ctx context.Context, c *client.Client) ([]Datamine, error) {
	// First, get all tables
	tables, err := ListTables(ctx, c)
	if err != nil {
		return nil, fmt.Errorf("failed to list tables: %w", err)
	}

	var allDatamines []Datamine
	for _, table := range tables {
		datamines, err := ListDatamines(ctx, c, table.ID)
		if err != nil {
			// Skip tables we can't read datamines from (permissions, etc.)
			continue
		}
		// Populate table name for display
		for i := range datamines {
			datamines[i].TableName = table.Name
		}
		allDatamines = append(allDatamines, datamines...)
	}

	return allDatamines, nil
}

// GetUUIDMappings returns UUID to Terraform reference mappings for a Datamine
func (d *Datamine) GetUUIDMappings(resourceName string) map[string]string {
	m := make(map[string]string)
	m[d.ID] = "elementum_datamine." + resourceName + ".id"
	return m
}

// parseSchedule parses schedule data from a map into a Datamine struct.
// This is used by tests and for backwards compatibility with ExecuteInto responses.
func parseSchedule(schedule map[string]interface{}, dm *Datamine) {
	if schedule == nil {
		return
	}

	typename, ok := schedule["__typename"].(string)
	if !ok {
		return
	}

	switch typename {
	case "DatamineScheduleFixed":
		dm.ScheduleType = "fixed"
		if unit, ok := schedule["timeUnit"].(string); ok {
			dm.TimeUnit = unit
		}
		if val, ok := schedule["value"].(float64); ok {
			dm.Value = int64(val)
		}
	case "DatamineScheduleCron":
		dm.ScheduleType = "cron"
		if expr, ok := schedule["cronExpression"].(string); ok {
			dm.CronExpression = expr
		}
	}
}

// FetchDatamineRelatedResources fetches all resources related to a datamine for beautification
func FetchDatamineRelatedResources(ctx context.Context, c *client.Client, datamine *Datamine, table *Table) ([]RelatedResource, error) {
	var resources []RelatedResource

	// Add the table
	if table != nil {
		resources = append(resources, RelatedResource{
			ID:           table.ID,
			Name:         table.Name,
			ResourceType: "elementum_table",
		})
	}

	// Add table's related resources (CloudLink, source, category)
	if table != nil {
		tableRelated, err := FetchTableRelatedResources(ctx, c, table)
		if err != nil {
			return nil, err
		}
		resources = append(resources, tableRelated...)
	}

	return resources, nil
}

// FetchTableRelatedResources fetches resources related to a table (CloudLink, source, category)
func FetchTableRelatedResources(ctx context.Context, c *client.Client, table *Table) ([]RelatedResource, error) {
	var resources []RelatedResource

	if table == nil {
		return resources, nil
	}

	// Add CloudLink if present
	if table.CloudLinkID != "" {
		if table.CloudLinkName == "" {
			return nil, fmt.Errorf("table %s has CloudLinkID %s but CloudLinkName is empty", table.ID, table.CloudLinkID)
		}
		resources = append(resources, RelatedResource{
			ID:           table.CloudLinkID,
			Name:         table.CloudLinkName,
			ResourceType: "elementum_cloudlink",
		})
	}

	return resources, nil
}
