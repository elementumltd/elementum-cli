// Copyright (c) ElementumAI, Inc.
// SPDX-License-Identifier: MPL-2.0

package export

import (
	"strings"
	"testing"

	"github.com/elementumltd/elementum-cli/discovery"
)

func TestExtractFlowFromPlan(t *testing.T) {
	planOutput := `
  # elementum_flow.deal_execution_flow will be imported
  # (config will be generated)
    resource "elementum_flow" "deal_execution_flow" {
        id        = "app-uuid-123"
        object_id = "app-uuid-123"
        stages    = [
            {
                nodes    = [
                    {
                        action          = {
                            automation = {
                                automation_id = "automation-uuid-456"
                                name          = "Review Draft"
                            }
                        }
                        connects_to     = []
                        description     = "Review the draft"
                        id              = "node-uuid-789"
                        key             = "review_draft"
                        source_position = "RIGHT"
                        symbol          = "SQUARE"
                        target_position = "LEFT"
                        x               = 100
                        y               = 200
                    },
                ]
                stage_id = "stage-uuid-abc"
            },
        ]
    }

  # __generated__ by OpenTofu
`

	result := ExtractResourceFromPlan(planOutput, "elementum_flow", "deal_execution_flow")

	if result == "" {
		t.Fatal("Expected flow to be extracted, got empty string")
	}

	// Should have the resource declaration
	if !strings.Contains(result, `resource "elementum_flow" "deal_execution_flow"`) {
		t.Error("Result should contain resource declaration")
	}

	// Should have stages
	if !strings.Contains(result, "stages") {
		t.Error("Result should contain stages")
	}

	// Should have the automation action
	if !strings.Contains(result, "automation-uuid-456") {
		t.Error("Result should contain automation UUID")
	}

	// Should be reformatted to 2-space indentation
	if !strings.Contains(result, "  object_id") {
		t.Error("Result should have 2-space indentation for top-level attributes")
	}
}

func TestBeautifyFlowConfig(t *testing.T) {
	tests := []struct {
		name            string
		input           string
		appResourceName string
		uuidMap         map[string]string
		wantContains    []string
		wantNotContains []string
	}{
		{
			name: "replaces top-level object_id with app reference",
			input: `resource "elementum_flow" "my_flow" {
  id = "app-uuid-123"
  object_id = "app-uuid-123"
  stages = []
}`,
			appResourceName: "my_app",
			uuidMap:         map[string]string{},
			wantContains: []string{
				"object_id = elementum_app.my_app.id",
			},
			wantNotContains: []string{
				`object_id = "app-uuid-123"`,
			},
		},
		{
			name: "replaces automation_id with resource reference",
			input: `resource "elementum_flow" "my_flow" {
  stages = [
    {
      nodes = [
        {
          action = {
            automation = {
              automation_id = "automation-uuid-456"
              name = "My Automation"
            }
          }
        }
      ]
    }
  ]
}`,
			appResourceName: "my_app",
			uuidMap: map[string]string{
				"automation-uuid-456": "elementum_automation.my_automation.id",
			},
			wantContains: []string{
				"automation_id = elementum_automation.my_automation.id",
			},
			wantNotContains: []string{
				`automation_id = "automation-uuid-456"`,
			},
		},
		{
			name: "replaces object_id in action blocks with stage reference",
			input: `resource "elementum_flow" "my_flow" {
  stages = [
    {
      nodes = [
        {
          action = {
            object = {
              name = "Some Object"
              object_id = "stage-uuid-abc"
            }
          }
        }
      ]
    }
  ]
}`,
			appResourceName: "my_app",
			uuidMap: map[string]string{
				"stage-uuid-abc": "local.stage_ids_by_key[\"open\"]",
			},
			wantContains: []string{
				"object_id = local.stage_ids_by_key[\"open\"]",
			},
			wantNotContains: []string{
				`object_id = "stage-uuid-abc"`,
			},
		},
		{
			name: "removes empty action blocks",
			input: `resource "elementum_flow" "my_flow" {
  stages = [
    {
      nodes = [
        {
          action = {}
          description = "Some node"
        }
      ]
    }
  ]
}`,
			appResourceName: "my_app",
			uuidMap:         map[string]string{},
			wantContains: []string{
				"description",
			},
			wantNotContains: []string{
				"action = {}",
			},
		},
		{
			name: "handles agent_id replacement",
			input: `resource "elementum_flow" "my_flow" {
  stages = [
    {
      nodes = [
        {
          action = {
            agent = {
              agent_id = "agent-uuid-xyz"
            }
          }
        }
      ]
    }
  ]
}`,
			appResourceName: "my_app",
			uuidMap: map[string]string{
				"agent-uuid-xyz": "elementum_agent.my_agent.id",
			},
			wantContains: []string{
				"agent_id = elementum_agent.my_agent.id",
			},
			wantNotContains: []string{
				`agent_id = "agent-uuid-xyz"`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := beautifyFlowConfig(tt.input, tt.uuidMap, tt.appResourceName)

			for _, want := range tt.wantContains {
				if !strings.Contains(result, want) {
					t.Errorf("Expected result to contain %q, got:\n%s", want, result)
				}
			}

			for _, notWant := range tt.wantNotContains {
				if strings.Contains(result, notWant) {
					t.Errorf("Expected result NOT to contain %q, got:\n%s", notWant, result)
				}
			}
		})
	}
}

