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

package analysis

import (
	"encoding/json"
	"testing"

	"github.com/elementumltd/elementum-cli/internal/client"
)

func TestBuildAutomationConfig_BasicLinearChain(t *testing.T) {
	wf := &client.WorkflowFullDetails{
		ID:      "wf-1",
		Name:    "Test Workflow",
		Version: 1,
		Status:  "published",
		Triggers: []client.WorkflowTriggerRaw{
			{
				ID:       "trig-1",
				Typename: "WorkflowRecordCreateTrigger",
				RawData:  map[string]interface{}{"id": "trig-1", "__typename": "WorkflowRecordCreateTrigger"},
			},
		},
		Tasks: []client.WorkflowTaskRaw{
			makeTask("task-1", "WorkflowMessageTask", "Send Notification", nil, &client.WorkflowTaskRef{ID: "task-2"}),
			makeTask("task-2", "WorkflowUpdateFieldTask", "Update Status", &client.WorkflowTaskRef{ID: "task-1"}, nil),
		},
		Outputs: []client.WorkflowOutputRaw{{Name: "result"}},
	}

	cfg := BuildAutomationConfig("auto-1", "My Automation", "ACTIVE", wf)

	if cfg.AutomationID != "auto-1" {
		t.Errorf("expected automation ID 'auto-1', got %s", cfg.AutomationID)
	}
	if cfg.AutomationName != "My Automation" {
		t.Errorf("expected automation name 'My Automation', got %s", cfg.AutomationName)
	}
	if cfg.Status != "ACTIVE" {
		t.Errorf("expected status ACTIVE, got %s", cfg.Status)
	}
	if cfg.Trigger == nil {
		t.Fatal("expected trigger node, got nil")
	}
	if cfg.Trigger.Type != "record_created" {
		t.Errorf("expected trigger type 'record_created', got %s", cfg.Trigger.Type)
	}
	if len(cfg.Tasks) != 2 {
		t.Fatalf("expected 2 tasks, got %d", len(cfg.Tasks))
	}
	if cfg.Tasks[0].Name != "Send Notification" {
		t.Errorf("expected first task 'Send Notification', got %s", cfg.Tasks[0].Name)
	}
	if cfg.Tasks[1].Name != "Update Status" {
		t.Errorf("expected second task 'Update Status', got %s", cfg.Tasks[1].Name)
	}
	if cfg.Tasks[0].NodeKind != NodeTask {
		t.Errorf("expected task node kind, got %s", cfg.Tasks[0].NodeKind)
	}
	if len(cfg.Outputs) != 1 || cfg.Outputs[0] != "result" {
		t.Errorf("expected 1 output 'result', got %v", cfg.Outputs)
	}

	// Summary
	if cfg.Summary.TriggerType != "record_created" {
		t.Errorf("expected summary trigger type 'record_created', got %s", cfg.Summary.TriggerType)
	}
	if cfg.Summary.TopLevelTasks != 2 {
		t.Errorf("expected 2 top-level tasks, got %d", cfg.Summary.TopLevelTasks)
	}
	if cfg.Summary.TotalTasks != 2 {
		t.Errorf("expected 2 total tasks, got %d", cfg.Summary.TotalTasks)
	}
}

