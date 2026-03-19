// Copyright (c) ElementumAI, Inc.
// SPDX-License-Identifier: MPL-2.0

package cmd

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
	"github.com/elementumltd/elementum-cli/auth"
	"github.com/elementumltd/elementum-cli/internal/client"
	"github.com/spf13/cobra"
)

var (
	successStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Bold(true)
	errorStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Bold(true)
	infoStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("12"))
	labelStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("14")).Bold(true)
)

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Manage authentication",
	Long:  "Authenticate with Elementum and manage credential profiles.",
}

var authLoginCmd = &cobra.Command{
	Use:   "login",
	Short: "Sign in to Elementum",
	Long:  "Interactively sign in to Elementum and store credentials securely.",
	RunE:  runAuthLogin,
}

var authStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show authentication status",
	Long:  "Display current authentication status and active profile.",
	RunE:  runAuthStatus,
}

var authLogoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Sign out",
	Long:  "Remove stored credentials for the current or specified profile.",
	RunE:  runAuthLogout,
}

var authSwitchCmd = &cobra.Command{
	Use:   "switch [profile]",
	Short: "Switch to a different profile",
	Long:  "Change the active authentication profile.",
	Args:  cobra.ExactArgs(1),
	RunE:  runAuthSwitch,
}

var authListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all auth profiles",
	Long:  "Display all configured authentication profiles.",
	RunE:  runAuthList,
}

var authRenameCmd = &cobra.Command{
	Use:   "rename <old-name> <new-name>",
	Short: "Rename a profile",
	Long:  "Rename an existing authentication profile.",
	Args:  cobra.ExactArgs(2),
	RunE:  runAuthRename,
}

var authEnvCmd = &cobra.Command{
	Use:   "env",
	Short: "Print environment variables for current profile",
	Long:  "Output shell export statements for ELEMENTUM_* environment variables.\n\nUsage:\n  eval $(ei auth env)    # Set env vars in current shell\n  ei auth env            # Print export statements",
	RunE:  runAuthEnv,
}

func init() {
	authLoginCmd.Flags().String("profile", "default", "Profile name to save credentials under")
	authLogoutCmd.Flags().String("profile", "", "Profile name to logout (defaults to current profile)")

	authCmd.AddCommand(authLoginCmd)
	authCmd.AddCommand(authStatusCmd)
	authCmd.AddCommand(authLogoutCmd)
	authCmd.AddCommand(authSwitchCmd)
	authCmd.AddCommand(authListCmd)
	authCmd.AddCommand(authRenameCmd)
	authCmd.AddCommand(authEnvCmd)
}

// GetAuthCmd returns the auth command for registration
func GetAuthCmd() *cobra.Command {
	return authCmd
}

func runAuthLogin(cmd *cobra.Command, args []string) error {
	profileName, _ := cmd.Flags().GetString("profile")

	// Show the interactive login form
	fmt.Println()
	loginData, err := auth.ShowLoginForm(profileName)
	if err != nil {
		return fmt.Errorf("login cancelled: %w", err)
	}

	// Test the credentials
	fmt.Println(infoStyle.Render("\n⠿ Testing credentials..."))
	if err := auth.TestCredentials(loginData); err != nil {
		fmt.Println(errorStyle.Render(fmt.Sprintf("✗ %v", err)))
		return err
	}

	// Save the credentials
	if err := auth.SaveLogin(loginData); err != nil {
		fmt.Println(errorStyle.Render(fmt.Sprintf("✗ Failed to save credentials: %v", err)))
		return err
	}

	// Get config path for display
	configPath, _ := auth.GetConfigPath()

	// Success message
	fmt.Println(successStyle.Render("✓ Authenticated successfully!"))
	fmt.Println(successStyle.Render(fmt.Sprintf("✓ Credentials saved to %s", configPath)))
	fmt.Println()
	fmt.Println(labelStyle.Render(fmt.Sprintf("Logged in as: %s", loginData.Organization)))
	fmt.Printf("  %s %s\n", labelStyle.Render("Instance:"), loginData.Instance)
	if loginData.Environment != "" {
		fmt.Printf("  %s %s\n", labelStyle.Render("Environment:"), loginData.Environment)
	}
	if loginData.Instance == "custom" {
		fmt.Printf("  %s %s\n", labelStyle.Render("GraphQL URL:"), loginData.CustomGraphQLURL)
		fmt.Printf("  %s %s\n", labelStyle.Render("API URL:"), loginData.CustomAPIURL)
	} else {
		fmt.Printf("  %s %s\n", labelStyle.Render("Endpoint:"), formatEndpointURL(loginData.Organization, loginData.Instance, loginData.Environment, loginData.CustomBaseURL))
	}
	fmt.Println()

	return nil
}

