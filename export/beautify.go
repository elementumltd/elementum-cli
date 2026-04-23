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
	"regexp"
	"strings"

	"github.com/elementumltd/elementum-cli/discovery"
)

// Pre-compiled regex patterns for performance
// These are compiled once at package initialization instead of on each function call
var (
	// Common patterns used across multiple functions
	nullPattern         = regexp.MustCompile(`^\s+\w+\s*=\s*null\s*$`)
	uuidPattern         = regexp.MustCompile(`"([0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12})"`)
	emptyActionPattern  = regexp.MustCompile(`(?m)^\s+action\s*=\s*\{\}\s*$`)
	multiNewlinePattern = regexp.MustCompile(`\n\n\n+`)

	// Resource declaration patterns
	resourceDeclPatternAlt = regexp.MustCompile(`^resource\s+"elementum_(\w+)"\s+"([^"]+)"`)

	// Field/Array patterns
	primaryColIDsPattern = regexp.MustCompile(`(primary_column_ids\s*=\s*)\[([^\]]+)\]`)

	// Flow patterns
	flowPattern             = regexp.MustCompile(`(?s)resource "elementum_flow" "([^"]+)" \{[^}]+object_id\s*=\s*([^\n]+)\n[^}]*stages\s*=\s*null[^}]*\}`)
	topLevelObjectIDPattern = regexp.MustCompile(`(?m)^(\s{2,4})object_id\s*=\s*"([^"]+)"`)
	nestedObjectIDPattern   = regexp.MustCompile(`(?m)^(\s{5,})object_id\s*=\s*"([^"]+)"`)

	// ID patterns for beautification
	automationIDPattern     = regexp.MustCompile(`automation_id\s*=\s*"([0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12})"`)
	automationIDPatternAny  = regexp.MustCompile(`automation_id\s*=\s*"([^"]+)"`)
	objectIDInObjectPattern = regexp.MustCompile(`(\s+object\s*=\s*\{[^}]*object_id\s*=\s*)"([0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12})"`)
	agentIDPattern          = regexp.MustCompile(`agent_id\s*=\s*"([0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12})"`)
	agentIDPatternAny       = regexp.MustCompile(`agent_id\s*=\s*"([^"]+)"`)

	// Value reference patterns
	// Pattern for record-based trigger refs: trigger.record.<UUID> or trigger.record.<UUID>.<suffix>
	triggerRefPattern = regexp.MustCompile(`((?:value|record_reference|value_reference)\s*=\s*)"(trigger\.record\.[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}(?:\.[a-zA-Z_]+)?)"`)
	// Pattern for on_demand trigger parameter refs: trigger.<paramName> (no "record." prefix, no UUID)
	triggerParamRefPattern = regexp.MustCompile(`((?:value|record_reference|value_reference)\s*=\s*)"(trigger\.[a-zA-Z][a-zA-Z0-9_]*)"`)
	taskRefPatternGlobal   = regexp.MustCompile(`((?:value|record_reference|value_reference)\s*=\s*)"(task\.[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}\.[^"]+)"`)

	// Table patterns
	tableResourcePattern     = regexp.MustCompile(`resource "elementum_table" "([^"]+)"`)
	tableCloudMappingPattern = regexp.MustCompile(`(?s)(resource "elementum_table" "[^"]+" \{[^}]*?cloud_mapping_cloudlink_id\s*=\s*[^\n]+)(\n)`)
	tableOpenPattern         = regexp.MustCompile(`(resource "elementum_table" "[^"]+" \{)(\n)`)
	referenceFieldIDPattern  = regexp.MustCompile(`(?m)^\s*reference_field_id\s*=\s*[^\n]+\n?`)
)

// PostProcessHCL applies remaining string-based transforms that haven't been converted to IR yet.
// This is called after the IR pipeline serializes blocks to HCL text, ensuring full parity
// with the old BeautifyConfig pipeline.
func PostProcessHCL(hcl string, imports []ImportBlock, app *discovery.App) string {
	return PostProcessHCLWithUUIDMap(hcl, imports, app, nil)
}

// PostProcessHCLWithUUIDMap is like PostProcessHCL but accepts a pre-built uuidMap
// to avoid rebuilding it on every call (expensive for multi-file exports with many files).
func PostProcessHCLWithUUIDMap(hcl string, imports []ImportBlock, app *discovery.App, uuidMap map[string]string) string {
	if uuidMap == nil {
		uuidMap = buildUUIDMap(imports, app)
	}

	// Beautify flow actions (resolve UUIDs in flow action blocks)
	hcl = beautifyFlowActions(hcl, uuidMap)

	// Clean up empty action blocks in flows
	hcl = stripEmptyFlowActions(hcl)

	// Inject hydrated app configs (adds status_options to elementum_app resource blocks)
	if app != nil {
		hcl = InjectHydratedAppConfigs(hcl, app, imports)
	}

	// Beautify value references in automation tasks (converts raw JSON refs to refs["Name"] syntax)
	// Belt-and-suspenders: CLI generators already produce correct refs, but this catches
	// any tofu-generated blocks that might contain raw value reference strings.
	hcl = beautifyValueReferences(hcl, uuidMap, app)

	// NOTE: beautifyAutomationIDs was removed here. It performed O(automations × tasks × uuidMap_size)
	// strings.ReplaceAll operations per file, causing ~9 minute runtime for large exports.
	// The IR pipeline's ResolveUUIDs transform already handles UUID→reference resolution
	// at the block level before serialization, making this string-based pass redundant.

	// Inject Snowflake cloud mapping for discovered tables
	if app != nil {
		for _, table := range app.DiscoveredTables {
			hcl = injectSnowflakeCloudMapping(hcl, table)
		}
	}

	// Inject Snowflake cloud mapping for discovered elements
	if app != nil {
		for _, element := range app.DiscoveredElements {
			hcl = injectElementCloudMapping(hcl, element)
		}
	}

	return hcl
}

// BuildUUIDMap creates a mapping from UUIDs to Terraform resource references.
// Exported so callers can build once and reuse across multiple PostProcessHCLWithUUIDMap calls.
func BuildUUIDMap(imports []ImportBlock, app *discovery.App) map[string]string {
	return buildUUIDMap(imports, app)
}

