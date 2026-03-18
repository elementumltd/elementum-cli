// Copyright (c) ElementumAI, Inc.
// SPDX-License-Identifier: MPL-2.0

package discovery

import (
	"context"
	"fmt"
	"sort"

	"github.com/elementumltd/elementum-cli/logger"
	"github.com/elementumltd/elementum-cli/internal/client"
)

// SnowflakeDatabaseInfo represents a discovered Snowflake database
type SnowflakeDatabaseInfo struct {
	Name    string
	Owner   string
	Comment string
}

// SnowflakeSchemaInfo represents a discovered Snowflake schema
type SnowflakeSchemaInfo struct {
	Name         string
	DatabaseName string
	Owner        string
	Comment      string
}

// SnowflakeTableInfo represents a discovered Snowflake table
type SnowflakeTableInfo struct {
	Name         string
	DatabaseName string
	SchemaName   string
	Type         string // TABLE, VIEW
	Rows         int64
	Bytes        int64
}

// ListSnowflakeDatabases retrieves all databases from a CloudLink
func ListSnowflakeDatabases(ctx context.Context, c *client.Client, cloudLinkID string) ([]SnowflakeDatabaseInfo, error) {
	logger.Debug("fetching snowflake databases", "cloudlink_id", cloudLinkID)

	result, err := client.ListSnowflakeDatabases(ctx, c.Genqlient(), cloudLinkID)
	if err != nil {
		return nil, fmt.Errorf("failed to list snowflake databases: %w", err)
	}

	databases := client.ExtractSnowflakeDatabases(result)
	if databases == nil {
		return nil, nil
	}

	infos := make([]SnowflakeDatabaseInfo, 0, len(databases))
	for _, db := range databases {
		infos = append(infos, SnowflakeDatabaseInfo{
			Name:    db.Name,
			Owner:   db.Owner,
			Comment: db.Comment,
		})
	}

	// Sort by name for consistent output
	sort.Slice(infos, func(i, j int) bool {
		return infos[i].Name < infos[j].Name
	})

	logger.Debug("got snowflake databases", "count", len(infos))
	return infos, nil
}

// ListSnowflakeSchemas retrieves all schemas from a CloudLink, optionally filtered by database
func ListSnowflakeSchemas(ctx context.Context, c *client.Client, cloudLinkID string, database string) ([]SnowflakeSchemaInfo, error) {
	logger.Debug("fetching snowflake schemas", "cloudlink_id", cloudLinkID, "database", database)

	result, err := client.ListSnowflakeSchemas(ctx, c.Genqlient(), cloudLinkID)
	if err != nil {
		return nil, fmt.Errorf("failed to list snowflake schemas: %w", err)
	}

	schemas := client.ExtractSnowflakeSchemas(result)
	if schemas == nil {
		return nil, nil
	}

	infos := make([]SnowflakeSchemaInfo, 0, len(schemas))
	for _, s := range schemas {
		// Filter by database if specified
		if database != "" && s.DatabaseName != database {
			continue
		}
		infos = append(infos, SnowflakeSchemaInfo{
			Name:         s.Name,
			DatabaseName: s.DatabaseName,
			Owner:        s.Owner,
			Comment:      s.Comment,
		})
	}

	// Sort by database then name for consistent output
	sort.Slice(infos, func(i, j int) bool {
		if infos[i].DatabaseName != infos[j].DatabaseName {
			return infos[i].DatabaseName < infos[j].DatabaseName
		}
		return infos[i].Name < infos[j].Name
	})

	logger.Debug("got snowflake schemas", "count", len(infos))
	return infos, nil
}

