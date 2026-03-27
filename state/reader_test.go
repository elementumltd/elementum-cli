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

package state

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestExtractRefs(t *testing.T) {
	t.Parallel()

	state := &TerraformState{
		Version: 4,
		Resources: []Resource{
			{
				Mode:     "managed",
				Type:     "elementum_record_created_trigger",
				Name:     "my_trigger",
				Provider: "registry.terraform.io/elementumltd/elementum",
				Instances: []Instance{
					{
						Attributes: map[string]any{
							"id":          "trigger-123",
							"workflow_id": "workflow-456",
							"refs": map[string]any{
								"ID":     `{"triggerReference":{"name":"record.id"},"fieldType":"TEXT"}`,
								"Title":  `{"triggerReference":{"name":"record.title"},"fieldType":"TEXT"}`,
								"Status": `{"triggerReference":{"name":"record.field-abc"},"fieldType":"PICKLIST"}`,
							},
						},
					},
				},
			},
			{
				Mode:     "managed",
				Type:     "elementum_variable_task",
				Name:     "my_task",
				Provider: "registry.terraform.io/elementumltd/elementum",
				Instances: []Instance{
					{
						Attributes: map[string]any{
							"id":          "task-789",
							"workflow_id": "workflow-456",
							"refs": map[string]any{
								"my_variable": `{"taskReference":{"taskId":"task-789","name":"my_variable"},"fieldType":"NUMBER"}`,
							},
						},
					},
				},
			},
			{
				Mode:     "data",
				Type:     "elementum_app",
				Name:     "my_app",
				Provider: "registry.terraform.io/elementumltd/elementum",
				Instances: []Instance{
					{
						Attributes: map[string]any{
							"id":   "app-123",
							"name": "My App",
						},
					},
				},
			},
		},
	}

	t.Run("extracts all refs", func(t *testing.T) {
		result, err := ExtractRefs(state, "")
		if err != nil {
			t.Fatalf("ExtractRefs failed: %v", err)
		}

		if len(result.Resources) != 2 {
			t.Errorf("expected 2 resources with refs, got %d", len(result.Resources))
		}
	})

	t.Run("filters by resource name", func(t *testing.T) {
		result, err := ExtractRefs(state, "my_trigger")
		if err != nil {
			t.Fatalf("ExtractRefs failed: %v", err)
		}

		if len(result.Resources) != 1 {
			t.Errorf("expected 1 resource matching filter, got %d", len(result.Resources))
		}
		if result.Resources[0].ResourceName != "my_trigger" {
			t.Errorf("expected resource name 'my_trigger', got '%s'", result.Resources[0].ResourceName)
		}
	})

	t.Run("parses ref types correctly", func(t *testing.T) {
		result, err := ExtractRefs(state, "my_trigger")
		if err != nil {
			t.Fatalf("ExtractRefs failed: %v", err)
		}

		refs := result.Resources[0].Refs
		if len(refs) != 3 {
			t.Errorf("expected 3 refs, got %d", len(refs))
		}

		// Find the Status ref
		var statusRef *RefInfo
		for i := range refs {
			if refs[i].Name == "Status" {
				statusRef = &refs[i]
				break
			}
		}

		if statusRef == nil {
			t.Fatal("Status ref not found")
		}
		if statusRef.Type != "PICKLIST" {
			t.Errorf("expected Status type 'PICKLIST', got '%s'", statusRef.Type)
		}
	})

	t.Run("skips data sources", func(t *testing.T) {
		result, err := ExtractRefs(state, "my_app")
		if err != nil {
			t.Fatalf("ExtractRefs failed: %v", err)
		}

		if len(result.Resources) != 0 {
			t.Errorf("expected 0 resources (data sources should be skipped), got %d", len(result.Resources))
		}
	})
}

