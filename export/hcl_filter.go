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
	"encoding/json"
	"fmt"
	"strings"
)

// FilterHCLGenerator generates HCL for filter objects using attribute syntax.
// This is the single source of truth for filter HCL generation across all resources
// (access policies, triggers, tasks, datamines, etc.)
//
// The filter schema uses attribute syntax (filter = { ... }) with these key fields:
//   - type: Filter type (and, or, equals, in, like, etc.)
//   - field_id: Direct field ID to compare
//   - left_value: Left side of comparison (for complex comparisons)
//   - value: Single value to compare against
//   - values: Array of values (for 'in' operator)
//   - children: Nested filters (for and/or)
//   - relative_value: Relative date/time value
//   - mode: Like mode (contains, starts_with, ends_with)
//   - not: Negate the filter
type FilterHCLGenerator struct {
	uuidMap map[string]string
}

// NewFilterHCLGenerator creates a new filter HCL generator
func NewFilterHCLGenerator(uuidMap map[string]string) *FilterHCLGenerator {
	if uuidMap == nil {
		uuidMap = make(map[string]string)
	}
	return &FilterHCLGenerator{
		uuidMap: uuidMap,
	}
}

// Generate generates HCL for a filter using attribute syntax.
// Returns empty string if filter is nil or empty.
// The indent parameter specifies the base indentation level (typically 1 for a resource attribute).
func (g *FilterHCLGenerator) Generate(filter map[string]interface{}, indent int) string {
	if filter == nil || len(filter) == 0 {
		return ""
	}

	var sb strings.Builder
	indentStr := strings.Repeat("  ", indent)

	sb.WriteString(fmt.Sprintf("%sfilter = {\n", indentStr))
	sb.WriteString(g.generateContent(filter, indent+1))
	sb.WriteString(fmt.Sprintf("%s}\n", indentStr))

	return sb.String()
}

// GenerateInline generates the filter content without the "filter = {" wrapper.
// Useful when the filter needs to be embedded in a different context.
func (g *FilterHCLGenerator) GenerateInline(filter map[string]interface{}, indent int) string {
	if filter == nil || len(filter) == 0 {
		return ""
	}
	return g.generateContent(filter, indent)
}

// generateContent generates the inner content of a filter object
func (g *FilterHCLGenerator) generateContent(filter map[string]interface{}, indent int) string {
	if filter == nil {
		return ""
	}

	var sb strings.Builder
	indentStr := strings.Repeat("  ", indent)

	// Get and normalize filter type (API returns UPPER_SNAKE_CASE, we output lowercase)
	filterType := g.getFilterType(filter)
	if filterType != "" {
		sb.WriteString(fmt.Sprintf("%stype = %q\n", indentStr, filterType))
	}

	// Handle 'not' attribute
	if not, ok := filter["not"].(bool); ok && not {
		sb.WriteString(fmt.Sprintf("%snot = true\n", indentStr))
	}

	// Handle field_id (API uses 'fieldId' or 'field')
	if fieldID := g.getStringField(filter, "fieldId", "field_id", "field"); fieldID != "" {
		sb.WriteString(fmt.Sprintf("%sfield_id = %s\n", indentStr, g.resolveRef(fieldID)))
	}

	// Handle left_value (API uses 'leftValue')
	if leftValue := g.getMapField(filter, "leftValue", "left_value"); leftValue != nil {
		content := g.generateLeftValueContent(leftValue, indent+1)
		if content != "" {
			sb.WriteString(fmt.Sprintf("%sleft_value = {\n", indentStr))
			sb.WriteString(content)
			sb.WriteString(fmt.Sprintf("%s}\n", indentStr))
		}
	}

	// Handle value (single value for comparison)
	if value := g.getMapField(filter, "value"); value != nil {
		content := g.generateValueContent(value, indent+1)
		if content != "" {
			sb.WriteString(fmt.Sprintf("%svalue = {\n", indentStr))
			sb.WriteString(content)
			sb.WriteString(fmt.Sprintf("%s}\n", indentStr))
		}
	}

	// Handle values (array for 'in' operator)
	if values := g.getArrayField(filter, "values"); values != nil {
		content := g.generateValuesArray(values, indent+1)
		if content != "" {
			sb.WriteString(fmt.Sprintf("%svalues = [\n", indentStr))
			sb.WriteString(content)
			sb.WriteString(fmt.Sprintf("%s]\n", indentStr))
		}
	}

	// Handle relative_value (API uses 'relativeValue')
	if relValue := g.getMapField(filter, "relativeValue", "relative_value"); relValue != nil {
		content := g.generateRelativeValueContent(relValue, indent+1)
		if content != "" {
			sb.WriteString(fmt.Sprintf("%srelative_value = {\n", indentStr))
			sb.WriteString(content)
			sb.WriteString(fmt.Sprintf("%s}\n", indentStr))
		}
	}

	// Handle children (for AND/OR filters)
	if children := g.getArrayField(filter, "children"); children != nil {
		content := g.generateChildrenArray(children, indent+1)
		if content != "" {
			sb.WriteString(fmt.Sprintf("%schildren = [\n", indentStr))
			sb.WriteString(content)
			sb.WriteString(fmt.Sprintf("%s]\n", indentStr))
		}
	}

	// Handle mode (for LIKE filters)
	if mode := g.getStringField(filter, "mode"); mode != "" {
		sb.WriteString(fmt.Sprintf("%smode = %q\n", indentStr, strings.ToLower(mode)))
	}

	// Handle value_reference at the filter level (shorthand)
	if vr := g.getStringField(filter, "valueReference", "value_reference"); vr != "" {
		sb.WriteString(fmt.Sprintf("%svalue_reference = %q\n", indentStr, vr))
	}

	return sb.String()
}

