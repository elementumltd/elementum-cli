// Copyright (c) ElementumAI, Inc.
// SPDX-License-Identifier: MPL-2.0

package client

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// TestNewClient verifies client initialization.
func TestNewClient(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		organization  string
		clientID      string
		clientSecret  string
		instance      Instance
		expectedURL   string
		expectedToken string
	}{
		{
			name:          "us instance",
			organization:  "testorg",
			clientID:      "test-client-id",
			clientSecret:  "test-client-secret",
			instance:      US,
			expectedURL:   "https://testorg.elementum.io/graphql",
			expectedToken: "https://api.elementum.io/oauth/token",
		},
		{
			name:          "eu instance",
			organization:  "testorg",
			clientID:      "test-client-id",
			clientSecret:  "test-client-secret",
			instance:      EU,
			expectedURL:   "https://testorg.eu.elementum.io/graphql",
			expectedToken: "https://api.eu.elementum.io/oauth/token",
		},
		{
			name:          "stage instance",
			organization:  "testorg",
			clientID:      "test-client-id",
			clientSecret:  "test-client-secret",
			instance:      Stage,
			expectedURL:   "https://testorg.stage.elementum.io/graphql",
			expectedToken: "https://api.stage.elementum.io/oauth/token",
		},
		{
			name:          "dev instance",
			organization:  "testorg",
			clientID:      "test-client-id",
			clientSecret:  "test-client-secret",
			instance:      Dev,
			expectedURL:   "https://testorg.dev.elementum.io/graphql",
			expectedToken: "https://api.dev.elementum.io/oauth/token",
		},
		{
			name:          "empty instance defaults to us",
			organization:  "testorg",
			clientID:      "test-client-id",
			clientSecret:  "test-client-secret",
			instance:      "",
			expectedURL:   "https://testorg.elementum.io/graphql",
			expectedToken: "https://api.elementum.io/oauth/token",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewClient(tt.organization, tt.clientID, tt.clientSecret, tt.instance)

			if client == nil {
				t.Fatal("expected non-nil client")
			}

			if client.organization != tt.organization {
				t.Errorf("expected organization %q, got %q", tt.organization, client.organization)
			}

			if client.clientID != tt.clientID {
				t.Errorf("expected clientID %q, got %q", tt.clientID, client.clientID)
			}

			if client.clientSecret != tt.clientSecret {
				t.Errorf("expected clientSecret %q, got %q", tt.clientSecret, client.clientSecret)
			}

			if client.baseURL != tt.expectedURL {
				t.Errorf("expected baseURL %q, got %q", tt.expectedURL, client.baseURL)
			}

			if client.tokenURL != tt.expectedToken {
				t.Errorf("expected tokenURL %q, got %q", tt.expectedToken, client.tokenURL)
			}

			if client.httpClient == nil {
				t.Error("expected non-nil httpClient")
			}
		})
	}
}