func TestBuildAutomationConfig_SwitchTask(t *testing.T) {
	wf := &client.WorkflowFullDetails{
		ID:      "wf-2",
		Version: 1,
		Triggers: []client.WorkflowTriggerRaw{
			{ID: "trig-1", Typename: "WorkflowRecordCreateTrigger", RawData: map[string]interface{}{}},
		},
		Tasks: []client.WorkflowTaskRaw{
			makeSwitchTask("switch-1", "Route by Priority", nil, nil,
				[]interface{}{
					map[string]interface{}{"id": "case-high", "label": "High Priority"},
					map[string]interface{}{"id": "case-default", "label": "Default"},
				},
			),
			makeTask("task-in-high", "WorkflowMessageTask", "Urgent Alert", &client.WorkflowTaskRef{ID: "case-high"}, nil),
			makeTask("task-in-default", "WorkflowMessageTask", "Normal Log", &client.WorkflowTaskRef{ID: "case-default"}, nil),
		},
	}

	cfg := BuildAutomationConfig("auto-2", "Switch Test", "ACTIVE", wf)

	if len(cfg.Tasks) != 1 {
		t.Fatalf("expected 1 top-level task (switch), got %d", len(cfg.Tasks))
	}

	switchNode := cfg.Tasks[0]
	if switchNode.NodeKind != NodeSwitch {
		t.Errorf("expected switch node kind, got %s", switchNode.NodeKind)
	}
	if len(switchNode.Cases) != 2 {
		t.Fatalf("expected 2 cases, got %d", len(switchNode.Cases))
	}
	if switchNode.Cases[0].Label != "High Priority" {
		t.Errorf("expected first case label 'High Priority', got %s", switchNode.Cases[0].Label)
	}
	if len(switchNode.Cases[0].Tasks) != 1 {
		t.Fatalf("expected 1 task in high priority case, got %d", len(switchNode.Cases[0].Tasks))
	}
	if switchNode.Cases[0].Tasks[0].Name != "Urgent Alert" {
		t.Errorf("expected task name 'Urgent Alert', got %s", switchNode.Cases[0].Tasks[0].Name)
	}
	if len(switchNode.Cases[1].Tasks) != 1 {
		t.Fatalf("expected 1 task in default case, got %d", len(switchNode.Cases[1].Tasks))
	}

	// Summary
	if cfg.Summary.SwitchCount != 1 {
		t.Errorf("expected 1 switch, got %d", cfg.Summary.SwitchCount)
	}
	if cfg.Summary.TotalCases != 2 {
		t.Errorf("expected 2 total cases, got %d", cfg.Summary.TotalCases)
	}
	if cfg.Summary.TotalTasks != 3 {
		t.Errorf("expected 3 total tasks, got %d", cfg.Summary.TotalTasks)
	}
}

func TestBuildAutomationConfig_ForEachTask(t *testing.T) {
	wf := &client.WorkflowFullDetails{
		ID:      "wf-3",
		Version: 1,
		Triggers: []client.WorkflowTriggerRaw{
			{ID: "trig-1", Typename: "WorkflowRecordCreateTrigger", RawData: map[string]interface{}{}},
		},
		Tasks: []client.WorkflowTaskRaw{
			makeForEachTask("loop-1", "Process Items", nil, nil, "chain-1", "All found records"),
			makeTask("task-in-loop", "WorkflowUpdateFieldTask", "Update Item", &client.WorkflowTaskRef{ID: "chain-1"}, nil),
			makeTask("task-after-loop", "WorkflowMessageTask", "Done", &client.WorkflowTaskRef{ID: "loop-1"}, nil),
		},
	}

	cfg := BuildAutomationConfig("auto-3", "ForEach Test", "ACTIVE", wf)

	if len(cfg.Tasks) != 2 {
		t.Fatalf("expected 2 top-level tasks, got %d", len(cfg.Tasks))
	}

	forEachNode := cfg.Tasks[0]
	if forEachNode.NodeKind != NodeForEach {
		t.Errorf("expected for_each node kind, got %s", forEachNode.NodeKind)
	}
	if forEachNode.LoopList != "All found records" {
		t.Errorf("expected loop list 'All found records', got %s", forEachNode.LoopList)
	}
	if len(forEachNode.LoopBody) != 1 {
		t.Fatalf("expected 1 task in loop body, got %d", len(forEachNode.LoopBody))
	}
	if forEachNode.LoopBody[0].Name != "Update Item" {
		t.Errorf("expected loop body task 'Update Item', got %s", forEachNode.LoopBody[0].Name)
	}

	afterLoop := cfg.Tasks[1]
	if afterLoop.Name != "Done" {
		t.Errorf("expected task after loop 'Done', got %s", afterLoop.Name)
	}

	// Summary
	if cfg.Summary.ForEachCount != 1 {
		t.Errorf("expected 1 for_each, got %d", cfg.Summary.ForEachCount)
	}
	if cfg.Summary.TotalTasks != 3 {
		t.Errorf("expected 3 total tasks, got %d", cfg.Summary.TotalTasks)
	}
}

