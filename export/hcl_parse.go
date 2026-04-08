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
	"unicode"
)

// ParseTofuOutput parses the text output of `tofu plan -generate-config-out`
// into IR blocks. This enables both the tofu-generated HCL and the CLI-generated
// HCL to be represented as []*HCLBlock, unifying the two-track generation split.
//
// The parser handles standard HCL resource/data blocks with:
//   - Simple attributes: name = "value", count = 42, enabled = true
//   - Object attributes: config = { key = "value" }
//   - List attributes: items = ["a", "b"], items = [{ key = "val" }]
//   - Nested blocks: trigger { ... }
//   - Heredoc strings: <<-EOT ... EOT
//   - Comments: # ... and // ...
func ParseTofuOutput(text string) ([]*HCLBlock, error) {
	p := &hclParser{
		lines: strings.Split(text, "\n"),
		pos:   0,
	}
	return p.parseTopLevel(), nil
}

// hclParser is a line-based parser for well-structured HCL output.
type hclParser struct {
	lines []string
	pos   int
}

func (p *hclParser) parseTopLevel() []*HCLBlock {
	var blocks []*HCLBlock
	for p.pos < len(p.lines) {
		line := p.lines[p.pos]
		trimmed := strings.TrimSpace(line)

		// Skip empty lines and comments
		if trimmed == "" || strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, "//") {
			p.pos++
			continue
		}

		// Try to parse a block (resource, data, or unlabeled)
		if block := p.tryParseBlock(trimmed); block != nil {
			blocks = append(blocks, block)
			continue
		}

		// Skip unrecognized lines
		p.pos++
	}
	return blocks
}

// tryParseBlock attempts to parse a block starting at the current line.
// Returns nil if the line doesn't start a block.
func (p *hclParser) tryParseBlock(trimmed string) *HCLBlock {
	// resource "type" "name" {
	if strings.HasPrefix(trimmed, "resource ") {
		return p.parseResourceOrDataBlock(trimmed, "resource")
	}
	// data "type" "name" {
	if strings.HasPrefix(trimmed, "data ") {
		return p.parseResourceOrDataBlock(trimmed, "data")
	}
	// unlabeled block: identifier {
	if strings.HasSuffix(trimmed, "{") && !strings.Contains(trimmed, "=") {
		return p.parseUnlabeledBlock(trimmed)
	}
	return nil
}

// parseResourceOrDataBlock parses: resource "type" "name" { ... }
func (p *hclParser) parseResourceOrDataBlock(trimmed, blockType string) *HCLBlock {
	// Extract labels from: resource "type" "name" {
	rest := strings.TrimPrefix(trimmed, blockType)
	rest = strings.TrimSpace(rest)

	labels := extractQuotedLabels(rest)
	if len(labels) < 2 {
		p.pos++
		return nil
	}

	b := &HCLBlock{
		Type:   blockType,
		Labels: labels,
		Body:   &HCLBody{},
	}

	if blockType == "resource" && len(labels) >= 2 {
		b.Meta = BlockMeta{
			ResourceType: labels[0],
			ResourceName: labels[1],
		}
	}

	p.pos++ // consume the opening line
	p.parseBody(b.Body)
	return b
}

// parseUnlabeledBlock parses: identifier { ... } or identifier "label" { ... }
func (p *hclParser) parseUnlabeledBlock(trimmed string) *HCLBlock {
	// Split into identifier and optional labels
	parts := strings.TrimSuffix(trimmed, "{")
	parts = strings.TrimSpace(parts)

	tokens := splitBlockHeader(parts)
	if len(tokens) == 0 {
		p.pos++
		return nil
	}

	b := &HCLBlock{
		Type:   tokens[0],
		Labels: extractQuotedLabels(strings.Join(tokens[1:], " ")),
		Body:   &HCLBody{},
	}

	p.pos++ // consume the opening line
	p.parseBody(b.Body)
	return b
}

