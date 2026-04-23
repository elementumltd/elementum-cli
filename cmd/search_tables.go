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

	"github.com/elementumltd/elementum-cli/auth"
	"github.com/elementumltd/elementum-cli/internal/client"
	"github.com/elementumltd/elementum-cli/ui"
	"github.com/spf13/cobra"
)

var searchTablesCmd = &cobra.Command{
	Use:   "search-tables",
	Short: "Manage AI search tables",
	Long:  "Commands for querying and managing AI search tables (Cortex search).",
}

var searchTablesQueryCmd = &cobra.Command{
	Use:   "query <table-id> <query>",
	Short: "Query an AI search table",
	Long: `Execute a semantic search query against an AI search table.

The table can be either an aspect (app/element) search table or a table search table.
By default, both types are tried automatically. Use --type to specify explicitly.

Use --filter to apply additional filters to the search results. The filter is a JSON
object with type, field, and value properties.

Examples:
  ei search-tables query 94828968-a51a-4c95-923b-e08f7f4b22a8 "search text"
  ei search-tables query 94828968-a51a-4c95-923b-e08f7f4b22a8 "search text" --limit 5
  ei search-tables query 94828968-a51a-4c95-923b-e08f7f4b22a8 "search text" --type table
  ei search-tables query 94828968-a51a-4c95-923b-e08f7f4b22a8 "search text" --json

Filter examples:
  # Text equals
  --filter '{"type":"EQUALS","field":"<field-id>","value":{"type":"TEXT","value":"Active"}}'

  # Boolean equals
  --filter '{"type":"EQUALS","field":"<field-id>","value":{"type":"BOOLEAN","value":true}}'

  # AND filter with multiple conditions
  --filter '{"type":"AND","children":[{"type":"EQUALS","field":"<id1>","value":{"type":"TEXT","value":"A"}},{"type":"EQUALS","field":"<id2>","value":{"type":"TEXT","value":"B"}}]}'`,
	Args: cobra.ExactArgs(2),
	RunE: runSearchTablesQuery,
}

var searchTablesRefreshCmd = &cobra.Command{
	Use:   "refresh <table-id>",
	Short: "Refresh an AI search table (incremental re-index)",
	Long: `Trigger an incremental refresh of an AI search table's Cortex Search Service.

This updates vector embeddings for records that have changed since the last refresh.
Use this for routine index maintenance.

Examples:
  ei search-tables refresh 950d9901-2cf7-4ad0-94cb-c9ab01dafcf1
  ei search-tables refresh 950d9901-2cf7-4ad0-94cb-c9ab01dafcf1 --json`,
	Args: cobra.ExactArgs(1),
	RunE: runSearchTablesRefresh,
}

var searchTablesRebuildCmd = &cobra.Command{
	Use:   "rebuild <table-id>",
	Short: "Rebuild an AI search table (full re-index)",
	Long: `Trigger a full rebuild of an AI search table's Cortex Search Service.

This performs CREATE OR REPLACE, dropping and recreating the entire service.
Use sparingly - only when incremental refresh is not sufficient.

WARNING: This temporarily makes the search table unavailable.

Examples:
  ei search-tables rebuild 950d9901-2cf7-4ad0-94cb-c9ab01dafcf1
  ei search-tables rebuild 950d9901-2cf7-4ad0-94cb-c9ab01dafcf1 --json`,
	Args: cobra.ExactArgs(1),
	RunE: runSearchTablesRebuild,
}

func init() {
	searchTablesCmd.AddCommand(searchTablesQueryCmd)
	searchTablesCmd.AddCommand(searchTablesRefreshCmd)
	searchTablesCmd.AddCommand(searchTablesRebuildCmd)
	searchTablesQueryCmd.Flags().Int("limit", 10, "Maximum number of results to return")
	searchTablesQueryCmd.Flags().String("type", "", "Search table type: aspect or table (auto-detects if not specified)")
	searchTablesQueryCmd.Flags().String("filter", "", `JSON filter (e.g., '{"type":"EQUALS","field":"<field-id>","value":{"type":"TEXT","value":"Active"}}')`)
}

