// Copyright (c) ElementumAI, Inc.
// SPDX-License-Identifier: MPL-2.0

package analysis

import (
	"fmt"
	"strings"
)

// TaskPriorityFields defines which fields are most important for each task type.
// These fields are shown first/prominently in the UI.
var TaskPriorityFields = map[string][]string{
	"api":                  {"method", "urlReference", "responseType", "authorization", "continueOnError"},
	"message":              {"contentsReference", "recordReference"},
	"send_email":           {"subject", "messageBody", "emailUsers", "emailGroups", "externalEmails"},
	"variable":             {"variables"},
	"update_field":         {"aspect", "recordReference", "workflowFields"},
	"create_record":        {"aspect", "relateToRecord", "workflowFields"},
	"record_search":        {"aspect", "limit", "filter"},
	"ai_agent":             {"agent", "agentPrompt", "outputType"},
	"for_each":             {"forEach"},
	"switch":               {"children"},
	"notification":         {"titleReference", "messageReference", "notifyWatchers"},
	"save_attachment":      {"recordId", "file", "tag"},
	"calculation":          {"calculations"},
	"user_search":          {"filter"},
	"relate_records":       {"sourceRecordIds", "targetRecordId"},
	"find_related_records": {"recordId", "relatedAspect", "limit"},
	"ai_classify":          {"textToClassify", "categorySource", "context"},
	"ai_summarize":         {"textToSummarize", "context", "length"},
	"ai_transform":         {"prompt", "fieldType"},
	"ai_file_read":         {"documentModel", "attachment"},
	"ai_search_table":      {"searchTable", "query", "limit"},
	"procedure":            {"cloudLinkId", "storedFunction", "parameters"},
	"run_automation":       {"automation", "synchronous", "inputMappings"},
	"add_watcher":          {"recordReference", "watcherUsers", "watcherGroups"},
	"approval_chain":       {"approvers", "reason"},
	"bulk_excel":           {"aspect", "attachment", "documentModel"},
}

// FormatValue formats a value for display, handling value references,
// template references, trigger/task references, and complex objects.
func FormatValue(v interface{}) string {
	if v == nil {
		return ""
	}

	switch val := v.(type) {
	case string:
		return val
	case float64:
		if val == float64(int(val)) {
			return fmt.Sprintf("%d", int(val))
		}
		return fmt.Sprintf("%v", val)
	case bool:
		return fmt.Sprintf("%v", val)
	case map[string]interface{}:
		return formatMapValue(val)
	case []interface{}:
		return formatArrayValue(val)
	default:
		return fmt.Sprintf("%v", val)
	}
}

// formatMapValue handles map/object values including value references
func formatMapValue(m map[string]interface{}) string {
	// Value reference pattern: {id, label, value}
	if label, ok := m["label"].(string); ok && label != "" {
		return label
	}

	// Template reference pattern: {template, parameters}
	if tmpl, ok := m["templateReference"].(map[string]interface{}); ok {
		return formatTemplateReference(tmpl)
	}

	// Direct trigger reference: {triggerReference: {name}}
	if trigRef, ok := m["triggerReference"].(map[string]interface{}); ok {
		if name, ok := trigRef["name"].(string); ok {
			return "trigger." + name
		}
	}

	// Direct task reference: {taskReference: {name, task: {id}}}
	if taskRef, ok := m["taskReference"].(map[string]interface{}); ok {
		if name, ok := taskRef["name"].(string); ok {
			return "task." + name
		}
	}

	// Variable reference: {variableReference: {variableId, name, type}}
	if varRef, ok := m["variableReference"].(map[string]interface{}); ok {
		if name, ok := varRef["name"].(string); ok {
			return "var." + name
		}
	}

	// ForEach reference: {forEachReference: {name}}
	if forRef, ok := m["forEachReference"].(map[string]interface{}); ok {
		if name, ok := forRef["name"].(string); ok {
			return "loop." + name
		}
	}

	// Aspect/entity reference with name
	if name, ok := m["name"].(string); ok && name != "" {
		return name
	}

	// Authorization object
	if typename, ok := m["__typename"].(string); ok {
		return formatAuthType(typename, m)
	}

	// Body object
	if typename, ok := m["__typename"].(string); ok && strings.HasPrefix(typename, "Api") {
		return formatBodyType(typename, m)
	}

	// Filter object
	if _, hasType := m["type"]; hasType {
		if _, hasField := m["field"]; hasField {
			return formatFilterBrief(m)
		}
		if _, hasFieldID := m["fieldId"]; hasFieldID {
			return formatFilterBrief(m)
		}
		if _, hasChildren := m["children"]; hasChildren {
			return formatFilterBrief(m)
		}
	}

	// Generic object with field count
	if len(m) > 0 {
		return fmt.Sprintf("{%d fields}", len(m))
	}
	return "{}"
}