func TestBuildAutomationConfig_JSONMarshal(t *testing.T) {
	wf := &client.WorkflowFullDetails{
		ID:      "wf-4",
		Version: 2,
		Triggers: []client.WorkflowTriggerRaw{
			{ID: "trig-1", Typename: "WorkflowWebhookTrigger", RawData: map[string]interface{}{"url": "https://example.com/hook"}},
		},
		Tasks: []client.WorkflowTaskRaw{
			makeTask("task-1", "WorkflowMessageTask", "Hello", nil, nil),
		},
	}

	cfg := BuildAutomationConfig("auto-4", "JSON Test", "INACTIVE", wf)

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	// Verify it's valid JSON and round-trips
	var parsed AutomationConfig
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if parsed.AutomationName != "JSON Test" {
		t.Errorf("round-trip name mismatch: got %s", parsed.AutomationName)
	}
	if parsed.Summary.TotalTasks != 1 {
		t.Errorf("round-trip total tasks mismatch: got %d", parsed.Summary.TotalTasks)
	}
}

func TestBuildAutomationConfig_EmptyWorkflow(t *testing.T) {
	wf := &client.WorkflowFullDetails{
		ID:      "wf-5",
		Version: 1,
	}

	cfg := BuildAutomationConfig("auto-5", "Empty", "ACTIVE", wf)

	if cfg.Trigger != nil {
		t.Errorf("expected nil trigger, got %+v", cfg.Trigger)
	}
	if len(cfg.Tasks) != 0 {
		t.Errorf("expected 0 tasks, got %d", len(cfg.Tasks))
	}
	if cfg.Summary.TotalTasks != 0 {
		t.Errorf("expected 0 total tasks, got %d", cfg.Summary.TotalTasks)
	}
}

// Helper functions for building test data

func makeTask(id, typename, name string, prev, next *client.WorkflowTaskRef) client.WorkflowTaskRaw {
	raw := map[string]interface{}{
		"id":         id,
		"__typename": typename,
		"name":       name,
	}
	if prev != nil {
		raw["previous"] = map[string]interface{}{"id": prev.ID}
	}
	if next != nil {
		raw["next"] = map[string]interface{}{"id": next.ID}
	}
	return client.WorkflowTaskRaw{
		ID:       id,
		Typename: typename,
		Name:     name,
		Previous: prev,
		Next:     next,
		RawData:  raw,
	}
}

func makeSwitchTask(id, name string, prev, next *client.WorkflowTaskRef, children []interface{}) client.WorkflowTaskRaw {
	raw := map[string]interface{}{
		"id":         id,
		"__typename": "WorkflowSwitchTask",
		"name":       name,
		"children":   children,
	}
	if prev != nil {
		raw["previous"] = map[string]interface{}{"id": prev.ID}
	}
	if next != nil {
		raw["next"] = map[string]interface{}{"id": next.ID}
	}
	return client.WorkflowTaskRaw{
		ID:       id,
		Typename: "WorkflowSwitchTask",
		Name:     name,
		Previous: prev,
		Next:     next,
		RawData:  raw,
	}
}

func makeForEachTask(id, name string, prev, next *client.WorkflowTaskRef, chainID, listLabel string) client.WorkflowTaskRaw {
	raw := map[string]interface{}{
		"id":         id,
		"__typename": "WorkflowForEachTask",
		"name":       name,
		"forEach":    map[string]interface{}{"label": listLabel},
		"children":   []interface{}{map[string]interface{}{"id": chainID}},
	}
	if prev != nil {
		raw["previous"] = map[string]interface{}{"id": prev.ID}
	}
	if next != nil {
		raw["next"] = map[string]interface{}{"id": next.ID}
	}
	return client.WorkflowTaskRaw{
		ID:       id,
		Typename: "WorkflowForEachTask",
		Name:     name,
		Previous: prev,
		Next:     next,
		RawData:  raw,
	}
}
