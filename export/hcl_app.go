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

// AppHCLGenerator generates HCL IR blocks for the app resource, fields, layouts, flows,
// and views. This replaces the `tofu plan -generate-config-out` path for these resource types.
type AppHCLGenerator struct {
	app     *discovery.App
	imports []ImportBlock
	uuidMap map[string]string
}

// NewAppHCLGenerator creates a new AppHCLGenerator
func NewAppHCLGenerator(app *discovery.App, imports []ImportBlock, uuidMap map[string]string) *AppHCLGenerator {
	return &AppHCLGenerator{
		app:     app,
		imports: imports,
		uuidMap: uuidMap,
	}
}

// GenerateAllIR generates IR blocks for the app, its fields, layouts, flows, and views.
func (g *AppHCLGenerator) GenerateAllIR() []*HCLBlock {
	var blocks []*HCLBlock

	// App resource
	if b := g.generateAppIR(g.app); b != nil {
		blocks = append(blocks, b)
	}

	// Fields
	blocks = append(blocks, g.generateFieldsIR(g.app.ID, g.app.Fields, false)...)

	// Layouts
	blocks = append(blocks, g.generateLayoutsIR(g.app.ID, g.app.Layouts)...)

	// Flows
	blocks = append(blocks, g.generateFlowsIR(g.app.ID, g.app.Flows)...)

	// Views
	blocks = append(blocks, g.generateViewsIR(g.app.ID, g.app.Views)...)

	// Discovered apps
	for _, discoveredApp := range g.app.DiscoveredApps {
		if b := g.generateAppIR(discoveredApp); b != nil {
			blocks = append(blocks, b)
		}
		blocks = append(blocks, g.generateFieldsIR(discoveredApp.ID, discoveredApp.Fields, false)...)
		blocks = append(blocks, g.generateLayoutsIR(discoveredApp.ID, discoveredApp.Layouts)...)
		blocks = append(blocks, g.generateFlowsIR(discoveredApp.ID, discoveredApp.Flows)...)
		blocks = append(blocks, g.generateViewsIR(discoveredApp.ID, discoveredApp.Views)...)
	}

	// Discovered elements
	for _, elem := range g.app.DiscoveredElements {
		if b := g.generateElementIR(elem); b != nil {
			blocks = append(blocks, b)
		}
		blocks = append(blocks, g.generateFieldsIR(elem.ID, elem.Fields, true)...)
		blocks = append(blocks, g.generateLayoutsIR(elem.ID, elem.Layouts)...)
		blocks = append(blocks, g.generateFlowsIR(elem.ID, elem.Flows)...)
		blocks = append(blocks, g.generateViewsIR(elem.ID, elem.Views)...)
	}

	// Discovered tasks (aspect tasks, not workflow tasks)
	for _, task := range g.app.DiscoveredTasks {
		if b := g.generateAspectTaskIR(task); b != nil {
			blocks = append(blocks, b)
		}
		blocks = append(blocks, g.generateFieldsIR(task.ID, task.Fields, true)...)
		blocks = append(blocks, g.generateLayoutsIR(task.ID, task.Layouts)...)
		blocks = append(blocks, g.generateViewsIR(task.ID, task.Views)...)
	}

	// Discovered tables
	for _, table := range g.app.DiscoveredTables {
		if b := g.generateTableIR(table); b != nil {
			blocks = append(blocks, b)
		}
	}
	for _, table := range g.app.ReferencedTables {
		if b := g.generateTableIR(table); b != nil {
			blocks = append(blocks, b)
		}
	}

	// Discovered datamines
	for _, dm := range g.app.DiscoveredDatamines {
		if b := g.generateDatamineIR(dm); b != nil {
			blocks = append(blocks, b)
		}
	}
	for _, dm := range g.app.ReferencedDatamines {
		if b := g.generateDatamineIR(dm); b != nil {
			blocks = append(blocks, b)
		}
	}

	return blocks
}

