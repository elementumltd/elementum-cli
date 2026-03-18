// Copyright (c) ElementumAI, Inc.
// SPDX-License-Identifier: MPL-2.0

package discovery

import (
	"errors"
	"strings"
	"testing"
)

// ============================================================================
// ClassifyError Tests
// ============================================================================

func TestClassifyError_NilIsWarning(t *testing.T) {
	t.Parallel()
	if got := ClassifyError(nil); got != SeverityWarning {
		t.Errorf("ClassifyError(nil) = %v, want SeverityWarning", got)
	}
}

func TestClassifyError_ItemDoesNotExist(t *testing.T) {
	t.Parallel()
	err := errors.New("The item does not exist")
	if got := ClassifyError(err); got != SeverityWarning {
		t.Errorf("ClassifyError(%q) = %v, want SeverityWarning", err, got)
	}
}

func TestClassifyError_NotFound(t *testing.T) {
	t.Parallel()
	err := errors.New("resource not found")
	if got := ClassifyError(err); got != SeverityWarning {
		t.Errorf("ClassifyError(%q) = %v, want SeverityWarning", err, got)
	}
}

func TestClassifyError_NotFoundWithTimeout(t *testing.T) {
	t.Parallel()
	// "not found" combined with "timeout" should be fatal (network issue, not data issue)
	err := errors.New("connection timeout: resource not found")
	if got := ClassifyError(err); got != SeverityFatal {
		t.Errorf("ClassifyError(%q) = %v, want SeverityFatal", err, got)
	}
}

func TestClassifyError_TypeMismatches(t *testing.T) {
	t.Parallel()
	cases := []struct {
		msg string
	}{
		{"aspect abc-123 is not an app"},
		{"aspect xyz is not an element"},
		{"aspect foo is not a task"},
	}
	for _, tc := range cases {
		err := errors.New(tc.msg)
		if got := ClassifyError(err); got != SeverityWarning {
			t.Errorf("ClassifyError(%q) = %v, want SeverityWarning", tc.msg, got)
		}
	}
}

func TestClassifyError_UnexpectedError(t *testing.T) {
	t.Parallel()
	err := errors.New("An unexpected error has occurred")
	if got := ClassifyError(err); got != SeverityWarning {
		t.Errorf("ClassifyError(%q) = %v, want SeverityWarning", err, got)
	}
}

func TestClassifyError_Timeout(t *testing.T) {
	t.Parallel()
	err := errors.New("context deadline exceeded (Client.Timeout exceeded)")
	if got := ClassifyError(err); got != SeverityFatal {
		t.Errorf("ClassifyError(%q) = %v, want SeverityFatal", err, got)
	}
}

func TestClassifyError_AuthError(t *testing.T) {
	t.Parallel()
	err := errors.New("401 Unauthorized: invalid token")
	if got := ClassifyError(err); got != SeverityFatal {
		t.Errorf("ClassifyError(%q) = %v, want SeverityFatal", err, got)
	}
}

func TestClassifyError_NetworkError(t *testing.T) {
	t.Parallel()
	err := errors.New("dial tcp: connection refused")
	if got := ClassifyError(err); got != SeverityFatal {
		t.Errorf("ClassifyError(%q) = %v, want SeverityFatal", err, got)
	}
}

func TestClassifyError_ServerError(t *testing.T) {
	t.Parallel()
	err := errors.New("unexpected status code 500: internal server error")
	if got := ClassifyError(err); got != SeverityFatal {
		t.Errorf("ClassifyError(%q) = %v, want SeverityFatal", err, got)
	}
}

// ============================================================================
// ErrorCollector Tests
// ============================================================================

func TestErrorCollector_Empty(t *testing.T) {
	t.Parallel()
	ec := NewErrorCollector()

	if ec.HasFatal() {
		t.Error("Empty collector should not have fatal errors")
	}
	if ec.HasErrors() {
		t.Error("Empty collector should not have any errors")
	}
	if got := ec.Summary(); got != "" {
		t.Errorf("Empty collector summary should be empty, got %q", got)
	}
	if got := ec.All(); len(got) != 0 {
		t.Errorf("Empty collector All() should return empty slice, got %d items", len(got))
	}
}

