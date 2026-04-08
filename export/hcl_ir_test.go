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

// ============================================================================
// IR Construction Tests
// ============================================================================

func TestNewResourceBlock(t *testing.T) {
	t.Parallel()
	b := NewResourceBlock("elementum_app", "my_app")

	if b.Type != "resource" {
		t.Errorf("expected type 'resource', got %q", b.Type)
	}
	if len(b.Labels) != 2 || b.Labels[0] != "elementum_app" || b.Labels[1] != "my_app" {
		t.Errorf("expected labels [elementum_app my_app], got %v", b.Labels)
	}
	if b.Meta.ResourceType != "elementum_app" {
		t.Errorf("expected meta ResourceType 'elementum_app', got %q", b.Meta.ResourceType)
	}
	if b.Meta.ResourceName != "my_app" {
		t.Errorf("expected meta ResourceName 'my_app', got %q", b.Meta.ResourceName)
	}
}

func TestNewDataBlock(t *testing.T) {
	t.Parallel()
	b := NewDataBlock("elementum_field", "status")

	if b.Type != "data" {
		t.Errorf("expected type 'data', got %q", b.Type)
	}
	if len(b.Labels) != 2 || b.Labels[0] != "elementum_field" || b.Labels[1] != "status" {
		t.Errorf("expected labels [elementum_field status], got %v", b.Labels)
	}
}

func TestBlockSetAttr(t *testing.T) {
	t.Parallel()
	b := NewBlock("resource", "elementum_app", "my_app")
	b.SetAttr("namespace", Str("my-app"))
	b.SetAttr("name", Str("My App"))

	if len(b.Body.Attributes) != 2 {
		t.Fatalf("expected 2 attributes, got %d", len(b.Body.Attributes))
	}
	if b.Body.Attributes[0].Name != "namespace" {
		t.Errorf("expected first attr name 'namespace', got %q", b.Body.Attributes[0].Name)
	}
}

func TestBlockAddBlock(t *testing.T) {
	t.Parallel()
	parent := NewBlock("resource", "elementum_automation", "my_auto")
	child := NewBlock("trigger")
	child.SetAttr("type", Str("record_created"))
	parent.AddBlock(child)

	if len(parent.Body.Blocks) != 1 {
		t.Fatalf("expected 1 nested block, got %d", len(parent.Body.Blocks))
	}
	if parent.Body.Blocks[0].Type != "trigger" {
		t.Errorf("expected nested block type 'trigger', got %q", parent.Body.Blocks[0].Type)
	}
}

// ============================================================================
// Serialization Tests
// ============================================================================

func TestSerialize_SimpleResource(t *testing.T) {
	t.Parallel()
	b := NewResourceBlock("elementum_app", "my_app")
	b.SetAttr("namespace", Str("my-app"))
	b.SetAttr("name", Str("My App"))

	got := SerializeBlocks([]*HCLBlock{b})
	expected := `resource "elementum_app" "my_app" {
  namespace = "my-app"
  name = "My App"
}
`

	if got != expected {
		t.Errorf("mismatch:\ngot:\n%s\nexpected:\n%s", got, expected)
	}
}

func TestSerialize_DataSource(t *testing.T) {
	t.Parallel()
	b := NewDataBlock("elementum_app", "tickets")
	b.SetAttr("name", Str("Tickets"))

	got := SerializeBlocks([]*HCLBlock{b})
	expected := `data "elementum_app" "tickets" {
  name = "Tickets"
}
`

	if got != expected {
		t.Errorf("mismatch:\ngot:\n%s\nexpected:\n%s", got, expected)
	}
}

