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
	"strings"
	"testing"
)

// Test helpers for creating API-format filter structures

// createAPIFieldLeftValue creates a leftValue in API format with type: "FIELD"
func createAPIFieldLeftValue(fieldID string) map[string]interface{} {
	return map[string]interface{}{
		"type":  "FIELD",
		"field": fieldID,
	}
}

// createAPIReferenceLeftValue creates a leftValue in API format with type: "REFERENCE"
func createAPIReferenceLeftValue(refType, refValue, realType string) map[string]interface{} {
	value := map[string]interface{}{
		"type": refType,
	}

	switch refType {
	case "TRIGGER":
		value["name"] = refValue
	case "TASK":
		parts := strings.SplitN(refValue, ".", 2)
		if len(parts) == 2 {
			value["taskId"] = parts[0]
			value["name"] = parts[1]
		}
	case "VARIABLE":
		value["name"] = refValue
	case "FOR_EACH":
		parts := strings.SplitN(refValue, ".", 2)
		if len(parts) == 2 {
			value["forEachTaskId"] = parts[0]
			value["name"] = parts[1]
		} else {
			value["name"] = refValue
		}
	}

	leftValue := map[string]interface{}{
		"type":  "REFERENCE",
		"value": value,
	}
	if realType != "" {
		leftValue["realType"] = realType
	}
	return leftValue
}

// createAPIValue creates a value in API format
func createAPIValue(valueType string, value interface{}) map[string]interface{} {
	return map[string]interface{}{
		"type":  valueType,
		"value": value,
	}
}

// =====================================================
// API Format Tests - leftValue with type: "FIELD"
// =====================================================

func TestFilterExport_LeftValueFieldType(t *testing.T) {
	gen := NewFilterHCLGenerator(nil)
	filter := map[string]interface{}{
		"type":      "EQUALS",
		"leftValue": createAPIFieldLeftValue("field-abc-123"),
		"value": map[string]interface{}{
			"type":  "TEXT",
			"value": "active",
		},
	}

	result := gen.Generate(filter, 1)

	assertContains(t, result, "left_value = {", "Expected left_value block")
	assertContains(t, result, `field_id = "field-abc-123"`, "Expected field_id in left_value")
}

func TestFilterExport_LeftValueFieldTypeWithUUIDMap(t *testing.T) {
	uuidMap := map[string]string{
		"field-abc-123": "elementum_field.status.id",
	}
	gen := NewFilterHCLGenerator(uuidMap)
	filter := map[string]interface{}{
		"type":      "EQUALS",
		"leftValue": createAPIFieldLeftValue("field-abc-123"),
		"value": map[string]interface{}{
			"type":  "TEXT",
			"value": "active",
		},
	}

	result := gen.Generate(filter, 1)

	assertContains(t, result, "field_id = elementum_field.status.id", "Expected beautified field_id")
}

// =====================================================
// API Format Tests - leftValue with type: "REFERENCE"
// =====================================================

func TestFilterExport_LeftValueReferenceTypeTrigger(t *testing.T) {
	gen := NewFilterHCLGenerator(nil)
	filter := map[string]interface{}{
		"type":      "EQUALS",
		"leftValue": createAPIReferenceLeftValue("TRIGGER", "record.field-uuid.id", "TEXT"),
		"value": map[string]interface{}{
			"type":  "TEXT",
			"value": "test",
		},
	}

	result := gen.Generate(filter, 1)

	assertContains(t, result, "left_value = {", "Expected left_value block")
	assertContains(t, result, `value_reference = "trigger.record.field-uuid.id"`, "Expected trigger reference")
	assertContains(t, result, `real_type = "text"`, "Expected real_type")
}

func TestFilterExport_LeftValueReferenceTypeTask(t *testing.T) {
	gen := NewFilterHCLGenerator(nil)
	filter := map[string]interface{}{
		"type":      "EQUALS",
		"leftValue": createAPIReferenceLeftValue("TASK", "task-123.result.count", "NUMBER"),
		"value": map[string]interface{}{
			"type":  "NUMBER",
			"value": float64(100),
		},
	}

	result := gen.Generate(filter, 1)

	assertContains(t, result, "left_value = {", "Expected left_value block")
	assertContains(t, result, `value_reference = "task.task-123.result.count"`, "Expected task reference")
	assertContains(t, result, `real_type = "number"`, "Expected real_type")
}

