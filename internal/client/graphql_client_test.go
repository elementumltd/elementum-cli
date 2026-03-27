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
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Khan/genqlient/graphql"
)

func TestBuildEqualsFilter(t *testing.T) {
	t.Parallel()

	field := "name"
	value := "Test App"
	raw := BuildEqualsFilter(field, value)

	if raw == nil {
		t.Fatal("expected non-nil result")
	}

	var filter map[string]interface{}
	if err := json.Unmarshal(*raw, &filter); err != nil {
		t.Fatalf("failed to unmarshal filter: %v", err)
	}

	if filter["field"] != field {
		t.Errorf("expected field %q, got %v", field, filter["field"])
	}
	if filter["type"] != "EQUALS" {
		t.Errorf("expected type EQUALS, got %v", filter["type"])
	}

	valMap, ok := filter["value"].(map[string]interface{})
	if !ok {
		t.Fatal("expected value to be a map")
	}
	if valMap["type"] != "TEXT" {
		t.Errorf("expected value type TEXT, got %v", valMap["type"])
	}
	if valMap["value"] != value {
		t.Errorf("expected value %q, got %v", value, valMap["value"])
	}
}

func TestBuildSearchFilter(t *testing.T) {
	t.Parallel()

	query := "search query"
	raw := BuildSearchFilter(query)

	if raw == nil {
		t.Fatal("expected non-nil result")
	}

	var filter map[string]interface{}
	if err := json.Unmarshal(*raw, &filter); err != nil {
		t.Fatalf("failed to unmarshal filter: %v", err)
	}

	if filter["field"] != "name" {
		t.Errorf("expected field 'name', got %v", filter["field"])
	}
	if filter["type"] != "LIKE" {
		t.Errorf("expected type LIKE, got %v", filter["type"])
	}
	if filter["mode"] != "CONTAINS" {
		t.Errorf("expected mode CONTAINS, got %v", filter["mode"])
	}
	if filter["caseInsensitive"] != true {
		t.Errorf("expected caseInsensitive true, got %v", filter["caseInsensitive"])
	}

	valMap, ok := filter["value"].(map[string]interface{})
	if !ok {
		t.Fatal("expected value to be a map")
	}
	if valMap["type"] != "TEXT" {
		t.Errorf("expected value type TEXT, got %v", valMap["type"])
	}
	if valMap["value"] != query {
		t.Errorf("expected value %q, got %v", query, valMap["value"])
	}
}

func TestBuildTableNameFilter(t *testing.T) {
	t.Parallel()

	name := "MyTable"
	raw := BuildTableNameFilter(name)

	if raw == nil {
		t.Fatal("expected non-nil result")
	}

	var filter map[string]interface{}
	if err := json.Unmarshal(*raw, &filter); err != nil {
		t.Fatalf("failed to unmarshal filter: %v", err)
	}

	equals, ok := filter["equals"].(map[string]interface{})
	if !ok {
		t.Fatal("expected equals to be a map")
	}
	if equals["field"] != "name" {
		t.Errorf("expected field 'name', got %v", equals["field"])
	}
	if equals["value"] != name {
		t.Errorf("expected value %q, got %v", name, equals["value"])
	}
}

func TestBuildNotDisabledFilter(t *testing.T) {
	t.Parallel()

	raw := BuildNotDisabledFilter()

	if raw == nil {
		t.Fatal("expected non-nil result")
	}

	var filter map[string]interface{}
	if err := json.Unmarshal(*raw, &filter); err != nil {
		t.Fatalf("failed to unmarshal filter: %v", err)
	}

	// Verify filter structure: { type: "EQUALS", field: "status", value: { type: "TEXT", value: "DISABLED" }, not: true }
	if filter["type"] != "EQUALS" {
		t.Errorf("expected type EQUALS, got %v", filter["type"])
	}
	if filter["field"] != "status" {
		t.Errorf("expected field 'status', got %v", filter["field"])
	}
	if filter["not"] != true {
		t.Errorf("expected not=true, got %v", filter["not"])
	}

	valMap, ok := filter["value"].(map[string]interface{})
	if !ok {
		t.Fatal("expected value to be a map")
	}
	if valMap["type"] != "TEXT" {
		t.Errorf("expected value type TEXT, got %v", valMap["type"])
	}
	if valMap["value"] != "DISABLED" {
		t.Errorf("expected value 'DISABLED', got %v", valMap["value"])
	}
}

