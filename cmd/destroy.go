// Copyright (c) ElementumAI, Inc.
// SPDX-License-Identifier: MPL-2.0

package cmd

import (
	"fmt"

	"github.com/elementumltd/elementum-cli/terraform"
	"github.com/elementumltd/elementum-cli/ui"
	"github.com/spf13/cobra"
)

var destroyCmd = &cobra.Command{
	Use:   "destroy [flags]",
	Short: "Run terraform/tofu destroy with Elementum provider credentials",
	Long: `Run terraform destroy (or tofu destroy) in the current directory with Elementum
provider credentials automatically injected from your stored authentication.

In interactive mode (default), ei will:
1. Run 'plan -destroy' to show what will be destroyed
2. Prompt for confirmation
3. Destroy the resources if confirmed

For non-interactive use (CI/CD), pass -auto-approve.

All flags are passed through to the underlying terraform/tofu command.

Examples:
  ei destroy                  # Interactive: shows plan and prompts for confirmation
  ei destroy -auto-approve    # Non-interactive: destroys without prompting
  ei destroy -var="foo=bar"   # Pass variables to plan/destroy`,
	DisableFlagParsing: true, // Pass all flags through to terraform/tofu
	SilenceUsage:       true, // Don't print usage on terraform/tofu errors
	RunE:               runDestroy,
}

// GetDestroyCmd returns the destroy command for registration
func GetDestroyCmd() *cobra.Command {
	return destroyCmd
}

func runDestroy(cmd *cobra.Command, args []string) error {
	// Handle help flags specially to show ei's help, not tofu's
	if hasFlag(args, "-h", "--help", "-help") {
		return cmd.Help()
	}

	// Extract ei-specific flags from args (before terraform gets them)
	flags, args := extractEIFlags(args)
	applyEIFlags(cmd, flags)

	// Detect terraform or tofu binary
	binary, err := terraform.DetectBinary()
	if err != nil {
		return err
	}

	// Show which binary we're using
	terraform.ShowDetectedBinary(binary)
	fmt.Println()

	// Fast path: auto-approve provided - pass through directly
	if hasFlag(args, "-auto-approve", "--auto-approve") {
		destroyArgs := append([]string{"destroy"}, args...)
		return terraform.Execute(cmd, binary, destroyArgs)
	}

	// Non-interactive mode: require -auto-approve
	if !isInteractive() {
		fmt.Println(ui.WarningStyle.Render("Non-interactive mode detected."))
		fmt.Println(ui.MutedStyle.Render("Use -auto-approve flag for non-interactive execution."))
		return fmt.Errorf("cannot prompt for confirmation in non-interactive mode")
	}

	// Interactive mode: plan -destroy → confirm → destroy
	return runInteractiveDestroy(cmd, binary, args)
}

func runInteractiveDestroy(cmd *cobra.Command, binary string, args []string) error {
	// Step 1: Show what will be destroyed
	fmt.Println(ui.WarningStyle.Render("Planning destruction..."))
	fmt.Println()

	planArgs := append([]string{"plan", "-destroy"}, args...)
	if err := terraform.Execute(cmd, binary, planArgs); err != nil {
		return err
	}

	fmt.Println()

	// Step 2: Confirm with warning
	confirmed, err := ui.Confirm(
		"Destroy these resources?",
		"This action cannot be undone",
	)
	if err != nil {
		return fmt.Errorf("confirmation failed: %w", err)
	}
	if !confirmed {
		fmt.Println(ui.MutedStyle.Render("Destroy cancelled."))
		return nil
	}

	// Step 3: Execute destroy with auto-approve (we already confirmed)
	fmt.Println()
	fmt.Println(ui.WarningStyle.Render("Destroying resources..."))
	fmt.Println()
	destroyArgs := append([]string{"destroy", "-auto-approve"}, args...)
	return terraform.Execute(cmd, binary, destroyArgs)
}