// CanGenerate returns true if there is an app to generate for.
func (g *AppHCLGenerator) CanGenerate() bool {
	return g.app != nil
}

// Name returns the generator name for logging.
func (g *AppHCLGenerator) Name() string { return "app_resources" }

// --- App ---

func (g *AppHCLGenerator) generateAppIR(app *discovery.App) *HCLBlock {
	resourceName := g.findResourceName("elementum_app", app.ID)
	if resourceName == "" {
		resourceName = AppResourceName(app)
	}

	b := NewResourceBlock("elementum_app", resourceName)

	b.SetAttr("name", Str(app.Name))
	if app.Namespace != "" {
		b.SetAttr("namespace", Str(app.Namespace))
	}

	// category_id
	if ref := g.resolveRef(app.CategoryID); ref != "" {
		b.SetAttr("category_id", refOrStr(ref))
	} else if app.CategoryName != "" {
		b.SetAttr("category_id", Ref(fmt.Sprintf("data.elementum_category.%s.id", SanitizeName(app.CategoryName))))
	}

	// cloud_link_id
	if app.CloudLinkID != "" {
		if ref := g.resolveRef(app.CloudLinkID); ref != "" {
			b.SetAttr("cloud_link_id", refOrStr(ref))
		} else if app.CloudLinkName != "" {
			b.SetAttr("cloud_link_id", Ref(fmt.Sprintf("data.elementum_cloudlink.%s.id", SanitizeName(app.CloudLinkName))))
		} else {
			b.SetAttr("cloud_link_id", Str(app.CloudLinkID))
		}
	}

	// handle (from status field semantic tag or namespace-based)
	handleValue := strings.ToUpper(app.Namespace)
	if handleValue == "" {
		handleValue = strings.ToUpper(SanitizeName(app.Name))
	}
	b.SetAttr("handle", Str(handleValue))

	// Status options from the status field (required attribute)
	statusOptions := g.findStatusOptions(app)
	if len(statusOptions) > 0 {
		var optionValues []HCLValue
		for _, opt := range statusOptions {
			attrs := []*HCLAttribute{
				Attr("label", Str(opt.Label)),
			}
			if opt.Color != "" {
				attrs = append(attrs, Attr("color", Str(opt.Color)))
			}
			if len(opt.Tags) == 0 {
				attrs = append(attrs, Attr("tags", List()))
			} else {
				var tagVals []HCLValue
				for _, tag := range opt.Tags {
					tagVals = append(tagVals, Str(tag))
				}
				attrs = append(attrs, Attr("tags", HCLList{Values: tagVals}))
			}
			optionValues = append(optionValues, Obj(attrs...))
		}
		b.SetAttr("status_options", HCLList{Values: optionValues})
	} else {
		// status_options is required - generate default options when none found
		defaultOptions := []HCLValue{
			Obj(
				Attr("label", Str("Open")),
				Attr("tags", List()),
			),
			Obj(
				Attr("label", Str("Closed")),
				Attr("tags", HCLList{Values: []HCLValue{Str("CLOSED")}}),
			),
		}
		b.SetAttrComment("status_options", HCLList{Values: defaultOptions}, "# TODO: Review and customize status options")
	}

	// Stages from layouts
	stages := g.buildStages(app)
	if len(stages) > 0 {
		b.SetAttr("stages", HCLList{Values: stages})
	}

	// Comments enabled
	if app.CommentsEnabled {
		b.SetAttr("comments_enabled", Bool(true))
	}

	// Lock stages
	if app.LockStages {
		b.SetAttr("lock_stages", Bool(true))
	}

	return b
}

func (g *AppHCLGenerator) findStatusOptions(app *discovery.App) []discovery.FieldOption {
	for _, field := range app.Fields {
		for _, tag := range field.SemanticTags {
			if tag == "status" || tag == "STATUS" {
				return field.Options
			}
		}
	}
	return nil
}