func TestFilterExport_LeftValueReferenceTypeVariable(t *testing.T) {
	gen := NewFilterHCLGenerator(nil)
	filter := map[string]interface{}{
		"type":      "EQUALS",
		"leftValue": createAPIReferenceLeftValue("VARIABLE", "my_variable", "TEXT"),
		"value": map[string]interface{}{
			"type":  "TEXT",
			"value": "expected",
		},
	}

	result := gen.Generate(filter, 1)

	assertContains(t, result, "left_value = {", "Expected left_value block")
	assertContains(t, result, `value_reference = "variable.my_variable"`, "Expected variable reference")
}

func TestFilterExport_LeftValueReferenceTypeForEach(t *testing.T) {
	gen := NewFilterHCLGenerator(nil)
	filter := map[string]interface{}{
		"type":      "EQUALS",
		"leftValue": createAPIReferenceLeftValue("FOR_EACH", "task-for-each.Item", "TEXT"),
		"value": map[string]interface{}{
			"type":  "TEXT",
			"value": "expected",
		},
	}

	result := gen.Generate(filter, 1)

	assertContains(t, result, "left_value = {", "Expected left_value block")
	assertContains(t, result, `value_reference = "for_each.task-for-each.Item"`, "Expected for_each reference")
}

// =====================================================
// API Format Tests - value with type: "REFERENCE"
// =====================================================

func TestFilterExport_ValueReferenceTypeTrigger(t *testing.T) {
	gen := NewFilterHCLGenerator(nil)
	filter := map[string]interface{}{
		"type":      "EQUALS",
		"leftValue": createAPIFieldLeftValue("field-123"),
		"value": map[string]interface{}{
			"type":     "REFERENCE",
			"realType": "TEXT",
			"value": map[string]interface{}{
				"type": "TRIGGER",
				"name": "record.created_by.id",
			},
		},
	}

	result := gen.Generate(filter, 1)

	assertContains(t, result, "value = {", "Expected value block")
	assertContains(t, result, `type = "reference"`, "Expected reference type")
	assertContains(t, result, `value_reference = "trigger.record.created_by.id"`, "Expected trigger reference in value")
}

func TestFilterExport_ValueReferenceTypeTask(t *testing.T) {
	gen := NewFilterHCLGenerator(nil)
	filter := map[string]interface{}{
		"type":      "GREATER_THAN",
		"leftValue": createAPIFieldLeftValue("field-score"),
		"value": map[string]interface{}{
			"type":     "REFERENCE",
			"realType": "NUMBER",
			"value": map[string]interface{}{
				"type":   "TASK",
				"taskId": "calc-task",
				"name":   "result",
			},
		},
	}

	result := gen.Generate(filter, 1)

	assertContains(t, result, "value = {", "Expected value block")
	assertContains(t, result, `type = "reference"`, "Expected reference type")
	assertContains(t, result, `value_reference = "task.calc-task.result"`, "Expected task reference in value")
}

func TestFilterExport_ValueFieldType(t *testing.T) {
	gen := NewFilterHCLGenerator(nil)
	filter := map[string]interface{}{
		"type":      "EQUALS",
		"leftValue": createAPIFieldLeftValue("field-a"),
		"value": map[string]interface{}{
			"type":  "FIELD",
			"field": "field-b",
		},
	}

	result := gen.Generate(filter, 1)

	assertContains(t, result, "value = {", "Expected value block")
	assertContains(t, result, `type = "field"`, "Expected field type")
	assertContains(t, result, `field_id = "field-b"`, "Expected field_id in value")
}

// =====================================================
// Comparison Operators
// =====================================================

