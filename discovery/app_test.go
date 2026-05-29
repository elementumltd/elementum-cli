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

package discovery

import (
	"testing"
)

func TestParseNamespaceFromURL(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "full URL with records path",
			input: "https://appdemo.elementum.io/app/accountassignment/records",
			want:  "accountassignment",
		},
		{
			name:  "full URL with record detail path",
			input: "https://appdemo.elementum.io/app/accountassignment/record/123-456",
			want:  "accountassignment",
		},
		{
			name:  "full URL with custom path",
			input: "https://demo.elementum.io/app/my-app/settings",
			want:  "my-app",
		},
		{
			name:  "URL without protocol",
			input: "demo.elementum.io/app/myapp/records",
			want:  "myapp",
		},
		{
			name:  "just namespace",
			input: "accountassignment",
			want:  "accountassignment",
		},
		{
			name:  "namespace with underscores",
			input: "my_app_name",
			want:  "my_app_name",
		},
		{
			name:  "namespace with dashes",
			input: "my-app-name",
			want:  "my-app-name",
		},
		{
			name:  "URL with trailing slash",
			input: "https://appdemo.elementum.io/app/myapp/",
			want:  "myapp",
		},
		{
			name:  "malformed URL defaults to input",
			input: "not-a-url-or-namespace",
			want:  "not-a-url-or-namespace",
		},
		{
			name:  "full URL with /apps/ (plural) and flow path",
			input: "https://appdemo.elementum.io/admin/apps/dealclosingprocess/flow",
			want:  "dealclosingprocess",
		},
		{
			name:  "full URL with /apps/ (plural) and records path",
			input: "https://appdemo.elementum.io/apps/accountassignment/records",
			want:  "accountassignment",
		},
		{
			name:  "full URL with /apps/ (plural) and record detail",
			input: "https://demo.elementum.io/apps/my-app/record/123-456",
			want:  "my-app",
		},
		{
			name:  "URL with /admin/apps/ prefix",
			input: "https://appdemo.elementum.io/admin/apps/myapp/settings",
			want:  "myapp",
		},
		{
			name:  "URL with /apps/ (plural) and trailing slash",
			input: "https://appdemo.elementum.io/apps/myapp/",
			want:  "myapp",
		},
		{
			name:  "URL without protocol with /apps/ (plural)",
			input: "demo.elementum.io/apps/myapp/records",
			want:  "myapp",
		},
		{
			name:  "path with /apps/ only (no protocol)",
			input: "/apps/dealclosingprocess/flow",
			want:  "dealclosingprocess",
		},
		{
			name:  "path with /admin/apps/ only (no protocol)",
			input: "/admin/apps/myapp/settings",
			want:  "myapp",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseNamespaceFromURL(tt.input)
			if err != nil {
				t.Fatalf("ParseNamespaceFromURL(%q) returned error: %v", tt.input, err)
			}
			if got != tt.want {
				t.Errorf("ParseNamespaceFromURL(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestMapFieldType(t *testing.T) {
	tests := []struct {
		name     string
		typename string
		want     string
	}{
		{
			name:     "text field",
			typename: "AspectTextField",
			want:     "text",
		},
		{
			name:     "html/longtext field",
			typename: "AspectHtmlField",
			want:     "longtext",
		},
		{
			name:     "number field",
			typename: "AspectNumberField",
			want:     "number",
		},
		{
			name:     "decimal field",
			typename: "AspectDecimalField",
			want:     "decimal",
		},
		{
			name:     "boolean field - maps to boolean for display",
			typename: "AspectBooleanField",
			want:     "boolean",
		},
		{
			name:     "date field",
			typename: "AspectDateField",
			want:     "date",
		},
		{
			name:     "datetime field",
			typename: "AspectDateTimeField",
			want:     "datetime",
		},
		{
			name:     "dropdown/picklist field",
			typename: "AspectPicklistField",
			want:     "dropdown",
		},
		{
			name:     "multiselect field - maps to multiselect for display",
			typename: "AspectMultiPicklistField",
			want:     "multiselect",
		},
		{
			name:     "user field",
			typename: "AspectUserField",
			want:     "user",
		},
		{
			name:     "group field",
			typename: "AspectGroupField",
			want:     "group",
		},
		{
			name:     "attachment field",
			typename: "AspectAttachmentField",
			want:     "attachment",
		},
		{
			name:     "json field",
			typename: "AspectJsonField",
			want:     "json",
		},
		{
			name:     "calculated field",
			typename: "AspectCalculatedField",
			want:     "calculated",
		},
		{
			name:     "handle field (system ID)",
			typename: "AspectHandleField",
			want:     "handle",
		},
		{
			name:     "relation field",
			typename: "AspectRelationField",
			want:     "relation",
		},
		{
			name:     "unknown field type returns unknown",
			typename: "AspectUnknownField",
			want:     "unknown",
		},
		{
			name:     "empty typename returns unknown",
			typename: "",
			want:     "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mapFieldType(tt.typename)
			if got != tt.want {
				t.Errorf("mapFieldType(%q) = %q, want %q", tt.typename, got, tt.want)
			}
		})
	}
}

func TestMapTriggerType(t *testing.T) {
	tests := []struct {
		name     string
		typename string
		want     string
	}{
		{
			name:     "record created trigger",
			typename: "WorkflowRecordCreateTrigger",
			want:     "record_created",
		},
		{
			name:     "record updated trigger",
			typename: "WorkflowRecordUpdateTrigger",
			want:     "record_updated",
		},
		{
			name:     "attachment added trigger",
			typename: "WorkflowAttachmentTrigger",
			want:     "attachment_added",
		},
		{
			name:     "comment added trigger",
			typename: "WorkflowCommentAddedTrigger",
			want:     "comment_added",
		},
		{
			name:     "webhook trigger",
			typename: "WorkflowWebhookTrigger",
			want:     "webhook",
		},
		{
			name:     "on demand trigger",
			typename: "WorkflowOnDemandTrigger",
			want:     "on_demand",
		},
		{
			name:     "slack message trigger",
			typename: "WorkflowSlackMessageTrigger",
			want:     "slack_message",
		},
		{
			name:     "email ingestion trigger",
			typename: "WorkflowEmailIngestionTrigger",
			want:     "email_ingestion",
		},
		{
			name:     "agent conversation ended trigger",
			typename: "WorkflowConversationEndTrigger",
			want:     "agent_conversation_ended",
		},
		{
			name:     "approval chain trigger",
			typename: "WorkflowApprovalChainTrigger",
			want:     "approval_chain",
		},
		{
			name:     "datamine trigger",
			typename: "WorkflowDatamineTrigger",
			want:     "datamine",
		},
		{
			name:     "unknown trigger type",
			typename: "WorkflowUnknownTrigger",
			want:     "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mapTriggerType(tt.typename)
			if got != tt.want {
				t.Errorf("mapTriggerType(%q) = %q, want %q", tt.typename, got, tt.want)
			}
		})
	}
}

