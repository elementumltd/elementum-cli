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
	"os"
	"time"

	"github.com/elementumltd/elementum-cli/auth"
	"github.com/elementumltd/elementum-cli/discovery"
	"github.com/elementumltd/elementum-cli/ui"
	"github.com/spf13/cobra"
)

var recordsGenerateCmd = &cobra.Command{
	Use:   "generate <namespace>",
	Short: "Generate synthetic records for testing",
	Long: `Generate and optionally create synthetic records in an Elementum App.

Records are generated based on the app's field schema, producing realistic
test data for each field type. By default records are created in the app;
use --dry-run to preview or --output to write to a file instead.

Examples:
  # Generate 10 records and create them
  ei records generate support-tickets --count 10

  # Preview without creating
  ei records generate support-tickets --count 5 --dry-run

  # Write to file (does not create in app)
  ei records generate support-tickets --count 100 --output test-data.json

  # Reproducible generation with seed
  ei records generate support-tickets --count 50 --seed 42`,
	Args: cobra.ExactArgs(1),
	RunE: runRecordsGenerate,
}

func init() {
	recordsGenerateCmd.Flags().Int("count", 10, "Number of records to generate")
	recordsGenerateCmd.Flags().Int64("seed", 0, "Random seed for reproducible generation (0 = random)")
	recordsGenerateCmd.Flags().Bool("dry-run", false, "Preview generated records without creating them")
	recordsGenerateCmd.Flags().StringP("output", "o", "", "Write generated records to file instead of creating (supports .csv, .json, .jsonl)")
}

func runRecordsGenerate(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	c, err := auth.GetClientFromCmd(cmd)
	if err != nil {
		return err
	}

	namespace := args[0]
	count, _ := cmd.Flags().GetInt("count")
	seed, _ := cmd.Flags().GetInt64("seed")
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	outputFile, _ := cmd.Flags().GetString("output")

	if count <= 0 {
		return fmt.Errorf("--count must be positive")
	}

	// Resolve namespace
	aspectID, _, err := resolveAspectByNamespace(ctx, c, namespace)
	if err != nil {
		return fmt.Errorf("failed to resolve namespace %q: %w", namespace, err)
	}

	// Get aspect info including type and fields
	aspectInfo, err := discovery.GetAspectInfo(ctx, c, aspectID)
	if err != nil {
		return fmt.Errorf("failed to fetch aspect info: %w", err)
	}
	fields := aspectInfo.Fields

	if !isJSONOutput(cmd) {
		aspectTypeStr := string(aspectInfo.Type)
		fmt.Println(ui.InfoStyle.Render(fmt.Sprintf("Generating %d records for %s (%s)...", count, namespace, aspectTypeStr)))
	}

	// Generate records with aspect type awareness (Elements require ID field)
	records := discovery.GenerateRecordsForAspect(fields, count, seed, aspectInfo.Type)

	// Write to file mode
	if outputFile != "" {
		return writeGeneratedRecords(cmd, records, fields, outputFile)
	}

	// Dry run mode
	if dryRun {
		return displayGeneratedPreview(cmd, records, fields, count)
	}

	// Create records
	if !isJSONOutput(cmd) {
		fmt.Println(ui.InfoStyle.Render(fmt.Sprintf("Creating %d records...", len(records))))
	}

	start := time.Now()

	result, err := discovery.BulkCreateRecords(ctx, c, discovery.BulkCreateInput{
		AspectID: aspectID,
		Fields:   fields,
		Records:  records,
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
			"seed":     seed,
		})
	}

	fmt.Println()
	fmt.Println(ui.SuccessStyle.Render(fmt.Sprintf("Generated and created %d records in %s", result.Total, elapsed.Round(time.Millisecond))))
	if seed != 0 {
		fmt.Printf("  %s  %d\n", ui.MutedStyle.Render("Seed:"), seed)
	}
	if len(result.Errors) > 0 {
		fmt.Println()
		fmt.Println(ui.WarningStyle.Render(fmt.Sprintf("%d errors:", len(result.Errors))))
		for _, e := range result.Errors {
			fmt.Printf("  - %s\n", e)
		}
	}
	fmt.Println()

	return nil
}

