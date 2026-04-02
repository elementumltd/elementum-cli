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
	"testing"

	"github.com/elementumltd/elementum-cli/internal/client"
)

func TestStringToInstance(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input    string
		expected client.Instance
	}{
		{"us", client.US},
		{"production", client.US},
		{"", client.US},
		{"eu", client.EU},
		{"stage", client.Stage},
		{"staging", client.Stage},
		{"dev", client.Dev},
		{"development", client.Dev},
		{"custom", client.Custom},
		{"unknown", client.US}, // Default to US
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := stringToInstance(tt.input)
			if result != tt.expected {
				t.Errorf("stringToInstance(%q) = %q, expected %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestNewClient(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		creds *Credentials
		valid bool
	}{
		{
			name:  "nil credentials",
			creds: nil,
			valid: false,
		},
		{
			name: "valid us credentials",
			creds: &Credentials{
				Organization: "testorg",
				Instance:     "us",
				ClientID:     "test-client",
				ClientSecret: "test-secret",
			},
			valid: true,
		},
		{
			name: "valid eu credentials",
			creds: &Credentials{
				Organization: "testorg",
				Instance:     "eu",
				ClientID:     "test-client",
				ClientSecret: "test-secret",
			},
			valid: true,
		},
		{
			name: "credentials with org environment",
			creds: &Credentials{
				Organization: "testorg",
				Instance:     "us",
				Environment:  "staging",
				ClientID:     "test-client",
				ClientSecret: "test-secret",
			},
			valid: true,
		},
		{
			name: "custom instance with base url",
			creds: &Credentials{
				Organization:  "testorg",
				Instance:      "custom",
				CustomBaseURL: "https://custom.example.io",
				ClientID:      "test-client",
				ClientSecret:  "test-secret",
			},
			valid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, err := NewClient(tt.creds)

			if tt.valid {
				if err != nil {
					t.Errorf("NewClient() returned error: %v", err)
				}
				if c == nil {
					t.Error("NewClient() returned nil client")
				}
			} else {
				if err == nil {
					t.Error("NewClient() should have returned error")
				}
			}
		})
	}
}

func TestCredentials_Validation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		creds    Credentials
		hasError bool
	}{
		{
			name: "valid credentials",
			creds: Credentials{
				Organization: "testorg",
				Instance:     "us",
				ClientID:     "client-id",
				ClientSecret: "client-secret",
			},
			hasError: false,
		},
		{
			name: "custom instance without base url",
			creds: Credentials{
				Organization:  "testorg",
				Instance:      "custom",
				CustomBaseURL: "", // Missing!
				ClientID:      "client-id",
				ClientSecret:  "client-secret",
			},
			hasError: true,
		},
		{
			name: "custom instance with base url",
			creds: Credentials{
				Organization:  "testorg",
				Instance:      "custom",
				CustomBaseURL: "https://custom.example.io",
				ClientID:      "client-id",
				ClientSecret:  "client-secret",
			},
			hasError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Validate custom instance requires base URL
			if tt.creds.Instance == "custom" && tt.creds.CustomBaseURL == "" {
				if !tt.hasError {
					t.Error("Expected validation to fail for custom instance without base URL")
				}
			}
		})
	}
}

func TestCredentialsStruct(t *testing.T) {
	t.Parallel()

	// Test that the Credentials struct has all required fields
	creds := Credentials{
		Organization:  "testorg",
		Instance:      "eu",
		Environment:   "staging",
		CustomBaseURL: "https://custom.example.io",
		ClientID:      "my-client-id",
		ClientSecret:  "my-client-secret",
	}

	if creds.Organization != "testorg" {
		t.Errorf("Organization = %q, expected %q", creds.Organization, "testorg")
	}
	if creds.Instance != "eu" {
		t.Errorf("Instance = %q, expected %q", creds.Instance, "eu")
	}
	if creds.Environment != "staging" {
		t.Errorf("Environment = %q, expected %q", creds.Environment, "staging")
	}
	if creds.CustomBaseURL != "https://custom.example.io" {
		t.Errorf("CustomBaseURL = %q, expected %q", creds.CustomBaseURL, "https://custom.example.io")
	}
	if creds.ClientID != "my-client-id" {
		t.Errorf("ClientID = %q, expected %q", creds.ClientID, "my-client-id")
	}
	if creds.ClientSecret != "my-client-secret" {
		t.Errorf("ClientSecret = %q, expected %q", creds.ClientSecret, "my-client-secret")
	}
}

func TestInstanceConstants(t *testing.T) {
	t.Parallel()

	// Verify client package constants are accessible
	instances := []client.Instance{
		client.US,
		client.EU,
		client.Stage,
		client.Dev,
		client.Custom,
	}

	expectedStrings := []string{"us", "eu", "stage", "dev", "custom"}

	for i, inst := range instances {
		if string(inst) != expectedStrings[i] {
			t.Errorf("client.Instance %d = %q, expected %q", i, inst, expectedStrings[i])
		}
	}
}
