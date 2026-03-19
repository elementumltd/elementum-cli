// Copyright (c) ElementumAI, Inc.
// SPDX-License-Identifier: MPL-2.0

package export

import (
	"strings"
	"testing"

	"github.com/elementumltd/elementum-cli/discovery"
)

// Test UUIDs for widget tests
const (
	TestWidgetID        = "widget-111-222-333-444"
	TestWidgetID2       = "widget-222-333-444-555"
	TestWidgetID3       = "widget-333-444-555-666"
	TestTargetAspectID  = "target-aaa-bbb-ccc-ddd"
	TestTargetAspectID2 = "target-bbb-ccc-ddd-eee"
)

// ============================================================================
// Core WidgetHCLGenerator Tests
// ============================================================================

func TestWidgetHCLGenerator_GenerateAll(t *testing.T) {
	app := &discovery.App{
		ID:   "app-123",
		Name: "Test App",
		Widgets: []discovery.Widget{
			{
				ID:       TestWidgetID,
				Name:     "Related Records",
				Type:     "DisplayWidgetRelatedAspect",
				AspectID: TestTargetAspectID,
				Columns:  []string{"col1", "col2"},
				Rows:     25,
			},
			{
				ID:         TestWidgetID2,
				Name:       "Link Record",
				Type:       "DisplayWidgetRelatedLinkAction",
				AspectID:   TestTargetAspectID,
				ButtonType: "PRIMARY",
				Color:      "#1976D2",
				Icon:       "link",
				FullWidth:  true,
				Columns:    []string{"col1"},
				Rows:       15,
			},
			{
				ID:         TestWidgetID3,
				Name:       "Create New",
				Type:       "DisplayWidgetRelatedCreateAction",
				AspectID:   TestTargetAspectID2,
				ButtonType: "SECONDARY",
				Icon:       "add",
				FullWidth:  false,
			},
		},
	}

	imports := []ImportBlock{
		{ID: "app-123", ResourceType: "elementum_app", ResourceName: "test_app"},
		{ID: "app-123:" + TestWidgetID, ResourceType: "elementum_widget", ResourceName: "related_records"},
		{ID: "app-123:" + TestWidgetID2, ResourceType: "elementum_widget", ResourceName: "link_record"},
		{ID: "app-123:" + TestWidgetID3, ResourceType: "elementum_widget", ResourceName: "create_new"},
	}
	uuidMap := make(map[string]string)

	generator := NewWidgetHCLGenerator(app, imports, uuidMap)
	hcl := generator.GenerateAll()

	// Check all three widgets are generated
	if !strings.Contains(hcl, `resource "elementum_widget" "related_records"`) {
		t.Error("Expected related_records widget resource")
	}
	if !strings.Contains(hcl, `resource "elementum_widget" "link_record"`) {
		t.Error("Expected link_record widget resource")
	}
	if !strings.Contains(hcl, `resource "elementum_widget" "create_new"`) {
		t.Error("Expected create_new widget resource")
	}

	// Check widget type attributes (must use attribute syntax with = for SingleNestedAttribute)
	if !strings.Contains(hcl, `related_object = {`) {
		t.Error("Expected related_object attribute syntax (with =)")
	}
	if !strings.Contains(hcl, `related_link_action = {`) {
		t.Error("Expected related_link_action attribute syntax (with =)")
	}
	if !strings.Contains(hcl, `related_create_action = {`) {
		t.Error("Expected related_create_action attribute syntax (with =)")
	}
}

func TestWidgetHCLGenerator_RelatedObject(t *testing.T) {
	app := &discovery.App{
		ID:   "app-123",
		Name: "Test App",
		Widgets: []discovery.Widget{
			{
				ID:       TestWidgetID,
				Name:     "Customer Orders",
				Type:     "DisplayWidgetRelatedAspect",
				AspectID: TestTargetAspectID,
				Columns:  []string{"order_id", "status", "total"},
				Rows:     100,
			},
		},
	}

	imports := []ImportBlock{
		{ID: "app-123", ResourceType: "elementum_app", ResourceName: "test_app"},
		{ID: "app-123:" + TestWidgetID, ResourceType: "elementum_widget", ResourceName: "customer_orders"},
	}

	generator := NewWidgetHCLGenerator(app, imports, make(map[string]string))
	hcl := generator.GenerateAll()

	// Verify resource block
	if !strings.Contains(hcl, `resource "elementum_widget" "customer_orders"`) {
		t.Errorf("Expected widget resource block, got:\n%s", hcl)
	}

	// Verify object_id reference
	if !strings.Contains(hcl, `object_id = elementum_app.test_app.id`) {
		t.Error("Expected object_id reference to parent app")
	}

	// Verify related_object attribute (must use = for SingleNestedAttribute)
	if !strings.Contains(hcl, `related_object = {`) {
		t.Error("Expected related_object attribute syntax (with =)")
	}

	// Verify name inside nested attribute
	if !strings.Contains(hcl, `name = "Customer Orders"`) {
		t.Error("Expected name attribute in related_object block")
	}

	// Verify target object_id
	if !strings.Contains(hcl, `object_id = "`+TestTargetAspectID+`"`) {
		t.Error("Expected target object_id in related_object block")
	}

	// Verify columns
	if !strings.Contains(hcl, `columns = [`) {
		t.Error("Expected columns array")
	}
	if !strings.Contains(hcl, `"order_id"`) {
		t.Error("Expected order_id column")
	}

	// Verify rows (non-default value)
	if !strings.Contains(hcl, `rows = 100`) {
		t.Error("Expected rows attribute with value 100")
	}
}

func TestWidgetHCLGenerator_RelatedObjectDefaultRows(t *testing.T) {
	app := &discovery.App{
		ID:   "app-123",
		Name: "Test App",
		Widgets: []discovery.Widget{
			{
				ID:       TestWidgetID,
				Name:     "Default Rows Widget",
				Type:     "DisplayWidgetRelatedAspect",
				AspectID: TestTargetAspectID,
				Rows:     50, // Default value - should be omitted
			},
		},
	}

	imports := []ImportBlock{
		{ID: "app-123", ResourceType: "elementum_app", ResourceName: "test_app"},
		{ID: "app-123:" + TestWidgetID, ResourceType: "elementum_widget", ResourceName: "default_rows"},
	}

	generator := NewWidgetHCLGenerator(app, imports, make(map[string]string))
	hcl := generator.GenerateAll()

	// Default rows (50) should NOT be included
	// Check specifically for "rows =" to avoid false positives from "default_rows" in resource name
	if strings.Contains(hcl, `rows =`) {
		t.Errorf("Default rows value (50) should not be included in HCL, got:\n%s", hcl)
	}
}

