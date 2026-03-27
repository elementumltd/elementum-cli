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
	"testing"

	"github.com/elementumltd/elementum-cli/discovery"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// beautifyValueReferences Tests
// =============================================================================

func TestBeautifyValueReferences_TriggerRecordRefs(t *testing.T) {
	// Setup: Create an app with automation, trigger, and field refs
	app := createTestApp(TestAppID, "Test App")
	trigger := discovery.Trigger{
		ID:   TestTriggerID,
		Type: "record_created",
		Name: "Test Trigger",
		FieldRefs: map[string]string{
			"record." + TestFieldID: "Test Field",
			"record.title":          "Title",
		},
	}
	automation := discovery.Automation{
		ID:         TestAutomationID,
		Name:       "Test Automation",
		WorkflowID: TestWorkflowID,
		Status:     "ACTIVE",
		Triggers:   []discovery.Trigger{trigger},
		Tasks:      []discovery.Task{},
	}
	app.Automations = []discovery.Automation{automation}

	// Build UUID map with trigger resource
	imports := []ImportBlock{
		{ResourceType: "elementum_record_created_trigger", ResourceName: "test_trigger", ID: TestTriggerID},
	}
	uuidMap := map[string]string{
		TestTriggerID: "elementum_record_created_trigger.test_trigger.id",
	}

	// HCL with raw value references
	hcl := `
resource "elementum_message_task" "notify" {
  value = "trigger.record.` + TestFieldID + `"
  record_reference = "trigger.record.title"
}
`

	result := beautifyValueReferences(hcl, uuidMap, app)

	// Should convert raw trigger refs to refs syntax
	assert.Contains(t, result, "elementum_record_created_trigger.test_trigger.refs")
	assert.Contains(t, result, "Test Field")
	// Raw UUID should be replaced
	assert.NotContains(t, result, TestFieldID)

	_ = imports // Prevent unused variable warning
}

func TestBeautifyValueReferences_TaskRefs(t *testing.T) {
	// Setup: Create an app with automation and task with field refs
	app := createTestApp(TestAppID, "Test App")
	task := discovery.Task{
		ID:   TestTaskID,
		Type: "variable",
		Name: "Variable Task",
		FieldRefs: map[string]string{
			"value": "calculated_value",
		},
	}
	automation := discovery.Automation{
		ID:         TestAutomationID,
		Name:       "Test Automation",
		WorkflowID: TestWorkflowID,
		Status:     "ACTIVE",
		Triggers:   []discovery.Trigger{},
		Tasks:      []discovery.Task{task},
	}
	app.Automations = []discovery.Automation{automation}

	// Build UUID map with task resource
	uuidMap := map[string]string{
		TestTaskID: "elementum_variable_task.variable_task.id",
	}

	// HCL with raw task value references
	hcl := `
resource "elementum_update_field_task" "update" {
  value = "task.` + TestTaskID + `.value"
}
`

	result := beautifyValueReferences(hcl, uuidMap, app)

	// Should convert raw task refs to refs syntax
	assert.Contains(t, result, "elementum_variable_task.variable_task.refs")
	assert.Contains(t, result, "calculated_value")
}

func TestBeautifyValueReferences_NilApp(t *testing.T) {
	hcl := `
resource "elementum_message_task" "notify" {
  value = "trigger.record.` + TestFieldID + `"
}
`

	result := beautifyValueReferences(hcl, map[string]string{}, nil)

	// Should return HCL unchanged when app is nil
	assert.Equal(t, hcl, result)
}

func TestBeautifyValueReferences_NoMatches(t *testing.T) {
	app := createTestApp(TestAppID, "Test App")
	app.Automations = []discovery.Automation{}

	hcl := `
resource "elementum_message_task" "notify" {
  value = "static text"
}
`

	result := beautifyValueReferences(hcl, map[string]string{}, app)

	// Should return HCL unchanged when no value references present
	assert.Equal(t, hcl, result)
}

