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

func TestNewHCLGenerator(t *testing.T) {
	app := &discovery.App{
		ID:   "app-123",
		Name: "Test App",
		Automations: []discovery.Automation{
			{
				ID:           "auto-1",
				Name:         "Test Automation",
				Status:       "ACTIVE",
				WorkflowID:   "workflow-1",
				HasPublished: true,
				Triggers: []discovery.Trigger{
					{
						ID:   "trigger-1",
						Type: "record_created",
					},
				},
				Tasks: []discovery.Task{
					{
						ID:         "task-1",
						Type:       "switch",
						Name:       "Check Condition",
						WorkflowID: "workflow-1",
						ParentID:   "trigger-1",
					},
					{
						ID:         "task-2",
						Type:       "update_field",
						Name:       "Update Record",
						WorkflowID: "workflow-1",
						ParentID:   "task-1",
						ObjectID:   "app-123",
					},
				},
			},
		},
	}

	imports := []ImportBlock{
		{ID: "app-123", ResourceType: "elementum_app", ResourceName: "test_app"},
		{ID: "app-123:auto-1", ResourceType: "elementum_automation", ResourceName: "test_automation"},
		{ID: "auto-1:trigger-1", ResourceType: "elementum_record_created_trigger", ResourceName: "test_automation_record_created_0"},
		{ID: "workflow-1:task-1", ResourceType: "elementum_switch_task", ResourceName: "test_automation_check_condition"},
		{ID: "workflow-1:task-2", ResourceType: "elementum_update_field_task", ResourceName: "test_automation_update_record"},
	}

	generator := NewHCLGenerator(app, imports)

	// Test that reference maps were built
	if len(generator.triggerRefMap) == 0 {
		t.Error("Expected trigger ref map to be populated")
	}
	if len(generator.taskRefMap) == 0 {
		t.Error("Expected task ref map to be populated")
	}
	if len(generator.parentMap) == 0 {
		t.Error("Expected parent map to be populated")
	}

	// Test trigger ref
	triggerRef := generator.GetTriggerRef("trigger-1")
	if !strings.Contains(triggerRef, "elementum_record_created_trigger") {
		t.Errorf("Expected trigger ref to contain resource type, got: %s", triggerRef)
	}

	// Test task ref
	taskRef := generator.GetTaskRef("task-1")
	if !strings.Contains(taskRef, "elementum_switch_task") {
		t.Errorf("Expected task ref to contain resource type, got: %s", taskRef)
	}

	// Test parent ref - task-1 should have trigger as parent
	parentRef := generator.GetParentRef("task-1")
	if !strings.Contains(parentRef, "trigger") {
		t.Errorf("Expected task-1 parent ref to be trigger, got: %s", parentRef)
	}

	// Test parent ref - task-2 should have task-1 as parent
	parentRef2 := generator.GetParentRef("task-2")
	if !strings.Contains(parentRef2, "switch_task") {
		t.Errorf("Expected task-2 parent ref to be switch_task, got: %s", parentRef2)
	}
}

func TestGenerateTaskResourcesOnly(t *testing.T) {
	app := &discovery.App{
		ID:   "app-123",
		Name: "Test App",
		Automations: []discovery.Automation{
			{
				ID:           "auto-1",
				Name:         "Test Automation",
				Status:       "ACTIVE",
				WorkflowID:   "workflow-1",
				HasPublished: true,
				Triggers: []discovery.Trigger{
					{
						ID:   "trigger-1",
						Type: "record_created",
					},
				},
				Tasks: []discovery.Task{
					{
						ID:         "task-1",
						Type:       "switch",
						Name:       "Did we find a record?",
						WorkflowID: "workflow-1",
						ParentID:   "trigger-1",
					},
				},
			},
		},
	}

	imports := []ImportBlock{
		{ID: "app-123", ResourceType: "elementum_app", ResourceName: "test_app"},
		{ID: "app-123:auto-1", ResourceType: "elementum_automation", ResourceName: "test_automation"},
		{ID: "auto-1:trigger-1", ResourceType: "elementum_record_created_trigger", ResourceName: "test_automation_record_created_0"},
		{ID: "workflow-1:task-1", ResourceType: "elementum_switch_task", ResourceName: "test_automation_did_we_find_a_record"},
	}

	generator := NewHCLGenerator(app, imports)
	hcl := generator.GenerateTaskResourcesOnly()

	// Check that the HCL contains the switch task
	if !strings.Contains(hcl, `resource "elementum_switch_task" "test_automation_did_we_find_a_record"`) {
		t.Errorf("Expected HCL to contain switch task resource, got:\n%s", hcl)
	}

	// Check that parent_id is set
	if !strings.Contains(hcl, "parent_id = elementum_record_created_trigger.test_automation_record_created_0.id") {
		t.Errorf("Expected HCL to have parent_id set to trigger, got:\n%s", hcl)
	}

	// Check that name is set
	if !strings.Contains(hcl, `name = "Did we find a record?"`) {
		t.Errorf("Expected HCL to have name set, got:\n%s", hcl)
	}
}

