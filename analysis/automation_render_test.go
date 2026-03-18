// Copyright (c) ElementumAI, Inc.
// SPDX-License-Identifier: MPL-2.0

package analysis

import (
	"encoding/json"
	"testing"

	"github.com/elementumltd/elementum-cli/internal/client"
)

func TestBuildAutomationConfig_NestedSwitchInForEach(t *testing.T) {
	wf := &client.WorkflowFullDetails{
		ID:      "wf-nested",
		Version: 1,
		Triggers: []client.WorkflowTriggerRaw{
			{ID: "trig-1", Typename: "WorkflowRecordCreateTrigger", RawData: map[string]interface{}{}},
		},
		Tasks: []client.WorkflowTaskRaw{
			makeForEachTask("loop-1", "Process Items", nil, nil, "chain-1", "All records"),
			makeSwitchTask("switch-in-loop", "Route Item", &client.WorkflowTaskRef{ID: "chain-1"}, nil,
				[]interface{}{
					map[string]interface{}{"id": "case-a", "label": "Type A"},
					map[string]interface{}{"id": "case-b", "label": "Type B"},
				},
			),
			makeTask("task-a", "WorkflowMessageTask", "Handle A", &client.WorkflowTaskRef{ID: "case-a"}, nil),
			makeTask("task-b", "WorkflowUpdateFieldTask", "Handle B", &client.WorkflowTaskRef{ID: "case-b"}, nil),
		},
	}

	cfg := BuildAutomationConfig("auto-nested", "Nested Test", "ACTIVE", wf)

	if len(cfg.Tasks) != 1 {
		t.Fatalf("expected 1 top-level task (for_each), got %d", len(cfg.Tasks))
	}

	forEachNode := cfg.Tasks[0]
	if forEachNode.NodeKind != NodeForEach {
		t.Fatalf("expected for_each, got %s", forEachNode.NodeKind)
	}
	if len(forEachNode.LoopBody) != 1 {
		t.Fatalf("expected 1 task in loop body, got %d", len(forEachNode.LoopBody))
	}

	switchInLoop := forEachNode.LoopBody[0]
	if switchInLoop.NodeKind != NodeSwitch {
		t.Fatalf("expected switch inside loop, got %s", switchInLoop.NodeKind)
	}
	if len(switchInLoop.Cases) != 2 {
		t.Fatalf("expected 2 cases, got %d", len(switchInLoop.Cases))
	}
	if switchInLoop.Cases[0].Label != "Type A" {
		t.Errorf("expected case label 'Type A', got %s", switchInLoop.Cases[0].Label)
	}
	if len(switchInLoop.Cases[0].Tasks) != 1 || switchInLoop.Cases[0].Tasks[0].Name != "Handle A" {
		t.Errorf("expected task 'Handle A' in case A")
	}
	if len(switchInLoop.Cases[1].Tasks) != 1 || switchInLoop.Cases[1].Tasks[0].Name != "Handle B" {
		t.Errorf("expected task 'Handle B' in case B")
	}

	// Total counts
	if cfg.Summary.TotalTasks != 4 {
		t.Errorf("expected 4 total tasks, got %d", cfg.Summary.TotalTasks)
	}
	if cfg.Summary.ForEachCount != 1 {
		t.Errorf("expected 1 for_each, got %d", cfg.Summary.ForEachCount)
	}
	if cfg.Summary.SwitchCount != 1 {
		t.Errorf("expected 1 switch, got %d", cfg.Summary.SwitchCount)
	}
}

