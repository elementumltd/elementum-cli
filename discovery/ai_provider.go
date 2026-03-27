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
