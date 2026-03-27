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
	"regexp"
	"strings"
)

// ExtractResourceFromPlan extracts a resource block from terraform plan output
// Works for any resource type (flows, automations, layouts, etc.)
func ExtractResourceFromPlan(planOutput, resourceType, resourceName string) string {
	// Pattern to match the resource in plan output
	// Example: resource "elementum_flow" "my_flow" { ... }
	pattern := regexp.MustCompile(
		`(?s)\s*resource "` + regexp.QuoteMeta(resourceType) + `" "` + regexp.QuoteMeta(resourceName) + `" \{.*?\n    \}`,
	)

	match := pattern.FindString(planOutput)
	if match == "" {
		return ""
	}

	// Normalize indentation from terraform plan format to file format
	return NormalizePlanIndentation(match)
}

// NormalizePlanIndentation converts terraform plan indentation (4-space base + 4-space per level)
// to standard file indentation (2-space per level)
func NormalizePlanIndentation(content string) string {
	lines := strings.Split(content, "\n")

	// Process each line
	for i, line := range lines {
		trimmed := strings.TrimLeft(line, " ")

		// Empty lines stay empty
		if trimmed == "" {
			lines[i] = ""
			continue
		}

		leadingSpaces := len(line) - len(trimmed)

		// First line (resource declaration) goes to column 0
		if i == 0 {
			lines[i] = trimmed
			continue
		}

		// Terraform plan uses 4 spaces as base indentation for content inside resource blocks
		// Then 4 additional spaces per nesting level
		// We want: 0 for resource line, 2 spaces per nesting level for content
		//
		// Plan format:
		//     resource "..." {      <- 4 spaces (we strip to 0)
		//         id = "..."         <- 8 spaces (4 base + 4 for level 1) -> 2 spaces
		//         stages = [         <- 8 spaces -> 2 spaces
		//             {              <- 12 spaces (4 base + 8 for level 2) -> 4 spaces
		//                 nodes = [] <- 16 spaces (4 base + 12 for level 3) -> 6 spaces
		//             }              <- 12 spaces -> 4 spaces
		//         ]                  <- 8 spaces -> 2 spaces
		//     }                      <- 4 spaces -> 0 spaces (closing brace)

		// The closing brace of the resource should be at column 0
		if strings.TrimSpace(line) == "}" && i == len(lines)-1 {
			lines[i] = "}"
			continue
		}

		// For all other lines: remove 4-space base, then halve the remaining
		// This converts 4-space-per-level to 2-space-per-level
		if leadingSpaces >= 4 {
			newSpaces := (leadingSpaces - 4) / 2
			lines[i] = strings.Repeat(" ", newSpaces) + trimmed
		} else {
			// If somehow less than 4 spaces, just keep as-is
			lines[i] = line
		}
	}

	return strings.Join(lines, "\n")
}
