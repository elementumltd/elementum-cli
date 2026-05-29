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

package export

import (
	"strings"
	"testing"

	"github.com/elementumltd/elementum-cli/discovery"
)

// Test injectSnowflakeCloudMapping injects Snowflake fields correctly
func TestInjectSnowflakeCloudMapping(t *testing.T) {
	tests := []struct {
		name         string
		hcl          string
		table        *discovery.Table
		wantContains []string
		wantMissing  []string
	}{
		{
			name: "injects all Snowflake fields",
			hcl: `resource "elementum_table" "sales_data" {
  name   = "Sales Data"
  handle = "sales"
}`,
			table: &discovery.Table{
				ID:                    "table-123",
				Name:                  "Sales Data",
				SnowflakeDatabaseName: "PROD_DB",
				SnowflakeSchemaName:   "PUBLIC",
				SnowflakeTableName:    "SALES",
			},
			wantContains: []string{
				`cloud_mapping_snowflake_database = "PROD_DB"`,
				`cloud_mapping_snowflake_schema   = "PUBLIC"`,
				`cloud_mapping_snowflake_table    = "SALES"`,
			},
		},
		{
			name: "injects partial Snowflake fields",
			hcl: `resource "elementum_table" "partial" {
  name = "Partial"
}`,
			table: &discovery.Table{
				ID:                    "table-456",
				Name:                  "Partial",
				SnowflakeDatabaseName: "ONLY_DB",
				// No schema or table name
			},
			wantContains: []string{
				`cloud_mapping_snowflake_database = "ONLY_DB"`,
			},
			wantMissing: []string{
				"cloud_mapping_snowflake_schema",
				"cloud_mapping_snowflake_table",
			},
		},
		{
			name: "no injection when no Snowflake data",
			hcl: `resource "elementum_table" "no_snowflake" {
  name = "No Snowflake"
}`,
			table: &discovery.Table{
				ID:   "table-789",
				Name: "No Snowflake",
				// No Snowflake fields
			},
			wantMissing: []string{
				"cloud_mapping_snowflake_database",
				"cloud_mapping_snowflake_schema",
				"cloud_mapping_snowflake_table",
			},
		},
		{
			name: "nil table returns unchanged HCL",
			hcl: `resource "elementum_table" "test" {
  name = "Test"
}`,
			table: nil,
			wantMissing: []string{
				"cloud_mapping_snowflake",
			},
		},
		{
			name: "handles special characters in values",
			hcl: `resource "elementum_table" "special" {
  name = "Special"
}`,
			table: &discovery.Table{
				ID:                    "table-special",
				Name:                  "Special",
				SnowflakeDatabaseName: "DB_WITH_UNDERSCORE",
				SnowflakeSchemaName:   "SCHEMA_123",
				SnowflakeTableName:    "TABLE_NAME_V2",
			},
			wantContains: []string{
				`cloud_mapping_snowflake_database = "DB_WITH_UNDERSCORE"`,
				`cloud_mapping_snowflake_schema   = "SCHEMA_123"`,
				`cloud_mapping_snowflake_table    = "TABLE_NAME_V2"`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := injectSnowflakeCloudMapping(tt.hcl, tt.table)

			for _, want := range tt.wantContains {
				if !strings.Contains(result, want) {
					t.Errorf("expected result to contain %q, got:\n%s", want, result)
				}
			}

			for _, notWant := range tt.wantMissing {
				if strings.Contains(result, notWant) {
					t.Errorf("expected result NOT to contain %q, got:\n%s", notWant, result)
				}
			}
		})
	}
}

