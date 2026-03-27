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

func TestSanitizeName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "simple lowercase",
			input: "myapp",
			want:  "myapp",
		},
		{
			name:  "uppercase to lowercase",
			input: "MyApp",
			want:  "myapp",
		},
		{
			name:  "spaces to underscores",
			input: "My App Name",
			want:  "my_app_name",
		},
		{
			name:  "dashes to underscores",
			input: "my-app-name",
			want:  "my_app_name",
		},
		{
			name:  "mixed special chars",
			input: "My App: Test/Demo",
			want:  "my_app__test_demo", // Double underscore from consecutive replacements
		},
		{
			name:  "starts with number",
			input: "123app",
			want:  "_123app",
		},
		{
			name:  "empty string",
			input: "",
			want:  "resource",
		},
		{
			name:  "only special chars",
			input: "!!!",
			want:  "resource",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SanitizeName(tt.input)
			if got != tt.want {
				t.Errorf("SanitizeName(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestGenerateImportBlocks_AIFileReaders(t *testing.T) {
	t.Parallel()

	app := &discovery.App{
		ID:   "app_123",
		Name: "Test App",
		AIFileReaders: []discovery.AIFileReader{
			{ID: "reader_1", Name: "Invoice Parser", Type: discovery.FileReaderTypeAI},
			{ID: "reader_2", Name: "Resume Analyzer", Type: discovery.FileReaderTypeAI},
		},
	}

	selectedTypes := map[string]bool{
		"ai_file_readers": true,
	}

	blocks := GenerateImportBlocks(app, selectedTypes)

	// Should have app + 2 file readers = 3 blocks
	if len(blocks) != 3 {
		t.Fatalf("Expected 3 blocks (1 app + 2 readers), got %d", len(blocks))
	}

	// Check app block
	if blocks[0].ResourceType != "elementum_app" {
		t.Errorf("Expected first block to be app, got %s", blocks[0].ResourceType)
	}

	// Check reader blocks
	readerCount := 0
	for _, block := range blocks {
		if block.ResourceType == "elementum_ai_file_reader" {
			readerCount++
			if !strings.Contains(block.ID, "app_123:") {
				t.Errorf("Expected reader ID to contain app_123:, got %s", block.ID)
			}
		}
	}

	if readerCount != 2 {
		t.Errorf("Expected 2 AI file reader blocks, got %d", readerCount)
	}
}

func TestGenerateImportBlocks_AllFileReaderTypes(t *testing.T) {
	t.Parallel()

	app := &discovery.App{
		ID:   "app_123",
		Name: "Test App",
		AIFileReaders: []discovery.FileReader{
			{ID: "reader_ai", Name: "AI Invoice Parser", Type: discovery.FileReaderTypeAI},
			{ID: "reader_ocr", Name: "OCR Scanner", Type: discovery.FileReaderTypeOCR},
			{ID: "reader_json", Name: "JSON Parser", Type: discovery.FileReaderTypeJSON},
			{ID: "reader_xml", Name: "XML Parser", Type: discovery.FileReaderTypeXML},
		},
	}

	selectedTypes := map[string]bool{
		"ai_file_readers": true,
	}

	blocks := GenerateImportBlocks(app, selectedTypes)

	// Should have app + 4 file readers = 5 blocks
	if len(blocks) != 5 {
		t.Fatalf("Expected 5 blocks (1 app + 4 readers), got %d", len(blocks))
	}

	// Check that each reader type is correctly identified
	expectedTypes := map[string]bool{
		"elementum_ai_file_reader":   false,
		"elementum_text_file_reader": false,
		"elementum_json_file_reader": false,
		"elementum_xml_file_reader":  false,
	}

	for _, block := range blocks {
		if _, ok := expectedTypes[block.ResourceType]; ok {
			expectedTypes[block.ResourceType] = true
		}
	}

	for resourceType, found := range expectedTypes {
		if !found {
			t.Errorf("Expected to find %s resource type, but it was not generated", resourceType)
		}
	}
}

func TestGenerateImportBlocks_ApprovalProcesses(t *testing.T) {
	t.Parallel()

	app := &discovery.App{
		ID:   "app_456",
		Name: "Test App",
		Approvals: []discovery.ApprovalProcess{
			{ID: "approval_1", Name: "Manager Approval"},
		},
	}

	selectedTypes := map[string]bool{
		"approval_processes": true,
	}

	blocks := GenerateImportBlocks(app, selectedTypes)

	// Should have app + 1 approval = 2 blocks
	if len(blocks) != 2 {
		t.Fatalf("Expected 2 blocks (1 app + 1 approval), got %d", len(blocks))
	}

	// Check approval block
	var approvalBlock *ImportBlock
	for i, block := range blocks {
		if block.ResourceType == "elementum_approval_process" {
			approvalBlock = &blocks[i]
			break
		}
	}

	if approvalBlock == nil {
		t.Fatal("No approval process block found")
	}

	if !strings.Contains(approvalBlock.ID, "app_456:") {
		t.Errorf("Expected approval ID to contain app_456:, got %s", approvalBlock.ID)
	}

	if approvalBlock.ResourceName != "manager_approval" {
		t.Errorf("Expected resource name 'manager_approval', got %s", approvalBlock.ResourceName)
	}
}

// TestGenerateImportBlocks_TableSearchTablesFromDiscoveredTables tests that
// table search tables from discovered tables are included in import blocks.
func TestGenerateImportBlocks_TableSearchTablesFromDiscoveredTables(t *testing.T) {
	t.Parallel()

	app := &discovery.App{
		ID:        "app-123",
		Name:      "Test App",
		Namespace: "test_app",
		DiscoveredTables: []*discovery.Table{
			{
				ID:   "table-1",
				Name: "Customer Data",
				SearchTables: []discovery.TableSearchTable{
					{
						ID:        "st-1",
						TableID:   "table-1",
						FieldID:   "field-desc",
						FieldName: "Description",
					},
					{
						ID:        "st-2",
						TableID:   "table-1",
						FieldID:   "field-notes",
						FieldName: "Notes",
					},
				},
			},
			{
				ID:   "table-2",
				Name: "Order History",
				SearchTables: []discovery.TableSearchTable{
					{
						ID:        "st-3",
						TableID:   "table-2",
						FieldID:   "field-details",
						FieldName: "Details",
					},
				},
			},
		},
	}

	selectedTypes := map[string]bool{
		"tables":              true,
		"table_search_tables": true,
	}

	blocks := GenerateImportBlocks(app, selectedTypes)

	// Count table search table blocks
	searchTableCount := 0
	for _, block := range blocks {
		if block.ResourceType == "elementum_table_search_table" {
			searchTableCount++
		}
	}

	// Should have 3 table search tables from discovered tables
	if searchTableCount != 3 {
		t.Errorf("Expected 3 table search table blocks, got %d", searchTableCount)
	}

	// Verify import IDs are correctly formatted (table_id/search_table_id)
	for _, block := range blocks {
		if block.ResourceType == "elementum_table_search_table" {
			if !strings.Contains(block.ID, "/") {
				t.Errorf("Expected table search table ID to contain '/', got %s", block.ID)
			}
		}
	}
}

// TestGenerateImportBlocks_TableSearchTablesFromReferencedTables tests that
// table search tables from referenced tables are included in import blocks.
func TestGenerateImportBlocks_TableSearchTablesFromReferencedTables(t *testing.T) {
	t.Parallel()

	app := &discovery.App{
		ID:        "app-123",
		Name:      "Test App",
		Namespace: "test_app",
		ReferencedTables: []*discovery.Table{
			{
				ID:   "table-ref-1",
				Name: "Referenced Table",
				SearchTables: []discovery.TableSearchTable{
					{
						ID:        "st-ref-1",
						TableID:   "table-ref-1",
						FieldID:   "field-content",
						FieldName: "Content",
					},
				},
			},
		},
	}

	selectedTypes := map[string]bool{
		"tables":              true,
		"table_search_tables": true,
	}

	blocks := GenerateImportBlocks(app, selectedTypes)

	// Count table search table blocks
	searchTableCount := 0
	for _, block := range blocks {
		if block.ResourceType == "elementum_table_search_table" {
			searchTableCount++
		}
	}

	// Should have 1 table search table from referenced tables
	if searchTableCount != 1 {
		t.Errorf("Expected 1 table search table block from referenced tables, got %d", searchTableCount)
	}
}

// TestGenerateImportBlocks_TableSearchTableNamingFromFieldName tests that
// table search table resource names are derived from the field name.
func TestGenerateImportBlocks_TableSearchTableNamingFromFieldName(t *testing.T) {
	t.Parallel()

	app := &discovery.App{
		ID:        "app-123",
		Name:      "Test App",
		Namespace: "test_app",
		DiscoveredTables: []*discovery.Table{
			{
				ID:   "table-1",
				Name: "Test Table",
				SearchTables: []discovery.TableSearchTable{
					{
						ID:        "st-1",
						TableID:   "table-1",
						FieldID:   "field-1",
						FieldName: "Full Description",
					},
				},
			},
		},
	}

	selectedTypes := map[string]bool{
		"tables":              true,
		"table_search_tables": true,
	}

	blocks := GenerateImportBlocks(app, selectedTypes)

	// Find the table search table block
	var searchTableBlock *ImportBlock
	for i, block := range blocks {
		if block.ResourceType == "elementum_table_search_table" {
			searchTableBlock = &blocks[i]
			break
		}
	}

	if searchTableBlock == nil {
		t.Fatal("No table search table block found")
	}

	// Resource name should be derived from field name (sanitized)
	if searchTableBlock.ResourceName != "full_description" {
		t.Errorf("Expected resource name 'full_description', got %s", searchTableBlock.ResourceName)
	}
}

// TestGenerateImportBlocks_TableSearchTablesImportIDFormat tests that
// the import ID format is correct (table_id/search_table_id).
func TestGenerateImportBlocks_TableSearchTablesImportIDFormat(t *testing.T) {
	t.Parallel()

	app := &discovery.App{
		ID:        "app-123",
		Name:      "Test App",
		Namespace: "test_app",
		DiscoveredTables: []*discovery.Table{
			{
				ID:   "table-abc-123",
				Name: "Test Table",
				SearchTables: []discovery.TableSearchTable{
					{
						ID:        "st-xyz-789",
						TableID:   "table-abc-123",
						FieldID:   "field-1",
						FieldName: "Description",
					},
				},
			},
		},
	}

	selectedTypes := map[string]bool{
		"tables":              true,
		"table_search_tables": true,
	}

	blocks := GenerateImportBlocks(app, selectedTypes)

	// Find the table search table block
	for _, block := range blocks {
		if block.ResourceType == "elementum_table_search_table" {
			// Import ID should be table_id/search_table_id
			if block.ID != "table-abc-123/st-xyz-789" {
				t.Errorf("Expected import ID 'table-abc-123/st-xyz-789', got %s", block.ID)
			}
			return
		}
	}

	t.Error("No table search table block found")
}

// TestGenerateImportBlocks_TableSearchTablesWithNoFieldName tests that
// resource names fall back when field name is empty.
func TestGenerateImportBlocks_TableSearchTablesWithNoFieldName(t *testing.T) {
	t.Parallel()

	app := &discovery.App{
		ID:        "app-123",
		Name:      "Test App",
		Namespace: "test_app",
		DiscoveredTables: []*discovery.Table{
			{
				ID:   "table-1",
				Name: "Test Table",
				SearchTables: []discovery.TableSearchTable{
					{
						ID:        "st-1",
						TableID:   "table-1",
						FieldID:   "field-1",
						FieldName: "", // Empty field name
					},
				},
			},
		},
	}

	selectedTypes := map[string]bool{
		"tables":              true,
		"table_search_tables": true,
	}

	blocks := GenerateImportBlocks(app, selectedTypes)

	// Find the table search table block
	var searchTableBlock *ImportBlock
	for i, block := range blocks {
		if block.ResourceType == "elementum_table_search_table" {
			searchTableBlock = &blocks[i]
			break
		}
	}

	if searchTableBlock == nil {
		t.Fatal("No table search table block found")
	}

	// Resource name should have a fallback
	if searchTableBlock.ResourceName == "" {
		t.Error("Expected non-empty resource name even with empty field name")
	}
}

func TestGenerateImportBlocks_AISearchTablesFromDiscoveredElements(t *testing.T) {
	t.Parallel()

	app := &discovery.App{
		ID:             "app-123",
		Name:           "Test App",
		Namespace:      "test_app",
		AISearchTables: []discovery.AISearchTable{}, // No app-level search tables
		DiscoveredElements: []*discovery.Element{
			{
				ID:        "elem-1",
				Name:      "Categories",
				Namespace: "categories",
				AISearchTables: []discovery.AISearchTable{
					{
						ID:        "st-1",
						ObjectID:  "elem-1",
						FieldID:   "field-desc",
						FieldName: "Description",
					},
					{
						ID:        "st-2",
						ObjectID:  "elem-1",
						FieldID:   "field-notes",
						FieldName: "Notes",
					},
				},
			},
			{
				ID:        "elem-2",
				Name:      "Locations",
				Namespace: "locations",
				AISearchTables: []discovery.AISearchTable{
					{
						ID:        "st-3",
						ObjectID:  "elem-2",
						FieldID:   "field-name",
						FieldName: "Full Name",
					},
				},
			},
		},
	}

	selectedTypes := map[string]bool{
		"ai_search_tables": true,
	}

	blocks := GenerateImportBlocks(app, selectedTypes)

	// Count AI search table blocks
	searchTableCount := 0
	for _, block := range blocks {
		if block.ResourceType == "elementum_ai_search_table" {
			searchTableCount++
		}
	}

	// Should have 3 AI search tables from discovered elements
	if searchTableCount != 3 {
		t.Errorf("Expected 3 AI search table blocks, got %d", searchTableCount)
	}

	// Verify import IDs are correctly formatted (object_id/search_table_id)
	for _, block := range blocks {
		if block.ResourceType == "elementum_ai_search_table" {
			if !strings.Contains(block.ID, "/") {
				t.Errorf("Expected AI search table ID to contain '/', got %s", block.ID)
			}
		}
	}
}

func TestGenerateImportBlocks_AISearchTablesFromAppAndElements(t *testing.T) {
	t.Parallel()

	app := &discovery.App{
		ID:        "app-123",
		Name:      "Test App",
		Namespace: "test_app",
		AISearchTables: []discovery.AISearchTable{
			{
				ID:        "st-app-1",
				ObjectID:  "app-123",
				FieldID:   "field-app-desc",
				FieldName: "App Description",
			},
		},
		DiscoveredElements: []*discovery.Element{
			{
				ID:        "elem-1",
				Name:      "Child Element",
				Namespace: "child_element",
				AISearchTables: []discovery.AISearchTable{
					{
						ID:        "st-elem-1",
						ObjectID:  "elem-1",
						FieldID:   "field-elem-notes",
						FieldName: "Element Notes",
					},
				},
			},
		},
	}

	selectedTypes := map[string]bool{
		"ai_search_tables": true,
	}

	blocks := GenerateImportBlocks(app, selectedTypes)

	// Count AI search table blocks
	searchTableCount := 0
	var appSearchTableFound, elemSearchTableFound bool
	for _, block := range blocks {
		if block.ResourceType == "elementum_ai_search_table" {
			searchTableCount++
			if strings.Contains(block.ID, "app-123") {
				appSearchTableFound = true
			}
			if strings.Contains(block.ID, "elem-1") {
				elemSearchTableFound = true
			}
		}
	}

	// Should have 2 AI search tables (1 from app, 1 from element)
	if searchTableCount != 2 {
		t.Errorf("Expected 2 AI search table blocks, got %d", searchTableCount)
	}

	if !appSearchTableFound {
		t.Error("Expected to find app-level AI search table")
	}
	if !elemSearchTableFound {
		t.Error("Expected to find element-level AI search table")
	}
}

func TestGenerateImportBlocks_AISearchTablesNamingFromFieldName(t *testing.T) {
	t.Parallel()

	app := &discovery.App{
		ID:        "app-123",
		Name:      "Test App",
		Namespace: "test_app",
		DiscoveredElements: []*discovery.Element{
			{
				ID:        "elem-1",
				Name:      "Test Element",
				Namespace: "test_element",
				AISearchTables: []discovery.AISearchTable{
					{
						ID:        "st-1",
						ObjectID:  "elem-1",
						FieldID:   "field-1",
						FieldName: "Full Description",
					},
				},
			},
		},
	}

	selectedTypes := map[string]bool{
		"ai_search_tables": true,
	}

	blocks := GenerateImportBlocks(app, selectedTypes)

	// Find the AI search table block
	var searchTableBlock *ImportBlock
	for i, block := range blocks {
		if block.ResourceType == "elementum_ai_search_table" {
			searchTableBlock = &blocks[i]
			break
		}
	}

	if searchTableBlock == nil {
		t.Fatal("No AI search table block found")
	}

	// Resource name should be derived from element prefix + field name
	if !strings.Contains(searchTableBlock.ResourceName, "full_description") {
		t.Errorf("Expected resource name to contain 'full_description', got %s", searchTableBlock.ResourceName)
	}
}

func TestGenerateImportBlocks_NoSelection(t *testing.T) {
	t.Parallel()

	app := &discovery.App{
		ID:   "app_789",
		Name: "Test App",
		Fields: []discovery.Field{
			{ID: "field_1", Name: "Title", Type: "text"},
		},
	}

	selectedTypes := map[string]bool{}

	blocks := GenerateImportBlocks(app, selectedTypes)

	// Should have no blocks when nothing is selected
	if len(blocks) != 0 {
		t.Errorf("Expected 0 blocks with no selection, got %d", len(blocks))
	}
}

func TestGenerateImportBlocks_DuplicateNames(t *testing.T) {
	t.Parallel()

	app := &discovery.App{
		ID:   "app_123",
		Name: "Test App",
		AIFileReaders: []discovery.AIFileReader{
			{ID: "reader_1", Name: "Parser"},
			{ID: "reader_2", Name: "Parser"},
			{ID: "reader_3", Name: "Parser"},
		},
	}

	selectedTypes := map[string]bool{
		"ai_file_readers": true,
	}

	blocks := GenerateImportBlocks(app, selectedTypes)

	// Check that duplicate names get suffixes
	names := make(map[string]int)
	for _, block := range blocks {
		if block.ResourceType == "elementum_ai_file_reader" {
			names[block.ResourceName]++
		}
	}

	// Should have parser, parser_1, parser_2
	if names["parser"] != 1 {
		t.Errorf("Expected 1 'parser', got %d", names["parser"])
	}
	if names["parser_1"] != 1 {
		t.Errorf("Expected 1 'parser_1', got %d", names["parser_1"])
	}
	if names["parser_2"] != 1 {
		t.Errorf("Expected 1 'parser_2', got %d", names["parser_2"])
	}
}

func TestRenderImportBlocks(t *testing.T) {
	t.Parallel()

	blocks := []ImportBlock{
		{
			ID:           "app_123",
			ResourceType: "elementum_app",
			ResourceName: "my_app",
		},
		{
			ID:           "app_123:reader_456",
			ResourceType: "elementum_ai_file_reader",
			ResourceName: "invoice_parser",
		},
	}

	output := RenderImportBlocks(blocks)

	// Check that output contains import blocks
	if !strings.Contains(output, "import {") {
		t.Error("Expected output to contain 'import {'")
	}
	if !strings.Contains(output, `id = "app_123"`) {
		t.Error("Expected output to contain app ID")
	}
	if !strings.Contains(output, `to = elementum_app.my_app`) {
		t.Error("Expected output to contain app resource reference")
	}
	if !strings.Contains(output, `to = elementum_ai_file_reader.invoice_parser`) {
		t.Error("Expected output to contain file reader resource reference")
	}
}

func TestRenderProviderConfig(t *testing.T) {
	t.Parallel()

	output := RenderProviderConfig("myorg", "eu", "production", "client_id_123", "secret_xyz")

	// Check provider config elements
	if !strings.Contains(output, `organization  = "myorg"`) {
		t.Error("Expected output to contain organization")
	}
	if !strings.Contains(output, `environment   = "production"`) {
		t.Error("Expected output to contain environment")
	}
	if !strings.Contains(output, `client_id     = "client_id_123"`) {
		t.Error("Expected output to contain client_id")
	}
	if !strings.Contains(output, "terraform {") {
		t.Error("Expected output to contain terraform block")
	}
	if !strings.Contains(output, "required_providers {") {
		t.Error("Expected output to contain required_providers")
	}
}

func TestGenerateImportBlocks_AllResourceTypes(t *testing.T) {
	t.Parallel()

	app := &discovery.App{
		ID:        "app_test",
		Name:      "Complete App",
		Namespace: "completeapp",
		Fields: []discovery.Field{
			{ID: "field_1", Name: "Title", Type: "text"},
		},
		Layouts: []discovery.Layout{
			{ID: "layout_1", Name: "Open"},
		},
		Flows: []discovery.Flow{
			{ID: "flow_1", Name: "Main"},
		},
		Agents: []discovery.Agent{
			{ID: "agent_1", Name: "Bot", Tools: []discovery.AgentTool{
				{ID: "tool_1", Name: "Search", Type: "AgentSearchAspectTool"},
			}},
		},
		Widgets: []discovery.Widget{
			{ID: "widget_1", Name: "Dashboard"},
		},
		AIFileReaders: []discovery.AIFileReader{
			{ID: "reader_1", Name: "Parser"},
		},
		Approvals: []discovery.ApprovalProcess{
			{ID: "approval_1", Name: "Approval"},
		},
	}

	selectedTypes := map[string]bool{
		"fields":             true,
		"layouts":            true,
		"flows":              true,
		"agents":             true,
		"widgets":            true,
		"ai_file_readers":    true,
		"approval_processes": true,
	}

	blocks := GenerateImportBlocks(app, selectedTypes)

	// Count resource types
	types := make(map[string]int)
	for _, block := range blocks {
		types[block.ResourceType]++
	}

	// Verify all types are present
	expectedTypes := []string{
		"elementum_app",
		"elementum_text_field",
		"elementum_layout",
		"elementum_flow",
		"elementum_agent",
		"elementum_agent_search_records_tool",
		"elementum_widget",
		"elementum_ai_file_reader",
		"elementum_approval_process",
	}

	for _, expectedType := range expectedTypes {
		if types[expectedType] == 0 {
			t.Errorf("Expected at least 1 %s block, got 0", expectedType)
		}
	}
}

func TestGenerateImportBlocks_Relationships(t *testing.T) {
	t.Parallel()

	app := &discovery.App{
		ID:        "app_orders",
		Name:      "Orders",
		Namespace: "orders",
		Relationships: []discovery.Relationship{
			{
				ID:                "rel_1",
				RelatedObjectID:   "app_customers",
				RelatedObjectType: "App",
				RelatedObjectName: "Customers",
				Columns: []discovery.RelationshipColumn{
					{
						FieldID:          "field_cust_ref",
						FieldName:        "Customer",
						RelatedFieldID:   "field_cust_title",
						RelatedFieldName: "Title",
					},
				},
			},
			{
				ID:                "rel_2",
				RelatedObjectID:   "elem_products",
				RelatedObjectType: "Element",
				RelatedObjectName: "Products",
				Columns: []discovery.RelationshipColumn{
					{
						FieldID:          "field_prod_ref",
						FieldName:        "Product",
						RelatedFieldID:   "field_prod_title",
						RelatedFieldName: "Title",
					},
				},
			},
		},
	}

	selectedTypes := map[string]bool{
		"relationships": true,
	}

	blocks := GenerateImportBlocks(app, selectedTypes)

	// Should have app + 2 relationships = 3 blocks
	if len(blocks) != 3 {
		t.Fatalf("Expected 3 blocks (1 app + 2 relationships), got %d", len(blocks))
	}

	// Check relationship blocks
	relBlocks := []ImportBlock{}
	for _, block := range blocks {
		if block.ResourceType == "elementum_relationship" {
			relBlocks = append(relBlocks, block)
		}
	}

	if len(relBlocks) != 2 {
		t.Errorf("Expected 2 relationship blocks, got %d", len(relBlocks))
	}

	// Check that names include app namespace prefix for uniqueness
	names := make(map[string]bool)
	for _, block := range relBlocks {
		names[block.ResourceName] = true
	}

	if !names["orders_to_customers"] {
		t.Error("Expected relationship named 'orders_to_customers'")
	}
	if !names["orders_to_products"] {
		t.Error("Expected relationship named 'orders_to_products'")
	}
}

func TestGenerateImportBlocks_RelationshipsDuplicateNames(t *testing.T) {
	t.Parallel()

	app := &discovery.App{
		ID:        "app_orders",
		Name:      "Orders",
		Namespace: "orders",
		Relationships: []discovery.Relationship{
			{
				ID:                "rel_1",
				RelatedObjectID:   "app_cust_1",
				RelatedObjectType: "App",
				RelatedObjectName: "Customers",
			},
			{
				ID:                "rel_2",
				RelatedObjectID:   "app_cust_2",
				RelatedObjectType: "App",
				RelatedObjectName: "Customers", // Same name, different ID
			},
		},
	}

	selectedTypes := map[string]bool{
		"relationships": true,
	}

	blocks := GenerateImportBlocks(app, selectedTypes)

	// Check that duplicate names get suffixes
	relBlocks := []ImportBlock{}
	for _, block := range blocks {
		if block.ResourceType == "elementum_relationship" {
			relBlocks = append(relBlocks, block)
		}
	}

	names := make(map[string]int)
	for _, block := range relBlocks {
		names[block.ResourceName]++
	}

	// Should have orders_to_customers and orders_to_customers_1 (with namespace prefix)
	if names["orders_to_customers"] != 1 {
		t.Errorf("Expected 1 'orders_to_customers', got %d", names["orders_to_customers"])
	}
	if names["orders_to_customers_1"] != 1 {
		t.Errorf("Expected 1 'orders_to_customers_1', got %d", names["orders_to_customers_1"])
	}
}

// TestGenerateImportBlocks_RelationshipsMultipleApps verifies that relationships
// from different apps to the same target get unique names (regression test for
// the namespace prefix fix).
func TestGenerateImportBlocks_RelationshipsMultipleApps(t *testing.T) {
	t.Parallel()

	// Two apps that both have relationships to "Customers"
	ordersApp := &discovery.App{
		ID:        "app_orders",
		Name:      "Orders",
		Namespace: "orders",
		Relationships: []discovery.Relationship{
			{
				ID:                "rel_orders_cust",
				RelatedObjectID:   "app_customers",
				RelatedObjectType: "App",
				RelatedObjectName: "Customers",
			},
		},
	}

	invoicesApp := &discovery.App{
		ID:        "app_invoices",
		Name:      "Invoices",
		Namespace: "invoices",
		Relationships: []discovery.Relationship{
			{
				ID:                "rel_invoices_cust",
				RelatedObjectID:   "app_customers",
				RelatedObjectType: "App",
				RelatedObjectName: "Customers",
			},
		},
	}

	selectedTypes := map[string]bool{
		"relationships": true,
	}

	// Generate blocks for both apps
	ordersBlocks := GenerateImportBlocks(ordersApp, selectedTypes)
	invoicesBlocks := GenerateImportBlocks(invoicesApp, selectedTypes)

	// Find relationship blocks
	var ordersRelName, invoicesRelName string
	for _, block := range ordersBlocks {
		if block.ResourceType == "elementum_relationship" {
			ordersRelName = block.ResourceName
		}
	}
	for _, block := range invoicesBlocks {
		if block.ResourceType == "elementum_relationship" {
			invoicesRelName = block.ResourceName
		}
	}

	// Names should be different due to namespace prefix
	if ordersRelName == invoicesRelName {
		t.Errorf("Relationship names should be different for different apps, both got: %s", ordersRelName)
	}

	// Verify expected names
	if ordersRelName != "orders_to_customers" {
		t.Errorf("Expected orders relationship named 'orders_to_customers', got %s", ordersRelName)
	}
	if invoicesRelName != "invoices_to_customers" {
		t.Errorf("Expected invoices relationship named 'invoices_to_customers', got %s", invoicesRelName)
	}
}

func TestGenerateImportBlocks_DeduplicatesBidirectionalRelationships(t *testing.T) {
	t.Parallel()

	// Create app with relationship to element, where the element also has the same
	// relationship back to the app (bidirectional relationship, same ID)
	app := &discovery.App{
		ID:        "app_docdigi",
		Name:      "DocDigi",
		Namespace: "docdigi",
		Relationships: []discovery.Relationship{
			{
				ID:                "rel_abc", // Same relationship ID
				RelatedObjectID:   "elem_digitized_doc",
				RelatedObjectType: "Element",
				RelatedObjectName: "DigitizedDocument",
				Columns: []discovery.RelationshipColumn{
					{
						FieldID:          "field_doc_ref",
						FieldName:        "Document",
						RelatedFieldID:   "field_title",
						RelatedFieldName: "Title",
					},
				},
			},
		},
		DiscoveredElements: []*discovery.Element{
			{
				ID:        "elem_digitized_doc",
				Name:      "DigitizedDocument",
				Namespace: "digitizeddocument",
				Relationships: []discovery.Relationship{
					{
						ID:                "rel_abc", // Same relationship ID!
						RelatedObjectID:   "app_docdigi",
						RelatedObjectType: "App",
						RelatedObjectName: "DocDigi",
						Columns: []discovery.RelationshipColumn{
							{
								FieldID:          "field_title",
								FieldName:        "Title",
								RelatedFieldID:   "field_doc_ref",
								RelatedFieldName: "Document",
							},
						},
					},
				},
			},
		},
	}

	selectedTypes := map[string]bool{
		"relationships": true,
	}

	blocks := GenerateImportBlocks(app, selectedTypes)

	// Count relationship blocks - should be 1 (not 2)
	relCount := 0
	for _, b := range blocks {
		if b.ResourceType == "elementum_relationship" {
			relCount++
		}
	}

	// Should only have 1 relationship, not 2 (deduplication)
	if relCount != 1 {
		t.Errorf("Expected 1 relationship block (deduplicated), got %d", relCount)
	}

	// Verify the relationship that was kept is from the app (first one processed)
	for _, b := range blocks {
		if b.ResourceType == "elementum_relationship" {
			if b.ResourceName != "docdigi_to_digitizeddocument" {
				t.Errorf("Expected relationship named 'docdigi_to_digitizeddocument', got %s", b.ResourceName)
			}
			// Verify it's the app's relationship (not element's) by checking the import ID
			if !strings.Contains(b.ID, "app_docdigi") {
				t.Errorf("Expected relationship to be from app, got ID: %s", b.ID)
			}
		}
	}
}

func TestGenerateDataSourceBlocks(t *testing.T) {
	t.Parallel()

	app := &discovery.App{
		ID:   "app_orders",
		Name: "Orders",
		RelatedObjects: []discovery.RelatedObject{
			{
				ID:        "app_customers",
				Name:      "Customers",
				Type:      "App",
				Namespace: "customers",
			},
			{
				ID:        "elem_products",
				Name:      "Products",
				Type:      "Element",
				Namespace: "products",
			},
		},
	}

	output := GenerateDataSourceBlocks(app)

	// Check that output contains data source blocks
	if !strings.Contains(output, `data "elementum_app" "customers"`) {
		t.Error("Expected output to contain app data source for customers")
	}
	if !strings.Contains(output, `data "elementum_element" "products"`) {
		t.Error("Expected output to contain element data source for products")
	}
	if !strings.Contains(output, `name = "Customers"`) {
		t.Error("Expected output to contain Customers name attribute")
	}
	if !strings.Contains(output, `name = "Products"`) {
		t.Error("Expected output to contain Products name attribute")
	}
}

func TestGenerateDataSourceBlocks_Empty(t *testing.T) {
	t.Parallel()

	app := &discovery.App{
		ID:             "app_orders",
		Name:           "Orders",
		RelatedObjects: []discovery.RelatedObject{},
	}

	output := GenerateDataSourceBlocks(app)

	if output != "" {
		t.Errorf("Expected empty output for app with no related objects, got: %s", output)
	}
}

func TestGenerateDataSourceBlocks_DuplicateNames(t *testing.T) {
	t.Parallel()

	app := &discovery.App{
		ID:   "app_orders",
		Name: "Orders",
		RelatedObjects: []discovery.RelatedObject{
			{
				ID:        "app_cust_1",
				Name:      "Customers",
				Type:      "App",
				Namespace: "customers_us",
			},
			{
				ID:        "app_cust_2",
				Name:      "Customers",
				Type:      "App",
				Namespace: "customers_eu",
			},
		},
	}

	output := GenerateDataSourceBlocks(app)

	// Check that duplicate names get suffixes
	if !strings.Contains(output, `data "elementum_app" "customers"`) {
		t.Error("Expected output to contain data source 'customers'")
	}
	if !strings.Contains(output, `data "elementum_app" "customers_1"`) {
		t.Error("Expected output to contain data source 'customers_1' for duplicate")
	}
}

func TestGenerateDataSourceBlocks_SkipsDiscoveredElements(t *testing.T) {
	t.Parallel()

	// Test that elements/apps in DiscoveredElements/DiscoveredApps do NOT get data sources
	// even if they also appear in RelatedObjects
	app := &discovery.App{
		ID:   "app_orders",
		Name: "Orders",
		RelatedObjects: []discovery.RelatedObject{
			{
				ID:        "elem_products",
				Name:      "Products",
				Type:      "Element",
				Namespace: "products",
			},
			{
				ID:        "elem_inventory",
				Name:      "Inventory",
				Type:      "Element",
				Namespace: "inventory",
			},
			{
				ID:        "app_customers",
				Name:      "Customers",
				Type:      "App",
				Namespace: "customers",
			},
			{
				ID:        "app_suppliers",
				Name:      "Suppliers",
				Type:      "App",
				Namespace: "suppliers",
			},
		},
		// Products element and Customers app are being exported as resources
		DiscoveredElements: []*discovery.Element{
			{
				ID:        "elem_products",
				Name:      "Products",
				Namespace: "products",
			},
		},
		DiscoveredApps: []*discovery.App{
			{
				ID:        "app_customers",
				Name:      "Customers",
				Namespace: "customers",
			},
		},
	}

	output := GenerateDataSourceBlocks(app)

	// Products and Customers should NOT have data sources (they're being exported as resources)
	if strings.Contains(output, `data "elementum_element" "products"`) {
		t.Error("Should NOT contain data source for 'products' element - it's being exported as a resource")
	}
	if strings.Contains(output, `data "elementum_app" "customers"`) {
		t.Error("Should NOT contain data source for 'customers' app - it's being exported as a resource")
	}

	// Inventory and Suppliers SHOULD have data sources (they're only in RelatedObjects)
	if !strings.Contains(output, `data "elementum_element" "inventory"`) {
		t.Error("Expected output to contain data source for 'inventory' element")
	}
	if !strings.Contains(output, `data "elementum_app" "suppliers"`) {
		t.Error("Expected output to contain data source for 'suppliers' app")
	}
}

// Test GenerateCloudLinkDataSourcesFromMap generates correct HCL
func TestGenerateCloudLinkDataSourcesFromMap(t *testing.T) {
	tests := []struct {
		name         string
		cloudlinks   map[string]*discovery.CloudLink
		wantContains []string
		wantMissing  []string
	}{
		{
			name: "single CloudLink",
			cloudlinks: map[string]*discovery.CloudLink{
				"cl-123": {
					ID:   "cl-123",
					Name: "Production Snowflake",
				},
			},
			wantContains: []string{
				`# CloudLink data sources (discovered from tables)`,
				`data "elementum_cloudlink" "production_snowflake"`,
				`name = "Production Snowflake"`,
			},
		},
		{
			name: "multiple CloudLinks",
			cloudlinks: map[string]*discovery.CloudLink{
				"cl-1": {
					ID:   "cl-1",
					Name: "Snowflake Prod",
				},
				"cl-2": {
					ID:   "cl-2",
					Name: "BigQuery Analytics",
				},
			},
			wantContains: []string{
				`data "elementum_cloudlink" "snowflake_prod"`,
				`name = "Snowflake Prod"`,
				`data "elementum_cloudlink" "bigquery_analytics"`,
				`name = "BigQuery Analytics"`,
			},
		},
		{
			name:         "empty map returns empty string",
			cloudlinks:   map[string]*discovery.CloudLink{},
			wantContains: []string{},
			wantMissing: []string{
				"data",
				"elementum_cloudlink",
			},
		},
		{
			name:       "nil map returns empty string",
			cloudlinks: nil,
			wantMissing: []string{
				"data",
			},
		},
		{
			name: "handles special characters in name",
			cloudlinks: map[string]*discovery.CloudLink{
				"cl-special": {
					ID:   "cl-special",
					Name: "Cloud Link - Dev (Test)",
				},
			},
			wantContains: []string{
				// Parentheses are removed (not converted to underscore)
				`data "elementum_cloudlink" "cloud_link___dev_test"`,
				`name = "Cloud Link - Dev (Test)"`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GenerateCloudLinkDataSourcesFromMap(tt.cloudlinks)

			for _, want := range tt.wantContains {
				if !strings.Contains(result, want) {
					t.Errorf("expected result to contain %q, got:\n%s", want, result)
				}
			}

			for _, notWant := range tt.wantMissing {
				if strings.Contains(result, notWant) {
					t.Errorf("expected result NOT to contain %q, got:\n%s", notWant, result)
				}
			}
		})
	}
}

// Test GenerateImportBlocks_Roles generates role import blocks for custom roles only
func TestGenerateImportBlocks_Roles(t *testing.T) {
	t.Parallel()

	app := &discovery.App{
		ID:        "app_123",
		Name:      "Test App",
		Namespace: "testapp",
		Roles: []discovery.Role{
			{ID: "role-1", Name: "Admin", Managed: true},            // Managed - should be skipped
			{ID: "role-2", Name: "Editor", Managed: true},           // Managed - should be skipped
			{ID: "role-3", Name: "Custom Role", Managed: false},     // Custom - should be included
			{ID: "role-4", Name: "Project Manager", Managed: false}, // Custom - should be included
		},
	}

	selectedTypes := map[string]bool{
		"roles": true,
	}

	blocks := GenerateImportBlocks(app, selectedTypes)

	// Should have app + 2 custom roles = 3 blocks
	if len(blocks) != 3 {
		t.Fatalf("Expected 3 blocks (1 app + 2 custom roles), got %d", len(blocks))
	}

	// Check role blocks - only custom roles should be present
	roleCount := 0
	for _, block := range blocks {
		if block.ResourceType == "elementum_role" {
			roleCount++
			// Verify ID format
			if !strings.Contains(block.ID, "app_123:") {
				t.Errorf("Expected role ID to contain app_123:, got %s", block.ID)
			}
		}
	}

	if roleCount != 2 {
		t.Errorf("Expected 2 custom role blocks, got %d", roleCount)
	}
}

// Test GenerateImportBlocks_Roles_OnlyManaged verifies no role blocks when all roles are managed
func TestGenerateImportBlocks_Roles_OnlyManaged(t *testing.T) {
	t.Parallel()

	app := &discovery.App{
		ID:   "app_123",
		Name: "Test App",
		Roles: []discovery.Role{
			{ID: "role-1", Name: "Admin", Managed: true},
			{ID: "role-2", Name: "Editor", Managed: true},
			{ID: "role-3", Name: "Viewer", Managed: true},
		},
	}

	selectedTypes := map[string]bool{
		"roles": true,
	}

	blocks := GenerateImportBlocks(app, selectedTypes)

	// Should have app + 0 custom roles = 1 block
	if len(blocks) != 1 {
		t.Fatalf("Expected 1 block (just app), got %d", len(blocks))
	}

	// Verify no role blocks
	for _, block := range blocks {
		if block.ResourceType == "elementum_role" {
			t.Error("Should not generate import blocks for managed roles")
		}
	}
}

// Test GenerateImportBlocks_Roles_DuplicateNames handles duplicate role names
func TestGenerateImportBlocks_Roles_DuplicateNames(t *testing.T) {
	t.Parallel()

	app := &discovery.App{
		ID:   "app_123",
		Name: "Test App",
		Roles: []discovery.Role{
			{ID: "role-1", Name: "Custom Role", Managed: false},
			{ID: "role-2", Name: "Custom Role", Managed: false}, // Same name
			{ID: "role-3", Name: "Custom Role", Managed: false}, // Same name again
		},
	}

	selectedTypes := map[string]bool{
		"roles": true,
	}

	blocks := GenerateImportBlocks(app, selectedTypes)

	// Check that duplicate names get suffixes
	roleNames := make(map[string]int)
	for _, block := range blocks {
		if block.ResourceType == "elementum_role" {
			roleNames[block.ResourceName]++
		}
	}

	// Should have custom_role, custom_role_1, custom_role_2
	if roleNames["custom_role"] != 1 {
		t.Errorf("Expected 1 'custom_role', got %d", roleNames["custom_role"])
	}
	if roleNames["custom_role_1"] != 1 {
		t.Errorf("Expected 1 'custom_role_1', got %d", roleNames["custom_role_1"])
	}
	if roleNames["custom_role_2"] != 1 {
		t.Errorf("Expected 1 'custom_role_2', got %d", roleNames["custom_role_2"])
	}
}

// Test GenerateRoleDataSourceBlocks generates correct data sources (integration test)
func TestImports_GenerateRoleDataSourceBlocks(t *testing.T) {
	t.Parallel()

	app := &discovery.App{
		ID:   "app_123",
		Name: "Test App",
		Roles: []discovery.Role{
			{
				ID:      "role-1",
				Name:    "Admin",
				Managed: true,
			},
			{
				ID:      "role-2",
				Name:    "Project Manager",
				Managed: false,
				Users: []discovery.RoleMember{
					{ID: "user-1", Name: "john@example.com"},
					{ID: "user-2", Name: "jane@example.com"},
				},
				Groups: []discovery.RoleMember{
					{ID: "group-1", Name: "Engineering"},
				},
			},
		},
	}

	output := GenerateRoleDataSourceBlocks(app, "elementum_app.test_app.id")

	// Check user data sources
	if !strings.Contains(output, `data "elementum_user" "john"`) {
		t.Error("Expected user data source for john")
	}
	if !strings.Contains(output, `email = "john@example.com"`) {
		t.Error("Expected john's email in user data source")
	}
	if !strings.Contains(output, `data "elementum_user" "jane"`) {
		t.Error("Expected user data source for jane")
	}

	// Check group data sources
	if !strings.Contains(output, `data "elementum_group" "engineering"`) {
		t.Error("Expected group data source for Engineering")
	}
	if !strings.Contains(output, `name = "Engineering"`) {
		t.Error("Expected Engineering name in group data source")
	}

	// Check managed role data sources
	if !strings.Contains(output, `data "elementum_role" "admin"`) {
		t.Error("Expected managed role data source for Admin")
	}
	if !strings.Contains(output, `object_id = elementum_app.test_app.id`) {
		t.Error("Expected object_id reference in managed role data source")
	}
}

// Test GenerateRoleDataSourceBlocks returns empty for no roles (integration test)
func TestImports_GenerateRoleDataSourceBlocks_NoRoles(t *testing.T) {
	t.Parallel()

	app := &discovery.App{
		ID:    "app_123",
		Name:  "Test App",
		Roles: []discovery.Role{},
	}

	output := GenerateRoleDataSourceBlocks(app, "elementum_app.test_app.id")

	if output != "" {
		t.Errorf("Expected empty output for app with no roles, got: %s", output)
	}
}

// Test GenerateRoleDataSourceBlocks_DuplicateUserEmails handles duplicate emails
func TestGenerateRoleDataSourceBlocks_DuplicateUserEmails(t *testing.T) {
	t.Parallel()

	app := &discovery.App{
		ID:   "app_123",
		Name: "Test App",
		Roles: []discovery.Role{
			{
				ID:      "role-1",
				Name:    "Role 1",
				Managed: false,
				Users: []discovery.RoleMember{
					{ID: "user-1", Name: "john@example.com"},
				},
			},
			{
				ID:      "role-2",
				Name:    "Role 2",
				Managed: false,
				Users: []discovery.RoleMember{
					{ID: "user-1", Name: "john@example.com"}, // Same user in different role
				},
			},
		},
	}

	output := GenerateRoleDataSourceBlocks(app, "elementum_app.test_app.id")

	// Should only have one data source for john (deduplication happens via map)
	count := strings.Count(output, `data "elementum_user"`)
	if count != 1 {
		t.Errorf("Expected 1 user data source (deduplicated), got %d", count)
	}
}

// Test sanitizeEmailToName converts emails correctly (integration test)
func TestImports_SanitizeEmailToName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		email string
		want  string
	}{
		{
			name:  "simple email",
			email: "john@example.com",
			want:  "john",
		},
		{
			name:  "email with dots",
			email: "john.doe@example.com",
			want:  "john_doe",
		},
		{
			name:  "email with plus",
			email: "john+test@example.com",
			want:  "johntest",
		},
		{
			name:  "uppercase email",
			email: "John.Doe@Example.com",
			want:  "john_doe",
		},
		{
			name:  "email with numbers",
			email: "john123@example.com",
			want:  "john123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sanitizeEmailToName(tt.email)
			if got != tt.want {
				t.Errorf("sanitizeEmailToName(%q) = %q, want %q", tt.email, got, tt.want)
			}
		})
	}
}

