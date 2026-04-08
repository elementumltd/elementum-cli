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

// Package state provides functionality for reading Terraform state files
// and extracting Elementum resource information, particularly refs.
package state

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// TerraformState represents the structure of a Terraform state file
type TerraformState struct {
	Version          int        `json:"version"`
	TerraformVersion string     `json:"terraform_version"`
	Serial           int        `json:"serial"`
	Lineage          string     `json:"lineage"`
	Resources        []Resource `json:"resources"`
}

// Resource represents a resource in Terraform state
type Resource struct {
	Module    string     `json:"module,omitempty"`
	Mode      string     `json:"mode"`
	Type      string     `json:"type"`
	Name      string     `json:"name"`
	Provider  string     `json:"provider"`
	Instances []Instance `json:"instances"`
}

// Instance represents an instance of a resource
type Instance struct {
	IndexKey      any            `json:"index_key,omitempty"`
	SchemaVersion int            `json:"schema_version"`
	Attributes    map[string]any `json:"attributes"`
}

// RefInfo represents a single ref entry with metadata
type RefInfo struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Multiple bool   `json:"multiple"`
	RawJSON  string `json:"raw_json,omitempty"`
}

// ResourceRefs represents refs for a single Terraform resource
type ResourceRefs struct {
	Address      string    `json:"address"`       // e.g., "elementum_record_created_trigger.my_trigger"
	ResourceType string    `json:"resource_type"` // e.g., "elementum_record_created_trigger"
	ResourceName string    `json:"resource_name"` // e.g., "my_trigger"
	Module       string    `json:"module,omitempty"`
	IndexKey     string    `json:"index_key,omitempty"`
	Refs         []RefInfo `json:"refs"`
}

// StateRefs contains all refs found in a state file
type StateRefs struct {
	StateFile string         `json:"state_file"`
	Resources []ResourceRefs `json:"resources"`
}

// FindStateFile looks for terraform.tfstate in the current directory or specified path
func FindStateFile(path string) (string, error) {
	if path != "" {
		// Check if the specified path exists
		if _, err := os.Stat(path); err != nil {
			return "", fmt.Errorf("state file not found: %s", path)
		}
		return path, nil
	}

	// Default: look for terraform.tfstate in current directory
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get current directory: %w", err)
	}

	defaultPath := filepath.Join(cwd, "terraform.tfstate")
	if _, err := os.Stat(defaultPath); err != nil {
		return "", fmt.Errorf("no state file found at %s (use --state-file to specify)", defaultPath)
	}

	return defaultPath, nil
}

// ReadStateFile reads and parses a Terraform state file
func ReadStateFile(path string) (*TerraformState, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read state file: %w", err)
	}

	var state TerraformState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("failed to parse state file: %w", err)
	}

	return &state, nil
}

// isElementumResource checks if a resource type is an Elementum resource with refs
func isElementumResource(resourceType string) bool {
	// All Elementum trigger and task resources have refs
	return strings.HasPrefix(resourceType, "elementum_") &&
		(strings.Contains(resourceType, "_trigger") || strings.Contains(resourceType, "_task"))
}

// ExtractRefs extracts all refs from a Terraform state
func ExtractRefs(state *TerraformState, resourceFilter string) (*StateRefs, error) {
	result := &StateRefs{
		Resources: []ResourceRefs{},
	}

	for _, resource := range state.Resources {
		// Skip non-managed resources (data sources)
		if resource.Mode != "managed" {
			continue
		}

		// Check if this is an Elementum resource with refs
		if !isElementumResource(resource.Type) {
			continue
		}

		for _, instance := range resource.Instances {
			// Build resource address
			address := buildResourceAddress(resource.Module, resource.Type, resource.Name, instance.IndexKey)

			// Apply resource filter if specified
			if resourceFilter != "" && !strings.Contains(address, resourceFilter) {
				continue
			}

			// Extract refs from attributes
			refs := extractRefsFromAttributes(instance.Attributes)
			if len(refs) == 0 {
				continue
			}

			resourceRefs := ResourceRefs{
				Address:      address,
				ResourceType: resource.Type,
				ResourceName: resource.Name,
				Module:       resource.Module,
				Refs:         refs,
			}

			if instance.IndexKey != nil {
				resourceRefs.IndexKey = fmt.Sprintf("%v", instance.IndexKey)
			}

			result.Resources = append(result.Resources, resourceRefs)
		}
	}

	// Sort resources by address for consistent output
	sort.Slice(result.Resources, func(i, j int) bool {
		return result.Resources[i].Address < result.Resources[j].Address
	})

	return result, nil
}

