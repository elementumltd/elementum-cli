// Copyright (c) ElementumAI, Inc.
// SPDX-License-Identifier: MPL-2.0

package export

import (
	"strings"
	"testing"

	"github.com/elementumltd/elementum-cli/discovery"
)

// Additional test UUIDs for agent tests
const (
	TestAgentToolID         = "eeeeeeee-eeee-eeee-eeee-eeeeeeeeeeee"
	TestAgentToolID2        = "eeeeeeee-eeee-eeee-eeee-eeeeeeeeeee2"
	TestTargetAgentID       = "ffffffff-ffff-ffff-ffff-ffffffffffff"
	TestRelatedAspectID     = "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaab"
	TestTableID             = "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbb2"
	TestAIProviderConnector = "dddddddd-dddd-dddd-dddd-ddddddddddd2"
)

// ============================================================================
// Core AgentHCLGenerator Tests
// ============================================================================

func TestAgentHCLGenerator_GenerateAll(t *testing.T) {
	app := &discovery.App{
		ID:   TestAppID,
		Name: "Test App",
		Agents: []discovery.Agent{
			{
				ID:           TestAgentID,
				Name:         "Support Agent",
				Description:  "A helpful support agent",
				Type:         "AgentElementum",
				Instructions: "Help users with their questions",
				FirstMessage: "Hello! How can I help you?",
				Tools: []discovery.AgentTool{
					{
						ID:          TestAgentToolID,
						Name:        "Search Tickets",
						Description: "Search for tickets",
						Type:        "AgentSearchAspectTool",
						AspectID:    TestElementID,
						AspectName:  "Tickets",
					},
				},
			},
		},
	}

	imports := []ImportBlock{
		{ID: TestAppID, ResourceType: "elementum_app", ResourceName: "test_app"},
		{ID: TestAppID + ":" + TestAgentID, ResourceType: "elementum_agent", ResourceName: "support_agent"},
		{ID: TestAppID + ":" + TestAgentID + ":" + TestAgentToolID, ResourceType: "elementum_agent_search_records_tool", ResourceName: "search_tickets"},
	}
	uuidMap := make(map[string]string)

	generator := NewAgentHCLGenerator(app, imports, uuidMap)
	hcl := generator.GenerateAll()

	assertHCLContains(t, hcl,
		`resource "elementum_agent" "support_agent"`,
		`resource "elementum_agent_search_records_tool" "search_tickets"`,
		`app_id = elementum_app.test_app.id`,
		`name = "Support Agent"`,
		`name = "Search Tickets"`,
	)
}

func TestAgentHCLGenerator_NoAgents(t *testing.T) {
	app := &discovery.App{
		ID:     TestAppID,
		Name:   "Test App",
		Agents: []discovery.Agent{},
	}

	imports := []ImportBlock{
		{ID: TestAppID, ResourceType: "elementum_app", ResourceName: "test_app"},
	}
	uuidMap := make(map[string]string)

	generator := NewAgentHCLGenerator(app, imports, uuidMap)
	hcl := generator.GenerateAll()

	if hcl != "" {
		t.Errorf("Expected empty HCL for app with no agents, got:\n%s", hcl)
	}
}

func TestAgentHCLGenerator_NoImports(t *testing.T) {
	app := &discovery.App{
		ID:   TestAppID,
		Name: "Test App",
		Agents: []discovery.Agent{
			{
				ID:           TestAgentID,
				Name:         "Support Agent",
				Type:         "AgentElementum",
				Instructions: "Help users",
			},
		},
	}

	// Empty imports - no resource names to match
	imports := []ImportBlock{}
	uuidMap := make(map[string]string)

	generator := NewAgentHCLGenerator(app, imports, uuidMap)
	hcl := generator.GenerateAll()

	// Should produce empty HCL since there's no matching import for agent
	if strings.Contains(hcl, "resource") {
		t.Errorf("Expected empty HCL when no imports match, got:\n%s", hcl)
	}
}

// ============================================================================
// Agent Type Tests
// ============================================================================

