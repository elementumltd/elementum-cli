// Copyright (c) ElementumAI, Inc.
// SPDX-License-Identifier: MPL-2.0

package export

import (
	"strings"
	"testing"

	"github.com/elementumltd/elementum-cli/discovery"
)

func TestStripNullAttributes(t *testing.T) {
	input := `resource "elementum_boolean_field" "test" {
  column_name       = null
  default_value     = false
  description       = null
  field_role        = null
  lock_on_create    = false
  name              = "Test Field"
  object_id         = "c5f48604-3870-4e46-a10b-f5d41837cbbb"
  required          = false
  required_on_close = false
  show_on_create    = false
}`

	expected := `resource "elementum_boolean_field" "test" {
  default_value     = false
  lock_on_create    = false
  name              = "Test Field"
  object_id         = "c5f48604-3870-4e46-a10b-f5d41837cbbb"
  required          = false
  required_on_close = false
  show_on_create    = false
}`

	result := stripNullAttributes(input)
	if result != expected {
		t.Errorf("stripNullAttributes() failed\nGot:\n%s\n\nExpected:\n%s", result, expected)
	}
}

func TestBuildUUIDMap(t *testing.T) {
	imports := []ImportBlock{
		{
			ID:           "app-uuid-123",
			ResourceType: "elementum_app",
			ResourceName: "my_app",
		},
		{
			ID:           "app-uuid-123:field-uuid-456",
			ResourceType: "elementum_boolean_field",
			ResourceName: "my_field",
		},
		{
			ID:           "app-uuid-123:agent-uuid-789:tool-uuid-abc",
			ResourceType: "elementum_agent_create_record_tool",
			ResourceName: "my_tool",
		},
	}

	uuidMap := buildUUIDMap(imports, nil)

	// Check app ID mapping
	if ref, ok := uuidMap["app-uuid-123"]; !ok || ref != "elementum_app.my_app.id" {
		t.Errorf("App UUID not mapped correctly, got: %s", ref)
	}

	// Check field ID mapping
	if ref, ok := uuidMap["field-uuid-456"]; !ok || ref != "elementum_boolean_field.my_field.id" {
		t.Errorf("Field UUID not mapped correctly, got: %s", ref)
	}

	// Check tool ID mapping
	if ref, ok := uuidMap["tool-uuid-abc"]; !ok || ref != "elementum_agent_create_record_tool.my_tool.id" {
		t.Errorf("Tool UUID not mapped correctly, got: %s", ref)
	}
}

func TestBuildUUIDMap_AISearchTablesFromDiscoveredElements(t *testing.T) {
	// Use real UUID format for proper testing
	app := &discovery.App{
		ID:             "11111111-1111-1111-1111-111111111111",
		Name:           "Test App",
		Namespace:      "test_app",
		AISearchTables: []discovery.AISearchTable{}, // No app-level search tables
		DiscoveredElements: []*discovery.Element{
			{
				ID:        "22222222-2222-2222-2222-222222222222",
				Name:      "Categories",
				Namespace: "categories",
				AISearchTables: []discovery.AISearchTable{
					{
						ID:        "33333333-3333-3333-3333-333333333333",
						ObjectID:  "22222222-2222-2222-2222-222222222222",
						FieldID:   "44444444-4444-4444-4444-444444444444",
						FieldName: "Description",
					},
					{
						ID:        "55555555-5555-5555-5555-555555555555",
						ObjectID:  "22222222-2222-2222-2222-222222222222",
						FieldID:   "66666666-6666-6666-6666-666666666666",
						FieldName: "Notes",
					},
				},
			},
		},
	}

	imports := []ImportBlock{
		{ID: "11111111-1111-1111-1111-111111111111", ResourceType: "elementum_app", ResourceName: "test_app"},
		{ID: "22222222-2222-2222-2222-222222222222", ResourceType: "elementum_element", ResourceName: "categories"},
		{ID: "22222222-2222-2222-2222-222222222222/33333333-3333-3333-3333-333333333333", ResourceType: "elementum_ai_search_table", ResourceName: "categories_description"},
		{ID: "22222222-2222-2222-2222-222222222222/55555555-5555-5555-5555-555555555555", ResourceType: "elementum_ai_search_table", ResourceName: "categories_notes"},
	}

	uuidMap := buildUUIDMap(imports, app)

	// Check that AI search table IDs are mapped
	if ref, ok := uuidMap["33333333-3333-3333-3333-333333333333"]; !ok || ref != "elementum_ai_search_table.categories_description.id" {
		t.Errorf("AISearchTable st-1 not mapped correctly, got: %s", ref)
	}
	if ref, ok := uuidMap["55555555-5555-5555-5555-555555555555"]; !ok || ref != "elementum_ai_search_table.categories_notes.id" {
		t.Errorf("AISearchTable st-2 not mapped correctly, got: %s", ref)
	}
}

func TestBuildUUIDMap_AISearchTablesFromAppAndElements(t *testing.T) {
	// Use real UUID format for proper testing
	app := &discovery.App{
		ID:        "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
		Name:      "Test App",
		Namespace: "test_app",
		AISearchTables: []discovery.AISearchTable{
			{
				ID:        "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb",
				ObjectID:  "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
				FieldID:   "cccccccc-cccc-cccc-cccc-cccccccccccc",
				FieldName: "App Description",
			},
		},
		DiscoveredElements: []*discovery.Element{
			{
				ID:        "dddddddd-dddd-dddd-dddd-dddddddddddd",
				Name:      "Child Element",
				Namespace: "child_element",
				AISearchTables: []discovery.AISearchTable{
					{
						ID:        "eeeeeeee-eeee-eeee-eeee-eeeeeeeeeeee",
						ObjectID:  "dddddddd-dddd-dddd-dddd-dddddddddddd",
						FieldID:   "ffffffff-ffff-ffff-ffff-ffffffffffff",
						FieldName: "Element Notes",
					},
				},
			},
		},
	}

	imports := []ImportBlock{
		{ID: "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa", ResourceType: "elementum_app", ResourceName: "test_app"},
		{ID: "dddddddd-dddd-dddd-dddd-dddddddddddd", ResourceType: "elementum_element", ResourceName: "child_element"},
		{ID: "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa/bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb", ResourceType: "elementum_ai_search_table", ResourceName: "app_description"},
		{ID: "dddddddd-dddd-dddd-dddd-dddddddddddd/eeeeeeee-eeee-eeee-eeee-eeeeeeeeeeee", ResourceType: "elementum_ai_search_table", ResourceName: "element_notes"},
	}

	uuidMap := buildUUIDMap(imports, app)

	// Check app-level AI search table
	if ref, ok := uuidMap["bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"]; !ok || ref != "elementum_ai_search_table.app_description.id" {
		t.Errorf("App AISearchTable not mapped correctly, got: %s", ref)
	}

	// Check element-level AI search table
	if ref, ok := uuidMap["eeeeeeee-eeee-eeee-eeee-eeeeeeeeeeee"]; !ok || ref != "elementum_ai_search_table.element_notes.id" {
		t.Errorf("Element AISearchTable not mapped correctly, got: %s", ref)
	}
}

func TestBeautify_SearchTableIDReplacement(t *testing.T) {
	// Test that search_table_id UUIDs get replaced with Terraform references
	// Use real UUID format (8-4-4-4-12 hex digits)
	input := `resource "elementum_agent_ai_search_tool" "search_categories" {
  agent_id        = elementum_agent.my_agent.id
  name            = "Search Categories"
  target_id       = elementum_element.categories.id
  search_table_id = "a95c8f2d-7bc7-4439-b2fb-b4965a3714ce"
}`

	uuidMap := map[string]string{
		"a95c8f2d-7bc7-4439-b2fb-b4965a3714ce": "elementum_ai_search_table.categories_description.id",
	}

	result := replaceUUIDs(input, uuidMap)

	// Verify the search_table_id was replaced
	if strings.Contains(result, `"a95c8f2d-7bc7-4439-b2fb-b4965a3714ce"`) {
		t.Errorf("search_table_id UUID was not replaced")
	}
	if !strings.Contains(result, "elementum_ai_search_table.categories_description.id") {
		t.Errorf("Expected search_table_id to reference elementum_ai_search_table.categories_description.id")
	}
}

func TestReplaceUUIDs(t *testing.T) {
	input := `object_id = "c5f48604-3870-4e46-a10b-f5d41837cbbb"`

	uuidMap := map[string]string{
		"c5f48604-3870-4e46-a10b-f5d41837cbbb": "elementum_app.my_app.id",
	}

	result := replaceUUIDs(input, uuidMap)
	expected := `object_id = elementum_app.my_app.id`

	if result != expected {
		t.Errorf("replaceUUIDs() failed\nGot: %s\nExpected: %s", result, expected)
	}
}

func TestGenerateStageLocals(t *testing.T) {
	app := &discovery.App{
		Layouts: []discovery.Layout{
			{ID: "stage-1", Name: "Open"},
			{ID: "stage-2", Name: "In Progress"},
		},
	}

	result := GenerateStageLocals(app, "my_app")

	if !strings.Contains(result, "locals {") {
		t.Errorf("Locals block not generated")
	}

	if !strings.Contains(result, "stage_ids_by_key") {
		t.Errorf("stage_ids_by_key not generated")
	}

	if !strings.Contains(result, "elementum_app.my_app.stage_ids_by_key") {
		t.Errorf("App stage_ids_by_key reference not included in locals")
	}

	// Check for core field IDs
	if !strings.Contains(result, "title_field_id") {
		t.Errorf("title_field_id not generated")
	}
	if !strings.Contains(result, "status_field_id") {
		t.Errorf("status_field_id not generated")
	}
	if !strings.Contains(result, "id_field_id") {
		t.Errorf("id_field_id not generated")
	}

	// Check for audit trail field IDs
	if !strings.Contains(result, "created_by_field_id") {
		t.Errorf("created_by_field_id not generated")
	}
	if !strings.Contains(result, "created_at_field_id") {
		t.Errorf("created_at_field_id not generated")
	}
	if !strings.Contains(result, "updated_by_field_id") {
		t.Errorf("updated_by_field_id not generated")
	}
	if !strings.Contains(result, "updated_at_field_id") {
		t.Errorf("updated_at_field_id not generated")
	}
	if !strings.Contains(result, "closed_by_field_id") {
		t.Errorf("closed_by_field_id not generated")
	}
	if !strings.Contains(result, "closed_at_field_id") {
		t.Errorf("closed_at_field_id not generated")
	}
}

