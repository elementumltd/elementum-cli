// Copyright (c) ElementumAI, Inc.
// SPDX-License-Identifier: MPL-2.0

package cmd

import (
	"testing"
)

func TestAgentToolsCommandRegistration(t *testing.T) {
	if agentToolsCmd.Use != "agent-tools" {
		t.Errorf("unexpected Use: %q, want %q", agentToolsCmd.Use, "agent-tools")
	}

	subcommands := map[string]bool{
		"list":   false,
		"create": false,
		"update": false,
		"delete": false,
	}

	for _, sub := range agentToolsCmd.Commands() {
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

func TestGetAgentToolsCmd(t *testing.T) {
	cmd := GetAgentToolsCmd()
	if cmd == nil {
		t.Fatal("GetAgentToolsCmd returned nil")
	}
	if cmd != agentToolsCmd {
		t.Error("GetAgentToolsCmd should return agentToolsCmd")
	}
}

func TestAgentToolsCreateCommandRegistration(t *testing.T) {
	if agentToolsCreateCmd.Use != "create <agent-name-or-id>" {
		t.Errorf("unexpected Use: %q", agentToolsCreateCmd.Use)
	}

	if agentToolsCreateCmd.Args == nil {
		t.Error("Args should not be nil")
	}

	requiredFlags := []string{"type", "name"}
	for _, name := range requiredFlags {
		if agentToolsCreateCmd.Flags().Lookup(name) == nil {
			t.Errorf("missing required flag: %s", name)
		}
	}

	optionalFlags := []string{"description", "start-message", "automation-id", "target-id",
		"related-target-id", "search-table-id", "target-agent-id", "query-description",
		"handle-description", "worker-task-prompt", "mcp-tool-name", "server-url", "result-limit"}
	for _, name := range optionalFlags {
		if agentToolsCreateCmd.Flags().Lookup(name) == nil {
			t.Errorf("missing flag: %s", name)
		}
	}
}

func TestAgentToolsUpdateCommandRegistration(t *testing.T) {
	if agentToolsUpdateCmd.Use != "update <tool-id>" {
		t.Errorf("unexpected Use: %q", agentToolsUpdateCmd.Use)
	}

	if agentToolsUpdateCmd.Args == nil {
		t.Error("Args should not be nil")
	}

	flags := []string{"type", "name", "description", "start-message",
		"query-description", "handle-description", "worker-task-prompt"}
	for _, name := range flags {
		if agentToolsUpdateCmd.Flags().Lookup(name) == nil {
			t.Errorf("missing flag: %s", name)
		}
	}
}

// buildAgentToolInput params: toolType, name, description, startMessage, automationID,
// targetID, relatedTargetID, searchTableID, targetAgentID, queryDescription,
// handleDescription, workerTaskPrompt, mcpToolName, serverURL string, resultLimit int

func TestBuildAgentToolInput_RunAutomation(t *testing.T) {
	input, err := buildAgentToolInput("run-automation", "Classify", "Classifies tickets", "", "auto-123", "", "", "", "", "", "", "", "", "", 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ew, ok := input["executeWorkflow"].(map[string]interface{})
	if !ok {
		t.Fatal("expected executeWorkflow key in input")
	}
	if ew["name"] != "Classify" {
		t.Errorf("name = %q, want %q", ew["name"], "Classify")
	}
	if ew["automationId"] != "auto-123" {
		t.Errorf("automationId = %q, want %q", ew["automationId"], "auto-123")
	}
}

func TestBuildAgentToolInput_RunAutomationMissingID(t *testing.T) {
	_, err := buildAgentToolInput("run-automation", "Classify", "", "", "", "", "", "", "", "", "", "", "", "", 0)
	if err == nil {
		t.Fatal("expected error for missing automation-id")
	}
}

func TestBuildAgentToolInput_CreateRecord(t *testing.T) {
	input, err := buildAgentToolInput("create-record", "Create Ticket", "", "", "", "app-123", "", "", "", "", "", "", "", "", 0)
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

func TestBuildAgentToolInput_CreateRecordMissingTarget(t *testing.T) {
	_, err := buildAgentToolInput("create-record", "Create", "", "", "", "", "", "", "", "", "", "", "", "", 0)
	if err == nil {
		t.Fatal("expected error for missing target-id")
	}
}

func TestBuildAgentToolInput_SearchRecords(t *testing.T) {
	input, err := buildAgentToolInput("search-records", "Find", "desc", "", "", "app-123", "", "", "", "Search tickets", "", "", "", "", 10)
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
	if sa["queryDescription"] != "Search tickets" {
		t.Errorf("queryDescription = %q, want %q", sa["queryDescription"], "Search tickets")
	}
	if sa["limit"] != 10 {
		t.Errorf("limit = %v, want 10", sa["limit"])
	}
}

func TestBuildAgentToolInput_SearchRecordsMissingFields(t *testing.T) {
	_, err := buildAgentToolInput("search-records", "Find", "", "", "", "", "", "", "", "query", "", "", "", "", 0)
	if err == nil {
		t.Fatal("expected error for missing target-id")
	}

	_, err = buildAgentToolInput("search-records", "Find", "", "", "", "app-123", "", "", "", "", "", "", "", "", 0)
	if err == nil {
		t.Fatal("expected error for missing query-description")
	}
}

func TestBuildAgentToolInput_UpdateRecord(t *testing.T) {
	input, err := buildAgentToolInput("update-record", "Update", "", "", "", "app-123", "", "", "", "", "Update the record", "", "", "", 0)
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

func TestBuildAgentToolInput_UpdateRecordMissingFields(t *testing.T) {
	_, err := buildAgentToolInput("update-record", "Update", "", "", "", "", "", "", "", "", "handle", "", "", "", 0)
	if err == nil {
		t.Fatal("expected error for missing target-id")
	}

	_, err = buildAgentToolInput("update-record", "Update", "", "", "", "app-123", "", "", "", "", "", "", "", "", 0)
	if err == nil {
		t.Fatal("expected error for missing handle-description")
	}
}

func TestBuildAgentToolInput_RelateRecord(t *testing.T) {
	input, err := buildAgentToolInput("relate-record", "Link KB", "", "", "", "app-123", "elem-456", "", "", "", "", "", "", "", 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	rr, ok := input["relateRecord"].(map[string]interface{})
	if !ok {
		t.Fatal("expected relateRecord key in input")
	}
	if rr["aspectId"] != "app-123" {
		t.Errorf("aspectId = %q, want %q", rr["aspectId"], "app-123")
	}
	if rr["relatedAspectId"] != "elem-456" {
		t.Errorf("relatedAspectId = %q, want %q", rr["relatedAspectId"], "elem-456")
	}
}

func TestBuildAgentToolInput_RelateRecordMissingFields(t *testing.T) {
	_, err := buildAgentToolInput("relate-record", "Link", "", "", "", "", "elem-456", "", "", "", "", "", "", "", 0)
	if err == nil {
		t.Fatal("expected error for missing target-id")
	}

	_, err = buildAgentToolInput("relate-record", "Link", "", "", "", "app-123", "", "", "", "", "", "", "", "", 0)
	if err == nil {
		t.Fatal("expected error for missing related-target-id")
	}
}

func TestBuildAgentToolInput_SearchTable(t *testing.T) {
	input, err := buildAgentToolInput("search-table", "Search KB", "", "", "", "app-123", "", "table-789", "", "Search knowledge base", "", "", "", "", 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	st, ok := input["searchTable"].(map[string]interface{})
	if !ok {
		t.Fatal("expected searchTable key in input")
	}
	if st["tableId"] != "table-789" {
		t.Errorf("tableId = %q, want %q", st["tableId"], "table-789")
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

func TestBuildAgentToolInput_SearchTableMissingFields(t *testing.T) {
	_, err := buildAgentToolInput("search-table", "Search", "", "", "", "", "", "", "", "query", "", "", "", "", 0)
	if err == nil {
		t.Fatal("expected error for missing search-table-id")
	}

	_, err = buildAgentToolInput("search-table", "Search", "", "", "", "", "", "table-123", "", "", "", "", "", "", 0)
	if err == nil {
		t.Fatal("expected error for missing query-description")
	}
}

func TestBuildAgentToolInput_RunAgent(t *testing.T) {
	input, err := buildAgentToolInput("run-agent", "Delegate", "", "", "", "", "", "", "agent-111", "", "", "Handle this", "", "", 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ra, ok := input["runAgent"].(map[string]interface{})
	if !ok {
		t.Fatal("expected runAgent key in input")
	}
	if ra["targetAgentId"] != "agent-111" {
		t.Errorf("targetAgentId = %q, want %q", ra["targetAgentId"], "agent-111")
	}
	if ra["workerTaskPrompt"] != "Handle this" {
		t.Errorf("workerTaskPrompt = %q, want %q", ra["workerTaskPrompt"], "Handle this")
	}
}

func TestBuildAgentToolInput_RunAgentMissingFields(t *testing.T) {
	_, err := buildAgentToolInput("run-agent", "Delegate", "", "", "", "", "", "", "", "", "", "prompt", "", "", 0)
	if err == nil {
		t.Fatal("expected error for missing target-agent-id")
	}

	_, err = buildAgentToolInput("run-agent", "Delegate", "", "", "", "", "", "", "agent-111", "", "", "", "", "", 0)
	if err == nil {
		t.Fatal("expected error for missing worker-task-prompt")
	}
}

func TestBuildAgentToolInput_MCP(t *testing.T) {
	input, err := buildAgentToolInput("mcp", "External API", "Calls API", "", "", "", "", "", "", "", "", "", "api_call", "https://example.com/mcp", 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	mcp, ok := input["mcp"].(map[string]interface{})
	if !ok {
		t.Fatal("expected mcp key in input")
	}
	if mcp["mcpToolName"] != "api_call" {
		t.Errorf("mcpToolName = %q, want %q", mcp["mcpToolName"], "api_call")
	}
	if mcp["serverUrl"] != "https://example.com/mcp" {
		t.Errorf("serverUrl = %q, want %q", mcp["serverUrl"], "https://example.com/mcp")
	}
}

func TestBuildAgentToolInput_MCPMissingFields(t *testing.T) {
	_, err := buildAgentToolInput("mcp", "MCP", "", "", "", "", "", "", "", "", "", "", "", "https://example.com", 0)
	if err == nil {
		t.Fatal("expected error for missing mcp-tool-name")
	}

	_, err = buildAgentToolInput("mcp", "MCP", "", "", "", "", "", "", "", "", "", "", "tool", "", 0)
	if err == nil {
		t.Fatal("expected error for missing server-url")
	}
}

func TestBuildAgentToolInput_InvalidType(t *testing.T) {
	_, err := buildAgentToolInput("invalid", "Name", "", "", "", "", "", "", "", "", "", "", "", "", 0)
	if err == nil {
		t.Fatal("expected error for invalid tool type")
	}
}

func TestBuildAgentToolInput_StartMessage(t *testing.T) {
	input, err := buildAgentToolInput("run-automation", "Name", "", "Processing...", "auto-123", "", "", "", "", "", "", "", "", "", 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ew := input["executeWorkflow"].(map[string]interface{})
	if ew["startMessage"] != "Processing..." {
		t.Errorf("startMessage = %q, want %q", ew["startMessage"], "Processing...")
	}
}

func TestBuildAgentToolInput_NoDescription(t *testing.T) {
	input, err := buildAgentToolInput("run-automation", "Name", "", "", "auto-123", "", "", "", "", "", "", "", "", "", 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ew := input["executeWorkflow"].(map[string]interface{})
	if _, ok := ew["description"]; ok {
		t.Error("description should not be set when empty")
	}
}

func TestBuildAgentToolInput_CaseInsensitiveType(t *testing.T) {
	input, err := buildAgentToolInput("RUN-AUTOMATION", "Name", "", "", "auto-123", "", "", "", "", "", "", "", "", "", 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := input["executeWorkflow"]; !ok {
		t.Error("expected executeWorkflow key for RUN-AUTOMATION (case insensitive)")
	}
}

func TestBuildAgentToolUpdateInput_AllTypes(t *testing.T) {
	types := []struct {
		toolType string
		key      string
	}{
		{"run-automation", "executeWorkflow"},
		{"create-record", "createRecord"},
		{"search-records", "searchAspect"},
		{"update-record", "updateRecord"},
		{"relate-record", "relateRecord"},
		{"search-table", "searchTable"},
		{"run-agent", "runAgent"},
		{"mcp", "mcp"},
	}

	for _, tt := range types {
		t.Run(tt.toolType, func(t *testing.T) {
			input, err := buildAgentToolUpdateInput(tt.toolType, "Name", "", "", "", "", "")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if _, ok := input[tt.key]; !ok {
				t.Errorf("expected key %q in input for type %q", tt.key, tt.toolType)
			}
		})
	}
}

func TestBuildAgentToolUpdateInput_InvalidType(t *testing.T) {
	_, err := buildAgentToolUpdateInput("invalid", "Name", "", "", "", "", "")
	if err == nil {
		t.Fatal("expected error for invalid tool type")
	}
}

func TestBuildAgentToolUpdateInput_SearchRecordsQueryDescription(t *testing.T) {
	input, err := buildAgentToolUpdateInput("search-records", "", "", "", "New query", "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	sa := input["searchAspect"].(map[string]interface{})
	if sa["queryDescription"] != "New query" {
		t.Errorf("queryDescription = %q, want %q", sa["queryDescription"], "New query")
	}
	if _, ok := sa["name"]; ok {
		t.Error("name should not be set when empty")
	}
}

func TestBuildAgentToolUpdateInput_UpdateRecordHandleDescription(t *testing.T) {
	input, err := buildAgentToolUpdateInput("update-record", "", "", "", "", "New handle", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ur := input["updateRecord"].(map[string]interface{})
	if ur["handleDescription"] != "New handle" {
		t.Errorf("handleDescription = %q, want %q", ur["handleDescription"], "New handle")
	}
}

func TestBuildAgentToolUpdateInput_RunAgentWorkerPrompt(t *testing.T) {
	input, err := buildAgentToolUpdateInput("run-agent", "", "", "", "", "", "New prompt")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ra := input["runAgent"].(map[string]interface{})
	if ra["workerTaskPrompt"] != "New prompt" {
		t.Errorf("workerTaskPrompt = %q, want %q", ra["workerTaskPrompt"], "New prompt")
	}
}
