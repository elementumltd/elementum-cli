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

package discovery

import (
	"fmt"
	"strings"
	"sync"
)

// ErrorSeverity indicates how severe an error is during export.
type ErrorSeverity int

const (
	// SeverityFatal means the export cannot produce valid output.
	// Examples: timeouts, 5xx errors, auth errors, network errors.
	SeverityFatal ErrorSeverity = iota
	// SeverityWarning means data might be incomplete but the error is expected.
	// Examples: "The item does not exist" for orphaned references.
	SeverityWarning
)

func (s ErrorSeverity) String() string {
	switch s {
	case SeverityFatal:
		return "FATAL"
	case SeverityWarning:
		return "WARNING"
	default:
		return "UNKNOWN"
	}
}

// ExportError represents a single error encountered during export.
type ExportError struct {
	Severity ErrorSeverity
	Phase    string // e.g., "discovery", "automation_fetch", "recursive_discovery"
	Resource string // e.g., "automations for Activities Dev", "element xyz"
	Err      error
}

func (e *ExportError) Error() string {
	return fmt.Sprintf("[%s] %s: %s - %v", e.Severity, e.Phase, e.Resource, e.Err)
}

// ErrorCollector accumulates errors during the export process.
// It tracks both fatal errors and warnings, allowing the caller to decide
// whether to abort (strict mode) or continue (lenient mode).
type ErrorCollector struct {
	mu     sync.Mutex
	errors []*ExportError
}

// NewErrorCollector creates a new ErrorCollector.
func NewErrorCollector() *ErrorCollector {
	return &ErrorCollector{}
}

// Add records an error.
func (ec *ErrorCollector) Add(severity ErrorSeverity, phase, resource string, err error) {
	ec.mu.Lock()
	defer ec.mu.Unlock()
	ec.errors = append(ec.errors, &ExportError{
		Severity: severity,
		Phase:    phase,
		Resource: resource,
		Err:      err,
	})
}

// AddFatal records a fatal error.
func (ec *ErrorCollector) AddFatal(phase, resource string, err error) {
	ec.Add(SeverityFatal, phase, resource, err)
}

// AddWarning records a warning.
func (ec *ErrorCollector) AddWarning(phase, resource string, err error) {
	ec.Add(SeverityWarning, phase, resource, err)
}

// HasFatal returns true if any fatal errors have been recorded.
func (ec *ErrorCollector) HasFatal() bool {
	ec.mu.Lock()
	defer ec.mu.Unlock()
	for _, e := range ec.errors {
		if e.Severity == SeverityFatal {
			return true
		}
	}
	return false
}

// HasErrors returns true if any errors (fatal or warning) have been recorded.
func (ec *ErrorCollector) HasErrors() bool {
	ec.mu.Lock()
	defer ec.mu.Unlock()
	return len(ec.errors) > 0
}

// FatalErrors returns only the fatal errors.
func (ec *ErrorCollector) FatalErrors() []*ExportError {
	ec.mu.Lock()
	defer ec.mu.Unlock()
	var fatals []*ExportError
	for _, e := range ec.errors {
		if e.Severity == SeverityFatal {
			fatals = append(fatals, e)
		}
	}
	return fatals
}

// Warnings returns only the warning errors.
func (ec *ErrorCollector) Warnings() []*ExportError {
	ec.mu.Lock()
	defer ec.mu.Unlock()
	var warnings []*ExportError
	for _, e := range ec.errors {
		if e.Severity == SeverityWarning {
			warnings = append(warnings, e)
		}
	}
	return warnings
}

// All returns all recorded errors.
func (ec *ErrorCollector) All() []*ExportError {
	ec.mu.Lock()
	defer ec.mu.Unlock()
	result := make([]*ExportError, len(ec.errors))
	copy(result, ec.errors)
	return result
}

// Summary returns a human-readable summary of all errors.
func (ec *ErrorCollector) Summary() string {
	ec.mu.Lock()
	defer ec.mu.Unlock()

	if len(ec.errors) == 0 {
		return ""
	}

	var sb strings.Builder
	fatals := 0
	warnings := 0
	for _, e := range ec.errors {
		if e.Severity == SeverityFatal {
			fatals++
		} else {
			warnings++
		}
	}

	if fatals > 0 {
		fmt.Fprintf(&sb, "\n✗ Export encountered %d fatal error(s):\n", fatals)
		for _, e := range ec.errors {
			if e.Severity == SeverityFatal {
				fmt.Fprintf(&sb, "  • [%s] %s: %v\n", e.Phase, e.Resource, e.Err)
			}
		}
	}

	if warnings > 0 {
		fmt.Fprintf(&sb, "\n⚠ Export encountered %d warning(s):\n", warnings)
		for _, e := range ec.errors {
			if e.Severity == SeverityWarning {
				fmt.Fprintf(&sb, "  • [%s] %s: %v\n", e.Phase, e.Resource, e.Err)
			}
		}
	}

	return sb.String()
}

// ClassifyError determines if a GraphQL/API error is fatal or just a warning.
// Warning-level errors are expected data issues in the platform that don't indicate
// connectivity or reliability problems. Fatal errors mean we can't trust the data.
func ClassifyError(err error) ErrorSeverity {
	if err == nil {
		return SeverityWarning
	}
	msg := err.Error()

	// Known non-fatal errors from the Elementum API:

	// Orphaned references - items deleted but still referenced
	if strings.Contains(msg, "The item does not exist") {
		return SeverityWarning
	}

	// Object not found - might have been deleted or we don't have access
	if strings.Contains(msg, "not found") && !strings.Contains(msg, "timeout") {
		return SeverityWarning
	}

	// Type mismatches during recursive discovery - object exists but is a different
	// type than expected (e.g., queued as "App" but is actually an AspectTask).
	// These are classification issues, not data integrity problems.
	if strings.Contains(msg, "is not an app") ||
		strings.Contains(msg, "is not an element") ||
		strings.Contains(msg, "is not a task") {
		return SeverityWarning
	}

	// "An unexpected error has occurred" from the API for specific fields/edges
	// is typically a server-side data issue, not a connectivity problem.
	// The partial data is still returned successfully.
	if strings.Contains(msg, "An unexpected error has occurred") {
		return SeverityWarning
	}

	// Everything else is fatal: timeouts, network errors, 5xx, auth, etc.
	return SeverityFatal
}
