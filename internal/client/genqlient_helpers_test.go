// Copyright (c) ElementumAI, Inc.
// SPDX-License-Identifier: MPL-2.0

package client

import (
	"encoding/json"
	"testing"
)

func TestExtractRoles(t *testing.T) {
	t.Parallel()

	t.Run("nil response", func(t *testing.T) {
		if roles := ExtractRoles(nil); roles != nil {
			t.Errorf("expected nil, got %v", roles)
		}
	})

	t.Run("App roles", func(t *testing.T) {
		jsonData := `{
			"organization": {
				"aspect": {
					"__typename": "AspectApp",
					"roles": {
						"edges": [
							{
								"node": {
									"id": "role-1",
									"name": "Admin",
									"description": "Administrator",
									"autoShare": ["ALWAYS"],
									"managed": true,
									"tags": ["system"],
									"users": { "total": 5 },
									"groups": { "total": 2 }
								}
							}
						]
					}
				}
			}
		}`
		var resp GetRolesResponse
		if err := json.Unmarshal([]byte(jsonData), &resp); err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}

		roles := ExtractRoles(&resp)
		if len(roles) != 1 {
			t.Fatalf("expected 1 role, got %d", len(roles))
		}
		if roles[0].ID != "role-1" || roles[0].Name != "Admin" {
			t.Errorf("unexpected role: %+v", roles[0])
		}
		if len(roles[0].AutoShare) != 1 || roles[0].AutoShare[0] != "ALWAYS" {
			t.Errorf("unexpected autoShare: %v", roles[0].AutoShare)
		}
	})

	t.Run("Element roles", func(t *testing.T) {
		jsonData := `{
			"organization": {
				"aspect": {
					"__typename": "AspectElement",
					"roles": {
						"edges": [
							{
								"node": {
									"id": "role-2",
									"name": "User",
									"autoShare": [],
									"managed": false,
									"tags": [],
									"users": { "total": 10 },
									"groups": { "total": 0 }
								}
							}
						]
					}
				}
			}
		}`
		var resp GetRolesResponse
		if err := json.Unmarshal([]byte(jsonData), &resp); err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}

		roles := ExtractRoles(&resp)
		if len(roles) != 1 {
			t.Fatalf("expected 1 role, got %d", len(roles))
		}
		if roles[0].ID != "role-2" || roles[0].Name != "User" {
			t.Errorf("unexpected role: %+v", roles[0])
		}
	})
}

func TestExtractCloudLink(t *testing.T) {
	t.Parallel()

	t.Run("nil response", func(t *testing.T) {
		if cl := ExtractCloudLinkFromGetCloudLink(nil); cl != nil {
			t.Errorf("expected nil, got %v", cl)
		}
	})

	t.Run("ExtractCloudLinkFromGetCloudLink", func(t *testing.T) {
		jsonData := `{
			"organization": {
				"cloudLink": {
					"__typename": "CloudLinkApi",
					"id": "cl-1",
					"name": "My API",
					"cache": true,
					"encryptCache": false,
					"usageType": "EXTERNAL",
					"system": false,
					"valid": true
				}
			}
		}`
		var resp GetCloudLinkResponse
		if err := json.Unmarshal([]byte(jsonData), &resp); err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}

		cl := ExtractCloudLinkFromGetCloudLink(&resp)
		if cl == nil {
			t.Fatal("expected non-nil cloudlink")
		}
		if cl.ID != "cl-1" || cl.Type != "api" || cl.Name != "My API" {
			t.Errorf("unexpected cloudlink: %+v", cl)
		}
	})

	t.Run("ExtractCloudLinksFromList", func(t *testing.T) {
		jsonData := `{
			"organization": {
				"cloudLinks": {
					"edges": [
						{
							"node": {
								"__typename": "CloudLinkSnowflake",
								"id": "cl-2",
								"name": "My Snowflake",
								"cache": false,
								"encryptCache": false,
								"usageType": "INTERNAL",
								"system": true,
								"valid": true
							}
						}
					]
				}
			}
		}`
		var resp ListCloudLinksResponse
		if err := json.Unmarshal([]byte(jsonData), &resp); err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}

		cls := ExtractCloudLinksFromList(&resp)
		if len(cls) != 1 {
			t.Fatalf("expected 1 cloudlink, got %d", len(cls))
		}
		if cls[0].ID != "cl-2" || cls[0].Type != "snowflake" {
			t.Errorf("unexpected cloudlink: %+v", cls[0])
		}
	})
}

