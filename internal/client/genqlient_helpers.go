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

package client

import (
	"encoding/json"
	"fmt"
	"strings"
)

// RoleData represents a simplified role structure for data sources.
type RoleData struct {
	ID          string
	Name        string
	Description *string
	AutoShare   []string
	Managed     bool
	Tags        []string
	UserCount   int
	GroupCount  int
}

// ExtractRoles extracts roles from GetRolesResponse, handling the interface type.
// Note: Only Apps and Elements have roles in the schema.
func ExtractRoles(resp *GetRolesResponse) []RoleData {
	if resp == nil || resp.Organization.Aspect == nil {
		return nil
	}

	var roles []RoleData

	switch aspect := (*resp.Organization.Aspect).(type) {
	case *GetRolesOrganizationAspectAspectApp:
		for _, edge := range aspect.Roles.Edges {
			node := edge.Node
			autoShare := make([]string, len(node.AutoShare))
			for i, as := range node.AutoShare {
				autoShare[i] = string(as)
			}
			roles = append(roles, RoleData{
				ID:          node.Id,
				Name:        node.Name,
				Description: node.Description,
				AutoShare:   autoShare,
				Managed:     node.Managed,
				Tags:        node.Tags,
				UserCount:   node.Users.Total,
				GroupCount:  node.Groups.Total,
			})
		}
	case *GetRolesOrganizationAspectAspectElement:
		for _, edge := range aspect.Roles.Edges {
			node := edge.Node
			autoShare := make([]string, len(node.AutoShare))
			for i, as := range node.AutoShare {
				autoShare[i] = string(as)
			}
			roles = append(roles, RoleData{
				ID:          node.Id,
				Name:        node.Name,
				Description: node.Description,
				AutoShare:   autoShare,
				Managed:     node.Managed,
				Tags:        node.Tags,
				UserCount:   node.Users.Total,
				GroupCount:  node.Groups.Total,
			})
		}
		// Note: AspectTask and AspectTransaction don't have roles in the schema
	}

	return roles
}

// AspectData represents simplified aspect data for lookups.
type AspectData struct {
	ID        string
	Name      string
	Typename  string
	Namespace string
	Handle    string
}

// ExtractAspects extracts aspects from SearchAspectsResponse.
func ExtractAspects(resp *SearchAspectsResponse) []AspectData {
	if resp == nil {
		return nil
	}

	var aspects []AspectData
	for _, edge := range resp.Organization.Aspects.Edges {
		node := edge.Node
		typename := ""
		if node.GetTypename() != nil {
			typename = *node.GetTypename()
		}

		// Get namespace and handle based on concrete type
		// Note: Only App, Element, and Task have namespace/handle; Transaction doesn't
		namespace := ""
		handle := ""
		switch concrete := node.(type) {
		case *SearchAspectsOrganizationAspectsAspectConnectionEdgesAspectEdgeNodeAspectApp:
			namespace = concrete.Namespace
			handle = concrete.Handle
		case *SearchAspectsOrganizationAspectsAspectConnectionEdgesAspectEdgeNodeAspectElement:
			namespace = concrete.Namespace
			handle = concrete.Handle
		case *SearchAspectsOrganizationAspectsAspectConnectionEdgesAspectEdgeNodeAspectTask:
			namespace = concrete.Namespace
			handle = concrete.Handle
		}

		aspects = append(aspects, AspectData{
			ID:        node.GetId(),
			Name:      node.GetName(),
			Typename:  typename,
			Namespace: namespace,
			Handle:    handle,
		})
	}
	return aspects
}

// AIProviderConnectorData represents simplified AI provider connector data.
type AIProviderConnectorData struct {
	ID           string
	ModelName    string
	ProviderName string
}

// ExtractAIProviderConnectors extracts AI provider connectors from response.
func ExtractAIProviderConnectors(resp *GetAIProviderConnectorsResponse) []AIProviderConnectorData {
	if resp == nil {
		return nil
	}

	var connectors []AIProviderConnectorData
	for _, edge := range resp.Organization.AiProviderConnectors.Edges {
		node := edge.Node
		connectors = append(connectors, AIProviderConnectorData{
			ID:           node.GetId(),
			ModelName:    node.GetModel().Name,
			ProviderName: node.GetProvider().GetName(),
		})
	}
	return connectors
}

// CloudLinkData represents simplified cloudlink data.
type CloudLinkData struct {
	ID           string
	Name         string
	Type         string
	Cache        bool
	EncryptCache bool
	UsageType    string
	System       bool
	Valid        *bool
}

// ExtractCloudLinkFromGetCloudLink extracts cloudlink data from GetCloudLinkResponse.
func ExtractCloudLinkFromGetCloudLink(resp *GetCloudLinkResponse) *CloudLinkData {
	if resp == nil || resp.Organization.CloudLink == nil {
		return nil
	}
	cl := *resp.Organization.CloudLink
	typename := ""
	if cl.GetTypename() != nil {
		typename = mapCloudLinkTypename(*cl.GetTypename())
	}
	return &CloudLinkData{
		ID:           cl.GetId(),
		Name:         cl.GetName(),
		Type:         typename,
		Cache:        cl.GetCache(),
		EncryptCache: cl.GetEncryptCache(),
		UsageType:    string(cl.GetUsageType()),
		System:       cl.GetSystem(),
		Valid:        cl.GetValid(),
	}
}

// mapCloudLinkTypename converts GraphQL __typename to Terraform type string.
func mapCloudLinkTypename(typename string) string {
	switch typename {
	case "CloudLinkApi":
		return "api"
	case "CloudLinkBigQuery":
		return "bigquery"
	case "CloudLinkDatabricks":
		return "databricks"
	case "CloudLinkElementum":
		return "elementum"
	case "CloudLinkElementumSnowflake":
		return "elementum_snowflake"
	case "CloudLinkSnowflake":
		return "snowflake"
	default:
		return typename
	}
}

// ExtractCloudLinksFromList extracts cloudlink data from ListCloudLinksResponse.
func ExtractCloudLinksFromList(resp *ListCloudLinksResponse) []CloudLinkData {
	if resp == nil || resp.Organization.CloudLinks == nil {
		return nil
	}

	var cloudlinks []CloudLinkData
	for _, edge := range resp.Organization.CloudLinks.Edges {
		cl := edge.Node
		typename := ""
		if cl.GetTypename() != nil {
			typename = mapCloudLinkTypename(*cl.GetTypename())
		}
		cloudlinks = append(cloudlinks, CloudLinkData{
			ID:           cl.GetId(),
			Name:         cl.GetName(),
			Type:         typename,
			Cache:        cl.GetCache(),
			EncryptCache: cl.GetEncryptCache(),
			UsageType:    string(cl.GetUsageType()),
			System:       cl.GetSystem(),
			Valid:        cl.GetValid(),
		})
	}
	return cloudlinks
}

// TableData represents simplified table data.
type TableData struct {
	ID            string
	Name          string
	Handle        string
	Type          string
	Configured    bool
	Editable      bool
	CloudManaged  bool
	CategoryID    *string
	SourceID      *string
	CloudLinkID   *string
	CloudLinkName *string // CloudLink name for data source generation
	// Snowflake connection fields (from cloudConnection)
	SnowflakeDatabaseName *string
	SnowflakeSchemaName   *string
	SnowflakeTableName    *string
	// Table fields (columns)
	Fields []SimpleTableField
}

// SimpleTableField represents a simplified table field/column.
type SimpleTableField struct {
	ID   string
	Name string
	Type string
}

// ExtractTableFromGetTable extracts table data from GetTableResponse.
func ExtractTableFromGetTable(resp *GetTableResponse) *TableData {
	if resp == nil {
		return nil
	}
	t := resp.Organization.Table
	data := &TableData{
		ID:           t.Id,
		Name:         t.Name,
		Handle:       t.Handle,
		Type:         t.Type,
		Configured:   t.Configured,
		Editable:     t.Editable,
		CloudManaged: t.CloudManaged,
	}
	if t.Category.Id != "" {
		data.CategoryID = &t.Category.Id
	}
	if t.Source != nil {
		data.SourceID = &t.Source.Id
	}
	if t.CloudLink != nil {
		cloudLinkID := (*t.CloudLink).GetId()
		cloudLinkName := (*t.CloudLink).GetName()
		data.CloudLinkID = &cloudLinkID
		data.CloudLinkName = &cloudLinkName
	}
	// Extract Snowflake connection data
	if t.CloudConnection != nil {
		switch conn := (*t.CloudConnection).(type) {
		case *GetTableOrganizationTableCloudConnectionSnowflakeConnection:
			data.SnowflakeDatabaseName = conn.DatabaseName
			data.SnowflakeSchemaName = conn.SchemaName
			data.SnowflakeTableName = conn.TableName
		}
	}

	// Extract table fields
	for _, field := range t.Fields {
		fieldData := SimpleTableField{
			ID:   field.GetId(),
			Name: field.GetName(),
			Type: string(field.GetFieldType()),
		}
		data.Fields = append(data.Fields, fieldData)
	}

	return data
}

// ExtractTablesFromGetTables extracts table data from GetTablesResponse.
// Note: List response has fewer fields than detail query - no Category/Source.
func ExtractTablesFromGetTables(resp *GetTablesResponse) []TableData {
	if resp == nil {
		return nil
	}

	var tables []TableData
	for _, edge := range resp.Organization.Tables.Edges {
		node := edge.Node
		data := TableData{
			ID:           node.Id,
			Name:         node.Name,
			Handle:       node.Handle,
			Type:         node.Type,
			Configured:   node.Configured,
			Editable:     node.Editable,
			CloudManaged: node.CloudManaged,
		}
		tables = append(tables, data)
	}
	return tables
}

// DatamineData represents simplified datamine data.
type DatamineData struct {
	ID               string
	Name             string
	Description      *string
	Active           bool
	AppID            *string
	TableID          *string
	PrimaryColumnIDs []string
	ScheduleFixed    *DatamineScheduleFixed
	ScheduleCron     *DatamineScheduleCron
}

// DatamineScheduleFixed represents a fixed interval schedule.
type DatamineScheduleFixed struct {
	TimeUnit    string
	Value       int
	Description *string
}

// DatamineScheduleCron represents a cron schedule.
type DatamineScheduleCron struct {
	CronExpression string
	Description    *string
}

// ExtractDatamine extracts datamine data from GetDatamineResponse.
func ExtractDatamine(resp *GetDatamineResponse) *DatamineData {
	if resp == nil || resp.Organization.Table.Datamine == nil {
		return nil
	}

	dm := resp.Organization.Table.Datamine
	data := &DatamineData{
		ID:          dm.Id,
		Name:        dm.Name,
		Description: dm.Description,
		Active:      dm.Active,
	}

	if dm.App != nil {
		data.AppID = &dm.App.Id
	}
	if dm.Table != nil {
		data.TableID = &dm.Table.Id
	}

	// Extract primary columns
	for _, col := range dm.PrimaryColumns {
		if col != nil {
			data.PrimaryColumnIDs = append(data.PrimaryColumnIDs, extractTableFieldID(col))
		}
	}

	// Extract schedule
	data.ScheduleFixed, data.ScheduleCron = extractDatamineSchedule(dm.Schedule)

	return data
}

