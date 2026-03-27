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
	"fmt"
	"strings"

	"github.com/elementumltd/elementum-cli/discovery"
)

// AutomationHCLGenerator generates HCL for automation resources
type AutomationHCLGenerator struct {
	app     *discovery.App
	imports []ImportBlock
	uuidMap map[string]string

	// Sub-generators for triggers and tasks
	triggerGen *TriggerHCLGenerator
	taskGen    *TaskHCLGenerator
}

// NewAutomationHCLGenerator creates a new automation HCL generator
func NewAutomationHCLGenerator(app *discovery.App, imports []ImportBlock, uuidMap map[string]string) *AutomationHCLGenerator {
	allAutomations := app.AllAutomations()
	return &AutomationHCLGenerator{
		app:        app,
		imports:    imports,
		uuidMap:    uuidMap,
		triggerGen: NewTriggerHCLGenerator(imports, uuidMap, allAutomations),
		taskGen:    NewTaskHCLGenerator(app, imports, uuidMap, allAutomations),
	}
}

// GenerateAll generates HCL for all automation resources (automations, triggers, tasks)
func (g *AutomationHCLGenerator) GenerateAll() string {
	return SerializeBlocks(g.GenerateAllIR())
}

// GenerateAutomationsOnly generates HCL for automation resources without triggers and tasks
func (g *AutomationHCLGenerator) GenerateAutomationsOnly() string {
	return SerializeBlocks(g.GenerateAutomationsOnlyIR())
}

// GenerateTriggersOnly generates HCL for all trigger resources
func (g *AutomationHCLGenerator) GenerateTriggersOnly() string {
	return SerializeBlocks(g.triggerGen.GenerateAllIR())
}

// GenerateTasksOnly generates HCL for all task resources
func (g *AutomationHCLGenerator) GenerateTasksOnly() string {
	return SerializeBlocks(g.taskGen.GenerateAllIR())
}

// getAutomationResourceName finds the resource name from imports for an automation ID
func (g *AutomationHCLGenerator) getAutomationResourceName(automationID string) string {
	for _, imp := range g.imports {
		if imp.ResourceType == "elementum_automation" && strings.Contains(imp.ID, automationID) {
			return imp.ResourceName
		}
	}
	return ""
}

// GetTriggerGenerator returns the trigger HCL generator for direct access
func (g *AutomationHCLGenerator) GetTriggerGenerator() *TriggerHCLGenerator {
	return g.triggerGen
}

// GetTaskGenerator returns the task HCL generator for direct access
func (g *AutomationHCLGenerator) GetTaskGenerator() *TaskHCLGenerator {
	return g.taskGen
}

// GenerateAllIR generates IR blocks for all automation resources (automations, triggers, tasks, workflow_publish)
func (g *AutomationHCLGenerator) GenerateAllIR() []*HCLBlock {
	var blocks []*HCLBlock
	blocks = append(blocks, g.GenerateAutomationsOnlyIR()...)
	blocks = append(blocks, g.triggerGen.GenerateAllIR()...)
	blocks = append(blocks, g.taskGen.GenerateAllIR()...)
	blocks = append(blocks, g.GenerateWorkflowPublishIR()...)
	return blocks
}

