// Copyright (c) ElementumAI, Inc.
// SPDX-License-Identifier: MPL-2.0

package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/elementumltd/elementum-cli/auth"
	"github.com/elementumltd/elementum-cli/logger"
	"github.com/elementumltd/elementum-cli/internal/client"
	"github.com/spf13/cobra"
)

var approvalsApproveCmd = &cobra.Command{
	Use:   "approve <approval-id>",
	Short: "Approve an approval request",
	Long:  "Approve a pending approval request by its ID.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		approvalID := args[0]
		comment, _ := cmd.Flags().GetString("comment")
		outputJSON, _ := cmd.Root().Flags().GetBool("json")

		ctx := context.Background()
		c, err := auth.GetClientFromCmd(cmd)
		if err != nil {
			return fmt.Errorf("failed to create client: %w", err)
		}

		logger.Debug("Approving approval", "approval_id", approvalID, "comment", comment)

		input := client.ApprovalStatusInput{
			Status: client.ApprovalStatusApproved,
		}
		if comment != "" {
			input.Reason = &comment
		}

		resp, err := client.UpdateApprovalStatus(ctx, c.Genqlient(), approvalID, input)
		if err != nil {
			return fmt.Errorf("failed to approve: %w", err)
		}

		if outputJSON {
			output := map[string]interface{}{
				"id":        resp.ApprovalUpdateV2.GetId(),
				"status":    string(resp.ApprovalUpdateV2.GetStatus()),
				"updatedAt": resp.ApprovalUpdateV2.GetUpdatedAt(),
			}
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(output)
		}

		fmt.Printf("Successfully approved approval %s\n", approvalID)
		fmt.Printf("New status: %s\n", resp.ApprovalUpdateV2.GetStatus())
		return nil
	},
}

func init() {
	approvalsApproveCmd.Flags().String("comment", "", "Optional comment/reason for the approval")
}
