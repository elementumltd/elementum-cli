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

package export

import (
	"strings"
	"testing"

	"github.com/elementumltd/elementum-cli/discovery"
)

func TestOrganizeBlocksExport_BasicGrouping(t *testing.T) {
	app := &discovery.App{
		ID:        "app-1",
		Name:      "Test App",
		Namespace: "test-app",
	}

	imports := []ImportBlock{
		{ResourceType: "elementum_app", ResourceName: "test_app", ID: "app-1"},
		{ResourceType: "elementum_text_field", ResourceName: "status", ID: "app-1:field-1"},
	}

	blocks := []*HCLBlock{
		NewResourceBlock("elementum_app", "test_app"),
		NewResourceBlock("elementum_text_field", "status"),
	}

	result := OrganizeBlocksExport(blocks, app, imports)

	if len(result.FileGroups) == 0 {
		t.Fatal("expected at least 1 file group")
	}

	// Both should be in app-test_app.tf (SanitizeName converts hyphen to underscore)
	found := false
	for _, fg := range result.FileGroups {
		if fg.FileName == "app-test_app.tf" {
			found = true
			if len(fg.Blocks) != 2 {
				t.Errorf("expected 2 blocks in app file, got %d", len(fg.Blocks))
			}
		}
	}
	if !found {
		t.Errorf("expected app-test_app.tf file group, got files:")
		for _, fg := range result.FileGroups {
			t.Errorf("  %s (%d blocks)", fg.FileName, len(fg.Blocks))
		}
	}
}

func TestOrganizeBlocksExport_AutomationGrouping(t *testing.T) {
	app := &discovery.App{
		ID:        "app-1",
		Name:      "Test App",
		Namespace: "test-app",
		Automations: []discovery.Automation{
			{
				ID:           "auto-1",
				Name:         "Process Ticket",
				HasPublished: true,
				Status:       "ACTIVE",
			},
		},
	}

	imports := []ImportBlock{
		{ResourceType: "elementum_automation", ResourceName: "process_ticket", ID: "auto-1"},
		{ResourceType: "elementum_record_created_trigger", ResourceName: "on_create", ID: "auto-1:trigger-1"},
		{ResourceType: "elementum_message_task", ResourceName: "notify", ID: "auto-1:task-1"},
	}

	blocks := []*HCLBlock{
		NewResourceBlock("elementum_automation", "process_ticket"),
		NewResourceBlock("elementum_record_created_trigger", "on_create"),
		NewResourceBlock("elementum_message_task", "notify"),
	}

	result := OrganizeBlocksExport(blocks, app, imports)

	// Automation should be in its own file
	foundAuto := false
	for _, fg := range result.FileGroups {
		if fg.FileName == "automation-process_ticket.tf" {
			foundAuto = true
			// Automation resource goes here
			break
		}
	}
	if !foundAuto {
		// Check what files we got
		for _, fg := range result.FileGroups {
			t.Logf("file: %s, blocks: %d", fg.FileName, len(fg.Blocks))
		}
		t.Error("expected automation-process_ticket.tf file group")
	}
}

func TestDeduplicateBlocks(t *testing.T) {
	blocks := []*HCLBlock{
		{Type: "resource", Labels: []string{"elementum_app", "my_app"}, Body: &HCLBody{}},
		{Type: "resource", Labels: []string{"elementum_field", "status"}, Body: &HCLBody{}},
		{Type: "resource", Labels: []string{"elementum_app", "my_app"}, Body: &HCLBody{}}, // duplicate
		{Type: "resource", Labels: []string{"elementum_field", "name"}, Body: &HCLBody{}},
	}

	result := DeduplicateBlocks(blocks)

	if len(result) != 3 {
		t.Fatalf("expected 3 blocks after dedup, got %d", len(result))
	}
}

func TestSortBlocksByType(t *testing.T) {
	blocks := []*HCLBlock{
		{Type: "resource", Labels: []string{"elementum_field", "zzz"}, Body: &HCLBody{}},
		{Type: "resource", Labels: []string{"elementum_app", "aaa"}, Body: &HCLBody{}},
		{Type: "resource", Labels: []string{"elementum_field", "bbb"}, Body: &HCLBody{}},
	}

	SortBlocksByType(blocks)

	expected := []string{"aaa", "bbb", "zzz"}
	for i, block := range blocks {
		if block.Labels[1] != expected[i] {
			t.Errorf("index %d: expected %q, got %q", i, expected[i], block.Labels[1])
		}
	}
}

func TestBlockFileGroup_Serialize(t *testing.T) {
	b := NewResourceBlock("elementum_app", "my_app")
	b.SetAttr("name", Str("My App"))

	fg := BlockFileGroup{
		FileName: "app.tf",
		Blocks:   []*HCLBlock{b},
		Comment:  "Main app config",
	}

	output := fg.Serialize()
	if output == "" {
		t.Fatal("expected non-empty serialized output")
	}
	if !containsSubstring(output, "# Main app config") {
		t.Error("expected comment in output")
	}
	if !containsSubstring(output, `resource "elementum_app" "my_app"`) {
		t.Error("expected resource declaration in output")
	}
}

