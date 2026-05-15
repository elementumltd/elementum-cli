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

package client

import (
	"fmt"
	"strings"
)

// ValueRefFragment is the standard GraphQL fragment for ValueReference fields.
// It includes all reference types (trigger, task, template, forEach, variable)
// so that the export pipeline can resolve value references to proper Terraform syntax.
// This is the same pattern used by the api task type.
const ValueRefFragment = `{ id value triggerReference { name } taskReference { name task { id } } templateReference { template parameters { ... on ValueReferenceTemplateValueParameter { name value { triggerReference { name } taskReference { name task { id } } variableReference { variableId name type } forEachReference { name forEachTask { id } } } } } } }`

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
						createValue: value ` + ValueRefFragment + `
					}
					... on WorkflowVariableTaskParameterUpdate {
						updateValue: value ` + ValueRefFragment + `
						variable ` + ValueRefFragment + `
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
				recordReference ` + ValueRefFragment + `
				workflowFields {
					field { id name }
					valueReference ` + ValueRefFragment + `
				}
			}
		`,
		Fields: []TaskFieldConfig{
			{Name: "object_id", GraphQLPath: "aspect.id", Required: false, IsReference: true, ReferenceType: "object"},
			{Name: "record_reference", GraphQLPath: "recordReference", Required: true, IsValueReference: true},
			{Name: "fields", GraphQLPath: "workflowFields", Required: true},
		},
	},
	"create_record": {
		TypeName:        "create_record",
		GraphQLTypename: "WorkflowCreateRecordTask",
		GraphQLFragment: `
			... on WorkflowCreateRecordTask {
				aspect { id name }
				recordReference ` + ValueRefFragment + `
				relateToRecord
				workflowFields {
					field { id name }
					valueReference ` + ValueRefFragment + `
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
				sort
				filterValueReferences {
					valueReferences ` + ValueRefFragment + `
				}
			}
		`,
		Fields: []TaskFieldConfig{
			{Name: "object_id", GraphQLPath: "aspect.id", Required: true, IsReference: true, ReferenceType: "object"},
			{Name: "limit", GraphQLPath: "limit", Required: false},
			{Name: "filter", GraphQLPath: "filter", Required: false},
			{Name: "sort", GraphQLPath: "sort", Required: false},
		},
	},
	"ai_agent": {
		TypeName:        "ai_agent",
		GraphQLTypename: "WorkflowAiAgentTask",
		GraphQLFragment: `
			... on WorkflowAiAgentTask {
				agent { id name }
				agentPrompt ` + ValueRefFragment + `
				outputType
				jsonFields {
					name
					type
					description
					required
				}
			}
		`,
		Fields: []TaskFieldConfig{
			{Name: "agent_id", GraphQLPath: "agent.id", Required: true, IsReference: true, ReferenceType: "agent"},
			{Name: "prompt", GraphQLPath: "agentPrompt", Required: true, IsValueReference: true},
			// outputType is a WorkflowAiAgentOutputType enum (e.g. "JSON", "TEXT").
			// The provider expects a string scalar; the generic default-case
			// emitter handles this as a quoted string.
			{Name: "output_type", GraphQLPath: "outputType", Required: false},
		},
	},
	"message": {
		TypeName:        "message",
		GraphQLTypename: "WorkflowMessageTask",
		GraphQLFragment: `
			... on WorkflowMessageTask {
				contentsReference ` + ValueRefFragment + `
				recordReference ` + ValueRefFragment + `
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
				subject ` + ValueRefFragment + `
				messageBody ` + ValueRefFragment + `
				fromPersonal
				fromUsername
			}
		`,
		// NOTE: users / groups / externalEmails / emailAttachments are
		// `[ValueReference!]` arrays on this task. Our generic emitter
		// doesn't yet render `[ValueReference!]` arrays (only single
		// ValueReference). If a future export of an app that uses
		// send_email needs them, add an IsValueReferenceList flag +
		// dedicated emitter — <app-namespace> doesn't use send_email, so the
		// array fields stay unfetched to avoid bloating RawData.
		Fields: []TaskFieldConfig{
			{Name: "subject", GraphQLPath: "subject", Required: true, IsValueReference: true},
			{Name: "message_body", GraphQLPath: "messageBody", Required: true, IsValueReference: true},
			{Name: "from_personal", GraphQLPath: "fromPersonal", Required: false},
			{Name: "from_username", GraphQLPath: "fromUsername", Required: false},
		},
	},
	"for_each": {
		TypeName:        "for_each",
		GraphQLTypename: "WorkflowForEachTask",
		GraphQLFragment: `
			... on WorkflowForEachTask {
				forEach ` + ValueRefFragment + `
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
				recordId ` + ValueRefFragment + `
				file {
					... on ValueReference ` + ValueRefFragment + `
				}
				tag ` + ValueRefFragment + `
				viewState
			}
		`,
		Fields: []TaskFieldConfig{
			{Name: "record_id", GraphQLPath: "recordId", Required: true, IsValueReference: true},
			{Name: "file", GraphQLPath: "file", Required: true, IsValueReference: true},
			{Name: "tag", GraphQLPath: "tag", Required: false, IsValueReference: true},
			// WorkflowSaveAttachmentViewState enum (ATTACHMENT_FIELD | FILE_FIELD).
			{Name: "view_state", GraphQLPath: "viewState", Required: false},
		},
	},
	"notification": {
		TypeName:        "notification",
		GraphQLTypename: "WorkflowNotificationTask",
		GraphQLFragment: `
			... on WorkflowNotificationTask {
				messageReference ` + ValueRefFragment + `
				titleReference ` + ValueRefFragment + `
				descriptionReference ` + ValueRefFragment + `
				notifyWatchers
			}
		`,
		// userValueReferences / groupValueReferences are `[ValueReference!]`
		// arrays; rendering them needs an IsValueReferenceList flag that
		// doesn't exist yet. <app-namespace> doesn't use notification tasks, so
		// we omit the array fields rather than half-implement.
		Fields: []TaskFieldConfig{
			{Name: "message", GraphQLPath: "messageReference", Required: true, IsValueReference: true},
			{Name: "title", GraphQLPath: "titleReference", Required: false, IsValueReference: true},
			{Name: "description", GraphQLPath: "descriptionReference", Required: false, IsValueReference: true},
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
				attachment ` + ValueRefFragment + `
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
				recordReference ` + ValueRefFragment + `
				watcherUsers: users { id }
				watcherGroups: groups { id }
			}
		`,
		Fields: []TaskFieldConfig{
			{Name: "record_reference", GraphQLPath: "recordReference", Required: true, IsValueReference: true},
			// user_ids / group_ids are list-of-id attributes on the provider.
			// The dedicated handler in hcl_task.go converts the [{id}, ...]
			// arrays from GraphQL into `[data.elementum_user.x.id, ...]` refs.
			{Name: "user_ids", GraphQLPath: "watcherUsers", Required: false},
			{Name: "group_ids", GraphQLPath: "watcherGroups", Required: false},
		},
	},
	"relate_records": {
		TypeName:        "relate_records",
		GraphQLTypename: "WorkflowRelateRecordsTask",
		GraphQLFragment: `
			... on WorkflowRelateRecordsTask {
				sourceRecordIds ` + ValueRefFragment + `
				targetRecordId ` + ValueRefFragment + `
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
				recordId ` + ValueRefFragment + `
				relatedAspect { id name }
				limit
			}
		`,
		Fields: []TaskFieldConfig{
			{Name: "record_reference", GraphQLPath: "recordId", Required: true, IsValueReference: true},
			{Name: "object_id", GraphQLPath: "relatedAspect.id", Required: true, IsReference: true, ReferenceType: "object"},
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
					valueReferences ` + ValueRefFragment + `
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
							valueReferences ` + ValueRefFragment + `
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
				textToSummarize ` + ValueRefFragment + `
				context ` + ValueRefFragment + `
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
				prompt ` + ValueRefFragment + `
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
		GraphQLTypename: "WorkflowApprovalChainTemplateTask",
		GraphQLFragment: `
			... on WorkflowApprovalChainTemplateTask {
				approvalChainTemplate { id }
				dynamicApprovers { id ` + ValueRefFragment + ` }
				recordId ` + ValueRefFragment + `
				requesterId ` + ValueRefFragment + `
				reason ` + ValueRefFragment + `
				summary ` + ValueRefFragment + `
			}
		`,
		Fields: []TaskFieldConfig{
			{Name: "approvalChainTemplate", GraphQLPath: "approvalChainTemplate", Required: true},
			{Name: "dynamicApprovers", GraphQLPath: "dynamicApprovers", Required: false, IsValueReference: true},
			{Name: "recordId", GraphQLPath: "recordId", Required: true, IsValueReference: true},
			{Name: "requesterId", GraphQLPath: "requesterId", Required: true, IsValueReference: true},
			{Name: "reason", GraphQLPath: "reason", Required: false, IsValueReference: true},
			{Name: "summary", GraphQLPath: "summary", Required: false, IsValueReference: true},
		},
	},
	"approval_status_update": {
		TypeName:        "approval_status_update",
		GraphQLTypename: "WorkflowApprovalStatusUpdateTask",
		GraphQLFragment: `
			... on WorkflowApprovalStatusUpdateTask {
				approvalChainTemplate { id }
				status
				reason ` + ValueRefFragment + `
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
				recordId ` + ValueRefFragment + `
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
				attachment ` + ValueRefFragment + `
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
				query ` + ValueRefFragment + `
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
					value ` + ValueRefFragment + `
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
				automationRef ` + ValueRefFragment + `
				synchronous
				inputMappings {
					parameter { id name }
					value ` + ValueRefFragment + `
				}
				dynamicInputMappings {
					name
					value ` + ValueRefFragment + `
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
	"execute_script": {
		TypeName:        "execute_script",
		GraphQLTypename: "WorkflowExecuteScriptTask",
		GraphQLFragment: `
			... on WorkflowExecuteScriptTask {
				code
				inputs {
					name
					value ` + ValueRefFragment + `
					fieldMappings { alias field { id name } }
				}
				outputSchema {
					__typename
					name
					... on JsonSchemaString { format }
					... on JsonSchemaObject {
						properties {
							__typename
							name
							... on JsonSchemaString { format }
							... on JsonSchemaArray {
								item {
									__typename
									name
									... on JsonSchemaString { format }
								}
							}
						}
					}
					... on JsonSchemaArray {
						item {
							__typename
							name
							... on JsonSchemaString { format }
							... on JsonSchemaObject {
								properties {
									__typename
									name
									... on JsonSchemaString { format }
								}
							}
						}
					}
				}
			}
		`,
		Fields: []TaskFieldConfig{
			{Name: "code", GraphQLPath: "code", Required: true},
			{Name: "inputs", GraphQLPath: "inputs", Required: true, IsComplexArray: true},
			{Name: "output_schema", GraphQLPath: "outputSchema", Required: false},
		},
	},
	"fork_join": {
		TypeName:        "fork_join",
		GraphQLTypename: "WorkflowForkJoinTask",
		GraphQLFragment: `
			... on WorkflowForkJoinTask {
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
			// Fork/join tasks only have base fields (parent_id, name)
			// Branches are separate resources
		},
	},
	"file_reader": {
		TypeName:        "file_reader",
		GraphQLTypename: "WorkflowDocumentModelTask",
		GraphQLFragment: `
			... on WorkflowDocumentModelTask {
				documentModelDataSource {
					attachment ` + ValueRefFragment + `
					attachmentId ` + ValueRefFragment + `
					file ` + ValueRefFragment + `
					rawText ` + ValueRefFragment + `
				}
				documentModel { id }
			}
		`,
		Fields: []TaskFieldConfig{
			// Input source is a union — exactly one of attachment / attachment_id
			// / file / raw_text is populated. Emitting all four and letting the
			// `IsValueReference` emitter skip nils keeps the registry 1:1 with
			// the provider's schema for round-trip parity.
			{Name: "attachment", GraphQLPath: "documentModelDataSource.attachment", Required: false, IsValueReference: true},
			{Name: "attachment_id", GraphQLPath: "documentModelDataSource.attachmentId", Required: false, IsValueReference: true},
			{Name: "file", GraphQLPath: "documentModelDataSource.file", Required: false, IsValueReference: true},
			{Name: "raw_text", GraphQLPath: "documentModelDataSource.rawText", Required: false, IsValueReference: true},
			{Name: "file_reader_id", GraphQLPath: "documentModel.id", Required: true, IsReference: true, ReferenceType: "file_reader"},
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
