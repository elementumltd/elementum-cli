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
	"github.com/charmbracelet/lipgloss"
)

// Color palette
var (
	ColorPrimary   = lipgloss.Color("12") // Cyan/Blue
	ColorSecondary = lipgloss.Color("14") // Light Cyan
	ColorSuccess   = lipgloss.Color("10") // Green
	ColorWarning   = lipgloss.Color("11") // Yellow
	ColorError     = lipgloss.Color("9")  // Red
	ColorMuted     = lipgloss.Color("8")  // Gray
	ColorHighlight = lipgloss.Color("13") // Magenta
)

// Text styles
var (
	TitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorPrimary).
			MarginBottom(1)

	SubtitleStyle = lipgloss.NewStyle().
			Foreground(ColorSecondary).
			MarginBottom(1)

	SuccessStyle = lipgloss.NewStyle().
			Foreground(ColorSuccess).
			Bold(true)

	ErrorStyle = lipgloss.NewStyle().
			Foreground(ColorError).
			Bold(true)

	WarningStyle = lipgloss.NewStyle().
			Foreground(ColorWarning).
			Bold(true)

	InfoStyle = lipgloss.NewStyle().
			Foreground(ColorPrimary)

	MutedStyle = lipgloss.NewStyle().
			Foreground(ColorMuted)

	LabelStyle = lipgloss.NewStyle().
			Foreground(ColorSecondary).
			Bold(true)

	ValueStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("15")) // White

	HighlightStyle = lipgloss.NewStyle().
			Foreground(ColorHighlight).
			Bold(true)
)

// Box styles
var (
	BoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorPrimary).
			Padding(1, 2)

	HeaderBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorPrimary).
			Padding(0, 2).
			Bold(true)
)

// Tree view styles
var (
	TreeBranchStyle = lipgloss.NewStyle().
			Foreground(ColorMuted)

	TreeItemStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("15"))

	TreeMetaStyle = lipgloss.NewStyle().
			Foreground(ColorMuted).
			Italic(true)

	TreeHighlightStyle = lipgloss.NewStyle().
				Foreground(ColorHighlight)
)

// Table styles
var (
	TableHeaderStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(ColorPrimary)

	TableCellStyle = lipgloss.NewStyle()

	TableBorderStyle = lipgloss.NewStyle().
				BorderStyle(lipgloss.RoundedBorder()).
				BorderForeground(ColorPrimary)
)

// Helper functions

// RenderCheckmark returns a styled checkmark
func RenderCheckmark() string {
	return SuccessStyle.Render("✓")
}

// RenderCross returns a styled cross
func RenderCross() string {
	return ErrorStyle.Render("✗")
}

// RenderSpinner returns a styled spinner character
func RenderSpinner(char string) string {
	return InfoStyle.Render(char)
}

// RenderBullet returns a styled bullet point
func RenderBullet() string {
	return MutedStyle.Render("•")
}

// RenderArrow returns a styled arrow
func RenderArrow() string {
	return InfoStyle.Render("→")
}

// Truncate truncates a string to maxLength and adds ellipsis if needed
func Truncate(s string, maxLength int) string {
	if len(s) <= maxLength {
		return s
	}
	if maxLength <= 3 {
		return s[:maxLength]
	}
	return s[:maxLength-3] + "..."
}

// Pad pads a string to the specified width
func Pad(s string, width int) string {
	if len(s) >= width {
		return s
	}
	return s + lipgloss.NewStyle().Width(width-len(s)).Render("")
}
