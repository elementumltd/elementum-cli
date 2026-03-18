// Copyright (c) ElementumAI, Inc.
// SPDX-License-Identifier: MPL-2.0

package export

import (
	"strings"
	"testing"
)

func TestFilterHCLGenerator_NilFilter(t *testing.T) {
	gen := NewFilterHCLGenerator(nil)
	result := gen.Generate(nil, 1)
	if result != "" {
		t.Errorf("Expected empty string for nil filter, got: %q", result)
	}
}

func TestFilterHCLGenerator_EmptyFilter(t *testing.T) {
	gen := NewFilterHCLGenerator(nil)
	result := gen.Generate(map[string]interface{}{}, 1)
	if result != "" {
		t.Errorf("Expected empty string for empty filter, got: %q", result)
	}
}

func TestFilterHCLGenerator_SimpleEquals(t *testing.T) {
	gen := NewFilterHCLGenerator(nil)
	filter := map[string]interface{}{
		"type":    "EQUALS",
		"fieldId": "field-123",
		"value": map[string]interface{}{
			"type":  "text",
			"value": "active",
		},
	}

	result := gen.Generate(filter, 1)

	// Check attribute syntax
	if !strings.Contains(result, "filter = {") {
		t.Errorf("Expected 'filter = {' syntax, got:\n%s", result)
	}

	// Check type is lowercased
	if !strings.Contains(result, `type = "equals"`) {
		t.Errorf("Expected type = \"equals\", got:\n%s", result)
	}

	// Check field_id
	if !strings.Contains(result, `field_id = "field-123"`) {
		t.Errorf("Expected field_id, got:\n%s", result)
	}

	// Check value block
	if !strings.Contains(result, "value = {") {
		t.Errorf("Expected value = {, got:\n%s", result)
	}
}

func TestFilterHCLGenerator_UUIDBeautification(t *testing.T) {
	uuidMap := map[string]string{
		"field-123": "elementum_field.status.id",
	}
	gen := NewFilterHCLGenerator(uuidMap)

	filter := map[string]interface{}{
		"type":    "EQUALS",
		"fieldId": "field-123",
		"value": map[string]interface{}{
			"literal": "active",
		},
	}

	result := gen.Generate(filter, 1)

	// Check field_id is beautified (no quotes around reference)
	if !strings.Contains(result, "field_id = elementum_field.status.id") {
		t.Errorf("Expected beautified field_id, got:\n%s", result)
	}
}

func TestFilterHCLGenerator_LiteralValue(t *testing.T) {
	gen := NewFilterHCLGenerator(nil)

	testCases := []struct {
		name     string
		literal  interface{}
		expected string
	}{
		{
			name:     "string literal",
			literal:  "hello",
			expected: `type = "text"`,
		},
		{
			name:     "number literal",
			literal:  float64(42),
			expected: `type = "number"`,
		},
		{
			name:     "boolean literal",
			literal:  true,
			expected: `type = "boolean"`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			filter := map[string]interface{}{
				"type":    "EQUALS",
				"fieldId": "field-1",
				"value": map[string]interface{}{
					"literal": tc.literal,
				},
			}

			result := gen.Generate(filter, 1)
			if !strings.Contains(result, tc.expected) {
				t.Errorf("Expected %q in result, got:\n%s", tc.expected, result)
			}
		})
	}
}

func TestFilterHCLGenerator_AndFilter(t *testing.T) {
	gen := NewFilterHCLGenerator(nil)
	filter := map[string]interface{}{
		"type": "AND",
		"children": []interface{}{
			map[string]interface{}{
				"type":    "EQUALS",
				"fieldId": "field-1",
				"value": map[string]interface{}{
					"literal": "active",
				},
			},
			map[string]interface{}{
				"type":    "GREATER_THAN",
				"fieldId": "field-2",
				"value": map[string]interface{}{
					"literal": float64(100),
				},
			},
		},
	}

	result := gen.Generate(filter, 1)

	// Check type
	if !strings.Contains(result, `type = "and"`) {
		t.Errorf("Expected type = \"and\", got:\n%s", result)
	}

	// Check children array
	if !strings.Contains(result, "children = [") {
		t.Errorf("Expected children = [, got:\n%s", result)
	}

	// Check child filter types
	if !strings.Contains(result, `type = "equals"`) {
		t.Errorf("Expected equals child, got:\n%s", result)
	}
	if !strings.Contains(result, `type = "greater_than"`) {
		t.Errorf("Expected greater_than child, got:\n%s", result)
	}
}