func TestFilterExport_Equals(t *testing.T) {
	gen := NewFilterHCLGenerator(nil)
	filter := map[string]interface{}{
		"type":      "EQUALS",
		"leftValue": createAPIFieldLeftValue("field-status"),
		"value":     createAPIValue("TEXT", "active"),
	}

	result := gen.Generate(filter, 1)

	assertContains(t, result, `type = "equals"`, "Expected equals type")
	assertContains(t, result, "left_value = {", "Expected left_value")
	assertContains(t, result, "value = {", "Expected value")
}

func TestFilterExport_EqualsWithNot(t *testing.T) {
	gen := NewFilterHCLGenerator(nil)
	filter := map[string]interface{}{
		"type":      "EQUALS",
		"not":       true,
		"leftValue": createAPIFieldLeftValue("field-status"),
		"value":     createAPIValue("TEXT", "inactive"),
	}

	result := gen.Generate(filter, 1)

	assertContains(t, result, `type = "equals"`, "Expected equals type")
	assertContains(t, result, "not = true", "Expected not flag")
}

func TestFilterExport_GreaterThan(t *testing.T) {
	gen := NewFilterHCLGenerator(nil)
	filter := map[string]interface{}{
		"type":      "GREATER_THAN",
		"leftValue": createAPIFieldLeftValue("field-amount"),
		"value":     createAPIValue("NUMBER", float64(100)),
	}

	result := gen.Generate(filter, 1)

	assertContains(t, result, `type = "greater_than"`, "Expected greater_than type")
}

func TestFilterExport_LessThan(t *testing.T) {
	gen := NewFilterHCLGenerator(nil)
	filter := map[string]interface{}{
		"type":      "LESS_THAN",
		"leftValue": createAPIFieldLeftValue("field-score"),
		"value":     createAPIValue("NUMBER", float64(50)),
	}

	result := gen.Generate(filter, 1)

	assertContains(t, result, `type = "less_than"`, "Expected less_than type")
}

func TestFilterExport_GreaterThanEquals(t *testing.T) {
	gen := NewFilterHCLGenerator(nil)
	filter := map[string]interface{}{
		"type":      "GREATER_THAN_EQUALS",
		"leftValue": createAPIFieldLeftValue("field-count"),
		"value":     createAPIValue("NUMBER", float64(10)),
	}

	result := gen.Generate(filter, 1)

	assertContains(t, result, `type = "greater_than_equals"`, "Expected greater_than_equals type")
}

func TestFilterExport_LessThanEquals(t *testing.T) {
	gen := NewFilterHCLGenerator(nil)
	filter := map[string]interface{}{
		"type":      "LESS_THAN_EQUALS",
		"leftValue": createAPIFieldLeftValue("field-priority"),
		"value":     createAPIValue("NUMBER", float64(5)),
	}

	result := gen.Generate(filter, 1)

	assertContains(t, result, `type = "less_than_equals"`, "Expected less_than_equals type")
}

func TestFilterExport_Between(t *testing.T) {
	gen := NewFilterHCLGenerator(nil)
	filter := map[string]interface{}{
		"type":      "BETWEEN",
		"leftValue": createAPIFieldLeftValue("field-date"),
		"min":       createAPIValue("DATE", "2024-01-01"),
		"max":       createAPIValue("DATE", "2024-12-31"),
	}

	result := gen.Generate(filter, 1)

	assertContains(t, result, `type = "between"`, "Expected between type")
	// Note: min/max handling may need to be verified based on actual implementation
}

// =====================================================
// Null Checking
// =====================================================

func TestFilterExport_IsNull(t *testing.T) {
	gen := NewFilterHCLGenerator(nil)
	filter := map[string]interface{}{
		"type":      "IS_NULL",
		"leftValue": createAPIFieldLeftValue("field-optional"),
	}

	result := gen.Generate(filter, 1)

	assertContains(t, result, `type = "is_null"`, "Expected is_null type")
	assertContains(t, result, "left_value = {", "Expected left_value")
}