func (g *AppHCLGenerator) buildStages(app *discovery.App) []HCLValue {
	var stages []HCLValue
	for _, layout := range app.Layouts {
		if layout.IsInitiate {
			continue
		}
		key := strings.ToLower(SanitizeName(layout.Name))
		attrs := []*HCLAttribute{
			Attr("key", Str(key)),
			Attr("name", Str(layout.Name)),
		}
		stages = append(stages, Obj(attrs...))
	}
	return stages
}

// --- Element ---

func (g *AppHCLGenerator) generateElementIR(elem *discovery.Element) *HCLBlock {
	resourceName := g.findResourceName("elementum_element", elem.ID)
	if resourceName == "" {
		resourceName = ElementResourceName(elem)
	}

	b := NewResourceBlock("elementum_element", resourceName)

	b.SetAttr("name", Str(elem.Name))
	if elem.Namespace != "" {
		b.SetAttr("namespace", Str(elem.Namespace))
	}
	if elem.Handle != "" {
		b.SetAttr("handle", Str(elem.Handle))
	}

	// category_id
	if elem.CategoryID != "" {
		if ref := g.resolveRef(elem.CategoryID); ref != "" {
			b.SetAttr("category_id", refOrStr(ref))
		} else if elem.CategoryName != "" {
			b.SetAttr("category_id", Ref(fmt.Sprintf("data.elementum_category.%s.id", SanitizeName(elem.CategoryName))))
		} else {
			b.SetAttr("category_id", Str(elem.CategoryID))
		}
	}

	// cloud_link_id
	if elem.CloudLinkID != "" {
		if ref := g.resolveRef(elem.CloudLinkID); ref != "" {
			b.SetAttr("cloud_link_id", refOrStr(ref))
		} else if elem.CloudLinkName != "" {
			b.SetAttr("cloud_link_id", Ref(fmt.Sprintf("data.elementum_cloudlink.%s.id", SanitizeName(elem.CloudLinkName))))
		} else {
			b.SetAttr("cloud_link_id", Str(elem.CloudLinkID))
		}
	}

	if elem.Description != "" {
		b.SetAttr("description", Str(elem.Description))
	}
	if elem.Icon != "" {
		b.SetAttr("icon", Str(elem.Icon))
	}
	if elem.Color != "" {
		b.SetAttr("color", Str(elem.Color))
	}
	if elem.CommentsEnabled {
		b.SetAttr("comments_enabled", Bool(true))
	}

	return b
}

// --- AspectTask (object type, not workflow task) ---

func (g *AppHCLGenerator) generateAspectTaskIR(task *discovery.AspectTask) *HCLBlock {
	resourceName := g.findResourceName("elementum_task", task.ID)
	if resourceName == "" {
		resourceName = TaskResourceName(task)
	}

	b := NewResourceBlock("elementum_task", resourceName)

	b.SetAttr("name", Str(task.Name))
	if task.Namespace != "" {
		b.SetAttr("namespace", Str(task.Namespace))
	}
	if task.Handle != "" {
		b.SetAttr("handle", Str(task.Handle))
	}

	// category_id
	if task.CategoryID != "" {
		if ref := g.resolveRef(task.CategoryID); ref != "" {
			b.SetAttr("category_id", refOrStr(ref))
		} else if task.CategoryName != "" {
			b.SetAttr("category_id", Ref(fmt.Sprintf("data.elementum_category.%s.id", SanitizeName(task.CategoryName))))
		} else {
			b.SetAttr("category_id", Str(task.CategoryID))
		}
	}

	// cloud_link_id
	if task.CloudLinkID != "" {
		if ref := g.resolveRef(task.CloudLinkID); ref != "" {
			b.SetAttr("cloud_link_id", refOrStr(ref))
		} else if task.CloudLinkName != "" {
			b.SetAttr("cloud_link_id", Ref(fmt.Sprintf("data.elementum_cloudlink.%s.id", SanitizeName(task.CloudLinkName))))
		} else {
			b.SetAttr("cloud_link_id", Str(task.CloudLinkID))
		}
	}

	if task.Description != "" {
		b.SetAttr("description", Str(task.Description))
	}
	if task.Icon != "" {
		b.SetAttr("icon", Str(task.Icon))
	}
	if task.Color != "" {
		b.SetAttr("color", Str(task.Color))
	}
	if task.CommentsEnabled {
		b.SetAttr("comments_enabled", Bool(true))
	}

	return b
}

