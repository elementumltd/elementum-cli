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
	"context"
	"encoding/json"
	"fmt"

	"github.com/elementumltd/elementum-cli/internal/client"
	"github.com/elementumltd/elementum-cli/logger"
)

// AspectType represents the type of aspect
type AspectType string

const (
	AspectTypeApp     AspectType = "App"
	AspectTypeElement AspectType = "Element"
	AspectTypeTask    AspectType = "Task"
)

// AspectCapabilities defines what resources each aspect type supports
type AspectCapabilities struct {
	SupportsFlows       bool
	SupportsAgents      bool
	SupportsApprovals   bool
	SupportsFileReaders bool
}

// Capabilities returns the capabilities for each aspect type
var Capabilities = map[AspectType]AspectCapabilities{
	AspectTypeApp:     {SupportsFlows: true, SupportsAgents: true, SupportsApprovals: true, SupportsFileReaders: true},
	AspectTypeElement: {SupportsFlows: false, SupportsAgents: false, SupportsApprovals: false, SupportsFileReaders: false},
	AspectTypeTask:    {SupportsFlows: false, SupportsAgents: true, SupportsApprovals: false, SupportsFileReaders: false},
}

// getAspectRoles fetches roles for any aspect type (App, Element, or Task)
func getAspectRoles(ctx context.Context, c *client.Client, aspectID string) ([]Role, error) {
	result, err := client.GetRolesWithMembers(ctx, c.Genqlient(), aspectID)
	if err != nil {
		return nil, err
	}

	org := result.GetOrganization()
	aspect := org.Aspect
	if aspect == nil {
		return nil, nil
	}

	// Handle all three aspect types using type switch
	switch a := (*aspect).(type) {
	case *client.GetRolesWithMembersOrganizationAspectAspectApp:
		return extractRolesFromApp(a), nil
	case *client.GetRolesWithMembersOrganizationAspectAspectElement:
		return extractRolesFromElement(a), nil
	case *client.GetRolesWithMembersOrganizationAspectAspectTask:
		return extractRolesFromTask(a), nil
	default:
		logger.Debug("unsupported aspect type for roles", "aspectID", aspectID)
		return nil, nil
	}
}

// extractRolesFromApp extracts roles from an AspectApp response
func extractRolesFromApp(aspectApp *client.GetRolesWithMembersOrganizationAspectAspectApp) []Role {
	roles := make([]Role, 0, len(aspectApp.Roles.Edges))
	for _, edge := range aspectApp.Roles.Edges {
		node := edge.Node

		role := Role{
			ID:      node.Id,
			Name:    node.Name,
			Managed: node.Managed,
			Tags:    node.Tags,
		}

		// Extract description (may be nil)
		if node.Description != nil {
			role.Description = *node.Description
		}

		// Convert AutoShare enum values to strings
		for _, as := range node.AutoShare {
			role.AutoShare = append(role.AutoShare, string(as))
		}

		// Extract users with emails
		for _, userEdge := range node.Users.Edges {
			role.Users = append(role.Users, RoleMember{
				ID:   userEdge.Node.Id,
				Name: userEdge.Node.Email,
			})
		}

		// Extract groups with names
		for _, groupEdge := range node.Groups.Edges {
			role.Groups = append(role.Groups, RoleMember{
				ID:   groupEdge.Node.Id,
				Name: groupEdge.Node.Name,
			})
		}

		// Extract permissions
		for _, permEdge := range node.PermissionsV2.Edges {
			perm := RolePermission{
				Group: string(permEdge.Node.Group.Group),
				Level: string(permEdge.Node.Level.Level),
			}
			// Extract custom permissions
			for _, ap := range permEdge.Node.AssignedPermissions {
				perm.CustomPermissions = append(perm.CustomPermissions, string(ap.Permission))
			}
			role.Permissions = append(role.Permissions, perm)
		}

		roles = append(roles, role)
	}
	return roles
}

