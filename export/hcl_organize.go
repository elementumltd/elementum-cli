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
		case strings.HasPrefix(group.FileName, "skill-"):
			result.AgentFiles = append(result.AgentFiles, fg)
		case strings.HasPrefix(group.FileName, "ai-search-"):
			result.AISearchTableFiles = append(result.AISearchTableFiles, fg)
		case group.FileName == "a2a-skills.tf":
			result.AgentFiles = append(result.AgentFiles, fg)
		case group.FileName == "skills.tf":
			result.AgentFiles = append(result.AgentFiles, fg)
		case group.FileName == "file-readers.tf":
			result.FileReaderFiles = append(result.FileReaderFiles, fg)
		case group.FileName == "views.tf":
			result.ViewFiles = append(result.ViewFiles, fg)
		case strings.HasSuffix(group.FileName, "-view.tf"):
			result.ViewFiles = append(result.ViewFiles, fg)
		case strings.HasSuffix(group.FileName, "-file-readers.tf"):
			result.FileReaderFiles = append(result.FileReaderFiles, fg)
		case strings.HasSuffix(group.FileName, "-security.tf"):
			result.SecurityFiles = append(result.SecurityFiles, fg)
		case strings.HasSuffix(group.FileName, "-ai-search.tf"):
			result.AISearchTableFiles = append(result.AISearchTableFiles, fg)
		case group.FileName == "data.tf":
			// Append IR-generated data sources to DataSourcesFile
			if result.DataSourcesFile != "" {
				result.DataSourcesFile += "\n"
			}
			result.DataSourcesFile += group.Serialize()
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
		// Use the platform's original automation name (CamelCase preserved) so
		// SanitizeFileName can split word boundaries — the resource name has
		// already been lowercased and lost them.
		if originalName := lookupAutomationOriginalName(resourceName, imports, app); originalName != "" {
			return fmt.Sprintf("automation-%s.tf", SanitizeFileName(originalName))
		}
		return fmt.Sprintf("automation-%s.tf", SanitizeFileName(resourceName))
	case strings.HasSuffix(resourceType, "_trigger"):
		if originalName := findAutomationOriginalNameForTrigger(resourceName, imports, app); originalName != "" {
			return fmt.Sprintf("automation-%s.tf", SanitizeFileName(originalName))
		}
		autoName := findAutomationNameForTrigger(resourceName, imports, app)
		if autoName != "" {
			return fmt.Sprintf("automation-%s.tf", SanitizeFileName(autoName))
		}
		return "automations.tf"
	case strings.HasSuffix(resourceType, "_task"):
		if originalName := findAutomationOriginalNameForTask(resourceName, imports, app); originalName != "" {
			return fmt.Sprintf("automation-%s.tf", SanitizeFileName(originalName))
		}
		autoName := findAutomationNameForTask(resourceName, imports, app)
		if autoName != "" {
			return fmt.Sprintf("automation-%s.tf", SanitizeFileName(autoName))
		}
		return "automations.tf"
	case resourceType == "elementum_workflow_publish":
		// workflow_publish resource name matches automation resource name.
		if originalName := lookupAutomationOriginalName(resourceName, imports, app); originalName != "" {
			return fmt.Sprintf("automation-%s.tf", SanitizeFileName(originalName))
		}
		return fmt.Sprintf("automation-%s.tf", SanitizeFileName(resourceName))

	// Agents and tools -> agent-{name}.tf
	case resourceType == "elementum_agent":
		if originalName := lookupAgentOriginalName(resourceName, imports, app); originalName != "" {
			return fmt.Sprintf("agent-%s.tf", stripAgentSuffix(SanitizeFileName(originalName)))
		}
		return fmt.Sprintf("agent-%s.tf", SanitizeFileName(stripAgentSuffix(resourceName)))
	case strings.Contains(resourceType, "_agent_") && strings.HasSuffix(resourceType, "_tool"):
		if originalName := findAgentOriginalNameForTool(resourceName, imports, app); originalName != "" {
			return fmt.Sprintf("agent-%s.tf", stripAgentSuffix(SanitizeFileName(originalName)))
		}
		agentName := findAgentNameForTool(resourceName, imports, app)
		if agentName != "" {
			return fmt.Sprintf("agent-%s.tf", SanitizeFileName(stripAgentSuffix(agentName)))
		}
		return "agents.tf"

	// A2A skills (on agents) -> a2a-skills.tf
	case resourceType == "elementum_agent_a2a_skill":
		return "a2a-skills.tf"

	// Phone services attach to an agent — route them into the agent's file.
	case resourceType == "elementum_phone_service":
		agentName := findAgentNameForPhoneService(resourceName, imports, app)
		if agentName != "" {
			return fmt.Sprintf("agent-%s.tf", stripAgentSuffix(SanitizeFileName(agentName)))
		}
		return "agents.tf"

	// Agentic skills and their tools -> skill-{name}.tf
	case resourceType == "elementum_agentic_skill":
		return fmt.Sprintf("skill-%s.tf", SanitizeFileName(resourceName))
	case resourceType == "elementum_agentic_skill_tool":
		skillName := findSkillNameForTool(resourceName, imports)
		if skillName != "" {
			return fmt.Sprintf("skill-%s.tf", SanitizeFileName(skillName))
		}
		return "skills.tf"

	// File readers -> file-readers.tf (flat, no namespace prefix — truth convention)
	case strings.HasSuffix(resourceType, "_file_reader"):
		return "file-readers.tf"

	// Access policies and roles -> {namespace}-security.tf or element/task specific
	case resourceType == "elementum_access_policy" || resourceType == "elementum_role":
		return determineSecurityFileName(app)

	// AI search tables -> ai-search-{name}.tf (one file per table)
	case resourceType == "elementum_ai_search_table":
		return fmt.Sprintf("ai-search-%s.tf", SanitizeFileName(resourceName))

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

	// Elements -> element-{name}.tf (truth convention).
	case resourceType == "elementum_element":
		return fmt.Sprintf("element-%s.tf", SanitizeFileName(resourceName))

	// App, fields, layouts, flows, views -> app-{name}.tf
	case resourceType == "elementum_app" ||
		strings.HasSuffix(resourceType, "_field") ||
		resourceType == "elementum_layout" ||
		resourceType == "elementum_flow":
		return determineAppFileName(app)

	case resourceType == "elementum_view":
		// Truth puts all views in a single views.tf.
		return "views.tf"

	// View-order pins go alongside the views they order.
	case resourceType == "elementum_managed_view_order":
		return "views.tf"

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

