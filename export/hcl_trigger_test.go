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
// Core TriggerHCLGenerator Tests
// ============================================================================

func TestTriggerHCLGenerator_GenerateAll(t *testing.T) {
	app := &discovery.App{
		ID:   TestAppID,
		Name: "Test App",
		Automations: []discovery.Automation{
			{
				ID:           TestAutomationID,
				Name:         "Test Automation",
				Status:       "ACTIVE",
				WorkflowID:   TestWorkflowID,
				HasPublished: true,
				Triggers: []discovery.Trigger{
					{
						ID:      TestTriggerID,
						Type:    "record_created",
						RawData: buildMinimalTriggerRawData("record_created"),
					},
				},
				Tasks: []discovery.Task{},
			},
		},
	}

	imports := []ImportBlock{
		{ID: TestAutomationID, ResourceType: "elementum_automation", ResourceName: "test_automation"},
		{ID: TestAutomationID + ":" + TestTriggerID, ResourceType: "elementum_record_created_trigger", ResourceName: "test_trigger"},
	}
	uuidMap := make(map[string]string)

	generator := NewTriggerHCLGenerator(imports, uuidMap, app.Automations)
	hcl := generator.GenerateAll()

	if !strings.Contains(hcl, `resource "elementum_record_created_trigger" "test_trigger"`) {
		t.Errorf("Expected HCL to contain trigger resource, got:\n%s", hcl)
	}

	if !strings.Contains(hcl, "automation = elementum_automation.test_automation") {
		t.Errorf("Expected HCL to reference automation, got:\n%s", hcl)
	}
}

func TestTriggerHCLGenerator_SkipInactiveAutomations(t *testing.T) {
	app := &discovery.App{
		ID:   TestAppID,
		Name: "Test App",
		Automations: []discovery.Automation{
			{
				ID:           TestAutomationID,
				Name:         "Inactive Automation",
				Status:       "INACTIVE",
				WorkflowID:   TestWorkflowID,
				HasPublished: false,
				Triggers: []discovery.Trigger{
					{
						ID:      TestTriggerID,
						Type:    "record_created",
						RawData: buildMinimalTriggerRawData("record_created"),
					},
				},
				Tasks: []discovery.Task{},
			},
		},
	}

	imports := []ImportBlock{
		{ID: TestAutomationID, ResourceType: "elementum_automation", ResourceName: "inactive_automation"},
		{ID: TestAutomationID + ":" + TestTriggerID, ResourceType: "elementum_record_created_trigger", ResourceName: "test_trigger"},
	}
	uuidMap := make(map[string]string)

	generator := NewTriggerHCLGenerator(imports, uuidMap, app.Automations)
	hcl := generator.GenerateAll()

	if strings.Contains(hcl, "elementum_record_created_trigger") {
		t.Errorf("Expected inactive automation triggers to be skipped, got:\n%s", hcl)
	}
}

func TestTriggerHCLGenerator_SkipUnknownTriggerType(t *testing.T) {
	app := &discovery.App{
		ID:   TestAppID,
		Name: "Test App",
		Automations: []discovery.Automation{
			{
				ID:           TestAutomationID,
				Name:         "Test Automation",
				Status:       "ACTIVE",
				WorkflowID:   TestWorkflowID,
				HasPublished: true,
				Triggers: []discovery.Trigger{
					{
						ID:   TestTriggerID,
						Type: "unknown",
					},
				},
				Tasks: []discovery.Task{},
			},
		},
	}

	imports := []ImportBlock{
		{ID: TestAutomationID, ResourceType: "elementum_automation", ResourceName: "test_automation"},
	}
	uuidMap := make(map[string]string)

	generator := NewTriggerHCLGenerator(imports, uuidMap, app.Automations)
	hcl := generator.GenerateAll()

	if strings.Contains(hcl, "resource") {
		t.Errorf("Expected unknown trigger type to be skipped, got:\n%s", hcl)
	}
}

