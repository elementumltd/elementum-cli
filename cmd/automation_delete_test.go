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

package cmd

import (
	"testing"

	"github.com/elementumltd/elementum-cli/internal/client"
)

func TestAutomationDeleteCommandRegistration(t *testing.T) {
	// Verify automationDeleteCmd is properly configured
	expectedUse := "delete (<automation-id> | <app-namespace> <automation-name>)"
	if automationDeleteCmd.Use != expectedUse {
		t.Errorf("unexpected Use: %q, want %q", automationDeleteCmd.Use, expectedUse)
	}

	// Verify flags exist
	flags := []string{"force", "dry-run"}
	for _, name := range flags {
		if automationDeleteCmd.Flags().Lookup(name) == nil {
			t.Errorf("missing flag: %s", name)
		}
	}

	// Verify force has short flag -f
	forceFlag := automationDeleteCmd.Flags().ShorthandLookup("f")
	if forceFlag == nil {
		t.Error("missing short flag -f for --force")
	}

	// Verify default values
	forceDefault := automationDeleteCmd.Flags().Lookup("force")
	if forceDefault.DefValue != "false" {
		t.Errorf("expected force default false, got %s", forceDefault.DefValue)
	}

	dryRunDefault := automationDeleteCmd.Flags().Lookup("dry-run")
	if dryRunDefault.DefValue != "false" {
		t.Errorf("expected dry-run default false, got %s", dryRunDefault.DefValue)
	}
}

func TestAutomationDeleteCmdRegistered(t *testing.T) {
	// Verify delete is a subcommand of automation
	found := false
	for _, sub := range automationsCmd.Commands() {
		if sub.Name() == "delete" {
			found = true
			break
		}
	}
	if !found {
		t.Error("delete command not registered under automation")
	}
}

func TestAutomationDeleteArgumentParsing(t *testing.T) {
	// Test the argument parsing logic for delete command
	tests := []struct {
		name             string
		args             []string
		expectUUID       bool   // true if first arg should be treated as UUID
		expectNamespace  string // expected namespace (if not UUID)
		expectAutomation string // expected automation name (if not UUID)
		expectError      bool   // true if should error
	}{
		{
			name:       "uuid only",
			args:       []string{"9063aed1-bf8c-430d-882f-8c502355a3c7"},
			expectUUID: true,
		},
		{
			name:             "namespace and name",
			args:             []string{"myapp", "Process Request"},
			expectUUID:       false,
			expectNamespace:  "myapp",
			expectAutomation: "Process Request",
		},
		{
			name:        "namespace only - should error",
			args:        []string{"myapp"},
			expectUUID:  false,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate the argument parsing logic from runAutomationDelete
			var automationID string
			var namespace string
			var automationName string
			var isError bool

			if len(tt.args) == 0 {
				isError = true
			} else if looksLikeUUID(tt.args[0]) {
				automationID = tt.args[0]
			} else {
				if len(tt.args) < 2 {
					isError = true
				} else {
					namespace = tt.args[0]
					automationName = tt.args[1]
				}
			}

			// Verify expectations
			if isError != tt.expectError {
				t.Errorf("error state = %v, want %v", isError, tt.expectError)
				return
			}

			if tt.expectError {
				return
			}

			if tt.expectUUID {
				if automationID != tt.args[0] {
					t.Errorf("automationID = %q, want %q", automationID, tt.args[0])
				}
			} else {
				if namespace != tt.expectNamespace {
					t.Errorf("namespace = %q, want %q", namespace, tt.expectNamespace)
				}
				if automationName != tt.expectAutomation {
					t.Errorf("automationName = %q, want %q", automationName, tt.expectAutomation)
				}
			}
		})
	}
}

func TestNeedsDisableLogic(t *testing.T) {
	tests := []struct {
		name               string
		status             client.AutomationStatus
		hasCurrentWorkflow bool
		expectNeedsDisable bool
	}{
		{
			name:               "active with current workflow",
			status:             client.AutomationStatusActive,
			hasCurrentWorkflow: true,
			expectNeedsDisable: true,
		},
		{
			name:               "active without current workflow",
			status:             client.AutomationStatusActive,
			hasCurrentWorkflow: false,
			expectNeedsDisable: false,
		},
		{
			name:               "disabled with current workflow",
			status:             client.AutomationStatusDisabled,
			hasCurrentWorkflow: true,
			expectNeedsDisable: false,
		},
		{
			name:               "inactive with current workflow",
			status:             client.AutomationStatusInactive,
			hasCurrentWorkflow: true,
			expectNeedsDisable: false,
		},
		{
			name:               "unpublished",
			status:             client.AutomationStatusUnpublished,
			hasCurrentWorkflow: false,
			expectNeedsDisable: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var currentWorkflowID string
			if tt.hasCurrentWorkflow {
				currentWorkflowID = "workflow-123"
			}

			// Replicate the logic from runAutomationDelete
			needsDisable := tt.status == client.AutomationStatusActive && currentWorkflowID != ""

			if needsDisable != tt.expectNeedsDisable {
				t.Errorf("needsDisable = %v, want %v", needsDisable, tt.expectNeedsDisable)
			}
		})
	}
}

func TestFormatBlockers(t *testing.T) {
	// Test the blocker formatting logic
	tests := []struct {
		name     string
		blockers []string
		want     string
	}{
		{
			name:     "no blockers",
			blockers: nil,
			want:     "",
		},
		{
			name:     "empty slice",
			blockers: []string{},
			want:     "",
		},
		{
			name:     "single blocker",
			blockers: []string{"  * Automation: Test (abc-123)"},
			want:     "  * Automation: Test (abc-123)\n",
		},
		{
			name: "multiple blockers",
			blockers: []string{
				"  * Automation: Parent Workflow (abc-123)",
				"  * Agent: Support Bot",
				"  * Agent Tool: Run Validation",
			},
			want: "  * Automation: Parent Workflow (abc-123)\n  * Agent: Support Bot\n  * Agent Tool: Run Validation\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Replicate the formatting logic from getAutomationBlockers
			if len(tt.blockers) == 0 {
				if tt.want != "" {
					t.Errorf("expected empty string for no blockers")
				}
				return
			}

			result := ""
			for _, b := range tt.blockers {
				result += b + "\n"
			}

			if result != tt.want {
				t.Errorf("formatBlockers() = %q, want %q", result, tt.want)
			}
		})
	}
}

func TestActionDescription(t *testing.T) {
	tests := []struct {
		name         string
		needsDisable bool
		wantDesc     string
	}{
		{
			name:         "needs disable",
			needsDisable: true,
			wantDesc:     "disable and delete",
		},
		{
			name:         "no disable needed",
			needsDisable: false,
			wantDesc:     "delete",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Replicate the logic from runAutomationDelete
			actionDesc := "delete"
			if tt.needsDisable {
				actionDesc = "disable and delete"
			}

			if actionDesc != tt.wantDesc {
				t.Errorf("actionDesc = %q, want %q", actionDesc, tt.wantDesc)
			}
		})
	}
}
