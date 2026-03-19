// Copyright (c) ElementumAI, Inc.
// SPDX-License-Identifier: MPL-2.0

package discovery

import (
	"context"
	"fmt"

	"github.com/elementumltd/elementum-cli/logger"
	"github.com/elementumltd/elementum-cli/internal/client"
)

// GetTaskByNamespace retrieves a task by its namespace
func GetTaskByNamespace(ctx context.Context, c *client.Client, namespace string) (*AspectTask, error) {
	// List all aspects and find task by namespace
	result, err := client.SearchAspects(ctx, c.Genqlient(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to search for task: %w", err)
	}

	// Extract aspects using helper
	aspects := client.ExtractAspects(result)

	// Find task with matching namespace
	for _, aspect := range aspects {
		if aspect.Typename == "AspectTask" && aspect.Namespace == namespace {
			// Found it, now get full details
			return GetTask(ctx, c, aspect.ID)
		}
	}

	return nil, fmt.Errorf("task not found with namespace: %s", namespace)
}

// GetTaskByName retrieves a task by its name
func GetTaskByName(ctx context.Context, c *client.Client, name string) (*AspectTask, error) {
	// List all aspects and find task by name
	result, err := client.SearchAspects(ctx, c.Genqlient(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to search for task: %w", err)
	}

	// Extract aspects using helper
	aspects := client.ExtractAspects(result)

	// Find task with matching name
	for _, aspect := range aspects {
		if aspect.Typename == "AspectTask" && aspect.Name == name {
			// Found it, now get full details
			return GetTask(ctx, c, aspect.ID)
		}
	}

	return nil, fmt.Errorf("task not found with name: %s", name)
}

// GetTask retrieves detailed information about a task using genqlient.
func GetTask(ctx context.Context, c *client.Client, taskID string) (*AspectTask, error) {
	// Use genqlient GetTask function
	result, err := client.GetTask(ctx, c.Genqlient(), taskID)
	if err != nil {
		return nil, fmt.Errorf("failed to get task: %w", err)
	}

	if result.Organization.Aspect == nil {
		return nil, fmt.Errorf("task not found: %s", taskID)
	}

	// Type assert to get the AspectTask type
	aspectTask, ok := (*result.Organization.Aspect).(*client.GetTaskOrganizationAspectAspectTask)
	if !ok {
		return nil, fmt.Errorf("aspect %s is not a task", taskID)
	}

	task := &AspectTask{
		ID:        aspectTask.Id,
		Name:      aspectTask.Name,
		Handle:    aspectTask.Handle,
		Namespace: aspectTask.Namespace,
	}

	// Handle optional fields
	if aspectTask.Description != nil {
		task.Description = *aspectTask.Description
	}
	if aspectTask.Icon != nil {
		task.Icon = *aspectTask.Icon
	}
	if aspectTask.Color != nil {
		task.Color = *aspectTask.Color
	}
	task.CategoryID = aspectTask.Category.Id
	task.CategoryName = aspectTask.Category.Name

	// Check comments status
	task.CommentsEnabled = checkAspectCommentsEnabled(ctx, c, taskID)
	logger.Debug("checked comments status", "taskID", taskID, "commentsEnabled", task.CommentsEnabled)

	// Get access policies using common function
	policies, err := getAspectAccessPolicies(ctx, c, taskID)
	if err != nil {
		logger.Warn("failed to get task access policies (optional)", "taskID", taskID, "error", err)
	} else {
		task.AccessPolicies = policies
	}
	logger.Debug("checked for access policies", "taskID", taskID, "accessPolicyCount", len(task.AccessPolicies))

	// Get roles using common function
	roles, err := getAspectRoles(ctx, c, taskID)
	if err != nil {
		logger.Warn("failed to get task roles (optional)", "taskID", taskID, "error", err)
	} else {
		task.Roles = roles
	}
	logger.Debug("checked for roles", "taskID", taskID, "roleCount", len(task.Roles))

	// Get views for this task
	views, err := getAspectViews(ctx, c, taskID)
	if err != nil {
		logger.Warn("failed to get task views (optional)", "taskID", taskID, "error", err)
	} else {
		task.Views = views
	}
	logger.Debug("checked for views", "taskID", taskID, "viewCount", len(task.Views))

	return task, nil
}

// ListTasks returns all tasks in the organization
func ListTasks(ctx context.Context, c *client.Client) ([]AspectTask, error) {
	result, err := client.SearchAspects(ctx, c.Genqlient(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list tasks: %w", err)
	}

	// Extract aspects using helper
	aspects := client.ExtractAspects(result)

	tasks := []AspectTask{}
	for _, aspect := range aspects {
		if aspect.Typename == "AspectTask" {
			// Get full details
			task, err := GetTask(ctx, c, aspect.ID)
			if err != nil {
				return nil, err
			}
			tasks = append(tasks, *task)
		}
	}

	return tasks, nil
}

// GetTaskFull retrieves a task with all its resources including fields, layouts, automations, views, agents, etc.
// This is used for full task export - similar to GetApp but for tasks.
// Note: Tasks do NOT support Flows, Approvals, or FileReaders (only Apps do).
func GetTaskFull(ctx context.Context, c *client.Client, taskID string) (*AspectTask, error) {
	// Start with basic task info
	task, err := GetTask(ctx, c, taskID)
	if err != nil {
		return nil, err
	}
	logger.Debug("got basic task info", "taskID", task.ID, "name", task.Name, "namespace", task.Namespace)

	// Get fields
	if err := getTaskFields(ctx, c, task); err != nil {
		logger.Warn("failed to get task fields (optional)", "taskID", taskID, "error", err)
	}
	logger.Debug("got task fields", "fieldCount", len(task.Fields))

	// Get layouts
	if err := getTaskLayouts(ctx, c, task); err != nil {
		logger.Warn("failed to get task layouts (optional)", "taskID", taskID, "error", err)
	}
	logger.Debug("got task layouts", "layoutCount", len(task.Layouts))

	// Get automations for this task using the generic function
	automations, err := GetAspectAutomations(ctx, c, taskID)
	if err != nil {
		logger.Warn("failed to get task automations (optional)", "taskID", taskID, "error", err)
	} else {
		task.Automations = automations
	}
	logger.Debug("got task automations", "automationCount", len(task.Automations))

	// Get agents (tasks support agents unlike elements)
	if err := getTaskAgents(ctx, c, task); err != nil {
		logger.Warn("failed to get task agents (optional)", "taskID", taskID, "error", err)
	}
	logger.Debug("got task agents", "agentCount", len(task.Agents))

	// Get widgets
	if err := getTaskWidgets(ctx, c, task); err != nil {
		logger.Warn("failed to get task widgets (optional)", "taskID", taskID, "error", err)
	}
	logger.Debug("got task widgets", "widgetCount", len(task.Widgets))

	// Get views
	views, err := getAspectViews(ctx, c, taskID)
	if err != nil {
		logger.Warn("failed to get task views (optional)", "taskID", taskID, "error", err)
	} else {
		task.Views = views
	}
	logger.Debug("got task views", "viewCount", len(task.Views))

	// Get relationships
	relationships, err := getObjectRelationships(ctx, c, taskID)
	if err != nil {
		logger.Warn("failed to get task relationships (optional)", "taskID", taskID, "error", err)
	} else {
		task.Relationships = relationships
	}
	logger.Debug("got task relationships", "relationshipCount", len(task.Relationships))

	return task, nil
}

// getTaskFields fetches fields for a task using the common aspect fields query
func getTaskFields(ctx context.Context, c *client.Client, task *AspectTask) error {
	query := client.GetAspectFieldsQuery

	var result struct {
		Organization struct {
			Aspect *struct {
				Fields struct {
					Edges []struct {
						Node struct {
							ID           string   `json:"id"`
							Name         string   `json:"name"`
							Typename     string   `json:"__typename"`
							Required     bool     `json:"required"`
							System       bool     `json:"system"`
							SemanticTags []string `json:"semanticTags"`
							Calculation  *struct {
								Text string `json:"text"`
							} `json:"calculation"`
							Values *struct {
								Edges []struct {
									Node struct {
										ID    string `json:"id"`
										Label string `json:"label"`
										Color string `json:"color"`
									} `json:"node"`
								} `json:"edges"`
							} `json:"values"`
						} `json:"node"`
					} `json:"edges"`
				} `json:"fields"`
			} `json:"aspect"`
		} `json:"organization"`
	}

	if err := c.ExecuteInto(ctx, query, map[string]interface{}{"aspectId": task.ID}, &result); err != nil {
		return err
	}

	if result.Organization.Aspect == nil {
		return nil
	}

	task.Fields = make([]Field, 0, len(result.Organization.Aspect.Fields.Edges))
	for _, edge := range result.Organization.Aspect.Fields.Edges {
		node := edge.Node
		field := Field{
			ID:           node.ID,
			Name:         node.Name,
			Type:         mapFieldType(node.Typename),
			Required:     node.Required,
			System:       node.System,
			SemanticTags: node.SemanticTags,
		}

		// Add calculation text for calculated fields
		if node.Calculation != nil {
			field.Calculation = node.Calculation.Text
		}

		// Add options for dropdown/multiselect fields
		if node.Values != nil {
			field.Options = make([]FieldOption, 0, len(node.Values.Edges))
			for _, optEdge := range node.Values.Edges {
				field.Options = append(field.Options, FieldOption{
					ID:    optEdge.Node.ID,
					Label: optEdge.Node.Label,
					Color: optEdge.Node.Color,
				})
			}
		}

		task.Fields = append(task.Fields, field)
	}

	return nil
}

// getTaskLayouts retrieves all layouts (stages) for a task
// Note: Tasks may have a simpler layout structure than apps
func getTaskLayouts(ctx context.Context, c *client.Client, task *AspectTask) error {
	// Tasks typically have a single "Initiate" layout like elements
	// The GraphQL type for AspectTask may not expose Stages the same way as AspectApp
	// For now, we'll skip layout fetching for tasks and rely on the default Initiate layout
	task.Layouts = []Layout{}
	return nil
}

// getTaskAgents retrieves all agents for a task
// Note: Tasks support agents but the GraphQL query may have limited fields
func getTaskAgents(ctx context.Context, c *client.Client, task *AspectTask) error {
	// The GetAspectAgents query may not include AspectTask in its fragments
	// For now, we'll skip agent fetching for tasks
	// TODO: Add proper agent fetching for tasks when the GraphQL query supports it
	task.Agents = []Agent{}
	return nil
}

// getTaskWidgets retrieves all widgets for a task
// Note: Widgets are optional and may not be exposed for tasks in the GraphQL query
func getTaskWidgets(ctx context.Context, c *client.Client, task *AspectTask) error {
	task.Widgets = []Widget{}
	return nil
}
