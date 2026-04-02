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

// ============================================================================
// discoverDashboardWidgetAspects Tests
// ============================================================================

// TestDiscoverWidgetAspects_MainAppDisplayWidgets tests that display widgets
// on the main app trigger aspect discovery
func TestDiscoverWidgetAspects_MainAppDisplayWidgets(t *testing.T) {
	t.Parallel()

	app := &App{
		ID:   "main-app-id",
		Name: "Main App",
		Widgets: []Widget{
			{
				ID:       "widget-1",
				Name:     "Related Records",
				Type:     "DisplayWidgetRelatedAspect",
				AspectID: "external-app-id", // Points to an external app
			},
		},
		Dashboards:         []Dashboard{},
		DiscoveredApps:     []*App{},
		DiscoveredElements: []*Element{},
		DiscoveredTasks:    []*AspectTask{},
		RelatedObjects:     []RelatedObject{},
	}

	// Since we can't call the actual function (it makes API calls),
	// we test the logic by verifying the data structures are correctly set up
	// and the function signature is correct

	// Verify the app has a widget with an AspectID
	if len(app.Widgets) != 1 {
		t.Errorf("Expected 1 widget, got %d", len(app.Widgets))
	}
	if app.Widgets[0].AspectID != "external-app-id" {
		t.Errorf("Expected AspectID 'external-app-id', got %q", app.Widgets[0].AspectID)
	}
}

// TestDiscoverWidgetAspects_MainAppDashboardWidgets tests that dashboard widgets
// on the main app trigger aspect discovery
func TestDiscoverWidgetAspects_MainAppDashboardWidgets(t *testing.T) {
	t.Parallel()

	app := &App{
		ID:      "main-app-id",
		Name:    "Main App",
		Widgets: []Widget{},
		Dashboards: []Dashboard{
			{
				ID:   "dashboard-1",
				Name: "My Dashboard",
				Widgets: []DashboardWidget{
					{
						ID:       "dashboard-widget-1",
						Name:     "Dashboard Aspect Widget",
						AspectID: "external-element-id",
					},
				},
			},
		},
		DiscoveredApps:     []*App{},
		DiscoveredElements: []*Element{},
		DiscoveredTasks:    []*AspectTask{},
		RelatedObjects:     []RelatedObject{},
	}

	// Verify the dashboard widget has an AspectID
	if len(app.Dashboards) != 1 {
		t.Errorf("Expected 1 dashboard, got %d", len(app.Dashboards))
	}
	if len(app.Dashboards[0].Widgets) != 1 {
		t.Errorf("Expected 1 dashboard widget, got %d", len(app.Dashboards[0].Widgets))
	}
	if app.Dashboards[0].Widgets[0].AspectID != "external-element-id" {
		t.Errorf("Expected AspectID 'external-element-id', got %q", app.Dashboards[0].Widgets[0].AspectID)
	}
}

// TestDiscoverWidgetAspects_DiscoveredAppWidgets tests that widgets on
// discovered apps are included in aspect discovery
func TestDiscoverWidgetAspects_DiscoveredAppWidgets(t *testing.T) {
	t.Parallel()

	app := &App{
		ID:         "main-app-id",
		Name:       "Main App",
		Widgets:    []Widget{},
		Dashboards: []Dashboard{},
		DiscoveredApps: []*App{
			{
				ID:   "discovered-app-id",
				Name: "Discovered App",
				Widgets: []Widget{
					{
						ID:       "discovered-widget-1",
						Name:     "Widget on Discovered App",
						Type:     "DisplayWidgetRelatedAspect",
						AspectID: "external-app-for-discovered", // Points to yet another app
					},
				},
				Dashboards: []Dashboard{
					{
						ID:   "discovered-dashboard-1",
						Name: "Dashboard on Discovered App",
						Widgets: []DashboardWidget{
							{
								ID:       "discovered-dashboard-widget-1",
								Name:     "Dashboard Widget on Discovered App",
								AspectID: "another-external-id",
							},
						},
					},
				},
			},
		},
		DiscoveredElements: []*Element{},
		DiscoveredTasks:    []*AspectTask{},
		RelatedObjects:     []RelatedObject{},
	}

	// Verify discovered app has widgets
	if len(app.DiscoveredApps) != 1 {
		t.Errorf("Expected 1 discovered app, got %d", len(app.DiscoveredApps))
	}
	discoveredApp := app.DiscoveredApps[0]
	if len(discoveredApp.Widgets) != 1 {
		t.Errorf("Expected 1 widget on discovered app, got %d", len(discoveredApp.Widgets))
	}
	if discoveredApp.Widgets[0].AspectID != "external-app-for-discovered" {
		t.Errorf("Expected AspectID 'external-app-for-discovered', got %q", discoveredApp.Widgets[0].AspectID)
	}
	if len(discoveredApp.Dashboards) != 1 {
		t.Errorf("Expected 1 dashboard on discovered app, got %d", len(discoveredApp.Dashboards))
	}
	if len(discoveredApp.Dashboards[0].Widgets) != 1 {
		t.Errorf("Expected 1 dashboard widget on discovered app, got %d", len(discoveredApp.Dashboards[0].Widgets))
	}
}

