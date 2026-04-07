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

package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/elementumltd/elementum-cli/internal/logging"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Instance represents the Elementum deployment instance/region.
type Instance string

const (
	US     Instance = "us"     // Production US (naked domain)
	EU     Instance = "eu"     // Production EU
	Stage  Instance = "stage"  // Staging instance
	Dev    Instance = "dev"    // Development instance
	Custom Instance = "custom" // Custom/self-hosted instance
)

// Environment is kept for backward compatibility, maps to Instance.
// Deprecated: Use Instance instead.
type Environment = Instance

// Backward compatibility constants.
const (
	Production Environment = US
)

// tokenRefreshBuffer is how early to refresh tokens before expiry (5 minutes).
const tokenRefreshBuffer = 5 * time.Minute

// Client is the Elementum GraphQL API client.
type Client struct {
	httpClient       *http.Client
	organization     string
	clientID         string
	clientSecret     string
	instance         Instance
	environment      string // org environment slug (e.g., "staging", "production")
	customBaseURL    string // deprecated: use customGraphQLURL and customAPIURL
	customGraphQLURL string // GraphQL endpoint for custom instance
	customAPIURL     string // API/OAuth endpoint for custom instance
	baseURL          string
	tokenURL         string

	// Token management
	mu          sync.RWMutex
	accessToken string
	expiresAt   time.Time

	// mockDelegate is used in testing to delegate calls to a MockClient.
	// When set, the Client acts as a thin wrapper around the MockClient.
	mockDelegate *MockClient
}

// ClientConfig holds configuration for creating a new client.
type ClientConfig struct {
	Organization     string
	ClientID         string
	ClientSecret     string
	Instance         Instance
	Environment      string        // org environment slug (optional)
	Timeout          time.Duration // HTTP client timeout (default: 60s)
	CustomBaseURL    string        // deprecated: use CustomGraphQLURL and CustomAPIURL
	CustomGraphQLURL string        // GraphQL endpoint for custom instance (e.g., http://localhost:3000)
	CustomAPIURL     string        // API/OAuth endpoint for custom instance (e.g., http://localhost:8080)
}

// NewClient creates a new Elementum API client with OAuth client credentials.
// Deprecated: Use NewClientWithConfig for full instance/environment support.
func NewClient(organization, clientID, clientSecret string, instance Instance) *Client {
	return NewClientWithConfig(ClientConfig{
		Organization: organization,
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Instance:     instance,
	})
}

// NewClientWithConfig creates a new Elementum API client with full configuration.
func NewClientWithConfig(cfg ClientConfig) *Client {
	// Default to US if not specified
	if cfg.Instance == "" {
		cfg.Instance = US
	}

	// Default timeout to 60 seconds if not specified
	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = 60 * time.Second
	}

	// Determine the effective GraphQL and API URLs
	// Priority: new separate URLs > legacy single URL
	effectiveGraphQLURL := cfg.CustomGraphQLURL
	effectiveAPIURL := cfg.CustomAPIURL
	if effectiveGraphQLURL == "" && cfg.CustomBaseURL != "" {
		effectiveGraphQLURL = cfg.CustomBaseURL
	}
	if effectiveAPIURL == "" && cfg.CustomBaseURL != "" {
		effectiveAPIURL = cfg.CustomBaseURL
	}

	baseURL := getBaseURL(cfg.Organization, cfg.Environment, cfg.Instance, effectiveGraphQLURL)
	tokenURL := getTokenURL(cfg.Instance, effectiveAPIURL)

	return &Client{
		httpClient:       &http.Client{Timeout: timeout},
		organization:     cfg.Organization,
		clientID:         cfg.ClientID,
		clientSecret:     cfg.ClientSecret,
		instance:         cfg.Instance,
		environment:      cfg.Environment,
		customBaseURL:    cfg.CustomBaseURL,
		customGraphQLURL: cfg.CustomGraphQLURL,
		customAPIURL:     cfg.CustomAPIURL,
		baseURL:          baseURL,
		tokenURL:         tokenURL,
	}
}

// FormatBaseURL constructs the base Elementum URL (without /graphql path).
// URL pattern: https://{env}-{org}.{instance}.elementum.io
// For US instance: https://{env}-{org}.elementum.io
// For custom: customBaseURL (as provided).
// This is exported for use by the CLI and other packages.
func FormatBaseURL(organization, environment string, instance Instance, customBaseURL string) string {
	if instance == Custom && customBaseURL != "" {
		return strings.TrimSuffix(customBaseURL, "/")
	}

	// Build the organization part (org or env-org)
	orgPart := organization
	if environment != "" {
		orgPart = fmt.Sprintf("%s-%s", environment, organization)
	}

	// Build the domain based on instance
	switch instance {
	case US, "production", "": // US is the naked domain
		return fmt.Sprintf("https://%s.elementum.io", orgPart)
	default: // EU, Stage, Dev
		return fmt.Sprintf("https://%s.%s.elementum.io", orgPart, instance)
	}
}