// formatArrayValue formats array values
func formatArrayValue(arr []interface{}) string {
	if len(arr) == 0 {
		return "[]"
	}

	// For small arrays of simple values, show them
	if len(arr) <= 3 {
		var parts []string
		allSimple := true
		for _, item := range arr {
			formatted := FormatValue(item)
			if len(formatted) > 30 {
				allSimple = false
				break
			}
			parts = append(parts, formatted)
		}
		if allSimple {
			return "[" + strings.Join(parts, ", ") + "]"
		}
	}

	// Otherwise show count
	return fmt.Sprintf("[%d items]", len(arr))
}

// formatTemplateReference formats a template reference with parameters
func formatTemplateReference(tmpl map[string]interface{}) string {
	template, _ := tmpl["template"].(string)
	if template == "" {
		return "{template}"
	}

	// Try to substitute parameters
	params, ok := tmpl["parameters"].([]interface{})
	if !ok || len(params) == 0 {
		return template
	}

	result := template
	for _, p := range params {
		pm, ok := p.(map[string]interface{})
		if !ok {
			continue
		}
		name, _ := pm["name"].(string)
		if name == "" {
			continue
		}

		// Get the value reference
		valueRef, _ := pm["value"].(map[string]interface{})
		replacement := FormatValue(valueRef)
		if replacement == "" {
			replacement = "{" + name + "}"
		}

		result = strings.ReplaceAll(result, "{{"+name+"}}", replacement)
	}

	return result
}

// formatAuthType formats an authorization block
func formatAuthType(typename string, m map[string]interface{}) string {
	switch typename {
	case "ApiBearerAuthorization":
		return "Bearer Token"
	case "ApiBasicAuthorization":
		if username, ok := m["username"].(string); ok && username != "" {
			return fmt.Sprintf("Basic (%s)", username)
		}
		return "Basic Auth"
	case "ApiOauthAuthorization":
		if url, ok := m["url"].(string); ok && url != "" {
			return fmt.Sprintf("OAuth (%s)", truncate(url, 30))
		}
		return "OAuth 2.0"
	}
	return typename
}

// formatBodyType formats a body type block
func formatBodyType(typename string, m map[string]interface{}) string {
	switch typename {
	case "ApiCustomBody":
		if ct, ok := m["contentType"].(string); ok {
			return fmt.Sprintf("Custom (%s)", ct)
		}
		return "Custom Body"
	case "ApiJsonBody":
		return "JSON Body"
	case "ApiFormBody":
		if values, ok := m["values"].([]interface{}); ok {
			return fmt.Sprintf("Form (%d fields)", len(values))
		}
		return "Form Body"
	case "ApiMultipartFormBody":
		if parts, ok := m["parts"].([]interface{}); ok {
			return fmt.Sprintf("Multipart (%d parts)", len(parts))
		}
		return "Multipart Form"
	}
	return typename
}

// formatFilterBrief returns a brief description of a filter
func formatFilterBrief(m map[string]interface{}) string {
	filterType, _ := m["type"].(string)

	switch filterType {
	case "and":
		if children, ok := m["children"].([]interface{}); ok {
			return fmt.Sprintf("AND (%d conditions)", len(children))
		}
		return "AND"
	case "or":
		if children, ok := m["children"].([]interface{}); ok {
			return fmt.Sprintf("OR (%d conditions)", len(children))
		}
		return "OR"
	default:
		// Comparison filter
		fieldName := ""
		if field, ok := m["field"].(map[string]interface{}); ok {
			fieldName, _ = field["name"].(string)
		}
		if fieldName == "" {
			fieldName, _ = m["fieldId"].(string)
		}
		if fieldName == "" {
			fieldName = "field"
		}

		return fmt.Sprintf("%s %s ...", fieldName, filterType)
	}
}

// FormatFilter formats a filter structure for detailed display
func FormatFilter(filter interface{}, indent string) string {
	m, ok := filter.(map[string]interface{})
	if !ok {
		return ""
	}

	filterType, _ := m["type"].(string)

	switch filterType {
	case "and", "or":
		children, _ := m["children"].([]interface{})
		if len(children) == 0 {
			return strings.ToUpper(filterType) + " (empty)"
		}

		var parts []string
		for _, child := range children {
			childStr := FormatFilter(child, indent+"  ")
			if childStr != "" {
				parts = append(parts, childStr)
			}
		}

		connector := "\n" + indent + strings.ToUpper(filterType) + " "
		return strings.Join(parts, connector)

	default:
		// Comparison filter
		fieldName := getFilterFieldName(m)
		value := getFilterValue(m)

		return fmt.Sprintf("%s %s %s", fieldName, filterType, value)
	}
}

func getFilterFieldName(m map[string]interface{}) string {
	if field, ok := m["field"].(map[string]interface{}); ok {
		if name, ok := field["name"].(string); ok {
			return name
		}
	}
	if fieldID, ok := m["fieldId"].(string); ok {
		return fieldID
	}
	return "field"
}

func getFilterValue(m map[string]interface{}) string {
	if value, ok := m["value"]; ok {
		return FormatValue(value)
	}
	if valueRef, ok := m["valueReference"]; ok {
		return FormatValue(valueRef)
	}
	if values, ok := m["values"].([]interface{}); ok {
		return formatArrayValue(values)
	}
	return "..."
}

