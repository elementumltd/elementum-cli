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
	"time"

	"github.com/elementumltd/elementum-cli/internal/client"
)

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		name string
		ms   int
		want string
	}{
		{"zero", 0, "-"},
		{"negative", -1, "-"},
		{"milliseconds small", 50, "50ms"},
		{"milliseconds", 500, "500ms"},
		{"just under 1s", 999, "999ms"},
		{"exactly 1s", 1000, "1.0s"},
		{"seconds", 1200, "1.2s"},
		{"seconds rounded", 5500, "5.5s"},
		{"just under 1m", 59999, "60.0s"},
		{"exactly 1m", 60000, "1m"},
		{"1m 15s", 75000, "1m 15s"},
		{"2m 30s", 150000, "2m 30s"},
		{"exactly 1h", 3600000, "1h"},
		{"1h 5m", 3900000, "1h 5m"},
		{"2h 30m", 9000000, "2h 30m"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatDuration(tt.ms)
			if got != tt.want {
				t.Errorf("formatDuration(%d) = %q, want %q", tt.ms, got, tt.want)
			}
		})
	}
}

func TestParseSince(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name      string
		value     string
		wantErr   bool
		checkFunc func(t time.Time) bool // validates the result is approximately correct
	}{
		{
			name:  "7 days",
			value: "7d",
			checkFunc: func(parsed time.Time) bool {
				expected := now.Add(-7 * 24 * time.Hour)
				return absDuration(parsed.Sub(expected)) < 2*time.Second
			},
		},
		{
			name:  "24 hours",
			value: "24h",
			checkFunc: func(parsed time.Time) bool {
				expected := now.Add(-24 * time.Hour)
				return absDuration(parsed.Sub(expected)) < 2*time.Second
			},
		},
		{
			name:  "30 minutes",
			value: "30m",
			checkFunc: func(parsed time.Time) bool {
				expected := now.Add(-30 * time.Minute)
				return absDuration(parsed.Sub(expected)) < 2*time.Second
			},
		},
		{
			name:  "60 seconds",
			value: "60s",
			checkFunc: func(parsed time.Time) bool {
				expected := now.Add(-60 * time.Second)
				return absDuration(parsed.Sub(expected)) < 2*time.Second
			},
		},
		{
			name:  "date only",
			value: "2026-02-01",
			checkFunc: func(parsed time.Time) bool {
				expected, _ := time.Parse("2006-01-02", "2026-02-01")
				return parsed.Equal(expected)
			},
		},
		{
			name:  "RFC3339",
			value: "2026-02-01T10:00:00Z",
			checkFunc: func(parsed time.Time) bool {
				expected, _ := time.Parse(time.RFC3339, "2026-02-01T10:00:00Z")
				return parsed.Equal(expected)
			},
		},
		{
			name:  "empty defaults to 7d",
			value: "",
			checkFunc: func(parsed time.Time) bool {
				expected := now.Add(-7 * 24 * time.Hour)
				return absDuration(parsed.Sub(expected)) < 2*time.Second
			},
		},
		{
			name:    "invalid",
			value:   "foobar",
			wantErr: true,
		},
		{
			name:    "invalid suffix",
			value:   "7x",
			wantErr: true,
		},
		{
			name:    "zero value",
			value:   "0d",
			wantErr: true, // 0 is not > 0
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseSince(tt.value)
			if tt.wantErr {
				if err == nil {
					t.Errorf("parseSince(%q) expected error, got %v", tt.value, got)
				}
				return
			}
			if err != nil {
				t.Errorf("parseSince(%q) unexpected error: %v", tt.value, err)
				return
			}
			if tt.checkFunc != nil && !tt.checkFunc(got) {
				t.Errorf("parseSince(%q) = %v, did not pass validation", tt.value, got)
			}
		})
	}
}

func TestStyledStatus(t *testing.T) {
	tests := []struct {
		status string
	}{
		{"SUCCESS"},
		{"FAILURE"},
		{"RUNNING"},
		{"QUEUED"},
		{"CANCELLED"},
	}

	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			result := styledStatus(tt.status)
			if result == "" {
				t.Errorf("styledStatus(%q) returned empty string", tt.status)
			}
			// The styled string should contain the status text (it wraps it in ANSI codes)
			if len(result) < len(tt.status) {
				t.Errorf("styledStatus(%q) result too short: %q", tt.status, result)
			}
		})
	}
}