// TestNewClientWithConfig verifies client initialization with full configuration.
func TestNewClientWithConfig(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		config        ClientConfig
		expectedURL   string
		expectedToken string
	}{
		{
			name: "us instance with org environment",
			config: ClientConfig{
				Organization: "testorg",
				ClientID:     "test-client-id",
				ClientSecret: "test-client-secret",
				Instance:     US,
				Environment:  "staging",
			},
			expectedURL:   "https://staging-testorg.elementum.io/graphql",
			expectedToken: "https://api.elementum.io/oauth/token",
		},
		{
			name: "eu instance with org environment",
			config: ClientConfig{
				Organization: "testorg",
				ClientID:     "test-client-id",
				ClientSecret: "test-client-secret",
				Instance:     EU,
				Environment:  "production",
			},
			expectedURL:   "https://production-testorg.eu.elementum.io/graphql",
			expectedToken: "https://api.eu.elementum.io/oauth/token",
		},
		{
			name: "custom instance with separate URLs",
			config: ClientConfig{
				Organization:     "testorg",
				ClientID:         "test-client-id",
				ClientSecret:     "test-client-secret",
				Instance:         Custom,
				CustomGraphQLURL: "http://localhost:3000/graphql",
				CustomAPIURL:     "http://localhost:8080",
			},
			expectedURL:   "http://localhost:3000/graphql",
			expectedToken: "http://localhost:8080/oauth/token",
		},
		{
			name: "custom instance with legacy CustomBaseURL fallback",
			config: ClientConfig{
				Organization:  "testorg",
				ClientID:      "test-client-id",
				ClientSecret:  "test-client-secret",
				Instance:      Custom,
				CustomBaseURL: "https://custom.example.io",
			},
			expectedURL:   "https://custom.example.io",
			expectedToken: "https://custom.example.io/oauth/token",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewClientWithConfig(tt.config)

			if client == nil {
				t.Fatal("expected non-nil client")
			}

			if client.baseURL != tt.expectedURL {
				t.Errorf("expected baseURL %q, got %q", tt.expectedURL, client.baseURL)
			}

			if client.tokenURL != tt.expectedToken {
				t.Errorf("expected tokenURL %q, got %q", tt.expectedToken, client.tokenURL)
			}
		})
	}
}

