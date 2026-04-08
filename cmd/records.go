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
	"fmt"
	"strings"

	"github.com/elementumltd/elementum-cli/auth"
	"github.com/elementumltd/elementum-cli/discovery"
	"github.com/elementumltd/elementum-cli/logger"
	"github.com/elementumltd/elementum-cli/ui"
	"github.com/elementumltd/elementum-cli/internal/client"
	"github.com/spf13/cobra"
)

var recordsCmd = &cobra.Command{
	Use:   "records",
	Short: "Manage records",
	Long:  "Commands for listing, creating, updating, and deleting records in Elementum apps.",
}

var recordsListCmd = &cobra.Command{
	Use:   "list <app-namespace-or-url>",
	Short: "List records from an app, element, or task",
	Long: `Fetch and display records from an Elementum App, Element, or Task.

You can specify the target using:
  - A namespace (e.g., "support-tickets", "my-app")
  - A full URL (e.g., "https://org.elementum.io/app/support-tickets/list")

Examples:
  ei records list support-tickets
  ei records list support-tickets --limit 50
  ei records list support-tickets --all
  ei records list support-tickets --columns "Title,Status,Priority"
  ei records list support-tickets --json
  ei records list "https://org.elementum.io/app/support-tickets/list"`,
	Args: cobra.ExactArgs(1),
	RunE: runRecordsList,
}

var recordsDeleteCmd = &cobra.Command{
	Use:   "delete <namespace> <handle>",
	Short: "Delete records from an app, element, or task",
	Long: `Delete one or more records from an Elementum App, Element, or Task.

You can delete records by:
  - Namespace and handle: ei records delete <namespace> <handle>
  - Full record ID: ei records delete --id <record-id>
  - All records in an aspect: ei records delete <namespace> --all (requires --force)

Examples:
  ei records delete support-tickets TKT-123
  ei records delete --id "abc123:TKT-123"
  ei records delete support-tickets --all --force`,
	Args: cobra.RangeArgs(0, 2),
	RunE: runRecordsDelete,
}

func init() {
	// List subcommand flags
	recordsListCmd.Flags().Int("limit", 25, "Maximum number of records to fetch per request")
	recordsListCmd.Flags().String("after", "", "Pagination cursor (fetch records after this cursor)")
	recordsListCmd.Flags().Bool("all", false, "Fetch all records (auto-paginate)")
	recordsListCmd.Flags().StringSlice("columns", nil, "Columns to display (comma-separated field names)")

	// Delete subcommand flags
	recordsDeleteCmd.Flags().String("id", "", "Full record ID to delete (format: aspectID:handle)")
	recordsDeleteCmd.Flags().Bool("all", false, "Delete all records in the aspect (dangerous!)")
	recordsDeleteCmd.Flags().Bool("force", false, "Skip confirmation prompt for bulk delete")
	recordsDeleteCmd.Flags().Bool("dry-run", false, "Show what would be deleted without actually deleting")

	recordsCmd.AddCommand(recordsListCmd)
	recordsCmd.AddCommand(recordsDeleteCmd)
	recordsCmd.AddCommand(recordsCreateCmd)
	recordsCmd.AddCommand(recordsUpdateCmd)
	recordsCmd.AddCommand(recordsGetCmd)
	recordsCmd.AddCommand(recordsImportCmd)
	recordsCmd.AddCommand(recordsExportCmd)
	recordsCmd.AddCommand(recordsGenerateCmd)
}

// GetRecordsCmd returns the records command for registration
func GetRecordsCmd() *cobra.Command {
	return recordsCmd
}

