// Copyright (c) ElementumAI, Inc.
// SPDX-License-Identifier: MPL-2.0

package export

import (
	"strings"
	"testing"
)

func TestFormatHCLString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "empty string",
			input:    "",
			expected: `""`,
		},
		{
			name:     "simple string",
			input:    "Hello World",
			expected: `"Hello World"`,
		},
		{
			name:     "string with double quotes",
			input:    `Say "hello"`,
			expected: `"Say \"hello\""`,
		},
		{
			name:     "string with backslash",
			input:    `path\to\file`,
			expected: `"path\\to\\file"`,
		},
		{
			name:     "string with tab",
			input:    "hello\tworld",
			expected: `"hello\tworld"`,
		},
		{
			name:     "multi-line simple",
			input:    "Line 1\nLine 2",
			expected: "<<-EOT\nLine 1\nLine 2\nEOT",
		},
		{
			name:     "multi-line with multiple newlines",
			input:    "Line 1\nLine 2\nLine 3",
			expected: "<<-EOT\nLine 1\nLine 2\nLine 3\nEOT",
		},
		{
			name:     "multi-line with trailing newline",
			input:    "Line 1\nLine 2\n",
			expected: "<<-EOT\nLine 1\nLine 2\nEOT",
		},
		{
			name:     "multi-line bullet points",
			input:    "Items:\n• Item 1\n• Item 2\n• Item 3",
			expected: "<<-EOT\nItems:\n• Item 1\n• Item 2\n• Item 3\nEOT",
		},
		{
			name:     "multi-line with carriage return",
			input:    "Line 1\r\nLine 2",
			expected: "<<-EOT\nLine 1\r\nLine 2\nEOT",
		},
		// Template interpolation escaping - ${...} should become $${...}
		{
			name:     "escapes template interpolation",
			input:    `Use ${task_key.output}`,
			expected: `"Use $${task_key.output}"`,
		},
		{
			name:     "escapes multiple template interpolations",
			input:    `${foo} and ${bar}`,
			expected: `"$${foo} and $${bar}"`,
		},
		{
			name:     "escapes template in quoted context",
			input:    `Value Syntax: "${task_key.output_key}" for dynamic values.`,
			expected: `"Value Syntax: \"$${task_key.output_key}\" for dynamic values."`,
		},
		{
			name:     "preserves already escaped template",
			input:    `Already escaped $${foo}`,
			expected: `"Already escaped $$${foo}"`,
		},
		{
			name:     "escapes template in multi-line heredoc",
			input:    "Example:\n${step1.ip}\n${provision_vm.ip_address}",
			expected: "<<-EOT\nExample:\n$${step1.ip}\n$${provision_vm.ip_address}\nEOT",
		},
		{
			name:     "dollar without brace not escaped",
			input:    `Price: $100`,
			expected: `"Price: $100"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatHCLString(tt.input)
			if result != tt.expected {
				t.Errorf("FormatHCLString(%q) =\n%s\nwant:\n%s", tt.input, result, tt.expected)
			}
		})
	}
}

func TestFormatHCLStringWithInterpolations(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "empty string",
			input:    "",
			expected: `""`,
		},
		{
			name:     "simple string no interpolation",
			input:    "Hello World",
			expected: `"Hello World"`,
		},
		{
			name:     "single interpolation",
			input:    `Value: ${var.name}`,
			expected: `"Value: ${var.name}"`,
		},
		{
			name:     "interpolation with quotes inside",
			input:    `Field: ${automation.refs["Field Name"]}`,
			expected: `"Field: ${automation.refs["Field Name"]}"`,
		},
		{
			name:     "multiple interpolations with quotes",
			input:    `${automation.refs["Title"]} - ${automation.refs["Status"]}`,
			expected: `"${automation.refs["Title"]} - ${automation.refs["Status"]}"`,
		},
		{
			name:     "quote outside interpolation",
			input:    `Say "hello" ${var.name}`,
			expected: `"Say \"hello\" ${var.name}"`,
		},
		{
			name:     "multi-line with interpolation",
			input:    "Name: ${var.name}\nAge: ${var.age}",
			expected: "<<-EOT\nName: ${var.name}\nAge: ${var.age}\nEOT",
		},
		{
			name:     "multi-line with refs containing quotes",
			input:    "Field: ${automation.refs[\"Name\"]}\nStatus: ${automation.refs[\"Status\"]}",
			expected: "<<-EOT\nField: ${automation.refs[\"Name\"]}\nStatus: ${automation.refs[\"Status\"]}\nEOT",
		},
		{
			name:     "complex multi-line prompt",
			input:    "Take this value,${automation.refs[\"p1\"]} and update using the matrix below:\n\n• R01 = PO Issue\n• R02 = Mismatch\n• R03 = Missing Data",
			expected: "<<-EOT\nTake this value,${automation.refs[\"p1\"]} and update using the matrix below:\n\n• R01 = PO Issue\n• R02 = Mismatch\n• R03 = Missing Data\nEOT",
		},
		{
			name:     "nested braces in interpolation",
			input:    `${jsonencode({"key": "value"})}`,
			expected: `"${jsonencode({"key": "value"})}"`,
		},
		{
			name:     "backslash outside interpolation",
			input:    `path\to\${var.file}`,
			expected: `"path\\to\\${var.file}"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatHCLStringWithInterpolations(tt.input)
			if result != tt.expected {
				t.Errorf("FormatHCLStringWithInterpolations(%q) =\n%s\nwant:\n%s", tt.input, result, tt.expected)
			}
		})
	}
}

