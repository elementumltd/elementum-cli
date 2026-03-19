// Copyright (c) ElementumAI, Inc.
// SPDX-License-Identifier: MPL-2.0

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