// =============================================================================
// Element System Field Data Source Tests
// =============================================================================

// TestGenerateElementSystemFieldDataSources generates data sources for element audit fields
func TestGenerateElementSystemFieldDataSources(t *testing.T) {
	t.Parallel()

	app := &discovery.App{
		ID:        "app-123",
		Name:      "Test App",
		Namespace: "testapp",
		DiscoveredElements: []*discovery.Element{
			{
				ID:        "elem-456",
				Name:      "Products",
				Namespace: "products",
				Fields: []discovery.Field{
					{ID: "field-1", Name: "Title", Type: "text", SemanticTags: []string{"TITLE"}},
					{ID: "field-2", Name: "Created By", Type: "user", SemanticTags: []string{"CREATED_BY"}},
					{ID: "field-3", Name: "Created At", Type: "datetime", SemanticTags: []string{"CREATED_AT"}},
					{ID: "field-4", Name: "Updated By", Type: "user", SemanticTags: []string{"UPDATED_BY"}},
					{ID: "field-5", Name: "Updated At", Type: "datetime", SemanticTags: []string{"UPDATED_AT"}},
				},
			},
		},
	}

	output := GenerateElementSystemFieldDataSources(app)

	// Should contain the header comment
	if !strings.Contains(output, "# Element system field data sources") {
		t.Error("Expected header comment in output")
	}

	// Should generate data sources for audit fields (not TITLE - that's exposed by element resource)
	if !strings.Contains(output, `data "elementum_field" "products_created_by"`) {
		t.Error("Expected CREATED_BY data source for products")
	}
	if !strings.Contains(output, `name      = "Created By"`) {
		t.Error("Expected 'Created By' name attribute")
	}
	if !strings.Contains(output, `data "elementum_field" "products_created_at"`) {
		t.Error("Expected CREATED_AT data source for products")
	}
	if !strings.Contains(output, `data "elementum_field" "products_updated_by"`) {
		t.Error("Expected UPDATED_BY data source for products")
	}
	if !strings.Contains(output, `data "elementum_field" "products_updated_at"`) {
		t.Error("Expected UPDATED_AT data source for products")
	}

	// Should reference the element resource
	if !strings.Contains(output, `object_id = elementum_element.products.id`) {
		t.Error("Expected object_id to reference element resource")
	}

	// TITLE should NOT have a data source (element resource exposes title_field_id)
	if strings.Contains(output, `"products_title"`) {
		t.Error("TITLE field should NOT have a data source - element resource exposes it")
	}
}

