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

package terraform

import (
	"regexp"
	"strings"

	"github.com/elementumltd/elementum-cli/ui"
)

var (
	// Regex patterns for terraform/tofu output
	createPattern  = regexp.MustCompile(`^\s*[+#]\s+`)
	modifyPattern  = regexp.MustCompile(`^\s*[~]\s+`)
	destroyPattern = regexp.MustCompile(`^\s*[-]\s+`)
	planSummary    = regexp.MustCompile(`^Plan:|to add|to change|to destroy`)
	applySummary   = regexp.MustCompile(`^Apply complete!|Resources:|^Destroy complete!`)
	errorPattern   = regexp.MustCompile(`^Error:|^╷|^│\s*Error`)
	warningPattern = regexp.MustCompile(`^Warning:|^│\s*Warning`)
)

// EnhanceLine applies color coding and formatting to a terraform/tofu output line
func EnhanceLine(line string) string {
	// Skip empty lines
	if strings.TrimSpace(line) == "" {
		return line
	}

	// Error messages - red
	if errorPattern.MatchString(line) {
		return ui.ErrorStyle.Render(line)
	}

	// Warning messages - yellow
	if warningPattern.MatchString(line) {
		return ui.WarningStyle.Render(line)
	}

	// Resources being created - green
	if createPattern.MatchString(line) {
		return ui.SuccessStyle.Render(line)
	}

	// Resources being modified - yellow
	if modifyPattern.MatchString(line) {
		return ui.WarningStyle.Render(line)
	}

	// Resources being destroyed - red
	if destroyPattern.MatchString(line) {
		return ui.ErrorStyle.Render(line)
	}

	// Plan summary - blue
	if planSummary.MatchString(line) {
		return ui.InfoStyle.Render(line)
	}

	// Apply/Destroy completion summary - green
	if applySummary.MatchString(line) {
		return ui.SuccessStyle.Render(line)
	}

	// Default - return as is
	return line
}
