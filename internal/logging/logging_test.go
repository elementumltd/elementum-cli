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

package logging

import (
	"sync"
	"testing"
)

func TestIsNonActionableGraphQLError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		message  string
		expected bool
	}{
		{"item does not exist", "The item does not exist", true},
		{"item does not exist with context", "GraphQL error: The item does not exist.", true},
		{"unexpected error", "An unexpected error has occurred", true},
		{"unexpected error with context", "GraphQL error: An unexpected error has occurred.", true},
		{"auth error", "This item requires authentication.", false},
		{"timeout", "context deadline exceeded", false},
		{"random error", "something went wrong", false},
		{"empty string", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := isNonActionableGraphQLError(tt.message)
			if got != tt.expected {
				t.Errorf("isNonActionableGraphQLError(%q) = %v, want %v", tt.message, got, tt.expected)
			}
		})
	}
}

func TestGraphQLWarn_DemotesNonActionableToDebug(t *testing.T) {
	// Save and restore global state
	origEnabled := warnEnabled
	origWarnFunc := warnFunc
	origDebugFunc := debugFunc
	defer func() {
		mu.Lock()
		warnEnabled = origEnabled
		warnFunc = origWarnFunc
		debugFunc = origDebugFunc
		mu.Unlock()
	}()

	var warnCalled, debugCalled bool
	var warnMsg, debugMsg string

	SetWarnEnabled(true)
	SetWarnFunc(func(msg string, keyvals ...any) {
		warnCalled = true
		warnMsg = msg
	})
	SetDebugFunc(func(msg string, keyvals ...any) {
		debugCalled = true
		debugMsg = msg
	})

	// Non-actionable error should go to debug, not warn
	GraphQLWarn("The item does not exist", "TestOp", nil, nil)

	if warnCalled {
		t.Errorf("Non-actionable error should not call warnFunc, but it was called with: %s", warnMsg)
	}
	if !debugCalled {
		t.Error("Non-actionable error should call debugFunc, but it was not called")
	}
	if debugMsg == "" {
		t.Error("debugFunc should receive a message")
	}
}

func TestGraphQLWarn_ActionableGoesToWarn(t *testing.T) {
	// Save and restore global state
	origEnabled := warnEnabled
	origWarnFunc := warnFunc
	origDebugFunc := debugFunc
	defer func() {
		mu.Lock()
		warnEnabled = origEnabled
		warnFunc = origWarnFunc
		debugFunc = origDebugFunc
		mu.Unlock()
	}()

	var warnCalled, debugCalled bool

	SetWarnEnabled(true)
	SetWarnFunc(func(msg string, keyvals ...any) {
		warnCalled = true
	})
	SetDebugFunc(func(msg string, keyvals ...any) {
		debugCalled = true
	})

	// Actionable error should go to warn, not debug
	GraphQLWarn("This item requires authentication.", "TestOp", nil, nil)

	if !warnCalled {
		t.Error("Actionable error should call warnFunc")
	}
	if debugCalled {
		t.Error("Actionable error should not call debugFunc")
	}
}

func TestGraphQLWarn_UnexpectedErrorDemotedToDebug(t *testing.T) {
	// Save and restore global state
	origEnabled := warnEnabled
	origWarnFunc := warnFunc
	origDebugFunc := debugFunc
	defer func() {
		mu.Lock()
		warnEnabled = origEnabled
		warnFunc = origWarnFunc
		debugFunc = origDebugFunc
		mu.Unlock()
	}()

	var warnCalled, debugCalled bool

	SetWarnEnabled(true)
	SetWarnFunc(func(msg string, keyvals ...any) {
		warnCalled = true
	})
	SetDebugFunc(func(msg string, keyvals ...any) {
		debugCalled = true
	})

	GraphQLWarn("An unexpected error has occurred", "TestOp", nil, nil)

	if warnCalled {
		t.Error("'An unexpected error has occurred' should be demoted to debug")
	}
	if !debugCalled {
		t.Error("'An unexpected error has occurred' should call debugFunc")
	}
}

func TestGraphQLWarn_NoDebugFuncSilentlyDrops(t *testing.T) {
	// Save and restore global state
	origEnabled := warnEnabled
	origWarnFunc := warnFunc
	origDebugFunc := debugFunc
	defer func() {
		mu.Lock()
		warnEnabled = origEnabled
		warnFunc = origWarnFunc
		debugFunc = origDebugFunc
		mu.Unlock()
	}()

	var warnCalled bool

	SetWarnEnabled(true)
	SetWarnFunc(func(msg string, keyvals ...any) {
		warnCalled = true
	})
	SetDebugFunc(nil) // No debug handler

	// Should silently drop without calling warn
	GraphQLWarn("The item does not exist", "TestOp", nil, nil)

	if warnCalled {
		t.Error("Non-actionable error with nil debugFunc should not fall through to warnFunc")
	}
}

func TestWarn_DisabledByDefault(t *testing.T) {
	// Save and restore global state
	origEnabled := warnEnabled
	origWarnFunc := warnFunc
	defer func() {
		mu.Lock()
		warnEnabled = origEnabled
		warnFunc = origWarnFunc
		mu.Unlock()
	}()

	var called bool
	SetWarnEnabled(false)
	SetWarnFunc(func(msg string, keyvals ...any) {
		called = true
	})

	Warn("test warning")

	if called {
		t.Error("Warn should not call warnFunc when disabled")
	}
}

func TestWarn_EnabledCallsFunc(t *testing.T) {
	// Save and restore global state
	origEnabled := warnEnabled
	origWarnFunc := warnFunc
	defer func() {
		mu.Lock()
		warnEnabled = origEnabled
		warnFunc = origWarnFunc
		mu.Unlock()
	}()

	var called bool
	var gotMsg string
	SetWarnEnabled(true)
	SetWarnFunc(func(msg string, keyvals ...any) {
		called = true
		gotMsg = msg
	})

	Warn("test warning")

	if !called {
		t.Error("Warn should call warnFunc when enabled")
	}
	if gotMsg != "test warning" {
		t.Errorf("Warn passed wrong message: got %q, want %q", gotMsg, "test warning")
	}
}

func TestConcurrentAccess(t *testing.T) {
	t.Parallel()

	// Save and restore global state
	origEnabled := warnEnabled
	origWarnFunc := warnFunc
	origDebugFunc := debugFunc
	defer func() {
		mu.Lock()
		warnEnabled = origEnabled
		warnFunc = origWarnFunc
		debugFunc = origDebugFunc
		mu.Unlock()
	}()

	SetWarnEnabled(true)
	SetWarnFunc(func(msg string, keyvals ...any) {})
	SetDebugFunc(func(msg string, keyvals ...any) {})

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			Warn("concurrent warn")
			GraphQLWarn("The item does not exist", "op", nil, nil)
			GraphQLWarn("auth error", "op", nil, nil)
		}()
	}
	wg.Wait()
}