func TestAgentHCL_ElementumAgent(t *testing.T) {
	app := &discovery.App{
		ID:   TestAppID,
		Name: "Test App",
		Agents: []discovery.Agent{
			{
				ID:                    TestAgentID,
				Name:                  "Elementum Agent",
				Description:           "A standard Elementum agent",
				Type:                  "AgentElementum",
				Instructions:          "Be helpful and concise",
				FirstMessage:          "Hi! I'm here to help.",
				AiProviderConnectorID: TestAIProviderConnector,
			},
		},
	}

	imports := []ImportBlock{
		{ID: TestAppID, ResourceType: "elementum_app", ResourceName: "test_app"},
		{ID: TestAppID + ":" + TestAgentID, ResourceType: "elementum_agent", ResourceName: "elementum_agent"},
	}
	uuidMap := map[string]string{
		TestAIProviderConnector: "data.elementum_ai_provider_connector.claude.id",
	}

	generator := NewAgentHCLGenerator(app, imports, uuidMap)
	hcl := generator.GenerateAll()

	assertHCLContains(t, hcl,
		`resource "elementum_agent" "elementum_agent"`,
		`name = "Elementum Agent"`,
		`description = "A standard Elementum agent"`,
		`instructions = "Be helpful and concise"`,
		`first_message = "Hi! I'm here to help."`,
		`ai_provider_connector_id = "`+TestAIProviderConnector+`"`,
	)

	// Should NOT contain type = "elementum" since it's the default
	if strings.Contains(hcl, `type = "elementum"`) {
		t.Errorf("Expected HCL to NOT contain type = elementum (it's the default), got:\n%s", hcl)
	}
}

func TestAgentHCL_SnowflakeAgent(t *testing.T) {
	app := &discovery.App{
		ID:   TestAppID,
		Name: "Test App",
		Agents: []discovery.Agent{
			{
				ID:                    TestAgentID,
				Name:                  "Data Analyst",
				Description:           "Analyzes data in Snowflake",
				Type:                  "AgentSnowflake",
				Instructions:          "Query the database to answer questions",
				AiProviderConnectorID: TestAIProviderConnector,
				SnowflakeDatabase:     "ANALYTICS_DB",
				SnowflakeSchema:       "PUBLIC",
				SnowflakeName:         "CORTEX_ANALYST",
			},
		},
	}

	imports := []ImportBlock{
		{ID: TestAppID, ResourceType: "elementum_app", ResourceName: "test_app"},
		{ID: TestAppID + ":" + TestAgentID, ResourceType: "elementum_agent", ResourceName: "data_analyst"},
	}
	uuidMap := make(map[string]string)

	generator := NewAgentHCLGenerator(app, imports, uuidMap)
	hcl := generator.GenerateAll()

	assertHCLContains(t, hcl,
		`resource "elementum_agent" "data_analyst"`,
		`name = "Data Analyst"`,
		`type = "snowflake"`,
		`snowflake_database = "ANALYTICS_DB"`,
		`snowflake_schema = "PUBLIC"`,
		`snowflake_name = "CORTEX_ANALYST"`,
	)
}

func TestAgentHCL_BedrockAgent(t *testing.T) {
	app := &discovery.App{
		ID:   TestAppID,
		Name: "Test App",
		Agents: []discovery.Agent{
			{
				ID:                    TestAgentID,
				Name:                  "AWS Agent",
				Description:           "AWS Bedrock agent",
				Type:                  "AgentBedrock",
				Instructions:          "Assist with AWS tasks",
				AiProviderConnectorID: TestAIProviderConnector,
				BedrockAgentArn:       "arn:aws:bedrock:us-east-1:123456789:agent/ABC123",
			},
		},
	}

	imports := []ImportBlock{
		{ID: TestAppID, ResourceType: "elementum_app", ResourceName: "test_app"},
		{ID: TestAppID + ":" + TestAgentID, ResourceType: "elementum_agent", ResourceName: "aws_agent"},
	}
	uuidMap := make(map[string]string)

	generator := NewAgentHCLGenerator(app, imports, uuidMap)
	hcl := generator.GenerateAll()

	assertHCLContains(t, hcl,
		`resource "elementum_agent" "aws_agent"`,
		`name = "AWS Agent"`,
		`type = "bedrock"`,
		`bedrock_agent_arn = "arn:aws:bedrock:us-east-1:123456789:agent/ABC123"`,
	)
}

func TestAgentHCL_BrowserUseAgent(t *testing.T) {
	app := &discovery.App{
		ID:   TestAppID,
		Name: "Test App",
		Agents: []discovery.Agent{
			{
				ID:                       TestAgentID,
				Name:                     "Browser Agent",
				Description:              "Browses the web",
				Type:                     "AgentBrowserUse",
				Instructions:             "Navigate websites to find information",
				AiProviderConnectorID:    TestAIProviderConnector,
				BrowserAgentInstructions: "Use Chrome browser to navigate",
			},
		},
	}

	imports := []ImportBlock{
		{ID: TestAppID, ResourceType: "elementum_app", ResourceName: "test_app"},
		{ID: TestAppID + ":" + TestAgentID, ResourceType: "elementum_agent", ResourceName: "browser_agent"},
	}
	uuidMap := make(map[string]string)

	generator := NewAgentHCLGenerator(app, imports, uuidMap)
	hcl := generator.GenerateAll()

	assertHCLContains(t, hcl,
		`resource "elementum_agent" "browser_agent"`,
		`name = "Browser Agent"`,
		`type = "browser_use"`,
		`browser_agent_instructions = "Use Chrome browser to navigate"`,
	)
}