func TestSanitizeFileName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"My App", "my_app"},
		{"app/sub", "app_sub"},
		{"Test App (v2)", "test_app_v2"},
	}

	for _, tt := range tests {
		got := SanitizeFileName(tt.input)
		if got != tt.expected {
			t.Errorf("SanitizeFileName(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestOrganizeBlocksExport_TableSearchTableRouting(t *testing.T) {
	app := &discovery.App{
		ID:        "app-1",
		Name:      "Test App",
		Namespace: "test-app",
	}

	imports := []ImportBlock{
		{ResourceType: "elementum_table_search_table", ResourceName: "customer_data_description", ID: "table-123/st-1"},
	}

	blocks := []*HCLBlock{
		NewResourceBlock("elementum_table_search_table", "customer_data_description"),
	}

	result := OrganizeBlocksExport(blocks, app, imports)

	if len(result.FileGroups) == 0 {
		t.Fatal("expected at least 1 file group")
	}

	// Table search tables should be routed to table-<resource_name>.tf
	found := false
	for _, fg := range result.FileGroups {
		if fg.FileName == "table-customer_data_description.tf" {
			found = true
			if len(fg.Blocks) != 1 {
				t.Errorf("expected 1 block in table file, got %d", len(fg.Blocks))
			}
		}
	}
	if !found {
		t.Errorf("expected table-customer_data_description.tf file group, got files:")
		for _, fg := range result.FileGroups {
			t.Errorf("  %s (%d blocks)", fg.FileName, len(fg.Blocks))
		}
	}
}

func containsSubstring(s, sub string) bool {
	return strings.Contains(s, sub)
}

// TestToMultiFileExport_DataSourceRouting verifies that data source blocks
// (Type == "data") are merged into DataSourcesFile, not lost to AppFiles.
// This is a regression test for the bug where access policy data sources
// (elementum_user, elementum_group) were generated but never written to data.tf.
func TestToMultiFileExport_DataSourceRouting(t *testing.T) {
	app := &discovery.App{
		ID:        "app-1",
		Name:      "Test App",
		Namespace: "test-app",
	}

	imports := []ImportBlock{}

	// Create data source blocks (Type = "data" instead of "resource")
	userBlock := &HCLBlock{
		Type:   "data",
		Labels: []string{"elementum_user", "john_doe"},
		Body:   &HCLBody{},
	}
	userBlock.SetAttr("email", Str("john@example.com"))

	groupBlock := &HCLBlock{
		Type:   "data",
		Labels: []string{"elementum_group", "admins"},
		Body:   &HCLBody{},
	}
	groupBlock.SetAttr("name", Str("Administrators"))

	blocks := []*HCLBlock{userBlock, groupBlock}

	result := OrganizeBlocksExport(blocks, app, imports)

	// Verify blocks are routed to data.tf
	dataFileFound := false
	for _, fg := range result.FileGroups {
		if fg.FileName == "data.tf" {
			dataFileFound = true
			if len(fg.Blocks) != 2 {
				t.Errorf("expected 2 blocks in data.tf, got %d", len(fg.Blocks))
			}
		}
	}
	if !dataFileFound {
		t.Error("expected data.tf file group")
		for _, fg := range result.FileGroups {
			t.Logf("  found: %s (%d blocks)", fg.FileName, len(fg.Blocks))
		}
	}

	// Convert to MultiFileExport and verify DataSourcesFile is populated
	export := result.ToMultiFileExport()

	if export.DataSourcesFile == "" {
		t.Error("expected DataSourcesFile to be populated")
	}

	if !strings.Contains(export.DataSourcesFile, `data "elementum_user" "john_doe"`) {
		t.Error("expected user data source in DataSourcesFile")
	}
	if !strings.Contains(export.DataSourcesFile, `data "elementum_group" "admins"`) {
		t.Error("expected group data source in DataSourcesFile")
	}

	// Verify data sources are NOT in AppFiles
	for _, fg := range export.AppFiles {
		content := strings.Join(fg.Resources, "\n")
		if strings.Contains(content, "elementum_user") || strings.Contains(content, "elementum_group") {
			t.Errorf("data sources should NOT be in AppFiles, found in %s", fg.FileName)
		}
	}
}

// TestToMultiFileExport_DataSourceMerging verifies that when DataSourcesFile
// already has content, additional data sources from the IR pipeline are merged
// rather than overwritten. This mirrors the apps.go scenario where category/cloudlink
// data sources are added after ToMultiFileExport().
func TestToMultiFileExport_DataSourceMerging(t *testing.T) {
	app := &discovery.App{
		ID:        "app-1",
		Name:      "Test App",
		Namespace: "test-app",
	}

	imports := []ImportBlock{}

	// Create data source blocks
	userBlock := &HCLBlock{
		Type:   "data",
		Labels: []string{"elementum_user", "admin"},
		Body:   &HCLBody{},
	}
	userBlock.SetAttr("email", Str("admin@example.com"))

	blocks := []*HCLBlock{userBlock}
	result := OrganizeBlocksExport(blocks, app, imports)

	// Pre-populate DataSourcesFile as apps.go does
	result.DataSourcesFile = `data "elementum_category" "existing" {
  name = "Existing Category"
}
`

	// Convert to MultiFileExport - this should MERGE, not overwrite
	export := result.ToMultiFileExport()

	// Verify both existing and new data sources are present
	if !strings.Contains(export.DataSourcesFile, `data "elementum_category" "existing"`) {
		t.Error("expected existing category data source to be preserved")
	}
	if !strings.Contains(export.DataSourcesFile, `data "elementum_user" "admin"`) {
		t.Error("expected user data source to be merged into DataSourcesFile")
	}
}

// TestOrganizeBlocksExport_TasksUseWorkflowID is a regression test to ensure tasks
// are correctly organized into their automation files when using workflow IDs.
// The task import ID format is {workflow_id}:{task_id}, NOT {automation_id}:{task_id}.
// This was a bug where tasks ended up in automations.tf instead of automation-{name}.tf.
func TestOrganizeBlocksExport_TasksUseWorkflowID(t *testing.T) {
	// Key: automation.ID != automation.WorkflowID
	// The task import ID uses workflow_id, not automation_id
	app := &discovery.App{
		ID:        "app-1",
		Name:      "Test App",
		Namespace: "test-app",
		Automations: []discovery.Automation{
			{
				ID:           "auto-123",     // Automation ID
				WorkflowID:   "workflow-456", // Workflow ID - DIFFERENT from auto ID
				Name:         "Process Ticket",
				HasPublished: true,
				Status:       "ACTIVE",
			},
		},
	}

	// Task import ID uses workflow_id (workflow-456), not automation_id (auto-123)
	imports := []ImportBlock{
		{ResourceType: "elementum_automation", ResourceName: "process_ticket", ID: "app-1:auto-123"},
		{ResourceType: "elementum_on_demand_trigger", ResourceName: "process_ticket_on_demand_0", ID: "auto-123:trigger-1"},
		{ResourceType: "elementum_variable_task", ResourceName: "process_ticket_set_status", ID: "workflow-456:task-1"},
		{ResourceType: "elementum_update_variable_task", ResourceName: "process_ticket_update_status", ID: "workflow-456:task-2"},
		{ResourceType: "elementum_api_task", ResourceName: "process_ticket_call_api", ID: "workflow-456:task-3"},
	}

	blocks := []*HCLBlock{
		NewResourceBlock("elementum_automation", "process_ticket"),
		NewResourceBlock("elementum_on_demand_trigger", "process_ticket_on_demand_0"),
		NewResourceBlock("elementum_variable_task", "process_ticket_set_status"),
		NewResourceBlock("elementum_update_variable_task", "process_ticket_update_status"),
		NewResourceBlock("elementum_api_task", "process_ticket_call_api"),
	}

	result := OrganizeBlocksExport(blocks, app, imports)

	// Find the automation file
	var autoFile *BlockFileGroup
	for i := range result.FileGroups {
		if result.FileGroups[i].FileName == "automation-process_ticket.tf" {
			autoFile = &result.FileGroups[i]
			break
		}
	}

	if autoFile == nil {
		t.Fatal("expected automation-process_ticket.tf file group")
	}

	// All 5 blocks should be in the automation file (automation, trigger, 3 tasks)
	if len(autoFile.Blocks) != 5 {
		t.Errorf("expected 5 blocks in automation file, got %d", len(autoFile.Blocks))
		t.Log("Blocks in automation file:")
		for _, b := range autoFile.Blocks {
			t.Logf("  %s.%s", b.Labels[0], b.Labels[1])
		}
		t.Log("All file groups:")
		for _, fg := range result.FileGroups {
			t.Logf("  %s: %d blocks", fg.FileName, len(fg.Blocks))
		}
	}

	// Verify tasks are NOT in automations.tf fallback
	for _, fg := range result.FileGroups {
		if fg.FileName == "automations.tf" {
			// automations.tf should not exist or be empty for this test
			t.Errorf("tasks should NOT be in automations.tf fallback, found %d blocks", len(fg.Blocks))
			for _, b := range fg.Blocks {
				t.Logf("  unexpected: %s.%s", b.Labels[0], b.Labels[1])
			}
		}
	}
}
