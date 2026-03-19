// Copyright (c) ElementumAI, Inc.
// SPDX-License-Identifier: MPL-2.0

package discovery

import (
	"testing"
)

func TestTable_GetUUIDMappings(t *testing.T) {
	table := &Table{
		ID:          "table-uuid-123",
		Name:        "Sales Data",
		Handle:      "sales_data",
		CategoryID:  "cat-uuid-456",
		CloudLinkID: "cloudlink-uuid-789",
	}

	resourceName := "sales_data"
	mappings := table.GetUUIDMappings(resourceName)

	expectedID := "elementum_table." + resourceName + ".id"
	if mappings[table.ID] != expectedID {
		t.Errorf("expected mapping for table ID to be '%s', got '%s'", expectedID, mappings[table.ID])
	}
}

func TestTable_FullModel(t *testing.T) {
	table := &Table{
		ID:                    "table-uuid-123",
		Name:                  "Sales Summary",
		Handle:                "sales_summary",
		Description:           "Aggregated sales data",
		CloudLinkID:           "cloudlink-456",
		CloudLinkName:         "Snowflake Production",
		SourceID:              "source-table-789",
		CategoryID:            "category-abc",
		Type:                  "TABLE",
		Configured:            true,
		SnowflakeDatabaseName: "ANALYTICS_DB",
		SnowflakeSchemaName:   "PUBLIC",
		SnowflakeTableName:    "SALES_SUMMARY",
	}

	if table.ID != "table-uuid-123" {
		t.Errorf("ID mismatch")
	}
	if table.Name != "Sales Summary" {
		t.Errorf("Name mismatch")
	}
	if table.Handle != "sales_summary" {
		t.Errorf("Handle mismatch")
	}
	if table.CloudLinkID != "cloudlink-456" {
		t.Errorf("CloudLinkID mismatch")
	}
	if table.CloudLinkName != "Snowflake Production" {
		t.Errorf("CloudLinkName mismatch")
	}
	if table.SourceID != "source-table-789" {
		t.Errorf("SourceID mismatch")
	}
	if table.SnowflakeDatabaseName != "ANALYTICS_DB" {
		t.Errorf("SnowflakeDatabaseName mismatch")
	}
	if table.SnowflakeSchemaName != "PUBLIC" {
		t.Errorf("SnowflakeSchemaName mismatch")
	}
	if table.SnowflakeTableName != "SALES_SUMMARY" {
		t.Errorf("SnowflakeTableName mismatch")
	}
}

func TestTable_CloudMappingFields(t *testing.T) {
	tests := []struct {
		name              string
		table             *Table
		hasCloudLink      bool
		hasSnowflake      bool
		expectedDatabase  string
		expectedSchema    string
		expectedTableName string
	}{
		{
			name: "table with full Snowflake mapping",
			table: &Table{
				ID:                    "t1",
				Name:                  "Cloud Table",
				CloudLinkID:           "cl-1",
				CloudLinkName:         "Snowflake Link",
				SnowflakeDatabaseName: "PROD_DB",
				SnowflakeSchemaName:   "DW",
				SnowflakeTableName:    "FACT_ORDERS",
			},
			hasCloudLink:      true,
			hasSnowflake:      true,
			expectedDatabase:  "PROD_DB",
			expectedSchema:    "DW",
			expectedTableName: "FACT_ORDERS",
		},
		{
			name: "table with CloudLink but no Snowflake",
			table: &Table{
				ID:            "t2",
				Name:          "API Table",
				CloudLinkID:   "cl-2",
				CloudLinkName: "API Connection",
			},
			hasCloudLink: true,
			hasSnowflake: false,
		},
		{
			name: "table with no cloud mapping",
			table: &Table{
				ID:   "t3",
				Name: "Local Table",
			},
			hasCloudLink: false,
			hasSnowflake: false,
		},
		{
			name: "table with partial Snowflake (database only)",
			table: &Table{
				ID:                    "t4",
				Name:                  "Partial Cloud",
				CloudLinkID:           "cl-4",
				CloudLinkName:         "Partial Link",
				SnowflakeDatabaseName: "ONLY_DB",
			},
			hasCloudLink:     true,
			hasSnowflake:     true,
			expectedDatabase: "ONLY_DB",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hasCloudLink := tt.table.CloudLinkID != ""
			if hasCloudLink != tt.hasCloudLink {
				t.Errorf("hasCloudLink: expected %v, got %v", tt.hasCloudLink, hasCloudLink)
			}

			hasSnowflake := tt.table.SnowflakeDatabaseName != "" ||
				tt.table.SnowflakeSchemaName != "" ||
				tt.table.SnowflakeTableName != ""
			if hasSnowflake != tt.hasSnowflake {
				t.Errorf("hasSnowflake: expected %v, got %v", tt.hasSnowflake, hasSnowflake)
			}

			if tt.expectedDatabase != "" && tt.table.SnowflakeDatabaseName != tt.expectedDatabase {
				t.Errorf("database: expected %s, got %s", tt.expectedDatabase, tt.table.SnowflakeDatabaseName)
			}
			if tt.expectedSchema != "" && tt.table.SnowflakeSchemaName != tt.expectedSchema {
				t.Errorf("schema: expected %s, got %s", tt.expectedSchema, tt.table.SnowflakeSchemaName)
			}
			if tt.expectedTableName != "" && tt.table.SnowflakeTableName != tt.expectedTableName {
				t.Errorf("tableName: expected %s, got %s", tt.expectedTableName, tt.table.SnowflakeTableName)
			}
		})
	}
}