func TestErrorCollector_AddWarning(t *testing.T) {
	t.Parallel()
	ec := NewErrorCollector()
	ec.AddWarning("discovery", "test resource", errors.New("item not found"))

	if ec.HasFatal() {
		t.Error("Should not have fatal errors")
	}
	if !ec.HasErrors() {
		t.Error("Should have errors")
	}
	if got := ec.Warnings(); len(got) != 1 {
		t.Errorf("Expected 1 warning, got %d", len(got))
	}
	if got := ec.FatalErrors(); len(got) != 0 {
		t.Errorf("Expected 0 fatal errors, got %d", len(got))
	}
}

func TestErrorCollector_AddFatal(t *testing.T) {
	t.Parallel()
	ec := NewErrorCollector()
	ec.AddFatal("discovery", "test resource", errors.New("connection timeout"))

	if !ec.HasFatal() {
		t.Error("Should have fatal errors")
	}
	if !ec.HasErrors() {
		t.Error("Should have errors")
	}
	if got := ec.FatalErrors(); len(got) != 1 {
		t.Errorf("Expected 1 fatal error, got %d", len(got))
	}
}

func TestErrorCollector_MixedErrors(t *testing.T) {
	t.Parallel()
	ec := NewErrorCollector()
	ec.AddWarning("discovery", "orphaned ref", errors.New("The item does not exist"))
	ec.AddFatal("discovery", "automations", errors.New("connection timeout"))
	ec.AddWarning("discovery", "type mismatch", errors.New("is not an app"))

	if got := ec.All(); len(got) != 3 {
		t.Errorf("Expected 3 total errors, got %d", len(got))
	}
	if got := ec.FatalErrors(); len(got) != 1 {
		t.Errorf("Expected 1 fatal error, got %d", len(got))
	}
	if got := ec.Warnings(); len(got) != 2 {
		t.Errorf("Expected 2 warnings, got %d", len(got))
	}
}

func TestErrorCollector_Summary_FatalAndWarnings(t *testing.T) {
	t.Parallel()
	ec := NewErrorCollector()
	ec.AddFatal("discovery", "automations", errors.New("timeout"))
	ec.AddWarning("discovery", "field ref", errors.New("The item does not exist"))

	summary := ec.Summary()

	if !strings.Contains(summary, "1 fatal error") {
		t.Errorf("Summary should mention fatal error count, got: %s", summary)
	}
	if !strings.Contains(summary, "1 warning") {
		t.Errorf("Summary should mention warning count, got: %s", summary)
	}
	if !strings.Contains(summary, "timeout") {
		t.Errorf("Summary should include fatal error message, got: %s", summary)
	}
	if !strings.Contains(summary, "The item does not exist") {
		t.Errorf("Summary should include warning message, got: %s", summary)
	}
}

func TestErrorCollector_Summary_WarningsOnly(t *testing.T) {
	t.Parallel()
	ec := NewErrorCollector()
	ec.AddWarning("discovery", "ref", errors.New("orphaned"))

	summary := ec.Summary()

	if strings.Contains(summary, "fatal") {
		t.Errorf("Summary with only warnings should not mention fatal, got: %s", summary)
	}
	if !strings.Contains(summary, "1 warning") {
		t.Errorf("Summary should mention warning count, got: %s", summary)
	}
}

func TestExportError_Error(t *testing.T) {
	t.Parallel()
	e := &ExportError{
		Severity: SeverityFatal,
		Phase:    "discovery",
		Resource: "test app",
		Err:      errors.New("timeout"),
	}
	got := e.Error()
	if !strings.Contains(got, "FATAL") || !strings.Contains(got, "discovery") || !strings.Contains(got, "timeout") {
		t.Errorf("ExportError.Error() = %q, expected to contain FATAL, discovery, timeout", got)
	}
}

func TestErrorSeverity_String(t *testing.T) {
	t.Parallel()
	if got := SeverityFatal.String(); got != "FATAL" {
		t.Errorf("SeverityFatal.String() = %q, want FATAL", got)
	}
	if got := SeverityWarning.String(); got != "WARNING" {
		t.Errorf("SeverityWarning.String() = %q, want WARNING", got)
	}
	if got := ErrorSeverity(99).String(); got != "UNKNOWN" {
		t.Errorf("ErrorSeverity(99).String() = %q, want UNKNOWN", got)
	}
}
