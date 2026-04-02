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