// --- Fields ---

func (g *AppHCLGenerator) generateFieldsIR(objectID string, fields []discovery.Field, allowStatus bool) []*HCLBlock {
	var blocks []*HCLBlock
	for _, field := range fields {
		if field.Type == "unknown" || field.Type == "" || field.Type == "handle" {
			continue
		}
		if isSystemField(field, allowStatus) {
			continue
		}

		b := g.generateFieldIR(objectID, &field)
		if b != nil {
			blocks = append(blocks, b)
		}
	}
	return blocks
}

func (g *AppHCLGenerator) generateFieldIR(objectID string, field *discovery.Field) *HCLBlock {
	resourceType := GetResourceTypeForField(*field)
	resourceName := g.findResourceNameForField(resourceType, field.ID)
	if resourceName == "" {
		resourceName = SanitizeName(field.Name)
	}

	b := NewResourceBlock(resourceType, resourceName)

	// object_id
	if ref := g.resolveRef(objectID); ref != "" {
		b.SetAttr("object_id", refOrStr(ref))
	} else {
		b.SetAttr("object_id", Str(objectID))
	}

	b.SetAttr("name", Str(field.Name))

	if field.Required {
		b.SetAttr("required", Bool(true))
	}

	// Dropdown/multiselect options
	if (field.Type == "dropdown" || field.Type == "multiselect") && len(field.Options) > 0 {
		var optionValues []HCLValue
		for _, opt := range field.Options {
			attrs := []*HCLAttribute{
				Attr("label", Str(opt.Label)),
			}
			if opt.Color != "" {
				attrs = append(attrs, Attr("color", Str(opt.Color)))
			}
			optionValues = append(optionValues, Obj(attrs...))
		}
		b.SetAttr("options", HCLList{Values: optionValues})
	}

	// Calculated field formula
	if field.Type == "calculated" && field.Calculation != "" {
		b.SetAttr("calculation", Str(field.Calculation))
	}

	return b
}

func (g *AppHCLGenerator) findResourceNameForField(resourceType, fieldID string) string {
	for _, imp := range g.imports {
		if imp.ResourceType == resourceType && strings.Contains(imp.ID, fieldID) {
			return imp.ResourceName
		}
	}
	return ""
}

// --- Layouts ---

func (g *AppHCLGenerator) generateLayoutsIR(objectID string, layouts []discovery.Layout) []*HCLBlock {
	var blocks []*HCLBlock
	for _, layout := range layouts {
		if layout.IsInitiate {
			continue
		}
		b := g.generateLayoutIR(objectID, &layout)
		if b != nil {
			blocks = append(blocks, b)
		}
	}
	return blocks
}

func (g *AppHCLGenerator) generateLayoutIR(objectID string, layout *discovery.Layout) *HCLBlock {
	resourceName := g.findResourceName("elementum_layout", layout.ID)
	if resourceName == "" {
		resourceName = SanitizeName(layout.Name)
	}

	b := NewResourceBlock("elementum_layout", resourceName)

	// object_id
	if ref := g.resolveRef(objectID); ref != "" {
		b.SetAttr("object_id", refOrStr(ref))
	} else {
		b.SetAttr("object_id", Str(objectID))
	}

	// stage_id - the layout ID is the stage ID
	if ref := g.resolveRef(layout.ID); ref != "" {
		b.SetAttr("stage_id", refOrStr(ref))
	} else {
		b.SetAttr("stage_id", Str(layout.ID))
	}

	// sections from display blocks
	if len(layout.DisplayBlocks) > 0 {
		var sections []HCLValue
		for _, block := range layout.DisplayBlocks {
			section := g.buildSection(&block)
			sections = append(sections, section)
		}
		b.SetAttr("sections", HCLList{Values: sections})
	}

	return b
}

