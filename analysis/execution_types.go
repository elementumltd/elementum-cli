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
	"time"

	"github.com/elementumltd/elementum-cli/internal/client"
)

// ExecutionAnalysis is the structured representation of an automation execution
// for rendering in CLI timeline or HTML waterfall formats.
type ExecutionAnalysis struct {
	AutomationID   string           `json:"automationId"`
	AutomationName string           `json:"automationName"`
	ExecutionID    string           `json:"executionId"`
	Status         string           `json:"status"`
	StartedAt      time.Time        `json:"startedAt"`
	CompletedAt    *time.Time       `json:"completedAt,omitempty"`
	Duration       int              `json:"duration"`
	Version        int              `json:"version"`
	Errors         []string         `json:"errors,omitempty"`
	Actions        []ActionAnalysis `json:"actions"`
	FailedAction   *ActionAnalysis  `json:"failedAction,omitempty"`
	Summary        ExecutionSummary `json:"summary"`
}

// ActionAnalysis represents a single action execution with optional inputs/outputs.
type ActionAnalysis struct {
	ID          string          `json:"id"`
	Name        string          `json:"name"`
	Type        string          `json:"type"`
	Status      string          `json:"status"`
	Duration    *int            `json:"duration,omitempty"`
	StartedAt   *time.Time      `json:"startedAt,omitempty"`
	CompletedAt *time.Time      `json:"completedAt,omitempty"`
	Error       *string         `json:"error,omitempty"`
	Inputs      json.RawMessage `json:"inputs,omitempty"`
	Outputs     json.RawMessage `json:"outputs,omitempty"`
	IOLoaded    bool            `json:"ioLoaded"` // Tracks if I/O has been fetched (for lazy loading)
}

// ExecutionSummary provides aggregate counts for the execution.
type ExecutionSummary struct {
	TotalActions    int   `json:"totalActions"`
	SuccessCount    int   `json:"successCount"`
	FailureCount    int   `json:"failureCount"`
	RunningCount    int   `json:"runningCount"`
	TotalDurationMs int64 `json:"totalDurationMs"`
	AvgDurationMs   int64 `json:"avgDurationMs"`
	MaxDurationMs   int64 `json:"maxDurationMs"`
}

// BuildExecutionAnalysis constructs an ExecutionAnalysis from the genqlient response
// for the basic GetAutomationExecution query (without inputs/outputs).
func BuildExecutionAnalysis(
	automationID, automationName string,
	exec *client.GetAutomationExecutionOrganizationAutomationExecution,
) (*ExecutionAnalysis, error) {
	startedAt, _ := time.Parse(time.RFC3339, exec.StartedAt)

	ea := &ExecutionAnalysis{
		AutomationID:   automationID,
		AutomationName: automationName,
		ExecutionID:    exec.Id,
		Status:         string(exec.Status),
		StartedAt:      startedAt,
		Duration:       exec.Duration,
		Version:        exec.Version,
		Errors:         exec.Errors,
		Actions:        []ActionAnalysis{},
	}

	if exec.CompletedAt != nil {
		completedAt, _ := time.Parse(time.RFC3339, *exec.CompletedAt)
		ea.CompletedAt = &completedAt
	}

	// Parse actions
	if exec.Actions != nil {
		for _, edge := range exec.Actions.Edges {
			action := convertBasicActionExecution(edge.Node)
			ea.Actions = append(ea.Actions, action)
		}
	}

	// Parse failed action
	if exec.FailedAction != nil {
		failed := convertFailedActionExecution(*exec.FailedAction)
		ea.FailedAction = &failed
	}

	// Compute summary
	ea.Summary = computeExecutionSummary(ea.Actions)

	return ea, nil
}

