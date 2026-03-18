// Copyright (c) ElementumAI, Inc.
// SPDX-License-Identifier: MPL-2.0

package export

import (
	"fmt"
	"strings"

	"github.com/elementumltd/elementum-cli/discovery"
	"github.com/elementumltd/elementum-cli/internal/client"
)

// TriggerHCLGenerator generates HCL for trigger resources
type TriggerHCLGenerator struct {
	imports     []ImportBlock
	uuidMap     map[string]string
	automations []discovery.Automation
}

// NewTriggerHCLGenerator creates a new trigger HCL generator
func NewTriggerHCLGenerator(imports []ImportBlock, uuidMap map[string]string, automations []discovery.Automation) *TriggerHCLGenerator {
	return &TriggerHCLGenerator{
		imports:     imports,
		uuidMap:     uuidMap,
		automations: automations,
	}
}

// GenerateAll generates HCL for all triggers in all automations
func (g *TriggerHCLGenerator) GenerateAll() string {
	return SerializeBlocks(g.GenerateAllIR())
}

// resolveRef resolves an ID to a terraform reference based on the reference type
func (g *TriggerHCLGenerator) resolveRef(id string, refType string) string {
	// First check our UUID map
	if ref, ok := g.uuidMap[id]; ok {
		return ref
	}

	// Try to find in imports
	switch refType {
	case "datamine":
		for _, imp := range g.imports {
			if imp.ResourceType == "elementum_datamine" && strings.Contains(imp.ID, id) {
				return "elementum_datamine." + imp.ResourceName + ".id"
			}
		}
	case "email_alias":
		for _, imp := range g.imports {
			if imp.ResourceType == "elementum_email_alias" && strings.Contains(imp.ID, id) {
				return "elementum_email_alias." + imp.ResourceName + ".id"
			}
		}
	case "approval_template":
		for _, imp := range g.imports {
			if imp.ResourceType == "elementum_approval_chain_template" && strings.Contains(imp.ID, id) {
				return "elementum_approval_chain_template." + imp.ResourceName + ".id"
			}
		}
		// Also check approval_process
		for _, imp := range g.imports {
			if imp.ResourceType == "elementum_approval_process" && strings.Contains(imp.ID, id) {
				return "elementum_approval_process." + imp.ResourceName + ".id"
			}
		}
	}

	// General fallback
	for _, imp := range g.imports {
		if strings.Contains(imp.ID, id) {
			return imp.ResourceType + "." + imp.ResourceName + ".id"
		}
	}

	return fmt.Sprintf("%q", id)
}

// GenerateAllIR generates IR blocks for all triggers
func (g *TriggerHCLGenerator) GenerateAllIR() []*HCLBlock {
	var blocks []*HCLBlock

	for _, automation := range g.automations {
		if !automation.HasPublished || automation.Status != "ACTIVE" {
			continue
		}

		automationResourceName := ""
		for _, imp := range g.imports {
			if imp.ResourceType == "elementum_automation" && strings.Contains(imp.ID, automation.ID) {
				automationResourceName = imp.ResourceName
				break
			}
		}
		if automationResourceName == "" {
			continue
		}

		for _, trigger := range automation.Triggers {
			if trigger.Type == "unknown" {
				continue
			}
			b := g.GenerateTriggerIR(&trigger, automationResourceName)
			if b != nil {
				blocks = append(blocks, b)
			}
		}
	}

	return blocks
}

// GenerateTriggerIR generates an IR block for a single trigger resource
func (g *TriggerHCLGenerator) GenerateTriggerIR(trigger *discovery.Trigger, automationResourceName string) *HCLBlock {
	triggerType := "elementum_" + trigger.Type + "_trigger"
	var resourceName string
	for _, imp := range g.imports {
		if imp.ResourceType == triggerType && strings.HasSuffix(imp.ID, ":"+trigger.ID) {
			resourceName = imp.ResourceName
			break
		}
	}
	if resourceName == "" {
		return nil
	}

	b := NewResourceBlock(triggerType, resourceName)
	b.SetAttr("automation_id", Ref("elementum_automation."+automationResourceName+".id"))

	// Add trigger-type-specific attributes from the type registry
	config, ok := client.GetTriggerTypeConfig(trigger.Type)
	if !ok {
		return b
	}

	for _, field := range config.Fields {
		if field.IsComputed {
			continue
		}
		g.addTriggerFieldIR(b, trigger, field)
	}

	return b
}

