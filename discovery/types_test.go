// Copyright (c) ElementumAI, Inc.
// SPDX-License-Identifier: MPL-2.0

package discovery

import (
	"strings"
	"testing"
)

func TestApp_GetUUIDMappings(t *testing.T) {
	app := &App{
		ID:   "app-uuid-123",
		Name: "Test App",
	}

	mappings := app.GetUUIDMappings("my_app")

	tests := []struct {
		uuid     string
		expected string
	}{
		{"app-uuid-123", "elementum_app.my_app.id"},
		{"app-uuid-123.title_field_id", "elementum_app.my_app.title_field_id"},
		{"app-uuid-123.status_field_id", "elementum_app.my_app.status_field_id"},
	}

	for _, tt := range tests {
		if got := mappings[tt.uuid]; got != tt.expected {
			t.Errorf("App.GetUUIDMappings()[%q] = %q, want %q", tt.uuid, got, tt.expected)
		}
	}
}

func TestField_GetUUIDMappings(t *testing.T) {
	tests := []struct {
		name         string
		field        Field
		resourceName string
		expectedKey  string
		expectedVal  string
	}{
		{
			name:         "text field",
			field:        Field{ID: "field-123", Type: "text"},
			resourceName: "my_field",
			expectedKey:  "field-123",
			expectedVal:  "elementum_text_field.my_field.id",
		},
		{
			name:         "boolean field",
			field:        Field{ID: "field-456", Type: "boolean"},
			resourceName: "enabled",
			expectedKey:  "field-456",
			expectedVal:  "elementum_boolean_field.enabled.id",
		},
		{
			name:         "dropdown field",
			field:        Field{ID: "field-789", Type: "dropdown"},
			resourceName: "priority",
			expectedKey:  "field-789",
			expectedVal:  "elementum_dropdown_field.priority.id",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mappings := tt.field.GetUUIDMappings(tt.resourceName)
			if got := mappings[tt.expectedKey]; got != tt.expectedVal {
				t.Errorf("Field.GetUUIDMappings()[%q] = %q, want %q", tt.expectedKey, got, tt.expectedVal)
			}
		})
	}
}

func TestAutomation_GetUUIDMappings(t *testing.T) {
	automation := &Automation{
		ID:         "auto-uuid-123",
		Name:       "Test Automation",
		WorkflowID: "workflow-uuid-456",
	}

	mappings := automation.GetUUIDMappings("my_automation")

	tests := []struct {
		uuid     string
		expected string
	}{
		{"auto-uuid-123", "elementum_automation.my_automation.id"},
		{"workflow-uuid-456", "elementum_automation.my_automation.workflow_id"},
	}

	for _, tt := range tests {
		if got := mappings[tt.uuid]; got != tt.expected {
			t.Errorf("Automation.GetUUIDMappings()[%q] = %q, want %q", tt.uuid, got, tt.expected)
		}
	}
}

func TestTrigger_GetUUIDMappings(t *testing.T) {
	tests := []struct {
		name         string
		trigger      Trigger
		resourceName string
		wantMappings map[string]string
	}{
		{
			name:         "record_created trigger",
			trigger:      Trigger{ID: "trigger-123", Type: "record_created"},
			resourceName: "on_create",
			wantMappings: map[string]string{
				"trigger-123": "elementum_record_created_trigger.on_create.id",
				"trigger.record.10000001-2000-4000-a000-800000000000": "elementum_record_created_trigger.on_create.record_id",
				"trigger.record.10000001-2000-4000-a000-800000000005": "elementum_record_created_trigger.on_create.record_url",
				"trigger.record.10000001-2000-4000-a000-800000000003": "elementum_record_created_trigger.on_create.attachments",
			},
		},
		{
			name:         "record_updated trigger",
			trigger:      Trigger{ID: "trigger-456", Type: "record_updated"},
			resourceName: "on_update",
			wantMappings: map[string]string{
				"trigger-456": "elementum_record_updated_trigger.on_update.id",
				"trigger.record.10000001-2000-4000-a000-800000000000": "elementum_record_updated_trigger.on_update.record_id",
				"trigger.record.10000001-2000-4000-a000-800000000005": "elementum_record_updated_trigger.on_update.record_url",
				"trigger.record.10000001-2000-4000-a000-800000000003": "elementum_record_updated_trigger.on_update.attachments",
			},
		},
		{
			name:         "webhook trigger",
			trigger:      Trigger{ID: "trigger-789", Type: "webhook"},
			resourceName: "webhook_handler",
			wantMappings: map[string]string{
				"trigger-789": "elementum_webhook_trigger.webhook_handler.id",
			},
		},
		{
			name:         "datamine trigger",
			trigger:      Trigger{ID: "trigger-dm-123", Type: "datamine", DatamineID: "dm-456"},
			resourceName: "stale_check",
			// NOTE: datamine_id is intentionally NOT mapped to avoid self-references
			// The datamine should be exported separately and referenced as elementum_datamine.<name>.id
			wantMappings: map[string]string{
				"trigger-dm-123": "elementum_datamine_trigger.stale_check.id",
			},
		},
		{
			name:         "datamine trigger without datamine_id",
			trigger:      Trigger{ID: "trigger-dm-789", Type: "datamine"},
			resourceName: "alert",
			wantMappings: map[string]string{
				"trigger-dm-789": "elementum_datamine_trigger.alert.id",
			},
		},
		{
			name: "on_demand trigger with parameters",
			trigger: Trigger{
				ID:   "trigger-od-123",
				Type: "on_demand",
				RawData: map[string]interface{}{
					"id":         "trigger-od-123",
					"__typename": "WorkflowTriggerOnDemand",
					"parameters": []interface{}{
						map[string]interface{}{
							"id":        "param-uuid-1",
							"name":      "requester_email",
							"fieldType": "TEXT",
							"required":  true,
						},
						map[string]interface{}{
							"id":        "param-uuid-2",
							"name":      "ticket_id",
							"fieldType": "TEXT",
							"required":  true,
						},
						map[string]interface{}{
							"id":        "param-uuid-3",
							"name":      "description",
							"fieldType": "TEXT",
							"required":  false,
						},
					},
				},
			},
			resourceName: "manual_trigger",
			wantMappings: map[string]string{
				"trigger-od-123": "elementum_on_demand_trigger.manual_trigger.id",
				"param-uuid-1":   `elementum_on_demand_trigger.manual_trigger.parameter_ids["requester_email"]`,
				"param-uuid-2":   `elementum_on_demand_trigger.manual_trigger.parameter_ids["ticket_id"]`,
				"param-uuid-3":   `elementum_on_demand_trigger.manual_trigger.parameter_ids["description"]`,
			},
		},
		{
			name: "on_demand trigger without parameters",
			trigger: Trigger{
				ID:   "trigger-od-456",
				Type: "on_demand",
				RawData: map[string]interface{}{
					"id":         "trigger-od-456",
					"__typename": "WorkflowTriggerOnDemand",
					"parameters": []interface{}{},
				},
			},
			resourceName: "simple_trigger",
			wantMappings: map[string]string{
				"trigger-od-456": "elementum_on_demand_trigger.simple_trigger.id",
			},
		},
		{
			name: "on_demand trigger with nil RawData",
			trigger: Trigger{
				ID:      "trigger-od-789",
				Type:    "on_demand",
				RawData: nil,
			},
			resourceName: "no_data_trigger",
			wantMappings: map[string]string{
				"trigger-od-789": "elementum_on_demand_trigger.no_data_trigger.id",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mappings := tt.trigger.GetUUIDMappings(tt.resourceName)

			for uuid, expectedRef := range tt.wantMappings {
				if got := mappings[uuid]; got != expectedRef {
					t.Errorf("Trigger.GetUUIDMappings()[%q] = %q, want %q", uuid, got, expectedRef)
				}
			}

			// Check that we got the expected number of mappings
			if len(mappings) != len(tt.wantMappings) {
				t.Errorf("Trigger.GetUUIDMappings() returned %d mappings, want %d", len(mappings), len(tt.wantMappings))
			}
		})
	}
}

