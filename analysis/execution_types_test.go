// Copyright (c) ElementumAI, Inc.
// SPDX-License-Identifier: MPL-2.0

package analysis

import (
	"encoding/json"
	"testing"
	"time"
)

func TestActionAnalysis_IOLoaded(t *testing.T) {
	action := ActionAnalysis{
		ID:       "action-1",
		Name:     "Test Action",
		Type:     "EXECUTE_SCRIPT",
		Status:   "SUCCESS",
		IOLoaded: false,
	}

	if action.IOLoaded {
		t.Error("expected IOLoaded to be false initially")
	}

	// Simulate loading I/O
	action.Inputs = json.RawMessage(`{"key": "value"}`)
	action.Outputs = json.RawMessage(`{"result": 42}`)
	action.IOLoaded = true

	if !action.IOLoaded {
		t.Error("expected IOLoaded to be true after loading")
	}
	if string(action.Inputs) != `{"key": "value"}` {
		t.Errorf("unexpected inputs: %s", action.Inputs)
	}
	if string(action.Outputs) != `{"result": 42}` {
		t.Errorf("unexpected outputs: %s", action.Outputs)
	}
}

func TestExecutionAnalysis_JSONMarshal(t *testing.T) {
	now := time.Now()
	analysis := &ExecutionAnalysis{
		AutomationID:   "auto-1",
		AutomationName: "Test",
		ExecutionID:    "exec-1",
		Status:         "SUCCESS",
		StartedAt:      now,
		Duration:       1000,
		Version:        1,
		Actions: []ActionAnalysis{
			{
				ID:       "action-1",
				Name:     "Action 1",
				Type:     "MESSAGE",
				Status:   "SUCCESS",
				IOLoaded: false,
			},
		},
		Summary: ExecutionSummary{
			TotalActions: 1,
			SuccessCount: 1,
		},
	}

	data, err := json.MarshalIndent(analysis, "", "  ")
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var parsed ExecutionAnalysis
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if parsed.AutomationID != "auto-1" {
		t.Errorf("round-trip automation ID mismatch: got %s", parsed.AutomationID)
	}
	if len(parsed.Actions) != 1 {
		t.Errorf("round-trip actions mismatch: got %d", len(parsed.Actions))
	}
	if parsed.Actions[0].IOLoaded {
		t.Error("round-trip IOLoaded should be false")
	}
}

func TestExecutionAnalysis_JSONMarshal_WithIO(t *testing.T) {
	now := time.Now()
	analysis := &ExecutionAnalysis{
		AutomationID:   "auto-2",
		AutomationName: "With IO",
		ExecutionID:    "exec-2",
		Status:         "SUCCESS",
		StartedAt:      now,
		Duration:       2000,
		Version:        1,
		Actions: []ActionAnalysis{
			{
				ID:       "action-1",
				Name:     "Action With IO",
				Type:     "EXECUTE_SCRIPT",
				Status:   "SUCCESS",
				IOLoaded: true,
				Inputs:   json.RawMessage(`{"param1": "value1"}`),
				Outputs:  json.RawMessage(`{"result": "success"}`),
			},
		},
		Summary: ExecutionSummary{
			TotalActions: 1,
			SuccessCount: 1,
		},
	}

	data, err := json.MarshalIndent(analysis, "", "  ")
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var parsed ExecutionAnalysis
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if !parsed.Actions[0].IOLoaded {
		t.Error("expected IOLoaded to be true")
	}

	// Parse the JSON to compare values (ignoring formatting)
	var parsedInputs map[string]string
	if err := json.Unmarshal(parsed.Actions[0].Inputs, &parsedInputs); err != nil {
		t.Fatalf("failed to unmarshal inputs: %v", err)
	}
	if parsedInputs["param1"] != "value1" {
		t.Errorf("unexpected inputs: %v", parsedInputs)
	}

	var parsedOutputs map[string]string
	if err := json.Unmarshal(parsed.Actions[0].Outputs, &parsedOutputs); err != nil {
		t.Fatalf("failed to unmarshal outputs: %v", err)
	}
	if parsedOutputs["result"] != "success" {
		t.Errorf("unexpected outputs: %v", parsedOutputs)
	}
}

func TestComputeExecutionSummary(t *testing.T) {
	d100 := 100
	d200 := 200
	d300 := 300

	actions := []ActionAnalysis{
		{Status: "SUCCESS", Duration: &d100},
		{Status: "SUCCESS", Duration: &d200},
		{Status: "FAILURE", Duration: &d300},
		{Status: "RUNNING", Duration: nil},
	}

	summary := computeExecutionSummary(actions)

	if summary.TotalActions != 4 {
		t.Errorf("expected 4 total actions, got %d", summary.TotalActions)
	}
	if summary.SuccessCount != 2 {
		t.Errorf("expected 2 success, got %d", summary.SuccessCount)
	}
	if summary.FailureCount != 1 {
		t.Errorf("expected 1 failure, got %d", summary.FailureCount)
	}
	if summary.RunningCount != 1 {
		t.Errorf("expected 1 running, got %d", summary.RunningCount)
	}
	if summary.TotalDurationMs != 600 {
		t.Errorf("expected total duration 600, got %d", summary.TotalDurationMs)
	}
	if summary.AvgDurationMs != 200 {
		t.Errorf("expected avg duration 200, got %d", summary.AvgDurationMs)
	}
	if summary.MaxDurationMs != 300 {
		t.Errorf("expected max duration 300, got %d", summary.MaxDurationMs)
	}
}

