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
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestIsValidInstance(t *testing.T) {
	t.Parallel()

	tests := []struct {
		instance string
		expected bool
	}{
		{"us", true},
		{"eu", true},
		{"stage", true},
		{"dev", true},
		{"custom", true},
		{"production", false}, // Old value, not valid
		{"staging", false},    // Old value, not valid
		{"invalid", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.instance, func(t *testing.T) {
			result := IsValidInstance(tt.instance)
			if result != tt.expected {
				t.Errorf("IsValidInstance(%q) = %v, expected %v", tt.instance, result, tt.expected)
			}
		})
	}
}

func TestMigrateProfiles(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		input            *Config
		expectedMigrated bool
		expectedInstance string
		expectedEnv      string
	}{
		{
			name: "migrate production to us",
			input: &Config{
				Profiles: map[string]*Profile{
					"default": {
						Organization: "testorg",
						Instance:     "",
						Environment:  "production",
					},
				},
			},
			expectedMigrated: true,
			expectedInstance: "us",
			expectedEnv:      "",
		},
		{
			name: "migrate stage to stage",
			input: &Config{
				Profiles: map[string]*Profile{
					"default": {
						Organization: "testorg",
						Instance:     "",
						Environment:  "stage",
					},
				},
			},
			expectedMigrated: true,
			expectedInstance: "stage",
			expectedEnv:      "",
		},
		{
			name: "migrate staging to stage",
			input: &Config{
				Profiles: map[string]*Profile{
					"default": {
						Organization: "testorg",
						Instance:     "",
						Environment:  "staging",
					},
				},
			},
			expectedMigrated: true,
			expectedInstance: "stage",
			expectedEnv:      "",
		},
		{
			name: "migrate dev to dev",
			input: &Config{
				Profiles: map[string]*Profile{
					"default": {
						Organization: "testorg",
						Instance:     "",
						Environment:  "dev",
					},
				},
			},
			expectedMigrated: true,
			expectedInstance: "dev",
			expectedEnv:      "",
		},
		{
			name: "migrate development to dev",
			input: &Config{
				Profiles: map[string]*Profile{
					"default": {
						Organization: "testorg",
						Instance:     "",
						Environment:  "development",
					},
				},
			},
			expectedMigrated: true,
			expectedInstance: "dev",
			expectedEnv:      "",
		},
		{
			name: "migrate empty to us",
			input: &Config{
				Profiles: map[string]*Profile{
					"default": {
						Organization: "testorg",
						Instance:     "",
						Environment:  "",
					},
				},
			},
			expectedMigrated: true,
			expectedInstance: "us",
			expectedEnv:      "",
		},
		{
			name: "keep custom env slug",
			input: &Config{
				Profiles: map[string]*Profile{
					"default": {
						Organization: "testorg",
						Instance:     "",
						Environment:  "my-custom-env",
					},
				},
			},
			expectedMigrated: true,
			expectedInstance: "us",
			expectedEnv:      "my-custom-env",
		},
		{
			name: "no migration needed",
			input: &Config{
				Profiles: map[string]*Profile{
					"default": {
						Organization: "testorg",
						Instance:     "eu",
						Environment:  "staging",
					},
				},
			},
			expectedMigrated: false,
			expectedInstance: "eu",
			expectedEnv:      "staging",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			migrated := migrateProfiles(tt.input)

			if migrated != tt.expectedMigrated {
				t.Errorf("migrateProfiles() returned %v, expected %v", migrated, tt.expectedMigrated)
			}

			profile := tt.input.Profiles["default"]
			if profile.Instance != tt.expectedInstance {
				t.Errorf("Instance = %q, expected %q", profile.Instance, tt.expectedInstance)
			}
			if profile.Environment != tt.expectedEnv {
				t.Errorf("Environment = %q, expected %q", profile.Environment, tt.expectedEnv)
			}
		})
	}
}

