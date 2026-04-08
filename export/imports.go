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
	"fmt"
	"strings"

	"github.com/elementumltd/elementum-cli/discovery"
	"github.com/elementumltd/elementum-cli/logger"
)

// ImportBlock represents a Terraform import block
type ImportBlock struct {
	ID           string
	ResourceType string
	ResourceName string
}

// systemFieldTagsWithStatus are semantic tags that indicate system-managed fields (includes STATUS for apps)
var systemFieldTagsWithStatus = []string{"TITLE", "STATUS", "STAGE", "HANDLE", "CREATED_AT", "CREATED_BY", "UPDATED_AT", "UPDATED_BY", "CLOSED_AT", "CLOSED_BY", "VERSION"}

// systemFieldTagsNoStatus are semantic tags for elements/tasks (STATUS is allowed)
var systemFieldTagsNoStatus = []string{"TITLE", "HANDLE", "CREATED_AT", "CREATED_BY", "UPDATED_AT", "UPDATED_BY", "CLOSED_AT", "CLOSED_BY", "VERSION"}

// systemFieldNamePatterns are field name patterns (lowercase) that indicate system fields
// These serve as a fallback when SemanticTags are not populated by the API
var systemFieldNamePatterns = []string{"closed on", "closed at", "closed by"}

// isSystemField checks if a field should be skipped based on system semantic tags or name patterns.
// Set allowStatus=true for elements/tasks where STATUS fields can be custom dropdowns.
// Returns true if the field is a system field that cannot be managed via Terraform.
func isSystemField(field discovery.Field, allowStatus bool) bool {
	systemTags := systemFieldTagsWithStatus
	if allowStatus {
		systemTags = systemFieldTagsNoStatus
	}

	// Check for system semantic tags
	for _, tag := range field.SemanticTags {
		for _, systemTag := range systemTags {
			if strings.EqualFold(tag, systemTag) {
				return true
			}
		}
	}

	// Fallback: check field name patterns for system fields that may not have SemanticTags populated
	lowerName := strings.ToLower(field.Name)
	for _, pattern := range systemFieldNamePatterns {
		if lowerName == pattern {
			return true
		}
	}

	return false
}

