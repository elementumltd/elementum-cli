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

package discovery

import (
	"context"
	"sort"

	"github.com/elementumltd/elementum-cli/internal/client"
	"github.com/elementumltd/elementum-cli/logger"
)

// SearchTablesResult contains both regular and linked AI search tables
type SearchTablesResult struct {
	AISearchTables       []AISearchTable
	LinkedAISearchTables []LinkedAISearchTable
}

// GetAISearchTablesForAspect retrieves all AI search tables for an aspect
func GetAISearchTablesForAspect(ctx context.Context, c *client.Client, aspectID string) ([]AISearchTable, error) {
	result, err := GetAllSearchTablesForAspect(ctx, c, aspectID)
	if err != nil {
		return nil, err
	}
	return result.AISearchTables, nil
}

// GetAllSearchTablesForAspect retrieves all AI search tables (both created and linked) for an aspect
func GetAllSearchTablesForAspect(ctx context.Context, c *client.Client, aspectID string) (*SearchTablesResult, error) {
	result, err := client.GetAspectSearchTables(ctx, c.Genqlient(), aspectID)
	if err != nil {
		return nil, err
	}

	if result.Organization.Aspect == nil {
		return &SearchTablesResult{}, nil
	}

	searchResult := &SearchTablesResult{}

	// Extract search tables from aspect (works for both AspectApp and AspectElement)
	switch aspect := (*result.Organization.Aspect).(type) {
	case *client.GetAspectSearchTablesOrganizationAspectAspectApp:
		if aspect.SearchTables != nil {
			for _, edge := range aspect.SearchTables.Edges {
				node := edge.GetNode()
				extractSearchTableNode(node, aspectID, searchResult)
			}
		}
	case *client.GetAspectSearchTablesOrganizationAspectAspectElement:
		if aspect.SearchTables != nil {
			for _, edge := range aspect.SearchTables.Edges {
				node := edge.GetNode()
				extractSearchTableNodeElement(node, aspectID, searchResult)
			}
		}
	}

	return searchResult, nil
}

// extractSearchTableNode extracts a search table from App aspect response
func extractSearchTableNode(node client.GetAspectSearchTablesOrganizationAspectAspectAppSearchTablesAspectSearchTableConnectionEdgesAspectSearchTableEdgeNodeAspectSearchTable, aspectID string, result *SearchTablesResult) {
	// Check the concrete type
	switch t := node.(type) {
	case *client.GetAspectSearchTablesOrganizationAspectAspectAppSearchTablesAspectSearchTableConnectionEdgesAspectSearchTableEdgeNodeAspectSearchSnowflakeTable:
		// Regular AI search table (creates a Cortex service)
		st := AISearchTable{
			ID:       t.GetId(),
			ObjectID: aspectID,
		}

		// Duration and TargetLag are nullable
		if d := t.GetDuration(); d != nil {
			st.Duration = string(*d)
		}
		if lag := t.GetTargetLag(); lag != nil {
			st.TargetLag = *lag
		}

		// Field - pointer-to-interface
		if fieldPtr := t.GetField(); fieldPtr != nil {
			field := *fieldPtr
			st.FieldID = field.GetId()
			st.FieldName = field.GetName()
		}

		// AI Provider Connector
		if connector := t.GetAiProviderConnector(); connector != nil {
			st.AIProviderConnectorID = connector.GetId()
			st.AIProviderConnectorName = connector.GetModel().Name
		}

		// Attribute Fields
		attrFields := t.GetAttributeFields()
		st.AttributeFieldIDs = make([]string, 0, len(attrFields))
		for _, af := range attrFields {
			st.AttributeFieldIDs = append(st.AttributeFieldIDs, af.GetId())
		}

		// Warehouse (optional)
		if warehouse := t.GetWarehouse(); warehouse != nil {
			st.Warehouse = *warehouse
		}

		result.AISearchTables = append(result.AISearchTables, st)

	case *client.GetAspectSearchTablesOrganizationAspectAspectAppSearchTablesAspectSearchTableConnectionEdgesAspectSearchTableEdgeNodeAspectLinkedSearchTable:
		// Linked AI search table (connects to existing Cortex service)
		lst := LinkedAISearchTable{
			ID:       t.GetId(),
			ObjectID: aspectID,
			Status:   string(t.GetStatus()),
		}

		// Warehouse (optional)
		if warehouse := t.GetWarehouse(); warehouse != nil {
			lst.Warehouse = *warehouse
		}

		// Note: CloudLinkID, Database, SchemaName, ServiceName, and DisplayName
		// are not available in this query. They would need to be fetched separately
		// via a more detailed query if needed for export functionality.

		result.LinkedAISearchTables = append(result.LinkedAISearchTables, lst)
	}
}

