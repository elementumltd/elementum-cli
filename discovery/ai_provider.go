// Copyright (c) ElementumAI, Inc.
// SPDX-License-Identifier: MPL-2.0

package discovery

import (
	"context"
	"fmt"

	"github.com/elementumltd/elementum-cli/internal/client"
)

// ListAiProviders returns all AI providers available in the organization
func ListAiProviders(ctx context.Context, c *client.Client) ([]AiProvider, error) {
	result, err := client.GetAiProviders(ctx, c.Genqlient())
	if err != nil {
		return nil, fmt.Errorf("failed to fetch AI providers: %w", err)
	}

	providers := make([]AiProvider, 0, len(result.Organization.AiProviders))
	for _, p := range result.Organization.AiProviders {
		provider := AiProvider{
			ID:   p.GetId(),
			Name: p.GetName(),
		}

		// Convert available models
		for _, m := range p.GetAvailableModels() {
			provider.AvailableModels = append(provider.AvailableModels, AiModel{
				Name:       m.GetName(),
				ModelType:  string(m.GetModelType()),
				MultiModal: m.GetMultiModal(),
				Legacy:     m.GetLegacy(),
			})
		}

		// Convert available features
		for _, f := range p.GetAvailableFeatures() {
			provider.AvailableFeatures = append(provider.AvailableFeatures, AiFeature{
				Feature: string(f.GetFeature()),
				Name:    f.GetName(),
			})
		}

		providers = append(providers, provider)
	}

	return providers, nil
}

// ListAiProviderConnectors returns all AI provider connectors (configured models) in the organization
func ListAiProviderConnectors(ctx context.Context, c *client.Client) ([]AiProviderConnectorDetail, error) {
	// Fetch up to 100 connectors (should be sufficient for most orgs)
	first := 100
	result, err := client.GetAiProviderConnectors(ctx, c.Genqlient(), &first)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch AI provider connectors: %w", err)
	}

	connectors := make([]AiProviderConnectorDetail, 0, len(result.Organization.AiProviderConnectors.Edges))
	for _, edge := range result.Organization.AiProviderConnectors.Edges {
		node := edge.Node
		connector := AiProviderConnectorDetail{
			ID:                   node.GetId(),
			ProviderID:           node.GetProvider().GetId(),
			ProviderName:         node.GetProvider().GetName(),
			ApiType:              string(node.GetApiType()),
			CostPerMillionTokens: node.GetCostPerMillionTokens(),
		}

		// Convert model
		model := node.GetModel()
		connector.Model = AiModel{
			Name:       model.GetName(),
			ModelType:  string(model.GetModelType()),
			MultiModal: model.GetMultiModal(),
			Legacy:     model.GetLegacy(),
		}

		// Convert available features
		for _, f := range node.GetAvailableFeatures() {
			connector.AvailableFeatures = append(connector.AvailableFeatures, AiFeature{
				Feature: string(f.GetFeature()),
				Name:    f.GetName(),
			})
		}

		connectors = append(connectors, connector)
	}

	return connectors, nil
}
