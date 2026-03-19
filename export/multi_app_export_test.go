// Copyright (c) ElementumAI, Inc.
// SPDX-License-Identifier: MPL-2.0

package export

import (
	"strings"
	"testing"

	"github.com/elementumltd/elementum-cli/discovery"
)

// TestBeautifyDiscoveredAppReferences verifies UUID mapping for discovered apps
func TestBeautifyDiscoveredAppReferences(t *testing.T) {
	app := &discovery.App{
		ID:        "main-app-id",
		Name:      "Main App",
		Namespace: "mainapp",
		DiscoveredApps: []*discovery.App{
			{
				ID:        "disc-app-uuid-123",
				Name:      "Discovered App",
				Namespace: "discoveredapp",
			},
		},
	}

	// Build imports that include the discovered app
	imports := []ImportBlock{
		{ResourceType: "elementum_app", ResourceName: "mainapp", ID: "main-app-id"},
		{ResourceType: "elementum_app", ResourceName: "discoveredapp", ID: "disc-app-uuid-123"},
	}

	// Build UUID map using the actual buildUUIDMap function
	uuidMap := buildUUIDMap(imports, app)

	// Verify the mapping is to a resource, not a data source
	expected := "elementum_app.discoveredapp.id"
	if uuidMap["disc-app-uuid-123"] != expected {
		t.Errorf("Expected discovered app to map to resource %q, got %q", expected, uuidMap["disc-app-uuid-123"])
	}

	// Verify it does NOT map to a data source
	if strings.Contains(uuidMap["disc-app-uuid-123"], "data.") {
		t.Error("Discovered app should NOT map to data source")
	}
}

// TestDiscoveredAppAlwaysReferencedAsResource verifies that discovered apps are ALWAYS
// referenced as resources, even when no import block exists (fallback behavior).
// This is the key fix: discovered apps should never be referenced as data sources.
func TestDiscoveredAppAlwaysReferencedAsResource(t *testing.T) {
	app := &discovery.App{
		ID:        "main-app-id",
		Name:      "Main App",
		Namespace: "mainapp",
		DiscoveredApps: []*discovery.App{
			{
				ID:        "disc-app-no-import",
				Name:      "Discovered App Without Import",
				Namespace: "discoveredappnoimport",
			},
		},
	}

	// Intentionally DO NOT include an import block for the discovered app
	// This tests the fallback behavior
	imports := []ImportBlock{
		{ResourceType: "elementum_app", ResourceName: "mainapp", ID: "main-app-id"},
		// Note: No import for disc-app-no-import
	}

	// Build UUID map
	uuidMap := buildUUIDMap(imports, app)

	// Verify the mapping is still to a RESOURCE (using namespace as fallback), NOT a data source
	// The fallback uses SanitizeName(Namespace) which gives "discoveredappnoimport"
	expected := "elementum_app.discoveredappnoimport.id"
	if uuidMap["disc-app-no-import"] != expected {
		t.Errorf("Expected discovered app without import to map to resource %q, got %q", expected, uuidMap["disc-app-no-import"])
	}

	// Critical: Verify it does NOT map to a data source
	if strings.Contains(uuidMap["disc-app-no-import"], "data.") {
		t.Error("Discovered app should NEVER map to data source, even without import block")
	}
}