// parseBody parses the contents of a block body until a closing brace is found.
func (p *hclParser) parseBody(body *HCLBody) {
	for p.pos < len(p.lines) {
		line := p.lines[p.pos]
		trimmed := strings.TrimSpace(line)

		// Closing brace ends this body
		if trimmed == "}" || trimmed == "}," {
			p.pos++
			return
		}

		// Skip empty lines and comments
		if trimmed == "" || strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, "//") {
			p.pos++
			continue
		}

		// Check for nested block (identifier { on its own line, no = sign)
		if strings.HasSuffix(trimmed, "{") && !strings.Contains(trimmed, "=") {
			nested := p.parseUnlabeledBlock(trimmed)
			if nested != nil {
				body.Blocks = append(body.Blocks, nested)
			}
			continue
		}

		// Check for attribute: name = value
		if eqIdx := findAttributeEquals(trimmed); eqIdx > 0 {
			attrName := strings.TrimSpace(trimmed[:eqIdx])
			attrValue := strings.TrimSpace(trimmed[eqIdx+1:])

			val := p.parseValue(attrValue)
			if val != nil {
				body.Attributes = append(body.Attributes, &HCLAttribute{
					Name:  attrName,
					Value: val,
				})
			}
			continue
		}

		// Unrecognized line, skip
		p.pos++
	}
}

// parseValue parses an HCL value starting from the given string.
// The current line (p.pos) has already been consumed for the attribute name;
// this processes the value portion.
func (p *hclParser) parseValue(s string) HCLValue {
	s = strings.TrimSpace(s)

	// Heredoc: <<-DELIM or <<DELIM
	if strings.HasPrefix(s, "<<-") || strings.HasPrefix(s, "<<") {
		return p.parseHeredoc(s)
	}

	// Quoted string
	if strings.HasPrefix(s, "\"") {
		str, rest := parseQuotedString(s)
		_ = rest
		p.pos++
		return Str(str)
	}

	// List: [...]
	if strings.HasPrefix(s, "[") {
		val := p.parseList(s)
		return val
	}

	// Object: { ... }
	if strings.HasPrefix(s, "{") {
		val := p.parseObject(s)
		return val
	}

	// Function call with multi-line body: jsonencode({ ... }), toset([...]), etc.
	if isFunctionCallStart(s) {
		return p.parseFunctionCall(s)
	}

	// Boolean
	if s == "true" || s == "true," {
		p.pos++
		return Bool(true)
	}
	if s == "false" || s == "false," {
		p.pos++
		return Bool(false)
	}

	// Null
	if s == "null" || s == "null," {
		p.pos++
		return HCLNull{}
	}

	// Number
	if isNumber(s) {
		p.pos++
		return Raw(strings.TrimSuffix(s, ","))
	}

	// Terraform reference or expression (unquoted, not a keyword)
	p.pos++
	return Raw(strings.TrimSuffix(s, ","))
}

// parseHeredoc parses a heredoc string (<<-EOT ... EOT)
func (p *hclParser) parseHeredoc(s string) HCLValue {
	// Extract delimiter
	delim := s
	if strings.HasPrefix(delim, "<<-") {
		delim = strings.TrimPrefix(delim, "<<-")
	} else {
		delim = strings.TrimPrefix(delim, "<<")
	}
	delim = strings.TrimSpace(delim)

	p.pos++ // consume the <<-DELIM line

	var lines []string
	for p.pos < len(p.lines) {
		line := p.lines[p.pos]
		trimmedLine := strings.TrimSpace(line)
		p.pos++

		if trimmedLine == delim {
			break
		}
		lines = append(lines, line)
	}

	// Determine minimum indentation for stripping
	content := strings.Join(lines, "\n")
	// Strip common leading whitespace if <<- was used
	if strings.HasPrefix(s, "<<-") {
		content = stripHeredocIndent(content)
	}

	return Heredoc(content, delim)
}

// parseList parses a list value starting with [
func (p *hclParser) parseList(s string) HCLValue {
	// Simple single-line list: [val1, val2, ...]
	if isCompleteBracketedExpression(s, '[', ']') {
		p.pos++
		inner := s[1 : len(s)-1]
		// Remove trailing comma if present (e.g., from "],")
		inner = strings.TrimSuffix(inner, ",")
		if rIdx := strings.LastIndex(s, "]"); rIdx > 0 {
			inner = s[1:rIdx]
		}
		inner = strings.TrimSpace(inner)
		if inner == "" {
			return List()
		}
		return p.parseInlineListElements(inner)
	}

	// Multi-line list
	p.pos++ // consume the [ line
	var elements []HCLValue
	for p.pos < len(p.lines) {
		line := p.lines[p.pos]
		trimmed := strings.TrimSpace(line)

		if trimmed == "]" || trimmed == "]," {
			p.pos++
			break
		}

		if trimmed == "" || strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, "//") {
			p.pos++
			continue
		}

		// Object element: { ... }
		if strings.HasPrefix(trimmed, "{") {
			obj := p.parseObject(trimmed)
			if obj != nil {
				elements = append(elements, obj)
			}
			continue
		}

		// Quoted string element
		if strings.HasPrefix(trimmed, "\"") {
			str, _ := parseQuotedString(trimmed)
			elements = append(elements, Str(str))
			p.pos++
			continue
		}

		// Other values (numbers, refs, booleans)
		val := strings.TrimSuffix(trimmed, ",")
		if val == "true" {
			elements = append(elements, Bool(true))
		} else if val == "false" {
			elements = append(elements, Bool(false))
		} else if val == "null" {
			elements = append(elements, HCLNull{})
		} else if isNumber(val) {
			elements = append(elements, Raw(val))
		} else {
			elements = append(elements, Raw(val))
		}
		p.pos++
	}

	return HCLList{Values: elements}
}

