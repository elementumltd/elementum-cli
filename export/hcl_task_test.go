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

// ============================================================================
// Core TaskHCLGenerator Tests
// ============================================================================

func TestTaskHCLGenerator_GenerateAll(t *testing.T) {
	app := createTestApp(TestAppID, "Test App")
	automations := []discovery.Automation{
		createTestAutomation(
			TestAutomationID,
			"Test Automation",
			TestWorkflowID,
			"ACTIVE",
			[]discovery.Trigger{{ID: TestTriggerID, Type: "record_created", RawData: buildMinimalTriggerRawData("record_created")}},
			[]discovery.Task{{
				ID:         TestTaskID,
				Type:       "message",
				Name:       "Send Message",
				WorkflowID: TestWorkflowID,
				ParentID:   TestTriggerID,
				RawData:    buildMinimalTaskRawData("message"),
			}},
		),
	}

	imports := []ImportBlock{
		{ID: TestAutomationID, ResourceType: "elementum_automation", ResourceName: "test_automation"},
		{ID: TestAutomationID + ":" + TestTriggerID, ResourceType: "elementum_record_created_trigger", ResourceName: "test_trigger"},
		{ID: TestWorkflowID + ":" + TestTaskID, ResourceType: "elementum_message_task", ResourceName: "send_message"},
	}
	uuidMap := make(map[string]string)

	generator := NewTaskHCLGenerator(app, imports, uuidMap, automations)
	hcl := generator.GenerateAll()

	assertHCLContains(t, hcl,
		`resource "elementum_message_task" "send_message"`,
		"parent = elementum_",
		"name",
	)
}

func TestTaskHCLGenerator_BuildReferenceMaps(t *testing.T) {
	app := createTestApp(TestAppID, "Test App")
	automations := []discovery.Automation{
		createTestAutomation(
			TestAutomationID,
			"Test Automation",
			TestWorkflowID,
			"ACTIVE",
			[]discovery.Trigger{{ID: TestTriggerID, Type: "record_created"}},
			[]discovery.Task{
				{ID: TestTaskID, Type: "variable", Name: "Task 1", WorkflowID: TestWorkflowID, ParentID: TestTriggerID},
				{ID: TestTask2ID, Type: "message", Name: "Task 2", WorkflowID: TestWorkflowID, ParentID: TestTaskID},
			},
		),
	}

	imports := []ImportBlock{
		{ID: TestAutomationID, ResourceType: "elementum_automation", ResourceName: "test_automation"},
		{ID: TestAutomationID + ":" + TestTriggerID, ResourceType: "elementum_record_created_trigger", ResourceName: "test_trigger"},
		{ID: TestWorkflowID + ":" + TestTaskID, ResourceType: "elementum_variable_task", ResourceName: "task_1"},
		{ID: TestWorkflowID + ":" + TestTask2ID, ResourceType: "elementum_message_task", ResourceName: "task_2"},
	}

	generator := NewTaskHCLGenerator(app, imports, make(map[string]string), automations)

	// Check trigger ref map was populated
	if _, ok := generator.triggerRefMap[TestTriggerID]; !ok {
		t.Error("Expected trigger ref map to contain trigger ID")
	}

	// Check task ref map was populated
	if _, ok := generator.taskRefMap[TestTaskID]; !ok {
		t.Error("Expected task ref map to contain task ID")
	}

	// Check parent map was populated
	if _, ok := generator.parentMap[TestTaskID]; !ok {
		t.Error("Expected parent map to contain task ID")
	}
}

func TestTaskHCLGenerator_GetParentRef_TriggerParent(t *testing.T) {
	app := createTestApp(TestAppID, "Test App")
	automations := []discovery.Automation{
		createTestAutomation(
			TestAutomationID,
			"Test Automation",
			TestWorkflowID,
			"ACTIVE",
			[]discovery.Trigger{{ID: TestTriggerID, Type: "record_created"}},
			[]discovery.Task{{ID: TestTaskID, Type: "message", Name: "Task", WorkflowID: TestWorkflowID, ParentID: TestTriggerID}},
		),
	}

	imports := []ImportBlock{
		{ID: TestAutomationID, ResourceType: "elementum_automation", ResourceName: "test_automation"},
		{ID: TestAutomationID + ":" + TestTriggerID, ResourceType: "elementum_record_created_trigger", ResourceName: "test_trigger"},
		{ID: TestWorkflowID + ":" + TestTaskID, ResourceType: "elementum_message_task", ResourceName: "task"},
	}

	generator := NewTaskHCLGenerator(app, imports, make(map[string]string), automations)

	parentRef := generator.parentMap[TestTaskID]
	if !strings.Contains(parentRef, "elementum_record_created_trigger") {
		t.Errorf("Expected parent ref to be trigger, got: %s", parentRef)
	}
}

func TestTaskHCLGenerator_GetParentRef_TaskParent(t *testing.T) {
	app := createTestApp(TestAppID, "Test App")
	automations := []discovery.Automation{
		createTestAutomation(
			TestAutomationID,
			"Test Automation",
			TestWorkflowID,
			"ACTIVE",
			[]discovery.Trigger{{ID: TestTriggerID, Type: "record_created"}},
			[]discovery.Task{
				{ID: TestTaskID, Type: "variable", Name: "Task 1", WorkflowID: TestWorkflowID, ParentID: TestTriggerID},
				{ID: TestTask2ID, Type: "message", Name: "Task 2", WorkflowID: TestWorkflowID, ParentID: TestTaskID},
			},
		),
	}

	imports := []ImportBlock{
		{ID: TestAutomationID, ResourceType: "elementum_automation", ResourceName: "test_automation"},
		{ID: TestAutomationID + ":" + TestTriggerID, ResourceType: "elementum_record_created_trigger", ResourceName: "test_trigger"},
		{ID: TestWorkflowID + ":" + TestTaskID, ResourceType: "elementum_variable_task", ResourceName: "task_1"},
		{ID: TestWorkflowID + ":" + TestTask2ID, ResourceType: "elementum_message_task", ResourceName: "task_2"},
	}

	generator := NewTaskHCLGenerator(app, imports, make(map[string]string), automations)

	parentRef := generator.parentMap[TestTask2ID]
	if !strings.Contains(parentRef, "elementum_variable_task") {
		t.Errorf("Expected parent ref to be variable_task, got: %s", parentRef)
	}
}

func TestTaskHCLGenerator_SkipInactiveAutomations(t *testing.T) {
	app := createTestApp(TestAppID, "Test App")
	automations := []discovery.Automation{
		createTestAutomation(
			TestAutomationID,
			"Inactive Automation",
			TestWorkflowID,
			"INACTIVE",
			[]discovery.Trigger{{ID: TestTriggerID, Type: "record_created"}},
			[]discovery.Task{{ID: TestTaskID, Type: "message", Name: "Task", WorkflowID: TestWorkflowID, ParentID: TestTriggerID}},
		),
	}

	imports := []ImportBlock{
		{ID: TestAutomationID, ResourceType: "elementum_automation", ResourceName: "inactive_automation"},
		{ID: TestAutomationID + ":" + TestTriggerID, ResourceType: "elementum_record_created_trigger", ResourceName: "test_trigger"},
		{ID: TestWorkflowID + ":" + TestTaskID, ResourceType: "elementum_message_task", ResourceName: "task"},
	}

	generator := NewTaskHCLGenerator(app, imports, make(map[string]string), automations)
	hcl := generator.GenerateAll()

	if strings.Contains(hcl, "elementum_message_task") {
		t.Errorf("Expected inactive automation tasks to be skipped, got:\n%s", hcl)
	}
}

func TestTaskHCLGenerator_SkipUnknownTaskType(t *testing.T) {
	app := createTestApp(TestAppID, "Test App")
	automations := []discovery.Automation{
		createTestAutomation(
			TestAutomationID,
			"Test Automation",
			TestWorkflowID,
			"ACTIVE",
			[]discovery.Trigger{{ID: TestTriggerID, Type: "record_created"}},
			[]discovery.Task{{ID: TestTaskID, Type: "unknown", Name: "Unknown Task", WorkflowID: TestWorkflowID, ParentID: TestTriggerID}},
		),
	}

	imports := []ImportBlock{
		{ID: TestAutomationID, ResourceType: "elementum_automation", ResourceName: "test_automation"},
		{ID: TestAutomationID + ":" + TestTriggerID, ResourceType: "elementum_record_created_trigger", ResourceName: "test_trigger"},
	}

	generator := NewTaskHCLGenerator(app, imports, make(map[string]string), automations)
	hcl := generator.GenerateAll()

	if strings.Contains(hcl, "resource") {
		t.Errorf("Expected unknown task type to be skipped, got:\n%s", hcl)
	}
}

// ============================================================================
// ALL 26 Task Type Tests
// ============================================================================

func TestTaskHCL_Switch(t *testing.T) {
	rawData := buildMinimalTaskRawData("switch")
	_, hcl := generateTaskHCL(t, "switch", "test_switch", rawData)

	assertHCLContains(t, hcl,
		`resource "elementum_switch_task" "test_switch"`,
		"parent = elementum_",
		"name",
	)
}

func TestTaskHCL_SwitchWithChildren(t *testing.T) {
	// Test that switch task children (switch cases) are generated as separate resources
	caseApprovedID := "case-approved-1111-111111111111"
	caseDefaultID := "case-default-1111-111111111111"

	switchChildren := []map[string]interface{}{
		{
			"id":    caseApprovedID,
			"label": "Approved",
			"filter": map[string]interface{}{
				"type":  "equals",
				"value": "APPROVED",
			},
		},
		{
			"id":     caseDefaultID,
			"label":  "Default",
			"filter": nil,
		},
	}

	app := createTestApp(TestAppID, "Test App")
	automations := []discovery.Automation{
		createTestAutomation(
			TestAutomationID,
			"Test Automation",
			TestWorkflowID,
			"ACTIVE",
			[]discovery.Trigger{{ID: TestTriggerID, Type: "record_created", RawData: buildMinimalTriggerRawData("record_created")}},
			[]discovery.Task{{
				ID:         TestTaskID,
				Type:       "switch",
				Name:       "Check Status",
				WorkflowID: TestWorkflowID,
				ParentID:   TestTriggerID,
				RawData:    buildMinimalTaskRawData("switch"),
				Children:   switchChildren,
			}},
		),
	}

	imports := []ImportBlock{
		{ID: TestAutomationID, ResourceType: "elementum_automation", ResourceName: "test_automation"},
		{ID: TestAutomationID + ":" + TestTriggerID, ResourceType: "elementum_record_created_trigger", ResourceName: "test_trigger"},
		{ID: TestWorkflowID + ":" + TestTaskID, ResourceType: "elementum_switch_task", ResourceName: "check_status"},
		{ID: caseApprovedID, ResourceType: "elementum_switch_case", ResourceName: "check_status_approved"},
		{ID: caseDefaultID, ResourceType: "elementum_switch_case", ResourceName: "check_status_default"},
	}

	generator := NewTaskHCLGenerator(app, imports, make(map[string]string), automations)
	hcl := generator.GenerateAll()

	// Switch task resource
	assertHCLContains(t, hcl,
		`resource "elementum_switch_task" "check_status"`,
	)

	// Switch case resources
	assertHCLContains(t, hcl,
		`resource "elementum_switch_case" "check_status_approved"`,
		`resource "elementum_switch_case" "check_status_default"`,
	)

	// Parent reference on cases
	assertHCLContains(t, hcl,
		`switch_task = elementum_switch_task.check_status`,
	)

	// Labels
	assertHCLContains(t, hcl,
		`label = "Approved"`,
		`label = "Default"`,
	)

	// Default case
	assertHCLContains(t, hcl,
		`is_default = true`,
	)

	// Case ordering chain
	assertHCLContains(t, hcl,
		`previous_case_id = elementum_switch_case.check_status_approved.id`,
	)
}

