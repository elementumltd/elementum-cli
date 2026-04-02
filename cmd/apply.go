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
	"fmt"
	"os"

	"github.com/elementumltd/elementum-cli/terraform"
	"github.com/elementumltd/elementum-cli/ui"
	"github.com/spf13/cobra"
)

var applyCmd = &cobra.Command{
	Use:   "apply [flags]",
	Short: "Run terraform/tofu apply with Elementum provider credentials",
	Long: `Run terraform apply (or tofu apply) in the current directory with Elementum
provider credentials automatically injected from your stored authentication.

In interactive mode (default), ei will:
1. Run 'plan' to show proposed changes
2. Prompt for confirmation
3. Apply the changes if confirmed

For non-interactive use (CI/CD), pass -auto-approve.

All flags are passed through to the underlying terraform/tofu command.

Examples:
  ei apply                    # Interactive: shows plan and prompts for confirmation
  ei apply -auto-approve      # Non-interactive: applies without prompting
  ei apply -var="foo=bar"     # Pass variables to plan/apply
  ei apply tfplan             # Apply a saved plan file directly`,
	DisableFlagParsing: true, // Pass all flags through to terraform/tofu
	SilenceUsage:       true, // Don't print usage on terraform/tofu errors
	RunE:               runApply,
}

// GetApplyCmd returns the apply command for registration
func GetApplyCmd() *cobra.Command {
	return applyCmd
}

func runApply(cmd *cobra.Command, args []string) error {
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

	// Fast path: auto-approve or plan file provided - pass through directly
	if hasFlag(args, "-auto-approve", "--auto-approve") || hasPlanFile(args) {
		applyArgs := append([]string{"apply"}, args...)
		return terraform.Execute(cmd, binary, applyArgs)
	}

	// Non-interactive mode: require -auto-approve
	if !isInteractive() {
		fmt.Println(ui.WarningStyle.Render("Non-interactive mode detected."))
		fmt.Println(ui.MutedStyle.Render("Use -auto-approve flag for non-interactive execution."))
		return fmt.Errorf("cannot prompt for confirmation in non-interactive mode")
	}

	// Interactive mode: plan → confirm → apply
	return runInteractiveApply(cmd, binary, args)
}

func runInteractiveApply(cmd *cobra.Command, binary string, args []string) error {
	planFile := ".ei-plan.tmp"
	defer os.Remove(planFile)

	// Step 1: Run plan and save to temp file
	fmt.Println(ui.InfoStyle.Render("Planning changes..."))
	fmt.Println()

	planArgs := append([]string{"plan", "-out=" + planFile}, args...)
	if err := terraform.Execute(cmd, binary, planArgs); err != nil {
		return err
	}

	fmt.Println()

	// Step 2: Confirm with user
	confirmed, err := ui.Confirm(
		"Apply these changes?",
		"This will execute the plan shown above",
	)
	if err != nil {
		return fmt.Errorf("confirmation failed: %w", err)
	}
	if !confirmed {
		fmt.Println(ui.MutedStyle.Render("Apply cancelled."))
		return nil
	}

	// Step 3: Apply the saved plan
	fmt.Println()
	fmt.Println(ui.InfoStyle.Render("Applying changes..."))
	fmt.Println()
	return terraform.Execute(cmd, binary, []string{"apply", planFile})
}