// buildResourceAddress constructs the full Terraform resource address
func buildResourceAddress(module, resourceType, name string, indexKey any) string {
	var address string
	if module != "" {
		address = module + "."
	}
	address += resourceType + "." + name

	if indexKey != nil {
		switch v := indexKey.(type) {
		case string:
			address += fmt.Sprintf("[\"%s\"]", v)
		case float64:
			address += fmt.Sprintf("[%d]", int(v))
		default:
			address += fmt.Sprintf("[%v]", v)
		}
	}

	return address
}

// extractRefsFromAttributes extracts refs from resource attributes
func extractRefsFromAttributes(attrs map[string]any) []RefInfo {
	refs := []RefInfo{}

	refsRaw, ok := attrs["refs"]
	if !ok || refsRaw == nil {
		return refs
	}

	refsMap, ok := refsRaw.(map[string]any)
	if !ok {
		return refs
	}

	// Extract each ref
	for name, value := range refsMap {
		valueStr, ok := value.(string)
		if !ok {
			continue
		}

		refInfo := parseRefJSON(name, valueStr)
		refs = append(refs, refInfo)
	}

	// Sort refs by name for consistent output
	sort.Slice(refs, func(i, j int) bool {
		return refs[i].Name < refs[j].Name
	})

	return refs
}

// parseRefJSON parses the JSON-encoded ref value to extract type and multiple flag
func parseRefJSON(name, jsonStr string) RefInfo {
	refInfo := RefInfo{
		Name:    name,
		Type:    "UNKNOWN",
		RawJSON: jsonStr,
	}

	// Parse the JSON to extract fieldType and multiple
	var refData map[string]any
	if err := json.Unmarshal([]byte(jsonStr), &refData); err != nil {
		return refInfo
	}

	if fieldType, ok := refData["fieldType"].(string); ok {
		refInfo.Type = fieldType
	}

	if multiple, ok := refData["multiple"].(bool); ok {
		refInfo.Multiple = multiple
	}

	return refInfo
}

// FormatRefType formats the type with multiple indicator
func FormatRefType(refInfo RefInfo) string {
	typeStr := refInfo.Type
	if refInfo.Multiple {
		typeStr += "[]"
	}
	return typeStr
}

// AutomationInfo represents an automation resource from state
type AutomationInfo struct {
	Address      string     `json:"address"`
	ResourceName string     `json:"resource_name"`
	Module       string     `json:"module,omitempty"`
	ID           string     `json:"id"`
	Name         string     `json:"name"`
	AppID        string     `json:"app_id"`
	WorkflowID   string     `json:"workflow_id"`
	Status       string     `json:"status,omitempty"`
	TriggerType  string     `json:"trigger_type,omitempty"`
	TriggerID    string     `json:"trigger_id,omitempty"`
	Refs         []RefInfo  `json:"refs,omitempty"`
	Tasks        []TaskInfo `json:"tasks,omitempty"`
}

// TaskInfo represents a task resource associated with an automation
type TaskInfo struct {
	Address      string    `json:"address"`
	ResourceType string    `json:"resource_type"`
	ResourceName string    `json:"resource_name"`
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	TaskType     string    `json:"task_type"`
	WorkflowID   string    `json:"workflow_id"`
	Refs         []RefInfo `json:"refs,omitempty"`
}

// StateAutomations contains all automations found in state
type StateAutomations struct {
	StateFile   string           `json:"state_file"`
	Automations []AutomationInfo `json:"automations"`
}