func runAuthStatus(cmd *cobra.Command, args []string) error {
	// Load config
	config, err := auth.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Check if any profiles exist
	if len(config.Profiles) == 0 {
		fmt.Println(infoStyle.Render("Not logged in. Run 'ei auth login' to sign in."))
		return nil
	}

	// Get current profile
	profile, err := auth.GetCurrentProfile(config)
	if err != nil {
		return fmt.Errorf("failed to get current profile: %w", err)
	}

	// Display status
	fmt.Println()
	fmt.Println(labelStyle.Render("Authentication Status"))
	fmt.Println()
	fmt.Printf("  %s %s\n", labelStyle.Render("Profile:"), config.CurrentProfile)
	fmt.Printf("  %s %s\n", labelStyle.Render("Organization:"), profile.Organization)
	fmt.Printf("  %s %s\n", labelStyle.Render("Instance:"), profile.Instance)
	if profile.Environment != "" {
		fmt.Printf("  %s %s\n", labelStyle.Render("Environment:"), profile.Environment)
	}
	if profile.Instance == "custom" {
		if profile.CustomGraphQLURL != "" {
			fmt.Printf("  %s %s\n", labelStyle.Render("GraphQL URL:"), profile.CustomGraphQLURL)
		}
		if profile.CustomAPIURL != "" {
			fmt.Printf("  %s %s\n", labelStyle.Render("API URL:"), profile.CustomAPIURL)
		}
		// Fallback to legacy CustomBaseURL if new URLs not set
		if profile.CustomGraphQLURL == "" && profile.CustomAPIURL == "" && profile.CustomBaseURL != "" {
			fmt.Printf("  %s %s\n", labelStyle.Render("Endpoint:"), profile.CustomBaseURL)
		}
	} else {
		fmt.Printf("  %s %s\n", labelStyle.Render("Endpoint:"), formatEndpointURL(profile.Organization, profile.Instance, profile.Environment, profile.CustomBaseURL))
	}
	fmt.Printf("  %s %s...\n", labelStyle.Render("Client ID:"), truncate(profile.ClientID, 12))
	fmt.Println()

	// List all profiles
	profiles := auth.ListProfiles(config)
	if len(profiles) > 1 {
		fmt.Println(labelStyle.Render("Available Profiles:"))
		for _, name := range profiles {
			if name == config.CurrentProfile {
				fmt.Printf("  • %s %s\n", name, successStyle.Render("(active)"))
			} else {
				fmt.Printf("  • %s\n", name)
			}
		}
		fmt.Println()
	}

	// Get config path
	configPath, _ := auth.GetConfigPath()
	fmt.Println(infoStyle.Render(fmt.Sprintf("Config: %s", configPath)))
	fmt.Println()

	return nil
}

func runAuthLogout(cmd *cobra.Command, args []string) error {
	// Load config
	config, err := auth.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Determine which profile to logout
	profileName, _ := cmd.Flags().GetString("profile")
	if profileName == "" {
		profileName = config.CurrentProfile
	}

	// Check if profile exists
	if _, exists := config.Profiles[profileName]; !exists {
		return fmt.Errorf("profile %q not found", profileName)
	}

	// Delete secret from keychain
	if err := auth.DeleteSecret(profileName); err != nil {
		fmt.Println(infoStyle.Render(fmt.Sprintf("Warning: could not delete secret from keychain: %v", err)))
	}

	// Delete profile from config
	if err := auth.DeleteProfile(config, profileName); err != nil {
		return err
	}

	// Save config
	if err := auth.SaveConfig(config); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Println(successStyle.Render(fmt.Sprintf("✓ Logged out of profile %q", profileName)))

	// Show new current profile if any
	if len(config.Profiles) > 0 {
		fmt.Println(infoStyle.Render(fmt.Sprintf("Current profile is now: %s", config.CurrentProfile)))
	} else {
		fmt.Println(infoStyle.Render("No profiles remaining. Run 'ei auth login' to sign in."))
	}
	fmt.Println()

	return nil
}