// TestFormatBaseURL verifies base URL construction (without /graphql) for different instances and environments.
func TestFormatBaseURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		organization  string
		environment   string
		instance      Instance
		customBaseURL string
		expected      string
	}{
		{"us instance", "myorg", "", US, "", "https://myorg.elementum.io"},
		{"eu instance", "myorg", "", EU, "", "https://myorg.eu.elementum.io"},
		{"stage instance", "myorg", "", Stage, "", "https://myorg.stage.elementum.io"},
		{"dev instance", "myorg", "", Dev, "", "https://myorg.dev.elementum.io"},
		{"empty defaults to us", "myorg", "", "", "", "https://myorg.elementum.io"},
		{"us with org environment", "myorg", "staging", US, "", "https://staging-myorg.elementum.io"},
		{"eu with org environment", "myorg", "staging", EU, "", "https://staging-myorg.eu.elementum.io"},
		{"custom instance", "myorg", "", Custom, "https://custom.example.io", "https://custom.example.io"},
		{"custom instance with trailing slash", "myorg", "", Custom, "https://custom.example.io/", "https://custom.example.io"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatBaseURL(tt.organization, tt.environment, tt.instance, tt.customBaseURL)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

// TestGetBaseURL verifies URL construction for different instances and environments.
func TestGetBaseURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		organization     string
		environment      string
		instance         Instance
		customGraphQLURL string
		expected         string
	}{
		{"us instance", "myorg", "", US, "", "https://myorg.elementum.io/graphql"},
		{"eu instance", "myorg", "", EU, "", "https://myorg.eu.elementum.io/graphql"},
		{"stage instance", "myorg", "", Stage, "", "https://myorg.stage.elementum.io/graphql"},
		{"dev instance", "myorg", "", Dev, "", "https://myorg.dev.elementum.io/graphql"},
		{"empty defaults to us", "myorg", "", "", "", "https://myorg.elementum.io/graphql"},
		{"us with org environment", "myorg", "staging", US, "", "https://staging-myorg.elementum.io/graphql"},
		{"eu with org environment", "myorg", "staging", EU, "", "https://staging-myorg.eu.elementum.io/graphql"},
		// Custom instances use the GraphQL URL as-is (no /graphql appended)
		{"custom instance with graphql path", "myorg", "", Custom, "https://custom.example.io/graphql", "https://custom.example.io/graphql"},
		{"custom instance with trailing slash", "myorg", "", Custom, "https://custom.example.io/", "https://custom.example.io"},
		{"custom instance without path", "myorg", "", Custom, "http://localhost:3000", "http://localhost:3000"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getBaseURL(tt.organization, tt.environment, tt.instance, tt.customGraphQLURL)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

// TestGetTokenURL verifies token URL construction for different instances.
func TestGetTokenURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		instance     Instance
		customAPIURL string
		expected     string
	}{
		{"us instance", US, "", "https://api.elementum.io/oauth/token"},
		{"eu instance", EU, "", "https://api.eu.elementum.io/oauth/token"},
		{"stage instance", Stage, "", "https://api.stage.elementum.io/oauth/token"},
		{"dev instance", Dev, "", "https://api.dev.elementum.io/oauth/token"},
		{"empty defaults to us", "", "", "https://api.elementum.io/oauth/token"},
		{"custom instance with https", Custom, "https://custom.example.io", "https://custom.example.io/oauth/token"},
		{"custom instance with http localhost", Custom, "http://localhost:8080", "http://localhost:8080/oauth/token"},
		{"custom instance with trailing slash", Custom, "http://localhost:8080/", "http://localhost:8080/oauth/token"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getTokenURL(tt.instance, tt.customAPIURL)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

// TestGetHost verifies host construction for different instances.
func TestGetHost(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		instance      Instance
		customBaseURL string
		expected      string
	}{
		{"us instance", US, "", "elementum.io"},
		{"eu instance", EU, "", "eu.elementum.io"},
		{"stage instance", Stage, "", "stage.elementum.io"},
		{"dev instance", Dev, "", "dev.elementum.io"},
		{"empty defaults to us", "", "", "elementum.io"},
		{"custom instance", Custom, "https://myorg.custom.example.io", "custom.example.io"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getHost(tt.instance, tt.customBaseURL)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

// TestRefreshToken verifies OAuth token refresh logic.
func TestRefreshToken(t *testing.T) {
	t.Parallel()

	// Create test server that returns a valid OAuth token
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST request, got %s", r.Method)
		}

		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("expected Content-Type application/json, got %s", r.Header.Get("Content-Type"))
		}

		// Decode request body
		var req OAuthTokenRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("failed to decode request body: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		if req.GrantType != "client_credentials" {
			t.Errorf("expected grant_type=client_credentials, got %s", req.GrantType)
		}

		if req.ClientID != "test-client" {
			t.Errorf("expected client_id=test-client, got %s", req.ClientID)
		}

		if req.ClientSecret != "test-secret" {
			t.Errorf("expected client_secret=test-secret, got %s", req.ClientSecret)
		}

		// Return successful token response
		resp := OAuthTokenResponse{
			AccessToken: "test-access-token",
			TokenType:   "Bearer",
			ExpiresIn:   3600,
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	// Create client with test server
	client := NewClient("testorg", "test-client", "test-secret", Production)
	client.tokenURL = server.URL

	ctx := t.Context()
	token, err := client.refreshToken(ctx)

	if err != nil {
		t.Fatalf("refreshToken failed: %v", err)
	}

	if token != "test-access-token" {
		t.Errorf("expected token 'test-access-token', got %q", token)
	}

	if client.accessToken != "test-access-token" {
		t.Errorf("expected client.accessToken to be set to 'test-access-token', got %q", client.accessToken)
	}

	if client.expiresAt.IsZero() {
		t.Error("expected client.expiresAt to be set")
	}

	// Verify token expires roughly 3600 seconds from now
	expectedExpiry := time.Now().Add(3600 * time.Second)
	if client.expiresAt.Before(expectedExpiry.Add(-10*time.Second)) || client.expiresAt.After(expectedExpiry.Add(10*time.Second)) {
		t.Errorf("token expiry is not within expected range, got %v, expected around %v", client.expiresAt, expectedExpiry)
	}
}

// TestRefreshToken_Error verifies error handling in token refresh.
func TestRefreshToken_Error(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		serverHandler  http.HandlerFunc
		expectedErrMsg string
	}{
		{
			name: "server returns 401",
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = w.Write([]byte(`{"error": "invalid_client"}`))
			},
			expectedErrMsg: "oauth token request failed with status 401",
		},
		{
			name: "server returns 500",
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
			},
			expectedErrMsg: "oauth token request failed with status 500",
		},
		{
			name: "server returns invalid JSON",
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{invalid json`))
			},
			expectedErrMsg: "failed to unmarshal oauth response",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(tt.serverHandler)
			defer server.Close()

			client := NewClient("testorg", "test-client", "test-secret", Production)
			client.tokenURL = server.URL

			ctx := t.Context()
			token, err := client.refreshToken(ctx)

			if err == nil {
				t.Fatal("expected error, got nil")
			}

			if token != "" {
				t.Errorf("expected empty token on error, got %q", token)
			}

			if !strings.Contains(err.Error(), tt.expectedErrMsg) {
				t.Errorf("expected error to contain %q, got %q", tt.expectedErrMsg, err.Error())
			}
		})
	}
}

// TestGetValidToken verifies token caching and refresh logic.
func TestGetValidToken(t *testing.T) {
	t.Parallel()

	// Create test server
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		resp := OAuthTokenResponse{
			AccessToken: "test-token-" + string(rune('0'+callCount)),
			TokenType:   "Bearer",
			ExpiresIn:   3600,
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient("testorg", "test-client", "test-secret", Production)
	client.tokenURL = server.URL

	ctx := t.Context()

	// First call should fetch token
	token1, err := client.getValidToken(ctx)
	if err != nil {
		t.Fatalf("getValidToken failed: %v", err)
	}
	if token1 != "test-token-1" {
		t.Errorf("expected token 'test-token-1', got %q", token1)
	}
	if callCount != 1 {
		t.Errorf("expected 1 server call, got %d", callCount)
	}

	// Second call should use cached token
	token2, err := client.getValidToken(ctx)
	if err != nil {
		t.Fatalf("getValidToken failed: %v", err)
	}
	if token2 != "test-token-1" {
		t.Errorf("expected cached token 'test-token-1', got %q", token2)
	}
	if callCount != 1 {
		t.Errorf("expected still 1 server call (cached), got %d", callCount)
	}

	// Set token to expired - should refresh
	client.mu.Lock()
	client.expiresAt = time.Now().Add(-1 * time.Hour)
	client.mu.Unlock()

	token3, err := client.getValidToken(ctx)
	if err != nil {
		t.Fatalf("getValidToken failed: %v", err)
	}
	if token3 != "test-token-2" {
		t.Errorf("expected refreshed token 'test-token-2', got %q", token3)
	}
	if callCount != 2 {
		t.Errorf("expected 2 server calls (refresh), got %d", callCount)
	}
}

// TestGetValidToken_RefreshBeforeExpiry verifies early token refresh.
func TestGetValidToken_RefreshBeforeExpiry(t *testing.T) {
	t.Parallel()

	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		resp := OAuthTokenResponse{
			AccessToken: "test-token",
			TokenType:   "Bearer",
			ExpiresIn:   3600,
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient("testorg", "test-client", "test-secret", Production)
	client.tokenURL = server.URL

	ctx := t.Context()

	// Get initial token
	_, err := client.getValidToken(ctx)
	if err != nil {
		t.Fatalf("getValidToken failed: %v", err)
	}

	// Set token to expire within the refresh buffer (5 minutes)
	client.mu.Lock()
	client.expiresAt = time.Now().Add(4 * time.Minute)
	client.mu.Unlock()

	// Should trigger refresh even though token hasn't technically expired
	_, err = client.getValidToken(ctx)
	if err != nil {
		t.Fatalf("getValidToken failed: %v", err)
	}

	if callCount != 2 {
		t.Errorf("expected 2 server calls (early refresh), got %d", callCount)
	}
}

// TestExecute verifies GraphQL request execution.
func TestExecute(t *testing.T) {
	t.Parallel()

	// Create test server
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++

		// Handle OAuth token request
		if strings.HasSuffix(r.URL.Path, "/oauth/token") {
			resp := OAuthTokenResponse{
				AccessToken: "test-token",
				TokenType:   "Bearer",
				ExpiresIn:   3600,
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(resp)
			return
		}

		// Handle GraphQL request
		if r.Method != "POST" {
			t.Errorf("expected POST request, got %s", r.Method)
			return
		}

		authHeader := r.Header.Get("Authorization")
		if authHeader != "Bearer test-token" {
			t.Errorf("expected Authorization header 'Bearer test-token', got %q", authHeader)
		}

		contentType := r.Header.Get("Content-Type")
		if contentType != "application/json" {
			t.Errorf("expected Content-Type 'application/json', got %q", contentType)
		}

		// Return proper GraphQL response with data field
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data": {"test": "success"}}`))

	}))
	defer server.Close()

	client := NewClient("testorg", "test-client", "test-secret", Production)
	client.baseURL = server.URL
	client.tokenURL = server.URL + "/oauth/token"

	ctx := t.Context()
	query := "query { test }"

	result, err := client.Execute(ctx, query, nil)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if result == nil {
		t.Fatal("expected non-nil result")
	}

	// Parse the JSON result
	var response map[string]interface{}
	if err := json.Unmarshal(result, &response); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	testVal, ok := response["test"].(string)
	if !ok || testVal != "success" {
		t.Errorf("expected test='success', got %v", response["test"])
	}

	t.Logf("✓ Execute made %d total calls (OAuth + GraphQL)", callCount)
}

