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
	"testing"
)

// TestQueueFromApp_QueuesAutomationReferences verifies that queueFromAspect
// extracts and queues references from automation tasks (cross-app references)
func TestQueueFromApp_QueuesAutomationReferences(t *testing.T) {
	t.Parallel()

	uCtx := NewUnifiedDiscoveryContext("root-app-id", "App")

	app := &App{
		ID:   "root-app-id",
		Name: "Root App",
		Automations: []Automation{
			{
				ID:   "auto-1",
				Name: "Test Automation",
				Triggers: []Trigger{
					{
						ID:         "trigger-1",
						Type:       "datamine",
						DatamineID: "datamine-uuid-1",
					},
				},
				Tasks: []Task{
					{
						ID:       "task-1",
						Type:     "record_search",
						ObjectID: "target-app-uuid", // Cross-app reference
					},
					{
						ID:              "task-2",
						Type:            "find_related_records",
						RelatedObjectID: "related-app-uuid", // Another cross-app reference
					},
					{
						ID:                      "task-3",
						Type:                    "ai_classify",
						DynamicCategoryAspectID: "classify-app-uuid", // Dynamic category reference
					},
				},
			},
		},
	}

	queueFromAspect(app, uCtx)

	// Verify datamine was queued
	if !uCtx.QueuedIDs["datamine-uuid-1"] {
		t.Error("Expected datamine-uuid-1 to be queued from automation trigger")
	}

	// Verify cross-app references were queued
	if !uCtx.QueuedIDs["target-app-uuid"] {
		t.Error("Expected target-app-uuid to be queued from task.ObjectID")
	}

	if !uCtx.QueuedIDs["related-app-uuid"] {
		t.Error("Expected related-app-uuid to be queued from task.RelatedObjectID")
	}

	if !uCtx.QueuedIDs["classify-app-uuid"] {
		t.Error("Expected classify-app-uuid to be queued from task.DynamicCategoryAspectID")
	}
}

// TestQueueFromApp_QueuesAgentToolReferences verifies that queueFromAspect
// extracts and queues references from agent tools
func TestQueueFromApp_QueuesAgentToolReferences(t *testing.T) {
	t.Parallel()

	uCtx := NewUnifiedDiscoveryContext("root-app-id", "App")

	app := &App{
		ID:   "root-app-id",
		Name: "Root App",
		Agents: []Agent{
			{
				ID:   "agent-1",
				Name: "Test Agent",
				Tools: []AgentTool{
					{
						ID:       "tool-1",
						Type:     "AgentSearchAspectTool",
						AspectID: "search-target-app-uuid",
					},
					{
						ID:              "tool-2",
						Type:            "AgentRelateRecordTool",
						RelatedAspectID: "relate-target-app-uuid",
					},
					{
						ID:                    "tool-3",
						Type:                  "AgentSearchTableTool",
						SearchTableAspectID:   "element-with-search-table-uuid",
						SearchTableAspectType: "Element",
					},
				},
			},
		},
	}

	queueFromAspect(app, uCtx)

	// Verify agent tool aspect references were queued
	if !uCtx.QueuedIDs["search-target-app-uuid"] {
		t.Error("Expected search-target-app-uuid to be queued from agent tool AspectID")
	}

	if !uCtx.QueuedIDs["relate-target-app-uuid"] {
		t.Error("Expected relate-target-app-uuid to be queued from agent tool RelatedAspectID")
	}

	if !uCtx.QueuedIDs["element-with-search-table-uuid"] {
		t.Error("Expected element-with-search-table-uuid to be queued from agent tool SearchTableAspectID")
	}
}

// TestQueueFromApp_SkipsSelfReferences verifies that queueFromAspect
// does not queue references back to the aspect itself
func TestQueueFromApp_SkipsSelfReferences(t *testing.T) {
	t.Parallel()

	uCtx := NewUnifiedDiscoveryContext("root-app-id", "App")

	app := &App{
		ID:   "root-app-id",
		Name: "Root App",
		Automations: []Automation{
			{
				ID:   "auto-1",
				Name: "Test Automation",
				Tasks: []Task{
					{
						ID:       "task-1",
						Type:     "record_search",
						ObjectID: "root-app-id", // Self-reference - should be skipped
					},
				},
			},
		},
		Agents: []Agent{
			{
				ID:   "agent-1",
				Name: "Test Agent",
				Tools: []AgentTool{
					{
						ID:       "tool-1",
						Type:     "AgentSearchAspectTool",
						AspectID: "root-app-id", // Self-reference - should be skipped
					},
				},
			},
		},
	}

	queueFromAspect(app, uCtx)

	// The root app ID is already in VisitedIDs from NewUnifiedDiscoveryContext
	// So it shouldn't be in QueuedIDs
	if uCtx.QueuedIDs["root-app-id"] {
		t.Error("Self-reference should not be queued")
	}
}

