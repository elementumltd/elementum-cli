// Copyright (c) ElementumAI, Inc.
// SPDX-License-Identifier: MPL-2.0

package export

import (
	"fmt"
	"sort"
	"strings"

	"github.com/elementumltd/elementum-cli/discovery"
)

// BlockFileGroup is the IR equivalent of FileGroup. Holds blocks instead of strings.
// Serialization to text happens at write time.
type BlockFileGroup struct {
	FileName string
	Blocks   []*HCLBlock
	Comment  string
}

// Serialize serializes all blocks in this group to HCL text.
func (g *BlockFileGroup) Serialize() string {
	if len(g.Blocks) == 0 {
		return ""
	}
	var sb strings.Builder
	if g.Comment != "" {
		sb.WriteString("# " + g.Comment + "\n\n")
	}
	sb.WriteString(SerializeBlocks(g.Blocks))
	return sb.String()
}

// BlockMultiFileExport is the IR equivalent of MultiFileExport.
type BlockMultiFileExport struct {
	FileGroups      []BlockFileGroup
	ProviderFile    string
	VariablesFile   string
	OutputsFile     string
	LocalsFile      string
	DataSourcesFile string
}

// ToMultiFileExport converts to the legacy MultiFileExport structure for backward compatibility.
func (e *BlockMultiFileExport) ToMultiFileExport() *MultiFileExport {
	result := &MultiFileExport{
		ProviderFile:    e.ProviderFile,
		VariablesFile:   e.VariablesFile,
		OutputsFile:     e.OutputsFile,
		LocalsFile:      e.LocalsFile,
		DataSourcesFile: e.DataSourcesFile,
	}

	for _, group := range e.FileGroups {
		fg := FileGroup{
			FileName:  group.FileName,
			Resources: []string{group.Serialize()},
			Comment:   group.Comment,
		}

		// Route to the correct field based on filename pattern
		switch {
		case strings.HasPrefix(group.FileName, "automation-"):
			result.AutomationFiles = append(result.AutomationFiles, fg)
		case strings.HasPrefix(group.FileName, "agent-"):
			result.AgentFiles = append(result.AgentFiles, fg)
		case strings.HasPrefix(group.FileName, "element-") && strings.HasSuffix(group.FileName, "-security.tf"):
			result.ElementSecurityFiles = append(result.ElementSecurityFiles, fg)
		case strings.HasPrefix(group.FileName, "element-"):
			result.ElementFiles = append(result.ElementFiles, fg)
		case strings.HasPrefix(group.FileName, "task-") && strings.HasSuffix(group.FileName, "-security.tf"):
			result.TaskSecurityFiles = append(result.TaskSecurityFiles, fg)
		case strings.HasPrefix(group.FileName, "task-"):
			result.TaskFiles = append(result.TaskFiles, fg)
		case strings.HasPrefix(group.FileName, "datamine-"):
			result.DatamineFiles = append(result.DatamineFiles, fg)
		case strings.HasPrefix(group.FileName, "table-"):
			result.TableFiles = append(result.TableFiles, fg)
		case strings.HasSuffix(group.FileName, "-view.tf"):
			result.ViewFiles = append(result.ViewFiles, fg)
		case strings.HasSuffix(group.FileName, "-file-readers.tf"):
			result.FileReaderFiles = append(result.FileReaderFiles, fg)
		case strings.HasSuffix(group.FileName, "-security.tf"):
			result.SecurityFiles = append(result.SecurityFiles, fg)
		case strings.HasSuffix(group.FileName, "-ai-search.tf"):
			result.AISearchTableFiles = append(result.AISearchTableFiles, fg)
		default:
			result.AppFiles = append(result.AppFiles, fg)
		}
	}

	return result
}

// OrganizeBlocksExport groups IR blocks into files based on their metadata.
// This is the IR equivalent of OrganizeExport() that eliminates regex-based block extraction.
//
// The approach:
//  1. Each block's Meta.Category and Meta.GroupKey determine which file it goes into
//  2. Blocks are grouped by (category, groupKey) into BlockFileGroups
//  3. File naming follows the same conventions as the legacy organize* functions
func OrganizeBlocksExport(blocks []*HCLBlock, app *discovery.App, imports []ImportBlock) *BlockMultiFileExport {
	export := &BlockMultiFileExport{}

	// Group blocks by their file destination
	fileMap := make(map[string]*BlockFileGroup)
	var fileOrder []string // preserve insertion order

	for _, block := range blocks {
		fileName := determineFileName(block, app, imports)
		if fileName == "" {
			continue
		}

		if _, exists := fileMap[fileName]; !exists {
			fileMap[fileName] = &BlockFileGroup{
				FileName: fileName,
			}
			fileOrder = append(fileOrder, fileName)
		}
		fileMap[fileName].Blocks = append(fileMap[fileName].Blocks, block)
	}

	// Build ordered file groups and sort blocks within each group
	for _, name := range fileOrder {
		fg := fileMap[name]
		// Sort blocks so workflow_publish comes last (it depends on all tasks)
		SortBlocksByType(fg.Blocks)
		export.FileGroups = append(export.FileGroups, *fg)
	}

	return export
}