// GenerateImportBlocks generates Terraform import blocks for an app and its resources
func GenerateImportBlocks(app *discovery.App, selectedTypes map[string]bool) []ImportBlock {
	blocks := []ImportBlock{}

	// Track relationship IDs to avoid exporting the same relationship twice
	// (bidirectional relationships appear from both app and element sides)
	seenRelationshipIDs := make(map[string]bool)

	// Track resource IDs to prevent duplicates (e.g., same table in both ReferencedTables and DiscoveredTables)
	seenResourceIDs := make(map[string]bool)

	// Always include the app itself if any resources are selected
	if len(selectedTypes) > 0 {
		blocks = append(blocks, ImportBlock{
			ID:           app.ID,
			ResourceType: "elementum_app",
			ResourceName: AppResourceName(app),
		})
	}

	// Fields
	if selectedTypes["fields"] {
		fieldNames := make(map[string]int)
		skippedFields := 0
		for _, field := range app.Fields {
			// Skip fields that cannot be managed via Terraform:
			// 1. Unknown/unrecognized field types
			// 2. Empty type (malformed data)
			// 3. Handle fields (system-managed ID field)
			// 4. Fields with system semantic tags (TITLE, STATUS, STAGE, etc.)
			if field.Type == "unknown" || field.Type == "" || field.Type == "handle" {
				logger.Debug("skipping system-managed field", "name", field.Name, "type", field.Type)
				skippedFields++
				continue
			}

			// Check for system fields (semantic tags or name patterns)
			// Apps don't allow STATUS fields to be created
			if isSystemField(field, false) {
				logger.Debug("skipping system field", "name", field.Name, "tags", field.SemanticTags)
				skippedFields++
				continue
			}

			baseName := SanitizeName(field.Name)
			count := fieldNames[baseName]
			fieldNames[baseName]++

			name := baseName
			if count > 0 {
				name = fmt.Sprintf("%s_%d", baseName, count)
			}

			blocks = append(blocks, ImportBlock{
				ID:           BuildFieldImportID(app.ID, field.ID, field.Type),
				ResourceType: GetResourceTypeForField(field),
				ResourceName: name,
			})
		}
		logger.Debug("fields summary", "total", len(app.Fields), "skipped", skippedFields, "included", len(app.Fields)-skippedFields)
	}

	// Layouts
	if selectedTypes["layouts"] {
		// Track layout names to ensure uniqueness
		layoutNames := make(map[string]int)
		for _, layout := range app.Layouts {
			// Skip Initiate layouts - they are platform-managed
			if layout.IsInitiate {
				logger.Debug("skipping initiate layout", "id", layout.ID, "name", layout.Name)
				continue
			}

			baseName := SanitizeName(layout.Name)
			count := layoutNames[baseName]
			layoutNames[baseName]++

			// Add suffix if duplicate
			name := baseName
			if count > 0 {
				name = fmt.Sprintf("%s_%d", baseName, count)
			}

			// Layout import ID format: app_id:stage_id
			blocks = append(blocks, ImportBlock{
				ID:           BuildImportID("elementum_layout", map[string]string{"app_id": app.ID, "layout_id": layout.ID}),
				ResourceType: "elementum_layout",
				ResourceName: name,
			})
		}
	}

	// Flows
	if selectedTypes["flows"] {
		flowNames := make(map[string]int)
		for _, flow := range app.Flows {
			baseName := SanitizeName(flow.Name)
			count := flowNames[baseName]
			flowNames[baseName]++

			name := baseName
			if count > 0 {
				name = fmt.Sprintf("%s_%d", baseName, count)
			}

			blocks = append(blocks, ImportBlock{
				ID:           BuildImportID("elementum_flow", map[string]string{"app_id": app.ID, "flow_id": flow.ID}),
				ResourceType: "elementum_flow",
				ResourceName: name,
			})
		}
	}

	// Automations (includes triggers and tasks) - only export published automations
	if selectedTypes["automations"] {
		automationNames := make(map[string]int)
		triggerNames := make(map[string]int)
		taskNames := make(map[string]int)

		for _, automation := range app.Automations {
			// Skip unpublished or inactive automations
			if !automation.HasPublished || automation.Status != "ACTIVE" {
				reason := "unknown"
				if !automation.HasPublished {
					reason = "no published workflow (current is null)"
				} else if automation.Status != "ACTIVE" {
					reason = "status is " + automation.Status
				}
				logger.Debug("skipping automation",
					"name", automation.Name,
					"id", automation.ID,
					"status", automation.Status,
					"hasPublished", automation.HasPublished,
					"reason", reason,
				)
				continue
			}

			// Automation itself
			autoBaseName := SanitizeName(automation.Name)
			autoCount := automationNames[autoBaseName]
			automationNames[autoBaseName]++

			autoName := autoBaseName
			if autoCount > 0 {
				autoName = fmt.Sprintf("%s_%d", autoBaseName, autoCount)
			}

			blocks = append(blocks, ImportBlock{
				ID:           BuildImportID("elementum_automation", map[string]string{"app_id": app.ID, "automation_id": automation.ID}),
				ResourceType: "elementum_automation",
				ResourceName: autoName,
			})

			// Workflow publish
			blocks = append(blocks, ImportBlock{
				ID:           BuildImportID("elementum_workflow_publish", map[string]string{"automation_id": automation.ID}),
				ResourceType: "elementum_workflow_publish",
				ResourceName: autoName,
			})

			// Triggers
			for i, trigger := range automation.Triggers {
				// Skip unknown trigger types - the provider doesn't have a resource for them yet
				if trigger.Type == "unknown" {
					logger.Debug("skipping unknown trigger type", "name", trigger.Name, "reason", "GraphQL typename not mapped")
					continue
				}

				triggerBaseName := SanitizeName(fmt.Sprintf("%s_%s_%d", automation.Name, trigger.Type, i))
				triggerCount := triggerNames[triggerBaseName]
				triggerNames[triggerBaseName]++

				triggerName := triggerBaseName
				if triggerCount > 0 {
					triggerName = fmt.Sprintf("%s_%d", triggerBaseName, triggerCount)
				}

				blocks = append(blocks, ImportBlock{
					ID:           BuildTriggerImportID(automation.ID, trigger.ID, trigger.Type),
					ResourceType: GetResourceTypeForTrigger(trigger),
					ResourceName: triggerName,
				})
			}

			// Tasks
			for _, task := range automation.Tasks {
				// Skip unknown task types - the provider doesn't have a resource for them yet
				if task.Type == "unknown" {
					logger.Debug("skipping unknown task type", "name", task.Name, "reason", "GraphQL typename not mapped")
					continue
				}

				taskBaseName := SanitizeName(fmt.Sprintf("%s_%s", automation.Name, task.Name))
				taskCount := taskNames[taskBaseName]
				taskNames[taskBaseName]++

				taskName := taskBaseName
				if taskCount > 0 {
					taskName = fmt.Sprintf("%s_%d", taskBaseName, taskCount)
				}

				blocks = append(blocks, ImportBlock{
					ID:           BuildTaskImportID(automation.WorkflowID, task.ID, task.Type),
					ResourceType: GetResourceTypeForTask(task),
					ResourceName: taskName,
				})
			}
		}
	}

	// Agents (and their tools)
	if selectedTypes["agents"] {
		agentNames := make(map[string]int)
		toolNames := make(map[string]int)

		for _, agent := range app.Agents {
			agentBaseName := SanitizeName(agent.Name)
			agentCount := agentNames[agentBaseName]
			agentNames[agentBaseName]++

			agentName := agentBaseName
			if agentCount > 0 {
				agentName = fmt.Sprintf("%s_%d", agentBaseName, agentCount)
			}

			blocks = append(blocks, ImportBlock{
				ID:           BuildImportID("elementum_agent", map[string]string{"app_id": app.ID, "agent_id": agent.ID}),
				ResourceType: "elementum_agent",
				ResourceName: agentName,
			})

			// Agent tools - use tool-specific resource types
			for _, tool := range agent.Tools {
				resourceType := GetResourceTypeForAgentTool(&tool)
				if resourceType == "" {
					// Unknown tool type, skip
					continue
				}

				toolBaseName := SanitizeName(fmt.Sprintf("%s_%s", agent.Name, tool.Name))
				toolCount := toolNames[toolBaseName]
				toolNames[toolBaseName]++

				toolName := toolBaseName
				if toolCount > 0 {
					toolName = fmt.Sprintf("%s_%d", toolBaseName, toolCount)
				}

				blocks = append(blocks, ImportBlock{
					ID: BuildImportID(resourceType, map[string]string{
						"app_id":   app.ID,
						"agent_id": agent.ID,
						"tool_id":  tool.ID,
					}),
					ResourceType: resourceType,
					ResourceName: toolName,
				})
			}
		}
	}

	// Widgets
	if selectedTypes["widgets"] {
		widgetNames := make(map[string]int)
		logger.Debug("discovered widgets in app", "count", len(app.Widgets))

		for _, widget := range app.Widgets {
			widgetBaseName := SanitizeName(widget.Name)
			widgetCount := widgetNames[widgetBaseName]
			widgetNames[widgetBaseName]++

			widgetName := widgetBaseName
			if widgetCount > 0 {
				widgetName = fmt.Sprintf("%s_%d", widgetBaseName, widgetCount)
			}

			blocks = append(blocks, ImportBlock{
				ID:           BuildImportID("elementum_widget", map[string]string{"app_id": app.ID, "widget_id": widget.ID}),
				ResourceType: "elementum_widget",
				ResourceName: widgetName,
			})
		}
		logger.Debug("generated widget import blocks", "count", len(app.Widgets))
	}

	// Views
	if selectedTypes["views"] {
		viewNames := make(map[string]int)
		logger.Debug("discovered views in app", "count", len(app.Views))

		for _, view := range app.Views {
			// Skip agent views if agents aren't being exported (they would reference undeclared agents)
			if view.Type == "AspectViewAgent" && !selectedTypes["agents"] {
				logger.Debug("skipping agent view - agents not selected", "view", view.Name)
				continue
			}

			viewBaseName := SanitizeName(view.Name)
			viewCount := viewNames[viewBaseName]
			viewNames[viewBaseName]++

			viewName := viewBaseName
			if viewCount > 0 {
				viewName = fmt.Sprintf("%s_%d", viewBaseName, viewCount)
			}

			blocks = append(blocks, ImportBlock{
				ID:           fmt.Sprintf("%s:%s", app.ID, view.ID),
				ResourceType: view.TerraformResourceType(),
				ResourceName: viewName,
			})
		}
		logger.Debug("generated view import blocks", "count", len(app.Views))
	}

	// File Readers (AI, Text/OCR, JSON, XML)
	if selectedTypes["ai_file_readers"] {
		fileReaderNames := make(map[string]int)

		for _, reader := range app.AIFileReaders {
			readerBaseName := SanitizeName(reader.Name)
			readerCount := fileReaderNames[readerBaseName]
			fileReaderNames[readerBaseName]++

			readerName := readerBaseName
			if readerCount > 0 {
				readerName = fmt.Sprintf("%s_%d", readerBaseName, readerCount)
			}

			// Use the correct resource type based on the file reader type
			resourceType := reader.TerraformResourceType()

			blocks = append(blocks, ImportBlock{
				ID: BuildImportID(resourceType, map[string]string{
					"app_id":         app.ID,
					"file_reader_id": reader.ID,
				}),
				ResourceType: resourceType,
				ResourceName: readerName,
			})
		}
		logger.Debug("generated file reader import blocks", "count", len(app.AIFileReaders))
	}

	// Approval Processes
	if selectedTypes["approval_processes"] {
		approvalNames := make(map[string]int)

		for _, approval := range app.Approvals {
			approvalBaseName := SanitizeName(approval.Name)
			approvalCount := approvalNames[approvalBaseName]
			approvalNames[approvalBaseName]++

			approvalName := approvalBaseName
			if approvalCount > 0 {
				approvalName = fmt.Sprintf("%s_%d", approvalBaseName, approvalCount)
			}

			blocks = append(blocks, ImportBlock{
				ID: BuildImportID("elementum_approval_process", map[string]string{
					"app_id":      app.ID,
					"approval_id": approval.ID,
				}),
				ResourceType: "elementum_approval_process",
				ResourceName: approvalName,
			})
		}
		logger.Debug("generated approval process import blocks", "count", len(app.Approvals))
	}

	// Relationships
	if selectedTypes["relationships"] {
		// Compute prefix from namespace or name for uniqueness across apps
		prefix := SanitizeName(app.Namespace)
		if prefix == "" {
			prefix = SanitizeName(app.Name)
		}

		relationshipNames := make(map[string]int)

		for _, relationship := range app.Relationships {
			// Skip if this relationship was already exported (handles bidirectional relationships)
			if seenRelationshipIDs[relationship.ID] {
				logger.Debug("skipping duplicate relationship", "id", relationship.ID, "related", relationship.RelatedObjectName)
				continue
			}
			seenRelationshipIDs[relationship.ID] = true

			// Generate name with app prefix to avoid conflicts when multiple apps relate to the same target
			relationshipBaseName := prefix + "_to_" + SanitizeName(relationship.RelatedObjectName)
			if relationship.RelatedObjectName == "" {
				relationshipBaseName = prefix + "_relationship"
			}
			relationshipCount := relationshipNames[relationshipBaseName]
			relationshipNames[relationshipBaseName]++

			relationshipName := relationshipBaseName
			if relationshipCount > 0 {
				relationshipName = fmt.Sprintf("%s_%d", relationshipBaseName, relationshipCount)
			}

			blocks = append(blocks, ImportBlock{
				ID: BuildImportID("elementum_relationship", map[string]string{
					"object_id":       app.ID,
					"relationship_id": relationship.ID,
				}),
				ResourceType: "elementum_relationship",
				ResourceName: relationshipName,
			})
		}
		logger.Debug("generated relationship import blocks", "count", len(app.Relationships))
	}

	// Roles (custom roles only - managed roles are exported as data sources)
	if selectedTypes["roles"] {
		roleNames := make(map[string]int)

		for _, role := range app.Roles {
			// Skip managed roles - they're exported as data sources, not resources
			if role.Managed {
				logger.Debug("skipping managed role for import (will be data source)", "name", role.Name, "id", role.ID)
				continue
			}

			roleBaseName := SanitizeName(role.Name)
			roleCount := roleNames[roleBaseName]
			roleNames[roleBaseName]++

			roleName := roleBaseName
			if roleCount > 0 {
				roleName = fmt.Sprintf("%s_%d", roleBaseName, roleCount)
			}

			blocks = append(blocks, ImportBlock{
				ID: BuildImportID("elementum_role", map[string]string{
					"object_id": app.ID,
					"role_id":   role.ID,
				}),
				ResourceType: "elementum_role",
				ResourceName: roleName,
			})
		}
		logger.Debug("generated role import blocks", "total", len(app.Roles), "customRoles", len(roleNames))
	}

	// Access Policies
	if selectedTypes["access_policies"] {
		accessPolicyNames := make(map[string]int)

		for _, policy := range app.AccessPolicies {
			accessPolicyBaseName := "access_policy"
			accessPolicyCount := accessPolicyNames[accessPolicyBaseName]
			accessPolicyNames[accessPolicyBaseName]++

			accessPolicyName := accessPolicyBaseName
			if accessPolicyCount > 0 {
				accessPolicyName = fmt.Sprintf("%s_%d", accessPolicyBaseName, accessPolicyCount)
			}

			blocks = append(blocks, ImportBlock{
				ID: BuildImportID("elementum_access_policy", map[string]string{
					"object_id": app.ID,
					"policy_id": policy.ID,
				}),
				ResourceType: "elementum_access_policy",
				ResourceName: accessPolicyName,
			})
		}
		logger.Debug("generated access policy import blocks", "count", len(app.AccessPolicies))
	}

	// Phone Services
	if selectedTypes["phone_services"] {
		phoneServiceNames := make(map[string]int)

		for _, service := range app.PhoneServices {
			// Use phone number as base name if available, otherwise use a generic name
			baseName := "phone_service"
			if service.PhoneNumber != "" {
				// Sanitize phone number for resource name (remove + and -)
				baseName = SanitizeName(service.PhoneNumber)
			}

			count := phoneServiceNames[baseName]
			phoneServiceNames[baseName]++

			name := baseName
			if count > 0 {
				name = fmt.Sprintf("%s_%d", baseName, count)
			}

			blocks = append(blocks, ImportBlock{
				ID: BuildImportID("elementum_phone_service", map[string]string{
					"app_id":     app.ID,
					"service_id": service.ID,
				}),
				ResourceType: "elementum_phone_service",
				ResourceName: name,
			})
		}
		logger.Debug("generated phone service import blocks", "count", len(app.PhoneServices))
	}

	// AI Search Tables
	if selectedTypes["ai_search_tables"] {
		searchTableNames := make(map[string]int)

		for _, searchTable := range app.AISearchTables {
			// Use field name as base name if available, otherwise use a generic name
			baseName := "ai_search_table"
			if searchTable.FieldName != "" {
				baseName = SanitizeName(searchTable.FieldName)
			}

			count := searchTableNames[baseName]
			searchTableNames[baseName]++

			name := baseName
			if count > 0 {
				name = fmt.Sprintf("%s_%d", baseName, count)
			}

			blocks = append(blocks, ImportBlock{
				ID: BuildImportID("elementum_ai_search_table", map[string]string{
					"object_id":       app.ID,
					"search_table_id": searchTable.ID,
				}),
				ResourceType: "elementum_ai_search_table",
				ResourceName: name,
			})
		}
		logger.Debug("generated AI search table import blocks", "count", len(app.AISearchTables))
	}

	// Managed View Order
	if selectedTypes["managed_view_order"] && app.ManagedViewOrder != nil && len(app.ManagedViewOrder.ViewIDs) > 0 {
		blocks = append(blocks, ImportBlock{
			ID: BuildImportID("elementum_managed_view_order", map[string]string{
				"object_id": app.ID,
			}),
			ResourceType: "elementum_managed_view_order",
			ResourceName: SanitizeName(app.Name) + "_view_order",
		})
		logger.Debug("generated managed view order import block")
	}

	// Dashboards (and their widgets)
	if selectedTypes["dashboards"] {
		dashboardNames := make(map[string]int)

		for _, dashboard := range app.Dashboards {
			baseName := SanitizeName(dashboard.Name)
			if baseName == "" {
				baseName = "dashboard"
			}

			count := dashboardNames[baseName]
			dashboardNames[baseName]++

			dashboardResourceName := baseName
			if count > 0 {
				dashboardResourceName = fmt.Sprintf("%s_%d", baseName, count)
			}

			// Add dashboard import block
			blocks = append(blocks, ImportBlock{
				ID: BuildImportID("elementum_dashboard", map[string]string{
					"dashboard_id": dashboard.ID,
				}),
				ResourceType: "elementum_dashboard",
				ResourceName: dashboardResourceName,
			})

			// Add widget import blocks
			widgetNames := make(map[string]int)
			for _, widget := range dashboard.Widgets {
				widgetBaseName := SanitizeName(widget.Name)
				if widgetBaseName == "" {
					widgetBaseName = "widget"
				}

				widgetCount := widgetNames[widgetBaseName]
				widgetNames[widgetBaseName]++

				widgetResourceName := dashboardResourceName + "_" + widgetBaseName
				if widgetCount > 0 {
					widgetResourceName = fmt.Sprintf("%s_%d", widgetResourceName, widgetCount)
				}

				blocks = append(blocks, ImportBlock{
					ID: BuildImportID("elementum_dashboard_widget", map[string]string{
						"dashboard_id": dashboard.ID,
						"widget_id":    widget.ID,
					}),
					ResourceType: "elementum_dashboard_widget",
					ResourceName: widgetResourceName,
				})
			}
		}
		widgetCount := 0
		for _, d := range app.Dashboards {
			widgetCount += len(d.Widgets)
		}
		logger.Debug("generated dashboard import blocks", "dashboardCount", len(app.Dashboards), "widgetCount", widgetCount)
	}

	// Datamines (discovered through automation dependencies)
	if len(app.ReferencedDatamines) > 0 {
		datamineNames := make(map[string]int)

		for _, datamine := range app.ReferencedDatamines {
			// Skip if already seen (can happen if same datamine in multiple lists)
			if seenResourceIDs[datamine.ID] {
				logger.Debug("skipping duplicate datamine", "id", datamine.ID, "name", datamine.Name)
				continue
			}
			seenResourceIDs[datamine.ID] = true

			datamineBaseName := SanitizeName(datamine.Name)
			datamineCount := datamineNames[datamineBaseName]
			datamineNames[datamineBaseName]++

			datamineName := datamineBaseName
			if datamineCount > 0 {
				datamineName = fmt.Sprintf("%s_%d", datamineBaseName, datamineCount)
			}

			// Datamine import requires table_id:datamine_id format
			importID := datamine.ID
			if datamine.TableID != "" {
				importID = fmt.Sprintf("%s:%s", datamine.TableID, datamine.ID)
			}

			blocks = append(blocks, ImportBlock{
				ID:           importID,
				ResourceType: "elementum_datamine",
				ResourceName: datamineName,
			})
		}
		logger.Debug("generated datamine import blocks", "count", len(app.ReferencedDatamines))
	}

	// Tables (discovered through automation dependencies)
	if len(app.ReferencedTables) > 0 {
		tableNames := make(map[string]int)

		for _, table := range app.ReferencedTables {
			// Skip if already seen (can happen if same table in multiple lists)
			if seenResourceIDs[table.ID] {
				logger.Debug("skipping duplicate table", "id", table.ID, "name", table.Name)
				continue
			}
			seenResourceIDs[table.ID] = true

			tableBaseName := SanitizeName(table.Name)
			tableCount := tableNames[tableBaseName]
			tableNames[tableBaseName]++

			tableName := tableBaseName
			if tableCount > 0 {
				tableName = fmt.Sprintf("%s_%d", tableBaseName, tableCount)
			}

			blocks = append(blocks, ImportBlock{
				ID:           table.ID,
				ResourceType: "elementum_table",
				ResourceName: tableName,
			})
		}
		logger.Debug("generated table import blocks", "count", len(app.ReferencedTables))

		// Table Search Tables from referenced tables
		for _, table := range app.ReferencedTables {
			for _, st := range table.SearchTables {
				name := SanitizeName(st.FieldName)
				if name == "" {
					name = "table_search"
				}
				blocks = append(blocks, ImportBlock{
					ID: BuildImportID("elementum_table_search_table", map[string]string{
						"table_id":        table.ID,
						"search_table_id": st.ID,
					}),
					ResourceType: "elementum_table_search_table",
					ResourceName: name,
				})
			}
		}
	}

	// Discovered apps from recursive discovery (full resource export)
	if len(app.DiscoveredApps) > 0 {
		for _, discoveredApp := range app.DiscoveredApps {
			blocks = append(blocks, generateDiscoveredAppFullBlocks(discoveredApp, selectedTypes, seenRelationshipIDs)...)
		}
		logger.Debug("generated discovered app full import blocks", "apps", len(app.DiscoveredApps))
	}

	// Discovered elements from recursive discovery
	// Generate full resource blocks for each discovered element (fields, layouts, automations, etc.)
	if len(app.DiscoveredElements) > 0 {
		for _, discoveredElement := range app.DiscoveredElements {
			blocks = append(blocks, generateDiscoveredElementFullBlocks(discoveredElement, selectedTypes, seenRelationshipIDs)...)
		}
		logger.Debug("generated discovered element import blocks", "elements", len(app.DiscoveredElements))
	}

	// Discovered tasks from recursive discovery (full resource export)
	// Generate full resource blocks for each discovered task (fields, layouts, automations, agents, views, etc.)
	// Note: Tasks don't support flows, approvals, or file readers
	if len(app.DiscoveredTasks) > 0 {
		for _, discoveredTask := range app.DiscoveredTasks {
			taskBlocks := generateDiscoveredTaskFullBlocks(discoveredTask, selectedTypes, seenRelationshipIDs)
			blocks = append(blocks, taskBlocks...)
		}
		logger.Debug("generated discovered task import blocks", "tasks", len(app.DiscoveredTasks))
	}

	// Discovered tables from recursive discovery
	if len(app.DiscoveredTables) > 0 {
		tableNames := make(map[string]int)

		for _, table := range app.DiscoveredTables {
			// Skip if already seen (can happen if same table in multiple lists)
			if seenResourceIDs[table.ID] {
				logger.Debug("skipping duplicate table", "id", table.ID, "name", table.Name)
				continue
			}
			seenResourceIDs[table.ID] = true

			tableBaseName := SanitizeName(table.Name)
			tableCount := tableNames[tableBaseName]
			tableNames[tableBaseName]++

			tableName := tableBaseName
			if tableCount > 0 {
				tableName = fmt.Sprintf("%s_%d", tableBaseName, tableCount)
			}

			blocks = append(blocks, ImportBlock{
				ID:           table.ID,
				ResourceType: "elementum_table",
				ResourceName: tableName,
			})
		}
		logger.Debug("generated discovered table import blocks", "count", len(app.DiscoveredTables))

		// Table Search Tables from discovered tables
		for _, table := range app.DiscoveredTables {
			for _, st := range table.SearchTables {
				name := SanitizeName(st.FieldName)
				if name == "" {
					name = "table_search"
				}
				blocks = append(blocks, ImportBlock{
					ID: BuildImportID("elementum_table_search_table", map[string]string{
						"table_id":        table.ID,
						"search_table_id": st.ID,
					}),
					ResourceType: "elementum_table_search_table",
					ResourceName: name,
				})
			}
		}
	}

	// Discovered datamines from recursive discovery
	// Import ID format: table_id:datamine_id (required by datamine Read function)
	if len(app.DiscoveredDatamines) > 0 {
		datamineNames := make(map[string]int)

		for _, datamine := range app.DiscoveredDatamines {
			// Skip if already seen (can happen if same datamine in multiple lists)
			if seenResourceIDs[datamine.ID] {
				logger.Debug("skipping duplicate datamine", "id", datamine.ID, "name", datamine.Name)
				continue
			}
			seenResourceIDs[datamine.ID] = true

			datamineBaseName := SanitizeName(datamine.Name)
			datamineCount := datamineNames[datamineBaseName]
			datamineNames[datamineBaseName]++

			datamineName := datamineBaseName
			if datamineCount > 0 {
				datamineName = fmt.Sprintf("%s_%d", datamineBaseName, datamineCount)
			}

			// Datamine import requires table_id:datamine_id format
			importID := datamine.ID
			if datamine.TableID != "" {
				importID = fmt.Sprintf("%s:%s", datamine.TableID, datamine.ID)
			}

			blocks = append(blocks, ImportBlock{
				ID:           importID,
				ResourceType: "elementum_datamine",
				ResourceName: datamineName,
			})
		}
		logger.Debug("generated discovered datamine import blocks", "count", len(app.DiscoveredDatamines))
	}

	return blocks
}

