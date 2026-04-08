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

func TestSkillsCommandRegistration(t *testing.T) {
	if skillsCmd.Use != "skills" {
		t.Errorf("unexpected Use: %q, want %q", skillsCmd.Use, "skills")
	}

	subcommands := map[string]bool{
		"list":   false,
		"create": false,
		"show":   false,
		"update": false,
		"delete": false,
	}

	for _, sub := range skillsCmd.Commands() {
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

func TestGetSkillsCmd(t *testing.T) {
	cmd := GetSkillsCmd()
	if cmd == nil {
		t.Fatal("GetSkillsCmd returned nil")
	}
	if cmd != skillsCmd {
		t.Error("GetSkillsCmd should return skillsCmd")
	}
}

func TestSkillsListCommandRegistration(t *testing.T) {
	if skillsListCmd.Use != "list [app-namespace]" {
		t.Errorf("unexpected Use: %q", skillsListCmd.Use)
	}

	if skillsListCmd.RunE == nil {
		t.Error("RunE should not be nil")
	}
}

func TestSkillsCreateCommandRegistration(t *testing.T) {
	if skillsCreateCmd.Use != "create" {
		t.Errorf("unexpected Use: %q, want %q", skillsCreateCmd.Use, "create")
	}

	requiredFlags := []string{"name", "description", "instructions"}
	for _, name := range requiredFlags {
		f := skillsCreateCmd.Flags().Lookup(name)
		if f == nil {
			t.Errorf("missing required flag: %s", name)
		}
	}

	optionalFlags := []string{"owner-id", "owner-type", "app", "status", "dry-run"}
	for _, name := range optionalFlags {
		if skillsCreateCmd.Flags().Lookup(name) == nil {
			t.Errorf("missing flag: %s", name)
		}
	}

	ownerTypeFlag := skillsCreateCmd.Flags().Lookup("owner-type")
	if ownerTypeFlag != nil && ownerTypeFlag.DefValue != "APP_ASPECT" {
		t.Errorf("owner-type flag default = %q, want %q", ownerTypeFlag.DefValue, "APP_ASPECT")
	}

	statusFlag := skillsCreateCmd.Flags().Lookup("status")
	if statusFlag != nil && statusFlag.DefValue != "ACTIVE" {
		t.Errorf("status flag default = %q, want %q", statusFlag.DefValue, "ACTIVE")
	}
}

func TestSkillsShowCommandRegistration(t *testing.T) {
	if skillsShowCmd.Use != "show <skill-id-or-name>" {
		t.Errorf("unexpected Use: %q", skillsShowCmd.Use)
	}

	if skillsShowCmd.Args == nil {
		t.Error("Args should not be nil")
	}

	if skillsShowCmd.RunE == nil {
		t.Error("RunE should not be nil")
	}
}

func TestSkillsUpdateCommandRegistration(t *testing.T) {
	if skillsUpdateCmd.Use != "update <skill-id-or-name>" {
		t.Errorf("unexpected Use: %q", skillsUpdateCmd.Use)
	}

	if skillsUpdateCmd.Args == nil {
		t.Error("Args should not be nil")
	}

	flags := []string{"name", "description", "instructions", "status"}
	for _, name := range flags {
		if skillsUpdateCmd.Flags().Lookup(name) == nil {
			t.Errorf("missing flag: %s", name)
		}
	}
}

func TestSkillsDeleteCommandRegistration(t *testing.T) {
	if skillsDeleteCmd.Use != "delete <skill-id-or-name>" {
		t.Errorf("unexpected Use: %q", skillsDeleteCmd.Use)
	}

	if skillsDeleteCmd.Args == nil {
		t.Error("Args should not be nil")
	}
}
