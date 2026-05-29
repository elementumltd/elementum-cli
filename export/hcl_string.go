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
	"strings"
)

// defaultDelimiters is a list of heredoc delimiters to try in order of preference
var defaultDelimiters = []string{"EOT", "EOF", "HEREDOC", "END", "DOC"}

// escapeTemplateInterpolation escapes ${...} patterns to $${...}
// so they are treated as literal text in HCL, not Terraform interpolations.
// This is needed when exporting strings that contain example syntax like
// "${task_key.output_key}" which should be displayed literally.
func escapeTemplateInterpolation(s string) string {
	return strings.ReplaceAll(s, "${", "$${")
}

// FormatHCLString formats a string for HCL output.
// - Single-line strings: uses double quotes with proper escaping
// - Multi-line strings (contains \n): uses <<-DELIMITER heredoc syntax
// - Template interpolations (${...}) are escaped to $${...} to be literal
func FormatHCLString(s string) string {
	if s == "" {
		return `""`
	}

	// Escape template interpolation patterns so they're treated as literals
	s = escapeTemplateInterpolation(s)

	if containsNewlines(s) {
		return formatAsHeredoc(s)
	}

	return fmt.Sprintf("%q", s)
}

// FormatHCLStringWithInterpolations formats a string that may contain
// Terraform interpolation expressions (${...}).
// - Single-line: uses double quotes with proper escaping (interpolations preserved)
// - Multi-line: uses <<-DELIMITER heredoc syntax (interpolations work inside heredocs)
func FormatHCLStringWithInterpolations(s string) string {
	if s == "" {
		return `""`
	}

	if containsNewlines(s) {
		return formatAsHeredoc(s)
	}

	// For single-line strings with interpolations, we need to handle escaping
	// carefully. The ${...} expressions should NOT be escaped, but quotes
	// and backslashes outside of interpolations should be.
	return escapeHCLStringWithInterpolations(s)
}

// containsNewlines checks if string has embedded newline characters
func containsNewlines(s string) bool {
	return strings.Contains(s, "\n")
}

// formatAsHeredoc formats a string using HCL heredoc syntax.
// Uses <<-DELIMITER format which strips common leading whitespace.
func formatAsHeredoc(s string) string {
	delimiter := chooseDelimiter(s)

	// Ensure the string doesn't end with the delimiter on its own line
	// by trimming trailing newlines and adding one back
	content := strings.TrimRight(s, "\n\r")

	return fmt.Sprintf("<<-%s\n%s\n%s", delimiter, content, delimiter)
}

// chooseDelimiter selects a heredoc delimiter that doesn't appear in the string
func chooseDelimiter(s string) string {
	for _, delim := range defaultDelimiters {
		// Check if the delimiter appears at the start of a line
		// (which would prematurely end the heredoc)
		if !strings.Contains(s, "\n"+delim) && !strings.HasPrefix(s, delim) {
			return delim
		}
	}

	// If all default delimiters conflict, generate a unique one
	for i := 1; ; i++ {
		delim := fmt.Sprintf("HEREDOC_%d", i)
		if !strings.Contains(s, "\n"+delim) && !strings.HasPrefix(s, delim) {
			return delim
		}
	}
}

// escapeHCLStringWithInterpolations properly escapes a string for double-quoted
// HCL format while preserving Terraform interpolation expressions (${...}).
func escapeHCLStringWithInterpolations(s string) string {
	var result strings.Builder
	result.WriteString(`"`)

	inInterpolation := false
	braceDepth := 0

	for i := 0; i < len(s); i++ {
		c := s[i]

		// Check for start of interpolation
		if !inInterpolation && c == '$' && i+1 < len(s) && s[i+1] == '{' {
			inInterpolation = true
			braceDepth = 1
			result.WriteString("${")
			i++ // Skip the '{'
			continue
		}

		// Track brace depth in interpolation
		if inInterpolation {
			switch c {
			case '{':
				braceDepth++
			case '}':
				braceDepth--
				if braceDepth == 0 {
					inInterpolation = false
				}
			}
			result.WriteByte(c)
			continue
		}

		// Outside interpolation - escape special characters
		switch c {
		case '"':
			result.WriteString(`\"`)
		case '\\':
			result.WriteString(`\\`)
		case '\n':
			result.WriteString(`\n`)
		case '\r':
			result.WriteString(`\r`)
		case '\t':
			result.WriteString(`\t`)
		default:
			result.WriteByte(c)
		}
	}

	result.WriteString(`"`)
	return result.String()
}