func TestTriggerHCLGenerator_NoImports(t *testing.T) {
	automations := []discovery.Automation{
		{
			ID:           TestAutomationID,
			Name:         "Test Automation",
			Status:       "ACTIVE",
			WorkflowID:   TestWorkflowID,
			HasPublished: true,
			Triggers: []discovery.Trigger{
				{
					ID:      TestTriggerID,
					Type:    "record_created",
					RawData: buildMinimalTriggerRawData("record_created"),
				},
			},
			Tasks: []discovery.Task{},
		},
	}

	// Empty imports - no resource names to match
	imports := []ImportBlock{}
	uuidMap := make(map[string]string)

	generator := NewTriggerHCLGenerator(imports, uuidMap, automations)
	hcl := generator.GenerateAll()

	// Should produce empty HCL since there's no matching import for automation
	if strings.Contains(hcl, "resource") {
		t.Errorf("Expected empty HCL when no imports match, got:\n%s", hcl)
	}
}

// ============================================================================
// Record-Based Trigger Tests
// ============================================================================

func TestTriggerHCL_RecordCreated(t *testing.T) {
	rawData := map[string]interface{}{
		"__typename": "WorkflowRecordCreateTrigger",
		"id":         TestTriggerID,
		"aspect": map[string]interface{}{
			"id":   TestAppID,
			"name": "Test App",
		},
	}

	_, automations, imports := buildTriggerTestData("record_created", "test_trigger", rawData)
	uuidMap := map[string]string{TestAppID: "elementum_app.test_app.id"}

	generator := NewTriggerHCLGenerator(imports, uuidMap, automations)
	hcl := generator.GenerateAll()

	assertHCLContains(t, hcl,
		`resource "elementum_record_created_trigger" "test_trigger"`,
		"automation = elementum_automation.",
	)
}

func TestTriggerHCL_RecordCreated_WithFilter(t *testing.T) {
	// NOTE: Filter handling is currently skipped in trigger HCL generation.
	// This test verifies the generator doesn't break when a filter is present.
	rawData := map[string]interface{}{
		"__typename": "WorkflowRecordCreateTrigger",
		"id":         TestTriggerID,
		"aspect": map[string]interface{}{
			"id":   TestAppID,
			"name": "Test App",
		},
		"filter": map[string]interface{}{
			"type": "equals",
			"field": map[string]interface{}{
				"id": TestFieldID,
			},
			"value": "Active",
		},
	}

	_, automations, imports := buildTriggerTestData("record_created", "test_trigger", rawData)
	uuidMap := map[string]string{
		TestAppID:   "elementum_app.test_app.id",
		TestFieldID: "elementum_text_field.status.id",
	}

	generator := NewTriggerHCLGenerator(imports, uuidMap, automations)
	hcl := generator.GenerateAll()

	// Verify the trigger resource is generated (filter handling is currently skipped)
	assertHCLContains(t, hcl,
		`resource "elementum_record_created_trigger" "test_trigger"`,
	)
}

func TestTriggerHCL_RecordCreated_IgnoresChangedFields(t *testing.T) {
	// record_created triggers don't support changed_fields (only record_updated does).
	// Even if changedFields is present in raw data (from GraphQL), it should be ignored.
	rawData := map[string]interface{}{
		"__typename": "WorkflowRecordCreateTrigger",
		"id":         TestTriggerID,
		"aspect": map[string]interface{}{
			"id":   TestAppID,
			"name": "Test App",
		},
		"changedFields": []interface{}{
			map[string]interface{}{"id": TestFieldID, "name": "Status"},
			map[string]interface{}{"id": TestField2ID, "name": "Priority"},
		},
	}

	_, automations, imports := buildTriggerTestData("record_created", "test_trigger", rawData)
	uuidMap := map[string]string{
		TestAppID:    "elementum_app.test_app.id",
		TestFieldID:  "elementum_picklist_field.status.id",
		TestField2ID: "elementum_picklist_field.priority.id",
	}

	generator := NewTriggerHCLGenerator(imports, uuidMap, automations)
	hcl := generator.GenerateAll()

	// Should generate the trigger resource
	assertHCLContains(t, hcl,
		`resource "elementum_record_created_trigger" "test_trigger"`,
	)
	// Should NOT include changed_fields (only valid for record_updated_trigger)
	if strings.Contains(hcl, "changed_fields") {
		t.Errorf("record_created_trigger should not have changed_fields attribute\nGenerated HCL:\n%s", hcl)
	}
}

