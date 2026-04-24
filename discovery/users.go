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

// Package discovery provides functions for discovering Elementum resources.
package discovery

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/elementumltd/elementum-cli/internal/client"
	"github.com/elementumltd/elementum-cli/logger"
)

// ListUsers fetches a single page of users, optionally filtered by a name/email query.
// If opts.Query is empty, no filter is applied (returns all users).
func ListUsers(ctx context.Context, c *client.Client, opts UserListOptions) (*UserListResult, error) {
	if opts.Limit <= 0 {
		opts.Limit = 25
	}

	var filter *json.RawMessage
	if opts.Query != "" {
		filter = client.BuildUserSearchFilter(opts.Query)
	}

	limit := opts.Limit
	var after *string
	if opts.After != "" {
		after = &opts.After
	}

	logger.Debug("fetching users", "limit", opts.Limit, "after", opts.After, "query", opts.Query)

	resp, err := client.SearchUsers(ctx, c.Genqlient(), &limit, after, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch users: %w", err)
	}

	result := &UserListResult{
		Users:       make([]User, 0, len(resp.Organization.Users.Edges)),
		Total:       resp.Organization.Users.Total,
		HasNextPage: resp.Organization.Users.PageInfo.HasNextPage,
	}
	if resp.Organization.Users.PageInfo.EndCursor != nil {
		result.EndCursor = *resp.Organization.Users.PageInfo.EndCursor
	}

	for _, edge := range resp.Organization.Users.Edges {
		u := User{
			ID:        edge.Node.Id,
			Email:     edge.Node.Email,
			Name:      edge.Node.Name,
			FirstName: edge.Node.FirstName,
			LastName:  edge.Node.LastName,
			Status:    string(edge.Node.Status),
		}
		if edge.Node.JobTitle != nil {
			u.JobTitle = *edge.Node.JobTitle
		}
		result.Users = append(result.Users, u)
	}

	return result, nil
}

// ListAllUsers fetches all users by auto-paginating until there are no more pages.
// Mirrors the ListAllRecords pattern in discovery/records.go.
func ListAllUsers(ctx context.Context, c *client.Client, opts UserListOptions) (*UserListResult, error) {
	var allUsers []User
	cursor := opts.After

	first, err := ListUsers(ctx, c, UserListOptions{
		Limit: opts.Limit,
		After: cursor,
		Query: opts.Query,
	})
	if err != nil {
		return nil, err
	}
	allUsers = append(allUsers, first.Users...)
	cursor = first.EndCursor

	for first.HasNextPage {
		next, err := ListUsers(ctx, c, UserListOptions{
			Limit: opts.Limit,
			After: cursor,
			Query: opts.Query,
		})
		if err != nil {
			return nil, err
		}
		allUsers = append(allUsers, next.Users...)

		// Guard against a server returning HasNextPage=true without advancing
		// the cursor — would otherwise loop forever on the same request.
		if next.EndCursor == "" || next.EndCursor == cursor {
			break
		}
		cursor = next.EndCursor
		first.HasNextPage = next.HasNextPage
	}

	return &UserListResult{
		Users:       allUsers,
		Total:       first.Total,
		HasNextPage: false,
		EndCursor:   "",
	}, nil
}