// GenerateWorkflowPublishIR generates IR blocks for workflow_publish resources.
// Each automation needs a workflow_publish to activate its tasks.
func (g *AutomationHCLGenerator) GenerateWorkflowPublishIR() []*HCLBlock {
	var blocks []*HCLBlock

	// Helper to generate workflow_publish blocks for a set of automations
	generateForAutomations := func(automations []discovery.Automation) {
		for _, automation := range automations {
			if !automation.HasPublished || automation.Status != "ACTIVE" {
				continue
			}

			// Find the workflow_publish import for this automation
			var publishResourceName string
			for _, imp := range g.imports {
				if imp.ResourceType == "elementum_workflow_publish" &&
					imp.ID == automation.ID {
					publishResourceName = imp.ResourceName
					break
				}
			}
			if publishResourceName == "" {
				continue
			}

			// Get automation resource name for the reference
			automationResourceName := g.getAutomationResourceName(automation.ID)
			if automationResourceName == "" {
				continue
			}

			b := NewResourceBlock("elementum_workflow_publish", publishResourceName)
			b.SetAttr("automation_id", Ref("elementum_automation."+automationResourceName+".id"))

			// Build task_ids list - collect all task references
			var taskRefs []HCLValue
			for _, task := range automation.Tasks {
				if task.Type == "unknown" {
					continue
				}
				taskType := "elementum_" + task.Type + "_task"
				// Find the task import to get resource name
				for _, imp := range g.imports {
					if imp.ResourceType == taskType && strings.HasSuffix(imp.ID, ":"+task.ID) {
						taskRef := Ref(taskType + "." + imp.ResourceName + ".id")
						taskRefs = append(taskRefs, taskRef)
						break
					}
				}
			}

			if len(taskRefs) > 0 {
				b.SetAttr("task_ids", List(taskRefs...))
			}

			// Add outputs if present (for on-demand automations)
			if len(automation.Outputs) > 0 {
				var outputBlocks []HCLValue
				for _, output := range automation.Outputs {
					// Build the value reference from the output value
					valueRef := g.buildOutputValueJSON(output.Value, automation)
					if valueRef == "" {
						continue
					}
					outputBlock := Obj(
						Attr("name", Str(output.Name)),
						Attr("value", Ref(valueRef)),
					)
					outputBlocks = append(outputBlocks, outputBlock)
				}
				if len(outputBlocks) > 0 {
					b.SetAttr("outputs", List(outputBlocks...))
				}
			}

			blocks = append(blocks, b)
		}
	}

	// Generate for main app automations
	generateForAutomations(g.app.Automations)

	// Generate for discovered app automations
	for _, da := range g.app.DiscoveredApps {
		generateForAutomations(da.Automations)
	}

	// Generate for discovered element automations
	for _, de := range g.app.DiscoveredElements {
		generateForAutomations(de.Automations)
	}

	// Generate for discovered task automations
	for _, dt := range g.app.DiscoveredTasks {
		generateForAutomations(dt.Automations)
	}

	return blocks
}

// buildOutputValueJSON converts a workflow output value reference to a Terraform expression.
// Handles multiple reference types:
// - taskReference with "result" property (uses last task in workflow)
// - taskReference with "record.<field_id>" format (trigger field reference)
// - variableReference with "variable|<task_id>" format
// - triggerReference for trigger values
func (g *AutomationHCLGenerator) buildOutputValueJSON(valueRef map[string]interface{}, automation discovery.Automation) string {
	if valueRef == nil {
		return ""
	}

	// Check for variableReference first (format: "variable|<variable_id>")
	// Note: variableId may not match any task ID if the variable is defined inside a switch/for_each
	if varRef, ok := valueRef["variableReference"].(map[string]interface{}); ok && varRef != nil {
		refName := getStringValue(varRef, "name")
		if strings.HasPrefix(refName, "variable|") {
			variableID := strings.TrimPrefix(refName, "variable|")
			// Try to find a variable/update_variable task with this ID
			for _, task := range automation.Tasks {
				if task.ID == variableID && (task.Type == "variable" || task.Type == "update_variable") {
					taskType := "elementum_" + task.Type + "_task"
					for _, imp := range g.imports {
						if imp.ResourceType == taskType && strings.HasSuffix(imp.ID, ":"+task.ID) {
							varName := getStringValue(task.RawData, "variableName")
							if varName == "" {
								varName = "result"
							}
							return taskType + "." + imp.ResourceName + ".refs[\"" + varName + "\"]"
						}
					}
				}
			}
			// If not found, the variable is inside a switch/for_each block and cannot be exported.
			// Per workflow_publish docs: outputs can only reference top-level tasks.
			// Return empty to skip this output.
		}
	}

	// Check for taskReference
	if taskRef, ok := valueRef["taskReference"].(map[string]interface{}); ok && taskRef != nil {
		refName := getStringValue(taskRef, "name")
		if refName != "" {
			// Format 1: "task.<task_id>.<property>"
			if strings.HasPrefix(refName, "task.") {
				parts := strings.SplitN(refName[5:], ".", 2)
				if len(parts) >= 2 {
					taskID := parts[0]
					property := parts[1]
					for _, task := range automation.Tasks {
						if task.ID == taskID {
							taskType := "elementum_" + task.Type + "_task"
							for _, imp := range g.imports {
								if imp.ResourceType == taskType && strings.HasSuffix(imp.ID, ":"+task.ID) {
									return taskType + "." + imp.ResourceName + ".refs[\"" + property + "\"]"
								}
							}
						}
					}
				}
			}

			// Format 2: "record.<field_id>" - this is a trigger field reference, use automation refs
			if strings.HasPrefix(refName, "record.") {
				automationResourceName := g.getAutomationResourceName(automation.ID)
				if automationResourceName != "" {
					return "elementum_automation." + automationResourceName + ".refs[\"" + refName + "\"]"
				}
			}

			// Format 3: Simple property name like "result" - uses the last task (run_agent, procedure, etc.)
			if !strings.Contains(refName, ".") && !strings.Contains(refName, "|") {
				// Find the leaf task (last task in the workflow)
				var leafTask *discovery.Task
				for i := range automation.Tasks {
					task := &automation.Tasks[i]
					// Check if this task has any children
					isLeaf := true
					for _, otherTask := range automation.Tasks {
						if otherTask.ParentID == task.ID {
							isLeaf = false
							break
						}
					}
					if isLeaf {
						leafTask = task
						break
					}
				}
				if leafTask != nil {
					taskType := "elementum_" + leafTask.Type + "_task"
					for _, imp := range g.imports {
						if imp.ResourceType == taskType && strings.HasSuffix(imp.ID, ":"+leafTask.ID) {
							return taskType + "." + imp.ResourceName + ".refs[\"" + refName + "\"]"
						}
					}
				}
			}
		}
	}

	// Check for triggerReference
	if triggerRef, ok := valueRef["triggerReference"].(map[string]interface{}); ok && triggerRef != nil {
		refName := getStringValue(triggerRef, "name")
		if refName != "" {
			automationResourceName := g.getAutomationResourceName(automation.ID)
			if automationResourceName != "" {
				return "elementum_automation." + automationResourceName + ".refs[\"" + refName + "\"]"
			}
		}
	}

	return ""
}

