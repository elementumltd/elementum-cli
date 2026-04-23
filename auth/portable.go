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
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"golang.org/x/crypto/scrypt"
	"gopkg.in/yaml.v3"
)

// PortableFormatVersion is the current envelope version for exported profiles.
const PortableFormatVersion = 1

// Default scrypt parameters. Tuned to be strong enough against offline attacks
// while still running in well under a second on modern hardware.
const (
	scryptN      = 32768
	scryptR      = 8
	scryptP      = 1
	scryptKeyLen = 32 // AES-256
	saltLen      = 16
)

// PortableProfile is the plaintext representation of a single profile used
// inside the encrypted payload. Names / URLs are stable between machines;
// secrets are written in the clear inside the ciphertext envelope only.
type PortableProfile struct {
	Name             string `json:"name"`
	Organization     string `json:"organization"`
	Instance         string `json:"instance"`
	Environment      string `json:"environment,omitempty"`
	CustomBaseURL    string `json:"custom_base_url,omitempty"`
	CustomGraphQLURL string `json:"custom_graphql_url,omitempty"`
	CustomAPIURL     string `json:"custom_api_url,omitempty"`
	ClientID         string `json:"client_id"`
	ClientSecret     string `json:"client_secret"`
}

// kdfParams describes how the encryption key was derived from the passphrase.
// Stored alongside the ciphertext so that import can reproduce the same key.
type kdfParams struct {
	N      int `yaml:"n"`
	R      int `yaml:"r"`
	P      int `yaml:"p"`
	KeyLen int `yaml:"key_len"`
}

// portableEnvelope is the on-disk YAML format for an exported profile bundle.
type portableEnvelope struct {
	Version    int       `yaml:"version"`
	KDF        string    `yaml:"kdf"`
	KDFParams  kdfParams `yaml:"kdf_params"`
	Salt       string    `yaml:"salt"`       // base64(raw salt)
	Nonce      string    `yaml:"nonce"`      // base64(GCM nonce)
	Ciphertext string    `yaml:"ciphertext"` // base64(AES-256-GCM(JSON(profiles)))
}

// ErrPortableBadPassphrase is returned when the passphrase cannot decrypt the
// envelope (wrong passphrase or tampered ciphertext).
var ErrPortableBadPassphrase = errors.New("incorrect passphrase or corrupted export file")

// EncodePortable serializes profiles to a passphrase-encrypted YAML envelope.
func EncodePortable(profiles []PortableProfile, passphrase string) ([]byte, error) {
	if passphrase == "" {
		return nil, fmt.Errorf("passphrase must not be empty")
	}

	plaintext, err := json.Marshal(profiles)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal profiles: %w", err)
	}

	salt := make([]byte, saltLen)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return nil, fmt.Errorf("failed to generate salt: %w", err)
	}

	key, err := scrypt.Key([]byte(passphrase), salt, scryptN, scryptR, scryptP, scryptKeyLen)
	if err != nil {
		return nil, fmt.Errorf("failed to derive key: %w", err)
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	nonce := make([]byte, aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	ciphertext := aead.Seal(nil, nonce, plaintext, nil)

	env := portableEnvelope{
		Version: PortableFormatVersion,
		KDF:     "scrypt",
		KDFParams: kdfParams{
			N:      scryptN,
			R:      scryptR,
			P:      scryptP,
			KeyLen: scryptKeyLen,
		},
		Salt:       base64.StdEncoding.EncodeToString(salt),
		Nonce:      base64.StdEncoding.EncodeToString(nonce),
		Ciphertext: base64.StdEncoding.EncodeToString(ciphertext),
	}

	out, err := yaml.Marshal(&env)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal envelope: %w", err)
	}
	return out, nil
}

// DecodePortable decrypts an exported YAML envelope with the given passphrase
// and returns the embedded profile slice.
func DecodePortable(data []byte, passphrase string) ([]PortableProfile, error) {
	if passphrase == "" {
		return nil, fmt.Errorf("passphrase must not be empty")
	}

	var env portableEnvelope
	if err := yaml.Unmarshal(data, &env); err != nil {
		return nil, fmt.Errorf("failed to parse export file: %w", err)
	}

	if env.Version != PortableFormatVersion {
		return nil, fmt.Errorf("unsupported export version %d (this build understands version %d)", env.Version, PortableFormatVersion)
	}
	if env.KDF != "scrypt" {
		return nil, fmt.Errorf("unsupported KDF %q", env.KDF)
	}
	if env.KDFParams.KeyLen != scryptKeyLen {
		return nil, fmt.Errorf("unsupported key length %d", env.KDFParams.KeyLen)
	}

	salt, err := base64.StdEncoding.DecodeString(env.Salt)
	if err != nil {
		return nil, fmt.Errorf("invalid salt: %w", err)
	}
	nonce, err := base64.StdEncoding.DecodeString(env.Nonce)
	if err != nil {
		return nil, fmt.Errorf("invalid nonce: %w", err)
	}
	ciphertext, err := base64.StdEncoding.DecodeString(env.Ciphertext)
	if err != nil {
		return nil, fmt.Errorf("invalid ciphertext: %w", err)
	}

	key, err := scrypt.Key([]byte(passphrase), salt, env.KDFParams.N, env.KDFParams.R, env.KDFParams.P, env.KDFParams.KeyLen)
	if err != nil {
		return nil, fmt.Errorf("failed to derive key: %w", err)
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	if len(nonce) != aead.NonceSize() {
		return nil, fmt.Errorf("invalid nonce length %d", len(nonce))
	}

	plaintext, err := aead.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		// GCM auth failure - passphrase is wrong or payload was tampered.
		return nil, ErrPortableBadPassphrase
	}

	var profiles []PortableProfile
	if err := json.Unmarshal(plaintext, &profiles); err != nil {
		return nil, fmt.Errorf("failed to parse decrypted payload: %w", err)
	}
	return profiles, nil
}
