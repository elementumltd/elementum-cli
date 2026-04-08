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
	"encoding/json"
	"fmt"
	"strings"

	"github.com/elementumltd/elementum-cli/discovery"
	"github.com/elementumltd/elementum-cli/logger"
	"github.com/elementumltd/elementum-cli/internal/client"
)

// TaskHCLGenerator generates HCL for task resources
type TaskHCLGenerator struct {
	app         *discovery.App
	imports     []ImportBlock
	uuidMap     map[string]string
	automations []discovery.Automation
	// Maps task ID to its parent terraform reference
	parentMap map[string]string
	// Maps task ID to its resource reference (with .id suffix)
	taskRefMap map[string]string
	// Maps trigger ID to its resource reference (with .id suffix)
	triggerRefMap map[string]string

	// Maps for value reference resolution (refs syntax)
	// Key: trigger/task ID, Value: its FieldRefs map
	triggerFieldRefs map[string]map[string]string
	taskFieldRefs    map[string]map[string]string
	// Maps trigger/task ID to terraform resource reference (without .id suffix)
	triggerResourceRef map[string]string // e.g., "elementum_record_created_trigger.my_trigger"
	taskResourceRef    map[string]string // e.g., "elementum_variable_task.my_task"
	// Maps task ID to its automation's first trigger ID
	taskToTrigger map[string]string
	// Maps variable name to the task ID that defines it (for lookupVariableRef)
	variableDefiningTask map[string]string
}

// NewTaskHCLGenerator creates a new task HCL generator
func NewTaskHCLGenerator(app *discovery.App, imports []ImportBlock, uuidMap map[string]string, automations []discovery.Automation) *TaskHCLGenerator {
	g := &TaskHCLGenerator{
		app:                  app,
		imports:              imports,
		uuidMap:              uuidMap,
		automations:          automations,
		parentMap:            make(map[string]string),
		taskRefMap:           make(map[string]string),
		triggerRefMap:        make(map[string]string),
		triggerFieldRefs:     make(map[string]map[string]string),
		taskFieldRefs:        make(map[string]map[string]string),
		triggerResourceRef:   make(map[string]string),
		taskResourceRef:      make(map[string]string),
		taskToTrigger:        make(map[string]string),
		variableDefiningTask: make(map[string]string),
	}
	g.buildReferenceMaps()
	g.buildFieldRefMaps()
	return g
}

// buildReferenceMaps builds mappings for trigger, task, and parent references
func (g *TaskHCLGenerator) buildReferenceMaps() {
	buildReferenceMapsFromAutomations(
		g.automations, g.imports,
		g.triggerRefMap, g.taskRefMap, g.parentMap,
		g.uuidMap, // TaskHCLGenerator uses uuidMap as fallback for switch case chains
	)
}

// buildFieldRefMaps populates the FieldRefs maps from automations for refs syntax conversion
func (g *TaskHCLGenerator) buildFieldRefMaps() {
	for _, automation := range g.automations {
		// Skip unpublished or inactive automations
		if !automation.HasPublished || automation.Status != "ACTIVE" {
			continue
		}

		// Find first trigger ID for this automation
		var firstTriggerID string
		for _, trigger := range automation.Triggers {
			if trigger.Type == "unknown" {
				continue
			}
			firstTriggerID = trigger.ID

			// Store trigger's FieldRefs
			if trigger.FieldRefs != nil {
				g.triggerFieldRefs[trigger.ID] = trigger.FieldRefs
				// Debug: log trigger FieldRefs
				logger.Debug("Trigger %s FieldRefs (%d entries):", trigger.ID[:8], len(trigger.FieldRefs))
				for k, v := range trigger.FieldRefs {
					logger.Debug("  %q -> %q", k, v)
				}
			}

			// Build resource ref (without .id suffix)
			if ref, ok := g.triggerRefMap[trigger.ID]; ok {
				g.triggerResourceRef[trigger.ID] = strings.TrimSuffix(ref, ".id")
			}

			break // Use first trigger
		}

		// Map all tasks in this automation to the first trigger
		for _, task := range automation.Tasks {
			if task.Type == "unknown" {
				continue
			}

			// Map task to its automation's first trigger
			if firstTriggerID != "" {
				g.taskToTrigger[task.ID] = firstTriggerID
			}

			// Store task's FieldRefs
			if len(task.FieldRefs) > 0 {
				g.taskFieldRefs[task.ID] = task.FieldRefs
				// Debug: log task FieldRefs
				logger.Debug("Task %s (%s) FieldRefs (%d entries):", task.ID[:8], task.Name, len(task.FieldRefs))
				for k, v := range task.FieldRefs {
					logger.Debug("  %q -> %q", k, v)
				}
			}

			// Build resource ref (without .id suffix)
			if ref, ok := g.taskRefMap[task.ID]; ok {
				g.taskResourceRef[task.ID] = strings.TrimSuffix(ref, ".id")
			}

			// For variable_task, extract the variable name it defines
			// This is used by lookupVariableRef to find the correct defining task
			if task.Type == "variable" && task.RawData != nil {
				if varName := extractVariableNameFromTask(task.RawData); varName != "" {
					g.variableDefiningTask[varName] = task.ID
					taskIDShort := task.ID
					if len(taskIDShort) > 8 {
						taskIDShort = taskIDShort[:8]
					}
					logger.Debug("Task %s defines variable %q", taskIDShort, varName)
				}
			}
		}
	}
}

// extractVariableNameFromTask extracts the variable name from a variable_task's RawData.
// Variable tasks have a "variables" array with WorkflowVariableTaskParameterCreate entries.
func extractVariableNameFromTask(rawData map[string]interface{}) string {
	variables, ok := rawData["variables"].([]interface{})
	if !ok || len(variables) == 0 {
		return ""
	}

	firstVar, ok := variables[0].(map[string]interface{})
	if !ok {
		return ""
	}

	// Only extract from CREATE operations, not UPDATE
	typename := getStringValue(firstVar, "__typename")
	if typename != "WorkflowVariableTaskParameterCreate" {
		return ""
	}

	// Get the variable name
	return getStringValue(firstVar, "name")
}

// GenerateAll generates HCL for all tasks in all automations
func (g *TaskHCLGenerator) GenerateAll() string {
	return SerializeBlocks(g.GenerateAllIR())
}

