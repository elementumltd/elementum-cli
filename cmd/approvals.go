// Copyright (c) ElementumAI, Inc.
// SPDX-License-Identifier: MPL-2.0

package cmd

import (
	"github.com/spf13/cobra"
)

var approvalsCmd = &cobra.Command{
	Use:   "approvals",
	Short: "Manage approvals for records",
	Long:  "Commands for listing and responding to approval requests.",
}

func init() {
	approvalsCmd.AddCommand(approvalsListCmd)
	approvalsCmd.AddCommand(approvalsApproveCmd)
	approvalsCmd.AddCommand(approvalsDenyCmd)
}

// GetApprovalsCmd returns the approvals command for registration
func GetApprovalsCmd() *cobra.Command {
	return approvalsCmd
}
