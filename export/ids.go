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
	"github.com/elementumltd/elementum-cli/discovery"
)

// ImportIDFormat maps resource types to their import ID format strings
var ImportIDFormats = map[string]string{
	"elementum_app":                              "{app_id}",
	"elementum_element":                          "{element_id}",
	"elementum_group":                            "{group_id}",
	"elementum_handle_field":                     "{app_id}:{field_id}",
	"elementum_text_field":                       "{app_id}:{field_id}",
	"elementum_longtext_field":                   "{app_id}:{field_id}",
	"elementum_number_field":                     "{app_id}:{field_id}",
	"elementum_decimal_field":                    "{app_id}:{field_id}",
	"elementum_boolean_field":                    "{app_id}:{field_id}",
	"elementum_date_field":                       "{app_id}:{field_id}",
	"elementum_datetime_field":                   "{app_id}:{field_id}",
	"elementum_dropdown_field":                   "{app_id}:{field_id}",
	"elementum_multiselect_field":                "{app_id}:{field_id}",
	"elementum_user_field":                       "{app_id}:{field_id}",
	"elementum_group_field":                      "{app_id}:{field_id}",
	"elementum_relation_field":                   "{app_id}:{field_id}",
	"elementum_attachment_field":                 "{app_id}:{field_id}",
	"elementum_calculated_field":                 "{app_id}:{field_id}",
	"elementum_json_field":                       "{app_id}:{field_id}",
	"elementum_layout":                           "{app_id}:{layout_id}",
	"elementum_flow":                             "{app_id}:{flow_id}",
	"elementum_automation":                       "{app_id}:{automation_id}",
	"elementum_webhook_trigger":                  "{automation_id}:{trigger_id}",
	"elementum_record_created_trigger":           "{automation_id}:{trigger_id}",
	"elementum_record_updated_trigger":           "{automation_id}:{trigger_id}",
	"elementum_on_demand_trigger":                "{automation_id}:{trigger_id}",
	"elementum_slack_message_trigger":            "{automation_id}:{trigger_id}",
	"elementum_email_ingestion_trigger":          "{automation_id}:{trigger_id}",
	"elementum_attachment_added_trigger":         "{automation_id}:{trigger_id}",
	"elementum_comment_added_trigger":            "{automation_id}:{trigger_id}",
	"elementum_agent_conversation_ended_trigger": "{automation_id}:{trigger_id}",
	"elementum_approval_chain_trigger":           "{automation_id}:{trigger_id}",
	"elementum_datamine_trigger":                 "{automation_id}:{trigger_id}",
	"elementum_message_task":                     "{workflow_id}:{task_id}",
	"elementum_update_field_task":                "{workflow_id}:{task_id}",
	"elementum_variable_task":                    "{workflow_id}:{task_id}",
	"elementum_update_variable_task":             "{workflow_id}:{task_id}",
	"elementum_ai_search_table_task":             "{workflow_id}:{task_id}",
	"elementum_procedure_task":                   "{workflow_id}:{task_id}",
	"elementum_create_record_task":               "{workflow_id}:{task_id}",
	"elementum_record_search_task":               "{workflow_id}:{task_id}",
	"elementum_send_email_task":                  "{workflow_id}:{task_id}",
	"elementum_api_task":                         "{workflow_id}:{task_id}",
	"elementum_approval_chain_task":              "{workflow_id}:{task_id}",
	"elementum_approval_status_update_task":      "{workflow_id}:{task_id}",
	"elementum_find_related_records_task":        "{workflow_id}:{task_id}",
	"elementum_ai_agent_task":                    "{workflow_id}:{task_id}",
	"elementum_for_each_task":                    "{workflow_id}:{task_id}",
	"elementum_switch_task":                      "{workflow_id}:{task_id}",
	"elementum_ai_file_read_task":                "{workflow_id}:{task_id}",
	"elementum_add_watcher_task":                 "{workflow_id}:{task_id}",
	"elementum_ai_classify_task":                 "{workflow_id}:{task_id}",
	"elementum_aspect_record_field_locking_task": "{workflow_id}:{task_id}",

	"elementum_ai_summarize_task":           "{workflow_id}:{task_id}",
	"elementum_ai_transform_task":           "{workflow_id}:{task_id}",
	"elementum_calculation_task":            "{workflow_id}:{task_id}",
	"elementum_file_reader_task":            "{workflow_id}:{task_id}",
	"elementum_notification_task":           "{workflow_id}:{task_id}",
	"elementum_relate_records_task":         "{workflow_id}:{task_id}",
	"elementum_save_attachment_task":        "{workflow_id}:{task_id}",
	"elementum_user_search_task":            "{workflow_id}:{task_id}",
	"elementum_run_automation_task":         "{workflow_id}:{task_id}",
	"elementum_execute_script_task":         "{workflow_id}:{task_id}",
	"elementum_bulk_excel_task":             "{workflow_id}:{task_id}",
	"elementum_switch_case":                 "{case_id}",
	"elementum_fork_join_branch":            "{branch_id}",
	"elementum_fork_join_task":              "{workflow_id}:{task_id}",
	"elementum_agent":                       "{app_id}:{agent_id}",
	"elementum_agent_create_record_tool":    "{app_id}:{agent_id}:{tool_id}",
	"elementum_agent_search_records_tool":   "{app_id}:{agent_id}:{tool_id}",
	"elementum_agent_update_record_tool":    "{app_id}:{agent_id}:{tool_id}",
	"elementum_agent_ai_search_tool":        "{app_id}:{agent_id}:{tool_id}",
	"elementum_agent_relate_record_tool":    "{app_id}:{agent_id}:{tool_id}",
	"elementum_agent_run_automation_tool":   "{app_id}:{agent_id}:{tool_id}",
	"elementum_agent_run_agent_tool":        "{app_id}:{agent_id}:{tool_id}",
	"elementum_agent_select_bot_route_tool": "{app_id}:{agent_id}:{tool_id}",
	"elementum_agent_mcp_tool":              "{app_id}:{agent_id}:{tool_id}",
	"elementum_widget":                      "{app_id}:{widget_id}",
	"elementum_approval_process":            "{app_id}:{approval_id}",
	"elementum_ai_file_reader":              "{app_id}:{file_reader_id}",
	"elementum_text_file_reader":            "{app_id}:{file_reader_id}",
	"elementum_json_file_reader":            "{app_id}:{file_reader_id}",
	"elementum_xml_file_reader":             "{app_id}:{file_reader_id}",
	"elementum_cloudlink":                   "{cloudlink_id}",
	"elementum_email_alias":                 "{alias_id}",
	"elementum_datamine":                    "{datamine_id}",
	"elementum_table":                       "{table_id}",
	"elementum_relationship":                "{object_id}:{relationship_id}",
	"elementum_role":                        "{object_id}:{role_id}",
	"elementum_access_policy":               "{object_id}:{policy_id}",
	"elementum_phone_provider":              "{provider_id}",
	"elementum_phone_service":               "{app_id}:{service_id}",
	"elementum_ai_search_table":             "{object_id}/{search_table_id}",
	"elementum_table_search_table":          "{table_id}/{search_table_id}",
	"elementum_managed_view_order":          "{object_id}",
	"elementum_dashboard":                   "{dashboard_id}",
	"elementum_dashboard_widget":            "{dashboard_id}:{widget_id}",
	"elementum_workflow_publish":            "{automation_id}",
	"elementum_agentic_skill":               "{skill_id}",
	"elementum_agentic_skill_tool":          "{skill_id}:{tool_id}",
	"elementum_agent_a2a_skill":             "{agent_id}:{skill_id}",
}