// buildUUIDMap creates a mapping from UUIDs to Terraform resource references
func buildUUIDMap(imports []ImportBlock, app *discovery.App) map[string]string {
	uuidMap := make(map[string]string)

	for _, imp := range imports {
		// Parse the import ID to extract individual UUIDs
		parts := strings.Split(imp.ID, ":")

		ref := imp.ResourceType + "." + imp.ResourceName

		switch len(parts) {
		case 1:
			uuidMap[parts[0]] = ref + ".id"
		case 2:
			uuidMap[parts[1]] = ref + ".id"
			if imp.ResourceType == "elementum_layout" {
				uuidMap[parts[1]] = ref + ".stage_id"
			}
		case 3:
			uuidMap[parts[2]] = ref + ".id"
		}
	}

	if app == nil {
		return uuidMap
	}

	// Find the app resource name and add app mappings
	appResourceName := findResourceName(imports, "elementum_app", app.ID)
	if appResourceName != "" {
		mergeMaps(uuidMap, app.GetUUIDMappings(appResourceName))
	}

	// Add category mapping (maps category_id UUID to data source reference)
	if app.CategoryID != "" && app.CategoryName != "" {
		sanitizedCatName := SanitizeName(app.CategoryName)
		uuidMap[app.CategoryID] = "data.elementum_category." + sanitizedCatName + ".id"
	}

	// Add cloudlink mapping (maps cloud_link_id UUID to data source reference)
	if app.CloudLinkID != "" && app.CloudLinkName != "" {
		sanitizedCloudLinkName := SanitizeName(app.CloudLinkName)
		uuidMap[app.CloudLinkID] = "data.elementum_cloudlink." + sanitizedCloudLinkName + ".id"
	}

	// Add AI provider connector mappings (maps ai_provider_connector_id UUID to data source reference)
	// Uses same naming logic as GenerateAiProviderConnectorDataSources for consistency
	aiConnectorNameCounter := make(map[string]int)
	for _, conn := range app.DiscoveredAiProviderConnectors {
		if conn.ID != "" && conn.ModelName != "" {
			baseName := SanitizeName(conn.ModelName)
			count := aiConnectorNameCounter[baseName]
			aiConnectorNameCounter[baseName]++

			resourceName := baseName
			if count > 0 {
				resourceName = fmt.Sprintf("%s_%d", baseName, count)
			}
			uuidMap[conn.ID] = fmt.Sprintf("data.elementum_ai_provider_connector.%s.id", resourceName)
		}
	}

	// Add stored function mappings (maps stored_function_id UUID to data source reference)
	// Uses same naming logic as GenerateStoredFunctionDataSources for consistency
	storedFunctionNameCounter := make(map[string]int)
	for _, sf := range app.DiscoveredStoredFunctions {
		if sf.ID != "" && sf.DisplayName != "" {
			baseName := SanitizeName(sf.DisplayName)
			count := storedFunctionNameCounter[baseName]
			storedFunctionNameCounter[baseName]++

			resourceName := baseName
			if count > 0 {
				resourceName = fmt.Sprintf("%s_%d", baseName, count)
			}
			uuidMap[sf.ID] = fmt.Sprintf("data.elementum_stored_function.%s.id", resourceName)
		}
	}

	// Add field mappings
	// Build prefix for local references (app resource name + underscore)
	localPrefix := ""
	if appResourceName != "" {
		localPrefix = appResourceName + "_"
	}

	for _, field := range app.Fields {
		resourceName := findResourceName(imports, "elementum_"+field.Type+"_field", field.ID)
		if resourceName != "" {
			mergeMaps(uuidMap, field.GetUUIDMappings(resourceName))
		}

		// Map system fields to local references (prefixed with app resource name)
		if len(field.SemanticTags) > 0 {
			for _, tag := range field.SemanticTags {
				switch strings.ToUpper(tag) {
				case "TITLE":
					uuidMap[field.ID] = "local." + localPrefix + "title_field_id"
				case "STATUS":
					uuidMap[field.ID] = "local." + localPrefix + "status_field_id"
					// Also map status option IDs - use direct resource reference (not locals)
					// since elementum_app exposes status_option_ids_by_label as computed attribute
					for _, opt := range field.Options {
						if opt.ID != "" && opt.Label != "" {
							uuidMap[opt.ID] = "elementum_app." + appResourceName + ".status_option_ids_by_label[\"" + opt.Label + "\"]"
						}
					}
				case "STAGE":
					uuidMap[field.ID] = "local." + localPrefix + "stage_field_id"
					// Also map stage option IDs (stage labels)
					for _, opt := range field.Options {
						if opt.ID != "" && opt.Label != "" {
							uuidMap[opt.ID] = "local." + localPrefix + "stage_options_by_label[\"" + opt.Label + "\"]"
						}
					}
				case "HANDLE", "ID":
					uuidMap[field.ID] = "local." + localPrefix + "id_field_id"
				case "CREATED_BY":
					uuidMap[field.ID] = "local." + localPrefix + "created_by_field_id"
				case "CREATED_AT":
					uuidMap[field.ID] = "local." + localPrefix + "created_at_field_id"
				case "UPDATED_BY":
					uuidMap[field.ID] = "local." + localPrefix + "updated_by_field_id"
				case "UPDATED_AT":
					uuidMap[field.ID] = "local." + localPrefix + "updated_at_field_id"
				case "CLOSED_BY":
					uuidMap[field.ID] = "local." + localPrefix + "closed_by_field_id"
				case "CLOSED_AT":
					uuidMap[field.ID] = "local." + localPrefix + "closed_at_field_id"
				}
			}
		}

		// Map any remaining dropdown/multi_select option IDs (non-system fields)
		if (field.Type == "dropdown" || field.Type == "multi_select") && resourceName != "" {
			for _, opt := range field.Options {
				if opt.ID != "" && opt.Label != "" {
					// Check if not already mapped (system fields above take precedence)
					if _, exists := uuidMap[opt.ID]; !exists {
						uuidMap[opt.ID] = "local." + resourceName + "_options_by_label[\"" + opt.Label + "\"]"
					}
				}
			}
		}
	}

	// Add layout mappings (stages)
	// Skip Initiate layouts - they are not exported
	for _, layout := range app.Layouts {
		if layout.IsInitiate {
			continue
		}
		resourceName := findResourceName(imports, "elementum_layout", layout.ID)
		if resourceName != "" {
			mergeMaps(uuidMap, layout.GetUUIDMappings(resourceName))
		}
		stageKey := SanitizeName(layout.Name)
		uuidMap[layout.ID] = "local." + localPrefix + "stage_ids_by_key[\"" + stageKey + "\"]"
	}

	// Add flow mappings
	for _, flow := range app.Flows {
		resourceName := findResourceName(imports, "elementum_flow", flow.ID)
		if resourceName != "" {
			mergeMaps(uuidMap, flow.GetUUIDMappings(resourceName))
		}
	}

	// Add automation, trigger, and task mappings
	for _, automation := range app.Automations {
		resourceName := findResourceName(imports, "elementum_automation", automation.ID)
		if resourceName != "" {
			mergeMaps(uuidMap, automation.GetUUIDMappings(resourceName))

			for _, trigger := range automation.Triggers {
				triggerResourceName := findResourceName(imports, "elementum_"+trigger.Type+"_trigger", trigger.ID)
				if triggerResourceName != "" {
					mergeMaps(uuidMap, trigger.GetUUIDMappings(triggerResourceName))
				}
			}

			for _, task := range automation.Tasks {
				taskResourceName := findResourceName(imports, "elementum_"+task.Type+"_task", task.ID)
				if taskResourceName != "" {
					mergeMaps(uuidMap, task.GetUUIDMappings(taskResourceName))
				}
			}
		}
	}

	// Add agent and agent tool mappings
	for _, agent := range app.Agents {
		resourceName := findResourceName(imports, "elementum_agent", agent.ID)
		if resourceName != "" {
			mergeMaps(uuidMap, agent.GetUUIDMappings(resourceName))

			for _, tool := range agent.Tools {
				toolResourceName := findResourceName(imports, tool.TerraformResourceType(), tool.ID)
				if toolResourceName != "" {
					mergeMaps(uuidMap, tool.GetUUIDMappings(toolResourceName))
				}
			}
		}
	}

	// Add widget mappings
	for _, widget := range app.Widgets {
		resourceName := findResourceName(imports, "elementum_widget", widget.ID)
		if resourceName != "" {
			mergeMaps(uuidMap, widget.GetUUIDMappings(resourceName))
		}
	}

	// Add file reader mappings (AI, Text/OCR, JSON, XML)
	for _, reader := range app.AIFileReaders {
		// Use the correct resource type based on the file reader type
		resourceType := reader.TerraformResourceType()
		resourceName := findResourceName(imports, resourceType, reader.ID)
		if resourceName != "" {
			mergeMaps(uuidMap, reader.GetUUIDMappings(resourceName))
		}
	}
	// If file-readers were deduped by name, rewrite every dropped ID to the
	// same reference as the canonical ID that replaced it. Without this,
	// file_reader_id attributes on tasks that were pointing at the dropped
	// duplicates would emit raw UUIDs (broken) instead of TF references.
	for droppedID, canonicalID := range app.FileReaderIDAlias {
		if canonicalRef, ok := uuidMap[canonicalID]; ok {
			uuidMap[droppedID] = canonicalRef
		}
	}

	// Agentic skill + skill-tool mappings. Agents reference skills via
	// skill_ids, and tools reference parents via skill_id — both need to
	// resolve to proper TF references rather than raw UUIDs. The empty-ID
	// guard matters: findResourceName does a `strings.Contains(imp.ID, id)`
	// which is always true for "", which would poison `uuidMap[""]` and
	// cause every quoted empty string in the output (`description = ""`,
	// etc.) to be rewritten to that resource's reference during
	// beautification.
	for _, skill := range app.Skills {
		if skill.ID != "" {
			if resourceName := findResourceName(imports, "elementum_agentic_skill", skill.ID); resourceName != "" {
				uuidMap[skill.ID] = "elementum_agentic_skill." + resourceName + ".id"
			}
		}
		for _, tool := range skill.Tools {
			if tool.ID == "" {
				continue
			}
			if toolResourceName := findResourceName(imports, "elementum_agentic_skill_tool", tool.ID); toolResourceName != "" {
				uuidMap[tool.ID] = "elementum_agentic_skill_tool." + toolResourceName + ".id"
			}
		}
	}

	// Agent-to-agent skills on agent cards.
	for _, agent := range app.AllAgents() {
		for _, sk := range agent.A2ASkills {
			if sk.ID == "" {
				continue
			}
			if name := findResourceName(imports, "elementum_agent_a2a_skill", sk.ID); name != "" {
				uuidMap[sk.ID] = "elementum_agent_a2a_skill." + name + ".id"
			}
		}
	}

	// Add approval process mappings
	for _, approval := range app.Approvals {
		resourceName := findResourceName(imports, "elementum_approval_process", approval.ID)
		if resourceName != "" {
			mergeMaps(uuidMap, approval.GetUUIDMappings(resourceName))
		}
	}

	// Add relationship mappings
	for _, relationship := range app.Relationships {
		resourceName := findResourceName(imports, "elementum_relationship", relationship.ID)
		if resourceName != "" {
			uuidMap[relationship.ID] = "elementum_relationship." + resourceName + ".id"
		}
	}

	// Add access policy mappings
	for i, policy := range app.AccessPolicies {
		resourceName := findResourceName(imports, "elementum_access_policy", policy.ID)
		if resourceName == "" {
			resourceName = fmt.Sprintf("%s_policy_%d", SanitizeName(app.Namespace), i+1)
		}
		uuidMap[policy.ID] = "elementum_access_policy." + resourceName + ".id"

		// Add user ID to data source mappings for beautification
		for userID, email := range policy.UserEmails {
			if email != "" {
				userResourceName := SanitizeName(email)
				uuidMap[userID] = "data.elementum_user." + userResourceName + ".id"
			}
		}

		// Add group ID to data source mappings for beautification
		for groupID, name := range policy.GroupNames {
			if name != "" {
				groupResourceName := SanitizeName(name)
				uuidMap[groupID] = "data.elementum_group." + groupResourceName + ".id"
			}
		}
	}

	// Add role mappings
	roleNames := make(map[string]int)
	for _, role := range app.Roles {
		baseName := SanitizeName(role.Name)
		count := roleNames[baseName]
		roleNames[baseName]++

		name := baseName
		if count > 0 {
			name = fmt.Sprintf("%s_%d", baseName, count)
		}

		if role.Managed {
			// Managed roles are data sources
			uuidMap[role.ID] = "data.elementum_role." + name + ".id"
		} else {
			// Custom roles are resources
			uuidMap[role.ID] = "elementum_role." + name + ".id"
		}
	}

	// Add user and group mappings from role membership
	roleUserNames := make(map[string]int)
	roleGroupNames := make(map[string]int)
	for _, role := range app.Roles {
		if role.Managed {
			continue // Only custom roles have membership we export
		}
		for _, user := range role.Users {
			if user.Name == "" {
				continue
			}
			baseName := sanitizeEmailToName(user.Name)
			if _, exists := roleUserNames[baseName]; !exists {
				count := roleUserNames[baseName]
				roleUserNames[baseName]++

				name := baseName
				if count > 0 {
					name = fmt.Sprintf("%s_%d", baseName, count)
				}
				uuidMap[user.ID] = "data.elementum_user." + name + ".id"
			}
		}
		for _, group := range role.Groups {
			if group.Name == "" {
				continue
			}
			baseName := SanitizeName(group.Name)
			if _, exists := roleGroupNames[baseName]; !exists {
				count := roleGroupNames[baseName]
				roleGroupNames[baseName]++

				name := baseName
				if count > 0 {
					name = fmt.Sprintf("%s_%d", baseName, count)
				}
				uuidMap[group.ID] = "data.elementum_group." + name + ".id"
			}
		}
	}

	// Add related object mappings
	// Build sets of discovered IDs to skip (same as GenerateRelationshipDataSources)
	// Discovered items are exported as resources, not data sources
	discoveredAppIDs := make(map[string]bool)
	for _, da := range app.DiscoveredApps {
		discoveredAppIDs[da.ID] = true
	}
	discoveredElementIDs := make(map[string]bool)
	for _, de := range app.DiscoveredElements {
		discoveredElementIDs[de.ID] = true
	}
	discoveredTaskIDs := make(map[string]bool)
	for _, dt := range app.DiscoveredTasks {
		discoveredTaskIDs[dt.ID] = true
	}

	relatedNames := make(map[string]int)
	generatedRelatedIDs := make(map[string]bool) // Track already-mapped IDs to avoid duplicates
	for _, relatedObj := range app.RelatedObjects {
		// Skip discovered items - they get resource refs from DiscoveredApps/Elements/Tasks loops
		switch relatedObj.Type {
		case "App":
			if discoveredAppIDs[relatedObj.ID] {
				continue
			}
		case "Element":
			if discoveredElementIDs[relatedObj.ID] {
				continue
			}
		case "Task":
			if discoveredTaskIDs[relatedObj.ID] {
				continue
			}
		}

		// Skip duplicate IDs to avoid overwriting with wrong counter suffix
		if generatedRelatedIDs[relatedObj.ID] {
			continue
		}
		generatedRelatedIDs[relatedObj.ID] = true

		// Increment counter only for items that will actually be mapped
		baseName := SanitizeName(relatedObj.Name)
		count := relatedNames[baseName]
		relatedNames[baseName]++

		name := baseName
		if count > 0 {
			name = fmt.Sprintf("%s_%d", baseName, count)
		}

		dataSourceType := "data.elementum_app"
		switch relatedObj.Type {
		case "Element":
			dataSourceType = "data.elementum_element"
		case "Task":
			dataSourceType = "data.elementum_task"
		}

		uuidMap[relatedObj.ID] = dataSourceType + "." + name + ".id"

		// Note: System field IDs (id_field_id, title_field_id, etc.) from related objects
		// cannot be automatically beautified because the data sources don't expose these
		// attributes. Users need to use field data sources for related object fields.
		// Example: data "elementum_field" "related_id" { app_id = data.elementum_element.xxx.id name = "ID" }
	}

	// Add datamine mappings (from automation dependencies)
	for _, datamine := range app.ReferencedDatamines {
		resourceName := findResourceName(imports, "elementum_datamine", datamine.ID)
		if resourceName != "" {
			uuidMap[datamine.ID] = "elementum_datamine." + resourceName + ".id"
		}
	}

	// Add table mappings (from automation dependencies)
	for _, table := range app.ReferencedTables {
		resourceName := findResourceName(imports, "elementum_table", table.ID)
		if resourceName != "" {
			uuidMap[table.ID] = "elementum_table." + resourceName + ".id"
		}
	}

	// Add discovered app mappings (from recursive discovery)
	// Discovered apps are always exported as resources
	for _, discoveredApp := range app.DiscoveredApps {
		// Find the resource name from the import blocks
		resourceName := findResourceName(imports, "elementum_app", discoveredApp.ID)
		if resourceName != "" {
			// Map to resource reference
			uuidMap[discoveredApp.ID] = "elementum_app." + resourceName + ".id"
		} else {
			// Fallback: use namespace/name for resource name (should always have import block)
			// Check raw value before sanitizing since SanitizeName("") returns "resource"
			var prefix string
			if discoveredApp.Namespace != "" {
				prefix = SanitizeName(discoveredApp.Namespace)
			} else {
				prefix = SanitizeName(discoveredApp.Name)
			}
			uuidMap[discoveredApp.ID] = "elementum_app." + prefix + ".id"
		}

		// Add discovered app category mapping (maps category_id UUID to data source reference)
		if discoveredApp.CategoryID != "" && discoveredApp.CategoryName != "" {
			sanitizedCatName := SanitizeName(discoveredApp.CategoryName)
			uuidMap[discoveredApp.CategoryID] = "data.elementum_category." + sanitizedCatName + ".id"
		}

		// Add discovered app cloudlink mapping (maps cloud_link_id UUID to data source reference)
		if discoveredApp.CloudLinkID != "" && discoveredApp.CloudLinkName != "" {
			sanitizedCloudLinkName := SanitizeName(discoveredApp.CloudLinkName)
			uuidMap[discoveredApp.CloudLinkID] = "data.elementum_cloudlink." + sanitizedCloudLinkName + ".id"
		}

		// Add discovered app system field mappings
		// Use Namespace for prefix if available, otherwise Name
		appPrefix := SanitizeName(discoveredApp.Namespace)
		if appPrefix == "" {
			appPrefix = SanitizeName(discoveredApp.Name)
		}
		appLocalPrefix := appPrefix + "_"

		// Discovered app's fields
		for _, field := range discoveredApp.Fields {
			fieldResourceName := findResourceName(imports, "elementum_"+field.Type+"_field", field.ID)
			if fieldResourceName != "" {
				mergeMaps(uuidMap, field.GetUUIDMappings(fieldResourceName))
			}

			// Map system fields to local references
			if len(field.SemanticTags) > 0 {
				for _, tag := range field.SemanticTags {
					switch strings.ToUpper(tag) {
					case "TITLE":
						uuidMap[field.ID] = "local." + appLocalPrefix + "title_field_id"
					case "STATUS":
						uuidMap[field.ID] = "local." + appLocalPrefix + "status_field_id"
						// Also map status option IDs - use direct resource reference (not locals)
						// since elementum_app exposes status_option_ids_by_label as computed attribute
						for _, opt := range field.Options {
							if opt.ID != "" && opt.Label != "" {
								uuidMap[opt.ID] = "elementum_app." + appPrefix + ".status_option_ids_by_label[\"" + opt.Label + "\"]"
							}
						}
					case "STAGE":
						uuidMap[field.ID] = "local." + appLocalPrefix + "stage_field_id"
						// Also map stage option IDs
						for _, opt := range field.Options {
							if opt.ID != "" && opt.Label != "" {
								uuidMap[opt.ID] = "local." + appLocalPrefix + "stage_options_by_label[\"" + opt.Label + "\"]"
							}
						}
					case "HANDLE", "ID":
						uuidMap[field.ID] = "local." + appLocalPrefix + "id_field_id"
					case "CREATED_BY":
						uuidMap[field.ID] = "local." + appLocalPrefix + "created_by_field_id"
					case "CREATED_AT":
						uuidMap[field.ID] = "local." + appLocalPrefix + "created_at_field_id"
					case "UPDATED_BY":
						uuidMap[field.ID] = "local." + appLocalPrefix + "updated_by_field_id"
					case "UPDATED_AT":
						uuidMap[field.ID] = "local." + appLocalPrefix + "updated_at_field_id"
					case "CLOSED_BY":
						uuidMap[field.ID] = "local." + appLocalPrefix + "closed_by_field_id"
					case "CLOSED_AT":
						uuidMap[field.ID] = "local." + appLocalPrefix + "closed_at_field_id"
					}
				}
			}

			// Map any remaining dropdown/multi_select option IDs (non-system fields)
			if (field.Type == "dropdown" || field.Type == "multi_select") && fieldResourceName != "" {
				for _, opt := range field.Options {
					if opt.ID != "" && opt.Label != "" {
						// Check if not already mapped (system fields above take precedence)
						if _, exists := uuidMap[opt.ID]; !exists {
							uuidMap[opt.ID] = "local." + fieldResourceName + "_options_by_label[\"" + opt.Label + "\"]"
						}
					}
				}
			}
		}
	}

	// Add discovered element mappings (from recursive discovery)
	for _, element := range app.DiscoveredElements {
		// Use Namespace for prefix if available, otherwise Handle, otherwise Name
		prefix := SanitizeName(element.Namespace)
		if prefix == "" {
			prefix = SanitizeName(element.Handle)
		}
		if prefix == "" {
			prefix = SanitizeName(element.Name)
		}

		// Element resource itself
		uuidMap[element.ID] = "elementum_element." + prefix + ".id"

		// Add element category mapping (maps category_id UUID to data source reference)
		if element.CategoryID != "" && element.CategoryName != "" {
			sanitizedCatName := SanitizeName(element.CategoryName)
			uuidMap[element.CategoryID] = "data.elementum_category." + sanitizedCatName + ".id"
		}

		// Add element cloudlink mapping (maps cloud_link_id UUID to data source reference)
		if element.CloudLinkID != "" && element.CloudLinkName != "" {
			sanitizedCloudLinkName := SanitizeName(element.CloudLinkName)
			uuidMap[element.CloudLinkID] = "data.elementum_cloudlink." + sanitizedCloudLinkName + ".id"
		}

		// Add element system field mappings (similar to app)
		elementLocalPrefix := prefix + "_"

		// Element's fields
		for _, field := range element.Fields {
			resourceName := findResourceName(imports, "elementum_"+field.Type+"_field", field.ID)
			if resourceName != "" {
				mergeMaps(uuidMap, field.GetUUIDMappings(resourceName))
			}

			// Map system fields - TITLE and ID use locals (element resource exposes these)
			// Other system fields use data source lookups (element resource doesn't expose audit fields)
			if len(field.SemanticTags) > 0 {
				for _, tag := range field.SemanticTags {
					switch strings.ToUpper(tag) {
					case "TITLE":
						uuidMap[field.ID] = "local." + elementLocalPrefix + "title_field_id"
					case "HANDLE", "ID":
						uuidMap[field.ID] = "local." + elementLocalPrefix + "id_field_id"
					case "CREATED_BY":
						uuidMap[field.ID] = "data.elementum_field." + prefix + "_created_by.id"
					case "CREATED_AT":
						uuidMap[field.ID] = "data.elementum_field." + prefix + "_created_at.id"
					case "UPDATED_BY":
						uuidMap[field.ID] = "data.elementum_field." + prefix + "_updated_by.id"
					case "UPDATED_AT":
						uuidMap[field.ID] = "data.elementum_field." + prefix + "_updated_at.id"
					case "CLOSED_BY":
						uuidMap[field.ID] = "data.elementum_field." + prefix + "_closed_by.id"
					case "CLOSED_AT":
						uuidMap[field.ID] = "data.elementum_field." + prefix + "_closed_at.id"
					}
				}
			}

			// Map any remaining dropdown/multi_select option IDs (non-system fields)
			if (field.Type == "dropdown" || field.Type == "multi_select") && resourceName != "" {
				for _, opt := range field.Options {
					if opt.ID != "" && opt.Label != "" {
						// Check if not already mapped (system fields above take precedence)
						if _, exists := uuidMap[opt.ID]; !exists {
							uuidMap[opt.ID] = "local." + resourceName + "_options_by_label[\"" + opt.Label + "\"]"
						}
					}
				}
			}
		}

		// Element's layouts
		for _, layout := range element.Layouts {
			resourceName := findResourceName(imports, "elementum_layout", layout.ID)
			if resourceName != "" {
				mergeMaps(uuidMap, layout.GetUUIDMappings(resourceName))
			}
		}

		// Element's widgets
		for _, widget := range element.Widgets {
			resourceName := findResourceName(imports, "elementum_widget", widget.ID)
			if resourceName != "" {
				mergeMaps(uuidMap, widget.GetUUIDMappings(resourceName))
			}
		}

		// Element's automations
		for _, automation := range element.Automations {
			resourceName := findResourceName(imports, "elementum_automation", automation.ID)
			if resourceName != "" {
				mergeMaps(uuidMap, automation.GetUUIDMappings(resourceName))

				for _, trigger := range automation.Triggers {
					triggerResourceName := findResourceName(imports, "elementum_"+trigger.Type+"_trigger", trigger.ID)
					if triggerResourceName != "" {
						mergeMaps(uuidMap, trigger.GetUUIDMappings(triggerResourceName))
					}
				}

				for _, task := range automation.Tasks {
					taskResourceName := findResourceName(imports, "elementum_"+task.Type+"_task", task.ID)
					if taskResourceName != "" {
						mergeMaps(uuidMap, task.GetUUIDMappings(taskResourceName))
					}
				}
			}
		}

		// Element's relationships
		for _, relationship := range element.Relationships {
			resourceName := findResourceName(imports, "elementum_relationship", relationship.ID)
			if resourceName != "" {
				uuidMap[relationship.ID] = "elementum_relationship." + resourceName + ".id"
			}
		}

		// Element's dashboards and widgets
		for _, dashboard := range element.Dashboards {
			resourceName := findResourceName(imports, "elementum_dashboard", dashboard.ID)
			if resourceName != "" {
				mergeMaps(uuidMap, dashboard.GetUUIDMappings(resourceName))
			}

			for _, widget := range dashboard.Widgets {
				widgetResourceName := findResourceName(imports, "elementum_dashboard_widget", widget.ID)
				if widgetResourceName != "" {
					mergeMaps(uuidMap, widget.GetUUIDMappings(widgetResourceName))
				}
			}
		}

		// Element's charts
		for _, chart := range element.Charts {
			resourceName := findResourceName(imports, "elementum_chart", chart.ID)
			if resourceName != "" {
				mergeMaps(uuidMap, chart.GetUUIDMappings(resourceName))
			}
		}
	}

	// Add discovered task mappings (from recursive discovery)
	// Discovered tasks are exported as full resources (like discovered apps)
	for _, task := range app.DiscoveredTasks {
		// Use Namespace for prefix if available, otherwise Name
		prefix := SanitizeName(task.Namespace)
		if prefix == "" {
			prefix = SanitizeName(task.Name)
		}

		// Task is exported as resource reference
		uuidMap[task.ID] = "elementum_task." + prefix + ".id"

		// Add task category mapping (maps category_id UUID to data source reference)
		if task.CategoryID != "" && task.CategoryName != "" {
			sanitizedCatName := SanitizeName(task.CategoryName)
			uuidMap[task.CategoryID] = "data.elementum_category." + sanitizedCatName + ".id"
		}

		// Add task cloudlink mapping if present
		if task.CloudLinkID != "" && task.CloudLinkName != "" {
			sanitizedCloudLinkName := SanitizeName(task.CloudLinkName)
			uuidMap[task.CloudLinkID] = "data.elementum_cloudlink." + sanitizedCloudLinkName + ".id"
		}

		// Add task system field mappings
		taskLocalPrefix := prefix + "_"

		// Task's fields
		for _, field := range task.Fields {
			resourceName := findResourceName(imports, "elementum_"+field.Type+"_field", field.ID)
			if resourceName != "" {
				mergeMaps(uuidMap, field.GetUUIDMappings(resourceName))
			}

			// Map system fields to local references
			if len(field.SemanticTags) > 0 {
				for _, tag := range field.SemanticTags {
					switch strings.ToUpper(tag) {
					case "TITLE":
						uuidMap[field.ID] = "local." + taskLocalPrefix + "title_field_id"
					case "STATUS":
						uuidMap[field.ID] = "local." + taskLocalPrefix + "status_field_id"
						// Also map status option IDs
						for _, opt := range field.Options {
							if opt.ID != "" && opt.Label != "" {
								uuidMap[opt.ID] = "local." + taskLocalPrefix + "status_options_by_label[\"" + opt.Label + "\"]"
							}
						}
					case "HANDLE", "ID":
						uuidMap[field.ID] = "local." + taskLocalPrefix + "id_field_id"
					case "CREATED_BY":
						uuidMap[field.ID] = "local." + taskLocalPrefix + "created_by_field_id"
					case "CREATED_AT":
						uuidMap[field.ID] = "local." + taskLocalPrefix + "created_at_field_id"
					case "UPDATED_BY":
						uuidMap[field.ID] = "local." + taskLocalPrefix + "updated_by_field_id"
					case "UPDATED_AT":
						uuidMap[field.ID] = "local." + taskLocalPrefix + "updated_at_field_id"
					}
				}
			}
		}
	}

	// Add discovered table mappings (from recursive discovery)
	for _, table := range app.DiscoveredTables {
		resourceName := findResourceName(imports, "elementum_table", table.ID)
		if resourceName != "" {
			uuidMap[table.ID] = "elementum_table." + resourceName + ".id"

			// Map table field IDs to local lookup references (for datamine primary_column_ids)
			// Uses pattern: local.<table_name>_field_ids_by_name["<field_name>"]
			for _, field := range table.Fields {
				if field.ID != "" && field.Name != "" {
					uuidMap[field.ID] = "local." + resourceName + "_field_ids_by_name[\"" + field.Name + "\"]"
				}
			}
		}
	}

	// Add discovered datamine mappings (from recursive discovery)
	for _, datamine := range app.DiscoveredDatamines {
		resourceName := findResourceName(imports, "elementum_datamine", datamine.ID)
		if resourceName != "" {
			uuidMap[datamine.ID] = "elementum_datamine." + resourceName + ".id"
		}
	}

	// Add phone service mappings
	for _, service := range app.PhoneServices {
		resourceName := findResourceName(imports, "elementum_phone_service", service.ID)
		if resourceName != "" {
			mergeMaps(uuidMap, service.GetUUIDMappings(resourceName))
		}
	}

	// Add AI search table mappings from App
	for _, searchTable := range app.AISearchTables {
		resourceName := findResourceName(imports, "elementum_ai_search_table", searchTable.ID)
		if resourceName != "" {
			mergeMaps(uuidMap, searchTable.GetUUIDMappings(resourceName))
		}
	}

	// Add AI search table mappings from discovered Elements
	for _, element := range app.DiscoveredElements {
		for _, searchTable := range element.AISearchTables {
			resourceName := findResourceName(imports, "elementum_ai_search_table", searchTable.ID)
			if resourceName != "" {
				mergeMaps(uuidMap, searchTable.GetUUIDMappings(resourceName))
			}
		}
	}

	// Add table search table mappings from referenced tables
	for _, table := range app.ReferencedTables {
		for _, searchTable := range table.SearchTables {
			resourceName := findResourceName(imports, "elementum_table_search_table", searchTable.ID)
			if resourceName != "" {
				mergeMaps(uuidMap, searchTable.GetUUIDMappings(resourceName))
			}
		}
	}

	// Add table search table mappings from discovered tables
	for _, table := range app.DiscoveredTables {
		for _, searchTable := range table.SearchTables {
			resourceName := findResourceName(imports, "elementum_table_search_table", searchTable.ID)
			if resourceName != "" {
				mergeMaps(uuidMap, searchTable.GetUUIDMappings(resourceName))
			}
		}
	}

	// Add managed view order mapping
	if app.ManagedViewOrder != nil {
		resourceName := findResourceName(imports, "elementum_managed_view_order", app.ManagedViewOrder.ObjectID)
		if resourceName != "" {
			mergeMaps(uuidMap, app.ManagedViewOrder.GetUUIDMappings(resourceName))
		}
	}

	// Add dashboard and widget mappings
	for _, dashboard := range app.Dashboards {
		resourceName := findResourceName(imports, "elementum_dashboard", dashboard.ID)
		if resourceName != "" {
			mergeMaps(uuidMap, dashboard.GetUUIDMappings(resourceName))
		}

		for _, widget := range dashboard.Widgets {
			widgetResourceName := findResourceName(imports, "elementum_dashboard_widget", widget.ID)
			if widgetResourceName != "" {
				mergeMaps(uuidMap, widget.GetUUIDMappings(widgetResourceName))
			}
		}
	}

	// Add chart mappings
	for _, chart := range app.Charts {
		resourceName := findResourceName(imports, "elementum_chart", chart.ID)
		if resourceName != "" {
			mergeMaps(uuidMap, chart.GetUUIDMappings(resourceName))
		}
	}

	return uuidMap
}