func TestMapTaskType(t *testing.T) {
	tests := []struct {
		name     string
		typename string
		want     string
	}{
		{
			name:     "message task",
			typename: "WorkflowMessageTask",
			want:     "message",
		},
		{
			name:     "update field task",
			typename: "WorkflowUpdateFieldTask",
			want:     "update_field",
		},
		{
			name:     "variable task",
			typename: "WorkflowVariableTask",
			want:     "variable",
		},
		{
			name:     "create record task",
			typename: "WorkflowCreateRecordTask",
			want:     "create_record",
		},
		{
			name:     "record search task",
			typename: "WorkflowRecordSearchTask",
			want:     "record_search",
		},
		{
			name:     "send email task",
			typename: "WorkflowSendEmailTask",
			want:     "send_email",
		},
		{
			name:     "api task",
			typename: "WorkflowApiTask",
			want:     "api",
		},
		{
			name:     "approval chain task",
			typename: "WorkflowApprovalChainTemplateTask",
			want:     "approval_chain",
		},
		{
			name:     "approval status update task",
			typename: "WorkflowApprovalStatusUpdateTask",
			want:     "approval_status_update",
		},
		{
			name:     "find related records task",
			typename: "WorkflowFindRelatedRecordsTask",
			want:     "find_related_records",
		},
		{
			name:     "ai agent task",
			typename: "WorkflowAiAgentTask",
			want:     "ai_agent",
		},
		{
			name:     "for each task",
			typename: "WorkflowForEachTask",
			want:     "for_each",
		},
		{
			name:     "switch task",
			typename: "WorkflowSwitchTask",
			want:     "switch",
		},
		{
			name:     "ai file read task",
			typename: "WorkflowAiFileAnalysisTask",
			want:     "ai_file_read",
		},
		{
			name:     "unknown task type",
			typename: "WorkflowUnknownTask",
			want:     "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mapTaskType(tt.typename)
			if got != tt.want {
				t.Errorf("mapTaskType(%q) = %q, want %q", tt.typename, got, tt.want)
			}
		})
	}
}