// getBaseURL constructs the GraphQL API URL based on instance and environment.
// For custom instances, the customGraphQLURL is used as-is (no /graphql appended).
// For standard instances, uses FormatBaseURL and appends /graphql.
func getBaseURL(organization, environment string, instance Instance, customGraphQLURL string) string {
	// For custom instances, use the GraphQL URL as-is (no /graphql appended)
	if instance == Custom && customGraphQLURL != "" {
		return strings.TrimSuffix(customGraphQLURL, "/")
	}

	baseURL := FormatBaseURL(organization, environment, instance, "")
	if !strings.HasSuffix(baseURL, "/graphql") {
		baseURL += "/graphql"
	}
	return baseURL
}

// getTokenURL returns the OAuth token endpoint URL for the given instance.
func getTokenURL(instance Instance, customAPIURL string) string {
	if instance == Custom && customAPIURL != "" {
		return strings.TrimSuffix(customAPIURL, "/") + "/oauth/token"
	}

	switch instance {
	case US, "production", "":
		return "https://api.elementum.io/oauth/token"
	default: // EU, Stage, Dev
		return fmt.Sprintf("https://api.%s.elementum.io/oauth/token", instance)
	}
}

// getHost returns the host domain for headers.
func getHost(instance Instance, customBaseURL string) string {
	if instance == Custom && customBaseURL != "" {
		// Extract host from custom URL
		baseURL := strings.TrimPrefix(customBaseURL, "https://")
		baseURL = strings.TrimPrefix(baseURL, "http://")
		// Remove path and org subdomain to get base host
		parts := strings.SplitN(baseURL, "/", 2)
		if len(parts) > 0 {
			hostParts := strings.SplitN(parts[0], ".", 2)
			if len(hostParts) > 1 {
				return hostParts[1] // Return everything after first subdomain
			}
			return parts[0]
		}
		return baseURL
	}

	switch instance {
	case US, "production", "":
		return "elementum.io"
	default:
		return fmt.Sprintf("%s.elementum.io", instance)
	}
}

// OAuthTokenRequest represents the OAuth token request body.
type OAuthTokenRequest struct {
	GrantType    string `json:"grant_type"`
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
}

// OAuthTokenResponse represents the OAuth token response.
type OAuthTokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"` // seconds
}

// getValidToken returns a valid access token, refreshing if necessary.
func (c *Client) getValidToken(ctx context.Context) (string, error) {
	c.mu.RLock()
	token := c.accessToken
	expiresAt := c.expiresAt
	c.mu.RUnlock()

	// Check if token is still valid (with buffer)
	if token != "" && time.Now().Add(tokenRefreshBuffer).Before(expiresAt) {
		return token, nil
	}

	// Need to refresh token
	return c.refreshToken(ctx)
}

// refreshToken fetches a new access token using client credentials.
func (c *Client) refreshToken(ctx context.Context) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Double-check after acquiring lock (another goroutine may have refreshed)
	if c.accessToken != "" && time.Now().Add(tokenRefreshBuffer).Before(c.expiresAt) {
		return c.accessToken, nil
	}

	reqBody := OAuthTokenRequest{
		GrantType:    "client_credentials",
		ClientID:     c.clientID,
		ClientSecret: c.clientSecret,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal oauth request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.tokenURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", fmt.Errorf("failed to create oauth request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to execute oauth request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read oauth response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("oauth token request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var tokenResp OAuthTokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return "", fmt.Errorf("failed to unmarshal oauth response: %w", err)
	}

	if tokenResp.AccessToken == "" {
		return "", fmt.Errorf("oauth response missing access_token")
	}

	// Store the new token
	c.accessToken = tokenResp.AccessToken
	c.expiresAt = time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)

	return c.accessToken, nil
}

// GraphQLRequest represents a GraphQL request.
type GraphQLRequest struct {
	Query     string                 `json:"query"`
	Variables map[string]interface{} `json:"variables,omitempty"`
	OpName    string                 `json:"operationName,omitempty"`
}