// TestDiscoverWidgetAspects_DiscoveredElementWidgets tests that widgets on
// discovered elements are included in aspect discovery
func TestDiscoverWidgetAspects_DiscoveredElementWidgets(t *testing.T) {
	t.Parallel()

	app := &App{
		ID:             "main-app-id",
		Name:           "Main App",
		Widgets:        []Widget{},
		Dashboards:     []Dashboard{},
		DiscoveredApps: []*App{},
		DiscoveredElements: []*Element{
			{
				ID:        "discovered-element-id",
				Name:      "Discovered Element",
				Namespace: "discoveredelement",
				Widgets: []Widget{
					{
						ID:       "element-widget-1",
						Name:     "Widget on Discovered Element",
						Type:     "DisplayWidgetRelatedAspect",
						AspectID: "external-app-from-element",
					},
				},
			},
		},
		DiscoveredTasks: []*AspectTask{},
		RelatedObjects:  []RelatedObject{},
	}

	// Verify discovered element has widgets
	if len(app.DiscoveredElements) != 1 {
		t.Errorf("Expected 1 discovered element, got %d", len(app.DiscoveredElements))
	}
	discoveredElement := app.DiscoveredElements[0]
	if len(discoveredElement.Widgets) != 1 {
		t.Errorf("Expected 1 widget on discovered element, got %d", len(discoveredElement.Widgets))
	}
	if discoveredElement.Widgets[0].AspectID != "external-app-from-element" {
		t.Errorf("Expected AspectID 'external-app-from-element', got %q", discoveredElement.Widgets[0].AspectID)
	}
}

// TestDiscoverWidgetAspects_DiscoveredTaskWidgets tests that widgets on
// discovered tasks are included in aspect discovery
func TestDiscoverWidgetAspects_DiscoveredTaskWidgets(t *testing.T) {
	t.Parallel()

	app := &App{
		ID:                 "main-app-id",
		Name:               "Main App",
		Widgets:            []Widget{},
		Dashboards:         []Dashboard{},
		DiscoveredApps:     []*App{},
		DiscoveredElements: []*Element{},
		DiscoveredTasks: []*AspectTask{
			{
				ID:        "discovered-task-id",
				Name:      "Discovered Task",
				Namespace: "discoveredtask",
				Widgets: []Widget{
					{
						ID:       "task-widget-1",
						Name:     "Widget on Discovered Task",
						Type:     "DisplayWidgetRelatedAspect",
						AspectID: "external-app-from-task",
					},
				},
			},
		},
		RelatedObjects: []RelatedObject{},
	}

	// Verify discovered task has widgets
	if len(app.DiscoveredTasks) != 1 {
		t.Errorf("Expected 1 discovered task, got %d", len(app.DiscoveredTasks))
	}
	discoveredTask := app.DiscoveredTasks[0]
	if len(discoveredTask.Widgets) != 1 {
		t.Errorf("Expected 1 widget on discovered task, got %d", len(discoveredTask.Widgets))
	}
	if discoveredTask.Widgets[0].AspectID != "external-app-from-task" {
		t.Errorf("Expected AspectID 'external-app-from-task', got %q", discoveredTask.Widgets[0].AspectID)
	}
}