// =============================================================================
// determineTaskType Tests
// =============================================================================

func TestDetermineTaskType_VariableCreate(t *testing.T) {
	t.Parallel()

	taskData := map[string]interface{}{
		"__typename": "WorkflowVariableTask",
		"id":         "task-123",
		"name":       "Set Counter",
		"variables": []interface{}{
			map[string]interface{}{
				"__typename": "WorkflowVariableTaskParameterCreate",
				"id":         "var-1",
				"name":       "counter",
				"type":       "NUMBER",
				"createValue": map[string]interface{}{
					"id":    "ref-1",
					"label": "0",
					"value": "0",
				},
			},
		},
	}

	got := determineTaskType(taskData)
	if got != "variable" {
		t.Errorf("determineTaskType() = %q, want %q", got, "variable")
	}
}

func TestDetermineTaskType_VariableUpdate(t *testing.T) {
	t.Parallel()

	taskData := map[string]interface{}{
		"__typename": "WorkflowVariableTask",
		"id":         "task-456",
		"name":       "Increment Counter",
		"variables": []interface{}{
			map[string]interface{}{
				"__typename": "WorkflowVariableTaskParameterUpdate",
				"variable": map[string]interface{}{
					"id":    "var-ref-1",
					"label": "counter",
					"value": `{"variableReference":{"id":"var-123"}}`,
				},
				"updateValue": map[string]interface{}{
					"id":    "ref-2",
					"label": "new value",
					"value": "new value",
				},
			},
		},
	}

	got := determineTaskType(taskData)
	if got != "update_variable" {
		t.Errorf("determineTaskType() = %q, want %q", got, "update_variable")
	}
}