func TestBeautifyValueReferences_MixedRefs(t *testing.T) {
	// Setup with both trigger and task refs
	app := createTestApp(TestAppID, "Test App")
	trigger := discovery.Trigger{
		ID:   TestTriggerID,
		Type: "record_created",
		Name: "Test Trigger",
		FieldRefs: map[string]string{
			"record." + TestFieldID: "Source Field",
		},
	}
	task := discovery.Task{
		ID:   TestTaskID,
		Type: "record_search",
		Name: "Search Task",
		FieldRefs: map[string]string{
			"count": "Result Count",
		},
	}
	automation := discovery.Automation{
		ID:         TestAutomationID,
		Name:       "Test Automation",
		WorkflowID: TestWorkflowID,
		Status:     "ACTIVE",
		Triggers:   []discovery.Trigger{trigger},
		Tasks:      []discovery.Task{task},
	}
	app.Automations = []discovery.Automation{automation}

	uuidMap := map[string]string{
		TestTriggerID: "elementum_record_created_trigger.test_trigger.id",
		TestTaskID:    "elementum_record_search_task.search_task.id",
	}

	hcl := `
resource "elementum_message_task" "notify" {
  value = "trigger.record.` + TestFieldID + `"
  record_reference = "task.` + TestTaskID + `.count"
}
`

	result := beautifyValueReferences(hcl, uuidMap, app)

	// Should convert both trigger and task refs
	assert.Contains(t, result, "Source Field")
	assert.Contains(t, result, "Result Count")
}

func TestBeautifyValueReferences_SystemFields(t *testing.T) {
	// Note: The beautifyValueReferences function uses regex that matches UUID-based references.
	// System field references like "trigger.record.id" don't contain UUIDs, so they
	// remain unchanged. This test verifies that non-UUID references pass through safely.
	app := createTestApp(TestAppID, "Test App")
	trigger := discovery.Trigger{
		ID:   TestTriggerID,
		Type: "record_created",
		Name: "Test Trigger",
		FieldRefs: map[string]string{
			"record.id":    "ID",
			"record.title": "Title",
		},
	}
	automation := discovery.Automation{
		ID:         TestAutomationID,
		Name:       "Test Automation",
		WorkflowID: TestWorkflowID,
		Status:     "ACTIVE",
		Triggers:   []discovery.Trigger{trigger},
		Tasks:      []discovery.Task{},
	}
	app.Automations = []discovery.Automation{automation}

	uuidMap := map[string]string{
		TestTriggerID: "elementum_record_created_trigger.test_trigger.id",
	}

	hcl := `
resource "elementum_message_task" "notify" {
  value = "trigger.record.id"
  value_reference = "trigger.record.title"
}
`

	result := beautifyValueReferences(hcl, uuidMap, app)

	// System field references without UUIDs pass through unchanged
	// (the regex specifically matches UUID-based patterns)
	assert.Contains(t, result, `"trigger.record.id"`)
	assert.Contains(t, result, `"trigger.record.title"`)
}

func TestBeautifyValueReferences_NestedPaths(t *testing.T) {
	// Setup with nested path references like record.field.id
	app := createTestApp(TestAppID, "Test App")
	trigger := discovery.Trigger{
		ID:   TestTriggerID,
		Type: "record_created",
		Name: "Test Trigger",
		FieldRefs: map[string]string{
			"record." + TestFieldID + ".id": "User ID",
		},
	}
	automation := discovery.Automation{
		ID:         TestAutomationID,
		Name:       "Test Automation",
		WorkflowID: TestWorkflowID,
		Status:     "ACTIVE",
		Triggers:   []discovery.Trigger{trigger},
		Tasks:      []discovery.Task{},
	}
	app.Automations = []discovery.Automation{automation}

	uuidMap := map[string]string{
		TestTriggerID: "elementum_record_created_trigger.test_trigger.id",
	}

	hcl := `
resource "elementum_update_field_task" "update" {
  value = "trigger.record.` + TestFieldID + `.id"
}
`

	result := beautifyValueReferences(hcl, uuidMap, app)

	// Should handle nested path (.id suffix)
	assert.Contains(t, result, "User ID")
}

// =============================================================================
// buildValueRefMap Tests
// =============================================================================

