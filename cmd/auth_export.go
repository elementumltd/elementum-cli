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
	"path/filepath"
	"sort"

	"github.com/charmbracelet/huh"
	"github.com/elementumltd/elementum-cli/auth"
	"github.com/spf13/cobra"
)

const defaultExportFilename = "elementum-profiles.yaml"

var authExportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export auth profiles to a passphrase-encrypted file",
	Long: `Export auth profiles (including client secrets from the OS keychain) to a
passphrase-encrypted YAML file that can be imported on another machine with
'ei auth import'.

By default all profiles are exported. Use --profile to export a single one.

Examples:
  ei auth export                             # exports all profiles to ./elementum-profiles.yaml
  ei auth export -o ~/Desktop/creds.yaml     # choose an output path
  ei auth export --profile myorg-prod        # export a single profile
  ei auth export --force                     # overwrite an existing output file`,
	RunE: runAuthExport,
}

func init() {
	authExportCmd.Flags().String("profile", "", "Export a single profile by name (default: all profiles)")
	authExportCmd.Flags().StringP("output", "o", defaultExportFilename, "Output file path")
	authExportCmd.Flags().Bool("force", false, "Overwrite the output file if it exists")
}

func runAuthExport(cmd *cobra.Command, _ []string) error {
	profileName, _ := cmd.Flags().GetString("profile")
	outputPath, _ := cmd.Flags().GetString("output")
	force, _ := cmd.Flags().GetBool("force")

	config, err := auth.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}
	if len(config.Profiles) == 0 {
		return fmt.Errorf("no profiles configured - run 'ei auth login' first")
	}

	// Determine which profiles to export.
	names := make([]string, 0, len(config.Profiles))
	if profileName != "" {
		if _, ok := config.Profiles[profileName]; !ok {
			return fmt.Errorf("profile %q not found", profileName)
		}
		names = append(names, profileName)
	} else {
		for name := range config.Profiles {
			names = append(names, name)
		}
		sort.Strings(names)
	}

	// Resolve plaintext secrets from the keychain up front, so we fail fast
	// if something is wrong before bothering the user for a passphrase.
	portable := make([]auth.PortableProfile, 0, len(names))
	for _, name := range names {
		p := config.Profiles[name]
		secret, err := auth.GetSecret(name, p.ClientSecret)
		if err != nil {
			return fmt.Errorf("profile %q: failed to retrieve client secret from keychain: %w", name, err)
		}
		portable = append(portable, auth.PortableProfile{
			Name:             name,
			Organization:     p.Organization,
			Instance:         p.Instance,
			Environment:      p.Environment,
			CustomBaseURL:    p.CustomBaseURL,
			CustomGraphQLURL: p.CustomGraphQLURL,
			CustomAPIURL:     p.CustomAPIURL,
			ClientID:         p.ClientID,
			ClientSecret:     secret,
		})
	}

	// Refuse to clobber unless --force.
	if _, err := os.Stat(outputPath); err == nil && !force {
		return fmt.Errorf("output file %q already exists (use --force to overwrite)", outputPath)
	} else if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to stat output path: %w", err)
	}

	// Prompt for passphrase (twice, with confirmation).
	passphrase, err := promptNewPassphrase()
	if err != nil {
		return err
	}

	data, err := auth.EncodePortable(portable, passphrase)
	if err != nil {
		return fmt.Errorf("failed to encrypt export: %w", err)
	}

	// Write atomically via a temp file in the same directory, so we don't
	// end up with a half-written file if something goes sideways.
	absPath, err := filepath.Abs(outputPath)
	if err != nil {
		return fmt.Errorf("failed to resolve output path: %w", err)
	}
	dir := filepath.Dir(absPath)
	tmp, err := os.CreateTemp(dir, ".ei-export-*.tmp")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpPath := tmp.Name()
	// Ensure the temp file is cleaned up if we don't rename it.
	defer func() { _ = os.Remove(tmpPath) }()

	if err := tmp.Chmod(0o600); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("failed to chmod temp file: %w", err)
	}
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("failed to write export: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("failed to close temp file: %w", err)
	}
	if err := os.Rename(tmpPath, absPath); err != nil {
		return fmt.Errorf("failed to move export into place: %w", err)
	}

	fmt.Println()
	fmt.Println(successStyle.Render(fmt.Sprintf("✓ Exported %d profile(s) to %s", len(portable), absPath)))
	fmt.Println(infoStyle.Render("  File is encrypted with your passphrase. Keep both safe — anyone with"))
	fmt.Println(infoStyle.Render("  the file AND the passphrase can recover your client secrets."))
	fmt.Println(infoStyle.Render("  Do not commit this file to source control."))
	fmt.Println()
	return nil
}

// promptNewPassphrase asks for a passphrase twice and verifies they match.
func promptNewPassphrase() (string, error) {
	var pass1, pass2 string

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewNote().
				Title("Encrypt export").
				Description("Choose a passphrase. You'll need this exact passphrase to import the file on another machine. It is not stored or recoverable."),

			huh.NewInput().
				Title("Passphrase").
				EchoMode(huh.EchoModePassword).
				Value(&pass1).
				Validate(func(s string) error {
					if s == "" {
						return fmt.Errorf("passphrase is required")
					}
					return nil
				}),

			huh.NewInput().
				Title("Confirm passphrase").
				EchoMode(huh.EchoModePassword).
				Value(&pass2).
				Validate(func(s string) error {
					if s == "" {
						return fmt.Errorf("passphrase is required")
					}
					return nil
				}),
		),
	)
	if err := form.Run(); err != nil {
		return "", fmt.Errorf("passphrase prompt cancelled: %w", err)
	}
	if pass1 != pass2 {
		return "", fmt.Errorf("passphrases do not match")
	}
	return pass1, nil
}