// parseObject parses an object value starting with {
func (p *hclParser) parseObject(s string) HCLValue {
	// Simple single-line object: { key = "val" } or {}
	if isCompleteBracketedExpression(s, '{', '}') {
		p.pos++
		inner := s[1 : len(s)-1]
		if rIdx := strings.LastIndex(s, "}"); rIdx > 0 {
			inner = s[1:rIdx]
		}
		inner = strings.TrimSpace(inner)
		if inner == "" {
			return Obj()
		}
		return p.parseInlineObjectAttrs(inner)
	}

	// Multi-line object
	p.pos++ // consume the { line
	var attrs []*HCLAttribute
	for p.pos < len(p.lines) {
		line := p.lines[p.pos]
		trimmed := strings.TrimSpace(line)

		if trimmed == "}" || trimmed == "}," {
			p.pos++
			break
		}

		if trimmed == "" || strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, "//") {
			p.pos++
			continue
		}

		// Nested block within object (e.g., nested { ... })
		if strings.HasSuffix(trimmed, "{") && !strings.Contains(trimmed, "=") {
			// This is unusual in tofu output but handle gracefully
			p.skipBlock()
			continue
		}

		// Attribute: key = value
		if eqIdx := findAttributeEquals(trimmed); eqIdx > 0 {
			attrName := strings.TrimSpace(trimmed[:eqIdx])
			attrValue := strings.TrimSpace(trimmed[eqIdx+1:])
			val := p.parseValue(attrValue)
			if val != nil {
				attrs = append(attrs, Attr(attrName, val))
			}
			continue
		}

		p.pos++
	}

	return Obj(attrs...)
}

// parseInlineListElements parses comma-separated values in a single-line list
func (p *hclParser) parseInlineListElements(s string) HCLValue {
	parts := splitRespectingQuotes(s, ',')
	var elements []HCLValue
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if strings.HasPrefix(part, "\"") {
			str, _ := parseQuotedString(part)
			elements = append(elements, Str(str))
		} else if part == "true" {
			elements = append(elements, Bool(true))
		} else if part == "false" {
			elements = append(elements, Bool(false))
		} else if part == "null" {
			elements = append(elements, HCLNull{})
		} else if isNumber(part) {
			elements = append(elements, Raw(part))
		} else {
			elements = append(elements, Raw(part))
		}
	}
	return HCLList{Values: elements}
}

// parseInlineObjectAttrs parses key = value pairs in a single-line object
func (p *hclParser) parseInlineObjectAttrs(s string) HCLValue {
	// Split on commas while respecting nested structures
	parts := splitRespectingQuotes(s, ',')
	var attrs []*HCLAttribute
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if eqIdx := findAttributeEquals(part); eqIdx > 0 {
			name := strings.TrimSpace(part[:eqIdx])
			valStr := strings.TrimSpace(part[eqIdx+1:])
			var val HCLValue
			if strings.HasPrefix(valStr, "\"") {
				str, _ := parseQuotedString(valStr)
				val = Str(str)
			} else if valStr == "true" {
				val = Bool(true)
			} else if valStr == "false" {
				val = Bool(false)
			} else if valStr == "null" {
				val = HCLNull{}
			} else if isNumber(valStr) {
				val = Raw(valStr)
			} else {
				val = Raw(valStr)
			}
			attrs = append(attrs, Attr(name, val))
		}
	}
	return Obj(attrs...)
}