// convertToRefsSyntax converts a value reference to terraform refs syntax
func (g *TaskHCLGenerator) convertToRefsSyntax(task *discovery.Task, ref map[string]interface{}) string {
	// Case -1: Handle wrapped value references like {id, label, value: "{\"variableReference\":{\"name\":\"myVar\"}}"}
	// This is common for update_variable_task's variable_reference field
	if valueStr, ok := ref["value"].(string); ok && valueStr != "" && strings.HasPrefix(valueStr, "{") {
		// Try to parse the value as JSON to extract the actual reference
		var parsedValue map[string]interface{}
		if err := json.Unmarshal([]byte(valueStr), &parsedValue); err == nil {
			// Recursively convert the parsed value
			if result := g.convertToRefsSyntax(task, parsedValue); result != "" {
				return result
			}
		}
	}

	// Case 0: Value with just {id, valid} - lookup by ID
	// This handles cases where the task data only has the reference ID without the full
	// triggerReference/taskReference structure. The availableReferences API provides the mapping.
	if refID, ok := ref["id"].(string); ok && refID != "" {
		// Only use ID lookup if there's no other reference type present
		_, hasTriggerRef := ref["triggerReference"].(map[string]interface{})
		_, hasTaskRef := ref["taskReference"].(map[string]interface{})
		_, hasForEachRef := ref["forEachReference"].(map[string]interface{})
		_, hasVarRef := ref["variableReference"].(map[string]interface{})
		_, hasTemplateRef := ref["templateReference"].(map[string]interface{})

		if !hasTriggerRef && !hasTaskRef && !hasForEachRef && !hasVarRef && !hasTemplateRef {
			if result := g.lookupByRefID(task, refID); result != "" {
				return result
			}
		}
	}

	// Case 1: Direct triggerReference
	if triggerRef, ok := ref["triggerReference"].(map[string]interface{}); ok && triggerRef != nil {
		if name, ok := triggerRef["name"].(string); ok && name != "" {
			result := g.lookupTriggerRef(task, name)
			if result == "" {
				fmt.Printf("[DEBUG] lookupTriggerRef failed for task=%s, path=%s\n", task.Name, name)
			}
			return result
		}
	}

	// Case 2: Direct taskReference
	if taskRef, ok := ref["taskReference"].(map[string]interface{}); ok && taskRef != nil {
		if name, ok := taskRef["name"].(string); ok && name != "" {
			// Try with task ID - this should always be available from GraphQL
			taskID := g.extractTaskIDFromRef(taskRef)
			if taskID != "" {
				if result := g.lookupTaskRef(taskID, name); result != "" {
					return result
				}
			}
			// If no task ID, log a warning. The caller will fall back to raw JSON output.
			// We do NOT use lookupTaskRefByName because it can match the wrong task
			// (e.g., a task that comes LATER in the workflow), which creates cycles.
			fmt.Printf("[WARN] Could not resolve taskReference: task ID not available for ref name=%s, task=%s\n", name, task.Name)
		}
	}

	// Case 3: templateReference with parameters
	if templateRef, ok := ref["templateReference"].(map[string]interface{}); ok && templateRef != nil {
		return g.handleTemplateRef(task, templateRef)
	}

	// Case 4: forEachReference
	if forEachRef, ok := ref["forEachReference"].(map[string]interface{}); ok && forEachRef != nil {
		if name, ok := forEachRef["name"].(string); ok && name != "" {
			// Extract forEach task ID
			forEachTaskID := ""
			if forEachTask, ok := forEachRef["forEachTask"].(map[string]interface{}); ok {
				if id, ok := forEachTask["id"].(string); ok {
					forEachTaskID = id
				}
			}
			if forEachTaskID != "" {
				if result := g.lookupTaskRef(forEachTaskID, name); result != "" {
					return result
				}
			}
			fmt.Printf("[WARN] Could not resolve forEachReference: name=%s, task=%s\n", name, task.Name)
		}
	}

	// Case 5: variableReference
	if varRef, ok := ref["variableReference"].(map[string]interface{}); ok && varRef != nil {
		if name, ok := varRef["name"].(string); ok && name != "" {
			// Look up by variable name across all tasks
			if result := g.lookupVariableRef(name); result != "" {
				return result
			}
			fmt.Printf("[WARN] Could not resolve variableReference: name=%s, task=%s\n", name, task.Name)
		}
	}

	return ""
}

// lookupTriggerRef resolves a trigger reference path to refs syntax
func (g *TaskHCLGenerator) lookupTriggerRef(task *discovery.Task, internalPath string) string {
	// Find the trigger for this task's automation
	triggerID := g.taskToTrigger[task.ID]
	if triggerID == "" {
		return ""
	}

	// Get the FieldRefs for this trigger
	fieldRefs, ok := g.triggerFieldRefs[triggerID]
	if !ok || fieldRefs == nil {
		return ""
	}

	// Look up human-readable name
	fieldName := g.lookupFieldName(fieldRefs, internalPath)
	if fieldName == "" {
		return ""
	}

	// Get terraform resource reference
	resourceRef, ok := g.triggerResourceRef[triggerID]
	if !ok || resourceRef == "" {
		return ""
	}

	// Generate refs syntax (unquoted terraform expression)
	return fmt.Sprintf(`%s.refs["%s"]`, resourceRef, fieldName)
}

// lookupTaskRef resolves a task reference to refs syntax
func (g *TaskHCLGenerator) lookupTaskRef(taskID, propPath string) string {
	if taskID == "" {
		return ""
	}

	// Get the FieldRefs for this task
	fieldRefs, ok := g.taskFieldRefs[taskID]
	if !ok || fieldRefs == nil {
		return ""
	}

	// Look up human-readable name
	propName, ok := fieldRefs[propPath]
	if !ok || propName == "" {
		return ""
	}

	// Get terraform resource reference
	resourceRef, ok := g.taskResourceRef[taskID]
	if !ok || resourceRef == "" {
		return ""
	}

	// Generate refs syntax (unquoted terraform expression)
	return fmt.Sprintf(`%s.refs["%s"]`, resourceRef, propName)
}

// lookupVariableRef finds a variable reference by name across all tasks.
// It first checks variableDefiningTask to find the task that actually defines
// the variable (for variable_task), then falls back to searching all FieldRefs.
func (g *TaskHCLGenerator) lookupVariableRef(varName string) string {
	// First, check if we know which task defines this variable
	// This avoids the issue where multiple tasks have the same variable in their FieldRefs
	if definingTaskID, ok := g.variableDefiningTask[varName]; ok {
		if fieldRefs, ok := g.taskFieldRefs[definingTaskID]; ok {
			key := "variable." + varName
			if propName, ok := fieldRefs[key]; ok {
				if resourceRef, ok := g.taskResourceRef[definingTaskID]; ok {
					return fmt.Sprintf(`%s.refs["%s"]`, resourceRef, propName)
				}
			}
		}
	}

	// Fallback: Search all tasks for a variable with this name
	// This is needed for variables that might be defined in other ways (e.g., inside switch/foreach)
	for taskID, fieldRefs := range g.taskFieldRefs {
		key := "variable." + varName
		if propName, ok := fieldRefs[key]; ok {
			if resourceRef, ok := g.taskResourceRef[taskID]; ok {
				return fmt.Sprintf(`%s.refs["%s"]`, resourceRef, propName)
			}
		}
	}
	return ""
}

// lookupByRefID finds a reference by its ID in trigger and task FieldRefs.
// This is used when the task data has only {id, valid} without triggerReference/taskReference.
func (g *TaskHCLGenerator) lookupByRefID(task *discovery.Task, refID string) string {
	idKey := "id:" + refID

	// First try trigger refs for this task's automation
	triggerID := g.taskToTrigger[task.ID]
	if triggerID != "" {
		if fieldRefs, ok := g.triggerFieldRefs[triggerID]; ok {
			if propName, ok := fieldRefs[idKey]; ok {
				if resourceRef, ok := g.triggerResourceRef[triggerID]; ok {
					return fmt.Sprintf(`%s.refs["%s"]`, resourceRef, propName)
				}
			}
		}
	}

	// Then try task refs across all tasks
	for taskID, fieldRefs := range g.taskFieldRefs {
		if propName, ok := fieldRefs[idKey]; ok {
			if resourceRef, ok := g.taskResourceRef[taskID]; ok {
				return fmt.Sprintf(`%s.refs["%s"]`, resourceRef, propName)
			}
		}
	}

	return ""
}

