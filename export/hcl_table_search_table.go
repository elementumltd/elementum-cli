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

// TableSearchTableHCLGenerator generates HCL for table search table resources
type TableSearchTableHCLGenerator struct {
	app     *discovery.App
	imports []ImportBlock
	uuidMap map[string]string
}

// NewTableSearchTableHCLGenerator creates a new table search table HCL generator
func NewTableSearchTableHCLGenerator(app *discovery.App, imports []ImportBlock, uuidMap map[string]string) *TableSearchTableHCLGenerator {
	return &TableSearchTableHCLGenerator{
		app:     app,
		imports: imports,
		uuidMap: uuidMap,
	}
}

// GenerateAll generates HCL for all table search tables (from discovered and referenced tables)
func (g *TableSearchTableHCLGenerator) GenerateAll() string {
	return SerializeBlocks(g.GenerateAllIR())
}

// GenerateAllIR generates IR blocks for all table search tables
func (g *TableSearchTableHCLGenerator) GenerateAllIR() []*HCLBlock {
	if g.app == nil {
		return nil
	}

	var blocks []*HCLBlock

	// Search tables from referenced tables
	for _, table := range g.app.ReferencedTables {
		if len(table.SearchTables) == 0 {
			continue
		}

		tableRef := g.resolveTableRef(table)
		for _, searchTable := range table.SearchTables {
			b := g.GenerateSearchTableIR(&searchTable, tableRef, table)
			if b != nil {
				blocks = append(blocks, b)
			}
		}
	}

	// Search tables from discovered tables
	for _, table := range g.app.DiscoveredTables {
		if len(table.SearchTables) == 0 {
			continue
		}

		tableRef := g.resolveTableRef(table)
		for _, searchTable := range table.SearchTables {
			b := g.GenerateSearchTableIR(&searchTable, tableRef, table)
			if b != nil {
				blocks = append(blocks, b)
			}
		}
	}

	return blocks
}

// resolveTableRef resolves the Terraform reference for a table
func (g *TableSearchTableHCLGenerator) resolveTableRef(table *discovery.Table) string {
	if ref, ok := g.uuidMap[table.ID]; ok {
		return strings.TrimSuffix(ref, ".id")
	}
	tableResourceName := SanitizeName(table.Name)
	return "elementum_table." + tableResourceName
}

// GenerateSearchTableIR generates an IR block for a single table search table resource
func (g *TableSearchTableHCLGenerator) GenerateSearchTableIR(searchTable *discovery.TableSearchTable, tableRef string, table *discovery.Table) *HCLBlock {
	// Find the resource name from imports
	var resourceName string
	for _, imp := range g.imports {
		if imp.ResourceType == "elementum_table_search_table" && strings.Contains(imp.ID, searchTable.ID) {
			resourceName = imp.ResourceName
			break
		}
	}
	if resourceName == "" {
		return nil
	}

	b := NewResourceBlock("elementum_table_search_table", resourceName)
	b.SetAttr("table_id", Ref(tableRef+".id"))

	// field_id - resolve using table field lookup
	fieldRef := g.resolveTableFieldID(searchTable.FieldID, table)
	b.SetAttr("field_id", refOrStr(fieldRef))

	// ai_provider_connector_id
	connectorRef := g.resolveAIProviderConnectorID(searchTable.AIProviderConnectorID)
	b.SetAttr("ai_provider_connector_id", refOrStr(connectorRef))

	// attribute_field_ids
	var attrFieldValues []HCLValue
	for _, attrFieldID := range searchTable.AttributeFieldIDs {
		attrFieldRef := g.resolveTableFieldID(attrFieldID, table)
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

// resolveTableFieldID attempts to resolve a table field ID to a Terraform reference
func (g *TableSearchTableHCLGenerator) resolveTableFieldID(fieldID string, table *discovery.Table) string {
	// First check the UUID map
	if ref, ok := g.uuidMap[fieldID]; ok {
		return ref
	}

	// Try to find the field name and use local lookup
	if table != nil {
		for _, f := range table.Fields {
			if f.ID == fieldID {
				tableResourceName := SanitizeName(table.Name)
				return fmt.Sprintf(`local.%s_field_ids_by_name["%s"]`, tableResourceName, f.Name)
			}
		}
	}

	// Fall back to quoted UUID
	return fmt.Sprintf("%q", fieldID)
}

// resolveAIProviderConnectorID attempts to resolve an AI provider connector ID
func (g *TableSearchTableHCLGenerator) resolveAIProviderConnectorID(connectorID string) string {
	if ref, ok := g.uuidMap[connectorID]; ok {
		return ref
	}
	// Fall back to quoted UUID
	return fmt.Sprintf("%q", connectorID)
}
