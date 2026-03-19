// Copyright (c) ElementumAI, Inc.
// SPDX-License-Identifier: MPL-2.0

package auth

import (
	"fmt"
	"os"
	"time"

	"github.com/elementumltd/elementum-cli/internal/client"
	"github.com/spf13/cobra"
)

// Credentials represents the authentication credentials needed to create a client
type Credentials struct {
	Organization     string
	Instance         string // us, eu, stage, dev, custom
	Environment      string // org environment slug (optional)
	CustomBaseURL    string // deprecated: use CustomGraphQLURL and CustomAPIURL
	CustomGraphQLURL string // GraphQL endpoint for custom instance (e.g., http://localhost:3000)
	CustomAPIURL     string // API/OAuth endpoint for custom instance (e.g., http://localhost:8080)
	ClientID         string
	ClientSecret     string
}

// GetCredentials retrieves credentials using the priority order:
// 1. Command-line flags
// 2. Environment variables
// 3. Config file (current profile)
func GetCredentials(cmd *cobra.Command, config *Config) (*Credentials, error) {
	creds := &Credentials{}

	// Try command-line flags first
	if org, _ := cmd.Flags().GetString("org"); org != "" {
		creds.Organization = org
	}
	if clientID, _ := cmd.Flags().GetString("client-id"); clientID != "" {
		creds.ClientID = clientID
	}
	if secret, _ := cmd.Flags().GetString("client-secret"); secret != "" {
		creds.ClientSecret = secret
	}
	if instance, _ := cmd.Flags().GetString("instance"); instance != "" {
		creds.Instance = instance
	}
	if env, _ := cmd.Flags().GetString("environment"); env != "" {
		creds.Environment = env
	}
	if customURL, _ := cmd.Flags().GetString("custom-base-url"); customURL != "" {
		creds.CustomBaseURL = customURL
	}
	if graphqlURL, _ := cmd.Flags().GetString("custom-graphql-url"); graphqlURL != "" {
		creds.CustomGraphQLURL = graphqlURL
	}
	if apiURL, _ := cmd.Flags().GetString("custom-api-url"); apiURL != "" {
		creds.CustomAPIURL = apiURL
	}

	// Check environment variables for any missing values
	if creds.Organization == "" {
		creds.Organization = os.Getenv("ELEMENTUM_ORGANIZATION")
	}
	if creds.ClientID == "" {
		creds.ClientID = os.Getenv("ELEMENTUM_CLIENT_ID")
	}
	if creds.ClientSecret == "" {
		creds.ClientSecret = os.Getenv("ELEMENTUM_CLIENT_SECRET")
	}
	if creds.Instance == "" {
		creds.Instance = os.Getenv("ELEMENTUM_INSTANCE")
	}
	if creds.Environment == "" {
		creds.Environment = os.Getenv("ELEMENTUM_ENVIRONMENT")
	}
	if creds.CustomBaseURL == "" {
		creds.CustomBaseURL = os.Getenv("ELEMENTUM_CUSTOM_BASE_URL")
	}
	if creds.CustomGraphQLURL == "" {
		creds.CustomGraphQLURL = os.Getenv("ELEMENTUM_CUSTOM_GRAPHQL_URL")
	}
	if creds.CustomAPIURL == "" {
		creds.CustomAPIURL = os.Getenv("ELEMENTUM_CUSTOM_API_URL")
	}

	// Check config file for any still-missing values
	if config != nil {
		// Get the profile name (from flag or current)
		// Check both local flags and root's persistent flags (for commands with DisableFlagParsing)
		profileName, _ := cmd.Flags().GetString("profile")
		if profileName == "" {
			profileName, _ = cmd.Root().PersistentFlags().GetString("profile")
		}
		if profileName == "" {
			profileName = config.CurrentProfile
		}

		profile, exists := config.Profiles[profileName]
		if exists {
			if creds.Organization == "" {
				creds.Organization = profile.Organization
			}
			if creds.ClientID == "" {
				creds.ClientID = profile.ClientID
			}
			if creds.ClientSecret == "" {
				// Retrieve from keychain if needed
				secret, err := GetSecret(profileName, profile.ClientSecret)
				if err != nil {
					return nil, fmt.Errorf("failed to retrieve client secret: %w", err)
				}
				creds.ClientSecret = secret
			}
			if creds.Instance == "" {
				creds.Instance = profile.Instance
			}
			if creds.Environment == "" {
				creds.Environment = profile.Environment
			}
			if creds.CustomBaseURL == "" {
				creds.CustomBaseURL = profile.CustomBaseURL
			}
			if creds.CustomGraphQLURL == "" {
				creds.CustomGraphQLURL = profile.CustomGraphQLURL
			}
			if creds.CustomAPIURL == "" {
				creds.CustomAPIURL = profile.CustomAPIURL
			}
		}
	}

	// Validate that we have all required credentials
	if creds.Organization == "" {
		return nil, fmt.Errorf("organization not set (use --org, ELEMENTUM_ORGANIZATION env var, or run 'ei auth login')")
	}
	if creds.ClientID == "" {
		return nil, fmt.Errorf("client ID not set (use --client-id, ELEMENTUM_CLIENT_ID env var, or run 'ei auth login')")
	}
	if creds.ClientSecret == "" {
		return nil, fmt.Errorf("client secret not set (use --client-secret, ELEMENTUM_CLIENT_SECRET env var, or run 'ei auth login')")
	}
	if creds.Instance == "" {
		// Default to US if not specified
		creds.Instance = "us"
	}
	// Validate instance
	if !IsValidInstance(creds.Instance) {
		return nil, fmt.Errorf("invalid instance %q (valid values: us, eu, stage, dev, custom)", creds.Instance)
	}
	// Custom instance requires either the legacy base URL or the new separate URLs
	if creds.Instance == "custom" {
		hasLegacyURL := creds.CustomBaseURL != ""
		hasNewURLs := creds.CustomGraphQLURL != "" && creds.CustomAPIURL != ""
		if !hasLegacyURL && !hasNewURLs {
			return nil, fmt.Errorf("custom instance requires either --custom-base-url or both --custom-graphql-url and --custom-api-url")
		}
	}

	return creds, nil
}

