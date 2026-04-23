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

package discovery

import (
	"context"

	"github.com/elementumltd/elementum-cli/internal/client"
)

// ConvertWorkflowToAutomation converts a WorkflowFullDetails into a discovery.Automation
// with fully populated RawData on all triggers and tasks.
func ConvertWorkflowToAutomation(automationID, automationName, workflowID, status string, fullWorkflow *client.WorkflowFullDetails) Automation {
	// Convert triggers
	var triggers []Trigger
	for _, t := range fullWorkflow.Triggers {
		triggers = append(triggers, extractTrigger(t.RawData))
	}

	// Convert tasks
	var tasks []Task
	for _, t := range fullWorkflow.Tasks {
		task := extractTask(t.RawData, workflowID)
		tasks = append(tasks, task)
	}

	// Convert outputs (WorkflowOutputRaw only has Name, no Value)
	var outputs []WorkflowOutput
	for _, o := range fullWorkflow.Outputs {
		outputs = append(outputs, WorkflowOutput{Name: o.Name})
	}

	return Automation{
		ID:           automationID,
		Name:         automationName,
		Status:       "ACTIVE", // Always treat as active for HCL export
		WorkflowID:   workflowID,
		HasPublished: true, // Always treat as published for HCL export
		Triggers:     triggers,
		Tasks:        tasks,
		Outputs:      outputs,
	}
}

// EnrichAutomationRefs queries the API for available references and populates
// FieldRefs maps on triggers and tasks. This enables the HCL generator to
// convert raw reference IDs to human-readable refs["Name"] syntax.
func EnrichAutomationRefs(ctx context.Context, c *client.Client, aspectID string, automation *Automation) {
	fetchAutomationRefs(ctx, c, aspectID, automation)
}
