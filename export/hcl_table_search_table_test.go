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

func TestTableSearchTableHCLGenerator_GenerateAll(t *testing.T) {
	app := &discovery.App{
		ID:        "app-123",
		Name:      "Test App",
		Namespace: "test_app",
		DiscoveredTables: []*discovery.Table{
			{
				ID:   "table-123",
				Name: "Customer Data",
				SearchTables: []discovery.TableSearchTable{
					{
						ID:                    "st-1",
						TableID:               "table-123",
						FieldID:               "field-desc",
						FieldName:             "Description",
						AIProviderConnectorID: "connector-1",
						AttributeFieldIDs:     []string{"field-title", "field-category"},
						Duration:              "DAYS",
						TargetLag:             1,
						Warehouse:             "COMPUTE_WH",
					},
				},
				Fields: []discovery.TableField{
					{ID: "field-desc", Name: "Description"},
					{ID: "field-title", Name: "Title"},
					{ID: "field-category", Name: "Category"},
				},
			},
		},
	}

	imports := []ImportBlock{
		{ID: "table-123", ResourceType: "elementum_table", ResourceName: "customer_data"},
		{ID: "table-123/st-1", ResourceType: "elementum_table_search_table", ResourceName: "description"},
	}

	uuidMap := map[string]string{
		"table-123":      "elementum_table.customer_data.id",
		"field-desc":     "elementum_longtext_field.description.id",
		"field-title":    "elementum_text_field.title.id",
		"field-category": "elementum_picklist_field.category.id",
	}

	gen := NewTableSearchTableHCLGenerator(app, imports, uuidMap)
	hcl := gen.GenerateAll()

	// Check resource declaration
	if !strings.Contains(hcl, `resource "elementum_table_search_table" "description"`) {
		t.Errorf("Expected table search table resource declaration, got:\n%s", hcl)
	}

	// Check table_id reference
	if !strings.Contains(hcl, `table_id = elementum_table.customer_data.id`) {
		t.Error("Expected table_id reference")
	}

	// Check field_id reference (from uuidMap)
	if !strings.Contains(hcl, `field_id = elementum_longtext_field.description.id`) {
		t.Error("Expected field_id reference from uuidMap")
	}

	// Check ai_provider_connector_id
	if !strings.Contains(hcl, `ai_provider_connector_id = "connector-1"`) {
		t.Error("Expected ai_provider_connector_id")
	}

	// Check attribute_field_ids
	if !strings.Contains(hcl, `attribute_field_ids = [`) {
		t.Error("Expected attribute_field_ids array")
	}

	// Check duration
	if !strings.Contains(hcl, `duration = "DAYS"`) {
		t.Error("Expected duration attribute")
	}

	// Check target_lag
	if !strings.Contains(hcl, `target_lag = 1`) {
		t.Error("Expected target_lag attribute")
	}

	// Check warehouse
	if !strings.Contains(hcl, `warehouse = "COMPUTE_WH"`) {
		t.Error("Expected warehouse attribute")
	}
}

func TestTableSearchTableHCLGenerator_WithoutWarehouse(t *testing.T) {
	app := &discovery.App{
		ID:        "app-123",
		Name:      "Test App",
		Namespace: "test_app",
		DiscoveredTables: []*discovery.Table{
			{
				ID:   "table-123",
				Name: "Customer Data",
				SearchTables: []discovery.TableSearchTable{
					{
						ID:                    "st-1",
						TableID:               "table-123",
						FieldID:               "field-desc",
						FieldName:             "Description",
						AIProviderConnectorID: "connector-1",
						AttributeFieldIDs:     []string{},
						Duration:              "HOURS",
						TargetLag:             12,
						Warehouse:             "", // No warehouse
					},
				},
			},
		},
	}

	imports := []ImportBlock{
		{ID: "table-123", ResourceType: "elementum_table", ResourceName: "customer_data"},
		{ID: "table-123/st-1", ResourceType: "elementum_table_search_table", ResourceName: "description"},
	}

	gen := NewTableSearchTableHCLGenerator(app, imports, make(map[string]string))
	hcl := gen.GenerateAll()

	// Check resource exists
	if !strings.Contains(hcl, `resource "elementum_table_search_table" "description"`) {
		t.Error("Expected table search table resource declaration")
	}

	// Check warehouse is NOT present
	if strings.Contains(hcl, `warehouse`) {
		t.Error("Warehouse should not be present when empty")
	}

	// Check empty attribute_field_ids is present
	if !strings.Contains(hcl, `attribute_field_ids = [`) {
		t.Error("Expected empty attribute_field_ids array")
	}

	// Check duration
	if !strings.Contains(hcl, `duration = "HOURS"`) {
		t.Error("Expected duration = HOURS")
	}

	// Check target_lag
	if !strings.Contains(hcl, `target_lag = 12`) {
		t.Error("Expected target_lag = 12")
	}
}