// extractRolesFromElement extracts roles from an AspectElement response
func extractRolesFromElement(aspectElement *client.GetRolesWithMembersOrganizationAspectAspectElement) []Role {
	roles := make([]Role, 0, len(aspectElement.Roles.Edges))
	for _, edge := range aspectElement.Roles.Edges {
		node := edge.Node

		role := Role{
			ID:      node.Id,
			Name:    node.Name,
			Managed: node.Managed,
			Tags:    node.Tags,
		}

		// Extract description (may be nil)
		if node.Description != nil {
			role.Description = *node.Description
		}

		// Convert AutoShare enum values to strings
		for _, as := range node.AutoShare {
			role.AutoShare = append(role.AutoShare, string(as))
		}

		// Extract users with emails
		for _, userEdge := range node.Users.Edges {
			role.Users = append(role.Users, RoleMember{
				ID:   userEdge.Node.Id,
				Name: userEdge.Node.Email,
			})
		}

		// Extract groups with names
		for _, groupEdge := range node.Groups.Edges {
			role.Groups = append(role.Groups, RoleMember{
				ID:   groupEdge.Node.Id,
				Name: groupEdge.Node.Name,
			})
		}

		// Extract permissions
		for _, permEdge := range node.PermissionsV2.Edges {
			perm := RolePermission{
				Group: string(permEdge.Node.Group.Group),
				Level: string(permEdge.Node.Level.Level),
			}
			// Extract custom permissions
			for _, ap := range permEdge.Node.AssignedPermissions {
				perm.CustomPermissions = append(perm.CustomPermissions, string(ap.Permission))
			}
			role.Permissions = append(role.Permissions, perm)
		}

		roles = append(roles, role)
	}
	return roles
}

// extractRolesFromTask extracts roles from an AspectTask response
func extractRolesFromTask(aspectTask *client.GetRolesWithMembersOrganizationAspectAspectTask) []Role {
	roles := make([]Role, 0, len(aspectTask.Roles.Edges))
	for _, edge := range aspectTask.Roles.Edges {
		node := edge.Node

		role := Role{
			ID:      node.Id,
			Name:    node.Name,
			Managed: node.Managed,
			Tags:    node.Tags,
		}

		// Extract description (may be nil)
		if node.Description != nil {
			role.Description = *node.Description
		}

		// Convert AutoShare enum values to strings
		for _, as := range node.AutoShare {
			role.AutoShare = append(role.AutoShare, string(as))
		}

		// Extract users with emails
		for _, userEdge := range node.Users.Edges {
			role.Users = append(role.Users, RoleMember{
				ID:   userEdge.Node.Id,
				Name: userEdge.Node.Email,
			})
		}

		// Extract groups with names
		for _, groupEdge := range node.Groups.Edges {
			role.Groups = append(role.Groups, RoleMember{
				ID:   groupEdge.Node.Id,
				Name: groupEdge.Node.Name,
			})
		}

		// Extract permissions
		for _, permEdge := range node.PermissionsV2.Edges {
			perm := RolePermission{
				Group: string(permEdge.Node.Group.Group),
				Level: string(permEdge.Node.Level.Level),
			}
			// Extract custom permissions
			for _, ap := range permEdge.Node.AssignedPermissions {
				perm.CustomPermissions = append(perm.CustomPermissions, string(ap.Permission))
			}
			role.Permissions = append(role.Permissions, perm)
		}

		roles = append(roles, role)
	}
	return roles
}

