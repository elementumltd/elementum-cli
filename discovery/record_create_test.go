// Copyright (c) ElementumAI, Inc.
// SPDX-License-Identifier: MPL-2.0

package discovery

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseFieldAssignment(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantName  string
		wantValue string
		wantErr   bool
	}{
		{
			name:      "simple assignment",
			input:     "Title=Bug Report",
			wantName:  "Title",
			wantValue: "Bug Report",
			wantErr:   false,
		},
		{
			name:      "value with equals sign",
			input:     "Formula=a=b+c",
			wantName:  "Formula",
			wantValue: "a=b+c",
			wantErr:   false,
		},
		{
			name:      "value with spaces",
			input:     "Description=This is a long description",
			wantName:  "Description",
			wantValue: "This is a long description",
			wantErr:   false,
		},
		{
			name:      "name with spaces trimmed",
			input:     "  Title  =Test",
			wantName:  "Title",
			wantValue: "Test",
			wantErr:   false,
		},
		{
			name:      "empty value",
			input:     "Title=",
			wantName:  "Title",
			wantValue: "",
			wantErr:   false,
		},
		{
			name:      "UUID as name",
			input:     "fdb890d8-a296-4b8d-b0ec-2656e2fcfb70=test",
			wantName:  "fdb890d8-a296-4b8d-b0ec-2656e2fcfb70",
			wantValue: "test",
			wantErr:   false,
		},
		{
			name:    "no equals sign",
			input:   "TitleBug Report",
			wantErr: true,
		},
		{
			name:    "empty name",
			input:   "=value",
			wantErr: true,
		},
		{
			name:    "only whitespace name",
			input:   "   =value",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			name, value, err := ParseFieldAssignment(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseFieldAssignment() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if name != tt.wantName {
					t.Errorf("ParseFieldAssignment() name = %q, want %q", name, tt.wantName)
				}
				if value != tt.wantValue {
					t.Errorf("ParseFieldAssignment() value = %q, want %q", value, tt.wantValue)
				}
			}
		})
	}
}

func TestParseAttachmentAssignment(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		wantFieldName string
		wantFilePath  string
		wantErr       bool
	}{
		{
			name:          "simple attachment",
			input:         "Attachments=/path/to/file.pdf",
			wantFieldName: "Attachments",
			wantFilePath:  "/path/to/file.pdf",
			wantErr:       false,
		},
		{
			name:          "relative path",
			input:         "Files=./screenshot.png",
			wantFieldName: "Files",
			wantFilePath:  "./screenshot.png",
			wantErr:       false,
		},
		{
			name:          "empty field name (attach to record)",
			input:         "=/path/to/file.pdf",
			wantFieldName: "",
			wantFilePath:  "/path/to/file.pdf",
			wantErr:       false,
		},
		{
			name:    "no equals sign",
			input:   "Attachments/path/to/file.pdf",
			wantErr: true,
		},
		{
			name:    "empty path",
			input:   "Attachments=",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fieldName, filePath, err := ParseAttachmentAssignment(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseAttachmentAssignment() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if fieldName != tt.wantFieldName {
					t.Errorf("ParseAttachmentAssignment() fieldName = %q, want %q", fieldName, tt.wantFieldName)
				}
				if filePath != tt.wantFilePath {
					t.Errorf("ParseAttachmentAssignment() filePath = %q, want %q", filePath, tt.wantFilePath)
				}
			}
		})
	}
}