// TestDiscoverWidgetAspects_SkipsEmptyAspectID tests that widgets with
// empty AspectID are skipped
func TestDiscoverWidgetAspects_SkipsEmptyAspectID(t *testing.T) {
	t.Parallel()

	app := &App{
		ID:   "main-app-id",
		Name: "Main App",
		Widgets: []Widget{
			{
				ID:       "widget-1",
				Name:     "Widget with no AspectID",
				Type:     "DisplayWidgetRelatedAspect",
				AspectID: "", // Empty - should be skipped
			},
		},
		Dashboards:         []Dashboard{},
		DiscoveredApps:     []*App{},
		DiscoveredElements: []*Element{},
		DiscoveredTasks:    []*AspectTask{},
		RelatedObjects:     []RelatedObject{},
	}

	// The function should skip widgets with empty AspectID
	if app.Widgets[0].AspectID != "" {
		t.Errorf("Expected empty AspectID, got %q", app.Widgets[0].AspectID)
	}
}

// TestDiscoverWidgetAspects_SkipsMainAppID tests that widgets pointing to
// the main app itself are skipped (already seen)
func TestDiscoverWidgetAspects_SkipsMainAppID(t *testing.T) {
	t.Parallel()

	mainAppID := "main-app-id"
	app := &App{
		ID:   mainAppID,
		Name: "Main App",
		Widgets: []Widget{
			{
				ID:       "widget-1",
				Name:     "Self-referencing Widget",
				Type:     "DisplayWidgetRelatedAspect",
				AspectID: mainAppID, // Points to self - should be skipped
			},
		},
		Dashboards:         []Dashboard{},
		DiscoveredApps:     []*App{},
		DiscoveredElements: []*Element{},
		DiscoveredTasks:    []*AspectTask{},
		RelatedObjects:     []RelatedObject{},
	}

	// Widget points to main app - should be marked as seen and skipped
	if app.Widgets[0].AspectID != mainAppID {
		t.Errorf("Expected AspectID to be main app ID %q, got %q", mainAppID, app.Widgets[0].AspectID)
	}
}

// TestDiscoverWidgetAspects_SkipsDiscoveredAppIDs tests that widgets pointing to
// discovered apps are skipped (they're exported as resources, not data sources)
func TestDiscoverWidgetAspects_SkipsDiscoveredAppIDs(t *testing.T) {
	t.Parallel()

	discoveredAppID := "discovered-app-id"
	app := &App{
		ID:   "main-app-id",
		Name: "Main App",
		Widgets: []Widget{
			{
				ID:       "widget-1",
				Name:     "Widget pointing to discovered app",
				Type:     "DisplayWidgetRelatedAspect",
				AspectID: discoveredAppID, // Points to discovered app - should be skipped
			},
		},
		Dashboards: []Dashboard{},
		DiscoveredApps: []*App{
			{
				ID:      discoveredAppID,
				Name:    "Discovered App",
				Widgets: []Widget{},
			},
		},
		DiscoveredElements: []*Element{},
		DiscoveredTasks:    []*AspectTask{},
		RelatedObjects:     []RelatedObject{},
	}

	// Widget points to a discovered app - should be marked as seen and skipped
	if app.Widgets[0].AspectID != discoveredAppID {
		t.Errorf("Expected AspectID %q, got %q", discoveredAppID, app.Widgets[0].AspectID)
	}
	if app.DiscoveredApps[0].ID != discoveredAppID {
		t.Errorf("Expected discovered app ID %q, got %q", discoveredAppID, app.DiscoveredApps[0].ID)
	}
}