// findResourceName finds the resource name from import blocks for a given resource type and ID
func findResourceName(imports []ImportBlock, resourceType, id string) string {
	// Defensive guard: `strings.Contains(x, "")` is always true, so an empty
	// id would match the first import of the requested type and poison the
	// caller's uuidMap (every empty string in the output would then be
	// rewritten to that resource's ref during beautification).
	if id == "" {
		return ""
	}
	for _, imp := range imports {
		if imp.ResourceType == resourceType && strings.Contains(imp.ID, id) {
			return imp.ResourceName
		}
	}
	return ""
}

// mergeMaps merges src map into dst map
func mergeMaps(dst, src map[string]string) {
	for k, v := range src {
		dst[k] = v
	}
}

// stripNullAttributes removes lines with `attribute = null`
func stripNullAttributes(hcl string) string {
	lines := strings.Split(hcl, "\n")
	var result []string

	for _, line := range lines {
		if nullPattern.MatchString(line) {
			continue
		}
		result = append(result, line)
	}

	return strings.Join(result, "\n")
}

// replaceUUIDs replaces UUID strings with resource references
func replaceUUIDs(hcl string, uuidMap map[string]string) string {
	return uuidPattern.ReplaceAllStringFunc(hcl, func(match string) string {
		uuid := match[1 : len(match)-1]

		if ref, ok := uuidMap[uuid]; ok {
			return ref
		}

		return match
	})
}

