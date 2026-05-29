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
	"github.com/elementumltd/elementum-cli/discovery"
	"github.com/elementumltd/elementum-cli/logger"
)

// ResourceGenerator is the interface implemented by all HCL generators
// that produce IR blocks. Each generator handles one category of resources
// (e.g., automations, agents, roles, file readers).
//
// Implementing this interface enables registration in a GeneratorRegistry,
// which replaces the explicit wiring in HCLGenerator's constructor and
// delegate methods.
type ResourceGenerator interface {
	// GenerateIR produces HCL blocks for all resources this generator handles.
	GenerateIR() []*HCLBlock

	// CanGenerate returns true if this generator has resources to produce.
	// Used to skip generators that have nothing to generate (e.g., no agents).
	CanGenerate() bool

	// Name returns a human-readable name for logging (e.g., "automations", "agents").
	Name() string
}

// GeneratorRegistry holds a collection of resource generators and
// orchestrates IR generation across all of them.
//
// Usage:
//
//	registry := NewGeneratorRegistry()
//	registry.Register(automationGen)
//	registry.Register(agentGen)
//	blocks := registry.GenerateAll()
type GeneratorRegistry struct {
	generators []ResourceGenerator
}

// NewGeneratorRegistry creates an empty generator registry.
func NewGeneratorRegistry() *GeneratorRegistry {
	return &GeneratorRegistry{}
}

// Register adds a generator to the registry.
func (r *GeneratorRegistry) Register(g ResourceGenerator) {
	r.generators = append(r.generators, g)
}

// GenerateAll runs all registered generators and returns the combined IR blocks.
// Generators that report CanGenerate() == false are skipped.
func (r *GeneratorRegistry) GenerateAll() []*HCLBlock {
	var blocks []*HCLBlock
	for _, g := range r.generators {
		if !g.CanGenerate() {
			logger.Debug("skipping generator %s: nothing to generate", g.Name())
			continue
		}
		genBlocks := g.GenerateIR()
		logger.Debug("generator %s produced %d blocks", g.Name(), len(genBlocks))
		blocks = append(blocks, genBlocks...)
	}
	return blocks
}

// GeneratorCount returns the number of registered generators.
func (r *GeneratorRegistry) GeneratorCount() int {
	return len(r.generators)
}

// --- Adapter types that wrap existing generators to implement ResourceGenerator ---

// AutomationGeneratorAdapter wraps AutomationHCLGenerator to implement ResourceGenerator.
type AutomationGeneratorAdapter struct {
	gen *AutomationHCLGenerator
}

func NewAutomationGeneratorAdapter(gen *AutomationHCLGenerator) *AutomationGeneratorAdapter {
	return &AutomationGeneratorAdapter{gen: gen}
}

func (a *AutomationGeneratorAdapter) GenerateIR() []*HCLBlock { return a.gen.GenerateAllIR() }
func (a *AutomationGeneratorAdapter) CanGenerate() bool {
	return a.gen.app != nil && len(a.gen.app.AllAutomations()) > 0
}
func (a *AutomationGeneratorAdapter) Name() string { return "automations" }

// AgentGeneratorAdapter wraps AgentHCLGenerator to implement ResourceGenerator.
type AgentGeneratorAdapter struct {
	gen *AgentHCLGenerator
}

func NewAgentGeneratorAdapter(gen *AgentHCLGenerator) *AgentGeneratorAdapter {
	return &AgentGeneratorAdapter{gen: gen}
}

func (a *AgentGeneratorAdapter) GenerateIR() []*HCLBlock { return a.gen.GenerateAllIR() }
func (a *AgentGeneratorAdapter) CanGenerate() bool {
	return a.gen.app != nil && len(a.gen.app.AllAgents()) > 0
}
func (a *AgentGeneratorAdapter) Name() string { return "agents" }

// SkillGeneratorAdapter wraps SkillHCLGenerator to implement ResourceGenerator.
type SkillGeneratorAdapter struct {
	gen *SkillHCLGenerator
}

func NewSkillGeneratorAdapter(gen *SkillHCLGenerator) *SkillGeneratorAdapter {
	return &SkillGeneratorAdapter{gen: gen}
}

