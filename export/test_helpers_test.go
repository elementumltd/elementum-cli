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
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/elementumltd/elementum-cli/discovery"
)

// ============================================================================
// Fixture Loading Functions
// ============================================================================

// loadTriggerFixture loads a trigger fixture from testdata/triggers
func loadTriggerFixture(t *testing.T, name string) *discovery.Trigger {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", "triggers", name+".json"))
	if err != nil {
		t.Fatalf("Failed to load trigger fixture %s: %v", name, err)
	}
	var trigger discovery.Trigger
	if err := json.Unmarshal(data, &trigger); err != nil {
		t.Fatalf("Failed to parse trigger fixture %s: %v", name, err)
	}
	return &trigger
}

// loadTaskFixture loads a task fixture from testdata/tasks
func loadTaskFixture(t *testing.T, name string) *discovery.Task {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", "tasks", name+".json"))
	if err != nil {
		t.Fatalf("Failed to load task fixture %s: %v", name, err)
	}
	var task discovery.Task
	if err := json.Unmarshal(data, &task); err != nil {
		t.Fatalf("Failed to parse task fixture %s: %v", name, err)
	}
	return &task
}

// loadAutomationFixture loads an automation fixture from testdata/automations
func loadAutomationFixture(t *testing.T, name string) *discovery.Automation {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", "automations", name+".json"))
	if err != nil {
		t.Fatalf("Failed to load automation fixture %s: %v", name, err)
	}
	var automation discovery.Automation
	if err := json.Unmarshal(data, &automation); err != nil {
		t.Fatalf("Failed to parse automation fixture %s: %v", name, err)
	}
	return &automation
}

// loadExpectedHCL loads expected HCL output from testdata/expected
func loadExpectedHCL(t *testing.T, category, name string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", "expected", category, name+".hcl"))
	if err != nil {
		t.Fatalf("Failed to load expected HCL %s/%s: %v", category, name, err)
	}
	return string(data)
}

// ============================================================================
// Test Data Creation Functions
// ============================================================================

// createTestApp creates a minimal App for testing
func createTestApp(id, name string) *discovery.App {
	return &discovery.App{
		ID:        id,
		Name:      name,
		Namespace: strings.ToLower(strings.ReplaceAll(name, " ", "_")),
	}
}

// createTestAutomation creates an automation with specified triggers and tasks
func createTestAutomation(id, name, workflowID string, status string, triggers []discovery.Trigger, tasks []discovery.Task) discovery.Automation {
	hasPublished := status == "ACTIVE"
	return discovery.Automation{
		ID:           id,
		Name:         name,
		Status:       status,
		WorkflowID:   workflowID,
		HasPublished: hasPublished,
		HasDraft:     !hasPublished,
		Triggers:     triggers,
		Tasks:        tasks,
	}
}

// createTestTrigger creates a trigger for testing
func createTestTrigger(id, triggerType, name string, rawData map[string]interface{}, fieldRefs map[string]string) discovery.Trigger {
	if rawData == nil {
		rawData = make(map[string]interface{})
	}
	if fieldRefs == nil {
		fieldRefs = make(map[string]string)
	}
	return discovery.Trigger{
		ID:        id,
		Type:      triggerType,
		Name:      name,
		RawData:   rawData,
		FieldRefs: fieldRefs,
	}
}

// createTestTask creates a task for testing
func createTestTask(id, taskType, name, workflowID, parentID string, rawData map[string]interface{}, fieldRefs map[string]string) discovery.Task {
	if rawData == nil {
		rawData = make(map[string]interface{})
	}
	if fieldRefs == nil {
		fieldRefs = make(map[string]string)
	}
	return discovery.Task{
		ID:         id,
		Type:       taskType,
		Name:       name,
		WorkflowID: workflowID,
		ParentID:   parentID,
		RawData:    rawData,
		FieldRefs:  fieldRefs,
	}
}

// ImportResource is a helper struct for creating import blocks
type ImportResource struct {
	ID           string
	ResourceType string
	ResourceName string
}

// createTestImports creates import blocks for testing from a list of resources
func createTestImports(resources ...ImportResource) []ImportBlock {
	imports := make([]ImportBlock, len(resources))
	for i, r := range resources {
		imports[i] = ImportBlock(r)
	}
	return imports
}

