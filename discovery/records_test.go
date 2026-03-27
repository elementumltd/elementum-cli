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

package discovery

import (
	"encoding/json"
	"testing"
)

func TestExtractDisplayValue(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    interface{}
		expected string
	}{
		{
			name:     "nil value",
			input:    nil,
			expected: "",
		},
		{
			name:     "simple string",
			input:    "Hello World",
			expected: "Hello World",
		},
		{
			name:     "integer as float64",
			input:    float64(42),
			expected: "42",
		},
		{
			name:     "float with decimals",
			input:    float64(3.14),
			expected: "3.14",
		},
		{
			name:     "boolean true",
			input:    true,
			expected: "Yes",
		},
		{
			name:     "boolean false",
			input:    false,
			expected: "No",
		},
		{
			name: "picklist with label",
			input: map[string]interface{}{
				"id":    "abc-123",
				"label": "Open",
			},
			expected: "Open",
		},
		{
			name: "user with name",
			input: map[string]interface{}{
				"id":    "user-123",
				"name":  "John Doe",
				"email": "john@example.com",
			},
			expected: "John Doe",
		},
		{
			name: "map with only id falls back",
			input: map[string]interface{}{
				"id": "abc-123",
			},
			expected: "abc-123",
		},
		{
			name:     "empty map",
			input:    map[string]interface{}{},
			expected: "",
		},
		{
			name: "multi-select array with labels",
			input: []interface{}{
				map[string]interface{}{"label": "urgent"},
				map[string]interface{}{"label": "bug"},
			},
			expected: "urgent, bug",
		},
		{
			name: "array of strings",
			input: []interface{}{
				"one",
				"two",
				"three",
			},
			expected: "one, two, three",
		},
		{
			name:     "empty array",
			input:    []interface{}{},
			expected: "",
		},
		{
			name: "mixed array",
			input: []interface{}{
				"text",
				map[string]interface{}{"label": "option"},
				float64(123),
			},
			expected: "text, option, 123",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := ExtractDisplayValue(tc.input)
			if result != tc.expected {
				t.Errorf("ExtractDisplayValue(%v) = %q, want %q", tc.input, result, tc.expected)
			}
		})
	}
}

func TestGetFieldValue(t *testing.T) {
	t.Parallel()

	record := Record{
		ID:     "test-id",
		Handle: "TEST-1",
		Data: map[string]interface{}{
			"Title":    "Test Record",
			"Priority": map[string]interface{}{"id": "p1", "label": "High"},
			"Tags": []interface{}{
				map[string]interface{}{"label": "bug"},
				map[string]interface{}{"label": "urgent"},
			},
			"Count":    float64(42),
			"IsActive": true,
		},
	}

	tests := []struct {
		name      string
		fieldName string
		expected  string
	}{
		{
			name:      "simple text field",
			fieldName: "Title",
			expected:  "Test Record",
		},
		{
			name:      "picklist field",
			fieldName: "Priority",
			expected:  "High",
		},
		{
			name:      "multi-select field",
			fieldName: "Tags",
			expected:  "bug, urgent",
		},
		{
			name:      "number field",
			fieldName: "Count",
			expected:  "42",
		},
		{
			name:      "boolean field",
			fieldName: "IsActive",
			expected:  "Yes",
		},
		{
			name:      "missing field",
			fieldName: "NonExistent",
			expected:  "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := GetFieldValue(record, tc.fieldName)
			if result != tc.expected {
				t.Errorf("GetFieldValue(record, %q) = %q, want %q", tc.fieldName, result, tc.expected)
			}
		})
	}
}

func TestBuildNamespaceFilter(t *testing.T) {
	t.Parallel()

	namespace := "my-app"
	filter := buildNamespaceFilter(namespace)

	if filter == nil {
		t.Fatal("expected non-nil filter")
	}

	// Verify it's valid JSON
	var parsed map[string]interface{}
	if err := json.Unmarshal(*filter, &parsed); err != nil {
		t.Fatalf("failed to parse filter JSON: %v", err)
	}

	// Verify filter structure
	if parsed["type"] != "LIKE" {
		t.Errorf("expected type LIKE, got %v", parsed["type"])
	}
	if parsed["field"] != "namespace" {
		t.Errorf("expected field namespace, got %v", parsed["field"])
	}
	if parsed["mode"] != "EQUALS" {
		t.Errorf("expected mode EQUALS, got %v", parsed["mode"])
	}

	value, ok := parsed["value"].(map[string]interface{})
	if !ok {
		t.Fatal("expected value to be a map")
	}
	if value["value"] != namespace {
		t.Errorf("expected value.value to be %q, got %v", namespace, value["value"])
	}
}

