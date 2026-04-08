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

package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/elementumltd/elementum-cli/update"
	"github.com/spf13/cobra"
)

var (
	updateCheckOnly bool
	updateForce     bool
	updateYes       bool
)

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update ei to the latest version",
	Long: `Check for and install updates to the ei CLI.

By default, this command will check for a newer version and prompt you to
confirm before installing. Use --check to only check without updating, or
--yes to skip the confirmation prompt.

Examples:
  ei update           # Check and update to latest (with confirmation)
  ei update --check   # Only check, don't update
  ei update --yes     # Update without confirmation
  ei update --force   # Update even if already on latest`,
	SilenceUsage: true,
	RunE:         runUpdate,
}

// GetUpdateCmd returns the update command for registration
func GetUpdateCmd() *cobra.Command {
	return updateCmd
}

func init() {
	updateCmd.Flags().BoolVar(&updateCheckOnly, "check", false, "Only check for updates, don't install")
	updateCmd.Flags().BoolVarP(&updateYes, "yes", "y", false, "Skip confirmation prompt")
	updateCmd.Flags().BoolVar(&updateForce, "force", false, "Force update even if already on latest version")
}

func runUpdate(cmd *cobra.Command, args []string) error {
	// Get current version from root command
	currentVersion := cmd.Root().Version
	if currentVersion == "" {
		currentVersion = "dev"
	}

	// Styles
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6"))
	successStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
	warnStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("3"))
	infoStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("8"))

	fmt.Println(titleStyle.Render("Checking for updates..."))
	fmt.Println()

	// Show current version
	fmt.Printf("Current version: %s\n", infoStyle.Render(currentVersion))

	// Check for updates
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	info, err := update.CheckForUpdate(ctx, currentVersion)
	if err != nil {
		return fmt.Errorf("failed to check for updates: %w", err)
	}

	if info.LatestVersion == "" {
		fmt.Println()
		fmt.Println(warnStyle.Render("No releases found. This might be because:"))
		fmt.Println("  - No releases have been published yet")
		fmt.Println("  - The GitHub API is temporarily unavailable")
		return nil
	}

	fmt.Printf("Latest version:  %s\n", infoStyle.Render(info.LatestVersion))

	if !info.HasUpdate() && !updateForce {
		fmt.Println()
		fmt.Println(successStyle.Render("You're already on the latest version!"))
		return nil
	}

	// Show what's new
	fmt.Println()
	if info.HasUpdate() {
		fmt.Println(successStyle.Render(fmt.Sprintf("A new version is available: %s -> %s", currentVersion, info.LatestVersion)))
	} else {
		fmt.Println(warnStyle.Render("Forcing reinstall of current version..."))
	}

	if !info.PublishedAt.IsZero() {
		fmt.Printf("Released: %s\n", info.PublishedAt.Format("Jan 2, 2006"))
	}

	// Show release notes summary if available
	if info.ReleaseNotes != "" {
		fmt.Println()
		fmt.Println(titleStyle.Render("What's new:"))
		// Truncate long release notes
		notes := info.ReleaseNotes
		lines := strings.Split(notes, "\n")
		if len(lines) > 10 {
			notes = strings.Join(lines[:10], "\n") + "\n..."
		}
		fmt.Println(infoStyle.Render(notes))
	}

	// If check-only, stop here
	if updateCheckOnly {
		fmt.Println()
		fmt.Println("Run 'ei update' to install this version.")
		return nil
	}

	// Confirm unless --yes
	if !updateYes {
		fmt.Println()
		fmt.Print("Do you want to update? [y/N] ")
		var response string
		_, _ = fmt.Scanln(&response)
		response = strings.ToLower(strings.TrimSpace(response))
		if response != "y" && response != "yes" {
			fmt.Println("Update cancelled.")
			return nil
		}
	}

	// Perform the update
	fmt.Println()
	fmt.Println(titleStyle.Render("Downloading and installing..."))

	if err := update.PerformUpdate(ctx, currentVersion); err != nil {
		return fmt.Errorf("update failed: %w", err)
	}

	fmt.Println()
	fmt.Println(successStyle.Render(fmt.Sprintf("Successfully updated to %s!", info.LatestVersion)))
	fmt.Println()
	fmt.Println("Run 'ei --version' to verify the update.")

	return nil
}

// CheckForUpdateNotification checks if there's a new version and returns
// a notification message if so. This is designed to be called from PersistentPostRun
// to show a non-blocking notification after command execution.
func CheckForUpdateNotification(currentVersion string) string {
	// Don't check for dev builds
	if currentVersion == "" || currentVersion == "dev" {
		return ""
	}

	// Check if disabled via environment
	if os.Getenv("EI_DISABLE_UPDATE_CHECK") == "1" {
		return ""
	}

	// Load cache
	cache, err := update.LoadCache()
	if err != nil {
		return "" // Silently fail
	}

	// If cache is fresh and we've already notified, skip
	if !update.IsCacheStale(cache) {
		if cache.Notified {
			return ""
		}
		// Cache is fresh, check if there's an update
		if cache.LatestVersion != "" && cache.LatestVersion != currentVersion {
			// Mark as notified for next time
			_ = update.MarkNotified()
			return fmt.Sprintf(
				"\nA new version of ei is available: %s (current: %s)\nRun 'ei update' to upgrade.\n",
				cache.LatestVersion,
				currentVersion,
			)
		}
		return ""
	}

	// Cache is stale, need to check (but do it in background for next time)
	// For now, just return empty - the actual check happens async
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_, _ = update.CheckForUpdate(ctx, currentVersion)
	}()

	return ""
}