func TestResolveFieldNameToID(t *testing.T) {
	fields := []AspectFieldInfo{
		{ID: "11111111-1111-1111-1111-111111111111", Name: "Title", Type: "AspectTextField"},
		{ID: "22222222-2222-2222-2222-222222222222", Name: "Status", Type: "AspectPicklistField"},
		{ID: "33333333-3333-3333-3333-333333333333", Name: "Priority", Type: "AspectPicklistField"},
	}

	tests := []struct {
		name      string
		nameOrID  string
		wantID    string
		wantField bool // whether we expect to find field info
		wantErr   bool
	}{
		{
			name:      "resolve by name",
			nameOrID:  "Title",
			wantID:    "11111111-1111-1111-1111-111111111111",
			wantField: true,
			wantErr:   false,
		},
		{
			name:      "resolve by name case insensitive",
			nameOrID:  "title",
			wantID:    "11111111-1111-1111-1111-111111111111",
			wantField: true,
			wantErr:   false,
		},
		{
			name:      "resolve by name mixed case",
			nameOrID:  "TITLE",
			wantID:    "11111111-1111-1111-1111-111111111111",
			wantField: true,
			wantErr:   false,
		},
		{
			name:      "resolve by UUID that exists",
			nameOrID:  "22222222-2222-2222-2222-222222222222",
			wantID:    "22222222-2222-2222-2222-222222222222",
			wantField: true,
			wantErr:   false,
		},
		{
			name:      "resolve by UUID that doesn't exist in list",
			nameOrID:  "fdb890d8-a296-4b8d-b0ec-2656e2fcfb70",
			wantID:    "fdb890d8-a296-4b8d-b0ec-2656e2fcfb70",
			wantField: false, // UUID accepted even if not in list
			wantErr:   false,
		},
		{
			name:     "name not found",
			nameOrID: "NonExistent",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id, fieldInfo, err := ResolveFieldNameToID(fields, tt.nameOrID)
			if (err != nil) != tt.wantErr {
				t.Errorf("ResolveFieldNameToID() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if id != tt.wantID {
					t.Errorf("ResolveFieldNameToID() id = %q, want %q", id, tt.wantID)
				}
				if tt.wantField && fieldInfo == nil {
					t.Error("ResolveFieldNameToID() expected field info, got nil")
				}
				if !tt.wantField && fieldInfo != nil {
					t.Errorf("ResolveFieldNameToID() expected nil field info, got %+v", fieldInfo)
				}
			}
		})
	}
}

func TestResolveFieldValue_Text(t *testing.T) {
	field := &AspectFieldInfo{Type: "AspectTextField"}

	value, err := ResolveFieldValue(field, "Hello World")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if value != "Hello World" {
		t.Errorf("got %v, want %q", value, "Hello World")
	}
}

func TestResolveFieldValue_Html(t *testing.T) {
	field := &AspectFieldInfo{Type: "AspectHtmlField"}

	value, err := ResolveFieldValue(field, "<p>Hello</p>")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if value != "<p>Hello</p>" {
		t.Errorf("got %v, want %q", value, "<p>Hello</p>")
	}
}

func TestResolveFieldValue_Number(t *testing.T) {
	field := &AspectFieldInfo{Type: "AspectNumberField"}

	tests := []struct {
		input   string
		want    int64
		wantErr bool
	}{
		{"123", 123, false},
		{"-456", -456, false},
		{"0", 0, false},
		{"123.7", 123, false}, // floats truncated to int
		{"abc", 0, true},
		{"", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			value, err := ResolveFieldValue(field, tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ResolveFieldValue() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if v, ok := value.(int64); !ok || v != tt.want {
					t.Errorf("got %v (%T), want %d", value, value, tt.want)
				}
			}
		})
	}
}

func TestResolveFieldValue_Decimal(t *testing.T) {
	field := &AspectFieldInfo{Type: "AspectDecimalField"}

	tests := []struct {
		input   string
		want    float64
		wantErr bool
	}{
		{"123.45", 123.45, false},
		{"-456.78", -456.78, false},
		{"0", 0, false},
		{"123", 123, false},
		{"abc", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			value, err := ResolveFieldValue(field, tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ResolveFieldValue() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if v, ok := value.(float64); !ok || v != tt.want {
					t.Errorf("got %v (%T), want %f", value, value, tt.want)
				}
			}
		})
	}
}

func TestResolveFieldValue_Boolean(t *testing.T) {
	field := &AspectFieldInfo{Type: "AspectBooleanField"}

	tests := []struct {
		input   string
		want    bool
		wantErr bool
	}{
		{"true", true, false},
		{"True", true, false},
		{"TRUE", true, false},
		{"yes", true, false},
		{"Yes", true, false},
		{"1", true, false},
		{"on", true, false},
		{"false", false, false},
		{"False", false, false},
		{"FALSE", false, false},
		{"no", false, false},
		{"No", false, false},
		{"0", false, false},
		{"off", false, false},
		{"maybe", false, true},
		{"", false, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			value, err := ResolveFieldValue(field, tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ResolveFieldValue() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if v, ok := value.(bool); !ok || v != tt.want {
					t.Errorf("got %v (%T), want %t", value, value, tt.want)
				}
			}
		})
	}
}

