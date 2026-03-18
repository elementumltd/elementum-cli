// Copyright (c) ElementumAI, Inc.
// SPDX-License-Identifier: MPL-2.0

package export

import (
	"strings"
	"testing"
)

func TestParseTofuOutput_SimpleResource(t *testing.T) {
	input := `
resource "elementum_app" "my_app" {
  name        = "My App"
  namespace   = "my-app"
  description = "A test app"
}
`
	blocks, err := ParseTofuOutput(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(blocks))
	}

	b := blocks[0]
	if b.Type != "resource" {
		t.Errorf("expected type 'resource', got %q", b.Type)
	}
	if len(b.Labels) != 2 || b.Labels[0] != "elementum_app" || b.Labels[1] != "my_app" {
		t.Errorf("unexpected labels: %v", b.Labels)
	}
	if b.Meta.ResourceType != "elementum_app" || b.Meta.ResourceName != "my_app" {
		t.Errorf("unexpected meta: %+v", b.Meta)
	}

	assertAttrCount(t, b, 3)
	assertAttrStr(t, b, "name", "My App")
	assertAttrStr(t, b, "namespace", "my-app")
	assertAttrStr(t, b, "description", "A test app")
}

func TestParseTofuOutput_NullAndBool(t *testing.T) {
	input := `
resource "elementum_text_field" "status" {
  name     = "Status"
  required = true
  readonly = false
  options  = null
}
`
	blocks, err := ParseTofuOutput(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(blocks))
	}

	b := blocks[0]
	assertAttrStr(t, b, "name", "Status")

	// Check bool attributes
	for _, attr := range b.Body.Attributes {
		switch attr.Name {
		case "required":
			if _, ok := attr.Value.(HCLBool); !ok {
				t.Errorf("expected required to be HCLBool, got %T", attr.Value)
			}
		case "readonly":
			if _, ok := attr.Value.(HCLBool); !ok {
				t.Errorf("expected readonly to be HCLBool, got %T", attr.Value)
			}
		case "options":
			if _, ok := attr.Value.(HCLNull); !ok {
				t.Errorf("expected options to be HCLNull, got %T", attr.Value)
			}
		}
	}
}

