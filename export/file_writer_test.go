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