func TestDetermineTaskType_OtherTaskTypes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		taskData map[string]interface{}
		want     string
	}{
		{
			name: "message task",
			taskData: map[string]interface{}{
				"__typename": "WorkflowMessageTask",
				"id":         "task-1",
			},
			want: "message",
		},
		{
			name: "update_field task",
			taskData: map[string]interface{}{
				"__typename": "WorkflowUpdateFieldTask",
				"id":         "task-2",
			},
			want: "update_field",
		},
		{
			name: "api task",
			taskData: map[string]interface{}{
				"__typename": "WorkflowApiTask",
				"id":         "task-3",
			},
			want: "api",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := determineTaskType(tt.taskData)
			if got != tt.want {
				t.Errorf("determineTaskType() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestIsUpdateVariableTask(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		taskData map[string]interface{}
		want     bool
	}{
		{
			name: "create variable task",
			taskData: map[string]interface{}{
				"variables": []interface{}{
					map[string]interface{}{
						"__typename": "WorkflowVariableTaskParameterCreate",
					},
				},
			},
			want: false,
		},
		{
			name: "update variable task",
			taskData: map[string]interface{}{
				"variables": []interface{}{
					map[string]interface{}{
						"__typename": "WorkflowVariableTaskParameterUpdate",
					},
				},
			},
			want: true,
		},
		{
			name:     "empty variables",
			taskData: map[string]interface{}{},
			want:     false,
		},
		{
			name: "nil variables array",
			taskData: map[string]interface{}{
				"variables": nil,
			},
			want: false,
		},
		{
			name: "empty variables array",
			taskData: map[string]interface{}{
				"variables": []interface{}{},
			},
			want: false,
		},
		{
			name: "malformed variables (not array)",
			taskData: map[string]interface{}{
				"variables": "not an array",
			},
			want: false,
		},
		{
			name: "malformed first variable (not map)",
			taskData: map[string]interface{}{
				"variables": []interface{}{
					"not a map",
				},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isUpdateVariableTask(tt.taskData)
			if got != tt.want {
				t.Errorf("isUpdateVariableTask() = %v, want %v", got, tt.want)
			}
		})
	}
}

// =============================================================================
// CollectSearchTableAiProviderConnectors Tests
// =============================================================================

// TestCollectSearchTableAiProviderConnectors_FromAppSearchTables tests that AI
// provider connectors are collected from app-level AI search tables
func TestCollectSearchTableAiProviderConnectors_FromAppSearchTables(t *testing.T) {
	t.Parallel()

	app := &App{
		ID:   "app-uuid",
		Name: "Test App",
		AISearchTables: []AISearchTable{
			{
				ID:                      "search-table-1",
				AIProviderConnectorID:   "connector-uuid-1",
				AIProviderConnectorName: "Snowflake Arctic L V2.0",
			},
			{
				ID:                      "search-table-2",
				AIProviderConnectorID:   "connector-uuid-2",
				AIProviderConnectorName: "OpenAI GPT-4",
			},
		},
		DiscoveredAiProviderConnectors: []AiProviderConnector{},
	}

	CollectSearchTableAiProviderConnectors(app)

	if len(app.DiscoveredAiProviderConnectors) != 2 {
		t.Errorf("Expected 2 connectors, got %d", len(app.DiscoveredAiProviderConnectors))
	}

	// Check first connector
	found1 := false
	for _, conn := range app.DiscoveredAiProviderConnectors {
		if conn.ID == "connector-uuid-1" && conn.ModelName == "Snowflake Arctic L V2.0" {
			found1 = true
			break
		}
	}
	if !found1 {
		t.Error("First connector not found")
	}

	// Check second connector
	found2 := false
	for _, conn := range app.DiscoveredAiProviderConnectors {
		if conn.ID == "connector-uuid-2" && conn.ModelName == "OpenAI GPT-4" {
			found2 = true
			break
		}
	}
	if !found2 {
		t.Error("Second connector not found")
	}
}

// TestCollectSearchTableAiProviderConnectors_FromDiscoveredElements tests that
// AI provider connectors are collected from discovered elements' AI search tables
func TestCollectSearchTableAiProviderConnectors_FromDiscoveredElements(t *testing.T) {
	t.Parallel()

	app := &App{
		ID:             "app-uuid",
		Name:           "Test App",
		AISearchTables: []AISearchTable{}, // No app-level search tables
		DiscoveredElements: []*Element{
			{
				ID:        "element-1",
				Name:      "Element 1",
				Namespace: "element1",
				AISearchTables: []AISearchTable{
					{
						ID:                      "search-table-elem-1",
						AIProviderConnectorID:   "connector-elem-1",
						AIProviderConnectorName: "Snowflake Arctic L V2.0",
					},
				},
			},
			{
				ID:        "element-2",
				Name:      "Element 2",
				Namespace: "element2",
				AISearchTables: []AISearchTable{
					{
						ID:                      "search-table-elem-2",
						AIProviderConnectorID:   "connector-elem-2",
						AIProviderConnectorName: "Azure OpenAI",
					},
				},
			},
		},
		DiscoveredAiProviderConnectors: []AiProviderConnector{},
	}

	CollectSearchTableAiProviderConnectors(app)

	if len(app.DiscoveredAiProviderConnectors) != 2 {
		t.Errorf("Expected 2 connectors from elements, got %d", len(app.DiscoveredAiProviderConnectors))
	}
}

// TestCollectSearchTableAiProviderConnectors_DeduplicatesConnectors tests that
// duplicate connector IDs are not added twice
func TestCollectSearchTableAiProviderConnectors_DeduplicatesConnectors(t *testing.T) {
	t.Parallel()

	// Same connector used in multiple search tables
	sharedConnectorID := "shared-connector-id"

	app := &App{
		ID:   "app-uuid",
		Name: "Test App",
		AISearchTables: []AISearchTable{
			{
				ID:                      "search-table-1",
				AIProviderConnectorID:   sharedConnectorID,
				AIProviderConnectorName: "Snowflake Arctic L V2.0",
			},
		},
		DiscoveredElements: []*Element{
			{
				ID:        "element-1",
				Name:      "Element 1",
				Namespace: "element1",
				AISearchTables: []AISearchTable{
					{
						ID:                      "search-table-elem-1",
						AIProviderConnectorID:   sharedConnectorID, // Same connector
						AIProviderConnectorName: "Snowflake Arctic L V2.0",
					},
				},
			},
		},
		DiscoveredAiProviderConnectors: []AiProviderConnector{},
	}

	CollectSearchTableAiProviderConnectors(app)

	// Should only have 1 connector (deduplicated)
	if len(app.DiscoveredAiProviderConnectors) != 1 {
		t.Errorf("Expected 1 deduplicated connector, got %d", len(app.DiscoveredAiProviderConnectors))
	}
}

// TestCollectSearchTableAiProviderConnectors_PreservesExistingConnectors tests
// that already-discovered connectors are preserved and not duplicated
func TestCollectSearchTableAiProviderConnectors_PreservesExistingConnectors(t *testing.T) {
	t.Parallel()

	existingConnectorID := "existing-connector"
	newConnectorID := "new-connector"

	app := &App{
		ID:   "app-uuid",
		Name: "Test App",
		AISearchTables: []AISearchTable{
			{
				ID:                      "search-table-1",
				AIProviderConnectorID:   existingConnectorID,
				AIProviderConnectorName: "Existing Model",
			},
			{
				ID:                      "search-table-2",
				AIProviderConnectorID:   newConnectorID,
				AIProviderConnectorName: "New Model",
			},
		},
		// Existing connector already discovered (e.g., from agent tools)
		DiscoveredAiProviderConnectors: []AiProviderConnector{
			{
				ID:        existingConnectorID,
				ModelName: "Existing Model",
			},
		},
	}

	CollectSearchTableAiProviderConnectors(app)

	// Should have 2 total (1 existing + 1 new)
	if len(app.DiscoveredAiProviderConnectors) != 2 {
		t.Errorf("Expected 2 connectors, got %d", len(app.DiscoveredAiProviderConnectors))
	}

	// Verify both are present
	hasExisting := false
	hasNew := false
	for _, conn := range app.DiscoveredAiProviderConnectors {
		if conn.ID == existingConnectorID {
			hasExisting = true
		}
		if conn.ID == newConnectorID {
			hasNew = true
		}
	}

	if !hasExisting {
		t.Error("Existing connector should be preserved")
	}
	if !hasNew {
		t.Error("New connector should be added")
	}
}

// TestCollectSearchTableAiProviderConnectors_SkipsEmptyConnectorIDs tests that
// search tables without connector IDs are skipped
func TestCollectSearchTableAiProviderConnectors_SkipsEmptyConnectorIDs(t *testing.T) {
	t.Parallel()

	app := &App{
		ID:   "app-uuid",
		Name: "Test App",
		AISearchTables: []AISearchTable{
			{
				ID:                      "search-table-1",
				AIProviderConnectorID:   "", // Empty - should be skipped
				AIProviderConnectorName: "",
			},
			{
				ID:                      "search-table-2",
				AIProviderConnectorID:   "valid-connector",
				AIProviderConnectorName: "Valid Model",
			},
		},
		DiscoveredAiProviderConnectors: []AiProviderConnector{},
	}

	CollectSearchTableAiProviderConnectors(app)

	// Should only have 1 connector (the valid one)
	if len(app.DiscoveredAiProviderConnectors) != 1 {
		t.Errorf("Expected 1 connector, got %d", len(app.DiscoveredAiProviderConnectors))
	}
}

// TestCollectSearchTableAiProviderConnectors_NilApp tests that nil app doesn't panic
func TestCollectSearchTableAiProviderConnectors_NilApp(t *testing.T) {
	t.Parallel()

	// Should not panic
	CollectSearchTableAiProviderConnectors(nil)
}

// TestCollectSearchTableAiProviderConnectors_EmptyApp tests that empty app doesn't panic
func TestCollectSearchTableAiProviderConnectors_EmptyApp(t *testing.T) {
	t.Parallel()

	app := &App{
		ID:                             "app-uuid",
		Name:                           "Empty App",
		AISearchTables:                 []AISearchTable{},
		DiscoveredElements:             []*Element{},
		DiscoveredAiProviderConnectors: []AiProviderConnector{},
	}

	// Should not panic
	CollectSearchTableAiProviderConnectors(app)

	if len(app.DiscoveredAiProviderConnectors) != 0 {
		t.Errorf("Expected 0 connectors for empty app, got %d", len(app.DiscoveredAiProviderConnectors))
	}
}

// =============================================================================
// HasWidgetsInDiscoveredResources Tests
// =============================================================================

// HasWidgetsInDiscoveredResources checks if there are widgets in the main app,
// any discovered apps, or any discovered elements.
// This is a test helper that mirrors the logic in export.go for selectedTypes["widgets"]
func hasWidgetsInDiscoveredResources(app *App) bool {
	if len(app.Widgets) > 0 {
		return true
	}
	for _, discoveredApp := range app.DiscoveredApps {
		if len(discoveredApp.Widgets) > 0 {
			return true
		}
	}
	for _, elem := range app.DiscoveredElements {
		if len(elem.Widgets) > 0 {
			return true
		}
	}
	return false
}

// hasViewsInDiscoveredResources checks if there are views in the main app,
// any discovered apps, or any discovered elements.
// This is a test helper that mirrors the logic in export.go for selectedTypes["views"]
func hasViewsInDiscoveredResources(app *App) bool {
	if len(app.Views) > 0 {
		return true
	}
	for _, discoveredApp := range app.DiscoveredApps {
		if len(discoveredApp.Views) > 0 {
			return true
		}
	}
	for _, elem := range app.DiscoveredElements {
		if len(elem.Views) > 0 {
			return true
		}
	}
	return false
}

// TestHasWidgetsInDiscoveredResources_MainAppWidgets tests that widgets
// on the main app are detected
func TestHasWidgetsInDiscoveredResources_MainAppWidgets(t *testing.T) {
	t.Parallel()

	app := &App{
		ID:   "app-uuid",
		Name: "Test App",
		Widgets: []Widget{
			{ID: "widget-1", Name: "Widget 1"},
		},
		DiscoveredApps:     []*App{},
		DiscoveredElements: []*Element{},
	}

	if !hasWidgetsInDiscoveredResources(app) {
		t.Error("Expected hasWidgetsInDiscoveredResources to return true for main app with widgets")
	}
}

// TestHasWidgetsInDiscoveredResources_DiscoveredAppWidgets tests that widgets
// on discovered apps are detected
func TestHasWidgetsInDiscoveredResources_DiscoveredAppWidgets(t *testing.T) {
	t.Parallel()

	app := &App{
		ID:      "app-uuid",
		Name:    "Test App",
		Widgets: []Widget{}, // Main app has no widgets
		DiscoveredApps: []*App{
			{
				ID:   "discovered-app-uuid",
				Name: "Discovered App",
				Widgets: []Widget{
					{ID: "widget-1", Name: "Widget on Discovered App"},
				},
			},
		},
		DiscoveredElements: []*Element{},
	}

	if !hasWidgetsInDiscoveredResources(app) {
		t.Error("Expected hasWidgetsInDiscoveredResources to return true for discovered app with widgets")
	}
}

// TestHasWidgetsInDiscoveredResources_DiscoveredElementWidgets tests that widgets
// on discovered elements are detected
func TestHasWidgetsInDiscoveredResources_DiscoveredElementWidgets(t *testing.T) {
	t.Parallel()

	app := &App{
		ID:             "app-uuid",
		Name:           "Test App",
		Widgets:        []Widget{}, // Main app has no widgets
		DiscoveredApps: []*App{},   // No discovered apps
		DiscoveredElements: []*Element{
			{
				ID:        "element-uuid",
				Name:      "Discovered Element",
				Namespace: "discoveredelement",
				Widgets: []Widget{
					{ID: "widget-1", Name: "Widget on Discovered Element"},
				},
			},
		},
	}

	if !hasWidgetsInDiscoveredResources(app) {
		t.Error("Expected hasWidgetsInDiscoveredResources to return true for discovered element with widgets")
	}
}

// TestHasWidgetsInDiscoveredResources_NoWidgets tests that the function returns
// false when there are no widgets anywhere
func TestHasWidgetsInDiscoveredResources_NoWidgets(t *testing.T) {
	t.Parallel()

	app := &App{
		ID:      "app-uuid",
		Name:    "Test App",
		Widgets: []Widget{},
		DiscoveredApps: []*App{
			{
				ID:      "discovered-app-uuid",
				Name:    "Discovered App",
				Widgets: []Widget{}, // No widgets
			},
		},
		DiscoveredElements: []*Element{
			{
				ID:        "element-uuid",
				Name:      "Discovered Element",
				Namespace: "discoveredelement",
				Widgets:   []Widget{}, // No widgets
			},
		},
	}

	if hasWidgetsInDiscoveredResources(app) {
		t.Error("Expected hasWidgetsInDiscoveredResources to return false when no widgets exist")
	}
}

// TestHasViewsInDiscoveredResources_MainAppViews tests that views
// on the main app are detected
func TestHasViewsInDiscoveredResources_MainAppViews(t *testing.T) {
	t.Parallel()

	app := &App{
		ID:   "app-uuid",
		Name: "Test App",
		Views: []View{
			{ID: "view-1", Name: "View 1"},
		},
		DiscoveredApps:     []*App{},
		DiscoveredElements: []*Element{},
	}

	if !hasViewsInDiscoveredResources(app) {
		t.Error("Expected hasViewsInDiscoveredResources to return true for main app with views")
	}
}

// TestHasViewsInDiscoveredResources_DiscoveredAppViews tests that views
// on discovered apps are detected
func TestHasViewsInDiscoveredResources_DiscoveredAppViews(t *testing.T) {
	t.Parallel()

	app := &App{
		ID:    "app-uuid",
		Name:  "Test App",
		Views: []View{}, // Main app has no views
		DiscoveredApps: []*App{
			{
				ID:   "discovered-app-uuid",
				Name: "Discovered App",
				Views: []View{
					{ID: "view-1", Name: "View on Discovered App"},
				},
			},
		},
		DiscoveredElements: []*Element{},
	}

	if !hasViewsInDiscoveredResources(app) {
		t.Error("Expected hasViewsInDiscoveredResources to return true for discovered app with views")
	}
}

// TestHasViewsInDiscoveredResources_DiscoveredElementViews tests that views
// on discovered elements are detected
func TestHasViewsInDiscoveredResources_DiscoveredElementViews(t *testing.T) {
	t.Parallel()

	app := &App{
		ID:             "app-uuid",
		Name:           "Test App",
		Views:          []View{}, // Main app has no views
		DiscoveredApps: []*App{}, // No discovered apps
		DiscoveredElements: []*Element{
			{
				ID:        "element-uuid",
				Name:      "Discovered Element",
				Namespace: "discoveredelement",
				Views: []View{
					{ID: "view-1", Name: "View on Discovered Element"},
				},
			},
		},
	}

	if !hasViewsInDiscoveredResources(app) {
		t.Error("Expected hasViewsInDiscoveredResources to return true for discovered element with views")
	}
}

// TestHasViewsInDiscoveredResources_NoViews tests that the function returns
// false when there are no views anywhere
func TestHasViewsInDiscoveredResources_NoViews(t *testing.T) {
	t.Parallel()

	app := &App{
		ID:    "app-uuid",
		Name:  "Test App",
		Views: []View{},
		DiscoveredApps: []*App{
			{
				ID:    "discovered-app-uuid",
				Name:  "Discovered App",
				Views: []View{}, // No views
			},
		},
		DiscoveredElements: []*Element{
			{
				ID:        "element-uuid",
				Name:      "Discovered Element",
				Namespace: "discoveredelement",
				Views:     []View{}, // No views
			},
		},
	}

	if hasViewsInDiscoveredResources(app) {
		t.Error("Expected hasViewsInDiscoveredResources to return false when no views exist")
	}
}