func TestFilterHCLGenerator_OrFilter(t *testing.T) {
	gen := NewFilterHCLGenerator(nil)
	filter := map[string]interface{}{
		"type": "OR",
		"children": []interface{}{
			map[string]interface{}{
				"type":    "EQUALS",
				"fieldId": "field-1",
				"value": map[string]interface{}{
					"literal": "pending",
				},
			},
			map[string]interface{}{
				"type":    "EQUALS",
				"fieldId": "field-1",
				"value": map[string]interface{}{
					"literal": "active",
				},
			},
		},
	}

	result := gen.Generate(filter, 1)

	if !strings.Contains(result, `type = "or"`) {
		t.Errorf("Expected type = \"or\", got:\n%s", result)
	}
}

func TestFilterHCLGenerator_NestedAndOr(t *testing.T) {
	gen := NewFilterHCLGenerator(nil)
	filter := map[string]interface{}{
		"type": "AND",
		"children": []interface{}{
			map[string]interface{}{
				"type": "OR",
				"children": []interface{}{
					map[string]interface{}{
						"type":    "EQUALS",
						"fieldId": "status",
						"value":   map[string]interface{}{"literal": "open"},
					},
					map[string]interface{}{
						"type":    "EQUALS",
						"fieldId": "status",
						"value":   map[string]interface{}{"literal": "pending"},
					},
				},
			},
			map[string]interface{}{
				"type":    "GREATER_THAN",
				"fieldId": "priority",
				"value":   map[string]interface{}{"literal": float64(50)},
			},
		},
	}

	result := gen.Generate(filter, 1)

	// Check both AND and OR are present
	if !strings.Contains(result, `type = "and"`) {
		t.Errorf("Expected top-level and, got:\n%s", result)
	}
	if !strings.Contains(result, `type = "or"`) {
		t.Errorf("Expected nested or, got:\n%s", result)
	}
}

func TestFilterHCLGenerator_InFilter(t *testing.T) {
	gen := NewFilterHCLGenerator(nil)
	filter := map[string]interface{}{
		"type":    "IN",
		"fieldId": "field-status",
		"values": []interface{}{
			map[string]interface{}{"type": "text", "value": "active"},
			map[string]interface{}{"type": "text", "value": "pending"},
			map[string]interface{}{"type": "text", "value": "review"},
		},
	}

	result := gen.Generate(filter, 1)

	if !strings.Contains(result, `type = "in"`) {
		t.Errorf("Expected type = \"in\", got:\n%s", result)
	}
	if !strings.Contains(result, "values = [") {
		t.Errorf("Expected values = [, got:\n%s", result)
	}
}

func TestFilterHCLGenerator_RelativeValue(t *testing.T) {
	gen := NewFilterHCLGenerator(nil)
	filter := map[string]interface{}{
		"type":    "CURRENT_RANGE",
		"fieldId": "field-due-date",
		"relativeValue": map[string]interface{}{
			"operator": "NEXT",
			"unit":     "DAY",
			"amount":   float64(7),
		},
	}

	result := gen.Generate(filter, 1)

	if !strings.Contains(result, `type = "current_range"`) {
		t.Errorf("Expected type = \"current_range\", got:\n%s", result)
	}
	if !strings.Contains(result, "relative_value = {") {
		t.Errorf("Expected relative_value = {, got:\n%s", result)
	}
	if !strings.Contains(result, `operator = "next"`) {
		t.Errorf("Expected operator = \"next\", got:\n%s", result)
	}
	if !strings.Contains(result, `unit = "day"`) {
		t.Errorf("Expected unit = \"day\", got:\n%s", result)
	}
	if !strings.Contains(result, "amount = 7") {
		t.Errorf("Expected amount = 7, got:\n%s", result)
	}
}

func TestFilterHCLGenerator_TriggerReference(t *testing.T) {
	gen := NewFilterHCLGenerator(nil)
	filter := map[string]interface{}{
		"type":    "EQUALS",
		"fieldId": "field-1",
		"value": map[string]interface{}{
			"triggerReference": map[string]interface{}{
				"name": "record.user_id.id",
			},
		},
	}

	result := gen.Generate(filter, 1)

	if !strings.Contains(result, `value_reference = "trigger.record.user_id.id"`) {
		t.Errorf("Expected trigger value_reference, got:\n%s", result)
	}
}

func TestFilterHCLGenerator_TaskReference(t *testing.T) {
	gen := NewFilterHCLGenerator(nil)
	filter := map[string]interface{}{
		"type":    "EQUALS",
		"fieldId": "field-1",
		"value": map[string]interface{}{
			"taskReference": map[string]interface{}{
				"task": map[string]interface{}{
					"id": "task-123",
				},
				"name": "result.count",
			},
		},
	}

	result := gen.Generate(filter, 1)

	if !strings.Contains(result, `value_reference = "task.task-123.result.count"`) {
		t.Errorf("Expected task value_reference, got:\n%s", result)
	}
}

