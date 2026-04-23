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
	"github.com/elementumltd/elementum-cli/discovery"
)

// ExportSingleAutomationHCL generates HCL for a single automation.
// It creates a minimal App struct containing just this automation,
// generates import blocks, and runs the HCL generators.
func ExportSingleAutomationHCL(aspectID, aspectName string, automation discovery.Automation) string {
	// Build minimal App
	app := &discovery.App{
		ID:          aspectID,
		Name:        aspectName,
		Automations: []discovery.Automation{automation},
	}

	// Generate imports for this automation
	imports := generateSingleAutomationImports(aspectID, &automation)

	// Build UUID map (empty - raw UUIDs will be in output)
	uuidMap := make(map[string]string)

	// Generate HCL
	gen := NewAutomationHCLGenerator(app, imports, uuidMap)
	return gen.GenerateAll()
}

// generateSingleAutomationImports creates ImportBlock entries for a single automation.
func generateSingleAutomationImports(aspectID string, automation *discovery.Automation) []ImportBlock {
	var imports []ImportBlock

	sanitizedName := SanitizeName(automation.Name)

	// Automation resource
	imports = append(imports, ImportBlock{
		ResourceType: "elementum_automation",
		ResourceName: sanitizedName,
		ID:           BuildImportID("elementum_automation", map[string]string{"app_id": aspectID, "automation_id": automation.ID}),
	})

	// Workflow publish
	imports = append(imports, ImportBlock{
		ResourceType: "elementum_workflow_publish",
		ResourceName: sanitizedName,
		ID:           automation.ID,
	})

	// Triggers
	for _, trigger := range automation.Triggers {
		if trigger.Type == "unknown" {
			continue
		}
		triggerResourceType := "elementum_" + trigger.Type + "_trigger"
		triggerName := sanitizedName + "_" + SanitizeName(trigger.Type) + "_0"
		imports = append(imports, ImportBlock{
			ResourceType: triggerResourceType,
			ResourceName: triggerName,
			ID:           BuildTriggerImportID(automation.ID, trigger.ID, trigger.Type),
		})
	}

	// Tasks
	for _, task := range automation.Tasks {
		if task.Type == "unknown" {
			continue
		}
		taskResourceType := "elementum_" + task.Type + "_task"
		taskName := sanitizedName + "_" + SanitizeName(task.Name)
		imports = append(imports, ImportBlock{
			ResourceType: taskResourceType,
			ResourceName: taskName,
			ID:           BuildTaskImportID(automation.WorkflowID, task.ID, task.Type),
		})
	}

	return imports
}
