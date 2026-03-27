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

	"github.com/elementumltd/elementum-cli/discovery"
)

func TestBeautifyDatamineConfig(t *testing.T) {
	tests := []struct {
		name             string
		hcl              string
		imports          []ImportBlock
		datamine         *discovery.Datamine
		table            *discovery.Table
		relatedResources []discovery.RelatedResource
		wantContains     []string
		wantNotContains  []string
	}{
		{
			name: "replaces table_id UUID with reference",
			hcl: `resource "elementum_datamine" "test" {
  table_id = "11111111-2222-3333-4444-555555555555"
  name     = "Test Datamine"
}`,
			imports: []ImportBlock{
				{
					ID:           "11111111-2222-3333-4444-555555555555:66666666-7777-8888-9999-aaaaaaaaaaaa",
					ResourceType: "elementum_datamine",
					ResourceName: "test",
				},
			},
			datamine: &discovery.Datamine{
				ID:      "66666666-7777-8888-9999-aaaaaaaaaaaa",
				Name:    "Test Datamine",
				TableID: "11111111-2222-3333-4444-555555555555",
			},
			table: &discovery.Table{
				ID:   "11111111-2222-3333-4444-555555555555",
				Name: "Test Table",
			},
			wantContains:    []string{"elementum_table.test_table.id"},
			wantNotContains: []string{"11111111-2222-3333-4444-555555555555"},
		},
		{
			name: "strips null attributes",
			hcl: `resource "elementum_datamine" "test" {
  table_id    = "11111111-2222-3333-4444-555555555555"
  name        = "Test Datamine"
  description = null
  limit       = null
}`,
			imports: []ImportBlock{},
			datamine: &discovery.Datamine{
				ID: "test-id",
			},
			table:           nil,
			wantContains:    []string{"table_id", "name"},
			wantNotContains: []string{"description = null", "limit = null"},
		},
		{
			name: "beautifies primary_column_ids array",
			hcl: `resource "elementum_datamine" "test" {
  primary_column_ids = ["aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee", "11111111-2222-3333-4444-555555555555"]
}`,
			imports: []ImportBlock{},
			datamine: &discovery.Datamine{
				ID: "test-id",
			},
			table: nil,
			relatedResources: []discovery.RelatedResource{
				{
					ID:           "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee",
					Name:         "Field One",
					ResourceType: "data.elementum_field",
				},
			},
			wantContains:    []string{"primary_column_ids", "data.elementum_field.field_one.id"},
			wantNotContains: []string{"aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := BeautifyDatamineConfig(tt.hcl, tt.imports, tt.datamine, tt.table, tt.relatedResources)

			for _, want := range tt.wantContains {
				if !strings.Contains(result, want) {
					t.Errorf("expected result to contain %q, got:\n%s", want, result)
				}
			}

			for _, notWant := range tt.wantNotContains {
				if strings.Contains(result, notWant) {
					t.Errorf("expected result NOT to contain %q, got:\n%s", notWant, result)
				}
			}
		})
	}
}

func TestBuildDatamineUUIDMap(t *testing.T) {
	tests := []struct {
		name             string
		imports          []ImportBlock
		datamine         *discovery.Datamine
		table            *discovery.Table
		relatedResources []discovery.RelatedResource
		wantMappings     map[string]string
	}{
		{
			name: "creates mapping from datamine import",
			imports: []ImportBlock{
				{
					ID:           "table-123:dm-456",
					ResourceType: "elementum_datamine",
					ResourceName: "my_datamine",
				},
			},
			datamine: &discovery.Datamine{
				ID:   "dm-456",
				Name: "My Datamine",
			},
			wantMappings: map[string]string{
				"dm-456": "elementum_datamine.my_datamine.id",
			},
		},
		{
			name:    "creates mapping from table",
			imports: []ImportBlock{},
			datamine: &discovery.Datamine{
				ID: "dm-123",
			},
			table: &discovery.Table{
				ID:   "table-123",
				Name: "Sales Data",
			},
			wantMappings: map[string]string{
				"table-123": "elementum_table.sales_data.id",
			},
		},
		{
			name:    "creates mapping from related resources",
			imports: []ImportBlock{},
			datamine: &discovery.Datamine{
				ID: "dm-123",
			},
			relatedResources: []discovery.RelatedResource{
				{
					ID:           "app-456",
					Name:         "My App",
					ResourceType: "elementum_app",
				},
				{
					ID:           "cl-789",
					Name:         "Data Source",
					ResourceType: "elementum_cloudlink",
				},
			},
			wantMappings: map[string]string{
				"app-456": "elementum_app.my_app.id",
				"cl-789":  "data.elementum_cloudlink.data_source.id",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildDatamineUUIDMap(tt.imports, tt.datamine, tt.table, tt.relatedResources)

			for uuid, expectedRef := range tt.wantMappings {
				if actualRef, ok := result[uuid]; !ok {
					t.Errorf("expected mapping for UUID %q, but not found", uuid)
				} else if actualRef != expectedRef {
					t.Errorf("expected mapping %q -> %q, got %q", uuid, expectedRef, actualRef)
				}
			}
		})
	}
}

func TestBeautifyPrimaryColumnIDArrays(t *testing.T) {
	uuidMap := map[string]string{
		"11111111-2222-3333-4444-555555555555": "elementum_text_field.field_one.id",
		"aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee": "elementum_number_field.field_two.id",
	}

	tests := []struct {
		name    string
		hcl     string
		want    string
		wantNot string
	}{
		{
			name:    "replaces single UUID in array",
			hcl:     `primary_column_ids = ["11111111-2222-3333-4444-555555555555"]`,
			want:    "elementum_text_field.field_one.id",
			wantNot: "11111111-2222-3333-4444-555555555555",
		},
		{
			name:    "replaces multiple UUIDs in array",
			hcl:     `primary_column_ids = ["11111111-2222-3333-4444-555555555555", "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"]`,
			want:    "elementum_number_field.field_two.id",
			wantNot: "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee",
		},
		{
			name: "keeps unknown UUIDs",
			hcl:  `primary_column_ids = ["99999999-8888-7777-6666-555555555555"]`,
			want: "99999999-8888-7777-6666-555555555555",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := beautifyPrimaryColumnIDArrays(tt.hcl, uuidMap)

			if !strings.Contains(result, tt.want) {
				t.Errorf("expected result to contain %q, got:\n%s", tt.want, result)
			}

			if tt.wantNot != "" && strings.Contains(result, tt.wantNot) {
				t.Errorf("expected result NOT to contain %q, got:\n%s", tt.wantNot, result)
			}
		})
	}
}

func TestDatamineImportIDFormat(t *testing.T) {
	// Verify the import ID format is correct
	// Datamine uses ImportStatePassthroughID, so the format is just the datamine_id
	expectedFormat := "{datamine_id}"
	actualFormat := ImportIDFormats["elementum_datamine"]

	if actualFormat != expectedFormat {
		t.Errorf("expected import ID format %q, got %q", expectedFormat, actualFormat)
	}
}

func TestBuildDatamineImportID(t *testing.T) {
	datamineID := "dm-uuid-456"

	result := BuildImportID("elementum_datamine", map[string]string{
		"datamine_id": datamineID,
	})

	// Datamine uses ImportStatePassthroughID, so just the datamine_id
	expected := datamineID
	if result != expected {
		t.Errorf("expected import ID %q, got %q", expected, result)
	}
}
