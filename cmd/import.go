// Copyright (c) ElementumAI, Inc.
// SPDX-License-Identifier: MPL-2.0

package cmd

import (
	"fmt"

	"github.com/elementumltd/elementum-cli/terraform"
	"github.com/spf13/cobra"
)

var importCmd = &cobra.Command{
	Use:   "import [flags] RESOURCE_ADDRESS ID",
	Short: "Run terraform/tofu import with Elementum provider credentials",
	Long: `Run terraform import (or tofu import) in the current directory with Elementum
provider credentials automatically injected from your stored authentication.

This command imports existing Elementum resources into your Terraform state,
associating a resource configuration block with an existing resource by its ID.

All flags are passed through to the underlying terraform/tofu command.

Examples:
  ei import elementum_app.my_app abc-123-def
  ei import elementum_field.status field-uuid-here
  ei import -var="foo=bar" elementum_automation.process auto-uuid-456`,
	DisableFlagParsing: true, // Pass all flags through to terraform/tofu
	SilenceUsage:       true, // Don't print usage on terraform/tofu errors
	RunE:               runImport,
}

// GetImportCmd returns the import command for registration
func GetImportCmd() *cobra.Command {
	return importCmd
}

func runImport(cmd *cobra.Command, args []string) error {
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

	// Execute import with all arguments passed through
	importArgs := append([]string{"import"}, args...)
	return terraform.Execute(cmd, binary, importArgs)
}
