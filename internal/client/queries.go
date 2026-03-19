// Copyright (c) ElementumAI, Inc.
// SPDX-License-Identifier: MPL-2.0

package client

// =============================================================================
// IMPORTANT: GraphQL Query Guidelines
// =============================================================================
//
// WARNING: Do not add queries that access workflows directly from organization.
//
// The following patterns DO NOT EXIST in the GraphQL schema:
//   - organization.workflowTask(id)    - DOES NOT EXIST
//   - organization.workflowTrigger(id) - DOES NOT EXIST
//
// The following pattern BYPASSES PERMISSION CHECKS:
//   - organization.workflows           - Bypasses permissions
//
// ALWAYS query workflows through aspect for proper permission handling:
//   - organization.aspect(id).workflow(id)
//   - organization.aspect(id).automation(id).draft/current
//
// For task parent resolution, use the provider-level registry (see provider/registry.go)
// instead of querying the API.
// =============================================================================

// GraphQL Queries for Elementum API
//
// NOTE: For large apps that experience timeouts, prefer the genqlient queries:
//   - GetAspectFieldsNoValues: Lightweight query without picklist values
//   - GetFieldValues: Fetch values for a specific field (static picklists only)
//   - GetAspectName: Get just the name of an aspect (for dynamic picklist configs)
// These are defined in operations/field.graphql and operations/aspect.graphql.

const (
	// GetAspectFieldsQuery fetches fields for a specific aspect with full configuration.
	// WARNING: For large apps with many dynamic picklist fields, this may timeout.
	// Consider using GetAspectFieldsNoValues + GetFieldValues instead (see above).
	GetAspectFieldsQuery = `
		query GetAspectFields($aspectId: ID!) {
			organization {
				aspect(id: $aspectId) {
					id
					name
					... on AspectApp {
						fields {
							edges {
								node {
									...FieldDetails
								}
							}
						}
					}
					... on AspectElement {
						fields {
							edges {
								node {
									...FieldDetails
								}
							}
						}
					}
					... on AspectTask {
						fields {
							edges {
								node {
									...FieldDetails
								}
							}
						}
					}
				}
			}
		}
		fragment FieldDetails on AspectField {
			id
			name
			__typename
			required
			requiredOnClose
			showOnCreate
			lockOnCreate
			description
			semanticTags
			system
			calculation {
				text
			}
			... on AspectTextField {
				textDefaultValue: defaultValue
			}
			... on AspectHtmlField {
				htmlDefaultValue: defaultValue
			}
			... on AspectNumberField {
				numberDefaultValue: defaultValue
				format {
					currency
					decimalPlaces
					delimiter
					percentage
				}
			}
			... on AspectDecimalField {
				decimalDefaultValue: defaultValue
				format {
					currency
					decimalPlaces
					delimiter
					percentage
				}
			}
			... on AspectBooleanField {
				booleanDefaultValue: defaultValue
			}
			... on AspectDateField {
				dateDefaultValue: defaultValue
			}
			... on AspectDateTimeField {
				datetimeDefaultValue: defaultValue
			}
			... on AspectPicklistField {
				picklistDefaultValue: defaultValue {
					id
					label
					color
				}
				config {
					... on AspectStaticPicklistConfig {
						parentField {
							id
							name
						}
					}
					... on AspectDynamicPicklistConfig {
						relatedAspect {
							id
							name
						}
						relatedLabelField {
							id
							name
						}
						autoRelate
					}
				}
				values(first: 100) {
					edges {
						node {
							id
							label
							color
							tags
							active
						}
					}
				}
			}
			... on AspectMultiPicklistField {
				multiPicklistDefaultValue: defaultValue {
					id
					label
					color
				}
				config {
					... on AspectStaticPicklistConfig {
						parentField {
							id
							name
						}
					}
					... on AspectDynamicPicklistConfig {
						relatedAspect {
							id
							name
						}
						relatedLabelField {
							id
							name
						}
						autoRelate
					}
				}
				values(first: 100) {
					edges {
						node {
							id
							label
							color
							tags
							active
						}
					}
				}
			}
			... on AspectUserField {
				userDefaultValue: defaultValue {
					id
					name
				}
			}
			... on AspectGroupField {
				groupDefaultValue: defaultValue {
					id
					name
				}
			}
			... on AspectJsonField {
				jsonDefaultValue: defaultValue
			}
			... on AspectAttachmentField {
				mediaTypes
			}
		}
	`

	// GetAspectAgentsQuery fetches agents for a specific app including their tools.
	GetAspectAgentsQuery = `
		query GetAspectAgents($aspectId: ID!) {
			organization {
				aspect(id: $aspectId) {
					id
					name
					... on AspectApp {
						agentsV2 {
							edges {
								node {
									id
									name
									description
									__typename
									... on AgentElementum {
										instructions
										firstMessage
										aiProviderConnector {
											id
											model {
												name
											}
											provider {
												name
											}
										}
										tools {
											edges {
												node {
													...AgentToolSummary
												}
											}
										}
									}
									... on AgentSnowflake {
										instructions
										firstMessage
										agentSnowflakeConfig {
											database
											name
											schema
										}
										aiProviderConnector {
											id
											model {
												name
											}
											provider {
												name
											}
										}
										tools {
											edges {
												node {
													...AgentToolSummary
												}
											}
										}
									}
									... on AgentBedrock {
										instructions
										firstMessage
										agentBedrockConfig {
											agentArn
										}
										aiProviderConnector {
											id
											model {
												name
											}
											provider {
												name
											}
										}
										tools {
											edges {
												node {
													...AgentToolSummary
												}
											}
										}
									}
									... on AgentBrowserUse {
										instructions
										firstMessage
										browserAgentInstructions
										aiProviderConnector {
											id
											model {
												name
											}
											provider {
												name
											}
										}
										tools {
											edges {
												node {
													...AgentToolSummary
												}
											}
										}
									}
								}
							}
						}
					}
				}
			}
		}
		fragment AgentToolSummary on AgentTool {
			id
			name
			description
			startMessage
			__typename
			... on AgentCreateRecordTool {
				aspect {
					id
					name
				}
				fieldsV2 {
					name
					description
					required
					field {
						id
						name
					}
				}
			}
			... on AgentSearchAspectTool {
				aspect {
					id
					name
				}
				queryDescription
				limit
				fields {
					description
					field {
						id
						name
					}
				}
			}
			... on AgentUpdateRecordTool {
				aspect {
					id
					name
				}
				handleDescription
				fieldsV2 {
					name
					description
					required
					field {
						id
						name
					}
				}
			}
			... on AgentSearchTableTool {
				aspect {
					id
					name
				}
				queryDescription
				limit
				table {
					id
					field {
						id
						name
					}
				}
				attributeFields {
					id
					name
				}
				returnFields {
					name
					field {
						id
						name
					}
				}
			}
			... on AgentRelateRecordTool {
				aspect {
					id
					name
				}
				relatedAspect {
					id
					name
				}
			}
			... on AgentExecuteWorkflowTool {
				automation {
					id
					name
				}
				inputs {
					name
					description
					required
					triggerParameter {
						id
						name
					}
				}
				outputs {
					name
					workflowPropertyName
				}
			}
			... on AgentRunAgentTool {
				targetAgent {
					id
					name
				}
				workerTaskPrompt
			}
		}
	`

	// GetAutomationDetailsQuery fetches automation details including triggers and parameters.
	// Returns both current (published) and draft workflows to track workflow state.
	// Deprecated: Use the genqlient GetAutomationDetails function instead.
	// This constant is kept for backward compatibility with tests.
	GetAutomationDetailsQuery = `
		query GetAutomationDetails($automationId: ID!) {
			organization {
				id
				automation(id: $automationId) {
					id
					name
					status
					entity {
						... on AspectApp {
							id
							name
						}
						... on AspectElement {
							id
							name
						}
						... on AspectTask {
							id
							name
						}
					}
					current {
						id
						version
						status
						terminal
						triggers {
							__typename
							... on WorkflowRecordCreateTrigger {
								filter
								changedFields {
									id
									name
								}
							}
							... on WorkflowRecordUpdateTrigger {
								filter
								changedFields {
									id
									name
								}
							}
							... on WorkflowOnDemandTrigger {
								parameters {
									id
									name
									fieldType
									required
									defaultValue
									multiple
								}
							}
							... on WorkflowWebhookTrigger {
								authenticated
								url
							}
						}
					}
					draft {
						id
						version
						status
					}
				}
			}
		}
	`
)