func TestTableSearchTableHCLGenerator_ReferencedTables(t *testing.T) {
	app := &discovery.App{
		ID:        "app-123",
		Name:      "Test App",
		Namespace: "test_app",
		ReferencedTables: []*discovery.Table{
			{
				ID:   "table-ref-123",
				Name: "Referenced Table",
				SearchTables: []discovery.TableSearchTable{
					{
						ID:                    "st-ref-1",
						TableID:               "table-ref-123",
						FieldID:               "field-notes",
						FieldName:             "Notes",
						AIProviderConnectorID: "connector-1",
						AttributeFieldIDs:     []string{},
						Duration:              "MINUTES",
						TargetLag:             30,
					},
				},
			},
		},
	}

	imports := []ImportBlock{
		{ID: "table-ref-123", ResourceType: "elementum_table", ResourceName: "referenced_table"},
		{ID: "table-ref-123/st-ref-1", ResourceType: "elementum_table_search_table", ResourceName: "notes"},
	}

	uuidMap := map[string]string{
		"table-ref-123": "elementum_table.referenced_table.id",
	}

	gen := NewTableSearchTableHCLGenerator(app, imports, uuidMap)
	hcl := gen.GenerateAll()

	// Check resource declaration for referenced table search table
	if !strings.Contains(hcl, `resource "elementum_table_search_table" "notes"`) {
		t.Errorf("Expected table search table resource for referenced table, got:\n%s", hcl)
	}

	// Check table_id points to referenced table
	if !strings.Contains(hcl, `table_id = elementum_table.referenced_table.id`) {
		t.Error("Expected table_id to reference table resource")
	}

	// Check duration
	if !strings.Contains(hcl, `duration = "MINUTES"`) {
		t.Error("Expected duration = MINUTES")
	}

	// Check target_lag
	if !strings.Contains(hcl, `target_lag = 30`) {
		t.Error("Expected target_lag = 30")
	}
}

func TestTableSearchTableHCLGenerator_MultipleSearchTables(t *testing.T) {
	app := &discovery.App{
		ID:        "app-123",
		Name:      "Test App",
		Namespace: "test_app",
		DiscoveredTables: []*discovery.Table{
			{
				ID:   "table-123",
				Name: "Customer Data",
				SearchTables: []discovery.TableSearchTable{
					{
						ID:                    "st-1",
						TableID:               "table-123",
						FieldID:               "field-desc",
						FieldName:             "Description",
						AIProviderConnectorID: "connector-1",
						Duration:              "DAYS",
						TargetLag:             1,
					},
					{
						ID:                    "st-2",
						TableID:               "table-123",
						FieldID:               "field-summary",
						FieldName:             "Summary",
						AIProviderConnectorID: "connector-1",
						Duration:              "DAYS",
						TargetLag:             1,
					},
				},
			},
		},
	}

	imports := []ImportBlock{
		{ID: "table-123", ResourceType: "elementum_table", ResourceName: "customer_data"},
		{ID: "table-123/st-1", ResourceType: "elementum_table_search_table", ResourceName: "description"},
		{ID: "table-123/st-2", ResourceType: "elementum_table_search_table", ResourceName: "summary"},
	}

	gen := NewTableSearchTableHCLGenerator(app, imports, make(map[string]string))
	hcl := gen.GenerateAll()

	// Check both resources are generated
	if !strings.Contains(hcl, `resource "elementum_table_search_table" "description"`) {
		t.Error("Expected first table search table resource")
	}
	if !strings.Contains(hcl, `resource "elementum_table_search_table" "summary"`) {
		t.Error("Expected second table search table resource")
	}
}