// TestGenerateElementSystemFieldDataSources_MultipleElements tests multiple elements
func TestGenerateElementSystemFieldDataSources_MultipleElements(t *testing.T) {
	t.Parallel()

	app := &discovery.App{
		ID:        "app-123",
		Name:      "Test App",
		Namespace: "testapp",
		DiscoveredElements: []*discovery.Element{
			{
				ID:        "elem-1",
				Name:      "Products",
				Namespace: "products",
				Fields: []discovery.Field{
					{ID: "field-1", Name: "Created By", Type: "user", SemanticTags: []string{"CREATED_BY"}},
				},
			},
			{
				ID:        "elem-2",
				Name:      "Orders",
				Namespace: "orders",
				Fields: []discovery.Field{
					{ID: "field-2", Name: "Created By", Type: "user", SemanticTags: []string{"CREATED_BY"}},
					{ID: "field-3", Name: "Updated At", Type: "datetime", SemanticTags: []string{"UPDATED_AT"}},
				},
			},
		},
	}

	output := GenerateElementSystemFieldDataSources(app)

	// Each element should have its own data sources
	if !strings.Contains(output, `data "elementum_field" "products_created_by"`) {
		t.Error("Expected products CREATED_BY data source")
	}
	if !strings.Contains(output, `object_id = elementum_element.products.id`) {
		t.Error("Expected products object_id reference")
	}

	if !strings.Contains(output, `data "elementum_field" "orders_created_by"`) {
		t.Error("Expected orders CREATED_BY data source")
	}
	if !strings.Contains(output, `data "elementum_field" "orders_updated_at"`) {
		t.Error("Expected orders UPDATED_AT data source")
	}
	if !strings.Contains(output, `object_id = elementum_element.orders.id`) {
		t.Error("Expected orders object_id reference")
	}
}

