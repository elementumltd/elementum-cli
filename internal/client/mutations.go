// Copyright (c) ElementumAI, Inc.
// SPDX-License-Identifier: MPL-2.0

package client

import (
	"context"
	"encoding/json"
	"fmt"
)

// =============================================================================
// Task Mutation Helpers (genqlient-backed)
// =============================================================================
// These helpers accept map[string]interface{} inputs for backward compatibility
// with the existing task resource builders, but route through genqlient for
// type-safe GraphQL execution.
//
// The map inputs are JSON-marshaled and unmarshaled into genqlient typed structs,
// ensuring consistency with the GraphQL schema while preserving the dynamic
// input building pattern used by all 30+ task types.
// =============================================================================

// mapToGenqlientTaskInput converts a map-based task input to a genqlient WorkflowTaskInput.
func mapToGenqlientTaskInput(taskInput map[string]interface{}) (WorkflowTaskInput, error) {
	data, err := json.Marshal(taskInput)
	if err != nil {
		return WorkflowTaskInput{}, fmt.Errorf("failed to marshal task input: %w", err)
	}

	var typed WorkflowTaskInput
	if err := json.Unmarshal(data, &typed); err != nil {
		return WorkflowTaskInput{}, fmt.Errorf("failed to convert task input to genqlient type: %w", err)
	}

	return typed, nil
}

// mapToGenqlientTaskUpdateInput converts a map-based task update input to a genqlient WorkflowTaskUpdateInput.
func mapToGenqlientTaskUpdateInput(taskInput map[string]interface{}) (WorkflowTaskUpdateInput, error) {
	data, err := json.Marshal(taskInput)
	if err != nil {
		return WorkflowTaskUpdateInput{}, fmt.Errorf("failed to marshal task update input: %w", err)
	}
	var typed WorkflowTaskUpdateInput
	if err := json.Unmarshal(data, &typed); err != nil {
		return WorkflowTaskUpdateInput{}, fmt.Errorf("failed to convert task update input to genqlient type: %w", err)
	}
	return typed, nil
}

// mapToGenqlientOperatorUpdateInput converts a map-based operator update input to a genqlient WorkflowTaskOperatorUpdateInput.
func mapToGenqlientOperatorUpdateInput(input map[string]interface{}) (WorkflowTaskOperatorUpdateInput, error) {
	data, err := json.Marshal(input)
	if err != nil {
		return WorkflowTaskOperatorUpdateInput{}, fmt.Errorf("failed to marshal operator update input: %w", err)
	}
	var typed WorkflowTaskOperatorUpdateInput
	if err := json.Unmarshal(data, &typed); err != nil {
		return WorkflowTaskOperatorUpdateInput{}, fmt.Errorf("failed to convert operator update input to genqlient type: %w", err)
	}
	return typed, nil
}

// mapToGenqlientOperatorInput converts a map-based operator input to a genqlient WorkflowTaskOperatorInput.
func mapToGenqlientOperatorInput(input map[string]interface{}) (WorkflowTaskOperatorInput, error) {
	data, err := json.Marshal(input)
	if err != nil {
		return WorkflowTaskOperatorInput{}, fmt.Errorf("failed to marshal operator input: %w", err)
	}
	var typed WorkflowTaskOperatorInput
	if err := json.Unmarshal(data, &typed); err != nil {
		return WorkflowTaskOperatorInput{}, fmt.Errorf("failed to convert operator input to genqlient type: %w", err)
	}
	return typed, nil
}

// CreateWorkflowTask creates the first (root) task in a workflow.
func (c *Client) CreateWorkflowTask(ctx context.Context, workflowID string, taskInput map[string]interface{}) (string, error) {
	typedInput, err := mapToGenqlientTaskInput(taskInput)
	if err != nil {
		return "", err
	}

	result, err := CreateWorkflowTask(ctx, c.Genqlient(), workflowID, typedInput)
	if err != nil {
		return "", err
	}

	return result.WorkflowTaskCreate.GetId(), nil
}

// InsertWorkflowTask inserts a task into a workflow after an existing task.
func (c *Client) InsertWorkflowTask(ctx context.Context, workflowID string, taskInput map[string]interface{}) (string, error) {
	typedInput, err := mapToGenqlientTaskInput(taskInput)
	if err != nil {
		return "", err
	}

	result, err := InsertWorkflowTask(ctx, c.Genqlient(), workflowID, typedInput)
	if err != nil {
		return "", err
	}

	return result.WorkflowTaskInsert.GetId(), nil
}

// UpdateWorkflowTask updates an existing task in a workflow.
func (c *Client) UpdateWorkflowTask(ctx context.Context, taskID string, taskInput map[string]interface{}) error {
	typedInput, err := mapToGenqlientTaskUpdateInput(taskInput)
	if err != nil {
		return err
	}

	_, err = UpdateWorkflowTask(ctx, c.Genqlient(), taskID, typedInput)
	return err
}

