// Copyright (c) ElementumAI, Inc.
// SPDX-License-Identifier: MPL-2.0

package discovery

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/elementumltd/elementum-cli/logger"
	"github.com/elementumltd/elementum-cli/internal/client"
)

// recordListItemGetter is an interface for accessing RecordListItem fields.
// This is needed because genqlient generates different concrete types for each
// aspect type's record nodes, but they all have the same methods from the fragment.
type recordListItemGetter interface {
	GetId() string
	GetHandle() string
	GetTitle() *string
	GetCreatedAt() *string
	GetUpdatedAt() *string
	GetUrl() string
	GetData() json.RawMessage
	GetStatus() *client.RecordListItemStatusAspectPicklistValue
	GetCreatedBy() *client.RecordListItemCreatedByUser
}

// PicklistValueMap maps picklist value IDs to their labels
// Key format: "fieldName:valueID" -> label
type PicklistValueMap map[string]string

// uuidRegex matches UUID format strings
var uuidRegex = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

// FetchPicklistValues fetches all picklist field values for an aspect
// Returns a map of "fieldName:valueID" -> label
func FetchPicklistValues(ctx context.Context, c *client.Client, aspectID string) (PicklistValueMap, error) {
	logger.Debug("fetching picklist field values", "aspectID", aspectID)

	resp, err := client.GetAspectPicklistFields(ctx, c.Genqlient(), aspectID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch picklist fields: %w", err)
	}

	if resp.Organization.Aspect == nil {
		return nil, fmt.Errorf("aspect not found: %s", aspectID)
	}

	valueMap := make(PicklistValueMap)

	// Dereference the aspect pointer (it's a pointer to interface)
	aspect := *resp.Organization.Aspect

	// Extract picklist values from fields
	for _, edge := range aspect.GetFields().Edges {
		node := edge.Node
		switch f := node.(type) {
		case *client.GetAspectPicklistFieldsOrganizationAspectFieldsAspectFieldConnectionEdgesAspectFieldEdgeNodeAspectPicklistField:
			fieldName := f.GetName()
			for _, valueEdge := range f.GetValues().Edges {
				key := fieldName + ":" + valueEdge.Node.Id
				valueMap[key] = valueEdge.Node.Label
			}
		case *client.GetAspectPicklistFieldsOrganizationAspectFieldsAspectFieldConnectionEdgesAspectFieldEdgeNodeAspectMultiPicklistField:
			fieldName := f.GetName()
			for _, valueEdge := range f.GetValues().Edges {
				key := fieldName + ":" + valueEdge.Node.Id
				valueMap[key] = valueEdge.Node.Label
			}
		}
	}

	logger.Debug("fetched picklist values", "count", len(valueMap))
	return valueMap, nil
}

// ResolvePicklistValues resolves picklist IDs to labels in record data
func ResolvePicklistValues(records []Record, valueMap PicklistValueMap) {
	for i := range records {
		for fieldName, value := range records[i].Data {
			records[i].Data[fieldName] = resolveValue(fieldName, value, valueMap)
		}
	}
}

// resolveValue recursively resolves picklist IDs to labels
func resolveValue(fieldName string, value interface{}, valueMap PicklistValueMap) interface{} {
	if value == nil {
		return nil
	}

	switch v := value.(type) {
	case string:
		// Check if it's a UUID that might be a picklist value
		if uuidRegex.MatchString(v) {
			key := fieldName + ":" + v
			if label, ok := valueMap[key]; ok {
				return label
			}
		}
		return v
	case []interface{}:
		// Array of values (multi-select)
		resolved := make([]interface{}, len(v))
		for i, item := range v {
			resolved[i] = resolveValue(fieldName, item, valueMap)
		}
		return resolved
	default:
		return v
	}
}