func (a *SkillGeneratorAdapter) GenerateIR() []*HCLBlock { return a.gen.GenerateAllIR() }
func (a *SkillGeneratorAdapter) CanGenerate() bool {
	if a.gen.app == nil {
		return false
	}
	if len(a.gen.app.Skills) > 0 {
		return true
	}
	for _, elem := range a.gen.app.DiscoveredElements {
		if len(elem.Skills) > 0 {
			return true
		}
	}
	// A2A skills live on agents — check those too.
	for _, agent := range a.gen.app.AllAgents() {
		if len(agent.A2ASkills) > 0 {
			return true
		}
	}
	return false
}
func (a *SkillGeneratorAdapter) Name() string { return "skills" }

// PhoneServiceGeneratorAdapter wraps PhoneServiceHCLGenerator.
type PhoneServiceGeneratorAdapter struct {
	gen *PhoneServiceHCLGenerator
}

func NewPhoneServiceGeneratorAdapter(gen *PhoneServiceHCLGenerator) *PhoneServiceGeneratorAdapter {
	return &PhoneServiceGeneratorAdapter{gen: gen}
}

func (a *PhoneServiceGeneratorAdapter) GenerateIR() []*HCLBlock { return a.gen.GenerateAllIR() }
func (a *PhoneServiceGeneratorAdapter) CanGenerate() bool {
	return a.gen.app != nil && len(a.gen.app.PhoneServices) > 0
}
func (a *PhoneServiceGeneratorAdapter) Name() string { return "phone_services" }

// ManagedViewOrderGeneratorAdapter wraps ManagedViewOrderHCLGenerator.
type ManagedViewOrderGeneratorAdapter struct {
	gen *ManagedViewOrderHCLGenerator
}

func NewManagedViewOrderGeneratorAdapter(gen *ManagedViewOrderHCLGenerator) *ManagedViewOrderGeneratorAdapter {
	return &ManagedViewOrderGeneratorAdapter{gen: gen}
}

func (a *ManagedViewOrderGeneratorAdapter) GenerateIR() []*HCLBlock { return a.gen.GenerateAllIR() }
func (a *ManagedViewOrderGeneratorAdapter) CanGenerate() bool {
	return a.gen.app != nil && a.gen.app.ManagedViewOrder != nil && len(a.gen.app.ManagedViewOrder.ViewIDs) > 0
}
func (a *ManagedViewOrderGeneratorAdapter) Name() string { return "managed_view_order" }

// RelationshipGeneratorAdapter wraps RelationshipHCLGenerator to implement ResourceGenerator.
type RelationshipGeneratorAdapter struct {
	gen *RelationshipHCLGenerator
}

func NewRelationshipGeneratorAdapter(gen *RelationshipHCLGenerator) *RelationshipGeneratorAdapter {
	return &RelationshipGeneratorAdapter{gen: gen}
}

func (a *RelationshipGeneratorAdapter) GenerateIR() []*HCLBlock { return a.gen.GenerateAllIR() }
func (a *RelationshipGeneratorAdapter) CanGenerate() bool {
	return a.gen.app != nil && len(a.gen.app.Relationships) > 0
}
func (a *RelationshipGeneratorAdapter) Name() string { return "relationships" }

// FileReaderGeneratorAdapter wraps FileReaderHCLGenerator to implement ResourceGenerator.
type FileReaderGeneratorAdapter struct {
	gen *FileReaderHCLGenerator
}

func NewFileReaderGeneratorAdapter(gen *FileReaderHCLGenerator) *FileReaderGeneratorAdapter {
	return &FileReaderGeneratorAdapter{gen: gen}
}

func (a *FileReaderGeneratorAdapter) GenerateIR() []*HCLBlock { return a.gen.GenerateAllIR() }
func (a *FileReaderGeneratorAdapter) CanGenerate() bool {
	return a.gen.app != nil && len(a.gen.app.AIFileReaders) > 0
}
func (a *FileReaderGeneratorAdapter) Name() string { return "file_readers" }

// AccessPolicyGeneratorAdapter wraps AccessPolicyHCLGenerator to implement ResourceGenerator.
type AccessPolicyGeneratorAdapter struct {
	gen *AccessPolicyHCLGenerator
}

func NewAccessPolicyGeneratorAdapter(gen *AccessPolicyHCLGenerator) *AccessPolicyGeneratorAdapter {
	return &AccessPolicyGeneratorAdapter{gen: gen}
}