func TestWidgetHCLGenerator_RelatedLinkAction(t *testing.T) {
	app := &discovery.App{
		ID:   "app-123",
		Name: "Test App",
		Widgets: []discovery.Widget{
			{
				ID:         TestWidgetID,
				Name:       "Link to Customer",
				Type:       "DisplayWidgetRelatedLinkAction",
				AspectID:   TestTargetAspectID,
				ButtonType: "PRIMARY",
				Color:      "#FF5722",
				Icon:       "link",
				FullWidth:  true,
				Columns:    []string{"name", "email"},
				Rows:       20,
			},
		},
	}

	imports := []ImportBlock{
		{ID: "app-123", ResourceType: "elementum_app", ResourceName: "test_app"},
		{ID: "app-123:" + TestWidgetID, ResourceType: "elementum_widget", ResourceName: "link_to_customer"},
	}

	generator := NewWidgetHCLGenerator(app, imports, make(map[string]string))
	hcl := generator.GenerateAll()

	// Verify resource block
	if !strings.Contains(hcl, `resource "elementum_widget" "link_to_customer"`) {
		t.Errorf("Expected widget resource block, got:\n%s", hcl)
	}

	// Verify related_link_action attribute (must use = for SingleNestedAttribute)
	if !strings.Contains(hcl, `related_link_action = {`) {
		t.Error("Expected related_link_action attribute syntax (with =)")
	}

	// Verify button_type
	if !strings.Contains(hcl, `button_type = "PRIMARY"`) {
		t.Error("Expected button_type attribute")
	}

	// Verify color
	if !strings.Contains(hcl, `color = "#FF5722"`) {
		t.Error("Expected color attribute")
	}

	// Verify icon
	if !strings.Contains(hcl, `icon = "link"`) {
		t.Error("Expected icon attribute")
	}

	// Verify full_width
	if !strings.Contains(hcl, `full_width = true`) {
		t.Error("Expected full_width = true")
	}

	// Verify columns
	if !strings.Contains(hcl, `columns = [`) {
		t.Error("Expected columns array")
	}

	// Verify rows (non-default)
	if !strings.Contains(hcl, `rows = 20`) {
		t.Error("Expected rows attribute")
	}
}

func TestWidgetHCLGenerator_RelatedLinkActionDefaultRows(t *testing.T) {
	app := &discovery.App{
		ID:   "app-123",
		Name: "Test App",
		Widgets: []discovery.Widget{
			{
				ID:         TestWidgetID,
				Name:       "Default Link Widget",
				Type:       "DisplayWidgetRelatedLinkAction",
				AspectID:   TestTargetAspectID,
				ButtonType: "INLINE",
				Rows:       10, // Default value - should be omitted
			},
		},
	}

	imports := []ImportBlock{
		{ID: "app-123", ResourceType: "elementum_app", ResourceName: "test_app"},
		{ID: "app-123:" + TestWidgetID, ResourceType: "elementum_widget", ResourceName: "default_link"},
	}

	generator := NewWidgetHCLGenerator(app, imports, make(map[string]string))
	hcl := generator.GenerateAll()

	// Default rows (10) should NOT be included
	if strings.Contains(hcl, `rows`) {
		t.Error("Default rows value (10) should not be included in HCL for link action")
	}
}

func TestWidgetHCLGenerator_RelatedCreateAction(t *testing.T) {
	app := &discovery.App{
		ID:   "app-123",
		Name: "Test App",
		Widgets: []discovery.Widget{
			{
				ID:         TestWidgetID,
				Name:       "Create New Order",
				Type:       "DisplayWidgetRelatedCreateAction",
				AspectID:   TestTargetAspectID,
				ButtonType: "SECONDARY",
				Color:      "#4CAF50",
				Icon:       "add",
				FullWidth:  true,
			},
		},
	}

	imports := []ImportBlock{
		{ID: "app-123", ResourceType: "elementum_app", ResourceName: "test_app"},
		{ID: "app-123:" + TestWidgetID, ResourceType: "elementum_widget", ResourceName: "create_new_order"},
	}

	generator := NewWidgetHCLGenerator(app, imports, make(map[string]string))
	hcl := generator.GenerateAll()

	// Verify resource block
	if !strings.Contains(hcl, `resource "elementum_widget" "create_new_order"`) {
		t.Errorf("Expected widget resource block, got:\n%s", hcl)
	}

	// Verify related_create_action attribute (must use = for SingleNestedAttribute)
	if !strings.Contains(hcl, `related_create_action = {`) {
		t.Error("Expected related_create_action attribute syntax (with =)")
	}

	// Verify button_type
	if !strings.Contains(hcl, `button_type = "SECONDARY"`) {
		t.Error("Expected button_type attribute")
	}

	// Verify color
	if !strings.Contains(hcl, `color = "#4CAF50"`) {
		t.Error("Expected color attribute")
	}

	// Verify icon
	if !strings.Contains(hcl, `icon = "add"`) {
		t.Error("Expected icon attribute")
	}

	// Verify full_width
	if !strings.Contains(hcl, `full_width = true`) {
		t.Error("Expected full_width = true")
	}

	// Create action should NOT have columns or rows
	if strings.Contains(hcl, `columns`) {
		t.Error("Create action should not have columns")
	}
	if strings.Contains(hcl, `rows`) {
		t.Error("Create action should not have rows")
	}
}