// generateLeftValueContent generates content for left_value object
func (g *FilterHCLGenerator) generateLeftValueContent(value map[string]interface{}, indent int) string {
	var sb strings.Builder
	indentStr := strings.Repeat("  ", indent)

	// Check for API format with "type" field first
	valueType := g.getStringField(value, "type")

	switch strings.ToUpper(valueType) {
	case "FIELD":
		// API format: {"type": "FIELD", "field": "field-uuid"}
		if field := g.getStringField(value, "field"); field != "" {
			sb.WriteString(fmt.Sprintf("%sfield_id = %s\n", indentStr, g.resolveRef(field)))
		}
		return sb.String()

	case "REFERENCE":
		// API format: {"type": "REFERENCE", "realType": "...", "value": {"type": "TRIGGER", "name": "..."}}
		if nestedValue := g.getMapField(value, "value"); nestedValue != nil {
			refType := g.getStringField(nestedValue, "type")
			switch refType {
			case "TRIGGER":
				if name := g.getStringField(nestedValue, "name"); name != "" {
					sb.WriteString(fmt.Sprintf("%svalue_reference = %q\n", indentStr, "trigger."+name))
				}
			case "TASK":
				taskID := g.getStringField(nestedValue, "taskId")
				name := g.getStringField(nestedValue, "name")
				if taskID != "" && name != "" {
					sb.WriteString(fmt.Sprintf("%svalue_reference = %q\n", indentStr, "task."+taskID+"."+name))
				}
			case "VARIABLE":
				if name := g.getStringField(nestedValue, "name"); name != "" {
					sb.WriteString(fmt.Sprintf("%svalue_reference = %q\n", indentStr, "variable."+name))
				}
			case "FOR_EACH":
				forEachTaskID := g.getStringField(nestedValue, "forEachTaskId")
				name := g.getStringField(nestedValue, "name")
				if name != "" {
					if forEachTaskID != "" {
						sb.WriteString(fmt.Sprintf("%svalue_reference = %q\n", indentStr, "for_each."+forEachTaskID+"."+name))
					} else {
						sb.WriteString(fmt.Sprintf("%svalue_reference = %q\n", indentStr, "for_each."+name))
					}
				}
			}
		}
		// Handle real_type for REFERENCE type
		if rt := g.getStringField(value, "realType", "real_type"); rt != "" {
			sb.WriteString(fmt.Sprintf("%sreal_type = %q\n", indentStr, strings.ToLower(rt)))
		}
		return sb.String()
	}

	// Legacy format handling (for backwards compatibility with existing tests/configs)

	// Handle field_id (legacy format: {"fieldId": "..."})
	if fieldID := g.getStringField(value, "fieldId", "field_id"); fieldID != "" {
		sb.WriteString(fmt.Sprintf("%sfield_id = %s\n", indentStr, g.resolveRef(fieldID)))
	}

	// Handle value_reference (legacy format: {"valueReference": "..."})
	if vr := g.getStringField(value, "valueReference", "value_reference"); vr != "" {
		sb.WriteString(fmt.Sprintf("%svalue_reference = %q\n", indentStr, vr))
	}

	// Handle triggerReference -> value_reference (legacy nested format)
	if triggerRef := g.getMapField(value, "triggerReference"); triggerRef != nil {
		if name := g.getStringField(triggerRef, "name"); name != "" {
			sb.WriteString(fmt.Sprintf("%svalue_reference = %q\n", indentStr, "trigger."+name))
		}
	}

	// Handle taskReference -> value_reference (legacy nested format)
	if taskRef := g.getMapField(value, "taskReference"); taskRef != nil {
		vr := g.buildTaskReference(taskRef)
		if vr != "" {
			sb.WriteString(fmt.Sprintf("%svalue_reference = %q\n", indentStr, vr))
		}
	}

	// Handle real_type (for legacy format)
	if rt := g.getStringField(value, "realType", "real_type"); rt != "" {
		sb.WriteString(fmt.Sprintf("%sreal_type = %q\n", indentStr, strings.ToLower(rt)))
	}

	return sb.String()
}

