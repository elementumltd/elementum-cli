// Copyright (c) ElementumAI, Inc.
// SPDX-License-Identifier: MPL-2.0

package cmd

import (
	"testing"
)

func TestFilterInterventions(t *testing.T) {
	interventions := []Intervention{
		{ID: "1", Status: "OPEN", AutomationName: "Process Orders"},
		{ID: "2", Status: "RESOLVED", AutomationName: "Send Notifications"},
		{ID: "3", Status: "OPEN", AutomationName: "Update Inventory"},
		{ID: "4", Status: "IN_PROGRESS", AutomationName: "Process Orders"},
		{ID: "5", Status: "IGNORED", AutomationName: "Archive Records"},
	}

	tests := []struct {
		name             string
		statusFilter     string
		automationFilter string
		wantCount        int
		wantIDs          []string
	}{
		{
			name:             "no filters",
			statusFilter:     "",
			automationFilter: "",
			wantCount:        5,
		},
		{
			name:         "filter by status OPEN",
			statusFilter: "OPEN",
			wantCount:    2,
			wantIDs:      []string{"1", "3"},
		},
		{
			name:         "filter by status case insensitive",
			statusFilter: "open",
			wantCount:    2,
			wantIDs:      []string{"1", "3"},
		},
		{
			name:         "filter by status RESOLVED",
			statusFilter: "RESOLVED",
			wantCount:    1,
			wantIDs:      []string{"2"},
		},
		{
			name:         "filter by status IN_PROGRESS",
			statusFilter: "IN_PROGRESS",
			wantCount:    1,
			wantIDs:      []string{"4"},
		},
		{
			name:             "filter by automation name",
			automationFilter: "Process",
			wantCount:        2,
			wantIDs:          []string{"1", "4"},
		},
		{
			name:             "filter by automation name case insensitive",
			automationFilter: "process",
			wantCount:        2,
			wantIDs:          []string{"1", "4"},
		},
		{
			name:             "filter by automation partial match",
			automationFilter: "Notif",
			wantCount:        1,
			wantIDs:          []string{"2"},
		},
		{
			name:             "combined filters",
			statusFilter:     "OPEN",
			automationFilter: "Process",
			wantCount:        1,
			wantIDs:          []string{"1"},
		},
		{
			name:             "no matches",
			statusFilter:     "OPEN",
			automationFilter: "NonExistent",
			wantCount:        0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := filterInterventions(interventions, tt.statusFilter, tt.automationFilter)
			if len(result) != tt.wantCount {
				t.Errorf("filterInterventions() returned %d items, want %d", len(result), tt.wantCount)
			}
			if tt.wantIDs != nil {
				for i, id := range tt.wantIDs {
					if i >= len(result) {
						t.Errorf("missing expected ID %s at index %d", id, i)
						continue
					}
					if result[i].ID != id {
						t.Errorf("result[%d].ID = %s, want %s", i, result[i].ID, id)
					}
				}
			}
		})
	}
}

func TestStyledInterventionStatus(t *testing.T) {
	tests := []struct {
		status string
	}{
		{"OPEN"},
		{"IN_PROGRESS"},
		{"RESOLVED"},
		{"IGNORED"},
	}

	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			result := styledInterventionStatus(tt.status)
			if result == "" {
				t.Errorf("styledInterventionStatus(%q) returned empty string", tt.status)
			}
			// The styled string should contain the status text (wrapped in ANSI codes)
			if len(result) < len(tt.status) {
				t.Errorf("styledInterventionStatus(%q) result too short: %q", tt.status, result)
			}
		})
	}
}

func TestStyledInterventionStatus_Unknown(t *testing.T) {
	result := styledInterventionStatus("UNKNOWN")
	if result != "UNKNOWN" {
		t.Errorf("styledInterventionStatus(\"UNKNOWN\") = %q, want \"UNKNOWN\"", result)
	}
}

