// Copyright (c) ElementumAI, Inc.
// SPDX-License-Identifier: MPL-2.0

package export

import (
	"testing"
)

func TestStripNulls(t *testing.T) {
	b := NewResourceBlock("elementum_app", "my_app")
	b.SetAttr("name", Str("My App"))
	b.SetAttr("description", HCLNull{})
	b.SetAttr("icon", Str("star"))
	b.SetAttr("options", HCLNull{})

	blocks := []*HCLBlock{b}
	StripNulls(blocks)

	if len(b.Body.Attributes) != 2 {
		t.Fatalf("expected 2 attributes after stripping nulls, got %d", len(b.Body.Attributes))
	}
	if b.Body.Attributes[0].Name != "name" {
		t.Errorf("expected first attr 'name', got %q", b.Body.Attributes[0].Name)
	}
	if b.Body.Attributes[1].Name != "icon" {
		t.Errorf("expected second attr 'icon', got %q", b.Body.Attributes[1].Name)
	}
}

func TestStripNulls_InObjects(t *testing.T) {
	b := NewResourceBlock("elementum_widget", "w")
	b.SetAttr("config", Obj(
		Attr("name", Str("test")),
		Attr("value", HCLNull{}),
		Attr("enabled", Bool(true)),
	))

	blocks := []*HCLBlock{b}
	StripNulls(blocks)

	obj, ok := b.Body.Attributes[0].Value.(HCLObject)
	if !ok {
		t.Fatal("expected HCLObject")
	}
	if len(obj.Attributes) != 2 {
		t.Fatalf("expected 2 attrs in object after stripping nulls, got %d", len(obj.Attributes))
	}
}

func TestStripEmptyObjects(t *testing.T) {
	b := NewResourceBlock("elementum_flow", "f")
	b.SetAttr("name", Str("flow"))
	b.SetAttr("action", Obj())                        // empty object
	b.SetAttr("config", Obj(Attr("key", Str("val")))) // non-empty

	blocks := []*HCLBlock{b}
	StripEmptyObjects(blocks)

	if len(b.Body.Attributes) != 2 {
		t.Fatalf("expected 2 attributes after stripping empty objects, got %d", len(b.Body.Attributes))
	}
	if b.Body.Attributes[0].Name != "name" {
		t.Errorf("expected first attr 'name', got %q", b.Body.Attributes[0].Name)
	}
	if b.Body.Attributes[1].Name != "config" {
		t.Errorf("expected second attr 'config', got %q", b.Body.Attributes[1].Name)
	}
}

func TestResolveUUIDs(t *testing.T) {
	uuidMap := map[string]string{
		"abc-123-def": "elementum_app.my_app.id",
		"field-uuid":  "elementum_text_field.status.id",
	}

	b := NewResourceBlock("elementum_field", "status")
	b.SetAttr("app_id", Str("abc-123-def"))
	b.SetAttr("name", Str("Status"))
	b.SetAttr("field_ids", List(Str("field-uuid"), Str("other-id")))

	blocks := []*HCLBlock{b}
	ResolveUUIDs(blocks, uuidMap)

	// app_id should be resolved to a reference
	appID := b.Body.Attributes[0]
	if ref, ok := appID.Value.(HCLReference); ok {
		if ref.Ref != "elementum_app.my_app.id" {
			t.Errorf("expected 'elementum_app.my_app.id', got %q", ref.Ref)
		}
	} else {
		t.Errorf("expected HCLReference for app_id, got %T", appID.Value)
	}

	// name should remain a string
	name := b.Body.Attributes[1]
	if _, ok := name.Value.(HCLString); !ok {
		t.Errorf("expected HCLString for name, got %T", name.Value)
	}

	// field_ids list: first element should be resolved, second stays
	fieldIDs := b.Body.Attributes[2]
	list, ok := fieldIDs.Value.(HCLList)
	if !ok {
		t.Fatalf("expected HCLList, got %T", fieldIDs.Value)
	}
	if _, ok := list.Values[0].(HCLReference); !ok {
		t.Errorf("expected first list element to be HCLReference, got %T", list.Values[0])
	}
	if _, ok := list.Values[1].(HCLString); !ok {
		t.Errorf("expected second list element to remain HCLString, got %T", list.Values[1])
	}
}

func TestResolveUUIDs_InObjects(t *testing.T) {
	uuidMap := map[string]string{
		"uuid-123": "elementum_app.related.id",
	}

	b := NewResourceBlock("elementum_widget", "w")
	b.SetAttr("related_object", Obj(
		Attr("object_id", Str("uuid-123")),
		Attr("name", Str("Related")),
	))

	blocks := []*HCLBlock{b}
	ResolveUUIDs(blocks, uuidMap)

	obj := b.Body.Attributes[0].Value.(HCLObject)
	if ref, ok := obj.Attributes[0].Value.(HCLReference); ok {
		if ref.Ref != "elementum_app.related.id" {
			t.Errorf("expected 'elementum_app.related.id', got %q", ref.Ref)
		}
	} else {
		t.Errorf("expected HCLReference for object_id, got %T", obj.Attributes[0].Value)
	}
}