func TestTableSearchTableHCLGenerator_UUIDFallback(t *testing.T) {
	app := &discovery.App{
		ID:        "app-123",
		Name:      "Test App",
		Namespace: "test_app",
		DiscoveredTables: []*discovery.Table{
			{
				ID:   "table-123",
				Name: "Customer Data",
				SearchTables: []discovery.TableSearchTable{
					{
						ID:                    "st-1",
						TableID:               "table-123",
						FieldID:               "unknown-field-uuid",
						FieldName:             "Unknown Field",
						AIProviderConnectorID: "unknown-connector-uuid",
						AttributeFieldIDs:     []string{"unknown-attr-1", "unknown-attr-2"},
						Duration:              "SECONDS",
						TargetLag:             60,
					},
				},
			},
		},
	}

	imports := []ImportBlock{
		{ID: "table-123", ResourceType: "elementum_table", ResourceName: "customer_data"},
		{ID: "table-123/st-1", ResourceType: "elementum_table_search_table", ResourceName: "unknown_field"},
	}

	// Empty UUID map - everything should fall back to quoted UUIDs
	gen := NewTableSearchTableHCLGenerator(app, imports, make(map[string]string))
	hcl := gen.GenerateAll()

	// Check ai_provider_connector_id falls back to quoted UUID
	if !strings.Contains(hcl, `ai_provider_connector_id = "unknown-connector-uuid"`) {
		t.Errorf("Expected ai_provider_connector_id to fall back to quoted UUID, got:\n%s", hcl)
	}

	// Check attribute_field_ids fall back to quoted UUIDs
	if !strings.Contains(hcl, `"unknown-attr-1"`) {
		t.Error("Expected attribute field UUID fallback")
	}
	if !strings.Contains(hcl, `"unknown-attr-2"`) {
		t.Error("Expected attribute field UUID fallback")
	}
}

func TestTableSearchTableHCLGenerator_EmptyApp(t *testing.T) {
	app := &discovery.App{
		ID:               "app-123",
		Name:             "Test App",
		Namespace:        "test_app",
		DiscoveredTables: []*discovery.Table{},
		ReferencedTables: []*discovery.Table{},
	}

	gen := NewTableSearchTableHCLGenerator(app, nil, make(map[string]string))
	hcl := gen.GenerateAll()

	if hcl != "" {
		t.Errorf("Expected empty output for app with no search tables, got: %s", hcl)
	}
}

func TestTableSearchTableHCLGenerator_NilApp(t *testing.T) {
	gen := NewTableSearchTableHCLGenerator(nil, nil, make(map[string]string))
	hcl := gen.GenerateAll()

	if hcl != "" {
		t.Errorf("Expected empty output for nil app, got: %s", hcl)
	}
}

func TestTableSearchTableHCLGenerator_MissingImport(t *testing.T) {
	app := &discovery.App{
		ID:        "app-123",
		Name:      "Test App",
		Namespace: "test_app",
		DiscoveredTables: []*discovery.Table{
			{
				ID:   "table-123",
				Name: "Customer Data",
				SearchTables: []discovery.TableSearchTable{
					{
						ID:                    "st-1",
						TableID:               "table-123",
						FieldID:               "field-desc",
						FieldName:             "Description",
						AIProviderConnectorID: "connector-1",
						Duration:              "DAYS",
						TargetLag:             1,
					},
				},
			},
		},
	}

	// No import block for the search table
	imports := []ImportBlock{
		{ID: "table-123", ResourceType: "elementum_table", ResourceName: "customer_data"},
	}

	gen := NewTableSearchTableHCLGenerator(app, imports, make(map[string]string))
	hcl := gen.GenerateAll()

	// Should not generate HCL for search tables without import blocks
	if strings.Contains(hcl, "elementum_table_search_table") {
		t.Error("Should not generate HCL for search table without import block")
	}
}