// lookupAutomationOriginalName returns the automation's original platform name
// (with CamelCase/spacing preserved) for a given Terraform resource name.
// Returns "" if the automation cannot be resolved.
//
// Automation import IDs are formatted `<app_id>:<automation_id>` — not just
// the automation ID — so we match by suffix/contains against the automation's
// ID.
func lookupAutomationOriginalName(resourceName string, imports []ImportBlock, app *discovery.App) string {
	if app == nil {
		return ""
	}
	for _, imp := range imports {
		if imp.ResourceType != "elementum_automation" || imp.ResourceName != resourceName {
			continue
		}
		for _, auto := range app.AllAutomations() {
			if auto.ID != "" && strings.Contains(imp.ID, auto.ID) {
				return auto.Name
			}
		}
	}
	return ""
}

// findAutomationOriginalNameForTrigger is like findAutomationNameForTrigger but
// returns the platform's original automation name instead of the sanitized
// Terraform resource name.
func findAutomationOriginalNameForTrigger(triggerResourceName string, imports []ImportBlock, app *discovery.App) string {
	if app == nil {
		return ""
	}
	for _, imp := range imports {
		if imp.ResourceName != triggerResourceName {
			continue
		}
		if !strings.HasSuffix(imp.ResourceType, "_trigger") {
			continue
		}
		parts := strings.Split(imp.ID, ":")
		if len(parts) == 0 {
			continue
		}
		autoID := parts[0]
		for _, auto := range app.AllAutomations() {
			if auto.ID == autoID {
				return auto.Name
			}
		}
	}
	return ""
}

