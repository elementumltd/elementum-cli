// Copyright (c) ElementumAI, Inc.
// SPDX-License-Identifier: MPL-2.0

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
