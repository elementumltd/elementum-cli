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
)

func TestNormalizePlanIndentation(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name: "simple resource with attributes",
			input: `    resource "elementum_flow" "test" {
        id        = "uuid"
        object_id = "app-id"
    }`,
			expected: `resource "elementum_flow" "test" {
  id        = "uuid"
  object_id = "app-id"
}`,
		},
		{
			name: "nested blocks",
			input: `    resource "elementum_flow" "test" {
        stages = [
            {
                stage_id = "stage1"
                nodes    = []
            },
        ]
    }`,
			expected: `resource "elementum_flow" "test" {
  stages = [
    {
      stage_id = "stage1"
      nodes    = []
    },
  ]
}`,
		},
		{
			name: "deeply nested structure",
			input: `    resource "elementum_flow" "test" {
        stages = [
            {
                nodes = [
                    {
                        action = {
                            automation = {
                                automation_id = "auto-123"
                            }
                        }
                    },
                ]
            },
        ]
    }`,
			expected: `resource "elementum_flow" "test" {
  stages = [
    {
      nodes = [
        {
          action = {
            automation = {
              automation_id = "auto-123"
            }
          }
        },
      ]
    },
  ]
}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NormalizePlanIndentation(tt.input)
			if result != tt.expected {
				t.Errorf("NormalizePlanIndentation() mismatch\nGot:\n%s\n\nExpected:\n%s", result, tt.expected)
			}
		})
	}
}

func TestExtractResourceFromPlan(t *testing.T) {
	tests := []struct {
		name         string
		planOutput   string
		resourceType string
		resourceName string
		wantContains []string
		wantEmpty    bool
	}{
		{
			name: "extracts flow resource",
			planOutput: `
  # elementum_flow.my_flow will be imported
    resource "elementum_flow" "my_flow" {
        id        = "flow-123"
        object_id = "app-456"
        stages    = []
    }

  # __generated__`,
			resourceType: "elementum_flow",
			resourceName: "my_flow",
			wantContains: []string{
				`resource "elementum_flow" "my_flow"`,
				"id",
				"object_id",
				"stages",
			},
		},
		{
			name: "extracts automation resource",
			planOutput: `
  # elementum_automation.my_automation will be imported
    resource "elementum_automation" "my_automation" {
        id     = "auto-123"
        name   = "Test Automation"
        app_id = "app-456"
    }

  # __generated__`,
			resourceType: "elementum_automation",
			resourceName: "my_automation",
			wantContains: []string{
				`resource "elementum_automation" "my_automation"`,
				"name",
				"app_id",
			},
		},
		{
			name: "returns empty for non-existent resource",
			planOutput: `
  # elementum_flow.other_flow will be imported
    resource "elementum_flow" "other_flow" {
        id = "flow-123"
    }`,
			resourceType: "elementum_flow",
			resourceName: "my_flow",
			wantEmpty:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractResourceFromPlan(tt.planOutput, tt.resourceType, tt.resourceName)

			if tt.wantEmpty {
				if result != "" {
					t.Errorf("Expected empty result, got: %s", result)
				}
				return
			}

			if result == "" {
				t.Fatal("Expected non-empty result, got empty string")
			}

			for _, want := range tt.wantContains {
				if !strings.Contains(result, want) {
					t.Errorf("Result should contain %q\nGot:\n%s", want, result)
				}
			}

			// Verify correct indentation (2-space for top-level attributes)
			if !strings.Contains(result, "\n  id") && !strings.Contains(result, "\n  name") {
				t.Errorf("Result should have 2-space indentation for attributes\nGot:\n%s", result)
			}
		})
	}
}

func TestExtractResourceFromPlanIndentationLevels(t *testing.T) {
	// This test specifically verifies the indentation conversion
	planOutput := `    resource "elementum_test" "test" {
        level_1 = "value"
        nested  = {
            level_2 = "value"
            deeper  = {
                level_3 = "value"
            }
        }
    }`

	result := ExtractResourceFromPlan(planOutput, "elementum_test", "test")

	// Check each indentation level
	expectedLines := []string{
		`resource "elementum_test" "test" {`,
		`  level_1 = "value"`,     // 2 spaces
		`  nested  = {`,           // 2 spaces
		`    level_2 = "value"`,   // 4 spaces
		`    deeper  = {`,         // 4 spaces
		`      level_3 = "value"`, // 6 spaces
		`    }`,                   // 4 spaces
		`  }`,                     // 2 spaces
		`}`,                       // 0 spaces
	}

	for _, expected := range expectedLines {
		if !strings.Contains(result, expected) {
			t.Errorf("Result missing expected line %q\nGot:\n%s", expected, result)
		}
	}
}
