// Copyright (c) ElementumAI, Inc.
// SPDX-License-Identifier: MPL-2.0

package export

import (
	"testing"
)

func TestBuildImportID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		resourceType string
		params       map[string]string
		want         string
	}{
		{
			name:         "app",
			resourceType: "elementum_app",
			params:       map[string]string{"app_id": "app_123"},
			want:         "app_123",
		},
		{
			name:         "element",
			resourceType: "elementum_element",
			params:       map[string]string{"element_id": "elem_456"},
			want:         "elem_456",
		},
		{
			name:         "group",
			resourceType: "elementum_group",
			params:       map[string]string{"group_id": "grp_789"},
			want:         "grp_789",
		},
		{
			name:         "cloudlink",
			resourceType: "elementum_cloudlink",
			params:       map[string]string{"cloudlink_id": "cl_abc"},
			want:         "cl_abc",
		},
		{
			name:         "text_field",
			resourceType: "elementum_text_field",
			params:       map[string]string{"app_id": "app_123", "field_id": "field_456"},
			want:         "app_123:field_456",
		},
		{
			name:         "ai_file_reader",
			resourceType: "elementum_ai_file_reader",
			params:       map[string]string{"app_id": "app_123", "file_reader_id": "reader_789"},
			want:         "app_123:reader_789",
		},
		{
			name:         "approval_process",
			resourceType: "elementum_approval_process",
			params:       map[string]string{"app_id": "app_123", "approval_id": "approval_xyz"},
			want:         "app_123:approval_xyz",
		},
		{
			name:         "agent",
			resourceType: "elementum_agent",
			params:       map[string]string{"app_id": "app_123", "agent_id": "agent_456"},
			want:         "app_123:agent_456",
		},
		{
			name:         "agent_create_record_tool",
			resourceType: "elementum_agent_create_record_tool",
			params:       map[string]string{"app_id": "app_123", "agent_id": "agent_456", "tool_id": "tool_789"},
			want:         "app_123:agent_456:tool_789",
		},
		{
			name:         "layout",
			resourceType: "elementum_layout",
			params:       map[string]string{"app_id": "app_123", "layout_id": "layout_open"},
			want:         "app_123:layout_open",
		},
		{
			name:         "widget",
			resourceType: "elementum_widget",
			params:       map[string]string{"app_id": "app_123", "widget_id": "widget_dashboard"},
			want:         "app_123:widget_dashboard",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BuildImportID(tt.resourceType, tt.params)
			if got != tt.want {
				t.Errorf("BuildImportID(%q, %v) = %q, want %q", tt.resourceType, tt.params, got, tt.want)
			}
		})
	}
}

func TestBuildFieldImportID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		appID     string
		fieldID   string
		fieldType string
		want      string
	}{
		{
			name:      "text field",
			appID:     "app_123",
			fieldID:   "field_456",
			fieldType: "text",
			want:      "app_123:field_456",
		},
		{
			name:      "dropdown field",
			appID:     "app_789",
			fieldID:   "field_abc",
			fieldType: "dropdown",
			want:      "app_789:field_abc",
		},
		{
			name:      "number field",
			appID:     "app_xyz",
			fieldID:   "field_def",
			fieldType: "number",
			want:      "app_xyz:field_def",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BuildFieldImportID(tt.appID, tt.fieldID, tt.fieldType)
			if got != tt.want {
				t.Errorf("BuildFieldImportID(%q, %q, %q) = %q, want %q", tt.appID, tt.fieldID, tt.fieldType, got, tt.want)
			}
		})
	}
}

func TestImportIDFormats_Coverage(t *testing.T) {
	t.Parallel()

	// Verify all expected resource types have import ID formats defined
	expectedFormats := []string{
		"elementum_app",
		"elementum_element",
		"elementum_group",
		"elementum_cloudlink",
		"elementum_ai_file_reader",
		"elementum_approval_process",
		"elementum_text_field",
		"elementum_number_field",
		"elementum_dropdown_field",
		"elementum_layout",
		"elementum_flow",
		"elementum_automation",
		"elementum_agent",
		"elementum_agent_create_record_tool",
		"elementum_agent_search_records_tool",
		"elementum_agent_update_record_tool",
		"elementum_agent_ai_search_tool",
		"elementum_agent_relate_record_tool",
		"elementum_agent_run_automation_tool",
		"elementum_agent_run_agent_tool",
		"elementum_agent_select_bot_route_tool",
		"elementum_agent_mcp_tool",
		"elementum_widget",
	}

	for _, resourceType := range expectedFormats {
		if _, exists := ImportIDFormats[resourceType]; !exists {
			t.Errorf("ImportIDFormats missing entry for %q", resourceType)
		}
	}
}

func TestImportIDFormats_NewResources(t *testing.T) {
	t.Parallel()

	// Specifically test that new resources we added have correct formats
	tests := []struct {
		resourceType string
		wantFormat   string
	}{
		{
			resourceType: "elementum_table_search_table",
			wantFormat:   "{table_id}/{search_table_id}",
		},
		{
			resourceType: "elementum_cloudlink",
			wantFormat:   "{cloudlink_id}",
		},
		{
			resourceType: "elementum_ai_file_reader",
			wantFormat:   "{app_id}:{file_reader_id}",
		},
		{
			resourceType: "elementum_text_file_reader",
			wantFormat:   "{app_id}:{file_reader_id}",
		},
		{
			resourceType: "elementum_json_file_reader",
			wantFormat:   "{app_id}:{file_reader_id}",
		},
		{
			resourceType: "elementum_xml_file_reader",
			wantFormat:   "{app_id}:{file_reader_id}",
		},
		{
			resourceType: "elementum_approval_process",
			wantFormat:   "{app_id}:{approval_id}",
		},
		{
			resourceType: "elementum_element",
			wantFormat:   "{element_id}",
		},
		{
			resourceType: "elementum_group",
			wantFormat:   "{group_id}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.resourceType, func(t *testing.T) {
			format, exists := ImportIDFormats[tt.resourceType]
			if !exists {
				t.Fatalf("ImportIDFormats missing entry for %q", tt.resourceType)
			}
			if format != tt.wantFormat {
				t.Errorf("ImportIDFormats[%q] = %q, want %q", tt.resourceType, format, tt.wantFormat)
			}
		})
	}
}

func TestBuildImportID_UnknownType(t *testing.T) {
	t.Parallel()

	// When resource type is not in the map, it should fallback to just the ID
	params := map[string]string{"id": "fallback_123"}
	got := BuildImportID("unknown_resource_type", params)

	if got != "fallback_123" {
		t.Errorf("BuildImportID with unknown type should fallback to id, got %q", got)
	}
}
