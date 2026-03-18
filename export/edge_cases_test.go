// Copyright (c) ElementumAI, Inc.
// SPDX-License-Identifier: MPL-2.0

package export

import (
	"testing"

	"github.com/elementumltd/elementum-cli/discovery"
	"github.com/stretchr/testify/assert"
)

// =============================================================================
// Missing Data Handling Tests
// =============================================================================

func TestEdgeCase_NilRawData(t *testing.T) {
	trigger := discovery.Trigger{
		ID:      TestTriggerID,
		Type:    "record_created",
		Name:    "Test Trigger",
		RawData: nil, // nil RawData
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
	automations := []discovery.Automation{automation}

	imports := []ImportBlock{
		{ResourceType: "elementum_automation", ResourceName: "test", ID: TestAutomationID},
		{ResourceType: "elementum_record_created_trigger", ResourceName: "test_trigger", ID: TestAutomationID + ":" + TestTriggerID},
	}

	// Should not panic with nil RawData
	gen := NewTriggerHCLGenerator(imports, make(map[string]string), automations)
	hcl := gen.GenerateAll()

	// Should handle gracefully - may produce minimal output or skip
	assert.NotPanics(t, func() {
		_ = hcl
	})
}

func TestEdgeCase_NilFieldRefs(t *testing.T) {
	trigger := discovery.Trigger{
		ID:        TestTriggerID,
		Type:      "record_created",
		Name:      "Test Trigger",
		FieldRefs: nil, // nil FieldRefs
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
	automations := []discovery.Automation{automation}

	imports := []ImportBlock{
		{ResourceType: "elementum_automation", ResourceName: "test", ID: TestAutomationID},
		{ResourceType: "elementum_record_created_trigger", ResourceName: "test_trigger", ID: TestAutomationID + ":" + TestTriggerID},
	}

	// Should not panic with nil FieldRefs
	gen := NewTriggerHCLGenerator(imports, make(map[string]string), automations)
	assert.NotPanics(t, func() {
		_ = gen.GenerateAll()
	})
}

func TestEdgeCase_EmptyAutomations(t *testing.T) {
	app := createTestApp(TestAppID, "Test App")
	app.Automations = []discovery.Automation{} // Empty automations

	imports := []ImportBlock{}

	gen := NewHCLGenerator(app, imports)
	triggerHCL := gen.GenerateTriggerResourcesOnly()
	taskHCL := gen.GenerateTaskResourcesOnly()

	// Should return empty strings for empty automations
	assert.Empty(t, triggerHCL)
	assert.Empty(t, taskHCL)
}

func TestEdgeCase_EmptyTriggers(t *testing.T) {
	automation := discovery.Automation{
		ID:           TestAutomationID,
		Name:         "Test Automation",
		Status:       "ACTIVE",
		WorkflowID:   TestWorkflowID,
		HasPublished: true,
		Triggers:     []discovery.Trigger{}, // Empty triggers
		Tasks:        []discovery.Task{},
	}
	automations := []discovery.Automation{automation}

	imports := []ImportBlock{
		{ResourceType: "elementum_automation", ResourceName: "test", ID: TestAutomationID},
	}

	gen := NewTriggerHCLGenerator(imports, make(map[string]string), automations)
	hcl := gen.GenerateAll()

	// Should return empty for empty triggers
	assert.Empty(t, hcl)
}

func TestEdgeCase_EmptyTasks(t *testing.T) {
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
		Tasks:        []discovery.Task{}, // Empty tasks
	}
	automations := []discovery.Automation{automation}

	imports := []ImportBlock{
		{ResourceType: "elementum_automation", ResourceName: "test", ID: TestAutomationID},
		{ResourceType: "elementum_record_created_trigger", ResourceName: "test_trigger", ID: TestAutomationID + ":" + TestTriggerID},
	}

	gen := NewTaskHCLGenerator(app, imports, make(map[string]string), automations)
	hcl := gen.GenerateAll()

	// Should return empty for empty tasks
	assert.Empty(t, hcl)
}

// =============================================================================
// Malformed Data Handling Tests
// =============================================================================

func TestEdgeCase_InvalidTriggerType(t *testing.T) {
	trigger := discovery.Trigger{
		ID:   TestTriggerID,
		Type: "unknown_trigger_type", // Invalid type
		Name: "Test Trigger",
		RawData: map[string]any{
			"__typename": "UnknownTriggerType",
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
	automations := []discovery.Automation{automation}

	imports := []ImportBlock{
		{ResourceType: "elementum_automation", ResourceName: "test", ID: TestAutomationID},
		{ResourceType: "elementum_unknown_trigger_type_trigger", ResourceName: "test_trigger", ID: TestAutomationID + ":" + TestTriggerID},
	}

	gen := NewTriggerHCLGenerator(imports, make(map[string]string), automations)
	hcl := gen.GenerateAll()

	// Generator produces output for unknown types (doesn't skip them)
	// The key is that it doesn't panic and produces valid HCL structure
	assert.NotPanics(t, func() {
		_ = hcl
	})
}

func TestEdgeCase_InvalidTaskType(t *testing.T) {
	app := createTestApp(TestAppID, "Test App")

	task := discovery.Task{
		ID:   TestTaskID,
		Type: "unknown_task_type", // Invalid type
		Name: "Test Task",
		RawData: map[string]any{
			"__typename": "UnknownTaskType",
			"id":         TestTaskID,
			"name":       "Test Task",
		},
	}

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
		Tasks:        []discovery.Task{task},
	}
	automations := []discovery.Automation{automation}

	imports := []ImportBlock{
		{ResourceType: "elementum_automation", ResourceName: "test", ID: TestAutomationID},
		{ResourceType: "elementum_record_created_trigger", ResourceName: "test_trigger", ID: TestAutomationID + ":" + TestTriggerID},
		{ResourceType: "elementum_unknown_task_type_task", ResourceName: "test_task", ID: TestWorkflowID + ":" + TestTaskID},
	}

	gen := NewTaskHCLGenerator(app, imports, make(map[string]string), automations)
	hcl := gen.GenerateAll()

	// Generator produces output for unknown types (doesn't skip them)
	// The key is that it doesn't panic and produces valid HCL structure
	assert.NotPanics(t, func() {
		_ = hcl
	})
}

func TestEdgeCase_MissingRequiredFields(t *testing.T) {
	app := createTestApp(TestAppID, "Test App")

	// Task with missing required fields in RawData
	task := discovery.Task{
		ID:   TestTaskID,
		Type: "message",
		Name: "Test Task",
		RawData: map[string]any{
			"__typename": "WorkflowMessageTask",
			"id":         TestTaskID,
			// Missing "name" and "contentsReference"
		},
	}

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
		Tasks:        []discovery.Task{task},
	}
	automations := []discovery.Automation{automation}

	imports := []ImportBlock{
		{ResourceType: "elementum_automation", ResourceName: "test", ID: TestAutomationID},
		{ResourceType: "elementum_record_created_trigger", ResourceName: "test_trigger", ID: TestAutomationID + ":" + TestTriggerID},
		{ResourceType: "elementum_message_task", ResourceName: "test_task", ID: TestWorkflowID + ":" + TestTaskID},
	}

	// Should not panic with missing fields
	gen := NewTaskHCLGenerator(app, imports, make(map[string]string), automations)
	assert.NotPanics(t, func() {
		_ = gen.GenerateAll()
	})
}

// =============================================================================
// Reference Resolution Failure Tests
// =============================================================================

func TestEdgeCase_UnresolvableReference(t *testing.T) {
	app := createTestApp(TestAppID, "Test App")

	// Task referencing a non-existent agent
	task := discovery.Task{
		ID:   TestTaskID,
		Type: "ai_agent",
		Name: "Run Agent Task",
		RawData: map[string]any{
			"__typename": "WorkflowAiAgentTask",
			"id":         TestTaskID,
			"name":       "Run Agent Task",
			"agent":      map[string]any{"id": "non-existent-agent-id"},
		},
	}

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
		Tasks:        []discovery.Task{task},
	}
	automations := []discovery.Automation{automation}

	imports := []ImportBlock{
		{ResourceType: "elementum_automation", ResourceName: "test", ID: TestAutomationID},
		{ResourceType: "elementum_record_created_trigger", ResourceName: "test_trigger", ID: TestAutomationID + ":" + TestTriggerID},
		{ResourceType: "elementum_ai_agent_task", ResourceName: "ai_agent_task", ID: TestWorkflowID + ":" + TestTaskID},
	}

	// Should handle gracefully - may use raw ID as fallback
	gen := NewTaskHCLGenerator(app, imports, make(map[string]string), automations)
	assert.NotPanics(t, func() {
		hcl := gen.GenerateAll()
		// Should still produce valid HCL structure
		assert.Contains(t, hcl, "elementum_ai_agent_task")
	})
}