func TestExtractTables(t *testing.T) {
	t.Parallel()

	t.Run("ExtractTableFromGetTable", func(t *testing.T) {
		jsonData := `{
			"organization": {
				"table": {
					"id": "tbl-1",
					"name": "My Table",
					"handle": "my_table",
					"type": "standard",
					"configured": true,
					"editable": true,
					"cloudManaged": false,
					"category": { "id": "cat-1" },
					"source": { "id": "src-1" },
					"cloudLink": {
						"__typename": "CloudLinkApi",
						"id": "cl-1",
						"name": "My API"
					},
					"fields": [
						{
							"__typename": "TableFieldCloud",
							"id": "f-1",
							"name": "Field 1",
							"fieldType": "TEXT"
						}
					]
				}
			}
		}`
		var resp GetTableResponse
		if err := json.Unmarshal([]byte(jsonData), &resp); err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}

		tbl := ExtractTableFromGetTable(&resp)
		if tbl == nil {
			t.Fatal("expected non-nil table")
		}
		if tbl.ID != "tbl-1" || tbl.Name != "My Table" || *tbl.CategoryID != "cat-1" {
			t.Errorf("unexpected table: %+v", tbl)
		}
		if len(tbl.Fields) != 1 || tbl.Fields[0].Name != "Field 1" {
			t.Errorf("unexpected fields: %v", tbl.Fields)
		}
	})

	t.Run("ExtractTablesFromGetTables", func(t *testing.T) {
		jsonData := `{
			"organization": {
				"tables": {
					"edges": [
						{
							"node": {
								"id": "tbl-2",
								"name": "Table 2",
								"handle": "table_2",
								"type": "virtual",
								"configured": true,
								"editable": false,
								"cloudManaged": true
							}
						}
					]
				}
			}
		}`
		var resp GetTablesResponse
		if err := json.Unmarshal([]byte(jsonData), &resp); err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}

		tbls := ExtractTablesFromGetTables(&resp)
		if len(tbls) != 1 {
			t.Fatalf("expected 1 table, got %d", len(tbls))
		}
		if tbls[0].ID != "tbl-2" || tbls[0].Name != "Table 2" {
			t.Errorf("unexpected table: %+v", tbls[0])
		}
	})
}

func TestExtractDatamine(t *testing.T) {
	t.Parallel()

	t.Run("ExtractDatamine", func(t *testing.T) {
		jsonData := `{
			"organization": {
				"table": {
					"datamine": {
						"id": "dm-1",
						"name": "My Datamine",
						"active": true,
						"app": { "id": "app-1" },
						"table": { "id": "tbl-1" },
						"primaryColumns": [
							{
								"__typename": "TableFieldCloud",
								"id": "col-1"
							}
						],
						"schedule": {
							"__typename": "DatamineScheduleFixed",
							"timeUnit": "HOURS",
							"value": 1
						}
					}
				}
			}
		}`
		var resp GetDatamineResponse
		if err := json.Unmarshal([]byte(jsonData), &resp); err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}

		dm := ExtractDatamine(&resp)
		if dm == nil {
			t.Fatal("expected non-nil datamine")
		}
		if dm.ID != "dm-1" || len(dm.PrimaryColumnIDs) != 1 || dm.ScheduleFixed.TimeUnit != "HOURS" {
			t.Errorf("unexpected datamine: %+v", dm)
		}
	})

	t.Run("ExtractDatamineFromList", func(t *testing.T) {
		jsonData := `{
			"organization": {
				"table": {
					"datamines": {
						"edges": [
							{
								"node": {
									"id": "dm-2",
									"name": "Target Datamine",
									"active": false,
									"primaryColumns": [
										{
											"__typename": "TableFieldCalculation",
											"id": "col-2"
										}
									],
									"schedule": {
										"__typename": "DatamineScheduleCron",
										"cronExpression": "0 0 * * *"
									}
								}
							}
						]
					}
				}
			}
		}`
		var resp GetDataminesResponse
		if err := json.Unmarshal([]byte(jsonData), &resp); err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}

		dm := ExtractDatamineFromList(&resp, "Target Datamine")
		if dm == nil {
			t.Fatal("expected non-nil datamine")
		}
		if dm.ID != "dm-2" || dm.ScheduleCron.CronExpression != "0 0 * * *" {
			t.Errorf("unexpected datamine: %+v", dm)
		}
	})
}

