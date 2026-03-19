// Copyright (c) ElementumAI, Inc.
// SPDX-License-Identifier: MPL-2.0

package export

import (
	"strings"

	"github.com/elementumltd/elementum-cli/discovery"
)

// graphqlToolTypeToTerraformResource maps GraphQL tool type names to Terraform resource types
var graphqlToolTypeToTerraformResource = map[string]string{
	"AgentCreateRecordTool":    "elementum_agent_create_record_tool",
	"AgentSearchAspectTool":    "elementum_agent_search_records_tool",
	"AgentUpdateRecordTool":    "elementum_agent_update_record_tool",
	"AgentSearchTableTool":     "elementum_agent_ai_search_tool",
	"AgentRelateRecordTool":    "elementum_agent_relate_record_tool",
	"AgentExecuteWorkflowTool": "elementum_agent_run_automation_tool",
	"AgentRunAgentTool":        "elementum_agent_run_agent_tool",
	"AgentSelectBotRouteTool":  "elementum_agent_select_bot_route_tool",
	"AgentMcpTool":             "elementum_agent_mcp_tool",
}

// graphqlAgentTypeToTerraform maps GraphQL agent type names to Terraform type attribute values
var graphqlAgentTypeToTerraform = map[string]string{
	"AgentElementum":  "elementum",
	"AgentSnowflake":  "snowflake",
	"AgentBedrock":    "bedrock",
	"AgentBrowserUse": "browser_use",
}

// AgentHCLGenerator generates HCL for agent and agent tool resources
type AgentHCLGenerator struct {
	app     *discovery.App
	imports []ImportBlock
	uuidMap map[string]string
}

// NewAgentHCLGenerator creates a new agent HCL generator
func NewAgentHCLGenerator(app *discovery.App, imports []ImportBlock, uuidMap map[string]string) *AgentHCLGenerator {
	return &AgentHCLGenerator{
		app:     app,
		imports: imports,
		uuidMap: uuidMap,
	}
}

// GenerateAll generates HCL for all agents and their tools
func (g *AgentHCLGenerator) GenerateAll() string {
	return SerializeBlocks(g.GenerateAllIR())
}

// GenerateAllIR generates IR blocks for all agents and their tools
func (g *AgentHCLGenerator) GenerateAllIR() []*HCLBlock {
	var blocks []*HCLBlock

	for _, agent := range g.app.Agents {
		b := g.GenerateAgentIR(&agent)
		if b != nil {
			blocks = append(blocks, b)
		}
		for _, tool := range agent.Tools {
			tb := g.GenerateAgentToolIR(&agent, &tool)
			if tb != nil {
				blocks = append(blocks, tb)
			}
		}
	}

	return blocks
}