// TestGenerateElementSystemFieldDataSources_NoElements returns empty for no elements
func TestGenerateElementSystemFieldDataSources_NoElements(t *testing.T) {
	t.Parallel()

	app := &discovery.App{
		ID:                 "app-123",
		Name:               "Test App",
		Namespace:          "testapp",
		DiscoveredElements: []*discovery.Element{},
	}

	output := GenerateElementSystemFieldDataSources(app)

	if output != "" {
		t.Errorf("Expected empty output for app with no discovered elements, got: %s", output)
	}
}

// TestGenerateElementSystemFieldDataSources_NilElements returns empty for nil
func TestGenerateElementSystemFieldDataSources_NilElements(t *testing.T) {
	t.Parallel()

	app := &discovery.App{
		ID:                 "app-123",
		Name:               "Test App",
		Namespace:          "testapp",
		DiscoveredElements: nil,
	}

	output := GenerateElementSystemFieldDataSources(app)

	if output != "" {
		t.Errorf("Expected empty output for app with nil discovered elements, got: %s", output)
	}
}

// TestGenerateElementSystemFieldDataSources_NoSystemFields returns empty when no audit fields
func TestGenerateElementSystemFieldDataSources_NoSystemFields(t *testing.T) {
	t.Parallel()

	app := &discovery.App{
		ID:        "app-123",
		Name:      "Test App",
		Namespace: "testapp",
		DiscoveredElements: []*discovery.Element{
			{
				ID:        "elem-1",
				Name:      "Products",
				Namespace: "products",
				Fields: []discovery.Field{
					// Only TITLE and ID tags - these are exposed by element resource
					{ID: "field-1", Name: "Title", Type: "text", SemanticTags: []string{"TITLE"}},
					{ID: "field-2", Name: "ID", Type: "text", SemanticTags: []string{"HANDLE"}},
					// Regular field - no system tag
					{ID: "field-3", Name: "Description", Type: "text"},
				},
			},
		},
	}

	output := GenerateElementSystemFieldDataSources(app)

	// Should return empty since no audit fields need data sources
	if output != "" {
		t.Errorf("Expected empty output when no audit system fields, got: %s", output)
	}
}

