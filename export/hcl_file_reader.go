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
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/elementumltd/elementum-cli/discovery"
)

// FileReaderHCLGenerator generates HCL for all file reader resource types
// Supports: elementum_ai_file_reader, elementum_text_file_reader, elementum_json_file_reader, elementum_xml_file_reader
type FileReaderHCLGenerator struct {
	app     *discovery.App
	imports []ImportBlock
	uuidMap map[string]string
}

// NewFileReaderHCLGenerator creates a new file reader HCL generator
func NewFileReaderHCLGenerator(app *discovery.App, imports []ImportBlock, uuidMap map[string]string) *FileReaderHCLGenerator {
	return &FileReaderHCLGenerator{
		app:     app,
		imports: imports,
		uuidMap: uuidMap,
	}
}

// GenerateAll generates HCL for all file readers in the app
func (g *FileReaderHCLGenerator) GenerateAll() string {
	return SerializeBlocks(g.GenerateAllIR())
}

// GenerateAllIR generates IR blocks for all file readers in the app
func (g *FileReaderHCLGenerator) GenerateAllIR() []*HCLBlock {
	if g.app == nil || len(g.app.AIFileReaders) == 0 {
		return nil
	}

	appResourceName := AppResourceName(g.app)
	var blocks []*HCLBlock

	for _, reader := range g.app.AIFileReaders {
		b := g.GenerateFileReaderIR(&reader, appResourceName)
		if b != nil {
			blocks = append(blocks, b)
		}
	}

	return blocks
}

// GenerateFileReaderIR generates an IR block for a single file reader resource
func (g *FileReaderHCLGenerator) GenerateFileReaderIR(reader *discovery.FileReader, appResourceName string) *HCLBlock {
	resourceType := reader.TerraformResourceType()

	// Find the resource name from imports
	var resourceName string
	for _, imp := range g.imports {
		if imp.ResourceType == resourceType && strings.Contains(imp.ID, reader.ID) {
			resourceName = imp.ResourceName
			break
		}
	}
	if resourceName == "" {
		return nil
	}

	appRef := Ref("elementum_app." + appResourceName + ".id")

	switch reader.Type {
	case discovery.FileReaderTypeAI:
		return g.generateAIFileReaderIR(reader, resourceName, appRef)
	case discovery.FileReaderTypeOCR:
		return g.generateTextFileReaderIR(reader, resourceName, appRef)
	case discovery.FileReaderTypeJSON:
		return g.generateJSONFileReaderIR(reader, resourceName, appRef)
	case discovery.FileReaderTypeXML:
		return g.generateXMLFileReaderIR(reader, resourceName, appRef)
	default:
		return g.generateAIFileReaderIR(reader, resourceName, appRef)
	}
}

func (g *FileReaderHCLGenerator) generateAIFileReaderIR(reader *discovery.FileReader, resourceName string, appRef HCLReference) *HCLBlock {
	b := NewResourceBlock("elementum_ai_file_reader", resourceName)
	b.SetAttr("app_id", appRef)
	b.SetAttr("name", Str(reader.Name))

	if reader.Instructions != "" {
		b.SetAttr("instructions", Str(reader.Instructions))
	}

	if len(reader.Fields) > 0 {
		var fieldObjs []HCLValue
		for _, field := range reader.Fields {
			attrs := []*HCLAttribute{
				Attr("name", Str(field.Name)),
				Attr("description", Str(field.Description)),
				Attr("type", Str(field.Type)),
			}
			if !field.Required {
				attrs = append(attrs, Attr("required", Bool(false)))
			}
			fieldObjs = append(fieldObjs, Obj(attrs...))
		}
		b.SetAttr("fields", HCLList{Values: fieldObjs})
	}

	return b
}

func (g *FileReaderHCLGenerator) generateTextFileReaderIR(reader *discovery.FileReader, resourceName string, appRef HCLReference) *HCLBlock {
	b := NewResourceBlock("elementum_text_file_reader", resourceName)
	b.SetAttr("app_id", appRef)
	b.SetAttr("name", Str(reader.Name))
	return b
}