// extractOperationName extracts the operation name from a GraphQL query string.
// It looks for patterns like "query OperationName" or "mutation OperationName".
func extractOperationName(query string) string {
	// Match "query <name>(" or "mutation <name>(" with optional whitespace
	re := regexp.MustCompile(`(?:query|mutation)\s+(\w+)`)
	matches := re.FindStringSubmatch(query)
	if len(matches) > 1 {
		return matches[1]
	}
	return "unknown"
}

// GraphQLResponse represents a GraphQL response.
type GraphQLResponse struct {
	Data   json.RawMessage `json:"data"`
	Errors []GraphQLError  `json:"errors,omitempty"`
}

// GraphQLError represents a GraphQL error.
type GraphQLError struct {
	Message    string                 `json:"message"`
	Path       []interface{}          `json:"path,omitempty"`
	Extensions map[string]interface{} `json:"extensions,omitempty"`
}

func (e GraphQLError) Error() string {
	return e.Message
}

// formatGraphQLError formats a GraphQL error with validation details if present.
func formatGraphQLError(gqlErr GraphQLError) error {
	// Check if this is a validation error with field-level details
	if gqlErr.Extensions != nil {
		if errorDetail, ok := gqlErr.Extensions["errorDetail"].(string); ok && errorDetail == "VALIDATION" {
			if errors, ok := gqlErr.Extensions["errors"].(map[string]interface{}); ok {
				// Build a detailed error message with all validation errors
				var details []string
				for field, msgs := range errors {
					if msgList, ok := msgs.([]interface{}); ok {
						for _, msg := range msgList {
							if msgStr, ok := msg.(string); ok {
								details = append(details, fmt.Sprintf("  - %s: %s", field, msgStr))
							}
						}
					}
				}
				if len(details) > 0 {
					return fmt.Errorf("validation error: %s\n%s", gqlErr.Message, strings.Join(details, "\n"))
				}
			}
		}

		// For other errors with extensions, include the errorType if available
		if errorType, ok := gqlErr.Extensions["errorType"].(string); ok {
			return fmt.Errorf("graphql error (%s): %s", errorType, gqlErr.Message)
		}
	}

	// Default error message
	return fmt.Errorf("graphql error: %s", gqlErr.Message)
}

// Execute executes a GraphQL query and returns the raw response data.
func (c *Client) Execute(ctx context.Context, query string, variables map[string]interface{}) (json.RawMessage, error) {
	// Get a valid access token
	token, err := c.getValidToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get access token: %w", err)
	}

	reqBody := GraphQLRequest{
		Query:     query,
		Variables: variables,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	tflog.Trace(ctx, "Client.Execute: GraphQL Request", map[string]interface{}{
		"request": string(jsonBody),
	})

	host := getHost(c.instance, c.customBaseURL)
	opName := extractOperationName(query)

	// Build organization part for header, including environment prefix if set
	orgPart := c.organization
	if c.environment != "" {
		orgPart = fmt.Sprintf("%s-%s", c.environment, c.organization)
	}

	var lastErr error

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			if !graphQLRetryBackoff(ctx, attempt-1, opName) {
				return nil, ctx.Err()
			}
		}

		resp, err := ExecuteWithRetry(ctx, func() (*http.Response, error) {
			req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL, bytes.NewBuffer(jsonBody))
			if err != nil {
				return nil, fmt.Errorf("failed to create request: %w", err)
			}

			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

			req.Header.Set("x-elementum-organization", fmt.Sprintf("%s.%s", orgPart, host))
			req.Header.Set("x-elementum-platform", "IOS")
			req.Header.Set("x-elementum-toe", uuid.New().String())
			req.Header.Set("x-elementum-base-url", host)

			return c.httpClient.Do(req)
		})
		if err != nil {
			return nil, fmt.Errorf("failed to execute request: %w", err)
		}

		body, err := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("failed to read response: %w", err)
		}

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, string(body))
		}

		var gqlResp GraphQLResponse
		if err := json.Unmarshal(body, &gqlResp); err != nil {
			return nil, fmt.Errorf("failed to unmarshal response: %w", err)
		}

		if len(gqlResp.Errors) > 0 {
			for _, gqlErr := range gqlResp.Errors {
				logging.GraphQLWarn(gqlErr.Message, opName, gqlErr.Path, variables)
			}

			if len(gqlResp.Data) > 0 {
				dataStr := strings.TrimSpace(string(gqlResp.Data))
				if dataStr != "" && dataStr != "null" {
					return gqlResp.Data, nil
				}
			}

			// Retry on transient server errors (INTERNAL, THROTTLED, etc.)
			if isRetryableGraphQLErrorLegacy(gqlResp.Errors[0]) {
				lastErr = formatGraphQLError(gqlResp.Errors[0])
				continue
			}

			return nil, formatGraphQLError(gqlResp.Errors[0])
		}

		return gqlResp.Data, nil
	}

	return nil, fmt.Errorf("max retries (%d) exceeded: %w", maxRetries, lastErr)
}