func runRecordsList(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Get authenticated client
	client, err := auth.GetClientFromCmd(cmd)
	if err != nil {
		return err
	}

	// Parse input (could be namespace or URL)
	input := args[0]
	var namespace string

	// Check if it looks like a URL
	if strings.Contains(input, "/") {
		resourceInfo, err := discovery.ParseResourceFromURL(input)
		if err != nil {
			return fmt.Errorf("failed to parse URL: %w", err)
		}
		namespace = resourceInfo.Identifier
	} else {
		namespace = input
	}

	// Get flags
	limit, _ := cmd.Flags().GetInt("limit")
	after, _ := cmd.Flags().GetString("after")
	fetchAll, _ := cmd.Flags().GetBool("all")
	columns, _ := cmd.Flags().GetStringSlice("columns")

	opts := discovery.RecordListOptions{
		Limit:   limit,
		After:   after,
		All:     fetchAll,
		Columns: columns,
	}

	// Show loading message (skip for JSON output)
	if !isJSONOutput(cmd) {
		fmt.Println(ui.InfoStyle.Render("⠿ Loading records..."))
	}

	// Fetch records by namespace
	var result *discovery.RecordListResult

	// First resolve namespace to aspect ID
	aspectID, _, err := resolveAspectByNamespace(ctx, client, namespace)
	if err != nil {
		return fmt.Errorf("failed to resolve namespace %q: %w", namespace, err)
	}

	if fetchAll {
		result, err = discovery.ListAllRecords(ctx, client, aspectID, opts)
	} else {
		result, err = discovery.ListRecords(ctx, client, aspectID, opts)
	}
	if err != nil {
		return fmt.Errorf("failed to fetch records: %w", err)
	}

	// Fetch picklist values and resolve them in record data
	if len(result.Records) > 0 {
		picklistValues, err := discovery.FetchPicklistValues(ctx, client, aspectID)
		if err != nil {
			// Log warning but continue - picklist resolution is optional
			logger.Debug("failed to fetch picklist values", "error", err)
		} else if len(picklistValues) > 0 {
			discovery.ResolvePicklistValues(result.Records, picklistValues)
		}
	}

	// JSON output
	if isJSONOutput(cmd) {
		return outputJSON(result)
	}

	// Display results
	if len(result.Records) == 0 {
		fmt.Println()
		fmt.Println(ui.WarningStyle.Render("No records found."))
		fmt.Println()
		return nil
	}

	// Determine columns to display
	displayColumns := columns
	if len(displayColumns) == 0 {
		displayColumns = []string{"Handle", "Title", "Status", "Created At"}
	}

	// Build table header
	headerCols := make([]string, len(displayColumns))
	for i, col := range displayColumns {
		headerCols[i] = strings.ToUpper(col)
	}
	table := ui.NewTable(headerCols)

	// Add rows
	for _, record := range result.Records {
		row := make([]string, len(displayColumns))
		for i, col := range displayColumns {
			row[i] = getRecordColumnValue(record, col)
		}
		table.AddRow(row...)
	}

	// Display table with title
	fmt.Println()
	title := fmt.Sprintf("📋 %s (%s)", result.AspectName, result.AspectType)
	fmt.Println(ui.TitleStyle.Render(title))
	fmt.Println()
	fmt.Println(table.Render())
	fmt.Println()

	// Show pagination info
	if result.HasNextPage {
		fmt.Println(ui.MutedStyle.Render(fmt.Sprintf("Showing %d of %d records (more available, use --all to fetch all or --after %q to continue)",
			len(result.Records), result.Total, result.EndCursor)))
	} else {
		fmt.Println(ui.MutedStyle.Render(fmt.Sprintf("Total: %d records", result.Total)))
	}
	fmt.Println()

	return nil
}

// getRecordColumnValue extracts a display value for a column from a record
func getRecordColumnValue(record discovery.Record, column string) string {
	// Handle built-in columns
	switch strings.ToLower(column) {
	case "id":
		return record.ID
	case "handle":
		return record.Handle
	case "title":
		return record.Title
	case "status":
		return record.Status
	case "url":
		return record.URL
	case "created at", "createdat", "created_at":
		return formatRecordTimestamp(record.CreatedAt)
	case "updated at", "updatedat", "updated_at":
		return formatRecordTimestamp(record.UpdatedAt)
	case "created by", "createdby", "created_by":
		return record.CreatedBy
	default:
		// Try to get from data map (custom fields)
		return discovery.GetFieldValue(record, column)
	}
}

// formatRecordTimestamp formats an ISO timestamp for display
func formatRecordTimestamp(ts string) string {
	if ts == "" {
		return ""
	}
	// Simple formatting - just take the date portion
	if len(ts) >= 10 {
		return ts[:10]
	}
	return ts
}

