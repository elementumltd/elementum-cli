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
	"fmt"
	"strings"

	"github.com/elementumltd/elementum-cli/logger"
	"github.com/elementumltd/elementum-cli/internal/client"
)

// GetElementByNamespace retrieves an element by its namespace
func GetElementByNamespace(ctx context.Context, c *client.Client, namespace string) (*Element, error) {
	// Use genqlient SearchAspects with filter for elements
	filter := client.BuildEqualsFilter("type", "ELEMENT")
	result, err := client.SearchAspects(ctx, c.Genqlient(), filter)
	if err != nil {
		return nil, fmt.Errorf("failed to search for element: %w", err)
	}

	// Extract aspects using helper
	aspects := client.ExtractAspects(result)

	// Find element with matching namespace
	for _, aspect := range aspects {
		if aspect.Typename == "AspectElement" && aspect.Namespace == namespace {
			// Found it, now get full details
			return GetElement(ctx, c, aspect.ID)
		}
	}

	return nil, fmt.Errorf("element not found with namespace: %s", namespace)
}

// GetElement retrieves detailed information about an element using genqlient.
func GetElement(ctx context.Context, c *client.Client, elementID string) (*Element, error) {
	// Use genqlient GetElement function
	result, err := client.GetElement(ctx, c.Genqlient(), elementID)
	if err != nil {
		return nil, fmt.Errorf("failed to get element: %w", err)
	}

	if result.Organization.Aspect == nil {
		return nil, fmt.Errorf("element not found: %s", elementID)
	}

	// Type assert to get the AspectElement type
	aspectElement, ok := (*result.Organization.Aspect).(*client.GetElementOrganizationAspectAspectElement)
	if !ok {
		return nil, fmt.Errorf("aspect %s is not an element", elementID)
	}

	element := &Element{
		ID:        aspectElement.Id,
		Name:      aspectElement.Name,
		Handle:    aspectElement.Handle,
		Namespace: aspectElement.Namespace,
	}

	// Handle optional fields
	if aspectElement.Description != nil {
		element.Description = *aspectElement.Description
	}
	if aspectElement.Icon != nil {
		element.Icon = *aspectElement.Icon
	}
	if aspectElement.Color != nil {
		element.Color = *aspectElement.Color
	}

	// Set category info (direct field access since getters are on pointer receivers)
	element.CategoryID = aspectElement.Category.Id
	element.CategoryName = aspectElement.Category.Name

	// Set cloudlink info if present (CloudLink is a pointer to interface)
	if aspectElement.CloudLink != nil {
		cloudLink := *aspectElement.CloudLink
		element.CloudLinkID = cloudLink.GetId()
		element.CloudLinkName = cloudLink.GetName()
	}

	// Set cloud connection details if present
	if cloudConn := aspectElement.GetCloudConnection(); cloudConn != nil {
		switch conn := (*cloudConn).(type) {
		case *client.GetElementOrganizationAspectAspectElementCloudConnectionSnowflakeConnection:
			if conn.DatabaseName != nil {
				element.SnowflakeDatabaseName = *conn.DatabaseName
			}
			if conn.SchemaName != nil {
				element.SnowflakeSchemaName = *conn.SchemaName
			}
			if conn.TableName != nil {
				element.SnowflakeTableName = *conn.TableName
			}
		}
	}

	return element, nil
}