func TestWidgetHCLGenerator_DiscoveredApps(t *testing.T) {
	discoveredApp := &discovery.App{
		ID:   "discovered-app-123",
		Name: "Discovered App",
		Widgets: []discovery.Widget{
			{
				ID:       "discovered-widget-1",
				Name:     "Discovered Widget",
				Type:     "DisplayWidgetRelatedAspect",
				AspectID: TestTargetAspectID,
			},
		},
	}

	app := &discovery.App{
		ID:             "app-123",
		Name:           "Main App",
		Widgets:        []discovery.Widget{}, // No widgets on main app
		DiscoveredApps: []*discovery.App{discoveredApp},
	}

	imports := []ImportBlock{
		{ID: "app-123", ResourceType: "elementum_app", ResourceName: "main_app"},
		{ID: "discovered-app-123", ResourceType: "elementum_app", ResourceName: "discovered_app"},
		{ID: "discovered-app-123:discovered-widget-1", ResourceType: "elementum_widget", ResourceName: "discovered_widget"},
	}

	generator := NewWidgetHCLGenerator(app, imports, make(map[string]string))
	hcl := generator.GenerateAll()

	// Verify discovered app's widget is generated
	if !strings.Contains(hcl, `resource "elementum_widget" "discovered_widget"`) {
		t.Errorf("Expected discovered widget resource block, got:\n%s", hcl)
	}

	// Verify it references the discovered app
	if !strings.Contains(hcl, `object_id = elementum_app.discovered_app.id`) {
		t.Error("Expected object_id to reference discovered app")
	}

	// Verify related_object attribute (must use = for SingleNestedAttribute)
	if !strings.Contains(hcl, `related_object = {`) {
		t.Error("Expected related_object attribute syntax (with =) for discovered widget")
	}
}

func TestWidgetHCLGenerator_DiscoveredElements(t *testing.T) {
	discoveredElement := &discovery.Element{
		ID:   "discovered-element-123",
		Name: "Discovered Element",
		Widgets: []discovery.Widget{
			{
				ID:         "element-widget-1",
				Name:       "Element Widget",
				Type:       "DisplayWidgetRelatedCreateAction",
				AspectID:   TestTargetAspectID,
				ButtonType: "PRIMARY",
			},
		},
	}

	app := &discovery.App{
		ID:                 "app-123",
		Name:               "Main App",
		Widgets:            []discovery.Widget{},
		DiscoveredElements: []*discovery.Element{discoveredElement},
	}

	imports := []ImportBlock{
		{ID: "app-123", ResourceType: "elementum_app", ResourceName: "main_app"},
		{ID: "discovered-element-123", ResourceType: "elementum_element", ResourceName: "discovered_element"},
		{ID: "discovered-element-123:element-widget-1", ResourceType: "elementum_widget", ResourceName: "element_widget"},
	}

	generator := NewWidgetHCLGenerator(app, imports, make(map[string]string))
	hcl := generator.GenerateAll()

	// Verify discovered element's widget is generated
	if !strings.Contains(hcl, `resource "elementum_widget" "element_widget"`) {
		t.Errorf("Expected element widget resource block, got:\n%s", hcl)
	}

	// Verify it references the element (not app)
	if !strings.Contains(hcl, `object_id = elementum_element.discovered_element.id`) {
		t.Error("Expected object_id to reference element")
	}

	// Verify related_create_action attribute (must use = for SingleNestedAttribute)
	if !strings.Contains(hcl, `related_create_action = {`) {
		t.Error("Expected related_create_action attribute syntax (with =) for element widget")
	}
}

func TestWidgetHCLGenerator_DiscoveredTasks(t *testing.T) {
	discoveredTask := &discovery.AspectTask{
		ID:   "discovered-task-123",
		Name: "Discovered Task",
		Widgets: []discovery.Widget{
			{
				ID:         "task-widget-1",
				Name:       "Task Widget",
				Type:       "DisplayWidgetRelatedLinkAction",
				AspectID:   TestTargetAspectID,
				ButtonType: "INLINE",
			},
		},
	}

	app := &discovery.App{
		ID:              "app-123",
		Name:            "Main App",
		Widgets:         []discovery.Widget{},
		DiscoveredTasks: []*discovery.AspectTask{discoveredTask},
	}

	imports := []ImportBlock{
		{ID: "app-123", ResourceType: "elementum_app", ResourceName: "main_app"},
		{ID: "discovered-task-123", ResourceType: "elementum_task", ResourceName: "discovered_task"},
		{ID: "discovered-task-123:task-widget-1", ResourceType: "elementum_widget", ResourceName: "task_widget"},
	}

	generator := NewWidgetHCLGenerator(app, imports, make(map[string]string))
	hcl := generator.GenerateAll()

	// Verify discovered task's widget is generated
	if !strings.Contains(hcl, `resource "elementum_widget" "task_widget"`) {
		t.Errorf("Expected task widget resource block, got:\n%s", hcl)
	}

	// Verify it references the task
	if !strings.Contains(hcl, `object_id = elementum_task.discovered_task.id`) {
		t.Error("Expected object_id to reference task")
	}

	// Verify related_link_action attribute (must use = for SingleNestedAttribute)
	if !strings.Contains(hcl, `related_link_action = {`) {
		t.Error("Expected related_link_action attribute syntax (with =) for task widget")
	}
}

func TestWidgetHCLGenerator_EmptyApp(t *testing.T) {
	app := &discovery.App{
		ID:      "app-123",
		Name:    "Test App",
		Widgets: []discovery.Widget{},
	}

	generator := NewWidgetHCLGenerator(app, nil, make(map[string]string))
	hcl := generator.GenerateAll()

	if hcl != "" {
		t.Errorf("Expected empty output for app with no widgets, got: %s", hcl)
	}
}

func TestWidgetHCLGenerator_NilApp(t *testing.T) {
	generator := NewWidgetHCLGenerator(nil, nil, make(map[string]string))
	hcl := generator.GenerateAll()

	if hcl != "" {
		t.Errorf("Expected empty output for nil app, got: %s", hcl)
	}
}

func TestWidgetHCLGenerator_MissingImport(t *testing.T) {
	app := &discovery.App{
		ID:   "app-123",
		Name: "Test App",
		Widgets: []discovery.Widget{
			{
				ID:       TestWidgetID,
				Name:     "Orphan Widget",
				Type:     "DisplayWidgetRelatedAspect",
				AspectID: TestTargetAspectID,
			},
		},
	}

	// No import block for the widget
	imports := []ImportBlock{
		{ID: "app-123", ResourceType: "elementum_app", ResourceName: "test_app"},
	}

	generator := NewWidgetHCLGenerator(app, imports, make(map[string]string))
	hcl := generator.GenerateAll()

	// Should not generate HCL for widgets without import blocks
	if strings.Contains(hcl, "elementum_widget") {
		t.Error("Should not generate HCL for widget without import block")
	}
}

