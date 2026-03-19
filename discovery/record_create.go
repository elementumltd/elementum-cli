// Copyright (c) ElementumAI, Inc.
// SPDX-License-Identifier: MPL-2.0

package discovery

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/elementumltd/elementum-cli/logger"
	"github.com/elementumltd/elementum-cli/internal/client"
)

// AspectFieldInfo contains field information for record creation
type AspectFieldInfo struct {
	ID           string
	Name         string
	Type         string // AspectTextField, AspectNumberField, etc.
	Required     bool
	System       bool
	SemanticTags []string
	Options      []FieldOptionInfo // For picklist/multi-picklist fields
}

// AspectInfo contains information about an aspect and its fields
type AspectInfo struct {
	ID     string
	Name   string
	Type   AspectType
	Fields []AspectFieldInfo
}

// FieldOptionInfo represents a picklist option
type FieldOptionInfo struct {
	ID    string
	Label string
}

// RecordCreateInput represents input for creating a record
type RecordCreateInput struct {
	AspectID    string
	Fields      map[string]interface{} // Field ID -> value
	Attachments []AttachmentInput
}

// AttachmentInput represents a file attachment
type AttachmentInput struct {
	FieldName string // Field name to attach to (optional)
	FieldID   string // Field ID to attach to (optional)
	FilePath  string // Path to the file
	FileName  string // Optional custom name for the file
}

// RecordCreateResult contains the result of record creation
type RecordCreateResult struct {
	ID        string
	Handle    string
	Title     string
	URL       string
	CreatedAt string
}

// GetAspectFields fetches all fields for an aspect
func GetAspectFields(ctx context.Context, c *client.Client, aspectID string) ([]AspectFieldInfo, error) {
	info, err := GetAspectInfo(ctx, c, aspectID)
	if err != nil {
		return nil, err
	}
	return info.Fields, nil
}

// GetAspectInfo fetches aspect information including type and fields
func GetAspectInfo(ctx context.Context, c *client.Client, aspectID string) (*AspectInfo, error) {
	logger.Debug("fetching aspect info", "aspectID", aspectID)

	resp, err := client.GetAspectFieldsForRecord(ctx, c.Genqlient(), aspectID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch aspect fields: %w", err)
	}

	if resp.Organization.Aspect == nil {
		return nil, fmt.Errorf("aspect not found: %s", aspectID)
	}

	aspect := *resp.Organization.Aspect
	info := &AspectInfo{
		ID: aspectID,
	}

	switch a := aspect.(type) {
	case *client.GetAspectFieldsForRecordOrganizationAspectAspectApp:
		info.Type = AspectTypeApp
		info.Name = a.GetName()
		info.Fields = extractFieldsFromApp(a.Fields.Edges)
	case *client.GetAspectFieldsForRecordOrganizationAspectAspectElement:
		info.Type = AspectTypeElement
		info.Name = a.GetName()
		info.Fields = extractFieldsFromElement(a.Fields.Edges)
	case *client.GetAspectFieldsForRecordOrganizationAspectAspectTask:
		info.Type = AspectTypeTask
		info.Name = a.GetName()
		info.Fields = extractFieldsFromTask(a.Fields.Edges)
	default:
		return nil, fmt.Errorf("unsupported aspect type: %T", aspect)
	}

	logger.Debug("fetched aspect info", "type", info.Type, "fieldCount", len(info.Fields))
	return info, nil
}

// GetIDField returns the ID/HANDLE field for an aspect
func GetIDField(fields []AspectFieldInfo) *AspectFieldInfo {
	for i := range fields {
		for _, tag := range fields[i].SemanticTags {
			if tag == "HANDLE" {
				return &fields[i]
			}
		}
	}
	return nil
}

// RequiresIDField returns true if the aspect requires an ID field to be provided
// HANDLE fields with system:true auto-populate, system:false require user input
func RequiresIDField(fields []AspectFieldInfo) bool {
	idField := GetIDField(fields)
	if idField == nil {
		return false
	}
	// system:true means auto-populated, system:false means user must provide
	return !idField.System
}

// GetTitleField returns the TITLE field for an aspect
func GetTitleField(fields []AspectFieldInfo) *AspectFieldInfo {
	for i := range fields {
		for _, tag := range fields[i].SemanticTags {
			if tag == "TITLE" {
				return &fields[i]
			}
		}
	}
	return nil
}