func TestSerialize_AllValueTypes(t *testing.T) {
	t.Parallel()
	b := NewBlock("resource", "test_resource", "all_types")
	b.SetAttr("str_val", Str("hello world"))
	b.SetAttr("num_int", Num(42))
	b.SetAttr("num_float", Num(3.14))
	b.SetAttr("bool_true", Bool(true))
	b.SetAttr("bool_false", Bool(false))
	b.SetAttr("null_val", Null())
	b.SetAttr("ref_val", Ref("elementum_app.my_app.id"))

	got := SerializeBlocks([]*HCLBlock{b})

	assertIRContains(t, got, `str_val = "hello world"`)
	assertIRContains(t, got, `num_int = 42`)
	assertIRContains(t, got, `num_float = 3.14`)
	assertIRContains(t, got, `bool_true = true`)
	assertIRContains(t, got, `bool_false = false`)
	assertIRContains(t, got, `null_val = null`)
	assertIRContains(t, got, `ref_val = elementum_app.my_app.id`)
}

func TestSerialize_StringEscaping(t *testing.T) {
	t.Parallel()
	b := NewBlock("resource", "test", "escaping")
	b.SetAttr("with_quotes", Str(`say "hello"`))
	b.SetAttr("with_backslash", Str(`path\to\file`))
	b.SetAttr("with_newline", Str("line1\nline2"))

	got := SerializeBlocks([]*HCLBlock{b})

	assertIRContains(t, got, `with_quotes = "say \"hello\""`)
	assertIRContains(t, got, `with_backslash = "path\\to\\file"`)
	assertIRContains(t, got, `with_newline = "line1\nline2"`)
}

func TestSerialize_Heredoc(t *testing.T) {
	t.Parallel()
	b := NewBlock("resource", "test", "heredoc")
	b.SetAttr("description", Heredoc("Hello World\nSecond line", "EOT"))

	got := SerializeBlocks([]*HCLBlock{b})

	assertIRContains(t, got, "<<-EOT")
	assertIRContains(t, got, "Hello World")
	assertIRContains(t, got, "Second line")
	assertIRContains(t, got, "EOT")
}

func TestSerialize_HeredocEmpty(t *testing.T) {
	t.Parallel()
	b := NewBlock("resource", "test", "heredoc_empty")
	b.SetAttr("description", Heredoc("", "EOF"))

	got := SerializeBlocks([]*HCLBlock{b})

	assertIRContains(t, got, "<<-EOF")
	assertIRContains(t, got, "EOF")
}

func TestSerialize_InterpolatedString(t *testing.T) {
	t.Parallel()
	b := NewBlock("resource", "test", "interp")
	b.SetAttr("message", HCLInterpolated{Parts: []InterpolationPart{
		{Literal: "Hello "},
		{Reference: "var.name"},
		{Literal: ", welcome!"},
	}})

	got := SerializeBlocks([]*HCLBlock{b})

	assertIRContains(t, got, `message = "Hello ${var.name}, welcome!"`)
}

func TestSerialize_InterpolatedMultipleRefs(t *testing.T) {
	t.Parallel()
	b := NewBlock("resource", "test", "multi_interp")
	b.SetAttr("message", HCLInterpolated{Parts: []InterpolationPart{
		{Literal: "New ticket: "},
		{Reference: `elementum_automation.example.refs["Title"]`},
		{Literal: " - Status: "},
		{Reference: `elementum_automation.example.refs["Status"]`},
	}})

	got := SerializeBlocks([]*HCLBlock{b})

	assertIRContains(t, got, `"New ticket: ${elementum_automation.example.refs["Title"]} - Status: ${elementum_automation.example.refs["Status"]}"`)
}

func TestSerialize_EmptyList(t *testing.T) {
	t.Parallel()
	b := NewBlock("resource", "test", "empty_list")
	b.SetAttr("tags", List())

	got := SerializeBlocks([]*HCLBlock{b})

	assertIRContains(t, got, `tags = []`)
}

func TestSerialize_SimpleInlineList(t *testing.T) {
	t.Parallel()
	b := NewBlock("resource", "test", "inline_list")
	b.SetAttr("tags", List(Str("a"), Str("b"), Str("c")))

	got := SerializeBlocks([]*HCLBlock{b})

	assertIRContains(t, got, `tags = ["a", "b", "c"]`)
}