// ListSnowflakeTables retrieves all tables from a CloudLink, optionally filtered by database and schema
func ListSnowflakeTables(ctx context.Context, c *client.Client, cloudLinkID string, database, schema string) ([]SnowflakeTableInfo, error) {
	logger.Debug("fetching snowflake tables", "cloudlink_id", cloudLinkID, "database", database, "schema", schema)

	result, err := client.ListSnowflakeTables(ctx, c.Genqlient(), cloudLinkID)
	if err != nil {
		return nil, fmt.Errorf("failed to list snowflake tables: %w", err)
	}

	tables := client.ExtractSnowflakeTables(result)
	if tables == nil {
		return nil, nil
	}

	infos := make([]SnowflakeTableInfo, 0, len(tables))
	for _, t := range tables {
		// Filter by database if specified
		if database != "" && t.DatabaseName != database {
			continue
		}
		// Filter by schema if specified
		if schema != "" && t.SchemaName != schema {
			continue
		}
		infos = append(infos, SnowflakeTableInfo{
			Name:         t.Name,
			DatabaseName: t.DatabaseName,
			SchemaName:   t.SchemaName,
			Type:         t.Type,
			Rows:         t.Rows,
			Bytes:        t.Bytes,
		})
	}

	// Sort by database, schema, then name for consistent output
	sort.Slice(infos, func(i, j int) bool {
		if infos[i].DatabaseName != infos[j].DatabaseName {
			return infos[i].DatabaseName < infos[j].DatabaseName
		}
		if infos[i].SchemaName != infos[j].SchemaName {
			return infos[i].SchemaName < infos[j].SchemaName
		}
		return infos[i].Name < infos[j].Name
	})

	logger.Debug("got snowflake tables", "count", len(infos))
	return infos, nil
}

// GetSnowflakeTableSchema retrieves the schema (columns) of a Snowflake table from a CloudLink
func GetSnowflakeTableSchema(ctx context.Context, c *client.Client, cloudLinkID, databaseName, schemaName, tableName string) (*SnowflakeTableSchema, error) {
	logger.Debug("fetching snowflake table schema",
		"cloudlink_id", cloudLinkID,
		"database", databaseName,
		"schema", schemaName,
		"table", tableName,
	)

	// Call the genqlient-generated function
	result, err := client.GetSnowflakeTableSchema(ctx, c.Genqlient(), cloudLinkID, databaseName, schemaName, tableName)
	if err != nil {
		return nil, fmt.Errorf("failed to get snowflake table schema: %w", err)
	}

	// Extract the table data from the result
	tableData := client.ExtractSnowflakeTableFromSchema(result)
	if tableData == nil {
		return nil, fmt.Errorf("table not found: %s.%s.%s", databaseName, schemaName, tableName)
	}

	// Convert to our discovery type
	schema := &SnowflakeTableSchema{
		Name:         tableData.Name,
		DatabaseName: tableData.DatabaseName,
		SchemaName:   tableData.SchemaName,
		Type:         tableData.Type,
		Rows:         tableData.Rows,
		Bytes:        tableData.Bytes,
		Columns:      make([]SnowflakeColumn, 0, len(tableData.Columns)),
	}

	for _, col := range tableData.Columns {
		schema.Columns = append(schema.Columns, SnowflakeColumn{
			Name:         col.Name,
			DatabaseType: col.DatabaseType,
			Type:         col.Type,
			Nullable:     col.Nullable,
			PrimaryKey:   col.PrimaryKey,
			UniqueKey:    col.UniqueKey,
			Comment:      col.Comment,
		})
	}

	logger.Debug("got snowflake table schema",
		"table", tableName,
		"columns", len(schema.Columns),
	)

	return schema, nil
}

// GetSnowflakeTableSchemaByCloudLinkName retrieves the schema using a CloudLink name instead of ID
func GetSnowflakeTableSchemaByCloudLinkName(ctx context.Context, c *client.Client, cloudLinkName, databaseName, schemaName, tableName string) (*SnowflakeTableSchema, error) {
	// First, look up the CloudLink by name
	cloudLink, err := GetCloudLinkByName(ctx, c, cloudLinkName)
	if err != nil {
		return nil, fmt.Errorf("failed to find cloudlink by name: %w", err)
	}

	// Then fetch the table schema
	return GetSnowflakeTableSchema(ctx, c, cloudLink.ID, databaseName, schemaName, tableName)
}
