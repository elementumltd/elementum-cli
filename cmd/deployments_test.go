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

package cmd

import (
	"encoding/json"
	"testing"
	"time"
)

func TestDeploymentsParentCommandRegistration(t *testing.T) {
	if deploymentsCmd.Use != "deployments" {
		t.Errorf("unexpected Use: %q, want %q", deploymentsCmd.Use, "deployments")
	}

	subcommands := map[string]bool{
		"list":      false,
		"show":      false,
		"create":    false,
		"configure": false,
	}

	for _, sub := range deploymentsCmd.Commands() {
		if _, ok := subcommands[sub.Name()]; ok {
			subcommands[sub.Name()] = true
		}
	}

	for name, found := range subcommands {
		if !found {
			t.Errorf("missing subcommand: %s", name)
		}
	}
}

func TestGetDeploymentsCmd(t *testing.T) {
	cmd := GetDeploymentsCmd()
	if cmd == nil {
		t.Fatal("GetDeploymentsCmd returned nil")
	}
	if cmd != deploymentsCmd {
		t.Error("GetDeploymentsCmd should return deploymentsCmd")
	}
}

func TestDeploymentsListCommandRegistration(t *testing.T) {
	if deploymentsListCmd.Use != "list" {
		t.Errorf("unexpected Use: %q, want %q", deploymentsListCmd.Use, "list")
	}

	if deploymentsListCmd.RunE == nil {
		t.Error("RunE should not be nil")
	}

	// Flags
	if deploymentsListCmd.Flags().Lookup("app") == nil {
		t.Error("missing flag: app")
	}
	if deploymentsListCmd.Flags().Lookup("limit") == nil {
		t.Error("missing flag: limit")
	}

	// Default values
	limitFlag := deploymentsListCmd.Flags().Lookup("limit")
	if limitFlag.DefValue != "20" {
		t.Errorf("limit default = %q, want %q", limitFlag.DefValue, "20")
	}
}

func TestDeploymentsShowCommandRegistration(t *testing.T) {
	if deploymentsShowCmd.Use != "show <deployment-id>" {
		t.Errorf("unexpected Use: %q, want %q", deploymentsShowCmd.Use, "show <deployment-id>")
	}

	if deploymentsShowCmd.Args == nil {
		t.Error("Args should not be nil")
	}

	if deploymentsShowCmd.RunE == nil {
		t.Error("RunE should not be nil")
	}
}

func TestDeploymentsCreateCommandRegistration(t *testing.T) {
	if deploymentsCreateCmd.Use != "create <app-namespace>" {
		t.Errorf("unexpected Use: %q, want %q", deploymentsCreateCmd.Use, "create <app-namespace>")
	}

	if deploymentsCreateCmd.Args == nil {
		t.Error("Args should not be nil")
	}

	if deploymentsCreateCmd.RunE == nil {
		t.Error("RunE should not be nil")
	}

	// Required flags
	requiredFlags := []string{"source", "target"}
	for _, name := range requiredFlags {
		f := deploymentsCreateCmd.Flags().Lookup(name)
		if f == nil {
			t.Errorf("missing required flag: %s", name)
		}
	}

	// Optional flags
	optionalFlags := []string{"name", "message", "no-wait", "timeout"}
	for _, name := range optionalFlags {
		if deploymentsCreateCmd.Flags().Lookup(name) == nil {
			t.Errorf("missing flag: %s", name)
		}
	}

	// Default values
	timeoutFlag := deploymentsCreateCmd.Flags().Lookup("timeout")
	if timeoutFlag.DefValue != "10m0s" {
		t.Errorf("timeout default = %q, want %q", timeoutFlag.DefValue, "10m0s")
	}
}

