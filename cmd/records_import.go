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
	"time"

	"github.com/elementumltd/elementum-cli/auth"
	"github.com/elementumltd/elementum-cli/discovery"
	"github.com/elementumltd/elementum-cli/ui"
	"github.com/spf13/cobra"
)

var recordsImportCmd = &cobra.Command{
	Use:   "import <namespace> <file>",
	Short: "Import records from a file (CSV, JSON, JSONL)",
	Long: `Bulk import records into an Elementum App from a file.

Supported formats: CSV, TSV, JSON, JSONL/NDJSON

The file format is auto-detected from the extension. Column headers (CSV)
or field keys (JSON) are matched to field names in the target app.

CSV files should have a header row with field names as columns.
JSON files can be an array of objects, or an object with a "records" key.
JSONL files should have one JSON object per line.

Examples:
  # Import from CSV
  ei records import support-tickets tickets.csv

  # Import from JSON
  ei records import support-tickets data.json

  # Import from JSONL/NDJSON
  ei records import support-tickets records.jsonl

  # Dry run (validate without creating)
  ei records import support-tickets tickets.csv --dry-run

  # Specify format explicitly
  ei records import support-tickets data.txt --format csv`,
	Args: cobra.ExactArgs(2),
	RunE: runRecordsImport,
}

func init() {
	recordsImportCmd.Flags().Bool("dry-run", false, "Validate file and show what would be imported without creating records")
	recordsImportCmd.Flags().String("format", "", "Force file format: csv, json, jsonl (default: auto-detect from extension)")
	recordsImportCmd.Flags().Int("batch-size", 100, "Number of records per API batch")
}

func runRecordsImport(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	c, err := auth.GetClientFromCmd(cmd)
	if err != nil {
		return err
	}

	namespace := args[0]
	filePath := args[1]
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	formatStr, _ := cmd.Flags().GetString("format")

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
		fmt.Println(ui.InfoStyle.Render(fmt.Sprintf("Loading records from %s...", filePath)))
	}

	// Load records from file
	rawRecords, err := discovery.LoadRecordsFromFile(filePath, format)
	if err != nil {
		return fmt.Errorf("failed to load file: %w", err)
	}

	if len(rawRecords) == 0 {
		fmt.Println(ui.WarningStyle.Render("No records found in file."))
		return nil
	}

	if !isJSONOutput(cmd) {
		fmt.Println(ui.InfoStyle.Render(fmt.Sprintf("Loaded %d records, resolving fields...", len(rawRecords))))
	}

	// Get aspect info including type and fields
	aspectInfo, err := discovery.GetAspectInfo(ctx, c, aspectID)
	if err != nil {
		return fmt.Errorf("failed to fetch aspect info: %w", err)
	}
	fields := aspectInfo.Fields

	// Validate required ID field if this object has a non-system HANDLE field
	if len(rawRecords) > 0 && discovery.RequiresIDField(fields) {
		idField := discovery.GetIDField(fields)
		if idField != nil {
			// Check if the first record has the ID field
			firstRecord := rawRecords[0]
			hasIDField := false
			for key := range firstRecord {
				if key == idField.Name || key == idField.ID {
					hasIDField = true
					break
				}
			}
			if !hasIDField {
				return fmt.Errorf("missing required field %q in import data\n\nThis object has a user-managed ID field that must be provided.\nEnsure your import file includes a %q column", idField.Name, idField.Name)
			}
		}
	}

	// Resolve field names and values
	resolved, err := discovery.ResolveRecordFieldValues(ctx, c, fields, rawRecords)
	if err != nil {
		return err
	}

	if dryRun {
		return displayImportDryRunWithAspectInfo(cmd, filePath, format, rawRecords, fields, aspectInfo)
	}

	if !isJSONOutput(cmd) {
		fmt.Println(ui.InfoStyle.Render(fmt.Sprintf("Creating %d records...", len(resolved))))
	}

	start := time.Now()

	result, err := discovery.BulkCreateWithFallback(ctx, c, discovery.BulkCreateInput{
		AspectID: aspectID,
		Fields:   fields,
		Records:  resolved,
	})
	if err != nil {
		return err
	}

	elapsed := time.Since(start)

	if isJSONOutput(cmd) {
		return outputJSON(map[string]interface{}{
			"created":  result.Total,
			"errors":   result.Errors,
			"duration": elapsed.String(),
		})
	}

	fmt.Println()
	fmt.Println(ui.SuccessStyle.Render(fmt.Sprintf("Imported %d records into %s in %s", result.Total, namespace, elapsed.Round(time.Millisecond))))

	if len(result.Errors) > 0 {
		fmt.Println()
		fmt.Println(ui.WarningStyle.Render(fmt.Sprintf("%d batch errors:", len(result.Errors))))
		for _, e := range result.Errors {
			fmt.Printf("  - %s\n", e)
		}
	}
	fmt.Println()

	return nil
}

func displayImportDryRun(cmd *cobra.Command, filePath string, format discovery.FileFormat, records []map[string]interface{}, fields []discovery.AspectFieldInfo) error {
	return displayImportDryRunWithAspectInfo(cmd, filePath, format, records, fields, nil)
}

func displayImportDryRunWithAspectInfo(cmd *cobra.Command, filePath string, format discovery.FileFormat, records []map[string]interface{}, fields []discovery.AspectFieldInfo, aspectInfo *discovery.AspectInfo) error {
	if isJSONOutput(cmd) {
		result := map[string]interface{}{
			"file":         filePath,
			"format":       string(format),
			"record_count": len(records),
			"sample":       records[0],
		}
		if aspectInfo != nil {
			result["aspect_type"] = string(aspectInfo.Type)
			result["aspect_name"] = aspectInfo.Name
		}
		return outputJSON(result)
	}

	fmt.Println()
	if aspectInfo != nil {
		fmt.Println(ui.TitleStyle.Render(fmt.Sprintf("Dry Run - Import Preview for %s (%s)", aspectInfo.Name, aspectInfo.Type)))
	} else {
		fmt.Println(ui.TitleStyle.Render("Dry Run - Import Preview"))
	}
	fmt.Println()
	fmt.Printf("  %s  %s\n", ui.MutedStyle.Render("File:"), filePath)
	fmt.Printf("  %s  %s\n", ui.MutedStyle.Render("Format:"), string(format))
	fmt.Printf("  %s  %d\n", ui.MutedStyle.Render("Records:"), len(records))
	fmt.Println()

	// Show column/field mapping
	if len(records) > 0 {
		fmt.Println(ui.SubtitleStyle.Render("Field Mapping:"))
		fieldByName := make(map[string]discovery.AspectFieldInfo)
		for _, f := range fields {
			fieldByName[f.Name] = f
		}
		for key := range records[0] {
			matched := "?"
			for _, f := range fields {
				if f.Name == key || f.ID == key {
					matched = fmt.Sprintf("%s (%s)", f.Name, f.Type)
					break
				}
			}
			fmt.Printf("  %s -> %s\n", key, matched)
		}
		fmt.Println()

		// Show first record preview
		fmt.Println(ui.SubtitleStyle.Render("First Record Preview:"))
		for key, val := range records[0] {
			fmt.Printf("  %s = %v\n", key, val)
		}
		fmt.Println()
	}

	fmt.Println(ui.MutedStyle.Render("Run without --dry-run to import records."))
	fmt.Println()

	return nil
}
