// Copyright (c) ElementumAI, Inc.
// SPDX-License-Identifier: MPL-2.0

package auth

import (
	"fmt"
	"strings"

	"github.com/zalando/go-keyring"
)

const (
	keyringService = "elementum-cli"
)

// StoreSecret stores a client secret securely in the OS keychain
// Returns the reference string to store in the config file
func StoreSecret(profileName, secret string) (string, error) {
	keyName := fmt.Sprintf("profile:%s:secret", profileName)

	err := keyring.Set(keyringService, keyName, secret)
	if err != nil {
		// Keychain not available, return the secret for file-based storage
		// In production, you might want to encrypt this
		return secret, fmt.Errorf("keychain not available, storing in config (warning: not encrypted): %w", err)
	}

	// Return a reference instead of the actual secret
	return fmt.Sprintf("keyring:%s", keyName), nil
}

// GetSecret retrieves a client secret from the OS keychain or config
func GetSecret(profileName, storedValue string) (string, error) {
	// Check if it's a keyring reference (handles profile names with spaces)
	if strings.HasPrefix(storedValue, "keyring:") {
		keyName := strings.TrimPrefix(storedValue, "keyring:")
		secret, err := keyring.Get(keyringService, keyName)
		if err != nil {
			return "", fmt.Errorf("failed to retrieve secret from keychain: %w", err)
		}
		return secret, nil
	}

	// It's stored directly in the config (fallback for when keychain is not available)
	return storedValue, nil
}

// DeleteSecret removes a secret from the OS keychain
func DeleteSecret(profileName string) error {
	keyName := fmt.Sprintf("profile:%s:secret", profileName)

	err := keyring.Delete(keyringService, keyName)
	if err != nil {
		// Keychain might not have this entry, which is fine
		return nil
	}

	return nil
}

// TestKeychain tests if the OS keychain is available
func TestKeychain() bool {
	testKey := "test-key"
	testValue := "test-value"

	// Try to set a value
	err := keyring.Set(keyringService, testKey, testValue)
	if err != nil {
		return false
	}

	// Clean up
	_ = keyring.Delete(keyringService, testKey)
	return true
}