func TestTaskHCL_Variable_Create(t *testing.T) {
	rawData := map[string]interface{}{
		"__typename": "WorkflowVariableTask",
		"id":         TestTaskID,
		"name":       "Set Variable",
		"variables": []interface{}{
			map[string]interface{}{
				"__typename": "WorkflowVariableTaskParameterCreate",
				"id":         "var-1",
				"name":       "my_variable",
				"type":       "TEXT",
				"createValue": map[string]interface{}{
					"id":    "ref-1",
					"label": "Hello",
					"value": "Hello",
				},
			},
		},
	}

	_, hcl := generateTaskHCL(t, "variable", "set_variable", rawData)

	assertHCLContains(t, hcl,
		`resource "elementum_variable_task" "set_variable"`,
		`variable_name = "my_variable"`,
		`variable_type = "TEXT"`,
	)
}

func TestTaskHCL_Variable_Update(t *testing.T) {
	// Test that update_variable task exports variable_reference and value attributes
	rawData := map[string]interface{}{
		"__typename": "WorkflowVariableTask",
		"id":         TestTaskID,
		"name":       "Increment Counter",
		"variables": []interface{}{
			map[string]interface{}{
				"__typename": "WorkflowVariableTaskParameterUpdate",
				"updateValue": map[string]interface{}{
					"id":    "ref-1",
					"label": "new value",
					"value": "new value",
				},
				"variable": map[string]interface{}{
					"id":    "var-ref-1",
					"label": "counter",
					"value": `{"variableReference":{"id":"var-123"}}`,
				},
			},
		},
	}

	_, hcl := generateTaskHCL(t, "update_variable", "increment_counter", rawData)

	t.Logf("Generated HCL:\n%s", hcl)

	assertHCLContains(t, hcl,
		`resource "elementum_update_variable_task" "increment_counter"`,
		"variable_reference",
		"value",
	)

	// Should NOT have variable_name or variable_type (those are for create)
	assertHCLNotContains(t, hcl,
		"variable_name",
		"variable_type",
	)
}

func TestTaskHCL_UpdateField(t *testing.T) {
	rawData := map[string]interface{}{
		"__typename": "WorkflowUpdateFieldTask",
		"id":         TestTaskID,
		"name":       "Update Record",
		"aspect": map[string]interface{}{
			"id":   TestAppID,
			"name": "Test App",
		},
		"workflowFields": []interface{}{},
	}

	uuidMap := map[string]string{TestAppID: "elementum_app.test_app.id"}
	_, hcl := generateTaskHCLWithUUIDMap(t, "update_field", "update_record", rawData, uuidMap)

	assertHCLContains(t, hcl,
		`resource "elementum_update_field_task" "update_record"`,
		"parent = elementum_",
	)
}

func TestTaskHCL_CreateRecord(t *testing.T) {
	rawData := map[string]interface{}{
		"__typename": "WorkflowCreateRecordTask",
		"id":         TestTaskID,
		"name":       "Create Record",
		"aspect": map[string]interface{}{
			"id":   TestAppID,
			"name": "Test App",
		},
		"workflowFields": []interface{}{},
	}

	uuidMap := map[string]string{TestAppID: "elementum_app.test_app.id"}
	_, hcl := generateTaskHCLWithUUIDMap(t, "create_record", "create_record", rawData, uuidMap)

	assertHCLContains(t, hcl,
		`resource "elementum_create_record_task" "create_record"`,
		"object_id",
	)
}

func TestTaskHCL_RecordSearch(t *testing.T) {
	rawData := map[string]interface{}{
		"__typename": "WorkflowRecordSearchTask",
		"id":         TestTaskID,
		"name":       "Search Records",
		"aspect": map[string]interface{}{
			"id":   TestAppID,
			"name": "Test App",
		},
		"limit": float64(100),
	}

	uuidMap := map[string]string{TestAppID: "elementum_app.test_app.id"}
	_, hcl := generateTaskHCLWithUUIDMap(t, "record_search", "search_records", rawData, uuidMap)

	assertHCLContains(t, hcl,
		`resource "elementum_record_search_task" "search_records"`,
		"object_id",
	)
}

func TestTaskHCL_AiAgent(t *testing.T) {
	rawData := map[string]interface{}{
		"__typename": "WorkflowAiAgentTask",
		"id":         TestTaskID,
		"name":       "Run Agent",
		"agent": map[string]interface{}{
			"id":   TestAgentID,
			"name": "My Agent",
		},
		"agentPrompt": map[string]interface{}{
			"id":    "ref-1",
			"label": "Analyze this",
			"value": "Analyze this",
		},
	}

	uuidMap := map[string]string{TestAgentID: "elementum_agent.my_agent.id"}
	_, hcl := generateTaskHCLWithUUIDMap(t, "ai_agent", "ai_agent", rawData, uuidMap)

	assertHCLContains(t, hcl,
		`resource "elementum_ai_agent_task" "ai_agent"`,
		"agent_id",
		"prompt",
	)
}

func TestTaskHCL_Message(t *testing.T) {
	rawData := map[string]interface{}{
		"__typename": "WorkflowMessageTask",
		"id":         TestTaskID,
		"name":       "Send Message",
		"contentsReference": map[string]interface{}{
			"id":    "ref-1",
			"label": "Hello World",
			"value": "Hello World",
		},
	}

	_, hcl := generateTaskHCL(t, "message", "send_message", rawData)

	assertHCLContains(t, hcl,
		`resource "elementum_message_task" "send_message"`,
		"message",
	)
}

func TestTaskHCL_SendEmail(t *testing.T) {
	rawData := map[string]interface{}{
		"__typename": "WorkflowSendEmailTask",
		"id":         TestTaskID,
		"name":       "Send Email",
		"subject": map[string]interface{}{
			"id":    "ref-1",
			"label": "Subject",
			"value": "Subject",
		},
		"messageBody": map[string]interface{}{
			"id":    "ref-2",
			"label": "Body",
			"value": "Body",
		},
	}

	_, hcl := generateTaskHCL(t, "send_email", "send_email", rawData)

	assertHCLContains(t, hcl,
		`resource "elementum_send_email_task" "send_email"`,
		"subject",
		"body",
	)
}

func TestTaskHCL_ForEach(t *testing.T) {
	rawData := map[string]interface{}{
		"__typename": "WorkflowForEachTask",
		"id":         TestTaskID,
		"name":       "For Each",
		"forEach": map[string]interface{}{
			"id":    "ref-1",
			"label": "Records",
			"value": "Records",
		},
		"children": []interface{}{},
	}

	_, hcl := generateTaskHCL(t, "for_each", "for_each", rawData)

	assertHCLContains(t, hcl,
		`resource "elementum_for_each_task" "for_each"`,
		"list",
	)
}

func TestTaskHCL_SaveAttachment(t *testing.T) {
	rawData := buildMinimalTaskRawData("save_attachment")
	_, hcl := generateTaskHCL(t, "save_attachment", "save_attachment", rawData)

	assertHCLContains(t, hcl,
		`resource "elementum_save_attachment_task" "save_attachment"`,
	)
}

func TestTaskHCL_Notification(t *testing.T) {
	rawData := map[string]interface{}{
		"__typename": "WorkflowNotificationTask",
		"id":         TestTaskID,
		"name":       "Send Notification",
		"messageReference": map[string]interface{}{
			"id":    "ref-1",
			"label": "Notification message",
			"value": "Notification message",
		},
	}

	_, hcl := generateTaskHCL(t, "notification", "send_notification", rawData)

	assertHCLContains(t, hcl,
		`resource "elementum_notification_task" "send_notification"`,
		"message",
	)
}

func TestTaskHCL_Api_GET(t *testing.T) {
	rawData := map[string]interface{}{
		"__typename": "WorkflowApiTask",
		"id":         TestTaskID,
		"name":       "API Call",
		"method":     "GET",
		"urlReference": map[string]interface{}{
			"id":    "ref-1",
			"label": "https://api.example.com",
			"value": "https://api.example.com",
		},
	}

	_, hcl := generateTaskHCL(t, "api", "api_call", rawData)

	assertHCLContains(t, hcl,
		`resource "elementum_api_task" "api_call"`,
		`method = "GET"`,
		"url",
	)
}

func TestTaskHCL_Api_POST_JSONBody(t *testing.T) {
	rawData := map[string]interface{}{
		"__typename": "WorkflowApiTask",
		"id":         TestTaskID,
		"name":       "API POST",
		"method":     "POST",
		"urlReference": map[string]interface{}{
			"id":    "ref-1",
			"label": "https://api.example.com",
			"value": "https://api.example.com",
		},
		"body": map[string]interface{}{
			"__typename": "ApiJsonBody",
			"jsonReference": map[string]interface{}{
				"id":    "ref-2",
				"label": `{"key": "value"}`,
				"value": `{"key": "value"}`,
			},
		},
	}

	_, hcl := generateTaskHCL(t, "api", "api_post", rawData)

	assertHCLContains(t, hcl,
		`resource "elementum_api_task" "api_post"`,
		`method = "POST"`,
	)
}

func TestTaskHCL_Api_OAuth_RequestType(t *testing.T) {
	tests := []struct {
		name        string
		requestType string
	}{
		{"BASIC stays uppercase", "BASIC"},
		{"FORM stays uppercase", "FORM"},
		{"lowercase basic becomes uppercase", "basic"},
		{"lowercase form becomes uppercase", "form"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rawData := map[string]interface{}{
				"__typename": "WorkflowApiTask",
				"id":         TestTaskID,
				"name":       "OAuth Request",
				"method":     "GET",
				"urlReference": map[string]interface{}{
					"value": "https://api.example.com/oauth-secure",
				},
				"authorization": map[string]interface{}{
					"__typename":   "ApiOauthAuthorization",
					"clientId":     "my-client-id",
					"clientSecret": "my-client-secret",
					"url":          "https://auth.example.com/token",
					"requestType":  tt.requestType,
				},
			}

			_, hcl := generateTaskHCL(t, "api", "oauth_request", rawData)

			assertHCLContains(t, hcl,
				`resource "elementum_api_task" "oauth_request"`,
			)
			assertAttributeValue(t, hcl, "request_type", `"`+strings.ToUpper(tt.requestType)+`"`)
			assertHCLNotContains(t, hcl,
				`request_type = "basic"`,
				`request_type = "form"`,
			)
		})
	}
}

