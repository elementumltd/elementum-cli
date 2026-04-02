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
	"github.com/elementumltd/elementum-cli/ui"
	"github.com/elementumltd/elementum-cli/internal/client"
	"github.com/spf13/cobra"
)

var usersCmd = &cobra.Command{
	Use:   "users",
	Short: "Manage users",
	Long:  "Commands for searching and managing users in your organization.",
}

var usersSearchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Search for users by name or email",
	Long:  "Search for users in your organization by name or email address (case-insensitive partial match). Uses server-side filtering for efficiency.",
	Args:  cobra.ExactArgs(1),
	RunE:  runUsersSearch,
}

func init() {
	usersCmd.AddCommand(usersSearchCmd)
}

// GetUsersCmd returns the users command for registration
func GetUsersCmd() *cobra.Command {
	return usersCmd
}

func runUsersSearch(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	query := args[0]

	// Get authenticated client
	c, err := auth.GetClientFromCmd(cmd)
	if err != nil {
		return err
	}

	// Show spinner while searching (skip for JSON output)
	if !isJSONOutput(cmd) {
		fmt.Printf("%s %s...\n", ui.InfoStyle.Render("*"), ui.InfoStyle.Render(fmt.Sprintf("Searching users for %q", query)))
	}

	// Build server-side filter for name OR email matching
	filter := client.BuildUserSearchFilter(query)

	// Search users with server-side filtering (first page of 20)
	result, err := client.SearchUsers(ctx, c.Genqlient(), ptr(20), nil, filter)
	if err != nil {
		return fmt.Errorf("search failed: %w", err)
	}

	// Build user list for output
	type UserResult struct {
		ID       string `json:"id"`
		Name     string `json:"name"`
		Email    string `json:"email"`
		Status   string `json:"status"`
		JobTitle string `json:"job_title,omitempty"`
	}

	var users []UserResult
	for _, edge := range result.Organization.Users.Edges {
		jobTitle := ""
		if edge.Node.JobTitle != nil {
			jobTitle = *edge.Node.JobTitle
		}
		users = append(users, UserResult{
			ID:       edge.Node.Id,
			Name:     edge.Node.Name,
			Email:    edge.Node.Email,
			Status:   string(edge.Node.Status),
			JobTitle: jobTitle,
		})
	}

	// JSON output
	if isJSONOutput(cmd) {
		return outputJSON(map[string]interface{}{
			"users": users,
			"total": result.Organization.Users.Total,
		})
	}

	if len(users) == 0 {
		fmt.Println()
		fmt.Println(ui.WarningStyle.Render(fmt.Sprintf("No users found matching %q.", query)))
		fmt.Println()
		return nil
	}

	// Create table
	table := ui.NewTable([]string{"NAME", "EMAIL", "STATUS", "JOB TITLE"})

	for _, user := range users {
		table.AddRow(
			user.Name,
			user.Email,
			user.Status,
			user.JobTitle,
		)
	}

	// Display table
	fmt.Println()
	fmt.Println(ui.TitleStyle.Render(fmt.Sprintf("User Search Results for %q", query)))
	fmt.Println()
	fmt.Println(table.Render())
	fmt.Println()

	// Show total count (may be more than displayed)
	total := result.Organization.Users.Total
	displayed := len(users)
	if total > displayed {
		fmt.Println(ui.MutedStyle.Render(fmt.Sprintf("Showing %d of %d matching users", displayed, total)))
	} else {
		fmt.Println(ui.MutedStyle.Render(fmt.Sprintf("Found %d matching users", total)))
	}
	fmt.Println()

	return nil
}

// ptr returns a pointer to the given value
func ptr[T any](v T) *T {
	return &v
}