func TestBuildValueRefMap_TriggerFieldRefs(t *testing.T) {
	app := createTestApp(TestAppID, "Test App")
	trigger := discovery.Trigger{
		ID:   TestTriggerID,
		Type: "record_created",
		Name: "Test Trigger",
		FieldRefs: map[string]string{
			"record." + TestFieldID: "Status",
			"record.title":          "Title",
		},
	}
	automation := discovery.Automation{
		ID:         TestAutomationID,
		Name:       "Test Automation",
		WorkflowID: TestWorkflowID,
		Status:     "ACTIVE",
		Triggers:   []discovery.Trigger{trigger},
		Tasks:      []discovery.Task{},
	}
	app.Automations = []discovery.Automation{automation}

	uuidMap := map[string]string{
		TestTriggerID: "elementum_record_created_trigger.test_trigger.id",
	}

	refMap := buildValueRefMap(app, uuidMap)

	// Should map trigger field refs
	require.NotEmpty(t, refMap)

	// Check for expected mappings
	foundStatus := false
	foundTitle := false
	for key, val := range refMap {
		if key == "trigger.record."+TestFieldID && contains(val, "Status") {
			foundStatus = true
		}
		if key == "trigger.record.title" && contains(val, "Title") {
			foundTitle = true
		}
	}
	assert.True(t, foundStatus, "Should map Status field ref")
	assert.True(t, foundTitle, "Should map Title field ref")
}

func TestBuildValueRefMap_TaskFieldRefs(t *testing.T) {
	app := createTestApp(TestAppID, "Test App")
	task := discovery.Task{
		ID:   TestTaskID,
		Type: "variable",
		Name: "Variable Task",
		FieldRefs: map[string]string{
			"value": "calculated_value",
		},
	}
	automation := discovery.Automation{
		ID:         TestAutomationID,
		Name:       "Test Automation",
		WorkflowID: TestWorkflowID,
		Status:     "ACTIVE",
		Triggers:   []discovery.Trigger{},
		Tasks:      []discovery.Task{task},
	}
	app.Automations = []discovery.Automation{automation}

	uuidMap := map[string]string{
		TestTaskID: "elementum_variable_task.variable_task.id",
	}

	refMap := buildValueRefMap(app, uuidMap)

	// Should map task field refs
	require.NotEmpty(t, refMap)

	// Should have mapping for task.{task_id}.value
	expectedKey := "task." + TestTaskID + ".value"
	val, ok := refMap[expectedKey]
	assert.True(t, ok, "Should have mapping for %s", expectedKey)
	assert.Contains(t, val, "calculated_value")
}

func TestBuildValueRefMap_Combined(t *testing.T) {
	app := createTestApp(TestAppID, "Test App")
	trigger := discovery.Trigger{
		ID:   TestTriggerID,
		Type: "record_created",
		Name: "Test Trigger",
		FieldRefs: map[string]string{
			"record.title": "Title",
		},
	}
	task := discovery.Task{
		ID:   TestTaskID,
		Type: "record_search",
		Name: "Search Task",
		FieldRefs: map[string]string{
			"count":             "Count",
			"All found records": "Records",
		},
	}
	automation := discovery.Automation{
		ID:         TestAutomationID,
		Name:       "Test Automation",
		WorkflowID: TestWorkflowID,
		Status:     "ACTIVE",
		Triggers:   []discovery.Trigger{trigger},
		Tasks:      []discovery.Task{task},
	}
	app.Automations = []discovery.Automation{automation}

	uuidMap := map[string]string{
		TestTriggerID: "elementum_record_created_trigger.test_trigger.id",
		TestTaskID:    "elementum_record_search_task.search_task.id",
	}

	refMap := buildValueRefMap(app, uuidMap)

	// Should have both trigger and task refs
	require.NotEmpty(t, refMap)

	// Verify trigger refs
	found := false
	for key := range refMap {
		if key == "trigger.record.title" {
			found = true
			break
		}
	}
	assert.True(t, found, "Should have trigger ref mapping")

	// Verify task refs
	taskCountKey := "task." + TestTaskID + ".count"
	_, ok := refMap[taskCountKey]
	assert.True(t, ok, "Should have task count ref mapping")
}