// =============================================================================
// Dynamic Query Builders
// =============================================================================
// These functions build queries dynamically using the trigger and task type
// registries, ensuring both CLI and provider use consistent, complete fragments.

// BuildAppAutomationsQuery builds a query to fetch all automations for an app
// with complete trigger and task fragments.
// NOTE: This query can timeout on large apps. For large apps, use the two-phase
// approach: GetAutomationsList (genqlient) + BuildWorkflowDetailsQuery per workflow.
func BuildAppAutomationsQuery() string {
	triggerFragments := GetAllTriggerGraphQLFragments()
	taskFragments := GetAllTaskGraphQLFragments()

	return `
		query GetAppAutomationsWithWorkflows($aspectId: ID!) {
			organization {
				aspect(id: $aspectId) {
					... on AspectApp {
						automations {
							edges {
								node {
									id
									name
									status
									current {
										id
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
											` + taskFragments + `
										}
									}
									draft {
										id
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
											` + taskFragments + `
										}
									}
								}
							}
						}
					}
				}
			}
		}
	`
}

// BuildWorkflowDetailsQuery builds a query to fetch a single workflow's full details.
// Deprecated: Use BuildFullWorkflowQuery() from workflow_query.go or
// Client.GetWorkflowWithFullTasks() which adds version/status/terminal/outputs.
func BuildWorkflowDetailsQuery() string {
	triggerFragments := GetAllTriggerGraphQLFragments()
	taskFragments := GetAllTaskGraphQLFragments()

	return `
		query GetWorkflowFullDetails($aspectId: ID!, $workflowId: ID!) {
			organization {
				aspect(id: $aspectId) {
					workflow(id: $workflowId) {
						id
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
							` + taskFragments + `
						}
					}
				}
			}
		}
	`
}