func TestWidgetHCLGenerator_UnknownType(t *testing.T) {
	app := &discovery.App{
		ID:   "app-123",
		Name: "Test App",
		Widgets: []discovery.Widget{
			{
				ID:   TestWidgetID,
				Name: "Unknown Widget",
				Type: "DisplayWidgetUnknownType",
			},
		},
	}

	imports := []ImportBlock{
		{ID: "app-123", ResourceType: "elementum_app", ResourceName: "test_app"},
		{ID: "app-123:" + TestWidgetID, ResourceType: "elementum_widget", ResourceName: "unknown_widget"},
	}

	generator := NewWidgetHCLGenerator(app, imports, make(map[string]string))
	hcl := generator.GenerateAll()

	// IR generator returns nil for unknown types, so output should be empty
	if hcl != "" {
		t.Errorf("Expected empty output for unknown widget type, got:\n%s", hcl)
	}
}

func TestWidgetHCLGenerator_OptionalFieldsOmitted(t *testing.T) {
	app := &discovery.App{
		ID:   "app-123",
		Name: "Test App",
		Widgets: []discovery.Widget{
			{
				ID:         TestWidgetID,
				Name:       "Minimal Link Widget",
				Type:       "DisplayWidgetRelatedLinkAction",
				AspectID:   TestTargetAspectID,
				ButtonType: "INLINE",
				// Color, Icon, FullWidth not set (should be omitted)
			},
		},
	}

	imports := []ImportBlock{
		{ID: "app-123", ResourceType: "elementum_app", ResourceName: "test_app"},
		{ID: "app-123:" + TestWidgetID, ResourceType: "elementum_widget", ResourceName: "minimal_link"},
	}

	generator := NewWidgetHCLGenerator(app, imports, make(map[string]string))
	hcl := generator.GenerateAll()

	// Verify resource is generated
	if !strings.Contains(hcl, `resource "elementum_widget" "minimal_link"`) {
		t.Errorf("Expected widget resource, got:\n%s", hcl)
	}

	// Optional fields should NOT be present when not set
	if strings.Contains(hcl, `color`) {
		t.Error("Empty color should not be included")
	}
	if strings.Contains(hcl, `icon`) {
		t.Error("Empty icon should not be included")
	}
	if strings.Contains(hcl, `full_width`) {
		t.Error("Default full_width (false) should not be included")
	}
}

// ============================================================================
// Element Widget Generator Tests
// ============================================================================

func TestElementWidgetHCLGenerator_GenerateAll(t *testing.T) {
	element := &discovery.Element{
		ID:   "element-123",
		Name: "Test Element",
		Widgets: []discovery.Widget{
			{
				ID:       TestWidgetID,
				Name:     "Element Related Widget",
				Type:     "DisplayWidgetRelatedAspect",
				AspectID: TestTargetAspectID,
			},
		},
	}

	imports := []ImportBlock{
		{ID: "element-123", ResourceType: "elementum_element", ResourceName: "test_element"},
		{ID: "element-123:" + TestWidgetID, ResourceType: "elementum_widget", ResourceName: "element_related_widget"},
	}

	generator := NewElementWidgetHCLGenerator(element, imports, make(map[string]string))
	hcl := generator.GenerateAll()

	// Verify resource block
	if !strings.Contains(hcl, `resource "elementum_widget" "element_related_widget"`) {
		t.Errorf("Expected widget resource block, got:\n%s", hcl)
	}

	// Verify it references the element
	if !strings.Contains(hcl, `object_id = elementum_element.test_element.id`) {
		t.Error("Expected object_id to reference element")
	}
}

func TestElementWidgetHCLGenerator_EmptyElement(t *testing.T) {
	element := &discovery.Element{
		ID:      "element-123",
		Name:    "Test Element",
		Widgets: []discovery.Widget{},
	}

	generator := NewElementWidgetHCLGenerator(element, nil, make(map[string]string))
	hcl := generator.GenerateAll()

	if hcl != "" {
		t.Errorf("Expected empty output for element with no widgets, got: %s", hcl)
	}
}

func TestElementWidgetHCLGenerator_NilElement(t *testing.T) {
	generator := NewElementWidgetHCLGenerator(nil, nil, make(map[string]string))
	hcl := generator.GenerateAll()

	if hcl != "" {
		t.Errorf("Expected empty output for nil element, got: %s", hcl)
	}
}

// ============================================================================
// Namespace-First Naming Tests (Issue #9 Fix)
// ============================================================================
// These tests verify that widget parent references use namespace-first naming
// to match how discovered resources are actually named in imports.go

func TestWidgetHCLGenerator_MainAppUsesNamespace(t *testing.T) {
	// Main app has both Name (with spaces/special chars) and Namespace (clean identifier)
	app := &discovery.App{
		ID:        "app-123",
		Name:      "Cases - Dev", // Friendly name with special chars
		Namespace: "casesdev",    // Clean namespace identifier
		Widgets: []discovery.Widget{
			{
				ID:       TestWidgetID,
				Name:     "Related Widget",
				Type:     "DisplayWidgetRelatedAspect",
				AspectID: TestTargetAspectID,
			},
		},
	}

	imports := []ImportBlock{
		{ID: "app-123", ResourceType: "elementum_app", ResourceName: "casesdev"},
		{ID: "app-123:" + TestWidgetID, ResourceType: "elementum_widget", ResourceName: "related_widget"},
	}

	generator := NewWidgetHCLGenerator(app, imports, make(map[string]string))
	hcl := generator.GenerateAll()

	// Widget should reference elementum_app.casesdev (namespace), NOT elementum_app.cases___dev (sanitized name)
	if !strings.Contains(hcl, `object_id = elementum_app.casesdev.id`) {
		t.Errorf("Widget should reference app using namespace 'casesdev', got:\n%s", hcl)
	}
	// Ensure it does NOT use the friendly name style
	if strings.Contains(hcl, `elementum_app.cases___dev`) {
		t.Error("Widget should NOT use friendly name style 'cases___dev'")
	}
}

