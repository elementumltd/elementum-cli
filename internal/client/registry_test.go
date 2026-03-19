// Copyright (c) ElementumAI, Inc.
// SPDX-License-Identifier: MPL-2.0

package client

import (
	"strings"
	"testing"
)

func TestTriggerTypeRegistry_Helpers(t *testing.T) {
	t.Parallel()

	// Test GetTriggerTypeConfig
	t.Run("GetTriggerTypeConfig", func(t *testing.T) {
		config, ok := GetTriggerTypeConfig("record_created")
		if !ok {
			t.Error("expected record_created to be in registry")
		}
		if config.TypeName != "record_created" {
			t.Errorf("expected TypeName 'record_created', got %q", config.TypeName)
		}

		_, ok = GetTriggerTypeConfig("non_existent")
		if ok {
			t.Error("expected non_existent to not be in registry")
		}
	})

	// Test GetAllTriggerGraphQLFragments
	t.Run("GetAllTriggerGraphQLFragments", func(t *testing.T) {
		fragments := GetAllTriggerGraphQLFragments()
		if fragments == "" {
			t.Error("expected non-empty fragments")
		}
		if !strings.Contains(fragments, "WorkflowRecordCreateTrigger") {
			t.Error("expected fragments to contain WorkflowRecordCreateTrigger")
		}
	})

	// Test BuildTriggersQueryFragment
	t.Run("BuildTriggersQueryFragment", func(t *testing.T) {
		fragment := BuildTriggersQueryFragment()
		if !strings.Contains(fragment, "triggers {") {
			t.Error("expected fragment to contain 'triggers {'")
		}
		if !strings.Contains(fragment, "__typename") {
			t.Error("expected fragment to contain '__typename'")
		}
	})

	// Test IsRecordBasedTrigger
	t.Run("IsRecordBasedTrigger", func(t *testing.T) {
		if !IsRecordBasedTrigger("record_created") {
			t.Error("expected record_created to be record-based")
		}
		if IsRecordBasedTrigger("webhook") {
			t.Error("expected webhook to not be record-based")
		}
		if IsRecordBasedTrigger("non_existent") {
			t.Error("expected non_existent to not be record-based")
		}
	})

	// Test MapTriggerTypename
	t.Run("MapTriggerTypename", func(t *testing.T) {
		if got := MapTriggerTypename("WorkflowRecordCreateTrigger"); got != "record_created" {
			t.Errorf("expected record_created, got %q", got)
		}
		if got := MapTriggerTypename("Unknown"); got != "unknown" {
			t.Errorf("expected unknown, got %q", got)
		}
	})
}

func TestTaskTypeRegistry_Helpers(t *testing.T) {
	t.Parallel()

	// Test GetTaskTypeConfig
	t.Run("GetTaskTypeConfig", func(t *testing.T) {
		config, ok := GetTaskTypeConfig("update_field")
		if !ok {
			t.Error("expected update_field to be in registry")
		}
		if config.TypeName != "update_field" {
			t.Errorf("expected TypeName 'update_field', got %q", config.TypeName)
		}

		_, ok = GetTaskTypeConfig("non_existent")
		if ok {
			t.Error("expected non_existent to not be in registry")
		}
	})

	// Test GetAllTaskGraphQLFragments
	t.Run("GetAllTaskGraphQLFragments", func(t *testing.T) {
		fragments := GetAllTaskGraphQLFragments()
		if fragments == "" {
			t.Error("expected non-empty fragments")
		}
		if !strings.Contains(fragments, "WorkflowUpdateFieldTask") {
			t.Error("expected fragments to contain WorkflowUpdateFieldTask")
		}
	})

	// Test BuildTasksQueryFragment
	t.Run("BuildTasksQueryFragment", func(t *testing.T) {
		fragment := BuildTasksQueryFragment()
		if !strings.Contains(fragment, "tasks {") {
			t.Error("expected fragment to contain 'tasks {'")
		}
		if !strings.Contains(fragment, "previous { id }") {
			t.Error("expected fragment to contain 'previous { id }'")
		}
	})

	// Test MapTaskTypename
	t.Run("MapTaskTypename", func(t *testing.T) {
		if got := MapTaskTypename("WorkflowUpdateFieldTask"); got != "update_field" {
			t.Errorf("expected update_field, got %q", got)
		}
		if got := MapTaskTypename("Unknown"); got != "unknown" {
			t.Errorf("expected unknown, got %q", got)
		}
	})
}