// getAspectAccessPolicies fetches access policies for any aspect type (App, Element, or Task)
func getAspectAccessPolicies(ctx context.Context, c *client.Client, aspectID string) ([]AccessPolicy, error) {
	result, err := client.GetAccessPolicies(ctx, c.Genqlient(), aspectID)
	if err != nil {
		return nil, err
	}

	org := result.GetOrganization()
	aspect := org.Aspect
	if aspect == nil {
		return nil, nil
	}

	// Handle all three aspect types using type switch
	switch a := (*aspect).(type) {
	case *client.GetAccessPoliciesOrganizationAspectAspectApp:
		return extractAccessPoliciesFromApp(a, aspectID), nil
	case *client.GetAccessPoliciesOrganizationAspectAspectElement:
		return extractAccessPoliciesFromElement(a, aspectID), nil
	case *client.GetAccessPoliciesOrganizationAspectAspectTask:
		return extractAccessPoliciesFromTask(a, aspectID), nil
	default:
		logger.Debug("unsupported aspect type for access policies", "aspectID", aspectID)
		return nil, nil
	}
}

// extractAccessPoliciesFromApp extracts access policies from an AspectApp response
func extractAccessPoliciesFromApp(aspectApp *client.GetAccessPoliciesOrganizationAspectAspectApp, aspectID string) []AccessPolicy {
	policies := make([]AccessPolicy, 0, len(aspectApp.AccessPolicies.Edges))
	for _, edge := range aspectApp.AccessPolicies.Edges {
		node := edge.Node
		policy := AccessPolicy{
			ID:         node.Id,
			ObjectID:   aspectID,
			UserEmails: make(map[string]string),
			GroupNames: make(map[string]string),
		}

		// Parse filter
		if len(node.Filter) > 0 {
			if err := json.Unmarshal(node.Filter, &policy.Filter); err != nil {
				logger.Warn("failed to unmarshal access policy filter", "policyID", node.Id, "error", err)
			}
		}

		// Extract users with IDs and emails
		for _, userEdge := range node.Users.Edges {
			policy.UserIDs = append(policy.UserIDs, userEdge.Node.Id)
			policy.UserEmails[userEdge.Node.Id] = userEdge.Node.Email
		}

		// Extract groups with IDs and names
		for _, groupEdge := range node.Groups.Edges {
			policy.GroupIDs = append(policy.GroupIDs, groupEdge.Node.Id)
			policy.GroupNames[groupEdge.Node.Id] = groupEdge.Node.Name
		}

		policies = append(policies, policy)
	}
	return policies
}

// extractAccessPoliciesFromElement extracts access policies from an AspectElement response
func extractAccessPoliciesFromElement(aspectElement *client.GetAccessPoliciesOrganizationAspectAspectElement, aspectID string) []AccessPolicy {
	policies := make([]AccessPolicy, 0, len(aspectElement.AccessPolicies.Edges))
	for _, edge := range aspectElement.AccessPolicies.Edges {
		node := edge.Node
		policy := AccessPolicy{
			ID:         node.Id,
			ObjectID:   aspectID,
			UserEmails: make(map[string]string),
			GroupNames: make(map[string]string),
		}

		// Parse filter
		if len(node.Filter) > 0 {
			if err := json.Unmarshal(node.Filter, &policy.Filter); err != nil {
				logger.Warn("failed to unmarshal access policy filter", "policyID", node.Id, "error", err)
			}
		}

		// Extract users with IDs and emails
		for _, userEdge := range node.Users.Edges {
			policy.UserIDs = append(policy.UserIDs, userEdge.Node.Id)
			policy.UserEmails[userEdge.Node.Id] = userEdge.Node.Email
		}

		// Extract groups with IDs and names
		for _, groupEdge := range node.Groups.Edges {
			policy.GroupIDs = append(policy.GroupIDs, groupEdge.Node.Id)
			policy.GroupNames[groupEdge.Node.Id] = groupEdge.Node.Name
		}

		policies = append(policies, policy)
	}
	return policies
}

