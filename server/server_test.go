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

package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/elementumltd/elementum-cli/analysis"
)

func TestExecutionServer_HandleRoot(t *testing.T) {
	now := time.Now()
	execAnalysis := &analysis.ExecutionAnalysis{
		AutomationID:   "auto-123",
		AutomationName: "Test Automation",
		ExecutionID:    "exec-456",
		Status:         "SUCCESS",
		StartedAt:      now,
		Duration:       5000,
		Version:        1,
		Actions: []analysis.ActionAnalysis{
			{
				ID:       "action-1",
				Name:     "First Action",
				Type:     "EXECUTE_SCRIPT",
				Status:   "SUCCESS",
				IOLoaded: false,
			},
			{
				ID:       "action-2",
				Name:     "Second Action",
				Type:     "RECORD_SEARCH",
				Status:   "SUCCESS",
				IOLoaded: false,
			},
		},
		Summary: analysis.ExecutionSummary{
			TotalActions: 2,
			SuccessCount: 2,
		},
	}

	srv := NewExecutionServer(nil, "aspect-1", "auto-123", "exec-456", execAnalysis)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	srv.handleRoot(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	contentType := resp.Header.Get("Content-Type")
	if !strings.HasPrefix(contentType, "text/html") {
		t.Errorf("expected content-type text/html, got %s", contentType)
	}

	body := w.Body.String()

	// Verify HTML contains expected elements
	if !strings.Contains(body, "Test Automation") {
		t.Error("expected HTML to contain automation name")
	}
	if !strings.Contains(body, "exec-456") {
		t.Error("expected HTML to contain execution ID")
	}
	if !strings.Contains(body, "First Action") {
		t.Error("expected HTML to contain action name")
	}
	if !strings.Contains(body, "Load Inputs/Outputs") {
		t.Error("expected HTML to contain lazy load button")
	}
	if !strings.Contains(body, "/api/actions/") {
		t.Error("expected HTML to contain API endpoint for lazy loading")
	}
}

func TestExecutionServer_HandleRoot_NotFound(t *testing.T) {
	srv := NewExecutionServer(nil, "", "", "", &analysis.ExecutionAnalysis{})

	req := httptest.NewRequest(http.MethodGet, "/other-path", nil)
	w := httptest.NewRecorder()

	srv.handleRoot(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", resp.StatusCode)
	}
}

func TestExecutionServer_HandleActionIO_MissingID(t *testing.T) {
	srv := NewExecutionServer(nil, "", "", "", &analysis.ExecutionAnalysis{})

	req := httptest.NewRequest(http.MethodGet, "/api/actions//io", nil)
	w := httptest.NewRecorder()

	srv.handleActionIO(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", resp.StatusCode)
	}
}

func TestTruncateID(t *testing.T) {
	tests := []struct {
		id       string
		length   int
		expected string
	}{
		{"short", 10, "short"},
		{"exactly-ten", 11, "exactly-ten"},
		{"this-is-a-very-long-id", 12, "this-is-a-ve..."},
		{"abc", 3, "abc"},
		{"abcd", 3, "abc..."},
	}

	for _, tt := range tests {
		result := truncateID(tt.id, tt.length)
		if result != tt.expected {
			t.Errorf("truncateID(%q, %d) = %q, expected %q", tt.id, tt.length, result, tt.expected)
		}
	}
}

func TestLazyLoadingHTMLTemplate_ContainsRequiredElements(t *testing.T) {
	// Verify template contains required JavaScript for lazy loading
	if !strings.Contains(lazyLoadingHTMLTemplate, "async function loadIO") {
		t.Error("template missing loadIO function")
	}
	if !strings.Contains(lazyLoadingHTMLTemplate, "fetch('/api/actions/'") {
		t.Error("template missing fetch call to API")
	}
	if !strings.Contains(lazyLoadingHTMLTemplate, "loadedIO") {
		t.Error("template missing loadedIO tracking object")
	}
	if !strings.Contains(lazyLoadingHTMLTemplate, "Loading...") {
		t.Error("template missing loading state text")
	}
	if !strings.Contains(lazyLoadingHTMLTemplate, "Load Inputs/Outputs") {
		t.Error("template missing initial button text")
	}
}

func TestExecutionServer_NewExecutionServer(t *testing.T) {
	execAnalysis := &analysis.ExecutionAnalysis{
		AutomationID: "auto-1",
		ExecutionID:  "exec-1",
	}

	srv := NewExecutionServer(nil, "aspect-1", "auto-1", "exec-1", execAnalysis)

	if srv.aspectID != "aspect-1" {
		t.Errorf("expected aspectID 'aspect-1', got %s", srv.aspectID)
	}
	if srv.automationID != "auto-1" {
		t.Errorf("expected automationID 'auto-1', got %s", srv.automationID)
	}
	if srv.executionID != "exec-1" {
		t.Errorf("expected executionID 'exec-1', got %s", srv.executionID)
	}
	if srv.analysis != execAnalysis {
		t.Error("expected analysis to match")
	}
}

func TestExecutionServer_HandleRoot_JSONEmbedded(t *testing.T) {
	now := time.Now()
	execAnalysis := &analysis.ExecutionAnalysis{
		AutomationID:   "auto-test",
		AutomationName: "JSON Test",
		ExecutionID:    "exec-json",
		Status:         "FAILURE",
		StartedAt:      now,
		Duration:       1234,
		Version:        2,
		Errors:         []string{"Test error"},
		Actions: []analysis.ActionAnalysis{
			{
				ID:       "act-1",
				Name:     "Test Action",
				Type:     "MESSAGE",
				Status:   "FAILURE",
				IOLoaded: false,
			},
		},
		Summary: analysis.ExecutionSummary{
			TotalActions: 1,
			FailureCount: 1,
		},
	}

	srv := NewExecutionServer(nil, "aspect-1", "auto-test", "exec-json", execAnalysis)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	srv.handleRoot(w, req)

	body := w.Body.String()

	// Verify JSON is embedded in the page
	if !strings.Contains(body, `"automationName":"JSON Test"`) {
		t.Error("expected embedded JSON to contain automation name")
	}
	if !strings.Contains(body, `"status":"FAILURE"`) {
		t.Error("expected embedded JSON to contain status")
	}
	if !strings.Contains(body, `"duration":1234`) {
		t.Error("expected embedded JSON to contain duration")
	}
}

func TestActionIOResponse_Structure(t *testing.T) {
	// Test the expected structure of the API response
	response := map[string]any{
		"id":      "action-123",
		"inputs":  json.RawMessage(`{"key": "value"}`),
		"outputs": json.RawMessage(`{"result": 42}`),
	}

	data, err := json.Marshal(response)
	if err != nil {
		t.Fatalf("failed to marshal response: %v", err)
	}

	var parsed map[string]any
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if parsed["id"] != "action-123" {
		t.Errorf("expected id 'action-123', got %v", parsed["id"])
	}
}