func TestTaskHCL_AiFileRead(t *testing.T) {
	rawData := map[string]interface{}{
		"__typename": "WorkflowAiFileAnalysisTask",
		"id":         TestTaskID,
		"name":       "Read File",
		"documentModel": map[string]interface{}{
			"id":   TestDocumentModelID,
			"name": "My Model",
		},
		"attachment": map[string]interface{}{
			"id":    "ref-1",
			"label": "File",
			"value": "File",
		},
	}

	uuidMap := map[string]string{TestDocumentModelID: "elementum_ai_file_reader.my_model.id"}
	_, hcl := generateTaskHCLWithUUIDMap(t, "ai_file_read", "read_file", rawData, uuidMap)

	assertHCLContains(t, hcl,
		`resource "elementum_ai_file_read_task" "read_file"`,
		"file_reader_id",
		"file_id",
	)
}

func TestTaskHCL_Calculation(t *testing.T) {
	rawData := map[string]interface{}{
		"__typename": "WorkflowCalculationTask",
		"id":         TestTaskID,
		"name":       "Calculate",
		"calculations": []interface{}{
			map[string]interface{}{
				"id":          "calc-1",
				"name":        "result",
				"calculation": "1 + 1",
				"calculationReference": map[string]interface{}{
					"id":    "ref-1",
					"label": "1 + 1",
					"value": "1 + 1",
				},
			},
		},
	}

	_, hcl := generateTaskHCL(t, "calculation", "calculate", rawData)

	assertHCLContains(t, hcl,
		`resource "elementum_calculation_task" "calculate"`,
		"calculations",
		`variable_name = "result"`,
		`formula = "1 + 1"`,
	)
}

func TestTaskHCL_AddWatcher(t *testing.T) {
	rawData := buildMinimalTaskRawData("add_watcher")
	_, hcl := generateTaskHCL(t, "add_watcher", "add_watcher", rawData)

	assertHCLContains(t, hcl,
		`resource "elementum_add_watcher_task" "add_watcher"`,
		"record_reference",
	)
}

func TestTaskHCL_RelateRecords(t *testing.T) {
	rawData := buildMinimalTaskRawData("relate_records")
	_, hcl := generateTaskHCL(t, "relate_records", "relate_records", rawData)

	assertHCLContains(t, hcl,
		`resource "elementum_relate_records_task" "relate_records"`,
		"source_record_ids",
		"target_record_id",
	)
}

func TestTaskHCL_FindRelatedRecords(t *testing.T) {
	rawData := map[string]interface{}{
		"__typename": "WorkflowFindRelatedRecordsTask",
		"id":         TestTaskID,
		"name":       "Find Related",
		"recordId": map[string]interface{}{
			"id":    "ref-1",
			"label": "Record",
			"value": "Record",
		},
		"relatedAspect": map[string]interface{}{
			"id":   TestElementID,
			"name": "Related Element",
		},
	}

	uuidMap := map[string]string{TestElementID: "elementum_element.related_element.id"}
	_, hcl := generateTaskHCLWithUUIDMap(t, "find_related_records", "find_related", rawData, uuidMap)

	assertHCLContains(t, hcl,
		`resource "elementum_find_related_records_task" "find_related"`,
		"record_reference",
		"object_id",
	)
}

func TestTaskHCL_UserSearch(t *testing.T) {
	rawData := buildMinimalTaskRawData("user_search")
	_, hcl := generateTaskHCL(t, "user_search", "user_search", rawData)

	t.Logf("Generated HCL:\n%s", hcl)

	assertHCLContains(t, hcl,
		`resource "elementum_user_search_task" "user_search"`,
		`email = "test@example.com"`,
	)

	// Should NOT have filter block
	assertHCLNotContains(t, hcl,
		"filter = {",
	)
}

// TestTaskHCL_UserSearch_WithEmailFromFilter tests that user_search task exports
// the email attribute (not filter) with proper refs syntax
func TestTaskHCL_UserSearch_WithEmailFromFilter(t *testing.T) {
	// Simulate the actual filter structure returned by API for user search
	// This is what gets stored when you create a user_search task with an email reference
	rawData := map[string]interface{}{
		"__typename": "WorkflowUserSearchTask",
		"id":         TestTaskID,
		"name":       "Get User Record",
		"filter": map[string]interface{}{
			"type":    "equals",
			"fieldId": "email",
			"value": map[string]interface{}{
				"triggerReference": map[string]interface{}{
					"name": "userEmail",
				},
			},
		},
		"filterValueReferences": map[string]interface{}{
			"valueReferences": []interface{}{
				map[string]interface{}{
					"id":    "ref-email",
					"label": "user_email",
					"value": "trigger.userEmail",
				},
			},
		},
	}

	// Set up the automation with an on_demand trigger that has the user_email parameter
	app := createTestApp(TestAppID, "Test App")

	// Create trigger with FieldRefs mapping for refs syntax conversion
	triggerFieldRefs := map[string]string{
		"userEmail": "user_email", // internal path -> human-readable name
	}
	trigger := discovery.Trigger{
		ID:        TestTriggerID,
		Type:      "on_demand",
		Name:      "Test Trigger",
		RawData:   buildMinimalTriggerRawData("on_demand"),
		FieldRefs: triggerFieldRefs,
	}

	task := discovery.Task{
		ID:         TestTaskID,
		Type:       "user_search",
		Name:       "Get User Record",
		WorkflowID: TestWorkflowID,
		ParentID:   TestTriggerID,
		RawData:    rawData,
	}

	automation := createTestAutomation(
		TestAutomationID,
		"Test Automation",
		TestWorkflowID,
		"ACTIVE",
		[]discovery.Trigger{trigger},
		[]discovery.Task{task},
	)

	imports := []ImportBlock{
		{ID: TestAutomationID, ResourceType: "elementum_automation", ResourceName: "test_automation"},
		{ID: TestAutomationID + ":" + TestTriggerID, ResourceType: "elementum_on_demand_trigger", ResourceName: "test_trigger"},
		{ID: TestWorkflowID + ":" + TestTaskID, ResourceType: "elementum_user_search_task", ResourceName: "get_user_record"},
	}

	generator := NewTaskHCLGenerator(app, imports, make(map[string]string), []discovery.Automation{automation})
	hcl := generator.GenerateAll()

	t.Logf("Generated HCL:\n%s", hcl)

	// Should have email attribute with refs syntax
	assertHCLContains(t, hcl,
		`resource "elementum_user_search_task" "get_user_record"`,
		"email",
	)

	// Should use refs syntax - the trigger reference should be converted
	assertHCLContains(t, hcl,
		`elementum_on_demand_trigger.test_trigger.refs["user_email"]`,
	)

	// Should NOT have filter block - that's the old broken format
	assertHCLNotContains(t, hcl,
		"filter = {",
		"filter =",
	)
}

// TestTaskHCL_UserSearch_WithStaticEmail tests user_search with a static email value
func TestTaskHCL_UserSearch_WithStaticEmail(t *testing.T) {
	rawData := map[string]interface{}{
		"__typename": "WorkflowUserSearchTask",
		"id":         TestTaskID,
		"name":       "Find Admin",
		"filter": map[string]interface{}{
			"type":    "equals",
			"fieldId": "email",
			"value": map[string]interface{}{
				"value": "admin@example.com",
			},
		},
	}

	_, hcl := generateTaskHCL(t, "user_search", "find_admin", rawData)

	t.Logf("Generated HCL:\n%s", hcl)

	// Should have email attribute with the static value
	assertHCLContains(t, hcl,
		`resource "elementum_user_search_task" "find_admin"`,
		`email`,
		`"admin@example.com"`,
	)

	// Should NOT have filter block
	assertHCLNotContains(t, hcl,
		"filter = {",
	)
}

// TestTaskHCL_UserSearch_WithTemplateReference tests user_search with a template string
// that combines static text with a reference (e.g., "${prefix}@example.com")
func TestTaskHCL_UserSearch_WithTemplateReference(t *testing.T) {
	// Template reference structure: combines static text with a parameter reference
	// Template: "{{{p1}}}@example.com" with p1 being a trigger reference
	rawData := map[string]interface{}{
		"__typename": "WorkflowUserSearchTask",
		"id":         TestTaskID,
		"name":       "Find User by Template",
		"filter": map[string]interface{}{
			"type":    "equals",
			"fieldId": "email",
			"value": map[string]interface{}{
				"templateReference": map[string]interface{}{
					"template": "{{{p1}}}@example.com",
					"parameters": []interface{}{
						map[string]interface{}{
							"name": "p1",
							"value": map[string]interface{}{
								"triggerReference": map[string]interface{}{
									"name": "username",
								},
							},
						},
					},
				},
			},
		},
	}

	// Set up automation with trigger that has the username parameter
	app := createTestApp(TestAppID, "Test App")

	triggerFieldRefs := map[string]string{
		"username": "username",
	}
	trigger := discovery.Trigger{
		ID:        TestTriggerID,
		Type:      "on_demand",
		Name:      "Test Trigger",
		RawData:   buildMinimalTriggerRawData("on_demand"),
		FieldRefs: triggerFieldRefs,
	}

	task := discovery.Task{
		ID:         TestTaskID,
		Type:       "user_search",
		Name:       "Find User by Template",
		WorkflowID: TestWorkflowID,
		ParentID:   TestTriggerID,
		RawData:    rawData,
	}

	automation := createTestAutomation(
		TestAutomationID,
		"Test Automation",
		TestWorkflowID,
		"ACTIVE",
		[]discovery.Trigger{trigger},
		[]discovery.Task{task},
	)

	imports := []ImportBlock{
		{ID: TestAutomationID, ResourceType: "elementum_automation", ResourceName: "test_automation"},
		{ID: TestAutomationID + ":" + TestTriggerID, ResourceType: "elementum_on_demand_trigger", ResourceName: "test_trigger"},
		{ID: TestWorkflowID + ":" + TestTaskID, ResourceType: "elementum_user_search_task", ResourceName: "find_user_template"},
	}

	generator := NewTaskHCLGenerator(app, imports, make(map[string]string), []discovery.Automation{automation})
	hcl := generator.GenerateAll()

	t.Logf("Generated HCL:\n%s", hcl)

	// Should have email attribute with interpolated template string
	assertHCLContains(t, hcl,
		`resource "elementum_user_search_task" "find_user_template"`,
		"email",
	)

	// Should use string interpolation with refs syntax
	// Expected: email = "${elementum_on_demand_trigger.test_trigger.refs["username"]}@example.com"
	assertHCLContains(t, hcl,
		`${elementum_on_demand_trigger.test_trigger.refs["username"]}`,
		`@example.com`,
	)

	// Should NOT have filter block
	assertHCLNotContains(t, hcl,
		"filter = {",
	)
}

