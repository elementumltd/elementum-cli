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

package terraform

import (
	"fmt"
	"os/exec"
	"sync"

	"github.com/elementumltd/elementum-cli/ui"
)

var (
	detectedBinary     string
	detectedBinaryOnce sync.Once
	detectionError     error
)

// DetectBinary detects whether terraform or tofu is available in PATH.
// It prefers tofu if both are available, and caches the result.
// Returns the binary name ("terraform" or "tofu") or an error with helpful installation instructions.
func DetectBinary() (string, error) {
	detectedBinaryOnce.Do(func() {
		// Prefer tofu over terraform
		if _, err := exec.LookPath("tofu"); err == nil {
			detectedBinary = "tofu"
			return
		}

		if _, err := exec.LookPath("terraform"); err == nil {
			detectedBinary = "terraform"
			return
		}

		// Neither found, return helpful error message
		detectionError = fmt.Errorf("%s\n\n%s\n\n%s\n%s\n%s\n\n%s\n%s\n%s",
			ui.ErrorStyle.Render("✗ Neither terraform nor tofu found in PATH"),
			"Please install one of the following:",
			ui.SuccessStyle.Render("• OpenTofu (recommended):"),
			"  brew install opentofu",
			"  or visit: https://opentofu.org/docs/intro/install",
			ui.InfoStyle.Render("• Terraform:"),
			"  brew install terraform",
			"  or visit: https://developer.hashicorp.com/terraform/install",
		)
	})

	return detectedBinary, detectionError
}

// ShowDetectedBinary prints which binary was detected
func ShowDetectedBinary(binary string) {
	fmt.Println(ui.InfoStyle.Render(fmt.Sprintf("⠿ Using %s", binary)))
}
