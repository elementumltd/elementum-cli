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
	"encoding/json"
	"os"
	"regexp"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/term"
)

// uuidRegex matches standard UUID format
var uuidRegex = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

// isInteractive returns true if running in an interactive terminal
func isInteractive() bool {
	return term.IsTerminal(int(os.Stdin.Fd()))
}

// hasFlag checks if a flag is present in args
func hasFlag(args []string, flags ...string) bool {
	for _, arg := range args {
		for _, flag := range flags {
			if arg == flag {
				return true
			}
		}
	}
	return false
}

// hasPlanFile checks if args contain a plan file (non-flag arg that exists as a file)
func hasPlanFile(args []string) bool {
	for _, arg := range args {
		if !strings.HasPrefix(arg, "-") {
			if info, err := os.Stat(arg); err == nil && !info.IsDir() {
				return true
			}
		}
	}
	return false
}

// isJSONOutput checks if the --json flag is set
func isJSONOutput(cmd *cobra.Command) bool {
	jsonFlag, _ := cmd.Flags().GetBool("json")
	return jsonFlag
}

// outputJSON writes data as JSON to stdout
func outputJSON(data any) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(data)
}

// isUUID checks if a string is a valid UUID
func isUUID(s string) bool {
	return uuidRegex.MatchString(s)
}

// EIFlags holds the extracted ei-specific flags from args.
// These flags need special handling for commands with DisableFlagParsing: true.
type EIFlags struct {
	Profile  string
	LogLevel string
	Debug    bool
	Quiet    bool
}

// extractEIFlags extracts ei-specific flags from args and returns filtered args.
// This is needed for commands with DisableFlagParsing: true that still need
// to support global flags like --profile, --log-level, --debug, --quiet.
func extractEIFlags(args []string) (flags EIFlags, filteredArgs []string) {
	filteredArgs = make([]string, 0, len(args))

	for i := 0; i < len(args); i++ {
		arg := args[i]

		// Handle --profile
		if strings.HasPrefix(arg, "--profile=") {
			flags.Profile = strings.TrimPrefix(arg, "--profile=")
			continue
		}
		if arg == "--profile" {
			if i+1 < len(args) {
				flags.Profile = args[i+1]
				i++
			}
			continue
		}

		// Handle --log-level
		if strings.HasPrefix(arg, "--log-level=") {
			flags.LogLevel = strings.TrimPrefix(arg, "--log-level=")
			continue
		}
		if arg == "--log-level" {
			if i+1 < len(args) {
				flags.LogLevel = args[i+1]
				i++
			}
			continue
		}

		// Handle --debug (boolean flag)
		if arg == "--debug" {
			flags.Debug = true
			continue
		}

		// Handle --quiet or -q (boolean flag)
		if arg == "--quiet" || arg == "-q" {
			flags.Quiet = true
			continue
		}

		filteredArgs = append(filteredArgs, arg)
	}

	return flags, filteredArgs
}

// applyEIFlags sets the extracted flags on the root command's persistent flags
// and initializes the logger appropriately.
func applyEIFlags(cmd *cobra.Command, flags EIFlags) {
	root := cmd.Root()
	if root == nil {
		return
	}

	if flags.Profile != "" {
		_ = root.PersistentFlags().Set("profile", flags.Profile)
	}
	if flags.LogLevel != "" {
		_ = root.PersistentFlags().Set("log-level", flags.LogLevel)
	}
	if flags.Debug {
		_ = root.PersistentFlags().Set("debug", "true")
	}
	if flags.Quiet {
		_ = root.PersistentFlags().Set("quiet", "true")
	}
}