// Test automation UUID mapping
func TestBuildUUIDMap_WithAutomations(t *testing.T) {
	imports := []ImportBlock{
		{
			ID:           "app-uuid-123",
			ResourceType: "elementum_app",
			ResourceName: "my_app",
		},
		{
			ID:           "auto-uuid-456",
			ResourceType: "elementum_automation",
			ResourceName: "my_automation",
		},
		{
			ID:           "auto-uuid-456:trigger-uuid-789",
			ResourceType: "elementum_record_created_trigger",
			ResourceName: "on_create",
		},
		{
			ID:           "workflow-uuid-abc:task-uuid-def",
			ResourceType: "elementum_message_task",
			ResourceName: "send_notification",
		},
	}

	app := &discovery.App{
		ID:   "app-uuid-123",
		Name: "My App",
		Automations: []discovery.Automation{
			{
				ID:         "auto-uuid-456",
				Name:       "My Automation",
				WorkflowID: "workflow-uuid-abc",
				Triggers: []discovery.Trigger{
					{ID: "trigger-uuid-789", Type: "record_created"},
				},
				Tasks: []discovery.Task{
					{ID: "task-uuid-def", Type: "message", WorkflowID: "workflow-uuid-abc"},
				},
			},
		},
	}

	uuidMap := buildUUIDMap(imports, app)

	tests := []struct {
		uuid     string
		expected string
		desc     string
	}{
		{"app-uuid-123", "elementum_app.my_app.id", "app ID"},
		{"auto-uuid-456", "elementum_automation.my_automation.id", "automation ID"},
		{"workflow-uuid-abc", "elementum_automation.my_automation.workflow_id", "workflow ID"},
		{"trigger-uuid-789", "elementum_record_created_trigger.on_create.id", "trigger ID"},
		{"task-uuid-def", "elementum_message_task.send_notification.id", "task ID"},
	}

	for _, tt := range tests {
		if got, ok := uuidMap[tt.uuid]; !ok {
			t.Errorf("%s (%s) not found in UUID map", tt.desc, tt.uuid)
		} else if got != tt.expected {
			t.Errorf("%s mapping incorrect: got %q, want %q", tt.desc, got, tt.expected)
		}
	}
}

// Test trigger computed references
func TestBuildUUIDMap_RecordBasedTriggerRefs(t *testing.T) {
	imports := []ImportBlock{
		{
			ID:           "app-123",
			ResourceType: "elementum_app",
			ResourceName: "my_app",
		},
		{
			ID:           "auto-123",
			ResourceType: "elementum_automation",
			ResourceName: "my_automation",
		},
		{
			ID:           "auto-123:trigger-456",
			ResourceType: "elementum_record_created_trigger",
			ResourceName: "on_create",
		},
	}

	app := &discovery.App{
		ID: "app-123",
		Automations: []discovery.Automation{
			{
				ID:         "auto-123",
				WorkflowID: "workflow-456",
				Triggers: []discovery.Trigger{
					{ID: "trigger-456", Type: "record_created"},
				},
			},
		},
	}

	uuidMap := buildUUIDMap(imports, app)

	// Check computed trigger references
	expectedRefs := []struct {
		key string
		val string
	}{
		{"trigger.record.10000001-2000-4000-a000-800000000000", "elementum_record_created_trigger.on_create.record_id"},
		{"trigger.record.10000001-2000-4000-a000-800000000005", "elementum_record_created_trigger.on_create.record_url"},
		{"trigger.record.10000001-2000-4000-a000-800000000003", "elementum_record_created_trigger.on_create.attachments"},
	}

	for _, ref := range expectedRefs {
		if got, ok := uuidMap[ref.key]; !ok {
			t.Errorf("Computed trigger reference %q not found in UUID map", ref.key)
		} else if got != ref.val {
			t.Errorf("Computed trigger reference %q incorrect: got %q, want %q", ref.key, got, ref.val)
		}
	}
}

// Test record_search task computed references
func TestBuildUUIDMap_RecordSearchTaskRefs(t *testing.T) {
	imports := []ImportBlock{
		{
			ID:           "app-123",
			ResourceType: "elementum_app",
			ResourceName: "my_app",
		},
		{
			ID:           "auto-123",
			ResourceType: "elementum_automation",
			ResourceName: "my_automation",
		},
		{
			ID:           "workflow-123:task-456",
			ResourceType: "elementum_record_search_task",
			ResourceName: "find_records",
		},
	}

	app := &discovery.App{
		ID: "app-123",
		Automations: []discovery.Automation{
			{
				ID:         "auto-123",
				WorkflowID: "workflow-123",
				Tasks: []discovery.Task{
					{ID: "task-456", Type: "record_search", WorkflowID: "workflow-123"},
				},
			},
		},
	}

	uuidMap := buildUUIDMap(imports, app)

	// Check computed task references
	expectedRefs := []struct {
		key string
		val string
	}{
		{"task-456", "elementum_record_search_task.find_records.id"},
		{"task-456.first_record_id", "elementum_record_search_task.find_records.first_record_id"},
		{"task-456.record_count", "elementum_record_search_task.find_records.record_count"},
	}

	for _, ref := range expectedRefs {
		if got, ok := uuidMap[ref.key]; !ok {
			t.Errorf("Computed task reference %q not found in UUID map", ref.key)
		} else if got != ref.val {
			t.Errorf("Computed task reference %q incorrect: got %q, want %q", ref.key, got, ref.val)
		}
	}
}

// Test findResourceName helper
func TestFindResourceName(t *testing.T) {
	imports := []ImportBlock{
		{
			ID:           "app-123",
			ResourceType: "elementum_app",
			ResourceName: "my_app",
		},
		{
			ID:           "app-123:field-456",
			ResourceType: "elementum_text_field",
			ResourceName: "my_field",
		},
		{
			ID:           "auto-789:trigger-abc",
			ResourceType: "elementum_webhook_trigger",
			ResourceName: "webhook_handler",
		},
	}

	tests := []struct {
		resourceType string
		id           string
		want         string
	}{
		{"elementum_app", "app-123", "my_app"},
		{"elementum_text_field", "field-456", "my_field"},
		{"elementum_webhook_trigger", "trigger-abc", "webhook_handler"},
		{"elementum_unknown", "unknown-id", ""}, // Not found
	}

	for _, tt := range tests {
		got := findResourceName(imports, tt.resourceType, tt.id)
		if got != tt.want {
			t.Errorf("findResourceName(%q, %q) = %q, want %q", tt.resourceType, tt.id, got, tt.want)
		}
	}
}

// Test mergeMaps helper
func TestMergeMaps(t *testing.T) {
	dst := map[string]string{
		"key1": "value1",
		"key2": "value2",
	}

	src := map[string]string{
		"key2": "updated_value2", // Should overwrite
		"key3": "value3",         // Should add
	}

	mergeMaps(dst, src)

	expectedSize := 3
	if len(dst) != expectedSize {
		t.Errorf("Merged map size = %d, want %d", len(dst), expectedSize)
	}

	if dst["key1"] != "value1" {
		t.Errorf("dst[key1] = %q, want %q", dst["key1"], "value1")
	}

	if dst["key2"] != "updated_value2" {
		t.Errorf("dst[key2] = %q, want %q (should be overwritten)", dst["key2"], "updated_value2")
	}

	if dst["key3"] != "value3" {
		t.Errorf("dst[key3] = %q, want %q", dst["key3"], "value3")
	}
}

// Test BeautifyTableConfig
func TestBeautifyTableConfig(t *testing.T) {
	input := `resource "elementum_table" "sales_summary" {
  category_id = "11111111-1111-1111-1111-111111111111"
  source_id   = "22222222-2222-2222-2222-222222222222"
  name        = "Sales Summary"
  handle      = "sales_summary"
  description = null
  color       = null
  cloud_mapping_cloudlink_id = "33333333-3333-3333-3333-333333333333"
}`

	imports := []ImportBlock{
		{
			ID:           "44444444-4444-4444-4444-444444444444",
			ResourceType: "elementum_table",
			ResourceName: "sales_summary",
		},
	}

	table := &discovery.Table{
		ID:          "44444444-4444-4444-4444-444444444444",
		Name:        "Sales Summary",
		Handle:      "sales_summary",
		CategoryID:  "11111111-1111-1111-1111-111111111111",
		SourceID:    "22222222-2222-2222-2222-222222222222",
		CloudLinkID: "33333333-3333-3333-3333-333333333333",
	}

	relatedResources := []discovery.RelatedResource{
		{
			ID:           "11111111-1111-1111-1111-111111111111",
			Name:         "Analytics",
			ResourceType: "data.elementum_category",
		},
		{
			ID:           "22222222-2222-2222-2222-222222222222",
			Name:         "Sales App",
			ResourceType: "elementum_app",
		},
		{
			ID:           "33333333-3333-3333-3333-333333333333",
			Name:         "Snowflake Production",
			ResourceType: "elementum_cloudlink",
		},
	}

	result := BeautifyTableConfig(input, imports, table, relatedResources)

	// Check nulls are stripped
	if strings.Contains(result, "= null") {
		t.Errorf("Null attributes not stripped")
	}

	// Check category_id is replaced with data source reference
	if strings.Contains(result, "11111111-1111-1111-1111-111111111111") {
		t.Errorf("Category UUID not replaced with reference")
	}
	if !strings.Contains(result, "data.elementum_category.analytics.id") {
		t.Errorf("Category data source reference not inserted")
	}

	// Check source_id is replaced with app reference
	if strings.Contains(result, "22222222-2222-2222-2222-222222222222") {
		t.Errorf("Source UUID not replaced with reference")
	}
	if !strings.Contains(result, "elementum_app.sales_app.id") {
		t.Errorf("App reference not inserted")
	}

	// Check cloudlink_id is replaced with cloudlink data source reference
	if strings.Contains(result, "33333333-3333-3333-3333-333333333333") {
		t.Errorf("CloudLink UUID not replaced with reference")
	}
	if !strings.Contains(result, "data.elementum_cloudlink.snowflake_production.id") {
		t.Errorf("CloudLink data source reference not inserted")
	}
}

// Test buildTableUUIDMap
func TestBuildTableUUIDMap(t *testing.T) {
	imports := []ImportBlock{
		{
			ID:           "table-uuid-123",
			ResourceType: "elementum_table",
			ResourceName: "my_table",
		},
	}

	table := &discovery.Table{
		ID:     "table-uuid-123",
		Name:   "My Table",
		Handle: "my_table",
	}

	relatedResources := []discovery.RelatedResource{
		{
			ID:           "cloudlink-uuid-456",
			Name:         "Snowflake Link",
			ResourceType: "elementum_cloudlink",
		},
		{
			ID:           "app-uuid-789",
			Name:         "Sales App",
			ResourceType: "elementum_app",
		},
		{
			ID:           "category-uuid-abc",
			Name:         "Analytics",
			ResourceType: "data.elementum_category",
		},
	}

	uuidMap := buildTableUUIDMap(imports, table, relatedResources)

	// Check table ID mapping
	if ref, ok := uuidMap["table-uuid-123"]; !ok || ref != "elementum_table.my_table.id" {
		t.Errorf("Table UUID not mapped correctly, got: %s", ref)
	}

	// Check cloudlink mapping (should be data source reference)
	if ref, ok := uuidMap["cloudlink-uuid-456"]; !ok || ref != "data.elementum_cloudlink.snowflake_link.id" {
		t.Errorf("CloudLink UUID not mapped correctly, got: %s", ref)
	}

	// Check app mapping
	if ref, ok := uuidMap["app-uuid-789"]; !ok || ref != "elementum_app.sales_app.id" {
		t.Errorf("App UUID not mapped correctly, got: %s", ref)
	}

	// Check category mapping (data source)
	if ref, ok := uuidMap["category-uuid-abc"]; !ok || ref != "data.elementum_category.analytics.id" {
		t.Errorf("Category UUID not mapped correctly, got: %s", ref)
	}
}

// Test BeautifyTableConfig with nil related resources
func TestBeautifyTableConfig_NilRelatedResources(t *testing.T) {
	input := `resource "elementum_table" "test" {
  name        = "Test"
  description = null
}`

	imports := []ImportBlock{
		{
			ID:           "table-123",
			ResourceType: "elementum_table",
			ResourceName: "test",
		},
	}

	table := &discovery.Table{
		ID:   "table-123",
		Name: "Test",
	}

	// Should not panic with nil related resources
	result := BeautifyTableConfig(input, imports, table, nil)

	// Check nulls are stripped
	if strings.Contains(result, "= null") {
		t.Errorf("Null attributes not stripped")
	}
}