func TestTask_GetUUIDMappings(t *testing.T) {
	tests := []struct {
		name         string
		task         Task
		resourceName string
		wantMappings map[string]string
	}{
		{
			name:         "record_search task",
			task:         Task{ID: "task-123", Type: "record_search"},
			resourceName: "find_records",
			wantMappings: map[string]string{
				"task-123":                 "elementum_record_search_task.find_records.id",
				"task-123.first_record_id": "elementum_record_search_task.find_records.first_record_id",
				"task-123.record_count":    "elementum_record_search_task.find_records.record_count",
			},
		},
		{
			name:         "message task",
			task:         Task{ID: "task-456", Type: "message"},
			resourceName: "send_notification",
			wantMappings: map[string]string{
				"task-456": "elementum_message_task.send_notification.id",
			},
		},
		{
			name:         "update_field task",
			task:         Task{ID: "task-789", Type: "update_field"},
			resourceName: "update_status",
			wantMappings: map[string]string{
				"task-789": "elementum_update_field_task.update_status.id",
			},
		},
		{
			name:         "create_record task",
			task:         Task{ID: "task-abc", Type: "create_record"},
			resourceName: "create_ticket",
			wantMappings: map[string]string{
				"task-abc": "elementum_create_record_task.create_ticket.id",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mappings := tt.task.GetUUIDMappings(tt.resourceName)

			for uuid, expectedRef := range tt.wantMappings {
				if got := mappings[uuid]; got != expectedRef {
					t.Errorf("Task.GetUUIDMappings()[%q] = %q, want %q", uuid, got, expectedRef)
				}
			}

			// Check that we got the expected number of mappings
			if len(mappings) != len(tt.wantMappings) {
				t.Errorf("Task.GetUUIDMappings() returned %d mappings, want %d", len(mappings), len(tt.wantMappings))
			}
		})
	}
}

func TestLayout_GetUUIDMappings(t *testing.T) {
	layout := &Layout{
		ID:   "layout-123",
		Name: "Active Stage",
	}

	mappings := layout.GetUUIDMappings("active")

	expected := "elementum_layout.active.id"
	if got := mappings["layout-123"]; got != expected {
		t.Errorf("Layout.GetUUIDMappings()[layout-123] = %q, want %q", got, expected)
	}
}

func TestFlow_GetUUIDMappings(t *testing.T) {
	flow := &Flow{
		ID:   "flow-123",
		Name: "Main Flow",
	}

	mappings := flow.GetUUIDMappings("main_flow")

	expected := "elementum_flow.main_flow.id"
	if got := mappings["flow-123"]; got != expected {
		t.Errorf("Flow.GetUUIDMappings()[flow-123] = %q, want %q", got, expected)
	}
}

func TestAgent_GetUUIDMappings(t *testing.T) {
	agent := &Agent{
		ID:   "agent-123",
		Name: "Support Agent",
	}

	mappings := agent.GetUUIDMappings("support_agent")

	expected := "elementum_agent.support_agent.id"
	if got := mappings["agent-123"]; got != expected {
		t.Errorf("Agent.GetUUIDMappings()[agent-123] = %q, want %q", got, expected)
	}
}

func TestAgentTool_GetUUIDMappings(t *testing.T) {
	tool := &AgentTool{
		ID:   "tool-123",
		Name: "Search Tool",
		Type: "AgentSearchAspectTool",
	}

	mappings := tool.GetUUIDMappings("search_tool")

	expected := "elementum_agent_search_records_tool.search_tool.id"
	if got := mappings["tool-123"]; got != expected {
		t.Errorf("AgentTool.GetUUIDMappings()[tool-123] = %q, want %q", got, expected)
	}
}

func TestAgentTool_TerraformResourceType(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		graphQLType  string
		expectedType string
	}{
		{
			name:         "CreateRecordTool",
			graphQLType:  "AgentCreateRecordTool",
			expectedType: "elementum_agent_create_record_tool",
		},
		{
			name:         "SearchAspectTool",
			graphQLType:  "AgentSearchAspectTool",
			expectedType: "elementum_agent_search_records_tool",
		},
		{
			name:         "UpdateRecordTool",
			graphQLType:  "AgentUpdateRecordTool",
			expectedType: "elementum_agent_update_record_tool",
		},
		{
			name:         "SearchTableTool (AI Search)",
			graphQLType:  "AgentSearchTableTool",
			expectedType: "elementum_agent_ai_search_tool",
		},
		{
			name:         "RelateRecordTool",
			graphQLType:  "AgentRelateRecordTool",
			expectedType: "elementum_agent_relate_record_tool",
		},
		{
			name:         "ExecuteWorkflowTool (Run Automation)",
			graphQLType:  "AgentExecuteWorkflowTool",
			expectedType: "elementum_agent_run_automation_tool",
		},
		{
			name:         "RunAgentTool",
			graphQLType:  "AgentRunAgentTool",
			expectedType: "elementum_agent_run_agent_tool",
		},
		{
			name:         "SelectBotRouteTool",
			graphQLType:  "AgentSelectBotRouteTool",
			expectedType: "elementum_agent_select_bot_route_tool",
		},
		{
			name:         "McpTool",
			graphQLType:  "AgentMcpTool",
			expectedType: "elementum_agent_mcp_tool",
		},
		{
			name:         "Unknown type falls back to create_record_tool",
			graphQLType:  "UnknownToolType",
			expectedType: "elementum_agent_create_record_tool",
		},
		{
			name:         "Empty type falls back to create_record_tool",
			graphQLType:  "",
			expectedType: "elementum_agent_create_record_tool",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tool := &AgentTool{
				ID:   "test-tool-id",
				Name: "Test Tool",
				Type: tt.graphQLType,
			}

			got := tool.TerraformResourceType()
			if got != tt.expectedType {
				t.Errorf("TerraformResourceType() = %q, want %q", got, tt.expectedType)
			}
		})
	}
}