func TestSerialize_RefList(t *testing.T) {
	t.Parallel()
	b := NewBlock("resource", "test", "ref_list")
	b.SetAttr("user_ids", List(
		Ref("data.elementum_user.admin.id"),
		Ref("data.elementum_user.editor.id"),
	))

	got := SerializeBlocks([]*HCLBlock{b})

	assertIRContains(t, got, `user_ids = [data.elementum_user.admin.id, data.elementum_user.editor.id]`)
}

func TestSerialize_MultiLineList(t *testing.T) {
	t.Parallel()
	b := NewBlock("resource", "test", "multi_list")
	b.SetAttr("columns", HCLList{Values: []HCLValue{
		Obj(
			Attr("field_id", Ref("elementum_text_field.name.id")),
			Attr("related_field_id", Ref("elementum_text_field.other_name.id")),
		),
		Obj(
			Attr("field_id", Ref("elementum_text_field.email.id")),
			Attr("related_field_id", Ref("elementum_text_field.other_email.id")),
		),
	}})

	got := SerializeBlocks([]*HCLBlock{b})

	// Multi-line list with objects
	assertIRContains(t, got, "columns = [")
	assertIRContains(t, got, "field_id = elementum_text_field.name.id")
	assertIRContains(t, got, "related_field_id = elementum_text_field.other_name.id")

	// Verify proper indentation
	lines := strings.Split(got, "\n")
	for _, line := range lines {
		if strings.Contains(line, "field_id = elementum_text_field.name.id") {
			// Should be indented 3 levels (resource > columns item > object attr)
			if !strings.HasPrefix(line, "      ") {
				t.Errorf("expected 6-space indent for object attr, got: %q", line)
			}
		}
	}
}

func TestSerialize_EmptyObject(t *testing.T) {
	t.Parallel()
	b := NewBlock("resource", "test", "empty_obj")
	b.SetAttr("config", Obj())

	got := SerializeBlocks([]*HCLBlock{b})

	assertIRContains(t, got, `config = {}`)
}

func TestSerialize_Object(t *testing.T) {
	t.Parallel()
	b := NewBlock("resource", "test", "obj")
	b.SetAttr("filter", Obj(
		Attr("type", Str("equals")),
		Attr("field_id", Ref("data.elementum_field.status.id")),
		Attr("value", Str("active")),
	))

	got := SerializeBlocks([]*HCLBlock{b})

	assertIRContains(t, got, "filter = {")
	assertIRContains(t, got, `type = "equals"`)
	assertIRContains(t, got, "field_id = data.elementum_field.status.id")
	assertIRContains(t, got, `value = "active"`)
}

func TestSerialize_NestedBlock(t *testing.T) {
	t.Parallel()
	parent := NewResourceBlock("elementum_automation", "process_ticket")
	parent.SetAttr("name", Str("Process Ticket"))

	trigger := NewBlock("trigger")
	trigger.SetAttr("type", Str("record_created"))
	parent.AddBlock(trigger)

	got := SerializeBlocks([]*HCLBlock{parent})

	expected := `resource "elementum_automation" "process_ticket" {
  name = "Process Ticket"

  trigger {
    type = "record_created"
  }
}
`

	if got != expected {
		t.Errorf("mismatch:\ngot:\n%s\nexpected:\n%s", got, expected)
	}
}

func TestSerialize_MultipleNestedBlocks(t *testing.T) {
	t.Parallel()
	parent := NewResourceBlock("elementum_automation", "my_auto")
	parent.SetAttr("app_id", Ref("elementum_app.my_app.id"))

	trigger1 := NewBlock("trigger")
	trigger1.SetAttr("type", Str("record_created"))
	parent.AddBlock(trigger1)

	trigger2 := NewBlock("trigger")
	trigger2.SetAttr("type", Str("record_updated"))
	parent.AddBlock(trigger2)

	got := SerializeBlocks([]*HCLBlock{parent})

	// Verify both triggers present with blank line between them
	assertIRContains(t, got, `type = "record_created"`)
	assertIRContains(t, got, `type = "record_updated"`)
}