// generateDiscoveredAppFullBlocks generates import blocks for all resources from a discovered app
// This includes the app itself, fields, layouts, flows, automations, agents, roles, access policies, etc.
// seenRelationshipIDs is used to deduplicate bidirectional relationships
func generateDiscoveredAppFullBlocks(discoveredApp *discovery.App, selectedTypes map[string]bool, seenRelationshipIDs map[string]bool) []ImportBlock {
	blocks := []ImportBlock{}

	// Use Namespace for prefix if available, otherwise Name
	prefix := SanitizeName(discoveredApp.Namespace)
	if prefix == "" {
		prefix = SanitizeName(discoveredApp.Name)
	}

	// App resource itself
	blocks = append(blocks, ImportBlock{
		ID:           discoveredApp.ID,
		ResourceType: "elementum_app",
		ResourceName: prefix,
	})

	// Fields (if selected)
	if selectedTypes["fields"] && len(discoveredApp.Fields) > 0 {
		fieldNames := make(map[string]int)

		for _, field := range discoveredApp.Fields {
			if field.Type == "unknown" || field.Type == "" || field.Type == "handle" {
				continue
			}

			// Check for system fields (semantic tags or name patterns)
			// Discovered apps don't allow STATUS fields to be created
			if isSystemField(field, false) {
				continue
			}

			baseName := prefix + "_" + SanitizeName(field.Name)
			count := fieldNames[baseName]
			fieldNames[baseName]++

			name := baseName
			if count > 0 {
				name = fmt.Sprintf("%s_%d", baseName, count)
			}

			blocks = append(blocks, ImportBlock{
				ID:           BuildFieldImportID(discoveredApp.ID, field.ID, field.Type),
				ResourceType: GetResourceTypeForField(field),
				ResourceName: name,
			})
		}
	}

	// Layouts (if selected)
	if selectedTypes["layouts"] && len(discoveredApp.Layouts) > 0 {
		layoutNames := make(map[string]int)

		for _, layout := range discoveredApp.Layouts {
			// Skip Initiate layouts - they are platform-managed
			if layout.IsInitiate {
				continue
			}

			baseName := prefix + "_" + SanitizeName(layout.Name)
			count := layoutNames[baseName]
			layoutNames[baseName]++

			name := baseName
			if count > 0 {
				name = fmt.Sprintf("%s_%d", baseName, count)
			}

			blocks = append(blocks, ImportBlock{
				ID:           BuildImportID("elementum_layout", map[string]string{"app_id": discoveredApp.ID, "layout_id": layout.ID}),
				ResourceType: "elementum_layout",
				ResourceName: name,
			})
		}
	}

	// Flows (if selected) - Apps support flows
	if selectedTypes["flows"] && len(discoveredApp.Flows) > 0 {
		flowNames := make(map[string]int)

		for _, flow := range discoveredApp.Flows {
			baseName := prefix + "_" + SanitizeName(flow.Name)
			count := flowNames[baseName]
			flowNames[baseName]++

			name := baseName
			if count > 0 {
				name = fmt.Sprintf("%s_%d", baseName, count)
			}

			blocks = append(blocks, ImportBlock{
				ID:           BuildImportID("elementum_flow", map[string]string{"app_id": discoveredApp.ID, "flow_id": flow.ID}),
				ResourceType: "elementum_flow",
				ResourceName: name,
			})
		}
	}

	// Automations (if selected)
	if selectedTypes["automations"] && len(discoveredApp.Automations) > 0 {
		blocks = append(blocks, generateDiscoveredAppAutomationBlocks(discoveredApp)...)
	}

	// Agents (if selected) - Apps support agents
	if selectedTypes["agents"] && len(discoveredApp.Agents) > 0 {
		agentNames := make(map[string]int)
		toolNames := make(map[string]int)

		for _, agent := range discoveredApp.Agents {
			baseName := prefix + "_" + SanitizeName(agent.Name)
			count := agentNames[baseName]
			agentNames[baseName]++

			name := baseName
			if count > 0 {
				name = fmt.Sprintf("%s_%d", baseName, count)
			}

			blocks = append(blocks, ImportBlock{
				ID:           BuildImportID("elementum_agent", map[string]string{"app_id": discoveredApp.ID, "agent_id": agent.ID}),
				ResourceType: "elementum_agent",
				ResourceName: name,
			})

			// Agent tools - use tool-specific resource types
			for _, tool := range agent.Tools {
				resourceType := GetResourceTypeForAgentTool(&tool)
				if resourceType == "" {
					// Unknown tool type, skip
					continue
				}

				toolBaseName := prefix + "_" + SanitizeName(fmt.Sprintf("%s_%s", agent.Name, tool.Name))
				toolCount := toolNames[toolBaseName]
				toolNames[toolBaseName]++

				toolName := toolBaseName
				if toolCount > 0 {
					toolName = fmt.Sprintf("%s_%d", toolBaseName, toolCount)
				}

				blocks = append(blocks, ImportBlock{
					ID: BuildImportID(resourceType, map[string]string{
						"app_id":   discoveredApp.ID,
						"agent_id": agent.ID,
						"tool_id":  tool.ID,
					}),
					ResourceType: resourceType,
					ResourceName: toolName,
				})
			}
		}
	}

	// Widgets (if selected)
	if selectedTypes["widgets"] && len(discoveredApp.Widgets) > 0 {
		widgetNames := make(map[string]int)

		for _, widget := range discoveredApp.Widgets {
			baseName := prefix + "_" + SanitizeName(widget.Name)
			count := widgetNames[baseName]
			widgetNames[baseName]++

			name := baseName
			if count > 0 {
				name = fmt.Sprintf("%s_%d", baseName, count)
			}

			blocks = append(blocks, ImportBlock{
				ID:           BuildImportID("elementum_widget", map[string]string{"app_id": discoveredApp.ID, "widget_id": widget.ID}),
				ResourceType: "elementum_widget",
				ResourceName: name,
			})
		}
	}

	// Views (if selected) - Apps support views
	if selectedTypes["views"] && len(discoveredApp.Views) > 0 {
		viewNames := make(map[string]int)

		for _, view := range discoveredApp.Views {
			// Skip agent views if agents aren't being exported (they would reference undeclared agents)
			if view.Type == "AspectViewAgent" && !selectedTypes["agents"] {
				continue
			}

			baseName := prefix + "_" + SanitizeName(view.Name)
			count := viewNames[baseName]
			viewNames[baseName]++

			name := baseName
			if count > 0 {
				name = fmt.Sprintf("%s_%d", baseName, count)
			}

			blocks = append(blocks, ImportBlock{
				ID:           fmt.Sprintf("%s:%s", discoveredApp.ID, view.ID),
				ResourceType: view.TerraformResourceType(),
				ResourceName: name,
			})
		}
	}

	// File Readers (if selected) - Apps support file readers (AI, OCR, JSON, XML)
	if selectedTypes["ai_file_readers"] && len(discoveredApp.AIFileReaders) > 0 {
		fileReaderNames := make(map[string]int)

		for _, reader := range discoveredApp.AIFileReaders {
			baseName := prefix + "_" + SanitizeName(reader.Name)
			count := fileReaderNames[baseName]
			fileReaderNames[baseName]++

			name := baseName
			if count > 0 {
				name = fmt.Sprintf("%s_%d", baseName, count)
			}

			resourceType := reader.TerraformResourceType()

			blocks = append(blocks, ImportBlock{
				ID: BuildImportID(resourceType, map[string]string{
					"app_id":         discoveredApp.ID,
					"file_reader_id": reader.ID,
				}),
				ResourceType: resourceType,
				ResourceName: name,
			})
		}
	}

	// Approval Processes (if selected) - Apps support approval processes
	if selectedTypes["approval_processes"] && len(discoveredApp.Approvals) > 0 {
		approvalNames := make(map[string]int)

		for _, approval := range discoveredApp.Approvals {
			baseName := prefix + "_" + SanitizeName(approval.Name)
			count := approvalNames[baseName]
			approvalNames[baseName]++

			name := baseName
			if count > 0 {
				name = fmt.Sprintf("%s_%d", baseName, count)
			}

			blocks = append(blocks, ImportBlock{
				ID: BuildImportID("elementum_approval_process", map[string]string{
					"app_id":      discoveredApp.ID,
					"approval_id": approval.ID,
				}),
				ResourceType: "elementum_approval_process",
				ResourceName: name,
			})
		}
	}

	// Relationships (if selected)
	if selectedTypes["relationships"] && len(discoveredApp.Relationships) > 0 {
		relationshipNames := make(map[string]int)

		for _, rel := range discoveredApp.Relationships {
			// Skip if this relationship was already exported
			if seenRelationshipIDs[rel.ID] {
				continue
			}
			seenRelationshipIDs[rel.ID] = true

			baseName := prefix + "_to_" + SanitizeName(rel.RelatedObjectName)
			if rel.RelatedObjectName == "" {
				baseName = prefix + "_relationship"
			}
			count := relationshipNames[baseName]
			relationshipNames[baseName]++

			name := baseName
			if count > 0 {
				name = fmt.Sprintf("%s_%d", baseName, count)
			}

			blocks = append(blocks, ImportBlock{
				ID: BuildImportID("elementum_relationship", map[string]string{
					"object_id":       discoveredApp.ID,
					"relationship_id": rel.ID,
				}),
				ResourceType: "elementum_relationship",
				ResourceName: name,
			})
		}
	}

	// Roles (custom roles only - if selected)
	if selectedTypes["roles"] && len(discoveredApp.Roles) > 0 {
		roleNames := make(map[string]int)

		for _, role := range discoveredApp.Roles {
			// Skip managed roles - they're exported as data sources, not resources
			if role.Managed {
				logger.Debug("skipping managed role for import (will be data source)", "app", discoveredApp.Name, "name", role.Name, "id", role.ID)
				continue
			}

			baseName := prefix + "_" + SanitizeName(role.Name)
			count := roleNames[baseName]
			roleNames[baseName]++

			name := baseName
			if count > 0 {
				name = fmt.Sprintf("%s_%d", baseName, count)
			}

			blocks = append(blocks, ImportBlock{
				ID: BuildImportID("elementum_role", map[string]string{
					"object_id": discoveredApp.ID,
					"role_id":   role.ID,
				}),
				ResourceType: "elementum_role",
				ResourceName: name,
			})
		}
		logger.Debug("generated discovered app role import blocks", "app", discoveredApp.Name, "roles", len(roleNames))
	}

	// Access Policies (if selected)
	if selectedTypes["access_policies"] && len(discoveredApp.AccessPolicies) > 0 {
		accessPolicyNames := make(map[string]int)

		for _, policy := range discoveredApp.AccessPolicies {
			baseName := prefix + "_access_policy"
			count := accessPolicyNames[baseName]
			accessPolicyNames[baseName]++

			name := baseName
			if count > 0 {
				name = fmt.Sprintf("%s_%d", baseName, count)
			}

			blocks = append(blocks, ImportBlock{
				ID: BuildImportID("elementum_access_policy", map[string]string{
					"object_id": discoveredApp.ID,
					"policy_id": policy.ID,
				}),
				ResourceType: "elementum_access_policy",
				ResourceName: name,
			})
		}
		logger.Debug("generated discovered app access policy import blocks", "app", discoveredApp.Name, "policies", len(accessPolicyNames))
	}

	return blocks
}