// addTriggerFieldIR adds a single trigger field to the IR block
func (g *TriggerHCLGenerator) addTriggerFieldIR(b *HCLBlock, trigger *discovery.Trigger, field client.TriggerFieldConfig) {
	if trigger.RawData == nil {
		if field.Name == "datamine_id" && trigger.DatamineID != "" {
			ref := g.resolveRef(trigger.DatamineID, "datamine")
			b.SetAttr("datamine_id", refOrStr(ref))
		}
		return
	}

	value := extractValueFromPath(trigger.RawData, field.GraphQLPath)
	if value == nil {
		return
	}

	switch {
	case field.IsArray:
		arr, ok := value.([]interface{})
		if !ok || len(arr) == 0 {
			return
		}
		var ids []string
		for _, item := range arr {
			switch v := item.(type) {
			case string:
				ids = append(ids, v)
			case map[string]interface{}:
				if field.ArrayElementPath != "" {
					if id, ok := v[field.ArrayElementPath].(string); ok && id != "" {
						ids = append(ids, id)
					}
				}
			}
		}
		if len(ids) > 0 {
			var vals []HCLValue
			for _, id := range ids {
				ref := g.resolveRef(id, "field")
				vals = append(vals, refOrStr(ref))
			}
			b.SetAttr(field.Name, List(vals...))
		}

	case field.IsReference:
		if strVal, ok := value.(string); ok && strVal != "" {
			ref := g.resolveRef(strVal, field.ReferenceType)
			b.SetAttr(field.Name, refOrStr(ref))
		}

	case field.Name == "filter":
		if filterData, ok := value.(map[string]interface{}); ok && len(filterData) > 0 {
			filterVal := GenerateFilterIR(filterData, g.uuidMap)
			if filterVal != nil {
				b.SetAttr("filter", filterVal)
			}
		}

	case field.Name == "parameters":
		params, ok := trigger.RawData["parameters"].([]interface{})
		if !ok || len(params) == 0 {
			return
		}
		var paramObjs []HCLValue
		for _, paramI := range params {
			param, ok := paramI.(map[string]interface{})
			if !ok {
				continue
			}
			attrs := []*HCLAttribute{
				Attr("name", Str(getStringValue(param, "name"))),
				Attr("type", Str(strings.ToLower(getStringValue(param, "fieldType")))),
			}
			if r, ok := param["required"].(bool); ok && r {
				attrs = append(attrs, Attr("required", Bool(true)))
			}
			if m, ok := param["multiple"].(bool); ok && m {
				attrs = append(attrs, Attr("multiple", Bool(true)))
			}
			if defaultVal := param["defaultValue"]; defaultVal != nil {
				switch dv := defaultVal.(type) {
				case string:
					if dv != "" {
						attrs = append(attrs, Attr("default_value", Str(dv)))
					}
				case map[string]interface{}:
					if decoded := decodeValueReference(dv); decoded != "" {
						attrs = append(attrs, Attr("default_value", Str(decoded)))
					}
				}
			}
			paramObjs = append(paramObjs, Obj(attrs...))
		}
		b.SetAttr("parameters", HCLList{Values: paramObjs})

	case field.Name == "trigger_on", field.Name == "status", field.Name == "conversation_type":
		if strVal, ok := value.(string); ok && strVal != "" {
			b.SetAttr(field.Name, Str(strings.ToLower(strVal)))
		}

	case field.Name == "fire_on_alert", field.Name == "fire_on_recovery", field.Name == "authenticated", field.Name == "show_triggered_by":
		if boolVal, ok := value.(bool); ok {
			b.SetAttr(field.Name, Bool(boolVal))
		}

	case field.Name == "value":
		switch v := value.(type) {
		case float64:
			b.SetAttr(field.Name, Num(v))
		case int:
			b.SetAttr(field.Name, Num(float64(v)))
		}

	case field.Name == "schedule":
		if strVal, ok := value.(string); ok && strVal != "" {
			b.SetAttr(field.Name, Str(strVal))
		}

	default:
		switch v := value.(type) {
		case string:
			if v != "" {
				b.SetAttr(field.Name, Str(v))
			}
		case float64:
			b.SetAttr(field.Name, Num(v))
		case bool:
			b.SetAttr(field.Name, Bool(v))
		}
	}
}

// BeautifyTrigger applies beautification to a single trigger resource block in HCL
func (g *TriggerHCLGenerator) BeautifyTrigger(hcl string, trigger *discovery.Trigger, automationResourceName string) string {
	triggerType := trigger.Type
	resourceType := "elementum_" + triggerType + "_trigger"

	// Find the trigger resource name
	triggerResourceName := ""
	for _, imp := range g.imports {
		if imp.ResourceType == resourceType && strings.HasSuffix(imp.ID, ":"+trigger.ID) {
			triggerResourceName = imp.ResourceName
			break
		}
	}
	if triggerResourceName == "" {
		return hcl
	}

	// Replace UUID strings with terraform references
	for uuid, ref := range g.uuidMap {
		// Replace quoted UUIDs
		hcl = strings.ReplaceAll(hcl, fmt.Sprintf("%q", uuid), ref)
	}

	return hcl
}

// GetSupportedTriggerTypes returns all supported trigger types
func GetSupportedTriggerTypes() []string {
	types := make([]string, 0)
	for typeName := range client.TriggerTypeRegistry {
		types = append(types, typeName)
	}
	return types
}