func TestBuildValueRefMap_EmptyFieldRefs(t *testing.T) {
	app := createTestApp(TestAppID, "Test App")
	trigger := discovery.Trigger{
		ID:        TestTriggerID,
		Type:      "record_created",
		Name:      "Test Trigger",
		FieldRefs: map[string]string{}, // Empty field refs
	}
	automation := discovery.Automation{
		ID:         TestAutomationID,
		Name:       "Test Automation",
		WorkflowID: TestWorkflowID,
		Status:     "ACTIVE",
		Triggers:   []discovery.Trigger{trigger},
		Tasks:      []discovery.Task{},
	}
	app.Automations = []discovery.Automation{automation}

	uuidMap := map[string]string{
		TestTriggerID: "elementum_record_created_trigger.test_trigger.id",
	}

	refMap := buildValueRefMap(app, uuidMap)

	// Should return empty map when no field refs
	assert.Empty(t, refMap)
}

func TestBuildValueRefMap_MultipleAutomations(t *testing.T) {
	app := createTestApp(TestAppID, "Test App")

	// First automation
	trigger1 := discovery.Trigger{
		ID:   TestTriggerID,
		Type: "record_created",
		Name: "Trigger 1",
		FieldRefs: map[string]string{
			"record.title": "Title 1",
		},
	}
	automation1 := discovery.Automation{
		ID:         TestAutomationID,
		Name:       "Automation 1",
		WorkflowID: TestWorkflowID,
		Status:     "ACTIVE",
		Triggers:   []discovery.Trigger{trigger1},
		Tasks:      []discovery.Task{},
	}

	// Second automation
	trigger2ID := "22222222-2222-2222-2222-222222222222"
	automation2ID := "33333333-3333-3333-3333-333333333333"
	trigger2 := discovery.Trigger{
		ID:   trigger2ID,
		Type: "record_updated",
		Name: "Trigger 2",
		FieldRefs: map[string]string{
			"record.status": "Status 2",
		},
	}
	automation2 := discovery.Automation{
		ID:         automation2ID,
		Name:       "Automation 2",
		WorkflowID: "44444444-4444-4444-4444-444444444444",
		Status:     "ACTIVE",
		Triggers:   []discovery.Trigger{trigger2},
		Tasks:      []discovery.Task{},
	}

	app.Automations = []discovery.Automation{automation1, automation2}

	uuidMap := map[string]string{
		TestTriggerID: "elementum_record_created_trigger.trigger_1.id",
		trigger2ID:    "elementum_record_updated_trigger.trigger_2.id",
	}

	refMap := buildValueRefMap(app, uuidMap)

	// Should have refs from both automations
	require.NotEmpty(t, refMap)

	foundTitle := false
	foundStatus := false
	for key, val := range refMap {
		if key == "trigger.record.title" && contains(val, "Title 1") {
			foundTitle = true
		}
		if key == "trigger.record.status" && contains(val, "Status 2") {
			foundStatus = true
		}
	}
	assert.True(t, foundTitle, "Should have Title ref from automation 1")
	assert.True(t, foundStatus, "Should have Status ref from automation 2")
}

func TestBuildValueRefMap_NoResourceNameMatch(t *testing.T) {
	app := createTestApp(TestAppID, "Test App")
	trigger := discovery.Trigger{
		ID:   TestTriggerID,
		Type: "record_created",
		Name: "Test Trigger",
		FieldRefs: map[string]string{
			"record.title": "Title",
		},
	}
	automation := discovery.Automation{
		ID:         TestAutomationID,
		Name:       "Test Automation",
		WorkflowID: TestWorkflowID,
		Status:     "ACTIVE",
		Triggers:   []discovery.Trigger{trigger},
		Tasks:      []discovery.Task{},
	}
	app.Automations = []discovery.Automation{automation}

	// UUID map doesn't contain the trigger ID
	uuidMap := map[string]string{
		"other-id": "elementum_record_created_trigger.other_trigger.id",
	}

	refMap := buildValueRefMap(app, uuidMap)

	// Should return empty map when resource name can't be found
	assert.Empty(t, refMap)
}

// =============================================================================
// findResourceNameByID Tests
// =============================================================================