// lookupFieldName finds the human-readable name for an internal reference path
func (g *TaskHCLGenerator) lookupFieldName(fieldRefs map[string]string, internalPath string) string {
	// Try exact match first
	if name, ok := fieldRefs[internalPath]; ok {
		return name
	}

	// Try without "record." prefix
	if strings.HasPrefix(internalPath, "record.") {
		path := strings.TrimPrefix(internalPath, "record.")
		if name, ok := fieldRefs[path]; ok {
			return name
		}
		// Try just the UUID portion (for nested paths like "record.{uuid}.email")
		parts := strings.SplitN(path, ".", 2)
		if name, ok := fieldRefs[parts[0]]; ok {
			// If there's a suffix (like .id or .email), append it
			if len(parts) > 1 {
				return name + "." + parts[1]
			}
			return name
		}
	}

	// Try with "record." prefix added
	if name, ok := fieldRefs["record."+internalPath]; ok {
		return name
	}

	return ""
}

// handleTemplateRef handles templateReference with parameters
func (g *TaskHCLGenerator) handleTemplateRef(task *discovery.Task, templateRef map[string]interface{}) string {
	template, _ := templateRef["template"].(string)
	params, ok := templateRef["parameters"].([]interface{})
	if !ok || len(params) == 0 {
		return ""
	}

	// Simple case: single parameter with {{{p1}}}
	if len(params) == 1 && template == "{{{p1}}}" {
		if param, ok := params[0].(map[string]interface{}); ok {
			if value, ok := param["value"].(map[string]interface{}); ok {
				// Recursively resolve the parameter's reference
				return g.convertToRefsSyntax(task, value)
			}
		}
		return ""
	}

	// Complex case: multiple parameters - build interpolated string
	result := template
	hasReplacements := false
	for _, p := range params {
		param, ok := p.(map[string]interface{})
		if !ok {
			continue
		}
		paramName, _ := param["name"].(string)
		if paramName == "" {
			continue
		}
		paramValue, ok := param["value"].(map[string]interface{})
		if !ok {
			continue
		}

		placeholder := "{{{" + paramName + "}}}"
		refExpr := g.convertToRefsSyntax(task, paramValue)
		if refExpr != "" {
			result = strings.Replace(result, placeholder, "${"+refExpr+"}", 1)
			hasReplacements = true
		}
	}

	// Only return if we made replacements
	if hasReplacements {
		// Use FormatHCLStringWithInterpolations which properly handles:
		// - Multi-line strings (uses heredoc syntax)
		// - Single-line strings with proper escaping
		// - Preserves ${...} interpolation expressions
		return FormatHCLStringWithInterpolations(result)
	}
	return ""
}

// extractTaskIDFromRef extracts the task ID from a taskReference
func (g *TaskHCLGenerator) extractTaskIDFromRef(taskRef map[string]interface{}) string {
	// Check for task.id directly
	if task, ok := taskRef["task"].(map[string]interface{}); ok {
		if id, ok := task["id"].(string); ok {
			return id
		}
	}
	return ""
}

// resolveObjectRef resolves an object ID (app/element) to a terraform reference
func (g *TaskHCLGenerator) resolveObjectRef(objectID string) string {
	// Check if it's the same app
	if g.app != nil && objectID == g.app.ID {
		return "elementum_app." + AppResourceName(g.app) + ".id"
	}

	// Check UUID map
	if ref, ok := g.uuidMap[objectID]; ok {
		return ref
	}

	// Check related objects
	if g.app != nil {
		for _, related := range g.app.RelatedObjects {
			if related.ID == objectID {
				dataType := "data.elementum_app"
				if related.Type == "Element" {
					dataType = "data.elementum_element"
				}
				return dataType + "." + SanitizeName(related.Name) + ".id"
			}
		}
	}

	// Return the raw ID as a string if we can't resolve it
	return fmt.Sprintf("%q", objectID)
}

// resolveRef resolves an ID to a terraform reference based on the reference type
func (g *TaskHCLGenerator) resolveRef(id string, refType string) string {
	// First check our UUID map
	if ref, ok := g.uuidMap[id]; ok {
		return ref
	}

	switch refType {
	case "object":
		return g.resolveObjectRef(id)
	case "agent":
		for _, imp := range g.imports {
			if imp.ResourceType == "elementum_agent" && strings.Contains(imp.ID, id) {
				return "elementum_agent." + imp.ResourceName + ".id"
			}
		}
	case "cloudlink":
		for _, imp := range g.imports {
			if imp.ResourceType == "elementum_cloudlink" && strings.Contains(imp.ID, id) {
				return "elementum_cloudlink." + imp.ResourceName + ".id"
			}
		}
	case "document_model", "file_reader":
		for _, imp := range g.imports {
			if (imp.ResourceType == "elementum_ai_file_reader" ||
				imp.ResourceType == "elementum_text_file_reader" ||
				imp.ResourceType == "elementum_json_file_reader" ||
				imp.ResourceType == "elementum_xml_file_reader") && strings.Contains(imp.ID, id) {
				return imp.ResourceType + "." + imp.ResourceName + ".id"
			}
		}
	case "ai_provider_connector":
		for _, imp := range g.imports {
			if imp.ResourceType == "elementum_ai_provider_connector" && strings.Contains(imp.ID, id) {
				return "elementum_ai_provider_connector." + imp.ResourceName + ".id"
			}
		}
		// Fall back to raw ID - user will need to define a data source or use the ID directly
		return fmt.Sprintf("%q", id)
	case "approval_template":
		for _, imp := range g.imports {
			if imp.ResourceType == "elementum_approval_chain_template" && strings.Contains(imp.ID, id) {
				return "elementum_approval_chain_template." + imp.ResourceName + ".id"
			}
		}
		for _, imp := range g.imports {
			if imp.ResourceType == "elementum_approval_process" && strings.Contains(imp.ID, id) {
				return "elementum_approval_process." + imp.ResourceName + ".id"
			}
		}
	case "relationship":
		for _, imp := range g.imports {
			if imp.ResourceType == "elementum_relationship" && strings.Contains(imp.ID, id) {
				return "elementum_relationship." + imp.ResourceName + ".id"
			}
		}
	case "automation":
		for _, imp := range g.imports {
			if imp.ResourceType == "elementum_automation" && strings.Contains(imp.ID, id) {
				return "elementum_automation." + imp.ResourceName + ".id"
			}
		}
	}

	// General fallback
	for _, imp := range g.imports {
		if strings.Contains(imp.ID, id) {
			return imp.ResourceType + "." + imp.ResourceName + ".id"
		}
	}

	return fmt.Sprintf("%q", id)
}