// ListRecords fetches records from an aspect by ID
func ListRecords(ctx context.Context, c *client.Client, aspectID string, opts RecordListOptions) (*RecordListResult, error) {
	if opts.Limit <= 0 {
		opts.Limit = 25
	}

	logger.Debug("fetching records from aspect", "aspectID", aspectID, "limit", opts.Limit, "after", opts.After)

	// Convert limit to pointer for genqlient
	limit := opts.Limit
	var after *string
	if opts.After != "" {
		after = &opts.After
	}

	resp, err := client.ListAspectRecords(ctx, c.Genqlient(), aspectID, &limit, after)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch records: %w", err)
	}

	if resp.Organization.Aspect == nil {
		return nil, fmt.Errorf("aspect not found: %s", aspectID)
	}

	result := &RecordListResult{
		Records:    []Record{},
		AspectID:   aspectID,
		AspectType: "",
	}

	// Extract records based on aspect type (polymorphic union)
	aspect := *resp.Organization.Aspect
	switch a := aspect.(type) {
	case *client.ListAspectRecordsOrganizationAspectAspectApp:
		result.AspectName = a.Name
		result.AspectType = "App"
		result.Namespace = a.Namespace
		result.Total = a.Records.Total
		result.HasNextPage = a.Records.PageInfo.HasNextPage
		if a.Records.PageInfo.EndCursor != nil {
			result.EndCursor = *a.Records.PageInfo.EndCursor
		}
		for _, edge := range a.Records.Edges {
			// The Node embeds RecordListItem, so we can use it via our interface
			record := extractRecordFromGetter(edge.Node)
			result.Records = append(result.Records, record)
		}
	case *client.ListAspectRecordsOrganizationAspectAspectElement:
		result.AspectName = a.Name
		result.AspectType = "Element"
		result.Namespace = a.Namespace
		result.Total = a.Records.Total
		result.HasNextPage = a.Records.PageInfo.HasNextPage
		if a.Records.PageInfo.EndCursor != nil {
			result.EndCursor = *a.Records.PageInfo.EndCursor
		}
		for _, edge := range a.Records.Edges {
			record := extractRecordFromGetter(edge.Node)
			result.Records = append(result.Records, record)
		}
	case *client.ListAspectRecordsOrganizationAspectAspectTask:
		result.AspectName = a.Name
		result.AspectType = "Task"
		result.Namespace = a.Namespace
		result.Total = a.Records.Total
		result.HasNextPage = a.Records.PageInfo.HasNextPage
		if a.Records.PageInfo.EndCursor != nil {
			result.EndCursor = *a.Records.PageInfo.EndCursor
		}
		for _, edge := range a.Records.Edges {
			record := extractRecordFromGetter(edge.Node)
			result.Records = append(result.Records, record)
		}
	default:
		return nil, fmt.Errorf("unsupported aspect type: %T", aspect)
	}

	logger.Debug("fetched records", "count", len(result.Records), "total", result.Total, "hasNextPage", result.HasNextPage)
	return result, nil
}

// ListRecordsByNamespace resolves an aspect namespace to ID and fetches records
func ListRecordsByNamespace(ctx context.Context, c *client.Client, namespace string, opts RecordListOptions) (*RecordListResult, error) {
	logger.Debug("resolving namespace to aspect ID", "namespace", namespace)

	// Search for the aspect by namespace
	aspectID, aspectType, err := resolveAspectByNamespace(ctx, c, namespace)
	if err != nil {
		return nil, err
	}

	logger.Debug("resolved namespace", "namespace", namespace, "aspectID", aspectID, "type", aspectType)
	return ListRecords(ctx, c, aspectID, opts)
}