// createTestUUIDMap creates a UUID map for testing
func createTestUUIDMap(mappings map[string]string) map[string]string {
	if mappings == nil {
		return make(map[string]string)
	}
	return mappings
}

// ============================================================================
// Assertion Helpers
// ============================================================================

// assertHCLContains checks that HCL output contains all expected strings
func assertHCLContains(t *testing.T, hcl string, expected ...string) {
	t.Helper()
	for _, exp := range expected {
		if !strings.Contains(hcl, exp) {
			t.Errorf("Expected HCL to contain %q, but it doesn't.\nHCL:\n%s", exp, hcl)
		}
	}
}

// assertHCLNotContains checks that HCL output does not contain certain strings
func assertHCLNotContains(t *testing.T, hcl string, unexpected ...string) {
	t.Helper()
	for _, unexp := range unexpected {
		if strings.Contains(hcl, unexp) {
			t.Errorf("Expected HCL NOT to contain %q, but it does.\nHCL:\n%s", unexp, hcl)
		}
	}
}

// validateHCLSyntax performs basic HCL syntax validation
func validateHCLSyntax(t *testing.T, hcl string) {
	t.Helper()

	// Check for balanced braces
	openBraces := strings.Count(hcl, "{")
	closeBraces := strings.Count(hcl, "}")
	if openBraces != closeBraces {
		t.Errorf("Unbalanced braces: %d open, %d close\nHCL:\n%s", openBraces, closeBraces, hcl)
	}

	// Check for balanced brackets
	openBrackets := strings.Count(hcl, "[")
	closeBrackets := strings.Count(hcl, "]")
	if openBrackets != closeBrackets {
		t.Errorf("Unbalanced brackets: %d open, %d close\nHCL:\n%s", openBrackets, closeBrackets, hcl)
	}

	// Check for balanced parentheses
	openParens := strings.Count(hcl, "(")
	closeParens := strings.Count(hcl, ")")
	if openParens != closeParens {
		t.Errorf("Unbalanced parentheses: %d open, %d close\nHCL:\n%s", openParens, closeParens, hcl)
	}
}

// assertNoNullAttributes checks that HCL doesn't contain null attribute assignments
func assertNoNullAttributes(t *testing.T, hcl string) {
	t.Helper()
	// Match patterns like "attribute = null" but not inside strings
	nullPattern := regexp.MustCompile(`^\s*\w+\s*=\s*null\s*$`)
	lines := strings.Split(hcl, "\n")
	for i, line := range lines {
		if nullPattern.MatchString(line) {
			t.Errorf("Found null attribute at line %d: %q\nHCL should have null attributes stripped", i+1, line)
		}
	}
}

// assertReferencesResolved checks that raw UUIDs have been resolved to terraform references
func assertReferencesResolved(t *testing.T, hcl string) {
	t.Helper()
	// UUID pattern
	uuidPattern := regexp.MustCompile(`"[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}"`)
	matches := uuidPattern.FindAllString(hcl, -1)
	if len(matches) > 0 {
		t.Errorf("Found unresolved UUIDs in HCL: %v\nHCL:\n%s", matches, hcl)
	}
}

// assertResourceBlockExists checks that a resource block with the given type and name exists
func assertResourceBlockExists(t *testing.T, hcl, resourceType, resourceName string) {
	t.Helper()
	pattern := regexp.MustCompile(`resource\s+"` + regexp.QuoteMeta(resourceType) + `"\s+"` + regexp.QuoteMeta(resourceName) + `"\s*\{`)
	if !pattern.MatchString(hcl) {
		t.Errorf("Expected resource block %q %q not found in HCL:\n%s", resourceType, resourceName, hcl)
	}
}

// assertAttributeValue checks that an attribute has the expected value in the HCL
func assertAttributeValue(t *testing.T, hcl, attributeName, expectedValue string) {
	t.Helper()
	// Simple pattern to find attribute = value
	pattern := regexp.MustCompile(`\b` + regexp.QuoteMeta(attributeName) + `\s*=\s*` + regexp.QuoteMeta(expectedValue))
	if !pattern.MatchString(hcl) {
		t.Errorf("Expected attribute %s = %s not found in HCL:\n%s", attributeName, expectedValue, hcl)
	}
}