func (a *AccessPolicyGeneratorAdapter) GenerateIR() []*HCLBlock { return a.gen.GenerateAllIR() }
func (a *AccessPolicyGeneratorAdapter) CanGenerate() bool {
	return a.gen.app != nil && len(a.gen.app.AccessPolicies) > 0
}
func (a *AccessPolicyGeneratorAdapter) Name() string { return "access_policies" }

// RoleGeneratorAdapter wraps RoleHCLGenerator to implement ResourceGenerator.
type RoleGeneratorAdapter struct {
	gen *RoleHCLGenerator
}

func NewRoleGeneratorAdapter(gen *RoleHCLGenerator) *RoleGeneratorAdapter {
	return &RoleGeneratorAdapter{gen: gen}
}

func (a *RoleGeneratorAdapter) GenerateIR() []*HCLBlock { return a.gen.GenerateAllIR() }
func (a *RoleGeneratorAdapter) CanGenerate() bool {
	return a.gen.app != nil && len(a.gen.app.Roles) > 0
}
func (a *RoleGeneratorAdapter) Name() string { return "roles" }

// WidgetGeneratorAdapter wraps WidgetHCLGenerator to implement ResourceGenerator.
type WidgetGeneratorAdapter struct {
	gen *WidgetHCLGenerator
}

func NewWidgetGeneratorAdapter(gen *WidgetHCLGenerator) *WidgetGeneratorAdapter {
	return &WidgetGeneratorAdapter{gen: gen}
}

func (a *WidgetGeneratorAdapter) GenerateIR() []*HCLBlock { return a.gen.GenerateAllIR() }
func (a *WidgetGeneratorAdapter) CanGenerate() bool {
	if a.gen.app == nil {
		return false
	}
	// Check main app widgets
	if len(a.gen.app.Widgets) > 0 {
		return true
	}
	// Check discovered apps' widgets
	for _, discoveredApp := range a.gen.app.DiscoveredApps {
		if len(discoveredApp.Widgets) > 0 {
			return true
		}
	}
	// Check discovered elements' widgets
	for _, discoveredElement := range a.gen.app.DiscoveredElements {
		if len(discoveredElement.Widgets) > 0 {
			return true
		}
	}
	// Check discovered tasks' widgets
	for _, discoveredTask := range a.gen.app.DiscoveredTasks {
		if len(discoveredTask.Widgets) > 0 {
			return true
		}
	}
	return false
}
func (a *WidgetGeneratorAdapter) Name() string { return "widgets" }

// AISearchTableGeneratorAdapter wraps AISearchTableHCLGenerator to implement ResourceGenerator.
type AISearchTableGeneratorAdapter struct {
	gen *AISearchTableHCLGenerator
}

func NewAISearchTableGeneratorAdapter(gen *AISearchTableHCLGenerator) *AISearchTableGeneratorAdapter {
	return &AISearchTableGeneratorAdapter{gen: gen}
}

func (a *AISearchTableGeneratorAdapter) GenerateIR() []*HCLBlock { return a.gen.GenerateAllIR() }
func (a *AISearchTableGeneratorAdapter) CanGenerate() bool {
	if a.gen.app == nil {
		return false
	}
	// Check main app AI search tables
	if len(a.gen.app.AISearchTables) > 0 {
		return true
	}
	// Check discovered elements' AI search tables
	for _, discoveredElement := range a.gen.app.DiscoveredElements {
		if len(discoveredElement.AISearchTables) > 0 {
			return true
		}
	}
	return false
}
func (a *AISearchTableGeneratorAdapter) Name() string { return "ai_search_tables" }

// TableSearchTableGeneratorAdapter wraps TableSearchTableHCLGenerator to implement ResourceGenerator.
type TableSearchTableGeneratorAdapter struct {
	gen *TableSearchTableHCLGenerator
}

func NewTableSearchTableGeneratorAdapter(gen *TableSearchTableHCLGenerator) *TableSearchTableGeneratorAdapter {
	return &TableSearchTableGeneratorAdapter{gen: gen}
}