// GetSearchTablesCmd returns the search-tables command for registration
func GetSearchTablesCmd() *cobra.Command {
	return searchTablesCmd
}

// searchRecord is a normalized record from either aspect or table search
type searchRecord struct {
	Values []searchValue `json:"values"`
}

type searchValue struct {
	Field string          `json:"field"`
	Value json.RawMessage `json:"value"`
}

func runSearchTablesQuery(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	c, err := auth.GetClientFromCmd(cmd)
	if err != nil {
		return err
	}

	tableID := args[0]
	queryText := args[1]
	limit, _ := cmd.Flags().GetInt("limit")
	searchType, _ := cmd.Flags().GetString("type")
	filterJSON, _ := cmd.Flags().GetString("filter")

	// Validate filter JSON if provided
	if filterJSON != "" {
		var filterObj map[string]any
		if err := json.Unmarshal([]byte(filterJSON), &filterObj); err != nil {
			return fmt.Errorf("invalid --filter JSON: %w", err)
		}
	}

	if !isJSONOutput(cmd) {
		fmt.Println(ui.InfoStyle.Render(fmt.Sprintf("Querying search table %s...", tableID)))
	}

	records, err := executeSearchQuery(ctx, c, tableID, queryText, limit, searchType, filterJSON)
	if err != nil {
		return err
	}

	// JSON output
	if isJSONOutput(cmd) {
		return outputJSON(map[string]any{
			"query":        queryText,
			"table_id":     tableID,
			"record_count": len(records),
			"records":      records,
		})
	}

	if len(records) == 0 {
		fmt.Println()
		fmt.Println(ui.WarningStyle.Render("No results found."))
		fmt.Println()
		return nil
	}

	// Display each record as a table
	fmt.Println()
	fmt.Println(ui.TitleStyle.Render(fmt.Sprintf("Search Results (%d records)", len(records))))

	for i, rec := range records {
		fmt.Println()
		fmt.Println(ui.SubtitleStyle.Render(fmt.Sprintf("Record %d", i+1)))
		fmt.Println()

		table := ui.NewTable([]string{"FIELD", "VALUE"})
		for _, v := range rec.Values {
			table.AddRow(v.Field, formatSearchValue(v.Value))
		}
		fmt.Println(table.Render())
	}

	fmt.Println()
	return nil
}

func executeSearchQuery(ctx context.Context, c *client.Client, tableID, queryText string, limit int, searchType, filterJSON string) ([]searchRecord, error) {
	switch searchType {
	case "aspect":
		return queryAspectSearchTable(ctx, c, tableID, queryText, limit, filterJSON)
	case "table":
		return queryTableSearchTable(ctx, c, tableID, queryText, limit, filterJSON)
	case "":
		// Auto-detect: try aspect first, fall back to table
		records, aspectErr := queryAspectSearchTable(ctx, c, tableID, queryText, limit, filterJSON)
		if aspectErr == nil {
			return records, nil
		}
		records, tableErr := queryTableSearchTable(ctx, c, tableID, queryText, limit, filterJSON)
		if tableErr == nil {
			return records, nil
		}
		return nil, fmt.Errorf("failed to query search table (tried both types):\n  aspect: %v\n  table: %v", aspectErr, tableErr)
	default:
		return nil, fmt.Errorf("invalid --type %q: must be \"aspect\" or \"table\"", searchType)
	}
}

func queryAspectSearchTable(ctx context.Context, c *client.Client, tableID, queryText string, limit int, filterJSON string) ([]searchRecord, error) {
	input := client.AspectSearchTableQueryInput{
		Query: queryText,
		Limit: &limit,
	}
	if filterJSON != "" {
		raw := json.RawMessage(filterJSON)
		input.Filter = &raw
	}
	resp, err := client.AspectSearchTableQuery(ctx, c.Genqlient(), tableID, input)
	if err != nil {
		return nil, err
	}

	var records []searchRecord
	for _, rec := range resp.AspectSearchTableQuery.Records {
		var values []searchValue
		for _, v := range rec.Values {
			val := json.RawMessage("null")
			if v.Value != nil {
				val = *v.Value
			}
			values = append(values, searchValue{Field: v.Field, Value: val})
		}
		records = append(records, searchRecord{Values: values})
	}
	return records, nil
}