// generateValueContent generates content for a value object
func (g *FilterHCLGenerator) generateValueContent(value map[string]interface{}, indent int) string {
	var sb strings.Builder
	indentStr := strings.Repeat("  ", indent)

	// Handle literal value first (common case)
	if literal, ok := value["literal"]; ok {
		return g.generateLiteralContent(literal, indent)
	}

	// Check for API format with "type" field
	valueType := g.getStringField(value, "type")

	// Handle REFERENCE type with nested value structure (API format)
	if strings.ToUpper(valueType) == "REFERENCE" {
		sb.WriteString(fmt.Sprintf("%stype = \"reference\"\n", indentStr))

		// Handle nested value containing the reference
		if nestedValue := g.getMapField(value, "value"); nestedValue != nil {
			refType := g.getStringField(nestedValue, "type")
			switch refType {
			case "TRIGGER":
				if name := g.getStringField(nestedValue, "name"); name != "" {
					sb.WriteString(fmt.Sprintf("%svalue_reference = %q\n", indentStr, "trigger."+name))
				}
			case "TASK":
				taskID := g.getStringField(nestedValue, "taskId")
				name := g.getStringField(nestedValue, "name")
				if taskID != "" && name != "" {
					sb.WriteString(fmt.Sprintf("%svalue_reference = %q\n", indentStr, "task."+taskID+"."+name))
				}
			case "VARIABLE":
				if name := g.getStringField(nestedValue, "name"); name != "" {
					sb.WriteString(fmt.Sprintf("%svalue_reference = %q\n", indentStr, "variable."+name))
				}
			case "FOR_EACH":
				forEachTaskID := g.getStringField(nestedValue, "forEachTaskId")
				name := g.getStringField(nestedValue, "name")
				if name != "" {
					if forEachTaskID != "" {
						sb.WriteString(fmt.Sprintf("%svalue_reference = %q\n", indentStr, "for_each."+forEachTaskID+"."+name))
					} else {
						sb.WriteString(fmt.Sprintf("%svalue_reference = %q\n", indentStr, "for_each."+name))
					}
				}
			}
		}

		// Handle real_type for REFERENCE type
		if rt := g.getStringField(value, "realType", "real_type"); rt != "" {
			sb.WriteString(fmt.Sprintf("%sreal_type = %q\n", indentStr, strings.ToLower(rt)))
		}
		return sb.String()
	}

	// Handle FIELD type with field key (API format)
	if strings.ToUpper(valueType) == "FIELD" {
		sb.WriteString(fmt.Sprintf("%stype = \"field\"\n", indentStr))
		if field := g.getStringField(value, "field"); field != "" {
			sb.WriteString(fmt.Sprintf("%sfield_id = %s\n", indentStr, g.resolveRef(field)))
		}
		return sb.String()
	}

	// Standard handling for other types

	// Handle type
	if valueType != "" {
		sb.WriteString(fmt.Sprintf("%stype = %q\n", indentStr, strings.ToLower(valueType)))
	}

	// Handle value (primitive types only - nested maps are handled above)
	if v := value["value"]; v != nil {
		switch val := v.(type) {
		case string:
			sb.WriteString(fmt.Sprintf("%svalue = %q\n", indentStr, val))
		case float64:
			sb.WriteString(fmt.Sprintf("%svalue = %q\n", indentStr, fmt.Sprintf("%v", val)))
		case bool:
			sb.WriteString(fmt.Sprintf("%svalue = %q\n", indentStr, fmt.Sprintf("%v", val)))
		}
	}

	// Handle value_reference (legacy format)
	if vr := g.getStringField(value, "valueReference", "value_reference"); vr != "" {
		sb.WriteString(fmt.Sprintf("%svalue_reference = %q\n", indentStr, vr))
	}

	// Handle triggerReference -> value_reference (legacy nested format)
	if triggerRef := g.getMapField(value, "triggerReference"); triggerRef != nil {
		if name := g.getStringField(triggerRef, "name"); name != "" {
			sb.WriteString(fmt.Sprintf("%svalue_reference = %q\n", indentStr, "trigger."+name))
		}
	}

	// Handle taskReference -> value_reference (legacy nested format)
	if taskRef := g.getMapField(value, "taskReference"); taskRef != nil {
		vr := g.buildTaskReference(taskRef)
		if vr != "" {
			sb.WriteString(fmt.Sprintf("%svalue_reference = %q\n", indentStr, vr))
		}
	}

	// Handle field_id (for 'field' type - legacy format)
	if fieldID := g.getStringField(value, "fieldId", "field_id"); fieldID != "" {
		sb.WriteString(fmt.Sprintf("%sfield_id = %s\n", indentStr, g.resolveRef(fieldID)))
	}

	// Handle current_user_type
	if cut := g.getStringField(value, "currentUserType", "current_user_type"); cut != "" {
		sb.WriteString(fmt.Sprintf("%scurrent_user_type = %q\n", indentStr, strings.ToLower(cut)))
	}

	// Handle real_type
	if rt := g.getStringField(value, "realType", "real_type"); rt != "" {
		sb.WriteString(fmt.Sprintf("%sreal_type = %q\n", indentStr, strings.ToLower(rt)))
	}

	// Handle record_field specific attributes
	if field := g.getStringField(value, "field"); field != "" {
		sb.WriteString(fmt.Sprintf("%sfield = %s\n", indentStr, g.resolveRef(field)))
	}
	if object := g.getStringField(value, "object"); object != "" {
		sb.WriteString(fmt.Sprintf("%sobject = %s\n", indentStr, g.resolveRef(object)))
	}
	if dynamicField := g.getStringField(value, "dynamicField", "dynamic_field"); dynamicField != "" {
		sb.WriteString(fmt.Sprintf("%sdynamic_field = %s\n", indentStr, g.resolveRef(dynamicField)))
	}
	if dynamicFieldObject := g.getStringField(value, "dynamicFieldObject", "dynamic_field_object"); dynamicFieldObject != "" {
		sb.WriteString(fmt.Sprintf("%sdynamic_field_object = %s\n", indentStr, g.resolveRef(dynamicFieldObject)))
	}
	if sourceFieldType := g.getStringField(value, "sourceFieldType", "source_field_type"); sourceFieldType != "" {
		sb.WriteString(fmt.Sprintf("%ssource_field_type = %q\n", indentStr, strings.ToLower(sourceFieldType)))
	}
	if required, ok := value["required"].(bool); ok {
		sb.WriteString(fmt.Sprintf("%srequired = %v\n", indentStr, required))
	}

	return sb.String()
}