// ExtractDatamineFromList extracts datamine data from GetDataminesResponse by name.
func ExtractDatamineFromList(resp *GetDataminesResponse, name string) *DatamineData {
	if resp == nil || resp.Organization.Table.Datamines == nil {
		return nil
	}

	for _, edge := range resp.Organization.Table.Datamines.Edges {
		if edge.Node.Name == name {
			dm := edge.Node
			data := &DatamineData{
				ID:          dm.Id,
				Name:        dm.Name,
				Description: dm.Description,
				Active:      dm.Active,
			}

			if dm.App != nil {
				data.AppID = &dm.App.Id
			}
			if dm.Table != nil {
				data.TableID = &dm.Table.Id
			}

			// Extract primary columns
			for _, col := range dm.PrimaryColumns {
				if col != nil {
					data.PrimaryColumnIDs = append(data.PrimaryColumnIDs, extractTableFieldIDFromList(col))
				}
			}

			// Extract schedule
			data.ScheduleFixed, data.ScheduleCron = extractDatamineScheduleFromList(dm.Schedule)

			return data
		}
	}
	return nil
}

func extractTableFieldID(col *GetDatamineOrganizationTableDataminePrimaryColumnsTableField) string {
	if col == nil {
		return ""
	}
	switch f := (*col).(type) {
	case *GetDatamineOrganizationTableDataminePrimaryColumnsTableFieldCalculation:
		return f.Id
	case *GetDatamineOrganizationTableDataminePrimaryColumnsTableFieldCloud:
		return f.Id
	case *GetDatamineOrganizationTableDataminePrimaryColumnsTableFieldReference:
		return f.Id
	default:
		return ""
	}
}

func extractTableFieldIDFromList(col *GetDataminesOrganizationTableDataminesDatamineConnectionEdgesDatamineEdgeNodeDataminePrimaryColumnsTableField) string {
	if col == nil {
		return ""
	}
	switch f := (*col).(type) {
	case *GetDataminesOrganizationTableDataminesDatamineConnectionEdgesDatamineEdgeNodeDataminePrimaryColumnsTableFieldCalculation:
		return f.Id
	case *GetDataminesOrganizationTableDataminesDatamineConnectionEdgesDatamineEdgeNodeDataminePrimaryColumnsTableFieldCloud:
		return f.Id
	case *GetDataminesOrganizationTableDataminesDatamineConnectionEdgesDatamineEdgeNodeDataminePrimaryColumnsTableFieldReference:
		return f.Id
	default:
		return ""
	}
}

func extractDatamineSchedule(schedule *GetDatamineOrganizationTableDatamineSchedule) (*DatamineScheduleFixed, *DatamineScheduleCron) {
	if schedule == nil {
		return nil, nil
	}

	switch s := (*schedule).(type) {
	case *GetDatamineOrganizationTableDatamineScheduleDatamineScheduleFixed:
		return &DatamineScheduleFixed{
			TimeUnit:    string(s.TimeUnit),
			Value:       s.Value,
			Description: s.Description,
		}, nil
	case *GetDatamineOrganizationTableDatamineScheduleDatamineScheduleCron:
		return nil, &DatamineScheduleCron{
			CronExpression: s.CronExpression,
			Description:    s.Description,
		}
	default:
		return nil, nil
	}
}

func extractDatamineScheduleFromList(schedule *GetDataminesOrganizationTableDataminesDatamineConnectionEdgesDatamineEdgeNodeDatamineSchedule) (*DatamineScheduleFixed, *DatamineScheduleCron) {
	if schedule == nil {
		return nil, nil
	}

	switch s := (*schedule).(type) {
	case *GetDataminesOrganizationTableDataminesDatamineConnectionEdgesDatamineEdgeNodeDatamineScheduleDatamineScheduleFixed:
		return &DatamineScheduleFixed{
			TimeUnit:    string(s.TimeUnit),
			Value:       s.Value,
			Description: s.Description,
		}, nil
	case *GetDataminesOrganizationTableDataminesDatamineConnectionEdgesDatamineEdgeNodeDatamineScheduleDatamineScheduleCron:
		return nil, &DatamineScheduleCron{
			CronExpression: s.CronExpression,
			Description:    s.Description,
		}
	default:
		return nil, nil
	}
}

// ExtractDatamineColumnIDFromList extracts the column ID from a datamine list response's primary column.
// This is an exported wrapper for use in cmd/elementum/discovery.
func ExtractDatamineColumnIDFromList(col *GetDataminesOrganizationTableDataminesDatamineConnectionEdgesDatamineEdgeNodeDataminePrimaryColumnsTableField) string {
	return extractTableFieldIDFromList(col)
}

// ExtractDatamineScheduleFromListNode extracts schedule data from a datamine list response node's schedule.
// This is an exported wrapper for use in cmd/elementum/discovery.
func ExtractDatamineScheduleFromListNode(schedule *GetDataminesOrganizationTableDataminesDatamineConnectionEdgesDatamineEdgeNodeDatamineSchedule) (*DatamineScheduleFixed, *DatamineScheduleCron) {
	return extractDatamineScheduleFromList(schedule)
}

// RelationshipData represents simplified relationship data.
type RelationshipData struct {
	ID              string
	ObjectID        string
	RelatedObjectID string
	Filter          map[string]interface{}
	Columns         []RelationshipColumnData
}

// RelationshipColumnData represents a column in a relationship.
type RelationshipColumnData struct {
	FieldID        string
	RelatedFieldID string
}

// ExtractRelationship extracts relationship data from GetAutoRelationResponse.
func ExtractRelationship(resp *GetAutoRelationResponse) (*RelationshipData, error) {
	if resp == nil || resp.Organization.AutoRelation == nil {
		return nil, nil
	}

	rel := resp.Organization.AutoRelation
	data := &RelationshipData{
		ID:       rel.Id,
		ObjectID: rel.Aspect.GetId(),
	}

	// Extract RelatedObjectID from the union type
	data.RelatedObjectID = extractAutoRelationTargetID(rel.RelatedAspectTable)

	// Extract columns
	for _, col := range rel.Columns {
		data.Columns = append(data.Columns, RelationshipColumnData{
			FieldID:        col.AspectField.GetId(),
			RelatedFieldID: extractRelatedFieldID(col.RelatedAspectTableField),
		})
	}

	// Extract filter
	if rel.Filter != nil {
		var filter map[string]interface{}
		if err := json.Unmarshal(*rel.Filter, &filter); err == nil {
			data.Filter = filter
		}
	}

	return data, nil
}

func extractAutoRelationTargetID(target AutoRelationTarget) string {
	if target == nil {
		return ""
	}
	switch t := target.(type) {
	case *AutoRelationTargetAspectApp:
		return t.AppId
	case *AutoRelationTargetAspectElement:
		return t.ElementId
	case *AutoRelationTargetAspectTask:
		return t.TaskId
	case *AutoRelationTargetAspectTransaction:
		// Transaction doesn't have an ID field in this context
		return ""
	case *AutoRelationTargetTable:
		return t.TableId
	default:
		return ""
	}
}

func extractRelatedFieldID(field GetAutoRelationOrganizationAutoRelationColumnsAutoColumnRelationRelatedAspectTableField) string {
	if field == nil {
		return ""
	}
	switch f := field.(type) {
	case *GetAutoRelationOrganizationAutoRelationColumnsAutoColumnRelationRelatedAspectTableFieldAspectAttachmentField:
		return f.AspectFieldId
	case *GetAutoRelationOrganizationAutoRelationColumnsAutoColumnRelationRelatedAspectTableFieldAspectBooleanField:
		return f.AspectFieldId
	case *GetAutoRelationOrganizationAutoRelationColumnsAutoColumnRelationRelatedAspectTableFieldAspectDateField:
		return f.AspectFieldId
	case *GetAutoRelationOrganizationAutoRelationColumnsAutoColumnRelationRelatedAspectTableFieldAspectDateTimeField:
		return f.AspectFieldId
	case *GetAutoRelationOrganizationAutoRelationColumnsAutoColumnRelationRelatedAspectTableFieldAspectDecimalField:
		return f.AspectFieldId
	case *GetAutoRelationOrganizationAutoRelationColumnsAutoColumnRelationRelatedAspectTableFieldAspectGroupField:
		return f.AspectFieldId
	case *GetAutoRelationOrganizationAutoRelationColumnsAutoColumnRelationRelatedAspectTableFieldAspectHandleField:
		return f.AspectFieldId
	case *GetAutoRelationOrganizationAutoRelationColumnsAutoColumnRelationRelatedAspectTableFieldAspectHtmlField:
		return f.AspectFieldId
	case *GetAutoRelationOrganizationAutoRelationColumnsAutoColumnRelationRelatedAspectTableFieldAspectMultiPicklistField:
		return f.AspectFieldId
	case *GetAutoRelationOrganizationAutoRelationColumnsAutoColumnRelationRelatedAspectTableFieldAspectNumberField:
		return f.AspectFieldId
	case *GetAutoRelationOrganizationAutoRelationColumnsAutoColumnRelationRelatedAspectTableFieldAspectPicklistField:
		return f.AspectFieldId
	case *GetAutoRelationOrganizationAutoRelationColumnsAutoColumnRelationRelatedAspectTableFieldAspectTextField:
		return f.AspectFieldId
	case *GetAutoRelationOrganizationAutoRelationColumnsAutoColumnRelationRelatedAspectTableFieldAspectUserField:
		return f.AspectFieldId
	case *GetAutoRelationOrganizationAutoRelationColumnsAutoColumnRelationRelatedAspectTableFieldTableFieldCalculation:
		return f.TableFieldId
	case *GetAutoRelationOrganizationAutoRelationColumnsAutoColumnRelationRelatedAspectTableFieldTableFieldCloud:
		return f.TableFieldId
	case *GetAutoRelationOrganizationAutoRelationColumnsAutoColumnRelationRelatedAspectTableFieldTableFieldReference:
		return f.TableFieldId
	default:
		return ""
	}
}

// SearchTableData represents simplified search table data.
type SearchTableData struct {
	ID           string
	FieldID      string
	FieldName    string
	Status       string
	ErrorMessage string
}

// TableSearchTableData represents simplified table search table data.
type TableSearchTableData struct {
	ID           string
	FieldID      string
	FieldName    string
	Status       string
	ErrorMessage string
}

// ExtractTableSearchTables extracts table search tables from GetTableSearchTablesResponse.
func ExtractTableSearchTables(resp *GetTableSearchTablesResponse) []TableSearchTableData {
	if resp == nil {
		return nil
	}

	var tables []TableSearchTableData
	searchTables := resp.Organization.Table.SearchTables
	if searchTables != nil {
		for _, edge := range searchTables.Edges {
			errorMsg := ""
			if edge.Node.GetErrorMessage() != nil {
				errorMsg = *edge.Node.GetErrorMessage()
			}
			tables = append(tables, TableSearchTableData{
				ID:           edge.Node.GetId(),
				FieldID:      edge.Node.GetField().GetId(),
				FieldName:    edge.Node.GetField().GetName(),
				Status:       string(edge.Node.GetStatus()),
				ErrorMessage: errorMsg,
			})
		}
	}

	return tables
}