// BuildExecutionAnalysisWithIO constructs an ExecutionAnalysis from the aspect-routed
// response that includes inputs/outputs.
func BuildExecutionAnalysisWithIO(
	automationID, automationName string,
	exec *client.GetAutomationExecutionWithActionIOOrganizationAspectAutomationExecution,
) (*ExecutionAnalysis, error) {
	startedAt, _ := time.Parse(time.RFC3339, exec.StartedAt)

	ea := &ExecutionAnalysis{
		AutomationID:   automationID,
		AutomationName: automationName,
		ExecutionID:    exec.Id,
		Status:         string(exec.Status),
		StartedAt:      startedAt,
		Duration:       exec.Duration,
		Version:        exec.Version,
		Errors:         exec.Errors,
		Actions:        []ActionAnalysis{},
	}

	if exec.CompletedAt != nil {
		completedAt, _ := time.Parse(time.RFC3339, *exec.CompletedAt)
		ea.CompletedAt = &completedAt
	}

	// Parse actions with I/O
	if exec.Actions != nil {
		for _, edge := range exec.Actions.Edges {
			action := convertActionExecutionWithIO(edge.Node)
			ea.Actions = append(ea.Actions, action)
		}
	}

	// Parse failed action with I/O
	if exec.FailedAction != nil {
		failed := convertFailedActionExecutionWithIO(*exec.FailedAction)
		ea.FailedAction = &failed
	}

	// Compute summary
	ea.Summary = computeExecutionSummary(ea.Actions)

	return ea, nil
}

// BuildExecutionAnalysisMetadata constructs an ExecutionAnalysis from the metadata-only
// response (no inputs/outputs). This is the fast path for initial display.
func BuildExecutionAnalysisMetadata(
	automationID, automationName string,
	exec *client.GetAutomationExecutionMetadataOrganizationAspectAutomationExecution,
) (*ExecutionAnalysis, error) {
	startedAt, _ := time.Parse(time.RFC3339, exec.StartedAt)

	ea := &ExecutionAnalysis{
		AutomationID:   automationID,
		AutomationName: automationName,
		ExecutionID:    exec.Id,
		Status:         string(exec.Status),
		StartedAt:      startedAt,
		Duration:       exec.Duration,
		Version:        exec.Version,
		Errors:         exec.Errors,
		Actions:        []ActionAnalysis{},
	}

	if exec.CompletedAt != nil {
		completedAt, _ := time.Parse(time.RFC3339, *exec.CompletedAt)
		ea.CompletedAt = &completedAt
	}

	// Parse actions (without I/O)
	if exec.Actions != nil {
		for _, edge := range exec.Actions.Edges {
			action := convertMetadataActionExecution(edge.Node)
			ea.Actions = append(ea.Actions, action)
		}
	}

	// Parse failed action
	if exec.FailedAction != nil {
		failed := convertFailedMetadataActionExecution(*exec.FailedAction)
		ea.FailedAction = &failed
	}

	// Compute summary
	ea.Summary = computeExecutionSummary(ea.Actions)

	return ea, nil
}

func convertMetadataActionExecution(node client.GetAutomationExecutionMetadataOrganizationAspectAutomationExecutionActionsActionExecutionConnectionEdgesActionExecutionEdgeNodeActionExecution) ActionAnalysis {
	// The node embeds ActionExecutionMetadata
	action := ActionAnalysis{
		ID:       node.Id,
		Type:     node.Type,
		Status:   string(node.Status),
		IOLoaded: false, // I/O not loaded in metadata-only query
	}

	if node.Action != nil {
		a := *node.Action
		if name := a.GetName(); name != nil {
			action.Name = *name
		}
	}

	if node.Duration != nil {
		action.Duration = node.Duration
	}

	if node.StartedAt != nil {
		if t, err := time.Parse(time.RFC3339, *node.StartedAt); err == nil {
			action.StartedAt = &t
		}
	}

	if node.CompletedAt != nil {
		if t, err := time.Parse(time.RFC3339, *node.CompletedAt); err == nil {
			action.CompletedAt = &t
		}
	}

	if node.Error != nil {
		action.Error = &node.Error.Reason
	}

	return action
}

func convertFailedMetadataActionExecution(node client.GetAutomationExecutionMetadataOrganizationAspectAutomationExecutionFailedActionActionExecution) ActionAnalysis {
	action := ActionAnalysis{
		ID:       node.Id,
		Type:     node.Type,
		Status:   string(node.Status),
		IOLoaded: false,
	}

	if node.Action != nil {
		a := *node.Action
		if name := a.GetName(); name != nil {
			action.Name = *name
		}
	}

	if node.Duration != nil {
		action.Duration = node.Duration
	}

	if node.Error != nil {
		action.Error = &node.Error.Reason
	}

	return action
}