func TestExtractAutomations(t *testing.T) {
	t.Parallel()

	state := &TerraformState{
		Version: 4,
		Resources: []Resource{
			{
				Mode:     "managed",
				Type:     "elementum_automation",
				Name:     "ticket_processor",
				Provider: "registry.terraform.io/elementumltd/elementum",
				Instances: []Instance{
					{
						Attributes: map[string]any{
							"id":          "automation-123",
							"name":        "Process Tickets",
							"app_id":      "app-456",
							"workflow_id": "workflow-789",
							"trigger": []any{
								map[string]any{
									"type": "record_created",
									"id":   "trigger-abc",
								},
							},
							"refs": map[string]any{
								"ID":    `{"triggerReference":{"name":"record.id"},"fieldType":"TEXT"}`,
								"Title": `{"triggerReference":{"name":"record.title"},"fieldType":"TEXT"}`,
							},
						},
					},
				},
			},
			{
				Mode:     "managed",
				Type:     "elementum_message_task",
				Name:     "notify_user",
				Provider: "registry.terraform.io/elementumltd/elementum",
				Instances: []Instance{
					{
						Attributes: map[string]any{
							"id":          "task-001",
							"name":        "Send Notification",
							"workflow_id": "workflow-789",
							"refs": map[string]any{
								"message_sent": `{"taskReference":{"taskId":"task-001"},"fieldType":"BOOLEAN"}`,
							},
						},
					},
				},
			},
		},
	}

	t.Run("extracts automation with associated tasks", func(t *testing.T) {
		result, err := ExtractAutomations(state, "")
		if err != nil {
			t.Fatalf("ExtractAutomations failed: %v", err)
		}

		if len(result.Automations) != 1 {
			t.Fatalf("expected 1 automation, got %d", len(result.Automations))
		}

		automation := result.Automations[0]
		if automation.Name != "Process Tickets" {
			t.Errorf("expected automation name 'Process Tickets', got '%s'", automation.Name)
		}
		if automation.TriggerType != "record_created" {
			t.Errorf("expected trigger type 'record_created', got '%s'", automation.TriggerType)
		}
		if len(automation.Tasks) != 1 {
			t.Errorf("expected 1 task associated with automation, got %d", len(automation.Tasks))
		}
		if len(automation.Refs) != 2 {
			t.Errorf("expected 2 refs on automation, got %d", len(automation.Refs))
		}
	})

	t.Run("filters by automation name", func(t *testing.T) {
		result, err := ExtractAutomations(state, "ticket_processor")
		if err != nil {
			t.Fatalf("ExtractAutomations failed: %v", err)
		}

		if len(result.Automations) != 1 {
			t.Errorf("expected 1 automation matching filter, got %d", len(result.Automations))
		}
	})
}

func TestFindAutomationByID(t *testing.T) {
	t.Parallel()

	state := &TerraformState{
		Version: 4,
		Resources: []Resource{
			{
				Mode: "managed",
				Type: "elementum_automation",
				Name: "test_automation",
				Instances: []Instance{
					{
						Attributes: map[string]any{
							"id":          "automation-xyz",
							"name":        "Test Automation",
							"workflow_id": "workflow-123",
						},
					},
				},
			},
		},
	}

	t.Run("finds automation by ID", func(t *testing.T) {
		automation, err := FindAutomationByID(state, "automation-xyz")
		if err != nil {
			t.Fatalf("FindAutomationByID failed: %v", err)
		}
		if automation.ID != "automation-xyz" {
			t.Errorf("expected ID 'automation-xyz', got '%s'", automation.ID)
		}
	})

	t.Run("returns error for non-existent ID", func(t *testing.T) {
		_, err := FindAutomationByID(state, "non-existent")
		if err == nil {
			t.Error("expected error for non-existent automation ID")
		}
	})
}

func TestFindAutomationByName(t *testing.T) {
	t.Parallel()

	state := &TerraformState{
		Version: 4,
		Resources: []Resource{
			{
				Mode: "managed",
				Type: "elementum_automation",
				Name: "my_automation",
				Instances: []Instance{
					{
						Attributes: map[string]any{
							"id":          "automation-123",
							"name":        "My Automation Display Name",
							"workflow_id": "workflow-456",
						},
					},
				},
			},
		},
	}

	t.Run("finds by resource name", func(t *testing.T) {
		automation, err := FindAutomationByName(state, "my_automation")
		if err != nil {
			t.Fatalf("FindAutomationByName failed: %v", err)
		}
		if automation.ResourceName != "my_automation" {
			t.Errorf("expected resource name 'my_automation', got '%s'", automation.ResourceName)
		}
	})

	t.Run("finds by display name", func(t *testing.T) {
		automation, err := FindAutomationByName(state, "My Automation Display Name")
		if err != nil {
			t.Fatalf("FindAutomationByName failed: %v", err)
		}
		if automation.Name != "My Automation Display Name" {
			t.Errorf("expected name 'My Automation Display Name', got '%s'", automation.Name)
		}
	})

	t.Run("returns error for non-existent name", func(t *testing.T) {
		_, err := FindAutomationByName(state, "non-existent")
		if err == nil {
			t.Error("expected error for non-existent automation name")
		}
	})
}