func TestSerialize_DeeplyNested(t *testing.T) {
	t.Parallel()

	// resource > filter > children > child_filter
	resource := NewResourceBlock("elementum_record_search_task", "search")
	resource.SetAttr("name", Str("Search"))

	filter := NewBlock("filter")
	filter.SetAttr("type", Str("and"))

	child := NewBlock("children")
	child.SetAttr("type", Str("equals"))
	child.SetAttr("field_id", Ref("data.elementum_field.status.id"))
	filter.AddBlock(child)

	resource.AddBlock(filter)

	got := SerializeBlocks([]*HCLBlock{resource})

	// Verify indentation levels
	assertIRContains(t, got, "  filter {")                                      // indent 1
	assertIRContains(t, got, `    type = "and"`)                                // indent 2
	assertIRContains(t, got, "    children {")                                  // indent 2
	assertIRContains(t, got, `      type = "equals"`)                           // indent 3
	assertIRContains(t, got, "      field_id = data.elementum_field.status.id") // indent 3
}

func TestSerialize_Comment(t *testing.T) {
	t.Parallel()
	b := NewBlock("resource", "test", "with_comment")
	b.SetAttrComment("field_id", Str("uuid-placeholder"), "# TODO: resolve reference")

	got := SerializeBlocks([]*HCLBlock{b})

	assertIRContains(t, got, `field_id = "uuid-placeholder" # TODO: resolve reference`)
}

func TestSerialize_RawValue(t *testing.T) {
	t.Parallel()
	b := NewBlock("resource", "test", "raw")
	b.SetAttr("complex_expr", Raw(`jsonencode({"key": "value"})`))

	got := SerializeBlocks([]*HCLBlock{b})

	assertIRContains(t, got, `complex_expr = jsonencode({"key": "value"})`)
}

func TestSerialize_MultipleBlocks_BlankLineSeparation(t *testing.T) {
	t.Parallel()
	block1 := NewResourceBlock("elementum_app", "app1")
	block1.SetAttr("name", Str("App 1"))

	block2 := NewResourceBlock("elementum_app", "app2")
	block2.SetAttr("name", Str("App 2"))

	got := SerializeBlocks([]*HCLBlock{block1, block2})

	// Should have blank line between blocks
	expected := `resource "elementum_app" "app1" {
  name = "App 1"
}

resource "elementum_app" "app2" {
  name = "App 2"
}
`

	if got != expected {
		t.Errorf("mismatch:\ngot:\n%s\nexpected:\n%s", got, expected)
	}
}

func TestSerialize_EmptyBlocks(t *testing.T) {
	t.Parallel()
	got := SerializeBlocks(nil)
	if got != "" {
		t.Errorf("expected empty string for nil blocks, got: %q", got)
	}

	got = SerializeBlocks([]*HCLBlock{})
	if got != "" {
		t.Errorf("expected empty string for empty blocks, got: %q", got)
	}
}

func TestSerialize_ImportBlock(t *testing.T) {
	t.Parallel()
	b := NewImportBlock()
	b.SetAttr("to", Ref("elementum_app.my_app"))
	b.SetAttr("id", Str("uuid-123"))

	got := SerializeBlocks([]*HCLBlock{b})

	expected := `import {
  to = elementum_app.my_app
  id = "uuid-123"
}
`

	if got != expected {
		t.Errorf("mismatch:\ngot:\n%s\nexpected:\n%s", got, expected)
	}
}

func TestSerialize_FullRelationship(t *testing.T) {
	t.Parallel()

	// Simulate a full relationship resource like the current generators produce
	b := NewResourceBlock("elementum_relationship", "tickets_to_users")
	b.SetAttr("object_id", Ref("elementum_app.tickets.id"))
	b.SetAttr("related_object_id", Ref("elementum_app.users.id"))
	b.SetAttr("columns", HCLList{Values: []HCLValue{
		Obj(
			Attr("field_id", Ref("elementum_user_field.assigned_user.id")),
			Attr("related_field_id", Ref("elementum_text_field.user_name.id")),
		),
	}})

	filter := NewBlock("filter")
	filter.SetAttr("type", Str("equals"))
	filter.SetAttr("field_id", Ref("elementum_dropdown_field.status.id"))
	b.AddBlock(filter)

	got := SerializeBlocks([]*HCLBlock{b})

	assertIRContains(t, got, `resource "elementum_relationship" "tickets_to_users"`)
	assertIRContains(t, got, "object_id = elementum_app.tickets.id")
	assertIRContains(t, got, "related_object_id = elementum_app.users.id")
	assertIRContains(t, got, "columns = [")
	assertIRContains(t, got, "field_id = elementum_user_field.assigned_user.id")
	assertIRContains(t, got, `type = "equals"`)
}

