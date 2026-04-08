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
)

func TestAgentsParentCommandRegistration(t *testing.T) {
	if agentsParentCmd.Use != "agents" {
		t.Errorf("unexpected Use: %q, want %q", agentsParentCmd.Use, "agents")
	}

	subcommands := map[string]bool{
		"list":   false,
		"create": false,
		"show":   false,
		"update": false,
		"delete": false,
	}

	for _, sub := range agentsParentCmd.Commands() {
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

func TestGetAgentsCmd(t *testing.T) {
	cmd := GetAgentsCmd()
	if cmd == nil {
		t.Fatal("GetAgentsCmd returned nil")
	}
	if cmd != agentsParentCmd {
		t.Error("GetAgentsCmd should return agentsParentCmd")
	}
}

func TestAgentsListCommandRegistration(t *testing.T) {
	if agentsListCmd.Use != "list <namespace>" {
		t.Errorf("unexpected Use: %q, want %q", agentsListCmd.Use, "list <namespace>")
	}

	if agentsListCmd.Args == nil {
		t.Error("Args should not be nil")
	}

	if agentsListCmd.RunE == nil {
		t.Error("RunE should not be nil")
	}
}

func TestAgentsCreateCommandRegistration(t *testing.T) {
	if agentsCreateCmd.Use != "create" {
		t.Errorf("unexpected Use: %q, want %q", agentsCreateCmd.Use, "create")
	}

	requiredFlags := []string{"app", "name", "description", "instructions"}
	for _, name := range requiredFlags {
		f := agentsCreateCmd.Flags().Lookup(name)
		if f == nil {
			t.Errorf("missing required flag: %s", name)
		}
	}

	optionalFlags := []string{"model", "ai-provider-connector-id", "first-message", "type", "dry-run"}
	for _, name := range optionalFlags {
		if agentsCreateCmd.Flags().Lookup(name) == nil {
			t.Errorf("missing flag: %s", name)
		}
	}

	typeFlag := agentsCreateCmd.Flags().Lookup("type")
	if typeFlag != nil && typeFlag.DefValue != "elementum" {
		t.Errorf("type flag default = %q, want %q", typeFlag.DefValue, "elementum")
	}
}

func TestAgentsShowCommandRegistration(t *testing.T) {
	if agentsShowCmd.Use != "show <agent-name-or-id>" {
		t.Errorf("unexpected Use: %q", agentsShowCmd.Use)
	}

	if agentsShowCmd.Args == nil {
		t.Error("Args should not be nil")
	}

	if agentsShowCmd.RunE == nil {
		t.Error("RunE should not be nil")
	}
}

func TestAgentsUpdateCommandRegistration(t *testing.T) {
	if agentsUpdateCmd.Use != "update <agent-name-or-id>" {
		t.Errorf("unexpected Use: %q", agentsUpdateCmd.Use)
	}

	if agentsUpdateCmd.Args == nil {
		t.Error("Args should not be nil")
	}

	flags := []string{"name", "description", "instructions", "model", "ai-provider-connector-id", "first-message"}
	for _, name := range flags {
		if agentsUpdateCmd.Flags().Lookup(name) == nil {
			t.Errorf("missing flag: %s", name)
		}
	}
}

func TestAgentsDeleteCommandRegistration(t *testing.T) {
	if agentsDeleteCmd.Use != "delete <agent-name-or-id>" {
		t.Errorf("unexpected Use: %q", agentsDeleteCmd.Use)
	}

	if agentsDeleteCmd.Args == nil {
		t.Error("Args should not be nil")
	}

	if agentsDeleteCmd.Flags().Lookup("force") == nil {
		t.Error("missing flag: force")
	}

	if agentsDeleteCmd.Flags().ShorthandLookup("f") == nil {
		t.Error("missing short flag -f for --force")
	}
}

func TestGetString(t *testing.T) {
	tests := []struct {
		name string
		m    map[string]interface{}
		key  string
		want string
	}{
		{"existing key", map[string]interface{}{"name": "test"}, "name", "test"},
		{"missing key", map[string]interface{}{"name": "test"}, "other", ""},
		{"nil map value", map[string]interface{}{"name": nil}, "name", ""},
		{"non-string value", map[string]interface{}{"count": 42}, "count", ""},
		{"empty string", map[string]interface{}{"name": ""}, "name", ""},
		{"empty map", map[string]interface{}{}, "name", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getString(tt.m, tt.key)
			if got != tt.want {
				t.Errorf("getString(%v, %q) = %q, want %q", tt.m, tt.key, got, tt.want)
			}
		})
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		maxLen int
		want   string
	}{
		{"short string", "hello", 10, "hello"},
		{"exact length", "hello", 5, "hello"},
		{"needs truncation", "hello world", 8, "hello wo..."},
		{"very long", "this is a very long string that needs truncation", 20, "this is a very long ..."},
		{"empty string", "", 10, ""},
		{"single char max", "hello", 4, "hell..."},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := truncate(tt.input, tt.maxLen)
			if got != tt.want {
				t.Errorf("truncate(%q, %d) = %q, want %q", tt.input, tt.maxLen, got, tt.want)
			}
		})
	}
}