// TestQueueFromApp_QueuesRelationships verifies that queueFromAspect
// queues relationship targets
func TestQueueFromApp_QueuesRelationships(t *testing.T) {
	t.Parallel()

	uCtx := NewUnifiedDiscoveryContext("root-app-id", "App")

	app := &App{
		ID:   "root-app-id",
		Name: "Root App",
		Relationships: []Relationship{
			{
				ID:                "rel-1",
				RelatedObjectID:   "related-element-uuid",
				RelatedObjectType: "Element",
			},
			{
				ID:                "rel-2",
				RelatedObjectID:   "related-table-uuid",
				RelatedObjectType: "Table",
			},
		},
	}

	queueFromAspect(app, uCtx)

	if !uCtx.QueuedIDs["related-element-uuid"] {
		t.Error("Expected related-element-uuid to be queued from relationship")
	}

	if !uCtx.QueuedIDs["related-table-uuid"] {
		t.Error("Expected related-table-uuid to be queued from relationship")
	}
}

// TestQueueFromTask_QueuesAutomationReferences verifies that queueFromAspect
// extracts and queues references from task automations
func TestQueueFromTask_QueuesAutomationReferences(t *testing.T) {
	t.Parallel()

	uCtx := NewUnifiedDiscoveryContext("root-task-id", "Task")

	task := &AspectTask{
		ID:   "root-task-id",
		Name: "Root Task",
		Automations: []Automation{
			{
				ID:   "auto-1",
				Name: "Task Automation",
				Triggers: []Trigger{
					{
						ID:         "trigger-1",
						Type:       "datamine",
						DatamineID: "task-datamine-uuid",
					},
				},
				Tasks: []Task{
					{
						ID:       "task-1",
						Type:     "record_search",
						ObjectID: "cross-app-from-task-uuid",
					},
				},
			},
		},
	}

	queueFromAspect(task, uCtx)

	if !uCtx.QueuedIDs["task-datamine-uuid"] {
		t.Error("Expected task-datamine-uuid to be queued from task automation trigger")
	}

	if !uCtx.QueuedIDs["cross-app-from-task-uuid"] {
		t.Error("Expected cross-app-from-task-uuid to be queued from task automation task")
	}
}

// TestQueueFromTask_QueuesAgentToolReferences verifies that queueFromAspect
// extracts and queues references from task agents (tasks support agents unlike elements)
func TestQueueFromTask_QueuesAgentToolReferences(t *testing.T) {
	t.Parallel()

	uCtx := NewUnifiedDiscoveryContext("root-task-id", "Task")

	task := &AspectTask{
		ID:   "root-task-id",
		Name: "Root Task",
		Agents: []Agent{
			{
				ID:   "task-agent-1",
				Name: "Task Agent",
				Tools: []AgentTool{
					{
						ID:       "tool-1",
						Type:     "AgentSearchAspectTool",
						AspectID: "agent-target-from-task-uuid",
					},
				},
			},
		},
	}

	queueFromAspect(task, uCtx)

	if !uCtx.QueuedIDs["agent-target-from-task-uuid"] {
		t.Error("Expected agent-target-from-task-uuid to be queued from task agent tool")
	}
}

// TestQueueFromTask_QueuesRelationships verifies that queueFromAspect
// queues relationship targets for tasks
func TestQueueFromTask_QueuesRelationships(t *testing.T) {
	t.Parallel()

	uCtx := NewUnifiedDiscoveryContext("root-task-id", "Task")

	task := &AspectTask{
		ID:   "root-task-id",
		Name: "Root Task",
		Relationships: []Relationship{
			{
				ID:                "rel-1",
				RelatedObjectID:   "task-related-app-uuid",
				RelatedObjectType: "App",
			},
		},
	}

	queueFromAspect(task, uCtx)

	if !uCtx.QueuedIDs["task-related-app-uuid"] {
		t.Error("Expected task-related-app-uuid to be queued from task relationship")
	}
}