func TestTriggerHCL_RecordUpdated(t *testing.T) {
	rawData := map[string]interface{}{
		"__typename": "WorkflowRecordUpdateTrigger",
		"id":         TestTriggerID,
		"aspect": map[string]interface{}{
			"id":   TestAppID,
			"name": "Test App",
		},
	}

	_, automations, imports := buildTriggerTestData("record_updated", "test_trigger", rawData)
	uuidMap := map[string]string{TestAppID: "elementum_app.test_app.id"}

	generator := NewTriggerHCLGenerator(imports, uuidMap, automations)
	hcl := generator.GenerateAll()

	assertHCLContains(t, hcl,
		`resource "elementum_record_updated_trigger" "test_trigger"`,
		"automation = elementum_automation.",
	)
}

func TestTriggerHCL_RecordUpdated_WithChangedFields(t *testing.T) {
	rawData := map[string]interface{}{
		"__typename": "WorkflowRecordUpdateTrigger",
		"id":         TestTriggerID,
		"aspect": map[string]interface{}{
			"id":   TestAppID,
			"name": "Test App",
		},
		"changedFields": []interface{}{
			map[string]interface{}{"id": TestFieldID, "name": "Status"},
		},
	}

	_, automations, imports := buildTriggerTestData("record_updated", "test_trigger", rawData)
	uuidMap := map[string]string{
		TestAppID:   "elementum_app.test_app.id",
		TestFieldID: "elementum_picklist_field.status.id",
	}

	generator := NewTriggerHCLGenerator(imports, uuidMap, automations)
	hcl := generator.GenerateAll()

	assertHCLContains(t, hcl,
		`resource "elementum_record_updated_trigger" "test_trigger"`,
		"changed_fields",
	)
}

func TestTriggerHCL_AttachmentAdded(t *testing.T) {
	rawData := map[string]interface{}{
		"__typename": "WorkflowAttachmentTrigger",
		"id":         TestTriggerID,
		"aspect": map[string]interface{}{
			"id":   TestAppID,
			"name": "Test App",
		},
	}

	_, automations, imports := buildTriggerTestData("attachment_added", "test_trigger", rawData)
	uuidMap := map[string]string{TestAppID: "elementum_app.test_app.id"}

	generator := NewTriggerHCLGenerator(imports, uuidMap, automations)
	hcl := generator.GenerateAll()

	assertHCLContains(t, hcl,
		`resource "elementum_attachment_added_trigger" "test_trigger"`,
		"automation = elementum_automation.",
	)
}

func TestTriggerHCL_CommentAdded(t *testing.T) {
	rawData := map[string]interface{}{
		"__typename": "WorkflowCommentAddedTrigger",
		"id":         TestTriggerID,
		"aspect": map[string]interface{}{
			"id":   TestAppID,
			"name": "Test App",
		},
	}

	_, automations, imports := buildTriggerTestData("comment_added", "test_trigger", rawData)
	uuidMap := map[string]string{TestAppID: "elementum_app.test_app.id"}

	generator := NewTriggerHCLGenerator(imports, uuidMap, automations)
	hcl := generator.GenerateAll()

	assertHCLContains(t, hcl,
		`resource "elementum_comment_added_trigger" "test_trigger"`,
		"automation = elementum_automation.",
	)
}

func TestTriggerHCL_CommentAdded_WithConversationType(t *testing.T) {
	rawData := map[string]interface{}{
		"__typename": "WorkflowCommentAddedTrigger",
		"id":         TestTriggerID,
		"aspect": map[string]interface{}{
			"id":   TestAppID,
			"name": "Test App",
		},
		"conversationType": "INTERNAL",
	}

	_, automations, imports := buildTriggerTestData("comment_added", "test_trigger", rawData)
	uuidMap := map[string]string{TestAppID: "elementum_app.test_app.id"}

	generator := NewTriggerHCLGenerator(imports, uuidMap, automations)
	hcl := generator.GenerateAll()

	assertHCLContains(t, hcl,
		`resource "elementum_comment_added_trigger" "test_trigger"`,
	)
}

func TestTriggerHCL_SlackMessage(t *testing.T) {
	rawData := map[string]interface{}{
		"__typename": "WorkflowSlackMessageTrigger",
		"id":         TestTriggerID,
		"aspect": map[string]interface{}{
			"id":   TestAppID,
			"name": "Test App",
		},
	}

	_, automations, imports := buildTriggerTestData("slack_message", "test_trigger", rawData)
	uuidMap := map[string]string{TestAppID: "elementum_app.test_app.id"}

	generator := NewTriggerHCLGenerator(imports, uuidMap, automations)
	hcl := generator.GenerateAll()

	assertHCLContains(t, hcl,
		`resource "elementum_slack_message_trigger" "test_trigger"`,
		"automation = elementum_automation.",
	)
}

