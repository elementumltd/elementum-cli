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
	"github.com/stretchr/testify/assert"
)

func TestGenerateAutomationsOnly_NonTerminalEmitsTriggersOtherAutomations(t *testing.T) {
	app := createTestApp(TestAppID, "Test App")
	app.Automations = []discovery.Automation{
		{
			ID:           TestAutomationID,
			Name:         "Non Terminal Automation",
			Status:       "ACTIVE",
			WorkflowID:   TestWorkflowID,
			HasPublished: true,
			Terminal:     false, // triggers other automations
		},
	}

	imports := []ImportBlock{
		{ResourceType: "elementum_app", ResourceName: "test_app", ID: TestAppID},
		{ResourceType: "elementum_automation", ResourceName: "non_terminal_automation", ID: TestAutomationID},
	}

	gen := NewHCLGenerator(app, imports)
	hcl := gen.GenerateAutomationResourcesOnly()

	assert.Contains(t, hcl, `resource "elementum_automation"`)
	assert.Contains(t, hcl, `"Non Terminal Automation"`)
	assert.Contains(t, hcl, `triggers_other_automations = true`)
}

func TestGenerateAutomationsOnly_TerminalOmitsTriggersOtherAutomations(t *testing.T) {
	app := createTestApp(TestAppID, "Test App")
	app.Automations = []discovery.Automation{
		{
			ID:           TestAutomationID,
			Name:         "Terminal Automation",
			Status:       "ACTIVE",
			WorkflowID:   TestWorkflowID,
			HasPublished: true,
			Terminal:     true, // does NOT trigger other automations (default)
		},
	}

	imports := []ImportBlock{
		{ResourceType: "elementum_app", ResourceName: "test_app", ID: TestAppID},
		{ResourceType: "elementum_automation", ResourceName: "terminal_automation", ID: TestAutomationID},
	}

	gen := NewHCLGenerator(app, imports)
	hcl := gen.GenerateAutomationResourcesOnly()

	assert.Contains(t, hcl, `resource "elementum_automation"`)
	assert.Contains(t, hcl, `"Terminal Automation"`)
	// Terminal automations now explicitly emit `triggers_other_automations = false`
	// so the roundtrip matches hand-authored truth HCL that's explicit about it.
	assert.Contains(t, hcl, `triggers_other_automations = false`)
}

func TestGenerateAutomationsOnly_MixedTerminalAutomations(t *testing.T) {
	auto2ID := "22222222-2222-2222-2222-222222222223"

	app := createTestApp(TestAppID, "Test App")
	app.Automations = []discovery.Automation{
		{
			ID:           TestAutomationID,
			Name:         "Terminal One",
			Status:       "ACTIVE",
			WorkflowID:   TestWorkflowID,
			HasPublished: true,
			Terminal:     true,
		},
		{
			ID:           auto2ID,
			Name:         "Non Terminal One",
			Status:       "ACTIVE",
			WorkflowID:   "33333333-3333-3333-3333-333333333334",
			HasPublished: true,
			Terminal:     false,
		},
	}

	imports := []ImportBlock{
		{ResourceType: "elementum_app", ResourceName: "test_app", ID: TestAppID},
		{ResourceType: "elementum_automation", ResourceName: "terminal_one", ID: TestAutomationID},
		{ResourceType: "elementum_automation", ResourceName: "non_terminal_one", ID: auto2ID},
	}

	gen := NewHCLGenerator(app, imports)
	hcl := gen.GenerateAutomationResourcesOnly()

	assert.Contains(t, hcl, `"Terminal One"`)
	assert.Contains(t, hcl, `"Non Terminal One"`)

	// Both terminal and non-terminal automations now emit the attribute
	// explicitly (= false and = true respectively) to match hand-authored
	// truth HCL's convention.
	count := strings.Count(hcl, "triggers_other_automations")
	assert.Equal(t, 2, count, "expected one triggers_other_automations attribute per automation")
	assert.Contains(t, hcl, "triggers_other_automations = true")
	assert.Contains(t, hcl, "triggers_other_automations = false")
}

// ============================================================================
// AutomationHCLGenerator.GenerateWorkflowPublishIR tests
// ============================================================================