// ListAllRecords fetches all records from an aspect, auto-paginating
func ListAllRecords(ctx context.Context, c *client.Client, aspectID string, opts RecordListOptions) (*RecordListResult, error) {
	var allRecords []Record
	cursor := opts.After

	// First fetch to get metadata
	firstResult, err := ListRecords(ctx, c, aspectID, RecordListOptions{
		Limit: opts.Limit,
		After: cursor,
	})
	if err != nil {
		return nil, err
	}

	allRecords = append(allRecords, firstResult.Records...)
	cursor = firstResult.EndCursor

	// Continue fetching while there are more pages
	for firstResult.HasNextPage {
		nextResult, err := ListRecords(ctx, c, aspectID, RecordListOptions{
			Limit: opts.Limit,
			After: cursor,
		})
		if err != nil {
			return nil, err
		}
		allRecords = append(allRecords, nextResult.Records...)
		cursor = nextResult.EndCursor
		firstResult.HasNextPage = nextResult.HasNextPage
	}

	return &RecordListResult{
		Records:     allRecords,
		Total:       firstResult.Total,
		HasNextPage: false,
		EndCursor:   "",
		AspectID:    firstResult.AspectID,
		AspectName:  firstResult.AspectName,
		AspectType:  firstResult.AspectType,
		Namespace:   firstResult.Namespace,
	}, nil
}

// resolveAspectByNamespace searches for an aspect by namespace and returns its ID and type
func resolveAspectByNamespace(ctx context.Context, c *client.Client, namespace string) (string, string, error) {
	// Build filter for exact namespace match
	filter := buildNamespaceFilter(namespace)

	resp, err := client.SearchAspects(ctx, c.Genqlient(), filter)
	if err != nil {
		return "", "", fmt.Errorf("failed to search aspects: %w", err)
	}

	aspects := client.ExtractAspects(resp)
	if len(aspects) == 0 {
		return "", "", fmt.Errorf("aspect not found with namespace: %s", namespace)
	}

	// Find exact match (case-insensitive)
	for _, aspect := range aspects {
		if strings.EqualFold(aspect.Namespace, namespace) {
			return aspect.ID, mapObjectType(aspect.Typename), nil
		}
	}

	// If no exact match, return first result
	return aspects[0].ID, mapObjectType(aspects[0].Typename), nil
}

// buildNamespaceFilter creates a filter for searching aspects by namespace
func buildNamespaceFilter(namespace string) *json.RawMessage {
	// Use LIKE filter to match namespace (it will be case-insensitive)
	filter := map[string]interface{}{
		"type":  "LIKE",
		"field": "namespace",
		"value": map[string]interface{}{
			"type":  "TEXT",
			"value": namespace,
		},
		"mode":            "EQUALS",
		"caseInsensitive": true,
	}
	data, _ := json.Marshal(filter)
	raw := json.RawMessage(data)
	return &raw
}

// extractRecordFromGetter converts a record node (implementing recordListItemGetter) to our Record type
func extractRecordFromGetter(node recordListItemGetter) Record {
	record := Record{
		ID:     node.GetId(),
		Handle: node.GetHandle(),
		URL:    node.GetUrl(),
		Data:   make(map[string]interface{}),
	}

	// Extract nullable fields
	if title := node.GetTitle(); title != nil {
		record.Title = *title
	}
	if createdAt := node.GetCreatedAt(); createdAt != nil {
		record.CreatedAt = *createdAt
	}
	if updatedAt := node.GetUpdatedAt(); updatedAt != nil {
		record.UpdatedAt = *updatedAt
	}

	// Extract status
	if status := node.GetStatus(); status != nil {
		record.Status = status.Label
		record.StatusID = status.Id
	}

	// Extract created by
	if createdBy := node.GetCreatedBy(); createdBy != nil {
		record.CreatedBy = createdBy.Name
	}

	// Parse the data JSON
	if data := node.GetData(); data != nil {
		if err := json.Unmarshal(data, &record.Data); err != nil {
			logger.Debug("failed to parse record data", "error", err, "recordID", record.ID)
		}
	}

	return record
}