func TestDeploymentsConfigureCommandRegistration(t *testing.T) {
	if deploymentsConfigureCmd.Use != "configure <deployment-id>" {
		t.Errorf("unexpected Use: %q, want %q", deploymentsConfigureCmd.Use, "configure <deployment-id>")
	}

	if deploymentsConfigureCmd.Args == nil {
		t.Error("Args should not be nil")
	}

	if deploymentsConfigureCmd.RunE == nil {
		t.Error("RunE should not be nil")
	}

	// Required flags
	if deploymentsConfigureCmd.Flags().Lookup("config") == nil {
		t.Error("missing required flag: config")
	}

	// Optional flags
	if deploymentsConfigureCmd.Flags().Lookup("dry-run") == nil {
		t.Error("missing flag: dry-run")
	}
}

func TestStyledDeploymentStatus(t *testing.T) {
	tests := []struct {
		status string
	}{
		{"COMPLETED"},
		{"FAILED"},
		{"CONFIGURATION"},
		{"PENDING"},
		{"SCOPING"},
		{"EXTRACTING"},
		{"DIFFING"},
	}

	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			result := styledDeploymentStatus(tt.status)
			if result == "" {
				t.Errorf("styledDeploymentStatus(%q) returned empty string", tt.status)
			}
			// The styled string should contain the status text (wrapped in ANSI codes)
			if len(result) < len(tt.status) {
				t.Errorf("styledDeploymentStatus(%q) result too short: %q", tt.status, result)
			}
		})
	}
}

func TestStyledDeploymentStatus_Unknown(t *testing.T) {
	result := styledDeploymentStatus("UNKNOWN_STATUS")
	if result != "UNKNOWN_STATUS" {
		t.Errorf("styledDeploymentStatus(\"UNKNOWN_STATUS\") = %q, want \"UNKNOWN_STATUS\"", result)
	}
}

func TestConfigFileParsing(t *testing.T) {
	tests := []struct {
		name        string
		json        string
		wantErr     bool
		wantCount   int
		checkFields bool
	}{
		{
			name: "valid single dataset",
			json: `{
				"datasets": [{
					"id": "a34340ae-1234-5678-9012-abcdef123456",
					"cloud_link_id": "5eb0b0d8-1234-5678-9012-abcdef123456",
					"snowflake_mapping": {
						"database_name": "MY_DB",
						"schema_name": "MY_SCHEMA",
						"table_name": "MY_TABLE"
					},
					"fields": [
						{"id": "f1", "name": "Title", "type": "TEXT", "column_name": "TITLE", "semantic_tags": ["TITLE"]},
						{"id": "f2", "name": "ID", "type": "TEXT", "column_name": "ID", "semantic_tags": ["HANDLE"]}
					]
				}]
			}`,
			wantCount:   1,
			checkFields: true,
		},
		{
			name: "valid multiple datasets",
			json: `{
				"datasets": [
					{"id": "ds1", "cloud_link_id": "cl1", "fields": []},
					{"id": "ds2", "cloud_link_id": "cl2", "fields": []}
				]
			}`,
			wantCount: 2,
		},
		{
			name:      "empty datasets array",
			json:      `{"datasets": []}`,
			wantCount: 0,
		},
		{
			name:    "invalid json",
			json:    `{not valid json}`,
			wantErr: true,
		},
		{
			name: "dataset without snowflake mapping",
			json: `{
				"datasets": [{
					"id": "ds1",
					"cloud_link_id": "cl1",
					"fields": [{"id": "f1", "name": "Title", "type": "TEXT", "column_name": "TITLE"}]
				}]
			}`,
			wantCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var cfg configFile
			err := json.Unmarshal([]byte(tt.json), &cfg)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if len(cfg.Datasets) != tt.wantCount {
				t.Errorf("dataset count = %d, want %d", len(cfg.Datasets), tt.wantCount)
			}

			if tt.checkFields && tt.wantCount > 0 {
				ds := cfg.Datasets[0]

				if ds.ID != "a34340ae-1234-5678-9012-abcdef123456" {
					t.Errorf("dataset ID = %q, want %q", ds.ID, "a34340ae-1234-5678-9012-abcdef123456")
				}
				if ds.CloudLinkID != "5eb0b0d8-1234-5678-9012-abcdef123456" {
					t.Errorf("cloud_link_id = %q, want %q", ds.CloudLinkID, "5eb0b0d8-1234-5678-9012-abcdef123456")
				}

				if ds.SnowflakeMapping == nil {
					t.Fatal("snowflake_mapping should not be nil")
				}
				if ds.SnowflakeMapping.DatabaseName != "MY_DB" {
					t.Errorf("database_name = %q, want %q", ds.SnowflakeMapping.DatabaseName, "MY_DB")
				}
				if ds.SnowflakeMapping.SchemaName != "MY_SCHEMA" {
					t.Errorf("schema_name = %q, want %q", ds.SnowflakeMapping.SchemaName, "MY_SCHEMA")
				}
				if ds.SnowflakeMapping.TableName != "MY_TABLE" {
					t.Errorf("table_name = %q, want %q", ds.SnowflakeMapping.TableName, "MY_TABLE")
				}

				if len(ds.Fields) != 2 {
					t.Fatalf("field count = %d, want 2", len(ds.Fields))
				}

				// First field
				f := ds.Fields[0]
				if f.ID != "f1" {
					t.Errorf("field[0].id = %q, want %q", f.ID, "f1")
				}
				if f.Name != "Title" {
					t.Errorf("field[0].name = %q, want %q", f.Name, "Title")
				}
				if f.Type != "TEXT" {
					t.Errorf("field[0].type = %q, want %q", f.Type, "TEXT")
				}
				if f.ColumnName != "TITLE" {
					t.Errorf("field[0].column_name = %q, want %q", f.ColumnName, "TITLE")
				}
				if len(f.SemanticTags) != 1 || f.SemanticTags[0] != "TITLE" {
					t.Errorf("field[0].semantic_tags = %v, want [TITLE]", f.SemanticTags)
				}

				// Second field
				f2 := ds.Fields[1]
				if f2.Name != "ID" {
					t.Errorf("field[1].name = %q, want %q", f2.Name, "ID")
				}
				if len(f2.SemanticTags) != 1 || f2.SemanticTags[0] != "HANDLE" {
					t.Errorf("field[1].semantic_tags = %v, want [HANDLE]", f2.SemanticTags)
				}
			}
		})
	}
}