func TestFilterExport_IsNotNull(t *testing.T) {
	gen := NewFilterHCLGenerator(nil)
	filter := map[string]interface{}{
		"type":      "IS_NULL",
		"not":       true,
		"leftValue": createAPIFieldLeftValue("field-required"),
	}

	result := gen.Generate(filter, 1)

	assertContains(t, result, `type = "is_null"`, "Expected is_null type")
	assertContains(t, result, "not = true", "Expected not flag for IS_NOT_NULL")
}

// =====================================================
// List Operators
// =====================================================

func TestFilterExport_In(t *testing.T) {
	gen := NewFilterHCLGenerator(nil)
	filter := map[string]interface{}{
		"type":      "IN",
		"leftValue": createAPIFieldLeftValue("field-status"),
		"values": []interface{}{
			createAPIValue("TEXT", "active"),
			createAPIValue("TEXT", "pending"),
			createAPIValue("TEXT", "review"),
		},
	}

	result := gen.Generate(filter, 1)

	assertContains(t, result, `type = "in"`, "Expected in type")
	assertContains(t, result, "values = [", "Expected values array")
}

func TestFilterExport_InWithNot(t *testing.T) {
	gen := NewFilterHCLGenerator(nil)
	filter := map[string]interface{}{
		"type":      "IN",
		"not":       true,
		"leftValue": createAPIFieldLeftValue("field-status"),
		"values": []interface{}{
			createAPIValue("TEXT", "deleted"),
			createAPIValue("TEXT", "archived"),
		},
	}

	result := gen.Generate(filter, 1)

	assertContains(t, result, `type = "in"`, "Expected in type")
	assertContains(t, result, "not = true", "Expected not flag for NOT IN")
}

func TestFilterExport_Overlaps(t *testing.T) {
	gen := NewFilterHCLGenerator(nil)
	filter := map[string]interface{}{
		"type":      "OVERLAPS",
		"leftValue": createAPIFieldLeftValue("field-tags"),
		"values": []interface{}{
			createAPIValue("TEXT", "urgent"),
			createAPIValue("TEXT", "important"),
		},
	}

	result := gen.Generate(filter, 1)

	assertContains(t, result, `type = "overlaps"`, "Expected overlaps type")
}

func TestFilterExport_Contains(t *testing.T) {
	gen := NewFilterHCLGenerator(nil)
	filter := map[string]interface{}{
		"type":      "CONTAINS",
		"leftValue": createAPIFieldLeftValue("field-labels"),
		"value":     createAPIValue("TEXT", "high-priority"),
	}

	result := gen.Generate(filter, 1)

	assertContains(t, result, `type = "contains"`, "Expected contains type")
}

// =====================================================
// Text Operators
// =====================================================

func TestFilterExport_Like(t *testing.T) {
	gen := NewFilterHCLGenerator(nil)
	filter := map[string]interface{}{
		"type":            "LIKE",
		"leftValue":       createAPIFieldLeftValue("field-name"),
		"value":           createAPIValue("TEXT", "test"),
		"caseInsensitive": true,
		"mode":            "CONTAINS",
	}

	result := gen.Generate(filter, 1)

	assertContains(t, result, `type = "like"`, "Expected like type")
	assertContains(t, result, `mode = "contains"`, "Expected mode")
}

func TestFilterExport_LikeStartsWith(t *testing.T) {
	gen := NewFilterHCLGenerator(nil)
	filter := map[string]interface{}{
		"type":      "LIKE",
		"leftValue": createAPIFieldLeftValue("field-title"),
		"value":     createAPIValue("TEXT", "URGENT"),
		"mode":      "STARTS_WITH",
	}

	result := gen.Generate(filter, 1)

	assertContains(t, result, `mode = "starts_with"`, "Expected starts_with mode")
}

func TestFilterExport_LikeEndsWith(t *testing.T) {
	gen := NewFilterHCLGenerator(nil)
	filter := map[string]interface{}{
		"type":      "LIKE",
		"leftValue": createAPIFieldLeftValue("field-email"),
		"value":     createAPIValue("TEXT", "@company.com"),
		"mode":      "ENDS_WITH",
	}

	result := gen.Generate(filter, 1)

	assertContains(t, result, `mode = "ends_with"`, "Expected ends_with mode")
}