func TestAgentTool_TerraformResourceType_AllTypesCount(t *testing.T) {
	t.Parallel()

	// Verify we have exactly 9 known tool types
	knownTypes := []string{
		"AgentCreateRecordTool",
		"AgentSearchAspectTool",
		"AgentUpdateRecordTool",
		"AgentSearchTableTool",
		"AgentRelateRecordTool",
		"AgentExecuteWorkflowTool",
		"AgentRunAgentTool",
		"AgentSelectBotRouteTool",
		"AgentMcpTool",
	}

	if len(knownTypes) != 9 {
		t.Errorf("Expected 9 known tool types, got %d", len(knownTypes))
	}

	// Verify each known type maps to a unique Terraform resource type
	resourceTypes := make(map[string]bool)
	for _, graphQLType := range knownTypes {
		tool := &AgentTool{Type: graphQLType}
		resourceType := tool.TerraformResourceType()

		if resourceTypes[resourceType] {
			t.Errorf("Duplicate resource type %q for GraphQL type %q", resourceType, graphQLType)
		}
		resourceTypes[resourceType] = true

		// Verify it doesn't return the fallback (unless unknown)
		if resourceType == "elementum_agent_create_record_tool" && graphQLType != "AgentCreateRecordTool" {
			t.Errorf("GraphQL type %q returned fallback resource type", graphQLType)
		}
	}
}

func TestAgentTool_SearchTableFields(t *testing.T) {
	t.Parallel()

	// Test AgentTool with SearchTable fields populated
	tool := &AgentTool{
		ID:                    "tool-123",
		Name:                  "AI Search Tool",
		Type:                  "AgentSearchTableTool",
		SearchTableID:         "search-table-456",
		SearchTableAspectID:   "element-789",
		SearchTableAspectType: "Element",
	}

	// Verify SearchTable fields are populated
	if tool.SearchTableID != "search-table-456" {
		t.Errorf("AgentTool.SearchTableID = %q, want %q", tool.SearchTableID, "search-table-456")
	}
	if tool.SearchTableAspectID != "element-789" {
		t.Errorf("AgentTool.SearchTableAspectID = %q, want %q", tool.SearchTableAspectID, "element-789")
	}
	if tool.SearchTableAspectType != "Element" {
		t.Errorf("AgentTool.SearchTableAspectType = %q, want %q", tool.SearchTableAspectType, "Element")
	}

	// Verify resource type mapping
	resourceType := tool.TerraformResourceType()
	if resourceType != "elementum_agent_ai_search_tool" {
		t.Errorf("AgentTool.TerraformResourceType() = %q, want %q", resourceType, "elementum_agent_ai_search_tool")
	}

	// Verify UUID mappings
	mappings := tool.GetUUIDMappings("ai_search_tool")
	expected := "elementum_agent_ai_search_tool.ai_search_tool.id"
	if got := mappings["tool-123"]; got != expected {
		t.Errorf("AgentTool.GetUUIDMappings()[tool-123] = %q, want %q", got, expected)
	}
}

func TestAgentTool_SearchTableAspectTypes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		aspectType string
		valid      bool
	}{
		{"Element type", "Element", true},
		{"App type", "App", true}, // Apps don't have search tables but handle it
		{"Empty type", "", true},  // Default to Element
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tool := &AgentTool{
				ID:                    "tool-123",
				Type:                  "AgentSearchTableTool",
				SearchTableID:         "st-456",
				SearchTableAspectID:   "aspect-789",
				SearchTableAspectType: tt.aspectType,
			}

			// Verify type is stored correctly
			if tool.SearchTableAspectType != tt.aspectType {
				t.Errorf("SearchTableAspectType = %q, want %q", tool.SearchTableAspectType, tt.aspectType)
			}
		})
	}
}

func TestElement_WithAISearchTables(t *testing.T) {
	t.Parallel()

	element := &Element{
		ID:        "elem-123",
		Name:      "Test Element",
		Namespace: "test_element",
		AISearchTables: []AISearchTable{
			{
				ID:                      "st-1",
				ObjectID:                "elem-123",
				FieldID:                 "field-desc",
				FieldName:               "Description",
				AIProviderConnectorID:   "connector-1",
				AIProviderConnectorName: "text-embedding-3-small",
				Duration:                "DAYS",
				TargetLag:               1,
			},
			{
				ID:                      "st-2",
				ObjectID:                "elem-123",
				FieldID:                 "field-notes",
				FieldName:               "Notes",
				AIProviderConnectorID:   "connector-1",
				AIProviderConnectorName: "text-embedding-3-small",
				Duration:                "HOURS",
				TargetLag:               6,
			},
		},
	}

	// Verify AISearchTables are populated
	if len(element.AISearchTables) != 2 {
		t.Errorf("Element.AISearchTables length = %d, want 2", len(element.AISearchTables))
	}

	// Verify first search table
	if element.AISearchTables[0].ID != "st-1" {
		t.Errorf("AISearchTable[0].ID = %q, want %q", element.AISearchTables[0].ID, "st-1")
	}
	if element.AISearchTables[0].FieldName != "Description" {
		t.Errorf("AISearchTable[0].FieldName = %q, want %q", element.AISearchTables[0].FieldName, "Description")
	}

	// Verify second search table
	if element.AISearchTables[1].Duration != "HOURS" {
		t.Errorf("AISearchTable[1].Duration = %q, want %q", element.AISearchTables[1].Duration, "HOURS")
	}
	if element.AISearchTables[1].TargetLag != 6 {
		t.Errorf("AISearchTable[1].TargetLag = %d, want 6", element.AISearchTables[1].TargetLag)
	}
}

func TestAISearchTable_GetUUIDMappings(t *testing.T) {
	t.Parallel()

	searchTable := &AISearchTable{
		ID:        "st-123",
		ObjectID:  "elem-456",
		FieldID:   "field-789",
		FieldName: "Description",
	}

	mappings := searchTable.GetUUIDMappings("description_search")

	// Verify search table ID mapping
	expected := "elementum_ai_search_table.description_search.id"
	if got := mappings["st-123"]; got != expected {
		t.Errorf("AISearchTable.GetUUIDMappings()[st-123] = %q, want %q", got, expected)
	}
}

func TestTableSearchTable_GetUUIDMappings(t *testing.T) {
	t.Parallel()

	searchTable := &TableSearchTable{
		ID:                    "st-123",
		TableID:               "table-456",
		FieldID:               "field-789",
		FieldName:             "Description",
		AIProviderConnectorID: "connector-abc",
		AttributeFieldIDs:     []string{"attr-1", "attr-2"},
		Duration:              "DAYS",
		TargetLag:             1,
		Warehouse:             "COMPUTE_WH",
	}

	mappings := searchTable.GetUUIDMappings("description_search")

	// Verify search table ID mapping
	expected := "elementum_table_search_table.description_search.id"
	if got := mappings["st-123"]; got != expected {
		t.Errorf("TableSearchTable.GetUUIDMappings()[st-123] = %q, want %q", got, expected)
	}
}

func TestTableSearchTable_BeautifiableInterface(t *testing.T) {
	t.Parallel()

	// Verify TableSearchTable implements Beautifiable
	var _ Beautifiable = &TableSearchTable{}
}

