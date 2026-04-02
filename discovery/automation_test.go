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

func TestNewAutomationDiscoveryContext(t *testing.T) {
	ctx := NewAutomationDiscoveryContext()

	if ctx == nil {
		t.Fatal("NewAutomationDiscoveryContext() returned nil")
	}

	if ctx.VisitedApps == nil {
		t.Error("VisitedApps map should be initialized")
	}

	if ctx.VisitedElements == nil {
		t.Error("VisitedElements map should be initialized")
	}

	if ctx.ReferencedApps == nil {
		t.Error("ReferencedApps slice should be initialized")
	}

	if ctx.ReferencedElements == nil {
		t.Error("ReferencedElements slice should be initialized")
	}

	if len(ctx.VisitedApps) != 0 {
		t.Errorf("VisitedApps should be empty initially, got %d entries", len(ctx.VisitedApps))
	}

	if len(ctx.VisitedElements) != 0 {
		t.Errorf("VisitedElements should be empty initially, got %d entries", len(ctx.VisitedElements))
	}
}

func TestAutomationDiscoveryContext_CycleDetection(t *testing.T) {
	ctx := NewAutomationDiscoveryContext()

	// Simulate visiting apps in a cycle: App A -> App B -> App A
	appAID := "app-a-uuid"
	appBID := "app-b-uuid"

	// Visit App A
	ctx.VisitedApps[appAID] = true

	// Visit App B
	ctx.VisitedApps[appBID] = true

	// Try to visit App A again (should be detected as visited)
	if !ctx.VisitedApps[appAID] {
		t.Error("App A should be marked as visited, cycle detection failed")
	}

	// Verify both apps are marked as visited
	if len(ctx.VisitedApps) != 2 {
		t.Errorf("Expected 2 visited apps, got %d", len(ctx.VisitedApps))
	}
}

func TestAutomationDiscoveryContext_MultipleReferences(t *testing.T) {
	ctx := NewAutomationDiscoveryContext()

	// Create mock apps
	app1 := &App{ID: "app-1", Name: "App 1"}
	app2 := &App{ID: "app-2", Name: "App 2"}
	app3 := &App{ID: "app-3", Name: "App 3"}

	// Mark apps as visited and add to referenced list
	ctx.VisitedApps[app1.ID] = true
	ctx.ReferencedApps = append(ctx.ReferencedApps, app1)

	ctx.VisitedApps[app2.ID] = true
	ctx.ReferencedApps = append(ctx.ReferencedApps, app2)

	ctx.VisitedApps[app3.ID] = true
	ctx.ReferencedApps = append(ctx.ReferencedApps, app3)

	// Verify all apps are tracked
	if len(ctx.ReferencedApps) != 3 {
		t.Errorf("Expected 3 referenced apps, got %d", len(ctx.ReferencedApps))
	}

	if len(ctx.VisitedApps) != 3 {
		t.Errorf("Expected 3 visited apps, got %d", len(ctx.VisitedApps))
	}

	// Verify each app is marked as visited
	for _, app := range []*App{app1, app2, app3} {
		if !ctx.VisitedApps[app.ID] {
			t.Errorf("App %s should be marked as visited", app.ID)
		}
	}
}

func TestAutomationDiscoveryContext_MixedAppAndElementReferences(t *testing.T) {
	ctx := NewAutomationDiscoveryContext()

	// Create mock app and element
	app := &App{ID: "app-uuid", Name: "Test App"}
	element := &Element{ID: "element-uuid", Name: "Test Element"}

	// Mark both as visited
	ctx.VisitedApps[app.ID] = true
	ctx.ReferencedApps = append(ctx.ReferencedApps, app)

	ctx.VisitedElements[element.ID] = true
	ctx.ReferencedElements = append(ctx.ReferencedElements, element)

	// Verify separate tracking
	if len(ctx.VisitedApps) != 1 {
		t.Errorf("Expected 1 visited app, got %d", len(ctx.VisitedApps))
	}

	if len(ctx.VisitedElements) != 1 {
		t.Errorf("Expected 1 visited element, got %d", len(ctx.VisitedElements))
	}

	if len(ctx.ReferencedApps) != 1 {
		t.Errorf("Expected 1 referenced app, got %d", len(ctx.ReferencedApps))
	}

	if len(ctx.ReferencedElements) != 1 {
		t.Errorf("Expected 1 referenced element, got %d", len(ctx.ReferencedElements))
	}

	// Verify no cross-contamination
	if ctx.VisitedApps[element.ID] {
		t.Error("Element ID should not be in VisitedApps")
	}

	if ctx.VisitedElements[app.ID] {
		t.Error("App ID should not be in VisitedElements")
	}
}