// GenerateAgentIR generates an IR block for a single agent
func (g *AgentHCLGenerator) GenerateAgentIR(agent *discovery.Agent) *HCLBlock {
	var resourceName string
	for _, imp := range g.imports {
		if imp.ResourceType == "elementum_agent" && strings.Contains(imp.ID, agent.ID) {
			resourceName = imp.ResourceName
			break
		}
	}
	if resourceName == "" {
		return nil
	}

	appResourceName := AppResourceName(g.app)
	b := NewResourceBlock("elementum_agent", resourceName)
	b.SetAttr("app_id", Ref("elementum_app."+appResourceName+".id"))
	b.SetAttr("name", Str(agent.Name))

	if agent.Description != "" {
		b.SetAttr("description", Str(agent.Description))
	}
	if agent.Instructions != "" {
		b.SetAttr("instructions", Str(agent.Instructions))
	}
	if agent.FirstMessage != "" {
		b.SetAttr("first_message", Str(agent.FirstMessage))
	}

	if tfType, ok := graphqlAgentTypeToTerraform[agent.Type]; ok && tfType != "elementum" {
		b.SetAttr("type", Str(tfType))
	}

	if agent.AiProviderConnectorID != "" {
		b.SetAttr("ai_provider_connector_id", Str(agent.AiProviderConnectorID))
	}

	// Type-specific configs
	switch agent.Type {
	case "AgentSnowflake":
		if agent.SnowflakeDatabase != "" {
			b.SetAttr("snowflake_database", Str(agent.SnowflakeDatabase))
		}
		if agent.SnowflakeSchema != "" {
			b.SetAttr("snowflake_schema", Str(agent.SnowflakeSchema))
		}
		if agent.SnowflakeName != "" {
			b.SetAttr("snowflake_name", Str(agent.SnowflakeName))
		}
	case "AgentBedrock":
		if agent.BedrockAgentArn != "" {
			b.SetAttr("bedrock_agent_arn", Str(agent.BedrockAgentArn))
		}
	case "AgentBrowserUse":
		if agent.BrowserAgentInstructions != "" {
			b.SetAttr("browser_agent_instructions", Str(agent.BrowserAgentInstructions))
		}
	}

	// Starting actions
	if len(agent.StartingActions) > 0 {
		var actionObjs []HCLValue
		for _, action := range agent.StartingActions {
			attrs := []*HCLAttribute{
				Attr("name", Str(action.Name)),
				Attr("prompt", Str(action.Prompt)),
				Attr("order", Num(float64(action.Order))),
			}
			if !action.Enabled {
				attrs = append(attrs, Attr("enabled", Bool(false)))
			}
			if action.Icon != "" {
				attrs = append(attrs, Attr("icon", Str(action.Icon)))
			}

			if len(action.Variables) > 0 {
				var varObjs []HCLValue
				for _, v := range action.Variables {
					vAttrs := []*HCLAttribute{
						Attr("key", Str(v.Key)),
						Attr("label", Str(v.Label)),
						Attr("type", Str(v.Type)),
					}
					if v.Required {
						vAttrs = append(vAttrs, Attr("required", Bool(true)))
					}
					if v.Placeholder != "" {
						vAttrs = append(vAttrs, Attr("placeholder", Str(v.Placeholder)))
					}
					if len(v.Options) > 0 {
						var optObjs []HCLValue
						for _, opt := range v.Options {
							optObjs = append(optObjs, Obj(
								Attr("label", Str(opt.Label)),
								Attr("value", Str(opt.Value)),
							))
						}
						vAttrs = append(vAttrs, Attr("options", HCLList{Values: optObjs}))
					}
					if v.Validation != nil {
						var valAttrs []*HCLAttribute
						if v.Validation.MinLength != nil {
							valAttrs = append(valAttrs, Attr("min_length", Num(float64(*v.Validation.MinLength))))
						}
						if v.Validation.MaxLength != nil {
							valAttrs = append(valAttrs, Attr("max_length", Num(float64(*v.Validation.MaxLength))))
						}
						if v.Validation.Pattern != "" {
							valAttrs = append(valAttrs, Attr("pattern", Str(v.Validation.Pattern)))
						}
						if v.Validation.ErrorMessage != "" {
							valAttrs = append(valAttrs, Attr("error_message", Str(v.Validation.ErrorMessage)))
						}
						if len(valAttrs) > 0 {
							vAttrs = append(vAttrs, Attr("validation", Obj(valAttrs...)))
						}
					}
					varObjs = append(varObjs, Obj(vAttrs...))
				}
				attrs = append(attrs, Attr("variables", HCLList{Values: varObjs}))
			}

			actionObjs = append(actionObjs, Obj(attrs...))
		}
		b.SetAttr("starting_actions", HCLList{Values: actionObjs})
	}

	return b
}