func TestWorkflowPublishIR_SingleTask_HasDependsOn(t *testing.T) {
	t.Parallel()
	app := createTestApp(TestAppID, "Test App")
	app.Automations = []discovery.Automation{
		createTestAutomation(
			TestAutomationID,
			"Process Order",
			TestWorkflowID,
			"ACTIVE",
			[]discovery.Trigger{
				createTestTrigger(TestTriggerID, "record_created", "On Record Created", nil, nil),
			},
			[]discovery.Task{
				createTestTask(TestTaskID, "message", "Send Notification", TestWorkflowID, TestTriggerID, nil, nil),
			},
		),
	}

	imports := createTestImports(
		ImportResource{TestAutomationID, "elementum_automation", "process_order"},
		ImportResource{TestAutomationID + ":" + TestTriggerID, "elementum_record_created_trigger", "process_order_on_record_created"},
		ImportResource{TestAutomationID + ":" + TestTaskID, "elementum_message_task", "process_order_send_notification"},
		ImportResource{TestAutomationID, "elementum_workflow_publish", "process_order"},
	)

	gen := NewAutomationHCLGenerator(app, imports, createTestUUIDMap(nil))
	blocks := gen.GenerateWorkflowPublishIR()
	hcl := SerializeBlocks(blocks)

	assertHCLContains(t, hcl,
		`resource "elementum_workflow_publish" "process_order"`,
		`automation = elementum_automation.process_order`,
		`workflow_revision = "1.0.0"`,
		`depends_on`,
		`elementum_message_task.process_order_send_notification`,
	)
}

func TestWorkflowPublishIR_MultipleTasks_HasAllDependencies(t *testing.T) {
	t.Parallel()
	app := createTestApp(TestAppID, "Test App")
	task2ID := TestTask2ID
	task3ID := "55555555-5555-5555-5555-555555555557"

	app.Automations = []discovery.Automation{
		createTestAutomation(
			TestAutomationID,
			"Complex Workflow",
			TestWorkflowID,
			"ACTIVE",
			[]discovery.Trigger{
				createTestTrigger(TestTriggerID, "webhook", "Webhook Trigger", nil, nil),
			},
			[]discovery.Task{
				createTestTask(TestTaskID, "variable", "Set Variables", TestWorkflowID, TestTriggerID, nil, nil),
				createTestTask(task2ID, "switch", "Check Condition", TestWorkflowID, TestTaskID, nil, nil),
				createTestTask(task3ID, "update_field", "Update Record", TestWorkflowID, task2ID, nil, nil),
			},
		),
	}

	imports := createTestImports(
		ImportResource{TestAutomationID, "elementum_automation", "complex_workflow"},
		ImportResource{TestAutomationID + ":" + TestTriggerID, "elementum_webhook_trigger", "complex_workflow_webhook"},
		ImportResource{TestAutomationID + ":" + TestTaskID, "elementum_variable_task", "complex_workflow_set_variables"},
		ImportResource{TestAutomationID + ":" + task2ID, "elementum_switch_task", "complex_workflow_check_condition"},
		ImportResource{TestAutomationID + ":" + task3ID, "elementum_update_field_task", "complex_workflow_update_record"},
		ImportResource{TestAutomationID, "elementum_workflow_publish", "complex_workflow"},
	)

	gen := NewAutomationHCLGenerator(app, imports, createTestUUIDMap(nil))
	blocks := gen.GenerateWorkflowPublishIR()
	hcl := SerializeBlocks(blocks)

	assertHCLContains(t, hcl,
		`elementum_variable_task.complex_workflow_set_variables`,
		`elementum_switch_task.complex_workflow_check_condition`,
		`elementum_update_field_task.complex_workflow_update_record`,
	)
}

