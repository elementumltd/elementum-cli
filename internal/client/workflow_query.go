// Copyright (c) ElementumAI, Inc.
// SPDX-License-Identifier: MPL-2.0

package client

import (
	"context"
	"encoding/json"
	"fmt"
)

// WorkflowFullDetails contains the complete details of a workflow including
// all task type-specific fields. This is the shared type used by both the
// provider and CLI for reading workflow/automation details.
type WorkflowFullDetails struct {
	ID       string               `json:"id"`
	Name     string               `json:"name"`
	Version  int                  `json:"version"`
	Status   string               `json:"status"`
	Terminal bool                 `json:"terminal"`
	Triggers []WorkflowTriggerRaw `json:"triggers"`
	Tasks    []WorkflowTaskRaw    `json:"tasks"`
	Outputs  []WorkflowOutputRaw  `json:"outputs"`
}

// WorkflowTriggerRaw represents a trigger with raw data including type-specific fields.
type WorkflowTriggerRaw struct {
	ID       string                 `json:"id"`
	Typename string                 `json:"__typename"`
	RawData  map[string]interface{} `json:"-"`
}

// WorkflowTaskRaw represents a task with raw data including type-specific fields.
type WorkflowTaskRaw struct {
	ID       string                 `json:"id"`
	Typename string                 `json:"__typename"`
	Name     string                 `json:"name"`
	Previous *WorkflowTaskRef       `json:"previous"`
	Next     *WorkflowTaskRef       `json:"next"`
	RawData  map[string]interface{} `json:"-"`
}

// WorkflowTaskRef is a reference to another task (previous/next chain).
type WorkflowTaskRef struct {
	ID string `json:"id"`
}

// WorkflowOutputRaw represents a workflow output definition.
type WorkflowOutputRaw struct {
	Name string `json:"name"`
}

// UnmarshalJSON implements custom unmarshaling to capture raw task data.
func (t *WorkflowTaskRaw) UnmarshalJSON(data []byte) error {
	if err := json.Unmarshal(data, &t.RawData); err != nil {
		return err
	}

	if id, ok := t.RawData["id"].(string); ok {
		t.ID = id
	}
	if typename, ok := t.RawData["__typename"].(string); ok {
		t.Typename = typename
	}
	if name, ok := t.RawData["name"].(string); ok {
		t.Name = name
	}
	if prev, ok := t.RawData["previous"].(map[string]interface{}); ok {
		if prevID, ok := prev["id"].(string); ok {
			t.Previous = &WorkflowTaskRef{ID: prevID}
		}
	}
	if next, ok := t.RawData["next"].(map[string]interface{}); ok {
		if nextID, ok := next["id"].(string); ok {
			t.Next = &WorkflowTaskRef{ID: nextID}
		}
	}

	return nil
}

// UnmarshalJSON implements custom unmarshaling to capture raw trigger data.
func (t *WorkflowTriggerRaw) UnmarshalJSON(data []byte) error {
	if err := json.Unmarshal(data, &t.RawData); err != nil {
		return err
	}

	if id, ok := t.RawData["id"].(string); ok {
		t.ID = id
	}
	if typename, ok := t.RawData["__typename"].(string); ok {
		t.Typename = typename
	}

	return nil
}

// GetWorkflowWithFullTasks fetches complete workflow details including all task
// type-specific fields using the task and trigger registries. This is the shared
// query method used by both provider and CLI.
//
// Uses aspect.workflow(id) path for proper permission handling.
func (c *Client) GetWorkflowWithFullTasks(ctx context.Context, aspectID, workflowID string) (*WorkflowFullDetails, error) {
	query := BuildFullWorkflowQuery()

	variables := map[string]interface{}{
		"aspectId":   aspectID,
		"workflowId": workflowID,
	}

	var result struct {
		Organization struct {
			Aspect struct {
				Workflow *WorkflowFullDetails `json:"workflow"`
			} `json:"aspect"`
		} `json:"organization"`
	}

	if err := c.ExecuteInto(ctx, query, variables, &result); err != nil {
		return nil, fmt.Errorf("failed to query workflow details: %w", err)
	}

	if result.Organization.Aspect.Workflow == nil {
		return nil, fmt.Errorf("workflow not found: %s", workflowID)
	}

	return result.Organization.Aspect.Workflow, nil
}

// GetWorkflowWithFullTasksByWorkflowID fetches complete workflow details using
// the organization.workflow(id) path. This bypasses the aspect requirement but
// may bypass permission checks in some configurations.
func (c *Client) GetWorkflowWithFullTasksByWorkflowID(ctx context.Context, workflowID string) (*WorkflowFullDetails, error) {
	query := BuildFullWorkflowByIDQuery()

	variables := map[string]interface{}{
		"workflowId": workflowID,
	}

	var result struct {
		Organization struct {
			Workflow *WorkflowFullDetails `json:"workflow"`
		} `json:"organization"`
	}

	if err := c.ExecuteInto(ctx, query, variables, &result); err != nil {
		return nil, fmt.Errorf("failed to query workflow details: %w", err)
	}

	if result.Organization.Workflow == nil {
		return nil, fmt.Errorf("workflow not found: %s", workflowID)
	}

	return result.Organization.Workflow, nil
}

// BuildFullWorkflowQuery builds a query to fetch a single workflow's full details
// via the aspect path (organization.aspect.workflow). Includes comprehensive
// fragments for all task and trigger types from the registries.
func BuildFullWorkflowQuery() string {
	triggerFragments := GetAllTriggerGraphQLFragments()
	taskFragments := GetAllTaskGraphQLFragments()

	return `
		query GetWorkflowFullDetails($aspectId: ID!, $workflowId: ID!) {
			organization {
				aspect(id: $aspectId) {
					workflow(id: $workflowId) {
						id
						name
						version
						status
						terminal
						triggers {
							id
							__typename
							` + triggerFragments + `
						}
						tasks {
							id
							__typename
							name
							previous { id }
							next { id }
							` + taskFragments + `
						}
						outputs {
							name
						}
					}
				}
			}
		}
	`
}

// BuildFullWorkflowByIDQuery builds a query to fetch a workflow's full details
// via the organization.workflow(id) path. Use when aspect ID is not available.
func BuildFullWorkflowByIDQuery() string {
	triggerFragments := GetAllTriggerGraphQLFragments()
	taskFragments := GetAllTaskGraphQLFragments()

	return `
		query GetWorkflowFullDetailsByID($workflowId: ID!) {
			organization {
				workflow(id: $workflowId) {
					id
					name
					version
					status
					terminal
					triggers {
						id
						__typename
						` + triggerFragments + `
					}
					tasks {
						id
						__typename
						name
						previous { id }
						next { id }
						` + taskFragments + `
					}
					outputs {
						name
					}
				}
			}
		}
	`
}

// GetTaskTypeLabel returns a human-readable label for a task __typename.
func GetTaskTypeLabel(typename string) string {
	tfType := MapTaskTypename(typename)
	if tfType != "unknown" {
		return tfType
	}
	return typename
}

// GetTriggerTypeLabel returns a human-readable label for a trigger __typename.
func GetTriggerTypeLabel(typename string) string {
	tfType := MapTriggerTypename(typename)
	if tfType != "unknown" {
		return tfType
	}
	return typename
}