// Test BeautifyTableConfig with Snowflake cloud mapping
func TestBeautifyTableConfig_WithSnowflake(t *testing.T) {
	input := `resource "elementum_table" "cloud_table" {
  category_id                = "11111111-1111-1111-1111-111111111111"
  cloud_mapping_cloudlink_id = "22222222-2222-2222-2222-222222222222"
  name                       = "Cloud Table"
  handle                     = "cloud"
  description                = null
}`

	imports := []ImportBlock{
		{
			ID:           "33333333-3333-3333-3333-333333333333",
			ResourceType: "elementum_table",
			ResourceName: "cloud_table",
		},
	}

	table := &discovery.Table{
		ID:                    "33333333-3333-3333-3333-333333333333",
		Name:                  "Cloud Table",
		Handle:                "cloud",
		CategoryID:            "11111111-1111-1111-1111-111111111111",
		CloudLinkID:           "22222222-2222-2222-2222-222222222222",
		CloudLinkName:         "Production Snowflake",
		SnowflakeDatabaseName: "ANALYTICS_DB",
		SnowflakeSchemaName:   "DW_SCHEMA",
		SnowflakeTableName:    "FACT_SALES",
	}

	relatedResources := []discovery.RelatedResource{
		{
			ID:           "11111111-1111-1111-1111-111111111111",
			Name:         "Analytics",
			ResourceType: "data.elementum_category",
		},
		{
			ID:           "22222222-2222-2222-2222-222222222222",
			Name:         "Production Snowflake",
			ResourceType: "elementum_cloudlink",
		},
	}

	result := BeautifyTableConfig(input, imports, table, relatedResources)

	// Check Snowflake fields are injected
	if !strings.Contains(result, `cloud_mapping_snowflake_database = "ANALYTICS_DB"`) {
		t.Errorf("Snowflake database not injected")
	}
	if !strings.Contains(result, `cloud_mapping_snowflake_schema   = "DW_SCHEMA"`) {
		t.Errorf("Snowflake schema not injected")
	}
	if !strings.Contains(result, `cloud_mapping_snowflake_table    = "FACT_SALES"`) {
		t.Errorf("Snowflake table not injected")
	}

	// Check CloudLink is beautified to data source reference
	if !strings.Contains(result, "data.elementum_cloudlink.production_snowflake.id") {
		t.Errorf("CloudLink not beautified to data source reference, got:\n%s", result)
	}

	// Check nulls are stripped
	if strings.Contains(result, "= null") {
		t.Errorf("Null attributes not stripped")
	}
}

// Test table with source referencing another table
func TestBeautifyTableConfig_SourceTable(t *testing.T) {
	input := `resource "elementum_table" "derived" {
  name      = "Derived Table"
  source_id = "44444444-4444-4444-4444-444444444444"
}`

	imports := []ImportBlock{
		{
			ID:           "55555555-5555-5555-5555-555555555555",
			ResourceType: "elementum_table",
			ResourceName: "derived",
		},
		{
			ID:           "44444444-4444-4444-4444-444444444444",
			ResourceType: "elementum_table",
			ResourceName: "source_table",
		},
	}

	table := &discovery.Table{
		ID:       "55555555-5555-5555-5555-555555555555",
		Name:     "Derived Table",
		SourceID: "44444444-4444-4444-4444-444444444444",
	}

	relatedResources := []discovery.RelatedResource{
		{
			ID:           "44444444-4444-4444-4444-444444444444",
			Name:         "Source Table",
			ResourceType: "elementum_table",
		},
	}

	result := BeautifyTableConfig(input, imports, table, relatedResources)

	// Check source_id is replaced with table reference
	if strings.Contains(result, "44444444-4444-4444-4444-444444444444") {
		t.Errorf("Source table UUID not replaced")
	}
	if !strings.Contains(result, "elementum_table.source_table.id") {
		t.Errorf("Source table reference not inserted, got:\n%s", result)
	}
}

// Test multiple CloudLinks in related resources
func TestBuildTableUUIDMap_MultipleCloudLinks(t *testing.T) {
	imports := []ImportBlock{
		{
			ID:           "table-123",
			ResourceType: "elementum_table",
			ResourceName: "test_table",
		},
	}

	table := &discovery.Table{
		ID:   "table-123",
		Name: "Test Table",
	}

	relatedResources := []discovery.RelatedResource{
		{
			ID:           "cloudlink-1",
			Name:         "Snowflake Prod",
			ResourceType: "elementum_cloudlink",
		},
		{
			ID:           "cloudlink-2",
			Name:         "BigQuery Dev",
			ResourceType: "elementum_cloudlink",
		},
	}

	uuidMap := buildTableUUIDMap(imports, table, relatedResources)

	// Both CloudLinks should be mapped as data sources
	if ref := uuidMap["cloudlink-1"]; ref != "data.elementum_cloudlink.snowflake_prod.id" {
		t.Errorf("First CloudLink not mapped correctly, got: %s", ref)
	}
	// SanitizeName now splits CamelCase at word boundaries — "BigQuery Dev"
	// becomes "big_query_dev" rather than "bigquery_dev".
	if ref := uuidMap["cloudlink-2"]; ref != "data.elementum_cloudlink.big_query_dev.id" {
		t.Errorf("Second CloudLink not mapped correctly, got: %s", ref)
	}
}