// TestTaskHCL_UserSearch_WithTaskReference tests user_search with email from a previous task's output
func TestTaskHCL_UserSearch_WithTaskReference(t *testing.T) {
	// Task reference: email comes from a previous task's output
	rawData := map[string]interface{}{
		"__typename": "WorkflowUserSearchTask",
		"id":         TestTaskID,
		"name":       "Find User from API",
		"filter": map[string]interface{}{
			"type":    "equals",
			"fieldId": "email",
			"value": map[string]interface{}{
				"taskReference": map[string]interface{}{
					"name": "user_email",
					"task": map[string]interface{}{
						"id": TestTask2ID,
					},
				},
			},
		},
	}

	app := createTestApp(TestAppID, "Test App")

	trigger := discovery.Trigger{
		ID:      TestTriggerID,
		Type:    "on_demand",
		Name:    "Test Trigger",
		RawData: buildMinimalTriggerRawData("on_demand"),
	}

	// Previous task that outputs user_email
	apiTask := discovery.Task{
		ID:         TestTask2ID,
		Type:       "api",
		Name:       "Fetch User Data",
		WorkflowID: TestWorkflowID,
		ParentID:   TestTriggerID,
		RawData:    buildMinimalTaskRawData("api"),
		FieldRefs: map[string]string{
			"user_email": "user_email", // internal path -> human-readable name
		},
	}

	// The user_search task that references the API task output
	userSearchTask := discovery.Task{
		ID:         TestTaskID,
		Type:       "user_search",
		Name:       "Find User from API",
		WorkflowID: TestWorkflowID,
		ParentID:   TestTask2ID,
		RawData:    rawData,
	}

	automation := createTestAutomation(
		TestAutomationID,
		"Test Automation",
		TestWorkflowID,
		"ACTIVE",
		[]discovery.Trigger{trigger},
		[]discovery.Task{apiTask, userSearchTask},
	)

	imports := []ImportBlock{
		{ID: TestAutomationID, ResourceType: "elementum_automation", ResourceName: "test_automation"},
		{ID: TestAutomationID + ":" + TestTriggerID, ResourceType: "elementum_on_demand_trigger", ResourceName: "test_trigger"},
		{ID: TestWorkflowID + ":" + TestTask2ID, ResourceType: "elementum_api_task", ResourceName: "fetch_user_data"},
		{ID: TestWorkflowID + ":" + TestTaskID, ResourceType: "elementum_user_search_task", ResourceName: "find_user_from_api"},
	}

	generator := NewTaskHCLGenerator(app, imports, make(map[string]string), []discovery.Automation{automation})
	hcl := generator.GenerateAll()

	t.Logf("Generated HCL:\n%s", hcl)

	// Should have email attribute with task refs syntax
	assertHCLContains(t, hcl,
		`resource "elementum_user_search_task" "find_user_from_api"`,
		"email",
	)

	// Should reference the API task's output using refs syntax
	assertHCLContains(t, hcl,
		`elementum_api_task.fetch_user_data.refs["user_email"]`,
	)

	// Should NOT have filter block
	assertHCLNotContains(t, hcl,
		"filter = {",
	)
}

func TestTaskHCL_AiClassify(t *testing.T) {
	rawData := map[string]interface{}{
		"__typename": "WorkflowAiClassifyTask",
		"id":         TestTaskID,
		"name":       "Classify",
		"textToClassify": map[string]interface{}{
			"id":    "ref-1",
			"label": "Text to classify",
			"value": "Text to classify",
		},
		"categories": []interface{}{},
	}

	_, hcl := generateTaskHCL(t, "ai_classify", "classify", rawData)

	assertHCLContains(t, hcl,
		`resource "elementum_ai_classify_task" "classify"`,
		"text_to_classify",
	)
}

func TestTaskHCL_AiSummarize(t *testing.T) {
	rawData := map[string]interface{}{
		"__typename": "WorkflowAiSummarizeTask",
		"id":         TestTaskID,
		"name":       "Summarize",
		"textToSummarize": map[string]interface{}{
			"id":    "ref-1",
			"label": "Text",
			"value": "Text",
		},
	}

	_, hcl := generateTaskHCL(t, "ai_summarize", "summarize", rawData)

	assertHCLContains(t, hcl,
		`resource "elementum_ai_summarize_task" "summarize"`,
		"text_to_summarize",
	)
}

func TestTaskHCL_AiTransform(t *testing.T) {
	rawData := map[string]interface{}{
		"__typename": "WorkflowAiTransformTask",
		"id":         TestTaskID,
		"name":       "Transform",
		"prompt": map[string]interface{}{
			"id":    "ref-1",
			"label": "Transform this",
			"value": "Transform this",
		},
	}

	_, hcl := generateTaskHCL(t, "ai_transform", "transform", rawData)

	assertHCLContains(t, hcl,
		`resource "elementum_ai_transform_task" "transform"`,
		"prompt",
	)
}

func TestTaskHCL_AiTransform_StaticPrompt(t *testing.T) {
	// Test that a static prompt (value field only, no references) is exported correctly
	rawData := map[string]interface{}{
		"__typename": "WorkflowAiTransformTask",
		"id":         TestTaskID,
		"name":       "Analyze Pain Points",
		"prompt": map[string]interface{}{
			"id":    "2f7a5280-1234-5678-9abc-def012345678",
			"value": "Analyze the following text for pain points and summarize them",
		},
		"aiProviderConnector": map[string]interface{}{
			"id":       "conn-123",
			"model":    map[string]interface{}{"name": "Claude Sonnet 4"},
			"provider": map[string]interface{}{"name": "Anthropic"},
		},
		"fieldType": "TEXT",
	}

	_, hcl := generateTaskHCL(t, "ai_transform", "analyze_pain_points", rawData)

	assertHCLContains(t, hcl,
		`resource "elementum_ai_transform_task" "analyze_pain_points"`,
		"Analyze the following text for pain points",
	)
	// Should not have empty prompt or TODO comment
	assertHCLNotContains(t, hcl,
		`prompt = ""`,
		"TODO: Add value reference",
	)
}

func TestTaskHCL_AiTransform_TemplatePrompt(t *testing.T) {
	// Test that a template-based prompt (using templateReference) is exported correctly.
	// This is the core fix for ISS-77: prompts with template interpolation were exported as empty.
	rawData := map[string]interface{}{
		"__typename": "WorkflowAiTransformTask",
		"id":         TestTaskID,
		"name":       "Analyze Pain Points",
		"prompt": map[string]interface{}{
			"id": "2f7a5280-1234-5678-9abc-def012345678",
			"templateReference": map[string]interface{}{
				"template": "Analyze the following text for pain points: {{{p1}}}",
				"parameters": []interface{}{
					map[string]interface{}{
						"name": "p1",
						"value": map[string]interface{}{
							"triggerReference": map[string]interface{}{
								"name": "record.some-field-id.text",
							},
						},
					},
				},
			},
		},
		"aiProviderConnector": map[string]interface{}{
			"id":       "conn-123",
			"model":    map[string]interface{}{"name": "Claude Sonnet 4"},
			"provider": map[string]interface{}{"name": "Anthropic"},
		},
		"fieldType": "TEXT",
	}

	_, hcl := generateTaskHCL(t, "ai_transform", "analyze_pain_points", rawData)

	assertHCLContains(t, hcl,
		`resource "elementum_ai_transform_task" "analyze_pain_points"`,
		"Analyze the following text for pain points",
		"prompt",
	)
	// Should not have empty prompt or TODO comment
	assertHCLNotContains(t, hcl,
		`prompt = ""`,
		"TODO: Add value reference",
	)
}

func TestTaskHCL_AiTransform_TriggerRefPrompt(t *testing.T) {
	// Test that a prompt using a direct trigger reference resolves to refs syntax
	rawData := map[string]interface{}{
		"__typename": "WorkflowAiTransformTask",
		"id":         TestTaskID,
		"name":       "Transform Field",
		"prompt": map[string]interface{}{
			"id": "ref-trigger-1",
			"triggerReference": map[string]interface{}{
				"name": "record.field-123.text",
			},
		},
		"fieldType": "TEXT",
	}

	_, hcl := generateTaskHCL(t, "ai_transform", "transform_field", rawData)

	assertHCLContains(t, hcl,
		`resource "elementum_ai_transform_task" "transform_field"`,
		"prompt",
	)
	// Should not have empty prompt or TODO comment
	assertHCLNotContains(t, hcl,
		`prompt = ""`,
		"TODO: Add value reference",
	)
}

func TestTaskHCL_AiTransform_ConnectorUUIDResolution(t *testing.T) {
	// Test that ai_provider_connector_id UUID gets resolved to a data source reference
	// when the UUID is present in the uuid map (as built by beautify.go from discoveries)
	rawData := map[string]interface{}{
		"__typename": "WorkflowAiTransformTask",
		"id":         TestTaskID,
		"name":       "Transform",
		"prompt": map[string]interface{}{
			"id":    "ref-1",
			"value": "Summarize this",
		},
		"aiProviderConnector": map[string]interface{}{
			"id":       "e0c8081d-e0ea-4ceb-a28c-7d01436240a2",
			"model":    map[string]interface{}{"name": "Claude Sonnet 4"},
			"provider": map[string]interface{}{"name": "Anthropic"},
		},
		"fieldType": "TEXT",
	}

	uuidMap := map[string]string{
		"e0c8081d-e0ea-4ceb-a28c-7d01436240a2": "data.elementum_ai_provider_connector.claude_sonnet_4.id",
	}

	_, hcl := generateTaskHCLWithUUIDMap(t, "ai_transform", "transform", rawData, uuidMap)

	assertHCLContains(t, hcl,
		`resource "elementum_ai_transform_task" "transform"`,
		"data.elementum_ai_provider_connector.claude_sonnet_4.id",
	)
	// Should NOT contain the raw UUID
	assertHCLNotContains(t, hcl,
		"e0c8081d-e0ea-4ceb-a28c-7d01436240a2",
	)
}

func TestTaskHCL_ApprovalChainTask(t *testing.T) {
	rawData := buildMinimalTaskRawData("approval_chain")
	_, hcl := generateTaskHCL(t, "approval_chain", "approval_chain", rawData)

	assertHCLContains(t, hcl,
		`resource "elementum_approval_chain_task" "approval_chain"`,
	)
}

func TestTaskHCL_ApprovalStatusUpdate(t *testing.T) {
	rawData := map[string]interface{}{
		"__typename": "WorkflowApprovalStatusUpdateTask",
		"id":         TestTaskID,
		"name":       "Update Approval",
		"approvalChainTemplate": map[string]interface{}{
			"id": "act-123",
		},
		"status": "APPROVED",
	}

	_, hcl := generateTaskHCL(t, "approval_status_update", "update_approval", rawData)

	assertHCLContains(t, hcl,
		`resource "elementum_approval_status_update_task" "update_approval"`,
		"approval_reference",
	)
	assertAttributeValue(t, hcl, "status", `"APPROVED"`)
	assertHCLNotContains(t, hcl,
		`status = "approved"`,
	)
}

