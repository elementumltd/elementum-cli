// Copyright (c) ElementumAI, Inc.
// SPDX-License-Identifier: MPL-2.0

package discovery

import (
	"context"

	"github.com/elementumltd/elementum-cli/internal/client"
)

// ListPhoneProviders returns all phone providers in the organization
func ListPhoneProviders(ctx context.Context, c *client.Client) ([]PhoneProvider, error) {
	first := 1000
	result, err := client.GetPhoneProviders(ctx, c.Genqlient(), &first)
	if err != nil {
		return nil, err
	}

	if result.Organization.PhoneProviders == nil {
		return []PhoneProvider{}, nil
	}

	providers := make([]PhoneProvider, 0, len(result.Organization.PhoneProviders.Edges))
	for _, edge := range result.Organization.PhoneProviders.Edges {
		node := edge.Node
		providers = append(providers, PhoneProvider{
			ID:           node.GetId(),
			Name:         node.GetName(),
			System:       node.GetSystem(),
			ProviderType: getPhoneProviderType(node),
		})
	}

	return providers, nil
}

// GetPhoneProvider returns a phone provider by ID
func GetPhoneProvider(ctx context.Context, c *client.Client, id string) (*PhoneProvider, error) {
	result, err := client.GetPhoneProvider(ctx, c.Genqlient(), id)
	if err != nil {
		return nil, err
	}

	if result.Organization.PhoneProvider == nil {
		return nil, nil
	}

	provider := *result.Organization.PhoneProvider
	return &PhoneProvider{
		ID:           provider.GetId(),
		Name:         provider.GetName(),
		System:       provider.GetSystem(),
		ProviderType: getPhoneProviderTypeFromGet(provider),
	}, nil
}

// getPhoneProviderType determines the provider type from list query node
func getPhoneProviderType(node client.GetPhoneProvidersOrganizationPhoneProvidersPhoneProviderConnectionEdgesPhoneProviderEdgeNodePhoneProvider) string {
	switch node.(type) {
	case *client.GetPhoneProvidersOrganizationPhoneProvidersPhoneProviderConnectionEdgesPhoneProviderEdgeNodePhoneProviderElementum:
		return "elementum"
	case *client.GetPhoneProvidersOrganizationPhoneProvidersPhoneProviderConnectionEdgesPhoneProviderEdgeNodePhoneProviderTwilio:
		return "twilio"
	case *client.GetPhoneProvidersOrganizationPhoneProvidersPhoneProviderConnectionEdgesPhoneProviderEdgeNodePhoneProviderElementumTwilio:
		return "elementum_twilio"
	case *client.GetPhoneProvidersOrganizationPhoneProvidersPhoneProviderConnectionEdgesPhoneProviderEdgeNodePhoneProviderUnconfigured:
		return "unconfigured"
	default:
		return "unknown"
	}
}

// getPhoneProviderTypeFromGet determines the provider type from get query response
func getPhoneProviderTypeFromGet(provider client.GetPhoneProviderOrganizationPhoneProvider) string {
	switch provider.(type) {
	case *client.GetPhoneProviderOrganizationPhoneProviderPhoneProviderElementum:
		return "elementum"
	case *client.GetPhoneProviderOrganizationPhoneProviderPhoneProviderTwilio:
		return "twilio"
	case *client.GetPhoneProviderOrganizationPhoneProviderPhoneProviderElementumTwilio:
		return "elementum_twilio"
	case *client.GetPhoneProviderOrganizationPhoneProviderPhoneProviderUnconfigured:
		return "unconfigured"
	default:
		return "unknown"
	}
}

// GetPhoneServicesForApp returns all phone services for an app
func GetPhoneServicesForApp(ctx context.Context, c *client.Client, aspectId string) ([]PhoneService, error) {
	first := 1000
	result, err := client.GetPhoneServices(ctx, c.Genqlient(), aspectId, &first)
	if err != nil {
		return nil, err
	}

	aspect := result.Organization.Aspect
	if aspect == nil {
		return []PhoneService{}, nil
	}

	services := extractPhoneServicesFromAspect(*aspect)
	return services, nil
}

// extractPhoneServicesFromAspect extracts phone services from aspect interface
func extractPhoneServicesFromAspect(aspect client.GetPhoneServicesOrganizationAspect) []PhoneService {
	var services []PhoneService

	switch a := aspect.(type) {
	case *client.GetPhoneServicesOrganizationAspectAspectApp:
		if a.PhoneServices != nil {
			for _, edge := range a.PhoneServices.Edges {
				services = append(services, convertPhoneServiceInfo(edge.Node))
			}
		}
	case *client.GetPhoneServicesOrganizationAspectAspectElement:
		if a.PhoneServices != nil {
			for _, edge := range a.PhoneServices.Edges {
				services = append(services, convertPhoneServiceInfo(edge.Node))
			}
		}
	}

	return services
}

// convertPhoneServiceInfo converts GraphQL PhoneServiceInfo to discovery PhoneService
func convertPhoneServiceInfo(service client.PhoneServiceInfo) PhoneService {
	ps := PhoneService{
		ID:         service.GetId(),
		ProviderID: service.GetProvider().GetId(),
	}

	if service.GetPhoneNumber() != nil {
		ps.PhoneNumber = *service.GetPhoneNumber()
	}

	if service.GetAgent() != nil {
		ps.AgentID = service.GetAgent().Id
		ps.AgentName = service.GetAgent().Name
	}

	if service.GetAspect() != nil {
		ps.AspectID = (*service.GetAspect()).GetId()
	}

	regProps := service.GetRegistrationProperties()
	ps.RegistrationProperties = PhoneRegistrationProperties{
		GenerateFirstMessage: regProps.GetGenerateFirstMessage(),
		ExpandLanguages:      regProps.GetExpandLanguages(),
		PreferredLanguages:   regProps.GetPreferredLanguages(),
	}
	if regProps.GetDefaultLanguage() != nil {
		ps.RegistrationProperties.DefaultLanguage = *regProps.GetDefaultLanguage()
	}
	if regProps.GetEnableLanguageRouting() != nil {
		ps.RegistrationProperties.EnableLanguageRouting = *regProps.GetEnableLanguageRouting()
	}

	return ps
}