// extractSearchTableNodeElement extracts a search table from Element aspect response
func extractSearchTableNodeElement(node client.GetAspectSearchTablesOrganizationAspectAspectElementSearchTablesAspectSearchTableConnectionEdgesAspectSearchTableEdgeNodeAspectSearchTable, aspectID string, result *SearchTablesResult) {
	// Check the concrete type
	switch t := node.(type) {
	case *client.GetAspectSearchTablesOrganizationAspectAspectElementSearchTablesAspectSearchTableConnectionEdgesAspectSearchTableEdgeNodeAspectSearchSnowflakeTable:
		// Regular AI search table (creates a Cortex service)
		st := AISearchTable{
			ID:       t.GetId(),
			ObjectID: aspectID,
		}

		// Duration and TargetLag are nullable
		if d := t.GetDuration(); d != nil {
			st.Duration = string(*d)
		}
		if lag := t.GetTargetLag(); lag != nil {
			st.TargetLag = *lag
		}

		// Field - pointer-to-interface
		if fieldPtr := t.GetField(); fieldPtr != nil {
			field := *fieldPtr
			st.FieldID = field.GetId()
			st.FieldName = field.GetName()
		}

		// AI Provider Connector
		if connector := t.GetAiProviderConnector(); connector != nil {
			st.AIProviderConnectorID = connector.GetId()
			st.AIProviderConnectorName = connector.GetModel().Name
		}

		// Attribute Fields
		attrFields := t.GetAttributeFields()
		st.AttributeFieldIDs = make([]string, 0, len(attrFields))
		for _, af := range attrFields {
			st.AttributeFieldIDs = append(st.AttributeFieldIDs, af.GetId())
		}

		// Warehouse (optional)
		if warehouse := t.GetWarehouse(); warehouse != nil {
			st.Warehouse = *warehouse
		}

		result.AISearchTables = append(result.AISearchTables, st)

	case *client.GetAspectSearchTablesOrganizationAspectAspectElementSearchTablesAspectSearchTableConnectionEdgesAspectSearchTableEdgeNodeAspectLinkedSearchTable:
		// Linked AI search table (connects to existing Cortex service)
		lst := LinkedAISearchTable{
			ID:       t.GetId(),
			ObjectID: aspectID,
			Status:   string(t.GetStatus()),
		}

		// Warehouse (optional)
		if warehouse := t.GetWarehouse(); warehouse != nil {
			lst.Warehouse = *warehouse
		}

		// Note: CloudLinkID, Database, SchemaName, ServiceName, and DisplayName
		// are not available in this query.

		result.LinkedAISearchTables = append(result.LinkedAISearchTables, lst)
	}
}

// GetSearchTablesForTable retrieves all search tables for a table
func GetSearchTablesForTable(ctx context.Context, c *client.Client, tableID string) ([]TableSearchTable, error) {
	result, err := client.GetTableSearchTables(ctx, c.Genqlient(), tableID)
	if err != nil {
		return nil, err
	}

	if result.Organization.Table.SearchTables == nil {
		return []TableSearchTable{}, nil
	}

	var searchTables []TableSearchTable

	for _, edge := range result.Organization.Table.SearchTables.Edges {
		node := edge.Node
		st := TableSearchTable{
			ID:        node.GetId(),
			TableID:   tableID,
			Duration:  string(node.GetDuration()),
			TargetLag: node.GetTargetLag(),
		}

		// Field
		field := node.GetField()
		st.FieldID = field.GetId()
		st.FieldName = field.GetName()

		// AI Provider Connector
		connector := node.GetAiProviderConnector()
		st.AIProviderConnectorID = connector.GetId()

		// Attribute Fields
		attrFields := node.GetAttributeFields()
		st.AttributeFieldIDs = make([]string, 0, len(attrFields))
		for _, af := range attrFields {
			st.AttributeFieldIDs = append(st.AttributeFieldIDs, af.GetId())
		}

		// Warehouse (optional)
		if warehouse := node.GetWarehouse(); warehouse != nil {
			st.Warehouse = *warehouse
		}

		searchTables = append(searchTables, st)
	}

	return searchTables, nil
}

