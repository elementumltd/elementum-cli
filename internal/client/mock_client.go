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
	"fmt"
	"sync"
)

// MockResponse represents a pre-configured response for a GraphQL operation.
type MockResponse struct {
	Data  interface{}
	Error error
}

// MockClient is a mock implementation of ClientInterface for testing.
type MockClient struct {
	mu sync.RWMutex
	// Responses maps query substrings to mock responses.
	// Use SetResponse to add responses.
	responses map[string]MockResponse
	// Calls records all Execute/ExecuteInto calls for verification.
	Calls []MockCall
	// DefaultResponse is returned when no matching response is found.
	DefaultResponse *MockResponse
}

// MockCall records a single call to Execute or ExecuteInto.
type MockCall struct {
	Query     string
	Variables map[string]interface{}
}

// NewMockClient creates a new MockClient.
func NewMockClient() *MockClient {
	return &MockClient{
		responses: make(map[string]MockResponse),
		Calls:     []MockCall{},
	}
}

// SetResponse sets a mock response for queries containing the given substring.
func (m *MockClient) SetResponse(querySubstring string, data interface{}, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.responses[querySubstring] = MockResponse{Data: data, Error: err}
}

// SetDefaultResponse sets the default response when no match is found.
func (m *MockClient) SetDefaultResponse(data interface{}, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.DefaultResponse = &MockResponse{Data: data, Error: err}
}

// ResetCalls clears recorded calls.
func (m *MockClient) ResetCalls() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Calls = []MockCall{}
}

// GetCalls returns a copy of all recorded calls.
func (m *MockClient) GetCalls() []MockCall {
	m.mu.RLock()
	defer m.mu.RUnlock()
	calls := make([]MockCall, len(m.Calls))
	copy(calls, m.Calls)
	return calls
}

// Execute implements ClientInterface.
func (m *MockClient) Execute(ctx context.Context, query string, variables map[string]interface{}) (json.RawMessage, error) {
	m.mu.Lock()
	m.Calls = append(m.Calls, MockCall{Query: query, Variables: variables})
	m.mu.Unlock()

	m.mu.RLock()
	defer m.mu.RUnlock()

	// Find matching response
	for substring, resp := range m.responses {
		if containsSubstring(query, substring) {
			if resp.Error != nil {
				return nil, resp.Error
			}
			jsonData, err := json.Marshal(resp.Data)
			if err != nil {
				return nil, fmt.Errorf("mock: failed to marshal response: %w", err)
			}
			return jsonData, nil
		}
	}

	// Use default response if set
	if m.DefaultResponse != nil {
		if m.DefaultResponse.Error != nil {
			return nil, m.DefaultResponse.Error
		}
		jsonData, err := json.Marshal(m.DefaultResponse.Data)
		if err != nil {
			return nil, fmt.Errorf("mock: failed to marshal default response: %w", err)
		}
		return jsonData, nil
	}

	return nil, fmt.Errorf("mock: no response configured for query: %s", truncateQuery(query))
}

// ExecuteInto implements ClientInterface.
func (m *MockClient) ExecuteInto(ctx context.Context, query string, variables map[string]interface{}, result interface{}) error {
	data, err := m.Execute(ctx, query, variables)
	if err != nil {
		return err
	}

	if err := json.Unmarshal(data, result); err != nil {
		return fmt.Errorf("mock: failed to unmarshal result: %w", err)
	}

	return nil
}

// Genqlient returns a GenqlientClient wrapper for the mock client.
// This allows the mock client to be used with genqlient-generated functions.
func (m *MockClient) Genqlient() *GenqlientClient {
	return &GenqlientClient{client: m.toClient()}
}

// toClient creates a minimal Client that delegates to the MockClient.
// This is needed because GenqlientClient expects a *Client.
func (m *MockClient) toClient() *Client {
	return &Client{
		mockDelegate: m,
	}
}

// ToRealClient returns a *Client that delegates to this MockClient.
// Use this when you need to pass a *Client to functions that expect it.
func (m *MockClient) ToRealClient() *Client {
	return m.toClient()
}

// Ensure MockClient implements ClientInterface.
var _ ClientInterface = (*MockClient)(nil)

// Helper functions

func containsSubstring(s, substr string) bool {
	return len(s) >= len(substr) && findSubstring(s, substr) >= 0
}

func findSubstring(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

func truncateQuery(query string) string {
	if len(query) > 100 {
		return query[:100] + "..."
	}
	return query
}