func TestWorkflowPublishIR_NoTasks_NoDependsOn(t *testing.T) {
	t.Parallel()
	app := createTestApp(TestAppID, "Test App")
	app.Automations = []discovery.Automation{
		createTestAutomation(
			TestAutomationID,
			"Trigger Only",
			TestWorkflowID,
			"ACTIVE",
			[]discovery.Trigger{
				createTestTrigger(TestTriggerID, "webhook", "Webhook", nil, nil),
			},
			[]discovery.Task{}, // no tasks
		),
	}

	imports := createTestImports(
		ImportResource{TestAutomationID, "elementum_automation", "trigger_only"},
		ImportResource{TestAutomationID + ":" + TestTriggerID, "elementum_webhook_trigger", "trigger_only_webhook"},
		ImportResource{TestAutomationID, "elementum_workflow_publish", "trigger_only"},
	)

	gen := NewAutomationHCLGenerator(app, imports, createTestUUIDMap(nil))
	blocks := gen.GenerateWorkflowPublishIR()
	hcl := SerializeBlocks(blocks)

	assertHCLContains(t, hcl,
		`resource "elementum_workflow_publish" "trigger_only"`,
		`automation = elementum_automation.trigger_only`,
	)
	assertHCLNotContains(t, hcl, "depends_on")
}

func TestWorkflowPublishIR_SkipsUnknownTasks(t *testing.T) {
	t.Parallel()
	app := createTestApp(TestAppID, "Test App")
	unknownTaskID := TestTask2ID

	app.Automations = []discovery.Automation{
		createTestAutomation(
			TestAutomationID,
			"With Unknown",
			TestWorkflowID,
			"ACTIVE",
			[]discovery.Trigger{
				createTestTrigger(TestTriggerID, "record_created", "Trigger", nil, nil),
			},
			[]discovery.Task{
				createTestTask(TestTaskID, "message", "Send Message", TestWorkflowID, TestTriggerID, nil, nil),
				createTestTask(unknownTaskID, "unknown", "Unknown Task", TestWorkflowID, TestTaskID, nil, nil),
			},
		),
	}

	imports := createTestImports(
		ImportResource{TestAutomationID, "elementum_automation", "with_unknown"},
		ImportResource{TestAutomationID + ":" + TestTriggerID, "elementum_record_created_trigger", "with_unknown_trigger"},
		ImportResource{TestAutomationID + ":" + TestTaskID, "elementum_message_task", "with_unknown_send_message"},
		ImportResource{TestAutomationID, "elementum_workflow_publish", "with_unknown"},
	)

	gen := NewAutomationHCLGenerator(app, imports, createTestUUIDMap(nil))
	blocks := gen.GenerateWorkflowPublishIR()
	hcl := SerializeBlocks(blocks)

	assertHCLContains(t, hcl,
		`elementum_message_task.with_unknown_send_message`,
	)
	assertHCLNotContains(t, hcl, "elementum_unknown_task")
}

func TestWorkflowPublishIR_InactiveAutomation_Skipped(t *testing.T) {
	t.Parallel()
	app := createTestApp(TestAppID, "Test App")
	app.Automations = []discovery.Automation{
		createTestAutomation(
			TestAutomationID,
			"Inactive Automation",
			TestWorkflowID,
			"INACTIVE",
			[]discovery.Trigger{
				createTestTrigger(TestTriggerID, "webhook", "Trigger", nil, nil),
			},
			[]discovery.Task{
				createTestTask(TestTaskID, "message", "Task", TestWorkflowID, TestTriggerID, nil, nil),
			},
		),
	}

	imports := createTestImports(
		ImportResource{TestAutomationID, "elementum_automation", "inactive_automation"},
		ImportResource{TestAutomationID + ":" + TestTaskID, "elementum_message_task", "inactive_task"},
		ImportResource{TestAutomationID, "elementum_workflow_publish", "inactive_automation"},
	)

	gen := NewAutomationHCLGenerator(app, imports, createTestUUIDMap(nil))
	blocks := gen.GenerateWorkflowPublishIR()
	hcl := SerializeBlocks(blocks)

	assertHCLNotContains(t, hcl, "elementum_workflow_publish")
}