// ============================================================================
// Agent Tool Type Tests
// ============================================================================

func TestAgentToolHCL_CreateRecordTool(t *testing.T) {
	app := &discovery.App{
		ID:   TestAppID,
		Name: "Test App",
		Agents: []discovery.Agent{
			{
				ID:           TestAgentID,
				Name:         "Support Agent",
				Type:         "AgentElementum",
				Instructions: "Help users",
				Tools: []discovery.AgentTool{
					{
						ID:           TestAgentToolID,
						Name:         "Create Ticket",
						Description:  "Creates a new support ticket",
						Type:         "AgentCreateRecordTool",
						StartMessage: "Creating a new ticket...",
						AspectID:     TestElementID,
						AspectName:   "Tickets",
						RawConfig: map[string]interface{}{
							"fields": []map[string]interface{}{
								{
									"field_id":    TestFieldID,
									"name":        "title",
									"description": "Ticket title",
									"required":    true,
								},
								{
									"field_id":    TestField2ID,
									"name":        "description",
									"description": "Ticket description",
									"required":    false,
								},
							},
						},
					},
				},
			},
		},
	}

	imports := []ImportBlock{
		{ID: TestAppID, ResourceType: "elementum_app", ResourceName: "test_app"},
		{ID: TestAppID + ":" + TestAgentID, ResourceType: "elementum_agent", ResourceName: "support_agent"},
		{ID: TestAppID + ":" + TestAgentID + ":" + TestAgentToolID, ResourceType: "elementum_agent_create_record_tool", ResourceName: "create_ticket"},
	}
	uuidMap := make(map[string]string)

	generator := NewAgentHCLGenerator(app, imports, uuidMap)
	hcl := generator.GenerateAll()

	assertHCLContains(t, hcl,
		`resource "elementum_agent_create_record_tool" "create_ticket"`,
		`agent_id = elementum_agent.support_agent.id`,
		`app_id = elementum_app.test_app.id`,
		`name = "Create Ticket"`,
		`description = "Creates a new support ticket"`,
		`start_message = "Creating a new ticket..."`,
		`target_id = "`,
		"fields = [",
		`field_id = "`,
		`name = "title"`,
		`required = true`,
	)
}

func TestAgentToolHCL_SearchRecordsTool(t *testing.T) {
	app := &discovery.App{
		ID:   TestAppID,
		Name: "Test App",
		Agents: []discovery.Agent{
			{
				ID:           TestAgentID,
				Name:         "Search Agent",
				Type:         "AgentElementum",
				Instructions: "Search for records",
				Tools: []discovery.AgentTool{
					{
						ID:          TestAgentToolID,
						Name:        "Search Tickets",
						Description: "Search for tickets",
						Type:        "AgentSearchAspectTool",
						AspectID:    TestElementID,
						AspectName:  "Tickets",
						RawConfig: map[string]interface{}{
							"query_description": "Search tickets by title or status",
							"result_limit":      10,
						},
					},
				},
			},
		},
	}

	imports := []ImportBlock{
		{ID: TestAppID, ResourceType: "elementum_app", ResourceName: "test_app"},
		{ID: TestAppID + ":" + TestAgentID, ResourceType: "elementum_agent", ResourceName: "search_agent"},
		{ID: TestAppID + ":" + TestAgentID + ":" + TestAgentToolID, ResourceType: "elementum_agent_search_records_tool", ResourceName: "search_tickets"},
	}
	uuidMap := make(map[string]string)

	generator := NewAgentHCLGenerator(app, imports, uuidMap)
	hcl := generator.GenerateAll()

	assertHCLContains(t, hcl,
		`resource "elementum_agent_search_records_tool" "search_tickets"`,
		`target_id = "`,
		`query_description = "Search tickets by title or status"`,
		`result_limit = 10`,
	)
}

