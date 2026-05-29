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

// TestBuildSection_GroupWithFieldIDs verifies that field_ids are included in
// the exported HCL for group sections with field IDs.
func TestBuildSection_GroupWithFieldIDs(t *testing.T) {
	t.Parallel()

	app := &discovery.App{
		ID: "app-123",
	}
	uuidMap := map[string]string{
		"field-1": "elementum_text_field.description.id",
		"field-2": "elementum_number_field.quantity.id",
	}

	g := NewAppHCLGenerator(app, nil, uuidMap)

	block := &discovery.DisplayBlock{
		ID:              "block-1",
		Type:            "group",
		Name:            "Details",
		FieldIDs:        []string{"field-1", "field-2"},
		Icon:            "star",
		Color:           "#FF0000",
		DisplayOrder:    1,
		DisplayLocation: "CENTER",
	}

	section := g.buildSection(block)
	hcl := renderHCLValue(section)

	if !strings.Contains(hcl, `type = "group"`) {
		t.Error("expected type = \"group\" in output")
	}
	if !strings.Contains(hcl, `name = "Details"`) {
		t.Error("expected name = \"Details\" in output")
	}
	if !strings.Contains(hcl, "elementum_text_field.description.id") {
		t.Error("expected field-1 resolved reference in field_ids")
	}
	if !strings.Contains(hcl, "elementum_number_field.quantity.id") {
		t.Error("expected field-2 resolved reference in field_ids")
	}
	if !strings.Contains(hcl, `icon = "star"`) {
		t.Error("expected icon = \"star\" in output")
	}
	if !strings.Contains(hcl, `color = "#FF0000"`) {
		t.Error("expected color in output")
	}
}

// TestBuildSection_GroupFiltersSystemFields verifies that system fields
// (HANDLE, TITLE, STATUS) are excluded from field_ids in exported layout HCL.
func TestBuildSection_GroupFiltersSystemFields(t *testing.T) {
	t.Parallel()

	app := &discovery.App{
		ID: "app-123",
		Fields: []discovery.Field{
			{ID: "title-field-id", Name: "Title", SemanticTags: []string{"TITLE"}},
			{ID: "status-field-id", Name: "Status", SemanticTags: []string{"STATUS"}},
			{ID: "id-field-id", Name: "ID", SemanticTags: []string{"HANDLE"}},
			{ID: "custom-field-id", Name: "Description", SemanticTags: []string{}},
		},
	}
	uuidMap := map[string]string{
		"title-field-id":  "local.myapp_title_field_id",
		"status-field-id": "local.myapp_status_field_id",
		"id-field-id":     "local.myapp_id_field_id",
		"custom-field-id": "elementum_text_field.description.id",
	}

	g := NewAppHCLGenerator(app, nil, uuidMap)

	block := &discovery.DisplayBlock{
		ID:   "block-1",
		Type: "group",
		Name: "System",
		FieldIDs: []string{
			"title-field-id",
			"status-field-id",
			"id-field-id",
			"custom-field-id",
		},
		DisplayOrder:    0,
		DisplayLocation: "CENTER",
	}

	section := g.buildSection(block)
	hcl := renderHCLValue(section)

	// System fields should be EXCLUDED
	if strings.Contains(hcl, "title_field_id") {
		t.Error("system field title_field_id should not appear in field_ids")
	}
	if strings.Contains(hcl, "status_field_id") {
		t.Error("system field status_field_id should not appear in field_ids")
	}
	if strings.Contains(hcl, "id_field_id") {
		t.Error("system field id_field_id should not appear in field_ids")
	}

	// Custom field should be INCLUDED
	if !strings.Contains(hcl, "elementum_text_field.description.id") {
		t.Error("custom field should appear in field_ids")
	}
}

// TestBuildSection_AllSystemFieldsFiltered verifies that when ALL fields
// in a section are system fields, the field_ids attribute is omitted.
func TestBuildSection_AllSystemFieldsFiltered(t *testing.T) {
	t.Parallel()

	app := &discovery.App{
		ID: "app-123",
		Fields: []discovery.Field{
			{ID: "f1", Name: "Title", SemanticTags: []string{"TITLE"}},
			{ID: "f2", Name: "Status", SemanticTags: []string{"STATUS"}},
		},
	}
	uuidMap := map[string]string{
		"f1": "local.myapp_title_field_id",
		"f2": "local.myapp_status_field_id",
	}

	g := NewAppHCLGenerator(app, nil, uuidMap)

	block := &discovery.DisplayBlock{
		ID:              "block-1",
		Type:            "group",
		Name:            "System",
		FieldIDs:        []string{"f1", "f2"},
		DisplayOrder:    0,
		DisplayLocation: "CENTER",
	}

	section := g.buildSection(block)
	hcl := renderHCLValue(section)

	if strings.Contains(hcl, "field_ids") {
		t.Error("field_ids should be omitted when all fields are system fields")
	}
}

// TestBuildSection_ComponentTypes verifies that non-group section types
// (status, activity_log, etc.) produce correct HCL without field_ids.
func TestBuildSection_ComponentTypes(t *testing.T) {
	t.Parallel()

	app := &discovery.App{ID: "app-123"}
	g := NewAppHCLGenerator(app, nil, nil)

	tests := []struct {
		blockType string
	}{
		{"status"},
		{"activity_log"},
		{"approvals"},
		{"attachments"},
	}

	for _, tt := range tests {
		t.Run(tt.blockType, func(t *testing.T) {
			block := &discovery.DisplayBlock{
				ID:              "block-1",
				Type:            tt.blockType,
				DisplayOrder:    0,
				DisplayLocation: "CENTER",
			}

			section := g.buildSection(block)
			hcl := renderHCLValue(section)

			if !strings.Contains(hcl, `type = "`+tt.blockType+`"`) {
				t.Errorf("expected type = %q in output", tt.blockType)
			}
			// Component sections should NOT have field_ids
			if strings.Contains(hcl, "field_ids") {
				t.Error("component sections should not have field_ids")
			}
		})
	}
}

// renderHCLValue is a test helper that renders an HCLValue to string for assertions.
func renderHCLValue(v HCLValue) string {
	var sb strings.Builder
	serializeValue(&sb, v, 0)
	return sb.String()
}