// generateDiscoveredAppAutomationBlocks generates import blocks for automations from a discovered app
func generateDiscoveredAppAutomationBlocks(discoveredApp *discovery.App) []ImportBlock {
	blocks := []ImportBlock{}

	// Use Handle for prefix if available, otherwise Name
	prefix := SanitizeName(discoveredApp.Namespace)
	if prefix == "" {
		prefix = SanitizeName(discoveredApp.Name)
	}

	automationNames := make(map[string]int)
	triggerNames := make(map[string]int)
	taskNames := make(map[string]int)

	for _, automation := range discoveredApp.Automations {
		// Skip unpublished or inactive automations
		if !automation.HasPublished || automation.Status != "ACTIVE" {
			continue
		}

		// Automation itself (prefixed with discovered app handle)
		autoBaseName := prefix + "_" + SanitizeName(automation.Name)
		autoCount := automationNames[autoBaseName]
		automationNames[autoBaseName]++

		autoName := autoBaseName
		if autoCount > 0 {
			autoName = fmt.Sprintf("%s_%d", autoBaseName, autoCount)
		}

		blocks = append(blocks, ImportBlock{
			ID:           BuildImportID("elementum_automation", map[string]string{"app_id": discoveredApp.ID, "automation_id": automation.ID}),
			ResourceType: "elementum_automation",
			ResourceName: autoName,
		})

		// Workflow publish
		blocks = append(blocks, ImportBlock{
			ID:           BuildImportID("elementum_workflow_publish", map[string]string{"automation_id": automation.ID}),
			ResourceType: "elementum_workflow_publish",
			ResourceName: autoName,
		})

		// Triggers
		for i, trigger := range automation.Triggers {
			if trigger.Type == "unknown" {
				continue
			}

			triggerBaseName := SanitizeName(fmt.Sprintf("%s_%s_%s_%d", prefix, automation.Name, trigger.Type, i))
			triggerCount := triggerNames[triggerBaseName]
			triggerNames[triggerBaseName]++

			triggerName := triggerBaseName
			if triggerCount > 0 {
				triggerName = fmt.Sprintf("%s_%d", triggerBaseName, triggerCount)
			}

			blocks = append(blocks, ImportBlock{
				ID:           BuildTriggerImportID(automation.ID, trigger.ID, trigger.Type),
				ResourceType: GetResourceTypeForTrigger(trigger),
				ResourceName: triggerName,
			})
		}

		// Tasks
		for _, task := range automation.Tasks {
			if task.Type == "unknown" {
				continue
			}

			taskBaseName := SanitizeName(fmt.Sprintf("%s_%s_%s", prefix, automation.Name, task.Name))
			taskCount := taskNames[taskBaseName]
			taskNames[taskBaseName]++

			taskName := taskBaseName
			if taskCount > 0 {
				taskName = fmt.Sprintf("%s_%d", taskBaseName, taskCount)
			}

			blocks = append(blocks, ImportBlock{
				ID:           BuildTaskImportID(automation.WorkflowID, task.ID, task.Type),
				ResourceType: GetResourceTypeForTask(task),
				ResourceName: taskName,
			})
		}
	}

	return blocks
}

// generateDiscoveredElementFullBlocks generates import blocks for all resources from a discovered element
// This includes the element itself, fields, layouts, widgets, automations, and relationships
// seenRelationshipIDs is used to deduplicate bidirectional relationships that appear from both app and element sides
func generateDiscoveredElementFullBlocks(discoveredElement *discovery.Element, selectedTypes map[string]bool, seenRelationshipIDs map[string]bool) []ImportBlock {
	blocks := []ImportBlock{}

	// Use Namespace for prefix if available, otherwise Handle, otherwise Name
	prefix := SanitizeName(discoveredElement.Namespace)
	if prefix == "" {
		prefix = SanitizeName(discoveredElement.Handle)
	}
	if prefix == "" {
		prefix = SanitizeName(discoveredElement.Name)
	}

	// Element resource itself
	blocks = append(blocks, ImportBlock{
		ID:           discoveredElement.ID,
		ResourceType: "elementum_element",
		ResourceName: prefix,
	})

	// Fields (if selected)
	if selectedTypes["fields"] && len(discoveredElement.Fields) > 0 {
		fieldNames := make(map[string]int)

		for _, field := range discoveredElement.Fields {
			// Skip fields that cannot be managed via Terraform
			if field.Type == "unknown" || field.Type == "" || field.Type == "handle" {
				continue
			}

			// Check for system fields (semantic tags or name patterns)
			// Elements allow STATUS fields since they may be custom dropdowns
			if isSystemField(field, true) {
				continue
			}

			baseName := prefix + "_" + SanitizeName(field.Name)
			count := fieldNames[baseName]
			fieldNames[baseName]++

			name := baseName
			if count > 0 {
				name = fmt.Sprintf("%s_%d", baseName, count)
			}

			blocks = append(blocks, ImportBlock{
				ID:           BuildFieldImportID(discoveredElement.ID, field.ID, field.Type),
				ResourceType: GetResourceTypeForField(field),
				ResourceName: name,
			})
		}
	}

	// Layouts (if selected)
	// Note: Elements don't have stages in the same way as apps - they typically have an "Initiate" layout
	// The import ID format is element_id:layout_id (same as app_id:stage_id)
	if selectedTypes["layouts"] && len(discoveredElement.Layouts) > 0 {
		layoutNames := make(map[string]int)

		for _, layout := range discoveredElement.Layouts {
			// Use <namespace>_layout naming for element layouts
			baseName := prefix + "_layout"
			count := layoutNames[baseName]
			layoutNames[baseName]++

			name := baseName
			if count > 0 {
				name = fmt.Sprintf("%s_%d", baseName, count)
			}

			// Layout import ID format: object_id:stage_id (element_id:layout_id)
			blocks = append(blocks, ImportBlock{
				ID:           BuildImportID("elementum_layout", map[string]string{"app_id": discoveredElement.ID, "layout_id": layout.ID}),
				ResourceType: "elementum_layout",
				ResourceName: name,
			})
		}
	}

	// Widgets (if selected)
	if selectedTypes["widgets"] && len(discoveredElement.Widgets) > 0 {
		widgetNames := make(map[string]int)

		for _, widget := range discoveredElement.Widgets {
			baseName := prefix + "_" + SanitizeName(widget.Name)
			count := widgetNames[baseName]
			widgetNames[baseName]++

			name := baseName
			if count > 0 {
				name = fmt.Sprintf("%s_%d", baseName, count)
			}

			blocks = append(blocks, ImportBlock{
				ID:           BuildImportID("elementum_widget", map[string]string{"app_id": discoveredElement.ID, "widget_id": widget.ID}),
				ResourceType: "elementum_widget",
				ResourceName: name,
			})
		}
	}

	// Views (if selected) - Elements support views
	if selectedTypes["views"] && len(discoveredElement.Views) > 0 {
		viewNames := make(map[string]int)

		for _, view := range discoveredElement.Views {
			// Skip agent views if agents aren't being exported (they would reference undeclared agents)
			if view.Type == "AspectViewAgent" && !selectedTypes["agents"] {
				continue
			}

			baseName := prefix + "_" + SanitizeName(view.Name)
			count := viewNames[baseName]
			viewNames[baseName]++

			name := baseName
			if count > 0 {
				name = fmt.Sprintf("%s_%d", baseName, count)
			}

			blocks = append(blocks, ImportBlock{
				ID:           fmt.Sprintf("%s:%s", discoveredElement.ID, view.ID),
				ResourceType: view.TerraformResourceType(),
				ResourceName: name,
			})
		}
	}

	// Automations (if selected)
	if selectedTypes["automations"] && len(discoveredElement.Automations) > 0 {
		blocks = append(blocks, generateDiscoveredElementAutomationBlocks(discoveredElement)...)
	}

	// Relationships (if selected)
	if selectedTypes["relationships"] && len(discoveredElement.Relationships) > 0 {
		relationshipNames := make(map[string]int)

		for _, rel := range discoveredElement.Relationships {
			// Skip if this relationship was already exported from the app or another element
			if seenRelationshipIDs[rel.ID] {
				logger.Debug("skipping duplicate relationship", "id", rel.ID, "element", discoveredElement.Name, "related", rel.RelatedObjectName)
				continue
			}
			seenRelationshipIDs[rel.ID] = true

			baseName := prefix + "_to_" + SanitizeName(rel.RelatedObjectName)
			count := relationshipNames[baseName]
			relationshipNames[baseName]++

			name := baseName
			if count > 0 {
				name = fmt.Sprintf("%s_%d", baseName, count)
			}

			blocks = append(blocks, ImportBlock{
				ID:           BuildImportID("elementum_relationship", map[string]string{"object_id": discoveredElement.ID, "relationship_id": rel.ID}),
				ResourceType: "elementum_relationship",
				ResourceName: name,
			})
		}
	}

	// Roles (custom roles only - if selected)
	if selectedTypes["roles"] && len(discoveredElement.Roles) > 0 {
		roleNames := make(map[string]int)

		for _, role := range discoveredElement.Roles {
			// Skip managed roles - they're exported as data sources, not resources
			if role.Managed {
				logger.Debug("skipping managed role for import (will be data source)", "element", discoveredElement.Name, "name", role.Name, "id", role.ID)
				continue
			}

			baseName := prefix + "_" + SanitizeName(role.Name)
			count := roleNames[baseName]
			roleNames[baseName]++

			name := baseName
			if count > 0 {
				name = fmt.Sprintf("%s_%d", baseName, count)
			}

			blocks = append(blocks, ImportBlock{
				ID: BuildImportID("elementum_role", map[string]string{
					"object_id": discoveredElement.ID,
					"role_id":   role.ID,
				}),
				ResourceType: "elementum_role",
				ResourceName: name,
			})
		}
		logger.Debug("generated element role import blocks", "element", discoveredElement.Name, "roles", len(roleNames))
	}

	// Access Policies (if selected)
	if selectedTypes["access_policies"] && len(discoveredElement.AccessPolicies) > 0 {
		accessPolicyNames := make(map[string]int)

		for _, policy := range discoveredElement.AccessPolicies {
			baseName := prefix + "_access_policy"
			count := accessPolicyNames[baseName]
			accessPolicyNames[baseName]++

			name := baseName
			if count > 0 {
				name = fmt.Sprintf("%s_%d", baseName, count)
			}

			blocks = append(blocks, ImportBlock{
				ID: BuildImportID("elementum_access_policy", map[string]string{
					"object_id": discoveredElement.ID,
					"policy_id": policy.ID,
				}),
				ResourceType: "elementum_access_policy",
				ResourceName: name,
			})
		}
		logger.Debug("generated element access policy import blocks", "element", discoveredElement.Name, "policies", len(accessPolicyNames))
	}

	// AI Search Tables (only Elements have these, not Apps)
	if selectedTypes["ai_search_tables"] && len(discoveredElement.AISearchTables) > 0 {
		searchTableNames := make(map[string]int)

		for _, searchTable := range discoveredElement.AISearchTables {
			// Use field name as base name if available, otherwise use element prefix
			baseName := prefix + "_ai_search"
			if searchTable.FieldName != "" {
				baseName = prefix + "_" + SanitizeName(searchTable.FieldName)
			}

			count := searchTableNames[baseName]
			searchTableNames[baseName]++

			name := baseName
			if count > 0 {
				name = fmt.Sprintf("%s_%d", baseName, count)
			}

			blocks = append(blocks, ImportBlock{
				ID: BuildImportID("elementum_ai_search_table", map[string]string{
					"object_id":       discoveredElement.ID,
					"search_table_id": searchTable.ID,
				}),
				ResourceType: "elementum_ai_search_table",
				ResourceName: name,
			})
		}
		logger.Debug("generated element AI search table import blocks", "element", discoveredElement.Name, "searchTables", len(searchTableNames))
	}

	// Dashboards (and their widgets) for this element
	if selectedTypes["dashboards"] && len(discoveredElement.Dashboards) > 0 {
		dashboardNames := make(map[string]int)

		for _, dashboard := range discoveredElement.Dashboards {
			baseName := prefix + "_" + SanitizeName(dashboard.Name)
			if baseName == prefix+"_" {
				baseName = prefix + "_dashboard"
			}
			count := dashboardNames[baseName]
			dashboardNames[baseName]++

			dashboardResourceName := baseName
			if count > 0 {
				dashboardResourceName = fmt.Sprintf("%s_%d", baseName, count)
			}

			// Add dashboard import block
			blocks = append(blocks, ImportBlock{
				ID: BuildImportID("elementum_dashboard", map[string]string{
					"dashboard_id": dashboard.ID,
				}),
				ResourceType: "elementum_dashboard",
				ResourceName: dashboardResourceName,
			})

			// Add widget import blocks for this dashboard
			widgetNames := make(map[string]int)
			for _, widget := range dashboard.Widgets {
				widgetBaseName := dashboardResourceName + "_" + SanitizeName(widget.Name)
				if widgetBaseName == dashboardResourceName+"_" {
					widgetBaseName = dashboardResourceName + "_widget"
				}
				widgetCount := widgetNames[widgetBaseName]
				widgetNames[widgetBaseName]++

				widgetResourceName := widgetBaseName
				if widgetCount > 0 {
					widgetResourceName = fmt.Sprintf("%s_%d", widgetBaseName, widgetCount)
				}

				blocks = append(blocks, ImportBlock{
					ID: BuildImportID("elementum_dashboard_widget", map[string]string{
						"dashboard_id": dashboard.ID,
						"widget_id":    widget.ID,
					}),
					ResourceType: "elementum_dashboard_widget",
					ResourceName: widgetResourceName,
				})
			}
		}
		widgetCount := 0
		for _, d := range discoveredElement.Dashboards {
			widgetCount += len(d.Widgets)
		}
		logger.Debug("generated element dashboard import blocks", "element", discoveredElement.Name, "dashboardCount", len(discoveredElement.Dashboards), "widgetCount", widgetCount)
	}

	// Charts for this element's dashboards
	if selectedTypes["charts"] && len(discoveredElement.Charts) > 0 {
		chartNames := make(map[string]int)

		for _, chart := range discoveredElement.Charts {
			baseName := prefix + "_" + SanitizeName(chart.Name)
			if baseName == prefix+"_" {
				baseName = prefix + "_chart"
			}
			count := chartNames[baseName]
			chartNames[baseName]++

			chartResourceName := baseName
			if count > 0 {
				chartResourceName = fmt.Sprintf("%s_%d", baseName, count)
			}

			blocks = append(blocks, ImportBlock{
				ID: BuildImportID("elementum_chart", map[string]string{
					"chart_id": chart.ID,
				}),
				ResourceType: "elementum_chart",
				ResourceName: chartResourceName,
			})
		}
		logger.Debug("generated element chart import blocks", "element", discoveredElement.Name, "chartCount", len(discoveredElement.Charts))
	}

	return blocks
}