func TestStyledStatus_Unknown(t *testing.T) {
	result := styledStatus("UNKNOWN")
	if result != "UNKNOWN" {
		t.Errorf("styledStatus(\"UNKNOWN\") = %q, want \"UNKNOWN\"", result)
	}
}

func TestFormatErrors(t *testing.T) {
	tests := []struct {
		name   string
		errors []string
		want   string
	}{
		{"no errors", nil, "-"},
		{"empty slice", []string{}, "-"},
		{"single error", []string{"Task timeout"}, "Task timeout"},
		{"multiple errors", []string{"Error 1", "Error 2"}, "Error 1 (+1 more)"},
		{"long error truncated", []string{"This is a very long error message that exceeds fifty characters limit for display"}, "This is a very long error message that exceeds ..."},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatErrors(tt.errors)
			if got != tt.want {
				t.Errorf("formatErrors() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFormatTimestamp(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"RFC3339", "2026-02-09T10:30:15Z"},
		{"RFC3339 with offset", "2026-02-09T10:30:15+05:00"},
		{"unparseable", "not-a-date"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatTimestamp(tt.input)
			if result == "" {
				t.Errorf("formatTimestamp(%q) returned empty string", tt.input)
			}
			// For unparseable, should return input as-is
			if tt.name == "unparseable" && result != tt.input {
				t.Errorf("formatTimestamp(%q) = %q, want %q (as-is for unparseable)", tt.input, result, tt.input)
			}
		})
	}
}

func TestBuildHealthSummary(t *testing.T) {
	tests := []struct {
		name     string
		statuses []client.AutomationExecutionStatus
		wantLen  int // minimum length (has styled text, so check > 0)
	}{
		{
			name:     "empty",
			statuses: nil,
			wantLen:  1, // "no executions" is rendered
		},
		{
			name:     "all success",
			statuses: []client.AutomationExecutionStatus{client.AutomationExecutionStatusSuccess, client.AutomationExecutionStatusSuccess},
			wantLen:  1,
		},
		{
			name: "mixed",
			statuses: []client.AutomationExecutionStatus{
				client.AutomationExecutionStatusSuccess,
				client.AutomationExecutionStatusFailure,
				client.AutomationExecutionStatusRunning,
			},
			wantLen: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Build edges from statuses
			edges := make([]client.GetAutomationExecutionsOrganizationAutomationExecutionsAutomationExecutionConnectionEdgesAutomationExecutionEdge, len(tt.statuses))
			for i, s := range tt.statuses {
				edges[i].Node.Status = s
			}

			result := buildHealthSummary(edges)
			if len(result) < tt.wantLen {
				t.Errorf("buildHealthSummary() result too short: %q", result)
			}
		})
	}
}

func TestCommandRegistration(t *testing.T) {
	// Verify automationStatusCmd is properly configured with new usage pattern
	expectedUse := "status (<automation-id> | <app-namespace> <automation-name>) [execution-id]"
	if automationStatusCmd.Use != expectedUse {
		t.Errorf("unexpected Use: %q, want %q", automationStatusCmd.Use, expectedUse)
	}

	// Verify flags exist
	flags := []string{"status", "limit", "since", "watch", "expand", "timeline", "html"}
	for _, name := range flags {
		if automationStatusCmd.Flags().Lookup(name) == nil {
			t.Errorf("missing flag: %s", name)
		}
	}

	// Verify watch has short flag
	watchFlag := automationStatusCmd.Flags().ShorthandLookup("w")
	if watchFlag == nil {
		t.Error("missing short flag -w for --watch")
	}

	// Verify default values
	limitFlag := automationStatusCmd.Flags().Lookup("limit")
	if limitFlag.DefValue != "20" {
		t.Errorf("expected limit default 20, got %s", limitFlag.DefValue)
	}

	sinceFlag := automationStatusCmd.Flags().Lookup("since")
	if sinceFlag.DefValue != "7d" {
		t.Errorf("expected since default 7d, got %s", sinceFlag.DefValue)
	}
}

