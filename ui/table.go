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
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Table represents a simple table for displaying data
type Table struct {
	Headers []string
	Rows    [][]string
	Widths  []int // Column widths (auto-calculated if nil)
}

// NewTable creates a new table
func NewTable(headers []string) *Table {
	return &Table{
		Headers: headers,
		Rows:    [][]string{},
	}
}

// AddRow adds a row to the table
func (t *Table) AddRow(cells ...string) {
	t.Rows = append(t.Rows, cells)
}

// calculateWidths calculates optimal column widths
func (t *Table) calculateWidths() []int {
	if len(t.Headers) == 0 {
		return []int{}
	}

	widths := make([]int, len(t.Headers))

	// Start with header widths
	for i, header := range t.Headers {
		widths[i] = len(header)
	}

	// Check all rows
	for _, row := range t.Rows {
		for i, cell := range row {
			if i < len(widths) && len(cell) > widths[i] {
				widths[i] = len(cell)
			}
		}
	}

	return widths
}

// Render renders the table as a string
func (t *Table) Render() string {
	if len(t.Headers) == 0 {
		return ""
	}

	// Calculate column widths if not set
	widths := t.Widths
	if widths == nil {
		widths = t.calculateWidths()
	}

	var b strings.Builder
	borderColor := lipgloss.NewStyle().Foreground(ColorPrimary)
	separatorColor := lipgloss.NewStyle().Foreground(ColorMuted)

	// Calculate total width (content + padding + separators)
	totalWidth := 0
	for _, w := range widths {
		totalWidth += w + 2 // +2 for left/right padding
	}
	totalWidth += len(widths) - 1 // For separators

	// Top border
	b.WriteString(borderColor.Render("┌" + strings.Repeat("─", totalWidth) + "┐"))
	b.WriteString("\n")

	// Headers
	b.WriteString(borderColor.Render("│"))
	for i, header := range t.Headers {
		padded := " " + padCenter(header, widths[i]) + " "
		b.WriteString(TableHeaderStyle.Render(padded))
		if i < len(t.Headers)-1 {
			b.WriteString(separatorColor.Render("│"))
		}
	}
	b.WriteString(borderColor.Render("│"))
	b.WriteString("\n")

	// Header separator
	b.WriteString(separatorColor.Render("├" + strings.Repeat("─", totalWidth) + "┤"))
	b.WriteString("\n")

	// Rows
	for rowIdx, row := range t.Rows {
		b.WriteString(borderColor.Render("│"))
		for i := 0; i < len(widths); i++ {
			var content string
			if i < len(row) {
				content = row[i]
			} else {
				content = ""
			}
			padded := " " + padRight(content, widths[i]) + " "
			b.WriteString(padded)
			if i < len(widths)-1 {
				b.WriteString(separatorColor.Render("│"))
			}
		}
		b.WriteString(borderColor.Render("│"))
		if rowIdx < len(t.Rows)-1 {
			b.WriteString("\n")
		}
	}

	// Bottom border
	b.WriteString("\n")
	b.WriteString(borderColor.Render("└" + strings.Repeat("─", totalWidth) + "┘"))

	return b.String()
}

// padRight pads a string to the right
func padRight(s string, width int) string {
	if len(s) >= width {
		return s[:width]
	}
	return s + strings.Repeat(" ", width-len(s))
}

// padCenter centers a string in the given width
func padCenter(s string, width int) string {
	if len(s) >= width {
		return s[:width]
	}
	leftPad := (width - len(s)) / 2
	rightPad := width - len(s) - leftPad
	return strings.Repeat(" ", leftPad) + s + strings.Repeat(" ", rightPad)
}

// RenderSimple renders a simpler table without borders (for compact display)
func (t *Table) RenderSimple() string {
	if len(t.Headers) == 0 {
		return ""
	}

	widths := t.Widths
	if widths == nil {
		widths = t.calculateWidths()
	}

	var b strings.Builder

	// Headers
	for i, header := range t.Headers {
		b.WriteString(TableHeaderStyle.Render(padRight(header, widths[i])))
		if i < len(t.Headers)-1 {
			b.WriteString("  ")
		}
	}
	b.WriteString("\n")

	// Separator
	for i, width := range widths {
		b.WriteString(MutedStyle.Render(strings.Repeat("─", width)))
		if i < len(widths)-1 {
			b.WriteString("  ")
		}
	}
	b.WriteString("\n")

	// Rows
	for _, row := range t.Rows {
		for i := 0; i < len(widths); i++ {
			var cell string
			if i < len(row) {
				cell = padRight(row[i], widths[i])
			} else {
				cell = padRight("", widths[i])
			}
			b.WriteString(cell)
			if i < len(widths)-1 {
				b.WriteString("  ")
			}
		}
		b.WriteString("\n")
	}

	return b.String()
}

// Example function to demonstrate usage
func ExampleTable() {
	table := NewTable([]string{"Name", "ID", "Fields"})
	table.AddRow("IT Support", "app_abc123", "12 fields")
	table.AddRow("Procurement", "app_def456", "8 fields")
	table.AddRow("Onboarding", "app_ghi789", "15 fields")

	fmt.Println(table.Render())
}