func TestAgentToolHCL_SearchRecordsTool_Int64Limit(t *testing.T) {
	// Test that result_limit works with int64 values (as returned by JSON unmarshaling)
	app := &discovery.App{
		ID:   TestAppID,
		Name: "Test App",
		Agents: []discovery.Agent{
			{
				ID:           TestAgentID,
				Name:         "Search Agent",
				Type:         "AgentElementum",
				Instructions: "Search for records",
				Tools: []discovery.AgentTool{
					{
						ID:          TestAgentToolID,
						Name:        "Search Tickets",
						Description: "Search for tickets",
						Type:        "AgentSearchAspectTool",
						AspectID:    TestElementID,
						AspectName:  "Tickets",
						RawConfig: map[string]interface{}{
							"query_description": "Search tickets",
							"result_limit":      int64(25), // int64 as returned by JSON
						},
					},
				},
			},
		},
	}

	imports := []ImportBlock{
		{ID: TestAppID, ResourceType: "elementum_app", ResourceName: "test_app"},
		{ID: TestAppID + ":" + TestAgentID, ResourceType: "elementum_agent", ResourceName: "search_agent"},
		{ID: TestAppID + ":" + TestAgentID + ":" + TestAgentToolID, ResourceType: "elementum_agent_search_records_tool", ResourceName: "search_tickets"},
	}
	uuidMap := make(map[string]string)

	generator := NewAgentHCLGenerator(app, imports, uuidMap)
	hcl := generator.GenerateAll()

	assertHCLContains(t, hcl,
		`resource "elementum_agent_search_records_tool" "search_tickets"`,
		`result_limit = 25`,
	)
}

func TestAgentToolHCL_UpdateRecordTool(t *testing.T) {
	app := &discovery.App{
		ID:   TestAppID,
		Name: "Test App",
		Agents: []discovery.Agent{
			{
				ID:           TestAgentID,
				Name:         "Update Agent",
				Type:         "AgentElementum",
				Instructions: "Update records",
				Tools: []discovery.AgentTool{
					{
						ID:          TestAgentToolID,
						Name:        "Update Ticket",
						Description: "Update a ticket",
						Type:        "AgentUpdateRecordTool",
						AspectID:    TestElementID,
						AspectName:  "Tickets",
						RawConfig: map[string]interface{}{
							"handle_description": "The ticket ID to update",
							"fields": []map[string]interface{}{
								{
									"field_id":    TestFieldID,
									"name":        "status",
									"description": "New status",
									"required":    true,
								},
							},
						},
					},
				},
			},
		},
	}

	imports := []ImportBlock{
		{ID: TestAppID, ResourceType: "elementum_app", ResourceName: "test_app"},
		{ID: TestAppID + ":" + TestAgentID, ResourceType: "elementum_agent", ResourceName: "update_agent"},
		{ID: TestAppID + ":" + TestAgentID + ":" + TestAgentToolID, ResourceType: "elementum_agent_update_record_tool", ResourceName: "update_ticket"},
	}
	uuidMap := make(map[string]string)

	generator := NewAgentHCLGenerator(app, imports, uuidMap)
	hcl := generator.GenerateAll()

	assertHCLContains(t, hcl,
		`resource "elementum_agent_update_record_tool" "update_ticket"`,
		`target_id = "`,
		`handle_description = "The ticket ID to update"`,
		"fields = [",
	)
}

func TestAgentToolHCL_AISearchTool(t *testing.T) {
	app := &discovery.App{
		ID:   TestAppID,
		Name: "Test App",
		Agents: []discovery.Agent{
			{
				ID:           TestAgentID,
				Name:         "Search Agent",
				Type:         "AgentElementum",
				Instructions: "Search records",
				Tools: []discovery.AgentTool{
					{
						ID:            TestAgentToolID,
						Name:          "Knowledge Search",
						Description:   "Search knowledge base",
						Type:          "AgentSearchTableTool",
						AspectID:      TestElementID,
						SearchTableID: TestTableID,
						RawConfig: map[string]interface{}{
							"query_description": "Search knowledge articles",
							"result_limit":      5,
							"attribute_fields": []map[string]interface{}{
								{
									"field_id":   TestFieldID,
									"field_name": "Title",
								},
								{
									"field_id":   TestField2ID,
									"field_name": "Description",
								},
							},
							"return_fields": []map[string]interface{}{
								{
									"field_id":     TestFieldID,
									"display_name": "Article Title",
								},
							},
						},
					},
				},
			},
		},
	}

	imports := []ImportBlock{
		{ID: TestAppID, ResourceType: "elementum_app", ResourceName: "test_app"},
		{ID: TestAppID + ":" + TestAgentID, ResourceType: "elementum_agent", ResourceName: "search_agent"},
		{ID: TestAppID + ":" + TestAgentID + ":" + TestAgentToolID, ResourceType: "elementum_agent_ai_search_tool", ResourceName: "knowledge_search"},
	}
	uuidMap := make(map[string]string)

	generator := NewAgentHCLGenerator(app, imports, uuidMap)
	hcl := generator.GenerateAll()

	assertHCLContains(t, hcl,
		`resource "elementum_agent_ai_search_tool" "knowledge_search"`,
		`target_id = "`,
		`search_table_id = "`,
		`query_description = "Search knowledge articles"`,
		`result_limit = 5`,
		"attribute_fields = [",
		`field_id = "`,
		`name = "Title"`,
		"return_fields = [",
		`display_name = "Article Title"`,
	)
}

