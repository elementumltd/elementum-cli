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

package auth

import (
	"context"
	"fmt"

	"github.com/charmbracelet/huh"
	"github.com/elementumltd/elementum-cli/logger"
	"github.com/elementumltd/elementum-cli/internal/client"
)

// LoginFormData holds the data collected from the login form
type LoginFormData struct {
	Organization     string
	Instance         string // us, eu, stage, dev, custom
	Environment      string // org environment slug (optional)
	CustomBaseURL    string // deprecated: use CustomGraphQLURL and CustomAPIURL
	CustomGraphQLURL string // GraphQL endpoint for custom instance (e.g., http://localhost:3000)
	CustomAPIURL     string // API/OAuth endpoint for custom instance (e.g., http://localhost:8080)
	ClientID         string
	ClientSecret     string
	ProfileName      string
}

// ShowLoginForm displays an interactive form to collect authentication credentials
func ShowLoginForm(defaultProfileName string) (*LoginFormData, error) {
	data := &LoginFormData{
		ProfileName: defaultProfileName,
		Instance:    "us", // Default
	}

	// Step 1: Instance selection
	instanceForm := huh.NewForm(
		huh.NewGroup(
			huh.NewNote().
				Title("Sign in to Elementum").
				Description("Select your Elementum deployment"),

			huh.NewSelect[string]().
				Title("Instance").
				Options(
					huh.NewOption("US (Production)", "us"),
					huh.NewOption("EU", "eu"),
					huh.NewOption("Stage", "stage"),
					huh.NewOption("Dev", "dev"),
					huh.NewOption("Custom (Local/Other)", "custom"),
				).
				Value(&data.Instance),
		),
	)

	if err := instanceForm.Run(); err != nil {
		return nil, err
	}

	// Step 2: Instance-specific credentials form
	var credentialsForm *huh.Form

	if data.Instance == "custom" {
		// Set defaults for custom instance
		data.Organization = "ironman"
		data.CustomGraphQLURL = "http://ironman.local.elementum.io:3000"
		data.CustomAPIURL = "http://ironman.local.elementum.io:8700"

		// Custom instance form - includes org for API headers
		credentialsForm = huh.NewForm(
			huh.NewGroup(
				huh.NewNote().
					Title("Custom Instance").
					Description("Enter your local or custom Elementum endpoints.\nGraphQL is for /graphql calls, API is for /oauth/token auth."),

				huh.NewInput().
					Title("Organization ID").
					Description("Your organization slug (e.g., 'acme')").
					Value(&data.Organization).
					Validate(requiredValidator("organization ID")),

				huh.NewInput().
					Title("GraphQL URL").
					Description("URL for GraphQL endpoint").
					Value(&data.CustomGraphQLURL).
					Validate(requiredValidator("GraphQL URL")),

				huh.NewInput().
					Title("API URL").
					Description("URL for OAuth/API endpoint").
					Value(&data.CustomAPIURL).
					Validate(requiredValidator("API URL")),

				huh.NewInput().
					Title("Client ID").
					Value(&data.ClientID).
					Validate(requiredValidator("client ID")),

				huh.NewInput().
					Title("Client Secret").
					EchoMode(huh.EchoModePassword).
					Value(&data.ClientSecret).
					Validate(requiredValidator("client secret")),

				huh.NewInput().
					Title("Profile Name").
					Description("Name for this configuration").
					Value(&data.ProfileName).
					Validate(requiredValidator("profile name")),
			),
		)
	} else {
		// Standard instance form - org + optional environment
		credentialsForm = huh.NewForm(
			huh.NewGroup(
				huh.NewNote().
					Title(instanceTitle(data.Instance)).
					Description(instanceDescription(data.Instance)),

				huh.NewInput().
					Title("Organization ID").
					Description("Your organization slug (e.g., 'acme')").
					Value(&data.Organization).
					Validate(requiredValidator("organization ID")),

				huh.NewInput().
					Title("Environment (Optional)").
					Description("For multi-env orgs (e.g., 'staging' → staging-acme.elementum.io)").
					Value(&data.Environment),

				huh.NewInput().
					Title("Client ID").
					Value(&data.ClientID).
					Validate(requiredValidator("client ID")),

				huh.NewInput().
					Title("Client Secret").
					EchoMode(huh.EchoModePassword).
					Value(&data.ClientSecret).
					Validate(requiredValidator("client secret")),

				huh.NewInput().
					Title("Profile Name").
					Description("Name for this configuration").
					Value(&data.ProfileName).
					Validate(requiredValidator("profile name")),
			),
		)
	}

	if err := credentialsForm.Run(); err != nil {
		return nil, err
	}

	return data, nil
}

// requiredValidator returns a validation function that checks if a field is non-empty
func requiredValidator(fieldName string) func(string) error {
	return func(s string) error {
		if s == "" {
			return fmt.Errorf("%s is required", fieldName)
		}
		return nil
	}
}

// instanceTitle returns the display title for an instance type
func instanceTitle(instance string) string {
	switch instance {
	case "us":
		return "US Production"
	case "eu":
		return "EU Instance"
	case "stage":
		return "Stage Instance"
	case "dev":
		return "Dev Instance"
	default:
		return "Elementum"
	}
}

// instanceDescription returns the description for an instance type
func instanceDescription(instance string) string {
	switch instance {
	case "us":
		return "Connect to {org}.elementum.io"
	case "eu":
		return "Connect to {org}.eu.elementum.io"
	case "stage":
		return "Connect to {org}.stage.elementum.io"
	case "dev":
		return "Connect to {org}.dev.elementum.io"
	default:
		return "Enter your credentials"
	}
}

// TestCredentials tests if the credentials are valid by making an API call
func TestCredentials(data *LoginFormData) error {
	// Convert instance string to Instance type
	instance := stringToInstance(data.Instance)

	// Create a client with the provided credentials
	c := client.NewClientWithConfig(client.ClientConfig{
		Organization:     data.Organization,
		ClientID:         data.ClientID,
		ClientSecret:     data.ClientSecret,
		Instance:         instance,
		Environment:      data.Environment,
		CustomBaseURL:    data.CustomBaseURL,
		CustomGraphQLURL: data.CustomGraphQLURL,
		CustomAPIURL:     data.CustomAPIURL,
	})

	// Use genqlient GetOrganization to test credentials
	result, err := client.GetOrganization(context.Background(), c.Genqlient())
	if err != nil {
		return fmt.Errorf("authentication failed: %w", err)
	}

	if result.Organization.Id == "" {
		return fmt.Errorf("authentication failed: could not retrieve organization details")
	}

	return nil
}

// SaveLogin saves the login data to the config file and keychain
func SaveLogin(data *LoginFormData) error {
	// Load existing config
	config, err := LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Store the client secret securely
	secretRef, err := StoreSecret(data.ProfileName, data.ClientSecret)
	if err != nil {
		// Log warning but continue - secret will be stored in file
		logger.Warn("could not store secret securely, storing in config file", "error", err)
	}

	// Create the profile
	profile := &Profile{
		Organization:     data.Organization,
		Instance:         data.Instance,
		Environment:      data.Environment,
		CustomBaseURL:    data.CustomBaseURL,
		CustomGraphQLURL: data.CustomGraphQLURL,
		CustomAPIURL:     data.CustomAPIURL,
		ClientID:         data.ClientID,
		ClientSecret:     secretRef,
	}

	// Add the profile to config
	SetProfile(config, data.ProfileName, profile)

	// Set as current profile
	config.CurrentProfile = data.ProfileName

	// Save config
	if err := SaveConfig(config); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	return nil
}
