// Copyright (c) ElementumAI, Inc.
// SPDX-License-Identifier: MPL-2.0

package export

import (
	"fmt"
	"strings"

	"github.com/elementumltd/elementum-cli/discovery"
)

// AccessPolicyHCLGenerator generates HCL for access policy resources
type AccessPolicyHCLGenerator struct {
	app       *discovery.App
	imports   []ImportBlock
	uuidMap   map[string]string
	namespace string // namespace of the aspect (app/element/task)
}

// NewAccessPolicyHCLGenerator creates a new access policy HCL generator
func NewAccessPolicyHCLGenerator(app *discovery.App, imports []ImportBlock, uuidMap map[string]string) *AccessPolicyHCLGenerator {
	return &AccessPolicyHCLGenerator{
		app:       app,
		imports:   imports,
		uuidMap:   uuidMap,
		namespace: app.Namespace,
	}
}

// GenerateAll generates HCL for all access policies
func (g *AccessPolicyHCLGenerator) GenerateAll() string {
	return SerializeBlocks(g.GenerateAllIR())
}

// GenerateAllIR generates IR blocks for all access policies (data sources + resources)
func (g *AccessPolicyHCLGenerator) GenerateAllIR() []*HCLBlock {
	if g.app == nil || len(g.app.AccessPolicies) == 0 {
		return nil
	}

	var blocks []*HCLBlock

	// Generate data source blocks for users and groups
	blocks = append(blocks, g.generateDataSourcesIR()...)

	// Generate access policy resource blocks
	for i, policy := range g.app.AccessPolicies {
		b := g.GenerateAccessPolicyIR(&policy, i)
		if b != nil {
			blocks = append(blocks, b)
		}
	}

	return blocks
}

// generateDataSourcesIR generates IR blocks for user and group data sources
func (g *AccessPolicyHCLGenerator) generateDataSourcesIR() []*HCLBlock {
	var blocks []*HCLBlock

	users := make(map[string]string)
	groups := make(map[string]string)

	for _, policy := range g.app.AccessPolicies {
		for id, email := range policy.UserEmails {
			if email != "" {
				users[id] = email
			}
		}
		for id, name := range policy.GroupNames {
			if name != "" {
				groups[id] = name
			}
		}
	}

	for id, email := range users {
		resourceName := sanitizeEmailToName(email)
		b := NewDataBlock("elementum_user", resourceName)
		b.SetAttr("email", Str(email))
		blocks = append(blocks, b)

		g.uuidMap[id] = fmt.Sprintf("data.elementum_user.%s.id", resourceName)
	}

	for id, name := range groups {
		resourceName := SanitizeName(name)
		b := NewDataBlock("elementum_group", resourceName)
		b.SetAttr("name", Str(name))
		blocks = append(blocks, b)

		g.uuidMap[id] = fmt.Sprintf("data.elementum_group.%s.id", resourceName)
	}

	return blocks
}

// GenerateAccessPolicyIR generates an IR block for a single access policy resource
func (g *AccessPolicyHCLGenerator) GenerateAccessPolicyIR(policy *discovery.AccessPolicy, index int) *HCLBlock {
	var resourceName string
	for _, imp := range g.imports {
		if imp.ResourceType == "elementum_access_policy" && strings.Contains(imp.ID, policy.ID) {
			resourceName = imp.ResourceName
			break
		}
	}
	if resourceName == "" {
		resourceName = fmt.Sprintf("%s_policy_%d", SanitizeName(g.namespace), index+1)
	}

	appResourceName := AppResourceName(g.app)
	b := NewResourceBlock("elementum_access_policy", resourceName)
	b.SetAttr("object_id", Ref("elementum_app."+appResourceName+".id"))

	// TODO(ir): Convert filter to IR blocks (Phase 2 - filter conversion)
	// For now, filters are handled by the string-based GenerateAll() path.

	// user_ids
	if len(policy.UserIDs) > 0 {
		var userValues []HCLValue
		for _, userID := range policy.UserIDs {
			if email, ok := policy.UserEmails[userID]; ok && email != "" {
				resourceName := sanitizeEmailToName(email)
				userValues = append(userValues, Ref("data.elementum_user."+resourceName+".id"))
			} else {
				userValues = append(userValues, Str(userID))
			}
		}
		b.SetAttr("user_ids", HCLList{Values: userValues})
	}

	// group_ids
	if len(policy.GroupIDs) > 0 {
		var groupValues []HCLValue
		for _, groupID := range policy.GroupIDs {
			if name, ok := policy.GroupNames[groupID]; ok && name != "" {
				resourceName := SanitizeName(name)
				groupValues = append(groupValues, Ref("data.elementum_group."+resourceName+".id"))
			} else {
				groupValues = append(groupValues, Str(groupID))
			}
		}
		b.SetAttr("group_ids", HCLList{Values: groupValues})
	}

	return b
}

// GenerateAccessPolicyHCLStandalone generates HCL for all access policies as a standalone function
func GenerateAccessPolicyHCLStandalone(app *discovery.App, imports []ImportBlock) string {
	if app == nil || len(app.AccessPolicies) == 0 {
		return ""
	}

	uuidMap := buildUUIDMap(imports, app)
	gen := NewAccessPolicyHCLGenerator(app, imports, uuidMap)
	return SerializeBlocks(gen.GenerateAllIR())
}

// GenerateAccessPolicyImports generates import blocks for access policies
func GenerateAccessPolicyImports(app *discovery.App) []ImportBlock {
	if app == nil || len(app.AccessPolicies) == 0 {
		return nil
	}

	imports := make([]ImportBlock, 0, len(app.AccessPolicies))
	for i, policy := range app.AccessPolicies {
		resourceName := fmt.Sprintf("%s_policy_%d", SanitizeName(app.Namespace), i+1)
		imports = append(imports, ImportBlock{
			ResourceType: "elementum_access_policy",
			ResourceName: resourceName,
			ID:           fmt.Sprintf("%s:%s", app.ID, policy.ID),
		})
	}

	return imports
}