func TestCloudLink_FullModel(t *testing.T) {
	cloudlink := &CloudLink{
		ID:   "cloudlink-uuid-123",
		Name: "Production Snowflake",
		Type: "CloudLinkSnowflake",
	}

	if cloudlink.ID != "cloudlink-uuid-123" {
		t.Errorf("ID mismatch")
	}
	if cloudlink.Name != "Production Snowflake" {
		t.Errorf("Name mismatch")
	}
	if cloudlink.Type != "CloudLinkSnowflake" {
		t.Errorf("Type mismatch")
	}
}

func TestUnifiedDiscoveryContext_CloudLinks(t *testing.T) {
	ctx := NewUnifiedDiscoveryContext("app-123", "App")

	// Verify CloudLinks map is initialized
	if ctx.CloudLinks == nil {
		t.Fatal("CloudLinks map should be initialized")
	}

	// Add CloudLinks
	ctx.CloudLinks["cl-1"] = &CloudLink{
		ID:   "cl-1",
		Name: "Snowflake Prod",
	}
	ctx.CloudLinks["cl-2"] = &CloudLink{
		ID:   "cl-2",
		Name: "BigQuery Analytics",
	}

	if len(ctx.CloudLinks) != 2 {
		t.Errorf("Expected 2 CloudLinks, got %d", len(ctx.CloudLinks))
	}

	if ctx.CloudLinks["cl-1"].Name != "Snowflake Prod" {
		t.Errorf("CloudLink 1 name mismatch")
	}
	if ctx.CloudLinks["cl-2"].Name != "BigQuery Analytics" {
		t.Errorf("CloudLink 2 name mismatch")
	}
}

func TestUnifiedDiscoveryContext_EnqueueCloudLink(t *testing.T) {
	ctx := NewUnifiedDiscoveryContext("table-123", "Table")

	// Simulate table discovery that queues a CloudLink
	ctx.Enqueue("cloudlink-456", "CloudLink")

	if len(ctx.Queue) != 1 {
		t.Fatalf("Expected 1 item in queue, got %d", len(ctx.Queue))
	}

	item := ctx.Queue[0]
	if item.ID != "cloudlink-456" {
		t.Errorf("Expected CloudLink ID 'cloudlink-456', got '%s'", item.ID)
	}
	if item.Type != "CloudLink" {
		t.Errorf("Expected type 'CloudLink', got '%s'", item.Type)
	}
}