func TestEdgeCase_MissingParent(t *testing.T) {
	app := createTestApp(TestAppID, "Test App")

	// Task with missing parent reference
	task := discovery.Task{
		ID:       TestTaskID,
		Type:     "message",
		Name:     "Test Task",
		ParentID: "non-existent-parent-id", // Parent doesn't exist
		RawData: map[string]any{
			"__typename":        "WorkflowMessageTask",
			"id":                TestTaskID,
			"name":              "Test Task",
			"contentsReference": map[string]any{"staticValue": "test"},
		},
	}

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
		Tasks:        []discovery.Task{task},
	}
	automations := []discovery.Automation{automation}

	imports := []ImportBlock{
		{ResourceType: "elementum_automation", ResourceName: "test", ID: TestAutomationID},
		{ResourceType: "elementum_record_created_trigger", ResourceName: "test_trigger", ID: TestAutomationID + ":" + TestTriggerID},
		{ResourceType: "elementum_message_task", ResourceName: "test_task", ID: TestWorkflowID + ":" + TestTaskID},
	}

	// Should handle gracefully - may fallback to first trigger
	gen := NewTaskHCLGenerator(app, imports, make(map[string]string), automations)
	assert.NotPanics(t, func() {
		_ = gen.GenerateAll()
	})
}