// Test BeautifyTableConfig with table referencing another table as source
func TestBeautifyTableConfig_TableSource(t *testing.T) {
	input := `resource "elementum_table" "derived_table" {
  source_id = "11111111-1111-1111-1111-111111111111"
  name      = "Derived Table"
}`

	imports := []ImportBlock{
		{
			ID:           "22222222-2222-2222-2222-222222222222",
			ResourceType: "elementum_table",
			ResourceName: "derived_table",
		},
	}

	table := &discovery.Table{
		ID:       "22222222-2222-2222-2222-222222222222",
		Name:     "Derived Table",
		SourceID: "11111111-1111-1111-1111-111111111111",
	}

	relatedResources := []discovery.RelatedResource{
		{
			ID:           "11111111-1111-1111-1111-111111111111",
			Name:         "Base Table",
			ResourceType: "elementum_table",
		},
	}

	result := BeautifyTableConfig(input, imports, table, relatedResources)

	// Check source_id is replaced with table reference
	if strings.Contains(result, "11111111-1111-1111-1111-111111111111") {
		t.Errorf("Source table UUID not replaced with reference")
	}
	if !strings.Contains(result, "elementum_table.base_table.id") {
		t.Errorf("Source table reference not inserted")
	}
}

// Test modular UUID mapping with all resource types
func TestBuildUUIDMap_ModularApproach(t *testing.T) {
	imports := []ImportBlock{
		{ID: "app-1", ResourceType: "elementum_app", ResourceName: "my_app"},
		{ID: "app-1:field-1", ResourceType: "elementum_text_field", ResourceName: "name_field"},
		{ID: "app-1:layout-1", ResourceType: "elementum_layout", ResourceName: "active_stage"},
		{ID: "app-1:flow-1", ResourceType: "elementum_flow", ResourceName: "main_flow"},
		{ID: "app-1:agent-1", ResourceType: "elementum_agent", ResourceName: "support_agent"},
		{ID: "app-1:agent-1:tool-1", ResourceType: "elementum_agent_search_records_tool", ResourceName: "search_tool"},
		{ID: "app-1:widget-1", ResourceType: "elementum_widget", ResourceName: "dashboard"},
		{ID: "app-1:reader-1", ResourceType: "elementum_ai_file_reader", ResourceName: "pdf_reader"},
		{ID: "app-1:approval-1", ResourceType: "elementum_approval_process", ResourceName: "manager_approval"},
		{ID: "auto-1", ResourceType: "elementum_automation", ResourceName: "my_automation"},
	}

	app := &discovery.App{
		ID: "app-1",
		Fields: []discovery.Field{
			{ID: "field-1", Type: "text"},
		},
		Layouts: []discovery.Layout{
			{ID: "layout-1", Name: "Active"},
		},
		Flows: []discovery.Flow{
			{ID: "flow-1", Name: "Main Flow"},
		},
		Agents: []discovery.Agent{
			{ID: "agent-1", Name: "Support Agent", Tools: []discovery.AgentTool{
				{ID: "tool-1", Name: "Search Tool", Type: "AgentSearchAspectTool"},
			}},
		},
		Widgets: []discovery.Widget{
			{ID: "widget-1", Name: "Dashboard"},
		},
		AIFileReaders: []discovery.AIFileReader{
			{ID: "reader-1", Name: "PDF Reader"},
		},
		Approvals: []discovery.ApprovalProcess{
			{ID: "approval-1", Name: "Manager Approval"},
		},
		Automations: []discovery.Automation{
			{ID: "auto-1", Name: "My Automation", WorkflowID: "workflow-1"},
		},
	}

	uuidMap := buildUUIDMap(imports, app)

	// Verify all resource types are mapped
	expectedMappings := map[string]string{
		"app-1":      "elementum_app.my_app.id",
		"field-1":    "elementum_text_field.name_field.id",
		"flow-1":     "elementum_flow.main_flow.id",
		"agent-1":    "elementum_agent.support_agent.id",
		"tool-1":     "elementum_agent_search_records_tool.search_tool.id",
		"widget-1":   "elementum_widget.dashboard.id",
		"reader-1":   "elementum_ai_file_reader.pdf_reader.id",
		"approval-1": "elementum_approval_process.manager_approval.id",
		"auto-1":     "elementum_automation.my_automation.id",
		"workflow-1": "elementum_automation.my_automation.workflow_id",
	}

	for uuid, expectedRef := range expectedMappings {
		if got, ok := uuidMap[uuid]; !ok {
			t.Errorf("UUID %q not found in map", uuid)
		} else if got != expectedRef {
			t.Errorf("UUID %q mapped to %q, want %q", uuid, got, expectedRef)
		}
	}

	// Verify minimum number of mappings
	minExpected := len(expectedMappings)
	if len(uuidMap) < minExpected {
		t.Errorf("Expected at least %d mappings, got %d", minExpected, len(uuidMap))
	}
}

// Test relationship UUID mapping
func TestBuildUUIDMap_WithRelationships(t *testing.T) {
	imports := []ImportBlock{
		{
			ID:           "app-orders",
			ResourceType: "elementum_app",
			ResourceName: "orders",
		},
		{
			ID:           "app-orders:rel-123",
			ResourceType: "elementum_relationship",
			ResourceName: "to_customers",
		},
	}

	app := &discovery.App{
		ID:   "app-orders",
		Name: "Orders",
		Relationships: []discovery.Relationship{
			{
				ID:                "rel-123",
				RelatedObjectID:   "app-customers",
				RelatedObjectType: "App",
				RelatedObjectName: "Customers",
			},
		},
		RelatedObjects: []discovery.RelatedObject{
			{
				ID:        "app-customers",
				Name:      "Customers",
				Type:      "App",
				Namespace: "customers",
				Fields: []discovery.Field{
					{ID: "field-cust-title", Name: "Title", Type: "text"},
					{ID: "field-cust-status", Name: "Status", Type: "dropdown"},
				},
			},
		},
	}

	uuidMap := buildUUIDMap(imports, app)

	// Check relationship ID mapping
	if got, ok := uuidMap["rel-123"]; !ok {
		t.Error("Relationship ID should be mapped")
	} else if got != "elementum_relationship.to_customers.id" {
		t.Errorf("Relationship ID mapping = %q, want %q", got, "elementum_relationship.to_customers.id")
	}

	// Check related object ID mapping
	if got, ok := uuidMap["app-customers"]; !ok {
		t.Error("Related object ID should be mapped")
	} else if got != "data.elementum_app.customers.id" {
		t.Errorf("Related object ID mapping = %q, want %q", got, "data.elementum_app.customers.id")
	}

	// Note: System field IDs (title_field_id, status_field_id) from related objects
	// are NOT automatically mapped because the data sources don't expose these attributes.
	// Users need to use field data sources for related object fields.
}

// Test related object element type
func TestBuildUUIDMap_WithRelatedElement(t *testing.T) {
	imports := []ImportBlock{
		{
			ID:           "app-orders",
			ResourceType: "elementum_app",
			ResourceName: "orders",
		},
	}

	app := &discovery.App{
		ID:   "app-orders",
		Name: "Orders",
		RelatedObjects: []discovery.RelatedObject{
			{
				ID:        "elem-products",
				Name:      "Products",
				Type:      "Element",
				Namespace: "products",
				Fields: []discovery.Field{
					{ID: "field-prod-title", Name: "Title", Type: "text"},
				},
			},
		},
	}

	uuidMap := buildUUIDMap(imports, app)

	// Check element data source reference
	if got, ok := uuidMap["elem-products"]; !ok {
		t.Error("Related element ID should be mapped")
	} else if got != "data.elementum_element.products.id" {
		t.Errorf("Related element ID mapping = %q, want %q", got, "data.elementum_element.products.id")
	}

	// Note: System field IDs (title_field_id, etc.) from related elements
	// are NOT automatically mapped because the data sources don't expose these attributes.
	// Users need to use field data sources for related object fields.
}

// Test multiple related objects with same name
func TestBuildUUIDMap_DuplicateRelatedObjectNames(t *testing.T) {
	imports := []ImportBlock{
		{
			ID:           "app-orders",
			ResourceType: "elementum_app",
			ResourceName: "orders",
		},
	}

	app := &discovery.App{
		ID:   "app-orders",
		Name: "Orders",
		RelatedObjects: []discovery.RelatedObject{
			{
				ID:        "app-cust-1",
				Name:      "Customers",
				Type:      "App",
				Namespace: "customers_us",
			},
			{
				ID:        "app-cust-2",
				Name:      "Customers",
				Type:      "App",
				Namespace: "customers_eu",
			},
		},
	}

	uuidMap := buildUUIDMap(imports, app)

	// Check first customers reference
	if got, ok := uuidMap["app-cust-1"]; !ok {
		t.Error("First customers app ID should be mapped")
	} else if got != "data.elementum_app.customers.id" {
		t.Errorf("First customers app ID mapping = %q, want %q", got, "data.elementum_app.customers.id")
	}

	// Check second customers reference (should have _1 suffix)
	if got, ok := uuidMap["app-cust-2"]; !ok {
		t.Error("Second customers app ID should be mapped")
	} else if got != "data.elementum_app.customers_1.id" {
		t.Errorf("Second customers app ID mapping = %q, want %q", got, "data.elementum_app.customers_1.id")
	}
}

// Test that buildUUIDMap skips discovered items in RelatedObjects
// This ensures consistency with GenerateRelationshipDataSources which also skips them
func TestBuildUUIDMap_SkipsDiscoveredItemsInRelatedObjects(t *testing.T) {
	imports := []ImportBlock{
		{
			ID:           "app-orders",
			ResourceType: "elementum_app",
			ResourceName: "orders",
		},
		// Import block for discovered element (exported as resource)
		{
			ID:           "elem-products",
			ResourceType: "elementum_element",
			ResourceName: "products",
		},
	}

	app := &discovery.App{
		ID:   "app-orders",
		Name: "Orders",
		RelatedObjects: []discovery.RelatedObject{
			// This element is discovered (exported as resource) - should be skipped
			{
				ID:        "elem-products",
				Name:      "Products",
				Type:      "Element",
				Namespace: "products",
			},
			// This element is NOT discovered - should get data source reference
			{
				ID:        "elem-inventory",
				Name:      "Inventory",
				Type:      "Element",
				Namespace: "inventory",
			},
		},
		// Products is being exported as a resource
		DiscoveredElements: []*discovery.Element{
			{
				ID:        "elem-products",
				Name:      "Products",
				Namespace: "products",
			},
		},
	}

	uuidMap := buildUUIDMap(imports, app)

	// Discovered element should map to RESOURCE reference (from DiscoveredElements loop)
	if got, ok := uuidMap["elem-products"]; !ok {
		t.Error("Products element ID should be mapped")
	} else if got != "elementum_element.products.id" {
		t.Errorf("Products element mapping = %q, want %q (resource ref, not data source)", got, "elementum_element.products.id")
	}

	// Non-discovered element should map to DATA SOURCE reference
	// Should be "inventory" (no suffix) because discovered item was skipped
	if got, ok := uuidMap["elem-inventory"]; !ok {
		t.Error("Inventory element ID should be mapped")
	} else if got != "data.elementum_element.inventory.id" {
		t.Errorf("Inventory element mapping = %q, want %q", got, "data.elementum_element.inventory.id")
	}
}