func TestUnifiedDiscoveryContext_TableWithCloudLinkAndSource(t *testing.T) {
	ctx := NewUnifiedDiscoveryContext("datamine-123", "Datamine")

	// Simulate a table being discovered from a datamine
	table := &Table{
		ID:            "table-456",
		Name:          "Data Table",
		SourceID:      "source-table-789",
		CloudLinkID:   "cloudlink-abc",
		CloudLinkName: "Snowflake Link",
	}

	ctx.Tables[table.ID] = table
	ctx.MarkVisited(table.ID)

	// Simulate queueing source table and CloudLink
	if table.SourceID != "" {
		ctx.Enqueue(table.SourceID, "Table")
	}
	if table.CloudLinkID != "" {
		ctx.Enqueue(table.CloudLinkID, "CloudLink")
	}

	// Verify queue has both items
	if len(ctx.Queue) != 2 {
		t.Fatalf("Expected 2 items in queue, got %d", len(ctx.Queue))
	}

	// Check source table is queued
	foundSource := false
	foundCloudLink := false
	for _, item := range ctx.Queue {
		if item.ID == "source-table-789" && item.Type == "Table" {
			foundSource = true
		}
		if item.ID == "cloudlink-abc" && item.Type == "CloudLink" {
			foundCloudLink = true
		}
	}

	if !foundSource {
		t.Error("Source table not queued")
	}
	if !foundCloudLink {
		t.Error("CloudLink not queued")
	}
}

func TestRelatedResource_CloudLink(t *testing.T) {
	resource := RelatedResource{
		ID:           "cloudlink-123",
		Name:         "Production Snowflake",
		ResourceType: "elementum_cloudlink",
	}

	if resource.ID != "cloudlink-123" {
		t.Errorf("ID mismatch")
	}
	if resource.Name != "Production Snowflake" {
		t.Errorf("Name mismatch")
	}
	if resource.ResourceType != "elementum_cloudlink" {
		t.Errorf("ResourceType mismatch")
	}
}

func TestFetchTableRelatedResources_RequiresCloudLinkName(t *testing.T) {
	// Test that FetchTableRelatedResources returns error when CloudLinkName is empty
	// but CloudLinkID is set

	// This test verifies the error behavior we implemented
	table := &Table{
		ID:          "table-123",
		Name:        "Test Table",
		CloudLinkID: "cloudlink-456",
		// CloudLinkName intentionally not set
	}

	_, err := FetchTableRelatedResources(nil, nil, table)
	if err == nil {
		t.Error("Expected error when CloudLinkName is empty but CloudLinkID is set")
	}

	expectedErrMsg := "CloudLinkName is empty"
	if err != nil && !contains(err.Error(), expectedErrMsg) {
		t.Errorf("Expected error containing '%s', got: %s", expectedErrMsg, err.Error())
	}
}

func TestFetchTableRelatedResources_WithValidCloudLink(t *testing.T) {
	table := &Table{
		ID:            "table-123",
		Name:          "Test Table",
		CloudLinkID:   "cloudlink-456",
		CloudLinkName: "Production Snowflake",
	}

	resources, err := FetchTableRelatedResources(nil, nil, table)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(resources) != 1 {
		t.Fatalf("Expected 1 resource, got %d", len(resources))
	}

	if resources[0].ID != "cloudlink-456" {
		t.Errorf("CloudLink ID mismatch")
	}
	if resources[0].Name != "Production Snowflake" {
		t.Errorf("CloudLink Name mismatch")
	}
	if resources[0].ResourceType != "elementum_cloudlink" {
		t.Errorf("ResourceType should be 'elementum_cloudlink'")
	}
}

func TestFetchTableRelatedResources_NoCloudLink(t *testing.T) {
	table := &Table{
		ID:   "table-123",
		Name: "Local Table",
		// No CloudLinkID
	}

	resources, err := FetchTableRelatedResources(nil, nil, table)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(resources) != 0 {
		t.Errorf("Expected 0 resources for table without CloudLink, got %d", len(resources))
	}
}

func TestFetchTableRelatedResources_NilTable(t *testing.T) {
	resources, err := FetchTableRelatedResources(nil, nil, nil)
	if err != nil {
		t.Fatalf("Unexpected error for nil table: %v", err)
	}

	if len(resources) != 0 {
		t.Errorf("Expected 0 resources for nil table, got %d", len(resources))
	}
}

// Helper function for string contains check
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