func TestWidgetHCLGenerator_MainAppFallsBackToNameWhenNoNamespace(t *testing.T) {
	// App has no namespace, should fall back to sanitized name
	app := &discovery.App{
		ID:        "app-123",
		Name:      "Test App",
		Namespace: "", // Empty namespace
		Widgets: []discovery.Widget{
			{
				ID:       TestWidgetID,
				Name:     "Related Widget",
				Type:     "DisplayWidgetRelatedAspect",
				AspectID: TestTargetAspectID,
			},
		},
	}

	imports := []ImportBlock{
		{ID: "app-123", ResourceType: "elementum_app", ResourceName: "test_app"},
		{ID: "app-123:" + TestWidgetID, ResourceType: "elementum_widget", ResourceName: "related_widget"},
	}

	generator := NewWidgetHCLGenerator(app, imports, make(map[string]string))
	hcl := generator.GenerateAll()

	// Should fall back to sanitized name when namespace is empty
	if !strings.Contains(hcl, `object_id = elementum_app.test_app.id`) {
		t.Errorf("Widget should fall back to sanitized name when namespace is empty, got:\n%s", hcl)
	}
}

func TestWidgetHCLGenerator_DiscoveredAppUsesNamespace(t *testing.T) {
	// This is the key test case from Issue #9:
	// Discovered app has Name="Cases - Dev" and Namespace="casesdev"
	// The resource is named "casesdev" (from namespace), but widget was using "cases___dev" (from name)
	discoveredApp := &discovery.App{
		ID:        "discovered-app-123",
		Name:      "Velocity Actions - Dev", // Friendly name with special chars
		Namespace: "velactionsdev",          // Clean namespace identifier
		Widgets: []discovery.Widget{
			{
				ID:       "discovered-widget-1",
				Name:     "Discovered Widget",
				Type:     "DisplayWidgetRelatedAspect",
				AspectID: TestTargetAspectID,
			},
		},
	}

	app := &discovery.App{
		ID:             "app-123",
		Name:           "Main App",
		Namespace:      "mainapp",
		Widgets:        []discovery.Widget{},
		DiscoveredApps: []*discovery.App{discoveredApp},
	}

	imports := []ImportBlock{
		{ID: "app-123", ResourceType: "elementum_app", ResourceName: "mainapp"},
		{ID: "discovered-app-123", ResourceType: "elementum_app", ResourceName: "velactionsdev"},
		{ID: "discovered-app-123:discovered-widget-1", ResourceType: "elementum_widget", ResourceName: "discovered_widget"},
	}

	generator := NewWidgetHCLGenerator(app, imports, make(map[string]string))
	hcl := generator.GenerateAll()

	// Widget should reference elementum_app.velactionsdev (namespace), NOT elementum_app.velocity_actions___dev
	if !strings.Contains(hcl, `object_id = elementum_app.velactionsdev.id`) {
		t.Errorf("Widget should reference discovered app using namespace 'velactionsdev', got:\n%s", hcl)
	}
	// Ensure it does NOT use the friendly name style
	if strings.Contains(hcl, `velocity_actions`) {
		t.Error("Widget should NOT use friendly name style derived from 'Velocity Actions - Dev'")
	}
}

func TestWidgetHCLGenerator_DiscoveredElementUsesNamespace(t *testing.T) {
	// Element has namespace priority: Namespace > Handle > Name
	discoveredElement := &discovery.Element{
		ID:        "discovered-element-123",
		Name:      "Action Components - Dev", // Friendly name
		Handle:    "actioncomponents",        // Handle
		Namespace: "actioncomponentsdev",     // Namespace (highest priority)
		Widgets: []discovery.Widget{
			{
				ID:         "element-widget-1",
				Name:       "Element Widget",
				Type:       "DisplayWidgetRelatedCreateAction",
				AspectID:   TestTargetAspectID,
				ButtonType: "PRIMARY",
			},
		},
	}

	app := &discovery.App{
		ID:                 "app-123",
		Name:               "Main App",
		Namespace:          "mainapp",
		Widgets:            []discovery.Widget{},
		DiscoveredElements: []*discovery.Element{discoveredElement},
	}

	imports := []ImportBlock{
		{ID: "app-123", ResourceType: "elementum_app", ResourceName: "mainapp"},
		{ID: "discovered-element-123", ResourceType: "elementum_element", ResourceName: "actioncomponentsdev"},
		{ID: "discovered-element-123:element-widget-1", ResourceType: "elementum_widget", ResourceName: "element_widget"},
	}

	generator := NewWidgetHCLGenerator(app, imports, make(map[string]string))
	hcl := generator.GenerateAll()

	// Widget should reference elementum_element.actioncomponentsdev (namespace)
	if !strings.Contains(hcl, `object_id = elementum_element.actioncomponentsdev.id`) {
		t.Errorf("Widget should reference element using namespace 'actioncomponentsdev', got:\n%s", hcl)
	}
	// Ensure it does NOT use the friendly name style
	if strings.Contains(hcl, `action_components`) {
		t.Error("Widget should NOT use friendly name style derived from 'Action Components - Dev'")
	}
}

func TestWidgetHCLGenerator_DiscoveredElementUsesHandleWhenNoNamespace(t *testing.T) {
	// Element without namespace should fall back to Handle
	discoveredElement := &discovery.Element{
		ID:        "discovered-element-123",
		Name:      "Test Element Name",
		Handle:    "testelementhandle",
		Namespace: "", // Empty namespace
		Widgets: []discovery.Widget{
			{
				ID:       "element-widget-1",
				Name:     "Element Widget",
				Type:     "DisplayWidgetRelatedAspect",
				AspectID: TestTargetAspectID,
			},
		},
	}

	app := &discovery.App{
		ID:                 "app-123",
		Name:               "Main App",
		Namespace:          "mainapp",
		Widgets:            []discovery.Widget{},
		DiscoveredElements: []*discovery.Element{discoveredElement},
	}

	imports := []ImportBlock{
		{ID: "app-123", ResourceType: "elementum_app", ResourceName: "mainapp"},
		{ID: "discovered-element-123", ResourceType: "elementum_element", ResourceName: "testelementhandle"},
		{ID: "discovered-element-123:element-widget-1", ResourceType: "elementum_widget", ResourceName: "element_widget"},
	}

	generator := NewWidgetHCLGenerator(app, imports, make(map[string]string))
	hcl := generator.GenerateAll()

	// Should use Handle when namespace is empty
	if !strings.Contains(hcl, `object_id = elementum_element.testelementhandle.id`) {
		t.Errorf("Widget should use element handle when namespace is empty, got:\n%s", hcl)
	}
}

