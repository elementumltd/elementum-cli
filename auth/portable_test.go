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
	"encoding/base64"
	"errors"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func sampleProfiles() []PortableProfile {
	return []PortableProfile{
		{
			Name:         "default",
			Organization: "acme",
			Instance:     "us",
			ClientID:     "client-id-1",
			ClientSecret: "super-secret-1",
		},
		{
			Name:             "custom-local",
			Organization:     "ironman",
			Instance:         "custom",
			Environment:      "staging",
			CustomGraphQLURL: "http://localhost:3000",
			CustomAPIURL:     "http://localhost:8700",
			ClientID:         "client-id-2",
			ClientSecret:     "super-secret-2",
		},
	}
}

func TestPortableRoundTrip(t *testing.T) {
	t.Parallel()

	profiles := sampleProfiles()
	passphrase := "correct horse battery staple"

	data, err := EncodePortable(profiles, passphrase)
	if err != nil {
		t.Fatalf("EncodePortable: %v", err)
	}

	// Envelope should be valid YAML with a version field and a ciphertext that
	// does NOT contain the plaintext secrets.
	var probe portableEnvelope
	if err := yaml.Unmarshal(data, &probe); err != nil {
		t.Fatalf("envelope not valid yaml: %v", err)
	}
	if probe.Version != PortableFormatVersion {
		t.Fatalf("envelope version = %d, want %d", probe.Version, PortableFormatVersion)
	}
	if strings.Contains(string(data), "super-secret-1") || strings.Contains(string(data), "super-secret-2") {
		t.Fatal("envelope leaked a plaintext secret")
	}

	got, err := DecodePortable(data, passphrase)
	if err != nil {
		t.Fatalf("DecodePortable: %v", err)
	}

	if len(got) != len(profiles) {
		t.Fatalf("decoded %d profiles, want %d", len(got), len(profiles))
	}
	for i := range profiles {
		if got[i] != profiles[i] {
			t.Errorf("profile %d mismatch:\n got  %+v\n want %+v", i, got[i], profiles[i])
		}
	}
}

func TestPortableWrongPassphrase(t *testing.T) {
	t.Parallel()

	data, err := EncodePortable(sampleProfiles(), "right")
	if err != nil {
		t.Fatalf("EncodePortable: %v", err)
	}

	_, err = DecodePortable(data, "wrong")
	if !errors.Is(err, ErrPortableBadPassphrase) {
		t.Fatalf("expected ErrPortableBadPassphrase, got %v", err)
	}
}

func TestPortableUnsupportedVersion(t *testing.T) {
	t.Parallel()

	env := portableEnvelope{
		Version:    999,
		KDF:        "scrypt",
		KDFParams:  kdfParams{N: scryptN, R: scryptR, P: scryptP, KeyLen: scryptKeyLen},
		Salt:       base64.StdEncoding.EncodeToString([]byte("0123456789abcdef")),
		Nonce:      base64.StdEncoding.EncodeToString(make([]byte, 12)),
		Ciphertext: base64.StdEncoding.EncodeToString([]byte("nope")),
	}
	data, err := yaml.Marshal(&env)
	if err != nil {
		t.Fatalf("yaml.Marshal: %v", err)
	}

	_, err = DecodePortable(data, "anything")
	if err == nil || !strings.Contains(err.Error(), "unsupported export version") {
		t.Fatalf("expected unsupported version error, got %v", err)
	}
}

func TestPortableTamperedCiphertext(t *testing.T) {
	t.Parallel()

	data, err := EncodePortable(sampleProfiles(), "passphrase")
	if err != nil {
		t.Fatalf("EncodePortable: %v", err)
	}

	var env portableEnvelope
	if err := yaml.Unmarshal(data, &env); err != nil {
		t.Fatalf("yaml.Unmarshal: %v", err)
	}

	raw, err := base64.StdEncoding.DecodeString(env.Ciphertext)
	if err != nil {
		t.Fatalf("decode ciphertext: %v", err)
	}
	raw[0] ^= 0xFF // flip a byte
	env.Ciphertext = base64.StdEncoding.EncodeToString(raw)

	tampered, err := yaml.Marshal(&env)
	if err != nil {
		t.Fatalf("yaml.Marshal: %v", err)
	}

	_, err = DecodePortable(tampered, "passphrase")
	if !errors.Is(err, ErrPortableBadPassphrase) {
		t.Fatalf("expected ErrPortableBadPassphrase after tampering, got %v", err)
	}
}

func TestPortableEmptyPassphrase(t *testing.T) {
	t.Parallel()

	if _, err := EncodePortable(sampleProfiles(), ""); err == nil {
		t.Fatal("expected error for empty passphrase on encode")
	}
	// Give DecodePortable a well-formed envelope so it reaches the passphrase
	// check rather than failing at YAML parsing.
	data, err := EncodePortable(sampleProfiles(), "x")
	if err != nil {
		t.Fatalf("EncodePortable: %v", err)
	}
	if _, err := DecodePortable(data, ""); err == nil {
		t.Fatal("expected error for empty passphrase on decode")
	}
}
