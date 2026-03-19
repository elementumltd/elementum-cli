// Copyright (c) ElementumAI, Inc.
// SPDX-License-Identifier: MPL-2.0

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