func TestWidgetHCLGenerator_DiscoveredElementUsesNameWhenNoNamespaceOrHandle(t *testing.T) {
	// Element without namespace or handle should fall back to Name
	discoveredElement := &discovery.Element{
		ID:        "discovered-element-123",
		Name:      "My Element Name",
		Handle:    "", // Empty handle
		Namespace: "", // Empty namespace
		Widgets: []discovery.Widget{
			{
				ID:       "element-widget-1",
				Name:     "Element Widget",
				Type:     "DisplayWidgetRelatedAspect",
				AspectID: TestTargetAspectID,
			},
		},
	}

	app := &discovery.App{
		ID:                 "app-123",
		Name:               "Main App",
		Namespace:          "mainapp",
		Widgets:            []discovery.Widget{},
		DiscoveredElements: []*discovery.Element{discoveredElement},
	}

	imports := []ImportBlock{
		{ID: "app-123", ResourceType: "elementum_app", ResourceName: "mainapp"},
		{ID: "discovered-element-123", ResourceType: "elementum_element", ResourceName: "my_element_name"},
		{ID: "discovered-element-123:element-widget-1", ResourceType: "elementum_widget", ResourceName: "element_widget"},
	}

	generator := NewWidgetHCLGenerator(app, imports, make(map[string]string))
	hcl := generator.GenerateAll()

	// Should use sanitized Name when namespace and handle are empty
	if !strings.Contains(hcl, `object_id = elementum_element.my_element_name.id`) {
		t.Errorf("Widget should use sanitized element name when namespace and handle are empty, got:\n%s", hcl)
	}
}

func TestWidgetHCLGenerator_DiscoveredTaskUsesNamespace(t *testing.T) {
	// Task has namespace priority: Namespace > Handle > Name
	discoveredTask := &discovery.AspectTask{
		ID:        "discovered-task-123",
		Name:      "Velocity Activity Tasks", // Friendly name
		Handle:    "velactivitytasks",        // Handle
		Namespace: "velocityactivitytasks",   // Namespace (highest priority)
		Widgets: []discovery.Widget{
			{
				ID:         "task-widget-1",
				Name:       "Task Widget",
				Type:       "DisplayWidgetRelatedLinkAction",
				AspectID:   TestTargetAspectID,
				ButtonType: "INLINE",
			},
		},
	}

	app := &discovery.App{
		ID:              "app-123",
		Name:            "Main App",
		Namespace:       "mainapp",
		Widgets:         []discovery.Widget{},
		DiscoveredTasks: []*discovery.AspectTask{discoveredTask},
	}

	imports := []ImportBlock{
		{ID: "app-123", ResourceType: "elementum_app", ResourceName: "mainapp"},
		{ID: "discovered-task-123", ResourceType: "elementum_task", ResourceName: "velocityactivitytasks"},
		{ID: "discovered-task-123:task-widget-1", ResourceType: "elementum_widget", ResourceName: "task_widget"},
	}

	generator := NewWidgetHCLGenerator(app, imports, make(map[string]string))
	hcl := generator.GenerateAll()

	// Widget should reference elementum_task.velocityactivitytasks (namespace)
	if !strings.Contains(hcl, `object_id = elementum_task.velocityactivitytasks.id`) {
		t.Errorf("Widget should reference task using namespace 'velocityactivitytasks', got:\n%s", hcl)
	}
	// Ensure it does NOT use the friendly name style
	if strings.Contains(hcl, `velocity_activity_tasks`) {
		t.Error("Widget should NOT use friendly name style derived from 'Velocity Activity Tasks'")
	}
}

func TestWidgetHCLGenerator_DiscoveredTaskUsesHandleWhenNoNamespace(t *testing.T) {
	// Task without namespace should fall back to Handle
	discoveredTask := &discovery.AspectTask{
		ID:        "discovered-task-123",
		Name:      "Test Task Name",
		Handle:    "testtaskhandle",
		Namespace: "", // Empty namespace
		Widgets: []discovery.Widget{
			{
				ID:       "task-widget-1",
				Name:     "Task Widget",
				Type:     "DisplayWidgetRelatedAspect",
				AspectID: TestTargetAspectID,
			},
		},
	}

	app := &discovery.App{
		ID:              "app-123",
		Name:            "Main App",
		Namespace:       "mainapp",
		Widgets:         []discovery.Widget{},
		DiscoveredTasks: []*discovery.AspectTask{discoveredTask},
	}

	imports := []ImportBlock{
		{ID: "app-123", ResourceType: "elementum_app", ResourceName: "mainapp"},
		{ID: "discovered-task-123", ResourceType: "elementum_task", ResourceName: "testtaskhandle"},
		{ID: "discovered-task-123:task-widget-1", ResourceType: "elementum_widget", ResourceName: "task_widget"},
	}

	generator := NewWidgetHCLGenerator(app, imports, make(map[string]string))
	hcl := generator.GenerateAll()

	// Should use Handle when namespace is empty
	if !strings.Contains(hcl, `object_id = elementum_task.testtaskhandle.id`) {
		t.Errorf("Widget should use task handle when namespace is empty, got:\n%s", hcl)
	}
}

