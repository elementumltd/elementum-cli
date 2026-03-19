// Copyright (c) ElementumAI, Inc.
// SPDX-License-Identifier: MPL-2.0

package cmd

import (
	"fmt"

	"github.com/elementumltd/elementum-cli/terraform"
	"github.com/spf13/cobra"
)

var planCmd = &cobra.Command{
	Use:   "plan [flags]",
	Short: "Run terraform/tofu plan with Elementum provider credentials",
	Long: `Run terraform plan (or tofu plan) in the current directory with Elementum
provider credentials automatically injected from your stored authentication.

All flags are passed through to the underlying terraform/tofu command.

Examples:
  ei plan
  ei plan -out=tfplan
  ei plan -var="foo=bar"`,
	DisableFlagParsing: true, // Pass all flags through to terraform/tofu
	SilenceUsage:       true, // Don't print usage on terraform/tofu errors
	RunE:               runPlan,
}

// GetPlanCmd returns the plan command for registration
func GetPlanCmd() *cobra.Command {
	return planCmd
}

func runPlan(cmd *cobra.Command, args []string) error {
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

	// Execute plan with all arguments passed through
	planArgs := append([]string{"plan"}, args...)
	return terraform.Execute(cmd, binary, planArgs)
}