func (a *TableSearchTableGeneratorAdapter) GenerateIR() []*HCLBlock { return a.gen.GenerateAllIR() }
func (a *TableSearchTableGeneratorAdapter) CanGenerate() bool {
	if a.gen.app == nil {
		return false
	}
	// Check referenced tables' search tables
	for _, table := range a.gen.app.ReferencedTables {
		if len(table.SearchTables) > 0 {
			return true
		}
	}
	// Check discovered tables' search tables
	for _, table := range a.gen.app.DiscoveredTables {
		if len(table.SearchTables) > 0 {
			return true
		}
	}
	return false
}
func (a *TableSearchTableGeneratorAdapter) Name() string { return "table_search_tables" }

// AppResourcesGeneratorAdapter wraps AppHCLGenerator to implement ResourceGenerator.
type AppResourcesGeneratorAdapter struct {
	gen *AppHCLGenerator
}

func NewAppResourcesGeneratorAdapter(gen *AppHCLGenerator) *AppResourcesGeneratorAdapter {
	return &AppResourcesGeneratorAdapter{gen: gen}
}

func (a *AppResourcesGeneratorAdapter) GenerateIR() []*HCLBlock { return a.gen.GenerateAllIR() }
func (a *AppResourcesGeneratorAdapter) CanGenerate() bool       { return a.gen.CanGenerate() }
func (a *AppResourcesGeneratorAdapter) Name() string            { return "app_resources" }

// NewAppGeneratorRegistry creates a fully-wired registry for an App export.
// All resource types are generated via CLI IR generators (no tofu dependency).
func NewAppGeneratorRegistry(app *discovery.App, imports []ImportBlock, uuidMap map[string]string) *GeneratorRegistry {
	registry := NewGeneratorRegistry()

	// App, fields, layouts, flows, views, tables, datamines, elements, tasks
	appGen := NewAppHCLGenerator(app, imports, uuidMap)
	registry.Register(NewAppResourcesGeneratorAdapter(appGen))

	// Automations (includes triggers + tasks)
	automationGen := NewAutomationHCLGenerator(app, imports, uuidMap)
	registry.Register(NewAutomationGeneratorAdapter(automationGen))

	// Agents + tools
	agentGen := NewAgentHCLGenerator(app, imports, uuidMap)
	registry.Register(NewAgentGeneratorAdapter(agentGen))

	// Agentic skills + tools
	skillGen := NewSkillHCLGenerator(app, imports, uuidMap)
	registry.Register(NewSkillGeneratorAdapter(skillGen))

	// Phone services
	phoneGen := NewPhoneServiceHCLGenerator(app, imports, uuidMap)
	registry.Register(NewPhoneServiceGeneratorAdapter(phoneGen))

	// Managed view order (one per aspect, pins view ordering)
	mvoGen := NewManagedViewOrderHCLGenerator(app, imports, uuidMap)
	registry.Register(NewManagedViewOrderGeneratorAdapter(mvoGen))

	// Relationships
	relationshipGen := NewRelationshipHCLGenerator(app, imports, uuidMap)
	registry.Register(NewRelationshipGeneratorAdapter(relationshipGen))

	// File readers
	fileReaderGen := NewFileReaderHCLGenerator(app, imports, uuidMap)
	registry.Register(NewFileReaderGeneratorAdapter(fileReaderGen))

	// Access policies
	accessPolicyGen := NewAccessPolicyHCLGenerator(app, imports, uuidMap)
	registry.Register(NewAccessPolicyGeneratorAdapter(accessPolicyGen))

	// Roles
	appResourceRef := ""
	if app != nil {
		appResourceRef = "elementum_app." + AppResourceName(app) + ".id"
	}
	roleGen := NewRoleHCLGenerator(app, imports, uuidMap, appResourceRef)
	registry.Register(NewRoleGeneratorAdapter(roleGen))

	// Widgets
	widgetGen := NewWidgetHCLGenerator(app, imports, uuidMap)
	registry.Register(NewWidgetGeneratorAdapter(widgetGen))

	// AI search tables (aspect-based)
	searchTableGen := NewAISearchTableHCLGenerator(app, imports, uuidMap)
	registry.Register(NewAISearchTableGeneratorAdapter(searchTableGen))

	// Table search tables (table-based)
	tableSearchTableGen := NewTableSearchTableHCLGenerator(app, imports, uuidMap)
	registry.Register(NewTableSearchTableGeneratorAdapter(tableSearchTableGen))

	return registry
}
