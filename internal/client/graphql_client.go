// Copyright (c) ElementumAI, Inc.
// SPDX-License-Identifier: MPL-2.0

package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/Khan/genqlient/graphql"
	"github.com/elementumltd/elementum-cli/internal/logging"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// GenqlientClient wraps our Client to implement genqlient's graphql.Client interface.
// This allows the generated type-safe functions to use our existing authentication
// and request handling infrastructure.
type GenqlientClient struct {
	client *Client
}

// Genqlient returns a genqlient-compatible client wrapper.
// Use this to call the generated type-safe GraphQL functions.
//
// Example:
//
//	resp, err := GetUserByID(ctx, c.Genqlient(), userId)
func (c *Client) Genqlient() *GenqlientClient {
	return &GenqlientClient{client: c}
}

// MakeRequest implements the graphql.Client interface for genqlient.
// It delegates to the existing Client's authentication and request handling.
func (g *GenqlientClient) MakeRequest(
	ctx context.Context,
	req *graphql.Request,
	resp *graphql.Response,
) error {
	c := g.client

	// If the client has a mock delegate, use it instead
	if c.mockDelegate != nil {
		return g.makeRequestWithMock(ctx, req, resp)
	}

	// Get a valid access token
	token, err := c.getValidToken(ctx)
	if err != nil {
		return fmt.Errorf("failed to get access token: %w", err)
	}

	// Build the request body - convert Variables to map if needed
	var variables map[string]interface{}
	if req.Variables != nil {
		if v, ok := req.Variables.(map[string]interface{}); ok {
			variables = v
		} else {
			// Marshal and unmarshal to convert to map
			varBytes, err := json.Marshal(req.Variables)
			if err != nil {
				return fmt.Errorf("failed to marshal variables: %w", err)
			}
			if err := json.Unmarshal(varBytes, &variables); err != nil {
				return fmt.Errorf("failed to unmarshal variables: %w", err)
			}
		}
		// Strip null values from variables - the Elementum API rejects explicit nulls
		// for optional fields (e.g., in TableCreateV2Input)
		if stripped, ok := stripNullValues(variables).(map[string]interface{}); ok {
			variables = stripped
		}
	}

	reqBody := GraphQLRequest{
		Query:     req.Query,
		Variables: variables,
		OpName:    req.OpName,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	tflog.Trace(ctx, "GenqlientClient.MakeRequest", map[string]interface{}{
		"request": string(jsonBody),
	})

	host := getHost(c.instance, c.customBaseURL)

	// Build organization part for header, including environment prefix if set
	orgPart := c.organization
	if c.environment != "" {
		orgPart = fmt.Sprintf("%s-%s", c.environment, c.organization)
	}

	var lastErr error

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			if !graphQLRetryBackoff(ctx, attempt-1, req.OpName) {
				return ctx.Err()
			}
		}

		// Execute with HTTP-level retry logic (handles 429, 5xx, network errors)
		httpResp, err := ExecuteWithRetry(ctx, func() (*http.Response, error) {
			httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL, bytes.NewBuffer(jsonBody))
			if err != nil {
				return nil, fmt.Errorf("failed to create request: %w", err)
			}

			httpReq.Header.Set("Content-Type", "application/json")
			httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

			httpReq.Header.Set("x-elementum-organization", fmt.Sprintf("%s.%s", orgPart, host))
			httpReq.Header.Set("x-elementum-platform", "IOS")
			httpReq.Header.Set("x-elementum-toe", uuid.New().String())
			httpReq.Header.Set("x-elementum-base-url", host)

			return c.httpClient.Do(httpReq)
		})
		if err != nil {
			return fmt.Errorf("failed to execute request: %w", err)
		}

		body, err := io.ReadAll(httpResp.Body)
		httpResp.Body.Close()
		if err != nil {
			return fmt.Errorf("failed to read response: %w", err)
		}

		if httpResp.StatusCode != http.StatusOK {
			return fmt.Errorf("unexpected status code %d: %s", httpResp.StatusCode, string(body))
		}

		// Parse into genqlient's response structure
		if err := json.Unmarshal(body, resp); err != nil {
			return fmt.Errorf("failed to unmarshal response: %w", err)
		}

		// Check for GraphQL errors in the response
		// GraphQL returns HTTP 200 even when there are errors, so we must check the response body
		if len(resp.Errors) > 0 {
			for _, gqlErr := range resp.Errors {
				logging.GraphQLWarn(gqlErr.Message, req.OpName, gqlErr.Path, variables)
			}

			if resp.Data != nil {
				return nil
			}

			// Retry on transient server errors (INTERNAL, THROTTLED, etc.)
			if isRetryableGraphQLError(resp.Errors[0]) {
				lastErr = fmt.Errorf("graphql error: %s", resp.Errors[0].Message)
				// Reset resp so the retry gets a clean slate
				*resp = graphql.Response{}
				continue
			}

			return fmt.Errorf("graphql error: %s", resp.Errors[0].Message)
		}

		return nil
	}

	return fmt.Errorf("max retries (%d) exceeded: %w", maxRetries, lastErr)
}

