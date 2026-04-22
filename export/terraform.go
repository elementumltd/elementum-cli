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

package export

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/elementumltd/elementum-cli/logger"
)

// TerraformRunner handles running Terraform commands
type TerraformRunner struct {
	WorkDir string
}

// NewTerraformRunner creates a new Terraform runner with a temporary working directory
func NewTerraformRunner() (*TerraformRunner, error) {
	tmpDir, err := os.MkdirTemp("", "elementum-export-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp directory: %w", err)
	}

	return &TerraformRunner{
		WorkDir: tmpDir,
	}, nil
}

// Cleanup removes the temporary working directory
func (tr *TerraformRunner) Cleanup() error {
	if tr.WorkDir != "" {
		return os.RemoveAll(tr.WorkDir)
	}
	return nil
}

// WriteImports writes import blocks and provider config to the working directory
func (tr *TerraformRunner) WriteImports(providerConfig, imports string) error {
	// Write provider.tf
	providerFile := filepath.Join(tr.WorkDir, "provider.tf")
	if err := os.WriteFile(providerFile, []byte(providerConfig), 0644); err != nil {
		return fmt.Errorf("failed to write provider config: %w", err)
	}

	// Write imports.tf
	importsFile := filepath.Join(tr.WorkDir, "imports.tf")
	if err := os.WriteFile(importsFile, []byte(imports), 0644); err != nil {
		return fmt.Errorf("failed to write imports: %w", err)
	}

	// Check if dev override exists - if so, copy provider binary
	if err := tr.setupLocalProvider(); err != nil {
		logger.Warn("could not setup local provider (not fatal)", "error", err)
	}

	return nil
}

// setupLocalProvider checks for a dev override and copies the provider to local plugins
func (tr *TerraformRunner) setupLocalProvider() error {
	// Check common locations for the provider binary
	possiblePaths := []string{
		filepath.Join(os.Getenv("HOME"), "go", "bin", "terraform-provider-elementum"),
		"./terraform-provider-elementum",
	}

	var providerBinary string
	for _, path := range possiblePaths {
		if _, err := os.Stat(path); err == nil {
			providerBinary = path
			break
		}
	}

	if providerBinary == "" {
		return fmt.Errorf("provider binary not found")
	}

	// Create plugins directory
	pluginsDir := filepath.Join(tr.WorkDir, ".terraform", "providers", "registry.opentofu.org", "elementumltd", "elementum", "0.1.0", "darwin_arm64")
	if err := os.MkdirAll(pluginsDir, 0755); err != nil {
		return fmt.Errorf("failed to create plugins directory: %w", err)
	}

	// Copy provider binary
	destPath := filepath.Join(pluginsDir, "terraform-provider-elementum")
	input, err := os.ReadFile(providerBinary)
	if err != nil {
		return fmt.Errorf("failed to read provider binary: %w", err)
	}

	if err := os.WriteFile(destPath, input, 0755); err != nil {
		return fmt.Errorf("failed to write provider binary: %w", err)
	}

	return nil
}

// Init runs terraform init
func (tr *TerraformRunner) Init() error {
	// Check if dev_overrides might be active by looking for .tofurc
	// When dev_overrides are set, init can fail because provider isn't in registry
	tofurcPath := filepath.Join(os.Getenv("HOME"), ".tofurc")
	if _, err := os.Stat(tofurcPath); err == nil {
		// .tofurc exists, check if it contains dev_overrides for elementum
		content, err := os.ReadFile(tofurcPath)
		if err == nil && containsString(string(content), "elementumltd/elementum") {
			// Dev overrides are likely active - skip init as it will fail
			// The provider binary will be found via dev_overrides
			// Remove any existing .terraform directory and lock file that might conflict
			_ = os.RemoveAll(filepath.Join(tr.WorkDir, ".terraform"))
			_ = os.Remove(filepath.Join(tr.WorkDir, ".terraform.lock.hcl"))
			fmt.Println("✓ Using provider dev_overrides [.tofurc] (skipping init)")
			return nil
		}
	} else {
		// .tofurc did not indicate dev_overrides — check .terraformrc as fallback
		terraformrcPath := filepath.Join(os.Getenv("HOME"), ".terraformrc")
		if _, err := os.Stat(terraformrcPath); err == nil {
			content, err := os.ReadFile(terraformrcPath)
			if err == nil && containsString(string(content), "elementumltd/elementum") {
				_ = os.RemoveAll(filepath.Join(tr.WorkDir, ".terraform"))
				_ = os.Remove(filepath.Join(tr.WorkDir, ".terraform.lock.hcl"))
				fmt.Println("✓ Using provider dev_overrides [.terraformrc] (skipping init)")
				return nil
			}
		}
	}

	// No dev_overrides, run init normally
	cmd := exec.Command("tofu", "init", "-no-color", "-input=false")
	cmd.Dir = tr.WorkDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		// Try terraform as fallback
		cmd = exec.Command("terraform", "init", "-no-color", "-input=false")
		cmd.Dir = tr.WorkDir
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		if err := cmd.Run(); err != nil {
			return fmt.Errorf("terraform/tofu init failed: %w", err)
		}
	}

	return nil
}

