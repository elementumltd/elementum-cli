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
	"context"
	"slices"
	"testing"
)

func TestEnvironmentsParentCommandRegistration(t *testing.T) {
	if environmentsCmd.Use != "environments" {
		t.Errorf("unexpected Use: %q, want %q", environmentsCmd.Use, "environments")
	}

	// Check aliases
	if !slices.Contains(environmentsCmd.Aliases, "envs") {
		t.Error("missing alias 'envs'")
	}

	// Check subcommands
	subcommands := map[string]bool{
		"list": false,
	}

	for _, sub := range environmentsCmd.Commands() {
		if _, ok := subcommands[sub.Name()]; ok {
			subcommands[sub.Name()] = true
		}
	}

	for name, found := range subcommands {
		if !found {
			t.Errorf("missing subcommand: %s", name)
		}
	}
}

func TestGetEnvironmentsCmd(t *testing.T) {
	cmd := GetEnvironmentsCmd()
	if cmd == nil {
		t.Fatal("GetEnvironmentsCmd returned nil")
	}
	if cmd != environmentsCmd {
		t.Error("GetEnvironmentsCmd should return environmentsCmd")
	}
}

func TestEnvironmentsListCommandRegistration(t *testing.T) {
	if environmentsListCmd.Use != "list" {
		t.Errorf("unexpected Use: %q, want %q", environmentsListCmd.Use, "list")
	}

	if environmentsListCmd.RunE == nil {
		t.Error("RunE should not be nil")
	}
}

func TestResolveEnvironmentByName_UUIDPassthrough(t *testing.T) {
	// When given a UUID, resolveEnvironmentByName should return it directly without API call.
	// This is a unit test that doesn't require API access.
	tests := []struct {
		name      string
		input     string
		wantUUID  bool
		wantID    string
		wantName  string
		wantError bool
	}{
		{
			name:     "valid UUID v4",
			input:    "550e8400-e29b-41d4-a716-446655440000",
			wantUUID: true,
			wantID:   "550e8400-e29b-41d4-a716-446655440000",
			wantName: "550e8400-e29b-41d4-a716-446655440000",
		},
		{
			name:     "valid UUID lowercase",
			input:    "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
			wantUUID: true,
			wantID:   "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
			wantName: "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
		},
		{
			name:     "valid UUID uppercase",
			input:    "A1B2C3D4-E5F6-7890-ABCD-EF1234567890",
			wantUUID: true,
			wantID:   "A1B2C3D4-E5F6-7890-ABCD-EF1234567890",
			wantName: "A1B2C3D4-E5F6-7890-ABCD-EF1234567890",
		},
		{
			name:      "non-UUID environment name",
			input:     "Production",
			wantUUID:  false,
			wantError: true, // Will fail because no API client, but that's expected
		},
		{
			name:      "non-UUID with dashes",
			input:     "my-staging-env",
			wantUUID:  false,
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// For UUID inputs, we can test without an API client
			if tt.wantUUID {
				id, name, err := resolveEnvironmentByName(context.TODO(), nil, tt.input)
				if err != nil {
					t.Errorf("unexpected error for UUID input: %v", err)
					return
				}
				if id != tt.wantID {
					t.Errorf("id = %q, want %q", id, tt.wantID)
				}
				if name != tt.wantName {
					t.Errorf("name = %q, want %q", name, tt.wantName)
				}
			}
			// For non-UUID inputs, we'd need an API client which would require
			// integration tests, so we just verify that UUID detection works
		})
	}
}