// generateLiteralContent generates HCL for a literal value
func (g *FilterHCLGenerator) generateLiteralContent(literal interface{}, indent int) string {
	indentStr := strings.Repeat("  ", indent)

	switch v := literal.(type) {
	case string:
		return fmt.Sprintf("%stype = \"text\"\n%svalue = %q\n", indentStr, indentStr, v)
	case float64:
		return fmt.Sprintf("%stype = \"number\"\n%svalue = %q\n", indentStr, indentStr, fmt.Sprintf("%v", v))
	case bool:
		return fmt.Sprintf("%stype = \"boolean\"\n%svalue = %q\n", indentStr, indentStr, fmt.Sprintf("%v", v))
	case []interface{}:
		jsonBytes, _ := json.Marshal(v)
		return fmt.Sprintf("%stype = \"text\"\n%svalue = %s\n", indentStr, indentStr, string(jsonBytes))
	default:
		jsonBytes, _ := json.Marshal(v)
		return fmt.Sprintf("%stype = \"text\"\n%svalue = %s\n", indentStr, indentStr, string(jsonBytes))
	}
}

// generateValuesArray generates content for a values array (used by 'in' operator)
func (g *FilterHCLGenerator) generateValuesArray(values []interface{}, indent int) string {
	if len(values) == 0 {
		return ""
	}

	var sb strings.Builder
	indentStr := strings.Repeat("  ", indent)

	for _, v := range values {
		if vMap, ok := v.(map[string]interface{}); ok {
			content := g.generateValueContent(vMap, indent+1)
			if content != "" {
				sb.WriteString(fmt.Sprintf("%s{\n", indentStr))
				sb.WriteString(content)
				sb.WriteString(fmt.Sprintf("%s},\n", indentStr))
			}
		}
	}

	return sb.String()
}