func TestTaskSpecificAttributes_RunAgent(t *testing.T) {
	app := &discovery.App{
		ID:   "app-123",
		Name: "Test App",
		Automations: []discovery.Automation{
			{
				ID:           "auto-1",
				Name:         "Test Automation",
				Status:       "ACTIVE",
				WorkflowID:   "workflow-1",
				HasPublished: true,
				Triggers: []discovery.Trigger{
					{
						ID:   "trigger-1",
						Type: "record_created",
					},
				},
				Tasks: []discovery.Task{
					{
						ID:         "task-1",
						Type:       "ai_agent",
						Name:       "Run My Agent",
						WorkflowID: "workflow-1",
						ParentID:   "trigger-1",
						RawData: map[string]interface{}{
							"id":         "task-1",
							"__typename": "WorkflowAiAgentTask",
							"name":       "Run My Agent",
							"agent": map[string]interface{}{
								"id":   "agent-123",
								"name": "Test Agent",
							},
							// Schema uses "agentPrompt" not "promptReference"
							"agentPrompt": map[string]interface{}{
								"id":    "ref-1",
								"label": "Analyze this record",
								"value": "Analyze ${trigger.record.Title}",
							},
							"outputType": "TEXT",
						},
					},
				},
			},
		},
	}

	imports := []ImportBlock{
		{ID: "app-123", ResourceType: "elementum_app", ResourceName: "test_app"},
		{ID: "app-123:auto-1", ResourceType: "elementum_automation", ResourceName: "test_automation"},
		{ID: "auto-1:trigger-1", ResourceType: "elementum_record_created_trigger", ResourceName: "test_automation_record_created_0"},
		{ID: "workflow-1:task-1", ResourceType: "elementum_ai_agent_task", ResourceName: "test_automation_run_my_agent"},
		{ID: "app-123:agent-123", ResourceType: "elementum_agent", ResourceName: "test_agent"},
	}

	generator := NewHCLGenerator(app, imports)
	hcl := generator.GenerateTaskResourcesOnly()

	// Check that the HCL contains the ai_agent task
	if !strings.Contains(hcl, `resource "elementum_ai_agent_task" "test_automation_run_my_agent"`) {
		t.Errorf("Expected HCL to contain ai_agent task resource, got:\n%s", hcl)
	}

	// Check that agent_id is resolved
	if !strings.Contains(hcl, "agent_id") {
		t.Errorf("Expected HCL to have agent_id, got:\n%s", hcl)
	}

	// Check that prompt is set from the value reference
	if !strings.Contains(hcl, "prompt") {
		t.Errorf("Expected HCL to have prompt, got:\n%s", hcl)
	}
}