// CreateWorkflowTaskChild creates a task inside an operator (for_each/switch).
func (c *Client) CreateWorkflowTaskChild(ctx context.Context, operatorTaskID string, taskInput map[string]interface{}) (string, error) {
	typedInput, err := mapToGenqlientTaskInput(taskInput)
	if err != nil {
		return "", err
	}

	result, err := CreateWorkflowTaskChild(ctx, c.Genqlient(), operatorTaskID, typedInput)
	if err != nil {
		return "", err
	}

	return result.WorkflowTaskChildCreate.GetId(), nil
}

// CreateWorkflowTaskOperatorChild creates a case/branch for a switch or fork/join operator.
func (c *Client) CreateWorkflowTaskOperatorChild(ctx context.Context, operatorTaskID string, input map[string]interface{}) (string, error) {
	typedInput, err := mapToGenqlientOperatorInput(input)
	if err != nil {
		return "", err
	}

	result, err := CreateWorkflowTaskOperatorChild(ctx, c.Genqlient(), operatorTaskID, typedInput)
	if err != nil {
		return "", err
	}

	return result.WorkflowTaskOperatorChildCreate.Id, nil
}

// UpdateWorkflowTaskOperatorChild updates a branch/case on a switch or fork/join operator.
func (c *Client) UpdateWorkflowTaskOperatorChild(ctx context.Context, operatorChildID string, operatorTaskID string, input map[string]interface{}) error {
	typedInput, err := mapToGenqlientOperatorUpdateInput(input)
	if err != nil {
		return err
	}

	_, err = UpdateWorkflowTaskOperatorChild(ctx, c.Genqlient(), operatorChildID, operatorTaskID, typedInput)
	return err
}

// DeleteWorkflowTaskOperatorChild deletes a branch/case from a fork/join or switch operator.
func (c *Client) DeleteWorkflowTaskOperatorChild(ctx context.Context, operatorChildID string) error {
	_, err := DeleteWorkflowTaskOperatorChild(ctx, c.Genqlient(), operatorChildID)
	return err
}

// DeleteWorkflowTaskByID deletes a task from a workflow.
func (c *Client) DeleteWorkflowTaskByID(ctx context.Context, taskID string) error {
	_, err := DeleteWorkflowTask(ctx, c.Genqlient(), taskID)
	return err
}

// =============================================================================
// Legacy Mutation Constants (still used by non-task resources)
// =============================================================================

const (
	// CreateAspectMutation creates a new aspect (app, element, task).
	CreateAspectMutation = `
		mutation CreateAspect($input: AspectInput!) {
			aspectCreate(input: $input) {
				id
				name
				__typename
				... on AspectApp {
					namespace
				}
			}
		}
	`

	// skillToolFields is the shared inline fragment block for all agentic skill tool types.
	skillToolFields = `
				id
				name
				description
				status
				type
				... on AgenticSkillAutomationTool {
					startMessage
					automation { id }
				}
				... on AgenticSkillCreateRecordTool {
					startMessage
					aspect { id }
					fields { field { id } name description required }
				}
				... on AgenticSkillSearchAspectTool {
					startMessage
					aspect { id }
					queryDescription
					limit
					fields { field { id } name description }
				}
				... on AgenticSkillSearchTableTool {
					startMessage
					aspect { id }
					table { id }
					queryDescription
					limit
					fields { field { id } name description }
				}
				... on AgenticSkillUpdateRecordTool {
					startMessage
					aspect { id }
					handleDescription
					fields { field { id } name description required }
				}
				... on AgenticSkillRunAgentTool {
					startMessage
					targetAgent { id }
					workerTaskPrompt
				}
	`

	// CreateAgenticSkillToolMutation creates a tool on an agentic skill.
	CreateAgenticSkillToolMutation = `
		mutation CreateAgenticSkillTool($skillId: ID!, $input: AgenticSkillToolCreateInput!) {
			agenticSkillToolCreate(skillId: $skillId, input: $input) {` + skillToolFields + `
			}
		}
	`

	// UpdateAgenticSkillToolMutation updates a tool on an agentic skill.
	UpdateAgenticSkillToolMutation = `
		mutation UpdateAgenticSkillTool($id: ID!, $input: AgenticSkillToolUpdateInput!) {
			agenticSkillToolUpdate(id: $id, input: $input) {` + skillToolFields + `
			}
		}
	`

	// DeleteAgenticSkillToolMutation deletes a tool from an agentic skill.
	DeleteAgenticSkillToolMutation = `
		mutation DeleteAgenticSkillTool($id: ID!) {
			agenticSkillToolDelete(id: $id) {
				id
			}
		}
	`

	// GetAgenticSkillToolsQuery fetches all tools for an agentic skill.
	GetAgenticSkillToolsQuery = `
		query GetAgenticSkillTools($skillId: ID!) {
			organization {
				agenticSkill(id: $skillId) {
					id
					tools {
						edges {
							node {` + skillToolFields + `
							}
						}
					}
				}
			}
		}
	`
)