func TestGenqlientClient_MakeRequest_WithMock(t *testing.T) {
	t.Parallel()

	mock := NewMockClient()
	client := mock.ToRealClient()
	genqlient := client.Genqlient()

	ctx := t.Context()
	opName := "GetSomething"
	expectedData := map[string]interface{}{
		"something": map[string]interface{}{
			"id":   "123",
			"name": "Test",
		},
	}

	mock.SetResponse(opName, expectedData, nil)

	req := &graphql.Request{
		OpName: opName,
		Query:  "query GetSomething { something { id name } }",
	}
	var resp graphql.Response

	err := genqlient.MakeRequest(ctx, req, &resp)
	if err != nil {
		t.Fatalf("MakeRequest failed: %v", err)
	}

	if resp.Errors != nil {
		t.Fatalf("expected no errors, got %v", resp.Errors)
	}

	// Unmarshal resp.Data into a map to verify
	dataBytes, err := json.Marshal(resp.Data)
	if err != nil {
		t.Fatalf("failed to marshal response data: %v", err)
	}
	var actualData map[string]interface{}
	if err := json.Unmarshal(dataBytes, &actualData); err != nil {
		t.Fatalf("failed to unmarshal response data: %v", err)
	}

	something, ok := actualData["something"].(map[string]interface{})
	if !ok {
		t.Fatal("expected something to be a map")
	}
	if something["id"] != "123" {
		t.Errorf("expected id '123', got %v", something["id"])
	}
}

func TestGenqlientClient_MakeRequest_Variables(t *testing.T) {
	t.Parallel()

	mock := NewMockClient()
	client := mock.ToRealClient()
	genqlient := client.Genqlient()

	ctx := t.Context()
	opName := "GetWithVars"
	variables := map[string]interface{}{
		"id": "abc-123",
	}
	expectedData := map[string]interface{}{
		"result": "success",
	}

	mock.SetResponse(opName, expectedData, nil)

	req := &graphql.Request{
		OpName:    opName,
		Query:     "query GetWithVars($id: ID!) { result(id: $id) }",
		Variables: variables,
	}
	var resp graphql.Response

	err := genqlient.MakeRequest(ctx, req, &resp)
	if err != nil {
		t.Fatalf("MakeRequest failed: %v", err)
	}

	// Verify the mock recorded the correct variables
	calls := mock.GetCalls()
	if len(calls) != 1 {
		t.Fatalf("expected 1 call, got %d", len(calls))
	}
	if calls[0].Variables["id"] != "abc-123" {
		t.Errorf("expected variable id 'abc-123', got %v", calls[0].Variables["id"])
	}
}

func TestGenqlientClient_MakeRequest_ComplexVariables(t *testing.T) {
	t.Parallel()

	mock := NewMockClient()
	client := mock.ToRealClient()
	genqlient := client.Genqlient()

	ctx := t.Context()
	opName := "ComplexVars"

	type Input struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}
	variables := Input{Name: "John", Age: 30}
	expectedData := map[string]interface{}{"ok": true}

	mock.SetResponse(opName, expectedData, nil)

	req := &graphql.Request{
		OpName:    opName,
		Variables: variables,
	}
	var resp graphql.Response

	err := genqlient.MakeRequest(ctx, req, &resp)
	if err != nil {
		t.Fatalf("MakeRequest failed: %v", err)
	}

	calls := mock.GetCalls()
	if len(calls) != 1 {
		t.Fatalf("expected 1 call, got %d", len(calls))
	}
	vars := calls[0].Variables
	age, ok := vars["age"].(float64)
	if !ok {
		t.Fatal("expected age to be a float64")
	}
	if vars["name"] != "John" || int(age) != 30 {
		t.Errorf("unexpected variables: %v", vars)
	}
}

