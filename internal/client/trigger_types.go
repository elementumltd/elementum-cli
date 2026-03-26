// Copyright (c) ElementumAI, Inc.
// SPDX-License-Identifier: MPL-2.0

package client

import (
	"strings"
)

// TriggerTypeConfig defines the configuration for a specific trigger type.
// Each trigger type has its own GraphQL fragment and field mappings.
// This is shared between the provider and CLI for consistent query definitions.
type TriggerTypeConfig struct {
	// TypeName is the terraform resource suffix (e.g., "record_created" -> "elementum_record_created_trigger")
	TypeName string
	// GraphQLTypename is the __typename from GraphQL (e.g., "WorkflowRecordCreateTrigger")
	GraphQLTypename string
	// GraphQLFragment defines what fields to query for this trigger type
	GraphQLFragment string
	// Fields lists the trigger-specific fields (not including base fields like id)
	Fields []TriggerFieldConfig
	// IsRecordBased indicates this trigger fires on record events
	IsRecordBased bool
}

// TriggerFieldConfig defines a field on a trigger type.
type TriggerFieldConfig struct {
	// Name is the terraform attribute name (e.g., "datamine_id")
	Name string
	// GraphQLPath is the path in the GraphQL response (e.g., "datamine.id")
	GraphQLPath string
	// Required indicates if the field is required in terraform
	Required bool
	// IsReference indicates this field should be resolved to a terraform reference
	IsReference bool
	// ReferenceType is the type of reference (e.g., "datamine", "email_alias")
	ReferenceType string
	// IsComputed indicates this field is computed by the API
	IsComputed bool
	// IsArray indicates this field is an array of values (e.g., changed_fields)
	IsArray bool
	// ArrayElementPath is the path to extract from each array element (e.g., "id" for changedFields[].id)
	ArrayElementPath string
}