// GetManagedViewOrderForAspect retrieves the managed view order for an aspect
// by reconstructing it from the views' displayOrder attribute
func GetManagedViewOrderForAspect(ctx context.Context, c *client.Client, aspectID string) (*ManagedViewOrder, error) {
	result, err := client.GetAspectViews(ctx, c.Genqlient(), aspectID)
	if err != nil {
		return nil, err
	}

	if result.Organization.Aspect == nil {
		return nil, nil
	}

	aspect := *result.Organization.Aspect
	views := aspect.GetViews()
	if len(views.Edges) == 0 {
		return nil, nil
	}

	// Collect views with their display order
	type viewOrder struct {
		ID           string
		DisplayOrder int
	}
	var viewList []viewOrder

	for _, edge := range views.Edges {
		node := edge.Node
		if node == nil {
			continue
		}
		displayOrder := 0
		if node.GetDisplayOrder() != nil {
			displayOrder = *node.GetDisplayOrder()
		}
		viewList = append(viewList, viewOrder{
			ID:           node.GetId(),
			DisplayOrder: displayOrder,
		})
	}

	if len(viewList) == 0 {
		return nil, nil
	}

	// Sort by display order
	sort.Slice(viewList, func(i, j int) bool {
		return viewList[i].DisplayOrder < viewList[j].DisplayOrder
	})

	// Extract ordered view IDs
	viewIDs := make([]string, len(viewList))
	for i, v := range viewList {
		viewIDs[i] = v.ID
	}

	return &ManagedViewOrder{
		ObjectID: aspectID,
		ViewIDs:  viewIDs,
	}, nil
}

// GetDashboardsForAspect retrieves all dashboards and their widgets for an aspect
func GetDashboardsForAspect(ctx context.Context, c *client.Client, aspectID string) ([]Dashboard, error) {
	result, err := client.GetDashboardsWithWidgetsByAspect(ctx, c.Genqlient(), aspectID)
	if err != nil {
		return nil, err
	}

	if result.Organization.Aspect == nil {
		return []Dashboard{}, nil
	}

	var dashboards []Dashboard

	// Extract dashboards from aspect (works for AspectApp, AspectElement, AspectTask)
	switch aspect := (*result.Organization.Aspect).(type) {
	case *client.GetDashboardsWithWidgetsByAspectOrganizationAspectAspectApp:
		if aspect.Dashboards != nil {
			for _, edge := range aspect.Dashboards.Edges {
				d := convertDashboard(edge.Node.DashboardInfo, "APP_ASPECT")
				// Convert widgets
				for _, widgetEdge := range edge.Node.Widgets.Edges {
					widget := convertWidgetInfo(widgetEdge.Node, d.ID)
					if widget != nil {
						d.Widgets = append(d.Widgets, *widget)
					}
				}
				dashboards = append(dashboards, d)
			}
		}
	case *client.GetDashboardsWithWidgetsByAspectOrganizationAspectAspectElement:
		if aspect.Dashboards != nil {
			for _, edge := range aspect.Dashboards.Edges {
				d := convertDashboard(edge.Node.DashboardInfo, "ELEMENT_ASPECT")
				// Convert widgets
				for _, widgetEdge := range edge.Node.Widgets.Edges {
					widget := convertWidgetInfo(widgetEdge.Node, d.ID)
					if widget != nil {
						d.Widgets = append(d.Widgets, *widget)
					}
				}
				dashboards = append(dashboards, d)
			}
		}
	case *client.GetDashboardsWithWidgetsByAspectOrganizationAspectAspectTask:
		if aspect.Dashboards != nil {
			for _, edge := range aspect.Dashboards.Edges {
				d := convertDashboard(edge.Node.DashboardInfo, "TASK_ASPECT")
				// Convert widgets
				for _, widgetEdge := range edge.Node.Widgets.Edges {
					widget := convertWidgetInfo(widgetEdge.Node, d.ID)
					if widget != nil {
						d.Widgets = append(d.Widgets, *widget)
					}
				}
				dashboards = append(dashboards, d)
			}
		}
	}

	return dashboards, nil
}

// convertDashboard converts DashboardInfo to Dashboard
func convertDashboard(info client.DashboardInfo, ownerType string) Dashboard {
	d := Dashboard{
		ID:        info.Id,
		Name:      info.Name,
		OwnerType: ownerType,
		Tags:      info.Tags,
	}
	if info.OwnerId != nil {
		d.OwnerID = *info.OwnerId
	}
	if info.Favorite != nil {
		d.Favorite = *info.Favorite
	}
	return d
}