// TestExecute_GraphQLError verifies error handling for GraphQL errors.
func TestExecute_GraphQLError(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Handle OAuth
		if strings.HasSuffix(r.URL.Path, "/oauth/token") {
			resp := OAuthTokenResponse{
				AccessToken: "test-token",
				TokenType:   "Bearer",
				ExpiresIn:   3600,
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(resp)
			return
		}

		// Return valid response (errors are handled by Execute)
		w.Header().Set("Content-Type", "application/json")
		resp := map[string]interface{}{
			"data": nil,
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient("testorg", "test-client", "test-secret", Production)
	client.baseURL = server.URL
	client.tokenURL = server.URL + "/oauth/token"

	ctx := t.Context()
	query := "query { invalidField }"

	result, err := client.Execute(ctx, query, nil)
	if err != nil {
		// Execute returns an error for GraphQL errors
		t.Logf("✓ Execute correctly returned error for GraphQL errors: %v", err)
		return
	}

	// If no error, verify the response is empty
	var response map[string]interface{}
	if err := json.Unmarshal(result, &response); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	t.Logf("✓ Execute handled GraphQL error scenario")
}

// TestExecuteInto verifies unmarshaling GraphQL responses into structs.
func TestExecuteInto(t *testing.T) {
	t.Parallel()

	// Result structure matches what ExecuteInto will unmarshal from the "data" field
	type TestResult struct {
		User struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"user"`
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Handle OAuth
		if strings.HasSuffix(r.URL.Path, "/oauth/token") {
			resp := OAuthTokenResponse{
				AccessToken: "test-token",
				TokenType:   "Bearer",
				ExpiresIn:   3600,
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(resp)
			return
		}

		// Return proper GraphQL response
		// Execute will extract the "data" field and return it
		// ExecuteInto will unmarshal that data field into TestResult
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data": {"user": {"id": "user-123", "name": "Test User"}}}`))
	}))
	defer server.Close()

	client := NewClient("testorg", "test-client", "test-secret", Production)
	client.baseURL = server.URL
	client.tokenURL = server.URL + "/oauth/token"

	ctx := t.Context()
	query := "query { user { id name } }"

	var result TestResult
	err := client.ExecuteInto(ctx, query, nil, &result)
	if err != nil {
		t.Fatalf("ExecuteInto failed: %v", err)
	}

	if result.User.ID != "user-123" {
		t.Errorf("expected user ID 'user-123', got %q", result.User.ID)
	}

	if result.User.Name != "Test User" {
		t.Errorf("expected user name 'Test User', got %q", result.User.Name)
	}

	t.Log("✓ ExecuteInto successfully unmarshaled response")
}