func TestBuildAutomationConfig_MultipleSwitchCasesWithChains(t *testing.T) {
	wf := &client.WorkflowFullDetails{
		ID:      "wf-multicase",
		Version: 1,
		Triggers: []client.WorkflowTriggerRaw{
			{ID: "trig-1", Typename: "WorkflowRecordUpdateTrigger", RawData: map[string]interface{}{}},
		},
		Tasks: []client.WorkflowTaskRaw{
			makeTask("task-before", "WorkflowVariableTask", "Set Var", nil, &client.WorkflowTaskRef{ID: "switch-1"}),
			makeSwitchTask("switch-1", "Branch", &client.WorkflowTaskRef{ID: "task-before"}, &client.WorkflowTaskRef{ID: "task-after"},
				[]interface{}{
					map[string]interface{}{"id": "case-1", "label": "Open"},
					map[string]interface{}{"id": "case-2", "label": "Closed"},
				},
			),
			makeTask("case1-task1", "WorkflowMessageTask", "Open Msg", &client.WorkflowTaskRef{ID: "case-1"}, &client.WorkflowTaskRef{ID: "case1-task2"}),
			makeTask("case1-task2", "WorkflowUpdateFieldTask", "Open Update", &client.WorkflowTaskRef{ID: "case1-task1"}, nil),
			makeTask("case2-task1", "WorkflowMessageTask", "Close Msg", &client.WorkflowTaskRef{ID: "case-2"}, nil),
			makeTask("task-after", "WorkflowNotificationTask", "Notify", &client.WorkflowTaskRef{ID: "switch-1"}, nil),
		},
	}

	cfg := BuildAutomationConfig("auto-mc", "Multi Case", "ACTIVE", wf)

	if len(cfg.Tasks) != 3 {
		t.Fatalf("expected 3 top-level tasks, got %d", len(cfg.Tasks))
	}

	// First: Set Var
	if cfg.Tasks[0].Name != "Set Var" {
		t.Errorf("expected first task 'Set Var', got %s", cfg.Tasks[0].Name)
	}

	// Second: Switch
	switchNode := cfg.Tasks[1]
	if switchNode.NodeKind != NodeSwitch {
		t.Fatalf("expected switch, got %s", switchNode.NodeKind)
	}
	if len(switchNode.Cases) != 2 {
		t.Fatalf("expected 2 cases, got %d", len(switchNode.Cases))
	}

	// Case 1 should have 2 chained tasks
	if len(switchNode.Cases[0].Tasks) != 2 {
		t.Fatalf("expected 2 tasks in case 'Open', got %d", len(switchNode.Cases[0].Tasks))
	}
	if switchNode.Cases[0].Tasks[0].Name != "Open Msg" {
		t.Errorf("expected first case task 'Open Msg', got %s", switchNode.Cases[0].Tasks[0].Name)
	}
	if switchNode.Cases[0].Tasks[1].Name != "Open Update" {
		t.Errorf("expected second case task 'Open Update', got %s", switchNode.Cases[0].Tasks[1].Name)
	}

	// Case 2 should have 1 task
	if len(switchNode.Cases[1].Tasks) != 1 {
		t.Fatalf("expected 1 task in case 'Closed', got %d", len(switchNode.Cases[1].Tasks))
	}

	// Third: Notify
	if cfg.Tasks[2].Name != "Notify" {
		t.Errorf("expected last task 'Notify', got %s", cfg.Tasks[2].Name)
	}
}

func TestBuildAutomationConfig_TimelineDoesNotPanic(t *testing.T) {
	wf := &client.WorkflowFullDetails{
		ID:      "wf-render",
		Version: 1,
		Triggers: []client.WorkflowTriggerRaw{
			{ID: "trig-1", Typename: "WorkflowRecordCreateTrigger", RawData: map[string]interface{}{}},
		},
		Tasks: []client.WorkflowTaskRaw{
			makeTask("task-1", "WorkflowMessageTask", "Hello", nil, nil),
			makeSwitchTask("switch-1", "Route", &client.WorkflowTaskRef{ID: "task-1"}, nil,
				[]interface{}{
					map[string]interface{}{"id": "c1", "label": "A"},
				},
			),
			makeTask("t-in-c1", "WorkflowUpdateFieldTask", "Update", &client.WorkflowTaskRef{ID: "c1"}, nil),
		},
	}

	cfg := BuildAutomationConfig("auto-render", "Render Test", "ACTIVE", wf)

	// Just verify these don't panic
	RenderAutomationTimeline(cfg)
	RenderAutomationTimeline(nil)
}

func TestBuildAutomationConfig_JSONStructure(t *testing.T) {
	wf := &client.WorkflowFullDetails{
		ID:      "wf-json",
		Version: 3,
		Triggers: []client.WorkflowTriggerRaw{
			{ID: "trig-1", Typename: "WorkflowWebhookTrigger", RawData: map[string]interface{}{"url": "https://example.com"}},
		},
		Tasks: []client.WorkflowTaskRaw{
			makeSwitchTask("switch-1", "Branch", nil, nil,
				[]interface{}{
					map[string]interface{}{"id": "c1", "label": "High"},
					map[string]interface{}{"id": "c2", "label": "Low"},
				},
			),
			makeTask("t1", "WorkflowMessageTask", "Alert", &client.WorkflowTaskRef{ID: "c1"}, nil),
		},
	}

	cfg := BuildAutomationConfig("auto-json", "JSON Structure", "ACTIVE", wf)

	data, err := json.Marshal(cfg)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	// Verify key fields are present in JSON
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if raw["automationId"] != "auto-json" {
		t.Errorf("missing or wrong automationId")
	}
	if raw["status"] != "ACTIVE" {
		t.Errorf("missing or wrong status")
	}
	if raw["trigger"] == nil {
		t.Error("missing trigger in JSON")
	}

	tasks, ok := raw["tasks"].([]interface{})
	if !ok || len(tasks) == 0 {
		t.Error("missing or empty tasks in JSON")
	}

	summary, ok := raw["summary"].(map[string]interface{})
	if !ok {
		t.Error("missing summary in JSON")
	} else {
		if summary["switchCount"].(float64) != 1 {
			t.Errorf("expected switchCount 1 in summary")
		}
	}
}