// assertHCLMatchesExpected compares generated HCL against expected HCL, ignoring whitespace differences
func assertHCLMatchesExpected(t *testing.T, got, expected string) {
	t.Helper()
	gotNormalized := normalizeHCL(got)
	expectedNormalized := normalizeHCL(expected)
	if gotNormalized != expectedNormalized {
		t.Errorf("HCL mismatch.\nGot:\n%s\n\nExpected:\n%s", got, expected)
	}
}

// normalizeHCL normalizes HCL for comparison by trimming whitespace
func normalizeHCL(hcl string) string {
	lines := strings.Split(hcl, "\n")
	var normalized []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			normalized = append(normalized, trimmed)
		}
	}
	return strings.Join(normalized, "\n")
}

// ============================================================================
// UUID Generation Helpers
// ============================================================================

// Test UUIDs for consistent test data
const (
	TestAppID              = "11111111-1111-1111-1111-111111111111"
	TestAutomationID       = "22222222-2222-2222-2222-222222222222"
	TestWorkflowID         = "33333333-3333-3333-3333-333333333333"
	TestTriggerID          = "44444444-4444-4444-4444-444444444444"
	TestTaskID             = "55555555-5555-5555-5555-555555555555"
	TestTask2ID            = "55555555-5555-5555-5555-555555555556"
	TestFieldID            = "66666666-6666-6666-6666-666666666666"
	TestField2ID           = "66666666-6666-6666-6666-666666666667"
	TestAgentID            = "77777777-7777-7777-7777-777777777777"
	TestDatamineID         = "88888888-8888-8888-8888-888888888888"
	TestEmailAliasID       = "99999999-9999-9999-9999-999999999999"
	TestApprovalTemplateID = "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
	TestDocumentModelID    = "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"
	TestElementID          = "cccccccc-cccc-cccc-cccc-cccccccccccc"
	TestLayoutID           = "dddddddd-dddd-dddd-dddd-dddddddddddd"
	TestCloudLinkID        = "eeeeeeee-eeee-eeee-eeee-eeeeeeeeeeee"
	TestStoredFunctionID   = "ffffffff-ffff-ffff-ffff-ffffffffffff"
	TestTargetAutomationID = "12121212-1212-1212-1212-121212121212"

	// System field UUIDs (used by Elementum for special fields)
	SystemFieldIDUUID          = "10000001-2000-4000-a000-800000000000"
	SystemFieldTitleUUID       = "10000001-2000-4000-a000-800000000001"
	SystemFieldStatusUUID      = "10000001-2000-4000-a000-800000000002"
	SystemFieldCreatedByUUID   = "10000001-2000-4000-a000-800000000003"
	SystemFieldAttachmentsUUID = "10000001-2000-4000-a000-800000000003"
	SystemFieldRecordURLUUID   = "10000001-2000-4000-a000-800000000005"
)

// ============================================================================
// Complex Test Data Builders
// ============================================================================

// buildTestAutomationWithTriggerAndTask creates a complete automation for testing
func buildTestAutomationWithTriggerAndTask(
	automationName string,
	triggerType string,
	triggerRawData map[string]interface{},
	taskType string,
	taskRawData map[string]interface{},
) (*discovery.App, []discovery.Automation, []ImportBlock) {
	app := createTestApp(TestAppID, "Test App")

	trigger := createTestTrigger(TestTriggerID, triggerType, "Test Trigger", triggerRawData, nil)
	task := createTestTask(TestTaskID, taskType, "Test Task", TestWorkflowID, TestTriggerID, taskRawData, nil)

	automation := createTestAutomation(
		TestAutomationID,
		automationName,
		TestWorkflowID,
		"ACTIVE",
		[]discovery.Trigger{trigger},
		[]discovery.Task{task},
	)

	imports := createTestImports(
		ImportResource{TestAutomationID, "elementum_automation", SanitizeName(automationName)},
		ImportResource{TestAutomationID + ":" + TestTriggerID, "elementum_" + triggerType + "_trigger", "test_trigger"},
		ImportResource{TestAutomationID + ":" + TestTaskID, "elementum_" + taskType + "_task", "test_task"},
	)

	return app, []discovery.Automation{automation}, imports
}

