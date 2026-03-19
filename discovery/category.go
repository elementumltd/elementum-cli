// Copyright (c) ElementumAI, Inc.
// SPDX-License-Identifier: MPL-2.0

package discovery

import (
	"context"
	"fmt"

	"github.com/elementumltd/elementum-cli/internal/client"
)

// ListCategories returns all categories in the organization using genqlient
func ListCategories(ctx context.Context, c *client.Client) ([]Category, error) {
	result, err := client.GetCategories(ctx, c.Genqlient())
	if err != nil {
		return nil, fmt.Errorf("failed to fetch categories: %w", err)
	}

	categories := make([]Category, 0, len(result.Organization.Categories.Edges))
	for _, edge := range result.Organization.Categories.Edges {
		categories = append(categories, Category{
			ID:   edge.Node.Id,
			Name: edge.Node.Name,
		})
	}

	return categories, nil
}
