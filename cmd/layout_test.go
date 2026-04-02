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

func TestLayoutCommandRegistration(t *testing.T) {
	if layoutsCmd.Use != "layouts" {
		t.Errorf("unexpected Use: %q, want %q", layoutsCmd.Use, "layouts")
	}

	subcommands := map[string]bool{
		"list":   false,
		"show":   false,
		"create": false,
		"delete": false,
		"update": false,
	}

	for _, sub := range layoutsCmd.Commands() {
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

func TestLayoutCreateCommandRegistration(t *testing.T) {
	if layoutCreateCmd.Use != "create <namespace>" {
		t.Errorf("unexpected Use: %q", layoutCreateCmd.Use)
	}

	if layoutCreateCmd.Args == nil {
		t.Error("Args should not be nil")
	}

	flags := []string{"name", "color", "icon", "dry-run"}
	for _, name := range flags {
		if layoutCreateCmd.Flags().Lookup(name) == nil {
			t.Errorf("missing flag: %s", name)
		}
	}
}

func TestLayoutDeleteCommandRegistration(t *testing.T) {
	if layoutDeleteCmd.Use != "delete <namespace> <stage-name>" {
		t.Errorf("unexpected Use: %q", layoutDeleteCmd.Use)
	}

	if layoutDeleteCmd.Args == nil {
		t.Error("Args should not be nil")
	}

	flags := []string{"force", "dry-run"}
	for _, name := range flags {
		if layoutDeleteCmd.Flags().Lookup(name) == nil {
			t.Errorf("missing flag: %s", name)
		}
	}

	if layoutDeleteCmd.Flags().ShorthandLookup("f") == nil {
		t.Error("missing short flag -f for --force")
	}
}

func TestLayoutUpdateCommandRegistration(t *testing.T) {
	if layoutUpdateCmd.Use != "update <namespace> <stage-name>" {
		t.Errorf("unexpected Use: %q", layoutUpdateCmd.Use)
	}

	if layoutUpdateCmd.Args == nil {
		t.Error("Args should not be nil")
	}

	flags := []string{"name", "color", "icon", "order"}
	for _, name := range flags {
		if layoutUpdateCmd.Flags().Lookup(name) == nil {
			t.Errorf("missing flag: %s", name)
		}
	}

	orderFlag := layoutUpdateCmd.Flags().Lookup("order")
	if orderFlag != nil && orderFlag.DefValue != "-1" {
		t.Errorf("order flag default = %q, want %q", orderFlag.DefValue, "-1")
	}
}

func TestLayoutShowCommandRegistration(t *testing.T) {
	if layoutShowCmd.Use != "show <namespace> <stage-name>" {
		t.Errorf("unexpected Use: %q", layoutShowCmd.Use)
	}

	if layoutShowCmd.Args == nil {
		t.Error("Args should not be nil")
	}
}

func TestLayoutListCommandRegistration(t *testing.T) {
	if layoutListCmd.Use != "list <namespace>" {
		t.Errorf("unexpected Use: %q", layoutListCmd.Use)
	}

	if layoutListCmd.Args == nil {
		t.Error("Args should not be nil")
	}
}

func TestGetLayoutsCmd(t *testing.T) {
	cmd := GetLayoutsCmd()
	if cmd == nil {
		t.Fatal("GetLayoutsCmd returned nil")
	}
	if cmd != layoutsCmd {
		t.Error("GetLayoutsCmd should return layoutsCmd")
	}
}