func TestAgentToolHCL_RelateRecordTool(t *testing.T) {
	app := &discovery.App{
		ID:   TestAppID,
		Name: "Test App",
		Agents: []discovery.Agent{
			{
				ID:           TestAgentID,
				Name:         "Relate Agent",
				Type:         "AgentElementum",
				Instructions: "Relate records",
				Tools: []discovery.AgentTool{
					{
						ID:                TestAgentToolID,
						Name:              "Link Ticket to Customer",
						Description:       "Links a ticket to a customer",
						Type:              "AgentRelateRecordTool",
						AspectID:          TestElementID,
						AspectName:        "Tickets",
						RelatedAspectID:   TestRelatedAspectID,
						RelatedAspectName: "Customers",
					},
				},
			},
		},
	}

	imports := []ImportBlock{
		{ID: TestAppID, ResourceType: "elementum_app", ResourceName: "test_app"},
		{ID: TestAppID + ":" + TestAgentID, ResourceType: "elementum_agent", ResourceName: "relate_agent"},
		{ID: TestAppID + ":" + TestAgentID + ":" + TestAgentToolID, ResourceType: "elementum_agent_relate_record_tool", ResourceName: "link_ticket_to_customer"},
	}
	uuidMap := make(map[string]string)

	generator := NewAgentHCLGenerator(app, imports, uuidMap)
	hcl := generator.GenerateAll()

	assertHCLContains(t, hcl,
		`resource "elementum_agent_relate_record_tool" "link_ticket_to_customer"`,
		`target_id = "`,
		`related_target_id = "`,
	)
}

func TestAgentToolHCL_RunAutomationTool(t *testing.T) {
	app := &discovery.App{
		ID:   TestAppID,
		Name: "Test App",
		Agents: []discovery.Agent{
			{
				ID:           TestAgentID,
				Name:         "Automation Agent",
				Type:         "AgentElementum",
				Instructions: "Run automations",
				Tools: []discovery.AgentTool{
					{
						ID:             TestAgentToolID,
						Name:           "Escalate Ticket",
						Description:    "Runs the escalation workflow",
						Type:           "AgentExecuteWorkflowTool",
						AutomationID:   TestAutomationID,
						AutomationName: "Escalation Workflow",
						RawConfig: map[string]interface{}{
							"inputs": []map[string]interface{}{
								{
									"name":                 "ticket_id",
									"description":          "The ticket to escalate",
									"required":             true,
									"trigger_parameter_id": "param-1",
								},
							},
							"outputs": []map[string]interface{}{
								{
									"name":        "escalation_status",
									"output_name": "result",
								},
							},
						},
					},
				},
			},
		},
	}

	imports := []ImportBlock{
		{ID: TestAppID, ResourceType: "elementum_app", ResourceName: "test_app"},
		{ID: TestAppID + ":" + TestAgentID, ResourceType: "elementum_agent", ResourceName: "automation_agent"},
		{ID: TestAppID + ":" + TestAgentID + ":" + TestAgentToolID, ResourceType: "elementum_agent_run_automation_tool", ResourceName: "escalate_ticket"},
	}
	uuidMap := make(map[string]string)

	generator := NewAgentHCLGenerator(app, imports, uuidMap)
	hcl := generator.GenerateAll()

	assertHCLContains(t, hcl,
		`resource "elementum_agent_run_automation_tool" "escalate_ticket"`,
		`automation_id = "`,
		"inputs = [",
		`name = "ticket_id"`,
		`description = "The ticket to escalate"`,
		`required = true`,
		`trigger_parameter_id = "param-1"`,
		"outputs = [",
		`name = "escalation_status"`,
		`output_name = "result"`,
	)
}