func TestTableSearchTableHCLGenerator_AllDurationTypes(t *testing.T) {
	durations := []struct {
		name     string
		duration string
	}{
		{"DAYS", "DAYS"},
		{"HOURS", "HOURS"},
		{"MINUTES", "MINUTES"},
		{"SECONDS", "SECONDS"},
	}

	for _, tc := range durations {
		t.Run(tc.name, func(t *testing.T) {
			app := &discovery.App{
				ID:        "app-123",
				Name:      "Test App",
				Namespace: "test_app",
				DiscoveredTables: []*discovery.Table{
					{
						ID:   "table-123",
						Name: "Test Table",
						SearchTables: []discovery.TableSearchTable{
							{
								ID:                    "st-1",
								TableID:               "table-123",
								FieldID:               "field-1",
								FieldName:             "Field 1",
								AIProviderConnectorID: "connector-1",
								Duration:              tc.duration,
								TargetLag:             1,
							},
						},
					},
				},
			}

			imports := []ImportBlock{
				{ID: "table-123", ResourceType: "elementum_table", ResourceName: "test_table"},
				{ID: "table-123/st-1", ResourceType: "elementum_table_search_table", ResourceName: "field_1"},
			}

			gen := NewTableSearchTableHCLGenerator(app, imports, make(map[string]string))
			hcl := gen.GenerateAll()

			expected := `duration = "` + tc.duration + `"`
			if !strings.Contains(hcl, expected) {
				t.Errorf("Expected %s, got:\n%s", expected, hcl)
			}
		})
	}
}

func TestTableSearchTableHCLGenerator_MixedReferencedAndDiscovered(t *testing.T) {
	app := &discovery.App{
		ID:        "app-123",
		Name:      "Test App",
		Namespace: "test_app",
		ReferencedTables: []*discovery.Table{
			{
				ID:   "table-ref",
				Name: "Referenced Table",
				SearchTables: []discovery.TableSearchTable{
					{
						ID:                    "st-ref",
						TableID:               "table-ref",
						FieldID:               "field-ref-desc",
						FieldName:             "Ref Description",
						AIProviderConnectorID: "connector-1",
						Duration:              "HOURS",
						TargetLag:             6,
					},
				},
			},
		},
		DiscoveredTables: []*discovery.Table{
			{
				ID:   "table-disc",
				Name: "Discovered Table",
				SearchTables: []discovery.TableSearchTable{
					{
						ID:                    "st-disc",
						TableID:               "table-disc",
						FieldID:               "field-disc-notes",
						FieldName:             "Disc Notes",
						AIProviderConnectorID: "connector-1",
						Duration:              "DAYS",
						TargetLag:             1,
					},
				},
			},
		},
	}

	imports := []ImportBlock{
		{ID: "table-ref", ResourceType: "elementum_table", ResourceName: "referenced_table"},
		{ID: "table-disc", ResourceType: "elementum_table", ResourceName: "discovered_table"},
		{ID: "table-ref/st-ref", ResourceType: "elementum_table_search_table", ResourceName: "ref_description"},
		{ID: "table-disc/st-disc", ResourceType: "elementum_table_search_table", ResourceName: "disc_notes"},
	}

	gen := NewTableSearchTableHCLGenerator(app, imports, make(map[string]string))
	hcl := gen.GenerateAll()

	// Check referenced table search table
	if !strings.Contains(hcl, `resource "elementum_table_search_table" "ref_description"`) {
		t.Error("Expected search table from referenced table")
	}

	// Check discovered table search table
	if !strings.Contains(hcl, `resource "elementum_table_search_table" "disc_notes"`) {
		t.Error("Expected search table from discovered table")
	}
}

func TestTableSearchTableHCLGenerator_FieldIdLookupFromTable(t *testing.T) {
	// Test that field_id is resolved using the local.<table>_field_ids_by_name lookup
	app := &discovery.App{
		ID:        "app-123",
		Name:      "Test App",
		Namespace: "test_app",
		DiscoveredTables: []*discovery.Table{
			{
				ID:   "table-123",
				Name: "Customer Data",
				SearchTables: []discovery.TableSearchTable{
					{
						ID:                    "st-1",
						TableID:               "table-123",
						FieldID:               "field-abc-123",
						FieldName:             "Description Field",
						AIProviderConnectorID: "connector-1",
						Duration:              "DAYS",
						TargetLag:             1,
					},
				},
				Fields: []discovery.TableField{
					{ID: "field-abc-123", Name: "Description Field"},
				},
			},
		},
	}

	imports := []ImportBlock{
		{ID: "table-123", ResourceType: "elementum_table", ResourceName: "customer_data"},
		{ID: "table-123/st-1", ResourceType: "elementum_table_search_table", ResourceName: "description_field"},
	}

	// No field ID in uuidMap - should use local lookup
	gen := NewTableSearchTableHCLGenerator(app, imports, make(map[string]string))
	hcl := gen.GenerateAll()

	// Should use local lookup pattern
	if !strings.Contains(hcl, `local.customer_data_field_ids_by_name["Description Field"]`) {
		t.Errorf("Expected field_id to use local lookup, got:\n%s", hcl)
	}
}