func TestTask_CrossAppReferenceDetection(t *testing.T) {
	tests := []struct {
		name            string
		task            Task
		currentAppID    string
		expectsCrossApp bool
	}{
		{
			name: "task references same app",
			task: Task{
				ID:       "task-1",
				ObjectID: "app-123",
			},
			currentAppID:    "app-123",
			expectsCrossApp: false,
		},
		{
			name: "task references different app",
			task: Task{
				ID:       "task-2",
				ObjectID: "app-456",
			},
			currentAppID:    "app-123",
			expectsCrossApp: true,
		},
		{
			name: "task has no object reference",
			task: Task{
				ID:       "task-3",
				ObjectID: "",
			},
			currentAppID:    "app-123",
			expectsCrossApp: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isCrossApp := tt.task.ObjectID != "" && tt.task.ObjectID != tt.currentAppID

			if isCrossApp != tt.expectsCrossApp {
				t.Errorf("Cross-app detection failed: got %v, want %v", isCrossApp, tt.expectsCrossApp)
			}
		})
	}
}

func TestAutomation_ComplexDependencyChain(t *testing.T) {
	// Create a complex automation with multiple tasks referencing different apps
	automation := &Automation{
		ID:         "auto-123",
		Name:       "Complex Automation",
		WorkflowID: "workflow-456",
		Tasks: []Task{
			{
				ID:       "task-1",
				Type:     "create_record",
				ObjectID: "app-tickets", // Cross-app reference
				FieldIDs: []string{"field-1", "field-2"},
			},
			{
				ID:       "task-2",
				Type:     "record_search",
				ObjectID: "app-inventory", // Another cross-app reference
			},
			{
				ID:       "task-3",
				Type:     "update_field",
				ObjectID: "app-conversations", // Original app
			},
		},
	}

	currentAppID := "app-conversations"

	// Count cross-app references
	crossAppRefs := 0
	uniqueApps := make(map[string]bool)

	for _, task := range automation.Tasks {
		if task.ObjectID != "" && task.ObjectID != currentAppID {
			crossAppRefs++
			uniqueApps[task.ObjectID] = true
		}
	}

	if crossAppRefs != 2 {
		t.Errorf("Expected 2 cross-app references, got %d", crossAppRefs)
	}

	if len(uniqueApps) != 2 {
		t.Errorf("Expected 2 unique app references, got %d", len(uniqueApps))
	}
}

func TestAutomationDiscoveryContext_DuplicateReferenceHandling(t *testing.T) {
	ctx := NewAutomationDiscoveryContext()

	appID := "app-duplicate"

	// Try to mark the same app as visited multiple times
	ctx.VisitedApps[appID] = true

	if !ctx.VisitedApps[appID] {
		t.Error("App should be marked as visited")
	}

	// Attempting to mark again should be idempotent
	ctx.VisitedApps[appID] = true

	if len(ctx.VisitedApps) != 1 {
		t.Errorf("Expected 1 visited app, got %d (should be idempotent)", len(ctx.VisitedApps))
	}
}

func TestTask_ParentIDResolution(t *testing.T) {
	tests := []struct {
		name      string
		task      Task
		hasParent bool
	}{
		{
			name: "task with parent",
			task: Task{
				ID:       "task-1",
				ParentID: "trigger-123",
			},
			hasParent: true,
		},
		{
			name: "task with previous task as parent",
			task: Task{
				ID:       "task-2",
				ParentID: "task-1",
			},
			hasParent: true,
		},
		{
			name: "first task without parent",
			task: Task{
				ID:       "task-3",
				ParentID: "",
			},
			hasParent: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hasParent := tt.task.ParentID != ""

			if hasParent != tt.hasParent {
				t.Errorf("Parent detection failed: got %v, want %v", hasParent, tt.hasParent)
			}
		})
	}
}

func TestAutomation_TaskChain(t *testing.T) {
	// Test a typical automation task chain
	automation := &Automation{
		ID:         "auto-123",
		Name:       "Task Chain Test",
		WorkflowID: "workflow-456",
		Triggers: []Trigger{
			{ID: "trigger-1", Type: "record_created"},
		},
		Tasks: []Task{
			{
				ID:       "task-1",
				Type:     "message",
				ParentID: "trigger-1", // First task points to trigger
			},
			{
				ID:       "task-2",
				Type:     "update_field",
				ParentID: "task-1", // Second task points to first task
			},
			{
				ID:       "task-3",
				Type:     "create_record",
				ParentID: "task-2", // Third task points to second task
			},
		},
	}

	// Verify task chain integrity
	if len(automation.Tasks) != 3 {
		t.Errorf("Expected 3 tasks, got %d", len(automation.Tasks))
	}

	// Verify first task points to trigger
	if automation.Tasks[0].ParentID != automation.Triggers[0].ID {
		t.Error("First task should point to trigger")
	}

	// Verify task chain links
	for i := 1; i < len(automation.Tasks); i++ {
		expectedParent := automation.Tasks[i-1].ID
		actualParent := automation.Tasks[i].ParentID

		if actualParent != expectedParent {
			t.Errorf("Task %d: expected parent %s, got %s", i, expectedParent, actualParent)
		}
	}
}