// generateToolFieldsIR converts a RawConfig array (fields, attribute_fields, return_fields, inputs, outputs)
// into an HCLList of HCLObject values. Returns nil if the key is missing or empty.
func generateToolFieldsIR(rawConfig map[string]interface{}, key string) HCLValue {
	fields, ok := rawConfig[key].([]map[string]interface{})
	if !ok || len(fields) == 0 {
		// Try []interface{} conversion
		if ifaces, ok := rawConfig[key].([]interface{}); ok && len(ifaces) > 0 {
			for _, iface := range ifaces {
				if m, ok := iface.(map[string]interface{}); ok {
					fields = append(fields, m)
				}
			}
		}
		if len(fields) == 0 {
			return nil
		}
	}

	var fieldObjs []HCLValue
	for _, f := range fields {
		var attrs []*HCLAttribute
		if fieldID, ok := f["field_id"].(string); ok && fieldID != "" {
			attrs = append(attrs, Attr("field_id", Str(fieldID)))
		}
		// "name" is used in fields and inputs; "field_name" is used in attribute_fields
		if name, ok := f["name"].(string); ok && name != "" {
			attrs = append(attrs, Attr("name", Str(name)))
		} else if fieldName, ok := f["field_name"].(string); ok && fieldName != "" {
			attrs = append(attrs, Attr("name", Str(fieldName)))
		}
		if desc, ok := f["description"].(string); ok && desc != "" {
			attrs = append(attrs, Attr("description", Str(desc)))
		}
		if required, ok := f["required"].(bool); ok && required {
			attrs = append(attrs, Attr("required", Bool(true)))
		}
		if dt, ok := f["data_type"].(string); ok && dt != "" {
			attrs = append(attrs, Attr("data_type", Str(dt)))
		}
		if ro, ok := f["read_only"].(bool); ok && ro {
			attrs = append(attrs, Attr("read_only", Bool(true)))
		}
		// display_name is used in return_fields
		if dn, ok := f["display_name"].(string); ok && dn != "" {
			attrs = append(attrs, Attr("display_name", Str(dn)))
		}
		// For inputs/outputs: type, default_value, multiple
		if t, ok := f["type"].(string); ok && t != "" {
			attrs = append(attrs, Attr("type", Str(t)))
		}
		if dv, ok := f["default_value"].(string); ok && dv != "" {
			attrs = append(attrs, Attr("default_value", Str(dv)))
		}
		if mult, ok := f["multiple"].(bool); ok && mult {
			attrs = append(attrs, Attr("multiple", Bool(true)))
		}
		// trigger_parameter_id for inputs
		if tpID, ok := f["trigger_parameter_id"].(string); ok && tpID != "" {
			attrs = append(attrs, Attr("trigger_parameter_id", Str(tpID)))
		}
		// output_name for outputs
		if outName, ok := f["output_name"].(string); ok && outName != "" {
			attrs = append(attrs, Attr("output_name", Str(outName)))
		}
		if len(attrs) > 0 {
			fieldObjs = append(fieldObjs, Obj(attrs...))
		}
	}
	if len(fieldObjs) == 0 {
		return nil
	}
	return HCLList{Values: fieldObjs}
}

// generateHeadersIR converts a RawConfig headers array into an HCLList of objects with key/value attributes.
// Handles both []map[string]interface{} and []interface{} forms.
func generateHeadersIR(rawConfig map[string]interface{}) HCLValue {
	var headers []map[string]interface{}
	if h, ok := rawConfig["headers"].([]map[string]interface{}); ok {
		headers = h
	} else if ifaces, ok := rawConfig["headers"].([]interface{}); ok {
		for _, iface := range ifaces {
			if m, ok := iface.(map[string]interface{}); ok {
				headers = append(headers, m)
			}
		}
	}
	if len(headers) == 0 {
		return nil
	}

	var objs []HCLValue
	for _, header := range headers {
		var attrs []*HCLAttribute
		if key, ok := header["key"].(string); ok && key != "" {
			attrs = append(attrs, Attr("key", Str(key)))
		}
		if value, ok := header["value"].(string); ok && value != "" {
			attrs = append(attrs, Attr("value", Str(value)))
		}
		if len(attrs) > 0 {
			objs = append(objs, Obj(attrs...))
		}
	}
	if len(objs) == 0 {
		return nil
	}
	return HCLList{Values: objs}
}