// beautifyFlowActions resolves UUIDs in flow action blocks
func beautifyFlowActions(hcl string, uuidMap map[string]string) string {
	hcl = automationIDPattern.ReplaceAllStringFunc(hcl, func(match string) string {
		submatch := automationIDPattern.FindStringSubmatch(match)
		if len(submatch) < 2 {
			return match
		}
		uuid := submatch[1]
		if ref, ok := uuidMap[uuid]; ok {
			return "automation_id = " + ref
		}
		return match
	})

	hcl = objectIDInObjectPattern.ReplaceAllStringFunc(hcl, func(match string) string {
		submatch := objectIDInObjectPattern.FindStringSubmatch(match)
		if len(submatch) < 3 {
			return match
		}
		prefix := submatch[1]
		uuid := submatch[2]
		if ref, ok := uuidMap[uuid]; ok {
			return prefix + ref
		}
		return match
	})

	hcl = agentIDPattern.ReplaceAllStringFunc(hcl, func(match string) string {
		submatch := agentIDPattern.FindStringSubmatch(match)
		if len(submatch) < 2 {
			return match
		}
		uuid := submatch[1]
		if ref, ok := uuidMap[uuid]; ok {
			return "agent_id = " + ref
		}
		return match
	})

	return hcl
}

// stripEmptyFlowActions removes empty action blocks from flow nodes
func stripEmptyFlowActions(hcl string) string {
	return emptyActionPattern.ReplaceAllString(hcl, "")
}

