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
	"fmt"
	"strings"

	"github.com/elementumltd/elementum-cli/discovery"
)

// RelationshipHCLGenerator generates HCL for relationship resources
type RelationshipHCLGenerator struct {
	app     *discovery.App
	imports []ImportBlock
	uuidMap map[string]string
}

// NewRelationshipHCLGenerator creates a new relationship HCL generator
func NewRelationshipHCLGenerator(app *discovery.App, imports []ImportBlock, uuidMap map[string]string) *RelationshipHCLGenerator {
	return &RelationshipHCLGenerator{
		app:     app,
		imports: imports,
		uuidMap: uuidMap,
	}
}

// GenerateAll generates HCL for all relationships in the app
func (g *RelationshipHCLGenerator) GenerateAll() string {
	return SerializeBlocks(g.GenerateAllIR())
}

// GenerateAllIR generates IR blocks for all relationships in the app
func (g *RelationshipHCLGenerator) GenerateAllIR() []*HCLBlock {
	if g.app == nil || len(g.app.Relationships) == 0 {
		return nil
	}

	var blocks []*HCLBlock
	for _, rel := range g.app.Relationships {
		b := g.GenerateRelationshipIR(&rel)
		if b != nil {
			blocks = append(blocks, b)
		}
	}

	return blocks
}

// GenerateRelationshipIR generates an IR block for a single relationship resource
func (g *RelationshipHCLGenerator) GenerateRelationshipIR(rel *discovery.Relationship) *HCLBlock {
	// Find the resource name from imports
	var resourceName string
	for _, imp := range g.imports {
		if imp.ResourceType == "elementum_relationship" && strings.Contains(imp.ID, rel.ID) {
			resourceName = imp.ResourceName
			break
		}
	}
	if resourceName == "" {
		resourceName = g.generateRelationshipName(rel)
	}

	b := NewResourceBlock("elementum_relationship", resourceName)

	// object_id
	objectRef := g.resolveObjectRef(g.app.ID)
	b.SetAttr("object_id", refOrStr(objectRef))

	// related_object_id
	relatedRef := g.resolveObjectRef(rel.RelatedObjectID)
	b.SetAttr("related_object_id", refOrStr(relatedRef))

	// columns
	if len(rel.Columns) > 0 {
		var colObjs []HCLValue
		for _, col := range rel.Columns {
			fieldRef := g.resolveFieldRef(col.FieldID)
			relatedFieldRef := g.resolveFieldRef(col.RelatedFieldID)
			colObjs = append(colObjs, Obj(
				Attr("field_id", refOrStr(fieldRef)),
				Attr("related_field_id", refOrStr(relatedFieldRef)),
			))
		}
		b.SetAttr("columns", HCLList{Values: colObjs})
	}

	// Filter
	if len(rel.Filter) > 0 {
		filterVal := GenerateFilterIR(rel.Filter, g.uuidMap)
		if filterVal != nil {
			b.SetAttr("filter", filterVal)
		}
	}

	return b
}

// generateRelationshipName generates a resource name for a relationship
func (g *RelationshipHCLGenerator) generateRelationshipName(rel *discovery.Relationship) string {
	prefix := SanitizeName(g.app.Namespace)
	if prefix == "" {
		prefix = SanitizeName(g.app.Name)
	}

	if rel.RelatedObjectName != "" {
		return prefix + "_to_" + SanitizeName(rel.RelatedObjectName)
	}
	return prefix + "_relationship"
}

// resolveObjectRef resolves an object ID to a terraform reference
func (g *RelationshipHCLGenerator) resolveObjectRef(objectID string) string {
	// Check if it's the same app
	if g.app != nil && objectID == g.app.ID {
		return "elementum_app." + AppResourceName(g.app) + ".id"
	}

	// Check UUID map
	if ref, ok := g.uuidMap[objectID]; ok {
		return ref
	}

	// Check related objects
	if g.app != nil {
		for _, related := range g.app.RelatedObjects {
			if related.ID == objectID {
				dataType := "data.elementum_app"
				if related.Type == "Element" {
					dataType = "data.elementum_element"
				}
				return dataType + "." + SanitizeName(related.Name) + ".id"
			}
		}
	}

	// Return quoted ID as fallback
	return fmt.Sprintf("%q", objectID)
}

// resolveFieldRef resolves a field ID to a terraform reference
func (g *RelationshipHCLGenerator) resolveFieldRef(fieldID string) string {
	// Check UUID map first
	if ref, ok := g.uuidMap[fieldID]; ok {
		return ref
	}

	// Check app fields
	if g.app != nil {
		for _, field := range g.app.Fields {
			if field.ID == fieldID {
				resourceType := "elementum_" + field.Type + "_field"
				resourceName := SanitizeName(field.Name)
				return resourceType + "." + resourceName + ".id"
			}
		}
	}

	// Check imports
	for _, imp := range g.imports {
		if strings.Contains(imp.ID, fieldID) && strings.HasSuffix(imp.ResourceType, "_field") {
			return imp.ResourceType + "." + imp.ResourceName + ".id"
		}
	}

	return fmt.Sprintf("%q", fieldID)
}

// GenerateRelationshipHCLStandalone generates HCL for all relationships as a standalone function
func GenerateRelationshipHCLStandalone(app *discovery.App, imports []ImportBlock) string {
	if app == nil || len(app.Relationships) == 0 {
		return ""
	}

	uuidMap := buildUUIDMap(imports, app)
	gen := NewRelationshipHCLGenerator(app, imports, uuidMap)
	return SerializeBlocks(gen.GenerateAllIR())
}