// extractAccessPoliciesFromTask extracts access policies from an AspectTask response
func extractAccessPoliciesFromTask(aspectTask *client.GetAccessPoliciesOrganizationAspectAspectTask, aspectID string) []AccessPolicy {
	policies := make([]AccessPolicy, 0, len(aspectTask.AccessPolicies.Edges))
	for _, edge := range aspectTask.AccessPolicies.Edges {
		node := edge.Node
		policy := AccessPolicy{
			ID:         node.Id,
			ObjectID:   aspectID,
			UserEmails: make(map[string]string),
			GroupNames: make(map[string]string),
		}

		// Parse filter
		if len(node.Filter) > 0 {
			if err := json.Unmarshal(node.Filter, &policy.Filter); err != nil {
				logger.Warn("failed to unmarshal access policy filter", "policyID", node.Id, "error", err)
			}
		}

		// Extract users with IDs and emails
		for _, userEdge := range node.Users.Edges {
			policy.UserIDs = append(policy.UserIDs, userEdge.Node.Id)
			policy.UserEmails[userEdge.Node.Id] = userEdge.Node.Email
		}

		// Extract groups with IDs and names
		for _, groupEdge := range node.Groups.Edges {
			policy.GroupIDs = append(policy.GroupIDs, groupEdge.Node.Id)
			policy.GroupNames[groupEdge.Node.Id] = groupEdge.Node.Name
		}

		policies = append(policies, policy)
	}
	return policies
}

// GetAspectAutomations fetches automations for any aspect type (App, Element, or Task)
// This is a generic function that works with any aspect ID.
// It automatically filters out DISABLED automations.
func GetAspectAutomations(ctx context.Context, c *client.Client, aspectID string) ([]Automation, error) {
	// Use the genqlient-generated query that includes task aspect references
	// Filter out DISABLED automations at the API level
	resp, err := client.GetAspectAutomationsForDiscovery(ctx, c.Genqlient(), aspectID, client.BuildNotDisabledFilter())
	if err != nil {
		return nil, err
	}

	// Extract automations using the helper function
	discoveryAutomations := client.ExtractDiscoveryAutomations(resp)

	// Convert to our discovery types
	automations := make([]Automation, 0, len(discoveryAutomations))
	for _, da := range discoveryAutomations {
		automation := Automation{
			ID:           da.ID,
			Name:         da.Name,
			Status:       da.Status,
			WorkflowID:   da.WorkflowID,
			HasPublished: da.HasPublished,
			HasDraft:     da.HasDraft,
		}

		// Convert triggers
		for _, dt := range da.Triggers {
			trigger := Trigger{
				ID:         dt.ID,
				Type:       dt.Type,
				Name:       dt.Type, // Use type as name since triggers don't have explicit names
				DatamineID: dt.DatamineID,
			}
			automation.Triggers = append(automation.Triggers, trigger)
		}

		// Convert tasks
		for _, dt := range da.Tasks {
			workflowTask := Task{
				ID:              dt.ID,
				Type:            dt.Type,
				Name:            dt.Name,
				WorkflowID:      da.WorkflowID,
				ParentID:        dt.PreviousID,
				ObjectID:        dt.ObjectID,
				RelatedObjectID: dt.RelatedObjectID,
			}
			automation.Tasks = append(automation.Tasks, workflowTask)
		}

		automations = append(automations, automation)
	}

	return automations, nil
}

