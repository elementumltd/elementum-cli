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

// HCLGenerator generates fully hydrated Terraform HCL for app resources.
// It coordinates specialized generators for each resource type.
type HCLGenerator struct {
	app     *discovery.App
	imports []ImportBlock
	uuidMap map[string]string

	// Maps for reference resolution (maintained for backward compatibility)
	triggerRefMap   map[string]string
	taskRefMap      map[string]string
	parentMap       map[string]string
	automationNames map[string]string

	// Specialized generators
	automationGen   *AutomationHCLGenerator
	agentGen        *AgentHCLGenerator
	fileReaderGen   *FileReaderHCLGenerator
	relationshipGen *RelationshipHCLGenerator
	accessPolicyGen *AccessPolicyHCLGenerator
	searchTableGen  *AISearchTableHCLGenerator
	widgetGen       *WidgetHCLGenerator
}

// NewHCLGenerator creates a new HCL generator for the given app and imports
func NewHCLGenerator(app *discovery.App, imports []ImportBlock) *HCLGenerator {
	// Build UUID map with all mappings including system fields
	uuidMap := buildUUIDMap(imports, app)

	g := &HCLGenerator{
		app:             app,
		imports:         imports,
		uuidMap:         uuidMap,
		triggerRefMap:   make(map[string]string),
		taskRefMap:      make(map[string]string),
		parentMap:       make(map[string]string),
		automationNames: make(map[string]string),
	}

	// Build reference maps first (needed by specialized generators)
	g.buildReferenceMaps()

	// Initialize specialized generators
	g.automationGen = NewAutomationHCLGenerator(app, imports, uuidMap)
	g.agentGen = NewAgentHCLGenerator(app, imports, uuidMap)
	g.fileReaderGen = NewFileReaderHCLGenerator(app, imports, uuidMap)
	g.relationshipGen = NewRelationshipHCLGenerator(app, imports, uuidMap)
	g.accessPolicyGen = NewAccessPolicyHCLGenerator(app, imports, uuidMap)
	g.searchTableGen = NewAISearchTableHCLGenerator(app, imports, uuidMap)
	g.widgetGen = NewWidgetHCLGenerator(app, imports, uuidMap)

	return g
}

// AspectElementHCLGenerator generates HCL for element resources including automations
type AspectElementHCLGenerator struct {
	element       *discovery.Element
	imports       []ImportBlock
	uuidMap       map[string]string
	automationGen *AspectAutomationHCLGenerator
}

// NewElementHCLGenerator creates a new HCL generator for the given element and imports
func NewElementHCLGenerator(element *discovery.Element, imports []ImportBlock) *AspectElementHCLGenerator {
	// Build UUID map for elements
	uuidMap := buildElementUUIDMap(imports, element)

	g := &AspectElementHCLGenerator{
		element: element,
		imports: imports,
		uuidMap: uuidMap,
	}

	// Initialize automation generator for elements
	g.automationGen = NewAspectAutomationHCLGenerator(
		element.ID,
		element.Name,
		"element",
		element.Automations,
		imports,
		uuidMap,
	)

	return g
}

// GenerateAutomationHCL generates HCL for all automation resources
func (g *AspectElementHCLGenerator) GenerateAutomationHCL() string {
	return SerializeBlocks(g.automationGen.GenerateAllIR())
}

// AspectTaskHCLGenerator generates HCL for AspectTask resources including automations
// Note: This is different from TaskHCLGenerator in hcl_task.go which generates workflow task HCL
type AspectTaskHCLGenerator struct {
	task          *discovery.AspectTask
	imports       []ImportBlock
	uuidMap       map[string]string
	automationGen *AspectAutomationHCLGenerator
}

// NewAspectTaskHCLGenerator creates a new HCL generator for the given AspectTask and imports
func NewAspectTaskHCLGenerator(task *discovery.AspectTask, imports []ImportBlock) *AspectTaskHCLGenerator {
	// Build UUID map for tasks
	uuidMap := buildTaskUUIDMap(imports, task)

	g := &AspectTaskHCLGenerator{
		task:    task,
		imports: imports,
		uuidMap: uuidMap,
	}

	// Initialize automation generator for tasks
	g.automationGen = NewAspectAutomationHCLGenerator(
		task.ID,
		task.Name,
		"task",
		task.Automations,
		imports,
		uuidMap,
	)

	return g
}