// TestGenerateElementSystemFieldDataSources_AllAuditFields tests all 6 audit field types
func TestGenerateElementSystemFieldDataSources_AllAuditFields(t *testing.T) {
	t.Parallel()

	app := &discovery.App{
		ID:        "app-123",
		Name:      "Test App",
		Namespace: "testapp",
		DiscoveredElements: []*discovery.Element{
			{
				ID:        "elem-1",
				Name:      "Records",
				Namespace: "records",
				Fields: []discovery.Field{
					{ID: "f1", Name: "Created By", Type: "user", SemanticTags: []string{"CREATED_BY"}},
					{ID: "f2", Name: "Created At", Type: "datetime", SemanticTags: []string{"CREATED_AT"}},
					{ID: "f3", Name: "Updated By", Type: "user", SemanticTags: []string{"UPDATED_BY"}},
					{ID: "f4", Name: "Updated At", Type: "datetime", SemanticTags: []string{"UPDATED_AT"}},
					{ID: "f5", Name: "Closed By", Type: "user", SemanticTags: []string{"CLOSED_BY"}},
					{ID: "f6", Name: "Closed At", Type: "datetime", SemanticTags: []string{"CLOSED_AT"}},
				},
			},
		},
	}

	output := GenerateElementSystemFieldDataSources(app)

	// All 6 audit fields should have data sources
	expectedDataSources := []string{
		`data "elementum_field" "records_created_by"`,
		`data "elementum_field" "records_created_at"`,
		`data "elementum_field" "records_updated_by"`,
		`data "elementum_field" "records_updated_at"`,
		`data "elementum_field" "records_closed_by"`,
		`data "elementum_field" "records_closed_at"`,
	}

	for _, expected := range expectedDataSources {
		if !strings.Contains(output, expected) {
			t.Errorf("Expected %s in output", expected)
		}
	}

	// Check field names are correct
	expectedNames := []string{
		`name      = "Created By"`,
		`name      = "Created At"`,
		`name      = "Updated By"`,
		`name      = "Updated At"`,
		`name      = "Closed By"`,
		`name      = "Closed At"`,
	}

	for _, expected := range expectedNames {
		if !strings.Contains(output, expected) {
			t.Errorf("Expected %s in output", expected)
		}
	}
}