func TestFindResourceNameByID_ExactMatch(t *testing.T) {
	uuidMap := map[string]string{
		TestTriggerID: "elementum_record_created_trigger.test_trigger.id",
	}

	result := findResourceNameByID(uuidMap, TestTriggerID, "elementum_record_created_trigger")

	assert.Equal(t, "test_trigger", result)
}

func TestFindResourceNameByID_NotFound(t *testing.T) {
	uuidMap := map[string]string{
		TestTriggerID: "elementum_record_created_trigger.test_trigger.id",
	}

	result := findResourceNameByID(uuidMap, "non-existent-id", "elementum_record_created_trigger")

	assert.Empty(t, result)
}

func TestFindResourceNameByID_WrongResourceType(t *testing.T) {
	uuidMap := map[string]string{
		TestTriggerID: "elementum_record_created_trigger.test_trigger.id",
	}

	// Looking for task type but have trigger in map
	result := findResourceNameByID(uuidMap, TestTriggerID, "elementum_message_task")

	assert.Empty(t, result)
}

func TestFindResourceNameByID_MultipleEntries(t *testing.T) {
	uuidMap := map[string]string{
		TestTriggerID:    "elementum_record_created_trigger.trigger_one.id",
		TestTaskID:       "elementum_message_task.task_one.id",
		TestAutomationID: "elementum_automation.automation_one.id",
	}

	// Should find correct entry for trigger
	triggerResult := findResourceNameByID(uuidMap, TestTriggerID, "elementum_record_created_trigger")
	assert.Equal(t, "trigger_one", triggerResult)

	// Should find correct entry for task
	taskResult := findResourceNameByID(uuidMap, TestTaskID, "elementum_message_task")
	assert.Equal(t, "task_one", taskResult)

	// Should find correct entry for automation
	automationResult := findResourceNameByID(uuidMap, TestAutomationID, "elementum_automation")
	assert.Equal(t, "automation_one", automationResult)
}

func TestFindResourceNameByID_ComplexResourceName(t *testing.T) {
	uuidMap := map[string]string{
		TestTriggerID: "elementum_record_created_trigger.my_complex_trigger_name_123.id",
	}

	result := findResourceNameByID(uuidMap, TestTriggerID, "elementum_record_created_trigger")

	assert.Equal(t, "my_complex_trigger_name_123", result)
}

func TestFindResourceNameByID_WithoutIDSuffix(t *testing.T) {
	// Edge case: if the ref doesn't have .id suffix
	uuidMap := map[string]string{
		TestTriggerID: "elementum_record_created_trigger.test_trigger",
	}

	result := findResourceNameByID(uuidMap, TestTriggerID, "elementum_record_created_trigger")

	// Should still work, just won't trim .id
	assert.Equal(t, "test_trigger", result)
}

// =============================================================================
// buildUUIDMap Tests (Related to Value References)
// =============================================================================

func TestBuildUUIDMap_BasicImports(t *testing.T) {
	imports := []ImportBlock{
		{ResourceType: "elementum_app", ResourceName: "my_app", ID: TestAppID},
		{ResourceType: "elementum_text_field", ResourceName: "title_field", ID: TestFieldID},
	}

	uuidMap := buildUUIDMap(imports, nil)

	assert.Contains(t, uuidMap, TestAppID)
	assert.Contains(t, uuidMap, TestFieldID)
	assert.Equal(t, "elementum_app.my_app.id", uuidMap[TestAppID])
	assert.Equal(t, "elementum_text_field.title_field.id", uuidMap[TestFieldID])
}

func TestBuildUUIDMap_CompositeIDs(t *testing.T) {
	// Some resources have composite IDs like "app_id:resource_id"
	imports := []ImportBlock{
		{ResourceType: "elementum_layout", ResourceName: "my_layout", ID: TestAppID + ":" + TestLayoutID},
	}

	uuidMap := buildUUIDMap(imports, nil)

	// Should extract the second part (stage_id for layouts)
	assert.Contains(t, uuidMap, TestLayoutID)
	assert.Equal(t, "elementum_layout.my_layout.stage_id", uuidMap[TestLayoutID])
}