// GenerateAutomationsOnlyIR generates IR blocks for automation resources only.
// Handles automations from the main app and all discovered apps.
func (g *AutomationHCLGenerator) GenerateAutomationsOnlyIR() []*HCLBlock {
	var blocks []*HCLBlock

	// Helper to generate automation blocks for a set of automations with a given app_id ref
	generateForApp := func(automations []discovery.Automation, appRef string) {
		for _, automation := range automations {
			if !automation.HasPublished || automation.Status != "ACTIVE" {
				continue
			}
			automationName := g.getAutomationResourceName(automation.ID)
			if automationName == "" {
				continue
			}

			b := NewResourceBlock("elementum_automation", automationName)
			b.SetAttr("app_id", Ref(appRef))
			b.SetAttr("name", Str(automation.Name))
			blocks = append(blocks, b)
		}
	}

	// Main app automations
	mainAppRef := "elementum_app." + AppResourceName(g.app) + ".id"
	generateForApp(g.app.Automations, mainAppRef)

	// Discovered app automations (each with their own app_id)
	for _, da := range g.app.DiscoveredApps {
		daRef := "elementum_app." + AppResourceName(da) + ".id"
		generateForApp(da.Automations, daRef)
	}

	// Discovered element automations (elements use element resource type)
	for _, de := range g.app.DiscoveredElements {
		deRef := "elementum_element." + SanitizeName(de.Name) + ".id"
		generateForApp(de.Automations, deRef)
	}

	// Discovered task automations (tasks use task resource type)
	for _, dt := range g.app.DiscoveredTasks {
		dtRef := "elementum_task." + SanitizeName(dt.Name) + ".id"
		generateForApp(dt.Automations, dtRef)
	}

	return blocks
}

// BeautifyAutomation applies beautification to an automation resource block in HCL
func (g *AutomationHCLGenerator) BeautifyAutomation(hcl string, automation *discovery.Automation) string {
	// Replace UUID strings with terraform references
	for uuid, ref := range g.uuidMap {
		hcl = strings.ReplaceAll(hcl, fmt.Sprintf("%q", uuid), ref)
	}

	return hcl
}