// determineFileName determines which output file a block should go into.
// Uses Meta.Category and Meta.GroupKey if set, otherwise derives from resource type.
func determineFileName(block *HCLBlock, app *discovery.App, imports []ImportBlock) string {
	// If category is explicitly set, use it
	if block.Meta.Category != "" {
		return buildFileName(block.Meta.Category, block.Meta.GroupKey)
	}

	// Derive from resource type
	if len(block.Labels) < 2 {
		return ""
	}

	resourceType := block.Labels[0]
	resourceName := block.Labels[1]

	switch {
	// Automations, triggers, tasks -> automation-{name}.tf
	case resourceType == "elementum_automation":
		return fmt.Sprintf("automation-%s.tf", SanitizeFileName(resourceName))
	case strings.HasSuffix(resourceType, "_trigger"):
		autoName := findAutomationNameForTrigger(resourceName, imports, app)
		if autoName != "" {
			return fmt.Sprintf("automation-%s.tf", SanitizeFileName(autoName))
		}
		return "automations.tf"
	case strings.HasSuffix(resourceType, "_task"):
		autoName := findAutomationNameForTask(resourceName, imports, app)
		if autoName != "" {
			return fmt.Sprintf("automation-%s.tf", SanitizeFileName(autoName))
		}
		return "automations.tf"
	case resourceType == "elementum_workflow_publish":
		// workflow_publish goes with its automation (resource name matches automation name)
		return fmt.Sprintf("automation-%s.tf", SanitizeFileName(resourceName))

	// Agents and tools -> agent-{name}.tf
	case resourceType == "elementum_agent":
		return fmt.Sprintf("agent-%s.tf", SanitizeFileName(resourceName))
	case strings.Contains(resourceType, "_agent_") && strings.HasSuffix(resourceType, "_tool"):
		agentName := findAgentNameForTool(resourceName, imports, app)
		if agentName != "" {
			return fmt.Sprintf("agent-%s.tf", SanitizeFileName(agentName))
		}
		return "agents.tf"

	// File readers -> {namespace}-file-readers.tf
	case strings.HasSuffix(resourceType, "_file_reader"):
		ns := appNamespace(app)
		return fmt.Sprintf("%s-file-readers.tf", ns)

	// Access policies and roles -> {namespace}-security.tf or element/task specific
	case resourceType == "elementum_access_policy" || resourceType == "elementum_role":
		return determineSecurityFileName(app)

	// AI search tables -> {namespace}-ai-search.tf
	case resourceType == "elementum_ai_search_table":
		ns := appNamespace(app)
		return fmt.Sprintf("%s-ai-search.tf", ns)

	// Table search tables -> table-{name}.tf (grouped with parent table)
	case resourceType == "elementum_table_search_table":
		return fmt.Sprintf("table-%s.tf", SanitizeFileName(resourceName))

	// Relationships -> same file as parent app
	case resourceType == "elementum_relationship":
		return determineAppFileName(app)

	// Widgets -> same file as parent
	case resourceType == "elementum_widget":
		return determineAppFileName(app)

	// Tables
	case resourceType == "elementum_table":
		return fmt.Sprintf("table-%s.tf", SanitizeFileName(resourceName))

	// Datamines
	case resourceType == "elementum_datamine":
		return fmt.Sprintf("datamine-%s.tf", SanitizeFileName(resourceName))

	// App, fields, layouts, flows, views -> app-{name}.tf
	case resourceType == "elementum_app" ||
		strings.HasSuffix(resourceType, "_field") ||
		resourceType == "elementum_layout" ||
		resourceType == "elementum_flow":
		return determineAppFileName(app)

	case resourceType == "elementum_view":
		ns := appNamespace(app)
		return fmt.Sprintf("%s-view.tf", ns)

	// Data sources go to data.tf
	case block.Type == "data":
		return "data.tf"

	default:
		return determineAppFileName(app)
	}
}

func buildFileName(category, groupKey string) string {
	if groupKey != "" {
		return fmt.Sprintf("%s-%s.tf", category, SanitizeFileName(groupKey))
	}
	return fmt.Sprintf("%s.tf", category)
}

func determineAppFileName(app *discovery.App) string {
	if app == nil {
		return "main.tf"
	}
	ns := appNamespace(app)
	return fmt.Sprintf("app-%s.tf", ns)
}

func determineSecurityFileName(app *discovery.App) string {
	ns := appNamespace(app)
	return fmt.Sprintf("%s-security.tf", ns)
}

