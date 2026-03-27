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