// Test consistency between buildUUIDMap and GenerateRelationshipDataSources
// When discovered items appear before non-discovered items with same name,
// the non-discovered item should get the same name in both functions
func TestBuildUUIDMap_ConsistentWithGenerateRelationshipDataSources(t *testing.T) {
	imports := []ImportBlock{
		{
			ID:           "app-orders",
			ResourceType: "elementum_app",
			ResourceName: "orders",
		},
		{
			ID:           "elem-products-discovered",
			ResourceType: "elementum_element",
			ResourceName: "products",
		},
	}

	app := &discovery.App{
		ID:   "app-orders",
		Name: "Orders",
		RelatedObjects: []discovery.RelatedObject{
			// First "Products" - discovered, will be skipped by GenerateRelationshipDataSources
			{
				ID:        "elem-products-discovered",
				Name:      "Products",
				Type:      "Element",
				Namespace: "products_main",
			},
			// Second "Products" - NOT discovered, will get data source
			{
				ID:        "elem-products-related",
				Name:      "Products",
				Type:      "Element",
				Namespace: "products_other",
			},
		},
		DiscoveredElements: []*discovery.Element{
			{
				ID:        "elem-products-discovered",
				Name:      "Products",
				Namespace: "products_main",
			},
		},
	}

	// Generate data sources (what imports.go does)
	dataSourceHCL := GenerateRelationshipDataSources(app)

	// Build UUID map (what beautify.go does)
	uuidMap := buildUUIDMap(imports, app)

	// The non-discovered "Products" element should be named "products" (no suffix)
	// in GenerateRelationshipDataSources because the discovered one was skipped
	if !strings.Contains(dataSourceHCL, `data "elementum_element" "products"`) {
		t.Errorf("Expected data source named 'products', got:\n%s", dataSourceHCL)
	}

	// The same element should have matching reference in buildUUIDMap
	expectedRef := "data.elementum_element.products.id"
	if got, ok := uuidMap["elem-products-related"]; !ok {
		t.Error("Non-discovered Products element should be mapped")
	} else if got != expectedRef {
		t.Errorf("buildUUIDMap reference = %q, want %q to match data source name", got, expectedRef)
	}
}

// Test that duplicate IDs in RelatedObjects are handled consistently
func TestBuildUUIDMap_DuplicateIDsInRelatedObjects(t *testing.T) {
	imports := []ImportBlock{
		{
			ID:           "app-orders",
			ResourceType: "elementum_app",
			ResourceName: "orders",
		},
	}

	app := &discovery.App{
		ID:   "app-orders",
		Name: "Orders",
		RelatedObjects: []discovery.RelatedObject{
			// Same ID appears twice (can happen with complex relationship graphs)
			{
				ID:        "elem-products",
				Name:      "Products",
				Type:      "Element",
				Namespace: "products",
			},
			{
				ID:        "elem-products", // Duplicate ID
				Name:      "Products",
				Type:      "Element",
				Namespace: "products",
			},
			// Different element
			{
				ID:        "elem-inventory",
				Name:      "Inventory",
				Type:      "Element",
				Namespace: "inventory",
			},
		},
	}

	uuidMap := buildUUIDMap(imports, app)

	// First Products should be "products" (no suffix)
	if got, ok := uuidMap["elem-products"]; !ok {
		t.Error("Products element ID should be mapped")
	} else if got != "data.elementum_element.products.id" {
		t.Errorf("Products element mapping = %q, want %q", got, "data.elementum_element.products.id")
	}

	// Inventory should be "inventory" (no suffix, because duplicate Products didn't increment counter)
	if got, ok := uuidMap["elem-inventory"]; !ok {
		t.Error("Inventory element ID should be mapped")
	} else if got != "data.elementum_element.inventory.id" {
		t.Errorf("Inventory element mapping = %q, want %q", got, "data.elementum_element.inventory.id")
	}
}

// Test beautifyValueReferences converts trigger.record references to refs syntax
func TestBeautifyValueReferences_TriggerRecord(t *testing.T) {
	fieldUUID := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	triggerUUID := "11111111-2222-3333-4444-555555555555"

	input := `resource "elementum_update_field_task" "update_status" {
  value = "trigger.record.` + fieldUUID + `"
}`

	imports := []ImportBlock{
		{
			ID:           "automation-id:" + triggerUUID,
			ResourceType: "elementum_record_created_trigger",
			ResourceName: "on_create",
		},
	}

	app := &discovery.App{
		ID: "app-uuid",
		Automations: []discovery.Automation{
			{
				ID:         "automation-id",
				WorkflowID: "workflow-id",
				Triggers: []discovery.Trigger{
					{
						ID:   triggerUUID,
						Type: "record_created",
						Name: "record_created",
						FieldRefs: map[string]string{
							"record." + fieldUUID: "Status",
						},
					},
				},
			},
		},
	}

	uuidMap := buildUUIDMap(imports, app)
	result := beautifyValueReferences(input, uuidMap, app)

	expected := `elementum_record_created_trigger.on_create.refs["Status"]`
	if !strings.Contains(result, expected) {
		t.Errorf("Expected value to contain refs syntax.\nGot:\n%s\nExpected to contain:\n%s", result, expected)
	}
}

// Test beautifyValueReferences converts task references to refs syntax
func TestBeautifyValueReferences_TaskReference(t *testing.T) {
	taskUUID := "aaaaaaaa-1111-2222-3333-444444444444"

	input := `resource "elementum_update_field_task" "update_record" {
  record_reference = "task.` + taskUUID + `.Records[0].ID"
}`

	imports := []ImportBlock{
		{
			ID:           "workflow-id:" + taskUUID,
			ResourceType: "elementum_record_search_task",
			ResourceName: "find_record",
		},
	}

	app := &discovery.App{
		ID: "app-uuid",
		Automations: []discovery.Automation{
			{
				ID:         "automation-id",
				WorkflowID: "workflow-id",
				Tasks: []discovery.Task{
					{
						ID:         taskUUID,
						Type:       "record_search",
						Name:       "Find Record",
						WorkflowID: "workflow-id",
						FieldRefs: map[string]string{
							"Records[0].ID": "First found record id",
						},
					},
				},
			},
		},
	}

	uuidMap := buildUUIDMap(imports, app)
	result := beautifyValueReferences(input, uuidMap, app)

	expected := `elementum_record_search_task.find_record.refs["First found record id"]`
	if !strings.Contains(result, expected) {
		t.Errorf("Expected record_reference to contain refs syntax.\nGot:\n%s\nExpected to contain:\n%s", result, expected)
	}
}

// Test buildValueRefMap creates correct mappings
func TestBuildValueRefMap(t *testing.T) {
	fieldUUID := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	triggerUUID := "11111111-2222-3333-4444-555555555555"
	taskUUID := "99999999-8888-7777-6666-555555555555"

	uuidMap := map[string]string{
		triggerUUID: "elementum_record_created_trigger.my_trigger.id",
		taskUUID:    "elementum_record_search_task.my_search.id",
	}

	app := &discovery.App{
		ID: "app-uuid",
		Automations: []discovery.Automation{
			{
				ID:         "automation-id",
				WorkflowID: "workflow-id",
				Triggers: []discovery.Trigger{
					{
						ID:   triggerUUID,
						Type: "record_created",
						FieldRefs: map[string]string{
							"record." + fieldUUID:                         "Title",
							"record.10000001-2000-4000-a000-800000000000": "ID",
						},
					},
				},
				Tasks: []discovery.Task{
					{
						ID:         taskUUID,
						Type:       "record_search",
						WorkflowID: "workflow-id",
						FieldRefs: map[string]string{
							"Records Size":  "Records Size",
							"Records[0].ID": "First found record id",
						},
					},
				},
			},
		},
	}

	refMap := buildValueRefMap(app, uuidMap)

	// Check trigger refs
	triggerKey := "trigger.record." + fieldUUID
	if got, ok := refMap[triggerKey]; !ok {
		t.Errorf("Expected trigger ref mapping for %q", triggerKey)
	} else if !strings.Contains(got, "my_trigger.refs") {
		t.Errorf("Trigger ref mapping = %q, expected to contain 'my_trigger.refs'", got)
	}

	// Check task refs
	taskKey := "task." + taskUUID + ".Records[0].ID"
	if got, ok := refMap[taskKey]; !ok {
		t.Errorf("Expected task ref mapping for %q", taskKey)
	} else if !strings.Contains(got, "my_search.refs") {
		t.Errorf("Task ref mapping = %q, expected to contain 'my_search.refs'", got)
	}
}

// NOTE: Tests for addAvailableRefsComments and limitRefs were removed
// because those functions were never implemented. They were placeholder tests
// for a feature that was planned but not completed.

// Test buildUUIDMap includes role mappings
func TestBuildUUIDMap_WithRoles(t *testing.T) {
	imports := []ImportBlock{
		{
			ID:           "app-123",
			ResourceType: "elementum_app",
			ResourceName: "my_app",
		},
		{
			ID:           "app-123:role-custom-1",
			ResourceType: "elementum_role",
			ResourceName: "project_manager",
		},
	}

	app := &discovery.App{
		ID:   "app-123",
		Name: "My App",
		Roles: []discovery.Role{
			{
				ID:      "role-managed-1",
				Name:    "Admin",
				Managed: true,
			},
			{
				ID:      "role-managed-2",
				Name:    "Editor",
				Managed: true,
			},
			{
				ID:      "role-custom-1",
				Name:    "Project Manager",
				Managed: false,
				Users: []discovery.RoleMember{
					{ID: "user-1", Name: "john@example.com"},
				},
				Groups: []discovery.RoleMember{
					{ID: "group-1", Name: "Engineering"},
				},
			},
		},
	}

	uuidMap := buildUUIDMap(imports, app)

	// Check managed role mappings (data sources)
	if got, ok := uuidMap["role-managed-1"]; !ok {
		t.Error("Managed role 'Admin' should be in UUID map")
	} else if got != "data.elementum_role.admin.id" {
		t.Errorf("Managed role mapping = %q, want %q", got, "data.elementum_role.admin.id")
	}

	if got, ok := uuidMap["role-managed-2"]; !ok {
		t.Error("Managed role 'Editor' should be in UUID map")
	} else if got != "data.elementum_role.editor.id" {
		t.Errorf("Managed role mapping = %q, want %q", got, "data.elementum_role.editor.id")
	}

	// Check custom role mappings (resources)
	if got, ok := uuidMap["role-custom-1"]; !ok {
		t.Error("Custom role 'Project Manager' should be in UUID map")
	} else if got != "elementum_role.project_manager.id" {
		t.Errorf("Custom role mapping = %q, want %q", got, "elementum_role.project_manager.id")
	}

	// Check user mappings from role membership
	if got, ok := uuidMap["user-1"]; !ok {
		t.Error("User from role membership should be in UUID map")
	} else if got != "data.elementum_user.john.id" {
		t.Errorf("User mapping = %q, want %q", got, "data.elementum_user.john.id")
	}

	// Check group mappings from role membership
	if got, ok := uuidMap["group-1"]; !ok {
		t.Error("Group from role membership should be in UUID map")
	} else if got != "data.elementum_group.engineering.id" {
		t.Errorf("Group mapping = %q, want %q", got, "data.elementum_group.engineering.id")
	}
}

