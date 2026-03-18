// Copyright (c) ElementumAI, Inc.
// SPDX-License-Identifier: MPL-2.0

package export

import (
	"strings"
	"testing"

	"github.com/elementumltd/elementum-cli/discovery"
)

func TestAccessPolicyHCLGenerator_GenerateAll(t *testing.T) {
	app := &discovery.App{
		ID:        "app-123",
		Name:      "Test App",
		Namespace: "testapp",
		AccessPolicies: []discovery.AccessPolicy{
			{
				ID:       "policy-1",
				ObjectID: "app-123",
				Filter: map[string]interface{}{
					"type":    "EQUALS",
					"fieldId": "field-status-123",
					"value": map[string]interface{}{
						"literal": "active",
					},
				},
				UserIDs:    []string{"user-1", "user-2"},
				GroupIDs:   []string{"group-1"},
				UserEmails: map[string]string{"user-1": "alice@example.com", "user-2": "bob@example.com"},
				GroupNames: map[string]string{"group-1": "Admins"},
			},
		},
	}

	imports := []ImportBlock{
		{ResourceType: "elementum_app", ResourceName: "testapp", ID: "app-123"},
		{ResourceType: "elementum_access_policy", ResourceName: "testapp_policy_1", ID: "app-123:policy-1"},
	}

	uuidMap := make(map[string]string)
	gen := NewAccessPolicyHCLGenerator(app, imports, uuidMap)
	hcl := gen.GenerateAll()

	// Verify user data source (SanitizeName removes @ so alice@example.com becomes aliceexample_com)
	if !strings.Contains(hcl, `data "elementum_user"`) {
		t.Error("Generated HCL should contain user data source")
	}

	// Verify group data source
	if !strings.Contains(hcl, `data "elementum_group" "admins"`) {
		t.Error("Generated HCL should contain group data source for Admins")
	}

	// Verify access policy resource
	if !strings.Contains(hcl, `resource "elementum_access_policy" "testapp_policy_1"`) {
		t.Error("Generated HCL should contain access policy resource")
	}

	// Verify object_id references app by namespace (not display name)
	if !strings.Contains(hcl, "object_id = elementum_app.testapp.id") {
		t.Error("Generated HCL should reference app for object_id using namespace")
	}
}

func TestAccessPolicyHCLGenerator_NoAccessPolicies(t *testing.T) {
	app := &discovery.App{
		ID:             "app-123",
		Name:           "Test App",
		Namespace:      "testapp",
		AccessPolicies: []discovery.AccessPolicy{},
	}

	imports := []ImportBlock{}
	uuidMap := make(map[string]string)
	gen := NewAccessPolicyHCLGenerator(app, imports, uuidMap)
	hcl := gen.GenerateAll()

	if hcl != "" {
		t.Errorf("GenerateAll() should return empty string for app with no access policies, got:\n%s", hcl)
	}
}

func TestAccessPolicyHCLGenerator_NilApp(t *testing.T) {
	imports := []ImportBlock{}
	uuidMap := make(map[string]string)
	gen := &AccessPolicyHCLGenerator{
		app:     nil,
		imports: imports,
		uuidMap: uuidMap,
	}
	hcl := gen.GenerateAll()

	if hcl != "" {
		t.Errorf("GenerateAll() should return empty string for nil app, got:\n%s", hcl)
	}
}

func TestGenerateAccessPolicyHCLStandalone(t *testing.T) {
	app := &discovery.App{
		ID:        "app-123",
		Name:      "Test App",
		Namespace: "testapp",
		AccessPolicies: []discovery.AccessPolicy{
			{
				ID:         "policy-1",
				ObjectID:   "app-123",
				UserIDs:    []string{"user-1"},
				GroupIDs:   []string{},
				UserEmails: map[string]string{"user-1": "test@example.com"},
				GroupNames: make(map[string]string),
			},
		},
	}

	imports := []ImportBlock{
		{ResourceType: "elementum_app", ResourceName: "test_app", ID: "app-123"},
	}

	hcl := GenerateAccessPolicyHCLStandalone(app, imports)

	if !strings.Contains(hcl, "elementum_access_policy") {
		t.Error("Standalone generator should produce access policy HCL")
	}
}

func TestGenerateAccessPolicyImports(t *testing.T) {
	app := &discovery.App{
		ID:        "app-123",
		Name:      "Test App",
		Namespace: "testapp",
		AccessPolicies: []discovery.AccessPolicy{
			{ID: "policy-1", ObjectID: "app-123"},
			{ID: "policy-2", ObjectID: "app-123"},
			{ID: "policy-3", ObjectID: "app-123"},
		},
	}

	imports := GenerateAccessPolicyImports(app)

	if len(imports) != 3 {
		t.Errorf("GenerateAccessPolicyImports() returned %d imports, want 3", len(imports))
	}

	// Verify first import
	if imports[0].ResourceType != "elementum_access_policy" {
		t.Errorf("Import[0].ResourceType = %q, want elementum_access_policy", imports[0].ResourceType)
	}
	if imports[0].ResourceName != "testapp_policy_1" {
		t.Errorf("Import[0].ResourceName = %q, want testapp_policy_1", imports[0].ResourceName)
	}
	if imports[0].ID != "app-123:policy-1" {
		t.Errorf("Import[0].ID = %q, want app-123:policy-1", imports[0].ID)
	}

	// Verify third import has correct index
	if imports[2].ResourceName != "testapp_policy_3" {
		t.Errorf("Import[2].ResourceName = %q, want testapp_policy_3", imports[2].ResourceName)
	}
}

func TestGenerateAccessPolicyImports_NilApp(t *testing.T) {
	imports := GenerateAccessPolicyImports(nil)

	if imports != nil {
		t.Error("GenerateAccessPolicyImports(nil) should return nil")
	}
}

func TestGenerateAccessPolicyImports_NoAccessPolicies(t *testing.T) {
	app := &discovery.App{
		ID:             "app-123",
		Name:           "Test App",
		Namespace:      "testapp",
		AccessPolicies: []discovery.AccessPolicy{},
	}

	imports := GenerateAccessPolicyImports(app)

	if imports != nil {
		t.Error("GenerateAccessPolicyImports() should return nil for app with no access policies")
	}
}