func TestSetProfile(t *testing.T) {
	t.Parallel()

	config := &Config{
		Profiles: make(map[string]*Profile),
	}

	profile := &Profile{
		Organization: "testorg",
		Instance:     "us",
		ClientID:     "test-client",
		ClientSecret: "test-secret",
	}

	SetProfile(config, "test-profile", profile)

	if _, exists := config.Profiles["test-profile"]; !exists {
		t.Error("Profile was not added to config")
	}

	if config.Profiles["test-profile"].Organization != "testorg" {
		t.Errorf("Organization = %q, expected %q", config.Profiles["test-profile"].Organization, "testorg")
	}
}

func TestSetProfile_NilMap(t *testing.T) {
	t.Parallel()

	config := &Config{
		Profiles: nil,
	}

	profile := &Profile{
		Organization: "testorg",
		Instance:     "us",
	}

	SetProfile(config, "test-profile", profile)

	if config.Profiles == nil {
		t.Error("Profiles map was not initialized")
	}

	if _, exists := config.Profiles["test-profile"]; !exists {
		t.Error("Profile was not added to config")
	}
}

func TestDeleteProfile(t *testing.T) {
	t.Parallel()

	config := &Config{
		CurrentProfile: "default",
		Profiles: map[string]*Profile{
			"default": {Organization: "org1"},
			"other":   {Organization: "org2"},
		},
	}

	err := DeleteProfile(config, "other")
	if err != nil {
		t.Fatalf("DeleteProfile returned error: %v", err)
	}

	if _, exists := config.Profiles["other"]; exists {
		t.Error("Profile was not deleted")
	}

	if config.CurrentProfile != "default" {
		t.Errorf("CurrentProfile changed unexpectedly to %q", config.CurrentProfile)
	}
}

func TestDeleteProfile_CurrentProfile(t *testing.T) {
	t.Parallel()

	config := &Config{
		CurrentProfile: "to-delete",
		Profiles: map[string]*Profile{
			"to-delete": {Organization: "org1"},
			"other":     {Organization: "org2"},
		},
	}

	err := DeleteProfile(config, "to-delete")
	if err != nil {
		t.Fatalf("DeleteProfile returned error: %v", err)
	}

	if _, exists := config.Profiles["to-delete"]; exists {
		t.Error("Profile was not deleted")
	}

	// Should switch to another profile
	if config.CurrentProfile == "to-delete" {
		t.Error("CurrentProfile was not updated after deletion")
	}
}

func TestDeleteProfile_NotFound(t *testing.T) {
	t.Parallel()

	config := &Config{
		Profiles: map[string]*Profile{
			"default": {Organization: "org1"},
		},
	}

	err := DeleteProfile(config, "nonexistent")
	if err == nil {
		t.Error("Expected error for non-existent profile, got nil")
	}
}

func TestSwitchProfile(t *testing.T) {
	t.Parallel()

	config := &Config{
		CurrentProfile: "default",
		Profiles: map[string]*Profile{
			"default": {Organization: "org1"},
			"other":   {Organization: "org2"},
		},
	}

	err := SwitchProfile(config, "other")
	if err != nil {
		t.Fatalf("SwitchProfile returned error: %v", err)
	}

	if config.CurrentProfile != "other" {
		t.Errorf("CurrentProfile = %q, expected %q", config.CurrentProfile, "other")
	}
}

func TestSwitchProfile_NotFound(t *testing.T) {
	t.Parallel()

	config := &Config{
		CurrentProfile: "default",
		Profiles: map[string]*Profile{
			"default": {Organization: "org1"},
		},
	}

	err := SwitchProfile(config, "nonexistent")
	if err == nil {
		t.Error("Expected error for non-existent profile, got nil")
	}
}