func TestResolveFieldValue_Date(t *testing.T) {
	field := &AspectFieldInfo{Type: "AspectDateField"}

	tests := []struct {
		input   string
		want    string
		wantErr bool
	}{
		{"2024-01-15", "2024-01-15", false},
		{"2024-12-31", "2024-12-31", false},
		{"01-15-2024", "", true}, // wrong format
		{"2024/01/15", "", true}, // wrong format
		{"not a date", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			value, err := ResolveFieldValue(field, tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ResolveFieldValue() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if v, ok := value.(string); !ok || v != tt.want {
					t.Errorf("got %v (%T), want %q", value, value, tt.want)
				}
			}
		})
	}
}

func TestResolveFieldValue_DateTime(t *testing.T) {
	field := &AspectFieldInfo{Type: "AspectDateTimeField"}

	tests := []struct {
		input   string
		wantErr bool
	}{
		{"2024-01-15T10:30:00Z", false},
		{"2024-01-15T10:30:00", false},
		{"2024-01-15 10:30:00", false},
		{"2024-01-15T10:30", false},
		{"2024-01-15 10:30", false},
		{"not a datetime", true},
		{"2024-01-15", true}, // date only, no time
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			value, err := ResolveFieldValue(field, tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ResolveFieldValue() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if _, ok := value.(string); !ok {
					t.Errorf("expected string, got %T", value)
				}
			}
		})
	}
}

func TestResolveFieldValue_Picklist(t *testing.T) {
	field := &AspectFieldInfo{
		Type: "AspectPicklistField",
		Options: []FieldOptionInfo{
			{ID: "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa", Label: "Open"},
			{ID: "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb", Label: "In Progress"},
			{ID: "cccccccc-cccc-cccc-cccc-cccccccccccc", Label: "Closed"},
		},
	}

	tests := []struct {
		input   string
		want    string
		wantErr bool
	}{
		{"Open", "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa", false},
		{"open", "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa", false}, // case insensitive
		{"OPEN", "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa", false},
		{"In Progress", "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb", false},
		{"cccccccc-cccc-cccc-cccc-cccccccccccc", "cccccccc-cccc-cccc-cccc-cccccccccccc", false}, // UUID directly
		{"fdb890d8-a296-4b8d-b0ec-2656e2fcfb70", "fdb890d8-a296-4b8d-b0ec-2656e2fcfb70", false}, // unknown UUID accepted
		{"Invalid", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			value, err := ResolveFieldValue(field, tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ResolveFieldValue() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if v, ok := value.(string); !ok || v != tt.want {
					t.Errorf("got %v (%T), want %q", value, value, tt.want)
				}
			}
		})
	}
}

func TestResolveFieldValue_MultiPicklist(t *testing.T) {
	field := &AspectFieldInfo{
		Type: "AspectMultiPicklistField",
		Options: []FieldOptionInfo{
			{ID: "11111111-0000-0000-0000-000000000001", Label: "Bug"},
			{ID: "11111111-0000-0000-0000-000000000002", Label: "Feature"},
			{ID: "11111111-0000-0000-0000-000000000003", Label: "Enhancement"},
		},
	}

	tests := []struct {
		input   string
		want    []string
		wantErr bool
	}{
		{"Bug", []string{"11111111-0000-0000-0000-000000000001"}, false},
		{"Bug, Feature", []string{"11111111-0000-0000-0000-000000000001", "11111111-0000-0000-0000-000000000002"}, false},
		{"Bug,Feature,Enhancement", []string{"11111111-0000-0000-0000-000000000001", "11111111-0000-0000-0000-000000000002", "11111111-0000-0000-0000-000000000003"}, false},
		{"bug, feature", []string{"11111111-0000-0000-0000-000000000001", "11111111-0000-0000-0000-000000000002"}, false}, // case insensitive
		{"Invalid", nil, true},
		{"Bug, Invalid", nil, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			value, err := ResolveFieldValue(field, tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ResolveFieldValue() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				v, ok := value.([]string)
				if !ok {
					t.Errorf("expected []string, got %T", value)
					return
				}
				if len(v) != len(tt.want) {
					t.Errorf("got %v, want %v", v, tt.want)
					return
				}
				for i := range v {
					if v[i] != tt.want[i] {
						t.Errorf("got[%d] = %q, want %q", i, v[i], tt.want[i])
					}
				}
			}
		})
	}
}

