// Copyright (c) ElementumAI, Inc.
// SPDX-License-Identifier: MPL-2.0

package discovery

import (
	"testing"
)

func TestRelationship_GetUUIDMappings(t *testing.T) {
	rel := &Relationship{
		ID:                "rel-123",
		RelatedObjectID:   "app-456",
		RelatedObjectType: "App",
		RelatedObjectName: "Customers",
	}

	mappings := rel.GetUUIDMappings("to_customers")

	expected := "elementum_relationship.to_customers.id"
	if got := mappings["rel-123"]; got != expected {
		t.Errorf("Relationship.GetUUIDMappings()[rel-123] = %q, want %q", got, expected)
	}
}

func TestRelatedObject_GetUUIDMappings_App(t *testing.T) {
	relatedObj := &RelatedObject{
		ID:        "app-123",
		Name:      "Customers",
		Type:      "App",
		Namespace: "customers",
		Fields: []Field{
			{ID: "field-title", Name: "Title", Type: "text"},
			{ID: "field-status", Name: "Status", Type: "dropdown"},
		},
		FieldIDs: []string{"field-title", "field-status"},
	}

	mappings := relatedObj.GetUUIDMappings("customers")

	// Check object ID mapping
	expectedID := "data.elementum_app.customers.id"
	if got := mappings["app-123"]; got != expectedID {
		t.Errorf("RelatedObject.GetUUIDMappings()[app-123] = %q, want %q", got, expectedID)
	}

	// Check title field mapping (system field)
	expectedTitle := "data.elementum_app.customers.title_field_id"
	if got := mappings["field-title"]; got != expectedTitle {
		t.Errorf("RelatedObject.GetUUIDMappings()[field-title] = %q, want %q", got, expectedTitle)
	}

	// Check status field mapping (system field)
	expectedStatus := "data.elementum_app.customers.status_field_id"
	if got := mappings["field-status"]; got != expectedStatus {
		t.Errorf("RelatedObject.GetUUIDMappings()[field-status] = %q, want %q", got, expectedStatus)
	}
}

func TestRelatedObject_GetUUIDMappings_Element(t *testing.T) {
	relatedObj := &RelatedObject{
		ID:        "elem-123",
		Name:      "Products",
		Type:      "Element",
		Namespace: "products",
		Fields: []Field{
			{ID: "field-1", Name: "Title", Type: "text"},
		},
		FieldIDs: []string{"field-1"},
	}

	mappings := relatedObj.GetUUIDMappings("products")

	// Check object ID mapping uses element data source
	expectedID := "data.elementum_element.products.id"
	if got := mappings["elem-123"]; got != expectedID {
		t.Errorf("RelatedObject.GetUUIDMappings()[elem-123] = %q, want %q", got, expectedID)
	}
}

func TestRelationshipColumn_Fields(t *testing.T) {
	col := RelationshipColumn{
		FieldID:          "field-src-123",
		FieldName:        "Customer ID",
		RelatedFieldID:   "field-tgt-456",
		RelatedFieldName: "ID",
	}

	if col.FieldID != "field-src-123" {
		t.Errorf("RelationshipColumn.FieldID = %q, want %q", col.FieldID, "field-src-123")
	}
	if col.FieldName != "Customer ID" {
		t.Errorf("RelationshipColumn.FieldName = %q, want %q", col.FieldName, "Customer ID")
	}
	if col.RelatedFieldID != "field-tgt-456" {
		t.Errorf("RelationshipColumn.RelatedFieldID = %q, want %q", col.RelatedFieldID, "field-tgt-456")
	}
	if col.RelatedFieldName != "ID" {
		t.Errorf("RelationshipColumn.RelatedFieldName = %q, want %q", col.RelatedFieldName, "ID")
	}
}

func TestMapAspectTypename(t *testing.T) {
	tests := []struct {
		typename string
		want     string
	}{
		{"AspectApp", "App"},
		{"AspectElement", "Element"},
		{"AspectTask", "Task"},
		{"Table", "Table"},
		{"Unknown", "Unknown"},
		{"", "Unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.typename, func(t *testing.T) {
			if got := mapAspectTypename(tt.typename); got != tt.want {
				t.Errorf("mapAspectTypename(%q) = %q, want %q", tt.typename, got, tt.want)
			}
		})
	}
}

