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
	"encoding/csv"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDetectFileFormat(t *testing.T) {
	t.Parallel()

	tests := []struct {
		path    string
		want    FileFormat
		wantErr bool
	}{
		{"data.csv", FormatCSV, false},
		{"DATA.CSV", FormatCSV, false},
		{"data.tsv", FormatCSV, false},
		{"records.json", FormatJSON, false},
		{"stream.jsonl", FormatJSONL, false},
		{"stream.ndjson", FormatJSONL, false},
		{"file.xlsx", "", true},
		{"file.xml", "", true},
		{"file", "", true},
	}

	for _, tc := range tests {
		t.Run(tc.path, func(t *testing.T) {
			got, err := DetectFileFormat(tc.path)
			if (err != nil) != tc.wantErr {
				t.Errorf("DetectFileFormat(%q) error = %v, wantErr %v", tc.path, err, tc.wantErr)
				return
			}
			if got != tc.want {
				t.Errorf("DetectFileFormat(%q) = %q, want %q", tc.path, got, tc.want)
			}
		})
	}
}

func TestLoadRecordsFromCSV(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		content   string
		wantCount int
		wantFirst map[string]interface{}
		wantErr   bool
	}{
		{
			name:      "basic CSV",
			content:   "Title,Status,Priority\nBug Report,Open,High\nFeature Request,New,Medium\n",
			wantCount: 2,
			wantFirst: map[string]interface{}{"Title": "Bug Report", "Status": "Open", "Priority": "High"},
		},
		{
			name:      "CSV with empty values",
			content:   "Title,Status,Priority\nBug Report,,High\n",
			wantCount: 1,
			wantFirst: map[string]interface{}{"Title": "Bug Report", "Priority": "High"},
		},
		{
			name:      "CSV with quoted fields",
			content:   "Title,Description\n\"Bug Report\",\"Has a, comma\"\n",
			wantCount: 1,
			wantFirst: map[string]interface{}{"Title": "Bug Report", "Description": "Has a, comma"},
		},
		{
			name:      "CSV with BOM",
			content:   "\xef\xbb\xbfTitle,Status\nTest,Open\n",
			wantCount: 1,
			wantFirst: map[string]interface{}{"Title": "Test", "Status": "Open"},
		},
		{
			name:      "empty CSV",
			content:   "Title,Status\n",
			wantErr:   false,
			wantCount: 0,
		},
		{
			name:      "header only",
			content:   "Title,Status",
			wantErr:   false,
			wantCount: 0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tmpFile := filepath.Join(t.TempDir(), "test.csv")
			if err := os.WriteFile(tmpFile, []byte(tc.content), 0644); err != nil {
				t.Fatalf("failed to write temp file: %v", err)
			}

			records, err := LoadRecordsFromFile(tmpFile, FormatCSV)
			if (err != nil) != tc.wantErr {
				t.Fatalf("LoadRecordsFromFile() error = %v, wantErr %v", err, tc.wantErr)
			}
			if tc.wantErr {
				return
			}

			if len(records) != tc.wantCount {
				t.Fatalf("got %d records, want %d", len(records), tc.wantCount)
			}

			if tc.wantCount > 0 && tc.wantFirst != nil {
				for key, want := range tc.wantFirst {
					got, ok := records[0][key]
					if !ok {
						t.Errorf("first record missing key %q", key)
						continue
					}
					if got != want {
						t.Errorf("first record[%q] = %v, want %v", key, got, want)
					}
				}
			}
		})
	}
}

func TestLoadRecordsFromTSV(t *testing.T) {
	t.Parallel()

	content := "Title\tStatus\tPriority\nBug Report\tOpen\tHigh\n"
	tmpFile := filepath.Join(t.TempDir(), "test.tsv")
	if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	records, err := LoadRecordsFromFile(tmpFile, FormatCSV)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(records) != 1 {
		t.Fatalf("got %d records, want 1", len(records))
	}
	if records[0]["Title"] != "Bug Report" {
		t.Errorf("Title = %v, want 'Bug Report'", records[0]["Title"])
	}
}