func TestTaskSpecificAttributes_Variable(t *testing.T) {
	app := &discovery.App{
		ID:   "app-123",
		Name: "Test App",
		Automations: []discovery.Automation{
			{
				ID:           "auto-1",
				Name:         "Test Automation",
				Status:       "ACTIVE",
				WorkflowID:   "workflow-1",
				HasPublished: true,
				Triggers: []discovery.Trigger{
					{
						ID:   "trigger-1",
						Type: "record_created",
					},
				},
				Tasks: []discovery.Task{
					{
						ID:         "task-1",
						Type:       "variable",
						Name:       "Set My Variable",
						WorkflowID: "workflow-1",
						ParentID:   "trigger-1",
						RawData: map[string]interface{}{
							"id":         "task-1",
							"__typename": "WorkflowVariableTask",
							"name":       "Set My Variable",
							"variables": []interface{}{
								map[string]interface{}{
									"__typename": "WorkflowVariableTaskParameterCreate",
									"id":         "var-1",
									"name":       "my_var",
									"type":       "TEXT",
									// GraphQL query uses alias "createValue: value" so response has "createValue" key
									"createValue": map[string]interface{}{
										"id":    "ref-1",
										"label": "My Value",
										"value": "hello world",
									},
								},
							},
						},
					},
				},
			},
		},
	}

	imports := []ImportBlock{
		{ID: "app-123", ResourceType: "elementum_app", ResourceName: "test_app"},
		{ID: "app-123:auto-1", ResourceType: "elementum_automation", ResourceName: "test_automation"},
		{ID: "auto-1:trigger-1", ResourceType: "elementum_record_created_trigger", ResourceName: "test_automation_record_created_0"},
		{ID: "workflow-1:task-1", ResourceType: "elementum_variable_task", ResourceName: "test_automation_set_my_variable"},
	}

	generator := NewHCLGenerator(app, imports)
	hcl := generator.GenerateTaskResourcesOnly()

	// Check that the HCL contains the variable task
	if !strings.Contains(hcl, `resource "elementum_variable_task" "test_automation_set_my_variable"`) {
		t.Errorf("Expected HCL to contain variable task resource, got:\n%s", hcl)
	}

	// Check that variable_name is set
	if !strings.Contains(hcl, `variable_name = "my_var"`) {
		t.Errorf("Expected HCL to have variable_name, got:\n%s", hcl)
	}

	// Check that variable_type is set
	if !strings.Contains(hcl, `variable_type = "TEXT"`) {
		t.Errorf("Expected HCL to have variable_type, got:\n%s", hcl)
	}

	// Check that value is set
	if !strings.Contains(hcl, `value = "hello world"`) {
		t.Errorf("Expected HCL to have value, got:\n%s", hcl)
	}
}

func TestTriggerSpecificAttributes_Datamine(t *testing.T) {
	app := &discovery.App{
		ID:   "app-123",
		Name: "Test App",
		Automations: []discovery.Automation{
			{
				ID:           "auto-1",
				Name:         "Alert Automation",
				Status:       "ACTIVE",
				WorkflowID:   "workflow-1",
				HasPublished: true,
				Triggers: []discovery.Trigger{
					{
						ID:         "trigger-1",
						Type:       "datamine",
						DatamineID: "datamine-123",
						RawData: map[string]interface{}{
							"id":         "trigger-1",
							"__typename": "WorkflowDatamineTrigger",
							"datamine": map[string]interface{}{
								"id":   "datamine-123",
								"name": "Stale Records Check",
							},
							"fireOnAlert":    true,
							"fireOnRecovery": false,
						},
					},
				},
				Tasks: []discovery.Task{},
			},
		},
	}

	imports := []ImportBlock{
		{ID: "app-123", ResourceType: "elementum_app", ResourceName: "test_app"},
		{ID: "app-123:auto-1", ResourceType: "elementum_automation", ResourceName: "alert_automation"},
		{ID: "auto-1:trigger-1", ResourceType: "elementum_datamine_trigger", ResourceName: "alert_automation_datamine_0"},
		{ID: "app-123:datamine-123", ResourceType: "elementum_datamine", ResourceName: "stale_records_check"},
	}

	generator := NewHCLGenerator(app, imports)
	hcl := generator.GenerateTriggerResourcesOnly()

	// Check that the HCL contains the datamine trigger
	if !strings.Contains(hcl, `resource "elementum_datamine_trigger" "alert_automation_datamine_0"`) {
		t.Errorf("Expected HCL to contain datamine trigger resource, got:\n%s", hcl)
	}

	// Check that datamine_id is resolved to a reference
	if !strings.Contains(hcl, "datamine_id") {
		t.Errorf("Expected HCL to have datamine_id, got:\n%s", hcl)
	}

	// Check that fire_on_alert is set
	if !strings.Contains(hcl, "fire_on_alert = true") {
		t.Errorf("Expected HCL to have fire_on_alert = true, got:\n%s", hcl)
	}
}

