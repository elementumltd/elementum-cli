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

package export

import (
	"strings"
	"testing"

	"github.com/elementumltd/elementum-cli/discovery"
)

func TestRelationshipHCLGenerator_GenerateAll(t *testing.T) {
	app := &discovery.App{
		ID:        "app-123",
		Name:      "Orders",
		Namespace: "orders",
		Relationships: []discovery.Relationship{
			{
				ID:                "rel-1",
				RelatedObjectID:   "app-456",
				RelatedObjectType: "App",
				RelatedObjectName: "Customers",
				Columns: []discovery.RelationshipColumn{
					{
						FieldID:          "field-cust-ref",
						FieldName:        "Customer",
						RelatedFieldID:   "field-cust-id",
						RelatedFieldName: "ID",
					},
				},
			},
		},
		RelatedObjects: []discovery.RelatedObject{
			{
				ID:   "app-456",
				Name: "Customers",
				Type: "App",
			},
		},
	}

	imports := []ImportBlock{
		{ResourceType: "elementum_app", ResourceName: "orders", ID: "app-123"},
		{ResourceType: "elementum_relationship", ResourceName: "orders_to_customers", ID: "app-123:rel-1"},
	}

	uuidMap := map[string]string{
		"field-cust-ref": "elementum_relation_field.customer.id",
		"field-cust-id":  "data.elementum_app.customers.title_field_id",
	}

	gen := NewRelationshipHCLGenerator(app, imports, uuidMap)
	hcl := gen.GenerateAll()

	// Verify resource block
	if !strings.Contains(hcl, `resource "elementum_relationship" "orders_to_customers"`) {
		t.Errorf("Generated HCL should contain relationship resource, got:\n%s", hcl)
	}

	// Verify object_id
	if !strings.Contains(hcl, "object_id = elementum_app.orders.id") {
		t.Errorf("Generated HCL should contain object_id, got:\n%s", hcl)
	}

	// Verify related_object_id (should be beautified from RelatedObjects)
	if !strings.Contains(hcl, "related_object_id = data.elementum_app.customers.id") {
		t.Errorf("Generated HCL should contain related_object_id, got:\n%s", hcl)
	}

	// Verify columns
	if !strings.Contains(hcl, "columns = [") {
		t.Errorf("Generated HCL should contain columns, got:\n%s", hcl)
	}

	// Verify field_id is beautified
	if !strings.Contains(hcl, "field_id = elementum_relation_field.customer.id") {
		t.Errorf("Generated HCL should beautify field_id, got:\n%s", hcl)
	}

	// Verify related_field_id is beautified
	if !strings.Contains(hcl, "related_field_id = data.elementum_app.customers.title_field_id") {
		t.Errorf("Generated HCL should beautify related_field_id, got:\n%s", hcl)
	}
}

func TestRelationshipHCLGenerator_WithFilter(t *testing.T) {
	app := &discovery.App{
		ID:        "app-123",
		Name:      "Orders",
		Namespace: "orders",
		Relationships: []discovery.Relationship{
			{
				ID:                "rel-1",
				RelatedObjectID:   "app-456",
				RelatedObjectType: "App",
				RelatedObjectName: "Customers",
				Columns: []discovery.RelationshipColumn{
					{
						FieldID:        "field-cust-ref",
						RelatedFieldID: "field-cust-id",
					},
				},
				Filter: map[string]interface{}{
					"type":    "EQUALS",
					"fieldId": "field-status",
					"value": map[string]interface{}{
						"literal": "active",
					},
				},
			},
		},
	}

	imports := []ImportBlock{
		{ResourceType: "elementum_relationship", ResourceName: "orders_to_customers", ID: "app-123:rel-1"},
	}

	uuidMap := map[string]string{
		"field-status": "elementum_picklist_field.status.id",
	}

	gen := NewRelationshipHCLGenerator(app, imports, uuidMap)
	hcl := gen.GenerateAll()

	// Verify filter is generated with attribute syntax
	if !strings.Contains(hcl, "filter = {") {
		t.Errorf("Generated HCL should contain filter, got:\n%s", hcl)
	}

	// Verify filter type
	if !strings.Contains(hcl, `type = "equals"`) {
		t.Errorf("Generated HCL should contain filter type, got:\n%s", hcl)
	}

	// Verify field_id in filter is beautified
	if !strings.Contains(hcl, "field_id = elementum_picklist_field.status.id") {
		t.Errorf("Generated HCL should beautify filter field_id, got:\n%s", hcl)
	}
}

func TestRelationshipHCLGenerator_ComplexFilter(t *testing.T) {
	app := &discovery.App{
		ID:        "app-123",
		Name:      "Orders",
		Namespace: "orders",
		Relationships: []discovery.Relationship{
			{
				ID:                "rel-1",
				RelatedObjectID:   "app-456",
				RelatedObjectType: "App",
				RelatedObjectName: "Customers",
				Columns: []discovery.RelationshipColumn{
					{
						FieldID:        "field-cust-ref",
						RelatedFieldID: "field-cust-id",
					},
				},
				Filter: map[string]interface{}{
					"type": "AND",
					"children": []interface{}{
						map[string]interface{}{
							"type":    "EQUALS",
							"fieldId": "field-status",
							"value":   map[string]interface{}{"literal": "active"},
						},
						map[string]interface{}{
							"type":    "GREATER_THAN",
							"fieldId": "field-amount",
							"value":   map[string]interface{}{"literal": float64(100)},
						},
					},
				},
			},
		},
	}

	imports := []ImportBlock{}
	uuidMap := map[string]string{}

	gen := NewRelationshipHCLGenerator(app, imports, uuidMap)
	hcl := gen.GenerateAll()

	// Verify AND filter
	if !strings.Contains(hcl, `type = "and"`) {
		t.Errorf("Generated HCL should contain AND filter, got:\n%s", hcl)
	}

	// Verify children array
	if !strings.Contains(hcl, "children = [") {
		t.Errorf("Generated HCL should contain children, got:\n%s", hcl)
	}

	// Verify child filter types
	if !strings.Contains(hcl, `type = "equals"`) {
		t.Errorf("Generated HCL should contain equals filter, got:\n%s", hcl)
	}
	if !strings.Contains(hcl, `type = "greater_than"`) {
		t.Errorf("Generated HCL should contain greater_than filter, got:\n%s", hcl)
	}
}