func (g *FileReaderHCLGenerator) generateJSONFileReaderIR(reader *discovery.FileReader, resourceName string, appRef HCLReference) *HCLBlock {
	b := NewResourceBlock("elementum_json_file_reader", resourceName)
	b.SetAttr("app_id", appRef)
	b.SetAttr("name", Str(reader.Name))
	// Truth convention: `structure = jsonencode({ properties = [...] })`
	// with HCL-identifier keys (`properties = [...]`, `bool = { name = ... }`)
	// rather than JSON-quoted keys (`"properties": [...]`). Emit by decoding
	// the reshaped JSON and rendering as an HCL object literal — both forms
	// round-trip to the same JSON at plan time, but matching the HCL form
	// keeps byte-level diffs minimal on roundtrip and works cleanly with
	// `tofu fmt`'s indentation.
	if reader.Structure != "" {
		if hcl := renderJSONAsHCL(reader.Structure, 1); hcl != "" {
			b.SetAttr("structure", Raw("jsonencode("+hcl+")"))
		}
	}
	return b
}

// renderJSONAsHCL decodes a JSON string and renders it as an HCL value
// literal suitable for placement inside a `jsonencode(...)` call. Object
// keys that are valid HCL identifiers are emitted bare (`key = value`);
// others fall back to quoted form (`"key-with-dashes" = value`). `indent`
// is the nesting level of the caller (0 = top of a file).
func renderJSONAsHCL(raw string, indent int) string {
	var v interface{}
	if err := json.Unmarshal([]byte(raw), &v); err != nil {
		// Fall back to the raw JSON — still valid HCL, just ugly.
		return raw
	}
	var sb strings.Builder
	writeJSONValueAsHCL(&sb, v, indent)
	return sb.String()
}