func TestConfigFileFieldTypes(t *testing.T) {
	// Verify all common field types can be deserialized
	fieldTypes := []string{"TEXT", "NUMBER", "DATE", "PICKLIST", "MULTI_PICKLIST", "BOOLEAN", "USER", "GROUP"}

	for _, ft := range fieldTypes {
		t.Run(ft, func(t *testing.T) {
			jsonStr := `{"datasets": [{"id": "ds1", "cloud_link_id": "cl1", "fields": [{"id": "f1", "name": "Test", "type": "` + ft + `", "column_name": "COL"}]}]}`
			var cfg configFile
			if err := json.Unmarshal([]byte(jsonStr), &cfg); err != nil {
				t.Fatalf("failed to parse config with field type %s: %v", ft, err)
			}
			if cfg.Datasets[0].Fields[0].Type != ft {
				t.Errorf("field type = %q, want %q", cfg.Datasets[0].Fields[0].Type, ft)
			}
		})
	}
}

func TestConfigFileSemanticTags(t *testing.T) {
	tests := []struct {
		name string
		tags []string
	}{
		{"no tags", nil},
		{"single tag", []string{"TITLE"}},
		{"multiple tags", []string{"TITLE", "HANDLE"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var f fieldConfig
			f.Name = "Test"
			f.Type = "TEXT"
			f.SemanticTags = tt.tags

			// Round-trip through JSON
			data, err := json.Marshal(f)
			if err != nil {
				t.Fatalf("marshal error: %v", err)
			}

			var f2 fieldConfig
			if err := json.Unmarshal(data, &f2); err != nil {
				t.Fatalf("unmarshal error: %v", err)
			}

			if len(f2.SemanticTags) != len(tt.tags) {
				t.Errorf("semantic_tags length = %d, want %d", len(f2.SemanticTags), len(tt.tags))
			}
			for i, tag := range tt.tags {
				if i < len(f2.SemanticTags) && f2.SemanticTags[i] != tag {
					t.Errorf("semantic_tags[%d] = %q, want %q", i, f2.SemanticTags[i], tag)
				}
			}
		})
	}
}