func TestExtractRelationship(t *testing.T) {
	t.Parallel()

	jsonData := `{
		"organization": {
			"autoRelation": {
				"id": "rel-1",
				"aspect": {
					"__typename": "AspectApp",
					"id": "app-1"
				},
				"relatedAspectTable": {
					"__typename": "Table",
					"tableId": "tbl-2"
				},
				"columns": [
					{
						"aspectField": {
							"__typename": "AspectTextField",
							"id": "f-1"
						},
						"relatedAspectTableField": {
							"__typename": "TableFieldReference",
							"tableFieldId": "f-2"
						}
					}
				],
				"filter": {"type":"EQUALS"}
			}
		}
	}`
	var resp GetAutoRelationResponse
	if err := json.Unmarshal([]byte(jsonData), &resp); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	rel, err := ExtractRelationship(&resp)
	if err != nil {
		t.Fatalf("ExtractRelationship failed: %v", err)
	}
	if rel == nil {
		t.Fatal("expected non-nil relationship")
	}
	if rel.ID != "rel-1" || rel.RelatedObjectID != "tbl-2" || len(rel.Columns) != 1 {
		t.Errorf("unexpected relationship: %+v", rel)
	}
	if rel.Filter["type"] != "EQUALS" {
		t.Errorf("unexpected filter: %v", rel.Filter)
	}
}

func TestExtractSearchTables(t *testing.T) {
	t.Parallel()

	t.Run("ExtractTableSearchTables", func(t *testing.T) {
		jsonData := `{
			"organization": {
				"table": {
					"searchTables": {
						"edges": [
							{
								"node": {
								"__typename": "TableSearchSnowflakeTable",
									"id": "st-1",
								"field": {
									"__typename": "TableFieldCloud",
									"id": "f-1",
									"name": "Field 1"
								}
								}
							}
						]
					}
				}
			}
		}`
		var resp GetTableSearchTablesResponse
		if err := json.Unmarshal([]byte(jsonData), &resp); err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}

		tables := ExtractTableSearchTables(&resp)
		if len(tables) != 1 || tables[0].FieldName != "Field 1" {
			t.Errorf("unexpected tables: %v", tables)
		}
	})
}

func TestExtractFieldsBasic(t *testing.T) {
	t.Parallel()

	jsonData := `{
		"organization": {
			"aspect": {
				"__typename": "AspectApp",
				"fields": {
					"edges": [
						{
							"node": {
								"__typename": "AspectTextField",
								"id": "f-1",
								"name": "Field 1"
							}
						}
					]
				}
			}
		}
	}`
	var resp GetAspectFieldsResponse
	if err := json.Unmarshal([]byte(jsonData), &resp); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	fields := ExtractFieldsBasic(&resp)
	if len(fields) != 1 || fields[0].Name != "Field 1" || fields[0].Typename != "AspectTextField" {
		t.Errorf("unexpected fields: %v", fields)
	}
}

func TestExtractDiscoveryAutomations(t *testing.T) {
	t.Parallel()

	jsonData := `{
		"organization": {
			"aspect": {
				"__typename": "AspectApp",
				"automations": {
					"edges": [
						{
							"node": {
								"id": "auto-1",
								"name": "Auto 1",
								"status": "PUBLISHED",
								"current": {
									"id": "wf-1",
									"triggers": [
										{
											"__typename": "WorkflowRecordCreateTrigger",
											"id": "tr-1"
										}
									],
									"tasks": [
										{
											"__typename": "WorkflowCreateRecordTask",
											"id": "ta-1",
											"name": "Task 1",
											"aspect": {
												"__typename": "AspectApp",
												"id": "app-2"
											}
										}
									]
								}
							}
						}
					]
				}
			}
		}
	}`
	var resp GetAspectAutomationsForDiscoveryResponse
	if err := json.Unmarshal([]byte(jsonData), &resp); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	autos := ExtractDiscoveryAutomations(&resp)
	if len(autos) != 1 || autos[0].Name != "Auto 1" {
		t.Fatalf("unexpected automations: %v", autos)
	}
	if autos[0].WorkflowID != "wf-1" || len(autos[0].Triggers) != 1 || len(autos[0].Tasks) != 1 {
		t.Errorf("unexpected automation details: %+v", autos[0])
	}
	if autos[0].Triggers[0].Type != "record_created" {
		t.Errorf("unexpected trigger type: %s", autos[0].Triggers[0].Type)
	}
	if autos[0].Tasks[0].Type != "create_record" || autos[0].Tasks[0].ObjectID != "app-2" {
		t.Errorf("unexpected task: %+v", autos[0].Tasks[0])
	}
}