func TestFilterExport_Search(t *testing.T) {
	gen := NewFilterHCLGenerator(nil)
	filter := map[string]interface{}{
		"type":  "SEARCH",
		"value": createAPIValue("TEXT", "query text"),
	}

	result := gen.Generate(filter, 1)

	assertContains(t, result, `type = "search"`, "Expected search type")
}

// =====================================================
// Logical Operators
// =====================================================

func TestFilterExport_And(t *testing.T) {
	gen := NewFilterHCLGenerator(nil)
	filter := map[string]interface{}{
		"type": "AND",
		"children": []interface{}{
			map[string]interface{}{
				"type":      "EQUALS",
				"leftValue": createAPIFieldLeftValue("field-status"),
				"value":     createAPIValue("TEXT", "active"),
			},
			map[string]interface{}{
				"type":      "GREATER_THAN",
				"leftValue": createAPIFieldLeftValue("field-priority"),
				"value":     createAPIValue("NUMBER", float64(5)),
			},
		},
	}

	result := gen.Generate(filter, 1)

	assertContains(t, result, `type = "and"`, "Expected and type")
	assertContains(t, result, "children = [", "Expected children array")
	assertContains(t, result, `type = "equals"`, "Expected equals child")
	assertContains(t, result, `type = "greater_than"`, "Expected greater_than child")
}

func TestFilterExport_Or(t *testing.T) {
	gen := NewFilterHCLGenerator(nil)
	filter := map[string]interface{}{
		"type": "OR",
		"children": []interface{}{
			map[string]interface{}{
				"type":      "EQUALS",
				"leftValue": createAPIFieldLeftValue("field-status"),
				"value":     createAPIValue("TEXT", "pending"),
			},
			map[string]interface{}{
				"type":      "EQUALS",
				"leftValue": createAPIFieldLeftValue("field-status"),
				"value":     createAPIValue("TEXT", "review"),
			},
		},
	}

	result := gen.Generate(filter, 1)

	assertContains(t, result, `type = "or"`, "Expected or type")
	assertContains(t, result, "children = [", "Expected children array")
}

func TestFilterExport_NestedAndOr(t *testing.T) {
	gen := NewFilterHCLGenerator(nil)
	filter := map[string]interface{}{
		"type": "AND",
		"children": []interface{}{
			map[string]interface{}{
				"type": "OR",
				"children": []interface{}{
					map[string]interface{}{
						"type":      "EQUALS",
						"leftValue": createAPIFieldLeftValue("field-status"),
						"value":     createAPIValue("TEXT", "open"),
					},
					map[string]interface{}{
						"type":      "EQUALS",
						"leftValue": createAPIFieldLeftValue("field-status"),
						"value":     createAPIValue("TEXT", "pending"),
					},
				},
			},
			map[string]interface{}{
				"type":      "GREATER_THAN",
				"leftValue": createAPIFieldLeftValue("field-priority"),
				"value":     createAPIValue("NUMBER", float64(50)),
			},
		},
	}

	result := gen.Generate(filter, 1)

	assertContains(t, result, `type = "and"`, "Expected top-level and")
	assertContains(t, result, `type = "or"`, "Expected nested or")
}

// =====================================================
// Special Filters
// =====================================================

func TestFilterExport_True(t *testing.T) {
	gen := NewFilterHCLGenerator(nil)
	filter := map[string]interface{}{
		"type": "TRUE",
	}

	result := gen.Generate(filter, 1)

	assertContains(t, result, `type = "true"`, "Expected true type")
}

func TestFilterExport_False(t *testing.T) {
	gen := NewFilterHCLGenerator(nil)
	filter := map[string]interface{}{
		"type": "FALSE",
	}

	result := gen.Generate(filter, 1)

	assertContains(t, result, `type = "false"`, "Expected false type")
}