// TestGenerateElementSystemFieldDataSources_UsesNamespaceForPrefix tests namespace preference
func TestGenerateElementSystemFieldDataSources_UsesNamespaceForPrefix(t *testing.T) {
	t.Parallel()

	app := &discovery.App{
		ID:        "app-123",
		Name:      "Test App",
		Namespace: "testapp",
		DiscoveredElements: []*discovery.Element{
			{
				ID:        "elem-1",
				Name:      "Product Items", // Name with space
				Namespace: "productitems",  // Clean namespace
				Handle:    "PROD",          // Different handle
				Fields: []discovery.Field{
					{ID: "f1", Name: "Created By", Type: "user", SemanticTags: []string{"CREATED_BY"}},
				},
			},
		},
	}

	output := GenerateElementSystemFieldDataSources(app)

	// Should use namespace (productitems) not name or handle
	if !strings.Contains(output, `data "elementum_field" "productitems_created_by"`) {
		t.Error("Expected namespace 'productitems' to be used as prefix")
	}
	if !strings.Contains(output, `object_id = elementum_element.productitems.id`) {
		t.Error("Expected namespace 'productitems' in element reference")
	}
}

// Test GenerateCloudLinkDataSourcesFromMap handles duplicate names
func TestGenerateCloudLinkDataSourcesFromMap_DuplicateNames(t *testing.T) {
	cloudlinks := map[string]*discovery.CloudLink{
		"cl-1": {
			ID:   "cl-1",
			Name: "Snowflake",
		},
		"cl-2": {
			ID:   "cl-2",
			Name: "Snowflake", // Same name as cl-1
		},
	}

	result := GenerateCloudLinkDataSourcesFromMap(cloudlinks)

	// Should have both data sources with unique names
	if !strings.Contains(result, `data "elementum_cloudlink" "snowflake"`) {
		t.Errorf("First CloudLink data source not found")
	}

	// Note: Due to map iteration order being non-deterministic, we just check
	// that either _1 suffix exists OR we have two distinct snowflake blocks
	count := strings.Count(result, `data "elementum_cloudlink"`)
	if count != 2 {
		t.Errorf("Expected 2 CloudLink data sources, found %d", count)
	}
}

// =============================================================================
// Stored Function Data Source Tests
// =============================================================================

// Test GenerateStoredFunctionDataSources generates correct HCL
func TestGenerateStoredFunctionDataSources(t *testing.T) {
	tests := []struct {
		name         string
		functions    []*discovery.StoredFunction
		wantContains []string
		wantMissing  []string
	}{
		{
			name: "single stored function",
			functions: []*discovery.StoredFunction{
				{
					ID:            "sf-123",
					Name:          "CALCULATE_SHIPPING",
					DisplayName:   "Calculate Shipping",
					CloudLinkID:   "cl-1",
					CloudLinkName: "Production Snowflake",
				},
			},
			wantContains: []string{
				`# Stored Function data sources (discovered from procedure tasks)`,
				`data "elementum_stored_function" "calculate_shipping"`,
				`cloudlink_id = data.elementum_cloudlink.production_snowflake.id`,
				`name         = "Calculate Shipping"`,
			},
		},
		{
			name: "multiple stored functions",
			functions: []*discovery.StoredFunction{
				{
					ID:            "sf-1",
					DisplayName:   "Validate Order",
					CloudLinkName: "Snowflake Prod",
				},
				{
					ID:            "sf-2",
					DisplayName:   "Calculate Tax",
					CloudLinkName: "Snowflake Prod",
				},
			},
			wantContains: []string{
				`data "elementum_stored_function" "validate_order"`,
				`name         = "Validate Order"`,
				`data "elementum_stored_function" "calculate_tax"`,
				`name         = "Calculate Tax"`,
			},
		},
		{
			name:         "empty slice returns empty string",
			functions:    []*discovery.StoredFunction{},
			wantContains: []string{},
			wantMissing: []string{
				"data",
				"elementum_stored_function",
			},
		},
		{
			name:      "nil slice returns empty string",
			functions: nil,
			wantMissing: []string{
				"data",
			},
		},
		{
			name: "skips functions with empty display name",
			functions: []*discovery.StoredFunction{
				{
					ID:            "sf-no-name",
					Name:          "PROC_WITHOUT_DISPLAY",
					DisplayName:   "", // Empty display name
					CloudLinkName: "Snowflake",
				},
				{
					ID:            "sf-with-name",
					DisplayName:   "Valid Function",
					CloudLinkName: "Snowflake",
				},
			},
			wantContains: []string{
				`data "elementum_stored_function" "valid_function"`,
			},
			wantMissing: []string{
				"proc_without_display",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GenerateStoredFunctionDataSources(tt.functions)

			for _, want := range tt.wantContains {
				if !strings.Contains(result, want) {
					t.Errorf("expected result to contain %q, got:\n%s", want, result)
				}
			}

			for _, missing := range tt.wantMissing {
				if strings.Contains(result, missing) {
					t.Errorf("expected result NOT to contain %q, got:\n%s", missing, result)
				}
			}
		})
	}
}

// Test GenerateStoredFunctionDataSources handles duplicate display names
func TestGenerateStoredFunctionDataSources_DuplicateNames(t *testing.T) {
	functions := []*discovery.StoredFunction{
		{
			ID:            "sf-1",
			DisplayName:   "Calculate Tax",
			CloudLinkName: "Snowflake Prod",
		},
		{
			ID:            "sf-2",
			DisplayName:   "Calculate Tax", // Same display name
			CloudLinkName: "Snowflake Dev",
		},
	}

	result := GenerateStoredFunctionDataSources(functions)

	// Should have both data sources with unique names
	count := strings.Count(result, `data "elementum_stored_function"`)
	if count != 2 {
		t.Errorf("Expected 2 stored function data sources, found %d", count)
	}

	// First one should be "calculate_tax", second should be "calculate_tax_1"
	if !strings.Contains(result, `"calculate_tax"`) {
		t.Error("First data source should be named 'calculate_tax'")
	}
}

// =============================================================================
// Deduplication Tests for Tables and Datamines
// =============================================================================

// TestGenerateImportBlocks_DeduplicatesTables verifies that the same table ID
// appearing in both ReferencedTables and DiscoveredTables is only exported once.
// This is a defensive test - currently these lists are mutually exclusive due to
// the !recursive check in export.go, but we want to ensure deduplication works.
func TestGenerateImportBlocks_DeduplicatesTables(t *testing.T) {
	t.Parallel()

	// Create an app where the same table appears in both lists
	// (simulating a potential future edge case)
	sharedTable := &discovery.Table{
		ID:   "table-shared-123",
		Name: "Shared Table",
	}

	app := &discovery.App{
		ID:        "app_123",
		Name:      "Test App",
		Namespace: "testapp",
		// Same table in both lists (this shouldn't happen in practice, but test dedup)
		ReferencedTables: []*discovery.Table{
			sharedTable,
			{ID: "table-ref-only", Name: "Referenced Only Table"},
		},
		DiscoveredTables: []*discovery.Table{
			sharedTable, // Same table!
			{ID: "table-disc-only", Name: "Discovered Only Table"},
		},
	}

	selectedTypes := map[string]bool{
		"tables": true,
	}

	blocks := GenerateImportBlocks(app, selectedTypes)

	// Count table blocks
	tableCount := 0
	tableIDs := make(map[string]int)
	for _, block := range blocks {
		if block.ResourceType == "elementum_table" {
			tableCount++
			tableIDs[block.ID]++
		}
	}

	// Should have 3 unique tables (shared + ref-only + disc-only), not 4
	if tableCount != 3 {
		t.Errorf("Expected 3 unique table blocks (deduplicated), got %d", tableCount)
	}

	// Verify the shared table ID only appears once
	if tableIDs["table-shared-123"] != 1 {
		t.Errorf("Expected shared table to appear once, got %d times", tableIDs["table-shared-123"])
	}

	// Verify other tables are present
	if tableIDs["table-ref-only"] != 1 {
		t.Errorf("Expected table-ref-only to appear once, got %d times", tableIDs["table-ref-only"])
	}
	if tableIDs["table-disc-only"] != 1 {
		t.Errorf("Expected table-disc-only to appear once, got %d times", tableIDs["table-disc-only"])
	}
}

// TestGenerateImportBlocks_DeduplicatesDatamines verifies that the same datamine ID
// appearing in both ReferencedDatamines and DiscoveredDatamines is only exported once.
func TestGenerateImportBlocks_DeduplicatesDatamines(t *testing.T) {
	t.Parallel()

	// Create an app where the same datamine appears in both lists
	sharedDatamine := &discovery.Datamine{
		ID:      "dm-shared-456",
		Name:    "Shared Datamine",
		TableID: "table-abc",
	}

	app := &discovery.App{
		ID:        "app_123",
		Name:      "Test App",
		Namespace: "testapp",
		// Same datamine in both lists
		ReferencedDatamines: []*discovery.Datamine{
			sharedDatamine,
			{ID: "dm-ref-only", Name: "Referenced Only Datamine", TableID: "table-def"},
		},
		DiscoveredDatamines: []*discovery.Datamine{
			sharedDatamine, // Same datamine!
			{ID: "dm-disc-only", Name: "Discovered Only Datamine", TableID: "table-ghi"},
		},
	}

	selectedTypes := map[string]bool{
		"datamines": true,
	}

	blocks := GenerateImportBlocks(app, selectedTypes)

	// Count datamine blocks
	datamineCount := 0
	datamineIDs := make(map[string]int)
	for _, block := range blocks {
		if block.ResourceType == "elementum_datamine" {
			datamineCount++
			// Extract the datamine ID from import ID (format: table_id:datamine_id)
			parts := strings.Split(block.ID, ":")
			if len(parts) == 2 {
				datamineIDs[parts[1]]++
			} else {
				datamineIDs[block.ID]++
			}
		}
	}

	// Should have 3 unique datamines (shared + ref-only + disc-only), not 4
	if datamineCount != 3 {
		t.Errorf("Expected 3 unique datamine blocks (deduplicated), got %d", datamineCount)
	}

	// Verify the shared datamine ID only appears once
	if datamineIDs["dm-shared-456"] != 1 {
		t.Errorf("Expected shared datamine to appear once, got %d times", datamineIDs["dm-shared-456"])
	}
}