func TestTriggerSpecificAttributes_OnDemand(t *testing.T) {
	app := &discovery.App{
		ID:   "app-123",
		Name: "Test App",
		Automations: []discovery.Automation{
			{
				ID:           "auto-1",
				Name:         "Manual Process",
				Status:       "ACTIVE",
				WorkflowID:   "workflow-1",
				HasPublished: true,
				Triggers: []discovery.Trigger{
					{
						ID:   "trigger-1",
						Type: "on_demand",
						RawData: map[string]interface{}{
							"id":         "trigger-1",
							"__typename": "WorkflowOnDemandTrigger",
							// Schema uses "parameters" not "inputs"
							"parameters": []interface{}{
								map[string]interface{}{
									"id":        "param-1",
									"name":      "customer_name",
									"fieldType": "TEXT",
									"required":  true,
									"multiple":  false,
								},
								map[string]interface{}{
									"id":        "param-2",
									"name":      "priority",
									"fieldType": "NUMBER",
									"required":  false,
									"multiple":  false,
								},
							},
						},
					},
				},
				Tasks: []discovery.Task{},
			},
		},
	}

	imports := []ImportBlock{
		{ID: "app-123", ResourceType: "elementum_app", ResourceName: "test_app"},
		{ID: "app-123:auto-1", ResourceType: "elementum_automation", ResourceName: "manual_process"},
		{ID: "auto-1:trigger-1", ResourceType: "elementum_on_demand_trigger", ResourceName: "manual_process_on_demand_0"},
	}

	generator := NewHCLGenerator(app, imports)
	hcl := generator.GenerateTriggerResourcesOnly()

	// Check that the HCL contains the on_demand trigger
	if !strings.Contains(hcl, `resource "elementum_on_demand_trigger" "manual_process_on_demand_0"`) {
		t.Errorf("Expected HCL to contain on_demand trigger resource, got:\n%s", hcl)
	}

	// Check that parameters are generated
	if !strings.Contains(hcl, "parameters") {
		t.Errorf("Expected HCL to have parameters, got:\n%s", hcl)
	}

	// Check that customer_name parameter is present
	if !strings.Contains(hcl, `name = "customer_name"`) {
		t.Errorf("Expected HCL to have customer_name parameter, got:\n%s", hcl)
	}

	// Check that required = true is set for customer_name
	if !strings.Contains(hcl, "required = true") {
		t.Errorf("Expected HCL to have required = true, got:\n%s", hcl)
	}
}

func TestInjectHydratedAppConfigs(t *testing.T) {
	app := &discovery.App{
		ID:        "app-123",
		Name:      "Velocity Activities",
		Namespace: "velocityactivities",
		Fields: []discovery.Field{
			{
				ID:           "field-status-123",
				Name:         "Status",
				Type:         "dropdown",
				SemanticTags: []string{"status"},
				Options: []discovery.FieldOption{
					{
						ID:    "opt-open-123",
						Label: "Open",
						Color: "#3B82F6",
						Tags:  []string{},
					},
					{
						ID:    "opt-closed-123",
						Label: "Closed",
						Color: "#10B981",
						Tags:  []string{"CLOSED"},
					},
				},
			},
			{
				ID:   "field-title-123",
				Name: "Title",
				Type: "text",
			},
		},
	}

	imports := []ImportBlock{
		{ID: "app-123", ResourceType: "elementum_app", ResourceName: "velocityactivities"},
	}

	// Simulate terraform-generated incomplete config (only namespace, missing status_options)
	inputHCL := `resource "elementum_app" "velocityactivities" {
  namespace = "velocityactivities"
}`

	result := InjectHydratedAppConfigs(inputHCL, app, imports)

	// Check that status_options is now present
	if !strings.Contains(result, "status_options") {
		t.Errorf("Expected status_options to be injected, got:\n%s", result)
	}

	// Check that Open option is present
	if !strings.Contains(result, `label = "Open"`) {
		t.Errorf("Expected Open status option, got:\n%s", result)
	}

	// Check that Closed option is present
	if !strings.Contains(result, `label = "Closed"`) {
		t.Errorf("Expected Closed status option, got:\n%s", result)
	}

	// Check that CLOSED tag is present
	if !strings.Contains(result, `tags = ["CLOSED"]`) {
		t.Errorf("Expected CLOSED tag on Closed option, got:\n%s", result)
	}

	// Check that Open has empty tags
	if !strings.Contains(result, "tags = []") {
		t.Errorf("Expected empty tags on Open option, got:\n%s", result)
	}

	// Check that colors are included
	if !strings.Contains(result, `color = "#3B82F6"`) {
		t.Errorf("Expected Open color, got:\n%s", result)
	}
	if !strings.Contains(result, `color = "#10B981"`) {
		t.Errorf("Expected Closed color, got:\n%s", result)
	}
}