// buildMinimalTriggerRawData creates minimal RawData for a trigger type
func buildMinimalTriggerRawData(triggerType string) map[string]interface{} {
	baseData := map[string]interface{}{
		"__typename": getGraphQLTypename(triggerType, "trigger"),
		"id":         TestTriggerID,
	}

	// Add type-specific minimal data
	switch triggerType {
	case "record_created", "record_updated", "attachment_added", "comment_added", "slack_message", "agent_conversation_ended":
		baseData["aspect"] = map[string]interface{}{
			"id":   TestAppID,
			"name": "Test App",
		}
	case "webhook":
		baseData["authenticated"] = false
		baseData["url"] = "https://example.com/webhook"
	case "on_demand":
		baseData["parameters"] = []interface{}{}
	case "email_ingestion":
		baseData["emailAlias"] = map[string]interface{}{
			"id":    TestEmailAliasID,
			"alias": "test@example.com",
		}
	case "datamine":
		baseData["datamine"] = map[string]interface{}{
			"id":   TestDatamineID,
			"name": "Test Datamine",
		}
		baseData["fireOnAlert"] = true
		baseData["fireOnRecovery"] = false
	case "approval_chain":
		baseData["aspect"] = map[string]interface{}{
			"id":   TestAppID,
			"name": "Test App",
		}
		baseData["approvalChainTemplate"] = map[string]interface{}{
			"id":   TestApprovalTemplateID,
			"name": "Test Approval Template",
		}
		baseData["status"] = "APPROVED"
	}

	return baseData
}