func TestTaskHCL_ApprovalStatusUpdate_StatusCase(t *testing.T) {
	tests := []struct {
		name   string
		status string
	}{
		{"APPROVED stays uppercase", "APPROVED"},
		{"DENIED stays uppercase", "DENIED"},
		{"EXCEPTION stays uppercase", "EXCEPTION"},
		{"lowercase approved becomes uppercase", "approved"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rawData := map[string]interface{}{
				"__typename": "WorkflowApprovalStatusUpdateTask",
				"id":         TestTaskID,
				"name":       "Update Approval",
				"approvalChainTemplate": map[string]interface{}{
					"id": "act-123",
				},
				"status": tt.status,
			}

			_, hcl := generateTaskHCL(t, "approval_status_update", "update_approval", rawData)

			assertAttributeValue(t, hcl, "status", `"`+strings.ToUpper(tt.status)+`"`)
		})
	}
}

func TestTaskHCL_AspectRecordFieldLocking(t *testing.T) {
	rawData := map[string]interface{}{
		"__typename": "WorkflowAspectRecordFieldLockingTask",
		"id":         TestTaskID,
		"name":       "Lock Fields",
		"aspect": map[string]interface{}{
			"id":   TestAppID,
			"name": "Test App",
		},
		"recordId": map[string]interface{}{
			"id":    "ref-1",
			"label": "Record",
			"value": "Record",
		},
		"fieldsToLock":   []interface{}{},
		"fieldsToUnlock": []interface{}{},
	}

	uuidMap := map[string]string{TestAppID: "elementum_app.test_app.id"}
	_, hcl := generateTaskHCLWithUUIDMap(t, "aspect_record_field_locking", "lock_fields", rawData, uuidMap)

	assertHCLContains(t, hcl,
		`resource "elementum_aspect_record_field_locking_task" "lock_fields"`,
		"object_id",
		"record_reference",
	)
}

func TestTaskHCL_BulkExcel(t *testing.T) {
	rawData := map[string]interface{}{
		"__typename": "WorkflowBulkExcelTask",
		"id":         TestTaskID,
		"name":       "Bulk Import",
		"aspect": map[string]interface{}{
			"id":   TestAppID,
			"name": "Test App",
		},
		"documentModel": map[string]interface{}{
			"id":   TestDocumentModelID,
			"name": "Excel Model",
		},
		"attachment": map[string]interface{}{
			"id":    "ref-1",
			"label": "File",
			"value": "File",
		},
	}

	uuidMap := map[string]string{
		TestAppID:           "elementum_app.test_app.id",
		TestDocumentModelID: "elementum_ai_file_reader.excel_model.id",
	}
	_, hcl := generateTaskHCLWithUUIDMap(t, "bulk_excel", "bulk_import", rawData, uuidMap)

	assertHCLContains(t, hcl,
		`resource "elementum_bulk_excel_task" "bulk_import"`,
		"object_id",
		"document_model_id",
		"attachment_reference",
	)
}

func TestTaskHCL_Procedure_Basic(t *testing.T) {
	rawData := map[string]interface{}{
		"__typename":  "WorkflowProcedureTask",
		"id":          TestTaskID,
		"name":        "Call Snowflake Procedure",
		"cloudLinkId": TestCloudLinkID,
		"storedFunction": map[string]interface{}{
			"id":   TestStoredFunctionID,
			"name": "my_procedure",
		},
		"parameters": []interface{}{},
	}

	uuidMap := map[string]string{TestCloudLinkID: "elementum_cloudlink.snowflake.id"}
	_, hcl := generateTaskHCLWithUUIDMap(t, "procedure", "call_snowflake", rawData, uuidMap)

	t.Logf("Generated HCL:\n%s", hcl)

	assertHCLContains(t, hcl,
		`resource "elementum_procedure_task" "call_snowflake"`,
		"cloudlink_id",
		"stored_function_id",
	)
}

func TestTaskHCL_Procedure_WithParameters(t *testing.T) {
	rawData := map[string]interface{}{
		"__typename":  "WorkflowProcedureTask",
		"id":          TestTaskID,
		"name":        "Call Procedure With Params",
		"cloudLinkId": TestCloudLinkID,
		"storedFunction": map[string]interface{}{
			"id":   TestStoredFunctionID,
			"name": "my_procedure",
		},
		"parameters": []interface{}{
			map[string]interface{}{
				"index": float64(0),
				"value": map[string]interface{}{
					"id":    "ref-1",
					"label": "Active",
					"value": "Active",
				},
			},
			map[string]interface{}{
				"index": float64(1),
				"value": map[string]interface{}{
					"id":    "ref-2",
					"label": "100",
					"value": "100",
				},
			},
		},
	}

	uuidMap := map[string]string{TestCloudLinkID: "elementum_cloudlink.snowflake.id"}
	_, hcl := generateTaskHCLWithUUIDMap(t, "procedure", "call_procedure_params", rawData, uuidMap)

	t.Logf("Generated HCL:\n%s", hcl)

	assertHCLContains(t, hcl,
		`resource "elementum_procedure_task" "call_procedure_params"`,
		"cloudlink_id",
		"stored_function_id",
		"parameters",
		"index",
		"value",
	)

	// Verify it uses attribute syntax (parameters = [...]) not block syntax
	assertHCLContains(t, hcl, "parameters = [")
}

func TestTaskHCL_Procedure_WithValueReference(t *testing.T) {
	// Test procedure with a value reference parameter (from trigger)
	rawData := map[string]interface{}{
		"__typename":  "WorkflowProcedureTask",
		"id":          TestTaskID,
		"name":        "Call Procedure With Ref",
		"cloudLinkId": TestCloudLinkID,
		"storedFunction": map[string]interface{}{
			"id":   TestStoredFunctionID,
			"name": "my_procedure",
		},
		"parameters": []interface{}{
			map[string]interface{}{
				"index": float64(0),
				"value": map[string]interface{}{
					"triggerReference": map[string]interface{}{
						"name": "record.status",
					},
				},
			},
		},
	}

	// Set up automation with trigger that has field refs
	app := createTestApp(TestAppID, "Test App")

	triggerFieldRefs := map[string]string{
		"record.status": "Status",
	}
	trigger := discovery.Trigger{
		ID:        TestTriggerID,
		Type:      "record_created",
		Name:      "Test Trigger",
		RawData:   buildMinimalTriggerRawData("record_created"),
		FieldRefs: triggerFieldRefs,
	}

	task := discovery.Task{
		ID:         TestTaskID,
		Type:       "procedure",
		Name:       "Call Procedure With Ref",
		WorkflowID: TestWorkflowID,
		ParentID:   TestTriggerID,
		RawData:    rawData,
	}

	automation := createTestAutomation(
		TestAutomationID,
		"Test Automation",
		TestWorkflowID,
		"ACTIVE",
		[]discovery.Trigger{trigger},
		[]discovery.Task{task},
	)

	imports := []ImportBlock{
		{ID: TestAutomationID, ResourceType: "elementum_automation", ResourceName: "test_automation"},
		{ID: TestAutomationID + ":" + TestTriggerID, ResourceType: "elementum_record_created_trigger", ResourceName: "test_trigger"},
		{ID: TestWorkflowID + ":" + TestTaskID, ResourceType: "elementum_procedure_task", ResourceName: "call_procedure_ref"},
	}

	uuidMap := map[string]string{TestCloudLinkID: "elementum_cloudlink.snowflake.id"}
	generator := NewTaskHCLGenerator(app, imports, uuidMap, []discovery.Automation{automation})
	hcl := generator.GenerateAll()

	t.Logf("Generated HCL:\n%s", hcl)

	assertHCLContains(t, hcl,
		`resource "elementum_procedure_task" "call_procedure_ref"`,
		"parameters = [",
		"index",
		"value",
	)

	// Should use refs syntax for the trigger reference
	assertHCLContains(t, hcl,
		`elementum_record_created_trigger.test_trigger.refs["Status"]`,
	)
}

// ============================================================================
// Run Automation Task Tests
// ============================================================================

func TestTaskHCL_RunAutomation_StaticMode_Basic(t *testing.T) {
	rawData := map[string]interface{}{
		"__typename":  "WorkflowRunAutomationTask",
		"id":          TestTaskID,
		"name":        "Call Sub-Workflow",
		"synchronous": true,
		"automation": map[string]interface{}{
			"id":   TestTargetAutomationID,
			"name": "Target Automation",
		},
		"inputMappings":        []interface{}{},
		"dynamicInputMappings": []interface{}{},
		"outputMappings":       []interface{}{},
	}

	uuidMap := map[string]string{TestTargetAutomationID: "elementum_automation.target_automation.id"}
	_, hcl := generateTaskHCLWithUUIDMap(t, "run_automation", "call_sub_workflow", rawData, uuidMap)

	t.Logf("Generated HCL:\n%s", hcl)

	assertHCLContains(t, hcl,
		`resource "elementum_run_automation_task" "call_sub_workflow"`,
		"parent = elementum_",
		"synchronous = true",
		"target_automation_id",
	)
}

func TestTaskHCL_RunAutomation_StaticMode_WithInputMappings(t *testing.T) {
	rawData := map[string]interface{}{
		"__typename":  "WorkflowRunAutomationTask",
		"id":          TestTaskID,
		"name":        "Call Processor",
		"synchronous": true,
		"automation": map[string]interface{}{
			"id":   TestTargetAutomationID,
			"name": "Processor",
		},
		"inputMappings": []interface{}{
			map[string]interface{}{
				"parameter": map[string]interface{}{
					"id":   "param-1",
					"name": "record_id",
				},
				"value": map[string]interface{}{
					"id":    "ref-1",
					"label": "Record ID",
					"value": "12345",
				},
			},
			map[string]interface{}{
				"parameter": map[string]interface{}{
					"id":   "param-2",
					"name": "status",
				},
				"value": map[string]interface{}{
					"id":    "ref-2",
					"label": "Active",
					"value": "Active",
				},
			},
		},
		"dynamicInputMappings": []interface{}{},
		"outputMappings":       []interface{}{},
	}

	uuidMap := map[string]string{TestTargetAutomationID: "elementum_automation.processor.id"}
	_, hcl := generateTaskHCLWithUUIDMap(t, "run_automation", "call_processor", rawData, uuidMap)

	t.Logf("Generated HCL:\n%s", hcl)

	assertHCLContains(t, hcl,
		`resource "elementum_run_automation_task" "call_processor"`,
		"synchronous = true",
		"target_automation_id",
		"input_mappings = [",
		"parameter_id",
		"value",
	)
}