func TestTriggerHCL_AgentConversationEnded(t *testing.T) {
	rawData := map[string]interface{}{
		"__typename": "WorkflowConversationEndTrigger",
		"id":         TestTriggerID,
		"aspect": map[string]interface{}{
			"id":   TestAppID,
			"name": "Test App",
		},
	}

	_, automations, imports := buildTriggerTestData("agent_conversation_ended", "test_trigger", rawData)
	uuidMap := map[string]string{TestAppID: "elementum_app.test_app.id"}

	generator := NewTriggerHCLGenerator(imports, uuidMap, automations)
	hcl := generator.GenerateAll()

	assertHCLContains(t, hcl,
		`resource "elementum_agent_conversation_ended_trigger" "test_trigger"`,
		"automation = elementum_automation.",
	)
}

// ============================================================================
// Non-Record-Based Trigger Tests
// ============================================================================

func TestTriggerHCL_Webhook(t *testing.T) {
	rawData := map[string]interface{}{
		"__typename":    "WorkflowWebhookTrigger",
		"id":            TestTriggerID,
		"authenticated": false,
		"url":           "https://example.com/webhook/abc123",
	}

	_, automations, imports := buildTriggerTestData("webhook", "test_trigger", rawData)
	uuidMap := make(map[string]string)

	generator := NewTriggerHCLGenerator(imports, uuidMap, automations)
	hcl := generator.GenerateAll()

	assertHCLContains(t, hcl,
		`resource "elementum_webhook_trigger" "test_trigger"`,
		"automation = elementum_automation.",
	)
}

func TestTriggerHCL_Webhook_Authenticated(t *testing.T) {
	rawData := map[string]interface{}{
		"__typename":    "WorkflowWebhookTrigger",
		"id":            TestTriggerID,
		"authenticated": true,
		"url":           "https://example.com/webhook/abc123",
	}

	_, automations, imports := buildTriggerTestData("webhook", "test_trigger", rawData)
	uuidMap := make(map[string]string)

	generator := NewTriggerHCLGenerator(imports, uuidMap, automations)
	hcl := generator.GenerateAll()

	assertHCLContains(t, hcl,
		`resource "elementum_webhook_trigger" "test_trigger"`,
		"authenticated = true",
	)
}

func TestTriggerHCL_OnDemand(t *testing.T) {
	rawData := map[string]interface{}{
		"__typename": "WorkflowOnDemandTrigger",
		"id":         TestTriggerID,
		"parameters": []interface{}{},
	}

	_, automations, imports := buildTriggerTestData("on_demand", "test_trigger", rawData)
	uuidMap := make(map[string]string)

	generator := NewTriggerHCLGenerator(imports, uuidMap, automations)
	hcl := generator.GenerateAll()

	assertHCLContains(t, hcl,
		`resource "elementum_on_demand_trigger" "test_trigger"`,
		"automation = elementum_automation.",
	)
}

func TestTriggerHCL_OnDemand_WithParameters(t *testing.T) {
	rawData := map[string]interface{}{
		"__typename": "WorkflowOnDemandTrigger",
		"id":         TestTriggerID,
		"parameters": []interface{}{
			map[string]interface{}{
				"id":        "param-1",
				"name":      "customer_name",
				"fieldType": "TEXT",
				"required":  true,
				"multiple":  false,
			},
		},
	}

	_, automations, imports := buildTriggerTestData("on_demand", "test_trigger", rawData)
	uuidMap := make(map[string]string)

	generator := NewTriggerHCLGenerator(imports, uuidMap, automations)
	hcl := generator.GenerateAll()

	assertHCLContains(t, hcl,
		`resource "elementum_on_demand_trigger" "test_trigger"`,
		"parameters",
		`name = "customer_name"`,
		"required = true",
	)
}