func TestContainsNewlines(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"hello", false},
		{"hello\nworld", true},
		{"\n", true},
		{"", false},
		{"hello\rworld", false}, // only \r is not a newline for our purposes
		{"hello\r\nworld", true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := containsNewlines(tt.input)
			if result != tt.expected {
				t.Errorf("containsNewlines(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestChooseDelimiter(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		shouldContain string
		shouldNotBe   []string
	}{
		{
			name:          "normal string uses EOT",
			input:         "hello world",
			shouldContain: "EOT",
		},
		{
			name:          "string with EOT uses EOF",
			input:         "line1\nEOT\nline2",
			shouldContain: "EOF",
			shouldNotBe:   []string{"EOT"},
		},
		{
			name:          "string starting with EOT uses EOF",
			input:         "EOT is a marker\nline2",
			shouldContain: "EOF",
			shouldNotBe:   []string{"EOT"},
		},
		{
			name:          "string with EOT and EOF uses HEREDOC",
			input:         "line1\nEOT\nline2\nEOF\nline3",
			shouldContain: "HEREDOC",
			shouldNotBe:   []string{"EOT", "EOF"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := chooseDelimiter(tt.input)
			if !strings.Contains(result, tt.shouldContain) {
				t.Errorf("chooseDelimiter() = %q, should contain %q", result, tt.shouldContain)
			}
			for _, notBe := range tt.shouldNotBe {
				if result == notBe {
					t.Errorf("chooseDelimiter() = %q, should not be %q", result, notBe)
				}
			}
		})
	}
}

func TestEscapeHCLStringWithInterpolations(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "no special chars",
			input:    "hello",
			expected: `"hello"`,
		},
		{
			name:     "quotes outside interpolation",
			input:    `say "hi"`,
			expected: `"say \"hi\""`,
		},
		{
			name:     "quotes inside interpolation preserved",
			input:    `${refs["name"]}`,
			expected: `"${refs["name"]}"`,
		},
		{
			name:     "mixed quotes",
			input:    `"prefix" ${refs["name"]} "suffix"`,
			expected: `"\"prefix\" ${refs["name"]} \"suffix\""`,
		},
		{
			name:     "backslash outside interpolation",
			input:    `back\slash ${var}`,
			expected: `"back\\slash ${var}"`,
		},
		{
			name:     "newline gets escaped in single line mode",
			input:    "no\nnewline",
			expected: `"no\nnewline"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := escapeHCLStringWithInterpolations(tt.input)
			if result != tt.expected {
				t.Errorf("escapeHCLStringWithInterpolations(%q) = %s, want %s", tt.input, result, tt.expected)
			}
		})
	}
}

func TestFormatAsHeredoc(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple two lines",
			input:    "line1\nline2",
			expected: "<<-EOT\nline1\nline2\nEOT",
		},
		{
			name:     "strips trailing newlines",
			input:    "line1\nline2\n\n",
			expected: "<<-EOT\nline1\nline2\nEOT",
		},
		{
			name:     "preserves internal empty lines",
			input:    "line1\n\nline3",
			expected: "<<-EOT\nline1\n\nline3\nEOT",
		},
		{
			name:     "with quotes",
			input:    "say \"hello\"\nworld",
			expected: "<<-EOT\nsay \"hello\"\nworld\nEOT",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatAsHeredoc(tt.input)
			if result != tt.expected {
				t.Errorf("formatAsHeredoc(%q) =\n%s\nwant:\n%s", tt.input, result, tt.expected)
			}
		})
	}
}

func TestEscapeTemplateInterpolation(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "no interpolation",
			input:    "hello world",
			expected: "hello world",
		},
		{
			name:     "single interpolation",
			input:    "${foo}",
			expected: "$${foo}",
		},
		{
			name:     "multiple interpolations",
			input:    "${foo} and ${bar}",
			expected: "$${foo} and $${bar}",
		},
		{
			name:     "nested braces",
			input:    "${task_key.output_key}",
			expected: "$${task_key.output_key}",
		},
		{
			name:     "dollar without brace unchanged",
			input:    "$100 price",
			expected: "$100 price",
		},
		{
			name:     "already escaped gets double escaped",
			input:    "$${already}",
			expected: "$$${already}",
		},
		{
			name:     "complex example from issue",
			input:    `Value Syntax: "${task_key.output_key}" for dynamic values.`,
			expected: `Value Syntax: "$${task_key.output_key}" for dynamic values.`,
		},
		{
			name:     "multiple patterns from issue",
			input:    "${step1.ip} and ${provision_vm.ip_address}",
			expected: "$${step1.ip} and $${provision_vm.ip_address}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := escapeTemplateInterpolation(tt.input)
			if result != tt.expected {
				t.Errorf("escapeTemplateInterpolation(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}