// discoverDashboardWidgetAspects discovers external aspects referenced by widgets
// (both dashboard widgets and regular display widgets) and adds them to RelatedObjects for beautification.
// This includes widgets on the main app AND on all discovered apps/elements/tasks.
func discoverDashboardWidgetAspects(ctx context.Context, c *client.Client, app *App) error {
	// Collect unique aspect IDs from widgets that are not the main app
	seenAspects := make(map[string]bool)
	seenAspects[app.ID] = true // Mark main app as seen

	// Also mark any already-discovered related objects as seen
	for _, obj := range app.RelatedObjects {
		seenAspects[obj.ID] = true
	}

	// Mark discovered apps/elements/tasks as seen (they're exported as resources, not data sources)
	for _, da := range app.DiscoveredApps {
		seenAspects[da.ID] = true
	}
	for _, de := range app.DiscoveredElements {
		seenAspects[de.ID] = true
	}
	for _, dt := range app.DiscoveredTasks {
		seenAspects[dt.ID] = true
	}

	// Helper to discover an aspect and add to RelatedObjects
	discoverAspect := func(aspectID, source string) {
		if aspectID == "" || seenAspects[aspectID] {
			return
		}
		seenAspects[aspectID] = true

		// Discover this aspect using getRelatedObject (objectType and objectName are ignored)
		relatedObj, err := getRelatedObject(ctx, c, aspectID, "", "")
		if err != nil {
			logger.Warn("failed to get widget aspect info", "aspectID", aspectID, "source", source, "error", err)
			return
		}
		if relatedObj != nil {
			app.RelatedObjects = append(app.RelatedObjects, *relatedObj)
			logger.Debug("discovered widget aspect", "aspectID", aspectID, "name", relatedObj.Name, "source", source)
		}
	}

	// Process main app's dashboard widgets
	for _, dashboard := range app.Dashboards {
		for _, widget := range dashboard.Widgets {
			discoverAspect(widget.AspectID, "dashboard widget")
		}
	}

	// Process main app's regular display widgets
	for _, widget := range app.Widgets {
		discoverAspect(widget.AspectID, "display widget")
	}

	// Process discovered apps' widgets
	for _, da := range app.DiscoveredApps {
		for _, dashboard := range da.Dashboards {
			for _, widget := range dashboard.Widgets {
				discoverAspect(widget.AspectID, "discovered app dashboard widget")
			}
		}
		for _, widget := range da.Widgets {
			discoverAspect(widget.AspectID, "discovered app display widget")
		}
	}

	// Process discovered elements' widgets
	for _, de := range app.DiscoveredElements {
		for _, widget := range de.Widgets {
			discoverAspect(widget.AspectID, "discovered element display widget")
		}
	}

	// Process discovered tasks' widgets
	for _, dt := range app.DiscoveredTasks {
		for _, widget := range dt.Widgets {
			discoverAspect(widget.AspectID, "discovered task display widget")
		}
	}

	return nil
}

// DiscoverWidgetAspects is an exported wrapper around discoverDashboardWidgetAspects.
// Call this after recursive discovery is complete to discover widget aspects from all discovered resources.
func DiscoverWidgetAspects(ctx context.Context, c *client.Client, app *App) error {
	return discoverDashboardWidgetAspects(ctx, c, app)
}

// convertWidgetInfo converts a widget interface to DashboardWidget
// Only supports AspectWidget; chart widgets are not supported
func convertWidgetInfo(widget client.WidgetInfo, dashboardID string) *DashboardWidget {
	dw := &DashboardWidget{
		ID:           widget.GetId(),
		DashboardID:  dashboardID,
		Name:         widget.GetName(),
		Size:         string(widget.GetSize()),
		DisplayOrder: widget.GetDisplayOrder(),
	}
	if widget.GetColor() != nil {
		dw.Color = *widget.GetColor()
	}
	if widget.GetIcon() != nil {
		dw.Icon = *widget.GetIcon()
	}

	// Check if it's an aspect widget (the only type we support)
	switch w := widget.(type) {
	case *client.WidgetInfoDashboardAspectWidget:
		if w.Aspect != nil {
			aspect := *w.Aspect
			dw.AspectID = aspect.GetId()
		}
		if w.Filter != nil {
			dw.Filter = string(*w.Filter)
		}
		if w.Sort != nil {
			dw.Sort = string(*w.Sort)
		}
		return dw
	default:
		// Chart widgets and other types are not supported
		logger.Debug("skipping unsupported widget type", "widgetID", widget.GetId())
		return nil
	}
}
