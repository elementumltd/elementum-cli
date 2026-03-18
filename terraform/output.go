// Copyright (c) ElementumAI, Inc.
// SPDX-License-Identifier: MPL-2.0

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