func (g *AppHCLGenerator) buildSection(block *discovery.DisplayBlock) HCLValue {
	var attrs []*HCLAttribute

	blockType := block.Type
	if blockType == "" {
		blockType = "group"
	}

	attrs = append(attrs, Attr("type", Str(blockType)))

	if block.Name != "" {
		attrs = append(attrs, Attr("name", Str(block.Name)))
	}

	// display_location for all block types except surveys
	if blockType != "surveys" {
		loc := block.DisplayLocation
		if loc == "" {
			loc = "CENTER"
		}
		attrs = append(attrs, Attr("display_location", Str(loc)))
	}

	attrs = append(attrs, Attr("display_order", Num(float64(block.DisplayOrder))))

	if !block.SideNavItem {
		attrs = append(attrs, Attr("side_nav_item", Bool(false)))
	}

	// Group-specific attributes
	if blockType == "group" {
		if block.Icon != "" {
			attrs = append(attrs, Attr("icon", Str(block.Icon)))
		}
		if block.Color != "" {
			attrs = append(attrs, Attr("color", Str(block.Color)))
		}

		// field_ids: resolve references and filter out system fields
		if len(block.FieldIDs) > 0 {
			systemFieldIDs := g.app.GetSystemFieldIDs()
			var fieldValues []HCLValue
			for _, fieldID := range block.FieldIDs {
				// Skip system fields (HANDLE, TITLE, STATUS, STAGE)
				if systemFieldIDs[fieldID] {
					continue
				}
				if ref := g.resolveRef(fieldID); ref != "" {
					fieldValues = append(fieldValues, refOrStr(ref))
				} else {
					fieldValues = append(fieldValues, Str(fieldID))
				}
			}
			if len(fieldValues) > 0 {
				attrs = append(attrs, Attr("field_ids", HCLList{Values: fieldValues}))
			}
		}
	}

	return Obj(attrs...)
}

// --- Flows ---

func (g *AppHCLGenerator) generateFlowsIR(objectID string, flows []discovery.Flow) []*HCLBlock {
	var blocks []*HCLBlock
	for _, flow := range flows {
		b := g.generateFlowIR(objectID, &flow)
		if b != nil {
			blocks = append(blocks, b)
		}
	}
	return blocks
}

func (g *AppHCLGenerator) generateFlowIR(objectID string, flow *discovery.Flow) *HCLBlock {
	resourceName := g.findResourceName("elementum_flow", flow.ID)
	if resourceName == "" {
		resourceName = SanitizeName(flow.Name)
	}

	b := NewResourceBlock("elementum_flow", resourceName)

	// object_id
	if ref := g.resolveRef(objectID); ref != "" {
		b.SetAttr("object_id", refOrStr(ref))
	} else {
		b.SetAttr("object_id", Str(objectID))
	}

	// Flow discovery only gives us ID and Name - detailed stage/node data
	// comes from the tofu import. Since we're removing tofu, flows will get
	// the object_id set and stages will be populated by the import mechanism.
	// The full flow configuration requires GetFlowConfiguration which discovery
	// doesn't currently call.

	return b
}

// --- Views ---

func (g *AppHCLGenerator) generateViewsIR(objectID string, views []discovery.View) []*HCLBlock {
	var blocks []*HCLBlock
	for _, view := range views {
		b := g.generateViewIR(objectID, &view)
		if b != nil {
			blocks = append(blocks, b)
		}
	}
	return blocks
}