// Test self-referencing source_id is stripped
func TestStripSelfReferencingSourceID(t *testing.T) {
	tests := []struct {
		name        string
		hcl         string
		table       *discovery.Table
		wantMissing string
	}{
		{
			name: "strips self-reference after UUID replacement",
			hcl: `resource "elementum_table" "my_table" {
  name      = "My Table"
  source_id = elementum_table.my_table.id
}`,
			table: &discovery.Table{
				ID:   "table-123",
				Name: "My Table",
			},
			wantMissing: "source_id",
		},
		{
			name: "keeps valid source reference",
			hcl: `resource "elementum_table" "derived" {
  name      = "Derived"
  source_id = elementum_table.source.id
}`,
			table: &discovery.Table{
				ID:       "derived-123",
				Name:     "Derived",
				SourceID: "source-456",
			},
			wantMissing: "", // should NOT be missing
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			imports := []ImportBlock{
				{
					ID:           tt.table.ID,
					ResourceType: "elementum_table",
					ResourceName: SanitizeName(tt.table.Name),
				},
			}

			result := stripSelfReferencingSourceID(tt.hcl, tt.table)

			if tt.wantMissing != "" {
				if strings.Contains(result, tt.wantMissing) {
					t.Errorf("expected %q to be stripped, got:\n%s", tt.wantMissing, result)
				}
			} else {
				// For the "keeps valid" case, ensure source_id is present
				if !strings.Contains(result, "source_id") {
					t.Errorf("source_id should be kept for valid references, got:\n%s", result)
				}
			}

			_ = imports // silence unused variable warning
		})
	}
}