func TestInjectHydratedAppConfigs_NoStatusField(t *testing.T) {
	// App without a status field should not have status_options injected
	app := &discovery.App{
		ID:        "app-456",
		Name:      "Simple App",
		Namespace: "simpleapp",
		Fields: []discovery.Field{
			{
				ID:   "field-title",
				Name: "Title",
				Type: "text",
			},
		},
	}

	imports := []ImportBlock{
		{ID: "app-456", ResourceType: "elementum_app", ResourceName: "simpleapp"},
	}

	inputHCL := `resource "elementum_app" "simpleapp" {
  namespace = "simpleapp"
}`

	result := InjectHydratedAppConfigs(inputHCL, app, imports)

	// Should not inject status_options if no status field exists
	if strings.Contains(result, "status_options") {
		t.Errorf("Expected no status_options for app without status field, got:\n%s", result)
	}
}

func TestInjectHydratedAppConfigs_AlreadyHasStatusOptions(t *testing.T) {
	app := &discovery.App{
		ID:        "app-789",
		Name:      "Existing App",
		Namespace: "existingapp",
		Fields: []discovery.Field{
			{
				ID:           "field-status",
				Name:         "Status",
				Type:         "dropdown",
				SemanticTags: []string{"status"},
				Options: []discovery.FieldOption{
					{ID: "opt-1", Label: "Open", Color: "#3B82F6", Tags: []string{}},
				},
			},
		},
	}

	imports := []ImportBlock{
		{ID: "app-789", ResourceType: "elementum_app", ResourceName: "existingapp"},
	}

	// Config already has status_options - should not double-inject
	inputHCL := `resource "elementum_app" "existingapp" {
  namespace = "existingapp"
  status_options = [
    { label = "Open", color = "#3B82F6", tags = [] }
  ]
}`

	result := InjectHydratedAppConfigs(inputHCL, app, imports)

	// Should not inject again since it already exists
	// Count occurrences of status_options - should only be 1
	count := strings.Count(result, "status_options")
	if count != 1 {
		t.Errorf("Expected exactly 1 status_options block, found %d in:\n%s", count, result)
	}
}

func TestInjectHydratedAppConfigs_DiscoveredApps(t *testing.T) {
	app := &discovery.App{
		ID:        "main-app",
		Name:      "Main App",
		Namespace: "mainapp",
		Fields: []discovery.Field{
			{
				ID:           "main-status",
				Name:         "Status",
				Type:         "dropdown",
				SemanticTags: []string{"status"},
				Options: []discovery.FieldOption{
					{ID: "opt-1", Label: "Open", Color: "#3B82F6", Tags: []string{}},
				},
			},
		},
		DiscoveredApps: []*discovery.App{
			{
				ID:        "discovered-app-1",
				Name:      "Cases Dev",
				Namespace: "casesdev",
				Fields: []discovery.Field{
					{
						ID:           "disc-status",
						Name:         "Status",
						Type:         "dropdown",
						SemanticTags: []string{"status"},
						Options: []discovery.FieldOption{
							{ID: "opt-a", Label: "New", Color: "#10B981", Tags: []string{}},
							{ID: "opt-b", Label: "Closed", Color: "#EF4444", Tags: []string{"closed"}},
						},
					},
				},
			},
			{
				ID:        "discovered-app-2",
				Name:      "Activities Dev",
				Namespace: "activitiesdev",
				Fields: []discovery.Field{
					{
						ID:           "disc2-status",
						Name:         "Status",
						Type:         "dropdown",
						SemanticTags: []string{"status"},
						Options: []discovery.FieldOption{
							{ID: "opt-x", Label: "Active", Color: "#3B82F6", Tags: []string{}},
						},
					},
				},
			},
		},
	}

	imports := []ImportBlock{
		{ID: "main-app", ResourceType: "elementum_app", ResourceName: "mainapp"},
		{ID: "discovered-app-1", ResourceType: "elementum_app", ResourceName: "casesdev"},
		{ID: "discovered-app-2", ResourceType: "elementum_app", ResourceName: "activitiesdev"},
	}

	inputHCL := `resource "elementum_app" "mainapp" {
  namespace = "mainapp"
}

resource "elementum_app" "casesdev" {
  namespace = "casesdev"
}

resource "elementum_app" "activitiesdev" {
  namespace = "activitiesdev"
}`

	result := InjectHydratedAppConfigs(inputHCL, app, imports)

	// All 3 apps should have status_options injected
	if strings.Count(result, "status_options") != 3 {
		t.Errorf("Expected 3 status_options blocks, found %d in:\n%s", strings.Count(result, "status_options"), result)
	}

	// Check specific labels
	if !strings.Contains(result, `label = "Open"`) {
		t.Error("Main app status option 'Open' not found")
	}
	if !strings.Contains(result, `label = "New"`) {
		t.Error("Discovered app 'Cases Dev' status option 'New' not found")
	}
	if !strings.Contains(result, `label = "Active"`) {
		t.Error("Discovered app 'Activities Dev' status option 'Active' not found")
	}
}