func TestResolvePicklistValues(t *testing.T) {
	t.Parallel()

	// Create a picklist value map using UUID-formatted IDs (matching the regex pattern)
	valueMap := PicklistValueMap{
		"Status:550e8400-e29b-41d4-a716-446655440001":   "Open",
		"Status:550e8400-e29b-41d4-a716-446655440002":   "Closed",
		"Priority:550e8400-e29b-41d4-a716-446655440003": "High",
		"Priority:550e8400-e29b-41d4-a716-446655440004": "Low",
		"Tags:550e8400-e29b-41d4-a716-446655440005":     "Bug",
		"Tags:550e8400-e29b-41d4-a716-446655440006":     "Feature",
	}

	// Create test records with UUID values
	records := []Record{
		{
			ID:     "1",
			Handle: "TEST-1",
			Data: map[string]interface{}{
				"Title":    "Test Record",
				"Status":   "550e8400-e29b-41d4-a716-446655440001", // Single picklist
				"Priority": "550e8400-e29b-41d4-a716-446655440003", // Single picklist
				"Tags": []interface{}{
					"550e8400-e29b-41d4-a716-446655440005",
					"550e8400-e29b-41d4-a716-446655440006",
				}, // Multi-picklist
				"Count": float64(42), // Non-picklist
			},
		},
		{
			ID:     "2",
			Handle: "TEST-2",
			Data: map[string]interface{}{
				"Status":   "550e8400-e29b-41d4-a716-446655440002",
				"Priority": "550e8400-e29b-41d4-a716-446655440099", // Unknown UUID (should stay as-is)
				"Notes":    "Some text",                            // Non-picklist
			},
		},
	}

	// Resolve picklist values
	ResolvePicklistValues(records, valueMap)

	// Verify first record
	if records[0].Data["Status"] != "Open" {
		t.Errorf("expected Status to be 'Open', got %v", records[0].Data["Status"])
	}
	if records[0].Data["Priority"] != "High" {
		t.Errorf("expected Priority to be 'High', got %v", records[0].Data["Priority"])
	}
	tags, ok := records[0].Data["Tags"].([]interface{})
	if !ok {
		t.Fatal("expected Tags to be []interface{}")
	}
	if len(tags) != 2 || tags[0] != "Bug" || tags[1] != "Feature" {
		t.Errorf("expected Tags to be ['Bug', 'Feature'], got %v", tags)
	}
	if records[0].Data["Count"] != float64(42) {
		t.Errorf("expected Count to remain 42, got %v", records[0].Data["Count"])
	}

	// Verify second record
	if records[1].Data["Status"] != "Closed" {
		t.Errorf("expected Status to be 'Closed', got %v", records[1].Data["Status"])
	}
	// Unknown UUID should remain unchanged (not in valueMap)
	if records[1].Data["Priority"] != "550e8400-e29b-41d4-a716-446655440099" {
		t.Errorf("expected Priority to remain '550e8400-e29b-41d4-a716-446655440099', got %v", records[1].Data["Priority"])
	}
	if records[1].Data["Notes"] != "Some text" {
		t.Errorf("expected Notes to remain 'Some text', got %v", records[1].Data["Notes"])
	}
}

func TestResolveValue(t *testing.T) {
	t.Parallel()

	valueMap := PicklistValueMap{
		"Status:550e8400-e29b-41d4-a716-446655440000": "Active",
	}

	tests := []struct {
		name      string
		fieldName string
		value     interface{}
		expected  interface{}
	}{
		{
			name:      "nil value",
			fieldName: "Status",
			value:     nil,
			expected:  nil,
		},
		{
			name:      "UUID that matches",
			fieldName: "Status",
			value:     "550e8400-e29b-41d4-a716-446655440000",
			expected:  "Active",
		},
		{
			name:      "UUID that doesn't match",
			fieldName: "Other",
			value:     "550e8400-e29b-41d4-a716-446655440000",
			expected:  "550e8400-e29b-41d4-a716-446655440000",
		},
		{
			name:      "non-UUID string",
			fieldName: "Status",
			value:     "some text",
			expected:  "some text",
		},
		{
			name:      "array of UUIDs",
			fieldName: "Status",
			value:     []interface{}{"550e8400-e29b-41d4-a716-446655440000", "other-id"},
			expected:  []interface{}{"Active", "other-id"},
		},
		{
			name:      "number value",
			fieldName: "Count",
			value:     float64(123),
			expected:  float64(123),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := resolveValue(tc.fieldName, tc.value, valueMap)

			// Handle slice comparison
			if expectedSlice, ok := tc.expected.([]interface{}); ok {
				resultSlice, ok := result.([]interface{})
				if !ok {
					t.Errorf("expected slice, got %T", result)
					return
				}
				if len(resultSlice) != len(expectedSlice) {
					t.Errorf("slice length mismatch: got %d, want %d", len(resultSlice), len(expectedSlice))
					return
				}
				for i := range expectedSlice {
					if resultSlice[i] != expectedSlice[i] {
						t.Errorf("slice[%d]: got %v, want %v", i, resultSlice[i], expectedSlice[i])
					}
				}
				return
			}

			if result != tc.expected {
				t.Errorf("resolveValue(%q, %v) = %v, want %v", tc.fieldName, tc.value, result, tc.expected)
			}
		})
	}
}

func TestDeleteRecords(t *testing.T) {
	t.Parallel()

	// Test the DeleteRecords batch function with empty list
	deleted, errors := DeleteRecords(nil, nil, []string{})
	if deleted != 0 {
		t.Errorf("expected 0 deleted, got %d", deleted)
	}
	if len(errors) != 0 {
		t.Errorf("expected 0 errors, got %d", len(errors))
	}
}
