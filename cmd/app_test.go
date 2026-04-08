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

func TestAppCommandRegistration(t *testing.T) {
	if appsCmd.Use != "apps" {
		t.Errorf("unexpected Use: %q, want %q", appsCmd.Use, "apps")
	}

	subcommands := map[string]bool{
		"list":   false,
		"show":   false,
		"export": false,
		"create": false,
		"delete": false,
		"update": false,
	}

	for _, sub := range appsCmd.Commands() {
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

func TestAppCreateCommandRegistration(t *testing.T) {
	if appCreateCmd.Use != "create" {
		t.Errorf("unexpected Use: %q", appCreateCmd.Use)
	}

	requiredFlags := []string{"name", "namespace", "category"}
	for _, name := range requiredFlags {
		f := appCreateCmd.Flags().Lookup(name)
		if f == nil {
			t.Errorf("missing flag: %s", name)
			continue
		}
	}

	optionalFlags := []string{"type", "description", "dry-run"}
	for _, name := range optionalFlags {
		if appCreateCmd.Flags().Lookup(name) == nil {
			t.Errorf("missing flag: %s", name)
		}
	}

	typeFlag := appCreateCmd.Flags().Lookup("type")
	if typeFlag != nil && typeFlag.DefValue != "app" {
		t.Errorf("type flag default = %q, want %q", typeFlag.DefValue, "app")
	}
}

func TestAppDeleteCommandRegistration(t *testing.T) {
	if appDeleteCmd.Use != "delete <namespace-or-id>" {
		t.Errorf("unexpected Use: %q", appDeleteCmd.Use)
	}

	if appDeleteCmd.Args == nil {
		t.Error("Args should not be nil")
	}

	flags := []string{"force", "dry-run"}
	for _, name := range flags {
		if appDeleteCmd.Flags().Lookup(name) == nil {
			t.Errorf("missing flag: %s", name)
		}
	}

	if appDeleteCmd.Flags().ShorthandLookup("f") == nil {
		t.Error("missing short flag -f for --force")
	}
}

func TestAppUpdateCommandRegistration(t *testing.T) {
	if appUpdateCmd.Use != "update <namespace-or-id>" {
		t.Errorf("unexpected Use: %q", appUpdateCmd.Use)
	}

	if appUpdateCmd.Args == nil {
		t.Error("Args should not be nil")
	}

	flags := []string{"name", "description"}
	for _, name := range flags {
		if appUpdateCmd.Flags().Lookup(name) == nil {
			t.Errorf("missing flag: %s", name)
		}
	}
}

func TestGetAppsCmd(t *testing.T) {
	cmd := GetAppsCmd()
	if cmd == nil {
		t.Fatal("GetAppsCmd returned nil")
	}
	if cmd != appsCmd {
		t.Error("GetAppsCmd should return appsCmd")
	}
}