// generateChildrenArray generates content for a children array (used by AND/OR)
func (g *FilterHCLGenerator) generateChildrenArray(children []interface{}, indent int) string {
	if len(children) == 0 {
		return ""
	}

	var sb strings.Builder
	indentStr := strings.Repeat("  ", indent)

	for _, child := range children {
		if childMap, ok := child.(map[string]interface{}); ok {
			content := g.generateContent(childMap, indent+1)
			if content != "" {
				sb.WriteString(fmt.Sprintf("%s{\n", indentStr))
				sb.WriteString(content)
				sb.WriteString(fmt.Sprintf("%s},\n", indentStr))
			}
		}
	}

	return sb.String()
}

// generateRelativeValueContent generates content for relative_value objects
func (g *FilterHCLGenerator) generateRelativeValueContent(relValue map[string]interface{}, indent int) string {
	var sb strings.Builder
	indentStr := strings.Repeat("  ", indent)

	if operator := g.getStringField(relValue, "operator"); operator != "" {
		sb.WriteString(fmt.Sprintf("%soperator = %q\n", indentStr, strings.ToLower(operator)))
	}
	if unit := g.getStringField(relValue, "unit"); unit != "" {
		sb.WriteString(fmt.Sprintf("%sunit = %q\n", indentStr, strings.ToLower(unit)))
	}
	if amount, ok := relValue["amount"].(float64); ok {
		sb.WriteString(fmt.Sprintf("%samount = %d\n", indentStr, int(amount)))
	}

	return sb.String()
}

// buildTaskReference builds a task reference string from a taskReference object
func (g *FilterHCLGenerator) buildTaskReference(taskRef map[string]interface{}) string {
	taskID := ""
	name := ""

	if task := g.getMapField(taskRef, "task"); task != nil {
		taskID = g.getStringField(task, "id")
	}
	name = g.getStringField(taskRef, "name")

	if taskID != "" && name != "" {
		return fmt.Sprintf("task.%s.%s", taskID, name)
	} else if name != "" {
		// Just the name if we don't have task ID
		return "task." + name
	}
	return ""
}

// getFilterType extracts and normalizes the filter type
func (g *FilterHCLGenerator) getFilterType(filter map[string]interface{}) string {
	if t, ok := filter["type"].(string); ok && t != "" {
		return strings.ToLower(t)
	}
	return ""
}

// getStringField extracts a string field from a map, trying multiple possible keys
func (g *FilterHCLGenerator) getStringField(m map[string]interface{}, keys ...string) string {
	for _, key := range keys {
		if v, ok := m[key].(string); ok && v != "" {
			return v
		}
	}
	return ""
}

// getMapField extracts a map field from a map, trying multiple possible keys
func (g *FilterHCLGenerator) getMapField(m map[string]interface{}, keys ...string) map[string]interface{} {
	for _, key := range keys {
		if v, ok := m[key].(map[string]interface{}); ok {
			return v
		}
	}
	return nil
}

// getArrayField extracts an array field from a map, trying multiple possible keys
func (g *FilterHCLGenerator) getArrayField(m map[string]interface{}, keys ...string) []interface{} {
	for _, key := range keys {
		if v, ok := m[key].([]interface{}); ok {
			return v
		}
	}
	return nil
}

// resolveRef resolves a UUID to a terraform reference if possible
func (g *FilterHCLGenerator) resolveRef(id string) string {
	if ref, ok := g.uuidMap[id]; ok {
		return ref
	}
	return fmt.Sprintf("%q", id)
}

// GenerateFilterHCL is a convenience function to generate filter HCL with a uuid map.
// This is the main entry point for filter HCL generation.
func GenerateFilterHCL(filter map[string]interface{}, uuidMap map[string]string, indent int) string {
	gen := NewFilterHCLGenerator(uuidMap)
	return gen.Generate(filter, indent)
}

