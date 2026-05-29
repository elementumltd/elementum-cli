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

// CategoryAspect is a lightweight descriptor of an aspect (app/element/task)
// inside a category, as returned by ListCategoryAspects.
type CategoryAspect struct {
	ID       string
	Typename string // "AspectApp", "AspectElement", "AspectTask"
	Name     string
}

// ListCategoryAspects returns all aspects that live inside a category so we
// can pull in child elements that the recursive task/tool graph wouldn't
// reach on its own (e.g., elements referenced only by standalone AI search
// tables or by field-id lookups in the hand-authored truth).
func ListCategoryAspects(ctx context.Context, c *client.Client, categoryID string) ([]CategoryAspect, error) {
	query := `
		query GetCategoryAspects($categoryId: ID!, $first: Int!) {
			organization {
				category(id: $categoryId) {
					id
					aspects(first: $first) {
						edges {
							node {
								__typename
								id
								... on AspectApp { name }
								... on AspectElement { name }
								... on AspectTask { name }
							}
						}
					}
				}
			}
		}
	`

	var result struct {
		Organization struct {
			Category *struct {
				ID      string `json:"id"`
				Aspects struct {
					Edges []struct {
						Node struct {
							Typename string `json:"__typename"`
							ID       string `json:"id"`
							Name     string `json:"name"`
						} `json:"node"`
					} `json:"edges"`
				} `json:"aspects"`
			} `json:"category"`
		} `json:"organization"`
	}

	err := c.ExecuteInto(ctx, query, map[string]interface{}{
		"categoryId": categoryID,
		"first":      500, // <app-namespace> has ~12 aspects; 500 is generous headroom
	}, &result)
	if err != nil {
		return nil, fmt.Errorf("failed to list aspects in category %s: %w", categoryID, err)
	}

	if result.Organization.Category == nil {
		return nil, nil
	}

	out := make([]CategoryAspect, 0, len(result.Organization.Category.Aspects.Edges))
	for _, edge := range result.Organization.Category.Aspects.Edges {
		out = append(out, CategoryAspect{
			ID:       edge.Node.ID,
			Typename: edge.Node.Typename,
			Name:     edge.Node.Name,
		})
	}
	return out, nil
}