// GetMissingRequiredFields returns a list of required fields that are not in the provided data
func GetMissingRequiredFields(fields []AspectFieldInfo, data map[string]interface{}, aspectType AspectType) []string {
	var missing []string
	for _, field := range fields {
		// Check if this is a HANDLE field
		hasHandleTag := false
		for _, tag := range field.SemanticTags {
			if tag == "HANDLE" {
				hasHandleTag = true
				break
			}
		}

		// HANDLE fields: system:true auto-populates, system:false requires user input
		if hasHandleTag {
			if !field.System {
				// Non-system HANDLE field must be provided by user
				if _, ok := data[field.ID]; !ok {
					if _, ok := data[field.Name]; !ok {
						missing = append(missing, field.Name)
					}
				}
			}
			continue
		}

		// For other fields, skip if not required or if system-managed
		if !field.Required || field.System {
			continue
		}

		// Skip fields that can't be set directly
		switch field.Type {
		case "AspectHandleField", "AspectAttachmentField",
			"AspectCalculationField", "AspectReferenceField",
			"AspectCreatedAtField", "AspectUpdatedAtField",
			"AspectCreatedByField":
			continue
		}

		// Check if field is provided
		if _, ok := data[field.ID]; !ok {
			if _, ok := data[field.Name]; !ok {
				missing = append(missing, field.Name)
			}
		}
	}
	return missing
}

// extractFieldsFromApp extracts field info from App response
func extractFieldsFromApp(edges []client.GetAspectFieldsForRecordOrganizationAspectAspectAppFieldsAspectFieldConnectionEdgesAspectFieldEdge) []AspectFieldInfo {
	fields := make([]AspectFieldInfo, 0, len(edges))
	for _, edge := range edges {
		fields = append(fields, extractFieldInfo(edge.Node))
	}
	return fields
}

// extractFieldsFromElement extracts field info from Element response
func extractFieldsFromElement(edges []client.GetAspectFieldsForRecordOrganizationAspectAspectElementFieldsAspectFieldConnectionEdgesAspectFieldEdge) []AspectFieldInfo {
	fields := make([]AspectFieldInfo, 0, len(edges))
	for _, edge := range edges {
		fields = append(fields, extractFieldInfo(edge.Node))
	}
	return fields
}

// extractFieldsFromTask extracts field info from Task response
func extractFieldsFromTask(edges []client.GetAspectFieldsForRecordOrganizationAspectAspectTaskFieldsAspectFieldConnectionEdgesAspectFieldEdge) []AspectFieldInfo {
	fields := make([]AspectFieldInfo, 0, len(edges))
	for _, edge := range edges {
		fields = append(fields, extractFieldInfo(edge.Node))
	}
	return fields
}

// picklistValuesGetter is an interface for types that have picklist values
// Both picklist and multi-picklist use the same return type
type picklistValuesGetter interface {
	GetValues() client.RecordFieldInfoValuesAspectPicklistValueConnection
}

// extractFieldInfo extracts field info from a RecordFieldInfo interface
func extractFieldInfo(node client.RecordFieldInfo) AspectFieldInfo {
	// Convert semantic tags from enum type to strings
	tags := node.GetSemanticTags()
	semanticTags := make([]string, len(tags))
	for i, tag := range tags {
		semanticTags[i] = string(tag)
	}

	field := AspectFieldInfo{
		ID:           node.GetId(),
		Name:         node.GetName(),
		Type:         safeStringFromPtr(node.GetTypename()),
		Required:     node.GetRequired(),
		System:       node.GetSystem(),
		SemanticTags: semanticTags,
	}

	// Extract picklist options if the type provides them
	// Both picklist and multi-picklist types implement picklistValuesGetter
	if pv, ok := node.(picklistValuesGetter); ok {
		values := pv.GetValues()
		for _, valueEdge := range values.Edges {
			field.Options = append(field.Options, FieldOptionInfo{
				ID:    valueEdge.Node.Id,
				Label: valueEdge.Node.Label,
			})
		}
	}

	return field
}

// ResolveFieldNameToID resolves a field name to its ID
func ResolveFieldNameToID(fields []AspectFieldInfo, nameOrID string) (string, *AspectFieldInfo, error) {
	// Check if it's already a UUID
	if uuidRegex.MatchString(nameOrID) {
		for i := range fields {
			if fields[i].ID == nameOrID {
				return nameOrID, &fields[i], nil
			}
		}
		// Accept the UUID even if not found in fields (might be a valid field we didn't fetch)
		return nameOrID, nil, nil
	}

	// Search by name (case-insensitive)
	for i := range fields {
		if strings.EqualFold(fields[i].Name, nameOrID) {
			return fields[i].ID, &fields[i], nil
		}
	}

	return "", nil, fmt.Errorf("field not found: %s", nameOrID)
}