func TestApp_WithRelationships(t *testing.T) {
	app := &App{
		ID:        "app-main",
		Name:      "Orders",
		Namespace: "orders",
		Relationships: []Relationship{
			{
				ID:                "rel-1",
				RelatedObjectID:   "app-customers",
				RelatedObjectType: "App",
				RelatedObjectName: "Customers",
				Columns: []RelationshipColumn{
					{
						FieldID:          "field-cust-id",
						FieldName:        "Customer",
						RelatedFieldID:   "field-cust-title",
						RelatedFieldName: "Title",
					},
				},
			},
			{
				ID:                "rel-2",
				RelatedObjectID:   "elem-products",
				RelatedObjectType: "Element",
				RelatedObjectName: "Products",
				Columns: []RelationshipColumn{
					{
						FieldID:          "field-prod-id",
						FieldName:        "Product",
						RelatedFieldID:   "field-prod-title",
						RelatedFieldName: "Title",
					},
				},
			},
		},
		RelatedObjects: []RelatedObject{
			{
				ID:        "app-customers",
				Name:      "Customers",
				Type:      "App",
				Namespace: "customers",
			},
			{
				ID:        "elem-products",
				Name:      "Products",
				Type:      "Element",
				Namespace: "products",
			},
		},
	}

	// Verify relationships
	if len(app.Relationships) != 2 {
		t.Errorf("App.Relationships length = %d, want 2", len(app.Relationships))
	}

	// Verify related objects
	if len(app.RelatedObjects) != 2 {
		t.Errorf("App.RelatedObjects length = %d, want 2", len(app.RelatedObjects))
	}

	// Verify first relationship
	rel1 := app.Relationships[0]
	if rel1.RelatedObjectType != "App" {
		t.Errorf("Relationship[0].RelatedObjectType = %q, want %q", rel1.RelatedObjectType, "App")
	}
	if rel1.RelatedObjectName != "Customers" {
		t.Errorf("Relationship[0].RelatedObjectName = %q, want %q", rel1.RelatedObjectName, "Customers")
	}

	// Verify second relationship
	rel2 := app.Relationships[1]
	if rel2.RelatedObjectType != "Element" {
		t.Errorf("Relationship[1].RelatedObjectType = %q, want %q", rel2.RelatedObjectType, "Element")
	}
}

func TestRelationship_WithFilter(t *testing.T) {
	rel := &Relationship{
		ID:                "rel-123",
		RelatedObjectID:   "app-456",
		RelatedObjectType: "App",
		RelatedObjectName: "Customers",
		Filter: map[string]interface{}{
			"type": "equals",
			"field": map[string]interface{}{
				"id": "field-status",
			},
			"value": "active",
		},
		Columns: []RelationshipColumn{
			{
				FieldID:          "field-cust-ref",
				FieldName:        "Customer Reference",
				RelatedFieldID:   "field-cust-id",
				RelatedFieldName: "ID",
			},
		},
	}

	// Verify filter is present
	if rel.Filter == nil {
		t.Error("Relationship.Filter should not be nil")
	}

	// Verify filter has expected structure
	filterType, ok := rel.Filter["type"]
	if !ok || filterType != "equals" {
		t.Errorf("Relationship.Filter[type] = %v, want %q", filterType, "equals")
	}
}

func TestRelatedObject_WithFields(t *testing.T) {
	relatedObj := &RelatedObject{
		ID:        "app-123",
		Name:      "Customers",
		Type:      "App",
		Namespace: "customers",
		Fields: []Field{
			{ID: "field-1", Name: "Title", Type: "text"},
			{ID: "field-2", Name: "Status", Type: "dropdown"},
			{ID: "field-3", Name: "Email", Type: "text"},
			{ID: "field-4", Name: "Created At", Type: "datetime"},
		},
		FieldIDs: []string{"field-1", "field-2", "field-3", "field-4"},
	}

	// Verify fields count
	if len(relatedObj.Fields) != 4 {
		t.Errorf("RelatedObject.Fields length = %d, want 4", len(relatedObj.Fields))
	}

	// Verify FieldIDs count
	if len(relatedObj.FieldIDs) != 4 {
		t.Errorf("RelatedObject.FieldIDs length = %d, want 4", len(relatedObj.FieldIDs))
	}

	// Verify field types
	fieldTypes := make(map[string]string)
	for _, f := range relatedObj.Fields {
		fieldTypes[f.Name] = f.Type
	}

	expectedTypes := map[string]string{
		"Title":      "text",
		"Status":     "dropdown",
		"Email":      "text",
		"Created At": "datetime",
	}

	for name, expectedType := range expectedTypes {
		if got := fieldTypes[name]; got != expectedType {
			t.Errorf("Field %q type = %q, want %q", name, got, expectedType)
		}
	}
}

func TestRelatedObjectMappingsWithSystemFields(t *testing.T) {
	relatedObj := &RelatedObject{
		ID:        "app-123",
		Name:      "Customers",
		Type:      "App",
		Namespace: "customers",
		Fields: []Field{
			{ID: "field-title", Name: "Title", Type: "text"},
			{ID: "field-status", Name: "Status", Type: "dropdown"},
			{ID: "field-id", Name: "ID", Type: "handle"},
			{ID: "field-custom", Name: "Custom Field", Type: "text"},
		},
	}

	mappings := relatedObj.GetUUIDMappings("customers")

	// Title should map to title_field_id
	if got, ok := mappings["field-title"]; !ok {
		t.Error("Title field should be mapped")
	} else if got != "data.elementum_app.customers.title_field_id" {
		t.Errorf("Title field mapping = %q, want title_field_id reference", got)
	}

	// Status should map to status_field_id
	if got, ok := mappings["field-status"]; !ok {
		t.Error("Status field should be mapped")
	} else if got != "data.elementum_app.customers.status_field_id" {
		t.Errorf("Status field mapping = %q, want status_field_id reference", got)
	}

	// ID field should map to id_field_id (note: field.Name is "ID" not the type)
	if got, ok := mappings["field-id"]; !ok {
		t.Error("ID field should be mapped")
	} else if got != "data.elementum_app.customers.id_field_id" {
		t.Errorf("ID field mapping = %q, want id_field_id reference", got)
	}

	// Custom field should NOT be mapped (no easy data source lookup)
	if _, ok := mappings["field-custom"]; ok {
		t.Error("Custom field should not be automatically mapped")
	}
}