func TestFormatFailureCode(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"CONDUCTOR_WORKFLOW_FAILED", "CONDUCTOR WORKFLOW FAILED"},
		{"HTTP_500", "HTTP 500"},
		{"CONFIGURATION_ERROR", "CONFIGURATION ERROR"},
		{"SIMPLE", "SIMPLE"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := formatFailureCode(tt.input)
			if got != tt.want {
				t.Errorf("formatFailureCode(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestFormatResolution(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"DATA_CORRECTED", "data corrected"},
		{"RETRY_FROM_START", "retry from start"},
		{"FIXED_CONFIGURATION", "fixed configuration"},
		{"IGNORED_ACCEPTED", "ignored accepted"},
		{"SIMPLE", "simple"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := formatResolution(tt.input)
			if got != tt.want {
				t.Errorf("formatResolution(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestWordWrap(t *testing.T) {
	tests := []struct {
		name  string
		text  string
		width int
		want  string
	}{
		{
			name:  "short text no wrap",
			text:  "Hello world",
			width: 20,
			want:  "Hello world",
		},
		{
			name:  "exact width",
			text:  "Hello",
			width: 5,
			want:  "Hello",
		},
		{
			name:  "needs wrap",
			text:  "Hello world this is a test",
			width: 12,
			want:  "Hello world\n           this is a\n           test",
		},
		{
			name:  "empty string",
			text:  "",
			width: 10,
			want:  "",
		},
		{
			name:  "single word longer than width",
			text:  "supercalifragilisticexpialidocious",
			width: 10,
			want:  "supercalifragilisticexpialidocious",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := wordWrap(tt.text, tt.width)
			if got != tt.want {
				t.Errorf("wordWrap(%q, %d) = %q, want %q", tt.text, tt.width, got, tt.want)
			}
		})
	}
}

func TestInterventionsCommandRegistration(t *testing.T) {
	// Verify interventionsCmd is properly configured
	if interventionsCmd.Use != "interventions" {
		t.Errorf("unexpected Use: %q", interventionsCmd.Use)
	}

	// Verify subcommands exist
	subcommands := []string{"list", "show"}
	for _, name := range subcommands {
		found := false
		for _, sub := range interventionsCmd.Commands() {
			if sub.Name() == name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("missing subcommand: %s", name)
		}
	}
}

func TestInterventionsListCommandFlags(t *testing.T) {
	// Verify list command flags exist
	flags := []string{"status", "automation", "limit"}
	for _, name := range flags {
		if interventionsListCmd.Flags().Lookup(name) == nil {
			t.Errorf("missing flag: %s", name)
		}
	}

	// Verify default values
	limitFlag := interventionsListCmd.Flags().Lookup("limit")
	if limitFlag.DefValue != "100" {
		t.Errorf("expected limit default 100, got %s", limitFlag.DefValue)
	}
}

func TestInterventionsShowCommandArgs(t *testing.T) {
	// Verify show command requires 2 args
	if interventionsShowCmd.Use != "show <aspect-namespace-or-id> <intervention-id>" {
		t.Errorf("unexpected Use: %q", interventionsShowCmd.Use)
	}
}

func TestInterventionStruct(t *testing.T) {
	// Test that Intervention struct has all expected fields
	i := Intervention{
		ID:                "test-id",
		Status:            "OPEN",
		FailureCode:       "HTTP_500",
		FailureReason:     strPtr("Server error"),
		Resolution:        strPtr("RETRY_FROM_START"),
		ResolutionSummary: strPtr("Retried successfully"),
		Notes:             strPtr("Some notes"),
		CreatedAt:         "2026-02-20T10:00:00Z",
		UpdatedAt:         "2026-02-20T11:00:00Z",
		FailureAt:         "2026-02-20T10:00:00Z",
		ClosedAt:          strPtr("2026-02-20T11:00:00Z"),
		AutomationID:      "automation-123",
		AutomationName:    "Test Automation",
	}

	if i.ID != "test-id" {
		t.Errorf("ID mismatch")
	}
	if i.Status != "OPEN" {
		t.Errorf("Status mismatch")
	}
	if i.AutomationName != "Test Automation" {
		t.Errorf("AutomationName mismatch")
	}
}

// strPtr is a helper to create string pointers for tests
func strPtr(s string) *string {
	return &s
}