// InjectFlowConfig extracts flow configuration from terraform plan output
// and injects it into the generated config (replacing `stages = null`)
func InjectFlowConfig(generatedContent, planOutput string, imports []ImportBlock, app *discovery.App) string {

	uuidMap := buildUUIDMap(imports, app)

	var appResourceName string
	for _, imp := range imports {
		if imp.ResourceType == "elementum_app" {
			appResourceName = imp.ResourceName
			break
		}
	}

	return flowPattern.ReplaceAllStringFunc(generatedContent, func(match string) string {
		submatch := flowPattern.FindStringSubmatch(match)
		if len(submatch) < 2 {
			return match
		}

		resourceName := submatch[1]

		flowConfig := ExtractResourceFromPlan(planOutput, "elementum_flow", resourceName)
		if flowConfig == "" {
			return match
		}

		flowConfig = beautifyFlowConfig(flowConfig, uuidMap, appResourceName)

		return flowConfig
	})
}

// beautifyFlowConfig applies beautification to flow configuration
func beautifyFlowConfig(flowConfig string, uuidMap map[string]string, appResourceName string) string {
	if appResourceName != "" {
		flowConfig = topLevelObjectIDPattern.ReplaceAllStringFunc(flowConfig, func(match string) string {
			submatch := topLevelObjectIDPattern.FindStringSubmatch(match)
			if len(submatch) < 3 {
				return match
			}
			leadingSpaces := submatch[1]
			return leadingSpaces + "object_id = elementum_app." + appResourceName + ".id"
		})
	}

	flowConfig = automationIDPatternAny.ReplaceAllStringFunc(flowConfig, func(match string) string {
		submatch := automationIDPatternAny.FindStringSubmatch(match)
		if len(submatch) < 2 {
			return match
		}
		uuid := submatch[1]
		if ref, ok := uuidMap[uuid]; ok {
			return "automation_id = " + ref
		}
		return match
	})

	flowConfig = nestedObjectIDPattern.ReplaceAllStringFunc(flowConfig, func(match string) string {
		submatch := nestedObjectIDPattern.FindStringSubmatch(match)
		if len(submatch) < 3 {
			return match
		}
		leadingWhitespace := submatch[1]
		uuid := submatch[2]
		if ref, ok := uuidMap[uuid]; ok {
			return leadingWhitespace + "object_id = " + ref
		}
		return match
	})

	flowConfig = agentIDPatternAny.ReplaceAllStringFunc(flowConfig, func(match string) string {
		submatch := agentIDPatternAny.FindStringSubmatch(match)
		if len(submatch) < 2 {
			return match
		}
		uuid := submatch[1]
		if ref, ok := uuidMap[uuid]; ok {
			return "agent_id = " + ref
		}
		return match
	})

	flowConfig = emptyActionPattern.ReplaceAllString(flowConfig, "")

	flowConfig = multiNewlinePattern.ReplaceAllString(flowConfig, "\n\n")

	return flowConfig
}

// BeautifyTableConfig post-processes generated Terraform HCL for table exports
func BeautifyTableConfig(hcl string, imports []ImportBlock, table *discovery.Table, relatedResources []discovery.RelatedResource) string {
	uuidMap := buildTableUUIDMap(imports, table, relatedResources)

	// Inject Snowflake cloud mapping attributes if the table has Snowflake connection data
	hcl = injectSnowflakeCloudMapping(hcl, table)

	// Build a set of known table IDs (from imports and related resources) for base table detection
	discoveredTableIDs := make(map[string]bool)
	for _, imp := range imports {
		if imp.ResourceType == "elementum_table" {
			discoveredTableIDs[imp.ID] = true
		}
	}
	for _, rel := range relatedResources {
		if rel.ResourceType == "elementum_table" {
			discoveredTableIDs[rel.ID] = true
		}
	}

	// Find the resource name for this table
	resourceName := findResourceName(imports, "elementum_table", table.ID)

	// Strip reference_field_id from base tables (prevents self-referential cycles)
	hcl = stripBaseTableReferenceFieldIDs(hcl, table, discoveredTableIDs, resourceName)

	hcl = stripNullAttributes(hcl)
	hcl = replaceUUIDs(hcl, uuidMap)

	// Strip self-referencing source_id AFTER UUID replacement (when source_id equals table's own ID, it's not meaningful)
	// This must run after replaceUUIDs so the pattern can match terraform references like "elementum_table.foo.id"
	hcl = stripSelfReferencingSourceID(hcl, table)

	return hcl
}

// stripSelfReferencingSourceID removes source_id lines where the value is a self-reference
// (e.g., source_id = elementum_table.foo.id within resource "elementum_table" "foo")
// When source_id equals the table's own ID, it means "this is a root table" not "derived from another table"
func stripSelfReferencingSourceID(hcl string, table *discovery.Table) string {
	if table == nil {
		return hcl
	}

	// Find the table resource name from the HCL
	match := tableResourcePattern.FindStringSubmatch(hcl)
	if match == nil {
		return hcl
	}
	resourceName := match[1]

	// Pattern to match self-referencing source_id line
	// Matches: source_id = elementum_table.resourceName.id or source_id = "uuid" (with flexible whitespace)
	// Using \s* for whitespace to handle spaces, tabs, and potential \r characters
	// Note: This pattern is dynamic based on the resource name, so it can't be pre-compiled
	selfRefPattern := regexp.MustCompile(fmt.Sprintf(`(?m)^\s*source_id\s*=\s*(elementum_table\.%s\.id|"%s")\s*$\r?\n?`, regexp.QuoteMeta(resourceName), regexp.QuoteMeta(table.ID)))

	return selfRefPattern.ReplaceAllString(hcl, "")
}

// stripBaseTableReferenceFieldIDs removes reference_field_id attributes from base table fields.
// Base tables define their own fields (directly from cloud columns) and should NOT have
// reference_field_id attributes, which would create a self-referential cycle.
//
// Derived tables have SourceID pointing to a DIFFERENT source table, and reference_field_id
// is valid because it points to the source table's fields (not its own).
//
// A table is considered a "base table" if:
// - SourceID is empty, OR
// - SourceID equals the table's own ID (self-reference), OR
// - SourceID doesn't point to another discovered table
//
// The resourceName parameter is used to target only the specific table's resource block,
// preventing unintended stripping from other tables in the same HCL.
func stripBaseTableReferenceFieldIDs(hcl string, table *discovery.Table, discoveredTableIDs map[string]bool, resourceName string) string {
	if table == nil || resourceName == "" {
		return hcl
	}

	// Check if this table derives from a DIFFERENT discovered table
	if table.SourceID != "" && table.SourceID != table.ID && discoveredTableIDs[table.SourceID] {
		// Derived table - source is a different table, keep reference_field_id
		return hcl
	}

	// Base table (no source, self-reference, or source is app/element not a table)
	// Strip reference_field_id lines only within this specific table's resource block
	return stripReferenceFieldIDFromTableBlock(hcl, resourceName)
}

// stripReferenceFieldIDFromTableBlock strips reference_field_id attributes from a specific
// table resource block identified by resourceName. This ensures we only affect the target
// table and not other tables in the same HCL.
func stripReferenceFieldIDFromTableBlock(hcl string, resourceName string) string {
	// Pattern to find the start of the specific table resource block
	// e.g., resource "elementum_table" "my_table" {
	blockStartPattern := regexp.MustCompile(fmt.Sprintf(`(?m)^resource "elementum_table" "%s" \{`, regexp.QuoteMeta(resourceName)))

	startMatch := blockStartPattern.FindStringIndex(hcl)
	if startMatch == nil {
		return hcl // Resource block not found, return unchanged
	}

	blockStart := startMatch[0]

	// Find the end of this resource block by counting braces
	// We start with braceCount=1 because the regex matched the opening {
	braceCount := 1
	blockEnd := -1

	for i := startMatch[1]; i < len(hcl); i++ {
		switch hcl[i] {
		case '{':
			braceCount++
		case '}':
			braceCount--
			if braceCount == 0 {
				blockEnd = i + 1
			}
		}
		if blockEnd != -1 {
			break
		}
	}

	if blockEnd == -1 {
		blockEnd = len(hcl) // Block extends to end of file
	}

	// Extract the block, strip reference_field_id, and reconstruct
	before := hcl[:blockStart]
	block := hcl[blockStart:blockEnd]
	after := hcl[blockEnd:]

	// Strip reference_field_id lines from this block only
	strippedBlock := referenceFieldIDPattern.ReplaceAllString(block, "")

	return before + strippedBlock + after
}