func TestTaskHCL_RunAutomation_StaticMode_WithValueReference(t *testing.T) {
	// Test run_automation with input mappings that use value references from trigger
	rawData := map[string]interface{}{
		"__typename":  "WorkflowRunAutomationTask",
		"id":          TestTaskID,
		"name":        "Call Processor With Ref",
		"synchronous": true,
		"automation": map[string]interface{}{
			"id":   TestTargetAutomationID,
			"name": "Processor",
		},
		"inputMappings": []interface{}{
			map[string]interface{}{
				"parameter": map[string]interface{}{
					"id":   "param-1",
					"name": "record_id",
				},
				"value": map[string]interface{}{
					"triggerReference": map[string]interface{}{
						"name": "record.id",
					},
				},
			},
		},
		"dynamicInputMappings": []interface{}{},
		"outputMappings":       []interface{}{},
	}

	// Set up automation with trigger that has field refs
	app := createTestApp(TestAppID, "Test App")

	triggerFieldRefs := map[string]string{
		"record.id": "ID",
	}
	trigger := discovery.Trigger{
		ID:        TestTriggerID,
		Type:      "record_created",
		Name:      "Test Trigger",
		RawData:   buildMinimalTriggerRawData("record_created"),
		FieldRefs: triggerFieldRefs,
	}

	task := discovery.Task{
		ID:         TestTaskID,
		Type:       "run_automation",
		Name:       "Call Processor With Ref",
		WorkflowID: TestWorkflowID,
		ParentID:   TestTriggerID,
		RawData:    rawData,
	}

	automation := createTestAutomation(
		TestAutomationID,
		"Test Automation",
		TestWorkflowID,
		"ACTIVE",
		[]discovery.Trigger{trigger},
		[]discovery.Task{task},
	)

	imports := []ImportBlock{
		{ID: TestAutomationID, ResourceType: "elementum_automation", ResourceName: "test_automation"},
		{ID: TestAutomationID + ":" + TestTriggerID, ResourceType: "elementum_record_created_trigger", ResourceName: "test_trigger"},
		{ID: TestWorkflowID + ":" + TestTaskID, ResourceType: "elementum_run_automation_task", ResourceName: "call_processor_ref"},
	}

	uuidMap := map[string]string{TestTargetAutomationID: "elementum_automation.processor.id"}
	generator := NewTaskHCLGenerator(app, imports, uuidMap, []discovery.Automation{automation})
	hcl := generator.GenerateAll()

	t.Logf("Generated HCL:\n%s", hcl)

	assertHCLContains(t, hcl,
		`resource "elementum_run_automation_task" "call_processor_ref"`,
		"input_mappings = [",
		"parameter_id",
		"value",
	)

	// Should use refs syntax for the trigger reference
	assertHCLContains(t, hcl,
		`elementum_record_created_trigger.test_trigger.refs["ID"]`,
	)
}

func TestTaskHCL_RunAutomation_DynamicMode_Basic(t *testing.T) {
	rawData := map[string]interface{}{
		"__typename":  "WorkflowRunAutomationTask",
		"id":          TestTaskID,
		"name":        "Call Dynamic Automation",
		"synchronous": true,
		"automationRef": map[string]interface{}{
			"id":    "ref-1",
			"label": "automation_id",
			"value": "trigger.automation_id",
		},
		"inputMappings":        []interface{}{},
		"dynamicInputMappings": []interface{}{},
		"outputMappings":       []interface{}{},
	}

	_, hcl := generateTaskHCL(t, "run_automation", "call_dynamic", rawData)

	t.Logf("Generated HCL:\n%s", hcl)

	assertHCLContains(t, hcl,
		`resource "elementum_run_automation_task" "call_dynamic"`,
		"synchronous = true",
		"target_automation_ref",
	)

	// Should NOT have target_automation_id
	assertHCLNotContains(t, hcl,
		"target_automation_id",
	)
}

func TestTaskHCL_RunAutomation_DynamicMode_WithDynamicInputMappings(t *testing.T) {
	rawData := map[string]interface{}{
		"__typename":  "WorkflowRunAutomationTask",
		"id":          TestTaskID,
		"name":        "Call Dynamic With Inputs",
		"synchronous": true,
		"automationRef": map[string]interface{}{
			"id":    "ref-1",
			"label": "automation_id",
			"value": "some_automation_id",
		},
		"inputMappings": []interface{}{},
		"dynamicInputMappings": []interface{}{
			map[string]interface{}{
				"name": "record_id",
				"value": map[string]interface{}{
					"id":    "ref-2",
					"label": "ID",
					"value": "12345",
				},
			},
			map[string]interface{}{
				"name": "status",
				"value": map[string]interface{}{
					"id":    "ref-3",
					"label": "Active",
					"value": "Active",
				},
			},
		},
		"outputMappings": []interface{}{},
	}

	_, hcl := generateTaskHCL(t, "run_automation", "call_dynamic_inputs", rawData)

	t.Logf("Generated HCL:\n%s", hcl)

	assertHCLContains(t, hcl,
		`resource "elementum_run_automation_task" "call_dynamic_inputs"`,
		"synchronous = true",
		"target_automation_ref",
		"dynamic_input_mappings = [",
		`name = "record_id"`,
		`name = "status"`,
		"value",
	)
}

func TestTaskHCL_RunAutomation_WithOutputMappings(t *testing.T) {
	rawData := map[string]interface{}{
		"__typename":  "WorkflowRunAutomationTask",
		"id":          TestTaskID,
		"name":        "Call With Outputs",
		"synchronous": true,
		"automation": map[string]interface{}{
			"id":   TestTargetAutomationID,
			"name": "Processor",
		},
		"inputMappings":        []interface{}{},
		"dynamicInputMappings": []interface{}{},
		"outputMappings": []interface{}{
			map[string]interface{}{
				"name": "result",
			},
			map[string]interface{}{
				"name": "error_message",
			},
		},
	}

	uuidMap := map[string]string{TestTargetAutomationID: "elementum_automation.processor.id"}
	_, hcl := generateTaskHCLWithUUIDMap(t, "run_automation", "call_with_outputs", rawData, uuidMap)

	t.Logf("Generated HCL:\n%s", hcl)

	assertHCLContains(t, hcl,
		`resource "elementum_run_automation_task" "call_with_outputs"`,
		"synchronous = true",
		"target_automation_id",
		"output_mappings = [",
		`name = "result"`,
		`name = "error_message"`,
	)
}

func TestTaskHCL_RunAutomation_AsyncMode(t *testing.T) {
	rawData := map[string]interface{}{
		"__typename":  "WorkflowRunAutomationTask",
		"id":          TestTaskID,
		"name":        "Fire and Forget",
		"synchronous": false,
		"automation": map[string]interface{}{
			"id":   TestTargetAutomationID,
			"name": "Background Processor",
		},
		"inputMappings":        []interface{}{},
		"dynamicInputMappings": []interface{}{},
		"outputMappings":       []interface{}{},
	}

	uuidMap := map[string]string{TestTargetAutomationID: "elementum_automation.background_processor.id"}
	_, hcl := generateTaskHCLWithUUIDMap(t, "run_automation", "fire_and_forget", rawData, uuidMap)

	t.Logf("Generated HCL:\n%s", hcl)

	assertHCLContains(t, hcl,
		`resource "elementum_run_automation_task" "fire_and_forget"`,
		"synchronous = false",
		"target_automation_id",
	)
}

// ============================================================================
// Helper Functions
// ============================================================================

// generateTaskHCL creates test data and generates HCL for a single task
func generateTaskHCL(t *testing.T, taskType, resourceName string, rawData map[string]interface{}) (*TaskHCLGenerator, string) {
	t.Helper()
	return generateTaskHCLWithUUIDMap(t, taskType, resourceName, rawData, make(map[string]string))
}

// generateTaskHCLWithUUIDMap creates test data and generates HCL for a single task with custom UUID map
func generateTaskHCLWithUUIDMap(t *testing.T, taskType, resourceName string, rawData map[string]interface{}, uuidMap map[string]string) (*TaskHCLGenerator, string) {
	t.Helper()

	app := createTestApp(TestAppID, "Test App")
	automations := []discovery.Automation{
		createTestAutomation(
			TestAutomationID,
			"Test Automation",
			TestWorkflowID,
			"ACTIVE",
			[]discovery.Trigger{{ID: TestTriggerID, Type: "record_created", RawData: buildMinimalTriggerRawData("record_created")}},
			[]discovery.Task{{
				ID:         TestTaskID,
				Type:       taskType,
				Name:       "Test Task",
				WorkflowID: TestWorkflowID,
				ParentID:   TestTriggerID,
				RawData:    rawData,
			}},
		),
	}

	imports := []ImportBlock{
		{ID: TestAutomationID, ResourceType: "elementum_automation", ResourceName: "test_automation"},
		{ID: TestAutomationID + ":" + TestTriggerID, ResourceType: "elementum_record_created_trigger", ResourceName: "test_trigger"},
		{ID: TestWorkflowID + ":" + TestTaskID, ResourceType: "elementum_" + taskType + "_task", ResourceName: resourceName},
	}

	generator := NewTaskHCLGenerator(app, imports, uuidMap, automations)
	hcl := generator.GenerateAll()

	return generator, hcl
}

// ============================================================================
// Broken Fields Tests
// ============================================================================

func TestTaskHCLGenerator_BrokenField_AddsTodoComment(t *testing.T) {
	t.Parallel()

	// Build an ai_agent task with a broken "agent" field (simulating GraphQL partial-data error)
	rawData := buildMinimalTaskRawData("ai_agent")
	// Remove the agent field to simulate it being null from GraphQL error
	delete(rawData, "agent")

	app := createTestApp(TestAppID, "Test App")
	automations := []discovery.Automation{
		createTestAutomation(
			TestAutomationID,
			"Test Automation",
			TestWorkflowID,
			"ACTIVE",
			[]discovery.Trigger{{ID: TestTriggerID, Type: "record_created", RawData: buildMinimalTriggerRawData("record_created")}},
			[]discovery.Task{{
				ID:         TestTaskID,
				Type:       "ai_agent",
				Name:       "Run Agent",
				WorkflowID: TestWorkflowID,
				ParentID:   TestTriggerID,
				RawData:    rawData,
				BrokenFields: map[string]string{
					"agent": "An unexpected error has occurred.",
				},
			}},
		),
	}

	imports := []ImportBlock{
		{ID: TestAutomationID, ResourceType: "elementum_automation", ResourceName: "test_automation"},
		{ID: TestAutomationID + ":" + TestTriggerID, ResourceType: "elementum_record_created_trigger", ResourceName: "test_trigger"},
		{ID: TestWorkflowID + ":" + TestTaskID, ResourceType: "elementum_ai_agent_task", ResourceName: "run_agent"},
	}
	uuidMap := make(map[string]string)

	generator := NewTaskHCLGenerator(app, imports, uuidMap, automations)
	hcl := generator.GenerateAll()

	// Should contain the TODO comment on the agent_id field
	assertHCLContains(t, hcl,
		`resource "elementum_ai_agent_task" "run_agent"`,
		`agent_id = ""`, // Empty value for broken field
		`# TODO: server error, fix manually after import`,
	)
}