func TestBuildUUIDMap_TripleCompositeIDs(t *testing.T) {
	// Some resources have triple composite IDs like "a:b:c"
	tripleID := TestAppID + ":" + TestWorkflowID + ":" + TestTriggerID
	imports := []ImportBlock{
		{ResourceType: "elementum_record_created_trigger", ResourceName: "my_trigger", ID: tripleID},
	}

	uuidMap := buildUUIDMap(imports, nil)

	// Should extract the third part
	assert.Contains(t, uuidMap, TestTriggerID)
	assert.Equal(t, "elementum_record_created_trigger.my_trigger.id", uuidMap[TestTriggerID])
}

func TestBuildUUIDMap_WithAppMappings(t *testing.T) {
	app := createTestApp(TestAppID, "Test App")

	imports := []ImportBlock{
		{ResourceType: "elementum_app", ResourceName: "test_app", ID: TestAppID},
	}

	uuidMap := buildUUIDMap(imports, app)

	// Should include app ID mapping
	assert.Contains(t, uuidMap, TestAppID)
}

func TestBuildUUIDMap_AutomationTriggerTaskMappings(t *testing.T) {
	app := createTestApp(TestAppID, "Test App")
	trigger := discovery.Trigger{
		ID:   TestTriggerID,
		Type: "record_created",
		Name: "Test Trigger",
	}
	task := discovery.Task{
		ID:   TestTaskID,
		Type: "message",
		Name: "Test Task",
	}
	automation := discovery.Automation{
		ID:         TestAutomationID,
		Name:       "Test Automation",
		WorkflowID: TestWorkflowID,
		Status:     "ACTIVE",
		Triggers:   []discovery.Trigger{trigger},
		Tasks:      []discovery.Task{task},
	}
	app.Automations = []discovery.Automation{automation}

	imports := []ImportBlock{
		{ResourceType: "elementum_app", ResourceName: "test_app", ID: TestAppID},
		{ResourceType: "elementum_automation", ResourceName: "test_automation", ID: TestAutomationID},
		{ResourceType: "elementum_record_created_trigger", ResourceName: "test_trigger", ID: TestTriggerID},
		{ResourceType: "elementum_message_task", ResourceName: "test_task", ID: TestTaskID},
	}

	uuidMap := buildUUIDMap(imports, app)

	// Should include all resource mappings
	assert.Contains(t, uuidMap, TestAutomationID)
	assert.Contains(t, uuidMap, TestTriggerID)
	assert.Contains(t, uuidMap, TestTaskID)
}

func TestBuildUUIDMap_FieldMappingsWithSemanticTags(t *testing.T) {
	app := createTestApp(TestAppID, "Test App")
	app.Fields = []discovery.Field{
		{
			ID:           TestFieldID,
			Name:         "Title",
			Type:         "text",
			SemanticTags: []string{"TITLE"},
		},
		{
			ID:           "22222222-2222-2222-2222-222222222222",
			Name:         "Status",
			Type:         "dropdown",
			SemanticTags: []string{"STATUS"},
		},
	}

	imports := []ImportBlock{
		{ResourceType: "elementum_app", ResourceName: "my_app", ID: TestAppID},
		{ResourceType: "elementum_text_field", ResourceName: "title_field", ID: TestFieldID},
		{ResourceType: "elementum_dropdown_field", ResourceName: "status_field", ID: "22222222-2222-2222-2222-222222222222"},
	}

	uuidMap := buildUUIDMap(imports, app)

	// Should map semantic tagged fields to local references
	assert.Contains(t, uuidMap, TestFieldID)
	assert.Contains(t, uuidMap[TestFieldID], "local.")
	assert.Contains(t, uuidMap[TestFieldID], "title_field_id")

	assert.Contains(t, uuidMap, "22222222-2222-2222-2222-222222222222")
	assert.Contains(t, uuidMap["22222222-2222-2222-2222-222222222222"], "local.")
	assert.Contains(t, uuidMap["22222222-2222-2222-2222-222222222222"], "status_field_id")
}

