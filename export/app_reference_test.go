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

package export

import (
	"strings"
	"testing"

	"github.com/elementumltd/elementum-cli/discovery"
)

// TestAppResourceName_NamespaceFirst verifies that AppResourceName uses namespace when available
func TestAppResourceName_NamespaceFirst(t *testing.T) {
	tests := []struct {
		name     string
		app      *discovery.App
		expected string
	}{
		{
			name: "uses namespace when set",
			app: &discovery.App{
				ID:        "app-123",
				Name:      "Velocity Activities (DEV)",
				Namespace: "activitiesdev",
			},
			expected: "activitiesdev",
		},
		{
			name: "falls back to name when namespace empty",
			app: &discovery.App{
				ID:        "app-123",
				Name:      "Velocity Activities (DEV)",
				Namespace: "",
			},
			expected: "velocity_activities_dev",
		},
		{
			name: "handles namespace with special chars",
			app: &discovery.App{
				ID:        "app-123",
				Name:      "Test App",
				Namespace: "test-app",
			},
			expected: "test_app",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := AppResourceName(tt.app)
			if result != tt.expected {
				t.Errorf("AppResourceName() = %q, want %q", result, tt.expected)
			}
		})
	}
}

// TestAutomationHCLGenerator_UsesNamespaceForAppReference verifies that automation
// generator uses namespace-based app references, not display name
func TestAutomationHCLGenerator_UsesNamespaceForAppReference(t *testing.T) {
	// This is the real-world scenario: app has namespace "activitiesdev"
	// but display name "Velocity Activities (DEV)"
	app := &discovery.App{
		ID:        TestAppID,
		Name:      "Velocity Activities (DEV)",
		Namespace: "activitiesdev",
		Automations: []discovery.Automation{
			{
				ID:           TestAutomationID,
				Name:         "Test Automation",
				Status:       "ACTIVE",
				WorkflowID:   TestWorkflowID,
				HasPublished: true,
			},
		},
	}

	imports := []ImportBlock{
		{ID: TestAppID, ResourceType: "elementum_app", ResourceName: "activitiesdev"},
		{ID: TestAutomationID, ResourceType: "elementum_automation", ResourceName: "test_automation"},
	}
	uuidMap := make(map[string]string)

	generator := NewAutomationHCLGenerator(app, imports, uuidMap)
	hcl := generator.GenerateAll()

	// MUST use namespace-based reference
	if !strings.Contains(hcl, "app_id = elementum_app.activitiesdev.id") {
		t.Errorf("Automation should reference app by namespace.\nExpected: app_id = elementum_app.activitiesdev.id\nGot HCL:\n%s", hcl)
	}

	// MUST NOT use display name-based reference
	if strings.Contains(hcl, "velocity_activities") {
		t.Errorf("Automation should NOT reference app by display name format.\nGot HCL:\n%s", hcl)
	}
}

// TestAgentHCLGenerator_UsesNamespaceForAppReference verifies that agent
// generator uses namespace-based app references
func TestAgentHCLGenerator_UsesNamespaceForAppReference(t *testing.T) {
	app := &discovery.App{
		ID:        TestAppID,
		Name:      "Velocity Activities (DEV)",
		Namespace: "activitiesdev",
		Agents: []discovery.Agent{
			{
				ID:           TestAgentID,
				Name:         "Test Agent",
				Description:  "A test agent",
				Type:         "AgentElementum",
				Instructions: "Help users",
				Tools: []discovery.AgentTool{
					{
						ID:          TestAgentToolID,
						Name:        "Search Records",
						Description: "Search for records",
						Type:        "AgentSearchAspectTool",
						AspectID:    TestAppID,
						AspectName:  "Test Aspect",
					},
				},
			},
		},
	}

	imports := []ImportBlock{
		{ID: TestAppID, ResourceType: "elementum_app", ResourceName: "activitiesdev"},
		{ID: TestAppID + ":" + TestAgentID, ResourceType: "elementum_agent", ResourceName: "test_agent"},
		{ID: TestAppID + ":" + TestAgentID + ":" + TestAgentToolID, ResourceType: "elementum_agent_search_records_tool", ResourceName: "search_records"},
	}
	uuidMap := make(map[string]string)

	generator := NewAgentHCLGenerator(app, imports, uuidMap)
	hcl := generator.GenerateAll()

	// Agent resource MUST use namespace-based reference
	if !strings.Contains(hcl, "app_id = elementum_app.activitiesdev.id") {
		t.Errorf("Agent should reference app by namespace.\nExpected: app_id = elementum_app.activitiesdev.id\nGot HCL:\n%s", hcl)
	}

	// Agent tool MUST use namespace-based reference
	if !strings.Contains(hcl, "app_id = elementum_app.activitiesdev.id") {
		t.Errorf("Agent tool should reference app by namespace.\nExpected: app_id = elementum_app.activitiesdev.id\nGot HCL:\n%s", hcl)
	}

	// MUST NOT use display name-based reference
	if strings.Contains(hcl, "velocity_activities") {
		t.Errorf("Agent/tool should NOT reference app by display name format.\nGot HCL:\n%s", hcl)
	}
}