func appNamespace(app *discovery.App) string {
	if app == nil {
		return "app"
	}
	if app.Namespace != "" {
		return SanitizeFileName(app.Namespace)
	}
	return SanitizeFileName(app.Name)
}

// findAutomationNameForTrigger finds the automation resource name for a trigger resource name.
func findAutomationNameForTrigger(triggerResourceName string, imports []ImportBlock, app *discovery.App) string {
	if app == nil {
		return ""
	}

	// Find the trigger ID from imports
	for _, imp := range imports {
		if imp.ResourceName != triggerResourceName {
			continue
		}
		if !strings.HasSuffix(imp.ResourceType, "_trigger") {
			continue
		}
		// Get the automation ID from the import ID (format: automationID:triggerID)
		parts := strings.Split(imp.ID, ":")
		if len(parts) >= 1 {
			autoID := parts[0]
			// Find the automation
			for _, auto := range app.AllAutomations() {
				if auto.ID == autoID {
					autoResourceName := findImportResourceName(imports, "elementum_automation", auto.ID)
					if autoResourceName != "" {
						return autoResourceName
					}
					return SanitizeName(auto.Name)
				}
			}
		}
	}
	return ""
}

// findAutomationNameForTask finds the automation resource name for a task resource name.
func findAutomationNameForTask(taskResourceName string, imports []ImportBlock, app *discovery.App) string {
	if app == nil {
		return ""
	}

	for _, imp := range imports {
		if imp.ResourceName != taskResourceName {
			continue
		}
		if !strings.HasSuffix(imp.ResourceType, "_task") {
			continue
		}
		// Task import ID format is {workflow_id}:{task_id}
		parts := strings.Split(imp.ID, ":")
		if len(parts) >= 1 {
			workflowID := parts[0]
			// Find the automation that owns this workflow
			for _, auto := range app.AllAutomations() {
				if auto.WorkflowID == workflowID {
					autoResourceName := findImportResourceName(imports, "elementum_automation", auto.ID)
					if autoResourceName != "" {
						return autoResourceName
					}
					return SanitizeName(auto.Name)
				}
			}
		}
	}
	return ""
}

// findAgentNameForTool finds the agent resource name for an agent tool resource name.
func findAgentNameForTool(toolResourceName string, imports []ImportBlock, app *discovery.App) string {
	if app == nil {
		return ""
	}

	for _, imp := range imports {
		if imp.ResourceName != toolResourceName {
			continue
		}
		if !strings.Contains(imp.ResourceType, "_agent_") {
			continue
		}
		parts := strings.Split(imp.ID, ":")
		if len(parts) >= 1 {
			agentID := parts[0]
			for _, agent := range app.AllAgents() {
				if agent.ID == agentID {
					agentResourceName := findImportResourceName(imports, "elementum_agent", agent.ID)
					if agentResourceName != "" {
						return agentResourceName
					}
					return SanitizeName(agent.Name)
				}
			}
		}
	}
	return ""
}

// findImportResourceName finds the resource name for a given resource type and ID.
func findImportResourceName(imports []ImportBlock, resourceType, id string) string {
	for _, imp := range imports {
		if imp.ResourceType == resourceType && strings.Contains(imp.ID, id) {
			return imp.ResourceName
		}
	}
	return ""
}

// SanitizeFileName sanitizes a name for use in filenames.
func SanitizeFileName(name string) string {
	// Reuse the existing SanitizeName but also handle path-unsafe chars
	sanitized := SanitizeName(name)
	sanitized = strings.ReplaceAll(sanitized, "/", "-")
	sanitized = strings.ReplaceAll(sanitized, "\\", "-")
	return sanitized
}

// DeduplicateBlocks removes duplicate blocks based on (type, labels) key.
// First occurrence wins. This replaces the WrittenResources tracker.
func DeduplicateBlocks(blocks []*HCLBlock) []*HCLBlock {
	seen := make(map[string]bool)
	var result []*HCLBlock

	for _, block := range blocks {
		key := blockKey(block)
		if key == "" {
			result = append(result, block)
			continue
		}
		if seen[key] {
			continue
		}
		seen[key] = true
		result = append(result, block)
	}
	return result
}

// SortBlocksByType sorts blocks by resource type then resource name.
// Ensures deterministic output ordering.
func SortBlocksByType(blocks []*HCLBlock) {
	sort.SliceStable(blocks, func(i, j int) bool {
		ki := blockSortKey(blocks[i])
		kj := blockSortKey(blocks[j])
		return ki < kj
	})
}

func blockSortKey(b *HCLBlock) string {
	if len(b.Labels) >= 2 {
		key := b.Labels[0] + "." + b.Labels[1]
		// workflow_publish should always be last in each file (it depends on all tasks)
		if b.Labels[0] == "elementum_workflow_publish" {
			return "~" + key // ~ sorts after all letters
		}
		return key
	}
	if len(b.Labels) >= 1 {
		return b.Labels[0]
	}
	return b.Type
}
