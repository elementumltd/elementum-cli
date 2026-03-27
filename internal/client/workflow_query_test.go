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

package client

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestBuildFullWorkflowQuery(t *testing.T) {
	t.Parallel()

	query := BuildFullWorkflowQuery()

	if query == "" {
		t.Fatal("BuildFullWorkflowQuery should return a non-empty query")
	}

	requiredFields := []string{
		"$aspectId: ID!",
		"$workflowId: ID!",
		"aspect(id: $aspectId)",
		"workflow(id: $workflowId)",
		"id",
		"name",
		"version",
		"status",
		"terminal",
		"triggers",
		"tasks",
		"__typename",
		"previous",
		"next",
		"outputs",
	}

	for _, field := range requiredFields {
		if !strings.Contains(query, field) {
			t.Errorf("BuildFullWorkflowQuery should contain %q", field)
		}
	}
}

func TestBuildFullWorkflowByIDQuery(t *testing.T) {
	t.Parallel()

	query := BuildFullWorkflowByIDQuery()

	if query == "" {
		t.Fatal("BuildFullWorkflowByIDQuery should return a non-empty query")
	}

	if !strings.Contains(query, "workflow(id: $workflowId)") {
		t.Error("should query workflow by ID")
	}

	if strings.Contains(query, "$aspectId") {
		t.Error("should NOT require aspectId")
	}
}

func TestBuildFullWorkflowQuery_IncludesTaskFragments(t *testing.T) {
	t.Parallel()

	query := BuildFullWorkflowQuery()

	taskTypes := []string{
		"WorkflowUpdateFieldTask",
		"WorkflowMessageTask",
		"WorkflowRecordSearchTask",
		"WorkflowVariableTask",
		"WorkflowSwitchTask",
		"WorkflowForEachTask",
		"WorkflowApiTask",
		"WorkflowAiAgentTask",
		"WorkflowCreateRecordTask",
	}

	for _, tt := range taskTypes {
		if !strings.Contains(query, tt) {
			t.Errorf("query should include fragment for %s", tt)
		}
	}
}

func TestBuildFullWorkflowQuery_IncludesTriggerFragments(t *testing.T) {
	t.Parallel()

	query := BuildFullWorkflowQuery()

	triggerTypes := []string{
		"WorkflowRecordCreateTrigger",
		"WorkflowRecordUpdateTrigger",
		"WorkflowOnDemandTrigger",
		"WorkflowWebhookTrigger",
	}

	for _, tt := range triggerTypes {
		if !strings.Contains(query, tt) {
			t.Errorf("query should include fragment for %s", tt)
		}
	}
}

func TestWorkflowTaskRaw_UnmarshalJSON(t *testing.T) {
	t.Parallel()

	raw := `{
		"id": "task-123",
		"__typename": "WorkflowMessageTask",
		"name": "Send Notification",
		"previous": {"id": "task-122"},
		"next": {"id": "task-124"},
		"contentsReference": {"id": "ref-1", "label": "message", "value": "Hello"}
	}`

	var task WorkflowTaskRaw
	if err := json.Unmarshal([]byte(raw), &task); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if task.ID != "task-123" {
		t.Errorf("expected ID 'task-123', got %q", task.ID)
	}
	if task.Typename != "WorkflowMessageTask" {
		t.Errorf("expected typename 'WorkflowMessageTask', got %q", task.Typename)
	}
	if task.Name != "Send Notification" {
		t.Errorf("expected name 'Send Notification', got %q", task.Name)
	}
	if task.Previous == nil || task.Previous.ID != "task-122" {
		t.Error("expected previous task ID 'task-122'")
	}
	if task.Next == nil || task.Next.ID != "task-124" {
		t.Error("expected next task ID 'task-124'")
	}

	// Raw data should contain type-specific fields
	if _, ok := task.RawData["contentsReference"]; !ok {
		t.Error("RawData should contain 'contentsReference'")
	}
}