func TestExtractValueFromPath(t *testing.T) {
	tests := []struct {
		name     string
		data     map[string]interface{}
		path     string
		expected interface{}
	}{
		{
			name: "simple key",
			data: map[string]interface{}{
				"name": "test",
			},
			path:     "name",
			expected: "test",
		},
		{
			name: "nested key",
			data: map[string]interface{}{
				"agent": map[string]interface{}{
					"id": "agent-123",
				},
			},
			path:     "agent.id",
			expected: "agent-123",
		},
		{
			name: "array access",
			data: map[string]interface{}{
				"variables": []interface{}{
					map[string]interface{}{
						"name": "var1",
						"type": "TEXT",
					},
				},
			},
			path:     "variables[0].name",
			expected: "var1",
		},
		{
			name: "missing key",
			data: map[string]interface{}{
				"name": "test",
			},
			path:     "missing",
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractValueFromPath(tt.data, tt.path)
			if result != tt.expected {
				t.Errorf("extractValueFromPath(%v, %q) = %v, want %v", tt.data, tt.path, result, tt.expected)
			}
		})
	}
}

func TestInjectLockStagesForApp(t *testing.T) {
	// App with LockStages = true should have lock_stages injected
	app := &discovery.App{
		ID:         "app-123",
		Name:       "Test App",
		Namespace:  "testapp",
		LockStages: true,
	}

	imports := []ImportBlock{
		{ID: "app-123", ResourceType: "elementum_app", ResourceName: "testapp"},
	}

	inputHCL := `resource "elementum_app" "testapp" {
  namespace = "testapp"
}`

	result := InjectHydratedAppConfigs(inputHCL, app, imports)

	// Check that lock_stages is now present
	if !strings.Contains(result, "lock_stages = true") {
		t.Errorf("Expected lock_stages = true to be injected, got:\n%s", result)
	}
}

func TestInjectLockStagesForApp_FalseNotInjected(t *testing.T) {
	// App with LockStages = false should NOT have lock_stages injected (it's the default)
	app := &discovery.App{
		ID:         "app-456",
		Name:       "Test App",
		Namespace:  "testapp2",
		LockStages: false,
	}

	imports := []ImportBlock{
		{ID: "app-456", ResourceType: "elementum_app", ResourceName: "testapp2"},
	}

	inputHCL := `resource "elementum_app" "testapp2" {
  namespace = "testapp2"
}`

	result := InjectHydratedAppConfigs(inputHCL, app, imports)

	// Check that lock_stages is NOT present (default is false)
	if strings.Contains(result, "lock_stages") {
		t.Errorf("Expected lock_stages NOT to be injected when false, got:\n%s", result)
	}
}

func TestInjectLockStagesForApp_AlreadyHasLockStages(t *testing.T) {
	// If lock_stages already exists, don't inject again
	app := &discovery.App{
		ID:         "app-789",
		Name:       "Test App",
		Namespace:  "testapp3",
		LockStages: true,
	}

	imports := []ImportBlock{
		{ID: "app-789", ResourceType: "elementum_app", ResourceName: "testapp3"},
	}

	inputHCL := `resource "elementum_app" "testapp3" {
  namespace   = "testapp3"
  lock_stages = true
}`

	result := InjectHydratedAppConfigs(inputHCL, app, imports)

	// Check that there's still only ONE lock_stages
	if strings.Count(result, "lock_stages") != 1 {
		t.Errorf("Expected exactly 1 lock_stages, found %d in:\n%s", strings.Count(result, "lock_stages"), result)
	}
}

func TestInjectLockStagesForApp_WithDiscoveredApps(t *testing.T) {
	// Test that lock_stages is injected for discovered apps as well
	mainApp := &discovery.App{
		ID:         "app-main",
		Name:       "Main App",
		Namespace:  "mainapp",
		LockStages: true,
		DiscoveredApps: []*discovery.App{
			{
				ID:         "app-discovered",
				Name:       "Discovered App",
				Namespace:  "discoveredapp",
				LockStages: true,
			},
		},
	}

	imports := []ImportBlock{
		{ID: "app-main", ResourceType: "elementum_app", ResourceName: "mainapp"},
		{ID: "app-discovered", ResourceType: "elementum_app", ResourceName: "discoveredapp"},
	}

	inputHCL := `resource "elementum_app" "mainapp" {
  namespace = "mainapp"
}

resource "elementum_app" "discoveredapp" {
  namespace = "discoveredapp"
}`

	result := InjectHydratedAppConfigs(inputHCL, mainApp, imports)

	// Both apps should have lock_stages injected
	if strings.Count(result, "lock_stages = true") != 2 {
		t.Errorf("Expected 2 lock_stages = true blocks, found %d in:\n%s", strings.Count(result, "lock_stages = true"), result)
	}
}