func TestGenqlientClient_MakeRequest_MockError(t *testing.T) {
	t.Parallel()

	mock := NewMockClient()
	client := mock.ToRealClient()
	genqlient := client.Genqlient()

	ctx := t.Context()
	opName := "ErrorOp"

	// Using DefaultResponse or SetResponse to return an error
	// The mock client currently returns an error directly if resp.Error is set
	mock.SetResponse(opName, nil, fmt.Errorf("Something went wrong"))

	req := &graphql.Request{
		OpName: opName,
	}
	var resp graphql.Response

	err := genqlient.MakeRequest(ctx, req, &resp)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !strings.Contains(err.Error(), "Something went wrong") {
		t.Errorf("expected error to contain 'Something went wrong', got %v", err)
	}
}

func TestGenqlientClient_MakeRequest_RealHTTP(t *testing.T) {
	t.Parallel()

	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

		// Verify headers
		authHeader := r.Header.Get("Authorization")
		if authHeader != "Bearer test-token" {
			t.Errorf("expected Authorization header 'Bearer test-token', got %q", authHeader)
		}

		orgHeader := r.Header.Get("x-elementum-organization")
		if !strings.Contains(orgHeader, "testorg") {
			t.Errorf("expected x-elementum-organization header to contain 'testorg', got %q", orgHeader)
		}

		// Return proper GraphQL response
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data": {"test": "http-success"}}`))
	}))
	defer server.Close()

	client := NewClient("testorg", "test-client", "test-secret", Production)
	client.baseURL = server.URL
	client.tokenURL = server.URL + "/oauth/token"

	genqlient := client.Genqlient()
	ctx := t.Context()

	req := &graphql.Request{
		OpName: "HTTPTest",
		Query:  "query HTTPTest { test }",
	}
	var resp graphql.Response

	err := genqlient.MakeRequest(ctx, req, &resp)
	if err != nil {
		t.Fatalf("MakeRequest failed: %v", err)
	}

	if resp.Errors != nil {
		t.Fatalf("expected no errors, got %v", resp.Errors)
	}

	dataBytes, _ := json.Marshal(resp.Data)
	var data map[string]interface{}
	_ = json.Unmarshal(dataBytes, &data)

	if data["test"] != "http-success" {
		t.Errorf("expected test='http-success', got %v", data["test"])
	}
}

func TestGenqlientClient_MakeRequest_GraphQLErrors(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Handle OAuth
		if strings.HasSuffix(r.URL.Path, "/oauth/token") {
			resp := OAuthTokenResponse{AccessToken: "token", ExpiresIn: 3600}
			_ = json.NewEncoder(w).Encode(resp)
			return
		}

		// Return GraphQL response with errors and no data
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"data": null,
			"errors": [{"message": "GraphQL error message"}]
		}`))
	}))
	defer server.Close()

	client := NewClient("testorg", "id", "secret", Production)
	client.baseURL = server.URL
	client.tokenURL = server.URL + "/oauth/token"

	genqlient := client.Genqlient()
	req := &graphql.Request{OpName: "ErrorTest"}
	var resp graphql.Response

	err := genqlient.MakeRequest(t.Context(), req, &resp)
	if err == nil {
		t.Fatal("expected error for GraphQL error without data, got nil")
	}

	if !strings.Contains(err.Error(), "GraphQL error message") {
		t.Errorf("expected error to contain message, got %v", err)
	}
}

func TestGenqlientClient_MakeRequest_PartialData(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Handle OAuth
		if strings.HasSuffix(r.URL.Path, "/oauth/token") {
			resp := OAuthTokenResponse{AccessToken: "token", ExpiresIn: 3600}
			_ = json.NewEncoder(w).Encode(resp)
			return
		}

		// Return GraphQL response with both data and errors
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"data": {"partial": "data"},
			"errors": [{"message": "Some warning"}]
		}`))
	}))
	defer server.Close()

	client := NewClient("testorg", "id", "secret", Production)
	client.baseURL = server.URL
	client.tokenURL = server.URL + "/oauth/token"

	genqlient := client.Genqlient()
	req := &graphql.Request{OpName: "PartialTest"}
	var resp graphql.Response

	// Should return nil error if data is present, despite GraphQL errors
	err := genqlient.MakeRequest(t.Context(), req, &resp)
	if err != nil {
		t.Fatalf("expected nil error for partial data, got %v", err)
	}

	if len(resp.Errors) != 1 {
		t.Errorf("expected 1 GraphQL error in response, got %d", len(resp.Errors))
	}
}