// generateDiscoveredTaskFullBlocks generates import blocks for all resources from a discovered task
// This includes the task itself, fields, layouts, automations, agents, views, and relationships
// Note: Tasks don't support flows, approvals, or file readers (only Apps do)
// seenRelationshipIDs is used to deduplicate bidirectional relationships
func generateDiscoveredTaskFullBlocks(discoveredTask *discovery.AspectTask, selectedTypes map[string]bool, seenRelationshipIDs map[string]bool) []ImportBlock {
	blocks := []ImportBlock{}

	// Use Namespace for prefix if available, otherwise Name
	prefix := SanitizeName(discoveredTask.Namespace)
	if prefix == "" {
		prefix = SanitizeName(discoveredTask.Name)
	}

	// Task resource itself
	blocks = append(blocks, ImportBlock{
		ID:           discoveredTask.ID,
		ResourceType: "elementum_task",
		ResourceName: prefix,
	})

	// Fields (if selected)
	if selectedTypes["fields"] && len(discoveredTask.Fields) > 0 {
		fieldNames := make(map[string]int)

		for _, field := range discoveredTask.Fields {
			// Skip fields that cannot be managed via Terraform
			if field.Type == "unknown" || field.Type == "" || field.Type == "handle" {
				continue
			}

			// Check for system fields (semantic tags or name patterns)
			// Tasks don't allow STATUS fields to be created
			if isSystemField(field, false) {
				continue
			}

			baseName := prefix + "_" + SanitizeName(field.Name)
			count := fieldNames[baseName]
			fieldNames[baseName]++

			name := baseName
			if count > 0 {
				name = fmt.Sprintf("%s_%d", baseName, count)
			}

			blocks = append(blocks, ImportBlock{
				ID:           BuildFieldImportID(discoveredTask.ID, field.ID, field.Type),
				ResourceType: GetResourceTypeForField(field),
				ResourceName: name,
			})
		}
	}

	// Layouts (if selected)
	if selectedTypes["layouts"] && len(discoveredTask.Layouts) > 0 {
		layoutNames := make(map[string]int)

		for _, layout := range discoveredTask.Layouts {
			baseName := prefix + "_" + SanitizeName(layout.Name)
			count := layoutNames[baseName]
			layoutNames[baseName]++

			name := baseName
			if count > 0 {
				name = fmt.Sprintf("%s_%d", baseName, count)
			}

			blocks = append(blocks, ImportBlock{
				ID:           BuildImportID("elementum_layout", map[string]string{"app_id": discoveredTask.ID, "layout_id": layout.ID}),
				ResourceType: "elementum_layout",
				ResourceName: name,
			})
		}
	}

	// Widgets (if selected)
	if selectedTypes["widgets"] && len(discoveredTask.Widgets) > 0 {
		widgetNames := make(map[string]int)

		for _, widget := range discoveredTask.Widgets {
			baseName := prefix + "_" + SanitizeName(widget.Name)
			count := widgetNames[baseName]
			widgetNames[baseName]++

			name := baseName
			if count > 0 {
				name = fmt.Sprintf("%s_%d", baseName, count)
			}

			blocks = append(blocks, ImportBlock{
				ID:           BuildImportID("elementum_widget", map[string]string{"app_id": discoveredTask.ID, "widget_id": widget.ID}),
				ResourceType: "elementum_widget",
				ResourceName: name,
			})
		}
	}

	// Views (if selected) - Tasks support views
	if selectedTypes["views"] && len(discoveredTask.Views) > 0 {
		viewNames := make(map[string]int)

		for _, view := range discoveredTask.Views {
			// Skip agent views if agents aren't being exported (they would reference undeclared agents)
			if view.Type == "AspectViewAgent" && !selectedTypes["agents"] {
				continue
			}

			baseName := prefix + "_" + SanitizeName(view.Name)
			count := viewNames[baseName]
			viewNames[baseName]++

			name := baseName
			if count > 0 {
				name = fmt.Sprintf("%s_%d", baseName, count)
			}

			blocks = append(blocks, ImportBlock{
				ID:           fmt.Sprintf("%s:%s", discoveredTask.ID, view.ID),
				ResourceType: view.TerraformResourceType(),
				ResourceName: name,
			})
		}
	}

	// Automations (if selected)
	if selectedTypes["automations"] && len(discoveredTask.Automations) > 0 {
		blocks = append(blocks, generateDiscoveredTaskAutomationBlocks(discoveredTask)...)
	}

	// Agents (if selected) - Tasks support agents (unlike Elements)
	if selectedTypes["agents"] && len(discoveredTask.Agents) > 0 {
		agentNames := make(map[string]int)
		toolNames := make(map[string]int)

		for _, agent := range discoveredTask.Agents {
			baseName := prefix + "_" + SanitizeName(agent.Name)
			count := agentNames[baseName]
			agentNames[baseName]++

			name := baseName
			if count > 0 {
				name = fmt.Sprintf("%s_%d", baseName, count)
			}

			blocks = append(blocks, ImportBlock{
				ID:           BuildImportID("elementum_agent", map[string]string{"app_id": discoveredTask.ID, "agent_id": agent.ID}),
				ResourceType: "elementum_agent",
				ResourceName: name,
			})

			// Agent tools - use tool-specific resource types
			for _, tool := range agent.Tools {
				resourceType := GetResourceTypeForAgentTool(&tool)
				if resourceType == "" {
					// Unknown tool type, skip
					continue
				}

				toolBaseName := prefix + "_" + SanitizeName(fmt.Sprintf("%s_%s", agent.Name, tool.Name))
				toolCount := toolNames[toolBaseName]
				toolNames[toolBaseName]++

				toolName := toolBaseName
				if toolCount > 0 {
					toolName = fmt.Sprintf("%s_%d", toolBaseName, toolCount)
				}

				blocks = append(blocks, ImportBlock{
					ID:           BuildImportID(resourceType, map[string]string{"agent_id": agent.ID, "tool_id": tool.ID}),
					ResourceType: resourceType,
					ResourceName: toolName,
				})
			}
		}
	}

	// Relationships (if selected)
	if selectedTypes["relationships"] && len(discoveredTask.Relationships) > 0 {
		relationshipNames := make(map[string]int)

		for _, rel := range discoveredTask.Relationships {
			// Skip if this relationship was already exported from another object
			if seenRelationshipIDs[rel.ID] {
				logger.Debug("skipping duplicate relationship", "id", rel.ID, "task", discoveredTask.Name, "related", rel.RelatedObjectName)
				continue
			}
			seenRelationshipIDs[rel.ID] = true

			baseName := prefix + "_to_" + SanitizeName(rel.RelatedObjectName)
			count := relationshipNames[baseName]
			relationshipNames[baseName]++

			name := baseName
			if count > 0 {
				name = fmt.Sprintf("%s_%d", baseName, count)
			}

			blocks = append(blocks, ImportBlock{
				ID:           BuildImportID("elementum_relationship", map[string]string{"object_id": discoveredTask.ID, "relationship_id": rel.ID}),
				ResourceType: "elementum_relationship",
				ResourceName: name,
			})
		}
	}

	// Roles (custom roles only - if selected)
	if selectedTypes["roles"] && len(discoveredTask.Roles) > 0 {
		roleNames := make(map[string]int)

		for _, role := range discoveredTask.Roles {
			// Skip managed roles - they're exported as data sources, not resources
			if role.Managed {
				logger.Debug("skipping managed role for import (will be data source)", "task", discoveredTask.Name, "name", role.Name, "id", role.ID)
				continue
			}

			baseName := prefix + "_" + SanitizeName(role.Name)
			count := roleNames[baseName]
			roleNames[baseName]++

			name := baseName
			if count > 0 {
				name = fmt.Sprintf("%s_%d", baseName, count)
			}

			blocks = append(blocks, ImportBlock{
				ID: BuildImportID("elementum_role", map[string]string{
					"object_id": discoveredTask.ID,
					"role_id":   role.ID,
				}),
				ResourceType: "elementum_role",
				ResourceName: name,
			})
		}
		logger.Debug("generated task role import blocks", "task", discoveredTask.Name, "roles", len(roleNames))
	}

	// Access Policies (if selected)
	if selectedTypes["access_policies"] && len(discoveredTask.AccessPolicies) > 0 {
		accessPolicyNames := make(map[string]int)

		for _, policy := range discoveredTask.AccessPolicies {
			baseName := prefix + "_access_policy"
			count := accessPolicyNames[baseName]
			accessPolicyNames[baseName]++

			name := baseName
			if count > 0 {
				name = fmt.Sprintf("%s_%d", baseName, count)
			}

			blocks = append(blocks, ImportBlock{
				ID: BuildImportID("elementum_access_policy", map[string]string{
					"object_id": discoveredTask.ID,
					"policy_id": policy.ID,
				}),
				ResourceType: "elementum_access_policy",
				ResourceName: name,
			})
		}
		logger.Debug("generated task access policy import blocks", "task", discoveredTask.Name, "policies", len(accessPolicyNames))
	}

	return blocks
}

