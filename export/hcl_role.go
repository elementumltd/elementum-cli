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

// MaxPermissionLevelByGroup defines the maximum permission level supported by each group.
// Some groups like ATTACHMENTS only have CRUD permissions with no admin-only operations.
// SERVICES, SLAS, and USERS are read-only at the aspect (app/element) level.
// Level hierarchy: ADMIN > EDIT > VIEW > NONE.
var MaxPermissionLevelByGroup = map[string]string{
	"AGENTS":        "ADMIN",
	"AI_PROVIDERS":  "ADMIN",
	"ANALYTICS":     "ADMIN",
	"APPS":          "ADMIN",
	"ATTACHMENTS":   "EDIT", // Only has CREATE, READ, UPDATE, DELETE - no admin permissions
	"AUTOMATIONS":   "ADMIN",
	"CONVERSATIONS": "ADMIN",
	"RECORDS":       "ADMIN",
	"RELATIONSHIPS": "ADMIN",
	"SERVICES":      "VIEW", // Read-only at aspect level
	"SLAS":          "VIEW", // Read-only at aspect level
	"USERS":         "VIEW", // Read-only at aspect level
}

// capPermissionLevel caps the permission level to the maximum allowed for the group.
// This prevents exporting invalid permission levels that would fail Terraform validation.
func capPermissionLevel(group, level string) string {
	// CUSTOM level is always valid - user specifies exact permissions
	if level == "CUSTOM" {
		return level
	}

	maxLevel, exists := MaxPermissionLevelByGroup[group]
	if !exists {
		// Unknown group - pass through unchanged, let the API/provider handle it
		return level
	}

	levelRank := map[string]int{"NONE": 0, "VIEW": 1, "EDIT": 2, "ADMIN": 3}
	if levelRank[level] > levelRank[maxLevel] {
		return maxLevel
	}
	return level
}

// RoleHCLGenerator generates HCL for role resources
type RoleHCLGenerator struct {
	app            *discovery.App
	imports        []ImportBlock
	uuidMap        map[string]string
	appResourceRef string
}

// NewRoleHCLGenerator creates a new role HCL generator
func NewRoleHCLGenerator(app *discovery.App, imports []ImportBlock, uuidMap map[string]string, appResourceRef string) *RoleHCLGenerator {
	return &RoleHCLGenerator{
		app:            app,
		imports:        imports,
		uuidMap:        uuidMap,
		appResourceRef: appResourceRef,
	}
}

// GenerateAll generates HCL for all custom roles
func (g *RoleHCLGenerator) GenerateAll() string {
	return SerializeBlocks(g.GenerateAllIR())
}

// buildUserRefMap builds a map from user email to data source reference
func (g *RoleHCLGenerator) buildUserRefMap() map[string]string {
	refMap := make(map[string]string)
	userNames := make(map[string]int)

	// Collect all unique users across roles
	for _, role := range g.app.Roles {
		if role.Managed {
			continue
		}
		for _, user := range role.Users {
			if user.Name == "" { // email
				continue
			}
			if _, exists := refMap[user.Name]; !exists {
				baseName := sanitizeEmailToName(user.Name)
				count := userNames[baseName]
				userNames[baseName]++

				name := baseName
				if count > 0 {
					name = fmt.Sprintf("%s_%d", baseName, count)
				}
				refMap[user.Name] = "data.elementum_user." + name + ".id"
			}
		}
	}
	return refMap
}

// buildGroupRefMap builds a map from group name to data source reference
func (g *RoleHCLGenerator) buildGroupRefMap() map[string]string {
	refMap := make(map[string]string)
	groupNames := make(map[string]int)

	// Collect all unique groups across roles
	for _, role := range g.app.Roles {
		if role.Managed {
			continue
		}
		for _, group := range role.Groups {
			if group.Name == "" {
				continue
			}
			if _, exists := refMap[group.Name]; !exists {
				baseName := SanitizeName(group.Name)
				count := groupNames[baseName]
				groupNames[baseName]++

				name := baseName
				if count > 0 {
					name = fmt.Sprintf("%s_%d", baseName, count)
				}
				refMap[group.Name] = "data.elementum_group." + name + ".id"
			}
		}
	}
	return refMap
}