// Test buildUUIDMap handles duplicate role names
func TestBuildUUIDMap_RoleDuplicateNames(t *testing.T) {
	imports := []ImportBlock{
		{
			ID:           "app-123",
			ResourceType: "elementum_app",
			ResourceName: "my_app",
		},
	}

	app := &discovery.App{
		ID:   "app-123",
		Name: "My App",
		Roles: []discovery.Role{
			{ID: "role-1", Name: "Admin", Managed: true},
			{ID: "role-2", Name: "Admin", Managed: true}, // Same name
		},
	}

	uuidMap := buildUUIDMap(imports, app)

	// First Admin should map to data.elementum_role.admin.id
	if got, ok := uuidMap["role-1"]; !ok {
		t.Error("First Admin role should be in UUID map")
	} else if got != "data.elementum_role.admin.id" {
		t.Errorf("First Admin mapping = %q, want %q", got, "data.elementum_role.admin.id")
	}

	// Second Admin should map to data.elementum_role.admin_1.id
	if got, ok := uuidMap["role-2"]; !ok {
		t.Error("Second Admin role should be in UUID map")
	} else if got != "data.elementum_role.admin_1.id" {
		t.Errorf("Second Admin mapping = %q, want %q", got, "data.elementum_role.admin_1.id")
	}
}

// =============================================================================
// Table Field Beautification Tests
// =============================================================================

// TestBuildUUIDMap_TableFieldMapping tests that table field IDs are correctly
// mapped to local lookup references for datamine primary_column_ids beautification
func TestBuildUUIDMap_TableFieldMapping(t *testing.T) {
	fieldUUID1 := "field-uuid-111"
	fieldUUID2 := "field-uuid-222"
	fieldUUID3 := "field-uuid-333"
	tableUUID := "table-uuid-123"
	appUUID := "app-uuid-456"

	imports := []ImportBlock{
		{
			ID:           appUUID,
			ResourceType: "elementum_app",
			ResourceName: "my_app",
		},
		{
			ID:           tableUUID,
			ResourceType: "elementum_table",
			ResourceName: "supplier_faq_table",
		},
	}

	app := &discovery.App{
		ID:   appUUID,
		Name: "My App",
		DiscoveredTables: []*discovery.Table{
			{
				ID:     tableUUID,
				Name:   "SUPPLIER_FAQ_TABLE",
				Handle: "SUPDOC",
				Fields: []discovery.TableField{
					{ID: fieldUUID1, Name: "RELATIVE_PATH"},
					{ID: fieldUUID2, Name: "LAST_MODIFIED"},
					{ID: fieldUUID3, Name: "MD5"},
				},
			},
		},
	}

	uuidMap := buildUUIDMap(imports, app)

	// Check table ID mapping
	if ref, ok := uuidMap[tableUUID]; !ok || ref != "elementum_table.supplier_faq_table.id" {
		t.Errorf("Table UUID not mapped correctly, got: %s", ref)
	}

	// Check field ID mappings use local lookup pattern
	expectedRef1 := `local.supplier_faq_table_field_ids_by_name["RELATIVE_PATH"]`
	if ref, ok := uuidMap[fieldUUID1]; !ok || ref != expectedRef1 {
		t.Errorf("Field UUID 1 not mapped correctly, expected %s, got: %s", expectedRef1, ref)
	}

	expectedRef2 := `local.supplier_faq_table_field_ids_by_name["LAST_MODIFIED"]`
	if ref, ok := uuidMap[fieldUUID2]; !ok || ref != expectedRef2 {
		t.Errorf("Field UUID 2 not mapped correctly, expected %s, got: %s", expectedRef2, ref)
	}

	expectedRef3 := `local.supplier_faq_table_field_ids_by_name["MD5"]`
	if ref, ok := uuidMap[fieldUUID3]; !ok || ref != expectedRef3 {
		t.Errorf("Field UUID 3 not mapped correctly, expected %s, got: %s", expectedRef3, ref)
	}
}

// TestGenerateLocals_TableFieldLookups tests that the locals block includes
// table field ID lookup maps for discovered tables
func TestGenerateLocals_TableFieldLookups(t *testing.T) {
	tableUUID := "table-uuid-123"
	appUUID := "app-uuid-456"

	imports := []ImportBlock{
		{
			ID:           appUUID,
			ResourceType: "elementum_app",
			ResourceName: "my_app",
		},
		{
			ID:           tableUUID,
			ResourceType: "elementum_table",
			ResourceName: "supplier_faq_table",
		},
	}

	app := &discovery.App{
		ID:   appUUID,
		Name: "My App",
		DiscoveredTables: []*discovery.Table{
			{
				ID:     tableUUID,
				Name:   "SUPPLIER_FAQ_TABLE",
				Handle: "SUPDOC",
				Fields: []discovery.TableField{
					{ID: "field-1", Name: "RELATIVE_PATH"},
					{ID: "field-2", Name: "SIZE"},
				},
			},
		},
	}

	result := GenerateLocals(app, "my_app", imports)

	// Check that locals block is generated
	if !strings.Contains(result, "locals {") {
		t.Errorf("Locals block not generated")
	}

	// Check for table field lookup comment
	if !strings.Contains(result, "Table field ID lookups") {
		t.Errorf("Table field ID lookups comment not generated")
	}

	// Check for field_ids_by_name local variable
	if !strings.Contains(result, "supplier_faq_table_field_ids_by_name") {
		t.Errorf("supplier_faq_table_field_ids_by_name local not generated")
	}

	// Check for the for-expression pattern
	if !strings.Contains(result, "for f in elementum_table.supplier_faq_table.fields : f.name => f.id") {
		t.Errorf("Table field lookup for-expression not generated correctly")
	}
}

// TestGenerateLocals_MultipleDiscoveredTables tests that locals are generated
// for multiple discovered tables
func TestGenerateLocals_MultipleDiscoveredTables(t *testing.T) {
	imports := []ImportBlock{
		{
			ID:           "app-uuid",
			ResourceType: "elementum_app",
			ResourceName: "my_app",
		},
		{
			ID:           "table-1",
			ResourceType: "elementum_table",
			ResourceName: "products_table",
		},
		{
			ID:           "table-2",
			ResourceType: "elementum_table",
			ResourceName: "orders_table",
		},
	}

	app := &discovery.App{
		ID:   "app-uuid",
		Name: "My App",
		DiscoveredTables: []*discovery.Table{
			{
				ID:     "table-1",
				Name:   "Products",
				Fields: []discovery.TableField{{ID: "f1", Name: "SKU"}},
			},
			{
				ID:     "table-2",
				Name:   "Orders",
				Fields: []discovery.TableField{{ID: "f2", Name: "ORDER_ID"}},
			},
		},
	}

	result := GenerateLocals(app, "my_app", imports)

	// Check both table field lookups are generated
	if !strings.Contains(result, "products_table_field_ids_by_name") {
		t.Errorf("products_table_field_ids_by_name local not generated")
	}
	if !strings.Contains(result, "orders_table_field_ids_by_name") {
		t.Errorf("orders_table_field_ids_by_name local not generated")
	}
}

// TestBuildUUIDMap_TableFieldMapping_NoFields tests that tables without fields
// don't cause errors
func TestBuildUUIDMap_TableFieldMapping_NoFields(t *testing.T) {
	tableUUID := "table-uuid-123"
	appUUID := "app-uuid-456"

	imports := []ImportBlock{
		{
			ID:           appUUID,
			ResourceType: "elementum_app",
			ResourceName: "my_app",
		},
		{
			ID:           tableUUID,
			ResourceType: "elementum_table",
			ResourceName: "empty_table",
		},
	}

	app := &discovery.App{
		ID:   appUUID,
		Name: "My App",
		DiscoveredTables: []*discovery.Table{
			{
				ID:     tableUUID,
				Name:   "Empty Table",
				Handle: "EMPTY",
				Fields: []discovery.TableField{}, // No fields
			},
		},
	}

	// Should not panic
	uuidMap := buildUUIDMap(imports, app)

	// Table itself should still be mapped
	if ref, ok := uuidMap[tableUUID]; !ok || ref != "elementum_table.empty_table.id" {
		t.Errorf("Table UUID not mapped correctly, got: %s", ref)
	}
}

// TestGenerateLocals_NoDiscoveredTables tests that locals generation works
// when there are no discovered tables
func TestGenerateLocals_NoDiscoveredTables(t *testing.T) {
	imports := []ImportBlock{
		{
			ID:           "app-uuid",
			ResourceType: "elementum_app",
			ResourceName: "my_app",
		},
	}

	app := &discovery.App{
		ID:               "app-uuid",
		Name:             "My App",
		DiscoveredTables: []*discovery.Table{}, // No tables
	}

	result := GenerateLocals(app, "my_app", imports)

	// Locals block should still be generated (for app system fields)
	if !strings.Contains(result, "locals {") {
		t.Errorf("Locals block not generated")
	}

	// Should NOT contain table field lookups section
	if strings.Contains(result, "Table field ID lookups") {
		t.Errorf("Table field ID lookups should not be present when no tables")
	}
}

// =============================================================================
// Element Category/CloudLink Beautification Tests
// =============================================================================

// TestBuildUUIDMap_ElementCategoryMapping tests that discovered element category
// and cloudlink IDs are correctly mapped to data source references
func TestBuildUUIDMap_ElementCategoryMapping(t *testing.T) {
	elementCategoryUUID := "cat-uuid-element"
	elementCloudLinkUUID := "cloudlink-uuid-element"
	elementUUID := "element-uuid-123"
	appUUID := "app-uuid-456"
	appCategoryUUID := "cat-uuid-app"

	imports := []ImportBlock{
		{
			ID:           appUUID,
			ResourceType: "elementum_app",
			ResourceName: "my_app",
		},
		{
			ID:           elementUUID,
			ResourceType: "elementum_element",
			ResourceName: "my_element",
		},
	}

	app := &discovery.App{
		ID:           appUUID,
		Name:         "My App",
		CategoryID:   appCategoryUUID,
		CategoryName: "App Category",
		DiscoveredElements: []*discovery.Element{
			{
				ID:            elementUUID,
				Name:          "My Element",
				Namespace:     "my_element",
				CategoryID:    elementCategoryUUID,
				CategoryName:  "Element Category",
				CloudLinkID:   elementCloudLinkUUID,
				CloudLinkName: "Operational Platform",
			},
		},
	}

	uuidMap := buildUUIDMap(imports, app)

	// Check element category ID mapping
	expectedCatRef := "data.elementum_category.element_category.id"
	if ref, ok := uuidMap[elementCategoryUUID]; !ok || ref != expectedCatRef {
		t.Errorf("Element category UUID not mapped correctly, expected %s, got: %s", expectedCatRef, ref)
	}

	// Check element cloudlink ID mapping
	expectedCloudLinkRef := "data.elementum_cloudlink.operational_platform.id"
	if ref, ok := uuidMap[elementCloudLinkUUID]; !ok || ref != expectedCloudLinkRef {
		t.Errorf("Element cloudlink UUID not mapped correctly, expected %s, got: %s", expectedCloudLinkRef, ref)
	}

	// Check app category is also mapped (different category)
	expectedAppCatRef := "data.elementum_category.app_category.id"
	if ref, ok := uuidMap[appCategoryUUID]; !ok || ref != expectedAppCatRef {
		t.Errorf("App category UUID not mapped correctly, expected %s, got: %s", expectedAppCatRef, ref)
	}
}