// ============================================================================
// IR Generation Methods
// ============================================================================

// GenerateIR generates an HCLObject representing the filter.
// Returns nil if the filter is nil or empty.
func (g *FilterHCLGenerator) GenerateIR(filter map[string]interface{}) HCLValue {
	if filter == nil || len(filter) == 0 {
		return nil
	}
	return g.generateContentIR(filter)
}

// generateContentIR converts a filter map into an HCLObject with properly typed values.
func (g *FilterHCLGenerator) generateContentIR(filter map[string]interface{}) HCLObject {
	var attrs []*HCLAttribute

	// type
	filterType := g.getFilterType(filter)
	if filterType != "" {
		attrs = append(attrs, Attr("type", Str(filterType)))
	}

	// not
	if not, ok := filter["not"].(bool); ok && not {
		attrs = append(attrs, Attr("not", Bool(true)))
	}

	// field_id
	if fieldID := g.getStringField(filter, "fieldId", "field_id", "field"); fieldID != "" {
		attrs = append(attrs, Attr("field_id", g.resolveRefIR(fieldID)))
	}

	// left_value
	if leftValue := g.getMapField(filter, "leftValue", "left_value"); leftValue != nil {
		obj := g.generateLeftValueIR(leftValue)
		if len(obj.Attributes) > 0 {
			attrs = append(attrs, Attr("left_value", obj))
		}
	}

	// value
	if value := g.getMapField(filter, "value"); value != nil {
		obj := g.generateValueContentIR(value)
		if len(obj.Attributes) > 0 {
			attrs = append(attrs, Attr("value", obj))
		}
	}

	// values (array for 'in' operator)
	if values := g.getArrayField(filter, "values"); values != nil {
		list := g.generateValuesArrayIR(values)
		if len(list.Values) > 0 {
			attrs = append(attrs, Attr("values", list))
		}
	}

	// relative_value
	if relValue := g.getMapField(filter, "relativeValue", "relative_value"); relValue != nil {
		obj := g.generateRelativeValueIR(relValue)
		if len(obj.Attributes) > 0 {
			attrs = append(attrs, Attr("relative_value", obj))
		}
	}

	// children (recursive)
	if children := g.getArrayField(filter, "children"); children != nil {
		list := g.generateChildrenArrayIR(children)
		if len(list.Values) > 0 {
			attrs = append(attrs, Attr("children", list))
		}
	}

	// mode
	if mode := g.getStringField(filter, "mode"); mode != "" {
		attrs = append(attrs, Attr("mode", Str(strings.ToLower(mode))))
	}

	// value_reference (shorthand)
	if vr := g.getStringField(filter, "valueReference", "value_reference"); vr != "" {
		attrs = append(attrs, Attr("value_reference", Str(vr)))
	}

	return HCLObject{Attributes: attrs}
}

// generateLeftValueIR produces an HCLObject for left_value
func (g *FilterHCLGenerator) generateLeftValueIR(value map[string]interface{}) HCLObject {
	var attrs []*HCLAttribute

	valueType := g.getStringField(value, "type")

	switch strings.ToUpper(valueType) {
	case "FIELD":
		if field := g.getStringField(value, "field"); field != "" {
			attrs = append(attrs, Attr("field_id", g.resolveRefIR(field)))
		}
		return HCLObject{Attributes: attrs}

	case "REFERENCE":
		attrs = append(attrs, g.referenceValueAttrs(value)...)
		if rt := g.getStringField(value, "realType", "real_type"); rt != "" {
			attrs = append(attrs, Attr("real_type", Str(strings.ToLower(rt))))
		}
		return HCLObject{Attributes: attrs}
	}

	// Legacy format handling
	if fieldID := g.getStringField(value, "fieldId", "field_id"); fieldID != "" {
		attrs = append(attrs, Attr("field_id", g.resolveRefIR(fieldID)))
	}
	if vr := g.getStringField(value, "valueReference", "value_reference"); vr != "" {
		attrs = append(attrs, Attr("value_reference", Str(vr)))
	}
	if triggerRef := g.getMapField(value, "triggerReference"); triggerRef != nil {
		if name := g.getStringField(triggerRef, "name"); name != "" {
			attrs = append(attrs, Attr("value_reference", Str("trigger."+name)))
		}
	}
	if taskRef := g.getMapField(value, "taskReference"); taskRef != nil {
		if vr := g.buildTaskReference(taskRef); vr != "" {
			attrs = append(attrs, Attr("value_reference", Str(vr)))
		}
	}
	if rt := g.getStringField(value, "realType", "real_type"); rt != "" {
		attrs = append(attrs, Attr("real_type", Str(strings.ToLower(rt))))
	}

	return HCLObject{Attributes: attrs}
}

