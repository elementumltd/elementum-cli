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
	"strings"

	"github.com/elementumltd/elementum-cli/discovery"
)

// SkillHCLGenerator produces IR blocks for agentic skills and their tools.
// Each `elementum_agentic_skill` gets one block, followed by one
// `elementum_agentic_skill_tool` block per tool. Filename routing in
// hcl_organize.go groups them into `skill-<name>.tf`.
type SkillHCLGenerator struct {
	app     *discovery.App
	imports []ImportBlock
	uuidMap map[string]string
}

func NewSkillHCLGenerator(app *discovery.App, imports []ImportBlock, uuidMap map[string]string) *SkillHCLGenerator {
	return &SkillHCLGenerator{app: app, imports: imports, uuidMap: uuidMap}
}

func (g *SkillHCLGenerator) GenerateAll() string {
	return SerializeBlocks(g.GenerateAllIR())
}

// GenerateAllIR walks all skills on the main app + discovered elements and
// emits one block per skill plus one per tool. Skills that have no matching
// import block are skipped (the imports list is the canonical source of
// truth for which skill IDs appear in the export).
func (g *SkillHCLGenerator) GenerateAllIR() []*HCLBlock {
	if g.app == nil {
		return nil
	}
	var blocks []*HCLBlock

	collect := func(skills []discovery.AgenticSkill, ownerRef string) {
		for i := range skills {
			skill := &skills[i]
			if b := g.generateSkillIR(skill, ownerRef); b != nil {
				blocks = append(blocks, b)
				// Tools flow in the same file as their parent skill — the
				// skill_id attribute establishes the linkage; hcl_organize
				// groups them into skill-<name>.tf via findSkillNameForTool.
				for j := range skill.Tools {
					if tb := g.generateSkillToolIR(&skill.Tools[j]); tb != nil {
						blocks = append(blocks, tb)
					}
				}
			}
		}
	}

	collect(g.app.Skills, "elementum_app."+AppResourceName(g.app))

	for _, elem := range g.app.DiscoveredElements {
		if len(elem.Skills) == 0 {
			continue
		}
		elementRef := "elementum_element." + ElementResourceName(elem)
		collect(elem.Skills, elementRef)
	}

	// A2A skills share this generator — same subsystem conceptually.
	blocks = append(blocks, g.GenerateA2ASkillsIR()...)

	return blocks
}

func (g *SkillHCLGenerator) generateSkillIR(skill *discovery.AgenticSkill, ownerRef string) *HCLBlock {
	resourceName := g.findImportedResourceName("elementum_agentic_skill", skill.ID)
	if resourceName == "" {
		resourceName = SanitizeName(skill.Name)
	}
	if resourceName == "" {
		return nil
	}

	b := NewResourceBlock("elementum_agentic_skill", resourceName)
	b.SetAttr("name", Str(skill.Name))
	b.SetAttr("owner_id", Ref(ownerRef+".id"))
	if skill.OwnerType != "" {
		b.SetAttr("owner_type", Str(skill.OwnerType))
	} else {
		b.SetAttr("owner_type", Str("APP_ASPECT"))
	}
	if skill.Status != "" {
		b.SetAttr("status", Str(skill.Status))
	}
	if skill.Type != "" {
		b.SetAttr("type", Str(skill.Type))
	}
	if skill.Description != "" {
		b.SetAttr("description", formatHCLStringIR(skill.Description))
	}
	if skill.Instructions != "" {
		b.SetAttr("instructions", formatHCLStringIR(skill.Instructions))
	}
	if skill.Properties != "" {
		// `properties` is declared as a string on the resource and Truth HCL
		// reflects whatever JSON the platform returned. Wrap via jsonencode()
		// so fmt roundtrips cleanly.
		b.SetAttr("properties", Raw("jsonencode("+skill.Properties+")"))
	}
	return b
}

func (g *SkillHCLGenerator) generateSkillToolIR(tool *discovery.AgenticSkillTool) *HCLBlock {
	resourceName := g.findImportedResourceName("elementum_agentic_skill_tool", tool.ID)
	if resourceName == "" {
		resourceName = SanitizeName(tool.Name)
	}
	if resourceName == "" {
		return nil
	}

	b := NewResourceBlock("elementum_agentic_skill_tool", resourceName)
	if skillRef := g.resolveIDRef(tool.SkillID); skillRef != "" {
		b.SetAttr("skill_id", Ref(skillRef))
	} else {
		b.SetAttr("skill_id", Str(tool.SkillID))
	}
	if tool.Type != "" {
		b.SetAttr("tool_type", Str(strings.ToUpper(tool.Type)))
	}
	b.SetAttr("name", Str(tool.Name))
	if tool.Description != "" {
		b.SetAttr("description", formatHCLStringIR(tool.Description))
	}
	if tool.Status != "" {
		b.SetAttr("status", Str(tool.Status))
	}
	if tool.StartMessage != "" {
		b.SetAttr("start_message", formatHCLStringIR(tool.StartMessage))
	}

	switch tool.Type {
	case "automation":
		if ref := g.resolveIDRef(tool.AutomationID); ref != "" {
			b.SetAttr("automation_id", Ref(ref))
		} else if tool.AutomationID != "" {
			b.SetAttr("automation_id", Str(tool.AutomationID))
		}
	case "create_record", "update_record", "search_aspect":
		if ref := g.resolveIDRef(tool.TargetAspectID); ref != "" {
			b.SetAttr("target_id", Ref(ref))
		} else if tool.TargetAspectID != "" {
			b.SetAttr("target_id", Str(tool.TargetAspectID))
		}
		if tool.QueryDescription != "" {
			b.SetAttr("query_description", formatHCLStringIR(tool.QueryDescription))
		}
		if tool.HandleDescription != "" {
			b.SetAttr("handle_description", formatHCLStringIR(tool.HandleDescription))
		}
		if tool.ResultLimit > 0 {
			b.SetAttr("result_limit", Num(float64(tool.ResultLimit)))
		}
		if fields := g.buildFieldList(tool.Fields); fields != nil {
			b.SetAttr("fields", fields)
		}
	case "search_table":
		if ref := g.resolveIDRef(tool.SearchTableID); ref != "" {
			b.SetAttr("search_table_id", Ref(ref))
		} else if tool.SearchTableID != "" {
			b.SetAttr("search_table_id", Str(tool.SearchTableID))
		}
		if tool.QueryDescription != "" {
			b.SetAttr("query_description", formatHCLStringIR(tool.QueryDescription))
		}
		if tool.ResultLimit > 0 {
			b.SetAttr("result_limit", Num(float64(tool.ResultLimit)))
		}
		if fields := g.buildFieldList(tool.Fields); fields != nil {
			b.SetAttr("fields", fields)
		}
	case "run_agent":
		if ref := g.resolveIDRef(tool.TargetAgentID); ref != "" {
			b.SetAttr("target_agent_id", Ref(ref))
		} else if tool.TargetAgentID != "" {
			b.SetAttr("target_agent_id", Str(tool.TargetAgentID))
		}
		if tool.WorkerTaskPrompt != "" {
			b.SetAttr("worker_task_prompt", formatHCLStringIR(tool.WorkerTaskPrompt))
		}
	}

	return b
}