// TestGenerateImportBlocks_DeduplicatesTablesAndDatamines verifies that both
// tables and datamines are deduplicated correctly when both have overlaps.
func TestGenerateImportBlocks_DeduplicatesTablesAndDatamines(t *testing.T) {
	t.Parallel()

	app := &discovery.App{
		ID:        "app_123",
		Name:      "Test App",
		Namespace: "testapp",
		// Overlapping tables
		ReferencedTables: []*discovery.Table{
			{ID: "table-1", Name: "Table One"},
			{ID: "table-2", Name: "Table Two"},
		},
		DiscoveredTables: []*discovery.Table{
			{ID: "table-2", Name: "Table Two"}, // Duplicate!
			{ID: "table-3", Name: "Table Three"},
		},
		// Overlapping datamines
		ReferencedDatamines: []*discovery.Datamine{
			{ID: "dm-1", Name: "Datamine One", TableID: "table-1"},
			{ID: "dm-2", Name: "Datamine Two", TableID: "table-2"},
		},
		DiscoveredDatamines: []*discovery.Datamine{
			{ID: "dm-2", Name: "Datamine Two", TableID: "table-2"}, // Duplicate!
			{ID: "dm-3", Name: "Datamine Three", TableID: "table-3"},
		},
	}

	selectedTypes := map[string]bool{
		"tables":    true,
		"datamines": true,
	}

	blocks := GenerateImportBlocks(app, selectedTypes)

	// Count by resource type
	tableCount := 0
	datamineCount := 0
	for _, block := range blocks {
		switch block.ResourceType {
		case "elementum_table":
			tableCount++
		case "elementum_datamine":
			datamineCount++
		}
	}

	// Should have 3 unique tables (table-1, table-2, table-3)
	if tableCount != 3 {
		t.Errorf("Expected 3 unique table blocks, got %d", tableCount)
	}

	// Should have 3 unique datamines (dm-1, dm-2, dm-3)
	if datamineCount != 3 {
		t.Errorf("Expected 3 unique datamine blocks, got %d", datamineCount)
	}
}

// TestGenerateImportBlocks_TablesOnlyReferenced verifies normal behavior when
// tables only appear in ReferencedTables (non-recursive mode).
func TestGenerateImportBlocks_TablesOnlyReferenced(t *testing.T) {
	t.Parallel()

	app := &discovery.App{
		ID:        "app_123",
		Name:      "Test App",
		Namespace: "testapp",
		ReferencedTables: []*discovery.Table{
			{ID: "table-1", Name: "Customer Data"},
			{ID: "table-2", Name: "Order History"},
		},
		DiscoveredTables: nil, // Empty in non-recursive mode
	}

	selectedTypes := map[string]bool{
		"tables": true,
	}

	blocks := GenerateImportBlocks(app, selectedTypes)

	// Count table blocks
	tableCount := 0
	for _, block := range blocks {
		if block.ResourceType == "elementum_table" {
			tableCount++
		}
	}

	if tableCount != 2 {
		t.Errorf("Expected 2 table blocks from ReferencedTables, got %d", tableCount)
	}
}

// TestGenerateImportBlocks_TablesOnlyDiscovered verifies normal behavior when
// tables only appear in DiscoveredTables (recursive mode).
func TestGenerateImportBlocks_TablesOnlyDiscovered(t *testing.T) {
	t.Parallel()

	app := &discovery.App{
		ID:               "app_123",
		Name:             "Test App",
		Namespace:        "testapp",
		ReferencedTables: nil, // Empty in recursive mode
		DiscoveredTables: []*discovery.Table{
			{ID: "table-1", Name: "Customer Data"},
			{ID: "table-2", Name: "Order History"},
			{ID: "table-3", Name: "Product Catalog"},
		},
	}

	selectedTypes := map[string]bool{
		"tables": true,
	}

	blocks := GenerateImportBlocks(app, selectedTypes)

	// Count table blocks
	tableCount := 0
	for _, block := range blocks {
		if block.ResourceType == "elementum_table" {
			tableCount++
		}
	}

	if tableCount != 3 {
		t.Errorf("Expected 3 table blocks from DiscoveredTables, got %d", tableCount)
	}
}

// TestGenerateImportBlocks_DatamineImportIDFormat verifies that datamine import IDs
// include the table_id:datamine_id format when tableID is present.
func TestGenerateImportBlocks_DatamineImportIDFormat(t *testing.T) {
	t.Parallel()

	app := &discovery.App{
		ID:        "app_123",
		Name:      "Test App",
		Namespace: "testapp",
		ReferencedDatamines: []*discovery.Datamine{
			{ID: "dm-1", Name: "With Table", TableID: "table-abc"},
			{ID: "dm-2", Name: "Without Table", TableID: ""}, // No table ID
		},
	}

	selectedTypes := map[string]bool{
		"datamines": true,
	}

	blocks := GenerateImportBlocks(app, selectedTypes)

	// Find datamine blocks and check ID format
	for _, block := range blocks {
		if block.ResourceType == "elementum_datamine" {
			if block.ResourceName == "with_table" {
				// Should have table_id:datamine_id format
				if block.ID != "table-abc:dm-1" {
					t.Errorf("Expected ID 'table-abc:dm-1' for datamine with table, got %s", block.ID)
				}
			}
			if block.ResourceName == "without_table" {
				// Should just be datamine_id (no table prefix)
				if block.ID != "dm-2" {
					t.Errorf("Expected ID 'dm-2' for datamine without table, got %s", block.ID)
				}
			}
		}
	}
}

// TestGenerateImportBlocks_SeenResourceIDsTracksAcrossLists verifies that the
// seenResourceIDs map correctly tracks IDs across different resource lists,
// ensuring no duplicates even when the same ID appears in different contexts.
func TestGenerateImportBlocks_SeenResourceIDsTracksAcrossLists(t *testing.T) {
	t.Parallel()

	// This tests the case where a resource ID might appear in multiple places
	// We want to ensure it's only emitted once
	app := &discovery.App{
		ID:        "app_123",
		Name:      "Test App",
		Namespace: "testapp",
		// Multiple references to same table
		ReferencedTables: []*discovery.Table{
			{ID: "common-id", Name: "First Reference"},
		},
		DiscoveredTables: []*discovery.Table{
			{ID: "common-id", Name: "Second Reference"}, // Same ID!
		},
	}

	selectedTypes := map[string]bool{
		"tables": true,
	}

	blocks := GenerateImportBlocks(app, selectedTypes)

	// Count how many times the common-id appears
	count := 0
	for _, block := range blocks {
		if block.ResourceType == "elementum_table" && block.ID == "common-id" {
			count++
		}
	}

	if count != 1 {
		t.Errorf("Expected common-id to appear exactly once, got %d times", count)
	}
}

// TestGenerateImportBlocks_PreservesFirstEncounteredName verifies that when
// deduplicating, we keep the first occurrence (which preserves the name from
// the first list processed).
func TestGenerateImportBlocks_PreservesFirstEncounteredName(t *testing.T) {
	t.Parallel()

	app := &discovery.App{
		ID:        "app_123",
		Name:      "Test App",
		Namespace: "testapp",
		// ReferencedTables is processed first
		ReferencedTables: []*discovery.Table{
			{ID: "dup-id", Name: "First Name"},
		},
		// DiscoveredTables is processed second
		DiscoveredTables: []*discovery.Table{
			{ID: "dup-id", Name: "Second Name"}, // Different name, same ID
		},
	}

	selectedTypes := map[string]bool{
		"tables": true,
	}

	blocks := GenerateImportBlocks(app, selectedTypes)

	// Find the table block
	var tableBlock *ImportBlock
	for i, block := range blocks {
		if block.ResourceType == "elementum_table" && block.ID == "dup-id" {
			tableBlock = &blocks[i]
			break
		}
	}

	if tableBlock == nil {
		t.Fatal("Expected to find table block with ID dup-id")
	}

	// Should use the first name (from ReferencedTables)
	if tableBlock.ResourceName != "first_name" {
		t.Errorf("Expected resource name 'first_name' (from first list), got %s", tableBlock.ResourceName)
	}
}

// ============================================================================
// Resource Name Helper Function Tests (Issue #9 Fix)
// ============================================================================