// resolveFieldRef resolves a field ID to a terraform reference
func (g *TaskHCLGenerator) resolveFieldRef(fieldID string) string {
	// Check UUID map first
	if ref, ok := g.uuidMap[fieldID]; ok {
		return ref
	}

	// Check app fields
	if g.app != nil {
		for _, field := range g.app.Fields {
			if field.ID == fieldID {
				resourceType := "elementum_" + field.Type + "_field"
				resourceName := SanitizeName(field.Name)
				return resourceType + "." + resourceName + ".id"
			}
		}
	}

	// Check imports
	for _, imp := range g.imports {
		if strings.Contains(imp.ID, fieldID) && strings.HasSuffix(imp.ResourceType, "_field") {
			return imp.ResourceType + "." + imp.ResourceName + ".id"
		}
	}

	return fmt.Sprintf("%q", fieldID)
}

// GetParentRef returns the resolved parent reference for a task ID
func (g *TaskHCLGenerator) GetParentRef(taskID string) string {
	return g.parentMap[taskID]
}

// GetTaskRef returns the terraform resource reference for a task ID
func (g *TaskHCLGenerator) GetTaskRef(taskID string) string {
	return g.taskRefMap[taskID]
}

// GetTriggerRef returns the terraform resource reference for a trigger ID
func (g *TaskHCLGenerator) GetTriggerRef(triggerID string) string {
	return g.triggerRefMap[triggerID]
}

// ---------------------------------------------------------------------------
// IR Generation Methods
// ---------------------------------------------------------------------------

// GenerateAllIR generates IR blocks for all tasks in all automations
func (g *TaskHCLGenerator) GenerateAllIR() []*HCLBlock {
	var blocks []*HCLBlock

	for _, automation := range g.automations {
		if !automation.HasPublished || automation.Status != "ACTIVE" {
			continue
		}
		for _, task := range automation.Tasks {
			if task.Type == "unknown" {
				continue
			}
			b := g.GenerateTaskIR(&task, automation)
			if b != nil {
				blocks = append(blocks, b)
			}
		}
	}
	return blocks
}

// GenerateTaskIR generates an IR block for a single task resource
func (g *TaskHCLGenerator) GenerateTaskIR(task *discovery.Task, automation discovery.Automation) *HCLBlock {
	taskType := "elementum_" + task.Type + "_task"
	var resourceName string
	for _, imp := range g.imports {
		if imp.ResourceType == taskType && strings.HasSuffix(imp.ID, ":"+task.ID) {
			resourceName = imp.ResourceName
			break
		}
	}
	if resourceName == "" {
		return nil
	}

	b := NewResourceBlock(taskType, resourceName)

	if parentRef, ok := g.parentMap[task.ID]; ok {
		b.SetAttr("parent_id", Raw(parentRef))
	}
	if task.Name != "" {
		b.SetAttr("name", Str(task.Name))
	}

	g.generateTaskSpecificAttributesIR(b, task)
	return b
}

// generateTaskSpecificAttributesIR generates IR attributes for task-type-specific fields
func (g *TaskHCLGenerator) generateTaskSpecificAttributesIR(b *HCLBlock, task *discovery.Task) {
	config, ok := client.GetTaskTypeConfig(task.Type)
	if !ok {
		if task.ObjectID != "" {
			objectRef := g.resolveObjectRef(task.ObjectID)
			b.SetAttr("object_id", refOrStr(objectRef))
		}
		return
	}

	for _, field := range config.Fields {
		g.generateFieldAttributeIR(b, task, field)
	}
}

// generateFieldAttributeIR generates an IR attribute for a single field
func (g *TaskHCLGenerator) generateFieldAttributeIR(b *HCLBlock, task *discovery.Task, field client.TaskFieldConfig) {
	if task.RawData == nil {
		if field.Name == "object_id" && task.ObjectID != "" {
			objectRef := g.resolveObjectRef(task.ObjectID)
			b.SetAttr("object_id", refOrStr(objectRef))
		}
		return
	}

	value := extractValueFromPath(task.RawData, field.GraphQLPath)
	if value == nil {
		if field.Required {
			// Check if this nil is due to a known GraphQL partial-data error
			if task.BrokenFields != nil {
				// Match the top-level GraphQL field name (e.g., "agent" from "agent.id")
				topField := field.GraphQLPath
				if dotIdx := strings.Index(topField, "."); dotIdx >= 0 {
					topField = topField[:dotIdx]
				}
				if bracketIdx := strings.Index(topField, "["); bracketIdx >= 0 {
					topField = topField[:bracketIdx]
				}
				if _, broken := task.BrokenFields[topField]; broken {
					b.SetAttrComment(field.Name, Str(""), "# TODO: server error, fix manually after import")
					return
				}
			}
			b.SetAttr(field.Name, Str(""))
		}
		return
	}

	switch {
	case field.IsUserSearchEmail:
		g.setEmailFromUserSearchFilterIR(b, task, value)

	case field.IsReference:
		if strVal, ok := value.(string); ok && strVal != "" {
			ref := g.resolveRef(strVal, field.ReferenceType)
			b.SetAttr(field.Name, refOrStr(ref))
		}

	case field.IsValueReference:
		val := g.generateValueRefIR(task, value, field.Name)
		if val != nil {
			b.SetAttr(field.Name, val)
		} else if field.Required {
			b.SetAttr(field.Name, Str(""))
		}

	case field.IsArray:
		g.generateArrayFieldIR(b, field.Name, value, field.ArrayElementPath)

	case field.IsComplexArray:
		switch field.Name {
		case "headers":
			g.generateHeadersIR(b, task.RawData)
		case "parameters":
			if task.Type == "procedure" {
				g.generateProcedureParametersIR(b, task)
			}
		case "input_mappings":
			if task.Type == "run_automation" {
				g.generateRunAutomationInputMappingsIR(b, task)
			}
		case "dynamic_input_mappings":
			if task.Type == "run_automation" {
				g.generateRunAutomationDynamicInputMappingsIR(b, task)
			}
		case "output_mappings":
			if task.Type == "run_automation" {
				g.generateRunAutomationOutputMappingsIR(b, task)
			}
		}

	case field.Name == "category_source" && task.Type == "ai_classify":
		g.generateCategorySourceIR(b, task.RawData)

	case field.Name == "body" && task.Type == "api":
		g.generateApiBodyIR(b, task, task.RawData)

	case field.Name == "authorization" && task.Type == "api":
		g.generateAuthorizationIR(b, task, task.RawData)

	case field.Name == "response_type" && task.Type == "api":
		if strVal, ok := value.(string); ok && strVal != "" {
			b.SetAttr("response_type", Str(strings.ToUpper(strVal)))
		}

	case field.Name == "continue_on_error" && task.Type == "api":
		if boolVal, ok := value.(bool); ok && boolVal {
			b.SetAttr("continue_on_error", Bool(boolVal))
		}

	case field.Name == "fields":
		g.generateWorkflowFieldsIR(b, task, task.RawData)

	case field.Name == "filter":
		if filterData, ok := value.(map[string]interface{}); ok && len(filterData) > 0 {
			filterVal := GenerateFilterIR(filterData, g.uuidMap)
			if filterVal != nil {
				b.SetAttr("filter", filterVal)
			}
		}

	case field.Name == "sort":
		g.generateSortIR(b, task.RawData)

	case field.Name == "calculations":
		g.generateCalculationsIR(b, task.RawData)

	case field.Name == "limit":
		switch v := value.(type) {
		case float64:
			b.SetAttr(field.Name, Num(v))
		case int:
			b.SetAttr(field.Name, Num(float64(v)))
		}

	case field.Name == "method":
		if strVal, ok := value.(string); ok && strVal != "" {
			b.SetAttr(field.Name, Str(strings.ToUpper(strVal)))
		}

	case field.Name == "length":
		if strVal, ok := value.(string); ok && strVal != "" {
			b.SetAttr(field.Name, Str(strings.ToLower(strVal)))
		}

	case field.Name == "status":
		if strVal, ok := value.(string); ok && strVal != "" {
			b.SetAttr(field.Name, Str(strings.ToLower(strVal)))
		}

	case field.Name == "field_type":
		if strVal, ok := value.(string); ok && strVal != "" {
			b.SetAttr(field.Name, Str(strings.ToUpper(strVal)))
		}

	case field.Name == "variable_name":
		if strVal, ok := value.(string); ok && strVal != "" {
			b.SetAttr(field.Name, Str(strVal))
		}

	case field.Name == "variable_type":
		if strVal, ok := value.(string); ok && strVal != "" {
			b.SetAttr(field.Name, Str(strings.ToUpper(strVal)))
		}

	case field.Name == "notify_watchers" || field.Name == "strict":
		if boolVal, ok := value.(bool); ok {
			b.SetAttr(field.Name, Bool(boolVal))
		}

	default:
		switch v := value.(type) {
		case string:
			if v != "" {
				b.SetAttr(field.Name, Str(v))
			}
		case float64:
			b.SetAttr(field.Name, Num(v))
		case bool:
			b.SetAttr(field.Name, Bool(v))
		}
	}
}