// AspectAutomationHCLGenerator generates HCL for automation resources on any aspect type (element, task)
// This is a generic version that doesn't require an App structure
type AspectAutomationHCLGenerator struct {
	aspectID    string
	aspectName  string
	aspectType  string // "element" or "task"
	automations []discovery.Automation
	imports     []ImportBlock
	uuidMap     map[string]string

	// Sub-generators for triggers and tasks
	triggerGen *TriggerHCLGenerator
	taskGen    *WorkflowTaskHCLGenerator
}

// NewAspectAutomationHCLGenerator creates a new automation HCL generator for any aspect type
func NewAspectAutomationHCLGenerator(
	aspectID string,
	aspectName string,
	aspectType string,
	automations []discovery.Automation,
	imports []ImportBlock,
	uuidMap map[string]string,
) *AspectAutomationHCLGenerator {
	return &AspectAutomationHCLGenerator{
		aspectID:    aspectID,
		aspectName:  aspectName,
		aspectType:  aspectType,
		automations: automations,
		imports:     imports,
		uuidMap:     uuidMap,
		triggerGen:  NewTriggerHCLGenerator(imports, uuidMap, automations),
		taskGen:     NewWorkflowTaskHCLGenerator(aspectID, aspectType, imports, uuidMap, automations),
	}
}

// GenerateAll generates HCL for all automation resources (automations, triggers, tasks)
func (g *AspectAutomationHCLGenerator) GenerateAll() string {
	return SerializeBlocks(g.GenerateAllIR())
}

// generateAutomationsOnly generates HCL for automation resources without triggers and tasks
func (g *AspectAutomationHCLGenerator) generateAutomationsOnly() string {
	return SerializeBlocks(g.generateAutomationsOnlyIR())
}

// getAutomationResourceName finds the resource name from imports for an automation ID
func (g *AspectAutomationHCLGenerator) getAutomationResourceName(automationID string) string {
	for _, imp := range g.imports {
		if imp.ResourceType == "elementum_automation" && strings.Contains(imp.ID, automationID) {
			return imp.ResourceName
		}
	}
	return ""
}

// GenerateAllIR generates IR blocks for all aspect automation resources
func (g *AspectAutomationHCLGenerator) GenerateAllIR() []*HCLBlock {
	var blocks []*HCLBlock
	blocks = append(blocks, g.generateAutomationsOnlyIR()...)
	blocks = append(blocks, g.triggerGen.GenerateAllIR()...)
	blocks = append(blocks, g.taskGen.GenerateAllIR()...)
	return blocks
}

// generateAutomationsOnlyIR generates IR blocks for automation resources only
func (g *AspectAutomationHCLGenerator) generateAutomationsOnlyIR() []*HCLBlock {
	var blocks []*HCLBlock
	aspectResourceName := SanitizeName(g.aspectName)

	idAttr := "app_id"
	resourceType := "elementum_app"
	switch g.aspectType {
	case "element":
		idAttr = "element_id"
		resourceType = "elementum_element"
	case "task":
		idAttr = "task_id"
		resourceType = "elementum_task"
	}

	for _, automation := range g.automations {
		if !automation.HasPublished || automation.Status != "ACTIVE" {
			continue
		}
		automationName := g.getAutomationResourceName(automation.ID)
		if automationName == "" {
			continue
		}

		b := NewResourceBlock("elementum_automation", automationName)
		b.SetAttr(idAttr, Ref(resourceType+"."+aspectResourceName+".id"))
		b.SetAttr("name", Str(automation.Name))
		blocks = append(blocks, b)
	}
	return blocks
}

// WorkflowTaskHCLGenerator generates HCL for workflow task resources in any aspect
// This is a wrapper around the existing task HCL generation logic for non-App aspects
type WorkflowTaskHCLGenerator struct {
	aspectID    string
	aspectType  string
	imports     []ImportBlock
	uuidMap     map[string]string
	automations []discovery.Automation

	// Reference maps
	parentMap        map[string]string
	taskRefMap       map[string]string
	triggerRefMap    map[string]string
	triggerFieldRefs map[string]map[string]string
	taskFieldRefs    map[string]map[string]string
}

