// Copyright (c) ElementumAI, Inc.
// SPDX-License-Identifier: MPL-2.0

package export

import (
	"testing"
)

func TestMultiFileExport_GetFileCount_IncludesSecurity(t *testing.T) {
	t.Parallel()

	export := &MultiFileExport{
		AppFiles:             []FileGroup{{FileName: "app.tf"}},
		AutomationFiles:      []FileGroup{{FileName: "auto1.tf"}, {FileName: "auto2.tf"}},
		AgentFiles:           []FileGroup{{FileName: "agent.tf"}},
		ElementFiles:         []FileGroup{{FileName: "element.tf"}},
		ElementSecurityFiles: []FileGroup{{FileName: "element-security.tf"}},
		FileReaderFiles:      []FileGroup{{FileName: "reader.tf"}},
		DatamineFiles:        []FileGroup{{FileName: "datamine.tf"}},
		TableFiles:           []FileGroup{{FileName: "table.tf"}},
		SecurityFiles:        []FileGroup{{FileName: "security.tf"}},
		ProviderFile:         "provider content",
		LocalsFile:           "locals content",
		DataSourcesFile:      "data content",
	}

	// 1 + 2 + 1 + 1 + 1 + 1 + 1 + 1 + 1 + 1 + 1 + 1 = 13
	expectedCount := 13
	if got := export.GetFileCount(); got != expectedCount {
		t.Errorf("GetFileCount() = %d, want %d", got, expectedCount)
	}
}

func TestMultiFileExport_GetFileNames_IncludesSecurity(t *testing.T) {
	t.Parallel()

	export := &MultiFileExport{
		AppFiles:      []FileGroup{{FileName: "app-test.tf"}},
		SecurityFiles: []FileGroup{{FileName: "test-security.tf"}},
	}

	names := export.GetFileNames()

	hasAppFile := false
	hasSecurityFile := false
	for _, name := range names {
		if name == "app-test.tf" {
			hasAppFile = true
		}
		if name == "test-security.tf" {
			hasSecurityFile = true
		}
	}

	if !hasAppFile {
		t.Error("GetFileNames() should include app file")
	}
	if !hasSecurityFile {
		t.Error("GetFileNames() should include security file")
	}
}

func TestMultiFileExport_IncludesAISearchTableFiles(t *testing.T) {
	t.Parallel()

	export := &MultiFileExport{
		AppFiles: []FileGroup{
			{FileName: "app-test.tf", Resources: []string{"resource {}"}},
		},
		AISearchTableFiles: []FileGroup{
			{FileName: "testapp-ai-search.tf", Resources: []string{"resource {}"}},
		},
	}

	// Test GetFileCount
	count := export.GetFileCount()
	if count != 2 {
		t.Errorf("Expected file count 2, got %d", count)
	}

	// Test GetFileNames
	names := export.GetFileNames()
	hasSearchFile := false
	for _, name := range names {
		if name == "testapp-ai-search.tf" {
			hasSearchFile = true
			break
		}
	}
	if !hasSearchFile {
		t.Error("Expected AI search table file in GetFileNames()")
	}
}