// TestGenerateWorkflowPublishIR tests that workflow_publish resources are generated
// with correct automation_id and task_ids dependencies.
func TestGenerateWorkflowPublishIR(t *testing.T) {
	app := &discovery.App{
		ID:        "app-123",
		Name:      "Test App",
		Namespace: "test_app",
		Automations: []discovery.Automation{
			{
				ID:           "auto-1",
				Name:         "Process Order",
				Status:       "ACTIVE",
				WorkflowID:   "workflow-1",
				HasPublished: true,
				Triggers: []discovery.Trigger{
					{
						ID:   "trigger-1",
						Type: "record_created",
					},
				},
				Tasks: []discovery.Task{
					{
						ID:         "task-1",
						Type:       "variable",
						Name:       "Set Status",
						WorkflowID: "workflow-1",
						ParentID:   "trigger-1",
					},
					{
						ID:         "task-2",
						Type:       "update_field",
						Name:       "Update Order",
						WorkflowID: "workflow-1",
						ParentID:   "task-1",
					},
				},
			},
		},
	}

	imports := []ImportBlock{
		{ID: "app-123", ResourceType: "elementum_app", ResourceName: "test_app"},
		{ID: "app-123:auto-1", ResourceType: "elementum_automation", ResourceName: "process_order"},
		{ID: "auto-1:trigger-1", ResourceType: "elementum_record_created_trigger", ResourceName: "process_order_record_created_0"},
		{ID: "workflow-1:task-1", ResourceType: "elementum_variable_task", ResourceName: "process_order_set_status"},
		{ID: "workflow-1:task-2", ResourceType: "elementum_update_field_task", ResourceName: "process_order_update_order"},
		{ID: "auto-1", ResourceType: "elementum_workflow_publish", ResourceName: "process_order"},
	}

	generator := NewAutomationHCLGenerator(app, imports, nil)
	blocks := generator.GenerateWorkflowPublishIR()

	if len(blocks) != 1 {
		t.Fatalf("Expected 1 workflow_publish block, got %d", len(blocks))
	}

	block := blocks[0]

	// Check resource type and name
	if block.Labels[0] != "elementum_workflow_publish" {
		t.Errorf("Expected resource type elementum_workflow_publish, got %s", block.Labels[0])
	}
	if block.Labels[1] != "process_order" {
		t.Errorf("Expected resource name process_order, got %s", block.Labels[1])
	}

	// Serialize the block and check contents
	serialized := SerializeBlock(block, 0)

	// Check automation_id attribute
	if !strings.Contains(serialized, "automation_id = elementum_automation.process_order.id") {
		t.Errorf("Expected automation_id to reference automation, got:\n%s", serialized)
	}

	// Check task_ids attribute contains all task references
	if !strings.Contains(serialized, "elementum_variable_task.process_order_set_status.id") {
		t.Errorf("Expected task_ids to contain variable_task ref, got:\n%s", serialized)
	}
	if !strings.Contains(serialized, "elementum_update_field_task.process_order_update_order.id") {
		t.Errorf("Expected task_ids to contain update_field_task ref, got:\n%s", serialized)
	}
}

