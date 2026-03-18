// Copyright (c) ElementumAI, Inc.
// SPDX-License-Identifier: MPL-2.0

package terraform

import (
	"strings"
	"testing"
)

func TestDetectBinary(t *testing.T) {
	// Note: This test depends on what's actually installed on the system
	// It will pass if either terraform or tofu is installed
	binary, err := DetectBinary()

	if err != nil {
		// Error should contain helpful installation instructions
		errStr := err.Error()

		if !strings.Contains(errStr, "Neither terraform nor tofu found") {
			t.Errorf("Error message should mention binary not found, got: %s", errStr)
		}

		if !strings.Contains(errStr, "brew install") {
			t.Errorf("Error message should include brew install instructions, got: %s", errStr)
		}

		if !strings.Contains(errStr, "opentofu") && !strings.Contains(errStr, "OpenTofu") {
			t.Errorf("Error message should mention OpenTofu, got: %s", errStr)
		}

		if !strings.Contains(errStr, "terraform") {
			t.Errorf("Error message should mention terraform, got: %s", errStr)
		}

		// This is expected if neither binary is installed
		t.Skip("Neither terraform nor tofu installed on this system (this is OK)")
		return
	}

	// If no error, we should have detected a binary
	if binary != "terraform" && binary != "tofu" {
		t.Errorf("DetectBinary() = %q, want either 'terraform' or 'tofu'", binary)
	}

	// Test that calling it again returns the same result (caching)
	binary2, err2 := DetectBinary()
	if err2 != nil {
		t.Errorf("Second call to DetectBinary() failed: %v", err2)
	}
	if binary2 != binary {
		t.Errorf("DetectBinary() caching broken: first=%q, second=%q", binary, binary2)
	}
}

func TestDetectBinaryPreference(t *testing.T) {
	// This test documents that tofu is preferred over terraform
	// We can't easily test the actual preference without mocking exec.LookPath,
	// but we can document the expected behavior

	binary, err := DetectBinary()

	if err != nil {
		t.Skip("No terraform/tofu binary found, skipping preference test")
		return
	}

	// Just document what was found
	t.Logf("Detected binary: %s (tofu is preferred if both exist)", binary)
}

func TestShowDetectedBinary(t *testing.T) {
	// Test that ShowDetectedBinary doesn't panic
	// This is a basic smoke test since the function prints to stdout

	defer func() {
		if r := recover(); r != nil {
			t.Errorf("ShowDetectedBinary() panicked: %v", r)
		}
	}()

	// Test with valid binary names
	ShowDetectedBinary("terraform")
	ShowDetectedBinary("tofu")
}

func TestErrorMessageFormat(t *testing.T) {
	// Test the error message format when no binary is found
	// We'll temporarily force an error by using a fresh detection
	// (This test assumes neither binary is in PATH, or we test the error format)

	_, err := DetectBinary()

	if err == nil {
		// Binary was found, can't test error format
		t.Skip("Binary found, cannot test error message format")
		return
	}

	errMsg := err.Error()

	// Check for key components of the error message
	mustContain := []string{
		"Neither terraform nor tofu found",
		"OpenTofu",
		"brew install opentofu",
		"opentofu.org",
		"Terraform",
		"brew install terraform",
		"hashicorp.com",
	}

	for _, substr := range mustContain {
		if !strings.Contains(errMsg, substr) {
			t.Errorf("Error message missing %q:\n%s", substr, errMsg)
		}
	}
}