func TestTaskHCLGenerator_BrokenField_NonBrokenFieldStillEmpty(t *testing.T) {
	t.Parallel()

	// Build an ai_agent task with a broken "agent" field but valid prompt
	rawData := buildMinimalTaskRawData("ai_agent")
	delete(rawData, "agent")
	// Keep agentPrompt intact — it should render normally

	app := createTestApp(TestAppID, "Test App")
	automations := []discovery.Automation{
		createTestAutomation(
			TestAutomationID,
			"Test Automation",
			TestWorkflowID,
			"ACTIVE",
			[]discovery.Trigger{{ID: TestTriggerID, Type: "record_created", RawData: buildMinimalTriggerRawData("record_created")}},
			[]discovery.Task{{
				ID:         TestTaskID,
				Type:       "ai_agent",
				Name:       "Run Agent",
				WorkflowID: TestWorkflowID,
				ParentID:   TestTriggerID,
				RawData:    rawData,
				BrokenFields: map[string]string{
					"agent": "An unexpected error has occurred.",
				},
			}},
		),
	}

	imports := []ImportBlock{
		{ID: TestAutomationID, ResourceType: "elementum_automation", ResourceName: "test_automation"},
		{ID: TestAutomationID + ":" + TestTriggerID, ResourceType: "elementum_record_created_trigger", ResourceName: "test_trigger"},
		{ID: TestWorkflowID + ":" + TestTaskID, ResourceType: "elementum_ai_agent_task", ResourceName: "run_agent"},
	}
	uuidMap := make(map[string]string)

	generator := NewTaskHCLGenerator(app, imports, uuidMap, automations)
	hcl := generator.GenerateAll()

	// The prompt field should still render normally (not broken)
	assertHCLContains(t, hcl, `prompt = "Test prompt"`)
}

func TestTaskHCLGenerator_NoBrokenFields_NoTodoComment(t *testing.T) {
	t.Parallel()

	// Normal ai_agent task with no broken fields
	app := createTestApp(TestAppID, "Test App")
	automations := []discovery.Automation{
		createTestAutomation(
			TestAutomationID,
			"Test Automation",
			TestWorkflowID,
			"ACTIVE",
			[]discovery.Trigger{{ID: TestTriggerID, Type: "record_created", RawData: buildMinimalTriggerRawData("record_created")}},
			[]discovery.Task{{
				ID:         TestTaskID,
				Type:       "ai_agent",
				Name:       "Run Agent",
				WorkflowID: TestWorkflowID,
				ParentID:   TestTriggerID,
				RawData:    buildMinimalTaskRawData("ai_agent"),
				// No BrokenFields — normal task
			}},
		),
	}

	imports := []ImportBlock{
		{ID: TestAutomationID, ResourceType: "elementum_automation", ResourceName: "test_automation"},
		{ID: TestAutomationID + ":" + TestTriggerID, ResourceType: "elementum_record_created_trigger", ResourceName: "test_trigger"},
		{ID: TestWorkflowID + ":" + TestTaskID, ResourceType: "elementum_ai_agent_task", ResourceName: "run_agent"},
	}
	uuidMap := make(map[string]string)

	generator := NewTaskHCLGenerator(app, imports, uuidMap, automations)
	hcl := generator.GenerateAll()

	// Should NOT contain TODO comment
	assertHCLNotContains(t, hcl, "# TODO: server error")
	// agent_id should have a real value
	assertHCLContains(t, hcl, "agent_id")
}

func TestTaskHCLGenerator_BrokenField_MissingRequiredWithoutBrokenMap(t *testing.T) {
	t.Parallel()

	// Task with a nil required field but NO BrokenFields map — should produce empty string without comment
	rawData := buildMinimalTaskRawData("ai_agent")
	delete(rawData, "agent")

	app := createTestApp(TestAppID, "Test App")
	automations := []discovery.Automation{
		createTestAutomation(
			TestAutomationID,
			"Test Automation",
			TestWorkflowID,
			"ACTIVE",
			[]discovery.Trigger{{ID: TestTriggerID, Type: "record_created", RawData: buildMinimalTriggerRawData("record_created")}},
			[]discovery.Task{{
				ID:         TestTaskID,
				Type:       "ai_agent",
				Name:       "Run Agent",
				WorkflowID: TestWorkflowID,
				ParentID:   TestTriggerID,
				RawData:    rawData,
				// No BrokenFields — unknown reason for nil
			}},
		),
	}

	imports := []ImportBlock{
		{ID: TestAutomationID, ResourceType: "elementum_automation", ResourceName: "test_automation"},
		{ID: TestAutomationID + ":" + TestTriggerID, ResourceType: "elementum_record_created_trigger", ResourceName: "test_trigger"},
		{ID: TestWorkflowID + ":" + TestTaskID, ResourceType: "elementum_ai_agent_task", ResourceName: "run_agent"},
	}
	uuidMap := make(map[string]string)

	generator := NewTaskHCLGenerator(app, imports, uuidMap, automations)
	hcl := generator.GenerateAll()

	// Should have empty agent_id but no TODO comment
	assertHCLContains(t, hcl, `agent_id = ""`)
	assertHCLNotContains(t, hcl, "# TODO: server error")
}

// TestTaskHCL_VariableRef_ResolvesToDefiningTask verifies that when resolving
// a variable reference, we select the task that DEFINES the variable (has RawData
// with variables[0].__typename == "WorkflowVariableTaskParameterCreate"), not just
// any task that happens to have the variable in its FieldRefs.
// This prevents dependency cycles where update_variable_task references a
// variable_task that comes AFTER it in the workflow.
func TestTaskHCL_VariableRef_ResolvesToDefiningTask(t *testing.T) {
	// Set up two variable tasks:
	// 1. task-defining: Creates variable "myVar" (has RawData with WorkflowVariableTaskParameterCreate)
	// 2. task-other: Has "myVar" in FieldRefs but doesn't define it
	//
	// Both will have variable.myVar in their FieldRefs (as done by the discovery code),
	// but only task-defining has the proper RawData.
	//
	// The update_variable task references myVar - it should resolve to task-defining.

	definingTaskID := "task-defining-id"
	otherTaskID := "task-other-id"
	updateTaskID := "task-update-id"

	// Task that DEFINES the variable
	definingTaskRawData := map[string]interface{}{
		"__typename": "WorkflowVariableTask",
		"id":         definingTaskID,
		"name":       "Create myVar",
		"variables": []interface{}{
			map[string]interface{}{
				"__typename": "WorkflowVariableTaskParameterCreate",
				"id":         "var-1",
				"name":       "myVar",
				"type":       "TEXT",
				"createValue": map[string]interface{}{
					"id":    "ref-1",
					"label": "initial",
					"value": "initial",
				},
			},
		},
	}

	// Another variable task that has myVar in FieldRefs but defines a different var
	otherTaskRawData := map[string]interface{}{
		"__typename": "WorkflowVariableTask",
		"id":         otherTaskID,
		"name":       "Create otherVar",
		"variables": []interface{}{
			map[string]interface{}{
				"__typename": "WorkflowVariableTaskParameterCreate",
				"id":         "var-2",
				"name":       "otherVar",
				"type":       "TEXT",
				"createValue": map[string]interface{}{
					"id":    "ref-2",
					"label": "other",
					"value": "other",
				},
			},
		},
	}

	// Update variable task that updates myVar
	updateTaskRawData := map[string]interface{}{
		"__typename": "WorkflowVariableTask",
		"id":         updateTaskID,
		"name":       "Update myVar",
		"variables": []interface{}{
			map[string]interface{}{
				"__typename": "WorkflowVariableTaskParameterUpdate",
				"variable": map[string]interface{}{
					"id":    "var-ref-1",
					"label": "myVar",
					"value": `{"variableReference":{"name":"myVar"}}`,
				},
				"updateValue": map[string]interface{}{
					"id":    "ref-3",
					"label": "updated",
					"value": "updated",
				},
			},
		},
	}

	triggers := []discovery.Trigger{
		createTestTrigger(TestTriggerID, "record_created", "Test Trigger", nil, nil),
	}

	tasks := []discovery.Task{
		createTestTask(definingTaskID, "variable", "Create myVar", TestWorkflowID, TestTriggerID,
			definingTaskRawData,
			map[string]string{"variable.myVar": "myVar"}),
		createTestTask(otherTaskID, "variable", "Create otherVar", TestWorkflowID, definingTaskID,
			otherTaskRawData,
			// This task also has myVar in FieldRefs even though it doesn't define it
			map[string]string{"variable.myVar": "myVar", "variable.otherVar": "otherVar"}),
		createTestTask(updateTaskID, "update_variable", "Update myVar", TestWorkflowID, otherTaskID,
			updateTaskRawData, nil),
	}

	automations := []discovery.Automation{
		createTestAutomation(TestAutomationID, "Test Automation", TestWorkflowID, "ACTIVE", triggers, tasks),
	}

	imports := []ImportBlock{
		{ID: TestAutomationID, ResourceType: "elementum_automation", ResourceName: "test_auto"},
		{ID: TestAutomationID + ":" + TestTriggerID, ResourceType: "elementum_record_created_trigger", ResourceName: "test_trigger"},
		{ID: TestWorkflowID + ":" + definingTaskID, ResourceType: "elementum_variable_task", ResourceName: "create_myvar"},
		{ID: TestWorkflowID + ":" + otherTaskID, ResourceType: "elementum_variable_task", ResourceName: "create_othervar"},
		{ID: TestWorkflowID + ":" + updateTaskID, ResourceType: "elementum_update_variable_task", ResourceName: "update_myvar"},
	}

	app := &discovery.App{
		ID:          TestAppID,
		Name:        "Test App",
		Automations: automations,
	}

	generator := NewTaskHCLGenerator(app, imports, make(map[string]string), automations)
	hcl := generator.GenerateAll()

	t.Logf("Generated HCL:\n%s", hcl)

	// The update_variable_task should reference the DEFINING task (create_myvar),
	// not the other task (create_othervar)
	assertHCLContains(t, hcl, `elementum_variable_task.create_myvar.refs["myVar"]`)
	assertHCLNotContains(t, hcl, `elementum_variable_task.create_othervar.refs["myVar"]`)
}

// ============================================================================
// ISS-76: Execute Script Task — Block Comment Escaping Tests
// ============================================================================

func TestExecuteScriptTask_BlockComments(t *testing.T) {
	// Reproducer for ISS-76: block comments (/** ... **/) must survive export
	code := "/**\n * Access your input values from the `input.parameters` object\n * Example:\n * const { participants } = input.parameters;\n**/\n\nconst { participants } = input.parameters;\nreturn { participants };"

	app := createTestApp(TestAppID, "Test App")
	automations := []discovery.Automation{
		createTestAutomation(
			TestAutomationID,
			"Test Automation",
			TestWorkflowID,
			"ACTIVE",
			[]discovery.Trigger{{ID: TestTriggerID, Type: "record_created", RawData: buildMinimalTriggerRawData("record_created")}},
			[]discovery.Task{{
				ID:         TestTaskID,
				Type:       "execute_script",
				Name:       "Script With Block Comments",
				WorkflowID: TestWorkflowID,
				ParentID:   TestTriggerID,
				RawData: func() map[string]interface{} {
					data := buildMinimalTaskRawData("execute_script")
					data["code"] = code
					return data
				}(),
			}},
		),
	}

	imports := []ImportBlock{
		{ID: TestAutomationID, ResourceType: "elementum_automation", ResourceName: "test_automation"},
		{ID: TestAutomationID + ":" + TestTriggerID, ResourceType: "elementum_record_created_trigger", ResourceName: "test_trigger"},
		{ID: TestWorkflowID + ":" + TestTaskID, ResourceType: "elementum_execute_script_task", ResourceName: "script_with_block_comments"},
	}

	generator := NewTaskHCLGenerator(app, imports, make(map[string]string), automations)
	hcl := generator.GenerateAll()

	t.Logf("Generated HCL:\n%s", hcl)

	// The code field MUST use heredoc format for multiline code
	assertHCLContains(t, hcl, "<<-")

	// Block comment markers must appear literally in the output
	assertHCLContains(t, hcl, "/**")
	assertHCLContains(t, hcl, "**/")

	// The full block comment content must be preserved
	assertHCLContains(t, hcl, "* Access your input values")
	assertHCLContains(t, hcl, "const { participants } = input.parameters;")
}

