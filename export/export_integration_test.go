// Copyright (c) ElementumAI, Inc.
// SPDX-License-Identifier: MPL-2.0

package export

import (
	"strings"
	"testing"

	"github.com/elementumltd/elementum-cli/discovery"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// Full Pipeline Integration Tests
// =============================================================================

func TestExportPipeline_SimpleNotification(t *testing.T) {
	// Create a simple automation: record_created trigger -> message task
	app := createTestApp(TestAppID, "Test App")

	trigger := discovery.Trigger{
		ID:   TestTriggerID,
		Type: "record_created",
		Name: "On Record Created",
		FieldRefs: map[string]string{
			"record.title": "Title",
		},
		RawData: map[string]any{
			"__typename": "WorkflowRecordCreateTrigger",
			"id":         TestTriggerID,
			"name":       "On Record Created",
		},
	}

	task := discovery.Task{
		ID:       TestTaskID,
		Type:     "message",
		Name:     "Send Notification",
		ParentID: TestTriggerID,
		FieldRefs: map[string]string{
			"text": "Message Text",
		},
		RawData: map[string]any{
			"__typename":        "WorkflowMessageTask",
			"id":                TestTaskID,
			"name":              "Send Notification",
			"contentsReference": map[string]any{"triggerReference": map[string]any{"name": "record.title"}},
		},
	}

	automation := discovery.Automation{
		ID:           TestAutomationID,
		Name:         "Simple Notification",
		Status:       "ACTIVE",
		WorkflowID:   TestWorkflowID,
		HasPublished: true,
		Triggers:     []discovery.Trigger{trigger},
		Tasks:        []discovery.Task{task},
	}
	app.Automations = []discovery.Automation{automation}

	imports := []ImportBlock{
		{ResourceType: "elementum_app", ResourceName: "test_app", ID: TestAppID},
		{ResourceType: "elementum_automation", ResourceName: "simple_notification", ID: TestAutomationID},
		{ResourceType: "elementum_record_created_trigger", ResourceName: "on_record_created", ID: TestAutomationID + ":" + TestTriggerID},
		{ResourceType: "elementum_message_task", ResourceName: "send_notification", ID: TestWorkflowID + ":" + TestTaskID},
	}

	// Generate HCL
	gen := NewHCLGenerator(app, imports)
	triggerHCL := gen.GenerateTriggerResourcesOnly()
	taskHCL := gen.GenerateTaskResourcesOnly()

	// Verify trigger HCL
	assert.Contains(t, triggerHCL, `resource "elementum_record_created_trigger"`)
	assert.Contains(t, triggerHCL, `"on_record_created"`)

	// Verify task HCL
	assert.Contains(t, taskHCL, `resource "elementum_message_task"`)
	assert.Contains(t, taskHCL, `"send_notification"`)

	// Validate no null attributes in output
	assertNoNullAttributes(t, triggerHCL)
	assertNoNullAttributes(t, taskHCL)
}

func TestExportPipeline_MultiTriggerWorkflow(t *testing.T) {
	// Automation with multiple triggers
	app := createTestApp(TestAppID, "Test App")

	trigger1 := discovery.Trigger{
		ID:   TestTriggerID,
		Type: "record_created",
		Name: "On Create",
		RawData: map[string]any{
			"__typename": "WorkflowRecordCreateTrigger",
			"id":         TestTriggerID,
			"name":       "On Create",
		},
	}

	trigger2ID := "55555555-6666-7777-8888-999999999999"
	trigger2 := discovery.Trigger{
		ID:   trigger2ID,
		Type: "record_updated",
		Name: "On Update",
		RawData: map[string]any{
			"__typename": "WorkflowRecordUpdateTrigger",
			"id":         trigger2ID,
			"name":       "On Update",
		},
	}

	task := discovery.Task{
		ID:       TestTaskID,
		Type:     "notification",
		Name:     "Notify",
		ParentID: TestTriggerID,
		RawData: map[string]any{
			"__typename": "WorkflowNotificationTask",
			"id":         TestTaskID,
			"name":       "Notify",
			"message":    map[string]any{"staticValue": "Record changed"},
		},
	}

	automation := discovery.Automation{
		ID:           TestAutomationID,
		Name:         "Multi Trigger",
		Status:       "ACTIVE",
		WorkflowID:   TestWorkflowID,
		HasPublished: true,
		Triggers:     []discovery.Trigger{trigger1, trigger2},
		Tasks:        []discovery.Task{task},
	}
	app.Automations = []discovery.Automation{automation}

	imports := []ImportBlock{
		{ResourceType: "elementum_app", ResourceName: "test_app", ID: TestAppID},
		{ResourceType: "elementum_automation", ResourceName: "multi_trigger", ID: TestAutomationID},
		{ResourceType: "elementum_record_created_trigger", ResourceName: "on_create", ID: TestAutomationID + ":" + TestTriggerID},
		{ResourceType: "elementum_record_updated_trigger", ResourceName: "on_update", ID: TestAutomationID + ":" + trigger2ID},
		{ResourceType: "elementum_notification_task", ResourceName: "notify", ID: TestWorkflowID + ":" + TestTaskID},
	}

	gen := NewHCLGenerator(app, imports)
	triggerHCL := gen.GenerateTriggerResourcesOnly()

	// Should have both trigger types
	assert.Contains(t, triggerHCL, `"elementum_record_created_trigger"`)
	assert.Contains(t, triggerHCL, `"elementum_record_updated_trigger"`)
	assert.Contains(t, triggerHCL, `"on_create"`)
	assert.Contains(t, triggerHCL, `"on_update"`)
}

func TestExportPipeline_ComplexTaskChain(t *testing.T) {
	// Automation with a chain of tasks: trigger -> task1 -> task2 -> task3
	app := createTestApp(TestAppID, "Test App")

	trigger := discovery.Trigger{
		ID:   TestTriggerID,
		Type: "record_created",
		Name: "Trigger",
		RawData: map[string]any{
			"__typename": "WorkflowRecordCreateTrigger",
			"id":         TestTriggerID,
			"name":       "Trigger",
		},
	}

	task1ID := TestTaskID
	task2ID := "task-2-uuid-1234-5678-abcdefabcdef"
	task3ID := "task-3-uuid-1234-5678-abcdefabcdef"

	task1 := discovery.Task{
		ID:       task1ID,
		Type:     "variable",
		Name:     "Set Variable",
		ParentID: TestTriggerID,
		RawData: map[string]any{
			"__typename":   "WorkflowVariableTask",
			"id":           task1ID,
			"name":         "Set Variable",
			"variableName": "counter",
			"variableType": "NUMBER",
			"value":        map[string]any{"staticValue": "1"},
		},
	}

	task2 := discovery.Task{
		ID:       task2ID,
		Type:     "calculation",
		Name:     "Calculate",
		ParentID: task1ID,
		RawData: map[string]any{
			"__typename": "WorkflowCalculationTask",
			"id":         task2ID,
			"name":       "Calculate",
			"calculations": []any{
				map[string]any{
					"name":                 "result",
					"calculationReference": map[string]any{"staticValue": "counter + 1"},
				},
			},
		},
	}

	task3 := discovery.Task{
		ID:       task3ID,
		Type:     "message",
		Name:     "Final Message",
		ParentID: task2ID,
		RawData: map[string]any{
			"__typename":        "WorkflowMessageTask",
			"id":                task3ID,
			"name":              "Final Message",
			"contentsReference": map[string]any{"staticValue": "Done"},
		},
	}

	automation := discovery.Automation{
		ID:           TestAutomationID,
		Name:         "Task Chain",
		Status:       "ACTIVE",
		WorkflowID:   TestWorkflowID,
		HasPublished: true,
		Triggers:     []discovery.Trigger{trigger},
		Tasks:        []discovery.Task{task1, task2, task3},
	}
	app.Automations = []discovery.Automation{automation}

	imports := []ImportBlock{
		{ResourceType: "elementum_app", ResourceName: "test_app", ID: TestAppID},
		{ResourceType: "elementum_automation", ResourceName: "task_chain", ID: TestAutomationID},
		{ResourceType: "elementum_record_created_trigger", ResourceName: "trigger", ID: TestAutomationID + ":" + TestTriggerID},
		{ResourceType: "elementum_variable_task", ResourceName: "set_variable", ID: TestWorkflowID + ":" + task1ID},
		{ResourceType: "elementum_calculation_task", ResourceName: "calculate", ID: TestWorkflowID + ":" + task2ID},
		{ResourceType: "elementum_message_task", ResourceName: "final_message", ID: TestWorkflowID + ":" + task3ID},
	}

	gen := NewHCLGenerator(app, imports)
	taskHCL := gen.GenerateTaskResourcesOnly()

	// Should have all task types
	assert.Contains(t, taskHCL, `"elementum_variable_task"`)
	assert.Contains(t, taskHCL, `"elementum_calculation_task"`)
	assert.Contains(t, taskHCL, `"elementum_message_task"`)

	// Should have parent references in the chain
	assert.Contains(t, taskHCL, `parent_id`)

	// Verify no null attributes
	assertNoNullAttributes(t, taskHCL)
}

func TestExportPipeline_BranchingWorkflow_Switch(t *testing.T) {
	// Automation with switch task for branching
	app := createTestApp(TestAppID, "Test App")

	trigger := discovery.Trigger{
		ID:   TestTriggerID,
		Type: "record_created",
		Name: "Trigger",
		RawData: map[string]any{
			"__typename": "WorkflowRecordCreateTrigger",
			"id":         TestTriggerID,
			"name":       "Trigger",
		},
	}

	switchTask := discovery.Task{
		ID:       TestTaskID,
		Type:     "switch",
		Name:     "Branch",
		ParentID: TestTriggerID,
		RawData: map[string]any{
			"__typename": "WorkflowSwitchTask",
			"id":         TestTaskID,
			"name":       "Branch",
		},
	}

	automation := discovery.Automation{
		ID:           TestAutomationID,
		Name:         "Branching",
		Status:       "ACTIVE",
		WorkflowID:   TestWorkflowID,
		HasPublished: true,
		Triggers:     []discovery.Trigger{trigger},
		Tasks:        []discovery.Task{switchTask},
	}
	app.Automations = []discovery.Automation{automation}

	imports := []ImportBlock{
		{ResourceType: "elementum_app", ResourceName: "test_app", ID: TestAppID},
		{ResourceType: "elementum_automation", ResourceName: "branching", ID: TestAutomationID},
		{ResourceType: "elementum_record_created_trigger", ResourceName: "trigger", ID: TestAutomationID + ":" + TestTriggerID},
		{ResourceType: "elementum_switch_task", ResourceName: "branch", ID: TestWorkflowID + ":" + TestTaskID},
	}

	gen := NewHCLGenerator(app, imports)
	taskHCL := gen.GenerateTaskResourcesOnly()

	assert.Contains(t, taskHCL, `"elementum_switch_task"`)
	assert.Contains(t, taskHCL, `"branch"`)
}

func TestExportPipeline_LoopingWorkflow_ForEach(t *testing.T) {
	// Automation with for_each task for looping
	app := createTestApp(TestAppID, "Test App")

	trigger := discovery.Trigger{
		ID:   TestTriggerID,
		Type: "record_created",
		Name: "Trigger",
		RawData: map[string]any{
			"__typename": "WorkflowRecordCreateTrigger",
			"id":         TestTriggerID,
			"name":       "Trigger",
		},
	}

	searchTask := discovery.Task{
		ID:       TestTaskID,
		Type:     "record_search",
		Name:     "Find Records",
		ParentID: TestTriggerID,
		RawData: map[string]any{
			"__typename": "WorkflowRecordSearchTask",
			"id":         TestTaskID,
			"name":       "Find Records",
			"aspect":     map[string]any{"id": TestAppID},
		},
	}

	forEachID := "foreach-task-uuid-1234-5678-abcdef"
	forEachTask := discovery.Task{
		ID:       forEachID,
		Type:     "for_each",
		Name:     "Loop Records",
		ParentID: TestTaskID,
		RawData: map[string]any{
			"__typename": "WorkflowForEachTask",
			"id":         forEachID,
			"name":       "Loop Records",
			"forEach":    map[string]any{"taskReference": map[string]any{"taskId": TestTaskID, "name": "All found records"}},
		},
	}

	automation := discovery.Automation{
		ID:           TestAutomationID,
		Name:         "Looping",
		Status:       "ACTIVE",
		WorkflowID:   TestWorkflowID,
		HasPublished: true,
		Triggers:     []discovery.Trigger{trigger},
		Tasks:        []discovery.Task{searchTask, forEachTask},
	}
	app.Automations = []discovery.Automation{automation}

	imports := []ImportBlock{
		{ResourceType: "elementum_app", ResourceName: "test_app", ID: TestAppID},
		{ResourceType: "elementum_automation", ResourceName: "looping", ID: TestAutomationID},
		{ResourceType: "elementum_record_created_trigger", ResourceName: "trigger", ID: TestAutomationID + ":" + TestTriggerID},
		{ResourceType: "elementum_record_search_task", ResourceName: "find_records", ID: TestWorkflowID + ":" + TestTaskID},
		{ResourceType: "elementum_for_each_task", ResourceName: "loop_records", ID: TestWorkflowID + ":" + forEachID},
	}

	gen := NewHCLGenerator(app, imports)
	taskHCL := gen.GenerateTaskResourcesOnly()

	assert.Contains(t, taskHCL, `"elementum_record_search_task"`)
	assert.Contains(t, taskHCL, `"elementum_for_each_task"`)
}

// =============================================================================
// HCL Validation Tests
// =============================================================================

func TestHCLValidation_NoSyntaxErrors(t *testing.T) {
	// Generate HCL and verify it has proper structure
	app := createTestApp(TestAppID, "Test App")

	trigger := discovery.Trigger{
		ID:   TestTriggerID,
		Type: "record_created",
		Name: "Test Trigger",
		RawData: map[string]any{
			"__typename": "WorkflowRecordCreateTrigger",
			"id":         TestTriggerID,
			"name":       "Test Trigger",
		},
	}

	automation := discovery.Automation{
		ID:           TestAutomationID,
		Name:         "Test Automation",
		Status:       "ACTIVE",
		WorkflowID:   TestWorkflowID,
		HasPublished: true,
		Triggers:     []discovery.Trigger{trigger},
		Tasks:        []discovery.Task{},
	}
	app.Automations = []discovery.Automation{automation}

	imports := []ImportBlock{
		{ResourceType: "elementum_automation", ResourceName: "test", ID: TestAutomationID},
		{ResourceType: "elementum_record_created_trigger", ResourceName: "test_trigger", ID: TestAutomationID + ":" + TestTriggerID},
	}

	gen := NewHCLGenerator(app, imports)
	hcl := gen.GenerateTriggerResourcesOnly()

	// Basic syntax validation
	validateHCLSyntax(t, hcl)
}

func TestHCLValidation_ProperFormatting(t *testing.T) {
	app := createTestApp(TestAppID, "Test App")

	trigger := discovery.Trigger{
		ID:   TestTriggerID,
		Type: "webhook",
		Name: "Test Webhook",
		RawData: map[string]any{
			"__typename":    "WorkflowWebhookTrigger",
			"id":            TestTriggerID,
			"name":          "Test Webhook",
			"authenticated": true,
		},
	}

	automation := discovery.Automation{
		ID:           TestAutomationID,
		Name:         "Test Automation",
		Status:       "ACTIVE",
		WorkflowID:   TestWorkflowID,
		HasPublished: true,
		Triggers:     []discovery.Trigger{trigger},
		Tasks:        []discovery.Task{},
	}
	app.Automations = []discovery.Automation{automation}

	imports := []ImportBlock{
		{ResourceType: "elementum_automation", ResourceName: "test", ID: TestAutomationID},
		{ResourceType: "elementum_webhook_trigger", ResourceName: "test_webhook", ID: TestAutomationID + ":" + TestTriggerID},
	}

	gen := NewHCLGenerator(app, imports)
	hcl := gen.GenerateTriggerResourcesOnly()

	// Should have proper resource block structure
	assert.Contains(t, hcl, "resource")
	assert.Contains(t, hcl, "{")
	assert.Contains(t, hcl, "}")

	// Should have proper attribute formatting (key = value)
	assert.Contains(t, hcl, " = ")
}

func TestHCLValidation_NoNullAttributes(t *testing.T) {
	app := createTestApp(TestAppID, "Test App")

	trigger := discovery.Trigger{
		ID:   TestTriggerID,
		Type: "webhook",
		Name: "Test Webhook",
		RawData: map[string]any{
			"__typename":    "WorkflowWebhookTrigger",
			"id":            TestTriggerID,
			"name":          "Test Webhook",
			"authenticated": false,
			"url":           "https://example.com/webhook",
		},
	}

	automation := discovery.Automation{
		ID:           TestAutomationID,
		Name:         "Test Automation",
		Status:       "ACTIVE",
		WorkflowID:   TestWorkflowID,
		HasPublished: true,
		Triggers:     []discovery.Trigger{trigger},
		Tasks:        []discovery.Task{},
	}
	app.Automations = []discovery.Automation{automation}

	imports := []ImportBlock{
		{ResourceType: "elementum_automation", ResourceName: "test", ID: TestAutomationID},
		{ResourceType: "elementum_webhook_trigger", ResourceName: "test_webhook", ID: TestAutomationID + ":" + TestTriggerID},
	}

	gen := NewHCLGenerator(app, imports)
	hcl := gen.GenerateTriggerResourcesOnly()

	// Should not have null attribute values
	assertNoNullAttributes(t, hcl)
}

func TestHCLValidation_BalancedBraces(t *testing.T) {
	app := createTestApp(TestAppID, "Test App")

	task := discovery.Task{
		ID:   TestTaskID,
		Type: "api",
		Name: "API Call",
		RawData: map[string]any{
			"__typename": "WorkflowApiTask",
			"id":         TestTaskID,
			"name":       "API Call",
			"method":     "POST",
			"urlReference": map[string]any{
				"staticValue": "https://api.example.com",
			},
			"body": map[string]any{
				"__typename": "ApiJsonBody",
				"json": map[string]any{
					"staticValue": `{"key": "value"}`,
				},
			},
		},
	}

	trigger := discovery.Trigger{
		ID:   TestTriggerID,
		Type: "record_created",
		Name: "Trigger",
		RawData: map[string]any{
			"__typename": "WorkflowRecordCreateTrigger",
			"id":         TestTriggerID,
			"name":       "Trigger",
		},
	}

	automation := discovery.Automation{
		ID:           TestAutomationID,
		Name:         "Test Automation",
		Status:       "ACTIVE",
		WorkflowID:   TestWorkflowID,
		HasPublished: true,
		Triggers:     []discovery.Trigger{trigger},
		Tasks:        []discovery.Task{task},
	}
	app.Automations = []discovery.Automation{automation}

	imports := []ImportBlock{
		{ResourceType: "elementum_automation", ResourceName: "test", ID: TestAutomationID},
		{ResourceType: "elementum_record_created_trigger", ResourceName: "trigger", ID: TestAutomationID + ":" + TestTriggerID},
		{ResourceType: "elementum_api_task", ResourceName: "api_call", ID: TestWorkflowID + ":" + TestTaskID},
	}

	gen := NewHCLGenerator(app, imports)
	hcl := gen.GenerateTaskResourcesOnly()

	// Verify balanced braces
	openBraces := strings.Count(hcl, "{")
	closeBraces := strings.Count(hcl, "}")
	assert.Equal(t, openBraces, closeBraces, "Braces should be balanced")

	// Verify balanced brackets
	openBrackets := strings.Count(hcl, "[")
	closeBrackets := strings.Count(hcl, "]")
	assert.Equal(t, openBrackets, closeBrackets, "Brackets should be balanced")
}

func TestHCLValidation_ValidResourceNames(t *testing.T) {
	app := createTestApp(TestAppID, "Test App")

	// Create trigger with special characters in name that should be sanitized
	trigger := discovery.Trigger{
		ID:   TestTriggerID,
		Type: "record_created",
		Name: "On Record Created (Test)",
		RawData: map[string]any{
			"__typename": "WorkflowRecordCreateTrigger",
			"id":         TestTriggerID,
			"name":       "On Record Created (Test)",
		},
	}

	automation := discovery.Automation{
		ID:           TestAutomationID,
		Name:         "Test Automation",
		Status:       "ACTIVE",
		WorkflowID:   TestWorkflowID,
		HasPublished: true,
		Triggers:     []discovery.Trigger{trigger},
		Tasks:        []discovery.Task{},
	}
	app.Automations = []discovery.Automation{automation}

	// Use sanitized resource name
	imports := []ImportBlock{
		{ResourceType: "elementum_automation", ResourceName: "test_automation", ID: TestAutomationID},
		{ResourceType: "elementum_record_created_trigger", ResourceName: "on_record_created_test", ID: TestAutomationID + ":" + TestTriggerID},
	}

	gen := NewHCLGenerator(app, imports)
	hcl := gen.GenerateTriggerResourcesOnly()

	// Resource names should be valid Terraform identifiers
	assert.Contains(t, hcl, `"on_record_created_test"`)
	// Should not contain invalid characters in resource name
	assert.NotContains(t, hcl, `"on_record_created_(test)"`)
}

// =============================================================================
// Generator Coordination Tests
// =============================================================================

func TestHCLGenerator_SkipsInactiveAutomations(t *testing.T) {
	app := createTestApp(TestAppID, "Test App")

	// Inactive automation
	inactiveTrigger := discovery.Trigger{
		ID:   TestTriggerID,
		Type: "record_created",
		Name: "Inactive Trigger",
		RawData: map[string]any{
			"__typename": "WorkflowRecordCreateTrigger",
			"id":         TestTriggerID,
			"name":       "Inactive Trigger",
		},
	}

	inactiveAutomation := discovery.Automation{
		ID:           TestAutomationID,
		Name:         "Inactive Automation",
		Status:       "INACTIVE",
		WorkflowID:   TestWorkflowID,
		HasPublished: true,
		Triggers:     []discovery.Trigger{inactiveTrigger},
		Tasks:        []discovery.Task{},
	}
	app.Automations = []discovery.Automation{inactiveAutomation}

	imports := []ImportBlock{
		{ResourceType: "elementum_automation", ResourceName: "inactive", ID: TestAutomationID},
		{ResourceType: "elementum_record_created_trigger", ResourceName: "inactive_trigger", ID: TestAutomationID + ":" + TestTriggerID},
	}

	gen := NewHCLGenerator(app, imports)
	hcl := gen.GenerateTriggerResourcesOnly()

	// Should not generate HCL for inactive automation's triggers
	assert.NotContains(t, hcl, "inactive_trigger")
}

func TestHCLGenerator_SkipsUnpublishedAutomations(t *testing.T) {
	app := createTestApp(TestAppID, "Test App")

	trigger := discovery.Trigger{
		ID:   TestTriggerID,
		Type: "record_created",
		Name: "Unpublished Trigger",
		RawData: map[string]any{
			"__typename": "WorkflowRecordCreateTrigger",
			"id":         TestTriggerID,
			"name":       "Unpublished Trigger",
		},
	}

	unpublishedAutomation := discovery.Automation{
		ID:           TestAutomationID,
		Name:         "Unpublished Automation",
		Status:       "ACTIVE",
		WorkflowID:   TestWorkflowID,
		HasPublished: false, // Not published
		Triggers:     []discovery.Trigger{trigger},
		Tasks:        []discovery.Task{},
	}
	app.Automations = []discovery.Automation{unpublishedAutomation}

	imports := []ImportBlock{
		{ResourceType: "elementum_automation", ResourceName: "unpublished", ID: TestAutomationID},
		{ResourceType: "elementum_record_created_trigger", ResourceName: "unpublished_trigger", ID: TestAutomationID + ":" + TestTriggerID},
	}

	gen := NewHCLGenerator(app, imports)
	hcl := gen.GenerateTriggerResourcesOnly()

	// Should not generate HCL for unpublished automation's triggers
	assert.NotContains(t, hcl, "unpublished_trigger")
}

func TestHCLGenerator_MultipleAutomations(t *testing.T) {
	app := createTestApp(TestAppID, "Test App")

	// First automation
	trigger1 := discovery.Trigger{
		ID:   TestTriggerID,
		Type: "record_created",
		Name: "Trigger 1",
		RawData: map[string]any{
			"__typename": "WorkflowRecordCreateTrigger",
			"id":         TestTriggerID,
			"name":       "Trigger 1",
		},
	}
	automation1 := discovery.Automation{
		ID:           TestAutomationID,
		Name:         "Automation 1",
		Status:       "ACTIVE",
		WorkflowID:   TestWorkflowID,
		HasPublished: true,
		Triggers:     []discovery.Trigger{trigger1},
		Tasks:        []discovery.Task{},
	}

	// Second automation
	automation2ID := "auto-2-uuid-1234-5678-abcdefabcdef"
	workflow2ID := "work-2-uuid-1234-5678-abcdefabcdef"
	trigger2ID := "trig-2-uuid-1234-5678-abcdefabcdef"
	trigger2 := discovery.Trigger{
		ID:   trigger2ID,
		Type: "webhook",
		Name: "Trigger 2",
		RawData: map[string]any{
			"__typename": "WorkflowWebhookTrigger",
			"id":         trigger2ID,
			"name":       "Trigger 2",
		},
	}
	automation2 := discovery.Automation{
		ID:           automation2ID,
		Name:         "Automation 2",
		Status:       "ACTIVE",
		WorkflowID:   workflow2ID,
		HasPublished: true,
		Triggers:     []discovery.Trigger{trigger2},
		Tasks:        []discovery.Task{},
	}

	app.Automations = []discovery.Automation{automation1, automation2}

	imports := []ImportBlock{
		{ResourceType: "elementum_automation", ResourceName: "automation_1", ID: TestAutomationID},
		{ResourceType: "elementum_automation", ResourceName: "automation_2", ID: automation2ID},
		{ResourceType: "elementum_record_created_trigger", ResourceName: "trigger_1", ID: TestAutomationID + ":" + TestTriggerID},
		{ResourceType: "elementum_webhook_trigger", ResourceName: "trigger_2", ID: automation2ID + ":" + trigger2ID},
	}

	gen := NewHCLGenerator(app, imports)
	hcl := gen.GenerateTriggerResourcesOnly()

	// Should have triggers from both automations
	assert.Contains(t, hcl, "trigger_1")
	assert.Contains(t, hcl, "trigger_2")
	assert.Contains(t, hcl, "elementum_record_created_trigger")
	assert.Contains(t, hcl, "elementum_webhook_trigger")
}

// =============================================================================
// Parent Reference Tests
// =============================================================================

func TestHCLGenerator_GetParentRef(t *testing.T) {
	app := createTestApp(TestAppID, "Test App")

	trigger := discovery.Trigger{
		ID:   TestTriggerID,
		Type: "record_created",
		Name: "Trigger",
		RawData: map[string]any{
			"__typename": "WorkflowRecordCreateTrigger",
			"id":         TestTriggerID,
			"name":       "Trigger",
		},
	}

	task1 := discovery.Task{
		ID:       TestTaskID,
		Type:     "message",
		Name:     "First Task",
		ParentID: TestTriggerID, // Parent is trigger
		RawData: map[string]any{
			"__typename": "WorkflowMessageTask",
			"id":         TestTaskID,
			"name":       "First Task",
		},
	}

	task2ID := "task-2-uuid-1234-5678-abcdefabcdef"
	task2 := discovery.Task{
		ID:       task2ID,
		Type:     "notification",
		Name:     "Second Task",
		ParentID: TestTaskID, // Parent is first task
		RawData: map[string]any{
			"__typename": "WorkflowNotificationTask",
			"id":         task2ID,
			"name":       "Second Task",
		},
	}

	automation := discovery.Automation{
		ID:           TestAutomationID,
		Name:         "Test Automation",
		Status:       "ACTIVE",
		WorkflowID:   TestWorkflowID,
		HasPublished: true,
		Triggers:     []discovery.Trigger{trigger},
		Tasks:        []discovery.Task{task1, task2},
	}
	app.Automations = []discovery.Automation{automation}

	imports := []ImportBlock{
		{ResourceType: "elementum_automation", ResourceName: "test", ID: TestAutomationID},
		{ResourceType: "elementum_record_created_trigger", ResourceName: "trigger", ID: TestAutomationID + ":" + TestTriggerID},
		{ResourceType: "elementum_message_task", ResourceName: "first_task", ID: TestWorkflowID + ":" + TestTaskID},
		{ResourceType: "elementum_notification_task", ResourceName: "second_task", ID: TestWorkflowID + ":" + task2ID},
	}

	gen := NewHCLGenerator(app, imports)

	// Task 1's parent should be the trigger
	parentRef1 := gen.GetParentRef(TestTaskID)
	require.NotEmpty(t, parentRef1)
	assert.Contains(t, parentRef1, "trigger")

	// Task 2's parent should be task 1
	parentRef2 := gen.GetParentRef(task2ID)
	require.NotEmpty(t, parentRef2)
	assert.Contains(t, parentRef2, "first_task")
}

// Note: assertNoNullAttributes and validateHCLSyntax are defined in test_helpers_test.go
