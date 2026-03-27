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
