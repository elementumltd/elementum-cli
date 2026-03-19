// Copyright (c) ElementumAI, Inc.
// SPDX-License-Identifier: MPL-2.0

package client

import (
	"context"
	"encoding/json"
)

// ClientInterface defines the interface for Elementum API clients.
// This allows for mocking in tests.
type ClientInterface interface {
	// Execute executes a GraphQL query and returns the raw response data.
	Execute(ctx context.Context, query string, variables map[string]interface{}) (json.RawMessage, error)

	// ExecuteInto executes a GraphQL query and unmarshals the response into the provided struct.
	ExecuteInto(ctx context.Context, query string, variables map[string]interface{}, result interface{}) error

	// Genqlient returns a genqlient-compatible client wrapper for type-safe GraphQL operations.
	Genqlient() *GenqlientClient
}

// Ensure Client implements ClientInterface.
var _ ClientInterface = (*Client)(nil)