// buildMinimalTaskRawData creates minimal RawData for a task type
func buildMinimalTaskRawData(taskType string) map[string]interface{} {
	baseData := map[string]interface{}{
		"__typename": getGraphQLTypename(taskType, "task"),
		"id":         TestTaskID,
		"name":       "Test Task",
	}

	// Add type-specific minimal data
	switch taskType {
	case "variable":
		baseData["variables"] = []interface{}{
			map[string]interface{}{
				"__typename": "WorkflowVariableTaskParameterCreate",
				"id":         "var-1",
				"name":       "test_variable",
				"type":       "TEXT",
				"createValue": map[string]interface{}{
					"id":    "ref-1",
					"label": "test value",
					"value": "test value",
				},
			},
		}
	case "update_variable":
		baseData["variables"] = []interface{}{
			map[string]interface{}{
				"__typename": "WorkflowVariableTaskParameterUpdate",
				"variable": map[string]interface{}{
					"id":    "var-ref-1",
					"label": "test_variable",
					"value": `{"variableReference":{"id":"var-123"}}`,
				},
				"updateValue": map[string]interface{}{
					"id":    "ref-1",
					"label": "new value",
					"value": "new value",
				},
			},
		}
	case "message":
		baseData["contentsReference"] = map[string]interface{}{
			"id":    "ref-1",
			"label": "Hello World",
			"value": "Hello World",
		}
	case "update_field", "create_record":
		baseData["aspect"] = map[string]interface{}{
			"id":   TestAppID,
			"name": "Test App",
		}
		baseData["workflowFields"] = []interface{}{}
	case "record_search":
		baseData["aspect"] = map[string]interface{}{
			"id":   TestAppID,
			"name": "Test App",
		}
		baseData["limit"] = float64(10)
	case "ai_agent":
		baseData["agent"] = map[string]interface{}{
			"id":   TestAgentID,
			"name": "Test Agent",
		}
		baseData["agentPrompt"] = map[string]interface{}{
			"id":    "ref-1",
			"label": "Test prompt",
			"value": "Test prompt",
		}
	case "send_email":
		baseData["subject"] = map[string]interface{}{
			"id":    "ref-1",
			"label": "Subject",
			"value": "Subject",
		}
		baseData["messageBody"] = map[string]interface{}{
			"id":    "ref-2",
			"label": "Body",
			"value": "Body",
		}
	case "for_each":
		baseData["forEach"] = map[string]interface{}{
			"id":    "ref-1",
			"label": "Records",
			"value": "Records",
		}
		baseData["children"] = []interface{}{}
	case "switch":
		baseData["children"] = []interface{}{}
	case "notification":
		baseData["messageReference"] = map[string]interface{}{
			"id":    "ref-1",
			"label": "Notification",
			"value": "Notification",
		}
	case "api":
		baseData["method"] = "GET"
		baseData["urlReference"] = map[string]interface{}{
			"id":    "ref-1",
			"label": "https://api.example.com",
			"value": "https://api.example.com",
		}
	case "calculation":
		baseData["calculations"] = []interface{}{
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
		}
	case "save_attachment":
		baseData["recordId"] = map[string]interface{}{
			"id":    "ref-1",
			"label": "Record",
			"value": "Record",
		}
		baseData["tag"] = map[string]interface{}{
			"id":    "ref-2",
			"label": "attachment",
			"value": "attachment",
		}
	case "add_watcher":
		baseData["recordReference"] = map[string]interface{}{
			"id":    "ref-1",
			"label": "Record",
			"value": "Record",
		}
	case "relate_records":
		baseData["sourceRecordIds"] = map[string]interface{}{
			"id":    "ref-1",
			"label": "Sources",
			"value": "Sources",
		}
		baseData["targetRecordId"] = map[string]interface{}{
			"id":    "ref-2",
			"label": "Target",
			"value": "Target",
		}
	case "find_related_records":
		baseData["recordId"] = map[string]interface{}{
			"id":    "ref-1",
			"label": "Record",
			"value": "Record",
		}
		baseData["relatedAspect"] = map[string]interface{}{
			"id":   TestElementID,
			"name": "Related Element",
		}
	case "user_search":
		// Minimal filter structure for user_search (email with static value)
		baseData["filter"] = map[string]interface{}{
			"type":    "equals",
			"fieldId": "email",
			"value": map[string]interface{}{
				"value": "test@example.com",
			},
		}
	case "ai_classify":
		baseData["textToClassify"] = map[string]interface{}{
			"id":    "ref-1",
			"label": "Text",
			"value": "Text",
		}
		baseData["categories"] = []interface{}{}
	case "ai_summarize":
		baseData["textToSummarize"] = map[string]interface{}{
			"id":    "ref-1",
			"label": "Text",
			"value": "Text",
		}
	case "ai_transform":
		baseData["prompt"] = map[string]interface{}{
			"id":    "ref-1",
			"label": "Transform this",
			"value": "Transform this",
		}
	case "ai_file_read":
		baseData["documentModel"] = map[string]interface{}{
			"id":   TestDocumentModelID,
			"name": "Test Document Model",
		}
		baseData["attachment"] = map[string]interface{}{
			"id":    "ref-1",
			"label": "File",
			"value": "File",
		}
	case "approval_chain":
		baseData["approvers"] = map[string]interface{}{
			"id":    "ref-1",
			"label": "Approvers",
			"value": "Approvers",
		}
	case "approval_status_update":
		baseData["approvalChainTemplate"] = map[string]interface{}{
			"id": "act-123",
		}
		baseData["status"] = "APPROVED"
	case "aspect_record_field_locking":
		baseData["aspect"] = map[string]interface{}{
			"id":   TestAppID,
			"name": "Test App",
		}
		baseData["recordId"] = map[string]interface{}{
			"id":    "ref-1",
			"label": "Record",
			"value": "Record",
		}
		baseData["fieldsToLock"] = []interface{}{}
		baseData["fieldsToUnlock"] = []interface{}{}
	case "bulk_excel":
		baseData["aspect"] = map[string]interface{}{
			"id":   TestAppID,
			"name": "Test App",
		}
		baseData["documentModel"] = map[string]interface{}{
			"id":   TestDocumentModelID,
			"name": "Excel Model",
		}
		baseData["attachment"] = map[string]interface{}{
			"id":    "ref-1",
			"label": "File",
			"value": "File",
		}
	case "procedure":
		baseData["cloudLinkId"] = TestCloudLinkID
		baseData["storedFunction"] = map[string]interface{}{
			"id":   TestStoredFunctionID,
			"name": "test_procedure",
		}
		baseData["parameters"] = []interface{}{
			map[string]interface{}{
				"index": float64(0),
				"value": map[string]interface{}{
					"id":    "ref-1",
					"label": "Status",
					"value": "Active",
				},
			},
		}
	case "run_automation":
		baseData["synchronous"] = true
		baseData["automation"] = map[string]interface{}{
			"id":   TestTargetAutomationID,
			"name": "Target Automation",
		}
		baseData["inputMappings"] = []interface{}{}
		baseData["dynamicInputMappings"] = []interface{}{}
		baseData["outputMappings"] = []interface{}{}
	case "execute_script":
		baseData["code"] = "/**\n * Access your input values from the `input.parameters` object\n * Example:\n * const { participants } = input.parameters;\n**/\n\nconst { participants } = input.parameters;\nlet numParticipants = participants.length;\nreturn { numParticipants };"
		baseData["inputs"] = []interface{}{
			map[string]interface{}{
				"name": "participants",
				"value": map[string]interface{}{
					"id":    "ref-1",
					"label": "participants",
					"value": "participants",
				},
			},
		}
	}

	return baseData
}