// GenerateAgentToolIR generates an IR block for a single agent tool
func (g *AgentHCLGenerator) GenerateAgentToolIR(agent *discovery.Agent, tool *discovery.AgentTool) *HCLBlock {
	tfResourceType, ok := graphqlToolTypeToTerraformResource[tool.Type]
	if !ok {
		return nil
	}

	var resourceName string
	for _, imp := range g.imports {
		if imp.ResourceType == tfResourceType && strings.Contains(imp.ID, tool.ID) {
			resourceName = imp.ResourceName
			break
		}
	}
	if resourceName == "" {
		return nil
	}

	appResourceName := AppResourceName(g.app)

	var agentResourceName string
	for _, imp := range g.imports {
		if imp.ResourceType == "elementum_agent" && strings.Contains(imp.ID, agent.ID) {
			agentResourceName = imp.ResourceName
			break
		}
	}
	if agentResourceName == "" {
		agentResourceName = SanitizeName(agent.Name)
	}

	b := NewResourceBlock(tfResourceType, resourceName)
	b.SetAttr("agent_id", Ref("elementum_agent."+agentResourceName+".id"))
	b.SetAttr("app_id", Ref("elementum_app."+appResourceName+".id"))
	b.SetAttr("name", Str(tool.Name))

	if tool.Description != "" {
		b.SetAttr("description", Str(tool.Description))
	}
	if tool.StartMessage != "" {
		b.SetAttr("start_message", Str(tool.StartMessage))
	}

	// Tool-specific fields
	switch tool.Type {
	case "AgentCreateRecordTool", "AgentSearchAspectTool", "AgentUpdateRecordTool":
		if tool.AspectID != "" {
			b.SetAttr("target_id", Str(tool.AspectID))
		}
	case "AgentSearchTableTool":
		if tool.AspectID != "" {
			b.SetAttr("target_id", Str(tool.AspectID))
		}
		if tool.SearchTableID != "" {
			b.SetAttr("search_table_id", Str(tool.SearchTableID))
		}
	case "AgentRelateRecordTool":
		if tool.AspectID != "" {
			b.SetAttr("target_id", Str(tool.AspectID))
		}
		if tool.RelatedAspectID != "" {
			b.SetAttr("related_target_id", Str(tool.RelatedAspectID))
		}
	case "AgentExecuteWorkflowTool":
		if tool.AutomationID != "" {
			b.SetAttr("automation_id", Str(tool.AutomationID))
		}
	case "AgentRunAgentTool":
		if tool.TargetAgentID != "" {
			b.SetAttr("target_agent_id", Str(tool.TargetAgentID))
		}
	case "AgentSelectBotRouteTool":
		if tool.RawConfig != nil {
			if botID, ok := tool.RawConfig["bot_id"].(string); ok && botID != "" {
				b.SetAttr("bot_id", Str(botID))
			}
		}
	case "AgentMcpTool":
		if tool.RawConfig != nil {
			if mcpName, ok := tool.RawConfig["mcp_tool_name"].(string); ok && mcpName != "" {
				b.SetAttr("mcp_tool_name", Str(mcpName))
			}
			if serverUrl, ok := tool.RawConfig["server_url"].(string); ok && serverUrl != "" {
				b.SetAttr("server_url", Str(serverUrl))
			}
		}
	}

	// RawConfig-based fields (fields, attribute_fields, return_fields, inputs, outputs, headers)
	if tool.RawConfig != nil {
		switch tool.Type {
		case "AgentCreateRecordTool", "AgentSearchAspectTool", "AgentUpdateRecordTool":
			if qd, ok := tool.RawConfig["query_description"].(string); ok && qd != "" {
				b.SetAttr("query_description", Str(qd))
			}
			if hd, ok := tool.RawConfig["handle_description"].(string); ok && hd != "" {
				b.SetAttr("handle_description", Str(hd))
			}
			if limit, ok := tool.RawConfig["result_limit"].(int); ok && limit > 0 {
				b.SetAttr("result_limit", Num(float64(limit)))
			} else if limit64, ok := tool.RawConfig["result_limit"].(int64); ok && limit64 > 0 {
				b.SetAttr("result_limit", Num(float64(limit64)))
			}
			if v := generateToolFieldsIR(tool.RawConfig, "fields"); v != nil {
				b.SetAttr("fields", v)
			}
			if v := generateToolFieldsIR(tool.RawConfig, "attribute_fields"); v != nil {
				b.SetAttr("attribute_fields", v)
			}
			if v := generateToolFieldsIR(tool.RawConfig, "return_fields"); v != nil {
				b.SetAttr("return_fields", v)
			}

		case "AgentSearchTableTool":
			if qd, ok := tool.RawConfig["query_description"].(string); ok && qd != "" {
				b.SetAttr("query_description", Str(qd))
			}
			if limit, ok := tool.RawConfig["result_limit"].(int); ok && limit > 0 {
				b.SetAttr("result_limit", Num(float64(limit)))
			} else if limit64, ok := tool.RawConfig["result_limit"].(int64); ok && limit64 > 0 {
				b.SetAttr("result_limit", Num(float64(limit64)))
			}
			if v := generateToolFieldsIR(tool.RawConfig, "attribute_fields"); v != nil {
				b.SetAttr("attribute_fields", v)
			}
			if v := generateToolFieldsIR(tool.RawConfig, "return_fields"); v != nil {
				b.SetAttr("return_fields", v)
			}

		case "AgentExecuteWorkflowTool":
			if v := generateToolFieldsIR(tool.RawConfig, "inputs"); v != nil {
				b.SetAttr("inputs", v)
			}
			if v := generateToolFieldsIR(tool.RawConfig, "outputs"); v != nil {
				b.SetAttr("outputs", v)
			}

		case "AgentRunAgentTool":
			if prompt, ok := tool.RawConfig["worker_task_prompt"].(string); ok && prompt != "" {
				b.SetAttr("worker_task_prompt", Str(prompt))
			}

		case "AgentMcpTool":
			if v := generateHeadersIR(tool.RawConfig); v != nil {
				b.SetAttr("headers", v)
			}
		}
	}

	return b
}