func TestTable_WithSearchTables(t *testing.T) {
	t.Parallel()

	table := &Table{
		ID:   "table-123",
		Name: "Customer Data",
		SearchTables: []TableSearchTable{
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
				FieldID:               "field-notes",
				FieldName:             "Notes",
				AIProviderConnectorID: "connector-1",
				Duration:              "HOURS",
				TargetLag:             6,
			},
		},
	}

	// Verify SearchTables are populated
	if len(table.SearchTables) != 2 {
		t.Errorf("Table.SearchTables length = %d, want 2", len(table.SearchTables))
	}

	// Verify first search table
	if table.SearchTables[0].ID != "st-1" {
		t.Errorf("SearchTable[0].ID = %q, want %q", table.SearchTables[0].ID, "st-1")
	}
	if table.SearchTables[0].FieldName != "Description" {
		t.Errorf("SearchTable[0].FieldName = %q, want %q", table.SearchTables[0].FieldName, "Description")
	}

	// Verify second search table
	if table.SearchTables[1].Duration != "HOURS" {
		t.Errorf("SearchTable[1].Duration = %q, want %q", table.SearchTables[1].Duration, "HOURS")
	}
	if table.SearchTables[1].TargetLag != 6 {
		t.Errorf("SearchTable[1].TargetLag = %d, want 6", table.SearchTables[1].TargetLag)
	}
}

func TestTableSearchTable_AllDurationTypes(t *testing.T) {
	t.Parallel()

	durations := []string{"DAYS", "HOURS", "MINUTES", "SECONDS"}

	for _, duration := range durations {
		t.Run(duration, func(t *testing.T) {
			st := &TableSearchTable{
				ID:       "st-" + duration,
				Duration: duration,
			}
			if st.Duration != duration {
				t.Errorf("TableSearchTable.Duration = %q, want %q", st.Duration, duration)
			}
		})
	}
}

func TestWidget_GetUUIDMappings(t *testing.T) {
	tests := []struct {
		name         string
		widget       Widget
		resourceName string
		expectedKey  string
		expectedVal  string
	}{
		{
			name:         "related_object widget",
			widget:       Widget{ID: "widget-123", Name: "Related Tasks", Type: "DisplayWidgetRelatedAspect"},
			resourceName: "related_tasks",
			expectedKey:  "widget-123",
			expectedVal:  "elementum_widget.related_tasks.id",
		},
		{
			name:         "related_link_action widget",
			widget:       Widget{ID: "widget-456", Name: "Link Tickets", Type: "DisplayWidgetRelatedLinkAction"},
			resourceName: "link_tickets",
			expectedKey:  "widget-456",
			expectedVal:  "elementum_widget.link_tickets.id",
		},
		{
			name:         "related_create_action widget",
			widget:       Widget{ID: "widget-789", Name: "Create Task", Type: "DisplayWidgetRelatedCreateAction"},
			resourceName: "create_task",
			expectedKey:  "widget-789",
			expectedVal:  "elementum_widget.create_task.id",
		},
		{
			name:         "widget without type (legacy)",
			widget:       Widget{ID: "widget-old", Name: "Dashboard Widget"},
			resourceName: "dashboard",
			expectedKey:  "widget-old",
			expectedVal:  "elementum_widget.dashboard.id",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mappings := tt.widget.GetUUIDMappings(tt.resourceName)
			if got := mappings[tt.expectedKey]; got != tt.expectedVal {
				t.Errorf("Widget.GetUUIDMappings()[%q] = %q, want %q", tt.expectedKey, got, tt.expectedVal)
			}
		})
	}
}

func TestWidget_TypeField(t *testing.T) {
	tests := []struct {
		name         string
		widgetType   string
		expectedType string
	}{
		{"related aspect", "DisplayWidgetRelatedAspect", "DisplayWidgetRelatedAspect"},
		{"related link action", "DisplayWidgetRelatedLinkAction", "DisplayWidgetRelatedLinkAction"},
		{"related create action", "DisplayWidgetRelatedCreateAction", "DisplayWidgetRelatedCreateAction"},
		{"empty type", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			widget := Widget{
				ID:   "widget-123",
				Name: "Test Widget",
				Type: tt.widgetType,
			}

			if widget.Type != tt.expectedType {
				t.Errorf("Widget.Type = %q, want %q", widget.Type, tt.expectedType)
			}
		})
	}
}