// NewClient creates an authenticated Elementum API client
func NewClient(creds *Credentials) (*client.Client, error) {
	if creds == nil {
		return nil, fmt.Errorf("credentials are nil")
	}

	// Convert instance string to Instance type
	instance := stringToInstance(creds.Instance)

	// Create client using the new config-based approach
	c := client.NewClientWithConfig(client.ClientConfig{
		Organization:     creds.Organization,
		ClientID:         creds.ClientID,
		ClientSecret:     creds.ClientSecret,
		Instance:         instance,
		Environment:      creds.Environment,
		Timeout:          60 * time.Second,
		CustomBaseURL:    creds.CustomBaseURL,
		CustomGraphQLURL: creds.CustomGraphQLURL,
		CustomAPIURL:     creds.CustomAPIURL,
	})

	return c, nil
}

// stringToInstance converts a string to client.Instance
func stringToInstance(inst string) client.Instance {
	switch inst {
	case "us", "production", "":
		return client.US
	case "eu":
		return client.EU
	case "stage", "staging":
		return client.Stage
	case "dev", "development":
		return client.Dev
	case "custom":
		return client.Custom
	default:
		return client.US
	}
}

// GetClientFromCmd is a convenience function to get a client from command flags + config
func GetClientFromCmd(cmd *cobra.Command) (*client.Client, error) {
	// Load config
	config, err := LoadConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	// Get credentials
	creds, err := GetCredentials(cmd, config)
	if err != nil {
		return nil, err
	}

	// Create client
	return NewClient(creds)
}
