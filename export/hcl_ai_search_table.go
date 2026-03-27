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

// AISearchTableHCLGenerator generates HCL for AI search table resources
type AISearchTableHCLGenerator struct {
	app     *discovery.App
	imports []ImportBlock
	uuidMap map[string]string
}

// NewAISearchTableHCLGenerator creates a new AI search table HCL generator
func NewAISearchTableHCLGenerator(app *discovery.App, imports []ImportBlock, uuidMap map[string]string) *AISearchTableHCLGenerator {
	return &AISearchTableHCLGenerator{
		app:     app,
		imports: imports,
		uuidMap: uuidMap,
	}
}

// GenerateAll generates HCL for all AI search tables in the app (including discovered elements)
func (g *AISearchTableHCLGenerator) GenerateAll() string {
	return SerializeBlocks(g.GenerateAllIR())
}

// GenerateAllIR generates IR blocks for all AI search tables
func (g *AISearchTableHCLGenerator) GenerateAllIR() []*HCLBlock {
	if g.app == nil {
		return nil
	}

	var blocks []*HCLBlock

	// App-level search tables
	if len(g.app.AISearchTables) > 0 {
		appResourceName := AppResourceName(g.app)
		for _, searchTable := range g.app.AISearchTables {
			b := g.GenerateSearchTableIR(&searchTable, "elementum_app."+appResourceName)
			if b != nil {
				blocks = append(blocks, b)
			}
		}
	}

	// Element-level search tables
	for _, element := range g.app.DiscoveredElements {
		if len(element.AISearchTables) == 0 {
			continue
		}

		var objectRef string
		if ref, ok := g.uuidMap[element.ID]; ok {
			objectRef = strings.TrimSuffix(ref, ".id")
		} else {
			elementResourceName := SanitizeName(element.Namespace)
			if elementResourceName == "" {
				elementResourceName = SanitizeName(element.Name)
			}
			objectRef = "elementum_element." + elementResourceName
		}

		for _, searchTable := range element.AISearchTables {
			b := g.GenerateSearchTableIR(&searchTable, objectRef)
			if b != nil {
				blocks = append(blocks, b)
			}
		}
	}

	return blocks
}

// GenerateSearchTableIR generates an IR block for a single AI search table resource
func (g *AISearchTableHCLGenerator) GenerateSearchTableIR(searchTable *discovery.AISearchTable, objectRef string) *HCLBlock {
	// Find the resource name from imports
	var resourceName string
	for _, imp := range g.imports {
		if imp.ResourceType == "elementum_ai_search_table" && strings.Contains(imp.ID, searchTable.ID) {
			resourceName = imp.ResourceName
			break
		}
	}
	if resourceName == "" {
		return nil
	}

	b := NewResourceBlock("elementum_ai_search_table", resourceName)
	b.SetAttr("object_id", Ref(objectRef+".id"))

	// field_id
	fieldRef := g.resolveFieldID(searchTable.FieldID)
	b.SetAttr("field_id", refOrStr(fieldRef))

	// ai_provider_connector_id
	connectorRef := g.resolveAIProviderConnectorID(searchTable.AIProviderConnectorID, searchTable.AIProviderConnectorName)
	if strings.Contains(connectorRef, " # ") {
		// Has a comment: split ref and comment
		parts := strings.SplitN(connectorRef, " # ", 2)
		b.SetAttrComment("ai_provider_connector_id", refOrStr(parts[0]), "# "+parts[1])
	} else {
		b.SetAttr("ai_provider_connector_id", refOrStr(connectorRef))
	}

	// attribute_field_ids
	var attrFieldValues []HCLValue
	for _, attrFieldID := range searchTable.AttributeFieldIDs {
		attrFieldRef := g.resolveFieldID(attrFieldID)
		attrFieldValues = append(attrFieldValues, refOrStr(attrFieldRef))
	}
	b.SetAttr("attribute_field_ids", HCLList{Values: attrFieldValues})

	b.SetAttr("duration", Str(searchTable.Duration))
	b.SetAttr("target_lag", Num(float64(searchTable.TargetLag)))

	if searchTable.Warehouse != "" {
		b.SetAttr("warehouse", Str(searchTable.Warehouse))
	}

	return b
}

// resolveFieldID attempts to resolve a field ID to a Terraform reference
func (g *AISearchTableHCLGenerator) resolveFieldID(fieldID string) string {
	if ref, ok := g.uuidMap[fieldID]; ok {
		return ref
	}
	// Fall back to quoted UUID
	return fmt.Sprintf("%q", fieldID)
}

// resolveAIProviderConnectorID attempts to resolve an AI provider connector ID
// Since these are typically data sources, we add a comment with the model name
func (g *AISearchTableHCLGenerator) resolveAIProviderConnectorID(connectorID, modelName string) string {
	if ref, ok := g.uuidMap[connectorID]; ok {
		return ref
	}
	// Fall back to quoted UUID with a comment about the model name
	if modelName != "" {
		return fmt.Sprintf("%q # %s", connectorID, modelName)
	}
	return fmt.Sprintf("%q", connectorID)
}