// ExtractSearchTables extracts search tables from GetAspectSearchTablesResponse.
func ExtractSearchTables(resp *GetAspectSearchTablesResponse) []SearchTableData {
	if resp == nil || resp.Organization.Aspect == nil {
		return nil
	}

	var tables []SearchTableData

	switch aspect := (*resp.Organization.Aspect).(type) {
	case *GetAspectSearchTablesOrganizationAspectAspectApp:
		if aspect.SearchTables != nil {
			for _, edge := range aspect.SearchTables.Edges {
				errorMsg := ""
				if edge.Node.GetErrorMessage() != nil {
					errorMsg = *edge.Node.GetErrorMessage()
				}
				tables = append(tables, SearchTableData{
					ID:           edge.Node.GetId(),
					FieldID:      edge.Node.GetField().GetId(),
					FieldName:    edge.Node.GetField().GetName(),
					Status:       string(edge.Node.GetStatus()),
					ErrorMessage: errorMsg,
				})
			}
		}
	case *GetAspectSearchTablesOrganizationAspectAspectElement:
		if aspect.SearchTables != nil {
			for _, edge := range aspect.SearchTables.Edges {
				errorMsg := ""
				if edge.Node.GetErrorMessage() != nil {
					errorMsg = *edge.Node.GetErrorMessage()
				}
				tables = append(tables, SearchTableData{
					ID:           edge.Node.GetId(),
					FieldID:      edge.Node.GetField().GetId(),
					FieldName:    edge.Node.GetField().GetName(),
					Status:       string(edge.Node.GetStatus()),
					ErrorMessage: errorMsg,
				})
			}
		}
		// Note: AspectTask doesn't have SearchTables in the schema
	}

	return tables
}

// FieldData represents simplified field data.
type FieldData struct {
	ID       string
	Name     string
	Typename string
	// Raw JSON for complex field data - allows access to all fields
	RawJSON json.RawMessage
}

// ExtractFieldsBasic extracts basic field info from GetAspectFieldsResponse.
// Note: Only App, Element, and Task have fields; Transaction doesn't.
func ExtractFieldsBasic(resp *GetAspectFieldsResponse) []FieldData {
	if resp == nil || resp.Organization.Aspect == nil {
		return nil
	}

	var fields []FieldData

	// Debug: log the actual type
	aspectType := fmt.Sprintf("%T", *resp.Organization.Aspect)
	_ = aspectType // used below

	switch aspect := (*resp.Organization.Aspect).(type) {
	case *GetAspectFieldsOrganizationAspectAspectApp:
		for _, edge := range aspect.Fields.Edges {
			typename := ""
			if edge.Node.GetTypename() != nil {
				typename = *edge.Node.GetTypename()
			}
			fields = append(fields, FieldData{
				ID:       edge.Node.GetId(),
				Name:     edge.Node.GetName(),
				Typename: typename,
			})
		}
	case *GetAspectFieldsOrganizationAspectAspectElement:
		for _, edge := range aspect.Fields.Edges {
			typename := ""
			if edge.Node.GetTypename() != nil {
				typename = *edge.Node.GetTypename()
			}
			fields = append(fields, FieldData{
				ID:       edge.Node.GetId(),
				Name:     edge.Node.GetName(),
				Typename: typename,
			})
		}
	case *GetAspectFieldsOrganizationAspectAspectTask:
		for _, edge := range aspect.Fields.Edges {
			typename := ""
			if edge.Node.GetTypename() != nil {
				typename = *edge.Node.GetTypename()
			}
			fields = append(fields, FieldData{
				ID:       edge.Node.GetId(),
				Name:     edge.Node.GetName(),
				Typename: typename,
			})
		}
	}

	return fields
}

// ExtractFieldsBasicNoValues extracts basic field info from GetAspectFieldsNoValuesResponse.
// This is the lightweight version that doesn't include picklist values.
// Note: Only App, Element, and Task have fields; Transaction doesn't.
func ExtractFieldsBasicNoValues(resp *GetAspectFieldsNoValuesResponse) []FieldData {
	if resp == nil || resp.Organization.Aspect == nil {
		return nil
	}

	var fields []FieldData

	switch aspect := (*resp.Organization.Aspect).(type) {
	case *GetAspectFieldsNoValuesOrganizationAspectAspectApp:
		for _, edge := range aspect.Fields.Edges {
			typename := ""
			if edge.Node.GetTypename() != nil {
				typename = *edge.Node.GetTypename()
			}
			fields = append(fields, FieldData{
				ID:       edge.Node.GetId(),
				Name:     edge.Node.GetName(),
				Typename: typename,
			})
		}
	case *GetAspectFieldsNoValuesOrganizationAspectAspectElement:
		for _, edge := range aspect.Fields.Edges {
			typename := ""
			if edge.Node.GetTypename() != nil {
				typename = *edge.Node.GetTypename()
			}
			fields = append(fields, FieldData{
				ID:       edge.Node.GetId(),
				Name:     edge.Node.GetName(),
				Typename: typename,
			})
		}
	case *GetAspectFieldsNoValuesOrganizationAspectAspectTask:
		for _, edge := range aspect.Fields.Edges {
			typename := ""
			if edge.Node.GetTypename() != nil {
				typename = *edge.Node.GetTypename()
			}
			fields = append(fields, FieldData{
				ID:       edge.Node.GetId(),
				Name:     edge.Node.GetName(),
				Typename: typename,
			})
		}
	}

	return fields
}

// BuildDatamineNameFilter creates a JSON raw message for a datamine name filter.
func BuildDatamineNameFilter(name string) json.RawMessage {
	filter := map[string]interface{}{
		"type": "EQUALS",
		"leftValue": map[string]interface{}{
			"type":  "FIELD",
			"field": "name",
		},
		"value": map[string]interface{}{
			"type":  "TEXT",
			"value": name,
		},
	}
	filterRaw, _ := json.Marshal(filter)
	return filterRaw
}

// WorkflowValueRefAutomationData represents simplified automation data for value references.
type WorkflowValueRefAutomationData struct {
	ID       string
	Name     string
	Current  *WorkflowValueRefWorkflowData
	EntityID string
	Fields   []WorkflowValueRefFieldData
}

// WorkflowValueRefWorkflowData represents workflow data for value references.
type WorkflowValueRefWorkflowData struct {
	ID       string
	Triggers []WorkflowValueRefTriggerData
	Tasks    []WorkflowValueRefTaskData
}

// WorkflowValueRefTriggerData represents trigger data for value references.
type WorkflowValueRefTriggerData struct {
	ID         string
	Typename   string
	Parameters []WorkflowValueRefParameterData
}

// WorkflowValueRefParameterData represents on-demand trigger parameter data.
type WorkflowValueRefParameterData struct {
	ID        string
	Name      string
	FieldType string
	Required  bool
}

// WorkflowValueRefTaskData represents task data for value references.
type WorkflowValueRefTaskData struct {
	ID                    string
	Name                  string
	Typename              string
	AspectID              string
	AspectName            string
	FilterValueReferences []WorkflowValueRefValueReference
	Variables             []WorkflowValueRefVariable
	Calculations          []WorkflowValueRefCalculation
}

// WorkflowValueRefValueReference represents a value reference from filter context.
type WorkflowValueRefValueReference struct {
	ID    string
	Label string
	Value interface{}
}

// WorkflowValueRefVariable represents a variable from a VariableTask.
type WorkflowValueRefVariable struct {
	Name string
	Type string
}

// WorkflowValueRefCalculation represents a calculation from a CalculationTask.
type WorkflowValueRefCalculation struct {
	ID   string
	Name string
}

// WorkflowValueRefFieldData represents field data from the automation's entity.
type WorkflowValueRefFieldData struct {
	ID       string
	Name     string
	Typename string
}

// ExtractWorkflowValueReferencesData extracts automation data from GetWorkflowValueReferencesResponse.
func ExtractWorkflowValueReferencesData(resp *GetWorkflowValueReferencesResponse) *WorkflowValueRefAutomationData {
	if resp == nil || resp.Organization.Automation == nil {
		return nil
	}

	automation := resp.Organization.Automation
	data := &WorkflowValueRefAutomationData{
		ID:   automation.Id,
		Name: automation.Name,
	}

	// Extract entity fields (only AspectApp has fields in this query)
	if automation.Entity != nil {
		switch entity := automation.Entity.(type) {
		case *GetWorkflowValueReferencesOrganizationAutomationEntityAspectApp:
			data.EntityID = entity.Id
			for _, edge := range entity.Fields.Edges {
				typename := ""
				if edge.Node.GetTypename() != nil {
					typename = *edge.Node.GetTypename()
				}
				data.Fields = append(data.Fields, WorkflowValueRefFieldData{
					ID:       edge.Node.GetId(),
					Name:     edge.Node.GetName(),
					Typename: typename,
				})
			}
		}
	}

	// Extract current workflow data
	if automation.Current != nil {
		data.Current = &WorkflowValueRefWorkflowData{
			ID: automation.Current.Id,
		}

		// Extract triggers
		for _, trigger := range automation.Current.Triggers {
			triggerData := WorkflowValueRefTriggerData{
				ID:       trigger.GetId(),
				Typename: "",
			}
			if trigger.GetTypename() != nil {
				triggerData.Typename = *trigger.GetTypename()
			}

			// Extract parameters for on-demand triggers
			if onDemand, ok := trigger.(*GetWorkflowValueReferencesOrganizationAutomationCurrentWorkflowTriggersWorkflowOnDemandTrigger); ok {
				for _, param := range onDemand.Parameters {
					paramID := ""
					if param.Id != nil {
						paramID = *param.Id
					}
					triggerData.Parameters = append(triggerData.Parameters, WorkflowValueRefParameterData{
						ID:        paramID,
						Name:      param.Name,
						FieldType: string(param.FieldType),
						Required:  param.Required,
					})
				}
			}

			data.Current.Triggers = append(data.Current.Triggers, triggerData)
		}

		// Extract tasks
		for _, task := range automation.Current.Tasks {
			taskData := WorkflowValueRefTaskData{
				ID:       task.GetId(),
				Typename: "",
			}
			if task.GetTypename() != nil {
				taskData.Typename = *task.GetTypename()
			}
			if task.GetName() != nil {
				taskData.Name = *task.GetName()
			}

			// Extract task-specific data based on type
			switch t := task.(type) {
			case *GetWorkflowValueReferencesOrganizationAutomationCurrentWorkflowTasksWorkflowRecordSearchTask:
				if t.Aspect != nil {
					taskData.AspectID = (*t.Aspect).GetId()
					taskData.AspectName = (*t.Aspect).GetName()
				}
				if t.FilterValueReferences != nil {
					for _, ref := range t.FilterValueReferences.ValueReferences {
						refData := WorkflowValueRefValueReference{
							ID: ref.Id,
						}
						if ref.Label != nil {
							refData.Label = *ref.Label
						}
						if ref.Value != nil {
							var val interface{}
							_ = json.Unmarshal(*ref.Value, &val)
							refData.Value = val
						}
						taskData.FilterValueReferences = append(taskData.FilterValueReferences, refData)
					}
				}

			case *GetWorkflowValueReferencesOrganizationAutomationCurrentWorkflowTasksWorkflowUserSearchTask:
				if t.FilterValueReferences != nil {
					for _, ref := range t.FilterValueReferences.ValueReferences {
						refData := WorkflowValueRefValueReference{
							ID: ref.Id,
						}
						if ref.Label != nil {
							refData.Label = *ref.Label
						}
						if ref.Value != nil {
							var val interface{}
							_ = json.Unmarshal(*ref.Value, &val)
							refData.Value = val
						}
						taskData.FilterValueReferences = append(taskData.FilterValueReferences, refData)
					}
				}

			case *GetWorkflowValueReferencesOrganizationAutomationCurrentWorkflowTasksWorkflowVariableTask:
				for _, v := range t.Variables {
					varType := ""
					varName := ""
					switch vv := v.(type) {
					case *WorkflowVariableTaskFieldsVariablesWorkflowVariableTaskParameterCreate:
						varName = vv.Name
						varType = string(vv.Type)
					case *WorkflowVariableTaskFieldsVariablesWorkflowVariableTaskParameterUpdate:
						// Update doesn't have direct name/type, skip
						continue
					}
					if varName != "" {
						taskData.Variables = append(taskData.Variables, WorkflowValueRefVariable{
							Name: varName,
							Type: varType,
						})
					}
				}

			case *GetWorkflowValueReferencesOrganizationAutomationCurrentWorkflowTasksWorkflowCalculationTask:
				for _, calc := range t.Calculations {
					taskData.Calculations = append(taskData.Calculations, WorkflowValueRefCalculation{
						ID:   calc.Id,
						Name: calc.Name,
					})
				}
			}

			data.Current.Tasks = append(data.Current.Tasks, taskData)
		}
	}

	return data
}

