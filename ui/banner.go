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

// Accent colors for visual flair
var (
	ColorPurple = lipgloss.Color("129") // Purple
	ColorBlue   = lipgloss.Color("27")  // Blue
	ColorRed    = lipgloss.Color("196") // Red
	ColorOrange = lipgloss.Color("208") // Orange
	ColorGreen  = lipgloss.Color("46")  // Green
	ColorYellow = lipgloss.Color("226") // Yellow
)

// Banner styles
var (
	bannerMainStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("15")) // White

	bannerGlowStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("117")) // Light blue glow
)

// Stone renders a single stone character with the appropriate color
func stone(s string, color lipgloss.Color) string {
	return lipgloss.NewStyle().Foreground(color).Bold(true).Render(s)
}

// RenderBanner returns the Elementum Infinity banner
func RenderBanner(version string) string {
	// Build the banner - 59 chars internal width
	lines := []string{
		"",
		bannerGlowStyle.Render("    ╔═════════════════════════════════════════════════════════╗"),
		bannerGlowStyle.Render("    ║") + bannerMainStyle.Render(" ██╗███╗   ██╗███████╗██╗███╗   ██╗██╗████████╗██╗   ██╗ ") + bannerGlowStyle.Render("║"),
		bannerGlowStyle.Render("    ║") + bannerMainStyle.Render(" ██║████╗  ██║██╔════╝██║████╗  ██║██║╚══██╔══╝╚██╗ ██╔╝ ") + bannerGlowStyle.Render("║"),
		bannerGlowStyle.Render("    ║") + bannerMainStyle.Render(" ██║██╔██╗ ██║█████╗  ██║██╔██╗ ██║██║   ██║    ╚████╔╝  ") + bannerGlowStyle.Render("║"),
		bannerGlowStyle.Render("    ║") + bannerMainStyle.Render(" ██║██║╚██╗██║██╔══╝  ██║██║╚██╗██║██║   ██║     ╚██╔╝   ") + bannerGlowStyle.Render("║"),
		bannerGlowStyle.Render("    ║") + bannerMainStyle.Render(" ██║██║ ╚████║██║     ██║██║ ╚████║██║   ██║      ██║    ") + bannerGlowStyle.Render("║"),
		bannerGlowStyle.Render("    ║") + bannerMainStyle.Render(" ╚═╝╚═╝  ╚═══╝╚═╝     ╚═╝╚═╝  ╚═══╝╚═╝   ╚═╝      ╚═╝    ") + bannerGlowStyle.Render("║"),
		bannerGlowStyle.Render("    ╠═════════════════════════════════════════════════════════╣"),
		bannerGlowStyle.Render("    ║") + renderAccentLine() + bannerGlowStyle.Render("║"),
		bannerGlowStyle.Render("    ╚═════════════════════════════════════════════════════════╝"),
		"",
	}

	return strings.Join(lines, "\n")
}

// renderAccentLine creates a colorful accent line
func renderAccentLine() string {
	// 59 chars total width inside the box (matching INFINITY text width)
	return fmt.Sprintf(
		"          %s  %s  %s  %s  %s  %s  elementum                    ",
		stone("*", ColorPurple),
		stone("*", ColorBlue),
		stone("*", ColorRed),
		stone("*", ColorOrange),
		stone("*", ColorGreen),
		stone("*", ColorYellow),
	)
}

// RenderCompactBanner returns a smaller banner for help output
func RenderCompactBanner() string {
	return fmt.Sprintf(
		"%s %s",
		bannerMainStyle.Render("INFINITY"),
		MutedStyle.Render("- elementum"),
	)
}