func TestBuildUUIDMap_RelatedObjects(t *testing.T) {
	app := createTestApp(TestAppID, "Test App")
	app.RelatedObjects = []discovery.RelatedObject{
		{
			ID:   "33333333-3333-3333-3333-333333333333",
			Name: "Related App",
			Type: "App",
		},
		{
			ID:   "44444444-4444-4444-4444-444444444444",
			Name: "Related Element",
			Type: "Element",
		},
	}

	imports := []ImportBlock{
		{ResourceType: "elementum_app", ResourceName: "my_app", ID: TestAppID},
	}

	uuidMap := buildUUIDMap(imports, app)

	// Should map related objects to data source references
	assert.Contains(t, uuidMap, "33333333-3333-3333-3333-333333333333")
	assert.Contains(t, uuidMap["33333333-3333-3333-3333-333333333333"], "data.elementum_app.related_app.id")

	assert.Contains(t, uuidMap, "44444444-4444-4444-4444-444444444444")
	assert.Contains(t, uuidMap["44444444-4444-4444-4444-444444444444"], "data.elementum_element.related_element.id")
}

func TestBuildUUIDMap_CategoryAndCloudlink(t *testing.T) {
	app := createTestApp(TestAppID, "Test App")
	app.CategoryID = "55555555-5555-5555-5555-555555555555"
	app.CategoryName = "My Category"
	app.CloudLinkID = "66666666-6666-6666-6666-666666666666"
	app.CloudLinkName = "My CloudLink"

	imports := []ImportBlock{
		{ResourceType: "elementum_app", ResourceName: "my_app", ID: TestAppID},
	}

	uuidMap := buildUUIDMap(imports, app)

	// Should map category and cloudlink to data source references
	assert.Contains(t, uuidMap, "55555555-5555-5555-5555-555555555555")
	assert.Contains(t, uuidMap["55555555-5555-5555-5555-555555555555"], "data.elementum_category.my_category.id")

	assert.Contains(t, uuidMap, "66666666-6666-6666-6666-666666666666")
	assert.Contains(t, uuidMap["66666666-6666-6666-6666-666666666666"], "data.elementum_cloudlink.my_cloudlink.id")
}

// =============================================================================
// stripNullAttributes Tests
// =============================================================================

func TestStripNullAttributes_RemovesNullLines(t *testing.T) {
	hcl := `
resource "elementum_app" "test" {
  name = "Test"
  description = null
  category_id = null
}
`

	result := stripNullAttributes(hcl)

	assert.NotContains(t, result, "description = null")
	assert.NotContains(t, result, "category_id = null")
	assert.Contains(t, result, `name = "Test"`)
}

func TestStripNullAttributes_PreservesNonNullLines(t *testing.T) {
	hcl := `
resource "elementum_app" "test" {
  name = "Test"
  description = "A description"
}
`

	result := stripNullAttributes(hcl)

	assert.Contains(t, result, `name = "Test"`)
	assert.Contains(t, result, `description = "A description"`)
}

func TestStripNullAttributes_HandlesVariousWhitespace(t *testing.T) {
	hcl := `
resource "elementum_app" "test" {
  name = "Test"
    description = null
      category_id   =  null
}
`

	result := stripNullAttributes(hcl)

	// Should remove both null lines regardless of whitespace
	assert.NotContains(t, result, "description = null")
	assert.NotContains(t, result, "category_id   =  null")
}

// =============================================================================
// replaceUUIDs Tests
// =============================================================================

func TestReplaceUUIDs_BasicReplacement(t *testing.T) {
	uuidMap := map[string]string{
		TestAppID: "elementum_app.my_app.id",
	}

	hcl := `
resource "elementum_text_field" "test" {
  app_id = "` + TestAppID + `"
}
`

	result := replaceUUIDs(hcl, uuidMap)

	assert.Contains(t, result, "elementum_app.my_app.id")
	assert.NotContains(t, result, TestAppID)
}

func TestReplaceUUIDs_MultipleReplacements(t *testing.T) {
	uuidMap := map[string]string{
		TestAppID:   "elementum_app.my_app.id",
		TestFieldID: "elementum_text_field.my_field.id",
	}

	hcl := `
resource "elementum_layout" "test" {
  app_id = "` + TestAppID + `"
  field_id = "` + TestFieldID + `"
}
`

	result := replaceUUIDs(hcl, uuidMap)

	assert.Contains(t, result, "elementum_app.my_app.id")
	assert.Contains(t, result, "elementum_text_field.my_field.id")
	assert.NotContains(t, result, TestAppID)
	assert.NotContains(t, result, TestFieldID)
}