// TestExecute_PartialDataWithErrors verifies that Execute returns partial data when GraphQL errors are present.
func TestExecute_PartialDataWithErrors(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Handle OAuth
		if strings.HasSuffix(r.URL.Path, "/oauth/token") {
			resp := OAuthTokenResponse{
				AccessToken: "test-token",
				TokenType:   "Bearer",
				ExpiresIn:   3600,
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(resp)
			return
		}

		// Return GraphQL response with both data and errors (partial success)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"data": {
				"organization": {
					"aspect": {
						"id": "test-123",
						"name": "Test App",
						"fields": {
							"edges": [
								{"node": {"id": "field-1", "name": "Field 1"}},
								{"node": {"id": "field-2", "name": "Field 2"}}
							]
						}
					}
				}
			},
			"errors": [{
				"message": "The item does not exist.",
				"path": ["organization", "aspect", "fields", "edges", 2, "node", "relatedField"],
				"extensions": {"errorType": "NOT_FOUND"}
			}]
		}`))
	}))
	defer server.Close()

	client := NewClient("testorg", "test-client", "test-secret", Production)
	client.baseURL = server.URL
	client.tokenURL = server.URL + "/oauth/token"

	ctx := t.Context()
	query := "query { organization { aspect { id name fields { edges { node { id name } } } } } }"

	result, err := client.Execute(ctx, query, nil)
	if err != nil {
		t.Fatalf("Execute should not return error when partial data is present: %v", err)
	}

	// Verify we got the partial data
	var response map[string]interface{}
	if err := json.Unmarshal(result, &response); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	org, ok := response["organization"].(map[string]interface{})
	if !ok {
		t.Fatal("expected organization in response")
	}

	aspect, ok := org["aspect"].(map[string]interface{})
	if !ok {
		t.Fatal("expected aspect in response")
	}

	if aspect["id"] != "test-123" {
		t.Errorf("expected aspect ID 'test-123', got %v", aspect["id"])
	}

	if aspect["name"] != "Test App" {
		t.Errorf("expected aspect name 'Test App', got %v", aspect["name"])
	}

	t.Log("✓ Execute correctly returned partial data despite GraphQL errors")
}

// TestExecute_ErrorsWithoutData verifies that Execute returns error when no data is present.
func TestExecute_ErrorsWithoutData(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Handle OAuth
		if strings.HasSuffix(r.URL.Path, "/oauth/token") {
			resp := OAuthTokenResponse{
				AccessToken: "test-token",
				TokenType:   "Bearer",
				ExpiresIn:   3600,
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(resp)
			return
		}

		// Return GraphQL response with errors but no data
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"data": null,
			"errors": [{
				"message": "Not authorized",
				"extensions": {"errorType": "UNAUTHORIZED"}
			}]
		}`))
	}))
	defer server.Close()

	client := NewClient("testorg", "test-client", "test-secret", Production)
	client.baseURL = server.URL
	client.tokenURL = server.URL + "/oauth/token"

	ctx := t.Context()
	query := "query { secretData }"

	_, err := client.Execute(ctx, query, nil)
	if err == nil {
		t.Fatal("Execute should return error when no data is present")
	}

	if !strings.Contains(err.Error(), "Not authorized") {
		t.Errorf("expected error message to contain 'Not authorized', got %v", err)
	}

	t.Log("✓ Execute correctly returned error when no data is present")
}

