// Copyright (c) ElementumAI, Inc.
// SPDX-License-Identifier: MPL-2.0

package export

import (
	"strings"
	"testing"

	"github.com/elementumltd/elementum-cli/discovery"
)

func TestRoleHCLGenerator_GenerateAll(t *testing.T) {
	app := &discovery.App{
		ID:        "app-123",
		Name:      "Test App",
		Namespace: "testapp",
		Roles: []discovery.Role{
			{
				ID:          "role-1",
				Name:        "Admin",
				Description: "System administrator role",
				Managed:     true, // Should be skipped
				Tags:        []string{"system"},
			},
			{
				ID:          "role-2",
				Name:        "Custom Role",
				Description: "A custom role for the app",
				Managed:     false,
				AutoShare:   []string{"ASSIGNEE", "MENTION"},
				Users: []discovery.RoleMember{
					{ID: "user-1", Name: "alice@example.com"},
					{ID: "user-2", Name: "bob@example.com"},
				},
				Groups: []discovery.RoleMember{
					{ID: "group-1", Name: "Engineering"},
				},
				Permissions: []discovery.RolePermission{
					{Group: "RECORDS", Level: "ADMIN"},
					{Group: "AUTOMATIONS", Level: "EDIT"},
				},
			},
		},
	}

	imports := []ImportBlock{
		{ResourceType: "elementum_app", ResourceName: "test_app", ID: "app-123"},
		{ResourceType: "elementum_role", ResourceName: "custom_role", ID: "app-123:role-2"},
	}

	uuidMap := make(map[string]string)
	gen := NewRoleHCLGenerator(app, imports, uuidMap, "elementum_app.test_app.id")
	hcl := gen.GenerateAll()

	// Verify managed role is NOT in output (it's a data source, not a resource)
	if strings.Contains(hcl, `resource "elementum_role" "admin"`) {
		t.Error("Generated HCL should NOT contain managed role as resource")
	}

	// Verify custom role IS in output
	if !strings.Contains(hcl, `resource "elementum_role" "custom_role"`) {
		t.Errorf("Generated HCL should contain custom role resource, got:\n%s", hcl)
	}

	// Verify object_id references app
	if !strings.Contains(hcl, "object_id = elementum_app.test_app.id") {
		t.Errorf("Generated HCL should reference app for object_id, got:\n%s", hcl)
	}

	// Verify name
	if !strings.Contains(hcl, `name = "Custom Role"`) {
		t.Errorf("Generated HCL should contain role name, got:\n%s", hcl)
	}

	// Verify description
	if !strings.Contains(hcl, `description = "A custom role for the app"`) {
		t.Errorf("Generated HCL should contain description, got:\n%s", hcl)
	}

	// Verify auto_share
	if !strings.Contains(hcl, `auto_share = ["ASSIGNEE", "MENTION"]`) {
		t.Errorf("Generated HCL should contain auto_share, got:\n%s", hcl)
	}
}

func TestRoleHCLGenerator_NoRoles(t *testing.T) {
	app := &discovery.App{
		ID:        "app-123",
		Name:      "Test App",
		Namespace: "testapp",
		Roles:     []discovery.Role{},
	}

	imports := []ImportBlock{}
	uuidMap := make(map[string]string)
	gen := NewRoleHCLGenerator(app, imports, uuidMap, "elementum_app.test_app.id")
	hcl := gen.GenerateAll()

	if hcl != "" {
		t.Errorf("GenerateAll() should return empty string for app with no roles, got:\n%s", hcl)
	}
}

func TestRoleHCLGenerator_OnlyManagedRoles(t *testing.T) {
	app := &discovery.App{
		ID:        "app-123",
		Name:      "Test App",
		Namespace: "testapp",
		Roles: []discovery.Role{
			{ID: "role-1", Name: "Admin", Managed: true},
			{ID: "role-2", Name: "Edit", Managed: true},
			{ID: "role-3", Name: "View", Managed: true},
		},
	}

	imports := []ImportBlock{}
	uuidMap := make(map[string]string)
	gen := NewRoleHCLGenerator(app, imports, uuidMap, "elementum_app.test_app.id")
	hcl := gen.GenerateAll()

	// Should return empty since all roles are managed (exported as data sources, not resources)
	if hcl != "" {
		t.Errorf("GenerateAll() should return empty string when all roles are managed, got:\n%s", hcl)
	}
}