func (g *SkillHCLGenerator) buildFieldList(fields []discovery.AgenticSkillField) HCLValue {
	if len(fields) == 0 {
		return nil
	}
	var items []HCLValue
	for _, f := range fields {
		attrs := []*HCLAttribute{}
		if ref := g.resolveIDRef(f.FieldID); ref != "" {
			attrs = append(attrs, Attr("field_id", Ref(ref)))
		} else if f.FieldID != "" {
			attrs = append(attrs, Attr("field_id", Str(f.FieldID)))
		}
		if f.Name != "" {
			attrs = append(attrs, Attr("name", Str(f.Name)))
		}
		if f.Description != "" {
			attrs = append(attrs, Attr("description", formatHCLStringIR(f.Description)))
		}
		if f.Required {
			attrs = append(attrs, Attr("required", Bool(true)))
		}
		items = append(items, Obj(attrs...))
	}
	return HCLList{Values: items}
}

// findImportedResourceName scans the imports list for a block of the given
// type whose import ID contains `id`. Returns the matching resource name or
// "" if not found.
func (g *SkillHCLGenerator) findImportedResourceName(resourceType, id string) string {
	for _, imp := range g.imports {
		if imp.ResourceType == resourceType && strings.Contains(imp.ID, id) {
			return imp.ResourceName
		}
	}
	return ""
}

// resolveIDRef returns a full Terraform reference expression (e.g.
// `elementum_automation.foo.id`) for a platform UUID if a mapping exists in
// the uuidMap — otherwise returns "". The `.id` suffix is already part of
// uuidMap entries, so we strip it from the returned value when the caller
// wants a bare ref to append `.id` themselves.
func (g *SkillHCLGenerator) resolveIDRef(id string) string {
	if id == "" {
		return ""
	}
	if ref, ok := g.uuidMap[id]; ok {
		return ref
	}
	return ""
}

// GenerateA2ASkillsIR emits one `elementum_agent_a2a_skill` block per A2A
// skill attached to any agent on the app or any discovered aspect.
// Filename routing in hcl_organize.go sends these to `a2a-skills.tf`.
func (g *SkillHCLGenerator) GenerateA2ASkillsIR() []*HCLBlock {
	if g.app == nil {
		return nil
	}
	var blocks []*HCLBlock
	for _, agent := range g.app.AllAgents() {
		for i := range agent.A2ASkills {
			if b := g.generateA2ASkillIR(&agent.A2ASkills[i]); b != nil {
				blocks = append(blocks, b)
			}
		}
	}
	return blocks
}

func (g *SkillHCLGenerator) generateA2ASkillIR(skill *discovery.AgentA2ASkill) *HCLBlock {
	resourceName := g.findImportedResourceName("elementum_agent_a2a_skill", skill.ID)
	if resourceName == "" {
		resourceName = SanitizeName(skill.Name)
	}
	if resourceName == "" {
		return nil
	}

	b := NewResourceBlock("elementum_agent_a2a_skill", resourceName)
	if agentRef := g.resolveIDRef(skill.AgentID); agentRef != "" {
		b.SetAttr("agent_id", Ref(agentRef))
	} else {
		b.SetAttr("agent_id", Str(skill.AgentID))
	}
	b.SetAttr("name", Str(skill.Name))
	if skill.Description != "" {
		b.SetAttr("description", formatHCLStringIR(skill.Description))
	}
	if len(skill.Tags) > 0 {
		b.SetAttr("tags", strList(skill.Tags))
	}
	if len(skill.Examples) > 0 {
		b.SetAttr("examples", strList(skill.Examples))
	}
	if len(skill.InputModes) > 0 {
		b.SetAttr("input_modes", strList(skill.InputModes))
	}
	if len(skill.OutputModes) > 0 {
		b.SetAttr("output_modes", strList(skill.OutputModes))
	}
	return b
}

// strList is a small helper to convert a []string into an HCLList of
// quoted string values in stable source order.
func strList(ss []string) HCLValue {
	if len(ss) == 0 {
		return nil
	}
	vals := make([]HCLValue, 0, len(ss))
	for _, s := range ss {
		vals = append(vals, Str(s))
	}
	return HCLList{Values: vals}
}