// ExtractDisplayValue extracts a display-friendly string from a field value.
// Handles various field types:
//   - string: returned as-is
//   - map with "label": returns the label (picklist)
//   - map with "name": returns the name (user/group)
//   - []interface{}: joins labels/names with ", " (multi-select)
//   - nil: returns ""
func ExtractDisplayValue(value interface{}) string {
	if value == nil {
		return ""
	}

	switch v := value.(type) {
	case string:
		return v
	case float64:
		// Numbers come as float64 from JSON
		if v == float64(int64(v)) {
			return fmt.Sprintf("%d", int64(v))
		}
		return fmt.Sprintf("%v", v)
	case bool:
		if v {
			return "Yes"
		}
		return "No"
	case map[string]interface{}:
		// Try label first (picklist), then name (user/group)
		if label, ok := v["label"].(string); ok {
			return label
		}
		if name, ok := v["name"].(string); ok {
			return name
		}
		// Fallback: try to get any string value
		for _, val := range v {
			if s, ok := val.(string); ok {
				return s
			}
		}
		return ""
	case []interface{}:
		// Multi-select or array - join all values
		var parts []string
		for _, item := range v {
			if s := ExtractDisplayValue(item); s != "" {
				parts = append(parts, s)
			}
		}
		return strings.Join(parts, ", ")
	default:
		return fmt.Sprintf("%v", v)
	}
}

// GetFieldValue retrieves a field value from a record's data and formats it for display
func GetFieldValue(record Record, fieldName string) string {
	value, ok := record.Data[fieldName]
	if !ok {
		return ""
	}
	return ExtractDisplayValue(value)
}

// DeleteRecord deletes a record by its full ID
func DeleteRecord(ctx context.Context, c *client.Client, recordID string) error {
	logger.Debug("deleting record", "recordID", recordID)

	_, err := client.DeleteRecord(ctx, c.Genqlient(), recordID)
	if err != nil {
		return fmt.Errorf("failed to delete record: %w", err)
	}

	logger.Debug("record deleted successfully", "recordID", recordID)
	return nil
}

// DeleteRecordByHandle deletes a record by handle within an aspect namespace
func DeleteRecordByHandle(ctx context.Context, c *client.Client, namespace string, handle string) error {
	logger.Debug("resolving record for deletion", "namespace", namespace, "handle", handle)

	// First resolve the namespace to aspect ID
	aspectID, _, err := resolveAspectByNamespace(ctx, c, namespace)
	if err != nil {
		return fmt.Errorf("failed to resolve namespace %q: %w", namespace, err)
	}

	// Build the full record ID (aspectID:handle format)
	recordID := aspectID + ":" + handle

	return DeleteRecord(ctx, c, recordID)
}

// DeleteRecords deletes multiple records by their full IDs (one at a time)
func DeleteRecords(ctx context.Context, c *client.Client, recordIDs []string) (deleted int, errors []error) {
	for _, id := range recordIDs {
		if err := DeleteRecord(ctx, c, id); err != nil {
			errors = append(errors, fmt.Errorf("failed to delete %s: %w", id, err))
		} else {
			deleted++
		}
	}
	return deleted, errors
}

// BulkDeleteAllRecords deletes all records in an aspect using the bulk delete API
// Returns the number of records deleted
func BulkDeleteAllRecords(ctx context.Context, c *client.Client, aspectID string) (int, error) {
	logger.Debug("bulk deleting all records", "aspectID", aspectID)

	// Build a filter that matches all records using the TRUE filter type
	filter := json.RawMessage(`{"type":"TRUE"}`)

	input := client.AspectRecordBulkDeleteInput{
		Filter: filter,
	}

	resp, err := client.BulkDeleteRecords(ctx, c.Genqlient(), aspectID, input)
	if err != nil {
		return 0, fmt.Errorf("failed to bulk delete records: %w", err)
	}

	deleted := resp.AspectRecordBulkDelete
	logger.Debug("bulk delete completed", "deleted", deleted)
	return deleted, nil
}
