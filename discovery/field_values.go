// Copyright (c) ElementumAI, Inc.
// SPDX-License-Identifier: MPL-2.0

package discovery

import (
	"context"
	"fmt"
	"strings"

	"github.com/elementumltd/elementum-cli/logger"
	"github.com/elementumltd/elementum-cli/internal/client"
)

// FieldValueInfo represents a picklist option value
type FieldValueInfo struct {
	ID     string `json:"id"`
	Label  string `json:"label"`
	Color  string `json:"color,omitempty"`
	Active bool   `json:"active"`
}

// FieldWithValues represents a field with its config type and values
type FieldWithValues struct {
	ID              string           `json:"id"`
	Name            string           `json:"name"`
	Type            string           `json:"type"`
	ConfigType      string           `json:"config_type"` // "static", "dynamic", "api", or "unknown"
	RelatedAspectID string           `json:"related_aspect_id,omitempty"`
	Values          []FieldValueInfo `json:"values,omitempty"`
}

// GetFieldWithValues fetches a field by name and returns its values and config info
func GetFieldWithValues(ctx context.Context, c *client.Client, aspectID, fieldName string) (*FieldWithValues, error) {
	logger.Debug("fetching field with values", "aspectID", aspectID, "fieldName", fieldName)

	// First, get all fields to find the field ID and type
	resp, err := client.GetAspectFieldsNoValues(ctx, c.Genqlient(), aspectID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch fields: %w", err)
	}

	if resp.Organization.Aspect == nil {
		return nil, fmt.Errorf("aspect not found: %s", aspectID)
	}

	// Find the field by name
	var fieldID string
	var fieldType string

	aspect := *resp.Organization.Aspect

	// Extract fields based on aspect type
	var fields []client.FieldDetailsNoValues
	switch a := aspect.(type) {
	case *client.GetAspectFieldsNoValuesOrganizationAspectAspectApp:
		for _, edge := range a.Fields.Edges {
			fields = append(fields, edge.Node)
		}
	case *client.GetAspectFieldsNoValuesOrganizationAspectAspectElement:
		for _, edge := range a.Fields.Edges {
			fields = append(fields, edge.Node)
		}
	case *client.GetAspectFieldsNoValuesOrganizationAspectAspectTask:
		for _, edge := range a.Fields.Edges {
			fields = append(fields, edge.Node)
		}
	default:
		return nil, fmt.Errorf("unsupported aspect type")
	}

	// Find matching field
	for _, f := range fields {
		if strings.EqualFold(f.GetName(), fieldName) {
			fieldID = f.GetId()
			if f.GetTypename() != nil {
				fieldType = *f.GetTypename()
			}
			break
		}
	}

	if fieldID == "" {
		return nil, fmt.Errorf("field %q not found in aspect", fieldName)
	}

	// Check if it's a picklist field based on typename
	if fieldType != "AspectPicklistField" && fieldType != "AspectMultiPicklistField" {
		return nil, fmt.Errorf("field %q is not a picklist/dropdown field (type: %s)", fieldName, fieldType)
	}

	result := &FieldWithValues{
		ID:         fieldID,
		Name:       fieldName,
		Type:       mapFieldTypeSimple(fieldType),
		ConfigType: "static", // Default to static, we'll try to fetch values
	}

	// Try to fetch values using GetFieldValues
	valuesResp, err := client.GetFieldValues(ctx, c.Genqlient(), aspectID, fieldID)
	if err != nil {
		logger.Warn("failed to fetch field values", "fieldID", fieldID, "error", err)
		result.ConfigType = "unknown"
	} else if valuesResp.Organization.Aspect != nil {
		fieldData := (*valuesResp.Organization.Aspect).GetField()
		if fieldData != nil {
			switch f := fieldData.(type) {
			case *client.GetFieldValuesOrganizationAspectFieldAspectPicklistField:
				for _, edge := range f.Values.Edges {
					color := ""
					if edge.Node.Color != nil {
						color = *edge.Node.Color
					}
					result.Values = append(result.Values, FieldValueInfo{
						ID:     edge.Node.Id,
						Label:  edge.Node.Label,
						Color:  color,
						Active: edge.Node.Active,
					})
				}
			case *client.GetFieldValuesOrganizationAspectFieldAspectMultiPicklistField:
				for _, edge := range f.Values.Edges {
					color := ""
					if edge.Node.Color != nil {
						color = *edge.Node.Color
					}
					result.Values = append(result.Values, FieldValueInfo{
						ID:     edge.Node.Id,
						Label:  edge.Node.Label,
						Color:  color,
						Active: edge.Node.Active,
					})
				}
			}
		}
	}

	// If no values found, it might be a dynamic or API picklist
	if len(result.Values) == 0 {
		result.ConfigType = "dynamic_or_api"
	}

	logger.Debug("fetched field with values", "fieldID", fieldID, "configType", result.ConfigType, "valueCount", len(result.Values))
	return result, nil
}

// mapFieldTypeSimple maps GraphQL typename to simple field type
func mapFieldTypeSimple(typename string) string {
	switch typename {
	case "AspectPicklistField":
		return "dropdown"
	case "AspectMultiPicklistField":
		return "multi_select"
	default:
		return typename
	}
}
