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

// Test injectElementCloudMapping injects Snowflake fields correctly
func TestInjectElementCloudMapping(t *testing.T) {
	tests := []struct {
		name         string
		hcl          string
		element      *discovery.Element
		wantContains []string
		wantMissing  []string
	}{
		{
			name: "injects all Snowflake fields",
			hcl: `resource "elementum_element" "products" {
  name        = "Products"
  category_id = "cat-123"
}`,
			element: &discovery.Element{
				ID:                    "elem-123",
				Name:                  "Products",
				SnowflakeDatabaseName: "PROD_DB",
				SnowflakeSchemaName:   "PUBLIC",
				SnowflakeTableName:    "PRODUCTS",
			},
			wantContains: []string{
				`cloud_mapping_snowflake_database = "PROD_DB"`,
				`cloud_mapping_snowflake_schema   = "PUBLIC"`,
				`cloud_mapping_snowflake_table    = "PRODUCTS"`,
			},
		},
		{
			name: "injects after cloud_link_id when present",
			hcl: `resource "elementum_element" "linked_element" {
  name          = "Linked Element"
  category_id   = "cat-123"
  cloud_link_id = "cloudlink-456"
}`,
			element: &discovery.Element{
				ID:                    "elem-456",
				Name:                  "Linked Element",
				CloudLinkID:           "cloudlink-456",
				SnowflakeDatabaseName: "ANALYTICS_DB",
				SnowflakeSchemaName:   "DW",
				SnowflakeTableName:    "FACT_SALES",
			},
			wantContains: []string{
				`cloud_mapping_snowflake_database = "ANALYTICS_DB"`,
				`cloud_mapping_snowflake_schema   = "DW"`,
				`cloud_mapping_snowflake_table    = "FACT_SALES"`,
			},
		},
		{
			name: "injects partial Snowflake fields",
			hcl: `resource "elementum_element" "partial" {
  name = "Partial"
  category_id = "cat-123"
}`,
			element: &discovery.Element{
				ID:                    "elem-789",
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
			hcl: `resource "elementum_element" "no_snowflake" {
  name = "No Snowflake"
  category_id = "cat-123"
}`,
			element: &discovery.Element{
				ID:   "elem-000",
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
			name: "nil element returns unchanged HCL",
			hcl: `resource "elementum_element" "test" {
  name = "Test"
  category_id = "cat-123"
}`,
			element: nil,
			wantMissing: []string{
				"cloud_mapping_snowflake",
			},
		},
		{
			name: "handles special characters in values",
			hcl: `resource "elementum_element" "special" {
  name = "Special"
  category_id = "cat-123"
}`,
			element: &discovery.Element{
				ID:                    "elem-special",
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
		{
			name: "handles element name with spaces",
			hcl: `resource "elementum_element" "my_products" {
  name = "My Products"
  category_id = "cat-123"
}`,
			element: &discovery.Element{
				ID:                    "elem-spaces",
				Name:                  "My Products",
				SnowflakeDatabaseName: "WAREHOUSE",
				SnowflakeSchemaName:   "RETAIL",
				SnowflakeTableName:    "PRODUCTS",
			},
			wantContains: []string{
				`cloud_mapping_snowflake_database = "WAREHOUSE"`,
				`cloud_mapping_snowflake_schema   = "RETAIL"`,
				`cloud_mapping_snowflake_table    = "PRODUCTS"`,
			},
		},
		{
			name: "handles element name with special chars",
			hcl: `resource "elementum_element" "products_v2" {
  name = "Products-V2"
  category_id = "cat-123"
}`,
			element: &discovery.Element{
				ID:                    "elem-v2",
				Name:                  "Products-V2",
				SnowflakeDatabaseName: "PROD",
				SnowflakeSchemaName:   "V2",
				SnowflakeTableName:    "ITEMS",
			},
			wantContains: []string{
				`cloud_mapping_snowflake_database = "PROD"`,
				`cloud_mapping_snowflake_schema   = "V2"`,
				`cloud_mapping_snowflake_table    = "ITEMS"`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := injectElementCloudMapping(tt.hcl, tt.element)

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

// Test injectElementCloudMapping preserves existing HCL structure
func TestInjectElementCloudMapping_PreservesHCLStructure(t *testing.T) {
	input := `resource "elementum_element" "test_element" {
  name        = "Test Element"
  category_id = "cat-123"
  description = "A test element"
  icon        = "box"
  color       = "#FF0000"
}`

	element := &discovery.Element{
		ID:                    "elem-123",
		Name:                  "Test Element",
		SnowflakeDatabaseName: "TEST_DB",
		SnowflakeSchemaName:   "TEST_SCHEMA",
		SnowflakeTableName:    "TEST_TABLE",
	}

	result := injectElementCloudMapping(input, element)

	// Should still contain all original attributes
	if !strings.Contains(result, `name        = "Test Element"`) {
		t.Error("Original name attribute missing")
	}
	if !strings.Contains(result, `category_id = "cat-123"`) {
		t.Error("Original category_id attribute missing")
	}
	if !strings.Contains(result, `description = "A test element"`) {
		t.Error("Original description attribute missing")
	}
	if !strings.Contains(result, `icon        = "box"`) {
		t.Error("Original icon attribute missing")
	}
	if !strings.Contains(result, `color       = "#FF0000"`) {
		t.Error("Original color attribute missing")
	}

	// And should have the new Snowflake fields
	if !strings.Contains(result, `cloud_mapping_snowflake_database = "TEST_DB"`) {
		t.Error("Snowflake database not injected")
	}
}
