// Copyright (c) ElementumAI, Inc.
// SPDX-License-Identifier: MPL-2.0

package client

import (
	"fmt"
	"strings"
)

// TaskTypeConfig defines the configuration for a specific task type.
// Each task type has its own GraphQL fragment and field mappings.
// This is shared between the provider and CLI for consistent query definitions.
type TaskTypeConfig struct {
	// TypeName is the terraform resource suffix (e.g., "switch" -> "elementum_switch_task")
	TypeName string
	// GraphQLTypename is the __typename from GraphQL (e.g., "WorkflowSwitchTask")
	GraphQLTypename string
	// GraphQLFragment defines what fields to query for this task type
	GraphQLFragment string
	// Fields lists the task-specific fields (not including base fields like id, name, previous)
	Fields []TaskFieldConfig
}

// TaskFieldConfig defines a field on a task type.
type TaskFieldConfig struct {
	// Name is the terraform attribute name (e.g., "object_id")
	Name string
	// GraphQLPath is the path in the GraphQL response (e.g., "aspect.id")
	GraphQLPath string
	// Required indicates if the field is required in terraform
	Required bool
	// IsReference indicates this field should be resolved to a terraform reference
	IsReference bool
	// ReferenceType is the type of reference (e.g., "app", "element", "agent")
	ReferenceType string
	// IsValueReference indicates this is a value reference (object with id/label/value)
	IsValueReference bool
	// IsArray indicates this field is an array of simple IDs
	IsArray bool
	// ArrayElementPath is the path to extract from each array element (e.g., "id" for fieldsToLock[].id)
	ArrayElementPath string
	// IsComplexArray indicates this is an array of complex objects (like categories, headers)
	IsComplexArray bool
	// IsUserSearchEmail indicates this field extracts email from a user_search filter structure
	// The API stores user search as a filter, but the provider schema expects a simple email attribute
	IsUserSearchEmail bool
}