func writeGeneratedRecords(cmd *cobra.Command, records []map[string]interface{}, fields []discovery.AspectFieldInfo, outputFile string) error {
	format, err := discovery.DetectFileFormat(outputFile)
	if err != nil {
		// Default to JSON
		format = discovery.FormatJSON
	}

	// Build field ID -> name map for readable output
	idToName := make(map[string]string)
	for _, f := range fields {
		idToName[f.ID] = f.Name
	}

	// Convert field IDs back to names for file output
	namedRecords := make([]map[string]interface{}, len(records))
	for i, rec := range records {
		named := make(map[string]interface{})
		for k, v := range rec {
			if name, ok := idToName[k]; ok {
				named[name] = v
			} else {
				named[k] = v
			}
		}
		namedRecords[i] = named
	}

	switch format {
	case discovery.FormatJSON:
		data, err := json.MarshalIndent(namedRecords, "", "  ")
		if err != nil {
			return err
		}
		if err := os.WriteFile(outputFile, data, 0644); err != nil {
			return err
		}
	case discovery.FormatCSV:
		// Convert to Record format for CSV export
		var recs []discovery.Record
		for _, rec := range namedRecords {
			r := discovery.Record{Data: rec}
			if title, ok := rec["Title"].(string); ok {
				r.Title = title
			}
			recs = append(recs, r)
		}
		if err := discovery.ExportRecordsToFile(recs, outputFile, format); err != nil {
			return err
		}
	case discovery.FormatJSONL:
		f, err := os.Create(outputFile)
		if err != nil {
			return err
		}
		defer f.Close()
		enc := json.NewEncoder(f)
		for _, rec := range namedRecords {
			if err := enc.Encode(rec); err != nil {
				return err
			}
		}
	}

	if !isJSONOutput(cmd) {
		fmt.Println()
		fmt.Println(ui.SuccessStyle.Render(fmt.Sprintf("Wrote %d generated records to %s", len(records), outputFile)))
		fmt.Println()
	}

	return nil
}

func displayGeneratedPreview(cmd *cobra.Command, records []map[string]interface{}, fields []discovery.AspectFieldInfo, count int) error {
	idToName := make(map[string]string)
	for _, f := range fields {
		idToName[f.ID] = f.Name
	}

	if isJSONOutput(cmd) {
		// Convert to named fields for JSON output
		named := make([]map[string]interface{}, 0, len(records))
		for _, rec := range records {
			nr := make(map[string]interface{})
			for k, v := range rec {
				if name, ok := idToName[k]; ok {
					nr[name] = v
				} else {
					nr[k] = v
				}
			}
			named = append(named, nr)
		}
		return outputJSON(map[string]interface{}{
			"count":   count,
			"records": named,
		})
	}

	fmt.Println()
	fmt.Println(ui.TitleStyle.Render(fmt.Sprintf("Preview: %d generated records", count)))
	fmt.Println()

	showCount := len(records)
	if showCount > 3 {
		showCount = 3
	}

	for i := 0; i < showCount; i++ {
		fmt.Println(ui.SubtitleStyle.Render(fmt.Sprintf("Record %d:", i+1)))
		for k, v := range records[i] {
			name := k
			if n, ok := idToName[k]; ok {
				name = n
			}
			valStr := fmt.Sprintf("%v", v)
			if len(valStr) > 60 {
				valStr = valStr[:57] + "..."
			}
			fmt.Printf("  %s = %s\n", name, valStr)
		}
		fmt.Println()
	}

	if len(records) > 3 {
		fmt.Println(ui.MutedStyle.Render(fmt.Sprintf("... and %d more records", len(records)-3)))
		fmt.Println()
	}

	fmt.Println(ui.MutedStyle.Render("Run without --dry-run to create these records."))
	fmt.Println()

	return nil
}
