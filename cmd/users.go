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
	"github.com/elementumltd/elementum-cli/ui"
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

var usersListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all users",
	Long:  "List all users in your organization. Supports pagination with --limit, --after, and --all flags.",
	RunE:  runUsersList,
}

func init() {
	usersListCmd.Flags().Int("limit", 25, "Maximum number of users to fetch per request")
	usersListCmd.Flags().String("after", "", "Pagination cursor (fetch users after this cursor)")
	usersListCmd.Flags().Bool("all", false, "Fetch all users (auto-paginate)")

	usersCmd.AddCommand(usersSearchCmd)
	usersCmd.AddCommand(usersListCmd)
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
		fmt.Printf("%s %s...\n", ui.InfoStyle.Render("⠿"), ui.InfoStyle.Render(fmt.Sprintf("Searching users for %q", query)))
	}

	result, err := discovery.ListUsers(ctx, c, discovery.UserListOptions{
		Limit: 25,
		Query: query,
	})
	if err != nil {
		return fmt.Errorf("search failed: %w", err)
	}

	// JSON output
	if isJSONOutput(cmd) {
		return outputJSON(map[string]interface{}{
			"users": result.Users,
			"total": result.Total,
		})
	}

	if len(result.Users) == 0 {
		fmt.Println()
		fmt.Println(ui.WarningStyle.Render(fmt.Sprintf("No users found matching %q.", query)))
		fmt.Println()
		return nil
	}

	// Create table
	table := ui.NewTable([]string{"NAME", "EMAIL", "STATUS", "JOB TITLE"})

	for _, user := range result.Users {
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
	total := result.Total
	displayed := len(result.Users)
	if total > displayed {
		fmt.Println(ui.MutedStyle.Render(fmt.Sprintf("Showing %d of %d matching users", displayed, total)))
	} else {
		fmt.Println(ui.MutedStyle.Render(fmt.Sprintf("Found %d matching users", total)))
	}
	fmt.Println()

	return nil
}

func runUsersList(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	c, err := auth.GetClientFromCmd(cmd)
	if err != nil {
		return err
	}

	limit, _ := cmd.Flags().GetInt("limit")
	after, _ := cmd.Flags().GetString("after")
	fetchAll, _ := cmd.Flags().GetBool("all")

	opts := discovery.UserListOptions{Limit: limit, After: after, All: fetchAll}

	if !isJSONOutput(cmd) {
		fmt.Println(ui.InfoStyle.Render("⠿ Loading users..."))
	}

	var result *discovery.UserListResult
	if fetchAll {
		result, err = discovery.ListAllUsers(ctx, c, opts)
	} else {
		result, err = discovery.ListUsers(ctx, c, opts)
	}
	if err != nil {
		return fmt.Errorf("failed to fetch users: %w", err)
	}

	if isJSONOutput(cmd) {
		return outputJSON(map[string]interface{}{
			"users":       result.Users,
			"total":       result.Total,
			"hasNextPage": result.HasNextPage,
			"endCursor":   result.EndCursor,
		})
	}

	if len(result.Users) == 0 {
		fmt.Println()
		fmt.Println(ui.WarningStyle.Render("No users found."))
		fmt.Println()
		return nil
	}

	table := ui.NewTable([]string{"NAME", "EMAIL", "STATUS", "JOB TITLE"})
	for _, u := range result.Users {
		table.AddRow(u.Name, u.Email, u.Status, u.JobTitle)
	}

	fmt.Println()
	fmt.Println(ui.TitleStyle.Render("Users"))
	fmt.Println()
	fmt.Println(table.Render())
	fmt.Println()

	if result.HasNextPage {
		fmt.Println(ui.MutedStyle.Render(fmt.Sprintf(
			"Showing %d of %d users (more available, use --all to fetch all or --after %q to continue)",
			len(result.Users), result.Total, result.EndCursor)))
	} else {
		fmt.Println(ui.MutedStyle.Render(fmt.Sprintf("Total: %d users", result.Total)))
	}
	fmt.Println()

	return nil
}
