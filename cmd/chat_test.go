// Copyright (c) ElementumAI, Inc.
// SPDX-License-Identifier: MPL-2.0

package cmd

import (
	"testing"
)

func TestChatCommandRegistration(t *testing.T) {
	expectedUse := "chat <agent-name-or-id>"
	if chatCmd.Use != expectedUse {
		t.Errorf("unexpected Use: %q, want %q", chatCmd.Use, expectedUse)
	}

	if chatCmd.Args == nil {
		t.Error("chatCmd.Args should not be nil")
	}

	flags := []string{"message", "continue", "list"}
	for _, name := range flags {
		if chatCmd.Flags().Lookup(name) == nil {
			t.Errorf("missing flag: %s", name)
		}
	}

	if chatCmd.Flags().ShorthandLookup("m") == nil {
		t.Error("missing short flag -m for --message")
	}
	if chatCmd.Flags().ShorthandLookup("c") == nil {
		t.Error("missing short flag -c for --continue")
	}
}

func TestConversationCommandRegistration(t *testing.T) {
	expectedUse := "conversation <agent-name-or-id> <conversation-id>"
	if conversationCmd.Use != expectedUse {
		t.Errorf("unexpected Use: %q, want %q", conversationCmd.Use, expectedUse)
	}
}

func TestAgentArgumentParsing(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		expectUUID bool
	}{
		{"valid uuid", "9063aed1-bf8c-430d-882f-8c502355a3c7", true},
		{"valid uuid uppercase", "9063AED1-BF8C-430D-882F-8C502355A3C7", true},
		{"agent name simple", "IT Support Agent", false},
		{"agent name lowercase", "my agent", false},
		{"agent name with special chars", "Agent - Production (v2)", false},
		{"empty string", "", false},
		{"namespace-like", "support-bot", false},
		{"single word", "Classifier", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isUUID := looksLikeUUID(tt.input)
			if isUUID != tt.expectUUID {
				t.Errorf("looksLikeUUID(%q) = %v, want %v", tt.input, isUUID, tt.expectUUID)
			}
		})
	}
}