func TestRoleHCLGenerator_BuildUserRefMap(t *testing.T) {
	app := &discovery.App{
		ID:        "app-123",
		Name:      "Test App",
		Namespace: "testapp",
		Roles: []discovery.Role{
			{
				ID:      "role-1",
				Name:    "Role 1",
				Managed: false,
				Users: []discovery.RoleMember{
					{ID: "user-1", Name: "alice@example.com"},
					{ID: "user-2", Name: "bob@example.com"},
				},
			},
			{
				ID:      "role-2",
				Name:    "Role 2",
				Managed: false,
				Users: []discovery.RoleMember{
					{ID: "user-1", Name: "alice@example.com"}, // Duplicate
					{ID: "user-3", Name: "charlie@example.com"},
				},
			},
			{
				ID:      "role-3",
				Name:    "Managed Role",
				Managed: true, // Should be skipped
				Users: []discovery.RoleMember{
					{ID: "user-4", Name: "managed@example.com"},
				},
			},
		},
	}

	imports := []ImportBlock{}
	uuidMap := make(map[string]string)
	gen := NewRoleHCLGenerator(app, imports, uuidMap, "elementum_app.test_app.id")

	refMap := gen.buildUserRefMap()

	// Verify alice is included
	if _, ok := refMap["alice@example.com"]; !ok {
		t.Error("User ref map should include alice@example.com")
	}

	// Verify bob is included
	if _, ok := refMap["bob@example.com"]; !ok {
		t.Error("User ref map should include bob@example.com")
	}

	// Verify charlie is included
	if _, ok := refMap["charlie@example.com"]; !ok {
		t.Error("User ref map should include charlie@example.com")
	}

	// Verify managed role's user is NOT included
	if _, ok := refMap["managed@example.com"]; ok {
		t.Error("User ref map should NOT include users from managed roles")
	}

	// Verify total count (alice, bob, charlie = 3 unique)
	if len(refMap) != 3 {
		t.Errorf("User ref map should have 3 unique users, got %d", len(refMap))
	}
}

func TestRoleHCLGenerator_BuildGroupRefMap(t *testing.T) {
	app := &discovery.App{
		ID:        "app-123",
		Name:      "Test App",
		Namespace: "testapp",
		Roles: []discovery.Role{
			{
				ID:      "role-1",
				Name:    "Role 1",
				Managed: false,
				Groups: []discovery.RoleMember{
					{ID: "group-1", Name: "Engineering"},
					{ID: "group-2", Name: "Sales"},
				},
			},
			{
				ID:      "role-2",
				Name:    "Role 2",
				Managed: false,
				Groups: []discovery.RoleMember{
					{ID: "group-1", Name: "Engineering"}, // Duplicate
					{ID: "group-3", Name: "Marketing"},
				},
			},
		},
	}

	imports := []ImportBlock{}
	uuidMap := make(map[string]string)
	gen := NewRoleHCLGenerator(app, imports, uuidMap, "elementum_app.test_app.id")

	refMap := gen.buildGroupRefMap()

	// Verify groups are included
	if _, ok := refMap["Engineering"]; !ok {
		t.Error("Group ref map should include Engineering")
	}
	if _, ok := refMap["Sales"]; !ok {
		t.Error("Group ref map should include Sales")
	}
	if _, ok := refMap["Marketing"]; !ok {
		t.Error("Group ref map should include Marketing")
	}

	// Verify total count (3 unique groups)
	if len(refMap) != 3 {
		t.Errorf("Group ref map should have 3 unique groups, got %d", len(refMap))
	}
}