// GenerateAllIR generates IR blocks for all custom roles
func (g *RoleHCLGenerator) GenerateAllIR() []*HCLBlock {
	userRefMap := g.buildUserRefMap()
	groupRefMap := g.buildGroupRefMap()

	var blocks []*HCLBlock
	for _, role := range g.app.Roles {
		if role.Managed {
			continue
		}
		b := g.GenerateRoleIR(&role, userRefMap, groupRefMap)
		if b != nil {
			blocks = append(blocks, b)
		}
	}
	return blocks
}

// GenerateRoleIR generates an IR block for a single custom role resource
func (g *RoleHCLGenerator) GenerateRoleIR(role *discovery.Role, userRefMap, groupRefMap map[string]string) *HCLBlock {
	// Find the resource name from imports
	var resourceName string
	for _, imp := range g.imports {
		if imp.ResourceType == "elementum_role" && strings.Contains(imp.ID, role.ID) {
			resourceName = imp.ResourceName
			break
		}
	}
	if resourceName == "" {
		resourceName = SanitizeName(role.Name)
	}

	b := NewResourceBlock("elementum_role", resourceName)
	b.SetAttr("object_id", refOrStr(g.appResourceRef))
	b.SetAttr("name", Str(role.Name))

	if role.Description != "" {
		b.SetAttr("description", Str(role.Description))
	}

	// AutoShare
	if len(role.AutoShare) > 0 {
		var shareValues []HCLValue
		for _, as := range role.AutoShare {
			shareValues = append(shareValues, Str(as))
		}
		b.SetAttr("auto_share", List(shareValues...))
	}

	// User IDs
	if len(role.Users) > 0 {
		var userValues []HCLValue
		for _, user := range role.Users {
			if ref, ok := userRefMap[user.Name]; ok {
				userValues = append(userValues, Ref(ref))
			}
		}
		if len(userValues) > 0 {
			b.SetAttr("user_ids", HCLList{Values: userValues})
		}
	}

	// Group IDs
	if len(role.Groups) > 0 {
		var groupValues []HCLValue
		for _, group := range role.Groups {
			if ref, ok := groupRefMap[group.Name]; ok {
				groupValues = append(groupValues, Ref(ref))
			}
		}
		if len(groupValues) > 0 {
			b.SetAttr("group_ids", HCLList{Values: groupValues})
		}
	}

	// Permissions
	if len(role.Permissions) > 0 {
		var permObjs []HCLValue
		for _, perm := range role.Permissions {
			cappedLevel := capPermissionLevel(perm.Group, perm.Level)
			attrs := []*HCLAttribute{
				Attr("group", Str(perm.Group)),
				Attr("level", Str(cappedLevel)),
			}
			if cappedLevel == "CUSTOM" && len(perm.CustomPermissions) > 0 {
				var cpValues []HCLValue
				for _, cp := range perm.CustomPermissions {
					cpValues = append(cpValues, Str(cp))
				}
				attrs = append(attrs, Attr("custom_permissions", List(cpValues...)))
			}
			permObjs = append(permObjs, Obj(attrs...))
		}
		b.SetAttr("permissions", HCLList{Values: permObjs})
	}

	return b
}

// BeautifyRole applies beautification to a role resource block in HCL
func (g *RoleHCLGenerator) BeautifyRole(hcl string) string {
	// Replace UUID strings with terraform references. Skip empty keys
	// (would overwrite every empty string in the HCL with the ref).
	for uuid, ref := range g.uuidMap {
		if uuid == "" {
			continue
		}
		hcl = strings.ReplaceAll(hcl, fmt.Sprintf("%q", uuid), ref)
	}

	return hcl
}