// findAutomationOriginalNameForTask is like findAutomationNameForTask but
// returns the platform's original automation name.
func findAutomationOriginalNameForTask(taskResourceName string, imports []ImportBlock, app *discovery.App) string {
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
		parts := strings.Split(imp.ID, ":")
		if len(parts) == 0 {
			continue
		}
		workflowID := parts[0]
		for _, auto := range app.AllAutomations() {
			if auto.WorkflowID == workflowID {
				return auto.Name
			}
		}
	}
	return ""
}

// lookupAgentOriginalName returns the agent's original platform name. Agent
// import IDs are `<app_id>:<agent_id>` — match by contains.
func lookupAgentOriginalName(resourceName string, imports []ImportBlock, app *discovery.App) string {
	if app == nil {
		return ""
	}
	for _, imp := range imports {
		if imp.ResourceType != "elementum_agent" || imp.ResourceName != resourceName {
			continue
		}
		for _, agent := range app.AllAgents() {
			if agent.ID != "" && strings.Contains(imp.ID, agent.ID) {
				return agent.Name
			}
		}
	}
	return ""
}

// findAgentOriginalNameForTool returns the agent's original platform name given
// a tool resource name.
func findAgentOriginalNameForTool(toolResourceName string, imports []ImportBlock, app *discovery.App) string {
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
		if len(parts) == 0 {
			continue
		}
		agentID := parts[0]
		for _, agent := range app.AllAgents() {
			if agent.ID == agentID {
				return agent.Name
			}
		}
	}
	return ""
}

// findSkillNameForTool returns the skill resource name that owns a given
// agentic_skill_tool resource. Skill tool import IDs are typically
// {skill_id}:{tool_id}; we look up the parent skill via the imports list.
// Returns "" if unresolved (tool falls back to skills.tf).
func findSkillNameForTool(toolResourceName string, imports []ImportBlock) string {
	for _, imp := range imports {
		if imp.ResourceName != toolResourceName {
			continue
		}
		if imp.ResourceType != "elementum_agentic_skill_tool" {
			continue
		}
		parts := strings.Split(imp.ID, ":")
		if len(parts) == 0 {
			continue
		}
		skillID := parts[0]
		if skillResName := findImportResourceName(imports, "elementum_agentic_skill", skillID); skillResName != "" {
			return skillResName
		}
	}
	return ""
}

// findAgentNameForPhoneService returns the original (platform) agent name for
// the agent a phone-service resource is attached to. Phone services are
// one-per-agent; we walk the app's services, match by the imports' resource
// name, then resolve to the parent agent.
func findAgentNameForPhoneService(resourceName string, imports []ImportBlock, app *discovery.App) string {
	if app == nil {
		return ""
	}
	for _, ps := range app.PhoneServices {
		importName := findImportResourceName(imports, "elementum_phone_service", ps.ID)
		if importName != resourceName {
			continue
		}
		// Resolve the parent agent's original name so SanitizeFileName can
		// split CamelCase the same way the agent's own filename does.
		for _, agent := range app.AllAgents() {
			if agent.ID == ps.AgentID {
				return agent.Name
			}
		}
	}
	return ""
}