// TestBuildUUIDMap_ElementCategorySharedWithApp tests that when element and app
// share the same category, the mapping still works correctly
func TestBuildUUIDMap_ElementCategorySharedWithApp(t *testing.T) {
	sharedCategoryUUID := "shared-cat-uuid"
	elementUUID := "element-uuid"
	appUUID := "app-uuid"

	imports := []ImportBlock{
		{
			ID:           appUUID,
			ResourceType: "elementum_app",
			ResourceName: "my_app",
		},
		{
			ID:           elementUUID,
			ResourceType: "elementum_element",
			ResourceName: "my_element",
		},
	}

	app := &discovery.App{
		ID:           appUUID,
		Name:         "My App",
		CategoryID:   sharedCategoryUUID,
		CategoryName: "Shared Category",
		DiscoveredElements: []*discovery.Element{
			{
				ID:           elementUUID,
				Name:         "My Element",
				Namespace:    "my_element",
				CategoryID:   sharedCategoryUUID, // Same as app
				CategoryName: "Shared Category",
			},
		},
	}

	uuidMap := buildUUIDMap(imports, app)

	// Check shared category ID is mapped (app mapping takes precedence)
	expectedRef := "data.elementum_category.shared_category.id"
	if ref, ok := uuidMap[sharedCategoryUUID]; !ok || ref != expectedRef {
		t.Errorf("Shared category UUID not mapped correctly, expected %s, got: %s", expectedRef, ref)
	}
}

// =============================================================================
// Element System Field UUID Mapping Tests
// =============================================================================

// TestBuildUUIDMap_ElementSystemFieldsToDataSources tests that discovered element
// system fields (CREATED_BY, CREATED_AT, UPDATED_BY, UPDATED_AT, etc.) are correctly
// mapped to data source references since elementum_element doesn't expose these
func TestBuildUUIDMap_ElementSystemFieldsToDataSources(t *testing.T) {
	t.Parallel()

	// UUIDs for system fields
	createdByFieldUUID := "created-by-field-uuid"
	createdAtFieldUUID := "created-at-field-uuid"
	updatedByFieldUUID := "updated-by-field-uuid"
	updatedAtFieldUUID := "updated-at-field-uuid"
	closedByFieldUUID := "closed-by-field-uuid"
	closedAtFieldUUID := "closed-at-field-uuid"
	titleFieldUUID := "title-field-uuid"
	idFieldUUID := "id-field-uuid"
	elementUUID := "element-uuid-123"
	appUUID := "app-uuid-456"

	imports := []ImportBlock{
		{
			ID:           appUUID,
			ResourceType: "elementum_app",
			ResourceName: "my_app",
		},
		{
			ID:           elementUUID,
			ResourceType: "elementum_element",
			ResourceName: "my_element",
		},
	}

	app := &discovery.App{
		ID:        appUUID,
		Name:      "My App",
		Namespace: "myapp",
		DiscoveredElements: []*discovery.Element{
			{
				ID:        elementUUID,
				Name:      "My Element",
				Namespace: "myelement",
				Fields: []discovery.Field{
					{ID: titleFieldUUID, Name: "Title", Type: "text", SemanticTags: []string{"TITLE"}},
					{ID: idFieldUUID, Name: "ID", Type: "text", SemanticTags: []string{"HANDLE"}},
					{ID: createdByFieldUUID, Name: "Created By", Type: "user", SemanticTags: []string{"CREATED_BY"}},
					{ID: createdAtFieldUUID, Name: "Created At", Type: "datetime", SemanticTags: []string{"CREATED_AT"}},
					{ID: updatedByFieldUUID, Name: "Updated By", Type: "user", SemanticTags: []string{"UPDATED_BY"}},
					{ID: updatedAtFieldUUID, Name: "Updated At", Type: "datetime", SemanticTags: []string{"UPDATED_AT"}},
					{ID: closedByFieldUUID, Name: "Closed By", Type: "user", SemanticTags: []string{"CLOSED_BY"}},
					{ID: closedAtFieldUUID, Name: "Closed At", Type: "datetime", SemanticTags: []string{"CLOSED_AT"}},
				},
			},
		},
	}

	uuidMap := buildUUIDMap(imports, app)

	// TITLE and ID should map to locals (element resource exposes these)
	tests := []struct {
		uuid     string
		expected string
		desc     string
	}{
		{titleFieldUUID, "local.myelement_title_field_id", "TITLE field should map to local"},
		{idFieldUUID, "local.myelement_id_field_id", "ID/HANDLE field should map to local"},
		// Audit fields should map to data sources (element resource doesn't expose these)
		{createdByFieldUUID, "data.elementum_field.myelement_created_by.id", "CREATED_BY should map to data source"},
		{createdAtFieldUUID, "data.elementum_field.myelement_created_at.id", "CREATED_AT should map to data source"},
		{updatedByFieldUUID, "data.elementum_field.myelement_updated_by.id", "UPDATED_BY should map to data source"},
		{updatedAtFieldUUID, "data.elementum_field.myelement_updated_at.id", "UPDATED_AT should map to data source"},
		{closedByFieldUUID, "data.elementum_field.myelement_closed_by.id", "CLOSED_BY should map to data source"},
		{closedAtFieldUUID, "data.elementum_field.myelement_closed_at.id", "CLOSED_AT should map to data source"},
	}

	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			if got, ok := uuidMap[tc.uuid]; !ok {
				t.Errorf("%s: UUID not found in map", tc.desc)
			} else if got != tc.expected {
				t.Errorf("%s: got %q, want %q", tc.desc, got, tc.expected)
			}
		})
	}
}

// TestBuildUUIDMap_MultipleElementsWithSystemFields tests correct mapping
// when multiple discovered elements have system fields
func TestBuildUUIDMap_MultipleElementsWithSystemFields(t *testing.T) {
	t.Parallel()

	// Use proper UUID format for testing (tests don't use replaceUUIDs so format doesn't matter for lookup)
	element1CreatedBy := "elem1-created-by"
	element2CreatedBy := "elem2-created-by"
	element1UUID := "element-1-uuid"
	element2UUID := "element-2-uuid"
	appUUID := "app-uuid"

	imports := []ImportBlock{
		{ID: appUUID, ResourceType: "elementum_app", ResourceName: "my_app"},
		{ID: element1UUID, ResourceType: "elementum_element", ResourceName: "products"},
		{ID: element2UUID, ResourceType: "elementum_element", ResourceName: "orders"},
	}

	app := &discovery.App{
		ID:        appUUID,
		Name:      "My App",
		Namespace: "myapp",
		DiscoveredElements: []*discovery.Element{
			{
				ID:        element1UUID,
				Name:      "Products",
				Namespace: "products",
				Fields: []discovery.Field{
					{ID: element1CreatedBy, Name: "Created By", Type: "user", SemanticTags: []string{"CREATED_BY"}},
				},
			},
			{
				ID:        element2UUID,
				Name:      "Orders",
				Namespace: "orders",
				Fields: []discovery.Field{
					{ID: element2CreatedBy, Name: "Created By", Type: "user", SemanticTags: []string{"CREATED_BY"}},
				},
			},
		},
	}

	uuidMap := buildUUIDMap(imports, app)

	// Each element's CREATED_BY should map to its own data source
	if got, ok := uuidMap[element1CreatedBy]; !ok {
		t.Error("Element 1 CREATED_BY not in map")
	} else if got != "data.elementum_field.products_created_by.id" {
		t.Errorf("Element 1 CREATED_BY = %q, want %q", got, "data.elementum_field.products_created_by.id")
	}

	if got, ok := uuidMap[element2CreatedBy]; !ok {
		t.Error("Element 2 CREATED_BY not in map")
	} else if got != "data.elementum_field.orders_created_by.id" {
		t.Errorf("Element 2 CREATED_BY = %q, want %q", got, "data.elementum_field.orders_created_by.id")
	}
}

// TestBuildUUIDMap_ElementWithoutSystemFields tests that elements without
// system fields don't cause errors
func TestBuildUUIDMap_ElementWithoutSystemFields(t *testing.T) {
	t.Parallel()

	regularFieldUUID := "regular-field-uuid"
	elementUUID := "element-uuid"
	appUUID := "app-uuid"

	imports := []ImportBlock{
		{ID: appUUID, ResourceType: "elementum_app", ResourceName: "my_app"},
		{ID: elementUUID, ResourceType: "elementum_element", ResourceName: "my_element"},
		{ID: appUUID + ":" + regularFieldUUID, ResourceType: "elementum_text_field", ResourceName: "my_field"},
	}

	app := &discovery.App{
		ID:        appUUID,
		Name:      "My App",
		Namespace: "myapp",
		DiscoveredElements: []*discovery.Element{
			{
				ID:        elementUUID,
				Name:      "My Element",
				Namespace: "myelement",
				Fields: []discovery.Field{
					// No semantic tags - regular field only
					{ID: regularFieldUUID, Name: "My Field", Type: "text"},
				},
			},
		},
	}

	// Should not panic
	uuidMap := buildUUIDMap(imports, app)

	// Regular field should still be mapped via resource reference
	if got, ok := uuidMap[regularFieldUUID]; !ok {
		t.Error("Regular field should be in map")
	} else if got != "elementum_text_field.my_field.id" {
		t.Errorf("Regular field = %q, want %q", got, "elementum_text_field.my_field.id")
	}
}

// =============================================================================
// Discovered App Field UUID Mapping Tests
// =============================================================================

// TestBuildUUIDMap_DiscoveredAppSystemFields tests that discovered app system
// fields are correctly mapped to local references
func TestBuildUUIDMap_DiscoveredAppSystemFields(t *testing.T) {
	t.Parallel()

	// UUIDs for discovered app system fields
	titleFieldUUID := "title-field-uuid"
	statusFieldUUID := "status-field-uuid"
	idFieldUUID := "id-field-uuid"
	createdByFieldUUID := "created-by-field-uuid"
	createdAtFieldUUID := "created-at-field-uuid"
	updatedByFieldUUID := "updated-by-field-uuid"
	updatedAtFieldUUID := "updated-at-field-uuid"
	closedByFieldUUID := "closed-by-field-uuid"
	closedAtFieldUUID := "closed-at-field-uuid"
	discoveredAppUUID := "discovered-app-uuid"
	mainAppUUID := "main-app-uuid"

	imports := []ImportBlock{
		{ID: mainAppUUID, ResourceType: "elementum_app", ResourceName: "main_app"},
		{ID: discoveredAppUUID, ResourceType: "elementum_app", ResourceName: "requestdev"},
	}

	app := &discovery.App{
		ID:        mainAppUUID,
		Name:      "Main App",
		Namespace: "mainapp",
		DiscoveredApps: []*discovery.App{
			{
				ID:        discoveredAppUUID,
				Name:      "Request Dev",
				Namespace: "requestdev",
				Fields: []discovery.Field{
					{ID: titleFieldUUID, Name: "Title", Type: "text", SemanticTags: []string{"TITLE"}},
					{ID: statusFieldUUID, Name: "Status", Type: "dropdown", SemanticTags: []string{"STATUS"}},
					{ID: idFieldUUID, Name: "ID", Type: "text", SemanticTags: []string{"HANDLE"}},
					{ID: createdByFieldUUID, Name: "Created By", Type: "user", SemanticTags: []string{"CREATED_BY"}},
					{ID: createdAtFieldUUID, Name: "Created At", Type: "datetime", SemanticTags: []string{"CREATED_AT"}},
					{ID: updatedByFieldUUID, Name: "Updated By", Type: "user", SemanticTags: []string{"UPDATED_BY"}},
					{ID: updatedAtFieldUUID, Name: "Updated At", Type: "datetime", SemanticTags: []string{"UPDATED_AT"}},
					{ID: closedByFieldUUID, Name: "Closed By", Type: "user", SemanticTags: []string{"CLOSED_BY"}},
					{ID: closedAtFieldUUID, Name: "Closed At", Type: "datetime", SemanticTags: []string{"CLOSED_AT"}},
				},
			},
		},
	}

	uuidMap := buildUUIDMap(imports, app)

	// Discovered app system fields should map to locals with the discovered app's prefix
	tests := []struct {
		uuid     string
		expected string
		desc     string
	}{
		{titleFieldUUID, "local.requestdev_title_field_id", "TITLE field"},
		{statusFieldUUID, "local.requestdev_status_field_id", "STATUS field"},
		{idFieldUUID, "local.requestdev_id_field_id", "ID/HANDLE field"},
		{createdByFieldUUID, "local.requestdev_created_by_field_id", "CREATED_BY field"},
		{createdAtFieldUUID, "local.requestdev_created_at_field_id", "CREATED_AT field"},
		{updatedByFieldUUID, "local.requestdev_updated_by_field_id", "UPDATED_BY field"},
		{updatedAtFieldUUID, "local.requestdev_updated_at_field_id", "UPDATED_AT field"},
		{closedByFieldUUID, "local.requestdev_closed_by_field_id", "CLOSED_BY field"},
		{closedAtFieldUUID, "local.requestdev_closed_at_field_id", "CLOSED_AT field"},
	}

	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			if got, ok := uuidMap[tc.uuid]; !ok {
				t.Errorf("%s: UUID not found in map", tc.desc)
			} else if got != tc.expected {
				t.Errorf("%s: got %q, want %q", tc.desc, got, tc.expected)
			}
		})
	}
}

