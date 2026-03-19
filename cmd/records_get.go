// Copyright (c) ElementumAI, Inc.
// SPDX-License-Identifier: MPL-2.0

package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/elementumltd/elementum-cli/auth"
	"github.com/elementumltd/elementum-cli/ui"
	"github.com/elementumltd/elementum-cli/internal/client"
	"github.com/spf13/cobra"
)

var recordsGetCmd = &cobra.Command{
	Use:   "get <namespace> <handle>",
	Short: "Get a single record by handle",
	Long: `Fetch and display a single record from an Elementum App.

Examples:
  ei records get support-tickets TKT-123
  ei records get support-tickets TKT-123 --json
  ei records get --id "abc123:TKT-123"`,
	Args: cobra.RangeArgs(0, 2),
	RunE: runRecordsGet,
}

func init() {
	recordsGetCmd.Flags().String("id", "", "Full record ID (format: aspectID:handle)")
}

func runRecordsGet(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	c, err := auth.GetClientFromCmd(cmd)
	if err != nil {
		return err
	}

	recordID, _ := cmd.Flags().GetString("id")

	var fullRecordID string
	if recordID != "" {
		fullRecordID = recordID
	} else {
		if len(args) < 2 {
			return fmt.Errorf("usage: ei records get <namespace> <handle> (or --id <record-id>)")
		}
		namespace := args[0]
		handle := args[1]

		aspectID, _, resolveErr := resolveAspectByNamespace(ctx, c, namespace)
		if resolveErr != nil {
			return fmt.Errorf("failed to resolve namespace %q: %w", namespace, resolveErr)
		}
		fullRecordID = aspectID + ":" + handle
	}

	resp, err := client.GetRecord(ctx, c.Genqlient(), fullRecordID)
	if err != nil {
		return fmt.Errorf("failed to get record: %w", err)
	}

	if resp.Organization.Record == nil {
		return fmt.Errorf("record not found: %s", fullRecordID)
	}

	// Dereference pointer-to-interface
	rec := *resp.Organization.Record
	recData := map[string]interface{}{
		"id":         rec.GetId(),
		"handle":     rec.GetHandle(),
		"title":      safeStringFromPtrCmd(rec.GetTitle()),
		"url":        rec.GetUrl(),
		"created_at": safeStringFromPtrCmd(rec.GetCreatedAt()),
		"updated_at": safeStringFromPtrCmd(rec.GetUpdatedAt()),
	}

	// Parse the data field
	if rec.GetData() != nil {
		var data map[string]interface{}
		if err := json.Unmarshal(rec.GetData(), &data); err == nil {
			recData["data"] = data
		}
	}

	if isJSONOutput(cmd) {
		return outputJSON(recData)
	}

	fmt.Println()
	fmt.Println(ui.TitleStyle.Render(fmt.Sprintf("Record: %s", rec.GetHandle())))
	fmt.Println()
	fmt.Printf("  %s  %s\n", ui.MutedStyle.Render("ID:"), rec.GetId())
	fmt.Printf("  %s  %s\n", ui.MutedStyle.Render("Handle:"), rec.GetHandle())
	if title := rec.GetTitle(); title != nil {
		fmt.Printf("  %s  %s\n", ui.MutedStyle.Render("Title:"), *title)
	}
	fmt.Printf("  %s  %s\n", ui.MutedStyle.Render("URL:"), rec.GetUrl())

	if rec.GetData() != nil {
		var data map[string]interface{}
		if err := json.Unmarshal(rec.GetData(), &data); err == nil && len(data) > 0 {
			fmt.Println()
			fmt.Println(ui.SubtitleStyle.Render("Fields:"))
			for key, val := range data {
				display := formatFieldValueForDisplay(val)
				if display != "" {
					fmt.Printf("  %s  %s\n", ui.MutedStyle.Render(key+":"), display)
				}
			}
		}
	}

	fmt.Println()
	return nil
}

func formatFieldValueForDisplay(val interface{}) string {
	if val == nil {
		return ""
	}
	switch v := val.(type) {
	case string:
		if len(v) > 100 {
			return v[:97] + "..."
		}
		return v
	case float64:
		if v == float64(int64(v)) {
			return fmt.Sprintf("%d", int64(v))
		}
		return fmt.Sprintf("%.2f", v)
	case bool:
		if v {
			return "true"
		}
		return "false"
	case map[string]interface{}:
		if label, ok := v["label"].(string); ok {
			return label
		}
		if name, ok := v["name"].(string); ok {
			return name
		}
		b, _ := json.Marshal(v)
		return string(b)
	case []interface{}:
		var parts []string
		for _, item := range v {
			parts = append(parts, formatFieldValueForDisplay(item))
		}
		return strings.Join(parts, ", ")
	default:
		return fmt.Sprintf("%v", val)
	}
}

func safeStringFromPtrCmd(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