// setEmailFromUserSearchFilterIR extracts email from user_search filter into IR
func (g *TaskHCLGenerator) setEmailFromUserSearchFilterIR(b *HCLBlock, task *discovery.Task, filterValue interface{}) {
	filter, ok := filterValue.(map[string]interface{})
	if !ok || filter == nil {
		return
	}

	valueRef, ok := filter["value"].(map[string]interface{})
	if !ok || valueRef == nil {
		return
	}

	if refsExpr := g.convertToRefsSyntax(task, valueRef); refsExpr != "" {
		b.SetAttr("email", Raw(refsExpr))
		return
	}

	if staticVal, ok := valueRef["value"].(string); ok && staticVal != "" {
		b.SetAttr("email", Str(staticVal))
		return
	}

	decoded := decodeValueReference(valueRef)
	if decoded != "" {
		// TODO: hasDynamicReference TODO comments are not yet natively supported on body-level
		b.SetAttr("email", Str(decoded))
		return
	}

	b.SetAttr("email", Str(""))
}

// generateValueRefIR generates an IR value for a value reference field
func (g *TaskHCLGenerator) generateValueRefIR(task *discovery.Task, value interface{}, fieldName string) HCLValue {
	ref, ok := value.(map[string]interface{})
	if !ok {
		return nil
	}

	if refsExpr := g.convertToRefsSyntax(task, ref); refsExpr != "" {
		return Raw(refsExpr)
	}

	decoded := decodeValueReference(ref)
	if decoded == "" {
		return nil
	}

	// TODO: hasDynamicReference TODO comments are not yet supported in IR values
	return formatHCLStringIR(decoded)
}

// generateArrayFieldIR generates IR for an array field
func (g *TaskHCLGenerator) generateArrayFieldIR(b *HCLBlock, fieldName string, value interface{}, elementPath string) {
	arr, ok := value.([]interface{})
	if !ok || len(arr) == 0 {
		return
	}

	var ids []string
	for _, item := range arr {
		switch v := item.(type) {
		case string:
			ids = append(ids, v)
		case map[string]interface{}:
			if elementPath != "" {
				if id, ok := v[elementPath].(string); ok && id != "" {
					ids = append(ids, id)
				}
			}
		}
	}

	if len(ids) == 0 {
		return
	}

	var vals []HCLValue
	for _, id := range ids {
		ref := g.resolveFieldRef(id)
		vals = append(vals, refOrStr(ref))
	}
	b.SetAttr(fieldName, List(vals...))
}

// generateWorkflowFieldsIR generates IR for workflow fields
func (g *TaskHCLGenerator) generateWorkflowFieldsIR(b *HCLBlock, task *discovery.Task, data map[string]interface{}) {
	wfFields, ok := data["workflowFields"].([]interface{})
	if !ok || len(wfFields) == 0 {
		return
	}

	var fieldObjs []HCLValue
	for _, wfInterface := range wfFields {
		wf, ok := wfInterface.(map[string]interface{})
		if !ok {
			continue
		}

		fieldID := ""
		if field, ok := wf["field"].(map[string]interface{}); ok {
			fieldID = getStringValue(field, "id")
		}
		if fieldID == "" {
			continue
		}

		fieldRef := g.resolveFieldRef(fieldID)

		attrs := []*HCLAttribute{
			Attr("field_id", refOrStr(fieldRef)),
		}

		if valueRef, ok := wf["valueReference"].(map[string]interface{}); ok {
			if refsExpr := g.convertToRefsSyntax(task, valueRef); refsExpr != "" {
				attrs = append(attrs, Attr("value", Raw(refsExpr)))
			} else {
				decoded := decodeValueReference(valueRef)
				if decoded != "" {
					attrs = append(attrs, Attr("value", Str(decoded)))
				}
			}
		}

		fieldObjs = append(fieldObjs, Obj(attrs...))
	}

	if len(fieldObjs) > 0 {
		b.SetAttr("fields", HCLList{Values: fieldObjs})
	}
}

// generateCategorySourceIR generates IR for ai_classify category source
func (g *TaskHCLGenerator) generateCategorySourceIR(b *HCLBlock, data map[string]interface{}) {
	categorySource, ok := data["categorySource"].(map[string]interface{})
	if !ok || categorySource == nil {
		g.generateLegacyCategoriesIR(b, data)
		return
	}

	typename := getStringValue(categorySource, "__typename")
	switch typename {
	case "AiClassifyCategoryStaticSource":
		g.generateStaticCategoriesIR(b, categorySource)
	case "AiClassifyCategoryDynamicSource":
		g.generateDynamicCategorySourceIR(b, categorySource)
	default:
		g.generateLegacyCategoriesIR(b, data)
	}
}

// generateStaticCategoriesIR generates IR for static categories
func (g *TaskHCLGenerator) generateStaticCategoriesIR(b *HCLBlock, source map[string]interface{}) {
	categories, ok := source["categories"].([]interface{})
	if !ok || len(categories) == 0 {
		b.SetAttr("categories", List())
		return
	}

	var catObjs []HCLValue
	for _, catInterface := range categories {
		cat, ok := catInterface.(map[string]interface{})
		if !ok {
			continue
		}

		name := ""
		if nameRef, ok := cat["name"].(map[string]interface{}); ok {
			name = decodeValueReference(nameRef)
		}
		desc := ""
		if descRef, ok := cat["description"].(map[string]interface{}); ok {
			desc = decodeValueReference(descRef)
		}

		attrs := []*HCLAttribute{
			Attr("label", formatHCLStringIR(name)),
		}
		if desc != "" {
			attrs = append(attrs, Attr("description", formatHCLStringIR(desc)))
		}
		catObjs = append(catObjs, Obj(attrs...))
	}

	b.SetAttr("categories", HCLList{Values: catObjs})
}