func (g *AppHCLGenerator) generateViewIR(objectID string, view *discovery.View) *HCLBlock {
	resourceType := view.TerraformResourceType()
	if resourceType == "" {
		return nil
	}

	resourceName := g.findResourceName(resourceType, view.ID)
	if resourceName == "" {
		resourceName = SanitizeName(view.Name)
	}

	b := NewResourceBlock(resourceType, resourceName)

	// aspect_id
	if ref := g.resolveRef(objectID); ref != "" {
		b.SetAttr("aspect_id", refOrStr(ref))
	} else {
		b.SetAttr("aspect_id", Str(objectID))
	}

	b.SetAttr("name", Str(view.Name))

	if view.AllowRecordCreation {
		b.SetAttr("allow_record_creation", Bool(true))
	}

	// Columns
	if len(view.Columns) > 0 {
		var colVals []HCLValue
		for _, col := range view.Columns {
			if ref := g.resolveRef(col); ref != "" {
				colVals = append(colVals, refOrStr(ref))
			} else {
				colVals = append(colVals, Str(col))
			}
		}
		b.SetAttr("columns", HCLList{Values: colVals})
	}

	// Type-specific attributes
	switch resourceType {
	case "elementum_list_view":
		if view.Density != "" {
			b.SetAttr("density", Str(view.Density))
		}
		if view.Rows > 0 {
			b.SetAttr("rows", Num(float64(view.Rows)))
		}
	case "elementum_kanban_view":
		if view.DisplayByField != "" {
			if ref := g.resolveRef(view.DisplayByField); ref != "" {
				b.SetAttr("display_by_field", refOrStr(ref))
			} else {
				b.SetAttr("display_by_field", Str(view.DisplayByField))
			}
		}
		if len(view.AllowedFilterFields) > 0 {
			var filterVals []HCLValue
			for _, f := range view.AllowedFilterFields {
				if ref := g.resolveRef(f); ref != "" {
					filterVals = append(filterVals, refOrStr(ref))
				} else {
					filterVals = append(filterVals, Str(f))
				}
			}
			b.SetAttr("allowed_filter_fields", HCLList{Values: filterVals})
		}
	case "elementum_calendar_view":
		if view.DisplayByField != "" {
			if ref := g.resolveRef(view.DisplayByField); ref != "" {
				b.SetAttr("display_by_field", refOrStr(ref))
			} else {
				b.SetAttr("display_by_field", Str(view.DisplayByField))
			}
		}
	case "elementum_dashboard_view":
		if view.DashboardID != "" {
			if ref := g.resolveRef(view.DashboardID); ref != "" {
				b.SetAttr("dashboard_id", refOrStr(ref))
			} else {
				b.SetAttr("dashboard_id", Str(view.DashboardID))
			}
		}
	case "elementum_agent_view":
		if view.AgentID != "" {
			if ref := g.resolveRef(view.AgentID); ref != "" {
				b.SetAttr("agent_id", refOrStr(ref))
			} else {
				b.SetAttr("agent_id", Str(view.AgentID))
			}
		}
	}

	return b
}

// --- Tables ---