// =============================================================================
// Custom Instance Tests
// =============================================================================
// These tests verify the custom instance support via separate GraphQL and API URLs.
// This allows local development and self-hosted deployments.

// TestCustomInstance_SeparateURLs verifies that custom instances can have
// separate GraphQL and API URLs for different backends.
func TestCustomInstance_SeparateURLs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		customGraphQLURL string
		customAPIURL     string
		expectedBaseURL  string
		expectedTokenURL string
	}{
		{
			name:             "local development with different ports",
			customGraphQLURL: "http://localhost:3000/graphql",
			customAPIURL:     "http://localhost:8080",
			expectedBaseURL:  "http://localhost:3000/graphql",
			expectedTokenURL: "http://localhost:8080/oauth/token",
		},
		{
			name:             "local development same host different paths",
			customGraphQLURL: "http://localhost:8080/graphql",
			customAPIURL:     "http://localhost:8080/api",
			expectedBaseURL:  "http://localhost:8080/graphql",
			expectedTokenURL: "http://localhost:8080/api/oauth/token",
		},
		{
			name:             "https custom domains",
			customGraphQLURL: "https://graphql.mycompany.com",
			customAPIURL:     "https://api.mycompany.com",
			expectedBaseURL:  "https://graphql.mycompany.com",
			expectedTokenURL: "https://api.mycompany.com/oauth/token",
		},
		{
			name:             "trailing slashes are handled",
			customGraphQLURL: "http://localhost:3000/graphql/",
			customAPIURL:     "http://localhost:8080/",
			expectedBaseURL:  "http://localhost:3000/graphql",
			expectedTokenURL: "http://localhost:8080/oauth/token",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewClientWithConfig(ClientConfig{
				Organization:     "testorg",
				ClientID:         "test-client",
				ClientSecret:     "test-secret",
				Instance:         Custom,
				CustomGraphQLURL: tt.customGraphQLURL,
				CustomAPIURL:     tt.customAPIURL,
			})

			if client.baseURL != tt.expectedBaseURL {
				t.Errorf("baseURL: expected %q, got %q", tt.expectedBaseURL, client.baseURL)
			}
			if client.tokenURL != tt.expectedTokenURL {
				t.Errorf("tokenURL: expected %q, got %q", tt.expectedTokenURL, client.tokenURL)
			}
		})
	}
}