// TestUnifiedDiscoveryContext_CyclePrevention verifies that the discovery
// context prevents infinite loops by not re-queuing visited items
func TestUnifiedDiscoveryContext_CyclePrevention(t *testing.T) {
	t.Parallel()

	uCtx := NewUnifiedDiscoveryContext("app-a", "App")

	// Simulate discovery: App A -> App B -> App C -> App A (cycle)
	// First, App A is already visited (root)
	if !uCtx.VisitedIDs["app-a"] {
		t.Error("Root app-a should be marked as visited initially")
	}

	// Queue App B
	queued := uCtx.Enqueue("app-b", "App")
	if !queued {
		t.Error("app-b should be queued successfully")
	}

	// Simulate processing App B - mark as visited
	uCtx.MarkVisited("app-b")

	// Queue App C
	queued = uCtx.Enqueue("app-c", "App")
	if !queued {
		t.Error("app-c should be queued successfully")
	}

	// Simulate processing App C - mark as visited
	uCtx.MarkVisited("app-c")

	// Try to queue App A again (cycle) - should fail
	queued = uCtx.Enqueue("app-a", "App")
	if queued {
		t.Error("app-a should NOT be queued again (cycle prevention)")
	}

	// Try to queue App B again - should fail
	queued = uCtx.Enqueue("app-b", "App")
	if queued {
		t.Error("app-b should NOT be queued again (already visited)")
	}
}

// TestUnifiedDiscoveryContext_QueueDeduplication verifies that items
// already in the queue are not added again
func TestUnifiedDiscoveryContext_QueueDeduplication(t *testing.T) {
	t.Parallel()

	uCtx := NewUnifiedDiscoveryContext("root-id", "App")

	// Queue the same item multiple times
	queued1 := uCtx.Enqueue("target-app", "App")
	queued2 := uCtx.Enqueue("target-app", "App")
	queued3 := uCtx.Enqueue("target-app", "App")

	if !queued1 {
		t.Error("First enqueue should succeed")
	}

	if queued2 {
		t.Error("Second enqueue of same item should fail (already queued)")
	}

	if queued3 {
		t.Error("Third enqueue of same item should fail (already queued)")
	}

	// Verify only one item in queue
	if len(uCtx.Queue) != 1 {
		t.Errorf("Expected 1 item in queue, got %d", len(uCtx.Queue))
	}
}

// TestFullRecursiveDiscovery_DiscoveredAppsGetFullQueuing verifies that
// discovered apps (not just root) have their automations and agents queued
func TestFullRecursiveDiscovery_DiscoveredAppsGetFullQueuing(t *testing.T) {
	t.Parallel()

	// This test verifies the key behavior change:
	// Previously, discovered apps used queueFromDiscoveredApp which only queued relationships.
	// Now, discovered apps should use queueFromAspect which queues relationships, automations, AND agents.

	uCtx := NewUnifiedDiscoveryContext("root-app-id", "App")

	// Simulate a discovered app (not root) with automations and agents
	discoveredApp := &App{
		ID:   "discovered-app-id",
		Name: "Discovered App",
		Relationships: []Relationship{
			{
				ID:                "rel-1",
				RelatedObjectID:   "rel-target-uuid",
				RelatedObjectType: "Element",
			},
		},
		Automations: []Automation{
			{
				ID:   "discovered-auto",
				Name: "Discovered Automation",
				Tasks: []Task{
					{
						ID:       "discovered-task",
						Type:     "record_search",
						ObjectID: "auto-referenced-app-uuid", // This should be queued!
					},
				},
			},
		},
		Agents: []Agent{
			{
				ID:   "discovered-agent",
				Name: "Discovered Agent",
				Tools: []AgentTool{
					{
						ID:       "discovered-tool",
						Type:     "AgentSearchAspectTool",
						AspectID: "agent-referenced-app-uuid", // This should be queued!
					},
				},
			},
		},
	}

	// Use queueFromAspect (full queuing) instead of old light queuing
	queueFromAspect(discoveredApp, uCtx)

	// Verify relationships are queued (this worked before)
	if !uCtx.QueuedIDs["rel-target-uuid"] {
		t.Error("Relationship target should be queued")
	}

	// Verify automations are queued (THIS IS THE KEY FIX)
	if !uCtx.QueuedIDs["auto-referenced-app-uuid"] {
		t.Error("Automation task ObjectID should be queued for discovered apps (full recursive discovery)")
	}

	// Verify agents are queued (THIS IS THE KEY FIX)
	if !uCtx.QueuedIDs["agent-referenced-app-uuid"] {
		t.Error("Agent tool AspectID should be queued for discovered apps (full recursive discovery)")
	}
}