func writeJSONValueAsHCL(sb *strings.Builder, v interface{}, indent int) {
	switch x := v.(type) {
	case map[string]interface{}:
		if len(x) == 0 {
			sb.WriteString("{}")
			return
		}
		// Small objects with only primitive-or-small-map values render on a
		// single line: `{ bool = { name = "x" } }`. Truth uses this for
		// JSON-structure property descriptors; expanding them to multi-line
		// balloons the diff 3-4×.
		if inline := tryInlineMap(x); inline != "" {
			sb.WriteString(inline)
			return
		}
		sb.WriteString("{\n")
		// Sort keys for determinism — matches the existing JSON path which
		// depended on Go's map iteration order and just happened to match
		// truth in some cases.
		keys := make([]string, 0, len(x))
		for k := range x {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		inner := strings.Repeat("  ", indent+1)
		for _, k := range keys {
			sb.WriteString(inner)
			sb.WriteString(hclKey(k))
			sb.WriteString(" = ")
			writeJSONValueAsHCL(sb, x[k], indent+1)
			sb.WriteString("\n")
		}
		sb.WriteString(strings.Repeat("  ", indent))
		sb.WriteString("}")
	case []interface{}:
		if len(x) == 0 {
			sb.WriteString("[]")
			return
		}
		sb.WriteString("[\n")
		inner := strings.Repeat("  ", indent+1)
		for _, item := range x {
			sb.WriteString(inner)
			writeJSONValueAsHCL(sb, item, indent+1)
			// Always add trailing comma — matches truth's HCL convention
			// and is valid for both HCL lists and JSON arrays (jsonencode
			// doesn't care about trailing commas in its HCL input).
			sb.WriteString(",\n")
		}
		sb.WriteString(strings.Repeat("  ", indent))
		sb.WriteString("]")
	case string:
		fmt.Fprintf(sb, "%q", x)
	case bool:
		if x {
			sb.WriteString("true")
		} else {
			sb.WriteString("false")
		}
	case float64:
		// Integers render without trailing decimals.
		if x == float64(int64(x)) {
			fmt.Fprintf(sb, "%d", int64(x))
		} else {
			fmt.Fprintf(sb, "%g", x)
		}
	case nil:
		sb.WriteString("null")
	default:
		// Unknown — emit as JSON string for safety.
		if b, err := json.Marshal(x); err == nil {
			sb.Write(b)
		} else {
			sb.WriteString("null")
		}
	}
}

// tryInlineMap returns a single-line HCL representation of a "small" map,
// or "" if the map is too large / nested to inline cleanly. The heuristic
// matches truth's JSON-structure property descriptor style:
//
//	{ bool = { name = "x" } }
//	{ text = { name = "x", format = "iso-8601" } }
//
// Rules:
//   - Map has at most 2 keys at the top level
//   - Values are primitives OR a map with up to 3 primitive values
//   - Inline only if the rendered string is ≤ 120 chars
func tryInlineMap(m map[string]interface{}) string {
	if len(m) > 2 {
		return ""
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var parts strings.Builder
	parts.WriteString("{ ")
	for i, k := range keys {
		parts.WriteString(hclKey(k))
		parts.WriteString(" = ")
		if !writeInlineValue(&parts, m[k]) {
			return ""
		}
		if i < len(keys)-1 {
			parts.WriteString(", ")
		}
	}
	parts.WriteString(" }")
	s := parts.String()
	if len(s) > 120 {
		return ""
	}
	return s
}

// writeInlineValue appends a single-line HCL representation of v and
// returns true. For maps it recurses through tryInlineMap once — if the
// nested map isn't itself inlineable, writeInlineValue returns false so
// the caller falls back to the block form.
func writeInlineValue(sb *strings.Builder, v interface{}) bool {
	switch x := v.(type) {
	case map[string]interface{}:
		inline := tryInlineMap(x)
		if inline == "" {
			return false
		}
		sb.WriteString(inline)
		return true
	case []interface{}:
		// Arrays force multi-line.
		return false
	case string:
		fmt.Fprintf(sb, "%q", x)
		return true
	case bool:
		if x {
			sb.WriteString("true")
		} else {
			sb.WriteString("false")
		}
		return true
	case float64:
		if x == float64(int64(x)) {
			fmt.Fprintf(sb, "%d", int64(x))
		} else {
			fmt.Fprintf(sb, "%g", x)
		}
		return true
	case nil:
		sb.WriteString("null")
		return true
	}
	return false
}

// hclKey returns an HCL object-key representation of a JSON key — bare if
// the key is a valid HCL identifier, quoted otherwise.
func hclKey(k string) string {
	if isHCLIdentifier(k) {
		return k
	}
	return fmt.Sprintf("%q", k)
}

// isHCLIdentifier reports whether s can appear unquoted as an HCL object
// key. HCL object keys that parse as a bare identifier match
// `[A-Za-z_][A-Za-z0-9_]*` — hyphens are NOT allowed because inside an
// object literal `{my-key = "x"}` parses as `{my - key = "x"}` (a
// subtraction expression between two variable refs). Keys that need a
// hyphen must be quoted: `{"my-key" = "x"}`.
func isHCLIdentifier(s string) bool {
	if s == "" {
		return false
	}
	for i, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r == '_':
			// always valid
		case r >= '0' && r <= '9':
			if i == 0 {
				return false
			}
		default:
			return false
		}
	}
	return true
}

func (g *FileReaderHCLGenerator) generateXMLFileReaderIR(reader *discovery.FileReader, resourceName string, appRef HCLReference) *HCLBlock {
	b := NewResourceBlock("elementum_xml_file_reader", resourceName)
	b.SetAttr("app_id", appRef)
	b.SetAttr("name", Str(reader.Name))
	if reader.Structure != "" {
		b.SetAttr("values", Str(reader.Structure))
	}
	return b
}