// injectSnowflakeCloudMapping injects cloud_mapping_snowflake_* attributes into the table resource
// if the table has Snowflake connection data from the cloudConnection field
func injectSnowflakeCloudMapping(hcl string, table *discovery.Table) string {
	if table == nil {
		return hcl
	}

	// Check if we have Snowflake connection data
	if table.SnowflakeDatabaseName == "" && table.SnowflakeSchemaName == "" && table.SnowflakeTableName == "" {
		return hcl
	}

	// Build the attributes to inject
	var attrs []string
	if table.SnowflakeDatabaseName != "" {
		attrs = append(attrs, fmt.Sprintf(`  cloud_mapping_snowflake_database = %q`, table.SnowflakeDatabaseName))
	}
	if table.SnowflakeSchemaName != "" {
		attrs = append(attrs, fmt.Sprintf(`  cloud_mapping_snowflake_schema   = %q`, table.SnowflakeSchemaName))
	}
	if table.SnowflakeTableName != "" {
		attrs = append(attrs, fmt.Sprintf(`  cloud_mapping_snowflake_table    = %q`, table.SnowflakeTableName))
	}

	if len(attrs) == 0 {
		return hcl
	}

	// Find the table resource block and inject attributes after cloud_mapping_cloudlink_id
	// Check if we have a match
	if tableCloudMappingPattern.MatchString(hcl) {
		// Inject after cloud_mapping_cloudlink_id
		injection := "\n" + strings.Join(attrs, "\n")
		hcl = tableCloudMappingPattern.ReplaceAllString(hcl, "${1}"+injection+"${2}")
	} else {
		// If cloud_mapping_cloudlink_id doesn't exist, inject after the opening brace
		// This is a fallback for tables that don't have a CloudLink
		if tableOpenPattern.MatchString(hcl) {
			injection := "\n" + strings.Join(attrs, "\n")
			hcl = tableOpenPattern.ReplaceAllString(hcl, "${1}"+injection+"${2}")
		}
	}

	return hcl
}

// injectElementCloudMapping injects cloud_mapping_snowflake_* attributes into the element resource
// if the element has Snowflake connection data from the cloudConnection field
func injectElementCloudMapping(hcl string, element *discovery.Element) string {
	if element == nil {
		return hcl
	}

	// Check if we have Snowflake connection data
	if element.SnowflakeDatabaseName == "" && element.SnowflakeSchemaName == "" && element.SnowflakeTableName == "" {
		return hcl
	}

	// Build the attributes to inject
	var attrs []string
	if element.SnowflakeDatabaseName != "" {
		attrs = append(attrs, fmt.Sprintf(`  cloud_mapping_snowflake_database = %q`, element.SnowflakeDatabaseName))
	}
	if element.SnowflakeSchemaName != "" {
		attrs = append(attrs, fmt.Sprintf(`  cloud_mapping_snowflake_schema   = %q`, element.SnowflakeSchemaName))
	}
	if element.SnowflakeTableName != "" {
		attrs = append(attrs, fmt.Sprintf(`  cloud_mapping_snowflake_table    = %q`, element.SnowflakeTableName))
	}

	if len(attrs) == 0 {
		return hcl
	}

	// Find the element resource block and inject attributes after cloud_link_id
	// Pattern matches: resource "elementum_element" "name" {
	elementResourcePattern := regexp.MustCompile(`(?s)(resource "elementum_element" "` + regexp.QuoteMeta(SanitizeName(element.Name)) + `" \{[^}]*?cloud_link_id\s*=\s*[^\n]+)(\n)`)

	// Check if we have a match
	if elementResourcePattern.MatchString(hcl) {
		// Inject after cloud_link_id
		injection := "\n" + strings.Join(attrs, "\n")
		hcl = elementResourcePattern.ReplaceAllString(hcl, "${1}"+injection+"${2}")
	} else {
		// If cloud_link_id doesn't exist, inject after the opening brace
		// This is a fallback for elements that don't have a CloudLink
		elementOpenPattern := regexp.MustCompile(`(resource "elementum_element" "` + regexp.QuoteMeta(SanitizeName(element.Name)) + `" \{)(\n)`)
		if elementOpenPattern.MatchString(hcl) {
			injection := "\n" + strings.Join(attrs, "\n")
			hcl = elementOpenPattern.ReplaceAllString(hcl, "${1}"+injection+"${2}")
		}
	}

	return hcl
}

// buildTableUUIDMap creates a mapping from UUIDs to Terraform resource references for tables
func buildTableUUIDMap(imports []ImportBlock, table *discovery.Table, relatedResources []discovery.RelatedResource) map[string]string {
	uuidMap := make(map[string]string)

	// Determine the current table's ID to avoid self-references
	currentTableID := ""
	if table != nil {
		currentTableID = table.ID
	}

	for _, imp := range imports {
		// Skip the current table's own ID to avoid self-references like source_id = self.id
		if imp.ID == currentTableID {
			continue
		}
		ref := imp.ResourceType + "." + imp.ResourceName
		uuidMap[imp.ID] = ref + ".id"
	}

	if table != nil {
		resourceName := ""
		for _, imp := range imports {
			if imp.ResourceType == "elementum_table" && imp.ID == table.ID {
				resourceName = imp.ResourceName
				break
			}
		}
		if resourceName != "" {
			mergeMaps(uuidMap, table.GetUUIDMappings(resourceName))
		}
	}

	for _, res := range relatedResources {
		sanitizedName := SanitizeName(res.Name)
		var ref string
		if res.ResourceType == "elementum_cloudlink" {
			// CloudLinks are exported as data sources, not resources
			ref = "data.elementum_cloudlink." + sanitizedName + ".id"
		} else {
			ref = res.ResourceType + "." + sanitizedName + ".id"
		}
		uuidMap[res.ID] = ref
	}

	// Add table search table mappings
	if table != nil {
		for _, searchTable := range table.SearchTables {
			resourceName := findResourceName(imports, "elementum_table_search_table", searchTable.ID)
			if resourceName != "" {
				mergeMaps(uuidMap, searchTable.GetUUIDMappings(resourceName))
			}
		}
	}

	return uuidMap
}

// BeautifyDatamineConfig post-processes generated Terraform HCL for datamine exports
func BeautifyDatamineConfig(hcl string, imports []ImportBlock, datamine *discovery.Datamine, table *discovery.Table, relatedResources []discovery.RelatedResource) string {
	uuidMap := buildDatamineUUIDMap(imports, datamine, table, relatedResources)

	hcl = stripNullAttributes(hcl)
	hcl = replaceUUIDs(hcl, uuidMap)
	hcl = beautifyPrimaryColumnIDArrays(hcl, uuidMap)

	return hcl
}

// buildDatamineUUIDMap creates a mapping from UUIDs to Terraform resource references for datamines
func buildDatamineUUIDMap(imports []ImportBlock, datamine *discovery.Datamine, table *discovery.Table, relatedResources []discovery.RelatedResource) map[string]string {
	uuidMap := make(map[string]string)

	for _, imp := range imports {
		parts := strings.Split(imp.ID, ":")
		ref := imp.ResourceType + "." + imp.ResourceName

		switch len(parts) {
		case 1:
			uuidMap[parts[0]] = ref + ".id"
		case 2:
			uuidMap[parts[1]] = ref + ".id"
		}
	}

	if datamine != nil {
		resourceName := ""
		for _, imp := range imports {
			if imp.ResourceType == "elementum_datamine" && strings.Contains(imp.ID, datamine.ID) {
				resourceName = imp.ResourceName
				break
			}
		}
		if resourceName != "" {
			mergeMaps(uuidMap, datamine.GetUUIDMappings(resourceName))
		}
	}

	if table != nil {
		tableName := SanitizeName(table.Name)
		uuidMap[table.ID] = "elementum_table." + tableName + ".id"

		// Map table field IDs to field references (for datamine primary_column_ids)
		for _, field := range table.Fields {
			if field.ID != "" && field.Name != "" {
				// Map to data source field lookup by name
				sanitizedFieldName := SanitizeName(field.Name)
				uuidMap[field.ID] = "data.elementum_field." + tableName + "_" + sanitizedFieldName + ".id"
			}
		}
	}

	for _, res := range relatedResources {
		sanitizedName := SanitizeName(res.Name)
		var ref string
		if res.ResourceType == "elementum_cloudlink" {
			// CloudLinks are exported as data sources, not resources
			ref = "data.elementum_cloudlink." + sanitizedName + ".id"
		} else {
			ref = res.ResourceType + "." + sanitizedName + ".id"
		}
		uuidMap[res.ID] = ref
	}

	return uuidMap
}

// beautifyPrimaryColumnIDArrays converts primary_column_ids arrays from UUIDs to references
func beautifyPrimaryColumnIDArrays(hcl string, uuidMap map[string]string) string {
	return primaryColIDsPattern.ReplaceAllStringFunc(hcl, func(match string) string {
		submatch := primaryColIDsPattern.FindStringSubmatch(match)
		if len(submatch) < 3 {
			return match
		}

		prefix := submatch[1]
		arrayContent := submatch[2]

		newContent := uuidPattern.ReplaceAllStringFunc(arrayContent, func(uuidMatch string) string {
			uuid := uuidMatch[1 : len(uuidMatch)-1]
			if ref, ok := uuidMap[uuid]; ok {
				return ref
			}
			return uuidMatch
		})

		return prefix + "[" + newContent + "]"
	})
}