func TestStripNullValues(t *testing.T) {
	t.Parallel()

	input := map[string]interface{}{
		"a": 1,
		"b": nil,
		"c": map[string]interface{}{
			"d": nil,
			"e": "val",
		},
		"f": []interface{}{
			nil,
			map[string]interface{}{"g": nil},
		},
	}

	expected := map[string]interface{}{
		"a": 1,
		"c": map[string]interface{}{
			"e": "val",
		},
		"f": []interface{}{
			nil,
			map[string]interface{}{},
		},
	}

	result := stripNullValues(input)

	resJSON, _ := json.Marshal(result)
	expJSON, _ := json.Marshal(expected)

	if string(resJSON) != string(expJSON) {
		t.Errorf("expected %s, got %s", string(expJSON), string(resJSON))
	}
}

func TestGenqlientClient_MakeRequest_ConvertVariables(t *testing.T) {
	t.Parallel()

	mock := NewMockClient()
	client := mock.ToRealClient()
	genqlient := client.Genqlient()

	type Input struct {
		Name string  `json:"name"`
		Null *string `json:"null"`
	}
	variables := Input{Name: "test"}
	mock.SetResponse("Op", map[string]interface{}{"ok": true}, nil)

	req := &graphql.Request{
		OpName:    "Op",
		Variables: variables,
	}
	var resp graphql.Response

	err := genqlient.MakeRequest(t.Context(), req, &resp)
	if err != nil {
		t.Fatalf("MakeRequest failed: %v", err)
	}

	calls := mock.GetCalls()
	if len(calls) != 1 {
		t.Fatal("expected 1 call")
	}
	if calls[0].Variables["name"] != "test" {
		t.Errorf("expected name 'test', got %v", calls[0].Variables["name"])
	}
	if _, ok := calls[0].Variables["null"]; ok {
		t.Error("expected 'null' field to be stripped")
	}
}

