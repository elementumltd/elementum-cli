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

	"github.com/elementumltd/elementum-cli/discovery"
)

// ManagedViewOrderHCLGenerator emits a single `elementum_managed_view_order`
// block per aspect that has one. It pins the display order of views on an
// app or element; truth HCL groups this with the views themselves inside
// `views.tf`. One resource per owning aspect.
type ManagedViewOrderHCLGenerator struct {
	app     *discovery.App
	imports []ImportBlock
	uuidMap map[string]string
}

func NewManagedViewOrderHCLGenerator(app *discovery.App, imports []ImportBlock, uuidMap map[string]string) *ManagedViewOrderHCLGenerator {
	return &ManagedViewOrderHCLGenerator{app: app, imports: imports, uuidMap: uuidMap}
}

func (g *ManagedViewOrderHCLGenerator) GenerateAll() string {
	return SerializeBlocks(g.GenerateAllIR())
}

func (g *ManagedViewOrderHCLGenerator) GenerateAllIR() []*HCLBlock {
	if g.app == nil {
		return nil
	}
	var blocks []*HCLBlock
	if b := g.generateMVOIR(g.app.ManagedViewOrder, "elementum_app."+AppResourceName(g.app)+".id"); b != nil {
		blocks = append(blocks, b)
	}
	return blocks
}

func (g *ManagedViewOrderHCLGenerator) generateMVOIR(mvo *discovery.ManagedViewOrder, objectRef string) *HCLBlock {
	if mvo == nil || len(mvo.ViewIDs) == 0 {
		return nil
	}
	resourceName := ""
	for _, imp := range g.imports {
		if imp.ResourceType == "elementum_managed_view_order" && strings.Contains(imp.ID, mvo.ObjectID) {
			resourceName = imp.ResourceName
			break
		}
	}
	if resourceName == "" {
		resourceName = SanitizeName("view_order_" + mvo.ObjectID)
	}

	b := NewResourceBlock("elementum_managed_view_order", resourceName)
	b.SetAttr("object_id", Ref(objectRef))

	values := make([]HCLValue, 0, len(mvo.ViewIDs))
	for _, vid := range mvo.ViewIDs {
		if ref, ok := g.uuidMap[vid]; ok {
			values = append(values, Raw(ref))
		} else {
			values = append(values, Str(vid))
		}
	}
	b.SetAttr("view_ids", HCLList{Values: values})
	return b
}