func TestLoadRecordsFromJSON_Array(t *testing.T) {
	t.Parallel()

	content := `[
		{"Title": "Bug Report", "Status": "Open"},
		{"Title": "Feature Request", "Status": "New"}
	]`
	tmpFile := filepath.Join(t.TempDir(), "test.json")
	if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	records, err := LoadRecordsFromFile(tmpFile, FormatJSON)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(records) != 2 {
		t.Fatalf("got %d records, want 2", len(records))
	}
	if records[0]["Title"] != "Bug Report" {
		t.Errorf("first record Title = %v, want 'Bug Report'", records[0]["Title"])
	}
}

func TestLoadRecordsFromJSON_Wrapper(t *testing.T) {
	t.Parallel()

	content := `{"records": [{"Title": "Test", "Status": "Open"}]}`
	tmpFile := filepath.Join(t.TempDir(), "test.json")
	if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	records, err := LoadRecordsFromFile(tmpFile, FormatJSON)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(records) != 1 {
		t.Fatalf("got %d records, want 1", len(records))
	}
}

func TestLoadRecordsFromJSON_SingleObject(t *testing.T) {
	t.Parallel()

	content := `{"Title": "Single Record", "Status": "Open"}`
	tmpFile := filepath.Join(t.TempDir(), "test.json")
	if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	records, err := LoadRecordsFromFile(tmpFile, FormatJSON)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(records) != 1 {
		t.Fatalf("got %d records, want 1", len(records))
	}
	if records[0]["Title"] != "Single Record" {
		t.Errorf("Title = %v, want 'Single Record'", records[0]["Title"])
	}
}