func TestFormatRefType(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		ref      RefInfo
		expected string
	}{
		{
			name:     "simple type",
			ref:      RefInfo{Name: "ID", Type: "TEXT", Multiple: false},
			expected: "TEXT",
		},
		{
			name:     "multiple type",
			ref:      RefInfo{Name: "Attachments", Type: "ATTACHMENT", Multiple: true},
			expected: "ATTACHMENT[]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatRefType(tt.ref)
			if result != tt.expected {
				t.Errorf("FormatRefType() = %q, expected %q", result, tt.expected)
			}
		})
	}
}

func TestReadStateFile(t *testing.T) {
	t.Parallel()

	// Create a temporary state file
	tempDir := t.TempDir()
	stateFile := filepath.Join(tempDir, "terraform.tfstate")

	testState := TerraformState{
		Version:          4,
		TerraformVersion: "1.5.0",
		Serial:           1,
		Lineage:          "test-lineage",
		Resources:        []Resource{},
	}

	data, err := json.Marshal(testState)
	if err != nil {
		t.Fatalf("failed to marshal test state: %v", err)
	}

	if err := os.WriteFile(stateFile, data, 0644); err != nil {
		t.Fatalf("failed to write test state file: %v", err)
	}

	t.Run("reads valid state file", func(t *testing.T) {
		state, err := ReadStateFile(stateFile)
		if err != nil {
			t.Fatalf("ReadStateFile failed: %v", err)
		}
		if state.Version != 4 {
			t.Errorf("expected version 4, got %d", state.Version)
		}
		if state.TerraformVersion != "1.5.0" {
			t.Errorf("expected terraform version '1.5.0', got '%s'", state.TerraformVersion)
		}
	})

	t.Run("returns error for non-existent file", func(t *testing.T) {
		_, err := ReadStateFile("/non/existent/path.tfstate")
		if err == nil {
			t.Error("expected error for non-existent file")
		}
	})
}

func TestFindStateFile(t *testing.T) {
	t.Parallel()

	t.Run("returns specified path if exists", func(t *testing.T) {
		tempDir := t.TempDir()
		stateFile := filepath.Join(tempDir, "custom.tfstate")
		if err := os.WriteFile(stateFile, []byte("{}"), 0644); err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}

		result, err := FindStateFile(stateFile)
		if err != nil {
			t.Fatalf("FindStateFile failed: %v", err)
		}
		if result != stateFile {
			t.Errorf("expected '%s', got '%s'", stateFile, result)
		}
	})

	t.Run("returns error for non-existent specified path", func(t *testing.T) {
		_, err := FindStateFile("/non/existent/path.tfstate")
		if err == nil {
			t.Error("expected error for non-existent file")
		}
	})
}

func TestBuildResourceAddress(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		module       string
		resourceType string
		resourceName string
		indexKey     any
		expected     string
	}{
		{
			name:         "simple resource",
			module:       "",
			resourceType: "elementum_automation",
			resourceName: "my_automation",
			indexKey:     nil,
			expected:     "elementum_automation.my_automation",
		},
		{
			name:         "resource in module",
			module:       "module.tickets",
			resourceType: "elementum_automation",
			resourceName: "processor",
			indexKey:     nil,
			expected:     "module.tickets.elementum_automation.processor",
		},
		{
			name:         "resource with string index",
			module:       "",
			resourceType: "elementum_field",
			resourceName: "fields",
			indexKey:     "status",
			expected:     `elementum_field.fields["status"]`,
		},
		{
			name:         "resource with numeric index",
			module:       "",
			resourceType: "elementum_field",
			resourceName: "fields",
			indexKey:     float64(0),
			expected:     "elementum_field.fields[0]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildResourceAddress(tt.module, tt.resourceType, tt.resourceName, tt.indexKey)
			if result != tt.expected {
				t.Errorf("buildResourceAddress() = %q, expected %q", result, tt.expected)
			}
		})
	}
}

func TestIsElementumResource(t *testing.T) {
	t.Parallel()

	tests := []struct {
		resourceType string
		expected     bool
	}{
		{"elementum_record_created_trigger", true},
		{"elementum_record_updated_trigger", true},
		{"elementum_message_task", true},
		{"elementum_variable_task", true},
		{"elementum_record_search_task", true},
		{"elementum_automation", false},
		{"elementum_app", false},
		{"elementum_field", false},
		{"aws_instance", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.resourceType, func(t *testing.T) {
			result := isElementumResource(tt.resourceType)
			if result != tt.expected {
				t.Errorf("isElementumResource(%q) = %v, expected %v", tt.resourceType, result, tt.expected)
			}
		})
	}
}