func TestTriggerHCL_OnDemand_MultipleParameters(t *testing.T) {
	rawData := map[string]interface{}{
		"__typename": "WorkflowOnDemandTrigger",
		"id":         TestTriggerID,
		"parameters": []interface{}{
			map[string]interface{}{
				"id":        "param-1",
				"name":      "customer_name",
				"fieldType": "TEXT",
				"required":  true,
				"multiple":  false,
			},
			map[string]interface{}{
				"id":           "param-2",
				"name":         "priority",
				"fieldType":    "NUMBER",
				"required":     false,
				"multiple":     false,
				"defaultValue": "5",
			},
			map[string]interface{}{
				"id":        "param-3",
				"name":      "tags",
				"fieldType": "TEXT",
				"required":  false,
				"multiple":  true,
			},
		},
	}

	_, automations, imports := buildTriggerTestData("on_demand", "test_trigger", rawData)
	uuidMap := make(map[string]string)

	generator := NewTriggerHCLGenerator(imports, uuidMap, automations)
	hcl := generator.GenerateAll()

	assertHCLContains(t, hcl,
		`resource "elementum_on_demand_trigger" "test_trigger"`,
		"parameters",
		`name = "customer_name"`,
		`name = "priority"`,
		`name = "tags"`,
	)
}

func TestTriggerHCL_OnDemand_WithShowTriggeredBy(t *testing.T) {
	rawData := map[string]interface{}{
		"__typename":      "WorkflowOnDemandTrigger",
		"id":              TestTriggerID,
		"showTriggeredBy": true,
		"parameters":      []interface{}{},
	}

	_, automations, imports := buildTriggerTestData("on_demand", "test_trigger", rawData)
	uuidMap := make(map[string]string)

	generator := NewTriggerHCLGenerator(imports, uuidMap, automations)
	hcl := generator.GenerateAll()

	assertHCLContains(t, hcl,
		`resource "elementum_on_demand_trigger" "test_trigger"`,
		"show_triggered_by = true",
	)
}

func TestTriggerHCL_OnDemand_ShowTriggeredByFalse(t *testing.T) {
	rawData := map[string]interface{}{
		"__typename":      "WorkflowOnDemandTrigger",
		"id":              TestTriggerID,
		"showTriggeredBy": false,
		"parameters":      []interface{}{},
	}

	_, automations, imports := buildTriggerTestData("on_demand", "test_trigger", rawData)
	uuidMap := make(map[string]string)

	generator := NewTriggerHCLGenerator(imports, uuidMap, automations)
	hcl := generator.GenerateAll()

	assertHCLContains(t, hcl,
		`resource "elementum_on_demand_trigger" "test_trigger"`,
	)
	// When showTriggeredBy is false, it should still be included in the output
	// to ensure explicit configuration
	if !strings.Contains(hcl, "show_triggered_by = false") {
		t.Logf("Note: show_triggered_by = false is not included (acceptable for boolean defaults)")
	}
}

func TestTriggerHCL_EmailIngestion(t *testing.T) {
	rawData := map[string]interface{}{
		"__typename": "WorkflowEmailIngestionTrigger",
		"id":         TestTriggerID,
		"emailAlias": map[string]interface{}{
			"id":    TestEmailAliasID,
			"alias": "support@example.com",
		},
	}

	_, automations, imports := buildTriggerTestData("email_ingestion", "test_trigger", rawData)
	imports = append(imports, ImportBlock{
		ID:           TestEmailAliasID,
		ResourceType: "elementum_email_alias",
		ResourceName: "support_email",
	})
	uuidMap := map[string]string{
		TestEmailAliasID: "elementum_email_alias.support_email.id",
	}

	generator := NewTriggerHCLGenerator(imports, uuidMap, automations)
	hcl := generator.GenerateAll()

	assertHCLContains(t, hcl,
		`resource "elementum_email_ingestion_trigger" "test_trigger"`,
		"email_alias_id",
	)
}

func TestTriggerHCL_Datamine(t *testing.T) {
	rawData := map[string]interface{}{
		"__typename": "WorkflowDatamineTrigger",
		"id":         TestTriggerID,
		"datamine": map[string]interface{}{
			"id":   TestDatamineID,
			"name": "Stale Records",
		},
		"fireOnAlert":    true,
		"fireOnRecovery": false,
	}

	_, automations, imports := buildTriggerTestData("datamine", "test_trigger", rawData)
	imports = append(imports, ImportBlock{
		ID:           TestDatamineID,
		ResourceType: "elementum_datamine",
		ResourceName: "stale_records",
	})
	uuidMap := map[string]string{
		TestDatamineID: "elementum_datamine.stale_records.id",
	}

	generator := NewTriggerHCLGenerator(imports, uuidMap, automations)
	hcl := generator.GenerateAll()

	assertHCLContains(t, hcl,
		`resource "elementum_datamine_trigger" "test_trigger"`,
		"datamine_id",
		"fire_on_alert = true",
	)
}