// generateDynamicCategorySourceIR generates IR for dynamic category source
func (g *TaskHCLGenerator) generateDynamicCategorySourceIR(b *HCLBlock, source map[string]interface{}) {
	var attrs []*HCLAttribute

	if aspect, ok := source["aspect"].(map[string]interface{}); ok {
		aspectID := getStringValue(aspect, "id")
		if aspectID != "" {
			objectRef := g.resolveObjectRef(aspectID)
			attrs = append(attrs, Attr("object_id", refOrStr(objectRef)))
		}
	}

	if labelField, ok := source["labelField"].(map[string]interface{}); ok {
		fieldID := getStringValue(labelField, "id")
		if fieldID != "" {
			fieldRef := g.resolveFieldRef(fieldID)
			attrs = append(attrs, Attr("label_field_id", refOrStr(fieldRef)))
		}
	}

	if descField, ok := source["descriptionField"].(map[string]interface{}); ok && descField != nil {
		fieldID := getStringValue(descField, "id")
		if fieldID != "" {
			fieldRef := g.resolveFieldRef(fieldID)
			attrs = append(attrs, Attr("description_field_id", refOrStr(fieldRef)))
		}
	}

	if filterData, ok := source["filter"].(map[string]interface{}); ok && len(filterData) > 0 {
		filterVal := GenerateFilterIR(filterData, g.uuidMap)
		if filterVal != nil {
			attrs = append(attrs, Attr("filter", filterVal))
		}
	}

	if limit, ok := source["limit"].(float64); ok && limit > 0 && limit != 500 {
		attrs = append(attrs, Attr("limit", Num(limit)))
	}

	b.SetAttr("dynamic_category_source", Obj(attrs...))
}

// generateLegacyCategoriesIR generates IR for legacy categories format
func (g *TaskHCLGenerator) generateLegacyCategoriesIR(b *HCLBlock, data map[string]interface{}) {
	categories, ok := data["categories"].([]interface{})
	if !ok || len(categories) == 0 {
		b.SetAttr("categories", List())
		return
	}

	var catObjs []HCLValue
	for _, catInterface := range categories {
		cat, ok := catInterface.(map[string]interface{})
		if !ok {
			continue
		}

		name := ""
		if nameRef, ok := cat["name"].(map[string]interface{}); ok {
			name = decodeValueReference(nameRef)
		}
		desc := ""
		if descRef, ok := cat["description"].(map[string]interface{}); ok {
			desc = decodeValueReference(descRef)
		}

		attrs := []*HCLAttribute{
			Attr("label", formatHCLStringIR(name)),
		}
		if desc != "" {
			attrs = append(attrs, Attr("description", formatHCLStringIR(desc)))
		}
		catObjs = append(catObjs, Obj(attrs...))
	}

	b.SetAttr("categories", HCLList{Values: catObjs})
}

// generateHeadersIR generates IR for api task headers
func (g *TaskHCLGenerator) generateHeadersIR(b *HCLBlock, data map[string]interface{}) {
	headers, ok := data["headers"].([]interface{})
	if !ok || len(headers) == 0 {
		return
	}

	var headerObjs []HCLValue
	for _, headerInterface := range headers {
		header, ok := headerInterface.(map[string]interface{})
		if !ok {
			continue
		}

		name := getStringValue(header, "header")
		if name == "" {
			name = getStringValue(header, "name")
		}
		if name == "" {
			continue
		}

		attrs := []*HCLAttribute{
			Attr("name", Str(name)),
		}

		if valueRef, ok := header["value"].(map[string]interface{}); ok {
			value := decodeValueReference(valueRef)
			if value != "" {
				attrs = append(attrs, Attr("value", Str(value)))
			}
		}

		headerObjs = append(headerObjs, Obj(attrs...))
	}

	if len(headerObjs) > 0 {
		b.SetAttr("headers", HCLList{Values: headerObjs})
	}
}

// generateProcedureParametersIR generates IR for procedure task parameters as a list attribute
func (g *TaskHCLGenerator) generateProcedureParametersIR(b *HCLBlock, task *discovery.Task) {
	params, ok := task.RawData["parameters"].([]interface{})
	if !ok || len(params) == 0 {
		return
	}

	var paramObjs []HCLValue
	for _, paramInterface := range params {
		param, ok := paramInterface.(map[string]interface{})
		if !ok {
			continue
		}

		var attrs []*HCLAttribute

		// Get index
		if idx, ok := param["index"].(float64); ok {
			attrs = append(attrs, Attr("index", Num(idx)))
		}

		// Get value (value reference)
		if valueRef, ok := param["value"].(map[string]interface{}); ok {
			// Try to convert to refs syntax first
			if refsExpr := g.convertToRefsSyntax(task, valueRef); refsExpr != "" {
				attrs = append(attrs, Attr("value", Raw(refsExpr)))
			} else {
				// Fall back to decoded value
				value := decodeValueReference(valueRef)
				if value != "" {
					attrs = append(attrs, Attr("value", Str(value)))
				}
			}
		}

		if len(attrs) > 0 {
			paramObjs = append(paramObjs, Obj(attrs...))
		}
	}

	if len(paramObjs) > 0 {
		b.SetAttr("parameters", HCLList{Values: paramObjs})
	}
}

// generateApiBodyIR generates IR for api task body (union type)
func (g *TaskHCLGenerator) generateApiBodyIR(b *HCLBlock, task *discovery.Task, data map[string]interface{}) {
	body, ok := data["body"].(map[string]interface{})
	if !ok || body == nil {
		return
	}

	typename := getStringValue(body, "__typename")
	switch typename {
	case "ApiCustomBody":
		bodyRef, ok := body["body"].(map[string]interface{})
		if !ok {
			return
		}
		contentType := getStringValue(body, "contentType")
		if contentType == "" {
			contentType = "text/plain"
		}

		var attrs []*HCLAttribute
		if refsExpr := g.convertToRefsSyntax(task, bodyRef); refsExpr != "" {
			attrs = append(attrs, Attr("content", Raw(refsExpr)))
		} else {
			value := decodeValueReference(bodyRef)
			if value != "" {
				attrs = append(attrs, Attr("content", Str(value)))
			}
		}
		attrs = append(attrs, Attr("content_type", Str(contentType)))
		b.SetAttr("custom_body", Obj(attrs...))

	case "ApiJsonBody":
		jsonRef, ok := body["jsonReference"].(map[string]interface{})
		if !ok {
			return
		}
		if refsExpr := g.convertToRefsSyntax(task, jsonRef); refsExpr != "" {
			b.SetAttr("json_body", Raw(refsExpr))
		} else {
			value := decodeValueReference(jsonRef)
			if value != "" {
				b.SetAttr("json_body", formatHCLStringIR(value))
			}
		}

	case "ApiFormBody":
		g.generateFormBodyIR(b, task, body)

	case "ApiMultipartFormBody":
		g.generateMultipartBodyIR(b, task, body)
	}
}