func runAuthSwitch(cmd *cobra.Command, args []string) error {
	profileName := args[0]

	// Load config
	config, err := auth.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Switch profile
	if err := auth.SwitchProfile(config, profileName); err != nil {
		return err
	}

	// Save config
	if err := auth.SaveConfig(config); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	// Get the profile details
	profile := config.Profiles[profileName]

	fmt.Println(successStyle.Render(fmt.Sprintf("✓ Switched to profile %q", profileName)))
	fmt.Println(infoStyle.Render(fmt.Sprintf("  Organization: %s", profile.Organization)))
	fmt.Println(infoStyle.Render(fmt.Sprintf("  Instance: %s", profile.Instance)))
	if profile.Environment != "" {
		fmt.Println(infoStyle.Render(fmt.Sprintf("  Environment: %s", profile.Environment)))
	}
	if profile.Instance == "custom" {
		if profile.CustomGraphQLURL != "" {
			fmt.Println(infoStyle.Render(fmt.Sprintf("  GraphQL URL: %s", profile.CustomGraphQLURL)))
		}
		if profile.CustomAPIURL != "" {
			fmt.Println(infoStyle.Render(fmt.Sprintf("  API URL: %s", profile.CustomAPIURL)))
		}
		if profile.CustomGraphQLURL == "" && profile.CustomAPIURL == "" && profile.CustomBaseURL != "" {
			fmt.Println(infoStyle.Render(fmt.Sprintf("  Endpoint: %s", profile.CustomBaseURL)))
		}
	} else {
		fmt.Println(infoStyle.Render(fmt.Sprintf("  Endpoint: %s", formatEndpointURL(profile.Organization, profile.Instance, profile.Environment, profile.CustomBaseURL))))
	}
	fmt.Println()

	return nil
}

func runAuthList(cmd *cobra.Command, args []string) error {
	// Load config
	config, err := auth.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Check if any profiles exist
	if len(config.Profiles) == 0 {
		fmt.Println(infoStyle.Render("No profiles configured. Run 'ei auth login' to create a profile."))
		return nil
	}

	// Display header
	fmt.Println()
	fmt.Println(labelStyle.Render("Auth Profiles"))
	fmt.Println()

	// List all profiles with details
	profiles := auth.ListProfiles(config)
	for _, name := range profiles {
		profile := config.Profiles[name]
		isActive := name == config.CurrentProfile

		// Profile name with active indicator
		if isActive {
			fmt.Printf("  %s %s\n", labelStyle.Render("●"), successStyle.Render(name+" (active)"))
		} else {
			fmt.Printf("  %s %s\n", infoStyle.Render("○"), name)
		}

		// Profile details
		fmt.Printf("    %s %s\n", labelStyle.Render("Organization:"), profile.Organization)
		fmt.Printf("    %s %s\n", labelStyle.Render("Instance:"), profile.Instance)
		if profile.Environment != "" {
			fmt.Printf("    %s %s\n", labelStyle.Render("Environment:"), profile.Environment)
		}
		if profile.Instance == "custom" {
			if profile.CustomGraphQLURL != "" {
				fmt.Printf("    %s %s\n", labelStyle.Render("GraphQL URL:"), profile.CustomGraphQLURL)
			}
			if profile.CustomAPIURL != "" {
				fmt.Printf("    %s %s\n", labelStyle.Render("API URL:"), profile.CustomAPIURL)
			}
			if profile.CustomGraphQLURL == "" && profile.CustomAPIURL == "" && profile.CustomBaseURL != "" {
				fmt.Printf("    %s %s\n", labelStyle.Render("Endpoint:"), profile.CustomBaseURL)
			}
		} else {
			fmt.Printf("    %s %s\n", labelStyle.Render("Endpoint:"), formatEndpointURL(profile.Organization, profile.Instance, profile.Environment, profile.CustomBaseURL))
		}
		fmt.Printf("    %s %s...\n", labelStyle.Render("Client ID:"), truncate(profile.ClientID, 12))
		fmt.Println()
	}

	// Get config path
	configPath, _ := auth.GetConfigPath()
	fmt.Println(infoStyle.Render(fmt.Sprintf("Config: %s", configPath)))
	fmt.Println()

	return nil
}