func TestExtractWorkflowValueReferencesData(t *testing.T) {
	t.Parallel()

	jsonData := `{
		"organization": {
			"automation": {
				"id": "auto-1",
				"name": "Auto 1",
				"entity": {
					"__typename": "AspectApp",
					"id": "app-1",
					"fields": {
						"edges": [
							{
								"node": {
									"__typename": "AspectTextField",
									"id": "f-1",
									"name": "Field 1"
								}
							}
						]
					}
				},
				"current": {
					"id": "wf-1",
					"triggers": [
						{
							"__typename": "WorkflowOnDemandTrigger",
							"id": "tr-1",
							"parameters": [
								{ "id": "p-1", "name": "Param 1", "fieldType": "TEXT", "required": true }
							]
						}
					],
					"tasks": [
						{
							"__typename": "WorkflowRecordSearchTask",
							"id": "ta-1",
							"name": "Search Task",
							"aspect": { "__typename": "AspectApp", "id": "app-2", "name": "App 2" },
							"filterValueReferences": {
								"valueReferences": [
									{ "id": "vr-1", "label": "Label 1", "value": {"v":1} }
								]
							}
						},
						{
							"__typename": "WorkflowVariableTask",
							"id": "ta-2",
							"name": "Var Task",
							"variables": [
								{
									"__typename": "WorkflowVariableTaskParameterCreate",
									"name": "var1",
									"type": "TEXT"
								}
							]
						}
					]
				}
			}
		}
	}`
	var resp GetWorkflowValueReferencesResponse
	if err := json.Unmarshal([]byte(jsonData), &resp); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	data := ExtractWorkflowValueReferencesData(&resp)
	if data == nil {
		t.Fatal("expected non-nil data")
	}
	if data.ID != "auto-1" || data.EntityID != "app-1" || len(data.Fields) != 1 {
		t.Errorf("unexpected automation data: %+v", data)
	}
	if data.Current == nil || len(data.Current.Triggers) != 1 || len(data.Current.Tasks) != 2 {
		t.Errorf("unexpected workflow data: %+v", data.Current)
	}
	if data.Current.Triggers[0].Parameters[0].Name != "Param 1" {
		t.Errorf("unexpected parameter: %v", data.Current.Triggers[0].Parameters[0])
	}
	if data.Current.Tasks[0].AspectID != "app-2" || data.Current.Tasks[0].FilterValueReferences[0].ID != "vr-1" {
		t.Errorf("unexpected search task data: %+v", data.Current.Tasks[0])
	}
	if data.Current.Tasks[1].Variables[0].Name != "var1" {
		t.Errorf("unexpected variable: %v", data.Current.Tasks[1].Variables[0])
	}
}

func TestExtractSearchTablesForAspects(t *testing.T) {
	t.Parallel()

	jsonData := `{
		"organization": {
			"aspect": {
				"__typename": "AspectApp",
				"searchTables": {
					"edges": [
						{
							"node": {
								"__typename": "AspectSearchSnowflakeTable",
								"id": "st-1",
								"field": {
									"__typename": "AspectTextField",
									"id": "f-1",
									"name": "Field 1"
								}
							}
						}
					]
				}
			}
		}
	}`
	var resp GetAspectSearchTablesResponse
	if err := json.Unmarshal([]byte(jsonData), &resp); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	tables := ExtractSearchTables(&resp)
	if len(tables) != 1 || tables[0].FieldName != "Field 1" {
		t.Errorf("unexpected tables: %v", tables)
	}
}

