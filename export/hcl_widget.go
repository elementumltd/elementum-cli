// Copyright (c) ElementumAI, Inc.
// SPDX-License-Identifier: MPL-2.0

package export

import (
	"strings"

	"github.com/elementumltd/elementum-cli/discovery"
)

// WidgetHCLGenerator generates HCL for widget resources
// Supports: elementum_widget with related_object, related_link_action, and related_create_action types
type WidgetHCLGenerator struct {
	app     *discovery.App
	imports []ImportBlock
	uuidMap map[string]string
}

// NewWidgetHCLGenerator creates a new widget HCL generator
func NewWidgetHCLGenerator(app *discovery.App, imports []ImportBlock, uuidMap map[string]string) *WidgetHCLGenerator {
	return &WidgetHCLGenerator{
		app:     app,
		imports: imports,
		uuidMap: uuidMap,
	}
}

// GenerateAll generates HCL for all widgets in the app and discovered apps
func (g *WidgetHCLGenerator) GenerateAll() string {
	return SerializeBlocks(g.GenerateAllIR())
}

// GenerateAllIR generates IR blocks for all widgets
func (g *WidgetHCLGenerator) GenerateAllIR() []*HCLBlock {
	if g.app == nil {
		return nil
	}

	var blocks []*HCLBlock

	if len(g.app.Widgets) > 0 {
		appResourceName := AppResourceName(g.app)
		for _, widget := range g.app.Widgets {
			b := g.GenerateWidgetIR(&widget, appResourceName, "elementum_app")
			if b != nil {
				blocks = append(blocks, b)
			}
		}
	}

	for _, discoveredApp := range g.app.DiscoveredApps {
		if len(discoveredApp.Widgets) > 0 {
			name := AppResourceName(discoveredApp)
			for _, widget := range discoveredApp.Widgets {
				b := g.GenerateWidgetIR(&widget, name, "elementum_app")
				if b != nil {
					blocks = append(blocks, b)
				}
			}
		}
	}

	for _, discoveredElement := range g.app.DiscoveredElements {
		if len(discoveredElement.Widgets) > 0 {
			name := ElementResourceName(discoveredElement)
			for _, widget := range discoveredElement.Widgets {
				b := g.GenerateWidgetIR(&widget, name, "elementum_element")
				if b != nil {
					blocks = append(blocks, b)
				}
			}
		}
	}

	for _, discoveredTask := range g.app.DiscoveredTasks {
		if len(discoveredTask.Widgets) > 0 {
			name := TaskResourceName(discoveredTask)
			for _, widget := range discoveredTask.Widgets {
				b := g.GenerateWidgetIR(&widget, name, "elementum_task")
				if b != nil {
					blocks = append(blocks, b)
				}
			}
		}
	}

	return blocks
}

// GenerateWidgetIR generates an IR block for a single widget
func (g *WidgetHCLGenerator) GenerateWidgetIR(widget *discovery.Widget, parentResourceName, parentResourceType string) *HCLBlock {
	var resourceName string
	for _, imp := range g.imports {
		if imp.ResourceType == "elementum_widget" && strings.Contains(imp.ID, widget.ID) {
			resourceName = imp.ResourceName
			break
		}
	}
	if resourceName == "" {
		return nil
	}

	b := NewResourceBlock("elementum_widget", resourceName)
	b.SetAttr("object_id", Ref(parentResourceType+"."+parentResourceName+".id"))

	// Beautify the nested object_id (AspectID) if we have a mapping for it
	var objectIDVal HCLValue
	if ref, ok := g.uuidMap[widget.AspectID]; ok {
		objectIDVal = Ref(ref)
	} else {
		objectIDVal = Str(widget.AspectID)
	}

	switch widget.Type {
	case "DisplayWidgetRelatedAspect":
		attrs := []*HCLAttribute{
			Attr("name", Str(widget.Name)),
			Attr("object_id", objectIDVal),
		}
		if len(widget.Columns) > 0 {
			var colVals []HCLValue
			for _, col := range widget.Columns {
				colVals = append(colVals, Str(col))
			}
			attrs = append(attrs, Attr("columns", HCLList{Values: colVals}))
		}
		if widget.Rows > 0 && widget.Rows != 50 {
			attrs = append(attrs, Attr("rows", Num(float64(widget.Rows))))
		}
		b.SetAttr("related_object", Obj(attrs...))

	case "DisplayWidgetRelatedLinkAction":
		attrs := []*HCLAttribute{
			Attr("name", Str(widget.Name)),
			Attr("object_id", objectIDVal),
			Attr("button_type", Str(widget.ButtonType)),
		}
		if widget.Color != "" {
			attrs = append(attrs, Attr("color", Str(widget.Color)))
		}
		if widget.Icon != "" {
			attrs = append(attrs, Attr("icon", Str(widget.Icon)))
		}
		if widget.FullWidth {
			attrs = append(attrs, Attr("full_width", Bool(true)))
		}
		if len(widget.Columns) > 0 {
			var colVals []HCLValue
			for _, col := range widget.Columns {
				colVals = append(colVals, Str(col))
			}
			attrs = append(attrs, Attr("columns", HCLList{Values: colVals}))
		}
		if widget.Rows > 0 && widget.Rows != 10 {
			attrs = append(attrs, Attr("rows", Num(float64(widget.Rows))))
		}
		b.SetAttr("related_link_action", Obj(attrs...))

	case "DisplayWidgetRelatedCreateAction":
		attrs := []*HCLAttribute{
			Attr("name", Str(widget.Name)),
			Attr("object_id", objectIDVal),
			Attr("button_type", Str(widget.ButtonType)),
		}
		if widget.Color != "" {
			attrs = append(attrs, Attr("color", Str(widget.Color)))
		}
		if widget.Icon != "" {
			attrs = append(attrs, Attr("icon", Str(widget.Icon)))
		}
		if widget.FullWidth {
			attrs = append(attrs, Attr("full_width", Bool(true)))
		}
		b.SetAttr("related_create_action", Obj(attrs...))

	default:
		return nil // Unknown type - handled by string path with comment
	}

	return b
}

// ElementWidgetHCLGenerator generates HCL for widgets on elements
type ElementWidgetHCLGenerator struct {
	element *discovery.Element
	imports []ImportBlock
	uuidMap map[string]string
}

// NewElementWidgetHCLGenerator creates a new element widget HCL generator
func NewElementWidgetHCLGenerator(element *discovery.Element, imports []ImportBlock, uuidMap map[string]string) *ElementWidgetHCLGenerator {
	return &ElementWidgetHCLGenerator{
		element: element,
		imports: imports,
		uuidMap: uuidMap,
	}
}

// GenerateAll generates HCL for all widgets on the element
func (g *ElementWidgetHCLGenerator) GenerateAll() string {
	return SerializeBlocks(g.GenerateAllIR())
}

// GenerateAllIR generates IR blocks for all widgets on the element
func (g *ElementWidgetHCLGenerator) GenerateAllIR() []*HCLBlock {
	if g.element == nil || len(g.element.Widgets) == 0 {
		return nil
	}

	var blocks []*HCLBlock
	elementResourceName := ElementResourceName(g.element)
	appGen := &WidgetHCLGenerator{imports: g.imports, uuidMap: g.uuidMap}

	for _, widget := range g.element.Widgets {
		b := appGen.GenerateWidgetIR(&widget, elementResourceName, "elementum_element")
		if b != nil {
			blocks = append(blocks, b)
		}
	}
	return blocks
}