// NewWorkflowTaskHCLGenerator creates a new workflow task HCL generator
func NewWorkflowTaskHCLGenerator(
	aspectID string,
	aspectType string,
	imports []ImportBlock,
	uuidMap map[string]string,
	automations []discovery.Automation,
) *WorkflowTaskHCLGenerator {
	g := &WorkflowTaskHCLGenerator{
		aspectID:         aspectID,
		aspectType:       aspectType,
		imports:          imports,
		uuidMap:          uuidMap,
		automations:      automations,
		parentMap:        make(map[string]string),
		taskRefMap:       make(map[string]string),
		triggerRefMap:    make(map[string]string),
		triggerFieldRefs: make(map[string]map[string]string),
		taskFieldRefs:    make(map[string]map[string]string),
	}

	g.buildReferenceMaps()
	return g
}

// buildReferenceMaps builds mappings for trigger, task, and parent references
func (g *WorkflowTaskHCLGenerator) buildReferenceMaps() {
	for _, automation := range g.automations {
		if !automation.HasPublished || automation.Status != "ACTIVE" {
			continue
		}

		// Build trigger reference map
		var firstTriggerRef string
		for _, trigger := range automation.Triggers {
			if trigger.Type == "unknown" {
				continue
			}
			triggerType := "elementum_" + trigger.Type + "_trigger"
			for _, imp := range g.imports {
				if imp.ResourceType == triggerType && strings.HasSuffix(imp.ID, ":"+trigger.ID) {
					ref := triggerType + "." + imp.ResourceName + ".id"
					g.triggerRefMap[trigger.ID] = ref
					if firstTriggerRef == "" {
						firstTriggerRef = ref
					}
					break
				}
			}
		}

		// Build task reference map (first pass - get all task refs)
		for _, task := range automation.Tasks {
			if task.Type == "unknown" {
				continue
			}
			taskType := "elementum_" + task.Type + "_task"
			for _, imp := range g.imports {
				if imp.ResourceType == taskType && strings.HasSuffix(imp.ID, ":"+task.ID) {
					ref := taskType + "." + imp.ResourceName + ".id"
					g.taskRefMap[task.ID] = ref
					break
				}
			}
		}

		// Build parent map (second pass - resolve parent references)
		for _, task := range automation.Tasks {
			if task.Type == "unknown" {
				continue
			}

			var parentRef string
			if task.ParentID != "" {
				// Check if parent is a trigger
				if ref, ok := g.triggerRefMap[task.ParentID]; ok {
					parentRef = ref
				} else if ref, ok := g.taskRefMap[task.ParentID]; ok {
					// Parent is another task
					parentRef = ref
				}
			}

			// Fallback to first trigger if no parent found
			if parentRef == "" && firstTriggerRef != "" {
				parentRef = firstTriggerRef
			}

			if parentRef != "" {
				g.parentMap[task.ID] = parentRef
			}
		}
	}
}

// GenerateAll generates HCL for all workflow task resources
func (g *WorkflowTaskHCLGenerator) GenerateAll() string {
	return SerializeBlocks(g.GenerateAllIR())
}

// GenerateAllIR generates IR blocks for all workflow task resources
func (g *WorkflowTaskHCLGenerator) GenerateAllIR() []*HCLBlock {
	var blocks []*HCLBlock

	for _, automation := range g.automations {
		if !automation.HasPublished || automation.Status != "ACTIVE" {
			continue
		}

		for _, task := range automation.Tasks {
			if task.Type == "unknown" {
				continue
			}

			taskResourceName := g.getTaskResourceName(task.ID, task.Type)
			if taskResourceName == "" {
				continue
			}

			taskType := "elementum_" + task.Type + "_task"
			b := NewResourceBlock(taskType, taskResourceName)
			b.SetAttr("workflow_id", Str(automation.WorkflowID))
			b.SetAttr("name", Str(task.Name))

			if parentRef, ok := g.parentMap[task.ID]; ok {
				b.SetAttr("previous_task_id", Raw(parentRef))
			}

			blocks = append(blocks, b)
		}
	}
	return blocks
}

// getTaskResourceName finds the resource name from imports for a task ID
func (g *WorkflowTaskHCLGenerator) getTaskResourceName(taskID string, taskType string) string {
	resourceType := "elementum_" + taskType + "_task"
	for _, imp := range g.imports {
		if imp.ResourceType == resourceType && strings.HasSuffix(imp.ID, ":"+taskID) {
			return imp.ResourceName
		}
	}
	return ""
}