// ExecuteInto executes a GraphQL query and unmarshals the response into the provided struct.
func (c *Client) ExecuteInto(ctx context.Context, query string, variables map[string]interface{}, result interface{}) error {
	tflog.Trace(ctx, "Client.ExecuteInto", map[string]interface{}{
		"query":     query,
		"variables": variables,
	})

	data, err := c.Execute(ctx, query, variables)
	if err != nil {
		return err
	}

	if err := json.Unmarshal(data, result); err != nil {
		return fmt.Errorf("failed to unmarshal result: %w", err)
	}

	return nil
}

// ExecuteWithWarnings executes a GraphQL query and returns both the raw data
// and any GraphQL errors that were returned alongside partial data.
// Unlike Execute, this does NOT swallow errors when partial data is present.
// Callers can inspect the warnings to understand what fields may be null/missing.
func (c *Client) ExecuteWithWarnings(ctx context.Context, query string, variables map[string]interface{}) (json.RawMessage, []GraphQLError, error) {
	token, err := c.getValidToken(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get access token: %w", err)
	}

	reqBody := GraphQLRequest{
		Query:     query,
		Variables: variables,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	host := getHost(c.instance, c.customBaseURL)

	// Build organization part for header, including environment prefix if set
	orgPart := c.organization
	if c.environment != "" {
		orgPart = fmt.Sprintf("%s-%s", c.environment, c.organization)
	}

	resp, err := ExecuteWithRetry(ctx, func() (*http.Response, error) {
		req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL, bytes.NewBuffer(jsonBody))
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
		req.Header.Set("x-elementum-organization", fmt.Sprintf("%s.%s", orgPart, host))
		req.Header.Set("x-elementum-platform", "IOS")
		req.Header.Set("x-elementum-toe", uuid.New().String())
		req.Header.Set("x-elementum-base-url", host)
		return c.httpClient.Do(req)
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, nil, fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, string(body))
	}

	var gqlResp GraphQLResponse
	if err := json.Unmarshal(body, &gqlResp); err != nil {
		return nil, nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	// Log warnings via the logging package (demoted to debug for known non-actionable)
	if len(gqlResp.Errors) > 0 {
		opName := extractOperationName(query)
		for _, gqlErr := range gqlResp.Errors {
			logging.GraphQLWarn(gqlErr.Message, opName, gqlErr.Path, variables)
		}
	}

	// Return partial data if available, plus all errors for the caller to inspect
	if len(gqlResp.Data) > 0 {
		dataStr := strings.TrimSpace(string(gqlResp.Data))
		if dataStr != "" && dataStr != "null" {
			return gqlResp.Data, gqlResp.Errors, nil
		}
	}

	// No data at all
	if len(gqlResp.Errors) > 0 {
		return nil, gqlResp.Errors, formatGraphQLError(gqlResp.Errors[0])
	}

	return gqlResp.Data, nil, nil
}

// ExecuteIntoWithWarnings is like ExecuteInto but also returns GraphQL warnings
// that were returned alongside partial data. This lets callers know which
// specific fields in the result may be null/broken due to server-side errors.
func (c *Client) ExecuteIntoWithWarnings(ctx context.Context, query string, variables map[string]interface{}, result interface{}) ([]GraphQLError, error) {
	data, warnings, err := c.ExecuteWithWarnings(ctx, query, variables)
	if err != nil {
		return warnings, err
	}

	if err := json.Unmarshal(data, result); err != nil {
		return warnings, fmt.Errorf("failed to unmarshal result: %w", err)
	}

	return warnings, nil
}

// GetRESTBaseURL returns the base URL for REST API calls (without /graphql suffix).
// This is used for endpoints like /api/v1/agent-conversations.
func (c *Client) GetRESTBaseURL() string {
	// Remove /graphql suffix if present
	return strings.TrimSuffix(c.baseURL, "/graphql")
}

// GetAccessToken returns a valid access token, refreshing if necessary.
// This is exported for use by components that need direct API access (like SSE streaming).
func (c *Client) GetAccessToken(ctx context.Context) (string, error) {
	return c.getValidToken(ctx)
}

// GetOrganization returns the organization identifier for this client.
func (c *Client) GetOrganization() string {
	return c.organization
}

// GetInstance returns the instance type for this client.
func (c *Client) GetInstance() Instance {
	return c.instance
}