// TestGenerateWorkflowPublishIR_IncludedInGenerateAllIR verifies that workflow_publish
// resources are included when generating all automation resources.
func TestGenerateWorkflowPublishIR_IncludedInGenerateAllIR(t *testing.T) {
	app := &discovery.App{
		ID:        "app-123",
		Name:      "Test App",
		Namespace: "test_app",
		Automations: []discovery.Automation{
			{
				ID:           "auto-1",
				Name:         "My Automation",
				Status:       "ACTIVE",
				WorkflowID:   "workflow-1",
				HasPublished: true,
				Triggers: []discovery.Trigger{
					{ID: "trigger-1", Type: "record_created"},
				},
				Tasks: []discovery.Task{
					{ID: "task-1", Type: "variable", Name: "Set Var", WorkflowID: "workflow-1", ParentID: "trigger-1"},
				},
			},
		},
	}

	imports := []ImportBlock{
		{ID: "app-123", ResourceType: "elementum_app", ResourceName: "test_app"},
		{ID: "app-123:auto-1", ResourceType: "elementum_automation", ResourceName: "my_automation"},
		{ID: "auto-1:trigger-1", ResourceType: "elementum_record_created_trigger", ResourceName: "my_automation_record_created_0"},
		{ID: "workflow-1:task-1", ResourceType: "elementum_variable_task", ResourceName: "my_automation_set_var"},
		{ID: "auto-1", ResourceType: "elementum_workflow_publish", ResourceName: "my_automation"},
	}

	generator := NewAutomationHCLGenerator(app, imports, nil)
	allBlocks := generator.GenerateAllIR()

	// Find workflow_publish block
	var publishBlock *HCLBlock
	for _, b := range allBlocks {
		if len(b.Labels) >= 1 && b.Labels[0] == "elementum_workflow_publish" {
			publishBlock = b
			break
		}
	}

	if publishBlock == nil {
		t.Fatal("Expected workflow_publish block in GenerateAllIR output")
	}
}

// TestGenerateWorkflowPublishIR_WithOutputs tests that on-demand automations
// have their outputs exported in the workflow_publish resource.
func TestGenerateWorkflowPublishIR_WithOutputs(t *testing.T) {
	app := &discovery.App{
		ID:        "app-123",
		Name:      "Test App",
		Namespace: "test_app",
		Automations: []discovery.Automation{
			{
				ID:           "auto-1",
				Name:         "Search Knowledge Base",
				Status:       "ACTIVE",
				WorkflowID:   "workflow-1",
				HasPublished: true,
				Triggers: []discovery.Trigger{
					{
						ID:   "trigger-1",
						Type: "on_demand",
						Name: "On Demand Trigger",
					},
				},
				Tasks: []discovery.Task{
					{
						ID:         "task-1",
						Type:       "record_search",
						Name:       "Search Records",
						WorkflowID: "workflow-1",
						ParentID:   "trigger-1",
					},
				},
				Outputs: []discovery.WorkflowOutput{
					{
						Name: "count",
						Value: map[string]interface{}{
							"taskReference": map[string]interface{}{
								"name": "task.task-1.Records Size",
							},
						},
					},
					{
						Name: "results",
						Value: map[string]interface{}{
							"taskReference": map[string]interface{}{
								"name": "task.task-1.All found records",
							},
						},
					},
				},
			},
		},
	}

	imports := []ImportBlock{
		{ID: "app-123", ResourceType: "elementum_app", ResourceName: "test_app"},
		{ID: "app-123:auto-1", ResourceType: "elementum_automation", ResourceName: "search_knowledge_base"},
		{ID: "workflow-1:trigger-1", ResourceType: "elementum_on_demand_trigger", ResourceName: "search_knowledge_base_on_demand_0"},
		{ID: "workflow-1:task-1", ResourceType: "elementum_record_search_task", ResourceName: "search_knowledge_base_search_records"},
		{ID: "auto-1", ResourceType: "elementum_workflow_publish", ResourceName: "search_knowledge_base"},
	}

	generator := NewAutomationHCLGenerator(app, imports, nil)
	blocks := generator.GenerateWorkflowPublishIR()

	if len(blocks) != 1 {
		t.Fatalf("Expected 1 workflow_publish block, got %d", len(blocks))
	}

	block := blocks[0]
	serialized := SerializeBlock(block, 0)

	// Check that outputs are present
	if !strings.Contains(serialized, "outputs") {
		t.Errorf("Expected outputs attribute in workflow_publish, got:\n%s", serialized)
	}

	// Check output names
	if !strings.Contains(serialized, `name = "count"`) {
		t.Errorf("Expected output name 'count', got:\n%s", serialized)
	}
	if !strings.Contains(serialized, `name = "results"`) {
		t.Errorf("Expected output name 'results', got:\n%s", serialized)
	}

	// Check output values reference tasks correctly
	if !strings.Contains(serialized, `elementum_record_search_task.search_knowledge_base_search_records.refs["Records Size"]`) {
		t.Errorf("Expected count output to reference task refs, got:\n%s", serialized)
	}
	if !strings.Contains(serialized, `elementum_record_search_task.search_knowledge_base_search_records.refs["All found records"]`) {
		t.Errorf("Expected results output to reference task refs, got:\n%s", serialized)
	}
}