// TaskTypeRegistry holds all task type configurations.
// Field names and types verified against schema.graphql.
// This registry is the single source of truth for task GraphQL fragments
// used by both the provider and CLI.
var TaskTypeRegistry = map[string]TaskTypeConfig{
	"switch": {
		TypeName:        "switch",
		GraphQLTypename: "WorkflowSwitchTask",
		GraphQLFragment: `
			... on WorkflowSwitchTask {
				children {
					id
					label
					filter
					tasks {
						id
						__typename
						name
						previous { id }
						next { id }
					}
				}
			}
		`,
		Fields: []TaskFieldConfig{
			// Switch tasks only have base fields (parent_id, name)
			// Cases are separate resources
		},
	},
	"variable": {
		TypeName:        "variable",
		GraphQLTypename: "WorkflowVariableTask",
		GraphQLFragment: `
			... on WorkflowVariableTask {
				variables {
					__typename
					... on WorkflowVariableTaskParameterCreate {
						id
						name
						type
						createValue: value { id label value }
					}
					... on WorkflowVariableTaskParameterUpdate {
						updateValue: value { id label value }
						variable { id label value }
					}
				}
			}
		`,
		Fields: []TaskFieldConfig{
			{Name: "variable_name", GraphQLPath: "variables[0].name", Required: true},
			{Name: "variable_type", GraphQLPath: "variables[0].type", Required: true},
			{Name: "value", GraphQLPath: "variables[0].createValue", Required: true, IsValueReference: true},
		},
	},
	"update_variable": {
		TypeName:        "update_variable",
		GraphQLTypename: "WorkflowVariableTask", // Same as "variable" - distinguished by content
		GraphQLFragment: ``,                     // Empty - shares fragment with "variable"
		Fields: []TaskFieldConfig{
			{Name: "variable_reference", GraphQLPath: "variables[0].variable", Required: true, IsValueReference: true},
			{Name: "value", GraphQLPath: "variables[0].updateValue", Required: true, IsValueReference: true},
		},
	},
	"update_field": {
		TypeName:        "update_field",
		GraphQLTypename: "WorkflowUpdateFieldTask",
		GraphQLFragment: `
			... on WorkflowUpdateFieldTask {
				aspect { id name }
				recordReference { id label value }
				workflowFields {
					field { id name }
					valueReference { id label value }
				}
			}
		`,
		Fields: []TaskFieldConfig{
			{Name: "object_id", GraphQLPath: "aspect.id", Required: false, IsReference: true, ReferenceType: "object"},
			{Name: "record_reference", GraphQLPath: "recordReference", Required: false, IsValueReference: true},
			{Name: "fields", GraphQLPath: "workflowFields", Required: false},
		},
	},
	"create_record": {
		TypeName:        "create_record",
		GraphQLTypename: "WorkflowCreateRecordTask",
		GraphQLFragment: `
			... on WorkflowCreateRecordTask {
				aspect { id name }
				recordReference { id label value }
				relateToRecord
				workflowFields {
					field { id name }
					valueReference { id label value }
				}
			}
		`,
		Fields: []TaskFieldConfig{
			{Name: "object_id", GraphQLPath: "aspect.id", Required: true, IsReference: true, ReferenceType: "object"},
			{Name: "record_reference", GraphQLPath: "recordReference", Required: false},
			{Name: "fields", GraphQLPath: "workflowFields", Required: false},
		},
	},
	"record_search": {
		TypeName:        "record_search",
		GraphQLTypename: "WorkflowRecordSearchTask",
		GraphQLFragment: `
			... on WorkflowRecordSearchTask {
				aspect { id name }
				limit
				filter
				filterValueReferences {
					valueReferences { id label value }
				}
			}
		`,
		Fields: []TaskFieldConfig{
			{Name: "object_id", GraphQLPath: "aspect.id", Required: true, IsReference: true, ReferenceType: "object"},
			{Name: "limit", GraphQLPath: "limit", Required: false},
			{Name: "filter", GraphQLPath: "filter", Required: false},
		},
	},
	"ai_agent": {
		TypeName:        "ai_agent",
		GraphQLTypename: "WorkflowAiAgentTask",
		GraphQLFragment: `
			... on WorkflowAiAgentTask {
				agent { id name }
				agentPrompt { id label value }
				outputType
			}
		`,
		Fields: []TaskFieldConfig{
			{Name: "agent_id", GraphQLPath: "agent.id", Required: true, IsReference: true, ReferenceType: "agent"},
			{Name: "prompt", GraphQLPath: "agentPrompt", Required: true, IsValueReference: true},
		},
	},
	"message": {
		TypeName:        "message",
		GraphQLTypename: "WorkflowMessageTask",
		GraphQLFragment: `
			... on WorkflowMessageTask {
				contentsReference { id label value }
				recordReference { id label value }
			}
		`,
		Fields: []TaskFieldConfig{
			{Name: "message", GraphQLPath: "contentsReference", Required: true, IsValueReference: true},
			{Name: "record_reference", GraphQLPath: "recordReference", Required: false, IsValueReference: true},
		},
	},
	"send_email": {
		TypeName:        "send_email",
		GraphQLTypename: "WorkflowSendEmailTask",
		GraphQLFragment: `
			... on WorkflowSendEmailTask {
				subject { id label value }
				messageBody { id label value }
				emailUsers: users { id label value }
				emailGroups: groups { id label value }
				externalEmails { id label value }
			}
		`,
		Fields: []TaskFieldConfig{
			{Name: "subject", GraphQLPath: "subject", Required: true, IsValueReference: true},
			{Name: "message_body", GraphQLPath: "messageBody", Required: true, IsValueReference: true},
			{Name: "users", GraphQLPath: "emailUsers", Required: false, IsValueReference: true},
		},
	},
	"for_each": {
		TypeName:        "for_each",
		GraphQLTypename: "WorkflowForEachTask",
		GraphQLFragment: `
			... on WorkflowForEachTask {
				forEach { id label value }
				children {
					id
					label
					tasks {
						id
						__typename
						name
						previous { id }
						next { id }
					}
				}
			}
		`,
		Fields: []TaskFieldConfig{
			{Name: "list", GraphQLPath: "forEach", Required: true, IsValueReference: true},
		},
	},
	"save_attachment": {
		TypeName:        "save_attachment",
		GraphQLTypename: "WorkflowSaveAttachmentTask",
		GraphQLFragment: `
			... on WorkflowSaveAttachmentTask {
				recordId { id label value }
				file {
					... on ValueReference { id label value }
				}
				tag { id label value }
			}
		`,
		Fields: []TaskFieldConfig{
			{Name: "record_id", GraphQLPath: "recordId", Required: true, IsValueReference: true},
			{Name: "file", GraphQLPath: "file", Required: true, IsValueReference: true},
			{Name: "tag", GraphQLPath: "tag", Required: false, IsValueReference: true},
		},
	},
	"notification": {
		TypeName:        "notification",
		GraphQLTypename: "WorkflowNotificationTask",
		GraphQLFragment: `
			... on WorkflowNotificationTask {
				messageReference { id label value }
				titleReference { id label value }
				userValueReferences { id label value }
				groupValueReferences { id label value }
				notifyWatchers
			}
		`,
		Fields: []TaskFieldConfig{
			{Name: "message", GraphQLPath: "messageReference", Required: true, IsValueReference: true},
			{Name: "title", GraphQLPath: "titleReference", Required: false, IsValueReference: true},
			{Name: "notify_watchers", GraphQLPath: "notifyWatchers", Required: false},
		},
	},
	"api": {
		TypeName:        "api",
		GraphQLTypename: "WorkflowApiTask",
		GraphQLFragment: `
			... on WorkflowApiTask {
				method
				urlReference { id value triggerReference { name } taskReference { name task { id } } templateReference { template parameters { ... on ValueReferenceTemplateValueParameter { name value { triggerReference { name } taskReference { name task { id } } variableReference { variableId name type } forEachReference { name forEachTask { id } } } } } } }
				authorization {
					__typename
					... on ApiBearerAuthorization {
						token
						referenceToken { id value triggerReference { name } taskReference { name task { id } } templateReference { template parameters { ... on ValueReferenceTemplateValueParameter { name value { triggerReference { name } taskReference { name task { id } } variableReference { variableId name type } forEachReference { name forEachTask { id } } } } } } }
					}
					... on ApiBasicAuthorization {
						username
						password
						referencePassword { id value triggerReference { name } taskReference { name task { id } } templateReference { template parameters { ... on ValueReferenceTemplateValueParameter { name value { triggerReference { name } taskReference { name task { id } } variableReference { variableId name type } forEachReference { name forEachTask { id } } } } } } }
					}
					... on ApiOauthAuthorization {
						clientId
						clientSecret
						url
						requestType
						headers { header value { id value triggerReference { name } taskReference { name task { id } } templateReference { template parameters { ... on ValueReferenceTemplateValueParameter { name value { triggerReference { name } taskReference { name task { id } } variableReference { variableId name type } forEachReference { name forEachTask { id } } } } } } } }
					}
				}
				body {
					__typename
					... on ApiCustomBody { body { id value triggerReference { name } taskReference { name task { id } } templateReference { template parameters { ... on ValueReferenceTemplateValueParameter { name value { triggerReference { name } taskReference { name task { id } } variableReference { variableId name type } forEachReference { name forEachTask { id } } } } } } } contentType }
					... on ApiJsonBody { jsonReference { id value triggerReference { name } taskReference { name task { id } } templateReference { template parameters { ... on ValueReferenceTemplateValueParameter { name value { triggerReference { name } taskReference { name task { id } } variableReference { variableId name type } forEachReference { name forEachTask { id } } } } } } } }
					... on ApiFormBody { values { key value { id value triggerReference { name } taskReference { name task { id } } templateReference { template parameters { ... on ValueReferenceTemplateValueParameter { name value { triggerReference { name } taskReference { name task { id } } variableReference { variableId name type } forEachReference { name forEachTask { id } } } } } } } } }
					... on ApiMultipartFormBody {
						parts {
							... on ApiMultipartFile {
								name
								attachment { id value triggerReference { name } taskReference { name task { id } } templateReference { template parameters { ... on ValueReferenceTemplateValueParameter { name value { triggerReference { name } taskReference { name task { id } } variableReference { variableId name type } forEachReference { name forEachTask { id } } } } } } }
								attachmentId { id value triggerReference { name } taskReference { name task { id } } templateReference { template parameters { ... on ValueReferenceTemplateValueParameter { name value { triggerReference { name } taskReference { name task { id } } variableReference { variableId name type } forEachReference { name forEachTask { id } } } } } } }
								file { id value triggerReference { name } taskReference { name task { id } } templateReference { template parameters { ... on ValueReferenceTemplateValueParameter { name value { triggerReference { name } taskReference { name task { id } } variableReference { variableId name type } forEachReference { name forEachTask { id } } } } } } }
							}
						}
					}
				}
				headers { header value { id value triggerReference { name } taskReference { name task { id } } templateReference { template parameters { ... on ValueReferenceTemplateValueParameter { name value { triggerReference { name } taskReference { name task { id } } variableReference { variableId name type } forEachReference { name forEachTask { id } } } } } } } }
				responseType
				continueOnError
			}
		`,
		Fields: []TaskFieldConfig{
			{Name: "method", GraphQLPath: "method", Required: true},
			{Name: "url", GraphQLPath: "urlReference", Required: true, IsValueReference: true},
			{Name: "authorization", GraphQLPath: "authorization", Required: false},
			{Name: "body", GraphQLPath: "body", Required: false},
			{Name: "headers", GraphQLPath: "headers", Required: false, IsComplexArray: true},
			{Name: "response_type", GraphQLPath: "responseType", Required: false},
			{Name: "continue_on_error", GraphQLPath: "continueOnError", Required: false},
		},
	},
	"ai_file_read": {
		TypeName:        "ai_file_read",
		GraphQLTypename: "WorkflowAiFileAnalysisTask",
		GraphQLFragment: `
			... on WorkflowAiFileAnalysisTask {
				documentModel { id }
				attachment { id label value }
				aiProviderConnector { id model { name } provider { name } }
			}
		`,
		Fields: []TaskFieldConfig{
			{Name: "file_reader_id", GraphQLPath: "documentModel.id", Required: true, IsReference: true, ReferenceType: "file_reader"},
			{Name: "file_id", GraphQLPath: "attachment", Required: true, IsValueReference: true},
			{Name: "ai_provider_connector_id", GraphQLPath: "aiProviderConnector.id", Required: true, IsReference: true, ReferenceType: "ai_provider_connector"},
		},
	},
	"calculation": {
		TypeName:        "calculation",
		GraphQLTypename: "WorkflowCalculationTask",
		GraphQLFragment: `
			... on WorkflowCalculationTask {
				calculations {
					id
					name
					calculation
					calculationReference {
						id
						label
						value
						templateReference {
							template
							parameters {
								... on ValueReferenceTemplateValueParameter {
									name
									value { triggerReference { name } taskReference { name } value }
								}
							}
						}
					}
				}
			}
		`,
		Fields: []TaskFieldConfig{
			{Name: "calculations", GraphQLPath: "calculations", Required: true},
		},
	},
	"add_watcher": {
		TypeName:        "add_watcher",
		GraphQLTypename: "WorkflowAddWatcherTask",
		GraphQLFragment: `
			... on WorkflowAddWatcherTask {
				recordReference { id label value }
				watcherUsers: users { id }
				watcherGroups: groups { id }
			}
		`,
		Fields: []TaskFieldConfig{
			{Name: "record_reference", GraphQLPath: "recordReference", Required: true, IsValueReference: true},
		},
	},
	"relate_records": {
		TypeName:        "relate_records",
		GraphQLTypename: "WorkflowRelateRecordsTask",
		GraphQLFragment: `
			... on WorkflowRelateRecordsTask {
				sourceRecordIds { id label value }
				targetRecordId { id label value }
			}
		`,
		Fields: []TaskFieldConfig{
			{Name: "source_record_ids", GraphQLPath: "sourceRecordIds", Required: true, IsValueReference: true},
			{Name: "target_record_id", GraphQLPath: "targetRecordId", Required: true, IsValueReference: true},
		},
	},
	"find_related_records": {
		TypeName:        "find_related_records",
		GraphQLTypename: "WorkflowFindRelatedRecordsTask",
		GraphQLFragment: `
			... on WorkflowFindRelatedRecordsTask {
				recordId { id label value }
				relatedAspect { id name }
				limit
			}
		`,
		Fields: []TaskFieldConfig{
			{Name: "record_reference", GraphQLPath: "recordId", Required: true, IsValueReference: true},
			{Name: "related_object_id", GraphQLPath: "relatedAspect.id", Required: true, IsReference: true, ReferenceType: "object"},
			{Name: "limit", GraphQLPath: "limit", Required: false},
		},
	},
	"user_search": {
		TypeName:        "user_search",
		GraphQLTypename: "WorkflowUserSearchTask",
		GraphQLFragment: `
			... on WorkflowUserSearchTask {
				filter
				filterValueReferences {
					valueReferences { id label value }
				}
			}
		`,
		Fields: []TaskFieldConfig{
			// The API stores user search as a filter on the email field,
			// but the provider schema expects a simple "email" attribute.
			// IsUserSearchEmail triggers special extraction logic in the export.
			{Name: "email", GraphQLPath: "filter", Required: true, IsUserSearchEmail: true},
		},
	},
	"ai_classify": {
		TypeName:        "ai_classify",
		GraphQLTypename: "WorkflowAiClassifyTask",
		GraphQLFragment: `
			... on WorkflowAiClassifyTask {
				aiProviderConnector { id model { name } provider { name } }
				textToClassify {
					id
					triggerReference { name }
					taskReference { name task { id } }
					templateReference {
						template
						parameters {
							... on ValueReferenceTemplateValueParameter { name value { triggerReference { name } taskReference { name task { id } } variableReference { variableId name type } forEachReference { name forEachTask { id } } } }
						}
					}
				}
				categorySource {
					__typename
					... on AiClassifyCategoryStaticSource {
						categories {
							name {
								id
								triggerReference { name }
								taskReference { name task { id } }
								templateReference {
									template
									parameters {
										... on ValueReferenceTemplateValueParameter { name value { triggerReference { name } taskReference { name task { id } } variableReference { variableId name type } forEachReference { name forEachTask { id } } } }
									}
								}
							}
							description {
								id
								triggerReference { name }
								taskReference { name task { id } }
								templateReference {
									template
									parameters {
										... on ValueReferenceTemplateValueParameter { name value { triggerReference { name } taskReference { name task { id } } variableReference { variableId name type } forEachReference { name forEachTask { id } } } }
									}
								}
							}
						}
					}
					... on AiClassifyCategoryDynamicSource {
						aspect { id name }
						labelField { id name }
						descriptionField { id name }
						filter
						filterValueReferences {
							valueReferences { id label value }
						}
						limit
					}
				}
				context {
					id
					triggerReference { name }
					taskReference { name task { id } }
					templateReference {
						template
						parameters {
							... on ValueReferenceTemplateValueParameter { name value { triggerReference { name } taskReference { name task { id } } variableReference { variableId name type } forEachReference { name forEachTask { id } } } }
						}
					}
				}
			}
		`,
		Fields: []TaskFieldConfig{
			{Name: "ai_provider_connector_id", GraphQLPath: "aiProviderConnector.id", Required: false, IsReference: true, ReferenceType: "ai_provider_connector"},
			{Name: "text_to_classify", GraphQLPath: "textToClassify", Required: true, IsValueReference: true},
			{Name: "category_source", GraphQLPath: "categorySource", Required: true},
			{Name: "context", GraphQLPath: "context", Required: false, IsValueReference: true},
		},
	},
	"ai_summarize": {
		TypeName:        "ai_summarize",
		GraphQLTypename: "WorkflowAiSummarizeTask",
		GraphQLFragment: `
			... on WorkflowAiSummarizeTask {
				aiProviderConnector { id model { name } provider { name } }
				textToSummarize { id label value }
				context { id label value }
				length
			}
		`,
		Fields: []TaskFieldConfig{
			{Name: "ai_provider_connector_id", GraphQLPath: "aiProviderConnector.id", Required: false, IsReference: true, ReferenceType: "ai_provider_connector"},
			{Name: "text_to_summarize", GraphQLPath: "textToSummarize", Required: true, IsValueReference: true},
			{Name: "context", GraphQLPath: "context", Required: false, IsValueReference: true},
			{Name: "length", GraphQLPath: "length", Required: false},
		},
	},
	"ai_transform": {
		TypeName:        "ai_transform",
		GraphQLTypename: "WorkflowAiTransformTask",
		GraphQLFragment: `
			... on WorkflowAiTransformTask {
				aiProviderConnector { id model { name } provider { name } }
				prompt { id label value }
				fieldType
			}
		`,
		Fields: []TaskFieldConfig{
			{Name: "ai_provider_connector_id", GraphQLPath: "aiProviderConnector.id", Required: false, IsReference: true, ReferenceType: "ai_provider_connector"},
			{Name: "prompt", GraphQLPath: "prompt", Required: true, IsValueReference: true},
			{Name: "field_type", GraphQLPath: "fieldType", Required: false},
		},
	},
	"approval_chain": {
		TypeName:        "approval_chain",
		GraphQLTypename: "WorkflowApprovalChainTask",
		GraphQLFragment: `
			... on WorkflowApprovalChainTask {
				approvers { id label value }
				reason { id label value }
			}
		`,
		Fields: []TaskFieldConfig{
			{Name: "approvers", GraphQLPath: "approvers", Required: false, IsValueReference: true},
			{Name: "reason", GraphQLPath: "reason", Required: false, IsValueReference: true},
		},
	},
	"approval_status_update": {
		TypeName:        "approval_status_update",
		GraphQLTypename: "WorkflowApprovalStatusUpdateTask",
		GraphQLFragment: `
			... on WorkflowApprovalStatusUpdateTask {
				approvalChainTemplate { id }
				status
				reason { id label value }
			}
		`,
		Fields: []TaskFieldConfig{
			// Note: API has approvalChainTemplate, but Terraform schema expects approval_reference
			// The schema maps this as a value reference string
			{Name: "approval_reference", GraphQLPath: "approvalChainTemplate.id", Required: true},
			{Name: "status", GraphQLPath: "status", Required: true},
			{Name: "reason", GraphQLPath: "reason", Required: false, IsValueReference: true},
		},
	},
	"aspect_record_field_locking": {
		TypeName:        "aspect_record_field_locking",
		GraphQLTypename: "WorkflowAspectRecordFieldLockingTask",
		GraphQLFragment: `
			... on WorkflowAspectRecordFieldLockingTask {
				aspect { id name }
				recordId { id label value }
				fieldsToLock { id name }
				fieldsToUnlock { id name }
			}
		`,
		Fields: []TaskFieldConfig{
			{Name: "object_id", GraphQLPath: "aspect.id", Required: true, IsReference: true, ReferenceType: "object"},
			{Name: "record_reference", GraphQLPath: "recordId", Required: true, IsValueReference: true},
			{Name: "fields_to_lock", GraphQLPath: "fieldsToLock", Required: false, IsArray: true, ArrayElementPath: "id"},
			{Name: "fields_to_unlock", GraphQLPath: "fieldsToUnlock", Required: false, IsArray: true, ArrayElementPath: "id"},
		},
	},
	"bulk_excel": {
		TypeName:        "bulk_excel",
		GraphQLTypename: "WorkflowBulkExcelTask",
		GraphQLFragment: `
			... on WorkflowBulkExcelTask {
				aspect { id name }
				attachment { id label value }
				documentModel { id }
				strict
			}
		`,
		Fields: []TaskFieldConfig{
			{Name: "object_id", GraphQLPath: "aspect.id", Required: true, IsReference: true, ReferenceType: "object"},
			{Name: "document_model_id", GraphQLPath: "documentModel.id", Required: true, IsReference: true, ReferenceType: "document_model"},
			{Name: "attachment_reference", GraphQLPath: "attachment", Required: true, IsValueReference: true},
			{Name: "strict", GraphQLPath: "strict", Required: false},
		},
	},
	"ai_search_table": {
		TypeName:        "ai_search_table",
		GraphQLTypename: "WorkflowAiSearchTableTask",
		GraphQLFragment: `
			... on WorkflowAiSearchTableTask {
				searchTable { id }
				query { id label value }
				aiSearchFilter: filter
				limit
			}
		`,
		Fields: []TaskFieldConfig{
			{Name: "search_table_id", GraphQLPath: "searchTable.id", Required: true, IsReference: true, ReferenceType: "ai_search_table"},
			{Name: "query", GraphQLPath: "query", Required: true, IsValueReference: true},
			{Name: "limit", GraphQLPath: "limit", Required: false},
		},
	},
	"procedure": {
		TypeName:        "procedure",
		GraphQLTypename: "WorkflowProcedureTask",
		GraphQLFragment: `
			... on WorkflowProcedureTask {
				cloudLinkId
				storedFunction { id name }
				parameters {
					index
					value { id label value }
				}
			}
		`,
		Fields: []TaskFieldConfig{
			{Name: "cloudlink_id", GraphQLPath: "cloudLinkId", Required: true, IsReference: true, ReferenceType: "cloudlink"},
			{Name: "stored_function_id", GraphQLPath: "storedFunction.id", Required: true},
			{Name: "parameters", GraphQLPath: "parameters", Required: false, IsComplexArray: true},
		},
	},
	"run_automation": {
		TypeName:        "run_automation",
		GraphQLTypename: "WorkflowRunAutomationTask",
		GraphQLFragment: `
			... on WorkflowRunAutomationTask {
				automation { id name }
				automationRef { id label value }
				synchronous
				inputMappings {
					parameter { id name }
					value { id label value }
				}
				dynamicInputMappings {
					name
					value { id label value }
				}
				outputMappings { name }
			}
		`,
		Fields: []TaskFieldConfig{
			{Name: "target_automation_id", GraphQLPath: "automation.id", Required: false, IsReference: true, ReferenceType: "automation"},
			{Name: "target_automation_ref", GraphQLPath: "automationRef", Required: false, IsValueReference: true},
			{Name: "synchronous", GraphQLPath: "synchronous", Required: true},
			{Name: "input_mappings", GraphQLPath: "inputMappings", Required: false, IsComplexArray: true},
			{Name: "dynamic_input_mappings", GraphQLPath: "dynamicInputMappings", Required: false, IsComplexArray: true},
			{Name: "output_mappings", GraphQLPath: "outputMappings", Required: false, IsComplexArray: true},
		},
	},
}

