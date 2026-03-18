// Copyright (c) ElementumAI, Inc.
// SPDX-License-Identifier: MPL-2.0

package auth

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/elementumltd/elementum-cli/logger"
	"gopkg.in/yaml.v3"
)

// Config represents the entire configuration file structure
type Config struct {
	CurrentProfile string              `yaml:"current_profile"`
	Profiles       map[string]*Profile `yaml:"profiles"`
}

// Profile represents a single authentication profile
type Profile struct {
	Organization     string `yaml:"organization"`
	Instance         string `yaml:"instance"`                     // us, eu, stage, dev, custom
	Environment      string `yaml:"environment,omitempty"`        // org environment slug (optional)
	CustomBaseURL    string `yaml:"custom_base_url,omitempty"`    // deprecated: use CustomGraphQLURL and CustomAPIURL
	CustomGraphQLURL string `yaml:"custom_graphql_url,omitempty"` // GraphQL endpoint for custom instance (e.g., http://localhost:3000)
	CustomAPIURL     string `yaml:"custom_api_url,omitempty"`     // API/OAuth endpoint for custom instance (e.g., http://localhost:8080)
	ClientID         string `yaml:"client_id"`
	ClientSecret     string `yaml:"client_secret"` // Stored encrypted or as keychain reference
}

// ValidInstances are the valid instance values
var ValidInstances = []string{"us", "eu", "stage", "dev", "custom"}

// IsValidInstance checks if the given instance is valid
func IsValidInstance(instance string) bool {
	for _, valid := range ValidInstances {
		if instance == valid {
			return true
		}
	}
	return false
}

// GetConfigPath returns the path to the config file
func GetConfigPath() (string, error) {
	var configDir string

	switch runtime.GOOS {
	case "windows":
		configDir = os.Getenv("APPDATA")
		if configDir == "" {
			return "", fmt.Errorf("APPDATA environment variable not set")
		}
	default: // macOS, Linux
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("failed to get home directory: %w", err)
		}
		configDir = filepath.Join(home, ".config")
	}

	configPath := filepath.Join(configDir, "ei", "config.yaml")
	return configPath, nil
}

// LoadConfig loads the configuration from disk
func LoadConfig() (*Config, error) {
	configPath, err := GetConfigPath()
	if err != nil {
		return nil, err
	}

	// If config doesn't exist, return empty config
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return &Config{
			CurrentProfile: "default",
			Profiles:       make(map[string]*Profile),
		}, nil
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Initialize profiles map if nil
	if config.Profiles == nil {
		config.Profiles = make(map[string]*Profile)
	}

	// Set default current profile if not set
	if config.CurrentProfile == "" {
		config.CurrentProfile = "default"
	}

	// Migrate old configs: convert old environment values to new instance format
	needsSave := migrateProfiles(&config)
	if needsSave {
		// Auto-save migrated config
		if err := SaveConfig(&config); err != nil {
			logger.Warn("failed to save migrated config", "error", err)
		}
	}

	return &config, nil
}

// migrateProfiles migrates old profile format to new instance-based format
// Returns true if any profiles were migrated
func migrateProfiles(config *Config) bool {
	migrated := false
	for _, profile := range config.Profiles {
		// If instance is not set but we have an old-style environment, migrate it
		if profile.Instance == "" {
			migrated = true
			// Map old environment values to new instance values
			switch profile.Environment {
			case "production", "":
				profile.Instance = "us"
				profile.Environment = "" // Clear old environment field
			case "stage", "staging":
				profile.Instance = "stage"
				profile.Environment = "" // Clear old environment field
			case "dev", "development":
				profile.Instance = "dev"
				profile.Environment = "" // Clear old environment field
			default:
				// If it looks like an org environment slug, assume US instance
				profile.Instance = "us"
				// Keep Environment as-is (it's an org env slug)
			}
		}
	}
	return migrated
}

// SaveConfig saves the configuration to disk
func SaveConfig(config *Config) error {
	configPath, err := GetConfigPath()
	if err != nil {
		return err
	}

	// Create config directory if it doesn't exist
	configDir := filepath.Dir(configPath)
	if err := os.MkdirAll(configDir, 0700); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Write with restricted permissions (owner read/write only)
	if err := os.WriteFile(configPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// GetCurrentProfile returns the current active profile
func GetCurrentProfile(config *Config) (*Profile, error) {
	if config == nil {
		return nil, fmt.Errorf("config is nil")
	}

	profile, exists := config.Profiles[config.CurrentProfile]
	if !exists {
		return nil, fmt.Errorf("profile %q not found", config.CurrentProfile)
	}

	return profile, nil
}

// SetProfile adds or updates a profile in the config
func SetProfile(config *Config, name string, profile *Profile) {
	if config.Profiles == nil {
		config.Profiles = make(map[string]*Profile)
	}
	config.Profiles[name] = profile
}

// DeleteProfile removes a profile from the config
func DeleteProfile(config *Config, name string) error {
	if _, exists := config.Profiles[name]; !exists {
		return fmt.Errorf("profile %q not found", name)
	}

	delete(config.Profiles, name)

	// If we deleted the current profile, switch to another or default
	if config.CurrentProfile == name {
		if len(config.Profiles) > 0 {
			// Pick the first available profile
			for profileName := range config.Profiles {
				config.CurrentProfile = profileName
				break
			}
		} else {
			config.CurrentProfile = "default"
		}
	}

	return nil
}

// SwitchProfile changes the current active profile
func SwitchProfile(config *Config, name string) error {
	if _, exists := config.Profiles[name]; !exists {
		return fmt.Errorf("profile %q not found", name)
	}

	config.CurrentProfile = name
	return nil
}

// ListProfiles returns a list of all profile names
func ListProfiles(config *Config) []string {
	profiles := make([]string, 0, len(config.Profiles))
	for name := range config.Profiles {
		profiles = append(profiles, name)
	}
	return profiles
}