// DiscoveryAutomation represents simplified automation data for discovery.
type DiscoveryAutomation struct {
	ID           string
	Name         string
	Status       string
	WorkflowID   string
	HasPublished bool
	HasDraft     bool
	Triggers     []DiscoveryTrigger
	Tasks        []DiscoveryTask
}

// DiscoveryTrigger represents simplified trigger data for discovery.
type DiscoveryTrigger struct {
	ID         string
	Type       string
	DatamineID string // For datamine triggers
}

// DiscoveryTask represents simplified task data for discovery.
type DiscoveryTask struct {
	ID              string
	Type            string
	Name            string
	PreviousID      string // Parent task ID
	ObjectID        string // From aspect field (create_record, update_field, record_search, etc.)
	RelatedObjectID string // From relatedAspect field (find_related_records)
}

// ExtractDiscoveryAutomations extracts automations from GetAspectAutomationsForDiscoveryResponse.
func ExtractDiscoveryAutomations(resp *GetAspectAutomationsForDiscoveryResponse) []DiscoveryAutomation {
	if resp == nil || resp.Organization.Aspect == nil {
		return nil
	}

	var automations []DiscoveryAutomation

	switch aspect := (*resp.Organization.Aspect).(type) {
	case *GetAspectAutomationsForDiscoveryOrganizationAspectAspectApp:
		if aspect.Automations != nil {
			for _, edge := range aspect.Automations.Edges {
				auto := extractDiscoveryAutomation(
					edge.Node.Id,
					edge.Node.Name,
					string(edge.Node.Status),
					edge.Node.Current,
					edge.Node.Draft,
				)
				automations = append(automations, auto)
			}
		}
	case *GetAspectAutomationsForDiscoveryOrganizationAspectAspectElement:
		if aspect.Automations != nil {
			for _, edge := range aspect.Automations.Edges {
				auto := extractDiscoveryAutomationElement(
					edge.Node.Id,
					edge.Node.Name,
					string(edge.Node.Status),
					edge.Node.Current,
					edge.Node.Draft,
				)
				automations = append(automations, auto)
			}
		}
	}

	return automations
}

// extractDiscoveryAutomation extracts automation from app response.
func extractDiscoveryAutomation(
	id, name, status string,
	current *GetAspectAutomationsForDiscoveryOrganizationAspectAspectAppAutomationsAutomationConnectionEdgesAutomationEdgeNodeAutomationCurrentWorkflow,
	draft *GetAspectAutomationsForDiscoveryOrganizationAspectAspectAppAutomationsAutomationConnectionEdgesAutomationEdgeNodeAutomationDraftWorkflow,
) DiscoveryAutomation {
	auto := DiscoveryAutomation{
		ID:     id,
		Name:   name,
		Status: status,
	}

	if current != nil {
		auto.WorkflowID = current.Id
		auto.HasPublished = true
		auto.Triggers = extractDiscoveryTriggersApp(current.Triggers)
		auto.Tasks = extractDiscoveryTasksApp(current.Tasks)
	}

	if draft != nil {
		auto.HasDraft = true
		if auto.WorkflowID == "" {
			auto.WorkflowID = draft.Id
		}
	}

	return auto
}

// extractDiscoveryAutomationElement extracts automation from element response.
func extractDiscoveryAutomationElement(
	id, name, status string,
	current *GetAspectAutomationsForDiscoveryOrganizationAspectAspectElementAutomationsAutomationConnectionEdgesAutomationEdgeNodeAutomationCurrentWorkflow,
	draft *GetAspectAutomationsForDiscoveryOrganizationAspectAspectElementAutomationsAutomationConnectionEdgesAutomationEdgeNodeAutomationDraftWorkflow,
) DiscoveryAutomation {
	auto := DiscoveryAutomation{
		ID:     id,
		Name:   name,
		Status: status,
	}

	if current != nil {
		auto.WorkflowID = current.Id
		auto.HasPublished = true
		auto.Triggers = extractDiscoveryTriggersElement(current.Triggers)
		auto.Tasks = extractDiscoveryTasksElement(current.Tasks)
	}

	if draft != nil {
		auto.HasDraft = true
		if auto.WorkflowID == "" {
			auto.WorkflowID = draft.Id
		}
	}

	return auto
}

// extractDiscoveryTriggersApp extracts triggers from app workflow.
func extractDiscoveryTriggersApp(triggers []GetAspectAutomationsForDiscoveryOrganizationAspectAspectAppAutomationsAutomationConnectionEdgesAutomationEdgeNodeAutomationCurrentWorkflowTriggersWorkflowTrigger) []DiscoveryTrigger {
	var result []DiscoveryTrigger
	for _, t := range triggers {
		trigger := DiscoveryTrigger{
			ID:   t.GetId(),
			Type: mapTriggerTypename(t.GetTypename()),
		}

		// Check for datamine trigger
		if dt, ok := t.(*GetAspectAutomationsForDiscoveryOrganizationAspectAspectAppAutomationsAutomationConnectionEdgesAutomationEdgeNodeAutomationCurrentWorkflowTriggersWorkflowDatamineTrigger); ok {
			if dt.Datamine != nil {
				trigger.DatamineID = dt.Datamine.Id
			}
		}

		result = append(result, trigger)
	}
	return result
}

// extractDiscoveryTriggersElement extracts triggers from element workflow.
func extractDiscoveryTriggersElement(triggers []GetAspectAutomationsForDiscoveryOrganizationAspectAspectElementAutomationsAutomationConnectionEdgesAutomationEdgeNodeAutomationCurrentWorkflowTriggersWorkflowTrigger) []DiscoveryTrigger {
	var result []DiscoveryTrigger
	for _, t := range triggers {
		trigger := DiscoveryTrigger{
			ID:   t.GetId(),
			Type: mapTriggerTypename(t.GetTypename()),
		}

		// Check for datamine trigger
		if dt, ok := t.(*GetAspectAutomationsForDiscoveryOrganizationAspectAspectElementAutomationsAutomationConnectionEdgesAutomationEdgeNodeAutomationCurrentWorkflowTriggersWorkflowDatamineTrigger); ok {
			if dt.Datamine != nil {
				trigger.DatamineID = dt.Datamine.Id
			}
		}

		result = append(result, trigger)
	}
	return result
}

// extractDiscoveryTasksApp extracts tasks from app workflow.
func extractDiscoveryTasksApp(tasks []GetAspectAutomationsForDiscoveryOrganizationAspectAspectAppAutomationsAutomationConnectionEdgesAutomationEdgeNodeAutomationCurrentWorkflowTasksWorkflowTask) []DiscoveryTask {
	var result []DiscoveryTask
	for _, t := range tasks {
		task := DiscoveryTask{
			ID:   t.GetId(),
			Type: mapTaskTypename(t.GetTypename()),
		}

		if t.GetName() != nil {
			task.Name = *t.GetName()
		}

		if t.GetPrevious() != nil {
			task.PreviousID = (*t.GetPrevious()).GetId()
		}

		// Extract ObjectID from aspect field based on task type
		switch tt := t.(type) {
		case *GetAspectAutomationsForDiscoveryOrganizationAspectAspectAppAutomationsAutomationConnectionEdgesAutomationEdgeNodeAutomationCurrentWorkflowTasksWorkflowCreateRecordTask:
			if tt.Aspect != nil {
				task.ObjectID = (*tt.Aspect).GetId()
			}
		case *GetAspectAutomationsForDiscoveryOrganizationAspectAspectAppAutomationsAutomationConnectionEdgesAutomationEdgeNodeAutomationCurrentWorkflowTasksWorkflowUpdateFieldTask:
			if tt.Aspect != nil {
				task.ObjectID = (*tt.Aspect).GetId()
			}
		case *GetAspectAutomationsForDiscoveryOrganizationAspectAspectAppAutomationsAutomationConnectionEdgesAutomationEdgeNodeAutomationCurrentWorkflowTasksWorkflowRecordSearchTask:
			if tt.Aspect != nil {
				task.ObjectID = (*tt.Aspect).GetId()
			}
		case *GetAspectAutomationsForDiscoveryOrganizationAspectAspectAppAutomationsAutomationConnectionEdgesAutomationEdgeNodeAutomationCurrentWorkflowTasksWorkflowAspectRecordFieldLockingTask:
			if tt.Aspect != nil {
				task.ObjectID = (*tt.Aspect).GetId()
			}
		case *GetAspectAutomationsForDiscoveryOrganizationAspectAspectAppAutomationsAutomationConnectionEdgesAutomationEdgeNodeAutomationCurrentWorkflowTasksWorkflowBulkExcelTask:
			if tt.Aspect != nil {
				task.ObjectID = (*tt.Aspect).GetId()
			}
		case *GetAspectAutomationsForDiscoveryOrganizationAspectAspectAppAutomationsAutomationConnectionEdgesAutomationEdgeNodeAutomationCurrentWorkflowTasksWorkflowFindRelatedRecordsTask:
			task.RelatedObjectID = tt.RelatedAspect.GetId()
		}

		result = append(result, task)
	}
	return result
}