func TestRoleHCLGenerator_BeautifyRole(t *testing.T) {
	app := &discovery.App{
		ID:        "app-123",
		Name:      "Test App",
		Namespace: "testapp",
	}

	imports := []ImportBlock{}
	uuidMap := map[string]string{
		"uuid-123": "elementum_field.status.id",
		"uuid-456": "data.elementum_user.alice.id",
	}
	gen := NewRoleHCLGenerator(app, imports, uuidMap, "elementum_app.test_app.id")

	hcl := `field_id = "uuid-123"
user_id = "uuid-456"`

	beautified := gen.BeautifyRole(hcl)

	if !strings.Contains(beautified, "field_id = elementum_field.status.id") {
		t.Errorf("BeautifyRole should replace UUID with reference, got:\n%s", beautified)
	}
	if !strings.Contains(beautified, "user_id = data.elementum_user.alice.id") {
		t.Errorf("BeautifyRole should replace UUID with reference, got:\n%s", beautified)
	}
}

func TestGenerateRoleDataSourceBlocks(t *testing.T) {
	app := &discovery.App{
		ID:        "app-123",
		Name:      "Test App",
		Namespace: "testapp",
		Roles: []discovery.Role{
			{
				ID:      "role-1",
				Name:    "Admin",
				Managed: true,
			},
			{
				ID:      "role-2",
				Name:    "Edit",
				Managed: true,
			},
			{
				ID:      "role-3",
				Name:    "Custom Role",
				Managed: false,
				Users: []discovery.RoleMember{
					{ID: "user-1", Name: "alice@example.com"},
					{ID: "user-2", Name: "bob@example.com"},
				},
				Groups: []discovery.RoleMember{
					{ID: "group-1", Name: "Engineering"},
				},
			},
		},
	}

	hcl := GenerateRoleDataSourceBlocks(app, "elementum_app.test_app.id")

	// Verify user data sources
	if !strings.Contains(hcl, `data "elementum_user" "alice"`) {
		t.Errorf("Generated HCL should contain user data source for alice, got:\n%s", hcl)
	}
	if !strings.Contains(hcl, `email = "alice@example.com"`) {
		t.Errorf("Generated HCL should contain alice's email, got:\n%s", hcl)
	}

	if !strings.Contains(hcl, `data "elementum_user" "bob"`) {
		t.Errorf("Generated HCL should contain user data source for bob, got:\n%s", hcl)
	}

	// Verify group data sources
	if !strings.Contains(hcl, `data "elementum_group" "engineering"`) {
		t.Errorf("Generated HCL should contain group data source, got:\n%s", hcl)
	}
	if !strings.Contains(hcl, `name = "Engineering"`) {
		t.Errorf("Generated HCL should contain group name, got:\n%s", hcl)
	}

	// Verify managed role data sources
	if !strings.Contains(hcl, `data "elementum_role" "admin"`) {
		t.Errorf("Generated HCL should contain managed role data source for Admin, got:\n%s", hcl)
	}
	if !strings.Contains(hcl, `data "elementum_role" "edit"`) {
		t.Errorf("Generated HCL should contain managed role data source for Edit, got:\n%s", hcl)
	}

	// Verify managed roles reference the app
	if !strings.Contains(hcl, "object_id = elementum_app.test_app.id") {
		t.Errorf("Generated HCL should reference app for managed role object_id, got:\n%s", hcl)
	}
}

func TestGenerateRoleDataSourceBlocks_NoRoles(t *testing.T) {
	app := &discovery.App{
		ID:        "app-123",
		Name:      "Test App",
		Namespace: "testapp",
		Roles:     []discovery.Role{},
	}

	hcl := GenerateRoleDataSourceBlocks(app, "elementum_app.test_app.id")

	if hcl != "" {
		t.Errorf("GenerateRoleDataSourceBlocks should return empty for app with no roles, got:\n%s", hcl)
	}
}

func TestGenerateRoleDataSourceBlocks_OnlyManagedRoles(t *testing.T) {
	app := &discovery.App{
		ID:        "app-123",
		Name:      "Test App",
		Namespace: "testapp",
		Roles: []discovery.Role{
			{ID: "role-1", Name: "Admin", Managed: true},
			{ID: "role-2", Name: "Edit", Managed: true},
		},
	}

	hcl := GenerateRoleDataSourceBlocks(app, "elementum_app.test_app.id")

	// Should only contain managed role data sources, no user/group data sources
	if strings.Contains(hcl, `data "elementum_user"`) {
		t.Errorf("Generated HCL should NOT contain user data sources when only managed roles exist, got:\n%s", hcl)
	}
	if strings.Contains(hcl, `data "elementum_group"`) {
		t.Errorf("Generated HCL should NOT contain group data sources when only managed roles exist, got:\n%s", hcl)
	}

	// Should still have managed role data sources
	if !strings.Contains(hcl, `data "elementum_role" "admin"`) {
		t.Errorf("Generated HCL should contain managed role data source, got:\n%s", hcl)
	}
}