// Test stripBaseTableReferenceFieldIDs removes reference_field_id from base tables
func TestStripBaseTableReferenceFieldIDs(t *testing.T) {
	tests := []struct {
		name               string
		hcl                string
		table              *discovery.Table
		discoveredTableIDs map[string]bool
		resourceName       string
		wantContains       []string
		wantMissing        []string
	}{
		{
			name: "strips reference_field_id from base table (self-referential sourceID)",
			hcl: `resource "elementum_table" "base_table" {
  name      = "Base Table"
  handle    = "base"
  fields = [
    {
      name               = "INVOICE_NUMBER"
      reference_field_id = "11111111-1111-1111-1111-111111111111"
      cloud_column_name  = "INVOICE_NUMBER"
      cloud_type         = "TEXT"
    },
    {
      name               = "AMOUNT"
      reference_field_id = "22222222-2222-2222-2222-222222222222"
      cloud_column_name  = "AMOUNT"
      cloud_type         = "NUMBER"
    },
  ]
}`,
			table: &discovery.Table{
				ID:       "table-123",
				Name:     "Base Table",
				SourceID: "table-123", // Self-reference = base table
			},
			discoveredTableIDs: map[string]bool{"table-123": true},
			resourceName:       "base_table",
			wantContains: []string{
				"cloud_column_name",
				"cloud_type",
				"INVOICE_NUMBER",
				"AMOUNT",
			},
			wantMissing: []string{
				"reference_field_id",
			},
		},
		{
			name: "keeps reference_field_id for derived table (sourceID points to different table)",
			hcl: `resource "elementum_table" "derived_table" {
  name      = "Derived Table"
  source_id = "source-456"
  fields = [
    {
      name               = "customer_count"
      reference_field_id = "33333333-3333-3333-3333-333333333333"
    },
  ]
}`,
			table: &discovery.Table{
				ID:       "derived-789",
				Name:     "Derived Table",
				SourceID: "source-456", // Points to different table = derived
			},
			discoveredTableIDs: map[string]bool{"derived-789": true, "source-456": true},
			resourceName:       "derived_table",
			wantContains: []string{
				"reference_field_id",
				"33333333-3333-3333-3333-333333333333",
			},
			wantMissing: []string{},
		},
		{
			name: "nil table returns unchanged HCL",
			hcl: `resource "elementum_table" "test" {
  fields = [
    {
      reference_field_id = "some-uuid"
    },
  ]
}`,
			table:              nil,
			discoveredTableIDs: nil,
			resourceName:       "test",
			wantContains: []string{
				"reference_field_id",
			},
			wantMissing: []string{},
		},
		{
			name: "strips multiple reference_field_id lines with varying whitespace",
			hcl: `resource "elementum_table" "multi" {
  fields = [
    {
      name               = "Field1"
      reference_field_id = "uuid-1"
    },
    {
      name                 = "Field2"
      reference_field_id   = "uuid-2"
    },
    {
			reference_field_id = "uuid-3"
      name               = "Field3"
    },
  ]
}`,
			table: &discovery.Table{
				ID:       "multi-table",
				Name:     "Multi",
				SourceID: "", // No source = base table
			},
			discoveredTableIDs: map[string]bool{"multi-table": true},
			resourceName:       "multi",
			wantContains: []string{
				"Field1",
				"Field2",
				"Field3",
			},
			wantMissing: []string{
				"reference_field_id",
				"uuid-1",
				"uuid-2",
				"uuid-3",
			},
		},
		{
			name: "strips reference_field_id when sourceID points to non-table (app/element)",
			hcl: `resource "elementum_table" "app_sourced" {
  fields = [
    {
      name               = "Column1"
      reference_field_id = "field-uuid-1"
    },
  ]
}`,
			table: &discovery.Table{
				ID:       "table-from-app",
				Name:     "App Sourced Table",
				SourceID: "app-123", // Points to app, not a table
			},
			// discoveredTableIDs only has tables, not the app
			discoveredTableIDs: map[string]bool{"table-from-app": true},
			resourceName:       "app_sourced",
			wantContains: []string{
				"Column1",
			},
			wantMissing: []string{
				"reference_field_id",
			},
		},
		{
			name: "strips reference_field_id when sourceID points to undiscovered table",
			hcl: `resource "elementum_table" "orphan" {
  fields = [
    {
      name               = "Data"
      reference_field_id = "field-uuid-2"
    },
  ]
}`,
			table: &discovery.Table{
				ID:       "orphan-table",
				Name:     "Orphan Table",
				SourceID: "undiscovered-table-id", // Points to table not in our discovery
			},
			// discoveredTableIDs doesn't include the source table
			discoveredTableIDs: map[string]bool{"orphan-table": true},
			resourceName:       "orphan",
			wantContains: []string{
				"Data",
			},
			wantMissing: []string{
				"reference_field_id",
			},
		},
		{
			name: "keeps reference_field_id with local reference syntax",
			hcl: `resource "elementum_table" "derived_with_local" {
  source_id = "source-table-id"
  fields = [
    {
      name               = "Aggregated"
      reference_field_id = local.source_table_field_ids_by_name["Original"]
    },
  ]
}`,
			table: &discovery.Table{
				ID:       "derived-local",
				Name:     "Derived With Local",
				SourceID: "source-table-id", // Different discovered table
			},
			discoveredTableIDs: map[string]bool{"derived-local": true, "source-table-id": true},
			resourceName:       "derived_with_local",
			wantContains: []string{
				"reference_field_id",
				"local.source_table_field_ids_by_name",
			},
			wantMissing: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := stripBaseTableReferenceFieldIDs(tt.hcl, tt.table, tt.discoveredTableIDs, tt.resourceName)

			for _, want := range tt.wantContains {
				if !strings.Contains(result, want) {
					t.Errorf("expected result to contain %q, got:\n%s", want, result)
				}
			}

			for _, notWant := range tt.wantMissing {
				if strings.Contains(result, notWant) {
					t.Errorf("expected result NOT to contain %q, got:\n%s", notWant, result)
				}
			}
		})
	}
}