func TestFixSelfReferences(t *testing.T) {
	b := NewResourceBlock("elementum_table", "my_table")
	b.SetAttr("name", Str("My Table"))
	b.SetAttr("source_id", Ref("elementum_table.my_table.id")) // self-reference

	blocks := []*HCLBlock{b}
	FixSelfReferences(blocks)

	// source_id should be stripped
	if len(b.Body.Attributes) != 1 {
		t.Fatalf("expected 1 attribute after fixing self-refs, got %d", len(b.Body.Attributes))
	}
	if b.Body.Attributes[0].Name != "name" {
		t.Errorf("expected remaining attr 'name', got %q", b.Body.Attributes[0].Name)
	}
}

func TestFixSelfReferences_NonSourceID(t *testing.T) {
	b := NewResourceBlock("elementum_app", "my_app")
	b.SetAttr("name", Str("My App"))
	b.SetAttr("parent_id", Ref("elementum_app.my_app.id")) // self-reference on non-source_id

	blocks := []*HCLBlock{b}
	FixSelfReferences(blocks)

	// parent_id should be set to "" instead of stripped
	if len(b.Body.Attributes) != 2 {
		t.Fatalf("expected 2 attributes, got %d", len(b.Body.Attributes))
	}
	parentID := b.Body.Attributes[1]
	if str, ok := parentID.Value.(HCLString); ok {
		if str.Value != "" {
			t.Errorf("expected empty string, got %q", str.Value)
		}
	} else {
		t.Errorf("expected HCLString for self-referencing non-source_id attr, got %T", parentID.Value)
	}
}

func TestInjectWarnings(t *testing.T) {
	b := NewResourceBlock("elementum_agent", "helper")
	b.SetAttr("name", Str("Helper"))
	b.SetAttr("ai_provider_connector_id", Str("12345678-1234-1234-1234-123456789abc"))

	blocks := []*HCLBlock{b}
	InjectWarnings(blocks)

	connAttr := b.Body.Attributes[1]
	if connAttr.Comment == "" {
		t.Error("expected a warning comment on ai_provider_connector_id")
	}
}

func TestMergeBlocks(t *testing.T) {
	// Base blocks (from tofu plan)
	base := []*HCLBlock{
		{Type: "resource", Labels: []string{"elementum_app", "my_app"}, Body: &HCLBody{
			Attributes: []*HCLAttribute{{Name: "name", Value: Str("App")}},
		}},
		{Type: "resource", Labels: []string{"elementum_field", "status"}, Body: &HCLBody{
			Attributes: []*HCLAttribute{{Name: "name", Value: Str("Status")}},
		}},
	}

	// Override blocks (from CLI generators)
	override := []*HCLBlock{
		{Type: "resource", Labels: []string{"elementum_app", "my_app"}, Body: &HCLBody{
			Attributes: []*HCLAttribute{
				{Name: "name", Value: Str("App")},
				{Name: "status_options", Value: List(Str("Active"), Str("Closed"))},
			},
		}},
		{Type: "resource", Labels: []string{"elementum_automation", "proc"}, Body: &HCLBody{
			Attributes: []*HCLAttribute{{Name: "name", Value: Str("Process")}},
		}},
	}

	result := MergeBlocks(base, override)

	if len(result) != 3 {
		t.Fatalf("expected 3 merged blocks, got %d", len(result))
	}

	// First block should be the override (has status_options)
	if len(result[0].Body.Attributes) != 2 {
		t.Errorf("expected overridden app block to have 2 attrs, got %d", len(result[0].Body.Attributes))
	}

	// Second block should be original (field unchanged)
	if result[1].Labels[1] != "status" {
		t.Errorf("expected second block 'status', got %q", result[1].Labels[1])
	}

	// Third block should be the new automation
	if result[2].Labels[1] != "proc" {
		t.Errorf("expected third block 'proc', got %q", result[2].Labels[1])
	}
}

func TestRemoveLayoutCommentsBlocks(t *testing.T) {
	b := NewResourceBlock("elementum_layout", "initiate")
	b.SetAttr("name", Str("Initiate"))
	b.SetAttr("comments", Obj(Attr("text", Str("some comment"))))

	blocks := []*HCLBlock{b}
	RemoveLayoutCommentsBlocks(blocks)

	if len(b.Body.Attributes) != 1 {
		t.Fatalf("expected 1 attribute after removing comments, got %d", len(b.Body.Attributes))
	}
	if b.Body.Attributes[0].Name != "name" {
		t.Errorf("expected remaining attr 'name', got %q", b.Body.Attributes[0].Name)
	}
}

func TestIsUUID(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"12345678-1234-1234-1234-123456789abc", true},
		{"abcdef01-2345-6789-abcd-ef0123456789", true},
		{"not-a-uuid", false},
		{"12345678-1234-1234-1234-12345678", false}, // too short
		{"", false},
	}

	for _, tt := range tests {
		if got := isUUID(tt.input); got != tt.expected {
			t.Errorf("isUUID(%q) = %v, want %v", tt.input, got, tt.expected)
		}
	}
}