// ExtractAutomations extracts all automation resources from state
func ExtractAutomations(state *TerraformState, automationFilter string) (*StateAutomations, error) {
	result := &StateAutomations{
		Automations: []AutomationInfo{},
	}

	// First pass: collect all automations
	automationsByWorkflowID := make(map[string]*AutomationInfo)

	for _, resource := range state.Resources {
		if resource.Mode != "managed" {
			continue
		}

		if resource.Type != "elementum_automation" {
			continue
		}

		for _, instance := range resource.Instances {
			address := buildResourceAddress(resource.Module, resource.Type, resource.Name, instance.IndexKey)

			// Apply filter if specified
			if automationFilter != "" && !strings.Contains(address, automationFilter) {
				continue
			}

			automation := extractAutomationInfo(resource, instance, address)
			result.Automations = append(result.Automations, automation)

			// Index by workflow_id for task association
			if automation.WorkflowID != "" {
				automationsByWorkflowID[automation.WorkflowID] = &result.Automations[len(result.Automations)-1]
			}
		}
	}

	// Second pass: associate tasks with automations
	for _, resource := range state.Resources {
		if resource.Mode != "managed" {
			continue
		}

		// Check if this is a task resource
		if !strings.HasPrefix(resource.Type, "elementum_") || !strings.HasSuffix(resource.Type, "_task") {
			continue
		}

		for _, instance := range resource.Instances {
			task := extractTaskInfo(resource, instance)

			// Associate with automation by workflow_id
			if automation, ok := automationsByWorkflowID[task.WorkflowID]; ok {
				automation.Tasks = append(automation.Tasks, task)
			}
		}
	}

	// Sort automations by address
	sort.Slice(result.Automations, func(i, j int) bool {
		return result.Automations[i].Address < result.Automations[j].Address
	})

	return result, nil
}

// extractAutomationInfo extracts automation details from a resource instance
func extractAutomationInfo(resource Resource, instance Instance, address string) AutomationInfo {
	attrs := instance.Attributes

	automation := AutomationInfo{
		Address:      address,
		ResourceName: resource.Name,
		Module:       resource.Module,
	}

	if id, ok := attrs["id"].(string); ok {
		automation.ID = id
	}
	if name, ok := attrs["name"].(string); ok {
		automation.Name = name
	}
	if appID, ok := attrs["app_id"].(string); ok {
		automation.AppID = appID
	}
	if workflowID, ok := attrs["workflow_id"].(string); ok {
		automation.WorkflowID = workflowID
	}
	if status, ok := attrs["status"].(string); ok {
		automation.Status = status
	}

	// Extract trigger info from nested block
	if triggers, ok := attrs["trigger"].([]any); ok && len(triggers) > 0 {
		if trigger, ok := triggers[0].(map[string]any); ok {
			if triggerType, ok := trigger["type"].(string); ok {
				automation.TriggerType = triggerType
			}
			if triggerID, ok := trigger["id"].(string); ok {
				automation.TriggerID = triggerID
			}
		}
	}

	// Extract refs
	automation.Refs = extractRefsFromAttributes(attrs)

	return automation
}

// extractTaskInfo extracts task details from a resource instance
func extractTaskInfo(resource Resource, instance Instance) TaskInfo {
	attrs := instance.Attributes
	address := buildResourceAddress(resource.Module, resource.Type, resource.Name, instance.IndexKey)

	// Derive task type from resource type (e.g., "elementum_message_task" -> "message")
	taskType := strings.TrimPrefix(resource.Type, "elementum_")
	taskType = strings.TrimSuffix(taskType, "_task")

	task := TaskInfo{
		Address:      address,
		ResourceType: resource.Type,
		ResourceName: resource.Name,
		TaskType:     taskType,
	}

	if id, ok := attrs["id"].(string); ok {
		task.ID = id
	}
	if name, ok := attrs["name"].(string); ok {
		task.Name = name
	}
	if workflowID, ok := attrs["workflow_id"].(string); ok {
		task.WorkflowID = workflowID
	}

	// Extract refs
	task.Refs = extractRefsFromAttributes(attrs)

	return task
}

// FindAutomationByID finds an automation in state by its ID
func FindAutomationByID(state *TerraformState, automationID string) (*AutomationInfo, error) {
	automations, err := ExtractAutomations(state, "")
	if err != nil {
		return nil, err
	}

	for _, automation := range automations.Automations {
		if automation.ID == automationID {
			return &automation, nil
		}
	}

	return nil, fmt.Errorf("automation with ID %s not found in state", automationID)
}

// FindAutomationByName finds an automation in state by its resource name
func FindAutomationByName(state *TerraformState, resourceName string) (*AutomationInfo, error) {
	automations, err := ExtractAutomations(state, "")
	if err != nil {
		return nil, err
	}

	for _, automation := range automations.Automations {
		if automation.ResourceName == resourceName || automation.Name == resourceName {
			return &automation, nil
		}
	}

	return nil, fmt.Errorf("automation '%s' not found in state", resourceName)
}