func TestRelationshipHCLGenerator_NoFilter(t *testing.T) {
	app := &discovery.App{
		ID:        "app-123",
		Name:      "Orders",
		Namespace: "orders",
		Relationships: []discovery.Relationship{
			{
				ID:                "rel-1",
				RelatedObjectID:   "app-456",
				RelatedObjectType: "App",
				RelatedObjectName: "Customers",
				Columns: []discovery.RelationshipColumn{
					{
						FieldID:        "field-cust-ref",
						RelatedFieldID: "field-cust-id",
					},
				},
				Filter: nil, // No filter
			},
		},
	}

	imports := []ImportBlock{}
	uuidMap := map[string]string{}

	gen := NewRelationshipHCLGenerator(app, imports, uuidMap)
	hcl := gen.GenerateAll()

	// Verify filter is NOT present
	if strings.Contains(hcl, "filter") {
		t.Errorf("Generated HCL should NOT contain filter when nil, got:\n%s", hcl)
	}
}

func TestRelationshipHCLGenerator_NoRelationships(t *testing.T) {
	app := &discovery.App{
		ID:            "app-123",
		Name:          "Orders",
		Namespace:     "orders",
		Relationships: []discovery.Relationship{},
	}

	imports := []ImportBlock{}
	uuidMap := map[string]string{}

	gen := NewRelationshipHCLGenerator(app, imports, uuidMap)
	hcl := gen.GenerateAll()

	if hcl != "" {
		t.Errorf("Expected empty string for app with no relationships, got:\n%s", hcl)
	}
}

func TestRelationshipHCLGenerator_NilApp(t *testing.T) {
	gen := &RelationshipHCLGenerator{
		app:     nil,
		imports: []ImportBlock{},
		uuidMap: map[string]string{},
	}
	hcl := gen.GenerateAll()

	if hcl != "" {
		t.Errorf("Expected empty string for nil app, got:\n%s", hcl)
	}
}

func TestRelationshipHCLGenerator_MultipleColumns(t *testing.T) {
	app := &discovery.App{
		ID:        "app-123",
		Name:      "Orders",
		Namespace: "orders",
		Relationships: []discovery.Relationship{
			{
				ID:                "rel-1",
				RelatedObjectID:   "app-456",
				RelatedObjectType: "App",
				RelatedObjectName: "Customers",
				Columns: []discovery.RelationshipColumn{
					{
						FieldID:        "field-cust-ref",
						RelatedFieldID: "field-cust-id",
					},
					{
						FieldID:        "field-region",
						RelatedFieldID: "field-cust-region",
					},
				},
			},
		},
	}

	imports := []ImportBlock{}
	uuidMap := map[string]string{}

	gen := NewRelationshipHCLGenerator(app, imports, uuidMap)
	hcl := gen.GenerateAll()

	// Count column entries (field_id = also matches as substring of related_field_id =)
	columnCount := strings.Count(hcl, "field_id =")
	if columnCount != 4 {
		t.Errorf("Expected 4 field_id= occurrences (2 field_id + 2 related_field_id), got %d in:\n%s", columnCount, hcl)
	}
}

func TestGenerateRelationshipHCLStandalone(t *testing.T) {
	app := &discovery.App{
		ID:        "app-123",
		Name:      "Orders",
		Namespace: "orders",
		Relationships: []discovery.Relationship{
			{
				ID:                "rel-1",
				RelatedObjectID:   "app-456",
				RelatedObjectType: "App",
				RelatedObjectName: "Customers",
				Columns: []discovery.RelationshipColumn{
					{FieldID: "field-1", RelatedFieldID: "field-2"},
				},
			},
		},
	}

	imports := []ImportBlock{
		{ResourceType: "elementum_app", ResourceName: "orders", ID: "app-123"},
	}

	hcl := GenerateRelationshipHCLStandalone(app, imports)

	if !strings.Contains(hcl, "elementum_relationship") {
		t.Error("Standalone generator should produce relationship HCL")
	}
}

func TestGenerateRelationshipHCLStandalone_NilApp(t *testing.T) {
	hcl := GenerateRelationshipHCLStandalone(nil, []ImportBlock{})
	if hcl != "" {
		t.Error("Should return empty string for nil app")
	}
}

func TestRelationshipHCLGenerator_GenerateRelationshipName(t *testing.T) {
	app := &discovery.App{
		ID:        "app-123",
		Name:      "Orders",
		Namespace: "orders",
	}

	gen := NewRelationshipHCLGenerator(app, []ImportBlock{}, map[string]string{})

	// Test with related object name
	rel := &discovery.Relationship{
		ID:                "rel-1",
		RelatedObjectName: "Customers",
	}
	name := gen.generateRelationshipName(rel)
	if name != "orders_to_customers" {
		t.Errorf("Expected 'orders_to_customers', got %q", name)
	}

	// Test without related object name
	rel2 := &discovery.Relationship{
		ID:                "rel-2",
		RelatedObjectName: "",
	}
	name2 := gen.generateRelationshipName(rel2)
	if name2 != "orders_relationship" {
		t.Errorf("Expected 'orders_relationship', got %q", name2)
	}
}