func TestInjectFlowConfig(t *testing.T) {
	// Mock generated content with stages = null
	generatedContent := `resource "elementum_app" "my_app" {
  name = "My App"
}

resource "elementum_flow" "my_flow" {
  object_id = elementum_app.my_app.id
  stages = null
}

resource "elementum_text_field" "my_field" {
  name = "Field"
}
`

	// Mock plan output with full flow data
	planOutput := `
  # elementum_flow.my_flow will be imported
    resource "elementum_flow" "my_flow" {
        id        = "app-uuid-123"
        object_id = "app-uuid-123"
        stages    = [
            {
                nodes    = [
                    {
                        action          = {
                            automation = {
                                automation_id = "automation-uuid-456"
                                name          = "Process"
                            }
                        }
                        description     = "Process node"
                        id              = "node-uuid"
                        key             = "process"
                        source_position = "RIGHT"
                        symbol          = "SQUARE"
                        target_position = "LEFT"
                        x               = 100
                        y               = 200
                    },
                ]
                stage_id = "stage-uuid-abc"
            },
        ]
    }

  # __generated__
`

	imports := []ImportBlock{
		{
			ID:           "app-uuid-123",
			ResourceType: "elementum_app",
			ResourceName: "my_app",
		},
		{
			ID:           "app-uuid-123:automation-uuid-456",
			ResourceType: "elementum_automation",
			ResourceName: "my_automation",
		},
	}

	app := &discovery.App{
		ID:   "app-uuid-123",
		Name: "My App",
		Automations: []discovery.Automation{
			{
				ID:   "automation-uuid-456",
				Name: "Process",
			},
		},
	}

	result := InjectFlowConfig(generatedContent, planOutput, imports, app)

	// Debug: print the result
	t.Logf("Result:\n%s", result)

	// Should still have the app resource
	if !strings.Contains(result, `resource "elementum_app" "my_app"`) {
		t.Error("Result should still contain app resource")
	}

	// Should still have the field resource
	if !strings.Contains(result, `resource "elementum_text_field" "my_field"`) {
		t.Error("Result should still contain field resource")
	}

	// Should have replaced stages = null with actual stages
	if strings.Contains(result, "stages = null") {
		t.Error("Result should not contain 'stages = null'")
	}

	// Should have the flow stages with nodes (terraform formats with alignment spaces)
	if !strings.Contains(result, "stages") || !strings.Contains(result, "= [") {
		t.Error("Result should contain stages array")
	}

	// Should have the node description
	if !strings.Contains(result, "Process node") {
		t.Error("Result should contain node description")
	}

	// Should have beautified the automation_id
	if !strings.Contains(result, "elementum_automation.my_automation.id") {
		t.Error("Result should have beautified automation_id reference")
	}

	// Should have beautified the top-level object_id
	if !strings.Contains(result, "object_id = elementum_app.my_app.id") {
		t.Error("Result should have beautified object_id reference to app")
	}
}

func TestIndentationReformatting(t *testing.T) {
	// Terraform plan uses 4-space indentation
	planFlowWithFourSpaces := `    resource "elementum_flow" "test" {
        id        = "uuid"
        object_id = "uuid"
        stages    = [
            {
                stage_id = "stage"
                nodes    = []
            },
        ]
    }`

	// After extraction, should be reformatted to 2-space indentation
	result := ExtractResourceFromPlan(planFlowWithFourSpaces, "elementum_flow", "test")

	if result == "" {
		t.Fatal("Failed to extract flow from plan")
	}

	t.Logf("Extracted result:\n%s", result)

	// Check that it's using 2-space indentation
	if !strings.Contains(result, "\n  object_id") {
		t.Errorf("Should have 2-space indentation for top-level attributes. Got:\n%s", result)
	}

	if !strings.Contains(result, "\n  stages") {
		t.Errorf("Should have 2-space indentation for stages. Got:\n%s", result)
	}

	if strings.Contains(result, "\n        object_id") {
		t.Error("Should not have 8-space indentation (terraform plan format)")
	}
}
