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
	"sort"

	"github.com/elementumltd/elementum-cli/internal/client"
)

// ListFeatureFlags returns all feature flags in the organization with their enabled state
func ListFeatureFlags(ctx context.Context, c *client.Client) ([]FeatureFlag, error) {
	result, err := client.GetFeatureFlags(ctx, c.Genqlient())
	if err != nil {
		return nil, fmt.Errorf("failed to fetch feature flags: %w", err)
	}

	org := result.GetOrganization()

	// Build a set of enabled features for quick lookup
	enabledSet := make(map[string]bool)
	for _, f := range org.GetEnabledFeatures() {
		enabledSet[string(f)] = true
	}

	// Convert all features to our type, marking enabled status
	allFeatures := org.GetFeatures()
	flags := make([]FeatureFlag, 0, len(allFeatures))
	for _, f := range allFeatures {
		featureName := string(f)
		flags = append(flags, FeatureFlag{
			Feature: featureName,
			Enabled: enabledSet[featureName],
		})
	}

	// Sort flags alphabetically by feature name
	sort.Slice(flags, func(i, j int) bool {
		return flags[i].Feature < flags[j].Feature
	})

	return flags, nil
}
