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
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/elementumltd/elementum-cli/ui"
)

// FileGroup represents a group of resources to be written to a single file
type FileGroup struct {
	FileName  string
	Resources []string
	Comment   string
}

// MultiFileExport contains the organized export data
type MultiFileExport struct {
	AppFiles             []FileGroup // app-{name}.tf files
	AutomationFiles      []FileGroup // automation-{name}.tf files
	AgentFiles           []FileGroup // agent-{name}.tf files
	ElementFiles         []FileGroup // element-{name}.tf files
	ElementSecurityFiles []FileGroup // element-{name}-security.tf files (roles + access policies)
	TaskFiles            []FileGroup // task-{name}.tf files
	TaskSecurityFiles    []FileGroup // task-{name}-security.tf files (roles + access policies)
	ViewFiles            []FileGroup // {namespace}-view.tf files
	FileReaderFiles      []FileGroup // {namespace}-file-readers.tf files
	DatamineFiles        []FileGroup // datamine-{name}.tf files
	TableFiles           []FileGroup // table-{name}.tf files
	SecurityFiles        []FileGroup // {namespace}-security.tf files (roles + access policies)
	AISearchTableFiles   []FileGroup // {namespace}-ai-search.tf files
	ProviderFile         string      // provider.tf content
	VariablesFile        string      // variables.tf content (if needed)
	OutputsFile          string      // outputs.tf content (if needed)
	LocalsFile           string      // locals.tf content (stage lookups, etc.)
	DataSourcesFile      string      // data.tf content (related objects)
	ImportsFile          string      // imports.tf content (Terraform 1.5+ import blocks for state import)
}

// PostProcessAll applies a transformation function to all resource strings in the multi-file export.
// This is used to apply remaining string-based post-processing (e.g., cloud mappings, flow injection)
// after the IR pipeline has serialized the blocks.
func (m *MultiFileExport) PostProcessAll(fn func(string) string) {
	processFileGroups := func(groups []FileGroup) {
		for i := range groups {
			for j := range groups[i].Resources {
				groups[i].Resources[j] = fn(groups[i].Resources[j])
			}
		}
	}

	processFileGroups(m.AppFiles)
	processFileGroups(m.AutomationFiles)
	processFileGroups(m.AgentFiles)
	processFileGroups(m.ElementFiles)
	processFileGroups(m.ElementSecurityFiles)
	processFileGroups(m.TaskFiles)
	processFileGroups(m.TaskSecurityFiles)
	processFileGroups(m.ViewFiles)
	processFileGroups(m.FileReaderFiles)
	processFileGroups(m.DatamineFiles)
	processFileGroups(m.TableFiles)
	processFileGroups(m.SecurityFiles)
	processFileGroups(m.AISearchTableFiles)
}

// WriteMultipleFiles writes the organized export to multiple files in the given directory
func WriteMultipleFiles(export *MultiFileExport, outputDir string) error {
	// Ensure output directory exists
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Write app files
	if len(export.AppFiles) > 0 {
		fmt.Printf("  %s Writing apps...\n", ui.RenderBullet())
		for _, group := range export.AppFiles {
			if err := writeFileGroup(group, outputDir); err != nil {
				return err
			}
		}
	}

	// Write automation files
	if len(export.AutomationFiles) > 0 {
		fmt.Printf("  %s Writing automations...\n", ui.RenderBullet())
		for _, group := range export.AutomationFiles {
			if err := writeFileGroup(group, outputDir); err != nil {
				return err
			}
		}
	}

	// Write agent files
	if len(export.AgentFiles) > 0 {
		fmt.Printf("  %s Writing agents...\n", ui.RenderBullet())
		for _, group := range export.AgentFiles {
			if err := writeFileGroup(group, outputDir); err != nil {
				return err
			}
		}
	}

	// Write element files
	if len(export.ElementFiles) > 0 {
		fmt.Printf("  %s Writing elements...\n", ui.RenderBullet())
		for _, group := range export.ElementFiles {
			if err := writeFileGroup(group, outputDir); err != nil {
				return err
			}
		}
	}

	// Write element security files (roles + access policies)
	for _, group := range export.ElementSecurityFiles {
		if err := writeFileGroup(group, outputDir); err != nil {
			return err
		}
	}

	// Write task files
	if len(export.TaskFiles) > 0 {
		fmt.Printf("  %s Writing tasks...\n", ui.RenderBullet())
		for _, group := range export.TaskFiles {
			if err := writeFileGroup(group, outputDir); err != nil {
				return err
			}
		}
	}

	// Write task security files (roles + access policies)
	for _, group := range export.TaskSecurityFiles {
		if err := writeFileGroup(group, outputDir); err != nil {
			return err
		}
	}

	// Write view files
	for _, group := range export.ViewFiles {
		if err := writeFileGroup(group, outputDir); err != nil {
			return err
		}
	}

	// Write file reader files
	for _, group := range export.FileReaderFiles {
		if err := writeFileGroup(group, outputDir); err != nil {
			return err
		}
	}

	// Write datamine files
	for _, group := range export.DatamineFiles {
		if err := writeFileGroup(group, outputDir); err != nil {
			return err
		}
	}

	// Write table files
	for _, group := range export.TableFiles {
		if err := writeFileGroup(group, outputDir); err != nil {
			return err
		}
	}

	// Write security files (roles + access policies)
	for _, group := range export.SecurityFiles {
		if err := writeFileGroup(group, outputDir); err != nil {
			return err
		}
	}

	// Write AI search table files
	for _, group := range export.AISearchTableFiles {
		if err := writeFileGroup(group, outputDir); err != nil {
			return err
		}
	}

	// Write locals file if present
	if export.LocalsFile != "" {
		localPath := filepath.Join(outputDir, "locals.tf")
		if err := os.WriteFile(localPath, []byte(export.LocalsFile), 0644); err != nil {
			return fmt.Errorf("failed to write locals.tf: %w", err)
		}
	}

	// Write data sources file if present
	if export.DataSourcesFile != "" {
		dataPath := filepath.Join(outputDir, "data.tf")
		if err := os.WriteFile(dataPath, []byte(export.DataSourcesFile), 0644); err != nil {
			return fmt.Errorf("failed to write data.tf: %w", err)
		}
	}

	// Write imports file if present. Users delete this after first apply.
	if export.ImportsFile != "" {
		importsPath := filepath.Join(outputDir, "imports.tf")
		if err := os.WriteFile(importsPath, []byte(export.ImportsFile), 0644); err != nil {
			return fmt.Errorf("failed to write imports.tf: %w", err)
		}
	}

	// Write providers file if present and doesn't already exist
	if export.ProviderFile != "" {
		providerPath := filepath.Join(outputDir, "providers.tf")
		// Only write if file doesn't exist (don't overwrite user's providers.tf)
		if _, err := os.Stat(providerPath); os.IsNotExist(err) {
			if err := os.WriteFile(providerPath, []byte(export.ProviderFile), 0644); err != nil {
				return fmt.Errorf("failed to write providers.tf: %w", err)
			}
		}
	}

	return nil
}