func TestFilterExport_CurrentRange(t *testing.T) {
	gen := NewFilterHCLGenerator(nil)
	filter := map[string]interface{}{
		"type":      "CURRENT_RANGE",
		"leftValue": createAPIFieldLeftValue("field-due-date"),
		"relativeValue": map[string]interface{}{
			"operator": "NEXT",
			"unit":     "DAY",
			"amount":   float64(7),
		},
	}

	result := gen.Generate(filter, 1)

	assertContains(t, result, `type = "current_range"`, "Expected current_range type")
	assertContains(t, result, "relative_value = {", "Expected relative_value")
}

func TestFilterExport_ConsecutiveRange(t *testing.T) {
	gen := NewFilterHCLGenerator(nil)
	filter := map[string]interface{}{
		"type":      "CONSECUTIVE_RANGE",
		"leftValue": createAPIFieldLeftValue("field-created-at"),
		"relativeValue": map[string]interface{}{
			"operator": "LAST",
			"unit":     "MONTH",
			"amount":   float64(3),
		},
	}

	result := gen.Generate(filter, 1)

	assertContains(t, result, `type = "consecutive_range"`, "Expected consecutive_range type")
}

// =====================================================
// Complex Real-World Scenarios
// =====================================================

func TestFilterExport_ComplexAndWithReferences(t *testing.T) {
	uuidMap := map[string]string{
		"field-status":   "elementum_field.status.id",
		"field-assignee": "elementum_user_field.assignee.id",
	}
	gen := NewFilterHCLGenerator(uuidMap)

	filter := map[string]interface{}{
		"type": "AND",
		"children": []interface{}{
			map[string]interface{}{
				"type":      "EQUALS",
				"leftValue": createAPIFieldLeftValue("field-status"),
				"value":     createAPIValue("TEXT", "open"),
			},
			map[string]interface{}{
				"type":      "EQUALS",
				"leftValue": createAPIFieldLeftValue("field-assignee"),
				"value": map[string]interface{}{
					"type":     "REFERENCE",
					"realType": "USER",
					"value": map[string]interface{}{
						"type": "TRIGGER",
						"name": "record.created_by.id",
					},
				},
			},
		},
	}

	result := gen.Generate(filter, 1)

	assertContains(t, result, "field_id = elementum_field.status.id", "Expected beautified status field")
	assertContains(t, result, "field_id = elementum_user_field.assignee.id", "Expected beautified assignee field")
	assertContains(t, result, `value_reference = "trigger.record.created_by.id"`, "Expected trigger reference")
}

func TestFilterExport_RecordUpdatedTriggerFilter(t *testing.T) {
	// Simulates the exact API response format for a record_updated trigger filter
	gen := NewFilterHCLGenerator(nil)
	filter := map[string]interface{}{
		"type": "AND",
		"children": []interface{}{
			map[string]interface{}{
				"type": "EQUALS",
				"leftValue": map[string]interface{}{
					"type":  "FIELD",
					"field": "field-status-123",
				},
				"value": map[string]interface{}{
					"type":  "TEXT",
					"value": "approved",
				},
			},
			map[string]interface{}{
				"type": "IS_NULL",
				"not":  true,
				"leftValue": map[string]interface{}{
					"type":  "FIELD",
					"field": "field-approver-456",
				},
			},
		},
	}

	result := gen.Generate(filter, 1)

	assertContains(t, result, "filter = {", "Expected filter block")
	assertContains(t, result, `type = "and"`, "Expected and type")
	assertContains(t, result, `field_id = "field-status-123"`, "Expected status field_id")
	assertContains(t, result, `field_id = "field-approver-456"`, "Expected approver field_id")
	assertContains(t, result, `type = "is_null"`, "Expected is_null type")
	assertContains(t, result, "not = true", "Expected not flag")
}

// =====================================================
// Test Helper Functions
// =====================================================

func assertContains(t *testing.T, result, expected, message string) {
	t.Helper()
	if !strings.Contains(result, expected) {
		t.Errorf("%s: expected %q in result:\n%s", message, expected, result)
	}
}

func assertNotContains(t *testing.T, result, unexpected, message string) {
	t.Helper()
	if strings.Contains(result, unexpected) {
		t.Errorf("%s: unexpected %q in result:\n%s", message, unexpected, result)
	}
}