// GenerateAutomationHCL generates HCL for all automation resources
func (g *AspectTaskHCLGenerator) GenerateAutomationHCL() string {
	return SerializeBlocks(g.automationGen.GenerateAllIR())
}

// buildAspectUUIDMap builds a UUID map for a non-app aspect (element or task).
// aspectID is the object's UUID, aspectType is the terraform resource type prefix
// (e.g., "elementum_element" or "elementum_task"), and fields are the aspect's fields.
func buildAspectUUIDMap(imports []ImportBlock, aspectID, aspectName, aspectType string, fields []discovery.Field) map[string]string {
	uuidMap := make(map[string]string)

	// Add the aspect itself
	sanitizedName := SanitizeName(aspectName)
	uuidMap[aspectID] = aspectType + "." + sanitizedName + ".id"

	// Add fields
	for _, field := range fields {
		fieldName := SanitizeName(field.Name)
		fieldType := "elementum_" + field.Type + "_field"
		uuidMap[field.ID] = fieldType + "." + fieldName + ".id"
	}

	// Add imports
	for _, imp := range imports {
		if strings.Contains(imp.ID, ":") {
			// Complex ID like "workflow_id:task_id:type"
			parts := strings.Split(imp.ID, ":")
			if len(parts) >= 2 {
				uuidMap[parts[len(parts)-2]] = imp.ResourceType + "." + imp.ResourceName + ".id"
			}
		} else {
			uuidMap[imp.ID] = imp.ResourceType + "." + imp.ResourceName + ".id"
		}
	}

	return uuidMap
}

// buildElementUUIDMap builds a UUID map for element imports
func buildElementUUIDMap(imports []ImportBlock, element *discovery.Element) map[string]string {
	return buildAspectUUIDMap(imports, element.ID, element.Name, "elementum_element", element.Fields)
}

// buildTaskUUIDMap builds a UUID map for task imports
func buildTaskUUIDMap(imports []ImportBlock, task *discovery.AspectTask) map[string]string {
	return buildAspectUUIDMap(imports, task.ID, task.Name, "elementum_task", task.Fields)
}

