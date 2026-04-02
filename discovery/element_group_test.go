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

package discovery

import (
	"testing"
)

func TestElement_Structure(t *testing.T) {
	t.Parallel()

	element := Element{
		ID:          "elem_123",
		Name:        "Products",
		Handle:      "PROD",
		Namespace:   "products",
		Description: "Product catalog",
		Icon:        "box",
		Color:       "#3B82F6",
		CategoryID:  "cat_456",
	}

	if element.ID != "elem_123" {
		t.Errorf("Expected ID elem_123, got %s", element.ID)
	}
	if element.Handle != "PROD" {
		t.Errorf("Expected Handle PROD, got %s", element.Handle)
	}
	if element.Namespace != "products" {
		t.Errorf("Expected Namespace products, got %s", element.Namespace)
	}
}

func TestGroup_Structure(t *testing.T) {
	t.Parallel()

	group := Group{
		ID:          "grp_789",
		Name:        "Engineering Team",
		Approver:    true,
		Assignable:  true,
		Mentionable: false,
		Watchable:   true,
		Dynamic:     false,
		Tags:        []string{"engineering", "dev"},
		Members:     []string{"user_1", "user_2"},
	}

	if group.ID != "grp_789" {
		t.Errorf("Expected ID grp_789, got %s", group.ID)
	}
	if !group.Approver {
		t.Error("Expected Approver to be true")
	}
	if len(group.Members) != 2 {
		t.Errorf("Expected 2 members, got %d", len(group.Members))
	}
}

func TestCloudLink_Structure(t *testing.T) {
	t.Parallel()

	cloudlink := CloudLink{
		ID:     "cl_abc",
		Name:   "Production API",
		Type:   "api",
		System: false,
		Valid:  true,
	}

	if cloudlink.ID != "cl_abc" {
		t.Errorf("Expected ID cl_abc, got %s", cloudlink.ID)
	}
	if cloudlink.Type != "api" {
		t.Errorf("Expected Type api, got %s", cloudlink.Type)
	}
	if !cloudlink.Valid {
		t.Error("Expected Valid to be true")
	}
}

func TestAIFileReader_Structure(t *testing.T) {
	t.Parallel()

	reader := AIFileReader{
		ID:   "reader_123",
		Name: "Invoice Parser",
	}

	if reader.ID != "reader_123" {
		t.Errorf("Expected ID reader_123, got %s", reader.ID)
	}
	if reader.Name != "Invoice Parser" {
		t.Errorf("Expected Name 'Invoice Parser', got %s", reader.Name)
	}
}

func TestApprovalProcess_Structure(t *testing.T) {
	t.Parallel()

	approval := ApprovalProcess{
		ID:   "approval_456",
		Name: "Manager Approval",
	}

	if approval.ID != "approval_456" {
		t.Errorf("Expected ID approval_456, got %s", approval.ID)
	}
	if approval.Name != "Manager Approval" {
		t.Errorf("Expected Name 'Manager Approval', got %s", approval.Name)
	}
}

func TestApp_WithAllResources(t *testing.T) {
	t.Parallel()

	app := App{
		ID:        "app_123",
		Name:      "Test App",
		Namespace: "testapp",
		Fields: []Field{
			{ID: "field_1", Name: "Title", Type: "text"},
			{ID: "field_2", Name: "Status", Type: "dropdown"},
		},
		Layouts: []Layout{
			{ID: "layout_1", Name: "Open"},
		},
		Flows: []Flow{
			{ID: "flow_1", Name: "Main Flow"},
		},
		Agents: []Agent{
			{ID: "agent_1", Name: "Support Agent"},
		},
		Widgets: []Widget{
			{ID: "widget_1", Name: "Dashboard"},
		},
		AIFileReaders: []AIFileReader{
			{ID: "reader_1", Name: "Invoice Parser"},
		},
		Approvals: []ApprovalProcess{
			{ID: "approval_1", Name: "Manager Approval"},
		},
	}

	if len(app.Fields) != 2 {
		t.Errorf("Expected 2 fields, got %d", len(app.Fields))
	}
	if len(app.AIFileReaders) != 1 {
		t.Errorf("Expected 1 AI file reader, got %d", len(app.AIFileReaders))
	}
	if len(app.Approvals) != 1 {
		t.Errorf("Expected 1 approval process, got %d", len(app.Approvals))
	}
}

func TestObjectSummary_Types(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		summary  ObjectSummary
		wantType string
	}{
		{
			name: "app summary",
			summary: ObjectSummary{
				ID:        "app_123",
				Name:      "Test App",
				Type:      "App",
				Namespace: "testapp",
			},
			wantType: "App",
		},
		{
			name: "element summary",
			summary: ObjectSummary{
				ID:        "elem_456",
				Name:      "Products",
				Type:      "Element",
				Namespace: "products",
			},
			wantType: "Element",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.summary.Type != tt.wantType {
				t.Errorf("Expected type %s, got %s", tt.wantType, tt.summary.Type)
			}
		})
	}
}

func TestDisplayBlock_Structure(t *testing.T) {
	t.Parallel()

	block := DisplayBlock{
		ID:   "block_123",
		Type: "group",
		Name: "Details Section",
	}

	if block.ID != "block_123" {
		t.Errorf("Expected ID block_123, got %s", block.ID)
	}
	if block.Type != "group" {
		t.Errorf("Expected Type group, got %s", block.Type)
	}
	if block.Name != "Details Section" {
		t.Errorf("Expected Name 'Details Section', got %s", block.Name)
	}
}

func TestLayout_WithDisplayBlocks(t *testing.T) {
	t.Parallel()

	layout := Layout{
		ID:   "stage_open",
		Name: "Open",
		DisplayBlocks: []DisplayBlock{
			{ID: "block_1", Type: "group", Name: "Basic Info"},
			{ID: "block_2", Type: "field"},
			{ID: "block_3", Type: "activity_log"},
		},
	}

	if len(layout.DisplayBlocks) != 3 {
		t.Errorf("Expected 3 display blocks, got %d", len(layout.DisplayBlocks))
	}
	if layout.DisplayBlocks[0].Type != "group" {
		t.Errorf("Expected first block type 'group', got %s", layout.DisplayBlocks[0].Type)
	}
	if layout.DisplayBlocks[0].Name != "Basic Info" {
		t.Errorf("Expected first block name 'Basic Info', got %s", layout.DisplayBlocks[0].Name)
	}
}

func TestFlow_Structure(t *testing.T) {
	t.Parallel()

	flow := Flow{
		ID:   "app_123",
		Name: "Sales Flow",
	}

	if flow.ID != "app_123" {
		t.Errorf("Expected ID app_123, got %s", flow.ID)
	}
	if flow.Name != "Sales Flow" {
		t.Errorf("Expected Name 'Sales Flow', got %s", flow.Name)
	}
}

func TestApp_WithFlows(t *testing.T) {
	t.Parallel()

	app := App{
		ID:        "app_123",
		Name:      "Sales App",
		Namespace: "sales",
		Flows: []Flow{
			{ID: "app_123", Name: "Sales App Flow"},
		},
	}

	if len(app.Flows) != 1 {
		t.Errorf("Expected 1 flow, got %d", len(app.Flows))
	}
	if app.Flows[0].ID != "app_123" {
		t.Errorf("Expected flow ID to match app ID, got %s", app.Flows[0].ID)
	}
}