// Test BeautifyTableConfig strips reference_field_id from base tables (integration test)
func TestBeautifyTableConfig_BaseTableReferenceFieldIDCycle(t *testing.T) {
	// This test verifies the fix for Issue #1: Base tables incorrectly export reference_field_id causing cycles
	// When a base table (self-referential or no sourceID) is exported, it should NOT have
	// reference_field_id attributes on its fields, as these create self-referential cycles.

	input := `resource "elementum_table" "srv_supplier_invoice_history_v2" {
  category_id                = "cat-123"
  cloud_mapping_cloudlink_id = "cloudlink-456"
  name                       = "SRV_SUPPLIER_INVOICE_HISTORY_V2"
  handle                     = "srv_supplier_invoice_history_v2"
  fields = [
    {
      name               = "INVOICE_NUMBER"
      reference_field_id = "field-uuid-1"
      cloud_column_name  = "INVOICE_NUMBER"
      cloud_type         = "TEXT"
    },
    {
      name               = "SUPPLIER_ID"
      reference_field_id = "field-uuid-2"
      cloud_column_name  = "SUPPLIER_ID"
      cloud_type         = "TEXT"
    },
    {
      name               = "AMOUNT"
      reference_field_id = "field-uuid-3"
      cloud_column_name  = "AMOUNT"
      cloud_type         = "NUMBER"
    },
  ]
}`

	imports := []ImportBlock{
		{
			ID:           "table-abc",
			ResourceType: "elementum_table",
			ResourceName: "srv_supplier_invoice_history_v2",
		},
	}

	// Base table: sourceID equals own ID (self-referential)
	table := &discovery.Table{
		ID:                    "table-abc",
		Name:                  "SRV_SUPPLIER_INVOICE_HISTORY_V2",
		Handle:                "srv_supplier_invoice_history_v2",
		SourceID:              "table-abc", // Self-reference = base table
		CloudLinkID:           "cloudlink-456",
		CloudLinkName:         "Production Snowflake",
		SnowflakeDatabaseName: "PROD_DB",
		SnowflakeSchemaName:   "PUBLIC",
		SnowflakeTableName:    "SRV_SUPPLIER_INVOICE_HISTORY",
	}

	relatedResources := []discovery.RelatedResource{
		{
			ID:           "cat-123",
			Name:         "Tables",
			ResourceType: "data.elementum_category",
		},
		{
			ID:           "cloudlink-456",
			Name:         "Production Snowflake",
			ResourceType: "elementum_cloudlink",
		},
	}

	result := BeautifyTableConfig(input, imports, table, relatedResources)

	// CRITICAL: reference_field_id must be stripped to prevent cycle errors
	if strings.Contains(result, "reference_field_id") {
		t.Errorf("reference_field_id should be stripped from base table to prevent cycles, got:\n%s", result)
	}

	// Verify cloud column attributes are preserved
	if !strings.Contains(result, "cloud_column_name") {
		t.Errorf("cloud_column_name should be preserved")
	}
	if !strings.Contains(result, "cloud_type") {
		t.Errorf("cloud_type should be preserved")
	}

	// Verify field names are preserved
	if !strings.Contains(result, "INVOICE_NUMBER") {
		t.Errorf("field names should be preserved")
	}

	// Verify Snowflake cloud mapping is injected
	if !strings.Contains(result, `cloud_mapping_snowflake_database = "PROD_DB"`) {
		t.Errorf("Snowflake database should be injected")
	}
}

// Test BeautifyTableConfig keeps reference_field_id for derived tables
func TestBeautifyTableConfig_DerivedTableKeepsReferenceFieldID(t *testing.T) {
	input := `resource "elementum_table" "aggregated_sales" {
  name      = "Aggregated Sales"
  source_id = "44444444-4444-4444-4444-444444444444"
  fields = [
    {
      name                  = "customer_count"
      reference_field_id    = "55555555-5555-5555-5555-555555555555"
      reference_aggregation = "COUNT_DISTINCT"
    },
  ]
}`

	imports := []ImportBlock{
		{
			ID:           "66666666-6666-6666-6666-666666666666",
			ResourceType: "elementum_table",
			ResourceName: "aggregated_sales",
		},
		{
			ID:           "44444444-4444-4444-4444-444444444444",
			ResourceType: "elementum_table",
			ResourceName: "source_table",
		},
	}

	// Derived table: sourceID points to a DIFFERENT table
	table := &discovery.Table{
		ID:       "66666666-6666-6666-6666-666666666666",
		Name:     "Aggregated Sales",
		SourceID: "44444444-4444-4444-4444-444444444444", // Different table = derived
	}

	relatedResources := []discovery.RelatedResource{
		{
			ID:           "44444444-4444-4444-4444-444444444444",
			Name:         "Source Table",
			ResourceType: "elementum_table",
		},
	}

	result := BeautifyTableConfig(input, imports, table, relatedResources)

	// Derived tables SHOULD keep reference_field_id
	if !strings.Contains(result, "reference_field_id") {
		t.Errorf("reference_field_id should be kept for derived tables, got:\n%s", result)
	}

	// source_id should be beautified to reference
	if !strings.Contains(result, "elementum_table.source_table.id") {
		t.Errorf("source_id should reference source table, got:\n%s", result)
	}
}
