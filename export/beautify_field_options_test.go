// Copyright (c) ElementumAI, Inc.
// SPDX-License-Identifier: MPL-2.0

package export

import (
	"strings"
	"testing"

	"github.com/elementumltd/elementum-cli/discovery"
)

// TestBeautifyDataminePrimaryColumnIDs tests that table field IDs in datamine primary_column_ids are mapped
func TestBeautifyDataminePrimaryColumnIDs(t *testing.T) {
	fieldPathID := "b56743fa-37b6-4aaf-bf2c-b405925584d3"
	fieldSizeID := "463bcb97-fd52-4633-a211-c9081fb81c16"
	fieldModifiedID := "27a5c963-3805-4c6e-8f74-27d0a8d947a6"

	table := &discovery.Table{
		ID:     "table-123",
		Name:   "FAQ Table",
		Handle: "faq_table",
		Fields: []discovery.TableField{
			{ID: fieldPathID, Name: "RELATIVE_PATH", Type: "TEXT"},
			{ID: fieldSizeID, Name: "SIZE", Type: "NUMBER"},
			{ID: fieldModifiedID, Name: "LAST_MODIFIED", Type: "DATE"},
		},
	}

	datamine := &discovery.Datamine{
		ID:               "dm-456",
		Name:             "Knowledge Base",
		PrimaryColumnIDs: []string{fieldPathID, fieldSizeID, fieldModifiedID},
	}

	imports := []ImportBlock{
		{
			ID:           "dm-456",
			ResourceType: "elementum_datamine",
			ResourceName: "knowledge_base",
		},
		{
			ID:           "table-123",
			ResourceType: "elementum_table",
			ResourceName: "faq_table",
		},
	}

	inputHCL := `resource "elementum_datamine" "knowledge_base" {
  name               = "Knowledge Base"
  table_id           = "table-123"
  primary_column_ids = ["` + fieldPathID + `", "` + fieldSizeID + `", "` + fieldModifiedID + `"]
}`

	result := BeautifyDatamineConfig(inputHCL, imports, datamine, table, nil)

	// Should map table field IDs to data source references
	if !strings.Contains(result, `data.elementum_field.faq_table_relative_path.id`) {
		t.Errorf("Expected table field to be mapped to data source, got:\n%s", result)
	}
	if !strings.Contains(result, `data.elementum_field.faq_table_size.id`) {
		t.Errorf("Expected SIZE field to be mapped, got:\n%s", result)
	}
	if !strings.Contains(result, `data.elementum_field.faq_table_last_modified.id`) {
		t.Errorf("Expected LAST_MODIFIED field to be mapped, got:\n%s", result)
	}
}

// TestBuildUUIDMapWithFieldOptions tests that field option IDs are added to UUID map
func TestBuildUUIDMapWithFieldOptions(t *testing.T) {
	app := &discovery.App{
		ID:   "app-123",
		Name: "Test App",
		Fields: []discovery.Field{
			{
				ID:           "field-status",
				Name:         "Status",
				Type:         "dropdown",
				SemanticTags: []string{"STATUS"},
				Options: []discovery.FieldOption{
					{ID: "opt-open", Label: "Open", Color: "green"},
					{ID: "opt-closed", Label: "Closed", Color: "red"},
				},
			},
			{
				ID:   "field-category",
				Name: "Category",
				Type: "dropdown",
				Options: []discovery.FieldOption{
					{ID: "cat-bug", Label: "Bug", Color: "red"},
					{ID: "cat-feature", Label: "Feature", Color: "blue"},
				},
			},
		},
	}

	imports := []ImportBlock{
		{
			ID:           "app-123",
			ResourceType: "elementum_app",
			ResourceName: "test_app",
		},
		{
			ID:           "app-123:field-category",
			ResourceType: "elementum_dropdown_field",
			ResourceName: "category",
		},
	}

	uuidMap := buildUUIDMap(imports, app)

	// Check status options (system field) - uses direct resource reference (Phase 2)
	expectedStatusRef := `elementum_app.test_app.status_option_ids_by_label["Open"]`
	if ref, ok := uuidMap["opt-open"]; !ok || ref != expectedStatusRef {
		t.Errorf("Status option not mapped correctly, expected %q, got: %s", expectedStatusRef, ref)
	}

	// Check custom dropdown options (still uses locals)
	if ref, ok := uuidMap["cat-bug"]; !ok || !strings.Contains(ref, "category_options_by_label") {
		t.Errorf("Custom dropdown option not mapped correctly, got: %s", ref)
	}
}

// TestFieldGetOptionIDMappings tests the new GetOptionIDMappings method
func TestFieldGetOptionIDMappings(t *testing.T) {
	field := discovery.Field{
		ID:   "field-123",
		Name: "Priority",
		Type: "dropdown",
		Options: []discovery.FieldOption{
			{ID: "opt-1", Label: "High"},
			{ID: "opt-2", Label: "Medium"},
			{ID: "opt-3", Label: "Low"},
		},
	}

	optionMap := field.GetOptionIDMappings()

	if len(optionMap) != 3 {
		t.Errorf("Expected 3 options, got %d", len(optionMap))
	}

	if optionMap["opt-1"] != "High" {
		t.Errorf("Expected 'High', got '%s'", optionMap["opt-1"])
	}
}