func TestWidgetHCLGenerator_DiscoveredTaskUsesNameWhenNoNamespaceOrHandle(t *testing.T) {
	// Task without namespace or handle should fall back to Name
	discoveredTask := &discovery.AspectTask{
		ID:        "discovered-task-123",
		Name:      "My Task Name",
		Handle:    "", // Empty handle
		Namespace: "", // Empty namespace
		Widgets: []discovery.Widget{
			{
				ID:       "task-widget-1",
				Name:     "Task Widget",
				Type:     "DisplayWidgetRelatedAspect",
				AspectID: TestTargetAspectID,
			},
		},
	}

	app := &discovery.App{
		ID:              "app-123",
		Name:            "Main App",
		Namespace:       "mainapp",
		Widgets:         []discovery.Widget{},
		DiscoveredTasks: []*discovery.AspectTask{discoveredTask},
	}

	imports := []ImportBlock{
		{ID: "app-123", ResourceType: "elementum_app", ResourceName: "mainapp"},
		{ID: "discovered-task-123", ResourceType: "elementum_task", ResourceName: "my_task_name"},
		{ID: "discovered-task-123:task-widget-1", ResourceType: "elementum_widget", ResourceName: "task_widget"},
	}

	generator := NewWidgetHCLGenerator(app, imports, make(map[string]string))
	hcl := generator.GenerateAll()

	// Should use sanitized Name when namespace and handle are empty
	if !strings.Contains(hcl, `object_id = elementum_task.my_task_name.id`) {
		t.Errorf("Widget should use sanitized task name when namespace and handle are empty, got:\n%s", hcl)
	}
}

// ============================================================================
// UUID Map Beautification Tests
// ============================================================================

// TestWidgetHCLGenerator_BeautifiesNestedObjectID tests that nested object_id
// values are beautified when the UUID is found in uuidMap
func TestWidgetHCLGenerator_BeautifiesNestedObjectID(t *testing.T) {
	targetAspectID := "external-app-uuid-123"
	app := &discovery.App{
		ID:   "app-123",
		Name: "Test App",
		Widgets: []discovery.Widget{
			{
				ID:       "widget-1",
				Name:     "Related Records",
				Type:     "DisplayWidgetRelatedAspect",
				AspectID: targetAspectID,
				Columns:  []string{"col1"},
				Rows:     25,
			},
		},
	}

	imports := []ImportBlock{
		{ID: "app-123", ResourceType: "elementum_app", ResourceName: "test_app"},
		{ID: "app-123:widget-1", ResourceType: "elementum_widget", ResourceName: "related_records"},
	}

	// uuidMap contains a mapping for the target AspectID
	uuidMap := map[string]string{
		targetAspectID: "data.elementum_app.external_app.id",
	}

	generator := NewWidgetHCLGenerator(app, imports, uuidMap)
	hcl := generator.GenerateAll()

	// The nested object_id should be beautified to a reference
	if !strings.Contains(hcl, `object_id = data.elementum_app.external_app.id`) {
		t.Errorf("Expected nested object_id to be beautified, got:\n%s", hcl)
	}

	// Should NOT contain the raw UUID
	if strings.Contains(hcl, targetAspectID) {
		t.Errorf("Expected raw UUID %q to be replaced, got:\n%s", targetAspectID, hcl)
	}
}

// TestWidgetHCLGenerator_BeautifiesNestedObjectID_LinkAction tests beautification
// for related_link_action widgets
func TestWidgetHCLGenerator_BeautifiesNestedObjectID_LinkAction(t *testing.T) {
	targetAspectID := "external-element-uuid-456"
	app := &discovery.App{
		ID:   "app-123",
		Name: "Test App",
		Widgets: []discovery.Widget{
			{
				ID:         "widget-1",
				Name:       "Link to Element",
				Type:       "DisplayWidgetRelatedLinkAction",
				AspectID:   targetAspectID,
				ButtonType: "PRIMARY",
			},
		},
	}

	imports := []ImportBlock{
		{ID: "app-123", ResourceType: "elementum_app", ResourceName: "test_app"},
		{ID: "app-123:widget-1", ResourceType: "elementum_widget", ResourceName: "link_widget"},
	}

	// uuidMap contains a mapping for the target AspectID
	uuidMap := map[string]string{
		targetAspectID: "elementum_element.external_element.id",
	}

	generator := NewWidgetHCLGenerator(app, imports, uuidMap)
	hcl := generator.GenerateAll()

	// The nested object_id should be beautified
	if !strings.Contains(hcl, `object_id = elementum_element.external_element.id`) {
		t.Errorf("Expected nested object_id in related_link_action to be beautified, got:\n%s", hcl)
	}

	// Should NOT contain the raw UUID
	if strings.Contains(hcl, targetAspectID) {
		t.Errorf("Expected raw UUID %q to be replaced, got:\n%s", targetAspectID, hcl)
	}
}

// TestWidgetHCLGenerator_BeautifiesNestedObjectID_CreateAction tests beautification
// for related_create_action widgets
func TestWidgetHCLGenerator_BeautifiesNestedObjectID_CreateAction(t *testing.T) {
	targetAspectID := "external-task-uuid-789"
	app := &discovery.App{
		ID:   "app-123",
		Name: "Test App",
		Widgets: []discovery.Widget{
			{
				ID:         "widget-1",
				Name:       "Create Record",
				Type:       "DisplayWidgetRelatedCreateAction",
				AspectID:   targetAspectID,
				ButtonType: "SECONDARY",
			},
		},
	}

	imports := []ImportBlock{
		{ID: "app-123", ResourceType: "elementum_app", ResourceName: "test_app"},
		{ID: "app-123:widget-1", ResourceType: "elementum_widget", ResourceName: "create_widget"},
	}

	// uuidMap contains a mapping for the target AspectID
	uuidMap := map[string]string{
		targetAspectID: "elementum_task.external_task.id",
	}

	generator := NewWidgetHCLGenerator(app, imports, uuidMap)
	hcl := generator.GenerateAll()

	// The nested object_id should be beautified
	if !strings.Contains(hcl, `object_id = elementum_task.external_task.id`) {
		t.Errorf("Expected nested object_id in related_create_action to be beautified, got:\n%s", hcl)
	}

	// Should NOT contain the raw UUID
	if strings.Contains(hcl, targetAspectID) {
		t.Errorf("Expected raw UUID %q to be replaced, got:\n%s", targetAspectID, hcl)
	}
}

