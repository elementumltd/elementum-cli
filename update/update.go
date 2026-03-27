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

// Package update provides self-update functionality for the ei CLI.
package update

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/creativeprojects/go-selfupdate"
)

const (
	// RepoOwner is the GitHub organization/user
	RepoOwner = "elementumltd"

	// RepoName is the GitHub repository name
	RepoName = "terraform-provider-elementum"

	// AssetPrefix is the prefix for CLI release assets
	AssetPrefix = "ei_"

	// CacheFileName is the name of the version check cache file
	CacheFileName = "version_check.json"

	// CacheExpiry is how long before we re-check for updates
	CacheExpiry = 24 * time.Hour
)

// ghToken is a read-only token for accessing releases from the private repo.
// This is a fine-grained PAT with only contents:read permission on this repo.
// Can be overridden via GITHUB_TOKEN env var.
// TODO: Replace with actual token before release
var ghToken = ""

// UpdateInfo contains information about an available update
type UpdateInfo struct {
	CurrentVersion string
	LatestVersion  string
	ReleaseNotes   string
	PublishedAt    time.Time
	AssetURL       string
	AssetName      string
}

// HasUpdate returns true if a newer version is available
func (u *UpdateInfo) HasUpdate() bool {
	return u.LatestVersion != "" && u.LatestVersion != u.CurrentVersion
}

// VersionCache stores the result of the last version check
type VersionCache struct {
	LatestVersion string    `json:"latest_version"`
	CheckedAt     time.Time `json:"checked_at"`
	Notified      bool      `json:"notified"`
}

// getCacheDir returns the directory for storing the version cache
func getCacheDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(homeDir, ".config", "ei"), nil
}

// getCachePath returns the full path to the cache file
func getCachePath() (string, error) {
	dir, err := getCacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, CacheFileName), nil
}

// LoadCache loads the version cache from disk
func LoadCache() (*VersionCache, error) {
	cachePath, err := getCachePath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(cachePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var cache VersionCache
	if err := json.Unmarshal(data, &cache); err != nil {
		return nil, err
	}

	return &cache, nil
}

// SaveCache saves the version cache to disk
func SaveCache(cache *VersionCache) error {
	cacheDir, err := getCacheDir()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return err
	}

	cachePath := filepath.Join(cacheDir, CacheFileName)
	data, err := json.MarshalIndent(cache, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(cachePath, data, 0644)
}

// IsCacheStale returns true if the cache is older than CacheExpiry
func IsCacheStale(cache *VersionCache) bool {
	if cache == nil {
		return true
	}
	return time.Since(cache.CheckedAt) > CacheExpiry
}

// MarkNotified marks that the user has been notified about an update
func MarkNotified() error {
	cache, err := LoadCache()
	if err != nil || cache == nil {
		return err
	}
	cache.Notified = true
	return SaveCache(cache)
}

// getGitHubToken returns the GitHub token to use for API requests.
// Priority: GITHUB_TOKEN env var > embedded token
func getGitHubToken() string {
	if token := os.Getenv("GITHUB_TOKEN"); token != "" {
		return token
	}
	return ghToken
}

// CheckForUpdate queries GitHub for the latest release
func CheckForUpdate(ctx context.Context, currentVersion string) (*UpdateInfo, error) {
	// Configure the updater to find CLI assets
	source, err := selfupdate.NewGitHubSource(selfupdate.GitHubConfig{
		APIToken: getGitHubToken(),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create GitHub source: %w", err)
	}

	updater, err := selfupdate.NewUpdater(selfupdate.Config{
		Source:    source,
		Validator: nil, // We'll rely on GitHub's checksums
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create updater: %w", err)
	}

	// Find the latest release
	latest, found, err := updater.DetectLatest(ctx, selfupdate.NewRepositorySlug(RepoOwner, RepoName))
	if err != nil {
		return nil, fmt.Errorf("failed to detect latest version: %w", err)
	}

	if !found {
		return &UpdateInfo{
			CurrentVersion: currentVersion,
		}, nil
	}

	info := &UpdateInfo{
		CurrentVersion: currentVersion,
		LatestVersion:  latest.Version(),
		ReleaseNotes:   latest.ReleaseNotes,
		PublishedAt:    latest.PublishedAt,
		AssetURL:       latest.AssetURL,
		AssetName:      latest.AssetName,
	}

	// Update cache
	cache := &VersionCache{
		LatestVersion: latest.Version(),
		CheckedAt:     time.Now(),
		Notified:      false,
	}
	// Best effort - don't fail if cache write fails
	_ = SaveCache(cache)

	return info, nil
}

// PerformUpdate downloads and installs the latest version
func PerformUpdate(ctx context.Context, currentVersion string) error {
	// Get the path to the current executable
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}

	// Resolve symlinks
	exe, err = filepath.EvalSymlinks(exe)
	if err != nil {
		return fmt.Errorf("failed to resolve executable path: %w", err)
	}

	// Configure the updater
	source, err := selfupdate.NewGitHubSource(selfupdate.GitHubConfig{
		APIToken: getGitHubToken(),
	})
	if err != nil {
		return fmt.Errorf("failed to create GitHub source: %w", err)
	}

	updater, err := selfupdate.NewUpdater(selfupdate.Config{
		Source:    source,
		Validator: nil,
	})
	if err != nil {
		return fmt.Errorf("failed to create updater: %w", err)
	}

	// Find the latest release
	latest, found, err := updater.DetectLatest(ctx, selfupdate.NewRepositorySlug(RepoOwner, RepoName))
	if err != nil {
		return fmt.Errorf("failed to detect latest version: %w", err)
	}

	if !found {
		return fmt.Errorf("no releases found")
	}

	// Check if we need to update
	if latest.Version() == currentVersion {
		return fmt.Errorf("already at latest version %s", currentVersion)
	}

	// Perform the update
	if err := updater.UpdateTo(ctx, latest, exe); err != nil {
		return fmt.Errorf("failed to update: %w", err)
	}

	return nil
}

// GetAssetName returns the expected asset name for the current platform
func GetAssetName() string {
	return fmt.Sprintf("%s%s_%s", AssetPrefix, runtime.GOOS, runtime.GOARCH)
}