// GetElementFull retrieves an element with all its resources (fields, layouts, automations, relationships, etc.)
// This is used by the unified discovery to recursively discover dependencies
func GetElementFull(ctx context.Context, c *client.Client, elementID string) (*Element, error) {
	// Start with basic element info
	element, err := GetElement(ctx, c, elementID)
	if err != nil {
		return nil, err
	}

	// Get fields for this element
	if err := getElementFields(ctx, c, element); err != nil {
		logger.Warn("failed to get element fields (optional)", "elementID", elementID, "error", err)
	}

	// Get layouts for this element
	if err := getElementLayouts(ctx, c, element); err != nil {
		logger.Warn("failed to get element layouts (optional)", "elementID", elementID, "error", err)
	}

	// Get widgets for this element
	if err := getElementWidgets(ctx, c, element); err != nil {
		logger.Warn("failed to get element widgets (optional)", "elementID", elementID, "error", err)
	}

	// Get views for this element
	views, err := getAspectViews(ctx, c, elementID)
	if err != nil {
		logger.Warn("failed to get element views (optional)", "elementID", elementID, "error", err)
	} else {
		element.Views = views
	}
	logger.Debug("checked for views", "elementID", elementID, "viewCount", len(element.Views))

	// Get automations for this element using the generic function
	automations, err := GetAspectAutomations(ctx, c, elementID)
	if err != nil {
		logger.Warn("failed to get element automations (optional)", "elementID", elementID, "error", err)
	} else {
		element.Automations = automations
	}

	// Get relationships for this element
	if err := getElementRelationships(ctx, c, element); err != nil {
		logger.Warn("failed to get element relationships (optional)", "elementID", elementID, "error", err)
	}

	// Check comments status
	element.CommentsEnabled = checkAspectCommentsEnabled(ctx, c, elementID)
	logger.Debug("checked comments status", "elementID", elementID, "commentsEnabled", element.CommentsEnabled)

	// Get access policies using common function
	policies, err := getAspectAccessPolicies(ctx, c, elementID)
	if err != nil {
		logger.Warn("failed to get element access policies (optional)", "elementID", elementID, "error", err)
	} else {
		element.AccessPolicies = policies
	}
	logger.Debug("checked for access policies", "elementID", elementID, "accessPolicyCount", len(element.AccessPolicies))

	// Get roles using common function
	roles, err := getAspectRoles(ctx, c, elementID)
	if err != nil {
		logger.Warn("failed to get element roles (optional)", "elementID", elementID, "error", err)
	} else {
		element.Roles = roles
	}
	logger.Debug("checked for roles", "elementID", elementID, "roleCount", len(element.Roles))

	// Get AI Search Tables (only Elements have these, not Apps)
	searchTables, err := GetAISearchTablesForAspect(ctx, c, elementID)
	if err != nil {
		logger.Warn("failed to get AI search tables (optional)", "elementID", elementID, "error", err)
	} else {
		element.AISearchTables = searchTables
	}
	logger.Debug("got AI search tables", "elementID", elementID, "aiSearchTableCount", len(element.AISearchTables))

	// Get Dashboards (and their widgets) for this element
	dashboards, err := GetDashboardsForAspect(ctx, c, elementID)
	if err != nil {
		logger.Warn("failed to get dashboards (optional)", "elementID", elementID, "error", err)
	} else {
		element.Dashboards = dashboards
	}
	widgetCount := 0
	for _, d := range element.Dashboards {
		widgetCount += len(d.Widgets)
	}
	logger.Debug("checked for dashboards", "elementID", elementID, "dashboardCount", len(element.Dashboards), "widgetCount", widgetCount)

	// Note: Charts discovery is not implemented yet (same as apps)

	return element, nil
}

// getElementRelationships fetches relationships for an element
func getElementRelationships(ctx context.Context, c *client.Client, element *Element) error {
	query := `
		query GetElementRelationships($aspectId: ID!) {
			organization {
				aspect(id: $aspectId) {
					... on AspectElement {
						autoRelations {
							edges {
								node {
									id
									relatedAspect {
										id
										name
										__typename
									}
									columns {
										aspectField { id name }
										relatedAspectField { id name }
									}
								}
							}
						}
					}
				}
			}
		}
	`

	var result struct {
		Organization struct {
			Aspect *struct {
				AutoRelations struct {
					Edges []struct {
						Node struct {
							ID            string `json:"id"`
							RelatedAspect struct {
								ID       string `json:"id"`
								Name     string `json:"name"`
								Typename string `json:"__typename"`
							} `json:"relatedAspect"`
							Columns []struct {
								AspectField        *struct{ ID, Name string } `json:"aspectField"`
								RelatedAspectField *struct{ ID, Name string } `json:"relatedAspectField"`
							} `json:"columns"`
						} `json:"node"`
					} `json:"edges"`
				} `json:"autoRelations"`
			} `json:"aspect"`
		} `json:"organization"`
	}

	err := c.ExecuteInto(ctx, query, map[string]any{"aspectId": element.ID}, &result)
	if err != nil {
		return fmt.Errorf("failed to get element relationships: %w", err)
	}

	if result.Organization.Aspect == nil {
		return nil
	}

	for _, edge := range result.Organization.Aspect.AutoRelations.Edges {
		node := edge.Node

		// Map typename to our type
		relType := "App"
		switch node.RelatedAspect.Typename {
		case "AspectElement":
			relType = "Element"
		case "AspectTask":
			relType = "Task"
		case "AspectTable":
			relType = "Table"
		}

		rel := Relationship{
			ID:                node.ID,
			RelatedObjectID:   node.RelatedAspect.ID,
			RelatedObjectType: relType,
			RelatedObjectName: node.RelatedAspect.Name,
		}

		// Add column mappings
		for _, col := range node.Columns {
			relCol := RelationshipColumn{}
			if col.AspectField != nil {
				relCol.FieldID = col.AspectField.ID
				relCol.FieldName = col.AspectField.Name
			}
			if col.RelatedAspectField != nil {
				relCol.RelatedFieldID = col.RelatedAspectField.ID
				relCol.RelatedFieldName = col.RelatedAspectField.Name
			}
			rel.Columns = append(rel.Columns, relCol)
		}

		element.Relationships = append(element.Relationships, rel)
	}

	return nil
}