func TestLooksLikeUUID(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		// Valid UUIDs
		{"valid uuid lowercase", "9063aed1-bf8c-430d-882f-8c502355a3c7", true},
		{"valid uuid uppercase", "9063AED1-BF8C-430D-882F-8C502355A3C7", true},
		{"valid uuid mixed case", "9063AED1-bf8c-430D-882f-8c502355a3c7", true},
		{"valid uuid v4", "550e8400-e29b-41d4-a716-446655440000", true},

		// Invalid - namespaces/names
		{"namespace simple", "financialview", false},
		{"namespace with numbers", "myapp123", false},
		{"namespace hyphenated", "my-app-name", false},
		{"automation name simple", "ValidateAndSubmit", false},
		{"automation name with spaces", "Process Request", false},
		{"automation name with dash", "David - testing", false},

		// Edge cases
		{"empty string", "", false},
		{"partial uuid", "9063aed1-bf8c-430d", false},
		{"uuid without dashes", "9063aed1bf8c430d882f8c502355a3c7", true}, // uuid.Parse accepts this format
		{"uuid with extra chars", "9063aed1-bf8c-430d-882f-8c502355a3c7x", false},
		{"looks like uuid but wrong length", "9063aed1-bf8c-430d-882f-8c502355a3c", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := looksLikeUUID(tt.input)
			if got != tt.want {
				t.Errorf("looksLikeUUID(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestArgumentParsing(t *testing.T) {
	// Test the argument parsing logic conceptually
	// These tests verify the expected behavior of different argument combinations

	tests := []struct {
		name              string
		args              []string
		expectUUID        bool   // true if first arg should be treated as UUID
		expectNamespace   string // expected namespace (if not UUID)
		expectAutomation  string // expected automation name (if not UUID)
		expectExecutionID string // expected execution ID
		expectError       bool   // true if should error
	}{
		{
			name:       "uuid only",
			args:       []string{"9063aed1-bf8c-430d-882f-8c502355a3c7"},
			expectUUID: true,
		},
		{
			name:              "uuid with execution id",
			args:              []string{"9063aed1-bf8c-430d-882f-8c502355a3c7", "exec-456"},
			expectUUID:        true,
			expectExecutionID: "exec-456",
		},
		{
			name:             "namespace and name",
			args:             []string{"financialview", "Load Data"},
			expectUUID:       false,
			expectNamespace:  "financialview",
			expectAutomation: "Load Data",
		},
		{
			name:              "namespace, name, and execution id",
			args:              []string{"financialview", "Load Data", "exec-789"},
			expectUUID:        false,
			expectNamespace:   "financialview",
			expectAutomation:  "Load Data",
			expectExecutionID: "exec-789",
		},
		{
			name:        "namespace only - should error",
			args:        []string{"financialview"},
			expectUUID:  false,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate the argument parsing logic from runAutomationStatus
			var automationID string
			var executionID string
			var namespace string
			var automationName string
			var isError bool

			if len(tt.args) == 0 {
				isError = true
			} else if looksLikeUUID(tt.args[0]) {
				// UUID pattern
				automationID = tt.args[0]
				if len(tt.args) == 2 {
					executionID = tt.args[1]
				}
			} else {
				// Namespace pattern
				if len(tt.args) < 2 {
					isError = true
				} else {
					namespace = tt.args[0]
					automationName = tt.args[1]
					if len(tt.args) == 3 {
						executionID = tt.args[2]
					}
				}
			}

			// Verify expectations
			if isError != tt.expectError {
				t.Errorf("error state = %v, want %v", isError, tt.expectError)
				return
			}

			if tt.expectError {
				return // No more checks needed for error cases
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

			if executionID != tt.expectExecutionID {
				t.Errorf("executionID = %q, want %q", executionID, tt.expectExecutionID)
			}
		})
	}
}

func TestAutomationCmdRegistration(t *testing.T) {
	// Verify status is a subcommand of automation
	found := false
	for _, sub := range automationsCmd.Commands() {
		if sub.Name() == "status" {
			found = true
			break
		}
	}
	if !found {
		t.Error("status command not registered under automation")
	}
}

// absDuration returns the absolute value of a duration.
func absDuration(d time.Duration) time.Duration {
	if d < 0 {
		return -d
	}
	return d
}
