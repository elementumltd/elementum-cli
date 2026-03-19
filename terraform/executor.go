// Copyright (c) ElementumAI, Inc.
// SPDX-License-Identifier: MPL-2.0

package terraform

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"

	"github.com/elementumltd/elementum-cli/auth"
	"github.com/elementumltd/elementum-cli/logger"
	"github.com/spf13/cobra"
)

// Execute runs a terraform/tofu command with authentication credentials injected
// as environment variables. It streams output in real-time with enhanced formatting.
func Execute(cmd *cobra.Command, binary string, args []string) error {
	// Load config and credentials
	config, _ := auth.LoadConfig()
	creds, err := auth.GetCredentials(cmd, config)
	if err != nil {
		return fmt.Errorf("failed to load credentials: %w", err)
	}

	// Create the command
	tfCmd := exec.Command(binary, args...)

	// Set working directory to current directory
	tfCmd.Dir = "."

	// Copy current environment and add Elementum credentials
	tfCmd.Env = os.Environ()
	tfCmd.Env = append(tfCmd.Env,
		fmt.Sprintf("ELEMENTUM_ORGANIZATION=%s", creds.Organization),
		fmt.Sprintf("ELEMENTUM_CLIENT_ID=%s", creds.ClientID),
		fmt.Sprintf("ELEMENTUM_CLIENT_SECRET=%s", creds.ClientSecret),
		fmt.Sprintf("ELEMENTUM_ENVIRONMENT=%s", creds.Environment),
		fmt.Sprintf("ELEMENTUM_INSTANCE=%s", creds.Instance),
	)

	// Add custom URLs if set (for custom instances)
	if creds.CustomGraphQLURL != "" {
		tfCmd.Env = append(tfCmd.Env, fmt.Sprintf("ELEMENTUM_CUSTOM_GRAPHQL_URL=%s", creds.CustomGraphQLURL))
	}
	if creds.CustomAPIURL != "" {
		tfCmd.Env = append(tfCmd.Env, fmt.Sprintf("ELEMENTUM_CUSTOM_API_URL=%s", creds.CustomAPIURL))
	}

	// Add TF_LOG based on unified log level
	if tfLogLevel := getTFLogLevel(); tfLogLevel != "" {
		tfCmd.Env = append(tfCmd.Env, fmt.Sprintf("TF_LOG=%s", tfLogLevel))
	}

	// Set up pipes for stdout and stderr
	stdout, err := tfCmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderr, err := tfCmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	// Start the command
	if err := tfCmd.Start(); err != nil {
		return fmt.Errorf("failed to start %s: %w", binary, err)
	}

	// Stream output in real-time with enhanced formatting
	done := make(chan error, 2)

	go func() {
		done <- streamOutput(stdout, false)
	}()

	go func() {
		done <- streamOutput(stderr, true)
	}()

	// Wait for both streams to finish
	<-done
	<-done

	// Wait for command to complete
	if err := tfCmd.Wait(); err != nil {
		return fmt.Errorf("%s command failed: %w", binary, err)
	}

	return nil
}

// streamOutput reads from a pipe and writes to stdout with optional enhancement
func streamOutput(reader io.Reader, isStderr bool) error {
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := scanner.Text()

		// Apply enhancement
		enhancedLine := EnhanceLine(line)

		// Write to appropriate output stream
		if isStderr {
			fmt.Fprintln(os.Stderr, enhancedLine)
		} else {
			fmt.Println(enhancedLine)
		}
	}

	return scanner.Err()
}

// getTFLogLevel determines the TF_LOG value based on the unified log level.
func getTFLogLevel() string {
	return logger.GetTFLogLevel(logger.GetCurrentLevel())
}