func TestLoadRecordsFromJSON_Invalid(t *testing.T) {
	t.Parallel()

	content := `not valid json`
	tmpFile := filepath.Join(t.TempDir(), "test.json")
	if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	_, err := LoadRecordsFromFile(tmpFile, FormatJSON)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestLoadRecordsFromJSONL(t *testing.T) {
	t.Parallel()

	content := `{"Title": "Record 1", "Status": "Open"}
{"Title": "Record 2", "Status": "Closed"}
{"Title": "Record 3", "Status": "New"}
`
	tmpFile := filepath.Join(t.TempDir(), "test.jsonl")
	if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	records, err := LoadRecordsFromFile(tmpFile, FormatJSONL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(records) != 3 {
		t.Fatalf("got %d records, want 3", len(records))
	}
	if records[2]["Title"] != "Record 3" {
		t.Errorf("third record Title = %v, want 'Record 3'", records[2]["Title"])
	}
}

func TestLoadRecordsFromJSONL_Empty(t *testing.T) {
	t.Parallel()

	tmpFile := filepath.Join(t.TempDir(), "test.jsonl")
	if err := os.WriteFile(tmpFile, []byte(""), 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	records, err := LoadRecordsFromFile(tmpFile, FormatJSONL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(records) != 0 {
		t.Fatalf("got %d records, want 0", len(records))
	}
}

func TestLoadRecordsFromFile_NonExistent(t *testing.T) {
	t.Parallel()

	_, err := LoadRecordsFromFile("/nonexistent/path.csv", FormatCSV)
	if err == nil {
		t.Fatal("expected error for non-existent file")
	}
}

func TestExportRecordsToCSV(t *testing.T) {
	t.Parallel()

	records := []Record{
		{
			Handle:    "TKT-1",
			Title:     "Bug Report",
			Status:    "Open",
			CreatedAt: "2024-01-15",
			Data:      map[string]interface{}{"Priority": "High", "Description": "Login broken"},
		},
		{
			Handle:    "TKT-2",
			Title:     "Feature Request",
			Status:    "New",
			CreatedAt: "2024-01-16",
			Data:      map[string]interface{}{"Priority": "Medium"},
		},
	}

	outPath := filepath.Join(t.TempDir(), "export.csv")
	err := ExportRecordsToFile(records, outPath, FormatCSV)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Read and verify
	f, _ := os.Open(outPath)
	defer func() { _ = f.Close() }()
	reader := csv.NewReader(f)
	rows, err := reader.ReadAll()
	if err != nil {
		t.Fatalf("failed to read CSV: %v", err)
	}

	// Header + 2 data rows
	if len(rows) != 3 {
		t.Fatalf("got %d rows, want 3", len(rows))
	}

	// First 4 columns should be Handle, Title, Status, Created At
	header := rows[0]
	if header[0] != "Handle" || header[1] != "Title" || header[2] != "Status" || header[3] != "Created At" {
		t.Errorf("unexpected header: %v", header)
	}

	// First data row
	if rows[1][0] != "TKT-1" || rows[1][1] != "Bug Report" {
		t.Errorf("unexpected first row: %v", rows[1])
	}
}

func TestExportRecordsToJSON(t *testing.T) {
	t.Parallel()

	records := []Record{
		{
			ID:     "id-1",
			Handle: "TKT-1",
			Title:  "Test",
			Data:   map[string]interface{}{"Priority": "High"},
		},
	}

	outPath := filepath.Join(t.TempDir(), "export.json")
	err := ExportRecordsToFile(records, outPath, FormatJSON)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	content, _ := os.ReadFile(outPath)
	var parsed []Record
	if err := json.Unmarshal(content, &parsed); err != nil {
		t.Fatalf("failed to parse exported JSON: %v", err)
	}
	if len(parsed) != 1 {
		t.Fatalf("got %d records, want 1", len(parsed))
	}
	if parsed[0].Handle != "TKT-1" {
		t.Errorf("Handle = %q, want 'TKT-1'", parsed[0].Handle)
	}
}

func TestExportRecordsToJSONL(t *testing.T) {
	t.Parallel()

	records := []Record{
		{Handle: "TKT-1", Title: "First", Data: map[string]interface{}{}},
		{Handle: "TKT-2", Title: "Second", Data: map[string]interface{}{}},
	}

	outPath := filepath.Join(t.TempDir(), "export.jsonl")
	err := ExportRecordsToFile(records, outPath, FormatJSONL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	content, _ := os.ReadFile(outPath)
	lines := strings.Split(strings.TrimSpace(string(content)), "\n")
	if len(lines) != 2 {
		t.Fatalf("got %d lines, want 2", len(lines))
	}

	var rec Record
	if err := json.Unmarshal([]byte(lines[0]), &rec); err != nil {
		t.Fatalf("failed to parse first line: %v", err)
	}
	if rec.Handle != "TKT-1" {
		t.Errorf("first line Handle = %q, want 'TKT-1'", rec.Handle)
	}
}

func TestExportRecordsToCSV_Empty(t *testing.T) {
	t.Parallel()

	outPath := filepath.Join(t.TempDir(), "empty.csv")
	err := ExportRecordsToFile([]Record{}, outPath, FormatCSV)
	if err == nil {
		t.Fatal("expected error for empty records")
	}
}

func TestBuildFieldNameMap(t *testing.T) {
	t.Parallel()

	fields := []AspectFieldInfo{
		{ID: "id-1", Name: "Title"},
		{ID: "id-2", Name: "Status"},
		{ID: "id-3", Name: "Priority"},
	}

	m := buildFieldNameMap(fields)

	if m["title"] != "id-1" {
		t.Errorf("title -> %q, want 'id-1'", m["title"])
	}
	if m["status"] != "id-2" {
		t.Errorf("status -> %q, want 'id-2'", m["status"])
	}
}

func TestResolveFieldKey(t *testing.T) {
	t.Parallel()

	nameToID := map[string]string{
		"title":  "id-1",
		"status": "id-2",
	}

	tests := []struct {
		key  string
		want string
	}{
		{"Title", "id-1"},
		{"title", "id-1"},
		{"STATUS", "id-2"},
		{"11111111-1111-1111-1111-111111111111", "11111111-1111-1111-1111-111111111111"}, // UUID passthrough
		{"unknown", "unknown"}, // unknown key passthrough
	}

	for _, tc := range tests {
		t.Run(tc.key, func(t *testing.T) {
			got := resolveFieldKey(tc.key, nameToID)
			if got != tc.want {
				t.Errorf("resolveFieldKey(%q) = %q, want %q", tc.key, got, tc.want)
			}
		})
	}
}

func TestCollectFieldIDs(t *testing.T) {
	t.Parallel()

	nameToID := map[string]string{
		"title":    "id-1",
		"status":   "id-2",
		"priority": "id-3",
	}

	records := []map[string]interface{}{
		{"Title": "A", "Status": "Open"},
		{"Title": "B", "Priority": "High"},
	}

	ids := collectFieldIDs(records, nameToID)

	if len(ids) != 3 {
		t.Fatalf("got %d IDs, want 3", len(ids))
	}

	// All three should be present
	idSet := make(map[string]bool)
	for _, id := range ids {
		idSet[id] = true
	}
	for _, expected := range []string{"id-1", "id-2", "id-3"} {
		if !idSet[expected] {
			t.Errorf("missing expected ID %q", expected)
		}
	}
}

func TestCollectAllFieldNames(t *testing.T) {
	t.Parallel()

	records := []Record{
		{Data: map[string]interface{}{"A": "1", "B": "2"}},
		{Data: map[string]interface{}{"B": "3", "C": "4"}},
	}

	names := collectAllFieldNames(records)
	if len(names) != 3 {
		t.Fatalf("got %d names, want 3", len(names))
	}

	nameSet := make(map[string]bool)
	for _, n := range names {
		nameSet[n] = true
	}
	for _, expected := range []string{"A", "B", "C"} {
		if !nameSet[expected] {
			t.Errorf("missing expected name %q", expected)
		}
	}
}

func TestCSVRoundTrip(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	// Create a CSV
	inputPath := filepath.Join(tmpDir, "input.csv")
	content := "Title,Status,Priority\nBug Report,Open,High\nFeature Request,New,Medium\n"
	if err := os.WriteFile(inputPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	// Load it
	records, err := LoadRecordsFromFile(inputPath, FormatCSV)
	if err != nil {
		t.Fatal(err)
	}

	// Convert to Record type and export
	var exportRecords []Record
	for _, rec := range records {
		r := Record{Data: rec}
		if title, ok := rec["Title"].(string); ok {
			r.Title = title
		}
		if status, ok := rec["Status"].(string); ok {
			r.Status = status
		}
		exportRecords = append(exportRecords, r)
	}

	outputPath := filepath.Join(tmpDir, "output.csv")
	if err := ExportRecordsToFile(exportRecords, outputPath, FormatCSV); err != nil {
		t.Fatal(err)
	}

	// Reload and verify
	reloaded, err := LoadRecordsFromFile(outputPath, FormatCSV)
	if err != nil {
		t.Fatal(err)
	}

	if len(reloaded) != len(records) {
		t.Fatalf("round trip: got %d records, want %d", len(reloaded), len(records))
	}
}

func TestJSONRoundTrip(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	records := []Record{
		{
			Handle: "TKT-1",
			Title:  "Test Record",
			Status: "Open",
			Data:   map[string]interface{}{"Priority": "High"},
		},
	}

	path := filepath.Join(tmpDir, "data.json")
	if err := ExportRecordsToFile(records, path, FormatJSON); err != nil {
		t.Fatal(err)
	}

	loaded, err := LoadRecordsFromFile(path, FormatJSON)
	if err != nil {
		t.Fatal(err)
	}

	if len(loaded) != 1 {
		t.Fatalf("got %d records, want 1", len(loaded))
	}
	if loaded[0]["Handle"] != "TKT-1" {
		t.Errorf("Handle = %v, want 'TKT-1'", loaded[0]["Handle"])
	}
}