// extractDiscoveryTasksElement extracts tasks from element workflow.
func extractDiscoveryTasksElement(tasks []GetAspectAutomationsForDiscoveryOrganizationAspectAspectElementAutomationsAutomationConnectionEdgesAutomationEdgeNodeAutomationCurrentWorkflowTasksWorkflowTask) []DiscoveryTask {
	var result []DiscoveryTask
	for _, t := range tasks {
		task := DiscoveryTask{
			ID:   t.GetId(),
			Type: mapTaskTypename(t.GetTypename()),
		}

		if t.GetName() != nil {
			task.Name = *t.GetName()
		}

		if t.GetPrevious() != nil {
			task.PreviousID = (*t.GetPrevious()).GetId()
		}

		// Extract ObjectID from aspect field based on task type
		switch tt := t.(type) {
		case *GetAspectAutomationsForDiscoveryOrganizationAspectAspectElementAutomationsAutomationConnectionEdgesAutomationEdgeNodeAutomationCurrentWorkflowTasksWorkflowCreateRecordTask:
			if tt.Aspect != nil {
				task.ObjectID = (*tt.Aspect).GetId()
			}
		case *GetAspectAutomationsForDiscoveryOrganizationAspectAspectElementAutomationsAutomationConnectionEdgesAutomationEdgeNodeAutomationCurrentWorkflowTasksWorkflowUpdateFieldTask:
			if tt.Aspect != nil {
				task.ObjectID = (*tt.Aspect).GetId()
			}
		case *GetAspectAutomationsForDiscoveryOrganizationAspectAspectElementAutomationsAutomationConnectionEdgesAutomationEdgeNodeAutomationCurrentWorkflowTasksWorkflowRecordSearchTask:
			if tt.Aspect != nil {
				task.ObjectID = (*tt.Aspect).GetId()
			}
		case *GetAspectAutomationsForDiscoveryOrganizationAspectAspectElementAutomationsAutomationConnectionEdgesAutomationEdgeNodeAutomationCurrentWorkflowTasksWorkflowAspectRecordFieldLockingTask:
			if tt.Aspect != nil {
				task.ObjectID = (*tt.Aspect).GetId()
			}
		case *GetAspectAutomationsForDiscoveryOrganizationAspectAspectElementAutomationsAutomationConnectionEdgesAutomationEdgeNodeAutomationCurrentWorkflowTasksWorkflowBulkExcelTask:
			if tt.Aspect != nil {
				task.ObjectID = (*tt.Aspect).GetId()
			}
		case *GetAspectAutomationsForDiscoveryOrganizationAspectAspectElementAutomationsAutomationConnectionEdgesAutomationEdgeNodeAutomationCurrentWorkflowTasksWorkflowFindRelatedRecordsTask:
			task.RelatedObjectID = tt.RelatedAspect.GetId()
		}

		result = append(result, task)
	}
	return result
}

// mapTriggerTypename maps GraphQL __typename to trigger type name.
func mapTriggerTypename(typename *string) string {
	if typename == nil {
		return "unknown"
	}
	switch *typename {
	case "WorkflowRecordCreateTrigger":
		return "record_created"
	case "WorkflowRecordUpdateTrigger":
		return "record_updated"
	case "WorkflowDatamineTrigger":
		return "datamine"
	case "WorkflowWebhookTrigger":
		return "webhook"
	case "WorkflowOnDemandTrigger":
		return "on_demand"
	case "WorkflowUserInitiatedTrigger":
		return "user_initiated"
	default:
		return "unknown"
	}
}

// mapTaskTypename maps GraphQL __typename to task type name.
func mapTaskTypename(typename *string) string {
	if typename == nil {
		return "unknown"
	}
	// Strip "Workflow" prefix and "Task" suffix, then convert to snake_case
	name := *typename
	if len(name) > 8 && name[:8] == "Workflow" {
		name = name[8:]
	}
	if len(name) > 4 && name[len(name)-4:] == "Task" {
		name = name[:len(name)-4]
	}
	// Convert camelCase to snake_case
	result := ""
	for i, r := range name {
		if i > 0 && r >= 'A' && r <= 'Z' {
			result += "_"
		}
		result += string(r)
	}
	return strings.ToLower(result)
}

// SnowflakeTableData represents a Snowflake table schema with columns.
type SnowflakeTableData struct {
	Name         string
	DatabaseName string
	SchemaName   string
	Type         string
	Rows         int64
	Bytes        int64
	Columns      []SnowflakeColumnData
}

// SnowflakeColumnData represents a column in a Snowflake table.
type SnowflakeColumnData struct {
	Name         string
	DatabaseType string
	Type         string
	Nullable     bool
	PrimaryKey   bool
	UniqueKey    bool
	Comment      string
}

// ExtractSnowflakeTableFromSchema extracts Snowflake table data from GetSnowflakeTableSchemaResponse.
// This function handles both CloudLinkSnowflake and CloudLinkElementumSnowflake types.
func ExtractSnowflakeTableFromSchema(resp *GetSnowflakeTableSchemaResponse) *SnowflakeTableData {
	if resp == nil || resp.Organization.CloudLink == nil {
		return nil
	}

	// Handle polymorphic CloudLink type
	switch cl := (*resp.Organization.CloudLink).(type) {
	case *GetSnowflakeTableSchemaOrganizationCloudLinkCloudLinkSnowflake:
		if cl.Table == nil {
			return nil
		}
		return extractSnowflakeTableFromCloudLinkSnowflake(cl.Table)
	case *GetSnowflakeTableSchemaOrganizationCloudLinkCloudLinkElementumSnowflake:
		if cl.Table == nil {
			return nil
		}
		return extractSnowflakeTableFromElementumSnowflake(cl.Table)
	}

	return nil
}

// extractSnowflakeTableFromCloudLinkSnowflake extracts table data from CloudLinkSnowflake.
func extractSnowflakeTableFromCloudLinkSnowflake(table *GetSnowflakeTableSchemaOrganizationCloudLinkCloudLinkSnowflakeTable) *SnowflakeTableData {
	data := &SnowflakeTableData{
		Name:         table.Name,
		DatabaseName: table.DatabaseName,
		SchemaName:   table.SchemaName,
		Type:         string(table.Type),
		Rows:         int64(table.Rows),
		Columns:      make([]SnowflakeColumnData, 0, len(table.Columns)),
	}

	if table.Bytes != nil {
		data.Bytes = int64(*table.Bytes)
	}

	for _, col := range table.Columns {
		data.Columns = append(data.Columns, SnowflakeColumnData{
			Name:         col.Name,
			DatabaseType: col.DatabaseType,
			Type:         string(col.Type),
			Nullable:     col.Nullable,
			PrimaryKey:   col.PrimaryKey,
			UniqueKey:    col.UniqueKey,
			Comment:      col.Comment,
		})
	}

	return data
}

// extractSnowflakeTableFromElementumSnowflake extracts table data from CloudLinkElementumSnowflake.
func extractSnowflakeTableFromElementumSnowflake(table *GetSnowflakeTableSchemaOrganizationCloudLinkCloudLinkElementumSnowflakeTable) *SnowflakeTableData {
	data := &SnowflakeTableData{
		Name:         table.Name,
		DatabaseName: table.DatabaseName,
		SchemaName:   table.SchemaName,
		Type:         string(table.Type),
		Rows:         int64(table.Rows),
		Columns:      make([]SnowflakeColumnData, 0, len(table.Columns)),
	}

	if table.Bytes != nil {
		data.Bytes = int64(*table.Bytes)
	}

	for _, col := range table.Columns {
		data.Columns = append(data.Columns, SnowflakeColumnData{
			Name:         col.Name,
			DatabaseType: col.DatabaseType,
			Type:         string(col.Type),
			Nullable:     col.Nullable,
			PrimaryKey:   col.PrimaryKey,
			UniqueKey:    col.UniqueKey,
			Comment:      col.Comment,
		})
	}

	return data
}

// SnowflakeDatabaseData represents simplified database data for discovery.
type SnowflakeDatabaseData struct {
	Name    string
	Owner   string
	Comment string
}

// ExtractSnowflakeDatabases extracts databases from ListSnowflakeDatabasesResponse.
func ExtractSnowflakeDatabases(resp *ListSnowflakeDatabasesResponse) []SnowflakeDatabaseData {
	if resp == nil || resp.Organization.CloudLink == nil {
		return nil
	}

	var databases []SnowflakeDatabaseData

	switch cl := (*resp.Organization.CloudLink).(type) {
	case *ListSnowflakeDatabasesOrganizationCloudLinkCloudLinkSnowflake:
		for _, db := range cl.Databases {
			comment := ""
			if db.Comment != nil {
				comment = *db.Comment
			}
			owner := ""
			if db.Owner != nil {
				owner = *db.Owner
			}
			databases = append(databases, SnowflakeDatabaseData{
				Name:    db.Name,
				Owner:   owner,
				Comment: comment,
			})
		}
	case *ListSnowflakeDatabasesOrganizationCloudLinkCloudLinkElementumSnowflake:
		for _, db := range cl.Databases {
			comment := ""
			if db.Comment != nil {
				comment = *db.Comment
			}
			owner := ""
			if db.Owner != nil {
				owner = *db.Owner
			}
			databases = append(databases, SnowflakeDatabaseData{
				Name:    db.Name,
				Owner:   owner,
				Comment: comment,
			})
		}
	}

	return databases
}

// SnowflakeSchemaData represents simplified schema data for discovery.
type SnowflakeSchemaData struct {
	Name         string
	DatabaseName string
	Owner        string
	Comment      string
}

// ExtractSnowflakeSchemas extracts schemas from ListSnowflakeSchemasResponse.
func ExtractSnowflakeSchemas(resp *ListSnowflakeSchemasResponse) []SnowflakeSchemaData {
	if resp == nil || resp.Organization.CloudLink == nil {
		return nil
	}

	var schemas []SnowflakeSchemaData

	switch cl := (*resp.Organization.CloudLink).(type) {
	case *ListSnowflakeSchemasOrganizationCloudLinkCloudLinkSnowflake:
		for _, s := range cl.Schemas {
			comment := ""
			if s.Comment != nil {
				comment = *s.Comment
			}
			owner := ""
			if s.Owner != nil {
				owner = *s.Owner
			}
			dbName := ""
			if s.DatabaseName != nil {
				dbName = *s.DatabaseName
			}
			schemas = append(schemas, SnowflakeSchemaData{
				Name:         s.Name,
				DatabaseName: dbName,
				Owner:        owner,
				Comment:      comment,
			})
		}
	case *ListSnowflakeSchemasOrganizationCloudLinkCloudLinkElementumSnowflake:
		for _, s := range cl.Schemas {
			comment := ""
			if s.Comment != nil {
				comment = *s.Comment
			}
			owner := ""
			if s.Owner != nil {
				owner = *s.Owner
			}
			dbName := ""
			if s.DatabaseName != nil {
				dbName = *s.DatabaseName
			}
			schemas = append(schemas, SnowflakeSchemaData{
				Name:         s.Name,
				DatabaseName: dbName,
				Owner:        owner,
				Comment:      comment,
			})
		}
	}

	return schemas
}

// SnowflakeTableListData represents simplified table data for discovery (list view).
type SnowflakeTableListData struct {
	Name         string
	DatabaseName string
	SchemaName   string
	Type         string
	Rows         int64
	Bytes        int64
}