func runAuthRename(cmd *cobra.Command, args []string) error {
	oldName := args[0]
	newName := args[1]

	// Validate new name is different
	if oldName == newName {
		return fmt.Errorf("new name must be different from old name")
	}

	// Load config
	config, err := auth.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Check that old profile exists
	oldProfile, exists := config.Profiles[oldName]
	if !exists {
		return fmt.Errorf("profile %q not found", oldName)
	}

	// Check that new profile doesn't exist
	if _, exists := config.Profiles[newName]; exists {
		return fmt.Errorf("profile %q already exists", newName)
	}

	// Get the secret from the old profile's keychain entry
	secret, err := auth.GetSecret(oldName, oldProfile.ClientSecret)
	if err != nil {
		return fmt.Errorf("failed to retrieve secret: %w", err)
	}

	// Store the secret under the new profile name
	newSecretRef, err := auth.StoreSecret(newName, secret)
	if err != nil {
		fmt.Println(infoStyle.Render(fmt.Sprintf("Warning: %v", err)))
	}

	// Create new profile with the new secret reference
	newProfile := &auth.Profile{
		Organization:     oldProfile.Organization,
		Instance:         oldProfile.Instance,
		Environment:      oldProfile.Environment,
		CustomBaseURL:    oldProfile.CustomBaseURL,
		CustomGraphQLURL: oldProfile.CustomGraphQLURL,
		CustomAPIURL:     oldProfile.CustomAPIURL,
		ClientID:         oldProfile.ClientID,
		ClientSecret:     newSecretRef,
	}

	// Add new profile and remove old profile
	auth.SetProfile(config, newName, newProfile)
	delete(config.Profiles, oldName)

	// Update current profile if it was the old name
	if config.CurrentProfile == oldName {
		config.CurrentProfile = newName
	}

	// Save config
	if err := auth.SaveConfig(config); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	// Delete old secret from keychain
	if err := auth.DeleteSecret(oldName); err != nil {
		fmt.Println(infoStyle.Render(fmt.Sprintf("Warning: could not delete old secret from keychain: %v", err)))
	}

	fmt.Println(successStyle.Render(fmt.Sprintf("✓ Renamed profile %q to %q", oldName, newName)))
	fmt.Println()

	return nil
}

func runAuthEnv(cmd *cobra.Command, args []string) error {
	// Load config
	config, err := auth.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Check if any profiles exist
	if len(config.Profiles) == 0 {
		return fmt.Errorf("not logged in - run 'ei auth login' first")
	}

	// Get current profile
	profile, err := auth.GetCurrentProfile(config)
	if err != nil {
		return fmt.Errorf("failed to get current profile: %w", err)
	}

	// Retrieve the actual client secret
	secret, err := auth.GetSecret(config.CurrentProfile, profile.ClientSecret)
	if err != nil {
		return fmt.Errorf("failed to retrieve client secret: %w", err)
	}

	// Output export statements
	fmt.Printf("export ELEMENTUM_ORGANIZATION=%q\n", profile.Organization)
	fmt.Printf("export ELEMENTUM_CLIENT_ID=%q\n", profile.ClientID)
	fmt.Printf("export ELEMENTUM_CLIENT_SECRET=%q\n", secret)

	if profile.Instance != "" && profile.Instance != "us" {
		fmt.Printf("export ELEMENTUM_INSTANCE=%q\n", profile.Instance)
	}

	if profile.Environment != "" {
		fmt.Printf("export ELEMENTUM_ENVIRONMENT=%q\n", profile.Environment)
	}

	return nil
}

// truncate truncates a string to the specified length and adds "..."
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// formatEndpointURL formats the Elementum endpoint URL for display.
// Delegates to client.FormatBaseURL for consistent URL construction.
func formatEndpointURL(organization, instance, environment, customBaseURL string) string {
	return client.FormatBaseURL(organization, environment, client.Instance(instance), customBaseURL)
}
