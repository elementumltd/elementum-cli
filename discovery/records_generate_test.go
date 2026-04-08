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
	"math/rand"
	"strings"
	"testing"
)

func TestGenerateRecords_Count(t *testing.T) {
	t.Parallel()

	fields := []AspectFieldInfo{
		{ID: "f1", Name: "Title", Type: "AspectTextField", Required: true, SemanticTags: []string{"TITLE"}},
		{ID: "f2", Name: "Status", Type: "AspectPicklistField", Required: true, Options: []FieldOptionInfo{
			{ID: "opt1", Label: "Open"},
			{ID: "opt2", Label: "Closed"},
		}},
	}

	for _, count := range []int{0, 1, 5, 50} {
		records := GenerateRecords(fields, count, 42)
		if len(records) != count {
			t.Errorf("GenerateRecords(count=%d) returned %d records", count, len(records))
		}
	}
}

func TestGenerateRecords_Deterministic(t *testing.T) {
	t.Parallel()

	fields := []AspectFieldInfo{
		{ID: "f1", Name: "Title", Type: "AspectTextField", Required: true, SemanticTags: []string{"TITLE"}},
		{ID: "f2", Name: "Count", Type: "AspectNumberField", Required: true},
		{ID: "f3", Name: "Active", Type: "AspectBooleanField", Required: true},
	}

	seed := int64(12345)
	run1 := GenerateRecords(fields, 10, seed)
	run2 := GenerateRecords(fields, 10, seed)

	if len(run1) != len(run2) {
		t.Fatalf("different lengths: %d vs %d", len(run1), len(run2))
	}

	for i := range run1 {
		for key := range run1[i] {
			v1 := run1[i][key]
			v2 := run2[i][key]
			if v1 != v2 {
				t.Errorf("record %d, field %q: %v != %v", i, key, v1, v2)
			}
		}
	}
}

func TestGenerateRecords_FieldTypes(t *testing.T) {
	t.Parallel()

	fields := []AspectFieldInfo{
		{ID: "f1", Name: "Title", Type: "AspectTextField", Required: true, SemanticTags: []string{"TITLE"}},
		{ID: "f2", Name: "Description", Type: "AspectHtmlField", Required: true},
		{ID: "f3", Name: "Count", Type: "AspectNumberField", Required: true},
		{ID: "f4", Name: "Score", Type: "AspectDecimalField", Required: true},
		{ID: "f5", Name: "Active", Type: "AspectBooleanField", Required: true},
		{ID: "f6", Name: "Due Date", Type: "AspectDateField", Required: true},
		{ID: "f7", Name: "Created", Type: "AspectDateTimeField", Required: true},
		{ID: "f8", Name: "Status", Type: "AspectPicklistField", Required: true, Options: []FieldOptionInfo{
			{ID: "opt1", Label: "Open"},
			{ID: "opt2", Label: "Closed"},
			{ID: "opt3", Label: "In Progress"},
		}},
		{ID: "f9", Name: "Tags", Type: "AspectMultiPicklistField", Required: true, Options: []FieldOptionInfo{
			{ID: "tag1", Label: "Bug"},
			{ID: "tag2", Label: "Feature"},
			{ID: "tag3", Label: "Enhancement"},
		}},
	}

	records := GenerateRecords(fields, 5, 42)
	if len(records) != 5 {
		t.Fatalf("got %d records, want 5", len(records))
	}

	for i, rec := range records {
		// Required fields should always be present
		if _, ok := rec["f1"]; !ok {
			t.Errorf("record %d: missing required Title (f1)", i)
		}

		// Check Title is a string
		if title, ok := rec["f1"].(string); ok {
			if title == "" {
				t.Errorf("record %d: empty Title", i)
			}
		} else {
			t.Errorf("record %d: Title is not a string: %T", i, rec["f1"])
		}

		// Check Number is int64
		if count, ok := rec["f3"]; ok {
			if _, isInt := count.(int64); !isInt {
				t.Errorf("record %d: Count is not int64: %T", i, count)
			}
		}

		// Check Decimal is float64
		if score, ok := rec["f4"]; ok {
			if _, isFloat := score.(float64); !isFloat {
				t.Errorf("record %d: Score is not float64: %T", i, score)
			}
		}

		// Check Boolean
		if active, ok := rec["f5"]; ok {
			if _, isBool := active.(bool); !isBool {
				t.Errorf("record %d: Active is not bool: %T", i, active)
			}
		}

		// Check Date format (YYYY-MM-DD)
		if dueDate, ok := rec["f6"].(string); ok {
			if len(dueDate) != 10 || dueDate[4] != '-' || dueDate[7] != '-' {
				t.Errorf("record %d: invalid date format: %q", i, dueDate)
			}
		}

		// Check Picklist returns a valid option ID
		if status, ok := rec["f8"].(string); ok {
			validIDs := map[string]bool{"opt1": true, "opt2": true, "opt3": true}
			if !validIDs[status] {
				t.Errorf("record %d: invalid picklist value: %q", i, status)
			}
		}
	}
}