func TestGenqlientClient_MakeRequest_HTTPError(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/oauth/token") {
			resp := OAuthTokenResponse{AccessToken: "token", ExpiresIn: 3600}
			_ = json.NewEncoder(w).Encode(resp)
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("Internal Error"))
	}))
	defer server.Close()

	client := NewClient("org", "id", "secret", Production)
	client.baseURL = server.URL
	client.tokenURL = server.URL + "/oauth/token"

	genqlient := client.Genqlient()
	req := &graphql.Request{OpName: "Fail"}
	var resp graphql.Response

	err := genqlient.MakeRequest(t.Context(), req, &resp)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	// ExecuteWithRetry wraps the HTTP error when retries are exhausted
	if !strings.Contains(err.Error(), "HTTP 500") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestGenqlientClient_MakeRequest_NoDataWithErrors(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/oauth/token") {
			resp := OAuthTokenResponse{AccessToken: "token", ExpiresIn: 3600}
			_ = json.NewEncoder(w).Encode(resp)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data": null, "errors": [{"message": "failed"}]}`))
	}))
	defer server.Close()

	client := NewClient("org", "id", "secret", Production)
	client.baseURL = server.URL
	client.tokenURL = server.URL + "/oauth/token"

	genqlient := client.Genqlient()
	req := &graphql.Request{OpName: "NoData"}
	var resp graphql.Response

	err := genqlient.MakeRequest(t.Context(), req, &resp)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "failed") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestGenqlientClient_MakeRequest_RetriesOnINTERNAL(t *testing.T) {
	t.Parallel()

	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/oauth/token") {
			resp := OAuthTokenResponse{AccessToken: "token", ExpiresIn: 3600}
			_ = json.NewEncoder(w).Encode(resp)
			return
		}
		callCount++
		w.Header().Set("Content-Type", "application/json")
		if callCount < 3 {
			_, _ = w.Write([]byte(`{"data": null, "errors": [{"message": "Something went wrong", "extensions": {"errorType": "INTERNAL"}}]}`))
			return
		}
		_, _ = w.Write([]byte(`{"data": {"result": "ok"}}`))
	}))
	defer server.Close()

	client := NewClient("org", "id", "secret", Production)
	client.baseURL = server.URL
	client.tokenURL = server.URL + "/oauth/token"

	genqlient := client.Genqlient()
	req := &graphql.Request{OpName: "RetryTest", Query: "mutation { delete }"}
	var resp graphql.Response

	err := genqlient.MakeRequest(t.Context(), req, &resp)
	if err != nil {
		t.Fatalf("expected success after retry, got error: %v", err)
	}
	if callCount != 3 {
		t.Errorf("expected 3 GraphQL calls (2 INTERNAL + 1 success), got %d", callCount)
	}
}

func TestGenqlientClient_MakeRequest_NoRetryOnNonTransient(t *testing.T) {
	t.Parallel()

	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/oauth/token") {
			resp := OAuthTokenResponse{AccessToken: "token", ExpiresIn: 3600}
			_ = json.NewEncoder(w).Encode(resp)
			return
		}
		callCount++
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data": null, "errors": [{"message": "Not found", "extensions": {"errorType": "NOT_FOUND"}}]}`))
	}))
	defer server.Close()

	client := NewClient("org", "id", "secret", Production)
	client.baseURL = server.URL
	client.tokenURL = server.URL + "/oauth/token"

	genqlient := client.Genqlient()
	req := &graphql.Request{OpName: "NoRetryTest"}
	var resp graphql.Response

	err := genqlient.MakeRequest(t.Context(), req, &resp)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if callCount != 1 {
		t.Errorf("expected 1 call (no retry for NOT_FOUND), got %d", callCount)
	}
}

func TestGenqlientClient_MakeRequest_RetriesExhausted(t *testing.T) {
	t.Parallel()

	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/oauth/token") {
			resp := OAuthTokenResponse{AccessToken: "token", ExpiresIn: 3600}
			_ = json.NewEncoder(w).Encode(resp)
			return
		}
		callCount++
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprintf(w, `{"data": null, "errors": [{"message": "Something went wrong ref:%d", "extensions": {"errorType": "INTERNAL"}}]}`, callCount)
	}))
	defer server.Close()

	client := NewClient("org", "id", "secret", Production)
	client.baseURL = server.URL
	client.tokenURL = server.URL + "/oauth/token"

	genqlient := client.Genqlient()
	req := &graphql.Request{OpName: "ExhaustedTest"}
	var resp graphql.Response

	err := genqlient.MakeRequest(t.Context(), req, &resp)
	if err == nil {
		t.Fatal("expected error after exhausted retries, got nil")
	}
	expectedCalls := maxRetries + 1
	if callCount != expectedCalls {
		t.Errorf("expected %d calls (all retries exhausted), got %d", expectedCalls, callCount)
	}
	if !strings.Contains(err.Error(), "max retries") {
		t.Errorf("expected 'max retries' in error, got: %v", err)
	}
}