// TestDiscoveredAppFallbackUsesNameWhenNoNamespace verifies that when a discovered app
// has no namespace, the fallback uses the sanitized Name for the resource reference.
func TestDiscoveredAppFallbackUsesNameWhenNoNamespace(t *testing.T) {
	app := &discovery.App{
		ID:        "main-app-id",
		Name:      "Main App",
		Namespace: "mainapp",
		DiscoveredApps: []*discovery.App{
			{
				ID:        "disc-app-no-namespace",
				Name:      "App With Spaces",
				Namespace: "", // Empty namespace - should fall back to sanitized Name
			},
		},
	}

	// No import block for the discovered app
	imports := []ImportBlock{
		{ResourceType: "elementum_app", ResourceName: "mainapp", ID: "main-app-id"},
	}

	uuidMap := buildUUIDMap(imports, app)

	// Should use sanitized Name: "App With Spaces" -> "app_with_spaces"
	expected := "elementum_app.app_with_spaces.id"
	if uuidMap["disc-app-no-namespace"] != expected {
		t.Errorf("Expected discovered app to use sanitized name as fallback, got %q, want %q", uuidMap["disc-app-no-namespace"], expected)
	}

	// Verify it's NOT a data source
	if strings.Contains(uuidMap["disc-app-no-namespace"], "data.") {
		t.Error("Discovered app should NEVER map to data source")
	}
}

// TestDiscoveredAppMixedImportScenarios verifies correct behavior when some discovered apps
// have import blocks and others don't - all should still reference as resources.
func TestDiscoveredAppMixedImportScenarios(t *testing.T) {
	app := &discovery.App{
		ID:        "main-app-id",
		Name:      "Main App",
		Namespace: "mainapp",
		DiscoveredApps: []*discovery.App{
			{
				ID:        "disc-app-with-import",
				Name:      "Has Import Block",
				Namespace: "hasimport",
			},
			{
				ID:        "disc-app-without-import",
				Name:      "No Import Block",
				Namespace: "noimport",
			},
			{
				ID:        "disc-app-custom-name",
				Name:      "Custom Named Import",
				Namespace: "customnamed",
			},
		},
	}

	// Only some discovered apps have import blocks
	imports := []ImportBlock{
		{ResourceType: "elementum_app", ResourceName: "mainapp", ID: "main-app-id"},
		{ResourceType: "elementum_app", ResourceName: "hasimport", ID: "disc-app-with-import"},
		// Note: disc-app-without-import has NO import block
		{ResourceType: "elementum_app", ResourceName: "my_custom_name", ID: "disc-app-custom-name"}, // Custom resource name
	}

	uuidMap := buildUUIDMap(imports, app)

	tests := []struct {
		name     string
		appID    string
		expected string
	}{
		{
			name:     "app with import block uses import resource name",
			appID:    "disc-app-with-import",
			expected: "elementum_app.hasimport.id",
		},
		{
			name:     "app without import block uses namespace fallback",
			appID:    "disc-app-without-import",
			expected: "elementum_app.noimport.id",
		},
		{
			name:     "app with custom import name uses that name",
			appID:    "disc-app-custom-name",
			expected: "elementum_app.my_custom_name.id",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := uuidMap[tt.appID]
			if got != tt.expected {
				t.Errorf("got %q, want %q", got, tt.expected)
			}
			// All should be resources, never data sources
			if strings.Contains(got, "data.") {
				t.Errorf("discovered app %s should NOT map to data source", tt.appID)
			}
		})
	}
}

// TestGenerateRelationshipDataSourcesSkipsDiscoveredApps verifies that discovered apps
// are not generated as data sources since they're exported as resources
func TestGenerateRelationshipDataSourcesSkipsDiscoveredApps(t *testing.T) {
	app := &discovery.App{
		ID:        "main-app-id",
		Name:      "Main App",
		Namespace: "mainapp",
		RelatedObjects: []discovery.RelatedObject{
			{ID: "related-app-id", Name: "Related App", Type: "App"},
		},
		DiscoveredApps: []*discovery.App{
			{ID: "discovered-app-id", Name: "Discovered App", Namespace: "discoveredapp"},
		},
	}

	// Generate relationship data sources
	result := GenerateRelationshipDataSources(app)

	// Should contain data source for related app (it's in RelatedObjects)
	if !strings.Contains(result, `data "elementum_app" "related_app"`) {
		t.Error("Expected data source for related app")
	}

	// Should NOT contain data source for discovered app (it's exported as resource)
	if strings.Contains(result, "discoveredapp") {
		t.Error("Should not generate data source for discovered app - it's exported as a resource")
	}
}
