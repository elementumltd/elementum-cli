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
	"strings"

	"github.com/elementumltd/elementum-cli/auth"
	"github.com/elementumltd/elementum-cli/discovery"
	"github.com/elementumltd/elementum-cli/ui"
	"github.com/spf13/cobra"
)

var recordsUpdateCmd = &cobra.Command{
	Use:   "update <namespace> <handle>",
	Short: "Update an existing record",
	Long: `Update fields on an existing record in an Elementum App, Element, or Task.

You can specify the record using namespace and handle, or by full record ID.

Examples:
  # Update by namespace and handle
  ei records update support-tickets TKT-123 -f "Status=Closed" -f "Priority=Low"

  # Update by record ID
  ei records update --id "abc123:TKT-123" -f "Status=Closed"

  # Update from JSON string
  ei records update support-tickets TKT-123 --data '{"Status": "Closed"}'`,
	Args: cobra.RangeArgs(0, 2),
	RunE: runRecordsUpdate,
}

func init() {
	recordsUpdateCmd.Flags().String("id", "", "Full record ID (format: aspectID:handle)")
	recordsUpdateCmd.Flags().StringArrayP("field", "f", nil, "Field value: \"Name=Value\" (repeatable)")
	recordsUpdateCmd.Flags().String("data", "", "JSON string with field values")
}

func runRecordsUpdate(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	c, err := auth.GetClientFromCmd(cmd)
	if err != nil {
		return err
	}

	recordID, _ := cmd.Flags().GetString("id")
	fieldFlags, _ := cmd.Flags().GetStringArray("field")
	dataJSON, _ := cmd.Flags().GetString("data")

	var aspectID, fullRecordID string

	if recordID != "" {
		fullRecordID = recordID
		// Extract aspectID from the record ID (format: aspectID:handle)
		parts := strings.SplitN(recordID, ":", 2)
		if len(parts) == 2 {
			aspectID = parts[0]
		}
	} else {
		if len(args) < 2 {
			return fmt.Errorf("usage: ei records update <namespace> <handle> -f Field=Value")
		}
		namespace := args[0]
		handle := args[1]

		var resolveErr error
		aspectID, _, resolveErr = resolveAspectByNamespace(ctx, c, namespace)
		if resolveErr != nil {
			return fmt.Errorf("failed to resolve namespace %q: %w", namespace, resolveErr)
		}
		fullRecordID = aspectID + ":" + handle
	}

	if len(fieldFlags) == 0 && dataJSON == "" {
		return fmt.Errorf("no field values provided; use -f or --data")
	}

	// Fetch fields for name resolution
	var fields []discovery.AspectFieldInfo
	if aspectID != "" {
		fields, err = discovery.GetAspectFields(ctx, c, aspectID)
		if err != nil {
			return fmt.Errorf("failed to fetch fields: %w", err)
		}
	}

	updateData := make(map[string]interface{})

	// Parse --data JSON
	if dataJSON != "" {
		var parsed map[string]interface{}
		if err := json.Unmarshal([]byte(dataJSON), &parsed); err != nil {
			return fmt.Errorf("invalid --data JSON: %w", err)
		}
		for k, v := range parsed {
			fieldID, _, resolveErr := discovery.ResolveFieldNameToID(fields, k)
			if resolveErr != nil {
				return fmt.Errorf("field %q: %w", k, resolveErr)
			}
			updateData[fieldID] = v
		}
	}

	// Parse -f flags (override --data)
	for _, ff := range fieldFlags {
		name, value, parseErr := discovery.ParseFieldAssignment(ff)
		if parseErr != nil {
			return parseErr
		}
		fieldID, fieldInfo, resolveErr := discovery.ResolveFieldNameToID(fields, name)
		if resolveErr != nil {
			return fmt.Errorf("field %q: %w", name, resolveErr)
		}
		resolved, resolveErr := resolveFieldValueWithLookup(ctx, c, fieldInfo, value)
		if resolveErr != nil {
			return fmt.Errorf("field %q: %w", name, resolveErr)
		}
		updateData[fieldID] = resolved
	}

	if !isJSONOutput(cmd) {
		fmt.Println(ui.InfoStyle.Render("Updating record..."))
	}

	updates := []discovery.BulkUpdateInput{{
		RecordID: fullRecordID,
		Fields:   updateData,
	}}

	result, err := discovery.BulkUpdateRecords(ctx, c, updates)
	if err != nil {
		return err
	}

	if isJSONOutput(cmd) {
		return outputJSON(result)
	}

	if result.UpdatedCount > 0 {
		rec := result.Records[0]
		fmt.Println()
		fmt.Println(ui.SuccessStyle.Render(fmt.Sprintf("Updated record %s", rec.Handle)))
		fmt.Printf("  %s  %s\n", ui.MutedStyle.Render("ID:"), rec.ID)
		if rec.Title != "" {
			fmt.Printf("  %s  %s\n", ui.MutedStyle.Render("Title:"), rec.Title)
		}
		fmt.Println()
	}

	return nil
}