func TestFilterHCLGenerator_LeftValue(t *testing.T) {
	gen := NewFilterHCLGenerator(nil)
	filter := map[string]interface{}{
		"type": "EQUALS",
		"leftValue": map[string]interface{}{
			"valueReference": "some.reference.path",
			"realType":       "NUMBER",
		},
		"value": map[string]interface{}{
			"literal": float64(100),
		},
	}

	result := gen.Generate(filter, 1)

	if !strings.Contains(result, "left_value = {") {
		t.Errorf("Expected left_value = {, got:\n%s", result)
	}
	if !strings.Contains(result, `value_reference = "some.reference.path"`) {
		t.Errorf("Expected value_reference in left_value, got:\n%s", result)
	}
	if !strings.Contains(result, `real_type = "number"`) {
		t.Errorf("Expected real_type = \"number\", got:\n%s", result)
	}
}

func TestFilterHCLGenerator_NotFilter(t *testing.T) {
	gen := NewFilterHCLGenerator(nil)
	filter := map[string]interface{}{
		"type":    "EQUALS",
		"not":     true,
		"fieldId": "field-1",
		"value": map[string]interface{}{
			"literal": "inactive",
		},
	}

	result := gen.Generate(filter, 1)

	if !strings.Contains(result, "not = true") {
		t.Errorf("Expected not = true, got:\n%s", result)
	}
}

func TestFilterHCLGenerator_LikeMode(t *testing.T) {
	gen := NewFilterHCLGenerator(nil)
	filter := map[string]interface{}{
		"type":    "LIKE",
		"fieldId": "field-title",
		"mode":    "STARTS_WITH",
		"value": map[string]interface{}{
			"literal": "URGENT",
		},
	}

	result := gen.Generate(filter, 1)

	if !strings.Contains(result, `type = "like"`) {
		t.Errorf("Expected type = \"like\", got:\n%s", result)
	}
	if !strings.Contains(result, `mode = "starts_with"`) {
		t.Errorf("Expected mode = \"starts_with\", got:\n%s", result)
	}
}

func TestFilterHCLGenerator_CurrentUserType(t *testing.T) {
	gen := NewFilterHCLGenerator(nil)
	filter := map[string]interface{}{
		"type":    "EQUALS",
		"fieldId": "field-owner",
		"value": map[string]interface{}{
			"type":            "current_user",
			"currentUserType": "ID",
		},
	}

	result := gen.Generate(filter, 1)

	if !strings.Contains(result, `type = "current_user"`) {
		t.Errorf("Expected type = \"current_user\", got:\n%s", result)
	}
	if !strings.Contains(result, `current_user_type = "id"`) {
		t.Errorf("Expected current_user_type = \"id\", got:\n%s", result)
	}
}

func TestFilterHCLGenerator_RecordFieldValue(t *testing.T) {
	gen := NewFilterHCLGenerator(nil)
	filter := map[string]interface{}{
		"type":    "EQUALS",
		"fieldId": "field-1",
		"value": map[string]interface{}{
			"type":            "record_field",
			"field":           "field-abc",
			"object":          "app-123",
			"sourceFieldType": "TEXT",
		},
	}

	result := gen.Generate(filter, 1)

	if !strings.Contains(result, `type = "record_field"`) {
		t.Errorf("Expected type = \"record_field\", got:\n%s", result)
	}
	if !strings.Contains(result, `field = "field-abc"`) {
		t.Errorf("Expected field, got:\n%s", result)
	}
	if !strings.Contains(result, `object = "app-123"`) {
		t.Errorf("Expected object, got:\n%s", result)
	}
	if !strings.Contains(result, `source_field_type = "text"`) {
		t.Errorf("Expected source_field_type, got:\n%s", result)
	}
}

func TestFilterHCLGenerator_Indentation(t *testing.T) {
	gen := NewFilterHCLGenerator(nil)
	filter := map[string]interface{}{
		"type":    "EQUALS",
		"fieldId": "field-1",
		"value": map[string]interface{}{
			"literal": "test",
		},
	}

	// Test indent level 1 (typical for resource attribute)
	result1 := gen.Generate(filter, 1)
	if !strings.HasPrefix(result1, "  filter = {") {
		t.Errorf("Expected 2-space indent at level 1, got:\n%s", result1)
	}

	// Test indent level 2 (for nested context)
	result2 := gen.Generate(filter, 2)
	if !strings.HasPrefix(result2, "    filter = {") {
		t.Errorf("Expected 4-space indent at level 2, got:\n%s", result2)
	}
}