func TestGenerateRecords_SkipsSystemFields(t *testing.T) {
	t.Parallel()

	fields := []AspectFieldInfo{
		{ID: "f1", Name: "Title", Type: "AspectTextField", Required: true},
		{ID: "f2", Name: "Handle", Type: "AspectHandleField", System: true},
		{ID: "f3", Name: "Created At", Type: "AspectCreatedAtField", System: true},
		{ID: "f4", Name: "Calc", Type: "AspectCalculationField"},
		{ID: "f5", Name: "Attachments", Type: "AspectAttachmentField"},
	}

	records := GenerateRecords(fields, 1, 42)
	if len(records) != 1 {
		t.Fatalf("got %d records, want 1", len(records))
	}

	rec := records[0]

	if _, ok := rec["f1"]; !ok {
		t.Error("missing Title field")
	}

	for _, skipID := range []string{"f2", "f3", "f4", "f5"} {
		if _, ok := rec[skipID]; ok {
			t.Errorf("should not have field %s", skipID)
		}
	}
}

func TestGenerateRecords_SkipsUserGroupFields(t *testing.T) {
	t.Parallel()

	fields := []AspectFieldInfo{
		{ID: "f1", Name: "Title", Type: "AspectTextField", Required: true},
		{ID: "f2", Name: "Assigned User", Type: "AspectUserField"},
		{ID: "f3", Name: "Team", Type: "AspectGroupField"},
	}

	records := GenerateRecords(fields, 1, 42)
	if len(records) != 1 {
		t.Fatalf("got %d records, want 1", len(records))
	}

	rec := records[0]
	if _, ok := rec["f2"]; ok {
		t.Error("should not generate user field values")
	}
	if _, ok := rec["f3"]; ok {
		t.Error("should not generate group field values")
	}
}

func TestGenerateTextValue_Semantic(t *testing.T) {
	t.Parallel()

	rng := rand.New(rand.NewSource(42))

	tests := []struct {
		name       string
		field      AspectFieldInfo
		wantSubstr string
	}{
		{
			name:       "semantic title tag",
			field:      AspectFieldInfo{Name: "Subject", Type: "AspectTextField", Required: true, SemanticTags: []string{"TITLE"}},
			wantSubstr: "#",
		},
		{
			name:       "title by name",
			field:      AspectFieldInfo{Name: "Title", Type: "AspectTextField", Required: true},
			wantSubstr: "#",
		},
		{
			name:       "email field",
			field:      AspectFieldInfo{Name: "Contact Email", Type: "AspectTextField", Required: true},
			wantSubstr: "@example.com",
		},
		{
			name:       "url field",
			field:      AspectFieldInfo{Name: "Website URL", Type: "AspectTextField", Required: true},
			wantSubstr: "https://",
		},
		{
			name:       "generic field",
			field:      AspectFieldInfo{Name: "Custom", Type: "AspectTextField", Required: true},
			wantSubstr: "Sample Custom",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			val := generateFieldValue(tc.field, 0, rng)
			s, ok := val.(string)
			if !ok {
				t.Fatalf("expected string, got %T", val)
			}
			if tc.wantSubstr != "" && !strings.Contains(s, tc.wantSubstr) {
				t.Errorf("value %q doesn't contain %q", s, tc.wantSubstr)
			}
			if s == "" {
				t.Error("empty string value")
			}
		})
	}
}

func TestGenerateHTMLValue(t *testing.T) {
	t.Parallel()

	rng := rand.New(rand.NewSource(42))
	field := AspectFieldInfo{Name: "Notes", Type: "AspectHtmlField", Required: true}
	val := generateFieldValue(field, 0, rng)
	s, ok := val.(string)
	if !ok {
		t.Fatalf("expected string, got %T", val)
	}
	if !strings.Contains(s, "<p>") {
		t.Errorf("HTML value missing <p> tag: %q", s)
	}
}

func TestGenerateMultiPicklistValue_Count(t *testing.T) {
	t.Parallel()

	field := AspectFieldInfo{
		Name: "Tags",
		Type: "AspectMultiPicklistField",
		Options: []FieldOptionInfo{
			{ID: "1", Label: "A"},
			{ID: "2", Label: "B"},
			{ID: "3", Label: "C"},
			{ID: "4", Label: "D"},
			{ID: "5", Label: "E"},
		},
	}

	rng := rand.New(rand.NewSource(42))
	for i := 0; i < 20; i++ {
		val := generateMultiPicklistValue(field, rng)
		if val == nil {
			continue
		}
		selected, ok := val.([]string)
		if !ok {
			t.Fatalf("expected []string, got %T", val)
		}
		if len(selected) < 1 || len(selected) > 3 {
			t.Errorf("multi-picklist selected %d items, want 1-3", len(selected))
		}
	}
}

func TestGeneratePicklistValue_Empty(t *testing.T) {
	t.Parallel()

	field := AspectFieldInfo{
		Name:    "Empty",
		Type:    "AspectPicklistField",
		Options: nil,
	}

	rng := rand.New(rand.NewSource(42))
	val := generatePicklistValue(field, rng)
	if val != nil {
		t.Errorf("expected nil for empty options, got %v", val)
	}
}

func TestGenerateRecords_NonRequiredFieldsMayBeSkipped(t *testing.T) {
	t.Parallel()

	fields := []AspectFieldInfo{
		{ID: "f1", Name: "Title", Type: "AspectTextField", Required: true},
		{ID: "f2", Name: "Optional", Type: "AspectTextField", Required: false},
	}

	// Generate enough records that the optional field should be missing at least once
	records := GenerateRecords(fields, 100, 42)

	hasOptional := 0
	missingOptional := 0
	for _, rec := range records {
		if _, ok := rec["f2"]; ok {
			hasOptional++
		} else {
			missingOptional++
		}
	}

	if missingOptional == 0 {
		t.Error("expected some records to skip the optional field")
	}
	if hasOptional == 0 {
		t.Error("expected some records to have the optional field")
	}
}