// beautifyValueReferences converts raw trigger/task value references to refs syntax
func beautifyValueReferences(hcl string, uuidMap map[string]string, app *discovery.App) string {
	if app == nil {
		return hcl
	}

	// Build per-automation trigger ref maps (keyed by automation slug)
	// This prevents cross-automation contamination when multiple automations reference the same field
	perAutomationTriggerRefs := buildPerAutomationTriggerRefMap(app, uuidMap)

	// Build global trigger ref map as fallback (for backward compatibility with tests)
	globalTriggerRefMap := buildGlobalTriggerRefMap(app, uuidMap)

	// Build global task ref map (task refs include task ID, so they're unique)
	taskRefMap := buildTaskRefMap(app, uuidMap)

	// Process HCL line by line, tracking automation context from resource names
	lines := strings.Split(hcl, "\n")
	var result []string
	currentAutomationSlug := ""

	for _, line := range lines {
		// Check if this line declares a new resource
		if match := resourceDeclPatternAlt.FindStringSubmatch(line); len(match) >= 3 {
			resourceName := match[2]
			// Try to extract automation slug from resource name
			if slug := extractAutomationSlug(resourceName, app); slug != "" {
				currentAutomationSlug = slug
			}
		}

		// Replace trigger refs using automation-scoped map if available, otherwise global
		// First handle record-based trigger refs (trigger.record.<UUID>)
		line = triggerRefPattern.ReplaceAllStringFunc(line, func(match string) string {
			submatch := triggerRefPattern.FindStringSubmatch(match)
			if len(submatch) < 3 {
				return match
			}
			prefix := submatch[1]
			ref := submatch[2]

			// Try automation-scoped refs first
			if currentAutomationSlug != "" {
				if triggerRefs, ok := perAutomationTriggerRefs[currentAutomationSlug]; ok {
					if replacement, ok := triggerRefs[ref]; ok {
						return prefix + replacement
					}
				}
			}

			// Fallback to global refs
			if replacement, ok := globalTriggerRefMap[ref]; ok {
				return prefix + replacement
			}
			return match
		})

		// Then handle on_demand trigger parameter refs (trigger.<paramName>)
		line = triggerParamRefPattern.ReplaceAllStringFunc(line, func(match string) string {
			submatch := triggerParamRefPattern.FindStringSubmatch(match)
			if len(submatch) < 3 {
				return match
			}
			prefix := submatch[1]
			ref := submatch[2]

			// Try automation-scoped refs first
			if currentAutomationSlug != "" {
				if triggerRefs, ok := perAutomationTriggerRefs[currentAutomationSlug]; ok {
					if replacement, ok := triggerRefs[ref]; ok {
						return prefix + replacement
					}
				}
			}

			// Fallback to global refs
			if replacement, ok := globalTriggerRefMap[ref]; ok {
				return prefix + replacement
			}
			return match
		})

		result = append(result, line)
	}

	hcl = strings.Join(result, "\n")

	// Replace task refs globally (they include task ID, so unique)
	hcl = taskRefPatternGlobal.ReplaceAllStringFunc(hcl, func(match string) string {
		submatch := taskRefPatternGlobal.FindStringSubmatch(match)
		if len(submatch) < 3 {
			return match
		}
		prefix := submatch[1]
		ref := submatch[2]

		if replacement, ok := taskRefMap[ref]; ok {
			return prefix + replacement
		}
		return match
	})

	return hcl
}

// buildTriggerRefEntries builds ref map entries for a single trigger.
// Returns entries mapping raw refs (trigger.<path>) to Terraform refs syntax.
func buildTriggerRefEntries(trigger discovery.Trigger, uuidMap map[string]string) map[string]string {
	resourceType := "elementum_" + trigger.Type + "_trigger"
	resourceName := findResourceNameByID(uuidMap, trigger.ID, resourceType)
	if resourceName == "" {
		return nil
	}

	entries := make(map[string]string)
	for fieldRef, fieldName := range trigger.FieldRefs {
		// Skip id: prefixed refs (they're UUID-based, not path-based)
		if strings.HasPrefix(fieldRef, "id:") {
			continue
		}

		refsRef := fmt.Sprintf(`%s.%s.refs["%s"]`, resourceType, resourceName, fieldName)

		// Store both "trigger.<path>" and "trigger.record.<path>" formats
		// because different trigger types output different formats:
		// - record_created/updated triggers use: trigger.record.<UUID>.<suffix>
		// - on_demand triggers use: trigger.<paramName>
		rawRef := "trigger." + fieldRef
		entries[rawRef] = refsRef

		if !strings.HasPrefix(fieldRef, "record.") {
			// Also store with record. prefix for record-based triggers
			rawRefWithRecord := "trigger.record." + fieldRef
			entries[rawRefWithRecord] = refsRef
		}
	}
	return entries
}

// buildGlobalTriggerRefMap creates a global trigger ref map (for backward compatibility)
func buildGlobalTriggerRefMap(app *discovery.App, uuidMap map[string]string) map[string]string {
	refMap := make(map[string]string)

	for _, automation := range app.AllAutomations() {
		for _, trigger := range automation.Triggers {
			for k, v := range buildTriggerRefEntries(trigger, uuidMap) {
				refMap[k] = v
			}
		}
	}

	return refMap
}

// extractAutomationSlug extracts the automation slug from a resource name
// e.g., "digitize_document_update_record" -> "digitize_document"
func extractAutomationSlug(resourceName string, app *discovery.App) string {
	for _, automation := range app.AllAutomations() {
		slug := SanitizeName(automation.Name)
		if strings.HasPrefix(resourceName, slug+"_") {
			return slug
		}
	}
	return ""
}

// buildPerAutomationTriggerRefMap creates trigger ref maps keyed by automation slug
func buildPerAutomationTriggerRefMap(app *discovery.App, uuidMap map[string]string) map[string]map[string]string {
	result := make(map[string]map[string]string)

	for _, automation := range app.AllAutomations() {
		automationSlug := SanitizeName(automation.Name)
		triggerRefs := make(map[string]string)

		for _, trigger := range automation.Triggers {
			for k, v := range buildTriggerRefEntries(trigger, uuidMap) {
				triggerRefs[k] = v
			}
		}

		if len(triggerRefs) > 0 {
			result[automationSlug] = triggerRefs
		}
	}

	return result
}

// buildTaskRefMap creates a global task ref map (task refs include task ID, so unique)
func buildTaskRefMap(app *discovery.App, uuidMap map[string]string) map[string]string {
	refMap := make(map[string]string)

	for _, automation := range app.AllAutomations() {
		for _, task := range automation.Tasks {
			resourceType := "elementum_" + task.Type + "_task"
			resourceName := findResourceNameByID(uuidMap, task.ID, resourceType)
			if resourceName == "" {
				continue
			}

			for propPath, propName := range task.FieldRefs {
				rawRef := "task." + task.ID + "." + propPath
				refsRef := fmt.Sprintf(`%s.%s.refs["%s"]`, resourceType, resourceName, propName)
				refMap[rawRef] = refsRef
			}
		}
	}

	return refMap
}

// buildValueRefMap creates a mapping from raw value references to refs syntax
func buildValueRefMap(app *discovery.App, uuidMap map[string]string) map[string]string {
	refMap := make(map[string]string)

	for _, automation := range app.AllAutomations() {
		for _, trigger := range automation.Triggers {
			resourceType := "elementum_" + trigger.Type + "_trigger"
			resourceName := findResourceNameByID(uuidMap, trigger.ID, resourceType)
			if resourceName == "" {
				continue
			}

			for fieldRef, fieldName := range trigger.FieldRefs {
				rawRef := "trigger." + fieldRef
				if !strings.HasPrefix(fieldRef, "record.") {
					rawRef = "trigger.record." + fieldRef
				}

				refsRef := fmt.Sprintf(`%s.%s.refs["%s"]`, resourceType, resourceName, fieldName)
				refMap[rawRef] = refsRef
			}
		}

		for _, task := range automation.Tasks {
			resourceType := "elementum_" + task.Type + "_task"
			resourceName := findResourceNameByID(uuidMap, task.ID, resourceType)
			if resourceName == "" {
				continue
			}

			for propPath, propName := range task.FieldRefs {
				rawRef := "task." + task.ID + "." + propPath

				refsRef := fmt.Sprintf(`%s.%s.refs["%s"]`, resourceType, resourceName, propName)
				refMap[rawRef] = refsRef
			}
		}
	}

	return refMap
}

// findResourceNameByID extracts the resource name from the uuidMap for a given ID and type
func findResourceNameByID(uuidMap map[string]string, id, resourceType string) string {
	if ref, ok := uuidMap[id]; ok {
		if strings.HasPrefix(ref, resourceType+".") {
			ref = strings.TrimPrefix(ref, resourceType+".")
			ref = strings.TrimSuffix(ref, ".id")
			return ref
		}
	}
	return ""
}

// GenerateStageLocals generates a locals block for stage ID lookups (legacy, calls GenerateLocals)
// Deprecated: Use GenerateLocals instead
func GenerateStageLocals(app *discovery.App, appResourceName string) string {
	return GenerateLocals(app, appResourceName, nil)
}

