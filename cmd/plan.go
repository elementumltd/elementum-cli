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
