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
	"errors"
	"fmt"
	"os"

	"github.com/charmbracelet/huh"
	"github.com/elementumltd/elementum-cli/auth"
	"github.com/spf13/cobra"
)

var authImportCmd = &cobra.Command{
	Use:   "import FILE",
	Short: "Import auth profiles from a passphrase-encrypted file",
	Long: `Import auth profiles from a file created by 'ei auth export'. You will be
prompted for the passphrase used during export.

By default you are prompted per conflicting profile. Use --force to overwrite
every conflict without asking, or --skip-existing to keep existing profiles.

The active profile is never changed by import. After importing, use
'ei auth switch <name>' if you want a different active profile.

Examples:
  ei auth import elementum-profiles.yaml
  ei auth import creds.yaml --force
  ei auth import creds.yaml --skip-existing`,
	Args: cobra.ExactArgs(1),
	RunE: runAuthImport,
}

func init() {
	authImportCmd.Flags().Bool("force", false, "Overwrite existing profiles without prompting")
	authImportCmd.Flags().Bool("skip-existing", false, "Skip existing profiles without prompting")
}

func runAuthImport(cmd *cobra.Command, args []string) error {
	inputPath := args[0]
	force, _ := cmd.Flags().GetBool("force")
	skipExisting, _ := cmd.Flags().GetBool("skip-existing")

	if force && skipExisting {
		return fmt.Errorf("--force and --skip-existing are mutually exclusive")
	}

	data, err := os.ReadFile(inputPath)
	if err != nil {
		return fmt.Errorf("failed to read %s: %w", inputPath, err)
	}

	passphrase, err := promptExistingPassphrase()
	if err != nil {
		return err
	}

	imported, err := auth.DecodePortable(data, passphrase)
	if err != nil {
		if errors.Is(err, auth.ErrPortableBadPassphrase) {
			return fmt.Errorf("incorrect passphrase or corrupted export file")
		}
		return fmt.Errorf("failed to decrypt export: %w", err)
	}
	if len(imported) == 0 {
		return fmt.Errorf("export file contains no profiles")
	}

	config, err := auth.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	var added, overwritten, skipped []string

	for _, p := range imported {
		if p.Name == "" {
			fmt.Println(infoStyle.Render("⚠ Skipping profile with empty name in export"))
			continue
		}

		_, exists := config.Profiles[p.Name]
		shouldWrite := true

		if exists {
			switch {
			case force:
				// overwrite silently
			case skipExisting:
				shouldWrite = false
			default:
				confirmed, err := confirmOverwrite(p.Name)
				if err != nil {
					return err
				}
				shouldWrite = confirmed
			}
		}

		if !shouldWrite {
			skipped = append(skipped, p.Name)
			continue
		}

		// Store the secret in this machine's keychain. StoreSecret returns a
		// `keyring:...` reference on success or the raw secret (with a warning
		// error) on keychain failure — matching the login flow, we tolerate
		// both but surface a warning to the user.
		secretRef, storeErr := auth.StoreSecret(p.Name, p.ClientSecret)
		if storeErr != nil {
			fmt.Println(infoStyle.Render(fmt.Sprintf("⚠ Profile %q: %v", p.Name, storeErr)))
		}

		profile := &auth.Profile{
			Organization:     p.Organization,
			Instance:         p.Instance,
			Environment:      p.Environment,
			CustomBaseURL:    p.CustomBaseURL,
			CustomGraphQLURL: p.CustomGraphQLURL,
			CustomAPIURL:     p.CustomAPIURL,
			ClientID:         p.ClientID,
			ClientSecret:     secretRef,
		}
		auth.SetProfile(config, p.Name, profile)

		if exists {
			overwritten = append(overwritten, p.Name)
		} else {
			added = append(added, p.Name)
		}
	}

	if err := auth.SaveConfig(config); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Println()
	fmt.Println(successStyle.Render(fmt.Sprintf("✓ Imported %d profile(s) from %s", len(added)+len(overwritten), inputPath)))
	if len(added) > 0 {
		fmt.Printf("  %s %v\n", labelStyle.Render("Added:"), added)
	}
	if len(overwritten) > 0 {
		fmt.Printf("  %s %v\n", labelStyle.Render("Overwritten:"), overwritten)
	}
	if len(skipped) > 0 {
		fmt.Printf("  %s %v\n", labelStyle.Render("Skipped:"), skipped)
	}
	fmt.Println()
	fmt.Println(infoStyle.Render(fmt.Sprintf("Active profile remains: %s", config.CurrentProfile)))
	fmt.Println(infoStyle.Render("  Run 'ei auth switch <name>' to change it."))
	fmt.Println()

	return nil
}

// promptExistingPassphrase asks for the passphrase that protects an export.
func promptExistingPassphrase() (string, error) {
	var pass string

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewNote().
				Title("Decrypt import").
				Description("Enter the passphrase used when the export file was created."),

			huh.NewInput().
				Title("Passphrase").
				EchoMode(huh.EchoModePassword).
				Value(&pass).
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
	return pass, nil
}

// confirmOverwrite asks the user whether to overwrite a colliding profile.
func confirmOverwrite(name string) (bool, error) {
	var confirmed bool
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title(fmt.Sprintf("Profile %q already exists. Overwrite?", name)).
				Description("Choosing 'No' will keep the existing profile and skip the one from the export.").
				Value(&confirmed),
		),
	)
	if err := form.Run(); err != nil {
		return false, fmt.Errorf("confirmation cancelled: %w", err)
	}
	return confirmed, nil
}
