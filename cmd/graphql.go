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
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/elementumltd/elementum-cli/auth"
	"github.com/elementumltd/elementum-cli/ui"
	"github.com/spf13/cobra"
)

var graphqlCmd = &cobra.Command{
	Use:   "graphql <query>",
	Short: "Execute a raw GraphQL query or mutation",
	Long: `Execute an arbitrary GraphQL query or mutation against the Elementum API.

The query can be provided as an argument, via stdin, or from a file using -f.

Examples:
  # Simple query as argument
  ei graphql 'query { organization { id name } }'

  # Query with variables
  ei graphql 'query($id: ID!) { organization { app(id: $id) { name } } }' -v '{"id": "abc-123"}'

  # Mutation
  ei graphql 'mutation { phoneServiceDelete(id: "abc-123") { id } }'

  # From file
  ei graphql -f query.graphql

  # From stdin
  cat query.graphql | ei graphql

  # Pretty print output
  ei graphql 'query { organization { id } }' --pretty`,
	Args: cobra.MaximumNArgs(1),
	RunE: runGraphQL,
}

func init() {
	graphqlCmd.Flags().StringP("file", "f", "", "Read query from file")
	graphqlCmd.Flags().StringP("variables", "v", "", "JSON variables for the query")
	graphqlCmd.Flags().Bool("pretty", false, "Pretty print JSON output")
}

// GetGraphQLCmd returns the graphql command for registration
func GetGraphQLCmd() *cobra.Command {
	return graphqlCmd
}

func runGraphQL(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Get authenticated client
	c, err := auth.GetClientFromCmd(cmd)
	if err != nil {
		return err
	}

	// Determine the query source
	var query string
	fileFlag, _ := cmd.Flags().GetString("file")

	if fileFlag != "" {
		// Read from file
		data, err := os.ReadFile(fileFlag)
		if err != nil {
			return fmt.Errorf("failed to read file %s: %w", fileFlag, err)
		}
		query = string(data)
	} else if len(args) > 0 {
		// Use argument
		query = args[0]
	} else {
		// Try reading from stdin
		stat, _ := os.Stdin.Stat()
		if (stat.Mode() & os.ModeCharDevice) == 0 {
			data, err := io.ReadAll(os.Stdin)
			if err != nil {
				return fmt.Errorf("failed to read from stdin: %w", err)
			}
			query = string(data)
		}
	}

	query = strings.TrimSpace(query)
	if query == "" {
		return fmt.Errorf("no query provided. Use an argument, -f flag, or pipe from stdin")
	}

	// Parse variables if provided
	varsFlag, _ := cmd.Flags().GetString("variables")
	var variables map[string]interface{}
	if varsFlag != "" {
		if err := json.Unmarshal([]byte(varsFlag), &variables); err != nil {
			return fmt.Errorf("failed to parse variables JSON: %w", err)
		}
	}

	// Build the request body
	requestBody := map[string]interface{}{
		"query": query,
	}
	if variables != nil {
		requestBody["variables"] = variables
	}

	// Execute the query using the client's Execute method
	var result map[string]interface{}
	err = c.ExecuteInto(ctx, query, variables, &result)
	if err != nil {
		return fmt.Errorf("GraphQL request failed: %w", err)
	}

	// Output the result
	pretty, _ := cmd.Flags().GetBool("pretty")
	var output []byte
	if pretty {
		output, err = json.MarshalIndent(result, "", "  ")
	} else {
		output, err = json.Marshal(result)
	}
	if err != nil {
		return fmt.Errorf("failed to marshal response: %w", err)
	}

	if !isJSONOutput(cmd) && !pretty {
		// If not explicitly JSON mode but also not pretty, show a hint
		fmt.Println(ui.MutedStyle.Render("Tip: Use --pretty for formatted output"))
		fmt.Println()
	}

	fmt.Println(string(output))
	return nil
}