// stripAgentSuffix drops a trailing "-agent" from a kebab-cased filename
// fragment so "intake-agent" becomes "intake". Truth author convention: the
// file is named after the agent's purpose, not the redundant "Agent" suffix.
// Applied AFTER kebab-casing so it also covers "IntakeAgent"/"intake_agent"
// inputs that both end up as "intake-agent".
func stripAgentSuffix(name string) string {
	const suf = "-agent"
	if strings.HasSuffix(name, suf) && len(name) > len(suf) {
		return name[:len(name)-len(suf)]
	}
	return name
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

// splitCamelCase inserts a single space at each CamelCase word boundary so
// callers can then lowercase and rejoin with their preferred separator
// (dash for filenames, underscore for HCL identifiers). See SanitizeFileName
// for the three splitting rules.
func splitCamelCase(name string) string {
	runes := []rune(name)
	isUpper := func(r rune) bool { return r >= 'A' && r <= 'Z' }
	isLower := func(r rune) bool { return r >= 'a' && r <= 'z' }

	// upperRunLen counts consecutive uppercase letters starting at position i.
	upperRunLen := func(i int) int {
		n := 0
		for j := i; j < len(runes) && isUpper(runes[j]); j++ {
			n++
		}
		return n
	}

	var out strings.Builder
	for i, r := range runes {
		if i > 0 && isUpper(r) {
			prev := runes[i-1]
			var next rune
			hasNext := i+1 < len(runes)
			if hasNext {
				next = runes[i+1]
			}
			switch {
			case isLower(prev) && (!hasNext || isLower(next)):
				out.WriteRune(' ')
			case isUpper(prev) && hasNext && isLower(next):
				out.WriteRune(' ')
			case isLower(prev) && hasNext && isUpper(next) && upperRunLen(i) >= 3:
				out.WriteRune(' ')
			}
		}
		out.WriteRune(r)
	}
	return out.String()
}

// SanitizeFileName produces a kebab-case filename fragment from an arbitrary
// platform name. Unlike SanitizeName (which must emit a valid HCL identifier
// and so uses underscores), filenames are free to keep dashes — and our hand-
// authored truth uses dashes.
//
// The tricky bit is CamelCase: "ValidateAdGroupRequest" should become
// "validate-ad-group-request", but "WaaSRequest" should become "waas-request"
// (the acronym stays together) and "A2A" should stay "a2a" rather than split
// on the digit boundary.
//
// Splitting rules — insert a space before an uppercase letter at position i:
//  1. prev is lowercase AND (next is lowercase OR end). Normal CamelCase
//     word boundary, e.g. "validate|Ad".
//  2. prev is uppercase AND next is lowercase. Acronym-to-word boundary,
//     e.g. "WaaS|Request" (split between S and R), "DL|Update" (L→U).
//  3. prev is lowercase AND next is uppercase AND the uppercase run that
//     starts at position i is at least 3 letters long. Word-to-acronym
//     boundary where the acronym is embedded in the middle of the name,
//     e.g. "Validate|DLUpdate" (the DLU run has length 3 because U is
//     also uppercase). Without this rule "ValidateDLUpdateRequest" would
//     collapse to "validatedl-update-request". We require ≥3 rather than
//     ≥2 so that "WaaSRequest" (WaaS has only S,R as the run of 2) does
//     NOT split between "a" and "S" — S is the tail of "WaaS", not the
//     start of a new acronym.
//
// Digit-adjacent uppercase transitions deliberately do NOT split so that
// "A2A" survives as a single token.
func SanitizeFileName(name string) string {
	// 1. Insert spaces at CamelCase boundaries (see splitCamelCase doc).
	s := splitCamelCase(name)

	// 2. Lowercase and normalise separators to a single dash.
	s = strings.ToLower(s)
	for _, sep := range []string{" ", "_", ".", ":", "/", "\\"} {
		s = strings.ReplaceAll(s, sep, "-")
	}

	// 3. Strip anything that isn't [a-z0-9-].
	var out strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			out.WriteRune(r)
		}
	}
	s = out.String()

	// 4. Collapse runs of dashes and trim leading/trailing.
	for strings.Contains(s, "--") {
		s = strings.ReplaceAll(s, "--", "-")
	}
	s = strings.Trim(s, "-")

	if s == "" {
		s = "resource"
	}
	return s
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
		resourceType := b.Labels[0]
		key := resourceType + "." + b.Labels[1]
		// Group per-automation blocks in a readable order:
		//   automation → triggers → tasks → switch_case → workflow_publish
		// Truth HCL follows this convention; without explicit prefixes, pure
		// alphabetical ordering interleaves tasks before the automation
		// because `elementum_ai_search_table_task` < `elementum_automation`.
		prefix := "5" // fallback bucket
		switch {
		case resourceType == "elementum_workflow_publish":
			prefix = "9" // last, depends on all tasks
		case resourceType == "elementum_switch_case":
			prefix = "4" // belongs with its parent switch_task
		case strings.HasSuffix(resourceType, "_task"):
			prefix = "3"
		case strings.HasSuffix(resourceType, "_trigger"):
			prefix = "2"
		case resourceType == "elementum_automation":
			prefix = "1"
		}
		return prefix + key
	}
	if len(b.Labels) >= 1 {
		return b.Labels[0]
	}
	return b.Type
}