func (g *AppHCLGenerator) generateTableIR(table *discovery.Table) *HCLBlock {
	if table == nil {
		return nil
	}

	resourceName := g.findResourceName("elementum_table", table.ID)
	if resourceName == "" {
		resourceName = SanitizeName(table.Name)
	}

	b := NewResourceBlock("elementum_table", resourceName)

	b.SetAttr("name", Str(table.Name))
	if table.Handle != "" {
		b.SetAttr("handle", Str(table.Handle))
	}

	// category_id
	if table.CategoryID != "" {
		if ref := g.resolveRef(table.CategoryID); ref != "" {
			b.SetAttr("category_id", refOrStr(ref))
		} else {
			b.SetAttr("category_id", Str(table.CategoryID))
		}
	}

	// cloud_link_id
	if table.CloudLinkID != "" {
		if ref := g.resolveRef(table.CloudLinkID); ref != "" {
			b.SetAttr("cloud_link_id", refOrStr(ref))
		} else if table.CloudLinkName != "" {
			b.SetAttr("cloud_link_id", Ref(fmt.Sprintf("data.elementum_cloudlink.%s.id", SanitizeName(table.CloudLinkName))))
		} else {
			b.SetAttr("cloud_link_id", Str(table.CloudLinkID))
		}
	}

	// source_id
	if table.SourceID != "" && table.SourceID != table.ID {
		if ref := g.resolveRef(table.SourceID); ref != "" {
			b.SetAttr("source_id", refOrStr(ref))
		} else {
			b.SetAttr("source_id", Str(table.SourceID))
		}
	}

	if table.Description != "" {
		b.SetAttr("description", Str(table.Description))
	}

	// Snowflake cloud mapping
	if table.SnowflakeDatabaseName != "" {
		cloudMapping := Obj(
			Attr("database_name", Str(table.SnowflakeDatabaseName)),
			Attr("schema_name", Str(table.SnowflakeSchemaName)),
			Attr("table_name", Str(table.SnowflakeTableName)),
		)
		b.SetAttr("snowflake_cloud_mapping", cloudMapping)
	}

	// Fields
	if len(table.Fields) > 0 {
		var fieldVals []HCLValue
		for _, f := range table.Fields {
			attrs := []*HCLAttribute{
				Attr("name", Str(f.Name)),
				Attr("type", Str(f.Type)),
			}
			fieldVals = append(fieldVals, Obj(attrs...))
		}
		b.SetAttr("fields", HCLList{Values: fieldVals})
	}

	return b
}

// --- Datamines ---

func (g *AppHCLGenerator) generateDatamineIR(dm *discovery.Datamine) *HCLBlock {
	if dm == nil {
		return nil
	}

	resourceName := g.findResourceName("elementum_datamine", dm.ID)
	if resourceName == "" {
		resourceName = SanitizeName(dm.Name)
	}

	b := NewResourceBlock("elementum_datamine", resourceName)

	b.SetAttr("name", Str(dm.Name))

	// table_id
	if dm.TableID != "" {
		if ref := g.resolveRef(dm.TableID); ref != "" {
			b.SetAttr("table_id", refOrStr(ref))
		} else {
			b.SetAttr("table_id", Str(dm.TableID))
		}
	}

	if dm.Description != "" {
		b.SetAttr("description", Str(dm.Description))
	}

	// Primary column IDs
	if len(dm.PrimaryColumnIDs) > 0 {
		var colVals []HCLValue
		for _, colID := range dm.PrimaryColumnIDs {
			if ref := g.resolveRef(colID); ref != "" {
				colVals = append(colVals, refOrStr(ref))
			} else {
				colVals = append(colVals, Str(colID))
			}
		}
		b.SetAttr("primary_column_ids", HCLList{Values: colVals})
	}

	// Schedule
	if dm.ScheduleType == "fixed" && dm.TimeUnit != "" {
		schedule := Obj(
			Attr("time_unit", Str(dm.TimeUnit)),
			Attr("value", Num(float64(dm.Value))),
		)
		b.SetAttr("schedule_fixed", schedule)
	} else if dm.ScheduleType == "cron" && dm.CronExpression != "" {
		schedule := Obj(
			Attr("cron_expression", Str(dm.CronExpression)),
		)
		b.SetAttr("schedule_cron", schedule)
	}

	return b
}

// --- Helpers ---

func (g *AppHCLGenerator) findResourceName(resourceType, id string) string {
	return findResourceName(g.imports, resourceType, id)
}

func (g *AppHCLGenerator) resolveRef(id string) string {
	if ref, ok := g.uuidMap[id]; ok {
		return ref
	}
	return ""
}