// generateValueContentIR produces an HCLObject for a value
func (g *FilterHCLGenerator) generateValueContentIR(value map[string]interface{}) HCLObject {
	var attrs []*HCLAttribute

	// Handle literal value
	if literal, ok := value["literal"]; ok {
		return g.generateLiteralIR(literal)
	}

	valueType := g.getStringField(value, "type")

	switch strings.ToUpper(valueType) {
	case "REFERENCE":
		attrs = append(attrs, Attr("type", Str("reference")))
		attrs = append(attrs, g.referenceValueAttrs(value)...)
		if rt := g.getStringField(value, "realType", "real_type"); rt != "" {
			attrs = append(attrs, Attr("real_type", Str(strings.ToLower(rt))))
		}
		return HCLObject{Attributes: attrs}

	case "FIELD":
		attrs = append(attrs, Attr("type", Str("field")))
		if field := g.getStringField(value, "field"); field != "" {
			attrs = append(attrs, Attr("field_id", g.resolveRefIR(field)))
		}
		return HCLObject{Attributes: attrs}
	}

	// Standard handling
	if valueType != "" {
		attrs = append(attrs, Attr("type", Str(strings.ToLower(valueType))))
	}

	if v := value["value"]; v != nil {
		switch val := v.(type) {
		case string:
			attrs = append(attrs, Attr("value", Str(val)))
		case float64:
			attrs = append(attrs, Attr("value", Str(fmt.Sprintf("%v", val))))
		case bool:
			attrs = append(attrs, Attr("value", Str(fmt.Sprintf("%v", val))))
		}
	}

	if vr := g.getStringField(value, "valueReference", "value_reference"); vr != "" {
		attrs = append(attrs, Attr("value_reference", Str(vr)))
	}
	if triggerRef := g.getMapField(value, "triggerReference"); triggerRef != nil {
		if name := g.getStringField(triggerRef, "name"); name != "" {
			attrs = append(attrs, Attr("value_reference", Str("trigger."+name)))
		}
	}
	if taskRef := g.getMapField(value, "taskReference"); taskRef != nil {
		if vr := g.buildTaskReference(taskRef); vr != "" {
			attrs = append(attrs, Attr("value_reference", Str(vr)))
		}
	}
	if fieldID := g.getStringField(value, "fieldId", "field_id"); fieldID != "" {
		attrs = append(attrs, Attr("field_id", g.resolveRefIR(fieldID)))
	}
	if cut := g.getStringField(value, "currentUserType", "current_user_type"); cut != "" {
		attrs = append(attrs, Attr("current_user_type", Str(strings.ToLower(cut))))
	}
	if rt := g.getStringField(value, "realType", "real_type"); rt != "" {
		attrs = append(attrs, Attr("real_type", Str(strings.ToLower(rt))))
	}
	if field := g.getStringField(value, "field"); field != "" {
		attrs = append(attrs, Attr("field", g.resolveRefIR(field)))
	}
	if object := g.getStringField(value, "object"); object != "" {
		attrs = append(attrs, Attr("object", g.resolveRefIR(object)))
	}
	if dynamicField := g.getStringField(value, "dynamicField", "dynamic_field"); dynamicField != "" {
		attrs = append(attrs, Attr("dynamic_field", g.resolveRefIR(dynamicField)))
	}
	if dynamicFieldObject := g.getStringField(value, "dynamicFieldObject", "dynamic_field_object"); dynamicFieldObject != "" {
		attrs = append(attrs, Attr("dynamic_field_object", g.resolveRefIR(dynamicFieldObject)))
	}
	if sourceFieldType := g.getStringField(value, "sourceFieldType", "source_field_type"); sourceFieldType != "" {
		attrs = append(attrs, Attr("source_field_type", Str(strings.ToLower(sourceFieldType))))
	}
	if required, ok := value["required"].(bool); ok {
		attrs = append(attrs, Attr("required", Bool(required)))
	}

	return HCLObject{Attributes: attrs}
}