// TestAccessPolicyHCLGenerator_UsesNamespaceForAppReference verifies that access policy
// generator uses namespace-based app references
func TestAccessPolicyHCLGenerator_UsesNamespaceForAppReference(t *testing.T) {
	app := &discovery.App{
		ID:        TestAppID,
		Name:      "Velocity Activities (DEV)",
		Namespace: "activitiesdev",
		AccessPolicies: []discovery.AccessPolicy{
			{
				ID:       "policy-1",
				ObjectID: TestAppID,
				UserIDs:  []string{"user-1"},
				UserEmails: map[string]string{
					"user-1": "test@example.com",
				},
			},
		},
	}

	imports := []ImportBlock{
		{ID: TestAppID, ResourceType: "elementum_app", ResourceName: "activitiesdev"},
		{ID: TestAppID + ":policy-1", ResourceType: "elementum_access_policy", ResourceName: "activitiesdev_policy_1"},
	}
	uuidMap := make(map[string]string)

	generator := NewAccessPolicyHCLGenerator(app, imports, uuidMap)
	hcl := generator.GenerateAll()

	// MUST use namespace-based reference
	if !strings.Contains(hcl, "object_id = elementum_app.activitiesdev.id") {
		t.Errorf("Access policy should reference app by namespace.\nExpected: object_id = elementum_app.activitiesdev.id\nGot HCL:\n%s", hcl)
	}

	// MUST NOT use display name-based reference
	if strings.Contains(hcl, "velocity_activities") {
		t.Errorf("Access policy should NOT reference app by display name format.\nGot HCL:\n%s", hcl)
	}
}

// TestTaskHCLGenerator_ResolveObjectRef_UsesNamespaceForAppReference verifies that
// task generator's resolveObjectRef uses namespace-based app references
func TestTaskHCLGenerator_ResolveObjectRef_UsesNamespaceForAppReference(t *testing.T) {
	app := &discovery.App{
		ID:        TestAppID,
		Name:      "Velocity Activities (DEV)",
		Namespace: "activitiesdev",
	}

	imports := []ImportBlock{
		{ID: TestAppID, ResourceType: "elementum_app", ResourceName: "activitiesdev"},
	}
	uuidMap := make(map[string]string)

	generator := NewTaskHCLGenerator(app, imports, uuidMap, nil)

	// Test resolveObjectRef when object is the same app
	ref := generator.resolveObjectRef(TestAppID)

	// MUST use namespace-based reference
	if ref != "elementum_app.activitiesdev.id" {
		t.Errorf("resolveObjectRef should return namespace-based reference.\nExpected: elementum_app.activitiesdev.id\nGot: %s", ref)
	}

	// MUST NOT use display name-based reference
	if strings.Contains(ref, "velocity_activities") {
		t.Errorf("resolveObjectRef should NOT return display name format.\nGot: %s", ref)
	}
}

// TestRelationshipHCLGenerator_ResolveObjectRef_UsesNamespaceForAppReference verifies that
// relationship generator's resolveObjectRef uses namespace-based app references
func TestRelationshipHCLGenerator_ResolveObjectRef_UsesNamespaceForAppReference(t *testing.T) {
	app := &discovery.App{
		ID:        TestAppID,
		Name:      "Velocity Activities (DEV)",
		Namespace: "activitiesdev",
	}

	imports := []ImportBlock{
		{ID: TestAppID, ResourceType: "elementum_app", ResourceName: "activitiesdev"},
	}
	uuidMap := make(map[string]string)

	generator := NewRelationshipHCLGenerator(app, imports, uuidMap)

	// Test resolveObjectRef when object is the same app
	ref := generator.resolveObjectRef(TestAppID)

	// MUST use namespace-based reference
	if ref != "elementum_app.activitiesdev.id" {
		t.Errorf("resolveObjectRef should return namespace-based reference.\nExpected: elementum_app.activitiesdev.id\nGot: %s", ref)
	}

	// MUST NOT use display name-based reference
	if strings.Contains(ref, "velocity_activities") {
		t.Errorf("resolveObjectRef should NOT return display name format.\nGot: %s", ref)
	}
}

