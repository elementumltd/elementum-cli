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

	"github.com/elementumltd/elementum-cli/auth"
	"github.com/elementumltd/elementum-cli/logger"
	"github.com/elementumltd/elementum-cli/ui"
	"github.com/elementumltd/elementum-cli/internal/client"
	"github.com/spf13/cobra"
)

var approvalsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List approvals for a record",
	Long:  "List all approval chains and their status for a specific record.",
	RunE: func(cmd *cobra.Command, args []string) error {
		recordID, _ := cmd.Flags().GetString("record-id")
		outputJSON, _ := cmd.Root().Flags().GetBool("json")

		if recordID == "" {
			return fmt.Errorf("--record-id is required")
		}

		ctx := context.Background()
		c, err := auth.GetClientFromCmd(cmd)
		if err != nil {
			return fmt.Errorf("failed to create client: %w", err)
		}

		logger.Debug("Fetching approvals for record", "record_id", recordID)
		resp, err := client.GetRecordApprovals(ctx, c.Genqlient(), recordID)
		if err != nil {
			return fmt.Errorf("failed to get approvals: %w", err)
		}

		if resp.Organization.Record == nil {
			return fmt.Errorf("record not found: %s", recordID)
		}

		record := *resp.Organization.Record

		if outputJSON {
			type ApprovalInfo struct {
				ID               string  `json:"id"`
				Status           string  `json:"status"`
				Name             string  `json:"name,omitempty"`
				Reason           string  `json:"reason,omitempty"`
				CreatedAt        string  `json:"createdAt"`
				TemplateName     string  `json:"templateName,omitempty"`
				ActiveApprovalID *string `json:"activeApprovalId,omitempty"`
			}

			var approvals []ApprovalInfo
			for _, edge := range record.GetApprovalChains().Edges {
				chain := edge.Node
				info := ApprovalInfo{
					ID:        chain.Id,
					Status:    string(chain.Status),
					Name:      ptrToStr(chain.Name),
					Reason:    ptrToStr(chain.Reason),
					CreatedAt: chain.CreatedAt,
				}
				if chain.ApprovalChainTemplate != nil {
					info.TemplateName = chain.ApprovalChainTemplate.Name
				}
				if chain.ActiveApprovalV2 != nil {
					approval := *chain.ActiveApprovalV2
					id := approval.GetId()
					info.ActiveApprovalID = &id
				}
				approvals = append(approvals, info)
			}

			output := map[string]interface{}{
				"recordId":    record.GetId(),
				"recordTitle": record.GetTitle(),
				"approvals":   approvals,
			}
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(output)
		}

		title := record.GetTitle()
		if title != nil {
			fmt.Printf("\nApprovals for record: %s\n\n", *title)
		} else {
			fmt.Printf("\nApprovals for record: %s\n\n", recordID)
		}

		approvalChains := record.GetApprovalChains()
		if len(approvalChains.Edges) == 0 {
			fmt.Println("No approval chains found for this record.")
			return nil
		}

		table := ui.NewTable([]string{"Approval ID", "Status", "Name", "Template", "Active Approval ID", "Created At"})

		for _, edge := range approvalChains.Edges {
			chain := edge.Node
			templateName := ""
			if chain.ApprovalChainTemplate != nil {
				templateName = chain.ApprovalChainTemplate.Name
			}
			activeID := "-"
			if chain.ActiveApprovalV2 != nil {
				approval := *chain.ActiveApprovalV2
				activeID = approval.GetId()
			}
			table.AddRow(
				chain.Id,
				string(chain.Status),
				ptrToStr(chain.Name),
				templateName,
				activeID,
				chain.CreatedAt,
			)
		}

		fmt.Println(table.Render())
		return nil
	},
}

func init() {
	approvalsListCmd.Flags().String("record-id", "", "Record ID to list approvals for (required)")
	_ = approvalsListCmd.MarkFlagRequired("record-id")
}

func ptrToStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