func TestAgentToolHCL_RunAgentTool(t *testing.T) {
	app := &discovery.App{
		ID:   TestAppID,
		Name: "Test App",
		Agents: []discovery.Agent{
			{
				ID:           TestAgentID,
				Name:         "Orchestrator Agent",
				Type:         "AgentElementum",
				Instructions: "Orchestrate other agents",
				Tools: []discovery.AgentTool{
					{
						ID:              TestAgentToolID,
						Name:            "Run Support Agent",
						Description:     "Delegates to the support agent",
						Type:            "AgentRunAgentTool",
						TargetAgentID:   TestTargetAgentID,
						TargetAgentName: "Support Agent",
						RawConfig: map[string]interface{}{
							"worker_task_prompt": "Handle this support request",
						},
					},
				},
			},
		},
	}

	imports := []ImportBlock{
		{ID: TestAppID, ResourceType: "elementum_app", ResourceName: "test_app"},
		{ID: TestAppID + ":" + TestAgentID, ResourceType: "elementum_agent", ResourceName: "orchestrator_agent"},
		{ID: TestAppID + ":" + TestAgentID + ":" + TestAgentToolID, ResourceType: "elementum_agent_run_agent_tool", ResourceName: "run_support_agent"},
	}
	uuidMap := make(map[string]string)

	generator := NewAgentHCLGenerator(app, imports, uuidMap)
	hcl := generator.GenerateAll()

	assertHCLContains(t, hcl,
		`resource "elementum_agent_run_agent_tool" "run_support_agent"`,
		`target_agent_id = "`,
		`worker_task_prompt = "Handle this support request"`,
	)
}

func TestAgentToolHCL_SelectBotRouteTool(t *testing.T) {
	testBotID := "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"
	app := &discovery.App{
		ID:   TestAppID,
		Name: "Test App",
		Agents: []discovery.Agent{
			{
				ID:           TestAgentID,
				Name:         "Router Agent",
				Type:         "AgentElementum",
				Instructions: "Route conversations to appropriate bots",
				Tools: []discovery.AgentTool{
					{
						ID:          TestAgentToolID,
						Name:        "Select Support Route",
						Description: "Routes to the support bot",
						Type:        "AgentSelectBotRouteTool",
						RawConfig: map[string]interface{}{
							"bot_id": testBotID,
						},
					},
				},
			},
		},
	}

	imports := []ImportBlock{
		{ID: TestAppID, ResourceType: "elementum_app", ResourceName: "test_app"},
		{ID: TestAppID + ":" + TestAgentID, ResourceType: "elementum_agent", ResourceName: "router_agent"},
		{ID: TestAppID + ":" + TestAgentID + ":" + TestAgentToolID, ResourceType: "elementum_agent_select_bot_route_tool", ResourceName: "select_support_route"},
	}
	uuidMap := make(map[string]string)

	generator := NewAgentHCLGenerator(app, imports, uuidMap)
	hcl := generator.GenerateAll()

	assertHCLContains(t, hcl,
		`resource "elementum_agent_select_bot_route_tool" "select_support_route"`,
		`bot_id = "`,
	)
}

func TestAgentToolHCL_McpTool(t *testing.T) {
	app := &discovery.App{
		ID:   TestAppID,
		Name: "Test App",
		Agents: []discovery.Agent{
			{
				ID:           TestAgentID,
				Name:         "MCP Agent",
				Type:         "AgentElementum",
				Instructions: "Use external MCP tools",
				Tools: []discovery.AgentTool{
					{
						ID:          TestAgentToolID,
						Name:        "External Search",
						Description: "Search using external MCP server",
						Type:        "AgentMcpTool",
						RawConfig: map[string]interface{}{
							"mcp_tool_name": "search_documents",
							"server_url":    "https://mcp.example.com/sse",
							"headers": []map[string]interface{}{
								{
									"key":   "Authorization",
									"value": "Bearer token123",
								},
							},
						},
					},
				},
			},
		},
	}

	imports := []ImportBlock{
		{ID: TestAppID, ResourceType: "elementum_app", ResourceName: "test_app"},
		{ID: TestAppID + ":" + TestAgentID, ResourceType: "elementum_agent", ResourceName: "mcp_agent"},
		{ID: TestAppID + ":" + TestAgentID + ":" + TestAgentToolID, ResourceType: "elementum_agent_mcp_tool", ResourceName: "external_search"},
	}
	uuidMap := make(map[string]string)

	generator := NewAgentHCLGenerator(app, imports, uuidMap)
	hcl := generator.GenerateAll()

	assertHCLContains(t, hcl,
		`resource "elementum_agent_mcp_tool" "external_search"`,
		`mcp_tool_name = "search_documents"`,
		`server_url = "https://mcp.example.com/sse"`,
		"headers = [",
		`key = "Authorization"`,
		`value = "Bearer token123"`,
	)
}

// ============================================================================
// Multiple Tools Test
// ============================================================================