// TestAISearchTableHCLGenerator_UsesNamespaceForAppReference verifies that AI search table
// generator uses namespace-based app references
func TestAISearchTableHCLGenerator_UsesNamespaceForAppReference(t *testing.T) {
	app := &discovery.App{
		ID:        TestAppID,
		Name:      "Velocity Activities (DEV)",
		Namespace: "activitiesdev",
		AISearchTables: []discovery.AISearchTable{
			{
				ID:                      "table-1",
				ObjectID:                TestAppID,
				FieldID:                 TestFieldID,
				FieldName:               "Description",
				AIProviderConnectorID:   "connector-1",
				AIProviderConnectorName: "text-embedding-3-small",
				Duration:                "DAYS",
				TargetLag:               1,
			},
		},
	}

	imports := []ImportBlock{
		{ID: TestAppID, ResourceType: "elementum_app", ResourceName: "activitiesdev"},
		{ID: TestAppID + ":table-1", ResourceType: "elementum_ai_search_table", ResourceName: "description"},
	}
	uuidMap := map[string]string{
		TestFieldID: "elementum_field.title.id",
	}

	generator := NewAISearchTableHCLGenerator(app, imports, uuidMap)
	hcl := generator.GenerateAll()

	// MUST use namespace-based reference
	if !strings.Contains(hcl, "object_id = elementum_app.activitiesdev.id") {
		t.Errorf("AI search table should reference app by namespace.\nExpected: object_id = elementum_app.activitiesdev.id\nGot HCL:\n%s", hcl)
	}

	// MUST NOT use display name-based reference
	if strings.Contains(hcl, "velocity_activities") {
		t.Errorf("AI search table should NOT reference app by display name format.\nGot HCL:\n%s", hcl)
	}
}

// TestFileReaderHCLGenerator_UsesNamespaceForAppReference verifies that file reader
// generator uses namespace-based app references
func TestFileReaderHCLGenerator_UsesNamespaceForAppReference(t *testing.T) {
	app := &discovery.App{
		ID:        TestAppID,
		Name:      "Velocity Activities (DEV)",
		Namespace: "activitiesdev",
		AIFileReaders: []discovery.FileReader{
			{
				ID:           "reader-1",
				Name:         "Test Reader",
				Type:         discovery.FileReaderTypeAI,
				Instructions: "Extract data",
			},
		},
	}

	imports := []ImportBlock{
		{ID: TestAppID, ResourceType: "elementum_app", ResourceName: "activitiesdev"},
		{ID: TestAppID + ":reader-1", ResourceType: "elementum_ai_file_reader", ResourceName: "test_reader"},
	}
	uuidMap := make(map[string]string)

	generator := NewFileReaderHCLGenerator(app, imports, uuidMap)
	hcl := generator.GenerateAll()

	// MUST use namespace-based reference (spacing may vary)
	if !strings.Contains(hcl, "elementum_app.activitiesdev.id") {
		t.Errorf("File reader should reference app by namespace.\nExpected reference to: elementum_app.activitiesdev.id\nGot HCL:\n%s", hcl)
	}

	// MUST NOT use display name-based reference
	if strings.Contains(hcl, "velocity_activities") {
		t.Errorf("File reader should NOT reference app by display name format.\nGot HCL:\n%s", hcl)
	}
}

// TestAllGenerators_FallbackToNameWhenNoNamespace verifies that all generators
// correctly fall back to sanitized name when namespace is not set
func TestAllGenerators_FallbackToNameWhenNoNamespace(t *testing.T) {
	// App without namespace - should fall back to sanitized name
	app := &discovery.App{
		ID:        TestAppID,
		Name:      "Test App",
		Namespace: "", // No namespace set
		Automations: []discovery.Automation{
			{
				ID:           TestAutomationID,
				Name:         "Test Automation",
				Status:       "ACTIVE",
				WorkflowID:   TestWorkflowID,
				HasPublished: true,
			},
		},
	}

	imports := []ImportBlock{
		{ID: TestAppID, ResourceType: "elementum_app", ResourceName: "test_app"},
		{ID: TestAutomationID, ResourceType: "elementum_automation", ResourceName: "test_automation"},
	}
	uuidMap := make(map[string]string)

	generator := NewAutomationHCLGenerator(app, imports, uuidMap)
	hcl := generator.GenerateAll()

	// Should use sanitized name when no namespace
	if !strings.Contains(hcl, "app_id = elementum_app.test_app.id") {
		t.Errorf("Should fall back to sanitized name when no namespace.\nExpected: app_id = elementum_app.test_app.id\nGot HCL:\n%s", hcl)
	}
}
