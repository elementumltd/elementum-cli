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

	"github.com/elementumltd/elementum-cli/auth"
	"github.com/elementumltd/elementum-cli/discovery"
	"github.com/elementumltd/elementum-cli/logger"
	"github.com/elementumltd/elementum-cli/ui"
	"github.com/spf13/cobra"
)

var recordsExportCmd = &cobra.Command{
	Use:   "export <namespace> <file>",
	Short: "Export records to a file (CSV, JSON, JSONL)",
	Long: `Export all records from an Elementum App to a file.

The output format is auto-detected from the file extension.
All records are fetched (auto-paginated) and written to the file.

Examples:
  # Export to CSV
  ei records export support-tickets tickets.csv

  # Export to JSON
  ei records export support-tickets data.json

  # Export to JSONL
  ei records export support-tickets records.jsonl

  # Export with specific format
  ei records export support-tickets output.txt --format csv`,
	Args: cobra.ExactArgs(2),
	RunE: runRecordsExport,
}

func init() {
	recordsExportCmd.Flags().String("format", "", "Force output format: csv, json, jsonl (default: auto-detect)")
	recordsExportCmd.Flags().Int("limit", 100, "Records per page when fetching")
}

func runRecordsExport(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	c, err := auth.GetClientFromCmd(cmd)
	if err != nil {
		return err
	}

	namespace := args[0]
	filePath := args[1]
	formatStr, _ := cmd.Flags().GetString("format")
	limit, _ := cmd.Flags().GetInt("limit")

	// Resolve namespace
	aspectID, _, err := resolveAspectByNamespace(ctx, c, namespace)
	if err != nil {
		return fmt.Errorf("failed to resolve namespace %q: %w", namespace, err)
	}

	// Detect format
	var format discovery.FileFormat
	if formatStr != "" {
		format = discovery.FileFormat(formatStr)
	} else {
		format, err = discovery.DetectFileFormat(filePath)
		if err != nil {
			return err
		}
	}

	if !isJSONOutput(cmd) {
		fmt.Println(ui.InfoStyle.Render("Fetching all records..."))
	}

	// Fetch all records
	result, err := discovery.ListAllRecords(ctx, c, aspectID, discovery.RecordListOptions{Limit: limit})
	if err != nil {
		return fmt.Errorf("failed to fetch records: %w", err)
	}

	if len(result.Records) == 0 {
		fmt.Println(ui.WarningStyle.Render("No records to export."))
		return nil
	}

	// Resolve picklist values for readable output
	picklistValues, err := discovery.FetchPicklistValues(ctx, c, aspectID)
	if err != nil {
		logger.Debug("failed to fetch picklist values for export", "error", err)
	} else if len(picklistValues) > 0 {
		discovery.ResolvePicklistValues(result.Records, picklistValues)
	}

	// Write to file
	if err := discovery.ExportRecordsToFile(result.Records, filePath, format); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	if !isJSONOutput(cmd) {
		fmt.Println()
		fmt.Println(ui.SuccessStyle.Render(fmt.Sprintf("Exported %d records to %s", len(result.Records), filePath)))
		fmt.Println()
	}

	return nil
}