func TestConfigFileSnowflakeMappingOptional(t *testing.T) {
	// snowflake_mapping should be nil when not provided
	jsonStr := `{"datasets": [{"id": "ds1", "cloud_link_id": "cl1", "fields": []}]}`
	var cfg configFile
	if err := json.Unmarshal([]byte(jsonStr), &cfg); err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if cfg.Datasets[0].SnowflakeMapping != nil {
		t.Error("snowflake_mapping should be nil when not provided")
	}
}

func TestConfigFileJSONRoundTrip(t *testing.T) {
	// Verify the config structures survive JSON round-tripping
	original := configFile{
		Datasets: []datasetConfig{
			{
				ID:          "ds-1",
				CloudLinkID: "cl-1",
				SnowflakeMapping: &snowflakeMappingConfig{
					DatabaseName: "DB",
					SchemaName:   "SCHEMA",
					TableName:    "TABLE",
				},
				Fields: []fieldConfig{
					{
						ID:           "f-1",
						Name:         "Title",
						Type:         "TEXT",
						ColumnName:   "TITLE",
						SemanticTags: []string{"TITLE"},
					},
				},
			},
		},
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var decoded configFile
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if len(decoded.Datasets) != 1 {
		t.Fatalf("dataset count = %d, want 1", len(decoded.Datasets))
	}

	ds := decoded.Datasets[0]
	if ds.ID != "ds-1" {
		t.Errorf("ID = %q, want %q", ds.ID, "ds-1")
	}
	if ds.CloudLinkID != "cl-1" {
		t.Errorf("CloudLinkID = %q, want %q", ds.CloudLinkID, "cl-1")
	}
	if ds.SnowflakeMapping.DatabaseName != "DB" {
		t.Errorf("DatabaseName = %q, want %q", ds.SnowflakeMapping.DatabaseName, "DB")
	}
	if len(ds.Fields) != 1 {
		t.Fatalf("field count = %d, want 1", len(ds.Fields))
	}
	if ds.Fields[0].Name != "Title" {
		t.Errorf("field name = %q, want %q", ds.Fields[0].Name, "Title")
	}
}

func TestFieldTypenameToType(t *testing.T) {
	tests := []struct {
		typename string
		want     string
	}{
		{"AspectTextField", "TEXT"},
		{"AspectNumberField", "NUMBER"},
		{"AspectDecimalField", "DECIMAL"},
		{"AspectBooleanField", "BOOLEAN"},
		{"AspectDateField", "DATE"},
		{"AspectDateTimeField", "DATETIME"},
		{"AspectPicklistField", "PICKLIST"},
		{"AspectMultiPicklistField", "MULTI_PICKLIST"},
		{"AspectUserField", "USER_ID"},
		{"AspectGroupField", "GROUP_ID"},
		{"AspectHtmlField", "HTML"},
		{"AspectJsonField", "JSON"},
		{"AspectAttachmentField", "ATTACHMENT_ID"},
		{"UnknownField", "TEXT"}, // fallback
	}

	for _, tt := range tests {
		t.Run(tt.typename, func(t *testing.T) {
			got := fieldTypenameToType(tt.typename)
			if got != tt.want {
				t.Errorf("fieldTypenameToType(%q) = %q, want %q", tt.typename, got, tt.want)
			}
		})
	}
}

func TestDerefOr(t *testing.T) {
	s := "hello"
	if got := derefOr(&s, "fallback"); got != "hello" {
		t.Errorf("derefOr(&%q, ...) = %q, want %q", s, got, "hello")
	}
	if got := derefOr(nil, "fallback"); got != "fallback" {
		t.Errorf("derefOr(nil, %q) = %q, want %q", "fallback", got, "fallback")
	}
}

func TestDeploymentPollingConstants(t *testing.T) {
	// Verify polling constants have sensible values
	if deploymentPollInterval <= 0 {
		t.Errorf("deploymentPollInterval should be positive, got %v", deploymentPollInterval)
	}
	if deploymentPollInterval > 30*time.Second {
		t.Errorf("deploymentPollInterval should not be too long, got %v", deploymentPollInterval)
	}

	if maxConsecutivePollErrors <= 0 {
		t.Errorf("maxConsecutivePollErrors should be positive, got %d", maxConsecutivePollErrors)
	}
	if maxConsecutivePollErrors > 20 {
		t.Errorf("maxConsecutivePollErrors should not be too high, got %d", maxConsecutivePollErrors)
	}
}

func TestTypenameToFieldTypeMap(t *testing.T) {
	// Verify the map has all expected entries
	expectedMappings := map[string]string{
		"Text":          "TEXT",
		"Number":        "NUMBER",
		"Decimal":       "DECIMAL",
		"Boolean":       "BOOLEAN",
		"Date":          "DATE",
		"DateTime":      "DATETIME",
		"Picklist":      "PICKLIST",
		"MultiPicklist": "MULTI_PICKLIST",
		"User":          "USER_ID",
		"Group":         "GROUP_ID",
		"Html":          "HTML",
		"Json":          "JSON",
		"Attachment":    "ATTACHMENT_ID",
	}

	for typename, expected := range expectedMappings {
		if got, ok := typenameToFieldType[typename]; !ok {
			t.Errorf("typenameToFieldType missing key %q", typename)
		} else if got != expected {
			t.Errorf("typenameToFieldType[%q] = %q, want %q", typename, got, expected)
		}
	}

	// Verify map size matches expected (no extra entries)
	if len(typenameToFieldType) != len(expectedMappings) {
		t.Errorf("typenameToFieldType has %d entries, expected %d", len(typenameToFieldType), len(expectedMappings))
	}
}

func TestBuildConfigTemplate(t *testing.T) {
	t.Run("empty configs", func(t *testing.T) {
		result := buildConfigTemplate(nil)
		if len(result) != 0 {
			t.Errorf("expected empty, got %d datasets", len(result))
		}
	})

	t.Run("skips non-element types", func(t *testing.T) {
		configs := []missingConfig{
			{Typename: "Table", ID: "t1", Name: "Some Table"},
			{Typename: "AiProviderUnconfigured", ID: "ai1", Name: "AI Provider"},
		}
		result := buildConfigTemplate(configs)
		if len(result) != 0 {
			t.Errorf("expected 0 datasets for non-element types, got %d", len(result))
		}
	})

	t.Run("element with cloud link and fields", func(t *testing.T) {
		cloudLinkID := "cl-123"
		dbName := "MY_DB"
		schemaName := "MY_SCHEMA"
		tableName := "MY_TABLE"
		colName := "TITLE_COL"

		configs := []missingConfig{
			{
				Typename: "AspectElement",
				ID:       "elem-1",
				Name:     "Locations",
				CloudLink: &struct {
					ID   string `json:"id"`
					Name string `json:"name"`
				}{ID: cloudLinkID, Name: "My CloudLink"},
				CloudConnection: &struct {
					Typename     string  `json:"__typename"`
					DatabaseName *string `json:"databaseName"`
					SchemaName   *string `json:"schemaName"`
					TableName    *string `json:"tableName"`
				}{
					Typename:     "SnowflakeConnection",
					DatabaseName: &dbName,
					SchemaName:   &schemaName,
					TableName:    &tableName,
				},
				Fields: &struct {
					Edges []struct {
						Node struct {
							ID                   string   `json:"id"`
							Name                 string   `json:"name"`
							Typename             string   `json:"__typename"`
							SemanticTags         []string `json:"semanticTags"`
							System               bool     `json:"system"`
							CloudFieldConnection *struct {
								ColumnName *string `json:"columnName"`
							} `json:"cloudFieldConnection"`
						} `json:"node"`
					} `json:"edges"`
				}{
					Edges: []struct {
						Node struct {
							ID                   string   `json:"id"`
							Name                 string   `json:"name"`
							Typename             string   `json:"__typename"`
							SemanticTags         []string `json:"semanticTags"`
							System               bool     `json:"system"`
							CloudFieldConnection *struct {
								ColumnName *string `json:"columnName"`
							} `json:"cloudFieldConnection"`
						} `json:"node"`
					}{
						{Node: struct {
							ID                   string   `json:"id"`
							Name                 string   `json:"name"`
							Typename             string   `json:"__typename"`
							SemanticTags         []string `json:"semanticTags"`
							System               bool     `json:"system"`
							CloudFieldConnection *struct {
								ColumnName *string `json:"columnName"`
							} `json:"cloudFieldConnection"`
						}{
							ID: "f1", Name: "Title", Typename: "AspectTextField",
							SemanticTags: []string{"TITLE"},
							CloudFieldConnection: &struct {
								ColumnName *string `json:"columnName"`
							}{ColumnName: &colName},
						}},
						{Node: struct {
							ID                   string   `json:"id"`
							Name                 string   `json:"name"`
							Typename             string   `json:"__typename"`
							SemanticTags         []string `json:"semanticTags"`
							System               bool     `json:"system"`
							CloudFieldConnection *struct {
								ColumnName *string `json:"columnName"`
							} `json:"cloudFieldConnection"`
						}{
							ID: "sys1", Name: "Created At", Typename: "AspectDateTimeField",
							System: true, // should be skipped
						}},
					},
				},
			},
		}

		result := buildConfigTemplate(configs)
		if len(result) != 1 {
			t.Fatalf("expected 1 dataset, got %d", len(result))
		}

		ds := result[0]
		if ds.ID != "elem-1" {
			t.Errorf("dataset ID = %q, want %q", ds.ID, "elem-1")
		}
		if ds.CloudLinkID != cloudLinkID {
			t.Errorf("cloud_link_id = %q, want %q", ds.CloudLinkID, cloudLinkID)
		}
		if ds.SnowflakeMapping == nil {
			t.Fatal("snowflake_mapping should not be nil")
		}
		if ds.SnowflakeMapping.DatabaseName != "MY_DB" {
			t.Errorf("database = %q, want %q", ds.SnowflakeMapping.DatabaseName, "MY_DB")
		}
		// Should have 1 field (system field skipped)
		if len(ds.Fields) != 1 {
			t.Fatalf("field count = %d, want 1 (system field should be skipped)", len(ds.Fields))
		}
		f := ds.Fields[0]
		if f.ID != "f1" {
			t.Errorf("field ID = %q, want %q", f.ID, "f1")
		}
		if f.Type != "TEXT" {
			t.Errorf("field type = %q, want %q", f.Type, "TEXT")
		}
		if f.ColumnName != "TITLE_COL" {
			t.Errorf("column_name = %q, want %q", f.ColumnName, "TITLE_COL")
		}
		if len(f.SemanticTags) != 1 || f.SemanticTags[0] != "TITLE" {
			t.Errorf("semantic_tags = %v, want [TITLE]", f.SemanticTags)
		}
	})

	t.Run("element without cloud connection uses placeholders", func(t *testing.T) {
		configs := []missingConfig{
			{
				Typename: "AspectElement",
				ID:       "elem-2",
				Name:     "No Connection",
			},
		}
		result := buildConfigTemplate(configs)
		if len(result) != 1 {
			t.Fatalf("expected 1 dataset, got %d", len(result))
		}
		ds := result[0]
		if ds.CloudLinkID != "<CLOUD_LINK_ID>" {
			t.Errorf("cloud_link_id = %q, want placeholder", ds.CloudLinkID)
		}
		if ds.SnowflakeMapping.DatabaseName != "<DATABASE_NAME>" {
			t.Errorf("database = %q, want placeholder", ds.SnowflakeMapping.DatabaseName)
		}
	})
}