// FetchAutomationFullDetails fetches full task/trigger details (including RawData) for automations.
// This populates the RawData field needed for HCL generation.
// aspectID is the ID of the aspect (app, element, or task) that owns these automations.
func FetchAutomationFullDetails(ctx context.Context, c *client.Client, aspectID string, automations []Automation) error {
	if len(automations) == 0 {
		return nil
	}

	tasksFragment := client.BuildTasksQueryFragment()
	triggersFragment := buildTriggersQueryFragment()

	for i := range automations {
		automation := &automations[i]

		// Skip automations without workflows or unpublished/inactive ones
		if automation.WorkflowID == "" || !automation.HasPublished || automation.Status != "ACTIVE" {
			continue
		}

		query := fmt.Sprintf(`
			query GetFullWorkflowDetails($aspectId: ID!, $automationId: ID!) {
				organization {
					aspect(id: $aspectId) {
						id
						automation(id: $automationId) {
							id
							current {
								id
								outputs {
									id
									name
									value {
										taskReference { name }
										triggerReference { name }
										variableReference { name }
									}
								}
								triggers {
									id
									__typename
									%s
								}
								%s
							}
						}
					}
				}
			}
		`, triggersFragment, tasksFragment)

		variables := map[string]interface{}{
			"aspectId":     aspectID,
			"automationId": automation.ID,
		}

		var result map[string]interface{}
		if err := c.ExecuteInto(ctx, query, variables, &result); err != nil {
			logger.Warn("failed to fetch full automation details",
				"automationId", automation.ID,
				"aspectId", aspectID,
				"error", err)
			continue // Don't fail the whole batch
		}

		// Extract workflow data from automation.current
		workflowData := extractCurrentWorkflowData(result)
		if workflowData == nil {
			continue
		}

		// Update trigger RawData
		if triggers, ok := workflowData["triggers"].([]interface{}); ok {
			triggerByID := make(map[string]*Trigger)
			for j := range automation.Triggers {
				triggerByID[automation.Triggers[j].ID] = &automation.Triggers[j]
			}
			for _, triggerInterface := range triggers {
				if triggerData, ok := triggerInterface.(map[string]interface{}); ok {
					triggerID := getString(triggerData, "id")
					if trigger, exists := triggerByID[triggerID]; exists {
						trigger.RawData = triggerData
					}
				}
			}
		}

		// Update task RawData
		if tasks, ok := workflowData["tasks"].([]interface{}); ok {
			taskByID := make(map[string]*Task)
			for j := range automation.Tasks {
				taskByID[automation.Tasks[j].ID] = &automation.Tasks[j]
			}
			for _, taskInterface := range tasks {
				if taskData, ok := taskInterface.(map[string]interface{}); ok {
					taskID := getString(taskData, "id")
					if task, exists := taskByID[taskID]; exists {
						task.RawData = taskData
						// Re-determine task type for variable/update_variable distinction
						if task.Type == "variable" && isUpdateVariableTask(taskData) {
							task.Type = "update_variable"
						}
						// Extract operator children (switch cases, fork/join branches)
						if childrenArr, ok := taskData["children"].([]interface{}); ok {
							for _, childInterface := range childrenArr {
								if childData, ok := childInterface.(map[string]interface{}); ok {
									task.Children = append(task.Children, childData)
								}
							}
						}
					}
				}
			}
		}

		// Parse workflow outputs
		if outputs, ok := workflowData["outputs"].([]interface{}); ok {
			for _, outputInterface := range outputs {
				if outputData, ok := outputInterface.(map[string]interface{}); ok {
					output := WorkflowOutput{
						Name: getString(outputData, "name"),
					}
					if value, ok := outputData["value"].(map[string]interface{}); ok {
						output.Value = value
					}
					automation.Outputs = append(automation.Outputs, output)
				}
			}
		}

		logger.Debug("fetched full automation details", "automationId", automation.ID, "taskCount", len(automation.Tasks))

		// Fetch available refs to populate FieldRefs for value reference resolution
		// Reuse the existing fetchAutomationRefs function from app.go
		fetchAutomationRefs(ctx, c, aspectID, automation)
	}

	return nil
}

// extractCurrentWorkflowData extracts workflow data from automation.current response
func extractCurrentWorkflowData(result map[string]interface{}) map[string]interface{} {
	org, ok := result["organization"].(map[string]interface{})
	if !ok {
		return nil
	}
	aspect, ok := org["aspect"].(map[string]interface{})
	if !ok {
		return nil
	}
	automation, ok := aspect["automation"].(map[string]interface{})
	if !ok {
		return nil
	}
	current, ok := automation["current"].(map[string]interface{})
	if !ok {
		return nil
	}
	return current
}