func TestAgentHCL_MultipleTools(t *testing.T) {
	app := &discovery.App{
		ID:   TestAppID,
		Name: "Test App",
		Agents: []discovery.Agent{
			{
				ID:           TestAgentID,
				Name:         "Multi-Tool Agent",
				Type:         "AgentElementum",
				Instructions: "Use multiple tools",
				Tools: []discovery.AgentTool{
					{
						ID:       TestAgentToolID,
						Name:     "Search Tickets",
						Type:     "AgentSearchAspectTool",
						AspectID: TestElementID,
					},
					{
						ID:       TestAgentToolID2,
						Name:     "Create Ticket",
						Type:     "AgentCreateRecordTool",
						AspectID: TestElementID,
					},
				},
			},
		},
	}

	imports := []ImportBlock{
		{ID: TestAppID, ResourceType: "elementum_app", ResourceName: "test_app"},
		{ID: TestAppID + ":" + TestAgentID, ResourceType: "elementum_agent", ResourceName: "multi_tool_agent"},
		{ID: TestAppID + ":" + TestAgentID + ":" + TestAgentToolID, ResourceType: "elementum_agent_search_records_tool", ResourceName: "search_tickets"},
		{ID: TestAppID + ":" + TestAgentID + ":" + TestAgentToolID2, ResourceType: "elementum_agent_create_record_tool", ResourceName: "create_ticket"},
	}
	uuidMap := make(map[string]string)

	generator := NewAgentHCLGenerator(app, imports, uuidMap)
	hcl := generator.GenerateAll()

	assertHCLContains(t, hcl,
		`resource "elementum_agent" "multi_tool_agent"`,
		`resource "elementum_agent_search_records_tool" "search_tickets"`,
		`resource "elementum_agent_create_record_tool" "create_ticket"`,
	)
}

// ============================================================================
// Multiple Agents Test
// ============================================================================

func TestAgentHCL_MultipleAgents(t *testing.T) {
	app := &discovery.App{
		ID:   TestAppID,
		Name: "Test App",
		Agents: []discovery.Agent{
			{
				ID:           TestAgentID,
				Name:         "Support Agent",
				Type:         "AgentElementum",
				Instructions: "Help with support",
			},
			{
				ID:                TestTargetAgentID,
				Name:              "Data Agent",
				Type:              "AgentSnowflake",
				Instructions:      "Query data",
				SnowflakeDatabase: "DB",
				SnowflakeSchema:   "SCHEMA",
				SnowflakeName:     "NAME",
			},
		},
	}

	imports := []ImportBlock{
		{ID: TestAppID, ResourceType: "elementum_app", ResourceName: "test_app"},
		{ID: TestAppID + ":" + TestAgentID, ResourceType: "elementum_agent", ResourceName: "support_agent"},
		{ID: TestAppID + ":" + TestTargetAgentID, ResourceType: "elementum_agent", ResourceName: "data_agent"},
	}
	uuidMap := make(map[string]string)

	generator := NewAgentHCLGenerator(app, imports, uuidMap)
	hcl := generator.GenerateAll()

	assertHCLContains(t, hcl,
		`resource "elementum_agent" "support_agent"`,
		`resource "elementum_agent" "data_agent"`,
		`type = "snowflake"`,
	)
}

// ============================================================================
// HCL Syntax Validation Tests
// ============================================================================

func TestAgentHCL_ValidSyntax(t *testing.T) {
	app := &discovery.App{
		ID:   TestAppID,
		Name: "Test App",
		Agents: []discovery.Agent{
			{
				ID:           TestAgentID,
				Name:         "Test Agent",
				Description:  "A test agent",
				Type:         "AgentElementum",
				Instructions: "Be helpful",
				FirstMessage: "Hello!",
				Tools: []discovery.AgentTool{
					{
						ID:          TestAgentToolID,
						Name:        "Create Record",
						Description: "Creates a record",
						Type:        "AgentCreateRecordTool",
						AspectID:    TestElementID,
						RawConfig: map[string]interface{}{
							"fields": []map[string]interface{}{
								{
									"field_id":    TestFieldID,
									"name":        "title",
									"description": "Title field",
									"required":    true,
								},
							},
						},
					},
				},
			},
		},
	}

	imports := []ImportBlock{
		{ID: TestAppID, ResourceType: "elementum_app", ResourceName: "test_app"},
		{ID: TestAppID + ":" + TestAgentID, ResourceType: "elementum_agent", ResourceName: "test_agent"},
		{ID: TestAppID + ":" + TestAgentID + ":" + TestAgentToolID, ResourceType: "elementum_agent_create_record_tool", ResourceName: "create_record"},
	}
	uuidMap := make(map[string]string)

	generator := NewAgentHCLGenerator(app, imports, uuidMap)
	hcl := generator.GenerateAll()

	// Validate HCL syntax
	validateHCLSyntax(t, hcl)
}

// ============================================================================
// Edge Cases
// ============================================================================