// TestBuildUUIDMap_DiscoveredAppStatusOptions tests that discovered app status
// field option IDs are correctly mapped to direct resource references (Phase 2)
func TestBuildUUIDMap_DiscoveredAppStatusOptions(t *testing.T) {
	t.Parallel()

	statusFieldUUID := "status-field-uuid"
	openOptionUUID := "open-option-uuid"
	closedOptionUUID := "closed-option-uuid"
	discoveredAppUUID := "discovered-app-uuid"
	mainAppUUID := "main-app-uuid"

	imports := []ImportBlock{
		{ID: mainAppUUID, ResourceType: "elementum_app", ResourceName: "main_app"},
		{ID: discoveredAppUUID, ResourceType: "elementum_app", ResourceName: "incidentdev"},
	}

	app := &discovery.App{
		ID:        mainAppUUID,
		Name:      "Main App",
		Namespace: "mainapp",
		DiscoveredApps: []*discovery.App{
			{
				ID:        discoveredAppUUID,
				Name:      "Incident Dev",
				Namespace: "incidentdev",
				Fields: []discovery.Field{
					{
						ID:           statusFieldUUID,
						Name:         "Status",
						Type:         "dropdown",
						SemanticTags: []string{"STATUS"},
						Options: []discovery.FieldOption{
							{ID: openOptionUUID, Label: "Open"},
							{ID: closedOptionUUID, Label: "Closed"},
						},
					},
				},
			},
		},
	}

	uuidMap := buildUUIDMap(imports, app)

	// Check status option mappings - use direct resource reference (Phase 2)
	expectedOpenRef := `elementum_app.incidentdev.status_option_ids_by_label["Open"]`
	if got, ok := uuidMap[openOptionUUID]; !ok {
		t.Error("Open option UUID not found in map")
	} else if got != expectedOpenRef {
		t.Errorf("Open option mapping = %q, want %q", got, expectedOpenRef)
	}

	expectedClosedRef := `elementum_app.incidentdev.status_option_ids_by_label["Closed"]`
	if got, ok := uuidMap[closedOptionUUID]; !ok {
		t.Error("Closed option UUID not found in map")
	} else if got != expectedClosedRef {
		t.Errorf("Closed option mapping = %q, want %q", got, expectedClosedRef)
	}
}

// TestGenerateLocals_DiscoveredApps tests that locals are generated correctly
// for discovered apps
func TestGenerateLocals_DiscoveredApps(t *testing.T) {
	t.Parallel()

	imports := []ImportBlock{
		{ID: "main-app-uuid", ResourceType: "elementum_app", ResourceName: "main_app"},
		{ID: "requestdev-uuid", ResourceType: "elementum_app", ResourceName: "requestdev"},
		{ID: "incidentdev-uuid", ResourceType: "elementum_app", ResourceName: "incidentdev"},
	}

	app := &discovery.App{
		ID:        "main-app-uuid",
		Name:      "Main App",
		Namespace: "mainapp",
		DiscoveredApps: []*discovery.App{
			{
				ID:        "requestdev-uuid",
				Name:      "Request Dev",
				Namespace: "requestdev",
			},
			{
				ID:        "incidentdev-uuid",
				Name:      "Incident Dev",
				Namespace: "incidentdev",
			},
		},
	}

	result := GenerateLocals(app, "main_app", imports)

	// Check that locals block is generated
	if !strings.Contains(result, "locals {") {
		t.Error("Locals block not generated")
	}

	// Check for discovered app system field locals
	if !strings.Contains(result, "# Discovered app system field IDs") {
		t.Error("Discovered app system field IDs comment not generated")
	}

	// Check for requestdev system fields
	if !strings.Contains(result, "requestdev_title_field_id") {
		t.Error("requestdev_title_field_id not generated")
	}
	if !strings.Contains(result, "requestdev_status_field_id") {
		t.Error("requestdev_status_field_id not generated")
	}
	if !strings.Contains(result, "requestdev_id_field_id") {
		t.Error("requestdev_id_field_id not generated")
	}
	if !strings.Contains(result, "requestdev_created_by_field_id") {
		t.Error("requestdev_created_by_field_id not generated")
	}
	if !strings.Contains(result, "requestdev_status_options_by_label") {
		t.Error("requestdev_status_options_by_label not generated")
	}

	// Check for incidentdev system fields
	if !strings.Contains(result, "incidentdev_title_field_id") {
		t.Error("incidentdev_title_field_id not generated")
	}
	if !strings.Contains(result, "incidentdev_status_options_by_label") {
		t.Error("incidentdev_status_options_by_label not generated")
	}

	// Check for proper reference format
	if !strings.Contains(result, "elementum_app.requestdev.title_field_id") {
		t.Error("Expected elementum_app.requestdev.title_field_id reference")
	}
	if !strings.Contains(result, "elementum_app.incidentdev.status_field_id") {
		t.Error("Expected elementum_app.incidentdev.status_field_id reference")
	}
}

// TestGenerateLocals_NoDiscoveredApps tests that locals generation works
// when there are no discovered apps
func TestGenerateLocals_NoDiscoveredApps(t *testing.T) {
	t.Parallel()

	imports := []ImportBlock{
		{ID: "app-uuid", ResourceType: "elementum_app", ResourceName: "my_app"},
	}

	app := &discovery.App{
		ID:             "app-uuid",
		Name:           "My App",
		Namespace:      "myapp",
		DiscoveredApps: []*discovery.App{}, // No discovered apps
	}

	result := GenerateLocals(app, "my_app", imports)

	// Locals block should still be generated (for main app system fields)
	if !strings.Contains(result, "locals {") {
		t.Error("Locals block not generated")
	}

	// Should NOT contain discovered app section
	if strings.Contains(result, "# Discovered app system field IDs") {
		t.Error("Discovered app section should not be present when no discovered apps")
	}
}

// =============================================================================
// Search Table ID Warning Tests
// =============================================================================

// =============================================================================
// AI Provider Connector from Search Tables Tests
// =============================================================================

// TestBuildUUIDMap_AIProviderConnectorsFromSearchTables tests that AI provider
// connectors from AI search tables are correctly mapped
func TestBuildUUIDMap_AIProviderConnectorsFromSearchTables(t *testing.T) {
	t.Parallel()

	connectorUUID := "connector-uuid-123"
	searchTableUUID := "search-table-uuid"
	appUUID := "app-uuid"

	imports := []ImportBlock{
		{ID: appUUID, ResourceType: "elementum_app", ResourceName: "my_app"},
		{ID: appUUID + "/" + searchTableUUID, ResourceType: "elementum_ai_search_table", ResourceName: "kb_search"},
	}

	app := &discovery.App{
		ID:   appUUID,
		Name: "My App",
		AISearchTables: []discovery.AISearchTable{
			{
				ID:                      searchTableUUID,
				AIProviderConnectorID:   connectorUUID,
				AIProviderConnectorName: "Snowflake Arctic L V2.0",
			},
		},
		DiscoveredAiProviderConnectors: []discovery.AiProviderConnector{
			{
				ID:        connectorUUID,
				ModelName: "Snowflake Arctic L V2.0",
			},
		},
	}

	uuidMap := buildUUIDMap(imports, app)

	// Check AI provider connector mapping
	expectedRef := "data.elementum_ai_provider_connector.snowflake_arctic_l_v2_0.id"
	if got, ok := uuidMap[connectorUUID]; !ok {
		t.Error("AI provider connector UUID not found in map")
	} else if got != expectedRef {
		t.Errorf("AI provider connector mapping = %q, want %q", got, expectedRef)
	}
}

// =============================================================================
// Phase 1: Stage Options By Label Tests (Issue #11)
// =============================================================================

// TestGenerateLocals_StageOptionsByLabel tests that stage_options_by_label
// locals are generated for the primary app when it has stages
func TestGenerateLocals_StageOptionsByLabel(t *testing.T) {
	t.Parallel()

	imports := []ImportBlock{
		{ID: "app-uuid", ResourceType: "elementum_app", ResourceName: "my_app"},
	}

	app := &discovery.App{
		ID:        "app-uuid",
		Name:      "My App",
		Namespace: "myapp",
		Layouts: []discovery.Layout{
			{ID: "layout-1", Name: "Intake"},
			{ID: "layout-2", Name: "In Progress"},
			{ID: "layout-3", Name: "Complete"},
		},
	}

	result := GenerateLocals(app, "my_app", imports)

	// Check that locals block is generated
	if !strings.Contains(result, "locals {") {
		t.Error("Locals block not generated")
	}

	// Check for stage_options_by_label comment
	if !strings.Contains(result, "Stage option IDs by label") {
		t.Error("Stage option IDs by label comment not generated")
	}

	// Check for stage_options_by_label local variable
	if !strings.Contains(result, "my_app_stage_options_by_label") {
		t.Error("my_app_stage_options_by_label local not generated")
	}

	// Check for the for-expression pattern using stages list
	if !strings.Contains(result, "for s in elementum_app.my_app.stages : s.name => s.id") {
		t.Errorf("Stage options lookup for-expression not generated correctly. Got:\n%s", result)
	}
}

// TestGenerateLocals_StageOptionsByLabel_NoStages tests that stage_options_by_label
// is NOT generated when app has no stages (no layouts)
func TestGenerateLocals_StageOptionsByLabel_NoStages(t *testing.T) {
	t.Parallel()

	imports := []ImportBlock{
		{ID: "app-uuid", ResourceType: "elementum_app", ResourceName: "my_app"},
	}

	app := &discovery.App{
		ID:        "app-uuid",
		Name:      "My App",
		Namespace: "myapp",
		Layouts:   []discovery.Layout{}, // No stages
	}

	result := GenerateLocals(app, "my_app", imports)

	// Should NOT contain stage_options_by_label since no stages
	if strings.Contains(result, "stage_options_by_label") {
		t.Error("stage_options_by_label should not be generated when app has no stages")
	}
}