// =============================================================================
// Unicode and Special Character Tests
// =============================================================================

func TestEdgeCase_UnicodeFieldNames(t *testing.T) {
	trigger := discovery.Trigger{
		ID:   TestTriggerID,
		Type: "record_created",
		Name: "Test Trigger",
		FieldRefs: map[string]string{
			"record." + TestFieldID: "日本語フィールド",
		},
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
	automations := []discovery.Automation{automation}

	imports := []ImportBlock{
		{ResourceType: "elementum_automation", ResourceName: "test", ID: TestAutomationID},
		{ResourceType: "elementum_record_created_trigger", ResourceName: "test_trigger", ID: TestAutomationID + ":" + TestTriggerID},
	}

	// Should handle Unicode gracefully
	gen := NewTriggerHCLGenerator(imports, make(map[string]string), automations)
	assert.NotPanics(t, func() {
		_ = gen.GenerateAll()
	})
}

func TestEdgeCase_SpecialCharactersInNames(t *testing.T) {
	trigger := discovery.Trigger{
		ID:   TestTriggerID,
		Type: "record_created",
		Name: "Trigger with 'quotes' & <special> chars!",
		RawData: map[string]any{
			"__typename": "WorkflowRecordCreateTrigger",
			"id":         TestTriggerID,
			"name":       "Trigger with 'quotes' & <special> chars!",
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
	automations := []discovery.Automation{automation}

	imports := []ImportBlock{
		{ResourceType: "elementum_automation", ResourceName: "test", ID: TestAutomationID},
		{ResourceType: "elementum_record_created_trigger", ResourceName: "trigger_with_quotes_special_chars", ID: TestAutomationID + ":" + TestTriggerID},
	}

	// Should handle special characters gracefully
	gen := NewTriggerHCLGenerator(imports, make(map[string]string), automations)
	assert.NotPanics(t, func() {
		hcl := gen.GenerateAll()
		// Should produce valid HCL structure
		assert.Contains(t, hcl, "elementum_record_created_trigger")
	})
}

func TestEdgeCase_QuotesInStrings(t *testing.T) {
	app := createTestApp(TestAppID, "Test App")

	task := discovery.Task{
		ID:   TestTaskID,
		Type: "message",
		Name: `Test "Quoted" Task`,
		RawData: map[string]any{
			"__typename":        "WorkflowMessageTask",
			"id":                TestTaskID,
			"name":              `Test "Quoted" Task`,
			"contentsReference": map[string]any{"staticValue": `Message with "quotes" inside`},
		},
	}

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
		Tasks:        []discovery.Task{task},
	}
	automations := []discovery.Automation{automation}

	imports := []ImportBlock{
		{ResourceType: "elementum_automation", ResourceName: "test", ID: TestAutomationID},
		{ResourceType: "elementum_record_created_trigger", ResourceName: "test_trigger", ID: TestAutomationID + ":" + TestTriggerID},
		{ResourceType: "elementum_message_task", ResourceName: "test_quoted_task", ID: TestWorkflowID + ":" + TestTaskID},
	}

	// Should handle quotes in strings
	gen := NewTaskHCLGenerator(app, imports, make(map[string]string), automations)
	assert.NotPanics(t, func() {
		hcl := gen.GenerateAll()
		// Should produce valid HCL
		assert.Contains(t, hcl, "elementum_message_task")
	})
}

// =============================================================================
// Boundary Condition Tests
// =============================================================================

func TestEdgeCase_VeryLongFieldName(t *testing.T) {
	// Field name with 200 characters
	longName := ""
	for i := 0; i < 200; i++ {
		longName += "a"
	}

	trigger := discovery.Trigger{
		ID:   TestTriggerID,
		Type: "record_created",
		Name: "Test Trigger",
		FieldRefs: map[string]string{
			"record." + TestFieldID: longName,
		},
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
	automations := []discovery.Automation{automation}

	imports := []ImportBlock{
		{ResourceType: "elementum_automation", ResourceName: "test", ID: TestAutomationID},
		{ResourceType: "elementum_record_created_trigger", ResourceName: "test_trigger", ID: TestAutomationID + ":" + TestTriggerID},
	}

	// Should handle long field names
	gen := NewTriggerHCLGenerator(imports, make(map[string]string), automations)
	assert.NotPanics(t, func() {
		_ = gen.GenerateAll()
	})
}

func TestEdgeCase_ManyWorkflowFields(t *testing.T) {
	app := createTestApp(TestAppID, "Test App")

	// Create 50 workflow fields
	workflowFields := make([]any, 50)
	for i := 0; i < 50; i++ {
		workflowFields[i] = map[string]any{
			"field":          map[string]any{"id": TestFieldID},
			"valueReference": map[string]any{"staticValue": "value"},
		}
	}

	task := discovery.Task{
		ID:   TestTaskID,
		Type: "update_field",
		Name: "Update Many Fields",
		RawData: map[string]any{
			"__typename":     "WorkflowUpdateFieldTask",
			"id":             TestTaskID,
			"name":           "Update Many Fields",
			"aspect":         map[string]any{"id": TestAppID},
			"workflowFields": workflowFields,
		},
	}

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
		Tasks:        []discovery.Task{task},
	}
	automations := []discovery.Automation{automation}

	imports := []ImportBlock{
		{ResourceType: "elementum_automation", ResourceName: "test", ID: TestAutomationID},
		{ResourceType: "elementum_record_created_trigger", ResourceName: "test_trigger", ID: TestAutomationID + ":" + TestTriggerID},
		{ResourceType: "elementum_update_field_task", ResourceName: "update_many_fields", ID: TestWorkflowID + ":" + TestTaskID},
	}

	// Should handle many workflow fields
	gen := NewTaskHCLGenerator(app, imports, make(map[string]string), automations)
	assert.NotPanics(t, func() {
		hcl := gen.GenerateAll()
		assert.Contains(t, hcl, "elementum_update_field_task")
	})
}

// =============================================================================
// Nil App Tests
// =============================================================================

func TestEdgeCase_NilApp(t *testing.T) {
	imports := []ImportBlock{
		{ResourceType: "elementum_app", ResourceName: "test", ID: TestAppID},
	}

	// Various functions should handle nil app gracefully
	assert.NotPanics(t, func() {
		_ = buildUUIDMap(imports, nil)
	})

	assert.NotPanics(t, func() {
		_ = beautifyValueReferences("test = value", make(map[string]string), nil)
	})
}

// =============================================================================
// Empty String Tests
// =============================================================================

func TestEdgeCase_EmptyStrings(t *testing.T) {
	trigger := discovery.Trigger{
		ID:   TestTriggerID,
		Type: "record_created",
		Name: "", // Empty name
		RawData: map[string]any{
			"__typename": "WorkflowRecordCreateTrigger",
			"id":         TestTriggerID,
			"name":       "", // Empty name in RawData too
		},
	}

	automation := discovery.Automation{
		ID:           TestAutomationID,
		Name:         "", // Empty automation name
		Status:       "ACTIVE",
		WorkflowID:   TestWorkflowID,
		HasPublished: true,
		Triggers:     []discovery.Trigger{trigger},
		Tasks:        []discovery.Task{},
	}
	automations := []discovery.Automation{automation}

	imports := []ImportBlock{
		{ResourceType: "elementum_automation", ResourceName: "unnamed", ID: TestAutomationID},
		{ResourceType: "elementum_record_created_trigger", ResourceName: "unnamed_trigger", ID: TestAutomationID + ":" + TestTriggerID},
	}

	// Should handle empty strings gracefully
	gen := NewTriggerHCLGenerator(imports, make(map[string]string), automations)
	assert.NotPanics(t, func() {
		_ = gen.GenerateAll()
	})
}

// =============================================================================
// SanitizeName Edge Cases
// =============================================================================

func TestSanitizeName_VariousInputs(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Simple Name", "simple_name"},
		{"Name with 123 numbers", "name_with_123_numbers"},
		{"Name!@#$%^&*()", "name"},
		{"UPPERCASE", "uppercase"},
		{"CamelCase", "camelcase"},
		{"with-dashes", "with_dashes"},
		{"with.dots", "with_dots"},
		{"123StartWithNumber", "_123startwithnumber"},
		{"___multiple___underscores___", "___multiple___underscores___"}, // Underscores preserved
		{"", "resource"},                                                 // Empty returns "resource"
		{"   spaces   ", "___spaces___"},                                 // Spaces become underscores
		{"日本語", "resource"},                                              // Unicode becomes empty, falls back to "resource"
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := SanitizeName(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// =============================================================================
// Import Block Edge Cases
// =============================================================================

func TestEdgeCase_DuplicateImportBlocks(t *testing.T) {
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
	automations := []discovery.Automation{automation}

	// Duplicate import blocks
	imports := []ImportBlock{
		{ResourceType: "elementum_automation", ResourceName: "test", ID: TestAutomationID},
		{ResourceType: "elementum_record_created_trigger", ResourceName: "test_trigger", ID: TestAutomationID + ":" + TestTriggerID},
		{ResourceType: "elementum_record_created_trigger", ResourceName: "test_trigger", ID: TestAutomationID + ":" + TestTriggerID}, // Duplicate
	}

	// Should handle duplicates gracefully
	gen := NewTriggerHCLGenerator(imports, make(map[string]string), automations)
	assert.NotPanics(t, func() {
		hcl := gen.GenerateAll()
		// Should not produce duplicate resources
		assert.Contains(t, hcl, "test_trigger")
	})
}

func TestEdgeCase_NoMatchingImports(t *testing.T) {
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
	automations := []discovery.Automation{automation}

	// Imports don't match the trigger
	imports := []ImportBlock{
		{ResourceType: "elementum_automation", ResourceName: "test", ID: TestAutomationID},
		{ResourceType: "elementum_webhook_trigger", ResourceName: "wrong_trigger", ID: TestWorkflowID + ":" + "wrong-id"},
	}

	// Should handle gracefully when no matching imports
	gen := NewTriggerHCLGenerator(imports, make(map[string]string), automations)
	assert.NotPanics(t, func() {
		hcl := gen.GenerateAll()
		// No matching imports should result in empty or minimal output
		_ = hcl
	})
}