func TestSanitizeEmailToName(t *testing.T) {
	tests := []struct {
		email    string
		expected string
	}{
		{"alice@example.com", "alice"},
		{"bob.smith@company.org", "bob_smith"},
		{"user-name@test.io", "user_name"},
		{"John.Doe@Example.COM", "john_doe"},
		{"test", "test"}, // No @ symbol
		{"", "resource"}, // Empty string
	}

	for _, tt := range tests {
		t.Run(tt.email, func(t *testing.T) {
			result := sanitizeEmailToName(tt.email)
			if result != tt.expected {
				t.Errorf("sanitizeEmailToName(%q) = %q, want %q", tt.email, result, tt.expected)
			}
		})
	}
}

func TestCapPermissionLevel(t *testing.T) {
	tests := []struct {
		name     string
		group    string
		level    string
		expected string
	}{
		// SERVICES max is VIEW - should cap EDIT and ADMIN
		{"SERVICES with VIEW stays VIEW", "SERVICES", "VIEW", "VIEW"},
		{"SERVICES with EDIT caps to VIEW", "SERVICES", "EDIT", "VIEW"},
		{"SERVICES with ADMIN caps to VIEW", "SERVICES", "ADMIN", "VIEW"},
		{"SERVICES with NONE stays NONE", "SERVICES", "NONE", "NONE"},

		// USERS max is VIEW - should cap EDIT and ADMIN
		{"USERS with VIEW stays VIEW", "USERS", "VIEW", "VIEW"},
		{"USERS with EDIT caps to VIEW", "USERS", "EDIT", "VIEW"},
		{"USERS with ADMIN caps to VIEW", "USERS", "ADMIN", "VIEW"},

		// SLAS max is VIEW - should cap EDIT and ADMIN
		{"SLAS with VIEW stays VIEW", "SLAS", "VIEW", "VIEW"},
		{"SLAS with EDIT caps to VIEW", "SLAS", "EDIT", "VIEW"},
		{"SLAS with ADMIN caps to VIEW", "SLAS", "ADMIN", "VIEW"},

		// ATTACHMENTS max is EDIT - should only cap ADMIN
		{"ATTACHMENTS with VIEW stays VIEW", "ATTACHMENTS", "VIEW", "VIEW"},
		{"ATTACHMENTS with EDIT stays EDIT", "ATTACHMENTS", "EDIT", "EDIT"},
		{"ATTACHMENTS with ADMIN caps to EDIT", "ATTACHMENTS", "ADMIN", "EDIT"},

		// RECORDS max is ADMIN - nothing should be capped
		{"RECORDS with VIEW stays VIEW", "RECORDS", "VIEW", "VIEW"},
		{"RECORDS with EDIT stays EDIT", "RECORDS", "EDIT", "EDIT"},
		{"RECORDS with ADMIN stays ADMIN", "RECORDS", "ADMIN", "ADMIN"},

		// AUTOMATIONS max is ADMIN - nothing should be capped
		{"AUTOMATIONS with ADMIN stays ADMIN", "AUTOMATIONS", "ADMIN", "ADMIN"},

		// CUSTOM level should never be capped (user specifies exact permissions)
		{"SERVICES with CUSTOM stays CUSTOM", "SERVICES", "CUSTOM", "CUSTOM"},
		{"USERS with CUSTOM stays CUSTOM", "USERS", "CUSTOM", "CUSTOM"},
		{"ATTACHMENTS with CUSTOM stays CUSTOM", "ATTACHMENTS", "CUSTOM", "CUSTOM"},

		// Unknown group should pass through unchanged
		{"Unknown group passes through", "UNKNOWN_GROUP", "ADMIN", "ADMIN"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := capPermissionLevel(tt.group, tt.level)
			if result != tt.expected {
				t.Errorf("capPermissionLevel(%q, %q) = %q, want %q", tt.group, tt.level, result, tt.expected)
			}
		})
	}
}