// generateDiscoveredTaskAutomationBlocks generates import blocks for automations from a discovered task
func generateDiscoveredTaskAutomationBlocks(discoveredTask *discovery.AspectTask) []ImportBlock {
	blocks := []ImportBlock{}

	// Use Namespace for prefix if available, otherwise Name
	prefix := SanitizeName(discoveredTask.Namespace)
	if prefix == "" {
		prefix = SanitizeName(discoveredTask.Name)
	}

	automationNames := make(map[string]int)
	triggerNames := make(map[string]int)
	taskNames := make(map[string]int)

	for _, automation := range discoveredTask.Automations {
		// Skip unpublished or inactive automations
		if !automation.HasPublished || automation.Status != "ACTIVE" {
			continue
		}

		// Automation itself (prefixed with discovered task namespace)
		autoBaseName := prefix + "_" + SanitizeName(automation.Name)
		autoCount := automationNames[autoBaseName]
		automationNames[autoBaseName]++

		autoName := autoBaseName
		if autoCount > 0 {
			autoName = fmt.Sprintf("%s_%d", autoBaseName, autoCount)
		}

		blocks = append(blocks, ImportBlock{
			ID:           BuildImportID("elementum_automation", map[string]string{"app_id": discoveredTask.ID, "workflow_id": automation.WorkflowID}),
			ResourceType: "elementum_automation",
			ResourceName: autoName,
		})

		// Workflow publish
		blocks = append(blocks, ImportBlock{
			ID:           BuildImportID("elementum_workflow_publish", map[string]string{"automation_id": automation.ID}),
			ResourceType: "elementum_workflow_publish",
			ResourceName: autoName,
		})

		// Triggers
		for _, trigger := range automation.Triggers {
			triggerBaseName := autoName + "_" + SanitizeName(trigger.Type)
			triggerCount := triggerNames[triggerBaseName]
			triggerNames[triggerBaseName]++

			triggerName := triggerBaseName
			if triggerCount > 0 {
				triggerName = fmt.Sprintf("%s_%d", triggerBaseName, triggerCount)
			}

			blocks = append(blocks, ImportBlock{
				ID:           BuildTriggerImportID(automation.WorkflowID, trigger.ID, trigger.Type),
				ResourceType: GetResourceTypeForTrigger(trigger),
				ResourceName: triggerName,
			})
		}

		// Tasks
		for _, task := range automation.Tasks {
			taskBaseName := autoName + "_" + SanitizeName(task.Name)
			taskCount := taskNames[taskBaseName]
			taskNames[taskBaseName]++

			taskName := taskBaseName
			if taskCount > 0 {
				taskName = fmt.Sprintf("%s_%d", taskBaseName, taskCount)
			}

			blocks = append(blocks, ImportBlock{
				ID:           BuildTaskImportID(automation.WorkflowID, task.ID, task.Type),
				ResourceType: GetResourceTypeForTask(task),
				ResourceName: taskName,
			})
		}
	}

	return blocks
}

// generateDiscoveredElementAutomationBlocks generates import blocks for automations from a discovered element
func generateDiscoveredElementAutomationBlocks(discoveredElement *discovery.Element) []ImportBlock {
	blocks := []ImportBlock{}

	// Use Handle for prefix if available, otherwise Name
	prefix := SanitizeName(discoveredElement.Handle)
	if prefix == "" {
		prefix = SanitizeName(discoveredElement.Name)
	}

	automationNames := make(map[string]int)
	triggerNames := make(map[string]int)
	taskNames := make(map[string]int)

	for _, automation := range discoveredElement.Automations {
		// Skip unpublished or inactive automations
		if !automation.HasPublished || automation.Status != "ACTIVE" {
			continue
		}

		// Automation itself (prefixed with discovered element handle)
		autoBaseName := prefix + "_" + SanitizeName(automation.Name)
		autoCount := automationNames[autoBaseName]
		automationNames[autoBaseName]++

		autoName := autoBaseName
		if autoCount > 0 {
			autoName = fmt.Sprintf("%s_%d", autoBaseName, autoCount)
		}

		// For elements, we use element_id instead of app_id in the import ID
		blocks = append(blocks, ImportBlock{
			ID:           BuildImportID("elementum_automation", map[string]string{"element_id": discoveredElement.ID, "automation_id": automation.ID}),
			ResourceType: "elementum_automation",
			ResourceName: autoName,
		})

		// Workflow publish
		blocks = append(blocks, ImportBlock{
			ID:           BuildImportID("elementum_workflow_publish", map[string]string{"automation_id": automation.ID}),
			ResourceType: "elementum_workflow_publish",
			ResourceName: autoName,
		})

		// Triggers
		for i, trigger := range automation.Triggers {
			if trigger.Type == "unknown" {
				continue
			}

			triggerBaseName := SanitizeName(fmt.Sprintf("%s_%s_%s_%d", prefix, automation.Name, trigger.Type, i))
			triggerCount := triggerNames[triggerBaseName]
			triggerNames[triggerBaseName]++

			triggerName := triggerBaseName
			if triggerCount > 0 {
				triggerName = fmt.Sprintf("%s_%d", triggerBaseName, triggerCount)
			}

			blocks = append(blocks, ImportBlock{
				ID:           BuildTriggerImportID(automation.ID, trigger.ID, trigger.Type),
				ResourceType: GetResourceTypeForTrigger(trigger),
				ResourceName: triggerName,
			})
		}

		// Tasks
		for _, task := range automation.Tasks {
			if task.Type == "unknown" {
				continue
			}

			taskBaseName := SanitizeName(fmt.Sprintf("%s_%s_%s", prefix, automation.Name, task.Name))
			taskCount := taskNames[taskBaseName]
			taskNames[taskBaseName]++

			taskName := taskBaseName
			if taskCount > 0 {
				taskName = fmt.Sprintf("%s_%d", taskBaseName, taskCount)
			}

			blocks = append(blocks, ImportBlock{
				ID:           BuildTaskImportID(automation.WorkflowID, task.ID, task.Type),
				ResourceType: GetResourceTypeForTask(task),
				ResourceName: taskName,
			})
		}
	}

	return blocks
}

// GenerateCategoryCloudLinkDataSources generates data source HCL blocks for category and cloudlink only.
// This should be called unconditionally since category and cloudlink are app-level properties
// that are needed regardless of whether relationships are selected for export.
func GenerateCategoryCloudLinkDataSources(app *discovery.App) string {
	var sb strings.Builder

	// Track generated categories and cloudlinks to avoid duplicates
	generatedCategories := make(map[string]bool)
	generatedCloudLinks := make(map[string]bool)

	// Generate category data source if app has a category
	if app.CategoryID != "" && app.CategoryName != "" {
		sb.WriteString("# Category data source\n")
		catName := SanitizeName(app.CategoryName)
		fmt.Fprintf(&sb, "data \"elementum_category\" %q {\n", catName)
		fmt.Fprintf(&sb, "  name = %q\n", app.CategoryName)
		sb.WriteString("}\n\n")
		generatedCategories[app.CategoryID] = true
	}

	// Generate cloudlink data source if app has a cloudlink
	if app.CloudLinkID != "" && app.CloudLinkName != "" {
		sb.WriteString("# CloudLink data source\n")
		cloudLinkName := SanitizeName(app.CloudLinkName)
		fmt.Fprintf(&sb, "data \"elementum_cloudlink\" %q {\n", cloudLinkName)
		fmt.Fprintf(&sb, "  name = %q\n", app.CloudLinkName)
		sb.WriteString("}\n\n")
		generatedCloudLinks[app.CloudLinkID] = true
	}

	// Generate category and cloudlink data sources for discovered apps
	for _, discoveredApp := range app.DiscoveredApps {
		// Generate category data source if discovered app has a category not already generated
		if discoveredApp.CategoryID != "" && discoveredApp.CategoryName != "" && !generatedCategories[discoveredApp.CategoryID] {
			catName := SanitizeName(discoveredApp.CategoryName)
			fmt.Fprintf(&sb, "data \"elementum_category\" %q {\n", catName)
			fmt.Fprintf(&sb, "  name = %q\n", discoveredApp.CategoryName)
			sb.WriteString("}\n\n")
			generatedCategories[discoveredApp.CategoryID] = true
		}

		// Generate cloudlink data source if discovered app has a cloudlink not already generated
		if discoveredApp.CloudLinkID != "" && discoveredApp.CloudLinkName != "" && !generatedCloudLinks[discoveredApp.CloudLinkID] {
			cloudLinkName := SanitizeName(discoveredApp.CloudLinkName)
			fmt.Fprintf(&sb, "data \"elementum_cloudlink\" %q {\n", cloudLinkName)
			fmt.Fprintf(&sb, "  name = %q\n", discoveredApp.CloudLinkName)
			sb.WriteString("}\n\n")
			generatedCloudLinks[discoveredApp.CloudLinkID] = true
		}
	}

	// Generate category and cloudlink data sources for discovered elements
	for _, elem := range app.DiscoveredElements {
		// Generate category data source if element has a category not already generated
		if elem.CategoryID != "" && elem.CategoryName != "" && !generatedCategories[elem.CategoryID] {
			catName := SanitizeName(elem.CategoryName)
			fmt.Fprintf(&sb, "data \"elementum_category\" %q {\n", catName)
			fmt.Fprintf(&sb, "  name = %q\n", elem.CategoryName)
			sb.WriteString("}\n\n")
			generatedCategories[elem.CategoryID] = true
		}

		// Generate cloudlink data source if element has a cloudlink not already generated
		if elem.CloudLinkID != "" && elem.CloudLinkName != "" && !generatedCloudLinks[elem.CloudLinkID] {
			cloudLinkName := SanitizeName(elem.CloudLinkName)
			fmt.Fprintf(&sb, "data \"elementum_cloudlink\" %q {\n", cloudLinkName)
			fmt.Fprintf(&sb, "  name = %q\n", elem.CloudLinkName)
			sb.WriteString("}\n\n")
			generatedCloudLinks[elem.CloudLinkID] = true
		}
	}

	return sb.String()
}