// getElementFields retrieves all fields for an element
// Uses the same query as apps since GetAspectFieldsQuery supports AspectElement
func getElementFields(ctx context.Context, c *client.Client, element *Element) error {
	query := client.GetAspectFieldsQuery

	var result struct {
		Organization struct {
			Aspect *struct {
				Fields struct {
					Edges []struct {
						Node struct {
							ID           string   `json:"id"`
							Name         string   `json:"name"`
							Typename     string   `json:"__typename"`
							Required     bool     `json:"required"`
							System       bool     `json:"system"`
							SemanticTags []string `json:"semanticTags"`
							Calculation  *struct {
								Text string `json:"text"`
							} `json:"calculation"`
							Values *struct {
								Edges []struct {
									Node struct {
										ID    string `json:"id"`
										Label string `json:"label"`
										Color string `json:"color"`
									} `json:"node"`
								} `json:"edges"`
							} `json:"values"`
						} `json:"node"`
					} `json:"edges"`
				} `json:"fields"`
			} `json:"aspect"`
		} `json:"organization"`
	}

	err := c.ExecuteInto(ctx, query, map[string]interface{}{"aspectId": element.ID}, &result)
	if err != nil {
		return err
	}

	if result.Organization.Aspect == nil {
		return nil
	}

	element.Fields = make([]Field, 0, len(result.Organization.Aspect.Fields.Edges))
	for _, edge := range result.Organization.Aspect.Fields.Edges {
		node := edge.Node
		field := Field{
			ID:           node.ID,
			Name:         node.Name,
			Type:         mapFieldType(node.Typename),
			Required:     node.Required,
			System:       node.System,
			SemanticTags: node.SemanticTags,
		}

		// Add calculation text for calculated fields
		if node.Calculation != nil {
			field.Calculation = node.Calculation.Text
		}

		// Add options for dropdown/multiselect fields
		if node.Values != nil {
			field.Options = make([]FieldOption, 0, len(node.Values.Edges))
			for _, optEdge := range node.Values.Edges {
				field.Options = append(field.Options, FieldOption{
					ID:    optEdge.Node.ID,
					Label: optEdge.Node.Label,
					Color: optEdge.Node.Color,
				})
			}
		}

		element.Fields = append(element.Fields, field)
	}

	return nil
}

// getElementLayouts retrieves layouts for an element
// Elements have a simpler layout structure - they only have an "Initiate" stage (no multi-stage workflow)
func getElementLayouts(ctx context.Context, c *client.Client, element *Element) error {
	// Query element stages - elements typically have just the Initiate stage
	result, err := client.GetAspectInitiateStage(ctx, c.Genqlient(), element.ID)
	if err != nil {
		return err
	}

	if result.Organization.Aspect == nil {
		return nil
	}

	// Type assert to get the AspectElement type
	aspectElement, ok := (*result.Organization.Aspect).(*client.GetAspectInitiateStageOrganizationAspectAspectElement)
	if !ok {
		return nil // Not an element
	}

	// Elements typically have one stage (Initiate)
	element.Layouts = make([]Layout, 0, len(aspectElement.Stages))
	for _, stage := range aspectElement.Stages {
		layout := Layout{
			ID:         stage.Id,
			Name:       stage.Name,
			Element:    element.ID,
			IsInitiate: stage.Name == "Initiate",
		}
		element.Layouts = append(element.Layouts, layout)
	}

	logger.Debug("retrieved element layouts", "elementID", element.ID, "layoutCount", len(element.Layouts))
	return nil
}