func TestTableSearchTableHCLGenerator_FieldIdFromUUIDMap(t *testing.T) {
	// Test that field_id from uuidMap takes precedence
	app := &discovery.App{
		ID:        "app-123",
		Name:      "Test App",
		Namespace: "test_app",
		DiscoveredTables: []*discovery.Table{
			{
				ID:   "table-123",
				Name: "Customer Data",
				SearchTables: []discovery.TableSearchTable{
					{
						ID:                    "st-1",
						TableID:               "table-123",
						FieldID:               "field-abc-123",
						FieldName:             "Description",
						AIProviderConnectorID: "connector-1",
						Duration:              "DAYS",
						TargetLag:             1,
					},
				},
			},
		},
	}

	imports := []ImportBlock{
		{ID: "table-123", ResourceType: "elementum_table", ResourceName: "customer_data"},
		{ID: "table-123/st-1", ResourceType: "elementum_table_search_table", ResourceName: "description"},
	}

	// Field ID in uuidMap - should use the reference
	uuidMap := map[string]string{
		"field-abc-123": "some_other_resource.description.id",
	}

	gen := NewTableSearchTableHCLGenerator(app, imports, uuidMap)
	hcl := gen.GenerateAll()

	// Should use the reference from uuidMap
	if !strings.Contains(hcl, `field_id = some_other_resource.description.id`) {
		t.Errorf("Expected field_id from uuidMap, got:\n%s", hcl)
	}
}

func TestTableSearchTableHCLGenerator_AttributeFieldIdsResolution(t *testing.T) {
	// Test that attribute_field_ids are resolved correctly
	app := &discovery.App{
		ID:        "app-123",
		Name:      "Test App",
		Namespace: "test_app",
		DiscoveredTables: []*discovery.Table{
			{
				ID:   "table-123",
				Name: "Customer Data",
				SearchTables: []discovery.TableSearchTable{
					{
						ID:                    "st-1",
						TableID:               "table-123",
						FieldID:               "field-desc",
						FieldName:             "Description",
						AIProviderConnectorID: "connector-1",
						AttributeFieldIDs:     []string{"field-attr-1", "field-attr-2", "field-attr-3"},
						Duration:              "DAYS",
						TargetLag:             1,
					},
				},
				Fields: []discovery.TableField{
					{ID: "field-desc", Name: "Description"},
					{ID: "field-attr-1", Name: "Title"},
					{ID: "field-attr-2", Name: "Category"},
					{ID: "field-attr-3", Name: "Unknown"}, // In fields but not in uuidMap
				},
			},
		},
	}

	imports := []ImportBlock{
		{ID: "table-123", ResourceType: "elementum_table", ResourceName: "customer_data"},
		{ID: "table-123/st-1", ResourceType: "elementum_table_search_table", ResourceName: "description"},
	}

	// Only some fields in uuidMap
	uuidMap := map[string]string{
		"field-attr-1": "elementum_text_field.title.id",
	}

	gen := NewTableSearchTableHCLGenerator(app, imports, uuidMap)
	hcl := gen.GenerateAll()

	// First attribute should use uuidMap reference
	if !strings.Contains(hcl, `elementum_text_field.title.id`) {
		t.Error("Expected first attribute to use uuidMap reference")
	}

	// Other attributes should use local lookup
	if !strings.Contains(hcl, `local.customer_data_field_ids_by_name["Category"]`) {
		t.Error("Expected second attribute to use local lookup")
	}
}
