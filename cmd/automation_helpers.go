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

package cmd

import (
	"fmt"

	eclient "github.com/elementumltd/elementum-cli/internal/client"
	"github.com/elementumltd/elementum-cli/ui"
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