// ExtractSnowflakeTables extracts tables from ListSnowflakeTablesResponse.
func ExtractSnowflakeTables(resp *ListSnowflakeTablesResponse) []SnowflakeTableListData {
	if resp == nil || resp.Organization.CloudLink == nil {
		return nil
	}

	var tables []SnowflakeTableListData

	switch cl := (*resp.Organization.CloudLink).(type) {
	case *ListSnowflakeTablesOrganizationCloudLinkCloudLinkSnowflake:
		for _, t := range cl.Tables {
			var bytes int64
			if t.Bytes != nil {
				bytes = int64(*t.Bytes)
			}
			tables = append(tables, SnowflakeTableListData{
				Name:         t.Name,
				DatabaseName: t.DatabaseName,
				SchemaName:   t.SchemaName,
				Type:         string(t.Type),
				Rows:         int64(t.Rows),
				Bytes:        bytes,
			})
		}
	case *ListSnowflakeTablesOrganizationCloudLinkCloudLinkElementumSnowflake:
		for _, t := range cl.Tables {
			var bytes int64
			if t.Bytes != nil {
				bytes = int64(*t.Bytes)
			}
			tables = append(tables, SnowflakeTableListData{
				Name:         t.Name,
				DatabaseName: t.DatabaseName,
				SchemaName:   t.SchemaName,
				Type:         string(t.Type),
				Rows:         int64(t.Rows),
				Bytes:        bytes,
			})
		}
	}

	return tables
}

// StoredFunctionParameter represents a stored function input parameter.
type StoredFunctionParameter struct {
	ID   string
	Name string
	Type string // DatasourceColumnType as string
}

// StoredFunctionOutput represents the return type of a stored function.
type StoredFunctionOutput struct {
	OutputType string // "simple" or "table"
	// For simple outputs
	ReturnType string
	Nullable   bool
	// For table outputs
	Columns []StoredFunctionOutputColumn
}

// StoredFunctionOutputColumn represents a column in a table output.
type StoredFunctionOutputColumn struct {
	Name string
	Type string
}

// StoredFunctionData represents simplified stored function data.
type StoredFunctionData struct {
	ID            string
	Name          string
	DisplayName   string
	Description   string
	DatabaseName  string
	SchemaName    string
	Type          string
	Configured    bool
	CloudLinkID   string
	CloudLinkName string
	Parameters    []StoredFunctionParameter
	Output        *StoredFunctionOutput
}

// ExtractStoredFunctionFromGetStoredFunction extracts stored function data from GetStoredFunctionResponse.
func ExtractStoredFunctionFromGetStoredFunction(resp *GetStoredFunctionResponse) *StoredFunctionData {
	if resp == nil || resp.Organization.CloudLink == nil {
		return nil
	}

	cloudLink := *resp.Organization.CloudLink

	// Only CloudLinkSnowflake has storedSnowflakeFunction
	switch cl := cloudLink.(type) {
	case *GetStoredFunctionOrganizationCloudLinkCloudLinkSnowflake:
		sf := cl.StoredSnowflakeFunction
		data := &StoredFunctionData{
			ID:            sf.Id,
			Name:          sf.Name,
			DisplayName:   sf.DisplayName,
			Description:   sf.Description,
			DatabaseName:  sf.DatabaseName,
			SchemaName:    sf.SchemaName,
			Type:          string(sf.Type),
			Configured:    sf.Configured,
			CloudLinkID:   cl.Id,
			CloudLinkName: cl.Name,
		}
		if sf.CloudLink != nil {
			data.CloudLinkID = (*sf.CloudLink).GetId()
			data.CloudLinkName = (*sf.CloudLink).GetName()
		}

		// Extract parameters
		for _, arg := range sf.InputArgs {
			name := ""
			if arg.Name != nil {
				name = *arg.Name
			}
			data.Parameters = append(data.Parameters, StoredFunctionParameter{
				ID:   arg.Id,
				Name: name,
				Type: string(arg.Type),
			})
		}

		// Extract output
		if sf.OutputArg != nil {
			data.Output = extractOutputFromGetStoredFunction(sf.OutputArg)
		}

		return data
	}

	return nil
}

// extractOutputFromGetStoredFunction extracts output info from GetStoredFunction response.
func extractOutputFromGetStoredFunction(outputArg GetStoredFunctionOrganizationCloudLinkCloudLinkSnowflakeStoredSnowflakeFunctionOutputArg) *StoredFunctionOutput {
	output := &StoredFunctionOutput{}

	switch o := outputArg.(type) {
	case *GetStoredFunctionOrganizationCloudLinkCloudLinkSnowflakeStoredSnowflakeFunctionOutputArgStoredSnowflakeFunctionSimpleOutput:
		output.OutputType = "simple"
		output.ReturnType = string(o.ReturnType)
		output.Nullable = o.Nullable
	case *GetStoredFunctionOrganizationCloudLinkCloudLinkSnowflakeStoredSnowflakeFunctionOutputArgStoredSnowflakeFunctionTableOutput:
		output.OutputType = "table"
		for _, col := range o.Columns {
			output.Columns = append(output.Columns, StoredFunctionOutputColumn{
				Name: col.Name,
				Type: string(col.ColumnType),
			})
		}
	}

	return output
}

// ExtractStoredFunctionsFromList extracts stored functions from ListStoredFunctionsResponse.
func ExtractStoredFunctionsFromList(resp *ListStoredFunctionsResponse) []StoredFunctionData {
	if resp == nil || resp.Organization.CloudLink == nil {
		return nil
	}

	cloudLink := *resp.Organization.CloudLink
	var functions []StoredFunctionData

	switch cl := cloudLink.(type) {
	case *ListStoredFunctionsOrganizationCloudLinkCloudLinkSnowflake:
		for _, edge := range cl.StoredSnowflakeFunctions.Edges {
			sf := edge.Node
			data := StoredFunctionData{
				ID:            sf.Id,
				Name:          sf.Name,
				DisplayName:   sf.DisplayName,
				Description:   sf.Description,
				DatabaseName:  sf.DatabaseName,
				SchemaName:    sf.SchemaName,
				Type:          string(sf.Type),
				Configured:    sf.Configured,
				CloudLinkID:   cl.Id,
				CloudLinkName: cl.Name,
			}
			if sf.CloudLink != nil {
				data.CloudLinkID = (*sf.CloudLink).GetId()
				data.CloudLinkName = (*sf.CloudLink).GetName()
			}

			// Extract parameters
			for _, arg := range sf.InputArgs {
				name := ""
				if arg.Name != nil {
					name = *arg.Name
				}
				data.Parameters = append(data.Parameters, StoredFunctionParameter{
					ID:   arg.Id,
					Name: name,
					Type: string(arg.Type),
				})
			}

			// Extract output
			if sf.OutputArg != nil {
				data.Output = extractOutputFromListStoredFunctions(sf.OutputArg)
			}

			functions = append(functions, data)
		}
	}

	return functions
}

// extractOutputFromListStoredFunctions extracts output info from ListStoredFunctions response.
func extractOutputFromListStoredFunctions(outputArg ListStoredFunctionsOrganizationCloudLinkCloudLinkSnowflakeStoredSnowflakeFunctionsStoredSnowflakeFunctionConnectionEdgesStoredSnowflakeFunctionEdgeNodeStoredSnowflakeFunctionOutputArg) *StoredFunctionOutput {
	output := &StoredFunctionOutput{}

	switch o := outputArg.(type) {
	case *ListStoredFunctionsOrganizationCloudLinkCloudLinkSnowflakeStoredSnowflakeFunctionsStoredSnowflakeFunctionConnectionEdgesStoredSnowflakeFunctionEdgeNodeStoredSnowflakeFunctionOutputArgStoredSnowflakeFunctionSimpleOutput:
		output.OutputType = "simple"
		output.ReturnType = string(o.ReturnType)
		output.Nullable = o.Nullable
	case *ListStoredFunctionsOrganizationCloudLinkCloudLinkSnowflakeStoredSnowflakeFunctionsStoredSnowflakeFunctionConnectionEdgesStoredSnowflakeFunctionEdgeNodeStoredSnowflakeFunctionOutputArgStoredSnowflakeFunctionTableOutput:
		output.OutputType = "table"
		for _, col := range o.Columns {
			output.Columns = append(output.Columns, StoredFunctionOutputColumn{
				Name: col.Name,
				Type: string(col.ColumnType),
			})
		}
	}

	return output
}

// ExtractAllStoredFunctions extracts all stored functions from ListAllStoredFunctionsResponse.
func ExtractAllStoredFunctions(resp *ListAllStoredFunctionsResponse) []StoredFunctionData {
	if resp == nil || resp.Organization.CloudLinks == nil {
		return nil
	}

	var functions []StoredFunctionData

	for _, edge := range resp.Organization.CloudLinks.Edges {
		cloudLink := edge.Node

		switch cl := cloudLink.(type) {
		case *ListAllStoredFunctionsOrganizationCloudLinksCloudLinkConnectionEdgesCloudLinkEdgeNodeCloudLinkSnowflake:
			for _, sfEdge := range cl.StoredSnowflakeFunctions.Edges {
				sf := sfEdge.Node
				data := StoredFunctionData{
					ID:            sf.Id,
					Name:          sf.Name,
					DisplayName:   sf.DisplayName,
					Description:   sf.Description,
					DatabaseName:  sf.DatabaseName,
					SchemaName:    sf.SchemaName,
					Type:          string(sf.Type),
					Configured:    sf.Configured,
					CloudLinkID:   cl.Id,
					CloudLinkName: cl.Name,
				}

				// Extract parameters
				for _, arg := range sf.InputArgs {
					name := ""
					if arg.Name != nil {
						name = *arg.Name
					}
					data.Parameters = append(data.Parameters, StoredFunctionParameter{
						ID:   arg.Id,
						Name: name,
						Type: string(arg.Type),
					})
				}

				// Extract output
				if sf.OutputArg != nil {
					data.Output = extractOutputFromListAllStoredFunctions(sf.OutputArg)
				}

				functions = append(functions, data)
			}
		}
	}

	return functions
}

// extractOutputFromListAllStoredFunctions extracts output info from ListAllStoredFunctions response.
func extractOutputFromListAllStoredFunctions(outputArg ListAllStoredFunctionsOrganizationCloudLinksCloudLinkConnectionEdgesCloudLinkEdgeNodeCloudLinkSnowflakeStoredSnowflakeFunctionsStoredSnowflakeFunctionConnectionEdgesStoredSnowflakeFunctionEdgeNodeStoredSnowflakeFunctionOutputArg) *StoredFunctionOutput {
	output := &StoredFunctionOutput{}

	switch o := outputArg.(type) {
	case *ListAllStoredFunctionsOrganizationCloudLinksCloudLinkConnectionEdgesCloudLinkEdgeNodeCloudLinkSnowflakeStoredSnowflakeFunctionsStoredSnowflakeFunctionConnectionEdgesStoredSnowflakeFunctionEdgeNodeStoredSnowflakeFunctionOutputArgStoredSnowflakeFunctionSimpleOutput:
		output.OutputType = "simple"
		output.ReturnType = string(o.ReturnType)
		output.Nullable = o.Nullable
	case *ListAllStoredFunctionsOrganizationCloudLinksCloudLinkConnectionEdgesCloudLinkEdgeNodeCloudLinkSnowflakeStoredSnowflakeFunctionsStoredSnowflakeFunctionConnectionEdgesStoredSnowflakeFunctionEdgeNodeStoredSnowflakeFunctionOutputArgStoredSnowflakeFunctionTableOutput:
		output.OutputType = "table"
		for _, col := range o.Columns {
			output.Columns = append(output.Columns, StoredFunctionOutputColumn{
				Name: col.Name,
				Type: string(col.ColumnType),
			})
		}
	}

	return output
}