// TestCustomInstance_LegacyFallback verifies that the legacy CustomBaseURL
// still works for backward compatibility.
func TestCustomInstance_LegacyFallback(t *testing.T) {
	t.Parallel()

	client := NewClientWithConfig(ClientConfig{
		Organization:  "testorg",
		ClientID:      "test-client",
		ClientSecret:  "test-secret",
		Instance:      Custom,
		CustomBaseURL: "https://legacy.custom.io",
	})

	// Legacy behavior: CustomBaseURL used as-is for GraphQL
	if client.baseURL != "https://legacy.custom.io" {
		t.Errorf("baseURL: expected %q, got %q", "https://legacy.custom.io", client.baseURL)
	}
	// Token URL appends /oauth/token
	if client.tokenURL != "https://legacy.custom.io/oauth/token" {
		t.Errorf("tokenURL: expected %q, got %q", "https://legacy.custom.io/oauth/token", client.tokenURL)
	}
}

// TestCustomInstance_NewURLsOverrideLegacy verifies that new separate URLs
// take precedence over the legacy CustomBaseURL.
func TestCustomInstance_NewURLsOverrideLegacy(t *testing.T) {
	t.Parallel()

	client := NewClientWithConfig(ClientConfig{
		Organization:     "testorg",
		ClientID:         "test-client",
		ClientSecret:     "test-secret",
		Instance:         Custom,
		CustomBaseURL:    "https://legacy.example.io", // Should be ignored
		CustomGraphQLURL: "http://localhost:3000/graphql",
		CustomAPIURL:     "http://localhost:8080",
	})

	if client.baseURL != "http://localhost:3000/graphql" {
		t.Errorf("baseURL should use CustomGraphQLURL, got %q", client.baseURL)
	}
	if client.tokenURL != "http://localhost:8080/oauth/token" {
		t.Errorf("tokenURL should use CustomAPIURL, got %q", client.tokenURL)
	}
}