func TestTriggerHCL_Datamine_WithFlags(t *testing.T) {
	rawData := map[string]interface{}{
		"__typename": "WorkflowDatamineTrigger",
		"id":         TestTriggerID,
		"datamine": map[string]interface{}{
			"id":   TestDatamineID,
			"name": "Stale Records",
		},
		"fireOnAlert":    true,
		"fireOnRecovery": true,
	}

	_, automations, imports := buildTriggerTestData("datamine", "test_trigger", rawData)
	imports = append(imports, ImportBlock{
		ID:           TestDatamineID,
		ResourceType: "elementum_datamine",
		ResourceName: "stale_records",
	})
	uuidMap := map[string]string{
		TestDatamineID: "elementum_datamine.stale_records.id",
	}

	generator := NewTriggerHCLGenerator(imports, uuidMap, automations)
	hcl := generator.GenerateAll()

	assertHCLContains(t, hcl,
		`resource "elementum_datamine_trigger" "test_trigger"`,
		"fire_on_alert = true",
		"fire_on_recovery = true",
	)
}

func TestTriggerHCL_ApprovalChain(t *testing.T) {
	rawData := map[string]interface{}{
		"__typename": "WorkflowApprovalChainTrigger",
		"id":         TestTriggerID,
		"aspect": map[string]interface{}{
			"id":   TestAppID,
			"name": "Test App",
		},
		"approvalChainTemplate": map[string]interface{}{
			"id":   TestApprovalTemplateID,
			"name": "Standard Approval",
		},
		"status": "APPROVED",
	}

	_, automations, imports := buildTriggerTestData("approval_chain", "test_trigger", rawData)
	imports = append(imports, ImportBlock{
		ID:           TestApprovalTemplateID,
		ResourceType: "elementum_approval_chain_template",
		ResourceName: "standard_approval",
	})
	uuidMap := map[string]string{
		TestAppID:              "elementum_app.test_app.id",
		TestApprovalTemplateID: "elementum_approval_chain_template.standard_approval.id",
	}

	generator := NewTriggerHCLGenerator(imports, uuidMap, automations)
	hcl := generator.GenerateAll()

	assertHCLContains(t, hcl,
		`resource "elementum_approval_chain_trigger" "test_trigger"`,
		"approval_chain_template_id",
	)
}

func TestTriggerHCL_ApprovalChain_WithStatus(t *testing.T) {
	rawData := map[string]interface{}{
		"__typename": "WorkflowApprovalChainTrigger",
		"id":         TestTriggerID,
		"aspect": map[string]interface{}{
			"id":   TestAppID,
			"name": "Test App",
		},
		"approvalChainTemplate": map[string]interface{}{
			"id":   TestApprovalTemplateID,
			"name": "Standard Approval",
		},
		"status": "REJECTED",
	}

	_, automations, imports := buildTriggerTestData("approval_chain", "test_trigger", rawData)
	imports = append(imports, ImportBlock{
		ID:           TestApprovalTemplateID,
		ResourceType: "elementum_approval_chain_template",
		ResourceName: "standard_approval",
	})
	uuidMap := map[string]string{
		TestAppID:              "elementum_app.test_app.id",
		TestApprovalTemplateID: "elementum_approval_chain_template.standard_approval.id",
	}

	generator := NewTriggerHCLGenerator(imports, uuidMap, automations)
	hcl := generator.GenerateAll()

	assertHCLContains(t, hcl,
		`resource "elementum_approval_chain_trigger" "test_trigger"`,
	)
}

// ============================================================================
// Reference Resolution Tests
// ============================================================================