// GetTaskTypeConfig returns the configuration for a task type.
func GetTaskTypeConfig(taskType string) (TaskTypeConfig, bool) {
	config, ok := TaskTypeRegistry[taskType]
	return config, ok
}

// GetAllTaskGraphQLFragments returns all task type fragments for queries.
func GetAllTaskGraphQLFragments() string {
	var fragments []string
	for _, config := range TaskTypeRegistry {
		if config.GraphQLFragment != "" {
			fragments = append(fragments, config.GraphQLFragment)
		}
	}
	return strings.Join(fragments, "\n")
}

// BuildTasksQueryFragment returns the tasks field with all type-specific fragments.
// This can be embedded in workflow queries to get complete task data.
func BuildTasksQueryFragment() string {
	fragments := GetAllTaskGraphQLFragments()
	return fmt.Sprintf(`
		tasks {
			id
			__typename
			name
			previous { id }
			next { id }
			%s
		}
	`, fragments)
}

// MapTaskTypename maps a GraphQL __typename to the terraform task type name.
// For types that share a typename (like variable/update_variable which both use
// WorkflowVariableTask), this returns the base type. Content-based determination
// should be done separately (e.g., by examiningthe task data).
func MapTaskTypename(typename string) string {
	// Special case: WorkflowVariableTask can be either "variable" or "update_variable"
	// Always return the base type - content-based determination happens elsewhere
	if typename == "WorkflowVariableTask" {
		return "variable"
	}

	for typeName, config := range TaskTypeRegistry {
		if config.GraphQLTypename == typename {
			return typeName
		}
	}
	return "unknown"
}
