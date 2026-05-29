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
	"fmt"
	"math"
	"strings"
)

// SerializeBlocks serializes a slice of top-level HCL blocks to a string,
// separated by blank lines. This is the main entry point for converting
// IR to HCL text.
func SerializeBlocks(blocks []*HCLBlock) string {
	if len(blocks) == 0 {
		return ""
	}

	var sb strings.Builder
	for i, block := range blocks {
		if i > 0 {
			sb.WriteString("\n")
		}
		serializeBlock(&sb, block, 0)
	}
	return sb.String()
}

// SerializeBlock serializes a single block at the given indentation level.
func SerializeBlock(block *HCLBlock, indent int) string {
	var sb strings.Builder
	serializeBlock(&sb, block, indent)
	return sb.String()
}

// serializeBlock writes a single block to the builder at the given indent level.
func serializeBlock(sb *strings.Builder, block *HCLBlock, indent int) {
	indentStr := strings.Repeat("  ", indent)

	// Write block header: type "label1" "label2" {
	sb.WriteString(indentStr)
	sb.WriteString(block.Type)
	for _, label := range block.Labels {
		fmt.Fprintf(sb, " %q", label)
	}
	sb.WriteString(" {\n")

	// Write body
	if block.Body != nil {
		serializeBody(sb, block.Body, indent+1)
	}

	// Write closing brace
	sb.WriteString(indentStr)
	sb.WriteString("}\n")
}

// serializeBody writes the body of a block (attributes + nested blocks).
func serializeBody(sb *strings.Builder, body *HCLBody, indent int) {
	indentStr := strings.Repeat("  ", indent)

	// Write attributes
	for _, attr := range body.Attributes {
		sb.WriteString(indentStr)
		sb.WriteString(attr.Name)
		sb.WriteString(" = ")
		serializeValue(sb, attr.Value, indent)
		if attr.Comment != "" {
			sb.WriteString(" ")
			sb.WriteString(attr.Comment)
		}
		sb.WriteString("\n")
	}

	// Write nested blocks
	for i, block := range body.Blocks {
		// Add blank line before nested block if there are attributes before it,
		// or between consecutive nested blocks
		if len(body.Attributes) > 0 || i > 0 {
			sb.WriteString("\n")
		}
		serializeBlock(sb, block, indent)
	}
}

// serializeValue writes a single HCL value to the builder.
func serializeValue(sb *strings.Builder, value HCLValue, indent int) {
	switch v := value.(type) {
	case HCLString:
		fmt.Fprintf(sb, "%q", v.Value)

	case HCLHeredoc:
		delimiter := v.Delimiter
		if delimiter == "" {
			delimiter = "EOT"
		}
		sb.WriteString("<<-")
		sb.WriteString(delimiter)
		sb.WriteString("\n")
		// Heredoc content sits one level DEEPER than the attribute itself.
		// `<<-` strips the minimum leading whitespace, so this is cosmetic
		// (`tofu fmt` leaves heredoc bodies alone), but the extra indent
		// matches the hand-authored truth convention and keeps byte-diffs
		// minimal on round-trip.
		heredocIndent := strings.Repeat("  ", indent+1)
		lines := strings.Split(v.Value, "\n")
		for _, line := range lines {
			if line == "" {
				sb.WriteString("\n")
			} else {
				sb.WriteString(heredocIndent)
				sb.WriteString(line)
				sb.WriteString("\n")
			}
		}
		// Closing delimiter lines up with the attribute, not the body.
		sb.WriteString(strings.Repeat("  ", indent))
		sb.WriteString(delimiter)

	case HCLNumber:
		// Format as integer if it's a whole number, otherwise as float
		if v.Value == math.Trunc(v.Value) && !math.IsInf(v.Value, 0) {
			fmt.Fprintf(sb, "%d", int64(v.Value))
		} else {
			fmt.Fprintf(sb, "%g", v.Value)
		}

	case HCLBool:
		if v.Value {
			sb.WriteString("true")
		} else {
			sb.WriteString("false")
		}

	case HCLNull:
		sb.WriteString("null")

	case HCLReference:
		sb.WriteString(v.Ref)

	case HCLInterpolated:
		sb.WriteString("\"")
		for _, part := range v.Parts {
			if part.Reference != "" {
				sb.WriteString("${")
				sb.WriteString(part.Reference)
				sb.WriteString("}")
			}
			if part.Literal != "" {
				// Escape special characters in literal parts
				escaped := strings.ReplaceAll(part.Literal, `\`, `\\`)
				escaped = strings.ReplaceAll(escaped, `"`, `\"`)
				escaped = strings.ReplaceAll(escaped, "\n", `\n`)
				sb.WriteString(escaped)
			}
		}
		sb.WriteString("\"")

	case HCLList:
		if len(v.Values) == 0 {
			sb.WriteString("[]")
			return
		}

		// Check if all values are simple (strings, numbers, bools, refs) for inline formatting
		if isSimpleList(v.Values) && len(v.Values) <= 5 {
			sb.WriteString("[")
			for i, val := range v.Values {
				if i > 0 {
					sb.WriteString(", ")
				}
				serializeValue(sb, val, indent)
			}
			sb.WriteString("]")
			return
		}

		// Multi-line list
		sb.WriteString("[\n")
		innerIndent := strings.Repeat("  ", indent+1)
		for _, val := range v.Values {
			sb.WriteString(innerIndent)
			serializeValue(sb, val, indent+1)
			sb.WriteString(",\n")
		}
		sb.WriteString(strings.Repeat("  ", indent))
		sb.WriteString("]")

	case HCLObject:
		if len(v.Attributes) == 0 {
			sb.WriteString("{}")
			return
		}

		sb.WriteString("{\n")
		innerIndent := strings.Repeat("  ", indent+1)
		for _, attr := range v.Attributes {
			sb.WriteString(innerIndent)
			sb.WriteString(attr.Name)
			sb.WriteString(" = ")
			serializeValue(sb, attr.Value, indent+1)
			if attr.Comment != "" {
				sb.WriteString(" ")
				sb.WriteString(attr.Comment)
			}
			sb.WriteString("\n")
		}
		sb.WriteString(strings.Repeat("  ", indent))
		sb.WriteString("}")

	case HCLRaw:
		sb.WriteString(v.Text)
	}
}

// isSimpleList returns true if all values in the list are simple scalar types.
func isSimpleList(values []HCLValue) bool {
	for _, v := range values {
		switch v.(type) {
		case HCLString, HCLNumber, HCLBool, HCLReference:
			// simple
		default:
			return false
		}
	}
	return true
}