// containsString checks if s contains substr (simple string check)
func containsString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// GenerateConfig runs terraform plan with -generate-config-out
// Returns the plan output for further processing
func (tr *TerraformRunner) GenerateConfig(outputFile string) (string, error) {
	// Remove the output file if it exists (terraform refuses to overwrite)
	outputPath := filepath.Join(tr.WorkDir, outputFile)
	if _, err := os.Stat(outputPath); err == nil {
		_ = os.Remove(outputPath)
	}

	// Try tofu first, fall back to terraform
	cmd := exec.Command("tofu", "plan", "-no-color", fmt.Sprintf("-generate-config-out=%s", outputFile))
	cmd.Dir = tr.WorkDir

	// Capture both stdout and stderr
	var stdout []byte

	cmd.Env = os.Environ()

	// Run and capture output
	output, err := cmd.CombinedOutput()
	stdout = output

	// Only show plan output at trace log level
	showOutput := logger.GetCurrentLevel() == logger.LevelTrace

	// Check if file was generated (tofu plan returns exit code 2 for "changes planned" which is success for our purposes)
	if _, statErr := os.Stat(outputPath); statErr == nil {
		// File was generated, consider it a success
		if showOutput {
			_, _ = os.Stdout.Write(stdout)
		}
		return string(stdout), nil
	}

	// File wasn't generated, try terraform as fallback
	if err != nil {
		cmd = exec.Command("terraform", "plan", "-no-color", fmt.Sprintf("-generate-config-out=%s", outputFile))
		cmd.Dir = tr.WorkDir
		cmd.Env = os.Environ()

		output, err = cmd.CombinedOutput()
		stdout = output

		// Check if file was generated
		if _, statErr := os.Stat(outputPath); statErr == nil {
			if showOutput {
				_, _ = os.Stdout.Write(stdout)
			}
			return string(stdout), nil
		}

		if err != nil {
			_, _ = os.Stdout.Write(stdout)
			return string(stdout), fmt.Errorf("terraform/tofu plan failed: %w", err)
		}
	}

	// Print the output only at trace level
	if showOutput {
		_, _ = os.Stdout.Write(stdout)
	}

	return string(stdout), nil
}

// GetGeneratedFile reads the generated configuration file
func (tr *TerraformRunner) GetGeneratedFile(filename string) (string, error) {
	content, err := os.ReadFile(filepath.Join(tr.WorkDir, filename))
	if err != nil {
		return "", fmt.Errorf("failed to read generated file: %w", err)
	}
	return string(content), nil
}

// CheckTerraformInstalled checks if terraform or tofu is installed and available
func CheckTerraformInstalled() error {
	// Check for tofu first
	if cmd := exec.Command("tofu", "version"); cmd.Run() == nil {
		return nil
	}

	// Fall back to terraform
	if cmd := exec.Command("terraform", "version"); cmd.Run() == nil {
		return nil
	}

	return fmt.Errorf("terraform/tofu not found in PATH - please install OpenTofu (https://opentofu.org) or Terraform (https://www.terraform.io/downloads)")
}