// TestWidgetHCLGenerator_PreservesRawUUID_NotInMap tests that raw UUIDs
// are preserved when not found in uuidMap
func TestWidgetHCLGenerator_PreservesRawUUID_NotInMap(t *testing.T) {
	targetAspectID := "unknown-uuid-not-in-map"
	app := &discovery.App{
		ID:   "app-123",
		Name: "Test App",
		Widgets: []discovery.Widget{
			{
				ID:       "widget-1",
				Name:     "Related Records",
				Type:     "DisplayWidgetRelatedAspect",
				AspectID: targetAspectID, // Not in uuidMap
			},
		},
	}

	imports := []ImportBlock{
		{ID: "app-123", ResourceType: "elementum_app", ResourceName: "test_app"},
		{ID: "app-123:widget-1", ResourceType: "elementum_widget", ResourceName: "related_records"},
	}

	// Empty uuidMap - no mappings
	uuidMap := map[string]string{}

	generator := NewWidgetHCLGenerator(app, imports, uuidMap)
	hcl := generator.GenerateAll()

	// The nested object_id should be the raw UUID (quoted)
	if !strings.Contains(hcl, `object_id = "`+targetAspectID+`"`) {
		t.Errorf("Expected nested object_id to be raw UUID when not in map, got:\n%s", hcl)
	}
}

// TestWidgetHCLGenerator_MultipleWidgets_MixedBeautification tests that
// widgets with and without mappings are handled correctly together
func TestWidgetHCLGenerator_MultipleWidgets_MixedBeautification(t *testing.T) {
	knownUUID := "known-uuid-123"
	unknownUUID := "unknown-uuid-456"

	app := &discovery.App{
		ID:   "app-123",
		Name: "Test App",
		Widgets: []discovery.Widget{
			{
				ID:       "widget-1",
				Name:     "Known Widget",
				Type:     "DisplayWidgetRelatedAspect",
				AspectID: knownUUID, // In uuidMap
			},
			{
				ID:       "widget-2",
				Name:     "Unknown Widget",
				Type:     "DisplayWidgetRelatedAspect",
				AspectID: unknownUUID, // Not in uuidMap
			},
		},
	}

	imports := []ImportBlock{
		{ID: "app-123", ResourceType: "elementum_app", ResourceName: "test_app"},
		{ID: "app-123:widget-1", ResourceType: "elementum_widget", ResourceName: "known_widget"},
		{ID: "app-123:widget-2", ResourceType: "elementum_widget", ResourceName: "unknown_widget"},
	}

	uuidMap := map[string]string{
		knownUUID: "data.elementum_app.known_app.id",
	}

	generator := NewWidgetHCLGenerator(app, imports, uuidMap)
	hcl := generator.GenerateAll()

	// First widget should be beautified
	if !strings.Contains(hcl, `object_id = data.elementum_app.known_app.id`) {
		t.Errorf("Expected first widget's object_id to be beautified, got:\n%s", hcl)
	}

	// Second widget should have raw UUID
	if !strings.Contains(hcl, `object_id = "`+unknownUUID+`"`) {
		t.Errorf("Expected second widget's object_id to be raw UUID, got:\n%s", hcl)
	}
}

// TestWidgetHCLGenerator_DiscoveredAppWidget_Beautification tests beautification
// for widgets on discovered apps
func TestWidgetHCLGenerator_DiscoveredAppWidget_Beautification(t *testing.T) {
	targetAspectID := "external-from-discovered"

	discoveredApp := &discovery.App{
		ID:        "discovered-app-123",
		Name:      "Discovered App",
		Namespace: "discoveredapp",
		Widgets: []discovery.Widget{
			{
				ID:       "discovered-widget-1",
				Name:     "Widget on Discovered",
				Type:     "DisplayWidgetRelatedAspect",
				AspectID: targetAspectID,
			},
		},
	}

	app := &discovery.App{
		ID:             "app-123",
		Name:           "Main App",
		Namespace:      "mainapp",
		Widgets:        []discovery.Widget{},
		DiscoveredApps: []*discovery.App{discoveredApp},
	}

	imports := []ImportBlock{
		{ID: "app-123", ResourceType: "elementum_app", ResourceName: "mainapp"},
		{ID: "discovered-app-123", ResourceType: "elementum_app", ResourceName: "discoveredapp"},
		{ID: "discovered-app-123:discovered-widget-1", ResourceType: "elementum_widget", ResourceName: "discovered_widget"},
	}

	uuidMap := map[string]string{
		targetAspectID: "data.elementum_element.external_element.id",
	}

	generator := NewWidgetHCLGenerator(app, imports, uuidMap)
	hcl := generator.GenerateAll()

	// The nested object_id should be beautified
	if !strings.Contains(hcl, `object_id = data.elementum_element.external_element.id`) {
		t.Errorf("Expected discovered app widget's object_id to be beautified, got:\n%s", hcl)
	}
}

// TestWidgetHCLGenerator_EmptyAspectID_NoBeautification tests that widgets
// with empty AspectID don't cause issues
func TestWidgetHCLGenerator_EmptyAspectID_HandledGracefully(t *testing.T) {
	app := &discovery.App{
		ID:   "app-123",
		Name: "Test App",
		Widgets: []discovery.Widget{
			{
				ID:       "widget-1",
				Name:     "Widget with Empty AspectID",
				Type:     "DisplayWidgetRelatedAspect",
				AspectID: "", // Empty
			},
		},
	}

	imports := []ImportBlock{
		{ID: "app-123", ResourceType: "elementum_app", ResourceName: "test_app"},
		{ID: "app-123:widget-1", ResourceType: "elementum_widget", ResourceName: "widget_empty"},
	}

	uuidMap := map[string]string{}

	generator := NewWidgetHCLGenerator(app, imports, uuidMap)
	hcl := generator.GenerateAll()

	// Should generate output with empty string (the widget still gets created)
	if !strings.Contains(hcl, `resource "elementum_widget" "widget_empty"`) {
		t.Errorf("Expected widget resource to be generated, got:\n%s", hcl)
	}

	// object_id should be empty string
	if !strings.Contains(hcl, `object_id = ""`) {
		t.Errorf("Expected empty object_id for widget with empty AspectID, got:\n%s", hcl)
	}
}