func TestComputeExecutionSummary_Empty(t *testing.T) {
	actions := []ActionAnalysis{}

	summary := computeExecutionSummary(actions)

	if summary.TotalActions != 0 {
		t.Errorf("expected 0 total actions, got %d", summary.TotalActions)
	}
	if summary.SuccessCount != 0 {
		t.Errorf("expected 0 success, got %d", summary.SuccessCount)
	}
	if summary.TotalDurationMs != 0 {
		t.Errorf("expected 0 total duration, got %d", summary.TotalDurationMs)
	}
	if summary.AvgDurationMs != 0 {
		t.Errorf("expected 0 avg duration, got %d", summary.AvgDurationMs)
	}
}

func TestComputeExecutionSummary_AllStatuses(t *testing.T) {
	d10 := 10

	actions := []ActionAnalysis{
		{Status: "SUCCESS", Duration: &d10},
		{Status: "FAILURE", Duration: &d10},
		{Status: "RUNNING", Duration: &d10},
		{Status: "QUEUED", Duration: nil}, // QUEUED actions shouldn't count
		{Status: "CANCELLED", Duration: nil},
	}

	summary := computeExecutionSummary(actions)

	if summary.TotalActions != 5 {
		t.Errorf("expected 5 total actions, got %d", summary.TotalActions)
	}
	if summary.SuccessCount != 1 {
		t.Errorf("expected 1 success, got %d", summary.SuccessCount)
	}
	if summary.FailureCount != 1 {
		t.Errorf("expected 1 failure, got %d", summary.FailureCount)
	}
	if summary.RunningCount != 1 {
		t.Errorf("expected 1 running, got %d", summary.RunningCount)
	}
}

func TestExecutionAnalysis_FailedAction(t *testing.T) {
	now := time.Now()
	errorMsg := "Script execution failed"

	analysis := &ExecutionAnalysis{
		AutomationID:   "auto-3",
		AutomationName: "Failed Test",
		ExecutionID:    "exec-3",
		Status:         "FAILURE",
		StartedAt:      now,
		Duration:       500,
		Version:        1,
		Errors:         []string{"Automation failed"},
		Actions: []ActionAnalysis{
			{
				ID:       "action-1",
				Name:     "Failing Task",
				Type:     "EXECUTE_SCRIPT",
				Status:   "FAILURE",
				Error:    &errorMsg,
				IOLoaded: false,
			},
		},
		FailedAction: &ActionAnalysis{
			ID:       "action-1",
			Name:     "Failing Task",
			Type:     "EXECUTE_SCRIPT",
			Status:   "FAILURE",
			Error:    &errorMsg,
			IOLoaded: false,
		},
		Summary: ExecutionSummary{
			TotalActions: 1,
			FailureCount: 1,
		},
	}

	if analysis.FailedAction == nil {
		t.Fatal("expected failed action")
	}
	if analysis.FailedAction.Error == nil || *analysis.FailedAction.Error != errorMsg {
		t.Errorf("expected error message '%s', got %v", errorMsg, analysis.FailedAction.Error)
	}
	if len(analysis.Errors) != 1 {
		t.Errorf("expected 1 error, got %d", len(analysis.Errors))
	}

	// Test JSON round-trip preserves error
	data, err := json.Marshal(analysis)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var parsed ExecutionAnalysis
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if parsed.FailedAction == nil {
		t.Fatal("expected failed action after unmarshal")
	}
	if parsed.FailedAction.Error == nil || *parsed.FailedAction.Error != errorMsg {
		t.Errorf("expected error after unmarshal '%s', got %v", errorMsg, parsed.FailedAction.Error)
	}
}

func TestActionAnalysis_Duration(t *testing.T) {
	// Test with duration
	d100 := 100
	action := ActionAnalysis{
		ID:       "action-1",
		Name:     "Fast Action",
		Duration: &d100,
	}

	if action.Duration == nil {
		t.Error("expected duration to be set")
	}
	if *action.Duration != 100 {
		t.Errorf("expected duration 100, got %d", *action.Duration)
	}

	// Test without duration
	actionNoD := ActionAnalysis{
		ID:   "action-2",
		Name: "No Duration",
	}

	if actionNoD.Duration != nil {
		t.Error("expected duration to be nil")
	}
}

func TestExecutionSummary_Struct(t *testing.T) {
	summary := ExecutionSummary{
		TotalActions:    10,
		SuccessCount:    7,
		FailureCount:    2,
		RunningCount:    1,
		TotalDurationMs: 5000,
		AvgDurationMs:   500,
		MaxDurationMs:   2000,
	}

	data, err := json.Marshal(summary)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var parsed ExecutionSummary
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if parsed.TotalActions != 10 {
		t.Errorf("expected 10 total, got %d", parsed.TotalActions)
	}
	if parsed.SuccessCount != 7 {
		t.Errorf("expected 7 success, got %d", parsed.SuccessCount)
	}
	if parsed.FailureCount != 2 {
		t.Errorf("expected 2 failure, got %d", parsed.FailureCount)
	}
	if parsed.MaxDurationMs != 2000 {
		t.Errorf("expected max 2000, got %d", parsed.MaxDurationMs)
	}
}