func TestExecuteScriptTask_TemplateInterpolation(t *testing.T) {
	// JavaScript template literals use ${expr} which must be escaped to $${expr}
	// to prevent HCL from interpreting them as Terraform interpolation
	code := "const name = `Hello ${user.name}`;\nconst url = `https://api.com/${id}/data`;\nreturn { name, url };"

	app := createTestApp(TestAppID, "Test App")
	automations := []discovery.Automation{
		createTestAutomation(
			TestAutomationID,
			"Test Automation",
			TestWorkflowID,
			"ACTIVE",
			[]discovery.Trigger{{ID: TestTriggerID, Type: "record_created", RawData: buildMinimalTriggerRawData("record_created")}},
			[]discovery.Task{{
				ID:         TestTaskID,
				Type:       "execute_script",
				Name:       "Script With Templates",
				WorkflowID: TestWorkflowID,
				ParentID:   TestTriggerID,
				RawData: func() map[string]interface{} {
					data := buildMinimalTaskRawData("execute_script")
					data["code"] = code
					return data
				}(),
			}},
		),
	}

	imports := []ImportBlock{
		{ID: TestAutomationID, ResourceType: "elementum_automation", ResourceName: "test_automation"},
		{ID: TestAutomationID + ":" + TestTriggerID, ResourceType: "elementum_record_created_trigger", ResourceName: "test_trigger"},
		{ID: TestWorkflowID + ":" + TestTaskID, ResourceType: "elementum_execute_script_task", ResourceName: "script_with_templates"},
	}

	generator := NewTaskHCLGenerator(app, imports, make(map[string]string), automations)
	hcl := generator.GenerateAll()

	t.Logf("Generated HCL:\n%s", hcl)

	// ${user.name} must be escaped to $${user.name} so HCL treats it as literal
	assertHCLContains(t, hcl, "$${user.name}")
	assertHCLContains(t, hcl, "$${id}")

	// Verify escaping is correct: every ${...} must be preceded by $$ (i.e., $${...})
	// We can't use assertHCLNotContains since "$${x}" contains "${x}" as a substring.
	// Instead, count occurrences: escaped ($${) count must equal total ${ count.
	totalDollarBrace := strings.Count(hcl, "${")
	escapedDollarBrace := strings.Count(hcl, "$${")
	if totalDollarBrace != escapedDollarBrace {
		t.Errorf("Found %d total ${ but only %d escaped $${ — some interpolations are not escaped", totalDollarBrace, escapedDollarBrace)
	}
}

func TestExecuteScriptTask_SingleLineCode(t *testing.T) {
	// Single-line code should stay as a quoted string, not heredoc
	code := "return { result: 42 };"

	app := createTestApp(TestAppID, "Test App")
	automations := []discovery.Automation{
		createTestAutomation(
			TestAutomationID,
			"Test Automation",
			TestWorkflowID,
			"ACTIVE",
			[]discovery.Trigger{{ID: TestTriggerID, Type: "record_created", RawData: buildMinimalTriggerRawData("record_created")}},
			[]discovery.Task{{
				ID:         TestTaskID,
				Type:       "execute_script",
				Name:       "Simple Script",
				WorkflowID: TestWorkflowID,
				ParentID:   TestTriggerID,
				RawData: func() map[string]interface{} {
					data := buildMinimalTaskRawData("execute_script")
					data["code"] = code
					return data
				}(),
			}},
		),
	}

	imports := []ImportBlock{
		{ID: TestAutomationID, ResourceType: "elementum_automation", ResourceName: "test_automation"},
		{ID: TestAutomationID + ":" + TestTriggerID, ResourceType: "elementum_record_created_trigger", ResourceName: "test_trigger"},
		{ID: TestWorkflowID + ":" + TestTaskID, ResourceType: "elementum_execute_script_task", ResourceName: "simple_script"},
	}

	generator := NewTaskHCLGenerator(app, imports, make(map[string]string), automations)
	hcl := generator.GenerateAll()

	t.Logf("Generated HCL:\n%s", hcl)

	// Single-line code should use quoted string, not heredoc
	assertHCLContains(t, hcl, `"return { result: 42 };"`)
	assertHCLNotContains(t, hcl, "<<-")
}

func TestExecuteScriptTask_CodeWithEOTDelimiter(t *testing.T) {
	// Code containing "EOT" at the start of a line should use a different delimiter
	code := "// This function returns EOT\nconst result = 'EOT';\nreturn { result };"

	app := createTestApp(TestAppID, "Test App")
	automations := []discovery.Automation{
		createTestAutomation(
			TestAutomationID,
			"Test Automation",
			TestWorkflowID,
			"ACTIVE",
			[]discovery.Trigger{{ID: TestTriggerID, Type: "record_created", RawData: buildMinimalTriggerRawData("record_created")}},
			[]discovery.Task{{
				ID:         TestTaskID,
				Type:       "execute_script",
				Name:       "EOT Script",
				WorkflowID: TestWorkflowID,
				ParentID:   TestTriggerID,
				RawData: func() map[string]interface{} {
					data := buildMinimalTaskRawData("execute_script")
					data["code"] = code
					return data
				}(),
			}},
		),
	}

	imports := []ImportBlock{
		{ID: TestAutomationID, ResourceType: "elementum_automation", ResourceName: "test_automation"},
		{ID: TestAutomationID + ":" + TestTriggerID, ResourceType: "elementum_record_created_trigger", ResourceName: "test_trigger"},
		{ID: TestWorkflowID + ":" + TestTaskID, ResourceType: "elementum_execute_script_task", ResourceName: "eot_script"},
	}

	generator := NewTaskHCLGenerator(app, imports, make(map[string]string), automations)
	hcl := generator.GenerateAll()

	t.Logf("Generated HCL:\n%s", hcl)

	// Should use heredoc (multiline) but code content must be preserved
	assertHCLContains(t, hcl, "<<-")
	assertHCLContains(t, hcl, "const result = 'EOT';")
}

func TestExecuteScriptTask_EdgeCases(t *testing.T) {
	tests := []struct {
		name             string
		code             string
		shouldContain    []string
		shouldNotContain []string
	}{
		{
			name:          "code starting with block comment",
			code:          "/* start */\nvar x = 1;\nreturn { x };",
			shouldContain: []string{"/* start */", "var x = 1;"},
		},
		{
			name:          "nested comment-like patterns",
			code:          "var re = /\\/* match *\\//;\n/* real comment */\nreturn { re };",
			shouldContain: []string{"/* real comment */"},
		},
		{
			name:          "line comments preserved",
			code:          "// This is a line comment\nvar x = 1; // inline\nreturn { x };",
			shouldContain: []string{"// This is a line comment", "// inline"},
		},
		{
			name:          "close comment in JS string",
			code:          "var end = \"end of comment: */\";\n/* actual comment */\nreturn { end };",
			shouldContain: []string{`end of comment: */`, "/* actual comment */"},
		},
		{
			name:          "mixed comment styles",
			code:          "/* block comment */\n// line comment\n/* another block */\nreturn { ok: true };",
			shouldContain: []string{"/* block comment */", "// line comment", "/* another block */"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := createTestApp(TestAppID, "Test App")
			automations := []discovery.Automation{
				createTestAutomation(
					TestAutomationID,
					"Test Automation",
					TestWorkflowID,
					"ACTIVE",
					[]discovery.Trigger{{ID: TestTriggerID, Type: "record_created", RawData: buildMinimalTriggerRawData("record_created")}},
					[]discovery.Task{{
						ID:         TestTaskID,
						Type:       "execute_script",
						Name:       "Edge Case Script",
						WorkflowID: TestWorkflowID,
						ParentID:   TestTriggerID,
						RawData: func() map[string]interface{} {
							data := buildMinimalTaskRawData("execute_script")
							data["code"] = tt.code
							return data
						}(),
					}},
				),
			}

			imports := []ImportBlock{
				{ID: TestAutomationID, ResourceType: "elementum_automation", ResourceName: "test_automation"},
				{ID: TestAutomationID + ":" + TestTriggerID, ResourceType: "elementum_record_created_trigger", ResourceName: "test_trigger"},
				{ID: TestWorkflowID + ":" + TestTaskID, ResourceType: "elementum_execute_script_task", ResourceName: "edge_case_script"},
			}

			generator := NewTaskHCLGenerator(app, imports, make(map[string]string), automations)
			hcl := generator.GenerateAll()

			t.Logf("Generated HCL:\n%s", hcl)

			assertHCLContains(t, hcl, tt.shouldContain...)
			if len(tt.shouldNotContain) > 0 {
				assertHCLNotContains(t, hcl, tt.shouldNotContain...)
			}
		})
	}
}

func TestExecuteScriptTask_EmptyCode(t *testing.T) {
	// Empty code string should not produce a code attribute
	app := createTestApp(TestAppID, "Test App")
	automations := []discovery.Automation{
		createTestAutomation(
			TestAutomationID,
			"Test Automation",
			TestWorkflowID,
			"ACTIVE",
			[]discovery.Trigger{{ID: TestTriggerID, Type: "record_created", RawData: buildMinimalTriggerRawData("record_created")}},
			[]discovery.Task{{
				ID:         TestTaskID,
				Type:       "execute_script",
				Name:       "Empty Script",
				WorkflowID: TestWorkflowID,
				ParentID:   TestTriggerID,
				RawData: func() map[string]interface{} {
					data := buildMinimalTaskRawData("execute_script")
					data["code"] = ""
					return data
				}(),
			}},
		),
	}

	imports := []ImportBlock{
		{ID: TestAutomationID, ResourceType: "elementum_automation", ResourceName: "test_automation"},
		{ID: TestAutomationID + ":" + TestTriggerID, ResourceType: "elementum_record_created_trigger", ResourceName: "test_trigger"},
		{ID: TestWorkflowID + ":" + TestTaskID, ResourceType: "elementum_execute_script_task", ResourceName: "empty_script"},
	}

	generator := NewTaskHCLGenerator(app, imports, make(map[string]string), automations)
	hcl := generator.GenerateAll()

	t.Logf("Generated HCL:\n%s", hcl)

	// Empty code should not produce a code attribute (the condition `strVal != ""` guards this)
	assertHCLNotContains(t, hcl, "code =")
}