func TestWorkflowPublishIR_UnpublishedAutomation_Skipped(t *testing.T) {
	t.Parallel()
	app := createTestApp(TestAppID, "Test App")
	automation := createTestAutomation(
		TestAutomationID,
		"Draft Automation",
		TestWorkflowID,
		"ACTIVE",
		[]discovery.Trigger{
			createTestTrigger(TestTriggerID, "webhook", "Trigger", nil, nil),
		},
		[]discovery.Task{
			createTestTask(TestTaskID, "message", "Task", TestWorkflowID, TestTriggerID, nil, nil),
		},
	)
	automation.HasPublished = false // override to test this specific condition
	app.Automations = []discovery.Automation{automation}

	imports := createTestImports(
		ImportResource{TestAutomationID, "elementum_automation", "draft_automation"},
		ImportResource{TestAutomationID + ":" + TestTaskID, "elementum_message_task", "draft_task"},
		ImportResource{TestAutomationID, "elementum_workflow_publish", "draft_automation"},
	)

	gen := NewAutomationHCLGenerator(app, imports, createTestUUIDMap(nil))
	blocks := gen.GenerateWorkflowPublishIR()
	hcl := SerializeBlocks(blocks)

	assertHCLNotContains(t, hcl, "elementum_workflow_publish")
}

func TestWorkflowPublishIR_WithOutputs_HasDependsOnAndOutputs(t *testing.T) {
	t.Parallel()
	app := createTestApp(TestAppID, "Test App")

	automation := createTestAutomation(
		TestAutomationID,
		"On Demand Flow",
		TestWorkflowID,
		"ACTIVE",
		[]discovery.Trigger{
			createTestTrigger(TestTriggerID, "on_demand", "Manual Trigger", nil, nil),
		},
		[]discovery.Task{
			createTestTask(TestTaskID, "message", "Generate Output", TestWorkflowID, TestTriggerID, nil, nil),
		},
	)
	automation.Outputs = []discovery.WorkflowOutput{
		{
			Name: "result",
			Value: map[string]interface{}{
				"taskReference": map[string]interface{}{
					"name": "task." + TestTaskID + ".result",
				},
			},
		},
	}
	app.Automations = []discovery.Automation{automation}

	imports := createTestImports(
		ImportResource{TestAutomationID, "elementum_automation", "on_demand_flow"},
		ImportResource{TestAutomationID + ":" + TestTriggerID, "elementum_on_demand_trigger", "on_demand_flow_trigger"},
		ImportResource{TestAutomationID + ":" + TestTaskID, "elementum_message_task", "on_demand_flow_generate_output"},
		ImportResource{TestAutomationID, "elementum_workflow_publish", "on_demand_flow"},
	)

	gen := NewAutomationHCLGenerator(app, imports, createTestUUIDMap(nil))
	blocks := gen.GenerateWorkflowPublishIR()
	hcl := SerializeBlocks(blocks)

	assertHCLContains(t, hcl,
		`depends_on`,
		`elementum_message_task.on_demand_flow_generate_output`,
		`outputs`,
	)
}

func TestWorkflowPublishIR_MixedTaskTypes_CorrectRefs(t *testing.T) {
	t.Parallel()
	app := createTestApp(TestAppID, "Test App")
	task2ID := TestTask2ID
	task3ID := "55555555-5555-5555-5555-555555555557"
	task4ID := "55555555-5555-5555-5555-555555555558"

	app.Automations = []discovery.Automation{
		createTestAutomation(
			TestAutomationID,
			"Mixed Tasks",
			TestWorkflowID,
			"ACTIVE",
			[]discovery.Trigger{
				createTestTrigger(TestTriggerID, "record_created", "Trigger", nil, nil),
			},
			[]discovery.Task{
				createTestTask(TestTaskID, "file_reader", "Parse Payload", TestWorkflowID, TestTriggerID, nil, nil),
				createTestTask(task2ID, "switch", "Route Event", TestWorkflowID, TestTaskID, nil, nil),
				createTestTask(task3ID, "create_record", "Create Record", TestWorkflowID, task2ID, nil, nil),
				createTestTask(task4ID, "record_search", "Find Record", TestWorkflowID, task2ID, nil, nil),
			},
		),
	}

	imports := createTestImports(
		ImportResource{TestAutomationID, "elementum_automation", "mixed_tasks"},
		ImportResource{TestAutomationID + ":" + TestTriggerID, "elementum_record_created_trigger", "mixed_tasks_trigger"},
		ImportResource{TestAutomationID + ":" + TestTaskID, "elementum_file_reader_task", "mixed_tasks_parse_payload"},
		ImportResource{TestAutomationID + ":" + task2ID, "elementum_switch_task", "mixed_tasks_route_event"},
		ImportResource{TestAutomationID + ":" + task3ID, "elementum_create_record_task", "mixed_tasks_create_record"},
		ImportResource{TestAutomationID + ":" + task4ID, "elementum_record_search_task", "mixed_tasks_find_record"},
		ImportResource{TestAutomationID, "elementum_workflow_publish", "mixed_tasks"},
	)

	gen := NewAutomationHCLGenerator(app, imports, createTestUUIDMap(nil))
	blocks := gen.GenerateWorkflowPublishIR()
	hcl := SerializeBlocks(blocks)

	assertHCLContains(t, hcl,
		`elementum_file_reader_task.mixed_tasks_parse_payload`,
		`elementum_switch_task.mixed_tasks_route_event`,
		`elementum_create_record_task.mixed_tasks_create_record`,
		`elementum_record_search_task.mixed_tasks_find_record`,
	)
}