func TestAgentHCL_EmptyDescription(t *testing.T) {
	app := &discovery.App{
		ID:   TestAppID,
		Name: "Test App",
		Agents: []discovery.Agent{
			{
				ID:           TestAgentID,
				Name:         "No Description Agent",
				Description:  "", // Empty
				Type:         "AgentElementum",
				Instructions: "Help users",
			},
		},
	}

	imports := []ImportBlock{
		{ID: TestAppID, ResourceType: "elementum_app", ResourceName: "test_app"},
		{ID: TestAppID + ":" + TestAgentID, ResourceType: "elementum_agent", ResourceName: "no_description_agent"},
	}
	uuidMap := make(map[string]string)

	generator := NewAgentHCLGenerator(app, imports, uuidMap)
	hcl := generator.GenerateAll()

	// Should NOT contain description attribute
	if strings.Contains(hcl, "description =") {
		t.Errorf("Expected no description attribute for empty description, got:\n%s", hcl)
	}
}

func TestAgentHCL_EmptyFirstMessage(t *testing.T) {
	app := &discovery.App{
		ID:   TestAppID,
		Name: "Test App",
		Agents: []discovery.Agent{
			{
				ID:           TestAgentID,
				Name:         "No First Message Agent",
				Type:         "AgentElementum",
				Instructions: "Help users",
				FirstMessage: "", // Empty
			},
		},
	}

	imports := []ImportBlock{
		{ID: TestAppID, ResourceType: "elementum_app", ResourceName: "test_app"},
		{ID: TestAppID + ":" + TestAgentID, ResourceType: "elementum_agent", ResourceName: "no_first_message_agent"},
	}
	uuidMap := make(map[string]string)

	generator := NewAgentHCLGenerator(app, imports, uuidMap)
	hcl := generator.GenerateAll()

	// Should NOT contain first_message attribute
	if strings.Contains(hcl, "first_message =") {
		t.Errorf("Expected no first_message attribute for empty first message, got:\n%s", hcl)
	}
}

func TestAgentToolHCL_NoImportMatch(t *testing.T) {
	app := &discovery.App{
		ID:   TestAppID,
		Name: "Test App",
		Agents: []discovery.Agent{
			{
				ID:           TestAgentID,
				Name:         "Test Agent",
				Type:         "AgentElementum",
				Instructions: "Help",
				Tools: []discovery.AgentTool{
					{
						ID:       TestAgentToolID,
						Name:     "Orphan Tool",
						Type:     "AgentSearchAspectTool",
						AspectID: TestElementID,
					},
				},
			},
		},
	}

	// Import for agent but NOT for tool
	imports := []ImportBlock{
		{ID: TestAppID, ResourceType: "elementum_app", ResourceName: "test_app"},
		{ID: TestAppID + ":" + TestAgentID, ResourceType: "elementum_agent", ResourceName: "test_agent"},
		// No import for tool
	}
	uuidMap := make(map[string]string)

	generator := NewAgentHCLGenerator(app, imports, uuidMap)
	hcl := generator.GenerateAll()

	// Agent should be present
	assertHCLContains(t, hcl, `resource "elementum_agent" "test_agent"`)

	// Tool should NOT be present
	if strings.Contains(hcl, "elementum_agent_tool") {
		t.Errorf("Expected no agent_tool resource when no import matches, got:\n%s", hcl)
	}
}

func TestAgentToolHCL_UnknownToolType(t *testing.T) {
	app := &discovery.App{
		ID:   TestAppID,
		Name: "Test App",
		Agents: []discovery.Agent{
			{
				ID:           TestAgentID,
				Name:         "Test Agent",
				Type:         "AgentElementum",
				Instructions: "Help",
				Tools: []discovery.AgentTool{
					{
						ID:   TestAgentToolID,
						Name: "Unknown Tool",
						Type: "AgentUnknownTool", // Unknown type
					},
				},
			},
		},
	}

	// No import for the unknown tool type - it won't match any known resource type
	imports := []ImportBlock{
		{ID: TestAppID, ResourceType: "elementum_app", ResourceName: "test_app"},
		{ID: TestAppID + ":" + TestAgentID, ResourceType: "elementum_agent", ResourceName: "test_agent"},
	}
	uuidMap := make(map[string]string)

	generator := NewAgentHCLGenerator(app, imports, uuidMap)
	hcl := generator.GenerateAll()

	// Agent should be present
	assertHCLContains(t, hcl, `resource "elementum_agent" "test_agent"`)

	// Unknown tool type should NOT generate any tool resource (since it can't match a known resource type)
	if strings.Contains(hcl, "unknown_tool") {
		t.Errorf("Expected no resource for unknown tool type, got:\n%s", hcl)
	}
}