func TestAppResourceName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		app      *discovery.App
		expected string
	}{
		{
			name: "uses namespace when available",
			app: &discovery.App{
				Name:      "Cases - Dev",
				Namespace: "casesdev",
			},
			expected: "casesdev",
		},
		{
			name: "falls back to name when no namespace",
			app: &discovery.App{
				Name:      "Test App",
				Namespace: "",
			},
			expected: "test_app",
		},
		{
			name: "sanitizes namespace",
			app: &discovery.App{
				Name:      "Some App",
				Namespace: "My-Namespace",
			},
			expected: "my_namespace",
		},
		{
			name: "sanitizes name fallback",
			app: &discovery.App{
				Name:      "App With Spaces - V2",
				Namespace: "",
			},
			expected: "app_with_spaces___v2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := AppResourceName(tt.app)
			if result != tt.expected {
				t.Errorf("AppResourceName() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestElementResourceName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		elem     *discovery.Element
		expected string
	}{
		{
			name: "uses namespace first",
			elem: &discovery.Element{
				Name:      "Action Components - Dev",
				Handle:    "actioncomponents",
				Namespace: "actioncomponentsdev",
			},
			expected: "actioncomponentsdev",
		},
		{
			name: "uses handle when no namespace",
			elem: &discovery.Element{
				Name:      "Test Element",
				Handle:    "testelementhandle",
				Namespace: "",
			},
			expected: "testelementhandle",
		},
		{
			name: "uses name when no namespace or handle",
			elem: &discovery.Element{
				Name:      "My Element",
				Handle:    "",
				Namespace: "",
			},
			expected: "my_element",
		},
		{
			name: "sanitizes namespace",
			elem: &discovery.Element{
				Name:      "Element",
				Handle:    "handle",
				Namespace: "My-Namespace",
			},
			expected: "my_namespace",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ElementResourceName(tt.elem)
			if result != tt.expected {
				t.Errorf("ElementResourceName() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestTaskResourceName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		task     *discovery.AspectTask
		expected string
	}{
		{
			name: "uses namespace first",
			task: &discovery.AspectTask{
				Name:      "Velocity Activity Tasks",
				Handle:    "velactivitytasks",
				Namespace: "velocityactivitytasks",
			},
			expected: "velocityactivitytasks",
		},
		{
			name: "uses handle when no namespace",
			task: &discovery.AspectTask{
				Name:      "Test Task",
				Handle:    "testtaskhandle",
				Namespace: "",
			},
			expected: "testtaskhandle",
		},
		{
			name: "uses name when no namespace or handle",
			task: &discovery.AspectTask{
				Name:      "My Task",
				Handle:    "",
				Namespace: "",
			},
			expected: "my_task",
		},
		{
			name: "sanitizes namespace",
			task: &discovery.AspectTask{
				Name:      "Task",
				Handle:    "handle",
				Namespace: "My-Namespace",
			},
			expected: "my_namespace",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := TaskResourceName(tt.task)
			if result != tt.expected {
				t.Errorf("TaskResourceName() = %q, want %q", result, tt.expected)
			}
		})
	}
}

// TestGenerateImportBlocks_MainAppUsesNamespace verifies that the main app resource
// uses namespace-first naming for consistency with discovered apps
func TestGenerateImportBlocks_MainAppUsesNamespace(t *testing.T) {
	t.Parallel()

	app := &discovery.App{
		ID:        "app-123",
		Name:      "Cases - Dev", // Friendly name with special chars
		Namespace: "casesdev",    // Clean namespace identifier
	}

	selectedTypes := map[string]bool{
		"fields": true, // Need at least one type to generate app block
	}

	blocks := GenerateImportBlocks(app, selectedTypes)

	// Find the app block
	var appBlock *ImportBlock
	for i, block := range blocks {
		if block.ResourceType == "elementum_app" {
			appBlock = &blocks[i]
			break
		}
	}

	if appBlock == nil {
		t.Fatal("Expected to find app import block")
	}

	// Should use namespace, not sanitized name
	if appBlock.ResourceName != "casesdev" {
		t.Errorf("Expected resource name 'casesdev' (from namespace), got %s", appBlock.ResourceName)
	}
}

// =============================================================================
// Agent View Export Tests (Issue #15 Fix)
// =============================================================================

// TestGenerateImportBlocks_AgentViewsSkippedWhenAgentsNotSelected verifies that
// agent views are NOT exported when views are selected but agents are NOT selected.
// This prevents the "undeclared agent" error when agent views reference agents
// that weren't exported.
func TestGenerateImportBlocks_AgentViewsSkippedWhenAgentsNotSelected(t *testing.T) {
	t.Parallel()

	app := &discovery.App{
		ID:        "app_123",
		Name:      "Test App",
		Namespace: "testapp",
		Views: []discovery.View{
			{ID: "view-list", Name: "All Records", Type: "AspectViewList"},
			{ID: "view-agent", Name: "AI Assistant", Type: "AspectViewAgent", AgentID: "agent-123"},
			{ID: "view-kanban", Name: "Board View", Type: "AspectViewKanban"},
		},
		Agents: []discovery.Agent{
			{ID: "agent-123", Name: "AI Assistant Agent"},
		},
	}

	// Views selected, but agents NOT selected
	selectedTypes := map[string]bool{
		"views": true,
		// "agents": false - intentionally not including agents
	}

	blocks := GenerateImportBlocks(app, selectedTypes)

	// Count view types
	listViewCount := 0
	agentViewCount := 0
	kanbanViewCount := 0
	for _, block := range blocks {
		switch block.ResourceType {
		case "elementum_list_view":
			listViewCount++
		case "elementum_agent_view":
			agentViewCount++
		case "elementum_kanban_view":
			kanbanViewCount++
		}
	}

	// Agent views should be skipped
	if agentViewCount != 0 {
		t.Errorf("Expected 0 agent views when agents not selected, got %d", agentViewCount)
	}

	// Other views should still be exported
	if listViewCount != 1 {
		t.Errorf("Expected 1 list view, got %d", listViewCount)
	}
	if kanbanViewCount != 1 {
		t.Errorf("Expected 1 kanban view, got %d", kanbanViewCount)
	}
}

// TestGenerateImportBlocks_AgentViewsExportedWhenAgentsSelected verifies that
// agent views ARE exported when both views and agents are selected.
func TestGenerateImportBlocks_AgentViewsExportedWhenAgentsSelected(t *testing.T) {
	t.Parallel()

	app := &discovery.App{
		ID:        "app_123",
		Name:      "Test App",
		Namespace: "testapp",
		Views: []discovery.View{
			{ID: "view-list", Name: "All Records", Type: "AspectViewList"},
			{ID: "view-agent", Name: "AI Assistant", Type: "AspectViewAgent", AgentID: "agent-123"},
		},
		Agents: []discovery.Agent{
			{ID: "agent-123", Name: "AI Assistant Agent"},
		},
	}

	// Both views and agents selected
	selectedTypes := map[string]bool{
		"views":  true,
		"agents": true,
	}

	blocks := GenerateImportBlocks(app, selectedTypes)

	// Count view types
	agentViewCount := 0
	agentCount := 0
	for _, block := range blocks {
		switch block.ResourceType {
		case "elementum_agent_view":
			agentViewCount++
		case "elementum_agent":
			agentCount++
		}
	}

	// Agent views should be exported when agents are selected
	if agentViewCount != 1 {
		t.Errorf("Expected 1 agent view when agents selected, got %d", agentViewCount)
	}

	// Agents should also be exported
	if agentCount != 1 {
		t.Errorf("Expected 1 agent, got %d", agentCount)
	}
}

// TestGenerateImportBlocks_DiscoveredAppAgentViewsSkippedWhenAgentsNotSelected verifies
// that agent views from discovered apps are skipped when agents are not selected.
// This is the specific scenario from Issue #15.
func TestGenerateImportBlocks_DiscoveredAppAgentViewsSkippedWhenAgentsNotSelected(t *testing.T) {
	t.Parallel()

	app := &discovery.App{
		ID:        "app_main",
		Name:      "Main App",
		Namespace: "mainapp",
		// Main app has views but NO agents
		Views: []discovery.View{
			{ID: "view-main-list", Name: "Main List", Type: "AspectViewList"},
		},
		// Discovered app has agent views AND agents
		DiscoveredApps: []*discovery.App{
			{
				ID:        "app_related",
				Name:      "Related App",
				Namespace: "relatedapp",
				Views: []discovery.View{
					{ID: "view-related-list", Name: "Related List", Type: "AspectViewList"},
					{ID: "view-related-agent", Name: "Lisa AI", Type: "AspectViewAgent", AgentID: "agent-lisa"},
				},
				Agents: []discovery.Agent{
					{ID: "agent-lisa", Name: "Lisa - Task Co-pilot"},
				},
			},
		},
	}

	// Views selected, but agents NOT selected
	// This simulates the bug scenario where main app has no agents,
	// so agents aren't auto-selected with --all
	selectedTypes := map[string]bool{
		"views": true,
		// "agents": false - agents not selected
	}

	blocks := GenerateImportBlocks(app, selectedTypes)

	// Count blocks
	listViewCount := 0
	agentViewCount := 0
	agentCount := 0
	for _, block := range blocks {
		switch block.ResourceType {
		case "elementum_list_view":
			listViewCount++
		case "elementum_agent_view":
			agentViewCount++
		case "elementum_agent":
			agentCount++
		}
	}

	// Agent views from discovered app should be skipped
	if agentViewCount != 0 {
		t.Errorf("Expected 0 agent views from discovered app when agents not selected, got %d", agentViewCount)
	}

	// List views from both apps should be exported
	if listViewCount != 2 {
		t.Errorf("Expected 2 list views (main + discovered), got %d", listViewCount)
	}

	// No agents should be exported
	if agentCount != 0 {
		t.Errorf("Expected 0 agents when agents not selected, got %d", agentCount)
	}
}

// TestGenerateImportBlocks_DiscoveredElementAgentViewsSkippedWhenAgentsNotSelected verifies
// that agent views from discovered elements are skipped when agents are not selected.
func TestGenerateImportBlocks_DiscoveredElementAgentViewsSkippedWhenAgentsNotSelected(t *testing.T) {
	t.Parallel()

	app := &discovery.App{
		ID:        "app_main",
		Name:      "Main App",
		Namespace: "mainapp",
		DiscoveredElements: []*discovery.Element{
			{
				ID:        "elem_products",
				Name:      "Products",
				Namespace: "products",
				Views: []discovery.View{
					{ID: "view-elem-list", Name: "Product List", Type: "AspectViewList"},
					{ID: "view-elem-agent", Name: "Product AI", Type: "AspectViewAgent", AgentID: "agent-prod"},
				},
			},
		},
	}

	selectedTypes := map[string]bool{
		"views": true,
		// "agents": false
	}

	blocks := GenerateImportBlocks(app, selectedTypes)

	// Count blocks
	agentViewCount := 0
	listViewCount := 0
	for _, block := range blocks {
		switch block.ResourceType {
		case "elementum_agent_view":
			agentViewCount++
		case "elementum_list_view":
			listViewCount++
		}
	}

	// Agent views from discovered element should be skipped
	if agentViewCount != 0 {
		t.Errorf("Expected 0 agent views from discovered element when agents not selected, got %d", agentViewCount)
	}

	// List views should be exported
	if listViewCount != 1 {
		t.Errorf("Expected 1 list view from discovered element, got %d", listViewCount)
	}
}

// TestGenerateImportBlocks_DiscoveredTaskAgentViewsSkippedWhenAgentsNotSelected verifies
// that agent views from discovered tasks are skipped when agents are not selected.
func TestGenerateImportBlocks_DiscoveredTaskAgentViewsSkippedWhenAgentsNotSelected(t *testing.T) {
	t.Parallel()

	app := &discovery.App{
		ID:        "app_main",
		Name:      "Main App",
		Namespace: "mainapp",
		DiscoveredTasks: []*discovery.AspectTask{
			{
				ID:        "task_items",
				Name:      "Task Items",
				Namespace: "taskitems",
				Views: []discovery.View{
					{ID: "view-task-list", Name: "Task List", Type: "AspectViewList"},
					{ID: "view-task-agent", Name: "Task AI", Type: "AspectViewAgent", AgentID: "agent-task"},
				},
			},
		},
	}

	selectedTypes := map[string]bool{
		"views": true,
		// "agents": false
	}

	blocks := GenerateImportBlocks(app, selectedTypes)

	// Count blocks
	agentViewCount := 0
	listViewCount := 0
	for _, block := range blocks {
		switch block.ResourceType {
		case "elementum_agent_view":
			agentViewCount++
		case "elementum_list_view":
			listViewCount++
		}
	}

	// Agent views from discovered task should be skipped
	if agentViewCount != 0 {
		t.Errorf("Expected 0 agent views from discovered task when agents not selected, got %d", agentViewCount)
	}

	// List views should be exported
	if listViewCount != 1 {
		t.Errorf("Expected 1 list view from discovered task, got %d", listViewCount)
	}
}
