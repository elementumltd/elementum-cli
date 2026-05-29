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
	"sync"

	"github.com/elementumltd/elementum-cli/ui"
)

// WarningUnresolvedRef is the message used when a value reference cannot be resolved
const WarningUnresolvedRef = "Value reference could not be resolved"

// ExportWarning represents a warning encountered during export
type ExportWarning struct {
	Resource    string // e.g., "elementum_create_record_task.my_task"
	Field       string // e.g., "fields[0].value"
	Message     string // e.g., "Value reference could not be resolved"
	Suggestion  string // e.g., "Add trigger.refs[\"FieldName\"] or task.refs[\"output\"]"
	RawRefValue string // The raw reference value that couldn't be resolved (for debugging)
}

// ExportWarnings collects warnings during export
type ExportWarnings struct {
	mu       sync.Mutex
	warnings []ExportWarning
}

// Global warnings collector for the current export operation
var globalWarnings = &ExportWarnings{}

// ResetWarnings clears all collected warnings (call at start of export)
func ResetWarnings() {
	globalWarnings.mu.Lock()
	defer globalWarnings.mu.Unlock()
	globalWarnings.warnings = nil
}

// AddWarning adds a warning to the collector
func AddWarning(resource, field, message, suggestion string) {
	globalWarnings.mu.Lock()
	defer globalWarnings.mu.Unlock()
	globalWarnings.warnings = append(globalWarnings.warnings, ExportWarning{
		Resource:   resource,
		Field:      field,
		Message:    message,
		Suggestion: suggestion,
	})
}

// AddWarningWithRef adds a warning with the raw reference value for debugging
func AddWarningWithRef(resource, field, message, suggestion, rawRef string) {
	globalWarnings.mu.Lock()
	defer globalWarnings.mu.Unlock()
	globalWarnings.warnings = append(globalWarnings.warnings, ExportWarning{
		Resource:    resource,
		Field:       field,
		Message:     message,
		Suggestion:  suggestion,
		RawRefValue: rawRef,
	})
}

// GetWarnings returns all collected warnings
func GetWarnings() []ExportWarning {
	globalWarnings.mu.Lock()
	defer globalWarnings.mu.Unlock()
	return append([]ExportWarning{}, globalWarnings.warnings...)
}

// HasWarnings returns true if there are any warnings
func HasWarnings() bool {
	globalWarnings.mu.Lock()
	defer globalWarnings.mu.Unlock()
	return len(globalWarnings.warnings) > 0
}

// WarningCount returns the number of warnings
func WarningCount() int {
	globalWarnings.mu.Lock()
	defer globalWarnings.mu.Unlock()
	return len(globalWarnings.warnings)
}

// PrintWarnings prints all warnings to the console
func PrintWarnings() {
	warnings := GetWarnings()
	if len(warnings) == 0 {
		return
	}

	fmt.Println()
	fmt.Println(ui.WarningStyle.Render(fmt.Sprintf("Export completed with %d warning(s):", len(warnings))))
	fmt.Println()

	// Group warnings by type for cleaner output
	unresolved := []ExportWarning{}
	other := []ExportWarning{}

	for _, w := range warnings {
		if w.Message == WarningUnresolvedRef {
			unresolved = append(unresolved, w)
		} else {
			other = append(other, w)
		}
	}

	// Print unresolved value references (grouped)
	if len(unresolved) > 0 {
		fmt.Printf("  %s Unresolved value references (%d):\n", ui.RenderBullet(), len(unresolved))
		// Show first few examples
		shown := 0
		for _, w := range unresolved {
			if shown >= 5 {
				fmt.Printf("      ... and %d more\n", len(unresolved)-5)
				break
			}
			if w.RawRefValue != "" {
				fmt.Printf("      • %s.%s (raw: %s)\n", w.Resource, w.Field, ui.Truncate(w.RawRefValue, 60))
			} else {
				fmt.Printf("      • %s.%s\n", w.Resource, w.Field)
			}
			shown++
		}
		fmt.Printf("    %s Look for '# TODO' comments in the generated files\n", ui.MutedStyle.Render("→"))
		fmt.Println()
	}

	// Print other warnings
	for _, w := range other {
		fmt.Printf("  %s %s: %s\n", ui.RenderBullet(), w.Resource, w.Message)
		if w.Field != "" {
			fmt.Printf("      Field: %s\n", w.Field)
		}
		if w.Suggestion != "" {
			fmt.Printf("    %s %s\n", ui.MutedStyle.Render("→"), w.Suggestion)
		}
		fmt.Println()
	}
}