// AvailableFunctionParameter represents an input parameter of an available Snowflake function.
type AvailableFunctionParameter struct {
	Name string
	Type string
}

// AvailableFunctionOutputColumn represents a column in a table-returning function output.
type AvailableFunctionOutputColumn struct {
	Name string
	Type string
}

// AvailableFunctionOutput represents the return type of an available Snowflake function.
type AvailableFunctionOutput struct {
	OutputType string // "simple" or "table"
	ReturnType string // For simple outputs
	Nullable   bool   // For simple outputs
	Columns    []AvailableFunctionOutputColumn
}

// AvailableFunctionData represents a Snowflake function available for configuration.
type AvailableFunctionData struct {
	Name          string
	DatabaseName  string
	SchemaName    string
	Type          string // "PROCEDURE" or "USER_DEFINED_FUNCTION"
	Description   string
	MinArgs       int
	MaxArgs       int
	BuiltIn       bool
	Secure        bool
	TableFunction bool
	Parameters    []AvailableFunctionParameter
	Output        *AvailableFunctionOutput
}

// ExtractAvailableFunctions extracts available functions from ListAvailableFunctionsResponse.
func ExtractAvailableFunctions(resp *ListAvailableFunctionsResponse) []AvailableFunctionData {
	if resp == nil || resp.Organization.CloudLink == nil {
		return nil
	}

	var functions []AvailableFunctionData

	switch cl := (*resp.Organization.CloudLink).(type) {
	case *ListAvailableFunctionsOrganizationCloudLinkCloudLinkSnowflake:
		for _, fn := range cl.AvailableFunctions {
			data := AvailableFunctionData{
				Name:          fn.Name,
				Type:          string(fn.Type),
				MinArgs:       fn.MinimumNumberOfArguments,
				MaxArgs:       fn.MaximumNumberOfArguments,
				BuiltIn:       fn.BuiltIn,
				Secure:        fn.Secure,
				TableFunction: fn.TableFunction,
			}
			if fn.DatabaseName != nil {
				data.DatabaseName = *fn.DatabaseName
			}
			if fn.SchemaName != nil {
				data.SchemaName = *fn.SchemaName
			}
			if fn.Description != nil {
				data.Description = *fn.Description
			}
			// Extract parameters
			if fn.Arguments != nil && fn.Arguments.InputArgs != nil {
				for _, arg := range fn.Arguments.InputArgs {
					param := AvailableFunctionParameter{
						Type: string(arg.Type),
					}
					if arg.Name != nil {
						param.Name = *arg.Name
					}
					data.Parameters = append(data.Parameters, param)
				}
			}
			// Extract output
			if fn.Arguments != nil {
				data.Output = extractOutputFromSnowflakeAvailableFunction(fn.Arguments.OutputArg)
			}
			functions = append(functions, data)
		}
	case *ListAvailableFunctionsOrganizationCloudLinkCloudLinkElementumSnowflake:
		for _, fn := range cl.AvailableFunctions {
			data := AvailableFunctionData{
				Name:          fn.Name,
				Type:          string(fn.Type),
				MinArgs:       fn.MinimumNumberOfArguments,
				MaxArgs:       fn.MaximumNumberOfArguments,
				BuiltIn:       fn.BuiltIn,
				Secure:        fn.Secure,
				TableFunction: fn.TableFunction,
			}
			if fn.DatabaseName != nil {
				data.DatabaseName = *fn.DatabaseName
			}
			if fn.SchemaName != nil {
				data.SchemaName = *fn.SchemaName
			}
			if fn.Description != nil {
				data.Description = *fn.Description
			}
			// Extract parameters
			if fn.Arguments != nil && fn.Arguments.InputArgs != nil {
				for _, arg := range fn.Arguments.InputArgs {
					param := AvailableFunctionParameter{
						Type: string(arg.Type),
					}
					if arg.Name != nil {
						param.Name = *arg.Name
					}
					data.Parameters = append(data.Parameters, param)
				}
			}
			// Extract output
			if fn.Arguments != nil {
				data.Output = extractOutputFromElementumSnowflakeAvailableFunction(fn.Arguments.OutputArg)
			}
			functions = append(functions, data)
		}
	}

	return functions
}

// extractOutputFromSnowflakeAvailableFunction extracts output info from CloudLinkSnowflake available function.
func extractOutputFromSnowflakeAvailableFunction(outputArg ListAvailableFunctionsOrganizationCloudLinkCloudLinkSnowflakeAvailableFunctionsSnowflakeFunctionArgumentsOutputArgSnowflakeFunctionOutput) *AvailableFunctionOutput {
	output := &AvailableFunctionOutput{}

	switch o := outputArg.(type) {
	case *ListAvailableFunctionsOrganizationCloudLinkCloudLinkSnowflakeAvailableFunctionsSnowflakeFunctionArgumentsOutputArgSnowflakeFunctionSimpleOutput:
		output.OutputType = "simple"
		output.ReturnType = string(o.ReturnType)
		output.Nullable = o.Nullable
	case *ListAvailableFunctionsOrganizationCloudLinkCloudLinkSnowflakeAvailableFunctionsSnowflakeFunctionArgumentsOutputArgSnowflakeFunctionTableOutput:
		output.OutputType = "table"
		for _, col := range o.Columns {
			name := ""
			if col.Name != nil {
				name = *col.Name
			}
			output.Columns = append(output.Columns, AvailableFunctionOutputColumn{
				Name: name,
				Type: string(col.ColumnType),
			})
		}
	}

	return output
}

// extractOutputFromElementumSnowflakeAvailableFunction extracts output info from CloudLinkElementumSnowflake available function.
func extractOutputFromElementumSnowflakeAvailableFunction(outputArg ListAvailableFunctionsOrganizationCloudLinkCloudLinkElementumSnowflakeAvailableFunctionsSnowflakeFunctionArgumentsOutputArgSnowflakeFunctionOutput) *AvailableFunctionOutput {
	output := &AvailableFunctionOutput{}

	switch o := outputArg.(type) {
	case *ListAvailableFunctionsOrganizationCloudLinkCloudLinkElementumSnowflakeAvailableFunctionsSnowflakeFunctionArgumentsOutputArgSnowflakeFunctionSimpleOutput:
		output.OutputType = "simple"
		output.ReturnType = string(o.ReturnType)
		output.Nullable = o.Nullable
	case *ListAvailableFunctionsOrganizationCloudLinkCloudLinkElementumSnowflakeAvailableFunctionsSnowflakeFunctionArgumentsOutputArgSnowflakeFunctionTableOutput:
		output.OutputType = "table"
		for _, col := range o.Columns {
			name := ""
			if col.Name != nil {
				name = *col.Name
			}
			output.Columns = append(output.Columns, AvailableFunctionOutputColumn{
				Name: name,
				Type: string(col.ColumnType),
			})
		}
	}

	return output
}

// CloudLinkStoredFunction represents a configured (stored) Snowflake function on a CloudLink.
type CloudLinkStoredFunction struct {
	ID          string
	Name        string
	DisplayName string
}

// CloudLinkFunctionsResult holds both stored and available functions for a CloudLink.
type CloudLinkFunctionsResult struct {
	StoredFunctions    []CloudLinkStoredFunction
	AvailableFunctions []AvailableFunctionData
}

// snowflakeFunc is satisfied by all genqlient-generated SnowflakeFunction structs,
// letting us extract AvailableFunctionData without duplicating code per query variant.
type snowflakeFunc interface {
	GetName() string
	GetType() SnowflakeFunctionType
	GetDatabaseName() *string
	GetSchemaName() *string
	GetDescription() *string
	GetMinimumNumberOfArguments() int
	GetMaximumNumberOfArguments() int
	GetBuiltIn() bool
	GetSecure() bool
	GetTableFunction() bool
}

// snowflakeInputArg is satisfied by genqlient-generated input arg types.
type snowflakeInputArg interface {
	GetName() *string
	GetType() DatasourceColumnType
}

// snowflakeOutputColumn is satisfied by genqlient-generated column arg types.
type snowflakeOutputColumn interface {
	GetName() *string
	GetColumnType() DatasourceColumnType
}

// convertSnowflakeFunc converts any genqlient SnowflakeFunction struct to AvailableFunctionData.
func convertSnowflakeFunc(fn snowflakeFunc, params []snowflakeInputArg, output *AvailableFunctionOutput) AvailableFunctionData {
	data := AvailableFunctionData{
		Name:          fn.GetName(),
		Type:          string(fn.GetType()),
		MinArgs:       fn.GetMinimumNumberOfArguments(),
		MaxArgs:       fn.GetMaximumNumberOfArguments(),
		BuiltIn:       fn.GetBuiltIn(),
		Secure:        fn.GetSecure(),
		TableFunction: fn.GetTableFunction(),
	}
	if v := fn.GetDatabaseName(); v != nil {
		data.DatabaseName = *v
	}
	if v := fn.GetSchemaName(); v != nil {
		data.SchemaName = *v
	}
	if v := fn.GetDescription(); v != nil {
		data.Description = *v
	}
	for _, arg := range params {
		param := AvailableFunctionParameter{Type: string(arg.GetType())}
		if v := arg.GetName(); v != nil {
			param.Name = *v
		}
		data.Parameters = append(data.Parameters, param)
	}
	data.Output = output
	return data
}

// extractSimpleTableOutput builds AvailableFunctionOutput from simple/table output variants.
func extractSimpleTableOutput(outputType string, returnType DatasourceColumnType, nullable bool, columns []snowflakeOutputColumn) *AvailableFunctionOutput {
	output := &AvailableFunctionOutput{}
	switch outputType {
	case "simple":
		output.OutputType = "simple"
		output.ReturnType = string(returnType)
		output.Nullable = nullable
	case "table":
		output.OutputType = "table"
		for _, col := range columns {
			name := ""
			if v := col.GetName(); v != nil {
				name = *v
			}
			output.Columns = append(output.Columns, AvailableFunctionOutputColumn{
				Name: name,
				Type: string(col.GetColumnType()),
			})
		}
	}
	return output
}

// ExtractCloudLinkFunctions extracts both stored and available functions from ListCloudLinkFunctionsResponse.
func ExtractCloudLinkFunctions(resp *ListCloudLinkFunctionsResponse) CloudLinkFunctionsResult {
	var result CloudLinkFunctionsResult
	if resp == nil || resp.Organization.CloudLink == nil {
		return result
	}

	switch cl := (*resp.Organization.CloudLink).(type) {
	case *ListCloudLinkFunctionsOrganizationCloudLinkCloudLinkSnowflake:
		for _, edge := range cl.StoredSnowflakeFunctions.Edges {
			result.StoredFunctions = append(result.StoredFunctions, CloudLinkStoredFunction{
				ID:          edge.Node.Id,
				Name:        edge.Node.Name,
				DisplayName: edge.Node.DisplayName,
			})
		}
		for _, fn := range cl.AvailableFunctions {
			result.AvailableFunctions = append(result.AvailableFunctions, extractCloudLinkSnowflakeFunc(&fn))
		}
	case *ListCloudLinkFunctionsOrganizationCloudLinkCloudLinkElementumSnowflake:
		for _, fn := range cl.AvailableFunctions {
			result.AvailableFunctions = append(result.AvailableFunctions, extractCloudLinkElementumSnowflakeFunc(&fn))
		}
	}

	return result
}