func TestWorkflowTriggerRaw_UnmarshalJSON(t *testing.T) {
	t.Parallel()

	raw := `{
		"id": "trigger-456",
		"__typename": "WorkflowOnDemandTrigger",
		"parameters": [{"id": "p1", "name": "input_text", "fieldType": "TEXT"}]
	}`

	var trigger WorkflowTriggerRaw
	if err := json.Unmarshal([]byte(raw), &trigger); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if trigger.ID != "trigger-456" {
		t.Errorf("expected ID 'trigger-456', got %q", trigger.ID)
	}
	if trigger.Typename != "WorkflowOnDemandTrigger" {
		t.Errorf("expected typename 'WorkflowOnDemandTrigger', got %q", trigger.Typename)
	}
	if _, ok := trigger.RawData["parameters"]; !ok {
		t.Error("RawData should contain 'parameters'")
	}
}

func TestGetTaskTypeLabel(t *testing.T) {
	t.Parallel()

	tests := []struct {
		typename string
		expected string
	}{
		{"WorkflowMessageTask", "message"},
		{"WorkflowUpdateFieldTask", "update_field"},
		{"WorkflowRecordSearchTask", "record_search"},
		{"WorkflowApiTask", "api"},
		{"WorkflowSwitchTask", "switch"},
		{"WorkflowForEachTask", "for_each"},
		{"WorkflowUnknownTask", "WorkflowUnknownTask"},
	}

	for _, tt := range tests {
		t.Run(tt.typename, func(t *testing.T) {
			got := GetTaskTypeLabel(tt.typename)
			if got != tt.expected {
				t.Errorf("GetTaskTypeLabel(%q) = %q, want %q", tt.typename, got, tt.expected)
			}
		})
	}
}

func TestGetTriggerTypeLabel(t *testing.T) {
	t.Parallel()

	tests := []struct {
		typename string
		expected string
	}{
		{"WorkflowRecordCreateTrigger", "record_created"},
		{"WorkflowRecordUpdateTrigger", "record_updated"},
		{"WorkflowOnDemandTrigger", "on_demand"},
		{"WorkflowWebhookTrigger", "webhook"},
		{"WorkflowUnknownTrigger", "WorkflowUnknownTrigger"},
	}

	for _, tt := range tests {
		t.Run(tt.typename, func(t *testing.T) {
			got := GetTriggerTypeLabel(tt.typename)
			if got != tt.expected {
				t.Errorf("GetTriggerTypeLabel(%q) = %q, want %q", tt.typename, got, tt.expected)
			}
		})
	}
}

func TestMapToGenqlientTaskInput(t *testing.T) {
	t.Parallel()

	input := map[string]interface{}{
		"message": map[string]interface{}{
			"contentsReference": map[string]interface{}{
				"triggerReference": map[string]interface{}{
					"name": "record",
				},
			},
			"tags": []interface{}{"public"},
			"name": "My Message Task",
		},
	}

	typed, err := mapToGenqlientTaskInput(input)
	if err != nil {
		t.Fatalf("failed to convert: %v", err)
	}

	if typed.Message == nil {
		t.Fatal("expected Message field to be set")
	}
}

func TestMapToGenqlientTaskUpdateInput(t *testing.T) {
	t.Parallel()

	input := map[string]interface{}{
		"message": map[string]interface{}{
			"contentsReference": map[string]interface{}{
				"triggerReference": map[string]interface{}{
					"name": "record",
				},
			},
			"name": "Updated Task",
		},
	}

	typed, err := mapToGenqlientTaskUpdateInput(input)
	if err != nil {
		t.Fatalf("failed to convert: %v", err)
	}

	if typed.Message == nil {
		t.Fatal("expected Message field to be set")
	}
}

func TestMapToGenqlientOperatorInput(t *testing.T) {
	t.Parallel()

	input := map[string]interface{}{
		"switch": map[string]interface{}{
			"label": "Case 1",
		},
	}

	typed, err := mapToGenqlientOperatorInput(input)
	if err != nil {
		t.Fatalf("failed to convert: %v", err)
	}

	if typed.Switch == nil {
		t.Fatal("expected Switch field to be set")
	}
}