func TestSerialize_FullAccessPolicy(t *testing.T) {
	t.Parallel()

	// Data sources
	userData := NewDataBlock("elementum_user", "admin")
	userData.SetAttr("email", Str("admin@example.com"))

	// Access policy
	policy := NewResourceBlock("elementum_access_policy", "admin_policy")
	policy.SetAttr("object_id", Ref("elementum_app.my_app.id"))

	filter := NewBlock("filter")
	filter.SetAttr("type", Str("equals"))
	filter.SetAttr("field_id", Ref("elementum_dropdown_field.status.id"))
	policy.AddBlock(filter)

	policy.SetAttr("user_ids", HCLList{Values: []HCLValue{
		Ref("data.elementum_user.admin.id"),
	}})

	got := SerializeBlocks([]*HCLBlock{userData, policy})

	assertIRContains(t, got, `data "elementum_user" "admin"`)
	assertIRContains(t, got, `email = "admin@example.com"`)
	assertIRContains(t, got, `resource "elementum_access_policy" "admin_policy"`)
	assertIRContains(t, got, "object_id = elementum_app.my_app.id")
}

func TestSerialize_LargeRefListMultiLine(t *testing.T) {
	t.Parallel()
	// More than 5 simple refs should go multi-line
	b := NewBlock("resource", "test", "many_refs")
	b.SetAttr("user_ids", List(
		Ref("data.elementum_user.a.id"),
		Ref("data.elementum_user.b.id"),
		Ref("data.elementum_user.c.id"),
		Ref("data.elementum_user.d.id"),
		Ref("data.elementum_user.e.id"),
		Ref("data.elementum_user.f.id"),
	))

	got := SerializeBlocks([]*HCLBlock{b})

	// Should be multi-line because > 5 items
	assertIRContains(t, got, "user_ids = [\n")
	assertIRContains(t, got, "data.elementum_user.a.id,")
}

func TestSerialize_MixedAttributesAndBlocks(t *testing.T) {
	t.Parallel()

	// Resource with attributes, then a block, then more blocks
	b := NewResourceBlock("elementum_record_search_task", "find_records")
	b.SetAttr("workflow_id", Ref("elementum_automation.my_auto.workflow_id"))
	b.SetAttr("name", Str("Find Records"))
	b.SetAttr("aspect_id", Ref("elementum_app.tickets.id"))

	filter := NewBlock("filter")
	filter.SetAttr("type", Str("and"))

	child1 := NewBlock("children")
	child1.SetAttr("type", Str("equals"))
	child1.SetAttr("field_id", Ref("elementum_dropdown_field.status.id"))
	filter.AddBlock(child1)

	child2 := NewBlock("children")
	child2.SetAttr("type", Str("not_empty"))
	child2.SetAttr("field_id", Ref("elementum_text_field.name.id"))
	filter.AddBlock(child2)

	b.AddBlock(filter)

	got := SerializeBlocks([]*HCLBlock{b})

	// Check structure
	assertIRContains(t, got, `name = "Find Records"`)
	assertIRContains(t, got, "  filter {")
	assertIRContains(t, got, `    type = "and"`)
	assertIRContains(t, got, "    children {")
}

// ============================================================================
// Helper
// ============================================================================

// assertIRContains checks if haystack contains needle, using a descriptive error message.
func assertIRContains(t *testing.T, haystack, needle string) {
	t.Helper()
	if !strings.Contains(haystack, needle) {
		t.Errorf("expected output to contain %q\ngot:\n%s", needle, haystack)
	}
}