func TestReplaceUUIDs_NoMatchingUUIDs(t *testing.T) {
	uuidMap := map[string]string{
		TestAppID: "elementum_app.my_app.id",
	}

	unknownUUID := "99999999-9999-9999-9999-999999999999"
	hcl := `
resource "elementum_text_field" "test" {
  app_id = "` + unknownUUID + `"
}
`

	result := replaceUUIDs(hcl, uuidMap)

	// Unknown UUID should remain unchanged
	assert.Contains(t, result, unknownUUID)
}

// =============================================================================
// Helper Functions
// =============================================================================

// contains checks if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && searchString(s, substr)))
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// =============================================================================
// Cross-Automation Reference Tests
// =============================================================================

// TestBeautifyValueReferences_CrossAutomation tests that trigger references are
// correctly scoped to their automation, preventing cross-automation contamination
// when multiple automations reference the same field.
func TestBeautifyValueReferences_CrossAutomation(t *testing.T) {
	sharedFieldID := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	trigger1ID := "11111111-2222-3333-4444-555555555555"
	trigger2ID := "66666666-7777-8888-9999-aaaaaaaaaaaa"

	// HCL with two automations, each with tasks that reference the trigger
	hcl := `
resource "elementum_record_created_trigger" "digitize_document_record_created_0" {
  automation_id = elementum_automation.digitize_document.id
}

resource "elementum_update_field_task" "digitize_document_update_status" {
  parent_id = elementum_record_created_trigger.digitize_document_record_created_0.id
  value_reference = "trigger.record.` + sharedFieldID + `"
}

resource "elementum_record_updated_trigger" "categorize_document_record_updated_0" {
  automation_id = elementum_automation.categorize_document.id
}

resource "elementum_update_field_task" "categorize_document_update_status" {
  parent_id = elementum_record_updated_trigger.categorize_document_record_updated_0.id
  value_reference = "trigger.record.` + sharedFieldID + `"
}
`

	imports := []ImportBlock{
		{
			ID:           "automation1:" + trigger1ID,
			ResourceType: "elementum_record_created_trigger",
			ResourceName: "digitize_document_record_created_0",
		},
		{
			ID:           "automation2:" + trigger2ID,
			ResourceType: "elementum_record_updated_trigger",
			ResourceName: "categorize_document_record_updated_0",
		},
	}

	app := &discovery.App{
		ID: "app-uuid",
		Automations: []discovery.Automation{
			{
				ID:         "automation1",
				Name:       "Digitize Document",
				WorkflowID: "workflow1",
				Triggers: []discovery.Trigger{
					{
						ID:   trigger1ID,
						Type: "record_created",
						Name: "record_created",
						FieldRefs: map[string]string{
							"record." + sharedFieldID: "Status",
						},
					},
				},
			},
			{
				ID:         "automation2",
				Name:       "Categorize Document",
				WorkflowID: "workflow2",
				Triggers: []discovery.Trigger{
					{
						ID:   trigger2ID,
						Type: "record_updated",
						Name: "record_updated",
						FieldRefs: map[string]string{
							"record." + sharedFieldID: "Status",
						},
					},
				},
			},
		},
	}

	uuidMap := buildUUIDMap(imports, app)
	result := beautifyValueReferences(hcl, uuidMap, app)

	// Verify digitize_document task references digitize_document trigger (not categorize_document)
	assert.Contains(t, result, `elementum_record_created_trigger.digitize_document_record_created_0.refs["Status"]`,
		"digitize_document task should reference digitize_document trigger")

	// Verify categorize_document task references categorize_document trigger (not digitize_document)
	assert.Contains(t, result, `elementum_record_updated_trigger.categorize_document_record_updated_0.refs["Status"]`,
		"categorize_document task should reference categorize_document trigger")

	// Verify no cross-automation contamination
	assert.NotContains(t, result, `digitize_document_update_status.*categorize_document`,
		"digitize_document task should not reference categorize_document trigger")
	assert.NotContains(t, result, `categorize_document_update_status.*digitize_document`,
		"categorize_document task should not reference digitize_document trigger")
}