// resolveAspectByNamespace is a helper that searches for an aspect by namespace
func resolveAspectByNamespace(ctx context.Context, c *client.Client, namespace string) (string, string, error) {
	// List objects and find matching namespace
	objects, err := discovery.ListObjects(ctx, c)
	if err != nil {
		return "", "", err
	}

	// Find exact match (case-insensitive)
	for _, obj := range objects {
		if strings.EqualFold(obj.Namespace, namespace) {
			return obj.ID, obj.Type, nil
		}
	}

	return "", "", fmt.Errorf("aspect not found with namespace: %s", namespace)
}

func runRecordsDelete(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Get authenticated client
	c, err := auth.GetClientFromCmd(cmd)
	if err != nil {
		return err
	}

	// Get flags
	recordID, _ := cmd.Flags().GetString("id")
	deleteAll, _ := cmd.Flags().GetBool("all")
	force, _ := cmd.Flags().GetBool("force")
	dryRun, _ := cmd.Flags().GetBool("dry-run")

	// Mode 1: Delete by full record ID
	if recordID != "" {
		if dryRun {
			fmt.Printf("Would delete record: %s\n", recordID)
			return nil
		}
		if err := discovery.DeleteRecord(ctx, c, recordID); err != nil {
			return err
		}
		fmt.Println(ui.SuccessStyle.Render(fmt.Sprintf("✓ Deleted record: %s", recordID)))
		return nil
	}

	// Validate args for namespace-based operations
	if len(args) < 1 {
		return fmt.Errorf("namespace is required (or use --id to delete by record ID)")
	}

	namespace := args[0]

	// Mode 2: Delete all records in namespace
	if deleteAll {
		if !force {
			return fmt.Errorf("deleting all records requires --force flag (this is a destructive operation)")
		}

		// Resolve namespace to aspect ID
		aspectID, _, err := resolveAspectByNamespace(ctx, c, namespace)
		if err != nil {
			return fmt.Errorf("failed to resolve namespace %q: %w", namespace, err)
		}

		if dryRun {
			// For dry-run, we need to fetch records to show what would be deleted
			result, err := discovery.ListAllRecords(ctx, c, aspectID, discovery.RecordListOptions{Limit: 100})
			if err != nil {
				return fmt.Errorf("failed to fetch records: %w", err)
			}

			if len(result.Records) == 0 {
				fmt.Println(ui.WarningStyle.Render("No records found to delete."))
				return nil
			}

			fmt.Printf("Would delete %d records from %s:\n", len(result.Records), namespace)
			for _, r := range result.Records {
				fmt.Printf("  - %s (%s)\n", r.Handle, r.Title)
			}
			return nil
		}

		// Use bulk delete API
		fmt.Printf("Deleting all records from %s...\n", namespace)
		deleted, err := discovery.BulkDeleteAllRecords(ctx, c, aspectID)
		if err != nil {
			return err
		}

		if deleted == 0 {
			fmt.Println(ui.WarningStyle.Render("No records found to delete."))
			return nil
		}

		fmt.Println(ui.SuccessStyle.Render(fmt.Sprintf("✓ Deleted %d records from %s", deleted, namespace)))
		return nil
	}

	// Mode 3: Delete single record by namespace and handle
	if len(args) < 2 {
		return fmt.Errorf("handle is required (usage: ei records delete <namespace> <handle>)")
	}

	handle := args[1]

	// Resolve namespace to aspect ID
	aspectID, _, err := resolveAspectByNamespace(ctx, c, namespace)
	if err != nil {
		return fmt.Errorf("failed to resolve namespace %q: %w", namespace, err)
	}

	// Build the full record ID
	fullRecordID := aspectID + ":" + handle

	if dryRun {
		fmt.Printf("Would delete record: %s (namespace: %s, handle: %s)\n", fullRecordID, namespace, handle)
		return nil
	}

	if err := discovery.DeleteRecord(ctx, c, fullRecordID); err != nil {
		return err
	}

	fmt.Println(ui.SuccessStyle.Render(fmt.Sprintf("✓ Deleted record: %s", handle)))
	return nil
}