// extractCloudLinkSnowflakeFunc converts a CloudLinkSnowflake available function using the shared helper.
func extractCloudLinkSnowflakeFunc(fn *ListCloudLinkFunctionsOrganizationCloudLinkCloudLinkSnowflakeAvailableFunctionsSnowflakeFunction) AvailableFunctionData {
	var params []snowflakeInputArg
	var output *AvailableFunctionOutput
	if fn.Arguments != nil {
		for i := range fn.Arguments.InputArgs {
			params = append(params, &fn.Arguments.InputArgs[i])
		}
		output = extractOutputFromCloudLinkSnowflake(fn.Arguments.OutputArg)
	}
	return convertSnowflakeFunc(fn, params, output)
}

// extractCloudLinkElementumSnowflakeFunc converts a CloudLinkElementumSnowflake available function using the shared helper.
func extractCloudLinkElementumSnowflakeFunc(fn *ListCloudLinkFunctionsOrganizationCloudLinkCloudLinkElementumSnowflakeAvailableFunctionsSnowflakeFunction) AvailableFunctionData {
	var params []snowflakeInputArg
	var output *AvailableFunctionOutput
	if fn.Arguments != nil {
		for i := range fn.Arguments.InputArgs {
			params = append(params, &fn.Arguments.InputArgs[i])
		}
		output = extractOutputFromCloudLinkElementumSnowflake(fn.Arguments.OutputArg)
	}
	return convertSnowflakeFunc(fn, params, output)
}

func extractOutputFromCloudLinkSnowflake(outputArg ListCloudLinkFunctionsOrganizationCloudLinkCloudLinkSnowflakeAvailableFunctionsSnowflakeFunctionArgumentsOutputArgSnowflakeFunctionOutput) *AvailableFunctionOutput {
	switch o := outputArg.(type) {
	case *ListCloudLinkFunctionsOrganizationCloudLinkCloudLinkSnowflakeAvailableFunctionsSnowflakeFunctionArgumentsOutputArgSnowflakeFunctionSimpleOutput:
		return extractSimpleTableOutput("simple", o.ReturnType, o.Nullable, nil)
	case *ListCloudLinkFunctionsOrganizationCloudLinkCloudLinkSnowflakeAvailableFunctionsSnowflakeFunctionArgumentsOutputArgSnowflakeFunctionTableOutput:
		cols := make([]snowflakeOutputColumn, len(o.Columns))
		for i := range o.Columns {
			cols[i] = &o.Columns[i]
		}
		return extractSimpleTableOutput("table", "", false, cols)
	}
	return &AvailableFunctionOutput{}
}

func extractOutputFromCloudLinkElementumSnowflake(outputArg ListCloudLinkFunctionsOrganizationCloudLinkCloudLinkElementumSnowflakeAvailableFunctionsSnowflakeFunctionArgumentsOutputArgSnowflakeFunctionOutput) *AvailableFunctionOutput {
	switch o := outputArg.(type) {
	case *ListCloudLinkFunctionsOrganizationCloudLinkCloudLinkElementumSnowflakeAvailableFunctionsSnowflakeFunctionArgumentsOutputArgSnowflakeFunctionSimpleOutput:
		return extractSimpleTableOutput("simple", o.ReturnType, o.Nullable, nil)
	case *ListCloudLinkFunctionsOrganizationCloudLinkCloudLinkElementumSnowflakeAvailableFunctionsSnowflakeFunctionArgumentsOutputArgSnowflakeFunctionTableOutput:
		cols := make([]snowflakeOutputColumn, len(o.Columns))
		for i := range o.Columns {
			cols[i] = &o.Columns[i]
		}
		return extractSimpleTableOutput("table", "", false, cols)
	}
	return &AvailableFunctionOutput{}
}

// ObjectDetailsData represents detailed information about an aspect (App, Element, or Task).
type ObjectDetailsData struct {
	ID            string
	Name          string
	Type          string // "App", "Element", "Task"
	Namespace     string
	Handle        string
	Description   string
	Icon          string
	Color         string
	SourceType    string
	StorageType   string
	CategoryID    string
	CategoryName  string
	CloudLinkID   string
	CloudLinkName string
	CloudLinkType string
	// Snowflake connection details
	DatabaseName string
	SchemaName   string
	TableName    string
	ReadOnly     bool
}

// ExtractAspectDetails extracts detailed object information from GetAspectDetailsResponse.
func ExtractAspectDetails(resp *GetAspectDetailsResponse) *ObjectDetailsData {
	if resp == nil {
		return nil
	}

	edges := resp.Organization.Aspects.Edges
	if len(edges) == 0 {
		return nil
	}

	node := edges[0].Node
	if node == nil {
		return nil
	}

	details := &ObjectDetailsData{
		ID:   node.GetId(),
		Name: node.GetName(),
	}

	// Extract type name
	if typename := node.GetTypename(); typename != nil {
		details.Type = *typename
	}

	// Extract type-specific fields based on concrete type
	switch concrete := node.(type) {
	case *GetAspectDetailsOrganizationAspectsAspectConnectionEdgesAspectEdgeNodeAspectApp:
		details.Type = "App"
		details.Namespace = concrete.Namespace
		details.Handle = concrete.Handle
		if concrete.Description != nil {
			details.Description = *concrete.Description
		}
		if concrete.Icon != nil {
			details.Icon = *concrete.Icon
		}
		if concrete.Color != nil {
			details.Color = *concrete.Color
		}
		details.SourceType = string(concrete.SourceType)
		if concrete.StorageType != nil {
			details.StorageType = string(*concrete.StorageType)
		}
		details.CategoryID = concrete.Category.Id
		details.CategoryName = concrete.Category.Name
		extractCloudLinkDetailsApp(concrete, details)
		extractCloudConnectionDetailsApp(concrete, details)

	case *GetAspectDetailsOrganizationAspectsAspectConnectionEdgesAspectEdgeNodeAspectElement:
		details.Type = "Element"
		details.Namespace = concrete.Namespace
		details.Handle = concrete.Handle
		if concrete.Description != nil {
			details.Description = *concrete.Description
		}
		if concrete.Icon != nil {
			details.Icon = *concrete.Icon
		}
		if concrete.Color != nil {
			details.Color = *concrete.Color
		}
		details.SourceType = string(concrete.SourceType)
		if concrete.StorageType != nil {
			details.StorageType = string(*concrete.StorageType)
		}
		details.CategoryID = concrete.Category.Id
		details.CategoryName = concrete.Category.Name
		extractCloudLinkDetailsElement(concrete, details)
		extractCloudConnectionDetailsElement(concrete, details)

	case *GetAspectDetailsOrganizationAspectsAspectConnectionEdgesAspectEdgeNodeAspectTask:
		details.Type = "Task"
		details.Namespace = concrete.Namespace
		details.Handle = concrete.Handle
		if concrete.Description != nil {
			details.Description = *concrete.Description
		}
		if concrete.Icon != nil {
			details.Icon = *concrete.Icon
		}
		if concrete.Color != nil {
			details.Color = *concrete.Color
		}
		details.SourceType = string(concrete.SourceType)
		if concrete.StorageType != nil {
			details.StorageType = string(*concrete.StorageType)
		}
		details.CategoryID = concrete.Category.Id
		details.CategoryName = concrete.Category.Name
		extractCloudLinkDetailsTask(concrete, details)
		extractCloudConnectionDetailsTask(concrete, details)
	}

	return details
}

func extractCloudLinkDetailsApp(app *GetAspectDetailsOrganizationAspectsAspectConnectionEdgesAspectEdgeNodeAspectApp, details *ObjectDetailsData) {
	if app.CloudLink == nil {
		return
	}
	cl := *app.CloudLink
	details.CloudLinkID = cl.GetId()
	details.CloudLinkName = cl.GetName()
	if typename := cl.GetTypename(); typename != nil {
		details.CloudLinkType = *typename
	}
}

func extractCloudConnectionDetailsApp(app *GetAspectDetailsOrganizationAspectsAspectConnectionEdgesAspectEdgeNodeAspectApp, details *ObjectDetailsData) {
	if app.CloudConnection == nil {
		return
	}
	cc := *app.CloudConnection
	details.ReadOnly = cc.GetReadOnly()
	if sf, ok := cc.(*GetAspectDetailsOrganizationAspectsAspectConnectionEdgesAspectEdgeNodeAspectAppCloudConnectionSnowflakeConnection); ok {
		if sf.DatabaseName != nil {
			details.DatabaseName = *sf.DatabaseName
		}
		if sf.SchemaName != nil {
			details.SchemaName = *sf.SchemaName
		}
		if sf.TableName != nil {
			details.TableName = *sf.TableName
		}
	}
}

func extractCloudLinkDetailsElement(elem *GetAspectDetailsOrganizationAspectsAspectConnectionEdgesAspectEdgeNodeAspectElement, details *ObjectDetailsData) {
	if elem.CloudLink == nil {
		return
	}
	cl := *elem.CloudLink
	details.CloudLinkID = cl.GetId()
	details.CloudLinkName = cl.GetName()
	if typename := cl.GetTypename(); typename != nil {
		details.CloudLinkType = *typename
	}
}

func extractCloudConnectionDetailsElement(elem *GetAspectDetailsOrganizationAspectsAspectConnectionEdgesAspectEdgeNodeAspectElement, details *ObjectDetailsData) {
	if elem.CloudConnection == nil {
		return
	}
	cc := *elem.CloudConnection
	details.ReadOnly = cc.GetReadOnly()
	if sf, ok := cc.(*GetAspectDetailsOrganizationAspectsAspectConnectionEdgesAspectEdgeNodeAspectElementCloudConnectionSnowflakeConnection); ok {
		if sf.DatabaseName != nil {
			details.DatabaseName = *sf.DatabaseName
		}
		if sf.SchemaName != nil {
			details.SchemaName = *sf.SchemaName
		}
		if sf.TableName != nil {
			details.TableName = *sf.TableName
		}
	}
}

func extractCloudLinkDetailsTask(task *GetAspectDetailsOrganizationAspectsAspectConnectionEdgesAspectEdgeNodeAspectTask, details *ObjectDetailsData) {
	if task.CloudLink == nil {
		return
	}
	cl := *task.CloudLink
	details.CloudLinkID = cl.GetId()
	details.CloudLinkName = cl.GetName()
	if typename := cl.GetTypename(); typename != nil {
		details.CloudLinkType = *typename
	}
}

func extractCloudConnectionDetailsTask(task *GetAspectDetailsOrganizationAspectsAspectConnectionEdgesAspectEdgeNodeAspectTask, details *ObjectDetailsData) {
	if task.CloudConnection == nil {
		return
	}
	cc := *task.CloudConnection
	details.ReadOnly = cc.GetReadOnly()
	if sf, ok := cc.(*GetAspectDetailsOrganizationAspectsAspectConnectionEdgesAspectEdgeNodeAspectTaskCloudConnectionSnowflakeConnection); ok {
		if sf.DatabaseName != nil {
			details.DatabaseName = *sf.DatabaseName
		}
		if sf.SchemaName != nil {
			details.SchemaName = *sf.SchemaName
		}
		if sf.TableName != nil {
			details.TableName = *sf.TableName
		}
	}
}