func TestParseTofuOutput_SimpleList(t *testing.T) {
	input := `
resource "elementum_text_field" "tags" {
  field_ids = ["abc-123", "def-456"]
}
`
	blocks, err := ParseTofuOutput(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	b := blocks[0]

	var listAttr *HCLAttribute
	for _, attr := range b.Body.Attributes {
		if attr.Name == "field_ids" {
			listAttr = attr
			break
		}
	}
	if listAttr == nil {
		t.Fatal("field_ids attribute not found")
	}

	list, ok := listAttr.Value.(HCLList)
	if !ok {
		t.Fatalf("expected HCLList, got %T", listAttr.Value)
	}
	if len(list.Values) != 2 {
		t.Fatalf("expected 2 list elements, got %d", len(list.Values))
	}
}

func TestParseTofuOutput_MultiLineList(t *testing.T) {
	input := `
resource "elementum_role" "admin" {
  permissions = [
    {
      resource = "records"
      actions  = "all"
    },
    {
      resource = "fields"
      actions  = "read"
    },
  ]
}
`
	blocks, err := ParseTofuOutput(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	b := blocks[0]

	var listAttr *HCLAttribute
	for _, attr := range b.Body.Attributes {
		if attr.Name == "permissions" {
			listAttr = attr
			break
		}
	}
	if listAttr == nil {
		t.Fatal("permissions attribute not found")
	}

	list, ok := listAttr.Value.(HCLList)
	if !ok {
		t.Fatalf("expected HCLList, got %T", listAttr.Value)
	}
	if len(list.Values) != 2 {
		t.Fatalf("expected 2 list elements, got %d", len(list.Values))
	}

	// First element should be an object
	obj, ok := list.Values[0].(HCLObject)
	if !ok {
		t.Fatalf("expected HCLObject, got %T", list.Values[0])
	}
	if len(obj.Attributes) != 2 {
		t.Fatalf("expected 2 attrs in first object, got %d", len(obj.Attributes))
	}
}

func TestParseTofuOutput_NestedBlock(t *testing.T) {
	input := `
resource "elementum_automation" "process" {
  app_id = "abc-123"
  name   = "Process"

  trigger {
    type = "record_created"
  }
}
`
	blocks, err := ParseTofuOutput(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	b := blocks[0]
	assertAttrStr(t, b, "name", "Process")

	if len(b.Body.Blocks) != 1 {
		t.Fatalf("expected 1 nested block, got %d", len(b.Body.Blocks))
	}

	trigger := b.Body.Blocks[0]
	if trigger.Type != "trigger" {
		t.Errorf("expected nested block type 'trigger', got %q", trigger.Type)
	}
}

func TestParseTofuOutput_ObjectAttribute(t *testing.T) {
	input := `
resource "elementum_widget" "chart" {
  related_object = {
    object_id = elementum_app.my_app.id
    name      = "Related"
  }
}
`
	blocks, err := ParseTofuOutput(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	b := blocks[0]

	var objAttr *HCLAttribute
	for _, attr := range b.Body.Attributes {
		if attr.Name == "related_object" {
			objAttr = attr
			break
		}
	}
	if objAttr == nil {
		t.Fatal("related_object attribute not found")
	}

	obj, ok := objAttr.Value.(HCLObject)
	if !ok {
		t.Fatalf("expected HCLObject, got %T", objAttr.Value)
	}
	if len(obj.Attributes) != 2 {
		t.Fatalf("expected 2 object attrs, got %d", len(obj.Attributes))
	}
}

func TestParseTofuOutput_Heredoc(t *testing.T) {
	input := `
resource "elementum_agent" "helper" {
  instructions = <<-EOT
    You are a helpful assistant.
    Be concise and accurate.
  EOT
  name = "Helper"
}
`
	blocks, err := ParseTofuOutput(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	b := blocks[0]

	var heredocAttr *HCLAttribute
	for _, attr := range b.Body.Attributes {
		if attr.Name == "instructions" {
			heredocAttr = attr
			break
		}
	}
	if heredocAttr == nil {
		t.Fatal("instructions attribute not found")
	}

	heredoc, ok := heredocAttr.Value.(HCLHeredoc)
	if !ok {
		t.Fatalf("expected HCLHeredoc, got %T", heredocAttr.Value)
	}
	if heredoc.Delimiter != "EOT" {
		t.Errorf("expected delimiter 'EOT', got %q", heredoc.Delimiter)
	}
	if !strings.Contains(heredoc.Value, "helpful assistant") {
		t.Errorf("heredoc content doesn't contain expected text: %q", heredoc.Value)
	}

	// name should still be parsed
	assertAttrStr(t, b, "name", "Helper")
}

func TestParseTofuOutput_MultipleBlocks(t *testing.T) {
	input := `
resource "elementum_app" "app1" {
  name = "App1"
}

resource "elementum_app" "app2" {
  name = "App2"
}

data "elementum_app" "lookup" {
  name = "External"
}
`
	blocks, err := ParseTofuOutput(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(blocks) != 3 {
		t.Fatalf("expected 3 blocks, got %d", len(blocks))
	}

	if blocks[0].Labels[1] != "app1" {
		t.Errorf("expected first block name 'app1', got %q", blocks[0].Labels[1])
	}
	if blocks[1].Labels[1] != "app2" {
		t.Errorf("expected second block name 'app2', got %q", blocks[1].Labels[1])
	}
	if blocks[2].Type != "data" {
		t.Errorf("expected third block type 'data', got %q", blocks[2].Type)
	}
}

func TestParseTofuOutput_TerraformReference(t *testing.T) {
	input := `
resource "elementum_text_field" "status" {
  app_id = elementum_app.my_app.id
  name   = "Status"
}
`
	blocks, err := ParseTofuOutput(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	b := blocks[0]

	var appIDAttr *HCLAttribute
	for _, attr := range b.Body.Attributes {
		if attr.Name == "app_id" {
			appIDAttr = attr
			break
		}
	}
	if appIDAttr == nil {
		t.Fatal("app_id attribute not found")
	}

	raw, ok := appIDAttr.Value.(HCLRaw)
	if !ok {
		t.Fatalf("expected HCLRaw for terraform ref, got %T", appIDAttr.Value)
	}
	if raw.Text != "elementum_app.my_app.id" {
		t.Errorf("expected 'elementum_app.my_app.id', got %q", raw.Text)
	}
}

func TestParseTofuOutput_Comments(t *testing.T) {
	input := `
# This is a comment
resource "elementum_app" "my_app" {
  # Inline comment
  name = "My App"
  // Another comment style
  namespace = "my-app"
}
`
	blocks, err := ParseTofuOutput(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(blocks))
	}
	assertAttrCount(t, blocks[0], 2)
}

func TestParseTofuOutput_EmptyInput(t *testing.T) {
	blocks, err := ParseTofuOutput("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(blocks) != 0 {
		t.Fatalf("expected 0 blocks, got %d", len(blocks))
	}
}

func TestParseTofuOutput_RoundTrip(t *testing.T) {
	// Generate IR, serialize, parse, serialize again - should be stable
	original := NewResourceBlock("elementum_app", "my_app")
	original.SetAttr("name", Str("My App"))
	original.SetAttr("namespace", Str("my-app"))
	original.SetAttr("enabled", Bool(true))

	serialized1 := SerializeBlocks([]*HCLBlock{original})

	blocks, err := ParseTofuOutput(serialized1)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if len(blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(blocks))
	}

	serialized2 := SerializeBlocks(blocks)

	if serialized1 != serialized2 {
		t.Errorf("round-trip mismatch:\n--- first ---\n%s\n--- second ---\n%s", serialized1, serialized2)
	}
}

func TestParseTofuOutput_InlineEmptyObject(t *testing.T) {
	input := `
resource "elementum_text_field" "test" {
  action = {}
}
`
	blocks, err := ParseTofuOutput(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	b := blocks[0]

	var actionAttr *HCLAttribute
	for _, attr := range b.Body.Attributes {
		if attr.Name == "action" {
			actionAttr = attr
			break
		}
	}
	if actionAttr == nil {
		t.Fatal("action attribute not found")
	}

	obj, ok := actionAttr.Value.(HCLObject)
	if !ok {
		t.Fatalf("expected HCLObject, got %T", actionAttr.Value)
	}
	if len(obj.Attributes) != 0 {
		t.Errorf("expected empty object, got %d attrs", len(obj.Attributes))
	}
}

func TestParseTofuOutput_Number(t *testing.T) {
	input := `
resource "elementum_text_field" "count" {
  limit  = 42
  offset = -10
  ratio  = 3.14
}
`
	blocks, err := ParseTofuOutput(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	b := blocks[0]
	assertAttrCount(t, b, 3)

	// Numbers come through as HCLRaw since we don't parse them into float64
	for _, attr := range b.Body.Attributes {
		if _, ok := attr.Value.(HCLRaw); !ok {
			t.Errorf("expected HCLRaw for number attr %q, got %T", attr.Name, attr.Value)
		}
	}
}

func TestParseTofuOutput_EscapedString(t *testing.T) {
	input := `
resource "elementum_text_field" "test" {
  description = "Line 1\nLine 2\twith tab"
  pattern     = "contains \"quotes\""
}
`
	blocks, err := ParseTofuOutput(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	b := blocks[0]

	assertAttrStr(t, b, "description", "Line 1\nLine 2\twith tab")
	assertAttrStr(t, b, "pattern", "contains \"quotes\"")
}

func TestParseTofuOutput_EmptyList(t *testing.T) {
	input := `
resource "elementum_text_field" "test" {
  items = []
}
`
	blocks, err := ParseTofuOutput(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	b := blocks[0]

	var listAttr *HCLAttribute
	for _, attr := range b.Body.Attributes {
		if attr.Name == "items" {
			listAttr = attr
			break
		}
	}
	if listAttr == nil {
		t.Fatal("items attribute not found")
	}

	list, ok := listAttr.Value.(HCLList)
	if !ok {
		t.Fatalf("expected HCLList, got %T", listAttr.Value)
	}
	if len(list.Values) != 0 {
		t.Fatalf("expected empty list, got %d elements", len(list.Values))
	}
}

func TestParseTofuOutput_FunctionCallMultiLine(t *testing.T) {
	input := `resource "elementum_agent_view" "my_view" {
  agent_id = elementum_agent.my_agent.id
  name = "My View"
  sort = jsonencode({
    orders = []
  })
}`

	blocks, err := ParseTofuOutput(input)
	if err != nil {
		t.Fatal(err)
	}
	if len(blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(blocks))
	}

	b := blocks[0]
	assertAttrCount(t, b, 3) // agent_id, name, sort

	// Find the sort attribute
	var sortAttr *HCLAttribute
	for _, attr := range b.Body.Attributes {
		if attr.Name == "sort" {
			sortAttr = attr
			break
		}
	}
	if sortAttr == nil {
		t.Fatal("sort attribute not found")
	}

	raw, ok := sortAttr.Value.(HCLRaw)
	if !ok {
		t.Fatalf("expected HCLRaw for sort, got %T", sortAttr.Value)
	}

	// The raw value should contain the full jsonencode expression
	if !strings.Contains(raw.Text, "jsonencode(") {
		t.Errorf("expected jsonencode in sort value, got: %s", raw.Text)
	}
	if !strings.Contains(raw.Text, "orders = []") {
		t.Errorf("expected 'orders = []' in sort value, got: %s", raw.Text)
	}
	if !strings.Contains(raw.Text, "})") {
		t.Errorf("expected '})' in sort value, got: %s", raw.Text)
	}

	// Serialize and verify round-trip
	output := SerializeBlocks(blocks)
	if !strings.Contains(output, "jsonencode(") {
		t.Errorf("serialized output missing jsonencode: %s", output)
	}
	if !strings.Contains(output, "orders = []") {
		t.Errorf("serialized output missing 'orders = []': %s", output)
	}
	if !strings.Contains(output, "})") {
		t.Errorf("serialized output missing '})': %s", output)
	}
}

// --- Test helpers ---

func assertAttrCount(t *testing.T, b *HCLBlock, expected int) {
	t.Helper()
	if len(b.Body.Attributes) != expected {
		t.Errorf("expected %d attributes, got %d", expected, len(b.Body.Attributes))
	}
}

func assertAttrStr(t *testing.T, b *HCLBlock, name, expected string) {
	t.Helper()
	for _, attr := range b.Body.Attributes {
		if attr.Name == name {
			if str, ok := attr.Value.(HCLString); ok {
				if str.Value != expected {
					t.Errorf("attr %q: expected %q, got %q", name, expected, str.Value)
				}
				return
			}
			t.Errorf("attr %q: expected HCLString, got %T", name, attr.Value)
			return
		}
	}
	t.Errorf("attr %q not found", name)
}