// TriggerTypeRegistry holds all trigger type configurations.
// Field names and types verified against schema.graphql.
// This registry is the single source of truth for trigger GraphQL fragments
// used by both the provider and CLI.
var TriggerTypeRegistry = map[string]TriggerTypeConfig{
	"record_created": {
		TypeName:        "record_created",
		GraphQLTypename: "WorkflowRecordCreateTrigger",
		GraphQLFragment: `
			... on WorkflowRecordCreateTrigger {
				aspect { id name }
				filter
			}
		`,
		Fields: []TriggerFieldConfig{
			{Name: "filter", GraphQLPath: "filter", Required: false},
		},
		IsRecordBased: true,
	},
	"record_updated": {
		TypeName:        "record_updated",
		GraphQLTypename: "WorkflowRecordUpdateTrigger",
		GraphQLFragment: `
			... on WorkflowRecordUpdateTrigger {
				aspect { id name }
				filter
				changedFields { id name }
			}
		`,
		Fields: []TriggerFieldConfig{
			{Name: "filter", GraphQLPath: "filter", Required: false},
			{Name: "changed_fields", GraphQLPath: "changedFields", Required: false, IsArray: true, ArrayElementPath: "id"},
		},
		IsRecordBased: true,
	},
	"attachment_added": {
		TypeName:        "attachment_added",
		GraphQLTypename: "WorkflowAttachmentTrigger",
		GraphQLFragment: `
			... on WorkflowAttachmentTrigger {
				aspect { id name }
			}
		`,
		Fields:        []TriggerFieldConfig{},
		IsRecordBased: true,
	},
	"comment_added": {
		TypeName:        "comment_added",
		GraphQLTypename: "WorkflowCommentAddedTrigger",
		GraphQLFragment: `
			... on WorkflowCommentAddedTrigger {
				aspect { id name }
				commentConversationType: conversationType
			}
		`,
		Fields: []TriggerFieldConfig{
			{Name: "conversation_type", GraphQLPath: "commentConversationType", Required: false},
		},
		IsRecordBased: true,
	},
	"webhook": {
		TypeName:        "webhook",
		GraphQLTypename: "WorkflowWebhookTrigger",
		GraphQLFragment: `
			... on WorkflowWebhookTrigger {
				authenticated
				url
			}
		`,
		Fields: []TriggerFieldConfig{
			{Name: "authenticated", GraphQLPath: "authenticated", Required: false},
			{Name: "webhook_url", GraphQLPath: "url", Required: false, IsComputed: true},
		},
		IsRecordBased: false,
	},
	"on_demand": {
		TypeName:        "on_demand",
		GraphQLTypename: "WorkflowOnDemandTrigger",
		GraphQLFragment: `
			... on WorkflowOnDemandTrigger {
				showTriggeredBy
				parameters {
					id
					name
					fieldType
					required
					multiple
					defaultValue
				}
			}
		`,
		Fields: []TriggerFieldConfig{
			{Name: "show_triggered_by", GraphQLPath: "showTriggeredBy", Required: false},
			{Name: "parameters", GraphQLPath: "parameters", Required: false},
		},
		IsRecordBased: false,
	},
	"slack_message": {
		TypeName:        "slack_message",
		GraphQLTypename: "WorkflowSlackMessageTrigger",
		GraphQLFragment: `
			... on WorkflowSlackMessageTrigger {
				aspect { id name }
			}
		`,
		Fields:        []TriggerFieldConfig{},
		IsRecordBased: true,
	},
	"email_ingestion": {
		TypeName:        "email_ingestion",
		GraphQLTypename: "WorkflowEmailIngestionTrigger",
		GraphQLFragment: `
			... on WorkflowEmailIngestionTrigger {
				emailAlias { id alias }
			}
		`,
		Fields: []TriggerFieldConfig{
			{Name: "email_alias_id", GraphQLPath: "emailAlias.id", Required: true, IsReference: true, ReferenceType: "email_alias"},
		},
		IsRecordBased: false,
	},
	"agent_conversation_ended": {
		TypeName:        "agent_conversation_ended",
		GraphQLTypename: "WorkflowConversationEndTrigger",
		GraphQLFragment: `
			... on WorkflowConversationEndTrigger {
				aspect { id name }
				agentConversationType: conversationType
			}
		`,
		Fields: []TriggerFieldConfig{
			{Name: "conversation_type", GraphQLPath: "agentConversationType", Required: true},
		},
		IsRecordBased: true,
	},
	"approval_chain": {
		TypeName:        "approval_chain",
		GraphQLTypename: "WorkflowApprovalChainTrigger",
		GraphQLFragment: `
			... on WorkflowApprovalChainTrigger {
				aspect { id name }
				approvalChainTemplate { id name }
				status
			}
		`,
		Fields: []TriggerFieldConfig{
			{Name: "approval_chain_template_id", GraphQLPath: "approvalChainTemplate.id", Required: true, IsReference: true, ReferenceType: "approval_template"},
			{Name: "status", GraphQLPath: "status", Required: true},
		},
		IsRecordBased: false,
	},
	"datamine": {
		TypeName:        "datamine",
		GraphQLTypename: "WorkflowDatamineTrigger",
		GraphQLFragment: `
			... on WorkflowDatamineTrigger {
				datamine { id name }
				fireOnAlert
				fireOnRecovery
			}
		`,
		Fields: []TriggerFieldConfig{
			{Name: "datamine_id", GraphQLPath: "datamine.id", Required: true, IsReference: true, ReferenceType: "datamine"},
			{Name: "fire_on_alert", GraphQLPath: "fireOnAlert", Required: false},
			{Name: "fire_on_recovery", GraphQLPath: "fireOnRecovery", Required: false},
		},
		IsRecordBased: false,
	},
}

// GetTriggerTypeConfig returns the configuration for a trigger type.
func GetTriggerTypeConfig(triggerType string) (TriggerTypeConfig, bool) {
	config, ok := TriggerTypeRegistry[triggerType]
	return config, ok
}

// GetAllTriggerGraphQLFragments returns all trigger type fragments for queries.
func GetAllTriggerGraphQLFragments() string {
	var fragments []string
	for _, config := range TriggerTypeRegistry {
		if config.GraphQLFragment != "" {
			fragments = append(fragments, config.GraphQLFragment)
		}
	}
	return strings.Join(fragments, "\n")
}

// BuildTriggersQueryFragment returns the triggers field with all type-specific fragments.
// This can be embedded in workflow queries to get complete trigger data.
func BuildTriggersQueryFragment() string {
	fragments := GetAllTriggerGraphQLFragments()
	return `
		triggers {
			id
			__typename
			` + fragments + `
		}
	`
}

// IsRecordBasedTrigger returns true if the trigger type fires on record events.
func IsRecordBasedTrigger(triggerType string) bool {
	if config, ok := TriggerTypeRegistry[triggerType]; ok {
		return config.IsRecordBased
	}
	return false
}

// MapTriggerTypename maps a GraphQL __typename to the terraform trigger type name.
func MapTriggerTypename(typename string) string {
	for typeName, config := range TriggerTypeRegistry {
		if config.GraphQLTypename == typename {
			return typeName
		}
	}
	return "unknown"
}