// ResolveFieldValue converts a string value to the appropriate type for a field
func ResolveFieldValue(field *AspectFieldInfo, value string) (interface{}, error) {
	if field == nil {
		// Unknown field type, return as-is
		return value, nil
	}

	switch field.Type {
	case "AspectTextField", "AspectHtmlField":
		return value, nil

	case "AspectNumberField":
		// Try to parse as integer first, then float
		if i, err := strconv.ParseInt(value, 10, 64); err == nil {
			return i, nil
		}
		if f, err := strconv.ParseFloat(value, 64); err == nil {
			return int64(f), nil
		}
		return nil, fmt.Errorf("invalid number value: %s", value)

	case "AspectDecimalField":
		f, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid decimal value: %s", value)
		}
		return f, nil

	case "AspectBooleanField":
		lower := strings.ToLower(value)
		switch lower {
		case "true", "yes", "1", "on":
			return true, nil
		case "false", "no", "0", "off":
			return false, nil
		default:
			return nil, fmt.Errorf("invalid boolean value: %s (use true/false, yes/no, 1/0)", value)
		}

	case "AspectDateField":
		// Validate date format (YYYY-MM-DD)
		if _, err := time.Parse("2006-01-02", value); err != nil {
			return nil, fmt.Errorf("invalid date format: %s (use YYYY-MM-DD)", value)
		}
		return value, nil

	case "AspectDateTimeField":
		// Accept various datetime formats
		formats := []string{
			time.RFC3339,
			"2006-01-02T15:04:05",
			"2006-01-02 15:04:05",
			"2006-01-02T15:04",
			"2006-01-02 15:04",
		}
		for _, format := range formats {
			if t, err := time.Parse(format, value); err == nil {
				return t.Format(time.RFC3339), nil
			}
		}
		return nil, fmt.Errorf("invalid datetime format: %s (use YYYY-MM-DDTHH:MM:SS)", value)

	case "AspectPicklistField":
		return resolvePicklistValue(field, value)

	case "AspectMultiPicklistField":
		return resolveMultiPicklistValue(field, value)

	case "AspectUserField":
		// For user fields, accept either a UUID or email
		// If it's a UUID, return as-is
		if uuidRegex.MatchString(value) {
			return value, nil
		}
		// Otherwise, it needs to be resolved by the caller using user lookup
		return nil, fmt.Errorf("user field requires UUID; use 'ei users' to find user IDs or provide email for lookup")

	case "AspectGroupField":
		// Similar to user fields
		if uuidRegex.MatchString(value) {
			return value, nil
		}
		return nil, fmt.Errorf("group field requires UUID; use 'ei groups' to find group IDs")

	case "AspectJsonField":
		// Validate JSON
		var js json.RawMessage
		if err := json.Unmarshal([]byte(value), &js); err != nil {
			return nil, fmt.Errorf("invalid JSON value: %w", err)
		}
		return js, nil

	default:
		// Unknown field type, return as-is
		return value, nil
	}
}

// resolvePicklistValue resolves a picklist value by label or ID
func resolvePicklistValue(field *AspectFieldInfo, value string) (string, error) {
	// If it's a UUID, check if it's a valid option
	if uuidRegex.MatchString(value) {
		for _, opt := range field.Options {
			if opt.ID == value {
				return value, nil
			}
		}
		// Accept the UUID even if not in options (might be valid but not fetched)
		return value, nil
	}

	// Search by label (case-insensitive)
	for _, opt := range field.Options {
		if strings.EqualFold(opt.Label, value) {
			return opt.ID, nil
		}
	}

	// List available options in error
	var optionLabels []string
	for _, opt := range field.Options {
		optionLabels = append(optionLabels, opt.Label)
	}
	return "", fmt.Errorf("picklist option not found: %s (available: %s)", value, strings.Join(optionLabels, ", "))
}

// resolveMultiPicklistValue resolves comma-separated picklist values
func resolveMultiPicklistValue(field *AspectFieldInfo, value string) ([]string, error) {
	parts := strings.Split(value, ",")
	ids := make([]string, 0, len(parts))

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		id, err := resolvePicklistValue(field, part)
		if err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}

	return ids, nil
}

