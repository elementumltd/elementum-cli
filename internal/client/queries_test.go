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
	"strings"
	"testing"
)

func TestQueries_BuildAppAutomationsQuery(t *testing.T) {
	t.Parallel()

	query := BuildAppAutomationsQuery()

	if !strings.Contains(query, "query GetAppAutomationsWithWorkflows") {
		t.Error("expected query to contain operation name")
	}

	if !strings.Contains(query, "triggers {") {
		t.Error("expected query to contain triggers field")
	}

	if !strings.Contains(query, "tasks {") {
		t.Error("expected query to contain tasks field")
	}

	// Verify it contains some fragments from the registries
	if !strings.Contains(query, "WorkflowRecordCreateTrigger") {
		t.Error("expected query to contain WorkflowRecordCreateTrigger fragment")
	}

	if !strings.Contains(query, "WorkflowUpdateFieldTask") {
		t.Error("expected query to contain WorkflowUpdateFieldTask fragment")
	}
}

func TestQueries_BuildWorkflowDetailsQuery(t *testing.T) {
	t.Parallel()

	query := BuildWorkflowDetailsQuery()

	if !strings.Contains(query, "query GetWorkflowFullDetails") {
		t.Error("expected query to contain operation name")
	}

	if !strings.Contains(query, "workflow(id: $workflowId)") {
		t.Error("expected query to contain workflow(id:) field")
	}

	if !strings.Contains(query, "triggers {") {
		t.Error("expected query to contain triggers field")
	}

	if !strings.Contains(query, "tasks {") {
		t.Error("expected query to contain tasks field")
	}

	// Verify it contains fragments
	if !strings.Contains(query, "WorkflowRecordCreateTrigger") {
		t.Error("expected query to contain trigger fragments")
	}

	if !strings.Contains(query, "WorkflowUpdateFieldTask") {
		t.Error("expected query to contain task fragments")
	}
}