// GenerateLocals generates a comprehensive locals block for all lookup maps:
// - Stage IDs by key
// - System field IDs (title, status, stage, id, audit fields)
// - Status option IDs by label
// - Dropdown/multi_select field option IDs by label
// - Element system field locals for discovered elements
func GenerateLocals(app *discovery.App, appResourceName string, imports []ImportBlock) string {
	if app == nil {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("# Lookup maps for " + appResourceName + " app system resources\n")
	sb.WriteString("locals {\n")

	prefix := appResourceName + "_"

	// Stage IDs by key (if app has layouts/stages)
	if len(app.Layouts) > 0 {
		sb.WriteString("  # Stage IDs by key (uses computed stage_ids_by_key from app)\n")
		sb.WriteString("  " + prefix + "stage_ids_by_key = elementum_app." + appResourceName + ".stage_ids_by_key\n\n")
	}

	// System field IDs (core fields)
	sb.WriteString("  # System field IDs (core fields)\n")
	sb.WriteString("  " + prefix + "title_field_id  = elementum_app." + appResourceName + ".title_field_id\n")
	sb.WriteString("  " + prefix + "status_field_id = elementum_app." + appResourceName + ".status_field_id\n")
	if len(app.Layouts) > 0 {
		sb.WriteString("  " + prefix + "stage_field_id  = elementum_app." + appResourceName + ".stage_field_id\n")
	}
	sb.WriteString("  " + prefix + "id_field_id     = elementum_app." + appResourceName + ".id_field_id\n\n")

	// Audit trail field IDs
	sb.WriteString("  # Audit trail field IDs\n")
	sb.WriteString("  " + prefix + "created_by_field_id = elementum_app." + appResourceName + ".created_by_field_id\n")
	sb.WriteString("  " + prefix + "created_at_field_id = elementum_app." + appResourceName + ".created_at_field_id\n")
	sb.WriteString("  " + prefix + "updated_by_field_id = elementum_app." + appResourceName + ".updated_by_field_id\n")
	sb.WriteString("  " + prefix + "updated_at_field_id = elementum_app." + appResourceName + ".updated_at_field_id\n")
	sb.WriteString("  " + prefix + "closed_by_field_id  = elementum_app." + appResourceName + ".closed_by_field_id\n")
	sb.WriteString("  " + prefix + "closed_at_field_id  = elementum_app." + appResourceName + ".closed_at_field_id\n\n")

	// Status option IDs by label (uses for-expression on status_options)
	sb.WriteString("  # Status option IDs by label (for looking up status option IDs by their display label)\n")
	sb.WriteString("  " + prefix + "status_options_by_label = {\n")
	sb.WriteString("    for opt in elementum_app." + appResourceName + ".status_options : opt.label => opt.id\n")
	sb.WriteString("  }\n\n")

	// Stage option IDs by label (uses for-expression on stages, if app has stages)
	if len(app.Layouts) > 0 {
		sb.WriteString("  # Stage option IDs by label (for looking up stage option IDs by their display name)\n")
		sb.WriteString("  " + prefix + "stage_options_by_label = {\n")
		sb.WriteString("    for s in elementum_app." + appResourceName + ".stages : s.name => s.id\n")
		sb.WriteString("  }\n\n")
	}

	// Dropdown/multi_select field option lookup maps
	if imports != nil {
		dropdownFields := findDropdownFieldImports(imports)
		if len(dropdownFields) > 0 {
			sb.WriteString("  # Dropdown/multi-select field option IDs by label\n")
			for _, field := range dropdownFields {
				resourceType := "elementum_dropdown_field"
				if strings.Contains(field.ResourceType, "multi_select") {
					resourceType = "elementum_multi_select_field"
				}
				sb.WriteString("  " + field.ResourceName + "_options_by_label = {\n")
				sb.WriteString("    for opt in " + resourceType + "." + field.ResourceName + ".options : opt.label => opt.id\n")
				sb.WriteString("  }\n")
			}
			sb.WriteString("\n")
		}
	}

	// Element system field locals for discovered elements
	if len(app.DiscoveredElements) > 0 {
		sb.WriteString("  # Discovered element system field IDs\n")
		for _, element := range app.DiscoveredElements {
			// Use Namespace for prefix if available, otherwise Handle, otherwise Name
			elementPrefix := SanitizeName(element.Namespace)
			if elementPrefix == "" {
				elementPrefix = SanitizeName(element.Handle)
			}
			if elementPrefix == "" {
				elementPrefix = SanitizeName(element.Name)
			}

			elementResourceName := elementPrefix
			elementLocalPrefix := elementPrefix + "_"

			sb.WriteString("\n  # " + element.Name + " element system field IDs\n")
			sb.WriteString("  " + elementLocalPrefix + "title_field_id  = elementum_element." + elementResourceName + ".title_field_id\n")
			sb.WriteString("  " + elementLocalPrefix + "id_field_id     = elementum_element." + elementResourceName + ".id_field_id\n")
		}
		sb.WriteString("\n")
	}

	// App system field locals for discovered apps (full exports)
	if len(app.DiscoveredApps) > 0 {
		sb.WriteString("  # Discovered app system field IDs\n")
		for _, discoveredApp := range app.DiscoveredApps {
			// Use Namespace for prefix if available, otherwise Name
			appPrefix := SanitizeName(discoveredApp.Namespace)
			if appPrefix == "" {
				appPrefix = SanitizeName(discoveredApp.Name)
			}

			appResourceName := appPrefix
			appLocalPrefix := appPrefix + "_"

			sb.WriteString("\n  # " + discoveredApp.Name + " app system field IDs\n")
			sb.WriteString("  " + appLocalPrefix + "title_field_id      = elementum_app." + appResourceName + ".title_field_id\n")
			sb.WriteString("  " + appLocalPrefix + "status_field_id     = elementum_app." + appResourceName + ".status_field_id\n")
			sb.WriteString("  " + appLocalPrefix + "id_field_id         = elementum_app." + appResourceName + ".id_field_id\n")
			sb.WriteString("  " + appLocalPrefix + "created_by_field_id = elementum_app." + appResourceName + ".created_by_field_id\n")
			sb.WriteString("  " + appLocalPrefix + "created_at_field_id = elementum_app." + appResourceName + ".created_at_field_id\n")
			sb.WriteString("  " + appLocalPrefix + "updated_by_field_id = elementum_app." + appResourceName + ".updated_by_field_id\n")
			sb.WriteString("  " + appLocalPrefix + "updated_at_field_id = elementum_app." + appResourceName + ".updated_at_field_id\n")
			sb.WriteString("  " + appLocalPrefix + "closed_by_field_id  = elementum_app." + appResourceName + ".closed_by_field_id\n")
			sb.WriteString("  " + appLocalPrefix + "closed_at_field_id  = elementum_app." + appResourceName + ".closed_at_field_id\n")

			// Status options lookup for discovered apps
			sb.WriteString("  " + appLocalPrefix + "status_options_by_label = {\n")
			sb.WriteString("    for opt in elementum_app." + appResourceName + ".status_options : opt.label => opt.id\n")
			sb.WriteString("  }\n")

			// Stage options lookup for discovered apps (if they have stages)
			if len(discoveredApp.Layouts) > 0 {
				sb.WriteString("  " + appLocalPrefix + "stage_field_id = elementum_app." + appResourceName + ".stage_field_id\n")
				sb.WriteString("  " + appLocalPrefix + "stage_options_by_label = {\n")
				sb.WriteString("    for s in elementum_app." + appResourceName + ".stages : s.name => s.id\n")
				sb.WriteString("  }\n")
			}
		}
		sb.WriteString("\n")
	}

	// Table field ID lookup maps for discovered tables (for datamine primary_column_ids)
	if len(app.DiscoveredTables) > 0 {
		sb.WriteString("  # Table field ID lookups (for datamine primary_column_ids)\n")
		for _, table := range app.DiscoveredTables {
			tableName := findResourceName(imports, "elementum_table", table.ID)
			if tableName == "" {
				tableName = SanitizeName(table.Name)
			}
			sb.WriteString("  " + tableName + "_field_ids_by_name = {\n")
			sb.WriteString("    for f in elementum_table." + tableName + ".fields : f.name => f.id\n")
			sb.WriteString("  }\n")
		}
		sb.WriteString("\n")
	}

	sb.WriteString("}\n\n")

	return sb.String()
}

// findDropdownFieldImports finds all dropdown and multi_select field imports
func findDropdownFieldImports(imports []ImportBlock) []ImportBlock {
	var dropdowns []ImportBlock
	for _, imp := range imports {
		if imp.ResourceType == "elementum_dropdown_field" ||
			imp.ResourceType == "elementum_multi_select_field" {
			dropdowns = append(dropdowns, imp)
		}
	}
	return dropdowns
}

// beautifyAutomationIDs applies ID resolution to all automation resources.
//
// Deprecated: This function is no longer called from PostProcessHCL because the IR pipeline's
// ResolveUUIDs transform already handles UUID→reference resolution at the block level.
// Kept for backward compatibility with any external callers.
func beautifyAutomationIDs(hcl string, uuidMap map[string]string, imports []ImportBlock, automations []discovery.Automation) string {
	// Resolve automation_id, parent_id, object_id, field_id references
	for _, automation := range automations {
		automationResourceName := ""
		for _, imp := range imports {
			if imp.ResourceType == "elementum_automation" && strings.Contains(imp.ID, automation.ID) {
				automationResourceName = imp.ResourceName
				break
			}
		}

		// Beautify triggers
		for _, trigger := range automation.Triggers {
			hcl = beautifyTriggerIDs(hcl, &trigger, automationResourceName, uuidMap, imports)
		}

		// Beautify tasks
		for _, task := range automation.Tasks {
			hcl = beautifyTaskIDs(hcl, &task, uuidMap, imports)
		}
	}

	return hcl
}

// beautifyTriggerIDs resolves IDs in a trigger resource block
func beautifyTriggerIDs(hcl string, trigger *discovery.Trigger, automationResourceName string, uuidMap map[string]string, imports []ImportBlock) string {
	triggerType := trigger.Type
	resourceType := "elementum_" + triggerType + "_trigger"

	triggerResourceName := ""
	for _, imp := range imports {
		if imp.ResourceType == resourceType && strings.HasSuffix(imp.ID, ":"+trigger.ID) {
			triggerResourceName = imp.ResourceName
			break
		}
	}
	if triggerResourceName == "" {
		return hcl
	}

	// Replace UUIDs with references in this specific resource block. Skip
	// empty-string keys — fmt.Sprintf("%q", "") is `""`, and ReplaceAll would
	// then overwrite every quoted empty string in the HCL (e.g. `description
	// = ""`) with whatever ref was last stored under the empty key.
	for uuid, ref := range uuidMap {
		if uuid == "" {
			continue
		}
		hcl = strings.ReplaceAll(hcl, fmt.Sprintf("%q", uuid), ref)
	}

	return hcl
}

// beautifyTaskIDs resolves IDs in a task resource block
func beautifyTaskIDs(hcl string, task *discovery.Task, uuidMap map[string]string, imports []ImportBlock) string {
	taskType := task.Type
	resourceType := "elementum_" + taskType + "_task"

	taskResourceName := ""
	for _, imp := range imports {
		if imp.ResourceType == resourceType && strings.HasSuffix(imp.ID, ":"+task.ID) {
			taskResourceName = imp.ResourceName
			break
		}
	}
	if taskResourceName == "" {
		return hcl
	}

	// Replace UUIDs with references in this specific resource block. Skip
	// empty-string keys — fmt.Sprintf("%q", "") is `""`, and ReplaceAll would
	// then overwrite every quoted empty string in the HCL (e.g. `description
	// = ""`) with whatever ref was last stored under the empty key.
	for uuid, ref := range uuidMap {
		if uuid == "" {
			continue
		}
		hcl = strings.ReplaceAll(hcl, fmt.Sprintf("%q", uuid), ref)
	}

	return hcl
}