// generateLiteralIR converts a literal value to an HCLObject
func (g *FilterHCLGenerator) generateLiteralIR(literal interface{}) HCLObject {
	switch v := literal.(type) {
	case string:
		return Obj(Attr("type", Str("text")), Attr("value", Str(v)))
	case float64:
		return Obj(Attr("type", Str("number")), Attr("value", Str(fmt.Sprintf("%v", v))))
	case bool:
		return Obj(Attr("type", Str("boolean")), Attr("value", Str(fmt.Sprintf("%v", v))))
	case []interface{}:
		jsonBytes, _ := json.Marshal(v)
		return Obj(Attr("type", Str("text")), Attr("value", Raw(string(jsonBytes))))
	default:
		jsonBytes, _ := json.Marshal(v)
		return Obj(Attr("type", Str("text")), Attr("value", Raw(string(jsonBytes))))
	}
}

// generateValuesArrayIR generates an HCLList for filter values
func (g *FilterHCLGenerator) generateValuesArrayIR(values []interface{}) HCLList {
	var items []HCLValue
	for _, v := range values {
		if vMap, ok := v.(map[string]interface{}); ok {
			obj := g.generateValueContentIR(vMap)
			if len(obj.Attributes) > 0 {
				items = append(items, obj)
			}
		}
	}
	return HCLList{Values: items}
}

// generateChildrenArrayIR generates an HCLList for filter children (recursive)
func (g *FilterHCLGenerator) generateChildrenArrayIR(children []interface{}) HCLList {
	var items []HCLValue
	for _, child := range children {
		if childMap, ok := child.(map[string]interface{}); ok {
			obj := g.generateContentIR(childMap)
			if len(obj.Attributes) > 0 {
				items = append(items, obj)
			}
		}
	}
	return HCLList{Values: items}
}

// generateRelativeValueIR generates an HCLObject for relative_value
func (g *FilterHCLGenerator) generateRelativeValueIR(relValue map[string]interface{}) HCLObject {
	var attrs []*HCLAttribute

	if operator := g.getStringField(relValue, "operator"); operator != "" {
		attrs = append(attrs, Attr("operator", Str(strings.ToLower(operator))))
	}
	if unit := g.getStringField(relValue, "unit"); unit != "" {
		attrs = append(attrs, Attr("unit", Str(strings.ToLower(unit))))
	}
	if amount, ok := relValue["amount"].(float64); ok {
		attrs = append(attrs, Attr("amount", Num(amount)))
	}

	return HCLObject{Attributes: attrs}
}

// referenceValueAttrs extracts value_reference from a nested reference value
func (g *FilterHCLGenerator) referenceValueAttrs(value map[string]interface{}) []*HCLAttribute {
	var attrs []*HCLAttribute
	if nestedValue := g.getMapField(value, "value"); nestedValue != nil {
		refType := g.getStringField(nestedValue, "type")
		switch refType {
		case "TRIGGER":
			if name := g.getStringField(nestedValue, "name"); name != "" {
				attrs = append(attrs, Attr("value_reference", Str("trigger."+name)))
			}
		case "TASK":
			taskID := g.getStringField(nestedValue, "taskId")
			name := g.getStringField(nestedValue, "name")
			if taskID != "" && name != "" {
				attrs = append(attrs, Attr("value_reference", Str("task."+taskID+"."+name)))
			}
		case "VARIABLE":
			if name := g.getStringField(nestedValue, "name"); name != "" {
				attrs = append(attrs, Attr("value_reference", Str("variable."+name)))
			}
		case "FOR_EACH":
			forEachTaskID := g.getStringField(nestedValue, "forEachTaskId")
			name := g.getStringField(nestedValue, "name")
			if name != "" {
				if forEachTaskID != "" {
					attrs = append(attrs, Attr("value_reference", Str("for_each."+forEachTaskID+"."+name)))
				} else {
					attrs = append(attrs, Attr("value_reference", Str("for_each."+name)))
				}
			}
		}
	}
	return attrs
}

// resolveRefIR resolves a UUID to the appropriate HCLValue (reference or string)
func (g *FilterHCLGenerator) resolveRefIR(id string) HCLValue {
	if ref, ok := g.uuidMap[id]; ok {
		return Ref(ref)
	}
	return Str(id)
}

// GenerateFilterIR is a convenience function to generate filter IR with a uuid map.
func GenerateFilterIR(filter map[string]interface{}, uuidMap map[string]string) HCLValue {
	gen := NewFilterHCLGenerator(uuidMap)
	return gen.GenerateIR(filter)
}