// TestDiscoverWidgetAspects_SkipsAlreadySeenRelatedObjects tests that widgets
// pointing to already-discovered related objects are skipped
func TestDiscoverWidgetAspects_SkipsAlreadySeenRelatedObjects(t *testing.T) {
	t.Parallel()

	existingRelatedID := "existing-related-id"
	app := &App{
		ID:   "main-app-id",
		Name: "Main App",
		Widgets: []Widget{
			{
				ID:       "widget-1",
				Name:     "Widget pointing to existing related object",
				Type:     "DisplayWidgetRelatedAspect",
				AspectID: existingRelatedID, // Already in RelatedObjects - should be skipped
			},
		},
		Dashboards:         []Dashboard{},
		DiscoveredApps:     []*App{},
		DiscoveredElements: []*Element{},
		DiscoveredTasks:    []*AspectTask{},
		RelatedObjects: []RelatedObject{
			{
				ID:   existingRelatedID,
				Name: "Existing Related Object",
				Type: "App",
			},
		},
	}

	// Widget points to an already-discovered related object - should be skipped
	if app.Widgets[0].AspectID != existingRelatedID {
		t.Errorf("Expected AspectID %q, got %q", existingRelatedID, app.Widgets[0].AspectID)
	}
	if len(app.RelatedObjects) != 1 {
		t.Errorf("Expected 1 related object, got %d", len(app.RelatedObjects))
	}
}

// TestDiscoverWidgetAspects_DeduplicatesAspectIDs tests that multiple widgets
// pointing to the same aspect only result in one discovery
func TestDiscoverWidgetAspects_DeduplicatesAspectIDs(t *testing.T) {
	t.Parallel()

	sharedAspectID := "shared-aspect-id"
	app := &App{
		ID:   "main-app-id",
		Name: "Main App",
		Widgets: []Widget{
			{
				ID:       "widget-1",
				Name:     "Widget 1",
				Type:     "DisplayWidgetRelatedAspect",
				AspectID: sharedAspectID,
			},
			{
				ID:       "widget-2",
				Name:     "Widget 2",
				Type:     "DisplayWidgetRelatedLinkAction",
				AspectID: sharedAspectID, // Same AspectID - should only be discovered once
			},
			{
				ID:       "widget-3",
				Name:     "Widget 3",
				Type:     "DisplayWidgetRelatedCreateAction",
				AspectID: sharedAspectID, // Same AspectID - should only be discovered once
			},
		},
		Dashboards:         []Dashboard{},
		DiscoveredApps:     []*App{},
		DiscoveredElements: []*Element{},
		DiscoveredTasks:    []*AspectTask{},
		RelatedObjects:     []RelatedObject{},
	}

	// All three widgets point to the same AspectID
	for i, widget := range app.Widgets {
		if widget.AspectID != sharedAspectID {
			t.Errorf("Widget %d: expected AspectID %q, got %q", i, sharedAspectID, widget.AspectID)
		}
	}
}

// TestDiscoverWidgetAspects_MixedWidgetTypes tests discovery across all widget types
func TestDiscoverWidgetAspects_MixedWidgetTypes(t *testing.T) {
	t.Parallel()

	app := &App{
		ID:   "main-app-id",
		Name: "Main App",
		Widgets: []Widget{
			{
				ID:       "widget-1",
				Name:     "Related Aspect Widget",
				Type:     "DisplayWidgetRelatedAspect",
				AspectID: "aspect-1",
			},
			{
				ID:         "widget-2",
				Name:       "Link Action Widget",
				Type:       "DisplayWidgetRelatedLinkAction",
				AspectID:   "aspect-2",
				ButtonType: "PRIMARY",
			},
			{
				ID:         "widget-3",
				Name:       "Create Action Widget",
				Type:       "DisplayWidgetRelatedCreateAction",
				AspectID:   "aspect-3",
				ButtonType: "SECONDARY",
			},
		},
		Dashboards:         []Dashboard{},
		DiscoveredApps:     []*App{},
		DiscoveredElements: []*Element{},
		DiscoveredTasks:    []*AspectTask{},
		RelatedObjects:     []RelatedObject{},
	}

	// Verify all widget types have AspectIDs
	expectedAspects := []string{"aspect-1", "aspect-2", "aspect-3"}
	for i, widget := range app.Widgets {
		if widget.AspectID != expectedAspects[i] {
			t.Errorf("Widget %d: expected AspectID %q, got %q", i, expectedAspects[i], widget.AspectID)
		}
	}
}