// buildReferenceMapsFromAutomations is the shared implementation for building trigger, task,
// and parent reference maps from a set of automations. The optional uuidMap is used as a
// fallback for parent resolution (e.g., switch case chains in TaskHCLGenerator).
func buildReferenceMapsFromAutomations(
	automations []discovery.Automation,
	imports []ImportBlock,
	triggerRefMap map[string]string,
	taskRefMap map[string]string,
	parentMap map[string]string,
	uuidMap map[string]string, // optional fallback for parent resolution; nil OK
) {
	for _, automation := range automations {
		// Skip unpublished or inactive automations
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
			for _, imp := range imports {
				if imp.ResourceType == triggerType && strings.HasSuffix(imp.ID, ":"+trigger.ID) {
					ref := triggerType + "." + imp.ResourceName + ".id"
					triggerRefMap[trigger.ID] = ref
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
			for _, imp := range imports {
				if imp.ResourceType == taskType && strings.HasSuffix(imp.ID, ":"+task.ID) {
					ref := taskType + "." + imp.ResourceName + ".id"
					taskRefMap[task.ID] = ref
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
				if ref, ok := triggerRefMap[task.ParentID]; ok {
					parentRef = ref
				} else if ref, ok := taskRefMap[task.ParentID]; ok {
					// Parent is another task
					parentRef = ref
				} else if uuidMap != nil {
					// Fallback: parent might be a switch case chain (referenced in uuidMap)
					if ref, ok := uuidMap[task.ParentID]; ok {
						parentRef = ref
					}
				}
			}

			// Fallback to first trigger if no parent found
			if parentRef == "" && firstTriggerRef != "" {
				parentRef = firstTriggerRef
			}

			if parentRef != "" {
				parentMap[task.ID] = parentRef
			}
		}
	}
}

// buildReferenceMaps builds mappings for trigger, task, and parent references
func (g *HCLGenerator) buildReferenceMaps() {
	// Build automation name map (HCLGenerator-specific)
	for _, automation := range g.app.AllAutomations() {
		if !automation.HasPublished || automation.Status != "ACTIVE" {
			continue
		}
		for _, imp := range g.imports {
			if imp.ResourceType == "elementum_automation" && strings.Contains(imp.ID, automation.ID) {
				g.automationNames[automation.ID] = imp.ResourceName
				break
			}
		}
	}

	// Delegate the shared logic
	buildReferenceMapsFromAutomations(
		g.app.AllAutomations(), g.imports,
		g.triggerRefMap, g.taskRefMap, g.parentMap,
		nil, // HCLGenerator doesn't use uuidMap fallback
	)
}

// GenerateAutomationResourcesOnly generates HCL for automation resources without triggers and tasks
func (g *HCLGenerator) GenerateAutomationResourcesOnly() string {
	return SerializeBlocks(g.automationGen.GenerateAutomationsOnlyIR())
}

// GenerateTriggerResourcesOnly generates HCL for all trigger resources
func (g *HCLGenerator) GenerateTriggerResourcesOnly() string {
	return SerializeBlocks(g.automationGen.triggerGen.GenerateAllIR())
}

// GenerateTaskResourcesOnly generates HCL for all task resources with proper parent_id resolution
func (g *HCLGenerator) GenerateTaskResourcesOnly() string {
	return SerializeBlocks(g.automationGen.taskGen.GenerateAllIR())
}

// GetParentRef returns the resolved parent reference for a task ID
func (g *HCLGenerator) GetParentRef(taskID string) string {
	return g.parentMap[taskID]
}

// GetTaskRef returns the terraform resource reference for a task ID
func (g *HCLGenerator) GetTaskRef(taskID string) string {
	return g.taskRefMap[taskID]
}

// GetTriggerRef returns the terraform resource reference for a trigger ID
func (g *HCLGenerator) GetTriggerRef(triggerID string) string {
	return g.triggerRefMap[triggerID]
}

// ============================================================================
// Shared Utility Functions
// ============================================================================

// extractValueFromPath extracts a value from a map using a dot-separated path
func extractValueFromPath(data map[string]interface{}, path string) interface{} {
	parts := strings.Split(path, ".")
	var current interface{} = data

	for _, part := range parts {
		// Handle array access like "variables[0]"
		if idx := strings.Index(part, "["); idx != -1 {
			key := part[:idx]
			m, ok := current.(map[string]interface{})
			if !ok {
				return nil
			}
			arr, ok := m[key].([]interface{})
			if !ok || len(arr) == 0 {
				return nil
			}
			current = arr[0]
			continue
		}

		m, ok := current.(map[string]interface{})
		if !ok {
			return nil
		}
		current = m[part]
		if current == nil {
			return nil
		}
	}

	return current
}

// decodeValueReference decodes a GraphQL value reference to a terraform-friendly string
// Value references have nested structures: triggerReference, taskReference, templateReference
func decodeValueReference(ref map[string]interface{}) string {
	// Check for direct triggerReference
	if triggerRef, ok := ref["triggerReference"].(map[string]interface{}); ok && triggerRef != nil {
		if name, ok := triggerRef["name"].(string); ok && name != "" {
			return fmt.Sprintf(`{"triggerReference":{"name":"%s"}}`, name)
		}
	}

	// Check for direct taskReference
	if taskRef, ok := ref["taskReference"].(map[string]interface{}); ok && taskRef != nil {
		if name, ok := taskRef["name"].(string); ok && name != "" {
			return fmt.Sprintf(`{"taskReference":{"name":"%s"}}`, name)
		}
	}

	// Check for nested calculationReference (used by calculation tasks)
	if calcRef, ok := ref["calculationReference"].(map[string]interface{}); ok && calcRef != nil {
		if calculation, ok := calcRef["calculation"].(string); ok && calculation != "" {
			return calculation
		}
	}

	// Check for templateReference with parameters
	if templateRef, ok := ref["templateReference"].(map[string]interface{}); ok && templateRef != nil {
		template, _ := templateRef["template"].(string)

		// Try to extract the reference from parameters
		if params, ok := templateRef["parameters"].([]interface{}); ok && len(params) > 0 {
			// For simple templates like {{{p1}}}, extract the first parameter's reference
			if len(params) == 1 && template == "{{{p1}}}" {
				if param, ok := params[0].(map[string]interface{}); ok {
					if value, ok := param["value"].(map[string]interface{}); ok {
						// Check for triggerReference in parameter
						if triggerRef, ok := value["triggerReference"].(map[string]interface{}); ok && triggerRef != nil {
							if name, ok := triggerRef["name"].(string); ok && name != "" {
								return fmt.Sprintf(`{"triggerReference":{"name":"%s"}}`, name)
							}
						}
						// Check for taskReference in parameter
						if taskRef, ok := value["taskReference"].(map[string]interface{}); ok && taskRef != nil {
							if name, ok := taskRef["name"].(string); ok && name != "" {
								return fmt.Sprintf(`{"taskReference":{"name":"%s"}}`, name)
							}
						}
					}
				}
			}
			// For complex templates with multiple parameters or custom template strings,
			// return the template with embedded refs
			return buildTemplateWithParams(template, params)
		}

		// No parameters, just return template
		if template != "" {
			return template
		}
	}

	// Legacy fallback: check for direct value field
	if value, ok := ref["value"]; ok {
		if s, ok := value.(string); ok {
			return s
		}
	}

	// Fallback to label
	if label, ok := ref["label"].(string); ok {
		return label
	}

	return ""
}

// buildTemplateWithParams replaces template placeholders with their parameter values
func buildTemplateWithParams(template string, params []interface{}) string {
	result := template
	for _, paramInterface := range params {
		param, ok := paramInterface.(map[string]interface{})
		if !ok {
			continue
		}

		name, _ := param["name"].(string)
		if name == "" {
			continue
		}

		value, ok := param["value"].(map[string]interface{})
		if !ok {
			continue
		}

		// Build the replacement string
		var refStr string
		if triggerRef, ok := value["triggerReference"].(map[string]interface{}); ok && triggerRef != nil {
			if refName, ok := triggerRef["name"].(string); ok && refName != "" {
				refStr = fmt.Sprintf(`{"triggerReference":{"name":"%s"}}`, refName)
			}
		} else if taskRef, ok := value["taskReference"].(map[string]interface{}); ok && taskRef != nil {
			if refName, ok := taskRef["name"].(string); ok && refName != "" {
				refStr = fmt.Sprintf(`{"taskReference":{"name":"%s"}}`, refName)
			}
		}

		if refStr != "" {
			// Replace {{{name}}} with the reference
			placeholder := fmt.Sprintf("{{{%s}}}", name)
			result = strings.ReplaceAll(result, placeholder, refStr)
		}
	}
	return result
}

// getStringValue safely extracts a string from a map
func getStringValue(m map[string]interface{}, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

// InjectHydratedAppConfigs injects status_options and lock_stages into elementum_app resource blocks.
// This is needed because terraform's import/generate-config only produces minimal configs
// (just namespace) since status_options are optional but important for complete config.
// Handles the main app and all discovered apps uniformly.
func InjectHydratedAppConfigs(hcl string, app *discovery.App, imports []ImportBlock) string {
	if app == nil {
		return hcl
	}

	// Inject for the main app
	hcl = injectAppConfigs(hcl, app, imports)

	// Inject for all discovered apps (same treatment)
	for _, discoveredApp := range app.DiscoveredApps {
		hcl = injectAppConfigs(hcl, discoveredApp, imports)
	}

	return hcl
}

// injectAppConfigs injects status_options and lock_stages into a single elementum_app resource block.
func injectAppConfigs(hcl string, app *discovery.App, imports []ImportBlock) string {
	hcl = injectStatusOptionsForApp(hcl, app, imports)
	hcl = injectLockStagesForApp(hcl, app, imports)
	return hcl
}

// injectStatusOptionsForApp injects status_options into a single elementum_app resource block.
func injectStatusOptionsForApp(hcl string, app *discovery.App, imports []ImportBlock) string {
	if app == nil {
		return hcl
	}

	// Find the status field and its options
	var statusOptions []discovery.FieldOption
	for _, field := range app.Fields {
		for _, tag := range field.SemanticTags {
			if tag == "status" {
				statusOptions = field.Options
				break
			}
		}
		if statusOptions != nil {
			break
		}
	}

	// No status field or no options, nothing to inject
	if len(statusOptions) == 0 {
		return hcl
	}

	// Find the app resource name from imports
	var resourceName string
	for _, imp := range imports {
		if imp.ResourceType == "elementum_app" && (imp.ID == app.ID || strings.HasSuffix(imp.ID, ":"+app.ID)) {
			resourceName = imp.ResourceName
			break
		}
	}
	if resourceName == "" {
		return hcl
	}

	// Find the resource block
	resourceStart := fmt.Sprintf(`resource "elementum_app" "%s" {`, resourceName)
	startIdx := strings.Index(hcl, resourceStart)
	if startIdx == -1 {
		return hcl
	}

	// Find the end of the resource block
	endIdx := startIdx + len(resourceStart)
	braceCount := 1
	for endIdx < len(hcl) && braceCount > 0 {
		if hcl[endIdx] == '{' {
			braceCount++
		} else if hcl[endIdx] == '}' {
			braceCount--
		}
		endIdx++
	}

	if braceCount != 0 {
		return hcl
	}

	// Extract the resource block content
	resourceBlock := hcl[startIdx:endIdx]

	// Check if status_options already exists
	if strings.Contains(resourceBlock, "status_options") {
		return hcl
	}

	// Generate status_options HCL
	statusOptionsHCL := generateStatusOptionsHCL(statusOptions)

	// Find where to insert (before the closing brace)
	insertPos := endIdx - 1
	for insertPos > startIdx && hcl[insertPos-1] != '\n' {
		insertPos--
	}

	// Insert the status_options block
	hcl = hcl[:insertPos] + statusOptionsHCL + hcl[insertPos:]

	return hcl
}

// generateStatusOptionsHCL generates the HCL for status_options attribute
func generateStatusOptionsHCL(options []discovery.FieldOption) string {
	var sb strings.Builder
	sb.WriteString("  status_options = [\n")

	for i, opt := range options {
		sb.WriteString("    {\n")
		sb.WriteString(fmt.Sprintf("      label = %q\n", opt.Label))
		if opt.Color != "" {
			sb.WriteString(fmt.Sprintf("      color = %q\n", opt.Color))
		}
		// Format tags
		if len(opt.Tags) == 0 {
			sb.WriteString("      tags = []\n")
		} else {
			tagStrings := make([]string, len(opt.Tags))
			for j, tag := range opt.Tags {
				tagStrings[j] = fmt.Sprintf("%q", tag)
			}
			sb.WriteString(fmt.Sprintf("      tags = [%s]\n", strings.Join(tagStrings, ", ")))
		}
		if i < len(options)-1 {
			sb.WriteString("    },\n")
		} else {
			sb.WriteString("    }\n")
		}
	}

	sb.WriteString("  ]\n")
	return sb.String()
}

// injectLockStagesForApp injects lock_stages = true into a single elementum_app resource block.
// Only injects if lock_stages is true (we don't need to inject false, that's the default).
func injectLockStagesForApp(hcl string, app *discovery.App, imports []ImportBlock) string {
	if app == nil || !app.LockStages {
		return hcl
	}

	// Find the app resource name from imports
	var resourceName string
	for _, imp := range imports {
		if imp.ResourceType == "elementum_app" && (imp.ID == app.ID || strings.HasSuffix(imp.ID, ":"+app.ID)) {
			resourceName = imp.ResourceName
			break
		}
	}
	if resourceName == "" {
		return hcl
	}

	// Find the resource block
	resourceStart := fmt.Sprintf(`resource "elementum_app" "%s" {`, resourceName)
	startIdx := strings.Index(hcl, resourceStart)
	if startIdx == -1 {
		return hcl
	}

	// Find the end of the resource block
	endIdx := startIdx + len(resourceStart)
	braceCount := 1
	for endIdx < len(hcl) && braceCount > 0 {
		if hcl[endIdx] == '{' {
			braceCount++
		} else if hcl[endIdx] == '}' {
			braceCount--
		}
		endIdx++
	}

	if braceCount != 0 {
		return hcl
	}

	// Extract the resource block content
	resourceBlock := hcl[startIdx:endIdx]

	// Check if lock_stages already exists
	if strings.Contains(resourceBlock, "lock_stages") {
		return hcl
	}

	// Generate lock_stages HCL
	lockStagesHCL := "  lock_stages = true\n"

	// Find where to insert (before the closing brace)
	insertPos := endIdx - 1
	for insertPos > startIdx && hcl[insertPos-1] != '\n' {
		insertPos--
	}

	// Insert the lock_stages attribute
	hcl = hcl[:insertPos] + lockStagesHCL + hcl[insertPos:]

	return hcl
}