// getGraphQLTypename returns the GraphQL typename for a trigger or task type
func getGraphQLTypename(typeName, category string) string {
	switch category {
	case "trigger":
		switch typeName {
		case "record_created":
			return "WorkflowRecordCreateTrigger"
		case "record_updated":
			return "WorkflowRecordUpdateTrigger"
		case "attachment_added":
			return "WorkflowAttachmentTrigger"
		case "comment_added":
			return "WorkflowCommentAddedTrigger"
		case "webhook":
			return "WorkflowWebhookTrigger"
		case "on_demand":
			return "WorkflowOnDemandTrigger"
		case "slack_message":
			return "WorkflowSlackMessageTrigger"
		case "email_ingestion":
			return "WorkflowEmailIngestionTrigger"
		case "agent_conversation_ended":
			return "WorkflowConversationEndTrigger"
		case "approval_chain":
			return "WorkflowApprovalChainTrigger"
		case "datamine":
			return "WorkflowDatamineTrigger"
		}
	case "task":
		switch typeName {
		case "switch":
			return "WorkflowSwitchTask"
		case "variable", "update_variable":
			return "WorkflowVariableTask"
		case "update_field":
			return "WorkflowUpdateFieldTask"
		case "create_record":
			return "WorkflowCreateRecordTask"
		case "record_search":
			return "WorkflowRecordSearchTask"
		case "ai_agent":
			return "WorkflowAiAgentTask"
		case "message":
			return "WorkflowMessageTask"
		case "send_email":
			return "WorkflowSendEmailTask"
		case "for_each":
			return "WorkflowForEachTask"
		case "save_attachment":
			return "WorkflowSaveAttachmentTask"
		case "notification":
			return "WorkflowNotificationTask"
		case "api":
			return "WorkflowApiTask"
		case "ai_file_read":
			return "WorkflowAiFileAnalysisTask"
		case "calculation":
			return "WorkflowCalculationTask"
		case "add_watcher":
			return "WorkflowAddWatcherTask"
		case "relate_records":
			return "WorkflowRelateRecordsTask"
		case "find_related_records":
			return "WorkflowFindRelatedRecordsTask"
		case "user_search":
			return "WorkflowUserSearchTask"
		case "ai_classify":
			return "WorkflowAiClassifyTask"
		case "ai_summarize":
			return "WorkflowAiSummarizeTask"
		case "ai_transform":
			return "WorkflowAiTransformTask"
		case "approval_chain":
			return "WorkflowApprovalChainTemplateTask"
		case "approval_status_update":
			return "WorkflowApprovalStatusUpdateTask"
		case "aspect_record_field_locking":
			return "WorkflowAspectRecordFieldLockingTask"
		case "bulk_excel":
			return "WorkflowBulkExcelTask"
		case "procedure":
			return "WorkflowProcedureTask"
		case "run_automation":
			return "WorkflowRunAutomationTask"
		}
	}
	return "Unknown"
}