// writeFileGroup writes a FileGroup to a file
func writeFileGroup(group FileGroup, outputDir string) error {
	if len(group.Resources) == 0 {
		return nil
	}

	var sb strings.Builder

	// Write comment header
	if group.Comment != "" {
		sb.WriteString(group.Comment)
	}

	// Write resources
	for i, resource := range group.Resources {
		sb.WriteString(resource)
		if i < len(group.Resources)-1 {
			sb.WriteString("\n\n")
		}
	}
	sb.WriteString("\n")

	filePath := filepath.Join(outputDir, group.FileName)
	if err := os.WriteFile(filePath, []byte(sb.String()), 0644); err != nil {
		return fmt.Errorf("failed to write %s: %w", group.FileName, err)
	}

	return nil
}

// GetFileCount returns the total number of files that will be written
func (e *MultiFileExport) GetFileCount() int {
	count := len(e.AppFiles) + len(e.AutomationFiles) + len(e.AgentFiles) + len(e.ElementFiles) + len(e.ElementSecurityFiles) + len(e.TaskFiles) + len(e.TaskSecurityFiles) + len(e.ViewFiles) + len(e.FileReaderFiles) + len(e.DatamineFiles) + len(e.TableFiles) + len(e.SecurityFiles) + len(e.AISearchTableFiles)
	if e.ProviderFile != "" {
		count++
	}
	if e.LocalsFile != "" {
		count++
	}
	if e.DataSourcesFile != "" {
		count++
	}
	if e.ImportsFile != "" {
		count++
	}
	return count
}

// GetFileNames returns all file names that will be written
func (e *MultiFileExport) GetFileNames() []string {
	var names []string
	for _, g := range e.AppFiles {
		names = append(names, g.FileName)
	}
	for _, g := range e.AutomationFiles {
		names = append(names, g.FileName)
	}
	for _, g := range e.AgentFiles {
		names = append(names, g.FileName)
	}
	for _, g := range e.ElementFiles {
		names = append(names, g.FileName)
	}
	for _, g := range e.ElementSecurityFiles {
		names = append(names, g.FileName)
	}
	for _, g := range e.TaskFiles {
		names = append(names, g.FileName)
	}
	for _, g := range e.TaskSecurityFiles {
		names = append(names, g.FileName)
	}
	for _, g := range e.ViewFiles {
		names = append(names, g.FileName)
	}
	for _, g := range e.FileReaderFiles {
		names = append(names, g.FileName)
	}
	for _, g := range e.DatamineFiles {
		names = append(names, g.FileName)
	}
	for _, g := range e.TableFiles {
		names = append(names, g.FileName)
	}
	for _, g := range e.SecurityFiles {
		names = append(names, g.FileName)
	}
	for _, g := range e.AISearchTableFiles {
		names = append(names, g.FileName)
	}
	if e.ProviderFile != "" {
		names = append(names, "providers.tf")
	}
	if e.LocalsFile != "" {
		names = append(names, "locals.tf")
	}
	if e.DataSourcesFile != "" {
		names = append(names, "data.tf")
	}
	if e.ImportsFile != "" {
		names = append(names, "imports.tf")
	}
	return names
}
