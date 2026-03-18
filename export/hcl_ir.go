// Copyright (c) ElementumAI, Inc.
// SPDX-License-Identifier: MPL-2.0

package export

import "strings"

// ============================================================================
// HCL Intermediate Representation (IR)
//
// This package defines a structured representation of HCL configuration that
// replaces raw string concatenation. Generators produce typed IR nodes,
// transforms operate on the tree, and serialization to text happens once
// at the end via SerializeBlocks.
// ============================================================================

// HCLBlock is the core IR node representing an HCL block.
// Top-level blocks (resource, data, locals, import) have Type and Labels.
// Nested blocks (trigger, filter, columns) have Type and optionally Labels.
type HCLBlock struct {
	Type   string    // "resource", "data", "locals", "import", "variable", "output", or nested block type
	Labels []string  // e.g., ["elementum_app", "my_app"] for resource blocks
	Body   *HCLBody  // Block body with attributes and nested blocks
	Meta   BlockMeta // Not serialized -- carries routing/transform metadata
}

// HCLBody holds the ordered contents of a block body.
type HCLBody struct {
	Attributes []*HCLAttribute // Ordered (insertion order preserved)
	Blocks     []*HCLBlock     // Nested blocks (trigger {}, filter {}, etc.)
}

// HCLAttribute represents a single name = value pair in a block body.
type HCLAttribute struct {
	Name    string
	Value   HCLValue
	Comment string // Optional inline comment (e.g., "# TODO: ...")
}

// HCLValue is the sum type for all possible HCL values.
type HCLValue interface {
	hclValue() // marker method -- makes the interface closed
}

// HCLString represents a quoted string literal: "hello"
type HCLString struct{ Value string }

// HCLHeredoc represents a heredoc string: <<-EOT ... EOT
type HCLHeredoc struct {
	Value     string
	Delimiter string // e.g., "EOT"
}

// HCLNumber represents a numeric literal: 42, 3.14
type HCLNumber struct{ Value float64 }

// HCLBool represents a boolean literal: true/false
type HCLBool struct{ Value bool }

// HCLNull represents the null literal
type HCLNull struct{}

// HCLReference represents an unquoted Terraform expression: elementum_app.my_app.id
type HCLReference struct{ Ref string }

// HCLInterpolated represents a string with embedded expressions:
// "text ${ref} text"
type HCLInterpolated struct{ Parts []InterpolationPart }

// HCLList represents a list value: [a, b, c]
type HCLList struct{ Values []HCLValue }

// HCLObject represents an inline object value: { key = val }
type HCLObject struct{ Attributes []*HCLAttribute }

// HCLRaw represents raw HCL text that is emitted as-is.
// This is an escape hatch for complex expressions that don't fit the IR.
type HCLRaw struct{ Text string }

// InterpolationPart represents one segment of an interpolated string.
// Either Literal or Reference is non-empty, not both.
type InterpolationPart struct {
	Literal   string // Static text segment
	Reference string // Terraform expression (the content between ${...})
}

// BlockMeta carries non-serialized metadata used for routing, grouping,
// and deduplication in the export pipeline.
type BlockMeta struct {
	ResourceType string // e.g., "elementum_app"
	ResourceName string // e.g., "my_app"
	Category     string // "app", "automation", "agent", "element", "table", etc.
	GroupKey     string // For file grouping (automation name, element handle, etc.)
	SourceID     string // Primary resource ID (for dedup)
}

// marker method implementations
func (HCLString) hclValue()       {}
func (HCLHeredoc) hclValue()      {}
func (HCLNumber) hclValue()       {}
func (HCLBool) hclValue()         {}
func (HCLNull) hclValue()         {}
func (HCLReference) hclValue()    {}
func (HCLInterpolated) hclValue() {}
func (HCLList) hclValue()         {}
func (HCLObject) hclValue()       {}
func (HCLRaw) hclValue()          {}

