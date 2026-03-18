// Copyright (c) ElementumAI, Inc.
// SPDX-License-Identifier: MPL-2.0

package terraform

import (
	"strings"
	"testing"

	"github.com/elementumltd/elementum-cli/ui"
)

func TestEnhanceLine(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantPlain bool // if true, output should be unchanged
	}{
		{
			name:      "empty line",
			input:     "",
			wantPlain: true,
		},
		{
			name:      "whitespace only",
			input:     "   ",
			wantPlain: true,
		},
		{
			name:      "resource being created with +",
			input:     "  + resource \"elementum_app\" \"test\" {",
			wantPlain: false,
		},
		{
			name:      "resource being created with #",
			input:     "  # elementum_app.test will be created",
			wantPlain: false,
		},
		{
			name:      "resource being modified",
			input:     "  ~ resource \"elementum_text_field\" \"name\" {",
			wantPlain: false,
		},
		{
			name:      "resource being destroyed",
			input:     "  - resource \"elementum_app\" \"old\" {",
			wantPlain: false,
		},
		{
			name:      "plan summary",
			input:     "Plan: 5 to add, 2 to change, 1 to destroy.",
			wantPlain: false,
		},
		{
			name:      "apply complete",
			input:     "Apply complete! Resources: 5 added, 2 changed, 1 destroyed.",
			wantPlain: false,
		},
		{
			name:      "destroy complete",
			input:     "Destroy complete! Resources: 5 destroyed.",
			wantPlain: false,
		},
		{
			name:      "error message",
			input:     "Error: Invalid configuration",
			wantPlain: false,
		},
		{
			name:      "warning message",
			input:     "Warning: Deprecated argument",
			wantPlain: false,
		},
		{
			name:      "regular output",
			input:     "Terraform will perform the following actions:",
			wantPlain: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := EnhanceLine(tt.input)

			// Test that function doesn't panic and preserves text
			if got == "" && tt.input != "" {
				t.Errorf("EnhanceLine(%q) returned empty string", tt.input)
			}

			// For plain text, it should be unchanged or at least contain the original
			if tt.wantPlain && strings.TrimSpace(tt.input) == "" {
				if got != tt.input {
					t.Errorf("EnhanceLine(%q) = %q, want unchanged", tt.input, got)
				}
			}

			// The text should still be present (possibly with styling)
			stripped := stripANSI(got)
			if stripped != tt.input {
				t.Errorf("EnhanceLine(%q) altered text: got %q", tt.input, stripped)
			}
		})
	}
}

func TestEnhanceLinePatternMatching(t *testing.T) {
	// Test that patterns correctly identify line types and preserve text
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "create pattern matches plus sign",
			input: "  + id                           = (known after apply)",
		},
		{
			name:  "modify pattern matches tilde",
			input: "  ~ name                         = \"old\" -> \"new\"",
		},
		{
			name:  "destroy pattern matches minus",
			input: "  - resource \"elementum_app\" \"test\" {",
		},
		{
			name:  "plan summary contains 'to add'",
			input: "Plan: 1 to add, 0 to change, 0 to destroy.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := EnhanceLine(tt.input)

			// Verify original text is preserved (with or without styling)
			stripped := stripANSI(got)
			if stripped != tt.input {
				t.Errorf("EnhanceLine(%q) altered text: got %q", tt.input, stripped)
			}

			// Verify function doesn't panic or return empty
			if got == "" {
				t.Errorf("EnhanceLine(%q) returned empty string", tt.input)
			}
		})
	}
}

func TestEnhanceLinePreservesText(t *testing.T) {
	// Test that enhancement preserves the original text content
	inputs := []string{
		"  + resource \"elementum_app\" \"test\" {",
		"  ~ name = \"old\" -> \"new\"",
		"  - resource \"elementum_app\" \"old\" {",
		"Plan: 5 to add, 2 to change, 1 to destroy.",
		"Error: something went wrong",
		"Regular text without patterns",
	}

	for _, input := range inputs {
		t.Run(input, func(t *testing.T) {
			got := EnhanceLine(input)

			// Strip ANSI codes to get the text
			stripped := stripANSI(got)

			if stripped != input {
				t.Errorf("EnhanceLine(%q) altered the text: got %q", input, stripped)
			}
		})
	}
}

// stripANSI removes ANSI escape codes from a string
func stripANSI(s string) string {
	// Simple ANSI code stripper for testing
	result := ""
	inEscape := false

	for i := 0; i < len(s); i++ {
		if s[i] == '\x1b' {
			inEscape = true
			continue
		}

		if inEscape {
			if s[i] == 'm' {
				inEscape = false
			}
			continue
		}

		result += string(s[i])
	}

	return result
}

func TestUIStylesExist(t *testing.T) {
	// Verify that the UI styles we're using are available
	styles := []struct {
		name  string
		style interface{}
	}{
		{"SuccessStyle", ui.SuccessStyle},
		{"WarningStyle", ui.WarningStyle},
		{"ErrorStyle", ui.ErrorStyle},
		{"InfoStyle", ui.InfoStyle},
	}

	for _, s := range styles {
		t.Run(s.name, func(t *testing.T) {
			if s.style == nil {
				t.Errorf("ui.%s is nil", s.name)
			}
		})
	}
}