// BuildImportID builds an import ID for a specific resource
func BuildImportID(resourceType string, params map[string]string) string {
	format, exists := ImportIDFormats[resourceType]
	if !exists {
		return params["id"] // Fallback to just the ID
	}

	// Replace placeholders with actual values
	id := format
	for key, value := range params {
		placeholder := "{" + key + "}"
		id = replaceAll(id, placeholder, value)
	}

	return id
}

// BuildFieldImportID builds an import ID for a field
func BuildFieldImportID(appID string, fieldID string, fieldType string) string {
	resourceType := "elementum_" + fieldType + "_field"
	return BuildImportID(resourceType, map[string]string{
		"app_id":   appID,
		"field_id": fieldID,
	})
}

// BuildTriggerImportID builds an import ID for a trigger
func BuildTriggerImportID(automationID string, triggerID string, triggerType string) string {
	resourceType := "elementum_" + triggerType + "_trigger"
	return BuildImportID(resourceType, map[string]string{
		"automation_id": automationID,
		"trigger_id":    triggerID,
	})
}

// BuildTaskImportID builds an import ID for a task
func BuildTaskImportID(workflowID string, taskID string, taskType string) string {
	resourceType := "elementum_" + taskType + "_task"
	return BuildImportID(resourceType, map[string]string{
		"workflow_id": workflowID,
		"task_id":     taskID,
	})
}

// GetResourceTypeForField returns the Terraform resource type for a field
func GetResourceTypeForField(field discovery.Field) string {
	return "elementum_" + field.Type + "_field"
}

// GetResourceTypeForTrigger returns the Terraform resource type for a trigger
func GetResourceTypeForTrigger(trigger discovery.Trigger) string {
	return "elementum_" + trigger.Type + "_trigger"
}

// GetResourceTypeForTask returns the Terraform resource type for a task
func GetResourceTypeForTask(task discovery.Task) string {
	return "elementum_" + task.Type + "_task"
}

// Simple string replace helper
func replaceAll(s, old, new string) string {
	result := ""
	for {
		idx := indexOf(s, old)
		if idx == -1 {
			result += s
			break
		}
		result += s[:idx] + new
		s = s[idx+len(old):]
	}
	return result
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