// GenerateRelationshipDataSources generates data source HCL blocks for related objects
// that are referenced but not exported as resources.
// NOTE: Discovered apps are exported as resources, not data sources.
func GenerateRelationshipDataSources(app *discovery.App) string {
	if len(app.RelatedObjects) == 0 {
		return ""
	}

	var sb strings.Builder

	// Build sets of IDs that are being exported as resources
	// Elements in DiscoveredElements with their own fields/resources are exported as resources
	discoveredElementIDs := make(map[string]bool)
	for _, elem := range app.DiscoveredElements {
		discoveredElementIDs[elem.ID] = true
	}

	// Apps in DiscoveredApps are exported as resources, not data sources
	discoveredAppIDs := make(map[string]bool)
	for _, discoveredApp := range app.DiscoveredApps {
		discoveredAppIDs[discoveredApp.ID] = true
	}

	// Tasks in DiscoveredTasks are exported as resources, not data sources
	discoveredTaskIDs := make(map[string]bool)
	for _, task := range app.DiscoveredTasks {
		discoveredTaskIDs[task.ID] = true
	}

	// Track what we've already generated data sources for
	generatedAppIDs := make(map[string]bool)
	generatedElementIDs := make(map[string]bool)

	sb.WriteString("# Data sources for related objects (discovered through relationships and automations)\n")
	sb.WriteString("# These allow referencing fields from related apps/elements\n\n")

	relatedNames := make(map[string]int)
	hasContent := false

	// Generate data sources for RelatedObjects
	for _, relatedObj := range app.RelatedObjects {
		// Check all skip conditions BEFORE incrementing the counter
		// This ensures consistent naming with buildUUIDMap
		switch relatedObj.Type {
		case "App":
			// Skip apps that are being exported as resources
			if discoveredAppIDs[relatedObj.ID] {
				continue
			}
			// Skip duplicate IDs
			if generatedAppIDs[relatedObj.ID] {
				continue
			}
		case "Element":
			// Skip elements that are being exported as resources
			if discoveredElementIDs[relatedObj.ID] {
				continue
			}
			// Skip duplicate IDs
			if generatedElementIDs[relatedObj.ID] {
				continue
			}
		case "Task":
			// Skip tasks that are being exported as resources from DiscoveredTasks
			if discoveredTaskIDs[relatedObj.ID] {
				continue
			}
		}

		// Increment counter only for items that will actually generate a data source
		baseName := SanitizeName(relatedObj.Name)
		count := relatedNames[baseName]
		relatedNames[baseName]++

		name := baseName
		if count > 0 {
			name = fmt.Sprintf("%s_%d", baseName, count)
		}

		switch relatedObj.Type {
		case "App":
			fmt.Fprintf(&sb, "data \"elementum_app\" %q {\n", name)
			fmt.Fprintf(&sb, "  name = %q\n", relatedObj.Name)
			sb.WriteString("}\n\n")
			hasContent = true
			generatedAppIDs[relatedObj.ID] = true
		case "Element":
			fmt.Fprintf(&sb, "data \"elementum_element\" %q {\n", name)
			fmt.Fprintf(&sb, "  name = %q\n", relatedObj.Name)
			sb.WriteString("}\n\n")
			hasContent = true
			generatedElementIDs[relatedObj.ID] = true
		case "Task":
			fmt.Fprintf(&sb, "data \"elementum_task\" %q {\n", name)
			fmt.Fprintf(&sb, "  name = %q\n", relatedObj.Name)
			sb.WriteString("}\n\n")
			hasContent = true
		}
	}

	// NOTE: Discovered apps and tasks are exported as resources (not data sources)
	// via generateDiscoveredAppFullBlocks and generateDiscoveredTaskFullBlocks

	if !hasContent {
		return ""
	}

	return sb.String()
}

// GenerateDataSourceBlocks generates data source HCL blocks for category, cloudlink, and related objects.
// This is a convenience function that combines GenerateCategoryCloudLinkDataSources and GenerateRelationshipDataSources.
// NOTE: Prefer using the individual functions for finer control over what gets generated.
func GenerateDataSourceBlocks(app *discovery.App) string {
	return GenerateCategoryCloudLinkDataSources(app) + GenerateRelationshipDataSources(app)
}

// GenerateRoleDataSourceBlocks generates data source HCL for managed roles, users, and groups
// This is used when roles are selected for export
// It collects users/groups from the main app, discovered apps, and discovered elements
func GenerateRoleDataSourceBlocks(app *discovery.App, appResourceRef string) string {
	var sb strings.Builder

	// Track users and groups needed for data sources (from custom roles across all aspects)
	usersNeeded := make(map[string]string)  // ID -> email
	groupsNeeded := make(map[string]string) // ID -> name

	// Helper function to collect users/groups from a set of roles
	collectRoleMembers := func(roles []discovery.Role) {
		for _, role := range roles {
			if role.Managed {
				continue // Only custom roles have membership we need to export
			}
			for _, user := range role.Users {
				if user.Name != "" { // email
					usersNeeded[user.ID] = user.Name
				}
			}
			for _, group := range role.Groups {
				if group.Name != "" {
					groupsNeeded[group.ID] = group.Name
				}
			}
		}
	}

	// Collect from main app
	collectRoleMembers(app.Roles)

	// Collect from discovered apps
	for _, discoveredApp := range app.DiscoveredApps {
		collectRoleMembers(discoveredApp.Roles)
	}

	// Collect from discovered elements
	for _, discoveredElement := range app.DiscoveredElements {
		collectRoleMembers(discoveredElement.Roles)
	}

	// Generate user data sources
	if len(usersNeeded) > 0 {
		sb.WriteString("# User data sources for role membership\n")
		userNames := make(map[string]int)
		for _, email := range usersNeeded {
			baseName := sanitizeEmailToName(email)
			count := userNames[baseName]
			userNames[baseName]++

			name := baseName
			if count > 0 {
				name = fmt.Sprintf("%s_%d", baseName, count)
			}

			fmt.Fprintf(&sb, "data \"elementum_user\" %q {\n", name)
			fmt.Fprintf(&sb, "  email = %q\n", email)
			sb.WriteString("}\n\n")
		}
	}

	// Generate group data sources
	if len(groupsNeeded) > 0 {
		sb.WriteString("# Group data sources for role membership\n")
		groupNames := make(map[string]int)
		for _, groupName := range groupsNeeded {
			baseName := SanitizeName(groupName)
			count := groupNames[baseName]
			groupNames[baseName]++

			name := baseName
			if count > 0 {
				name = fmt.Sprintf("%s_%d", baseName, count)
			}

			fmt.Fprintf(&sb, "data \"elementum_group\" %q {\n", name)
			fmt.Fprintf(&sb, "  name = %q\n", groupName)
			sb.WriteString("}\n\n")
		}
	}

	// Generate managed role data sources for main app
	managedRoleCount := 0
	for _, role := range app.Roles {
		if role.Managed {
			managedRoleCount++
		}
	}

	if managedRoleCount > 0 {
		sb.WriteString("# Managed role data sources (system-created roles)\n")
		roleNames := make(map[string]int)
		for _, role := range app.Roles {
			if !role.Managed {
				continue
			}

			baseName := SanitizeName(role.Name)
			count := roleNames[baseName]
			roleNames[baseName]++

			name := baseName
			if count > 0 {
				name = fmt.Sprintf("%s_%d", baseName, count)
			}

			fmt.Fprintf(&sb, "data \"elementum_role\" %q {\n", name)
			fmt.Fprintf(&sb, "  object_id = %s\n", appResourceRef)
			fmt.Fprintf(&sb, "  name      = %q\n", role.Name)
			sb.WriteString("}\n\n")
		}
	}

	// Generate managed role data sources for discovered apps
	for _, discoveredApp := range app.DiscoveredApps {
		appManagedCount := 0
		for _, role := range discoveredApp.Roles {
			if role.Managed {
				appManagedCount++
			}
		}

		if appManagedCount > 0 {
			prefix := SanitizeName(discoveredApp.Namespace)
			if prefix == "" {
				prefix = SanitizeName(discoveredApp.Name)
			}

			fmt.Fprintf(&sb, "# Managed role data sources for discovered app: %s\n", discoveredApp.Name)
			roleNames := make(map[string]int)
			for _, role := range discoveredApp.Roles {
				if !role.Managed {
					continue
				}

				baseName := prefix + "_" + SanitizeName(role.Name)
				count := roleNames[baseName]
				roleNames[baseName]++

				name := baseName
				if count > 0 {
					name = fmt.Sprintf("%s_%d", baseName, count)
				}

				fmt.Fprintf(&sb, "data \"elementum_role\" %q {\n", name)
				fmt.Fprintf(&sb, "  object_id = elementum_app.%s.id\n", prefix)
				fmt.Fprintf(&sb, "  name      = %q\n", role.Name)
				sb.WriteString("}\n\n")
			}
		}
	}

	// Note: Discovered element role data sources are generated in their respective element files
	// via generateElementRoleDataSources() in file_writer.go

	return sb.String()
}

// sanitizeEmailToName converts an email to a valid Terraform resource name
func sanitizeEmailToName(email string) string {
	// Take the part before @ and sanitize it
	parts := strings.Split(email, "@")
	if len(parts) > 0 {
		return SanitizeName(parts[0])
	}
	return SanitizeName(email)
}

// GenerateAiProviderConnectorDataSources generates data source HCL for discovered AI provider connectors.
// These connectors are discovered from:
// - AI tasks in automations (ai_file_read, ai_classify, run_agent, etc.)
// - AI agents (elementum, snowflake, bedrock, browser_use agents)
func GenerateAiProviderConnectorDataSources(connectors []discovery.AiProviderConnector) string {
	if len(connectors) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("# AI Provider Connector data sources (discovered from AI tasks and agents)\n")

	nameCounter := make(map[string]int)
	for _, conn := range connectors {
		if conn.ModelName == "" {
			continue // Skip if no model name - can't generate lookup
		}

		baseName := SanitizeName(conn.ModelName)
		count := nameCounter[baseName]
		nameCounter[baseName]++

		name := baseName
		if count > 0 {
			name = fmt.Sprintf("%s_%d", baseName, count)
		}

		fmt.Fprintf(&sb, "data \"elementum_ai_provider_connector\" %q {\n", name)
		fmt.Fprintf(&sb, "  model_name = %q\n", conn.ModelName)
		sb.WriteString("}\n\n")
	}

	return sb.String()
}

// GenerateCloudLinkDataSourcesFromMap generates data source HCL for discovered CloudLinks
func GenerateCloudLinkDataSourcesFromMap(cloudlinks map[string]*discovery.CloudLink) string {
	if len(cloudlinks) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("# CloudLink data sources (discovered from tables)\n")

	cloudlinkNames := make(map[string]int)
	for _, cl := range cloudlinks {
		baseName := SanitizeName(cl.Name)
		count := cloudlinkNames[baseName]
		cloudlinkNames[baseName]++

		name := baseName
		if count > 0 {
			name = fmt.Sprintf("%s_%d", baseName, count)
		}

		fmt.Fprintf(&sb, "data \"elementum_cloudlink\" %q {\n", name)
		fmt.Fprintf(&sb, "  name = %q\n", cl.Name)
		sb.WriteString("}\n\n")
	}

	return sb.String()
}

// GenerateStoredFunctionDataSources generates data source HCL for discovered stored functions.
// These functions are discovered from procedure tasks in automations.
func GenerateStoredFunctionDataSources(functions []*discovery.StoredFunction) string {
	if len(functions) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("# Stored Function data sources (discovered from procedure tasks)\n")

	nameCounter := make(map[string]int)
	for _, sf := range functions {
		if sf.DisplayName == "" {
			continue // Skip if no display name - can't generate lookup
		}

		baseName := SanitizeName(sf.DisplayName)
		count := nameCounter[baseName]
		nameCounter[baseName]++

		name := baseName
		if count > 0 {
			name = fmt.Sprintf("%s_%d", baseName, count)
		}

		fmt.Fprintf(&sb, "data \"elementum_stored_function\" %q {\n", name)
		fmt.Fprintf(&sb, "  cloudlink_id = data.elementum_cloudlink.%s.id\n", SanitizeName(sf.CloudLinkName))
		fmt.Fprintf(&sb, "  name         = %q\n", sf.DisplayName)
		sb.WriteString("}\n\n")
	}

	return sb.String()
}

// RenderImportBlocks renders import blocks as Terraform HCL
func RenderImportBlocks(blocks []ImportBlock) string {
	var sb strings.Builder

	sb.WriteString("# Generated by elementum\n")
	sb.WriteString("# Import these resources with: terraform plan -generate-config-out=generated.tf\n\n")

	for _, block := range blocks {
		sb.WriteString("import {\n")
		fmt.Fprintf(&sb, "  id = %q\n", block.ID)
		fmt.Fprintf(&sb, "  to = %s.%s\n", block.ResourceType, block.ResourceName)
		sb.WriteString("}\n\n")
	}

	return sb.String()
}

// RenderProviderConfig renders a provider configuration block
func RenderProviderConfig(organization, instance, environment, clientID, clientSecret string) string {
	var sb strings.Builder

	sb.WriteString("# Provider configuration\n")
	sb.WriteString("# Credentials are set from CLI configuration\n\n")

	sb.WriteString("terraform {\n")
	sb.WriteString("  required_providers {\n")
	sb.WriteString("    elementum = {\n")
	sb.WriteString("      source = \"elementumltd/elementum\"\n")
	sb.WriteString("    }\n")
	sb.WriteString("  }\n")
	sb.WriteString("}\n\n")

	sb.WriteString("provider \"elementum\" {\n")
	fmt.Fprintf(&sb, "  organization  = %q\n", organization)
	// Only include instance if it's not "us" (the default)
	if instance != "" && instance != "us" {
		fmt.Fprintf(&sb, "  instance      = %q\n", instance)
	}
	// Only include environment if it's set (some orgs don't use environments)
	if environment != "" {
		fmt.Fprintf(&sb, "  environment   = %q\n", environment)
	}
	fmt.Fprintf(&sb, "  client_id     = %q\n", clientID)
	fmt.Fprintf(&sb, "  client_secret = %q\n", clientSecret)
	sb.WriteString("}\n\n")

	return sb.String()
}