// ============================================================================
// Constructors -- convenience helpers for building IR nodes
// ============================================================================

// NewBlock creates a new HCLBlock with the given type and labels.
func NewBlock(blockType string, labels ...string) *HCLBlock {
	return &HCLBlock{
		Type:   blockType,
		Labels: labels,
		Body:   &HCLBody{},
	}
}

// NewResourceBlock creates a new resource block: resource "type" "name" { ... }
func NewResourceBlock(resourceType, resourceName string) *HCLBlock {
	b := NewBlock("resource", resourceType, resourceName)
	b.Meta = BlockMeta{
		ResourceType: resourceType,
		ResourceName: resourceName,
	}
	return b
}

// NewDataBlock creates a new data source block: data "type" "name" { ... }
func NewDataBlock(dataType, dataName string) *HCLBlock {
	return NewBlock("data", dataType, dataName)
}

// NewImportBlock creates a new import block: import { ... }
func NewImportBlock() *HCLBlock {
	return NewBlock("import")
}

// SetAttr adds or replaces an attribute on the block body.
func (b *HCLBlock) SetAttr(name string, value HCLValue) *HCLBlock {
	if b.Body == nil {
		b.Body = &HCLBody{}
	}
	b.Body.Attributes = append(b.Body.Attributes, &HCLAttribute{
		Name:  name,
		Value: value,
	})
	return b
}

// SetAttrComment adds an attribute with an inline comment.
func (b *HCLBlock) SetAttrComment(name string, value HCLValue, comment string) *HCLBlock {
	if b.Body == nil {
		b.Body = &HCLBody{}
	}
	b.Body.Attributes = append(b.Body.Attributes, &HCLAttribute{
		Name:    name,
		Value:   value,
		Comment: comment,
	})
	return b
}

// AddBlock appends a nested block to the body.
func (b *HCLBlock) AddBlock(child *HCLBlock) *HCLBlock {
	if b.Body == nil {
		b.Body = &HCLBody{}
	}
	b.Body.Blocks = append(b.Body.Blocks, child)
	return b
}

// Str creates an HCLString value.
func Str(s string) HCLString { return HCLString{Value: s} }

// Ref creates an HCLReference value.
func Ref(r string) HCLReference { return HCLReference{Ref: r} }

// Num creates an HCLNumber value.
func Num(n float64) HCLNumber { return HCLNumber{Value: n} }

// Bool creates an HCLBool value.
func Bool(b bool) HCLBool { return HCLBool{Value: b} }

// Null creates an HCLNull value.
func Null() HCLNull { return HCLNull{} }

// Raw creates an HCLRaw value (escape hatch).
func Raw(text string) HCLRaw { return HCLRaw{Text: text} }

// List creates an HCLList from values.
func List(values ...HCLValue) HCLList { return HCLList{Values: values} }

// Heredoc creates an HCLHeredoc value.
func Heredoc(value, delimiter string) HCLHeredoc {
	return HCLHeredoc{Value: value, Delimiter: delimiter}
}

// Obj creates an HCLObject from key-value pairs.
func Obj(attrs ...*HCLAttribute) HCLObject {
	return HCLObject{Attributes: attrs}
}

// Attr creates an HCLAttribute (for use with Obj).
func Attr(name string, value HCLValue) *HCLAttribute {
	return &HCLAttribute{Name: name, Value: value}
}

// AttrComment creates an HCLAttribute with a comment.
func AttrComment(name string, value HCLValue, comment string) *HCLAttribute {
	return &HCLAttribute{Name: name, Value: value, Comment: comment}
}

// refOrStr converts a resolved reference string to the appropriate HCLValue.
// If the string starts with a quote (i.e., it's a quoted UUID fallback), it becomes HCLString.
// Otherwise it's treated as an unquoted Terraform reference.
func refOrStr(s string) HCLValue {
	if strings.HasPrefix(s, "\"") {
		return Str(strings.Trim(s, "\""))
	}
	return Ref(s)
}