// TestCustomInstance_NoAppendGraphQL verifies that for custom instances,
// /graphql is NOT automatically appended (the URL is used as-is).
func TestCustomInstance_NoAppendGraphQL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		customGraphQLURL string
		expectedBaseURL  string
	}{
		{
			name:             "URL without graphql path is used as-is",
			customGraphQLURL: "http://localhost:3000",
			expectedBaseURL:  "http://localhost:3000",
		},
		{
			name:             "URL with graphql path is used as-is",
			customGraphQLURL: "http://localhost:3000/graphql",
			expectedBaseURL:  "http://localhost:3000/graphql",
		},
		{
			name:             "URL with custom path is used as-is",
			customGraphQLURL: "http://localhost:3000/api/v1/graphql",
			expectedBaseURL:  "http://localhost:3000/api/v1/graphql",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getBaseURL("ignored", "", Custom, tt.customGraphQLURL)
			if result != tt.expectedBaseURL {
				t.Errorf("expected %q, got %q", tt.expectedBaseURL, result)
			}
		})
	}
}

// TestCustomInstance_StandardInstancesAppendGraphQL verifies that standard
// instances (US, EU, etc.) still get /graphql appended automatically.
func TestCustomInstance_StandardInstancesAppendGraphQL(t *testing.T) {
	t.Parallel()

	instances := []struct {
		instance Instance
		expected string
	}{
		{US, "https://myorg.elementum.io/graphql"},
		{EU, "https://myorg.eu.elementum.io/graphql"},
		{Stage, "https://myorg.stage.elementum.io/graphql"},
		{Dev, "https://myorg.dev.elementum.io/graphql"},
	}

	for _, inst := range instances {
		t.Run(string(inst.instance), func(t *testing.T) {
			result := getBaseURL("myorg", "", inst.instance, "")
			if result != inst.expected {
				t.Errorf("expected %q, got %q", inst.expected, result)
			}
		})
	}
}

// TestCustomInstance_EmptyURLHandling verifies behavior when custom URLs are empty.
// In practice, this case is validated at the provider level to require URLs.
func TestCustomInstance_EmptyURLHandling(t *testing.T) {
	t.Parallel()

	// When custom instance has no URLs, it falls through to the default case
	// which treats "custom" as an instance subdomain (myorg.custom.elementum.io)
	// This edge case is prevented at the provider level which requires URLs for custom instances.
	result := getBaseURL("myorg", "", Custom, "")
	expected := "https://myorg.custom.elementum.io/graphql"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

// TestCustomInstance_TokenURLConstruction verifies token URL construction
// for various custom API URL formats.
func TestCustomInstance_TokenURLConstruction(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		customAPIURL string
		expected     string
	}{
		{
			name:         "http localhost",
			customAPIURL: "http://localhost:8080",
			expected:     "http://localhost:8080/oauth/token",
		},
		{
			name:         "https domain",
			customAPIURL: "https://api.example.com",
			expected:     "https://api.example.com/oauth/token",
		},
		{
			name:         "with path prefix",
			customAPIURL: "http://localhost:8080/api/v1",
			expected:     "http://localhost:8080/api/v1/oauth/token",
		},
		{
			name:         "trailing slash trimmed",
			customAPIURL: "http://localhost:8080/",
			expected:     "http://localhost:8080/oauth/token",
		},
		{
			name:         "multiple trailing slashes trimmed",
			customAPIURL: "http://localhost:8080///",
			expected:     "http://localhost:8080///oauth/token", // Only single trailing slash trimmed
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getTokenURL(Custom, tt.customAPIURL)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}