func TestListProfiles(t *testing.T) {
	t.Parallel()

	config := &Config{
		Profiles: map[string]*Profile{
			"default": {Organization: "org1"},
			"staging": {Organization: "org2"},
			"dev":     {Organization: "org3"},
		},
	}

	profiles := ListProfiles(config)

	if len(profiles) != 3 {
		t.Errorf("Expected 3 profiles, got %d", len(profiles))
	}

	// Check all profiles are present
	profileSet := make(map[string]bool)
	for _, p := range profiles {
		profileSet[p] = true
	}

	for _, expected := range []string{"default", "staging", "dev"} {
		if !profileSet[expected] {
			t.Errorf("Profile %q not found in list", expected)
		}
	}
}

func TestGetCurrentProfile(t *testing.T) {
	t.Parallel()

	config := &Config{
		CurrentProfile: "default",
		Profiles: map[string]*Profile{
			"default": {Organization: "testorg", Instance: "us"},
		},
	}

	profile, err := GetCurrentProfile(config)
	if err != nil {
		t.Fatalf("GetCurrentProfile returned error: %v", err)
	}

	if profile.Organization != "testorg" {
		t.Errorf("Organization = %q, expected %q", profile.Organization, "testorg")
	}

	if profile.Instance != "us" {
		t.Errorf("Instance = %q, expected %q", profile.Instance, "us")
	}
}

func TestGetCurrentProfile_NilConfig(t *testing.T) {
	t.Parallel()

	_, err := GetCurrentProfile(nil)
	if err == nil {
		t.Error("Expected error for nil config, got nil")
	}
}

func TestGetCurrentProfile_NotFound(t *testing.T) {
	t.Parallel()

	config := &Config{
		CurrentProfile: "nonexistent",
		Profiles:       map[string]*Profile{},
	}

	_, err := GetCurrentProfile(config)
	if err == nil {
		t.Error("Expected error for non-existent profile, got nil")
	}
}

func TestSaveAndLoadConfig(t *testing.T) {
	// Create a temporary directory for the test
	tempDir, err := os.MkdirTemp("", "elementum-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create config in temp dir
	configPath := filepath.Join(tempDir, "config.yaml")

	config := &Config{
		CurrentProfile: "test",
		Profiles: map[string]*Profile{
			"test": {
				Organization:  "testorg",
				Instance:      "eu",
				Environment:   "staging",
				CustomBaseURL: "",
				ClientID:      "test-client-id",
				ClientSecret:  "test-secret",
			},
		},
	}

	// Manually write the config
	data, err := yaml.Marshal(config)
	if err != nil {
		t.Fatalf("Failed to marshal config: %v", err)
	}

	if err := os.WriteFile(configPath, data, 0600); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}

	// Read and verify
	readData, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("Failed to read config: %v", err)
	}

	var loadedConfig Config
	if err := yaml.Unmarshal(readData, &loadedConfig); err != nil {
		t.Fatalf("Failed to unmarshal config: %v", err)
	}

	if loadedConfig.CurrentProfile != "test" {
		t.Errorf("CurrentProfile = %q, expected %q", loadedConfig.CurrentProfile, "test")
	}

	profile := loadedConfig.Profiles["test"]
	if profile == nil {
		t.Fatal("Profile 'test' not found")
	}

	if profile.Organization != "testorg" {
		t.Errorf("Organization = %q, expected %q", profile.Organization, "testorg")
	}

	if profile.Instance != "eu" {
		t.Errorf("Instance = %q, expected %q", profile.Instance, "eu")
	}

	if profile.Environment != "staging" {
		t.Errorf("Environment = %q, expected %q", profile.Environment, "staging")
	}
}

func TestValidInstances(t *testing.T) {
	t.Parallel()

	expected := []string{"us", "eu", "stage", "dev", "custom"}

	if len(ValidInstances) != len(expected) {
		t.Errorf("ValidInstances has %d items, expected %d", len(ValidInstances), len(expected))
	}

	for i, inst := range expected {
		if ValidInstances[i] != inst {
			t.Errorf("ValidInstances[%d] = %q, expected %q", i, ValidInstances[i], inst)
		}
	}
}