// TestGenerateLocals_DiscoveredAppStageOptionsByLabel tests that stage_options_by_label
// locals are generated for discovered apps
func TestGenerateLocals_DiscoveredAppStageOptionsByLabel(t *testing.T) {
	t.Parallel()

	imports := []ImportBlock{
		{ID: "main-app-uuid", ResourceType: "elementum_app", ResourceName: "main_app"},
		{ID: "velocityactivitytasks-uuid", ResourceType: "elementum_app", ResourceName: "velocityactivitytasks"},
	}

	app := &discovery.App{
		ID:        "main-app-uuid",
		Name:      "Main App",
		Namespace: "mainapp",
		Layouts: []discovery.Layout{
			{ID: "layout-1", Name: "Open"},
		},
		DiscoveredApps: []*discovery.App{
			{
				ID:        "velocityactivitytasks-uuid",
				Name:      "Velocity Activity Tasks",
				Namespace: "velocityactivitytasks",
				Layouts: []discovery.Layout{
					{ID: "layout-a", Name: "Queued"},
					{ID: "layout-b", Name: "Ready"},
					{ID: "layout-c", Name: "Acting"},
				},
			},
		},
	}

	result := GenerateLocals(app, "main_app", imports)

	// Check for discovered app stage_options_by_label
	if !strings.Contains(result, "velocityactivitytasks_stage_options_by_label") {
		t.Errorf("velocityactivitytasks_stage_options_by_label local not generated. Got:\n%s", result)
	}

	// Check for the for-expression pattern for discovered app
	if !strings.Contains(result, "for s in elementum_app.velocityactivitytasks.stages : s.name => s.id") {
		t.Errorf("Discovered app stage options lookup for-expression not generated correctly. Got:\n%s", result)
	}
}

// TestBuildUUIDMap_StageOptionsMappedToLocals tests that stage option IDs
// are mapped to local.appname_stage_options_by_label["Label"] references
func TestBuildUUIDMap_StageOptionsMappedToLocals(t *testing.T) {
	t.Parallel()

	stageIntakeOptID := "stage-intake-opt-uuid"
	stageReviewOptID := "stage-review-opt-uuid"
	stageFieldID := "stage-field-uuid"
	appUUID := "app-uuid"

	imports := []ImportBlock{
		{ID: appUUID, ResourceType: "elementum_app", ResourceName: "my_app"},
	}

	app := &discovery.App{
		ID:        appUUID,
		Name:      "My App",
		Namespace: "myapp",
		Fields: []discovery.Field{
			{
				ID:           stageFieldID,
				Name:         "Stage",
				Type:         "dropdown",
				SemanticTags: []string{"STAGE"},
				Options: []discovery.FieldOption{
					{ID: stageIntakeOptID, Label: "Intake"},
					{ID: stageReviewOptID, Label: "Review"},
				},
			},
		},
		Layouts: []discovery.Layout{
			{ID: "layout-1", Name: "Intake"},
			{ID: "layout-2", Name: "Review"},
		},
	}

	uuidMap := buildUUIDMap(imports, app)

	// Check stage field ID mapping
	expectedFieldRef := "local.my_app_stage_field_id"
	if got, ok := uuidMap[stageFieldID]; !ok {
		t.Error("Stage field UUID not found in map")
	} else if got != expectedFieldRef {
		t.Errorf("Stage field mapping = %q, want %q", got, expectedFieldRef)
	}

	// Check stage option ID mappings
	expectedIntakeRef := `local.my_app_stage_options_by_label["Intake"]`
	if got, ok := uuidMap[stageIntakeOptID]; !ok {
		t.Error("Stage Intake option UUID not found in map")
	} else if got != expectedIntakeRef {
		t.Errorf("Stage Intake option mapping = %q, want %q", got, expectedIntakeRef)
	}

	expectedReviewRef := `local.my_app_stage_options_by_label["Review"]`
	if got, ok := uuidMap[stageReviewOptID]; !ok {
		t.Error("Stage Review option UUID not found in map")
	} else if got != expectedReviewRef {
		t.Errorf("Stage Review option mapping = %q, want %q", got, expectedReviewRef)
	}
}

// TestBuildUUIDMap_DiscoveredAppStageOptions tests that stage option IDs
// from discovered apps are correctly mapped
func TestBuildUUIDMap_DiscoveredAppStageOptions(t *testing.T) {
	t.Parallel()

	stageFieldUUID := "disc-stage-field-uuid"
	actingOptionUUID := "acting-option-uuid"
	queuedOptionUUID := "queued-option-uuid"
	discoveredAppUUID := "velocityactivitytasks-uuid"
	mainAppUUID := "main-app-uuid"

	imports := []ImportBlock{
		{ID: mainAppUUID, ResourceType: "elementum_app", ResourceName: "main_app"},
		{ID: discoveredAppUUID, ResourceType: "elementum_app", ResourceName: "velocityactivitytasks"},
	}

	app := &discovery.App{
		ID:        mainAppUUID,
		Name:      "Main App",
		Namespace: "mainapp",
		DiscoveredApps: []*discovery.App{
			{
				ID:        discoveredAppUUID,
				Name:      "Velocity Activity Tasks",
				Namespace: "velocityactivitytasks",
				Fields: []discovery.Field{
					{
						ID:           stageFieldUUID,
						Name:         "Stage",
						Type:         "dropdown",
						SemanticTags: []string{"STAGE"},
						Options: []discovery.FieldOption{
							{ID: actingOptionUUID, Label: "Acting"},
							{ID: queuedOptionUUID, Label: "Queued"},
						},
					},
				},
				Layouts: []discovery.Layout{
					{ID: "layout-1", Name: "Acting"},
					{ID: "layout-2", Name: "Queued"},
				},
			},
		},
	}

	uuidMap := buildUUIDMap(imports, app)

	// Check discovered app stage field ID mapping
	expectedFieldRef := "local.velocityactivitytasks_stage_field_id"
	if got, ok := uuidMap[stageFieldUUID]; !ok {
		t.Error("Discovered app stage field UUID not found in map")
	} else if got != expectedFieldRef {
		t.Errorf("Discovered app stage field mapping = %q, want %q", got, expectedFieldRef)
	}

	// Check discovered app stage option ID mappings
	expectedActingRef := `local.velocityactivitytasks_stage_options_by_label["Acting"]`
	if got, ok := uuidMap[actingOptionUUID]; !ok {
		t.Error("Acting option UUID not found in map")
	} else if got != expectedActingRef {
		t.Errorf("Acting option mapping = %q, want %q", got, expectedActingRef)
	}

	expectedQueuedRef := `local.velocityactivitytasks_stage_options_by_label["Queued"]`
	if got, ok := uuidMap[queuedOptionUUID]; !ok {
		t.Error("Queued option UUID not found in map")
	} else if got != expectedQueuedRef {
		t.Errorf("Queued option mapping = %q, want %q", got, expectedQueuedRef)
	}
}

// =============================================================================
// Phase 2: Direct Status Option References Tests
// =============================================================================

// TestBuildUUIDMap_StatusOptionsDirectReference tests that status option IDs
// are mapped directly to elementum_app.*.status_option_ids_by_label["Label"]
// instead of using locals (Phase 2 simplification)
func TestBuildUUIDMap_StatusOptionsDirectReference(t *testing.T) {
	t.Parallel()

	statusFieldUUID := "status-field-uuid"
	openOptionUUID := "open-option-uuid"
	closedOptionUUID := "closed-option-uuid"
	appUUID := "app-uuid"

	imports := []ImportBlock{
		{ID: appUUID, ResourceType: "elementum_app", ResourceName: "my_app"},
	}

	app := &discovery.App{
		ID:        appUUID,
		Name:      "My App",
		Namespace: "myapp",
		Fields: []discovery.Field{
			{
				ID:           statusFieldUUID,
				Name:         "Status",
				Type:         "dropdown",
				SemanticTags: []string{"STATUS"},
				Options: []discovery.FieldOption{
					{ID: openOptionUUID, Label: "Open"},
					{ID: closedOptionUUID, Label: "Closed"},
				},
			},
		},
	}

	uuidMap := buildUUIDMap(imports, app)

	// Phase 2: Status options should use direct resource reference, not locals
	expectedOpenRef := `elementum_app.my_app.status_option_ids_by_label["Open"]`
	if got, ok := uuidMap[openOptionUUID]; !ok {
		t.Error("Open option UUID not found in map")
	} else if got != expectedOpenRef {
		t.Errorf("Open option mapping = %q, want %q (direct resource reference)", got, expectedOpenRef)
	}

	expectedClosedRef := `elementum_app.my_app.status_option_ids_by_label["Closed"]`
	if got, ok := uuidMap[closedOptionUUID]; !ok {
		t.Error("Closed option UUID not found in map")
	} else if got != expectedClosedRef {
		t.Errorf("Closed option mapping = %q, want %q (direct resource reference)", got, expectedClosedRef)
	}
}

// TestBuildUUIDMap_DiscoveredAppStatusOptionsDirectReference tests that status
// option IDs from discovered apps use direct resource references
func TestBuildUUIDMap_DiscoveredAppStatusOptionsDirectReference(t *testing.T) {
	t.Parallel()

	statusFieldUUID := "disc-status-field-uuid"
	pendingOptionUUID := "pending-option-uuid"
	approvedOptionUUID := "approved-option-uuid"
	discoveredAppUUID := "incidentdev-uuid"
	mainAppUUID := "main-app-uuid"

	imports := []ImportBlock{
		{ID: mainAppUUID, ResourceType: "elementum_app", ResourceName: "main_app"},
		{ID: discoveredAppUUID, ResourceType: "elementum_app", ResourceName: "incidentdev"},
	}

	app := &discovery.App{
		ID:        mainAppUUID,
		Name:      "Main App",
		Namespace: "mainapp",
		DiscoveredApps: []*discovery.App{
			{
				ID:        discoveredAppUUID,
				Name:      "Incident Dev",
				Namespace: "incidentdev",
				Fields: []discovery.Field{
					{
						ID:           statusFieldUUID,
						Name:         "Status",
						Type:         "dropdown",
						SemanticTags: []string{"STATUS"},
						Options: []discovery.FieldOption{
							{ID: pendingOptionUUID, Label: "Pending"},
							{ID: approvedOptionUUID, Label: "Approved"},
						},
					},
				},
			},
		},
	}

	uuidMap := buildUUIDMap(imports, app)

	// Phase 2: Discovered app status options should use direct resource reference
	expectedPendingRef := `elementum_app.incidentdev.status_option_ids_by_label["Pending"]`
	if got, ok := uuidMap[pendingOptionUUID]; !ok {
		t.Error("Pending option UUID not found in map")
	} else if got != expectedPendingRef {
		t.Errorf("Pending option mapping = %q, want %q (direct resource reference)", got, expectedPendingRef)
	}

	expectedApprovedRef := `elementum_app.incidentdev.status_option_ids_by_label["Approved"]`
	if got, ok := uuidMap[approvedOptionUUID]; !ok {
		t.Error("Approved option UUID not found in map")
	} else if got != expectedApprovedRef {
		t.Errorf("Approved option mapping = %q, want %q (direct resource reference)", got, expectedApprovedRef)
	}
}