// isFunctionCallStart checks if a string looks like the start of a function call
// that may span multiple lines, e.g. "jsonencode({", "toset([", "merge({"
func isFunctionCallStart(s string) bool {
	parenIdx := strings.Index(s, "(")
	if parenIdx <= 0 {
		return false
	}
	// The part before ( should be a valid identifier (alphanumeric + underscore)
	name := s[:parenIdx]
	for _, ch := range name {
		if (ch < 'a' || ch > 'z') && (ch < 'A' || ch > 'Z') && (ch < '0' || ch > '9') && ch != '_' {
			return false
		}
	}
	// Check that parens are not balanced (i.e., it's multi-line)
	depth := 0
	inQuote := false
	escaped := false
	for i := 0; i < len(s); i++ {
		ch := s[i]
		if escaped {
			escaped = false
			continue
		}
		if ch == '\\' && inQuote {
			escaped = true
			continue
		}
		if ch == '"' {
			inQuote = !inQuote
			continue
		}
		if !inQuote {
			if ch == '(' {
				depth++
			}
			if ch == ')' {
				depth--
			}
		}
	}
	return depth > 0 // unbalanced = multi-line function call
}

// parseFunctionCall collects a multi-line function call expression like:
//
//	jsonencode({
//	  orders = []
//	})
//
// It tracks parentheses/braces/brackets to find the complete expression,
// then returns the whole thing as a Raw value.
func (p *hclParser) parseFunctionCall(s string) HCLValue {
	var lines []string
	lines = append(lines, s)
	p.pos++

	// Count initial balance
	depth := countBraceDepth(s)

	for p.pos < len(p.lines) && depth > 0 {
		line := p.lines[p.pos]
		lines = append(lines, line)
		depth += countBraceDepth(line)
		p.pos++
	}

	// Join all lines and return as raw expression
	full := strings.Join(lines, "\n")
	// Trim trailing comma if present
	full = strings.TrimSpace(full)
	full = strings.TrimSuffix(full, ",")
	return Raw(full)
}

// countBraceDepth counts the net change in brace/paren/bracket depth for a line,
// respecting quoted strings.
func countBraceDepth(s string) int {
	depth := 0
	inQuote := false
	escaped := false
	for i := 0; i < len(s); i++ {
		ch := s[i]
		if escaped {
			escaped = false
			continue
		}
		if ch == '\\' && inQuote {
			escaped = true
			continue
		}
		if ch == '"' {
			inQuote = !inQuote
			continue
		}
		if !inQuote {
			switch ch {
			case '(', '{', '[':
				depth++
			case ')', '}', ']':
				depth--
			}
		}
	}
	return depth
}

// skipBlock skips to the matching closing brace
func (p *hclParser) skipBlock() {
	depth := 1
	p.pos++ // skip opening line
	for p.pos < len(p.lines) && depth > 0 {
		trimmed := strings.TrimSpace(p.lines[p.pos])
		if strings.HasSuffix(trimmed, "{") {
			depth++
		}
		if trimmed == "}" || trimmed == "}," {
			depth--
		}
		p.pos++
	}
}

// --- Helper functions ---

// extractQuotedLabels extracts all double-quoted strings from a line fragment.
func extractQuotedLabels(s string) []string {
	var labels []string
	for {
		start := strings.Index(s, "\"")
		if start < 0 {
			break
		}
		end := strings.Index(s[start+1:], "\"")
		if end < 0 {
			break
		}
		labels = append(labels, s[start+1:start+1+end])
		s = s[start+1+end+1:]
	}
	return labels
}

// splitBlockHeader splits a block header into identifier tokens,
// respecting quoted strings.
func splitBlockHeader(s string) []string {
	var tokens []string
	var current strings.Builder
	inQuote := false

	for _, ch := range s {
		if ch == '"' {
			inQuote = !inQuote
			current.WriteRune(ch)
		} else if unicode.IsSpace(ch) && !inQuote {
			if current.Len() > 0 {
				tokens = append(tokens, current.String())
				current.Reset()
			}
		} else {
			current.WriteRune(ch)
		}
	}
	if current.Len() > 0 {
		tokens = append(tokens, current.String())
	}
	return tokens
}