// generateFormBodyIR generates IR for api form body
func (g *TaskHCLGenerator) generateFormBodyIR(b *HCLBlock, task *discovery.Task, body map[string]interface{}) {
	values, ok := body["values"].([]interface{})
	if !ok || len(values) == 0 {
		return
	}

	var formObjs []HCLValue
	for _, v := range values {
		val, ok := v.(map[string]interface{})
		if !ok {
			continue
		}

		key := getStringValue(val, "key")
		if key == "" {
			continue
		}

		attrs := []*HCLAttribute{
			Attr("key", Str(key)),
		}

		if valueRef, ok := val["value"].(map[string]interface{}); ok {
			if refsExpr := g.convertToRefsSyntax(task, valueRef); refsExpr != "" {
				attrs = append(attrs, Attr("value", Raw(refsExpr)))
			} else {
				value := decodeValueReference(valueRef)
				attrs = append(attrs, Attr("value", Str(value)))
			}
		}

		formObjs = append(formObjs, Obj(attrs...))
	}

	if len(formObjs) > 0 {
		b.SetAttr("form_body", HCLList{Values: formObjs})
	}
}

// generateMultipartBodyIR generates IR for api multipart form body
func (g *TaskHCLGenerator) generateMultipartBodyIR(b *HCLBlock, task *discovery.Task, body map[string]interface{}) {
	parts, ok := body["parts"].([]interface{})
	if !ok || len(parts) == 0 {
		return
	}

	var partObjs []HCLValue
	for _, p := range parts {
		part, ok := p.(map[string]interface{})
		if !ok {
			continue
		}

		name := getStringValue(part, "name")
		if name == "" {
			continue
		}

		attrs := []*HCLAttribute{
			Attr("name", Str(name)),
		}

		// attachment
		if att, ok := part["attachment"].(map[string]interface{}); ok && att != nil {
			if refsExpr := g.convertToRefsSyntax(task, att); refsExpr != "" {
				attrs = append(attrs, Attr("attachment", Raw(refsExpr)))
			} else {
				attValue := decodeValueReference(att)
				if attValue != "" {
					attrs = append(attrs, Attr("attachment", Str(attValue)))
				}
			}
		}

		// attachmentId
		if attID, ok := part["attachmentId"].(map[string]interface{}); ok && attID != nil {
			if refsExpr := g.convertToRefsSyntax(task, attID); refsExpr != "" {
				attrs = append(attrs, Attr("attachment_id", Raw(refsExpr)))
			} else {
				attIDValue := decodeValueReference(attID)
				if attIDValue != "" {
					attrs = append(attrs, Attr("attachment_id", Str(attIDValue)))
				}
			}
		}

		// file
		if file, ok := part["file"].(map[string]interface{}); ok && file != nil {
			if refsExpr := g.convertToRefsSyntax(task, file); refsExpr != "" {
				attrs = append(attrs, Attr("file", Raw(refsExpr)))
			} else {
				fileValue := decodeValueReference(file)
				if fileValue != "" {
					attrs = append(attrs, Attr("file", Str(fileValue)))
				}
			}
		}

		partObjs = append(partObjs, Obj(attrs...))
	}

	if len(partObjs) > 0 {
		b.SetAttr("multipart_body", Obj(
			Attr("parts", HCLList{Values: partObjs}),
		))
	}
}

// generateAuthorizationIR generates IR for api task authorization (union type)
func (g *TaskHCLGenerator) generateAuthorizationIR(b *HCLBlock, task *discovery.Task, data map[string]interface{}) {
	auth, ok := data["authorization"].(map[string]interface{})
	if !ok || auth == nil {
		return
	}

	typename := getStringValue(auth, "__typename")
	switch typename {
	case "ApiBearerAuthorization":
		g.generateBearerAuthIR(b, task, auth)
	case "ApiBasicAuthorization":
		g.generateBasicAuthIR(b, task, auth)
	case "ApiOauthAuthorization":
		g.generateOauthIR(b, task, auth)
	}
}

// generateBearerAuthIR generates IR for bearer authorization
func (g *TaskHCLGenerator) generateBearerAuthIR(b *HCLBlock, task *discovery.Task, auth map[string]interface{}) {
	var attrs []*HCLAttribute

	if token, ok := auth["token"].(string); ok && token != "" {
		attrs = append(attrs, Attr("token", Str(token)))
	} else if refToken, ok := auth["referenceToken"].(map[string]interface{}); ok && refToken != nil {
		if refsExpr := g.convertToRefsSyntax(task, refToken); refsExpr != "" {
			attrs = append(attrs, Attr("reference_token", Raw(refsExpr)))
		} else {
			value := decodeValueReference(refToken)
			if value != "" {
				attrs = append(attrs, Attr("reference_token", Str(value)))
			}
		}
	}

	b.SetAttr("bearer_auth", Obj(attrs...))
}

// generateBasicAuthIR generates IR for basic authorization
func (g *TaskHCLGenerator) generateBasicAuthIR(b *HCLBlock, task *discovery.Task, auth map[string]interface{}) {
	var attrs []*HCLAttribute

	if username, ok := auth["username"].(string); ok && username != "" {
		attrs = append(attrs, Attr("username", Str(username)))
	}

	if password, ok := auth["password"].(string); ok && password != "" {
		attrs = append(attrs, Attr("password", Str(password)))
	} else if refPassword, ok := auth["referencePassword"].(map[string]interface{}); ok && refPassword != nil {
		if refsExpr := g.convertToRefsSyntax(task, refPassword); refsExpr != "" {
			attrs = append(attrs, Attr("reference_password", Raw(refsExpr)))
		} else {
			value := decodeValueReference(refPassword)
			if value != "" {
				attrs = append(attrs, Attr("reference_password", Str(value)))
			}
		}
	}

	b.SetAttr("basic_auth", Obj(attrs...))
}

// generateOauthIR generates IR for OAuth authorization
func (g *TaskHCLGenerator) generateOauthIR(b *HCLBlock, task *discovery.Task, auth map[string]interface{}) {
	var attrs []*HCLAttribute

	if clientID, ok := auth["clientId"].(string); ok && clientID != "" {
		attrs = append(attrs, Attr("client_id", Str(clientID)))
	}
	if clientSecret, ok := auth["clientSecret"].(string); ok && clientSecret != "" {
		attrs = append(attrs, Attr("client_secret", Str(clientSecret)))
	}
	if url, ok := auth["url"].(string); ok && url != "" {
		attrs = append(attrs, Attr("url", Str(url)))
	}
	if requestType, ok := auth["requestType"].(string); ok && requestType != "" {
		attrs = append(attrs, Attr("request_type", Str(strings.ToLower(requestType))))
	}

	if headers, ok := auth["headers"].([]interface{}); ok && len(headers) > 0 {
		var headerObjs []HCLValue
		for _, headerInterface := range headers {
			header, ok := headerInterface.(map[string]interface{})
			if !ok {
				continue
			}

			name := getStringValue(header, "header")
			if name == "" {
				name = getStringValue(header, "name")
			}
			if name == "" {
				continue
			}

			hAttrs := []*HCLAttribute{
				Attr("name", Str(name)),
			}

			if valueRef, ok := header["value"].(map[string]interface{}); ok {
				if refsExpr := g.convertToRefsSyntax(task, valueRef); refsExpr != "" {
					hAttrs = append(hAttrs, Attr("value", Raw(refsExpr)))
				} else {
					value := decodeValueReference(valueRef)
					if value != "" {
						hAttrs = append(hAttrs, Attr("value", Str(value)))
					}
				}
			}

			headerObjs = append(headerObjs, Obj(hAttrs...))
		}
		attrs = append(attrs, Attr("headers", HCLList{Values: headerObjs}))
	}

	b.SetAttr("oauth", Obj(attrs...))
}