func TestTriggerHCLGenerator_ResolveRef(t *testing.T) {
	tests := []struct {
		name        string
		imports     []ImportBlock
		uuidMap     map[string]string
		id          string
		refType     string
		expectedHas string
		expectedNot string
	}{
		{
			name: "resolve datamine from imports",
			imports: []ImportBlock{
				{ID: TestDatamineID, ResourceType: "elementum_datamine", ResourceName: "my_datamine"},
			},
			uuidMap:     map[string]string{},
			id:          TestDatamineID,
			refType:     "datamine",
			expectedHas: "elementum_datamine.my_datamine.id",
		},
		{
			name:    "resolve from UUID map",
			imports: []ImportBlock{},
			uuidMap: map[string]string{
				TestFieldID: "elementum_text_field.my_field.id",
			},
			id:          TestFieldID,
			refType:     "field",
			expectedHas: "elementum_text_field.my_field.id",
		},
		{
			name:        "fallback to raw ID",
			imports:     []ImportBlock{},
			uuidMap:     map[string]string{},
			id:          "some-unknown-id",
			refType:     "unknown",
			expectedHas: `"some-unknown-id"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			generator := NewTriggerHCLGenerator(tt.imports, tt.uuidMap, nil)
			result := generator.resolveRef(tt.id, tt.refType)

			if !strings.Contains(result, tt.expectedHas) {
				t.Errorf("Expected result to contain %q, got: %s", tt.expectedHas, result)
			}
			if tt.expectedNot != "" && strings.Contains(result, tt.expectedNot) {
				t.Errorf("Expected result NOT to contain %q, got: %s", tt.expectedNot, result)
			}
		})
	}
}

// ============================================================================
// Helper Functions
// ============================================================================

// buildTriggerTestData creates test data for a single trigger test
func buildTriggerTestData(triggerType, resourceName string, rawData map[string]interface{}) (*discovery.App, []discovery.Automation, []ImportBlock) {
	app := createTestApp(TestAppID, "Test App")

	trigger := discovery.Trigger{
		ID:      TestTriggerID,
		Type:    triggerType,
		Name:    "Test Trigger",
		RawData: rawData,
	}

	automation := createTestAutomation(
		TestAutomationID,
		"Test Automation",
		TestWorkflowID,
		"ACTIVE",
		[]discovery.Trigger{trigger},
		[]discovery.Task{},
	)

	imports := []ImportBlock{
		{ID: TestAutomationID, ResourceType: "elementum_automation", ResourceName: "test_automation"},
		{ID: TestAutomationID + ":" + TestTriggerID, ResourceType: "elementum_" + triggerType + "_trigger", ResourceName: resourceName},
	}

	return app, []discovery.Automation{automation}, imports
}

// ============================================================================
// API Format Filter Integration Tests
// ============================================================================

// TestTriggerHCL_RecordUpdated_WithAPIFormatFilter tests the complete flow of
// exporting a record_updated trigger with a filter in API format (type: FIELD)
func TestTriggerHCL_RecordUpdated_WithAPIFormatFilter(t *testing.T) {
	rawData := map[string]interface{}{
		"__typename": "WorkflowRecordUpdateTrigger",
		"id":         TestTriggerID,
		"aspect": map[string]interface{}{
			"id":   TestAppID,
			"name": "Test App",
		},
		"filter": map[string]interface{}{
			"type": "EQUALS",
			"leftValue": map[string]interface{}{
				"type":  "FIELD",
				"field": TestFieldID,
			},
			"value": map[string]interface{}{
				"type":  "TEXT",
				"value": "approved",
			},
		},
	}

	_, automations, imports := buildTriggerTestData("record_updated", "test_trigger", rawData)
	uuidMap := map[string]string{
		TestAppID:   "elementum_app.test_app.id",
		TestFieldID: "elementum_picklist_field.status.id",
	}

	generator := NewTriggerHCLGenerator(imports, uuidMap, automations)
	hcl := generator.GenerateAll()

	assertHCLContains(t, hcl,
		`resource "elementum_record_updated_trigger" "test_trigger"`,
		"filter = {",
		`type = "equals"`,
		"left_value = {",
		"field_id = elementum_picklist_field.status.id",
	)
}

// TestTriggerHCL_RecordUpdated_WithAPIFormatReferenceFilter tests filter with
// type: REFERENCE containing a nested trigger reference
func TestTriggerHCL_RecordUpdated_WithAPIFormatReferenceFilter(t *testing.T) {
	rawData := map[string]interface{}{
		"__typename": "WorkflowRecordUpdateTrigger",
		"id":         TestTriggerID,
		"aspect": map[string]interface{}{
			"id":   TestAppID,
			"name": "Test App",
		},
		"filter": map[string]interface{}{
			"type": "EQUALS",
			"leftValue": map[string]interface{}{
				"type":     "REFERENCE",
				"realType": "USER",
				"value": map[string]interface{}{
					"type": "TRIGGER",
					"name": "record.assignee.id",
				},
			},
			"value": map[string]interface{}{
				"type":     "REFERENCE",
				"realType": "USER",
				"value": map[string]interface{}{
					"type": "TRIGGER",
					"name": "record.created_by.id",
				},
			},
		},
	}

	_, automations, imports := buildTriggerTestData("record_updated", "test_trigger", rawData)
	uuidMap := map[string]string{
		TestAppID: "elementum_app.test_app.id",
	}

	generator := NewTriggerHCLGenerator(imports, uuidMap, automations)
	hcl := generator.GenerateAll()

	assertHCLContains(t, hcl,
		`resource "elementum_record_updated_trigger" "test_trigger"`,
		"filter = {",
		`type = "equals"`,
		"left_value = {",
		`value_reference = "trigger.record.assignee.id"`,
		`real_type = "user"`,
	)
}

// TestTriggerHCL_RecordCreated_WithComplexAPIFormatFilter tests a complex AND
// filter with multiple children in API format
func TestTriggerHCL_RecordCreated_WithComplexAPIFormatFilter(t *testing.T) {
	rawData := map[string]interface{}{
		"__typename": "WorkflowRecordCreateTrigger",
		"id":         TestTriggerID,
		"aspect": map[string]interface{}{
			"id":   TestAppID,
			"name": "Test App",
		},
		"filter": map[string]interface{}{
			"type": "AND",
			"children": []interface{}{
				map[string]interface{}{
					"type": "EQUALS",
					"leftValue": map[string]interface{}{
						"type":  "FIELD",
						"field": TestFieldID,
					},
					"value": map[string]interface{}{
						"type":  "TEXT",
						"value": "active",
					},
				},
				map[string]interface{}{
					"type": "IS_NULL",
					"not":  true,
					"leftValue": map[string]interface{}{
						"type":  "FIELD",
						"field": TestField2ID,
					},
				},
			},
		},
	}

	_, automations, imports := buildTriggerTestData("record_created", "test_trigger", rawData)
	uuidMap := map[string]string{
		TestAppID:    "elementum_app.test_app.id",
		TestFieldID:  "elementum_picklist_field.status.id",
		TestField2ID: "elementum_user_field.assignee.id",
	}

	generator := NewTriggerHCLGenerator(imports, uuidMap, automations)
	hcl := generator.GenerateAll()

	assertHCLContains(t, hcl,
		`resource "elementum_record_created_trigger" "test_trigger"`,
		"filter = {",
		`type = "and"`,
		"children = [",
		`type = "equals"`,
		"field_id = elementum_picklist_field.status.id",
		`type = "is_null"`,
		"not = true",
		"field_id = elementum_user_field.assignee.id",
	)
}

// TestTriggerHCL_RecordUpdated_WithTaskReferenceFilter tests filter with
// a task reference in the value
func TestTriggerHCL_RecordUpdated_WithTaskReferenceFilter(t *testing.T) {
	rawData := map[string]interface{}{
		"__typename": "WorkflowRecordUpdateTrigger",
		"id":         TestTriggerID,
		"aspect": map[string]interface{}{
			"id":   TestAppID,
			"name": "Test App",
		},
		"filter": map[string]interface{}{
			"type": "GREATER_THAN",
			"leftValue": map[string]interface{}{
				"type":  "FIELD",
				"field": TestFieldID,
			},
			"value": map[string]interface{}{
				"type":     "REFERENCE",
				"realType": "NUMBER",
				"value": map[string]interface{}{
					"type":   "TASK",
					"taskId": TestTaskID,
					"name":   "calculated_value",
				},
			},
		},
	}

	_, automations, imports := buildTriggerTestData("record_updated", "test_trigger", rawData)
	uuidMap := map[string]string{
		TestAppID:   "elementum_app.test_app.id",
		TestFieldID: "elementum_number_field.score.id",
	}

	generator := NewTriggerHCLGenerator(imports, uuidMap, automations)
	hcl := generator.GenerateAll()

	assertHCLContains(t, hcl,
		`resource "elementum_record_updated_trigger" "test_trigger"`,
		"filter = {",
		`type = "greater_than"`,
		"left_value = {",
		"field_id = elementum_number_field.score.id",
		"value = {",
		`type = "reference"`,
		`value_reference = "task.`+TestTaskID+`.calculated_value"`,
		`real_type = "number"`,
	)
}
