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

package ui

import (
	"testing"
)

func TestTruncate(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		maxLength int
		want      string
	}{
		{
			name:      "short string unchanged",
			input:     "hello",
			maxLength: 10,
			want:      "hello",
		},
		{
			name:      "exact length unchanged",
			input:     "hello",
			maxLength: 5,
			want:      "hello",
		},
		{
			name:      "truncated with ellipsis",
			input:     "hello world",
			maxLength: 8,
			want:      "hello...",
		},
		{
			name:      "very short max length",
			input:     "hello",
			maxLength: 2,
			want:      "he",
		},
		{
			name:      "empty string",
			input:     "",
			maxLength: 5,
			want:      "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Truncate(tt.input, tt.maxLength)
			if got != tt.want {
				t.Errorf("Truncate(%q, %d) = %q, want %q", tt.input, tt.maxLength, got, tt.want)
			}
		})
	}
}

func TestTruncateUTF8(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		maxLength int
		want      string
	}{
		{
			name:      "ASCII short string unchanged",
			input:     "hello",
			maxLength: 10,
			want:      "hello",
		},
		{
			name:      "ASCII exact length unchanged",
			input:     "hello",
			maxLength: 5,
			want:      "hello",
		},
		{
			name:      "ASCII truncated with ellipsis",
			input:     "hello world",
			maxLength: 8,
			want:      "hello...",
		},
		{
			name:      "UTF-8 emoji preserved",
			input:     "hello 🌍🌎🌏",
			maxLength: 10,
			want:      "hello 🌍🌎🌏",
		},
		{
			name:      "UTF-8 emoji truncated safely",
			input:     "hello 🌍🌎🌏 world",
			maxLength: 10,
			want:      "hello 🌍...",
		},
		{
			name:      "Chinese characters preserved",
			input:     "你好世界",
			maxLength: 10,
			want:      "你好世界",
		},
		{
			name:      "Chinese characters truncated safely",
			input:     "你好世界，这是一个测试",
			maxLength: 7,
			want:      "你好世界...",
		},
		{
			name:      "mixed UTF-8 truncated at rune boundary",
			input:     "日本語テスト",
			maxLength: 5,
			want:      "日本...",
		},
		{
			name:      "very short max length with UTF-8",
			input:     "日本語",
			maxLength: 2,
			want:      "日本",
		},
		{
			name:      "empty string",
			input:     "",
			maxLength: 5,
			want:      "",
		},
		{
			name:      "accented characters",
			input:     "café résumé naïve",
			maxLength: 10,
			want:      "café ré...",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := TruncateUTF8(tt.input, tt.maxLength)
			if got != tt.want {
				t.Errorf("TruncateUTF8(%q, %d) = %q, want %q", tt.input, tt.maxLength, got, tt.want)
			}
		})
	}
}

func TestTruncateUTF8_DoesNotSplitRunes(t *testing.T) {
	// Specifically test that multi-byte characters are not split
	// The string "日" is 3 bytes in UTF-8
	input := "日本語テスト"

	// Verify the function counts runes, not bytes
	runeCount := len([]rune(input))
	if runeCount != 6 {
		t.Fatalf("expected 6 runes, got %d", runeCount)
	}

	// Truncate to 4 runes (1 + 3 for ellipsis)
	got := TruncateUTF8(input, 4)
	want := "日..."
	if got != want {
		t.Errorf("TruncateUTF8(%q, 4) = %q, want %q", input, got, want)
	}

	// Verify the result is valid UTF-8 and has correct rune count
	gotRunes := []rune(got)
	if len(gotRunes) != 4 {
		t.Errorf("result has %d runes, want 4", len(gotRunes))
	}
}