// makeRequestWithMock handles requests when using a MockClient delegate.
func (g *GenqlientClient) makeRequestWithMock(
	ctx context.Context,
	req *graphql.Request,
	resp *graphql.Response,
) error {
	c := g.client

	// Build the request body - convert Variables to map if needed
	var variables map[string]interface{}
	if req.Variables != nil {
		if v, ok := req.Variables.(map[string]interface{}); ok {
			variables = v
		} else {
			// Marshal and unmarshal to convert to map
			varBytes, err := json.Marshal(req.Variables)
			if err != nil {
				return fmt.Errorf("failed to marshal variables: %w", err)
			}
			if err := json.Unmarshal(varBytes, &variables); err != nil {
				return fmt.Errorf("failed to unmarshal variables: %w", err)
			}
		}
		// Strip null values from variables - the Elementum API rejects explicit nulls
		if stripped, ok := stripNullValues(variables).(map[string]interface{}); ok {
			variables = stripped
		}
	}

	// Use the operation name as the query for mock matching
	query := req.OpName
	if query == "" {
		query = req.Query
	}

	// Execute through the mock client
	data, err := c.mockDelegate.Execute(ctx, query, variables)
	if err != nil {
		return err
	}

	// The mock client returns the inner data, but genqlient expects it wrapped in {"data": ...}
	// Wrap the response in the standard GraphQL response format
	wrappedData := map[string]json.RawMessage{
		"data": data,
	}
	wrappedJSON, err := json.Marshal(wrappedData)
	if err != nil {
		return fmt.Errorf("failed to wrap mock response: %w", err)
	}

	// Parse into genqlient's response structure
	if err := json.Unmarshal(wrappedJSON, resp); err != nil {
		return fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return nil
}

// BuildEqualsFilter creates a JSON filter for exact field matching.
// This is a helper to simplify filter creation for common lookup patterns.
func BuildEqualsFilter(field, value string) *json.RawMessage {
	filter := map[string]interface{}{
		"field": field,
		"type":  "EQUALS",
		"value": map[string]interface{}{
			"type":  "TEXT",
			"value": value,
		},
	}
	data, _ := json.Marshal(filter)
	raw := json.RawMessage(data)
	return &raw
}

// BuildSearchFilter creates a JSON filter for case-insensitive name searching.
func BuildSearchFilter(query string) *json.RawMessage {
	filter := map[string]interface{}{
		"field": "name",
		"type":  "LIKE",
		"value": map[string]interface{}{
			"type":  "TEXT",
			"value": query,
		},
		"mode":            "CONTAINS",
		"caseInsensitive": true,
	}
	data, _ := json.Marshal(filter)
	raw := json.RawMessage(data)
	return &raw
}

// BuildTableNameFilter creates a filter for table name matching.
// Tables use a different filter format than other entities.
func BuildTableNameFilter(name string) *json.RawMessage {
	filter := map[string]interface{}{
		"equals": map[string]interface{}{
			"field": "name",
			"value": name,
		},
	}
	data, _ := json.Marshal(filter)
	raw := json.RawMessage(data)
	return &raw
}

// BuildNotDisabledFilter creates a filter to exclude DISABLED automations.
// This filter matches automations where status is NOT equal to "DISABLED".
func BuildNotDisabledFilter() *json.RawMessage {
	filter := map[string]interface{}{
		"type":  "EQUALS",
		"field": "status",
		"value": map[string]interface{}{
			"type":  "TEXT",
			"value": "DISABLED",
		},
		"not": true,
	}
	data, _ := json.Marshal(filter)
	raw := json.RawMessage(data)
	return &raw
}

// BuildUserSearchFilter creates an OR filter for searching users by name or email.
// Uses LIKE with CONTAINS mode for case-insensitive partial matching.
func BuildUserSearchFilter(query string) *json.RawMessage {
	filter := map[string]interface{}{
		"type": "OR",
		"children": []map[string]interface{}{
			{
				"field":           "name",
				"type":            "LIKE",
				"caseInsensitive": true,
				"mode":            "CONTAINS",
				"value":           map[string]interface{}{"type": "TEXT", "value": query},
			},
			{
				"field":           "email",
				"type":            "LIKE",
				"caseInsensitive": true,
				"mode":            "CONTAINS",
				"value":           map[string]interface{}{"type": "TEXT", "value": query},
			},
		},
	}
	data, _ := json.Marshal(filter)
	raw := json.RawMessage(data)
	return &raw
}

// stripNullValues recursively removes null values from maps and slices.
// This is necessary because the Elementum GraphQL API rejects explicit null values
// for optional fields - they must be omitted entirely instead.
func stripNullValues(v interface{}) interface{} {
	switch val := v.(type) {
	case map[string]interface{}:
		result := make(map[string]interface{})
		for k, v := range val {
			if v == nil {
				continue // Skip null values
			}
			stripped := stripNullValues(v)
			if stripped != nil {
				result[k] = stripped
			}
		}
		return result
	case []interface{}:
		result := make([]interface{}, 0, len(val))
		for _, item := range val {
			stripped := stripNullValues(item)
			// For arrays, we keep the element even if it's null (to preserve array structure)
			// but we still strip nulls from nested objects
			result = append(result, stripped)
		}
		return result
	default:
		return v
	}
}