// getElementWidgets retrieves widgets for an element using genqlient
func getElementWidgets(ctx context.Context, c *client.Client, element *Element) error {
	result, err := client.GetAspectDisplayWidgets(ctx, c.Genqlient(), element.ID)
	if err != nil {
		return err
	}

	if result.Organization.Aspect == nil {
		return nil
	}

	// Type assert to get the AspectElement type
	aspectElement, ok := (*result.Organization.Aspect).(*client.GetAspectDisplayWidgetsOrganizationAspectAspectElement)
	if !ok {
		return nil // Not an element
	}

	element.Widgets = make([]Widget, 0, len(aspectElement.DisplayWidgets.Edges))
	for _, edge := range aspectElement.DisplayWidgets.Edges {
		// Node is an interface, use getter methods
		widgetType := ""
		if tn := edge.Node.GetTypename(); tn != nil {
			widgetType = *tn
		}
		widget := Widget{
			ID:   edge.Node.GetId(),
			Name: edge.Node.GetName(),
			Type: widgetType,
		}

		// Extract type-specific fields based on widget type
		switch w := edge.Node.(type) {
		case *client.GetAspectDisplayWidgetsOrganizationAspectAspectElementDisplayWidgetsDisplayWidgetConnectionEdgesDisplayWidgetEdgeNodeDisplayWidgetRelatedAspect:
			if w.Aspect != nil {
				widget.AspectID = (*w.Aspect).GetId()
			}
			widget.Columns = w.Columns
			widget.Rows = w.Rows
		case *client.GetAspectDisplayWidgetsOrganizationAspectAspectElementDisplayWidgetsDisplayWidgetConnectionEdgesDisplayWidgetEdgeNodeDisplayWidgetRelatedLinkAction:
			if w.Aspect != nil {
				widget.AspectID = (*w.Aspect).GetId()
			}
			widget.ButtonType = string(w.ButtonType)
			if w.Color != nil {
				widget.Color = *w.Color
			}
			if w.Icon != nil {
				widget.Icon = *w.Icon
			}
			widget.FullWidth = w.FullWidth
			widget.Columns = w.Columns
			widget.Rows = w.Rows
		case *client.GetAspectDisplayWidgetsOrganizationAspectAspectElementDisplayWidgetsDisplayWidgetConnectionEdgesDisplayWidgetEdgeNodeDisplayWidgetRelatedCreateAction:
			if w.Aspect != nil {
				widget.AspectID = (*w.Aspect).GetId()
			}
			widget.ButtonType = string(w.ButtonType)
			if w.Color != nil {
				widget.Color = *w.Color
			}
			if w.Icon != nil {
				widget.Icon = *w.Icon
			}
			widget.FullWidth = w.FullWidth
		}

		element.Widgets = append(element.Widgets, widget)
	}

	logger.Debug("retrieved element widgets", "elementID", element.ID, "widgetCount", len(element.Widgets))
	return nil
}

// ListElements returns all elements in the organization
func ListElements(ctx context.Context, c *client.Client) ([]Element, error) {
	// Use genqlient SearchAspects with filter for elements
	filter := client.BuildEqualsFilter("type", "ELEMENT")
	result, err := client.SearchAspects(ctx, c.Genqlient(), filter)
	if err != nil {
		return nil, fmt.Errorf("failed to list elements: %w", err)
	}

	// Extract aspects using helper
	aspects := client.ExtractAspects(result)

	elements := []Element{}
	for _, aspect := range aspects {
		if aspect.Typename == "AspectElement" {
			// Get full details
			element, err := GetElement(ctx, c, aspect.ID)
			if err != nil {
				return nil, err
			}
			elements = append(elements, *element)
		}
	}

	return elements, nil
}

// GetGroupByName retrieves a group by its name
func GetGroupByName(ctx context.Context, c *client.Client, name string) (*Group, error) {
	// Use genqlient GetGroups with name filter
	filter := client.BuildEqualsFilter("name", name)
	result, err := client.GetGroups(ctx, c.Genqlient(), filter)
	if err != nil {
		return nil, fmt.Errorf("failed to search for group: %w", err)
	}

	// Find group with matching name (case-insensitive)
	searchName := strings.ToLower(name)
	for _, edge := range result.Organization.Groups.Edges {
		if strings.ToLower(edge.Node.Name) == searchName {
			group := &Group{
				ID:          edge.Node.Id,
				Name:        edge.Node.Name,
				Approver:    edge.Node.Approver,
				Assignable:  edge.Node.Assignable,
				Mentionable: edge.Node.Mentionable,
				Watchable:   edge.Node.Watchable,
				Dynamic:     edge.Node.Dynamic,
				Tags:        edge.Node.Tags,
			}

			// Extract member IDs
			for _, userEdge := range edge.Node.Users.Edges {
				group.Members = append(group.Members, userEdge.Node.Id)
			}

			return group, nil
		}
	}

	return nil, fmt.Errorf("group not found with name: %s", name)
}