func TestFilterHCLGenerator_GenerateInline(t *testing.T) {
	gen := NewFilterHCLGenerator(nil)
	filter := map[string]interface{}{
		"type":    "EQUALS",
		"fieldId": "field-1",
		"value": map[string]interface{}{
			"literal": "test",
		},
	}

	result := gen.GenerateInline(filter, 1)

	// Should NOT contain "filter = {"
	if strings.Contains(result, "filter = {") {
		t.Errorf("GenerateInline should not contain 'filter = {', got:\n%s", result)
	}

	// Should contain the content
	if !strings.Contains(result, `type = "equals"`) {
		t.Errorf("GenerateInline should contain filter content, got:\n%s", result)
	}
}

func TestGenerateFilterHCL_ConvenienceFunction(t *testing.T) {
	uuidMap := map[string]string{
		"field-1": "elementum_field.status.id",
	}
	filter := map[string]interface{}{
		"type":    "EQUALS",
		"fieldId": "field-1",
		"value": map[string]interface{}{
			"literal": "active",
		},
	}

	result := GenerateFilterHCL(filter, uuidMap, 1)

	// Check attribute syntax
	if !strings.Contains(result, "filter = {") {
		t.Errorf("Expected 'filter = {' syntax, got:\n%s", result)
	}

	// Check beautification worked
	if !strings.Contains(result, "field_id = elementum_field.status.id") {
		t.Errorf("Expected beautified field_id, got:\n%s", result)
	}
}

func TestFilterHCLGenerator_ComplexRealWorldFilter(t *testing.T) {
	// Test a complex filter similar to what we'd see in real exports
	uuidMap := map[string]string{
		"field-status":   "elementum_picklist_field.status.id",
		"field-priority": "elementum_number_field.priority.id",
		"field-owner":    "elementum_user_field.owner.id",
	}
	gen := NewFilterHCLGenerator(uuidMap)

	filter := map[string]interface{}{
		"type": "AND",
		"children": []interface{}{
			map[string]interface{}{
				"type": "OR",
				"children": []interface{}{
					map[string]interface{}{
						"type":    "EQUALS",
						"fieldId": "field-status",
						"value":   map[string]interface{}{"literal": "open"},
					},
					map[string]interface{}{
						"type":    "EQUALS",
						"fieldId": "field-status",
						"value":   map[string]interface{}{"literal": "in_progress"},
					},
				},
			},
			map[string]interface{}{
				"type":    "GREATER_THAN",
				"fieldId": "field-priority",
				"value":   map[string]interface{}{"literal": float64(50)},
			},
			map[string]interface{}{
				"type":    "IS_NULL",
				"not":     true,
				"fieldId": "field-owner",
			},
		},
	}

	result := gen.Generate(filter, 1)

	// Verify structure
	if !strings.Contains(result, `type = "and"`) {
		t.Errorf("Missing top-level AND, got:\n%s", result)
	}
	if !strings.Contains(result, `type = "or"`) {
		t.Errorf("Missing nested OR, got:\n%s", result)
	}
	if !strings.Contains(result, `type = "greater_than"`) {
		t.Errorf("Missing GREATER_THAN, got:\n%s", result)
	}
	if !strings.Contains(result, `type = "is_null"`) {
		t.Errorf("Missing IS_NULL, got:\n%s", result)
	}
	if !strings.Contains(result, "not = true") {
		t.Errorf("Missing not = true, got:\n%s", result)
	}

	// Verify UUID beautification
	if !strings.Contains(result, "field_id = elementum_picklist_field.status.id") {
		t.Errorf("Status field not beautified, got:\n%s", result)
	}
	if !strings.Contains(result, "field_id = elementum_number_field.priority.id") {
		t.Errorf("Priority field not beautified, got:\n%s", result)
	}
	if !strings.Contains(result, "field_id = elementum_user_field.owner.id") {
		t.Errorf("Owner field not beautified, got:\n%s", result)
	}
}

func TestFilterHCLGenerator_IsNullFilter(t *testing.T) {
	gen := NewFilterHCLGenerator(nil)
	filter := map[string]interface{}{
		"type":    "IS_NULL",
		"fieldId": "field-assigned",
	}

	result := gen.Generate(filter, 1)

	if !strings.Contains(result, `type = "is_null"`) {
		t.Errorf("Expected type = \"is_null\", got:\n%s", result)
	}
	if !strings.Contains(result, `field_id = "field-assigned"`) {
		t.Errorf("Expected field_id, got:\n%s", result)
	}
	// Should NOT have a value block
	if strings.Contains(result, "value = {") {
		t.Errorf("IS_NULL should not have value block, got:\n%s", result)
	}
}