func TestExtractSnowflakeTableFromSchema(t *testing.T) {
	t.Parallel()

	jsonData := `{
		"organization": {
			"cloudLink": {
				"__typename": "CloudLinkSnowflake",
				"table": {
					"name": "TBL",
					"databaseName": "DB",
					"schemaName": "SC",
					"type": "BASE_TABLE",
					"rows": 100,
					"bytes": 1024,
					"columns": [
						{
							"name": "COL1",
							"databaseType": "VARCHAR",
							"type": "TEXT",
							"nullable": true,
							"primaryKey": true,
							"uniqueKey": false,
							"comment": "cmt"
						}
					]
				}
			}
		}
	}`
	var resp GetSnowflakeTableSchemaResponse
	if err := json.Unmarshal([]byte(jsonData), &resp); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	table := ExtractSnowflakeTableFromSchema(&resp)
	if table == nil {
		t.Fatal("expected non-nil table")
	}
	if table.Name != "TBL" || len(table.Columns) != 1 || table.Columns[0].Name != "COL1" {
		t.Errorf("unexpected snowflake table: %+v", table)
	}
}

func TestBuildDatamineNameFilter(t *testing.T) {
	t.Parallel()

	raw := BuildDatamineNameFilter("my-dm")
	var filter map[string]interface{}
	if err := json.Unmarshal(raw, &filter); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if filter["type"] != "EQUALS" {
		t.Errorf("expected type EQUALS, got %v", filter["type"])
	}
	val, ok := filter["value"].(map[string]interface{})
	if !ok {
		t.Fatal("expected value to be a map")
	}
	if val["value"] != "my-dm" {
		t.Errorf("expected value 'my-dm', got %v", val["value"])
	}
}

func TestExtractAspects(t *testing.T) {
	t.Parallel()

	t.Run("nil response", func(t *testing.T) {
		if aspects := ExtractAspects(nil); aspects != nil {
			t.Errorf("expected nil, got %v", aspects)
		}
	})

	t.Run("multiple aspects", func(t *testing.T) {
		jsonData := `{
			"organization": {
				"aspects": {
					"edges": [
						{
							"node": {
								"__typename": "AspectApp",
								"id": "app-1",
								"name": "App 1",
								"namespace": "ns1",
								"handle": "h1"
							}
						},
						{
							"node": {
								"__typename": "AspectElement",
								"id": "el-1",
								"name": "Element 1",
								"namespace": "ns2",
								"handle": "h2"
							}
						},
						{
							"node": {
								"__typename": "AspectTask",
								"id": "task-1",
								"name": "Task 1",
								"namespace": "ns3",
								"handle": "h3"
							}
						}
					]
				}
			}
		}`
		var resp SearchAspectsResponse
		if err := json.Unmarshal([]byte(jsonData), &resp); err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}

		aspects := ExtractAspects(&resp)
		if len(aspects) != 3 {
			t.Fatalf("expected 3 aspects, got %d", len(aspects))
		}
		if aspects[0].ID != "app-1" || aspects[0].Typename != "AspectApp" || aspects[0].Namespace != "ns1" {
			t.Errorf("unexpected aspect 0: %+v", aspects[0])
		}
		if aspects[1].ID != "el-1" || aspects[1].Typename != "AspectElement" || aspects[1].Namespace != "ns2" {
			t.Errorf("unexpected aspect 1: %+v", aspects[1])
		}
		if aspects[2].ID != "task-1" || aspects[2].Typename != "AspectTask" || aspects[2].Namespace != "ns3" {
			t.Errorf("unexpected aspect 2: %+v", aspects[2])
		}
	})
}

func TestExtractAIProviderConnectors(t *testing.T) {
	t.Parallel()

	t.Run("nil response", func(t *testing.T) {
		if connectors := ExtractAIProviderConnectors(nil); connectors != nil {
			t.Errorf("expected nil, got %v", connectors)
		}
	})

	t.Run("connectors", func(t *testing.T) {
		jsonData := `{
			"organization": {
				"aiProviderConnectors": {
					"edges": [
						{
							"node": {
								"__typename": "AiProviderConnectorLlm",
								"id": "conn-1",
								"model": { "name": "gpt-4" },
								"provider": {
									"__typename": "AiOpenAiProvider",
									"name": "OpenAI"
								}
							}
						}
					]
				}
			}
		}`
		var resp GetAIProviderConnectorsResponse
		if err := json.Unmarshal([]byte(jsonData), &resp); err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}

		connectors := ExtractAIProviderConnectors(&resp)
		if len(connectors) != 1 {
			t.Fatalf("expected 1 connector, got %d", len(connectors))
		}
		if connectors[0].ID != "conn-1" || connectors[0].ModelName != "gpt-4" || connectors[0].ProviderName != "OpenAI" {
			t.Errorf("unexpected connector: %+v", connectors[0])
		}
	})
}