// ============================================================================
// AspectAutomationHCLGenerator.GenerateWorkflowPublishIR tests
// ============================================================================

func TestAspectWorkflowPublishIR_GeneratesBlock(t *testing.T) {
	t.Parallel()
	automations := []discovery.Automation{
		createTestAutomation(
			TestAutomationID,
			"Element Automation",
			TestWorkflowID,
			"ACTIVE",
			[]discovery.Trigger{
				createTestTrigger(TestTriggerID, "record_created", "Trigger", nil, nil),
			},
			[]discovery.Task{
				createTestTask(TestTaskID, "message", "Notify", TestWorkflowID, TestTriggerID, nil, nil),
			},
		),
	}

	imports := createTestImports(
		ImportResource{TestAutomationID, "elementum_automation", "element_automation"},
		ImportResource{TestAutomationID + ":" + TestTriggerID, "elementum_record_created_trigger", "element_automation_trigger"},
		ImportResource{TestAutomationID + ":" + TestTaskID, "elementum_message_task", "element_automation_notify"},
		ImportResource{TestAutomationID, "elementum_workflow_publish", "element_automation"},
	)

	gen := NewAspectAutomationHCLGenerator(
		TestElementID, "Test Element", "element",
		automations, imports, createTestUUIDMap(nil),
	)
	blocks := gen.GenerateAllIR()
	hcl := SerializeBlocks(blocks)

	assertHCLContains(t, hcl,
		`resource "elementum_workflow_publish" "element_automation"`,
		`automation = elementum_automation.element_automation`,
		`workflow_revision = "1.0.0"`,
	)
}

func TestAspectWorkflowPublishIR_HasDependsOn(t *testing.T) {
	t.Parallel()
	task2ID := TestTask2ID

	automations := []discovery.Automation{
		createTestAutomation(
			TestAutomationID,
			"Aspect Flow",
			TestWorkflowID,
			"ACTIVE",
			[]discovery.Trigger{
				createTestTrigger(TestTriggerID, "record_updated", "On Update", nil, nil),
			},
			[]discovery.Task{
				createTestTask(TestTaskID, "update_field", "Update Status", TestWorkflowID, TestTriggerID, nil, nil),
				createTestTask(task2ID, "notification", "Send Alert", TestWorkflowID, TestTaskID, nil, nil),
			},
		),
	}

	imports := createTestImports(
		ImportResource{TestAutomationID, "elementum_automation", "aspect_flow"},
		ImportResource{TestAutomationID + ":" + TestTriggerID, "elementum_record_updated_trigger", "aspect_flow_on_update"},
		ImportResource{TestAutomationID + ":" + TestTaskID, "elementum_update_field_task", "aspect_flow_update_status"},
		ImportResource{TestAutomationID + ":" + task2ID, "elementum_notification_task", "aspect_flow_send_alert"},
		ImportResource{TestAutomationID, "elementum_workflow_publish", "aspect_flow"},
	)

	gen := NewAspectAutomationHCLGenerator(
		TestElementID, "Test Element", "element",
		automations, imports, createTestUUIDMap(nil),
	)
	blocks := gen.GenerateWorkflowPublishIR()
	hcl := SerializeBlocks(blocks)

	assertHCLContains(t, hcl,
		`depends_on`,
		`elementum_update_field_task.aspect_flow_update_status`,
		`elementum_notification_task.aspect_flow_send_alert`,
	)
}