func TestFilterHCLGenerator_BetweenFilter(t *testing.T) {
	gen := NewFilterHCLGenerator(nil)
	filter := map[string]interface{}{
		"type":    "BETWEEN",
		"fieldId": "field-score",
		"value": map[string]interface{}{
			"type":  "number",
			"value": "50",
		},
		// Note: Between typically has min/max, but this tests our current handling
	}

	result := gen.Generate(filter, 1)

	if !strings.Contains(result, `type = "between"`) {
		t.Errorf("Expected type = \"between\", got:\n%s", result)
	}
}

func TestFilterHCLGenerator_OverlapsFilter(t *testing.T) {
	gen := NewFilterHCLGenerator(nil)
	filter := map[string]interface{}{
		"type":    "OVERLAPS",
		"fieldId": "field-tags",
		"values": []interface{}{
			map[string]interface{}{"type": "text", "value": "urgent"},
			map[string]interface{}{"type": "text", "value": "important"},
		},
	}

	result := gen.Generate(filter, 1)

	if !strings.Contains(result, `type = "overlaps"`) {
		t.Errorf("Expected type = \"overlaps\", got:\n%s", result)
	}
	if !strings.Contains(result, "values = [") {
		t.Errorf("Expected values = [, got:\n%s", result)
	}
}

func TestFilterHCLGenerator_ContainsFilter(t *testing.T) {
	gen := NewFilterHCLGenerator(nil)
	filter := map[string]interface{}{
		"type":    "CONTAINS",
		"fieldId": "field-description",
		"value": map[string]interface{}{
			"literal": "error",
		},
	}

	result := gen.Generate(filter, 1)

	if !strings.Contains(result, `type = "contains"`) {
		t.Errorf("Expected type = \"contains\", got:\n%s", result)
	}
}

func TestFilterHCLGenerator_SearchFilter(t *testing.T) {
	gen := NewFilterHCLGenerator(nil)
	filter := map[string]interface{}{
		"type": "SEARCH",
		"value": map[string]interface{}{
			"literal": "query text",
		},
	}

	result := gen.Generate(filter, 1)

	if !strings.Contains(result, `type = "search"`) {
		t.Errorf("Expected type = \"search\", got:\n%s", result)
	}
}

func TestFilterHCLGenerator_TrueFilter(t *testing.T) {
	gen := NewFilterHCLGenerator(nil)
	filter := map[string]interface{}{
		"type": "TRUE",
	}

	result := gen.Generate(filter, 1)

	if !strings.Contains(result, `type = "true"`) {
		t.Errorf("Expected type = \"true\", got:\n%s", result)
	}
}

func TestFilterHCLGenerator_FalseFilter(t *testing.T) {
	gen := NewFilterHCLGenerator(nil)
	filter := map[string]interface{}{
		"type": "FALSE",
	}

	result := gen.Generate(filter, 1)

	if !strings.Contains(result, `type = "false"`) {
		t.Errorf("Expected type = \"false\", got:\n%s", result)
	}
}

func TestFilterHCLGenerator_LeftValueWithTriggerReference(t *testing.T) {
	gen := NewFilterHCLGenerator(nil)
	filter := map[string]interface{}{
		"type": "EQUALS",
		"leftValue": map[string]interface{}{
			"triggerReference": map[string]interface{}{
				"name": "record.status.id",
			},
		},
		"value": map[string]interface{}{
			"literal": "completed",
		},
	}

	result := gen.Generate(filter, 1)

	if !strings.Contains(result, "left_value = {") {
		t.Errorf("Expected left_value, got:\n%s", result)
	}
	if !strings.Contains(result, `value_reference = "trigger.record.status.id"`) {
		t.Errorf("Expected trigger reference in left_value, got:\n%s", result)
	}
}

func TestFilterHCLGenerator_ValuesWithLiterals(t *testing.T) {
	gen := NewFilterHCLGenerator(nil)
	filter := map[string]interface{}{
		"type":    "IN",
		"fieldId": "field-status",
		"values": []interface{}{
			map[string]interface{}{"literal": "active"},
			map[string]interface{}{"literal": "pending"},
		},
	}

	result := gen.Generate(filter, 1)

	if !strings.Contains(result, "values = [") {
		t.Errorf("Expected values = [, got:\n%s", result)
	}
	// Literals should be converted to type/value format
	if !strings.Contains(result, `type = "text"`) {
		t.Errorf("Expected text type for literals, got:\n%s", result)
	}
}
