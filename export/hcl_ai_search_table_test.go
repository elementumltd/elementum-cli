// Copyright (c) ElementumAI, Inc.
// SPDX-License-Identifier: MPL-2.0

package export

import (
	"strings"
	"testing"

	"github.com/elementumltd/elementum-cli/discovery"
)

func TestAISearchTableHCLGenerator_GenerateAll(t *testing.T) {
	app := &discovery.App{
		ID:        "app-123",
		Name:      "Test App",
		Namespace: "test_app",
		AISearchTables: []discovery.AISearchTable{
			{
				ID:                      "st-1",
				ObjectID:                "app-123",
				FieldID:                 "field-desc",
				FieldName:               "Description",
				AIProviderConnectorID:   "connector-1",
				AIProviderConnectorName: "text-embedding-3-small",
				AttributeFieldIDs:       []string{"field-title", "field-category"},
				Duration:                "DAYS",
				TargetLag:               1,
				Warehouse:               "COMPUTE_WH",
			},
		},
	}

	imports := []ImportBlock{
		{ID: "app-123", ResourceType: "elementum_app", ResourceName: "test_app"},
		{ID: "app-123:st-1", ResourceType: "elementum_ai_search_table", ResourceName: "description"},
	}

	uuidMap := map[string]string{
		"app-123":        "elementum_app.test_app.id",
		"field-desc":     "elementum_longtext_field.description.id",
		"field-title":    "elementum_text_field.title.id",
		"field-category": "elementum_picklist_field.category.id",
	}

	gen := NewAISearchTableHCLGenerator(app, imports, uuidMap)
	hcl := gen.GenerateAll()

	// Check resource declaration
	if !strings.Contains(hcl, `resource "elementum_ai_search_table" "description"`) {
		t.Errorf("Expected AI search table resource declaration, got:\n%s", hcl)
	}

	// Check object_id reference
	if !strings.Contains(hcl, `object_id = elementum_app.test_app.id`) {
		t.Error("Expected object_id reference")
	}

	// Check field_id reference
	if !strings.Contains(hcl, `field_id = elementum_longtext_field.description.id`) {
		t.Error("Expected field_id reference")
	}

	// Check ai_provider_connector_id with comment
	if !strings.Contains(hcl, `ai_provider_connector_id = "connector-1"`) {
		t.Error("Expected ai_provider_connector_id")
	}
	if !strings.Contains(hcl, `text-embedding-3-small`) {
		t.Error("Expected model name comment for ai_provider_connector_id")
	}

	// Check attribute_field_ids
	if !strings.Contains(hcl, `attribute_field_ids = [`) {
		t.Error("Expected attribute_field_ids array")
	}
	if !strings.Contains(hcl, `elementum_text_field.title.id`) {
		t.Error("Expected title field reference in attribute_field_ids")
	}
	if !strings.Contains(hcl, `elementum_picklist_field.category.id`) {
		t.Error("Expected category field reference in attribute_field_ids")
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

func TestAISearchTableHCLGenerator_WithoutWarehouse(t *testing.T) {
	app := &discovery.App{
		ID:        "app-123",
		Name:      "Test App",
		Namespace: "test_app",
		AISearchTables: []discovery.AISearchTable{
			{
				ID:                      "st-1",
				ObjectID:                "app-123",
				FieldID:                 "field-desc",
				FieldName:               "Description",
				AIProviderConnectorID:   "connector-1",
				AIProviderConnectorName: "text-embedding-3-small",
				AttributeFieldIDs:       []string{},
				Duration:                "HOURS",
				TargetLag:               12,
				Warehouse:               "", // No warehouse
			},
		},
	}

	imports := []ImportBlock{
		{ID: "app-123", ResourceType: "elementum_app", ResourceName: "test_app"},
		{ID: "app-123:st-1", ResourceType: "elementum_ai_search_table", ResourceName: "description"},
	}

	gen := NewAISearchTableHCLGenerator(app, imports, make(map[string]string))
	hcl := gen.GenerateAll()

	// Check resource exists
	if !strings.Contains(hcl, `resource "elementum_ai_search_table" "description"`) {
		t.Error("Expected AI search table resource declaration")
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

func TestAISearchTableHCLGenerator_ElementLevel(t *testing.T) {
	app := &discovery.App{
		ID:             "app-123",
		Name:           "Test App",
		Namespace:      "test_app",
		AISearchTables: []discovery.AISearchTable{}, // No app-level search tables
		DiscoveredElements: []*discovery.Element{
			{
				ID:        "elem-1",
				Name:      "Line Items",
				Namespace: "line_items",
				AISearchTables: []discovery.AISearchTable{
					{
						ID:                      "st-elem-1",
						ObjectID:                "elem-1",
						FieldID:                 "field-notes",
						FieldName:               "Notes",
						AIProviderConnectorID:   "connector-2",
						AIProviderConnectorName: "ada-002",
						AttributeFieldIDs:       []string{"field-amount"},
						Duration:                "MINUTES",
						TargetLag:               30,
					},
				},
			},
		},
	}

	imports := []ImportBlock{
		{ID: "app-123", ResourceType: "elementum_app", ResourceName: "test_app"},
		{ID: "elem-1", ResourceType: "elementum_element", ResourceName: "line_items"},
		{ID: "elem-1:st-elem-1", ResourceType: "elementum_ai_search_table", ResourceName: "notes"},
	}

	uuidMap := map[string]string{
		"elem-1":       "elementum_element.line_items.id",
		"field-notes":  "elementum_longtext_field.notes.id",
		"field-amount": "elementum_decimal_field.amount.id",
	}

	gen := NewAISearchTableHCLGenerator(app, imports, uuidMap)
	hcl := gen.GenerateAll()

	// Check resource declaration
	if !strings.Contains(hcl, `resource "elementum_ai_search_table" "notes"`) {
		t.Errorf("Expected AI search table resource for element, got:\n%s", hcl)
	}

	// Check object_id points to element
	if !strings.Contains(hcl, `object_id = elementum_element.line_items.id`) {
		t.Error("Expected object_id to reference element")
	}

	// Check field reference
	if !strings.Contains(hcl, `field_id = elementum_longtext_field.notes.id`) {
		t.Error("Expected field_id reference")
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

func TestAISearchTableHCLGenerator_MultipleSearchTables(t *testing.T) {
	app := &discovery.App{
		ID:        "app-123",
		Name:      "Test App",
		Namespace: "test_app",
		AISearchTables: []discovery.AISearchTable{
			{
				ID:                      "st-1",
				ObjectID:                "app-123",
				FieldID:                 "field-desc",
				FieldName:               "Description",
				AIProviderConnectorID:   "connector-1",
				AIProviderConnectorName: "text-embedding-3-small",
				AttributeFieldIDs:       []string{},
				Duration:                "DAYS",
				TargetLag:               1,
			},
			{
				ID:                      "st-2",
				ObjectID:                "app-123",
				FieldID:                 "field-summary",
				FieldName:               "Summary",
				AIProviderConnectorID:   "connector-1",
				AIProviderConnectorName: "text-embedding-3-small",
				AttributeFieldIDs:       []string{},
				Duration:                "DAYS",
				TargetLag:               1,
			},
		},
	}

	imports := []ImportBlock{
		{ID: "app-123", ResourceType: "elementum_app", ResourceName: "test_app"},
		{ID: "app-123:st-1", ResourceType: "elementum_ai_search_table", ResourceName: "description"},
		{ID: "app-123:st-2", ResourceType: "elementum_ai_search_table", ResourceName: "summary"},
	}

	gen := NewAISearchTableHCLGenerator(app, imports, make(map[string]string))
	hcl := gen.GenerateAll()

	// Check both resources are generated
	if !strings.Contains(hcl, `resource "elementum_ai_search_table" "description"`) {
		t.Error("Expected first AI search table resource")
	}
	if !strings.Contains(hcl, `resource "elementum_ai_search_table" "summary"`) {
		t.Error("Expected second AI search table resource")
	}
}

func TestAISearchTableHCLGenerator_UUIDFallback(t *testing.T) {
	app := &discovery.App{
		ID:        "app-123",
		Name:      "Test App",
		Namespace: "test_app",
		AISearchTables: []discovery.AISearchTable{
			{
				ID:                      "st-1",
				ObjectID:                "app-123",
				FieldID:                 "unknown-field-uuid",
				FieldName:               "Unknown Field",
				AIProviderConnectorID:   "unknown-connector-uuid",
				AIProviderConnectorName: "",
				AttributeFieldIDs:       []string{"unknown-attr-1", "unknown-attr-2"},
				Duration:                "SECONDS",
				TargetLag:               60,
			},
		},
	}

	imports := []ImportBlock{
		{ID: "app-123", ResourceType: "elementum_app", ResourceName: "test_app"},
		{ID: "app-123:st-1", ResourceType: "elementum_ai_search_table", ResourceName: "unknown_field"},
	}

	// Empty UUID map - everything should fall back to quoted UUIDs
	gen := NewAISearchTableHCLGenerator(app, imports, make(map[string]string))
	hcl := gen.GenerateAll()

	// Check field_id falls back to quoted UUID
	if !strings.Contains(hcl, `field_id = "unknown-field-uuid"`) {
		t.Errorf("Expected field_id to fall back to quoted UUID, got:\n%s", hcl)
	}

	// Check ai_provider_connector_id falls back to quoted UUID (no comment since name is empty)
	if !strings.Contains(hcl, `ai_provider_connector_id = "unknown-connector-uuid"`) {
		t.Error("Expected ai_provider_connector_id to fall back to quoted UUID")
	}

	// Check attribute_field_ids fall back to quoted UUIDs
	if !strings.Contains(hcl, `"unknown-attr-1"`) {
		t.Error("Expected attribute field UUID fallback")
	}
	if !strings.Contains(hcl, `"unknown-attr-2"`) {
		t.Error("Expected attribute field UUID fallback")
	}
}

func TestAISearchTableHCLGenerator_EmptyApp(t *testing.T) {
	app := &discovery.App{
		ID:             "app-123",
		Name:           "Test App",
		Namespace:      "test_app",
		AISearchTables: []discovery.AISearchTable{},
	}

	gen := NewAISearchTableHCLGenerator(app, nil, make(map[string]string))
	hcl := gen.GenerateAll()

	if hcl != "" {
		t.Errorf("Expected empty output for app with no search tables, got: %s", hcl)
	}
}

func TestAISearchTableHCLGenerator_NilApp(t *testing.T) {
	gen := NewAISearchTableHCLGenerator(nil, nil, make(map[string]string))
	hcl := gen.GenerateAll()

	if hcl != "" {
		t.Errorf("Expected empty output for nil app, got: %s", hcl)
	}
}

func TestAISearchTableHCLGenerator_MissingImport(t *testing.T) {
	app := &discovery.App{
		ID:        "app-123",
		Name:      "Test App",
		Namespace: "test_app",
		AISearchTables: []discovery.AISearchTable{
			{
				ID:                      "st-1",
				ObjectID:                "app-123",
				FieldID:                 "field-desc",
				FieldName:               "Description",
				AIProviderConnectorID:   "connector-1",
				AIProviderConnectorName: "text-embedding-3-small",
				AttributeFieldIDs:       []string{},
				Duration:                "DAYS",
				TargetLag:               1,
			},
		},
	}

	// No import block for the search table
	imports := []ImportBlock{
		{ID: "app-123", ResourceType: "elementum_app", ResourceName: "test_app"},
	}

	gen := NewAISearchTableHCLGenerator(app, imports, make(map[string]string))
	hcl := gen.GenerateAll()

	// Should not generate HCL for search tables without import blocks
	if strings.Contains(hcl, "elementum_ai_search_table") {
		t.Error("Should not generate HCL for search table without import block")
	}
}

func TestAISearchTableHCLGenerator_AllDurationTypes(t *testing.T) {
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
				AISearchTables: []discovery.AISearchTable{
					{
						ID:                      "st-1",
						ObjectID:                "app-123",
						FieldID:                 "field-1",
						AIProviderConnectorID:   "connector-1",
						AIProviderConnectorName: "model",
						Duration:                tc.duration,
						TargetLag:               1,
					},
				},
			}

			imports := []ImportBlock{
				{ID: "app-123", ResourceType: "elementum_app", ResourceName: "test_app"},
				{ID: "app-123:st-1", ResourceType: "elementum_ai_search_table", ResourceName: "field_1"},
			}

			gen := NewAISearchTableHCLGenerator(app, imports, make(map[string]string))
			hcl := gen.GenerateAll()

			expected := `duration = "` + tc.duration + `"`
			if !strings.Contains(hcl, expected) {
				t.Errorf("Expected %s, got:\n%s", expected, hcl)
			}
		})
	}
}

func TestAISearchTableHCLGenerator_MixedAppAndElement(t *testing.T) {
	app := &discovery.App{
		ID:        "app-123",
		Name:      "Test App",
		Namespace: "test_app",
		AISearchTables: []discovery.AISearchTable{
			{
				ID:                      "st-app-1",
				ObjectID:                "app-123",
				FieldID:                 "field-app-desc",
				FieldName:               "App Description",
				AIProviderConnectorID:   "connector-1",
				AIProviderConnectorName: "embedding-model",
				Duration:                "DAYS",
				TargetLag:               1,
			},
		},
		DiscoveredElements: []*discovery.Element{
			{
				ID:        "elem-1",
				Name:      "Child Element",
				Namespace: "child_element",
				AISearchTables: []discovery.AISearchTable{
					{
						ID:                      "st-elem-1",
						ObjectID:                "elem-1",
						FieldID:                 "field-elem-notes",
						FieldName:               "Element Notes",
						AIProviderConnectorID:   "connector-1",
						AIProviderConnectorName: "embedding-model",
						Duration:                "HOURS",
						TargetLag:               6,
					},
				},
			},
		},
	}

	imports := []ImportBlock{
		{ID: "app-123", ResourceType: "elementum_app", ResourceName: "test_app"},
		{ID: "elem-1", ResourceType: "elementum_element", ResourceName: "child_element"},
		{ID: "app-123:st-app-1", ResourceType: "elementum_ai_search_table", ResourceName: "app_description"},
		{ID: "elem-1:st-elem-1", ResourceType: "elementum_ai_search_table", ResourceName: "element_notes"},
	}

	gen := NewAISearchTableHCLGenerator(app, imports, make(map[string]string))
	hcl := gen.GenerateAll()

	// Check app-level search table
	if !strings.Contains(hcl, `resource "elementum_ai_search_table" "app_description"`) {
		t.Error("Expected app-level AI search table resource")
	}
	if !strings.Contains(hcl, `object_id = elementum_app.test_app.id`) {
		t.Error("Expected app object_id reference")
	}

	// Check element-level search table
	if !strings.Contains(hcl, `resource "elementum_ai_search_table" "element_notes"`) {
		t.Error("Expected element-level AI search table resource")
	}
	if !strings.Contains(hcl, `object_id = elementum_element.child_element.id`) {
		t.Error("Expected element object_id reference")
	}
}

// TestAISearchTableHCLGenerator_ElementAsDataSource tests that element-level search tables
// correctly reference elements as data sources when the uuidMap contains a data source reference.
// This happens when --recursive flag is OFF and elements are discovered through relationships.
func TestAISearchTableHCLGenerator_ElementAsDataSource(t *testing.T) {
	app := &discovery.App{
		ID:             "app-123",
		Name:           "Test App",
		Namespace:      "test_app",
		AISearchTables: []discovery.AISearchTable{},
		DiscoveredElements: []*discovery.Element{
			{
				ID:        "elem-velocity-actions",
				Name:      "Velocity Action Versions - DEV",
				Namespace: "velocity_action_versions___dev",
				AISearchTables: []discovery.AISearchTable{
					{
						ID:                      "st-elem-1",
						ObjectID:                "elem-velocity-actions",
						FieldID:                 "field-notes",
						FieldName:               "Notes",
						AIProviderConnectorID:   "connector-1",
						AIProviderConnectorName: "text-embedding-3-small",
						AttributeFieldIDs:       []string{},
						Duration:                "DAYS",
						TargetLag:               1,
					},
				},
			},
		},
	}

	imports := []ImportBlock{
		{ID: "app-123", ResourceType: "elementum_app", ResourceName: "test_app"},
		{ID: "elem-velocity-actions:st-elem-1", ResourceType: "elementum_ai_search_table", ResourceName: "velocity_action_versions___dev_notes"},
	}

	// uuidMap with DATA SOURCE reference (simulates --recursive OFF scenario)
	// where elements are discovered through relationships but not exported as resources
	uuidMap := map[string]string{
		"elem-velocity-actions": "data.elementum_element.velocity_action_versions___dev.id",
		"field-notes":           "elementum_longtext_field.notes.id",
	}

	gen := NewAISearchTableHCLGenerator(app, imports, uuidMap)
	hcl := gen.GenerateAll()

	// Check resource declaration exists
	if !strings.Contains(hcl, `resource "elementum_ai_search_table" "velocity_action_versions___dev_notes"`) {
		t.Errorf("Expected AI search table resource declaration, got:\n%s", hcl)
	}

	// CRITICAL: Check object_id uses DATA SOURCE reference from uuidMap
	if !strings.Contains(hcl, `object_id = data.elementum_element.velocity_action_versions___dev.id`) {
		t.Errorf("Expected object_id to use data source reference from uuidMap, got:\n%s", hcl)
	}

	// Should NOT contain resource reference
	if strings.Contains(hcl, `object_id = elementum_element.velocity_action_versions___dev.id`) {
		t.Error("Should NOT use resource reference when uuidMap contains data source reference")
	}
}

// TestAISearchTableHCLGenerator_ElementAsResource tests that element-level search tables
// correctly reference elements as resources when the uuidMap contains a resource reference.
// This happens when --recursive flag is ON and elements are fully exported.
func TestAISearchTableHCLGenerator_ElementAsResource(t *testing.T) {
	app := &discovery.App{
		ID:             "app-123",
		Name:           "Test App",
		Namespace:      "test_app",
		AISearchTables: []discovery.AISearchTable{},
		DiscoveredElements: []*discovery.Element{
			{
				ID:        "elem-velocity-actions",
				Name:      "Velocity Action Versions - DEV",
				Namespace: "velocity_action_versions___dev",
				AISearchTables: []discovery.AISearchTable{
					{
						ID:                      "st-elem-1",
						ObjectID:                "elem-velocity-actions",
						FieldID:                 "field-notes",
						FieldName:               "Notes",
						AIProviderConnectorID:   "connector-1",
						AIProviderConnectorName: "text-embedding-3-small",
						AttributeFieldIDs:       []string{},
						Duration:                "DAYS",
						TargetLag:               1,
					},
				},
			},
		},
	}

	imports := []ImportBlock{
		{ID: "app-123", ResourceType: "elementum_app", ResourceName: "test_app"},
		{ID: "elem-velocity-actions", ResourceType: "elementum_element", ResourceName: "velocity_action_versions___dev"},
		{ID: "elem-velocity-actions:st-elem-1", ResourceType: "elementum_ai_search_table", ResourceName: "velocity_action_versions___dev_notes"},
	}

	// uuidMap with RESOURCE reference (simulates --recursive ON scenario)
	uuidMap := map[string]string{
		"elem-velocity-actions": "elementum_element.velocity_action_versions___dev.id",
		"field-notes":           "elementum_longtext_field.notes.id",
	}

	gen := NewAISearchTableHCLGenerator(app, imports, uuidMap)
	hcl := gen.GenerateAll()

	// Check resource declaration exists
	if !strings.Contains(hcl, `resource "elementum_ai_search_table" "velocity_action_versions___dev_notes"`) {
		t.Errorf("Expected AI search table resource declaration, got:\n%s", hcl)
	}

	// CRITICAL: Check object_id uses RESOURCE reference from uuidMap
	if !strings.Contains(hcl, `object_id = elementum_element.velocity_action_versions___dev.id`) {
		t.Errorf("Expected object_id to use resource reference from uuidMap, got:\n%s", hcl)
	}

	// Should NOT contain data source reference
	if strings.Contains(hcl, `object_id = data.elementum_element`) {
		t.Error("Should NOT use data source reference when uuidMap contains resource reference")
	}
}

// TestAISearchTableHCLGenerator_MultipleElementsMixedRefs tests multiple elements
// where some are resources and some are data sources based on uuidMap.
func TestAISearchTableHCLGenerator_MultipleElementsMixedRefs(t *testing.T) {
	app := &discovery.App{
		ID:             "app-123",
		Name:           "Test App",
		Namespace:      "test_app",
		AISearchTables: []discovery.AISearchTable{},
		DiscoveredElements: []*discovery.Element{
			{
				ID:        "elem-resource",
				Name:      "Resource Element",
				Namespace: "resource_element",
				AISearchTables: []discovery.AISearchTable{
					{
						ID:                      "st-1",
						ObjectID:                "elem-resource",
						FieldID:                 "field-1",
						FieldName:               "Field 1",
						AIProviderConnectorID:   "connector-1",
						AIProviderConnectorName: "model",
						Duration:                "DAYS",
						TargetLag:               1,
					},
				},
			},
			{
				ID:        "elem-datasource",
				Name:      "Data Source Element",
				Namespace: "datasource_element",
				AISearchTables: []discovery.AISearchTable{
					{
						ID:                      "st-2",
						ObjectID:                "elem-datasource",
						FieldID:                 "field-2",
						FieldName:               "Field 2",
						AIProviderConnectorID:   "connector-1",
						AIProviderConnectorName: "model",
						Duration:                "HOURS",
						TargetLag:               6,
					},
				},
			},
		},
	}

	imports := []ImportBlock{
		{ID: "app-123", ResourceType: "elementum_app", ResourceName: "test_app"},
		{ID: "elem-resource:st-1", ResourceType: "elementum_ai_search_table", ResourceName: "resource_element_field_1"},
		{ID: "elem-datasource:st-2", ResourceType: "elementum_ai_search_table", ResourceName: "datasource_element_field_2"},
	}

	// uuidMap with MIXED references - one resource, one data source
	uuidMap := map[string]string{
		"elem-resource":   "elementum_element.resource_element.id",
		"elem-datasource": "data.elementum_element.datasource_element.id",
	}

	gen := NewAISearchTableHCLGenerator(app, imports, uuidMap)
	hcl := gen.GenerateAll()

	// First element should use RESOURCE reference
	if !strings.Contains(hcl, `object_id = elementum_element.resource_element.id`) {
		t.Errorf("First element should use resource reference, got:\n%s", hcl)
	}

	// Second element should use DATA SOURCE reference
	if !strings.Contains(hcl, `object_id = data.elementum_element.datasource_element.id`) {
		t.Errorf("Second element should use data source reference, got:\n%s", hcl)
	}
}

// TestAISearchTableHCLGenerator_ElementFallbackToResource tests that when an element ID
// is not in uuidMap, it falls back to constructing a resource reference (current behavior).
func TestAISearchTableHCLGenerator_ElementFallbackToResource(t *testing.T) {
	app := &discovery.App{
		ID:             "app-123",
		Name:           "Test App",
		Namespace:      "test_app",
		AISearchTables: []discovery.AISearchTable{},
		DiscoveredElements: []*discovery.Element{
			{
				ID:        "elem-unknown",
				Name:      "Unknown Element",
				Namespace: "unknown_element",
				AISearchTables: []discovery.AISearchTable{
					{
						ID:                      "st-1",
						ObjectID:                "elem-unknown",
						FieldID:                 "field-1",
						FieldName:               "Field 1",
						AIProviderConnectorID:   "connector-1",
						AIProviderConnectorName: "model",
						Duration:                "DAYS",
						TargetLag:               1,
					},
				},
			},
		},
	}

	imports := []ImportBlock{
		{ID: "app-123", ResourceType: "elementum_app", ResourceName: "test_app"},
		{ID: "elem-unknown:st-1", ResourceType: "elementum_ai_search_table", ResourceName: "unknown_element_field_1"},
	}

	// Empty uuidMap - element ID is NOT in map
	uuidMap := map[string]string{}

	gen := NewAISearchTableHCLGenerator(app, imports, uuidMap)
	hcl := gen.GenerateAll()

	// Should fall back to RESOURCE reference constructed from element name
	if !strings.Contains(hcl, `object_id = elementum_element.unknown_element.id`) {
		t.Errorf("Should fall back to resource reference when element not in uuidMap, got:\n%s", hcl)
	}
}