// RenderSimpleProviderConfig renders a provider configuration that uses environment variables
// This is suitable for multi-file export where credentials come from the elementum CLI
func RenderSimpleProviderConfig() string {
	var sb strings.Builder

	sb.WriteString("terraform {\n")
	sb.WriteString("  required_version = \">= 1.6.0\"\n\n")
	sb.WriteString("  required_providers {\n")
	sb.WriteString("    elementum = {\n")
	sb.WriteString("      source = \"elementumltd/elementum\"\n")
	sb.WriteString("    }\n")
	sb.WriteString("  }\n")
	sb.WriteString("}\n\n")

	sb.WriteString("# Provider uses environment variables automatically:\n")
	sb.WriteString("# ELEMENTUM_ORGANIZATION, ELEMENTUM_CLIENT_ID, ELEMENTUM_CLIENT_SECRET, ELEMENTUM_ENVIRONMENT\n")
	sb.WriteString("# These are injected by elementum when running plan/apply/destroy\n")
	sb.WriteString("provider \"elementum\" {}\n")

	return sb.String()
}

// SplitImportBlocks separates import blocks into simple (terraform-generated) and automation (CLI-generated) blocks.
// Simple resources like app, fields, layouts, flows are handled by terraform's generate-config-out.
// Complex automation resources (automations, triggers, tasks) are generated directly by the CLI.
func SplitImportBlocks(blocks []ImportBlock) (simpleBlocks, automationBlocks []ImportBlock) {
	for _, block := range blocks {
		if IsAutomationRelated(block.ResourceType) {
			automationBlocks = append(automationBlocks, block)
		} else {
			simpleBlocks = append(simpleBlocks, block)
		}
	}
	return
}

// IsAutomationRelated checks if a resource type is automation-related (automation, trigger, or task).
// These resources are generated directly by the CLI rather than terraform's generate-config-out
// because their ImportState doesn't populate the full schema needed for config generation.
func IsAutomationRelated(resourceType string) bool {
	// elementum_task is an AspectTask resource (standalone task), NOT a workflow task.
	// It should be handled by terraform's generate-config-out, not the CLI.
	if resourceType == "elementum_task" {
		return false
	}
	return resourceType == "elementum_automation" ||
		strings.HasSuffix(resourceType, "_trigger") ||
		strings.HasSuffix(resourceType, "_task")
}

// graphqlToolTypeToTerraformResource maps GraphQL tool type names to Terraform resource types
var graphqlToolTypeToTerraformResourceType = map[string]string{
	"AgentCreateRecordTool":    "elementum_agent_create_record_tool",
	"AgentSearchAspectTool":    "elementum_agent_search_records_tool",
	"AgentUpdateRecordTool":    "elementum_agent_update_record_tool",
	"AgentSearchTableTool":     "elementum_agent_ai_search_tool",
	"AgentRelateRecordTool":    "elementum_agent_relate_record_tool",
	"AgentExecuteWorkflowTool": "elementum_agent_run_automation_tool",
	"AgentRunAgentTool":        "elementum_agent_run_agent_tool",
	"AgentSelectBotRouteTool":  "elementum_agent_select_bot_route_tool",
	"AgentMcpTool":             "elementum_agent_mcp_tool",
}

// GetResourceTypeForAgentTool returns the Terraform resource type for an agent tool
func GetResourceTypeForAgentTool(tool *discovery.AgentTool) string {
	if resourceType, ok := graphqlToolTypeToTerraformResourceType[tool.Type]; ok {
		return resourceType
	}
	// Fallback for unknown types - log and skip
	logger.Debug("unknown agent tool type", "type", tool.Type, "name", tool.Name)
	return ""
}

// SanitizeName converts a name to a valid Terraform resource name
func SanitizeName(name string) string {
	// Replace invalid characters with underscores
	name = strings.ToLower(name)
	name = strings.ReplaceAll(name, " ", "_")
	name = strings.ReplaceAll(name, "-", "_")
	name = strings.ReplaceAll(name, ".", "_")
	name = strings.ReplaceAll(name, "/", "_")
	name = strings.ReplaceAll(name, ":", "_")

	// Remove any non-alphanumeric characters except underscores
	var result strings.Builder
	for _, char := range name {
		if (char >= 'a' && char <= 'z') || (char >= '0' && char <= '9') || char == '_' {
			result.WriteRune(char)
		}
	}

	name = result.String()

	// Ensure it doesn't start with a number
	if len(name) > 0 && name[0] >= '0' && name[0] <= '9' {
		name = "_" + name
	}

	// Ensure it's not empty
	if name == "" {
		name = "resource"
	}

	return name
}

// AppResourceName returns the canonical resource name for an app using namespace-first logic.
// This ensures consistent naming across all export code:
// - If Namespace is set, use it (clean identifier)
// - Otherwise fall back to sanitized Name
func AppResourceName(app *discovery.App) string {
	if app.Namespace != "" {
		return SanitizeName(app.Namespace)
	}
	return SanitizeName(app.Name)
}

// ElementResourceName returns the canonical resource name for an element using namespace-first logic.
// Priority: Namespace > Handle > Name
func ElementResourceName(elem *discovery.Element) string {
	if elem.Namespace != "" {
		return SanitizeName(elem.Namespace)
	}
	if elem.Handle != "" {
		return SanitizeName(elem.Handle)
	}
	return SanitizeName(elem.Name)
}

// TaskResourceName returns the canonical resource name for a task using namespace-first logic.
// Priority: Namespace > Handle > Name
func TaskResourceName(task *discovery.AspectTask) string {
	if task.Namespace != "" {
		return SanitizeName(task.Namespace)
	}
	if task.Handle != "" {
		return SanitizeName(task.Handle)
	}
	return SanitizeName(task.Name)
}

// GenerateAspectAutomationImportBlocks generates import blocks for automations from any aspect type (App, Element, or Task)
// This is a generic function that can be used by element and task exports.
// Parameters:
// - aspectID: the ID of the aspect (app, element, or task)
// - aspectType: "app", "element", or "task" (used for import ID key)
// - prefix: resource name prefix (typically the aspect's handle or sanitized name)
// - automations: the list of automations to generate import blocks for
func GenerateAspectAutomationImportBlocks(aspectID string, aspectType string, prefix string, automations []discovery.Automation) []ImportBlock {
	blocks := []ImportBlock{}

	automationNames := make(map[string]int)
	triggerNames := make(map[string]int)
	taskNames := make(map[string]int)

	// Determine the ID key based on aspect type
	idKey := "app_id"
	switch aspectType {
	case "element":
		idKey = "element_id"
	case "task":
		idKey = "task_id"
	}

	for _, automation := range automations {
		// Skip unpublished or inactive automations
		if !automation.HasPublished || automation.Status != "ACTIVE" {
			logger.Debug("skipping automation",
				"name", automation.Name,
				"id", automation.ID,
				"status", automation.Status,
				"hasPublished", automation.HasPublished,
			)
			continue
		}

		// Automation itself (prefixed with aspect handle/name)
		autoBaseName := prefix + "_" + SanitizeName(automation.Name)
		autoCount := automationNames[autoBaseName]
		automationNames[autoBaseName]++

		autoName := autoBaseName
		if autoCount > 0 {
			autoName = fmt.Sprintf("%s_%d", autoBaseName, autoCount)
		}

		blocks = append(blocks, ImportBlock{
			ID:           BuildImportID("elementum_automation", map[string]string{idKey: aspectID, "automation_id": automation.ID}),
			ResourceType: "elementum_automation",
			ResourceName: autoName,
		})

		// Workflow publish
		blocks = append(blocks, ImportBlock{
			ID:           BuildImportID("elementum_workflow_publish", map[string]string{"automation_id": automation.ID}),
			ResourceType: "elementum_workflow_publish",
			ResourceName: autoName,
		})

		// Triggers
		for i, trigger := range automation.Triggers {
			if trigger.Type == "unknown" {
				logger.Debug("skipping unknown trigger type", "name", trigger.Name, "reason", "GraphQL typename not mapped")
				continue
			}

			triggerBaseName := SanitizeName(fmt.Sprintf("%s_%s_%s_%d", prefix, automation.Name, trigger.Type, i))
			triggerCount := triggerNames[triggerBaseName]
			triggerNames[triggerBaseName]++

			triggerName := triggerBaseName
			if triggerCount > 0 {
				triggerName = fmt.Sprintf("%s_%d", triggerBaseName, triggerCount)
			}

			blocks = append(blocks, ImportBlock{
				ID:           BuildTriggerImportID(automation.ID, trigger.ID, trigger.Type),
				ResourceType: GetResourceTypeForTrigger(trigger),
				ResourceName: triggerName,
			})
		}

		// Tasks
		for _, task := range automation.Tasks {
			if task.Type == "unknown" {
				logger.Debug("skipping unknown task type", "name", task.Name, "reason", "GraphQL typename not mapped")
				continue
			}

			taskBaseName := SanitizeName(fmt.Sprintf("%s_%s_%s", prefix, automation.Name, task.Name))
			taskCount := taskNames[taskBaseName]
			taskNames[taskBaseName]++

			taskName := taskBaseName
			if taskCount > 0 {
				taskName = fmt.Sprintf("%s_%d", taskBaseName, taskCount)
			}

			blocks = append(blocks, ImportBlock{
				ID:           BuildTaskImportID(automation.WorkflowID, task.ID, task.Type),
				ResourceType: GetResourceTypeForTask(task),
				ResourceName: taskName,
			})
		}
	}

	return blocks
}

// GenerateDataSourceBlocksForElement generates data source HCL blocks for element dependencies
// This includes category and cloudlink data sources needed for element resource references
func GenerateDataSourceBlocksForElement(elem *discovery.Element) string {
	var sb strings.Builder

	// Generate category data source if element has a category
	if elem.CategoryID != "" && elem.CategoryName != "" {
		sb.WriteString("# Category data source\n")
		catName := SanitizeName(elem.CategoryName)
		fmt.Fprintf(&sb, "data \"elementum_category\" %q {\n", catName)
		fmt.Fprintf(&sb, "  name = %q\n", elem.CategoryName)
		sb.WriteString("}\n\n")
	}

	// Generate cloudlink data source if element has a cloudlink
	if elem.CloudLinkID != "" && elem.CloudLinkName != "" {
		sb.WriteString("# CloudLink data source\n")
		cloudLinkName := SanitizeName(elem.CloudLinkName)
		fmt.Fprintf(&sb, "data \"elementum_cloudlink\" %q {\n", cloudLinkName)
		fmt.Fprintf(&sb, "  name = %q\n", elem.CloudLinkName)
		sb.WriteString("}\n\n")
	}

	return sb.String()
}

// GenerateElementSystemFieldDataSources generates data source HCL blocks for element system fields
// The elementum_element resource only exposes title_field_id and id_field_id, but layouts can reference
// other system fields (created_by, created_at, etc.). This generates data sources for those fields.
func GenerateElementSystemFieldDataSources(app *discovery.App) string {
	if len(app.DiscoveredElements) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("# Element system field data sources\n")
	sb.WriteString("# These are needed because elementum_element doesn't expose audit field IDs\n\n")

	// System field semantic tags and their human-readable names
	systemFieldNames := map[string]string{
		"CREATED_BY": "Created By",
		"CREATED_AT": "Created At",
		"UPDATED_BY": "Updated By",
		"UPDATED_AT": "Updated At",
		"CLOSED_BY":  "Closed By",
		"CLOSED_AT":  "Closed At",
	}

	hasContent := false
	for _, element := range app.DiscoveredElements {
		// Use Namespace for prefix if available, otherwise Handle, otherwise Name
		prefix := SanitizeName(element.Namespace)
		if prefix == "" {
			prefix = SanitizeName(element.Handle)
		}
		if prefix == "" {
			prefix = SanitizeName(element.Name)
		}

		// Track which system fields are actually referenced in this element's layouts
		neededFields := make(map[string]bool)
		for _, field := range element.Fields {
			for _, tag := range field.SemanticTags {
				upperTag := strings.ToUpper(tag)
				if _, isSystemField := systemFieldNames[upperTag]; isSystemField {
					neededFields[upperTag] = true
				}
			}
		}

		// Generate data sources for the system fields that exist on this element
		for tag, fieldName := range systemFieldNames {
			if neededFields[tag] {
				dsName := prefix + "_" + strings.ToLower(strings.ReplaceAll(tag, "_", "_"))
				fmt.Fprintf(&sb, "data \"elementum_field\" %q {\n", dsName)
				fmt.Fprintf(&sb, "  object_id = elementum_element.%s.id\n", prefix)
				fmt.Fprintf(&sb, "  name      = %q\n", fieldName)
				sb.WriteString("}\n\n")
				hasContent = true
			}
		}
	}

	if !hasContent {
		return ""
	}

	return sb.String()
}
