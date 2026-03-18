// Copyright (c) ElementumAI, Inc.
// SPDX-License-Identifier: MPL-2.0

package cmd

import (
	"testing"
)

func TestSkillToolsCommandRegistration(t *testing.T) {
	if skillToolsCmd.Use != "skill-tools" {
		t.Errorf("unexpected Use: %q, want %q", skillToolsCmd.Use, "skill-tools")
	}

	subcommands := map[string]bool{
		"list":       false,
		"create":     false,
		"update":     false,
		"delete":     false,
		"delete-all": false,
	}

	for _, sub := range skillToolsCmd.Commands() {
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

func TestGetSkillToolsCmd(t *testing.T) {
	cmd := GetSkillToolsCmd()
	if cmd == nil {
		t.Fatal("GetSkillToolsCmd returned nil")
	}
	if cmd != skillToolsCmd {
		t.Error("GetSkillToolsCmd should return skillToolsCmd")
	}
}

func TestSkillToolsCreateCommandRegistration(t *testing.T) {
	if skillToolsCreateCmd.Use != "create <skill-id-or-name>" {
		t.Errorf("unexpected Use: %q", skillToolsCreateCmd.Use)
	}

	if skillToolsCreateCmd.Args == nil {
		t.Error("Args should not be nil")
	}

	requiredFlags := []string{"type", "name"}
	for _, name := range requiredFlags {
		if skillToolsCreateCmd.Flags().Lookup(name) == nil {
			t.Errorf("missing required flag: %s", name)
		}
	}

	optionalFlags := []string{"description", "start-message", "automation-id", "target-id",
		"search-table-id", "target-agent-id", "query-description", "handle-description",
		"worker-task-prompt", "result-limit"}
	for _, name := range optionalFlags {
		if skillToolsCreateCmd.Flags().Lookup(name) == nil {
			t.Errorf("missing flag: %s", name)
		}
	}
}

func TestSkillToolsUpdateCommandRegistration(t *testing.T) {
	if skillToolsUpdateCmd.Use != "update <tool-id>" {
		t.Errorf("unexpected Use: %q", skillToolsUpdateCmd.Use)
	}

	if skillToolsUpdateCmd.Args == nil {
		t.Error("Args should not be nil")
	}

	flags := []string{"type", "name", "description", "start-message",
		"query-description", "handle-description", "worker-task-prompt"}
	for _, name := range flags {
		if skillToolsUpdateCmd.Flags().Lookup(name) == nil {
			t.Errorf("missing flag: %s", name)
		}
	}
}

func TestBuildSkillToolInput_Automation(t *testing.T) {
	input, err := buildSkillToolInput("automation", "Classify", "Classifies tickets", "", "auto-123", "", "", "", "", "", "", 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	auto, ok := input["automation"].(map[string]interface{})
	if !ok {
		t.Fatal("expected automation key in input")
	}
	if auto["name"] != "Classify" {
		t.Errorf("name = %q, want %q", auto["name"], "Classify")
	}
	if auto["description"] != "Classifies tickets" {
		t.Errorf("description = %q, want %q", auto["description"], "Classifies tickets")
	}
	if auto["automationId"] != "auto-123" {
		t.Errorf("automationId = %q, want %q", auto["automationId"], "auto-123")
	}
}

func TestBuildSkillToolInput_AutomationMissingID(t *testing.T) {
	_, err := buildSkillToolInput("automation", "Classify", "", "", "", "", "", "", "", "", "", 0)
	if err == nil {
		t.Fatal("expected error for missing automation-id")
	}
}

func TestBuildSkillToolInput_CreateRecord(t *testing.T) {
	input, err := buildSkillToolInput("create-record", "Create Ticket", "", "", "", "app-123", "", "", "", "", "", 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	cr, ok := input["createRecord"].(map[string]interface{})
	if !ok {
		t.Fatal("expected createRecord key in input")
	}
	if cr["aspectId"] != "app-123" {
		t.Errorf("aspectId = %q, want %q", cr["aspectId"], "app-123")
	}
}

func TestBuildSkillToolInput_CreateRecordMissingTarget(t *testing.T) {
	_, err := buildSkillToolInput("create-record", "Create", "", "", "", "", "", "", "", "", "", 0)
	if err == nil {
		t.Fatal("expected error for missing target-id")
	}
}

func TestBuildSkillToolInput_SearchRecords(t *testing.T) {
	input, err := buildSkillToolInput("search-records", "Find Tickets", "desc", "", "", "app-123", "", "", "Search for tickets", "", "", 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	sa, ok := input["searchAspect"].(map[string]interface{})
	if !ok {
		t.Fatal("expected searchAspect key in input")
	}
	if sa["aspectId"] != "app-123" {
		t.Errorf("aspectId = %q, want %q", sa["aspectId"], "app-123")
	}
	if sa["queryDescription"] != "Search for tickets" {
		t.Errorf("queryDescription = %q, want %q", sa["queryDescription"], "Search for tickets")
	}
	if sa["limit"] != 10 {
		t.Errorf("limit = %v, want 10", sa["limit"])
	}
}

func TestBuildSkillToolInput_SearchRecordsMissingFields(t *testing.T) {
	_, err := buildSkillToolInput("search-records", "Find", "", "", "", "", "", "", "query", "", "", 0)
	if err == nil {
		t.Fatal("expected error for missing target-id")
	}

	_, err = buildSkillToolInput("search-records", "Find", "", "", "", "app-123", "", "", "", "", "", 0)
	if err == nil {
		t.Fatal("expected error for missing query-description")
	}
}

func TestBuildSkillToolInput_UpdateRecord(t *testing.T) {
	input, err := buildSkillToolInput("update-record", "Update Ticket", "", "", "", "app-123", "", "", "", "Update the record", "", 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ur, ok := input["updateRecord"].(map[string]interface{})
	if !ok {
		t.Fatal("expected updateRecord key in input")
	}
	if ur["aspectId"] != "app-123" {
		t.Errorf("aspectId = %q, want %q", ur["aspectId"], "app-123")
	}
	if ur["handleDescription"] != "Update the record" {
		t.Errorf("handleDescription = %q, want %q", ur["handleDescription"], "Update the record")
	}
}

func TestBuildSkillToolInput_UpdateRecordMissingFields(t *testing.T) {
	_, err := buildSkillToolInput("update-record", "Update", "", "", "", "", "", "", "", "handle", "", 0)
	if err == nil {
		t.Fatal("expected error for missing target-id")
	}

	_, err = buildSkillToolInput("update-record", "Update", "", "", "", "app-123", "", "", "", "", "", 0)
	if err == nil {
		t.Fatal("expected error for missing handle-description")
	}
}

func TestBuildSkillToolInput_SearchTable(t *testing.T) {
	input, err := buildSkillToolInput("search-table", "Search KB", "", "", "", "app-123", "table-456", "", "Search knowledge base", "", "", 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	st, ok := input["searchTable"].(map[string]interface{})
	if !ok {
		t.Fatal("expected searchTable key in input")
	}
	if st["tableId"] != "table-456" {
		t.Errorf("tableId = %q, want %q", st["tableId"], "table-456")
	}
	if st["queryDescription"] != "Search knowledge base" {
		t.Errorf("queryDescription = %q, want %q", st["queryDescription"], "Search knowledge base")
	}
	if st["aspectId"] != "app-123" {
		t.Errorf("aspectId = %q, want %q", st["aspectId"], "app-123")
	}
	if st["limit"] != 5 {
		t.Errorf("limit = %v, want 5", st["limit"])
	}
}

func TestBuildSkillToolInput_SearchTableMissingFields(t *testing.T) {
	_, err := buildSkillToolInput("search-table", "Search", "", "", "", "", "", "", "query", "", "", 0)
	if err == nil {
		t.Fatal("expected error for missing search-table-id")
	}

	_, err = buildSkillToolInput("search-table", "Search", "", "", "", "", "table-123", "", "", "", "", 0)
	if err == nil {
		t.Fatal("expected error for missing query-description")
	}
}

func TestBuildSkillToolInput_RunAgent(t *testing.T) {
	input, err := buildSkillToolInput("run-agent", "Delegate", "", "", "", "", "", "agent-789", "", "", "Handle this", 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ra, ok := input["runAgent"].(map[string]interface{})
	if !ok {
		t.Fatal("expected runAgent key in input")
	}
	if ra["targetAgentId"] != "agent-789" {
		t.Errorf("targetAgentId = %q, want %q", ra["targetAgentId"], "agent-789")
	}
	if ra["workerTaskPrompt"] != "Handle this" {
		t.Errorf("workerTaskPrompt = %q, want %q", ra["workerTaskPrompt"], "Handle this")
	}
}

func TestBuildSkillToolInput_RunAgentMissingFields(t *testing.T) {
	_, err := buildSkillToolInput("run-agent", "Delegate", "", "", "", "", "", "", "", "", "prompt", 0)
	if err == nil {
		t.Fatal("expected error for missing target-agent-id")
	}

	_, err = buildSkillToolInput("run-agent", "Delegate", "", "", "", "", "", "agent-789", "", "", "", 0)
	if err == nil {
		t.Fatal("expected error for missing worker-task-prompt")
	}
}

func TestBuildSkillToolInput_InvalidType(t *testing.T) {
	_, err := buildSkillToolInput("invalid", "Name", "", "", "", "", "", "", "", "", "", 0)
	if err == nil {
		t.Fatal("expected error for invalid tool type")
	}
}

func TestBuildSkillToolInput_StartMessage(t *testing.T) {
	input, err := buildSkillToolInput("automation", "Name", "", "Processing...", "auto-123", "", "", "", "", "", "", 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	auto := input["automation"].(map[string]interface{})
	if auto["startMessage"] != "Processing..." {
		t.Errorf("startMessage = %q, want %q", auto["startMessage"], "Processing...")
	}
}

func TestBuildSkillToolInput_NoDescription(t *testing.T) {
	input, err := buildSkillToolInput("automation", "Name", "", "", "auto-123", "", "", "", "", "", "", 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	auto := input["automation"].(map[string]interface{})
	if _, ok := auto["description"]; ok {
		t.Error("description should not be set when empty")
	}
}

func TestBuildSkillToolInput_NoResultLimit(t *testing.T) {
	input, err := buildSkillToolInput("search-records", "Find", "", "", "", "app-123", "", "", "query", "", "", 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	sa := input["searchAspect"].(map[string]interface{})
	if _, ok := sa["limit"]; ok {
		t.Error("limit should not be set when 0")
	}
}

func TestBuildSkillToolUpdateInput_Automation(t *testing.T) {
	input, err := buildSkillToolUpdateInput("automation", "New Name", "New Desc", "", "", "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	auto, ok := input["automation"].(map[string]interface{})
	if !ok {
		t.Fatal("expected automation key in input")
	}
	if auto["name"] != "New Name" {
		t.Errorf("name = %q, want %q", auto["name"], "New Name")
	}
	if auto["description"] != "New Desc" {
		t.Errorf("description = %q, want %q", auto["description"], "New Desc")
	}
}

func TestBuildSkillToolUpdateInput_SearchRecords(t *testing.T) {
	input, err := buildSkillToolUpdateInput("search-records", "", "", "", "Updated query", "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	sa, ok := input["searchAspect"].(map[string]interface{})
	if !ok {
		t.Fatal("expected searchAspect key in input")
	}
	if sa["queryDescription"] != "Updated query" {
		t.Errorf("queryDescription = %q, want %q", sa["queryDescription"], "Updated query")
	}
	if _, ok := sa["name"]; ok {
		t.Error("name should not be set when empty")
	}
}

func TestBuildSkillToolUpdateInput_UpdateRecord(t *testing.T) {
	input, err := buildSkillToolUpdateInput("update-record", "", "", "", "", "New handle", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ur, ok := input["updateRecord"].(map[string]interface{})
	if !ok {
		t.Fatal("expected updateRecord key in input")
	}
	if ur["handleDescription"] != "New handle" {
		t.Errorf("handleDescription = %q, want %q", ur["handleDescription"], "New handle")
	}
}

func TestBuildSkillToolUpdateInput_RunAgent(t *testing.T) {
	input, err := buildSkillToolUpdateInput("run-agent", "", "", "", "", "", "New prompt")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ra, ok := input["runAgent"].(map[string]interface{})
	if !ok {
		t.Fatal("expected runAgent key in input")
	}
	if ra["workerTaskPrompt"] != "New prompt" {
		t.Errorf("workerTaskPrompt = %q, want %q", ra["workerTaskPrompt"], "New prompt")
	}
}

func TestBuildSkillToolUpdateInput_InvalidType(t *testing.T) {
	_, err := buildSkillToolUpdateInput("invalid", "Name", "", "", "", "", "")
	if err == nil {
		t.Fatal("expected error for invalid tool type")
	}
}

func TestBuildSkillToolUpdateInput_AllTypes(t *testing.T) {
	types := []struct {
		toolType string
		key      string
	}{
		{"automation", "automation"},
		{"create-record", "createRecord"},
		{"search-records", "searchAspect"},
		{"update-record", "updateRecord"},
		{"search-table", "searchTable"},
		{"run-agent", "runAgent"},
	}

	for _, tt := range types {
		t.Run(tt.toolType, func(t *testing.T) {
			input, err := buildSkillToolUpdateInput(tt.toolType, "Name", "", "", "", "", "")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if _, ok := input[tt.key]; !ok {
				t.Errorf("expected key %q in input for type %q", tt.key, tt.toolType)
			}
		})
	}
}