func TestFileReader_GetUUIDMappings(t *testing.T) {
	tests := []struct {
		name         string
		readerType   string
		resourceName string
		expected     string
	}{
		{
			name:         "AI file reader",
			readerType:   FileReaderTypeAI,
			resourceName: "invoice_reader",
			expected:     "elementum_ai_file_reader.invoice_reader.id",
		},
		{
			name:         "Text/OCR file reader",
			readerType:   FileReaderTypeOCR,
			resourceName: "document_scanner",
			expected:     "elementum_text_file_reader.document_scanner.id",
		},
		{
			name:         "JSON file reader",
			readerType:   FileReaderTypeJSON,
			resourceName: "order_parser",
			expected:     "elementum_json_file_reader.order_parser.id",
		},
		{
			name:         "XML file reader",
			readerType:   FileReaderTypeXML,
			resourceName: "invoice_parser",
			expected:     "elementum_xml_file_reader.invoice_parser.id",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := &FileReader{
				ID:   "reader-123",
				Name: "Test Reader",
				Type: tt.readerType,
			}

			mappings := reader.GetUUIDMappings(tt.resourceName)

			if got := mappings["reader-123"]; got != tt.expected {
				t.Errorf("FileReader.GetUUIDMappings() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestFileReader_TerraformResourceType(t *testing.T) {
	tests := []struct {
		name       string
		readerType string
		expected   string
	}{
		{
			name:       "AI type",
			readerType: FileReaderTypeAI,
			expected:   "elementum_ai_file_reader",
		},
		{
			name:       "OCR type",
			readerType: FileReaderTypeOCR,
			expected:   "elementum_text_file_reader",
		},
		{
			name:       "JSON type",
			readerType: FileReaderTypeJSON,
			expected:   "elementum_json_file_reader",
		},
		{
			name:       "XML type",
			readerType: FileReaderTypeXML,
			expected:   "elementum_xml_file_reader",
		},
		{
			name:       "Unknown type defaults to AI",
			readerType: "UnknownType",
			expected:   "elementum_ai_file_reader",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := &FileReader{
				ID:   "reader-123",
				Type: tt.readerType,
			}

			if got := reader.TerraformResourceType(); got != tt.expected {
				t.Errorf("FileReader.TerraformResourceType() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestFileReaderTypeConstants(t *testing.T) {
	// Verify the constants match what GraphQL returns
	if FileReaderTypeAI != "DocumentModelAi" {
		t.Errorf("FileReaderTypeAI = %q, want DocumentModelAi", FileReaderTypeAI)
	}
	if FileReaderTypeOCR != "DocumentModelOCR" {
		t.Errorf("FileReaderTypeOCR = %q, want DocumentModelOCR", FileReaderTypeOCR)
	}
	if FileReaderTypeJSON != "DocumentModelJson" {
		t.Errorf("FileReaderTypeJSON = %q, want DocumentModelJson", FileReaderTypeJSON)
	}
	if FileReaderTypeXML != "DocumentModelXml" {
		t.Errorf("FileReaderTypeXML = %q, want DocumentModelXml", FileReaderTypeXML)
	}
}

// Legacy alias test for backward compatibility
func TestAIFileReader_Alias(t *testing.T) {
	// AIFileReader should be an alias for FileReader
	reader := &AIFileReader{
		ID:   "reader-123",
		Name: "PDF Reader",
		Type: FileReaderTypeAI,
	}

	mappings := reader.GetUUIDMappings("pdf_reader")

	expected := "elementum_ai_file_reader.pdf_reader.id"
	if got := mappings["reader-123"]; got != expected {
		t.Errorf("AIFileReader.GetUUIDMappings()[reader-123] = %q, want %q", got, expected)
	}
}

func TestApprovalProcess_GetUUIDMappings(t *testing.T) {
	approval := &ApprovalProcess{
		ID:   "approval-123",
		Name: "Manager Approval",
	}

	mappings := approval.GetUUIDMappings("manager_approval")

	expected := "elementum_approval_process.manager_approval.id"
	if got := mappings["approval-123"]; got != expected {
		t.Errorf("ApprovalProcess.GetUUIDMappings()[approval-123] = %q, want %q", got, expected)
	}
}

func TestTrigger_GetUUIDMappings_WithFieldRefs(t *testing.T) {
	t.Parallel()

	trigger := Trigger{
		ID:   "trigger-123",
		Type: "record_created",
		FieldRefs: map[string]string{
			"record.field-uuid-456": "Status",
			"field-uuid-789":        "Priority",
		},
	}

	mappings := trigger.GetUUIDMappings("my_trigger")

	// FieldRefs should produce UNQUOTED terraform expressions
	// NOT quoted strings like "${...refs[\"...\"]}"
	tests := []struct {
		key      string
		expected string
	}{
		// Correct: unquoted refs expression
		{"trigger.record.field-uuid-456", `elementum_record_created_trigger.my_trigger.refs["Status"]`},
		{"trigger.record.field-uuid-789", `elementum_record_created_trigger.my_trigger.refs["Priority"]`},
	}

	for _, tt := range tests {
		got := mappings[tt.key]
		if got != tt.expected {
			t.Errorf("GetUUIDMappings()[%q] = %q, want %q", tt.key, got, tt.expected)
		}
		// Also verify NO escaped quotes or ${} wrapper
		if strings.Contains(got, `\"`) {
			t.Errorf("GetUUIDMappings()[%q] should not contain escaped quotes, got %q", tt.key, got)
		}
		if strings.Contains(got, "${") {
			t.Errorf("GetUUIDMappings()[%q] should not contain ${} wrapper, got %q", tt.key, got)
		}
	}
}

func TestTask_GetUUIDMappings_WithFieldRefs(t *testing.T) {
	t.Parallel()

	task := Task{
		ID:   "task-123",
		Type: "variable",
		FieldRefs: map[string]string{
			"my_variable": "Score",
			"other_var":   "Result",
		},
	}

	mappings := task.GetUUIDMappings("calc_task")

	// FieldRefs should produce UNQUOTED terraform expressions
	// NOT quoted strings like "${...refs[\"...\"]}"
	tests := []struct {
		key      string
		expected string
	}{
		// Correct: unquoted refs expression
		{"task.task-123.my_variable", `elementum_variable_task.calc_task.refs["Score"]`},
		{"task.task-123.other_var", `elementum_variable_task.calc_task.refs["Result"]`},
	}

	for _, tt := range tests {
		got := mappings[tt.key]
		if got != tt.expected {
			t.Errorf("GetUUIDMappings()[%q] = %q, want %q", tt.key, got, tt.expected)
		}
		// Also verify NO escaped quotes or ${} wrapper
		if strings.Contains(got, `\"`) {
			t.Errorf("GetUUIDMappings()[%q] should not contain escaped quotes, got %q", tt.key, got)
		}
		if strings.Contains(got, "${") {
			t.Errorf("GetUUIDMappings()[%q] should not contain ${} wrapper, got %q", tt.key, got)
		}
	}
}

func TestTrigger_GetUUIDMappings_OnDemandParameters(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		trigger      Trigger
		resourceName string
		wantKeys     []string
		wantValues   []string
	}{
		{
			name: "single parameter",
			trigger: Trigger{
				ID:   "trigger-1",
				Type: "on_demand",
				RawData: map[string]interface{}{
					"parameters": []interface{}{
						map[string]interface{}{
							"id":   "param-abc",
							"name": "user_input",
						},
					},
				},
			},
			resourceName: "my_trigger",
			wantKeys:     []string{"param-abc"},
			wantValues:   []string{`elementum_on_demand_trigger.my_trigger.parameter_ids["user_input"]`},
		},
		{
			name: "multiple parameters",
			trigger: Trigger{
				ID:   "trigger-2",
				Type: "on_demand",
				RawData: map[string]interface{}{
					"parameters": []interface{}{
						map[string]interface{}{"id": "p1", "name": "email"},
						map[string]interface{}{"id": "p2", "name": "subject"},
						map[string]interface{}{"id": "p3", "name": "body"},
					},
				},
			},
			resourceName: "email_trigger",
			wantKeys:     []string{"p1", "p2", "p3"},
			wantValues: []string{
				`elementum_on_demand_trigger.email_trigger.parameter_ids["email"]`,
				`elementum_on_demand_trigger.email_trigger.parameter_ids["subject"]`,
				`elementum_on_demand_trigger.email_trigger.parameter_ids["body"]`,
			},
		},
		{
			name: "parameter with special characters in name",
			trigger: Trigger{
				ID:   "trigger-3",
				Type: "on_demand",
				RawData: map[string]interface{}{
					"parameters": []interface{}{
						map[string]interface{}{
							"id":   "param-special",
							"name": "user_email_address",
						},
					},
				},
			},
			resourceName: "special_trigger",
			wantKeys:     []string{"param-special"},
			wantValues:   []string{`elementum_on_demand_trigger.special_trigger.parameter_ids["user_email_address"]`},
		},
		{
			name: "parameter missing id - should be skipped",
			trigger: Trigger{
				ID:   "trigger-4",
				Type: "on_demand",
				RawData: map[string]interface{}{
					"parameters": []interface{}{
						map[string]interface{}{
							"name": "orphan_param", // no id
						},
					},
				},
			},
			resourceName: "skip_trigger",
			wantKeys:     []string{}, // no parameter mappings expected
			wantValues:   []string{},
		},
		{
			name: "parameter missing name - should be skipped",
			trigger: Trigger{
				ID:   "trigger-5",
				Type: "on_demand",
				RawData: map[string]interface{}{
					"parameters": []interface{}{
						map[string]interface{}{
							"id": "param-no-name", // no name
						},
					},
				},
			},
			resourceName: "skip_trigger_2",
			wantKeys:     []string{}, // no parameter mappings expected
			wantValues:   []string{},
		},
		{
			name: "parameter with empty id - should be skipped",
			trigger: Trigger{
				ID:   "trigger-6",
				Type: "on_demand",
				RawData: map[string]interface{}{
					"parameters": []interface{}{
						map[string]interface{}{
							"id":   "",
							"name": "empty_id_param",
						},
					},
				},
			},
			resourceName: "empty_id_trigger",
			wantKeys:     []string{},
			wantValues:   []string{},
		},
		{
			name: "parameter with empty name - should be skipped",
			trigger: Trigger{
				ID:   "trigger-7",
				Type: "on_demand",
				RawData: map[string]interface{}{
					"parameters": []interface{}{
						map[string]interface{}{
							"id":   "param-empty-name",
							"name": "",
						},
					},
				},
			},
			resourceName: "empty_name_trigger",
			wantKeys:     []string{},
			wantValues:   []string{},
		},
		{
			name: "mixed valid and invalid parameters",
			trigger: Trigger{
				ID:   "trigger-8",
				Type: "on_demand",
				RawData: map[string]interface{}{
					"parameters": []interface{}{
						map[string]interface{}{"id": "valid-1", "name": "good_param"},
						map[string]interface{}{"id": "", "name": "bad_empty_id"}, // skipped
						map[string]interface{}{"id": "valid-2", "name": "another_good"},
						map[string]interface{}{"name": "no_id"},             // skipped
						map[string]interface{}{"id": "valid-3", "name": ""}, // skipped
					},
				},
			},
			resourceName: "mixed_trigger",
			wantKeys:     []string{"valid-1", "valid-2"},
			wantValues: []string{
				`elementum_on_demand_trigger.mixed_trigger.parameter_ids["good_param"]`,
				`elementum_on_demand_trigger.mixed_trigger.parameter_ids["another_good"]`,
			},
		},
		{
			name: "non-on_demand trigger should not map parameters",
			trigger: Trigger{
				ID:   "trigger-9",
				Type: "webhook", // not on_demand
				RawData: map[string]interface{}{
					"parameters": []interface{}{
						map[string]interface{}{"id": "p1", "name": "should_not_map"},
					},
				},
			},
			resourceName: "webhook_trigger",
			wantKeys:     []string{},
			wantValues:   []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mappings := tt.trigger.GetUUIDMappings(tt.resourceName)

			// Check expected parameter mappings
			for i, key := range tt.wantKeys {
				got, exists := mappings[key]
				if !exists {
					t.Errorf("Expected mapping for key %q not found", key)
					continue
				}
				if got != tt.wantValues[i] {
					t.Errorf("Mapping for %q = %q, want %q", key, got, tt.wantValues[i])
				}
			}

			// Verify no unexpected parameter mappings (for skip tests)
			if len(tt.wantKeys) == 0 {
				for key, val := range mappings {
					if strings.Contains(val, "parameter_ids") {
						t.Errorf("Unexpected parameter mapping: %q -> %q", key, val)
					}
				}
			}
		})
	}
}

func TestIsRecordBasedTrigger(t *testing.T) {
	tests := []struct {
		triggerType string
		want        bool
	}{
		{"record_created", true},
		{"record_updated", true},
		{"webhook", false},
		{"on_demand", false},
		{"attachment_added", false},
		{"comment_added", false},
		{"email_ingestion", false},
		{"slack_message", false},
		{"agent_conversation_ended", false},
		{"approval_chain", false},
	}

	for _, tt := range tests {
		t.Run(tt.triggerType, func(t *testing.T) {
			if got := isRecordBasedTrigger(tt.triggerType); got != tt.want {
				t.Errorf("isRecordBasedTrigger(%q) = %v, want %v", tt.triggerType, got, tt.want)
			}
		})
	}
}

func TestBeautifiableInterface(t *testing.T) {
	// Test that all types implement the Beautifiable interface
	var _ Beautifiable = &App{}
	var _ Beautifiable = &Field{}
	var _ Beautifiable = &Automation{}
	var _ Beautifiable = &Trigger{}
	var _ Beautifiable = &Task{}
	var _ Beautifiable = &Layout{}
	var _ Beautifiable = &Flow{}
	var _ Beautifiable = &Agent{}
	var _ Beautifiable = &AgentTool{}
	var _ Beautifiable = &Widget{}
	var _ Beautifiable = &AIFileReader{}
	var _ Beautifiable = &ApprovalProcess{}
}

func TestTask_WithCrossAppFields(t *testing.T) {
	task := &Task{
		ID:         "task-123",
		Type:       "create_record",
		Name:       "Create Ticket",
		WorkflowID: "workflow-456",
		ParentID:   "trigger-789",
		ObjectID:   "app-cross-ref",
		FieldIDs:   []string{"field-1", "field-2", "field-3"},
	}

	// Verify task has all expected fields populated
	if task.ParentID == "" {
		t.Error("Task.ParentID should be populated")
	}
	if task.ObjectID == "" {
		t.Error("Task.ObjectID should be populated")
	}
	if len(task.FieldIDs) == 0 {
		t.Error("Task.FieldIDs should be populated")
	}

	// Verify GetUUIDMappings works with cross-app task
	mappings := task.GetUUIDMappings("create_ticket")
	expected := "elementum_create_record_task.create_ticket.id"
	if got := mappings["task-123"]; got != expected {
		t.Errorf("Task.GetUUIDMappings()[task-123] = %q, want %q", got, expected)
	}
}

func TestAutomation_WithTriggersAndTasks(t *testing.T) {
	automation := &Automation{
		ID:         "auto-123",
		Name:       "Test Automation",
		WorkflowID: "workflow-456",
		Triggers: []Trigger{
			{ID: "trigger-1", Type: "record_created"},
			{ID: "trigger-2", Type: "webhook"},
		},
		Tasks: []Task{
			{ID: "task-1", Type: "message"},
			{ID: "task-2", Type: "record_search"},
			{ID: "task-3", Type: "update_field"},
		},
	}

	if len(automation.Triggers) != 2 {
		t.Errorf("Automation.Triggers length = %d, want 2", len(automation.Triggers))
	}
	if len(automation.Tasks) != 3 {
		t.Errorf("Automation.Tasks length = %d, want 3", len(automation.Tasks))
	}

	mappings := automation.GetUUIDMappings("test_auto")
	if len(mappings) != 2 {
		t.Errorf("Automation.GetUUIDMappings() returned %d mappings, want 2", len(mappings))
	}
}

func TestAccessPolicy_GetUUIDMappings(t *testing.T) {
	policy := &AccessPolicy{
		ID:       "policy-uuid-123",
		ObjectID: "app-uuid-456",
		UserIDs:  []string{"user-1", "user-2"},
		GroupIDs: []string{"group-1"},
	}

	mappings := policy.GetUUIDMappings("my_policy")

	expected := "elementum_access_policy.my_policy.id"
	if got := mappings["policy-uuid-123"]; got != expected {
		t.Errorf("AccessPolicy.GetUUIDMappings()[policy-uuid-123] = %q, want %q", got, expected)
	}
}

func TestAccessPolicy_WithFilter(t *testing.T) {
	policy := &AccessPolicy{
		ID:       "policy-123",
		ObjectID: "app-456",
		Filter: map[string]interface{}{
			"comparison": map[string]interface{}{
				"fieldId":  "field-789",
				"operator": "equals",
				"value": map[string]interface{}{
					"literal": "active",
				},
			},
		},
		UserIDs:    []string{"user-1", "user-2"},
		GroupIDs:   []string{"group-1", "group-2"},
		UserEmails: map[string]string{"user-1": "alice@example.com", "user-2": "bob@example.com"},
		GroupNames: map[string]string{"group-1": "Admins", "group-2": "Editors"},
	}

	// Verify filter is populated
	if policy.Filter == nil {
		t.Error("AccessPolicy.Filter should be populated")
	}

	// Verify user/group mappings
	if len(policy.UserEmails) != 2 {
		t.Errorf("AccessPolicy.UserEmails length = %d, want 2", len(policy.UserEmails))
	}
	if len(policy.GroupNames) != 2 {
		t.Errorf("AccessPolicy.GroupNames length = %d, want 2", len(policy.GroupNames))
	}

	// Verify email lookup
	if policy.UserEmails["user-1"] != "alice@example.com" {
		t.Errorf("AccessPolicy.UserEmails[user-1] = %q, want alice@example.com", policy.UserEmails["user-1"])
	}

	// Verify group name lookup
	if policy.GroupNames["group-1"] != "Admins" {
		t.Errorf("AccessPolicy.GroupNames[group-1] = %q, want Admins", policy.GroupNames["group-1"])
	}
}

func TestAccessPolicy_BeautifiableInterface(t *testing.T) {
	// Test that AccessPolicy implements the Beautifiable interface
	var _ Beautifiable = &AccessPolicy{}
}

func TestApp_WithAccessPolicies(t *testing.T) {
	app := &App{
		ID:        "app-123",
		Name:      "Test App",
		Namespace: "testapp",
		AccessPolicies: []AccessPolicy{
			{
				ID:       "policy-1",
				ObjectID: "app-123",
				UserIDs:  []string{"user-1"},
				GroupIDs: []string{"group-1"},
			},
			{
				ID:       "policy-2",
				ObjectID: "app-123",
				UserIDs:  []string{"user-2", "user-3"},
				GroupIDs: []string{},
			},
		},
	}

	if len(app.AccessPolicies) != 2 {
		t.Errorf("App.AccessPolicies length = %d, want 2", len(app.AccessPolicies))
	}

	// Verify first policy
	if app.AccessPolicies[0].ID != "policy-1" {
		t.Errorf("AccessPolicy[0].ID = %q, want policy-1", app.AccessPolicies[0].ID)
	}

	// Verify second policy
	if len(app.AccessPolicies[1].UserIDs) != 2 {
		t.Errorf("AccessPolicy[1].UserIDs length = %d, want 2", len(app.AccessPolicies[1].UserIDs))
	}
}

func TestElement_WithAccessPolicies(t *testing.T) {
	element := &Element{
		ID:        "element-123",
		Name:      "Test Element",
		Namespace: "testelement",
		AccessPolicies: []AccessPolicy{
			{
				ID:       "policy-1",
				ObjectID: "element-123",
				GroupIDs: []string{"group-1", "group-2"},
			},
		},
	}

	if len(element.AccessPolicies) != 1 {
		t.Errorf("Element.AccessPolicies length = %d, want 1", len(element.AccessPolicies))
	}
}

// Tests for Chart types

func TestChart_GetUUIDMappings(t *testing.T) {
	t.Parallel()

	chart := &Chart{
		ID:   "chart-uuid-123",
		Name: "Pipeline by Stage",
		Type: "BAR",
	}

	mappings := chart.GetUUIDMappings("pipeline_by_stage")

	expected := "elementum_chart.pipeline_by_stage.id"
	if got := mappings["chart-uuid-123"]; got != expected {
		t.Errorf("Chart.GetUUIDMappings()[chart-uuid-123] = %q, want %q", got, expected)
	}
}

func TestChart_BeautifiableInterface(t *testing.T) {
	t.Parallel()

	// Test that Chart implements the Beautifiable interface
	var _ Beautifiable = &Chart{}
}

func TestChart_WithSubscriptions(t *testing.T) {
	t.Parallel()

	chart := &Chart{
		ID:   "chart-123",
		Name: "Revenue Chart",
		Type: "LINE",
		Subscriptions: []ChartSubscription{
			{
				DatasourceID: "app-456",
				Filter:       `{"type":"equals","field_id":"field-789","value":"Active"}`,
				Sort:         `[{"field_id":"field-abc","direction":"DESC"}]`,
				Limit:        "100",
				Tags:         []string{"revenue", "sales"},
				Columns: []ChartColumn{
					{
						ColumnID:    "field-revenue",
						Aggregation: "SUM",
						Name:        "Total Revenue",
					},
					{
						Expression: `{"operation":"divide","operands":[...]}`,
						Name:       "Percentage",
						ValueType:  "DECIMAL",
					},
				},
				GroupBys: []ChartGroupBy{
					{ColumnID: "field-region"},
					{Expression: `{"function":"date_trunc","args":["month","field-date"]}`},
				},
				Joins: []ChartJoin{
					{
						DatasourceID:  "element-789",
						LeftColumnID:  "field-ref",
						RightColumnID: "field-id",
						Type:          "LEFT",
					},
				},
			},
		},
	}

	// Verify subscriptions populated
	if len(chart.Subscriptions) != 1 {
		t.Errorf("Chart.Subscriptions length = %d, want 1", len(chart.Subscriptions))
	}

	sub := chart.Subscriptions[0]

	// Verify subscription fields
	if sub.DatasourceID != "app-456" {
		t.Errorf("Subscription.DatasourceID = %q, want %q", sub.DatasourceID, "app-456")
	}
	if sub.Limit != "100" {
		t.Errorf("Subscription.Limit = %q, want 100", sub.Limit)
	}
	if len(sub.Tags) != 2 {
		t.Errorf("Subscription.Tags length = %d, want 2", len(sub.Tags))
	}

	// Verify columns
	if len(sub.Columns) != 2 {
		t.Errorf("Subscription.Columns length = %d, want 2", len(sub.Columns))
	}
	if sub.Columns[0].Aggregation != "SUM" {
		t.Errorf("Column[0].Aggregation = %q, want %q", sub.Columns[0].Aggregation, "SUM")
	}
	if sub.Columns[1].ValueType != "DECIMAL" {
		t.Errorf("Column[1].ValueType = %q, want %q", sub.Columns[1].ValueType, "DECIMAL")
	}

	// Verify group_bys
	if len(sub.GroupBys) != 2 {
		t.Errorf("Subscription.GroupBys length = %d, want 2", len(sub.GroupBys))
	}
	if sub.GroupBys[1].Expression == "" {
		t.Error("GroupBy[1].Expression should be populated")
	}

	// Verify joins
	if len(sub.Joins) != 1 {
		t.Errorf("Subscription.Joins length = %d, want 1", len(sub.Joins))
	}
	if sub.Joins[0].Type != "LEFT" {
		t.Errorf("Join[0].Type = %q, want %q", sub.Joins[0].Type, "LEFT")
	}
}

func TestChart_AllChartTypes(t *testing.T) {
	t.Parallel()

	// All supported chart types
	chartTypes := []string{
		"AREA",
		"BAR",
		"DOUGHNUT",
		"LINE",
		"PIE",
		"RADAR",
		"SINGLEVALUE",
	}

	for _, chartType := range chartTypes {
		t.Run(chartType, func(t *testing.T) {
			chart := &Chart{
				ID:   "chart-" + chartType,
				Name: chartType + " Chart",
				Type: chartType,
			}

			if chart.Type != chartType {
				t.Errorf("Chart.Type = %q, want %q", chart.Type, chartType)
			}
		})
	}
}

func TestChartColumn_AllAggregationTypes(t *testing.T) {
	t.Parallel()

	// All supported aggregation types
	aggTypes := []string{
		"AVG",
		"COUNT",
		"COUNT_DISTINCT",
		"MAX",
		"MIN",
		"SUM",
	}

	for _, aggType := range aggTypes {
		t.Run(aggType, func(t *testing.T) {
			col := ChartColumn{
				ColumnID:    "field-123",
				Aggregation: aggType,
				Name:        "Test Column",
			}

			if col.Aggregation != aggType {
				t.Errorf("ChartColumn.Aggregation = %q, want %q", col.Aggregation, aggType)
			}
		})
	}
}

func TestChartColumn_AllValueTypes(t *testing.T) {
	t.Parallel()

	// All supported value types
	valueTypes := []string{
		"BOOLEAN",
		"DATE",
		"DATETIME",
		"DECIMAL",
		"LONG",
		"TEXT",
		"UUID",
	}

	for _, valueType := range valueTypes {
		t.Run(valueType, func(t *testing.T) {
			col := ChartColumn{
				Value:     "static_value",
				ValueType: valueType,
				Name:      "Test Column",
			}

			if col.ValueType != valueType {
				t.Errorf("ChartColumn.ValueType = %q, want %q", col.ValueType, valueType)
			}
		})
	}
}

func TestChartJoin_AllJoinTypes(t *testing.T) {
	t.Parallel()

	// All supported join types
	joinTypes := []string{
		"INNER",
		"LEFT",
		"RIGHT",
		"FULL",
	}

	for _, joinType := range joinTypes {
		t.Run(joinType, func(t *testing.T) {
			join := ChartJoin{
				DatasourceID:  "ds-123",
				LeftColumnID:  "left-col",
				RightColumnID: "right-col",
				Type:          joinType,
			}

			if join.Type != joinType {
				t.Errorf("ChartJoin.Type = %q, want %q", join.Type, joinType)
			}
		})
	}
}

func TestChartJoin_WithExpressions(t *testing.T) {
	t.Parallel()

	join := ChartJoin{
		DatasourceID:    "ds-456",
		LeftExpression:  `{"function":"lower","args":["field-name"]}`,
		RightExpression: `{"function":"lower","args":["field-ref"]}`,
		Type:            "INNER",
	}

	// Verify expression-based join (no column IDs)
	if join.LeftColumnID != "" {
		t.Error("Expression-based join should not have LeftColumnID")
	}
	if join.RightColumnID != "" {
		t.Error("Expression-based join should not have RightColumnID")
	}
	if join.LeftExpression == "" {
		t.Error("Expression-based join should have LeftExpression")
	}
	if join.RightExpression == "" {
		t.Error("Expression-based join should have RightExpression")
	}
}

func TestChart_WithProperties(t *testing.T) {
	t.Parallel()

	target := 100.0
	beginAtZero := true
	fill := true
	horizontal := false
	legend := true
	stacked := false
	targetProp := 50.0

	chart := &Chart{
		ID:     "chart-props",
		Name:   "Chart with Properties",
		Type:   "BAR",
		Target: &target,
		Properties: ChartProperties{
			BeginAtZero: &beginAtZero,
			Fill:        &fill,
			Horizontal:  &horizontal,
			Legend:      &legend,
			Stacked:     &stacked,
			Target:      &targetProp,
		},
	}

	// Verify target
	if chart.Target == nil || *chart.Target != 100.0 {
		t.Error("Chart.Target should be 100.0")
	}

	// Verify properties
	if chart.Properties.BeginAtZero == nil || !*chart.Properties.BeginAtZero {
		t.Error("Properties.BeginAtZero should be true")
	}
	if chart.Properties.Fill == nil || !*chart.Properties.Fill {
		t.Error("Properties.Fill should be true")
	}
	if chart.Properties.Horizontal == nil || *chart.Properties.Horizontal {
		t.Error("Properties.Horizontal should be false")
	}
	if chart.Properties.Legend == nil || !*chart.Properties.Legend {
		t.Error("Properties.Legend should be true")
	}
	if chart.Properties.Stacked == nil || *chart.Properties.Stacked {
		t.Error("Properties.Stacked should be false")
	}
	if chart.Properties.Target == nil || *chart.Properties.Target != 50.0 {
		t.Error("Properties.Target should be 50.0")
	}
}

func TestApp_WithCharts(t *testing.T) {
	t.Parallel()

	app := &App{
		ID:        "app-123",
		Name:      "Test App",
		Namespace: "testapp",
		Charts: []Chart{
			{
				ID:   "chart-1",
				Name: "Revenue Chart",
				Type: "LINE",
			},
			{
				ID:   "chart-2",
				Name: "Pipeline Chart",
				Type: "BAR",
			},
		},
	}

	if len(app.Charts) != 2 {
		t.Errorf("App.Charts length = %d, want 2", len(app.Charts))
	}

	if app.Charts[0].Type != "LINE" {
		t.Errorf("Chart[0].Type = %q, want LINE", app.Charts[0].Type)
	}
	if app.Charts[1].Type != "BAR" {
		t.Errorf("Chart[1].Type = %q, want BAR", app.Charts[1].Type)
	}
}

func TestUnifiedDiscoveryContext_StoreChartSafe(t *testing.T) {
	t.Parallel()

	ctx := &UnifiedDiscoveryContext{
		Charts: make(map[string]*Chart),
	}

	chart := &Chart{ID: "chart-123", Name: "Test Chart", Type: "BAR"}
	ctx.StoreChartSafe("chart-123", chart)

	if ctx.Charts["chart-123"] != chart {
		t.Error("Chart should be stored")
	}
}

func TestDashboardWidget_WithChartID(t *testing.T) {
	t.Parallel()

	widget := DashboardWidget{
		ID:          "widget-123",
		DashboardID: "dashboard-456",
		Name:        "Pipeline Widget",
		Size:        "MEDIUM",
		ChartID:     "chart-789", // Chart widget
	}

	if widget.ChartID != "chart-789" {
		t.Errorf("DashboardWidget.ChartID = %q, want chart-789", widget.ChartID)
	}

	// Aspect widget should have empty ChartID
	aspectWidget := DashboardWidget{
		ID:          "widget-456",
		DashboardID: "dashboard-456",
		Name:        "Records Widget",
		Size:        "LARGE",
		AspectID:    "app-123", // Aspect widget
	}

	if aspectWidget.ChartID != "" {
		t.Error("Aspect widget should have empty ChartID")
	}
	if aspectWidget.AspectID != "app-123" {
		t.Errorf("DashboardWidget.AspectID = %q, want app-123", aspectWidget.AspectID)
	}
}