// generateSortIR generates IR for sort configuration
func (g *TaskHCLGenerator) generateSortIR(b *HCLBlock, data map[string]interface{}) {
	sort, ok := data["sort"].(map[string]interface{})
	if !ok {
		return
	}

	fieldID := getStringValue(sort, "aspectFieldId")
	direction := getStringValue(sort, "direction")
	if fieldID == "" {
		return
	}

	var attrs []*HCLAttribute
	fieldRef := g.resolveFieldRef(fieldID)
	attrs = append(attrs, Attr("field_id", refOrStr(fieldRef)))
	if direction != "" {
		attrs = append(attrs, Attr("direction", Str(strings.ToLower(direction))))
	}

	b.SetAttr("sort", Obj(attrs...))
}

// generateCalculationsIR generates IR for calculation task calculations
func (g *TaskHCLGenerator) generateCalculationsIR(b *HCLBlock, data map[string]interface{}) {
	calculations, ok := data["calculations"].([]interface{})
	if !ok || len(calculations) == 0 {
		return
	}

	var calcObjs []HCLValue
	for _, calcInterface := range calculations {
		calc, ok := calcInterface.(map[string]interface{})
		if !ok {
			continue
		}

		variableName := getStringValue(calc, "name")
		formula := ""
		// First try the direct calculation field (deprecated but still populated)
		if calcStr, ok := calc["calculation"].(string); ok && calcStr != "" {
			formula = calcStr
		} else if calcRef, ok := calc["calculationReference"].(map[string]interface{}); ok {
			// Fall back to calculationReference for dynamic/templated formulas
			formula = decodeValueReference(calcRef)
		}

		attrs := []*HCLAttribute{
			Attr("variable_name", Str(variableName)),
		}
		if formula != "" {
			attrs = append(attrs, Attr("formula", Str(formula)))
		}
		calcObjs = append(calcObjs, Obj(attrs...))
	}

	b.SetAttr("calculations", HCLList{Values: calcObjs})
}

// generateRunAutomationInputMappingsIR generates IR for run_automation task input_mappings
func (g *TaskHCLGenerator) generateRunAutomationInputMappingsIR(b *HCLBlock, task *discovery.Task) {
	mappings, ok := task.RawData["inputMappings"].([]interface{})
	if !ok || len(mappings) == 0 {
		return
	}

	var mappingObjs []HCLValue
	for _, mappingInterface := range mappings {
		mapping, ok := mappingInterface.(map[string]interface{})
		if !ok {
			continue
		}

		var attrs []*HCLAttribute

		// Extract parameter.id
		if param, ok := mapping["parameter"].(map[string]interface{}); ok {
			if paramID, ok := param["id"].(string); ok && paramID != "" {
				// Parameter IDs reference the target automation's On Demand trigger parameters
				attrs = append(attrs, Attr("parameter_id", Str(paramID)))
			}
		}

		// Extract value reference
		if valueRef, ok := mapping["value"].(map[string]interface{}); ok && valueRef != nil {
			if refsExpr := g.convertToRefsSyntax(task, valueRef); refsExpr != "" {
				attrs = append(attrs, Attr("value", Raw(refsExpr)))
			} else {
				value := decodeValueReference(valueRef)
				if value != "" {
					attrs = append(attrs, Attr("value", Str(value)))
				}
			}
		}

		if len(attrs) > 0 {
			mappingObjs = append(mappingObjs, Obj(attrs...))
		}
	}

	if len(mappingObjs) > 0 {
		b.SetAttr("input_mappings", HCLList{Values: mappingObjs})
	}
}

// generateRunAutomationDynamicInputMappingsIR generates IR for run_automation task dynamic_input_mappings
func (g *TaskHCLGenerator) generateRunAutomationDynamicInputMappingsIR(b *HCLBlock, task *discovery.Task) {
	mappings, ok := task.RawData["dynamicInputMappings"].([]interface{})
	if !ok || len(mappings) == 0 {
		return
	}

	var mappingObjs []HCLValue
	for _, mappingInterface := range mappings {
		mapping, ok := mappingInterface.(map[string]interface{})
		if !ok {
			continue
		}

		var attrs []*HCLAttribute

		// Extract name
		if name, ok := mapping["name"].(string); ok && name != "" {
			attrs = append(attrs, Attr("name", Str(name)))
		}

		// Extract value reference
		if valueRef, ok := mapping["value"].(map[string]interface{}); ok && valueRef != nil {
			if refsExpr := g.convertToRefsSyntax(task, valueRef); refsExpr != "" {
				attrs = append(attrs, Attr("value", Raw(refsExpr)))
			} else {
				value := decodeValueReference(valueRef)
				if value != "" {
					attrs = append(attrs, Attr("value", Str(value)))
				}
			}
		}

		if len(attrs) > 0 {
			mappingObjs = append(mappingObjs, Obj(attrs...))
		}
	}

	if len(mappingObjs) > 0 {
		b.SetAttr("dynamic_input_mappings", HCLList{Values: mappingObjs})
	}
}

// generateRunAutomationOutputMappingsIR generates IR for run_automation task output_mappings
func (g *TaskHCLGenerator) generateRunAutomationOutputMappingsIR(b *HCLBlock, task *discovery.Task) {
	mappings, ok := task.RawData["outputMappings"].([]interface{})
	if !ok || len(mappings) == 0 {
		return
	}

	var mappingObjs []HCLValue
	for _, mappingInterface := range mappings {
		mapping, ok := mappingInterface.(map[string]interface{})
		if !ok {
			continue
		}

		// Extract name
		if name, ok := mapping["name"].(string); ok && name != "" {
			attrs := []*HCLAttribute{
				Attr("name", Str(name)),
			}
			mappingObjs = append(mappingObjs, Obj(attrs...))
		}
	}

	if len(mappingObjs) > 0 {
		b.SetAttr("output_mappings", HCLList{Values: mappingObjs})
	}
}

// formatHCLStringIR returns an HCLValue for a string that may need heredoc formatting
func formatHCLStringIR(s string) HCLValue {
	if strings.Contains(s, "\n") {
		return Heredoc(s, "EOT")
	}
	return Str(s)
}

// BeautifyTask applies beautification to a single task resource block in HCL
func (g *TaskHCLGenerator) BeautifyTask(hcl string, task *discovery.Task) string {
	// Replace UUID strings with terraform references
	for uuid, ref := range g.uuidMap {
		// Replace quoted UUIDs
		hcl = strings.ReplaceAll(hcl, fmt.Sprintf("%q", uuid), ref)
	}

	return hcl
}

// GetSupportedTaskTypes returns all supported task types
func GetSupportedTaskTypesFromRegistry() []string {
	types := make([]string, 0)
	for typeName := range client.TaskTypeRegistry {
		types = append(types, typeName)
	}
	return types
}