func queryTableSearchTable(ctx context.Context, c *client.Client, tableID, queryText string, limit int, filterJSON string) ([]searchRecord, error) {
	input := client.TableSearchTableQueryInput{
		Query: queryText,
		Limit: &limit,
	}
	if filterJSON != "" {
		raw := json.RawMessage(filterJSON)
		input.Filter = &raw
	}
	resp, err := client.TableSearchTableQuery(ctx, c.Genqlient(), tableID, input)
	if err != nil {
		return nil, err
	}

	var records []searchRecord
	for _, rec := range resp.TableSearchTableQuery.Records {
		var values []searchValue
		for _, v := range rec.Values {
			val := json.RawMessage("null")
			if v.Value != nil {
				val = *v.Value
			}
			values = append(values, searchValue{Field: v.Field, Value: val})
		}
		records = append(records, searchRecord{Values: values})
	}
	return records, nil
}

// formatSearchValue converts a json.RawMessage to a human-readable string
func formatSearchValue(raw json.RawMessage) string {
	if len(raw) == 0 || string(raw) == "null" {
		return ""
	}
	// Try string first (most common)
	var s string
	if json.Unmarshal(raw, &s) == nil {
		return s
	}
	// Fall back to compact JSON representation
	return string(raw)
}

func runSearchTablesRefresh(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	c, err := auth.GetClientFromCmd(cmd)
	if err != nil {
		return err
	}

	tableID := args[0]

	if !isJSONOutput(cmd) {
		fmt.Println(ui.InfoStyle.Render(fmt.Sprintf("Refreshing search table %s...", tableID)))
	}

	result, err := client.RefreshAspectSearchTable(ctx, c.Genqlient(), tableID)
	if err != nil {
		return fmt.Errorf("refresh failed: %w", err)
	}

	st := result.GetAspectSearchTableRefresh()

	if isJSONOutput(cmd) {
		return outputJSON(map[string]any{
			"id":            st.GetId(),
			"status":        st.GetStatus(),
			"error_message": st.GetErrorMessage(),
		})
	}

	fmt.Println()
	fmt.Println(ui.SuccessStyle.Render("Search table refreshed"))
	fmt.Println()
	fmt.Printf("  ID:     %s\n", st.GetId())
	fmt.Printf("  Status: %s\n", st.GetStatus())
	if st.GetErrorMessage() != nil && *st.GetErrorMessage() != "" {
		fmt.Printf("  Error:  %s\n", *st.GetErrorMessage())
	}
	fmt.Println()

	return nil
}

func runSearchTablesRebuild(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	c, err := auth.GetClientFromCmd(cmd)
	if err != nil {
		return err
	}

	tableID := args[0]

	if !isJSONOutput(cmd) {
		fmt.Println(ui.InfoStyle.Render(fmt.Sprintf("Rebuilding search table %s...", tableID)))
	}

	result, err := client.RebuildAspectSearchTable(ctx, c.Genqlient(), tableID)
	if err != nil {
		return fmt.Errorf("rebuild failed: %w", err)
	}

	st := result.GetAspectSearchTableRebuild()

	if isJSONOutput(cmd) {
		return outputJSON(map[string]any{
			"id":            st.GetId(),
			"status":        st.GetStatus(),
			"error_message": st.GetErrorMessage(),
		})
	}

	fmt.Println()
	fmt.Println(ui.SuccessStyle.Render("Search table rebuild initiated"))
	fmt.Println()
	fmt.Printf("  ID:     %s\n", st.GetId())
	fmt.Printf("  Status: %s\n", st.GetStatus())
	if st.GetErrorMessage() != nil && *st.GetErrorMessage() != "" {
		fmt.Printf("  Error:  %s\n", *st.GetErrorMessage())
	}
	fmt.Println()

	return nil
}