// FormatWorkflowFields formats a workflowFields array for display
func FormatWorkflowFields(fields []interface{}) []string {
	var result []string
	for _, f := range fields {
		fm, ok := f.(map[string]interface{})
		if !ok {
			continue
		}

		fieldName := ""
		if field, ok := fm["field"].(map[string]interface{}); ok {
			fieldName, _ = field["name"].(string)
		}
		if fieldName == "" {
			fieldName = "field"
		}

		valueStr := ""
		if valueRef, ok := fm["valueReference"]; ok {
			valueStr = FormatValue(valueRef)
		}

		result = append(result, fmt.Sprintf("%s = %s", fieldName, valueStr))
	}
	return result
}

// FormatVariables formats a variables array from variable tasks
func FormatVariables(variables []interface{}) []string {
	var result []string
	for _, v := range variables {
		vm, ok := v.(map[string]interface{})
		if !ok {
			continue
		}

		typename, _ := vm["__typename"].(string)

		if typename == "WorkflowVariableTaskParameterCreate" {
			name, _ := vm["name"].(string)
			varType, _ := vm["type"].(string)
			value := FormatValue(vm["createValue"])
			result = append(result, fmt.Sprintf("%s (%s) = %s", name, varType, value))
		} else if typename == "WorkflowVariableTaskParameterUpdate" {
			varRef := FormatValue(vm["variable"])
			value := FormatValue(vm["updateValue"])
			result = append(result, fmt.Sprintf("%s = %s", varRef, value))
		}
	}
	return result
}

// FormatHeaders formats API headers for display
func FormatHeaders(headers []interface{}) []string {
	var result []string
	for _, h := range headers {
		hm, ok := h.(map[string]interface{})
		if !ok {
			continue
		}
		header, _ := hm["header"].(string)
		value := FormatValue(hm["value"])
		result = append(result, fmt.Sprintf("%s: %s", header, value))
	}
	return result
}

// FormatCalculations formats calculation definitions
func FormatCalculations(calcs []interface{}) []string {
	var result []string
	for _, c := range calcs {
		cm, ok := c.(map[string]interface{})
		if !ok {
			continue
		}
		name, _ := cm["name"].(string)
		calcRef := FormatValue(cm["calculationReference"])
		result = append(result, fmt.Sprintf("%s = %s", name, calcRef))
	}
	return result
}

// GetPriorityDetails returns the priority fields for a task type, formatted
func GetPriorityDetails(taskType string, details map[string]any) map[string]string {
	result := make(map[string]string)

	priorityFields := TaskPriorityFields[taskType]
	if len(priorityFields) == 0 {
		// Fall back to all fields
		for k, v := range details {
			formatted := FormatValue(v)
			if formatted != "" {
				result[k] = formatted
			}
		}
		return result
	}

	// Get priority fields first
	for _, field := range priorityFields {
		if v, ok := details[field]; ok {
			formatted := FormatValue(v)
			if formatted != "" {
				result[field] = formatted
			}
		}
	}

	return result
}

// GetAllFormattedDetails returns all details formatted
func GetAllFormattedDetails(details map[string]any) map[string]string {
	result := make(map[string]string)
	for k, v := range details {
		formatted := FormatValue(v)
		if formatted != "" {
			result[k] = formatted
		}
	}
	return result
}

// FieldDisplayName returns a human-readable name for a field
var FieldDisplayNames = map[string]string{
	"urlReference":        "URL",
	"contentsReference":   "Message",
	"messageReference":    "Message",
	"titleReference":      "Title",
	"messageBody":         "Body",
	"agentPrompt":         "Prompt",
	"aspect":              "Object",
	"recordReference":     "Record",
	"workflowFields":      "Fields",
	"forEach":             "List",
	"sourceRecordIds":     "Source Records",
	"targetRecordId":      "Target Record",
	"relatedAspect":       "Related Object",
	"emailUsers":          "Users",
	"emailGroups":         "Groups",
	"externalEmails":      "External Emails",
	"textToClassify":      "Text",
	"textToSummarize":     "Text",
	"categorySource":      "Categories",
	"aiProviderConnector": "AI Provider",
	"storedFunction":      "Function",
	"cloudLinkId":         "CloudLink",
	"searchTable":         "Search Table",
	"continueOnError":     "Continue On Error",
	"responseType":        "Response Type",
}

// GetFieldDisplayName returns the display name for a field
func GetFieldDisplayName(field string) string {
	if name, ok := FieldDisplayNames[field]; ok {
		return name
	}
	// Convert camelCase to Title Case
	return toTitleCase(field)
}

func toTitleCase(s string) string {
	if s == "" {
		return s
	}

	var result strings.Builder
	result.WriteRune(rune(strings.ToUpper(string(s[0]))[0]))

	for i := 1; i < len(s); i++ {
		if s[i] >= 'A' && s[i] <= 'Z' {
			result.WriteRune(' ')
		}
		result.WriteByte(s[i])
	}

	return result.String()
}
