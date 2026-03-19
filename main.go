// Copyright (c) ElementumAI, Inc.
// SPDX-License-Identifier: MPL-2.0

package main

import (
	"fmt"
	"os"

	elementumcmd "github.com/elementumltd/elementum-cli/cmd"
	"github.com/elementumltd/elementum-cli/logger"
	"github.com/elementumltd/elementum-cli/ui"
	"github.com/spf13/cobra"
)

var version = "dev"

var rootCmd = &cobra.Command{
	Use:   "ei",
	Short: "Elementum Infinity - The power to create and destroy",
	Long: `A CLI tool for discovering and exporting Elementum resources to Terraform.

Use this tool to generate import configurations for terraform plan -generate-config-out,
making it easy to manage your Elementum apps with Infrastructure as Code.`,
	Version: version,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		// Determine log level from flags and environment
		level := getLogLevel(cmd)
		logger.Init(level)

		// Start background update check (non-blocking)
		go checkForUpdateAsync(cmd)
	},
	PersistentPostRun: func(cmd *cobra.Command, args []string) {
		// Show update notification if available (non-blocking check from cache)
		if notification := elementumcmd.CheckForUpdateNotification(version); notification != "" {
			fmt.Fprint(os.Stderr, notification)
		}
	},
	Run: func(cmd *cobra.Command, args []string) {
		// Show the banner when no subcommand is provided
		fmt.Println(ui.RenderBanner(version))
		cmd.Help()
	},
}

// checkForUpdateAsync performs a background version check without blocking.
// This populates the cache so the PersistentPostRun can show a notification.
func checkForUpdateAsync(cmd *cobra.Command) {
	// Skip for dev builds or if disabled
	if version == "" || version == "dev" {
		return
	}
	if os.Getenv("EI_DISABLE_UPDATE_CHECK") == "1" {
		return
	}

	// Skip for the update command itself
	if cmd.Name() == "update" {
		return
	}

	// The actual check happens in CheckForUpdateNotification via cache
	// This goroutine just ensures the cache gets populated in the background
}

// getLogLevel determines the effective log level from flags and environment.
// Priority: --quiet > --log-level > --debug > ELEMENTUM_LOG_LEVEL > default (info)
func getLogLevel(cmd *cobra.Command) logger.Level {
	// 1. Check --quiet flag (highest priority)
	if quiet, _ := cmd.Flags().GetBool("quiet"); quiet {
		return logger.LevelError
	}

	// 2. Check --log-level flag
	if lvl, _ := cmd.Flags().GetString("log-level"); lvl != "" {
		return logger.ParseLevel(lvl)
	}

	// 3. Check --debug flag (backwards compatibility)
	if debug, _ := cmd.Flags().GetBool("debug"); debug {
		return logger.LevelDebug
	}

	// 4. Check ELEMENTUM_LOG_LEVEL environment variable
	if lvl := os.Getenv("ELEMENTUM_LOG_LEVEL"); lvl != "" {
		return logger.ParseLevel(lvl)
	}

	// 5. Default to info
	return logger.LevelInfo
}

func init() {
	// Global flags
	rootCmd.PersistentFlags().String("org", "", "Organization ID (overrides config)")
	rootCmd.PersistentFlags().String("client-id", "", "Client ID (overrides config)")
	rootCmd.PersistentFlags().String("client-secret", "", "Client Secret (overrides config)")
	rootCmd.PersistentFlags().String("instance", "", "Instance/region: us, eu, stage, dev, custom (overrides config)")
	rootCmd.PersistentFlags().String("environment", "", "Organization environment slug, e.g. 'staging' (overrides config)")
	rootCmd.PersistentFlags().String("custom-base-url", "", "Custom base URL for custom instances (deprecated: use --custom-graphql-url and --custom-api-url)")
	rootCmd.PersistentFlags().String("custom-graphql-url", "", "Custom GraphQL URL for custom instances (e.g., http://localhost:3000)")
	rootCmd.PersistentFlags().String("custom-api-url", "", "Custom API/OAuth URL for custom instances (e.g., http://localhost:8080)")
	rootCmd.PersistentFlags().String("profile", "", "Config profile to use")

	// Logging flags
	rootCmd.PersistentFlags().String("log-level", "", "Log level: trace, debug, info, warn, error, off (default: info)")
	rootCmd.PersistentFlags().BoolP("quiet", "q", false, "Suppress all non-error output (shortcut for --log-level error)")
	rootCmd.PersistentFlags().Bool("debug", false, "Enable debug logging (shortcut for --log-level debug)")

	// Output format flags
	rootCmd.PersistentFlags().Bool("json", false, "Output in JSON format")

	// Register commands

	// Core authentication and operations
	rootCmd.AddCommand(elementumcmd.GetAuthCmd())
	rootCmd.AddCommand(elementumcmd.GetRefsCmd())
	rootCmd.AddCommand(elementumcmd.GetPlanCmd())
	rootCmd.AddCommand(elementumcmd.GetApplyCmd())
	rootCmd.AddCommand(elementumcmd.GetDestroyCmd())
	rootCmd.AddCommand(elementumcmd.GetImportCmd())
	rootCmd.AddCommand(elementumcmd.GetChatCmd())
	rootCmd.AddCommand(elementumcmd.GetConversationCmd())
	rootCmd.AddCommand(elementumcmd.GetUpdateCmd())

	// Resource commands (ei <resource> <verb> pattern)
	rootCmd.AddCommand(elementumcmd.GetAppsCmd())
	rootCmd.AddCommand(elementumcmd.GetElementsCmd())
	rootCmd.AddCommand(elementumcmd.GetGroupsCmd())
	rootCmd.AddCommand(elementumcmd.GetTablesCmd())
	rootCmd.AddCommand(elementumcmd.GetTasksCmd())
	rootCmd.AddCommand(elementumcmd.GetDataminesCmd())
	rootCmd.AddCommand(elementumcmd.GetFieldsCmd())
	rootCmd.AddCommand(elementumcmd.GetLayoutsCmd())
	rootCmd.AddCommand(elementumcmd.GetRecordsCmd())
	rootCmd.AddCommand(elementumcmd.GetAutomationsCmd())
	rootCmd.AddCommand(elementumcmd.GetAgentsCmd())
	rootCmd.AddCommand(elementumcmd.GetSkillsCmd())
	rootCmd.AddCommand(elementumcmd.GetSkillToolsCmd())
	rootCmd.AddCommand(elementumcmd.GetAgentToolsCmd())
	rootCmd.AddCommand(elementumcmd.GetObjectsCmd())
	rootCmd.AddCommand(elementumcmd.GetCategoriesCmd())
	rootCmd.AddCommand(elementumcmd.GetCloudlinksCmd())
	rootCmd.AddCommand(elementumcmd.GetUsersCmd())
	rootCmd.AddCommand(elementumcmd.GetApprovalsCmd())
	rootCmd.AddCommand(elementumcmd.GetInterventionsCmd())
	rootCmd.AddCommand(elementumcmd.GetFileReadersCmd())
	rootCmd.AddCommand(elementumcmd.GetTableCmd())

	// Infrastructure/config commands
	rootCmd.AddCommand(elementumcmd.GetAiProvidersCmd())
	rootCmd.AddCommand(elementumcmd.GetAiServicesCmd())
	rootCmd.AddCommand(elementumcmd.GetFeatureFlagsCmd())
	rootCmd.AddCommand(elementumcmd.GetPhoneProvidersCmd())
	rootCmd.AddCommand(elementumcmd.GetPhoneServicesCmd())
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
