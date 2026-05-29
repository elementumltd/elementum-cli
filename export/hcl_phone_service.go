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

// PhoneServiceHCLGenerator emits `elementum_phone_service` blocks. One
// service per voice channel: provider + agent + caller-access mode + an
// optional registration properties block.
type PhoneServiceHCLGenerator struct {
	app     *discovery.App
	imports []ImportBlock
	uuidMap map[string]string
}

func NewPhoneServiceHCLGenerator(app *discovery.App, imports []ImportBlock, uuidMap map[string]string) *PhoneServiceHCLGenerator {
	return &PhoneServiceHCLGenerator{app: app, imports: imports, uuidMap: uuidMap}
}

func (g *PhoneServiceHCLGenerator) GenerateAll() string {
	return SerializeBlocks(g.GenerateAllIR())
}

func (g *PhoneServiceHCLGenerator) GenerateAllIR() []*HCLBlock {
	if g.app == nil || len(g.app.PhoneServices) == 0 {
		return nil
	}
	var blocks []*HCLBlock
	for i := range g.app.PhoneServices {
		if b := g.generatePhoneServiceIR(&g.app.PhoneServices[i]); b != nil {
			blocks = append(blocks, b)
		}
	}
	return blocks
}

func (g *PhoneServiceHCLGenerator) generatePhoneServiceIR(ps *discovery.PhoneService) *HCLBlock {
	resourceName := ""
	for _, imp := range g.imports {
		if imp.ResourceType == "elementum_phone_service" && strings.Contains(imp.ID, ps.ID) {
			resourceName = imp.ResourceName
			break
		}
	}
	if resourceName == "" {
		resourceName = SanitizeName("phone_service_" + ps.ID)
	}

	b := NewResourceBlock("elementum_phone_service", resourceName)
	b.SetAttr("app_id", Ref("elementum_app."+AppResourceName(g.app)+".id"))
	if ref := g.uuidMap[ps.ProviderID]; ref != "" {
		b.SetAttr("provider_id", Ref(ref))
	} else if ps.ProviderName != "" {
		// Fall back to data-source lookup by name — same pattern other
		// resources use when the provider isn't in our import set.
		b.SetAttr("provider_id", Ref("data.elementum_phone_provider."+SanitizeName(ps.ProviderName)+".id"))
	} else if ps.ProviderID != "" {
		b.SetAttr("provider_id", Str(ps.ProviderID))
	}
	if ref := g.uuidMap[ps.AgentID]; ref != "" {
		b.SetAttr("agent_id", Ref(ref))
	} else if ps.AgentID != "" {
		b.SetAttr("agent_id", Str(ps.AgentID))
	}
	// twilio_existing_number is the safe roundtrip form when a phone number
	// already exists on the service — the twilio_new_number input is only
	// meaningful on initial provisioning.
	if ps.PhoneNumber != "" {
		b.SetAttr("twilio_existing_number", Obj(
			Attr("phone_number", Str(ps.PhoneNumber)),
		))
	}

	reg := ps.RegistrationProperties
	regAttrs := []*HCLAttribute{
		Attr("generate_first_message", Bool(reg.GenerateFirstMessage)),
		Attr("expand_languages", Bool(reg.ExpandLanguages)),
	}
	if reg.DefaultLanguage != "" {
		regAttrs = append(regAttrs, Attr("default_language", Str(reg.DefaultLanguage)))
	}
	regAttrs = append(regAttrs, Attr("enable_language_routing", Bool(reg.EnableLanguageRouting)))
	if len(reg.PreferredLanguages) > 0 {
		vals := make([]HCLValue, 0, len(reg.PreferredLanguages))
		for _, p := range reg.PreferredLanguages {
			vals = append(vals, Str(p))
		}
		regAttrs = append(regAttrs, Attr("preferred_languages", HCLList{Values: vals}))
	}
	b.SetAttr("registration", Obj(regAttrs...))

	return b
}
