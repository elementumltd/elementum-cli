// Copyright (c) ElementumAI, Inc.
// SPDX-License-Identifier: MPL-2.0

package cmd

import (
	"fmt"

	"github.com/elementumltd/elementum-cli/ui"
	eclient "github.com/elementumltd/elementum-cli/internal/client"
)

// renderFullWorkflow renders a workflow's full details (triggers, tasks, outputs) as a tree.
func renderFullWorkflow(root *ui.TreeNode, workflow *eclient.WorkflowFullDetails) {
	// Add triggers
	if len(workflow.Triggers) > 0 {
		triggersNode := root.AddChildWithLabel("Triggers", fmt.Sprintf("(%d)", len(workflow.Triggers)))
		for _, t := range workflow.Triggers {
			triggersNode.AddChildWithLabel(t.Typename, t.ID)
		}
	}

	// Add tasks
	if len(workflow.Tasks) > 0 {
		tasksNode := root.AddChildWithLabel("Tasks", fmt.Sprintf("(%d)", len(workflow.Tasks)))
		for _, t := range workflow.Tasks {
			taskLabel := fmt.Sprintf("%s [%s]", t.Name, t.Typename)
			taskNode := tasksNode.AddChildWithLabel(taskLabel, t.ID)

			// Show previous/next chain if available
			if t.Previous != nil {
				taskNode.AddChildWithLabel("Previous", t.Previous.ID)
			}
		}
	}

	// Add outputs
	if len(workflow.Outputs) > 0 {
		outputsNode := root.AddChildWithLabel("Outputs", fmt.Sprintf("(%d)", len(workflow.Outputs)))
		for _, o := range workflow.Outputs {
			outputsNode.AddChildWithLabel(o.Name)
		}
	}
}