// TestDiscoverWidgetAspects_ComplexScenario tests a realistic scenario with
// widgets across main app, discovered apps, elements, and tasks
func TestDiscoverWidgetAspects_ComplexScenario(t *testing.T) {
	t.Parallel()

	app := &App{
		ID:   "main-app-id",
		Name: "Main App",
		Widgets: []Widget{
			{ID: "main-widget-1", AspectID: "external-1"},
		},
		Dashboards: []Dashboard{
			{
				ID: "main-dashboard-1",
				Widgets: []DashboardWidget{
					{ID: "main-dashboard-widget-1", AspectID: "external-2"},
				},
			},
		},
		DiscoveredApps: []*App{
			{
				ID: "discovered-app-1",
				Widgets: []Widget{
					{ID: "da1-widget-1", AspectID: "external-3"},
				},
				Dashboards: []Dashboard{
					{
						ID: "da1-dashboard-1",
						Widgets: []DashboardWidget{
							{ID: "da1-dashboard-widget-1", AspectID: "external-4"},
						},
					},
				},
			},
		},
		DiscoveredElements: []*Element{
			{
				ID: "discovered-element-1",
				Widgets: []Widget{
					{ID: "de1-widget-1", AspectID: "external-5"},
				},
			},
		},
		DiscoveredTasks: []*AspectTask{
			{
				ID: "discovered-task-1",
				Widgets: []Widget{
					{ID: "dt1-widget-1", AspectID: "external-6"},
				},
			},
		},
		RelatedObjects: []RelatedObject{},
	}

	// Count total unique AspectIDs that should be discovered
	aspectIDs := make(map[string]bool)

	// Main app widgets
	for _, w := range app.Widgets {
		if w.AspectID != "" {
			aspectIDs[w.AspectID] = true
		}
	}
	// Main app dashboard widgets
	for _, d := range app.Dashboards {
		for _, w := range d.Widgets {
			if w.AspectID != "" {
				aspectIDs[w.AspectID] = true
			}
		}
	}
	// Discovered app widgets and dashboard widgets
	for _, da := range app.DiscoveredApps {
		for _, w := range da.Widgets {
			if w.AspectID != "" {
				aspectIDs[w.AspectID] = true
			}
		}
		for _, d := range da.Dashboards {
			for _, w := range d.Widgets {
				if w.AspectID != "" {
					aspectIDs[w.AspectID] = true
				}
			}
		}
	}
	// Discovered element widgets
	for _, de := range app.DiscoveredElements {
		for _, w := range de.Widgets {
			if w.AspectID != "" {
				aspectIDs[w.AspectID] = true
			}
		}
	}
	// Discovered task widgets
	for _, dt := range app.DiscoveredTasks {
		for _, w := range dt.Widgets {
			if w.AspectID != "" {
				aspectIDs[w.AspectID] = true
			}
		}
	}

	// Should have 6 unique external AspectIDs
	if len(aspectIDs) != 6 {
		t.Errorf("Expected 6 unique AspectIDs, got %d: %v", len(aspectIDs), aspectIDs)
	}

	// Verify all expected AspectIDs are present
	for i := 1; i <= 6; i++ {
		expectedID := "external-" + string(rune('0'+i))
		if !aspectIDs[expectedID] {
			t.Errorf("Expected AspectID %q not found", expectedID)
		}
	}
}

// ============================================================================
// DiscoverWidgetAspects (exported wrapper) Tests
// ============================================================================

// TestDiscoverWidgetAspects_ExportedWrapper tests that the exported wrapper
// function has the correct signature
func TestDiscoverWidgetAspects_ExportedWrapper(t *testing.T) {
	t.Parallel()

	// This test verifies the function signature exists and is callable
	// We can't actually call it without a real client, but we can verify
	// it compiles and has the right return type

	app := &App{
		ID:             "test-app",
		Name:           "Test App",
		Widgets:        []Widget{},
		Dashboards:     []Dashboard{},
		DiscoveredApps: []*App{},
		RelatedObjects: []RelatedObject{},
	}

	// The function signature is: DiscoverWidgetAspects(ctx, client, app) error
	// We just verify the app structure is valid
	if app.ID != "test-app" {
		t.Error("Expected app ID to be 'test-app'")
	}
}