func TestResolveFieldValue_User(t *testing.T) {
	field := &AspectFieldInfo{Type: "AspectUserField"}

	tests := []struct {
		input   string
		wantErr bool
	}{
		{"fdb890d8-a296-4b8d-b0ec-2656e2fcfb70", false}, // UUID accepted
		{"user@example.com", true},                      // email requires lookup
		{"not-a-uuid", true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			_, err := ResolveFieldValue(field, tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ResolveFieldValue() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestResolveFieldValue_Group(t *testing.T) {
	field := &AspectFieldInfo{Type: "AspectGroupField"}

	tests := []struct {
		input   string
		wantErr bool
	}{
		{"fdb890d8-a296-4b8d-b0ec-2656e2fcfb70", false}, // UUID accepted
		{"Support Team", true},                          // name requires lookup
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			_, err := ResolveFieldValue(field, tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ResolveFieldValue() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestResolveFieldValue_Json(t *testing.T) {
	field := &AspectFieldInfo{Type: "AspectJsonField"}

	tests := []struct {
		input   string
		wantErr bool
	}{
		{`{"key": "value"}`, false},
		{`[1, 2, 3]`, false},
		{`"string"`, false},
		{`123`, false},
		{`true`, false},
		{`null`, false},
		{`{invalid json}`, true},
		{``, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			_, err := ResolveFieldValue(field, tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ResolveFieldValue() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestResolveFieldValue_NilField(t *testing.T) {
	// When field info is nil, value should pass through as-is
	value, err := ResolveFieldValue(nil, "any value")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if value != "any value" {
		t.Errorf("got %v, want %q", value, "any value")
	}
}

func TestResolveFieldValue_UnknownType(t *testing.T) {
	field := &AspectFieldInfo{Type: "AspectUnknownField"}

	// Unknown field types should pass through
	value, err := ResolveFieldValue(field, "some value")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if value != "some value" {
		t.Errorf("got %v, want %q", value, "some value")
	}
}

func TestValidateAttachmentPath(t *testing.T) {
	// Create a temp file for testing
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(tmpFile, []byte("test content"), 0644); err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}

	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{
			name:    "valid file",
			path:    tmpFile,
			wantErr: false,
		},
		{
			name:    "non-existent file",
			path:    filepath.Join(tmpDir, "nonexistent.txt"),
			wantErr: true,
		},
		{
			name:    "directory instead of file",
			path:    tmpDir,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateAttachmentPath(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateAttachmentPath() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestParseJSONInputFile(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name       string
		content    string
		wantFields int
		wantAttach int
		wantErr    bool
	}{
		{
			name:       "simple fields",
			content:    `{"Title": "Test", "Priority": "High"}`,
			wantFields: 2,
			wantAttach: 0,
			wantErr:    false,
		},
		{
			name:       "with attachments",
			content:    `{"Title": "Test", "attachments": [{"path": "./file.pdf"}]}`,
			wantFields: 1,
			wantAttach: 1,
			wantErr:    false,
		},
		{
			name:       "null value skipped in processing",
			content:    `{"Title": "Test", "Optional": null}`,
			wantFields: 2, // null is still parsed, filtering happens later
			wantAttach: 0,
			wantErr:    false,
		},
		{
			name:    "invalid JSON",
			content: `{invalid}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpFile := filepath.Join(tmpDir, tt.name+".json")
			if err := os.WriteFile(tmpFile, []byte(tt.content), 0644); err != nil {
				t.Fatalf("failed to write test file: %v", err)
			}

			result, err := ParseJSONInputFile(tmpFile)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseJSONInputFile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if len(result.Fields) != tt.wantFields {
					t.Errorf("got %d fields, want %d", len(result.Fields), tt.wantFields)
				}
				if len(result.Attachments) != tt.wantAttach {
					t.Errorf("got %d attachments, want %d", len(result.Attachments), tt.wantAttach)
				}
			}
		})
	}
}

func TestParseJSONInputFile_NotFound(t *testing.T) {
	_, err := ParseJSONInputFile("/nonexistent/path/file.json")
	if err == nil {
		t.Error("expected error for non-existent file")
	}
}