func convertBasicActionExecution(node client.GetAutomationExecutionOrganizationAutomationExecutionActionsActionExecutionConnectionEdgesActionExecutionEdgeNodeActionExecution) ActionAnalysis {
	action := ActionAnalysis{
		ID:     node.Id,
		Type:   node.Type,
		Status: string(node.Status),
	}

	if node.Action != nil {
		a := *node.Action
		if name := a.GetName(); name != nil {
			action.Name = *name
		}
	}

	if node.Duration != nil {
		action.Duration = node.Duration
	}

	if node.StartedAt != nil {
		if t, err := time.Parse(time.RFC3339, *node.StartedAt); err == nil {
			action.StartedAt = &t
		}
	}

	if node.CompletedAt != nil {
		if t, err := time.Parse(time.RFC3339, *node.CompletedAt); err == nil {
			action.CompletedAt = &t
		}
	}

	if node.Error != nil {
		action.Error = &node.Error.Reason
	}

	return action
}

func convertFailedActionExecution(node client.GetAutomationExecutionOrganizationAutomationExecutionFailedActionActionExecution) ActionAnalysis {
	action := ActionAnalysis{
		ID:     node.Id,
		Type:   node.Type,
		Status: string(node.Status),
	}

	if node.Action != nil {
		a := *node.Action
		if name := a.GetName(); name != nil {
			action.Name = *name
		}
	}

	if node.Error != nil {
		action.Error = &node.Error.Reason
	}

	return action
}

func convertActionExecutionWithIO(node client.GetAutomationExecutionWithActionIOOrganizationAspectAutomationExecutionActionsActionExecutionConnectionEdgesActionExecutionEdgeNodeActionExecution) ActionAnalysis {
	// The node embeds ActionExecutionWithIO
	action := ActionAnalysis{
		ID:       node.Id,
		Type:     node.Type,
		Status:   string(node.Status),
		IOLoaded: true, // I/O is loaded with this query
	}

	if node.Action != nil {
		a := *node.Action
		if name := a.GetName(); name != nil {
			action.Name = *name
		}
	}

	if node.Duration != nil {
		action.Duration = node.Duration
	}

	if node.StartedAt != nil {
		if t, err := time.Parse(time.RFC3339, *node.StartedAt); err == nil {
			action.StartedAt = &t
		}
	}

	if node.CompletedAt != nil {
		if t, err := time.Parse(time.RFC3339, *node.CompletedAt); err == nil {
			action.CompletedAt = &t
		}
	}

	if node.Error != nil {
		action.Error = &node.Error.Reason
	}

	// Include inputs/outputs from the embedded ActionExecutionWithIO
	if node.InputsOutputs != nil {
		if node.InputsOutputs.Inputs != nil {
			action.Inputs = *node.InputsOutputs.Inputs
		}
		if node.InputsOutputs.Outputs != nil {
			action.Outputs = *node.InputsOutputs.Outputs
		}
	}

	return action
}

func convertFailedActionExecutionWithIO(node client.GetAutomationExecutionWithActionIOOrganizationAspectAutomationExecutionFailedActionActionExecution) ActionAnalysis {
	// The node embeds ActionExecutionWithIO
	action := ActionAnalysis{
		ID:     node.Id,
		Type:   node.Type,
		Status: string(node.Status),
	}

	if node.Action != nil {
		a := *node.Action
		if name := a.GetName(); name != nil {
			action.Name = *name
		}
	}

	if node.Duration != nil {
		action.Duration = node.Duration
	}

	if node.Error != nil {
		action.Error = &node.Error.Reason
	}

	// Include inputs/outputs
	if node.InputsOutputs != nil {
		if node.InputsOutputs.Inputs != nil {
			action.Inputs = *node.InputsOutputs.Inputs
		}
		if node.InputsOutputs.Outputs != nil {
			action.Outputs = *node.InputsOutputs.Outputs
		}
	}

	return action
}

func computeExecutionSummary(actions []ActionAnalysis) ExecutionSummary {
	summary := ExecutionSummary{
		TotalActions: len(actions),
	}

	var totalMs int64
	var count int
	for _, a := range actions {
		switch a.Status {
		case "SUCCESS":
			summary.SuccessCount++
		case "FAILURE":
			summary.FailureCount++
		case "RUNNING":
			summary.RunningCount++
		}

		if a.Duration != nil {
			d := int64(*a.Duration)
			totalMs += d
			count++
			if d > summary.MaxDurationMs {
				summary.MaxDurationMs = d
			}
		}
	}

	summary.TotalDurationMs = totalMs
	if count > 0 {
		summary.AvgDurationMs = totalMs / int64(count)
	}

	return summary
}