// PrepareAttachment reads a file and prepares it for upload
func PrepareAttachment(filePath string, customName string, fieldID string) (*client.AspectRecordAttachmentInput, error) {
	// Read file content
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", filePath, err)
	}

	// Determine filename
	fileName := customName
	if fileName == "" {
		fileName = filepath.Base(filePath)
	}

	// Detect MIME type
	mimeType := http.DetectContentType(content)
	// For common extensions, use more specific types
	ext := strings.ToLower(filepath.Ext(filePath))
	switch ext {
	case ".pdf":
		mimeType = "application/pdf"
	case ".png":
		mimeType = "image/png"
	case ".jpg", ".jpeg":
		mimeType = "image/jpeg"
	case ".gif":
		mimeType = "image/gif"
	case ".txt":
		mimeType = "text/plain"
	case ".csv":
		mimeType = "text/csv"
	case ".json":
		mimeType = "application/json"
	case ".xml":
		mimeType = "application/xml"
	case ".doc":
		mimeType = "application/msword"
	case ".docx":
		mimeType = "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
	case ".xls":
		mimeType = "application/vnd.ms-excel"
	case ".xlsx":
		mimeType = "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
	}

	// Base64 encode the content
	encoded := base64.StdEncoding.EncodeToString(content)

	attachment := &client.AspectRecordAttachmentInput{
		Name:       fileName,
		Attachment: encoded,
		FileType:   mimeType,
	}

	// Add field ID if specified
	if fieldID != "" {
		attachment.FieldId = &fieldID
	}

	logger.Debug("prepared attachment", "file", filePath, "name", fileName, "mimeType", mimeType, "size", len(content))
	return attachment, nil
}

// CreateRecord creates a new record in an aspect
func CreateRecord(ctx context.Context, c *client.Client, input RecordCreateInput) (*RecordCreateResult, error) {
	logger.Debug("creating record", "aspectID", input.AspectID, "fieldCount", len(input.Fields))

	// Build the record data
	recordData := make(map[string]interface{})
	for fieldID, value := range input.Fields {
		recordData[fieldID] = value
	}

	// Convert to JSON for the GraphQL scalar
	dataJSON, err := json.Marshal(recordData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal record data: %w", err)
	}

	// Prepare attachments if present
	var attachments []client.AspectRecordAttachmentInput
	for _, att := range input.Attachments {
		prepared, err := PrepareAttachment(att.FilePath, att.FileName, att.FieldID)
		if err != nil {
			return nil, err
		}
		attachments = append(attachments, *prepared)
	}

	// Create the record using the appropriate mutation
	if len(attachments) > 0 {
		resp, err := client.CreateRecordWithAttachments(ctx, c.Genqlient(), input.AspectID, json.RawMessage(dataJSON), attachments)
		if err != nil {
			return nil, fmt.Errorf("failed to create record: %w", err)
		}
		return &RecordCreateResult{
			ID:        resp.AspectRecordCreate.GetId(),
			Handle:    resp.AspectRecordCreate.GetHandle(),
			Title:     safeStringFromPtr(resp.AspectRecordCreate.GetTitle()),
			URL:       resp.AspectRecordCreate.GetUrl(),
			CreatedAt: safeStringFromPtr(resp.AspectRecordCreate.GetCreatedAt()),
		}, nil
	}

	// Use the simpler mutation without attachments
	resp, err := client.CreateRecord(ctx, c.Genqlient(), input.AspectID, json.RawMessage(dataJSON))
	if err != nil {
		return nil, fmt.Errorf("failed to create record: %w", err)
	}
	return &RecordCreateResult{
		ID:        resp.AspectRecordCreate.GetId(),
		Handle:    resp.AspectRecordCreate.GetHandle(),
		Title:     safeStringFromPtr(resp.AspectRecordCreate.GetTitle()),
		URL:       resp.AspectRecordCreate.GetUrl(),
		CreatedAt: safeStringFromPtr(resp.AspectRecordCreate.GetCreatedAt()),
	}, nil
}

