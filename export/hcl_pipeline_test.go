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

func TestExportPipeline_CLIOnly(t *testing.T) {
	app := &discovery.App{
		ID:        "app-1",
		Name:      "My App",
		Namespace: "my-app",
	}
	imports := []ImportBlock{
		{ResourceType: "elementum_app", ResourceName: "my_app", ID: "app-1"},
	}

	pipeline := NewExportPipeline(app, imports)
	pipeline.AddCLIBlocks()

	result := pipeline.Execute()

	if !strings.Contains(result, `resource "elementum_app" "my_app"`) {
		t.Error("expected app resource in output")
	}
	if !strings.Contains(result, `"My App"`) {
		t.Error("expected app name in output")
	}
}

func TestExportPipeline_CLIWithAutomations(t *testing.T) {
	app := &discovery.App{
		ID:        "app-1",
		Name:      "My App",
		Namespace: "my-app",
		Automations: []discovery.Automation{
			{
				ID:           "auto-1",
				Name:         "Process",
				HasPublished: true,
				Status:       "ACTIVE",
			},
		},
	}
	imports := []ImportBlock{
		{ResourceType: "elementum_app", ResourceName: "my_app", ID: "app-1"},
		{ResourceType: "elementum_automation", ResourceName: "process", ID: "auto-1"},
	}

	pipeline := NewExportPipeline(app, imports)
	pipeline.AddCLIBlocks()

	result := pipeline.Execute()

	if !strings.Contains(result, `"My App"`) {
		t.Error("expected app name in output")
	}
	if !strings.Contains(result, `"elementum_automation"`) {
		t.Error("expected automation resource from CLI generators")
	}
}

func TestExportPipeline_MultiFile(t *testing.T) {
	app := &discovery.App{
		ID:        "app-1",
		Name:      "Test App",
		Namespace: "test-app",
		Automations: []discovery.Automation{
			{
				ID:           "auto-1",
				Name:         "Process",
				HasPublished: true,
				Status:       "ACTIVE",
			},
		},
	}
	imports := []ImportBlock{
		{ResourceType: "elementum_app", ResourceName: "test_app", ID: "app-1"},
		{ResourceType: "elementum_automation", ResourceName: "process", ID: "auto-1"},
	}

	pipeline := NewExportPipeline(app, imports)
	pipeline.AddCLIBlocks()

	result := pipeline.ExecuteMultiFile()

	if len(result.FileGroups) == 0 {
		t.Fatal("expected at least 1 file group in multi-file export")
	}

	hasAuto := false
	for _, fg := range result.FileGroups {
		if strings.Contains(fg.FileName, "automation") {
			hasAuto = true
			break
		}
	}
	if !hasAuto {
		t.Error("expected an automation file group")
	}
}

func TestExportPipeline_Deduplication(t *testing.T) {
	app := &discovery.App{
		ID:        "app-1",
		Name:      "Test App",
		Namespace: "test-app",
		Automations: []discovery.Automation{
			{
				ID:           "auto-1",
				Name:         "Process",
				HasPublished: true,
				Status:       "ACTIVE",
			},
		},
	}
	imports := []ImportBlock{
		{ResourceType: "elementum_app", ResourceName: "test_app", ID: "app-1"},
		{ResourceType: "elementum_automation", ResourceName: "process", ID: "auto-1"},
	}

	pipeline := NewExportPipeline(app, imports)
	pipeline.AddCLIBlocks()

	// Add duplicate blocks manually
	pipeline.AddBlocks([]*HCLBlock{
		NewResourceBlock("elementum_automation", "process"),
	})

	blocks := pipeline.GetMergedBlocks()

	autoCount := 0
	for _, b := range blocks {
		if len(b.Labels) >= 2 && b.Labels[0] == "elementum_automation" && b.Labels[1] == "process" {
			autoCount++
		}
	}
	if autoCount != 1 {
		t.Errorf("expected 1 automation block after dedup, got %d", autoCount)
	}
}

func TestExportPipeline_TransformsApplied(t *testing.T) {
	app := &discovery.App{
		ID:        "app-1",
		Name:      "Test App",
		Namespace: "test-app",
		DiscoveredTables: []*discovery.Table{
			{
				ID:   "table-id-1",
				Name: "My Table",
			},
		},
	}
	imports := []ImportBlock{
		{ResourceType: "elementum_app", ResourceName: "test_app", ID: "app-1"},
		{ResourceType: "elementum_table", ResourceName: "my_table", ID: "table-id-1"},
	}

	pipeline := NewExportPipeline(app, imports)
	pipeline.AddCLIBlocks()

	blocks := pipeline.GetMergedBlocks()

	for _, b := range blocks {
		if len(b.Labels) >= 2 && b.Labels[0] == "elementum_table" {
			for _, attr := range b.Body.Attributes {
				if _, isNull := attr.Value.(HCLNull); isNull {
					t.Error("expected null attributes to be stripped")
				}
			}
		}
	}
}

func TestExportPipeline_AddTofuOutput_Deprecated(t *testing.T) {
	app := &discovery.App{
		ID:        "app-1",
		Name:      "Test App",
		Namespace: "test-app",
	}
	imports := []ImportBlock{
		{ResourceType: "elementum_app", ResourceName: "test_app", ID: "app-1"},
	}

	pipeline := NewExportPipeline(app, imports)
	err := pipeline.AddTofuOutput("anything")
	if err != nil {
		t.Errorf("AddTofuOutput should not error (deprecated no-op), got: %v", err)
	}
}

func TestExportPipeline_Fields(t *testing.T) {
	app := &discovery.App{
		ID:        "app-1",
		Name:      "Test App",
		Namespace: "test-app",
		Fields: []discovery.Field{
			{ID: "f-1", Name: "Email", Type: "text"},
			{ID: "f-2", Name: "Priority", Type: "dropdown", Options: []discovery.FieldOption{
				{Label: "Low", Color: "#green"},
				{Label: "High", Color: "#red"},
			}},
		},
	}
	imports := []ImportBlock{
		{ResourceType: "elementum_app", ResourceName: "test_app", ID: "app-1"},
		{ResourceType: "elementum_text_field", ResourceName: "email", ID: "app-1:f-1"},
		{ResourceType: "elementum_dropdown_field", ResourceName: "priority", ID: "app-1:f-2"},
	}

	pipeline := NewExportPipeline(app, imports)
	pipeline.AddCLIBlocks()

	result := pipeline.Execute()

	if !strings.Contains(result, `resource "elementum_text_field" "email"`) {
		t.Error("expected text field in output")
	}
	if !strings.Contains(result, `resource "elementum_dropdown_field" "priority"`) {
		t.Error("expected dropdown field in output")
	}
	if !strings.Contains(result, `"Low"`) {
		t.Error("expected dropdown options in output")
	}
}
