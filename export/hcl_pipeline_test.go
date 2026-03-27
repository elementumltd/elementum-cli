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

func TestExportPipeline_TofuOnly(t *testing.T) {
	tofuOutput := `
resource "elementum_app" "my_app" {
  name      = "My App"
  namespace = "my-app"
  options   = null
}
`
	app := &discovery.App{
		ID:        "app-1",
		Name:      "My App",
		Namespace: "my-app",
	}
	imports := []ImportBlock{
		{ResourceType: "elementum_app", ResourceName: "my_app", ID: "app-1"},
	}

	pipeline := NewExportPipeline(app, imports)
	err := pipeline.AddTofuOutput(tofuOutput)
	if err != nil {
		t.Fatalf("AddTofuOutput error: %v", err)
	}

	result := pipeline.Execute()

	// Should have the app resource
	if !strings.Contains(result, `resource "elementum_app" "my_app"`) {
		t.Error("expected app resource in output")
	}

	// Null attributes should be stripped
	if strings.Contains(result, "null") {
		t.Error("expected null attributes to be stripped")
	}
}

func TestExportPipeline_MergesCLIOverTofu(t *testing.T) {
	// Tofu generates a minimal app block
	tofuOutput := `
resource "elementum_app" "my_app" {
  name      = "My App"
  namespace = "my-app"
}
`
	// CLI generates a richer automation block
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
	err := pipeline.AddTofuOutput(tofuOutput)
	if err != nil {
		t.Fatalf("AddTofuOutput error: %v", err)
	}
	pipeline.AddCLIBlocks()

	result := pipeline.Execute()

	// Should have the app resource from tofu
	if !strings.Contains(result, `"My App"`) {
		t.Error("expected app name in output")
	}

	// Should have the automation from CLI generators
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

	// Check that automation file exists
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
	// Both tofu and CLI produce the same block - should be deduplicated
	tofuOutput := `
resource "elementum_automation" "process" {
  app_id = "app-1"
  name   = "Process"
}
`
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
	err := pipeline.AddTofuOutput(tofuOutput)
	if err != nil {
		t.Fatalf("AddTofuOutput error: %v", err)
	}
	pipeline.AddCLIBlocks()

	blocks := pipeline.GetMergedBlocks()

	// Count automation blocks
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
	tofuOutput := `
resource "elementum_table" "my_table" {
  name      = "My Table"
  source_id = "table-id-1"
  options   = null
}
`
	app := &discovery.App{
		ID:        "app-1",
		Name:      "Test App",
		Namespace: "test-app",
	}
	imports := []ImportBlock{
		{ResourceType: "elementum_app", ResourceName: "test_app", ID: "app-1"},
		{ResourceType: "elementum_table", ResourceName: "my_table", ID: "table-id-1"},
	}

	pipeline := NewExportPipeline(app, imports)
	err := pipeline.AddTofuOutput(tofuOutput)
	if err != nil {
		t.Fatalf("AddTofuOutput error: %v", err)
	}

	blocks := pipeline.GetMergedBlocks()

	// Find the table block
	for _, b := range blocks {
		if len(b.Labels) >= 2 && b.Labels[0] == "elementum_table" {
			// Null should be stripped
			for _, attr := range b.Body.Attributes {
				if _, isNull := attr.Value.(HCLNull); isNull {
					t.Error("expected null attributes to be stripped")
				}
			}

			// Self-referencing source_id should be stripped
			for _, attr := range b.Body.Attributes {
				if attr.Name == "source_id" {
					t.Error("expected self-referencing source_id to be stripped")
				}
			}
		}
	}
}