// LookupUserByEmail finds a user by email and returns their ID
func LookupUserByEmail(ctx context.Context, c *client.Client, email string) (string, error) {
	// Build filter for email match
	filter := map[string]interface{}{
		"type":  "LIKE",
		"field": "email",
		"value": map[string]interface{}{
			"type":  "TEXT",
			"value": email,
		},
		"mode":            "EQUALS",
		"caseInsensitive": true,
	}
	filterJSON, _ := json.Marshal(filter)
	raw := json.RawMessage(filterJSON)

	resp, err := client.GetUsers(ctx, c.Genqlient(), &raw)
	if err != nil {
		return "", fmt.Errorf("failed to lookup user: %w", err)
	}

	for _, edge := range resp.Organization.Users.Edges {
		if strings.EqualFold(edge.Node.Email, email) {
			return edge.Node.Id, nil
		}
	}

	return "", fmt.Errorf("user not found with email: %s", email)
}

// LookupGroupByName finds a group by name and returns its ID
func LookupGroupByName(ctx context.Context, c *client.Client, name string) (string, error) {
	// Build filter for name match
	filter := map[string]interface{}{
		"type":  "LIKE",
		"field": "name",
		"value": map[string]interface{}{
			"type":  "TEXT",
			"value": name,
		},
		"mode":            "EQUALS",
		"caseInsensitive": true,
	}
	filterJSON, _ := json.Marshal(filter)
	raw := json.RawMessage(filterJSON)

	resp, err := client.GetGroups(ctx, c.Genqlient(), &raw)
	if err != nil {
		return "", fmt.Errorf("failed to lookup group: %w", err)
	}

	for _, edge := range resp.Organization.Groups.Edges {
		if strings.EqualFold(edge.Node.Name, name) {
			return edge.Node.Id, nil
		}
	}

	return "", fmt.Errorf("group not found with name: %s", name)
}

// safeStringFromPtr safely dereferences a string pointer
func safeStringFromPtr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// ParseFieldAssignment parses a "Name=Value" string into field name and value
func ParseFieldAssignment(s string) (name, value string, err error) {
	// Find the first = sign
	idx := strings.Index(s, "=")
	if idx == -1 {
		return "", "", fmt.Errorf("invalid field assignment: %q (expected format: Name=Value)", s)
	}

	name = strings.TrimSpace(s[:idx])
	value = s[idx+1:] // Don't trim value - preserve leading/trailing spaces if intentional

	if name == "" {
		return "", "", fmt.Errorf("invalid field assignment: %q (field name cannot be empty)", s)
	}

	return name, value, nil
}

// ParseAttachmentAssignment parses a "FieldName=/path/to/file" string
func ParseAttachmentAssignment(s string) (fieldName, filePath string, err error) {
	// Find the first = sign
	idx := strings.Index(s, "=")
	if idx == -1 {
		return "", "", fmt.Errorf("invalid attachment: %q (expected format: FieldName=/path/to/file)", s)
	}

	fieldName = strings.TrimSpace(s[:idx])
	filePath = strings.TrimSpace(s[idx+1:])

	if filePath == "" {
		return "", "", fmt.Errorf("invalid attachment: %q (file path cannot be empty)", s)
	}

	return fieldName, filePath, nil
}

// JSONInputFile represents the structure of a --from-file JSON input
type JSONInputFile struct {
	Fields      map[string]interface{} `json:"-"` // Populated from top-level keys
	Attachments []JSONAttachment       `json:"attachments,omitempty"`
}

// JSONAttachment represents an attachment in a JSON input file
type JSONAttachment struct {
	Path      string `json:"path"`
	Name      string `json:"name,omitempty"`
	FieldName string `json:"field,omitempty"`
}

// ParseJSONInputFile parses a JSON input file for record creation
func ParseJSONInputFile(filePath string) (*JSONInputFile, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// First, parse as a generic map to get all fields
	var rawData map[string]interface{}
	if err := json.Unmarshal(content, &rawData); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}

	result := &JSONInputFile{
		Fields: make(map[string]interface{}),
	}

	// Process each key
	for key, value := range rawData {
		if key == "attachments" {
			// Parse attachments array
			attachmentsJSON, _ := json.Marshal(value)
			if err := json.Unmarshal(attachmentsJSON, &result.Attachments); err != nil {
				return nil, fmt.Errorf("invalid attachments format: %w", err)
			}
		} else {
			// Regular field
			result.Fields[key] = value
		}
	}

	return result, nil
}

// ValidateAttachmentPath checks if an attachment file exists and is readable
func ValidateAttachmentPath(filePath string) error {
	info, err := os.Stat(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("file not found: %s", filePath)
		}
		return fmt.Errorf("cannot access file: %w", err)
	}

	if info.IsDir() {
		return fmt.Errorf("path is a directory: %s", filePath)
	}

	return nil
}