// findAttributeEquals finds the index of the = sign in an attribute line,
// ensuring it's not inside a string or part of == or !=.
func findAttributeEquals(s string) int {
	inQuote := false
	escaped := false
	for i, ch := range s {
		if escaped {
			escaped = false
			continue
		}
		if ch == '\\' && inQuote {
			escaped = true
			continue
		}
		if ch == '"' {
			inQuote = !inQuote
			continue
		}
		if !inQuote && ch == '=' {
			// Make sure it's not == or !=
			if i+1 < len(s) && s[i+1] == '=' {
				continue
			}
			if i > 0 && s[i-1] == '!' {
				continue
			}
			return i
		}
	}
	return -1
}

// parseQuotedString extracts a quoted string value, handling escape sequences.
// Returns the unescaped string and the remaining text after the closing quote.
func parseQuotedString(s string) (string, string) {
	if len(s) < 2 || s[0] != '"' {
		return s, ""
	}

	var result strings.Builder
	i := 1
	for i < len(s) {
		ch := s[i]
		if ch == '\\' && i+1 < len(s) {
			next := s[i+1]
			switch next {
			case 'n':
				result.WriteByte('\n')
			case 't':
				result.WriteByte('\t')
			case '"':
				result.WriteByte('"')
			case '\\':
				result.WriteByte('\\')
			default:
				result.WriteByte('\\')
				result.WriteByte(next)
			}
			i += 2
			continue
		}
		if ch == '"' {
			return result.String(), s[i+1:]
		}
		result.WriteByte(ch)
		i++
	}

	return result.String(), ""
}

// isCompleteBracketedExpression checks if a string is a complete [...] or {...} expression.
func isCompleteBracketedExpression(s string, open, close byte) bool {
	s = strings.TrimSpace(s)
	// Remove trailing comma
	s = strings.TrimSuffix(s, ",")

	if len(s) < 2 || s[0] != open {
		return false
	}

	depth := 0
	inQuote := false
	escaped := false
	for i := 0; i < len(s); i++ {
		ch := s[i]
		if escaped {
			escaped = false
			continue
		}
		if ch == '\\' && inQuote {
			escaped = true
			continue
		}
		if ch == '"' {
			inQuote = !inQuote
			continue
		}
		if !inQuote {
			if ch == open {
				depth++
			}
			if ch == close {
				depth--
				if depth == 0 {
					return i == len(s)-1
				}
			}
		}
	}
	return false
}

// splitRespectingQuotes splits a string by delimiter while respecting quoted strings
// and nested brackets.
func splitRespectingQuotes(s string, delim byte) []string {
	var parts []string
	var current strings.Builder
	inQuote := false
	escaped := false
	depth := 0

	for i := 0; i < len(s); i++ {
		ch := s[i]
		if escaped {
			escaped = false
			current.WriteByte(ch)
			continue
		}
		if ch == '\\' && inQuote {
			escaped = true
			current.WriteByte(ch)
			continue
		}
		if ch == '"' {
			inQuote = !inQuote
			current.WriteByte(ch)
			continue
		}
		if !inQuote {
			if ch == '{' || ch == '[' || ch == '(' {
				depth++
			}
			if ch == '}' || ch == ']' || ch == ')' {
				depth--
			}
			if ch == delim && depth == 0 {
				parts = append(parts, current.String())
				current.Reset()
				continue
			}
		}
		current.WriteByte(ch)
	}
	if current.Len() > 0 {
		parts = append(parts, current.String())
	}
	return parts
}

// isNumber checks if a string looks like a number (possibly with trailing comma).
func isNumber(s string) bool {
	s = strings.TrimSuffix(s, ",")
	if s == "" {
		return false
	}
	hasDigit := false
	hasDot := false
	for i, ch := range s {
		if ch == '-' && i == 0 {
			continue
		}
		if ch == '.' && !hasDot {
			hasDot = true
			continue
		}
		if ch >= '0' && ch <= '9' {
			hasDigit = true
			continue
		}
		return false
	}
	return hasDigit
}

// stripHeredocIndent removes common leading whitespace from heredoc content.
func stripHeredocIndent(content string) string {
	lines := strings.Split(content, "\n")

	// Find minimum indentation
	minIndent := -1
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		indent := 0
		for _, ch := range line {
			if ch == ' ' || ch == '\t' {
				indent++
			} else {
				break
			}
		}
		if minIndent < 0 || indent < minIndent {
			minIndent = indent
		}
	}

	if minIndent <= 0 {
		return content
	}

	// Strip common indentation
	for i, line := range lines {
		if len(line) >= minIndent {
			lines[i] = line[minIndent:]
		}
	}
	return strings.Join(lines, "\n")
}
