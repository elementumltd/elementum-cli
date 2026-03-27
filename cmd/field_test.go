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
	"testing"

	"github.com/elementumltd/elementum-cli/internal/client"
)

func TestFieldCommandRegistration(t *testing.T) {
	if fieldsCmd.Use != "fields" {
		t.Errorf("unexpected Use: %q, want %q", fieldsCmd.Use, "fields")
	}

	subcommands := map[string]bool{
		"values": false,
		"list":   false,
		"create": false,
		"delete": false,
		"update": false,
	}

	for _, sub := range fieldsCmd.Commands() {
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

func TestFieldCreateCommandRegistration(t *testing.T) {
	if fieldCreateCmd.Use != "create <namespace>" {
		t.Errorf("unexpected Use: %q", fieldCreateCmd.Use)
	}

	if fieldCreateCmd.Args == nil {
		t.Error("Args should not be nil")
	}

	requiredFlags := []string{"name", "type"}
	for _, name := range requiredFlags {
		if fieldCreateCmd.Flags().Lookup(name) == nil {
			t.Errorf("missing required flag: %s", name)
		}
	}

	optionalFlags := []string{"options", "description", "required", "show-on-create", "dry-run"}
	for _, name := range optionalFlags {
		if fieldCreateCmd.Flags().Lookup(name) == nil {
			t.Errorf("missing flag: %s", name)
		}
	}
}

func TestFieldDeleteCommandRegistration(t *testing.T) {
	if fieldDeleteCmd.Use != "delete <namespace> <field-name>" {
		t.Errorf("unexpected Use: %q", fieldDeleteCmd.Use)
	}

	flags := []string{"force", "dry-run"}
	for _, name := range flags {
		if fieldDeleteCmd.Flags().Lookup(name) == nil {
			t.Errorf("missing flag: %s", name)
		}
	}

	if fieldDeleteCmd.Flags().ShorthandLookup("f") == nil {
		t.Error("missing short flag -f for --force")
	}
}

func TestFieldUpdateCommandRegistration(t *testing.T) {
	if fieldUpdateCmd.Use != "update <namespace> <field-name>" {
		t.Errorf("unexpected Use: %q", fieldUpdateCmd.Use)
	}

	flags := []string{"name", "description", "required", "no-required", "show-on-create", "no-show-on-create"}
	for _, name := range flags {
		if fieldUpdateCmd.Flags().Lookup(name) == nil {
			t.Errorf("missing flag: %s", name)
		}
	}
}

func TestFieldListCommandRegistration(t *testing.T) {
	if fieldListCmd.Use != "list <namespace>" {
		t.Errorf("unexpected Use: %q", fieldListCmd.Use)
	}

	if fieldListCmd.Flags().Lookup("all") == nil {
		t.Error("missing flag: all")
	}
}

func TestBuildFieldCreateInput_Text(t *testing.T) {
	input, err := buildFieldCreateInput("text", "Title", "A title field", true, false, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if input.Text == nil {
		t.Fatal("expected Text to be set")
	}
	if input.Text.Name != "Title" {
		t.Errorf("Name = %q, want %q", input.Text.Name, "Title")
	}
	if input.Text.Description == nil || *input.Text.Description != "A title field" {
		t.Error("Description not set correctly")
	}
	if input.Text.Required == nil || *input.Text.Required != true {
		t.Error("Required not set to true")
	}
	if input.Text.ShowOnCreate != nil {
		t.Error("ShowOnCreate should be nil when false")
	}
}

func TestBuildFieldCreateInput_Number(t *testing.T) {
	input, err := buildFieldCreateInput("number", "Score", "", false, true, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if input.Number == nil {
		t.Fatal("expected Number to be set")
	}
	if input.Number.Name != "Score" {
		t.Errorf("Name = %q, want %q", input.Number.Name, "Score")
	}
	if input.Number.Description != nil {
		t.Error("Description should be nil for empty string")
	}
	if input.Number.ShowOnCreate == nil || *input.Number.ShowOnCreate != true {
		t.Error("ShowOnCreate not set to true")
	}
}

func TestBuildFieldCreateInput_Boolean(t *testing.T) {
	input, err := buildFieldCreateInput("boolean", "Active", "", false, false, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if input.Bool == nil {
		t.Fatal("expected Bool to be set")
	}
	if input.Bool.Name != "Active" {
		t.Errorf("Name = %q, want %q", input.Bool.Name, "Active")
	}
}

func TestBuildFieldCreateInput_Date(t *testing.T) {
	input, err := buildFieldCreateInput("date", "Due Date", "", false, false, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if input.Date == nil {
		t.Fatal("expected Date to be set")
	}
	if input.Date.Name != "Due Date" {
		t.Errorf("Name = %q, want %q", input.Date.Name, "Due Date")
	}
}

func TestBuildFieldCreateInput_DateTime(t *testing.T) {
	input, err := buildFieldCreateInput("datetime", "Created At", "", false, false, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if input.DateTime == nil {
		t.Fatal("expected DateTime to be set")
	}
	if input.DateTime.Name != "Created At" {
		t.Errorf("Name = %q, want %q", input.DateTime.Name, "Created At")
	}
}

func TestBuildFieldCreateInput_Dropdown(t *testing.T) {
	input, err := buildFieldCreateInput("dropdown", "Priority", "", true, false, []string{"Low", "Medium", "High"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if input.Picklist == nil {
		t.Fatal("expected Picklist to be set")
	}
	if input.Picklist.Name != "Priority" {
		t.Errorf("Name = %q, want %q", input.Picklist.Name, "Priority")
	}
	if len(input.Picklist.PicklistValues) != 3 {
		t.Fatalf("PicklistValues count = %d, want 3", len(input.Picklist.PicklistValues))
	}
	expectedValues := []string{"Low", "Medium", "High"}
	for i, pv := range input.Picklist.PicklistValues {
		if pv.Value != expectedValues[i] {
			t.Errorf("PicklistValues[%d].Value = %q, want %q", i, pv.Value, expectedValues[i])
		}
	}
}

func TestBuildFieldCreateInput_Multiselect(t *testing.T) {
	input, err := buildFieldCreateInput("multiselect", "Tags", "", false, false, []string{"Bug", "Feature"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if input.MultiPicklist == nil {
		t.Fatal("expected MultiPicklist to be set")
	}
	if input.MultiPicklist.Name != "Tags" {
		t.Errorf("Name = %q, want %q", input.MultiPicklist.Name, "Tags")
	}
	if len(input.MultiPicklist.PicklistValues) != 2 {
		t.Fatalf("PicklistValues count = %d, want 2", len(input.MultiPicklist.PicklistValues))
	}
}

func TestBuildFieldCreateInput_User(t *testing.T) {
	input, err := buildFieldCreateInput("user", "Assigned To", "", false, false, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if input.User == nil {
		t.Fatal("expected User to be set")
	}
	if input.User.Name != "Assigned To" {
		t.Errorf("Name = %q, want %q", input.User.Name, "Assigned To")
	}
}

func TestBuildFieldCreateInput_Group(t *testing.T) {
	input, err := buildFieldCreateInput("group", "Team", "", false, false, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if input.Group == nil {
		t.Fatal("expected Group to be set")
	}
	if input.Group.Name != "Team" {
		t.Errorf("Name = %q, want %q", input.Group.Name, "Team")
	}
}

func TestBuildFieldCreateInput_Longtext(t *testing.T) {
	input, err := buildFieldCreateInput("longtext", "Notes", "Rich text notes", false, false, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if input.Html == nil {
		t.Fatal("expected Html to be set")
	}
	if input.Html.Name != "Notes" {
		t.Errorf("Name = %q, want %q", input.Html.Name, "Notes")
	}
	if input.Html.Description == nil || *input.Html.Description != "Rich text notes" {
		t.Error("Description not set correctly")
	}
}

func TestBuildFieldCreateInput_Attachment(t *testing.T) {
	input, err := buildFieldCreateInput("attachment", "Files", "", false, false, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if input.Attachment == nil {
		t.Fatal("expected Attachment to be set")
	}
	if input.Attachment.Name != "Files" {
		t.Errorf("Name = %q, want %q", input.Attachment.Name, "Files")
	}
}

func TestBuildFieldCreateInput_JSON(t *testing.T) {
	input, err := buildFieldCreateInput("json", "Metadata", "", false, false, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if input.Json == nil {
		t.Fatal("expected Json to be set")
	}
	if input.Json.Name != "Metadata" {
		t.Errorf("Name = %q, want %q", input.Json.Name, "Metadata")
	}
}

func TestBuildFieldCreateInput_Decimal(t *testing.T) {
	input, err := buildFieldCreateInput("decimal", "Amount", "", true, true, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if input.Decimal == nil {
		t.Fatal("expected Decimal to be set")
	}
	if input.Decimal.Name != "Amount" {
		t.Errorf("Name = %q, want %q", input.Decimal.Name, "Amount")
	}
	if input.Decimal.Required == nil || *input.Decimal.Required != true {
		t.Error("Required not set to true")
	}
	if input.Decimal.ShowOnCreate == nil || *input.Decimal.ShowOnCreate != true {
		t.Error("ShowOnCreate not set to true")
	}
}

func TestBuildFieldCreateInput_InvalidType(t *testing.T) {
	_, err := buildFieldCreateInput("invalid", "Bad", "", false, false, nil)
	if err == nil {
		t.Fatal("expected error for invalid type")
	}
}

func TestBuildFieldCreateInput_AllTypes(t *testing.T) {
	types := []struct {
		fieldType string
		checker   func(client.AspectFieldCreateInput) bool
	}{
		{"text", func(i client.AspectFieldCreateInput) bool { return i.Text != nil }},
		{"number", func(i client.AspectFieldCreateInput) bool { return i.Number != nil }},
		{"boolean", func(i client.AspectFieldCreateInput) bool { return i.Bool != nil }},
		{"date", func(i client.AspectFieldCreateInput) bool { return i.Date != nil }},
		{"datetime", func(i client.AspectFieldCreateInput) bool { return i.DateTime != nil }},
		{"dropdown", func(i client.AspectFieldCreateInput) bool { return i.Picklist != nil }},
		{"multiselect", func(i client.AspectFieldCreateInput) bool { return i.MultiPicklist != nil }},
		{"user", func(i client.AspectFieldCreateInput) bool { return i.User != nil }},
		{"group", func(i client.AspectFieldCreateInput) bool { return i.Group != nil }},
		{"longtext", func(i client.AspectFieldCreateInput) bool { return i.Html != nil }},
		{"attachment", func(i client.AspectFieldCreateInput) bool { return i.Attachment != nil }},
		{"json", func(i client.AspectFieldCreateInput) bool { return i.Json != nil }},
		{"decimal", func(i client.AspectFieldCreateInput) bool { return i.Decimal != nil }},
	}

	for _, tt := range types {
		t.Run(tt.fieldType, func(t *testing.T) {
			input, err := buildFieldCreateInput(tt.fieldType, "Test", "", false, false, nil)
			if err != nil {
				t.Fatalf("unexpected error for type %q: %v", tt.fieldType, err)
			}
			if !tt.checker(input) {
				t.Errorf("expected %s input to be set", tt.fieldType)
			}
		})
	}
}

func TestBuildFieldCreateInput_OnlyOneFieldSet(t *testing.T) {
	types := []string{"text", "number", "boolean", "date", "datetime", "dropdown",
		"multiselect", "user", "group", "longtext", "attachment", "json", "decimal"}

	for _, ft := range types {
		t.Run(ft, func(t *testing.T) {
			input, err := buildFieldCreateInput(ft, "Test", "", false, false, nil)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			setCount := 0
			if input.Text != nil {
				setCount++
			}
			if input.Number != nil {
				setCount++
			}
			if input.Bool != nil {
				setCount++
			}
			if input.Date != nil {
				setCount++
			}
			if input.DateTime != nil {
				setCount++
			}
			if input.Picklist != nil {
				setCount++
			}
			if input.MultiPicklist != nil {
				setCount++
			}
			if input.User != nil {
				setCount++
			}
			if input.Group != nil {
				setCount++
			}
			if input.Html != nil {
				setCount++
			}
			if input.Attachment != nil {
				setCount++
			}
			if input.Json != nil {
				setCount++
			}
			if input.Decimal != nil {
				setCount++
			}

			if setCount != 1 {
				t.Errorf("expected exactly 1 field set for type %q, got %d", ft, setCount)
			}
		})
	}
}

func TestBuildPicklistValues(t *testing.T) {
	tests := []struct {
		name    string
		options []string
		want    int
	}{
		{"nil options", nil, 0},
		{"empty options", []string{}, 0},
		{"single option", []string{"Open"}, 1},
		{"multiple options", []string{"Low", "Medium", "High"}, 3},
		{"with whitespace", []string{" Low ", " Medium ", " High "}, 3},
		{"empty strings filtered", []string{"Low", "", "High"}, 2},
		{"all empty", []string{"", "", ""}, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			values := buildPicklistValues(tt.options)
			if len(values) != tt.want {
				t.Errorf("buildPicklistValues() returned %d values, want %d", len(values), tt.want)
			}
		})
	}
}

func TestBuildPicklistValues_Values(t *testing.T) {
	values := buildPicklistValues([]string{"Open", "Closed", " In Progress "})
	if len(values) != 3 {
		t.Fatalf("expected 3 values, got %d", len(values))
	}
	if values[0].Value != "Open" {
		t.Errorf("values[0].Value = %q, want %q", values[0].Value, "Open")
	}
	if values[1].Value != "Closed" {
		t.Errorf("values[1].Value = %q, want %q", values[1].Value, "Closed")
	}
	if values[2].Value != "In Progress" {
		t.Errorf("values[2].Value = %q, want %q", values[2].Value, "In Progress")
	}
}

func TestSimplifyFieldType(t *testing.T) {
	tests := []struct {
		typename string
		want     string
	}{
		{"AspectTextField", "text"},
		{"AspectNumberField", "number"},
		{"AspectBooleanField", "boolean"},
		{"AspectDateField", "date"},
		{"AspectDateTimeField", "datetime"},
		{"AspectPicklistField", "dropdown"},
		{"AspectMultiPicklistField", "multiselect"},
		{"AspectUserField", "user"},
		{"AspectGroupField", "group"},
		{"AspectHtmlField", "longtext"},
		{"AspectAttachmentField", "attachment"},
		{"AspectJsonField", "json"},
		{"AspectDecimalField", "decimal"},
		{"AspectHandleField", "handle"},
		{"AspectUnknownField", "UnknownField"},
	}

	for _, tt := range tests {
		t.Run(tt.typename, func(t *testing.T) {
			got := simplifyFieldType(tt.typename)
			if got != tt.want {
				t.Errorf("simplifyFieldType(%q) = %q, want %q", tt.typename, got, tt.want)
			}
		})
	}
}

func TestBuildFieldUpdateInput_AllTypes(t *testing.T) {
	newName := "New Name"
	desc := "New desc"

	tests := []struct {
		typeName string
		checker  func(client.AspectFieldUpdateInput) bool
	}{
		{"AspectTextField", func(i client.AspectFieldUpdateInput) bool {
			return i.Text != nil && *i.Text.Name == newName && *i.Text.Description == desc
		}},
		{"AspectNumberField", func(i client.AspectFieldUpdateInput) bool {
			return i.Number != nil && *i.Number.Name == newName
		}},
		{"AspectBooleanField", func(i client.AspectFieldUpdateInput) bool {
			return i.Bool != nil && *i.Bool.Name == newName
		}},
		{"AspectDateField", func(i client.AspectFieldUpdateInput) bool {
			return i.Date != nil && *i.Date.Name == newName
		}},
		{"AspectDateTimeField", func(i client.AspectFieldUpdateInput) bool {
			return i.DateTime != nil && *i.DateTime.Name == newName
		}},
		{"AspectPicklistField", func(i client.AspectFieldUpdateInput) bool {
			return i.Picklist != nil && *i.Picklist.Name == newName
		}},
		{"AspectMultiPicklistField", func(i client.AspectFieldUpdateInput) bool {
			return i.MultiPicklist != nil && *i.MultiPicklist.Name == newName
		}},
		{"AspectUserField", func(i client.AspectFieldUpdateInput) bool {
			return i.User != nil && *i.User.Name == newName
		}},
		{"AspectGroupField", func(i client.AspectFieldUpdateInput) bool {
			return i.Group != nil && *i.Group.Name == newName
		}},
		{"AspectHtmlField", func(i client.AspectFieldUpdateInput) bool {
			return i.Html != nil && *i.Html.Name == newName
		}},
		{"AspectAttachmentField", func(i client.AspectFieldUpdateInput) bool {
			return i.Attachment != nil && *i.Attachment.Name == newName
		}},
		{"AspectJsonField", func(i client.AspectFieldUpdateInput) bool {
			return i.Json != nil && *i.Json.Name == newName
		}},
		{"AspectDecimalField", func(i client.AspectFieldUpdateInput) bool {
			return i.Decimal != nil && *i.Decimal.Name == newName
		}},
		{"AspectHandleField", func(i client.AspectFieldUpdateInput) bool {
			return i.Handle != nil && *i.Handle.Name == newName
		}},
	}

	for _, tt := range tests {
		t.Run(tt.typeName, func(t *testing.T) {
			input, err := buildFieldUpdateInput(tt.typeName, newName, desc, false, false, false, false)
			if err != nil {
				t.Fatalf("unexpected error for %q: %v", tt.typeName, err)
			}
			if !tt.checker(input) {
				t.Errorf("update input not built correctly for %q", tt.typeName)
			}
		})
	}
}

func TestBuildFieldUpdateInput_RequiredFlags(t *testing.T) {
	tests := []struct {
		name           string
		required       bool
		noRequired     bool
		showOnCreate   bool
		noShowOnCreate bool
		wantRequired   *bool
		wantShow       *bool
	}{
		{"no flags", false, false, false, false, nil, nil},
		{"required true", true, false, false, false, boolPtr(true), nil},
		{"required false", false, true, false, false, boolPtr(false), nil},
		{"show on create true", false, false, true, false, nil, boolPtr(true)},
		{"show on create false", false, false, false, true, nil, boolPtr(false)},
		{"both required and show", true, false, true, false, boolPtr(true), boolPtr(true)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input, err := buildFieldUpdateInput("AspectTextField", "Name", "",
				tt.required, tt.noRequired, tt.showOnCreate, tt.noShowOnCreate)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if input.Text == nil {
				t.Fatal("expected Text to be set")
			}
			if tt.wantRequired == nil {
				if input.Text.Required != nil {
					t.Errorf("Required should be nil, got %v", *input.Text.Required)
				}
			} else {
				if input.Text.Required == nil {
					t.Error("Required should not be nil")
				} else if *input.Text.Required != *tt.wantRequired {
					t.Errorf("Required = %v, want %v", *input.Text.Required, *tt.wantRequired)
				}
			}
			if tt.wantShow == nil {
				if input.Text.ShowOnCreate != nil {
					t.Errorf("ShowOnCreate should be nil, got %v", *input.Text.ShowOnCreate)
				}
			} else {
				if input.Text.ShowOnCreate == nil {
					t.Error("ShowOnCreate should not be nil")
				} else if *input.Text.ShowOnCreate != *tt.wantShow {
					t.Errorf("ShowOnCreate = %v, want %v", *input.Text.ShowOnCreate, *tt.wantShow)
				}
			}
		})
	}
}

func TestBuildFieldUpdateInput_UnsupportedType(t *testing.T) {
	_, err := buildFieldUpdateInput("AspectCalculatedField", "Name", "", false, false, false, false)
	if err == nil {
		t.Fatal("expected error for unsupported type")
	}
}

func TestBuildFieldUpdateInput_NilPointers(t *testing.T) {
	input, err := buildFieldUpdateInput("AspectTextField", "", "", false, false, false, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if input.Text == nil {
		t.Fatal("expected Text to be set")
	}
	if input.Text.Name != nil {
		t.Error("Name should be nil when empty string")
	}
	if input.Text.Description != nil {
		t.Error("Description should be nil when empty string")
	}
	if input.Text.Required != nil {
		t.Error("Required should be nil when no flag set")
	}
	if input.Text.ShowOnCreate != nil {
		t.Error("ShowOnCreate should be nil when no flag set")
	}
}

func boolPtr(b bool) *bool {
	return &b
}