// GetGroup retrieves detailed information about a group
func GetGroup(ctx context.Context, c *client.Client, groupID string) (*Group, error) {
	// Use genqlient GetGroupByID
	result, err := client.GetGroupByID(ctx, c.Genqlient(), groupID)
	if err != nil {
		return nil, fmt.Errorf("failed to get group: %w", err)
	}

	node := result.Organization.Group
	group := &Group{
		ID:          node.Id,
		Name:        node.Name,
		Approver:    node.Approver,
		Assignable:  node.Assignable,
		Mentionable: node.Mentionable,
		Watchable:   node.Watchable,
		Dynamic:     node.Dynamic,
		Tags:        node.Tags,
	}

	// Extract member IDs
	for _, userEdge := range node.Users.Edges {
		group.Members = append(group.Members, userEdge.Node.Id)
	}

	return group, nil
}

// ListGroups returns all groups in the organization
func ListGroups(ctx context.Context, c *client.Client) ([]Group, error) {
	// Use genqlient GetGroups without filter to get all groups
	result, err := client.GetGroups(ctx, c.Genqlient(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list groups: %w", err)
	}

	groups := []Group{}
	for _, edge := range result.Organization.Groups.Edges {
		group := Group{
			ID:          edge.Node.Id,
			Name:        edge.Node.Name,
			Approver:    edge.Node.Approver,
			Assignable:  edge.Node.Assignable,
			Mentionable: edge.Node.Mentionable,
			Watchable:   edge.Node.Watchable,
			Dynamic:     edge.Node.Dynamic,
			Tags:        edge.Node.Tags,
		}

		// Extract member IDs
		for _, userEdge := range edge.Node.Users.Edges {
			group.Members = append(group.Members, userEdge.Node.Id)
		}

		groups = append(groups, group)
	}

	return groups, nil
}

// GetCloudLinkByName retrieves a CloudLink by its name
func GetCloudLinkByName(ctx context.Context, c *client.Client, name string) (*CloudLink, error) {
	cloudlinks, err := ListCloudLinks(ctx, c)
	if err != nil {
		return nil, err
	}

	for _, cl := range cloudlinks {
		if cl.Name == name {
			return &cl, nil
		}
	}

	return nil, fmt.Errorf("cloudlink not found with name: %s", name)
}

// GetCloudLink retrieves detailed information about a CloudLink
func GetCloudLink(ctx context.Context, c *client.Client, cloudlinkID string) (*CloudLink, error) {
	// Use genqlient GetCloudLink
	result, err := client.GetCloudLink(ctx, c.Genqlient(), cloudlinkID)
	if err != nil {
		return nil, fmt.Errorf("failed to get cloudlink: %w", err)
	}

	cloudLinkData := client.ExtractCloudLinkFromGetCloudLink(result)
	if cloudLinkData == nil {
		return nil, fmt.Errorf("cloudlink not found: %s", cloudlinkID)
	}

	valid := false
	if cloudLinkData.Valid != nil {
		valid = *cloudLinkData.Valid
	}

	return &CloudLink{
		ID:     cloudLinkData.ID,
		Name:   cloudLinkData.Name,
		Type:   cloudLinkData.Type,
		System: cloudLinkData.System,
		Valid:  valid,
	}, nil
}

// ListCloudLinks returns all CloudLinks in the organization
func ListCloudLinks(ctx context.Context, c *client.Client) ([]CloudLink, error) {
	// Use genqlient ListCloudLinks
	result, err := client.ListCloudLinks(ctx, c.Genqlient())
	if err != nil {
		return nil, fmt.Errorf("failed to list cloudlinks: %w", err)
	}

	cloudLinkDataList := client.ExtractCloudLinksFromList(result)

	cloudlinks := []CloudLink{}
	for _, clData := range cloudLinkDataList {
		valid := false
		if clData.Valid != nil {
			valid = *clData.Valid
		}

		cloudlinks = append(cloudlinks, CloudLink{
			ID:     clData.ID,
			Name:   clData.Name,
			Type:   clData.Type,
			System: clData.System,
			Valid:  valid,
		})
	}

	return cloudlinks, nil
}
